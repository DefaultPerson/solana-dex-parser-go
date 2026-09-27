package tests

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/jupiter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// jupRawIdx formats the idx of a raw instruction like the parser does.
func jupRawIdx(x jupRawIx) string {
	if x.Inner < 0 {
		return fmt.Sprintf("%d", x.Outer)
	}
	return fmt.Sprintf("%d-%d", x.Outer, x.Inner)
}

// jupV6Trades runs the Jupiter v6 parser alone on tx.
func jupV6Trades(tx *adapter.SolanaTransaction) []types.TradeInfo {
	ctx := newParseContext(tx, nil)
	p := jupiter.NewJupiterParser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions, ctx.Classifier.GetInstructions(jupV6ID))
	return p.ProcessTrades()
}

// jupCheckHops asserts one trade per decoded hop, in order, with the hop's
// mints and amounts, the hop AMM's name, and an idx that points at an
// instruction of the hop's AMM program (or at the event for legacy SwapEvent).
func jupCheckHops(t *testing.T, sig string, trades []types.TradeInfo, hops []jupHop, hopIxs []jupRawIx) {
	t.Helper()
	if len(trades) != len(hops) {
		t.Errorf("%.8s: %d trades, %d hops", sig, len(trades), len(hops))
		return
	}
	byIdx := make(map[string]jupRawIx)
	for _, x := range jupRawInstructions(t, sig) {
		byIdx[jupRawIdx(x)] = x
	}
	seen := make(map[string]bool)
	for i, h := range hops {
		tr := trades[i]
		if tr.InputToken.Mint != h.InMint || tr.InputToken.AmountRaw != h.In.String() ||
			tr.OutputToken.Mint != h.OutMint || tr.OutputToken.AmountRaw != h.Out.String() {
			t.Errorf("%.8s hop %d: %s:%s -> %s:%s, event %s:%s -> %s:%s", sig, i,
				tr.InputToken.Mint, tr.InputToken.AmountRaw, tr.OutputToken.Mint, tr.OutputToken.AmountRaw, h.InMint, h.In, h.OutMint, h.Out)
		}
		name := constants.GetProgramName(h.AMM)
		if tr.AMM != name || tr.AMM == "" {
			t.Errorf("%.8s hop %d: AMM %q, want %q", sig, i, tr.AMM, name)
		}
		if seen[tr.Idx] {
			t.Errorf("%.8s hop %d: duplicate idx %s", sig, i, tr.Idx)
		}
		seen[tr.Idx] = true
		x, ok := byIdx[tr.Idx]
		switch {
		case hopIxs != nil:
			if !ok || jupRawIdx(hopIxs[i]) != tr.Idx {
				t.Errorf("%.8s hop %d: idx %s, event at %s", sig, i, tr.Idx, jupRawIdx(hopIxs[i]))
			}
		case !ok || x.Program != h.AMM:
			t.Errorf("%.8s hop %d: idx %s is not an instruction of %s", sig, i, tr.Idx, h.AMM)
		}
	}
}

// TestJupiterRouteV2SwapsEvent: the Jupiter v6 route_v2 family
// (route_v2, shared_accounts_route_v2, exact_out_route_v2,
// shared_accounts_exact_out_route_v2) emits one SwapsEvent listing every hop
// (Vec<SwapEventV2{input_mint, input_amount, output_mint, output_amount,
// amm}>, on-chain JUP6 IDL) instead of one SwapEvent per hop. The parser
// ignored it and produced no Jupiter trade. amm-3, core-23, constants-1,
// parity-14; 5sV51Yrw is a version-1 transaction (amm-23).
func TestJupiterRouteV2SwapsEvent(t *testing.T) {
	prefixes := []string{
		"1V8cnQVrAApj", "2jm6y95jSBHP", "2zRhMn1ADKEU", "34sGGDUK4A1x", "NeF1UiWXKUbu", // shared_accounts_route_v2
		"2d32B4VReyxf", "3VdmWK159atQ", "4YEiZ5wX7cJj", "5pXteqTXGiFk", "3pnp1kpTvnZM", // route_v2
		"5iaG7mCfMRQx", "3cbL8koKqz7s", "3Nb6n7meLAN6", "4wFykYbVDgXK", "5sV51YrwGRFw",
	}
	kinds := make(map[string]int)
	for _, pre := range prefixes {
		sig := jupFindSig(t, pre)
		for _, x := range jupRawInstructions(t, sig) {
			if x.Program != jupV6ID || len(x.Data) < 8 {
				continue
			}
			for _, n := range []string{"route_v2", "shared_accounts_route_v2", "exact_out_route_v2", "shared_accounts_exact_out_route_v2"} {
				if bytes.Equal(x.Data[:8], jupDisc("global:"+n)) {
					kinds[n]++
				}
			}
		}
		_, payloads := jupEvents(t, sig, jupV6ID, "SwapsEvent")
		if len(payloads) != 1 {
			t.Fatalf("%.8s: %d SwapsEvents", sig, len(payloads))
		}
		hops := jupDecodeSwapsEvent(payloads[0])
		tx := loadFixture(t, sig)
		jupCheckHops(t, sig, jupV6Trades(tx), hops, nil)

		// Through ParseAll the signer's own token legs match the route's
		// first input and last output. Arbitrages are excluded: 5sV51Yrw
		// wraps the route in another program's instruction, 3Nb6n7me routes
		// a token bought earlier in the transaction.
		if pre == "5sV51YrwGRFw" || pre == "3Nb6n7meLAN6" {
			continue
		}
		_, r := parseFixture(t, sig, nil)
		agg := r.AggregateTrade
		if agg == nil {
			t.Errorf("%.8s: no aggregate", sig)
			continue
		}
		// the aggregate lists every hop AMM in execution order (amm-14)
		var amms []string
		for _, h := range hops {
			if name := constants.GetProgramName(h.AMM); !containsStr(amms, name) {
				amms = append(amms, name)
			}
		}
		if len(hops) > 1 && fmt.Sprint(agg.AMMs) != fmt.Sprint(amms) {
			t.Errorf("%.8s: aggregate AMMs %v, hops %v", sig, agg.AMMs, amms)
		}
		signer := rawAccountKeys(tx)[0]
		if agg.InputToken.Mint == agg.OutputToken.Mint {
			continue
		}
		if d := ownerTokenDelta(tx, signer, agg.InputToken.Mint); d.Sign() < 0 && new(big.Int).Neg(d).String() != agg.InputToken.AmountRaw {
			t.Errorf("%.8s: aggregate input %s %s, signer delta %s", sig, agg.InputToken.Mint, agg.InputToken.AmountRaw, d)
		}
		if d := ownerTokenDelta(tx, signer, agg.OutputToken.Mint); d.Sign() > 0 && d.String() != agg.OutputToken.AmountRaw {
			t.Errorf("%.8s: aggregate output %s %s, signer delta %s", sig, agg.OutputToken.Mint, agg.OutputToken.AmountRaw, d)
		}
	}
	if kinds["route_v2"] == 0 || kinds["shared_accounts_route_v2"] == 0 {
		t.Errorf("route kinds covered: %v", kinds)
	}
}

// TestJupiterRouteV1SwapEvent: the legacy routes keep one trade per SwapEvent
// hop, labelled with the hop AMM.
func TestJupiterRouteV1SwapEvent(t *testing.T) {
	for _, pre := range []string{"5vjkR1vooYfu", "5XZyrL4mfCZd", "33Pob2j9ZRVJ", "2WMyNETEo1WA", "46ncRkgcUzF1"} {
		sig := jupFindSig(t, pre)
		ixs, payloads := jupEvents(t, sig, jupV6ID, "SwapEvent")
		if len(payloads) == 0 {
			t.Fatalf("%.8s: no SwapEvent", sig)
		}
		var hops []jupHop
		for _, p := range payloads {
			hops = append(hops, jupDecodeSwapEvent(p))
		}
		jupCheckHops(t, sig, jupV6Trades(loadFixture(t, sig)), hops, ixs)
	}
}

// TestJupiterFeeEvent: the platform fee of a Jupiter v6 route is emitted as
// FeeEvent {account, mint, amount} (JUP6 IDL). It becomes the Fee of the
// route's first hop when taken from the input (the event precedes the hops)
// and of its last hop when taken from the output; that hop's amount then is
// what the user paid (gross input) or received (net output). amm-15, amm-3.
func TestJupiterFeeEvent(t *testing.T) {
	cases := []struct {
		prefix  string
		fromOut bool
	}{
		{"3TZKJLxy4H2w", false}, // SOL -> token, fee 850 lamports of WSOL from the input
		{"46rMXoH7t2Sy", true},  // token -> SOL, fee from the output
		{"4FYCvxfq6Tfo", true},  // 3 hops, fee in the output token
		{"4txewy5B76FN", false}, // 2 hops, fee in the input token
	}
	for _, c := range cases {
		sig := jupFindSig(t, c.prefix)
		tx, r := parseFixture(t, sig, nil)
		_, payloads := jupEvents(t, sig, jupV6ID, "FeeEvent")
		if len(payloads) != 1 {
			t.Fatalf("%.8s: %d FeeEvents", sig, len(payloads))
		}
		p := payloads[0]
		account, mint, amount := base58.Encode(p[0:32]), base58.Encode(p[32:64]), jupU64(p[64:72])

		var withFee []types.TradeInfo
		for _, tr := range r.Trades {
			if tr.Fee != nil {
				withFee = append(withFee, tr)
			}
		}
		if len(withFee) != 1 {
			t.Fatalf("%.8s: %d trades with a fee", sig, len(withFee))
		}
		fee := withFee[0].Fee
		if fee.Mint != mint || fee.AmountRaw != amount.String() || fee.Recipient != account || fee.Type != "platform" {
			t.Errorf("%.8s: fee %+v, FeeEvent %s %s -> %s", sig, *fee, mint, amount, account)
		}
		hop := withFee[0].Idx
		first, last := r.Trades[0].Idx, r.Trades[len(r.Trades)-1].Idx
		if (c.fromOut && hop != last) || (!c.fromOut && hop != first) {
			t.Errorf("%.8s: fee on hop %s (first %s, last %s)", sig, hop, first, last)
		}

		// The aggregate is what the signer paid and received
		agg := r.AggregateTrade
		signer := rawAccountKeys(tx)[0]
		paid, received := new(big.Int), new(big.Int)
		if agg.InputToken.Mint == solMint {
			paid.Neg(lamportDelta(tx, signer)).Sub(paid, new(big.Int).SetUint64(tx.Meta.Fee))
		} else {
			paid.Neg(ownerTokenDelta(tx, signer, agg.InputToken.Mint))
		}
		if agg.OutputToken.Mint == solMint {
			received.Add(lamportDelta(tx, signer), new(big.Int).SetUint64(tx.Meta.Fee))
		} else {
			received.Set(ownerTokenDelta(tx, signer, agg.OutputToken.Mint))
		}
		if agg.InputToken.AmountRaw != paid.String() || agg.OutputToken.AmountRaw != received.String() {
			t.Errorf("%.8s: aggregate %s -> %s, signer paid %s received %s", sig, agg.InputToken.AmountRaw, agg.OutputToken.AmountRaw, paid, received)
		}
		aggFee := agg.Fee
		for i := range agg.Fees {
			if agg.Fees[i].AmountRaw == amount.String() {
				aggFee = &agg.Fees[i]
			}
		}
		if aggFee == nil || aggFee.AmountRaw != amount.String() || aggFee.Mint != mint {
			t.Errorf("%.8s: aggregate fee %+v / %v", sig, agg.Fee, agg.Fees)
		}
	}
}

// TestJupiterHopUnknownAMM: a hop through a program without a known name is
// labelled "Unknown" and keeps the AMM's program id in ProgramId (it used to
// carry the Jupiter program id, losing the venue). Synthetic: no fixture has
// a Jupiter hop through an unnamed program (all current venues are in the
// constants), so the SwapEvent AMM of the real route 5vjkR1vo is replaced by
// an unregistered id. amm-14.
func TestJupiterHopUnknownAMM(t *testing.T) {
	unknown := base58.Encode(bytes.Repeat([]byte{7}, 32))
	if constants.GetProgramName(unknown) != "Unknown" {
		t.Fatalf("%s is registered", unknown)
	}
	raw, err := readFixture(sigJupPumpswap, "json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	replaced := 0
	for _, set := range doc["meta"].(map[string]interface{})["innerInstructions"].([]interface{}) {
		for _, x := range set.(map[string]interface{})["instructions"].([]interface{}) {
			ix := x.(map[string]interface{})
			data, _ := base58.Decode(ix["data"].(string))
			if len(data) >= 16+112 && bytes.Equal(data[8:16], jupDisc("event:SwapEvent")) {
				copy(data[16:48], bytes.Repeat([]byte{7}, 32))
				ix["data"] = base58.Encode(data)
				replaced++
			}
		}
	}
	if replaced != 1 {
		t.Fatalf("%d SwapEvents replaced", replaced)
	}
	raw, _ = json.Marshal(doc)
	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(raw, &tx); err != nil {
		t.Fatal(err)
	}
	trades := jupV6Trades(&tx)
	if len(trades) != 1 || trades[0].AMM != "Unknown" || trades[0].ProgramId != unknown {
		t.Errorf("trades %+v", trades)
	}

	// A named hop keeps the Jupiter program id and the AMM name
	named := jupV6Trades(loadFixture(t, sigJupPumpswap))
	if len(named) != 1 || named[0].AMM != "Pumpswap" || named[0].ProgramId != jupV6ID {
		t.Errorf("named hop %+v", named)
	}

	// No Jupiter hop in the fixture pool is left without an AMM label
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		found := false
		for _, k := range rawAccountKeys(tx) {
			found = found || k == jupV6ID
		}
		if !found {
			continue
		}
		for _, tr := range jupV6Trades(tx) {
			if tr.AMM == "" || tr.ProgramId == "" {
				t.Errorf("%.8s %s: AMM %q ProgramId %q", sig, tr.Idx, tr.AMM, tr.ProgramId)
			}
		}
	}
}
