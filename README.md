# Solana DEX Parser (Go)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Docs](https://img.shields.io/badge/docs-GitHub%20Pages-blue.svg)](https://defaultperson.github.io/solana-dex-parser-go/)

A high-performance Go library for parsing Solana DEX transactions. Port of [solana-dex-parser](https://github.com/cxcx-ai/solana-dex-parser).

Supports **30+ protocols** including Jupiter, Raydium, Orca, Meteora, Pumpfun, Pumpswap, Moonit, and more.

## Installation

```bash
go get github.com/solana-dex-parser-go
```

## Quick Start

```go
package main

import (
    "encoding/json"
    "fmt"

    dexparser "github.com/solana-dex-parser-go"
)

func main() {
    var tx dexparser.SolanaTransaction
    json.Unmarshal([]byte(txJSON), &tx)

    parser := dexparser.NewDexParser()
    result := parser.ParseAll(&tx, nil)

    fmt.Printf("Trades: %d\n", len(result.Trades))
    fmt.Printf("Liquidities: %d\n", len(result.Liquidities))
    fmt.Printf("MemeEvents: %d\n", len(result.MemeEvents))
}
```

## Features

- **Multi-Protocol** - Jupiter, Raydium, Orca, Meteora, Pumpfun, Moonit, etc.
- **Rich Data** - Trades, liquidity events, transfers, meme events
- **High Performance** - Optimized JSON parsing, memory pooling
- **gRPC Support** - ShredParser for Helius/Triton streams

## Documentation

- [Getting Started](https://defaultperson.github.io/solana-dex-parser-go/getting-started)
- [Supported Protocols](https://defaultperson.github.io/solana-dex-parser-go/protocols)
- [Code Examples](https://defaultperson.github.io/solana-dex-parser-go/examples/)
- [API Reference](https://defaultperson.github.io/solana-dex-parser-go/api/)

## Supported Protocols

| Category | Protocols |
|----------|-----------|
| **Aggregators** | Jupiter, OKX DEX |
| **AMMs** | Raydium (V4, CPMM, CL), Orca, Meteora (DLMM, DAMM), PumpSwap |
| **Meme Platforms** | Pumpfun, Raydium Launchpad, Meteora DBC, Moonit, Heaven, Sugar, Bonk, BoopFun |
| **Trading Bots** | BananaGun, Maestro, Nova, Bloom, Mintech, Apepro |

[Full protocol list](https://defaultperson.github.io/solana-dex-parser-go/protocols)

## License

MIT License - see [LICENSE](LICENSE)

## Acknowledgments

- [solana-dex-parser](https://github.com/cxcx-ai/solana-dex-parser) - Original TypeScript implementation
