package tests

import (
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// PumpSwap regression tests. Expected event fields come from decoding the
// events with the pump_amm.json IDL (pump-public-docs 8109141); expected user
// amounts from the raw transfers of the fixtures.

const (
	sigPumpswapBuy          = "3RBJqTFMecxueY3WrFzCPHcYgmVyri7p1SyuofNT9s94zePwScQPjSU9DoigsRozyqqStDcudR1wDdMHXhCGFLwe"
	sigPumpswapBuyExactIn   = "3DscdqDWQTVxhbN7vpFtw2spvidhUyNuf9pbKgpBDxvN9gR9dkovmfoWcJbohkcRY59dr7H3i33dxh3Tg36G71PE"
	sigPumpswapSell         = "4SoyMnqo4zNexyqUn5cGknanJabF2xQHAviePZvnrhobWZzAMsyHGrqPcPq4emzFSoThRyjkuGBqvPo8ZP4TPXFv"
	sigPumpswapUSDCBuy      = "3VzySjet8EzwJGrWznGv2TDK176Up2SCaHSqKG21YV8jB13PGDbgHi5cGw2BpVdmUE8R67NzW5h3bkgJmjiFzrdT"
	sigPumpswapLegacyBuy    = "2W1ScejYBFe6kS4VnTmj14qaEmeqiV1Rf6TXfQU9PZcJEW5ERqY19kSWWgLtdfVJKx1PCMBGvXiJWc65o59VNAtf"
	sigPumpswapBuyAndBurn   = "4hWfxmtVBFPmuN2wPwQkkZ2nr6NW9VYnKDmvVkF8SmQumVHShRHT3ahizq6j4PsgUMo4voRgPGZRWcVDKb4K34Sd"
	sigPumpswapCreatePoolV2 = sigPumpMigrate192
)

// meme-12: the buy input was quote_amount_in_with_lp_fee (without protocol
// and creator fees); it is now what the user paid (user_quote_amount_in, or
// quote_amount_in for buy_exact_quote_in), the same in TradeInfo and MemeEvent.
func TestMemePumpswapUserSideAmounts(t *testing.T) {
	cases := []struct {
		sig, idx string
		outer    int
		isBuy    bool
		want     uint64
	}{
		{sigPumpswapBuy, "6-6", 6, true, 4656852},          // user_quote_amount_in
		{sigPumpswapBuyExactIn, "5-6", 5, true, 310687},    // buy_exact_quote_in: quote_amount_in
		{sigPumpswapSell, "4-6", 4, false, 473485},         // user_quote_amount_out
		{sigPumpswapUSDCBuy, "3-6", 3, true, 31171},        // USDC pool
		{sigPumpswapLegacyBuy, "3-4", 3, true, 4821932231}, // 304-byte event, no creator fee
	}
	for _, c := range cases {
		r := memeParse(t, c.sig)
		tr := memeTradeAt(t, r, c.idx)
		raw := loadMemeRaw(t, c.sig)
		innerEvent := 0
		if _, err := sscanIdx(c.idx, &innerEvent); err != nil {
			t.Fatal(err)
		}
		var got string
		if c.isBuy {
			got = tr.InputToken.AmountRaw
			paid := raw.sumTransfers(c.outer, 0, innerEvent, func(x memeRawTransfer) bool {
				return x.Mint == tr.InputToken.Mint && x.Authority == tr.User
			})
			if paid.String() != u64s(c.want) {
				t.Errorf("%.8s: the user's quote transfers sum to %s, want %d", c.sig, paid, c.want)
			}
		} else {
			got = tr.OutputToken.AmountRaw
			// sell accounts (pump_amm.json): 6 = user_quote_token_account
			userQuoteATA := raw.accounts(c.outer, -1)[6]
			received := raw.sumTransfers(c.outer, 0, innerEvent, func(x memeRawTransfer) bool {
				return x.Mint == tr.OutputToken.Mint && x.To == userQuoteATA
			})
			if received.String() != u64s(c.want) {
				t.Errorf("%.8s: the user's quote credit is %s, want %d", c.sig, received, c.want)
			}
		}
		if got != u64s(c.want) {
			t.Errorf("%.8s: trade quote amount %s, want %d", c.sig, got, c.want)
		}
		e := memeEventAt(t, r, c.idx)
		quote := e.OutputToken
		if c.isBuy {
			quote = e.InputToken
		}
		if quote == nil || quote.AmountRaw != u64s(c.want) {
			t.Errorf("%.8s: meme event quote amount %+v, want %d", c.sig, quote, c.want)
		}
	}
}

// meme-11: the sell Fee had Amount = protocol + creator but AmountRaw =
// protocol only. Fee is now the exact sum of the fee components, and the
// buyback part of protocol_fee is listed separately.
func TestMemePumpswapSellFee(t *testing.T) {
	r := memeParse(t, sigPumpswapSell)
	tr := memeTradeAt(t, r, "4-6")
	// SellEvent: protocol_fee 238 (incl. buyback_fee 119), coin_creator_fee 238
	if tr.Fee == nil || tr.Fee.AmountRaw != "476" || tr.Fee.Amount != 4.76e-07 || tr.Fee.Mint != solMint {
		t.Errorf("sell Fee %+v, want 476 lamports", tr.Fee)
	}
	if feeOf(tr.Fees, "protocol") != "119" || feeOf(tr.Fees, "buyback") != "119" || feeOf(tr.Fees, "coinCreator") != "238" {
		t.Errorf("sell Fees %+v", tr.Fees)
	}
	// the protocol fee recipient's token account is credited the protocol part
	raw := loadMemeRaw(t, sigPumpswapSell)
	const recipientATA = "BWXT6RUhit9FfJQM3pBmqeFLPYmuxgmyhMGC5sGr8RbA"
	if got := raw.sumTransfers(4, 0, 6, func(x memeRawTransfer) bool { return x.To == recipientATA }); got.String() != "119" {
		t.Errorf("protocol fee recipient credited %s", got)
	}
}

// meme-v-1: the PumpSwap MemeEvent hardcoded 9/6 decimals, so USDC pools
// reported quote amounts and fees 1000x too small.
func TestMemePumpswapUSDCDecimals(t *testing.T) {
	r := memeParse(t, sigPumpswapUSDCBuy)
	e := memeEventAt(t, r, "3-6")
	checkToken(t, "meme input", e.InputToken, usdcMint, "31171", 6)
	if e.InputToken.Amount != 0.031171 || e.ProtocolFee == nil || *e.ProtocolFee != 0.000016 || e.CreatorFee == nil || *e.CreatorFee != 0.000217 {
		t.Errorf("USDC meme event amount %v protocolFee %v creatorFee %v", e.InputToken.Amount, e.ProtocolFee, e.CreatorFee)
	}
	tr := memeTradeAt(t, r, "3-6")
	if tr.InputToken.Decimals != 6 || tr.InputToken.Amount != e.InputToken.Amount || tr.OutputToken.Decimals != e.OutputToken.Decimals {
		t.Errorf("trade and meme event disagree: %+v / %+v", tr.InputToken, e.InputToken)
	}
}

// streamer item 20: boost_buy_and_burn emits a BuyEvent for the protocol's
// own buy; it is not a user trade. It becomes a BUY_AND_BURN meme event with
// the quote used and the base amount burned.
func TestMemePumpswapBoostBuyAndBurn(t *testing.T) {
	r := memeParse(t, sigPumpswapBuyAndBurn)
	if len(r.Trades) != 0 || r.AggregateTrade != nil {
		t.Errorf("buy-and-burn produced trades: %v", memeTradeIdxs(r.Trades))
	}
	if len(r.MemeEvents) != 1 {
		t.Fatalf("meme events %+v", r.MemeEvents)
	}
	e := r.MemeEvents[0]
	raw := loadMemeRaw(t, sigPumpswapBuyAndBurn)
	quote := raw.sumTransfers(2, 0, 2, func(x memeRawTransfer) bool { return x.Mint == solMint })
	burnData := raw.data(2, 1) // Token-2022 Burn: tag 8, u64 amount
	burned := uint64(0)
	for i := 8; i >= 1; i-- {
		burned = burned<<8 | uint64(burnData[i])
	}
	if e.Type != types.TradeTypeBuyAndBurn || e.Idx != "2-3" || e.BaseMint != "9obrHEVFKGDSna68jhZb1KzffZ8Jiv1rzL1iJep7pump" ||
		e.Pool != "FaEYWevRwU5GsnX7w2To5zvSjndpAQyb2iWMFt3oaqkM" || e.QuoteMint != solMint {
		t.Errorf("buy-and-burn event %+v", e)
	}
	checkToken(t, "quote used", e.InputToken, solMint, quote.String(), 9)
	checkToken(t, "base burned", e.OutputToken, e.BaseMint, u64s(burned), 6)
	if burnData[0] != 8 || e.OutputToken.AmountRaw != "217137970890" {
		t.Errorf("burn instruction %x, event burned %s", burnData[:1], e.OutputToken.AmountRaw)
	}
}

// streamer item 17: CreatePoolEvent fields appended for coin creator,
// mayhem mode, creator fee bps and holder rewards.
func TestMemePumpswapCreatePoolFields(t *testing.T) {
	e := memeEventAt(t, memeParse(t, sigPumpswapCreatePoolV2), "1-34")
	if e.Type != types.TradeTypeCreate || e.Creator != "4jb5KXet5Xq2nVoe8waoSPA1WzWz6vsRQBu5S7pyMRjN" || e.IsMayhemMode || e.IsHolderReward ||
		e.CreatorFeeBps == nil || *e.CreatorFeeBps != 0 || e.Decimals == nil || *e.Decimals != 6 || e.Pool != "FaEYWevRwU5GsnX7w2To5zvSjndpAQyb2iWMFt3oaqkM" {
		t.Errorf("create pool event %+v", e)
	}
}

// sscanIdx parses the inner index of an "outer-inner" idx
func sscanIdx(idx string, inner *int) (int, error) {
	for i := 0; i < len(idx); i++ {
		if idx[i] == '-' {
			n := 0
			for _, ch := range idx[i+1:] {
				n = n*10 + int(ch-'0')
			}
			*inner = n
			return 1, nil
		}
	}
	return 0, nil
}
