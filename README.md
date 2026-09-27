# Solana DEX Parser (Go)

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Tests](https://github.com/DefaultPerson/solana-dex-parser-go/actions/workflows/test.yml/badge.svg)](https://github.com/DefaultPerson/solana-dex-parser-go/actions/workflows/test.yml)
[![Docs](https://img.shields.io/badge/docs-GitHub%20Pages-blue.svg)](https://defaultperson.github.io/solana-dex-parser-go/)

A Go library that turns Solana transactions into trades, liquidity events, meme-launchpad events and transfers.
It started as a port of the TypeScript [solana-dex-parser](https://github.com/cxcx-ai/solana-dex-parser) and has since diverged (see [Differences from the TypeScript library](#differences-from-the-typescript-library)).

It knows **150 DEX program IDs** (63 of them from the Jupiter venue label list and 21 trading bot programs), registers default parsers for **39 trade**, **8 liquidity**, **4 transfer** and **8 meme event** programs, attributes trades to **16 trading bots** through 72 fee accounts, and reports tips paid to 150 accounts of 17 transaction-landing providers.

These numbers are computed from the code by `TestDocsReadmeCounts` (tests/docs_counts_test.go).

## Installation

```bash
go get github.com/DefaultPerson/solana-dex-parser-go
```

Requires Go 1.22 or newer.

## Quick Start

Parse the Pump.fun create + buy `4Cod1cNG…` (a copy is in testdata/example):

<!-- example: ExampleDexParser_ParseAll -->
```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
)

const txFile = "tx.json"

func main() {
	// txFile holds the "result" of a getTransaction RPC call made with
	// {"encoding": "json", "maxSupportedTransactionVersion": 1}
	raw, err := os.ReadFile(txFile)
	if err != nil {
		log.Fatal(err)
	}
	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(raw, &tx); err != nil {
		log.Fatal(err)
	}

	parser := dexparser.NewDexParser()
	result := parser.ParseAll(&tx, nil) // nil config: parse everything
	if !result.State {
		log.Fatal(result.Msg)
	}

	fmt.Println("status:", result.TxStatus, "fee:", result.Fee.Amount)
	for _, trade := range result.Trades {
		fmt.Printf("trade: %s %s %s %s -> %s %s\n", trade.AMM, trade.Type,
			trade.InputToken.AmountRaw, trade.InputToken.Mint,
			trade.OutputToken.AmountRaw, trade.OutputToken.Mint)
	}
	for _, event := range result.MemeEvents {
		fmt.Printf("meme: %s %s %s\n", event.Protocol, event.Type, event.BaseMint)
	}
	if result.Tip != nil {
		fmt.Println("tip:", result.Tip.Amount)
	}
}
```

Output:

```text
status: success fee: 80285
trade: Pumpfun BUY 2000000000 So11111111111111111111111111111111111111112 -> 67062499999999 B9Z9mKUoVy5k8KuL2HauUD1mhmfF3PPNnJoK83S1pump
meme: Pumpfun CREATE B9Z9mKUoVy5k8KuL2HauUD1mhmfF3PPNnJoK83S1pump
meme: Pumpfun BUY B9Z9mKUoVy5k8KuL2HauUD1mhmfF3PPNnJoK83S1pump
tip: 4000000
```

The same code runs as `ExampleDexParser_ParseAll` in example_test.go.
Fetch transactions with `"encoding": "json"` and `"maxSupportedTransactionVersion": 1`: with 0 the RPC rejects version 1 transactions, which are live on mainnet.
`jsonParsed` input is accepted too.

## Features

- **Trades** with exact raw amounts (`AmountRaw` strings), decimals from the transaction, pool addresses, and fees taken only from protocol events or explicit fee transfers.
- **Aggregated trade** (`AggregateTrade`) of multi-hop and routed swaps, computed in addition to the individual trades.
- **Jupiter**: v6 routes including the `*_v2` family (SwapsEvent, FeeEvent platform fees), Jupiter Z (RFQ) fills, DCA (Recurring), Limit Order v1 (historical flash fills) and v2 (Trigger), and Value Average (legacy).
- **Pump.fun and PumpSwap**: v2 instructions, non-SOL quote mints (USDC and custom quotes), protocol, creator, buyback and cashback fees.
- **Prop AMMs**: SolFi (V1, V2), GoonFi (V1, V2), HumidiFi, Obric V2, BisonFi, TesseraV, AlphaQ, ZeroFi, Scorch, Quantum, Manifest, Byreal and Saros DLMM.
- **Liquidity events** for Raydium V4/CPMM/CLMM, Orca Whirlpool, Meteora DLMM/DAMM v1/DAMM v2 and PumpSwap, amounts from program events or `ray_log` where available.
- **Meme launchpad events** (create, buy, sell, migrate, complete) for Pump.fun, PumpSwap, Raydium LaunchLab, Meteora DBC, Moonit, Heaven, Sugar and Boop.fun.
- **Bots and tips**: `TradeInfo.Bot` from fee payments to known bot wallets, bot router programs as `Route`, and `ParseResult.Tip` for Jito and other landing services.
- **Failed transactions** are recognised: by default only fee, signer, balance changes and `TxStatus` are reported (`ParseConfig.IncludeFailedTxs` parses them fully).
- **Version 1 transactions**, v0 address lookup tables (`meta.loadedAddresses`, `ParseConfig.AddressLookupTables` or an `ALTsFetcher`), Address Lookup Table program events.
- **ShredParser** for pre-execution data: decodes instruction arguments of Pump.fun, PumpSwap, Jupiter, Raydium V4, LaunchLab, Meteora DBC, DFlow, Photon and System/Token transfers.
- **Stream input**: `ConvertGeyserTransaction` / `ConvertYellowstoneTransaction` for Yellowstone gRPC (Helius LaserStream, Triton) and `DecodeShredEntries` for Jito ShredStream entries.
- **Robust**: `ParseAll` never returns nil and never panics on malformed input unless `ThrowError` is set; output is deterministic and in execution order.

## Parsing semantics

| Setting | Behaviour |
|---------|-----------|
| `nil` config | `types.DefaultParseConfig()`: everything, including `AggregateTrade`, with `TryUnknownDEX` on |
| `&types.ParseConfig{}` (zero value) | everything except `AggregateTrade`; `TryUnknownDEX` off. `AggregateTrades: true` adds the aggregate |
| `ParseType` set | only the listed kinds; aggregation is `ParseType.AggregateTrade \|\| AggregateTrades`. `ParseType{AggregateTrade: true}` alone returns only the aggregate |
| `ProgramIds` / `IgnoreProgramIds` | restrict every parser, Jupiter included. A transaction without any of `ProgramIds` gives `State=false`, `Msg="No matching program ids"` |
| failed transaction (`meta.err` set) | no trades, liquidity, meme, ALT events or transfers; `State=true`, `Msg="transaction failed"`, `TxStatus=failed`. `IncludeFailedTxs: true` parses them anyway |

- `Trades` always holds the individual trades when trades are parsed; `ParseTrades` never comes back empty because of aggregation.
- When any Jupiter program runs in the transaction, its trades are authoritative for the instructions it covers and nested AMM trades are not repeated. Liquidity, meme events and transfers are still parsed for the whole transaction.
- `Transfers` is filled only when the transaction has no trades and no liquidity events (DCA, VA and limit-order programs report their own deposits and withdrawals).
- `Idx` is `"N"` for an outer instruction and `"N-M"` for inner instruction M of outer instruction N; every list is sorted numerically by it.
- With `TryUnknownDEX` (on for a nil config), programs without a dedicated parser, known or not, are parsed from their transfers when one leg is SOL or a stablecoin; their AMM is the program name or `"Unknown"`.
- `ParseResult.Warnings` lists what made a result incomplete (fetcher errors, unresolved lookup-table accounts); `State` is not affected.

Full reference: [Getting Started](https://defaultperson.github.io/solana-dex-parser-go/getting-started/).

## Supported protocols

✅ parsed by a dedicated parser, ❌ not parsed, ➖ not applicable. "Known" programs have no dedicated parser: they name the route or AMM of a trade and, with `TryUnknownDEX`, are parsed from their transfers (see above).
Legacy entries are kept so historical transactions still parse.

### Aggregators and routers

| Protocol | Trades | Transfers | Notes |
|----------|--------|-----------|-------|
| **Jupiter v6** | ✅ | ➖ | route, shared_accounts, exact_out and `*_v2` routes; one trade per hop, `AMM` = hop venue, `User` = the route's token owner |
| **Jupiter Z** | ✅ | ➖ | RFQ fill as the taker's trade |
| **Jupiter DCA (Recurring)** | ✅ | ✅ | fills; open and close as transfers |
| **Jupiter Limit Order v2 (Trigger)** | ✅ | ✅ | fills as the maker's trade with the actual fee |
| **Jupiter Limit Order v1** | ✅ | ✅ | legacy: historical flash fills |
| **Jupiter VA** | ✅ | ✅ | legacy (last activity 2026-07) |
| **DFlow** | ✅ | ➖ | |
| **Raydium Route** | ✅ | ➖ | |
| **Photon** | Known | ➖ | trades come from the venue parsers with `Route = Photon`; decoded by ShredParser |
| **OKX DEX Router V2**, **Titan** | Known | ➖ | hop trades come from the venue parsers; `propamm.OKXV2Parser` and `propamm.TitanParser` decode the routers' swap events but are not registered by default |
| **OKX DEX (V1)** | Known | ➖ | legacy |
| **Sanctum** | Known | ➖ | |
| **Jupiter V2, V4** | Known | ➖ | V2 legacy |

### AMMs

| Protocol | Trades | Liquidity | Notes |
|----------|--------|-----------|-------|
| **Raydium V4** (+ RaydiumAMM `5quB…`) | ✅ | ✅ (V4) | amounts from `ray_log` |
| **Raydium CPMM** | ✅ | ✅ | |
| **Raydium CLMM** | ✅ | ✅ | limit-order instructions are not parsed |
| **Orca Whirlpool** | ✅ | ✅ | including `*_v2` and two-hop swaps |
| **Meteora DLMM** | ✅ | ✅ | including pair creation and rebalance_liquidity; limit orders are not parsed |
| **Meteora DAMM v1 (Pools)** | ✅ | ✅ | |
| **Meteora DAMM v2** | ✅ | ✅ | |
| **PumpSwap** | ✅ | ✅ | |
| **Orca V1, V2**, **Phoenix**, **OpenBook** `opnb…`, **Serum V3**, **Stabble**, **Saber**, **Saros**, **Mercurial**, **Crema**, **Aldrin**, **GooseFX GAMMA**, **1Dex** | Known | ❌ | |
| **Aldrin V2**, **Lifinity**, **Lifinity V2** | Known | ❌ | legacy |
| 63 venues of the Jupiter label list (Bonkswap, DefiTuna, Obsidian, PancakeSwap, Perena, Stabble CLMM, Woofi, …) | Known | ❌ | `constants.JUPITER_LABEL_PROGRAMS` |

### Prop AMMs

| Protocol | Trades | Notes |
|----------|--------|-------|
| **HumidiFi** | ✅ | |
| **SolFi V2**, **GoonFi V2**, **BisonFi**, **TesseraV**, **AlphaQ**, **ZeroFi**, **Scorch**, **Quantum**, **Manifest**, **Byreal**, **Saros DLMM** | ✅ | |
| **SolFi (V1)** | ✅ | legacy: not seen executing recently |
| **GoonFi (V1)** | ✅ | legacy: vaults closed 2026-02 |
| **Obric V2** | ✅ | legacy: last swap found 2024-11 |

### Meme launchpads

| Protocol | Trades | Create | Migrate | Notes |
|----------|--------|--------|---------|-------|
| **Pump.fun** | ✅ | ✅ | ✅ | v2 instructions, non-SOL quotes, curve COMPLETE events |
| **PumpSwap** | ✅ | ✅ (pool) | ➖ | also BUY/SELL meme events and BUY_AND_BURN (boost_buy_and_burn) |
| **Raydium LaunchLab** | ✅ | ✅ | ✅ | also used by LetsBonk.fun; USD1 and other quote mints |
| **Meteora DBC** | ✅ | ✅ | ✅ | curve COMPLETE events |
| **Moonit** | ✅ | ✅ | ✅ | |
| **Heaven** | ✅ | ✅ | ➖ | pools launch directly, there is no migration |
| **Sugar** | ✅ | ✅ | ✅ | migrates to Raydium CPMM with a pSOL quote; nearly idle |
| **Boop.fun** | ✅ | ✅ | ✅ (COMPLETE) | the curve COMPLETE event marks the graduation |

### Trading bots

`TradeInfo.Bot` is set when a bot's fee account (`constants.BOT_FEE_ACCOUNTS`), or a token account it owns, receives in the transaction at least 10000 lamports of SOL/WSOL (or 0.1% of the trade's SOL leg if that is smaller), or at least 0.1% of a trade leg in that leg's token. The mere presence of a fee account never counts, and the trader's own accounts are never fee receivers.

| Bot | Fee accounts | Bot programs (`Route`) |
|-----|--------------|------------------------|
| **Axiom** | 20 | FLASHX8D…, BLUR9cL8… (legacy) |
| **BananaGun** | 11 | BANANAjs… |
| **GMGN** | 9 | GMgnVFR8…, GMGNreQc…, DGMgNKpq… (legacy) |
| **Trojan** | 7 | troyXT7T…, TroYL71c…, troY36Yi… |
| **STBot** | 7 | Stbot61L… |
| **MevX** | 3 | MevxQ9iQ… |
| **BONKbot** | 2 | CxvksNjw…, BBRouter1… |
| **Maestro** | 2 | MaestroA… |
| **Nova** | 2 | BujKR6sa…, NoVA1TmD… (legacy) |
| **Padre** | 2 | term9YPb…, 9Fox6i7o… (legacy) |
| **BullX** | 2 | |
| **Bloom** | 1 | b1oomGGq… |
| **Fomo** | 1 | |
| **PepeBoost** | 1 | |
| **Photon** | 1 | |
| **Raybot** | 1 | |
| **Mintech**, **Apepro** (legacy) | – | program only |

Bot programs are listed in `constants.DEX_PROGRAMS` (tag `bot`) and `constants.BOT_ROUTER_PROGRAMS`; trades executed through them report the bot as `Route`.
Tips paid by System transfers to `constants.TIP_ACCOUNTS` (Jito verified; Helius, Temporal, NextBlock, bloXroute and 12 more from sol-parser-sdk, unverified) are summed in `ParseResult.Tip` and never counted as a trade fee.

## ShredParser

`ShredParser` decodes DEX instructions from their arguments, without execution results:

```go
result := shredParser.ParseAll(tx, nil)
for _, ins := range result.ParsedInstructions {
	fmt.Println(ins.Idx, ins.ProgramName, ins.Action, ins.InputAmountKind, ins.OutputAmountKind)
}
```

- **Input**: pre-execution transactions (Jito ShredStream entries decoded with `DecodeShredEntries`, any transaction without meta) show only outer instructions, so CPI calls made by bots and routers are invisible. Yellowstone gRPC updates are executed transactions with meta; ShredParser then decodes inner instructions too, but `DexParser` gives executed amounts for them.
- **Amounts** are what the signer asked for: often slippage limits. `ParsedShredInstruction.InputAmountKind` / `OutputAmountKind` say which: `exact`, `max`, `min`, `quote` or `unknown` (reported as 0).
- **Mints and decimals** are never guessed: an unknown mint is `""` with decimals 0, and the trade type is `SWAP` when the direction cannot be told. Decimals come from the transaction or `constants.TOKEN_DECIMALS` (Pump.fun curve tokens: 6).
- **Lookup tables**: v0 accounts resolve from meta, `ParseConfig.AddressLookupTables` or an `ALTsFetcher`; otherwise they are `""` and `HasUnresolvedAccounts` / `UnresolvedAccounts` are set.
- Failed transactions are skipped unless `IncludeFailedTxs` is set; `TxStatus` is `unknown` without meta.

| Protocol | Decoded instructions | Typed output |
|----------|---------------------|--------------|
| **Pump.fun** | buy, sell, buy_exact_sol_in, `*_v2`, buy_exact_quote_in_v2, create, create_v2, migrate, migrate_v2 | Trade + MemeEvent, MemeEvent |
| **PumpSwap** | buy, sell, buy_exact_quote_in, create_pool, deposit, withdraw | Trade, PoolEvent |
| **Jupiter v6** | route, route_with_token_ledger, exact_out_route, shared_accounts_* and `*_v2` routes | Trade |
| **Raydium V4** | swaps (tags 9, 11, 16, 17), initialize2, deposit, withdraw | Trade, PoolEvent |
| **Raydium LaunchLab** | buy/sell exact in/out, initialize (v2, Token-2022), migrate_to_amm, migrate_to_cpswap | MemeEvent |
| **Meteora DBC** | swap, swap2 (with transfer hook), pool initializers, DAMM migrations | MemeEvent |
| **DFlow** | swap, swap2 and the `*_with_destination(_native)` forms | Trade |
| **Photon** | Pump.fun (v2), PumpSwap and Moonit swaps, collect_fee | Trade |
| **System, Token, Token-2022** | transfers | Transfer |

See [ShredParser](https://defaultperson.github.io/solana-dex-parser-go/shred-parser/) for the full description and the gRPC and ShredStream examples.

## Differences from the TypeScript library

- Failed transactions are not parsed into trades by default (`IncludeFailedTxs`).
- Jupiter-routed transactions keep their liquidity, meme events and transfers; the TypeScript parser returns after the Jupiter trades.
- Trade fees come only from protocol events and explicit fee transfers; the "output minus balance change" fee estimate is not used.
- Outer-instruction `Idx` is `"N"` (not `"N-0"`) and all lists are sorted numerically by `Idx`.
- Go-only parsers and fields: PumpSwap meme events, Heaven and Sugar trades, the prop AMM and Jupiter Z parsers, `TradeInfo.Bot`, `ParseResult.Tip` and `Warnings`, `ShredParser` typed output and amount kinds.
- The TypeScript `PUMP_FUN` liquidity parser registration has no counterpart.

## Documentation

- [Getting Started](https://defaultperson.github.io/solana-dex-parser-go/getting-started/): fetching transactions, configuration, result fields, gRPC
- [ShredParser](https://defaultperson.github.io/solana-dex-parser-go/shred-parser/): pre-execution parsing
- [Examples](https://defaultperson.github.io/solana-dex-parser-go/examples/)
- [Development](https://defaultperson.github.io/solana-dex-parser-go/development/)
- [CHANGELOG](CHANGELOG.md)

## License

MIT License, see [LICENSE](LICENSE). Third-party notices: [NOTICE](NOTICE).

## Acknowledgements

- [solana-dex-parser](https://github.com/cxcx-ai/solana-dex-parser) (MIT, Copyright (c) 2025 Caspod): the TypeScript library this one was ported from.
- [sol-parser-sdk](https://github.com/0xfnzero/sol-parser-sdk) and [solana-streamer](https://github.com/0xfnzero/solana-streamer) (MIT, Copyright (c) 2024 MiracleAI-Labs): the relay tip account list in `constants/tips.go` comes from sol-parser-sdk, and their IDLs and decoders served as a reference.
- [pump-fun/pump-public-docs](https://github.com/pump-fun/pump-public-docs): Pump.fun and PumpSwap IDLs used as the reference for the event layouts.
