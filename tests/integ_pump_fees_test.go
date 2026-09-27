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
