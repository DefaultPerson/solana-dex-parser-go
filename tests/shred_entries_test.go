package tests

import (
	"encoding/binary"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// ShredStream entry decoding. The entries are built here from real
// transactions with the wire formats of solana-sdk (VersionedTransaction:
// short_vec signatures + legacy / 0x80-prefixed v0 message; v1: 0x81, legacy
// header, config mask, blockhash, counts, addresses, config values,
// instruction headers and payloads, then the signatures) and bincode
// Vec<Entry{num_hashes u64, hash, transactions}>. streamer.md item 10.

func shortVec(n int) []byte {
	var out []byte
	for {
		b := byte(n & 0x7f)
		n >>= 7
		if n == 0 {
			return append(out, b)
		}
		out = append(out, b|0x80)
	}
}

func le64b(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

// wireTx serializes a "json" fixture transaction
func wireTx(t *testing.T, tx *adapter.SolanaTransaction) []byte {
	t.Helper()
	dec := func(s string) []byte {
		if s == "" {
			return nil
		}
		b, err := base58.Decode(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	msg := tx.Transaction.Message
	header := []byte{byte(msg.Header.NumRequiredSignatures), byte(msg.Header.NumReadonlySignedAccounts), byte(msg.Header.NumReadonlyUnsignedAccounts)}
	blockhash := make([]byte, 32)
	var sigs []byte
	for _, s := range tx.Transaction.Signatures {
		sigs = append(sigs, dec(s)...)
	}
	type ix struct {
		program  byte
		accounts []byte
		data     []byte
	}
	var ixs []ix
	for _, v := range msg.Instructions {
		m := v.(map[string]interface{})
		var accounts []byte
		for _, a := range m["accounts"].([]interface{}) {
			accounts = append(accounts, byte(jsonInt(a)))
		}
		ixs = append(ixs, ix{byte(jsonInt(m["programIdIndex"])), accounts, dec(m["data"].(string))})
	}

	if jsonInt(tx.Version) == 1 {
		out := []byte{0x81}
		out = append(out, header...)
		var mask uint32
		var values []byte
		cfg := msg.TransactionConfig
		if cfg != nil && cfg.PriorityFee != nil {
			mask |= 0b11
			values = append(values, le64b(*cfg.PriorityFee)...)
		}
		for i, v := range []*uint64{cfgField(cfg, 0), cfgField(cfg, 1), cfgField(cfg, 2)} {
			if v != nil {
				mask |= 0b100 << i
				values = binary.LittleEndian.AppendUint32(values, uint32(*v))
			}
		}
		out = binary.LittleEndian.AppendUint32(out, mask)
		out = append(out, blockhash...)
		out = append(out, byte(len(ixs)), byte(len(msg.AccountKeys)))
		for _, k := range msg.AccountKeys {
			out = append(out, dec(k.Pubkey)...)
		}
		out = append(out, values...)
		for _, x := range ixs {
			out = append(out, x.program, byte(len(x.accounts)))
			out = binary.LittleEndian.AppendUint16(out, uint16(len(x.data)))
		}
		for _, x := range ixs {
			out = append(out, x.accounts...)
			out = append(out, x.data...)
		}
		return append(out, sigs...)
	}

	out := append(shortVec(len(tx.Transaction.Signatures)), sigs...)
	if tx.Version != "legacy" {
		out = append(out, 0x80)
	}
	out = append(out, header...)
	out = append(out, shortVec(len(msg.AccountKeys))...)
	for _, k := range msg.AccountKeys {
		out = append(out, dec(k.Pubkey)...)
	}
	out = append(out, blockhash...)
	out = append(out, shortVec(len(ixs))...)
	for _, x := range ixs {
		out = append(out, x.program)
		out = append(out, shortVec(len(x.accounts))...)
		out = append(out, x.accounts...)
		out = append(out, shortVec(len(x.data))...)
		out = append(out, x.data...)
	}
	if tx.Version != "legacy" {
		out = append(out, shortVec(len(msg.AddressTableLookups))...)
		for _, l := range msg.AddressTableLookups {
			out = append(out, dec(l.AccountKey)...)
			out = append(out, shortVec(len(l.WritableIndexes))...)
			for _, i := range l.WritableIndexes {
				out = append(out, byte(i))
			}
			out = append(out, shortVec(len(l.ReadonlyIndexes))...)
			for _, i := range l.ReadonlyIndexes {
				out = append(out, byte(i))
			}
		}
	}
	return out
}

func cfgField(cfg *adapter.TransactionConfig, i int) *uint64 {
	if cfg == nil {
		return nil
	}
	return []*uint64{cfg.ComputeUnitLimit, cfg.LoadedAccountsDataSizeLimit, cfg.HeapSize}[i]
}

// entries serializes Vec<Entry> with the given transactions per entry
func entries(t *testing.T, groups ...[]*adapter.SolanaTransaction) []byte {
	out := le64b(uint64(len(groups)))
	for _, g := range groups {
		out = append(out, le64b(1)...)              // num_hashes
		out = append(out, make([]byte, 32)...)      // hash
		out = append(out, le64b(uint64(len(g)))...) // transactions
		for _, tx := range g {
			out = append(out, wireTx(t, tx)...)
		}
	}
	return out
}

// TestShredEntriesDecode decodes legacy, v0 and v1 transactions from
// ShredStream entries; ShredParser results must equal those of the
// meta-stripped RPC transactions.
func TestShredEntriesDecode(t *testing.T) {
	legacy := loadFixture(t, sigPumpBuyLegacy)
	v0 := loadFixture(t, sigJupRouteV1)
	v1 := loadFixture(t, sigLcpInit2022)
	if legacy.Version != "legacy" || jsonInt(v0.Version) != 0 || jsonInt(v1.Version) != 1 {
		t.Fatalf("fixture versions %v %v %v", legacy.Version, v0.Version, v1.Version)
	}
	originals := []*adapter.SolanaTransaction{legacy, v0, v1}
	data := entries(t, []*adapter.SolanaTransaction{legacy, v0}, []*adapter.SolanaTransaction{}, []*adapter.SolanaTransaction{v1})

	txs, err := dexparser.DecodeShredEntries(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 3 {
		t.Fatalf("decoded %d transactions, want 3", len(txs))
	}
	tables := lookupTables(t, v0)
	for i, orig := range originals {
		got := txs[i]
		if got.Meta != nil || got.Transaction.Signatures[0] != orig.Transaction.Signatures[0] {
			t.Errorf("tx %d: signature %s meta %v", i, got.Transaction.Signatures[0], got.Meta)
		}
		if jsonOf(t, rawAccountKeys(got)) != jsonOf(t, rawAccountKeys(preExec(t, orig))) {
			t.Errorf("tx %d: account keys differ", i)
		}
		cfg := &types.ParseConfig{AddressLookupTables: tables}
		want := parseShred(t, preExec(t, orig), cfg)
		res := parseShred(t, got, cfg)
		want.Slot, want.Timestamp = 0, 0
		for j := range want.ParsedInstructions {
			clearContext(&want.ParsedInstructions[j])
		}
		for j := range res.ParsedInstructions {
			clearContext(&res.ParsedInstructions[j])
		}
		if jsonOf(t, res.ParsedInstructions) != jsonOf(t, want.ParsedInstructions) || len(want.ParsedInstructions) == 0 {
			t.Errorf("tx %d: shred results differ:\n got %s\nwant %s", i, jsonOf(t, res.ParsedInstructions), jsonOf(t, want.ParsedInstructions))
		}
	}
	if cfg, want := txs[2].Transaction.Message.TransactionConfig, v1.Transaction.Message.TransactionConfig; jsonOf(t, cfg) != jsonOf(t, want) {
		t.Errorf("v1 config %s, want %s", jsonOf(t, cfg), jsonOf(t, want))
	}
}

// clearContext removes the slot/time context a ShredStream entry lacks
func clearContext(ins *types.ParsedShredInstruction) {
	if ins.Trade != nil {
		ins.Trade.Slot, ins.Trade.Timestamp = 0, 0
	}
	if ins.MemeEvent != nil {
		ins.MemeEvent.Slot, ins.MemeEvent.Timestamp = 0, 0
	}
	if ins.Liquidity != nil {
		ins.Liquidity.Slot, ins.Liquidity.Timestamp = 0, 0
	}
	if ins.Transfer != nil {
		ins.Transfer.Timestamp = 0
	}
}

// TestShredEntriesMalformed: counts come from untrusted data; they must be
// checked against the remaining bytes, and malformed data must return an
// error, not panic or allocate.
func TestShredEntriesMalformed(t *testing.T) {
	good := entries(t, []*adapter.SolanaTransaction{loadFixture(t, sigPumpBuyLegacy), loadFixture(t, sigJupRouteV1)})

	huge := append([]byte{}, good...)
	binary.LittleEndian.PutUint64(huge, 1<<62)
	if txs, err := dexparser.DecodeShredEntries(huge); err == nil || len(txs) != 0 {
		t.Errorf("entry count 2^62: %d txs, err %v", len(txs), err)
	}

	hugeTxs := append([]byte{}, good...)
	binary.LittleEndian.PutUint64(hugeTxs[8+8+32:], 1<<62)
	if _, err := dexparser.DecodeShredEntries(hugeTxs); err == nil {
		t.Error("transaction count 2^62 accepted")
	}

	// Truncated inside the second transaction: the first one is returned
	first := len(entries(t, []*adapter.SolanaTransaction{loadFixture(t, sigPumpBuyLegacy)}))
	truncated := good[:first+100]
	txs, err := dexparser.DecodeShredEntries(truncated)
	if err == nil || len(txs) != 1 || txs[0].Transaction.Signatures[0] != sigPumpBuyLegacy {
		t.Errorf("truncated: %d txs, err %v", len(txs), err)
	}

	// Unknown transaction version byte
	bad := append([]byte{}, good...)
	bad[8+8+32+8] = 0x82
	if _, err := dexparser.DecodeShredEntries(bad); err == nil {
		t.Error("version byte 0x82 accepted")
	}

	for n := 0; n < len(good); n += 7 {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic on %d-byte prefix: %v", n, r)
				}
			}()
			dexparser.DecodeShredEntries(good[:n])
		}()
	}
}
