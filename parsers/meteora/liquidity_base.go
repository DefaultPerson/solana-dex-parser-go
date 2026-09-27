package meteora

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
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

// ParseInstruction parses a liquidity instruction. innerIndex is the
// instruction's inner index, -1 for an outer instruction; the event Idx is
// "outer" or "outer-inner" (utils.FormatIdx).
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

	transfers := p.InstructionTransfers(programId, outerIndex, innerIndex)

	var event *types.PoolEvent
	switch eventType {
	case types.PoolEventTypeCreate:
		event = actionGetter.ParseCreateLiquidityEvent(instruction, outerIndex, data, transfers)
	case types.PoolEventTypeAdd:
		event = actionGetter.ParseAddLiquidityEvent(instruction, outerIndex, data, transfers)
	case types.PoolEventTypeRemove:
		event = actionGetter.ParseRemoveLiquidityEvent(instruction, outerIndex, data, transfers)
	}
	if event != nil {
		event.Idx = utils.FormatIdx(outerIndex, innerIndex)
	}
	return event
}

// followingEvents returns the self-CPI events the instruction of programId
// at (outerIndex, innerIndex) emitted (utils.EmittedEvents: the event's
// parent instruction when stack heights are known, else the program's event
// instructions that follow it up to its next non-event instruction)
func followingEvents(adapt *adapter.TransactionAdapter, instructions []types.ClassifiedInstruction, programId string, outerIndex, innerIndex int) []types.ClassifiedInstruction {
	return utils.EmittedEvents(adapt, instructions, types.ClassifiedInstruction{ProgramId: programId, OuterIndex: outerIndex, InnerIndex: innerIndex})
}

// findEvent returns the data of the first of events with discriminator
func findEvent(adapt *adapter.TransactionAdapter, events []types.ClassifiedInstruction, discriminator []byte) []byte {
	for _, e := range events {
		if data := adapt.GetInstructionData(e.Instruction); constants.MatchDiscriminator(data, discriminator) {
			return data
		}
	}
	return nil
}

// InstructionTransfers returns the transfers of the instruction at
// (outerIndex, innerIndex), innerIndex -1 for an outer instruction. Transfer
// grouping switches to every non-system CPI, so the transfers an instruction
// makes after emitting a self-CPI event are grouped under that event
// instruction; they are included. The events of an instruction are the
// program's event instructions that follow it in the same outer instruction,
// up to its next non-event instruction. Native SOL transfers of the System
// program (account rent, fees) are not pool deposits or withdrawals and are
// left out: the pools hold wrapped SOL.
func (p *MeteoraLiquidityParserBase) InstructionTransfers(programId string, outerIndex int, innerIndex int) []types.TransferData {
	var transfers []types.TransferData
	add := func(group []types.TransferData) {
		for _, t := range group {
			if t.ProgramId != constants.SYSTEM_PROGRAM_ID {
				transfers = append(transfers, t)
			}
		}
	}
	add(p.GetTransfersForInstruction(programId, outerIndex, innerIndex, nil))
	for _, ci := range followingEvents(p.Adapter, p.ClassifiedInstructions, programId, outerIndex, innerIndex) {
		add(p.GetTransfersForInstruction(programId, outerIndex, ci.InnerIndex, nil))
	}
	return transfers
}
