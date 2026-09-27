package tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Regression tests for the AMM findings of the final review (amm-1 to amm-6,
// amm-v1). Expected values come from the raw fixtures: event logs and
// instruction data decoded with the programs' layouts, and balance changes.

// anchorEventDisc returns the Anchor event discriminator sha256("event:<name>")[:8]
func anchorEventDisc(name string) []byte {
	h := sha256.Sum256([]byte("event:" + name))
	return h[:8]
}

// programDataEvents returns the decoded "Program data:" payloads of a fixture
// that start with disc
func programDataEvents(tx *adapterTx, disc []byte) [][]byte {
	var out [][]byte
	if tx.Meta == nil {
		return nil
	}
	for _, l := range tx.Meta.LogMessages {
		if !strings.HasPrefix(l, "Program data: ") {
			continue
		}
		b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(l, "Program data: "))
		if err == nil && len(b) >= 8 && string(b[:8]) == string(disc) {
			out = append(out, b)
		}
	}
	return out
}

// TestFinalCLMMTradeFee: current Raydium CLMM SwapEvents (221 bytes) end
// with trade_fee_0 and trade_fee_1 (raydium-clmm states/pool.rs), which the
// parser ignored, so CLMM trades had no fee. Truth: the events decoded here;
// the fee is in token 0 or token 1 (JdqsS4: on the output side). amm-3.
func TestFinalCLMMTradeFee(t *testing.T) {
	disc := anchorEventDisc("SwapEvent")
	for _, sig := range []string{
		"mLQ2LNtkgyRoTd62NoAoDs4pUBNqFXVx1TT4eTfnxZvmE3389x9zZZEEDC4SaEo49egH51ySS5gnwVvP7iQXq9v",
		"JdqsS4UeTSBb1Rs4Bw3sYoM9qSTTjiUd7zsB3EZWGFwijNrtJTJY7gNyVZmY7kq2sbsSQWyyCu6W7fybpA8Nkyn",
		"5gpxeUJjYMmPifsT2JQq1BmvYdTLunsub5JiwrfhiE1X29HyHKtJSjpk5kkeqiLoGW5DbEN8qscijvrqGPMdgY8y",
	} {
		tx, res := parseFixture(t, sig, nil)
		var checked int
		for _, ev := range programDataEvents(tx, disc) {
			if len(ev) != 221 {
				continue
			}
			pool := base58.Encode(ev[8:40])
			fees := map[int]uint64{0: binary.LittleEndian.Uint64(ev[205:]), 1: binary.LittleEndian.Uint64(ev[213:])}
			accounts := map[int]string{0: base58.Encode(ev[72:104]), 1: base58.Encode(ev[104:136])}
			zeroForOne := ev[168] != 0
			var trade *types.TradeInfo
			for i := range res.Trades {
				if res.Trades[i].AMM == constants.DEX_PROGRAMS.RAYDIUM_CL.Name && len(res.Trades[i].Pool) > 0 && res.Trades[i].Pool[0] == pool {
					trade = &res.Trades[i]
				}
			}
			if trade == nil {
				t.Fatalf("%s: no RaydiumCL trade on pool %s", sig[:8], pool)
			}
			var want []types.FeeInfo
			for side := 0; side < 2; side++ {
				if fees[side] > 0 {
					// the mint of the token account the event names, else
					// (a temporary WSOL account) the trade leg of that side
					mint := tokenBalanceMint(tx, accounts[side])
					if mint == "" {
						mint = trade.OutputToken.Mint
						if (side == 0) == zeroForOne {
							mint = trade.InputToken.Mint
						}
					}
					want = append(want, types.FeeInfo{Mint: mint, AmountRaw: u64str(fees[side]), Type: "trade"})
				}
			}
			if len(want) == 0 {
				t.Fatalf("%s: event of pool %s has no trade fee", sig[:8], pool)
			}
			var got []types.FeeInfo
			if trade.Fee != nil {
				got = append(got, *trade.Fee)
			}
			got = append(got, trade.Fees...)
			for _, w := range want {
				found := false
				for _, g := range got {
					if g.Mint == w.Mint && g.AmountRaw == w.AmountRaw && g.Type == w.Type && g.Dex == constants.DEX_PROGRAMS.RAYDIUM_CL.Name {
						found = true
					}
				}
				if !found {
					t.Errorf("%s pool %s: fee %s %s %s missing from %+v", sig[:8], pool, w.Type, w.AmountRaw, w.Mint, got)
				}
			}
			checked++
		}
		if checked == 0 {
			t.Fatalf("%s: no 221-byte CLMM SwapEvent", sig[:8])
		}
	}
}

// TestFinalJupiterHopSkipsEventCPI: a Jupiter route_v2 hop was matched to the
// first inner instruction of the hop's AMM program, so two consecutive hops
// through one event_cpi AMM (Meteora DLMM) gave the second hop the first
// hop's Anchor event self-CPI: no pool, no fees. Truth: the inner
// instruction listing (the DLMM swap2 at stack height 2 on the named pool).
// amm-1.
func TestFinalJupiterHopSkipsEventCPI(t *testing.T) {
	eventPrefix := []byte{0xe4, 0x45, 0xa5, 0x2e, 0x51, 0xcb, 0x9a, 0x1d}
	for _, c := range []struct {
		sig, idx, pool string
	}{
		{"33VPMZm2hGteXKCiwvsri7tZ2UzeXZpB3tiA2nzA1gn7oZTDyN7HogjF3SzoSzsVWEKUfSxtto1XkMzKNqvGToNi", "3-8", "4gMVHBCa1GNuofv67zvQCxZ5kyhh2kJa7EiFk4z57hwi"},
		{"xuVHmb58LEQsEQwSGzHqr2otgvyTDmijdjnQntirfg1MqLNe66EdiyoF2SaK5eHeRGnM5QAks1Qs5mdQpH4Myok", "2-8", "GgoFghiA6Ca7YNmG6dRxwKQWkZhRK8ND47EREz46arot"},
	} {
		tx, res := parseFixture(t, c.sig, nil)
		ixs := map[string]rawIx{}
		for _, ix := range fixtureIxs(t, tx) {
			if ix.inner >= 0 {
				ixs[fmtIdx(ix.outer, ix.inner)] = ix
			}
		}
		// the expected hop: a DLMM instruction on the pool, not an event
		want := ixs[c.idx]
		if want.programId != constants.DEX_PROGRAMS.METEORA.ID || string(want.data[:8]) == string(eventPrefix) || !containsStr(want.accounts, c.pool) {
			t.Fatalf("%s: %s is not a DLMM swap on %s", c.sig[:8], c.idx, c.pool)
		}
		found := false
		for _, tr := range res.Trades {
			if tr.Route != constants.DEX_PROGRAMS.JUPITER.Name {
				continue
			}
			ix, ok := ixs[tr.Idx]
			if !ok {
				continue
			}
			if string(ix.data[:8]) == string(eventPrefix) {
				t.Errorf("%s: hop %s (%s) names an Anchor event self-CPI", c.sig[:8], tr.Idx, tr.AMM)
			}
			if h := jsonInt(ix.m["stackHeight"]); h != 2 {
				t.Errorf("%s: hop %s (%s) at stack height %d, want 2 (invoked by the route)", c.sig[:8], tr.Idx, tr.AMM, h)
			}
			if tr.Idx == c.idx {
				found = true
				if len(tr.Pool) != 1 || tr.Pool[0] != c.pool || (tr.Fee == nil && len(tr.Fees) == 0) {
					t.Errorf("%s: hop %s pool %v fee %v fees %d, want pool %s with the DLMM fees", c.sig[:8], tr.Idx, tr.Pool, tr.Fee, len(tr.Fees), c.pool)
				}
			}
		}
		if !found {
			t.Errorf("%s: no Jupiter hop at %s", c.sig[:8], c.idx)
		}
	}
}

func fmtIdx(outer, inner int) string {
	if inner < 0 {
		return u64str(uint64(outer))
	}
	return u64str(uint64(outer)) + "-" + u64str(uint64(inner))
}

// clmmLiquidityAt returns the RaydiumCL liquidity event of DexParser at idx
func clmmLiquidityAt(t *testing.T, tx *adapterTx, idx string) *types.PoolEvent {
	t.Helper()
	res := dexparser.NewDexParser().ParseAll(tx, nil)
	var found *types.PoolEvent
	for i := range res.Liquidities {
		if res.Liquidities[i].ProgramId == constants.DEX_PROGRAMS.RAYDIUM_CL.ID && res.Liquidities[i].Idx == idx {
			if found != nil {
				t.Fatalf("two RaydiumCL events at %s", idx)
			}
			found = &res.Liquidities[i]
		}
	}
	return found
}

// checkCLMMSides checks an event's two sides against the expected mint ->
// raw amount map, and that token1 is the quote mint (SOL)
func checkCLMMSides(t *testing.T, name string, ev *types.PoolEvent, want map[string]string) {
	t.Helper()
	if ev == nil {
		t.Errorf("%s: no RaydiumCL liquidity event", name)
		return
	}
	got := map[string]string{ev.Token0Mint: ev.Token0AmountRaw, ev.Token1Mint: ev.Token1AmountRaw}
	for mint, amount := range want {
		if got[mint] != amount {
			t.Errorf("%s: %s %s, want %s (event %s %s:%s / %s:%s)", name, mint, got[mint], amount, ev.Type, ev.Token0Mint, ev.Token0AmountRaw, ev.Token1Mint, ev.Token1AmountRaw)
		}
	}
	decimals := func(d *uint8) int {
		if d == nil {
			return -1
		}
		return int(*d)
	}
	if ev.Token1Mint != solMint || decimals(ev.Token0Decimals) != 6 || decimals(ev.Token1Decimals) != 9 {
		t.Errorf("%s: token1 %s, decimals %d/%d: want token0 USDC (6), token1 SOL (9)", name, ev.Token1Mint, decimals(ev.Token0Decimals), decimals(ev.Token1Decimals))
	}
}

// TestFinalCLMMOneSidedRemove: a decrease_liquidity_v2 of an out-of-range
// position withdraws one token only; the other side reported the
// instruction's amount_1_min (a slippage limit) with an empty mint and 0
// decimals, or put SOL in token0. Truth: the pool vaults' balance changes
// (vault_0/vault_1 at accounts 5/6, mints at 14/15). amm-2.
func TestFinalCLMMOneSidedRemove(t *testing.T) {
	for _, sig := range []string{
		"25VouSoGsjj7dPr451fPhpNcsCHpyDd99nP5WCuvXgwW3S9deLUz6TkPNPhj29NwjVUihgrPfQkXUst16H1eQDbx", // USDC only
		"3i2HiLrPv8o8gyw8ezntj1Kg42LxtqiE1tMXXZJu6nkptFT9H7o8t9MLZ4hXVDDGT2NLbpXhniJ1ooMbVJyweHdv", // SOL only
	} {
		tx := loadFixture(t, sig)
		ix := findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_CL.ID, constants.DISCRIMINATORS.RAYDIUM_CL.REMOVE_LIQUIDITY.DECREASE_LIQUIDITY_V2)
		want := map[string]string{}
		moved := 0
		for side := 0; side < 2; side++ {
			vault, mint := ix.accounts[5+side], ix.accounts[14+side]
			delta := accountTokenDelta(tx, vault, mint)
			want[mint] = delta.Neg(delta).String()
			if want[mint] != "0" {
				moved++
			}
		}
		if moved != 1 {
			t.Fatalf("%s: %d vaults moved, want a one-sided withdrawal", sig[:8], moved)
		}
		checkCLMMSides(t, sig[:8], clmmLiquidityAt(t, tx, fmtIdx(ix.outer, ix.inner)), want)
	}
}

// dropInnerTransferTo returns a copy of tx without the inner token transfer
// into account (as if the program had skipped that side)
func dropInnerTransferTo(t *testing.T, tx *adapterTx, outer int, account string) *adapterTx {
	t.Helper()
	c := cloneTx(t, tx)
	keys := rawAccountKeys(c)
	for s := range c.Meta.InnerInstructions {
		set := &c.Meta.InnerInstructions[s]
		if set.Index != outer {
			continue
		}
		for j, v := range set.Instructions {
			m, _ := v.(map[string]interface{})
			if m == nil || keys[jsonInt(m["programIdIndex"])] != constants.TOKEN_PROGRAM_ID {
				continue
			}
			accounts, _ := m["accounts"].([]interface{})
			// transferChecked: source, mint, destination, authority
			if len(accounts) == 4 && keys[jsonInt(accounts[2])] == account {
				set.Instructions = append(set.Instructions[:j:j], set.Instructions[j+1:]...)
				return c
			}
		}
	}
	t.Fatalf("no inner transfer to %s in %d", account, outer)
	return nil
}

// TestFinalCLMMOneSidedAdd: a position out of range deposits one token only
// (raydium-clmm util/token.rs skips a zero transfer). increase_liquidity(_v2)
// with one token transfer was dropped (ADD needed two transfers), and
// open_position_with_token22_nft, where the position NFT mintTo counted as
// the second transfer, reported amount_1_max as the other side's deposit with
// an empty mint. No real one-sided add is in the corpus: both transactions
// are built from real two-sided adds (2haSYLTf increase_liquidity_v2, 2A7e9G6Q
// open_position_with_token22_nft) by removing the transfer into the SOL
// vault. Truth: the remaining USDC transfer and 0 SOL. amm-6, amm-v1.
func TestFinalCLMMOneSidedAdd(t *testing.T) {
	for _, c := range []struct {
		sig          string
		disc         []byte
		vault0, mint int // SOL vault and USDC mint account of the instruction
		usdcVault    int
	}{
		{"2haSYLTfhkEwfiZ4fqnzoq6deznDV5oHBkvxYNFc7JJ8bEjsunjazGWXVjcJ7MUaTVeS4Jq2CXit4Fakub6g8cTD", constants.DISCRIMINATORS.RAYDIUM_CL.ADD_LIQUIDITY.INCREASE_LIQUIDITY_V2, 9, 14, 10},
		{"2A7e9G6Qb91Z1kFhgsSr2p2NvZjRBREuziqmZ46RdQ6NGnHUhzVuBnc962AL9QsvpFRxUpaZVSbgK7scmCimRNhb", constants.DISCRIMINATORS.RAYDIUM_CL.ADD_LIQUIDITY.OPEN_POSITION_WITH_TOKEN22, 11, 19, 12},
	} {
		tx := loadFixture(t, c.sig)
		ix := findIx(t, tx, constants.DEX_PROGRAMS.RAYDIUM_CL.ID, c.disc)
		if ix.accounts[c.mint] != usdcMint || tokenBalanceMint(tx, ix.accounts[c.vault0]) != solMint {
			t.Fatalf("%s: unexpected pool layout", c.sig[:8])
		}
		idx := fmtIdx(ix.outer, ix.inner)
		usdc := accountTokenDelta(tx, ix.accounts[c.usdcVault], usdcMint).String()
		sol := accountTokenDelta(tx, ix.accounts[c.vault0], solMint).String()

		// the real two-sided add is unchanged
		checkCLMMSides(t, c.sig[:8]+" two-sided", clmmLiquidityAt(t, tx, idx), map[string]string{usdcMint: usdc, solMint: sol})

		oneSided := dropInnerTransferTo(t, tx, ix.outer, ix.accounts[c.vault0])
		checkCLMMSides(t, c.sig[:8]+" one-sided", clmmLiquidityAt(t, oneSided, idx), map[string]string{usdcMint: usdc, solMint: "0"})
	}
}

// dlmmReserveFlows sums the inner transferChecked amounts paid out of (or,
// for deposits, into) the reserve accounts of the outer DLMM instruction at
// outer, by mint, from the raw fixture
func dlmmReserveFlows(t *testing.T, tx *adapterTx, outer int, reserves map[string]bool) map[string]string {
	t.Helper()
	sums := map[string]uint64{}
	for _, ix := range fixtureIxs(t, tx) {
		if ix.outer != outer || ix.inner < 0 || ix.programId != constants.TOKEN_PROGRAM_ID || len(ix.data) != 10 || ix.data[0] != 12 {
			continue
		}
		// transferChecked: source, mint, destination, authority
		if reserves[ix.accounts[0]] || reserves[ix.accounts[2]] {
			sums[ix.accounts[1]] += le64At(ix.data, 1)
		}
	}
	out := map[string]string{}
	for mint, v := range sums {
		out[mint] = u64str(v)
	}
	return out
}

// TestFinalDLMMLiquiditySides: DLMM liquidity events put the quote mint in
// token0 or token1 depending on how many tokens moved (flipped between two
// removes of one pool in one transaction; a one-side add had USDC as token0
// and an empty token1). Truth: the IDL accounts (reserve_x/y 5/6,
// token_x/y_mint 7/8; one-side adds: reserve 4, token_mint 5) and the
// transfers out of or into the reserves; the quote mint is token1. amm-4.
func TestFinalDLMMLiquiditySides(t *testing.T) {
	for _, c := range []struct {
		sig      string
		outer    int
		oneSide  bool
		wantType types.PoolEventType
	}{
		{"2mNBApzw9HXF3TJXsLoBtSxc3DqC6XVYbtwchmyUA9mvt9oVamYzpHa9axF62y9aNnXMzyKTzpfCcsHNsLgUcSnE", 0, false, types.PoolEventTypeRemove},
		{"2mNBApzw9HXF3TJXsLoBtSxc3DqC6XVYbtwchmyUA9mvt9oVamYzpHa9axF62y9aNnXMzyKTzpfCcsHNsLgUcSnE", 1, false, types.PoolEventTypeRemove},
		{"4HAaBUNWYGQZsEf1VJAvzAtF6sLL8pb7CEMeBxNdivrWwP6XhNVNBH446NpoVtVyFoGuT1bu8jSpVHa8xe1yXtf", 2, true, types.PoolEventTypeAdd},
	} {
		tx, res := parseFixture(t, c.sig, nil)
		var ix rawIx
		for _, x := range fixtureIxs(t, tx) {
			if x.programId == constants.DEX_PROGRAMS.METEORA.ID && x.outer == c.outer && x.inner < 0 {
				ix = x
			}
		}
		var mints []string
		reserves := map[string]bool{}
		if c.oneSide {
			mints, reserves[ix.accounts[4]] = []string{ix.accounts[5]}, true
		} else {
			mints, reserves[ix.accounts[5]], reserves[ix.accounts[6]] = []string{ix.accounts[7], ix.accounts[8]}, true, true
		}
		flows := dlmmReserveFlows(t, tx, c.outer, reserves)
		var ev *types.PoolEvent
		for i := range res.Liquidities {
			if res.Liquidities[i].Idx == fmtIdx(c.outer, -1) {
				ev = &res.Liquidities[i]
			}
		}
		if ev == nil || ev.Type != c.wantType {
			t.Fatalf("%s %d: event %+v, want %s", c.sig[:8], c.outer, ev, c.wantType)
		}
		got := map[string]string{ev.Token0Mint: ev.Token0AmountRaw, ev.Token1Mint: ev.Token1AmountRaw}
		for _, mint := range mints {
			want := flows[mint]
			if want == "" {
				want = "0"
			}
			if got[mint] != want {
				t.Errorf("%s %d: %s %q, want %s (event %s:%s / %s:%s)", c.sig[:8], c.outer, mint, got[mint], want, ev.Token0Mint, ev.Token0AmountRaw, ev.Token1Mint, ev.Token1AmountRaw)
			}
		}
		// the quote mint of the pair is token1
		quote := ""
		for _, mint := range mints {
			if mint == solMint || (quote == "" && constants.IsQuoteToken(mint)) {
				quote = mint
			}
		}
		if ev.Token1Mint != quote {
			t.Errorf("%s %d: token1 %s, want the quote mint %s", c.sig[:8], c.outer, ev.Token1Mint, quote)
		}
	}
}
