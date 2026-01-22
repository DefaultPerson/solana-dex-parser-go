# Solana DEX Parser (Go)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Docs](https://img.shields.io/badge/docs-GitHub%20Pages-blue.svg)](https://defaultperson.github.io/solana-dex-parser-go/)

A high-performance Go library for parsing Solana DEX transactions.

Supports **23 full parsers** + **55 routing detection** (program IDs), including Jupiter, Raydium, Orca, Meteora, Pumpfun, Pumpswap, and **9 trading bots** with fee detection + **6 bot programs**.

## Installation

```bash
go get github.com/DefaultPerson/solana-dex-parser-go
```

## Quick Start

```go
package main

import (
    "encoding/json"
    "fmt"

    dexparser "github.com/DefaultPerson/solana-dex-parser-go"
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
- [Examples](https://defaultperson.github.io/solana-dex-parser-go/examples/)

## Supported Protocols

**Status Legend:**
- ✅ Parser — Full trade/liquidity parsing with instruction decoding
- 🔗 Constants — Program ID defined for routing detection only

### DEX Aggregators & Routers
| Protocol | Trades | Liquidity | Transfers | Status |
|----------|--------|-----------|-----------|--------|
| **Jupiter** (V6, DCA, Limit, VA) | ✅ | ❌ | ✅ | ✅ Parser |
| **OKX DEX** | ✅ | ❌ | ✅ | 🔗 Constants |
| **DFlow** | ✅ | ❌ | ✅ | ✅ Parser |
| **Sanctum** | ✅ | ❌ | ✅ | 🔗 Constants |
| **Photon** | ✅ | ❌ | ✅ | ✅ Parser |
| **Raydium Route** | ✅ | ❌ | ✅ | ✅ Parser |

### Major AMMs
| Protocol | Trades | Liquidity | Transfers | Status |
|----------|--------|-----------|-----------|--------|
| **Raydium V4** | ✅ | ✅ | ✅ | ✅ Parser |
| **Raydium CPMM** | ✅ | ✅ | ✅ | ✅ Parser |
| **Raydium CL** | ✅ | ✅ | ✅ | ✅ Parser |
| **Orca Whirlpool** | ✅ | ✅ | ✅ | ✅ Parser |
| **Meteora DLMM** | ✅ | ✅ | ✅ | ✅ Parser |
| **Meteora Pools** | ✅ | ✅ | ✅ | ✅ Parser |
| **Meteora DAMM** | ✅ | ✅ | ✅ | ✅ Parser |
| **PumpSwap** | ✅ | ✅ | ✅ | ✅ Parser |
| **Phoenix** | ✅ | ❌ | ✅ | 🔗 Constants |
| **Lifinity** | ✅ | ❌ | ✅ | 🔗 Constants |
| **Lifinity V2** | ✅ | ❌ | ✅ | 🔗 Constants |
| **OpenBook** | ✅ | ❌ | ✅ | 🔗 Constants |

### Prop AMM / Dark Pools
| Protocol | Trades | Liquidity | Transfers | Status |
|----------|--------|-----------|-----------|--------|
| **SolFi** | ✅ | ❌ | ✅ | ✅ Parser |
| **GoonFi** | ✅ | ❌ | ✅ | ✅ Parser |
| **Obric V2** | ✅ | ❌ | ✅ | ✅ Parser |
| **HumidiFi** | ✅ | ❌ | ✅ | ✅ Parser |

### Meme & Launch Platforms
| Protocol | Trades | Create | Migrate | Status |
|----------|--------|--------|---------|--------|
| **Pumpfun** | ✅ | ✅ | ✅ | ✅ Parser |
| **Raydium Launchpad** | ✅ | ✅ | ✅ | ✅ Parser |
| **Meteora DBC** | ✅ | ✅ | ✅ | ✅ Parser |
| **Moonit** | ✅ | ✅ | ✅ | ✅ Parser |
| **Heaven.xyz** | ✅ | ✅ | ✅ | ✅ Parser |
| **Sugar.money** | ✅ | ✅ | ✅ | ✅ Parser |
| **Bonk** | ✅ | ✅ | ✅ | 🔗 Constants |
| **BoopFun** | ✅ | ✅ | ✅ | ✅ Parser |

### Trading Bots (Fee Account Detection)
| Bot | Detection | Status |
|-----|-----------|--------|
| **Trojan** | ✅ | ✅ Parser |
| **BONKbot** | ✅ | ✅ Parser |
| **Axiom** | ✅ | ✅ Parser |
| **GMGN** | ✅ | ✅ Parser |
| **BullX** | ✅ | ✅ Parser |
| **Maestro** | ✅ | ✅ Parser |
| **Bloom** | ✅ | ✅ Parser |
| **BananaGun** | ✅ | ✅ Parser |
| **Raybot** | ✅ | ✅ Parser |

### Trading Bots (Program Detection)
| Bot | Detection | Status |
|-----|-----------|--------|
| **Mintech** | ✅ | 🔗 Constants |
| **Nova** | ✅ | 🔗 Constants |
| **Apepro** | ✅ | 🔗 Constants |
| **BananaGun** | ✅ | 🔗 Constants |
| **Bloom** | ✅ | 🔗 Constants |
| **Maestro** | ✅ | 🔗 Constants |

### Additional AMMs
| Protocol | Trades | Liquidity | Transfers | Status |
|----------|--------|-----------|-----------|--------|
| **GooseFX** | ✅ | ❌ | ✅ | 🔗 Constants |
| **Mercurial** | ✅ | ❌ | ✅ | 🔗 Constants |
| **Stabble** | ✅ | ❌ | ✅ | 🔗 Constants |
| **1Dex** | ✅ | ❌ | ✅ | 🔗 Constants |
| **ZeroFi** | ✅ | ❌ | ✅ | 🔗 Constants |

### Legacy Protocols
| Protocol | Trades | Liquidity | Status |
|----------|--------|-----------|--------|
| **Serum V3** | ✅ | ❌ | 🔗 Constants |
| **Aldrin** | ✅ | ❌ | 🔗 Constants |
| **Aldrin V2** | ✅ | ❌ | 🔗 Constants |
| **Crema** | ✅ | ❌ | 🔗 Constants |
| **Saber** | ✅ | ❌ | 🔗 Constants |
| **Saros** | ✅ | ❌ | 🔗 Constants |

*Total: 23 full parsers + 55 routing detection (program IDs)*

## Shred Parser Support

Real-time shred-stream processing for live blockchain data analysis via gRPC streams (Helius, Triton, etc.):

```go
parser := dexparser.NewShredParser()
result := parser.ParseAll(&tx, nil)

// Access parsed instructions by program
for program, instructions := range result.Instructions {
    fmt.Printf("%s: %d instructions\n", program, len(instructions))
}
```

### Key Differences: ShredParser vs DexParser

| Feature | DexParser | ShredParser |
|---------|-----------|-------------|
| **Input Data** | Complete transaction | Raw message only |
| **Execution State** | Post-execution | Pre-execution |
| **Transfer Data** | ✅ Actual results | ❌ No execution |
| **Use Case** | Historical analysis | Real-time monitoring |

### Shred Parser Protocol Support

| Protocol | Status | Notes |
|----------|--------|-------|
| **Pumpfun** | ✅ | Buy, Sell, Create, Migrate |
| **PumpSwap** | ✅ | Buy, Sell, Add/Remove Liquidity |
| **Jupiter V6** | ✅ | Route, SharedAccountsRoute |
| **Raydium V4** | ✅ | Swap instructions |
| **Raydium Launchpad** | ✅ | Buy, Sell, Create |
| **Meteora DBC** | ✅ | Dynamic bonding curve |
| **DFlow** | ✅ | Swap routing |
| **Photon** | ✅ | Multi-hop aggregation |
| **System Program** | ✅ | SOL transfers |
| **Token Program** | ✅ | SPL transfers |
| **Token 2022** | ✅ | Token extensions |

## License

MIT License - see [LICENSE](LICENSE)

## Acknowledgments

- [solana-dex-parser](https://github.com/cxcx-ai/solana-dex-parser) - Original TypeScript implementation
