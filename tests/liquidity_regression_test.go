package tests

import (
	"encoding/binary"
	"math/big"
	"strconv"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/meteora"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

const wsol = "So11111111111111111111111111111111111111112"

// instructionAt returns the classified instruction of programId at (outer, inner)
func instructionAt(t *testing.T, ctx *parseContext, programId string, outer, inner int) types.ClassifiedInstruction {
	t.Helper()
	for _, ci := range ctx.Classifier.GetInstructions(programId) {
		if ci.OuterIndex == outer && ci.InnerIndex == inner {
			return ci
		}
	}
	t.Fatalf("no %s instruction at %d/%d", programId, outer, inner)
	return types.ClassifiedInstruction{}
}

func u64At(data []byte, offset int) string {
	return new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[offset : offset+8])).String()
}

// signerTokenChange returns the signer's balance change of mint (raw, signed)
func signerTokenChange(t *testing.T, result *types.ParseResult, mint string) string {
	t.Helper()
	change, ok := result.TokenBalanceChange[mint]
	if !ok || change == nil {
		t.Fatalf("no signer balance change for %s", mint)
	}
	return change.Change.Amount
}

func findLiquidity(events []types.PoolEvent, typ types.PoolEventType, idx string) *types.PoolEvent {
	for i := range events {
		if events[i].Type == typ && events[i].Idx == idx {
			return &events[i]
		}
	}
	return nil
}

func neg(amount string) string {
	v, _ := new(big.Int).SetString(amount, 10)
	return new(big.Int).Neg(v).String()
}

// TestLiquidityOuterInstructionTransfers: the liquidity parsers must look up the
// transfers of an outer instruction under "<program>:<outer>", not
// "<program>:<outer>-0" (amm-4, parity-3). Truth: the signer's token balance
// change of the non-SOL token equals the deposited amount.
func TestLiquidityOuterInstructionTransfers(t *testing.T) {
	cases := []struct {
		name, sig, idx string
		typ            types.PoolEventType
		mint0, amount0 string
		amount1        string
	}{
		{"Orca increase_liquidity", "3mZcyeDJysgs79nLcvtN4XQ6iepyERqG93P2F2ZYgUX4ZF1Yr1XFBMKR8DHd7z4gN2EmvAqMc3KhQTQpGMbtvhF7", "5", types.PoolEventTypeAdd, "JUPyiwrYJFskUPiHa7hkeR8VUtAeFoSYbKedZNsDvCN", "6931015285", "45407996732"},
		{"DLMM add_liquidity", "2vkT747Y9udxkiCD6bqGTME65G47xnrQ9mtvwoHXBwPNQJ2he6XBCGemxiD55oDeKcZ4vHbfgTYiX8ofr1a4phD6", "2", types.PoolEventTypeAdd, "7w4XU7wWKoCB3bfM7Vi7zmHVSokoPP7Nb2oQuceiWb3s", "8240658", ""},
		{"Raydium V4 deposit", "4zFeXoUVaaQ18chkY899hvvTYBRMAJ6CaNfWxbchMDDEY1L56maqwAdY19Faif5LBoTxwBPeEamcEsh76b39fjia", "4", types.PoolEventTypeAdd, "7oBYdEhV4GkXC19ZfgAvXpJWp2Rn9pm1Bx2cVNxFpump", "1950837707", "147754335"},
		{"Raydium V4 initialize2", "2YxPyAJNfnBLrVpBwMx7qMVNPSvBDhxiquwJGhBjwXhkP6i6AbooUg4b4wpi15bQq2Qs4t7BpL1UVvTMcXL8P4uS", "4", types.PoolEventTypeCreate, "3ZmaEAKC75ereKNTxZeaDxxHLNWag2rL43UY19Rw6v7D", "50000000000000", "1000000000"},
		{"Raydium CPMM initialize", "xZmKodPHYxesDJzPLHfjqG4VvKJcQpwuo3fTa2T64LY9nePfAa97xtSmzCvSWJ5EFzYjAURLD666zqV8oJL8kTp", "4", types.PoolEventTypeCreate, "znv3FZt2HFAvzYf5LxzVyryh3mBXWuTRRng25gEZAjh", "1000000000000000", "20000000000"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			tx := loadFixture(t, c.sig)
			result := dexparser.NewDexParser().ParseAll(tx, &types.ParseConfig{ParseType: types.ParseType{Liquidity: true}})
			e := findLiquidity(result.Liquidities, c.typ, c.idx)
			if e == nil {
				t.Fatalf("no %s event at idx %s: %+v", c.typ, c.idx, result.Liquidities)
			}
			if e.Token0Mint != c.mint0 || e.Token0AmountRaw != c.amount0 {
				t.Errorf("token0 = %s %s, want %s %s", e.Token0Mint, e.Token0AmountRaw, c.mint0, c.amount0)
			}
			if got := signerTokenChange(t, result, c.mint0); got != neg(c.amount0) {
				t.Errorf("signer balance change of %s = %s, event says %s was deposited", c.mint0, got, c.amount0)
			}
			if c.amount1 != "" && (e.Token1Mint != wsol || e.Token1AmountRaw != c.amount1) {
				t.Errorf("token1 = %s %s, want %s %s", e.Token1Mint, e.Token1AmountRaw, wsol, c.amount1)
			}
		})
	}
}

// TestRaydiumV4CreateMatchesInitLog: initialize2 reports the initial pool
// amounts; they must equal the program's own InitLog ("ray_log:" line) and
// the LP minted to the creator (amm-13, parity-16).
func TestRaydiumV4CreateMatchesInitLog(t *testing.T) {
	for _, sig := range []string{
		"2YxPyAJNfnBLrVpBwMx7qMVNPSvBDhxiquwJGhBjwXhkP6i6AbooUg4b4wpi15bQq2Qs4t7BpL1UVvTMcXL8P4uS",
		"FHz3LurEFNnWREXSfqpenJTRuzybQrmxsNjndmMDw36XUwUofSQ1vRQwVi6uT422Xe2Whb7gfFY3tHGqvEBg6dF",
		"5MRWUUoCWpaFm8B9jSoLp4w6B19p46jcJMQrm26SHHGRZQpAtG3mdEcqGVDwsgUEGKRmC1R6JMFS5NhzmXUnR3X1",
		"4998xRsghpLWWcZsFFtfN8UBss9SVkN8sMUPkqdBzYS6eRVuxve3RoT89pqBaYvHDgqPSsFt9GDGQc6Us3pwPX5v",
		"49ejTjaCdqV3hwAs1GomGsTLqZNUWztHduHRCNr8m3Dogfyii29qkNPNauAr9VfEh9j5m7QHq8HYRd6FiaAACCf",
	} {
		sig := sig
		t.Run(sig[:12], func(t *testing.T) {
			tx := loadFixture(t, sig)
			ctx := newParseContext(tx, nil)
			var initLog *raydium.InitLog
			for _, l := range ctx.Utils.GetRayLogs() {
				if il, ok := raydium.DecodeRaydiumLog(l.Data).(*raydium.InitLog); ok {
					initLog = il
				}
			}
			if initLog == nil {
				t.Fatal("no InitLog in the transaction logs")
			}
			events := dexparser.NewDexParser().ParseLiquidity(tx, liquidityConfig())
			if len(events) != 1 || events[0].Type != types.PoolEventTypeCreate {
				t.Fatalf("want one CREATE event, got %+v", events)
			}
			e := events[0]
			amounts := map[string]bool{e.Token0AmountRaw: true, e.Token1AmountRaw: true}
			if !amounts[initLog.CoinAmount.String()] || !amounts[initLog.PcAmount.String()] {
				t.Errorf("amounts %s / %s, InitLog coin %s pc %s", e.Token0AmountRaw, e.Token1AmountRaw, initLog.CoinAmount, initLog.PcAmount)
			}
			if e.PoolLpMint == "" || e.LpAmountRaw == "" || e.LpAmountRaw == "0" {
				t.Errorf("LP mint %q amount %q missing", e.PoolLpMint, e.LpAmountRaw)
			}
		})
	}
}

// TestRaydiumCLLiquidityEvents covers the Raydium CLMM liquidity parser: the
// named instruction type used to fall through the type switch (amm-5), and
// pool indices and event types follow the raydium_clmm IDL and upstream
// RaydiumCLPoolV2Parser (amm-10, parity-8, constants-3).
func TestRaydiumCLLiquidityEvents(t *testing.T) {
	cases := []struct {
		name, sig, idx string
		typ            types.PoolEventType
		pool           string
	}{
		{"decrease_liquidity_v2", "48oGGt6rsBqbiyj7xyzWD8oRXk3sGhE6Kt6LzRG5QofL1LJqwtcfp4uKXeinDn4a24uyWJVGDPTaAFac2R5eNX1w", "4", types.PoolEventTypeRemove, "DFX9AHEnoU8caagtFFGiv7xnsEJ3DTh5TQo8XktJHoTN"},
		// pool_state is account 4, account 2 is the position NFT mint
		{"open_position_with_token22_nft", "5BEKeMuUfah3wFkCvMmaGDq5JDTat2nokaqMURYZubNDm9WdQrMmwWK4YcL7nksuq94k62wxgbbbwUf5LCgtXU4J", "4", types.PoolEventTypeAdd, "DFX9AHEnoU8caagtFFGiv7xnsEJ3DTh5TQo8XktJHoTN"},
		// open_position deposits into an existing pool: ADD, pool_state is account 5
		{"open_position (inner)", "54t2sbzBxmejGNYmttn5nr4fDeRHdZC6CF4eiAm3Was7WKrvw6LEH51gb2RGw5Wom2y12vW11o2hwaHBE3W4Rgx9", "2-0", types.PoolEventTypeAdd, "5MczDZ1DYBpCyjXWhyLXAW4AofywKizDF8w4cLZqvpuV"},
		{"open_position_v2", "61auheJ9MhRbQeXqiAMitgWWv2yxkDSCqb6qauMr77UaENwR7yUfc5LCH8pExkjo1QqnqYLvRu9vNXi5QRZST19S", "4", types.PoolEventTypeAdd, "EYy5nwcFQHfSCKrrALgMB2DbU2vhszWHjuEdDFgAvfpu"},
		// create_pool: pool_state is account 2, account 4 is token_mint_1
		{"create_pool", "4Vv9ZWLizvRE7um22gF8bUWvD5UfK1TMXsP4hF8TVF4gc2BNmmPG8kFu7Dyod9Zw5x16xAsGeDJnUznCwaKXim5n", "2", types.PoolEventTypeCreate, "GQsPr4RJk9AZkkfWHud7v4MtotcxhaYzZHdsPCg9vNvW"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			events := dexparser.NewDexParser().ParseLiquidity(loadFixture(t, c.sig), liquidityConfig())
			e := findLiquidity(events, c.typ, c.idx)
			if e == nil {
				t.Fatalf("no %s event at idx %s: %+v", c.typ, c.idx, events)
			}
			if e.PoolId != c.pool {
				t.Errorf("pool = %s, want %s", e.PoolId, c.pool)
			}
			for _, other := range events {
				if other.Type == types.PoolEventTypeCreate && c.typ != types.PoolEventTypeCreate {
					t.Errorf("unexpected CREATE event %+v", other)
				}
			}
		})
	}
}

// TestRaydiumCLCreateMatchesPoolCreatedEvent: the CREATE event of create_pool
// carries the pool and mints the program reports in its PoolCreatedEvent log.
func TestRaydiumCLCreateMatchesPoolCreatedEvent(t *testing.T) {
	poolCreated := constants.DISCRIMINATORS.RAYDIUM_CL.EVENTS.POOL_CREATED
	for _, sig := range []string{
		"PGYHMva3trxE1BR2Qr4H8DZDcSLeHdHcCgJT2iAarvrE9xovBu5dtrNiti3xXxUKoNgT9WeifFPA6zX8DjsASdP",
		"4Vv9ZWLizvRE7um22gF8bUWvD5UfK1TMXsP4hF8TVF4gc2BNmmPG8kFu7Dyod9Zw5x16xAsGeDJnUznCwaKXim5n",
	} {
		sig := sig
		t.Run(sig[:12], func(t *testing.T) {
			tx := loadFixture(t, sig)
			ctx := newParseContext(tx, nil)
			var mint0, mint1, pool string
			for _, l := range ctx.Utils.GetProgramDataLogs() {
				// PoolCreatedEvent: token_mint_0, token_mint_1, tick_spacing u16, pool_state, ...
				if l.ProgramId == constants.DEX_PROGRAMS.RAYDIUM_CL.ID && constants.MatchDiscriminator(l.Data, poolCreated) && len(l.Data) >= 8+32+32+2+32 {
					mint0 = base58.Encode(l.Data[8:40])
					mint1 = base58.Encode(l.Data[40:72])
					pool = base58.Encode(l.Data[74:106])
				}
			}
			if pool == "" {
				t.Fatal("no PoolCreatedEvent log")
			}
			events := dexparser.NewDexParser().ParseLiquidity(tx, liquidityConfig())
			var create *types.PoolEvent
			for i := range events {
				if events[i].Type == types.PoolEventTypeCreate {
					create = &events[i]
				}
			}
			if create == nil {
				t.Fatalf("no CREATE event: %+v", events)
			}
			if create.PoolId != pool {
				t.Errorf("pool = %s, PoolCreatedEvent pool_state %s", create.PoolId, pool)
			}
			mints := map[string]bool{create.Token0Mint: true, create.Token1Mint: true}
			if !mints[mint0] || !mints[mint1] {
				t.Errorf("mints %s / %s, PoolCreatedEvent %s / %s", create.Token0Mint, create.Token1Mint, mint0, mint1)
			}
			if create.Token1Mint != wsol {
				t.Errorf("token1 = %s, want the quote mint SOL", create.Token1Mint)
			}
		})
	}
}

// TestRaydiumCLDecreaseDataOffsets: without transfers the decrease_liquidity
// amounts come from the instruction args: liquidity u128 @8, amount_0_min @24,
// amount_1_min @32 (the old offsets 32/24 swapped the two tokens, amm-10).
func TestRaydiumCLDecreaseDataOffsets(t *testing.T) {
	tx := loadFixture(t, "48oGGt6rsBqbiyj7xyzWD8oRXk3sGhE6Kt6LzRG5QofL1LJqwtcfp4uKXeinDn4a24uyWJVGDPTaAFac2R5eNX1w")
	ctx := newParseContext(tx, nil)
	ci := instructionAt(t, ctx, constants.DEX_PROGRAMS.RAYDIUM_CL.ID, 4, -1)
	data := ctx.Adapter.GetInstructionData(ci.Instruction)

	parser := raydium.NewRaydiumCLPoolParser(ctx.Adapter, map[string][]types.TransferData{}, []types.ClassifiedInstruction{ci})
	events := parser.ProcessLiquidity()
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %+v", events)
	}
	if events[0].Token0AmountRaw != u64At(data, 24) || events[0].Token1AmountRaw != u64At(data, 32) {
		t.Errorf("amounts %s / %s, want amount_0_min %s / amount_1_min %s",
			events[0].Token0AmountRaw, events[0].Token1AmountRaw, u64At(data, 24), u64At(data, 32))
	}
}

// TestMeteoraDLMMIdxAndClaimFee: DLMM events at outer 10 and 11 had idx "0"
// and "1", and the claim_fee event took the position as pool and the user
// token accounts as mints (amm-11). Truth: the ClaimFee CPI event's lb_pair
// and the claim_fee IDL accounts (token_x_mint 9, token_y_mint 10).
func TestMeteoraDLMMIdxAndClaimFee(t *testing.T) {
	const lbPair = "76AdJZLoVDovb37KD6V8EywZLX3UXVVeTXkZ8PEz6Sth"
	tx := loadFixture(t, "7YPF21r7JBDeoXuMJn6KSqDVYGrm821U87Cnje3xPvZpMUVaAEAvCGJPP6va2b5oMLAzGku5s3TcNAsN6zdXPRn")
	events := dexparser.NewDexParser().ParseLiquidity(tx, liquidityConfig())

	var idxs []string
	for _, e := range events {
		idxs = append(idxs, e.Idx)
	}
	for _, want := range []string{"4", "10", "11"} {
		if findLiquidityIdx(events, want) == nil {
			t.Errorf("no event with idx %s (got %v)", want, idxs)
		}
	}

	ctx := newParseContext(tx, nil)
	claimEvent := instructionAt(t, ctx, constants.DEX_PROGRAMS.METEORA.ID, 11, 1)
	eventData := ctx.Adapter.GetInstructionData(claimEvent.Instruction)
	if got := base58.Encode(eventData[16:48]); got != lbPair {
		t.Fatalf("ClaimFee event lb_pair = %s, want %s", got, lbPair)
	}
	claim := findLiquidityIdx(events, "11")
	if claim == nil {
		t.Fatal("no claim_fee event")
	}
	if claim.Type != types.PoolEventTypeRemove || claim.PoolId != lbPair {
		t.Errorf("claim_fee event %s pool=%s, want REMOVE pool=%s", claim.Type, claim.PoolId, lbPair)
	}
	claimIx := ctx.Adapter.GetInstructionAccounts(instructionAt(t, ctx, constants.DEX_PROGRAMS.METEORA.ID, 11, -1).Instruction)
	if claim.Token0Mint != claimIx[9] || claim.Token1Mint != claimIx[10] {
		t.Errorf("claim_fee mints %s / %s, want token_x_mint %s / token_y_mint %s", claim.Token0Mint, claim.Token1Mint, claimIx[9], claimIx[10])
	}
	if claim.Token0AmountRaw == "" {
		t.Error("claim_fee amount missing")
	}
}

func findLiquidityIdx(events []types.PoolEvent, idx string) *types.PoolEvent {
	for i := range events {
		if events[i].Idx == idx {
			return &events[i]
		}
	}
	return nil
}

// TestLiquidityInnerInstructionIdx: events of inner instructions carry the
// "outer-inner" idx (amm-22).
func TestLiquidityInnerInstructionIdx(t *testing.T) {
	cases := []struct{ sig, idx string }{
		{"54t2sbzBxmejGNYmttn5nr4fDeRHdZC6CF4eiAm3Was7WKrvw6LEH51gb2RGw5Wom2y12vW11o2hwaHBE3W4Rgx9", "2-0"}, // CLMM open_position via CPI
		{"7JX8oo13G3q812rQ7FwarAKGHzHBKS4JqQSfkpso826HeFifupXgBo6yhnkXE8Yt8xfAG2Qv6SwvrNgRYWiEy8K", "2-0"},  // CLMM decrease_liquidity via CPI
		{"5yrQXvm3LHuJo9RChAswShLUnYJJDLPRUEWu6WmYZ9xsKFutbtQokgG6duU4gy5C6N3MXrM2a7c622bu49agW2uA", "0-1"}, // DAMM v2 pool created by a DBC migration
	}
	for _, c := range cases {
		events := dexparser.NewDexParser().ParseLiquidity(loadFixture(t, c.sig), liquidityConfig())
		if len(events) != 1 || events[0].Idx != c.idx {
			t.Errorf("%s: want one event with idx %s, got %+v", c.sig[:12], c.idx, events)
		}
	}
}

// TestMeteoraDammV1DataOffsets: without transfers, DAMM v1 amounts come from
// the instruction args per the amm IDL (amm-12). Truth: with transfers, the
// same instructions deposit exactly token_a_amount / token_b_amount (create)
// or at most maximum_token_a/b_amount (add).
func TestMeteoraDammV1DataOffsets(t *testing.T) {
	cases := []struct {
		name, sig          string
		outer              int
		offsetA, offsetB   int
		mintAIdx, mintBIdx int
		lpOffset           int
	}{
		// initialize_permissionless_constant_product_pool_with_config: token_a_amount @8, token_b_amount @16
		{"create", "2GWLwbEjyR7moFYK5JfapbsDBrBz3298BVWsAebhUECPaXjLbTZ6DkbEN34BF57jdGot7GkwDnrzszFB3H9AJxmS", 6, 8, 16, 3, 4, -1},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			tx := loadFixture(t, c.sig)
			ctx := newParseContext(tx, nil)
			ci := instructionAt(t, ctx, constants.DEX_PROGRAMS.METEORA_DAMM.ID, c.outer, -1)
			data := ctx.Adapter.GetInstructionData(ci.Instruction)
			accounts := ctx.Adapter.GetInstructionAccounts(ci.Instruction)

			// with transfers: the real deposit
			full := meteora.NewMeteoraPoolsParser(ctx.Adapter, ctx.TransferActions, []types.ClassifiedInstruction{ci}).ProcessLiquidity()
			// without transfers: the data fallback
			fallback := meteora.NewMeteoraPoolsParser(ctx.Adapter, map[string][]types.TransferData{}, []types.ClassifiedInstruction{ci}).ProcessLiquidity()
			if len(full) != 1 || len(fallback) != 1 {
				t.Fatalf("want one event each, got %+v / %+v", full, fallback)
			}
			fb := fallback[0]
			if fb.Token0Mint != accounts[c.mintAIdx] || fb.Token1Mint != accounts[c.mintBIdx] {
				t.Errorf("fallback mints %s / %s, want token_a_mint %s / token_b_mint %s", fb.Token0Mint, fb.Token1Mint, accounts[c.mintAIdx], accounts[c.mintBIdx])
			}
			if fb.Token0AmountRaw != u64At(data, c.offsetA) || fb.Token1AmountRaw != u64At(data, c.offsetB) {
				t.Errorf("fallback amounts %s / %s, want token_a %s / token_b %s", fb.Token0AmountRaw, fb.Token1AmountRaw, u64At(data, c.offsetA), u64At(data, c.offsetB))
			}
			realByMint := map[string]string{full[0].Token0Mint: full[0].Token0AmountRaw, full[0].Token1Mint: full[0].Token1AmountRaw}
			if realByMint[fb.Token0Mint] != fb.Token0AmountRaw || realByMint[fb.Token1Mint] != fb.Token1AmountRaw {
				t.Errorf("fallback %s=%s %s=%s differs from the transferred amounts %v", fb.Token0Mint, fb.Token0AmountRaw, fb.Token1Mint, fb.Token1AmountRaw, realByMint)
			}
		})
	}
}

// TestMeteoraDammV2DynamicConfigMints: initialize_pool_with_dynamic_config
// has token_a_mint at 9 and token_b_mint at 10 (amm-12). Truth: the
// EvtInitializePool self-CPI event (pool, token_a_mint, token_b_mint).
func TestMeteoraDammV2DynamicConfigMints(t *testing.T) {
	tx := loadFixture(t, "5yrQXvm3LHuJo9RChAswShLUnYJJDLPRUEWu6WmYZ9xsKFutbtQokgG6duU4gy5C6N3MXrM2a7c622bu49agW2uA")
	ctx := newParseContext(tx, nil)
	program := constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID
	evt := ctx.Adapter.GetInstructionData(instructionAt(t, ctx, program, 0, 20).Instruction)
	if !constants.MatchDiscriminator(evt, constants.DISCRIMINATORS.METEORA_DAMM_V2.EVT_INITIALIZE_POOL) || len(evt) < 16+96 {
		t.Fatalf("0-20 is not an EvtInitializePool event")
	}
	pool, mintA, mintB := base58.Encode(evt[16:48]), base58.Encode(evt[48:80]), base58.Encode(evt[80:112])

	for _, transfers := range []map[string][]types.TransferData{ctx.TransferActions, {}} {
		events := meteora.NewMeteoraDAMMPoolParser(ctx.Adapter, transfers, ctx.Classifier.GetInstructions(program)).ProcessLiquidity()
		if len(events) != 1 {
			t.Fatalf("want one event, got %+v", events)
		}
		e := events[0]
		if e.PoolId != pool {
			t.Errorf("pool %s, EvtInitializePool %s", e.PoolId, pool)
		}
		if mints := map[string]bool{e.Token0Mint: true, e.Token1Mint: true}; !mints[mintA] || !mints[mintB] {
			t.Errorf("mints %s / %s, EvtInitializePool %s / %s (%d transfer groups)", e.Token0Mint, e.Token1Mint, mintA, mintB, len(transfers))
		}
	}
}

// TestMeteoraDammV2LiquidityChangeEvent: a DAMM v2 add_liquidity whose
// transfers are not available still gets its amounts from the
// EvtLiquidityChange self-CPI (streamer item 5). Truth: the real transfers of
// the same instruction (5-0, 5-1).
func TestMeteoraDammV2LiquidityChangeEvent(t *testing.T) {
	tx := loadFixture(t, "67SA1qv4f6ZY948qt7C22dTReS8EcGG8PkVJYdoqSXUxf3h2QPUjdnbu6hqdR79WR1CYxweCePycpcuTFR8WYWbr")
	ctx := newParseContext(tx, nil)
	program := constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID
	full := meteora.NewMeteoraDAMMPoolParser(ctx.Adapter, ctx.TransferActions, ctx.Classifier.GetInstructions(program)).ProcessLiquidity()
	fromEvent := meteora.NewMeteoraDAMMPoolParser(ctx.Adapter, map[string][]types.TransferData{}, ctx.Classifier.GetInstructions(program)).ProcessLiquidity()
	if len(full) != 1 || len(fromEvent) != 1 {
		t.Fatalf("want one event each, got %+v / %+v", full, fromEvent)
	}
	a, b := full[0], fromEvent[0]
	if a.Type != types.PoolEventTypeAdd || a.Token0AmountRaw != transferAmount(t, ctx, 5, 0) || a.Token1AmountRaw != transferAmount(t, ctx, 5, 1) {
		t.Errorf("with transfers: %s %s %s / %s %s", a.Type, a.Token0Mint, a.Token0AmountRaw, a.Token1Mint, a.Token1AmountRaw)
	}
	if b.Token0Mint != a.Token0Mint || b.Token1Mint != a.Token1Mint || b.Token0AmountRaw != a.Token0AmountRaw || b.Token1AmountRaw != a.Token1AmountRaw {
		t.Errorf("from EvtLiquidityChange: %s %s / %s %s, transfers %s %s / %s %s",
			b.Token0Mint, b.Token0AmountRaw, b.Token1Mint, b.Token1AmountRaw, a.Token0Mint, a.Token0AmountRaw, a.Token1Mint, a.Token1AmountRaw)
	}
}

// TestMeteoraDammV1NewLiquidityInstructions: the DAMM v1 create variants and
// bootstrap_liquidity are liquidity events (amm-18). There is no real
// transaction for them in the fixture pool; each case is synthetic, built
// here from a real transaction whose instruction has the same account layout
// and leading args: initialize_permissionless_constant_product_pool_with_config2
// (token_a_amount, token_b_amount, activation_point) from ..._with_config,
// bootstrap_liquidity (pool 0, lp_mint 1, same accounts) from
// add_balance_liquidity.
func TestMeteoraDammV1NewLiquidityInstructions(t *testing.T) {
	d := constants.DISCRIMINATORS.METEORA_DAMM
	cases := []struct {
		name, sig string
		outer     int
		disc      []byte
		typ       types.PoolEventType
	}{
		{"create_with_config2", "2GWLwbEjyR7moFYK5JfapbsDBrBz3298BVWsAebhUECPaXjLbTZ6DkbEN34BF57jdGot7GkwDnrzszFB3H9AJxmS", 6, d.CREATE_WITH_CONFIG2, types.PoolEventTypeCreate},
		{"bootstrap_liquidity", "LaocVd6PpfdH1KTdQuRTf5WwnUzmyf3gdAy16xro747nzrhpgXg1oxFrpgBk31tPh24ksVAyiSkNW7vncoKTGyH", 2, d.BOOTSTRAP_LIQUIDITY, types.PoolEventTypeAdd},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			tx := loadFixture(t, c.sig)
			original := dexparser.NewDexParser().ParseLiquidity(tx, liquidityConfig())
			ctx := newParseContext(tx, nil)
			data := append([]byte(nil), ctx.Adapter.GetInstructionData(ctx.Adapter.InstructionAt(c.outer))...)
			copy(data, c.disc)
			patchInstructionData(t, tx, c.outer, data)
			events := dexparser.NewDexParser().ParseLiquidity(tx, liquidityConfig())
			if len(original) != 1 || len(events) != 1 {
				t.Fatalf("want one event, got %+v / %+v", original, events)
			}
			o, e := original[0], events[0]
			if e.Type != c.typ || e.PoolId != o.PoolId || e.Token0Mint != o.Token0Mint || e.Token0AmountRaw != o.Token0AmountRaw ||
				e.Token1Mint != o.Token1Mint || e.Token1AmountRaw != o.Token1AmountRaw {
				t.Errorf("got %s pool=%s %s %s / %s %s, want %s like %+v", e.Type, e.PoolId, e.Token0Mint, e.Token0AmountRaw, e.Token1Mint, e.Token1AmountRaw, c.typ, o)
			}
		})
	}
}

// TestAmmNewLiquidityInstructions: Orca increase_liquidity_by_token_amounts_v2
// and Raydium CLMM create_customizable_pool are liquidity events (amm-6,
// amm-18, constants-2). No real transaction for them is in the fixture pool;
// each case is synthetic, built here from a real transaction of the
// instruction with the same account layout (increase_liquidity_v2,
// create_pool).
func TestAmmNewLiquidityInstructions(t *testing.T) {
	cases := []struct {
		name, sig string
		outer     int
		disc      []byte
	}{
		{"orca increase_liquidity_by_token_amounts_v2", "4Kv6gQgdSsCPSxRApiCNMHFE1dKKGVugrJTzdzSYX5a2aXho4o7jaQDSHLH3RTsr5aVwpkzWL1o5mSCyDtHeZKZr", 6, constants.DISCRIMINATORS.ORCA.ADD_LIQUIDITY_BY_TOKEN_AMOUNTS_V2},
		{"clmm create_customizable_pool", "4Vv9ZWLizvRE7um22gF8bUWvD5UfK1TMXsP4hF8TVF4gc2BNmmPG8kFu7Dyod9Zw5x16xAsGeDJnUznCwaKXim5n", 2, constants.DISCRIMINATORS.RAYDIUM_CL.CREATE.CREATE_CUSTOMIZABLE_POOL},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			tx := loadFixture(t, c.sig)
			idx := strconv.Itoa(c.outer)
			original := findLiquidityIdx(dexparser.NewDexParser().ParseLiquidity(tx, liquidityConfig()), idx)
			ctx := newParseContext(tx, nil)
			data := append([]byte(nil), ctx.Adapter.GetInstructionData(ctx.Adapter.InstructionAt(c.outer))...)
			copy(data, c.disc)
			patchInstructionData(t, tx, c.outer, data)
			e := findLiquidityIdx(dexparser.NewDexParser().ParseLiquidity(tx, liquidityConfig()), idx)
			if original == nil || e == nil {
				t.Fatalf("no event at %s: %+v / %+v", idx, original, e)
			}
			if e.Type != original.Type || e.PoolId != original.PoolId || e.Token0Mint != original.Token0Mint || e.Token0AmountRaw != original.Token0AmountRaw ||
				e.Token1Mint != original.Token1Mint || e.Token1AmountRaw != original.Token1AmountRaw {
				t.Errorf("got %+v, want the event of the original instruction %+v", *e, *original)
			}
		})
	}
}
