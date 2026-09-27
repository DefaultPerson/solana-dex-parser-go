package tests

import (
	"encoding/binary"
	"math/big"
	"strconv"
	"testing"

	"github.com/goccy/go-json"
	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Helpers for the meme regression tests. memeRawTx re-reads a fixture with
// plain structs and decodes token and SOL transfers itself, so expected values
// do not depend on the adapter or the transfer parser under test.

const (
	memeSystemProgram = "11111111111111111111111111111111"
	memeTokenProgram  = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	memeToken2022     = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"
	usd1Mint          = "USD1ttGY1N17NEEHLmELoaybftRBUSErhqYiQzvEmuB"
)

type memeRawIx struct {
	ProgramIdIndex int    `json:"programIdIndex"`
	Accounts       []int  `json:"accounts"`
	Data           string `json:"data"`
}

type memeRawBalance struct {
	AccountIndex  int    `json:"accountIndex"`
	Mint          string `json:"mint"`
	Owner         string `json:"owner"`
	UiTokenAmount struct {
		Amount string `json:"amount"`
	} `json:"uiTokenAmount"`
}

type memeRawTx struct {
	keys  []string
	outer []memeRawIx
	inner map[int][]memeRawIx
	owner map[string]string // token account -> owner
	mint  map[string]string // token account -> mint
}

// memeRawTransfer is a SOL or token transfer decoded from raw instruction data
type memeRawTransfer struct {
	Inner     int // inner index, -1 for an outer instruction
	Program   string
	From, To  string
	Authority string
	Mint      string // SOL mint for System transfers
	Amount    *big.Int
}

func loadMemeRaw(t testing.TB, sig string) *memeRawTx {
	t.Helper()
	loadFixture(t, sig) // fetch when missing and allowed
	raw, err := readFixture(sig, "json")
	if err != nil {
		t.Fatalf("read %s: %v", sig, err)
	}
	var f struct {
		Transaction struct {
			Message struct {
				AccountKeys  []string    `json:"accountKeys"`
				Instructions []memeRawIx `json:"instructions"`
			} `json:"message"`
		} `json:"transaction"`
		Meta struct {
			InnerInstructions []struct {
				Index        int         `json:"index"`
				Instructions []memeRawIx `json:"instructions"`
			} `json:"innerInstructions"`
			LoadedAddresses *struct {
				Writable []string `json:"writable"`
				Readonly []string `json:"readonly"`
			} `json:"loadedAddresses"`
			PreTokenBalances  []memeRawBalance `json:"preTokenBalances"`
			PostTokenBalances []memeRawBalance `json:"postTokenBalances"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("unmarshal %s: %v", sig, err)
	}
	r := &memeRawTx{
		keys:  f.Transaction.Message.AccountKeys,
		outer: f.Transaction.Message.Instructions,
		inner: map[int][]memeRawIx{},
		owner: map[string]string{},
		mint:  map[string]string{},
	}
	if la := f.Meta.LoadedAddresses; la != nil {
		r.keys = append(r.keys, la.Writable...)
		r.keys = append(r.keys, la.Readonly...)
	}
	for _, s := range f.Meta.InnerInstructions {
		r.inner[s.Index] = s.Instructions
	}
	for _, b := range append(f.Meta.PreTokenBalances, f.Meta.PostTokenBalances...) {
		r.owner[r.keys[b.AccountIndex]] = b.Owner
		r.mint[r.keys[b.AccountIndex]] = b.Mint
	}
	return r
}

func (r *memeRawTx) ix(outer, inner int) memeRawIx {
	if inner < 0 {
		return r.outer[outer]
	}
	return r.inner[outer][inner]
}

// accounts returns the account keys of an instruction
func (r *memeRawTx) accounts(outer, inner int) []string {
	var out []string
	for _, a := range r.ix(outer, inner).Accounts {
		out = append(out, r.keys[a])
	}
	return out
}

func (r *memeRawTx) program(outer, inner int) string {
	return r.keys[r.ix(outer, inner).ProgramIdIndex]
}

func (r *memeRawTx) data(outer, inner int) []byte {
	d, _ := base58.Decode(r.ix(outer, inner).Data)
	return d
}

// transfers decodes System transfers and SPL Token / Token-2022 transfer and
// transferChecked instructions in the outer instruction and its inner ones
func (r *memeRawTx) transfers(outer int) []memeRawTransfer {
	var out []memeRawTransfer
	for inner := -1; inner < len(r.inner[outer]); inner++ {
		prog := r.program(outer, inner)
		data := r.data(outer, inner)
		acc := r.accounts(outer, inner)
		switch {
		case prog == memeSystemProgram && len(data) >= 12 && binary.LittleEndian.Uint32(data) == 2 && len(acc) >= 2:
			out = append(out, memeRawTransfer{inner, prog, acc[0], acc[1], acc[0], solMint,
				new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[4:]))})
		case (prog == memeTokenProgram || prog == memeToken2022) && len(data) >= 9 && data[0] == 3 && len(acc) >= 3:
			out = append(out, memeRawTransfer{inner, prog, acc[0], acc[1], acc[2], r.mint[acc[1]],
				new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[1:]))})
		case (prog == memeTokenProgram || prog == memeToken2022) && len(data) >= 9 && data[0] == 12 && len(acc) >= 4:
			out = append(out, memeRawTransfer{inner, prog, acc[0], acc[2], acc[3], acc[1],
				new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[1:]))})
		}
	}
	return out
}

// sumTransfers sums the transfers of outer with inner index in [from, to)
// that match keep
func (r *memeRawTx) sumTransfers(outer, from, to int, keep func(memeRawTransfer) bool) *big.Int {
	sum := new(big.Int)
	for _, tr := range r.transfers(outer) {
		if tr.Inner >= from && tr.Inner < to && keep(tr) {
			sum.Add(sum, tr.Amount)
		}
	}
	return sum
}

// memeParse parses a fixture with the default config (ParseAll)
func memeParse(t testing.TB, sig string) *types.ParseResult {
	t.Helper()
	r := dexparser.NewDexParser().ParseAll(loadFixture(t, sig), nil)
	if r == nil || !r.State {
		t.Fatalf("%.8s: parse failed: %+v", sig, r)
	}
	return r
}

func memeTradeAt(t testing.TB, r *types.ParseResult, idx string) types.TradeInfo {
	t.Helper()
	for _, tr := range r.Trades {
		if tr.Idx == idx {
			return tr
		}
	}
	t.Fatalf("%.8s: no trade at %s (trades: %v)", r.Signature, idx, memeTradeIdxs(r.Trades))
	return types.TradeInfo{}
}

func memeTradeIdxs(trades []types.TradeInfo) []string {
	var out []string
	for _, tr := range trades {
		out = append(out, tr.Idx+" "+tr.AMM+" "+string(tr.Type))
	}
	return out
}

func memeEventAt(t testing.TB, r *types.ParseResult, idx string) types.MemeEvent {
	t.Helper()
	for _, e := range r.MemeEvents {
		if e.Idx == idx {
			return e
		}
	}
	var got []string
	for _, e := range r.MemeEvents {
		got = append(got, e.Idx+" "+e.Protocol+" "+string(e.Type))
	}
	t.Fatalf("%.8s: no meme event at %s (events: %v)", r.Signature, idx, got)
	return types.MemeEvent{}
}

// feeOf returns the raw amount of the fee component of type typ ("" if none)
func feeOf(fees []types.FeeInfo, typ string) string {
	for _, f := range fees {
		if f.Type == typ {
			return f.AmountRaw
		}
	}
	return ""
}

func u64s(v uint64) string { return strconv.FormatUint(v, 10) }

// checkToken compares a token side with the expected mint, raw amount and decimals
func checkToken(t testing.TB, what string, got *types.TokenInfo, mint, amountRaw string, decimals uint8) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: token is nil", what)
		return
	}
	if got.Mint != mint || got.AmountRaw != amountRaw || got.Decimals != decimals {
		t.Errorf("%s: got %s %s dec %d, want %s %s dec %d", what, got.Mint, got.AmountRaw, got.Decimals, mint, amountRaw, decimals)
	}
}
