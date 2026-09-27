package tests

import (
	"strings"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// constDecodes32 reports whether s is a base58 string of a 32-byte public key.
func constDecodes32(s string) bool {
	b, err := base58.Decode(s)
	return err == nil && len(b) == 32
}

// All program IDs decode to 32 bytes, are unique and resolve through the lookup.
func TestConstantsProgramIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, id := range constants.DEX_PROGRAM_IDS {
		if !constDecodes32(id) {
			t.Errorf("program ID %q is not a 32-byte base58 key", id)
		}
		if seen[id] {
			t.Errorf("program ID %s listed twice", id)
		}
		seen[id] = true
		p := constants.GetDexProgramByID(id)
		if p.ID != id || p.Name == "" || len(p.Tags) == 0 {
			t.Errorf("GetDexProgramByID(%s) = %+v", id, p)
		}
		if !constants.IsDexProgram(id) {
			t.Errorf("IsDexProgram(%s) = false", id)
		}
	}
	for _, list := range [][]string{constants.SYSTEM_PROGRAMS, constants.SKIP_PROGRAM_IDS, constants.KNOWN_AUTHORITIES} {
		for _, id := range list {
			if !constDecodes32(id) {
				t.Errorf("%q is not a 32-byte base58 key", id)
			}
		}
	}
	t.Logf("programs: %d (of which %d from the Jupiter label list)", len(constants.DEX_PROGRAM_IDS), len(constants.JUPITER_LABEL_PROGRAMS))
}

// meme-9, constants-11, constants-13: new venues, aggregators and the GMGN bot program
// (IDs and names from the Jupiter label list, on-chain IDL metadata and audit transactions).
func TestConstantsNewPrograms(t *testing.T) {
	cases := []struct {
		id, name, tag string
	}{
		{"SV2EYYJyRz2YhfXwXnhNAevDEui5Q6yrfyo13WtupPF", "SolFiV2", "amm"},
		{"goonuddtQRrWqqn5nFyczVKaie28f3kDkHWkHtURSLE", "GoonFiV2", "amm"},
		{"BiSoNHVpsVZW2F7rx2eQ59yQwKxzU5NvBcmKshCSUypi", "BisonFi", "amm"},
		{"TessVdML9pBGgG9yGks7o4HewRaXVAMuoVj4x83GLQH", "TesseraV", "amm"},
		{"ALPHAQmeA7bjrVuccPsYPiCvsi428SNwte66Srvs4pHA", "AlphaQ", "amm"},
		{"ZERor4xhbUycZ6gb9ntrhqscUcZmAbQDjEAtCf4hbZY", "ZeroFi", "amm"},
		{"ojh19ojaKduoJZuaJADhcVGp4xt1TcdAvZmpVsCorch", "Scorch", "amm"},
		{"SCoRcH8c2dpjvcJD6FiPbCSQyQgu3PcUAWj2Xxx3mqn", "Scorch", "amm"},
		{"QuaNtZsgYRe5Z9Bk4LZ4cTD9tbkVoyCNf1R2BN9bBDv", "Quantum", "amm"},
		{"MNFSTqtC93rEfYHB6hF82sKdZpUDFWkViLByLd1k1Ms", "Manifest", "amm"},
		{"REALQqNEomY6cQGZJUGwywTBD2UmDT32rZcNnfxQ5N2", "Byreal", "amm"},
		{"1qbkdrr3z4ryLA7pZykqxvxWPoeifcVKo6ZG9CfkvVE", "SarosDLMM", "amm"},
		{"T1TANpTeScyeqVzzgNViGDNrkQ6qHz9KrSBS4aNXvGT", "Titan", "route"},
		{"proVF4pMXVaYqmy4NjniPh4pqKNfMmsihgd4wdkCX3u", "OKXV2", "route"},
		{"61DFfeTKM7trxYcPQCM78bJ794ddZprZpAwAnLiwTpYH", "JupiterZ", "route"},
		{"GMgnVFR8Jb39LoXsEVzb3DvBy3ywCmdmJquHUy1Lrkqb", "GMGN", "bot"},
		// from the Jupiter label list without a named field
		{"2DNbzPochEcyCcWMbL4d9S3u9QqQEj5bbe6cSZFvKsbh", "BisonFi Predict", "amm"},
		{"Archer8kgiavM61GyusMzaaS2ft5sALtNsD1HxkUPMhy", "Archer", "amm"},
		{"FLUX6xBayGxLX9UcimVRxXFMHH6q43mAbRvDzSpCsvfK", "Flux", "amm"},
		{"fUSioN9YKKSa3CUC2YUc4tPkHJ5Y6XW1yz8y6F7qWz9", "DefiTuna", "amm"},
	}
	for _, c := range cases {
		p := constants.GetDexProgramByID(c.id)
		if p.Name != c.name || len(p.Tags) != 1 || p.Tags[0] != c.tag {
			t.Errorf("GetDexProgramByID(%s) = %+v, want name %s tag %s", c.id, p, c.name, c.tag)
		}
		if got := constants.GetProgramName(c.id); got != c.name {
			t.Errorf("GetProgramName(%s) = %q, want %q", c.id, got, c.name)
		}
	}
}

// constants-8: DCA keepers and OKX_ROUTER are system-owned wallets (getMultipleAccounts:
// executable=false, owner 11111111111111111111111111111111), not DEX programs.
func TestConstantsAuthoritiesAreNotPrograms(t *testing.T) {
	wallets := []constants.DexProgram{
		constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER1,
		constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER2,
		constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER3,
		constants.DEX_PROGRAMS.OKX_ROUTER,
	}
	for _, w := range wallets {
		if constants.IsDexProgram(w.ID) {
			t.Errorf("IsDexProgram(%s) = true for wallet %s", w.ID, w.Name)
		}
		if constants.GetDexProgramByID(w.ID).Name != "" {
			t.Errorf("GetDexProgramByID(%s) returns wallet %s", w.ID, w.Name)
		}
		if !constants.IsKnownAuthority(w.ID) {
			t.Errorf("%s (%s) missing from KNOWN_AUTHORITIES", w.ID, w.Name)
		}
	}
}

// D3: unknown program IDs are named "Unknown" (upstream TS getProgramName).
func TestConstantsUnknownProgramName(t *testing.T) {
	if got := constants.GetProgramName("11111111111111111111111111111111"); got != "Unknown" {
		t.Errorf("GetProgramName(system) = %q, want Unknown", got)
	}
	if got := constants.GetProgramName(constants.DEX_PROGRAMS.ORCA.ID); got != "Orca" {
		t.Errorf("GetProgramName(Orca) = %q", got)
	}
	if constants.GetDexProgramByID("11111111111111111111111111111111").Name != "" {
		t.Error("GetDexProgramByID returns a program for an unknown ID")
	}
	if !constants.IsVaultProgram(constants.DEX_PROGRAMS.SERUM_V3.ID) || !constants.IsVaultProgram(constants.DEX_PROGRAMS.METEORA_VAULT.ID) {
		t.Error("IsVaultProgram misses a vault-tagged program")
	}
	if constants.IsVaultProgram(constants.DEX_PROGRAMS.ORCA.ID) {
		t.Error("IsVaultProgram(Orca) = true")
	}
}

// parity-16 / amm-13: OpenBook is a system program (upstream TS SYSTEM_PROGRAMS). Raydium V4
// initialize2 CPIs OpenBook; the pool's token transfers must stay grouped under the Raydium V4
// instruction and must not become an unknown-DEX trade (upstream TS reports no trade for
// 2YxPyAJN; before the fix Go reported SELL 50000000000000 from program srmqPv...).
func TestConstantsOpenBookIsSystemProgram(t *testing.T) {
	if !constants.IsSystemProgram("srmqPvymJeFKQ4zGQed1GFppgkRHL9kaELCbyksJtPX") {
		t.Fatal("OpenBook srmqPv... is not in SYSTEM_PROGRAMS")
	}
	for _, sig := range []string{
		"2YxPyAJNfnBLrVpBwMx7qMVNPSvBDhxiquwJGhBjwXhkP6i6AbooUg4b4wpi15bQq2Qs4t7BpL1UVvTMcXL8P4uS",
		"FHz3LurEFNnWREXSfqpenJTRuzybQrmxsNjndmMDw36XUwUofSQ1vRQwVi6uT422Xe2Whb7gfFY3tHGqvEBg6dF",
		"5MRWUUoCWpaFm8B9jSoLp4w6B19p46jcJMQrm26SHHGRZQpAtG3mdEcqGVDwsgUEGKRmC1R6JMFS5NhzmXUnR3X1",
		"4998xRsghpLWWcZsFFtfN8UBss9SVkN8sMUPkqdBzYS6eRVuxve3RoT89pqBaYvHDgqPSsFt9GDGQc6Us3pwPX5v",
		"49ejTjaCdqV3hwAs1GomGsTLqZNUWztHduHRCNr8m3Dogfyii29qkNPNauAr9VfEh9j5m7QHq8HYRd6FiaAACCf",
	} {
		tx := loadFixture(t, sig)
		ctx := newParseContext(tx, nil)
		for key := range ctx.TransferActions {
			if strings.HasPrefix(key, "srmqPvymJeFKQ4zGQed1GFppgkRHL9kaELCbyksJtPX:") {
				t.Errorf("%s: transfers grouped under OpenBook key %s", sig[:8], key)
			}
		}
		res := dexparser.NewDexParser().ParseAll(tx, nil)
		for _, tr := range res.Trades {
			t.Errorf("%s: phantom trade %s program=%s amm=%q in=%s %s", sig[:8], tr.Type, tr.ProgramId, tr.AMM, tr.InputToken.Mint, tr.InputToken.AmountRaw)
		}
	}
}

// constants-v1: the SPL Memo program is skipped for transfer grouping. Whirlpool
// collect_fees_v2 in 5AzH3HAp CPIs Memo before its two transfers; before the fix they were
// grouped under "MemoSq4g...:6-0" and the unknown-DEX fallback emitted a trade with amm "".
func TestConstantsMemoProgramSkipped(t *testing.T) {
	for _, id := range []string{"MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr", "Memo1UhkJRfHyvLMcVucJwxXeuD728EqVDDwQDxFMNo"} {
		found := false
		for _, s := range constants.SKIP_PROGRAM_IDS {
			found = found || s == id
		}
		if !found {
			t.Errorf("%s missing from SKIP_PROGRAM_IDS", id)
		}
	}
	tx := loadFixture(t, "5AzH3HApZUEnGECG5Xk26jgUpbiRAzRsAqTRHiTj7Jf6bX3jfSXZCj5zqREvgnYwzgD2mUXw9g6FrN2hVtJZF5JN")
	ctx := newParseContext(tx, nil)
	for key := range ctx.TransferActions {
		if strings.HasPrefix(key, "Memo") {
			t.Errorf("transfers grouped under Memo key %s", key)
		}
	}
	orcaKey := constants.DEX_PROGRAMS.ORCA.ID + ":6"
	if n := len(ctx.TransferActions[orcaKey]); n != 2 {
		t.Errorf("collect_fees_v2 group %s has %d transfers, want 2", orcaKey, n)
	}
	res := dexparser.NewDexParser().ParseAll(tx, nil)
	for _, tr := range res.Trades {
		if strings.HasPrefix(tr.ProgramId, "Memo") || tr.AMM == "" {
			t.Errorf("trade attributed to the Memo program: program=%s amm=%q", tr.ProgramId, tr.AMM)
		}
	}
}
