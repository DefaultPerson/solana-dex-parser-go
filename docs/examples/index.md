# Examples

Practical code examples for common use cases.

## Parse All Data

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"

    dexparser "github.com/DefaultPerson/solana-dex-parser-go"
    "github.com/DefaultPerson/solana-dex-parser-go/adapter"
)

func main() {
    // Get transaction from RPC
    signature := "4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz"
    tx, _ := getTransaction(signature, "https://api.mainnet-beta.solana.com")

    // Parse all data in one call
    parser := dexparser.NewDexParser()
    result := parser.ParseAll(tx, nil)

    fmt.Printf("Trades: %d\n", len(result.Trades))
    fmt.Printf("Liquidities: %d\n", len(result.Liquidities))
    fmt.Printf("Transfers: %d\n", len(result.Transfers))
    fmt.Printf("MemeEvents: %d\n", len(result.MemeEvents))
}

func getTransaction(sig, rpc string) (*adapter.SolanaTransaction, error) {
    payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"getTransaction","params":["%s",{"encoding":"jsonParsed","maxSupportedTransactionVersion":0}]}`, sig)
    resp, err := http.Post(rpc, "application/json", strings.NewReader(payload))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    var rpcResp struct {
        Result *adapter.SolanaTransaction `json:"result"`
    }
    json.Unmarshal(body, &rpcResp)
    return rpcResp.Result, nil
}
```

**Output:**
```
Trades: 1
Liquidities: 0
Transfers: 2
MemeEvents: 1
```

## Parse Trades

```go
parser := dexparser.NewDexParser()
trades := parser.ParseTrades(&tx, nil)

for _, trade := range trades {
    fmt.Printf("Type: %s\n", trade.Type)
    fmt.Printf("AMM: %s\n", trade.AMM)
    fmt.Printf("Input: %s (%.6f)\n", trade.InputToken.Mint[:8], trade.InputToken.Amount)
    fmt.Printf("Output: %s (%.6f)\n", trade.OutputToken.Mint[:8], trade.OutputToken.Amount)
    fmt.Printf("User: %s\n", trade.User)
}
```

**Output:**
```
Type: BUY
AMM: Pumpfun
Input: So11111.. (0.050000)
Output: 9gyfSMQ.. (1234567.890000)
User: 7xKXtg2..
```

## Parse Liquidity Events

```go
events := parser.ParseLiquidity(&tx, nil)

for _, event := range events {
    fmt.Printf("Type: %s\n", event.Type)
    fmt.Printf("Pool: %s\n", event.PoolId[:8])
    fmt.Printf("Token0: %s (%.2f)\n", event.Token0Mint[:8], event.Token0Amount)
    fmt.Printf("Token1: %s (%.2f)\n", event.Token1Mint[:8], event.Token1Amount)
    fmt.Printf("LP Tokens: %.2f\n", event.LpAmount)
}
```

**Output:**
```
Type: ADD
Pool: 5Q544fK..
Token0: So11111.. (10.00)
Token1: EPjFWdd.. (1500.00)
LP Tokens: 122.47
```

## Parse Meme Events

```go
result := parser.ParseAll(&tx, nil)

for _, event := range result.MemeEvents {
    fmt.Printf("Type: %s\n", event.Type)
    fmt.Printf("Protocol: %s\n", event.Protocol)
    fmt.Printf("Mint: %s\n", event.BaseMint[:8])
    fmt.Printf("User: %s\n", event.User[:8])
}
```

**Output:**
```
Type: BUY
Protocol: Pumpfun
Mint: 9gyfSMQ..
User: 7xKXtg2..
```

## Filter by Program

```go
import "github.com/DefaultPerson/solana-dex-parser-go/constants"

config := &types.ParseConfig{
    ProgramIds: []string{
        constants.DEX_PROGRAMS.PUMP_FUN.ID,
        constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
    },
}
result := parser.ParseAll(tx, config)
```

## Ignore Specific Programs

```go
config := &types.ParseConfig{
    IgnoreProgramIds: []string{
        constants.DEX_PROGRAMS.PHOENIX.ID,
    },
}
result := parser.ParseAll(&tx, config)
```

## Aggregate Trades

```go
config := &types.ParseConfig{
    AggregateTrades: true,
}
result := parser.ParseAll(&tx, config)

if result.AggregateTrade != nil {
    fmt.Printf("Total Input: %.6f\n", result.AggregateTrade.InputToken.Amount)
    fmt.Printf("Total Output: %.6f\n", result.AggregateTrade.OutputToken.Amount)
}
```

## ShredParser for gRPC Streams

ShredParser provides pre-execution instruction analysis for real-time blockchain monitoring via gRPC streams (Helius, Triton, etc.).

### Basic Usage

```go
import (
    dexparser "github.com/DefaultPerson/solana-dex-parser-go"
    "github.com/DefaultPerson/solana-dex-parser-go/constants"
    "github.com/DefaultPerson/solana-dex-parser-go/types"
)

shredParser := dexparser.NewShredParser()

config := &types.ParseConfig{
    ProgramIds: []string{
        constants.DEX_PROGRAMS.PUMP_FUN.ID,
        constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
    },
}

result := shredParser.ParseAll(&tx, config)

// Access parsed instructions by program
for program, instructions := range result.Instructions {
    fmt.Printf("[%s] %d instructions\n", program[:8], len(instructions))
}

// Access typed instructions
for _, inst := range result.ParsedInstructions {
    fmt.Printf("[%s] %s\n", inst.ProgramName, inst.Action)
    if inst.Trade != nil {
        fmt.Printf("  Trade: %s -> %s\n",
            inst.Trade.InputToken.Mint[:8],
            inst.Trade.OutputToken.Mint[:8])
    }
}
```

**Output:**
```
[6EF8rre..] 1 instructions
[Pumpfun] BUY
  Trade: So11111.. -> 9gyfSMQ..
```

### Supported Protocols

| Protocol | Instructions | Notes |
|----------|--------------|-------|
| **Jupiter V6** | Route variants | All route types including shared accounts |
| **Raydium V4** | Swap, Create, Add/Remove Liquidity | Full AMM support |
| **Raydium Launchpad** | Buy, Sell, Create, Migrate | Meme token launches |
| **Meteora DBC** | Swap, Init Pool, Migrate | Dynamic bonding curve |
| **DFlow** | Swap routing | Order flow aggregation |
| **Photon** | Multi-hop swaps | Cross-AMM routing |
| **System/Token** | Transfers | SOL and SPL tokens |

### Use Cases

- **MEV Detection**: Monitor instructions pre-execution
- **Real-time Pricing**: Track incoming trades
- **Launch Monitoring**: Detect new token launches instantly

## Raydium Logs Decode

```go
import "github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"

// logData is the base64-encoded log from transaction
log := raydium.DecodeRaydiumLog(logData)

if log != nil {
    if swap := raydium.ParseRaydiumSwapLog(log); swap != nil {
        fmt.Printf("Type: %s\n", swap.Type)
        fmt.Printf("Mode: %s\n", swap.Mode)
        fmt.Printf("Input: %s\n", swap.InputAmount.String())
        fmt.Printf("Output: %s\n", swap.OutputAmount.String())
    }
}
```

**Output:**
```
Type: Buy
Mode: Exact Input
Input: 50000000
Output: 1234567890000
```
