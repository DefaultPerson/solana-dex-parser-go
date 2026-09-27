package tests

import (
	"bytes"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// A PumpSwap trade in a pool quoted in a token other than SOL or a
// stablecoin takes its type from the event: a buy of the pool's base token
// is BUY, a sell SELL. Before, PumpSwap typed every trade whose input was
// not SOL, USDC or USDT as SELL, and Jupiter hops copy the venue's type.
//
// Real trades in the PumpSwap pool 95XExVtM..., whose base is 3KM7dv... and
// whose quote is the PUMP mint: direct PumpSwap buys CPI'd by Axiom and
// DFlow, and a buy and a sell as Jupiter hops. The truth is the side of the
// pump_amm instruction (buy or sell, IDL accounts pool 0, base_mint 3,
// quote_mint 4), decoded from the raw fixture.
func TestIntegPumpSwapTokenQuotedType(t *testing.T) {
	const pool = "95XExVtMNeEW5NFajZwaL22Q1gPSzMoin864akzRCo7"
	sigs := []string{
		"2nZogzJEMFtAS8rsKiAohAT4ja15N5aLe6iuuh2bakxMoJx6vq1XLhgZrwRwA1CY25rYjKFQeCCiauSMMmCeycrF", // Axiom: USDC -> PUMP -> base
		"i8tFZYDUoAXQLtpQjQKXmuo5oi1t2aXnf4AEMEcbrDRXMef3tzcznU8qrn317DygRhfNgm8qKdYpYerZFv6UGyE",  // DFlow
		"vdEvGDG7gCv6By1jPYqpZQtPo4KaNkjzeXjHtmQQ3sfSm2yPKKadmD4uxxiGjXctk5zyVYEgShrfw4GzHk32zdp",  // Jupiter hop, buy
		"4GGEEgo8Bw9NcusTT2Hm8utYEcp72iZnmzXNE3h8rcMnLeFyx5ApFu1A4TiEk3NPimGMUrDnJjNd32794BroFvbk", // Jupiter hop, sell
	}
	d := constants.DISCRIMINATORS.PUMPSWAP
	sides := map[types.TradeType]int{}
	for _, sig := range sigs {
		// the pump_amm swap on the pool, from the raw transaction
		raw := loadMemeRaw(t, sig)
		var want types.TradeType
		var baseMint, quoteMint string
		for outer := range raw.outer {
			for inner := -1; inner < len(raw.inner[outer]); inner++ {
				accounts := raw.accounts(outer, inner)
				if raw.program(outer, inner) != constants.DEX_PROGRAMS.PUMP_SWAP.ID || len(accounts) < 5 || accounts[0] != pool {
					continue
				}
				data := raw.data(outer, inner)
				switch {
				case bytes.HasPrefix(data, d.BUY) || bytes.HasPrefix(data, d.BUY_EXACT_QUOTE_IN):
					want = types.TradeTypeBuy
				case bytes.HasPrefix(data, d.SELL):
					want = types.TradeTypeSell
				default:
					continue
				}
				baseMint, quoteMint = accounts[3], accounts[4]
			}
		}
		if want == "" {
			t.Fatalf("%.8s: no pump_amm buy or sell on %s", sig, pool)
		}
		if constants.IsQuoteToken(quoteMint) || constants.IsQuoteToken(baseMint) {
			t.Fatalf("%.8s: pool %s is quoted in %s, not a token-quoted pool", sig, pool, quoteMint)
		}
		sides[want]++

		var trades []types.TradeInfo
		for _, tr := range dexparser.NewDexParser().ParseAll(loadFixture(t, sig), nil).Trades {
			if containsStr(tr.Pool, pool) {
				trades = append(trades, tr)
			}
		}
		if len(trades) != 1 {
			t.Fatalf("%.8s: %d trades in pool %s, want 1", sig, len(trades), pool)
		}
		tr := trades[0]
		in, out := quoteMint, baseMint
		if want == types.TradeTypeSell {
			in, out = baseMint, quoteMint
		}
		if tr.Type != want || tr.AMM != constants.DEX_PROGRAMS.PUMP_SWAP.Name || tr.InputToken.Mint != in || tr.OutputToken.Mint != out {
			t.Errorf("%.8s: %s %s %s -> %s, want %s %s -> %s", sig, tr.AMM, tr.Type, tr.InputToken.Mint, tr.OutputToken.Mint, want, in, out)
		}
	}
	if sides[types.TradeTypeBuy] == 0 || sides[types.TradeTypeSell] == 0 {
		t.Errorf("fixtures cover sides %v, want both", sides)
	}
}
