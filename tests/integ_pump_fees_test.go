package tests

import (
	"encoding/binary"
	"math/big"
	"strconv"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// pumpEvent returns the data (after the 16-byte prefix) of the first
// self-CPI event with discriminator disc of the fixture, and its outer index
func pumpEvent(t *testing.T, tx *adapter.SolanaTransaction, disc []byte) ([]byte, int) {
	t.Helper()
	for _, x := range fixtureIxs(t, tx) {
		if (x.programId == constants.DEX_PROGRAMS.PUMP_FUN.ID || x.programId == constants.DEX_PROGRAMS.PUMP_SWAP.ID) &&
			constants.MatchDiscriminator(x.data, disc) {
			return x.data[16:], x.outer
		}
	}
	t.Fatalf("no event % x", disc)
	return nil, 0
}

// meme-15 (rest): Pump.fun and PumpSwap fee payouts (creator fee
// collection, cashback claims, creator-fee and holder distributions) were
// not reported. They are transfers with a Type naming the payout (see
// pumpfun.PumpFeeClaimParser); a MemeEvent type would not fit: they are no
// step of a coin's life. Real transactions found through coin creators'
// histories. Truth: the programs' events and the balance changes.
func TestIntegPumpFeePayouts(t *testing.T) {
	pf, ps := constants.DISCRIMINATORS.PUMPFUN, constants.DISCRIMINATORS.PUMPSWAP
	u64 := func(b []byte) string { return strconv.FormatUint(binary.LittleEndian.Uint64(b), 10) }
	parse := func(sig string) (*adapter.SolanaTransaction, []types.TransferData) {
		tx := loadFixture(t, sig)
		r := dexparser.NewDexParser().ParseAll(tx, nil)
		if len(r.Trades) != 0 || len(r.Liquidities) != 0 {
			t.Errorf("%s: %d trades, %d liquidity events", sig[:8], len(r.Trades), len(r.Liquidities))
		}
		return tx, r.Transfers
	}
	one := func(sig string, transfers []types.TransferData, typ string) types.TransferData {
		var found []types.TransferData
		for _, tr := range transfers {
			if tr.Type == typ {
				found = append(found, tr)
			}
		}
		if len(found) != 1 {
			t.Fatalf("%s: %d %s transfers in %+v", sig[:8], len(found), typ, transfers)
		}
		return found[0]
	}

	// collect_creator_fee (v1) and collect_creator_fee_v2 with a SOL quote:
	// CollectCreatorFeeEvent (timestamp, creator, creator_fee, quote_mint),
	// paid in lamports out of the creator vault
	for _, sig := range []string{
		"5MUftpmsoS5S5WPzXjkLgaSQEpAfgAHv7V4uTHedwuBUJV2fHscpXaF2BJr1xffP5je2CLLXWfujH5nZA4AH1MBL",
		"5ZS43kwu2JmcwiAw13iZZcXptUwQ1G1JqPKMCtZewBmVxefww9i2SqsMhTGGTXkfpt8Ak3RHmLC7nqEYUDD6Lstq",
	} {
		tx, transfers := parse(sig)
		ev, outer := pumpEvent(t, tx, pf.COLLECT_CREATOR_FEE_EVENT)
		tr := one(sig, transfers, "collectCreatorFee")
		creator := base58.Encode(ev[8:40])
		if tr.ProgramId != constants.DEX_PROGRAMS.PUMP_FUN.ID || tr.Idx != strconv.Itoa(outer) || tr.Info.Mint != constants.TOKENS.SOL ||
			tr.Info.TokenAmount.Amount != u64(ev[40:48]) || tr.Info.Destination != creator ||
			new(big.Int).Neg(lamportDelta(tx, tr.Info.Source)).String() != tr.Info.TokenAmount.Amount {
			t.Errorf("%s: collectCreatorFee %s %s %s -> %s, event %s to %s, vault change %d", sig[:8], tr.Idx, tr.Info.TokenAmount.Amount,
				tr.Info.Source, tr.Info.Destination, u64(ev[40:48]), creator, lamportDelta(tx, tr.Info.Source))
		}
	}

	// PumpSwap collect_coin_creator_fee: CollectCoinCreatorFeeEvent
	// (timestamp, coin_creator, coin_creator_fee, vault ATA, creator token
	// account)
	{
		const sig = "5ZS43kwu2JmcwiAw13iZZcXptUwQ1G1JqPKMCtZewBmVxefww9i2SqsMhTGGTXkfpt8Ak3RHmLC7nqEYUDD6Lstq"
		tx, transfers := parse(sig)
		ev, outer := pumpEvent(t, tx, ps.COLLECT_COIN_CREATOR_FEE_EVENT)
		tr := one(sig, transfers, "collectCoinCreatorFee")
		if tr.ProgramId != constants.DEX_PROGRAMS.PUMP_SWAP.ID || tr.Idx != strconv.Itoa(outer) || tr.Info.TokenAmount.Amount != u64(ev[40:48]) ||
			tr.Info.Source != base58.Encode(ev[48:80]) || tr.Info.Destination != base58.Encode(ev[80:112]) {
			t.Errorf("collectCoinCreatorFee %+v", tr)
		}
	}

	// claim_cashback_v2 moves lamports without a transfer instruction:
	// ClaimCashbackEvent (user, amount) paid out of the user volume
	// accumulator (account 1)
	{
		const sig = "4qzEV5jjQnj5JYH5kNyNnNB1xDXYCEfhkwpns1tcnytAEz94CH6MTiUDfGGpSABrYHX9DFYR5veu3AH3tSCF7HZV"
		tx, transfers := parse(sig)
		ev, outer := pumpEvent(t, tx, pf.CLAIM_CASHBACK_EVENT)
		tr := one(sig, transfers, "claimCashback")
		if tr.Idx != strconv.Itoa(outer) || tr.Info.Mint != constants.TOKENS.SOL || tr.Info.TokenAmount.Amount != u64(ev[32:40]) ||
			tr.Info.Destination != base58.Encode(ev[0:32]) || new(big.Int).Neg(lamportDelta(tx, tr.Info.Source)).String() != tr.Info.TokenAmount.Amount {
			t.Errorf("claimCashback %s %s -> %s, event %s, source change %d", tr.Info.TokenAmount.Amount, tr.Info.Source, tr.Info.Destination,
				u64(ev[32:40]), lamportDelta(tx, tr.Info.Source))
		}
	}

	// Distributions report every transfer of the instruction; they add up to
	// the event's total. DistributeFeeToHoldersEvent: timestamp, mint,
	// quote_mint, recipients u64, total u64. DistributeCreatorFeesEvent:
	// timestamp, mint, bonding_curve, sharing_config, admin, shareholders
	// (vec of pubkey + u16), distributed u64, quote_mint. The PumpSwap
	// transfer_creator_fees_to_pump before it moved the same SOL into the
	// Pump.fun creator vault.
	sum := func(transfers []types.TransferData, typ string) string {
		total := new(big.Int)
		for _, tr := range transfers {
			if tr.Type == typ {
				v, _ := new(big.Int).SetString(tr.Info.TokenAmount.Amount, 10)
				total.Add(total, v)
			}
		}
		return total.String()
	}
	{
		const sig = "56rR1vUtPE4jf9Mq6f2JeDY4ZkEjG44Nw39N6TUqg8AUTcizj9q5cWBGUEtt49dUJV6dwqznVLuTvpryxz1Dknx4"
		tx, transfers := parse(sig)
		ev, _ := pumpEvent(t, tx, pf.DISTRIBUTE_FEE_TO_HOLDERS_EVENT)
		if got := sum(transfers, "distributeFeeToHolders"); got != u64(ev[80:88]) || got == "0" {
			t.Errorf("distributeFeeToHolders total %s, event %s", got, u64(ev[80:88]))
		}
	}
	{
		const sig = "2rEQeEV6Jgetkg8BeaHXJSD4cc4qsJBLd7DxEqzzHYbpR9DBNHjrLuDS3a4aETHDDvFrPSmwXcsMVPBjJj2GQbYM"
		tx, transfers := parse(sig)
		ev, _ := pumpEvent(t, tx, pf.DISTRIBUTE_CREATOR_FEES_EVENT)
		n := int(binary.LittleEndian.Uint32(ev[136:140]))
		distributed := u64(ev[140+n*34 : 148+n*34])
		if got := sum(transfers, "distributeCreatorFees"); got != distributed || got == "0" {
			t.Errorf("distributeCreatorFees total %s, event %s", got, distributed)
		}
		if got := sum(transfers, "transferCreatorFeesToPump"); got != distributed {
			t.Errorf("transferCreatorFeesToPump %s, want %s", got, distributed)
		}
	}
}

// PumpSwap claim_cashback pays the user's WSOL cashback by a token
// transfer from the user volume accumulator's WSOL account (account 4) to
// the user's WSOL account (account 5); it is reported as claimCashback, next
// to a Pump.fun claim_cashback_v2 (lamports, no transfer instruction) in the
// same transaction. Real transactions of one user, found through the
// history of that user's PumpSwap cashback WSOL account. Truth: the
// ClaimCashbackEvent amounts and the balance changes.
func TestIntegPumpSwapClaimCashback(t *testing.T) {
	const (
		user      = "25jZ7EwnKfZo2DZgHM27pbU5Tf54PYG8jc7qNL3gtkxG"
		cashback  = "2bdTTjPUVChqxVeNSNwgF2xpBm8PpKiasNi2ZKdkcEid" // PumpSwap claim_cashback account 4
		userWSOL  = "Bh2mmErdC8uRpSbLh1YP2nodxaSh1sz2JMsN2UmAupz6" // account 5
		pfAccount = "Ckiz4e7t3QSV4PSikRDxTQvbigkCchiAEDxaAyCuqH5D" // Pump.fun user volume accumulator
	)
	events := func(tx *adapter.SolanaTransaction) map[string]string {
		amounts := map[string]string{}
		for _, x := range fixtureIxs(t, tx) {
			if constants.MatchDiscriminator(x.data, constants.DISCRIMINATORS.PUMPFUN.CLAIM_CASHBACK_EVENT) &&
				len(x.data) >= 56 && base58.Encode(x.data[16:48]) == user {
				amounts[x.programId] = strconv.FormatUint(binary.LittleEndian.Uint64(x.data[48:56]), 10)
			}
		}
		return amounts
	}
	cases := []struct {
		sig      string
		pumpswap string // idx of the PumpSwap claim
		pumpfun  string // idx of the Pump.fun claim, "" when none
	}{
		{"634UbC8itcvvrvPcJW7GC9FyomJSUFEYRSiGfoiM8PKMc5qvVVvTbWLVEasTYxpxUHBcdm9nCCbKhCKwJvC9cK9J", "3", ""},
		{"21hgrduf2xb2jU2ZmZakFzh3nc1yEfTWHfCrwpwj2Pw5oHqr8PB2vfy9h8oFXorJBcKWw8Yoe8sKu8sccGLMxnfd", "4", "3"},
	}
	for _, tc := range cases {
		tx := loadFixture(t, tc.sig)
		r := dexparser.NewDexParser().ParseAll(tx, nil)
		if len(r.Trades) != 0 || len(r.Liquidities) != 0 {
			t.Errorf("%s: %d trades, %d liquidity events", tc.sig[:8], len(r.Trades), len(r.Liquidities))
		}
		amounts := events(tx)
		got := map[string]types.TransferData{}
		for _, tr := range r.Transfers {
			if tr.Type == "claimCashback" {
				got[tr.ProgramId] = tr
			}
		}
		want := 1
		if tc.pumpfun != "" {
			want = 2
		}
		if len(got) != want || len(amounts) != want {
			t.Fatalf("%s: claimCashback transfers %+v, events %v", tc.sig[:8], got, amounts)
		}

		ps := got[constants.DEX_PROGRAMS.PUMP_SWAP.ID]
		amount := amounts[constants.DEX_PROGRAMS.PUMP_SWAP.ID]
		if ps.Idx != tc.pumpswap || ps.Info.Mint != constants.TOKENS.SOL || ps.Info.TokenAmount.Amount != amount ||
			ps.Info.Source != cashback || ps.Info.Destination != userWSOL ||
			new(big.Int).Neg(accountTokenDelta(tx, cashback, constants.TOKENS.SOL)).String() != amount {
			t.Errorf("%s: PumpSwap claimCashback %s %s %s %s -> %s, event %s, source change %s", tc.sig[:8], ps.Idx, ps.Info.TokenAmount.Amount,
				ps.Info.Mint, ps.Info.Source, ps.Info.Destination, amount, accountTokenDelta(tx, cashback, constants.TOKENS.SOL))
		}
		if tc.pumpfun == "" {
			continue
		}
		pf := got[constants.DEX_PROGRAMS.PUMP_FUN.ID]
		amount = amounts[constants.DEX_PROGRAMS.PUMP_FUN.ID]
		if pf.Idx != tc.pumpfun || pf.Info.Mint != constants.TOKENS.SOL || pf.Info.TokenAmount.Amount != amount ||
			pf.Info.Source != pfAccount || pf.Info.Destination != user ||
			new(big.Int).Neg(lamportDelta(tx, pfAccount)).String() != amount {
			t.Errorf("%s: Pump.fun claimCashback %s %s %s -> %s, event %s, source change %s", tc.sig[:8], pf.Idx, pf.Info.TokenAmount.Amount,
				pf.Info.Source, pf.Info.Destination, amount, lamportDelta(tx, pfAccount))
		}
	}
}

// claim_token_incentives of either program pays the user's token
// incentives (ClaimTokenIncentivesEvent: user, mint, amount, ...) by a token
// transfer from the global incentive token account (account 3) to the
// user's token account (account 1); it is reported as claimTokenIncentives.
// Synthetic: no claim was found on mainnet (the global incentive token
// accounts of both programs for the PUMP mint, derived with the IDL seeds,
// had no successful transaction on 2026-09-27). Built from the real
// PumpSwap claim_cashback 634UbC8i..., whose outer 3 transfers 4633813 WSOL
// from a program-owned token account to the user's: the instruction gets
// the claim_token_incentives discriminator and account order, its event
// becomes a ClaimTokenIncentivesEvent, and WSOL becomes the PUMP mint (6
// decimals, as in the real 24U4tyX7...). For Pump.fun the program id is
// swapped as well.
func TestIntegPumpClaimTokenIncentives(t *testing.T) {
	const (
		sig      = "634UbC8itcvvrvPcJW7GC9FyomJSUFEYRSiGfoiM8PKMc5qvVVvTbWLVEasTYxpxUHBcdm9nCCbKhCKwJvC9cK9J"
		pumpMint = "pumpCmXqMfrsAkQ5r49WcJnRayYRqmXz6ae8H7H9Dfn"
		user     = "25jZ7EwnKfZo2DZgHM27pbU5Tf54PYG8jc7qNL3gtkxG"
		source   = "2bdTTjPUVChqxVeNSNwgF2xpBm8PpKiasNi2ZKdkcEid"
		userATA  = "Bh2mmErdC8uRpSbLh1YP2nodxaSh1sz2JMsN2UmAupz6"
		amount   = 4633813
	)
	u64 := func(v uint64) []byte { b := make([]byte, 8); binary.LittleEndian.PutUint64(b, v); return b }
	for _, program := range []constants.DexProgram{constants.DEX_PROGRAMS.PUMP_SWAP, constants.DEX_PROGRAMS.PUMP_FUN} {
		tx := cloneTx(t, loadFixture(t, sig))
		rename := func(from, to string) {
			for i := range tx.Transaction.Message.AccountKeys {
				if tx.Transaction.Message.AccountKeys[i].Pubkey == from {
					tx.Transaction.Message.AccountKeys[i].Pubkey = to
				}
			}
			for _, list := range [][]string{tx.Meta.LoadedAddresses.Writable, tx.Meta.LoadedAddresses.Readonly} {
				for i := range list {
					if list[i] == from {
						list[i] = to
					}
				}
			}
		}
		rename(constants.TOKENS.SOL, pumpMint)
		rename(constants.DEX_PROGRAMS.PUMP_SWAP.ID, program.ID)
		for _, balances := range [][]adapter.TokenBalance{tx.Meta.PreTokenBalances, tx.Meta.PostTokenBalances} {
			for i := range balances {
				if balances[i].Mint == constants.TOKENS.SOL {
					balances[i].Mint, balances[i].UiTokenAmount.Decimals, balances[i].UiTokenAmount.UIAmount = pumpMint, 6, nil
				}
			}
		}

		var claim, event, transfer *rawIx
		ixs := fixtureIxs(t, tx)
		for i := range ixs {
			x := &ixs[i]
			switch {
			case x.inner == -1 && x.outer == 3:
				claim = x
			case x.outer == 3 && x.programId == program.ID:
				event = x
			case x.outer == 3 && x.programId == constants.TOKEN_PROGRAM_ID:
				transfer = x
			}
		}
		if claim == nil || event == nil || transfer == nil || !constants.MatchDiscriminator(claim.data, constants.DISCRIMINATORS.PUMPSWAP.CLAIM_CASHBACK) ||
			len(transfer.data) != 10 || binary.LittleEndian.Uint64(transfer.data[1:9]) != amount {
			t.Fatal("outer 3 is not the claim_cashback with its 4633813 WSOL transfer")
		}
		// claim_cashback: user 0, user_volume_accumulator 1, quote_mint 2,
		// quote_token_program 3, accumulator WSOL 4, user WSOL 5, system 6,
		// event_authority 7, program 8. claim_token_incentives: user,
		// user_ata, global_volume_accumulator, global_incentive_token_account,
		// user_volume_accumulator, mint, token_program, system,
		// associated_token_program, event_authority, program, payer.
		acc := claim.m["accounts"].([]interface{})
		ataProgram := tx.Transaction.Message.Instructions[2].(map[string]interface{})["programIdIndex"]
		claim.m["accounts"] = []interface{}{acc[0], acc[5], acc[1], acc[4], acc[1], acc[2], acc[3], acc[6], ataProgram, acc[7], acc[8], acc[0]}
		claim.m["data"] = base58.Encode(constants.DISCRIMINATORS.PUMPFUN.CLAIM_TOKEN_INCENTIVES)
		transfer.data[9] = 6 // transfer_checked decimals
		transfer.m["data"] = base58.Encode(transfer.data)
		userKey, _ := base58.Decode(user)
		mintKey, _ := base58.Decode(pumpMint)
		var data []byte
		for _, part := range [][]byte{constants.DISCRIMINATORS.PUMPFUN.CLAIM_TOKEN_INCENTIVES_EVENT, userKey, mintKey, u64(amount), u64(uint64(*tx.BlockTime)), u64(amount), u64(0)} {
			data = append(data, part...)
		}
		event.m["data"] = base58.Encode(data)

		r := dexparser.NewDexParser().ParseAll(tx, nil)
		if len(r.Trades) != 0 || len(r.Liquidities) != 0 {
			t.Errorf("%s: %d trades, %d liquidity events", program.Name, len(r.Trades), len(r.Liquidities))
		}
		var found []types.TransferData
		for _, tr := range r.Transfers {
			if tr.Type == "claimTokenIncentives" {
				found = append(found, tr)
			}
		}
		if len(found) != 1 {
			t.Fatalf("%s: claimTokenIncentives transfers %+v in %+v", program.Name, found, r.Transfers)
		}
		if tr := found[0]; tr.ProgramId != program.ID || tr.Idx != "3" || tr.Info.Mint != pumpMint || tr.Info.TokenAmount.Amount != "4633813" ||
			tr.Info.TokenAmount.Decimals != 6 || tr.Info.Source != source || tr.Info.Destination != userATA {
			t.Errorf("%s: claimTokenIncentives %s %s %s %s (%d) %s -> %s", program.Name, tr.ProgramId, tr.Idx, tr.Info.TokenAmount.Amount,
				tr.Info.Mint, tr.Info.TokenAmount.Decimals, tr.Info.Source, tr.Info.Destination)
		}
	}
}
