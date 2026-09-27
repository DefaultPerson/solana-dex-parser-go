package tests

import (
	"math/big"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Raydium LaunchLab regression tests. Expected values come from decoding the
// TradeEvent / PoolCreateEvent with the raydium_launchpad IDL and from the
// raw transfers of the fixtures.

const (
	sigLaunchLabUSD1Buy  = "3d6W6W6GXDKHVYnJdWA2kQar2dxiYD9iN6ySFWimVQXq7cR1pNHN4WXVcJkgf5tmEQR7Q9NKHPNEWGdiHGwUh8Cd"
	sigLaunchLabUSD1Sell = "zuaKyxjpM7G5et2XqZofjjGNczNduGs6g8ipCEeZKKV7h6FFgRNJbXnzfufSZWD3bEacmf8sVktXpZaadQhmVuJ"
	sigLaunchLabSOLSell  = "5UrQjjQ1SbHRHkaZww9JpQZ28prYAFJjtFnvMsieqVj9b6Ry6U22SnEJvvVKwwYgGT9cMSDz9VBi4CrDkPzBE2aZ"
	sigLaunchLabCreate   = "4x8k2aQKevA8yuCVX1V8EaH2GBqdbZ1dgYxwtkwZJ7SmCQeng7CCs17AvyjFv6nMoUkBgpBwLHAABdCxGHbAWxo4"
	sigLaunchLabV1Events = "61AN23VGPknSqskF6CvtZqrD4LtL2CNGYKeyFc5nLVcfaUaHV5LsQe6HTnRFM6pNX8qf7fkqZ5tEZfnNEF73H8MX"
)

// meme-2: quote decimals were hardcoded to 9 (USD1 pairs 1000x off), the fee
// raw amount went through a float with 9 decimals, and Pool was always empty.
func TestMemeLaunchLabTrades(t *testing.T) {
	cases := []struct {
		sig, idx     string
		outer, inner int // trade instruction; its transfers follow it up to inner+5
		// routed: the outer instruction is a Jupiter v6 route. Under D3 the
		// Jupiter trade is the trade at idx (no LaunchLab pool or fee), so
		// pool and fee are checked on the LaunchLab meme event only.
		routed bool
		typ    types.TradeType
		quote  string // mint
		amount string // quote amount the user paid / received
		fee    string // protocol + platform + creator + share (IDL decode)
		pool   string // pool_state
	}{
		{sigLaunchLabUSD1Buy, "2-1", 2, 1, false, types.TradeTypeBuy, usd1Mint, "424499", "6368", "4qvv7cbNA1P7J8NG4i6yBNyJXthYntbPwxEJuJGVfBpU"},
		{sigLaunchLabUSD1Sell, "5-0", 5, 0, true, types.TradeTypeSell, usd1Mint, "85749244", "1305827", "7jey32kZWJDp1uUMiHa9yuXQT8Cr4rTmR3a8kjGgwwnY"},
		{sigLaunchLabSOLSell, "1-0", 1, 0, false, types.TradeTypeSell, solMint, "3361960", "51198", "8hEdzYqEGRjyFgE8k16kwrMZ8ciQ1coooJxgnd6f7P2b"},
	}
	for _, c := range cases {
		r := memeParse(t, c.sig)
		tr := memeTradeAt(t, r, c.idx)
		e := memeEventAt(t, r, c.idx)
		raw := loadMemeRaw(t, c.sig)
		acc := raw.accounts(c.outer, c.inner)
		quoteSide, memeQuoteSide := &tr.OutputToken, e.OutputToken
		// user_quote_token (6) sends the quote on buys and receives it on sells
		moved := raw.sumTransfers(c.outer, c.inner+1, c.inner+5, func(x memeRawTransfer) bool { return x.Mint == c.quote && x.To == acc[6] })
		if c.typ == types.TradeTypeBuy {
			quoteSide, memeQuoteSide = &tr.InputToken, e.InputToken
			moved = raw.sumTransfers(c.outer, c.inner+1, c.inner+5, func(x memeRawTransfer) bool { return x.Mint == c.quote && x.From == acc[6] })
		}
		if moved.String() != c.amount {
			t.Fatalf("%.8s: fixture quote transfer %s, table %s", c.sig, moved, c.amount)
		}
		if tr.Type != c.typ {
			t.Errorf("%.8s: trade %s, want %s", c.sig, tr.Type, c.typ)
		}
		checkToken(t, c.sig[:8]+" quote", quoteSide, c.quote, c.amount, raw.decimals[c.quote])
		if !c.routed {
			if len(tr.Pool) != 1 || tr.Pool[0] != c.pool {
				t.Errorf("%.8s: trade pool %v, want %s", c.sig, tr.Pool, c.pool)
			}
			if tr.Fee == nil || tr.Fee.AmountRaw != c.fee || tr.Fee.Mint != c.quote || tr.Fee.Decimals != raw.decimals[c.quote] {
				t.Errorf("%.8s: Fee %+v, want %s", c.sig, tr.Fee, c.fee)
			}
			if len(e.Fees) != len(tr.Fees) {
				t.Errorf("%.8s: meme event fees %+v, trade fees %+v", c.sig, e.Fees, tr.Fees)
			}
		}

		// the LaunchLab meme event, routed or not
		if e.Type != c.typ || e.Pool != c.pool || e.PlatformConfig != acc[3] {
			t.Errorf("%.8s: meme event %s pool %s platform %s", c.sig, e.Type, e.Pool, e.PlatformConfig)
		}
		checkToken(t, c.sig[:8]+" meme quote", memeQuoteSide, c.quote, c.amount, raw.decimals[c.quote])
		feeSum := new(big.Int)
		for _, f := range e.Fees {
			v, ok := new(big.Int).SetString(f.AmountRaw, 10)
			if !ok || f.Mint != c.quote || f.Decimals != raw.decimals[c.quote] {
				t.Errorf("%.8s: meme event fee component %+v", c.sig, f)
				continue
			}
			feeSum.Add(feeSum, v)
		}
		if feeSum.String() != c.fee {
			t.Errorf("%.8s: meme event fees %+v sum %s, want %s", c.sig, e.Fees, feeSum, c.fee)
		}
	}

	// the platform fee of the USD1 buy is paid out of the quote vault
	raw := loadMemeRaw(t, sigLaunchLabUSD1Buy)
	tr := memeTradeAt(t, memeParse(t, sigLaunchLabUSD1Buy), "2-1")
	platform := raw.sumTransfers(2, 2, 6, func(x memeRawTransfer) bool { return x.Mint == usd1Mint && x.From == raw.accounts(2, 1)[8] })
	if feeOf(tr.Fees, "platform") != platform.String() || feeOf(tr.Fees, "protocol") != "1062" {
		t.Errorf("USD1 buy Fees %+v, platform transfer %s", tr.Fees, platform)
	}
}

// 130-byte TradeEvents (before creator_fee) keep decoding.
func TestMemeLaunchLabV1TradeEvent(t *testing.T) {
	r := memeParse(t, sigLaunchLabV1Events)
	buy := memeTradeAt(t, r, "5")
	sell := memeTradeAt(t, r, "6")
	if buy.Type != types.TradeTypeBuy || buy.InputToken.AmountRaw != "16080810" || buy.Fee == nil || buy.Fee.AmountRaw != "160809" ||
		sell.Type != types.TradeTypeSell || sell.OutputToken.AmountRaw != "15760800" || sell.Fee == nil || sell.Fee.AmountRaw != "159200" {
		t.Errorf("v1 buy %+v / sell %+v", buy, sell)
	}
}

// meme-16: curve params were read with migrate_type first; the IDL order is
// (supply, [total_base_sell], total_quote_fund_raising, migrate_type).
func TestMemeLaunchLabCreateCurve(t *testing.T) {
	e := memeEventAt(t, memeParse(t, sigLaunchLabCreate), "0-7")
	want := types.MemeCurveParams{Type: "Constant", Supply: "1000000000000000", TotalBaseSell: "793100000000000",
		TotalQuoteFundRaising: "85000000000", MigrateType: 1, TotalLockedAmount: "0", CliffPeriod: "0", UnlockPeriod: "0"}
	if e.Type != types.TradeTypeCreate || e.Curve == nil || *e.Curve != want || e.Decimals == nil || *e.Decimals != 6 ||
		e.BaseMint != "25phz2ZHEfB81RQXKvNLvkDbK32nyUwFnQdqk6MLcook" || e.QuoteMint != solMint ||
		e.Pool != "CPTNvVYT7qCzX3HnRRtSRAFpMipVgSP3eynXrW9p9YgD" || e.PlatformConfig != "2G15pADKW42tWNXcZLdm1FP34o1kjnoY6XvbYqBSGvQV" {
		t.Errorf("create event %+v curve %+v", e, e.Curve)
	}
}

// meme-16: the create event took the mints from the OUTER instruction, which
// is another program's instruction when initialize is called through CPI.
// Synthetic: 4x8k2a with its outer initialize moved into a CPI under a
// wrapper instruction (no CPI-created LaunchLab pool was found on mainnet).
func TestMemeLaunchLabCreateViaCPI(t *testing.T) {
	tx := cloneTx(t, loadFixture(t, sigLaunchLabCreate))
	msg := tx.Transaction.Message
	init := msg.Instructions[0].(map[string]interface{})
	initCopy := map[string]interface{}{}
	for k, v := range init {
		initCopy[k] = v
	}
	accounts := init["accounts"].([]interface{})
	reversed := make([]interface{}, len(accounts))
	for i := range accounts {
		reversed[i] = accounts[len(accounts)-1-i]
	}
	// wrapper: the ATA program's index, other accounts, no data meaning
	msg.Instructions[0] = map[string]interface{}{"programIdIndex": msg.Instructions[1].(map[string]interface{})["programIdIndex"], "accounts": reversed, "data": "2"}
	// initialize now runs at stack height 2, the instructions it invoked one
	// level deeper
	set := &tx.Meta.InnerInstructions[0]
	for _, ix := range set.Instructions {
		if m, ok := ix.(map[string]interface{}); ok {
			if h := adapter.InstructionStackHeight(m); h > 0 {
				m["stackHeight"] = float64(h + 1)
			}
		}
	}
	initCopy["stackHeight"] = float64(2)
	set.Instructions = append([]interface{}{initCopy}, set.Instructions...)

	r := dexparserParse(tx)
	if r == nil || !r.State {
		t.Fatalf("parse failed %+v", r)
	}
	var create *types.MemeEvent
	for i := range r.MemeEvents {
		if r.MemeEvents[i].Type == types.TradeTypeCreate {
			create = &r.MemeEvents[i]
		}
	}
	if create == nil || create.Idx != "0-8" || create.BaseMint != "25phz2ZHEfB81RQXKvNLvkDbK32nyUwFnQdqk6MLcook" || create.QuoteMint != solMint {
		t.Errorf("CPI create event %+v", create)
	}
}

// meme-19: LaunchLab events were not sorted at all; the event parser now
// returns them in execution order (outer instructions keep FormatIdx "5").
func TestMemeLaunchLabEventParserOrder(t *testing.T) {
	ctx := newParseContext(loadFixture(t, sigLaunchLabV1Events), nil)
	events := raydium.NewRaydiumLaunchpadEventParser(ctx.Adapter, ctx.TransferActions).ProcessEvents()
	if len(events) != 2 || events[0].Idx != "5" || events[1].Idx != "6" {
		t.Errorf("events %+v", events)
	}
}
