package meteora

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// MeteoraDBCEventParser parses Meteora DBC events
type MeteoraDBCEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
	utils           *utils.TransactionUtils
}

// NewMeteoraDBCEventParser creates a new event parser
func NewMeteoraDBCEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *MeteoraDBCEventParser {
	return &MeteoraDBCEventParser{
		adapter:         adapter,
		transferActions: transferActions,
		utils:           utils.NewTransactionUtils(adapter),
	}
}

// ParseInstructions parses classified instructions into meme events
func (p *MeteoraDBCEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	for _, ci := range instructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.METEORA_DBC.ID {
			continue
		}

		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		innerIdx := ci.InnerIndex
		if innerIdx < 0 {
			innerIdx = 0
		}

		var event *types.MemeEvent

		disc := data[:8]

		// Check for trade discriminators
		if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.SWAP) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.SWAP_V2) {
			event = p.decodeTradeEvent(data[8:], ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx)
		}

		// Check for create discriminators
		if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_SPL) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022) {
			event = p.decodeCreateEvent(data[8:], ci.Instruction)
		}

		// Check for migrate discriminators
		if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.METEORA_DBC_MIGRATE_DAMM) {
			event = p.decodeDBCMigrateDammEvent(ci.Instruction)
		}
		if bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.METEORA_DBC_MIGRATE_DAMM_V2) {
			event = p.decodeDBCMigrateDammV2Event(ci.Instruction)
		}

		if event != nil {
			event.Protocol = constants.DEX_PROGRAMS.METEORA_DBC.Name
			event.Signature = p.adapter.Signature()
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Idx = fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx)
			events = append(events, event)
		}
	}

	// Sort by Idx
	sort.Slice(events, func(i, j int) bool {
		return events[i].Idx < events[j].Idx
	})

	return events
}

// decodeTradeEvent decodes a trade event
func (p *MeteoraDBCEventParser) decodeTradeEvent(data []byte, instruction interface{}, programId string, outerIndex int, innerIndex int) *types.MemeEvent {
	if len(data) < 16 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	inputAmount := reader.ReadU64AsBigInt()
	outputAmount := reader.ReadU64AsBigInt()

	if reader.HasError() {
		return nil
	}

	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 10 {
		return nil
	}

	userAccount := accounts[9]
	baseMint := accounts[7]
	quoteMint := accounts[8]
	inputTokenAccount := accounts[3]
	outputTokenAccount := accounts[4]

	// Determine trade type based on accounts
	signer := p.adapter.Signer()
	tradeType := getAccountTradeType(signer, baseMint, inputTokenAccount, outputTokenAccount)

	var inputMint, outputMint string
	if tradeType == types.TradeTypeSell {
		inputMint = baseMint
		outputMint = quoteMint
	} else {
		inputMint = quoteMint
		outputMint = baseMint
	}

	event := &types.MemeEvent{
		Type:         tradeType,
		BaseMint:     baseMint,
		QuoteMint:    quoteMint,
		BondingCurve: accounts[2],
		Pool:         accounts[2],
		User:         userAccount,
		InputToken: &types.TokenInfo{
			Mint:      inputMint,
			AmountRaw: inputAmount.String(),
		},
		OutputToken: &types.TokenInfo{
			Mint:      outputMint,
			AmountRaw: outputAmount.String(),
		},
	}

	// Try to get better token info from transfers
	transfers := p.getTransfersForInstruction(programId, outerIndex, innerIndex)
	if len(transfers) >= 2 {
		trade := p.utils.ProcessSwapData(transfers[:2], types.DexInfo{}, false)
		if trade != nil {
			event.InputToken = &trade.InputToken
			event.OutputToken = &trade.OutputToken
		}
	}

	return event
}

// decodeCreateEvent decodes a create event
func (p *MeteoraDBCEventParser) decodeCreateEvent(data []byte, instruction interface{}) *types.MemeEvent {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	name, err := reader.ReadString()
	if err != nil {
		return nil
	}
	symbol, err := reader.ReadString()
	if err != nil {
		return nil
	}
	uri, err := reader.ReadString()
	if err != nil {
		return nil
	}

	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 10 {
		return nil
	}

	return &types.MemeEvent{
		Type:           types.TradeTypeCreate,
		Name:           name,
		Symbol:         symbol,
		URI:            uri,
		User:           accounts[2],
		BaseMint:       accounts[3],
		QuoteMint:      accounts[4],
		Pool:           accounts[5],
		BondingCurve:   accounts[5],
		PlatformConfig: accounts[0],
	}
}

// decodeDBCMigrateDammEvent decodes a migrate to DAMM event
func (p *MeteoraDBCEventParser) decodeDBCMigrateDammEvent(instruction interface{}) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 8 {
		return nil
	}

	return &types.MemeEvent{
		Type:           types.TradeTypeMigrate,
		BaseMint:       accounts[7],
		QuoteMint:      accounts[8],
		PlatformConfig: accounts[2],
		BondingCurve:   accounts[0],
		Pool:           accounts[4],
		PoolDex:        constants.DEX_PROGRAMS.METEORA_DAMM.Name,
	}
}

// decodeDBCMigrateDammV2Event decodes a migrate to DAMM V2 event
func (p *MeteoraDBCEventParser) decodeDBCMigrateDammV2Event(instruction interface{}) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 15 {
		return nil
	}

	return &types.MemeEvent{
		Type:           types.TradeTypeMigrate,
		BaseMint:       accounts[13],
		QuoteMint:      accounts[14],
		PlatformConfig: accounts[2],
		BondingCurve:   accounts[0],
		Pool:           accounts[4],
		PoolDex:        constants.DEX_PROGRAMS.METEORA_DAMM_V2.Name,
	}
}

// getTransfersForInstruction gets transfers for a specific instruction
func (p *MeteoraDBCEventParser) getTransfersForInstruction(programId string, outerIndex int, innerIndex int) []types.TransferData {
	key := fmt.Sprintf("%s-%d-%d", programId, outerIndex, innerIndex)
	if transfers, ok := p.transferActions[key]; ok {
		return transfers
	}
	return nil
}

// getAccountTradeType determines trade type based on account relationships
func getAccountTradeType(signer, baseMint, inputAccount, outputAccount string) types.TradeType {
	// Simple heuristic: if input account is derived from base mint, it's a sell
	// This is a simplified version - full implementation would check PDA derivation
	// For now, we'll use a simple comparison
	if len(inputAccount) > 0 && len(baseMint) > 0 {
		// Check first few characters as a simple heuristic
		if inputAccount[:8] == baseMint[:8] {
			return types.TradeTypeSell
		}
	}
	return types.TradeTypeBuy
}

// ProcessEvents implements the EventParser interface
func (p *MeteoraDBCEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForProgramMeteoraDBC(p.adapter, constants.DEX_PROGRAMS.METEORA_DBC.ID)
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}

// getAllInstructionsForProgramMeteoraDBC gets all instructions for Meteora DBC program
func getAllInstructionsForProgramMeteoraDBC(adapter *adapter.TransactionAdapter, programId string) []types.ClassifiedInstruction {
	var instructions []types.ClassifiedInstruction

	// Process outer instructions
	for i, ix := range adapter.Instructions() {
		ixProgramId := adapter.GetInstructionProgramId(ix)
		if ixProgramId == programId {
			instructions = append(instructions, types.ClassifiedInstruction{
				ProgramId:   ixProgramId,
				Instruction: ix,
				OuterIndex:  i,
				InnerIndex:  -1,
			})
		}
	}

	// Process inner instructions
	for _, innerSet := range adapter.InnerInstructions() {
		for j, innerIx := range innerSet.Instructions {
			ixProgramId := adapter.GetInstructionProgramId(innerIx)
			if ixProgramId == programId {
				instructions = append(instructions, types.ClassifiedInstruction{
					ProgramId:   ixProgramId,
					Instruction: innerIx,
					OuterIndex:  innerSet.Index,
					InnerIndex:  j,
				})
			}
		}
	}

	return instructions
}
