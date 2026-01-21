# API Reference

Detailed documentation for all public APIs.

## Quick Links

- [DexParser](dex-parser.md) - Main parser class
- [ShredParser](shred-parser.md) - gRPC stream parser
- [Types](types.md) - Type definitions

## Package Structure

```go
import (
    dexparser "github.com/solana-dex-parser-go"
    "github.com/solana-dex-parser-go/types"
    "github.com/solana-dex-parser-go/constants"
    "github.com/solana-dex-parser-go/adapter"
)
```

## Main Entry Points

| Class | Description |
|-------|-------------|
| `DexParser` | Parse DEX transactions (trades, liquidity, transfers) |
| `ShredParser` | Parse raw instruction data from gRPC streams |

## Constants

```go
// Access program IDs
constants.DEX_PROGRAMS.JUPITER_V6.ID
constants.DEX_PROGRAMS.RAYDIUM_V4.ID
constants.DEX_PROGRAMS.PUMP_FUN.ID
// ... etc

// Token addresses
constants.TOKENS.WSOL
constants.TOKENS.USDC
constants.TOKENS.USDT
```
