package tests

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/meme"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Heaven, Boop.fun and Moonit regression tests. Expected amounts come from
// the raw transfers of the fixtures and, for Moonit, from the program's own
// "Transfering collateral ..." log line.

const (
	sigHeavenSell        = "3cbL8koKqz7sejeZBh2qh9XUGsJA6YYLJmtmrLFd9WHjJpotsNU9Dk65UbHHB2rDruYrLcwfn9Eboh41tGHvWcXt"
	sigHeavenSellRouted  = "4e8T9Hnh2AbXV482VcwLr1FWcZeCtQNKW5qsw3uJRt5oQpvyp6qRK2aoxP7JYHaQRScEzFWid1krx7ezDUNAyx1X"
	sigHeavenBuy         = "2fvjJbqUPDcY9x8YrZTsTzHEPJEcrhxBWtvoqtVv5BBZsYJPuPxey415ztmaCJeCd6CSXdwGBeYxuxjpgDALEvUc"
	sigBoopBuy           = "28S2MakapF1zTrnqYHdMxdnN9uqAfKV2fa5ez9HpE466L3xWz8AXwsz4eKXXnpvX8p49Ckbp26doG5fgW5f6syk9"
	sigBoopSellOuter     = "5ugVUyDQdxkkBRNqQh2Mk1SFcEnGFenx21fdg9C5yK2st9AjJaQz9g1eaLXJejkQHATu3CNbWDAsU9XdNiBsP2QS"
	sigBoopCreateBuy     = "4YxPRX9p3rdN7H6cbjC6pKqyQfTu589nkVH3PqxFQyaoP5cZxEgfLK2SJmHFzUTXoJceGtxC8eGXeDqFjLE2UycH"
	sigBoopSellRouted    = "5eCyapuWUXoSkbTY4PDMVvB6e7LP4ujUYtfY8ULQQaAYYD5hqSGwHMuFrywjHndVzyVDDp25JAAvKKv6Fa2Tr8XX"
	sigMoonitSell        = "2XYu86VrUXiwNNj8WvngcXGytrCsSrpay69Rt3XBz9YZvCQcZJLjvDfh9UWETFtFW47vi4xG2CkiarRJwSe6VekE"
	sigMoonitBuy11       = "4wQnGcxkvvqFYwTKXVp5emsDydb5Ts6HG8yzroqoF19EYdkJCUdsHQDUGgTnTxXxo5gVfkCrv7V1DkQ82V7WnTQS"
	sigMoonitSellUSDCHop = "5mPd52BGevpvqU4XNaUuhMa7MqAvC8MXvmoYnAx1Za5XHsSQ6quU7orGYQ93CZtSTyWugnxv62g6yof7ahAUgzvN"
	sigMoonitSellRouted  = "Dr6ZkVfaHmFvJtH81Amt5z3gPzbwyWKGrx9g6C3wRWBUHSxMoVQSLHwgBKEfZvjs2naRrikELmQrGyrAYS9XXWt"
	sigMoonitBuy14       = "3TZKJLxy4H2wQiYenuSVoQ2ox7xveRoG5bxr7yfmEdtMPqKmdHWcf5Q9B8uUBi6ystp2gQsZdP5qxiYK4JnUpm7"
	sigMoonitBuy         = "AhiFQX1Z3VYbkKQH64ryPDRwxUv8oEPzQVjSvT7zY58UYDm4Yvkkt2Ee9VtSXtF6fJz8fXmb5j3xYVDF17Gr9CG"
)

// meme-7, meme-13: Heaven used accounts 3/4/5/6 (user = System program, base
// mint = pool, quote mint = user wallet) and the instruction args as amounts
// (min_out, 0 on the buy); upstream uses pool 4, user 5, base 6, quote 7.
func TestMemeHeavenTrades(t *testing.T) {
	cases := []struct {
		sig          string
		outer, inner int
		typ          types.TradeType
	}{
		{sigHeavenSell, 3, 0, types.TradeTypeSell},
		{sigHeavenSellRouted, 3, 1, types.TradeTypeSell},
		{sigHeavenBuy, 6, 0, types.TradeTypeBuy},
	}
	for _, c := range cases {
		raw := loadMemeRaw(t, c.sig)
		acc := raw.accounts(c.outer, c.inner)
		pool, user, base, quote := acc[4], acc[5], acc[6], acc[7]
		inMint, outMint := quote, base
		if c.typ == types.TradeTypeSell {
			inMint, outMint = base, quote
		}
		sent := raw.sumTransfers(c.outer, c.inner+1, c.inner+6, func(x memeRawTransfer) bool { return x.Mint == inMint && x.Authority == user })
		got := raw.sumTransfers(c.outer, c.inner+1, c.inner+6, func(x memeRawTransfer) bool { return x.Mint == outMint && x.Authority != user })
		if sent.Sign() == 0 || got.Sign() == 0 {
			t.Fatalf("%.8s: no fixture transfers", c.sig)
		}
		idx := fmt.Sprintf("%d-%d", c.outer, c.inner)
		e := memeEventAt(t, memeParse(t, c.sig), idx)
		if e.Type != c.typ || e.User != user || e.Pool != pool || e.BaseMint != base || e.QuoteMint != quote {
			t.Errorf("%.8s: %s user %s pool %s base %s quote %s", c.sig, e.Type, e.User, e.Pool, e.BaseMint, e.QuoteMint)
		}
		checkToken(t, c.sig[:8]+" input", e.InputToken, inMint, sent.String(), raw.decimals[inMint])
		checkToken(t, c.sig[:8]+" output", e.OutputToken, outMint, got.String(), raw.decimals[outMint])
		if user == memeSystemProgram {
			t.Errorf("%.8s: user is the System program", c.sig)
		}

		// the trade parser gives the same (Jupiter covers these outer
		// instructions in ParseAll, so call it directly)
		ctx := newParseContext(loadFixture(t, c.sig), nil)
		trades := meme.NewHeavenParser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions, ctx.Classifier.GetInstructions(constants.DEX_PROGRAMS.HEAVEN.ID)).ProcessTrades()
		if len(trades) != 1 || trades[0].Idx != idx || trades[0].InputToken.AmountRaw != sent.String() || trades[0].OutputToken.AmountRaw != got.String() || trades[0].Pool[0] != pool {
			t.Errorf("%.8s: Heaven trades %+v", c.sig, trades)
		}
	}
}

// meme-13, parity-3 and the core request on 28S2Makap: for outer Boop
// instructions the transfer key was "prog:N-0" instead of "prog:N", so the
// buy output and the sell output were 0.
func TestMemeBoopfunOuterTrades(t *testing.T) {
	cases := []struct {
		sig   string
		outer int
		typ   types.TradeType
	}{
		{sigBoopBuy, 4, types.TradeTypeBuy},
		{sigBoopCreateBuy, 5, types.TradeTypeBuy},
		{sigBoopSellOuter, 2, types.TradeTypeSell},
	}
	for _, c := range cases {
		raw := loadMemeRaw(t, c.sig)
		acc := raw.accounts(c.outer, -1)
		mint, user := acc[0], acc[6]
		r := memeParse(t, c.sig)
		tr := memeTradeAt(t, r, u64s(uint64(c.outer)))
		if c.typ == types.TradeTypeBuy {
			// the user pays buy_amount (curve + trading fee) in SOL and
			// receives the tokens; balance deltas confirm the token side
			paid := raw.sumTransfers(c.outer, 0, 100, func(x memeRawTransfer) bool { return x.Mint == solMint && x.From == user })
			tokens := ownerTokenDelta(loadFixture(t, c.sig), user, mint)
			if tr.InputToken.AmountRaw != paid.String() || tr.OutputToken.AmountRaw != tokens.String() || tokens.Sign() <= 0 {
				t.Errorf("%.8s: buy %s -> %s, user paid %s and received %s", c.sig, tr.InputToken.AmountRaw, tr.OutputToken.AmountRaw, paid, tokens)
			}
			data := raw.data(c.outer, -1)
			if buyAmount := new(big.Int).SetBytes(reverse(data[8:16])); buyAmount.Cmp(paid) != 0 {
				t.Errorf("%.8s: buy_amount %s, transfers %s", c.sig, buyAmount, paid)
			}
		} else {
			got := raw.sumTransfers(c.outer, 0, 100, func(x memeRawTransfer) bool { return x.Mint == solMint && x.To == user })
			if tr.OutputToken.Mint != solMint || tr.OutputToken.AmountRaw != got.String() || got.Sign() <= 0 {
				t.Errorf("%.8s: sell output %s, user received %s", c.sig, tr.OutputToken.AmountRaw, got)
			}
		}
		if tr.User != user || tr.Type != c.typ || tr.Pool[0] != acc[1] {
			t.Errorf("%.8s: %s user %s pool %v", c.sig, tr.Type, tr.User, tr.Pool)
		}
		if tr.OutputToken.Decimals != raw.decimals[tr.OutputToken.Mint] || tr.InputToken.Decimals != raw.decimals[tr.InputToken.Mint] {
			t.Errorf("%.8s: decimals %d/%d, fixture %d/%d", c.sig, tr.InputToken.Decimals, tr.OutputToken.Decimals, raw.decimals[tr.InputToken.Mint], raw.decimals[tr.OutputToken.Mint])
		}
	}
}

// Moving SOL between the user's own accounts (wrapping the proceeds into
// WSOL) inside the transfer group is not part of the trade: the routed Boop
// sell received 203146245 lamports once.
func TestMemeBoopfunIgnoresSelfTransfers(t *testing.T) {
	raw := loadMemeRaw(t, sigBoopSellRouted)
	acc := raw.accounts(3, 0)
	got := raw.sumTransfers(3, 1, 4, func(x memeRawTransfer) bool { return x.Mint == solMint && x.To == acc[6] })
	e := memeEventAt(t, memeParse(t, sigBoopSellRouted), "3-0")
	if e.OutputToken == nil || e.OutputToken.AmountRaw != got.String() || got.String() != "203146245" {
		t.Errorf("routed sell output %+v, user received %s", e.OutputToken, got)
	}
}

// meme-13, parity-18: Moonit amounts came from the signer's balance changes
// and the collateral mint was USDC whenever USDC appeared in the tx; buys
// with 11 accounts produced no meme event and the trade parser required
// exactly 11 accounts.
func TestMemeMoonitTrades(t *testing.T) {
	cases := []struct {
		sig, idx     string
		outer, inner int
		typ          types.TradeType
	}{
		{sigMoonitSell, "2", 2, -1, types.TradeTypeSell},
		{sigMoonitBuy11, "1", 1, -1, types.TradeTypeBuy},
		{sigMoonitBuy, "1", 1, -1, types.TradeTypeBuy},
		{sigMoonitSellUSDCHop, "8-8", 8, 8, types.TradeTypeSell},
		{sigMoonitSellRouted, "4-2", 4, 2, types.TradeTypeSell},
		{sigMoonitBuy14, "5-6", 5, 6, types.TradeTypeBuy},
	}
	for _, c := range cases {
		tx := loadFixture(t, c.sig)
		raw := loadMemeRaw(t, c.sig)
		acc := raw.accounts(c.outer, c.inner)
		user, curve, mint := acc[0], acc[2], acc[6]

		// the program's log line: collateral moved and both fees
		var collateral, helio, dex int64
		for _, l := range tx.Meta.LogMessages {
			if strings.HasPrefix(l, "Program log: Transfering collateral") {
				if _, err := fmt.Sscanf(l[strings.Index(l, ": ")+2:], "Transfering collateral from buyer to curve account: %d, Helio fee: %d, Dex fee: %d", &collateral, &helio, &dex); err != nil {
					t.Fatalf("%.8s: log %q: %v", c.sig, l, err)
				}
			}
		}
		wantSOL := collateral - helio - dex
		if c.typ == types.TradeTypeBuy {
			wantSOL = collateral + helio + dex
			// buys pay with System transfers from the user
			paid := raw.sumTransfers(c.outer, c.inner+1, c.inner+12, func(x memeRawTransfer) bool { return x.Mint == solMint && x.From == user })
			if paid.Int64() != wantSOL {
				t.Fatalf("%.8s: log %d, transfers %s", c.sig, wantSOL, paid)
			}
		}
		tokens := raw.sumTransfers(c.outer, c.inner+1, c.inner+12, func(x memeRawTransfer) bool { return x.Mint == mint })

		e := memeEventAt(t, memeParse(t, c.sig), c.idx)
		sol, tok := e.OutputToken, e.InputToken
		if c.typ == types.TradeTypeBuy {
			sol, tok = e.InputToken, e.OutputToken
		}
		if e.Type != c.typ || e.User != user || e.Pool != curve || e.QuoteMint != solMint || e.BaseMint != mint {
			t.Errorf("%.8s: %s user %s pool %s quote %s", c.sig, e.Type, e.User, e.Pool, e.QuoteMint)
		}
		checkToken(t, c.sig[:8]+" SOL", sol, solMint, fmt.Sprint(wantSOL), 9)
		checkToken(t, c.sig[:8]+" token", tok, mint, tokens.String(), raw.decimals[mint])
		if feeOf(e.Fees, "dex") != fmt.Sprint(dex) || feeOf(e.Fees, "helio") != fmt.Sprint(helio) {
			t.Errorf("%.8s: fees %+v", c.sig, e.Fees)
		}

		ctx := newParseContext(tx, nil)
		trades := meme.NewMoonitParser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions, ctx.Classifier.GetInstructions(constants.DEX_PROGRAMS.MOONIT.ID)).ProcessTrades()
		if len(trades) != 1 || trades[0].User != user || trades[0].Pool[0] != curve ||
			trades[0].InputToken.AmountRaw != e.InputToken.AmountRaw || trades[0].OutputToken.AmountRaw != e.OutputToken.AmountRaw {
			t.Errorf("%.8s: Moonit trades %+v", c.sig, trades)
		}
	}
}

func reverse(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[i] = b[len(b)-1-i]
	}
	return out
}

// meme-17: the Boop.fun CREATE event had no bonding curve; it is account 2
// of the deploy_bonding_curve instruction for the same mint, which the buy
// of the same transaction trades against.
func TestMemeBoopfunCreateBondingCurve(t *testing.T) {
	r := memeParse(t, sigBoopCreateBuy)
	create := memeEventAt(t, r, "2")
	raw := loadMemeRaw(t, sigBoopCreateBuy)
	deploy := raw.accounts(3, -1)
	if create.Type != types.TradeTypeCreate || create.BaseMint != deploy[0] || create.BondingCurve != deploy[2] || create.PlatformConfig != deploy[5] ||
		create.BondingCurve != memeTradeAt(t, r, "5").Pool[0] {
		t.Errorf("create %+v, deploy accounts %v", create, deploy[:6])
	}
}

// Without usable logs (missing or truncated) Moonit falls back to the
// instruction's transfers and args instead of attributing a TradeEvent to
// the wrong instruction. Synthetic: the real 4wQnGcxk buy and 2XYu86 sell
// with their log messages removed or truncated.
func TestMemeMoonitWithoutLogs(t *testing.T) {
	for _, mutate := range []func(logs []string) []string{
		func(logs []string) []string { return nil },
		func(logs []string) []string { return append(logs[:len(logs)/2:len(logs)/2], "Log truncated") },
	} {
		// buy: the SOL paid is the user's System transfers
		tx := cloneTx(t, loadFixture(t, sigMoonitBuy11))
		tx.Meta.LogMessages = mutate(tx.Meta.LogMessages)
		e := memeEventAt(t, dexparserParse(tx), "1")
		if e.InputToken.AmountRaw != "1213981" || e.OutputToken.AmountRaw != "3796278637331" || len(e.Fees) != 0 {
			t.Errorf("buy without logs: %+v %+v fees %+v", e.InputToken, e.OutputToken, e.Fees)
		}
		// sell: the curve pays lamports directly; the args give the quote
		tx = cloneTx(t, loadFixture(t, sigMoonitSell))
		tx.Meta.LogMessages = mutate(tx.Meta.LogMessages)
		e = memeEventAt(t, dexparserParse(tx), "2")
		if e.InputToken.AmountRaw != "59948049312246101" || e.OutputToken.Mint != solMint || e.OutputToken.AmountRaw != "1761102483" {
			t.Errorf("sell without logs: %+v %+v", e.InputToken, e.OutputToken)
		}
	}
}

// Program output that contains "failed", "success" or "invoke [" must not
// end or open an invocation frame: a "Program log:" line with those words
// made the Moonit TradeEvent logs unusable, so amounts fell back to the
// slippage-bounded args. Synthetic: the real 2XYu86 sell with such lines
// added inside the Moonit invocation; the result must equal the real one.
func TestMemeMoonitLogLinesWithFrameWords(t *testing.T) {
	want := memeEventAt(t, memeParse(t, sigMoonitSell), "2")
	if len(want.Fees) == 0 {
		t.Fatalf("real sell has no TradeEvent fees: %+v", want)
	}
	tx := cloneTx(t, loadFixture(t, sigMoonitSell))
	var logs []string
	for _, l := range tx.Meta.LogMessages {
		logs = append(logs, l)
		if l == "Program log: Instruction: Sell" {
			logs = append(logs,
				"Program log: slippage check failed once, retrying",
				"Program log: step success",
				"Program log: Program 11111111111111111111111111111111 invoke [2]",
				"Program return: "+constants.DEX_PROGRAMS.MOONIT.ID+" AAAA")
		}
	}
	if len(logs) != len(tx.Meta.LogMessages)+4 {
		t.Fatalf("fixture: no Sell log line")
	}
	tx.Meta.LogMessages = logs
	got := memeEventAt(t, dexparserParse(tx), "2")
	if got.InputToken.AmountRaw != want.InputToken.AmountRaw || got.OutputToken.AmountRaw != want.OutputToken.AmountRaw ||
		feeOf(got.Fees, "dex") != feeOf(want.Fees, "dex") || feeOf(got.Fees, "helio") != feeOf(want.Fees, "helio") {
		t.Errorf("with program output lines: %+v %+v fees %+v, want %+v %+v fees %+v",
			got.InputToken, got.OutputToken, got.Fees, want.InputToken, want.OutputToken, want.Fees)
	}
}
