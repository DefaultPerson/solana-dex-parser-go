package tests

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/dflow"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/jupiter"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/meteora"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/photon"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/systoken"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Robustness and output-shape regression tests for ShredParser.
// shred-5, shred-6, shred-12, shred-13, shred-29, shred-30.

const (
	// 20 outer System transfers (idx 0-19)
	sigSystemTransfers = "Z6u214mPxbJn8Ug7eA7AwaoZij4U7F8VpRrgx79dNaUstkptYDGcmssHVZH2wDJUEEDhuQQ1b82fVafDF22pKcB"
	// legacy Pump.fun buy at outer 5
	sigPumpBuyLegacy = "v8s37Srj6QPMtRC1HfJcrSenCHvYebHiGkHVuFFiQ6UviqHnoVx4U77M3TZhQQXewXadHYh5t35LkesJi3ztPZZ"
	// Jupiter v6 route (v1 layout) at outer 4
	sigJupRouteV1 = "5jNLWdMv1CfPjhGE6oKAj7gugdJa2EbrmJwbdEPyxdDAXxYFPAfNEK3braDoKdgSS7sYLWUgZfdt5vd1UiR2GuRR"
	// Jupiter v6 route_v2 at outer 5
	sigJupRouteV2 = "5qJs7ws4UY3qmtPZf5R2LbBQWjkUNEyzdC6gAXssYRYMwWoo1qBQ61Mk8owtXYUSPouLY2z4i4wnohdKCyxgtrkS"
)

// TestShredNilTransaction: a nil transaction made the deferred recover
// handler dereference tx and the panic escaped ParseAll. shred-5.
func TestShredNilTransaction(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ParseAll(nil) panicked: %v", r)
		}
	}()
	res := dexparser.NewShredParser().ParseAll(nil, nil)
	if res == nil || res.State || res.Msg != "nil transaction" {
		t.Fatalf("ParseAll(nil) = %+v, want State=false Msg=\"nil transaction\"", res)
	}
}

// TestShredPhotonHopBounds: a Photon two_hop_swap (Raydium V4 -> Meteora DBC)
// with 17 accounts read accounts[17]; the recovered panic made ParseAll
// return nil and dropped the other programs' results. No real Photon
// two_hop_swap exists (the instruction was never observed on-chain), so the
// test builds one: a real tx (20 System transfers) gets an extra outer Photon
// instruction. shred-5.
func TestShredPhotonHopBounds(t *testing.T) {
	tx := cloneTx(t, loadFixture(t, sigSystemTransfers))
	keys := &tx.Transaction.Message.AccountKeys
	add := func(k string) int {
		*keys = append(*keys, adapter.AccountKey{Pubkey: k})
		return len(*keys) - 1
	}
	photonIdx := add(constants.DEX_PROGRAMS.PHOTON.ID)
	rayIdx := add(constants.DEX_PROGRAMS.RAYDIUM_V4.ID)
	dbcIdx := add(constants.DEX_PROGRAMS.METEORA_DBC.ID)
	accounts := make([]interface{}, 17)
	for i := range accounts {
		accounts[i] = 0
	}
	accounts[6], accounts[11] = rayIdx, dbcIdx
	data := append(append([]byte{}, constants.DISCRIMINATORS.PHOTON.HOP_TWO_SWAP...), make([]byte, 16)...)
	tx.Transaction.Message.Instructions = append(tx.Transaction.Message.Instructions, map[string]interface{}{
		"programIdIndex": photonIdx, "accounts": accounts, "data": base58.Encode(data),
	})

	res := dexparser.NewShredParser().ParseAll(tx, nil)
	if res == nil {
		t.Fatal("ParseAll returned nil")
	}
	if !res.State {
		t.Fatalf("State=false Msg=%q", res.Msg)
	}
	if n := len(res.Instructions["System"]); n != 20 {
		t.Errorf("System transfers = %d, want the 20 of the tx", n)
	}
	if _, ok := res.Instructions["Photon"]; ok {
		t.Errorf("malformed two_hop_swap decoded: %+v", res.Instructions["Photon"])
	}
}

// TestShredSystemTokenReachable: GetAllProgramIds drops the System and Token
// programs, so their shred branches never ran. shred-13.
func TestShredSystemTokenReachable(t *testing.T) {
	tx := loadFixture(t, sigSystemTransfers)
	for _, mode := range []string{"full", "pre-exec"} {
		in := tx
		if mode == "pre-exec" {
			in = preExec(t, tx)
		}
		res := parseShred(t, in, nil)
		events := res.Instructions["System"]
		if len(events) != 20 {
			t.Fatalf("%s: System transfers = %d, want 20", mode, len(events))
		}
		// Each outer instruction is a System transfer: tag 2, lamports u64
		for i, ix := range fixtureIxs(t, tx)[:20] {
			want := le64At(ix.data, 4)
			got := events[i].(*systoken.TokenInstruction).Data
			if got.Idx != utils.FormatIdx(i, -1) || got.Info.Source != ix.accounts[0] || got.Info.Destination != ix.accounts[1] {
				t.Errorf("%s: transfer %d = %+v, want %s -> %s at idx %d", mode, i, got, ix.accounts[0], ix.accounts[1], i)
			}
			if got.Info.TokenAmount.Amount != u64str(want) {
				t.Errorf("%s: transfer %d amount = %s, want %d", mode, i, got.Info.TokenAmount.Amount, want)
			}
		}
	}

	// Token transfers of an executed tx (inner instructions of a Jupiter route)
	res := parseShred(t, loadFixture(t, sigJupRouteV1), nil)
	if len(res.Instructions["Token"]) == 0 {
		t.Error("no Token transfers decoded")
	}
}

// TestShredTypedOrderNumeric: typed instructions of different programs are
// returned in numeric idx order ("2" before "10"). shred-29.
func TestShredTypedOrderNumeric(t *testing.T) {
	res := parseShred(t, loadFixture(t, sigSystemTransfers), nil)
	if len(res.ParsedInstructions) < 20 {
		t.Fatalf("typed instructions %d, want the 20 System transfers", len(res.ParsedInstructions))
	}
	for i := 1; i < len(res.ParsedInstructions); i++ {
		a, b := res.ParsedInstructions[i-1].Idx, res.ParsedInstructions[i].Idx
		if utils.CompareIdx(a, b) > 0 {
			t.Fatalf("typed instructions out of order: %s before %s", a, b)
		}
	}
}

// TestShredPumpEventsNumericOrder: Pump.fun events were sorted by idx
// string, so "10" came before "2". No real tx has Pump.fun instructions at
// outer indexes below and above 10, so the test repeats the real buy of
// v8s37 twelve times. shred-29.
func TestShredPumpEventsNumericOrder(t *testing.T) {
	tx := preExec(t, loadFixture(t, sigPumpBuyLegacy))
	buy := findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, constants.DISCRIMINATORS.PUMPFUN.BUY)
	tx.Transaction.Message.Instructions = nil
	for i := 0; i < 12; i++ {
		ix := map[string]interface{}{}
		for k, v := range buy.m {
			ix[k] = v
		}
		tx.Transaction.Message.Instructions = append(tx.Transaction.Message.Instructions, ix)
	}
	res := parseShred(t, tx, nil)
	events := res.Instructions[constants.DEX_PROGRAMS.PUMP_FUN.Name]
	if len(events) != 12 {
		t.Fatalf("Pump.fun events = %d, want 12", len(events))
	}
	for i, e := range events {
		if idx := e.(*dexparser.PumpfunInstruction).Idx; idx != utils.FormatIdx(i, -1) {
			t.Fatalf("event %d idx = %s, want %d (order %v)", i, idx, i, events)
		}
	}
}

// TestShredMalformedDataNoEvents: a decoder that rejected truncated data
// still emitted an event with Data=null (a typed nil pointer in
// interface{}), and Pump.fun/PumpSwap stored null for programs without
// events. Each case truncates the arguments of a real instruction to 3
// bytes: no event, no typed instruction and no empty program entry may
// remain. shred-12, shred-30.
func TestShredMalformedDataNoEvents(t *testing.T) {
	cases := []struct {
		name, sig, program string
		prefix             []byte
	}{
		{"RaydiumV4 swap", "5kaAWK5X9DdMmsWm6skaUXLd6prFisuYJavd9B62A941nRGcrmwvncg3tRtUfn7TcMLsrrmjCChdEjK3sjxS6YG9", constants.DEX_PROGRAMS.RAYDIUM_V4.ID, constants.DISCRIMINATORS.RAYDIUM.SWAP},
		{"Pumpfun buy", sigPumpBuyLegacy, constants.DEX_PROGRAMS.PUMP_FUN.ID, constants.DISCRIMINATORS.PUMPFUN.BUY},
		{"Pumpswap buy", "4b6jX4P8vqKUPJLLFWLxQ3gWH25qgy3uxQqQ98FW7tmUnk29PqWXjNwDeCb3tkP4w2MDtLWjqQYRK42abVnfz55m", constants.DEX_PROGRAMS.PUMP_SWAP.ID, constants.DISCRIMINATORS.PUMPSWAP.BUY},
		{"Jupiter route_v2", sigJupRouteV2, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE_V2},
		{"LaunchLab buy_exact_in", "Gi44zBwsd8eUGEVPS1jstts457hKLbm8SSMLrRVHVK2McrhJjosiszb65U1LdrjsF1WfCXoesLMhm8RX3dchx4s", constants.DEX_PROGRAMS.RAYDIUM_LCP.ID, constants.DISCRIMINATORS.RAYDIUM_LCP.BUY_EXACT_IN},
		{"DBC swap", "5Y1z7B8doTaXMjqzCtRSnTPbSkpU3Ryjym857NVBBfdYLkoVP95o7V7KPdJZXNdsMXoosqghhb6aNYCmW42Ugs6t", constants.DEX_PROGRAMS.METEORA_DBC.ID, constants.DISCRIMINATORS.METEORA_DBC.SWAP},
		{"Photon pump_buy_v2", "61tXHLrQufhFiHbjq8z4Ugm1TDY8ig65MfESi7sQCFDRN18gszH8crcNbaMHVxvEHrtY8o5xMddJ58X9gvp7GcQA", constants.DEX_PROGRAMS.PHOTON.ID, constants.DISCRIMINATORS.PHOTON.PUMPFUN_BUY_V2},
		{"DFlow swap", "4VsduK7H3ZmrK9fqDCXu59NuhZxG9g5zmRUBZ71efVL5vU14W8EN7Ew8RisJVxFxmEh9VhaPwNqhBgKZBndjGzrk", constants.DEX_PROGRAMS.DFLOW.ID, constants.DISCRIMINATORS.DFLOW.SWAP},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tx := preExec(t, loadFixture(t, c.sig))
			ix := findIx(t, tx, c.program, c.prefix)
			if ix.inner >= 0 {
				t.Fatalf("instruction is not outer")
			}
			setIxData(ix, append(append([]byte{}, ix.data[:len(c.prefix)]...), 1, 2, 3))

			res := parseShred(t, tx, nil)
			for name, events := range res.Instructions {
				if len(events) == 0 {
					t.Errorf("program %s has an empty event list", name)
				}
				for _, e := range events {
					if data := eventData(e); data == nil {
						t.Errorf("program %s: event with nil Data: %+v", name, e)
					}
				}
			}
			if got := typedAt(res, c.program, utils.FormatIdx(ix.outer, -1)); len(got) != 0 {
				t.Errorf("typed instruction decoded from truncated data: %+v", got)
			}
		})
	}
}

// eventData returns the Data of a legacy shred event, or nil for a nil
// pointer stored in the interface
func eventData(e interface{}) interface{} {
	var d interface{}
	switch ev := e.(type) {
	case *dexparser.PumpfunInstruction:
		d = ev.Data
	case *dexparser.PumpswapInstruction:
		d = ev.Data
	case *jupiter.JupiterShredInstruction:
		d = ev.Data
	case *raydium.RaydiumV4ShredInstruction:
		d = ev.Data
	case *raydium.LaunchpadShredInstruction:
		d = ev.Data
	case *meteora.DBCShredInstruction:
		d = ev.Data
	case *photon.PhotonInstruction:
		d = ev.Data
	case *dflow.DFlowShredInstruction:
		d = ev.Data
	case *systoken.TokenInstruction:
		d = ev.Data
	}
	if d == nil {
		return nil
	}
	if v := reflect.ValueOf(d); v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}
	return d
}

// TestShredAltExtendUntrustedCount: ExtendLookupTable allocated capacity for
// the u64 count taken from the instruction data; 2^62 made the allocation
// panic (DexParser returned nil) and 2^36 crashed the process with an
// unrecoverable out-of-memory error. No real transaction carries such a
// count, so the test uses the fixture's own ALT-free message and adds one
// Extend instruction with 2 addresses and count 2^62. shred-6.
func TestShredAltExtendUntrustedCount(t *testing.T) {
	tx := cloneTx(t, loadFixture(t, sigSystemTransfers))
	keys := &tx.Transaction.Message.AccountKeys
	*keys = append(*keys, adapter.AccountKey{Pubkey: constants.ALT_PROGRAM_ID})
	altIdx := len(*keys) - 1
	a1, a2 := (*keys)[1].Pubkey, (*keys)[2].Pubkey
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data, 2)
	binary.LittleEndian.PutUint64(data[4:], 1<<62)
	for _, k := range []string{a1, a2} {
		b, _ := base58.Decode(k)
		data = append(data, b...)
	}
	tx.Transaction.Message.Instructions = append(tx.Transaction.Message.Instructions, map[string]interface{}{
		"programIdIndex": altIdx, "accounts": []interface{}{1, 0, 0}, "data": base58.Encode(data),
	})

	res := dexparser.NewDexParser().ParseAll(tx, nil)
	if res == nil || !res.State {
		t.Fatalf("ParseAll = %+v", res)
	}
	if len(res.AltEvents) != 1 {
		t.Fatalf("ALT events = %d, want 1", len(res.AltEvents))
	}
	got := res.AltEvents[0].NewAddresses
	if len(got) != 2 || got[0] != a1 || got[1] != a2 {
		t.Errorf("NewAddresses = %v, want [%s %s] (the addresses the data holds)", got, a1, a2)
	}
}

// TestShredAltExtendReal decodes a real ExtendLookupTable with 10 addresses.
// shred-6.
func TestShredAltExtendReal(t *testing.T) {
	const sig = "3b4PGxj3ZxA1FgSzsfYAh6EZpEfqRn4pDUo45bTmf8uDRAdmuV6n6Kq3ZL1QYb4JM5PJKbeKzkV3vgJSrh7wE9hx"
	tx := loadFixture(t, sig)
	ix := findIx(t, tx, constants.ALT_PROGRAM_ID, []byte{2, 0, 0, 0})
	count := int(le64At(ix.data, 4))
	if count != (len(ix.data)-12)/32 {
		t.Fatalf("fixture: count %d for %d bytes", count, len(ix.data))
	}
	res := dexparser.NewDexParser().ParseAll(tx, nil)
	if len(res.AltEvents) != 1 {
		t.Fatalf("ALT events %d", len(res.AltEvents))
	}
	got := res.AltEvents[0].NewAddresses
	if len(got) != count {
		t.Fatalf("NewAddresses %d, want %d", len(got), count)
	}
	for i := range got {
		if got[i] != base58.Encode(ix.data[12+32*i:44+32*i]) {
			t.Errorf("address %d = %s", i, got[i])
		}
	}
}
