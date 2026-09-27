package tests

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Regression tests for trade amounts, fees, balance changes and bot
// attribution (DESIGN D6-D9). Expected values come from the transactions'
// balances.

const (
	// Raydium CPMM buy of 7atgF8 (Token-2022, 4% transfer fee)
	sigT22Fee  = "51nj5GtAmDC23QkeyfCNfTJ6Pdgwx7eq4BARfq1sMmeEaPeLsx9stFA3Dzt9MeLV5xFujBgvghLGcayC3ZevaQYi"
	t22FeeMint = "7atgF8KQo4wJrD5ATGX7t1V2zVvykPJbFfNeVf1icFv1"
	// Pump.fun create + buy (Metaplex CPI with a rent transfer)
	sigPumpCreate = "4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz"
	// LaunchLab sell whose token account is closed in the tx (issue #82)
	sigSellAndClose = "2iWEMsA1bp4ce5UGKj4A9pL6wLPT4ci6LJuqewbAbvDTCREWSrZcDecmjMDceQ8afZZGT4qTm3vb7hoWRU1nrt3k"
	// v1 PumpSwap sell sending 1 and 10 lamports to Axiom fee wallets
	sigAxiomDust = "43krAzD5JNy8LciSpcDN4CbPi7DcWuDfh6ifcUribrZn1sfYVb3rU7zYKyzib1GkYyBMHDmQiyF2QA6XiH2CVZZQ"
	// v1 Boop.fun sell paying 13511 lamports to an Axiom fee wallet
	sigAxiomPaid = "45zuVzTVAMkLUQEKxfsZ4L2rkRDGWNBCnsdBM1sQML6c1fBmFG8T7JwoQ8HcyoBmSMmySHjRyuZ9SYf65FD517ua"
)

// TestCoreNoInventedTradeFee: AttachTradeFee set fee = output - user's
// balance change, booking forwarded outputs, arbitrage profit and network
// fees as trade fees (up to 99.99% of the output). D8. core-7, amm-v1, parity-24.
func TestCoreNoInventedTradeFee(t *testing.T) {
	cases := []struct {
		sig  string
		note string
	}{
		{"mLQ2LNtkgyRoTd62NoAoDs4pUBNqFXVx1TT4eTfnxZvmE3389x9zZZEEDC4SaEo49egH51ySS5gnwVvP7iQXq9v", "swap then pay: fee was 39039682 of 39041035 USDC"},
		{"5sV51YrwGRFwpwgnWHWKCq3TWmCQ47XnsNuMNAKs7Jypf9g6H9JTPnY6ZqW5tfSTBwBL8Un5MPxK5cKZsSkaacty", "SOL arbitrage: fee was 114528994"},
		{"5nFMqRsLrVP4JVaLS1wpSNzrhgyWRkaMfEMKdBxXgPVFM2wGHKRKyiF2Zp6VrWp3XCpbzmW5Fi5cuDff4jhAA9gN", "USDT->USDC forwarded"},
		{"NeF1UiWXKUbuswNNw14gJ2uup7yrV6KyXijQ7dsjLYanj9dnsBSeqGPDVZ3wP3NfhXQ84rJncRo5XwbrbSdWVWW", "SOL output: fee was the network fee"},
	}
	for _, c := range cases {
		sig := c.sig
		tx, r := parseFixture(t, sig, nil)
		agg := r.AggregateTrade
		if agg == nil {
			t.Fatalf("%.8s: no aggregate", sig)
		}
		// none of these transactions has a protocol fee event or a transfer to
		// a known fee account
		ctx := newParseContext(tx, nil)
		for _, ts := range ctx.TransferActions {
			for _, x := range ts {
				if x.IsFee {
					t.Fatalf("%.8s: fixture has an explicit fee transfer", sig)
				}
			}
		}
		if agg.Fee != nil || len(agg.Fees) != 0 {
			t.Errorf("%.8s (%s): aggregate fee %v %v, want none", sig, c.note, agg.Fee, agg.Fees)
		}
		for _, tr := range r.Trades {
			if tr.Fee != nil && tr.Fee.AmountRaw == agg.OutputToken.AmountRaw {
				t.Errorf("%.8s: trade fee equals the output", sig)
			}
		}
	}
}

// TestCoreAggregateDoesNotAliasTrades: for one trade GetFinalSwap returned
// &trades[0], and AttachTradeFee mutated result.Trades[0]. core-19, amm-v1.
func TestCoreAggregateDoesNotAliasTrades(t *testing.T) {
	_, r := parseFixture(t, "mLQ2LNtkgyRoTd62NoAoDs4pUBNqFXVx1TT4eTfnxZvmE3389x9zZZEEDC4SaEo49egH51ySS5gnwVvP7iQXq9v", nil)
	if len(r.Trades) != 1 || r.AggregateTrade == nil {
		t.Fatalf("trades=%d aggregate=%v", len(r.Trades), r.AggregateTrade != nil)
	}
	before := r.Trades[0]
	r.AggregateTrade.OutputToken.AmountRaw = "0"
	r.AggregateTrade.Signer[0] = "x"
	if r.Trades[0].OutputToken.AmountRaw != before.OutputToken.AmountRaw || r.Trades[0].Signer[0] == "x" {
		t.Error("the aggregate shares memory with result.Trades[0]")
	}

	trades := []types.TradeInfo{{Idx: "1", Signer: []string{"a"}, Fee: &types.FeeInfo{AmountRaw: "5"}}}
	agg := utils.GetFinalSwap(trades, nil)
	agg.Signer[0] = "b"
	agg.Fee.AmountRaw = "6"
	if trades[0].Signer[0] != "a" || trades[0].Fee.AmountRaw != "5" {
		t.Error("GetFinalSwap(one trade) aliases the input")
	}
}

// TestCoreAggregateKeepsTokenDetails: the aggregate carried only mint,
// amounts and decimals; upstream spreads the first input and last output
// token (authority, source, destination, balances, balanceChange) and the
// signer. core-19, parity-17.
func TestCoreAggregateKeepsTokenDetails(t *testing.T) {
	_, r := parseFixture(t, "4txewy5B76FNPmogsAaWPTJREgqzCrexCDG6dhFiFh9bq5MYRFmu12icLmQVwujck8yY6DT27QBHiuEXTTJvGJJ4", nil)
	if len(r.Trades) < 2 || r.AggregateTrade == nil {
		t.Fatalf("trades=%d aggregate=%v", len(r.Trades), r.AggregateTrade != nil)
	}
	agg := r.AggregateTrade
	first, last := r.Trades[0].InputToken, r.Trades[len(r.Trades)-1].OutputToken
	in, out := agg.InputToken, agg.OutputToken
	if in.Authority != first.Authority || in.Source != first.Source || in.Destination != first.Destination ||
		in.DestinationOwner != first.DestinationOwner || in.BalanceChange != first.BalanceChange || in.SourceBalance == nil || in.SourcePreBalance == nil {
		t.Errorf("aggregate input %+v, want the details of the first trade's input %+v", in, first)
	}
	if out.Authority != last.Authority || out.Source != last.Source || out.Destination != last.Destination ||
		out.BalanceChange != last.BalanceChange || out.DestinationBalance == nil {
		t.Errorf("aggregate output %+v, want the details of the last trade's output %+v", out, last)
	}
	if len(agg.Signer) == 0 || agg.Signer[0] != r.Signer[0] {
		t.Errorf("aggregate Signer = %v, want %v", agg.Signer, r.Signer)
	}
}

// TestCoreSwapLegSumming: transfers were deduplicated by amount+mint, so
// independent legs of equal size were counted once; pass-through legs (the
// same amount forwarded from the account the previous leg credited) must still
// be counted once. core-20.
//
// Synthetic transfers (no real transaction with equal independent legs in a
// ProcessSwapData group was found among the fixtures); the adapter comes from
// a real fixture only to provide the signer.
func TestCoreSwapLegSumming(t *testing.T) {
	a := adapter.NewTransactionAdapter(loadFixture(t, sigJupPumpswap), nil)
	tu := utils.NewTransactionUtils(a)
	user := a.Signer()
	leg := func(idx, mint, amount, src, dst string) types.TransferData {
		ui := 0.0
		authority := user
		if src == "pool" {
			authority = "poolAuthority"
		}
		return types.TransferData{Type: "transfer", Idx: idx, ProgramId: constants.TOKEN_PROGRAM_ID, Info: types.TransferDataInfo{
			Mint: mint, Source: src, Destination: dst, Authority: authority,
			TokenAmount: types.TokenAmount{Amount: amount, Decimals: 6, UIAmount: &ui},
		}}
	}
	// two independent 5-USDC legs into two pools, one token output
	split := []types.TransferData{
		leg("1-0", usdcMint, "5000000", "userUsdc", "poolA"),
		leg("1-1", usdcMint, "5000000", "userUsdc", "poolB"),
		leg("1-2", "TOKEN", "70", "pool", "userTok"),
	}
	tr := tu.ProcessSwapData(split, types.DexInfo{}, false)
	if tr == nil || tr.InputToken.Mint != usdcMint || tr.InputToken.AmountRaw != "10000000" || tr.OutputToken.AmountRaw != "70" {
		t.Errorf("split legs: %v, want 10000000 USDC in", tr)
	}
	// pass-through: user -> router account -> pool, same amount
	chained := []types.TransferData{
		leg("1-0", usdcMint, "5000000", "userUsdc", "router"),
		leg("1-1", usdcMint, "5000000", "router", "pool"),
		leg("1-2", "TOKEN", "70", "pool", "userTok"),
	}
	tr = tu.ProcessSwapData(chained, types.DexInfo{}, false)
	if tr == nil || tr.InputToken.AmountRaw != "5000000" {
		t.Errorf("pass-through legs: %v, want 5000000 USDC in", tr)
	}
}

// TestCoreSkipNativeInUnknownDEX: skipNative compared the mint with
// TOKENS.NATIVE while native transfers carry TOKENS.SOL, so System transfers
// (rent, fees, tips) became swap legs of unknown programs: here a Metaplex
// rent payment plus the mint of the whole supply became a BUY. core-10.
func TestCoreSkipNativeInUnknownDEX(t *testing.T) {
	tx := loadFixture(t, sigPumpCreate)
	ctx := newParseContext(tx, nil)
	var group []types.TransferData
	for _, key := range utils.SortedTransferKeys(ctx.TransferActions) {
		if len(key) > len(constants.METAPLEX_PROGRAM_ID) && key[:len(constants.METAPLEX_PROGRAM_ID)] == constants.METAPLEX_PROGRAM_ID {
			group = ctx.TransferActions[key]
		}
	}
	if len(group) == 0 {
		t.Fatal("no Metaplex transfer group in the fixture")
	}
	if ctx.Utils.ProcessSwapData(group, types.DexInfo{}, false) == nil {
		t.Fatal("without skipNative the group should form a (bogus) trade")
	}
	if tr := ctx.Utils.ProcessSwapData(group, types.DexInfo{}, true); tr != nil {
		t.Errorf("skipNative: got trade %s from a rent transfer", tradeKey(*tr))
	}

	p := dexparser.NewDexParser()
	r := p.ParseAll(tx, &types.ParseConfig{ParseType: types.ParseAll(), TryUnknownDEX: true})
	user := tx.Transaction.Message.AccountKeys[0].Pubkey
	const mint = "B9Z9mKUoVy5k8KuL2HauUD1mhmfF3PPNnJoK83S1pump"
	if len(r.Trades) != 1 || r.Trades[0].ProgramId != constants.DEX_PROGRAMS.PUMP_FUN.ID {
		t.Fatalf("trades = %v, want only the Pump.fun buy", r.Trades)
	}
	agg := r.AggregateTrade
	if agg == nil || agg.InputToken.AmountRaw != r.Trades[0].InputToken.AmountRaw ||
		bigStr(agg.OutputToken.AmountRaw).Cmp(ownerTokenDelta(tx, user, mint)) != 0 {
		t.Errorf("aggregate = %v, want the buy only (output = user's token delta %s)", agg, ownerTokenDelta(tx, user, mint))
	}
}

// TestCoreClosedAccountBalanceChange: token accounts closed within the tx
// reported change 0 and were keyed by the token account instead of the owner,
// so a sell-all-and-close lost the sold token. core-11, parity-20.
func TestCoreClosedAccountBalanceChange(t *testing.T) {
	const soldMint = "54g1ouuZzEDL3uNCgPQqapoZC7LSV7fyBvh79QBrbonk"
	tx, r := parseFixture(t, sigSellAndClose, nil)
	user := tx.Transaction.Message.AccountKeys[0].Pubkey
	want := ownerTokenDelta(tx, user, soldMint) // pre-only entry: -pre
	if want.Sign() >= 0 {
		t.Fatal("fixture: the sold token account is not closed")
	}
	got, ok := r.TokenBalanceChange[soldMint]
	if !ok || bigStr(got.Change.Amount).Cmp(want) != 0 || got.Post.Amount != "0" {
		t.Errorf("TokenBalanceChange[%s] = %+v, want change %s post 0", soldMint, got, want)
	}
	spent := new(big.Int).Neg(want).String()
	if len(r.Trades) == 0 || r.Trades[0].InputToken.BalanceChange != spent {
		t.Errorf("trade input BalanceChange = %v, want %s", r.Trades, spent)
	}

	// per-account view: every closed account reports -pre under its owner
	a := adapter.NewTransactionAdapter(tx, nil)
	byOwner := a.GetAccountTokenBalanceChanges(true)
	for _, b := range tx.Meta.PreTokenBalances {
		closed := true
		for _, pb := range tx.Meta.PostTokenBalances {
			if pb.AccountIndex == b.AccountIndex {
				closed = false
			}
		}
		if !closed || b.UiTokenAmount.Amount == "0" {
			continue
		}
		if c := byOwner[b.Owner][b.Mint]; c == nil || bigStr(c.Change.Amount).Cmp(ownerTokenDelta(tx, b.Owner, b.Mint)) != 0 {
			t.Errorf("owner %s mint %s: change %+v, want %s", b.Owner, b.Mint, c, ownerTokenDelta(tx, b.Owner, b.Mint))
		}
	}
}

// TestCoreClosedAccountsAllOwners checks every fixture: the owner-keyed token
// balance changes equal the owner's summed post - pre from the raw balances,
// including accounts closed within the transaction. core-11.
func TestCoreClosedAccountsAllOwners(t *testing.T) {
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		a := adapter.NewTransactionAdapter(tx, nil)
		changes := a.GetAccountTokenBalanceChanges(true)
		for _, b := range append(append([]adapter.TokenBalance{}, tx.Meta.PreTokenBalances...), tx.Meta.PostTokenBalances...) {
			if b.Owner == "" {
				continue
			}
			want := ownerTokenDelta(tx, b.Owner, b.Mint)
			c := changes[b.Owner][b.Mint]
			if want.Sign() == 0 {
				continue
			}
			if c == nil || bigStr(c.Change.Amount).Cmp(want) != 0 {
				t.Errorf("%.12s owner %.8s mint %.8s: change %v, want %s", sig, b.Owner, b.Mint, c, want)
			}
		}
	}
}

// TestCoreToken2022TransferFeeOutput: for a Token-2022 mint with a transfer
// fee the trade output was the gross transfer, not what the user received.
// amm-8.
func TestCoreToken2022TransferFeeOutput(t *testing.T) {
	tx, r := parseFixture(t, sigT22Fee, nil)
	user := tx.Transaction.Message.AccountKeys[0].Pubkey
	received := ownerTokenDelta(tx, user, t22FeeMint)
	if len(r.Trades) != 1 {
		t.Fatalf("trades = %v", r.Trades)
	}
	tr := r.Trades[0]
	if bigStr(tr.OutputToken.AmountRaw).Cmp(received) != 0 {
		t.Errorf("output = %s, want %s (the user's balance delta)", tr.OutputToken.AmountRaw, received)
	}
	withheld := new(big.Int).Sub(bigStr("390793006"), received).String()
	found := false
	for _, f := range tr.Fees {
		if f.Type == "transferFee" && f.Mint == t22FeeMint && f.AmountRaw == withheld {
			found = true
		}
	}
	if !found {
		t.Errorf("Fees = %v, want transferFee %s", tr.Fees, withheld)
	}
	if r.AggregateTrade == nil || r.AggregateTrade.OutputToken.AmountRaw != tr.OutputToken.AmountRaw {
		t.Errorf("aggregate output = %v", r.AggregateTrade)
	}
}

// TestCoreTransferCheckedWithFee: the Token-2022 TransferCheckedWithFee
// instruction (26/1) was not extracted, so the output leg was missing.
// core-13.
//
// Synthetic: built from the real 51nj5GtA by rewriting its Token-2022
// transferChecked (same accounts and amount) into TransferCheckedWithFee
// carrying the fee the mint actually withheld.
func TestCoreTransferCheckedWithFee(t *testing.T) {
	orig := loadFixture(t, sigT22Fee)
	user := orig.Transaction.Message.AccountKeys[0].Pubkey
	received := ownerTokenDelta(orig, user, t22FeeMint)

	tx := cloneTx(t, orig)
	rewritten := false
	for si := range tx.Meta.InnerInstructions {
		for ii, ix := range tx.Meta.InnerInstructions[si].Instructions {
			m := ix.(map[string]interface{})
			data, _ := base58.Decode(m["data"].(string))
			keys := rawAccountKeys(tx)
			pid := jsonInt(m["programIdIndex"])
			if keys[pid] != constants.TOKEN_2022_PROGRAM_ID || len(data) != 10 || data[0] != constants.SPLTokenTransferChecked {
				continue
			}
			amount := binary.LittleEndian.Uint64(data[1:9])
			fee := amount - received.Uint64()
			nd := []byte{26, 1}
			nd = binary.LittleEndian.AppendUint64(nd, amount)
			nd = append(nd, data[9])
			nd = binary.LittleEndian.AppendUint64(nd, fee)
			m["data"] = base58.Encode(nd)
			tx.Meta.InnerInstructions[si].Instructions[ii] = m
			rewritten = true
		}
	}
	if !rewritten {
		t.Fatal("no Token-2022 transferChecked in the fixture")
	}

	ctx := newParseContext(tx, nil)
	var leg *types.TransferData
	for _, x := range utils.SortedTransfers(ctx.TransferActions) {
		if x.Info.Mint == t22FeeMint {
			x := x
			leg = &x
		}
	}
	if leg == nil || leg.Type != "transferChecked" || leg.Info.TokenAmount.Amount != "390793006" {
		t.Fatalf("TransferCheckedWithFee leg = %+v", leg)
	}
	trades := dexparser.NewDexParser().ParseTrades(tx, nil)
	if len(trades) != 1 || bigStr(trades[0].OutputToken.AmountRaw).Cmp(received) != 0 {
		t.Errorf("trades = %v, want output %s", trades, received)
	}
}

// TestCoreTransferMintPreference: a transfer into an account the tx does not
// describe (created and closed within the tx) took the destination's default
// SOL entry as mint; upstream prefers the non-SOL mint of either side. core-12.
//
// Synthetic: the real 51nj5GtA with the user's output account hidden
// (hideTokenAccount).
func TestCoreTransferMintPreference(t *testing.T) {
	const dest = "4EHZFwbbVsHzsCN2QxoKgrqSHr41z5LCYxGhMeZ9mdXo"
	tx := hideTokenAccount(t, loadFixture(t, sigT22Fee), dest)
	a := adapter.NewTransactionAdapter(tx, nil)
	if !a.IsGuessedTokenAccount(dest) {
		t.Fatal("destination should be unknown to the adapter")
	}
	ctx := newParseContext(tx, nil)
	for _, x := range utils.SortedTransfers(ctx.TransferActions) {
		if x.Info.Destination == dest {
			if x.Info.Mint != t22FeeMint || x.Info.TokenAmount.Amount != "390793006" {
				t.Errorf("transfer into the temporary account: mint %s amount %s, want %s", x.Info.Mint, x.Info.TokenAmount.Amount, t22FeeMint)
			}
			return
		}
	}
	t.Error("transfer into the temporary account not found")
}

// hideTokenAccount returns a copy of a real "json" fixture in which nothing
// reveals the mint of token account dest except the source of the transfer
// into it: its token balance entries are dropped, the Token-2022
// transferChecked into it becomes a plain Transfer (3) and its
// InitializeAccount3 (18) becomes a no-op InitializeImmutableOwner (22), as in
// data without the account's creation.
func hideTokenAccount(t *testing.T, orig *adapter.SolanaTransaction, dest string) *adapter.SolanaTransaction {
	t.Helper()
	tx := cloneTx(t, orig)
	keys := rawAccountKeys(tx)
	drop := func(bals []adapter.TokenBalance) []adapter.TokenBalance {
		var out []adapter.TokenBalance
		for _, b := range bals {
			if keys[b.AccountIndex] != dest {
				out = append(out, b)
			}
		}
		return out
	}
	tx.Meta.PreTokenBalances = drop(tx.Meta.PreTokenBalances)
	tx.Meta.PostTokenBalances = drop(tx.Meta.PostTokenBalances)
	for si := range tx.Meta.InnerInstructions {
		for _, ix := range tx.Meta.InnerInstructions[si].Instructions {
			m := ix.(map[string]interface{})
			data, _ := base58.Decode(m["data"].(string))
			pid := jsonInt(m["programIdIndex"])
			if keys[pid] != constants.TOKEN_2022_PROGRAM_ID || len(data) == 0 {
				continue
			}
			switch {
			case len(data) == 10 && data[0] == constants.SPLTokenTransferChecked:
				accs := m["accounts"].([]interface{})
				m["accounts"] = []interface{}{accs[0], accs[2], accs[3]}
				m["data"] = base58.Encode(append([]byte{constants.SPLTokenTransfer}, data[1:9]...))
			case data[0] == 18:
				m["data"] = base58.Encode([]byte{22})
			}
		}
	}
	return tx
}

// TestCoreDetectBotNeedsFee: DetectBot attributed a bot when any of its fee
// accounts merely appeared in the tx (1-lamport dust tagged a trade as Axiom).
// A credit counts only when it is >= 10000 lamports or >= 0.1% of a trade leg
// in that mint. D9. constants-5.
func TestCoreDetectBotNeedsFee(t *testing.T) {
	p := dexparser.NewDexParser()
	dust := loadFixture(t, sigAxiomDust)
	const (
		axiomOne = "8m5GkL7nVy95G4YVUbs79z873oVKqg2afgKRmqxsiiRm"
		axiomTen = "DZfEurFKFtSbdWZsKSDTqpqsQgvXxmESpvRtXkAdgLwM"
	)
	for acct, want := range map[string]int64{axiomOne: 1, axiomTen: 10} {
		if constants.GetBotName(acct) != "Axiom" || lamportDelta(dust, acct).Int64() != want {
			t.Fatalf("fixture: %s is not an Axiom account credited %d lamports", acct, want)
		}
	}
	// The same credits on a 0.01 SOL leg (threshold 10000 lamports) are dust
	trade := types.TradeInfo{
		User:        "FiVhagUrpAQDn2QveV94Q84zx1VLUsRBRx6xPXEcTnRF",
		InputToken:  types.TokenInfo{Mint: "BxftAowY2dVa2h9KMqDTPk4oMxzU9k6uVbZuoorXpump", AmountRaw: "226925"},
		OutputToken: types.TokenInfo{Mint: "So11111111111111111111111111111111111111112", AmountRaw: "10000000"},
	}
	utils.NewTransactionUtils(adapter.NewTransactionAdapter(dust, nil)).DetectBot(&trade)
	if trade.Bot != "" {
		t.Errorf("dust credits on a 0.01 SOL leg: Bot = %q, want none", trade.Bot)
	}
	// The real trade sells for 1001 lamports through Axiom's router FLASHX8D;
	// the 10 lamports to DZfE are Axiom's 1% fee (>= 0.1% of the leg), the
	// 1 lamport to 8m5G alone would not count (< 2 lamports)
	if r := p.ParseAll(dust, nil); r.AggregateTrade == nil || r.AggregateTrade.OutputToken.AmountRaw != "1001" || r.AggregateTrade.Bot != "Axiom" {
		t.Errorf("1%% fee on a 1001-lamport leg: aggregate %v, want output 1001 and bot Axiom", r.AggregateTrade)
	}

	paid := loadFixture(t, sigAxiomPaid)
	const feeWallet = "DKyUs1xXMDy8Z11zNsLnUg3dy9HZf6hYZidB6WodcaGy"
	if constants.GetBotName(feeWallet) != "Axiom" || lamportDelta(paid, feeWallet).Int64() < utils.BotFeeMinLamports {
		t.Fatal("fixture: no Axiom fee payment")
	}
	if r := p.ParseAll(paid, nil); r.AggregateTrade == nil || r.AggregateTrade.Bot != "Axiom" {
		t.Errorf("paid fee: aggregate %v, want bot Axiom", r.AggregateTrade)
	}
}
