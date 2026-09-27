package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
)

// Helpers for the jupiter WP tests. They decode the raw "json" fixture
// directly (instructions, Anchor events), independently of the adapter, the
// classifier and the constants, so expected values do not come from the code
// under test.

const (
	jupV6ID     = "JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4"
	jupDCAID    = "DCA265Vj8a9CEuX1eb1LWRnDT7uK6q1xMipnNyatn23M"
	jupLimit2ID = "j1o2qRpjcyUwEvwtcfhEQefh773ZgjxcVRry7LDqg5X"
	jupVAID     = "VALaaymxQh2mNy2trH9jUqHT1mTow76wpTcGmSWSwJe"
	jupZID      = "61DFfeTKM7trxYcPQCM78bJ794ddZprZpAwAnLiwTpYH"
)

// jupRawIx is one instruction of a raw fixture; Inner is -1 for outer ones.
type jupRawIx struct {
	Outer    int
	Inner    int
	Program  string
	Accounts []string
	Data     []byte
}

// jupRawInstructions decodes all outer and inner instructions of a "json"
// fixture in execution order.
func jupRawInstructions(t *testing.T, sig string) []jupRawIx {
	t.Helper()
	raw, err := readFixture(sig, "json")
	if err != nil {
		t.Fatalf("%.8s: %v", sig, err)
	}
	type ix struct {
		ProgramIdIndex int    `json:"programIdIndex"`
		Accounts       []int  `json:"accounts"`
		Data           string `json:"data"`
	}
	var doc struct {
		Transaction struct {
			Message struct {
				AccountKeys  []string `json:"accountKeys"`
				Instructions []ix     `json:"instructions"`
			} `json:"message"`
		} `json:"transaction"`
		Meta struct {
			InnerInstructions []struct {
				Index        int  `json:"index"`
				Instructions []ix `json:"instructions"`
			} `json:"innerInstructions"`
			LoadedAddresses *struct {
				Writable []string `json:"writable"`
				Readonly []string `json:"readonly"`
			} `json:"loadedAddresses"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("%.8s: %v", sig, err)
	}
	keys := append([]string{}, doc.Transaction.Message.AccountKeys...)
	if la := doc.Meta.LoadedAddresses; la != nil {
		keys = append(append(keys, la.Writable...), la.Readonly...)
	}
	conv := func(outer, inner int, x ix) jupRawIx {
		var data []byte
		var err error
		if x.Data != "" {
			data, err = base58.Decode(x.Data)
		}
		if err != nil {
			t.Fatalf("%.8s: data of %d-%d: %v", sig, outer, inner, err)
		}
		r := jupRawIx{Outer: outer, Inner: inner, Program: keys[x.ProgramIdIndex], Data: data}
		for _, a := range x.Accounts {
			r.Accounts = append(r.Accounts, keys[a])
		}
		return r
	}
	var out []jupRawIx
	for i, x := range doc.Transaction.Message.Instructions {
		out = append(out, conv(i, -1, x))
		for _, set := range doc.Meta.InnerInstructions {
			if set.Index == i {
				for j, y := range set.Instructions {
					out = append(out, conv(i, j, y))
				}
			}
		}
	}
	return out
}

// jupDisc returns the first 8 bytes of sha256(preimage), the Anchor
// discriminator of "global:<ix>" or "event:<Event>".
func jupDisc(preimage string) []byte {
	h := sha256.Sum256([]byte(preimage))
	return h[:8]
}

// jupEvents returns the payloads (after the 16-byte self-CPI prefix) of the
// Anchor events named event emitted by program, with their instructions.
func jupEvents(t *testing.T, sig, program, event string) ([]jupRawIx, [][]byte) {
	t.Helper()
	// EVENT_IX_TAG: sha256("anchor:event")[:8] read as a big-endian u64 and
	// written little-endian, i.e. the 8 bytes reversed (e445a52e51cb9a1d)
	tag := jupDisc("anchor:event")
	prefix := make([]byte, 8)
	for i := range tag {
		prefix[i] = tag[7-i]
	}
	disc := jupDisc("event:" + event)
	var ixs []jupRawIx
	var payloads [][]byte
	for _, x := range jupRawInstructions(t, sig) {
		if x.Program != program || len(x.Data) < 16 || string(x.Data[:8]) != string(prefix) || string(x.Data[8:16]) != string(disc) {
			continue
		}
		ixs = append(ixs, x)
		payloads = append(payloads, x.Data[16:])
	}
	return ixs, payloads
}

// jupHop is one swap leg of a Jupiter v6 SwapEvent or SwapsEvent.
type jupHop struct {
	AMM, InMint, OutMint string
	In, Out              *big.Int
}

func jupU64(b []byte) *big.Int { return new(big.Int).SetUint64(binary.LittleEndian.Uint64(b)) }

// jupDecodeSwapEvent decodes SwapEvent {amm, input_mint, input_amount,
// output_mint, output_amount} (JUP6 IDL).
func jupDecodeSwapEvent(p []byte) jupHop {
	return jupHop{AMM: base58.Encode(p[0:32]), InMint: base58.Encode(p[32:64]), In: jupU64(p[64:72]), OutMint: base58.Encode(p[72:104]), Out: jupU64(p[104:112])}
}

// jupDecodeSwapsEvent decodes SwapsEvent {swap_events: Vec<SwapEventV2
// {input_mint, input_amount, output_mint, output_amount, amm}>} (JUP6 IDL).
func jupDecodeSwapsEvent(p []byte) []jupHop {
	n := int(binary.LittleEndian.Uint32(p[0:4]))
	var hops []jupHop
	for i := 0; i < n; i++ {
		b := p[4+i*112 : 4+(i+1)*112]
		hops = append(hops, jupHop{InMint: base58.Encode(b[0:32]), In: jupU64(b[32:40]), OutMint: base58.Encode(b[40:72]), Out: jupU64(b[72:80]), AMM: base58.Encode(b[80:112])})
	}
	return hops
}

// jupFindSig returns the full signature of a fixture from its prefix.
func jupFindSig(t *testing.T, prefix string) string {
	t.Helper()
	for _, s := range fixtureSignatures(t, "json") {
		if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
			return s
		}
	}
	t.Fatalf("no fixture %s", prefix)
	return ""
}

// jupInstruction returns the first instruction of program whose data starts
// with the Anchor discriminator of global:<name>.
func jupInstruction(t *testing.T, sig, program, name string) jupRawIx {
	t.Helper()
	for _, x := range jupRawInstructions(t, sig) {
		if x.Program == program && len(x.Data) >= 8 && bytes.Equal(x.Data[:8], jupDisc("global:"+name)) {
			return x
		}
	}
	t.Fatalf("%.8s: no %s instruction", sig, name)
	return jupRawIx{}
}

// jupRouteAuthorityPos is the position of user_transfer_authority, the owner
// of the swapped tokens, in the accounts of each Jupiter v6 route instruction
// (on-chain JUP6 IDL).
var jupRouteAuthorityPos = map[string]int{
	"route": 1, "route_with_token_ledger": 1, "exact_out_route": 1,
	"shared_accounts_route": 2, "shared_accounts_route_with_token_ledger": 2, "shared_accounts_exact_out_route": 2,
	"route_v2": 0, "exact_out_route_v2": 0,
	"shared_accounts_route_v2": 1, "shared_accounts_exact_out_route_v2": 1,
}

// jupRouteAuthorities returns the user_transfer_authority of every Jupiter v6
// route instruction of sig, in execution order.
func jupRouteAuthorities(t *testing.T, sig string) []string {
	t.Helper()
	var out []string
	for _, x := range jupRawInstructions(t, sig) {
		if x.Program != jupV6ID || len(x.Data) < 8 {
			continue
		}
		for name, pos := range jupRouteAuthorityPos {
			if bytes.Equal(x.Data[:8], jupDisc("global:"+name)) && pos < len(x.Accounts) {
				out = append(out, x.Accounts[pos])
			}
		}
	}
	return out
}

// jupUserValueDelta returns the balance change of owner in mint: its token
// accounts, plus for SOL its lamports (the transaction fee added back when
// owner pays it).
func jupUserValueDelta(tx *adapter.SolanaTransaction, owner, mint string) *big.Int {
	d := ownerTokenDelta(tx, owner, mint)
	if mint == solMint {
		d.Add(d, lamportDelta(tx, owner))
		if rawAccountKeys(tx)[0] == owner {
			d.Add(d, new(big.Int).SetUint64(tx.Meta.Fee))
		}
	}
	return d
}
