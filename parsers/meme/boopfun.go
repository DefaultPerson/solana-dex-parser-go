package meme

import (
	"bytes"
	"math/big"

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

// ParseInstructions parses classified instructions into meme events, in
// execution order
func (p *BoopfunEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	ordered := append([]types.ClassifiedInstruction(nil), instructions...)
	types.SortInstructionsByExecution(ordered)

	// deploy_bonding_curve accounts by mint (0 mint, 2 bonding curve, 5
	// config): the bonding curve of a created token
	deploys := map[string][]string{}
	for _, ci := range ordered {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if ci.ProgramId == constants.DEX_PROGRAMS.BOOP_FUN.ID && len(data) >= 8 && bytes.Equal(data[:8], constants.DISCRIMINATORS.BOOPFUN.DEPLOY) {
			if accounts := p.adapter.GetInstructionAccounts(ci.Instruction); len(accounts) >= 6 {
				deploys[accounts[0]] = accounts
			}
		}
	}

	for _, ci := range ordered {
		if ci.ProgramId != constants.DEX_PROGRAMS.BOOP_FUN.ID {
			continue
		}

		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		disc := data[:8]
		var event *types.MemeEvent

		if bytes.Equal(disc, constants.DISCRIMINATORS.BOOPFUN.BUY) {
			event = p.decodeTradeEvent(data[8:], ci, types.TradeTypeBuy)
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.BOOPFUN.SELL) {
			event = p.decodeTradeEvent(data[8:], ci, types.TradeTypeSell)
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.BOOPFUN.CREATE) {
			event = p.decodeCreateEvent(data[8:], ci.Instruction)
			if event != nil {
				if deploy, ok := deploys[event.BaseMint]; ok {
					event.BondingCurve = deploy[2]
					event.Pool = deploy[2]
					event.PlatformConfig = deploy[5]
				}
			}
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.BOOPFUN.COMPLETE) {
			event = p.decodeCompleteEvent(ci.Instruction)
		}

		if event != nil {
			event.Signature = p.adapter.Signature()
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Idx = formatIdx(ci.OuterIndex, ci.InnerIndex)
			events = append(events, event)
		}
	}

	return events
}

// decodeTradeEvent decodes buy_token (args: buy_amount, amount_out_min) and
// sell_token (args: sell_amount, amount_out_min). Accounts: 0 mint, 1
// bonding curve, 6 user. The amounts are the user's transfers in the
// instruction (buys: the SOL paid including the trading fee; sells: the SOL
// received after it); without them the first argument is used for the input
// and the output is 0.
func (p *BoopfunEventParser) decodeTradeEvent(data []byte, ci types.ClassifiedInstruction, tradeType types.TradeType) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	argIn := reader.ReadU64AsBigInt()
	if reader.HasError() {
		return nil
	}

	mint := accounts[0]
	quoteMint := constants.TOKENS.SOL
	user := accounts[6]
	inMint, outMint := quoteMint, mint
	if tradeType == types.TradeTypeSell {
		inMint, outMint = mint, quoteMint
	}

	in, out := userLegs(p.adapter, instructionTransfers(p.transferActions, ci), user, inMint, outMint)
	if in == nil {
		in = tokenInfoFromRaw(p.adapter, inMint, argIn)
	}
	if out == nil {
		out = tokenInfoFromRaw(p.adapter, outMint, new(big.Int))
	}

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.BOOP_FUN.Name,
		Type:         tradeType,
		BondingCurve: accounts[1],
		Pool:         accounts[1],
		BaseMint:     mint,
		QuoteMint:    quoteMint,
		User:         user,
		InputToken:   in,
		OutputToken:  out,
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

func (p *BoopfunEventParser) decodeCompleteEvent(instruction interface{}) *types.MemeEvent {
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

// ProcessEvents implements the EventParser interface
func (p *BoopfunEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForMultiPrograms(p.adapter, []string{constants.DEX_PROGRAMS.BOOP_FUN.ID})
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}
