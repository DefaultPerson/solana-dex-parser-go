package tests

import (
	"sort"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// Pump.fun Global (4wTV1Ymi...) and PumpSwap GlobalConfig (ADyA8hde...) decoded on-chain with
// the current IDLs on 2026-09-27 (the decode consumed exactly the 1087 / 949 account bytes).
var constPumpReservedAndBuybackRecipients = []string{
	// reserved_fee_recipient + reserved_fee_recipients
	"GesfTA3X2arioaHp8bbKdjG9vJtskViWACZoYvxp4twS",
	"4budycTjhs9fD6xw62VBducVTNgMgJJ5BgtKq7mAZwn6",
	"8SBKzEQU4nLSzcwF4a74F2iaUDQyTfjGndn6qUWBnrpR",
	"4UQeTP1T39KZ9Sfxzo3WR5skgsaP6NZa87BAkuazLEKH",
	"8sNeir4QsLsJdYpc9RZacohhK1Y5FLU3nC5LXgYB4aa6",
	"Fh9HmeLNUMVCvejxCtCL2DbYaRyBFVJ5xrWkLnMH6fdk",
	"463MEnMeGyJekNZFQSTUABBEbLnvMTALbT6ZmsxAbAdq",
	"6AUH3WEHucYZyC61hqpqYUWVto5qA5hjHuNQ32GNnNxA",
	// buyback_fee_recipients
	"5YxQFdt3Tr9zJLvkFccqXVUwhdTWJQc1fFg2YPbxvxeD",
	"9M4giFFMxmFGXtc3feFzRai56WbBqehoSeRE5GK7gf7",
	"GXPFM2caqTtQYC2cJ5yJRi9VDkpsYZXzYdwYpGnLmtDL",
	"3BpXnfJaUTiwXnJNe7Ej1rcbzqTTQUvLShZaWazebsVR",
	"5cjcW9wExnJJiqgLjq7DEG75Pm6JBgE1hNv4B2vHXUW6",
	"EHAAiTxcdDwQ3U4bU6YcMsQGaekdzLS3B5SmYo46kJtL",
	"5eHhjP8JaYkz83CWwvGU2uMUXefd3AazWGx4gpcuEEYD",
	"A7hAgCzFw14fejgCp387JUJRMNyz4j89JKnhtKU8piqW",
}

// constants-12 / parity-15: Pump reserved and buyback fee recipients are fee accounts; the
// migrator (withdraw_authority) is not.
func TestConstantsPumpFeeAccounts(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range constants.FEE_ACCOUNTS {
		if !constDecodes32(a) {
			t.Errorf("fee account %q is not a 32-byte base58 key", a)
		}
		if seen[a] {
			t.Errorf("fee account %s listed twice", a)
		}
		seen[a] = true
	}
	for _, a := range constPumpReservedAndBuybackRecipients {
		if !constants.IsFeeAccount(a) {
			t.Errorf("Pump fee recipient %s is not a fee account", a)
		}
	}
	migrator := "39azUYFWPz3VHgKCf3VChUwbpURdCHRxjWVowf5jUJjg"
	if constants.IsFeeAccount(migrator) {
		t.Errorf("migrator %s is listed as a fee account", migrator)
	}
	if len(constants.PUMPFUN_MIGRATORS) == 0 || constants.PUMPFUN_MIGRATORS[0] != migrator {
		t.Errorf("PUMPFUN_MIGRATORS = %v", constants.PUMPFUN_MIGRATORS)
	}
}

// constants-12: in 2med14no (Jupiter -> PumpSwap buy_exact_quote_in) PumpSwap pays 693011
// WSOL to an account owned by buyback recipient 3BpXnfJa...; the transfer is a fee.
func TestConstantsPumpBuybackTransferIsFee(t *testing.T) {
	tx := loadFixture(t, "2med14noC7dxn5rjnUNvcQFYFG8Lr4Xhxpd2Ev2GZuvnNLYNChvdvXKeRkXcmHhbBtkfNiNGzVC2teecNRCxzhpG")
	ctx := newParseContext(tx, nil)
	found := false
	for _, list := range ctx.TransferActions {
		for _, tr := range list {
			if tr.Info.DestinationOwner == "3BpXnfJaUTiwXnJNe7Ej1rcbzqTTQUvLShZaWazebsVR" {
				found = true
				if !tr.IsFee || tr.Info.TokenAmount.Amount != "693011" {
					t.Errorf("buyback transfer amount=%s IsFee=%v, want 693011 and IsFee", tr.Info.TokenAmount.Amount, tr.IsFee)
				}
			}
		}
	}
	if !found {
		t.Error("buyback fee transfer not found")
	}
}

// Bot fee accounts decode, are unique across bots, and bot programs are tagged "bot"
// (constants-13, constants-14).
func TestConstantsBots(t *testing.T) {
	seen := map[string]string{}
	total := 0
	for bot, accounts := range constants.BOT_FEE_ACCOUNTS {
		for _, a := range accounts {
			total++
			if !constDecodes32(a) {
				t.Errorf("%s fee account %q is not a 32-byte base58 key", bot, a)
			}
			if other, ok := seen[a]; ok {
				t.Errorf("fee account %s listed for %s and %s", a, other, bot)
			}
			seen[a] = bot
		}
	}
	if got := constants.GetBotName("noVaE91mUL5jTb8e9Vf6dqJdNPzJpEQ3uAdnQ8h4nVz"); got != "Nova" {
		t.Errorf("GetBotName(Nova fee wallet) = %q", got)
	}
	var botPrograms []string
	for _, id := range constants.DEX_PROGRAM_IDS {
		for _, tag := range constants.GetDexProgramByID(id).Tags {
			if tag == "bot" {
				botPrograms = append(botPrograms, constants.GetDexProgramByID(id).Name)
			}
		}
	}
	sort.Strings(botPrograms)
	hasGMGN := false
	for _, n := range botPrograms {
		hasGMGN = hasGMGN || n == "GMGN"
	}
	if !hasGMGN {
		t.Error("GMGN program is not tagged bot")
	}
	names := constants.GetBotNames()
	sort.Strings(names)
	t.Logf("bots with fee accounts: %d %v; fee accounts: %d; bot programs: %d %v", len(names), names, total, len(botPrograms), botPrograms)
}

// constants-19: USDY is a yield-accruing note (Jupiter price 1.14 USD on 2026-09-27), not a
// $1 stablecoin; USDS, JupUSD, CASH and USDe are (Jupiter token API "stable" tag). Mints and
// decimals verified with getAccountInfo jsonParsed.
func TestConstantsStablecoins(t *testing.T) {
	if constants.IsStablecoin(constants.TOKENS.USDY) {
		t.Error("USDY treated as a stablecoin")
	}
	if d, ok := constants.GetTokenDecimals(constants.TOKENS.USDY); !ok || d != 6 {
		t.Errorf("USDY decimals = %d, %v", d, ok)
	}
	cases := []struct {
		mint     string
		decimals uint8
	}{
		{"USDSwr9ApdHk5bvJKMjzff41FfuX8bSxdKcR81vTwcA", 6},
		{"JuprjznTrTSp2UFa3ZBUFgwdAmtZCq4MQCwysN55USD", 6},
		{"CASHx9KJUStyftLFWGvEVf59SGeG9sh5FfcnZMVPCASH", 6},
		{"DEkqHyPN7GMRJ5cArtQFAWefqbZb33Hyf6s5iCwjEonT", 9},
	}
	for _, c := range cases {
		if !constants.IsStablecoin(c.mint) || !constants.IsQuoteToken(c.mint) {
			t.Errorf("%s is not a stablecoin/quote token", c.mint)
		}
		if d, ok := constants.GetTokenDecimals(c.mint); !ok || d != c.decimals {
			t.Errorf("%s decimals = %d, %v; want %d", c.mint, d, ok, c.decimals)
		}
	}
	for mint := range constants.TOKEN_DECIMALS {
		if !constDecodes32(mint) {
			t.Errorf("token %q is not a 32-byte base58 key", mint)
		}
	}
}
