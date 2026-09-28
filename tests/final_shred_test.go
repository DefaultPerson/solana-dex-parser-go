package tests

import (
	"strconv"
	"strings"
	"testing"

	"github.com/goccy/go-json"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Regression tests for the shred findings of the final review (shred-1 to
// shred-4). Expected values come from the raw fixtures: instruction bytes
// decoded with the IDL layouts and the signer's balance changes.

// TestFinalShredPumpswapWsolBaseType: PumpSwap shred trades took their type
// from the instruction name (buy = BUY), which is inverted in pools whose
// base mint is WSOL and whose quote is a token. Truth: the signer's native
// SOL change (SOL spent is a BUY, SOL received a SELL), before and after
// execution. shred-1.
func TestFinalShredPumpswapWsolBaseType(t *testing.T) {
	for _, c := range []struct {
		sig    string
		outer  int
		action string
		want   types.TradeType
	}{
		// sell of the WSOL base: the signer pays about 2.9 SOL
		{"9gCqA5ozzfS9jvZmzGi1Hzfb5uMPUTXz1VLSjHkGGeayeT2G6po1KKYZhAAghUrxFxg8wfeDzgo4qEWUVQFa7QP", 4, "sell", types.TradeTypeBuy},
		// buy_exact_quote_in of the WSOL base: the signer gets about 3.3 SOL
		{"4TLw5ASBmhsc4zFjsqk7s9NARouq7BtxUgPDEnubeikrKUCRGUZgE2fpRp6bjdwPFbcbA39MD2VNakf8ijnPXzpo", 2, "buy_exact_quote_in", types.TradeTypeSell},
	} {
		tx := loadFixture(t, c.sig)
		var ix rawIx
		for _, x := range fixtureIxs(t, tx) {
			if x.programId == constants.DEX_PROGRAMS.PUMP_SWAP.ID && x.outer == c.outer && x.inner < 0 {
				ix = x
			}
		}
		if len(ix.accounts) < 9 || ix.accounts[3] != solMint || constants.IsQuoteToken(ix.accounts[4]) {
			t.Fatalf("%s: outer %d is not a PumpSwap swap in a WSOL-base, token-quoted pool", c.sig[:8], c.outer)
		}
		signer := rawAccountKeys(tx)[0]
		solDelta := lamportDelta(tx, signer)
		wantFromBalance := types.TradeTypeSell
		if solDelta.Sign() < 0 {
			wantFromBalance = types.TradeTypeBuy
		}
		if wantFromBalance != c.want {
			t.Fatalf("%s: signer SOL delta %s, want a %s", c.sig[:8], solDelta, c.want)
		}

		idx := utils.FormatIdx(c.outer, -1)
		for name, res := range map[string]*types.ParseShredResult{
			"executed":      parseShred(t, tx, nil),
			"pre-execution": parseShred(t, preExec(t, tx), nil),
		} {
			ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.PUMP_SWAP.ID, idx)
			if ins.Action != c.action || ins.Trade == nil {
				t.Fatalf("%s %s: %s at %s, want a %s trade", c.sig[:8], name, ins.Action, idx, c.action)
			}
			if ins.Trade.Type != c.want {
				t.Errorf("%s %s: %s %s -> %s typed %s, want %s", c.sig[:8], name, c.action,
					ins.Trade.InputToken.Mint, ins.Trade.OutputToken.Mint, ins.Trade.Type, c.want)
			}
		}
	}
}

// TestFinalShredPoolSideTradeType: the pool-side rule with unknown mints
// (pre-execution lookup table accounts). shred-1.
func TestFinalShredPoolSideTradeType(t *testing.T) {
	const token = "J9FP7JTbzmtBGP8GZzGRpBJuN53sQWe6kc4ASYWP6Ssk"
	for _, c := range []struct {
		buy         bool
		base, quote string
		want        types.TradeType
	}{
		{true, token, solMint, types.TradeTypeBuy},
		{false, token, solMint, types.TradeTypeSell},
		{true, solMint, token, types.TradeTypeSell},
		{false, solMint, token, types.TradeTypeBuy},
		{true, token, "", types.TradeTypeBuy},
		{true, solMint, "", types.TradeTypeSell},
		{false, "", solMint, types.TradeTypeSell},
		{true, "", token, types.TradeTypeSwap},
		{true, usdcMint, "", types.TradeTypeSwap},
		{true, "", "", types.TradeTypeSwap},
	} {
		if got := utils.GetPoolSideTradeType(c.buy, c.base, c.quote); got != c.want {
			t.Errorf("buy=%v base=%.6s quote=%.6s: %s, want %s", c.buy, c.base, c.quote, got, c.want)
		}
	}
}

// TestFinalShredGuessedTransferMint: a plain SPL Token transfer names no
// mint. Without meta the adapter defaults the source account to WSOL, and the
// shred output reported that guess (9 decimals, a UI amount 1000x too small
// for USDC). Truth: the source account's token balance (USDC, 6 decimals).
// shred-2.
func TestFinalShredGuessedTransferMint(t *testing.T) {
	const sig = "Z4CChBawHPHwuUd9JmpvtyaddghjoeyxioFXB2HK4NSioKV5tipgajmznRbzx9LhiEdvYL5hedXWnEqEzmYVsGA"
	tx := loadFixture(t, sig)
	var ix rawIx
	for _, x := range fixtureIxs(t, tx) {
		if x.programId == constants.TOKEN_PROGRAM_ID && x.outer == 1 && x.inner < 0 {
			ix = x
		}
	}
	if len(ix.data) != 9 || ix.data[0] != 3 {
		t.Fatalf("outer 1 is not an SPL Token transfer (tag 3)")
	}
	source := ix.accounts[0]
	wantMint, wantDecimals := tokenBalanceMint(tx, source), tokenBalanceDecimals(tx, source)
	if wantMint != usdcMint || wantDecimals != 6 {
		t.Fatalf("fixture: source mint %s/%d", wantMint, wantDecimals)
	}
	amount := u64str(le64At(ix.data, 1))

	transferAt := func(res *types.ParseShredResult) *types.TransferData {
		ins := oneTypedAt(t, res, constants.TOKEN_PROGRAM_ID, "1")
		if ins.Transfer == nil {
			t.Fatalf("no transfer at 1")
		}
		return ins.Transfer
	}
	executed := transferAt(parseShred(t, tx, nil))
	if executed.Info.Mint != wantMint || executed.Info.TokenAmount.Decimals != wantDecimals || executed.Info.TokenAmount.Amount != amount {
		t.Errorf("executed: %s %s/%d, want %s %s/%d", executed.Info.TokenAmount.Amount, executed.Info.Mint, executed.Info.TokenAmount.Decimals, amount, wantMint, wantDecimals)
	}
	pre := transferAt(parseShred(t, preExec(t, tx), nil))
	if pre.Info.Mint != "" || pre.Info.TokenAmount.Decimals != 0 || pre.Info.TokenAmount.Amount != amount {
		t.Errorf("pre-execution: %s %s/%d, want %s with an unknown mint (\"\"/0)", pre.Info.TokenAmount.Amount, pre.Info.Mint, pre.Info.TokenAmount.Decimals, amount)
	}
}

// finalIdxLess compares "outer" / "outer-inner" idx strings numerically,
// with an outer instruction before its inner instructions
func finalIdxLess(a, b string) bool {
	split := func(s string) (int, int) {
		o, i, found := strings.Cut(s, "-")
		outer, _ := strconv.Atoi(o)
		inner := -1
		if found {
			inner, _ = strconv.Atoi(i)
		}
		return outer, inner
	}
	ao, ai := split(a)
	bo, bi := split(b)
	if ao != bo {
		return ao < bo
	}
	return ai < bi
}

// TestFinalShredLegacyEventOrder: the legacy per-program event lists of
// ParseShredResult.Instructions came in classifier order (the program's outer
// instructions first, then its inner ones) for every program but Pump.fun
// and PumpSwap. They are in execution order now, on every fixture. shred-3.
func TestFinalShredLegacyEventOrder(t *testing.T) {
	checked := 0
	for _, sig := range fixtureSignatures(t, "json") {
		res := parseShred(t, loadFixture(t, sig), &types.ParseConfig{IncludeFailedTxs: true})
		for program, events := range res.Instructions {
			var prev string
			for i, e := range events {
				idx := finalLegacyIdx(e)
				if idx == "" {
					t.Fatalf("%s %s: event %d has no idx (%T)", sig[:8], program, i, e)
				}
				if i > 0 && finalIdxLess(idx, prev) {
					t.Errorf("%s %s: event %s after %s", sig[:8], program, idx, prev)
					break
				}
				prev = idx
			}
			checked++
		}
	}
	if checked < 100 {
		t.Fatalf("only %d program lists checked", checked)
	}
}

// finalLegacyIdx reads the idx of a legacy shred event through its JSON form
func finalLegacyIdx(e interface{}) string {
	var m struct {
		Idx string `json:"idx"`
	}
	raw, err := json.Marshal(e)
	if err != nil || json.Unmarshal(raw, &m) != nil {
		return ""
	}
	return m.Idx
}

// TestFinalShredDBCPartialFillKinds: in swap2 partial-fill mode (swap_mode 1)
// the DBC program may take less than amount_0 (EvtSwap2 amount_left), so
// amount_0 is a maximum, not the exact input. Truth: swap_mode decoded from
// every DBC swap2 in the fixtures (IDL: amount_0 u64, amount_1 u64,
// swap_mode u8). shred-4.
func TestFinalShredDBCPartialFillKinds(t *testing.T) {
	seen := map[uint8]int{}
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		var swaps []rawIx
		for _, ix := range fixtureIxs(t, tx) {
			d := constants.DISCRIMINATORS.METEORA_DBC.SWAP_V2
			if ix.programId == constants.DEX_PROGRAMS.METEORA_DBC.ID && len(ix.data) >= 25 && string(ix.data[:8]) == string(d) {
				swaps = append(swaps, ix)
			}
		}
		if len(swaps) == 0 {
			continue
		}
		res := parseShred(t, tx, &types.ParseConfig{IncludeFailedTxs: true})
		for _, ix := range swaps {
			mode := ix.data[24]
			wantIn, wantOut := types.ShredAmountExact, types.ShredAmountMin
			switch mode {
			case 1:
				wantIn = types.ShredAmountMax
			case 2:
				wantIn, wantOut = types.ShredAmountMax, types.ShredAmountExact
			}
			ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.METEORA_DBC.ID, utils.FormatIdx(ix.outer, ix.inner))
			if ins.InputAmountKind != wantIn || ins.OutputAmountKind != wantOut {
				t.Errorf("%s %s swap_mode %d: kinds %s/%s, want %s/%s", sig[:8], utils.FormatIdx(ix.outer, ix.inner), mode, ins.InputAmountKind, ins.OutputAmountKind, wantIn, wantOut)
			}
			seen[mode]++
		}
	}
	if seen[0] == 0 || seen[1] == 0 {
		t.Fatalf("swap_mode coverage %v: want exact-in and partial-fill swaps", seen)
	}
}
