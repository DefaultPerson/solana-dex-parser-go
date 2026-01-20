package types

// PoolEventType represents the type of liquidity pool event
type PoolEventType string

const (
	PoolEventTypeCreate PoolEventType = "CREATE"
	PoolEventTypeAdd    PoolEventType = "ADD"
	PoolEventTypeRemove PoolEventType = "REMOVE"
)

// PoolEventBase contains base fields for pool events
type PoolEventBase struct {
	User      string        `json:"user"`                // User address
	Type      PoolEventType `json:"type"`                // Event type (CREATE/ADD/REMOVE)
	ProgramId string        `json:"programId,omitempty"` // DEX program ID
	AMM       string        `json:"amm,omitempty"`       // AMM type
	Slot      uint64        `json:"slot"`                // Block slot number
	Timestamp int64         `json:"timestamp"`           // Unix timestamp
	Signature string        `json:"signature"`           // Transaction signature
	Idx       string        `json:"idx"`                 // Instruction indexes
	Signer    []string      `json:"signer,omitempty"`    // Original signer
}

// PoolEvent represents a liquidity pool event
type PoolEvent struct {
	PoolEventBase

	// PoolId is the AMM pool address (market)
	PoolId string `json:"poolId"`

	// Config is the pool config address (platform config)
	Config string `json:"config,omitempty"`

	// PoolLpMint is the LP mint address
	PoolLpMint string `json:"poolLpMint,omitempty"`

	// Token0Mint is Token A mint address (TOKEN)
	Token0Mint string `json:"token0Mint,omitempty"`

	// Token0Amount is Token A uiAmount (TOKEN)
	Token0Amount *float64 `json:"token0Amount,omitempty"`

	// Token0AmountRaw is Token A raw amount (TOKEN)
	Token0AmountRaw string `json:"token0AmountRaw,omitempty"`

	// Token0BalanceChange is user token0 balance changed amount
	Token0BalanceChange string `json:"token0BalanceChange,omitempty"`

	// Token0Decimals is Token A decimals
	Token0Decimals *uint8 `json:"token0Decimals,omitempty"`

	// Token1Mint is Token B mint address (SOL/USDC/USDT)
	Token1Mint string `json:"token1Mint,omitempty"`

	// Token1Amount is Token B uiAmount (SOL/USDC/USDT)
	Token1Amount *float64 `json:"token1Amount,omitempty"`

	// Token1AmountRaw is Token B raw amount (SOL/USDC/USDT)
	Token1AmountRaw string `json:"token1AmountRaw,omitempty"`

	// Token1BalanceChange is user token1 balance changed amount
	Token1BalanceChange string `json:"token1BalanceChange,omitempty"`

	// Token1Decimals is Token B decimals
	Token1Decimals *uint8 `json:"token1Decimals,omitempty"`

	// LpAmount is the LP token amount (UI)
	LpAmount *float64 `json:"lpAmount,omitempty"`

	// LpAmountRaw is the LP token raw amount
	LpAmountRaw string `json:"lpAmountRaw,omitempty"`
}
