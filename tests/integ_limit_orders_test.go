package tests

import (
	"encoding/binary"
	"strconv"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// Limit orders on AMMs (amm R3-G1, streamer item 16): Raydium CLMM
// open/settle_limit_order and Meteora DLMM place/cancel_limit_order are
// neither trades nor liquidity events. They are reported as transfers with
// a Type naming the action, like Jupiter limit orders; before, they came out
// as untyped transfers. Real transactions found through the programs'
// open limit-order accounts (getProgramAccounts). Truth: the instructions'
// IDL accounts and arguments, and the DLMM CancelLimitOrderEvt.
func TestIntegAMMLimitOrders(t *testing.T) {
	cases := []struct {
		name, sig, idx, transferType string
		program                      string
		source, destination, mint    int // instruction account indexes
		amount                       func(data []byte) string
	}{
		// open_limit_order: 5 input_token_account, 6 input_vault,
		// 7 input_vault_mint; args nonce_index u8, zero_for_one bool,
		// tick_index i32, amount u64
		{"clmm open", "2P1EgTYZsf4VdtYa755X4WYCeELXswDob1HsELfz791BYftKWQD3Pa3TNGRUYXcMArHDPrZwboWcUoV6WAvzBdc6", "3", "openLimitOrder",
			constants.DEX_PROGRAMS.RAYDIUM_CL.ID, 5, 6, 7, func(d []byte) string { return strconv.FormatUint(binary.LittleEndian.Uint64(d[14:22]), 10) }},
		// settle_limit_order (by a keeper): 4 output_token_account,
		// 5 output_vault, 6 output_vault_mint
		{"clmm settle", "2vKxn9cgQssaseSvjhtsbMjcAScHoqYburzuSevUUfc46MLmZuqpvBUPz34kGzdGoXzwuPygVAjqzZZoZ2uw4yxZ", "0", "settleLimitOrder",
			constants.DEX_PROGRAMS.RAYDIUM_CL.ID, 5, 4, 6, nil},
		// decrease_limit_order: 4 input_token_account, 6 input_vault,
		// 8 input_vault_mint; args amount u64, amount_min u64 (the unfilled
		// input paid back)
		{"clmm decrease", "59wbSySGRRB6kR5y9uqUJxW7WanEeTqrz8GKGvb34gEUhXBxXt1tpYaBHgZDfPguNMdMW8Ky7qAFLtBLhrPhJx1Y", "4", "decreaseLimitOrder",
			constants.DEX_PROGRAMS.RAYDIUM_CL.ID, 6, 4, 8, func(d []byte) string { return strconv.FormatUint(binary.LittleEndian.Uint64(d[8:16]), 10) }},
		// place_limit_order (by CPI): 2 reserve, 3 token_mint, 7 user_token
		{"dlmm place", "5HxKQKDJCC78xtJ9adhebsUETFDT5HgdMKPgJMgvNz34dBBrt1vSooZxc3HquadjSqCPhmA91yTSnjoK18CeHWsn", "4-1", "placeLimitOrder",
			constants.DEX_PROGRAMS.METEORA.ID, 7, 2, 3, nil},
		// cancel_limit_order: 2 reserve_x, 4 token_x_mint, 7 owner_token_x
		{"dlmm cancel", "5HxKQKDJCC78xtJ9adhebsUETFDT5HgdMKPgJMgvNz34dBBrt1vSooZxc3HquadjSqCPhmA91yTSnjoK18CeHWsn", "2-1", "cancelLimitOrder",
			constants.DEX_PROGRAMS.METEORA.ID, 2, 7, 4, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := loadFixture(t, tc.sig)
			result := dexparser.NewDexParser().ParseAll(tx, nil)
			if len(result.Trades) != 0 || len(result.Liquidities) != 0 || result.AggregateTrade != nil {
				t.Errorf("%d trades, %d liquidity events: a limit order is neither", len(result.Trades), len(result.Liquidities))
			}
			var ix *rawIx
			for _, x := range fixtureIxs(t, tx) {
				x := x
				if x.programId == tc.program && idxOf(x.outer, x.inner) == tc.idx {
					ix = &x
				}
			}
			if ix == nil {
				t.Fatalf("no instruction of %s at %s", tc.program, tc.idx)
			}
			n := 0
			for _, tr := range result.Transfers {
				if tr.Idx != tc.idx {
					continue
				}
				n++
				a := ix.accounts
				if tr.Type != tc.transferType || tr.ProgramId != tc.program || tr.Info.Source != a[tc.source] ||
					tr.Info.Destination != a[tc.destination] || tr.Info.Mint != a[tc.mint] {
					t.Errorf("transfer %s %s %s -> %s %s, want %s %s -> %s %s", tr.Type, tr.ProgramId, tr.Info.Source, tr.Info.Destination, tr.Info.Mint,
						tc.transferType, a[tc.source], a[tc.destination], a[tc.mint])
				}
				if tc.amount != nil && tr.Info.TokenAmount.Amount != tc.amount(ix.data) {
					t.Errorf("amount %s, want the instruction's %s", tr.Info.TokenAmount.Amount, tc.amount(ix.data))
				}
			}
			if n != 1 {
				t.Errorf("%d transfers at %s, want 1", n, tc.idx)
			}
		})
	}

	// close_limit_order moves no tokens: nothing is reported for it (59wbSySG
	// closes after its decrease at 5, 5N9K48hm is a keeper's close)
	for _, sig := range []string{
		"59wbSySGRRB6kR5y9uqUJxW7WanEeTqrz8GKGvb34gEUhXBxXt1tpYaBHgZDfPguNMdMW8Ky7qAFLtBLhrPhJx1Y",
		"5N9K48hmhL1SXsAQhLuBcN6VPuQzXbKE5fgudoDZvS2QetEp9tM4FGHVM3ddGAEWCXX4JoEEtHxHm3anJGm6JoXC",
	} {
		r := dexparser.NewDexParser().ParseAll(loadFixture(t, sig), nil)
		for _, tr := range r.Transfers {
			if tr.Type != "decreaseLimitOrder" {
				t.Errorf("%s: transfer %s at %s, want none for close_limit_order", sig[:8], tr.Type, tr.Idx)
			}
		}
		if len(r.Trades) != 0 || len(r.Liquidities) != 0 {
			t.Errorf("%s: %d trades, %d liquidity events", sig[:8], len(r.Trades), len(r.Liquidities))
		}
	}

	// The cancel paid back what CancelLimitOrderEvt reports (lb_pair, from,
	// limit_order, amounts [x, y], ...): amounts[0] of token X
	tx := loadFixture(t, "5HxKQKDJCC78xtJ9adhebsUETFDT5HgdMKPgJMgvNz34dBBrt1vSooZxc3HquadjSqCPhmA91yTSnjoK18CeHWsn")
	var event []byte
	for _, x := range fixtureIxs(t, tx) {
		if x.programId == constants.DEX_PROGRAMS.METEORA.ID && constants.IsAnchorEvent(x.data) &&
			string(x.data[8:16]) == string([]byte{131, 234, 194, 133, 9, 14, 189, 209}) {
			event = x.data[16:]
		}
	}
	if len(event) < 3*32+16 {
		t.Fatal("no CancelLimitOrderEvt")
	}
	x := strconv.FormatUint(binary.LittleEndian.Uint64(event[96:104]), 10)
	for _, tr := range dexparser.NewDexParser().ParseAll(tx, nil).Transfers {
		if tr.Type == "cancelLimitOrder" && tr.Info.TokenAmount.Amount != x {
			t.Errorf("cancel transfer %s, event amount %s", tr.Info.TokenAmount.Amount, x)
		}
	}
}

func idxOf(outer, inner int) string {
	if inner < 0 {
		return strconv.Itoa(outer)
	}
	return strconv.Itoa(outer) + "-" + strconv.Itoa(inner)
}

// increase_limit_order (IDL accounts: owner 0, pool_state 1, tick_array 2,
// limit_order 3, input_token_account 4, input_vault 5, input_vault_mint 6,
// input_token_program 7; args amount u64) is reported like an open.
// Synthetic, as no real increase was found (90 transactions of the most
// active limit-order owners scanned): the real open 2P1EgTYZ... with its
// instruction rewritten as an increase of the same amount and accounts.
func TestIntegCLMMIncreaseLimitOrder(t *testing.T) {
	tx := cloneTx(t, loadFixture(t, "2P1EgTYZsf4VdtYa755X4WYCeELXswDob1HsELfz791BYftKWQD3Pa3TNGRUYXcMArHDPrZwboWcUoV6WAvzBdc6"))
	ix := tx.Transaction.Message.Instructions[3].(map[string]interface{})
	data, err := base58.Decode(ix["data"].(string))
	if err != nil || !constants.MatchDiscriminator(data, constants.DISCRIMINATORS.RAYDIUM_CL.LIMIT_ORDER.OPEN_LIMIT_ORDER) {
		t.Fatal("3 is not an open_limit_order")
	}
	amount := data[14:22]
	ix["data"] = base58.Encode(append(append([]byte{}, constants.DISCRIMINATORS.RAYDIUM_CL.LIMIT_ORDER.INCREASE_LIMIT_ORDER...), amount...))
	// open: payer 0, pool 1, tick_array 2, nonce 3, limit_order 4, input 5,
	// vault 6, mint 7, token program 8, system 9
	open := ix["accounts"].([]interface{})
	ix["accounts"] = []interface{}{open[0], open[1], open[2], open[4], open[5], open[6], open[7], open[8]}

	result := dexparser.NewDexParser().ParseAll(tx, nil)
	keys := rawAccountKeys(tx)
	accounts := ix["accounts"].([]interface{})
	key := func(i int) string { return keys[jsonInt(accounts[i])] }
	if len(result.Trades) != 0 || len(result.Liquidities) != 0 || len(result.Transfers) != 1 {
		t.Fatalf("%d trades, %d liquidity events, %d transfers", len(result.Trades), len(result.Liquidities), len(result.Transfers))
	}
	tr := result.Transfers[0]
	if tr.Type != "increaseLimitOrder" || tr.Idx != "3" || tr.Info.Source != key(4) || tr.Info.Destination != key(5) ||
		tr.Info.Mint != key(6) || tr.Info.TokenAmount.Amount != strconv.FormatUint(binary.LittleEndian.Uint64(amount), 10) {
		t.Errorf("transfer %+v", tr)
	}
}
