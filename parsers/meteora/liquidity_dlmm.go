package meteora

import (
	"bytes"
	"strconv"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// MeteoraDLMMPoolParser parses Meteora DLMM pool events
type MeteoraDLMMPoolParser struct {
	*MeteoraLiquidityParserBase
}

// NewMeteoraDLMMPoolParser creates a new DLMM pool parser
func NewMeteoraDLMMPoolParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MeteoraDLMMPoolParser {
	return &MeteoraDLMMPoolParser{
		MeteoraLiquidityParserBase: NewMeteoraLiquidityParserBase(adapter, transferActions, classifiedInstructions),
	}
}

// GetPoolAction determines the pool action type from instruction data
func (p *MeteoraDLMMPoolParser) GetPoolAction(data []byte) interface{} {
	if len(data) < 8 {
		return nil
	}
	disc := data[:8]

	// Check ADD_LIQUIDITY discriminators
	for name, d := range constants.DISCRIMINATORS.METEORA_DLMM.ADD_LIQUIDITY {
		if bytes.Equal(disc, d) {
			return &PoolActionResult{Name: name, Type: types.PoolEventTypeAdd}
		}
	}

	// Check REMOVE_LIQUIDITY discriminators
	for name, d := range constants.DISCRIMINATORS.METEORA_DLMM.REMOVE_LIQUIDITY {
		if bytes.Equal(disc, d) {
			return &PoolActionResult{Name: name, Type: types.PoolEventTypeRemove}
		}
	}

	return nil
}

// ProcessLiquidity parses liquidity events
func (p *MeteoraDLMMPoolParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.METEORA.ID {
			event := p.ParseInstruction(ci.Instruction, ci.ProgramId, ci.OuterIndex, ci.InnerIndex, p)
			if event != nil {
				events = append(events, *event)
			}
		}
	}

	return events
}

// ParseAddLiquidityEvent parses add liquidity event
func (p *MeteoraDLMMPoolParser) ParseAddLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	token0, token1 := p.normalizeTokens(transfers)
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

	base := p.Adapter.GetPoolEventBase(types.PoolEventTypeAdd, programId)

	event := &types.PoolEvent{
		PoolEventBase:  base,
		Token0Mint:     token0Mint,
		Token1Mint:     token1Mint,
		Token0Decimals: &token0Decimals,
		Token1Decimals: &token1Decimals,
	}
	event.Idx = strconv.Itoa(index)

	if len(accounts) > 1 {
		event.PoolId = accounts[1]
		event.PoolLpMint = accounts[1]
	}

	if token0 != nil && token0.Info.TokenAmount.UIAmount != nil {
		event.Token0Amount = token0.Info.TokenAmount.UIAmount
		event.Token0AmountRaw = token0.Info.TokenAmount.Amount
	}
	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	}

	return event
}

// dlmmRemoveLayout gives the lb_pair and token_x/y_mint account indices of a
// DLMM instruction that pays tokens out of the pool (lb_clmm IDL 0.12.0)
func dlmmRemoveLayout(data []byte) (poolIndex, mintXIndex, mintYIndex int) {
	dlmm := constants.DISCRIMINATORS.METEORA_DLMM.REMOVE_LIQUIDITY
	switch {
	case constants.MatchDiscriminator(data, dlmm["claimFee"]):
		// claim_fee: lb_pair 0, position 1, ..., token_x_mint 9, token_y_mint 10
		return 0, 9, 10
	case constants.MatchDiscriminator(data, dlmm["claimFeeV2"]):
		// claim_fee2: lb_pair 0, position 1, sender 2, reserves 3-4, user tokens 5-6, mints 7-8
		return 0, 7, 8
	}
	// remove_liquidity*, remove_all_liquidity: position 0, lb_pair 1, ..., mints 7-8
	return 1, 7, 8
}

// ParseRemoveLiquidityEvent parses remove liquidity event. Fee claims
// (claim_fee, claim_fee2) pay the position's fees out of the pool reserves
// and are reported as REMOVE events, as upstream does (DAMM v2
// claim_position_fee likewise); their LpAmount stays empty. Reward claims
// are not liquidity events.
func (p *MeteoraDLMMPoolParser) ParseRemoveLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	poolIndex, mintXIndex, mintYIndex := dlmmRemoveLayout(data)
	accountAt := func(i int) string {
		if i < len(accounts) {
			return accounts[i]
		}
		return ""
	}
	token0, token1 := p.normalizeTokens(transfers)

	// Normalize tokens based on account positions
	if token1 == nil && token0 != nil && token0.Info.Mint == accountAt(mintYIndex) {
		token1 = token0
		token0 = nil
	} else if token0 == nil && token1 != nil && token1.Info.Mint == accountAt(mintXIndex) {
		token0 = token1
		token1 = nil
	}

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	} else {
		token0Mint = accountAt(mintXIndex)
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
	} else {
		token1Mint = accountAt(mintYIndex)
	}

	programId := p.Adapter.GetInstructionProgramId(instruction)
	token0Decimals := p.Adapter.GetTokenDecimals(token0Mint)
	token1Decimals := p.Adapter.GetTokenDecimals(token1Mint)

	base := p.Adapter.GetPoolEventBase(types.PoolEventTypeRemove, programId)

	event := &types.PoolEvent{
		PoolEventBase:  base,
		Token0Mint:     token0Mint,
		Token1Mint:     token1Mint,
		Token0Decimals: &token0Decimals,
		Token1Decimals: &token1Decimals,
	}
	event.Idx = strconv.Itoa(index)
	event.PoolId = accountAt(poolIndex)
	event.PoolLpMint = event.PoolId

	if token0 != nil && token0.Info.TokenAmount.UIAmount != nil {
		event.Token0Amount = token0.Info.TokenAmount.UIAmount
		event.Token0AmountRaw = token0.Info.TokenAmount.Amount
	}
	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	}

	return event
}

// ParseCreateLiquidityEvent - DLMM doesn't have create events in this parser
func (p *MeteoraDLMMPoolParser) ParseCreateLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	return nil
}

// normalizeTokens normalizes token transfers for DLMM
func (p *MeteoraDLMMPoolParser) normalizeTokens(transfers []types.TransferData) (*types.TransferData, *types.TransferData) {
	lpTransfers := p.Utils.GetLPTransfers(transfers)
	var token0, token1 *types.TransferData
	if len(lpTransfers) > 0 {
		token0 = &lpTransfers[0]
	}
	if len(lpTransfers) > 1 {
		token1 = &lpTransfers[1]
	}

	// Special case: if only one transfer and it's SOL, put it as token1
	if len(transfers) == 1 && transfers[0].Info.Mint == constants.TOKENS.SOL {
		token1 = &transfers[0]
		token0 = nil
	}

	return token0, token1
}
