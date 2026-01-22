package raydium

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
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

// GetPoolAction gets the pool action type from instruction data
func (p *RaydiumCLPoolParser) GetPoolAction(data []byte) interface{} {
	if len(data) < 8 {
		return nil
	}
	instructionType := data[:8]

	// CREATE discriminators
	createDiscs := map[string][]byte{
		"openPosition":   constants.DISCRIMINATORS.RAYDIUM_CL.CREATE.OPEN_POSITION,
		"openPositionV2": constants.DISCRIMINATORS.RAYDIUM_CL.CREATE.OPEN_POSITION_V2,
		"createPool":     constants.DISCRIMINATORS.RAYDIUM_CL.CREATE.CREATE_POOL,
	}
	for name, disc := range createDiscs {
		if bytes.Equal(instructionType, disc) {
			return InstructionTypeInfo{Name: name, Type: types.PoolEventTypeCreate}
		}
	}

	// ADD_LIQUIDITY discriminators
	addDiscs := map[string][]byte{
		"increaseLiquidity":        constants.DISCRIMINATORS.RAYDIUM_CL.ADD_LIQUIDITY.INCREASE_LIQUIDITY,
		"increaseLiquidityV2":      constants.DISCRIMINATORS.RAYDIUM_CL.ADD_LIQUIDITY.INCREASE_LIQUIDITY_V2,
		"openPositionWithToken22":  constants.DISCRIMINATORS.RAYDIUM_CL.ADD_LIQUIDITY.OPEN_POSITION_WITH_TOKEN22,
	}
	for name, disc := range addDiscs {
		if bytes.Equal(instructionType, disc) {
			return InstructionTypeInfo{Name: name, Type: types.PoolEventTypeAdd}
		}
	}

	// REMOVE_LIQUIDITY discriminators
	removeDiscs := map[string][]byte{
		"decreaseLiquidity":   constants.DISCRIMINATORS.RAYDIUM_CL.REMOVE_LIQUIDITY.DECREASE_LIQUIDITY,
		"decreaseLiquidityV2": constants.DISCRIMINATORS.RAYDIUM_CL.REMOVE_LIQUIDITY.DECREASE_LIQUIDITY_V2,
	}
	for name, disc := range removeDiscs {
		if bytes.Equal(instructionType, disc) {
			return InstructionTypeInfo{Name: name, Type: types.PoolEventTypeRemove}
		}
	}

	return nil
}

// GetEventConfig gets the event configuration for a pool event type
func (p *RaydiumCLPoolParser) GetEventConfig(eventType types.PoolEventType, instructionType interface{}) *ParseEventConfig {
	info, ok := instructionType.(InstructionTypeInfo)
	if !ok {
		return nil
	}

	switch eventType {
	case types.PoolEventTypeCreate:
		poolIdIndex := 4
		if info.Name == "openPosition" || info.Name == "openPositionV2" {
			poolIdIndex = 5
		}
		return &ParseEventConfig{
			EventType:   types.PoolEventTypeCreate,
			PoolIdIndex: poolIdIndex,
			LpMintIndex: poolIdIndex,
		}
	case types.PoolEventTypeAdd:
		return &ParseEventConfig{
			EventType:   types.PoolEventTypeAdd,
			PoolIdIndex: 2,
			LpMintIndex: 2,
			TokenAmountOffsets: &TokenAmountOffsets{
				Token0: 32,
				Token1: 24,
				Lp:     8,
			},
		}
	case types.PoolEventTypeRemove:
		return &ParseEventConfig{
			EventType:   types.PoolEventTypeRemove,
			PoolIdIndex: 3,
			LpMintIndex: 3,
			TokenAmountOffsets: &TokenAmountOffsets{
				Token0: 32,
				Token1: 24,
				Lp:     8,
			},
		}
	}
	return nil
}

// ProcessLiquidity parses liquidity events
func (p *RaydiumCLPoolParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.RAYDIUM_CL.ID {
			innerIdx := ci.InnerIndex
			if innerIdx < 0 {
				innerIdx = 0
			}
			event := p.ParseRaydiumInstruction(ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx, p)
			if event != nil {
				events = append(events, *event)
			}
		}
	}

	return events
}
