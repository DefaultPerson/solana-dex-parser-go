package tests

import (
	"math/big"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/jupiter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Jupiter v6 shred decoding. Expected amounts are read from the instruction
// bytes with the on-chain JUP6 IDL layouts: v1 routes put (amount u64,
// amount u64, slippage_bps u16, platform_fee_bps u8) after the route plan,
// token-ledger routes (quoted_out u64, slippage_bps u16, platform_fee_bps
// u8); unless a client appends bytes (see TestShredJupiterRouteV1TrailingBytes)
// they end the data. v2 routes start with (amount u64, amount u64,
// slippage_bps u16, platform_fee_bps u16, positive_slippage_bps u16).
// shred-1, shred-9, shred-10, shred-11, shred-22, shred-18, shred-30.

const (
	// shared_accounts_route (v1) at outer 3: token -> SOL
	sigJupSharedRoute = "2DaS55TwMuryadgsirCLRJywZ7rbBoLugXnicbJR2YFSSgn8PAvGN8Mhr9p4Ux8SEtgPzrRRuqV9T7XS7WWtdxvR"
	// route_with_token_ledger at outer 2: USDC -> SOL
	sigJupTokenLedger = "52X1pL1TpyLpmU5GuFYsAoXTwycAamDejjrXCT2a79kt8LzzD2tusdi9GRWt3DDdGad9cfGPxVCDY6kb2wb3Tw9i"
	// shared_accounts_route_v2 at outer 2: USDC -> token
	sigJupSharedRouteV2 = "1V8cnQVrAApjfY61PdnG1mCoj9ejqNDj5xKyEnNWUrzQtA1VN129zCVYmkUqTkuiTQzrmWY9FWQenQaBrXDWEXS"
)

// jupTrade returns the typed Jupiter trade at idx
func jupTrade(t *testing.T, res *types.ParseShredResult, idx string) (types.ParsedShredInstruction, *types.TradeInfo) {
	t.Helper()
	ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.JUPITER.ID, idx)
	if ins.Trade == nil {
		t.Fatalf("no trade at %s", idx)
	}
	return ins, ins.Trade
}

// minOut is the minimum output a quote allows with slippageBps
func minOut(quote uint64, slippageBps uint16) *big.Int {
	v := new(big.Int).Mul(new(big.Int).SetUint64(quote), big.NewInt(int64(10000-int(slippageBps))))
	return v.Div(v, big.NewInt(10000))
}

// TestShredJupiterRouteV1Tail: v1 routes read their amounts one byte off
// (in 10664523917613334580, out 2305843009213949458, slippage 3 on 5jNLWdMv)
// and route used the source token account as the input mint. shred-1,
// shred-11.
func TestShredJupiterRouteV1Tail(t *testing.T) {
	tx := loadFixture(t, sigJupRouteV1)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE)
	tail := ix.data[len(ix.data)-19:]
	in, quote, slippage := le64At(tail, 0), le64At(tail, 8), le16At(tail, 16)
	if in != 13358 || quote != 65409684 || slippage != 800 {
		t.Fatalf("fixture decode = %d %d %d, want 13358 65409684 800 (audit Python decode)", in, quote, slippage)
	}
	user := ix.accounts[1]
	outMint := ix.accounts[5]

	res := parseShred(t, tx, nil)
	ins, trade := jupTrade(t, res, "4")
	if trade.InputToken.AmountRaw != u64str(in) || trade.OutputToken.AmountRaw != u64str(quote) || trade.SlippageBps == nil || *trade.SlippageBps != int(slippage) {
		t.Errorf("amounts = in %s out %s slippage %v, want %d %d %d", trade.InputToken.AmountRaw, trade.OutputToken.AmountRaw, trade.SlippageBps, in, quote, slippage)
	}
	// The source token account is a temporary WSOL account: the mint comes
	// from the token balances, not the account address
	if trade.InputToken.Mint != constants.TOKENS.SOL || trade.InputToken.Mint == ix.accounts[2] {
		t.Errorf("input mint = %s, want SOL (source token account %s)", trade.InputToken.Mint, ix.accounts[2])
	}
	if trade.OutputToken.Mint != outMint || trade.User != user || trade.Type != types.TradeTypeBuy {
		t.Errorf("trade = %s %s -> %s by %s, want BUY SOL -> %s by %s", trade.Type, trade.InputToken.Mint, trade.OutputToken.Mint, trade.User, outMint, user)
	}
	if ins.InputAmountKind != types.ShredAmountExact || ins.OutputAmountKind != types.ShredAmountQuote {
		t.Errorf("amount kinds = %s/%s, want exact/quote", ins.InputAmountKind, ins.OutputAmountKind)
	}
	// The user paid in_amount plus the network fee, and received at least
	// the slippage-adjusted quote
	if paid := new(big.Int).Neg(lamportDelta(tx, user)); paid.Cmp(new(big.Int).SetUint64(in+tx.Meta.Fee)) != 0 {
		t.Errorf("user paid %s lamports, want in_amount + fee = %d", paid, in+tx.Meta.Fee)
	}
	if got := ownerTokenDelta(tx, user, outMint); got.Cmp(minOut(quote, slippage)) < 0 {
		t.Errorf("received %s, below the quote's minimum %s", got, minOut(quote, slippage))
	}
	// Decimals come from the transaction (D6)
	if trade.InputToken.Decimals != 9 || trade.OutputToken.Decimals != 6 {
		t.Errorf("decimals = %d/%d, want 9/6", trade.InputToken.Decimals, trade.OutputToken.Decimals)
	}
	// Typed trades carry the transaction context
	if trade.Signature != sigJupRouteV1 || trade.Slot != tx.Slot || trade.Idx != "4" || trade.Timestamp != *tx.BlockTime {
		t.Errorf("trade context = %s %d %s %d", trade.Signature, trade.Slot, trade.Idx, trade.Timestamp)
	}

	// Shared-accounts v1 route (id u8 first): token -> SOL
	tx = loadFixture(t, sigJupSharedRoute)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.SHARE_ACCOUNTS_ROUTE)
	tail = ix.data[len(ix.data)-19:]
	in, quote, slippage = le64At(tail, 0), le64At(tail, 8), le16At(tail, 16)
	res = parseShred(t, tx, nil)
	_, trade = jupTrade(t, res, utils.FormatIdx(ix.outer, -1))
	if trade.InputToken.AmountRaw != u64str(in) || trade.OutputToken.AmountRaw != u64str(quote) || *trade.SlippageBps != int(slippage) {
		t.Errorf("shared route amounts = %s %s %d, want %d %d %d", trade.InputToken.AmountRaw, trade.OutputToken.AmountRaw, *trade.SlippageBps, in, quote, slippage)
	}
	if trade.InputToken.Mint != ix.accounts[7] || trade.OutputToken.Mint != ix.accounts[8] || trade.User != ix.accounts[2] || trade.Type != types.TradeTypeSell {
		t.Errorf("shared route = %s %s -> %s by %s", trade.Type, trade.InputToken.Mint, trade.OutputToken.Mint, trade.User)
	}
	if got := ownerTokenDelta(tx, ix.accounts[2], ix.accounts[7]); got.Cmp(new(big.Int).Neg(new(big.Int).SetUint64(in))) != 0 {
		t.Errorf("user token delta %s, want -in_amount %d", got, in)
	}
}

// jupiterTrailingByteRoutes are the v1 route instructions of the fixtures
// whose data carries one byte after the arguments (arbitrage bots using
// DynamicV1 route steps; Anchor ignores the extra byte). Values from an
// independent forward Borsh decode of the data with the on-chain JUP6 IDL.
// 2nGCisM2, 3B6obLz1, 3TQUG4mM, 3iqy9mXd and 63YrUjW9 are failed arbitrage
// attempts (Jupiter error 6001, slippage exceeded) and are parsed with
// IncludeFailedTxs for their arguments.
var jupiterTrailingByteRoutes = []struct {
	sig, idx  string
	in, quote uint64
	slippage  uint16
}{
	{"22VFxreHs9cEvHoYiooU62iLsK76vJo6E1WGXsjFRcM7PwCrUUhNiBhvUMztPCdxCgyfcDCB79TUaRDEC2VPLYAF", "2", 248815402, 248815402, 0},
	{"2nGCisM2rPEsCiYSJf9nF3gTT2TAPwE8HnTZdF5hTTU4HdQPgAga5rbWp15ex2gjhGHrUh5WQoFEhtS7e6tMzDaP", "2", 1981207406, 1981207406, 0},
	{"3B6obLz1zWtKuaWXutuDM5JrQWmBaNRRPxWmKjgne3utvkzjeUNJiq8bBxdDrbSTbwQeEdoAwqitGTSx3K4Bv65W", "2", 1999106820, 1999106820, 0},
	{"3PdYUcfbEcdMk22ECoy6Rm9PBVXWU7TUDrXV86ePbuW55ucuoYx15Qd42S4Ja37vAWC3KVQeP9C62kKKSA1PXqeK", "2", 249994350, 249994350, 0},
	{"3TQUG4mMQHBqr4PwkLTJHCgjF8YK6zKAYPqucoxD3x3YSp3wzdvoQwj3oQNLS9wXf2111W4khBTtrCk77UJTeEQp", "2", 399946582, 399946582, 0},
	{"3iqy9mXdNDq5h25yu4YhV4CXabDMxXjoDT77XvZd1YH8o5M7HjuoRqxPQ3DqE8rpVeFVVd4Mp1brcqA76dV1nNmU", "2", 1981104831, 1981104831, 0},
	{"3pwNydeB4tYLfKKSHhUuXuc4bCVRg3yprB9Rvp2r76nVYmCE35iTZfz2MNr7sc7K6PEEKjmpd33RedtDr3EpyW5u", "2", 399948180, 399948180, 0},
	{"42R1nX4F9ru6RqcV24FN64ABpgX4g6iFRdD6aQ1eoaHcQhLLsx26rJGhS1RYX6XxnpAc1meHXmiXj3pk5REgLbaz", "2", 249992150, 249992150, 0},
	{"4icGS61HbeVVTNfNgHJbSsHfMawwm7XwZTw3WcVzNfpyQU4dR6YuGnS1TcySPRxEb9zYUcpX2tbZpzzxqy7648Vk", "2", 248824789, 248824789, 0},
	{"4ukv6RptTLbnNUUEyiURNGLnvMYJJr5a176N6Ls3vD5ppDx44GmjMW8MKogowNeKr93htRdeuwnLYM2nhVAo69tF", "2", 248824631, 248824631, 0},
	{"58sMwRAPd3P5Rx1kWmjG5Qx4JPUUebgXzH49HvH8Voex6F3dTL6d1kQ79hYNGw3DDBPjc8G6DjqBZLAQFrJHqHV9", "2", 498854111, 498854111, 0},
	{"5ZNgvprhpDXXDkgN1og1b5RfXUx7KbNbdxQwGt65dMNrPry9gC5Z5J2We1BbwwEEFj3xiAyyCKjz4kyaxrMo1NJU", "2", 399743501, 399743501, 0},
	{"5aYQQooT6XAxdoBGVkH9BQqYkFaJNdnRGhuGHG4jLYBpHC5PoPffpepyCZaKQGpuFdg76AssrTwLnEkW7rUnL48V", "2", 399902365, 399902365, 0},
	{"5bVS61KWqTJFGE83tEV8qcYvD2VpGdnYm4X8nf29jGL8oCVzPmBX31DpiHgxMdKyAYGC7pqabrW21Wqvys89w6CE", "2", 249998500, 249998500, 0},
	{"5taLhKa6CAsbst2Q8iGEhTVRmyXp4oMKSHbiGfL8RVeJrKzUzAnvtD1kT8SEHuTvdAWp14cTGLVhEHJoSSe94gju", "2", 399692248, 399692248, 0},
	{"63YrUjW9noejDkUUi9URvhpr8XiHCtJmteb6AHgkrAUPpMLCZtJupoAfVmJ7d9g8AtbEGH9RfG6fnWXHET11gNcG", "2", 1981151337, 1981151337, 0},
	{"VGufZu2U81ohKNt1bYZN6Yvf5nTwK3yCspJBxay99iHED9kxNShntatU5KoKTNzGj1iEwKtciAyUdPGzGFoR4zL", "2", 250001650, 250001650, 0},
	{"YQSsXmPHCmpNYuwT3J4gPnj2pLdG4phTCYUCBsSbN7MvByAPhqs6Pma5MJkHPWZzEW2dpG4DdK2UWaEzc1ZxCjW", "2", 248827292, 248827292, 0},
	{"jyt8hjNDDvKtbSDJ8ETzvSRb4AgPKZ9Nx4Dg7y7Mtbhs8SB73CwSqJCEFLuqAojCTSrCd58QHJxTVK8v8N1LgrL", "2", 248823466, 248823466, 0},
}

// firstTokenTransferFrom returns the amount of the first SPL Token transfer
// (Transfer or TransferChecked) out of source inside outer instruction outer
func firstTokenTransferFrom(t *testing.T, tx *adapterTx, outer int, source string) (uint64, bool) {
	t.Helper()
	for _, ix := range fixtureIxs(t, tx) {
		if ix.outer != outer || ix.inner < 0 || len(ix.data) < 9 || len(ix.accounts) == 0 || ix.accounts[0] != source {
			continue
		}
		if ix.programId != constants.TOKEN_PROGRAM_ID && ix.programId != constants.TOKEN_2022_PROGRAM_ID {
			continue
		}
		if ix.data[0] == 3 || ix.data[0] == 12 {
			return le64At(ix.data, 1), true
		}
	}
	return 0, false
}

// TestShredJupiterRouteV1TrailingBytes: the v1 arguments were read from the
// last 19 bytes of the data, so a route with a byte after its arguments gave
// garbage (22VFxreH: in 3026418949593945247, out 971935 instead of
// 248815402/248815402). The decoder now walks the route plan. shred-1.
func TestShredJupiterRouteV1TrailingBytes(t *testing.T) {
	for _, c := range jupiterTrailingByteRoutes {
		t.Run(c.sig[:12], func(t *testing.T) {
			tx := loadFixture(t, c.sig)
			ix := findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE)
			if utils.FormatIdx(ix.outer, -1) != c.idx {
				t.Fatalf("route at %d, want %s", ix.outer, c.idx)
			}
			// The arguments end one byte before the end of the data
			n := len(ix.data)
			if le64At(ix.data, n-20) != c.in || le64At(ix.data, n-12) != c.quote || le16At(ix.data, n-4) != c.slippage {
				t.Fatalf("fixture data does not hold the arguments at len-20")
			}
			// Execution: the source token account sent in_amount
			if sent, ok := firstTokenTransferFrom(t, tx, ix.outer, ix.accounts[2]); !ok || sent != c.in {
				t.Errorf("source account sent %d (found %v), want in_amount %d", sent, ok, c.in)
			}

			res := parseShred(t, tx, &types.ParseConfig{IncludeFailedTxs: true})
			_, trade := jupTrade(t, res, c.idx)
			if trade.InputToken.AmountRaw != u64str(c.in) || trade.OutputToken.AmountRaw != u64str(c.quote) || *trade.SlippageBps != int(c.slippage) {
				t.Errorf("amounts = in %s out %s slippage %d, want %d %d %d", trade.InputToken.AmountRaw, trade.OutputToken.AmountRaw, *trade.SlippageBps, c.in, c.quote, c.slippage)
			}
			ev := res.Instructions[constants.DEX_PROGRAMS.JUPITER.Name]
			if len(ev) != 1 {
				t.Fatalf("%d Jupiter events, want 1", len(ev))
			}
			data := ev[0].(*jupiter.JupiterShredInstruction).Data.(*jupiter.JupiterRouteData)
			if data.InputAmount != c.in || data.OutputAmount != c.quote || data.SlippageBps != c.slippage || data.PlatformFeeBps != 0 {
				t.Errorf("legacy event = %+v", data)
			}
		})
	}
}

// TestShredJupiterUnknownSwapVariant: a route step with a Swap variant newer
// than the decoder's table falls back to the argument tail; a malformed
// route plan gives no event. Synthetic: the Swap tag and an Option tag of
// real routes are overwritten, as no real transaction has them. shred-1.
func TestShredJupiterUnknownSwapVariant(t *testing.T) {
	// 5jNLWdMv (route, no trailing bytes): first Swap tag at data[12]
	tx := loadFixture(t, sigJupRouteV1)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE)
	data := append([]byte{}, ix.data...)
	data[12] = 0xff
	setIxData(ix, data)
	res := parseShred(t, tx, nil)
	_, trade := jupTrade(t, res, "4")
	if trade.InputToken.AmountRaw != "13358" || trade.OutputToken.AmountRaw != "65409684" || *trade.SlippageBps != 800 {
		t.Errorf("unknown variant fallback = %s %s %d, want 13358 65409684 800", trade.InputToken.AmountRaw, trade.OutputToken.AmountRaw, *trade.SlippageBps)
	}

	// 22VFxreH: the first step is DynamicV1 (tag 111) with one BisonFiV2
	// candidate; its best_position Option tag is at data[19]
	c := jupiterTrailingByteRoutes[0]
	tx = loadFixture(t, c.sig)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE)
	if ix.data[12] != 111 || ix.data[17] != 7 || ix.data[19] != 1 {
		t.Fatalf("unexpected route plan head %x", ix.data[12:20])
	}
	data = append([]byte{}, ix.data...)
	data[19] = 2
	setIxData(ix, data)
	res = parseShred(t, tx, nil)
	if got := typedAt(res, constants.DEX_PROGRAMS.JUPITER.ID, c.idx); len(got) != 0 {
		t.Errorf("route with an invalid Option tag decoded: %+v", got[0].Trade)
	}
	if ev := res.Instructions[constants.DEX_PROGRAMS.JUPITER.Name]; len(ev) != 0 {
		t.Errorf("legacy events for an invalid route plan: %d", len(ev))
	}
}

// TestShredCircularRouteIsSwap: an arbitrage route that starts and ends in
// the same mint was reported as BUY (USDC -> USDC on 22VFxreH, SOL -> SOL on
// 2ZeNbKRU). G2 of the shred verifier.
func TestShredCircularRouteIsSwap(t *testing.T) {
	for _, c := range []struct{ sig, idx, mint string }{
		{jupiterTrailingByteRoutes[0].sig, "2", constants.TOKENS.USDC},
		{"2ZeNbKRUUQPfiZzNu1kgfAxMyXZYgvhhyS6n2nwXaj5LhrtJEqu7Ei7i9RjNudZQev6qvnp6cVFv69TvQQSFw84k", "8", constants.TOKENS.SOL},
	} {
		tx := loadFixture(t, c.sig)
		ix := findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE)
		// Independent: the source token account (by its token balance or a
		// TransferChecked out of it) and the destination mint (account 5)
		// are both c.mint
		srcMint := tokenBalanceMint(tx, ix.accounts[2])
		for _, in := range fixtureIxs(t, tx) {
			if srcMint == "" && in.outer == ix.outer && in.inner >= 0 && in.programId == constants.TOKEN_PROGRAM_ID &&
				len(in.data) > 0 && in.data[0] == 12 && len(in.accounts) > 1 && in.accounts[0] == ix.accounts[2] {
				srcMint = in.accounts[1]
			}
		}
		if srcMint != c.mint || ix.accounts[5] != c.mint {
			t.Fatalf("%s: route is not %s -> %s", c.sig[:8], c.mint, c.mint)
		}
		_, trade := jupTrade(t, parseShred(t, tx, nil), c.idx)
		if trade.InputToken.Mint != c.mint || trade.OutputToken.Mint != c.mint || trade.Type != types.TradeTypeSwap {
			t.Errorf("%s: trade %s %s -> %s, want SWAP %s -> %s", c.sig[:8], trade.Type, trade.InputToken.Mint, trade.OutputToken.Mint, c.mint, c.mint)
		}
	}
}

// TestShredJupiterTokenLedger: token-ledger routes end with 11 bytes; the
// decoder read the last 10. The input amount is only known at execution.
// shred-1.
func TestShredJupiterTokenLedger(t *testing.T) {
	tx := loadFixture(t, sigJupTokenLedger)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE_WITH_TOKEN_LEDGER)
	tail := ix.data[len(ix.data)-11:]
	quote, slippage := le64At(tail, 0), le16At(tail, 8)

	res := parseShred(t, tx, nil)
	ins, trade := jupTrade(t, res, utils.FormatIdx(ix.outer, -1))
	if trade.OutputToken.AmountRaw != u64str(quote) || *trade.SlippageBps != int(slippage) || trade.InputToken.AmountRaw != "0" {
		t.Errorf("token ledger = in %s out %s slippage %d, want 0 %d %d", trade.InputToken.AmountRaw, trade.OutputToken.AmountRaw, *trade.SlippageBps, quote, slippage)
	}
	if ins.InputAmountKind != types.ShredAmountUnknown {
		t.Errorf("input amount kind = %s, want unknown", ins.InputAmountKind)
	}
	if trade.InputToken.Mint != usdcMint || trade.OutputToken.Mint != solMint {
		t.Errorf("mints = %s -> %s, want USDC -> SOL", trade.InputToken.Mint, trade.OutputToken.Mint)
	}
	if got := ownerTokenDelta(tx, ix.accounts[1], solMint); got.Cmp(minOut(quote, slippage)) < 0 {
		t.Errorf("received %s below the minimum %s", got, minOut(quote, slippage))
	}
}

// TestShredJupiterRouteV2: the *_v2 routes (amounts first, route plan last)
// were not decoded. shred-9.
func TestShredJupiterRouteV2(t *testing.T) {
	cases := []struct {
		sig    string
		disc   []byte
		shared bool
		// account indexes of user, source mint, destination mint
		user, src, dst int
	}{
		{sigJupRouteV2, constants.DISCRIMINATORS.JUPITER.ROUTE_V2, false, 0, 3, 4},
		{sigJupSharedRouteV2, constants.DISCRIMINATORS.JUPITER.SHARED_ACCOUNTS_ROUTE_V2, true, 1, 6, 7},
	}
	for _, c := range cases {
		tx := loadFixture(t, c.sig)
		ix := findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, c.disc)
		head := ix.data[8:]
		if c.shared {
			head = head[1:]
		}
		in, quote, slippage, platformFee := le64At(head, 0), le64At(head, 8), le16At(head, 16), le16At(head, 18)

		res := parseShred(t, tx, nil)
		ins, trade := jupTrade(t, res, utils.FormatIdx(ix.outer, -1))
		if trade.InputToken.AmountRaw != u64str(in) || trade.OutputToken.AmountRaw != u64str(quote) || *trade.SlippageBps != int(slippage) {
			t.Errorf("%s: amounts %s %s %d, want %d %d %d", ins.Action, trade.InputToken.AmountRaw, trade.OutputToken.AmountRaw, *trade.SlippageBps, in, quote, slippage)
		}
		if trade.User != ix.accounts[c.user] || trade.InputToken.Mint != ix.accounts[c.src] || trade.OutputToken.Mint != ix.accounts[c.dst] {
			t.Errorf("%s: %s -> %s by %s", ins.Action, trade.InputToken.Mint, trade.OutputToken.Mint, trade.User)
		}
		if got := ownerTokenDelta(tx, trade.User, trade.OutputToken.Mint); got.Cmp(minOut(quote, slippage)) < 0 {
			t.Errorf("%s: received %s below the minimum %s", ins.Action, got, minOut(quote, slippage))
		}
		raw := res.Instructions[constants.DEX_PROGRAMS.JUPITER.Name]
		if len(raw) != 1 {
			t.Fatalf("%s: raw events %d", ins.Action, len(raw))
		}
		data := raw[0].(*jupiter.JupiterShredInstruction).Data.(*jupiter.JupiterRouteData)
		if data.PlatformFeeBps != platformFee || data.ExactOut {
			t.Errorf("%s: platform fee %d exactOut %v, want %d false", ins.Action, data.PlatformFeeBps, data.ExactOut, platformFee)
		}
	}

	// route_v2 input: the user paid in_amount lamports plus the fee
	tx := loadFixture(t, sigJupRouteV2)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE_V2)
	if paid := new(big.Int).Neg(lamportDelta(tx, ix.accounts[0])); paid.Cmp(new(big.Int).SetUint64(le64At(ix.data, 8)+tx.Meta.Fee)) != 0 {
		t.Errorf("user paid %s, want in_amount + fee", paid)
	}
}

// TestShredJupiterPreExecution: without meta, lookup-table accounts are
// unknown. Mints must not be guessed (route used to default to SOL), the
// instruction is flagged, and ParseConfig.AddressLookupTables resolves them.
// shred-11, shred-18.
func TestShredJupiterPreExecution(t *testing.T) {
	tx := loadFixture(t, sigJupRouteV1)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE)

	res := parseShred(t, preExec(t, tx), nil)
	if !res.HasUnresolvedAccounts || res.TxStatus != types.TransactionStatusUnknown {
		t.Errorf("HasUnresolvedAccounts=%v TxStatus=%s, want true unknown", res.HasUnresolvedAccounts, res.TxStatus)
	}
	ins, trade := jupTrade(t, res, "4")
	if !ins.UnresolvedAccounts {
		t.Error("instruction with lookup-table accounts not flagged")
	}
	if trade.OutputToken.Mint != "" || trade.InputToken.Mint != "" || trade.Type != types.TradeTypeSwap {
		t.Errorf("pre-exec trade %s %q -> %q, want SWAP with unknown mints", trade.Type, trade.InputToken.Mint, trade.OutputToken.Mint)
	}
	if trade.InputToken.Decimals != 0 || trade.OutputToken.Decimals != 0 {
		t.Errorf("unknown mints must report decimals 0, got %d/%d", trade.InputToken.Decimals, trade.OutputToken.Decimals)
	}

	res = parseShred(t, preExec(t, tx), &types.ParseConfig{AddressLookupTables: lookupTables(t, tx)})
	ins, trade = jupTrade(t, res, "4")
	if res.HasUnresolvedAccounts || ins.UnresolvedAccounts {
		t.Error("accounts still unresolved with the lookup tables")
	}
	if trade.OutputToken.Mint != ix.accounts[5] {
		t.Errorf("output mint with lookup tables = %q, want %s", trade.OutputToken.Mint, ix.accounts[5])
	}
}

const (
	// failed tx with an exact_out_route CPI at 1-0 (arguments still valid)
	sigJupExactOut = "2KBxSbgdxnU18EET7BvKMfcpJRhxjuLqnkxaBuHGBT7YKB38QViVJPNSDikSCoLt1GECBFbY9Uppriec8jVfGbC"
	// failed tx with a shared_accounts_route_with_token_ledger at outer 9
	sigJupSharedLedger = "26HdF31VdqpmpPCGmB6iwRGKf8rbN7JHQr5DRNDn2AKQhWhAg42w38Hqf1g4D9gtqf3vNQe4cRrd9fGG1goEEfiG"
)

// TestShredJupiterExactOut: exact-out routes reported out_amount as the
// input and quoted_in_amount as the output. The fixture is a failed
// transaction; its instruction arguments are what a pre-execution consumer
// sees. shred-10.
func TestShredJupiterExactOut(t *testing.T) {
	tx := loadFixture(t, sigJupExactOut)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE_EXACT_OUT)
	tail := ix.data[len(ix.data)-19:]
	out, quotedIn, slippage := le64At(tail, 0), le64At(tail, 8), le16At(tail, 16)
	if ix.accounts[6] != usdcMint || out != 1000000 {
		t.Fatalf("fixture: destination mint %s out %d", ix.accounts[6], out)
	}

	res := parseShred(t, tx, &types.ParseConfig{IncludeFailedTxs: true})
	ins, trade := jupTrade(t, res, utils.FormatIdx(ix.outer, ix.inner))
	if trade.InputToken.Mint != ix.accounts[5] || trade.InputToken.AmountRaw != u64str(quotedIn) ||
		trade.OutputToken.Mint != ix.accounts[6] || trade.OutputToken.AmountRaw != u64str(out) || *trade.SlippageBps != int(slippage) {
		t.Errorf("exact_out_route = %s %s -> %s %s, want %s %d -> %s %d", trade.InputToken.Mint, trade.InputToken.AmountRaw, trade.OutputToken.Mint, trade.OutputToken.AmountRaw, ix.accounts[5], quotedIn, ix.accounts[6], out)
	}
	if ins.InputAmountKind != types.ShredAmountQuote || ins.OutputAmountKind != types.ShredAmountExact {
		t.Errorf("kinds %s/%s, want quote/exact", ins.InputAmountKind, ins.OutputAmountKind)
	}

	// No exact_out_route_v2 was found in the fixtures or sampled blocks; its
	// accounts are those of route_v2 (JUP6 IDL), so the test re-tags the real
	// route_v2 of 5qJs7ws4: (out_amount, quoted_in_amount) at the head
	full := loadFixture(t, sigJupRouteV2)
	accounts := findIx(t, full, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE_V2).accounts
	tx = preExec(t, full)
	ix = findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.ROUTE_V2)
	setIxData(ix, append(append([]byte{}, constants.DISCRIMINATORS.JUPITER.EXACT_OUT_ROUTE_V2...), ix.data[8:]...))
	ins, trade = jupTrade(t, parseShred(t, tx, &types.ParseConfig{AddressLookupTables: lookupTables(t, full)}), utils.FormatIdx(ix.outer, -1))
	if ins.Action != "exact_out_route_v2" || trade.OutputToken.AmountRaw != u64str(le64At(ix.data, 8)) || trade.InputToken.AmountRaw != u64str(le64At(ix.data, 16)) ||
		trade.InputToken.Mint != accounts[3] || trade.OutputToken.Mint != accounts[4] {
		t.Errorf("exact_out_route_v2 = %s %s %s -> %s %s", ins.Action, trade.InputToken.Mint, trade.InputToken.AmountRaw, trade.OutputToken.Mint, trade.OutputToken.AmountRaw)
	}
}

// TestShredJupiterSharedTokenLedger decodes the 11-byte tail of
// shared_accounts_route_with_token_ledger (failed tx; arguments valid).
// shred-1.
func TestShredJupiterSharedTokenLedger(t *testing.T) {
	tx := loadFixture(t, sigJupSharedLedger)
	ix := findIx(t, tx, constants.DEX_PROGRAMS.JUPITER.ID, constants.DISCRIMINATORS.JUPITER.SHARE_ACCOUNTS_ROUTE_WITH_TOKEN_LEDGER)
	tail := ix.data[len(ix.data)-11:]
	res := parseShred(t, tx, &types.ParseConfig{IncludeFailedTxs: true})
	ins, trade := jupTrade(t, res, utils.FormatIdx(ix.outer, -1))
	if trade.OutputToken.AmountRaw != u64str(le64At(tail, 0)) || *trade.SlippageBps != int(le16At(tail, 8)) || trade.InputToken.AmountRaw != "0" ||
		trade.InputToken.Mint != ix.accounts[7] || trade.OutputToken.Mint != ix.accounts[8] || trade.User != ix.accounts[2] || ins.InputAmountKind != types.ShredAmountUnknown {
		t.Errorf("shared token ledger = %+v kinds %s", trade, ins.InputAmountKind)
	}
	// Failed transactions are skipped by default
	if res := parseShred(t, tx, nil); len(res.ParsedInstructions) != 0 || res.TxStatus != types.TransactionStatusFailed {
		t.Errorf("failed tx decoded by default: %d instructions, status %s", len(res.ParsedInstructions), res.TxStatus)
	}
}
