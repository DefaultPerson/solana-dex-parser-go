---
layout: default
title: Parse Liquidity
parent: Examples
nav_order: 2
---

# Parse Liquidity Events

## Basic Liquidity Parsing

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
    events := parser.ParseLiquidity(&tx, nil)

    for _, event := range events {
        fmt.Printf("Type: %s\n", event.Type)
        fmt.Printf("Pool: %s\n", event.PoolId)
        fmt.Printf("Token0: %s (Amount: %f)\n", event.Token0Mint, event.Token0Amount)
        fmt.Printf("Token1: %s (Amount: %f)\n", event.Token1Mint, event.Token1Amount)
        fmt.Printf("LP Tokens: %f\n", event.LpAmount)
        fmt.Println("---")
    }
}
```

## PoolEvent Structure

```go
type PoolEvent struct {
    Type         string  // "CREATE", "ADD", "REMOVE"
    PoolId       string  // Pool address
    Token0Mint   string  // First token mint
    Token0Amount float64 // First token amount
    Token1Mint   string  // Second token mint
    Token1Amount float64 // Second token amount
    LpMint       string  // LP token mint (if applicable)
    LpAmount     float64 // LP tokens minted/burned
    User         string  // User wallet address
    ProgramId    string  // AMM program ID
    AMM          string  // AMM name
    Slot         uint64  // Solana slot
    Timestamp    int64   // Unix timestamp
    Signature    string  // Transaction signature
    Idx          int     // Instruction index
    InnerIdx     int     // Inner instruction index
}
```

## Event Types

### CREATE - Pool Creation

```go
for _, event := range events {
    if event.Type == "CREATE" {
        fmt.Printf("New pool created: %s\n", event.PoolId)
        fmt.Printf("Initial liquidity: %s/%s\n", event.Token0Mint, event.Token1Mint)
    }
}
```

### ADD - Add Liquidity

```go
for _, event := range events {
    if event.Type == "ADD" {
        fmt.Printf("Liquidity added to pool: %s\n", event.PoolId)
        fmt.Printf("Amount: %f %s + %f %s\n",
            event.Token0Amount, event.Token0Mint,
            event.Token1Amount, event.Token1Mint)
        fmt.Printf("LP tokens received: %f\n", event.LpAmount)
    }
}
```

### REMOVE - Remove Liquidity

```go
for _, event := range events {
    if event.Type == "REMOVE" {
        fmt.Printf("Liquidity removed from pool: %s\n", event.PoolId)
        fmt.Printf("LP tokens burned: %f\n", event.LpAmount)
        fmt.Printf("Received: %f %s + %f %s\n",
            event.Token0Amount, event.Token0Mint,
            event.Token1Amount, event.Token1Mint)
    }
}
```

## Filter by AMM

```go
config := &types.ParseConfig{
    ProgramIds: []string{
        constants.DEX_PROGRAMS.METEORA_DLMM.ID,
    },
}

events := parser.ParseLiquidity(&tx, config)
```
