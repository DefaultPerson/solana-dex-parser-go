package types

// FetchFilterType specifies when to invoke a fetcher callback
type FetchFilterType string

const (
	// FetchFilterAll calls fetcher for all transactions
	FetchFilterAll FetchFilterType = "all"

	// FetchFilterProgram calls fetcher only for transactions matching ParseConfig.ProgramIds
	FetchFilterProgram FetchFilterType = "program"

	// FetchFilterAccount calls fetcher only for transactions matching ParseConfig.AccountInclude
	FetchFilterAccount FetchFilterType = "account"
)

// LoadedAddresses contains resolved addresses from Address Lookup Tables
type LoadedAddresses struct {
	Writable []string `json:"writable"`
	Readonly []string `json:"readonly"`
}

// AddressTableLookup represents an address lookup table reference in transaction
type AddressTableLookup struct {
	AccountKey      string `json:"accountKey"`
	WritableIndexes []int  `json:"writableIndexes"`
	ReadonlyIndexes []int  `json:"readonlyIndexes"`
}

// ALTsFetcher provides pluggable Address Lookup Table resolution for v0
// transactions without meta.loadedAddresses. It is called once per
// transaction, for the lookups not covered by ParseConfig.AddressLookupTables.
type ALTsFetcher struct {
	// Filter specifies when to invoke the fetcher (FetchFilterProgram and
	// FetchFilterAccount are checked against the static account keys)
	Filter FetchFilterType

	// Fetch resolves ALT references to actual addresses
	// Input: slice of ALT lookup references from transaction
	// Output: map of ALT account key -> LoadedAddresses, where Writable holds
	// the addresses at the lookup's WritableIndexes and Readonly those at its
	// ReadonlyIndexes, in index order. On error the accounts stay unresolved
	// and the error text goes into ParseResult.Warnings (URLs cut to scheme
	// and host), so do not put other secrets in it.
	Fetch func(alts []AddressTableLookup) (map[string]*LoadedAddresses, error)
}

// TokenAccountInfo contains token account metadata
type TokenAccountInfo struct {
	Mint     string `json:"mint"`
	Owner    string `json:"owner"`
	Amount   string `json:"amount"`
	Decimals uint8  `json:"decimals"`
}

// TokenAccountsFetcher provides pluggable token account info resolution. It
// is called once per transaction with the token accounts used by token
// program instructions whose mint the transaction does not reveal (no token
// balance entry and no mint in the instruction).
type TokenAccountsFetcher struct {
	// Filter specifies when to invoke the fetcher (FetchFilterProgram and
	// FetchFilterAccount are checked against the transaction's account keys)
	Filter FetchFilterType

	// Fetch retrieves token account information for given account keys
	// Input: slice of token account public keys
	// Output: slice of TokenAccountInfo in the same order (nil for accounts that couldn't be fetched)
	// On error the error text goes into ParseResult.Warnings (URLs cut to
	// scheme and host), so do not put other secrets in it.
	Fetch func(accountKeys []string) ([]*TokenAccountInfo, error)
}

// PoolInfoFetcher provides pluggable pool information resolution.
//
// Deprecated: no parser uses it (see ParseConfig.PoolInfoFetcher).
type PoolInfoFetcher struct {
	// Filter specifies when to invoke the fetcher
	Filter FetchFilterType

	// Fetch retrieves pool information for given pool keys
	// Input: slice of pool public keys
	// Output: slice of pool info (interface{} to support different pool types)
	Fetch func(poolKeys []string) ([]interface{}, error)
}

// NewALTsFetcher creates a new ALTs fetcher with specified filter and function
func NewALTsFetcher(
	filter FetchFilterType,
	fetcher func(alts []AddressTableLookup) (map[string]*LoadedAddresses, error),
) *ALTsFetcher {
	return &ALTsFetcher{
		Filter: filter,
		Fetch:  fetcher,
	}
}

// NewTokenAccountsFetcher creates a new token accounts fetcher with specified filter and function
func NewTokenAccountsFetcher(
	filter FetchFilterType,
	fetcher func(accountKeys []string) ([]*TokenAccountInfo, error),
) *TokenAccountsFetcher {
	return &TokenAccountsFetcher{
		Filter: filter,
		Fetch:  fetcher,
	}
}

// NewPoolInfoFetcher creates a new pool info fetcher with specified filter and function.
//
// Deprecated: see PoolInfoFetcher.
func NewPoolInfoFetcher(
	filter FetchFilterType,
	fetcher func(poolKeys []string) ([]interface{}, error),
) *PoolInfoFetcher {
	return &PoolInfoFetcher{
		Filter: filter,
		Fetch:  fetcher,
	}
}
