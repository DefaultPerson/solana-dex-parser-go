package meme

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/parsers"
	"github.com/solana-dex-parser-go/types"
	"github.com/solana-dex-parser-go/utils"
)

// HeavenParser parses Heaven transactions
type HeavenParser struct {
	*parsers.BaseParser
	eventParser *HeavenEventParser
}

// NewHeavenParser creates a new Heaven parser
func NewHeavenParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *HeavenParser {
	return &HeavenParser{
		BaseParser:  parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
		eventParser: NewHeavenEventParser(adapter, transferActions),
	}
}

// ProcessTrades parses Heaven trades
func (p *HeavenParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	events := p.eventParser.ParseInstructions(p.ClassifiedInstructions)

	for _, event := range events {
		if event.Type == types.TradeTypeBuy || event.Type == types.TradeTypeSell || event.Type == "SWAP" {
			trade := p.createTradeInfo(event)
			if trade != nil {
				trades = append(trades, *trade)
			}
		}
	}

	return trades
}

func (p *HeavenParser) createTradeInfo(event *types.MemeEvent) *types.TradeInfo {
	if event.InputToken == nil || event.OutputToken == nil {
		return nil
	}

	var pool []string
	if event.BondingCurve != "" {
		pool = []string{event.BondingCurve}
	}

	amm := p.DexInfo.AMM
	if amm == "" {
		amm = constants.DEX_PROGRAMS.HEAVEN.Name
	}

	trade := &types.TradeInfo{
		Type:        event.Type,
		Pool:        pool,
		InputToken:  *event.InputToken,
		OutputToken: *event.OutputToken,
		User:        event.User,
		ProgramId:   constants.DEX_PROGRAMS.HEAVEN.ID,
		AMM:         amm,
		Route:       p.DexInfo.Route,
		Slot:        p.Adapter.Slot(),
		Timestamp:   event.Timestamp,
		Signature:   p.Adapter.Signature(),
		Idx:         event.Idx,
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// HeavenEventParser parses Heaven events
type HeavenEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
	utils           *utils.TransactionUtils
}

// NewHeavenEventParser creates a new event parser
func NewHeavenEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *HeavenEventParser {
	return &HeavenEventParser{
		adapter:         adapter,
		transferActions: transferActions,
		utils:           utils.NewTransactionUtils(adapter),
	}
}

// ParseInstructions parses classified instructions into meme events
func (p *HeavenEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	for _, ci := range instructions {
		// Check Heaven program
		if ci.ProgramId != constants.DEX_PROGRAMS.HEAVEN.ID {
			continue
		}

		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		disc := data[:8]
		innerIdx := ci.InnerIndex
		if innerIdx < 0 {
			innerIdx = 0
		}

		var event *types.MemeEvent

		if bytes.Equal(disc, constants.DISCRIMINATORS.HEAVEN.BUY) {
			event = p.decodeBuyEvent(data[8:], ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx)
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.HEAVEN.SELL) {
			event = p.decodeSellEvent(data[8:], ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx)
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.HEAVEN.CREATE_POOL) {
			event = p.decodeInitialBuyEvent(ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx)
		}

		if event != nil {
			event.Signature = p.adapter.Signature()
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Idx = fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx)
			events = append(events, event)
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Idx < events[j].Idx
	})

	return events
}

func (p *HeavenEventParser) decodeInitialBuyEvent(instruction interface{}, programId string, outerIndex int, innerIndex int) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 12 {
		return nil
	}

	bondingCurve := accounts[10]
	userAccount := accounts[4]
	inputMint := accounts[6]  // quoteMint
	outputMint := accounts[5] // baseMint

	event := &types.MemeEvent{
		Protocol:       constants.DEX_PROGRAMS.HEAVEN.Name,
		Type:           types.TradeTypeBuy,
		BaseMint:       outputMint,
		QuoteMint:      inputMint,
		BondingCurve:   bondingCurve,
		Pool:           bondingCurve,
		User:           userAccount,
		PlatformConfig: accounts[11],
	}

	// Get transfers and fill token info
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

func (p *HeavenEventParser) decodeBuyEvent(data []byte, instruction interface{}, programId string, outerIndex int, innerIndex int) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
	solAmount := reader.ReadU64AsBigInt()
	tokenAmount := reader.ReadU64AsBigInt()

	if reader.HasError() {
		return nil
	}

	bondingCurve := accounts[6]
	userAccount := accounts[3]
	outputMint := accounts[4] // baseMint
	inputMint := accounts[5]  // quoteMint

	inputUIAmount := types.ConvertToUIAmountUint64(solAmount.Uint64(), 9)
	outputUIAmount := types.ConvertToUIAmountUint64(tokenAmount.Uint64(), 6)

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.HEAVEN.Name,
		Type:         types.TradeTypeBuy,
		BaseMint:     outputMint,
		QuoteMint:    inputMint,
		BondingCurve: bondingCurve,
		Pool:         bondingCurve,
		User:         userAccount,
		InputToken: &types.TokenInfo{
			Mint:      inputMint,
			AmountRaw: solAmount.String(),
			Amount:    inputUIAmount,
			Decimals:  9,
		},
		OutputToken: &types.TokenInfo{
			Mint:      outputMint,
			AmountRaw: tokenAmount.String(),
			Amount:    outputUIAmount,
			Decimals:  6,
		},
	}
}

func (p *HeavenEventParser) decodeSellEvent(data []byte, instruction interface{}, programId string, outerIndex int, innerIndex int) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
	tokenAmount := reader.ReadU64AsBigInt()
	solAmount := reader.ReadU64AsBigInt()

	if reader.HasError() {
		return nil
	}

	bondingCurve := accounts[6]
	userAccount := accounts[3]
	inputMint := accounts[4]  // baseMint
	outputMint := accounts[5] // quoteMint

	inputUIAmount := types.ConvertToUIAmountUint64(tokenAmount.Uint64(), 6)
	outputUIAmount := types.ConvertToUIAmountUint64(solAmount.Uint64(), 9)

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.HEAVEN.Name,
		Type:         types.TradeTypeSell,
		BaseMint:     inputMint,
		QuoteMint:    outputMint,
		BondingCurve: bondingCurve,
		Pool:         bondingCurve,
		User:         userAccount,
		InputToken: &types.TokenInfo{
			Mint:      inputMint,
			AmountRaw: tokenAmount.String(),
			Amount:    inputUIAmount,
			Decimals:  6,
		},
		OutputToken: &types.TokenInfo{
			Mint:      outputMint,
			AmountRaw: solAmount.String(),
			Amount:    outputUIAmount,
			Decimals:  9,
		},
	}
}

func (p *HeavenEventParser) getTransfersForInstruction(programId string, outerIndex int, innerIndex int) []types.TransferData {
	key := fmt.Sprintf("%s:%d-%d", programId, outerIndex, innerIndex)
	if transfers, ok := p.transferActions[key]; ok {
		return transfers
	}
	return nil
}

// ProcessEvents implements the EventParser interface
func (p *HeavenEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForProgramHeaven(p.adapter, constants.DEX_PROGRAMS.HEAVEN.ID)
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}

// getAllInstructionsForProgramHeaven gets all instructions for Heaven program
func getAllInstructionsForProgramHeaven(adapter *adapter.TransactionAdapter, programId string) []types.ClassifiedInstruction {
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
