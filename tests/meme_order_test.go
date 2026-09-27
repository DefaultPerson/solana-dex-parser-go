package tests

import (
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/meme"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/meteora"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/pumpfun"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// memeEventParsers builds every meme event parser for a transaction
func memeEventParsers(a *adapter.TransactionAdapter, t map[string][]types.TransferData) map[string]parsers.EventParser {
	return map[string]parsers.EventParser{
		"Pumpfun":   pumpfun.NewPumpfunEventParser(a, t),
		"Pumpswap":  pumpfun.NewPumpswapEventParser(a, t),
		"DBC":       meteora.NewMeteoraDBCEventParser(a, t),
		"LaunchLab": raydium.NewRaydiumLaunchpadEventParser(a, t),
		"Boopfun":   meme.NewBoopfunEventParser(a, t),
		"Moonit":    meme.NewMoonitEventParser(a, t),
		"Heaven":    meme.NewHeavenEventParser(a, t),
		"Sugar":     meme.NewSugarEventParser(a, t),
	}
}

// meme-19: the meme event parsers sorted idx as strings ("10-5" < "2-22");
// they now return events in execution order. Every fixture, every parser.
func TestMemeEventParsersExecutionOrder(t *testing.T) {
	for _, sig := range fixtureSignatures(t, "json") {
		ctx := newParseContext(loadFixture(t, sig), nil)
		for name, p := range memeEventParsers(ctx.Adapter, ctx.TransferActions) {
			events := p.ProcessEvents()
			for i := 1; i < len(events); i++ {
				if utils.CompareIdx(events[i-1].Idx, events[i].Idx) > 0 {
					t.Errorf("%.8s %s: %s before %s", sig, name, events[i-1].Idx, events[i].Idx)
				}
			}
		}
	}
}

// meme-19 with an index that sorts differently as a string. Synthetic: the
// real 5zMo16 (CreateEvent at 2-22, TradeEvent at 4-5) with the inner
// instructions of outer 2 relabelled as outer 10, so the events are 4-5 and
// 10-22 ("10-22" < "4-5" as strings). No fixture has two decoded meme events
// of one parser whose string order differs.
func TestMemePumpfunEventOrderNumeric(t *testing.T) {
	tx := cloneTx(t, loadFixture(t, sigPumpCreateV2BuyV2))
	for i := range tx.Meta.InnerInstructions {
		if tx.Meta.InnerInstructions[i].Index == 2 {
			tx.Meta.InnerInstructions[i].Index = 10
		}
	}
	ctx := newParseContext(tx, nil)
	events := pumpfun.NewPumpfunEventParser(ctx.Adapter, ctx.TransferActions).ProcessEvents()
	if len(events) != 2 || events[0].Idx != "4-5" || events[1].Idx != "10-22" {
		var got []string
		for _, e := range events {
			got = append(got, e.Idx)
		}
		t.Errorf("event order %v, want [4-5 10-22]", got)
	}
}
