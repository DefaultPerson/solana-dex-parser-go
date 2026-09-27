package tests

import (
	"math/big"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Regression tests for bot attribution (DESIGN D9, revised) and the bot
// constants. constants-5, constants-13, constants-14. Expected fee credits are
// computed from the raw balances of each fixture, independently of the adapter.

const (
	// Meteora DAMM buy; the user then pays 171348057 of the bought token to
	// BONKbot's token account CDDXJJ... (owner ZG98FU...)
	sigBonkTokenFee = "4uuw76SPksFw6PvxLFkG9jRyReV1F4EyPYNc3DdSECip8tM22ewqGWJUaRZ1SJEZpuLJz1qPTEPb2es8Zuegng9Z"
	// v1 BONKbot router (CxvksNjw/BBRouter1) PumpSwap trade paying the fee in
	// WSOL to HVbzPx..., the WSOL token account of ZG98FU...
	sigBonkWsolFee = "3Z3oth9Zo6BBq7EvW99A3XRcjQKKmLfRSfuvcy6f9y9uoQhE1g56oGbxEJ65TcrLFqhvDUNZYAvPYCEDQfdk5FUY"
	// Fomo (relayer AgmLJB... signs) OKX V2 route selling USDC; the fee is paid
	// in USDC to HrTf9C..., a token account of R4rNJH... that is not itself
	// among the account keys
	sigFomoUsdcFee = "3ou2zc4g8GGfufg1yX9DJpmSPmr6Uo25LsuocezSf5swegypu1aSYtfw47ZSte6PCUBexSeA4Veim65vqaDXp8zr"

	bonkFeeWallet = "ZG98FUCjb8mJ824Gbs6RsgVmr1FhXb2oNiJHa2dwmPd"
	bonkWsolATA   = "HVbzPxiet4ZgAP6CySVouY8RC3MNxgS35MgE3pWvUVC8"
	fomoFeeWallet = "R4rNJHaffSUotNmqSKNEfDcJE8A7zJUkaoM5Jkd7cYX"
)

// TestCore2DetectBotFeeInTradedToken: BONKbot takes its fee in the bought
// token; strict SOL/WSOL detection missed it. D9: a credit of the trade's
// output mint of at least 0.1% of that leg counts.
func TestCore2DetectBotFeeInTradedToken(t *testing.T) {
	tx, r := parseFixture(t, sigBonkTokenFee, nil)
	agg := r.AggregateTrade
	if agg == nil {
		t.Fatal("no aggregate trade")
	}
	fee := ownerTokenDelta(tx, bonkFeeWallet, agg.OutputToken.Mint)
	if fee.Cmp(big.NewInt(171348057)) != 0 || lamportDelta(tx, bonkFeeWallet).Int64() >= utils.BotFeeMinLamports {
		t.Fatalf("fixture: BONKbot token fee %s, SOL credit %s", fee, lamportDelta(tx, bonkFeeWallet))
	}
	// at least 0.1% of the output leg (the fee is ~0.8%)
	if new(big.Int).Mul(fee, big.NewInt(utils.BotFeeMinLegDivisor)).Cmp(bigStr(agg.OutputToken.AmountRaw)) < 0 {
		t.Fatalf("fixture: fee %s below 0.1%% of output %s", fee, agg.OutputToken.AmountRaw)
	}
	if agg.Bot != "BONKbot" {
		t.Errorf("Bot = %q, want BONKbot", agg.Bot)
	}
}

// TestCore2DetectBotWalletNotInAccountKeys: Fomo's fee wallet only appears as
// the owner of a USDC token account; DetectBot visited account keys only and
// SOL/WSOL only. The fee is 0.85% of the USDC input leg.
func TestCore2DetectBotWalletNotInAccountKeys(t *testing.T) {
	tx, r := parseFixture(t, sigFomoUsdcFee, nil)
	if containsStr(rawAccountKeys(tx), fomoFeeWallet) {
		t.Fatal("fixture: fee wallet is an account key")
	}
	agg := r.AggregateTrade
	if agg == nil || agg.InputToken.Mint != usdcMint {
		t.Fatalf("aggregate %v, want a USDC input", agg)
	}
	fee := ownerTokenDelta(tx, fomoFeeWallet, usdcMint)
	if fee.Cmp(big.NewInt(332500)) != 0 {
		t.Fatalf("fixture: Fomo USDC fee %s", fee)
	}
	if agg.InputToken.AmountRaw != "39050000" {
		t.Fatalf("input %s, want the user's 39050000 USDC", agg.InputToken.AmountRaw)
	}
	if agg.Bot != "Fomo" {
		t.Errorf("Bot = %q, want Fomo", agg.Bot)
	}
}

// TestCore2DetectBotWsolTokenAccount: the BONKbot fee arrives as WSOL in its
// fee wallet's token account HVbzPx... (now listed itself). The lamports move
// with the WSOL, so the account's SOL and WSOL credits are equal.
func TestCore2DetectBotWsolTokenAccount(t *testing.T) {
	tx, r := parseFixture(t, sigBonkWsolFee, nil)
	if got := accountTokenDelta(tx, bonkWsolATA, solMint); got.Cmp(big.NewInt(705691)) != 0 || lamportDelta(tx, bonkWsolATA).Cmp(got) != 0 {
		t.Fatalf("fixture: WSOL credit %s, lamports %s", got, lamportDelta(tx, bonkWsolATA))
	}
	if constants.GetBotName(bonkWsolATA) != "BONKbot" {
		t.Error("HVbzPx... is not listed for BONKbot")
	}
	if r.AggregateTrade == nil || r.AggregateTrade.Bot != "BONKbot" {
		t.Errorf("aggregate %v, want bot BONKbot", r.AggregateTrade)
	}
}

// TestCore2DetectBotThresholds: presence and dust never count; a traded-mint
// credit counts from 0.1% of the leg; the trader's own accounts never count.
// Unit test on the real balances of the Fomo and Axiom-dust fixtures.
func TestCore2DetectBotThresholds(t *testing.T) {
	detect := func(tx *adapter.SolanaTransaction, trade types.TradeInfo) string {
		utils.NewTransactionUtils(adapter.NewTransactionAdapter(tx, nil)).DetectBot(&trade)
		return trade.Bot
	}
	usdcLeg := func(amount string) types.TradeInfo {
		return types.TradeInfo{
			User:        "11111111111111111111111111111111",
			InputToken:  types.TokenInfo{Mint: usdcMint, AmountRaw: amount},
			OutputToken: types.TokenInfo{Mint: "So11111111111111111111111111111111111111112", AmountRaw: "1"},
		}
	}
	fomo := loadFixture(t, sigFomoUsdcFee)
	// fee 332500: counts up to a leg of 332500000 (0.1%), not above
	if got := detect(fomo, usdcLeg("332500000")); got != "Fomo" {
		t.Errorf("fee = 0.1%% of the leg: Bot = %q, want Fomo", got)
	}
	if got := detect(fomo, usdcLeg("332500001")); got != "" {
		t.Errorf("fee < 0.1%% of the leg: Bot = %q, want none", got)
	}
	// a USDC credit does not count when USDC is not a trade leg
	other := usdcLeg("39050000")
	other.InputToken.Mint = usdtMint
	if got := detect(fomo, other); got != "" {
		t.Errorf("fee in a mint that is not traded: Bot = %q, want none", got)
	}
	// the fee wallet as the trader is not a fee receiver
	self := usdcLeg("39050000")
	self.User = fomoFeeWallet
	if got := detect(fomo, self); got != "" {
		t.Errorf("fee wallet as the user: Bot = %q, want none", got)
	}

	// SOL: the 10-lamport credit to an Axiom wallet counts up to a leg of
	// 10000 lamports (0.1%), not above; the 10000-lamport floor caps the
	// threshold on large legs
	dust := loadFixture(t, sigAxiomDust)
	solLeg := func(amount string) types.TradeInfo {
		return types.TradeInfo{
			InputToken:  types.TokenInfo{Mint: usdcMint, AmountRaw: "100"},
			OutputToken: types.TokenInfo{Mint: solMint, AmountRaw: amount},
		}
	}
	if got := detect(dust, solLeg("10000")); got != "Axiom" {
		t.Errorf("10 lamports = 0.1%% of the SOL leg: Bot = %q, want Axiom", got)
	}
	if got := detect(dust, solLeg("10001")); got != "" {
		t.Errorf("10 lamports < 0.1%% of the SOL leg: Bot = %q, want none", got)
	}
	paid := loadFixture(t, sigAxiomPaid) // 13511 lamports to an Axiom wallet
	if got := detect(paid, solLeg("1000000000000")); got != "Axiom" {
		t.Errorf(">= 10000 lamports on a 1000 SOL leg: Bot = %q, want Axiom", got)
	}
}

// TestCore2BotConstants: the bot programs and fee accounts added from the
// 2026-09 research (research/bots.json, recommendation "add", evidence
// re-checked on-chain) are present and consistent; nothing was removed (D12).
func TestCore2BotConstants(t *testing.T) {
	programs := map[string]string{
		"GMgnVFR8Jb39LoXsEVzb3DvBy3ywCmdmJquHUy1Lrkqb": "GMGN",
		"GMGNreQcJFufBiCTLDBgKhYEfEe9B454UjpDr5CaSLA1": "GMGN",
		"DGMgNKpqygARV2pHZfW4kNQSHT9F3Ly2BKWqvpYrAg5C": "GMGN",
		"BLUR9cL8HqZzu5bSaC7VRX25RCG93Hv3T6NPyKxQhWUT": "Axiom",
		"FLASHX8DrLbgeR8FcfNV1F5krxYcYMUdBkrP1EPBtxB9": "Axiom",
		"9Fox6i7oT8p4qHn76Qj3dks8RRMGsXQyfMSBScA5yVyX": "Padre",
		"term9YPb9mzAsABaqN71A4xdbxHmpBNZavpBiQKZzN3":  "Padre",
		"troyXT7Ty3s2rjJe4bqWaroUrS4Fjd8rbHHNHxcACF4":  "Trojan",
		"TroYL71c8P2XNtDxHs98VtVLuiASJ7Ao5FvUoKyp3Bk":  "Trojan",
		"troY36YiPGqMyAYCNbEqYCdN2tb91Zf7bHcQt7KUi61":  "Trojan",
		"CxvksNjwhdHDLr3qbCXNKVdeYACW8cs93vFqLqtgyFE5": "BONKbot",
		"BBRouter1cVunVXvkcqeKkZQcBK7ruan37PPm3xzWaXD": "BONKbot",
		"Stbot61LkCD5HE4p1TDtYRt9bzkjX81pKrshZz4Awny":  "STBot",
		"MevxQ9iQNGNQSWqa2CtDQs2BwrJA6MifzocykZAgGBr":  "MevX",
		"BujKR6saP3wZa7PEZ2u8ktyumdv8jqGsgHjUU7Ur7kLp": "Nova",
	}
	for id, bot := range programs {
		p := constants.GetDexProgramByID(id)
		if p.Name != bot || len(p.Tags) != 1 || p.Tags[0] != "bot" || !constants.IsDexProgram(id) {
			t.Errorf("program %s = %+v, want %s tagged bot", id, p, bot)
		}
		if len(constants.BOT_FEE_ACCOUNTS[bot]) == 0 {
			t.Errorf("bot %s has no fee accounts", bot)
		}
	}
	feeAccounts := map[string]string{
		"JBok73TJsWdgeJy2x59TaTFKtngtxeYPvizafTnhvMGV": "BananaGun",
		"6nPV8EChoA3HUZRbFmM6cXDv41NApX9ALKwDw4vYfjWp": "BananaGun",
		"35q8cao77A8ceJVxQaoN5w9kyTjTJUsmQcy2RRNrJBMc": "BananaGun",
		"3spK1TmrAnFUDRN2bErDAVw225tLE7uFpQeP9WNXP2nY": "BananaGun",
		"36ZCrKd6N9iGmArccia15saDyepL5wShHJyGTvea3Akf": "BananaGun",
		bonkWsolATA: "BONKbot",
		"noVakKQGTTjpHARvecAUbVnc85AatCLm3ijDFk8JXZB": "Nova",
		"K1LRSA1DSoKBtC5DkcvnermRQ62YxogWSCZZPWQrdG5": "STBot",
		fomoFeeWallet: "Fomo",
		// kept (D12): closed or dormant legacy entries
		"FRMxAnZgkW58zbYcE7Bxqsg99VWpJh6sMP5xLzAWNabN": "Maestro",
		"96aFQc9qyqpjMfqdUeurZVYRrrwPJG2uPV6pceu4B1yb": "STBot",
		"Cj297UauzMX64FU9dKJZRUBWszJ7tEWpVheasq4CfATV": "BananaGun",
	}
	for account, bot := range feeAccounts {
		if got := constants.GetBotName(account); got != bot {
			t.Errorf("GetBotName(%s) = %q, want %s", account, got, bot)
		}
	}
	// the Jupiter referral vault owning BananaGun's legacy WSOL accounts is
	// not a bot wallet
	if got := constants.GetBotName("45ruCyfdRkWpRNGEqWzjCiXRHkZs8WXCLQ67Pnpye7Hp"); got != "" {
		t.Errorf("Jupiter referral vault labelled %q", got)
	}
	// bot programs are neither fee accounts nor tip accounts
	for id := range programs {
		if constants.IsBotFeeAccount(id) || constants.IsTipAccount(id) {
			t.Errorf("bot program %s is listed as an account", id)
		}
	}
}

// TestCore2BotProgramsAreRoutes: transactions executed through a bot program
// report the bot as the route (bot programs are tagged "bot", which GetDexInfo
// treats as a route); before, these programs were unknown and the route empty.
func TestCore2BotProgramsAreRoutes(t *testing.T) {
	cases := []struct{ sig, program, route string }{
		{sigAxiomDust, "FLASHX8DrLbgeR8FcfNV1F5krxYcYMUdBkrP1EPBtxB9", "Axiom"},
		{"36Q2tYo1CPa42GF51bzA493nYQCG8fPbpQJEzRhZQURYuBcRKpj97HWBCLCzDwgQJ8tnVrW9fDZKWaPBdADEsxTE", "CxvksNjwhdHDLr3qbCXNKVdeYACW8cs93vFqLqtgyFE5", "BONKbot"},
		{"mszDRFhMPQLkH7ciTTtBfUJvpMoSqhjvv16ZTHK7pZ9sd2Pi8XpPALgUZwDfiC3VnouA2iauvAdAN89t5Xo1PkR", "9Fox6i7oT8p4qHn76Qj3dks8RRMGsXQyfMSBScA5yVyX", "Padre"},
		{"5tn4UECvbP7n6AKNmzZh646k2P5uPsbtoDHXPbH2U8YF5wWnNwiWcpY5hcTs8xW9tZbabroFJPHa6227qiy89Vcu", "BLUR9cL8HqZzu5bSaC7VRX25RCG93Hv3T6NPyKxQhWUT", "Axiom"},
	}
	p := dexparser.NewDexParser()
	for _, c := range cases {
		tx := loadFixture(t, c.sig)
		first := ""
		for _, ix := range tx.Transaction.Message.Instructions {
			m := ix.(map[string]interface{})
			id := rawAccountKeys(tx)[jsonInt(m["programIdIndex"])]
			if constants.IsDexProgram(id) {
				first = id
				break
			}
		}
		if first != c.program {
			t.Fatalf("%s: first known outer program %s, want %s", c.sig[:8], first, c.program)
		}
		r := p.ParseAll(tx, nil)
		if r.AggregateTrade == nil || r.AggregateTrade.Route != c.route {
			t.Errorf("%s: aggregate %v, want route %s", c.sig[:8], r.AggregateTrade, c.route)
		}
	}
}
