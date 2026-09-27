package tests

import (
	"strings"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// Regression tests for instruction stack heights (streamer item 12).

// TestCore2StackHeight: the adapter dropped stackHeight, so CPI events could
// not be matched to the instruction that emitted them. streamer item 12.
// 28Durhr routes through Jupiter into PumpSwap; each PumpSwap event (self-CPI
// with the e445a52e51cb9a1d prefix) has the PumpSwap swap (buy_exact_quote_in)
// as parent.
func TestCore2StackHeight(t *testing.T) {
	const sig = "28DurhrCQrkkrM6MeJki2yQr76Hfx2pUMRZnzzjbfXJSW17PaZVbBMs7LpxdNt56FbxtJuSgyHJVbDxRYYtFKYJ1"
	tx := loadFixture(t, sig)
	a := adapter.NewTransactionAdapter(tx, nil)
	c := classifier.NewInstructionClassifier(a)
	keys := rawAccountKeys(tx)

	// ClassifiedInstruction.StackHeight equals the raw value (outer: 1)
	raw := map[[2]int]int{}
	for _, set := range tx.Meta.InnerInstructions {
		for j, ix := range set.Instructions {
			raw[[2]int{set.Index, j}] = jsonInt(ix.(map[string]interface{})["stackHeight"])
		}
	}
	for _, id := range append(c.GetAllProgramIds(), constants.TOKEN_PROGRAM_ID) {
		for _, ci := range c.GetInstructions(id) {
			want := 1
			if ci.InnerIndex >= 0 {
				want = raw[[2]int{ci.OuterIndex, ci.InnerIndex}]
			}
			if ci.StackHeight != want || want == 0 {
				t.Errorf("%s %s: StackHeight %d, want %d", id[:6], ci.GetIdx(), ci.StackHeight, want)
			}
		}
	}

	pumpSwap := constants.DEX_PROGRAMS.PUMP_SWAP.ID
	events := 0
	for _, ci := range c.GetInstructions(pumpSwap) {
		data := a.GetInstructionData(ci.Instruction)
		if ci.InnerIndex < 0 || len(data) < 16 || !strings.HasPrefix(string(data), string([]byte{0xe4, 0x45, 0xa5, 0x2e, 0x51, 0xcb, 0x9a, 0x1d})) {
			continue
		}
		events++
		parent, parentInner, ok := a.GetParentInstruction(ci.OuterIndex, ci.InnerIndex)
		if !ok || parentInner < 0 || parentInner >= ci.InnerIndex {
			t.Fatalf("event %s: parent %d ok=%v", ci.GetIdx(), parentInner, ok)
		}
		pm := parent.(map[string]interface{})
		pdata := a.GetInstructionData(parent)
		if keys[jsonInt(pm["programIdIndex"])] != pumpSwap || jsonInt(pm["stackHeight"]) != ci.StackHeight-1 {
			t.Errorf("event %s: parent program %s height %v", ci.GetIdx(), keys[jsonInt(pm["programIdIndex"])], pm["stackHeight"])
		}
		swap := false
		for _, d := range [][]byte{constants.DISCRIMINATORS.PUMPSWAP.BUY, constants.DISCRIMINATORS.PUMPSWAP.SELL, constants.DISCRIMINATORS.PUMPSWAP.BUY_EXACT_QUOTE_IN} {
			swap = swap || (len(pdata) >= 8 && string(pdata[:8]) == string(d))
		}
		if !swap {
			t.Errorf("event %s: parent is not a PumpSwap swap", ci.GetIdx())
		}
		// the Jupiter instruction invoked the swap
		if _, gp, ok := a.GetParentInstruction(ci.OuterIndex, parentInner); !ok || gp != -1 {
			t.Errorf("event %s: grandparent %d ok=%v, want the outer Jupiter instruction", ci.GetIdx(), gp, ok)
		}
	}
	if events == 0 {
		t.Fatal("fixture: no PumpSwap CPI event")
	}
	if _, _, ok := a.GetParentInstruction(0, -1); ok {
		t.Error("outer instruction has a parent")
	}

	// Go-typed instructions carry the height; without it there is no parent
	typed := &adapter.SolanaTransaction{Meta: &adapter.TransactionMeta{InnerInstructions: []adapter.InnerInstructionSet{{Index: 0, Instructions: []interface{}{
		adapter.CompiledInstruction{StackHeight: 2},
		adapter.CompiledInstruction{StackHeight: 3},
		map[string]interface{}{"programIdIndex": 0},
	}}}}}
	typed.Transaction.Message.Instructions = []interface{}{adapter.CompiledInstruction{}}
	ta := adapter.NewTransactionAdapter(typed, nil)
	if _, pi, ok := ta.GetParentInstruction(0, 1); !ok || pi != 0 {
		t.Errorf("typed: parent %d ok=%v, want 0", pi, ok)
	}
	if _, _, ok := ta.GetParentInstruction(0, 2); ok {
		t.Error("instruction without stackHeight has a parent")
	}
}
