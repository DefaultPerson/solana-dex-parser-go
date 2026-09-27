# Getting Started

## Prerequisites

- Go 1.22 or higher

## Installation

```bash
go get github.com/DefaultPerson/solana-dex-parser-go
```

## Quick Start

The parser takes the `result` of a `getTransaction` RPC call, decoded into `adapter.SolanaTransaction`.
This program parses the Pump.fun create + buy `4Cod1cNG…` (a copy is in testdata/example of the repository):

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
trade: Pumpfun BUY 2020000000 So11111111111111111111111111111111111111112 -> 67062499999999 B9Z9mKUoVy5k8KuL2HauUD1mhmfF3PPNnJoK83S1pump
meme: Pumpfun CREATE B9Z9mKUoVy5k8KuL2HauUD1mhmfF3PPNnJoK83S1pump
meme: Pumpfun BUY B9Z9mKUoVy5k8KuL2HauUD1mhmfF3PPNnJoK83S1pump
tip: 4000000
```

## Fetching transactions

Request `"encoding": "json"` and `"maxSupportedTransactionVersion": 1`.
Version 1 transactions are live on mainnet, and an RPC call made with `maxSupportedTransactionVersion` 0 fails on them with error -32015.
`jsonParsed` results parse as well.

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
)

// getTransaction fetches a confirmed transaction from a Solana RPC node
func getTransaction(rpcURL, signature string) (*adapter.SolanaTransaction, error) {
	request, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getTransaction",
		"params": []interface{}{signature, map[string]interface{}{
			"encoding":                       "json",
			"maxSupportedTransactionVersion": 1,
			"commitment":                     "confirmed",
		}},
	})
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(request))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body struct {
		Result *adapter.SolanaTransaction `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", body.Error.Code, body.Error.Message)
	}
	if body.Result == nil {
		return nil, fmt.Errorf("transaction %s not found", signature)
	}
	return body.Result, nil
}

func main() {
	signature := "4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz"
	tx, err := getTransaction("https://api.mainnet-beta.solana.com", signature)
	if err != nil {
		log.Fatal(err)
	}

	result := dexparser.NewDexParser().ParseAll(tx, nil)
	fmt.Printf("trades: %d, liquidities: %d, meme events: %d, transfers: %d\n",
		len(result.Trades), len(result.Liquidities), len(result.MemeEvents), len(result.Transfers))
}
```

## Parser methods

| Method | Returns |
|--------|---------|
| `ParseAll(tx, config)` | `*types.ParseResult` with everything `config` asks for. Never nil |
| `ParseTrades(tx, config)` | the individual trades (`[]types.TradeInfo`) |
| `ParseLiquidity(tx, config)` | the liquidity events (`[]types.PoolEvent`) |
| `ParseTransfers(tx, config)` | the transfers (`[]types.TransferData`) |
| `ParseBatch(txs, config, maxWorkers)` | one result per transaction, in input order; `maxWorkers > 1` parses concurrently |
| `ParseBatchWithCallback(txs, config, maxWorkers, callback)` | as `ParseBatch`, calling `callback` per result; returning false stops early |
| `RegisterTradeParser`, `RegisterLiquidityParser`, `RegisterTransferParser`, `RegisterMemeEventParser` | add or replace the parser of a program ID |

All methods are nil-safe: a nil transaction gives `State=false`, `Msg="nil transaction"`.
A panic inside a parser is recovered into `State=false` and a `Msg` with the signature, unless `ThrowError` is set.

## Configuration

`types.ParseConfig` fields:

| Field | Zero value | Meaning |
|-------|-----------|---------|
| `ParseType` | unset: parse everything except the aggregate | which result kinds to produce (see below) |
| `TryUnknownDEX` | false | parse programs without a parser from their transfers (a SOL or stablecoin leg is required) |
| `ProgramIds` | all | only these programs are parsed; a transaction with none of them gives `State=false`, `Msg="No matching program ids"` |
| `IgnoreProgramIds` | none | these programs are skipped by every parser, Jupiter included |
| `AccountInclude` | none | skip the transaction (`State=false`) unless one of these accounts is in it |
| `AccountExclude` | none | skip the transaction (`State=false`) if one of these accounts is in it |
| `ThrowError` | false | re-panic instead of returning `State=false` |
| `AggregateTrades` | false | deprecated switch for the aggregate; with `ParseType` unset it is the only one, otherwise it is OR-ed with `ParseType.AggregateTrade` |
| `IncludeFailedTxs` | false | parse failed transactions like successful ones |
| `AddressLookupTables` | none | lookup table address -> its addresses, for v0 transactions without `meta.loadedAddresses` |
| `ALTsFetcher` | nil | resolves the remaining lookup tables of such transactions |
| `TokenAccountsFetcher` | nil | resolves mint, owner and decimals of token accounts the transaction does not describe |
| `PoolInfoFetcher` | nil | deprecated, has no effect |

A nil config means `types.DefaultParseConfig()`: every kind including the aggregate, with `TryUnknownDEX` on.
The zero value `&types.ParseConfig{}` is different: `TryUnknownDEX` is off and there is no aggregate.

`ParseType` fields are `AggregateTrade`, `Trade`, `Liquidity`, `Transfer`, `MemeEvent` and `AltEvent`; presets are `types.ParseAll()`, `types.ParseTradesOnly()` and `types.ParseLiquidityOnly()`.
When any field is set only those kinds are produced. `Trades` always holds the individual trades when `Trade` is set, and `AggregateTrade` is computed in addition.
`ParseType{AggregateTrade: true}` alone parses the trades but returns only the aggregate.

<!-- example: ExampleDexParser_ParseAll_config -->
```go
tx := loadTransaction(txFile)

config := &types.ParseConfig{
	// Trades and the aggregated trade only
	ParseType: types.ParseType{Trade: true, AggregateTrade: true},
	// Only transactions and programs among these
	ProgramIds:    []string{constants.DEX_PROGRAMS.PUMP_FUN.ID},
	TryUnknownDEX: true,
}
result := dexparser.NewDexParser().ParseAll(tx, config)

fmt.Println("trades:", len(result.Trades), "meme events:", len(result.MemeEvents))
if agg := result.AggregateTrade; agg != nil {
	fmt.Println("aggregate:", agg.Type, agg.InputToken.AmountRaw, "->", agg.OutputToken.AmountRaw)
}
```

Output:

```text
trades: 1 meme events: 0
aggregate: BUY 2020000000 -> 67062499999999
```

`loadTransaction` stands for the file reading of the Quick Start.

## The result

`types.ParseResult`:

| Field | Content |
|-------|---------|
| `State`, `Msg` | false with a message when the transaction could not be parsed or was filtered out |
| `TxStatus` | `success`, `failed`, or `unknown` without meta |
| `Signature`, `Slot`, `Timestamp`, `Signer` | transaction context |
| `Fee` | the network fee (lamports, 9 decimals) |
| `ComputeUnits` | compute units consumed (from meta, for every transaction version) |
| `Trades` | individual trades, in execution order |
| `AggregateTrade` | the trade from the first input to the last output of all trades (a copy) |
| `Liquidities` | liquidity events: `CREATE`, `ADD`, `REMOVE` |
| `MemeEvents` | launchpad events: `CREATE`, `BUY`, `SELL`, `MIGRATE`, `COMPLETE`, `BUY_AND_BURN` |
| `Transfers` | token transfers; only when there are no trades and no liquidity events |
| `AltEvents` | Address Lookup Table program events |
| `SolBalanceChange`, `TokenBalanceChange` | the signer's balance changes (independent copies) |
| `Tip` | lamports paid by System transfers to known tip accounts (`constants.TIP_ACCOUNTS`); nil when none |
| `Warnings` | why a result may be incomplete: fetcher errors (URLs cut to scheme and host) and unresolved lookup-table accounts |

In a `types.TradeInfo`:

- `InputToken` / `OutputToken`: `Mint`, `AmountRaw` (exact integer string), `Amount` (float, for display), `Decimals`, and the token accounts and balances involved. Amounts are what the user sent and received where the program reports it (events, `ray_log`), otherwise the transfers.
- `Type`: `BUY` or `SELL` relative to SOL or a stablecoin, `SWAP` otherwise.
- `Fee` and `Fees`: only fees reported by the protocol (events) or paid by transfers flagged as fees; each `FeeInfo` has `Type` (for example `protocol`, `coinCreator`, `platform`, `transferFee`), `Dex` and `Recipient`.
- `ProgramId`, `AMM`, `AMMs`, `Route`, `Pool`, `User`, `Bot`, `Idx` (`"N"` or `"N-M"`).

### Failed transactions

A transaction with `meta.err` set reverted everything it did.
By default its result has `State=true`, `Msg="transaction failed"`, `TxStatus=failed`, the fee, the signer and the balance changes, and no trades, liquidity, meme, ALT events, transfers or tip.
Set `IncludeFailedTxs: true` to decode its instructions anyway.

### Version 1 transactions

Version 1 messages (compute budget in `message.transactionConfig`) are decoded into `adapter.TransactionMessage.TransactionConfig`; they parse like other versions.

### Address lookup tables and fetchers

v0 transactions from RPC carry their lookup-table addresses in `meta.loadedAddresses`.
Transactions without meta (shreds) need the tables from elsewhere:

```go
config := &types.ParseConfig{
	// Tables you already know: table address -> all its addresses, in order
	AddressLookupTables: map[string][]string{},
	// Called once per transaction for the tables not in AddressLookupTables
	ALTsFetcher: types.NewALTsFetcher(types.FetchFilterAll,
		func(lookups []types.AddressTableLookup) (map[string]*types.LoadedAddresses, error) {
			resolved := make(map[string]*types.LoadedAddresses, len(lookups))
			for _, lookup := range lookups {
				// Load the table lookup.AccountKey (getAccountInfo) and pick
				// the addresses at lookup.WritableIndexes and ReadonlyIndexes
				resolved[lookup.AccountKey] = &types.LoadedAddresses{}
			}
			return resolved, nil
		}),
}
result := shredParser.ParseAll(tx, config)
fmt.Println("unresolved accounts:", result.HasUnresolvedAccounts)
```

Accounts that stay unresolved are empty strings: parsers never guess them, `ParseShredResult.HasUnresolvedAccounts` is set, and `DexParser` adds a warning.
`TokenAccountsFetcher` works the same way for token accounts whose mint and owner the transaction does not show.
Fetcher errors go to `ParseResult.Warnings`.

## Bots and tips

`TradeInfo.Bot` names the trading bot that took a fee on the trade.
A bot is attributed when one of its fee accounts (`constants.BOT_FEE_ACCOUNTS`), or a token account owned by one, receives in the transaction:

- at least 10000 lamports of SOL or WSOL (`utils.BotFeeMinLamports`), or 0.1% of the trade's SOL leg if that is smaller, or
- at least 0.1% (`1/utils.BotFeeMinLegDivisor`) of the trade's input or output leg in that leg's token (BONKbot, for example, charges in the traded token).

The presence of a bot account alone never counts, and the trader's own accounts never receive a bot fee.
Trades executed through a bot program (`constants.BOT_ROUTER_PROGRAMS` and the `bot`-tagged `DEX_PROGRAMS`) report the bot as `Route`.

`ParseResult.Tip` sums the System transfers to the tip accounts of Jito and 16 other landing services (`constants.TIP_ACCOUNTS`; only Jito's are verified on-chain).
Tips are never a trade fee.

```go
fmt.Println(constants.GetBotName("ZG98FUCjb8mJ824Gbs6RsgVmr1FhXb2oNiJHa2dwmPd"))     // BONKbot
fmt.Println(constants.GetTipProvider("96gYZGLnJYVFmbjzopPSU6QiEV5fGqZNyN9nmNhvrZU5")) // Jito
fmt.Println(constants.IsTradeFeeAccount("96gYZGLnJYVFmbjzopPSU6QiEV5fGqZNyN9nmNhvrZU5"))
```

## Streaming with Yellowstone gRPC

Yellowstone gRPC (Helius LaserStream, Triton and others) delivers executed transactions with meta, so `DexParser` parses them like RPC results.
`ConvertGeyserTransaction` takes the generated `*SubscribeUpdateTransactionInfo` of either `github.com/rpcpool/yellowstone-grpc/examples/golang/proto` or `github.com/helius-labs/laserstream-sdk/go/proto`.
It reads the generated field names by reflection, so this library does not import those packages: they need Go 1.24 or newer, which your program then requires.
`ConvertYellowstoneTransaction` converts a `YellowstoneTransaction` you fill yourself (the field mapping is documented in grpc_utils.go).

Transaction updates carry no block time: pass 0 or your own clock.

<!-- snippet: external -->
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// tokenAuth sends the x-token header that most Yellowstone providers require
type tokenAuth struct{ token string }

func (t tokenAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"x-token": t.token}, nil
}

func (tokenAuth) RequireTransportSecurity() bool { return true }

func main() {
	// Endpoint host:port and token of your provider
	endpoint := os.Getenv("YELLOWSTONE_ENDPOINT")
	token := os.Getenv("YELLOWSTONE_TOKEN")

	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
		grpc.WithPerRPCCredentials(tokenAuth{token}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	stream, err := pb.NewGeyserClient(conn).Subscribe(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	failed := false
	commitment := pb.CommitmentLevel_CONFIRMED
	err = stream.Send(&pb.SubscribeRequest{
		Transactions: map[string]*pb.SubscribeRequestFilterTransactions{
			"pumpfun": {
				AccountInclude: []string{constants.DEX_PROGRAMS.PUMP_FUN.ID},
				Failed:         &failed,
			},
		},
		Commitment: &commitment,
	})
	if err != nil {
		log.Fatal(err)
	}

	parser := dexparser.NewDexParser()
	config := &types.ParseConfig{ParseType: types.ParseTradesOnly()}
	for {
		update, err := stream.Recv()
		if err != nil {
			log.Fatal(err)
		}
		txUpdate := update.GetTransaction()
		if txUpdate == nil {
			continue
		}
		tx, err := dexparser.ConvertGeyserTransaction(txUpdate.GetTransaction(), txUpdate.GetSlot(), 0)
		if err != nil {
			log.Println(err)
			continue
		}
		result := parser.ParseAll(tx, config)
		for _, trade := range result.Trades {
			fmt.Printf("%s %s %s: %s %s -> %s %s\n", result.Signature, trade.AMM, trade.Type,
				trade.InputToken.AmountRaw, trade.InputToken.Mint,
				trade.OutputToken.AmountRaw, trade.OutputToken.Mint)
		}
	}
}
```

Dependencies for this program:

```bash
go get github.com/rpcpool/yellowstone-grpc/examples/golang@latest google.golang.org/grpc
```

For pre-execution data (Jito ShredStream, transactions without meta) use [ShredParser](shred-parser.md).

## Next Steps

- [ShredParser](shred-parser.md): pre-execution parsing
- [Examples](examples/index.md): code for specific use cases
- [Development](development.md): contributing and testing
