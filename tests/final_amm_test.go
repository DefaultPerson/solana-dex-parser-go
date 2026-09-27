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
