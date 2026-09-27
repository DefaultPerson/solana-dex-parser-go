package tests

import (
	"reflect"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// insertBetweenEmitterAndEvent returns a copy of the LaunchLab sell
// 5UrQjjQ1... (sell at 1-0, stack height 2; its TradeEvent self-CPI at 1-1,
// height 3) with two inner instructions inserted before the event: a Token
// program call at height 3 invoked by the sell, and a copy of the sell at
// height 4 invoked by that call. The event moves to 1-3; its parent is
// still the sell at 1-0.
func insertBetweenEmitterAndEvent(t *testing.T) (orig, modified *types.MemeEvent) {
	t.Helper()
	tx := loadFixture(t, sigLaunchLabSOLSell)
	ctx := newParseContext(tx, nil)
	for _, e := range raydium.NewRaydiumLaunchpadEventParser(ctx.Adapter, ctx.TransferActions).ProcessEvents() {
		if e.Idx == "1-0" {
			e := e
			orig = &e
		}
	}

	tx = cloneTx(t, tx)
	set := &tx.Meta.InnerInstructions[1]
	if set.Index != 1 || len(set.Instructions) < 3 {
		t.Fatalf("unexpected fixture layout")
	}
	sell := set.Instructions[0].(map[string]interface{})
	tokenCall := set.Instructions[2].(map[string]interface{}) // a Token transfer at height 3
	nested := map[string]interface{}{"stackHeight": float64(4)}
	for k, v := range sell {
		if k != "stackHeight" {
			nested[k] = v
		}
	}
	call := map[string]interface{}{"stackHeight": float64(3)}
	for k, v := range tokenCall {
		if k != "stackHeight" {
			call[k] = v
		}
	}
	rest := append([]interface{}{call, nested}, set.Instructions[1:]...)
	set.Instructions = append([]interface{}{set.Instructions[0]}, rest...)

	ctx = newParseContext(tx, nil)
	for _, e := range raydium.NewRaydiumLaunchpadEventParser(ctx.Adapter, ctx.TransferActions).ProcessEvents() {
		if e.Idx == "1-0" {
			e := e
			modified = &e
		}
	}
	return orig, modified
}

// With stack heights, a self-CPI event belongs to its parent instruction.
// The previous matching took the nearest preceding instruction of the
// program (or stopped at the next one), so an instruction of the same
// program nested deeper between the emitter and its event captured the
// event. Synthetic, see insertBetweenEmitterAndEvent.
func TestIntegEventEmitterIsStackParent(t *testing.T) {
	orig, modified := insertBetweenEmitterAndEvent(t)
	if orig == nil || orig.Type != types.TradeTypeSell || orig.InputToken == nil || orig.OutputToken == nil ||
		orig.VirtualBaseReserves == "" || len(orig.Fees) == 0 {
		t.Fatalf("unmodified sell event %+v", orig)
	}
	if modified == nil || modified.InputToken == nil || modified.OutputToken == nil {
		t.Fatalf("sell event at 1-0 after the insertion: %+v", modified)
	}
	if modified.InputToken.AmountRaw != orig.InputToken.AmountRaw || modified.OutputToken.AmountRaw != orig.OutputToken.AmountRaw ||
		modified.VirtualBaseReserves != orig.VirtualBaseReserves || !reflect.DeepEqual(modified.Fees, orig.Fees) {
		t.Errorf("sell at 1-0: %s -> %s, reserves %s, fees %+v; want %s -> %s, %s, %+v (from its TradeEvent)",
			modified.InputToken.AmountRaw, modified.OutputToken.AmountRaw, modified.VirtualBaseReserves, modified.Fees,
			orig.InputToken.AmountRaw, orig.OutputToken.AmountRaw, orig.VirtualBaseReserves, orig.Fees)
	}
}

// utils.FindEventEmitter: the event's stack parent when heights are known,
// else the nearest preceding non-event instruction of the event's program
// that matches.
func TestIntegFindEventEmitter(t *testing.T) {
	tx := loadFixture(t, sigLaunchLabSOLSell)
	ctx := newParseContext(tx, nil)
	lcp := ctx.Classifier.GetInstructions(constants.DEX_PROGRAMS.RAYDIUM_LCP.ID)
	var sell, event types.ClassifiedInstruction
	for _, ci := range lcp {
		switch {
		case ci.OuterIndex == 1 && ci.InnerIndex == 0:
			sell = ci
		case ci.OuterIndex == 1 && ci.InnerIndex == 1:
			event = ci
		}
	}
	if !constants.IsAnchorEvent(ctx.Adapter.GetInstructionData(event.Instruction)) {
		t.Fatalf("1-1 is not an event")
	}
	if e := utils.FindEventEmitter(ctx.Adapter, lcp, event, nil); e == nil || e.InnerIndex != 0 {
		t.Errorf("emitter %+v, want 1-0", e)
	}
	if e := utils.FindEventEmitter(ctx.Adapter, lcp, event, func([]byte, []string) bool { return false }); e != nil {
		t.Errorf("rejected emitter returned: %+v", e)
	}
	if ev := utils.EmittedEvents(ctx.Adapter, lcp, sell); len(ev) != 1 || ev[0].InnerIndex != 1 {
		t.Errorf("events of the sell: %+v, want 1-1", ev)
	}

	// Without stack heights (synthetic: heights removed), the nearest
	// preceding instruction of the program
	tx = cloneTx(t, tx)
	for _, set := range tx.Meta.InnerInstructions {
		for _, ix := range set.Instructions {
			delete(ix.(map[string]interface{}), "stackHeight")
		}
	}
	ctx = newParseContext(tx, nil)
	lcp = ctx.Classifier.GetInstructions(constants.DEX_PROGRAMS.RAYDIUM_LCP.ID)
	for _, ci := range lcp {
		if ci.OuterIndex == 1 && ci.InnerIndex == 1 {
			if e := utils.FindEventEmitter(ctx.Adapter, lcp, ci, nil); e == nil || e.InnerIndex != 0 {
				t.Errorf("without heights: emitter %+v, want 1-0", e)
			}
		}
	}
}
