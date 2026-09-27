# ShredParser

`ShredParser` decodes DEX instructions from their arguments, without relying on execution results.
It is meant for data that arrives before execution, where `DexParser` has nothing to work with.

## Input sources

| Source | What the transaction contains | What ShredParser sees |
|--------|-------------------------------|-----------------------|
| Jito ShredStream entries (`DecodeShredEntries`), any transaction without meta | message only, no meta | outer instructions only: CPI calls made by bots, routers and aggregators are invisible |
| Yellowstone gRPC transaction updates (`ConvertGeyserTransaction`), RPC results | executed transaction with meta | outer and inner instructions; `DexParser` gives the executed amounts for the same data |

`DecodeShredEntries` decodes the `entries` bytes of a ShredStream `Entry` message (a bincode `Vec<Entry>`) into transactions (legacy, v0 and v1).
They have no meta and `Slot` 0: set it from `Entry.slot`.
A malformed transaction stops decoding; the transactions before it are returned with the error.

```go
// data is Entry.entries of a SubscribeEntries message, slot is Entry.slot
var data []byte
var slot uint64

txs, err := dexparser.DecodeShredEntries(data)
if err != nil {
	log.Println("partial entry:", err)
}
for _, tx := range txs {
	tx.Slot = slot
	result := shredParser.ParseAll(tx, nil)
	for _, ins := range result.ParsedInstructions {
		fmt.Println(result.Signature, ins.Idx, ins.ProgramName, ins.Action)
	}
}
```

## Example

<!-- example: ExampleShredParser_ParseAll -->
```go
tx := loadTransaction(txFile)
// Pre-execution sources (e.g. Jito ShredStream, see DecodeShredEntries)
// deliver no meta; drop it to see what ShredParser gets from them.
tx.Meta = nil

result := dexparser.NewShredParser().ParseAll(tx, nil)
fmt.Println("status:", result.TxStatus)
for _, ins := range result.ParsedInstructions {
	fmt.Printf("%s %s %s", ins.Idx, ins.ProgramName, ins.Action)
	if t := ins.Trade; t != nil {
		fmt.Printf(": %s %s (%s) -> %s (%s)", t.Type,
			t.InputToken.AmountRaw, ins.InputAmountKind,
			t.OutputToken.AmountRaw, ins.OutputAmountKind)
	}
	fmt.Println()
}
```

Output:

```text
status: unknown
1 System transfer
3 Pumpfun create
5 Pumpfun buy: BUY 2020000000 (max) -> 67062499999999 (exact)
```

The buy asks for exactly 67062499999999 tokens and allows at most 2020000000 lamports.
`DexParser` reports what the buyer paid in the same transaction: 2020000000, that is 2000000000 into the bonding curve plus the 1% fee of 20000000 transferred to the fee recipient `CebN5WGQ…`, which is the trade's `Fee`.
`loadTransaction` stands for the file reading of the [Quick Start](getting-started.md#quick-start); `tx` is the Pump.fun create + buy `4Cod1cNG…`.

## Result

`types.ParseShredResult`:

| Field | Content |
|-------|---------|
| `ParsedInstructions` | typed instructions of every program, in execution order |
| `Instructions` | the older per-program format: program name (`"Pumpfun"`, `"Jupiter"`, `"System"`, `"Token"`, …) -> decoded events. Programs without decoded instructions are left out |
| `TxStatus` | `unknown` without meta, else `success` or `failed` |
| `HasUnresolvedAccounts` | some lookup-table accounts could not be resolved |
| `State`, `Msg`, `Signature`, `Slot`, `Timestamp`, `Signer` | as in `ParseResult` |

`types.ParsedShredInstruction` has `ProgramID`, `ProgramName`, `Action` (the instruction name), `Idx`, `Accounts`, `Data` (the decoded arguments) and one of `Trade`, `Liquidity`, `Transfer` or `MemeEvent`.

## Semantics

- **Amounts are instruction arguments**, often slippage limits. `InputAmountKind` and `OutputAmountKind` tell what the trade's (or meme event's) amounts mean:

  | Kind | Meaning |
  |------|---------|
  | `exact` | the exact amount sent or received |
  | `max` | a slippage limit: the most the user may send |
  | `min` | a slippage limit: the least the user accepts |
  | `quote` | the router's expected amount; the limit follows from `SlippageBps` |
  | `unknown` | known only after execution; reported as 0 |

- **Mints are never guessed.** A mint the transaction does not reveal is `""`, with decimals 0.
- **Decimals** come from the transaction's token balances or `constants.TOKEN_DECIMALS`, else 0 (unknown). Pump.fun bonding-curve tokens always have 6.
- **Trade type** is `SWAP` when the direction cannot be determined: for Jupiter and DFlow routes whenever a mint is unknown or both are the same (circular arbitrage). Raydium V4 trades with one known mint are `BUY` for WSOL in and `SELL` for WSOL out.
- **Lookup tables**: v0 accounts resolve from `meta.loadedAddresses`, `ParseConfig.AddressLookupTables` or an `ALTsFetcher` (see [Getting Started](getting-started.md#address-lookup-tables-and-fetchers)). Unresolved accounts are `""`, and `UnresolvedAccounts` is set on the instruction (and `unresolvedAccounts` on the older events).
- **Failed transactions** (meta.err set) give no instructions and `Msg="transaction failed"` unless `IncludeFailedTxs` is set.
- **Idx** is `"N"` for outer and `"N-M"` for inner instructions.
- A nil config means `TryUnknownDEX` on; `ProgramIds` and `IgnoreProgramIds` select programs. `ShredParser.ParseAll` never returns nil.

## Protocols

| Protocol | Decoded instructions | Typed output |
|----------|---------------------|--------------|
| **Pump.fun** | buy, sell, buy_exact_sol_in, buy_v2, sell_v2, buy_exact_quote_in_v2, create, create_v2, migrate, migrate_v2 | Trade + MemeEvent; MemeEvent for create and migrate |
| **PumpSwap** | buy, sell, buy_exact_quote_in, create_pool, deposit, withdraw | Trade; PoolEvent |
| **Jupiter v6** | route, route_with_token_ledger, exact_out_route, the shared_accounts_* routes and the `*_v2` routes | Trade (`ExactOut`, `PlatformFeeBps`, slippage in `Data`) |
| **Raydium V4** | swap (tags 9, 11, 16, 17), initialize2, deposit, withdraw | Trade; PoolEvent |
| **Raydium LaunchLab** | buy/sell exact in and out, initialize (v2, Token-2022), migrate_to_amm, migrate_to_cpswap | MemeEvent |
| **Meteora DBC** | swap, swap2 (with transfer hook), pool initializers, DAMM migrations | MemeEvent |
| **DFlow** | swap, swap2 and their `_with_destination` and `_native` forms | Trade |
| **Photon** | Pump.fun (v1, v2), PumpSwap and Moonit swaps, collect_fee | Trade |
| **System, Token, Token-2022** | transfers | Transfer |

## gRPC input

Yellowstone gRPC updates are executed transactions: convert them with `ConvertGeyserTransaction` (see [Getting Started](getting-started.md#streaming-with-yellowstone-grpc)).
For them `DexParser` gives executed amounts; `ShredParser` is useful when you want the signed limits.
An update without meta (`YellowstoneTransaction.NoMeta`, deshred transactions) converts to a transaction without meta, as ShredParser expects for pre-execution input.
