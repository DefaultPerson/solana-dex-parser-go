package tests

import (
	"reflect"
	"testing"

	"github.com/goccy/go-json"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
)

// parity-26 / upstream issue #93: Helius's getTransactionsForAddress RPC
// method (transactionDetails "full") returns {data: [...], paginationToken}
// where each item is a getTransaction result (slot, blockTime, transaction,
// meta, version) plus transactionIndex. Such items are SolanaTransaction
// input as they are: built here from real fixtures ("json" and "jsonParsed"
// encodings), a page parses exactly like the fixtures themselves. (The
// legacy Enhanced Transactions REST endpoint getTransactionsByAddress returns
// Helius's own parsed format without the raw transaction, which cannot be
// parsed.)
func TestIntegHeliusTransactionsForAddressPage(t *testing.T) {
	sigs := []string{
		"4C2p65nuttUqBv5VhLNXHq6cNEaexT5i4YGbGSH2hodRGLBQyJBzRCkr2VXPKPNof8Lut8WSm51tkxVv3mvUkpuZ",
		"41NE3vcJ2aXixCvWvPV5fb6d1PxDfHT7GNGS5nFJqKMnJcy5g6iGkqkxt8nUWYaK9HLS8fuz6s5d4MhsmTC6AjEL",
	}
	var items []json.RawMessage
	var want []*adapter.SolanaTransaction
	add := func(raw []byte, tx *adapter.SolanaTransaction, index int) {
		var item map[string]interface{}
		if err := json.Unmarshal(raw, &item); err != nil {
			t.Fatal(err)
		}
		item["transactionIndex"] = index
		b, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, b)
		want = append(want, tx)
	}
	for i, sig := range sigs {
		raw, err := readFixture(sig, "json")
		if err != nil {
			t.Fatal(err)
		}
		add(raw, loadFixture(t, sig), i)
	}
	const parsedSig = "NeF1UiWXKUbuswNNw14gJ2uup7yrV6KyXijQ7dsjLYanj9dnsBSeqGPDVZ3wP3NfhXQ84rJncRo5XwbrbSdWVWW"
	raw, err := readFixture(parsedSig, "jsonParsed")
	if err != nil {
		t.Fatal(err)
	}
	add(raw, loadParsedFixture(t, parsedSig), 7)

	page, err := json.Marshal(map[string]interface{}{"data": items, "paginationToken": "429335117:1"})
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data            []*adapter.SolanaTransaction `json:"data"`
		PaginationToken *string                      `json:"paginationToken"`
	}
	if err := json.Unmarshal(page, &result); err != nil {
		t.Fatalf("unmarshal page: %v", err)
	}
	if len(result.Data) != len(want) {
		t.Fatalf("%d items, want %d", len(result.Data), len(want))
	}
	p := dexparser.NewDexParser()
	for i, tx := range result.Data {
		got, exp := p.ParseAll(tx, nil), p.ParseAll(want[i], nil)
		if !got.State || len(got.Trades) == 0 {
			t.Errorf("item %d: State %v, %d trades", i, got.State, len(got.Trades))
		}
		if !reflect.DeepEqual(got, exp) {
			t.Errorf("item %d (%s) parses differently from the fixture", i, got.Signature)
		}
	}
}
