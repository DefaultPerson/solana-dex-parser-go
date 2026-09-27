package meme

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
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

// ParseInstructions parses classified instructions into meme events, in
// execution order
func (p *HeavenEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	ordered := append([]types.ClassifiedInstruction(nil), instructions...)
	utils.SortInstructionsByExecution(ordered)

	// Heaven pool creations by base mint: the Metaplex create of that mint
	// in the same transaction is the Heaven CREATE event (on mainnet it is
	// a separate outer instruction before create_standard_liquidity_pool)
	poolCreations := map[string][]string{}
	for _, ci := range ordered {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if ci.ProgramId == constants.DEX_PROGRAMS.HEAVEN.ID && len(data) >= 8 && bytes.Equal(data[:8], constants.DISCRIMINATORS.HEAVEN.CREATE_POOL) {
			if accounts := p.adapter.GetInstructionAccounts(ci.Instruction); len(accounts) > 6 {
				poolCreations[accounts[5]] = accounts
			}
		}
	}

	for _, ci := range ordered {
		data := p.adapter.GetInstructionData(ci.Instruction)
		var event *types.MemeEvent

		switch ci.ProgramId {
		case constants.DEX_PROGRAMS.HEAVEN.ID:
			if len(data) < 8 {
				continue
			}
			disc := data[:8]
			if bytes.Equal(disc, constants.DISCRIMINATORS.HEAVEN.BUY) {
				event = p.decodeTradeEvent(data[8:], ci, types.TradeTypeBuy)
			} else if bytes.Equal(disc, constants.DISCRIMINATORS.HEAVEN.SELL) {
				event = p.decodeTradeEvent(data[8:], ci, types.TradeTypeSell)
			} else if bytes.Equal(disc, constants.DISCRIMINATORS.HEAVEN.CREATE_POOL) {
				event = p.decodeInitialBuyEvent(ci)
			}
		case constants.METAPLEX_PROGRAM_ID:
			if len(poolCreations) > 0 && len(data) >= 1 && bytes.Equal(data[:1], constants.DISCRIMINATORS.METAPLEX.CREATE_MINT) {
				event = p.decodeCreateEvent(data[1:], ci, poolCreations)
			}
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

// decodeInitialBuyEvent decodes create_standard_liquidity_pool, which
// includes the creator's initial buy. Accounts (upstream): 4 user, 5 base
// mint, 6 quote mint, 10 pool, 11 platform config.
func (p *HeavenEventParser) decodeInitialBuyEvent(ci types.ClassifiedInstruction) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 12 {
		return nil
	}

	bondingCurve := accounts[10]
	userAccount := accounts[4]
	quoteMint := accounts[6]
	baseMint := accounts[5]

	event := &types.MemeEvent{
		Protocol:       constants.DEX_PROGRAMS.HEAVEN.Name,
		Type:           types.TradeTypeBuy,
		BaseMint:       baseMint,
		QuoteMint:      quoteMint,
		BondingCurve:   bondingCurve,
		Pool:           bondingCurve,
		User:           userAccount,
		PlatformConfig: accounts[11],
	}

	// the initial buy is what the user paid in the quote mint for the base
	// it received; the pool's seed liquidity is not the user's
	in, out := userLegs(p.adapter, instructionTransfers(p.transferActions, ci), userAccount, quoteMint, baseMint)
	if in == nil || out == nil {
		return nil
	}
	event.InputToken, event.OutputToken = in, out
	return event
}

// decodeTradeEvent decodes buy and sell. Accounts (upstream
// parser-heaven-event.ts): 4 pool, 5 user, 6 base mint, 7 quote mint, 12
// platform config. Amounts are the user's transfers in the instruction;
// without them the instruction args are used (amount in, minimum out).
func (p *HeavenEventParser) decodeTradeEvent(data []byte, ci types.ClassifiedInstruction, tradeType types.TradeType) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 8 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	argIn := reader.ReadU64AsBigInt()
	argOut := reader.ReadU64AsBigInt()
	if reader.HasError() {
		return nil
	}

	pool := accounts[4]
	user := accounts[5]
	baseMint := accounts[6]
	quoteMint := accounts[7]
	inMint, outMint := quoteMint, baseMint
	if tradeType == types.TradeTypeSell {
		inMint, outMint = baseMint, quoteMint
	}

	event := &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.HEAVEN.Name,
		Type:         tradeType,
		BaseMint:     baseMint,
		QuoteMint:    quoteMint,
		BondingCurve: pool,
		Pool:         pool,
		User:         user,
	}
	if len(accounts) > 12 {
		event.PlatformConfig = accounts[12]
	}

	in, out := userLegs(p.adapter, instructionTransfers(p.transferActions, ci), user, inMint, outMint)
	if in == nil {
		in = tokenInfoFromRaw(p.adapter, inMint, argIn)
	}
	if out == nil {
		out = tokenInfoFromRaw(p.adapter, outMint, argOut)
	}
	event.InputToken, event.OutputToken = in, out
	return event
}

// decodeCreateEvent decodes the Metaplex create instruction of a token
// launched with a Heaven pool (upstream decodes it the same way): data is the
// CreateArgs variant, then name, symbol, uri; accounts 2 mint, 4 payer. The
// quote mint, pool and platform config come from the pool creation of that
// mint (accounts 6, 10, 11 of create_standard_liquidity_pool); a Metaplex
// create of any other mint is not a Heaven event.
func (p *HeavenEventParser) decodeCreateEvent(data []byte, ci types.ClassifiedInstruction, poolCreations map[string][]string) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 5 {
		return nil
	}
	creation := poolCreations[accounts[2]]
	if creation == nil {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	reader.Skip(1) // CreateArgs variant
	name, _ := reader.ReadString()
	symbol, _ := reader.ReadString()
	uri, _ := reader.ReadString()
	if reader.HasError() {
		return nil
	}

	event := &types.MemeEvent{
		Protocol:  constants.DEX_PROGRAMS.HEAVEN.Name,
		Type:      types.TradeTypeCreate,
		Timestamp: p.adapter.BlockTime(),
		User:      accounts[4],
		Creator:   accounts[4],
		BaseMint:  accounts[2],
		QuoteMint: creation[6],
		Name:      name,
		Symbol:    symbol,
		URI:       uri,
	}
	if len(creation) > 11 {
		event.BondingCurve = creation[10]
		event.Pool = creation[10]
		event.PlatformConfig = creation[11]
	}
	if d, ok := p.adapter.KnownDecimals(accounts[2]); ok {
		event.Decimals = &d
	}
	return event
}

// ProcessEvents implements the EventParser interface
func (p *HeavenEventParser) ProcessEvents() []types.MemeEvent {
	instructions := utils.ProgramInstructions(p.adapter,
		constants.DEX_PROGRAMS.HEAVEN.ID,
		constants.METAPLEX_PROGRAM_ID,
	)
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}
