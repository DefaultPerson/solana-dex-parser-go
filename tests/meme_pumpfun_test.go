package tests

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Pump.fun regression tests. Expected event fields come from decoding the
// events with the pump.json IDL (pump-fun/pump-public-docs 8109141, the
// audit's evdec.py); expected user amounts from the raw transfer
// instructions of the fixtures.

const (
	sigPumpCreateV2BuyV2 = "5zMo16ECF3UwYWzj8kG7ccYavPjs3oMMJGkcDuvgiyUsqHxFG1suvzyF6T4CoWrgbWryiiFw4oSitkdmUsa8mZae"
	sigPumpSellLegacy    = "2PHFQkMMWdVHunTzLN7KK6kYRMk3oYvp2sHTSQyB6v29KzPvK8NTfYLzLPbgek5EZZEyJC7KKJppGHjpX1HTuzKu"
	sigPumpSellV2        = "2DdZiRsXJWVkD1nD31bcBQpZPKmmmd1gG1T4GAFcLDw6f7Zpxao876E6pPcreUqMmKLWjmxo5FRvq6WynU2pPXaW"
	sigPumpUSDCSell      = "4gfccDphF5CgAYGQ6zWRAGeQ4wJge85CtQiBRqw1wyu7geDhMAgYCBMdK1jGAoHHE9pzyE6bMyHSuwnRv2ms5KaB"
	sigPumpUSDCSell2     = "zbKixjo7JSvbLuqCWQ5SmZtePpbsNPP9HDP2oxMC7vFFrsT1TWqreYqnvDE5s57JX5NuTqrfsBv5Gjux1VuUfKc"
	sigPumpUSDCCreateBuy = "3MVawF6EPtG7rEPXdsyQfQUBLv3epRVNpNS4tRE4uwTPMqLNPqhuABwxU3QZH4uD6CuVupcpGchpNRK5HTbHRLNK"
	sigPumpCashbackBuy   = "H6azwLqtRtrnVNC5iwcjYM9idU3e9SRyLZXTwjfJGJxA4X7dZL7vyhFAJNvQy7bb6bmQNmFHUt1KkkPPmhdge3G"
	sigPumpShareholders  = "2DbpcY2at3eWL3Mm6KBQbrSSMB9d4qMa5NsrpZyuFciVVs49poqLNFNv9mymZmpA9LKSKzcThGK4h5McBPU2bWKV"
	sigPumpLegacyEvent   = "2CYBHseAoZy1WHTNnVj1cTV9gnDeXE5WHAq6xXP62RL6h54uN1ft1AM1r5VkhMXYtav54CaP4nbR2rDe5TZdPzbR"
	sigPumpMigrate160    = "5fiQbExgdp1FAjDnrv9aEpXajCMUtm1c3E7NnsDdu4CtKr5xBpALdK7ENzx5LN1SzZKJk7cxbWWc84T7yHwb8p2x"
	sigPumpMigrate192    = "3ck1kPo6VzPsahWM1wpotwCfFR3yW5B8ATwDPptfVEVQCjmXRWtm44XiAgaojJHwgnEyckrX2YHkFCEz3MMVUXwM"
)

// meme-3, parity-11: fee and creator_fee were read 16 bytes early (virtual
// reserves not consumed, u64 basis points read as u16), giving ~1e10 SOL.
func TestMemePumpfunTradeEventFees(t *testing.T) {
	cases := []struct {
		sig, idx                  string
		fee, buyback, creatorFee  uint64 // IDL decode: fee (incl. buyback), buyback_fee, creator_fee
		feeRecipient, feeCreator  string
		protocolFeeUI, creatorFUI float64
	}{
		{sigPumpCreateV2BuyV2, "4-5", 562963, 0, 177778, "463MEnMeGyJekNZFQSTUABBEbLnvMTALbT6ZmsxAbAdq", "CF4pLmAtLC4xVtQDsxeXbGVVnXmhdo5vDXSh5EQMLzfj", 0.000562963, 0.000177778},
		{sigPumpSellLegacy, "0-3", 479217, 239608, 151332, "7VtfL8fvgNfhz17qKRMjzQEXgbdpnHHHQRh54R9jP2RJ", "9z1j9xAzS4EKZgVSnfMtnFCMTBepBSKMbDUcaMFFjyJ9", 0.000479217, 0.000151332},
	}
	for _, c := range cases {
		r := memeParse(t, c.sig)
		e := memeEventAt(t, r, c.idx)
		if e.ProtocolFee == nil || *e.ProtocolFee != c.protocolFeeUI || e.CreatorFee == nil || *e.CreatorFee != c.creatorFUI {
			t.Errorf("%.8s: ProtocolFee %v CreatorFee %v, want %v %v", c.sig, e.ProtocolFee, e.CreatorFee, c.protocolFeeUI, c.creatorFUI)
		}
		if got, want := feeOf(e.Fees, "protocol"), u64s(c.fee-c.buyback); got != want {
			t.Errorf("%.8s: protocol fee %s, want %s", c.sig, got, want)
		}
		if c.buyback > 0 && feeOf(e.Fees, "buyback") != u64s(c.buyback) {
			t.Errorf("%.8s: buyback fee %s, want %d", c.sig, feeOf(e.Fees, "buyback"), c.buyback)
		}
		if feeOf(e.Fees, "coinCreator") != u64s(c.creatorFee) {
			t.Errorf("%.8s: creator fee %s, want %d", c.sig, feeOf(e.Fees, "coinCreator"), c.creatorFee)
		}
		for _, f := range e.Fees {
			if f.Type == "protocol" && f.Recipient != c.feeRecipient || f.Type == "coinCreator" && f.Recipient != c.feeCreator {
				t.Errorf("%.8s: %s fee recipient %s", c.sig, f.Type, f.Recipient)
			}
		}
		tr := memeTradeAt(t, r, c.idx)
		if tr.Fee == nil || tr.Fee.AmountRaw != u64s(c.fee+c.creatorFee) || tr.Fee.Mint != solMint {
			t.Errorf("%.8s: trade Fee %+v, want %d SOL", c.sig, tr.Fee, c.fee+c.creatorFee)
		}
		if len(tr.Fees) != len(e.Fees) {
			t.Errorf("%.8s: trade Fees %+v, meme Fees %+v", c.sig, tr.Fees, e.Fees)
		}
	}
}

// meme-4 and the core request on 4gfccDph/zbKixjo7: USDC-quoted bonding
// curves were reported as SOL with amount sol_amount (0) and 9 decimals.
func TestMemePumpfunUSDCQuote(t *testing.T) {
	cases := []struct {
		sig, user string
		outer     int
		quote     uint64 // IDL decode: quote_amount (sol_amount is 0)
		fee, cf   uint64
	}{
		{sigPumpUSDCSell, "CkXGYJJYy9JXa9fn9r4vw2ntsCsjUAiBZJ3dnLx3i6bq", 5, 2528219, 24019, 7585},
		{sigPumpUSDCSell2, "", 5, 177731328, 1688448, 533194},
	}
	for _, c := range cases {
		r := memeParse(t, c.sig)
		tr := memeTradeAt(t, r, "5-8")
		net := u64s(c.quote - c.fee - c.cf)
		if tr.Type != types.TradeTypeSell || tr.OutputToken.Mint != usdcMint || tr.OutputToken.AmountRaw != net || tr.OutputToken.Decimals != 6 {
			t.Errorf("%.8s: trade %s out %s %s dec %d, want SELL for %s USDC dec 6", c.sig, tr.Type, tr.OutputToken.Mint, tr.OutputToken.AmountRaw, tr.OutputToken.Decimals, net)
		}
		// the user's USDC account is credited exactly that by the bonding curve
		raw := loadMemeRaw(t, c.sig)
		got := raw.sumTransfers(c.outer, 0, 8, func(x memeRawTransfer) bool {
			return x.Mint == usdcMint && raw.owner[x.To] == tr.User
		})
		if got.String() != net {
			t.Errorf("%.8s: user USDC credit in the Pump.fun instruction %s, trade output %s", c.sig, got, net)
		}
		e := memeEventAt(t, r, "5-8")
		if e.QuoteMint != usdcMint || e.OutputToken == nil || e.OutputToken.AmountRaw != net || e.Fees[0].Mint != usdcMint {
			t.Errorf("%.8s: meme event quote %s out %+v fees %+v", c.sig, e.QuoteMint, e.OutputToken, e.Fees)
		}
	}

	// create_v2 on a USDC pair followed by buy_v2: quote mint from the
	// CreateEvent, the buyer pays quote + fee + cashback in USDC
	r := memeParse(t, sigPumpUSDCCreateBuy)
	create := memeEventAt(t, r, "2-19")
	if create.Type != types.TradeTypeCreate || create.QuoteMint != usdcMint || !create.IsCashbackEnabled {
		t.Errorf("3MVawF create: %s quote %s cashback %v", create.Type, create.QuoteMint, create.IsCashbackEnabled)
	}
	tr := memeTradeAt(t, r, "5-6")
	raw := loadMemeRaw(t, sigPumpUSDCCreateBuy)
	paid := raw.sumTransfers(5, 0, 6, func(x memeRawTransfer) bool { return x.Mint == usdcMint && x.Authority == tr.User })
	if tr.InputToken.Mint != usdcMint || tr.InputToken.AmountRaw != paid.String() || paid.String() != u64s(295262701+2804996+885789) {
		t.Errorf("3MVawF buy input %s %s, user paid %s", tr.InputToken.Mint, tr.InputToken.AmountRaw, paid)
	}
	if tr.Pool[0] != "2uqtaJK2Nx8E6oc9U3LeEgKPQkMxtoeVztnA3JQFTmQx" || create.BondingCurve != tr.Pool[0] {
		t.Errorf("3MVawF pool %v, create bonding curve %s", tr.Pool, create.BondingCurve)
	}
}

// meme-5: for buy_v2/sell_v2/buy_exact_quote_in_v2 accounts[3] is the base
// token program; the bonding curve is accounts[10].
func TestMemePumpfunPoolV2(t *testing.T) {
	const curve = "BGd1vHrg7Cf6iQtKgdeaJoXA1nSNsE1wvtb4L6KmYhfh" // IDL: sell_v2 bonding_curve
	r := memeParse(t, sigPumpSellV2)
	tr := memeTradeAt(t, r, "3-3")
	if len(tr.Pool) != 1 || tr.Pool[0] != curve {
		t.Errorf("sell_v2 pool %v, want %s", tr.Pool, curve)
	}
	if e := memeEventAt(t, r, "3-3"); e.BondingCurve != curve || e.TokenProgram != constants.TOKEN_PROGRAM_ID {
		t.Errorf("sell_v2 meme bonding curve %s token program %s", e.BondingCurve, e.TokenProgram)
	}
	if pda := pdaOf(t, "bonding-curve", "EddcMDFnsvdSZ4Mksn8XyhT3ibPDkyVWYFBzzhfNpump"); pda != curve {
		t.Errorf("bonding-curve PDA %s, IDL account %s", pda, curve)
	}
}

// meme-6: with create_v2 (outer 2) and buy_v2 (outer 4) the previous
// instruction lookup hit the CreateEvent self-CPI and left Pool empty.
func TestMemePumpfunCreateAndBuyPool(t *testing.T) {
	const curve = "BHRi2zQpe4KnH5iXpCr963VCBoiHbgBcWN1y4tmNEcW9" // CreateEvent bonding_curve
	r := memeParse(t, sigPumpCreateV2BuyV2)
	tr := memeTradeAt(t, r, "4-5")
	if len(tr.Pool) != 1 || tr.Pool[0] != curve {
		t.Errorf("buy_v2 pool %v, want %s", tr.Pool, curve)
	}
	if create := memeEventAt(t, r, "2-22"); create.BondingCurve != curve {
		t.Errorf("create bonding curve %s", create.BondingCurve)
	}
	if pda := pdaOf(t, "bonding-curve", "E4YTGzonBMPD1UKTDYwPNqchFKp4nj1H72C7pJUipump"); pda != curve {
		t.Errorf("bonding-curve PDA %s", pda)
	}
}

// meme-12: trade amounts are what the user paid or received. Buys add the
// protocol fee, creator fee and cashback to the event's quote amount, sells
// deduct them; before, the curve amount (net of fees) was reported.
func TestMemePumpfunUserSideAmounts(t *testing.T) {
	buys := []struct {
		sig, idx, user string
		outer, from    int // native SOL transfers of the user in the Pump.fun instruction
		to             int
		want           uint64
	}{
		// buy_v2: sol_amount 59259258 + fee 562963 + creator_fee 177778
		{sigPumpCreateV2BuyV2, "4-5", "CF4pLmAtLC4xVtQDsxeXbGVVnXmhdo5vDXSh5EQMLzfj", 4, 0, 5, 59999999},
		// cashback coin: sol_amount 3e9 + fee 28500000 + cashback 9000000 (creator_fee 0)
		{sigPumpCashbackBuy, "4-6", "25jZ7EwnKfZo2DZgHM27pbU5Tf54PYG8jc7qNL3gtkxG", 4, 0, 6, 3037500000},
		// buy_exact_quote_in_v2 with a shareholders vec in the event
		{sigPumpShareholders, "2-8", "4uSM8GjFrVVB43onTSWpSzAnRWnFT9Re3Tob3ESKFtpx", 2, 1, 8, 99000000},
	}
	for _, c := range buys {
		r := memeParse(t, c.sig)
		tr := memeTradeAt(t, r, c.idx)
		raw := loadMemeRaw(t, c.sig)
		paid := raw.sumTransfers(c.outer, c.from, c.to, func(x memeRawTransfer) bool { return x.Mint == solMint && x.From == c.user })
		if tr.User != c.user || tr.InputToken.Mint != solMint || tr.InputToken.AmountRaw != u64s(c.want) || paid.String() != u64s(c.want) {
			t.Errorf("%.8s: user %.8s input %s %s, the user sent %s (want %d)", c.sig, tr.User, tr.InputToken.Mint, tr.InputToken.AmountRaw, paid, c.want)
		}
		if e := memeEventAt(t, r, c.idx); e.InputToken == nil || e.InputToken.AmountRaw != tr.InputToken.AmountRaw {
			t.Errorf("%.8s: meme input %+v differs from trade input %s", c.sig, e.InputToken, tr.InputToken.AmountRaw)
		}
	}
	if e := memeEventAt(t, memeParse(t, sigPumpCashbackBuy), "4-6"); feeOf(e.Fees, "cashback") != "9000000" || !e.IsCashbackEnabled {
		t.Errorf("cashback buy fees %+v cashback %v", e.Fees, e.IsCashbackEnabled)
	}

	// sells: sol_amount - fee - creator_fee (the curve pays lamports directly,
	// so there is no transfer to compare with)
	sells := []struct {
		sig, idx string
		want     uint64
	}{
		{sigPumpSellV2, "3-3", 11941360 - 113443 - 35825},
		{sigPumpSellLegacy, "0-3", 50443880 - 479217 - 151332},
	}
	for _, c := range sells {
		tr := memeTradeAt(t, memeParse(t, c.sig), c.idx)
		if tr.OutputToken.Mint != solMint || tr.OutputToken.AmountRaw != u64s(c.want) {
			t.Errorf("%.8s: output %s %s, want %d SOL", c.sig, tr.OutputToken.Mint, tr.OutputToken.AmountRaw, c.want)
		}
	}
}

// Legacy 121-byte TradeEvents (no fee fields) still decode. They end after
// real_sol_reserves and real_token_reserves (IDL offsets 105 and 113), which
// were decoded but not surfaced. The buy's fee is its transfer of 100000
// lamports to the fee recipient (see TestPolishPumpfunLegacyBuyFee).
func TestMemePumpfunLegacyTradeEvent(t *testing.T) {
	r := memeParse(t, sigPumpLegacyEvent)
	tr := memeTradeAt(t, r, "4-3")
	if tr.Type != types.TradeTypeBuy || tr.InputToken.AmountRaw != "10100000" || tr.OutputToken.AmountRaw != "357547484171" ||
		tr.Fee == nil || tr.Fee.AmountRaw != "100000" || len(tr.Fees) != 1 || len(tr.Pool) != 1 || tr.Pool[0] != "7L7PmfSpSEdZP6H5VA8C6S11yFShgG1BVJJuyp9aSuff" {
		t.Errorf("legacy buy %s in %s out %s fee %+v pool %v", tr.Type, tr.InputToken.AmountRaw, tr.OutputToken.AmountRaw, tr.Fee, tr.Pool)
	}

	payload := loadMemeRaw(t, sigPumpLegacyEvent).data(4, 3)
	if len(payload) != 16+121 || !bytes.Equal(payload[:16], constants.DISCRIMINATORS.PUMPFUN.TRADE_EVENT) {
		t.Fatalf("fixture: 4-3 is not a 121-byte TradeEvent (%d bytes)", len(payload))
	}
	payload = payload[16:]
	e := memeEventAt(t, r, "4-3")
	if e.VirtualQuoteReserves != u64s(binary.LittleEndian.Uint64(payload[89:])) || e.VirtualBaseReserves != u64s(binary.LittleEndian.Uint64(payload[97:])) ||
		e.RealQuoteReserves != u64s(binary.LittleEndian.Uint64(payload[105:])) || e.RealBaseReserves != u64s(binary.LittleEndian.Uint64(payload[113:])) ||
		e.RealQuoteReserves == "0" || e.CreatorFeeBps != nil || len(e.Fees) != 1 {
		t.Errorf("legacy event reserves virtual %s/%s real %s/%s, fees %+v", e.VirtualQuoteReserves, e.VirtualBaseReserves, e.RealQuoteReserves, e.RealBaseReserves, e.Fees)
	}
}

// parity-10: the original CompletePumpAmmMigrationEvent is 160 bytes and was
// dropped (length check 168); the current one is 192 bytes (+quote_mint).
func TestMemePumpfunMigrateEvent(t *testing.T) {
	cases := []struct {
		sig, idx, pool, curve string
		base, quote, fee      string
	}{
		{sigPumpMigrate160, "2-40", "4goR4vPYZNYnK8jxynPv1vW1F9LJZTcyBymK575ETxdT", "4HBrrXKRCQkVpSP6PHDMV1mkpFjdvAYBi77uT4WcpFow", "206900000000001", "84990359085", "15000001"},
		{sigPumpMigrate192, "1-49", "FaEYWevRwU5GsnX7w2To5zvSjndpAQyb2iWMFt3oaqkM", "GTg9MwCsTHWZoSWpgL7kApjPEoNb2ksthLkTvfNtpRFT", "206900000000000", "84990359056", "15000001"},
	}
	for _, c := range cases {
		e := memeEventAt(t, memeParse(t, c.sig), c.idx)
		if e.Type != types.TradeTypeMigrate || e.Pool != c.pool || e.BondingCurve != c.curve || e.PoolDex != constants.DEX_PROGRAMS.PUMP_SWAP.Name ||
			e.QuoteMint != solMint || e.BaseAmount != c.base || e.QuoteAmount != c.quote || e.MigrationFee != c.fee {
			t.Errorf("%.8s: migrate event %+v", c.sig, e)
		}
	}
}

// meme-15, parity-15, parity-26: new CreateEvent/TradeEvent fields (token
// program, mayhem, cashback, creator fee bps, reserves, ix_name).
func TestMemePumpfunEventFields(t *testing.T) {
	r := memeParse(t, sigPumpCreateV2BuyV2)
	create := memeEventAt(t, r, "2-22")
	if create.TokenProgram != constants.TOKEN_2022_PROGRAM_ID || !create.IsMayhemMode || create.IsCashbackEnabled || create.IsHolderReward ||
		create.CreatorFeeBps == nil || *create.CreatorFeeBps != 0 || create.QuoteMint != solMint ||
		create.VirtualBaseReserves != "1073000000000000" || create.VirtualQuoteReserves != "30000000000" || create.RealBaseReserves != "793100000000000" ||
		create.Creator != "CF4pLmAtLC4xVtQDsxeXbGVVnXmhdo5vDXSh5EQMLzfj" || create.TotalSupply == nil || *create.TotalSupply != 1e9 {
		t.Errorf("create_v2 event %+v", create)
	}
	buy := memeEventAt(t, r, "4-5")
	if buy.IxName != "buy" || !buy.IsMayhemMode || buy.CreatorFeeBps == nil || *buy.CreatorFeeBps != 30 ||
		buy.VirtualBaseReserves != "1070884672297204" || buy.VirtualQuoteReserves != "30059259258" ||
		buy.RealBaseReserves != "790984672297204" || buy.RealQuoteReserves != "59259258" || buy.TokenProgram != constants.TOKEN_2022_PROGRAM_ID {
		t.Errorf("buy_v2 event %+v", buy)
	}

	// 415-byte event with one shareholder: the fields after the vec decode
	e := memeEventAt(t, memeParse(t, sigPumpShareholders), "2-8")
	if e.IxName != "buy_exact_quote_in" || e.QuoteMint != solMint || e.VirtualQuoteReserves != "94775781089" || e.RealQuoteReserves != "64775781089" {
		t.Errorf("shareholders event %+v", e)
	}

	// holder-rewards coin (USDC pair)
	if e := memeEventAt(t, memeParse(t, sigPumpUSDCSell), "5-8"); !e.IsHolderReward || e.IxName != "sell" {
		t.Errorf("holder reward sell %+v", e)
	}
}

// Truncated or malformed event data never panics: decoding stops at the end
// of the data (or at a vec count larger than the data) and the fields read so
// far stay valid. Synthetic: built from the real 2DbpcY TradeEvent (415
// bytes, one shareholder) by cutting its data or corrupting the vec length.
func TestMemePumpfunTruncatedEvents(t *testing.T) {
	const outer, inner = 2, 8
	orig := loadFixture(t, sigPumpShareholders)
	payload := memeInnerData(t, orig, outer, inner)[16:]
	if len(payload) != 415 {
		t.Fatalf("TradeEvent payload %d bytes", len(payload))
	}
	const shareholdersAt = 305 // IDL offset of the shareholders vec length
	if payload[shareholdersAt] != 1 {
		t.Fatalf("shareholders count %d", payload[shareholdersAt])
	}
	variants := map[string][]byte{}
	for _, n := range []int{121, 150, 266, 271, 300, shareholdersAt + 2, 350, 414} {
		variants["cut"+u64s(uint64(n))] = append([]byte(nil), payload[:n]...)
	}
	huge := append([]byte(nil), payload...)
	huge[shareholdersAt], huge[shareholdersAt+1], huge[shareholdersAt+2], huge[shareholdersAt+3] = 0xff, 0xff, 0xff, 0x7f
	variants["hugeVec"] = huge

	for name, v := range variants {
		tx := cloneTx(t, orig)
		memeSetInnerData(t, tx, outer, inner, append(append([]byte(nil), constants.DISCRIMINATORS.PUMPFUN.TRADE_EVENT...), v...))
		res := dexparserParse(tx)
		if res == nil || !res.State {
			t.Errorf("%s: parse failed %+v", name, res)
			continue
		}
		var tr *types.TradeInfo
		for i := range res.Trades {
			if res.Trades[i].Idx == "2-8" {
				tr = &res.Trades[i]
			}
		}
		if tr == nil || tr.Type != types.TradeTypeBuy || tr.OutputToken.AmountRaw != "350763729068" || tr.OutputToken.Mint != "GW5xNXBtW9usH62Uimis4L5T1Fi5ceeMa9dupjTVpxaD" {
			t.Errorf("%s: trade %+v", name, tr)
			continue
		}
		// the quote mint and amount come from the fixed part when the quote
		// fields are cut off; SOL-paired, so the value is the same
		if tr.InputToken.Mint != solMint {
			t.Errorf("%s: input mint %s", name, tr.InputToken.Mint)
		}
	}
}

func dexparserParse(tx *adapter.SolanaTransaction) *types.ParseResult {
	return dexparser.NewDexParser().ParseAll(tx, nil)
}

// memeInnerData returns the decoded data of inner instruction (outer, inner)
func memeInnerData(t testing.TB, tx *adapter.SolanaTransaction, outer, inner int) []byte {
	t.Helper()
	for _, set := range tx.Meta.InnerInstructions {
		if set.Index == outer {
			m := set.Instructions[inner].(map[string]interface{})
			d, err := base58.Decode(m["data"].(string))
			if err != nil {
				t.Fatalf("data: %v", err)
			}
			return d
		}
	}
	t.Fatalf("no inner instructions for %d", outer)
	return nil
}

// memeSetInnerData replaces the data of inner instruction (outer, inner)
func memeSetInnerData(t testing.TB, tx *adapter.SolanaTransaction, outer, inner int, data []byte) {
	t.Helper()
	for _, set := range tx.Meta.InnerInstructions {
		if set.Index == outer {
			set.Instructions[inner].(map[string]interface{})["data"] = base58.Encode(data)
			return
		}
	}
	t.Fatalf("no inner instructions for %d", outer)
}

// pdaOf derives a Pump.fun PDA from a seed and a pubkey
func pdaOf(t testing.TB, seed, key string) string {
	t.Helper()
	b := mustDecode58(t, key)
	pda, _, err := utils.FindProgramAddress([][]byte{[]byte(seed), b}, constants.DEX_PROGRAMS.PUMP_FUN.ID)
	if err != nil {
		t.Fatalf("PDA: %v", err)
	}
	return pda
}

func mustDecode58(t testing.TB, s string) []byte {
	t.Helper()
	b, err := base58.Decode(s)
	if err != nil || len(b) != 32 {
		t.Fatalf("base58 %s: %v", s, err)
	}
	return b
}

// streamer item 4: when a CreateEvent carries no quote_mint, the quote mint
// of create_v2 is remaining account 16. Synthetic: the real USDC create_v2
// of 3MVawF with its CreateEvent cut right before quote_mint (all real
// create_v2 events carry it).
func TestMemePumpfunCreateV2QuoteFromAccounts(t *testing.T) {
	const outer, inner = 2, 19
	tx := cloneTx(t, loadFixture(t, sigPumpUSDCCreateBuy))
	data := memeInnerData(t, tx, outer, inner)
	usdc := mustDecode58(t, usdcMint)
	cut := -1
	for i := 16; i+32 <= len(data); i++ {
		if string(data[i:i+32]) == string(usdc) {
			cut = i
		}
	}
	if cut < 0 {
		t.Fatal("quote_mint not found in the CreateEvent")
	}
	memeSetInnerData(t, tx, outer, inner, data[:cut])
	r := dexparserParse(tx)
	var create *types.MemeEvent
	for i := range r.MemeEvents {
		if r.MemeEvents[i].Type == types.TradeTypeCreate {
			create = &r.MemeEvents[i]
		}
	}
	if create == nil || create.QuoteMint != usdcMint || create.CreatorFeeBps != nil {
		t.Errorf("create event %+v", create)
	}
	if raw := loadMemeRaw(t, sigPumpUSDCCreateBuy); raw.accounts(outer, -1)[16] != usdcMint {
		t.Errorf("create_v2 account 16 is %s", raw.accounts(outer, -1)[16])
	}
}
