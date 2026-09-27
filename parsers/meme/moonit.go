package meme

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// MoonitParser parses Moonit (MoonShot) transactions
type MoonitParser struct {
	*parsers.BaseParser
	eventParser *MoonitEventParser
}

// NewMoonitParser creates a new Moonit parser
func NewMoonitParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MoonitParser {
	return &MoonitParser{
		BaseParser:  parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
		eventParser: NewMoonitEventParser(adapter, transferActions),
	}
}

// ProcessTrades parses Moonit trades from the buy and sell instructions. The
// amounts are the trader's (the instruction's sender), not the signer's
// balance changes, so routed and multi-trade transactions are correct.
func (p *MoonitParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, event := range p.eventParser.ParseInstructions(p.ClassifiedInstructions) {
		if event.Type != types.TradeTypeBuy && event.Type != types.TradeTypeSell {
			continue
		}
		if event.InputToken == nil || event.OutputToken == nil {
			continue
		}
		trade := &types.TradeInfo{
			Type:        event.Type,
			Pool:        []string{event.Pool},
			InputToken:  *event.InputToken,
			OutputToken: *event.OutputToken,
			User:        event.User,
			ProgramId:   constants.DEX_PROGRAMS.MOONIT.ID,
			AMM:         constants.DEX_PROGRAMS.MOONIT.Name,
			Route:       p.DexInfo.Route,
			Slot:        p.Adapter.Slot(),
			Timestamp:   p.Adapter.BlockTime(),
			Signature:   p.Adapter.Signature(),
			Idx:         event.Idx,
		}
		if len(event.Fees) > 0 {
			trade.Fees = append([]types.FeeInfo(nil), event.Fees...)
			trade.Fee = types.TotalFee(event.Fees)
		}
		trades = append(trades, *p.Utils.AttachTokenTransferInfo(trade, p.TransferActions))
	}

	return trades
}
