package tests

import (
	"reflect"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// TestFixtureSmoke parses every stored fixture (both encodings) with the
// default config and with everything enabled: no panic, a non-nil result with
// State=true. For signatures stored in both encodings, trades, liquidities and
// meme events must be equal.
func TestFixtureSmoke(t *testing.T) {
	p := dexparser.NewDexParser()
	all := &types.ParseConfig{ParseType: types.ParseAll(), TryUnknownDEX: true, IncludeFailedTxs: true}
	results := map[string]map[string]*types.ParseResult{}
	for _, encoding := range []string{"json", "jsonParsed"} {
		for _, sig := range fixtureSignatures(t, encoding) {
			tx := loadFixtureEncoding(t, sig, encoding)
			for name, cfg := range map[string]*types.ParseConfig{"default": nil, "all": all} {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("%s %s %s: panic %v", sig, encoding, name, r)
						}
					}()
					r := p.ParseAll(tx, cfg)
					if r == nil || !r.State {
						t.Errorf("%s %s %s: result %+v", sig, encoding, name, r)
						return
					}
					if r.Signature != sig {
						t.Errorf("%s %s %s: signature %s", sig, encoding, name, r.Signature)
					}
					if name == "all" {
						if results[sig] == nil {
							results[sig] = map[string]*types.ParseResult{}
						}
						results[sig][encoding] = r
					}
				}()
			}
		}
	}
	both := 0
	for sig, byEnc := range results {
		j, jp := byEnc["json"], byEnc["jsonParsed"]
		if j == nil || jp == nil {
			continue
		}
		both++
		if !reflect.DeepEqual(j.Trades, jp.Trades) || !reflect.DeepEqual(j.AggregateTrade, jp.AggregateTrade) {
			t.Errorf("%s: trades differ between json and jsonParsed", sig)
		}
		if !reflect.DeepEqual(j.Liquidities, jp.Liquidities) {
			t.Errorf("%s: liquidities differ between json and jsonParsed", sig)
		}
		if !reflect.DeepEqual(j.MemeEvents, jp.MemeEvents) {
			t.Errorf("%s: meme events differ between json and jsonParsed", sig)
		}
	}
	if both == 0 {
		t.Error("no signature stored in both encodings")
	}
}
