package pumpfun

import (
	"fmt"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// PumpswapLiquidityParser parses Pumpswap liquidity events
type PumpswapLiquidityParser struct {
	adapter                *adapter.TransactionAdapter
	transferActions        map[string][]types.TransferData
	classifiedInstructions []types.ClassifiedInstruction
	eventParser            *PumpswapEventParser
}

// NewPumpswapLiquidityParser creates a new Pumpswap liquidity parser
func NewPumpswapLiquidityParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *PumpswapLiquidityParser {
	return &PumpswapLiquidityParser{
		adapter:                adapter,
		transferActions:        transferActions,
		classifiedInstructions: classifiedInstructions,
		eventParser:            NewPumpswapEventParser(adapter, transferActions),
	}
}

// ProcessLiquidity parses Pumpswap liquidity events
func (p *PumpswapLiquidityParser) ProcessLiquidity() []types.PoolEvent {
	// Parse events and filter for liquidity types
	events := p.eventParser.ParseInstructions(p.classifiedInstructions)

	var liquidityEvents []*PumpswapEvent
	for _, event := range events {
		if event.Type == "CREATE" || event.Type == "ADD" || event.Type == "REMOVE" {
			liquidityEvents = append(liquidityEvents, event)
		}
	}

	if len(liquidityEvents) == 0 {
		return nil
	}

	return p.parseLiquidityEvents(liquidityEvents)
}

// parseLiquidityEvents converts Pumpswap events to PoolEvents
func (p *PumpswapLiquidityParser) parseLiquidityEvents(events []*PumpswapEvent) []types.PoolEvent {
	var poolEvents []types.PoolEvent

	for _, event := range events {
		var poolEvent *types.PoolEvent

		switch event.Type {
		case "CREATE":
			poolEvent = p.parseCreateEvent(event)
		case "ADD":
			poolEvent = p.parseDepositEvent(event)
		case "REMOVE":
			poolEvent = p.parseWithdrawEvent(event)
		}

		if poolEvent != nil {
			poolEvents = append(poolEvents, *poolEvent)
		}
	}

	return poolEvents
}

// parseCreateEvent parses a create pool event
func (p *PumpswapLiquidityParser) parseCreateEvent(event *PumpswapEvent) *types.PoolEvent {
	data, ok := event.Data.(*PumpswapCreatePoolEventData)
	if !ok || data == nil {
		return nil
	}

	base := p.adapter.GetPoolEventBase(types.PoolEventTypeCreate, constants.DEX_PROGRAMS.PUMP_SWAP.ID)
	base.Idx = event.Idx

	token0Amount := types.ConvertToUIAmountUint64(data.BaseAmountIn, data.BaseMintDecimals)
	token1Amount := types.ConvertToUIAmountUint64(data.QuoteAmountIn, data.QuoteMintDecimals)

	return &types.PoolEvent{
		PoolEventBase:   base,
		PoolId:          data.Pool,
		PoolLpMint:      data.LpMint,
		Token0Mint:      data.BaseMint,
		Token1Mint:      data.QuoteMint,
		Token0Amount:    &token0Amount,
		Token0AmountRaw: fmt.Sprintf("%d", data.BaseAmountIn),
		Token1Amount:    &token1Amount,
		Token1AmountRaw: fmt.Sprintf("%d", data.QuoteAmountIn),
		Token0Decimals:  &data.BaseMintDecimals,
		Token1Decimals:  &data.QuoteMintDecimals,
	}
}

// parseDepositEvent parses an add liquidity event
func (p *PumpswapLiquidityParser) parseDepositEvent(event *PumpswapEvent) *types.PoolEvent {
	data, ok := event.Data.(*PumpswapDepositEventData)
	if !ok || data == nil {
		return nil
	}

	base := p.adapter.GetPoolEventBase(types.PoolEventTypeAdd, constants.DEX_PROGRAMS.PUMP_SWAP.ID)
	base.Idx = event.Idx

	// Get token mints from SPL token map
	token0Mint := p.adapter.GetSplTokenMint(data.UserBaseTokenAccount)
	token1Mint := p.adapter.GetSplTokenMint(data.UserQuoteTokenAccount)
	lpMint := p.adapter.GetSplTokenMint(data.UserPoolTokenAccount)

	token0Decimals := p.adapter.GetTokenDecimals(token0Mint)
	token1Decimals := p.adapter.GetTokenDecimals(token1Mint)

	token0Amount := types.ConvertToUIAmountUint64(data.BaseAmountIn, token0Decimals)
	token1Amount := types.ConvertToUIAmountUint64(data.QuoteAmountIn, token1Decimals)

	return &types.PoolEvent{
		PoolEventBase:   base,
		PoolId:          data.Pool,
		PoolLpMint:      lpMint,
		Token0Mint:      token0Mint,
		Token1Mint:      token1Mint,
		Token0Amount:    &token0Amount,
		Token0AmountRaw: fmt.Sprintf("%d", data.BaseAmountIn),
		Token1Amount:    &token1Amount,
		Token1AmountRaw: fmt.Sprintf("%d", data.QuoteAmountIn),
		Token0Decimals:  &token0Decimals,
		Token1Decimals:  &token1Decimals,
	}
}

// parseWithdrawEvent parses a remove liquidity event
func (p *PumpswapLiquidityParser) parseWithdrawEvent(event *PumpswapEvent) *types.PoolEvent {
	data, ok := event.Data.(*PumpswapWithdrawEventData)
	if !ok || data == nil {
		return nil
	}

	base := p.adapter.GetPoolEventBase(types.PoolEventTypeRemove, constants.DEX_PROGRAMS.PUMP_SWAP.ID)
	base.Idx = event.Idx

	// Get token mints from SPL token map
	token0Mint := p.adapter.GetSplTokenMint(data.UserBaseTokenAccount)
	token1Mint := p.adapter.GetSplTokenMint(data.UserQuoteTokenAccount)
	lpMint := p.adapter.GetSplTokenMint(data.UserPoolTokenAccount)

	token0Decimals := p.adapter.GetTokenDecimals(token0Mint)
	token1Decimals := p.adapter.GetTokenDecimals(token1Mint)

	token0Amount := types.ConvertToUIAmountUint64(data.BaseAmountOut, token0Decimals)
	token1Amount := types.ConvertToUIAmountUint64(data.QuoteAmountOut, token1Decimals)

	return &types.PoolEvent{
		PoolEventBase:   base,
		PoolId:          data.Pool,
		PoolLpMint:      lpMint,
		Token0Mint:      token0Mint,
		Token1Mint:      token1Mint,
		Token0Amount:    &token0Amount,
		Token0AmountRaw: fmt.Sprintf("%d", data.BaseAmountOut),
		Token1Amount:    &token1Amount,
		Token1AmountRaw: fmt.Sprintf("%d", data.QuoteAmountOut),
		Token0Decimals:  &token0Decimals,
		Token1Decimals:  &token1Decimals,
	}
}
