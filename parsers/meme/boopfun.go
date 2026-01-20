package meme

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// BoopfunParser parses Boopfun transactions
type BoopfunParser struct {
	*parsers.BaseParser
	eventParser *BoopfunEventParser
}

// NewBoopfunParser creates a new Boopfun parser
func NewBoopfunParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *BoopfunParser {
	return &BoopfunParser{
		BaseParser:  parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
		eventParser: NewBoopfunEventParser(adapter, transferActions),
	}
}

// ProcessTrades parses Boopfun trades
func (p *BoopfunParser) ProcessTrades() []types.TradeInfo {
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

// createTradeInfo creates a TradeInfo from a MemeEvent
func (p *BoopfunParser) createTradeInfo(event *types.MemeEvent) *types.TradeInfo {
	if event.InputToken == nil || event.OutputToken == nil {
		return nil
	}

	var pool []string
	if event.BondingCurve != "" {
		pool = []string{event.BondingCurve}
	}

	amm := p.DexInfo.AMM
	if amm == "" {
		amm = constants.DEX_PROGRAMS.BOOP_FUN.Name
	}

	trade := &types.TradeInfo{
		Type:        event.Type,
		Pool:        pool,
		InputToken:  *event.InputToken,
		OutputToken: *event.OutputToken,
		User:        event.User,
		ProgramId:   constants.DEX_PROGRAMS.BOOP_FUN.ID,
		AMM:         amm,
		Route:       p.DexInfo.Route,
		Slot:        p.Adapter.Slot(),
		Timestamp:   event.Timestamp,
		Signature:   p.Adapter.Signature(),
		Idx:         event.Idx,
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// BoopfunEventParser parses Boopfun events
type BoopfunEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
}

// NewBoopfunEventParser creates a new event parser
func NewBoopfunEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *BoopfunEventParser {
	return &BoopfunEventParser{
		adapter:         adapter,
		transferActions: transferActions,
	}
}

// ParseInstructions parses classified instructions into meme events
func (p *BoopfunEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	for _, ci := range instructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.BOOP_FUN.ID {
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

		if bytes.Equal(disc, constants.DISCRIMINATORS.BOOPFUN.BUY) {
			event = p.decodeBuyEvent(data[8:], ci.Instruction, ci.OuterIndex, innerIdx)
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.BOOPFUN.SELL) {
			event = p.decodeSellEvent(data[8:], ci.Instruction, ci.OuterIndex, innerIdx)
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.BOOPFUN.CREATE) {
			event = p.decodeCreateEvent(data[8:], ci.Instruction)
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.BOOPFUN.COMPLETE) {
			event = p.decodeCompleteEvent(ci.Instruction, ci.OuterIndex, innerIdx)
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

func (p *BoopfunEventParser) decodeBuyEvent(data []byte, instruction interface{}, outerIndex int, innerIndex int) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	solAmount := reader.ReadU64AsBigInt()

	if reader.HasError() {
		return nil
	}

	// Get transfers to find token amount
	programId := p.adapter.GetInstructionProgramId(instruction)
	transfers := p.getTransfersForInstruction(programId, outerIndex, innerIndex)
	var tokenAmount uint64
	for _, t := range transfers {
		if t.Info.Mint == accounts[0] {
			if t.Info.TokenAmount.Amount != "" {
				if amt, err := parseUint64(t.Info.TokenAmount.Amount); err == nil {
					tokenAmount = amt
				}
			}
			break
		}
	}

	mint := accounts[0]
	quoteMint := constants.TOKENS.SOL
	user := accounts[6]
	bondingCurve := accounts[1]

	inputUIAmount := types.ConvertToUIAmountUint64(solAmount.Uint64(), 9)
	outputUIAmount := types.ConvertToUIAmountUint64(tokenAmount, 6)

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.BOOP_FUN.Name,
		Type:         types.TradeTypeBuy,
		BondingCurve: bondingCurve,
		BaseMint:     mint,
		QuoteMint:    quoteMint,
		User:         user,
		InputToken: &types.TokenInfo{
			Mint:      quoteMint,
			AmountRaw: solAmount.String(),
			Amount:    inputUIAmount,
			Decimals:  9,
		},
		OutputToken: &types.TokenInfo{
			Mint:      mint,
			AmountRaw: fmt.Sprintf("%d", tokenAmount),
			Amount:    outputUIAmount,
			Decimals:  6,
		},
	}
}

func (p *BoopfunEventParser) decodeSellEvent(data []byte, instruction interface{}, outerIndex int, innerIndex int) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	tokenAmount := reader.ReadU64AsBigInt()

	if reader.HasError() {
		return nil
	}

	// Get transfers to find SOL amount
	programId := p.adapter.GetInstructionProgramId(instruction)
	transfers := p.getTransfersForInstruction(programId, outerIndex, innerIndex)
	var solAmount uint64
	for _, t := range transfers {
		if t.Info.Mint == constants.TOKENS.SOL {
			if t.Info.TokenAmount.Amount != "" {
				if amt, err := parseUint64(t.Info.TokenAmount.Amount); err == nil {
					solAmount = amt
				}
			}
			break
		}
	}

	mint := accounts[0]
	quoteMint := constants.TOKENS.SOL
	user := accounts[6]
	bondingCurve := accounts[1]

	inputUIAmount := types.ConvertToUIAmountUint64(tokenAmount.Uint64(), 6)
	outputUIAmount := types.ConvertToUIAmountUint64(solAmount, 9)

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.BOOP_FUN.Name,
		Type:         types.TradeTypeSell,
		BondingCurve: bondingCurve,
		BaseMint:     mint,
		QuoteMint:    quoteMint,
		User:         user,
		InputToken: &types.TokenInfo{
			Mint:      mint,
			AmountRaw: tokenAmount.String(),
			Amount:    inputUIAmount,
			Decimals:  6,
		},
		OutputToken: &types.TokenInfo{
			Mint:      quoteMint,
			AmountRaw: fmt.Sprintf("%d", solAmount),
			Amount:    outputUIAmount,
			Decimals:  9,
		},
	}
}

func (p *BoopfunEventParser) decodeCreateEvent(data []byte, instruction interface{}) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 4 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	reader.Skip(8) // skip first u64

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

	return &types.MemeEvent{
		Protocol:  constants.DEX_PROGRAMS.BOOP_FUN.Name,
		Type:      types.TradeTypeCreate,
		Timestamp: p.adapter.BlockTime(),
		User:      accounts[3],
		BaseMint:  accounts[2],
		QuoteMint: constants.TOKENS.SOL,
		Name:      name,
		Symbol:    symbol,
		URI:       uri,
		Creator:   accounts[3],
	}
}

func (p *BoopfunEventParser) decodeCompleteEvent(instruction interface{}, outerIndex int, innerIndex int) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 11 {
		return nil
	}

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.BOOP_FUN.Name,
		Type:         types.TradeTypeComplete,
		Timestamp:    p.adapter.BlockTime(),
		User:         accounts[10],
		BaseMint:     accounts[0],
		QuoteMint:    constants.TOKENS.SOL,
		BondingCurve: accounts[7],
	}
}

func (p *BoopfunEventParser) getTransfersForInstruction(programId string, outerIndex int, innerIndex int) []types.TransferData {
	key := fmt.Sprintf("%s:%d-%d", programId, outerIndex, innerIndex)
	if transfers, ok := p.transferActions[key]; ok {
		var filtered []types.TransferData
		for _, t := range transfers {
			if t.Type == "transfer" || t.Type == "transferChecked" {
				filtered = append(filtered, t)
			}
		}
		return filtered
	}
	return nil
}

func parseUint64(s string) (uint64, error) {
	var v uint64
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

// ProcessEvents implements the EventParser interface
func (p *BoopfunEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForProgramBoopfun(p.adapter, constants.DEX_PROGRAMS.BOOP_FUN.ID)
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}

// getAllInstructionsForProgramBoopfun gets all instructions for Boopfun program
func getAllInstructionsForProgramBoopfun(adapter *adapter.TransactionAdapter, programId string) []types.ClassifiedInstruction {
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
