package tests

import (
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Shred trade types when a mint is unknown. Without meta a v1 Jupiter route
// names only its source token account, so the input mint is unknown, and the
// old rule typed the trade from the one known side: every route ending in
// SOL or a stablecoin became SELL, circular USDC -> USDC arbitrage included.
// G4 of the shred verifier (shred-11).

// executedTypes maps "programId idx action" to the trade type of the
// executed transaction (meta, failed transactions included)
func executedTypes(t *testing.T, tx *adapter.SolanaTransaction) map[string]types.TradeType {
	t.Helper()
	out := map[string]types.TradeType{}
	for _, ins := range parseShred(t, tx, &types.ParseConfig{IncludeFailedTxs: true}).ParsedInstructions {
		if ins.Trade != nil {
			out[ins.ProgramID+" "+ins.Idx+" "+ins.Action] = ins.Trade.Type
		}
	}
	return out
}

// TestShredOneKnownMintIsSwap: aggregator routes with one unknown mint are
// SWAP before execution. Truth: the route's mints from the executed
// transaction's instruction accounts and token balances.
func TestShredOneKnownMintIsSwap(t *testing.T) {
	const usd1Mint = "USD1ttGY1N17NEEHLmELoaybftRBUSErhqYiQzvEmuB"
	for _, c := range []struct {
		sig   string
		outer int
		// srcAccount: the user's source token account; inMint/outMint:
		// instruction account indexes of the mints, -1 when not an account
		srcAccount, inMint, outMint int
		wantIn, wantOut             string
	}{
		// route: 2 user_source_token_account, 5 destination_mint
		{"31NSV7BhhtfFDV8u6ocVF3UACvnTLmmb9PSGvmjio5RzraKnNibKH2tHNHUsZPbGMgH3NupGqotBRjkfFJboGDah", 2, 2, -1, 5, usdcMint, usdcMint},
		// shared_accounts_route_with_token_ledger: 3 source_token_account,
		// 7 source_mint, 8 destination_mint
		{sigJupSharedLedger, 9, 3, 7, 8, usdcMint, usd1Mint},
		{sigJupSharedLedger, 10, 3, 7, 8, usd1Mint, solMint},
	} {
		tx := loadFixture(t, c.sig)
		var ix rawIx
		for _, x := range fixtureIxs(t, tx) {
			if x.programId == constants.DEX_PROGRAMS.JUPITER.ID && x.outer == c.outer && x.inner < 0 {
				ix = x
			}
		}
		if ix.accounts == nil {
			t.Fatalf("%s: no Jupiter instruction at %d", c.sig[:8], c.outer)
		}
		in := tokenBalanceMint(tx, ix.accounts[c.srcAccount])
		if c.inMint >= 0 && (in == "" || in == ix.accounts[c.inMint]) {
			in = ix.accounts[c.inMint]
		}
		if in != c.wantIn || ix.accounts[c.outMint] != c.wantOut {
			t.Fatalf("%s %d: fixture route %s -> %s, want %s -> %s", c.sig[:8], c.outer, in, ix.accounts[c.outMint], c.wantIn, c.wantOut)
		}
		want := utils.GetTradeType(c.wantIn, c.wantOut)
		if c.wantIn == c.wantOut {
			want = types.TradeTypeSwap
		}
		idx := utils.FormatIdx(c.outer, -1)

		_, full := jupTrade(t, parseShred(t, tx, &types.ParseConfig{IncludeFailedTxs: true}), idx)
		if full.InputToken.Mint != c.wantIn || full.OutputToken.Mint != c.wantOut || full.Type != want {
			t.Errorf("%s %s executed: %s %s -> %s, want %s %s -> %s", c.sig[:8], idx, full.Type, full.InputToken.Mint, full.OutputToken.Mint, want, c.wantIn, c.wantOut)
		}

		_, pre := jupTrade(t, parseShred(t, preExec(t, tx), nil), idx)
		if pre.InputToken.Mint != "" && pre.OutputToken.Mint != "" {
			t.Fatalf("%s %s pre-execution: both mints known, the case is not exercised", c.sig[:8], idx)
		}
		if pre.Type != types.TradeTypeSwap {
			t.Errorf("%s %s pre-execution: %s (%q -> %q), want SWAP (executed: %s)", c.sig[:8], idx, pre.Type, pre.InputToken.Mint, pre.OutputToken.Mint, want)
		}
	}
}

// TestShredPreExecTradeTypes: over every fixture, a pre-execution Jupiter or
// DFlow trade with an unknown mint is SWAP, and every pre-execution BUY or
// SELL equals the executed transaction's type for that instruction.
func TestShredPreExecTradeTypes(t *testing.T) {
	aggregators := map[string]bool{constants.DEX_PROGRAMS.JUPITER.ID: true, constants.DEX_PROGRAMS.DFLOW.ID: true}
	var oneSided, decided int
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		executed := executedTypes(t, tx)
		for _, ins := range parseShred(t, preExec(t, tx), nil).ParsedInstructions {
			tr := ins.Trade
			if tr == nil {
				continue
			}
			if aggregators[ins.ProgramID] && (tr.InputToken.Mint == "" || tr.OutputToken.Mint == "") {
				oneSided++
				if tr.Type != types.TradeTypeSwap {
					t.Errorf("%s %s %s: %s with mints %q -> %q, want SWAP", sig[:8], ins.ProgramName, ins.Idx, tr.Type, tr.InputToken.Mint, tr.OutputToken.Mint)
				}
			}
			if tr.Type == types.TradeTypeSwap {
				continue
			}
			decided++
			if got := executed[ins.ProgramID+" "+ins.Idx+" "+ins.Action]; got != tr.Type {
				t.Errorf("%s %s %s %s: pre-execution %s, executed %s", sig[:8], ins.ProgramName, ins.Idx, ins.Action, tr.Type, got)
			}
		}
	}
	t.Logf("aggregator trades with an unknown mint: %d, pre-execution BUY/SELL: %d", oneSided, decided)
	if oneSided == 0 || decided == 0 {
		t.Fatalf("fixtures exercise nothing: %d one-sided, %d decided", oneSided, decided)
	}
}

// TestShredRaydiumV4OneKnownMint: in a single pool a known WSOL side decides
// the direction, a known stablecoin does not (its pool may pair it with SOL).
// 5kaAWK5X is real: before execution only its WSOL output is known. No
// pre-execution swap with only a stablecoin known was found, so the second
// case is the real USDC -> SOL v2 swap of 4gfccDph (inner 5-9, SOL/USDC
// pool) with what names the mint of its WSOL destination and WSOL vault
// removed (their token balances and the inner instructions of the ATA
// create that initializes the destination), leaving only the USDC side known.
func TestShredRaydiumV4OneKnownMint(t *testing.T) {
	d := constants.DISCRIMINATORS.RAYDIUM

	real := loadFixture(t, sigRaySwap18)
	ix := findIx(t, real, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, d.SWAP)
	// The destination (account 16) is created in the transaction: an outer
	// InitializeAccount (tag 1: account, mint) names its mint
	var dstMint string
	for _, x := range fixtureIxs(t, real) {
		if x.inner < 0 && x.programId == constants.TOKEN_PROGRAM_ID && len(x.data) > 0 && x.data[0] == 1 && len(x.accounts) > 1 && x.accounts[0] == ix.accounts[16] {
			dstMint = x.accounts[1]
		}
	}
	if dstMint != solMint || tokenBalanceMint(real, ix.accounts[15]) == solMint {
		t.Fatalf("5kaAWK5X: not a token -> WSOL swap (destination mint %q)", dstMint)
	}
	idx := utils.FormatIdx(ix.outer, -1)
	pre := oneTypedAt(t, parseShred(t, preExec(t, real), nil), constants.DEX_PROGRAMS.RAYDIUM_V4.ID, idx).Trade
	if pre.InputToken.Mint != "" || pre.OutputToken.Mint != solMint || pre.Type != types.TradeTypeSell {
		t.Errorf("5kaAWK5X pre-execution: %s %q -> %q, want SELL \"\" -> WSOL", pre.Type, pre.InputToken.Mint, pre.OutputToken.Mint)
	}

	tx := loadFixture(t, sigRaySwapV2)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_V4.ID, d.SWAP_V2)
	// v2 accounts: 3 coin vault, 4 pc vault, 5 source, 6 destination
	coin, pc := tokenBalanceMint(tx, ix.accounts[3]), tokenBalanceMint(tx, ix.accounts[4])
	if tokenBalanceMint(tx, ix.accounts[5]) != usdcMint || !(coin == solMint && pc == usdcMint) {
		t.Fatalf("4gfccDph: not a USDC -> SOL swap in a SOL/USDC pool (source %s, vaults %s/%s)", tokenBalanceMint(tx, ix.accounts[5]), coin, pc)
	}
	idx = utils.FormatIdx(ix.outer, ix.inner)
	if full := oneTypedAt(t, parseShred(t, tx, nil), constants.DEX_PROGRAMS.RAYDIUM_V4.ID, idx).Trade; full.Type != types.TradeTypeSell {
		t.Fatalf("4gfccDph executed: %s, want SELL (USDC -> SOL)", full.Type)
	}

	syn := cloneTx(t, tx)
	keys := rawAccountKeys(syn)
	drop := map[string]bool{ix.accounts[3]: true, ix.accounts[6]: true}
	filter := func(list []adapter.TokenBalance) []adapter.TokenBalance {
		var kept []adapter.TokenBalance
		for _, b := range list {
			if b.AccountIndex >= len(keys) || !drop[keys[b.AccountIndex]] {
				kept = append(kept, b)
			}
		}
		return kept
	}
	syn.Meta.PreTokenBalances, syn.Meta.PostTokenBalances = filter(syn.Meta.PreTokenBalances), filter(syn.Meta.PostTokenBalances)
	create := -1
	for _, x := range fixtureIxs(t, tx) {
		if x.inner < 0 && x.programId == constants.ASSOCIATED_TOKEN_PROGRAM_ID && len(x.accounts) > 3 && x.accounts[1] == ix.accounts[6] {
			create = x.outer
		}
	}
	if create < 0 {
		t.Fatalf("4gfccDph: no ATA create for the destination")
	}
	var inner []adapter.InnerInstructionSet
	for _, set := range syn.Meta.InnerInstructions {
		if set.Index != create {
			inner = append(inner, set)
		}
	}
	syn.Meta.InnerInstructions = inner
	tr := oneTypedAt(t, parseShred(t, syn, nil), constants.DEX_PROGRAMS.RAYDIUM_V4.ID, idx).Trade
	if tr.InputToken.Mint != usdcMint || tr.OutputToken.Mint != "" {
		t.Fatalf("synthetic: mints %q -> %q, want USDC -> \"\"", tr.InputToken.Mint, tr.OutputToken.Mint)
	}
	if tr.Type != types.TradeTypeSwap {
		t.Errorf("synthetic USDC -> unknown: %s, want SWAP (executed: SELL)", tr.Type)
	}
}
