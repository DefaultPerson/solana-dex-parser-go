# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Fixed
- **Critical bug**: Float precision issue in transfer deduplication (`transaction_utils.go:309`)
  - Changed from `fmt.Sprintf("%f", amount)` to `AmountRaw` string to prevent false duplicates

### Performance
- **JSON parsing**: Replaced `encoding/json` with `goccy/go-json` (~2x faster unmarshaling)
- **BinaryReader pooling**: Added `sync.Pool` for `BinaryReader` instances to reduce allocations
- **String formatting**: Replaced `fmt.Sprintf` with `strconv` + concatenation in hot paths
- **Pre-allocation**: Added capacity hints for maps and slices in critical paths:
  - `deduplicateTrades()`: map/slice pre-allocated to input length
  - `NewDexParser()`: factory maps pre-allocated (20, 10, 5, 10)
  - `GetMultiInstructions()`: result slice pre-allocated
  - `NewTransactionAdapter()`: token maps pre-allocated (32, 16)

### Added
- `utils/format.go`: Optimized string formatting utilities (`FormatTransferKey`, `FormatDedupeKey`)
- `tests/benchmark_test.go`: Performance benchmarks for JSON, BinaryReader, and DexParser

## [1.0.0] - 2026-01-20

### Added
- Initial Go port of `solana-dex-parser` TypeScript library
- Support for 18 DEX trade parsers (Jupiter, Raydium, Meteora, Orca, Pumpfun, etc.)
- Support for 8 liquidity pool parsers
- Support for 8 meme event parsers
- ShredParser for gRPC streams
- 28 integration tests with real Solana transactions
