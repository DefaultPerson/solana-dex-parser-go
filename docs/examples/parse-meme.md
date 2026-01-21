# Parse Meme Events

Parse events from meme coin platforms like Pumpfun, Raydium Launchpad, Meteora DBC, etc.

## Basic Meme Event Parsing

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

    for _, event := range result.MemeEvents {
        fmt.Printf("Type: %s\n", event.Type)
        fmt.Printf("Protocol: %s\n", event.Protocol)
        fmt.Printf("Mint: %s\n", event.BaseMint)
        fmt.Printf("User: %s\n", event.User)
        fmt.Println("---")
    }
}
```

## MemeEvent Structure

```go
type MemeEvent struct {
    Type        string  // "CREATE", "BUY", "SELL", "COMPLETE", "MIGRATE"
    Protocol    string  // "Pumpfun", "RaydiumLaunchpad", "MeteoraDBC", etc.
    BaseMint    string  // Token mint address
    QuoteMint   string  // Quote token (usually SOL)
    BaseAmount  float64 // Token amount
    QuoteAmount float64 // SOL amount
    User        string  // User wallet
    BondingCurve string // Bonding curve address (Pumpfun)
    Pool        string  // Pool address (after migration)
    Slot        uint64  // Solana slot
    Timestamp   int64   // Unix timestamp
    Signature   string  // Transaction signature
    Idx         int     // Instruction index
    InnerIdx    int     // Inner instruction index
}
```

## Event Types

### CREATE - Token Launch

```go
for _, event := range result.MemeEvents {
    if event.Type == "CREATE" {
        fmt.Printf("New token launched on %s\n", event.Protocol)
        fmt.Printf("Mint: %s\n", event.BaseMint)
        fmt.Printf("Creator: %s\n", event.User)
    }
}
```

### BUY/SELL - Trading on Bonding Curve

```go
for _, event := range result.MemeEvents {
    if event.Type == "BUY" || event.Type == "SELL" {
        fmt.Printf("%s on %s\n", event.Type, event.Protocol)
        fmt.Printf("Token: %s\n", event.BaseMint)
        fmt.Printf("Amount: %f tokens for %f SOL\n",
            event.BaseAmount, event.QuoteAmount)
    }
}
```

### COMPLETE - Bonding Curve Completion

```go
for _, event := range result.MemeEvents {
    if event.Type == "COMPLETE" {
        fmt.Printf("Bonding curve completed for %s\n", event.BaseMint)
        fmt.Printf("Protocol: %s\n", event.Protocol)
    }
}
```

### MIGRATE - DEX Migration

```go
for _, event := range result.MemeEvents {
    if event.Type == "MIGRATE" {
        fmt.Printf("Token %s migrated to DEX\n", event.BaseMint)
        fmt.Printf("New pool: %s\n", event.Pool)
    }
}
```

## Supported Platforms

| Platform | Description |
|----------|-------------|
| **Pumpfun** | Original bonding curve platform |
| **Raydium Launchpad** | Raydium's meme launcher |
| **Meteora DBC** | Meteora's dynamic bonding curve |
| **Moonit** | DEXScreener's platform |
| **Heaven.xyz** | Heaven launcher |
| **Sugar.money** | Sugar launcher |
| **BoopFun** | Boop launcher |
| **Bonk** | Bonk launcher |
