package tests

import (
	"errors"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// utils.GetShredTradeType: SWAP when a mint is unknown or both are equal,
// else utils.GetTradeType.
func TestIntegGetShredTradeType(t *testing.T) {
	const token = "Ce2gx9KGXJ6C9Mp5b5x1sn9Mg87JwEbrQby4Zqo3pump"
	cases := []struct {
		in, out string
		want    types.TradeType
	}{
		{"", token, types.TradeTypeSwap},
		{solMint, "", types.TradeTypeSwap},
		{usdcMint, usdcMint, types.TradeTypeSwap},
		{solMint, token, types.TradeTypeBuy},
		{token, solMint, types.TradeTypeSell},
		{usdcMint, token, types.TradeTypeBuy},
		{token, usdcMint, types.TradeTypeSell},
	}
	for _, c := range cases {
		if got := utils.GetShredTradeType(c.in, c.out); got != c.want {
			t.Errorf("GetShredTradeType(%q, %q) = %s, want %s", c.in, c.out, got, c.want)
		}
	}
}

// L1: in a Raydium AMM v4 pool the two mints differ, so a known input that
// is neither SOL nor a stablecoin is a SELL whatever the other mint is (as
// utils.GetTradeType says for every output). Synthetic, as no real
// pre-execution swap with only such an input known exists in the fixtures:
// the real token -> WSOL swap 5kaAWK5X (executed: SELL) with what names the
// mint of its WSOL destination and of the WSOL vault removed (their token
// balances, and the outer InitializeAccount of the destination turned into
// a SyncNative of it), leaving only the token input known. Before, it was
// SWAP.
func TestIntegShredRaydiumV4KnownTokenInput(t *testing.T) {
	d := constants.DISCRIMINATORS.RAYDIUM
	real := loadFixture(t, sigRaySwap18)
	ix := findIx(t, real, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, d.SWAP)
	idx := utils.FormatIdx(ix.outer, ix.inner)
	if full := oneTypedAt(t, parseShred(t, real, nil), constants.DEX_PROGRAMS.RAYDIUM_V4.ID, idx).Trade; full.Type != types.TradeTypeSell ||
		full.OutputToken.Mint != solMint || constants.IsStablecoin(full.InputToken.Mint) || full.InputToken.Mint == solMint {
		t.Fatalf("5kaAWK5X executed: %s %s -> %s, want SELL token -> WSOL", full.Type, full.InputToken.Mint, full.OutputToken.Mint)
	}

	syn := cloneTx(t, real)
	keys := rawAccountKeys(syn)
	// 18-account layout: 5 coin vault, 6 pc vault, 15 source, 16 destination
	drop := map[string]bool{ix.accounts[16]: true}
	for _, v := range []string{ix.accounts[5], ix.accounts[6]} {
		if tokenBalanceMint(real, v) == solMint {
			drop[v] = true
		}
	}
	filter := func(list []adapter.TokenBalance) []adapter.TokenBalance {
		var kept []adapter.TokenBalance
		for _, b := range list {
			if b.AccountIndex >= len(keys) || !drop[keys[b.AccountIndex]] {
				kept = append(kept, b)
			}
		}
		return kept
	}
	syn.Meta.PreTokenBalances, syn.Meta.PostTokenBalances = filter(syn.Meta.PreTokenBalances), filter(syn.Meta.PostTokenBalances)
	for _, x := range fixtureIxs(t, syn) {
		if x.programId == constants.TOKEN_PROGRAM_ID && len(x.data) > 0 && (x.data[0] == 1 || x.data[0] == 16 || x.data[0] == 18) &&
			len(x.accounts) > 1 && x.accounts[0] == ix.accounts[16] {
			x.m["data"] = base58.Encode([]byte{17}) // SyncNative
		}
	}

	tr := oneTypedAt(t, parseShred(t, syn, nil), constants.DEX_PROGRAMS.RAYDIUM_V4.ID, idx).Trade
	if tr.InputToken.Mint == "" || tr.OutputToken.Mint != "" {
		t.Fatalf("synthetic: mints %q -> %q, want token -> \"\"", tr.InputToken.Mint, tr.OutputToken.Mint)
	}
	if tr.Type != types.TradeTypeSell {
		t.Errorf("synthetic token -> unknown: %s, want SELL (executed: SELL)", tr.Type)
	}
}

// Shred results carry the adapter's warnings like ParseResult does (fetcher
// errors, unresolved lookup accounts); they were dropped.
func TestIntegShredWarnings(t *testing.T) {
	stripped := cloneTx(t, loadFixture(t, sigTwoLookups))
	stripped.Meta.LoadedAddresses = nil
	cfg := types.DefaultParseConfig()
	cfg.ALTsFetcher = types.NewALTsFetcher(types.FetchFilterAll, func([]types.AddressTableLookup) (map[string]*types.LoadedAddresses, error) {
		return nil, errors.New("rpc down")
	})
	r := dexparser.NewShredParser().ParseAll(stripped, &cfg)
	if r == nil || !hasWarning(r.Warnings, "ALTsFetcher: rpc down") || !hasWarning(r.Warnings, "unresolved address lookup table accounts") {
		t.Errorf("shred Warnings = %v", r.Warnings)
	}
	if r := dexparser.NewShredParser().ParseAll(loadFixture(t, sigTwoLookups), nil); len(r.Warnings) != 0 {
		t.Errorf("complete tx: shred Warnings = %v", r.Warnings)
	}
}

// Shred System/Token transfers to relay tip accounts (Jito etc.) are not
// fees: FEE_ACCOUNTS lists the Jito tip accounts, and the shred transfer
// parser flagged every transfer to them IsFee (the DexParser no longer
// does, core2). Checked on every fixture with a tip; transfers to the other
// fee accounts stay IsFee.
func TestIntegShredTipIsNotFee(t *testing.T) {
	tips, fees := 0, 0
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		if tx.Meta == nil || tx.Meta.Err != nil {
			continue
		}
		for _, ins := range dexparser.NewShredParser().ParseAll(tx, nil).ParsedInstructions {
			tr := ins.Transfer
			if tr == nil {
				continue
			}
			switch {
			case constants.IsTipAccount(tr.Info.Destination):
				tips++
				if tr.IsFee {
					t.Errorf("%s %s: tip to %s flagged IsFee", sig[:8], ins.Idx, tr.Info.Destination)
				}
			case constants.IsFeeAccount(tr.Info.Destination):
				fees++
				if !tr.IsFee {
					t.Errorf("%s %s: transfer to fee account %s not flagged IsFee", sig[:8], ins.Idx, tr.Info.Destination)
				}
			}
		}
	}
	if tips == 0 || fees == 0 {
		t.Fatalf("fixtures exercise nothing: %d tips, %d fee transfers", tips, fees)
	}
}
