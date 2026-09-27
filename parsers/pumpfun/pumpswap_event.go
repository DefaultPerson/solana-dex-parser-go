package pumpfun

import (
	"bytes"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// PumpswapEvent represents a parsed Pumpswap event
type PumpswapEvent struct {
	Type      string
	Data      interface{}
	Slot      uint64
	Timestamp int64
	Signature string
	Idx       string
}

// PumpswapBuyEventData contains buy event data
type PumpswapBuyEventData struct {
	Timestamp                        int64
	BaseAmountOut                    uint64
	MaxQuoteAmountIn                 uint64
	UserBaseTokenReserves            uint64
	UserQuoteTokenReserves           uint64
	PoolBaseTokenReserves            uint64
	PoolQuoteTokenReserves           uint64
	QuoteAmountIn                    uint64
	LpFeeBasisPoints                 uint64
	LpFee                            uint64
	ProtocolFeeBasisPoints           uint64
	ProtocolFee                      uint64
	QuoteAmountInWithLpFee           uint64
	UserQuoteAmountIn                uint64
	Pool                             string
	User                             string
	UserBaseTokenAccount             string
	UserQuoteTokenAccount            string
	ProtocolFeeRecipient             string
	ProtocolFeeRecipientTokenAccount string
	CoinCreator                      string
	CoinCreatorFeeBasisPoints        uint64
	CoinCreatorFee                   uint64

	// Appended by later program versions (zero when absent)
	MinBaseAmountOut     uint64
	IxName               string
	CashbackFeeBps       uint64
	Cashback             uint64
	BuybackFeeBps        uint64
	BuybackFee           uint64 // part of ProtocolFee
	VirtualQuoteReserves *big.Int
	CanBoost             bool
	BaseSupply           uint64
	HolderRewardsBps     uint64
	HolderRewards        uint64 // equals CoinCreatorFee on holder-rewards coins

	// ExactQuoteIn is set for buy_exact_quote_in: QuoteAmountIn is then the
	// user's total (UserQuoteAmountIn is not)
	ExactQuoteIn bool
}

// UserQuoteIn returns the quote amount the user paid: pool input with the LP
// fee plus the protocol and coin-creator fees
func (e *PumpswapBuyEventData) UserQuoteIn() uint64 {
	if e.ExactQuoteIn {
		return e.QuoteAmountIn
	}
	return e.UserQuoteAmountIn
}

// PumpswapSellEventData contains sell event data
type PumpswapSellEventData struct {
	Timestamp                        int64
	BaseAmountIn                     uint64
	MinQuoteAmountOut                uint64
	UserBaseTokenReserves            uint64
	UserQuoteTokenReserves           uint64
	PoolBaseTokenReserves            uint64
	PoolQuoteTokenReserves           uint64
	QuoteAmountOut                   uint64
	LpFeeBasisPoints                 uint64
	LpFee                            uint64
	ProtocolFeeBasisPoints           uint64
	ProtocolFee                      uint64
	QuoteAmountOutWithoutLpFee       uint64
	UserQuoteAmountOut               uint64
	Pool                             string
	User                             string
	UserBaseTokenAccount             string
	UserQuoteTokenAccount            string
	ProtocolFeeRecipient             string
	ProtocolFeeRecipientTokenAccount string
	CoinCreator                      string
	CoinCreatorFeeBasisPoints        uint64
	CoinCreatorFee                   uint64

	// Appended by later program versions (zero when absent)
	CashbackFeeBps       uint64
	Cashback             uint64
	BuybackFeeBps        uint64
	BuybackFee           uint64 // part of ProtocolFee
	VirtualQuoteReserves *big.Int
	CanBoost             bool
	BaseSupply           uint64
	HolderRewardsBps     uint64
	HolderRewards        uint64 // equals CoinCreatorFee on holder-rewards coins
}

// PumpswapDepositEventData contains deposit event data
type PumpswapDepositEventData struct {
	Timestamp              int64
	LpTokenAmountOut       uint64
	MaxBaseAmountIn        uint64
	MaxQuoteAmountIn       uint64
	UserBaseTokenReserves  uint64
	UserQuoteTokenReserves uint64
	PoolBaseTokenReserves  uint64
	PoolQuoteTokenReserves uint64
	BaseAmountIn           uint64
	QuoteAmountIn          uint64
	LpMintSupply           uint64
	Pool                   string
	User                   string
	UserBaseTokenAccount   string
	UserQuoteTokenAccount  string
	UserPoolTokenAccount   string
}

// PumpswapWithdrawEventData contains withdraw event data
type PumpswapWithdrawEventData struct {
	Timestamp              int64
	LpTokenAmountIn        uint64
	MinBaseAmountOut       uint64
	MinQuoteAmountOut      uint64
	UserBaseTokenReserves  uint64
	UserQuoteTokenReserves uint64
	PoolBaseTokenReserves  uint64
	PoolQuoteTokenReserves uint64
	BaseAmountOut          uint64
	QuoteAmountOut         uint64
	LpMintSupply           uint64
	Pool                   string
	User                   string
	UserBaseTokenAccount   string
	UserQuoteTokenAccount  string
	UserPoolTokenAccount   string
}

// PumpswapCreatePoolEventData contains create pool event data
type PumpswapCreatePoolEventData struct {
	Timestamp             int64
	Index                 uint16
	Creator               string
	BaseMint              string
	QuoteMint             string
	BaseMintDecimals      uint8
	QuoteMintDecimals     uint8
	BaseAmountIn          uint64
	QuoteAmountIn         uint64
	PoolBaseAmount        uint64
	PoolQuoteAmount       uint64
	MinimumLiquidity      uint64
	InitialLiquidity      uint64
	LpTokenAmountOut      uint64
	PoolBump              uint8
	Pool                  string
	LpMint                string
	UserBaseTokenAccount  string
	UserQuoteTokenAccount string

	// Appended by later program versions (zero when absent)
	CoinCreator       string
	IsMayhemMode      bool
	CreatorFeeBps     *uint64
	CanEditCreatorFee bool
	IsHolderReward    bool
}

// PumpswapBoostBuyAndBurnEventData contains a BoostBuyAndBurnEvent: the
// protocol buys the coin with the boost vault and burns it
type PumpswapBoostBuyAndBurnEventData struct {
	Timestamp              int64
	Mint                   string
	BondingCurve           string
	Pool                   string
	Authority              string
	QuoteAmountInRequested uint64
	QuoteAmountInUsed      uint64
	BaseAmountBurned       uint64
	VirtualQuoteReserves   *big.Int
	RealQuoteReservesAfter uint64
	BaseReservesAfter      uint64
	BoostVaultRemaining    uint64

	// QuoteMint is the quote_mint account of the boost_buy_and_burn
	// instruction (the event does not carry it)
	QuoteMint string
}

// PumpswapEventParser parses Pumpswap events
type PumpswapEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
}

// NewPumpswapEventParser creates a new event parser
func NewPumpswapEventParser(adapter *adapter.TransactionAdapter, transferActions map[string][]types.TransferData) *PumpswapEventParser {
	return &PumpswapEventParser{
		adapter:         adapter,
		transferActions: transferActions,
	}
}

// ProcessEvents implements EventParser interface
func (p *PumpswapEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForProgram(p.adapter, constants.DEX_PROGRAMS.PUMP_SWAP.ID)
	events := p.ParseInstructions(instructions)

	// Convert to MemeEvent slice
	var result []types.MemeEvent
	for _, e := range events {
		if e != nil {
			memeEvent := p.convertToMemeEvent(e)
			if memeEvent != nil {
				result = append(result, *memeEvent)
			}
		}
	}
	return result
}

// convertToMemeEvent converts PumpswapEvent to MemeEvent. Trade amounts and
// fees are the same as in the TradeInfo of the trade: what the user paid or
// received, with decimals of the mints involved.
func (p *PumpswapEventParser) convertToMemeEvent(event *PumpswapEvent) *types.MemeEvent {
	memeEvent := &types.MemeEvent{
		Protocol:  constants.DEX_PROGRAMS.PUMP_SWAP.Name,
		Slot:      event.Slot,
		Timestamp: event.Timestamp,
		Signature: event.Signature,
		Idx:       event.Idx,
	}

	switch data := event.Data.(type) {
	case *PumpswapBuyEventData:
		baseMint, quoteMint := p.tradeMints(data.UserBaseTokenAccount, data.UserQuoteTokenAccount)
		baseDecimals := p.adapter.GetTokenDecimals(baseMint)
		quoteDecimals := p.adapter.GetTokenDecimals(quoteMint)
		memeEvent.Type = types.TradeTypeBuy
		memeEvent.User = data.User
		memeEvent.Pool = data.Pool
		memeEvent.BaseMint = baseMint
		memeEvent.QuoteMint = quoteMint
		memeEvent.InputToken = &types.TokenInfo{
			Mint:      quoteMint,
			AmountRaw: uint64ToString(data.UserQuoteIn()),
			Amount:    types.ConvertToUIAmountUint64(data.UserQuoteIn(), quoteDecimals),
			Decimals:  quoteDecimals,
		}
		memeEvent.OutputToken = &types.TokenInfo{
			Mint:      baseMint,
			AmountRaw: uint64ToString(data.BaseAmountOut),
			Amount:    types.ConvertToUIAmountUint64(data.BaseAmountOut, baseDecimals),
			Decimals:  baseDecimals,
		}
		memeEvent.IxName = data.IxName
		p.setTradeFeeFields(memeEvent, pumpswapFees(pumpswapFeeParams{
			protocolFee: data.ProtocolFee, buybackFee: data.BuybackFee, coinCreatorFee: data.CoinCreatorFee, cashback: data.Cashback,
			protocolRecipient: data.ProtocolFeeRecipient, coinCreator: data.CoinCreator, user: data.User,
		}, quoteMint, quoteDecimals), data.ProtocolFee, data.CoinCreatorFee, quoteDecimals)
		memeEvent.IsCashbackEnabled = data.CashbackFeeBps > 0
		memeEvent.IsHolderReward = data.HolderRewardsBps > 0
		if data.CoinCreator != defaultPubkey {
			memeEvent.Creator = data.CoinCreator
		}
	case *PumpswapSellEventData:
		baseMint, quoteMint := p.tradeMints(data.UserBaseTokenAccount, data.UserQuoteTokenAccount)
		baseDecimals := p.adapter.GetTokenDecimals(baseMint)
		quoteDecimals := p.adapter.GetTokenDecimals(quoteMint)
		memeEvent.Type = types.TradeTypeSell
		memeEvent.User = data.User
		memeEvent.Pool = data.Pool
		memeEvent.BaseMint = baseMint
		memeEvent.QuoteMint = quoteMint
		memeEvent.InputToken = &types.TokenInfo{
			Mint:      baseMint,
			AmountRaw: uint64ToString(data.BaseAmountIn),
			Amount:    types.ConvertToUIAmountUint64(data.BaseAmountIn, baseDecimals),
			Decimals:  baseDecimals,
		}
		memeEvent.OutputToken = &types.TokenInfo{
			Mint:      quoteMint,
			AmountRaw: uint64ToString(data.UserQuoteAmountOut),
			Amount:    types.ConvertToUIAmountUint64(data.UserQuoteAmountOut, quoteDecimals),
			Decimals:  quoteDecimals,
		}
		memeEvent.IxName = "sell"
		p.setTradeFeeFields(memeEvent, pumpswapFees(pumpswapFeeParams{
			protocolFee: data.ProtocolFee, buybackFee: data.BuybackFee, coinCreatorFee: data.CoinCreatorFee, cashback: data.Cashback,
			protocolRecipient: data.ProtocolFeeRecipient, coinCreator: data.CoinCreator, user: data.User,
		}, quoteMint, quoteDecimals), data.ProtocolFee, data.CoinCreatorFee, quoteDecimals)
		memeEvent.IsCashbackEnabled = data.CashbackFeeBps > 0
		memeEvent.IsHolderReward = data.HolderRewardsBps > 0
		if data.CoinCreator != defaultPubkey {
			memeEvent.Creator = data.CoinCreator
		}
	case *PumpswapCreatePoolEventData:
		memeEvent.Type = types.TradeTypeCreate
		memeEvent.User = data.Creator
		memeEvent.Pool = data.Pool
		memeEvent.BaseMint = data.BaseMint
		memeEvent.QuoteMint = data.QuoteMint
		memeEvent.Creator = data.Creator
		if data.CoinCreator != "" && data.CoinCreator != defaultPubkey {
			memeEvent.Creator = data.CoinCreator
		}
		decimals := data.BaseMintDecimals
		memeEvent.Decimals = &decimals
		memeEvent.IsMayhemMode = data.IsMayhemMode
		memeEvent.CreatorFeeBps = data.CreatorFeeBps
		memeEvent.IsHolderReward = data.IsHolderReward
	case *PumpswapBoostBuyAndBurnEventData:
		quoteMint := data.QuoteMint
		if quoteMint == "" {
			quoteMint = constants.TOKENS.SOL
		}
		baseDecimals := tokenDecimals(p.adapter, data.Mint, pumpfunBaseDecimals)
		quoteDecimals := p.adapter.GetTokenDecimals(quoteMint)
		memeEvent.Type = types.TradeTypeBuyAndBurn
		memeEvent.User = data.Authority
		memeEvent.Pool = data.Pool
		memeEvent.BondingCurve = data.BondingCurve
		memeEvent.BaseMint = data.Mint
		memeEvent.QuoteMint = quoteMint
		memeEvent.InputToken = &types.TokenInfo{
			Mint:      quoteMint,
			AmountRaw: uint64ToString(data.QuoteAmountInUsed),
			Amount:    types.ConvertToUIAmountUint64(data.QuoteAmountInUsed, quoteDecimals),
			Decimals:  quoteDecimals,
		}
		memeEvent.OutputToken = &types.TokenInfo{
			Mint:      data.Mint,
			AmountRaw: uint64ToString(data.BaseAmountBurned),
			Amount:    types.ConvertToUIAmountUint64(data.BaseAmountBurned, baseDecimals),
			Decimals:  baseDecimals,
		}
		memeEvent.IxName = "boost_buy_and_burn"
		memeEvent.RealBaseReserves = uint64ToString(data.BaseReservesAfter)
		memeEvent.RealQuoteReserves = uint64ToString(data.RealQuoteReservesAfter)
		if data.VirtualQuoteReserves != nil {
			memeEvent.VirtualQuoteReserves = data.VirtualQuoteReserves.String()
		}
	default:
		return nil
	}

	return memeEvent
}

// tradeMints returns the base and quote mints of a trade from the user's
// token accounts (quote defaults to SOL)
func (p *PumpswapEventParser) tradeMints(userBaseTokenAccount, userQuoteTokenAccount string) (string, string) {
	baseMint := p.adapter.GetSplTokenMint(userBaseTokenAccount)
	quoteMint := p.adapter.GetSplTokenMint(userQuoteTokenAccount)
	if quoteMint == "" {
		quoteMint = constants.TOKENS.SOL
	}
	return baseMint, quoteMint
}

// setTradeFeeFields sets the fee fields of a PumpSwap trade meme event
func (p *PumpswapEventParser) setTradeFeeFields(event *types.MemeEvent, fees []types.FeeInfo, protocolFee, coinCreatorFee uint64, decimals uint8) {
	event.Fees = fees
	event.ProtocolFee = uiPtr(u64(protocolFee), decimals)
	if coinCreatorFee > 0 {
		event.CreatorFee = uiPtr(u64(coinCreatorFee), decimals)
	}
}

// pumpswapFeeParams holds the fee fields of a Buy/SellEvent
type pumpswapFeeParams struct {
	protocolFee, buybackFee, coinCreatorFee, cashback uint64
	protocolRecipient, coinCreator, user              string
}

// pumpswapFees lists the fee components of a PumpSwap trade in the quote mint:
// protocol (protocol_fee without its buyback part, paid to the protocol fee
// recipient), buyback, coin creator and cashback. The LP fee stays in the pool
// and is part of the price, so it is not listed.
func pumpswapFees(f pumpswapFeeParams, mint string, decimals uint8) []types.FeeInfo {
	dex := constants.DEX_PROGRAMS.PUMP_SWAP.Name
	protocol := f.protocolFee
	if f.buybackFee <= protocol {
		protocol -= f.buybackFee
	}
	fees := []types.FeeInfo{feeInfo(mint, u64(protocol), decimals, dex, "protocol", f.protocolRecipient)}
	if f.buybackFee > 0 && f.buybackFee <= f.protocolFee {
		fees = append(fees, feeInfo(mint, u64(f.buybackFee), decimals, dex, "buyback", ""))
	}
	if f.coinCreatorFee > 0 {
		fees = append(fees, feeInfo(mint, u64(f.coinCreatorFee), decimals, dex, "coinCreator", f.coinCreator))
	}
	if f.cashback > 0 {
		fees = append(fees, feeInfo(mint, u64(f.cashback), decimals, dex, "cashback", f.user))
	}
	return fees
}

// ParseInstructions parses classified instructions into Pumpswap events. A
// BuyEvent emitted by boost_buy_and_burn is the protocol buying for a burn,
// not a user trade: it is dropped, and the BoostBuyAndBurnEvent becomes a
// BUY_AND_BURN event.
func (p *PumpswapEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*PumpswapEvent {
	var events []*PumpswapEvent

	ordered := executionOrder(instructions)
	for pos, ci := range ordered {
		if ci.ProgramId != constants.DEX_PROGRAMS.PUMP_SWAP.ID {
			continue
		}

		data := p.adapter.GetInstructionData(ci.Instruction)
		if !isEventData(data) {
			continue
		}

		disc := data[:16]
		var event *PumpswapEvent

		// Check event discriminators
		if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPSWAP.CREATE_POOL_EVENT) {
			eventData := p.decodeCreateEvent(data[16:])
			if eventData != nil {
				event = &PumpswapEvent{Type: "CREATE", Data: eventData}
			}
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPSWAP.ADD_LIQUIDITY_EVENT) {
			eventData := p.decodeAddLiquidity(data[16:])
			if eventData != nil {
				event = &PumpswapEvent{Type: "ADD", Data: eventData}
			}
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPSWAP.REMOVE_LIQUIDITY_EVENT) {
			eventData := p.decodeRemoveLiquidity(data[16:])
			if eventData != nil {
				event = &PumpswapEvent{Type: "REMOVE", Data: eventData}
			}
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPSWAP.BUY_EVENT) {
			eventData := p.decodeBuyEvent(data[16:])
			if eventData != nil {
				parentDisc := p.parentDiscriminator(ordered, pos)
				if bytes.Equal(parentDisc, constants.DISCRIMINATORS.PUMPSWAP.BOOST_BUY_AND_BURN) {
					continue
				}
				eventData.ExactQuoteIn = eventData.IxName == "buy_exact_quote_in" ||
					(eventData.IxName == "" && bytes.Equal(parentDisc, constants.DISCRIMINATORS.PUMPSWAP.BUY_EXACT_QUOTE_IN))
				event = &PumpswapEvent{Type: "BUY", Data: eventData}
			}
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPSWAP.SELL_EVENT) {
			eventData := p.decodeSellEvent(data[16:])
			if eventData != nil {
				event = &PumpswapEvent{Type: "SELL", Data: eventData}
			}
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPSWAP.BOOST_BUY_AND_BURN_EVENT) {
			eventData := p.decodeBoostBuyAndBurnEvent(data[16:])
			if eventData != nil {
				parent := findParentInstruction(p.adapter, ordered, pos, constants.DEX_PROGRAMS.PUMP_SWAP.ID,
					func(data []byte, accounts []string) bool {
						return len(data) >= 8 && bytes.Equal(data[:8], constants.DISCRIMINATORS.PUMPSWAP.BOOST_BUY_AND_BURN)
					})
				if parent != nil {
					if accounts := p.adapter.GetInstructionAccounts(parent.Instruction); len(accounts) > 4 {
						eventData.QuoteMint = accounts[4]
					}
				}
				event = &PumpswapEvent{Type: string(types.TradeTypeBuyAndBurn), Data: eventData}
			}
		}

		if event != nil {
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Signature = p.adapter.Signature()
			event.Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
			events = append(events, event)
		}
	}

	// ordered is in execution order already
	return events
}

// parentDiscriminator returns the 8-byte discriminator of the PumpSwap
// instruction that emitted the event at position pos of ordered, or nil
func (p *PumpswapEventParser) parentDiscriminator(ordered []types.ClassifiedInstruction, pos int) []byte {
	parent := findParentInstruction(p.adapter, ordered, pos, constants.DEX_PROGRAMS.PUMP_SWAP.ID,
		func(data []byte, accounts []string) bool { return len(data) >= 8 })
	if parent == nil {
		return nil
	}
	return p.adapter.GetInstructionData(parent.Instruction)[:8]
}

// decodeBuyEvent decodes a BuyEvent in IDL order (pump_amm.json). Events
// before the coin-creator fee end after protocol_fee_recipient_token_account;
// later fields are read only while the data lasts.
func (p *PumpswapEventParser) decodeBuyEvent(data []byte) *PumpswapBuyEventData {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	evt := &PumpswapBuyEventData{}
	evt.Timestamp, _ = reader.ReadI64()
	evt.BaseAmountOut, _ = reader.ReadU64()
	evt.MaxQuoteAmountIn, _ = reader.ReadU64()
	evt.UserBaseTokenReserves, _ = reader.ReadU64()
	evt.UserQuoteTokenReserves, _ = reader.ReadU64()
	evt.PoolBaseTokenReserves, _ = reader.ReadU64()
	evt.PoolQuoteTokenReserves, _ = reader.ReadU64()
	evt.QuoteAmountIn, _ = reader.ReadU64()
	evt.LpFeeBasisPoints, _ = reader.ReadU64()
	evt.LpFee, _ = reader.ReadU64()
	evt.ProtocolFeeBasisPoints, _ = reader.ReadU64()
	evt.ProtocolFee, _ = reader.ReadU64()
	evt.QuoteAmountInWithLpFee, _ = reader.ReadU64()
	evt.UserQuoteAmountIn, _ = reader.ReadU64()
	evt.Pool, _ = reader.ReadPubkey()
	evt.User, _ = reader.ReadPubkey()
	evt.UserBaseTokenAccount, _ = reader.ReadPubkey()
	evt.UserQuoteTokenAccount, _ = reader.ReadPubkey()
	evt.ProtocolFeeRecipient, _ = reader.ReadPubkey()
	evt.ProtocolFeeRecipientTokenAccount, _ = reader.ReadPubkey()
	if reader.HasError() {
		return nil
	}

	t := newTailReader(reader)
	evt.CoinCreator = t.pubkey()
	if evt.CoinCreator == "" {
		evt.CoinCreator = defaultPubkey
	}
	evt.CoinCreatorFeeBasisPoints = t.u64()
	evt.CoinCreatorFee = t.u64()
	t.boolean() // track_volume
	t.u64()     // total_unclaimed_tokens
	t.u64()     // total_claimed_tokens
	t.u64()     // current_sol_volume
	t.u64()     // last_update_timestamp
	evt.MinBaseAmountOut = t.u64()
	evt.IxName = t.str()
	readPumpswapTradeTail(t, &evt.CashbackFeeBps, &evt.Cashback, &evt.BuybackFeeBps, &evt.BuybackFee,
		&evt.VirtualQuoteReserves, &evt.CanBoost, &evt.BaseSupply, &evt.HolderRewardsBps, &evt.HolderRewards)
	return evt
}

// readPumpswapTradeTail reads the fields both BuyEvent and SellEvent end with:
// cashback (16 bytes), buyback (16), virtual_quote_reserves i128, can_boost
// and base_supply (25), holder rewards (16)
func readPumpswapTradeTail(t *tailReader, cashbackFeeBps, cashback, buybackFeeBps, buybackFee *uint64,
	virtualQuoteReserves **big.Int, canBoost *bool, baseSupply, holderRewardsBps, holderRewards *uint64) {
	*cashbackFeeBps = t.u64()
	*cashback = t.u64()
	*buybackFeeBps = t.u64()
	*buybackFee = t.u64()
	*virtualQuoteReserves = t.i128()
	*canBoost = t.boolean()
	*baseSupply = t.u64()
	*holderRewardsBps = t.u64()
	*holderRewards = t.u64()
}

// decodeSellEvent decodes a SellEvent in IDL order (pump_amm.json)
func (p *PumpswapEventParser) decodeSellEvent(data []byte) *PumpswapSellEventData {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	evt := &PumpswapSellEventData{}
	evt.Timestamp, _ = reader.ReadI64()
	evt.BaseAmountIn, _ = reader.ReadU64()
	evt.MinQuoteAmountOut, _ = reader.ReadU64()
	evt.UserBaseTokenReserves, _ = reader.ReadU64()
	evt.UserQuoteTokenReserves, _ = reader.ReadU64()
	evt.PoolBaseTokenReserves, _ = reader.ReadU64()
	evt.PoolQuoteTokenReserves, _ = reader.ReadU64()
	evt.QuoteAmountOut, _ = reader.ReadU64()
	evt.LpFeeBasisPoints, _ = reader.ReadU64()
	evt.LpFee, _ = reader.ReadU64()
	evt.ProtocolFeeBasisPoints, _ = reader.ReadU64()
	evt.ProtocolFee, _ = reader.ReadU64()
	evt.QuoteAmountOutWithoutLpFee, _ = reader.ReadU64()
	evt.UserQuoteAmountOut, _ = reader.ReadU64()
	evt.Pool, _ = reader.ReadPubkey()
	evt.User, _ = reader.ReadPubkey()
	evt.UserBaseTokenAccount, _ = reader.ReadPubkey()
	evt.UserQuoteTokenAccount, _ = reader.ReadPubkey()
	evt.ProtocolFeeRecipient, _ = reader.ReadPubkey()
	evt.ProtocolFeeRecipientTokenAccount, _ = reader.ReadPubkey()
	if reader.HasError() {
		return nil
	}

	t := newTailReader(reader)
	evt.CoinCreator = t.pubkey()
	if evt.CoinCreator == "" {
		evt.CoinCreator = defaultPubkey
	}
	evt.CoinCreatorFeeBasisPoints = t.u64()
	evt.CoinCreatorFee = t.u64()
	readPumpswapTradeTail(t, &evt.CashbackFeeBps, &evt.Cashback, &evt.BuybackFeeBps, &evt.BuybackFee,
		&evt.VirtualQuoteReserves, &evt.CanBoost, &evt.BaseSupply, &evt.HolderRewardsBps, &evt.HolderRewards)
	return evt
}

// decodeBoostBuyAndBurnEvent decodes a BoostBuyAndBurnEvent (pump_amm.json)
func (p *PumpswapEventParser) decodeBoostBuyAndBurnEvent(data []byte) *PumpswapBoostBuyAndBurnEventData {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	evt := &PumpswapBoostBuyAndBurnEventData{}
	evt.Timestamp, _ = reader.ReadI64()
	evt.Mint, _ = reader.ReadPubkey()
	evt.BondingCurve, _ = reader.ReadPubkey()
	evt.Pool, _ = reader.ReadPubkey()
	evt.Authority, _ = reader.ReadPubkey()
	evt.QuoteAmountInRequested, _ = reader.ReadU64()
	evt.QuoteAmountInUsed, _ = reader.ReadU64()
	evt.BaseAmountBurned, _ = reader.ReadU64()
	if reader.HasError() {
		return nil
	}
	t := newTailReader(reader)
	evt.VirtualQuoteReserves = t.i128()
	evt.RealQuoteReservesAfter = t.u64()
	evt.BaseReservesAfter = t.u64()
	evt.BoostVaultRemaining = t.u64()
	return evt
}

// decodeAddLiquidity decodes an add liquidity event
func (p *PumpswapEventParser) decodeAddLiquidity(data []byte) *PumpswapDepositEventData {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	timestamp, _ := reader.ReadI64()
	lpTokenAmountOut, _ := reader.ReadU64()
	maxBaseAmountIn, _ := reader.ReadU64()
	maxQuoteAmountIn, _ := reader.ReadU64()
	userBaseTokenReserves, _ := reader.ReadU64()
	userQuoteTokenReserves, _ := reader.ReadU64()
	poolBaseTokenReserves, _ := reader.ReadU64()
	poolQuoteTokenReserves, _ := reader.ReadU64()
	baseAmountIn, _ := reader.ReadU64()
	quoteAmountIn, _ := reader.ReadU64()
	lpMintSupply, _ := reader.ReadU64()
	pool, _ := reader.ReadPubkey()
	user, _ := reader.ReadPubkey()
	userBaseTokenAccount, _ := reader.ReadPubkey()
	userQuoteTokenAccount, _ := reader.ReadPubkey()
	userPoolTokenAccount, _ := reader.ReadPubkey()

	if reader.HasError() {
		return nil
	}

	return &PumpswapDepositEventData{
		Timestamp:              timestamp,
		LpTokenAmountOut:       lpTokenAmountOut,
		MaxBaseAmountIn:        maxBaseAmountIn,
		MaxQuoteAmountIn:       maxQuoteAmountIn,
		UserBaseTokenReserves:  userBaseTokenReserves,
		UserQuoteTokenReserves: userQuoteTokenReserves,
		PoolBaseTokenReserves:  poolBaseTokenReserves,
		PoolQuoteTokenReserves: poolQuoteTokenReserves,
		BaseAmountIn:           baseAmountIn,
		QuoteAmountIn:          quoteAmountIn,
		LpMintSupply:           lpMintSupply,
		Pool:                   pool,
		User:                   user,
		UserBaseTokenAccount:   userBaseTokenAccount,
		UserQuoteTokenAccount:  userQuoteTokenAccount,
		UserPoolTokenAccount:   userPoolTokenAccount,
	}
}

// decodeCreateEvent decodes a create pool event
func (p *PumpswapEventParser) decodeCreateEvent(data []byte) *PumpswapCreatePoolEventData {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	timestamp, _ := reader.ReadI64()
	index, _ := reader.ReadU16()
	creator, _ := reader.ReadPubkey()
	baseMint, _ := reader.ReadPubkey()
	quoteMint, _ := reader.ReadPubkey()
	baseMintDecimals, _ := reader.ReadU8()
	quoteMintDecimals, _ := reader.ReadU8()
	baseAmountIn, _ := reader.ReadU64()
	quoteAmountIn, _ := reader.ReadU64()
	poolBaseAmount, _ := reader.ReadU64()
	poolQuoteAmount, _ := reader.ReadU64()
	minimumLiquidity, _ := reader.ReadU64()
	initialLiquidity, _ := reader.ReadU64()
	lpTokenAmountOut, _ := reader.ReadU64()
	poolBump, _ := reader.ReadU8()
	pool, _ := reader.ReadPubkey()
	lpMint, _ := reader.ReadPubkey()
	userBaseTokenAccount, _ := reader.ReadPubkey()
	userQuoteTokenAccount, _ := reader.ReadPubkey()

	if reader.HasError() {
		return nil
	}

	// Fields appended by later program versions
	t := newTailReader(reader)
	coinCreator := t.pubkey()
	isMayhemMode := t.boolean()
	creatorFeeBps := t.u64()
	var creatorFeeBpsPtr *uint64
	if t.ok {
		creatorFeeBpsPtr = &creatorFeeBps
	}
	canEditCreatorFee := t.boolean()
	isHolderReward := t.boolean()

	return &PumpswapCreatePoolEventData{
		CoinCreator:           coinCreator,
		IsMayhemMode:          isMayhemMode,
		CreatorFeeBps:         creatorFeeBpsPtr,
		CanEditCreatorFee:     canEditCreatorFee,
		IsHolderReward:        isHolderReward,
		Timestamp:             timestamp,
		Index:                 index,
		Creator:               creator,
		BaseMint:              baseMint,
		QuoteMint:             quoteMint,
		BaseMintDecimals:      baseMintDecimals,
		QuoteMintDecimals:     quoteMintDecimals,
		BaseAmountIn:          baseAmountIn,
		QuoteAmountIn:         quoteAmountIn,
		PoolBaseAmount:        poolBaseAmount,
		PoolQuoteAmount:       poolQuoteAmount,
		MinimumLiquidity:      minimumLiquidity,
		InitialLiquidity:      initialLiquidity,
		LpTokenAmountOut:      lpTokenAmountOut,
		PoolBump:              poolBump,
		Pool:                  pool,
		LpMint:                lpMint,
		UserBaseTokenAccount:  userBaseTokenAccount,
		UserQuoteTokenAccount: userQuoteTokenAccount,
	}
}

// decodeRemoveLiquidity decodes a remove liquidity event
func (p *PumpswapEventParser) decodeRemoveLiquidity(data []byte) *PumpswapWithdrawEventData {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	timestamp, _ := reader.ReadI64()
	lpTokenAmountIn, _ := reader.ReadU64()
	minBaseAmountOut, _ := reader.ReadU64()
	minQuoteAmountOut, _ := reader.ReadU64()
	userBaseTokenReserves, _ := reader.ReadU64()
	userQuoteTokenReserves, _ := reader.ReadU64()
	poolBaseTokenReserves, _ := reader.ReadU64()
	poolQuoteTokenReserves, _ := reader.ReadU64()
	baseAmountOut, _ := reader.ReadU64()
	quoteAmountOut, _ := reader.ReadU64()
	lpMintSupply, _ := reader.ReadU64()
	pool, _ := reader.ReadPubkey()
	user, _ := reader.ReadPubkey()
	userBaseTokenAccount, _ := reader.ReadPubkey()
	userQuoteTokenAccount, _ := reader.ReadPubkey()
	userPoolTokenAccount, _ := reader.ReadPubkey()

	if reader.HasError() {
		return nil
	}

	return &PumpswapWithdrawEventData{
		Timestamp:              timestamp,
		LpTokenAmountIn:        lpTokenAmountIn,
		MinBaseAmountOut:       minBaseAmountOut,
		MinQuoteAmountOut:      minQuoteAmountOut,
		UserBaseTokenReserves:  userBaseTokenReserves,
		UserQuoteTokenReserves: userQuoteTokenReserves,
		PoolBaseTokenReserves:  poolBaseTokenReserves,
		PoolQuoteTokenReserves: poolQuoteTokenReserves,
		BaseAmountOut:          baseAmountOut,
		QuoteAmountOut:         quoteAmountOut,
		LpMintSupply:           lpMintSupply,
		Pool:                   pool,
		User:                   user,
		UserBaseTokenAccount:   userBaseTokenAccount,
		UserQuoteTokenAccount:  userQuoteTokenAccount,
		UserPoolTokenAccount:   userPoolTokenAccount,
	}
}
