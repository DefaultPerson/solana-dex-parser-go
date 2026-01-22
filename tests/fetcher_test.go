package tests

import (
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

func TestFetchFilterType(t *testing.T) {
	tests := []struct {
		filter   types.FetchFilterType
		expected string
	}{
		{types.FetchFilterAll, "all"},
		{types.FetchFilterProgram, "program"},
		{types.FetchFilterAccount, "account"},
	}

	for _, tt := range tests {
		if string(tt.filter) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.filter)
		}
	}
}

func TestAddressTableLookup(t *testing.T) {
	lookup := types.AddressTableLookup{
		AccountKey:      "TableKey123",
		WritableIndexes: []int{0, 1, 2},
		ReadonlyIndexes: []int{3, 4},
	}

	if lookup.AccountKey != "TableKey123" {
		t.Errorf("Expected AccountKey 'TableKey123', got '%s'", lookup.AccountKey)
	}
	if len(lookup.WritableIndexes) != 3 {
		t.Errorf("Expected 3 WritableIndexes, got %d", len(lookup.WritableIndexes))
	}
	if len(lookup.ReadonlyIndexes) != 2 {
		t.Errorf("Expected 2 ReadonlyIndexes, got %d", len(lookup.ReadonlyIndexes))
	}
}

func TestLoadedAddresses(t *testing.T) {
	loaded := types.LoadedAddresses{
		Writable: []string{"writable1", "writable2"},
		Readonly: []string{"readonly1"},
	}

	if len(loaded.Writable) != 2 {
		t.Errorf("Expected 2 Writable addresses, got %d", len(loaded.Writable))
	}
	if len(loaded.Readonly) != 1 {
		t.Errorf("Expected 1 Readonly address, got %d", len(loaded.Readonly))
	}
}

func TestALTsFetcher(t *testing.T) {
	// Test that ALTsFetcher can be created with a custom fetch function
	fetcher := types.NewALTsFetcher(
		types.FetchFilterProgram,
		func(alts []types.AddressTableLookup) (map[string]*types.LoadedAddresses, error) {
			result := make(map[string]*types.LoadedAddresses)
			for _, alt := range alts {
				result[alt.AccountKey] = &types.LoadedAddresses{
					Writable: []string{"test1"},
					Readonly: []string{"test2"},
				}
			}
			return result, nil
		},
	)

	if fetcher.Filter != types.FetchFilterProgram {
		t.Errorf("Expected filter FetchFilterProgram, got %s", fetcher.Filter)
	}

	// Test the fetch function
	lookups := []types.AddressTableLookup{
		{AccountKey: "key1"},
		{AccountKey: "key2"},
	}

	result, err := fetcher.Fetch(lookups)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}
	if result["key1"] == nil {
		t.Error("Expected result for 'key1'")
	}
}

func TestTokenAccountsFetcher(t *testing.T) {
	fetcher := types.NewTokenAccountsFetcher(
		types.FetchFilterAll,
		func(keys []string) ([]*types.TokenAccountInfo, error) {
			result := make([]*types.TokenAccountInfo, len(keys))
			for i, key := range keys {
				result[i] = &types.TokenAccountInfo{
					Mint:     "mint123",
					Owner:    key,
					Amount:   "1000000",
					Decimals: 6,
				}
			}
			return result, nil
		},
	)

	if fetcher.Filter != types.FetchFilterAll {
		t.Errorf("Expected filter FetchFilterAll, got %s", fetcher.Filter)
	}

	result, err := fetcher.Fetch([]string{"account1"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}
	if result[0].Mint != "mint123" {
		t.Errorf("Expected mint 'mint123', got '%s'", result[0].Mint)
	}
}

func TestPoolInfoFetcher(t *testing.T) {
	type PoolData struct {
		TokenAMint string
		TokenBMint string
	}

	fetcher := types.NewPoolInfoFetcher(
		types.FetchFilterAccount,
		func(pools []string) ([]interface{}, error) {
			result := make([]interface{}, len(pools))
			for i := range pools {
				result[i] = &PoolData{
					TokenAMint: "tokenA",
					TokenBMint: "tokenB",
				}
			}
			return result, nil
		},
	)

	if fetcher.Filter != types.FetchFilterAccount {
		t.Errorf("Expected filter FetchFilterAccount, got %s", fetcher.Filter)
	}

	result, err := fetcher.Fetch([]string{"pool1"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}
	poolData, ok := result[0].(*PoolData)
	if !ok {
		t.Error("Expected PoolData type")
	}
	if poolData.TokenAMint != "tokenA" {
		t.Errorf("Expected TokenAMint 'tokenA', got '%s'", poolData.TokenAMint)
	}
}

func TestTokenAccountInfo(t *testing.T) {
	info := types.TokenAccountInfo{
		Mint:     "So11111111111111111111111111111111111111112",
		Owner:    "owner123",
		Amount:   "1000000000",
		Decimals: 9,
	}

	if info.Mint != "So11111111111111111111111111111111111111112" {
		t.Errorf("Expected SOL mint, got '%s'", info.Mint)
	}
	if info.Decimals != 9 {
		t.Errorf("Expected 9 decimals, got %d", info.Decimals)
	}
}
