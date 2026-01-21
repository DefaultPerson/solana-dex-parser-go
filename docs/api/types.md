# Type Definitions

## TradeInfo

Represents a single swap/trade operation.

```go
type TradeInfo struct {
    Type        string    // "BUY" or "SELL"
    InputToken  TokenInfo // Token sent by user
    OutputToken TokenInfo // Token received by user
    User        string    // User wallet address
    ProgramId   string    // DEX program ID
    AMM         string    // AMM name (e.g., "Raydium", "Jupiter")
    Route       string    // Route aggregator if applicable
    PoolId      string    // Pool address (if available)
    Slot        uint64    // Solana slot number
    Timestamp   int64     // Unix timestamp
    Signature   string    // Transaction signature
    Idx         int       // Instruction index
    InnerIdx    int       // Inner instruction index
}
```

## TokenInfo

Token amount information.

```go
type TokenInfo struct {
    Mint     string  // Token mint address
    Amount   float64 // Token amount (human readable)
    Decimals uint8   // Token decimals
}
```

## TokenAmount

Raw token amount with precision.

```go
type TokenAmount struct {
    Amount   float64 // Human readable amount
    AmountRaw string // Raw amount string (full precision)
    Decimals uint8   // Token decimals
}
```

## PoolEvent

Liquidity pool operation.

```go
type PoolEvent struct {
    Type         string  // "CREATE", "ADD", "REMOVE"
    PoolId       string  // Pool address
    Token0Mint   string  // First token mint
    Token0Amount float64 // First token amount
    Token1Mint   string  // Second token mint
    Token1Amount float64 // Second token amount
    LpMint       string  // LP token mint
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

## MemeEvent

Meme platform event (Pumpfun, etc.).

```go
type MemeEvent struct {
    Type         string  // "CREATE", "BUY", "SELL", "COMPLETE", "MIGRATE"
    Protocol     string  // "Pumpfun", "RaydiumLaunchpad", etc.
    BaseMint     string  // Token mint address
    QuoteMint    string  // Quote token (usually SOL)
    BaseAmount   float64 // Token amount
    QuoteAmount  float64 // SOL amount
    User         string  // User wallet
    BondingCurve string  // Bonding curve address
    Pool         string  // Pool address (after migration)
    Slot         uint64  // Solana slot
    Timestamp    int64   // Unix timestamp
    Signature    string  // Transaction signature
    Idx          int     // Instruction index
    InnerIdx     int     // Inner instruction index
}
```

## TransferData

Token transfer information.

```go
type TransferData struct {
    Type      string  // "transfer", "transferChecked"
    Source    string  // Source token account
    Dest      string  // Destination token account
    Mint      string  // Token mint
    Amount    float64 // Transfer amount
    Authority string  // Transfer authority
    Slot      uint64  // Solana slot
    Timestamp int64   // Unix timestamp
    Signature string  // Transaction signature
    Idx       int     // Instruction index
    InnerIdx  int     // Inner instruction index
}
```

## ParseConfig

Parser configuration options.

```go
type ParseConfig struct {
    TryUnknownDEX    bool     // Try unknown DEX programs
    ProgramIds       []string // Filter to specific programs
    IgnoreProgramIds []string // Ignore specific programs
    ThrowError       bool     // Panic on errors
    AggregateTrades  bool     // Aggregate trades into one
}
```

## ParseResult

Result of parsing a transaction.

```go
type ParseResult struct {
    State          bool             // Success status
    Fee            TokenAmount      // Gas fee
    AggregateTrade *TradeInfo       // Combined trade
    Trades         []TradeInfo      // Individual trades
    Liquidities    []PoolEvent      // Pool events
    Transfers      []TransferData   // Transfers
    MemeEvents     []MemeEvent      // Meme events
    Slot           uint64           // Slot number
    Timestamp      int64            // Timestamp
    Signature      string           // Signature
    Signer         []string         // Signers
    Msg            string           // Error message
}
```
