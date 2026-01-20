package meteora

import (
	"bytes"
	"fmt"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/types"
)

// MeteoraDAMMPoolParser parses Meteora DAMM V2 pool events
type MeteoraDAMMPoolParser struct {
	*MeteoraLiquidityParserBase
}

// NewMeteoraDAMMPoolParser creates a new DAMM V2 pool parser
func NewMeteoraDAMMPoolParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MeteoraDAMMPoolParser {
	return &MeteoraDAMMPoolParser{
		MeteoraLiquidityParserBase: NewMeteoraLiquidityParserBase(adapter, transferActions, classifiedInstructions),
	}
}

// GetPoolAction determines the pool action type from instruction data
func (p *MeteoraDAMMPoolParser) GetPoolAction(data []byte) interface{} {
	if len(data) < 8 {
		return nil
	}
	disc := data[:8]

	if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.INITIALIZE_POOL) ||
		bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.INITIALIZE_CUSTOM_POOL) ||
		bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.INITIALIZE_POOL_WITH_DYNAMIC_CONFIG) {
		return types.PoolEventTypeCreate
	}
	if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.ADD_LIQUIDITY) {
		return types.PoolEventTypeAdd
	}
	if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.CLAIM_POSITION_FEE) ||
		bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.REMOVE_LIQUIDITY) ||
		bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.REMOVE_ALL_LIQUIDITY) {
		return types.PoolEventTypeRemove
	}

	return nil
}

// ProcessLiquidity parses liquidity events
func (p *MeteoraDAMMPoolParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID {
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

// ParseCreateLiquidityEvent parses create pool event
func (p *MeteoraDAMMPoolParser) ParseCreateLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	disc := data[:8]

	// Get event instruction for transfers
	eventInstruction := p.getInstructionByDiscriminator(constants.DISCRIMINATORS.METEORA_DAMM_V2.CREATE_POSITION_EVENT, 16)
	if eventInstruction != nil {
		eventTransfers := p.GetTransfersForInstruction(
			eventInstruction.ProgramId,
			eventInstruction.OuterIndex,
			eventInstruction.InnerIndex,
			nil,
		)
		if len(eventTransfers) > 0 {
			transfers = eventTransfers
		}
	}

	lpTransfers := p.Utils.GetLPTransfers(transfers)
	var token0, token1, lpToken *types.TransferData

	if len(lpTransfers) > 0 {
		token0 = &lpTransfers[0]
	}
	if len(lpTransfers) > 1 {
		token1 = &lpTransfers[1]
	}

	for i := range transfers {
		if transfers[i].Type == "mintTo" {
			lpToken = &transfers[i]
			break
		}
	}

	accounts := p.Adapter.GetInstructionAccounts(instruction)

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	} else {
		if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.INITIALIZE_CUSTOM_POOL) && len(accounts) > 7 {
			token0Mint = accounts[7]
		} else if len(accounts) > 8 {
			token0Mint = accounts[8]
		}
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
	} else {
		if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.INITIALIZE_CUSTOM_POOL) && len(accounts) > 8 {
			token1Mint = accounts[8]
		} else if len(accounts) > 9 {
			token1Mint = accounts[9]
		}
	}

	programId := p.Adapter.GetInstructionProgramId(instruction)
	token0Decimals := p.Adapter.GetTokenDecimals(token0Mint)
	token1Decimals := p.Adapter.GetTokenDecimals(token1Mint)

	base := p.Adapter.GetPoolEventBase(types.PoolEventTypeCreate, programId)

	event := &types.PoolEvent{
		PoolEventBase:  base,
		Token0Mint:     token0Mint,
		Token1Mint:     token1Mint,
		Token0Decimals: &token0Decimals,
		Token1Decimals: &token1Decimals,
	}
	event.Idx = fmt.Sprintf("%d", index)

	// Determine pool ID based on discriminator
	if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.INITIALIZE_CUSTOM_POOL) && len(accounts) > 5 {
		event.PoolId = accounts[5]
	} else if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM_V2.INITIALIZE_POOL_WITH_DYNAMIC_CONFIG) && len(accounts) > 7 {
		event.PoolId = accounts[7]
	} else if len(accounts) > 6 {
		event.PoolId = accounts[6]
	}

	if lpToken != nil {
		event.PoolLpMint = lpToken.Info.Mint
	} else if len(accounts) > 1 {
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

	if lpToken != nil && lpToken.Info.TokenAmount.UIAmount != nil {
		event.LpAmount = lpToken.Info.TokenAmount.UIAmount
		event.LpAmountRaw = lpToken.Info.TokenAmount.Amount
	} else {
		one := float64(1)
		event.LpAmount = &one
		event.LpAmountRaw = "1"
	}

	return event
}

// ParseAddLiquidityEvent parses add liquidity event
func (p *MeteoraDAMMPoolParser) ParseAddLiquidityEvent(
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
	event.Idx = fmt.Sprintf("%d", index)

	if len(accounts) > 0 {
		event.PoolId = accounts[0]
	}
	if len(accounts) > 1 {
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
func (p *MeteoraDAMMPoolParser) ParseRemoveLiquidityEvent(
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
	event.Idx = fmt.Sprintf("%d", index)

	if len(accounts) > 1 {
		event.PoolId = accounts[1]
	}
	if len(accounts) > 2 {
		event.PoolLpMint = accounts[2]
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

// normalizeTokens normalizes token transfers
func (p *MeteoraDAMMPoolParser) normalizeTokens(transfers []types.TransferData) (*types.TransferData, *types.TransferData) {
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

// getInstructionByDiscriminator finds instruction by discriminator
func (p *MeteoraDAMMPoolParser) getInstructionByDiscriminator(discriminator []byte, length int) *types.ClassifiedInstruction {
	for _, ci := range p.ClassifiedInstructions {
		data := p.Adapter.GetInstructionData(ci.Instruction)
		if len(data) >= length && bytes.Equal(data[:length], discriminator) {
			return &ci
		}
	}
	return nil
}
