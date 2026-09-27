package tests

import (
	"math/big"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/jupiter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// TestJupiterDCAFillFee: a DCA (Recurring) fill reports the DCA program's own
// fee from the Filled event, which equals CollectedFee. The Jupiter v6 hops of
// the keeper route no longer get an invented 0.1% "DCA fee"; they are dropped
// in favour of the DCA trade (core rule). amm-15.
func TestJupiterDCAFillFee(t *testing.T) {
	for _, pre := range []string{"2dpDUBWCdva5", "4mxr44yo5Qi7", "eR2M5iiSwNAR", "2euJJaq2LCag", "2TqzmwGqQrb7", "5pz55MX8fzgX", "u4jQ56eFanST", "L6Xt5dHZL7sb", "5BcChhE76GfA", "2WaR3JjxzbDQ"} {
		sig := jupFindSig(t, pre)
		tx, r := parseFixture(t, sig, tradesConfig())
		_, filled := jupEvents(t, sig, jupDCAID, "Filled")
		_, collected := jupEvents(t, sig, jupDCAID, "CollectedFee")
		if len(filled) != 1 || len(collected) != 1 {
			t.Fatalf("%.8s: %d Filled, %d CollectedFee", sig, len(filled), len(collected))
		}
		f := filled[0]
		user, inMint, outMint := base58.Encode(f[0:32]), base58.Encode(f[64:96]), base58.Encode(f[96:128])
		in, out, feeMint, fee := jupU64(f[128:136]), jupU64(f[136:144]), base58.Encode(f[144:176]), jupU64(f[176:184])
		if collectedFee := jupU64(collected[0][96:104]); collectedFee.Cmp(fee) != 0 {
			t.Fatalf("%.8s: Filled fee %s, CollectedFee %s", sig, fee, collectedFee)
		}
		if len(r.Trades) != 1 {
			t.Fatalf("%.8s: %d trades", sig, len(r.Trades))
		}
		tr := r.Trades[0]
		if tr.ProgramId != jupDCAID || tr.User != user || tr.InputToken.Mint != inMint || tr.InputToken.AmountRaw != in.String() ||
			tr.OutputToken.Mint != outMint || tr.OutputToken.AmountRaw != out.String() {
			t.Errorf("%.8s: trade %s %s %s:%s -> %s:%s", sig, tr.ProgramId, tr.User, tr.InputToken.Mint, tr.InputToken.AmountRaw, tr.OutputToken.Mint, tr.OutputToken.AmountRaw)
		}
		if tr.Fee == nil || tr.Fee.Mint != feeMint || tr.Fee.AmountRaw != fee.String() || len(tr.Fees) != 0 {
			t.Errorf("%.8s: fee %+v %v, want %s %s", sig, tr.Fee, tr.Fees, feeMint, fee)
		}
		for _, hop := range jupV6Trades(tx) {
			if hop.Fee != nil || len(hop.Fees) != 0 {
				t.Errorf("%.8s: Jupiter hop %s has fee %+v", sig, hop.Idx, hop.Fee)
			}
		}
	}
}

// TestJupiterOrderAMMDeterministic: the AMM of DCA, VA and Limit v2 trades
// was picked from a map iteration (4mxr44yo gave RaydiumV4 or 1Dex from run
// to run); it is now the first AMM in execution order. RaydiumV4 is the
// upstream TS result for 4mxr44yo. parity-13.
func TestJupiterOrderAMMDeterministic(t *testing.T) {
	cases := map[string]string{"4mxr44yo5Qi7": "RaydiumV4", "26yDmTJAh3yp": "", "5Rz1hFVZvzap": ""}
	p := dexparser.NewDexParser()
	for pre, want := range cases {
		sig := jupFindSig(t, pre)
		tx := loadFixture(t, sig)
		first := ""
		for i := 0; i < 64; i++ {
			trades := p.ParseTrades(tx, tradesConfig())
			if len(trades) != 1 {
				t.Fatalf("%.8s: %d trades", sig, len(trades))
			}
			if i == 0 {
				first = trades[0].AMM
			}
			if trades[0].AMM != first {
				t.Fatalf("%.8s: AMM %s then %s", sig, first, trades[0].AMM)
			}
		}
		if want != "" && first != want {
			t.Errorf("%.8s: AMM %s, want %s", sig, first, want)
		}
	}
}

// TestJupiterDCAOpenClose: DCA open/close transfers come from the Opened and
// Closed events: the deposited input (any mint) and the refunded unfilled
// input. They reported the signer's SOL balance change (rent and tx fee
// included) as a SOL transfer whatever the input mint. amm-21.
func TestJupiterDCAOpenClose(t *testing.T) {
	cfg := &types.ParseConfig{ParseType: types.ParseType{Transfer: true}}
	for _, pre := range []string{"4PvkHqhgTJa6", "2XG97y4BzqeU"} {
		sig := jupFindSig(t, pre)
		tx, r := parseFixture(t, sig, cfg)
		_, opened := jupEvents(t, sig, jupDCAID, "Opened")
		if len(opened) != 1 {
			t.Fatalf("%.8s: %d Opened", sig, len(opened))
		}
		user, deposited, inMint := base58.Encode(opened[0][0:32]), jupU64(opened[0][64:72]), base58.Encode(opened[0][72:104])
		open := jupInstruction(t, sig, jupDCAID, "open_dca_v2")
		inAta := open.Accounts[6] // open_dca_v2: dca, user, payer, input_mint, output_mint, user_ata, in_ata, ...
		if d := accountTokenDelta(tx, inAta, inMint); d.Cmp(deposited) != 0 {
			t.Fatalf("%.8s: DCA input account received %s, Opened %s", sig, d, deposited)
		}
		if len(r.Transfers) != 1 {
			t.Fatalf("%.8s: transfers %+v", sig, r.Transfers)
		}
		tf := r.Transfers[0]
		if tf.Type != "OpenDca" || tf.ProgramId != jupDCAID || tf.Info.Mint != inMint || tf.Info.TokenAmount.Amount != deposited.String() ||
			tf.Info.Destination != inAta || tf.Info.Authority != user || tf.Idx != jupRawIdx(open) {
			t.Errorf("%.8s: transfer %s %s %s -> %s auth %s idx %s", sig, tf.Type, tf.Info.Mint, tf.Info.TokenAmount.Amount, tf.Info.Destination, tf.Info.Authority, tf.Idx)
		}
	}

	// A user's close_dca returns the unfilled input to the user
	sig := jupFindSig(t, "2oSCggfbBGsA")
	tx, r := parseFixture(t, sig, cfg)
	_, closed := jupEvents(t, sig, jupDCAID, "Closed")
	if len(closed) != 1 {
		t.Fatalf("%.8s: %d Closed", sig, len(closed))
	}
	inMint, unfilled := base58.Encode(closed[0][72:104]), jupU64(closed[0][176:184])
	closeIx := jupInstruction(t, sig, jupDCAID, "close_dca")
	inAta := closeIx.Accounts[4] // close_dca: user, dca, input_mint, output_mint, in_ata, ...
	if d := accountTokenDelta(tx, inAta, inMint); new(big.Int).Neg(d).Cmp(unfilled) != 0 || closed[0][184] != 1 {
		t.Fatalf("%.8s: DCA input account delta %s, unfilled %s", sig, d, unfilled)
	}
	if len(r.Transfers) != 1 {
		t.Fatalf("%.8s: transfers %+v", sig, r.Transfers)
	}
	tf := r.Transfers[0]
	// the transfer is reported under the DCA program like OpenDca (R2-G3)
	if tf.Type != "CloseDca" || tf.ProgramId != jupDCAID || tf.Info.Mint != inMint || tf.Info.TokenAmount.Amount != unfilled.String() || tf.Info.Source != inAta {
		t.Errorf("%.8s: transfer %s prog %s %s %s from %s", sig, tf.Type, tf.ProgramId, tf.Info.Mint, tf.Info.TokenAmount.Amount, tf.Info.Source)
	}
}

// TestJupiterVAWithdraw: the VA withdraw event layout is 98 bytes but the
// length check allowed 90, so 90..97 bytes panicked; and a withdrawal of a
// token (not SOL) was dropped because the user's balance was looked up by
// token account instead of by owner. amm-16.
func TestJupiterVAWithdraw(t *testing.T) {
	for n := 0; n < 98; n++ {
		func() {
			defer func() {
				if v := recover(); v != nil {
					t.Fatalf("%d bytes: panic %v", n, v)
				}
			}()
			if _, err := jupiter.ParseJupiterVAWithdrawLayout(make([]byte, n)); err == nil {
				t.Errorf("%d bytes: no error", n)
			}
		}()
	}
	if _, err := jupiter.ParseJupiterVAWithdrawLayout(make([]byte, 98)); err != nil {
		t.Errorf("98 bytes: %v", err)
	}

	cfg := &types.ParseConfig{ParseType: types.ParseType{Transfer: true}}
	for _, pre := range []string{"4TV3SDoebSQj", "5HUr6dJnj5YP", "2Ya3925mhPCB", "8dzwQH14GZ4t"} {
		sig := jupFindSig(t, pre)
		tx, r := parseFixture(t, sig, cfg)
		_, events := jupEvents(t, sig, jupVAID, "Withdraw")
		if len(events) != 1 {
			t.Fatalf("%.8s: %d Withdraw events", sig, len(events))
		}
		mint, amount := base58.Encode(events[0][32:64]), jupU64(events[0][64:72])
		withdraw := jupInstruction(t, sig, jupVAID, "withdraw")
		user := withdraw.Accounts[1]
		var got *types.TransferData
		for i := range r.Transfers {
			if r.Transfers[i].Type == "withdraw" {
				got = &r.Transfers[i]
			}
		}
		if got == nil || got.Info.Mint != mint || got.Info.TokenAmount.Amount != amount.String() || got.Info.Destination != user {
			t.Fatalf("%.8s: transfers %+v, want %s %s to %s", sig, r.Transfers, mint, amount, user)
		}
		if mint != solMint {
			// the user's token accounts received exactly the withdrawn amount
			if d := ownerTokenDelta(tx, user, mint); d.Cmp(amount) != 0 || got.Info.DestinationBalance == nil {
				t.Errorf("%.8s: user delta %s, amount %s, balance %v", sig, d, amount, got.Info.DestinationBalance)
			}
		}
	}
}
