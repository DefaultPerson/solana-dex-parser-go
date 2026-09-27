package tests

import (
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// rawTokenTransfer is an SPL Token / Token-2022 Transfer or TransferChecked
// inner instruction decoded straight from the fixture JSON
type rawTokenTransfer struct {
	inner, height                  int
	source, destination, authority string
	amount                         string
}

// rawInnerTransfers decodes the token transfers among the inner
// instructions of outer
func rawInnerTransfers(t *testing.T, tx *adapter.SolanaTransaction, outer int) []rawTokenTransfer {
	t.Helper()
	keys := rawAccountKeys(tx)
	var out []rawTokenTransfer
	for _, set := range tx.Meta.InnerInstructions {
		if set.Index != outer {
			continue
		}
		for j, ix := range set.Instructions {
			m := ix.(map[string]interface{})
			pid, _ := strconv.Atoi(stringOf(m["programIdIndex"]))
			if pid >= len(keys) || (keys[pid] != constants.TOKEN_PROGRAM_ID && keys[pid] != constants.TOKEN_2022_PROGRAM_ID) {
				continue
			}
			data, err := base58.Decode(m["data"].(string))
			if err != nil || len(data) < 9 {
				continue
			}
			var acc []int
			for _, a := range m["accounts"].([]interface{}) {
				n, _ := strconv.Atoi(stringOf(a))
				acc = append(acc, n)
			}
			tr := rawTokenTransfer{inner: j, height: adapter.InstructionStackHeight(m), amount: strconv.FormatUint(binary.LittleEndian.Uint64(data[1:9]), 10)}
			switch {
			case data[0] == 3 && len(acc) >= 3:
				tr.source, tr.destination, tr.authority = keys[acc[0]], keys[acc[1]], keys[acc[2]]
			case data[0] == 12 && len(acc) >= 4:
				tr.source, tr.destination, tr.authority = keys[acc[0]], keys[acc[2]], keys[acc[3]]
			default:
				continue
			}
			out = append(out, tr)
		}
	}
	return out
}

func stringOf(v interface{}) string {
	switch n := v.(type) {
	case float64:
		return strconv.Itoa(int(n))
	case interface{ String() string }:
		return n.String()
	}
	return ""
}

// amm request / item 9: a trade leg takes the transfer made inside the
// trade's own instruction. In a route the same amount also moves in the
// previous hop (its output into the user's intermediate account), and the
// whole-transaction search gave a hop's input leg that transfer. Checked on
// the raw fixtures: the input leg's transfer (source, destination,
// authority, amount) must be an inner instruction of the hop's CPI subtree
// (stack height greater than the hop's, before the next instruction at its
// height).
func TestIntegTradeLegFromOwnInstruction(t *testing.T) {
	cases := []struct {
		sig  string
		idxs []string
	}{
		{"34sGGDUK4A1xDzYKXLYz8gsU5gF76yz7GcQBxd17AcDMEwbdYL3bukiCvRMcC7sdo3HUR3bNtWxav2ufExsLunJB", []string{"5-1", "5-4", "5-7"}},
		{"5qJs7ws4UY3qmtPZf5R2LbBQWjkUNEyzdC6gAXssYRYMwWoo1qBQ61Mk8owtXYUSPouLY2z4i4wnohdKCyxgtrkS", []string{"5-3", "5-7"}},
		{"zuaKyxjpM7G5et2XqZofjjGNczNduGs6g8ipCEeZKKV7h6FFgRNJbXnzfufSZWD3bEacmf8sVktXpZaadQhmVuJ", []string{"5-5", "5-8"}},
		{"NeF1UiWXKUbuswNNw14gJ2uup7yrV6KyXijQ7dsjLYanj9dnsBSeqGPDVZ3wP3NfhXQ84rJncRo5XwbrbSdWVWW", []string{"3-8"}},
	}
	for _, tc := range cases {
		t.Run(tc.sig[:8], func(t *testing.T) {
			tx := loadFixture(t, tc.sig)
			result := dexparser.NewDexParser().ParseAll(tx, nil)
			for _, idx := range tc.idxs {
				var found bool
				for _, tr := range result.Trades {
					if tr.Idx != idx {
						continue
					}
					found = true
					outer, inner := utils.SplitIdx(idx)
					transfers := rawInnerTransfers(t, tx, outer)
					hopHeight := 0
					for _, set := range tx.Meta.InnerInstructions {
						if set.Index == outer {
							hopHeight = adapter.InstructionStackHeight(set.Instructions[inner])
						}
					}
					var leg *rawTokenTransfer
					for i := range transfers {
						r := &transfers[i]
						if r.source == tr.InputToken.Source && r.destination == tr.InputToken.Destination &&
							r.authority == tr.InputToken.Authority && r.amount == tr.InputToken.AmountRaw {
							leg = r
							break
						}
					}
					if leg == nil {
						t.Errorf("%s: input leg %s -> %s (%s) is no transfer of outer %d", idx, tr.InputToken.Source, tr.InputToken.Destination, tr.InputToken.AmountRaw, outer)
						continue
					}
					// inside the subtree: after the hop, deeper, and no
					// instruction at the hop's height or above in between
					inside := leg.inner > inner && leg.height > hopHeight
					for _, set := range tx.Meta.InnerInstructions {
						if set.Index != outer {
							continue
						}
						for j := inner + 1; inside && j < leg.inner; j++ {
							if adapter.InstructionStackHeight(set.Instructions[j]) <= hopHeight {
								inside = false
							}
						}
					}
					if !inside {
						t.Errorf("%s %s: input leg is transfer %d-%d, outside the hop's instruction", idx, tr.AMM, outer, leg.inner)
					}
				}
				if !found {
					t.Errorf("no trade at %s", idx)
				}
			}
		})
	}
}
