package raydium

import (
	"fmt"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// ParseEventConfig holds configuration for parsing pool events
type ParseEventConfig struct {
	EventType          types.PoolEventType
	PoolIdIndex        int
	LpMintIndex        int
	TokenAmountOffsets *TokenAmountOffsets
}

// TokenAmountOffsets holds byte offsets for token amounts in instruction data
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

// ParseRaydiumInstruction parses a Raydium instruction into a pool event
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

	return p.parseEvent(instruction, outerIndex, data, filteredTransfers, config)
}

// parseEvent parses instruction into pool event
func (p *RaydiumLiquidityParserBase) parseEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
	config *ParseEventConfig,
) *types.PoolEvent {
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
	base.Idx = intToString(index)

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
	} else if config.TokenAmountOffsets != nil && len(data) > config.TokenAmountOffsets.Token0+8 {
		amt := readU64LE(data, config.TokenAmountOffsets.Token0)
		uiAmt := types.ConvertToUIAmount(amt, token0Decimals)
		event.Token0Amount = &uiAmt
		event.Token0AmountRaw = amt.String()
	}

	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	} else if config.TokenAmountOffsets != nil && len(data) > config.TokenAmountOffsets.Token1+8 {
		amt := readU64LE(data, config.TokenAmountOffsets.Token1)
		uiAmt := types.ConvertToUIAmount(amt, token1Decimals)
		event.Token1Amount = &uiAmt
		event.Token1AmountRaw = amt.String()
	}

	if lpToken != nil && lpToken.Info.TokenAmount.UIAmount != nil {
		event.LpAmount = lpToken.Info.TokenAmount.UIAmount
		event.LpAmountRaw = lpToken.Info.TokenAmount.Amount
	} else if config.TokenAmountOffsets != nil && len(data) > config.TokenAmountOffsets.Lp+8 {
		amt := readU64LE(data, config.TokenAmountOffsets.Lp)
		uiAmt := types.ConvertToUIAmount(amt, 0)
		event.LpAmount = &uiAmt
		event.LpAmountRaw = amt.String()
	} else {
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

func intToString(i int) string {
	return fmt.Sprintf("%d", i)
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
