package raydium

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// InstructionTypeInfo holds instruction type info with name
type InstructionTypeInfo struct {
	Name string
	Type types.PoolEventType
}

// RaydiumCLPoolParser parses Raydium CL (Concentrated Liquidity) operations
type RaydiumCLPoolParser struct {
	*RaydiumLiquidityParserBase
}

// NewRaydiumCLPoolParser creates a new Raydium CL pool parser
func NewRaydiumCLPoolParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *RaydiumCLPoolParser {
	return &RaydiumCLPoolParser{
		RaydiumLiquidityParserBase: NewRaydiumLiquidityParserBase(adapter, transferActions, classifiedInstructions),
	}
}

// GetPoolAction gets the pool action type from instruction data.
// create_pool and create_customizable_pool create a pool (CREATE); opening a
// position deposits liquidity into an existing pool (ADD), as in upstream
// RaydiumCLPoolV2Parser. close_position moves no tokens and is not an event.
func (p *RaydiumCLPoolParser) GetPoolAction(data []byte) interface{} {
	if len(data) < 8 {
		return nil
	}
	instructionType := data[:8]
	cl := constants.DISCRIMINATORS.RAYDIUM_CL

	actions := []struct {
		name string
		disc []byte
		typ  types.PoolEventType
	}{
		{"createPool", cl.CREATE.CREATE_POOL, types.PoolEventTypeCreate},
		{"createCustomizablePool", cl.CREATE.CREATE_CUSTOMIZABLE_POOL, types.PoolEventTypeCreate},
		{"openPosition", cl.CREATE.OPEN_POSITION, types.PoolEventTypeAdd},
		{"openPositionV2", cl.CREATE.OPEN_POSITION_V2, types.PoolEventTypeAdd},
		{"openPositionWithToken22Nft", cl.ADD_LIQUIDITY.OPEN_POSITION_WITH_TOKEN22, types.PoolEventTypeAdd},
		{"increaseLiquidity", cl.ADD_LIQUIDITY.INCREASE_LIQUIDITY, types.PoolEventTypeAdd},
		{"increaseLiquidityV2", cl.ADD_LIQUIDITY.INCREASE_LIQUIDITY_V2, types.PoolEventTypeAdd},
		{"decreaseLiquidity", cl.REMOVE_LIQUIDITY.DECREASE_LIQUIDITY, types.PoolEventTypeRemove},
		{"decreaseLiquidityV2", cl.REMOVE_LIQUIDITY.DECREASE_LIQUIDITY_V2, types.PoolEventTypeRemove},
	}
	for _, a := range actions {
		if bytes.Equal(instructionType, a.disc) {
			return InstructionTypeInfo{Name: a.name, Type: a.typ}
		}
	}
	return nil
}

// GetEventConfig gets the event configuration for a pool event type.
// Account indices and amount offsets follow the raydium_clmm IDL:
//   - open_position / open_position_v2: pool_state 5; args tick_lower_index,
//     tick_upper_index, tick_array_lower_start_index,
//     tick_array_upper_start_index (i32 each), liquidity u128 @24,
//     amount_0_max @40, amount_1_max @48
//   - open_position_with_token22_nft: pool_state 4, same args
//   - increase_liquidity(_v2): pool_state 2; liquidity u128 @8,
//     amount_0_max @24, amount_1_max @32
//   - decrease_liquidity(_v2): pool_state 3; liquidity u128 @8,
//     amount_0_min @24, amount_1_min @32
//
// The token vaults (open_position(_v2) 12/13, open_position_with_token22_nft
// 11/12, increase_liquidity(_v2) 9/10, decrease_liquidity(_v2) 5/6) and the
// v2 instructions' vault mints (open_position_v2 20/21, token22_nft 18/19,
// increase_liquidity_v2 13/14, decrease_liquidity_v2 14/15) give each side
// its transfer and mint: a position out of range moves one token only. The
// token data amounts are slippage limits and are not reported.
func (p *RaydiumCLPoolParser) GetEventConfig(eventType types.PoolEventType, instructionType interface{}) *ParseEventConfig {
	info, ok := instructionType.(InstructionTypeInfo)
	if !ok {
		return nil
	}

	switch info.Name {
	case "openPosition", "openPositionV2":
		config := &ParseEventConfig{
			EventType:          types.PoolEventTypeAdd,
			PoolIdIndex:        5,
			LpMintIndex:        5,
			TokenAmountOffsets: &TokenAmountOffsets{Token0: 40, Token1: 48, Lp: 24},
			VaultIndexes:       []int{12, 13},
		}
		if info.Name == "openPositionV2" {
			config.MintIndexes = []int{20, 21}
		}
		return config
	case "openPositionWithToken22Nft":
		return &ParseEventConfig{
			EventType:          types.PoolEventTypeAdd,
			PoolIdIndex:        4,
			LpMintIndex:        4,
			TokenAmountOffsets: &TokenAmountOffsets{Token0: 40, Token1: 48, Lp: 24},
			VaultIndexes:       []int{11, 12},
			MintIndexes:        []int{18, 19},
		}
	case "increaseLiquidity", "increaseLiquidityV2":
		config := &ParseEventConfig{
			EventType:          types.PoolEventTypeAdd,
			PoolIdIndex:        2,
			LpMintIndex:        2,
			TokenAmountOffsets: &TokenAmountOffsets{Token0: 24, Token1: 32, Lp: 8},
			VaultIndexes:       []int{9, 10},
		}
		if info.Name == "increaseLiquidityV2" {
			config.MintIndexes = []int{13, 14}
		}
		return config
	case "decreaseLiquidity", "decreaseLiquidityV2":
		config := &ParseEventConfig{
			EventType:          types.PoolEventTypeRemove,
			PoolIdIndex:        3,
			LpMintIndex:        3,
			TokenAmountOffsets: &TokenAmountOffsets{Token0: 24, Token1: 32, Lp: 8},
			VaultIndexes:       []int{5, 6},
		}
		if info.Name == "decreaseLiquidityV2" {
			config.MintIndexes = []int{14, 15}
		}
		return config
	}
	return nil
}

// ProcessLiquidity parses liquidity events
func (p *RaydiumCLPoolParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.RAYDIUM_CL.ID {
			continue
		}
		var event *types.PoolEvent
		data := p.Adapter.GetInstructionData(ci.Instruction)
		if info, ok := p.GetPoolAction(data).(InstructionTypeInfo); ok && info.Type == types.PoolEventTypeCreate {
			event = p.parseCreateEvent(ci)
		} else {
			event = p.ParseRaydiumInstruction(ci.Instruction, ci.ProgramId, ci.OuterIndex, ci.InnerIndex, p)
		}
		if event != nil {
			events = append(events, *event)
		}
	}

	return events
}

// parseCreateEvent parses create_pool and create_customizable_pool. They
// create the pool but move no tokens (liquidity comes with the first
// position), so the event carries the pool, config and mints only. Accounts:
// pool_creator 0, amm_config 1, pool_state 2, token_mint_0 3, token_mint_1 4.
// As in upstream, the quote mint (SOL or a stablecoin) is token1.
func (p *RaydiumCLPoolParser) parseCreateEvent(ci types.ClassifiedInstruction) *types.PoolEvent {
	accounts := p.Adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 5 {
		return nil
	}
	token0Mint, token1Mint := accounts[3], accounts[4]
	if utils.GetTradeType(token0Mint, token1Mint) == types.TradeTypeBuy {
		token0Mint, token1Mint = token1Mint, token0Mint
	}
	token0Decimals := p.Adapter.GetTokenDecimals(token0Mint)
	token1Decimals := p.Adapter.GetTokenDecimals(token1Mint)

	base := p.Adapter.GetPoolEventBase(types.PoolEventTypeCreate, ci.ProgramId)
	base.Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	return &types.PoolEvent{
		PoolEventBase:  base,
		PoolId:         accounts[2],
		Config:         accounts[1],
		Token0Mint:     token0Mint,
		Token1Mint:     token1Mint,
		Token0Decimals: &token0Decimals,
		Token1Decimals: &token1Decimals,
	}
}
