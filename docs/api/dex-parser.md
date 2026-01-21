# DexParser

Main class for parsing Solana DEX transactions.

## Constructor

```go
func NewDexParser() *DexParser
```

Creates a new DexParser instance.

## Methods

### ParseAll

```go
func (p *DexParser) ParseAll(tx *SolanaTransaction, config *types.ParseConfig) *types.ParseResult
```

Parse all data types from a transaction.

**Parameters:**

- `tx` - Solana transaction (JSON-RPC format)
- `config` - Optional configuration (can be nil)

**Returns:** `ParseResult` containing trades, liquidity events, transfers, and meme events.

### ParseTrades

```go
func (p *DexParser) ParseTrades(tx *SolanaTransaction, config *types.ParseConfig) []types.TradeInfo
```

Parse only trade/swap data.

### ParseLiquidity

```go
func (p *DexParser) ParseLiquidity(tx *SolanaTransaction, config *types.ParseConfig) []types.PoolEvent
```

Parse only liquidity pool events.

### ParseTransfers

```go
func (p *DexParser) ParseTransfers(tx *SolanaTransaction, config *types.ParseConfig) []types.TransferData
```

Parse only token transfers.

## ParseConfig

```go
type ParseConfig struct {
    TryUnknownDEX    bool     // Try to parse unknown DEX programs (default: true)
    ProgramIds       []string // Only parse specific program IDs
    IgnoreProgramIds []string // Ignore specific program IDs
    ThrowError       bool     // Panic on errors instead of returning error state
    AggregateTrades  bool     // Aggregate multiple trades into one
}
```

## ParseResult

```go
type ParseResult struct {
    State          bool             // Parsing success status
    Fee            TokenAmount      // Transaction gas fee
    AggregateTrade *TradeInfo       // Aggregated trade (if enabled)
    Trades         []TradeInfo      // Individual trades
    Liquidities    []PoolEvent      // Liquidity operations
    Transfers      []TransferData   // Token transfers
    MemeEvents     []MemeEvent      // Meme platform events
    Slot           uint64           // Solana slot number
    Timestamp      int64            // Unix timestamp
    Signature      string           // Transaction signature
    Signer         []string         // Signers
    Msg            string           // Error message if any
}
```

## Example

```go
parser := dexparser.NewDexParser()

// Parse all
result := parser.ParseAll(&tx, nil)
if result.State {
    for _, trade := range result.Trades {
        fmt.Printf("%s: %s -> %s\n",
            trade.AMM,
            trade.InputToken.Mint,
            trade.OutputToken.Mint)
    }
}

// Parse with config
config := &types.ParseConfig{
    ProgramIds: []string{constants.DEX_PROGRAMS.JUPITER_V6.ID},
    AggregateTrades: true,
}
result = parser.ParseAll(&tx, config)
```
