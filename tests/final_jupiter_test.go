package tests

import (
	"math/big"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Regression tests for the Jupiter Limit order cancel findings of the final
// review (robust-1, robust-verify-1). Expected values come from the raw
// fixtures' lamport balances and instruction accounts.

// limitCancel is a Limit v1/v2 cancel instruction of a fixture with the
// lamports its order and reserve accounts released (pre - post, from meta)
type limitCancel struct {
	idx      string
	released *big.Int
}

// limitCancels lists the cancel instructions of a fixture: Limit v2
// cancel_order (order 2, input_mint_reserve 3) and Limit v1 cancelOrder
// (order 0, reserve 1)
func limitCancels(t *testing.T, tx *adapterTx) []limitCancel {
	t.Helper()
	keys := rawAccountKeys(tx)
	lamports := func(account string) *big.Int {
		for i, k := range keys {
			if k == account {
				return new(big.Int).Sub(new(big.Int).SetUint64(tx.Meta.PreBalances[i]), new(big.Int).SetUint64(tx.Meta.PostBalances[i]))
			}
		}
		return new(big.Int)
	}
	var out []limitCancel
	for _, ix := range fixtureIxs(t, tx) {
		var order, reserve string
		switch {
		case ix.programId == constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID && constants.MatchDiscriminator(ix.data, constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER_V2.CANCEL_ORDER):
			order, reserve = ix.accounts[2], ix.accounts[3]
		case ix.programId == constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER.ID && constants.MatchDiscriminator(ix.data, constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER.CANCEL_ORDER):
			order, reserve = ix.accounts[0], ix.accounts[1]
		default:
			continue
		}
		out = append(out, limitCancel{fmtIdx(ix.outer, ix.inner), new(big.Int).Add(lamports(order), lamports(reserve))})
	}
	return out
}

// checkCancelSolLegs checks that every cancel's SOL leg is the rent its own
// order and reserve accounts released, positive and not a fee
func checkCancelSolLegs(t *testing.T, name string, tx *adapterTx, transfers []types.TransferData) *big.Int {
	t.Helper()
	cancels := limitCancels(t, tx)
	if len(cancels) == 0 {
		t.Fatalf("%s: no Limit cancel instruction", name)
	}
	total := new(big.Int)
	for _, c := range cancels {
		var legs []types.TransferData
		for _, tr := range transfers {
			if tr.Idx == c.idx && tr.Info.Mint == solMint {
				legs = append(legs, tr)
			}
		}
		if len(legs) != 1 {
			t.Errorf("%s %s: %d SOL legs, want 1", name, c.idx, len(legs))
			continue
		}
		if legs[0].Info.TokenAmount.Amount != c.released.String() || legs[0].IsFee {
			t.Errorf("%s %s: SOL leg %s IsFee=%v, want the %s lamports of the closed order and reserve, not a fee",
				name, c.idx, legs[0].Info.TokenAmount.Amount, legs[0].IsFee, c.released)
		}
		total.Add(total, bigStr(legs[0].Info.TokenAmount.Amount))
	}
	for _, tr := range transfers {
		if v, ok := new(big.Int).SetString(tr.Info.TokenAmount.Amount, 10); !ok || v.Sign() < 0 {
			t.Errorf("%s %s: %s amount %q", name, tr.Idx, tr.Type, tr.Info.TokenAmount.Amount)
		}
	}
	return total
}

// TestFinalLimitBatchCancelSolLegs: a Limit v2 transaction with five
// cancel_order instructions reported the signer's whole-transaction SOL
// change (27584176) on each of them, flagged IsFee: 5x the real amount, and a
// refund counted as a fee. Truth: each cancel's SOL leg is its order and
// reserve rent (3480000 + 2039280), and the legs add up to the signer's net
// SOL change plus the fee. robust-verify-1.
func TestFinalLimitBatchCancelSolLegs(t *testing.T) {
	const sig = "4ciwk55AB5jjV3YrZX6heuDMPgEyN1Hp8PAfFCRzDUWJKn9R4CraUJsXkTK7sseCjV2imJd63SdQvr6KoWh8Dvgm"
	tx, res := parseFixture(t, sig, nil)
	total := checkCancelSolLegs(t, sig[:8], tx, res.Transfers)
	signer := rawAccountKeys(tx)[0]
	want := new(big.Int).Add(lamportDelta(tx, signer), new(big.Int).SetUint64(tx.Meta.Fee))
	if total.Cmp(want) != 0 {
		t.Errorf("SOL legs sum %s, want the signer's net change plus the fee %s", total, want)
	}
}

// TestFinalLimitCancelSolLegNotNegative: the SOL leg of a cancel was the
// signer's net SOL change, negative once the transaction's fee exceeds the
// rent refund. The transaction is the real Limit v2 cancel 3CbJs3hE with its
// fee raised by 0.005 SOL (meta.fee and the signer's post balance), as no
// real high-fee cancel is in the corpus; the Limit v1 cancel 4gjGYCX2 is
// checked unchanged. robust-1.
func TestFinalLimitCancelSolLegNotNegative(t *testing.T) {
	highFee := cloneTx(t, loadFixture(t, "3CbJs3hEW1h27bicr2YUo7CeKrb9Nx3jnwNoMProwsByoTSSRJqpKieE68YiHKqbPCTQ7HRRngU26Uo61Zhm2Xys"))
	highFee.Meta.Fee += 5000000
	highFee.Meta.PostBalances[0] -= 5000000
	if lamportDelta(highFee, rawAccountKeys(highFee)[0]).Sign() >= 0 {
		t.Fatal("the high-fee variant should leave the signer with a net SOL loss")
	}
	checkCancelSolLegs(t, "3CbJs3hE high fee", highFee, dexparser.NewDexParser().ParseAll(highFee, nil).Transfers)

	v1 := loadFixture(t, "4gjGYCX2t4dvjr1C6Nh4kbphhf8SptBfwJ5Jxe7296taktxuKG1USHTEQ8KUiaC2LTWN5Kddf13LNppi8V85moer")
	checkCancelSolLegs(t, "4gjGYCX2", v1, dexparser.NewDexParser().ParseAll(v1, nil).Transfers)
}

// TestFinalTypedTransfersUnderRouter: ParseAll ran a transfer parser only for
// the first known DEX program of the transaction (DexInfo), so a Limit
// order cancel, DCA or fee claim that runs after another router got the
// untyped transfer list. No such real transaction is in the corpus (3102
// cached transactions give the same output either way): the transaction is
// the real Limit v2 cancel 3CbJs3hE with its first outer instruction (an
// idempotent ATA create without inner instructions) made an instruction of
// the legacy OKX router (OKX_DEX), a known DEX program without a transfer
// parser that now comes first. Truth: the same typed cancelOrder transfers
// as the unmodified transaction. E2 of the final brief.
func TestFinalTypedTransfersUnderRouter(t *testing.T) {
	real := loadFixture(t, "3CbJs3hEW1h27bicr2YUo7CeKrb9Nx3jnwNoMProwsByoTSSRJqpKieE68YiHKqbPCTQ7HRRngU26Uo61Zhm2Xys")
	want := dexparser.NewDexParser().ParseAll(real, nil).Transfers
	if len(want) == 0 || want[0].Type != "cancelOrder" {
		t.Fatalf("fixture: transfers %+v", want)
	}

	tx := cloneTx(t, real)
	for _, set := range tx.Meta.InnerInstructions {
		if set.Index == 0 {
			t.Fatal("fixture: outer 0 has inner instructions")
		}
	}
	tx.Transaction.Message.AccountKeys = append(tx.Transaction.Message.AccountKeys, adapter.AccountKey{Pubkey: constants.DEX_PROGRAMS.OKX_DEX.ID})
	tx.Transaction.Message.Instructions[0].(map[string]interface{})["programIdIndex"] = float64(len(tx.Transaction.Message.AccountKeys) - 1)

	res := dexparser.NewDexParser().ParseAll(tx, nil)
	ctx := newParseContext(tx, nil)
	if ctx.DexInfo.ProgramId != constants.DEX_PROGRAMS.OKX_DEX.ID {
		t.Fatalf("DexInfo %+v, want the OKX router program first", ctx.DexInfo)
	}
	if len(res.Transfers) != len(want) {
		t.Fatalf("%d transfers %+v, want the %d typed cancelOrder transfers", len(res.Transfers), res.Transfers, len(want))
	}
	for i := range want {
		g, w := res.Transfers[i], want[i]
		if g.Type != w.Type || g.Idx != w.Idx || g.Info.Mint != w.Info.Mint || g.Info.TokenAmount.Amount != w.Info.TokenAmount.Amount {
			t.Errorf("transfer %d: %s %s %s %s, want %s %s %s %s", i, g.Type, g.Idx, g.Info.TokenAmount.Amount, g.Info.Mint, w.Type, w.Idx, w.Info.TokenAmount.Amount, w.Info.Mint)
		}
	}
}
