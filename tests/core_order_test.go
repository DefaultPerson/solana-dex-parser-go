package tests

import (
	"math/big"
	"reflect"
	"strings"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Regression tests for program order, route selection, trade order and
// deterministic output (DESIGN D1, D2, D5).

const (
	// Jupiter route_v2: SOL -> 3ZLekZ (Orca-like "ojh19o"), 3ZLekZ -> F5zMFh (LaunchLab)
	sigRouteV2A = "41NE3vcJ2aXixCvWvPV5fb6d1PxDfHT7GNGS5nFJqKMnJcy5g6iGkqkxt8nUWYaK9HLS8fuz6s5d4MhsmTC6AjEL"
	sigRouteV2B = "4yQCvQwEo1skrvjvEqYQ3rj26uo57ZCkTHTcBXZGR7igBDeuW9YWNKjgruCmZJvAxMegUj3pMBR5XKRm7HxrR3CB"
	launchMint  = "F5zMFhfDtvRPhj71xPgnCWWResmWWuNSmS2AKgyz9KDk"
	// Jupiter -> 3 Meteora DAMM v1 hops (vault CPIs) -> 1Dex
	sigJupDammVault = "55VfB9g52vYyi6eGYmiCUe2sLpbBsH16DqN6rXpA9UKkXg6oo4y1ohZa7Dc1QJFfrkiLD5b8b7ZVBhA9p56xVzMG"
	// plain Meteora DAMM v1 swap (vault CPIs)
	sigDammV1 = "4uuw76SPksFw6PvxLFkG9jRyReV1F4EyPYNc3DdSECip8tM22ewqGWJUaRZ1SJEZpuLJz1qPTEPb2es8Zuegng9Z"
	// Jupiter 3-hop route (many transfers)
	sigJup3Hop = "NeF1UiWXKUbuswNNw14gJ2uup7yrV6KyXijQ7dsjLYanj9dnsBSeqGPDVZ3wP3NfhXQ84rJncRo5XwbrbSdWVWW"
)

// TestCoreProgramsInFirstAppearanceOrder: program IDs used to be sorted
// alphabetically. D1: outer instructions in order, then inner ones.
func TestCoreProgramsInFirstAppearanceOrder(t *testing.T) {
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		ctx := newParseContext(tx, nil)
		// independent computation from the raw message
		keys := rawAccountKeys(tx)
		var want []string
		seen := map[string]bool{}
		add := func(ix interface{}) {
			m, ok := ix.(map[string]interface{})
			if !ok {
				return
			}
			i := jsonInt(m["programIdIndex"])
			if i < 0 || i >= len(keys) {
				return
			}
			id := keys[i]
			if seen[id] || inList(constants.SYSTEM_PROGRAMS, id) || inList(constants.SKIP_PROGRAM_IDS, id) {
				return
			}
			seen[id] = true
			want = append(want, id)
		}
		for _, ix := range tx.Transaction.Message.Instructions {
			add(ix)
		}
		for _, set := range tx.Meta.InnerInstructions {
			for _, ix := range set.Instructions {
				add(ix)
			}
		}
		// programs that appear only in inner instructions come after all
		// outer programs; within each part, first appearance wins
		if got := ctx.Classifier.GetAllProgramIds(); !reflect.DeepEqual(got, want) {
			t.Errorf("%.12s: GetAllProgramIds = %v, want %v", sig, got, want)
		}
	}
}

func inList(list []string, id string) bool {
	for _, p := range list {
		if p == id {
			return true
		}
	}
	return false
}

// TestCoreTwoHopAggregateDirection: a 2-hop Jupiter route_v2 aggregated
// backwards (3ZLekZ -> 3ZLekZ) because trades were in alphabetical program
// order and GetFinalSwap sorted only more than 2 trades. core-8, amm-2.
func TestCoreTwoHopAggregateDirection(t *testing.T) {
	for _, sig := range []string{sigRouteV2A, sigRouteV2B} {
		tx, r := parseFixture(t, sig, nil)
		user := tx.Transaction.Message.AccountKeys[0].Pubkey
		if len(r.Trades) != 2 || utils.CompareIdx(r.Trades[0].Idx, r.Trades[1].Idx) >= 0 {
			t.Fatalf("%.8s: trades %v not in execution order", sig, r.Trades)
		}
		agg := r.AggregateTrade
		want := ownerTokenDelta(tx, user, launchMint)
		if agg == nil || agg.Type != types.TradeTypeBuy || agg.InputToken.Mint != solMint || agg.OutputToken.Mint != launchMint ||
			bigStr(agg.OutputToken.AmountRaw).Cmp(want) != 0 {
			t.Errorf("%.8s: aggregate %v, want BUY SOL -> %s %s", sig, agg, launchMint, want)
		}
		// the user spent more SOL (lamports) than the route input: fees and rent
		spent := new(big.Int).Neg(lamportDelta(tx, user))
		if agg != nil && bigStr(agg.InputToken.AmountRaw).Cmp(spent) > 0 {
			t.Errorf("%.8s: aggregate input %s exceeds the user's SOL spend %s", sig, agg.InputToken.AmountRaw, spent)
		}
	}
}

// TestCoreGetFinalSwapSortsTwoTrades: unit check of the 2-trade case with the
// second hop first.
func TestCoreGetFinalSwapSortsTwoTrades(t *testing.T) {
	hop1 := types.TradeInfo{Idx: "2-2", InputToken: types.TokenInfo{Mint: solMint, AmountRaw: "1000"}, OutputToken: types.TokenInfo{Mint: "X", AmountRaw: "500"}}
	hop2 := types.TradeInfo{Idx: "2-10", InputToken: types.TokenInfo{Mint: "X", AmountRaw: "500"}, OutputToken: types.TokenInfo{Mint: usdcMint, AmountRaw: "7"}}
	agg := utils.GetFinalSwap([]types.TradeInfo{hop2, hop1}, nil)
	if agg.InputToken.Mint != solMint || agg.InputToken.AmountRaw != "1000" || agg.OutputToken.Mint != usdcMint || agg.OutputToken.AmountRaw != "7" || agg.Idx != "2-2" {
		t.Errorf("aggregate = %s", tradeKey(*agg))
	}
}

// TestCoreRouteSelection: GetDexInfo picked routes alphabetically and treated
// vault-tagged programs as routes (MeteoraVault became the route of Jupiter
// and plain DAMM v1 swaps). constants-9, parity-12, amm-1.
func TestCoreRouteSelection(t *testing.T) {
	p := dexparser.NewDexParser()

	// Jupiter through DAMM v1 hops: the upstream TS output is 4 Jupiter-route
	// trades (non-aggregated), not the Jupiter legs plus duplicate DAMM legs
	r := p.ParseAll(loadFixture(t, sigJupDammVault), nil)
	if len(r.Trades) != 4 {
		t.Errorf("55VfB9g5: %d trades, want 4", len(r.Trades))
	}
	for _, tr := range r.Trades {
		if tr.Route != "Jupiter" || tr.ProgramId != constants.DEX_PROGRAMS.JUPITER.ID {
			t.Errorf("55VfB9g5 trade %s route=%s, want Jupiter", tr.Idx, tr.Route)
		}
	}

	// plain DAMM v1 swap: DAMM is the AMM, the vault is never the route
	ctx := newParseContext(loadFixture(t, sigDammV1), nil)
	if ctx.DexInfo.AMM != "MeteoraDamm" || ctx.DexInfo.Route != "" || ctx.DexInfo.ProgramId != constants.DEX_PROGRAMS.METEORA_DAMM.ID {
		t.Errorf("4uuw76SP DexInfo = %+v, want AMM MeteoraDamm without route", ctx.DexInfo)
	}

	// no fixture picks a vault or a wallet entry as ProgramId
	for _, sig := range fixtureSignatures(t, "json") {
		ctx := newParseContext(loadFixture(t, sig), nil)
		prog := constants.GetDexProgramByID(ctx.DexInfo.ProgramId)
		for _, tag := range prog.Tags {
			if tag == "vault" {
				t.Errorf("%.12s: DexInfo.ProgramId %s is a vault", sig, prog.Name)
			}
		}
		// D2: the first known DEX program in execution order
		for _, id := range ctx.Classifier.GetAllProgramIds() {
			p := constants.GetDexProgramByID(id)
			if p.Name == "" || strings.Contains(strings.Join(p.Tags, ","), "vault") {
				continue
			}
			if id == constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER1.ID || id == constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER2.ID ||
				id == constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER3.ID || id == constants.DEX_PROGRAMS.OKX_ROUTER.ID {
				continue
			}
			if ctx.DexInfo.ProgramId != id {
				t.Errorf("%.12s: DexInfo.ProgramId = %s, want first known program %s", sig, ctx.DexInfo.ProgramId, id)
			}
			break
		}
	}
}

// TestCoreVaultProgramsFromTags: isIgnoredProgram hardcoded three vaults and
// missed SERUM_V3 (tagged vault): transfers inside a vault CPI must be grouped
// under the calling AMM. core-21.
//
// Synthetic: built from the real 4uuw76SP (DAMM v1 -> Meteora vault CPIs) by
// replacing the Meteora vault program key with the Serum V3 program ID.
func TestCoreVaultProgramsFromTags(t *testing.T) {
	orig := loadFixture(t, sigDammV1)
	origKeys := groupKeys(newParseContext(orig, nil).TransferActions)

	tx := cloneTx(t, orig)
	replaced := false
	for i, k := range tx.Transaction.Message.AccountKeys {
		if k.Pubkey == constants.DEX_PROGRAMS.METEORA_VAULT.ID {
			tx.Transaction.Message.AccountKeys[i].Pubkey = constants.DEX_PROGRAMS.SERUM_V3.ID
			replaced = true
		}
	}
	for i, k := range tx.Meta.LoadedAddresses.Readonly {
		if k == constants.DEX_PROGRAMS.METEORA_VAULT.ID {
			tx.Meta.LoadedAddresses.Readonly[i] = constants.DEX_PROGRAMS.SERUM_V3.ID
			replaced = true
		}
	}
	if !replaced {
		t.Fatal("fixture has no Meteora vault key")
	}
	keys := groupKeys(newParseContext(tx, nil).TransferActions)
	if !reflect.DeepEqual(keys, origKeys) {
		t.Errorf("groups with a Serum V3 CPI = %v, want %v (vault CPIs grouped under the caller)", keys, origKeys)
	}
	for _, k := range keys {
		if strings.HasPrefix(k, constants.DEX_PROGRAMS.SERUM_V3.ID) || strings.HasPrefix(k, constants.DEX_PROGRAMS.METEORA_VAULT.ID) {
			t.Errorf("transfer group keyed by a vault: %s", k)
		}
	}
}

func groupKeys(actions map[string][]types.TransferData) []string {
	return utils.SortedTransferKeys(actions)
}

// TestCoreTradesInExecutionOrder: result.Trades used to follow the
// alphabetical program order. core-8, parity-13.
func TestCoreTradesInExecutionOrder(t *testing.T) {
	p := dexparser.NewDexParser()
	multi := 0
	for _, sig := range fixtureSignatures(t, "json") {
		trades := p.ParseTrades(loadFixture(t, sig), &types.ParseConfig{TryUnknownDEX: true, IncludeFailedTxs: true})
		if len(trades) > 1 {
			multi++
		}
		for i := 1; i < len(trades); i++ {
			if utils.CompareIdx(trades[i-1].Idx, trades[i].Idx) > 0 {
				t.Errorf("%.12s: trade %s before %s", sig, trades[i-1].Idx, trades[i].Idx)
			}
		}
	}
	if multi < 20 {
		t.Errorf("only %d multi-trade fixtures", multi)
	}
}

// TestCoreDeterministicOutput: transfers, unknown-DEX trades and attached
// transfer info came from Go map iteration and changed between runs. core-9.
func TestCoreDeterministicOutput(t *testing.T) {
	p := dexparser.NewDexParser()
	for _, sig := range []string{sigJup3Hop, sigRouteV2A, "5sV51YrwGRFwpwgnWHWKCq3TWmCQ47XnsNuMNAKs7Jypf9g6H9JTPnY6ZqW5tfSTBwBL8Un5MPxK5cKZsSkaacty"} {
		tx := loadFixture(t, sig)
		first := p.ParseAll(tx, &types.ParseConfig{ParseType: types.ParseAll(), TryUnknownDEX: true})
		transfers := p.ParseTransfers(tx, nil)
		for i := 1; i < len(transfers); i++ {
			if utils.CompareIdx(transfers[i-1].Idx, transfers[i].Idx) > 0 {
				t.Errorf("%.8s: transfer %s before %s", sig, transfers[i-1].Idx, transfers[i].Idx)
			}
		}
		for run := 0; run < 30; run++ {
			again := p.ParseAll(tx, &types.ParseConfig{ParseType: types.ParseAll(), TryUnknownDEX: true})
			if !reflect.DeepEqual(again.Trades, first.Trades) || !reflect.DeepEqual(again.AggregateTrade, first.AggregateTrade) {
				t.Fatalf("%.8s: run %d differs", sig, run)
			}
			if got := p.ParseTransfers(tx, nil); !reflect.DeepEqual(got, transfers) {
				t.Fatalf("%.8s: transfers differ on run %d", sig, run)
			}
		}
	}
}
