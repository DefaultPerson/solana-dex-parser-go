package tests

import (
	"math/big"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Pump.fun and PumpSwap shred decoding against real transactions. Expected
// values are read from the instruction bytes and accounts with the on-chain
// IDL layouts (pump-public-docs idl/pump.json, pump_amm.json) and checked
// against the balance changes. shred-15, shred-17, shred-20, shred-30,
// constants-4.

const (
	// create_v2 (19 accounts, WSOL quote) + buy_exact_sol_in (18 accounts)
	sigPumpCreateV2BuyExactSolIn = "3jWGFYXT5V33Qc2roEBFDRAWHeybDowr53dSdnYSRkrPdYybU7oyEH9BfgSRxkgFHVKmUjv4e5T33AEnhJvBCuP2"
	// create_v2 with a USDC quote (remaining accounts 16-18) + buy_v2
	sigPumpCreateV2USDC = "3MVawF6EPtG7rEPXdsyQfQUBLv3epRVNpNS4tRE4uwTPMqLNPqhuABwxU3QZH4uD6CuVupcpGchpNRK5HTbHRLNK"
	// create_v2 + buy_exact_quote_in_v2
	sigPumpCreateV2ExactQuote = "5HwZKTwcGFjSBPugSX5hE9JSq5wKmUooK3tLXuEoyDDzrTvHu7op3XDbhBXuteiC5EePNPh8TC1j6Fns47YvnyeG"
	// create_v2 with 16 accounts (SOL quote) + legacy buy (18 accounts)
	sigPumpCreateV2SOL = "H6azwLqtRtrnVNC5iwcjYM9idU3e9SRyLZXTwjfJGJxA4X7dZL7vyhFAJNvQy7bb6bmQNmFHUt1KkkPPmhdge3G"
	// migrate_v2 (+ PumpSwap create_pool 2-22)
	sigPumpMigrateV2 = "v5rg9RMc6D4pMsAqD8TrmXGFwHQBePFDWXBbtsQmP5gttLBKvExSEiPcGMipaDWP61VdWaxEyJCr7oXPxFH4DQf"
	// legacy migrate (+ PumpSwap create_pool 2-22 and PumpSwap buy at 7)
	sigPumpMigrate = "5fiQbExgdp1FAjDnrv9aEpXajCMUtm1c3E7NnsDdu4CtKr5xBpALdK7ENzx5LN1SzZKJk7cxbWWc84T7yHwb8p2x"
	// sell_v2 at 3-0
	sigPumpSellV2 = "2DdZiRsXJWVkD1nD31bcBQpZPKmmmd1gG1T4GAFcLDw6f7Zpxao876E6pPcreUqMmKLWjmxo5FRvq6WynU2pPXaW"
	// legacy sell at 3
	sigPumpSell = "46sErg2LPybXGCbwsY6bESNbLT3ysp1U8DE1Hw46soEF4g4bmGUhhhbd42dvuVjGi6LRLekhmyW3ArZxU6dpyoGt"
	// PumpSwap buy_exact_quote_in at 5
	sigPswapBuyExactQuoteIn = "Q2riWA8oA5RbLW531yo32SYU1rRHQ2PdzfxMRXSJ24qfxWUqTV48NeswMuAiLSG6uDhwLpafJHyKbctU8vyaadn"
	// PumpSwap sell at 5
	sigPswapSell = "36Q2tYo1CPa42GF51bzA493nYQCG8fPbpQJEzRhZQURYuBcRKpj97HWBCLCzDwgQJ8tnVrW9fDZKWaPBdADEsxTE"
	// PumpSwap deposit at 6
	sigPswapDeposit = "63PQNtzDQdBDnf7FMC4jafDPhDVZHHhZwhJAbCT25efDgXt4H3fxPSQCAacV5Psaz5aKrRk35ubc97oQuozg9rwV"
)

// pumpTrade checks the typed trade (and meme event) of a Pump.fun instruction
func pumpTrade(t *testing.T, res *types.ParseShredResult, ix rawIx, wantType types.TradeType, in, out types.TokenInfo, inKind, outKind types.ShredAmountKind) {
	t.Helper()
	ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.PUMP_FUN.ID, utils.FormatIdx(ix.outer, ix.inner))
	tr := ins.Trade
	if tr == nil || ins.MemeEvent == nil {
		t.Fatalf("%s: trade %v meme %v", ins.Action, tr, ins.MemeEvent)
	}
	if tr.Type != wantType || ins.MemeEvent.Type != wantType {
		t.Errorf("%s: type %s/%s, want %s", ins.Action, tr.Type, ins.MemeEvent.Type, wantType)
	}
	check := func(side string, got types.TokenInfo, want types.TokenInfo) {
		if got.Mint != want.Mint || got.AmountRaw != want.AmountRaw || got.Decimals != want.Decimals {
			t.Errorf("%s %s = %s %s (%d dec), want %s %s (%d dec)", ins.Action, side, got.Mint, got.AmountRaw, got.Decimals, want.Mint, want.AmountRaw, want.Decimals)
		}
	}
	check("input", tr.InputToken, in)
	check("output", tr.OutputToken, out)
	check("meme input", *ins.MemeEvent.InputToken, in)
	check("meme output", *ins.MemeEvent.OutputToken, out)
	if ins.InputAmountKind != inKind || ins.OutputAmountKind != outKind {
		t.Errorf("%s: amount kinds %s/%s, want %s/%s", ins.Action, ins.InputAmountKind, ins.OutputAmountKind, inKind, outKind)
	}
}

func tok(mint string, amount uint64, decimals uint8) types.TokenInfo {
	return types.TokenInfo{Mint: mint, AmountRaw: u64str(amount), Decimals: decimals}
}

// TestShredPumpfunV2Instructions: buy_exact_sol_in, create_v2, buy_v2,
// sell_v2 and buy_exact_quote_in_v2 were not decoded, and Pump.fun had no
// typed output. shred-17, shred-20, constants-4.
func TestShredPumpfunV2Instructions(t *testing.T) {
	d := constants.DISCRIMINATORS.PUMPFUN

	// create_v2 (WSOL quote at remaining account 16) + buy_exact_sol_in
	tx := loadFixture(t, sigPumpCreateV2BuyExactSolIn)
	create := findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, d.CREATE_V2)
	buy := findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, d.BUY_EXACT_SOL_IN)
	res := parseShred(t, tx, nil)
	mint, user := buy.accounts[2], buy.accounts[6]
	spendable, minTokens := le64At(buy.data, 8), le64At(buy.data, 16)
	pumpTrade(t, res, buy, types.TradeTypeBuy, tok(solMint, spendable, 9), tok(mint, minTokens, 6), types.ShredAmountExact, types.ShredAmountMin)
	if got := ownerTokenDelta(tx, user, mint); got.Cmp(new(big.Int).SetUint64(minTokens)) < 0 {
		t.Errorf("tokens received %s < min_tokens_out %d", got, minTokens)
	}
	cm := oneTypedAt(t, res, constants.DEX_PROGRAMS.PUMP_FUN.ID, utils.FormatIdx(create.outer, -1)).MemeEvent
	if cm == nil || cm.Type != types.TradeTypeCreate || cm.BaseMint != create.accounts[0] || cm.User != create.accounts[5] ||
		cm.BondingCurve != create.accounts[2] || cm.QuoteMint != create.accounts[16] || cm.QuoteMint != solMint || cm.Name == "" || cm.Symbol == "" {
		t.Errorf("create_v2 meme event = %+v", cm)
	}

	// create_v2 with a USDC quote + buy_v2 (amount, max_sol_cost)
	tx = loadFixture(t, sigPumpCreateV2USDC)
	create = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, d.CREATE_V2)
	buy = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, d.BUY_V2)
	res = parseShred(t, tx, nil)
	if create.accounts[16] != usdcMint || buy.accounts[2] != usdcMint {
		t.Fatalf("fixture: quote accounts %s %s", create.accounts[16], buy.accounts[2])
	}
	amount, maxCost := le64At(buy.data, 8), le64At(buy.data, 16)
	pumpTrade(t, res, buy, types.TradeTypeBuy, tok(usdcMint, maxCost, 6), tok(buy.accounts[1], amount, 6), types.ShredAmountMax, types.ShredAmountExact)
	user = buy.accounts[13]
	if got := ownerTokenDelta(tx, user, buy.accounts[1]); got.Cmp(new(big.Int).SetUint64(amount)) != 0 {
		t.Errorf("tokens received %s, want amount %d", got, amount)
	}
	if spent := new(big.Int).Neg(ownerTokenDelta(tx, user, usdcMint)); spent.Sign() <= 0 || spent.Cmp(new(big.Int).SetUint64(maxCost)) > 0 {
		t.Errorf("USDC spent %s, want within (0, max_sol_cost %d]", spent, maxCost)
	}
	if cm := oneTypedAt(t, res, constants.DEX_PROGRAMS.PUMP_FUN.ID, utils.FormatIdx(create.outer, -1)).MemeEvent; cm.QuoteMint != usdcMint {
		t.Errorf("create_v2 quote = %s, want USDC", cm.QuoteMint)
	}
	// create_v2 flags: the args end with is_mayhem_mode 0, is_cashback_enabled 1
	if tail := create.data[len(create.data)-2:]; tail[0] != 0 || tail[1] != 1 {
		t.Fatalf("fixture: create_v2 flags %x", tail)
	}
	for _, e := range res.Instructions[constants.DEX_PROGRAMS.PUMP_FUN.Name] {
		if c, ok := e.(*dexparser.PumpfunInstruction).Data.(*dexparser.PumpfunCreateData); ok && (c.IsMayhemMode || !c.IsCashbackEnabled || c.Creator != buy.accounts[13]) {
			t.Errorf("create_v2 flags/creator = %+v", c)
		}
	}

	// create_v2 + buy_exact_quote_in_v2 (spendable_quote_in, min_tokens_out)
	tx = loadFixture(t, sigPumpCreateV2ExactQuote)
	buy = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, d.BUY_EXACT_QUOTE_IN_V2)
	res = parseShred(t, tx, nil)
	pumpTrade(t, res, buy, types.TradeTypeBuy, tok(buy.accounts[2], le64At(buy.data, 8), 9), tok(buy.accounts[1], le64At(buy.data, 16), 6), types.ShredAmountExact, types.ShredAmountMin)
	if got := ownerTokenDelta(tx, buy.accounts[13], buy.accounts[1]); got.Cmp(new(big.Int).SetUint64(le64At(buy.data, 16))) < 0 {
		t.Errorf("tokens received %s < min_tokens_out", got)
	}

	// create_v2 with 16 accounts: SOL quote; legacy buy with 18 accounts
	tx = loadFixture(t, sigPumpCreateV2SOL)
	create = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, d.CREATE_V2)
	buy = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, d.BUY)
	if len(create.accounts) != 16 {
		t.Fatalf("fixture: create_v2 has %d accounts", len(create.accounts))
	}
	res = parseShred(t, tx, nil)
	if cm := oneTypedAt(t, res, constants.DEX_PROGRAMS.PUMP_FUN.ID, utils.FormatIdx(create.outer, -1)).MemeEvent; cm.QuoteMint != solMint {
		t.Errorf("16-account create_v2 quote = %s, want SOL", cm.QuoteMint)
	}
	pumpTrade(t, res, buy, types.TradeTypeBuy, tok(solMint, le64At(buy.data, 16), 9), tok(buy.accounts[2], le64At(buy.data, 8), 6), types.ShredAmountMax, types.ShredAmountExact)

	// sell_v2 (amount, min_sol_output), inner
	tx = loadFixture(t, sigPumpSellV2)
	sell := findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, d.SELL_V2)
	res = parseShred(t, tx, nil)
	pumpTrade(t, res, sell, types.TradeTypeSell, tok(sell.accounts[1], le64At(sell.data, 8), 6), tok(sell.accounts[2], le64At(sell.data, 16), 9), types.ShredAmountExact, types.ShredAmountMin)
	if got := ownerTokenDelta(tx, sell.accounts[13], sell.accounts[1]); got.Cmp(new(big.Int).Neg(new(big.Int).SetUint64(le64At(sell.data, 8)))) != 0 {
		t.Errorf("tokens sold %s, want -amount", got)
	}

	// legacy sell
	tx = loadFixture(t, sigPumpSell)
	sell = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, d.SELL)
	res = parseShred(t, tx, nil)
	pumpTrade(t, res, sell, types.TradeTypeSell, tok(sell.accounts[2], le64At(sell.data, 8), 6), tok(solMint, le64At(sell.data, 16), 9), types.ShredAmountExact, types.ShredAmountMin)
}

// TestShredPumpfunMigrate: migrate reported accounts[4]
// (associated_bonding_curve) as the quote mint; the IDL has wsol_mint at 14.
// migrate_v2 names the quote mint at 3. The pool must be the PumpSwap pool
// the migration creates. shred-15.
func TestShredPumpfunMigrate(t *testing.T) {
	for _, c := range []struct {
		sig             string
		disc            []byte
		quote, pool, bc int
	}{
		{sigPumpMigrate, constants.DISCRIMINATORS.PUMPFUN.MIGRATE, 14, 9, 3},
		{sigPumpMigrateV2, constants.DISCRIMINATORS.PUMPFUN.MIGRATE_V2, 3, 10, 4},
	} {
		tx := loadFixture(t, c.sig)
		ix := findIx(t, tx, constants.DEX_PROGRAMS.PUMP_FUN.ID, c.disc)
		createPool := findIx(t, tx, constants.DEX_PROGRAMS.PUMP_SWAP.ID, constants.DISCRIMINATORS.PUMPSWAP.CREATE_POOL)
		res := parseShred(t, tx, nil)

		var data *dexparser.PumpfunMigrateData
		for _, e := range res.Instructions[constants.DEX_PROGRAMS.PUMP_FUN.Name] {
			if m, ok := e.(*dexparser.PumpfunInstruction).Data.(*dexparser.PumpfunMigrateData); ok {
				data = m
			}
		}
		if data == nil {
			t.Fatalf("%x: no migrate event", c.disc)
		}
		if data.QuoteMint != ix.accounts[c.quote] || data.QuoteMint != solMint {
			t.Errorf("%s: quote mint %s, want wsol %s", data.Instruction, data.QuoteMint, ix.accounts[c.quote])
		}
		if data.Pool != createPool.accounts[0] || data.BondingCurve != ix.accounts[c.bc] {
			t.Errorf("%s: pool %s curve %s, want created pool %s and curve %s", data.Instruction, data.Pool, data.BondingCurve, createPool.accounts[0], ix.accounts[c.bc])
		}
		m := oneTypedAt(t, res, constants.DEX_PROGRAMS.PUMP_FUN.ID, utils.FormatIdx(ix.outer, ix.inner)).MemeEvent
		if m.Type != types.TradeTypeMigrate || m.QuoteMint != solMint || m.Pool != createPool.accounts[0] || m.PoolDex != constants.DEX_PROGRAMS.PUMP_SWAP.Name {
			t.Errorf("%s: meme event %+v", data.Instruction, m)
		}
	}
}

// TestShredPumpswapInstructions: buy_exact_quote_in was not decoded, PumpSwap
// had no typed output and create_pool named quote_amount_in QuoteAmountOut.
// shred-17, shred-20, shred-30.
func TestShredPumpswapInstructions(t *testing.T) {
	d := constants.DISCRIMINATORS.PUMPSWAP
	check := func(res *types.ParseShredResult, ix rawIx, wantType types.TradeType, in, out types.TokenInfo, inKind, outKind types.ShredAmountKind) {
		t.Helper()
		ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.PUMP_SWAP.ID, utils.FormatIdx(ix.outer, ix.inner))
		tr := ins.Trade
		if tr == nil || tr.Type != wantType || tr.User != ix.accounts[1] || len(tr.Pool) != 1 || tr.Pool[0] != ix.accounts[0] {
			t.Fatalf("%s: trade %+v", ins.Action, tr)
		}
		if tr.InputToken.Mint != in.Mint || tr.InputToken.AmountRaw != in.AmountRaw || tr.OutputToken.Mint != out.Mint || tr.OutputToken.AmountRaw != out.AmountRaw {
			t.Errorf("%s: %s %s -> %s %s, want %s %s -> %s %s", ins.Action, tr.InputToken.Mint, tr.InputToken.AmountRaw, tr.OutputToken.Mint, tr.OutputToken.AmountRaw, in.Mint, in.AmountRaw, out.Mint, out.AmountRaw)
		}
		if ins.InputAmountKind != inKind || ins.OutputAmountKind != outKind {
			t.Errorf("%s: kinds %s/%s", ins.Action, ins.InputAmountKind, ins.OutputAmountKind)
		}
	}

	// buy_exact_quote_in (spendable_quote_in, min_base_amount_out)
	tx := loadFixture(t, sigPswapBuyExactQuoteIn)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.PUMP_SWAP.ID, d.BUY_EXACT_QUOTE_IN)
	res := parseShred(t, tx, nil)
	check(res, ix, types.TradeTypeBuy, tok(ix.accounts[4], le64At(ix.data, 8), 9), tok(ix.accounts[3], le64At(ix.data, 16), 6), types.ShredAmountExact, types.ShredAmountMin)
	raw := res.Instructions[constants.DEX_PROGRAMS.PUMP_SWAP.Name]
	if len(raw) != 1 {
		t.Fatalf("PumpSwap events %d", len(raw))
	}
	if b, ok := raw[0].(*dexparser.PumpswapInstruction).Data.(*dexparser.PumpswapBuyExactQuoteInData); !ok || b.SpendableQuoteIn != le64At(ix.data, 8) {
		t.Errorf("raw buy_exact_quote_in = %+v", raw[0])
	}

	// sell (base_amount_in, min_quote_amount_out)
	tx = loadFixture(t, sigPswapSell)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_SWAP.ID, d.SELL)
	res = parseShred(t, tx, nil)
	check(res, ix, types.TradeTypeSell, tok(ix.accounts[3], le64At(ix.data, 8), 6), tok(ix.accounts[4], le64At(ix.data, 16), 9), types.ShredAmountExact, types.ShredAmountMin)
	if got := ownerTokenDelta(tx, ix.accounts[1], ix.accounts[3]); got.Cmp(new(big.Int).Neg(new(big.Int).SetUint64(le64At(ix.data, 8)))) != 0 {
		t.Errorf("base sold %s, want -base_amount_in", got)
	}

	// buy (base_amount_out, max_quote_amount_in), after a legacy migrate
	tx = loadFixture(t, sigPumpMigrate)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_SWAP.ID, d.BUY)
	res = parseShred(t, tx, nil)
	check(res, ix, types.TradeTypeBuy, tok(ix.accounts[4], le64At(ix.data, 16), 9), tok(ix.accounts[3], le64At(ix.data, 8), 6), types.ShredAmountMax, types.ShredAmountExact)

	// create_pool (index u16, base_amount_in, quote_amount_in), inner
	ix = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_SWAP.ID, d.CREATE_POOL)
	var created *dexparser.PumpswapCreatePoolInstructionData
	for _, e := range res.Instructions[constants.DEX_PROGRAMS.PUMP_SWAP.Name] {
		if c, ok := e.(*dexparser.PumpswapInstruction).Data.(*dexparser.PumpswapCreatePoolInstructionData); ok {
			created = c
		}
	}
	baseIn, quoteIn := le64At(ix.data, 10), le64At(ix.data, 18)
	if created == nil || created.BaseAmountIn != baseIn || created.QuoteAmountIn != quoteIn || created.Pool != ix.accounts[0] {
		t.Errorf("create_pool = %+v, want base %d quote %d pool %s", created, baseIn, quoteIn, ix.accounts[0])
	}
	liq := oneTypedAt(t, res, constants.DEX_PROGRAMS.PUMP_SWAP.ID, utils.FormatIdx(ix.outer, ix.inner)).Liquidity
	if liq == nil || liq.Type != types.PoolEventTypeCreate || liq.PoolId != ix.accounts[0] || liq.Token0Mint != ix.accounts[3] || liq.Token1Mint != ix.accounts[4] ||
		liq.Token0AmountRaw != u64str(baseIn) || liq.Token1AmountRaw != u64str(quoteIn) {
		t.Errorf("create_pool pool event = %+v", liq)
	}

	// deposit (lp_token_amount_out, max_base_amount_in, max_quote_amount_in)
	tx = loadFixture(t, sigPswapDeposit)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.PUMP_SWAP.ID, d.ADD_LIQUIDITY)
	res = parseShred(t, tx, nil)
	add := oneTypedAt(t, res, constants.DEX_PROGRAMS.PUMP_SWAP.ID, utils.FormatIdx(ix.outer, ix.inner))
	liq = add.Liquidity
	if liq == nil || liq.Type != types.PoolEventTypeAdd || liq.User != ix.accounts[2] || liq.LpAmountRaw != u64str(le64At(ix.data, 8)) ||
		liq.Token0AmountRaw != u64str(le64At(ix.data, 16)) || liq.Token1AmountRaw != u64str(le64At(ix.data, 24)) || add.InputAmountKind != types.ShredAmountMax {
		t.Errorf("deposit pool event = %+v kinds %s/%s", liq, add.InputAmountKind, add.OutputAmountKind)
	}
	if got := ownerTokenDelta(tx, ix.accounts[2], ix.accounts[5]); got.Cmp(new(big.Int).SetUint64(le64At(ix.data, 8))) != 0 {
		t.Errorf("LP received %s, want lp_token_amount_out", got)
	}
}
