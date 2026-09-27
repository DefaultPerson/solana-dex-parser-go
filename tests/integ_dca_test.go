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

// A user's own DCA deposit (top-up) and withdraw instructions emit Deposit
// and Withdraw (user_withdraw true); they are reported as DepositDca and
// WithdrawDca transfers. Synthetic, as no user deposit or withdraw was found
// on mainnet (fills pay out automatically): the real open_dca_v2 of
// 4PvkHqhg... (outer 5, its 500005000 WSOL transfer 5-11 into the DCA) with
// the instruction and its Opened event rewritten as deposit and Deposit, and
// the real fill 2TqzmwGq... with its transfer instruction (outer 4) and
// Withdraw event rewritten as the user's withdraw of the output.
func TestIntegDCAUserDepositWithdraw(t *testing.T) {
	d := constants.DISCRIMINATORS.JUPITER_DCA
	u64 := func(v uint64) []byte { b := make([]byte, 8); binary.LittleEndian.PutUint64(b, v); return b }
	cat := func(parts ...[]byte) []byte {
		var out []byte
		for _, p := range parts {
			out = append(out, p...)
		}
		return out
	}

	// Deposit
	tx := cloneTx(t, loadFixture(t, "4PvkHqhgTJa61cChu52gCyPcBK1rXGoKSJk4vRStXteXNTvuw7o9VoH5aqAgoYKGjQWSVdNvpwfEbDvHi7tZQZqw"))
	keys := rawAccountKeys(tx)
	open := tx.Transaction.Message.Instructions[5].(map[string]interface{})
	openData, _ := base58.Decode(open["data"].(string))
	if !constants.MatchDiscriminator(openData, d.OPEN_DCA_V2) {
		t.Fatal("outer 5 is not open_dca_v2")
	}
	// open_dca_v2: dca 0, user 1, payer 2, input_mint 3, output_mint 4,
	// user_ata 5, in_ata 6, out_ata 7, ..., token_program 9,
	// event_authority 11, program 12; deposit: user, dca, in_ata,
	// user_in_ata, token_program, event_authority, program
	acc := open["accounts"].([]interface{})
	const amount = 500005000
	open["accounts"] = []interface{}{acc[1], acc[0], acc[6], acc[5], acc[9], acc[11], acc[12]}
	open["data"] = base58.Encode(cat(d.DEPOSIT, u64(amount)))
	dcaKey, _ := base58.Decode(keys[jsonInt(acc[0])])
	for _, set := range tx.Meta.InnerInstructions {
		for _, ix := range set.Instructions {
			m := ix.(map[string]interface{})
			if data, _ := base58.Decode(m["data"].(string)); set.Index == 5 && constants.MatchDiscriminator(data, d.OPENED_EVENT) {
				m["data"] = base58.Encode(cat(d.DEPOSIT_EVENT, dcaKey, u64(amount)))
			}
		}
	}
	transfers := dexparser.NewDexParser().ParseTransfers(tx, nil)
	if len(transfers) != 1 {
		t.Fatalf("deposit: %d transfers %+v, want 1", len(transfers), transfers)
	}
	if tr := transfers[0]; tr.Type != "DepositDca" || tr.Idx != "5" || tr.Info.TokenAmount.Amount != "500005000" ||
		tr.Info.Mint != keys[jsonInt(acc[3])] || tr.Info.Source != keys[jsonInt(acc[5])] || tr.Info.Destination != keys[jsonInt(acc[6])] {
		t.Errorf("deposit transfer %s %s %s %s %s -> %s", tr.Type, tr.Idx, tr.Info.TokenAmount.Amount, tr.Info.Mint, tr.Info.Source, tr.Info.Destination)
	}

	// Withdraw
	tx = cloneTx(t, loadFixture(t, "2TqzmwGqQrb7uHQEKm1TxhyHQJan7YhmaguP7cM9o7KNMX4u1ERWkwjuCzJvLv3BfgwSK3X9AfLNzZmSsEeUd3YF"))
	keys = rawAccountKeys(tx)
	transfer := tx.Transaction.Message.Instructions[4].(map[string]interface{})
	fill := tx.Transaction.Message.Instructions[1].(map[string]interface{})
	transferData, _ := base58.Decode(transfer["data"].(string))
	if !constants.MatchDiscriminator(transferData, d.TRANSFER) {
		t.Fatal("outer 4 is not the DCA transfer")
	}
	// transfer: keeper 0, dca 1, user 2, output_mint 3, dca_out_ata 4,
	// user_out_ata 5, intermediate 6, system 7, token 8, ata 9,
	// event_authority 10, program 11; the fill's input_mint is its account 2.
	// withdraw: user 0, dca 1, input_mint 2, output_mint 3, dca_ata 4,
	// user_in_ata 5, user_out_ata 6, system 7, token 8, ata 9,
	// event_authority 10, program 11
	acc = transfer["accounts"].([]interface{})
	inputMint := fill["accounts"].([]interface{})[2]
	transfer["accounts"] = []interface{}{acc[2], acc[1], inputMint, acc[3], acc[4], acc[6], acc[5], acc[7], acc[8], acc[9], acc[10], acc[11]}
	var out uint64
	for _, set := range tx.Meta.InnerInstructions {
		for _, ix := range set.Instructions {
			m := ix.(map[string]interface{})
			if data, _ := base58.Decode(m["data"].(string)); set.Index == 4 && constants.MatchDiscriminator(data, d.WITHDRAW_EVENT) && len(data) >= 16+49 {
				out = binary.LittleEndian.Uint64(data[56:64])
				data[64] = 1 // user_withdraw
				m["data"] = base58.Encode(data)
			}
		}
	}
	transfer["data"] = base58.Encode(cat(d.WITHDRAW, u64(out), []byte{1})) // Withdrawal::Out
	var withdraws []string
	for _, tr := range dexparser.NewDexParser().ParseTransfers(tx, nil) {
		if tr.Type != "WithdrawDca" {
			continue
		}
		withdraws = append(withdraws, tr.Idx)
		if tr.Idx != "4" || tr.Info.TokenAmount.Amount != strconv.FormatUint(out, 10) || tr.Info.Mint != keys[jsonInt(acc[3])] ||
			tr.Info.Source != keys[jsonInt(acc[4])] || tr.Info.Destination != keys[jsonInt(acc[5])] {
			t.Errorf("withdraw transfer %s %s %s %s -> %s", tr.Idx, tr.Info.TokenAmount.Amount, tr.Info.Mint, tr.Info.Source, tr.Info.Destination)
		}
	}
	if len(withdraws) != 1 || out == 0 {
		t.Errorf("withdraw: WithdrawDca at %v (out %d), want one at 4", withdraws, out)
	}
}

// Limit Order v1 cancel_expired_order (v1 IDL: the accounts of cancel_order)
// is reported like cancel_order, as cancelExpiredOrder transfers; it was
// ignored. Synthetic: no cancel_expired_order was found in about 200
// sampled Limit v1 transactions (2026-09), so the real cancel_order
// 4gjGYCX2... gets the cancel_expired_order discriminator.
func TestIntegLimitV1CancelExpiredOrder(t *testing.T) {
	const sig = "4gjGYCX2t4dvjr1C6Nh4kbphhf8SptBfwJ5Jxe7296taktxuKG1USHTEQ8KUiaC2LTWN5Kddf13LNppi8V85moer"
	want := dexparser.NewDexParser().ParseAll(loadFixture(t, sig), nil).Transfers
	if len(want) == 0 || want[0].Type != "cancelOrder" {
		t.Fatalf("cancel_order transfers %+v", want)
	}
	tx := cloneTx(t, loadFixture(t, sig))
	ix := tx.Transaction.Message.Instructions[0].(map[string]interface{})
	data, _ := base58.Decode(ix["data"].(string))
	if !constants.MatchDiscriminator(data, constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER.CANCEL_ORDER) {
		t.Fatal("outer 0 is not cancel_order")
	}
	ix["data"] = base58.Encode(append(append([]byte{}, constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER.CANCEL_EXPIRED_ORDER...), data[8:]...))
	got := dexparser.NewDexParser().ParseAll(tx, nil).Transfers
	if len(got) != len(want) {
		t.Fatalf("%d transfers, want %d like cancel_order", len(got), len(want))
	}
	for i := range got {
		g, w := got[i], want[i]
		if g.Type != "cancelExpiredOrder" || g.Info.Mint != w.Info.Mint || g.Info.TokenAmount.Amount != w.Info.TokenAmount.Amount ||
			g.Info.Source != w.Info.Source || g.Info.Destination != w.Info.Destination {
			t.Errorf("transfer %d: %s %s %s, want cancelExpiredOrder %s %s", i, g.Type, g.Info.TokenAmount.Amount, g.Info.Mint, w.Info.TokenAmount.Amount, w.Info.Mint)
		}
	}
}
