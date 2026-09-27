package tests

import (
	"math/big"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/meteora"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Meteora DBC shred decoding. Layouts: DBC IDL (swap: amount_in,
// minimum_amount_out; swap2: amount_0, amount_1, swap_mode with 0 exact in,
// 1 partial fill, 2 exact out; accounts 2 pool, 3 input token account,
// 4 output token account, 7 base mint, 8 quote mint, 9 payer). shred-3,
// shred-21, shred-22, hygiene-1.

const (
	// swap2 SELL (exact in) at outer 2; base mint has 6 decimals
	sigDBCSwap2Sell = "2iZbN25JjrpUDjPeYmhmAm3MWnN3XNNV8Pf9gQc742we7XZDQ9uu2ZddE2sfUBquRGH1mtgsP8FGAGGyGBbG44qT"
	// swap (v1) BUY at outer 1
	sigDBCSwapBuy = "5Y1z7B8doTaXMjqzCtRSnTPbSkpU3Ryjym857NVBBfdYLkoVP95o7V7KPdJZXNdsMXoosqghhb6aNYCmW42Ugs6t"
	// swap2 partial fill (swap_mode 1) SELL at outer 4
	sigDBCSwap2PartialFill = "55cWJsXvNHwrHrL3bzNvqb7pRCcUtq3vrm98kTXrzw64xtjEniMNfrUhrfd2WKY9A2ZifTQHhfKZSrqs1tDLmbNr"
	// initialize_virtual_pool_with_token2022 at outer 0
	sigDBCInit2022 = "3uwWqXt9wgrkfp1bwLJjMApxLW9TYxriEPtan4aWouH7ExebphFD3LthPmzXmP6eeZ5nKdtrUZodpAZiWR5P8DNV"
	// migration_damm_v2 at outer 0
	sigDBCMigrateV2 = "2P2i1ZR2JctufV5QDgfuhb42SNARTtyDoeiBV7i6hgEjuDfQkLCSyJtdpmRcCArBehMZrLkTw9viXXMvgTcwVDyD"
)

func dbcMeme(t *testing.T, res *types.ParseShredResult, ix rawIx) types.ParsedShredInstruction {
	t.Helper()
	ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.METEORA_DBC.ID, utils.FormatIdx(ix.outer, ix.inner))
	if ins.MemeEvent == nil {
		t.Fatalf("%s: no meme event", ins.Action)
	}
	return ins
}

// TestShredDBCSwapDirection: every DBC swap was reported as BUY (the decoder
// compared "SELL" with "sell") with swapped mints and 9 hardcoded decimals.
// shred-3, shred-22.
func TestShredDBCSwapDirection(t *testing.T) {
	for _, c := range []struct {
		sig      string
		disc     []byte
		wantType types.TradeType
		// wantInKind: amount_0 is the exact input, except in partial-fill
		// mode where the program may take less (shred-4 of the final review)
		wantInKind types.ShredAmountKind
	}{
		{sigDBCSwap2Sell, constants.DISCRIMINATORS.METEORA_DBC.SWAP_V2, types.TradeTypeSell, types.ShredAmountExact},
		{sigDBCSwapBuy, constants.DISCRIMINATORS.METEORA_DBC.SWAP, types.TradeTypeBuy, types.ShredAmountExact},
		{sigDBCSwap2PartialFill, constants.DISCRIMINATORS.METEORA_DBC.SWAP_V2, types.TradeTypeSell, types.ShredAmountMax},
	} {
		tx := loadFixture(t, c.sig)
		ix := findIx(t, tx, constants.DEX_PROGRAMS.METEORA_DBC.ID, c.disc)
		base, quote, payer := ix.accounts[7], ix.accounts[8], ix.accounts[9]
		amountIn, minOut := le64At(ix.data, 8), le64At(ix.data, 16)
		inMint, outMint := quote, base
		if c.wantType == types.TradeTypeSell {
			inMint, outMint = base, quote
		}
		// The direction is the input token account's mint
		if got := tokenBalanceMint(tx, ix.accounts[3]); got != inMint {
			t.Fatalf("fixture: input account mint %s, want %s", got, inMint)
		}

		ins := dbcMeme(t, parseShred(t, tx, nil), ix)
		m := ins.MemeEvent
		if m.Type != c.wantType || m.InputToken.Mint != inMint || m.OutputToken.Mint != outMint || m.User != payer {
			t.Errorf("%s: %s %s -> %s by %s, want %s %s -> %s by %s", c.sig[:8], m.Type, m.InputToken.Mint, m.OutputToken.Mint, m.User, c.wantType, inMint, outMint, payer)
		}
		if m.InputToken.AmountRaw != u64str(amountIn) || m.OutputToken.AmountRaw != u64str(minOut) || ins.InputAmountKind != c.wantInKind || ins.OutputAmountKind != types.ShredAmountMin {
			t.Errorf("%s: amounts %s/%s kinds %s/%s", c.sig[:8], m.InputToken.AmountRaw, m.OutputToken.AmountRaw, ins.InputAmountKind, ins.OutputAmountKind)
		}
		if m.InputToken.Decimals != mintDecimals(tx, inMint) || m.OutputToken.Decimals != mintDecimals(tx, outMint) {
			t.Errorf("%s: decimals %d/%d, want %d/%d", c.sig[:8], m.InputToken.Decimals, m.OutputToken.Decimals, mintDecimals(tx, inMint), mintDecimals(tx, outMint))
		}
		// Exact in: the user's input account lost amount_in
		if got := ownerTokenDelta(tx, payer, inMint); inMint != solMint && got.Cmp(new(big.Int).Neg(new(big.Int).SetUint64(amountIn))) != 0 {
			t.Errorf("%s: input delta %s, want -%d", c.sig[:8], got, amountIn)
		}
	}
}

// TestShredDBCPreExecutionDirection: without meta the direction comes from
// the payer's associated token account. In 2iZbN25J the payer's base ATA
// needs a bump below 255, which the old address derivation (bump 255 only)
// never found, so the swap was not recognized as a SELL. hygiene-1.
func TestShredDBCPreExecutionDirection(t *testing.T) {
	tx := loadFixture(t, sigDBCSwap2Sell)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.METEORA_DBC.ID, constants.DISCRIMINATORS.METEORA_DBC.SWAP_V2)
	base, payer := ix.accounts[7], ix.accounts[9]
	tokenProgram := ix.accounts[10]
	if old := bump255Address(t, payer, tokenProgram, base); old == ix.accounts[3] {
		t.Fatalf("fixture: the input ATA uses bump 255")
	}

	ins := dbcMeme(t, parseShred(t, preExec(t, tx), nil), ix)
	m := ins.MemeEvent
	if m.Type != types.TradeTypeSell || m.InputToken.Mint != base || m.OutputToken.Mint != ix.accounts[8] {
		t.Errorf("pre-exec %s %s -> %s, want SELL %s -> %s", m.Type, m.InputToken.Mint, m.OutputToken.Mint, base, ix.accounts[8])
	}
}

// TestShredDBCUnknownDirection: when neither the token account mints nor
// the payer's ATAs decide the direction, the swap is reported as SWAP with
// unknown mints instead of BUY. The test swaps the real swap's token
// accounts for unrelated accounts of the same transaction. shred-3.
func TestShredDBCUnknownDirection(t *testing.T) {
	tx := preExec(t, loadFixture(t, sigDBCSwap2Sell))
	ix := findIx(t, tx, constants.DEX_PROGRAMS.METEORA_DBC.ID, constants.DISCRIMINATORS.METEORA_DBC.SWAP_V2)
	accs := ix.m["accounts"].([]interface{})
	accs[3], accs[4] = accs[1], accs[0] // config and pool authority: no token accounts of the payer

	ins := dbcMeme(t, parseShred(t, tx, nil), ix)
	m := ins.MemeEvent
	if m.Type != types.TradeTypeSwap || m.InputToken.Mint != "" || m.OutputToken.Mint != "" {
		t.Errorf("undetermined direction reported as %s %q -> %q, want SWAP with unknown mints", m.Type, m.InputToken.Mint, m.OutputToken.Mint)
	}
}

// TestShredDBCSwap2ExactOut: swap2 ignored swap_mode; in exact-out mode
// amount_0 is the output and amount_1 the maximum input. No exact-out swap2
// was found in the fixtures or sampled blocks (modes 0 and 1 only), so the
// test sets swap_mode 2 on the real swap2 of 2iZbN25J. shred-21.
func TestShredDBCSwap2ExactOut(t *testing.T) {
	tx := loadFixture(t, sigDBCSwap2Sell)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.METEORA_DBC.ID, constants.DISCRIMINATORS.METEORA_DBC.SWAP_V2)
	if ix.data[24] != 0 {
		t.Fatalf("fixture swap_mode %d", ix.data[24])
	}
	data := append([]byte{}, ix.data...)
	data[24] = 2
	setIxData(ix, data)
	amountOut, maxIn := le64At(data, 8), le64At(data, 16)

	res := parseShred(t, tx, nil)
	ins := dbcMeme(t, res, ix)
	m := ins.MemeEvent
	if m.OutputToken.AmountRaw != u64str(amountOut) || m.InputToken.AmountRaw != u64str(maxIn) || ins.InputAmountKind != types.ShredAmountMax || ins.OutputAmountKind != types.ShredAmountExact {
		t.Errorf("exact out: in %s out %s kinds %s/%s, want in %d (max) out %d (exact)", m.InputToken.AmountRaw, m.OutputToken.AmountRaw, ins.InputAmountKind, ins.OutputAmountKind, maxIn, amountOut)
	}
	raw := res.Instructions[constants.DEX_PROGRAMS.METEORA_DBC.Name][0].(*meteora.DBCShredInstruction).Data.(*meteora.DBCSwapData)
	if raw.SwapMode != 2 || !raw.ExactOut {
		t.Errorf("raw swap data %+v", raw)
	}
}

// TestShredDBCCreateAndMigrate checks the pool creation and migration events
// against the IDL account layouts.
func TestShredDBCCreateAndMigrate(t *testing.T) {
	tx := loadFixture(t, sigDBCInit2022)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.METEORA_DBC.ID, constants.DISCRIMINATORS.METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022)
	nameLen := int(le32At(ix.data, 8))
	m := dbcMeme(t, parseShred(t, tx, nil), ix).MemeEvent
	if m.Type != types.TradeTypeCreate || m.Name != string(ix.data[12:12+nameLen]) || m.User != ix.accounts[2] || m.BaseMint != ix.accounts[3] || m.QuoteMint != ix.accounts[4] || m.Pool != ix.accounts[5] {
		t.Errorf("create = %+v", m)
	}

	tx = loadFixture(t, sigDBCMigrateV2)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.METEORA_DBC.ID, constants.DISCRIMINATORS.METEORA_DBC.METEORA_DBC_MIGRATE_DAMM_V2)
	m = dbcMeme(t, parseShred(t, tx, nil), ix).MemeEvent
	if m.Type != types.TradeTypeMigrate || m.BaseMint != ix.accounts[13] || m.QuoteMint != ix.accounts[14] || m.Pool != ix.accounts[4] || m.BondingCurve != ix.accounts[0] {
		t.Errorf("migrate = %+v", m)
	}
}
