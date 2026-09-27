package tests

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Regression tests for relay tips (streamer item 13): the tip registry,
// ParseResult.Tip, and tips never booked as trade fees.

// systemTipTransfers sums, from the raw instructions (outer and inner), the
// lamports of System transfers to TIP_ACCOUNTS entries.
func systemTipTransfers(t *testing.T, tx *adapter.SolanaTransaction) *big.Int {
	keys := rawAccountKeys(tx)
	total := new(big.Int)
	check := func(ix interface{}) {
		m, ok := ix.(map[string]interface{})
		if !ok || keys[jsonInt(m["programIdIndex"])] != constants.SYSTEM_PROGRAM_ID {
			return
		}
		data, _ := base58.Decode(m["data"].(string))
		accs, _ := m["accounts"].([]interface{})
		if len(data) < 12 || binary.LittleEndian.Uint32(data) != 2 || len(accs) < 2 {
			return
		}
		if constants.IsTipAccount(keys[jsonInt(accs[1])]) {
			total.Add(total, new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[4:12])))
		}
	}
	for _, ix := range tx.Transaction.Message.Instructions {
		check(ix)
	}
	for _, set := range tx.Meta.InnerInstructions {
		for _, ix := range set.Instructions {
			check(ix)
		}
	}
	return total
}

// TestCore2TipTotal: ParseResult.Tip is the SOL paid to relay tip accounts,
// checked on every fixture against the raw System transfers. streamer item 13.
func TestCore2TipTotal(t *testing.T) {
	p := dexparser.NewDexParser()
	withTip := 0
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		if tx.Meta == nil || len(tx.Transaction.Message.Instructions) == 0 {
			continue
		}
		want := systemTipTransfers(t, tx)
		r := p.ParseAll(tx, nil)
		if tx.Meta.Err != nil {
			want.SetInt64(0) // reverted
		}
		switch {
		case want.Sign() == 0 && r.Tip != nil:
			t.Errorf("%s: Tip %s, want none", sig[:8], r.Tip.Amount)
		case want.Sign() > 0 && (r.Tip == nil || r.Tip.Amount != want.String() || r.Tip.Decimals != 9):
			t.Errorf("%s: Tip %v, want %s lamports", sig[:8], r.Tip, want)
		}
		if want.Sign() > 0 {
			withTip++
		}
	}
	if withTip < 20 {
		t.Errorf("only %d fixtures pay a tip", withTip)
	}
	// v8s37: a Pump.fun buy with a 0.0248 SOL Jito tip
	if r := p.ParseAll(loadFixture(t, "v8s37Srj6QPMtRC1HfJcrSenCHvYebHiGkHVuFFiQ6UviqHnoVx4U77M3TZhQQXewXadHYh5t35LkesJi3ztPZZ"), nil); r.Tip == nil || r.Tip.Amount != "24800000" {
		t.Errorf("v8s37: Tip %v, want 24800000", r.Tip)
	}
}

// TestCore2TipIsNotTradeFee: a tip transfer grouped with a swap's transfers
// was booked as the trade's fee (the Jito tip accounts are in FEE_ACCOUNTS).
// D8, parity-24, streamer item 13.
//
// Synthetic: no real transaction with a tip CPI'd inside a swap group was
// found (295 fixtures, 1280 cached bot transactions). Built from the real
// Orca swap 5AzH3HAp by copying its outer Jito tip transfer into the Orca
// instruction's inner instructions.
func TestCore2TipIsNotTradeFee(t *testing.T) {
	const sig = "5AzH3HApZUEnGECG5Xk26jgUpbiRAzRsAqTRHiTj7Jf6bX3jfSXZCj5zqREvgnYwzgD2mUXw9g6FrN2hVtJZF5JN"
	orig := loadFixture(t, sig)
	keys := rawAccountKeys(orig)
	var tip map[string]interface{}
	orcaOuter := -1
	for i, ix := range orig.Transaction.Message.Instructions {
		m := ix.(map[string]interface{})
		switch keys[jsonInt(m["programIdIndex"])] {
		case constants.SYSTEM_PROGRAM_ID:
			if accs := m["accounts"].([]interface{}); len(accs) == 2 && constants.GetTipProvider(keys[jsonInt(accs[1])]) == "Jito" {
				tip = m
			}
		case constants.DEX_PROGRAMS.ORCA.ID:
			orcaOuter = i
		}
	}
	if tip == nil || orcaOuter < 0 {
		t.Fatal("fixture: no Jito tip or no outer Orca instruction")
	}
	tx := cloneTx(t, orig)
	for i := range tx.Meta.InnerInstructions {
		if tx.Meta.InnerInstructions[i].Index == orcaOuter {
			inner := map[string]interface{}{"programIdIndex": tip["programIdIndex"], "accounts": tip["accounts"], "data": tip["data"], "stackHeight": 2}
			tx.Meta.InnerInstructions[i].Instructions = append(tx.Meta.InnerInstructions[i].Instructions, inner)
		}
	}

	p := dexparser.NewDexParser()
	want := p.ParseAll(orig, tradesConfig())
	got := p.ParseAll(tx, tradesConfig())
	if len(want.Trades) != 1 || len(got.Trades) != 1 {
		t.Fatalf("trades: %d and %d, want 1", len(want.Trades), len(got.Trades))
	}
	w, g := want.Trades[0], got.Trades[0]
	if g.Fee != nil || len(g.Fees) != 0 {
		t.Errorf("tip booked as trade fee: %+v %+v", g.Fee, g.Fees)
	}
	if g.InputToken.Mint != w.InputToken.Mint || g.InputToken.AmountRaw != w.InputToken.AmountRaw ||
		g.OutputToken.Mint != w.OutputToken.Mint || g.OutputToken.AmountRaw != w.OutputToken.AmountRaw {
		t.Errorf("trade changed by the tip: %s %s -> %s %s, want %s %s -> %s %s",
			g.InputToken.Mint, g.InputToken.AmountRaw, g.OutputToken.Mint, g.OutputToken.AmountRaw,
			w.InputToken.Mint, w.InputToken.AmountRaw, w.OutputToken.Mint, w.OutputToken.AmountRaw)
	}
	for _, transfers := range utils.NewTransactionUtils(adapter.NewTransactionAdapter(tx, nil)).GetTransferActions(nil) {
		for _, tr := range transfers {
			if constants.IsTipAccount(tr.Info.Destination) && tr.IsFee {
				t.Errorf("tip transfer at %s flagged IsFee", tr.Idx)
			}
		}
	}
}

// TestCore2TipAccounts: the tip registry. Jito's 8 accounts are owned by the
// Jito tip-payment program (verified on-chain); they stay in FEE_ACCOUNTS (D12)
// but are not trade fee accounts. No tip account is a bot fee account or a
// DEX program.
func TestCore2TipAccounts(t *testing.T) {
	if n := len(constants.TIP_ACCOUNTS["Jito"]); n != 8 {
		t.Errorf("Jito tip accounts: %d, want 8", n)
	}
	seen := map[string]bool{}
	for provider, accounts := range constants.TIP_ACCOUNTS {
		for _, a := range accounts {
			if !constDecodes32(a) || seen[a] {
				t.Errorf("%s tip account %q invalid or duplicated", provider, a)
			}
			seen[a] = true
			if constants.GetTipProvider(a) != provider || constants.IsBotFeeAccount(a) || constants.IsDexProgram(a) || constants.IsTradeFeeAccount(a) {
				t.Errorf("%s tip account %s: provider %q, bot %v, dex %v", provider, a, constants.GetTipProvider(a), constants.IsBotFeeAccount(a), constants.IsDexProgram(a))
			}
		}
	}
	for _, a := range constants.TIP_ACCOUNTS["Jito"] {
		if !constants.IsFeeAccount(a) {
			t.Errorf("Jito tip account %s removed from FEE_ACCOUNTS", a)
		}
	}
	if !constants.IsTradeFeeAccount("AVUCZyuT35YSuj4RH7fwiyPu82Djn2Hfg7y2ND2XcnZH") {
		t.Error("Photon fee vault is not a trade fee account")
	}
}
