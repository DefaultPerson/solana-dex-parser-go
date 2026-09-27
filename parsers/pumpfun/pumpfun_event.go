package pumpfun

import (
	"bytes"
	"math/big"
	"strconv"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
	"github.com/mr-tron/base58"
)

// pumpfunBaseDecimals is the decimals of every Pump.fun bonding-curve mint
const pumpfunBaseDecimals = 6

// PumpfunEventParser parses Pumpfun events
type PumpfunEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
}

// NewPumpfunEventParser creates a new event parser
func NewPumpfunEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *PumpfunEventParser {
	return &PumpfunEventParser{
		adapter:         adapter,
		transferActions: transferActions,
	}
}

// pumpfunTradeEvent is a decoded Pump.fun TradeEvent. Fields after
// creatorFee exist only in newer program versions (zero when absent).
type pumpfunTradeEvent struct {
	Mint                  string
	SolAmount             uint64
	TokenAmount           uint64
	IsBuy                 bool
	User                  string
	Timestamp             int64
	VirtualSolReserves    uint64
	VirtualTokenReserves  uint64
	RealSolReserves       uint64
	RealTokenReserves     uint64
	HasFees               bool // fee_recipient .. creator_fee present
	FeeRecipient          string
	FeeBasisPoints        uint64
	Fee                   uint64 // protocol fee, including BuybackFee
	Creator               string
	CreatorFeeBasisPoints uint64
	CreatorFee            uint64
	IxName                string
	MayhemMode            bool
	CashbackFeeBps        uint64
	Cashback              uint64
	BuybackFeeBps         uint64
	BuybackFee            uint64
	HasQuote              bool // quote_mint .. real_quote_reserves present
	QuoteMint             string
	QuoteAmount           uint64
	VirtualQuoteReserves  uint64
	RealQuoteReserves     uint64
	HolderRewardsBps      uint64
	HolderRewards         uint64
}

// pumpfunTradeIx describes the account layout of a Pump.fun trade
// instruction: where the base mint and the bonding curve are
type pumpfunTradeIx struct {
	name         string
	mintIndex    int
	curveIndex   int
	quoteIndex   int // -1: SOL only (legacy layouts)
	programIndex int // base token program, -1 when not in the layout
	buybackIndex int // buyback fee recipient, -1 when not in the layout
}

// pumpfunTradeIxs maps the trade instruction discriminators to their layouts
// (pump.json IDL): legacy buy/sell/buy_exact_sol_in keep the bonding curve at
// 3; the quote-mint aware *_v2 instructions move it to 10.
var pumpfunTradeIxs = []struct {
	disc []byte
	ix   pumpfunTradeIx
}{
	{constants.DISCRIMINATORS.PUMPFUN.BUY, pumpfunTradeIx{"buy", 2, 3, -1, -1, -1}},
	{constants.DISCRIMINATORS.PUMPFUN.SELL, pumpfunTradeIx{"sell", 2, 3, -1, -1, -1}},
	{constants.DISCRIMINATORS.PUMPFUN.BUY_EXACT_SOL_IN, pumpfunTradeIx{"buy_exact_sol_in", 2, 3, -1, -1, -1}},
	{constants.DISCRIMINATORS.PUMPFUN.BUY_V2, pumpfunTradeIx{"buy_v2", 1, 10, 2, 3, 8}},
	{constants.DISCRIMINATORS.PUMPFUN.SELL_V2, pumpfunTradeIx{"sell_v2", 1, 10, 2, 3, 8}},
	{constants.DISCRIMINATORS.PUMPFUN.BUY_EXACT_QUOTE_IN_V2, pumpfunTradeIx{"buy_exact_quote_in_v2", 1, 10, 2, 3, 8}},
}

func pumpfunTradeIxLayout(data []byte) *pumpfunTradeIx {
	for i := range pumpfunTradeIxs {
		if len(data) >= 8 && bytes.Equal(data[:8], pumpfunTradeIxs[i].disc) {
			return &pumpfunTradeIxs[i].ix
		}
	}
	return nil
}

// normalizePumpfunIxName maps the v2 instruction names to the legacy names
// they share semantics with
func normalizePumpfunIxName(name string) string {
	switch name {
	case "buy_v2":
		return "buy"
	case "sell_v2":
		return "sell"
	case "buy_exact_quote_in_v2":
		return "buy_exact_quote_in"
	}
	return name
}

// ParseInstructions parses classified instructions into meme events. Events
// are matched to the instruction that emitted them within the same outer
// instruction, so several Pump.fun instructions in one transaction (create +
// buy, bundles) each get their own bonding curve.
func (p *PumpfunEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	ordered := executionOrder(instructions)
	for pos, ci := range ordered {
		if ci.ProgramId != constants.DEX_PROGRAMS.PUMP_FUN.ID {
			continue
		}

		data := p.adapter.GetInstructionData(ci.Instruction)
		if !isEventData(data) {
			continue
		}

		disc := data[:16]
		var event *types.MemeEvent

		switch {
		case bytes.Equal(disc, constants.DISCRIMINATORS.PUMPFUN.TRADE_EVENT):
			if evt := decodePumpfunTradeEvent(data[16:]); evt != nil {
				event = p.tradeEventToMeme(evt, ordered, pos)
			}
		case bytes.Equal(disc, constants.DISCRIMINATORS.PUMPFUN.CREATE_EVENT):
			event = p.decodeCreateEvent(data[16:], ordered, pos)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PUMPFUN.COMPLETE_EVENT):
			event = p.decodeCompleteEvent(data[16:])
		case bytes.Equal(disc, constants.DISCRIMINATORS.PUMPFUN.MIGRATE_EVENT):
			event = p.decodeMigrateEvent(data[16:])
		}

		if event != nil {
			event.Signature = p.adapter.Signature()
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
			events = append(events, event)
		}
	}

	// ordered is in execution order already
	return events
}

// decodePumpfunTradeEvent decodes a TradeEvent field by field in IDL order
// (pump-fun/pump-public-docs idl/pump.json). The oldest events end after
// virtual_token_reserves (121 bytes); every later program version appended
// fields, so decoding stops cleanly at the end of the data.
func decodePumpfunTradeEvent(data []byte) *pumpfunTradeEvent {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	evt := &pumpfunTradeEvent{}
	evt.Mint, _ = reader.ReadPubkey()
	evt.SolAmount, _ = reader.ReadU64()
	evt.TokenAmount, _ = reader.ReadU64()
	evt.IsBuy, _ = reader.ReadBool()
	evt.User, _ = reader.ReadPubkey()
	evt.Timestamp, _ = reader.ReadI64()
	evt.VirtualSolReserves, _ = reader.ReadU64()
	evt.VirtualTokenReserves, _ = reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	t := newTailReader(reader)
	evt.RealSolReserves = t.u64()
	evt.RealTokenReserves = t.u64()
	evt.FeeRecipient = t.pubkey()
	evt.FeeBasisPoints = t.u64() // u64 in the IDL (not u16)
	evt.Fee = t.u64()
	evt.Creator = t.pubkey()
	evt.CreatorFeeBasisPoints = t.u64()
	evt.CreatorFee = t.u64()
	evt.HasFees = t.ok

	t.boolean() // track_volume
	t.u64()     // total_unclaimed_tokens
	t.u64()     // total_claimed_tokens
	t.u64()     // current_sol_volume
	t.u64()     // last_update_timestamp
	evt.IxName = t.str()
	evt.MayhemMode = t.boolean()
	evt.CashbackFeeBps = t.u64()
	evt.Cashback = t.u64()
	evt.BuybackFeeBps = t.u64()
	evt.BuybackFee = t.u64()
	t.skipVec(34) // shareholders: Vec<(pubkey, u16)>
	evt.QuoteMint = t.pubkey()
	evt.QuoteAmount = t.u64()
	evt.VirtualQuoteReserves = t.u64()
	evt.RealQuoteReserves = t.u64()
	evt.HasQuote = t.ok
	evt.HolderRewardsBps = t.u64()
	evt.HolderRewards = t.u64()
	return evt
}

// tradeEventToMeme builds the meme event of a TradeEvent at position pos of
// ordered.
//
// Amounts are what the user sent (buy) or received (sell) in the quote mint:
// the event's quote amount is the bonding-curve amount, and the program adds
// the protocol fee (which includes the buyback fee), the creator fee and the
// cashback on top of it for buys and deducts them for sells. The components
// are listed in Fees.
func (p *PumpfunEventParser) tradeEventToMeme(evt *pumpfunTradeEvent, ordered []types.ClassifiedInstruction, pos int) *types.MemeEvent {
	parent := findParentInstruction(p.adapter, ordered, pos, constants.DEX_PROGRAMS.PUMP_FUN.ID,
		func(data []byte, accounts []string) bool {
			layout := pumpfunTradeIxLayout(data)
			return layout != nil && layout.mintIndex < len(accounts) && accounts[layout.mintIndex] == evt.Mint
		})

	var parentLayout *pumpfunTradeIx
	var parentAccounts []string
	if parent != nil {
		parentLayout = pumpfunTradeIxLayout(p.adapter.GetInstructionData(parent.Instruction))
		parentAccounts = p.adapter.GetInstructionAccounts(parent.Instruction)
	}

	// Quote mint: from the event (Pubkey::default = SOL), else from the v2
	// instruction accounts, else SOL (legacy events are SOL-only)
	quoteMint := constants.TOKENS.SOL
	if evt.HasQuote {
		quoteMint = normalizeQuoteMint(evt.QuoteMint)
	} else if parentLayout != nil && parentLayout.quoteIndex >= 0 && parentLayout.quoteIndex < len(parentAccounts) {
		quoteMint = normalizeQuoteMint(parentAccounts[parentLayout.quoteIndex])
	}
	quoteAmount := u64(evt.SolAmount)
	if evt.HasQuote && evt.QuoteAmount != 0 {
		quoteAmount = u64(evt.QuoteAmount)
	}
	quoteDecimals := tokenDecimals(p.adapter, quoteMint, 0)
	baseDecimals := tokenDecimals(p.adapter, evt.Mint, pumpfunBaseDecimals)

	// Bonding curve: account of the emitting instruction, else the PDA
	bondingCurve := ""
	if parentLayout != nil && parentLayout.curveIndex < len(parentAccounts) {
		bondingCurve = parentAccounts[parentLayout.curveIndex]
	}
	if bondingCurve == "" {
		bondingCurve = pumpfunBondingCurvePDA(evt.Mint)
	}

	dex := constants.DEX_PROGRAMS.PUMP_FUN.Name
	var fees []types.FeeInfo
	if evt.Fee > evt.BuybackFee {
		fees = append(fees, feeInfo(quoteMint, u64(evt.Fee-evt.BuybackFee), quoteDecimals, dex, "protocol", evt.FeeRecipient))
	}
	if evt.BuybackFee > 0 {
		buybackRecipient := ""
		if parentLayout != nil && parentLayout.buybackIndex >= 0 && parentLayout.buybackIndex < len(parentAccounts) {
			buybackRecipient = parentAccounts[parentLayout.buybackIndex]
		}
		fees = append(fees, feeInfo(quoteMint, u64(evt.BuybackFee), quoteDecimals, dex, "buyback", buybackRecipient))
	}
	if evt.CreatorFee > 0 {
		fees = append(fees, feeInfo(quoteMint, u64(evt.CreatorFee), quoteDecimals, dex, "coinCreator", evt.Creator))
	}
	if evt.Cashback > 0 {
		fees = append(fees, feeInfo(quoteMint, u64(evt.Cashback), quoteDecimals, dex, "cashback", evt.User))
	}
	totalFee := sumFees(fees)

	userQuote := new(big.Int).Set(quoteAmount)
	if evt.IsBuy {
		userQuote.Add(userQuote, totalFee)
	} else if userQuote.Cmp(totalFee) >= 0 {
		userQuote.Sub(userQuote, totalFee)
	}
	quoteToken := &types.TokenInfo{
		Mint:      quoteMint,
		AmountRaw: userQuote.String(),
		Amount:    types.ConvertToUIAmount(userQuote, quoteDecimals),
		Decimals:  quoteDecimals,
	}
	baseToken := &types.TokenInfo{
		Mint:      evt.Mint,
		AmountRaw: strconv.FormatUint(evt.TokenAmount, 10),
		Amount:    types.ConvertToUIAmountUint64(evt.TokenAmount, baseDecimals),
		Decimals:  baseDecimals,
	}

	event := &types.MemeEvent{
		Protocol:            dex,
		Type:                types.TradeTypeSell,
		BaseMint:            evt.Mint,
		QuoteMint:           quoteMint,
		User:                evt.User,
		Timestamp:           evt.Timestamp,
		InputToken:          baseToken,
		OutputToken:         quoteToken,
		BondingCurve:        bondingCurve,
		Pool:                bondingCurve,
		IxName:              normalizePumpfunIxName(evt.IxName),
		IsMayhemMode:        evt.MayhemMode,
		IsCashbackEnabled:   evt.CashbackFeeBps > 0,
		IsHolderReward:      evt.HolderRewardsBps > 0,
		Fees:                fees,
		VirtualBaseReserves: strconv.FormatUint(evt.VirtualTokenReserves, 10),
	}
	if evt.IsBuy {
		event.Type = types.TradeTypeBuy
		event.InputToken, event.OutputToken = quoteToken, baseToken
	}
	if event.IxName == "" && parentLayout != nil {
		event.IxName = normalizePumpfunIxName(parentLayout.name)
	}
	if parentLayout != nil && parentLayout.programIndex >= 0 && parentLayout.programIndex < len(parentAccounts) {
		event.TokenProgram = parentAccounts[parentLayout.programIndex]
	}
	if evt.HasFees {
		event.Creator = evt.Creator
		event.ProtocolFee = uiPtr(u64(evt.Fee), quoteDecimals)
		event.CreatorFee = uiPtr(u64(evt.CreatorFee), quoteDecimals)
		bps := evt.CreatorFeeBasisPoints
		event.CreatorFeeBps = &bps
		event.RealBaseReserves = strconv.FormatUint(evt.RealTokenReserves, 10)
		event.RealQuoteReserves = strconv.FormatUint(evt.RealSolReserves, 10)
	}
	event.VirtualQuoteReserves = strconv.FormatUint(evt.VirtualSolReserves, 10)
	if evt.HasQuote {
		event.VirtualQuoteReserves = strconv.FormatUint(evt.VirtualQuoteReserves, 10)
		event.RealQuoteReserves = strconv.FormatUint(evt.RealQuoteReserves, 10)
	}
	return event
}

// pumpfunBondingCurvePDA derives the bonding curve of mint: PDA of
// ["bonding-curve", mint] under the Pump.fun program
func pumpfunBondingCurvePDA(mint string) string {
	mintBytes, err := base58.Decode(mint)
	if err != nil || len(mintBytes) != 32 {
		return ""
	}
	pda, _, err := utils.FindProgramAddress([][]byte{[]byte("bonding-curve"), mintBytes}, constants.DEX_PROGRAMS.PUMP_FUN.ID)
	if err != nil {
		return ""
	}
	return pda
}

// decodeCreateEvent decodes a CreateEvent field by field in IDL order; older
// events end after user (no creator) or after timestamp (no reserves)
func (p *PumpfunEventParser) decodeCreateEvent(data []byte, ordered []types.ClassifiedInstruction, pos int) *types.MemeEvent {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	name, _ := reader.ReadString()
	symbol, _ := reader.ReadString()
	uri, _ := reader.ReadString()
	mint, _ := reader.ReadPubkey()
	bondingCurve, _ := reader.ReadPubkey()
	user, _ := reader.ReadPubkey()
	if reader.HasError() {
		return nil
	}

	event := &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.PUMP_FUN.Name,
		Type:         types.TradeTypeCreate,
		User:         user,
		BaseMint:     mint,
		QuoteMint:    constants.TOKENS.SOL,
		Name:         name,
		Symbol:       symbol,
		URI:          uri,
		BondingCurve: bondingCurve,
	}

	t := newTailReader(reader)
	event.Creator = t.pubkey()
	event.Timestamp = int64(t.u64())
	virtualTokenReserves := t.u64()
	virtualSolReserves := t.u64()
	realTokenReserves := t.u64()
	totalSupply := t.u64()
	if t.ok {
		event.VirtualBaseReserves = strconv.FormatUint(virtualTokenReserves, 10)
		event.VirtualQuoteReserves = strconv.FormatUint(virtualSolReserves, 10)
		event.RealBaseReserves = strconv.FormatUint(realTokenReserves, 10)
		supply := types.ConvertToUIAmountUint64(totalSupply, pumpfunBaseDecimals)
		event.TotalSupply = &supply
		decimals := uint8(pumpfunBaseDecimals)
		event.Decimals = &decimals
	}
	event.TokenProgram = t.pubkey()
	event.IsMayhemMode = t.boolean()
	event.IsCashbackEnabled = t.boolean()
	if quoteMint := t.pubkey(); quoteMint != "" {
		event.QuoteMint = normalizeQuoteMint(quoteMint)
	} else {
		// No quote_mint in the event: for create_v2, remaining accounts
		// 16-18 are quote_mint, its bonding-curve account and token program
		// (WSOL or absent means SOL)
		parent := findParentInstruction(p.adapter, ordered, pos, constants.DEX_PROGRAMS.PUMP_FUN.ID,
			func(data []byte, accounts []string) bool {
				return len(data) >= 8 && bytes.Equal(data[:8], constants.DISCRIMINATORS.PUMPFUN.CREATE_V2)
			})
		if parent != nil {
			if accounts := p.adapter.GetInstructionAccounts(parent.Instruction); len(accounts) > 16 {
				event.QuoteMint = normalizeQuoteMint(accounts[16])
			}
		}
	}
	if virtualQuoteReserves := t.u64(); t.ok {
		event.VirtualQuoteReserves = strconv.FormatUint(virtualQuoteReserves, 10)
	}
	creatorFeeBps := t.u64()
	if t.ok {
		event.CreatorFeeBps = &creatorFeeBps
	}
	event.IsHolderReward = t.boolean()

	return event
}

// decodeCompleteEvent decodes a CompleteEvent (the bonding curve is full);
// quote_mint was appended later
func (p *PumpfunEventParser) decodeCompleteEvent(data []byte) *types.MemeEvent {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	user, _ := reader.ReadPubkey()
	mint, _ := reader.ReadPubkey()
	bondingCurve, _ := reader.ReadPubkey()
	timestamp, _ := reader.ReadI64()
	if reader.HasError() {
		return nil
	}

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.PUMP_FUN.Name,
		Type:         types.TradeTypeComplete,
		Timestamp:    timestamp,
		User:         user,
		BaseMint:     mint,
		QuoteMint:    normalizeQuoteMint(newTailReader(reader).pubkey()),
		BondingCurve: bondingCurve,
	}
}

// decodeMigrateEvent decodes a CompletePumpAmmMigrationEvent: 160 bytes in
// the original layout, 192 bytes since quote_mint was appended
func (p *PumpfunEventParser) decodeMigrateEvent(data []byte) *types.MemeEvent {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	user, _ := reader.ReadPubkey()
	mint, _ := reader.ReadPubkey()
	mintAmount, _ := reader.ReadU64()
	solAmount, _ := reader.ReadU64()
	poolMigrationFee, _ := reader.ReadU64()
	bondingCurve, _ := reader.ReadPubkey()
	timestamp, _ := reader.ReadI64()
	pool, _ := reader.ReadPubkey()
	if reader.HasError() {
		return nil
	}

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.PUMP_FUN.Name,
		Type:         types.TradeTypeMigrate,
		Timestamp:    timestamp,
		User:         user,
		BaseMint:     mint,
		QuoteMint:    normalizeQuoteMint(newTailReader(reader).pubkey()),
		BondingCurve: bondingCurve,
		Pool:         pool,
		PoolDex:      constants.DEX_PROGRAMS.PUMP_SWAP.Name,
		BaseAmount:   strconv.FormatUint(mintAmount, 10),
		QuoteAmount:  strconv.FormatUint(solAmount, 10),
		MigrationFee: strconv.FormatUint(poolMigrationFee, 10),
	}
}

// ProcessEvents implements EventParser interface for meme event parsers
func (p *PumpfunEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForProgram(p.adapter, constants.DEX_PROGRAMS.PUMP_FUN.ID)
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}

// getAllInstructionsForProgram gets all instructions for a program ID
func getAllInstructionsForProgram(adapter *adapter.TransactionAdapter, programId string) []types.ClassifiedInstruction {
	var instructions []types.ClassifiedInstruction

	// Outer instructions
	for outerIdx, ix := range adapter.Instructions() {
		programIdFromIx := adapter.GetInstructionProgramId(ix)
		if programIdFromIx == programId {
			instructions = append(instructions, types.ClassifiedInstruction{
				ProgramId:   programIdFromIx,
				Instruction: ix,
				OuterIndex:  outerIdx,
				InnerIndex:  -1,
			})
		}
	}

	// Inner instructions
	for _, innerSet := range adapter.InnerInstructions() {
		for innerIdx, ix := range innerSet.Instructions {
			programIdFromIx := adapter.GetInstructionProgramId(ix)
			if programIdFromIx == programId {
				instructions = append(instructions, types.ClassifiedInstruction{
					ProgramId:   programIdFromIx,
					Instruction: ix,
					OuterIndex:  innerSet.Index,
					InnerIndex:  innerIdx,
				})
			}
		}
	}

	return instructions
}
