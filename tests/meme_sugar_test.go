package tests

import (
	"bytes"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
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

const (
	sigSugarCreate  = "3nFeuZaTBjCVkcTMYN57THW59ZBCewjM3cey1qEEVSaC2F4tvm1fvmF6DWweJxzBR72Jre7kHW7nqSwW1JStXMwx"
	sigSugarMigrate = "2kWg1XifH9P7JYinGhgPNL7EaaKb2K8KBzipRDjhfDwyhasmY9hKzsqNwiDPCB8ALbYioDpPziCztZqsUoBLi4yR"
)

// meme-18: the Sugar create read the user from account 0 (the global
// config), the base mint from account 6 (the creator) and the bonding curve
// from account 1 (the metadata account). In the real create at outer 3 the
// mint is the one initialised at outer 1 and named in the Metaplex create at
// 3-2, and the pool is account 2 of the buy of the same token at outer 5.
func TestMemeSugarCreate(t *testing.T) {
	raw := loadMemeRaw(t, sigSugarCreate)
	mint := raw.accounts(1, -1)[0] // InitializeMint2
	metaplex := raw.accounts(3, 2) // metadata, mint, mint authority, payer
	pool := raw.accounts(5, -1)[2] // buy_exact_in pool
	creator := raw.keys[0]         // signer and fee payer
	if metaplex[1] != mint || !bytes.Contains(raw.data(3, 2), []byte("AI TROLL LEVEL 9000")) {
		t.Fatalf("fixture: metaplex create %v", metaplex)
	}

	r := memeParse(t, sigSugarCreate)
	e := memeEventAt(t, r, "3")
	if e.Type != types.TradeTypeCreate || e.BaseMint != mint || e.Pool != pool || e.BondingCurve != pool ||
		e.User != creator || e.Creator != creator || e.QuoteMint != solMint || e.Name != "AI TROLL LEVEL 9000" {
		t.Errorf("create %+v, want mint %s pool %s creator %s", e, mint, pool, creator)
	}
	if e.Decimals == nil || *e.Decimals != raw.decimals[mint] {
		t.Errorf("create decimals %v, want %d", e.Decimals, raw.decimals[mint])
	}
	if buy := memeEventAt(t, r, "5"); buy.BaseMint != mint || buy.Pool != e.Pool {
		t.Errorf("buy %+v does not trade the created token", buy)
	}
}

// meme-18: migrate_to_radium was not decoded. In the real migration at outer
// 2 the program creates a Raydium CPMM pool (inner 2-4: creator, config,
// authority, pool state, token 0 mint, token 1 mint) and the bonding curve
// authorises the transfer of the remaining tokens (inner 2-2).
func TestMemeSugarMigrate(t *testing.T) {
	raw := loadMemeRaw(t, sigSugarMigrate)
	if raw.program(2, 4) != constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID {
		t.Fatalf("fixture: inner 2-4 is %s", raw.program(2, 4))
	}
	cpmm := raw.accounts(2, 4)
	creator, cpmmPool, quote, base := cpmm[0], cpmm[3], cpmm[4], cpmm[5]
	var curve string
	for _, tr := range raw.transfers(2) {
		if tr.Inner == 2 {
			curve = tr.Authority
		}
	}

	e := memeEventAt(t, memeParse(t, sigSugarMigrate), "2")
	if e.Type != types.TradeTypeMigrate || e.BaseMint != base || e.QuoteMint != quote || e.BondingCurve != curve ||
		e.Pool != cpmmPool || e.User != creator || e.PoolDex != constants.DEX_PROGRAMS.RAYDIUM_CPMM.Name {
		t.Errorf("migrate %+v, want base %s quote %s curve %s pool %s user %s", e, base, quote, curve, cpmmPool, creator)
	}
}
