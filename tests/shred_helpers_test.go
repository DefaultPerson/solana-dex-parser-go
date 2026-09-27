package tests

import (
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Helpers for the shred WP tests. Expected values come from the raw fixture
// (instruction bytes decoded here with the on-chain IDL layouts, balances),
// independently of the decoders under test.

// parseShred runs ShredParser and fails on a nil or failed result
func parseShred(t testing.TB, tx *adapter.SolanaTransaction, config *types.ParseConfig) *types.ParseShredResult {
	t.Helper()
	res := dexparser.NewShredParser().ParseAll(tx, config)
	if res == nil {
		t.Fatal("ShredParser.ParseAll returned nil")
	}
	if !res.State {
		t.Fatalf("ShredParser.ParseAll: State=false Msg=%q", res.Msg)
	}
	return res
}

// preExec returns a copy of tx without meta, as a pre-execution source
// (ShredStream) delivers it
func preExec(t testing.TB, tx *adapter.SolanaTransaction) *adapter.SolanaTransaction {
	t.Helper()
	c := cloneTx(t, tx)
	c.Meta = nil
	return c
}

// rawIx is an instruction of a "json" fixture with its resolved accounts
type rawIx struct {
	outer, inner int
	m            map[string]interface{}
	programId    string
	accounts     []string
	data         []byte
}

// fixtureIxs lists the outer and (with meta) inner instructions of a "json"
// fixture in classifier order
func fixtureIxs(t testing.TB, tx *adapter.SolanaTransaction) []rawIx {
	t.Helper()
	keys := rawAccountKeys(tx)
	key := func(v interface{}) string {
		i := jsonInt(v)
		if i < 0 || i >= len(keys) {
			return ""
		}
		return keys[i]
	}
	build := func(outer, inner int, v interface{}) rawIx {
		m, ok := v.(map[string]interface{})
		if !ok {
			t.Fatalf("instruction %d/%d is not a map", outer, inner)
		}
		ix := rawIx{outer: outer, inner: inner, m: m, programId: key(m["programIdIndex"])}
		for _, a := range m["accounts"].([]interface{}) {
			ix.accounts = append(ix.accounts, key(a))
		}
		ix.data, _ = base58.Decode(m["data"].(string))
		return ix
	}
	var out []rawIx
	for i, v := range tx.Transaction.Message.Instructions {
		out = append(out, build(i, -1, v))
	}
	if tx.Meta != nil {
		for _, set := range tx.Meta.InnerInstructions {
			for j, v := range set.Instructions {
				out = append(out, build(set.Index, j, v))
			}
		}
	}
	return out
}

// findIx returns the first instruction of programId whose data starts with prefix
func findIx(t testing.TB, tx *adapter.SolanaTransaction, programId string, prefix []byte) rawIx {
	t.Helper()
	for _, ix := range fixtureIxs(t, tx) {
		if ix.programId == programId && len(ix.data) >= len(prefix) && string(ix.data[:len(prefix)]) == string(prefix) {
			return ix
		}
	}
	t.Fatalf("no %s instruction with prefix %x", programId, prefix)
	return rawIx{}
}

// setIxData replaces the data of a fixture instruction (in place)
func setIxData(ix rawIx, data []byte) {
	ix.m["data"] = base58.Encode(data)
}

func le64At(b []byte, off int) uint64 { return binary.LittleEndian.Uint64(b[off:]) }
func le16At(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off:]) }
func le32At(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }

func base58Encode(b []byte) string { return base58.Encode(b) }

// adapterTx is the fixture transaction type
type adapterTx = adapter.SolanaTransaction

// typedAt returns the typed instructions of a program at idx
func typedAt(res *types.ParseShredResult, programId, idx string) []types.ParsedShredInstruction {
	var out []types.ParsedShredInstruction
	for _, p := range res.ParsedInstructions {
		if p.ProgramID == programId && p.Idx == idx {
			out = append(out, p)
		}
	}
	return out
}

// oneTypedAt returns the single typed instruction of a program at idx
func oneTypedAt(t testing.TB, res *types.ParseShredResult, programId, idx string) types.ParsedShredInstruction {
	t.Helper()
	got := typedAt(res, programId, idx)
	if len(got) != 1 {
		t.Fatalf("typed instructions of %s at %s: %d, want 1 (all: %+v)", programId, idx, len(got), res.ParsedInstructions)
	}
	return got[0]
}

func u64str(v uint64) string { return strconv.FormatUint(v, 10) }

// tokenBalance returns the (post, else pre) token balance entry of account
// from the raw meta of a "json" fixture
func tokenBalance(tx *adapter.SolanaTransaction, account string) *adapter.TokenBalance {
	keys := rawAccountKeys(tx)
	for _, list := range [][]adapter.TokenBalance{tx.Meta.PostTokenBalances, tx.Meta.PreTokenBalances} {
		for i := range list {
			if list[i].AccountIndex < len(keys) && keys[list[i].AccountIndex] == account {
				return &list[i]
			}
		}
	}
	return nil
}

// tokenBalanceMint returns the mint of a token account from the raw token balances
func tokenBalanceMint(tx *adapter.SolanaTransaction, account string) string {
	if b := tokenBalance(tx, account); b != nil {
		return b.Mint
	}
	return ""
}

// tokenBalanceDecimals returns the decimals of a token account from the raw token balances
func tokenBalanceDecimals(tx *adapter.SolanaTransaction, account string) uint8 {
	if b := tokenBalance(tx, account); b != nil {
		return b.UiTokenAmount.Decimals
	}
	return 0
}

// mintDecimals returns the decimals of mint from any raw token balance of the
// fixture, else the well-known value (WSOL, stablecoins), else 0
func mintDecimals(tx *adapter.SolanaTransaction, mint string) uint8 {
	if tx.Meta != nil {
		for _, list := range [][]adapter.TokenBalance{tx.Meta.PostTokenBalances, tx.Meta.PreTokenBalances} {
			for _, b := range list {
				if b.Mint == mint {
					return b.UiTokenAmount.Decimals
				}
			}
		}
	}
	return constants.TOKEN_DECIMALS[mint]
}
