package tests

import (
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
)

// TestTransferFeeOnUserSideInput checks that a trade whose input already is
// what the user sent (a Jupiter hop) still lists the Token-2022 withheld
// transfer fee. Truth: the user's transfer minus the pool vault's credit,
// exactly the mint's fee rate (1% and 3%).
func TestTransferFeeOnUserSideInput(t *testing.T) {
	cases := []struct {
		sig, idx, input, fee string
	}{
		{"5iaG7mCfMRQxWySMFZxVRamKLUtfs9gtcmaPBmfpg8Ph4yW5Wvez48X5JZM4WTytckZtHbLLSfyr9Cu8Q8XbvjHK", "3-0", "230600359485", "2306003595"},
		{"NeF1UiWXKUbuswNNw14gJ2uup7yrV6KyXijQ7dsjLYanj9dnsBSeqGPDVZ3wP3NfhXQ84rJncRo5XwbrbSdWVWW", "3-0", "3796748774", "37967488"},
		{"wYtzWCq7VRtivvcyBT3eZ58sipiN5yefkaMpZty1Udi4wB8LR8hXuhkYP39DK66tgCAzX8LxwRWHEtJubZN8DeC", "7-5", "110610674", "3318321"},
	}
	parser := dexparser.NewDexParser()
	for _, tc := range cases {
		res := parser.ParseAll(loadFixture(t, tc.sig), nil)
		found := false
		for _, trade := range res.Trades {
			if trade.Idx != tc.idx {
				continue
			}
			found = true
			if trade.InputToken.AmountRaw != tc.input {
				t.Errorf("%s %s: input %s, want %s", tc.sig[:8], tc.idx, trade.InputToken.AmountRaw, tc.input)
			}
			fees := 0
			for _, f := range trade.Fees {
				if f.Type == "transferFee" && f.Mint == trade.InputToken.Mint {
					fees++
					if f.AmountRaw != tc.fee {
						t.Errorf("%s %s: transferFee %s, want %s", tc.sig[:8], tc.idx, f.AmountRaw, tc.fee)
					}
				}
			}
			if fees != 1 {
				t.Errorf("%s %s: %d transferFee entries for the input mint, want 1", tc.sig[:8], tc.idx, fees)
			}
		}
		if !found {
			t.Errorf("%s: no trade at %s", tc.sig[:8], tc.idx)
		}
	}
}
