package tests

import (
	"math"
	"strings"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/dflow"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// DFlow swap_orchestrator shred decoding. The expected values were decoded
// with an independent IDL-driven Borsh decoder (Python, on-chain IDL account
// Cp2dCjxC..., /tmp/sdp-audit/work/research/shred-scratch/dflow_an.out):
// action list, SwapParams / Swap2Params and the first swap action's amount,
// and the SwapEvent legs. shred-24.

type dflowCase struct {
	sig                    string
	name                   string
	actions                string
	firstAmount            uint64
	quoted                 uint64
	slippage, platformFee  uint16
	positiveSlippage       uint8
	inMint, outMint        string
	inAmountUnknown        bool
	destination, dstNative bool
}

var dflowCases = []dflowCase{
	{sig: "2NcCFY7NTiH2WJgT7exX2gHVsQUqk6TVZMDLoueQHr28B3VNUD7C32bLnYbXB6uYegvsH8ZEP9xST1Zc7135wGP8", name: "swap",
		actions: "BisonFiSwap", firstAmount: 243820729, inMint: usdcMint, outMint: solMint},
	// sponsored: user_token_authority is not the fee payer
	{sig: "2BL4GjmzqMy262NMWA8JdMzhoxnmPwabfQjvQBAnZHAcTsVTdZiYfHqvi4SdfjRKN3M3oH71ZVu5iHGrhFubaPfK", name: "swap",
		actions:     "RecordId,SetSponsor,SetSponsorIntermediateCloseAuthority,InitAtaIdempotent,SetMinimumLegPrices,DeriverseSwap,RaydiumClmmSwap,PumpFunBuy",
		firstAmount: 3556827, quoted: 189562578787, slippage: 4500, inMint: usdcMint, outMint: "D7p9uqrR7qo4GE4Rh92Ueu8C2tN3v6bXRnDK9cfJUmE2"},
	// one trailing byte after SwapParams
	{sig: "4i7siGdtHddKrJxQv7GQCJLayCJees49jiakaKNBUdpwEg7pWqHcLaJDk5k1TCreR2NLucK8PhaGUNXcZBZusWVC", name: "swap",
		actions: "BisonFiSwap", firstAmount: 363000000, quoted: 1, inMint: usdcMint, outMint: solMint},
	{sig: "2uiVt28VHZe3NRkzxzmRHcrRmSWW6CBi4HkmNbURtZBt2bVGDWEV8APt6gLLQgvaHMzp82U4iTik3dCAkH4J54CN", name: "swap2",
		actions: "BisonFiSwap", firstAmount: 2001495196, quoted: 248147365, slippage: 100, inMint: solMint, outMint: usdcMint},
	// first action amount u64::MAX: the input is only known after execution
	{sig: "2WtdgsixFYLFLD8Ner7Ke4k4GKevTghPnhhPv6ydT6GVgzYmbtqKkbCyuhiKsZUyFHrEQMnJKwsM4MvnCbyD9vC", name: "swap2",
		actions: "BisonFiSwap", quoted: 172178675, inAmountUnknown: true, inMint: solMint, outMint: usdcMint},
	// destination authority and mint passed as the program id (omitted)
	{sig: "W4Umv2pT7notEpq3JU3zhP3wdHAXzoqx4Edy4oEpWMZ2AQJGUxbUAVrPgScyqxHi53LNpfaSzyKTzP48mQauyEX", name: "swap_with_destination",
		actions: "RecordId,SetMinimumLegPrices,BisonFiSwap,PumpFunAmmBuy", firstAmount: 60921361, quoted: 400420726289, slippage: 146,
		inMint: usdcMint, outMint: "B6f27ETGcjgGNB1fqULJbXVmw9FnL8HgBp7R83hmpump", destination: true},
	{sig: "5awNRRvdTmtvQfkzSWeBi4FNd47p9CgvmFZNQ4M56j9myAcw9io3pZJYkoF5V3gG2gyFZwbBwzmdUVoBHy3tw8d2", name: "swap2_with_destination",
		actions: "RecordId,SetMinimumLegPrices,DeriverseSwap,DFlowDynamicRouteV1", firstAmount: 100279957, quoted: 100298162, slippage: 160, positiveSlippage: 5,
		inMint: "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", outMint: "CASHx9KJUStyftLFWGvEVf59SGeG9sh5FfcnZMVPCASH", destination: true},
	// the destination SOL delta equals quoted_out_amount (research §3.3)
	{sig: "38iesCRDJKFGDFCnYPpuZiJd4xFTArXoy33LqvyZNTpvqRnp9CTYqZbqtEDiwMboRdMM6kMN86Shjga9NJ1QyrKg", name: "swap2_with_destination_native",
		actions: "RecordId,SetMinimumLegPrices,DFlowDynamicRouteV1,DFlowDynamicRouteV1", firstAmount: 99992438, quoted: 798030331, slippage: 100, platformFee: 85, positiveSlippage: 5,
		inMint: "CASHx9KJUStyftLFWGvEVf59SGeG9sh5FfcnZMVPCASH", outMint: solMint, destination: true, dstNative: true},
}

// TestShredDFlowSwaps: ShredParser had no DFlow decoder although the docs
// listed DFlow. shred-24.
func TestShredDFlowSwaps(t *testing.T) {
	for _, c := range dflowCases {
		t.Run(c.sig[:8], func(t *testing.T) {
			tx := loadFixture(t, c.sig)
			var ix rawIx
			for _, x := range fixtureIxs(t, tx) {
				if x.programId == constants.DEX_PROGRAMS.DFLOW.ID && x.inner < 0 {
					ix = x
				}
			}
			for _, mode := range []string{"full", "pre-exec"} {
				in := tx
				if mode == "pre-exec" {
					in = preExec(t, tx)
				}
				res := parseShred(t, in, nil)
				raw := res.Instructions[constants.DEX_PROGRAMS.DFLOW.Name]
				if len(raw) != 1 {
					t.Fatalf("%s: DFlow events %d, want 1", mode, len(raw))
				}
				ev := raw[0].(*dflow.DFlowShredInstruction)
				d := ev.Data.(*dflow.DFlowSwapData)
				if ev.Type != c.name || strings.Join(d.Actions, ",") != c.actions || d.QuotedOutAmount != c.quoted || d.SlippageBps != c.slippage ||
					d.PlatformFeeBps != c.platformFee || d.PositiveSlippageFeeLimitPct != c.positiveSlippage || d.InputAmount != c.firstAmount || d.InputAmountUnknown != c.inAmountUnknown {
					t.Errorf("%s: decoded %s %+v", mode, ev.Type, d)
				}
				if d.User != ix.accounts[3] {
					t.Errorf("%s: user %s, want user_token_authority %s", mode, d.User, ix.accounts[3])
				}
				if c.destination && d.DestinationAccount != ix.accounts[4] {
					t.Errorf("%s: destination %s, want %s", mode, d.DestinationAccount, ix.accounts[4])
				}

				ins := oneTypedAt(t, res, constants.DEX_PROGRAMS.DFLOW.ID, utils.FormatIdx(ix.outer, -1))
				tr := ins.Trade
				wantKind := types.ShredAmountExact
				if c.inAmountUnknown {
					wantKind = types.ShredAmountUnknown
				}
				if tr.OutputToken.AmountRaw != u64str(c.quoted) || tr.InputToken.AmountRaw != u64str(c.firstAmount) || ins.InputAmountKind != wantKind || ins.OutputAmountKind != types.ShredAmountQuote {
					t.Errorf("%s: trade amounts %s/%s kinds %s/%s", mode, tr.InputToken.AmountRaw, tr.OutputToken.AmountRaw, ins.InputAmountKind, ins.OutputAmountKind)
				}
				if mode == "full" {
					// Mints from the SwapEvents (and destination) of the executed tx
					if tr.InputToken.Mint != c.inMint || tr.OutputToken.Mint != c.outMint {
						t.Errorf("full: mints %s -> %s, want %s -> %s", tr.InputToken.Mint, tr.OutputToken.Mint, c.inMint, c.outMint)
					}
					if want := utils.GetTradeType(c.inMint, c.outMint); tr.Type != want {
						t.Errorf("full: type %s, want %s", tr.Type, want)
					}
					continue
				}
				// Pre-execution: only the native destination names its mint,
				// and one known side does not give a route's direction (G4)
				wantOut := ""
				if c.dstNative {
					wantOut = solMint
				}
				if tr.InputToken.Mint != "" || tr.OutputToken.Mint != wantOut || tr.Type != types.TradeTypeSwap {
					t.Errorf("pre-exec: %s %q -> %q, want SWAP \"\" -> %q", tr.Type, tr.InputToken.Mint, tr.OutputToken.Mint, wantOut)
				}
			}
		})
	}

	// swap2_with_destination_native: the destination received quoted_out_amount
	c := dflowCases[len(dflowCases)-1]
	tx := loadFixture(t, c.sig)
	for _, x := range fixtureIxs(t, tx) {
		if x.programId == constants.DEX_PROGRAMS.DFLOW.ID && x.inner < 0 {
			if got := lamportDelta(tx, x.accounts[4]); got.Uint64() != c.quoted {
				t.Errorf("destination received %s lamports, want quoted_out_amount %d", got, c.quoted)
			}
		}
	}

	// The u64::MAX case: the fixture's first action amount is u64::MAX
	c = dflowCases[4]
	tx = loadFixture(t, c.sig)
	for _, x := range fixtureIxs(t, tx) {
		if x.programId == constants.DEX_PROGRAMS.DFLOW.ID && x.inner < 0 {
			// disc(8) + vec len(4) + tag(1) + amount
			if le64At(x.data, 13) != math.MaxUint64 {
				t.Errorf("fixture: first amount %d, want u64::MAX", le64At(x.data, 13))
			}
		}
	}
}

// TestShredDFlowUnknownAction: an action tag outside the IDL makes the rest
// of the arguments undecodable, so the swap is skipped rather than decoded
// from misaligned bytes. The test changes the only action tag of the real
// swap 2NcCFY7N to 200.
func TestShredDFlowUnknownAction(t *testing.T) {
	tx := preExec(t, loadFixture(t, dflowCases[0].sig))
	var ix rawIx
	for _, x := range fixtureIxs(t, tx) {
		if x.programId == constants.DEX_PROGRAMS.DFLOW.ID {
			ix = x
		}
	}
	// disc(8) + vec len u32(4) + tag
	data := append([]byte{}, ix.data...)
	if le32At(data, 8) != 1 || data[12] != 40 {
		t.Fatalf("fixture: actions %d tag %d, want 1 BisonFiSwap (40)", le32At(data, 8), data[12])
	}
	data[12] = 200
	setIxData(ix, data)
	res := parseShred(t, tx, nil)
	if n := len(res.Instructions[constants.DEX_PROGRAMS.DFLOW.Name]); n != 0 {
		t.Errorf("swap with an unknown action decoded (%d events)", n)
	}
}
