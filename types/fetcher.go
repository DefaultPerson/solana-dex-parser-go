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

// ALTsFetcher provides pluggable Address Lookup Table resolution
type ALTsFetcher struct {
	// Filter specifies when to invoke the fetcher
	Filter FetchFilterType

	// Fetch resolves ALT references to actual addresses
	// Input: slice of ALT lookup references from transaction
	// Output: map of ALT account key -> LoadedAddresses
	Fetch func(alts []AddressTableLookup) (map[string]*LoadedAddresses, error)
}

// TokenAccountInfo contains token account metadata
type TokenAccountInfo struct {
	Mint     string `json:"mint"`
	Owner    string `json:"owner"`
	Amount   string `json:"amount"`
	Decimals uint8  `json:"decimals"`
}

// TokenAccountsFetcher provides pluggable token account info resolution
type TokenAccountsFetcher struct {
	// Filter specifies when to invoke the fetcher
	Filter FetchFilterType

	// Fetch retrieves token account information for given account keys
	// Input: slice of token account public keys
	// Output: slice of TokenAccountInfo (nil for accounts that couldn't be fetched)
	Fetch func(accountKeys []string) ([]*TokenAccountInfo, error)
}

// PoolInfoFetcher provides pluggable pool information resolution
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

// NewPoolInfoFetcher creates a new pool info fetcher with specified filter and function
func NewPoolInfoFetcher(
	filter FetchFilterType,
	fetcher func(poolKeys []string) ([]interface{}, error),
) *PoolInfoFetcher {
	return &PoolInfoFetcher{
		Filter: filter,
		Fetch:  fetcher,
	}
}
