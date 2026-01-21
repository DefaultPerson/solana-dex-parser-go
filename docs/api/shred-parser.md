# ShredParser

Parser for raw transaction data from gRPC streams (Helius/Triton).

## Constructor

```go
func NewShredParser() *ShredParser
```

Creates a new ShredParser instance.

## Methods

### ParseAll

```go
func (p *ShredParser) ParseAll(tx *SolanaTransaction, config *types.ParseConfig) *ShredParseResult
```

Parse raw instructions from a transaction.

**Parameters:**

- `tx` - Solana transaction
- `config` - Optional configuration

**Returns:** `ShredParseResult` with parsed instructions grouped by program.

## ShredParseResult

```go
type ShredParseResult struct {
    Instructions map[string][]ParsedInstruction // Program ID -> Instructions
    Slot         uint64
    Signature    string
}
```

## ParsedInstruction

```go
type ParsedInstruction struct {
    Type     string      // Instruction type
    Data     interface{} // Parsed data (type depends on program)
    Idx      int         // Instruction index
    InnerIdx int         // Inner instruction index
}
```

## Supported Programs

### Pumpfun

Instruction types: `CREATE`, `BUY`, `SELL`, `MIGRATE`

```go
type PumpfunCreateData struct {
    Mint         string
    BondingCurve string
    Creator      string
}

type PumpfunTradeData struct {
    Type         string // "BUY" or "SELL"
    Mint         string
    TokenAmount  uint64
    SolAmount    uint64
    User         string
}
```

### Pumpswap

Instruction types: `CREATE`, `ADD`, `REMOVE`, `BUY`, `SELL`

```go
type PumpswapTradeData struct {
    Type        string // "BUY" or "SELL"
    Pool        string
    TokenAmount uint64
    SolAmount   uint64
    User        string
}
```

## Example

```go
shredParser := dexparser.NewShredParser()

config := &types.ParseConfig{
    ProgramIds: []string{
        constants.DEX_PROGRAMS.PUMP_FUN.ID,
        constants.DEX_PROGRAMS.PUMP_SWAP.ID,
    },
}

result := shredParser.ParseAll(&tx, config)

for programId, instructions := range result.Instructions {
    fmt.Printf("Program: %s\n", programId)
    for _, inst := range instructions {
        fmt.Printf("  %s: %+v\n", inst.Type, inst.Data)
    }
}
```
