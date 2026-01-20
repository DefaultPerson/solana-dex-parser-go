package meteora

import (
	"bytes"

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
			innerIdx := ci.InnerIndex
			if innerIdx < 0 {
				innerIdx = 0
			}
			event := p.ParseInstruction(ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx, p)
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
	event.Idx = intToString(index)

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

// ParseRemoveLiquidityEvent parses remove liquidity event
func (p *MeteoraDLMMPoolParser) ParseRemoveLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	token0, token1 := p.normalizeTokens(transfers)

	// Normalize tokens based on account positions
	if token1 == nil && token0 != nil && len(accounts) > 8 && token0.Info.Mint == accounts[8] {
		token1 = token0
		token0 = nil
	} else if token0 == nil && token1 != nil && len(accounts) > 7 && token1.Info.Mint == accounts[7] {
		token0 = token1
		token1 = nil
	}

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	} else if len(accounts) > 7 {
		token0Mint = accounts[7]
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
	} else if len(accounts) > 8 {
		token1Mint = accounts[8]
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
	event.Idx = intToString(index)

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

func intToString(i int) string {
	return string(rune('0' + i%10))
}
