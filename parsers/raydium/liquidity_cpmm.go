package raydium

import (
	"bytes"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/types"
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
	configs := map[types.PoolEventType]*ParseEventConfig{
		types.PoolEventTypeCreate: {
			EventType:   types.PoolEventTypeCreate,
			PoolIdIndex: 3,
			LpMintIndex: 6,
			TokenAmountOffsets: &TokenAmountOffsets{
				Token0: 8,
				Token1: 16,
				Lp:     0,
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
