package tests

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/goccy/go-json"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Regression tests for the core follow-up (WP core2): no aliasing of cached
// balance changes, fetcher errors surfaced, wallet entries from constants.

// TestCore2BalanceChangesNotAliased: result.TokenBalanceChange,
// result.SolBalanceChange and the balance pointers of trades pointed into the
// adapter's cached maps, so editing one changed the others.
func TestCore2BalanceChangesNotAliased(t *testing.T) {
	p := dexparser.NewDexParser()
	checked := 0
	for _, sig := range fixtureSignatures(t, "json") {
		r := p.ParseAll(loadFixture(t, sig), nil)
		if r.AggregateTrade == nil {
			continue
		}
		before, _ := json.Marshal([]interface{}{r.SolBalanceChange, r.TokenBalanceChange})
		var amounts []*types.TokenAmount
		for _, tr := range append(append([]types.TradeInfo{}, r.Trades...), *r.AggregateTrade) {
			for _, tok := range []types.TokenInfo{tr.InputToken, tr.OutputToken} {
				amounts = append(amounts, tok.SourceBalance, tok.SourcePreBalance, tok.DestinationBalance, tok.DestinationPreBalance)
			}
		}
		for _, a := range amounts {
			if a != nil {
				a.Amount = "-1"
				if a.UIAmount != nil {
					*a.UIAmount = -1
				}
			}
		}
		after, _ := json.Marshal([]interface{}{r.SolBalanceChange, r.TokenBalanceChange})
		if string(before) != string(after) {
			t.Errorf("%s: editing trade balances changed the result's balance changes", sig[:8])
		}
		checked++
	}
	if checked < 50 {
		t.Fatalf("only %d fixtures with an aggregate trade", checked)
	}

	// the result's balance changes are copies of the adapter's cached maps
	tx := loadFixture(t, sigT22Fee)
	r := p.ParseAll(tx, nil)
	a := adapter.NewTransactionAdapter(tx, nil)
	signer := a.Signer()
	for mint, c := range r.TokenBalanceChange {
		c.Change.Amount = "0"
		if a.GetAccountTokenBalanceChanges(true)[signer][mint].Change.Amount == "0" {
			t.Errorf("TokenBalanceChange[%s] aliases the adapter cache", mint)
		}
	}
}

// TestCore2FetcherErrorsSurfaced: ALTsFetcher and TokenAccountsFetcher errors
// were dropped without a trace, and unresolved lookup accounts were only
// visible on the adapter. They are now listed in ParseResult.Warnings; State
// is unchanged.
func TestCore2FetcherErrorsSurfaced(t *testing.T) {
	p := dexparser.NewDexParser()
	if r := p.ParseAll(loadFixture(t, sigTwoLookups), nil); len(r.Warnings) != 0 {
		t.Errorf("complete tx: Warnings = %v", r.Warnings)
	}

	stripped := cloneTx(t, loadFixture(t, sigTwoLookups))
	stripped.Meta.LoadedAddresses = nil
	cfg := types.DefaultParseConfig()
	cfg.ALTsFetcher = types.NewALTsFetcher(types.FetchFilterAll, func([]types.AddressTableLookup) (map[string]*types.LoadedAddresses, error) {
		return nil, errors.New("rpc down")
	})
	r := p.ParseAll(stripped, &cfg)
	if !r.State || !hasWarning(r.Warnings, "ALTsFetcher: rpc down") || !hasWarning(r.Warnings, "unresolved address lookup table accounts") {
		t.Errorf("ALTsFetcher error: State=%v Warnings=%v", r.State, r.Warnings)
	}

	const dest = "4EHZFwbbVsHzsCN2QxoKgrqSHr41z5LCYxGhMeZ9mdXo"
	hidden := hideTokenAccount(t, loadFixture(t, sigT22Fee), dest)
	cfg = types.DefaultParseConfig()
	cfg.TokenAccountsFetcher = types.NewTokenAccountsFetcher(types.FetchFilterAll, func([]string) ([]*types.TokenAccountInfo, error) {
		return nil, errors.New("timeout")
	})
	r = p.ParseAll(hidden, &cfg)
	if !r.State || !hasWarning(r.Warnings, "TokenAccountsFetcher: timeout") {
		t.Errorf("TokenAccountsFetcher error: State=%v Warnings=%v", r.State, r.Warnings)
	}
}

// TestCore2FetcherWarningsRedactURLs: fetcher error text went into
// ParseResult.Warnings verbatim, and RPC client errors quote the endpoint URL,
// whose query, path or user info often holds an API key. URLs are now cut to
// scheme and host.
func TestCore2FetcherWarningsRedactURLs(t *testing.T) {
	const secret = "SECRETKEY123"
	errs := map[error]string{
		// net/http client error (*url.Error)
		&url.Error{Op: "Post", URL: "https://mainnet.helius-rpc.com/?api-key=" + secret, Err: errors.New("context deadline exceeded")}: `ALTsFetcher: Post "https://mainnet.helius-rpc.com": context deadline exceeded`,
		// key in the path, wrapped error
		fmt.Errorf("getMultipleAccounts: %w", errors.New("429 from https://solana-mainnet.g.alchemy.com/v2/"+secret+" retry later")): "ALTsFetcher: getMultipleAccounts: 429 from https://solana-mainnet.g.alchemy.com retry later",
		// key in the user info of a websocket URL
		errors.New("dial wss://user:" + secret + "@rpc.example.com:8900/ws failed"): "ALTsFetcher: dial wss://rpc.example.com:8900 failed",
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

func hasWarning(warnings []string, want string) bool {
	for _, w := range warnings {
		if w == want {
			return true
		}
	}
	return false
}

// TestCore2AuthoritiesFromConstants: GetDexInfo skipped a hardcoded list of
// wallet entries, so a wallet added to constants.KNOWN_AUTHORITIES was still
// treated as a route. The test registers the Raydium route program of 51nj as
// an authority for its duration.
func TestCore2AuthoritiesFromConstants(t *testing.T) {
	tx := loadFixture(t, sigT22Fee)
	info := func() types.DexInfo {
		a := adapter.NewTransactionAdapter(tx, nil)
		return utils.NewTransactionUtils(a).GetDexInfo(classifier.NewInstructionClassifier(a))
	}
	if got := info(); got.Route != "RaydiumRoute" {
		t.Fatalf("fixture: DexInfo %+v, want route RaydiumRoute", got)
	}
	saved := constants.KNOWN_AUTHORITIES
	constants.KNOWN_AUTHORITIES = append(append([]string{}, saved...), constants.DEX_PROGRAMS.RAYDIUM_ROUTE.ID)
	defer func() { constants.KNOWN_AUTHORITIES = saved }()
	if got := info(); got.Route != "" || got.AMM != "RaydiumCPMM" {
		t.Errorf("with the route listed as authority: DexInfo %+v, want AMM RaydiumCPMM", got)
	}
}
