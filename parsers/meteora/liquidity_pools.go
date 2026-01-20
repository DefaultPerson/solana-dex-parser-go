package meteora

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// MeteoraPoolsParser parses Meteora DAMM pool events
type MeteoraPoolsParser struct {
	*MeteoraLiquidityParserBase
}

// NewMeteoraPoolsParser creates a new Meteora pools parser
func NewMeteoraPoolsParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MeteoraPoolsParser {
	return &MeteoraPoolsParser{
		MeteoraLiquidityParserBase: NewMeteoraLiquidityParserBase(adapter, transferActions, classifiedInstructions),
	}
}

// GetPoolAction determines the pool action type from instruction data
func (p *MeteoraPoolsParser) GetPoolAction(data []byte) interface{} {
	if len(data) < 8 {
		return nil
	}
	disc := data[:8]

	if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM.CREATE) {
		return types.PoolEventTypeCreate
	}
	if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM.ADD_LIQUIDITY) ||
		bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM.ADD_IMBALANCE_LIQUIDITY) {
		return types.PoolEventTypeAdd
	}
	if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DAMM.REMOVE_LIQUIDITY) {
		return types.PoolEventTypeRemove
	}

	return nil
}

// ProcessLiquidity parses liquidity events
func (p *MeteoraPoolsParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.METEORA_DAMM.ID {
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
func (p *MeteoraPoolsParser) ParseCreateLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
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

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	} else if len(accounts) > 3 {
		token0Mint = accounts[3]
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
	} else if len(accounts) > 4 {
		token1Mint = accounts[4]
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

	if len(accounts) > 0 {
		event.PoolId = accounts[0]
	}
	if len(accounts) > 2 {
		event.PoolLpMint = accounts[2]
	}

	// Token amounts from transfers or instruction data
	if token0 != nil && token0.Info.TokenAmount.UIAmount != nil {
		event.Token0Amount = token0.Info.TokenAmount.UIAmount
		event.Token0AmountRaw = token0.Info.TokenAmount.Amount
	} else if len(data) >= 24 {
		amt := binary.LittleEndian.Uint64(data[16:24])
		uiAmt := types.ConvertToUIAmountUint64(amt, token0Decimals)
		event.Token0Amount = &uiAmt
		event.Token0AmountRaw = fmt.Sprintf("%d", amt)
	}

	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	} else if len(data) >= 16 {
		amt := binary.LittleEndian.Uint64(data[8:16])
		uiAmt := types.ConvertToUIAmountUint64(amt, token1Decimals)
		event.Token1Amount = &uiAmt
		event.Token1AmountRaw = fmt.Sprintf("%d", amt)
	}

	if lpToken != nil && lpToken.Info.TokenAmount.UIAmount != nil {
		event.LpAmount = lpToken.Info.TokenAmount.UIAmount
		event.LpAmountRaw = lpToken.Info.TokenAmount.Amount
	}

	return event
}

// ParseAddLiquidityEvent parses add liquidity event
func (p *MeteoraPoolsParser) ParseAddLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
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

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
	}

	programId := p.Adapter.GetInstructionProgramId(instruction)
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

	// Token amounts
	if token0 != nil && token0.Info.TokenAmount.UIAmount != nil {
		event.Token0Amount = token0.Info.TokenAmount.UIAmount
		event.Token0AmountRaw = token0.Info.TokenAmount.Amount
	} else if len(data) >= 32 {
		amt := binary.LittleEndian.Uint64(data[24:32])
		uiAmt := types.ConvertToUIAmountUint64(amt, token0Decimals)
		event.Token0Amount = &uiAmt
		event.Token0AmountRaw = fmt.Sprintf("%d", amt)
	}

	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	} else if len(data) >= 24 {
		amt := binary.LittleEndian.Uint64(data[16:24])
		uiAmt := types.ConvertToUIAmountUint64(amt, token1Decimals)
		event.Token1Amount = &uiAmt
		event.Token1AmountRaw = fmt.Sprintf("%d", amt)
	}

	if lpToken != nil && lpToken.Info.TokenAmount.UIAmount != nil {
		event.LpAmount = lpToken.Info.TokenAmount.UIAmount
		event.LpAmountRaw = lpToken.Info.TokenAmount.Amount
	} else if len(data) >= 16 && len(accounts) > 1 {
		amt := binary.LittleEndian.Uint64(data[8:16])
		lpDecimals := p.Adapter.GetTokenDecimals(accounts[1])
		uiAmt := types.ConvertToUIAmountUint64(amt, lpDecimals)
		event.LpAmount = &uiAmt
		event.LpAmountRaw = fmt.Sprintf("%d", amt)
	}

	return event
}

// ParseRemoveLiquidityEvent parses remove liquidity event
func (p *MeteoraPoolsParser) ParseRemoveLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	lpTransfers := p.Utils.GetLPTransfers(transfers)
	var token0, token1, lpToken *types.TransferData

	if len(lpTransfers) > 0 {
		token0 = &lpTransfers[0]
	}
	if len(lpTransfers) > 1 {
		token1 = &lpTransfers[1]
	}

	for i := range transfers {
		if transfers[i].Type == "burn" {
			lpToken = &transfers[i]
			break
		}
	}

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
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

	if len(accounts) > 0 {
		event.PoolId = accounts[0]
	}
	if len(accounts) > 1 {
		event.PoolLpMint = accounts[1]
	}

	// Token amounts
	if token0 != nil && token0.Info.TokenAmount.UIAmount != nil {
		event.Token0Amount = token0.Info.TokenAmount.UIAmount
		event.Token0AmountRaw = token0.Info.TokenAmount.Amount
	} else if len(data) >= 32 {
		amt := binary.LittleEndian.Uint64(data[24:32])
		uiAmt := types.ConvertToUIAmountUint64(amt, token0Decimals)
		event.Token0Amount = &uiAmt
		event.Token0AmountRaw = fmt.Sprintf("%d", amt)
	}

	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	} else if len(data) >= 24 {
		amt := binary.LittleEndian.Uint64(data[16:24])
		uiAmt := types.ConvertToUIAmountUint64(amt, token1Decimals)
		event.Token1Amount = &uiAmt
		event.Token1AmountRaw = fmt.Sprintf("%d", amt)
	}

	if lpToken != nil && lpToken.Info.TokenAmount.UIAmount != nil {
		event.LpAmount = lpToken.Info.TokenAmount.UIAmount
		event.LpAmountRaw = lpToken.Info.TokenAmount.Amount
	} else if len(data) >= 16 && len(accounts) > 1 {
		amt := binary.LittleEndian.Uint64(data[8:16])
		lpDecimals := p.Adapter.GetTokenDecimals(accounts[1])
		uiAmt := types.ConvertToUIAmountUint64(amt, lpDecimals)
		event.LpAmount = &uiAmt
		event.LpAmountRaw = fmt.Sprintf("%d", amt)
	}

	return event
}
