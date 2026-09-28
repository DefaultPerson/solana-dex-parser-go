package tests

import (
	"math/big"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Regression test for meme-1 of the final review. Expected values come from
// the raw fixtures' token balances.

// TestFinalLaunchLabTransferFeeAmounts: LaunchLab TradeEvent base amounts are
// the pool's side. For a Token-2022 base mint with a transfer fee, sells
// reported the vault's credit as the input (1-3% less than the user sent),
// in the trade and in the aggregate, and buys' meme events reported the
// gross output. Truth: the user's base-mint balance change; the withheld
// amount is a "transferFee" fee in the base mint (a quote mint with a
// transfer fee of its own, as in 34NzNezC, adds one in the quote mint).
func TestFinalLaunchLabTransferFeeAmounts(t *testing.T) {
	for _, c := range []struct {
		sig  string
		sell bool
		// the LaunchLab trade is the whole transaction: the user's quote
		// balance change is the trade's quote amount
		direct bool
	}{
		{"3JG4tuLfr7dMAH52EsbdDsy2nqi3fYJuuyKBDAcsuqwKJuahL599UucAj9iohjJcV9XoiBpwgGKXAbwxGyPHC3bL", true, false},
		{"5dnD4akxGFE31hLnSfybpL24geVkFDHNVcyCsDwVLwig2HPwL26EjKRkwEm8CKajGamYRDV1DN4BkFDDWMnJntEv", true, false},
		{"2LcwEtHvr6sep7kcgKk6GkHhd5TEUPrpdQGij9yEEJtxxyscggoFYd89DwuKd8C6quXprAxHcCx9x6Q4G1SGkgfb", true, false},
		{"34NzNezCpveDCqd7ufFcHsQnvv1sLdAxdFGgDkVzc2DaCNPSrjd69TNBRrL3AxuX7eHFqma2X6bwpLbmBpUfYEiF", true, true},
		{"4YGZcD8fwC7dXB9v3cmEfXjx3mfbsbNq49CxxopsaoBZCnNaPtAB8Kmk74DVxfZez1zaKHotrNQXFbjf8utXgDuy", true, false},
		{"4hW26D8Ue9UCrptobryYG55MkdJUKcqFLiitWPFkaofKShuHPQP7NDfbTmQVm5SGcwcZo5VNcVoqFq14t7UmJ9Ce", false, false},
	} {
		tx, res := parseFixture(t, c.sig, nil)
		var trade *types.TradeInfo
		for i := range res.Trades {
			if res.Trades[i].AMM == constants.DEX_PROGRAMS.RAYDIUM_LCP.Name {
				trade = &res.Trades[i]
			}
		}
		var meme *types.MemeEvent
		for i := range res.MemeEvents {
			if res.MemeEvents[i].Protocol == constants.DEX_PROGRAMS.RAYDIUM_LCP.Name && res.MemeEvents[i].Type != types.TradeTypeSwap && res.MemeEvents[i].InputToken != nil {
				meme = &res.MemeEvents[i]
			}
		}
		if trade == nil || meme == nil {
			t.Fatalf("%s: LaunchLab trade %v, meme event %v", c.sig[:8], trade != nil, meme != nil)
		}
		base, baseToken, memeToken := meme.BaseMint, &trade.OutputToken, meme.OutputToken
		if c.sell {
			baseToken, memeToken = &trade.InputToken, meme.InputToken
		}
		delta := ownerTokenDelta(tx, trade.User, base)
		want := new(big.Int).Abs(delta).String()
		if (delta.Sign() < 0) != c.sell || baseToken.Mint != base {
			t.Fatalf("%s: user base delta %s for a sell=%v", c.sig[:8], delta, c.sell)
		}
		if baseToken.AmountRaw != want {
			t.Errorf("%s: trade base amount %s, want the user's %s", c.sig[:8], baseToken.AmountRaw, want)
		}
		if memeToken.AmountRaw != want {
			t.Errorf("%s: meme event base amount %s, want the user's %s", c.sig[:8], memeToken.AmountRaw, want)
		}
		if c.sell && res.AggregateTrade != nil && res.AggregateTrade.InputToken.Mint == base && res.AggregateTrade.InputToken.AmountRaw != want {
			t.Errorf("%s: aggregate input %s, want %s", c.sig[:8], res.AggregateTrade.InputToken.AmountRaw, want)
		}
		// the quote side is what the user received or paid too (34NzNezC's
		// quote mint withholds a transfer fee of its own)
		quoteToken := &trade.InputToken
		if c.sell {
			quoteToken = &trade.OutputToken
		}
		if wantQuote := new(big.Int).Abs(ownerTokenDelta(tx, trade.User, meme.QuoteMint)).String(); c.direct && quoteToken.AmountRaw != wantQuote {
			t.Errorf("%s: trade quote amount %s, want the user's %s", c.sig[:8], quoteToken.AmountRaw, wantQuote)
		}
		for name, fees := range map[string][]types.FeeInfo{"trade": trade.Fees, "meme event": meme.Fees} {
			n := 0
			for _, f := range fees {
				if f.Type == "transferFee" {
					switch f.Mint {
					case base:
						n++
					case meme.QuoteMint:
					default:
						t.Errorf("%s %s: transfer fee in %s, want the base or quote mint", c.sig[:8], name, f.Mint)
					}
				}
			}
			if n != 1 {
				t.Errorf("%s %s: %d base-mint transfer fees, want 1", c.sig[:8], name, n)
			}
		}
		if trade.Fee == nil || trade.Fee.Mint != meme.QuoteMint {
			t.Errorf("%s: trade Fee %+v, want the total of the quote-mint fees", c.sig[:8], trade.Fee)
		}
	}
}
