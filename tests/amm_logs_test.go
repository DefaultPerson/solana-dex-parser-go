package tests

import (
	"bytes"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// TestParseProgramLogsAttribution checks the invoke-stack attribution of the
// log decoder on hand-written log lines (the rules of the Solana runtime log
// format): a payload belongs to the program on top of the stack, CPIs count
// as the next inner instruction, a return pops the stack, and decoding stops
// at "Log truncated".
func TestParseProgramLogsAttribution(t *testing.T) {
	logs := []string{
		"Program A111 invoke [1]",
		"Program data: AQ==", // A, outer 0
		"Program B222 invoke [2]",
		"Program data: Ag==", // B, inner 0
		"Program C333 invoke [3]",
		"Program C333 success",
		"Program data: Aw==", // B again (C returned), inner 0
		"Program B222 consumed 100 of 200 compute units",
		"Program B222 success",
		"Program data: BA==", // A after the CPI returned: outer
		"Program D444 invoke [2]",
		"Program data: BQ==", // D, inner 2 (C was inner 1)
		"Program D444 failed: custom program error: 0x1",
		"Program A111 success",
		"Program E555 invoke [1]",
		"Program data: not base64!", // skipped
		"Program log: ray_log: Bg==",
		"Program data: Bw==", // E, outer 1
		"Log truncated",
		"Program data: CA==",
	}
	got := utils.ParseProgramDataLogs(logs, nil)
	want := []utils.ProgramLog{
		{ProgramId: "A111", OuterIndex: 0, InnerIndex: -1, Depth: 1, Data: []byte{1}},
		{ProgramId: "B222", OuterIndex: 0, InnerIndex: 0, Depth: 2, Data: []byte{2}},
		{ProgramId: "B222", OuterIndex: 0, InnerIndex: 0, Depth: 2, Data: []byte{3}},
		{ProgramId: "A111", OuterIndex: 0, InnerIndex: -1, Depth: 1, Data: []byte{4}},
		{ProgramId: "D444", OuterIndex: 0, InnerIndex: 2, Depth: 2, Data: []byte{5}},
		{ProgramId: "E555", OuterIndex: 1, InnerIndex: -1, Depth: 1, Data: []byte{7}},
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

	ray := utils.ParseRayLogs(logs, nil)
	if len(ray) != 1 || ray[0].ProgramId != "E555" || !bytes.Equal(ray[0].Data, []byte{6}) {
		t.Errorf("ray logs = %+v", ray)
	}

	// With the outer program ids, an outer instruction that writes no invoke
	// line does not shift the indices of the following ones
	aligned := utils.ParseProgramDataLogs([]string{"Program E555 invoke [1]", "Program data: AQ==", "Program E555 success"}, []string{"Pre111", "E555"})
	if len(aligned) != 1 || aligned[0].OuterIndex != 1 {
		t.Errorf("aligned = %+v, want outer 1", aligned)
	}
}

// TestProgramLogsOnFixtures attributes real log payloads: the Raydium CLMM
// PoolCreatedEvent of create_pool (outer 2) and the ray_log of a Raydium AMM
// v4 swap called by the Raydium route program (inner 0 of outer 3).
func TestProgramLogsOnFixtures(t *testing.T) {
	ctx := newParseContext(loadFixture(t, "PGYHMva3trxE1BR2Qr4H8DZDcSLeHdHcCgJT2iAarvrE9xovBu5dtrNiti3xXxUKoNgT9WeifFPA6zX8DjsASdP"), nil)
	var found bool
	for _, l := range ctx.Utils.GetProgramDataLogs() {
		if constants.MatchDiscriminator(l.Data, constants.DISCRIMINATORS.RAYDIUM_CL.EVENTS.POOL_CREATED) {
			found = true
			if l.ProgramId != constants.DEX_PROGRAMS.RAYDIUM_CL.ID || l.OuterIndex != 2 || l.InnerIndex != -1 {
				t.Errorf("PoolCreatedEvent attributed to %+v, want Raydium CL outer 2", l)
			}
		}
	}
	if !found {
		t.Error("no PoolCreatedEvent found")
	}

	ctx = newParseContext(loadFixture(t, "2iHYs4AHC5nutcbBxpA5aptBYTGaDUYBgamohfetDnAPiPBW5NkguxgnjVF5886Jy8MZ19UXdeZyPKq9C5wqAki4"), nil)
	ray := ctx.Utils.GetRayLogs()
	if len(ray) != 1 || ray[0].ProgramId != constants.DEX_PROGRAMS.RAYDIUM_V4.ID || ray[0].OuterIndex != 3 || ray[0].InnerIndex != 0 || ray[0].Depth != 2 {
		t.Errorf("ray_log attribution = %+v, want Raydium V4 at 3-0", ray)
	}
}
