package tests

import (
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Sugar regression tests on mainnet trades found with getSignaturesForAddress
// on the Sugar program (2026-09-27 scan).

const (
	sigSugarSell       = "4oawudsw5gnFCcptnCCuEKGwhQJSChvhKGsQzYkcBUpZaF522VX54tu1VHDE7Dcr4dEZeRoFVRHZQzEcdNHFbxHv"
	sigSugarSellRouted = "4V3sxzz3TZqE7pvkPYMcFrWiE2ukMmfbAZvhGNSTFy5bFPiN9EkhghEGwYtqBrkYUvN7gWHM22zP6x4RNt5n5whi"
	sigSugarBuyMaxOut  = "2cPM7KBvQ9WrZECHJGfLhtf6b3EGNGVng99vHa6UadoGT2GkEVLpmQzij4nEEHVTWYuk9yd98kbGuYWLLouc3naw"
)

// meme-18: Sugar used pool 1, user 0, base 6, quote 7 (upstream: base 1, pool
// 2, user 6), amounts were the instruction args, and SELL_EXACT_OUT had a
// 9-byte discriminator. The real sell_exact_in and buy_max_out instructions
// confirm base 1, pool 2, user 6 and native SOL as the quote.
func TestMemeSugarTrades(t *testing.T) {
	cases := []struct {
		sig, idx     string
		outer, inner int
		typ          types.TradeType
	}{
		{sigSugarSell, "2", 2, -1, types.TradeTypeSell},
		{sigSugarSellRouted, "2-0", 2, 0, types.TradeTypeSell},
		{sigSugarBuyMaxOut, "3-0", 3, 0, types.TradeTypeBuy},
	}
	for _, c := range cases {
		raw := loadMemeRaw(t, c.sig)
		acc := raw.accounts(c.outer, c.inner)
		base, pool, user := acc[1], acc[2], acc[6]
		from := c.inner + 1
		var sol, tokens string
		if c.typ == types.TradeTypeBuy {
			sol = raw.sumTransfers(c.outer, from, from+4, func(x memeRawTransfer) bool { return x.Mint == solMint && x.From == user }).String()
			tokens = raw.sumTransfers(c.outer, from, from+4, func(x memeRawTransfer) bool { return x.Mint == base && raw.owner[x.To] == user }).String()
		} else {
			sol = raw.sumTransfers(c.outer, from, from+4, func(x memeRawTransfer) bool { return x.Mint == solMint && x.To == user }).String()
			tokens = raw.sumTransfers(c.outer, from, from+4, func(x memeRawTransfer) bool { return x.Mint == base && x.Authority == user }).String()
		}
		r := memeParse(t, c.sig)
		tr := memeTradeAt(t, r, c.idx)
		solSide, tokenSide := &tr.OutputToken, &tr.InputToken
		if c.typ == types.TradeTypeBuy {
			solSide, tokenSide = &tr.InputToken, &tr.OutputToken
		}
		if tr.Type != c.typ || tr.User != user || len(tr.Pool) != 1 || tr.Pool[0] != pool {
			t.Errorf("%.8s: %s user %s pool %v, want %s %s %s", c.sig, tr.Type, tr.User, tr.Pool, c.typ, user, pool)
		}
		checkToken(t, c.sig[:8]+" SOL", solSide, solMint, sol, 9)
		checkToken(t, c.sig[:8]+" token", tokenSide, base, tokens, raw.decimals[base])
		if e := memeEventAt(t, r, c.idx); e.BaseMint != base || e.Pool != pool || e.User != user {
			t.Errorf("%.8s: meme event %+v", c.sig, e)
		}
	}
}
