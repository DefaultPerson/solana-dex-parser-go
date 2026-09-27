package tests

import (
	"crypto/sha256"
	"testing"

	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// bump255Address is the address the old implementation returned: the bump-255
// hash without the on-curve check.
func bump255Address(t *testing.T, owner, tokenProgram, mint string) string {
	t.Helper()
	var buf []byte
	for _, s := range []string{owner, tokenProgram, mint} {
		b, err := base58.Decode(s)
		if err != nil {
			t.Fatal(err)
		}
		buf = append(buf, b...)
	}
	buf = append(buf, 255)
	ata, _ := base58.Decode(constants.ASSOCIATED_TOKEN_PROGRAM_ID)
	buf = append(buf, ata...)
	buf = append(buf, "ProgramDerivedAddress"...)
	h := sha256.Sum256(buf)
	return base58.Encode(h[:])
}

// TestCoreATARealAccounts derives the ATA for every (owner, mint) token balance
// in the fixtures and checks it against the real token account. Token accounts
// that are not ATAs (pool vaults, keypair accounts) simply do not match; the
// test requires >= 20 matches and that some of them need a bump below 255.
func TestCoreATARealAccounts(t *testing.T) {
	matched, needLowerBump := 0, 0
	seen := map[string]bool{}
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		if tx.Meta == nil {
			continue
		}
		ctx := newParseContext(tx, nil)
		for _, b := range tx.Meta.PostTokenBalances {
			account := ctx.Adapter.GetAccountKey(b.AccountIndex)
			if b.Owner == "" || b.Mint == "" || account == "" || seen[account] {
				continue
			}
			seen[account] = true
			std, t22, err := utils.FindAssociatedTokenAddress(b.Owner, b.Mint)
			if err != nil {
				t.Fatalf("%s: %v", sig, err)
			}
			program := ""
			switch account {
			case std:
				program = constants.TOKEN_PROGRAM_ID
			case t22:
				program = constants.TOKEN_2022_PROGRAM_ID
			default:
				continue
			}
			matched++
			if bump255Address(t, b.Owner, program, b.Mint) != account {
				needLowerBump++
			}
		}
	}
	t.Logf("real ATAs matched=%d, of which bump<255=%d", matched, needLowerBump)
	if matched < 20 {
		t.Errorf("expected >= 20 real ATAs to match the derivation, got %d", matched)
	}
	if needLowerBump == 0 {
		t.Errorf("expected some real ATAs with a bump below 255")
	}
}

// TestCoreATAKnownBumps checks ATAs whose bump is below 255. The expected
// addresses come from postTokenBalances of real transactions (and, for the
// first one, from a live getTokenAccountsByOwner call during the audit); the
// bumps were computed independently with a Python ed25519 implementation.
func TestCoreATAKnownBumps(t *testing.T) {
	cases := []struct {
		owner, mint, program, ata string
		bump                      int
	}{
		{"9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", constants.TOKEN_PROGRAM_ID, "FGETo8T8wMcN2wCjav8VK6eh3dLk63evNDPxzLSJra8B", 254},
		{"sysYyMzKebh3awca8BE9r9YbPZS4j5JoDdzKjjPfiaX", "So11111111111111111111111111111111111111112", constants.TOKEN_PROGRAM_ID, "EgfRosVsgBcaFUgkdtyxKAtVmsf1uefUgNQ4x7jdjbo6", 250},
		{"GP8StUXNYSZjPikyRsvkTbvRV1GBxMErb59cpeCJnDf1", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", constants.TOKEN_PROGRAM_ID, "HoBCz6z9AG92GGozMWEkBPE9UhQWGZ5cXhYcjoGJvwP2", 254},
		{"2c4Nd25BehQBNpug4E5DAVVSNmyYZxYBVUnjFMu19iMS", "B8A4bitd8bfsLYZQ67qbuwQxAZAY7D1UCXjJoYd2pump", constants.TOKEN_2022_PROGRAM_ID, "8dvLegUtGXBwWZKJRkyTEioNaFiQxGHmRG4bMeUpgUPQ", 252},
		{"9ZQuCDRBDvDNsgaDe593dRHnyGgEYaisXF4VYMrZDLBa", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", constants.TOKEN_PROGRAM_ID, "5DSCMCPahPgj74gAuectLCEUVGtEWVsJid5JSLvjriaq", 251},
	}
	for _, c := range cases {
		std, t22, err := utils.FindAssociatedTokenAddress(c.owner, c.mint)
		if err != nil {
			t.Fatal(err)
		}
		got := std
		if c.program == constants.TOKEN_2022_PROGRAM_ID {
			got = t22
		}
		if got != c.ata {
			t.Errorf("ATA(%s, %s) = %s, want %s", c.owner, c.mint, got, c.ata)
		}
		owner, _ := base58.Decode(c.owner)
		prog, _ := base58.Decode(c.program)
		mint, _ := base58.Decode(c.mint)
		addr, bump, err := utils.FindProgramAddress([][]byte{owner, prog, mint}, constants.ASSOCIATED_TOKEN_PROGRAM_ID)
		if err != nil || addr != c.ata || int(bump) != c.bump {
			t.Errorf("FindProgramAddress = %s bump %d (%v), want %s bump %d", addr, bump, err, c.ata, c.bump)
		}
		// The shred parsers classify direction from the user's ATA.
		if tt := utils.GetAccountTradeType(c.owner, c.mint, c.ata, "other"); tt != types.TradeTypeSell {
			t.Errorf("GetAccountTradeType(input=ATA) = %s, want SELL", tt)
		}
		if tt := utils.GetAccountTradeType(c.owner, c.mint, "other", c.ata); tt != types.TradeTypeBuy {
			t.Errorf("GetAccountTradeType(output=ATA) = %s, want BUY", tt)
		}
	}
}

// TestCoreIsOnCurve checks the curve test on known points: the ed25519 base
// point and a real wallet key are on the curve, a real PDA (an ATA) is not.
func TestCoreIsOnCurve(t *testing.T) {
	for _, c := range []struct {
		key  string
		want bool
	}{
		{"9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM", true},  // wallet (ed25519 public key)
		{"FGETo8T8wMcN2wCjav8VK6eh3dLk63evNDPxzLSJra8B", false}, // ATA (PDA)
		{"EgfRosVsgBcaFUgkdtyxKAtVmsf1uefUgNQ4x7jdjbo6", false}, // ATA (PDA)
	} {
		b, _ := base58.Decode(c.key)
		if got := utils.IsOnCurve(b); got != c.want {
			t.Errorf("IsOnCurve(%s) = %v, want %v", c.key, got, c.want)
		}
	}
	// ed25519 base point: y = 4/5, encoded little-endian as 0x58 0x66 ... 0x66
	base := make([]byte, 32)
	base[0] = 0x58
	for i := 1; i < 32; i++ {
		base[i] = 0x66
	}
	if !utils.IsOnCurve(base) {
		t.Error("base point reported off curve")
	}
}
