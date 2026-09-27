package tests

import (
	"encoding/base64"
	"encoding/binary"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/meteora"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/orca"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// tokenDelta returns post - pre of a token account (raw)
func tokenDelta(ctx *parseContext, account string) *big.Int {
	amount := func(b *types.TokenAmount) *big.Int {
		v := new(big.Int)
		if b != nil {
			v.SetString(b.Amount, 10)
		}
		return v
	}
	post := amount(ctx.Adapter.GetTokenAccountBalance([]string{account})[0])
	pre := amount(ctx.Adapter.GetTokenAccountPreBalance([]string{account})[0])
	return post.Sub(post, pre)
}

// transferAmount returns the amount of the SPL Token transfer or
// transferChecked instruction at (outer, inner)
func transferAmount(t *testing.T, ctx *parseContext, outer, inner int) string {
	t.Helper()
	data := ctx.Adapter.GetInstructionData(ctx.Adapter.GetInnerInstruction(outer, inner))
	if len(data) < 9 || (data[0] != 3 && data[0] != 12) {
		t.Fatalf("%d-%d is not a token transfer", outer, inner)
	}
	return strconv.FormatUint(binary.LittleEndian.Uint64(data[1:9]), 10)
}

// programTrades runs a trade parser over all instructions of programId
func programTrades(ctx *parseContext, programId string, newParser func(*adapter.TransactionAdapter, types.DexInfo, map[string][]types.TransferData, []types.ClassifiedInstruction) interface{ ProcessTrades() []types.TradeInfo }) []types.TradeInfo {
	dexInfo := types.DexInfo{ProgramId: programId, AMM: constants.GetProgramName(programId)}
	return newParser(ctx.Adapter, dexInfo, ctx.TransferActions, ctx.Classifier.GetInstructions(programId)).ProcessTrades()
}

func orcaParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) interface{ ProcessTrades() []types.TradeInfo } {
	return orca.NewOrcaParser(a, d, t, c)
}

func raydiumParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) interface{ ProcessTrades() []types.TradeInfo } {
	return raydium.NewRaydiumParser(a, d, t, c)
}

func meteoraParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) interface{ ProcessTrades() []types.TradeInfo } {
	return meteora.NewMeteoraParser(a, d, t, c)
}

func tradeAt(t *testing.T, trades []types.TradeInfo, idx string) types.TradeInfo {
	t.Helper()
	for _, tr := range trades {
		if tr.Idx == idx {
			return tr
		}
	}
	t.Fatalf("no trade at idx %s: %+v", idx, trades)
	return types.TradeInfo{}
}

func feeOfType(trade types.TradeInfo, feeType string) string {
	if trade.Fee != nil && trade.Fee.Type == feeType {
		return trade.Fee.AmountRaw
	}
	for _, f := range trade.Fees {
		if f.Type == feeType {
			return f.AmountRaw
		}
	}
	return ""
}

func u64s(data []byte, offset int) string {
	return strconv.FormatUint(binary.LittleEndian.Uint64(data[offset:offset+8]), 10)
}

// TestAmmNoTradesFromLiquidityInstructions: pool creation, liquidity and fee
// instructions move tokens but are not swaps (amm-6, constants-2, parity-7,
// parity-16, constants-v1). Real transactions; before the fix each produced
// a trade.
func TestAmmNoTradesFromLiquidityInstructions(t *testing.T) {
	cases := []struct {
		sig, note, liquidity string
	}{
		{"5AzH3HApZUEnGECG5Xk26jgUpbiRAzRsAqTRHiTj7Jf6bX3jfSXZCj5zqREvgnYwzgD2mUXw9g6FrN2hVtJZF5JN", "Orca collect_fees_v2 (with a Memo CPI)", ""},
		{"3874qjiBkmSNk3rRMEst2fAfwSx9jPNNi3sCcFBxETzEYxpPeRnU9emKz26M2x3ttxJGJmjV4ctZziQMFmDgKBkZ", "DAMM v2 initialize_pool (EvtCreatePosition CPI)", "CREATE"},
		{"3qiyKhA1zXD6Mvu7sxDfi8CpY5pfGhTJbRSjSPdsTmmr5kapM4EMYfqyWvAWxVLeUa53BtbxxxXMEar3wbDtaD9r", "DAMM v2 initialize_pool", "CREATE"},
		{"5yrQXvm3LHuJo9RChAswShLUnYJJDLPRUEWu6WmYZ9xsKFutbtQokgG6duU4gy5C6N3MXrM2a7c622bu49agW2uA", "DBC migration to DAMM v2", "CREATE"},
		{"2P2i1ZR2JctufV5QDgfuhb42SNARTtyDoeiBV7i6hgEjuDfQkLCSyJtdpmRcCArBehMZrLkTw9viXXMvgTcwVDyD", "DBC migration to DAMM v2", "CREATE"},
		{"2YxPyAJNfnBLrVpBwMx7qMVNPSvBDhxiquwJGhBjwXhkP6i6AbooUg4b4wpi15bQq2Qs4t7BpL1UVvTMcXL8P4uS", "Raydium V4 initialize2 (OpenBook CPI)", "CREATE"},
		{"h3sGiriW4jCGgbnNF8DaEKnsWhjtcH5ZM1dkyZFidgD2fx9aDNatray38yRmkxaWez3g5qFNpyXhE8ho716vjgp", "DLMM add_liquidity2", "ADD"},
		{"Cj2c5dEmHvmMWwkMa4QMauQE6aBbyRz5mn4fEYARez2bHqukkJ3nbYAdst9ixQsAMh9G9tUNntAxEXpgrz5T1Qi", "DLMM remove_liquidity", "REMOVE"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.sig[:12], func(t *testing.T) {
			r := dexparser.NewDexParser().ParseAll(loadFixture(t, c.sig), nil)
			for _, tr := range r.Trades {
				t.Errorf("%s: unexpected trade %s %s %s:%s -> %s:%s at %s", c.note, tr.AMM, tr.Type,
					tr.InputToken.Mint, tr.InputToken.AmountRaw, tr.OutputToken.Mint, tr.OutputToken.AmountRaw, tr.Idx)
			}
			if c.liquidity != "" {
				found := false
				for _, e := range r.Liquidities {
					found = found || string(e.Type) == c.liquidity
				}
				if !found {
					t.Errorf("%s: no %s liquidity event: %+v", c.note, c.liquidity, r.Liquidities)
				}
			}
		})
	}
}

// patchInstructionData replaces the data of the outer instruction at index
// (JSON-decoded fixtures keep instructions as maps)
func patchInstructionData(t *testing.T, tx *adapter.SolanaTransaction, outer int, data []byte) {
	t.Helper()
	ix, ok := tx.Transaction.Message.Instructions[outer].(map[string]interface{})
	if !ok {
		t.Fatalf("instruction %d is not a JSON map", outer)
	}
	ix["data"] = base58.Encode(data)
}

// TestAmmSwapWhitelist: only swap instructions become trades. There is no
// real transaction for these instructions in the fixture pool, so each case
// takes a real liquidity transaction and replaces the discriminator of its
// liquidity instruction (synthetic, built here). The transfers stay as they
// were, which the old exclusion lists turned into trades (amm-6,
// constants-2).
func TestAmmSwapWhitelist(t *testing.T) {
	d := constants.DISCRIMINATORS
	cases := []struct {
		name, sig string
		outer     int
		disc      []byte
		program   string
	}{
		{"Orca decrease_liquidity_v2", "23zkGAorUC3aHSk7zJYiUvvs6gEXPPzp8xRiWfACzkrqBEaQrKiH9QCgrmwSTD6hxKKEjryEGbEvurt6xSBpuBMC", 5, d.ORCA.REMOVE_LIQUIDITY_V2, constants.DEX_PROGRAMS.ORCA.ID},
		{"Orca collect_fees_v2", "23zkGAorUC3aHSk7zJYiUvvs6gEXPPzp8xRiWfACzkrqBEaQrKiH9QCgrmwSTD6hxKKEjryEGbEvurt6xSBpuBMC", 5, d.ORCA.COLLECT_FEES_V2, constants.DEX_PROGRAMS.ORCA.ID},
		{"Orca increase_liquidity_by_token_amounts_v2", "3mZcyeDJysgs79nLcvtN4XQ6iepyERqG93P2F2ZYgUX4ZF1Yr1XFBMKR8DHd7z4gN2EmvAqMc3KhQTQpGMbtvhF7", 5, d.ORCA.ADD_LIQUIDITY_BY_TOKEN_AMOUNTS_V2, constants.DEX_PROGRAMS.ORCA.ID},
		{"CPMM collect_creator_fee", "5eVe9LMAHgz6Ze7VrUn1XHgoaWJXinM2uoEvRciVoJsADCDc8v4HrewQorp4JUx3jAzBSw5p3RrSAYFqd95udWxR", 4, d.RAYDIUM_CPMM.COLLECT_CREATOR_FEE, constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID},
		{"CLMM collect_fund_fee", "48oGGt6rsBqbiyj7xyzWD8oRXk3sGhE6Kt6LzRG5QofL1LJqwtcfp4uKXeinDn4a24uyWJVGDPTaAFac2R5eNX1w", 4, d.RAYDIUM_CL.OTHER.COLLECT_FUND_FEE, constants.DEX_PROGRAMS.RAYDIUM_CL.ID},
		{"DLMM rebalance_liquidity", "7YPF21r7JBDeoXuMJn6KSqDVYGrm821U87Cnje3xPvZpMUVaAEAvCGJPP6va2b5oMLAzGku5s3TcNAsN6zdXPRn", 10, d.METEORA_DLMM.OTHER["rebalanceLiquidity"], constants.DEX_PROGRAMS.METEORA.ID},
		{"DAMM v2 claim_reward", "5S7ikDVhmBxHiRoVxCECYr9NZzXgoUaJKxSNmqcvfDTVh2FVnCQfyo8sQoUcFdAjBgGNsSHYCBD8vadhf7k2kQ3w", 2, d.METEORA_DAMM_V2.CLAIM_REWARD, constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID},
		{"DAMM v1 partner_claim_fee", "2xEAewTjtSHgpEHHzaNjHiuoMHNZQXz5vySXHCSL8omujjvaxq9JsfGWjusz43ndmzcu5riESKm1UH4riWDX9v1v", 1, d.METEORA_DAMM.PARTNER_CLAIM_FEE, constants.DEX_PROGRAMS.METEORA_DAMM.ID},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			tx := loadFixture(t, c.sig)
			ctx := newParseContext(tx, nil)
			data := append([]byte(nil), ctx.Adapter.GetInstructionData(ctx.Adapter.InstructionAt(c.outer))...)
			if got := ctx.Adapter.GetInstructionProgramId(ctx.Adapter.InstructionAt(c.outer)); got != c.program {
				t.Fatalf("outer %d is %s", c.outer, got)
			}
			copy(data, c.disc)
			patchInstructionData(t, tx, c.outer, data)

			r := dexparser.NewDexParser().ParseAll(tx, &types.ParseConfig{ParseType: types.ParseType{Trade: true}})
			for _, tr := range r.Trades {
				if strings.HasPrefix(tr.Idx, strconv.Itoa(c.outer)) {
					t.Errorf("unexpected trade %s %s %s:%s -> %s:%s at %s", tr.AMM, tr.Type,
						tr.InputToken.Mint, tr.InputToken.AmountRaw, tr.OutputToken.Mint, tr.OutputToken.AmountRaw, tr.Idx)
				}
			}
		})
	}
}

// TestOrcaTradePoolAndTradedEvent: Orca trades carry the whirlpool (swap:
// account 2, swap_v2: account 4) (amm-22). With a Traded event, the fees
// come from it and the amounts equal the swap's transfers (streamer item 8).
func TestOrcaTradePoolAndTradedEvent(t *testing.T) {
	// swap (v1), no Traded event in this older transaction
	ctx := newParseContext(loadFixture(t, "2kAW5GAhPZjM3NoSrhJVHdEpwjmq9neWtckWnjopCfsmCGB27e3v2ZyMM79FdsL4VWGEtYSFi1sF1Zhs7bqdoaVT"), nil)
	trades := programTrades(ctx, constants.DEX_PROGRAMS.ORCA.ID, orcaParser)
	swapIx := instructionAt(t, ctx, constants.DEX_PROGRAMS.ORCA.ID, 3, -1)
	if len(trades) != 1 || len(trades[0].Pool) != 1 || trades[0].Pool[0] != ctx.Adapter.GetInstructionAccounts(swapIx.Instruction)[2] {
		t.Errorf("swap trade pool %+v, want the whirlpool account 2", trades)
	}

	// swap_v2 via Jupiter with Traded events (inner 4-0 and 4-6)
	ctx = newParseContext(loadFixture(t, "5gpxeUJjYMmPifsT2JQq1BmvYdTLunsub5JiwrfhiE1X29HyHKtJSjpk5kkeqiLoGW5DbEN8qscijvrqGPMdgY8y"), nil)
	trades = programTrades(ctx, constants.DEX_PROGRAMS.ORCA.ID, orcaParser)
	var traded [][]byte
	for _, l := range ctx.Utils.GetProgramDataLogs() {
		if l.ProgramId == constants.DEX_PROGRAMS.ORCA.ID && constants.MatchDiscriminator(l.Data, constants.DISCRIMINATORS.ORCA.TRADED_EVENT) {
			traded = append(traded, l.Data)
		}
	}
	if len(traded) != 2 || len(trades) != 2 {
		t.Fatalf("want 2 Traded events and 2 trades, got %d / %d", len(traded), len(trades))
	}
	for i, c := range []struct {
		inner          int
		transferIn     [2]int
		transferOutIdx [2]int
	}{{0, [2]int{4, 1}, [2]int{4, 2}}, {6, [2]int{4, 7}, [2]int{4, 8}}} {
		tr := trades[i]
		ev := traded[i]
		// Traded (whirlpool 0.9.0 IDL): disc, whirlpool, a_to_b, pre/post_sqrt_price u128,
		// input_amount @73, output_amount @81, transfer fees @89/@97, lp_fee @105, protocol_fee @113
		whirlpool := base58.Encode(ev[8:40])
		accounts := ctx.Adapter.GetInstructionAccounts(instructionAt(t, ctx, constants.DEX_PROGRAMS.ORCA.ID, 4, c.inner).Instruction)
		if whirlpool != accounts[4] || len(tr.Pool) != 1 || tr.Pool[0] != whirlpool {
			t.Errorf("hop %d: pool %v, Traded whirlpool %s, account 4 %s", i, tr.Pool, whirlpool, accounts[4])
		}
		if tr.InputToken.AmountRaw != transferAmount(t, ctx, c.transferIn[0], c.transferIn[1]) ||
			tr.OutputToken.AmountRaw != transferAmount(t, ctx, c.transferOutIdx[0], c.transferOutIdx[1]) {
			t.Errorf("hop %d: amounts %s -> %s differ from the transfers", i, tr.InputToken.AmountRaw, tr.OutputToken.AmountRaw)
		}
		if tr.Fee == nil || tr.Fee.AmountRaw != u64s(ev, 105) || tr.Fee.Mint != tr.InputToken.Mint {
			t.Errorf("hop %d: fee %+v, Traded lp_fee %s", i, tr.Fee, u64s(ev, 105))
		}
		if feeOfType(tr, "protocol") != u64s(ev, 113) {
			t.Errorf("hop %d: protocol fee %s, Traded protocol_fee %s", i, feeOfType(tr, "protocol"), u64s(ev, 113))
		}
	}
}

// TestRaydiumV4SwapFromRayLog: an AMM v4 swap (called by the Raydium route
// program) takes its amounts from the ray_log SwapBaseIn; they equal the
// user's input balance change and the swap's transfers (the output account is
// a temporary WSOL account closed in the same transaction) (streamer item 7).
func TestRaydiumV4SwapFromRayLog(t *testing.T) {
	tx := loadFixture(t, "2iHYs4AHC5nutcbBxpA5aptBYTGaDUYBgamohfetDnAPiPBW5NkguxgnjVF5886Jy8MZ19UXdeZyPKq9C5wqAki4")
	ctx := newParseContext(tx, nil)
	trades := programTrades(ctx, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, raydiumParser)
	if len(trades) != 1 {
		t.Fatalf("want 1 trade, got %+v", trades)
	}
	tr := trades[0]
	accounts := ctx.Adapter.GetInstructionAccounts(instructionAt(t, ctx, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, 3, 0).Instruction)
	userSource, userDest := accounts[len(accounts)-3], accounts[len(accounts)-2]
	if got := new(big.Int).Neg(tokenDelta(ctx, userSource)).String(); tr.InputToken.AmountRaw != got || got != transferAmount(t, ctx, 3, 1) {
		t.Errorf("input %s, user source delta %s", tr.InputToken.AmountRaw, got)
	}
	if got := transferAmount(t, ctx, 3, 2); tr.OutputToken.AmountRaw != got {
		t.Errorf("output %s, transferred %s", tr.OutputToken.AmountRaw, got)
	}
	if tr.InputToken.Mint != ctx.Adapter.GetSplTokenMint(userSource) || tr.OutputToken.Mint != ctx.Adapter.GetSplTokenMint(userDest) {
		t.Errorf("mints %s -> %s", tr.InputToken.Mint, tr.OutputToken.Mint)
	}
	if len(tr.Pool) != 1 || tr.Pool[0] != accounts[1] {
		t.Errorf("pool %v, want amm %s", tr.Pool, accounts[1])
	}
	if tr.Fee != nil {
		t.Errorf("fee %+v: ray_log reports no fee", tr.Fee)
	}

	// The amounts come from the ray_log: a changed out_amount (the last u64
	// of SwapBaseIn) changes the trade
	logs := append([]string(nil), tx.Meta.LogMessages...)
	for i, l := range tx.Meta.LogMessages {
		if payload, ok := strings.CutPrefix(l, "Program log: ray_log: "); ok {
			data, err := base64.StdEncoding.DecodeString(payload)
			if err != nil || len(data) != 57 {
				t.Fatalf("unexpected ray_log %q", payload)
			}
			binary.LittleEndian.PutUint64(data[49:], binary.LittleEndian.Uint64(data[49:])+1)
			tx.Meta.LogMessages[i] = "Program log: ray_log: " + base64.StdEncoding.EncodeToString(data)
		}
	}
	changed := programTrades(newParseContext(tx, nil), constants.DEX_PROGRAMS.RAYDIUM_V4.ID, raydiumParser)
	want, _ := new(big.Int).SetString(tr.OutputToken.AmountRaw, 10)
	if len(changed) != 1 || changed[0].OutputToken.AmountRaw != want.Add(want, big.NewInt(1)).String() {
		t.Errorf("trade does not follow the ray_log: %+v", changed)
	}
	tx.Meta.LogMessages = logs

	// Without logs the transfers give the same trade
	tx.Meta.LogMessages = nil
	noLogs := programTrades(newParseContext(tx, nil), constants.DEX_PROGRAMS.RAYDIUM_V4.ID, raydiumParser)
	if len(noLogs) != 1 || noLogs[0].InputToken.AmountRaw != tr.InputToken.AmountRaw || noLogs[0].OutputToken.AmountRaw != tr.OutputToken.AmountRaw {
		t.Errorf("without logs: %+v", noLogs)
	}
}

// TestRaydiumCPMMSwapEventToken2022: the CPMM SwapEvent gives the net output
// of a Token-2022 mint with a transfer fee and the trade and creator fees
// (streamer item 8, D7). Truth: the next hop spends exactly the net amount
// (transfer at 3-5), and the fee values of the event (raydium_cp_swap IDL).
func TestRaydiumCPMMSwapEventToken2022(t *testing.T) {
	ctx := newParseContext(loadFixture(t, "NeF1UiWXKUbuswNNw14gJ2uup7yrV6KyXijQ7dsjLYanj9dnsBSeqGPDVZ3wP3NfhXQ84rJncRo5XwbrbSdWVWW"), nil)
	trades := programTrades(ctx, constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID, raydiumParser)
	if len(trades) != 1 {
		t.Fatalf("want 1 trade, got %+v", trades)
	}
	tr := trades[0]
	if gross, net := transferAmount(t, ctx, 3, 2), transferAmount(t, ctx, 3, 5); tr.OutputToken.AmountRaw != net || gross == net {
		t.Errorf("output %s, want the net amount %s (pool sent %s)", tr.OutputToken.AmountRaw, net, gross)
	}
	if tr.InputToken.AmountRaw != transferAmount(t, ctx, 3, 1) {
		t.Errorf("input %s, user sent %s", tr.InputToken.AmountRaw, transferAmount(t, ctx, 3, 1))
	}
	var ev []byte
	for _, l := range ctx.Utils.GetProgramDataLogs() {
		if constants.MatchDiscriminator(l.Data, constants.DISCRIMINATORS.RAYDIUM_CPMM.SWAP_EVENT) && l.ProgramId == constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID {
			ev = l.Data
		}
	}
	if len(ev) < 170 {
		t.Fatal("no SwapEvent with fees")
	}
	// SwapEvent: ..., output_transfer_fee @80, ..., trade_fee @153, creator_fee @161
	if tr.Fee == nil || tr.Fee.AmountRaw != u64s(ev, 153) || tr.Fee.Mint != tr.InputToken.Mint {
		t.Errorf("fee %+v, SwapEvent trade_fee %s", tr.Fee, u64s(ev, 153))
	}
	if feeOfType(tr, "creator") != u64s(ev, 161) {
		t.Errorf("creator fee %s, SwapEvent %s", feeOfType(tr, "creator"), u64s(ev, 161))
	}
	var outputTransferFee string
	for _, f := range tr.Fees {
		if f.Type == "transferFee" && f.Mint == tr.OutputToken.Mint {
			outputTransferFee = f.AmountRaw
		}
	}
	if outputTransferFee != u64s(ev, 80) {
		t.Errorf("output transfer fee %s, SwapEvent %s", outputTransferFee, u64s(ev, 80))
	}
}

// TestRaydiumCLMMSwapEvent: CLMM swaps take pool and amounts from the
// SwapEvent; the amounts equal the swap's transfers (streamer item 8).
func TestRaydiumCLMMSwapEvent(t *testing.T) {
	ctx := newParseContext(loadFixture(t, "2TyjCWrh3zqNmDg7NgdGAFqGaCbEkUHE8zrjzsRyqRVTXYZNKFFJgFEDce5mD4se3h2u6GyNJLXbeAW1ad8dsApq"), nil)
	trades := programTrades(ctx, constants.DEX_PROGRAMS.RAYDIUM_CL.ID, raydiumParser)
	if len(trades) != 1 {
		t.Fatalf("want 1 trade, got %+v", trades)
	}
	tr := trades[0]
	accounts := ctx.Adapter.GetInstructionAccounts(instructionAt(t, ctx, constants.DEX_PROGRAMS.RAYDIUM_CL.ID, 3, 6).Instruction)
	if len(tr.Pool) != 1 || tr.Pool[0] != accounts[2] {
		t.Errorf("pool %v, want pool_state %s", tr.Pool, accounts[2])
	}
	if tr.InputToken.AmountRaw != transferAmount(t, ctx, 3, 7) || tr.OutputToken.AmountRaw != transferAmount(t, ctx, 3, 8) {
		t.Errorf("amounts %s -> %s, transfers %s -> %s", tr.InputToken.AmountRaw, tr.OutputToken.AmountRaw, transferAmount(t, ctx, 3, 7), transferAmount(t, ctx, 3, 8))
	}
}

// TestRaydiumCLMMRouterSwap: swap_router_base_in swaps through several pools
// in one instruction; the trade takes the first hop's input, the last hop's
// output and every pool (amm-17). No real transaction is in the fixture
// pool, so this one is synthetic, built here from a real transaction with
// two CLMM swap_v2 hops (2-0 and 2-8): the first becomes swap_router_base_in
// and its log frame is extended to hold both SwapEvents.
func TestRaydiumCLMMRouterSwap(t *testing.T) {
	tx := loadFixture(t, "2YyxqLMgF9vQfx8vBNSsqRSLc33dG5wZLNFBvoZyrLAmbFBYeortegNiYfSpivnhuryjws3Tzi59FzRLuzjEKgG3")
	clmm := constants.DEX_PROGRAMS.RAYDIUM_CL.ID
	ctx := newParseContext(tx, nil)
	first := instructionAt(t, ctx, clmm, 2, 0)
	second := instructionAt(t, ctx, clmm, 2, 8)
	firstAccounts := ctx.Adapter.GetInstructionAccounts(first.Instruction)
	secondAccounts := ctx.Adapter.GetInstructionAccounts(second.Instruction)
	hop1In, hop2Out := firstAccounts[11], secondAccounts[12] // input_vault_mint / output_vault_mint

	// The logs of the first hop run up to its "success"; drop that line and
	// everything up to the second hop's invoke, so both SwapEvents are
	// written inside the first instruction's frame
	var logs []string
	skipping, invokes := false, 0
	for _, l := range tx.Meta.LogMessages {
		if strings.HasPrefix(l, "Program "+clmm+" invoke") {
			invokes++
			if invokes == 2 {
				skipping = false
				continue
			}
		}
		if invokes == 1 && l == "Program "+clmm+" success" {
			skipping = true
		}
		if !skipping {
			logs = append(logs, l)
		}
	}
	tx.Meta.LogMessages = logs
	ix := tx.Meta.InnerInstructions
	for s := range ix {
		if ix[s].Index != 2 {
			continue
		}
		m := ix[s].Instructions[0].(map[string]interface{})
		data := ctx.Adapter.GetInstructionData(first.Instruction)
		patched := append(append([]byte(nil), constants.DISCRIMINATORS.RAYDIUM_CL.SWAP.SWAP_ROUTER_BASE_IN...), data[8:24]...)
		m["data"] = base58.Encode(patched)
	}

	ctx = newParseContext(tx, nil)
	router := instructionAt(t, ctx, clmm, 2, 0)
	trades := raydium.NewRaydiumParser(ctx.Adapter, types.DexInfo{ProgramId: clmm, AMM: "RaydiumCL"}, ctx.TransferActions, []types.ClassifiedInstruction{router}).ProcessTrades()
	if len(trades) != 1 {
		t.Fatalf("want 1 trade, got %+v", trades)
	}
	tr := trades[0]
	if len(tr.Pool) != 2 || tr.Pool[0] != firstAccounts[2] || tr.Pool[1] != secondAccounts[2] {
		t.Errorf("pools %v, want %s and %s", tr.Pool, firstAccounts[2], secondAccounts[2])
	}
	if tr.InputToken.Mint != hop1In || tr.OutputToken.Mint != hop2Out {
		t.Errorf("mints %s -> %s, want %s -> %s", tr.InputToken.Mint, tr.OutputToken.Mint, hop1In, hop2Out)
	}
	if tr.InputToken.AmountRaw != transferAmount(t, ctx, 2, 1) || tr.OutputToken.AmountRaw != transferAmount(t, ctx, 2, 10) {
		t.Errorf("amounts %s -> %s, want first hop input %s and last hop output %s",
			tr.InputToken.AmountRaw, tr.OutputToken.AmountRaw, transferAmount(t, ctx, 2, 1), transferAmount(t, ctx, 2, 10))
	}
}

// TestMeteoraDammV1SwapEvent: the DAMM v1 Swap log gives the output the pool
// paid; the transfers of the group also hold the caller's later transfers,
// which doubled the output (streamer item 8). Truth: the pool token vault's
// balance change.
func TestMeteoraDammV1SwapEvent(t *testing.T) {
	ctx := newParseContext(loadFixture(t, "3d6W6W6GXDKHVYnJdWA2kQar2dxiYD9iN6ySFWimVQXq7cR1pNHN4WXVcJkgf5tmEQR7Q9NKHPNEWGdiHGwUh8Cd"), nil)
	trades := programTrades(ctx, constants.DEX_PROGRAMS.METEORA_DAMM.ID, meteoraParser)
	if len(trades) != 1 {
		t.Fatalf("want 1 trade, got %+v", trades)
	}
	accounts := ctx.Adapter.GetInstructionAccounts(instructionAt(t, ctx, constants.DEX_PROGRAMS.METEORA_DAMM.ID, 2, 12).Instruction)
	// swap accounts: a_token_vault 5, b_token_vault 6 (vault program token accounts)
	var paid string
	for _, vault := range []string{accounts[5], accounts[6]} {
		if d := tokenDelta(ctx, vault); d.Sign() < 0 {
			paid = new(big.Int).Neg(d).String()
		}
	}
	if trades[0].OutputToken.AmountRaw != paid {
		t.Errorf("output %s, pool vault paid %s", trades[0].OutputToken.AmountRaw, paid)
	}
	if trades[0].Fee == nil || trades[0].Fee.Type != "lp" {
		t.Errorf("fee %+v, want the Swap trade_fee", trades[0].Fee)
	}
}

// TestMeteoraDammV2SwapEvents: DAMM v2 swaps take amounts from EvtSwap2 (or
// the legacy EvtSwap): the Token-2022 input including its transfer fee
// (4vPzV2), the output without a referral fee paid to someone else (5tn4UE),
// and with one paid to the user's own account (5GXaLd) (streamer item 5).
func TestMeteoraDammV2SwapEvents(t *testing.T) {
	program := constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID

	// Token-2022 input with a transfer fee: the user sent more than the pool received
	ctx := newParseContext(loadFixture(t, "4vPzV2JPbDZghNgE6pYQhcjTFrNQt1V2A23bRXCpmYRQdjmKTGYgrfiLWXdXrdZpBu1CrWWLQoJxbL7GnoQJLRhv"), nil)
	trades := programTrades(ctx, program, meteoraParser)
	accounts := ctx.Adapter.GetInstructionAccounts(instructionAt(t, ctx, program, 4, 6).Instruction)
	if len(trades) != 1 {
		t.Fatalf("want 1 trade, got %+v", trades)
	}
	sent, received := transferAmount(t, ctx, 4, 7), tokenDelta(ctx, accounts[4]).String() // token_a_vault
	if trades[0].InputToken.AmountRaw != sent || sent == received {
		t.Errorf("input %s, user sent %s (pool received %s)", trades[0].InputToken.AmountRaw, sent, received)
	}

	// legacy EvtSwap (and EvtSwap2) with a referral fee to another account
	ctx = newParseContext(loadFixture(t, "5tn4UECvbP7n6AKNmzZh646k2P5uPsbtoDHXPbH2U8YF5wWnNwiWcpY5hcTs8xW9tZbabroFJPHa6227qiy89Vcu"), nil)
	trades = programTrades(ctx, program, meteoraParser)
	accounts = ctx.Adapter.GetInstructionAccounts(instructionAt(t, ctx, program, 3, 0).Instruction)
	poolPaid := new(big.Int).Neg(tokenDelta(ctx, accounts[5])) // token_b_vault (WSOL)
	referral := tokenDelta(ctx, accounts[11])
	want := new(big.Int).Sub(poolPaid, referral).String()
	if len(trades) != 1 || trades[0].OutputToken.AmountRaw != want || referral.Sign() <= 0 {
		t.Errorf("output %+v, want pool paid %s minus referral %s", trades, poolPaid, referral)
	}

	// self-referral: the referral account is the user's output account
	ctx = newParseContext(loadFixture(t, "5GXaLd1g1tHYa7VVBmpm2wo2HSob1H6X8gtVDtw5FQmh88Yf74787v973eTSRCiqQo93v1KWbh79n4WyHKQPFzVv"), nil)
	trades = programTrades(ctx, program, meteoraParser)
	accounts = ctx.Adapter.GetInstructionAccounts(instructionAt(t, ctx, program, 4, -1).Instruction)
	if accounts[11] != accounts[3] {
		t.Fatal("fixture is not a self-referral")
	}
	if got := tokenDelta(ctx, accounts[3]).String(); len(trades) != 1 || trades[0].OutputToken.AmountRaw != got {
		t.Errorf("output %+v, user received %s", trades, got)
	}
}

// TestMeteoraDLMMSwap2Event: a DLMM swap that emits Swap and Swap2Evt takes
// the fees from Swap2Evt (mm_fee as Fee, protocol fee in Fees); the Swap
// event's fee is mm_fee plus protocol_fee (streamer item 15).
func TestMeteoraDLMMSwap2Event(t *testing.T) {
	ctx := newParseContext(loadFixture(t, "4bo3keYw6cNyKFBPWkyVKBRxx5HR9pqUX7oUCKfaCDHyEHUtcvWf83JMyw8aCw95miYpCibmvZ47yzEU2QgL1SpC"), nil)
	program := constants.DEX_PROGRAMS.METEORA.ID
	trades := programTrades(ctx, program, meteoraParser)
	if len(trades) != 1 {
		t.Fatalf("want 1 trade, got %+v", trades)
	}
	tr := trades[0]
	swapEv := ctx.Adapter.GetInstructionData(instructionAt(t, ctx, program, 1, 2).Instruction)
	swap2 := ctx.Adapter.GetInstructionData(instructionAt(t, ctx, program, 1, 3).Instruction)
	if !constants.MatchDiscriminator(swapEv, constants.DISCRIMINATORS.METEORA_DLMM.EVENTS["swap"]) ||
		!constants.MatchDiscriminator(swap2, constants.DISCRIMINATORS.METEORA_DLMM.EVENTS["swap2Evt"]) {
		t.Fatal("1-2/1-3 are not the Swap/Swap2Evt events")
	}
	// Swap2Evt: prefix 16, lb_pair, from, bins, swap_for_y @88, fee_bps u128,
	// amount_in @105, amount_left @113, amount_out @121, mm_fee @129, protocol_fee @137
	if tr.InputToken.AmountRaw != u64s(swap2, 105) || tr.OutputToken.AmountRaw != u64s(swap2, 121) {
		t.Errorf("amounts %s -> %s, Swap2Evt %s -> %s", tr.InputToken.AmountRaw, tr.OutputToken.AmountRaw, u64s(swap2, 105), u64s(swap2, 121))
	}
	if tr.InputToken.AmountRaw != transferAmount(t, ctx, 1, 0) || tr.OutputToken.AmountRaw != transferAmount(t, ctx, 1, 1) {
		t.Errorf("amounts differ from the transfers")
	}
	if tr.Fee == nil || tr.Fee.AmountRaw != u64s(swap2, 129) || feeOfType(tr, "protocol") != u64s(swap2, 137) {
		t.Errorf("fees %+v %+v, Swap2Evt mm_fee %s protocol %s", tr.Fee, tr.Fees, u64s(swap2, 129), u64s(swap2, 137))
	}
	// Swap: fee @105, protocol_fee @113
	fee, _ := new(big.Int).SetString(u64s(swapEv, 105), 10)
	protocol, _ := new(big.Int).SetString(u64s(swapEv, 113), 10)
	if new(big.Int).Sub(fee, protocol).String() != u64s(swap2, 129) {
		t.Errorf("Swap fee %s - protocol %s != Swap2Evt mm_fee %s", fee, protocol, u64s(swap2, 129))
	}
}

// TestAmmTradeLegTransfers: each leg of an AMM trade carries the token
// accounts, authority and balances of the swap instruction's own transfer.
// Before, a leg took the first transfer of the whole transaction with its
// mint and amount: in a route the input got the previous hop's output (same
// amount), a Token-2022 output with a transfer fee (net amount, which no
// transfer carries) was left without them, or got the next hop's input
// (NeF1UiWX). Truth: the swap instruction's accounts per IDL and the owner
// in the token balances. Parser level: inside a Jupiter route ParseAll
// reports the route's trade instead of the hops.
func TestAmmTradeLegTransfers(t *testing.T) {
	cpmm := constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID
	dammV2 := constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID
	whirlpool := constants.DEX_PROGRAMS.ORCA.ID
	dlmm := constants.DEX_PROGRAMS.METEORA.ID
	rayAMM := constants.DEX_PROGRAMS.RAYDIUM_AMM.ID
	// CPMM swap_base_input: payer 0, authority 1, input_token_account 4,
	// output_token_account 5, input_vault 6, output_vault 7
	cpmmIn, cpmmOut := [3]int{4, 6, 0}, [3]int{7, 5, 1}
	// whirlpool swap_v2 b to a: token_authority 3, whirlpool 4,
	// token_owner_account_a 7, token_vault_a 8, token_owner_account_b 9,
	// token_vault_b 10
	orcaV2In, orcaV2Out := [3]int{9, 10, 3}, [3]int{8, 7, 4}
	type newParser = func(*adapter.TransactionAdapter, types.DexInfo, map[string][]types.TransferData, []types.ClassifiedInstruction) interface{ ProcessTrades() []types.TradeInfo }
	cases := []struct {
		name, prefix, program string
		parser                newParser
		outer, inner          int
		tradeIdx              string
		// source, destination and authority of each leg as instruction
		// account indices
		in, out [3]int
	}{
		// event trades; the outputs have a Token-2022 transfer fee
		{"cpmm", "34sGGDUK4A1x", cpmm, raydiumParser, 5, 7, "5-8", cpmmIn, cpmmOut},
		{"cpmm 4gKb", "4gKbbWmpYHRW", cpmm, raydiumParser, 5, 18, "5-19", cpmmIn, cpmmOut},
		{"cpmm 51nj", "51nj5GtAmDC2", cpmm, raydiumParser, 1, 5, "1-6", cpmmIn, cpmmOut},
		{"cpmm 5bLd", "5bLdxNz8YJyt", cpmm, raydiumParser, 5, 7, "5-8", cpmmIn, cpmmOut},
		{"cpmm 5qJs", "5qJs7ws4UY3q", cpmm, raydiumParser, 5, 7, "5-8", cpmmIn, cpmmOut},
		// the net output equals the next hop's (DLMM) input
		{"cpmm NeF1", "NeF1UiWXKUbu", cpmm, raydiumParser, 3, 0, "3-1", cpmmIn, cpmmOut},
		// DAMM v2 swap, B to A: pool_authority 0, input_token_account 2,
		// output_token_account 3, token_a_vault 4, token_b_vault 5, payer 8
		{"damm v2", "5GXaLd1g1tHY", dammV2, meteoraParser, 4, -1, "4-0", [3]int{2, 5, 8}, [3]int{4, 3, 0}},
		{"orca", "Dr6ZkVfaHmFv", whirlpool, orcaParser, 4, 7, "4-8", orcaV2In, orcaV2Out},
		{"orca outer", "2nTCUTK4dYky", whirlpool, orcaParser, 5, -1, "5-0", orcaV2In, orcaV2Out},
		// DLMM swap, X to Y: lb_pair 0, reserve_x 2, reserve_y 3,
		// user_token_in 4, user_token_out 5, user 10
		{"dlmm", "2K23xSbLP1SG", dlmm, meteoraParser, 2, 4, "2-5", [3]int{4, 2, 10}, [3]int{3, 5, 0}},
		// trades from transfers (no ray_log, no Traded event).
		// Raydium AMM swap: authority 2, pool vaults 4/5, user source 15,
		// user destination 16, user owner 17
		{"raydium amm", "33VnDBtrFawB", rayAMM, raydiumParser, 4, 4, "4-5", [3]int{15, 5, 17}, [3]int{4, 16, 2}},
		// whirlpool swap b to a: token_authority 1, whirlpool 2,
		// token_owner_account_a 3, token_vault_a 4, token_owner_account_b 5,
		// token_vault_b 6
		{"orca v1", "4MSVpVBwxnYT", whirlpool, orcaParser, 2, 4, "2-5", [3]int{5, 6, 1}, [3]int{4, 3, 2}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			tx := loadFixture(t, fixtureSig(t, c.prefix))
			ctx := newParseContext(tx, nil)
			accounts := ctx.Adapter.GetInstructionAccounts(instructionAt(t, ctx, c.program, c.outer, c.inner).Instruction)
			trade := tradeAt(t, programTrades(ctx, c.program, c.parser), c.tradeIdx)
			for _, leg := range []struct {
				name  string
				token types.TokenInfo
				want  [3]int
			}{{"input", trade.InputToken, c.in}, {"output", trade.OutputToken, c.out}} {
				src, dst, auth := accounts[leg.want[0]], accounts[leg.want[1]], accounts[leg.want[2]]
				if leg.token.Source != src || leg.token.Destination != dst || leg.token.Authority != auth {
					t.Errorf("%s source %q destination %q authority %q, want %s %s %s", leg.name,
						leg.token.Source, leg.token.Destination, leg.token.Authority, src, dst, auth)
				}
			}
			out := trade.OutputToken
			if owner := ctx.Adapter.GetTokenAccountOwner(accounts[c.out[1]]); owner == "" || out.DestinationOwner != owner {
				t.Errorf("output destinationOwner %q, token balances say %q", out.DestinationOwner, owner)
			}
			if out.SourceBalance == nil || out.SourcePreBalance == nil {
				t.Errorf("output source balances missing")
			}
		})
	}
}

// fixtureSig returns the full signature of the stored fixture that starts
// with prefix
func fixtureSig(t *testing.T, prefix string) string {
	t.Helper()
	var found string
	for _, sig := range fixtureSignatures(t, "json") {
		if strings.HasPrefix(sig, prefix) {
			if found != "" {
				t.Fatalf("prefix %s is ambiguous", prefix)
			}
			found = sig
		}
	}
	if found == "" {
		t.Fatalf("no fixture %s*", prefix)
	}
	return found
}
