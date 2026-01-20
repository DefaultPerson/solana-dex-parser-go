package tests

import (
	"os"
	"testing"

	"github.com/goccy/go-json"
	dexparser "github.com/solana-dex-parser-go"
	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/types"
	"github.com/solana-dex-parser-go/utils"
)

// Sample transaction JSON for benchmarks (Pumpfun BUY)
var benchTxJSON = []byte(`{
	"slot": 123456789,
	"blockTime": 1704067200,
	"transaction": {
		"signatures": ["4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz"],
		"message": {
			"accountKeys": [
				"9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin",
				"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
				"11111111111111111111111111111111"
			],
			"instructions": []
		}
	},
	"meta": {
		"err": null,
		"fee": 5000,
		"innerInstructions": [],
		"postTokenBalances": [],
		"preTokenBalances": []
	}
}`)

func BenchmarkJSONUnmarshalStdlib(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var tx adapter.SolanaTransaction
		if err := json.Unmarshal(benchTxJSON, &tx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBinaryReaderNew(b *testing.B) {
	data := make([]byte, 128)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := utils.NewBinaryReader(data)
		r.ReadU64()
		r.ReadPubkey()
		r.ReadU64()
	}
}

func BenchmarkBinaryReaderPooled(b *testing.B) {
	data := make([]byte, 128)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := utils.GetBinaryReader(data)
		r.ReadU64()
		r.ReadPubkey()
		r.ReadU64()
		r.Release()
	}
}

func BenchmarkNewDexParser(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = dexparser.NewDexParser()
	}
}

func BenchmarkParseTransactionMinimal(b *testing.B) {
	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(benchTxJSON, &tx); err != nil {
		b.Fatal(err)
	}

	parser := dexparser.NewDexParser()
	cfg := types.DefaultParseConfig()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parser.ParseAll(&tx, &cfg)
	}
}

// BenchmarkParseRealTransaction benchmarks with a real transaction if available
func BenchmarkParseRealTransaction(b *testing.B) {
	// Try to load a cached transaction
	data, err := os.ReadFile("testdata/pumpfun_buy.json")
	if err != nil {
		b.Skip("testdata/pumpfun_buy.json not found, skipping real tx benchmark")
		return
	}

	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(data, &tx); err != nil {
		b.Fatal(err)
	}

	parser := dexparser.NewDexParser()
	cfg := types.DefaultParseConfig()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parser.ParseAll(&tx, &cfg)
	}
}
