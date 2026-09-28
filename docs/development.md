# Development

## Prerequisites

- Go 1.22+ (the floor in go.mod; CI also tests the two current Go releases)
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

Tests run offline on real mainnet transactions stored in `testdata/tx/<signature>.json.gz` (`getTransaction` results).

```bash
go test ./... -race -count=1
```

A test that needs a transaction that is not stored yet can download it from the public RPC (or `SOLANA_RPC_URL`):

```bash
SDP_FETCH_FIXTURES=1 go test ./tests -run TestName
```

Fixtures are fetched with `"maxSupportedTransactionVersion": 1`.

`TestGolden` compares the `DexParser.ParseAll` and `ShredParser.ParseAll` output of every fixture with `testdata/golden` (canonical JSON, gzip).
After a change of output, check the reported diff, then rewrite the files that changed (this also adds the files of new fixtures):

```bash
go test ./tests -run TestGolden -update
```

### Documentation checks

- `example_test.go` holds the examples shown in README.md and docs/; `go test .` checks their output.
- `tests/docs_snippets_test.go` compiles every Go block of README.md and docs/, checks that the blocks bound to an example (`<!-- example: ExampleName -->`) are identical to it, and rejects constant-bound slicing such as `name[:8]`. Blocks marked `<!-- snippet: external -->` need extra modules and compile only with `SDP_DOCS_EXTERNAL=1` (network access; the gRPC example needs Go 1.24+, e.g. `GOTOOLCHAIN=go1.26.8`).
- `tests/docs_counts_test.go` checks the program, parser and bot counts in README.md against the code and prints the sentence to paste when they change.

### Benchmarks

```bash
go test ./tests -bench=. -benchmem
```

## Lint

CI runs these on the stable Go release:

```bash
gofmt -l .                     # must print nothing
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
```

staticcheck v0.8.1 (2026.2.1) and govulncheck v1.8.0 need Go 1.26 or newer.

## Project Structure

```
solana-dex-parser-go/
├── dex_parser.go          # DexParser: parse flow and parser registration
├── shred_parser.go        # ShredParser (pre-execution) and the Pump.fun/PumpSwap decoders
├── shred_entries.go       # DecodeShredEntries (Jito ShredStream)
├── grpc_utils.go          # Yellowstone gRPC converters
├── example_test.go        # examples shown in the documentation
├── adapter/               # TransactionAdapter: RPC JSON, jsonParsed, v0/v1, lookup tables
├── classifier/            # instruction classification by program
├── constants/             # program IDs, discriminators, tokens, bots, tip accounts
├── types/                 # result types and ParseConfig
├── utils/                 # transfers, trade helpers, bot detection, log decoding
├── parsers/
│   ├── jupiter/           # Jupiter v6, DCA, VA, Limit Order v1/v2, Jupiter Z
│   ├── raydium/           # V4, CPMM, CLMM, Route, LaunchLab
│   ├── meteora/           # DLMM, DAMM v1/v2, DBC
│   ├── orca/              # Whirlpool
│   ├── pumpfun/           # Pump.fun, PumpSwap
│   ├── meme/              # Moonit, Heaven, Sugar, Boop.fun
│   ├── propamm/           # prop AMMs, Titan and OKX V2 route events
│   ├── dflow/, photon/    # aggregators
│   ├── systoken/          # System and Token transfers (ShredParser)
│   └── alt/               # Address Lookup Table events
├── testdata/tx/           # stored mainnet transactions
├── testdata/golden/       # expected ParseAll output per stored transaction (TestGolden)
└── tests/                 # tests (package tests)
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes and add tests on real transactions
4. Run `gofmt -l .`, `go vet ./...` and `go test ./... -race -count=1`
5. Commit with a Conventional Commits message: `git commit -m "feat: add my feature"`
6. Push and open a Pull Request

## Code Style

- Follow standard Go conventions and `gofmt`
- Raw amounts are exact integer strings; never round-trip them through float64
- Take fees only from protocol events or explicit fee transfers
- Keep output deterministic: sort by numeric `Idx`, never iterate maps into results unsorted
