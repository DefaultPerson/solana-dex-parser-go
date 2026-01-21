---
layout: default
title: Home
nav_order: 1
---

# Solana DEX Parser (Go)

A high-performance Go library for parsing Solana DEX transactions. Port of [solana-dex-parser](https://github.com/cxcx-ai/solana-dex-parser) TypeScript library.

## Features

- **Multi-Protocol Support** - Jupiter, Raydium, Orca, Meteora, Pumpfun, Moonit, and 30+ more
- **High Performance** - Optimized JSON parsing, memory pooling, zero-allocation hot paths
- **Rich Data Extraction** - Trades, liquidity events, transfers, fees, meme events
- **gRPC Support** - ShredParser for Helius/Triton streams
- **Type Safety** - Strongly typed Go structs

## Quick Install

```bash
go get github.com/solana-dex-parser-go
```

## Minimal Example

```go
parser := dexparser.NewDexParser()
result := parser.ParseAll(&tx, nil)

fmt.Printf("Trades: %d\n", len(result.Trades))
fmt.Printf("Liquidities: %d\n", len(result.Liquidities))
```

## Documentation

- [Getting Started](getting-started.md) - Installation and first steps
- [Supported Protocols](protocols.md) - Full list of DEXes and platforms
- [Examples](examples/) - Code examples for all use cases
- [API Reference](api/) - Detailed API documentation
- [Development](development.md) - Contributing and testing

## License

MIT License - see [LICENSE](https://github.com/defaultperson/solana-dex-parser-go/blob/main/LICENSE)
