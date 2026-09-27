package tests

import (
	"math/big"
	"testing"

	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/photon"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Photon shred decoding. Photon has no IDL; the pump_buy_v2, pump_sell_v2,
// collect_fee and moonshot_sell layouts were verified on real transactions
// (/tmp/sdp-audit/work/research/shred.md §1: the arguments are forwarded
// unchanged to the inner Pump.fun / Moonshot instruction, and the Photon fee
// equals the System transfer to the fee vault). The test checks both
// relations on the fixtures. shred-v1, shred-23, shred-31, constants-10.

const (
	// pump_buy_v2 at outer 2 (inner buy_exact_quote_in_v2 at 2-1)
	sigPhotonPumpBuyV2 = "2DbpcY2at3eWL3Mm6KBQbrSSMB9d4qMa5NsrpZyuFciVVs49poqLNFNv9mymZmpA9LKSKzcThGK4h5McBPU2bWKV"
	// pump_sell_v2 at outer 2 (inner sell_v2 at 2-0)
	sigPhotonPumpSellV2 = "2EpiLGAw4eZ5MvoJ5FmNXqCJ3NqaUpcTZoxhEe1We3WTGTeWKLbqaBnhTBfhZ5RK1bsSHgfE9aJwyqrGaosWE439"
	// moonshot_sell at outer 1
	sigPhotonMoonshotSell = "3zPVtpLHPFd3qns33nNyMTJynYVXah59miRWGuc7Hdsv9Y7RFyaBgu2dFzzCRKF4GwhkQus2ZmNhpo59geJktS3A"
	// collect_fee mode 0 (lamports) after a PumpSwap buy_exact_quote_in
	sigPhotonCollectFeeLamports = "1bdrt5y1Nz5tRygDaEBxTv8wCL8nPazSm6LRRpTPG74tX6DzPtmvx55iTqaX7Uc8B3x2q7oxZ5yc9hvPrQgUALv"
	// collect_fee mode 1 (bps) after a PumpSwap sell
	sigPhotonCollectFeeBps = "2F1tqd2pi7BRCEsffg3nvtQdfuA3dbwNaT8aSGtyjVd4p2y4YZh8ZfvTbWU4CqPjDAij9Qdg58GkX35paoH79XdE"
)

func photonTrade(t *testing.T, res *types.ParseShredResult, ix rawIx) types.ParsedShredInstruction {
	t.Helper()
	ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.PHOTON.ID, utils.FormatIdx(ix.outer, ix.inner))
	if ins.Trade == nil {
		t.Fatalf("%s: no trade", ins.Action)
	}
	return ins
}

// systemTransferTo returns the lamports of the inner System transfer to
// account within outer instruction outer
func systemTransferTo(t *testing.T, tx *adapter.SolanaTransaction, outer int, account string) uint64 {
	t.Helper()
	for _, ix := range fixtureIxs(t, tx) {
		if ix.outer == outer && ix.inner >= 0 && ix.programId == constants.SYSTEM_PROGRAM_ID && len(ix.data) == 12 && ix.data[0] == 2 && ix.accounts[1] == account {
			return le64At(ix.data, 4)
		}
	}
	t.Fatalf("no System transfer to %s in %d", account, outer)
	return 0
}

// TestShredPhotonPumpV2: Photon now calls pump_buy_v2 / pump_sell_v2, which
// were not decoded (and the old decoders assumed a leading timestamp).
// constants-10, shred-31.
func TestShredPhotonPumpV2(t *testing.T) {
	// pump_buy_v2: spendable_quote_in, min_tokens_out, photon_fee_lamports, flag
	tx := loadFixture(t, sigPhotonPumpBuyV2)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.PHOTON.ID, constants.DISCRIMINATORS.PHOTON.PUMPFUN_BUY_V2)
	inner := findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, constants.DISCRIMINATORS.PUMPFUN.BUY_EXACT_QUOTE_IN_V2)
	spend, minOut, fee := le64At(ix.data, 8), le64At(ix.data, 16), le64At(ix.data, 24)
	if spend != le64At(inner.data, 8) || minOut != le64At(inner.data, 16) {
		t.Fatalf("fixture: Photon args %d/%d differ from the inner Pump.fun args", spend, minOut)
	}
	if got := systemTransferTo(t, tx, ix.outer, ix.accounts[6]); got != fee {
		t.Fatalf("fixture: Photon fee transfer %d, want photon_fee_lamports %d", got, fee)
	}
	res := parseShred(t, tx, nil)
	ins := photonTrade(t, res, ix)
	tr := ins.Trade
	if tr.Type != types.TradeTypeBuy || tr.User != ix.accounts[0] || tr.Pool[0] != ix.accounts[16] ||
		tr.InputToken.Mint != ix.accounts[10] || tr.InputToken.AmountRaw != u64str(spend) || tr.InputToken.Decimals != 9 ||
		tr.OutputToken.Mint != ix.accounts[9] || tr.OutputToken.AmountRaw != u64str(minOut) || tr.OutputToken.Decimals != 6 {
		t.Errorf("pump_buy_v2 trade = %+v", tr)
	}
	if tr.ProgramId != constants.DEX_PROGRAMS.PUMP_FUN.ID || tr.Route != constants.DEX_PROGRAMS.PHOTON.Name || ins.InputAmountKind != types.ShredAmountExact || ins.OutputAmountKind != types.ShredAmountMin {
		t.Errorf("pump_buy_v2 program %s route %s kinds %s/%s", tr.ProgramId, tr.Route, ins.InputAmountKind, ins.OutputAmountKind)
	}
	if tr.Fee == nil || tr.Fee.AmountRaw != u64str(fee) || tr.Fee.Recipient != ix.accounts[6] {
		t.Errorf("pump_buy_v2 fee = %+v, want %d to %s", tr.Fee, fee, ix.accounts[6])
	}
	if got := ownerTokenDelta(tx, ix.accounts[0], ix.accounts[9]); got.Cmp(new(big.Int).SetUint64(minOut)) < 0 {
		t.Errorf("tokens received %s below min_tokens_out", got)
	}

	// pump_sell_v2: amount, min_sol_output, photon_fee_bps, flag
	tx = loadFixture(t, sigPhotonPumpSellV2)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.PHOTON.ID, constants.DISCRIMINATORS.PHOTON.PUMPFUN_SELL_V2)
	inner = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, constants.DISCRIMINATORS.PUMPFUN.SELL_V2)
	amount, minSol := le64At(ix.data, 8), le64At(ix.data, 16)
	if amount != le64At(inner.data, 8) || minSol != le64At(inner.data, 16) {
		t.Fatalf("fixture: Photon args differ from the inner sell_v2 args")
	}
	res = parseShred(t, tx, nil)
	ins = photonTrade(t, res, ix)
	tr = ins.Trade
	if tr.Type != types.TradeTypeSell || tr.InputToken.Mint != ix.accounts[9] || tr.InputToken.AmountRaw != u64str(amount) ||
		tr.OutputToken.Mint != ix.accounts[10] || tr.OutputToken.AmountRaw != u64str(minSol) || ins.InputAmountKind != types.ShredAmountExact {
		t.Errorf("pump_sell_v2 trade = %+v", tr)
	}
	if got := ownerTokenDelta(tx, ix.accounts[0], ix.accounts[9]); got.Cmp(new(big.Int).Neg(new(big.Int).SetUint64(amount))) != 0 {
		t.Errorf("tokens sold %s, want -%d", got, amount)
	}
	raw := res.Instructions["Photon"][0].(*photon.PhotonInstruction).Data.(*photon.PhotonPumpfunData)
	if raw.PhotonFeeBps != le64At(ix.data, 24) {
		t.Errorf("photon fee bps %d", raw.PhotonFeeBps)
	}
}

// TestShredPhotonCollectFee decodes collect_fee (timestamp, amount, mode):
// mode 0 amounts are the lamports transferred to the fee vault.
func TestShredPhotonCollectFee(t *testing.T) {
	tx := loadFixture(t, sigPhotonCollectFeeLamports)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.PHOTON.ID, constants.DISCRIMINATORS.PHOTON.COLLECT_FEE)
	res := parseShred(t, tx, nil)
	fee := oneTypedAt(t, res, constants.DEX_PROGRAMS.PHOTON.ID, utils.FormatIdx(ix.outer, -1)).Data.(*photon.PhotonCollectFeeData)
	if fee.Mode != 0 || fee.Amount != systemTransferTo(t, tx, ix.outer, ix.accounts[2]) || fee.User != ix.accounts[0] || fee.FeeVault != ix.accounts[2] {
		t.Errorf("collect_fee = %+v", fee)
	}

	tx = loadFixture(t, sigPhotonCollectFeeBps)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.PHOTON.ID, constants.DISCRIMINATORS.PHOTON.COLLECT_FEE)
	res = parseShred(t, tx, nil)
	fee = oneTypedAt(t, res, constants.DEX_PROGRAMS.PHOTON.ID, utils.FormatIdx(ix.outer, -1)).Data.(*photon.PhotonCollectFeeData)
	if fee.Mode != 1 || fee.Amount != le64At(ix.data, 16) {
		t.Errorf("collect_fee bps = %+v", fee)
	}
}

// TestShredPhotonMoonshot: the Moonit sell path hardcoded 6 base decimals
// (the Moonshot mints have 9), and buys reported the token as input and SOL
// as output. shred-23.
func TestShredPhotonMoonshot(t *testing.T) {
	tx := loadFixture(t, sigPhotonMoonshotSell)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.PHOTON.ID, constants.DISCRIMINATORS.PHOTON.MOONIT_SELL)
	tokens, collateral := le64At(ix.data, 16), le64At(ix.data, 24)
	res := parseShred(t, tx, nil)
	ins := photonTrade(t, res, ix)
	tr := ins.Trade
	if tr.Type != types.TradeTypeSell || tr.InputToken.Mint != ix.accounts[7] || tr.InputToken.AmountRaw != u64str(tokens) || tr.InputToken.Decimals != mintDecimals(tx, ix.accounts[7]) ||
		tr.OutputToken.Mint != solMint || tr.OutputToken.AmountRaw != u64str(collateral) || tr.OutputToken.Decimals != 9 || tr.Pool[0] != ix.accounts[2] {
		t.Errorf("moonshot_sell = %+v", tr)
	}
	if got := ownerTokenDelta(tx, ix.accounts[0], ix.accounts[7]); got.Cmp(new(big.Int).Neg(new(big.Int).SetUint64(tokens))) != 0 {
		t.Errorf("tokens sold %s, want -%d", got, tokens)
	}

	// No Photon moonshot_buy exists in the fixtures or the 859 sampled Photon
	// and Moonshot transactions; the test re-tags the real moonshot_sell.
	// Whatever its field layout, a BUY spends SOL for the token.
	tx = preExec(t, tx)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.PHOTON.ID, constants.DISCRIMINATORS.PHOTON.MOONIT_SELL)
	setIxData(ix, append(append([]byte{}, constants.DISCRIMINATORS.PHOTON.MOONIT_BUY...), ix.data[8:]...))
	tr = photonTrade(t, parseShred(t, tx, nil), ix).Trade
	if tr.Type != types.TradeTypeBuy || tr.InputToken.Mint != solMint || tr.InputToken.Decimals != 9 || tr.OutputToken.Mint != ix.accounts[7] ||
		tr.InputToken.AmountRaw != u64str(collateral) || tr.OutputToken.AmountRaw != u64str(tokens) {
		t.Errorf("moonshot_buy = %s %s %s -> %s %s", tr.Type, tr.InputToken.Mint, tr.InputToken.AmountRaw, tr.OutputToken.Mint, tr.OutputToken.AmountRaw)
	}
}

// TestShredPhotonPumpswapDirection: PUMPSWAP_TRADE compared the trade type
// with "sell" while GetAccountTradeType returns "SELL", so every sell was a
// BUY with swapped mints. pump_amm_swap is gone from the deployed program
// and no real call exists, so the test appends one (IDL-less layout of the
// decoder: 0 pool, 1 user, 3 base mint, 4 quote mint, 5 input and 6 output
// token accounts, 16 PumpSwap) to a real transaction, with the input being
// the user's base-mint ATA. shred-v1.
func TestShredPhotonPumpswapDirection(t *testing.T) {
	tx := preExec(t, loadFixture(t, sigPswapSell))
	sell := findIx(t, tx, constants.DEX_PROGRAMS.PUMP_SWAP.ID, constants.DISCRIMINATORS.PUMPSWAP.SELL)
	user, base, quote := sell.accounts[1], sell.accounts[3], sell.accounts[4]
	baseAta := sell.accounts[5]
	if std, t22, _ := utils.FindAssociatedTokenAddress(user, base); baseAta != std && baseAta != t22 {
		t.Fatalf("fixture: user base account is not the ATA")
	}
	keys := rawAccountKeys(tx)
	index := func(k string) int {
		for i, key := range keys {
			if key == k {
				return i
			}
		}
		t.Fatalf("key %s not in the tx", k)
		return -1
	}
	tx.Transaction.Message.AccountKeys = append(tx.Transaction.Message.AccountKeys, adapter.AccountKey{Pubkey: constants.DEX_PROGRAMS.PHOTON.ID})
	photonIdx := len(tx.Transaction.Message.AccountKeys) - 1
	accounts := make([]interface{}, 17)
	for i := range accounts {
		accounts[i] = index(sell.accounts[0])
	}
	accounts[1], accounts[3], accounts[4], accounts[5], accounts[6] = index(user), index(base), index(quote), index(baseAta), index(sell.accounts[6])
	accounts[16] = index(constants.DEX_PROGRAMS.PUMP_SWAP.ID)
	data := append(append([]byte{}, constants.DISCRIMINATORS.PHOTON.PUMPSWAP_TRADE...), sell.data[8:24]...)
	tx.Transaction.Message.Instructions = append(tx.Transaction.Message.Instructions, map[string]interface{}{
		"programIdIndex": photonIdx, "accounts": accounts, "data": base58.Encode(data),
	})
	outer := len(tx.Transaction.Message.Instructions) - 1

	res := parseShred(t, tx, nil)
	tr := oneTypedAt(t, res, constants.DEX_PROGRAMS.PHOTON.ID, utils.FormatIdx(outer, -1)).Trade
	if tr.Type != types.TradeTypeSell || tr.InputToken.Mint != base || tr.OutputToken.Mint != quote {
		t.Errorf("Photon PumpSwap sell = %s %s -> %s, want SELL %s -> %s", tr.Type, tr.InputToken.Mint, tr.OutputToken.Mint, base, quote)
	}
}
