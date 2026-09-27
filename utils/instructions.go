package utils

import (
	"sort"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// FindEventEmitter returns the instruction that emitted the Anchor self-CPI
// event instruction event (emit_cpi!: the program invokes itself with the
// event as data), provided it is not an event itself and match accepts it
// (a nil match accepts any). candidates are instructions of the event's
// program, in any order; the emitter is taken from them when present.
//
// With stack heights the emitter is the event's parent instruction
// (adapter.GetParentInstruction); nil when the parent belongs to another
// program or is rejected. Without stack heights (older transactions, input
// without stackHeight) it is the nearest preceding accepted instruction of
// the event's program in the same outer instruction.
func FindEventEmitter(
	a *adapter.TransactionAdapter,
	candidates []types.ClassifiedInstruction,
	event types.ClassifiedInstruction,
	match func(data []byte, accounts []string) bool,
) *types.ClassifiedInstruction {
	accept := func(ci *types.ClassifiedInstruction) bool {
		data := a.GetInstructionData(ci.Instruction)
		return !constants.IsAnchorEvent(data) && (match == nil || match(data, a.GetInstructionAccounts(ci.Instruction)))
	}

	if parent, parentInner, ok := a.GetParentInstruction(event.OuterIndex, event.InnerIndex); ok {
		if a.GetInstructionProgramId(parent) != event.ProgramId {
			return nil
		}
		emitter := &types.ClassifiedInstruction{
			Instruction: parent,
			ProgramId:   event.ProgramId,
			OuterIndex:  event.OuterIndex,
			InnerIndex:  parentInner,
			StackHeight: a.GetInstructionStackHeight(event.OuterIndex, parentInner),
		}
		for i := range candidates {
			if candidates[i].OuterIndex == event.OuterIndex && candidates[i].InnerIndex == parentInner {
				emitter = &candidates[i]
				break
			}
		}
		if !accept(emitter) {
			return nil
		}
		return emitter
	}

	var found *types.ClassifiedInstruction
	for i := range candidates {
		ci := &candidates[i]
		if ci.ProgramId != event.ProgramId || ci.OuterIndex != event.OuterIndex || ci.InnerIndex >= event.InnerIndex {
			continue
		}
		if (found == nil || ci.InnerIndex > found.InnerIndex) && accept(ci) {
			found = ci
		}
	}
	return found
}

// EmittedEvents returns the Anchor self-CPI events among candidates (in their
// order) that the instruction ci emitted, per FindEventEmitter
func EmittedEvents(a *adapter.TransactionAdapter, candidates []types.ClassifiedInstruction, ci types.ClassifiedInstruction) []types.ClassifiedInstruction {
	var events []types.ClassifiedInstruction
	for _, e := range candidates {
		if e.ProgramId != ci.ProgramId || e.OuterIndex != ci.OuterIndex || e.InnerIndex <= ci.InnerIndex ||
			!constants.IsAnchorEvent(a.GetInstructionData(e.Instruction)) {
			continue
		}
		if emitter := FindEventEmitter(a, candidates, e, nil); emitter != nil && emitter.InnerIndex == ci.InnerIndex {
			events = append(events, e)
		}
	}
	return events
}

// CPIGroup returns the inner indexes [first, last] of the instructions that
// ran inside ci (its CPI subtree), empty when last < first. With stack
// heights these are the inner instructions after ci with a greater stack
// height, up to the next one at ci's height or above. Without them the group
// ends before the next instruction of ci's program in the same outer
// instruction.
func CPIGroup(a *adapter.TransactionAdapter, ci types.ClassifiedInstruction) (first, last int) {
	first, last = ci.InnerIndex+1, ci.InnerIndex
	height := a.GetInstructionStackHeight(ci.OuterIndex, ci.InnerIndex)
	for j := first; ; j++ {
		ix := a.GetInnerInstruction(ci.OuterIndex, j)
		if ix == nil {
			break
		}
		if h := adapter.InstructionStackHeight(ix); height > 0 && h > 0 {
			if h <= height {
				break
			}
		} else if a.GetInstructionProgramId(ix) == ci.ProgramId {
			break
		}
		last = j
	}
	return first, last
}

// SortInstructionsByExecution sorts instructions in place in execution
// order: an outer instruction first, then its inner instructions by inner
// index. The sort is stable.
func SortInstructionsByExecution(instructions []types.ClassifiedInstruction) {
	sort.SliceStable(instructions, func(i, j int) bool {
		if instructions[i].OuterIndex != instructions[j].OuterIndex {
			return instructions[i].OuterIndex < instructions[j].OuterIndex
		}
		return instructions[i].InnerIndex < instructions[j].InnerIndex
	})
}
