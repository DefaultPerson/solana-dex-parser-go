package jupiter

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/parsers"
	"github.com/solana-dex-parser-go/types"
	"github.com/solana-dex-parser-go/utils"
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

// ProcessTrades parses Jupiter swap trades
func (p *JupiterParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if p.isJupiterRouteEventInstruction(ci.Instruction, ci.ProgramId) {
			innerIdx := ci.InnerIndex
			if innerIdx < 0 {
				innerIdx = 0
			}
			event := p.parseJupiterRouteEventInstruction(ci.Instruction, fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx))
			if event != nil {
				data := p.processSwapData([]*JupiterSwapEvent{event})
				if data != nil {
					trades = append(trades, *p.Utils.AttachTokenTransferInfo(data, p.TransferActions))
				}
			}
		}
	}

	return trades
}

// isJupiterRouteEventInstruction checks if instruction is Jupiter route event
func (p *JupiterParser) isJupiterRouteEventInstruction(instruction interface{}, programId string) bool {
	if programId != constants.DEX_PROGRAMS.JUPITER.ID {
		return false
	}

	data := p.Adapter.GetInstructionData(instruction)
	if len(data) < 16 {
		return false
	}

	return bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER.ROUTE_EVENT)
}

// parseJupiterRouteEventInstruction parses Jupiter route event instruction
func (p *JupiterParser) parseJupiterRouteEventInstruction(instruction interface{}, idx string) *JupiterSwapEvent {
	data := p.Adapter.GetInstructionData(instruction)
	if len(data) < 16 {
		return nil
	}

	eventData := data[16:]
	layout, err := ParseJupiterSwapLayout(eventData)
	if err != nil {
		return nil
	}

	event := layout.ToSwapEvent()
	event.InputMintDecimals = p.Adapter.GetTokenDecimals(event.InputMint)
	event.OutputMintDecimals = p.Adapter.GetTokenDecimals(event.OutputMint)
	event.Idx = idx

	return event
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
	TokenIn  map[string]*big.Int
	TokenOut map[string]*big.Int
	Decimals map[string]uint8
	Idx      string
}

// buildIntermediateInfo builds intermediate swap info from events
func (p *JupiterParser) buildIntermediateInfo(events []*JupiterSwapEvent) *JupiterSwapInfo {
	info := &JupiterSwapInfo{
		AMMs:     make([]string, 0),
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
		ProgramId: p.DexInfo.ProgramId,
		AMM:       p.getAMM(info),
		Route:     p.DexInfo.Route,
		Slot:      p.Adapter.Slot(),
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
		Idx:       info.Idx,
	}

	if p.containsDCAProgram() {
		// Jupiter DCA fee 0.1%
		feeAmount := new(big.Int).Div(outAmount, big.NewInt(1000))
		feeUIAmount := types.ConvertToUIAmount(feeAmount, outDecimals)
		trade.Fee = &types.FeeInfo{
			Mint:      outMint,
			Amount:    feeUIAmount,
			AmountRaw: feeAmount.String(),
			Decimals:  outDecimals,
		}
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// getAMM gets the AMM name
func (p *JupiterParser) getAMM(info *JupiterSwapInfo) string {
	if len(info.AMMs) > 0 {
		return info.AMMs[0]
	}
	if p.DexInfo.AMM != "" {
		return p.DexInfo.AMM
	}
	return ""
}

// containsDCAProgram checks if transaction contains DCA program
func (p *JupiterParser) containsDCAProgram() bool {
	for _, key := range p.Adapter.AccountKeys {
		if key == constants.DEX_PROGRAMS.JUPITER_DCA.ID {
			return true
		}
	}
	return false
}
