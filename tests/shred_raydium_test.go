package tests

import (
	"math/big"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Raydium AMM v4 and LaunchLab shred decoding against real transactions.
// Layouts: raydium-amm program/src/instruction.rs (swap_base_in with 18 or,
// from the program's own builder, 17 accounts; swap_base_in_v2 with 8;
// InitializeInstruction2 {nonce, open_time, init_pc_amount,
// init_coin_amount}); LaunchLab on-chain IDL v0.2.0. shred-4, shred-8,
// shred-14, shred-16, shred-22, shred-26, shred-27, constants-16.

const (
	// swap_base_in, 18 accounts, outer 4
	sigRaySwap18 = "5kaAWK5X9DdMmsWm6skaUXLd6prFisuYJavd9B62A941nRGcrmwvncg3tRtUfn7TcMLsrrmjCChdEjK3sjxS6YG9"
	// swap_base_in, 17 accounts, inner 2-1 (Jupiter CPI)
	sigRaySwap17 = "4mxr44yo5Qi7Rabwbknkh8MNUEWAMKmzFQEmqUVdx5JpHEEuh59TrqiMCjZ7mgZMozRK1zW8me34w8Myi8Qi1tWP"
	// swap_base_in, 17 accounts, inner 3-0 of a legacy tx
	sigRaySwap17Legacy = "2iHYs4AHC5nutcbBxpA5aptBYTGaDUYBgamohfetDnAPiPBW5NkguxgnjVF5886Jy8MZ19UXdeZyPKq9C5wqAki4"
	// swap_base_in_v2, 8 accounts, inner 5-9
	sigRaySwapV2 = "4gfccDphF5CgAYGQ6zWRAGeQ4wJge85CtQiBRqw1wyu7geDhMAgYCBMdK1jGAoHHE9pzyE6bMyHSuwnRv2ms5KaB"
	// initialize2, outer 4
	sigRayInit2 = "2YxPyAJNfnBLrVpBwMx7qMVNPSvBDhxiquwJGhBjwXhkP6i6AbooUg4b4wpi15bQq2Qs4t7BpL1UVvTMcXL8P4uS"
	// deposit, outer 4
	sigRayDeposit = "2S4DdkD4FpqazTn5qHd4x9X5bAHu9g5Ry3jCLWkE44UCmW8bwkcet9eeAsHbzyR7HWtiE3gV268MtsBNrc4X6KpL"
	// withdraw, 22 accounts, outer 5
	sigRayWithdraw = "2MvpoPWEY3gnEE5WxsQATRRa15Go6p8HxBbuxATiTUMxCRJeVQsBCPnCRdrM5YModikwpTiKu1iZbPBeTdHkg3uv"

	// LaunchLab initialize + buy_exact_in
	sigLcpInitBuy = "4x8k2aQKevA8yuCVX1V8EaH2GBqdbZ1dgYxwtkwZJ7SmCQeng7CCs17AvyjFv6nMoUkBgpBwLHAABdCxGHbAWxo4"
	// LaunchLab initialize_with_token_2022 (JTO quote) + buy_exact_in (v1 tx)
	sigLcpInit2022 = "4hW26D8Ue9UCrptobryYG55MkdJUKcqFLiitWPFkaofKShuHPQP7NDfbTmQVm5SGcwcZo5VNcVoqFq14t7UmJ9Ce"
	// LaunchLab buy_exact_out + sell_exact_in
	sigLcpBuyOutSellIn = "61AN23VGPknSqskF6CvtZqrD4LtL2CNGYKeyFc5nLVcfaUaHV5LsQe6HTnRFM6pNX8qf7fkqZ5tEZfnNEF73H8MX"
	// LaunchLab sell_exact_in at outer 3 of a v1 transaction
	sigLcpSellV1 = "3JG4tuLfr7dMAH52EsbdDsy2nqi3fYJuuyKBDAcsuqwKJuahL599UucAj9iohjJcV9XoiBpwgGKXAbwxGyPHC3bL"
	// LaunchLab sell_exact_in with a USD1 quote (inner 5-0)
	sigLcpUSD1 = "zuaKyxjpM7G5et2XqZofjjGNczNduGs6g8ipCEeZKKV7h6FFgRNJbXnzfufSZWD3bEacmf8sVktXpZaadQhmVuJ"
)

// TestShredRaydiumV4Swaps: the canonical 17-account swap was rejected,
// swap_base_out and the v2 swaps were not decoded, and the Mint fields held
// token accounts with 9 hardcoded decimals. shred-14, shred-22.
func TestShredRaydiumV4Swaps(t *testing.T) {
	d := constants.DISCRIMINATORS.RAYDIUM
	for _, c := range []struct {
		sig                  string
		tag                  []byte
		source, dest, owner  int
		wantIn, wantOutKnown bool
	}{
		{sigRaySwap18, d.SWAP, 15, 16, 17, true, false},
		{sigRaySwap17, d.SWAP, 14, 15, 16, true, true},
		{sigRaySwap17Legacy, d.SWAP, 14, 15, 16, true, true},
		{sigRaySwapV2, d.SWAP_V2, 5, 6, 7, true, false},
	} {
		tx := loadFixture(t, c.sig)
		ix := findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, c.tag)
		res := parseShred(t, tx, nil)
		ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, utils.FormatIdx(ix.outer, ix.inner))
		tr := ins.Trade
		if tr == nil {
			t.Fatalf("%d accounts: no trade", len(ix.accounts))
		}
		in, minOut := le64At(ix.data, 1), le64At(ix.data, 9)
		if tr.InputToken.AmountRaw != u64str(in) || tr.OutputToken.AmountRaw != u64str(minOut) || ins.InputAmountKind != types.ShredAmountExact || ins.OutputAmountKind != types.ShredAmountMin {
			t.Errorf("%d accounts: amounts %s/%s kinds %s/%s, want %d/%d exact/min", len(ix.accounts), tr.InputToken.AmountRaw, tr.OutputToken.AmountRaw, ins.InputAmountKind, ins.OutputAmountKind, in, minOut)
		}
		if tr.User != ix.accounts[c.owner] || tr.Pool[0] != ix.accounts[1] {
			t.Errorf("%d accounts: user %s pool %v, want %s %s", len(ix.accounts), tr.User, tr.Pool, ix.accounts[c.owner], ix.accounts[1])
		}
		// Mints are the mints of the user's token accounts, never the accounts
		srcMint := tokenBalanceMint(tx, ix.accounts[c.source])
		if tr.InputToken.Mint != srcMint || tr.InputToken.Mint == ix.accounts[c.source] || tr.InputToken.Decimals != tokenBalanceDecimals(tx, ix.accounts[c.source]) {
			t.Errorf("%d accounts: input %s (%d dec), want mint %s of %s", len(ix.accounts), tr.InputToken.Mint, tr.InputToken.Decimals, srcMint, ix.accounts[c.source])
		}
		// The output mint is the destination account's mint, or when the
		// transaction does not list that account, the pool vault mint the
		// input does not use
		wantOut := tokenBalanceMint(tx, ix.accounts[c.dest])
		if wantOut == "" {
			vault := map[int]int{18: 5, 17: 4, 8: 3}[len(ix.accounts)]
			coin, pc := tokenBalanceMint(tx, ix.accounts[vault]), tokenBalanceMint(tx, ix.accounts[vault+1])
			switch srcMint {
			case coin:
				wantOut = pc
			case pc:
				wantOut = coin
			}
		}
		if c.wantOutKnown && (wantOut == "" || tr.OutputToken.Mint != wantOut) {
			t.Errorf("%d accounts: output mint %s, want %s", len(ix.accounts), tr.OutputToken.Mint, wantOut)
		}
		if tr.OutputToken.Mint == ix.accounts[c.dest] {
			t.Errorf("%d accounts: output mint is the destination token account", len(ix.accounts))
		}
	}

	// The 18-account swap: the user's source account lost exactly amount_in
	tx := loadFixture(t, sigRaySwap18)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, d.SWAP)
	if got := accountTokenDelta(tx, ix.accounts[15], tokenBalanceMint(tx, ix.accounts[15])); got.Cmp(new(big.Int).Neg(new(big.Int).SetUint64(le64At(ix.data, 1)))) != 0 {
		t.Errorf("source delta %s, want -amount_in", got)
	}
}

// TestShredRaydiumV4SwapBaseOut: tag 11 (max_amount_in, amount_out) was not
// decoded. No swap_base_out was found in the fixtures or sampled blocks, so
// the test re-tags the real 18-account swap_base_in of 5kaAWK5X (same account
// layout, instruction.rs swap_base_out). shred-14.
func TestShredRaydiumV4SwapBaseOut(t *testing.T) {
	tx := preExec(t, loadFixture(t, sigRaySwap18))
	ix := findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, constants.DISCRIMINATORS.RAYDIUM.SWAP)
	data := append([]byte{}, ix.data...)
	data[0] = constants.DISCRIMINATORS.RAYDIUM.SWAP_EXACT_OUT[0]
	setIxData(ix, data)

	res := parseShred(t, tx, nil)
	ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, utils.FormatIdx(ix.outer, -1))
	if ins.Action != "swap_base_out" || ins.Trade.InputToken.AmountRaw != u64str(le64At(data, 1)) || ins.Trade.OutputToken.AmountRaw != u64str(le64At(data, 9)) ||
		ins.InputAmountKind != types.ShredAmountMax || ins.OutputAmountKind != types.ShredAmountExact {
		t.Errorf("swap_base_out = %s %s/%s %s/%s", ins.Action, ins.Trade.InputToken.AmountRaw, ins.Trade.OutputToken.AmountRaw, ins.InputAmountKind, ins.OutputAmountKind)
	}
	raw := res.Instructions[constants.DEX_PROGRAMS.RAYDIUM_V4.Name][0].(*raydium.RaydiumV4ShredInstruction).Data.(*raydium.RaydiumV4SwapData)
	if !raw.ExactOut || raw.User != ix.accounts[17] {
		t.Errorf("raw swap_base_out = %+v", raw)
	}
}

// TestShredRaydiumV4Liquidity: initialize2 skipped a field (base 0, quote =
// init_coin_amount); deposit and withdraw reported the pool vaults as mints.
// shred-8, shred-26.
func TestShredRaydiumV4Liquidity(t *testing.T) {
	d := constants.DISCRIMINATORS.RAYDIUM

	// initialize2: nonce u8, open_time u64, init_pc_amount, init_coin_amount
	tx := loadFixture(t, sigRayInit2)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, d.CREATE)
	initPc, initCoin := le64At(ix.data, 10), le64At(ix.data, 18)
	res := parseShred(t, tx, nil)
	liq := oneTypedAt(t, res, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, utils.FormatIdx(ix.outer, -1)).Liquidity
	if liq.Token0Mint != ix.accounts[8] || liq.Token0AmountRaw != u64str(initCoin) || liq.Token1Mint != ix.accounts[9] || liq.Token1AmountRaw != u64str(initPc) {
		t.Errorf("initialize2 = %s %s / %s %s, want coin %s %d / pc %s %d", liq.Token0Mint, liq.Token0AmountRaw, liq.Token1Mint, liq.Token1AmountRaw, ix.accounts[8], initCoin, ix.accounts[9], initPc)
	}
	// The pool vaults hold exactly the initial amounts afterwards
	if got := accountTokenDelta(tx, ix.accounts[10], ix.accounts[8]); got.Cmp(new(big.Int).SetUint64(initCoin)) != 0 {
		t.Errorf("coin vault received %s, want init_coin_amount %d", got, initCoin)
	}
	if got := accountTokenDelta(tx, ix.accounts[11], ix.accounts[9]); got.Cmp(new(big.Int).SetUint64(initPc)) != 0 {
		t.Errorf("pc vault received %s, want init_pc_amount %d", got, initPc)
	}
	if liq.User != ix.accounts[17] || liq.PoolId != ix.accounts[4] || *liq.Token0Decimals != tokenBalanceDecimals(tx, ix.accounts[10]) {
		t.Errorf("initialize2 user %s pool %s dec %d", liq.User, liq.PoolId, *liq.Token0Decimals)
	}

	// deposit: max_coin_amount, max_pc_amount; accounts 6/7 are vaults
	tx = loadFixture(t, sigRayDeposit)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, d.ADD_LIQUIDITY)
	res = parseShred(t, tx, nil)
	add := oneTypedAt(t, res, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, utils.FormatIdx(ix.outer, -1))
	liq = add.Liquidity
	coinMint, pcMint := tokenBalanceMint(tx, ix.accounts[6]), tokenBalanceMint(tx, ix.accounts[7])
	if liq.Token0Mint != coinMint || liq.Token1Mint != pcMint || liq.Token0Mint == ix.accounts[6] || liq.User != ix.accounts[12] {
		t.Errorf("deposit mints %s/%s user %s, want %s/%s %s", liq.Token0Mint, liq.Token1Mint, liq.User, coinMint, pcMint, ix.accounts[12])
	}
	maxCoin, maxPc := le64At(ix.data, 1), le64At(ix.data, 9)
	if liq.Token0AmountRaw != u64str(maxCoin) || liq.Token1AmountRaw != u64str(maxPc) || add.InputAmountKind != types.ShredAmountMax {
		t.Errorf("deposit amounts %s/%s kind %s, want max %d/%d", liq.Token0AmountRaw, liq.Token1AmountRaw, add.InputAmountKind, maxCoin, maxPc)
	}
	if got := accountTokenDelta(tx, ix.accounts[6], coinMint); got.Sign() <= 0 || got.Cmp(new(big.Int).SetUint64(maxCoin)) > 0 {
		t.Errorf("coin deposited %s, want within (0, max %d]", got, maxCoin)
	}
	raw := res.Instructions[constants.DEX_PROGRAMS.RAYDIUM_V4.Name][0].(*raydium.RaydiumV4ShredInstruction).Data.(*raydium.RaydiumV4LiquidityData)
	if raw.BaseVault != ix.accounts[6] || raw.QuoteVault != ix.accounts[7] || raw.BaseMint != coinMint {
		t.Errorf("raw deposit = %+v", raw)
	}

	// withdraw (22 accounts): amount, min_coin_amount, min_pc_amount; owner at 18
	tx = loadFixture(t, sigRayWithdraw)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, d.REMOVE_LIQUIDITY)
	res = parseShred(t, tx, nil)
	remove := oneTypedAt(t, res, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, utils.FormatIdx(ix.outer, -1))
	liq = remove.Liquidity
	lp, minCoin, minPc := le64At(ix.data, 1), le64At(ix.data, 9), le64At(ix.data, 17)
	coinMint, pcMint = tokenBalanceMint(tx, ix.accounts[6]), tokenBalanceMint(tx, ix.accounts[7])
	if liq.User != ix.accounts[18] || liq.Token0Mint != coinMint || liq.Token1Mint != pcMint || liq.LpAmountRaw != u64str(lp) ||
		liq.Token0AmountRaw != u64str(minCoin) || liq.Token1AmountRaw != u64str(minPc) || remove.OutputAmountKind != types.ShredAmountMin {
		t.Errorf("withdraw = %+v kind %s", liq, remove.OutputAmountKind)
	}
	if got := ownerTokenDelta(tx, ix.accounts[18], ix.accounts[5]); got.Cmp(new(big.Int).Neg(new(big.Int).SetUint64(lp))) != 0 {
		t.Errorf("LP burned %s, want -amount %d", got, lp)
	}
}

// lcpMeme returns the typed LaunchLab meme event of an instruction
func lcpMeme(t *testing.T, res *types.ParseShredResult, ix rawIx) types.ParsedShredInstruction {
	t.Helper()
	ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.RAYDIUM_LCP.ID, utils.FormatIdx(ix.outer, ix.inner))
	if ins.MemeEvent == nil {
		t.Fatalf("%s: no meme event", ins.Action)
	}
	return ins
}

// TestShredLaunchLabTrades: buys reported the quote mint as BaseMint and the
// base mint as QuoteMint; decimals were hardcoded 9/6 (USD1 and JTO quotes
// exist). shred-4, shred-22.
func TestShredLaunchLabTrades(t *testing.T) {
	d := constants.DISCRIMINATORS.RAYDIUM_LCP
	for _, c := range []struct {
		sig             string
		disc            []byte
		buy, exactOut   bool
		inKind, outKind types.ShredAmountKind
	}{
		{sigLcpInitBuy, d.BUY_EXACT_IN, true, false, types.ShredAmountExact, types.ShredAmountMin},
		{sigLcpBuyOutSellIn, d.BUY_EXACT_OUT, true, true, types.ShredAmountMax, types.ShredAmountExact},
		{sigLcpBuyOutSellIn, d.SELL_EXACT_IN, false, false, types.ShredAmountExact, types.ShredAmountMin},
		{sigLcpUSD1, d.SELL_EXACT_IN, false, false, types.ShredAmountExact, types.ShredAmountMin},
		{sigLcpSellV1, d.SELL_EXACT_IN, false, false, types.ShredAmountExact, types.ShredAmountMin},
		{sigLcpInit2022, d.BUY_EXACT_IN, true, false, types.ShredAmountExact, types.ShredAmountMin},
	} {
		tx := loadFixture(t, c.sig)
		ix := findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_LCP.ID, c.disc)
		res := parseShred(t, tx, nil)
		ins := lcpMeme(t, res, ix)
		m := ins.MemeEvent
		base, quote := ix.accounts[9], ix.accounts[10]
		if m.BaseMint != base || m.QuoteMint != quote || m.User != ix.accounts[0] || m.Pool != ix.accounts[4] || m.PlatformConfig != ix.accounts[3] {
			t.Errorf("%s: base %s quote %s user %s pool %s, want %s %s %s %s", ins.Action, m.BaseMint, m.QuoteMint, m.User, m.Pool, base, quote, ix.accounts[0], ix.accounts[4])
		}
		first, second := le64At(ix.data, 8), le64At(ix.data, 16)
		in, out := first, second
		if c.exactOut {
			in, out = second, first
		}
		inMint, outMint := quote, base
		wantType := types.TradeTypeBuy
		if !c.buy {
			inMint, outMint, wantType = base, quote, types.TradeTypeSell
		}
		if m.Type != wantType || m.InputToken.Mint != inMint || m.OutputToken.Mint != outMint || m.InputToken.AmountRaw != u64str(in) || m.OutputToken.AmountRaw != u64str(out) {
			t.Errorf("%s: %s %s %s -> %s %s, want %s %s %d -> %s %d", ins.Action, m.Type, m.InputToken.Mint, m.InputToken.AmountRaw, m.OutputToken.Mint, m.OutputToken.AmountRaw, wantType, inMint, in, outMint, out)
		}
		if ins.InputAmountKind != c.inKind || ins.OutputAmountKind != c.outKind {
			t.Errorf("%s: kinds %s/%s, want %s/%s", ins.Action, ins.InputAmountKind, ins.OutputAmountKind, c.inKind, c.outKind)
		}
		// Decimals from the transaction's token balances (D6)
		wantIn, wantOut := mintDecimals(tx, inMint), mintDecimals(tx, outMint)
		if m.InputToken.Decimals != wantIn || m.OutputToken.Decimals != wantOut {
			t.Errorf("%s: decimals %d/%d, want %d/%d", ins.Action, m.InputToken.Decimals, m.OutputToken.Decimals, wantIn, wantOut)
		}
	}

	// Pre-execution: the base mint's decimals are unknown (0), not guessed
	tx := preExec(t, loadFixture(t, sigLcpBuyOutSellIn))
	ix := findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_LCP.ID, d.BUY_EXACT_OUT)
	m := lcpMeme(t, parseShred(t, tx, nil), ix).MemeEvent
	if m.InputToken.Decimals != 9 || m.OutputToken.Decimals != 0 {
		t.Errorf("pre-exec decimals %d/%d, want 9 (WSOL) / 0 (unknown)", m.InputToken.Decimals, m.OutputToken.Decimals)
	}
}

// TestShredLaunchLabCreate: initialize_v2 and initialize_with_token_2022
// were not decoded. shred-27, constants-16.
func TestShredLaunchLabCreate(t *testing.T) {
	d := constants.DISCRIMINATORS.RAYDIUM_LCP
	check := func(tx *adapterTx, disc []byte) {
		t.Helper()
		ix := findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_LCP.ID, disc)
		ins := lcpMeme(t, parseShred(t, tx, nil), ix)
		m := ins.MemeEvent
		// MintParams: decimals u8, name, symbol, uri
		nameLen := int(le32At(ix.data, 9))
		name := string(ix.data[13 : 13+nameLen])
		if m.Type != types.TradeTypeCreate || m.User != ix.accounts[1] || m.Pool != ix.accounts[5] || m.BaseMint != ix.accounts[6] || m.QuoteMint != ix.accounts[7] ||
			m.Name != name || m.Decimals == nil || *m.Decimals != ix.data[8] || m.PlatformConfig != ix.accounts[3] {
			t.Errorf("%x: create = %+v, want name %q", disc, m, name)
		}
	}
	check(loadFixture(t, sigLcpInitBuy), d.INITIALIZE)
	check(loadFixture(t, sigLcpInit2022), d.INITIALIZE_WITH_TOKEN_2022)

	// No initialize_v2 was found in the fixtures or sampled blocks; its
	// accounts 0-7 and first argument (MintParams) are those of initialize
	// (IDL v0.2.0), so the test re-tags the real initialize of 4x8k2aQK
	tx := preExec(t, loadFixture(t, sigLcpInitBuy))
	ix := findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_LCP.ID, d.INITIALIZE)
	setIxData(ix, append(append([]byte{}, d.INITIALIZE_V2...), ix.data[8:]...))
	check(tx, d.INITIALIZE_V2)
}

// TestShredLaunchLabMigrate: migrate_to_amm reported the authority account
// (13) as the pool; the on-chain IDL has amm_pool at 5 and pool_state at 14.
// No migrate transaction was found in the fixtures or sampled blocks, so the
// test appends migrate instructions with the IDL account layouts to a real
// LaunchLab transaction. shred-16.
func TestShredLaunchLabMigrate(t *testing.T) {
	tx := preExec(t, loadFixture(t, sigLcpInitBuy))
	keys := rawAccountKeys(tx)
	lcp := -1
	for i, k := range keys {
		if k == constants.DEX_PROGRAMS.RAYDIUM_LCP.ID {
			lcp = i
		}
	}
	accounts := func(n int) []interface{} {
		out := make([]interface{}, n)
		for i := range out {
			out[i] = i % len(keys)
		}
		return out
	}
	d := constants.DISCRIMINATORS.RAYDIUM_LCP
	tx.Transaction.Message.Instructions = append(tx.Transaction.Message.Instructions,
		map[string]interface{}{"programIdIndex": lcp, "accounts": accounts(23), "data": base58Encode(d.MIGRATE_TO_AMM)},
		map[string]interface{}{"programIdIndex": lcp, "accounts": accounts(28), "data": base58Encode(d.MIGRATE_TO_CPSWAP)},
	)
	n := len(tx.Transaction.Message.Instructions)
	res := parseShred(t, tx, nil)

	amm := oneTypedAt(t, res, constants.DEX_PROGRAMS.RAYDIUM_LCP.ID, utils.FormatIdx(n-2, -1)).MemeEvent
	if amm.Pool != keys[5%len(keys)] || amm.BondingCurve != keys[14%len(keys)] || amm.BaseMint != keys[1] || amm.QuoteMint != keys[2] || amm.PoolDex != constants.DEX_PROGRAMS.RAYDIUM_V4.Name {
		t.Errorf("migrate_to_amm = %+v, want pool = account 5, curve = account 14", amm)
	}
	cp := oneTypedAt(t, res, constants.DEX_PROGRAMS.RAYDIUM_LCP.ID, utils.FormatIdx(n-1, -1)).MemeEvent
	if cp.Pool != keys[5%len(keys)] || cp.BondingCurve != keys[17%len(keys)] || cp.PoolDex != constants.DEX_PROGRAMS.RAYDIUM_CPMM.Name {
		t.Errorf("migrate_to_cpswap = %+v, want pool = account 5, curve = account 17", cp)
	}
}
