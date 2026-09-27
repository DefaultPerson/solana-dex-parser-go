package tests

import (
	"reflect"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Regression tests for address lookup table resolution and the fetchers.
// shred-18, core-14.

// lookupTables rebuilds the table contents a v0 fixture used, from its
// meta.loadedAddresses: writable addresses in lookup order, then readonly
// ones, placed at the lookup's indexes. Other table slots are left empty
// (they are not referenced).
func lookupTables(t *testing.T, tx *adapter.SolanaTransaction) map[string][]string {
	t.Helper()
	tables := map[string][]string{}
	place := func(table string, idx int, addr string) {
		for len(tables[table]) <= idx {
			tables[table] = append(tables[table], "")
		}
		tables[table][idx] = addr
	}
	w, r := 0, 0
	for _, l := range tx.Transaction.Message.AddressTableLookups {
		for _, idx := range l.WritableIndexes {
			place(l.AccountKey, idx, tx.Meta.LoadedAddresses.Writable[w])
			w++
		}
	}
	for _, l := range tx.Transaction.Message.AddressTableLookups {
		for _, idx := range l.ReadonlyIndexes {
			place(l.AccountKey, idx, tx.Meta.LoadedAddresses.Readonly[r])
			r++
		}
	}
	if w != len(tx.Meta.LoadedAddresses.Writable) || r != len(tx.Meta.LoadedAddresses.Readonly) {
		t.Fatalf("lookups do not match loadedAddresses")
	}
	return tables
}

// TestCoreAddressLookupTables: v0 transactions without meta.loadedAddresses
// (pre-execution data) had "" for every lookup account and no way to supply
// the tables. shred-18.
func TestCoreAddressLookupTables(t *testing.T) {
	p := dexparser.NewDexParser()
	orig := loadFixture(t, sigTwoLookups)
	if len(orig.Transaction.Message.AddressTableLookups) != 2 {
		t.Fatal("fixture should have two lookups")
	}
	tables := lookupTables(t, orig)
	wantKeys := rawAccountKeys(orig)
	want := p.ParseAll(orig, nil)
	if len(want.Trades) == 0 {
		t.Fatal("fixture has no trades")
	}

	stripped := cloneTx(t, orig)
	stripped.Meta.LoadedAddresses = nil

	t.Run("tables in config", func(t *testing.T) {
		cfg := types.DefaultParseConfig()
		cfg.AddressLookupTables = tables
		a := adapter.NewTransactionAdapter(stripped, &cfg)
		if a.HasUnresolvedAccounts() || !reflect.DeepEqual(a.AccountKeys, wantKeys) {
			t.Errorf("AccountKeys = %v, want %v", a.AccountKeys, wantKeys)
		}
		if got := p.ParseAll(stripped, &cfg); !sameEvents(got, want) {
			t.Errorf("trades %v, want %v", got.Trades, want.Trades)
		}
	})

	t.Run("fetcher", func(t *testing.T) {
		calls := 0
		cfg := types.DefaultParseConfig()
		cfg.ALTsFetcher = types.NewALTsFetcher(types.FetchFilterAll, func(alts []types.AddressTableLookup) (map[string]*types.LoadedAddresses, error) {
			calls++
			out := map[string]*types.LoadedAddresses{}
			for _, l := range alts {
				la := &types.LoadedAddresses{}
				for _, i := range l.WritableIndexes {
					la.Writable = append(la.Writable, tables[l.AccountKey][i])
				}
				for _, i := range l.ReadonlyIndexes {
					la.Readonly = append(la.Readonly, tables[l.AccountKey][i])
				}
				out[l.AccountKey] = la
			}
			return out, nil
		})
		a := adapter.NewTransactionAdapter(stripped, &cfg)
		if calls != 1 || a.HasUnresolvedAccounts() || !reflect.DeepEqual(a.AccountKeys, wantKeys) {
			t.Errorf("calls=%d AccountKeys = %v, want %v", calls, a.AccountKeys, wantKeys)
		}

		// a table given in the config is not fetched again
		first := stripped.Transaction.Message.AddressTableLookups[0].AccountKey
		cfg.AddressLookupTables = map[string][]string{first: tables[first]}
		var fetched []string
		inner := cfg.ALTsFetcher.Fetch
		cfg.ALTsFetcher = types.NewALTsFetcher(types.FetchFilterAll, func(alts []types.AddressTableLookup) (map[string]*types.LoadedAddresses, error) {
			for _, l := range alts {
				fetched = append(fetched, l.AccountKey)
			}
			return inner(alts)
		})
		a = adapter.NewTransactionAdapter(stripped, &cfg)
		if len(fetched) != 1 || fetched[0] == first || !reflect.DeepEqual(a.AccountKeys, wantKeys) {
			t.Errorf("fetched %v; AccountKeys resolved=%v", fetched, reflect.DeepEqual(a.AccountKeys, wantKeys))
		}

		// the fetcher filter is honoured
		cfg.AddressLookupTables = nil
		calls = 0
		cfg.ALTsFetcher = types.NewALTsFetcher(types.FetchFilterProgram, func(alts []types.AddressTableLookup) (map[string]*types.LoadedAddresses, error) {
			calls++
			return nil, nil
		})
		cfg.ProgramIds = []string{"NoSuchProgram1111111111111111111111111111111"}
		adapter.NewTransactionAdapter(stripped, &cfg)
		if calls != 0 {
			t.Errorf("FetchFilterProgram without a matching program: fetcher called %d times", calls)
		}
	})

	t.Run("pre-execution (no meta)", func(t *testing.T) {
		noMeta := cloneTx(t, orig)
		noMeta.Meta = nil
		cfg := types.DefaultParseConfig()
		cfg.AddressLookupTables = tables
		a := adapter.NewTransactionAdapter(noMeta, &cfg)
		if a.HasUnresolvedAccounts() || !reflect.DeepEqual(a.AccountKeys, wantKeys) {
			t.Errorf("AccountKeys = %v", a.AccountKeys)
		}
		// the instruction accounts of the Jupiter route resolve to the same keys
		oa := adapter.NewTransactionAdapter(orig, nil)
		for i := range orig.Transaction.Message.Instructions {
			if !reflect.DeepEqual(a.GetInstructionAccounts(a.InstructionAt(i)), oa.GetInstructionAccounts(oa.InstructionAt(i))) {
				t.Errorf("instruction %d accounts differ", i)
			}
		}
	})

	t.Run("unresolved", func(t *testing.T) {
		a := adapter.NewTransactionAdapter(stripped, nil)
		if !a.HasUnresolvedAccounts() || len(a.AccountKeys) != len(wantKeys) {
			t.Errorf("HasUnresolvedAccounts=%v keys=%d want %d", a.HasUnresolvedAccounts(), len(a.AccountKeys), len(wantKeys))
		}
		// a table too short for an index leaves that account unresolved
		cfg := types.DefaultParseConfig()
		short := map[string][]string{}
		for k, v := range tables {
			short[k] = v[:1]
		}
		cfg.AddressLookupTables = short
		if a := adapter.NewTransactionAdapter(stripped, &cfg); !a.HasUnresolvedAccounts() {
			t.Error("short table: HasUnresolvedAccounts() = false")
		}
	})
}

// TestCoreTokenAccountsFetcher: TokenAccountsFetcher was never called.
// Token accounts the transaction does not describe are resolved through it.
// core-14.
//
// Synthetic: built from the real 51nj5GtA by removing what reveals the user's
// output token account (its balances, its InitializeAccount3 and the mint in
// the transfer), as for data without that account's creation.
func TestCoreTokenAccountsFetcher(t *testing.T) {
	const dest = "4EHZFwbbVsHzsCN2QxoKgrqSHr41z5LCYxGhMeZ9mdXo"
	orig := loadFixture(t, sigT22Fee)
	var info types.TokenAccountInfo
	for _, b := range orig.Meta.PostTokenBalances {
		if rawAccountKeys(orig)[b.AccountIndex] == dest {
			info = types.TokenAccountInfo{Mint: b.Mint, Owner: b.Owner, Amount: b.UiTokenAmount.Amount, Decimals: b.UiTokenAmount.Decimals}
		}
	}
	if info.Mint == "" {
		t.Fatal("fixture: destination has no post balance")
	}

	tx := hideTokenAccount(t, orig, dest)
	var asked []string
	cfg := types.DefaultParseConfig()
	cfg.TokenAccountsFetcher = types.NewTokenAccountsFetcher(types.FetchFilterAll, func(keys []string) ([]*types.TokenAccountInfo, error) {
		asked = append(asked, keys...)
		out := make([]*types.TokenAccountInfo, len(keys))
		for i, k := range keys {
			if k == dest {
				c := info
				out[i] = &c
			}
		}
		return out, nil
	})
	a := adapter.NewTransactionAdapter(tx, &cfg)
	if !containsStr(asked, dest) {
		t.Fatalf("fetcher asked for %v, want %s", asked, dest)
	}
	if a.IsGuessedTokenAccount(dest) || a.GetSplTokenMint(dest) != info.Mint || a.GetTokenAccountOwner(dest) != info.Owner {
		t.Errorf("after fetch: mint %s owner %s", a.GetSplTokenMint(dest), a.GetTokenAccountOwner(dest))
	}
	if b := adapter.NewTransactionAdapter(tx, nil); !b.IsGuessedTokenAccount(dest) {
		t.Error("without the fetcher the account should stay unknown")
	}
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
