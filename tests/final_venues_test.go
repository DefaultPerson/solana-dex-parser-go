package tests

import (
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// TestFinalZeroFiOnlyRoute: a Titan route whose only hop is a 17-byte ZeroFi
// swap yielded no trade and no aggregate, because the decoder accepted only
// the 18-byte form. Truth: the user FdzxLjXJ's SNDK and USDC balance changes.
// meme-2 of the final review.
func TestFinalZeroFiOnlyRoute(t *testing.T) {
	const sig = "psShYLi8kZ5ZFsCfBSTYpXUGe2h1okSfq4aDsnVW9nPfsMQZeiJeXFRbTZ4v3a7P6KuzvCW2nX4qWbb77QhifuT"
	const sndk = "SNDKbwMUQvZhnLnxLduradgLHG5KrPuKwpnrkkGRhfH"
	tx, res := parseFixture(t, sig, nil)
	user := res.Signer[0]
	sent := ownerTokenDelta(tx, user, sndk)
	if sent.String() != "-7448" {
		t.Fatalf("fixture: user SNDK delta %s", sent)
	}
	var zerofi int
	for _, tr := range res.Trades {
		if tr.ProgramId == constants.DEX_PROGRAMS.ZERO_FI.ID {
			zerofi++
		}
	}
	if zerofi != 1 {
		t.Errorf("ZeroFi trades %d, want 1 (trades %d)", zerofi, len(res.Trades))
	}
	agg := res.AggregateTrade
	if agg == nil {
		t.Fatal("no aggregate trade")
	}
	if agg.InputToken.Mint != sndk || agg.InputToken.AmountRaw != "7448" || agg.OutputToken.Mint != usdcMint {
		t.Errorf("aggregate %s %s -> %s %s, want 7448 SNDK -> USDC", agg.InputToken.AmountRaw, agg.InputToken.Mint, agg.OutputToken.AmountRaw, agg.OutputToken.Mint)
	}
}
