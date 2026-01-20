package pumpfun

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/types"
	"github.com/solana-dex-parser-go/utils"
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
}

// PumpswapDepositEventData contains deposit event data
type PumpswapDepositEventData struct {
	Timestamp             int64
	LpTokenAmountOut      uint64
	MaxBaseAmountIn       uint64
	MaxQuoteAmountIn      uint64
	UserBaseTokenReserves uint64
	UserQuoteTokenReserves uint64
	PoolBaseTokenReserves uint64
	PoolQuoteTokenReserves uint64
	BaseAmountIn          uint64
	QuoteAmountIn         uint64
	LpMintSupply          uint64
	Pool                  string
	User                  string
	UserBaseTokenAccount  string
	UserQuoteTokenAccount string
	UserPoolTokenAccount  string
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

// convertToMemeEvent converts PumpswapEvent to MemeEvent
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
		memeEvent.Type = types.TradeTypeBuy
		memeEvent.User = data.User
		memeEvent.Pool = data.Pool
		baseMint := p.adapter.GetSplTokenMint(data.UserBaseTokenAccount)
		quoteMint := p.adapter.GetSplTokenMint(data.UserQuoteTokenAccount)
		if quoteMint == "" {
			quoteMint = constants.TOKENS.SOL
		}
		memeEvent.BaseMint = baseMint
		memeEvent.QuoteMint = quoteMint
		memeEvent.InputToken = &types.TokenInfo{
			Mint:      quoteMint,
			AmountRaw: fmt.Sprintf("%d", data.QuoteAmountIn),
			Amount:    types.ConvertToUIAmountUint64(data.QuoteAmountIn, 9),
			Decimals:  9,
		}
		memeEvent.OutputToken = &types.TokenInfo{
			Mint:      baseMint,
			AmountRaw: fmt.Sprintf("%d", data.BaseAmountOut),
			Amount:    types.ConvertToUIAmountUint64(data.BaseAmountOut, 6),
			Decimals:  6,
		}
		fee := types.ConvertToUIAmountUint64(data.ProtocolFee, 9)
		memeEvent.ProtocolFee = &fee
	case *PumpswapSellEventData:
		memeEvent.Type = types.TradeTypeSell
		memeEvent.User = data.User
		memeEvent.Pool = data.Pool
		baseMint := p.adapter.GetSplTokenMint(data.UserBaseTokenAccount)
		quoteMint := p.adapter.GetSplTokenMint(data.UserQuoteTokenAccount)
		if quoteMint == "" {
			quoteMint = constants.TOKENS.SOL
		}
		memeEvent.BaseMint = baseMint
		memeEvent.QuoteMint = quoteMint
		memeEvent.InputToken = &types.TokenInfo{
			Mint:      baseMint,
			AmountRaw: fmt.Sprintf("%d", data.BaseAmountIn),
			Amount:    types.ConvertToUIAmountUint64(data.BaseAmountIn, 6),
			Decimals:  6,
		}
		memeEvent.OutputToken = &types.TokenInfo{
			Mint:      quoteMint,
			AmountRaw: fmt.Sprintf("%d", data.QuoteAmountOut),
			Amount:    types.ConvertToUIAmountUint64(data.QuoteAmountOut, 9),
			Decimals:  9,
		}
		fee := types.ConvertToUIAmountUint64(data.ProtocolFee, 9)
		memeEvent.ProtocolFee = &fee
	case *PumpswapCreatePoolEventData:
		memeEvent.Type = types.TradeTypeCreate
		memeEvent.User = data.Creator
		memeEvent.Pool = data.Pool
		memeEvent.BaseMint = data.BaseMint
		memeEvent.QuoteMint = data.QuoteMint
		memeEvent.Creator = data.Creator
	default:
		return nil
	}

	return memeEvent
}

// ParseInstructions parses classified instructions into Pumpswap events
func (p *PumpswapEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*PumpswapEvent {
	var events []*PumpswapEvent

	for _, ci := range instructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.PUMP_SWAP.ID {
			continue
		}

		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 16 {
			continue
		}

		disc := data[:16]
		innerIdx := ci.InnerIndex
		if innerIdx < 0 {
			innerIdx = 0
		}

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
				event = &PumpswapEvent{Type: "BUY", Data: eventData}
			}
		} else if bytes.Equal(disc, constants.DISCRIMINATORS.PUMPSWAP.SELL_EVENT) {
			eventData := p.decodeSellEvent(data[16:])
			if eventData != nil {
				event = &PumpswapEvent{Type: "SELL", Data: eventData}
			}
		}

		if event != nil {
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Signature = p.adapter.Signature()
			event.Idx = fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx)
			events = append(events, event)
		}
	}

	// Sort by Idx
	sort.Slice(events, func(i, j int) bool {
		return events[i].Idx < events[j].Idx
	})

	return events
}

// decodeBuyEvent decodes a buy event
func (p *PumpswapEventParser) decodeBuyEvent(data []byte) *PumpswapBuyEventData {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	timestamp, _ := reader.ReadI64()
	baseAmountOut, _ := reader.ReadU64()
	maxQuoteAmountIn, _ := reader.ReadU64()
	userBaseTokenReserves, _ := reader.ReadU64()
	userQuoteTokenReserves, _ := reader.ReadU64()
	poolBaseTokenReserves, _ := reader.ReadU64()
	poolQuoteTokenReserves, _ := reader.ReadU64()
	quoteAmountIn, _ := reader.ReadU64()
	lpFeeBasisPoints, _ := reader.ReadU64()
	lpFee, _ := reader.ReadU64()
	protocolFeeBasisPoints, _ := reader.ReadU64()
	protocolFee, _ := reader.ReadU64()
	quoteAmountInWithLpFee, _ := reader.ReadU64()
	userQuoteAmountIn, _ := reader.ReadU64()
	pool, _ := reader.ReadPubkey()
	user, _ := reader.ReadPubkey()
	userBaseTokenAccount, _ := reader.ReadPubkey()
	userQuoteTokenAccount, _ := reader.ReadPubkey()
	protocolFeeRecipient, _ := reader.ReadPubkey()
	protocolFeeRecipientTokenAccount, _ := reader.ReadPubkey()

	if reader.HasError() {
		return nil
	}

	evt := &PumpswapBuyEventData{
		Timestamp:                        timestamp,
		BaseAmountOut:                    baseAmountOut,
		MaxQuoteAmountIn:                 maxQuoteAmountIn,
		UserBaseTokenReserves:            userBaseTokenReserves,
		UserQuoteTokenReserves:           userQuoteTokenReserves,
		PoolBaseTokenReserves:            poolBaseTokenReserves,
		PoolQuoteTokenReserves:           poolQuoteTokenReserves,
		QuoteAmountIn:                    quoteAmountIn,
		LpFeeBasisPoints:                 lpFeeBasisPoints,
		LpFee:                            lpFee,
		ProtocolFeeBasisPoints:           protocolFeeBasisPoints,
		ProtocolFee:                      protocolFee,
		QuoteAmountInWithLpFee:           quoteAmountInWithLpFee,
		UserQuoteAmountIn:                userQuoteAmountIn,
		Pool:                             pool,
		User:                             user,
		UserBaseTokenAccount:             userBaseTokenAccount,
		UserQuoteTokenAccount:            userQuoteTokenAccount,
		ProtocolFeeRecipient:             protocolFeeRecipient,
		ProtocolFeeRecipientTokenAccount: protocolFeeRecipientTokenAccount,
		CoinCreator:                      "11111111111111111111111111111111",
	}

	// Extended fields if available
	if len(data) > 304 {
		coinCreator, _ := reader.ReadPubkey()
		coinCreatorFeeBasisPoints, _ := reader.ReadU64()
		coinCreatorFee, _ := reader.ReadU64()
		evt.CoinCreator = coinCreator
		evt.CoinCreatorFeeBasisPoints = coinCreatorFeeBasisPoints
		evt.CoinCreatorFee = coinCreatorFee
	}

	return evt
}

// decodeSellEvent decodes a sell event
func (p *PumpswapEventParser) decodeSellEvent(data []byte) *PumpswapSellEventData {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	timestamp, _ := reader.ReadI64()
	baseAmountIn, _ := reader.ReadU64()
	minQuoteAmountOut, _ := reader.ReadU64()
	userBaseTokenReserves, _ := reader.ReadU64()
	userQuoteTokenReserves, _ := reader.ReadU64()
	poolBaseTokenReserves, _ := reader.ReadU64()
	poolQuoteTokenReserves, _ := reader.ReadU64()
	quoteAmountOut, _ := reader.ReadU64()
	lpFeeBasisPoints, _ := reader.ReadU64()
	lpFee, _ := reader.ReadU64()
	protocolFeeBasisPoints, _ := reader.ReadU64()
	protocolFee, _ := reader.ReadU64()
	quoteAmountOutWithoutLpFee, _ := reader.ReadU64()
	userQuoteAmountOut, _ := reader.ReadU64()
	pool, _ := reader.ReadPubkey()
	user, _ := reader.ReadPubkey()
	userBaseTokenAccount, _ := reader.ReadPubkey()
	userQuoteTokenAccount, _ := reader.ReadPubkey()
	protocolFeeRecipient, _ := reader.ReadPubkey()
	protocolFeeRecipientTokenAccount, _ := reader.ReadPubkey()

	if reader.HasError() {
		return nil
	}

	evt := &PumpswapSellEventData{
		Timestamp:                        timestamp,
		BaseAmountIn:                     baseAmountIn,
		MinQuoteAmountOut:                minQuoteAmountOut,
		UserBaseTokenReserves:            userBaseTokenReserves,
		UserQuoteTokenReserves:           userQuoteTokenReserves,
		PoolBaseTokenReserves:            poolBaseTokenReserves,
		PoolQuoteTokenReserves:           poolQuoteTokenReserves,
		QuoteAmountOut:                   quoteAmountOut,
		LpFeeBasisPoints:                 lpFeeBasisPoints,
		LpFee:                            lpFee,
		ProtocolFeeBasisPoints:           protocolFeeBasisPoints,
		ProtocolFee:                      protocolFee,
		QuoteAmountOutWithoutLpFee:       quoteAmountOutWithoutLpFee,
		UserQuoteAmountOut:               userQuoteAmountOut,
		Pool:                             pool,
		User:                             user,
		UserBaseTokenAccount:             userBaseTokenAccount,
		UserQuoteTokenAccount:            userQuoteTokenAccount,
		ProtocolFeeRecipient:             protocolFeeRecipient,
		ProtocolFeeRecipientTokenAccount: protocolFeeRecipientTokenAccount,
		CoinCreator:                      "11111111111111111111111111111111",
	}

	// Extended fields if available
	if len(data) > 304 {
		coinCreator, _ := reader.ReadPubkey()
		coinCreatorFeeBasisPoints, _ := reader.ReadU64()
		coinCreatorFee, _ := reader.ReadU64()
		evt.CoinCreator = coinCreator
		evt.CoinCreatorFeeBasisPoints = coinCreatorFeeBasisPoints
		evt.CoinCreatorFee = coinCreatorFee
	}

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

	return &PumpswapCreatePoolEventData{
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
