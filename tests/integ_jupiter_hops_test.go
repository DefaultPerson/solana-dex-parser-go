package tests

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// integHop returns the trade at idx of a ParseAll result
func integHop(t *testing.T, result *types.ParseResult, idx string) *types.TradeInfo {
	t.Helper()
	for i := range result.Trades {
		if result.Trades[i].Idx == idx {
			return &result.Trades[i]
		}
	}
	t.Fatalf("no trade at %s", idx)
	return nil
}

// Jupiter hops take Pool, Type and the venue's fees from the trade the
// venue's own parser reports for the hop's instruction (amm-14, jupiter G3,
// venue fees); amounts stay the route's; the trade count does not change.
//
// 41NE3vcJ... 5-4 is a LaunchLab buy_exact_in quoted in a token that is
// neither SOL nor a stablecoin (IDL accounts: 4 pool_state, 10
// quote_token_mint). GetTradeType called it SELL; the input is the pool's
// quote mint, so it is a BUY.
func TestIntegJupiterHopFromVenue(t *testing.T) {
	const sig = "41NE3vcJ2aXixCvWvPV5fb6d1PxDfHT7GNGS5nFJqKMnJcy5g6iGkqkxt8nUWYaK9HLS8fuz6s5d4MhsmTC6AjEL"
	tx := loadFixture(t, sig)
	result := dexparser.NewDexParser().ParseAll(tx, nil)
	if len(result.Trades) != 2 {
		t.Errorf("%d trades, want the 2 hops", len(result.Trades))
	}
	ctx := newParseContext(tx, nil)
	ix := ctx.Adapter.GetInnerInstruction(5, 4)
	accounts := ctx.Adapter.GetInstructionAccounts(ix)
	if ctx.Adapter.GetInstructionProgramId(ix) != constants.DEX_PROGRAMS.RAYDIUM_LCP.ID ||
		!constants.MatchDiscriminator(ctx.Adapter.GetInstructionData(ix), constants.DISCRIMINATORS.RAYDIUM_LCP.BUY_EXACT_IN) || len(accounts) < 11 {
		t.Fatal("5-4 is not a LaunchLab buy_exact_in")
	}
	hop := integHop(t, result, "5-4")
	if hop.InputToken.Mint != accounts[10] {
		t.Fatalf("hop input %s, want the pool's quote mint %s", hop.InputToken.Mint, accounts[10])
	}
	if hop.Type != types.TradeTypeBuy {
		t.Errorf("hop type %s, want BUY (input is the quote mint)", hop.Type)
	}
	if len(hop.Pool) != 1 || hop.Pool[0] != accounts[4] {
		t.Errorf("hop pool %v, want pool_state %s", hop.Pool, accounts[4])
	}
	var protocol, platform bool
	for _, f := range hop.Fees {
		protocol = protocol || (f.Type == "protocol" && f.Dex == constants.DEX_PROGRAMS.RAYDIUM_LCP.Name && f.Mint == accounts[10])
		platform = platform || (f.Type == "platform" && f.Dex == constants.DEX_PROGRAMS.RAYDIUM_LCP.Name && f.Mint == accounts[10])
	}
	if !protocol || !platform {
		t.Errorf("hop fees %+v, want LaunchLab protocol and platform fees", hop.Fees)
	}
	// Amounts are the route event's: what the user's accounts moved
	if hop.InputToken.AmountRaw != "3266000000" || hop.OutputToken.AmountRaw != "356622750432" {
		t.Errorf("hop %s -> %s, want the event's 3266000000 -> 356622750432", hop.InputToken.AmountRaw, hop.OutputToken.AmountRaw)
	}
}

// NeF1UiWX... 3-0 is a Raydium CPMM swap_base_input (IDL accounts: 3
// pool_state) whose SwapEvent reports a trade fee of 9396954 and a creator
// fee of 19502; 4YEiZ5wX... 2-3 a DLMM swap with an lp and a protocol fee.
func TestIntegJupiterHopVenueFees(t *testing.T) {
	cases := []struct {
		sig, idx  string
		program   constants.DexProgram
		poolIndex int
		trades    int
		feeType   string
		feeAmount string
		otherType string
		otherFee  string
	}{
		{"NeF1UiWXKUbuswNNw14gJ2uup7yrV6KyXijQ7dsjLYanj9dnsBSeqGPDVZ3wP3NfhXQ84rJncRo5XwbrbSdWVWW", "3-0", constants.DEX_PROGRAMS.RAYDIUM_CPMM, 3, 3, "trade", "9396954", "creator", "19502"},
		{"4YEiZ5wX7cJj4XkDseVueSMwKgfMRA7HZ6NAvrQon7x6mJp4PSV43EcB1GzPjdibu8qrupmNt2AiC4QENtxYrcgF", "2-3", constants.DEX_PROGRAMS.METEORA, 0, 2, "lp", "115423658", "protocol", "12824850"},
	}
	for _, tc := range cases {
		t.Run(tc.sig[:8], func(t *testing.T) {
			tx := loadFixture(t, tc.sig)
			result := dexparser.NewDexParser().ParseAll(tx, nil)
			if len(result.Trades) != tc.trades {
				t.Errorf("%d trades, want %d", len(result.Trades), tc.trades)
			}
			ctx := newParseContext(tx, nil)
			var outer, inner int
			if _, err := fmt.Sscanf(tc.idx, "%d-%d", &outer, &inner); err != nil {
				t.Fatal(err)
			}
			ix := ctx.Adapter.GetInnerInstruction(outer, inner)
			accounts := ctx.Adapter.GetInstructionAccounts(ix)
			if ctx.Adapter.GetInstructionProgramId(ix) != tc.program.ID || len(accounts) <= tc.poolIndex {
				t.Fatalf("%s is not a %s instruction", tc.idx, tc.program.Name)
			}
			hop := integHop(t, result, tc.idx)
			if hop.ProgramId != constants.DEX_PROGRAMS.JUPITER.ID || hop.AMM != tc.program.Name {
				t.Errorf("hop %s %s, want a Jupiter hop through %s", hop.ProgramId, hop.AMM, tc.program.Name)
			}
			if len(hop.Pool) != 1 || hop.Pool[0] != accounts[tc.poolIndex] {
				t.Errorf("hop pool %v, want %s", hop.Pool, accounts[tc.poolIndex])
			}
			if hop.Fee == nil || hop.Fee.Type != tc.feeType || hop.Fee.AmountRaw != tc.feeAmount || hop.Fee.Dex != tc.program.Name {
				t.Errorf("hop fee %+v, want %s %s", hop.Fee, tc.feeType, tc.feeAmount)
			}
			found := false
			for _, f := range hop.Fees {
				found = found || (f.Type == tc.otherType && f.AmountRaw == tc.otherFee && f.Dex == tc.program.Name)
			}
			if !found {
				t.Errorf("hop fees %+v, want %s %s", hop.Fees, tc.otherType, tc.otherFee)
			}
		})
	}
}

// When the venue's amounts disagree with the route's event, the event's are
// kept and ParseResult.Warnings says so. 4S6bFmLV... 2-6: the PumpSwap buy at
// 2-0 took 7558855408 + 3771884 + 3771884 = 7566399176 WSOL from the user's
// account (transfers 2-2..2-4, what PumpSwap's event reports), the Jupiter
// SwapsEvent says 7566467275.
func TestIntegJupiterHopAmountWarning(t *testing.T) {
	tx := loadFixture(t, "4S6bFmLVsNumkv4m8kwzSQvX4tsDQLgTsBaTXWknZAvGqj2wW9cEYH4A8NJiU5EwzE2u5MNthSwQ3M9yG1QWtT8h")
	result := dexparser.NewDexParser().ParseAll(tx, nil)
	hop := integHop(t, result, "2-6")
	if hop.InputToken.AmountRaw != "7566467275" {
		t.Errorf("hop input %s, want the event's 7566467275", hop.InputToken.AmountRaw)
	}
	sent := new(big.Int)
	for _, r := range rawInnerTransfers(t, tx, 2) {
		if r.inner >= 2 && r.inner <= 4 {
			v, _ := new(big.Int).SetString(r.amount, 10)
			sent.Add(sent, v)
		}
	}
	if sent.String() != "7566399176" {
		t.Fatalf("fixture: PumpSwap took %s", sent)
	}
	var warning string
	for _, w := range result.Warnings {
		if strings.HasPrefix(w, "hop 2-6 ") {
			warning = w
		}
	}
	if !strings.Contains(warning, "7566399176") || !strings.Contains(warning, "7566467275") {
		t.Errorf("warnings %q, want one for hop 2-6 naming both inputs", result.Warnings)
	}
}
