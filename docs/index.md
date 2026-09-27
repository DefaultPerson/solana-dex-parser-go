# Solana DEX Parser (Go)

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/DefaultPerson/solana-dex-parser-go/blob/main/LICENSE)
[![Tests](https://github.com/DefaultPerson/solana-dex-parser-go/actions/workflows/test.yml/badge.svg)](https://github.com/DefaultPerson/solana-dex-parser-go/actions/workflows/test.yml)

A Go library that turns Solana transactions into trades, liquidity events, meme-launchpad events and transfers.

## Features

- **Trades** with exact raw amounts, pools, fees from protocol events, bot attribution and relay tips
- **Jupiter** v6 (including the `*_v2` routes), Jupiter Z, DCA, Limit Order and VA
- **AMMs and prop AMMs**: Raydium, Orca, Meteora, PumpSwap, HumidiFi, SolFi, GoonFi, BisonFi, TesseraV and more
- **Liquidity events** for Raydium, Orca, Meteora and PumpSwap pools
- **Meme launchpads**: Pump.fun, Raydium LaunchLab, Meteora DBC, Moonit, Heaven, Sugar, Boop.fun
- **Failed and version 1 transactions**, address lookup tables with pluggable fetchers
- **ShredParser** for pre-execution data (Jito ShredStream), converters for Yellowstone gRPC

The full protocol and bot tables are in the [README](https://github.com/DefaultPerson/solana-dex-parser-go#supported-protocols).

## Quick Install

```bash
go get github.com/DefaultPerson/solana-dex-parser-go
```

## Minimal Example

```go
result := dexparser.NewDexParser().ParseAll(tx, nil)

fmt.Printf("Trades: %d\n", len(result.Trades))
fmt.Printf("Liquidities: %d\n", len(result.Liquidities))
```

`tx` is a `*adapter.SolanaTransaction`: see [Getting Started](getting-started.md) for how to fetch and decode one.

## Documentation

- [Getting Started](getting-started.md): fetching transactions, configuration, result fields, gRPC
- [ShredParser](shred-parser.md): pre-execution parsing
- [Examples](examples/index.md): code for specific use cases
- [Development](development.md): contributing and testing

## License

MIT License, see [LICENSE](https://github.com/DefaultPerson/solana-dex-parser-go/blob/main/LICENSE) and [NOTICE](https://github.com/DefaultPerson/solana-dex-parser-go/blob/main/NOTICE).

## Acknowledgements

Ported from the TypeScript [solana-dex-parser](https://github.com/cxcx-ai/solana-dex-parser).
The relay tip account list comes from [sol-parser-sdk](https://github.com/0xfnzero/sol-parser-sdk); its IDLs and those of [solana-streamer](https://github.com/0xfnzero/solana-streamer) and [pump-public-docs](https://github.com/pump-fun/pump-public-docs) served as references.
