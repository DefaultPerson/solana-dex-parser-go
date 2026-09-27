package tests

import (
	"testing"

	"github.com/goccy/go-json"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// TestCoreVersion1Transactions: mainnet carries version-1 transactions, which
// the old fetch helpers (maxSupportedTransactionVersion 0) could not load and
// which carry the compute budget in message.transactionConfig instead of
// ComputeBudget instructions. The fixtures were fetched with
// maxSupportedTransactionVersion 1. core-17, meme-20, constants-21, amm-23.
func TestCoreVersion1Transactions(t *testing.T) {
	p := dexparser.NewDexParser()
	n, withEvents := 0, 0
	for _, sig := range fixtureSignatures(t, "json") {
		raw, err := readFixture(sig, "json")
		if err != nil {
			t.Fatal(err)
		}
		// independent decoding of the fields under test
		var doc struct {
			Version     interface{} `json:"version"`
			Transaction struct {
				Message struct {
					TransactionConfig *struct {
						ComputeUnitLimit *uint64 `json:"computeUnitLimit"`
						PriorityFee      *uint64 `json:"priorityFee"`
					} `json:"transactionConfig"`
				} `json:"message"`
			} `json:"transaction"`
			Meta struct {
				Fee                  uint64 `json:"fee"`
				ComputeUnitsConsumed uint64 `json:"computeUnitsConsumed"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		if v, ok := doc.Version.(float64); !ok || v != 1 {
			continue
		}
		n++
		tx := loadFixture(t, sig)
		cfg := doc.Transaction.Message.TransactionConfig
		got := tx.Transaction.Message.TransactionConfig
		if cfg == nil || got == nil || cfg.ComputeUnitLimit == nil || got.ComputeUnitLimit == nil || *got.ComputeUnitLimit != *cfg.ComputeUnitLimit ||
			(cfg.PriorityFee == nil) != (got.PriorityFee == nil) || (cfg.PriorityFee != nil && *got.PriorityFee != *cfg.PriorityFee) {
			t.Errorf("%.12s: TransactionConfig = %+v", sig, got)
		}
		r := p.ParseAll(tx, &types.ParseConfig{ParseType: types.ParseAll(), TryUnknownDEX: true, IncludeFailedTxs: true})
		if !r.State || r.ComputeUnits != doc.Meta.ComputeUnitsConsumed || r.Fee.Amount != bigStr("0").SetUint64(doc.Meta.Fee).String() {
			t.Errorf("%.12s: State=%v ComputeUnits=%d Fee=%s", sig, r.State, r.ComputeUnits, r.Fee.Amount)
		}
		// two fixtures call a single program without any CPI or transfer
		if len(r.Trades) > 0 || len(r.Liquidities) > 0 || len(r.MemeEvents) > 0 || len(r.Transfers) > 0 {
			withEvents++
		}
	}
	if n < 10 || withEvents < 8 {
		t.Errorf("%d version-1 fixtures, %d with events", n, withEvents)
	}

	// a v1 Jupiter-routed circular swap: three hops, SOL in and out
	tx, r := parseFixture(t, "5sV51YrwGRFwpwgnWHWKCq3TWmCQ47XnsNuMNAKs7Jypf9g6H9JTPnY6ZqW5tfSTBwBL8Un5MPxK5cKZsSkaacty", nil)
	user := tx.Transaction.Message.AccountKeys[0].Pubkey
	if len(r.Trades) != 3 || r.AggregateTrade == nil || r.AggregateTrade.InputToken.Mint != solMint || r.AggregateTrade.OutputToken.Mint != solMint {
		t.Fatalf("5sV51Yrw: trades %v aggregate %v", r.Trades, r.AggregateTrade)
	}
	// the arbitrage profit landed in the user's WSOL account
	profit := bigStr(r.AggregateTrade.OutputToken.AmountRaw)
	profit.Sub(profit, bigStr(r.AggregateTrade.InputToken.AmountRaw))
	if profit.Cmp(ownerTokenDelta(tx, user, solMint)) != 0 {
		t.Errorf("5sV51Yrw: output - input = %s, WSOL delta %s", profit, ownerTokenDelta(tx, user, solMint))
	}
}
