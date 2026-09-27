package tests

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"math/big"
	"reflect"
	"strings"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/propamm"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Titan and OKX DEX Router V2 routes: Trades stay the venues' hop trades,
// and the aggregate trade is the aggregator's own route total, i.e. what
// the user sent and received (their token balance changes), with the
// aggregator's fee in Fee. Before, the aggregate summed the hops: what the
// pools paid out, before the fee the aggregator kept.
func TestIntegAggregatorRouteAggregate(t *testing.T) {
	cases := []struct {
		name, sig, idx, user string
		program              constants.DexProgram
		hops                 int
		inMint, inAmount     string
		outMint, outAmount   string
		feeMint, feeAmount   string
		feeType              string
		checkIn, checkOut    bool
	}{
		// Titan keeps fee_c (99989 USDC): the hop paid out 999897114
		{"titan single hop", "4C2p65nuttUqBv5VhLNXHq6cNEaexT5i4YGbGSH2hodRGLBQyJBzRCkr2VXPKPNof8Lut8WSm51tkxVv3mvUkpuZ", "2", "EX9NvTr9xcZy9QEgZYpyBeKFkxR4y7jdAzEHRoYnMDye",
			constants.DEX_PROGRAMS.TITAN, 1, venueUSDT, "1000011964", venueUSDC, "999797125", "", "", "", true, true},
		// Split over SolFi V2 and Manifest; the hops paid out 1000117997
		{"titan split", "4X2GpZMxpeRKzE3KPzqa2ZPbNVt4He4G6L9Z7a7kSze5oMAKjFheTuPYkxRtBUi2dgkk8z9KRysx1ZEfhFCJSa9y", "2", "EX9NvTr9xcZy9QEgZYpyBeKFkxR4y7jdAzEHRoYnMDye",
			constants.DEX_PROGRAMS.TITAN, 2, venueUSDC, "999999000", venueUSDT, "1000017986", "", "", "", true, true},
		// Output-side fee_a of 2495 USDC to an integrator fee account
		{"titan output fee", "24FsRUqwK7CSax3qqwhXwgUzQvZ2MZ7MonsYm1RLx92S9pisUTNT6SUEEuuNoqTriZQCuao1Lhgso1bz4wcSQAWb", "4", "3hEp7vMMZugfynHhPBqNyLmApaspwbE8JJLpT94FcKhY",
			constants.DEX_PROGRAMS.TITAN, 1, venueCBTC, "29592", venueUSDC, "24956038", venueUSDC, "2495", "platform", true, true},
		// Titan invoked by another program (2-1); the fee went to the user
		{"titan inner", "ZoQtiC3w65x9v6vZkKsP4KpKZHKENGYis4gNDMQovycr4ZUhKXLHw2p7B9hVz6DyiPjf8o3nFPLyurs5yp1gJKL", "2-1", "D9Y7KCHciPeieqaxiubMYm8Tg8a4Je8ziCkLJsYMznVK",
			constants.DEX_PROGRAMS.TITAN, 1, venueCBTC, "1272", venueUSDC, "1074088", "", "", "", true, true},
		// Input commission 1275120 USDC: the hops got 148739084 (split over
		// BisonFi 26773035 and Raydium CLMM 121966049)
		{"okx input commission", "262n1pEgfLq9G87vRNBsb5xKdmADSQxFGWXXb2qXkxqtsPViu5ifNWC2UsGCFpo1aXaagpsetjxNZKoR2mG6D3xw", "5", "DaYgWwWaqiQKhGLFJHLizxjTatqDaLNnLXWj25yNetGW",
			constants.DEX_PROGRAMS.OKX_DEX_V2, 3, venueUSDC, "150014204", "Ce2gx9KGXJ6C9Mp5b5x1sn9Mg87JwEbrQby4Zqo3pump", "3667578810", venueUSDC, "1275120", "commission", true, true},
		// Output commission 7472162 USDC: the last hop paid out 879077963
		{"okx output commission", "4tzaGfjUtBPNKaSJcZWcWbUeVyMj3aEsQ6XXpq341dJWUEYAFK7MX5jCcDKkCNuZRNLqBhEBMMVf8APdRXHtX4rT", "6", "2PhRZsgSFjeUjNS7i15xipuGEytfjr2AnoCdbJaVhhVr",
			constants.DEX_PROGRAMS.OKX_DEX_V2, 5, "8RVBk8vxLiUHueLUW1f4izFVqN3nWippLhkohKg6EGkS", "102474923269", venueUSDC, "871605801", venueUSDC, "7472162", "commission", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := loadFixture(t, tc.sig)
			result := dexparser.NewDexParser().ParseAll(tx, nil)
			if !result.State {
				t.Fatalf("parse failed: %s", result.Msg)
			}
			if len(result.Trades) != tc.hops {
				t.Errorf("%d trades, want the %d hops", len(result.Trades), tc.hops)
			}
			for _, hop := range result.Trades {
				if hop.ProgramId == tc.program.ID {
					t.Errorf("trade %s is the route, want only hops", hop.Idx)
				}
			}
			agg := result.AggregateTrade
			if agg == nil {
				t.Fatal("no aggregate trade")
			}
			if agg.Idx != tc.idx || agg.ProgramId != tc.program.ID || agg.Route != tc.program.Name || agg.User != tc.user {
				t.Errorf("aggregate idx %s program %s route %s user %s, want %s %s %s %s", agg.Idx, agg.ProgramId, agg.Route, agg.User, tc.idx, tc.program.ID, tc.program.Name, tc.user)
			}
			if agg.InputToken.Mint != tc.inMint || agg.InputToken.AmountRaw != tc.inAmount ||
				agg.OutputToken.Mint != tc.outMint || agg.OutputToken.AmountRaw != tc.outAmount {
				t.Errorf("aggregate %s %s -> %s %s, want %s %s -> %s %s", agg.InputToken.AmountRaw, agg.InputToken.Mint,
					agg.OutputToken.AmountRaw, agg.OutputToken.Mint, tc.inAmount, tc.inMint, tc.outAmount, tc.outMint)
			}
			if tc.checkIn {
				if d := ownerTokenDelta(tx, tc.user, tc.inMint); new(big.Int).Neg(d).String() != agg.InputToken.AmountRaw {
					t.Errorf("user input change %s, aggregate input %s", d, agg.InputToken.AmountRaw)
				}
			}
			if tc.checkOut {
				if d := ownerTokenDelta(tx, tc.user, tc.outMint); d.String() != agg.OutputToken.AmountRaw {
					t.Errorf("user output change %s, aggregate output %s", d, agg.OutputToken.AmountRaw)
				}
			}
			if tc.feeAmount == "" {
				if agg.Fee != nil {
					t.Errorf("unexpected fee %+v", agg.Fee)
				}
				return
			}
			if agg.Fee == nil || agg.Fee.Mint != tc.feeMint || agg.Fee.AmountRaw != tc.feeAmount ||
				agg.Fee.Type != tc.feeType || agg.Fee.Dex != tc.program.Name {
				t.Errorf("fee %+v, want %s %s type %s dex %s", agg.Fee, tc.feeAmount, tc.feeMint, tc.feeType, tc.program.Name)
			}
			if len(agg.Fees) == 0 || agg.Fees[0] != *agg.Fee {
				t.Errorf("Fees %+v do not start with the aggregator fee", agg.Fees)
			}
		})
	}
}

// Titan's fee_a is a transfer Titan itself makes (a direct child of the
// route instruction) out of the route, never a hop's transfer or the user's
// input into Titan's intermediate account that happen to carry the same
// amount. Synthetic: 24FsRUqw... (fee_a 2495 USDC paid at 4-4) with fee_a in
// the swap event set to 29592, the amount of the user's cbBTC transfer into
// Titan (4-0) and of the GoonFi V2 hop's input (4-2, inside the venue).
// Before, the first transfer with that amount anywhere in the route was
// taken as the fee.
func TestIntegTitanFeeIsTitansOwnTransfer(t *testing.T) {
	tx := loadFixture(t, "24FsRUqwK7CSax3qqwhXwgUzQvZ2MZ7MonsYm1RLx92S9pisUTNT6SUEEuuNoqTriZQCuao1Lhgso1bz4wcSQAWb")
	const prefix = "Program data: "
	found := false
	for i, l := range tx.Meta.LogMessages {
		if !strings.HasPrefix(l, prefix) {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(l[len(prefix):])
		if err != nil || len(data) < 56 || !bytes.HasPrefix(data, constants.DISCRIMINATORS.TITAN.SWAP_EVENT) {
			continue
		}
		if feeA := binary.LittleEndian.Uint64(data[32:40]); feeA != 2495 {
			t.Fatalf("fee_a %d, want 2495", feeA)
		}
		binary.LittleEndian.PutUint64(data[32:40], 29592)
		tx.Meta.LogMessages[i] = prefix + base64.StdEncoding.EncodeToString(data)
		found = true
	}
	if !found {
		t.Fatal("no Titan swap event")
	}

	ctx := newParseContext(tx, nil)
	trades := propamm.NewTitanParser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions,
		ctx.Classifier.GetInstructions(constants.DEX_PROGRAMS.TITAN.ID)).ProcessTrades()
	if len(trades) != 1 {
		t.Fatalf("want 1 route trade, got %d", len(trades))
	}
	if f := trades[0].Fee; f != nil {
		t.Errorf("fee %s %s taken from a transfer that is not a fee", f.AmountRaw, f.Mint)
	}
}

// A program with a registered route parser is not read by the unknown-DEX
// fallback: its transfer groups hold the user's transfers to and from the
// aggregator, not a swap. Shown with a program the fallback turns into a
// trade (satRush in 47CbsEer..., "Unknown" 2-4) and a route parser that
// reports nothing.
func TestIntegRouteProgramSkipsUnknownDEX(t *testing.T) {
	const sig = "47CbsEeriB5JtFmSVU61NS1thpD1HabJ7AGzzwxiViQqUNQJxrJXtaBU1EeBCLWtbQE7whvKVBjxXa2ELCMc16wp"
	const satRush = "satRushGBRY2vgapeTAkoxz26vL2cYqyPi6CnBj7Tco"
	count := func(p *dexparser.DexParser) int {
		n := 0
		for _, tr := range p.ParseAll(loadFixture(t, sig), nil).Trades {
			if tr.ProgramId == satRush {
				n++
			}
		}
		return n
	}
	if count(dexparser.NewDexParser()) == 0 {
		t.Fatal("fixture: no unknown-DEX trade of satRush")
	}
	p := dexparser.NewDexParser()
	p.RegisterRouteParser(satRush, func(*adapter.TransactionAdapter, types.DexInfo, map[string][]types.TransferData, []types.ClassifiedInstruction) parsers.TradeParser {
		return noTrades{}
	})
	if n := count(p); n != 0 {
		t.Errorf("%d unknown-DEX trades of a route program", n)
	}
}

type noTrades struct{}

func (noTrades) ProcessTrades() []types.TradeInfo { return nil }

// routeTrades returns the route trades of program in the fixture sig
func routeTrades(t *testing.T, sig string, program constants.DexProgram) []types.TradeInfo {
	t.Helper()
	ctx := newParseContext(loadFixture(t, sig), nil)
	instructions := ctx.Classifier.GetInstructions(program.ID)
	switch program.ID {
	case constants.DEX_PROGRAMS.TITAN.ID:
		return propamm.NewTitanParser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions, instructions).ProcessTrades()
	case constants.DEX_PROGRAMS.OKX_DEX_V2.ID:
		return propamm.NewOKXV2Parser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions, instructions).ProcessTrades()
	}
	t.Fatalf("no route parser for %s", program.Name)
	return nil
}

// A route trade lists its fee in Fee and in Fees. When the aggregate is
// built from more than one trade (two routes, or a route and a trade outside
// it), each fee is listed once. Before, utils.FeeComponents dropped only a
// Fee that was the untyped total of Fees, so every route listed its typed
// fee twice. The route trades are real (OKX input commission 1275120 USDC
// in 262n1pEg..., Titan fee_a 2495 USDC in 24FsRUqw...); their combination
// is not: no fixture has two routes or a route next to another trade.
func TestIntegRouteFeeListedOnce(t *testing.T) {
	okx := routeTrades(t, "262n1pEgfLq9G87vRNBsb5xKdmADSQxFGWXXb2qXkxqtsPViu5ifNWC2UsGCFpo1aXaagpsetjxNZKoR2mG6D3xw", constants.DEX_PROGRAMS.OKX_DEX_V2)
	titan := routeTrades(t, "24FsRUqwK7CSax3qqwhXwgUzQvZ2MZ7MonsYm1RLx92S9pisUTNT6SUEEuuNoqTriZQCuao1Lhgso1bz4wcSQAWb", constants.DEX_PROGRAMS.TITAN)
	if len(okx) != 1 || len(titan) != 1 || okx[0].Fee == nil || titan[0].Fee == nil {
		t.Fatalf("want one route trade with a fee each: OKX %+v, Titan %+v", okx, titan)
	}
	for _, route := range []types.TradeInfo{okx[0], titan[0]} {
		if !reflect.DeepEqual(route.Fees, []types.FeeInfo{*route.Fee}) {
			t.Fatalf("%s route: Fees %+v, want its Fee %+v", route.Route, route.Fees, *route.Fee)
		}
		if got := utils.FeeComponents(&route); !reflect.DeepEqual(got, []types.FeeInfo{*route.Fee}) {
			t.Errorf("%s route: fee components %+v, want its fee once", route.Route, got)
		}
	}

	second := titan[0]
	second.Idx = "9" // after the OKX route
	outside := okx[0]
	outside.Idx, outside.Fee, outside.Fees = "9", nil, nil
	for name, trades := range map[string][]types.TradeInfo{
		"two routes":        {okx[0], second},
		"route and a trade": {okx[0], outside},
	} {
		agg := utils.GetFinalSwap(trades, nil)
		var want []types.FeeInfo
		for _, tr := range trades {
			if tr.Fee != nil {
				want = append(want, *tr.Fee)
			}
		}
		if !reflect.DeepEqual(agg.Fees, want) {
			t.Errorf("%s: aggregate Fees %+v, want %+v", name, agg.Fees, want)
		}
	}
}

// Titan's fee_a recipient is the owner of the receiving token account (the
// integrator's wallet), like the other parsers' fee recipients, not the
// token account. In 24FsRUqw... the 2495 USDC went to 9sz8Xggo..., owned by
// FdTEHJ9f... (token balances). And the user-kept check uses that resolved
// owner: a caller's transfer without DestinationOwner that pays fee_a to an
// account of the user leaves the fee in what the user received. Synthetic
// for the second part: the fee account's owner set to the user, and the
// transfers passed to the parser without DestinationOwner.
func TestIntegTitanFeeRecipientIsOwner(t *testing.T) {
	const (
		sig        = "24FsRUqwK7CSax3qqwhXwgUzQvZ2MZ7MonsYm1RLx92S9pisUTNT6SUEEuuNoqTriZQCuao1Lhgso1bz4wcSQAWb"
		feeAccount = "9sz8XggoqbCR68vaojcRPxYxhbHGvaYmdzW7ChGdyap9"
		user       = "3hEp7vMMZugfynHhPBqNyLmApaspwbE8JJLpT94FcKhY"
	)
	tx := loadFixture(t, sig)
	keys := rawAccountKeys(tx)
	owner := ""
	for _, b := range tx.Meta.PostTokenBalances {
		if b.AccountIndex < len(keys) && keys[b.AccountIndex] == feeAccount {
			owner = b.Owner
		}
	}
	if owner == "" || owner == user {
		t.Fatalf("fixture: fee account owner %q", owner)
	}
	agg := dexparser.NewDexParser().ParseAll(tx, nil).AggregateTrade
	if agg == nil {
		t.Fatal("no aggregate trade")
	}
	if agg.Fee == nil || agg.Fee.AmountRaw != "2495" || agg.Fee.Recipient != owner {
		t.Errorf("aggregate fee %+v, want 2495 to %s", agg.Fee, owner)
	}

	tx = cloneTx(t, tx)
	for _, balances := range [][]adapter.TokenBalance{tx.Meta.PreTokenBalances, tx.Meta.PostTokenBalances} {
		for i := range balances {
			if balances[i].AccountIndex < len(keys) && keys[balances[i].AccountIndex] == feeAccount {
				balances[i].Owner = user
			}
		}
	}
	ctx := newParseContext(tx, nil)
	for _, transfers := range ctx.TransferActions {
		for i := range transfers {
			transfers[i].Info.DestinationOwner = ""
		}
	}
	trades := propamm.NewTitanParser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions,
		ctx.Classifier.GetInstructions(constants.DEX_PROGRAMS.TITAN.ID)).ProcessTrades()
	if len(trades) != 1 {
		t.Fatalf("want 1 route trade, got %d", len(trades))
	}
	if tr := trades[0]; tr.Fee != nil || tr.OutputToken.AmountRaw != "24958533" {
		t.Errorf("fee %+v, output %s; want no fee and 24956038 + 2495", tr.Fee, tr.OutputToken.AmountRaw)
	}
}
