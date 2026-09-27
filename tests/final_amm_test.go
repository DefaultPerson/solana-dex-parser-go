package tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/mr-tron/base58"

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
