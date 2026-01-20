package jupiter

import (
	"encoding/binary"
	"math/big"

	"github.com/mr-tron/base58"
)

// JupiterSwapLayout represents Jupiter V6 swap event data
type JupiterSwapLayout struct {
	AMM          [32]byte
	InputMint    [32]byte
	InputAmount  uint64
	OutputMint   [32]byte
	OutputAmount uint64
}

// ParseJupiterSwapLayout parses Jupiter V6 swap event from data
// Layout: amm(32) + inputMint(32) + inputAmount(8) + outputMint(32) + outputAmount(8)
func ParseJupiterSwapLayout(data []byte) (*JupiterSwapLayout, error) {
	if len(data) < 112 {
		return nil, ErrInsufficientData
	}

	layout := &JupiterSwapLayout{}
	copy(layout.AMM[:], data[0:32])
	copy(layout.InputMint[:], data[32:64])
	layout.InputAmount = binary.LittleEndian.Uint64(data[64:72])
	copy(layout.OutputMint[:], data[72:104])
	layout.OutputAmount = binary.LittleEndian.Uint64(data[104:112])

	return layout, nil
}

// ToSwapEvent converts layout to swap event data
func (l *JupiterSwapLayout) ToSwapEvent() *JupiterSwapEvent {
	return &JupiterSwapEvent{
		AMM:          base58.Encode(l.AMM[:]),
		InputMint:    base58.Encode(l.InputMint[:]),
		InputAmount:  new(big.Int).SetUint64(l.InputAmount),
		OutputMint:   base58.Encode(l.OutputMint[:]),
		OutputAmount: new(big.Int).SetUint64(l.OutputAmount),
	}
}

// JupiterSwapEvent represents parsed swap event
type JupiterSwapEvent struct {
	AMM                string
	InputMint          string
	InputAmount        *big.Int
	OutputMint         string
	OutputAmount       *big.Int
	InputMintDecimals  uint8
	OutputMintDecimals uint8
	Idx                string
}

// JupiterDCAFilledLayout represents Jupiter DCA filled event data
type JupiterDCAFilledLayout struct {
	UserKey    [32]byte
	DCAKey     [32]byte
	InputMint  [32]byte
	OutputMint [32]byte
	InAmount   uint64
	OutAmount  uint64
	FeeMint    [32]byte
	Fee        uint64
}

// ParseJupiterDCAFilledLayout parses Jupiter DCA filled event
// Layout: userKey(32) + dcaKey(32) + inputMint(32) + outputMint(32) + inAmount(8) + outAmount(8) + feeMint(32) + fee(8)
func ParseJupiterDCAFilledLayout(data []byte) (*JupiterDCAFilledLayout, error) {
	if len(data) < 184 {
		return nil, ErrInsufficientData
	}

	layout := &JupiterDCAFilledLayout{}
	copy(layout.UserKey[:], data[0:32])
	copy(layout.DCAKey[:], data[32:64])
	copy(layout.InputMint[:], data[64:96])
	copy(layout.OutputMint[:], data[96:128])
	layout.InAmount = binary.LittleEndian.Uint64(data[128:136])
	layout.OutAmount = binary.LittleEndian.Uint64(data[136:144])
	copy(layout.FeeMint[:], data[144:176])
	layout.Fee = binary.LittleEndian.Uint64(data[176:184])

	return layout, nil
}

// ToObject converts layout to object with base58 encoded keys
func (l *JupiterDCAFilledLayout) ToObject() *JupiterDCAFilledEvent {
	return &JupiterDCAFilledEvent{
		UserKey:    base58.Encode(l.UserKey[:]),
		DCAKey:     base58.Encode(l.DCAKey[:]),
		InputMint:  base58.Encode(l.InputMint[:]),
		OutputMint: base58.Encode(l.OutputMint[:]),
		InAmount:   new(big.Int).SetUint64(l.InAmount),
		OutAmount:  new(big.Int).SetUint64(l.OutAmount),
		FeeMint:    base58.Encode(l.FeeMint[:]),
		Fee:        new(big.Int).SetUint64(l.Fee),
	}
}

// JupiterDCAFilledEvent represents parsed DCA filled event
type JupiterDCAFilledEvent struct {
	UserKey    string
	DCAKey     string
	InputMint  string
	OutputMint string
	InAmount   *big.Int
	OutAmount  *big.Int
	FeeMint    string
	Fee        *big.Int
}

// JupiterLimitOrderV2TradeLayout represents Jupiter Limit Order V2 trade event
type JupiterLimitOrderV2TradeLayout struct {
	OrderKey             [32]byte
	Taker                [32]byte
	RemainingMakingAmt   uint64
	RemainingTakingAmt   uint64
	MakingAmount         uint64
	TakingAmount         uint64
}

// ParseJupiterLimitOrderV2TradeLayout parses Jupiter Limit Order V2 trade event
func ParseJupiterLimitOrderV2TradeLayout(data []byte) (*JupiterLimitOrderV2TradeLayout, error) {
	if len(data) < 96 {
		return nil, ErrInsufficientData
	}

	layout := &JupiterLimitOrderV2TradeLayout{}
	copy(layout.OrderKey[:], data[0:32])
	copy(layout.Taker[:], data[32:64])
	layout.RemainingMakingAmt = binary.LittleEndian.Uint64(data[64:72])
	layout.RemainingTakingAmt = binary.LittleEndian.Uint64(data[72:80])
	layout.MakingAmount = binary.LittleEndian.Uint64(data[80:88])
	layout.TakingAmount = binary.LittleEndian.Uint64(data[88:96])

	return layout, nil
}

// ToObject converts layout to object
func (l *JupiterLimitOrderV2TradeLayout) ToObject() *JupiterLimitOrderV2TradeEvent {
	return &JupiterLimitOrderV2TradeEvent{
		OrderKey:           base58.Encode(l.OrderKey[:]),
		Taker:              base58.Encode(l.Taker[:]),
		RemainingMakingAmt: new(big.Int).SetUint64(l.RemainingMakingAmt),
		RemainingTakingAmt: new(big.Int).SetUint64(l.RemainingTakingAmt),
		MakingAmount:       new(big.Int).SetUint64(l.MakingAmount),
		TakingAmount:       new(big.Int).SetUint64(l.TakingAmount),
	}
}

// JupiterLimitOrderV2TradeEvent represents parsed trade event
type JupiterLimitOrderV2TradeEvent struct {
	OrderKey           string
	Taker              string
	RemainingMakingAmt *big.Int
	RemainingTakingAmt *big.Int
	MakingAmount       *big.Int
	TakingAmount       *big.Int
}

// JupiterLimitOrderV2CreateOrderLayout represents Jupiter Limit Order V2 create event
type JupiterLimitOrderV2CreateOrderLayout struct {
	OrderKey           [32]byte
	Maker              [32]byte
	InputMint          [32]byte
	OutputMint         [32]byte
	InputTokenProgram  [32]byte
	OutputTokenProgram [32]byte
	MakingAmount       uint64
	TakingAmount       uint64
	ExpiredAt          *int64
	FeeBps             uint16
	FeeAccount         [32]byte
}

// ParseJupiterLimitOrderV2CreateOrderLayout parses Jupiter Limit Order V2 create order event
func ParseJupiterLimitOrderV2CreateOrderLayout(data []byte) (*JupiterLimitOrderV2CreateOrderLayout, error) {
	if len(data) < 243 { // 32*6 + 8*2 + 1 + 8? + 2 + 32
		return nil, ErrInsufficientData
	}

	layout := &JupiterLimitOrderV2CreateOrderLayout{}
	offset := 0

	copy(layout.OrderKey[:], data[offset:offset+32])
	offset += 32
	copy(layout.Maker[:], data[offset:offset+32])
	offset += 32
	copy(layout.InputMint[:], data[offset:offset+32])
	offset += 32
	copy(layout.OutputMint[:], data[offset:offset+32])
	offset += 32
	copy(layout.InputTokenProgram[:], data[offset:offset+32])
	offset += 32
	copy(layout.OutputTokenProgram[:], data[offset:offset+32])
	offset += 32
	layout.MakingAmount = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.TakingAmount = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Optional expiredAt
	if data[offset] == 1 {
		offset++
		if len(data) < offset+8 {
			return nil, ErrInsufficientData
		}
		expiredAt := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
		layout.ExpiredAt = &expiredAt
		offset += 8
	} else {
		offset++
	}

	if len(data) < offset+34 {
		return nil, ErrInsufficientData
	}
	layout.FeeBps = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2
	copy(layout.FeeAccount[:], data[offset:offset+32])

	return layout, nil
}

// ToObject converts layout to object
func (l *JupiterLimitOrderV2CreateOrderLayout) ToObject() *JupiterLimitOrderV2CreateOrderEvent {
	var expiredAt *string
	if l.ExpiredAt != nil {
		s := big.NewInt(*l.ExpiredAt).String()
		expiredAt = &s
	}
	return &JupiterLimitOrderV2CreateOrderEvent{
		OrderKey:           base58.Encode(l.OrderKey[:]),
		Maker:              base58.Encode(l.Maker[:]),
		InputMint:          base58.Encode(l.InputMint[:]),
		OutputMint:         base58.Encode(l.OutputMint[:]),
		InputTokenProgram:  base58.Encode(l.InputTokenProgram[:]),
		OutputTokenProgram: base58.Encode(l.OutputTokenProgram[:]),
		MakingAmount:       new(big.Int).SetUint64(l.MakingAmount),
		TakingAmount:       new(big.Int).SetUint64(l.TakingAmount),
		ExpiredAt:          expiredAt,
		FeeBps:             l.FeeBps,
		FeeAccount:         base58.Encode(l.FeeAccount[:]),
	}
}

// JupiterLimitOrderV2CreateOrderEvent represents parsed create order event
type JupiterLimitOrderV2CreateOrderEvent struct {
	OrderKey           string
	Maker              string
	InputMint          string
	OutputMint         string
	InputTokenProgram  string
	OutputTokenProgram string
	MakingAmount       *big.Int
	TakingAmount       *big.Int
	ExpiredAt          *string
	FeeBps             uint16
	FeeAccount         string
}

// JupiterVAFillLayout represents Jupiter VA fill event data
type JupiterVAFillLayout struct {
	ValueAverage       [32]byte
	User               [32]byte
	Keeper             [32]byte
	InputMint          [32]byte
	OutputMint         [32]byte
	InputAmount        uint64
	OutputAmount       uint64
	Fee                uint64
	NewActualUsdcValue uint64
	SupposedUsdcValue  uint64
	Value              uint64
	InLeft             uint64
	InUsed             uint64
	OutReceived        uint64
}

// ParseJupiterVAFillLayout parses Jupiter VA fill event
func ParseJupiterVAFillLayout(data []byte) (*JupiterVAFillLayout, error) {
	if len(data) < 232 { // 32*5 + 8*9 = 160 + 72 = 232
		return nil, ErrInsufficientData
	}

	layout := &JupiterVAFillLayout{}
	offset := 0

	copy(layout.ValueAverage[:], data[offset:offset+32])
	offset += 32
	copy(layout.User[:], data[offset:offset+32])
	offset += 32
	copy(layout.Keeper[:], data[offset:offset+32])
	offset += 32
	copy(layout.InputMint[:], data[offset:offset+32])
	offset += 32
	copy(layout.OutputMint[:], data[offset:offset+32])
	offset += 32
	layout.InputAmount = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.OutputAmount = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.Fee = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.NewActualUsdcValue = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.SupposedUsdcValue = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.Value = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.InLeft = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.InUsed = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.OutReceived = binary.LittleEndian.Uint64(data[offset : offset+8])

	return layout, nil
}

// ToObject converts layout to object
func (l *JupiterVAFillLayout) ToObject() *JupiterVAFillEvent {
	return &JupiterVAFillEvent{
		ValueAverage:       base58.Encode(l.ValueAverage[:]),
		User:               base58.Encode(l.User[:]),
		Keeper:             base58.Encode(l.Keeper[:]),
		InputMint:          base58.Encode(l.InputMint[:]),
		OutputMint:         base58.Encode(l.OutputMint[:]),
		InputAmount:        new(big.Int).SetUint64(l.InputAmount),
		OutputAmount:       new(big.Int).SetUint64(l.OutputAmount),
		Fee:                new(big.Int).SetUint64(l.Fee),
		NewActualUsdcValue: l.NewActualUsdcValue,
		SupposedUsdcValue:  l.SupposedUsdcValue,
		Value:              l.Value,
		InLeft:             l.InLeft,
		InUsed:             l.InUsed,
		OutReceived:        l.OutReceived,
	}
}

// JupiterVAFillEvent represents parsed VA fill event
type JupiterVAFillEvent struct {
	ValueAverage       string
	User               string
	Keeper             string
	InputMint          string
	OutputMint         string
	InputAmount        *big.Int
	OutputAmount       *big.Int
	Fee                *big.Int
	NewActualUsdcValue uint64
	SupposedUsdcValue  uint64
	Value              uint64
	InLeft             uint64
	InUsed             uint64
	OutReceived        uint64
}

// JupiterVAOpenLayout represents Jupiter VA open event data
type JupiterVAOpenLayout struct {
	User               [32]byte
	ValueAverage       [32]byte
	Deposit            uint64
	InputMint          [32]byte
	OutputMint         [32]byte
	ReferralFeeAccount [32]byte
	OrderInterval      int64
	IncrementUsdcValue uint64
	CreatedAt          int64
}

// ParseJupiterVAOpenLayout parses Jupiter VA open event
func ParseJupiterVAOpenLayout(data []byte) (*JupiterVAOpenLayout, error) {
	if len(data) < 192 { // 32*4 + 8*4 = 128 + 32 = 160... actually: 32+32+8+32+32+32+8+8+8 = 192
		return nil, ErrInsufficientData
	}

	layout := &JupiterVAOpenLayout{}
	offset := 0

	copy(layout.User[:], data[offset:offset+32])
	offset += 32
	copy(layout.ValueAverage[:], data[offset:offset+32])
	offset += 32
	layout.Deposit = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	copy(layout.InputMint[:], data[offset:offset+32])
	offset += 32
	copy(layout.OutputMint[:], data[offset:offset+32])
	offset += 32
	copy(layout.ReferralFeeAccount[:], data[offset:offset+32])
	offset += 32
	layout.OrderInterval = int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8
	layout.IncrementUsdcValue = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.CreatedAt = int64(binary.LittleEndian.Uint64(data[offset : offset+8]))

	return layout, nil
}

// ToObject converts layout to object
func (l *JupiterVAOpenLayout) ToObject() *JupiterVAOpenEvent {
	return &JupiterVAOpenEvent{
		User:               base58.Encode(l.User[:]),
		ValueAverage:       base58.Encode(l.ValueAverage[:]),
		Deposit:            l.Deposit,
		InputMint:          base58.Encode(l.InputMint[:]),
		OutputMint:         base58.Encode(l.OutputMint[:]),
		ReferralFeeAccount: base58.Encode(l.ReferralFeeAccount[:]),
		OrderInterval:      l.OrderInterval,
		IncrementUsdcValue: l.IncrementUsdcValue,
		CreatedAt:          l.CreatedAt,
	}
}

// JupiterVAOpenEvent represents parsed VA open event
type JupiterVAOpenEvent struct {
	User               string
	ValueAverage       string
	Deposit            uint64
	InputMint          string
	OutputMint         string
	ReferralFeeAccount string
	OrderInterval      int64
	IncrementUsdcValue uint64
	CreatedAt          int64
}

// JupiterVAWithdrawLayout represents Jupiter VA withdraw event data
type JupiterVAWithdrawLayout struct {
	ValueAverage [32]byte
	Mint         [32]byte
	Amount       uint64
	InOrOut      uint8
	UserWithdraw uint8
	InLeft       uint64
	InWithdrawn  uint64
	OutWithdrawn uint64
}

// ParseJupiterVAWithdrawLayout parses Jupiter VA withdraw event
func ParseJupiterVAWithdrawLayout(data []byte) (*JupiterVAWithdrawLayout, error) {
	if len(data) < 90 { // 32+32+8+1+1+8+8+8 = 98
		return nil, ErrInsufficientData
	}

	layout := &JupiterVAWithdrawLayout{}
	offset := 0

	copy(layout.ValueAverage[:], data[offset:offset+32])
	offset += 32
	copy(layout.Mint[:], data[offset:offset+32])
	offset += 32
	layout.Amount = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.InOrOut = data[offset]
	offset++
	layout.UserWithdraw = data[offset]
	offset++
	layout.InLeft = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.InWithdrawn = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	layout.OutWithdrawn = binary.LittleEndian.Uint64(data[offset : offset+8])

	return layout, nil
}

// ToObject converts layout to object
func (l *JupiterVAWithdrawLayout) ToObject() *JupiterVAWithdrawEvent {
	inOrOut := "In"
	if l.InOrOut != 0 {
		inOrOut = "Out"
	}
	return &JupiterVAWithdrawEvent{
		ValueAverage: base58.Encode(l.ValueAverage[:]),
		Mint:         base58.Encode(l.Mint[:]),
		Amount:       l.Amount,
		InOrOut:      inOrOut,
		UserWithdraw: l.UserWithdraw == 1,
		InLeft:       l.InLeft,
		InWithdrawn:  l.InWithdrawn,
		OutWithdrawn: l.OutWithdrawn,
	}
}

// JupiterVAWithdrawEvent represents parsed VA withdraw event
type JupiterVAWithdrawEvent struct {
	ValueAverage string
	Mint         string
	Amount       uint64
	InOrOut      string
	UserWithdraw bool
	InLeft       uint64
	InWithdrawn  uint64
	OutWithdrawn uint64
}

// Custom error
var ErrInsufficientData = &InsufficientDataError{}

type InsufficientDataError struct{}

func (e *InsufficientDataError) Error() string {
	return "insufficient data for layout parsing"
}
