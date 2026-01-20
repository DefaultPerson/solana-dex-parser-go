package raydium

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// RaydiumV4PoolParser parses Raydium V4 liquidity operations
type RaydiumV4PoolParser struct {
	*RaydiumLiquidityParserBase
}

// NewRaydiumV4PoolParser creates a new Raydium V4 pool parser
func NewRaydiumV4PoolParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *RaydiumV4PoolParser {
	return &RaydiumV4PoolParser{
		RaydiumLiquidityParserBase: NewRaydiumLiquidityParserBase(adapter, transferActions, classifiedInstructions),
	}
}

// GetPoolAction gets the pool action type from instruction data
func (p *RaydiumV4PoolParser) GetPoolAction(data []byte) interface{} {
	if len(data) < 1 {
		return nil
	}
	instructionType := data[:1]
	if bytes.Equal(instructionType, constants.DISCRIMINATORS.RAYDIUM.CREATE) {
		return types.PoolEventTypeCreate
	}
	if bytes.Equal(instructionType, constants.DISCRIMINATORS.RAYDIUM.ADD_LIQUIDITY) {
		return types.PoolEventTypeAdd
	}
	if bytes.Equal(instructionType, constants.DISCRIMINATORS.RAYDIUM.REMOVE_LIQUIDITY) {
		return types.PoolEventTypeRemove
	}
	return nil
}

// GetEventConfig gets the event configuration for a pool event type
func (p *RaydiumV4PoolParser) GetEventConfig(eventType types.PoolEventType, instructionType interface{}) *ParseEventConfig {
	configs := map[types.PoolEventType]*ParseEventConfig{
		types.PoolEventTypeCreate: {EventType: types.PoolEventTypeCreate, PoolIdIndex: 4, LpMintIndex: 7},
		types.PoolEventTypeAdd:    {EventType: types.PoolEventTypeAdd, PoolIdIndex: 1, LpMintIndex: 5},
		types.PoolEventTypeRemove: {EventType: types.PoolEventTypeRemove, PoolIdIndex: 1, LpMintIndex: 5},
	}
	return configs[eventType]
}

// ProcessLiquidity parses liquidity events
func (p *RaydiumV4PoolParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.RAYDIUM_V4.ID ||
			ci.ProgramId == constants.DEX_PROGRAMS.RAYDIUM_AMM.ID {
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
