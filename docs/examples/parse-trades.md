# Parse DEX Trades

## Basic Trade Parsing

```go
package main

import (
    "encoding/json"
    "fmt"

    dexparser "github.com/solana-dex-parser-go"
)

func main() {
    // Get transaction from RPC
    var tx dexparser.SolanaTransaction
    json.Unmarshal([]byte(txJSON), &tx)

    parser := dexparser.NewDexParser()
    trades := parser.ParseTrades(&tx, nil)

    for _, trade := range trades {
        fmt.Printf("Type: %s\n", trade.Type)
        fmt.Printf("AMM: %s\n", trade.AMM)
        fmt.Printf("Input: %s (%f)\n", trade.InputToken.Mint, trade.InputToken.Amount)
        fmt.Printf("Output: %s (%f)\n", trade.OutputToken.Mint, trade.OutputToken.Amount)
        fmt.Printf("User: %s\n", trade.User)
        fmt.Println("---")
    }
}
```

## TradeInfo Structure

```go
type TradeInfo struct {
    Type        string    // "BUY" or "SELL"
    InputToken  TokenInfo // Token sent
    OutputToken TokenInfo // Token received
    User        string    // User wallet address
    ProgramId   string    // DEX program ID
    AMM         string    // AMM name (e.g., "Raydium", "Jupiter")
    Route       string    // Route if via aggregator
    PoolId      string    // Pool address (if available)
    Slot        uint64    // Solana slot number
    Timestamp   int64     // Unix timestamp
    Signature   string    // Transaction signature
    Idx         int       // Instruction index
    InnerIdx    int       // Inner instruction index
}

type TokenInfo struct {
    Mint     string  // Token mint address
    Amount   float64 // Token amount
    Decimals uint8   // Token decimals
}
```

## Filter by Specific DEX

```go
import "github.com/solana-dex-parser-go/constants"

config := &types.ParseConfig{
    ProgramIds: []string{
        constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
        constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID,
    },
}

trades := parser.ParseTrades(&tx, config)
```

## Aggregate Multiple Trades

```go
config := &types.ParseConfig{
    AggregateTrades: true,
}

result := parser.ParseAll(&tx, config)

// result.AggregateTrade contains the combined trade
if result.AggregateTrade != nil {
    fmt.Printf("Total Input: %f\n", result.AggregateTrade.InputToken.Amount)
    fmt.Printf("Total Output: %f\n", result.AggregateTrade.OutputToken.Amount)
}
```
