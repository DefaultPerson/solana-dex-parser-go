package tests

import (
	"os"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// benchmarkTxs holds pre-fetched transactions for benchmarks
var benchmarkTxs []*adapter.SolanaTransaction

func setupBenchmarkTxs(b *testing.B) []*adapter.SolanaTransaction {
	apiKey := os.Getenv("HELIUS_API_KEY")
	if apiKey == "" {
		b.Skip("HELIUS_API_KEY not set, skipping benchmarks")
	}

	if benchmarkTxs != nil {
		return benchmarkTxs
	}

	signatures := []string{
		"4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz", // Pumpfun
		"v8s37Srj6QPMtRC1HfJcrSenCHvYebHiGkHVuFFiQ6UviqHnoVx4U77M3TZhQQXewXadHYh5t35LkesJi3ztPZZ",  // Pumpfun sell
	}

	txs := make([]*adapter.SolanaTransaction, 0, len(signatures))
	for _, sig := range signatures {
		tx, err := fetchTransaction(sig)
		if err != nil {
			b.Fatalf("Failed to fetch transaction %s: %v", sig, err)
		}
		txs = append(txs, tx)
	}

	benchmarkTxs = txs
	return txs
}

// BenchmarkParseSingle benchmarks single transaction parsing
func BenchmarkParseSingle(b *testing.B) {
	txs := setupBenchmarkTxs(b)
	parser := dexparser.NewDexParser()
	config := types.DefaultParseConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tx := range txs {
			_ = parser.ParseAll(tx, &config)
		}
	}
}

// BenchmarkParseBatchSequential benchmarks batch parsing with 1 worker (sequential)
func BenchmarkParseBatchSequential(b *testing.B) {
	txs := setupBenchmarkTxs(b)
	parser := dexparser.NewDexParser()
	config := types.DefaultParseConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.ParseBatch(txs, &config, 1)
	}
}

// BenchmarkParseBatchConcurrent2 benchmarks batch parsing with 2 workers
func BenchmarkParseBatchConcurrent2(b *testing.B) {
	txs := setupBenchmarkTxs(b)
	parser := dexparser.NewDexParser()
	config := types.DefaultParseConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.ParseBatch(txs, &config, 2)
	}
}

// BenchmarkParseBatchConcurrent4 benchmarks batch parsing with 4 workers
func BenchmarkParseBatchConcurrent4(b *testing.B) {
	txs := setupBenchmarkTxs(b)
	parser := dexparser.NewDexParser()
	config := types.DefaultParseConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.ParseBatch(txs, &config, 4)
	}
}

// BenchmarkParseTradesOnly benchmarks parsing with only trades enabled
func BenchmarkParseTradesOnly(b *testing.B) {
	txs := setupBenchmarkTxs(b)
	parser := dexparser.NewDexParser()
	config := types.ParseConfig{
		ParseType:     types.ParseTradesOnly(),
		TryUnknownDEX: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tx := range txs {
			_ = parser.ParseAll(tx, &config)
		}
	}
}

// BenchmarkParseLiquidityOnly benchmarks parsing with only liquidity enabled
func BenchmarkParseLiquidityOnly(b *testing.B) {
	txs := setupBenchmarkTxs(b)
	parser := dexparser.NewDexParser()
	config := types.ParseConfig{
		ParseType:     types.ParseLiquidityOnly(),
		TryUnknownDEX: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tx := range txs {
			_ = parser.ParseAll(tx, &config)
		}
	}
}

// BenchmarkParseAllTypes benchmarks parsing with all types enabled
func BenchmarkParseAllTypes(b *testing.B) {
	txs := setupBenchmarkTxs(b)
	parser := dexparser.NewDexParser()
	config := types.ParseConfig{
		ParseType:     types.ParseAll(),
		TryUnknownDEX: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tx := range txs {
			_ = parser.ParseAll(tx, &config)
		}
	}
}
