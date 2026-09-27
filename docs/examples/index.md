# Examples

In the snippets below `tx` is a `*adapter.SolanaTransaction` (see [Fetching transactions](../getting-started.md#fetching-transactions)), `parser` is `dexparser.NewDexParser()` and `shredParser` is `dexparser.NewShredParser()`.
Complete, runnable programs are in [Getting Started](../getting-started.md); every snippet here is compiled by the test suite.

## Parse all data

```go
result := parser.ParseAll(tx, nil)
if !result.State {
	log.Fatal(result.Msg)
}
fmt.Println("status:", result.TxStatus)
fmt.Println("trades:", len(result.Trades))
fmt.Println("liquidities:", len(result.Liquidities))
fmt.Println("meme events:", len(result.MemeEvents))
fmt.Println("transfers:", len(result.Transfers))
for _, warning := range result.Warnings {
	fmt.Println("warning:", warning)
}
```

## Trades

```go
for _, trade := range parser.ParseTrades(tx, nil) {
	fmt.Println("type:", trade.Type, "amm:", trade.AMM, "route:", trade.Route, "bot:", trade.Bot)
	fmt.Println("user:", trade.User, "pool:", trade.Pool, "idx:", trade.Idx)
	// AmountRaw is exact; Amount is a float64 for display
	fmt.Println("in: ", trade.InputToken.AmountRaw, trade.InputToken.Mint, trade.InputToken.Decimals)
	fmt.Println("out:", trade.OutputToken.AmountRaw, trade.OutputToken.Mint, trade.OutputToken.Decimals)
	for _, fee := range trade.Fees {
		fmt.Println("fee:", fee.Type, fee.Dex, fee.AmountRaw, fee.Mint, fee.Recipient)
	}
}
```

## Aggregated trade

A multi-hop or routed swap has one trade per hop; `AggregateTrade` goes from the first input to the last output.
A nil config computes it; with a config, set `ParseType.AggregateTrade`.
For a Titan or OKX DEX Router V2 route it is the aggregator's route total, with the aggregator fee in `Fee`.

```go
result := parser.ParseAll(tx, &types.ParseConfig{ParseType: types.ParseTradesOnly()})
if agg := result.AggregateTrade; agg != nil {
	fmt.Println(agg.Type, agg.InputToken.AmountRaw, agg.InputToken.Mint, "->",
		agg.OutputToken.AmountRaw, agg.OutputToken.Mint, "via", agg.AMMs)
	fmt.Println("arbitrage:", result.IsArbitrage())
}
```

## Liquidity events

```go
for _, event := range parser.ParseLiquidity(tx, nil) {
	fmt.Println(event.Type, event.AMM, "pool:", event.PoolId)
	fmt.Println("token0:", event.Token0AmountRaw, event.Token0Mint)
	fmt.Println("token1:", event.Token1AmountRaw, event.Token1Mint)
	fmt.Println("lp:", event.LpAmountRaw, event.PoolLpMint)
}
```

## Meme events

```go
result := parser.ParseAll(tx, nil)
for _, event := range result.MemeEvents {
	fmt.Println(event.Protocol, event.Type, "mint:", event.BaseMint, "quote:", event.QuoteMint)
	fmt.Println("user:", event.User, "pool:", event.Pool, "idx:", event.Idx)
}
```

## Transfers

`Transfers` is filled when the transaction has no trades and no liquidity events.
Program actions such as Jupiter DCA deposits, limit orders and Pump.fun fee payouts set `Type` (`OpenDca`, `settleLimitOrder`, `claimCashback`, …).

```go
for _, transfer := range parser.ParseTransfers(tx, nil) {
	fmt.Println(transfer.Type, transfer.Info.TokenAmount.Amount, transfer.Info.Mint,
		transfer.Info.Source, "->", transfer.Info.Destination, "fee:", transfer.IsFee)
}
```

## Filter by program

```go
config := &types.ParseConfig{
	ProgramIds: []string{
		constants.DEX_PROGRAMS.PUMP_FUN.ID,
		constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
	},
}
result := parser.ParseAll(tx, config)
if !result.State {
	fmt.Println(result.Msg) // "No matching program ids"
}
```

## Ignore programs

```go
config := &types.ParseConfig{
	IgnoreProgramIds: []string{constants.DEX_PROGRAMS.PHOENIX.ID},
}
result := parser.ParseAll(tx, config)
fmt.Println(len(result.Trades))
```

## Failed transactions

```go
result := parser.ParseAll(tx, nil)
if result.TxStatus == types.TransactionStatusFailed {
	fmt.Println(result.Msg, "fee:", result.Fee.Amount) // "transaction failed"
}

// Decode the reverted instructions anyway
config := types.DefaultParseConfig()
config.IncludeFailedTxs = true
result = parser.ParseAll(tx, &config)
fmt.Println(len(result.Trades))
```

## Batch parsing

```go
txs := []*adapter.SolanaTransaction{tx}
results := parser.ParseBatch(txs, nil, 8) // 8 workers; results keep the input order
for i, result := range results {
	fmt.Println(i, result.Signature, len(result.Trades))
}
```

## ShredParser

```go
result := shredParser.ParseAll(tx, nil)
for _, ins := range result.ParsedInstructions {
	switch {
	case ins.Trade != nil:
		fmt.Println(ins.Idx, ins.ProgramName, ins.Action, ins.Trade.Type,
			ins.Trade.InputToken.AmountRaw, ins.InputAmountKind,
			ins.Trade.OutputToken.AmountRaw, ins.OutputAmountKind)
	case ins.MemeEvent != nil:
		fmt.Println(ins.Idx, ins.ProgramName, ins.Action, ins.MemeEvent.Type, ins.MemeEvent.BaseMint)
	case ins.Transfer != nil:
		fmt.Println(ins.Idx, ins.ProgramName, ins.Action, ins.Transfer.Info.TokenAmount.Amount)
	}
}
```

See [ShredParser](../shred-parser.md) for what these amounts mean.

## Raydium V4 logs

The Raydium AMM v4 program writes a `ray_log` line per swap, deposit and withdraw:

```go
for _, line := range tx.Meta.LogMessages {
	payload, ok := strings.CutPrefix(line, "Program log: ray_log: ")
	if !ok {
		continue
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		continue
	}
	if swap := raydium.ParseRaydiumSwapLog(raydium.DecodeRaydiumLog(data)); swap != nil {
		fmt.Println(swap.Type, swap.Mode, swap.InputAmount, swap.OutputAmount, swap.SlippageProtection)
	}
}
```

## Constants

```go
fmt.Println(constants.GetProgramName(constants.DEX_PROGRAMS.ORCA.ID)) // Orca
fmt.Println(constants.GetProgramName("11111111111111111111111111111111")) // Unknown: not a DEX program
fmt.Println(constants.IsDexProgram(constants.DEX_PROGRAMS.PUMP_SWAP.ID))  // true
fmt.Println(constants.GetBotNames())
fmt.Println(constants.GetTipProvider("96gYZGLnJYVFmbjzopPSU6QiEV5fGqZNyN9nmNhvrZU5")) // Jito
```
