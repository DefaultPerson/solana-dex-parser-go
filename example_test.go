package dexparser_test

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// The bodies of these examples are copied verbatim into README.md and docs/
// (tests/docs_snippets_test.go checks that they stay identical).

// txFile is a real Pump.fun create + buy (4Cod1c...): the "result" of a
// getTransaction call with {"encoding": "json", "maxSupportedTransactionVersion": 1}
const txFile = "testdata/example/4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz.json"

// loadTransaction reads a getTransaction result, as in the Quick Start
func loadTransaction(path string) *adapter.SolanaTransaction {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(raw, &tx); err != nil {
		log.Fatal(err)
	}
	return &tx
}

func ExampleDexParser_ParseAll() {
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

	// Output:
	// status: success fee: 80285
	// trade: Pumpfun BUY 2000000000 So11111111111111111111111111111111111111112 -> 67062499999999 B9Z9mKUoVy5k8KuL2HauUD1mhmfF3PPNnJoK83S1pump
	// meme: Pumpfun CREATE B9Z9mKUoVy5k8KuL2HauUD1mhmfF3PPNnJoK83S1pump
	// meme: Pumpfun BUY B9Z9mKUoVy5k8KuL2HauUD1mhmfF3PPNnJoK83S1pump
	// tip: 4000000
}

func ExampleDexParser_ParseAll_config() {
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

	// Output:
	// trades: 1 meme events: 0
	// aggregate: BUY 2000000000 -> 67062499999999
}

func ExampleShredParser_ParseAll() {
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

	// Output:
	// status: unknown
	// 1 System transfer
	// 3 Pumpfun create
	// 5 Pumpfun buy: BUY 2020000000 (max) -> 67062499999999 (exact)
}
