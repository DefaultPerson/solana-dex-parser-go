package tests

import (
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// amm-21: every DCA fill ends with the DCA program's transfer instruction
// paying the fill's output to the user and emitting a Withdraw event
// (dca_key, in_amount, out_amount, user_withdraw). It was not decoded, so
// ParseTransfers reported nothing for it. Truth: the Withdraw event bytes
// and the transfer instruction's accounts (DCA IDL: keeper 0, dca 1, user 2,
// output_mint 3, dca_out_ata 4, user_out_ata 5).
func TestIntegDCAWithdrawTransfers(t *testing.T) {
	dca := constants.DEX_PROGRAMS.JUPITER_DCA.ID
	fills := 0
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		var transferIx *rawIx
		var outAmount uint64
		var dcaKey string
		for _, x := range fixtureIxs(t, tx) {
			if x.programId != dca {
				continue
			}
			x := x
			switch {
			case constants.MatchDiscriminator(x.data, constants.DISCRIMINATORS.JUPITER_DCA.TRANSFER) && len(x.accounts) > 5:
				transferIx = &x
			case constants.MatchDiscriminator(x.data, constants.DISCRIMINATORS.JUPITER_DCA.WITHDRAW_EVENT) && len(x.data) >= 16+49:
				dcaKey = base58.Encode(x.data[16:48])
				outAmount = binary.LittleEndian.Uint64(x.data[56:64])
			}
		}
		if transferIx == nil || outAmount == 0 {
			continue
		}
		fills++
		var withdraws []string
		for _, tr := range dexparser.NewDexParser().ParseTransfers(tx, nil) {
			if tr.Type != "WithdrawDca" {
				continue
			}
			withdraws = append(withdraws, tr.Idx)
			a := transferIx.accounts
			if tr.ProgramId != dca || tr.Info.Mint != a[3] || tr.Info.TokenAmount.Amount != uint64String(outAmount) ||
				tr.Info.Source != a[4] || tr.Info.Destination != a[5] || dcaKey != a[1] {
				t.Errorf("%s: WithdrawDca %s %s %s -> %s, want %d %s %s -> %s", sig[:8], tr.Info.TokenAmount.Amount, tr.Info.Mint,
					tr.Info.Source, tr.Info.Destination, outAmount, a[3], a[4], a[5])
			}
		}
		if len(withdraws) != 1 {
			t.Errorf("%s: %d WithdrawDca transfers %v, want 1", sig[:8], len(withdraws), withdraws)
		}
	}
	if fills == 0 {
		t.Fatal("no DCA fill fixture")
	}
}

func uint64String(v uint64) string {
	return strconv.FormatUint(v, 10)
}
