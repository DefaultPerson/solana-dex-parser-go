package types

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
}
