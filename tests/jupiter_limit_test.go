package tests

import (
	"bytes"
	"encoding/base64"
	"math/big"
	"strings"
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

// jupLoggedEvents returns the payloads (after the 8-byte discriminator) of the
// Anchor events named event that sig logged with emit! ("Program data: ...").
func jupLoggedEvents(t *testing.T, sig, event string) [][]byte {
	t.Helper()
	raw, err := readFixture(sig, "json")
	if err != nil {
		t.Fatalf("%.8s: %v", sig, err)
	}
	var doc struct {
		Meta struct {
			LogMessages []string `json:"logMessages"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("%.8s: %v", sig, err)
	}
	disc := jupDisc("event:" + event)
	var out [][]byte
	for _, l := range doc.Meta.LogMessages {
		if !strings.HasPrefix(l, "Program data: ") {
			continue
		}
		b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(l, "Program data: "))
		if err == nil && len(b) >= 8 && bytes.Equal(b[:8], disc) {
			out = append(out, b[8:])
		}
	}
	return out
}

// TestJupiterLimitV1FlashFill: Limit Order v1 fills (flash_fill_order,
// historical: the keeper stopped filling v1 orders around 2025-01) are the
// maker's trade; before, v1 had no trade parser. Truth: the maker is
// flash_fill_order accounts[2], the mints accounts[9] and [11] (v1 IDL); the
// input is the TradeEvent in_amount (logged with emit!: order_key, taker,
// remaining_in, remaining_out, in_amount, out_amount), which the order's
// reserve paid out; the output is what the maker received and the fee what
// the program fee account (accounts[7]) received, together the TradeEvent
// out_amount. Covers token -> USDC, SOL input, SOL output and USDT output.
// The aggregate is the maker's fill. amm-18, R2-G4.
func TestJupiterLimitV1FlashFill(t *testing.T) {
	for _, sig := range []string{
		"2Yt5jma9Acb5pNPDFDXt11aQhC7EwMkvJxAbKtUyqc2x9QUK8aXRxQM7kNnKCnbRuDTHPYYssD2toduPjSoqaonx",
		"4T6PszRFbToRzGNFtP5vssqumu8edopLvMpVWkwcrTHStXLuWWMH1fbD8eB8HNoHvZFZkdXcUnfH6KBRoyGpYcNj",
		"3kq4diRGoo3S2v4MxVUw3n8sa3MmBap1zjX377VXMPA1q7FUB1bMYNoTBqy1RiihXjbNhmn8cWyQMk25Rf3HJ5Tt",
		"5GR9DsLakYy6S6qhkNY4wmpWCFWsRBGPuNoEjqL3dq5wALBkqwaduxRJ76tt3zqR1s5o8evLFy3LNWZujHPedKyf",
	} {
		tx := loadFixture(t, sig)
		_, r := parseFixture(t, sig, nil)
		fill := jupInstruction(t, sig, jupLimit1ID, "flash_fill_order")
		reserve, maker, makerOut, feeAccount := fill.Accounts[1], fill.Accounts[2], fill.Accounts[4], fill.Accounts[7]
		inMint, outMint := fill.Accounts[9], fill.Accounts[11]
		events := jupLoggedEvents(t, sig, "TradeEvent")
		if len(events) != 1 || len(events[0]) != 96 {
			t.Fatalf("%.8s: TradeEvents %d", sig, len(events))
		}
		inAmount, outAmount := jupU64(events[0][80:88]), jupU64(events[0][88:96])

		// what the reserve paid, the maker received and the fee account got
		delta := func(account, mint string) *big.Int {
			if mint == solMint {
				return lamportDelta(tx, account)
			}
			return accountTokenDelta(tx, account, mint)
		}
		paid := new(big.Int).Neg(delta(reserve, inMint))
		recvAccount := makerOut
		if outMint == solMint {
			recvAccount = maker
		}
		received, fee := delta(recvAccount, outMint), delta(feeAccount, outMint)
		if paid.Cmp(inAmount) != 0 || new(big.Int).Add(received, fee).Cmp(outAmount) != 0 || fee.Sign() <= 0 {
			t.Fatalf("%.8s: reserve paid %s maker got %s fee %s, TradeEvent %s -> %s", sig, paid, received, fee, inAmount, outAmount)
		}

		var got []types.TradeInfo
		for _, tr := range r.Trades {
			if tr.ProgramId == jupLimit1ID {
				got = append(got, tr)
			}
		}
		// the keeper's Jupiter v6 route that sourced the fill is not a trade
		// of its own (the Limit v1 program is a Jupiter order program)
		if len(got) != 1 || len(r.Trades) != 1 {
			t.Fatalf("%.8s: %d Limit v1 trades in %+v", sig, len(got), r.Trades)
		}
		tr := got[0]
		if agg := r.AggregateTrade; agg == nil || agg.User != maker || agg.InputToken.AmountRaw != inAmount.String() || agg.OutputToken.AmountRaw != received.String() {
			t.Errorf("%.8s: aggregate %+v", sig, r.AggregateTrade)
		}
		if tr.User != maker || tr.Idx != jupRawIdx(fill) ||
			tr.InputToken.Mint != inMint || tr.InputToken.AmountRaw != inAmount.String() ||
			tr.OutputToken.Mint != outMint || tr.OutputToken.AmountRaw != received.String() {
			t.Errorf("%.8s: trade %s user %s %s:%s -> %s:%s, want %s user %s %s:%s -> %s:%s", sig, tr.Idx, tr.User,
				tr.InputToken.Mint, tr.InputToken.AmountRaw, tr.OutputToken.Mint, tr.OutputToken.AmountRaw,
				jupRawIdx(fill), maker, inMint, inAmount, outMint, received)
		}
		if tr.Fee == nil || tr.Fee.Mint != outMint || tr.Fee.AmountRaw != fee.String() || tr.Fee.Recipient != feeAccount || tr.Fee.Type != "protocol" {
			t.Errorf("%.8s: fee %+v, want %s %s to %s", sig, tr.Fee, outMint, fee, feeAccount)
		}
	}
}
