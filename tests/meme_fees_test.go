package tests

import (
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// constants-12: the protocol, reserved (mayhem) and buyback fee recipients
// that Pump.fun and PumpSwap trades name in their events and instructions are
// all known fee accounts, so their transfers are flagged IsFee. Covers every
// Pump.fun and PumpSwap trade in the fixtures.
func TestMemePumpFeeRecipientsAreFeeAccounts(t *testing.T) {
	seen := 0
	for _, sig := range fixtureSignatures(t, "json") {
		r := dexparserParse(loadFixture(t, sig))
		for _, e := range r.MemeEvents {
			if e.Protocol != constants.DEX_PROGRAMS.PUMP_FUN.Name && e.Protocol != constants.DEX_PROGRAMS.PUMP_SWAP.Name {
				continue
			}
			for _, f := range e.Fees {
				if (f.Type != "protocol" && f.Type != "buyback") || f.Recipient == "" {
					continue
				}
				seen++
				if !constants.IsFeeAccount(f.Recipient) {
					t.Errorf("%.8s %s: %s fee recipient %s is not in FEE_ACCOUNTS", sig, e.Idx, f.Type, f.Recipient)
				}
			}
		}
	}
	if seen < 20 {
		t.Errorf("only %d fee recipients checked", seen)
	}
}
