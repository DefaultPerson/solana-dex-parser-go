# Solana DEX Parser (Go)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/DefaultPerson/solana-dex-parser-go/blob/main/LICENSE)
[![Tests](https://github.com/DefaultPerson/solana-dex-parser-go/actions/workflows/test.yml/badge.svg)](https://github.com/DefaultPerson/solana-dex-parser-go/actions/workflows/test.yml)

A high-performance Go library for parsing Solana DEX transactions. Port of [solana-dex-parser](https://github.com/cxcx-ai/solana-dex-parser) TypeScript library.

## Features

- **Multi-Protocol Support** - Jupiter, Raydium, Orca, Meteora, Pumpfun, Moonit, and 30+ more
- **High Performance** - Optimized JSON parsing, memory pooling, zero-allocation hot paths
- **Rich Data Extraction** - Trades, liquidity events, transfers, fees, meme events
- **gRPC Support** - ShredParser for Helius/Triton streams
- **Type Safety** - Strongly typed Go structs

## Quick Install

```bash
go get github.com/DefaultPerson/solana-dex-parser-go
```

## Minimal Example

```go
parser := dexparser.NewDexParser()
result := parser.ParseAll(&tx, nil)

fmt.Printf("Trades: %d\n", len(result.Trades))
fmt.Printf("Liquidities: %d\n", len(result.Liquidities))
```

## Supported Protocols

| Category | Protocols |
|----------|-----------|
| **Aggregators** | Jupiter, OKX DEX |
| **AMMs** | Raydium (V4, CPMM, CL), Orca, Meteora (DLMM, DAMM), PumpSwap |
| **Meme Platforms** | Pumpfun, Raydium Launchpad, Meteora DBC, Moonit, Heaven, Sugar, Bonk, BoopFun |
| **Trading Bots** | BananaGun, Maestro, Nova, Bloom, Mintech, Apepro |

## Documentation

- [Getting Started](getting-started.md) - Installation and first steps
- [Examples](examples/index.md) - Code examples for all use cases
- [Development](development.md) - Contributing and testing

## License

MIT License - see [LICENSE](https://github.com/DefaultPerson/solana-dex-parser-go/blob/main/LICENSE)
