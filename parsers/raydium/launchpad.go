package raydium

import (
	"bytes"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// RaydiumLaunchpadParser parses Raydium Launchpad transactions
type RaydiumLaunchpadParser struct {
	*parsers.BaseParser
	eventParser *RaydiumLaunchpadEventParser
}

// NewRaydiumLaunchpadParser creates a new Raydium Launchpad parser
func NewRaydiumLaunchpadParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *RaydiumLaunchpadParser {
	return &RaydiumLaunchpadParser{
		BaseParser:  parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
		eventParser: NewRaydiumLaunchpadEventParser(adapter, transferActions),
	}
}

// ProcessTrades parses Raydium Launchpad trades
func (p *RaydiumLaunchpadParser) ProcessTrades() []types.TradeInfo {
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
func (p *RaydiumLaunchpadParser) createTradeInfo(event *types.MemeEvent) *types.TradeInfo {
	if event.InputToken == nil || event.OutputToken == nil || event.InputToken.Mint == "" || event.OutputToken.Mint == "" {
		return nil
	}

	var pool []string
	if event.Pool != "" {
		pool = []string{event.Pool}
	}

	programId := p.DexInfo.ProgramId
	if programId == "" {
		programId = constants.DEX_PROGRAMS.RAYDIUM_LCP.ID
	}

	trade := &types.TradeInfo{
		Type:        event.Type,
		Pool:        pool,
		InputToken:  *event.InputToken,
		OutputToken: *event.OutputToken,
		User:        event.User,
		ProgramId:   programId,
		AMM:         constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Route:       p.DexInfo.Route,
		Slot:        p.Adapter.Slot(),
		Timestamp:   p.Adapter.BlockTime(),
		Signature:   p.Adapter.Signature(),
		Idx:         event.Idx,
	}

	// Fees from the TradeEvent (protocol, platform, creator, share), all in
	// the quote mint; Fee is their exact sum
	if len(event.Fees) > 0 {
		trade.Fees = append([]types.FeeInfo(nil), event.Fees...)
		trade.Fee = types.TotalFee(event.Fees)
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// RaydiumLaunchpadEventParser parses Raydium Launchpad events
type RaydiumLaunchpadEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
}

// NewRaydiumLaunchpadEventParser creates a new event parser
func NewRaydiumLaunchpadEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *RaydiumLaunchpadEventParser {
	return &RaydiumLaunchpadEventParser{
		adapter:         adapter,
		transferActions: transferActions,
	}
}

// lcpEventPrefix is the 8-byte prefix of Anchor self-CPI event instructions
var lcpEventPrefix = []byte{228, 69, 165, 46, 81, 203, 154, 29}

// lcpTradeIxNames maps the trade instruction discriminators to their names
var lcpTradeIxNames = []struct {
	disc []byte
	name string
}{
	{constants.DISCRIMINATORS.RAYDIUM_LCP.BUY_EXACT_IN, "buy_exact_in"},
	{constants.DISCRIMINATORS.RAYDIUM_LCP.BUY_EXACT_OUT, "buy_exact_out"},
	{constants.DISCRIMINATORS.RAYDIUM_LCP.SELL_EXACT_IN, "sell_exact_in"},
	{constants.DISCRIMINATORS.RAYDIUM_LCP.SELL_EXACT_OUT, "sell_exact_out"},
}

// lcpInitializeDiscs are the pool creation instructions; their accounts
// share the layout 1 creator, 3 platform_config, 5 pool_state, 6 base_mint,
// 7 quote_mint
var lcpInitializeDiscs = [][]byte{
	constants.DISCRIMINATORS.RAYDIUM_LCP.INITIALIZE,
	constants.DISCRIMINATORS.RAYDIUM_LCP.INITIALIZE_V2,
	constants.DISCRIMINATORS.RAYDIUM_LCP.INITIALIZE_WITH_TOKEN_2022,
}

// ParseInstructions parses classified instructions into meme events, in
// execution order. Trades and creates are matched to the TradeEvent /
// PoolCreateEvent self-CPI that follows the instruction in the same outer
// group, so CPI calls (launch platforms, routers) decode like outer ones.
func (p *RaydiumLaunchpadEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	ordered := make([]types.ClassifiedInstruction, 0, len(instructions))
	for _, ci := range instructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.RAYDIUM_LCP.ID {
			ordered = append(ordered, ci)
		}
	}
	types.SortInstructionsByExecution(ordered)

	for pos, ci := range ordered {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		var event *types.MemeEvent
		disc := data[:8]

		switch {
		case bytes.Equal(disc, lcpEventPrefix):
			if len(data) >= 16 && bytes.Equal(data[:16], constants.DISCRIMINATORS.RAYDIUM_LCP.CREATE_EVENT) {
				event = p.decodeCreateEvent(data[16:], ordered, pos)
			}
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.MIGRATE_TO_AMM) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.MIGRATE_TO_CPSWAP):
			event = p.decodeCompleteInstruction(data, ci.Instruction)
		default:
			for _, ix := range lcpTradeIxNames {
				if bytes.Equal(disc, ix.disc) {
					event = p.decodeTradeInstruction(ci, ordered, pos, ix.name)
					break
				}
			}
		}

		if event != nil {
			event.Signature = p.adapter.Signature()
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
			events = append(events, event)
		}
	}

	return events
}

// followingEvent returns the data (after the 16-byte discriminator) of the
// first event with discriminator disc that follows position pos in the same
// outer group, before the next non-event LaunchLab instruction
func (p *RaydiumLaunchpadEventParser) followingEvent(ordered []types.ClassifiedInstruction, pos int, disc []byte) []byte {
	outer := ordered[pos].OuterIndex
	for j := pos + 1; j < len(ordered) && ordered[j].OuterIndex == outer; j++ {
		data := p.adapter.GetInstructionData(ordered[j].Instruction)
		if len(data) < 16 || !bytes.Equal(data[:8], lcpEventPrefix) {
			break
		}
		if bytes.Equal(data[:16], disc) {
			return data[16:]
		}
	}
	return nil
}

// decodeLCPTradeEvent decodes a TradeEvent: 139 bytes in the current layout
// (creator_fee before share_fee, exact_in at the end), 130 bytes in the
// original one (no creator_fee)
func decodeLCPTradeEvent(data []byte) *RaydiumLCPTradeEvent {
	if len(data) >= 138 {
		if layout, err := ParseRaydiumLCPTradeV2Layout(data); err == nil {
			return layout.ToObject()
		}
		return nil
	}
	if layout, err := ParseRaydiumLCPTradeLayout(data); err == nil {
		return layout.ToObject()
	}
	return nil
}

// decodeTradeInstruction decodes a trade instruction with the TradeEvent it
// emits. Accounts (IDL): 0 payer, 3 platform_config, 4 pool_state, 9 base
// mint, 10 quote mint. The event amounts are what the user paid (buys:
// amount_in includes the fees) and received (sells: amount_out after fees).
func (p *RaydiumLaunchpadEventParser) decodeTradeInstruction(ci types.ClassifiedInstruction, ordered []types.ClassifiedInstruction, pos int, ixName string) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 11 {
		return nil
	}
	eventData := p.followingEvent(ordered, pos, constants.DISCRIMINATORS.RAYDIUM_LCP.TRADE_EVENT)
	if eventData == nil {
		return nil
	}
	evt := decodeLCPTradeEvent(eventData)
	if evt == nil || (evt.PoolState != accounts[4] && accounts[4] != "") {
		return nil
	}
	evt.User = accounts[0]
	evt.BaseMint = accounts[9]
	evt.QuoteMint = accounts[10]

	baseDecimals := p.adapter.GetTokenDecimals(evt.BaseMint)
	quoteDecimals := p.adapter.GetTokenDecimals(evt.QuoteMint)
	token := func(mint string, amount *big.Int, decimals uint8) *types.TokenInfo {
		return &types.TokenInfo{
			Mint:      mint,
			AmountRaw: amount.String(),
			Amount:    types.ConvertToUIAmount(amount, decimals),
			Decimals:  decimals,
		}
	}

	event := &types.MemeEvent{
		Protocol:             constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:                 types.TradeTypeSell,
		BondingCurve:         evt.PoolState,
		Pool:                 evt.PoolState,
		PlatformConfig:       accounts[3],
		BaseMint:             evt.BaseMint,
		QuoteMint:            evt.QuoteMint,
		User:                 evt.User,
		IxName:               ixName,
		InputToken:           token(evt.BaseMint, evt.AmountIn, baseDecimals),
		OutputToken:          token(evt.QuoteMint, evt.AmountOut, quoteDecimals),
		ProtocolFee:          uiAmountPtr(evt.ProtocolFee, quoteDecimals),
		PlatformFee:          uiAmountPtr(evt.PlatformFee, quoteDecimals),
		ShareFee:             uiAmountPtr(evt.ShareFee, quoteDecimals),
		CreatorFee:           uiAmountPtr(evt.CreatorFee, quoteDecimals),
		VirtualBaseReserves:  evt.VirtualBase.String(),
		VirtualQuoteReserves: evt.VirtualQuote.String(),
		RealBaseReserves:     evt.RealBaseAfter.String(),
		RealQuoteReserves:    evt.RealQuoteAfter.String(),
	}
	if evt.TradeDirection == TradeDirectionBuy {
		event.Type = types.TradeTypeBuy
		event.InputToken = token(evt.QuoteMint, evt.AmountIn, quoteDecimals)
		event.OutputToken = token(evt.BaseMint, evt.AmountOut, baseDecimals)
	}

	dex := constants.DEX_PROGRAMS.RAYDIUM_LCP.Name
	for _, f := range []struct {
		amount  *big.Int
		feeType string
	}{
		{evt.ProtocolFee, "protocol"},
		{evt.PlatformFee, "platform"},
		{evt.CreatorFee, "coinCreator"},
		{evt.ShareFee, "share"},
	} {
		if f.amount == nil || f.amount.Sign() == 0 {
			continue
		}
		event.Fees = append(event.Fees, types.FeeInfo{
			Mint:      evt.QuoteMint,
			Amount:    types.ConvertToUIAmount(f.amount, quoteDecimals),
			AmountRaw: f.amount.String(),
			Decimals:  quoteDecimals,
			Dex:       dex,
			Type:      f.feeType,
		})
	}
	return event
}

func uiAmountPtr(val *big.Int, decimals uint8) *float64 {
	v := types.ConvertToUIAmount(val, decimals)
	return &v
}

// decodeCreateEvent decodes a PoolCreateEvent. The mints are accounts 6 and
// 7 of the initialize* instruction that emitted it (the nearest preceding
// LaunchLab instruction in the same outer group), which is not necessarily
// the outer instruction.
func (p *RaydiumLaunchpadEventParser) decodeCreateEvent(data []byte, ordered []types.ClassifiedInstruction, pos int) *types.MemeEvent {
	layout, err := ParsePoolCreateEventLayout(data)
	if err != nil {
		return nil
	}
	evt := layout.ToObject()

	var platformConfig string
	outer := ordered[pos].OuterIndex
	for j := pos - 1; j >= 0 && ordered[j].OuterIndex == outer; j-- {
		ixData := p.adapter.GetInstructionData(ordered[j].Instruction)
		if len(ixData) < 8 || bytes.Equal(ixData[:8], lcpEventPrefix) {
			continue
		}
		accounts := p.adapter.GetInstructionAccounts(ordered[j].Instruction)
		for _, disc := range lcpInitializeDiscs {
			if bytes.Equal(ixData[:8], disc) && len(accounts) >= 8 && accounts[5] == evt.PoolState {
				platformConfig = accounts[3]
				evt.BaseMint = accounts[6]
				evt.QuoteMint = accounts[7]
			}
		}
		break
	}
	if evt.BaseMint == "" {
		return nil
	}

	decimals := evt.BaseMintParam.Decimals
	return &types.MemeEvent{
		Protocol:       constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:           types.TradeTypeCreate,
		Timestamp:      p.adapter.BlockTime(),
		User:           evt.Creator,
		BaseMint:       evt.BaseMint,
		QuoteMint:      evt.QuoteMint,
		Name:           evt.BaseMintParam.Name,
		Symbol:         evt.BaseMintParam.Symbol,
		URI:            evt.BaseMintParam.URI,
		Decimals:       &decimals,
		BondingCurve:   evt.PoolState,
		Pool:           evt.PoolState,
		PlatformConfig: platformConfig,
		Creator:        evt.Creator,
		Curve:          evt.memeCurveParams(),
	}
}

// decodeCompleteInstruction decodes a migrate instruction
func (p *RaydiumLaunchpadEventParser) decodeCompleteInstruction(data []byte, instruction interface{}) *types.MemeEvent {
	if len(data) < 8 {
		return nil
	}

	discriminator := data[:8]
	accounts := p.adapter.GetInstructionAccounts(instruction)

	var baseMint, quoteMint, poolMint string
	var amm string

	if bytes.Equal(discriminator, constants.DISCRIMINATORS.RAYDIUM_LCP.MIGRATE_TO_AMM) {
		if len(accounts) < 17 {
			return nil
		}
		baseMint = accounts[1]
		quoteMint = accounts[2]
		poolMint = accounts[13]
		amm = constants.DEX_PROGRAMS.RAYDIUM_V4.Name
	} else {
		if len(accounts) < 8 {
			return nil
		}
		baseMint = accounts[1]
		quoteMint = accounts[2]
		poolMint = accounts[5]
		amm = constants.DEX_PROGRAMS.RAYDIUM_CPMM.Name
	}

	return &types.MemeEvent{
		Protocol:  constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:      types.TradeTypeMigrate,
		Timestamp: p.adapter.BlockTime(),
		BaseMint:  baseMint,
		QuoteMint: quoteMint,
		Pool:      poolMint,
		PoolDex:   amm,
	}
}

// ProcessEvents implements the EventParser interface
func (p *RaydiumLaunchpadEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForProgramRaydiumLCP(p.adapter, constants.DEX_PROGRAMS.RAYDIUM_LCP.ID)
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}

// getAllInstructionsForProgramRaydiumLCP gets all instructions for Raydium Launchpad program
func getAllInstructionsForProgramRaydiumLCP(adapter *adapter.TransactionAdapter, programId string) []types.ClassifiedInstruction {
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
