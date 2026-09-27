package raydium

import (
	"errors"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
	"github.com/mr-tron/base58"
)

// TradeDirection represents the direction of a trade
type TradeDirection uint8

const (
	TradeDirectionBuy  TradeDirection = 0
	TradeDirectionSell TradeDirection = 1
)

// PoolStatus represents the status of a pool
type PoolStatus uint8

// CurveType represents the type of bonding curve
type CurveType uint8

const (
	CurveTypeConstant CurveType = 0
	CurveTypeFixed    CurveType = 1
	CurveTypeLinear   CurveType = 2
)

// MintParams represents token mint parameters
type MintParams struct {
	Decimals uint8
	Name     string
	Symbol   string
	URI      string
}

// ConstantCurve represents constant curve parameters
type ConstantCurve struct {
	Supply                *big.Int
	TotalBaseSell         *big.Int
	TotalQuoteFundRaising *big.Int
	MigrateType           uint8
}

// FixedCurve represents fixed curve parameters
type FixedCurve struct {
	Supply                *big.Int
	TotalQuoteFundRaising *big.Int
	MigrateType           uint8
}

// LinearCurve represents linear curve parameters
type LinearCurve struct {
	Supply                *big.Int
	TotalQuoteFundRaising *big.Int
	MigrateType           uint8
}

// CurveParams represents curve parameters (variant)
type CurveParams struct {
	Variant string
	Data    interface{}
}

// VestingParams represents vesting parameters
type VestingParams struct {
	TotalLockedAmount *big.Int
	CliffPeriod       *big.Int
	UnlockPeriod      *big.Int
}

// RaydiumLCPCreateEvent represents a pool creation event
type RaydiumLCPCreateEvent struct {
	PoolState     string
	Creator       string
	Config        string
	BaseMintParam MintParams
	CurveParam    CurveParams
	VestingParam  VestingParams
	AmmFeeOn      *uint8 // 0: QuoteToken, 1: BothToken; nil in older events
	BaseMint      string
	QuoteMint     string
}

// RaydiumLCPTradeEvent represents a trade event
type RaydiumLCPTradeEvent struct {
	PoolState       string
	TotalBaseSell   *big.Int
	VirtualBase     *big.Int
	VirtualQuote    *big.Int
	RealBaseBefore  *big.Int
	RealQuoteBefore *big.Int
	RealBaseAfter   *big.Int
	RealQuoteAfter  *big.Int
	AmountIn        *big.Int
	AmountOut       *big.Int
	ProtocolFee     *big.Int
	PlatformFee     *big.Int
	CreatorFee      *big.Int
	ShareFee        *big.Int
	TradeDirection  TradeDirection
	PoolStatus      PoolStatus
	BaseMint        string
	QuoteMint       string
	User            string
}

// PoolCreateEventLayout parses pool creation event data
type PoolCreateEventLayout struct {
	PoolState     []byte
	Creator       []byte
	Config        []byte
	BaseMintParam MintParams
	CurveParam    CurveParams
	VestingParam  VestingParams
	AmmFeeOn      *uint8 // 0: QuoteToken, 1: BothToken; nil in older events
}

// ParsePoolCreateEventLayout parses pool creation event from bytes
func ParsePoolCreateEventLayout(data []byte) (*PoolCreateEventLayout, error) {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	poolState, _ := reader.ReadFixedArray(32)
	creator, _ := reader.ReadFixedArray(32)
	config, _ := reader.ReadFixedArray(32)

	// Read baseMintParam
	decimals, _ := reader.ReadU8()
	name, _ := reader.ReadString()
	symbol, _ := reader.ReadString()
	uri, _ := reader.ReadString()

	baseMintParam := MintParams{
		Decimals: decimals,
		Name:     name,
		Symbol:   symbol,
		URI:      uri,
	}

	// Read curveParam: an enum tag, then the variant's fields in IDL order
	// (ConstantCurve: supply, total_base_sell, total_quote_fund_raising,
	// migrate_type; FixedCurve and LinearCurve: supply,
	// total_quote_fund_raising, migrate_type)
	variant, _ := reader.ReadU8()
	var curveParam CurveParams

	switch CurveType(variant) {
	case CurveTypeConstant:
		supply := reader.ReadU64AsBigInt()
		totalBaseSell := reader.ReadU64AsBigInt()
		totalQuoteFundRaising := reader.ReadU64AsBigInt()
		migrateType, _ := reader.ReadU8()
		curveParam = CurveParams{
			Variant: "Constant",
			Data: ConstantCurve{
				Supply:                supply,
				TotalBaseSell:         totalBaseSell,
				TotalQuoteFundRaising: totalQuoteFundRaising,
				MigrateType:           migrateType,
			},
		}
	case CurveTypeFixed:
		supply := reader.ReadU64AsBigInt()
		totalQuoteFundRaising := reader.ReadU64AsBigInt()
		migrateType, _ := reader.ReadU8()
		curveParam = CurveParams{
			Variant: "Fixed",
			Data: FixedCurve{
				Supply:                supply,
				TotalQuoteFundRaising: totalQuoteFundRaising,
				MigrateType:           migrateType,
			},
		}
	case CurveTypeLinear:
		supply := reader.ReadU64AsBigInt()
		totalQuoteFundRaising := reader.ReadU64AsBigInt()
		migrateType, _ := reader.ReadU8()
		curveParam = CurveParams{
			Variant: "Linear",
			Data: LinearCurve{
				Supply:                supply,
				TotalQuoteFundRaising: totalQuoteFundRaising,
				MigrateType:           migrateType,
			},
		}
	default:
		return nil, errUnknownCurveType
	}

	// Read vestingParam
	vestingParam := VestingParams{
		TotalLockedAmount: reader.ReadU64AsBigInt(),
		CliffPeriod:       reader.ReadU64AsBigInt(),
		UnlockPeriod:      reader.ReadU64AsBigInt(),
	}

	if reader.HasError() {
		return nil, reader.Error()
	}

	// amm_fee_on (AmmCreatorFeeOn enum) was appended later
	var ammFeeOn *uint8
	if reader.Remaining() >= 1 {
		v, _ := reader.ReadU8()
		ammFeeOn = &v
	}

	return &PoolCreateEventLayout{
		PoolState:     poolState,
		Creator:       creator,
		Config:        config,
		BaseMintParam: baseMintParam,
		CurveParam:    curveParam,
		VestingParam:  vestingParam,
		AmmFeeOn:      ammFeeOn,
	}, nil
}

// errUnknownCurveType is returned for a CurveParams tag the IDL does not define
var errUnknownCurveType = errors.New("unknown LaunchLab curve type")

// ToObject converts layout to RaydiumLCPCreateEvent
func (l *PoolCreateEventLayout) ToObject() *RaydiumLCPCreateEvent {
	return &RaydiumLCPCreateEvent{
		PoolState:     base58.Encode(l.PoolState),
		Creator:       base58.Encode(l.Creator),
		Config:        base58.Encode(l.Config),
		BaseMintParam: l.BaseMintParam,
		CurveParam:    l.CurveParam,
		VestingParam:  l.VestingParam,
		AmmFeeOn:      l.AmmFeeOn,
		BaseMint:      "",
		QuoteMint:     "",
	}
}

// memeCurveParams converts the curve and vesting parameters for MemeEvent
func (e *RaydiumLCPCreateEvent) memeCurveParams() *types.MemeCurveParams {
	str := func(v *big.Int) string {
		if v == nil {
			return ""
		}
		return v.String()
	}
	c := &types.MemeCurveParams{
		Type:              e.CurveParam.Variant,
		TotalLockedAmount: str(e.VestingParam.TotalLockedAmount),
		CliffPeriod:       str(e.VestingParam.CliffPeriod),
		UnlockPeriod:      str(e.VestingParam.UnlockPeriod),
	}
	switch d := e.CurveParam.Data.(type) {
	case ConstantCurve:
		c.Supply, c.TotalBaseSell, c.TotalQuoteFundRaising, c.MigrateType = str(d.Supply), str(d.TotalBaseSell), str(d.TotalQuoteFundRaising), d.MigrateType
	case FixedCurve:
		c.Supply, c.TotalQuoteFundRaising, c.MigrateType = str(d.Supply), str(d.TotalQuoteFundRaising), d.MigrateType
	case LinearCurve:
		c.Supply, c.TotalQuoteFundRaising, c.MigrateType = str(d.Supply), str(d.TotalQuoteFundRaising), d.MigrateType
	}
	return c
}

// RaydiumLCPTradeLayout parses trade event data (v1)
type RaydiumLCPTradeLayout struct {
	PoolState       []byte
	TotalBaseSell   *big.Int
	VirtualBase     *big.Int
	VirtualQuote    *big.Int
	RealBaseBefore  *big.Int
	RealQuoteBefore *big.Int
	RealBaseAfter   *big.Int
	RealQuoteAfter  *big.Int
	AmountIn        *big.Int
	AmountOut       *big.Int
	ProtocolFee     *big.Int
	PlatformFee     *big.Int
	ShareFee        *big.Int
	TradeDirection  uint8
	PoolStatus      uint8
}

// ParseRaydiumLCPTradeLayout parses trade layout from bytes
// V1 layout (130 bytes total):
//
//	poolState        [32]byte  bytes 0-31
//	totalBaseSell    u64       bytes 32-39
//	virtualBase      u64       bytes 40-47
//	virtualQuote     u64       bytes 48-55
//	realBaseBefore   u64       bytes 56-63
//	realQuoteBefore  u64       bytes 64-71
//	realBaseAfter    u64       bytes 72-79
//	realQuoteAfter   u64       bytes 80-87
//	amountIn         u64       bytes 88-95
//	amountOut        u64       bytes 96-103
//	protocolFee      u64       bytes 104-111
//	platformFee      u64       bytes 112-119
//	shareFee         u64       bytes 120-127
//	tradeDirection   u8        byte 128
//	poolStatus       u8        byte 129
func ParseRaydiumLCPTradeLayout(data []byte) (*RaydiumLCPTradeLayout, error) {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	poolState, _ := reader.ReadFixedArray(32)

	// Read all u64 fields first (12 fields × 8 bytes = 96 bytes)
	layout := &RaydiumLCPTradeLayout{
		PoolState:       poolState,
		TotalBaseSell:   reader.ReadU64AsBigInt(),
		VirtualBase:     reader.ReadU64AsBigInt(),
		VirtualQuote:    reader.ReadU64AsBigInt(),
		RealBaseBefore:  reader.ReadU64AsBigInt(),
		RealQuoteBefore: reader.ReadU64AsBigInt(),
		RealBaseAfter:   reader.ReadU64AsBigInt(),
		RealQuoteAfter:  reader.ReadU64AsBigInt(),
		AmountIn:        reader.ReadU64AsBigInt(),
		AmountOut:       reader.ReadU64AsBigInt(),
		ProtocolFee:     reader.ReadU64AsBigInt(),
		PlatformFee:     reader.ReadU64AsBigInt(),
		ShareFee:        reader.ReadU64AsBigInt(),
	}

	// tradeDirection and poolStatus are at the END (bytes 128-129)
	layout.TradeDirection, _ = reader.ReadU8()
	layout.PoolStatus, _ = reader.ReadU8()

	if reader.HasError() {
		return nil, reader.Error()
	}

	return layout, nil
}

// ToObject converts layout to RaydiumLCPTradeEvent
func (l *RaydiumLCPTradeLayout) ToObject() *RaydiumLCPTradeEvent {
	return &RaydiumLCPTradeEvent{
		PoolState:       base58.Encode(l.PoolState),
		TotalBaseSell:   l.TotalBaseSell,
		VirtualBase:     l.VirtualBase,
		VirtualQuote:    l.VirtualQuote,
		RealBaseBefore:  l.RealBaseBefore,
		RealQuoteBefore: l.RealQuoteBefore,
		RealBaseAfter:   l.RealBaseAfter,
		RealQuoteAfter:  l.RealQuoteAfter,
		AmountIn:        l.AmountIn,
		AmountOut:       l.AmountOut,
		ProtocolFee:     l.ProtocolFee,
		PlatformFee:     l.PlatformFee,
		CreatorFee:      big.NewInt(0),
		ShareFee:        l.ShareFee,
		TradeDirection:  TradeDirection(l.TradeDirection),
		PoolStatus:      PoolStatus(l.PoolStatus),
		BaseMint:        "",
		QuoteMint:       "",
		User:            "",
	}
}

// RaydiumLCPTradeV2Layout parses trade event data (v2)
type RaydiumLCPTradeV2Layout struct {
	PoolState       []byte
	TotalBaseSell   *big.Int
	VirtualBase     *big.Int
	VirtualQuote    *big.Int
	RealBaseBefore  *big.Int
	RealQuoteBefore *big.Int
	RealBaseAfter   *big.Int
	RealQuoteAfter  *big.Int
	AmountIn        *big.Int
	AmountOut       *big.Int
	ProtocolFee     *big.Int
	PlatformFee     *big.Int
	CreatorFee      *big.Int
	ShareFee        *big.Int
	TradeDirection  uint8
	PoolStatus      uint8
}

// ParseRaydiumLCPTradeV2Layout parses trade v2 layout from bytes
func ParseRaydiumLCPTradeV2Layout(data []byte) (*RaydiumLCPTradeV2Layout, error) {
	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	poolState, _ := reader.ReadFixedArray(32)

	layout := &RaydiumLCPTradeV2Layout{
		PoolState:       poolState,
		TotalBaseSell:   reader.ReadU64AsBigInt(),
		VirtualBase:     reader.ReadU64AsBigInt(),
		VirtualQuote:    reader.ReadU64AsBigInt(),
		RealBaseBefore:  reader.ReadU64AsBigInt(),
		RealQuoteBefore: reader.ReadU64AsBigInt(),
		RealBaseAfter:   reader.ReadU64AsBigInt(),
		RealQuoteAfter:  reader.ReadU64AsBigInt(),
		AmountIn:        reader.ReadU64AsBigInt(),
		AmountOut:       reader.ReadU64AsBigInt(),
		ProtocolFee:     reader.ReadU64AsBigInt(),
		PlatformFee:     reader.ReadU64AsBigInt(),
		CreatorFee:      reader.ReadU64AsBigInt(),
		ShareFee:        reader.ReadU64AsBigInt(),
	}

	layout.TradeDirection, _ = reader.ReadU8()
	layout.PoolStatus, _ = reader.ReadU8()

	if reader.HasError() {
		return nil, reader.Error()
	}

	return layout, nil
}

// ToObject converts layout to RaydiumLCPTradeEvent
func (l *RaydiumLCPTradeV2Layout) ToObject() *RaydiumLCPTradeEvent {
	return &RaydiumLCPTradeEvent{
		PoolState:       base58.Encode(l.PoolState),
		TotalBaseSell:   l.TotalBaseSell,
		VirtualBase:     l.VirtualBase,
		VirtualQuote:    l.VirtualQuote,
		RealBaseBefore:  l.RealBaseBefore,
		RealQuoteBefore: l.RealQuoteBefore,
		RealBaseAfter:   l.RealBaseAfter,
		RealQuoteAfter:  l.RealQuoteAfter,
		AmountIn:        l.AmountIn,
		AmountOut:       l.AmountOut,
		ProtocolFee:     l.ProtocolFee,
		PlatformFee:     l.PlatformFee,
		CreatorFee:      l.CreatorFee,
		ShareFee:        l.ShareFee,
		TradeDirection:  TradeDirection(l.TradeDirection),
		PoolStatus:      PoolStatus(l.PoolStatus),
		BaseMint:        "",
		QuoteMint:       "",
		User:            "",
	}
}

// LogType represents the Raydium AMM v4 "ray_log" types (raydium-amm
// program/src/log.rs). The payload is bincode: little-endian integers,
// pubkeys as 32 bytes.
type LogType uint8

const (
	LogTypeInit        LogType = 0
	LogTypeDeposit     LogType = 1
	LogTypeWithdraw    LogType = 2
	LogTypeSwapBaseIn  LogType = 3
	LogTypeSwapBaseOut LogType = 4
)

// InitLog is written by initialize2: the initial pool amounts
type InitLog struct {
	LogType      LogType
	Time         uint64
	PcDecimals   uint8
	CoinDecimals uint8
	PcLotSize    uint64
	CoinLotSize  uint64
	PcAmount     *big.Int
	CoinAmount   *big.Int
	Market       string
}

// DepositLog represents a deposit log
type DepositLog struct {
	LogType    LogType
	MaxCoin    *big.Int
	MaxPc      *big.Int
	Base       *big.Int
	PoolCoin   *big.Int
	PoolPc     *big.Int
	PoolLp     *big.Int
	CalcPnlX   *big.Int
	CalcPnlY   *big.Int
	DeductCoin *big.Int
	DeductPc   *big.Int
	MintLp     *big.Int
}

// WithdrawLog represents a withdraw log
type WithdrawLog struct {
	LogType    LogType
	WithdrawLp *big.Int
	UserLp     *big.Int
	PoolCoin   *big.Int
	PoolPc     *big.Int
	PoolLp     *big.Int
	CalcPnlX   *big.Int
	CalcPnlY   *big.Int
	OutCoin    *big.Int
	OutPc      *big.Int
}

// SwapBaseInLog represents a swap base in log
type SwapBaseInLog struct {
	LogType    LogType
	AmountIn   *big.Int
	MinimumOut *big.Int
	Direction  *big.Int
	UserSource *big.Int
	PoolCoin   *big.Int
	PoolPc     *big.Int
	OutAmount  *big.Int
}

// SwapBaseOutLog represents a swap base out log
type SwapBaseOutLog struct {
	LogType    LogType
	MaxIn      *big.Int
	AmountOut  *big.Int
	Direction  *big.Int
	UserSource *big.Int
	PoolCoin   *big.Int
	PoolPc     *big.Int
	DeductIn   *big.Int
}

// DecodeRaydiumLog decodes the payload of a "ray_log:" line (already base64
// decoded) into *InitLog, *DepositLog, *WithdrawLog, *SwapBaseInLog or
// *SwapBaseOutLog. It returns nil for an unknown type or a short payload.
func DecodeRaydiumLog(data []byte) interface{} {
	if len(data) < 1 {
		return nil
	}

	logType := LogType(data[0])
	reader := utils.GetBinaryReader(data[1:])
	defer reader.Release()

	result := decodeRaydiumLog(logType, reader)
	if reader.HasError() {
		return nil
	}
	return result
}

func decodeRaydiumLog(logType LogType, reader *utils.BinaryReader) interface{} {
	switch logType {
	case LogTypeInit:
		l := &InitLog{LogType: logType}
		l.Time, _ = reader.ReadU64()
		l.PcDecimals, _ = reader.ReadU8()
		l.CoinDecimals, _ = reader.ReadU8()
		l.PcLotSize, _ = reader.ReadU64()
		l.CoinLotSize, _ = reader.ReadU64()
		l.PcAmount = reader.ReadU64AsBigInt()
		l.CoinAmount = reader.ReadU64AsBigInt()
		l.Market, _ = reader.ReadPubkey()
		return l
	case LogTypeDeposit:
		return &DepositLog{
			LogType:    logType,
			MaxCoin:    reader.ReadU64AsBigInt(),
			MaxPc:      reader.ReadU64AsBigInt(),
			Base:       reader.ReadU64AsBigInt(),
			PoolCoin:   reader.ReadU64AsBigInt(),
			PoolPc:     reader.ReadU64AsBigInt(),
			PoolLp:     reader.ReadU64AsBigInt(),
			CalcPnlX:   reader.ReadU128AsBigInt(),
			CalcPnlY:   reader.ReadU128AsBigInt(),
			DeductCoin: reader.ReadU64AsBigInt(),
			DeductPc:   reader.ReadU64AsBigInt(),
			MintLp:     reader.ReadU64AsBigInt(),
		}
	case LogTypeWithdraw:
		return &WithdrawLog{
			LogType:    logType,
			WithdrawLp: reader.ReadU64AsBigInt(),
			UserLp:     reader.ReadU64AsBigInt(),
			PoolCoin:   reader.ReadU64AsBigInt(),
			PoolPc:     reader.ReadU64AsBigInt(),
			PoolLp:     reader.ReadU64AsBigInt(),
			CalcPnlX:   reader.ReadU128AsBigInt(),
			CalcPnlY:   reader.ReadU128AsBigInt(),
			OutCoin:    reader.ReadU64AsBigInt(),
			OutPc:      reader.ReadU64AsBigInt(),
		}
	case LogTypeSwapBaseIn:
		return &SwapBaseInLog{
			LogType:    logType,
			AmountIn:   reader.ReadU64AsBigInt(),
			MinimumOut: reader.ReadU64AsBigInt(),
			Direction:  reader.ReadU64AsBigInt(),
			UserSource: reader.ReadU64AsBigInt(),
			PoolCoin:   reader.ReadU64AsBigInt(),
			PoolPc:     reader.ReadU64AsBigInt(),
			OutAmount:  reader.ReadU64AsBigInt(),
		}
	case LogTypeSwapBaseOut:
		return &SwapBaseOutLog{
			LogType:    logType,
			MaxIn:      reader.ReadU64AsBigInt(),
			AmountOut:  reader.ReadU64AsBigInt(),
			Direction:  reader.ReadU64AsBigInt(),
			UserSource: reader.ReadU64AsBigInt(),
			PoolCoin:   reader.ReadU64AsBigInt(),
			PoolPc:     reader.ReadU64AsBigInt(),
			DeductIn:   reader.ReadU64AsBigInt(),
		}
	}

	return nil
}

// Swap direction constants: the direction field of the swap logs
// (raydium-amm program/src/math.rs SwapDirection)
const (
	SwapDirectionPCToCoin = 1 // pc (quote) in, coin (base) out
	SwapDirectionCoinToPC = 2 // coin (base) in, pc (quote) out
)

// SwapOperation represents parsed swap details
type SwapOperation struct {
	Type               string // "Buy" or "Sell"
	Mode               string // "Exact Input" or "Exact Output"
	InputAmount        *big.Int
	OutputAmount       *big.Int
	SlippageProtection *big.Int
}

// ParseRaydiumSwapLog parses swap operation details from a swap log
func ParseRaydiumSwapLog(log interface{}) *SwapOperation {
	switch l := log.(type) {
	case *SwapBaseInLog:
		isBuy := l.Direction.Int64() == SwapDirectionPCToCoin
		opType := "Sell"
		if isBuy {
			opType = "Buy"
		}
		return &SwapOperation{
			Type:               opType,
			Mode:               "Exact Input",
			InputAmount:        l.AmountIn,
			OutputAmount:       l.OutAmount,
			SlippageProtection: l.MinimumOut,
		}

	case *SwapBaseOutLog:
		isBuy := l.Direction.Int64() == SwapDirectionPCToCoin
		opType := "Sell"
		if isBuy {
			opType = "Buy"
		}
		return &SwapOperation{
			Type:               opType,
			Mode:               "Exact Output",
			InputAmount:        l.DeductIn,
			OutputAmount:       l.AmountOut,
			SlippageProtection: l.MaxIn,
		}

	default:
		return nil
	}
}
