package types

// TradeTypeBuyAndBurn is the MemeEvent type of a protocol buy-and-burn
// (PumpSwap boost_buy_and_burn): the protocol buys the coin with its boost
// vault and burns it. It is not a user trade and never produces a TradeInfo.
const TradeTypeBuyAndBurn TradeType = "BUY_AND_BURN"

// MemeEvent contains the unified event data for meme token operations
type MemeEvent struct {
	Type      TradeType `json:"type"`      // Type of the event (create/trade/migrate)
	Timestamp int64     `json:"timestamp"` // Event timestamp
	Idx       string    `json:"idx"`       // Event index
	Slot      uint64    `json:"slot"`      // Event slot
	Signature string    `json:"signature"` // Event signature

	// Common fields for all events
	User string `json:"user"` // User/trader address

	BaseMint  string `json:"baseMint"`  // Token mint address
	QuoteMint string `json:"quoteMint"` // Quote mint address

	// Trade-specific fields
	InputToken  *TokenInfo `json:"inputToken,omitempty"`  // Amount in
	OutputToken *TokenInfo `json:"outputToken,omitempty"` // Amount out

	// Token creation fields
	Name        string   `json:"name,omitempty"`        // Token name
	Symbol      string   `json:"symbol,omitempty"`      // Token symbol
	URI         string   `json:"uri,omitempty"`         // Token metadata URI
	Decimals    *uint8   `json:"decimals,omitempty"`    // Token decimals
	TotalSupply *float64 `json:"totalSupply,omitempty"` // Token total supply

	// Fee and economic fields
	Fee         *float64 `json:"fee,omitempty"`         // Fee
	ProtocolFee *float64 `json:"protocolFee,omitempty"` // Protocol fee
	PlatformFee *float64 `json:"platformFee,omitempty"` // Platform fee
	ShareFee    *float64 `json:"shareFee,omitempty"`    // Share fee
	CreatorFee  *float64 `json:"creatorFee,omitempty"`  // Creator fee

	// Protocol-specific addresses
	Protocol       string   `json:"protocol,omitempty"`       // Protocol name
	PlatformConfig string   `json:"platformConfig,omitempty"` // Platform config address
	Creator        string   `json:"creator,omitempty"`        // Token creator address
	BondingCurve   string   `json:"bondingCurve,omitempty"`   // Bonding curve address
	Pool           string   `json:"pool,omitempty"`           // Pool address
	PoolDex        string   `json:"poolDex,omitempty"`        // Pool Dex name
	PoolAReserve   *float64 `json:"poolAReserve,omitempty"`   // Pool A reserve
	PoolBReserve   *float64 `json:"poolBReserve,omitempty"`   // Pool B reserve
	PoolFeeRate    *float64 `json:"poolFeeRate,omitempty"`    // Pool fee rate

	// Extended protocol fields, set when the event carries them. Raw amounts
	// are exact integer strings in the smallest unit of their mint.
	IxName            string    `json:"ixName,omitempty"`            // Instruction that emitted the event (e.g. "buy", "sell", "buy_exact_quote_in")
	TokenProgram      string    `json:"tokenProgram,omitempty"`      // Token program of the base mint
	IsMayhemMode      bool      `json:"isMayhemMode,omitempty"`      // Pump.fun mayhem-mode coin
	IsCashbackEnabled bool      `json:"isCashbackEnabled,omitempty"` // Pump.fun cashback coin (creator fee paid back to traders)
	IsHolderReward    bool      `json:"isHolderReward,omitempty"`    // Pump.fun holder-rewards coin (creator fee set aside for holders)
	CreatorFeeBps     *uint64   `json:"creatorFeeBps,omitempty"`     // Creator fee rate in basis points
	Fees              []FeeInfo `json:"fees,omitempty"`              // Fee components with exact raw amounts

	VirtualBaseReserves  string `json:"virtualBaseReserves,omitempty"`  // Virtual base reserves after the event
	VirtualQuoteReserves string `json:"virtualQuoteReserves,omitempty"` // Virtual quote reserves after the event
	RealBaseReserves     string `json:"realBaseReserves,omitempty"`     // Real base reserves after the event
	RealQuoteReserves    string `json:"realQuoteReserves,omitempty"`    // Real quote reserves after the event

	BaseAmount   string `json:"baseAmount,omitempty"`   // Migrations: base amount moved to the new pool
	QuoteAmount  string `json:"quoteAmount,omitempty"`  // Migrations: quote amount moved to the new pool
	MigrationFee string `json:"migrationFee,omitempty"` // Migrations: pool migration fee (quote units)

	Curve *MemeCurveParams `json:"curve,omitempty"` // Bonding-curve parameters set at creation
}

// MemeCurveParams holds the bonding-curve parameters of a launchpad pool
// (Raydium LaunchLab PoolCreateEvent). Amounts are raw integer strings.
type MemeCurveParams struct {
	Type                  string `json:"type"`                            // "Constant", "Fixed" or "Linear"
	Supply                string `json:"supply,omitempty"`                // Total base supply
	TotalBaseSell         string `json:"totalBaseSell,omitempty"`         // Base amount sold on the curve (Constant curves)
	TotalQuoteFundRaising string `json:"totalQuoteFundRaising,omitempty"` // Quote amount raised before migration
	MigrateType           uint8  `json:"migrateType"`                     // 0: AMM, 1: CPSwap
	TotalLockedAmount     string `json:"totalLockedAmount,omitempty"`     // Vesting: locked base amount
	CliffPeriod           string `json:"cliffPeriod,omitempty"`           // Vesting: cliff period (seconds)
	UnlockPeriod          string `json:"unlockPeriod,omitempty"`          // Vesting: unlock period (seconds)
}
