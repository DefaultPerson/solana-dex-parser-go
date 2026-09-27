package tests

import (
	"errors"
	"strings"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Fetcher errors reach ParseResult.Warnings with every URL cut to scheme
// and host. Two forms leaked a secret: a JSON-escaped URL (an RPC error body
// quoted verbatim) was not recognised, and a password containing '@' left
// its tail in front of the host.
func TestIntegFetcherWarningRedaction(t *testing.T) {
	const secret = "SECRETKEY123"
	errs := map[error]string{
		errors.New(`{"error":"bad gateway at https:\/\/rpc.example.com\/v2\/` + secret + `"}`): `ALTsFetcher: {"error":"bad gateway at https://rpc.example.com"}`,
		errors.New("dial wss://user:p@" + secret + "@rpc.example.com:8900/ws failed"):          "ALTsFetcher: dial wss://rpc.example.com:8900 failed",
	}
	p := dexparser.NewDexParser()
	for fetchErr, want := range errs {
		stripped := cloneTx(t, loadFixture(t, sigTwoLookups))
		stripped.Meta.LoadedAddresses = nil
		cfg := types.DefaultParseConfig()
		cfg.ALTsFetcher = types.NewALTsFetcher(types.FetchFilterAll, func([]types.AddressTableLookup) (map[string]*types.LoadedAddresses, error) {
			return nil, fetchErr
		})
		r := p.ParseAll(stripped, &cfg)
		if !hasWarning(r.Warnings, want) || strings.Contains(strings.Join(r.Warnings, " "), secret) {
			t.Errorf("error %q: Warnings = %q, want %q", fetchErr, r.Warnings, want)
		}
	}
}

// result.Transfers balances are independent copies: they shared their
// UIAmount pointer with the caller's tx.Meta token balances (the adapter
// copied the TokenAmount shallowly), and the Jupiter limit order / VA
// transfers pointed into the adapter's cached balance-change maps, so two
// transfers could share one balance. Checked on every fixture, ParseAll and
// ParseTransfers.
func TestIntegTransferBalancesAreCopies(t *testing.T) {
	p := dexparser.NewDexParser()
	checked := 0
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		meta := map[*float64]bool{}
		if tx.Meta != nil {
			for _, list := range [][]adapter.TokenBalance{tx.Meta.PreTokenBalances, tx.Meta.PostTokenBalances} {
				for i := range list {
					if list[i].UiTokenAmount.UIAmount != nil {
						meta[list[i].UiTokenAmount.UIAmount] = true
					}
				}
			}
		}
		for _, transfers := range [][]types.TransferData{p.ParseAll(tx, nil).Transfers, p.ParseTransfers(tx, nil)} {
			seen := map[*types.TokenAmount]string{}
			for i := range transfers {
				info := &transfers[i].Info
				for name, b := range map[string]*types.TokenAmount{
					"SourceBalance": info.SourceBalance, "SourcePreBalance": info.SourcePreBalance,
					"DestinationBalance": info.DestinationBalance, "DestinationPreBalance": info.DestinationPreBalance,
				} {
					if b == nil {
						continue
					}
					checked++
					where := transfers[i].Idx + " " + name
					if other, ok := seen[b]; ok {
						t.Errorf("%s: %s and %s share one balance", sig[:8], other, where)
					}
					seen[b] = where
					if b.UIAmount != nil && meta[b.UIAmount] {
						t.Errorf("%s: %s UIAmount points into tx.Meta", sig[:8], where)
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no transfer balances checked")
	}
}
