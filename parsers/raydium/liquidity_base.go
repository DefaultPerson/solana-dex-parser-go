package raydium

import (
	"math/big"
	"strings"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// ParseEventConfig holds configuration for parsing pool events
type ParseEventConfig struct {
	EventType          types.PoolEventType
	PoolIdIndex        int
	LpMintIndex        int
	TokenAmountOffsets *TokenAmountOffsets
	// VaultIndexes are the accounts of the pool's token vaults [vault_0,
	// vault_1], nil when not used. With them each token transfer is matched
	// to its side by vault, a side without a transfer moved nothing (0: a
	// concentrated position out of range deposits or withdraws one token)
	// and the instruction data amounts (slippage limits) are not used.
	VaultIndexes []int
	// MintIndexes are the accounts of the pool's mints [mint_0, mint_1]
	// when the instruction names them, nil otherwise (the vaults' mints are
	// used then)
	MintIndexes []int
}

// TokenAmountOffsets holds byte offsets of u64 amounts in the instruction data,
// used when the transfers do not provide the amounts. A negative offset means
// the instruction data has no such amount.
type TokenAmountOffsets struct {
	Token0 int
	Token1 int
	Lp     int
}

// RaydiumLiquidityParserBase is base parser for Raydium liquidity operations
type RaydiumLiquidityParserBase struct {
	*parsers.BaseLiquidityParser
}

// NewRaydiumLiquidityParserBase creates a new base liquidity parser
func NewRaydiumLiquidityParserBase(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *RaydiumLiquidityParserBase {
	return &RaydiumLiquidityParserBase{
		BaseLiquidityParser: parsers.NewBaseLiquidityParser(adapter, transferActions, classifiedInstructions),
	}
}

// PoolActionGetter interface for getting pool action type
type PoolActionGetter interface {
	GetPoolAction(data []byte) interface{}
	GetEventConfig(eventType types.PoolEventType, instructionType interface{}) *ParseEventConfig
}

// ParseRaydiumInstruction parses a Raydium instruction into a pool event.
// innerIndex is the instruction's inner index, -1 for an outer instruction.
func (p *RaydiumLiquidityParserBase) ParseRaydiumInstruction(
	instruction interface{},
	programId string,
	outerIndex int,
	innerIndex int,
	actionGetter PoolActionGetter,
) *types.PoolEvent {
	data := p.Adapter.GetInstructionData(instruction)
	instructionType := actionGetter.GetPoolAction(data)
	if instructionType == nil {
		return nil
	}

	accounts := p.Adapter.GetInstructionAccounts(instruction)

	var eventType types.PoolEventType
	switch v := instructionType.(type) {
	case types.PoolEventType:
		eventType = v
	case InstructionTypeInfo:
		eventType = v.Type
	case struct {
		Name string
		Type types.PoolEventType
	}:
		eventType = v.Type
	default:
		return nil
	}

	transfers := p.GetTransfersForInstruction(programId, outerIndex, innerIndex, nil)
	// Filter transfers
	var filteredTransfers []types.TransferData
	for _, t := range transfers {
		if t.Info.Destination == "" || (t.Info.Authority != "" && contains(accounts, t.Info.Destination)) {
			filteredTransfers = append(filteredTransfers, t)
		}
	}

	config := actionGetter.GetEventConfig(eventType, instructionType)
	if config == nil {
		return nil
	}

	return p.parseEvent(instruction, outerIndex, innerIndex, data, filteredTransfers, config)
}

// parseEvent parses instruction into pool event
func (p *RaydiumLiquidityParserBase) parseEvent(
	instruction interface{},
	outerIndex int,
	innerIndex int,
	data []byte,
	transfers []types.TransferData,
	config *ParseEventConfig,
) *types.PoolEvent {
	if config.VaultIndexes != nil {
		return p.parseVaultEvent(instruction, outerIndex, innerIndex, data, transfers, config)
	}
	if config.EventType == types.PoolEventTypeAdd && len(transfers) < 2 {
		return nil
	}

	// GetLPTransfers returns transfers that contain "transfer" in their type
	lpTransfers := p.Utils.GetLPTransfers(transfers)
	var token0, token1, lpToken *types.TransferData
	if len(lpTransfers) > 0 {
		token0 = &lpTransfers[0]
	}
	if len(lpTransfers) > 1 {
		token1 = &lpTransfers[1]
	}

	// Find LP token (mintTo for add, burn for remove)
	for i := range transfers {
		expectedType := "mintTo"
		if config.EventType == types.PoolEventTypeRemove {
			expectedType = "burn"
		}
		if transfers[i].Type == expectedType {
			lpToken = &transfers[i]
			break
		}
	}

	programId := p.Adapter.GetInstructionProgramId(instruction)
	accounts := p.Adapter.GetInstructionAccounts(instruction)

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
	}

	token0Decimals := p.Adapter.GetTokenDecimals(token0Mint)
	token1Decimals := p.Adapter.GetTokenDecimals(token1Mint)

	// Create PoolEvent with embedded base
	base := p.Adapter.GetPoolEventBase(config.EventType, programId)
	base.Idx = utils.FormatIdx(outerIndex, innerIndex)

	event := &types.PoolEvent{
		PoolEventBase:  base,
		Token0Mint:     token0Mint,
		Token1Mint:     token1Mint,
		Token0Decimals: &token0Decimals,
		Token1Decimals: &token1Decimals,
	}

	if config.PoolIdIndex < len(accounts) {
		event.PoolId = accounts[config.PoolIdIndex]
	}

	if lpToken != nil {
		event.PoolLpMint = lpToken.Info.Mint
	} else if config.LpMintIndex < len(accounts) {
		event.PoolLpMint = accounts[config.LpMintIndex]
	}

	// Set token amounts from transfers or instruction data
	if token0 != nil && token0.Info.TokenAmount.UIAmount != nil {
		event.Token0Amount = token0.Info.TokenAmount.UIAmount
		event.Token0AmountRaw = token0.Info.TokenAmount.Amount
	} else if config.TokenAmountOffsets != nil && hasU64At(data, config.TokenAmountOffsets.Token0) {
		amt := readU64LE(data, config.TokenAmountOffsets.Token0)
		uiAmt := types.ConvertToUIAmount(amt, token0Decimals)
		event.Token0Amount = &uiAmt
		event.Token0AmountRaw = amt.String()
	}

	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	} else if config.TokenAmountOffsets != nil && hasU64At(data, config.TokenAmountOffsets.Token1) {
		amt := readU64LE(data, config.TokenAmountOffsets.Token1)
		uiAmt := types.ConvertToUIAmount(amt, token1Decimals)
		event.Token1Amount = &uiAmt
		event.Token1AmountRaw = amt.String()
	}

	if lpToken != nil && lpToken.Info.TokenAmount.UIAmount != nil {
		event.LpAmount = lpToken.Info.TokenAmount.UIAmount
		event.LpAmountRaw = lpToken.Info.TokenAmount.Amount
	} else if config.TokenAmountOffsets != nil && hasU64At(data, config.TokenAmountOffsets.Lp) {
		amt := readU64LE(data, config.TokenAmountOffsets.Lp)
		uiAmt := types.ConvertToUIAmount(amt, 0)
		event.LpAmount = &uiAmt
		event.LpAmountRaw = amt.String()
	} else {
		event.LpAmountRaw = "0"
	}

	return event
}

// parseVaultEvent builds an ADD or REMOVE event from the transfers into or
// out of the pool's vaults (config.VaultIndexes). A side without a transfer
// is 0 with the pool's mint of that side. Without any token transfer (no
// inner instructions) an ADD is not an event and a REMOVE takes the
// instruction data amounts, its minimum amounts. As with GetLPTransfers,
// the quote mint (SOL, or a supported token paired with an unsupported one)
// is token1.
func (p *RaydiumLiquidityParserBase) parseVaultEvent(
	instruction interface{},
	outerIndex int,
	innerIndex int,
	data []byte,
	transfers []types.TransferData,
	config *ParseEventConfig,
) *types.PoolEvent {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	account := func(indexes []int, side int) string {
		if side < len(indexes) && indexes[side] >= 0 && indexes[side] < len(accounts) {
			return accounts[indexes[side]]
		}
		return ""
	}

	var sides [2]*types.TransferData
	var lpToken *types.TransferData
	lpType := "mintTo"
	if config.EventType == types.PoolEventTypeRemove {
		lpType = "burn"
	}
	for i := range transfers {
		t := &transfers[i]
		if t.Type == lpType && lpToken == nil {
			lpToken = t
			continue
		}
		if !strings.Contains(t.Type, "transfer") {
			continue
		}
		for side := 0; side < 2; side++ {
			vault := account(config.VaultIndexes, side)
			if sides[side] == nil && vault != "" && (t.Info.Destination == vault || t.Info.Source == vault) {
				sides[side] = t
				break
			}
		}
	}
	noTransfers := sides[0] == nil && sides[1] == nil
	if noTransfers && config.EventType == types.PoolEventTypeAdd {
		return nil
	}

	var mints [2]string
	var amounts [2]*types.TokenAmount
	for side := 0; side < 2; side++ {
		if t := sides[side]; t != nil {
			mints[side] = t.Info.Mint
			amounts[side] = &t.Info.TokenAmount
			continue
		}
		mints[side] = account(config.MintIndexes, side)
		if mints[side] == "" {
			mints[side] = p.Adapter.KnownTokenAccountMint(account(config.VaultIndexes, side))
		}
		if offsets := config.TokenAmountOffsets; noTransfers && offsets != nil {
			offset := offsets.Token0
			if side == 1 {
				offset = offsets.Token1
			}
			if hasU64At(data, offset) {
				amt := readU64LE(data, offset)
				ui := types.ConvertToUIAmount(amt, p.Adapter.GetTokenDecimals(mints[side]))
				amounts[side] = &types.TokenAmount{Amount: amt.String(), UIAmount: &ui}
			}
		}
	}
	if mints[0] == constants.TOKENS.SOL ||
		(p.Adapter.IsSupportedToken(mints[0]) && !p.Adapter.IsSupportedToken(mints[1])) {
		mints[0], mints[1] = mints[1], mints[0]
		amounts[0], amounts[1] = amounts[1], amounts[0]
	}

	programId := p.Adapter.GetInstructionProgramId(instruction)
	base := p.Adapter.GetPoolEventBase(config.EventType, programId)
	base.Idx = utils.FormatIdx(outerIndex, innerIndex)
	token0Decimals := p.Adapter.GetTokenDecimals(mints[0])
	token1Decimals := p.Adapter.GetTokenDecimals(mints[1])
	event := &types.PoolEvent{
		PoolEventBase:  base,
		PoolId:         account([]int{config.PoolIdIndex}, 0),
		Token0Mint:     mints[0],
		Token1Mint:     mints[1],
		Token0Decimals: &token0Decimals,
		Token1Decimals: &token1Decimals,
	}
	setAmount := func(amount *types.TokenAmount, ui **float64, raw *string) {
		if amount != nil && amount.UIAmount != nil {
			*ui, *raw = amount.UIAmount, amount.Amount
			return
		}
		zero := float64(0)
		*ui, *raw = &zero, "0"
	}
	setAmount(amounts[0], &event.Token0Amount, &event.Token0AmountRaw)
	setAmount(amounts[1], &event.Token1Amount, &event.Token1AmountRaw)

	if lpToken != nil {
		event.PoolLpMint = lpToken.Info.Mint
	} else if config.LpMintIndex < len(accounts) {
		event.PoolLpMint = accounts[config.LpMintIndex]
	}
	switch {
	case lpToken != nil && lpToken.Info.TokenAmount.UIAmount != nil:
		event.LpAmount = lpToken.Info.TokenAmount.UIAmount
		event.LpAmountRaw = lpToken.Info.TokenAmount.Amount
	case config.TokenAmountOffsets != nil && hasU64At(data, config.TokenAmountOffsets.Lp):
		amt := readU64LE(data, config.TokenAmountOffsets.Lp)
		uiAmt := types.ConvertToUIAmount(amt, 0)
		event.LpAmount = &uiAmt
		event.LpAmountRaw = amt.String()
	default:
		event.LpAmountRaw = "0"
	}
	return event
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// hasU64At reports whether data holds a u64 at offset (a negative offset
// means the instruction has no such field)
func hasU64At(data []byte, offset int) bool {
	return offset >= 0 && offset+8 <= len(data)
}

func readU64LE(data []byte, offset int) *big.Int {
	if offset+8 > len(data) {
		return big.NewInt(0)
	}
	val := uint64(data[offset]) |
		uint64(data[offset+1])<<8 |
		uint64(data[offset+2])<<16 |
		uint64(data[offset+3])<<24 |
		uint64(data[offset+4])<<32 |
		uint64(data[offset+5])<<40 |
		uint64(data[offset+6])<<48 |
		uint64(data[offset+7])<<56
	return new(big.Int).SetUint64(val)
}
