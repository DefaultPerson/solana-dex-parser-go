package alt

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// AltInstructionType represents ALT instruction types
type AltInstructionType uint32

const (
	CreateLookupTable     AltInstructionType = 0
	FreezeLookupTable     AltInstructionType = 1
	ExtendLookupTable     AltInstructionType = 2
	DeactivateLookupTable AltInstructionType = 3
	CloseLookupTable      AltInstructionType = 4
)

// AltEventParser parses Address Lookup Table events
type AltEventParser struct {
	adapter                *adapter.TransactionAdapter
	classifiedInstructions []types.ClassifiedInstruction
}

// NewAltEventParser creates a new AltEventParser
func NewAltEventParser(
	adapter *adapter.TransactionAdapter,
	classifiedInstructions []types.ClassifiedInstruction,
) *AltEventParser {
	return &AltEventParser{
		adapter:                adapter,
		classifiedInstructions: classifiedInstructions,
	}
}

// ProcessEvents parses ALT events from classified instructions
func (p *AltEventParser) ProcessEvents() []types.AltEvent {
	events := make([]types.AltEvent, 0)

	for _, cInst := range p.classifiedInstructions {
		if cInst.ProgramId != constants.ALT_PROGRAM_ID {
			continue
		}

		data := p.adapter.GetInstructionData(cInst.Instruction)
		if len(data) < 4 {
			continue
		}

		eventData := p.decodeAltInstruction(cInst, data)
		if eventData != nil {
			eventData.Idx = cInst.GetIdx()
			events = append(events, *eventData)
		}
	}

	return events
}

func (p *AltEventParser) decodeAltInstruction(cInst types.ClassifiedInstruction, data []byte) *types.AltEvent {
	reader := utils.NewBinaryReader(data)
	accounts := p.adapter.GetInstructionAccounts(cInst.Instruction)

	instructionTypeValue, err := reader.ReadU32()
	if err != nil {
		return nil
	}

	instructionType := AltInstructionType(instructionTypeValue)

	switch instructionType {
	case CreateLookupTable:
		if len(accounts) < 3 {
			return nil
		}
		recentSlot, err := reader.ReadU64()
		if err != nil {
			return nil
		}
		return &types.AltEvent{
			Type:         "CreateLookupTable",
			AltAccount:   accounts[0],
			AltAuthority: accounts[1],
			PayerAccount: accounts[2],
			RecentSlot:   recentSlot,
		}

	case FreezeLookupTable:
		if len(accounts) < 2 {
			return nil
		}
		return &types.AltEvent{
			Type:         "FreezeLookupTable",
			AltAccount:   accounts[0],
			AltAuthority: accounts[1],
		}

	case ExtendLookupTable:
		if len(accounts) < 2 {
			return nil
		}
		var payerAccount string
		if len(accounts) > 2 {
			payerAccount = accounts[2]
		}

		count, err := reader.ReadU64()
		if err != nil {
			return nil
		}

		newAddresses := make([]string, 0, count)
		for i := uint64(0); i < count; i++ {
			pubkey, err := reader.ReadPubkey()
			if err != nil {
				break
			}
			newAddresses = append(newAddresses, pubkey)
		}

		return &types.AltEvent{
			Type:         "ExtendLookupTable",
			AltAccount:   accounts[0],
			AltAuthority: accounts[1],
			PayerAccount: payerAccount,
			NewAddresses: newAddresses,
		}

	case DeactivateLookupTable:
		if len(accounts) < 2 {
			return nil
		}
		return &types.AltEvent{
			Type:         "DeactivateLookupTable",
			AltAccount:   accounts[0],
			AltAuthority: accounts[1],
		}

	case CloseLookupTable:
		if len(accounts) < 3 {
			return nil
		}
		return &types.AltEvent{
			Type:         "CloseLookupTable",
			AltAccount:   accounts[0],
			AltAuthority: accounts[1],
			Recipient:    accounts[2],
		}
	}

	return nil
}
