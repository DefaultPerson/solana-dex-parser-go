package meteora

import (
	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/parsers"
	"github.com/solana-dex-parser-go/types"
)

// MeteoraLiquidityParserBase is base parser for Meteora liquidity operations
type MeteoraLiquidityParserBase struct {
	*parsers.BaseLiquidityParser
}

// NewMeteoraLiquidityParserBase creates a new base liquidity parser
func NewMeteoraLiquidityParserBase(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MeteoraLiquidityParserBase {
	return &MeteoraLiquidityParserBase{
		BaseLiquidityParser: parsers.NewBaseLiquidityParser(adapter, transferActions, classifiedInstructions),
	}
}

// PoolActionResult holds the result of pool action detection
type PoolActionResult struct {
	Name string
	Type types.PoolEventType
}

// MeteoraPoolActionGetter interface for getting pool action type
type MeteoraPoolActionGetter interface {
	GetPoolAction(data []byte) interface{}
	ParseAddLiquidityEvent(instruction interface{}, index int, data []byte, transfers []types.TransferData) *types.PoolEvent
	ParseRemoveLiquidityEvent(instruction interface{}, index int, data []byte, transfers []types.TransferData) *types.PoolEvent
	ParseCreateLiquidityEvent(instruction interface{}, index int, data []byte, transfers []types.TransferData) *types.PoolEvent
}

// ParseInstruction parses a liquidity instruction
func (p *MeteoraLiquidityParserBase) ParseInstruction(
	instruction interface{},
	programId string,
	outerIndex int,
	innerIndex int,
	actionGetter MeteoraPoolActionGetter,
) *types.PoolEvent {
	data := p.Adapter.GetInstructionData(instruction)
	action := actionGetter.GetPoolAction(data)
	if action == nil {
		return nil
	}

	// Get event type from action
	var eventType types.PoolEventType
	switch v := action.(type) {
	case types.PoolEventType:
		eventType = v
	case PoolActionResult:
		eventType = v.Type
	case *PoolActionResult:
		eventType = v.Type
	default:
		return nil
	}

	transfers := p.GetTransfersForInstruction(programId, outerIndex, innerIndex, nil)
	if len(transfers) == 0 {
		transfers = p.GetTransfersForInstruction(programId, outerIndex, 0, nil)
	}

	switch eventType {
	case types.PoolEventTypeCreate:
		return actionGetter.ParseCreateLiquidityEvent(instruction, outerIndex, data, transfers)
	case types.PoolEventTypeAdd:
		return actionGetter.ParseAddLiquidityEvent(instruction, outerIndex, data, transfers)
	case types.PoolEventTypeRemove:
		return actionGetter.ParseRemoveLiquidityEvent(instruction, outerIndex, data, transfers)
	}

	return nil
}
