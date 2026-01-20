package orca

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// OrcaLiquidityParser parses Orca liquidity operations
type OrcaLiquidityParser struct {
	*parsers.BaseLiquidityParser
}

// NewOrcaLiquidityParser creates a new Orca liquidity parser
func NewOrcaLiquidityParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *OrcaLiquidityParser {
	return &OrcaLiquidityParser{
		BaseLiquidityParser: parsers.NewBaseLiquidityParser(adapter, transferActions, classifiedInstructions),
	}
}

// ProcessLiquidity parses liquidity events
func (p *OrcaLiquidityParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.ORCA.ID {
			innerIdx := ci.InnerIndex
			if innerIdx < 0 {
				innerIdx = 0
			}
			event := p.parseInstruction(ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx)
			if event != nil {
				events = append(events, *event)
			}
		}
	}

	return events
}

// parseInstruction parses a single instruction
func (p *OrcaLiquidityParser) parseInstruction(instruction interface{}, programId string, outerIndex int, innerIndex int) *types.PoolEvent {
	data := p.Adapter.GetInstructionData(instruction)
	action := p.getPoolAction(data)
	if action == "" {
		return nil
	}

	transfers := p.GetTransfersForInstruction(programId, outerIndex, innerIndex, nil)

	switch action {
	case types.PoolEventTypeAdd:
		return p.parseAddLiquidityEvent(instruction, outerIndex, data, transfers)
	case types.PoolEventTypeRemove:
		return p.parseRemoveLiquidityEvent(instruction, outerIndex, data, transfers)
	}

	return nil
}

// getPoolAction determines pool action from instruction data
func (p *OrcaLiquidityParser) getPoolAction(data []byte) types.PoolEventType {
	if len(data) < 8 {
		return ""
	}

	disc := data[:8]

	if bytes.Equal(disc, constants.DISCRIMINATORS.ORCA.ADD_LIQUIDITY) ||
		bytes.Equal(disc, constants.DISCRIMINATORS.ORCA.ADD_LIQUIDITY2) {
		return types.PoolEventTypeAdd
	}
	if bytes.Equal(disc, constants.DISCRIMINATORS.ORCA.REMOVE_LIQUIDITY) {
		return types.PoolEventTypeRemove
	}

	return ""
}

// parseAddLiquidityEvent parses add liquidity event
func (p *OrcaLiquidityParser) parseAddLiquidityEvent(instruction interface{}, index int, data []byte, transfers []types.TransferData) *types.PoolEvent {
	lpTransfers := p.Utils.GetLPTransfers(transfers)
	var token0, token1 *types.TransferData
	if len(lpTransfers) > 0 {
		token0 = &lpTransfers[0]
	}
	if len(lpTransfers) > 1 {
		token1 = &lpTransfers[1]
	}

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
	}

	programId := p.Adapter.GetInstructionProgramId(instruction)
	accounts := p.Adapter.GetInstructionAccounts(instruction)
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
		event.PoolLpMint = accounts[0]
	}

	if token0 != nil && token0.Info.TokenAmount.UIAmount != nil {
		event.Token0Amount = token0.Info.TokenAmount.UIAmount
		event.Token0AmountRaw = token0.Info.TokenAmount.Amount
	}
	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	}

	// LP amount from instruction data
	if len(data) >= 16 && len(accounts) > 1 {
		lpAmount := binary.LittleEndian.Uint64(data[8:16])
		lpDecimals := p.Adapter.GetTokenDecimals(accounts[1])
		uiAmt := types.ConvertToUIAmountUint64(lpAmount, lpDecimals)
		event.LpAmount = &uiAmt
		event.LpAmountRaw = fmt.Sprintf("%d", lpAmount)
	}

	return event
}

// parseRemoveLiquidityEvent parses remove liquidity event
func (p *OrcaLiquidityParser) parseRemoveLiquidityEvent(instruction interface{}, index int, data []byte, transfers []types.TransferData) *types.PoolEvent {
	lpTransfers := p.Utils.GetLPTransfers(transfers)
	var token0, token1 *types.TransferData
	if len(lpTransfers) > 0 {
		token0 = &lpTransfers[0]
	}
	if len(lpTransfers) > 1 {
		token1 = &lpTransfers[1]
	}

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
	}

	programId := p.Adapter.GetInstructionProgramId(instruction)
	accounts := p.Adapter.GetInstructionAccounts(instruction)
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

	if len(accounts) > 0 {
		event.PoolId = accounts[0]
		event.PoolLpMint = accounts[0]
	}

	if token0 != nil && token0.Info.TokenAmount.UIAmount != nil {
		event.Token0Amount = token0.Info.TokenAmount.UIAmount
		event.Token0AmountRaw = token0.Info.TokenAmount.Amount
	}
	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	}

	// LP amount from instruction data
	if len(data) >= 16 && len(accounts) > 1 {
		lpAmount := binary.LittleEndian.Uint64(data[8:16])
		lpDecimals := p.Adapter.GetTokenDecimals(accounts[1])
		uiAmt := types.ConvertToUIAmountUint64(lpAmount, lpDecimals)
		event.LpAmount = &uiAmt
		event.LpAmountRaw = fmt.Sprintf("%d", lpAmount)
	}

	return event
}
