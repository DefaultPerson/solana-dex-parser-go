package pumpfun

import (
	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/parsers"
	"github.com/solana-dex-parser-go/types"
)

// PumpswapParser parses Pumpswap transactions
type PumpswapParser struct {
	*parsers.BaseParser
	eventParser *PumpswapEventParser
}

// NewPumpswapParser creates a new Pumpswap parser
func NewPumpswapParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *PumpswapParser {
	return &PumpswapParser{
		BaseParser:  parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
		eventParser: NewPumpswapEventParser(adapter, transferActions),
	}
}

// ProcessTrades parses Pumpswap trades
func (p *PumpswapParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	events := p.eventParser.ParseInstructions(p.ClassifiedInstructions)

	for _, event := range events {
		if event.Type == "BUY" {
			trade := p.createBuyInfo(event)
			if trade != nil {
				trades = append(trades, *trade)
			}
		} else if event.Type == "SELL" {
			trade := p.createSellInfo(event)
			if trade != nil {
				trades = append(trades, *trade)
			}
		}
	}

	return trades
}

// createBuyInfo creates trade info from buy event
func (p *PumpswapParser) createBuyInfo(event *PumpswapEvent) *types.TradeInfo {
	data, ok := event.Data.(*PumpswapBuyEventData)
	if !ok {
		return nil
	}

	// Get token mints from adapter
	inputMint := p.Adapter.GetSplTokenMint(data.UserQuoteTokenAccount)
	if inputMint == "" {
		return nil
	}
	outputMint := p.Adapter.GetSplTokenMint(data.UserBaseTokenAccount)
	if outputMint == "" {
		return nil
	}
	feeMint := p.Adapter.GetSplTokenMint(data.ProtocolFeeRecipientTokenAccount)
	if feeMint == "" {
		return nil
	}

	inputDecimals := p.Adapter.GetTokenDecimals(inputMint)
	outputDecimals := p.Adapter.GetTokenDecimals(outputMint)
	feeDecimals := p.Adapter.GetTokenDecimals(feeMint)

	trade := getPumpswapBuyInfo(
		data,
		tokenInfo{Mint: inputMint, Decimals: inputDecimals},
		tokenInfo{Mint: outputMint, Decimals: outputDecimals},
		tokenInfo{Mint: feeMint, Decimals: feeDecimals},
		tradeInfoParams{
			Slot:      event.Slot,
			Signature: event.Signature,
			Timestamp: event.Timestamp,
			Idx:       event.Idx,
			DexInfo:   p.DexInfo,
		},
	)

	return p.Utils.AttachTokenTransferInfo(&trade, p.TransferActions)
}

// createSellInfo creates trade info from sell event
func (p *PumpswapParser) createSellInfo(event *PumpswapEvent) *types.TradeInfo {
	data, ok := event.Data.(*PumpswapSellEventData)
	if !ok {
		return nil
	}

	// Get token mints from adapter
	inputMint := p.Adapter.GetSplTokenMint(data.UserBaseTokenAccount)
	if inputMint == "" {
		return nil
	}
	outputMint := p.Adapter.GetSplTokenMint(data.UserQuoteTokenAccount)
	if outputMint == "" {
		return nil
	}
	feeMint := p.Adapter.GetSplTokenMint(data.ProtocolFeeRecipientTokenAccount)
	if feeMint == "" {
		return nil
	}

	inputDecimals := p.Adapter.GetTokenDecimals(inputMint)
	outputDecimals := p.Adapter.GetTokenDecimals(outputMint)
	feeDecimals := p.Adapter.GetTokenDecimals(feeMint)

	trade := getPumpswapSellInfo(
		data,
		tokenInfo{Mint: inputMint, Decimals: inputDecimals},
		tokenInfo{Mint: outputMint, Decimals: outputDecimals},
		tokenInfo{Mint: feeMint, Decimals: feeDecimals},
		tradeInfoParams{
			Slot:      event.Slot,
			Signature: event.Signature,
			Timestamp: event.Timestamp,
			Idx:       event.Idx,
			DexInfo:   p.DexInfo,
		},
	)

	return p.Utils.AttachTokenTransferInfo(&trade, p.TransferActions)
}
