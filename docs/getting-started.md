# Getting Started

## Prerequisites

- Go 1.21 or higher

## Installation

```bash
go get github.com/solana-dex-parser-go
```

## Quick Start

### 1. Import the package

```go
import dexparser "github.com/solana-dex-parser-go"
```

### 2. Create a parser instance

```go
parser := dexparser.NewDexParser()
```

### 3. Parse a transaction

```go
// Get transaction from RPC
var tx dexparser.SolanaTransaction
json.Unmarshal([]byte(txJSON), &tx)

// Parse all data types
result := parser.ParseAll(&tx, nil)

fmt.Printf("Trades: %d\n", len(result.Trades))
fmt.Printf("Liquidities: %d\n", len(result.Liquidities))
fmt.Printf("Transfers: %d\n", len(result.Transfers))
fmt.Printf("MemeEvents: %d\n", len(result.MemeEvents))
```

## Configuration Options

```go
type ParseConfig struct {
    TryUnknownDEX    bool     // Try to parse unknown DEX programs (default: true)
    ProgramIds       []string // Only parse specific program IDs
    IgnoreProgramIds []string // Ignore specific program IDs
    ThrowError       bool     // Panic on errors instead of returning error state
    AggregateTrades  bool     // Aggregate multiple trades into one
}
```

### Example with config

```go
config := &types.ParseConfig{
    ProgramIds: []string{
        constants.DEX_PROGRAMS.PUMP_FUN.ID,
        constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
    },
    AggregateTrades: true,
}

result := parser.ParseAll(&tx, config)
```

## Next Steps

- [Examples](examples/index.md) - Code examples for specific use cases
- [Supported Protocols](protocols.md) - Full list of supported DEXes
- [API Reference](api/index.md) - Detailed API documentation
