---
layout: default
title: Examples
nav_order: 4
has_children: true
---

# Code Examples

Practical examples for common use cases.

## Quick Links

- [Parse Trades](parse-trades.md) - Extract swap/trade data from transactions
- [Parse Liquidity](parse-liquidity.md) - Track liquidity pool events
- [Parse Meme Events](parse-meme.md) - Monitor meme coin platforms
- [ShredParser](shred-parser.md) - Process gRPC streams
- [Raydium Logs](raydium-logs.md) - Decode Raydium swap logs

## Common Patterns

### Parse Everything

```go
parser := dexparser.NewDexParser()
result := parser.ParseAll(&tx, nil)

// Access all data
result.Trades      // []TradeInfo
result.Liquidities // []PoolEvent
result.Transfers   // []TransferData
result.MemeEvents  // []MemeEvent
```

### Filter by Program

```go
config := &types.ParseConfig{
    ProgramIds: []string{
        constants.DEX_PROGRAMS.JUPITER_V6.ID,
    },
}
result := parser.ParseAll(&tx, config)
```

### Ignore Specific Programs

```go
config := &types.ParseConfig{
    IgnoreProgramIds: []string{
        constants.DEX_PROGRAMS.PHOENIX.ID,
    },
}
result := parser.ParseAll(&tx, config)
```

### Aggregate Trades

```go
config := &types.ParseConfig{
    AggregateTrades: true,
}
result := parser.ParseAll(&tx, config)

// result.AggregateTrade contains combined trade
```
