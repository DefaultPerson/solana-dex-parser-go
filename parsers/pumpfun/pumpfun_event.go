package pumpfun

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// PumpfunEventParser parses Pumpfun events
type PumpfunEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
}

// NewPumpfunEventParser creates a new event parser
func NewPumpfunEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *PumpfunEventParser {
	return &PumpfunEventParser{
		adapter:         adapter,
		transferActions: transferActions,
	}
}

// ParseInstructions parses classified instructions into meme events
func (p *PumpfunEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	for _, ci := range instructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.PUMP_FUN.ID {
			continue
		}

		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 16 {
			continue
		}

		disc := data[:16]
		innerIdx := ci.InnerIndex
		if innerIdx < 0 {
			innerIdx = 0
		}

		var event *types.MemeEvent

		// Check event discriminators
		if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPFUN.TRADE_EVENT) {
			event = p.decodeTradeEvent(data[16:])
			if event != nil {
				// Get bonding curve from previous instruction
				prevInst := getPrevInstructionByIndex(instructions, ci.OuterIndex, innerIdx)
				if prevInst != nil {
					accounts := p.adapter.GetInstructionAccounts(prevInst.Instruction)
					if len(accounts) > 3 {
						event.BondingCurve = accounts[3]
					}
				}
			}
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPFUN.CREATE_EVENT) {
			event = p.decodeCreateEvent(data[16:])
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPFUN.COMPLETE_EVENT) {
			event = p.decodeCompleteEvent(data[16:])
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPFUN.MIGRATE_EVENT) {
			event = p.decodeMigrateEvent(data[16:])
		}

		if event != nil {
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
func (p *PumpfunEventParser) decodeTradeEvent(data []byte) *types.MemeEvent {
	if len(data) < 90 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	mint, _ := reader.ReadPubkey()
	quoteMint := constants.TOKENS.SOL
	solAmount := reader.ReadU64AsBigInt()
	tokenAmount := reader.ReadU64AsBigInt()
	isBuyByte, _ := reader.ReadU8()
	isBuy := isBuyByte == 1
	user, _ := reader.ReadPubkey()
	timestamp, _ := reader.ReadI64()
	// virtualSolReserves, virtualTokenReserves are also read but not used in output

	if reader.HasError() {
		return nil
	}

	// Read optional extended fields
	var fee, creatorFee uint64
	if reader.Remaining() >= 52 {
		reader.Skip(16) // realSolReserves, realTokenReserves
		reader.Skip(32) // feeRecipient
		reader.Skip(2)  // feeBasisPoints
		f, _ := reader.ReadU64()
		fee = f
		reader.Skip(32) // creator
		reader.Skip(2)  // creatorFeeBasisPoints
		cf, _ := reader.ReadU64()
		creatorFee = cf
	}

	var inputMint, outputMint string
	var inputAmount, outputAmount uint64
	var inputDecimals, outputDecimals uint8

	if isBuy {
		inputMint = quoteMint
		inputAmount = solAmount.Uint64()
		inputDecimals = 9
		outputMint = mint
		outputAmount = tokenAmount.Uint64()
		outputDecimals = 6
	} else {
		inputMint = mint
		inputAmount = tokenAmount.Uint64()
		inputDecimals = 6
		outputMint = quoteMint
		outputAmount = solAmount.Uint64()
		outputDecimals = 9
	}

	inputUIAmount := types.ConvertToUIAmountUint64(inputAmount, inputDecimals)
	outputUIAmount := types.ConvertToUIAmountUint64(outputAmount, outputDecimals)

	eventType := types.TradeTypeSell
	if isBuy {
		eventType = types.TradeTypeBuy
	}

	feeFloat := types.ConvertToUIAmountUint64(fee, 9)
	creatorFeeFloat := types.ConvertToUIAmountUint64(creatorFee, 9)

	return &types.MemeEvent{
		Protocol:  constants.DEX_PROGRAMS.PUMP_FUN.Name,
		Type:      eventType,
		BaseMint:  mint,
		QuoteMint: quoteMint,
		User:      user,
		Timestamp: timestamp,
		InputToken: &types.TokenInfo{
			Mint:      inputMint,
			AmountRaw: fmt.Sprintf("%d", inputAmount),
			Amount:    inputUIAmount,
			Decimals:  inputDecimals,
		},
		OutputToken: &types.TokenInfo{
			Mint:      outputMint,
			AmountRaw: fmt.Sprintf("%d", outputAmount),
			Amount:    outputUIAmount,
			Decimals:  outputDecimals,
		},
		ProtocolFee: &feeFloat,
		CreatorFee:  &creatorFeeFloat,
	}
}

// decodeCreateEvent decodes a create event
func (p *PumpfunEventParser) decodeCreateEvent(data []byte) *types.MemeEvent {
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
	mint, _ := reader.ReadPubkey()
	bondingCurve, _ := reader.ReadPubkey()
	user, _ := reader.ReadPubkey()

	if reader.HasError() {
		return nil
	}

	var creator string
	var timestamp int64
	if reader.Remaining() >= 40 {
		creator, _ = reader.ReadPubkey()
		timestamp, _ = reader.ReadI64()
	}

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.PUMP_FUN.Name,
		Type:         types.TradeTypeCreate,
		Timestamp:    timestamp,
		User:         user,
		BaseMint:     mint,
		QuoteMint:    constants.TOKENS.SOL,
		Name:         name,
		Symbol:       symbol,
		URI:          uri,
		BondingCurve: bondingCurve,
		Creator:      creator,
	}
}

// decodeCompleteEvent decodes a complete event
func (p *PumpfunEventParser) decodeCompleteEvent(data []byte) *types.MemeEvent {
	if len(data) < 104 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	user, _ := reader.ReadPubkey()
	mint, _ := reader.ReadPubkey()
	bondingCurve, _ := reader.ReadPubkey()
	timestamp, _ := reader.ReadI64()

	if reader.HasError() {
		return nil
	}

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.PUMP_FUN.Name,
		Type:         types.TradeTypeComplete,
		Timestamp:    timestamp,
		User:         user,
		BaseMint:     mint,
		QuoteMint:    constants.TOKENS.SOL,
		BondingCurve: bondingCurve,
	}
}

// decodeMigrateEvent decodes a migrate event
func (p *PumpfunEventParser) decodeMigrateEvent(data []byte) *types.MemeEvent {
	if len(data) < 168 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	user, _ := reader.ReadPubkey()
	mint, _ := reader.ReadPubkey()
	reader.Skip(24) // mintAmount, solAmount, poolMigrateFee
	bondingCurve, _ := reader.ReadPubkey()
	timestamp, _ := reader.ReadI64()
	pool, _ := reader.ReadPubkey()

	if reader.HasError() {
		return nil
	}

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.PUMP_FUN.Name,
		Type:         types.TradeTypeMigrate,
		Timestamp:    timestamp,
		User:         user,
		BaseMint:     mint,
		QuoteMint:    constants.TOKENS.SOL,
		BondingCurve: bondingCurve,
		Pool:         pool,
		PoolDex:      constants.DEX_PROGRAMS.PUMP_SWAP.Name,
	}
}

// ProcessEvents implements EventParser interface for meme event parsers
func (p *PumpfunEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForProgram(p.adapter, constants.DEX_PROGRAMS.PUMP_FUN.ID)
	events := p.ParseInstructions(instructions)

	// Convert to non-pointer slice
	result := make([]types.MemeEvent, len(events))
	for i, e := range events {
		if e != nil {
			result[i] = *e
		}
	}
	return result
}

// getAllInstructionsForProgram gets all instructions for a program ID
func getAllInstructionsForProgram(adapter *adapter.TransactionAdapter, programId string) []types.ClassifiedInstruction {
	var instructions []types.ClassifiedInstruction

	// Outer instructions
	for outerIdx, ix := range adapter.Instructions() {
		programIdFromIx := adapter.GetInstructionProgramId(ix)
		if programIdFromIx == programId {
			instructions = append(instructions, types.ClassifiedInstruction{
				ProgramId:   programIdFromIx,
				Instruction: ix,
				OuterIndex:  outerIdx,
				InnerIndex:  -1,
			})
		}
	}

	// Inner instructions
	for _, innerSet := range adapter.InnerInstructions() {
		for innerIdx, ix := range innerSet.Instructions {
			programIdFromIx := adapter.GetInstructionProgramId(ix)
			if programIdFromIx == programId {
				instructions = append(instructions, types.ClassifiedInstruction{
					ProgramId:   programIdFromIx,
					Instruction: ix,
					OuterIndex:  innerSet.Index,
					InnerIndex:  innerIdx,
				})
			}
		}
	}

	return instructions
}

// getPrevInstructionByIndex finds the previous instruction
func getPrevInstructionByIndex(instructions []types.ClassifiedInstruction, outerIndex int, innerIndex int) *types.ClassifiedInstruction {
	for i := len(instructions) - 1; i >= 0; i-- {
		ci := instructions[i]
		if ci.OuterIndex == outerIndex && ci.InnerIndex < innerIndex {
			return &ci
		}
		if ci.OuterIndex < outerIndex {
			return &ci
		}
	}
	return nil
}
