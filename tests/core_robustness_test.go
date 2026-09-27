package tests

import (
	"strings"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Jupiter v6 -> Pumpswap buy (v0, two address lookup tables)
const sigJupPumpswap = "5vjkR1vooYfu6SN5LTMCQT7fuhqS8DabTVivaN9u184bdHKYnY8FgbyEL92S6T56ZU3UxxecSJpQkRwTcFDj4QAi"

// v0 Jupiter swap with two lookup tables (writable and readonly entries)
const sigTwoLookups = "2d32B4VReyxfJf3J3MGRnJgzYJndY2TTi1JE2eMtyZxk2X89fBRw5PTCNr6VLKDV8L4mBFmwLMVVhPi3oqKXWwbK"

type panicTradeParser struct{}

func (panicTradeParser) ProcessTrades() []types.TradeInfo { panic("boom trade") }

type panicLiquidityParser struct{}

func (panicLiquidityParser) ProcessLiquidity() []types.PoolEvent { panic("boom liquidity") }

// TestCoreRecoveredPanicReturnsResult: a panic inside a parser used to make
// ParseAll return nil (unnamed result + recover), and ParseTrades /
// ParseLiquidity then dereferenced it and panicked out of the library.
func TestCoreRecoveredPanicReturnsResult(t *testing.T) {
	tx := loadFixture(t, sigJupPumpswap)
	p := dexparser.NewDexParser()
	p.RegisterTradeParser(constants.DEX_PROGRAMS.PUMP_SWAP.ID, func(*adapter.TransactionAdapter, types.DexInfo, map[string][]types.TransferData, []types.ClassifiedInstruction) parsers.TradeParser {
		return panicTradeParser{}
	})
	p.RegisterLiquidityParser(constants.DEX_PROGRAMS.PUMP_SWAP.ID, func(*adapter.TransactionAdapter, map[string][]types.TransferData, []types.ClassifiedInstruction) parsers.LiquidityParser {
		return panicLiquidityParser{}
	})
	// IgnoreProgramIds Jupiter so that the Pumpswap parser is reached
	cfg := &types.ParseConfig{ParseType: types.ParseAll(), IgnoreProgramIds: []string{constants.DEX_PROGRAMS.JUPITER.ID}}

	result := p.ParseAll(tx, cfg)
	if result == nil {
		t.Fatal("ParseAll returned nil after a recovered panic")
	}
	if result.State || !strings.Contains(result.Msg, "boom") || !strings.Contains(result.Msg, sigJupPumpswap) {
		t.Errorf("State=%v Msg=%q, want State=false and a message with the signature and the panic", result.State, result.Msg)
	}
	if result.Signature != sigJupPumpswap {
		t.Errorf("Signature=%q, want the tx signature", result.Signature)
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseTrades panicked: %v", r)
			}
		}()
		if trades := p.ParseTrades(tx, cfg); len(trades) != 0 {
			t.Errorf("ParseTrades after a panic = %d trades, want 0", len(trades))
		}
		if pools := p.ParseLiquidity(tx, cfg); len(pools) != 0 {
			t.Errorf("ParseLiquidity after a panic = %d events, want 0", len(pools))
		}
	}()

	// ThrowError re-panics
	throwCfg := *cfg
	throwCfg.ThrowError = true
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("ThrowError: expected the panic to propagate")
			}
		}()
		p.ParseAll(tx, &throwCfg)
	}()

	// ParseBatch hands a non-nil result to the callback
	results := p.ParseBatchWithCallback([]*adapter.SolanaTransaction{tx}, cfg, 1, func(_ int, _ *adapter.SolanaTransaction, r *types.ParseResult, _ error) bool {
		if r == nil || r.State {
			t.Errorf("ParseBatch callback result = %+v, want a failed result", r)
		}
		return true
	})
	if len(results) != 1 || results[0] == nil {
		t.Errorf("ParseBatch results = %v", results)
	}
}

// TestCoreNilTransaction: ParseAll(nil) used to panic out of the library
// (result.Slot = tx.Slot ran before the recover was installed).
func TestCoreNilTransaction(t *testing.T) {
	p := dexparser.NewDexParser()
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("nil tx panicked: %v", r)
			}
		}()
		result := p.ParseAll(nil, nil)
		if result == nil || result.State || result.Msg != "nil transaction" {
			t.Errorf("ParseAll(nil) = %+v, want State=false Msg=\"nil transaction\"", result)
		}
		if len(p.ParseTrades(nil, nil)) != 0 || len(p.ParseLiquidity(nil, nil)) != 0 || len(p.ParseTransfers(nil, nil)) != 0 {
			t.Error("Parse*(nil) should return no events")
		}
	}()
}

// TestCoreMalformedIndexes: out-of-range account, instruction and inner
// instruction indexes used to panic (index out of range) or make ParseAll
// return nil.
func TestCoreMalformedIndexes(t *testing.T) {
	p := dexparser.NewDexParser()
	base := loadFixture(t, sigTwoLookups)
	want := p.ParseTrades(base, tradesConfig())
	if len(want) == 0 {
		t.Fatal("fixture has no trades")
	}

	mutations := map[string]func(tx *adapter.SolanaTransaction){
		"inner set index out of range": func(tx *adapter.SolanaTransaction) {
			tx.Meta.InnerInstructions = append(tx.Meta.InnerInstructions, adapter.InnerInstructionSet{Index: 99, Instructions: []interface{}{
				map[string]interface{}{"programIdIndex": 3, "accounts": []interface{}{0, 1}, "data": "3Bxs4h24hBtQy9rw"},
			}})
		},
		"token balance account index out of range": func(tx *adapter.SolanaTransaction) {
			tx.Meta.PostTokenBalances = append(tx.Meta.PostTokenBalances, adapter.TokenBalance{AccountIndex: 500, Mint: usdcMint, Owner: "x"})
			tx.Meta.PreTokenBalances = append(tx.Meta.PreTokenBalances, adapter.TokenBalance{AccountIndex: -1, Mint: usdcMint, Owner: "x"})
		},
		"instruction account index out of range": func(tx *adapter.SolanaTransaction) {
			tx.Transaction.Message.Instructions = append(tx.Transaction.Message.Instructions,
				map[string]interface{}{"programIdIndex": 250, "accounts": []interface{}{-3, 700}, "data": ""})
		},
		"balances shorter than account keys": func(tx *adapter.SolanaTransaction) {
			tx.Meta.PreBalances = tx.Meta.PreBalances[:2]
			tx.Meta.PostBalances = tx.Meta.PostBalances[:1]
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			tx := cloneTx(t, base)
			mutate(tx)
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panicked: %v", r)
				}
			}()
			result := p.ParseAll(tx, nil)
			if result == nil || !result.State {
				t.Fatalf("ParseAll = %+v, want State=true", result)
			}
			if name != "balances shorter than account keys" {
				got := p.ParseTrades(tx, tradesConfig())
				if len(got) != len(want) || tradeKey(got[0]) != tradeKey(want[0]) {
					t.Errorf("trades changed: got %d, want %d", len(got), len(want))
				}
			}
		})
	}

	a := adapter.NewTransactionAdapter(base, nil)
	if a.GetInnerInstruction(0, -1) != nil || a.GetInnerInstruction(99, 0) != nil || a.InstructionAt(-1) != nil || a.InstructionAt(1000) != nil {
		t.Error("out-of-range instruction lookups should return nil")
	}
	if a.GetAccountKey(-1) != "" || a.GetAccountKey(10000) != "" {
		t.Error("out-of-range GetAccountKey should return \"\"")
	}
}

// TestCoreV0WithoutLoadedAddresses: a v0 transaction without
// meta.loadedAddresses (pre-execution data) used to panic in
// NewTransactionAdapter (index out of range on token balances).
func TestCoreV0WithoutLoadedAddresses(t *testing.T) {
	tx := cloneTx(t, loadFixture(t, sigTwoLookups))
	tx.Meta.LoadedAddresses = nil
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked: %v", r)
		}
	}()
	a := adapter.NewTransactionAdapter(tx, nil)
	if !a.HasUnresolvedAccounts() {
		t.Error("HasUnresolvedAccounts() = false for unresolved lookups")
	}
	// static keys, then one "" per unresolved lookup index
	if want := len(tx.Transaction.Message.AccountKeys) + 6 + 9; len(a.AccountKeys) != want {
		t.Errorf("AccountKeys = %d, want %d (unresolved positions kept)", len(a.AccountKeys), want)
	}
	result := dexparser.NewDexParser().ParseAll(tx, nil)
	if result == nil || !result.State {
		t.Fatalf("ParseAll = %+v", result)
	}
	_ = utils.NewTransactionUtils(a).GetTransferActions(nil)
}
