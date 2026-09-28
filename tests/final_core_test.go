package tests

import (
	"errors"
	"runtime"
	"strconv"
	"strings"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Regression tests for the core findings of the final review (core-1,
// core-2, complete-1, complete-2, robust-2, robust-3). Expected values come
// from the raw fixtures: instructions, token balances and balance changes.

// TestFinalParsedMultisigAuthority: the RPC's jsonParsed encoding names the
// authority of a transfer "multisigAuthority" whenever signer accounts follow
// it (every Token-2022 transfer-hook transfer), and the parser read only
// "authority", so the json and jsonParsed trades of 3ou2zc4g differed in the
// output's authority. Truth: account 3 of the transferChecked at 4-13 in the
// json encoding. complete-2.
func TestFinalParsedMultisigAuthority(t *testing.T) {
	const sig = "3ou2zc4g8GGfufg1yX9DJpmSPmr6Uo25LsuocezSf5swegypu1aSYtfw47ZSte6PCUBexSeA4Veim65vqaDXp8zr"
	var authority string
	for _, ix := range fixtureIxs(t, loadFixture(t, sig)) {
		if ix.outer == 4 && ix.inner == 13 && ix.programId == constants.TOKEN_2022_PROGRAM_ID && len(ix.accounts) > 4 {
			authority = ix.accounts[3]
		}
	}
	if authority != "FhVo3mqL8PW5pH5U2CN4XE33DokiyZnUwuGpH2hmHLuM" {
		t.Fatalf("fixture: authority of 4-13 %q", authority)
	}
	parser := dexparser.NewDexParser()
	for encoding, tx := range map[string]*adapterTx{"json": loadFixture(t, sig), "jsonParsed": loadParsedFixture(t, sig)} {
		res := parser.ParseAll(tx, nil)
		found := false
		for _, tr := range res.Trades {
			if tr.Idx == "4-11" {
				found = true
				if tr.OutputToken.Authority != authority {
					t.Errorf("%s: trade 4-11 output authority %q, want %s", encoding, tr.OutputToken.Authority, authority)
				}
			}
		}
		if !found {
			t.Errorf("%s: no trade at 4-11", encoding)
		}
	}
}

// TestFinalBatchBoundedWorkers: the concurrent batch started one goroutine
// per transaction, each waiting on a semaphore, so a large batch held
// thousands of goroutine stacks whatever maxWorkers was. Truth:
// runtime.NumGoroutine sampled in the callback stays within maxWorkers of the
// baseline. robust-2.
func TestFinalBatchBoundedWorkers(t *testing.T) {
	tx := loadFixture(t, "Z4CChBawHPHwuUd9JmpvtyaddghjoeyxioFXB2HK4NSioKV5tipgajmznRbzx9LhiEdvYL5hedXWnEqEzmYVsGA")
	txs := make([]*adapterTx, 3000)
	for i := range txs {
		txs[i] = tx
	}
	const workers = 4
	baseline := runtime.NumGoroutine()
	peak := 0
	results := dexparser.NewDexParser().ParseBatchWithCallback(txs, nil, workers, func(int, *adapterTx, *types.ParseResult, error) bool {
		if n := runtime.NumGoroutine(); n > peak {
			peak = n
		}
		return true
	})
	if len(results) != len(txs) {
		t.Fatalf("%d results for %d transactions", len(results), len(txs))
	}
	if peak > baseline+workers+1 {
		t.Errorf("peak goroutines %d, baseline %d: want at most %d workers", peak, baseline, workers)
	}
}

// TestFinalBatchEarlyStopResults: after the callback stopped a batch, the
// unparsed entries were nil although ParseAll never returns nil and the docs
// promise one result per transaction; callers iterating the results
// dereferenced nil. They are now failed results with BatchSkippedMsg.
// robust-2.
func TestFinalBatchEarlyStopResults(t *testing.T) {
	tx := loadFixture(t, "Z4CChBawHPHwuUd9JmpvtyaddghjoeyxioFXB2HK4NSioKV5tipgajmznRbzx9LhiEdvYL5hedXWnEqEzmYVsGA")
	txs := make([]*adapterTx, 200)
	for i := range txs {
		txs[i] = tx
	}
	for _, workers := range []int{1, 4} {
		results := dexparser.NewDexParser().ParseBatchWithCallback(txs, nil, workers, func(int, *adapterTx, *types.ParseResult, error) bool {
			return false
		})
		parsed, skipped := 0, 0
		for i, r := range results {
			switch {
			case r == nil:
				t.Fatalf("workers=%d: result %d is nil", workers, i)
			case r.State:
				parsed++
			case r.Msg == dexparser.BatchSkippedMsg && r.Signature == tx.Transaction.Signatures[0]:
				skipped++
			default:
				t.Errorf("workers=%d: result %d State=false Msg=%q", workers, i, r.Msg)
			}
		}
		if parsed == 0 || skipped == 0 || parsed+skipped != len(txs) {
			t.Errorf("workers=%d: %d parsed, %d skipped of %d", workers, parsed, skipped, len(txs))
		}
	}
}

// TestFinalFetcherRedaction: fetcher failures reach ParseResult.Warnings with
// URLs cut to scheme and host, but some URL forms leaked their secret (a
// quoted query value, a URL without scheme, a percent-encoded URL, a url.URL
// dump), and a fetcher that panicked failed the whole parse with the panic
// text, URL included, in Msg. Truth: no warning or message contains the
// secret, and a panicking fetcher is a warning, not a failed parse. robust-3.
func TestFinalFetcherRedaction(t *testing.T) {
	const secret = "SECRETXYZ"
	stripped := cloneTx(t, loadFixture(t, sigTwoLookups))
	stripped.Meta.LoadedAddresses = nil
	check := func(name string, fetch func([]types.AddressTableLookup) (map[string]*types.LoadedAddresses, error)) {
		t.Helper()
		cfg := types.DefaultParseConfig()
		cfg.ALTsFetcher = types.NewALTsFetcher(types.FetchFilterAll, fetch)
		dex := dexparser.NewDexParser().ParseAll(stripped, &cfg)
		shred := dexparser.NewShredParser().ParseAll(stripped, &cfg)
		for parser, r := range map[string][]string{
			"DexParser":   append([]string{strconv.FormatBool(dex.State), dex.Msg}, dex.Warnings...),
			"ShredParser": append([]string{strconv.FormatBool(shred.State), shred.Msg}, shred.Warnings...),
		} {
			text := strings.Join(r[1:], " ")
			if r[0] != "true" || strings.Contains(text, secret) || !strings.Contains(text, "ALTsFetcher: ") {
				t.Errorf("%s %s: State=%s Msg and Warnings %q", name, parser, r[0], r[1:])
			}
		}
	}
	for name, msg := range map[string]string{
		"quoted value":   "get https://host.example/?key='" + secret + "'",
		"no scheme":      "request to rpc.example/?api-key=" + secret + " failed",
		"percent-encode": "bad url https%3A%2F%2Frpc.example%2F%3Fapi-key%3D" + secret,
		"url dump":       `&url.URL{Scheme:"https", Host:"rpc.example", RawQuery:"api-key=` + secret + `"}`,
	} {
		msg := msg
		check(name, func([]types.AddressTableLookup) (map[string]*types.LoadedAddresses, error) {
			return nil, errors.New(msg)
		})
	}
	check("panic", func([]types.AddressTableLookup) (map[string]*types.LoadedAddresses, error) {
		panic(`Post "https://mainnet.helius-rpc.com/?api-key=` + secret + `": EOF`)
	})
}
