# Raydium Logs Decode

Decode Raydium swap logs from transaction logs.

## Basic Usage

```go
package main

import (
    "fmt"

    "github.com/solana-dex-parser-go/parsers/raydium"
)

func main() {
    // logData is the base64-encoded log from transaction
    log := raydium.DecodeRaydiumLog(logData)

    if log != nil {
        if swap := raydium.ParseRaydiumSwapLog(log); swap != nil {
            fmt.Printf("Type: %s\n", swap.Type)
            fmt.Printf("Mode: %s\n", swap.Mode)
            fmt.Printf("InputAmount: %s\n", swap.InputAmount.String())
            fmt.Printf("OutputAmount: %s\n", swap.OutputAmount.String())
        }
    }
}
```

## SwapOperation Structure

```go
type SwapOperation struct {
    Type         string   // "Buy" or "Sell"
    Mode         string   // "Exact Input" or "Exact Output"
    InputAmount  *big.Int // Input token amount (raw)
    OutputAmount *big.Int // Output token amount (raw)
}
```

## Log Format

Raydium logs are base64-encoded and contain:

- Log discriminator
- Swap direction (buy/sell)
- Input/output amounts
- Pool state changes

## Example with Transaction

```go
import (
    "github.com/solana-dex-parser-go/adapter"
    "github.com/solana-dex-parser-go/parsers/raydium"
)

func parseRaydiumTx(tx *SolanaTransaction) {
    txAdapter := adapter.NewTransactionAdapter(tx)

    // Get logs from transaction
    logs := txAdapter.GetLogs()

    for _, log := range logs {
        // Check if it's a Raydium log
        if decoded := raydium.DecodeRaydiumLog(log); decoded != nil {
            if swap := raydium.ParseRaydiumSwapLog(decoded); swap != nil {
                fmt.Printf("Raydium %s: %s -> %s\n",
                    swap.Type,
                    swap.InputAmount.String(),
                    swap.OutputAmount.String())
            }
        }
    }
}
```
