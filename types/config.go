package types

// ParseType defines configuration options for parsing output granularity.
// Each field controls whether to parse and return specific event types.
type ParseType struct {
	// AggregateTrade if true, returns the aggregated trade record (ParseResult.AggregateTrade)
	// combining all trades of the transaction. It is computed in addition to the
	// individual trades; set alone, trades are parsed internally but only the
	// aggregate is returned.
	AggregateTrade bool `json:"aggregateTrade,omitempty"`

	// Trade if true, returns individual trade events (ParseResult.Trades)
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

	// AggregateTrades if true, also returns the aggregated trade record (ParseResult.AggregateTrade).
	// With ParseType unset it is the only switch for aggregation; with ParseType set,
	// aggregation is ParseType.AggregateTrade || AggregateTrades.
	// Deprecated: Use ParseType.AggregateTrade instead. Kept for backward compatibility.
	AggregateTrades bool `json:"aggregateTrades,omitempty"`

	// IncludeFailedTxs if true, parses failed transactions (meta.err set) like
	// successful ones. By default their trades, liquidity events, meme events,
	// ALT events and transfers are not returned because they were reverted; fee,
	// signer, balance changes and TxStatus=failed are still filled.
	IncludeFailedTxs bool `json:"includeFailedTxs,omitempty"`

	// AddressLookupTables resolves address lookup table accounts of v0
	// transactions that carry no meta.loadedAddresses (pre-execution data such
	// as shreds): table address -> table contents (all addresses, in order).
	AddressLookupTables map[string][]string `json:"-"`

	// ALTsFetcher if set, resolves address lookup tables that are not in
	// AddressLookupTables when a v0 transaction carries no meta.loadedAddresses
	ALTsFetcher *ALTsFetcher `json:"-"`

	// TokenAccountsFetcher if set, resolves mint, owner and decimals of token
	// accounts used by token transfers that the transaction's token balances do
	// not describe (for example transactions without meta)
	TokenAccountsFetcher *TokenAccountsFetcher `json:"-"`

	// PoolInfoFetcher is not used by any parser.
	//
	// Deprecated: pool data needs protocol-specific decoding that the parsers do
	// not perform; the field is kept for API compatibility and has no effect.
	PoolInfoFetcher *PoolInfoFetcher `json:"-"`
}

// DefaultParseConfig returns default parsing configuration with all events
// enabled, including the aggregated trade (this is what a nil config means)
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

// GetEffectiveParseType returns the effective ParseType. When ParseType is
// unset, everything is parsed and AggregateTrade follows the legacy
// AggregateTrades field; when it is set, it is used as is with AggregateTrade
// = ParseType.AggregateTrade || AggregateTrades.
func (c *ParseConfig) GetEffectiveParseType() ParseType {
	if c.IsParseTypeSet() {
		pt := c.ParseType
		pt.AggregateTrade = pt.AggregateTrade || c.AggregateTrades
		return pt
	}
	pt := ParseAll()
	pt.AggregateTrade = c.AggregateTrades
	return pt
}
