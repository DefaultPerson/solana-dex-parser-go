package tests

import (
	"reflect"
	"sync/atomic"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

func TestParseBatch(t *testing.T) {
	parser := dexparser.NewDexParser()

	// Load multiple transactions from the fixtures
	signatures := []string{
		"4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz", // Pumpfun
		"v8s37Srj6QPMtRC1HfJcrSenCHvYebHiGkHVuFFiQ6UviqHnoVx4U77M3TZhQQXewXadHYh5t35LkesJi3ztPZZ",  // Pumpfun buy
	}

	var txs []*adapter.SolanaTransaction
	for _, sig := range signatures {
		txs = append(txs, loadFixture(t, sig))
	}

	config := types.DefaultParseConfig()

	// Test sequential processing
	results := parser.ParseBatch(txs, &config, 1)
	if len(results) != len(txs) {
		t.Errorf("Expected %d results, got %d", len(txs), len(results))
	}

	for i, result := range results {
		if result == nil {
			t.Errorf("Result %d is nil", i)
			continue
		}
		if result.Signature != signatures[i] || len(result.Trades) != 1 || len(result.MemeEvents) == 0 {
			t.Errorf("Result %d: signature=%s, trades=%d, memeEvents=%d", i, result.Signature, len(result.Trades), len(result.MemeEvents))
		}
		if single := parser.ParseAll(txs[i], &config); !reflect.DeepEqual(single, result) {
			t.Errorf("Result %d differs from ParseAll", i)
		}
	}

	// Test concurrent processing
	concurrent := parser.ParseBatch(txs, &config, 4)
	if len(concurrent) != len(txs) {
		t.Fatalf("Expected %d results with concurrent processing, got %d", len(txs), len(concurrent))
	}
	if !reflect.DeepEqual(concurrent, results) {
		t.Error("concurrent results differ from sequential results")
	}
}

func TestParseBatchWithCallback(t *testing.T) {
	parser := dexparser.NewDexParser()

	// Load multiple transactions from the fixtures
	signatures := []string{
		"4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz",
		"v8s37Srj6QPMtRC1HfJcrSenCHvYebHiGkHVuFFiQ6UviqHnoVx4U77M3TZhQQXewXadHYh5t35LkesJi3ztPZZ",
	}

	var txs []*adapter.SolanaTransaction
	for _, sig := range signatures {
		txs = append(txs, loadFixture(t, sig))
	}

	config := types.DefaultParseConfig()
	var callbackCount int32

	callback := func(index int, tx *adapter.SolanaTransaction, result *types.ParseResult, err error) bool {
		atomic.AddInt32(&callbackCount, 1)
		if result == nil || err != nil || result.Signature != signatures[index] {
			t.Errorf("Callback index=%d: result=%v err=%v", index, result, err)
		}
		return true // continue processing
	}

	results := parser.ParseBatchWithCallback(txs, &config, 2, callback)

	if len(results) != len(txs) {
		t.Errorf("Expected %d results, got %d", len(txs), len(results))
	}

	if int(callbackCount) != len(txs) {
		t.Errorf("Expected %d callbacks, got %d", len(txs), callbackCount)
	}
}

func TestParseBatchEarlyTermination(t *testing.T) {
	parser := dexparser.NewDexParser()

	// Load multiple transactions from the fixtures
	signatures := []string{
		"4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz",
		"v8s37Srj6QPMtRC1HfJcrSenCHvYebHiGkHVuFFiQ6UviqHnoVx4U77M3TZhQQXewXadHYh5t35LkesJi3ztPZZ",
		"2kAW5GAhPZjM3NoSrhJVHdEpwjmq9neWtckWnjopCfsmCGB27e3v2ZyMM79FdsL4VWGEtYSFi1sF1Zhs7bqdoaVT",
	}

	var txs []*adapter.SolanaTransaction
	for _, sig := range signatures {
		txs = append(txs, loadFixture(t, sig))
	}

	config := types.DefaultParseConfig()
	var callbackCount int32

	// Callback that stops after first result
	callback := func(index int, tx *adapter.SolanaTransaction, result *types.ParseResult, err error) bool {
		atomic.AddInt32(&callbackCount, 1)
		return false // stop after first
	}

	// Sequential processing for predictable early termination
	_ = parser.ParseBatchWithCallback(txs, &config, 1, callback)

	if int(callbackCount) != 1 {
		t.Errorf("Expected 1 callback due to early termination, got %d", callbackCount)
	}
}

func TestParseBatchEmpty(t *testing.T) {
	parser := dexparser.NewDexParser()
	config := types.DefaultParseConfig()

	results := parser.ParseBatch([]*adapter.SolanaTransaction{}, &config, 4)
	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty batch, got %d", len(results))
	}
}
