# Development

## Prerequisites

- Go 1.21+
- Git

## Setup

```bash
git clone https://github.com/DefaultPerson/solana-dex-parser-go.git
cd solana-dex-parser-go
go mod download
```

## Build

```bash
go build ./...
```

## Testing

### Run all tests

```bash
go test ./... -v
```

### Run integration tests

Integration tests use real Solana transactions via Helius RPC:

```bash
# Set environment variable
export HELIUS_API_KEY=your-api-key

# Run tests
go test ./tests -v -run TestIntegration
```

### Run benchmarks

```bash
go test ./tests -bench=. -benchmem
```

## Project Structure

```
solana-dex-parser-go/
├── dex_parser.go          # Main DexParser
├── shred_parser.go        # ShredParser for gRPC
├── types/
│   ├── trade.go           # TradeInfo, TokenInfo
│   ├── pool.go            # PoolEvent
│   ├── meme.go            # MemeEvent
│   └── common.go          # ParseResult, ClassifiedInstruction
├── constants/
│   ├── programs.go        # DEX program IDs
│   ├── discriminators.go  # Instruction discriminators
│   └── tokens.go          # Token constants
├── adapter/
│   └── transaction.go     # TransactionAdapter
├── classifier/
│   └── instruction.go     # InstructionClassifier
├── utils/
│   ├── utils.go           # Helper functions
│   ├── binary_reader.go   # Binary data parsing
│   └── transaction_utils.go
├── parsers/
│   ├── jupiter/           # Jupiter parsers
│   ├── raydium/           # Raydium parsers
│   ├── meteora/           # Meteora parsers
│   ├── orca/              # Orca parsers
│   ├── pumpfun/           # Pumpfun parsers
│   └── meme/              # Meme platform parsers
└── tests/
    ├── integration_test.go
    └── benchmark_test.go
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests: `go test ./...`
5. Commit: `git commit -m "feat: add my feature"`
6. Push: `git push origin feature/my-feature`
7. Open a Pull Request

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add tests for new functionality
- Keep functions focused and small
