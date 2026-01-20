package pumpfun

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// PumpfunParser parses Pumpfun transactions
type PumpfunParser struct {
	*parsers.BaseParser
	eventParser *PumpfunEventParser
}

// NewPumpfunParser creates a new Pumpfun parser
func NewPumpfunParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *PumpfunParser {
	return &PumpfunParser{
		BaseParser:  parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
		eventParser: NewPumpfunEventParser(adapter, transferActions),
	}
}

// ProcessTrades parses Pumpfun trades
func (p *PumpfunParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	events := p.eventParser.ParseInstructions(p.ClassifiedInstructions)

	for _, event := range events {
		if event.Type == types.TradeTypeBuy || event.Type == types.TradeTypeSell {
			trade := p.createTradeInfo(event)
			if trade != nil {
				trades = append(trades, *trade)
			}
		}
	}

	return trades
}

// createTradeInfo creates a TradeInfo from a MemeEvent
func (p *PumpfunParser) createTradeInfo(event *types.MemeEvent) *types.TradeInfo {
	trade := getPumpfunTradeInfo(event, tradeInfoParams{
		Slot:      p.Adapter.Slot(),
		Signature: p.Adapter.Signature(),
		Timestamp: event.Timestamp,
		Idx:       event.Idx,
		DexInfo:   p.DexInfo,
	})

	return p.Utils.AttachTokenTransferInfo(&trade, p.TransferActions)
}
