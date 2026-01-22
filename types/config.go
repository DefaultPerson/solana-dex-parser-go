package types

// ParseType defines configuration options for parsing output granularity.
// Each field controls whether to parse and return specific event types.
type ParseType struct {
	// AggregateTrade if true, returns the aggregated trade record for Jupiter routes
	AggregateTrade bool `json:"aggregateTrade,omitempty"`

	// Trade if true, returns individual trade events
	Trade bool `json:"trade,omitempty"`

	// Liquidity if true, returns liquidity pool events (add/remove/create)
	Liquidity bool `json:"liquidity,omitempty"`

	// Transfer if true, returns token transfer events
	Transfer bool `json:"transfer,omitempty"`

	// MemeEvent if true, returns meme platform events (create/buy/sell/migrate)
	MemeEvent bool `json:"memeEvent,omitempty"`

	// AltEvent if true, returns Address Lookup Table events
	AltEvent bool `json:"altEvent,omitempty"`
}

// ParseAll returns a ParseType with all parsing options enabled
func ParseAll() ParseType {
	return ParseType{
		AggregateTrade: true,
		Trade:          true,
		Liquidity:      true,
		Transfer:       true,
		MemeEvent:      true,
		AltEvent:       true,
	}
}

// ParseTradesOnly returns a ParseType for parsing only trades
func ParseTradesOnly() ParseType {
	return ParseType{
		AggregateTrade: true,
		Trade:          true,
	}
}

// ParseLiquidityOnly returns a ParseType for parsing only liquidity events
func ParseLiquidityOnly() ParseType {
	return ParseType{
		Liquidity: true,
	}
}

// ParseConfig contains configuration options for transaction parsing
type ParseConfig struct {
	// ParseType controls which event types to parse and return
	ParseType ParseType `json:"parseType,omitempty"`

	// TryUnknownDEX if true, will try to parse unknown DEXes (results may be inaccurate)
	TryUnknownDEX bool `json:"tryUnknownDEX,omitempty"`

	// ProgramIds if set, will only parse transactions from these program IDs
	ProgramIds []string `json:"programIds,omitempty"`

	// IgnoreProgramIds if set, will ignore transactions from these program IDs
	IgnoreProgramIds []string `json:"ignoreProgramIds,omitempty"`

	// AccountInclude if set, will only parse transactions that include any of these accounts
	AccountInclude []string `json:"accountInclude,omitempty"`

	// AccountExclude if set, will skip transactions that include any of these accounts
	AccountExclude []string `json:"accountExclude,omitempty"`

	// ThrowError if true, will panic on parse errors instead of returning error state
	ThrowError bool `json:"throwError,omitempty"`

	// AggregateTrades if true, will return the finalSwap record instead of detail route trades
	// Deprecated: Use ParseType.AggregateTrade instead. Kept for backward compatibility.
	AggregateTrades bool `json:"aggregateTrades,omitempty"`

	// ALTsFetcher if set, will use this callback to fetch Address Lookup Table accounts
	ALTsFetcher *ALTsFetcher `json:"-"`

	// TokenAccountsFetcher if set, will use this callback to fetch token account info
	TokenAccountsFetcher *TokenAccountsFetcher `json:"-"`

	// PoolInfoFetcher if set, will use this callback to fetch pool information
	PoolInfoFetcher *PoolInfoFetcher `json:"-"`
}

// DefaultParseConfig returns default parsing configuration with all events enabled
func DefaultParseConfig() ParseConfig {
	return ParseConfig{
		ParseType:     ParseAll(),
		TryUnknownDEX: true,
	}
}

// DefaultParseConfigTradesOnly returns parsing configuration for trades only
func DefaultParseConfigTradesOnly() ParseConfig {
	return ParseConfig{
		ParseType:     ParseTradesOnly(),
		TryUnknownDEX: true,
	}
}

// ShouldAggregateTrades returns true if trades should be aggregated
// Checks both ParseType.AggregateTrade and legacy AggregateTrades field
func (c *ParseConfig) ShouldAggregateTrades() bool {
	return c.ParseType.AggregateTrade || c.AggregateTrades
}

// IsParseTypeSet returns true if any ParseType field is explicitly set
func (c *ParseConfig) IsParseTypeSet() bool {
	return c.ParseType.AggregateTrade || c.ParseType.Trade ||
		c.ParseType.Liquidity || c.ParseType.Transfer ||
		c.ParseType.MemeEvent || c.ParseType.AltEvent
}

// GetEffectiveParseType returns the effective ParseType, defaulting to ParseAll if not set
func (c *ParseConfig) GetEffectiveParseType() ParseType {
	if c.IsParseTypeSet() {
		return c.ParseType
	}
	// Default to all parsing enabled for backward compatibility
	return ParseAll()
}
