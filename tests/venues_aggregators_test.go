package tests

import (
	"math/big"
	"reflect"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/propamm"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

const (
	venueUSDC = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	venueUSDT = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
	venueSOL  = "So11111111111111111111111111111111111111112"
	venueCBTC = "cbbtcf3aa214zXHbiAZQwf4122FBYbraNdFqgw4iMij"
)

// meme-9 regression: SolFi V2 routed by Titan (4C2p65nu...). The unknown-DEX
// fallback summed the hop's USDC payout with Titan's payout to the user:
// "BUY USDT 1000011964 -> USDC 1999694239". The hop is 1000011964 USDT ->
// 999897114 USDC (the SolFi V2 vault deltas); the user received 999797125
// USDC (Titan kept 99989).
func TestVenuesTitanDoubleCount(t *testing.T) {
	tx := loadFixture(t, "4C2p65nuttUqBv5VhLNXHq6cNEaexT5i4YGbGSH2hodRGLBQyJBzRCkr2VXPKPNof8Lut8WSm51tkxVv3mvUkpuZ")
	result := dexparser.NewDexParser().ParseAll(tx, nil)
	if !result.State {
		t.Fatalf("parse failed: %s", result.Msg)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("want 1 trade (the SolFi V2 hop), got %d", len(result.Trades))
	}
	hop := result.Trades[0]
	if hop.Idx != "2-1" || hop.AMM != constants.DEX_PROGRAMS.SOLFI_V2.Name ||
		hop.InputToken.Mint != venueUSDT || hop.InputToken.AmountRaw != "1000011964" ||
		hop.OutputToken.Mint != venueUSDC || hop.OutputToken.AmountRaw != "999897114" {
		t.Errorf("hop %s %s: %s %s -> %s %s", hop.Idx, hop.AMM, hop.InputToken.AmountRaw, hop.InputToken.Mint, hop.OutputToken.AmountRaw, hop.OutputToken.Mint)
	}

	agg := result.AggregateTrade
	if agg == nil {
		t.Fatal("no aggregate trade")
	}
	if agg.InputToken.Mint != venueUSDT || agg.InputToken.AmountRaw != "1000011964" || agg.OutputToken.Mint != venueUSDC {
		t.Errorf("aggregate %s %s -> %s", agg.InputToken.AmountRaw, agg.InputToken.Mint, agg.OutputToken.Mint)
	}
	// Between what the user received (Titan's total) and what the pool paid
	// out (the hop), never the double count
	out, _ := new(big.Int).SetString(agg.OutputToken.AmountRaw, 10)
	if out == nil || out.Cmp(big.NewInt(999797125)) < 0 || out.Cmp(big.NewInt(999897114)) > 0 {
		t.Errorf("aggregate output %s USDC, want 999797125..999897114", agg.OutputToken.AmountRaw)
	}
}

// Titan route trades come from the SwapRouteV3 swap event: the user's totals,
// equal to the user's token balance changes.
func TestVenuesTitanRoute(t *testing.T) {
	cases := []struct {
		name, sig, idx, user        string
		inMint, inAmount            string
		outMint, outAmount          string
		amms                        []string
		feeMint, feeAmount          string
		checkInDelta, checkOutDelta bool
	}{
		// Single hop; Titan keeps 99989 USDC (fee_c, not a transfer)
		{"single hop", "4C2p65nuttUqBv5VhLNXHq6cNEaexT5i4YGbGSH2hodRGLBQyJBzRCkr2VXPKPNof8Lut8WSm51tkxVv3mvUkpuZ", "2", "EX9NvTr9xcZy9QEgZYpyBeKFkxR4y7jdAzEHRoYnMDye",
			venueUSDT, "1000011964", venueUSDC, "999797125", []string{"SolFiV2"}, "", "", true, true},
		// Split over SolFi V2 and Manifest; the hops paid out 1000117997
		{"split route", "4X2GpZMxpeRKzE3KPzqa2ZPbNVt4He4G6L9Z7a7kSze5oMAKjFheTuPYkxRtBUi2dgkk8z9KRysx1ZEfhFCJSa9y", "2", "EX9NvTr9xcZy9QEgZYpyBeKFkxR4y7jdAzEHRoYnMDye",
			venueUSDC, "999999000", venueUSDT, "1000017986", []string{"SolFiV2", "Manifest"}, "", "", true, true},
		// Output-side fee_a of 2495 USDC paid to an integrator fee account
		{"output fee", "24FsRUqwK7CSax3qqwhXwgUzQvZ2MZ7MonsYm1RLx92S9pisUTNT6SUEEuuNoqTriZQCuao1Lhgso1bz4wcSQAWb", "4", "3hEp7vMMZugfynHhPBqNyLmApaspwbE8JJLpT94FcKhY",
			venueCBTC, "29592", venueUSDC, "24956038", []string{"GoonFiV2"}, venueUSDC, "2495", true, true},
		// Native SOL input through a temporary WSOL account (no token
		// balance); input-side fee_a of 1620000 lamports (transfer 2-13);
		// Token-2022 output with a transfer fee: out_amount is what arrived
		{"SOL input", "4ebbdR9NqVCa57D6yy4ThZfCESz1xgZAfzWZK1J5FNcLFg8VQW2J5NhF8STRGzZ787y7tB3Tc6q6qa37pSQvbpWP", "2", "CZnR4az9AhK5xay15Jw81fnzjNx6rYpqV6zdBwHDd69w",
			venueSOL, "200000000", "6gxc6ZWCMA2XzwWFGD9CKFHe8UMBmPxdKDHC1w3HfzDw", "1764441714011", []string{"MeteoraDLMM"}, venueSOL, "1620000", false, true},
		// Titan invoked by another program (inner 2-1); its fee_a of 3222
		// USDC went to the user's own account, so the user received it
		{"inner, fee to user", "ZoQtiC3w65x9v6vZkKsP4KpKZHKENGYis4gNDMQovycr4ZUhKXLHw2p7B9hVz6DyiPjf8o3nFPLyurs5yp1gJKL", "2-1", "D9Y7KCHciPeieqaxiubMYm8Tg8a4Je8ziCkLJsYMznVK",
			venueCBTC, "1272", venueUSDC, "1074088", []string{"GoonFiV2"}, "", "", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := loadFixture(t, tc.sig)
			ctx := newParseContext(tx, nil)
			trades := propamm.NewTitanParser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions,
				ctx.Classifier.GetInstructions(constants.DEX_PROGRAMS.TITAN.ID)).ProcessTrades()
			if len(trades) != 1 {
				t.Fatalf("want 1 route trade, got %d", len(trades))
			}
			checkRouteTrade(t, tx, trades[0], constants.DEX_PROGRAMS.TITAN, tc.idx, tc.user, tc.inMint, tc.inAmount, tc.outMint, tc.outAmount, tc.amms, tc.feeMint, tc.feeAmount, tc.checkInDelta, tc.checkOutDelta)
		})
	}
}

// OKX DEX Router V2 route trades come from the final swap event (emit_cpi).
func TestVenuesOKXV2Route(t *testing.T) {
	cases := []struct {
		name, sig, idx, user        string
		inMint, inAmount            string
		outMint, outAmount          string
		amms                        []string
		feeMint, feeAmount          string
		checkInDelta, checkOutDelta bool
	}{
		// swap_tob, SwapWithFeesCpiEvent2 with a 1275120 USDC commission on
		// the input side (amount_in 148739084 + commission)
		{"input commission", "262n1pEgfLq9G87vRNBsb5xKdmADSQxFGWXXb2qXkxqtsPViu5ifNWC2UsGCFpo1aXaagpsetjxNZKoR2mG6D3xw", "5", "DaYgWwWaqiQKhGLFJHLizxjTatqDaLNnLXWj25yNetGW",
			venueUSDC, "150014204", "Ce2gx9KGXJ6C9Mp5b5x1sn9Mg87JwEbrQby4Zqo3pump", "3667578810", []string{"BisonFi", "RaydiumCL"}, venueUSDC, "1275120", true, true},
		// swap_tob, commission of 7472162 USDC on the output side: the last
		// hop (HumidiFi) paid out 879077963
		{"output commission", "4tzaGfjUtBPNKaSJcZWcWbUeVyMj3aEsQ6XXpq341dJWUEYAFK7MX5jCcDKkCNuZRNLqBhEBMMVf8APdRXHtX4rT", "6", "2PhRZsgSFjeUjNS7i15xipuGEytfjr2AnoCdbJaVhhVr",
			"8RVBk8vxLiUHueLUW1f4izFVqN3nWippLhkohKg6EGkS", "102474923269", venueUSDC, "871605801", []string{"MeteoraDLMM", "Manifest", "HumidiFi"}, venueUSDC, "7472162", true, true},
		// swap_tob_v3, SwapWithFeeCpiEventV3, SOL -> SOL cycle over three
		// prop AMMs; the user's WSOL grew by 44051
		{"v3 cycle", "5PrZGSv25nELLAXnGg6N7WL45UVJ6ZPPku4FWo4fgmew3qPDUEzBi8BNNujkxSKykjHKw9St6L4cvRfYFqzenr6D", "2", "DDyDDYFniwBYKfMzoL61Yyc53Xa5N5znNWKGLjxyAh69",
			venueSOL, "690000000", venueSOL, "690044051", []string{"GoonFiV2", "SolFiV2", "Quantum"}, "", "", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := loadFixture(t, tc.sig)
			ctx := newParseContext(tx, nil)
			trades := propamm.NewOKXV2Parser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions,
				ctx.Classifier.GetInstructions(constants.DEX_PROGRAMS.OKX_DEX_V2.ID)).ProcessTrades()
			if len(trades) != 1 {
				t.Fatalf("want 1 route trade, got %d", len(trades))
			}
			checkRouteTrade(t, tx, trades[0], constants.DEX_PROGRAMS.OKX_DEX_V2, tc.idx, tc.user, tc.inMint, tc.inAmount, tc.outMint, tc.outAmount, tc.amms, tc.feeMint, tc.feeAmount, tc.checkInDelta, tc.checkOutDelta)
		})
	}

	// The SOL -> SOL cycle: net gain equals the user's WSOL change
	tx := loadFixture(t, "5PrZGSv25nELLAXnGg6N7WL45UVJ6ZPPku4FWo4fgmew3qPDUEzBi8BNNujkxSKykjHKw9St6L4cvRfYFqzenr6D")
	if d := ownerTokenDelta(tx, "DDyDDYFniwBYKfMzoL61Yyc53Xa5N5znNWKGLjxyAh69", venueSOL); d.Int64() != 690044051-690000000 {
		t.Errorf("user WSOL change %s, want 44051", d)
	}
}

func checkRouteTrade(t *testing.T, tx *adapter.SolanaTransaction, tr types.TradeInfo, program constants.DexProgram,
	idx, user, inMint, inAmount, outMint, outAmount string, amms []string, feeMint, feeAmount string, checkInDelta, checkOutDelta bool) {
	t.Helper()
	if tr.Idx != idx || tr.ProgramId != program.ID || tr.Route != program.Name || tr.User != user {
		t.Errorf("idx %s program %s route %s user %s, want %s %s %s %s", tr.Idx, tr.ProgramId, tr.Route, tr.User, idx, program.ID, program.Name, user)
	}
	if tr.InputToken.Mint != inMint || tr.InputToken.AmountRaw != inAmount {
		t.Errorf("input %s %s, want %s %s", tr.InputToken.AmountRaw, tr.InputToken.Mint, inAmount, inMint)
	}
	if tr.OutputToken.Mint != outMint || tr.OutputToken.AmountRaw != outAmount {
		t.Errorf("output %s %s, want %s %s", tr.OutputToken.AmountRaw, tr.OutputToken.Mint, outAmount, outMint)
	}
	if !reflect.DeepEqual(tr.AMMs, amms) {
		t.Errorf("AMMs %v, want %v", tr.AMMs, amms)
	}
	if feeAmount == "" {
		if tr.Fee != nil {
			t.Errorf("unexpected fee %s %s", tr.Fee.AmountRaw, tr.Fee.Mint)
		}
	} else if tr.Fee == nil || tr.Fee.Mint != feeMint || tr.Fee.AmountRaw != feeAmount {
		t.Errorf("fee %+v, want %s %s", tr.Fee, feeAmount, feeMint)
	}
	if checkInDelta {
		if d := ownerTokenDelta(tx, user, inMint); new(big.Int).Neg(d).String() != inAmount {
			t.Errorf("user %s change %s, input %s", inMint, d, inAmount)
		}
	}
	if checkOutDelta {
		if d := ownerTokenDelta(tx, user, outMint); d.String() != outAmount {
			t.Errorf("user %s change %s, output %s", outMint, d, outAmount)
		}
	}
}
