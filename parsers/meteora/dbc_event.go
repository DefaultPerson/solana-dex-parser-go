package meteora

import (
	"bytes"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// DBC trade directions (EvtSwap trade_direction)
const (
	dbcBaseToQuote = 0 // sell
	dbcQuoteToBase = 1 // buy
)

// MeteoraDBCEventParser parses Meteora DBC events
type MeteoraDBCEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
	utils           *utils.TransactionUtils
}

// NewMeteoraDBCEventParser creates a new event parser
func NewMeteoraDBCEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *MeteoraDBCEventParser {
	return &MeteoraDBCEventParser{
		adapter:         adapter,
		transferActions: transferActions,
		utils:           utils.NewTransactionUtils(adapter),
	}
}

// dbcSwapEvent is a decoded EvtSwap, EvtSwap2 or EvtSwap2WithTransferHook
type dbcSwapEvent struct {
	Pool           string
	TradeDirection uint8
	InputAmount    uint64 // what the user sent, fee included
	OutputAmount   uint64 // what the user received
	FeeOnInput     bool   // fees were taken from the input side
	TradingFee     uint64
	ProtocolFee    uint64
	ReferralFee    uint64
	QuoteReserve   *uint64 // EvtSwap2: pool quote reserve after the swap
}

// ParseInstructions parses classified instructions into meme events: one
// trade per swap instruction (amounts, direction and fees from the swap
// event it emits), creates, migrations and curve-complete events. Events are
// returned in execution order.
func (p *MeteoraDBCEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	ordered := make([]types.ClassifiedInstruction, 0, len(instructions))
	for _, ci := range instructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.METEORA_DBC.ID {
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
		case bytes.Equal(disc, constants.ANCHOR_EVENT_PREFIX):
			if len(data) >= 16 && (bytes.Equal(data[:16], constants.DISCRIMINATORS.METEORA_DBC.EVT_CURVE_COMPLETE) ||
				bytes.Equal(data[:16], constants.DISCRIMINATORS.METEORA_DBC.EVT_CURVE_COMPLETE_WITH_TRANSFER_HOOK)) {
				event = p.decodeCurveCompleteEvent(data[16:], ordered, pos)
			}
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.SWAP) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.SWAP_V2) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.SWAP2_WITH_TRANSFER_HOOK):
			event = p.decodeTradeEvent(ci, ordered, pos)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_SPL) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022_TRANSFER_HOOK):
			event = p.decodeCreateEvent(data[8:], ci.Instruction)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.METEORA_DBC_MIGRATE_DAMM):
			event = p.decodeDBCMigrateDammEvent(ci.Instruction)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.METEORA_DBC_MIGRATE_DAMM_V2):
			event = p.decodeDBCMigrateDammV2Event(ci.Instruction)
		}

		if event != nil {
			event.Protocol = constants.DEX_PROGRAMS.METEORA_DBC.Name
			event.Signature = p.adapter.Signature()
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
			events = append(events, event)
		}
	}

	return events
}

// swapEventFor returns the swap event emitted by the swap instruction at
// position pos (utils.EmittedEvents): the EvtSwap2 (or its transfer-hook
// variant), else the legacy EvtSwap. Both are emitted by current program
// versions.
func (p *MeteoraDBCEventParser) swapEventFor(ordered []types.ClassifiedInstruction, pos int, pool string) *dbcSwapEvent {
	var legacy *dbcSwapEvent
	for _, e := range utils.EmittedEvents(p.adapter, ordered, ordered[pos]) {
		data := p.adapter.GetInstructionData(e.Instruction)
		var evt *dbcSwapEvent
		switch {
		case bytes.Equal(data[:16], constants.DISCRIMINATORS.METEORA_DBC.EVT_SWAP2),
			bytes.Equal(data[:16], constants.DISCRIMINATORS.METEORA_DBC.EVT_SWAP2_WITH_TRANSFER_HOOK):
			if evt = decodeDBCEvtSwap2(data[16:]); evt != nil && evt.Pool == pool {
				return evt
			}
		case bytes.Equal(data[:16], constants.DISCRIMINATORS.METEORA_DBC.EVT_SWAP):
			if evt = decodeDBCEvtSwap(data[16:]); evt != nil && evt.Pool == pool && legacy == nil {
				legacy = evt
			}
		}
	}
	return legacy
}

// decodeDBCEvtSwap decodes the legacy EvtSwap: pool, config, trade_direction,
// has_referral, params {amount_in, minimum_amount_out}, swap_result
// {actual_input_amount, output_amount, next_sqrt_price u128, trading_fee,
// protocol_fee, referral_fee}, amount_in, current_timestamp
func decodeDBCEvtSwap(data []byte) *dbcSwapEvent {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	evt := &dbcSwapEvent{}
	evt.Pool, _ = reader.ReadPubkey()
	reader.Skip(32) // config
	evt.TradeDirection, _ = reader.ReadU8()
	reader.Skip(1)  // has_referral
	reader.Skip(16) // params
	actualInput, _ := reader.ReadU64()
	evt.OutputAmount, _ = reader.ReadU64()
	reader.Skip(16) // next_sqrt_price
	evt.TradingFee, _ = reader.ReadU64()
	evt.ProtocolFee, _ = reader.ReadU64()
	evt.ReferralFee, _ = reader.ReadU64()
	evt.InputAmount, _ = reader.ReadU64() // amount_in: transferred from the user
	if reader.HasError() {
		return nil
	}
	evt.FeeOnInput = evt.InputAmount > actualInput
	return evt
}

// decodeDBCEvtSwap2 decodes EvtSwap2 / EvtSwap2WithTransferHook: pool,
// config, trade_direction, has_referral, swap_parameters {amount_0,
// amount_1, swap_mode}, swap_result {included_fee_input_amount,
// excluded_fee_input_amount, amount_left, output_amount, next_sqrt_price
// u128, trading_fee, protocol_fee, referral_fee}, quote_reserve_amount,
// migration_threshold, current_timestamp
func decodeDBCEvtSwap2(data []byte) *dbcSwapEvent {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	evt := &dbcSwapEvent{}
	evt.Pool, _ = reader.ReadPubkey()
	reader.Skip(32) // config
	evt.TradeDirection, _ = reader.ReadU8()
	reader.Skip(1)  // has_referral
	reader.Skip(17) // swap_parameters
	evt.InputAmount, _ = reader.ReadU64()
	excludedFeeInput, _ := reader.ReadU64()
	reader.Skip(8) // amount_left
	evt.OutputAmount, _ = reader.ReadU64()
	reader.Skip(16) // next_sqrt_price
	evt.TradingFee, _ = reader.ReadU64()
	evt.ProtocolFee, _ = reader.ReadU64()
	evt.ReferralFee, _ = reader.ReadU64()
	quoteReserve, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}
	evt.FeeOnInput = evt.InputAmount > excludedFeeInput
	evt.QuoteReserve = &quoteReserve
	return evt
}

// decodeTradeEvent builds the trade of a swap / swap2 /
// swap2_with_transfer_hook instruction. Accounts (IDL): 2 pool, 3 input
// token account, 4 output token account, 7 base mint, 8 quote mint, 9 payer.
func (p *MeteoraDBCEventParser) decodeTradeEvent(ci types.ClassifiedInstruction, ordered []types.ClassifiedInstruction, pos int) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 10 {
		return nil
	}

	pool := accounts[2]
	baseMint := accounts[7]
	quoteMint := accounts[8]
	user := accounts[9]

	event := &types.MemeEvent{
		BaseMint:     baseMint,
		QuoteMint:    quoteMint,
		BondingCurve: pool,
		Pool:         pool,
		User:         user,
	}

	if evt := p.swapEventFor(ordered, pos, pool); evt != nil {
		inputMint, outputMint := quoteMint, baseMint
		event.Type = types.TradeTypeBuy
		if evt.TradeDirection == dbcBaseToQuote {
			inputMint, outputMint = baseMint, quoteMint
			event.Type = types.TradeTypeSell
		}
		event.InputToken = p.tokenInfo(inputMint, new(big.Int).SetUint64(evt.InputAmount))
		event.OutputToken = p.tokenInfo(outputMint, new(big.Int).SetUint64(evt.OutputAmount))

		feeMint := outputMint
		if evt.FeeOnInput {
			feeMint = inputMint
		}
		event.Fees = p.fees(feeMint, evt)
		if evt.QuoteReserve != nil {
			event.RealQuoteReserves = new(big.Int).SetUint64(*evt.QuoteReserve).String()
		}
		return event
	}

	// No swap event: take the instruction's own transfers and the direction
	// from the payer's base-mint token account
	transfers := p.utils.FilterTransfersForInstruction(p.transferActions, ci.ProgramId, ci.OuterIndex, ci.InnerIndex, []string{"transfer", "transferChecked"})
	if len(transfers) < 2 {
		return nil
	}
	trade := p.utils.ProcessSwapData(transfers[:2], types.DexInfo{}, false)
	if trade == nil {
		return nil
	}
	event.Type = getAccountTradeType(user, baseMint, accounts[3], accounts[4])
	if event.Type == types.TradeTypeSwap {
		event.Type = trade.Type
	}
	event.InputToken = &trade.InputToken
	event.OutputToken = &trade.OutputToken
	return event
}

// tokenInfo returns a token amount with the decimals known to the transaction
func (p *MeteoraDBCEventParser) tokenInfo(mint string, amount *big.Int) *types.TokenInfo {
	decimals := p.adapter.GetTokenDecimals(mint)
	return &types.TokenInfo{
		Mint:      mint,
		AmountRaw: amount.String(),
		Amount:    types.ConvertToUIAmount(amount, decimals),
		Decimals:  decimals,
	}
}

// fees lists the trading (pool / partner and creator), protocol and referral
// fees of a swap, in the mint they were charged in
func (p *MeteoraDBCEventParser) fees(mint string, evt *dbcSwapEvent) []types.FeeInfo {
	decimals := p.adapter.GetTokenDecimals(mint)
	var fees []types.FeeInfo
	add := func(amount uint64, feeType string) {
		if amount == 0 {
			return
		}
		v := new(big.Int).SetUint64(amount)
		fees = append(fees, types.FeeInfo{
			Mint:      mint,
			Amount:    types.ConvertToUIAmount(v, decimals),
			AmountRaw: v.String(),
			Decimals:  decimals,
			Dex:       constants.DEX_PROGRAMS.METEORA_DBC.Name,
			Type:      feeType,
		})
	}
	add(evt.TradingFee, "trading")
	add(evt.ProtocolFee, "protocol")
	add(evt.ReferralFee, "referral")
	return fees
}

// decodeCurveCompleteEvent decodes EvtCurveComplete (and its transfer-hook
// variant): pool, config, base_reserve, quote_reserve. The mints come from
// the swap instruction on the same pool that emitted it.
func (p *MeteoraDBCEventParser) decodeCurveCompleteEvent(data []byte, ordered []types.ClassifiedInstruction, pos int) *types.MemeEvent {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	pool, _ := reader.ReadPubkey()
	config, _ := reader.ReadPubkey()
	baseReserve := reader.ReadU64AsBigInt()
	quoteReserve := reader.ReadU64AsBigInt()
	if reader.HasError() {
		return nil
	}

	event := &types.MemeEvent{
		Type:              types.TradeTypeComplete,
		Pool:              pool,
		BondingCurve:      pool,
		PlatformConfig:    config,
		RealBaseReserves:  baseReserve.String(),
		RealQuoteReserves: quoteReserve.String(),
	}
	emitter := utils.FindEventEmitter(p.adapter, ordered, ordered[pos], func(data []byte, accounts []string) bool {
		return len(data) >= 8 && len(accounts) >= 10 && accounts[2] == pool
	})
	if emitter != nil {
		accounts := p.adapter.GetInstructionAccounts(emitter.Instruction)
		event.BaseMint = accounts[7]
		event.QuoteMint = accounts[8]
		event.User = accounts[9]
	}
	return event
}

// decodeCreateEvent decodes a create event
func (p *MeteoraDBCEventParser) decodeCreateEvent(data []byte, instruction interface{}) *types.MemeEvent {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

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

	// config 0, creator 2, base_mint 3, quote_mint 4, pool 5 in every
	// initialize_virtual_pool_* variant
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 6 {
		return nil
	}

	return &types.MemeEvent{
		Type:           types.TradeTypeCreate,
		Name:           name,
		Symbol:         symbol,
		URI:            uri,
		User:           accounts[2],
		Creator:        accounts[2],
		BaseMint:       accounts[3],
		QuoteMint:      accounts[4],
		Pool:           accounts[5],
		BondingCurve:   accounts[5],
		PlatformConfig: accounts[0],
	}
}

// decodeDBCMigrateDammEvent decodes a migrate to DAMM event. Accounts (IDL,
// 31): 0 virtual_pool, 2 config, 4 pool, 7 token_a_mint, 8 token_b_mint.
func (p *MeteoraDBCEventParser) decodeDBCMigrateDammEvent(instruction interface{}) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 9 {
		return nil
	}

	return &types.MemeEvent{
		Type:           types.TradeTypeMigrate,
		BaseMint:       accounts[7],
		QuoteMint:      accounts[8],
		PlatformConfig: accounts[2],
		BondingCurve:   accounts[0],
		Pool:           accounts[4],
		PoolDex:        constants.DEX_PROGRAMS.METEORA_DAMM.Name,
	}
}

// decodeDBCMigrateDammV2Event decodes a migrate to DAMM V2 event
func (p *MeteoraDBCEventParser) decodeDBCMigrateDammV2Event(instruction interface{}) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 15 {
		return nil
	}

	return &types.MemeEvent{
		Type:           types.TradeTypeMigrate,
		BaseMint:       accounts[13],
		QuoteMint:      accounts[14],
		PlatformConfig: accounts[2],
		BondingCurve:   accounts[0],
		Pool:           accounts[4],
		PoolDex:        constants.DEX_PROGRAMS.METEORA_DAMM_V2.Name,
	}
}

// getAccountTradeType determines the trade type from the user's token
// accounts: selling when the input account is the user's associated token
// account of the base mint, buying when the output account is
func getAccountTradeType(user, baseMint, inputAccount, outputAccount string) types.TradeType {
	standard, token2022, err := utils.FindAssociatedTokenAddress(user, baseMint)
	if err != nil {
		return types.TradeTypeSwap
	}
	switch {
	case inputAccount == standard || inputAccount == token2022:
		return types.TradeTypeSell
	case outputAccount == standard || outputAccount == token2022:
		return types.TradeTypeBuy
	}
	return types.TradeTypeSwap
}

// ProcessEvents implements the EventParser interface
func (p *MeteoraDBCEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForProgramMeteoraDBC(p.adapter, constants.DEX_PROGRAMS.METEORA_DBC.ID)
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}

// getAllInstructionsForProgramMeteoraDBC gets all instructions for Meteora DBC program
func getAllInstructionsForProgramMeteoraDBC(adapter *adapter.TransactionAdapter, programId string) []types.ClassifiedInstruction {
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
