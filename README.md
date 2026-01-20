# Solana DEX Parser (Go)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

A Go library for parsing Solana DEX swap transactions. This is a Go port of the TypeScript library [solana-dex-parser](https://github.com/cxcx-ai/solana-dex-parser).

Supports multiple DEX protocols including Jupiter, Raydium, Meteora, Orca, PumpFun, Pumpswap, Moonit, and more.

## Contents

- [Features](#features)
- [Supported Protocols](#supported-protocols)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Examples](#examples)
- [Testing](#testing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

## Features

- **DexParser** - Parse DEX transactions and extract Trade/Liquidity/Transfer data
- **Multi-Protocol Support** - Jupiter, Raydium, Orca, Meteora, Pumpfun, Moonit, etc.
- **Type Safety** - Strongly typed Go structs
- **High Performance** - Optimized for large transaction volumes
- **Rich Data Extraction** - Trades, liquidity events, transfers, and fees
- **Meme Parsing** - MemeEvent parsers for Pumpfun/MeteoraDBC/Raydium Launchpad/Moonit etc.
- **gRPC Support** - Raw data processing capabilities for Helius/Triton streams

## Supported Protocols

### DEX Aggregators & Routers
| Protocol | Trades | Liquidity | Transfers |
|----------|--------|-----------|-----------|
| **Jupiter** (All versions) | ✅ | ❌ | ✅ |
| **OKX DEX** | ✅ | ❌ | ✅ |

### Major AMMs
| Protocol | Trades | Liquidity | Transfers |
|----------|--------|-----------|-----------|
| **PumpSwap** | ✅ | ✅ | ✅ |
| **Raydium V4** | ✅ | ✅ | ✅ |
| **Raydium CPMM** | ✅ | ✅ | ✅ |
| **Raydium CL** | ✅ | ✅ | ✅ |
| **Orca Whirlpool** | ✅ | ✅ | ✅ |
| **Meteora DLMM** | ✅ | ✅ | ✅ |
| **Meteora Pools** | ✅ | ✅ | ✅ |
| **Meteora DAMM V2** | ✅ | ✅ | ✅ |

### Meme & Launch Platforms
| Protocol | Trades | Create | Migrate |
|----------|--------|--------|---------|
| **Pumpfun** | ✅ | ✅ | ✅ |
| **Raydium Launchpad** | ✅ | ✅ | ✅ |
| **Meteora DBC** | ✅ | ✅ | ✅ |
| **Moonit** | ✅ | ✅ | ✅ |
| **Heaven.xyz** | ✅ | ✅ | ✅ |
| **Sugar.money** | ✅ | ✅ | ✅ |
| **BoopFun** | ✅ | ✅ | ✅ |

## Installation

```bash
go get github.com/solana-dex-parser-go
```

## Quick Start

### Configuration Options

```go
type ParseConfig struct {
    TryUnknownDEX    bool     // Try to parse unknown DEX programs (default: true)
    ProgramIds       []string // Only parse specific program IDs
    IgnoreProgramIds []string // Ignore specific program IDs
    ThrowError       bool     // Panic on errors instead of returning error state
    AggregateTrades  bool     // Aggregate multiple trades into one
}
```

### Parse All (Trades, Liquidity and Transfers)

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    dexparser "github.com/solana-dex-parser-go"
)

func main() {
    // Get transaction from RPC (example using JSON-RPC response)
    txJSON := `{"transaction": {...}, "meta": {...}}`

    var tx dexparser.SolanaTransaction
    json.Unmarshal([]byte(txJSON), &tx)

    // Parse all types
    parser := dexparser.NewDexParser()
    result := parser.ParseAll(&tx, nil)

    fmt.Printf("Trades: %d\n", len(result.Trades))
    fmt.Printf("Liquidities: %d\n", len(result.Liquidities))
    fmt.Printf("Transfers: %d\n", len(result.Transfers))
    fmt.Printf("MemeEvents: %d\n", len(result.MemeEvents))
}
```

### Parse Result Structure

```go
type ParseResult struct {
    State            bool                    // Parsing success status
    Fee              types.TokenAmount       // Transaction gas fee
    AggregateTrade   *types.TradeInfo        // Aggregated trade info
    Trades           []types.TradeInfo       // Individual trades
    Liquidities      []types.PoolEvent       // Liquidity operations
    Transfers        []types.TransferData    // Token transfers
    MemeEvents       []types.MemeEvent       // Meme platform events
    Slot             uint64                  // Solana slot number
    Timestamp        int64                   // Unix timestamp
    Signature        string                  // Transaction signature
    Signer           []string                // Signers
    Msg              string                  // Error message if any
}
```

## Examples

### Parse DEX Trades

```go
parser := dexparser.NewDexParser()
trades := parser.ParseTrades(&tx, nil)

for _, trade := range trades {
    fmt.Printf("Type: %s\n", trade.Type)
    fmt.Printf("AMM: %s\n", trade.AMM)
    fmt.Printf("Input: %s (%f)\n", trade.InputToken.Mint, trade.InputToken.Amount)
    fmt.Printf("Output: %s (%f)\n", trade.OutputToken.Mint, trade.OutputToken.Amount)
    fmt.Printf("User: %s\n", trade.User)
}
```

### Parse Liquidity Events

```go
parser := dexparser.NewDexParser()
events := parser.ParseLiquidity(&tx, nil)

for _, event := range events {
    fmt.Printf("Type: %s\n", event.Type) // CREATE, ADD, REMOVE
    fmt.Printf("Pool: %s\n", event.PoolId)
    fmt.Printf("Token0: %s\n", event.Token0Mint)
    fmt.Printf("Token1: %s\n", event.Token1Mint)
}
```

### Parse Meme Events (Pumpfun, etc.)

```go
parser := dexparser.NewDexParser()
result := parser.ParseAll(&tx, nil)

for _, event := range result.MemeEvents {
    fmt.Printf("Type: %s\n", event.Type) // CREATE, BUY, SELL, COMPLETE, MIGRATE
    fmt.Printf("Protocol: %s\n", event.Protocol)
    fmt.Printf("Mint: %s\n", event.BaseMint)
    fmt.Printf("User: %s\n", event.User)
}
```

### ShredParser for gRPC Streams

For parsing raw transaction data from Helius/Triton gRPC streams:

```go
shredParser := dexparser.NewShredParser()
result := shredParser.ParseAll(&tx, &types.ParseConfig{
    ProgramIds: []string{constants.DEX_PROGRAMS.PUMP_FUN.ID},
})

for program, instructions := range result.Instructions {
    fmt.Printf("Program: %s, Instructions: %d\n", program, len(instructions))
}
```

### Raydium Logs Decode

```go
import "github.com/solana-dex-parser-go/parsers/raydium"

log := raydium.DecodeRaydiumLog(logData)
if log != nil {
    if swap := raydium.ParseRaydiumSwapLog(log); swap != nil {
        fmt.Printf("Type: %s\n", swap.Type) // "Buy" or "Sell"
        fmt.Printf("Mode: %s\n", swap.Mode) // "Exact Input" or "Exact Output"
        fmt.Printf("InputAmount: %s\n", swap.InputAmount.String())
        fmt.Printf("OutputAmount: %s\n", swap.OutputAmount.String())
    }
}
```

## Testing

Run all tests:
```bash
go test ./... -v
```

Run integration tests:
```bash
go test ./tests -v -run TestIntegration
```

## Development

### Prerequisites

- Go 1.21+

### Setup

```bash
git clone https://github.com/solana-dex-parser-go.git
cd solana-dex-parser-go
go mod download
```

### Build

```bash
go build ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- **[solana-dex-parser](https://github.com/cxcx-ai/solana-dex-parser)** - Original TypeScript implementation that this Go library is ported from
