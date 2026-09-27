package tests

import (
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// Regression tests for the core findings of the final review (core-1,
// core-2, complete-1, complete-2, robust-2, robust-3). Expected values come
// from the raw fixtures: instructions, token balances and balance changes.

// TestFinalParsedMultisigAuthority: the RPC's jsonParsed encoding names the
// authority of a transfer "multisigAuthority" whenever signer accounts follow
// it (every Token-2022 transfer-hook transfer), and the parser read only
// "authority", so the json and jsonParsed trades of 3ou2zc4g differed in the
// output's authority. Truth: account 3 of the transferChecked at 4-13 in the
// json encoding. complete-2.
func TestFinalParsedMultisigAuthority(t *testing.T) {
	const sig = "3ou2zc4g8GGfufg1yX9DJpmSPmr6Uo25LsuocezSf5swegypu1aSYtfw47ZSte6PCUBexSeA4Veim65vqaDXp8zr"
	var authority string
	for _, ix := range fixtureIxs(t, loadFixture(t, sig)) {
		if ix.outer == 4 && ix.inner == 13 && ix.programId == constants.TOKEN_2022_PROGRAM_ID && len(ix.accounts) > 4 {
			authority = ix.accounts[3]
		}
	}
	if authority != "FhVo3mqL8PW5pH5U2CN4XE33DokiyZnUwuGpH2hmHLuM" {
		t.Fatalf("fixture: authority of 4-13 %q", authority)
	}
	parser := dexparser.NewDexParser()
	for encoding, tx := range map[string]*adapterTx{"json": loadFixture(t, sig), "jsonParsed": loadParsedFixture(t, sig)} {
		res := parser.ParseAll(tx, nil)
		found := false
		for _, tr := range res.Trades {
			if tr.Idx == "4-11" {
				found = true
				if tr.OutputToken.Authority != authority {
					t.Errorf("%s: trade 4-11 output authority %q, want %s", encoding, tr.OutputToken.Authority, authority)
				}
			}
		}
		if !found {
			t.Errorf("%s: no trade at 4-11", encoding)
		}
	}
}
