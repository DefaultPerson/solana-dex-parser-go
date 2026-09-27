package tests

import (
	"testing"

	"github.com/goccy/go-json"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Regression tests for the core follow-up (WP core2): no aliasing of cached
// balance changes, fetcher errors surfaced, wallet entries from constants.

// TestCore2BalanceChangesNotAliased: result.TokenBalanceChange,
// result.SolBalanceChange and the balance pointers of trades pointed into the
// adapter's cached maps, so editing one changed the others.
func TestCore2BalanceChangesNotAliased(t *testing.T) {
	p := dexparser.NewDexParser()
	checked := 0
	for _, sig := range fixtureSignatures(t, "json") {
		r := p.ParseAll(loadFixture(t, sig), nil)
		if r.AggregateTrade == nil {
			continue
		}
		before, _ := json.Marshal([]interface{}{r.SolBalanceChange, r.TokenBalanceChange})
		var amounts []*types.TokenAmount
		for _, tr := range append(append([]types.TradeInfo{}, r.Trades...), *r.AggregateTrade) {
			for _, tok := range []types.TokenInfo{tr.InputToken, tr.OutputToken} {
				amounts = append(amounts, tok.SourceBalance, tok.SourcePreBalance, tok.DestinationBalance, tok.DestinationPreBalance)
			}
		}
		for _, a := range amounts {
			if a != nil {
				a.Amount = "-1"
				if a.UIAmount != nil {
					*a.UIAmount = -1
				}
			}
		}
		after, _ := json.Marshal([]interface{}{r.SolBalanceChange, r.TokenBalanceChange})
		if string(before) != string(after) {
			t.Errorf("%s: editing trade balances changed the result's balance changes", sig[:8])
		}
		checked++
	}
	if checked < 50 {
		t.Fatalf("only %d fixtures with an aggregate trade", checked)
	}

	// the result's balance changes are copies of the adapter's cached maps
	tx := loadFixture(t, sigT22Fee)
	r := p.ParseAll(tx, nil)
	a := adapter.NewTransactionAdapter(tx, nil)
	signer := a.Signer()
	for mint, c := range r.TokenBalanceChange {
		c.Change.Amount = "0"
		if a.GetAccountTokenBalanceChanges(true)[signer][mint].Change.Amount == "0" {
			t.Errorf("TokenBalanceChange[%s] aliases the adapter cache", mint)
		}
	}
}
