package tests

import (
	"math/big"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// TestJupiterZFill: a Jupiter Z (order_engine RFQ) fill is the taker's trade:
// user = taker (accounts[0]), input = input_amount of input_mint
// (accounts[6]), output = output_amount of output_mint (accounts[8]), per the
// on-chain IDL. Without a parser the unknown-DEX fallback reported it from the
// fee payer's view: user = maker, side inverted. amm-7.
func TestJupiterZFill(t *testing.T) {
	for _, pre := range []string{"4tke4p2A2RqX", "xZeKHSc3fSkL", "24U4tyX7TDbU", "4rfgo1jxZFJz", "5nM2GFnn5vAf"} {
		sig := jupFindSig(t, pre)
		tx, r := parseFixture(t, sig, nil)
		fill := jupInstruction(t, sig, jupZID, "fill")
		taker, maker, inMint, outMint := fill.Accounts[0], fill.Accounts[1], fill.Accounts[6], fill.Accounts[8]
		in, out := jupU64(fill.Data[8:16]), jupU64(fill.Data[16:24])
		if rawAccountKeys(tx)[0] == taker {
			t.Fatalf("%.8s: the taker pays the fee", sig)
		}

		// The taker's balances moved by the fill amounts; a native SOL leg
		// shows on the maker's WSOL account instead
		paid := new(big.Int).Neg(ownerTokenDelta(tx, taker, inMint))
		if paid.Sign() == 0 {
			paid = ownerTokenDelta(tx, maker, inMint)
		}
		received := ownerTokenDelta(tx, taker, outMint)
		if received.Sign() == 0 {
			received = new(big.Int).Neg(ownerTokenDelta(tx, maker, outMint))
		}
		if paid.Cmp(in) != 0 || received.Cmp(out) != 0 {
			t.Fatalf("%.8s: taker paid %s received %s, fill %s -> %s", sig, paid, received, in, out)
		}

		if len(r.Trades) != 1 || r.AggregateTrade == nil {
			t.Fatalf("%.8s: trades %+v", sig, r.Trades)
		}
		for _, tr := range []types.TradeInfo{r.Trades[0], *r.AggregateTrade} {
			if tr.User != taker || tr.AMM != "JupiterZ" || tr.InputToken.Mint != inMint || tr.InputToken.AmountRaw != in.String() ||
				tr.OutputToken.Mint != outMint || tr.OutputToken.AmountRaw != out.String() {
				t.Errorf("%.8s: %s %s %s:%s -> %s:%s", sig, tr.User, tr.AMM, tr.InputToken.Mint, tr.InputToken.AmountRaw, tr.OutputToken.Mint, tr.OutputToken.AmountRaw)
			}
		}
		if r.Trades[0].ProgramId != jupZID || r.Trades[0].Idx != jupRawIdx(fill) {
			t.Errorf("%.8s: program %s idx %s", sig, r.Trades[0].ProgramId, r.Trades[0].Idx)
		}
	}
	// 4tke4p2A: the taker sold Grass for USDT
	_, r := parseFixture(t, jupFindSig(t, "4tke4p2A2RqX"), nil)
	if r.Trades[0].Type != types.TradeTypeSell {
		t.Errorf("4tke4p2A: type %s", r.Trades[0].Type)
	}
}
