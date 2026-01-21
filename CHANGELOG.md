# Changelog

All notable changes to this project will be documented in this file.

## [1.1.0] - 2026-01-21

### Added
- Full documentation site with GitHub Pages
- Documentation for all APIs, types, and examples
- GitHub Actions workflow for docs deployment
- Test parity with TypeScript (33 integration tests)
- Route detection in trade parsing (BananaGun, Maestro, OKX, Jupiter, Bloom)
- Deterministic program ID ordering in classifier

### Changed
- README restructured with links to docs/
- GetDexInfo() now correctly detects both AMM and Route

### Fixed
- Non-deterministic Route detection due to map iteration order
- Float precision issue in transfer deduplication

### Performance
- JSON parsing: Replaced `encoding/json` with `goccy/go-json` (~2x faster)
- BinaryReader pooling with `sync.Pool` to reduce allocations
- String formatting optimized with `strconv` in hot paths
- Pre-allocation for maps and slices in critical paths

## [1.0.0] - 2026-01-20

### Added
- Initial Go port of `solana-dex-parser` TypeScript library
- Support for 18 DEX trade parsers (Jupiter, Raydium, Meteora, Orca, Pumpfun, etc.)
- Support for 8 liquidity pool parsers
- Support for 8 meme event parsers
- ShredParser for gRPC streams
- 28 integration tests with real Solana transactions
