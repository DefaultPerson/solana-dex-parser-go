package tests

import (
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// A PumpSwap trade in a pool quoted in a token other than SOL or a
// stablecoin takes its type from the event: a buy of the pool's base token
// is BUY, a sell SELL. Before, PumpSwap typed every trade whose input was
// not SOL, USDC or USDT as SELL, and Jupiter hops copy the venue's type.
// Synthetic: no such pool is in the fixtures. Built from the real buy
// 1bdrt5y1... and sell 2F1tqd2p... of WSOL-quoted pools, with WSOL renamed
// to the PUMP mint (6 decimals, as in the real 24U4tyX7...) in the account
// keys and token balances.
func TestIntegPumpSwapTokenQuotedType(t *testing.T) {
	const pumpMint = "pumpCmXqMfrsAkQ5r49WcJnRayYRqmXz6ae8H7H9Dfn"
	cases := []struct {
		sig  string
		want types.TradeType
	}{
		{"1bdrt5y1Nz5tRygDaEBxTv8wCL8nPazSm6LRRpTPG74tX6DzPtmvx55iTqaX7Uc8B3x2q7oxZ5yc9hvPrQgUALv", types.TradeTypeBuy},
		{"2F1tqd2pi7BRCEsffg3nvtQdfuA3dbwNaT8aSGtyjVd4p2y4YZh8ZfvTbWU4CqPjDAij9Qdg58GkX35paoH79XdE", types.TradeTypeSell},
	}
	for _, tc := range cases {
		tx := cloneTx(t, loadFixture(t, tc.sig))
		for i := range tx.Transaction.Message.AccountKeys {
			if tx.Transaction.Message.AccountKeys[i].Pubkey == constants.TOKENS.SOL {
				tx.Transaction.Message.AccountKeys[i].Pubkey = pumpMint
			}
		}
		if tx.Meta.LoadedAddresses != nil {
			for _, list := range [][]string{tx.Meta.LoadedAddresses.Writable, tx.Meta.LoadedAddresses.Readonly} {
				for i := range list {
					if list[i] == constants.TOKENS.SOL {
						list[i] = pumpMint
					}
				}
			}
		}
		for _, balances := range [][]adapter.TokenBalance{tx.Meta.PreTokenBalances, tx.Meta.PostTokenBalances} {
			for i := range balances {
				if balances[i].Mint == constants.TOKENS.SOL {
					balances[i].Mint, balances[i].UiTokenAmount.Decimals, balances[i].UiTokenAmount.UIAmount = pumpMint, 6, nil
				}
			}
		}

		var trades []types.TradeInfo
		for _, tr := range dexparser.NewDexParser().ParseAll(tx, nil).Trades {
			if tr.ProgramId == constants.DEX_PROGRAMS.PUMP_SWAP.ID {
				trades = append(trades, tr)
			}
		}
		if len(trades) != 1 {
			t.Fatalf("%s: %d PumpSwap trades, want 1", tc.sig[:8], len(trades))
		}
		tr := trades[0]
		quote := tr.InputToken.Mint
		if tc.want == types.TradeTypeSell {
			quote = tr.OutputToken.Mint
		}
		if tr.Type != tc.want || quote != pumpMint {
			t.Errorf("%s: %s %s -> %s, want %s with the PUMP quote", tc.sig[:8], tr.Type, tr.InputToken.Mint, tr.OutputToken.Mint, tc.want)
		}
	}
}
