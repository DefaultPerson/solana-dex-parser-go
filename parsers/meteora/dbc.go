package meteora

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// MeteoraDBCParser parses Meteora Dynamic Bonding Curve transactions
type MeteoraDBCParser struct {
	*parsers.BaseParser
	eventParser *MeteoraDBCEventParser
}

// NewMeteoraDBCParser creates a new Meteora DBC parser
func NewMeteoraDBCParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MeteoraDBCParser {
	return &MeteoraDBCParser{
		BaseParser:  parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
		eventParser: NewMeteoraDBCEventParser(adapter, transferActions),
	}
}

// ProcessTrades parses Meteora DBC trades
func (p *MeteoraDBCParser) ProcessTrades() []types.TradeInfo {
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
func (p *MeteoraDBCParser) createTradeInfo(event *types.MemeEvent) *types.TradeInfo {
	if event.InputToken == nil || event.OutputToken == nil {
		return nil
	}

	programId := p.DexInfo.ProgramId
	if programId == "" {
		programId = constants.DEX_PROGRAMS.METEORA_DBC.ID
	}

	trade := &types.TradeInfo{
		Type:        event.Type,
		Pool:        []string{event.Pool},
		InputToken:  *event.InputToken,
		OutputToken: *event.OutputToken,
		User:        event.User,
		ProgramId:   programId,
		AMM:         constants.DEX_PROGRAMS.METEORA_DBC.Name,
		Route:       p.DexInfo.Route,
		Slot:        p.Adapter.Slot(),
		Timestamp:   event.Timestamp,
		Signature:   p.Adapter.Signature(),
		Idx:         event.Idx,
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}
