package tests

import (
	"errors"
	"math/big"
	"runtime"
	"strconv"
	"strings"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
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

// ownerTransfers sums the inner SPL Token transfers (transfer and
// transferChecked) of outer instruction outer by mint: those out of token
// accounts owned by owner and those into them, from the raw fixture
func ownerTransfers(t *testing.T, tx *adapterTx, outer int, owner string) (out, in map[string]*big.Int) {
	t.Helper()
	out, in = map[string]*big.Int{}, map[string]*big.Int{}
	add := func(m map[string]*big.Int, mint string, amount uint64) {
		if m[mint] == nil {
			m[mint] = new(big.Int)
		}
		m[mint].Add(m[mint], new(big.Int).SetUint64(amount))
	}
	for _, ix := range fixtureIxs(t, tx) {
		if ix.outer != outer || ix.inner < 0 || ix.programId != constants.TOKEN_PROGRAM_ID || len(ix.data) < 9 {
			continue
		}
		var source, destination string
		switch {
		case ix.data[0] == 3 && len(ix.accounts) >= 3: // transfer: source, destination, authority
			source, destination = ix.accounts[0], ix.accounts[1]
		case ix.data[0] == 12 && len(ix.accounts) >= 4: // transferChecked: source, mint, destination, authority
			source, destination = ix.accounts[0], ix.accounts[2]
		default:
			continue
		}
		mint := tokenBalanceMint(tx, source)
		if b := tokenBalance(tx, source); b != nil && b.Owner == owner {
			add(out, mint, le64At(ix.data, 1))
		}
		if b := tokenBalance(tx, destination); b != nil && b.Owner == owner {
			add(in, mint, le64At(ix.data, 1))
		}
	}
	return out, in
}

// TestFinalFallbackNoPhantomTrades: the unknown-DEX fallback turned transfer
// groups in which nobody both sends and receives into trades: a Raydium CLMM
// zap through a wrapper (2Bm6, the position NFT as output), a two-token
// deposit (4Vp) and a payout that also corrupted the aggregate (47Cb).
// Truth: in 2Bm6 and 4Vp the signer receives no token at all (only
// outflows), so there is no trade; in 47Cb the only swap is the Jupiter
// route of outer 3, whose owner FbVd1f sends USDC and receives cbBTC.
// core-1, complete-1.
func TestFinalFallbackNoPhantomTrades(t *testing.T) {
	for _, sig := range []string{
		"2Bm6Xh3UQYCYywCQPEJ1tCQKrSPHRPGukN4qTmutCYnR45PKMBhTuBjMdbT7x3fsLnqvduAeiTQeVRQ76NA2NMGN",
		"4VpDFKjjyBNjS3amzzqW22mx1aLNQnYwm1EyZT4kNkHrDpfNCfuZWjm8UL4br5ReNdVUjTmFdqgoUBX2eKuhVndt",
	} {
		tx, res := parseFixture(t, sig, nil)
		signer := res.Signer[0]
		for _, b := range append(append([]adapter.TokenBalance{}, tx.Meta.PreTokenBalances...), tx.Meta.PostTokenBalances...) {
			if b.Owner == signer && ownerTokenDelta(tx, signer, b.Mint).Sign() > 0 {
				t.Fatalf("%s: the signer receives %s", sig[:8], b.Mint)
			}
		}
		if len(res.Trades) != 0 || res.AggregateTrade != nil {
			t.Errorf("%s: %d trades (first %+v), aggregate %v: want none", sig[:8], len(res.Trades), res.Trades, res.AggregateTrade != nil)
		}
		if len(res.Liquidities) == 0 {
			t.Errorf("%s: the liquidity event is gone", sig[:8])
		}
	}

	const sig = "47CbsEeriB5JtFmSVU61NS1thpD1HabJ7AGzzwxiViQqUNQJxrJXtaBU1EeBCLWtbQE7whvKVBjxXa2ELCMc16wp"
	const cbbtc = "cbbtcf3aa214zXHbiAZQwf4122FBYbraNdFqgw4iMij"
	tx, res := parseFixture(t, sig, nil)
	out, in := ownerTransfers(t, tx, 3, "FbVd1fsYKpEj1Bzupbjo2VGJyfgLU9aw4r8U5uuR8v6s")
	for _, tr := range res.Trades {
		if tr.ProgramId == "satRushGBRY2vgapeTAkoxz26vL2cYqyPi6CnBj7Tco" {
			t.Errorf("47CbsEer: phantom satRush trade %s", tr.Idx)
		}
	}
	agg := res.AggregateTrade
	if agg == nil || agg.InputToken.Mint != usdcMint || agg.InputToken.AmountRaw != out[usdcMint].String() ||
		agg.OutputToken.Mint != cbbtc || agg.OutputToken.AmountRaw != in[cbbtc].String() {
		t.Errorf("47CbsEer: aggregate %+v, want USDC %s -> cbBTC %s", agg, out[usdcMint], in[cbbtc])
	}
}

// TestFinalFallbackDirectionOtherSigner: the fallback took the swap
// direction from the fee payer only, so an unparsed venue hop whose tokens
// belong to the transaction's second signer (2BL4: Deriverse inside a DFlow
// route) came out reversed, and the aggregate summed the intermediate USDT
// twice. Truth: the transfers of outer 2 at 2-6/2-7: the second signer
// 79Tv3i sends USDC 3556827 and receives USDT 3557271, which the next hop
// spends. core-2.
func TestFinalFallbackDirectionOtherSigner(t *testing.T) {
	const sig = "2BL4GjmzqMy262NMWA8JdMzhoxnmPwabfQjvQBAnZHAcTsVTdZiYfHqvi4SdfjRKN3M3oH71ZVu5iHGrhFubaPfK"
	tx, res := parseFixture(t, sig, nil)
	party := res.Signer[1]
	var usdcSent, usdtReceived *big.Int
	for _, ix := range fixtureIxs(t, tx) {
		// SPL Token transfer: source, destination, authority
		if ix.outer != 2 || ix.programId != constants.TOKEN_PROGRAM_ID || len(ix.data) != 9 || ix.data[0] != 3 {
			continue
		}
		switch ix.inner {
		case 7: // USDC signed by the party
			if tokenBalanceMint(tx, ix.accounts[0]) == usdcMint && ix.accounts[2] == party {
				usdcSent = new(big.Int).SetUint64(le64At(ix.data, 1))
			}
		case 6: // USDT into the party's associated token account, which
			// the transaction creates (ATA create: payer, account, owner,
			// mint) and closes
			for _, ata := range fixtureIxs(t, tx) {
				if ata.programId == "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL" && len(ata.accounts) >= 4 &&
					ata.accounts[1] == ix.accounts[1] && ata.accounts[2] == party && ata.accounts[3] == usdtMint {
					usdtReceived = new(big.Int).SetUint64(le64At(ix.data, 1))
				}
			}
		}
	}
	if usdcSent == nil || usdtReceived == nil {
		t.Fatal("fixture: Deriverse legs not found")
	}
	var hop *types.TradeInfo
	for i := range res.Trades {
		if res.Trades[i].ProgramId == "DRVSpZ2YUYYKgZP8XtLhAGtT1zYSCKzeHfb4DgRnrgqD" {
			hop = &res.Trades[i]
		}
	}
	if hop == nil || hop.InputToken.Mint != usdcMint || hop.InputToken.AmountRaw != usdcSent.String() ||
		hop.OutputToken.Mint != usdtMint || hop.OutputToken.AmountRaw != usdtReceived.String() || hop.User != party {
		t.Errorf("Deriverse hop %+v, want %s: USDC %s -> USDT %s", hop, party, usdcSent, usdtReceived)
	}
	if agg := res.AggregateTrade; agg == nil || agg.InputToken.Mint != usdcMint || agg.InputToken.AmountRaw != usdcSent.String() {
		t.Errorf("aggregate %+v, want the input USDC %s", agg, usdcSent)
	}
}

// TestFinalFallbackDepositWithRefunds: a user who deposits two tokens and
// gets part of each back both sends and receives, so without more care the
// fallback would take the deposit of one token and the refund of the other
// for a swap. The transaction is built from the real 4VpDFKjj (a USDC + USDT
// deposit through the ensSuX wrapper, refunds at 3-5/3-6) by removing the
// Raydium CLMM instruction at 3-2, so that the refunds fall into the
// wrapper's transfer group as they would without a CPI in between. Truth:
// the user sends each token before receiving any of it back, a deposit, not
// a swap. core-1.
func TestFinalFallbackDepositWithRefunds(t *testing.T) {
	tx := cloneTx(t, loadFixture(t, "4VpDFKjjyBNjS3amzzqW22mx1aLNQnYwm1EyZT4kNkHrDpfNCfuZWjm8UL4br5ReNdVUjTmFdqgoUBX2eKuhVndt"))
	keys := rawAccountKeys(tx)
	removed := false
	for s := range tx.Meta.InnerInstructions {
		set := &tx.Meta.InnerInstructions[s]
		if set.Index != 3 || len(set.Instructions) != 7 {
			continue
		}
		m, _ := set.Instructions[2].(map[string]interface{})
		if m != nil && keys[jsonInt(m["programIdIndex"])] == constants.DEX_PROGRAMS.RAYDIUM_CL.ID {
			set.Instructions = append(set.Instructions[:2:2], set.Instructions[3:]...)
			removed = true
		}
	}
	if !removed {
		t.Fatal("fixture: no Raydium CLMM instruction at 3-2")
	}
	res := dexparser.NewDexParser().ParseAll(tx, nil)
	for _, tr := range res.Trades {
		t.Errorf("trade %s %s %s %s -> %s %s", tr.Idx, tr.AMM, tr.InputToken.AmountRaw, tr.InputToken.Mint, tr.OutputToken.AmountRaw, tr.OutputToken.Mint)
	}
}
