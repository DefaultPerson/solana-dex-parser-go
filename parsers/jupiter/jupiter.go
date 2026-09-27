package jupiter

import (
	"bytes"
	"math/big"
	"strconv"
	"strings"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// JupiterParser parses Jupiter V6 swap transactions
type JupiterParser struct {
	*parsers.BaseParser
}

// NewJupiterParser creates a new Jupiter parser
func NewJupiterParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *JupiterParser {
	return &JupiterParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// ProcessTrades parses Jupiter V6 swap trades, one trade per hop: the legacy
// route instructions emit one SwapEvent per hop, the *_v2 route instructions
// one SwapsEvent listing all hops. Hop amounts are the event amounts, i.e. what
// the AMM received and paid. A FeeEvent (platform fee) of a route becomes the
// Fee of that route's first hop when it precedes the hops (fee taken from the
// input; the hop input then includes it) and of its last hop otherwise (fee
// taken from the output; the hop output then excludes it).
func (p *JupiterParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo
	var feeEvents []jupiterFeeEventAt

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.JUPITER.ID {
			continue
		}
		data := p.Adapter.GetInstructionData(ci.Instruction)
		if len(data) < 16 {
			continue
		}
		idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)

		switch {
		case bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER.ROUTE_EVENT):
			layout, err := ParseJupiterSwapLayout(data[16:])
			if err != nil {
				continue
			}
			event := layout.ToSwapEvent()
			event.Idx = idx
			trades = p.appendHopTrade(trades, event)
		case bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER.SWAPS_EVENT):
			events, err := ParseJupiterSwapsEvent(data[16:])
			if err != nil {
				continue
			}
			hopIdx := p.hopIndexes(ci, events)
			for i, event := range events {
				event.Idx = hopIdx[i]
				trades = p.appendHopTrade(trades, event)
			}
		case bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER.FEE_EVENT):
			event, err := ParseJupiterFeeEvent(data[16:])
			if err != nil {
				continue
			}
			feeEvents = append(feeEvents, jupiterFeeEventAt{outer: ci.OuterIndex, idx: idx, event: event})
		}
	}

	trades = utils.SortTradesByIdx(trades)
	p.attachFeeEvents(trades, feeEvents)
	return trades
}

// jupiterFeeEventAt is a decoded FeeEvent with its position in the transaction
type jupiterFeeEventAt struct {
	outer int
	idx   string
	event *JupiterFeeEvent
}

// appendHopTrade converts one hop event into a trade and appends it
func (p *JupiterParser) appendHopTrade(trades []types.TradeInfo, event *JupiterSwapEvent) []types.TradeInfo {
	event.InputMintDecimals = p.Adapter.GetTokenDecimals(event.InputMint)
	event.OutputMintDecimals = p.Adapter.GetTokenDecimals(event.OutputMint)
	if trade := p.processSwapData([]*JupiterSwapEvent{event}); trade != nil {
		trades = append(trades, *trade)
	}
	return trades
}

// hopIndexes returns an idx for every hop of a SwapsEvent: the idx of the
// instruction of the hop's AMM program that executed it. The hops are matched
// in order against the instructions between the route instruction that
// emitted the event and the event itself. A hop whose AMM instruction is not
// found gets the idx of the event.
func (p *JupiterParser) hopIndexes(ci types.ClassifiedInstruction, events []*JupiterSwapEvent) []string {
	eventIdx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	result := make([]string, len(events))
	for i := range result {
		result[i] = eventIdx
	}
	if ci.InnerIndex < 0 {
		return result
	}

	var inner []interface{}
	for _, set := range p.Adapter.InnerInstructions() {
		if set.Index == ci.OuterIndex {
			inner = set.Instructions
			break
		}
	}
	end := ci.InnerIndex
	if end > len(inner) {
		end = len(inner)
	}

	// The hops follow the Jupiter route instruction that emitted the event
	// (outer when no Jupiter instruction precedes the event in this set)
	cursor := 0
	for j := end - 1; j >= 0; j-- {
		if p.Adapter.GetInstructionProgramId(inner[j]) == constants.DEX_PROGRAMS.JUPITER.ID && !isAnchorEvent(p.Adapter.GetInstructionData(inner[j])) {
			cursor = j + 1
			break
		}
	}

	for i, event := range events {
		for j := cursor; j < end; j++ {
			if p.Adapter.GetInstructionProgramId(inner[j]) == event.AMM {
				result[i] = utils.FormatIdx(ci.OuterIndex, j)
				cursor = j + 1
				break
			}
		}
	}
	return result
}

// attachFeeEvents sets each FeeEvent as the fee of a hop of the same outer
// instruction: the first hop when the event precedes the hops, else the last
func (p *JupiterParser) attachFeeEvents(trades []types.TradeInfo, feeEvents []jupiterFeeEventAt) {
	for _, fe := range feeEvents {
		first, last := -1, -1
		for i := range trades {
			if outerIndexOf(trades[i].Idx) != fe.outer {
				continue
			}
			if first < 0 {
				first = i
			}
			last = i
		}
		if first < 0 {
			continue
		}
		target := last
		if utils.CompareIdx(fe.idx, trades[first].Idx) < 0 {
			target = first
		}

		decimals := p.Adapter.GetTokenDecimals(fe.event.Mint)
		fee := types.FeeInfo{
			Mint:      fe.event.Mint,
			Amount:    types.ConvertToUIAmount(fe.event.Amount, decimals),
			AmountRaw: fe.event.Amount.String(),
			Decimals:  decimals,
			Dex:       constants.DEX_PROGRAMS.JUPITER.Name,
			Type:      "platform",
			Recipient: fe.event.Account,
		}
		trade := &trades[target]
		if trade.Fee == nil {
			trade.Fee = &fee
		} else {
			trade.Fees = append(trade.Fees, fee)
		}

		// Amounts are what the user sent and received: a fee taken from the
		// input is paid on top of the first hop's input (gross input), a fee
		// taken from the output is deducted from the last hop's output (net
		// output). The transfer details stay those of the hop's own transfers.
		if target == first && trade.InputToken.Mint == fe.event.Mint {
			adjustTokenAmount(&trade.InputToken, fe.event.Amount)
		} else if target == last && trade.OutputToken.Mint == fe.event.Mint {
			adjustTokenAmount(&trade.OutputToken, new(big.Int).Neg(fe.event.Amount))
		}
	}
}

// adjustTokenAmount adds delta to the raw and UI amount of token
func adjustTokenAmount(token *types.TokenInfo, delta *big.Int) {
	amount, ok := new(big.Int).SetString(token.AmountRaw, 10)
	if !ok {
		return
	}
	amount.Add(amount, delta)
	if amount.Sign() < 0 {
		return
	}
	token.AmountRaw = amount.String()
	token.Amount = types.ConvertToUIAmount(amount, token.Decimals)
}

// isAnchorEvent reports whether instruction data is an Anchor self-CPI event
func isAnchorEvent(data []byte) bool {
	return len(data) >= 16 && bytes.Equal(data[:8], constants.DISCRIMINATORS.JUPITER.ROUTE_EVENT[:8])
}

// outerIndexOf returns the outer instruction index of an idx ("5" or "5-3"),
// or -1 when it cannot be parsed
func outerIndexOf(idx string) int {
	if i := strings.IndexByte(idx, '-'); i >= 0 {
		idx = idx[:i]
	}
	n, err := strconv.Atoi(idx)
	if err != nil {
		return -1
	}
	return n
}

// processSwapData processes swap events into trade info
func (p *JupiterParser) processSwapData(events []*JupiterSwapEvent) *types.TradeInfo {
	if len(events) == 0 {
		return nil
	}

	info := p.buildIntermediateInfo(events)
	return p.convertToTradeInfo(info)
}

// JupiterSwapInfo holds intermediate swap information
type JupiterSwapInfo struct {
	AMMs     []string
	AMMIds   []string // program ids of AMMs, same order
	TokenIn  map[string]*big.Int
	TokenOut map[string]*big.Int
	Decimals map[string]uint8
	Idx      string
}

// buildIntermediateInfo builds intermediate swap info from events
func (p *JupiterParser) buildIntermediateInfo(events []*JupiterSwapEvent) *JupiterSwapInfo {
	info := &JupiterSwapInfo{
		AMMs:     make([]string, 0),
		AMMIds:   make([]string, 0),
		TokenIn:  make(map[string]*big.Int),
		TokenOut: make(map[string]*big.Int),
		Decimals: make(map[string]uint8),
	}

	for _, event := range events {
		inputMint := event.InputMint
		outputMint := event.OutputMint

		// Accumulate input amounts
		if existing, ok := info.TokenIn[inputMint]; ok {
			info.TokenIn[inputMint] = new(big.Int).Add(existing, event.InputAmount)
		} else {
			info.TokenIn[inputMint] = new(big.Int).Set(event.InputAmount)
		}

		// Accumulate output amounts
		if existing, ok := info.TokenOut[outputMint]; ok {
			info.TokenOut[outputMint] = new(big.Int).Add(existing, event.OutputAmount)
		} else {
			info.TokenOut[outputMint] = new(big.Int).Set(event.OutputAmount)
		}

		info.Decimals[inputMint] = event.InputMintDecimals
		info.Decimals[outputMint] = event.OutputMintDecimals
		info.Idx = event.Idx
		info.AMMs = append(info.AMMs, constants.GetProgramName(event.AMM))
		info.AMMIds = append(info.AMMIds, event.AMM)
	}

	p.removeIntermediateTokens(info)
	return info
}

// removeIntermediateTokens removes intermediate tokens where in/out amounts match
func (p *JupiterParser) removeIntermediateTokens(info *JupiterSwapInfo) {
	for mint, inAmount := range info.TokenIn {
		if outAmount, ok := info.TokenOut[mint]; ok {
			if inAmount.Cmp(outAmount) == 0 {
				delete(info.TokenIn, mint)
				delete(info.TokenOut, mint)
			}
		}
	}
}

// convertToTradeInfo converts intermediate info to trade info
func (p *JupiterParser) convertToTradeInfo(info *JupiterSwapInfo) *types.TradeInfo {
	if len(info.TokenIn) != 1 || len(info.TokenOut) != 1 {
		return nil
	}

	var inMint, outMint string
	var inAmount, outAmount *big.Int

	for mint, amount := range info.TokenIn {
		inMint = mint
		inAmount = amount
	}
	for mint, amount := range info.TokenOut {
		outMint = mint
		outAmount = amount
	}

	inDecimals := info.Decimals[inMint]
	outDecimals := info.Decimals[outMint]

	signerIndex := 0
	if p.containsDCAProgram() {
		signerIndex = 2
	}
	signer := p.Adapter.GetAccountKey(signerIndex)

	inUIAmount := types.ConvertToUIAmount(inAmount, inDecimals)
	outUIAmount := types.ConvertToUIAmount(outAmount, outDecimals)

	// A hop through an AMM without a known name is labelled "Unknown" and
	// keeps the AMM's program id in ProgramId, so the venue is never lost
	amm, programId := p.getAMM(info)

	trade := &types.TradeInfo{
		Type: utils.GetTradeType(inMint, outMint),
		InputToken: types.TokenInfo{
			Mint:      inMint,
			Amount:    inUIAmount,
			AmountRaw: inAmount.String(),
			Decimals:  inDecimals,
		},
		OutputToken: types.TokenInfo{
			Mint:      outMint,
			Amount:    outUIAmount,
			AmountRaw: outAmount.String(),
			Decimals:  outDecimals,
		},
		User:      signer,
		ProgramId: programId,
		AMM:       amm,
		Route:     p.DexInfo.Route,
		Slot:      p.Adapter.Slot(),
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
		Idx:       info.Idx,
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// getAMM returns the AMM name of the first hop in info and the program id the
// trade reports: the Jupiter program for a named AMM, the AMM's own program id
// when the AMM is unknown (name "Unknown"). Without hops it falls back to the
// DEX info.
func (p *JupiterParser) getAMM(info *JupiterSwapInfo) (string, string) {
	if len(info.AMMs) > 0 {
		name := info.AMMs[0]
		if name != "" && name != unknownAMM {
			return name, p.DexInfo.ProgramId
		}
		if len(info.AMMIds) > 0 && info.AMMIds[0] != "" {
			return unknownAMM, info.AMMIds[0]
		}
		return unknownAMM, p.DexInfo.ProgramId
	}
	if p.DexInfo.AMM != "" {
		return p.DexInfo.AMM, p.DexInfo.ProgramId
	}
	return unknownAMM, p.DexInfo.ProgramId
}

// unknownAMM is the AMM label of hops through programs without a known name
// (the name constants.GetProgramName gives unknown programs)
const unknownAMM = "Unknown"

// containsDCAProgram checks if transaction contains DCA program
func (p *JupiterParser) containsDCAProgram() bool {
	for _, key := range p.Adapter.AccountKeys {
		if key == constants.DEX_PROGRAMS.JUPITER_DCA.ID {
			return true
		}
	}
	return false
}

// findEmittingInstruction returns the instruction of event's program that
// emitted the Anchor self-CPI event: the last non-event instruction of that
// program before the event in the same outer instruction (the outer
// instruction when the program is called directly), or nil.
func findEmittingInstruction(a *adapter.TransactionAdapter, instructions []types.ClassifiedInstruction, event types.ClassifiedInstruction) *types.ClassifiedInstruction {
	var found *types.ClassifiedInstruction
	for i := range instructions {
		ci := &instructions[i]
		if ci.ProgramId != event.ProgramId || ci.OuterIndex != event.OuterIndex || ci.InnerIndex >= event.InnerIndex {
			continue
		}
		if isAnchorEvent(a.GetInstructionData(ci.Instruction)) {
			continue
		}
		if found == nil || ci.InnerIndex > found.InnerIndex {
			found = ci
		}
	}
	return found
}
