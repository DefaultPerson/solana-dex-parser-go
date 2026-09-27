package tests

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// TestJupiterLimitV2Fill: a Limit Order v2 (Trigger) fill is the maker's
// trade: user = the order's maker (fill_order accounts[1]) instead of the
// taker (the keeper); input = making_amount; output = what the maker's output
// account received; fee = the transfer to the order's fee account (was a fixed
// 0.1% of taking_amount). amm-15.
func TestJupiterLimitV2Fill(t *testing.T) {
	for _, pre := range []string{"26yDmTJAh3yp", "28PwqmbGYfpW", "4ZCKSArCUQTc", "5Rz1hFVZvzap"} {
		sig := jupFindSig(t, pre)
		tx, r := parseFixture(t, sig, tradesConfig())
		fill := jupInstruction(t, sig, jupLimit2ID, "fill_order")
		maker, feeAccount, inMint, outMint := fill.Accounts[1], fill.Accounts[6], fill.Accounts[8], fill.Accounts[10]
		_, events := jupEvents(t, sig, jupLimit2ID, "TradeEvent")
		if len(events) != 1 {
			t.Fatalf("%.8s: %d TradeEvents", sig, len(events))
		}
		taker := base58.Encode(events[0][32:64])
		making, taking := jupU64(events[0][80:88]), jupU64(events[0][88:96])

		received := ownerTokenDelta(tx, maker, outMint)
		fee := accountTokenDelta(tx, feeAccount, outMint)
		if received.Sign() <= 0 || fee.Sign() <= 0 || new(big.Int).Add(received, fee).Cmp(taking) != 0 {
			t.Fatalf("%.8s: maker received %s, fee %s, taking %s", sig, received, fee, taking)
		}
		if len(r.Trades) != 1 {
			t.Fatalf("%.8s: %d trades", sig, len(r.Trades))
		}
		tr := r.Trades[0]
		if tr.User != maker || tr.User == taker || tr.ProgramId != jupLimit2ID {
			t.Errorf("%.8s: user %s program %s, maker %s taker %s", sig, tr.User, tr.ProgramId, maker, taker)
		}
		if tr.InputToken.Mint != inMint || tr.InputToken.AmountRaw != making.String() ||
			tr.OutputToken.Mint != outMint || tr.OutputToken.AmountRaw != received.String() {
			t.Errorf("%.8s: %s:%s -> %s:%s, want %s:%s -> %s:%s", sig, tr.InputToken.Mint, tr.InputToken.AmountRaw,
				tr.OutputToken.Mint, tr.OutputToken.AmountRaw, inMint, making, outMint, received)
		}
		if tr.Fee == nil || tr.Fee.Mint != outMint || tr.Fee.AmountRaw != fee.String() || tr.Fee.Recipient != feeAccount {
			t.Errorf("%.8s: fee %+v, want %s to %s", sig, tr.Fee, fee, feeAccount)
		}
	}
}

// TestJupiterLimitV2CancelDustOrder: cancel_dust_order (limit_order_2 IDL:
// signer, maker, order, input_mint_reserve, maker_input_mint_account,
// input_mint, ...) returns the order's remaining input to the maker like
// cancel_order and is reported as a cancelDustOrder transfer. Synthetic: no
// cancel_dust_order was found in 133 sampled Limit v2 transactions
// (2026-09), so the real cancel_order 25PcXrRK gets the cancel_dust_order
// discriminator. amm-18.
func TestJupiterLimitV2CancelDustOrder(t *testing.T) {
	sig := jupFindSig(t, "25PcXrRKekZ8")
	cfg := &types.ParseConfig{ParseType: types.ParseType{Transfer: true}}
	_, want := parseFixture(t, sig, cfg)
	if len(want.Transfers) == 0 || want.Transfers[0].Type != "cancelOrder" {
		t.Fatalf("cancel_order transfers %+v", want.Transfers)
	}

	raw, err := readFixture(sig, "json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	replaced := 0
	for _, x := range doc["transaction"].(map[string]interface{})["message"].(map[string]interface{})["instructions"].([]interface{}) {
		ix := x.(map[string]interface{})
		data, _ := base58.Decode(ix["data"].(string))
		if len(data) >= 8 && bytes.Equal(data[:8], jupDisc("global:cancel_order")) {
			ix["data"] = base58.Encode(jupDisc("global:cancel_dust_order"))
			replaced++
		}
	}
	if replaced != 1 {
		t.Fatalf("%d instructions replaced", replaced)
	}
	raw, _ = json.Marshal(doc)
	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(raw, &tx); err != nil {
		t.Fatal(err)
	}
	got := dexparser.NewDexParser().ParseAll(&tx, cfg)
	if len(got.Transfers) != len(want.Transfers) {
		t.Fatalf("transfers %+v, want %+v", got.Transfers, want.Transfers)
	}
	for i := range got.Transfers {
		g, w := got.Transfers[i], want.Transfers[i]
		if g.Type != "cancelDustOrder" || g.Info.Mint != w.Info.Mint || g.Info.TokenAmount.Amount != w.Info.TokenAmount.Amount || g.Info.Destination != w.Info.Destination {
			t.Errorf("transfer %d: %+v, want %+v", i, g, w)
		}
	}
}
