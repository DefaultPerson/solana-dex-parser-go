# ShredParser for gRPC Streams

ShredParser is designed for parsing raw transaction data from Helius/Triton gRPC streams.

## Basic Usage

```go
package main

import (
    "fmt"

    dexparser "github.com/solana-dex-parser-go"
    "github.com/solana-dex-parser-go/constants"
    "github.com/solana-dex-parser-go/types"
)

func main() {
    shredParser := dexparser.NewShredParser()

    // Parse with specific program filter
    result := shredParser.ParseAll(&tx, &types.ParseConfig{
        ProgramIds: []string{constants.DEX_PROGRAMS.PUMP_FUN.ID},
    })

    for program, instructions := range result.Instructions {
        fmt.Printf("Program: %s, Instructions: %d\n", program, len(instructions))
    }
}
```

## ShredParseResult Structure

```go
type ShredParseResult struct {
    Instructions map[string][]ParsedInstruction // Program ID -> Instructions
    Slot         uint64
    Signature    string
}

type ParsedInstruction struct {
    Type     string      // Instruction type (e.g., "BUY", "SELL", "CREATE")
    Data     interface{} // Parsed instruction data
    Idx      int         // Instruction index
    InnerIdx int         // Inner instruction index
}
```

## Pumpfun Instructions

```go
config := &types.ParseConfig{
    ProgramIds: []string{constants.DEX_PROGRAMS.PUMP_FUN.ID},
}

result := shredParser.ParseAll(&tx, config)

for _, inst := range result.Instructions[constants.DEX_PROGRAMS.PUMP_FUN.ID] {
    switch inst.Type {
    case "CREATE":
        // Token creation
        data := inst.Data.(*pumpfun.CreateInstruction)
        fmt.Printf("Token created: %s\n", data.Mint)

    case "BUY":
        // Buy on bonding curve
        data := inst.Data.(*pumpfun.TradeInstruction)
        fmt.Printf("Buy: %d tokens\n", data.TokenAmount)

    case "SELL":
        // Sell on bonding curve
        data := inst.Data.(*pumpfun.TradeInstruction)
        fmt.Printf("Sell: %d tokens\n", data.TokenAmount)

    case "MIGRATE":
        // Migration to DEX
        data := inst.Data.(*pumpfun.MigrateInstruction)
        fmt.Printf("Migrated to: %s\n", data.Pool)
    }
}
```

## Pumpswap Instructions

```go
config := &types.ParseConfig{
    ProgramIds: []string{constants.DEX_PROGRAMS.PUMP_SWAP.ID},
}

result := shredParser.ParseAll(&tx, config)

for _, inst := range result.Instructions[constants.DEX_PROGRAMS.PUMP_SWAP.ID] {
    switch inst.Type {
    case "CREATE":
        // Pool creation
    case "ADD":
        // Add liquidity
    case "REMOVE":
        // Remove liquidity
    case "BUY":
        // Buy swap
    case "SELL":
        // Sell swap
    }
}
```

## Processing gRPC Stream

```go
func processStream(stream <-chan *Transaction) {
    shredParser := dexparser.NewShredParser()

    for tx := range stream {
        result := shredParser.ParseAll(tx, nil)

        // Process instructions
        for program, instructions := range result.Instructions {
            for _, inst := range instructions {
                handleInstruction(program, inst)
            }
        }
    }
}
```
