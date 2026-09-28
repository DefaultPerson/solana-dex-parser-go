package tests

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Helpers for the core regression tests. Expected values are computed from
// the raw transaction meta (balances), independently of the adapter.

const (
	solMint  = "So11111111111111111111111111111111111111112"
	usdcMint = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	usdtMint = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
)

// cloneTx deep-copies a transaction through JSON.
func cloneTx(t testing.TB, tx *adapter.SolanaTransaction) *adapter.SolanaTransaction {
	t.Helper()
	raw, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var c adapter.SolanaTransaction
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("unmarshal clone: %v", err)
	}
	return &c
}

// rawAccountKeys returns the account keys of a "json"-encoded fixture:
// message keys, then loaded writable and readonly addresses.
func rawAccountKeys(tx *adapter.SolanaTransaction) []string {
	var keys []string
	for _, k := range tx.Transaction.Message.AccountKeys {
		keys = append(keys, k.Pubkey)
	}
	if tx.Meta != nil && tx.Meta.LoadedAddresses != nil {
		keys = append(keys, tx.Meta.LoadedAddresses.Writable...)
		keys = append(keys, tx.Meta.LoadedAddresses.Readonly...)
	}
	return keys
}

// ownerTokenDelta sums post - pre of all token balances of owner and mint.
func ownerTokenDelta(tx *adapter.SolanaTransaction, owner, mint string) *big.Int {
	d := new(big.Int)
	add := func(bals []adapter.TokenBalance, sign int64) {
		for _, b := range bals {
			if b.Owner != owner || b.Mint != mint {
				continue
			}
			v, ok := new(big.Int).SetString(b.UiTokenAmount.Amount, 10)
			if !ok {
				continue
			}
			d.Add(d, v.Mul(v, big.NewInt(sign)))
		}
	}
	add(tx.Meta.PreTokenBalances, -1)
	add(tx.Meta.PostTokenBalances, 1)
	return d
}

// accountTokenDelta returns post - pre of one token account (by key).
func accountTokenDelta(tx *adapter.SolanaTransaction, account, mint string) *big.Int {
	keys := rawAccountKeys(tx)
	d := new(big.Int)
	add := func(bals []adapter.TokenBalance, sign int64) {
		for _, b := range bals {
			if b.AccountIndex < len(keys) && keys[b.AccountIndex] == account && b.Mint == mint {
				v, _ := new(big.Int).SetString(b.UiTokenAmount.Amount, 10)
				if v != nil {
					d.Add(d, v.Mul(v, big.NewInt(sign)))
				}
			}
		}
	}
	add(tx.Meta.PreTokenBalances, -1)
	add(tx.Meta.PostTokenBalances, 1)
	return d
}

// lamportDelta returns post - pre lamports of account.
func lamportDelta(tx *adapter.SolanaTransaction, account string) *big.Int {
	for i, k := range rawAccountKeys(tx) {
		if k == account && i < len(tx.Meta.PreBalances) && i < len(tx.Meta.PostBalances) {
			return new(big.Int).Sub(new(big.Int).SetUint64(tx.Meta.PostBalances[i]), new(big.Int).SetUint64(tx.Meta.PreBalances[i]))
		}
	}
	return new(big.Int)
}

func bigStr(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return new(big.Int)
	}
	return v
}

// tradeKey summarizes the fields of a trade that must match between encodings.
func tradeKey(tr types.TradeInfo) string {
	return tr.Idx + " " + string(tr.Type) + " " + tr.ProgramId + " " + tr.AMM + " " + tr.Route + " " +
		tr.InputToken.Mint + ":" + tr.InputToken.AmountRaw + " " + tr.OutputToken.Mint + ":" + tr.OutputToken.AmountRaw + " " + tr.User
}

// parseFixture parses a "json" fixture with config (nil = default).
func parseFixture(t testing.TB, sig string, config *types.ParseConfig) (*adapter.SolanaTransaction, *types.ParseResult) {
	t.Helper()
	tx := loadFixture(t, sig)
	result := dexparser.NewDexParser().ParseAll(tx, config)
	if result == nil {
		t.Fatalf("%s: ParseAll returned nil", sig)
	}
	return tx, result
}

// tradesConfig returns individual trades only.
func tradesConfig() *types.ParseConfig {
	return &types.ParseConfig{ParseType: types.ParseType{Trade: true}, TryUnknownDEX: true}
}

// jsonInt reads a JSON number decoded into an interface{} (json.Number or
// float64) as int.
func jsonInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case interface{ String() string }:
		return int(bigStr(n.String()).Int64())
	}
	return -1
}
