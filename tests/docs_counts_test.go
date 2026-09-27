package tests

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// docsCounts are the numbers the README states about the library's coverage,
// computed from the code
type docsCounts struct {
	dexProgramIDs   int // unique entries of constants.DEX_PROGRAM_IDS
	labelPrograms   int // constants.JUPITER_LABEL_PROGRAMS
	botPrograms     int // programs tagged "bot" (DEX_PROGRAMS and BOT_ROUTER_PROGRAMS)
	bots            int // bots with fee accounts (constants.BOT_FEE_ACCOUNTS)
	botFeeAccounts  int
	tipProviders    int
	tipAccounts     int
	tradeParsers    int // program IDs with a default trade parser
	liquidityParser int
	transferParsers int
	memeParsers     int
	routeParsers    int // aggregator route parsers (RegisterRouteParser)
}

// registeredParserPrograms returns, per factory map of DexParser
// (tradeParserFactories, ...), the program IDs registered in
// registerDefaultParsers, read from dex_parser.go
func registeredParserPrograms(t *testing.T) map[string][]string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "../dex_parser.go", nil, 0)
	if err != nil {
		t.Fatalf("parse dex_parser.go: %v", err)
	}
	found := map[string]map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "registerDefaultParsers" {
			return true
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok || len(assign.Lhs) != 1 {
				return true
			}
			index, ok := assign.Lhs[0].(*ast.IndexExpr)
			if !ok {
				return true
			}
			sel, ok := index.X.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if found[sel.Sel.Name] == nil {
				found[sel.Sel.Name] = map[string]bool{}
			}
			// constants.DEX_PROGRAMS.X.ID -> X
			key := exprString(index.Index)
			found[sel.Sel.Name][key] = true
			return true
		})
		return false
	})
	result := map[string][]string{}
	for kind, keys := range found {
		for key := range keys {
			result[kind] = append(result[kind], key)
		}
		sort.Strings(result[kind])
	}
	return result
}

func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.Ident:
		return v.Name
	case *ast.BasicLit:
		return v.Value
	}
	return fmt.Sprintf("%T", e)
}

func computeDocsCounts(t *testing.T) docsCounts {
	t.Helper()
	var c docsCounts

	unique := map[string]bool{}
	for _, id := range constants.DEX_PROGRAM_IDS {
		unique[id] = true
	}
	c.dexProgramIDs = len(unique)
	c.labelPrograms = len(constants.JUPITER_LABEL_PROGRAMS)

	botIDs := map[string]bool{}
	for id := range unique {
		p := constants.GetDexProgramByID(id)
		for _, tag := range p.Tags {
			if tag == "bot" {
				botIDs[id] = true
			}
		}
	}
	c.botPrograms = len(botIDs)

	c.bots = len(constants.BOT_FEE_ACCOUNTS)
	c.botFeeAccounts = len(constants.GetAllBotFeeAccounts())
	c.tipProviders = len(constants.TIP_ACCOUNTS)
	for _, accounts := range constants.TIP_ACCOUNTS {
		c.tipAccounts += len(accounts)
	}

	parsers := registeredParserPrograms(t)
	c.tradeParsers = len(parsers["tradeParserFactories"])
	c.liquidityParser = len(parsers["liquidityParserFactories"])
	c.transferParsers = len(parsers["transferParserFactories"])
	c.memeParsers = len(parsers["memeEventParserFactories"])
	c.routeParsers = len(parsers["routeParserFactories"])
	return c
}

// readmeCountsSentence is the README sentence that states the counts
func readmeCountsSentence(c docsCounts) string {
	return fmt.Sprintf("It knows **%d DEX program IDs** (%d of them from the Jupiter venue label list and %d trading bot programs), "+
		"registers default parsers for **%d trade**, **%d liquidity**, **%d transfer**, **%d meme event** and **%d aggregator route** programs, "+
		"attributes trades to **%d trading bots** through %d fee accounts, and reports tips paid to %d accounts of %d transaction-landing providers.",
		c.dexProgramIDs, c.labelPrograms, c.botPrograms,
		c.tradeParsers, c.liquidityParser, c.transferParsers, c.memeParsers, c.routeParsers,
		c.bots, c.botFeeAccounts, c.tipAccounts, c.tipProviders)
}

// TestDocsReadmeCounts checks that the coverage numbers stated in README.md
// match the code (constants-20: the README said 55 program IDs while
// DEX_PROGRAM_IDS had 63). On failure it prints the sentence to paste.
func TestDocsReadmeCounts(t *testing.T) {
	c := computeDocsCounts(t)
	for kind, ids := range registeredParserPrograms(t) {
		t.Logf("%s (%d): %s", kind, len(ids), strings.Join(ids, " "))
	}
	want := readmeCountsSentence(c)
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), want) {
		t.Errorf("README.md does not state the current counts; expected the sentence:\n%s", want)
	}
}
