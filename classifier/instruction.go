package classifier

import (
	"bytes"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/types"
)

// InstructionClassifier organizes transaction instructions by program ID
type InstructionClassifier struct {
	adapter        *adapter.TransactionAdapter
	instructionMap map[string][]types.ClassifiedInstruction
}

// NewInstructionClassifier creates a new InstructionClassifier
func NewInstructionClassifier(adapter *adapter.TransactionAdapter) *InstructionClassifier {
	ic := &InstructionClassifier{
		adapter:        adapter,
		instructionMap: make(map[string][]types.ClassifiedInstruction),
	}
	ic.classifyInstructions()
	return ic
}

// classifyInstructions processes all instructions and groups them by program ID
func (ic *InstructionClassifier) classifyInstructions() {
	// Classify outer instructions
	for outerIndex, instruction := range ic.adapter.Instructions() {
		programId := ic.adapter.GetInstructionProgramId(instruction)
		ic.addInstruction(types.ClassifiedInstruction{
			Instruction: instruction,
			ProgramId:   programId,
			OuterIndex:  outerIndex,
			InnerIndex:  -1,
		})
	}

	// Classify inner instructions
	for _, set := range ic.adapter.InnerInstructions() {
		for innerIndex, instruction := range set.Instructions {
			programId := ic.adapter.GetInstructionProgramId(instruction)
			ic.addInstruction(types.ClassifiedInstruction{
				Instruction: instruction,
				ProgramId:   programId,
				OuterIndex:  set.Index,
				InnerIndex:  innerIndex,
			})
		}
	}
}

// addInstruction adds a classified instruction to the map
func (ic *InstructionClassifier) addInstruction(classified types.ClassifiedInstruction) {
	if classified.ProgramId == "" {
		return
	}

	instructions := ic.instructionMap[classified.ProgramId]
	instructions = append(instructions, classified)
	ic.instructionMap[classified.ProgramId] = instructions
}

// GetInstructions returns all instructions for a specific program ID
func (ic *InstructionClassifier) GetInstructions(programId string) []types.ClassifiedInstruction {
	if instructions, ok := ic.instructionMap[programId]; ok {
		return instructions
	}
	return []types.ClassifiedInstruction{}
}

// GetMultiInstructions returns all instructions for multiple program IDs
func (ic *InstructionClassifier) GetMultiInstructions(programIds []string) []types.ClassifiedInstruction {
	result := make([]types.ClassifiedInstruction, 0, len(programIds)*4)
	for _, programId := range programIds {
		result = append(result, ic.GetInstructions(programId)...)
	}
	return result
}

// GetInstructionByDiscriminator finds an instruction by its discriminator
func (ic *InstructionClassifier) GetInstructionByDiscriminator(discriminator []byte, slice int) *types.ClassifiedInstruction {
	for _, instructions := range ic.instructionMap {
		for _, instruction := range instructions {
			data := ic.adapter.GetInstructionData(instruction.Instruction)
			if len(data) >= slice && bytes.Equal(discriminator, data[:slice]) {
				return &instruction
			}
		}
	}
	return nil
}

// GetAllProgramIds returns all non-system program IDs
func (ic *InstructionClassifier) GetAllProgramIds() []string {
	var result []string
	for programId := range ic.instructionMap {
		if !isSystemProgram(programId) && !isSkipProgram(programId) {
			result = append(result, programId)
		}
	}
	return result
}

// HasProgram checks if instructions exist for a program ID
func (ic *InstructionClassifier) HasProgram(programId string) bool {
	_, ok := ic.instructionMap[programId]
	return ok
}

// GetAdapter returns the underlying TransactionAdapter
func (ic *InstructionClassifier) GetAdapter() *adapter.TransactionAdapter {
	return ic.adapter
}

// isSystemProgram checks if a program ID is a system program
func isSystemProgram(programId string) bool {
	for _, p := range constants.SYSTEM_PROGRAMS {
		if p == programId {
			return true
		}
	}
	return false
}

// isSkipProgram checks if a program ID should be skipped
func isSkipProgram(programId string) bool {
	for _, p := range constants.SKIP_PROGRAM_IDS {
		if p == programId {
			return true
		}
	}
	return false
}

// GetInstructionsByDiscriminators finds all instructions matching any of the discriminators
func (ic *InstructionClassifier) GetInstructionsByDiscriminators(programId string, discriminators map[string][]byte) map[string][]types.ClassifiedInstruction {
	result := make(map[string][]types.ClassifiedInstruction)

	instructions := ic.GetInstructions(programId)
	for _, instruction := range instructions {
		data := ic.adapter.GetInstructionData(instruction.Instruction)
		if len(data) == 0 {
			continue
		}

		for name, disc := range discriminators {
			if len(data) >= len(disc) && bytes.Equal(disc, data[:len(disc)]) {
				result[name] = append(result[name], instruction)
				break
			}
		}
	}

	return result
}

// FilterInstructionsByDiscriminator filters instructions that don't match any discriminator
func (ic *InstructionClassifier) FilterInstructionsByDiscriminator(programId string, excludeDiscriminators [][]byte) []types.ClassifiedInstruction {
	var result []types.ClassifiedInstruction

	instructions := ic.GetInstructions(programId)
	for _, instruction := range instructions {
		data := ic.adapter.GetInstructionData(instruction.Instruction)
		if len(data) == 0 {
			result = append(result, instruction)
			continue
		}

		excluded := false
		for _, disc := range excludeDiscriminators {
			if len(data) >= len(disc) && bytes.Equal(disc, data[:len(disc)]) {
				excluded = true
				break
			}
		}

		if !excluded {
			result = append(result, instruction)
		}
	}

	return result
}
