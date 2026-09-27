package tests

import (
	"compress/gzip"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/goccy/go-json"
	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Regression tests for input encodings: jsonParsed, Token-2022 parsed
// instructions, Node.js Buffer JSON and Go-typed instruction maps.

// TestCoreJSONParsedMatchesJSON: jsonParsed input panicked on every system
// transfer (lamports is a JSON number) and ParseAll returned nil. For every
// signature stored in both encodings the results must be equal. core-2,
// parity-5.
func TestCoreJSONParsedMatchesJSON(t *testing.T) {
	p := dexparser.NewDexParser()
	sigs := fixtureSignatures(t, "jsonParsed")
	if len(sigs) < 8 {
		t.Fatalf("only %d jsonParsed fixtures", len(sigs))
	}
	for _, sig := range sigs {
		parsed := loadParsedFixture(t, sig)
		raw := loadFixture(t, sig)
		rp, rj := p.ParseAll(parsed, nil), p.ParseAll(raw, nil)
		if rp == nil || !rp.State {
			t.Fatalf("%.12s jsonParsed: %+v", sig, rp)
		}
		if len(rp.Trades) == 0 && len(rp.MemeEvents) == 0 && len(rp.Liquidities) == 0 {
			t.Errorf("%.12s jsonParsed: no events", sig)
		}
		if !sameEvents(rp, rj) {
			t.Errorf("%.12s: jsonParsed and json results differ", sig)
		}
		// system transfers carry the exact lamports of the numeric field
		for _, x := range p.ParseTransfers(parsed, &types.ParseConfig{ParseType: types.ParseType{Transfer: true}}) {
			if x.ProgramId == constants.SYSTEM_PROGRAM_ID && (x.Info.TokenAmount.Amount == "" || x.Info.TokenAmount.Amount == "0") {
				t.Errorf("%.12s: system transfer without amount", sig)
			}
		}
	}
}

// sameEvents compares trades, aggregate, liquidities and meme events.
func sameEvents(a, b *types.ParseResult) bool {
	return reflect.DeepEqual(a.Trades, b.Trades) && reflect.DeepEqual(a.AggregateTrade, b.AggregateTrade) &&
		reflect.DeepEqual(a.Liquidities, b.Liquidities) && reflect.DeepEqual(a.MemeEvents, b.MemeEvents)
}

// TestCoreJSONParsedUpstreamOutputs: upstream TS 2.6.7 on the jsonParsed
// encoding returns a Pumpswap SELL 5637949066881 -> 168803601 for 36Q2 and two
// LaunchLab trades plus two meme events for 61AN; Go returned no trade and a
// nil result. parity-5.
func TestCoreJSONParsedUpstreamOutputs(t *testing.T) {
	p := dexparser.NewDexParser()
	cfg := &types.ParseConfig{ParseType: types.ParseType{Trade: true, MemeEvent: true}, TryUnknownDEX: true}

	r := p.ParseAll(loadParsedFixture(t, "36Q2tYo1CPa42GF51bzA493nYQCG8fPbpQJEzRhZQURYuBcRKpj97HWBCLCzDwgQJ8tnVrW9fDZKWaPBdADEsxTE"), cfg)
	if len(r.Trades) != 1 || r.Trades[0].AMM != "Pumpswap" || r.Trades[0].Type != types.TradeTypeSell ||
		r.Trades[0].InputToken.AmountRaw != "5637949066881" || r.Trades[0].OutputToken.AmountRaw != "168803601" {
		t.Errorf("36Q2 jsonParsed trades = %v", r.Trades)
	}

	r = p.ParseAll(loadParsedFixture(t, "61AN23VGPknSqskF6CvtZqrD4LtL2CNGYKeyFc5nLVcfaUaHV5LsQe6HTnRFM6pNX8qf7fkqZ5tEZfnNEF73H8MX"), cfg)
	if len(r.Trades) != 2 || len(r.MemeEvents) != 2 {
		t.Errorf("61AN jsonParsed: %d trades, %d meme events, want 2 and 2", len(r.Trades), len(r.MemeEvents))
	}
	for _, tr := range r.Trades {
		if tr.AMM != "RaydiumLaunchpad" {
			t.Errorf("61AN trade AMM %s", tr.AMM)
		}
	}
}

// TestCoreJSONParsedToken2022: jsonParsed labels Token-2022 instructions as
// program "spl-token"; the parsed-path checks also required the Token program
// ID, so Token-2022 transfers, mints and burns were dropped. core-13.
func TestCoreJSONParsedToken2022(t *testing.T) {
	const sig = "5AbBMjFDWHkUaeVuvuYG6UaSG2aCPcCxsGkUT1EBEsWsTyCMzKTBnVh6unah8ztVkRQ6EeGdmSvUdgCsrM58sW8Z"
	parsed := newParseContext(loadParsedFixture(t, sig), nil)
	raw := newParseContext(loadFixture(t, sig), nil)
	count := func(actions map[string][]types.TransferData) int {
		n := 0
		for _, x := range utils.SortedTransfers(actions) {
			if x.ProgramId == constants.TOKEN_2022_PROGRAM_ID {
				n++
			}
		}
		return n
	}
	nRaw := count(raw.TransferActions)
	if nRaw == 0 {
		t.Fatal("fixture has no Token-2022 transfers")
	}
	if got := count(parsed.TransferActions); got != nRaw {
		t.Errorf("jsonParsed Token-2022 transfers = %d, json = %d", got, nRaw)
	}
	pa := utils.SortedTransfers(parsed.TransferActions)
	ra := utils.SortedTransfers(raw.TransferActions)
	if len(pa) != len(ra) {
		t.Fatalf("transfer count %d vs %d", len(pa), len(ra))
	}
	for i := range pa {
		if pa[i].Idx != ra[i].Idx || pa[i].Type != ra[i].Type || pa[i].Info.Mint != ra[i].Info.Mint || pa[i].Info.TokenAmount.Amount != ra[i].Info.TokenAmount.Amount {
			t.Errorf("transfer %d: jsonParsed %s %s %s %s, json %s %s %s %s", i, pa[i].Idx, pa[i].Type, pa[i].Info.Mint, pa[i].Info.TokenAmount.Amount, ra[i].Idx, ra[i].Type, ra[i].Info.Mint, ra[i].Info.TokenAmount.Amount)
		}
	}
}

// TestCoreJSONParsedTransferCheckedWithFee: a jsonParsed Token-2022
// transferCheckedWithFee is extracted with its gross amount. core-13.
//
// Synthetic: built from the real 5AbBMjFD (jsonParsed) by relabelling its
// Token-2022 transferChecked at 7-2 as transferCheckedWithFee with a feeAmount.
func TestCoreJSONParsedTransferCheckedWithFee(t *testing.T) {
	const sig = "5AbBMjFDWHkUaeVuvuYG6UaSG2aCPcCxsGkUT1EBEsWsTyCMzKTBnVh6unah8ztVkRQ6EeGdmSvUdgCsrM58sW8Z"
	tx := cloneTx(t, loadParsedFixture(t, sig))
	var set *adapter.InnerInstructionSet
	for i := range tx.Meta.InnerInstructions {
		if tx.Meta.InnerInstructions[i].Index == 7 {
			set = &tx.Meta.InnerInstructions[i]
		}
	}
	ix := set.Instructions[2].(map[string]interface{})
	if ix["programId"] != constants.TOKEN_2022_PROGRAM_ID {
		t.Fatalf("7-2 is not a Token-2022 instruction: %v", ix["programId"])
	}
	parsed := ix["parsed"].(map[string]interface{})
	parsed["type"] = "transferCheckedWithFee"
	parsed["info"].(map[string]interface{})["feeAmount"] = map[string]interface{}{"amount": "16", "decimals": 6, "uiAmountString": "0.000016"}

	ctx := newParseContext(tx, nil)
	for _, x := range utils.SortedTransfers(ctx.TransferActions) {
		if x.Idx == "7-2" {
			if x.Type != "transferChecked" || x.ProgramId != constants.TOKEN_2022_PROGRAM_ID || x.Info.TokenAmount.Amount != "4116" ||
				x.Info.Mint != "RBLXDGRD64AtRamHMFVcjqne3Ar7NLWtFtYNtsrf1cE" {
				t.Errorf("transfer 7-2 = %+v", x)
			}
			return
		}
	}
	t.Error("transferCheckedWithFee at 7-2 not extracted")
}

// TestCoreBufferEncodingUpstreamFixture: transactions serialised by Node.js
// tooling ({"type":"Buffer","data":[...]} for signatures, keys, account index
// lists and data) failed to unmarshal. parity-21.
//
// testdata/upstream/shred-tx-1.json.gz is txs[1] of upstream
// src/__tests__/shred-tx.test.case.ts (a real pre-execution transaction).
func TestCoreBufferEncodingUpstreamFixture(t *testing.T) {
	f, err := os.Open("../testdata/upstream/shred-tx-1.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(raw, &tx); err != nil {
		t.Fatalf("unmarshal Buffer JSON: %v", err)
	}

	// independent decoding of the same JSON
	var doc struct {
		Slot        string `json:"slot"`
		Transaction struct {
			Signatures []struct{ Data []byte } `json:"signatures"`
			Message    struct {
				AccountKeys  []struct{ Data []byte } `json:"accountKeys"`
				Instructions []struct {
					ProgramIdIndex int `json:"programIdIndex"`
					Accounts       struct {
						Data []byte `json:"data"`
					} `json:"accounts"`
					Data struct {
						Data []byte `json:"data"`
					} `json:"data"`
				} `json:"instructions"`
			} `json:"message"`
		} `json:"transaction"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if tx.Slot != 345203603 {
		t.Errorf("slot = %d", tx.Slot)
	}
	a := adapter.NewTransactionAdapter(&tx, nil)
	if a.Signature() != base58.Encode(doc.Transaction.Signatures[0].Data) || a.Signature() != "3GsVModrL66q7ScCRRSTFYr61GR6vpoCSPTUzKRMog9eBYHyNTgTi4uHNmM4bG43vQVVK8XZm3bHKsczfcLU2uUT" {
		t.Errorf("signature = %s", a.Signature())
	}
	if len(a.AccountKeys) != len(doc.Transaction.Message.AccountKeys) {
		t.Fatalf("account keys %d, want %d", len(a.AccountKeys), len(doc.Transaction.Message.AccountKeys))
	}
	for i, k := range doc.Transaction.Message.AccountKeys {
		if a.AccountKeys[i] != base58.Encode(k.Data) {
			t.Errorf("key %d = %s", i, a.AccountKeys[i])
		}
	}
	for i, ix := range doc.Transaction.Message.Instructions {
		ui := a.GetInstruction(a.InstructionAt(i))
		if ui.ProgramId != a.AccountKeys[ix.ProgramIdIndex] || !reflect.DeepEqual(ui.Data, ix.Data.Data) || len(ui.Accounts) != len(ix.Accounts.Data) {
			t.Errorf("instruction %d: program %s data %v accounts %v", i, ui.ProgramId, ui.Data, ui.Accounts)
			continue
		}
		for j, idx := range ix.Accounts.Data {
			if ui.Accounts[j] != a.AccountKeys[idx] {
				t.Errorf("instruction %d account %d = %s", i, j, ui.Accounts[j])
			}
		}
	}

	// instruction 4 is a System transfer of 100000 lamports
	transfers := dexparser.NewDexParser().ParseTransfers(&tx, nil)
	found := false
	for _, x := range transfers {
		if x.ProgramId == constants.SYSTEM_PROGRAM_ID && x.Info.TokenAmount.Amount == "100000" && x.Info.Source == a.AccountKeys[0] {
			found = true
		}
	}
	if !found {
		t.Errorf("System transfer of 100000 lamports not found in %v", transfers)
	}
}

// TestCoreBufferEncodingRealFixture converts a real "json" fixture into the
// Buffer form (signatures, keys, loaded addresses, instruction data and
// account lists) and checks that parsing gives the same result. parity-21.
func TestCoreBufferEncodingRealFixture(t *testing.T) {
	p := dexparser.NewDexParser()
	raw, err := readFixture(sigTwoLookups, "json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	buffer := func(b []byte) map[string]interface{} {
		ints := make([]interface{}, len(b))
		for i, v := range b {
			ints[i] = int(v)
		}
		return map[string]interface{}{"type": "Buffer", "data": ints}
	}
	key := func(v interface{}) interface{} {
		b, err := base58.Decode(v.(string))
		if err != nil {
			t.Fatal(err)
		}
		return buffer(b)
	}
	convertIx := func(v interface{}) {
		ix := v.(map[string]interface{})
		data, _ := base58.Decode(ix["data"].(string))
		ix["data"] = buffer(data)
		var accs []byte
		for _, a := range ix["accounts"].([]interface{}) {
			accs = append(accs, byte(a.(float64)))
		}
		ix["accounts"] = buffer(accs)
	}
	txd := doc["transaction"].(map[string]interface{})
	sigs := txd["signatures"].([]interface{})
	for i := range sigs {
		sigs[i] = key(sigs[i])
	}
	msg := txd["message"].(map[string]interface{})
	keys := msg["accountKeys"].([]interface{})
	for i := range keys {
		keys[i] = key(keys[i])
	}
	for _, ix := range msg["instructions"].([]interface{}) {
		convertIx(ix)
	}
	meta := doc["meta"].(map[string]interface{})
	for _, set := range meta["innerInstructions"].([]interface{}) {
		for _, ix := range set.(map[string]interface{})["instructions"].([]interface{}) {
			convertIx(ix)
		}
	}
	loaded := meta["loadedAddresses"].(map[string]interface{})
	for _, side := range []string{"writable", "readonly"} {
		list := loaded[side].([]interface{})
		for i := range list {
			list[i] = key(list[i])
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(out, &tx); err != nil {
		t.Fatalf("unmarshal Buffer form: %v", err)
	}
	want := p.ParseAll(loadFixture(t, sigTwoLookups), nil)
	got := p.ParseAll(&tx, nil)
	if got.Signature != sigTwoLookups || len(got.Trades) == 0 || !sameEvents(got, want) {
		t.Errorf("Buffer form: signature %s, trades %v; want %v", got.Signature, got.Trades, want.Trades)
	}
}

// TestCoreGoTypedInstructionMaps: instruction maps built in Go (what
// ConvertYellowstoneTransaction emits) with accounts as []int and data as
// []byte were decoded without accounts. core-4 (adapter part).
func TestCoreGoTypedInstructionMaps(t *testing.T) {
	p := dexparser.NewDexParser()
	orig := loadFixture(t, sigJupPumpswap)
	tx := cloneTx(t, orig)
	convert := func(v interface{}) interface{} {
		ix := v.(map[string]interface{})
		data, _ := base58.Decode(ix["data"].(string))
		var accs []int
		for _, a := range ix["accounts"].([]interface{}) {
			accs = append(accs, jsonInt(a))
		}
		pid := jsonInt(ix["programIdIndex"])
		return map[string]interface{}{"programIdIndex": pid, "accounts": accs, "data": data}
	}
	for i, ix := range tx.Transaction.Message.Instructions {
		tx.Transaction.Message.Instructions[i] = convert(ix)
	}
	for si := range tx.Meta.InnerInstructions {
		for i, ix := range tx.Meta.InnerInstructions[si].Instructions {
			tx.Meta.InnerInstructions[si].Instructions[i] = convert(ix)
		}
	}
	a := adapter.NewTransactionAdapter(tx, nil)
	oa := adapter.NewTransactionAdapter(orig, nil)
	for i := range tx.Transaction.Message.Instructions {
		got, want := a.GetInstruction(a.InstructionAt(i)), oa.GetInstruction(oa.InstructionAt(i))
		if !reflect.DeepEqual(got.Accounts, want.Accounts) || !reflect.DeepEqual(got.Data, want.Data) || got.ProgramId != want.ProgramId {
			t.Errorf("instruction %d decoded differently", i)
		}
	}
	if got, want := p.ParseAll(tx, nil), p.ParseAll(orig, nil); len(got.Trades) == 0 || !sameEvents(got, want) {
		t.Errorf("Go-typed maps: trades %v, want %v", got.Trades, want.Trades)
	}
}
