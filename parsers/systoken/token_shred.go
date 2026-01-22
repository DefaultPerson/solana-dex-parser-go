package systoken

import (
	"fmt"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// SystemTokenShredParser parses system token instructions from shred-stream
type SystemTokenShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
	txUtils    *utils.TransactionUtils
}

// NewSystemTokenShredParser creates a new SystemTokenShredParser
func NewSystemTokenShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *SystemTokenShredParser {
	return &SystemTokenShredParser{
		adapter:    adapter,
		classifier: classifier,
		txUtils:    utils.NewTransactionUtils(adapter),
	}
}

// ProcessTokenInstructions processes SPL Token instructions and returns parsed results
func (p *SystemTokenShredParser) ProcessTokenInstructions() []interface{} {
	return p.processTokenShred(constants.TOKEN_PROGRAM_ID)
}

// ProcessToken2022Instructions processes SPL Token 2022 instructions
func (p *SystemTokenShredParser) ProcessToken2022Instructions() []interface{} {
	return p.processTokenShred(constants.TOKEN_2022_PROGRAM_ID)
}

// ProcessNativeInstructions processes System program (SOL) transfers
func (p *SystemTokenShredParser) ProcessNativeInstructions() []interface{} {
	return p.processTokenShred(constants.SYSTEM_PROGRAM_ID)
}

// ProcessTypedTokenInstructions returns typed ParsedShredInstruction results for SPL Token
func (p *SystemTokenShredParser) ProcessTypedTokenInstructions() []types.ParsedShredInstruction {
	return p.processTypedTokenShred(constants.TOKEN_PROGRAM_ID)
}

// ProcessTypedToken2022Instructions returns typed ParsedShredInstruction results for Token 2022
func (p *SystemTokenShredParser) ProcessTypedToken2022Instructions() []types.ParsedShredInstruction {
	return p.processTypedTokenShred(constants.TOKEN_2022_PROGRAM_ID)
}

// ProcessTypedNativeInstructions returns typed ParsedShredInstruction results for System (SOL)
func (p *SystemTokenShredParser) ProcessTypedNativeInstructions() []types.ParsedShredInstruction {
	return p.processTypedTokenShred(constants.SYSTEM_PROGRAM_ID)
}

func (p *SystemTokenShredParser) processTokenShred(programID string) []interface{} {
	var events []interface{}
	extraTypes := []string{"mintTo", "burn", "mintToChecked", "burnChecked"}

	instructions := p.classifier.GetInstructions(programID)

	for _, ci := range instructions {
		idx := formatInstructionIdx(ci.OuterIndex, ci.InnerIndex)
		transfer := p.txUtils.ParseInstructionAction(ci.Instruction, idx, extraTypes)

		if transfer != nil {
			// Check if it's a fee transfer
			if constants.IsFeeAccount(transfer.Info.Destination) ||
				constants.IsFeeAccount(transfer.Info.DestinationOwner) {
				transfer.IsFee = true
			}

			event := &TokenInstruction{
				Type:        transfer.Type,
				Data:        transfer,
				ProgramID:   transfer.ProgramId,
				ProgramName: getSysProgramName(transfer.ProgramId),
				Slot:        p.adapter.Slot(),
				Timestamp:   p.adapter.BlockTime(),
				Signature:   p.adapter.Signature(),
				Idx:         idx,
				Signer:      p.adapter.Signers(),
			}
			events = append(events, event)
		}
	}

	return events
}

func (p *SystemTokenShredParser) processTypedTokenShred(programID string) []types.ParsedShredInstruction {
	var events []types.ParsedShredInstruction
	extraTypes := []string{"mintTo", "burn", "mintToChecked", "burnChecked"}

	instructions := p.classifier.GetInstructions(programID)

	for _, ci := range instructions {
		idx := formatInstructionIdx(ci.OuterIndex, ci.InnerIndex)
		transfer := p.txUtils.ParseInstructionAction(ci.Instruction, idx, extraTypes)

		if transfer != nil {
			// Check if it's a fee transfer
			if constants.IsFeeAccount(transfer.Info.Destination) ||
				constants.IsFeeAccount(transfer.Info.DestinationOwner) {
				transfer.IsFee = true
			}

			event := types.ParsedShredInstruction{
				ProgramID:   transfer.ProgramId,
				ProgramName: getSysProgramName(transfer.ProgramId),
				Action:      transfer.Type,
				Transfer:    transfer,
				Accounts:    p.adapter.GetInstructionAccounts(ci.Instruction),
				Idx:         idx,
			}
			events = append(events, event)
		}
	}

	return events
}

// TokenInstruction represents a parsed token instruction
type TokenInstruction struct {
	Type        string              `json:"type"`
	Data        *types.TransferData `json:"data"`
	ProgramID   string              `json:"programId"`
	ProgramName string              `json:"programName"`
	Slot        uint64              `json:"slot"`
	Timestamp   int64               `json:"timestamp"`
	Signature   string              `json:"signature"`
	Idx         string              `json:"idx"`
	Signer      []string            `json:"signer"`
}

func formatInstructionIdx(outerIndex int, innerIndex int) string {
	if innerIndex < 0 {
		return fmt.Sprintf("%d", outerIndex)
	}
	return fmt.Sprintf("%d-%d", outerIndex, innerIndex)
}

func getSysProgramName(programID string) string {
	switch programID {
	case constants.SYSTEM_PROGRAM_ID:
		return "System"
	case constants.TOKEN_PROGRAM_ID:
		return "Token"
	case constants.TOKEN_2022_PROGRAM_ID:
		return "Token2022"
	}
	return "Unknown"
}
