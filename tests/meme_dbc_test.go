package tests

import (
	"testing"

	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Meteora DBC regression tests. Expected directions and fees come from
// decoding EvtSwap/EvtSwap2 with the dynamic-bonding-curve IDL (0.2.1);
// expected amounts from the raw transfers into and out of the swap's
// input/output token accounts.

const (
	sigDBCSell            = "5J6cC1QJJxTLSqD2VBeqCFqcRFqMptV1wH3QiNCUYEu2aZ9uUc4ySJfHecYBSxR5pcpQttVtq8piBR6mj36gR2vE"
	sigDBCBuy             = "3sSd47j5dSqqoxmjZjhfrE4sdW1haLA2hASNxg7D7ZHYegtvnuSTiqbZggxkGxMvz2wm7HEdxxDVsgWhAMVkmM3v"
	sigDBCBuyPartial      = "51whTP7RjvqzE4UCDTjK4EKjV7dvhZDeTboWkwjWq1QZJgygC2pseYejf4JYoGMQh5Vf2amg1uhqAFV8WZJUdQBS"
	sigDBCBuyExactOut     = "2eN69356mJ5AVgnY9Dh7P5XjnsXTBbYv6GuL2PuE6c6D6Dpwz6gfUcA6nLFkNGXium9jX91vzeWcDRbhsJCwZZ5f"
	sigDBCRoutedSell      = "128TrBNx8icLgpsCY1Tkk6WXv4QwyJDEBTGuqfRQaT7YG3CFdYTuuw3aVCu36Wqxruu3YdE5HuhiciAooKCw9nHJ"
	sigDBCCompleteBuy     = "ZADe3kHFeqgpN96HQKoxBPdsfq5y9hnPdt1Ue523D8biaeCogpZDz7gdkoNtLLUDfWL7bz142vQwvc9uX44zQdo"
	sigDBCTransferHookBuy = "3ou2zc4g8GGfufg1yX9DJpmSPmr6Uo25LsuocezSf5swegypu1aSYtfw47ZSte6PCUBexSeA4Veim65vqaDXp8zr"
	sigDBCSwapV1          = "5Y1z7B8doTaXMjqzCtRSnTPbSkpU3Ryjym857NVBBfdYLkoVP95o7V7KPdJZXNdsMXoosqghhb6aNYCmW42Ugs6t"
	sigDBCSellSwap2       = "2iZbN25JjrpUDjPeYmhmAm3MWnN3XNNV8Pf9gQc742we7XZDQ9uu2ZddE2sfUBquRGH1mtgsP8FGAGGyGBbG44qT"
	sigDBCMigrateDammV2   = "2P2i1ZR2JctufV5QDgfuhb42SNARTtyDoeiBV7i6hgEjuDfQkLCSyJtdpmRcCArBehMZrLkTw9viXXMvgTcwVDyD"
	sigDBCCreate          = "3uwWqXt9wgrkfp1bwLJjMApxLW9TYxriEPtan4aWouH7ExebphFD3LthPmzXmP6eeZ5nKdtrUZodpAZiWR5P8DNV"
)

// meme-1, parity-2, meme-14 and the core request on 128TrBNx: every DBC
// trade was a BUY (stub direction check), the transfer lookup key never
// matched, and amounts fell back to the instruction args (min_out) with
// decimals 0. swap2_with_transfer_hook was not recognised.
func TestMemeDBCTrades(t *testing.T) {
	cases := []struct {
		sig          string
		outer, inner int // swap instruction
		typ          types.TradeType
		in, out      string // raw amounts the user sent / received
	}{
		{sigDBCSell, 3, -1, types.TradeTypeSell, "107836091692", "45159476"},       // trade_direction 0
		{sigDBCSellSwap2, 2, -1, types.TradeTypeSell, "243584347625", "139721562"}, // trade_direction 0
		{sigDBCBuy, 3, -1, types.TradeTypeBuy, "10000000", "17271977390"},          // min_out 10209645265
		{sigDBCBuyPartial, 6, -1, types.TradeTypeBuy, "50000000", "99244682753"},   // swap_mode 1
		{sigDBCBuyExactOut, 6, -1, types.TradeTypeBuy, "1844150", "3661053452"},    // swap_mode 1
		{sigDBCRoutedSell, 2, 5, types.TradeTypeSell, "359450717759", "16132169"},  // DFlow CPI, output was 0
		{sigDBCCompleteBuy, 1, -1, types.TradeTypeBuy, "10026", "219999968000913"}, // fee on input
		{sigDBCTransferHookBuy, 4, 11, types.TradeTypeBuy, "313831430", "4766792806358"},
		{sigDBCSwapV1, 1, -1, types.TradeTypeBuy, "2800000", "22608416383"}, // swap (v1)
	}
	for _, c := range cases {
		r := memeParse(t, c.sig)
		idx := u64s(uint64(c.outer))
		if c.inner >= 0 {
			idx += "-" + u64s(uint64(c.inner))
		}
		tr := memeTradeAt(t, r, idx)
		raw := loadMemeRaw(t, c.sig)
		acc := raw.accounts(c.outer, c.inner)
		base, quote, payer := acc[7], acc[8], acc[9]
		inMint, outMint := quote, base
		if c.typ == types.TradeTypeSell {
			inMint, outMint = base, quote
		}
		// independent truth: the transfers out of the input account and into
		// the output account inside the swap
		from, to := 0, 1000
		if c.inner >= 0 {
			from = c.inner + 1
		}
		sent := raw.sumTransfers(c.outer, from, to, func(x memeRawTransfer) bool { return x.From == acc[3] && x.Authority == payer })
		got := raw.sumTransfers(c.outer, from, to, func(x memeRawTransfer) bool { return x.To == acc[4] && x.Mint == outMint })
		if sent.String() != c.in || got.String() != c.out {
			t.Fatalf("%.8s: fixture transfers %s -> %s, table %s -> %s", c.sig, sent, got, c.in, c.out)
		}
		if tr.Type != c.typ || tr.User != payer || tr.Pool[0] != acc[2] {
			t.Errorf("%.8s: %s user %s pool %v, want %s by %s", c.sig, tr.Type, tr.User, tr.Pool, c.typ, payer)
		}
		checkToken(t, c.sig[:8]+" input", &tr.InputToken, inMint, c.in, raw.decimals[inMint])
		checkToken(t, c.sig[:8]+" output", &tr.OutputToken, outMint, c.out, raw.decimals[outMint])
		e := memeEventAt(t, r, idx)
		if e.Type != c.typ || e.InputToken == nil || e.InputToken.AmountRaw != c.in || e.OutputToken.AmountRaw != c.out {
			t.Errorf("%.8s: meme event %s %+v %+v", c.sig, e.Type, e.InputToken, e.OutputToken)
		}
	}

	// 128TrBNx: the DBC leg is traded by 8EQVMv, which owns the sold tokens;
	// the fee payer AgmLJB is a relayer that signs unrelated txs (3ou2zc4g,
	// 3H4KQatU) for other traders
	raw := loadMemeRaw(t, sigDBCRoutedSell)
	tr := memeTradeAt(t, memeParse(t, sigDBCRoutedSell), "2-5")
	if owner := raw.owner[raw.accounts(2, 5)[3]]; tr.User != owner || owner != "8EQVMv3fhiPW52GSkRYZ4m5PsDSw79YtXn8RRRxxFsDk" {
		t.Errorf("128TrB user %s, owner of the sold tokens %s", tr.User, owner)
	}
}

// DBC fees come from the swap event, in the mint they are charged in.
func TestMemeDBCFees(t *testing.T) {
	cases := []struct {
		sig, idx, mint, trading, protocol string
	}{
		{sigDBCSell, "3", solMint, "90546", "22636"},                                                      // fee on the SOL output
		{sigDBCBuyPartial, "6", "DfNX8qA257nmcoYK2NgAqzdeuQFRvviiiy24DMP5cH7G", "801977235", "200494308"}, // on the base output
		{sigDBCCompleteBuy, "1", solMint, "21", "5"},                                                      // on the input: 10026 - 10000
	}
	for _, c := range cases {
		tr := memeTradeAt(t, memeParse(t, c.sig), c.idx)
		if tr.Fee == nil || tr.Fee.Mint != c.mint || feeOf(tr.Fees, "trading") != c.trading || feeOf(tr.Fees, "protocol") != c.protocol {
			t.Errorf("%.8s: Fee %+v Fees %+v", c.sig, tr.Fee, tr.Fees)
		}
	}
}

// meme-14: EvtCurveComplete becomes a COMPLETE meme event; creates and
// migrations keep working.
func TestMemeDBCCurveCompleteCreateMigrate(t *testing.T) {
	r := memeParse(t, sigDBCCompleteBuy)
	e := memeEventAt(t, r, "1-4")
	if e.Type != types.TradeTypeComplete || e.Pool != "3Zz5xdvCngrdfiNKo3qnWWa7agM3tvzqpV76WAJES4j4" ||
		e.RealBaseReserves != "780000031999087" || e.RealQuoteReserves != "10000" || e.BaseMint != "5iJA3mcRuvCHm86UwhY9SL2C19vviUiox1vDaNnVK777" || e.QuoteMint != solMint {
		t.Errorf("curve complete %+v", e)
	}
	create := memeEventAt(t, memeParse(t, sigDBCCreate), "0")
	if create.Type != types.TradeTypeCreate || create.Name != "FomoRelay" || create.Pool != e.Pool || create.BaseMint != e.BaseMint {
		t.Errorf("create %+v", create)
	}
	migrate := memeEventAt(t, memeParse(t, sigDBCMigrateDammV2), "0")
	if migrate.Type != types.TradeTypeMigrate || migrate.BaseMint != e.BaseMint || migrate.BondingCurve != e.Pool ||
		migrate.PoolDex != constants.DEX_PROGRAMS.METEORA_DAMM_V2.Name {
		t.Errorf("migrate %+v", migrate)
	}
}

// meme-v-2: migrate_meteora_damm with 8 accounts read accounts[8] and the
// panic dropped the whole parse. Synthetic: the real migration_damm_v2 of
// 2P2i1ZR2 relabelled as migrate_meteora_damm and cut to 8 accounts (no real
// DAMM v1 migration is in the fixtures).
func TestMemeDBCMigrateDammShortAccounts(t *testing.T) {
	tx := cloneTx(t, loadFixture(t, sigDBCMigrateDammV2))
	ix := tx.Transaction.Message.Instructions[0].(map[string]interface{})
	data, _ := base58.Decode(ix["data"].(string))
	copy(data, constants.DISCRIMINATORS.METEORA_DBC.METEORA_DBC_MIGRATE_DAMM)
	ix["data"] = base58.Encode(data)
	ix["accounts"] = ix["accounts"].([]interface{})[:8]
	r := dexparserParse(tx)
	if r == nil || !r.State {
		t.Fatalf("parse failed: %+v", r)
	}
	for _, e := range r.MemeEvents {
		if e.Protocol == constants.DEX_PROGRAMS.METEORA_DBC.Name {
			t.Errorf("migrate event from 8 accounts: %+v", e)
		}
	}
}
