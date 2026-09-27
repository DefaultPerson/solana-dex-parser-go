package raydium

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// RaydiumCPMMPoolParser parses Raydium CPMM liquidity operations
type RaydiumCPMMPoolParser struct {
	*RaydiumLiquidityParserBase
}

// NewRaydiumCPMMPoolParser creates a new Raydium CPMM pool parser
func NewRaydiumCPMMPoolParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *RaydiumCPMMPoolParser {
	return &RaydiumCPMMPoolParser{
		RaydiumLiquidityParserBase: NewRaydiumLiquidityParserBase(adapter, transferActions, classifiedInstructions),
	}
}

// GetPoolAction gets the pool action type from instruction data
func (p *RaydiumCPMMPoolParser) GetPoolAction(data []byte) interface{} {
	if len(data) < 8 {
		return nil
	}
	instructionType := data[:8]
	if bytes.Equal(instructionType, constants.DISCRIMINATORS.RAYDIUM_CPMM.CREATE) {
		return types.PoolEventTypeCreate
	}
	if bytes.Equal(instructionType, constants.DISCRIMINATORS.RAYDIUM_CPMM.INITIALIZE_WITH_PERMISSION) {
		return InstructionTypeInfo{Name: "initializeWithPermission", Type: types.PoolEventTypeCreate}
	}
	if bytes.Equal(instructionType, constants.DISCRIMINATORS.RAYDIUM_CPMM.ADD_LIQUIDITY) {
		return types.PoolEventTypeAdd
	}
	if bytes.Equal(instructionType, constants.DISCRIMINATORS.RAYDIUM_CPMM.REMOVE_LIQUIDITY) {
		return types.PoolEventTypeRemove
	}
	return nil
}

// GetEventConfig gets the event configuration for a pool event type
func (p *RaydiumCPMMPoolParser) GetEventConfig(eventType types.PoolEventType, instructionType interface{}) *ParseEventConfig {
	// initialize_with_permission has payer and creator first: pool_state 4, lp_mint 7
	if info, ok := instructionType.(InstructionTypeInfo); ok && info.Name == "initializeWithPermission" {
		return &ParseEventConfig{
			EventType:          types.PoolEventTypeCreate,
			PoolIdIndex:        4,
			LpMintIndex:        7,
			TokenAmountOffsets: &TokenAmountOffsets{Token0: 8, Token1: 16, Lp: -1},
		}
	}
	configs := map[types.PoolEventType]*ParseEventConfig{
		types.PoolEventTypeCreate: {
			EventType:   types.PoolEventTypeCreate,
			PoolIdIndex: 3,
			LpMintIndex: 6,
			// initialize args: init_amount_0, init_amount_1, open_time (no LP amount)
			TokenAmountOffsets: &TokenAmountOffsets{
				Token0: 8,
				Token1: 16,
				Lp:     -1,
			},
		},
		types.PoolEventTypeAdd: {
			EventType:   types.PoolEventTypeAdd,
			PoolIdIndex: 2,
			LpMintIndex: 12,
			TokenAmountOffsets: &TokenAmountOffsets{
				Token0: 16,
				Token1: 24,
				Lp:     8,
			},
		},
		types.PoolEventTypeRemove: {
			EventType:   types.PoolEventTypeRemove,
			PoolIdIndex: 2,
			LpMintIndex: 12,
			TokenAmountOffsets: &TokenAmountOffsets{
				Token0: 16,
				Token1: 24,
				Lp:     8,
			},
		},
	}
	return configs[eventType]
}

// ProcessLiquidity parses liquidity events
func (p *RaydiumCPMMPoolParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID {
			event := p.ParseRaydiumInstruction(ci.Instruction, ci.ProgramId, ci.OuterIndex, ci.InnerIndex, p)
			if event != nil {
				events = append(events, *event)
			}
		}
	}

	return events
}
