package tests

import (
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Regression tests for the DexParser parse flow (DESIGN D3). Expected amounts
// come from the transactions' token balances (ownerTokenDelta), not from the
// parser.

const (
	// Jupiter v6 route -> Pumpswap buy of BCdwQB (legacy route instruction)
	jupPumpswapMint = "BCdwQBAn8dYB5YjTsoB6TdHAWokxv28k2oZUodERpump"
	// Jupiter v6 route CL -> CPMM -> Moonit sell (meme event at 8-8)
	sigJupMoonit = "5mPd52BGevpvqU4XNaUuhMa7MqAvC8MXvmoYnAx1Za5XHsSQ6quU7orGYQ93CZtSTyWugnxv62g6yof7ahAUgzvN"
	// Jupiter v6 route -> Moonit buy
	sigJupMoonitBuy = "3TZKJLxy4H2wQiYenuSVoQ2ox7xveRoG5bxr7yfmEdtMPqKmdHWcf5Q9B8uUBi6ystp2gQsZdP5qxiYK4JnUpm7"
	// Jupiter DCA fill: keeper routes DCA -> JUP6 -> Manifest/GoonFi/Meteora DAMM v1
	sigDCAFill = "2TqzmwGqQrb7uHQEKm1TxhyHQJan7YhmaguP7cM9o7KNMX4u1ERWkwjuCzJvLv3BfgwSK3X9AfLNzZmSsEeUd3YF"
	// failed Jupiter arbitrage (InstructionError Custom 6001)
	sigFailedJup = "3v5TTW8pwaUdtJuh3vwgkKcg5JL8VHGTAGt5JZduhoS8yYcwGHpJTtgCBrNFaa7rRQaUnbpzqzxQeMWWq6cLcqxB"
)

// TestCoreJupiterTradesUnderDefaultConfig: with a nil config, the default
// config or a config without ParseType, Jupiter swaps used to return no
// trades (only AggregateTrade, then an early return). core-3, amm-9, parity-1.
func TestCoreJupiterTradesUnderDefaultConfig(t *testing.T) {
	p := dexparser.NewDexParser()
	defaultCfg := types.DefaultParseConfig()
	configs := map[string]*types.ParseConfig{
		"nil":                     nil,
		"DefaultParseConfig":      &defaultCfg,
		"TryUnknownDEX only":      {TryUnknownDEX: true},
		"legacy AggregateTrades":  {TryUnknownDEX: true, AggregateTrades: true},
		"ParseType Trade only":    {ParseType: types.ParseType{Trade: true}},
		"ParseType Trade and agg": {ParseType: types.ParseType{Trade: true, AggregateTrade: true}},
	}
	for name, cfg := range configs {
		t.Run(name, func(t *testing.T) {
			tx := loadFixture(t, sigJupPumpswap)
			trades := p.ParseTrades(tx, cfg)
			if len(trades) != 1 {
				t.Fatalf("ParseTrades = %d trades, want 1 (the Jupiter route)", len(trades))
			}
			tr := trades[0]
			user := tx.Transaction.Message.AccountKeys[0].Pubkey
			want := ownerTokenDelta(tx, user, jupPumpswapMint)
			if tr.ProgramId != constants.DEX_PROGRAMS.JUPITER.ID || tr.Type != types.TradeTypeBuy ||
				tr.InputToken.Mint != solMint || tr.OutputToken.Mint != jupPumpswapMint ||
				bigStr(tr.OutputToken.AmountRaw).Cmp(want) != 0 {
				t.Errorf("trade = %s, want JUP6 BUY SOL -> %s %s", tradeKey(tr), jupPumpswapMint, want)
			}

			result := p.ParseAll(tx, cfg)
			if len(result.Trades) != 1 {
				t.Errorf("ParseAll: %d trades, want 1", len(result.Trades))
			}
		})
	}
}

// TestCoreAggregationSwitches: AggregateTrades=false could not disable
// aggregation when ParseType is unset, and ParseType{AggregateTrade} alone
// returned nothing. core-14, parity-1.
func TestCoreAggregationSwitches(t *testing.T) {
	p := dexparser.NewDexParser()
	tx := loadFixture(t, sigJupPumpswap)

	r := p.ParseAll(tx, &types.ParseConfig{TryUnknownDEX: true, AggregateTrades: false})
	if r.AggregateTrade != nil || len(r.Trades) != 1 {
		t.Errorf("AggregateTrades=false: trades=%d aggregate=%v, want 1 trade and no aggregate", len(r.Trades), r.AggregateTrade != nil)
	}
	if len(r.MemeEvents) == 0 {
		t.Error("ParseType unset should still parse meme events")
	}

	r = p.ParseAll(tx, &types.ParseConfig{ParseType: types.ParseType{AggregateTrade: true}})
	if r.AggregateTrade == nil || len(r.Trades) != 0 {
		t.Fatalf("ParseType{AggregateTrade}: trades=%d aggregate=%v, want only the aggregate", len(r.Trades), r.AggregateTrade != nil)
	}
	user := tx.Transaction.Message.AccountKeys[0].Pubkey
	if r.AggregateTrade.OutputToken.Mint != jupPumpswapMint || bigStr(r.AggregateTrade.OutputToken.AmountRaw).Cmp(ownerTokenDelta(tx, user, jupPumpswapMint)) != 0 {
		t.Errorf("aggregate = %s", tradeKey(*r.AggregateTrade))
	}
	if len(r.MemeEvents) != 0 || len(r.Liquidities) != 0 {
		t.Error("ParseType{AggregateTrade} alone must not return other events")
	}
}

// TestCoreProgramFiltersApplyToJupiter: ProgramIds / IgnoreProgramIds were
// not applied in the Jupiter branch. core-14.
func TestCoreProgramFiltersApplyToJupiter(t *testing.T) {
	p := dexparser.NewDexParser()
	tx := loadFixture(t, sigJupPumpswap)

	trades := p.ParseTrades(tx, &types.ParseConfig{TryUnknownDEX: true, IgnoreProgramIds: []string{constants.DEX_PROGRAMS.JUPITER.ID}})
	if len(trades) != 1 || trades[0].ProgramId != constants.DEX_PROGRAMS.PUMP_SWAP.ID {
		t.Errorf("IgnoreProgramIds=[Jupiter]: %d trades %v, want only the Pumpswap trade", len(trades), trades)
	}
	trades = p.ParseTrades(tx, &types.ParseConfig{TryUnknownDEX: true, ProgramIds: []string{constants.DEX_PROGRAMS.PUMP_SWAP.ID}})
	if len(trades) != 1 || trades[0].ProgramId != constants.DEX_PROGRAMS.PUMP_SWAP.ID {
		t.Errorf("ProgramIds=[Pumpswap]: %d trades %v, want only the Pumpswap trade", len(trades), trades)
	}
	if len(trades) == 1 {
		// the Pumpswap leg delivers what the user received
		user := tx.Transaction.Message.AccountKeys[0].Pubkey
		if bigStr(trades[0].OutputToken.AmountRaw).Cmp(ownerTokenDelta(tx, user, jupPumpswapMint)) != 0 {
			t.Errorf("Pumpswap trade output %s", trades[0].OutputToken.AmountRaw)
		}
	}
}

// TestCoreJupiterKeepsOtherEvents: the Jupiter early return dropped meme
// events (and liquidity/ALT/transfer processing) of Jupiter-routed txs.
// meme-8, core-15.
func TestCoreJupiterKeepsOtherEvents(t *testing.T) {
	p := dexparser.NewDexParser()
	cases := []struct {
		sig, idx string
		typ      types.TradeType
		protocol string
	}{
		{sigJupMoonit, "8-8", types.TradeTypeSell, "Moonit"},
		{sigJupMoonitBuy, "5-6", types.TradeTypeBuy, "Moonit"},
		{sigJupPumpswap, "5-10", types.TradeTypeBuy, "Pumpswap"},
	}
	for _, c := range cases {
		tx := loadFixture(t, c.sig)
		r := p.ParseAll(tx, nil)
		if len(r.Trades) == 0 || r.AggregateTrade == nil {
			t.Errorf("%.8s: trades=%d aggregate=%v", c.sig, len(r.Trades), r.AggregateTrade != nil)
		}
		if len(r.MemeEvents) != 1 || r.MemeEvents[0].Idx != c.idx || r.MemeEvents[0].Type != c.typ || r.MemeEvents[0].Protocol != c.protocol {
			t.Errorf("%.8s: meme events %+v, want one %s %s at %s", c.sig, r.MemeEvents, c.protocol, c.typ, c.idx)
		}
	}
}

// TestCoreJupiterCoversNestedAMMs: when another route or bot program comes
// first, the Jupiter branch was skipped and both the Jupiter trade and the
// AMM trade under it were emitted, doubling the aggregate. core-15, amm-1.
//
// Synthetic: built from the real 5vjkR1vo by adding a BananaGun outer
// instruction in front (no real bot-wrapped Jupiter tx was found).
func TestCoreJupiterCoversNestedAMMs(t *testing.T) {
	p := dexparser.NewDexParser()
	orig := loadFixture(t, sigJupPumpswap)
	want := p.ParseAll(orig, nil).AggregateTrade
	if want == nil {
		t.Fatal("no aggregate on the original tx")
	}

	tx := cloneTx(t, orig)
	bot := constants.DEX_PROGRAMS.BANANA_GUN.ID
	tx.Meta.LoadedAddresses.Readonly = append(tx.Meta.LoadedAddresses.Readonly, bot)
	botIndex := len(rawAccountKeys(tx)) - 1
	tx.Transaction.Message.Instructions = append([]interface{}{
		map[string]interface{}{"programIdIndex": botIndex, "accounts": []interface{}{}, "data": ""},
	}, tx.Transaction.Message.Instructions...)
	for i := range tx.Meta.InnerInstructions {
		tx.Meta.InnerInstructions[i].Index++
	}

	r := p.ParseAll(tx, nil)
	if len(r.Trades) != 1 || r.Trades[0].Idx != "6-11" || r.Trades[0].ProgramId != constants.DEX_PROGRAMS.JUPITER.ID {
		t.Fatalf("trades = %v, want only the Jupiter trade at 6-11", r.Trades)
	}
	agg := r.AggregateTrade
	if agg == nil || agg.InputToken.AmountRaw != want.InputToken.AmountRaw || agg.OutputToken.AmountRaw != want.OutputToken.AmountRaw {
		t.Errorf("aggregate = %v, want amounts %s/%s (not doubled)", agg, want.InputToken.AmountRaw, want.OutputToken.AmountRaw)
	}
	if agg != nil && agg.Route != "BananaGun" {
		t.Errorf("aggregate route = %q, want BananaGun (first program in execution order)", agg.Route)
	}
}

// TestCoreDCAFillCountedOnce: a Jupiter DCA fill (DCA -> JUP6 -> AMMs) used
// to emit the DCA trade, the JUP6 hop trades and the raw AMM trades, tripling
// the aggregate, with route MeteoraVault. amm-1.
func TestCoreDCAFillCountedOnce(t *testing.T) {
	tx, r := parseFixture(t, sigDCAFill, nil)
	if len(r.Trades) != 1 {
		t.Fatalf("trades = %d %v, want 1 (the DCA fill)", len(r.Trades), r.Trades)
	}
	tr := r.Trades[0]
	// the DCA escrow (the DCA account) paid 100 USDC for this fill
	const dcaOwner = "53EuvfbVxy2hCdPCw5ZhueguuemfsRpJi4pTevxAZn7G"
	const outMint = "2Wu1g2ft7qZHfTpfzP3wLdfPeV1is4EwQ3CXBfRYAciD"
	if tr.ProgramId != constants.DEX_PROGRAMS.JUPITER_DCA.ID || tr.User != dcaOwner || tr.Route != "JupiterDCA" ||
		tr.InputToken.Mint != usdcMint || tr.InputToken.AmountRaw != "100000000" ||
		tr.OutputToken.Mint != outMint || tr.OutputToken.AmountRaw != "1231863" {
		t.Errorf("trade = %s route=%s", tradeKey(tr), tr.Route)
	}
	// independent check: the keeper's route moved exactly 100 USDC out of the
	// DCA escrow token account
	escrowSpent := false
	for _, b := range tx.Meta.PreTokenBalances {
		if b.Mint == usdcMint {
			key := rawAccountKeys(tx)[b.AccountIndex]
			if accountTokenDelta(tx, key, usdcMint).String() == "-100000000" {
				escrowSpent = true
			}
		}
	}
	if !escrowSpent {
		t.Error("no USDC account lost exactly 100000000 in the fixture")
	}
	if r.AggregateTrade == nil || r.AggregateTrade.InputToken.AmountRaw != "100000000" || r.AggregateTrade.OutputToken.AmountRaw != "1231863" {
		t.Errorf("aggregate = %v", r.AggregateTrade)
	}
}

// TestCoreFailedTransactions: trades of failed (reverted) transactions were
// reported with State=true. core-6.
func TestCoreFailedTransactions(t *testing.T) {
	p := dexparser.NewDexParser()
	tx := loadFixture(t, sigFailedJup)
	if tx.Meta.Err == nil {
		t.Fatal("fixture is not a failed transaction")
	}

	r := p.ParseAll(tx, nil)
	if !r.State || r.Msg != "transaction failed" || r.TxStatus != types.TransactionStatusFailed {
		t.Errorf("State=%v Msg=%q TxStatus=%s", r.State, r.Msg, r.TxStatus)
	}
	if len(r.Trades) != 0 || r.AggregateTrade != nil || len(r.Liquidities) != 0 || len(r.MemeEvents) != 0 || len(r.Transfers) != 0 {
		t.Errorf("failed tx returned events: trades=%d agg=%v", len(r.Trades), r.AggregateTrade != nil)
	}
	if r.Fee.Amount != "6000" || r.Signer[0] != "ENY9JreWWWtq8jSKfLtXpzjBteknLPdahPCwG7ZCVbJe" || r.Signature != sigFailedJup {
		t.Errorf("fee=%s signer=%v signature=%s, want them filled", r.Fee.Amount, r.Signer, r.Signature)
	}
	if got := r.SolBalanceChange; got == nil || got.Change.Amount != "-6000" {
		t.Errorf("SolBalanceChange = %+v, want -6000 (the fee)", got)
	}
	if len(p.ParseTrades(tx, nil)) != 0 || len(p.ParseTransfers(tx, nil)) != 0 {
		t.Error("ParseTrades/ParseTransfers on a failed tx should be empty")
	}

	// IncludeFailedTxs restores the previous behaviour
	trades := p.ParseTrades(tx, &types.ParseConfig{TryUnknownDEX: true, IncludeFailedTxs: true})
	if len(trades) == 0 {
		t.Error("IncludeFailedTxs: no trades")
	}
}

// TestCoreParseResultNeverEmptyByAggregation: ParseTrades never returns empty
// because aggregation is on, for any Jupiter fixture that has trades. D3.
func TestCoreParseResultNeverEmptyByAggregation(t *testing.T) {
	p := dexparser.NewDexParser()
	trades := 0
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		all := p.ParseAll(tx, nil)
		if all.AggregateTrade == nil {
			continue
		}
		trades++
		if len(all.Trades) == 0 {
			t.Errorf("%.12s: aggregate without trades", sig)
		}
		if len(p.ParseTrades(tx, nil)) == 0 {
			t.Errorf("%.12s: ParseTrades empty while ParseAll has an aggregate", sig)
		}
	}
	if trades < 100 {
		t.Errorf("only %d fixtures with an aggregate", trades)
	}
}
