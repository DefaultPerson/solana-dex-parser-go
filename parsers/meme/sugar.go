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

// SugarParser parses Sugar transactions
type SugarParser struct {
	*parsers.BaseParser
	eventParser *SugarEventParser
}

// NewSugarParser creates a new Sugar parser
func NewSugarParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *SugarParser {
	return &SugarParser{
		BaseParser:  parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
		eventParser: NewSugarEventParser(adapter, transferActions),
	}
}

// ProcessTrades parses Sugar trades
func (p *SugarParser) ProcessTrades() []types.TradeInfo {
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

func (p *SugarParser) createTradeInfo(event *types.MemeEvent) *types.TradeInfo {
	if event.InputToken == nil || event.OutputToken == nil {
		return nil
	}

	var pool []string
	if event.BondingCurve != "" {
		pool = []string{event.BondingCurve}
	}

	amm := p.DexInfo.AMM
	if amm == "" {
		amm = constants.DEX_PROGRAMS.SUGAR.Name
	}

	trade := &types.TradeInfo{
		Type:        event.Type,
		Pool:        pool,
		InputToken:  *event.InputToken,
		OutputToken: *event.OutputToken,
		User:        event.User,
		ProgramId:   constants.DEX_PROGRAMS.SUGAR.ID,
		AMM:         amm,
		Route:       p.DexInfo.Route,
		Slot:        p.Adapter.Slot(),
		Timestamp:   event.Timestamp,
		Signature:   p.Adapter.Signature(),
		Idx:         event.Idx,
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// SugarEventParser parses Sugar events
type SugarEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
	utils           *utils.TransactionUtils
}

// NewSugarEventParser creates a new event parser
func NewSugarEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *SugarEventParser {
	return &SugarEventParser{
		adapter:         adapter,
		transferActions: transferActions,
		utils:           utils.NewTransactionUtils(adapter),
	}
}

// ParseInstructions parses classified instructions into meme events
func (p *SugarEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	for _, ci := range instructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.SUGAR.ID {
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

		// Buy discriminators
		if bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.BUY_EXACT_IN) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.BUY_EXACT_OUT) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.BUY_MAX_OUT) {
			event = p.decodeBuyEvent(data[8:], ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx)
		}

		// Sell discriminators
		if bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.SELL_EXACT_IN) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.SELL_EXACT_OUT) {
			event = p.decodeSellEvent(data[8:], ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx)
		}

		// Create discriminator
		if bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.CREATE) {
			event = p.decodeCreateEvent(data[8:], ci.Instruction)
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

func (p *SugarEventParser) decodeBuyEvent(data []byte, instruction interface{}, programId string, outerIndex int, innerIndex int) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 8 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	inputAmount := reader.ReadU64AsBigInt()
	outputAmount := reader.ReadU64AsBigInt()

	if reader.HasError() {
		return nil
	}

	bondingCurve := accounts[1]
	userAccount := accounts[0]
	inputMint := accounts[7]  // quoteMint (SOL)
	outputMint := accounts[6] // baseMint

	inputUIAmount := types.ConvertToUIAmountUint64(inputAmount.Uint64(), 9)
	outputUIAmount := types.ConvertToUIAmountUint64(outputAmount.Uint64(), 6)

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.SUGAR.Name,
		Type:         types.TradeTypeBuy,
		BaseMint:     outputMint,
		QuoteMint:    inputMint,
		BondingCurve: bondingCurve,
		Pool:         bondingCurve,
		User:         userAccount,
		InputToken: &types.TokenInfo{
			Mint:      inputMint,
			AmountRaw: inputAmount.String(),
			Amount:    inputUIAmount,
			Decimals:  9,
		},
		OutputToken: &types.TokenInfo{
			Mint:      outputMint,
			AmountRaw: outputAmount.String(),
			Amount:    outputUIAmount,
			Decimals:  6,
		},
	}
}

func (p *SugarEventParser) decodeSellEvent(data []byte, instruction interface{}, programId string, outerIndex int, innerIndex int) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 8 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	inputAmount := reader.ReadU64AsBigInt()
	outputAmount := reader.ReadU64AsBigInt()

	if reader.HasError() {
		return nil
	}

	bondingCurve := accounts[1]
	userAccount := accounts[0]
	inputMint := accounts[6]  // baseMint
	outputMint := accounts[7] // quoteMint (SOL)

	inputUIAmount := types.ConvertToUIAmountUint64(inputAmount.Uint64(), 6)
	outputUIAmount := types.ConvertToUIAmountUint64(outputAmount.Uint64(), 9)

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.SUGAR.Name,
		Type:         types.TradeTypeSell,
		BaseMint:     inputMint,
		QuoteMint:    outputMint,
		BondingCurve: bondingCurve,
		Pool:         bondingCurve,
		User:         userAccount,
		InputToken: &types.TokenInfo{
			Mint:      inputMint,
			AmountRaw: inputAmount.String(),
			Amount:    inputUIAmount,
			Decimals:  6,
		},
		OutputToken: &types.TokenInfo{
			Mint:      outputMint,
			AmountRaw: outputAmount.String(),
			Amount:    outputUIAmount,
			Decimals:  9,
		},
	}
}

func (p *SugarEventParser) decodeCreateEvent(data []byte, instruction interface{}) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 8 {
		return nil
	}

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

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.SUGAR.Name,
		Type:         types.TradeTypeCreate,
		Timestamp:    p.adapter.BlockTime(),
		User:         accounts[0],
		BaseMint:     accounts[6],
		QuoteMint:    constants.TOKENS.SOL,
		Name:         name,
		Symbol:       symbol,
		URI:          uri,
		BondingCurve: accounts[1],
		Creator:      accounts[0],
	}
}

// ProcessEvents implements the EventParser interface
func (p *SugarEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForProgramSugar(p.adapter, constants.DEX_PROGRAMS.SUGAR.ID)
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}

// getAllInstructionsForProgramSugar gets all instructions for Sugar program
func getAllInstructionsForProgramSugar(adapter *adapter.TransactionAdapter, programId string) []types.ClassifiedInstruction {
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
