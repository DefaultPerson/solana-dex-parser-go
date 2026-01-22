package tests

import (
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

func TestParseType(t *testing.T) {
	// Test ParseAll returns all options enabled
	pt := types.ParseAll()
	if !pt.AggregateTrade {
		t.Error("ParseAll should have AggregateTrade enabled")
	}
	if !pt.Trade {
		t.Error("ParseAll should have Trade enabled")
	}
	if !pt.Liquidity {
		t.Error("ParseAll should have Liquidity enabled")
	}
	if !pt.Transfer {
		t.Error("ParseAll should have Transfer enabled")
	}
	if !pt.MemeEvent {
		t.Error("ParseAll should have MemeEvent enabled")
	}
	if !pt.AltEvent {
		t.Error("ParseAll should have AltEvent enabled")
	}
}

func TestParseTradesOnly(t *testing.T) {
	pt := types.ParseTradesOnly()
	if !pt.AggregateTrade {
		t.Error("ParseTradesOnly should have AggregateTrade enabled")
	}
	if !pt.Trade {
		t.Error("ParseTradesOnly should have Trade enabled")
	}
	if pt.Liquidity {
		t.Error("ParseTradesOnly should have Liquidity disabled")
	}
	if pt.Transfer {
		t.Error("ParseTradesOnly should have Transfer disabled")
	}
	if pt.MemeEvent {
		t.Error("ParseTradesOnly should have MemeEvent disabled")
	}
	if pt.AltEvent {
		t.Error("ParseTradesOnly should have AltEvent disabled")
	}
}

func TestParseLiquidityOnly(t *testing.T) {
	pt := types.ParseLiquidityOnly()
	if pt.AggregateTrade {
		t.Error("ParseLiquidityOnly should have AggregateTrade disabled")
	}
	if pt.Trade {
		t.Error("ParseLiquidityOnly should have Trade disabled")
	}
	if !pt.Liquidity {
		t.Error("ParseLiquidityOnly should have Liquidity enabled")
	}
}

func TestDefaultParseConfig(t *testing.T) {
	config := types.DefaultParseConfig()

	// Default should have ParseAll enabled
	if !config.ParseType.Trade {
		t.Error("DefaultParseConfig should have Trade enabled")
	}
	if !config.TryUnknownDEX {
		t.Error("DefaultParseConfig should have TryUnknownDEX enabled")
	}
}

func TestIsParseTypeSet(t *testing.T) {
	// Empty config
	config := types.ParseConfig{}
	if config.IsParseTypeSet() {
		t.Error("Empty config should return false for IsParseTypeSet")
	}

	// Config with Trade set
	config.ParseType.Trade = true
	if !config.IsParseTypeSet() {
		t.Error("Config with Trade=true should return true for IsParseTypeSet")
	}
}

func TestGetEffectiveParseType(t *testing.T) {
	// Empty config should return ParseAll
	config := types.ParseConfig{}
	effective := config.GetEffectiveParseType()
	if !effective.Trade || !effective.Liquidity || !effective.MemeEvent {
		t.Error("Empty config should return ParseAll for GetEffectiveParseType")
	}

	// Config with specific ParseType set
	config.ParseType.Trade = true
	effective = config.GetEffectiveParseType()
	if !effective.Trade {
		t.Error("Config with Trade=true should return Trade=true")
	}
	if effective.Liquidity {
		t.Error("Config with only Trade=true should return Liquidity=false")
	}
}

func TestShouldAggregateTrades(t *testing.T) {
	// Empty config with ParseType.AggregateTrade
	config := types.ParseConfig{}
	config.ParseType.AggregateTrade = true
	if !config.ShouldAggregateTrades() {
		t.Error("Config with ParseType.AggregateTrade=true should aggregate trades")
	}

	// Legacy AggregateTrades field
	config2 := types.ParseConfig{}
	config2.AggregateTrades = true
	if !config2.ShouldAggregateTrades() {
		t.Error("Config with AggregateTrades=true should aggregate trades")
	}
}

func TestParseResultIsArbitrage(t *testing.T) {
	// No aggregate trade
	result := types.NewParseResult()
	if result.IsArbitrage() {
		t.Error("Result without aggregate trade should not be arbitrage")
	}

	// Aggregate trade with same input/output
	result.AggregateTrade = &types.TradeInfo{
		InputToken:  types.TokenInfo{Mint: "So11111111111111111111111111111111111111112"},
		OutputToken: types.TokenInfo{Mint: "So11111111111111111111111111111111111111112"},
	}
	if !result.IsArbitrage() {
		t.Error("Result with same input/output mint should be arbitrage")
	}

	// Aggregate trade with different input/output
	result.AggregateTrade.OutputToken.Mint = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	if result.IsArbitrage() {
		t.Error("Result with different input/output mint should not be arbitrage")
	}
}

func TestAccountFiltering(t *testing.T) {
	config := types.ParseConfig{
		AccountInclude: []string{"account1", "account2"},
		AccountExclude: []string{"account3"},
	}

	if len(config.AccountInclude) != 2 {
		t.Errorf("Expected 2 AccountInclude, got %d", len(config.AccountInclude))
	}
	if len(config.AccountExclude) != 1 {
		t.Errorf("Expected 1 AccountExclude, got %d", len(config.AccountExclude))
	}
}
