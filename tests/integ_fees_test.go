package tests

import (
	"math/big"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// The aggregate trade lists each fee once. The Pump.fun, PumpSwap,
// LaunchLab, DBC and meme parsers set a trade's Fee to the total of its Fees
// components (as upstream does), so listing Fee and Fees of every trade
// counted those fees twice: e.g. 29b7yFfv... listed the PumpSwap total
// 5923523 next to its components 197451 + 197451 + 5528621. Checked on every
// fixture: no untyped fee of the aggregate equals the sum of the typed fees
// of its dex and mint.
func TestIntegAggregateFeesCountedOnce(t *testing.T) {
	p := dexparser.NewDexParser()
	for _, sig := range fixtureSignatures(t, "json") {
		agg := p.ParseAll(loadFixture(t, sig), nil).AggregateTrade
		if agg == nil {
			continue
		}
		typed := map[[2]string]*big.Int{}
		for _, f := range agg.Fees {
			if f.Type == "" || f.Type == "transferFee" {
				continue
			}
			key := [2]string{f.Dex, f.Mint}
			if typed[key] == nil {
				typed[key] = new(big.Int)
			}
			v, _ := new(big.Int).SetString(f.AmountRaw, 10)
			if v != nil {
				typed[key].Add(typed[key], v)
			}
		}
		for _, f := range agg.Fees {
			if f.Type != "" {
				continue
			}
			if sum := typed[[2]string{f.Dex, f.Mint}]; sum != nil && sum.String() == f.AmountRaw {
				t.Errorf("%s: aggregate lists the %s total %s next to its components", sig[:8], f.Dex, f.AmountRaw)
			}
		}
	}
}

// utils.FeeComponents keeps a Fee that is a separate fee (the AMM
// convention: Fee is the LP fee, Fees the others) and drops one that is the
// total of the Fees of its mint (the meme convention).
func TestIntegFeeComponents(t *testing.T) {
	fee := func(amount, typ string) types.FeeInfo {
		return types.FeeInfo{Mint: venueSOL, AmountRaw: amount, Type: typ}
	}
	lp := fee("100", "lp")
	amm := types.TradeInfo{Fee: &lp, Fees: []types.FeeInfo{fee("20", "protocol")}}
	if got := utils.FeeComponents(&amm); len(got) != 2 || got[0].AmountRaw != "100" || got[1].AmountRaw != "20" {
		t.Errorf("AMM convention: %+v", got)
	}
	total := fee("30", "")
	meme := types.TradeInfo{Fee: &total, Fees: []types.FeeInfo{fee("10", "protocol"), fee("20", "coinCreator")}}
	if got := utils.FeeComponents(&meme); len(got) != 2 || got[0].Type != "protocol" || got[1].Type != "coinCreator" {
		t.Errorf("meme convention: %+v", got)
	}
	// An untyped fee that is not the total is kept
	other := fee("31", "")
	notTotal := types.TradeInfo{Fee: &other, Fees: meme.Fees}
	if got := utils.FeeComponents(&notTotal); len(got) != 3 {
		t.Errorf("untyped non-total fee: %+v", got)
	}
}
