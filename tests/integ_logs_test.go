package tests

import (
	"bytes"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/propamm"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Program-written lines never open or close an invoke frame, whatever their
// text: only the runtime's "Program <id> invoke [n]", "Program <id> success"
// and "Program <id> failed: ..." lines do. Before the fix "Program log:
// invoke [2]" opened a frame for a program "log:", so the payload after it
// was attributed to "log:" at inner 0 and the real CPI shifted to inner 1.
func TestIntegProgramLogsIgnoreProgramWrittenLines(t *testing.T) {
	logs := []string{
		"Program A111 invoke [1]",
		"Program log: invoke [2]",
		"Program data: AQ==", // still A, outer 0
		"Program log: success",
		"Program log: failed to refresh quote, using cached",
		"Program return: A111 AQ==",
		"Program consumption: 1000 units remaining",
		"Program B222 invoke [2]",
		"Program data: Ag==", // B, inner 0
		"Program log: B222 success",
		"Program data: Aw==", // still B: its return line was written by a program
		"Program B222 success",
		"Program data: BA==", // A again
		"Program A111 success",
	}
	got := utils.ParseProgramDataLogs(logs, nil)
	want := []utils.ProgramLog{
		{ProgramId: "A111", OuterIndex: 0, InnerIndex: -1, Depth: 1, Data: []byte{1}},
		{ProgramId: "B222", OuterIndex: 0, InnerIndex: 0, Depth: 2, Data: []byte{2}},
		{ProgramId: "B222", OuterIndex: 0, InnerIndex: 0, Depth: 2, Data: []byte{3}},
		{ProgramId: "A111", OuterIndex: 0, InnerIndex: -1, Depth: 1, Data: []byte{4}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d logs %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.ProgramId != w.ProgramId || g.OuterIndex != w.OuterIndex || g.InnerIndex != w.InnerIndex || g.Depth != w.Depth || !bytes.Equal(g.Data, w.Data) {
			t.Errorf("log %d = %+v, want %+v", i, g, w)
		}
	}
}

// venues-logs-1: the Titan swap event survives program-written lines that
// contain "success" or "failed". propamm's own decoder read "Program log:
// success" as a return of a program "log:", gave up on the logs and dropped
// the route trade. Synthetic: the real Titan route 4C2p65nu... with those
// lines injected inside the SolFi V2 hop.
func TestIntegTitanEventSurvivesProgramLogLines(t *testing.T) {
	for _, extra := range []string{
		"Program log: success",
		"Program log: failed to refresh quote, using cached",
		"Program log: invoke [3]",
	} {
		t.Run(extra, func(t *testing.T) {
			tx := loadFixture(t, "4C2p65nuttUqBv5VhLNXHq6cNEaexT5i4YGbGSH2hodRGLBQyJBzRCkr2VXPKPNof8Lut8WSm51tkxVv3mvUkpuZ")
			logs := tx.Meta.LogMessages
			if len(logs) < 24 || logs[11] != "Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [3]" {
				t.Fatalf("unexpected fixture logs")
			}
			tx.Meta.LogMessages = append(append(append([]string{}, logs[:12]...), extra), logs[12:]...)
			ctx := newParseContext(tx, nil)
			trades := propamm.NewTitanParser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions,
				ctx.Classifier.GetInstructions(constants.DEX_PROGRAMS.TITAN.ID)).ProcessTrades()
			if len(trades) != 1 {
				t.Fatalf("want 1 route trade, got %d", len(trades))
			}
			if tr := trades[0]; tr.InputToken.AmountRaw != "1000011964" || tr.OutputToken.AmountRaw != "999797125" {
				t.Errorf("route %s -> %s, want 1000011964 -> 999797125", tr.InputToken.AmountRaw, tr.OutputToken.AmountRaw)
			}
		})
	}
}

// The transaction's own decoder checks every CPI invoke line against the
// recorded inner instructions and stops at the first that does not match,
// instead of attributing the following payloads to the wrong instruction.
// Synthetic: the real Raydium route swap 2iHYs4AH... (ray_log of the AMM v4
// CPI at 3-0) with an invoke/success pair of a program that is not among
// its inner instructions inserted before the AMM v4 invoke. Before the
// check the ray_log was attributed to inner 1, an instruction of another
// program.
func TestIntegProgramLogsCheckInnerInstructions(t *testing.T) {
	const sig = "2iHYs4AHC5nutcbBxpA5aptBYTGaDUYBgamohfetDnAPiPBW5NkguxgnjVF5886Jy8MZ19UXdeZyPKq9C5wqAki4"
	ctx := newParseContext(loadFixture(t, sig), nil)
	if ray := ctx.Utils.GetRayLogs(); len(ray) != 1 || ray[0].OuterIndex != 3 || ray[0].InnerIndex != 0 {
		t.Fatalf("unmodified: ray logs %+v, want one at 3-0", ray)
	}

	tx := loadFixture(t, sig)
	logs := tx.Meta.LogMessages
	if len(logs) < 24 || logs[23] != "Program 675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8 invoke [2]" {
		t.Fatalf("unexpected fixture logs")
	}
	stray := []string{"Program MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr invoke [2]", "Program MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr success"}
	tx.Meta.LogMessages = append(append(append([]string{}, logs[:23]...), stray...), logs[23:]...)
	ctx = newParseContext(tx, nil)
	for _, l := range ctx.Utils.GetRayLogs() {
		if l.OuterIndex == 3 && l.InnerIndex != 0 {
			t.Errorf("ray_log attributed to 3-%d (%s), an instruction that did not write it", l.InnerIndex, l.ProgramId)
		}
	}
}
