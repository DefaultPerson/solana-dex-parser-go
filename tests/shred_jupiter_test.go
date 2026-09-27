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
// bytes with the on-chain JUP6 IDL layouts: v1 routes end with (amount u64,
// amount u64, slippage_bps u16, platform_fee_bps u8), token-ledger routes
// with (quoted_out u64, slippage_bps u16, platform_fee_bps u8), v2 routes
// start with (amount u64, amount u64, slippage_bps u16, platform_fee_bps u16,
// positive_slippage_bps u16). shred-1, shred-9, shred-10, shred-11, shred-22,
// shred-18, shred-30.

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
