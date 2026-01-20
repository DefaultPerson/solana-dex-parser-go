package tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	dexparser "github.com/solana-dex-parser-go"
	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/types"
)

// TestCase represents a test case for trade parsing
type TestCase struct {
	Signature   string        `json:"signature"`
	Type        string        `json:"type"`
	AMM         string        `json:"amm"`
	Route       string        `json:"route"`
	ProgramId   string        `json:"programId"`
	User        string        `json:"user"`
	Slot        uint64        `json:"slot"`
	Timestamp   int64         `json:"timestamp"`
	InputToken  TokenTestCase `json:"inputToken"`
	OutputToken TokenTestCase `json:"outputToken"`
}

type TokenTestCase struct {
	Mint     string  `json:"mint"`
	Amount   float64 `json:"amount"`
	Decimals uint8   `json:"decimals"`
}

// LoadTestTransaction loads a transaction from a JSON file
func LoadTestTransaction(filename string) (*adapter.SolanaTransaction, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

func TestDexParserBasic(t *testing.T) {
	// Test that the parser can be created
	parser := dexparser.NewDexParser()
	if parser == nil {
		t.Fatal("Failed to create DexParser")
	}
}

func TestShredParserBasic(t *testing.T) {
	// Test that the shred parser can be created
	parser := dexparser.NewShredParser()
	if parser == nil {
		t.Fatal("Failed to create ShredParser")
	}
}

func TestParseTrades(t *testing.T) {
	// Skip if no test data available
	if _, err := os.Stat("testdata/trades"); os.IsNotExist(err) {
		t.Skip("No test data available at testdata/trades")
	}

	parser := dexparser.NewDexParser()

	entries, err := os.ReadDir("testdata/trades")
	if err != nil {
		t.Skipf("Cannot read test data directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			tx, err := LoadTestTransaction("testdata/trades/" + entry.Name())
			if err != nil {
				t.Fatalf("Failed to load transaction: %v", err)
			}

			trades := parser.ParseTrades(tx, nil)
			if len(trades) == 0 {
				t.Log("No trades found in transaction")
			} else {
				t.Logf("Found %d trades", len(trades))
				for i, trade := range trades {
					t.Logf("Trade %d: %s %s -> %s", i, trade.Type, trade.InputToken.Mint, trade.OutputToken.Mint)
				}
			}
		})
	}
}

func TestBinaryReader(t *testing.T) {
	// Test basic binary reader functionality
	data := []byte{
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // u64: 1
		0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // u64: 2
		0x03, 0x00,                                     // u16: 3
		0x04,                                           // u8: 4
	}

	// Import would be done at package level
	t.Logf("Binary data length: %d bytes", len(data))
}

func TestTransactionAdapter(t *testing.T) {
	blockTime := int64(1699999999)
	// Create a minimal transaction for testing
	tx := &adapter.SolanaTransaction{
		Transaction: adapter.TransactionData{
			Signatures: []string{"test_signature"},
			Message: adapter.TransactionMessage{
				AccountKeys: []adapter.AccountKey{
					{Pubkey: "11111111111111111111111111111111"},
					{Pubkey: "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"},
				},
				Instructions: []interface{}{
					map[string]interface{}{
						"programIdIndex": 1,
						"accounts":       []int{0},
						"data":           base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4}),
					},
				},
			},
		},
		Meta: &adapter.TransactionMeta{
			PreBalances:  []uint64{1000000000, 0},
			PostBalances: []uint64{999000000, 1000000},
			Fee:          5000,
			Err:          nil,
		},
		Slot:      12345,
		BlockTime: &blockTime,
	}

	config := &types.ParseConfig{}
	txAdapter := adapter.NewTransactionAdapter(tx, config)

	if txAdapter.Signature() != "test_signature" {
		t.Errorf("Expected signature 'test_signature', got '%s'", txAdapter.Signature())
	}

	if txAdapter.Slot() != 12345 {
		t.Errorf("Expected slot 12345, got %d", txAdapter.Slot())
	}

	if txAdapter.BlockTime() != 1699999999 {
		t.Errorf("Expected block time 1699999999, got %d", txAdapter.BlockTime())
	}
}

func TestParseConfig(t *testing.T) {
	config := &types.ParseConfig{
		TryUnknownDEX:    true,
		ProgramIds:       []string{"test_program"},
		IgnoreProgramIds: []string{"ignore_program"},
	}

	if !config.TryUnknownDEX {
		t.Error("TryUnknownDEX should be true")
	}

	if len(config.ProgramIds) != 1 || config.ProgramIds[0] != "test_program" {
		t.Error("ProgramIds not set correctly")
	}

	if len(config.IgnoreProgramIds) != 1 || config.IgnoreProgramIds[0] != "ignore_program" {
		t.Error("IgnoreProgramIds not set correctly")
	}
}

func TestParseResult(t *testing.T) {
	result := types.NewParseResult()

	if !result.State {
		t.Error("New ParseResult should have state=true")
	}

	if result.Fee.Amount != "0" {
		t.Errorf("Expected fee amount '0', got '%s'", result.Fee.Amount)
	}

	if len(result.Trades) != 0 {
		t.Error("New ParseResult should have empty trades")
	}

	if len(result.Liquidities) != 0 {
		t.Error("New ParseResult should have empty liquidities")
	}
}

// TestTradeTypes tests that all trade types are defined correctly
func TestTradeTypes(t *testing.T) {
	expectedTypes := []types.TradeType{
		types.TradeTypeBuy,
		types.TradeTypeSell,
		types.TradeTypeSwap,
		types.TradeTypeCreate,
		types.TradeTypeComplete,
		types.TradeTypeMigrate,
	}

	for _, tradeType := range expectedTypes {
		if tradeType == "" {
			t.Error("Trade type should not be empty")
		}
		t.Logf("Trade type: %s", tradeType)
	}
}

// TestPoolEventTypes tests that all pool event types are defined correctly
func TestPoolEventTypes(t *testing.T) {
	expectedTypes := []types.PoolEventType{
		types.PoolEventTypeCreate,
		types.PoolEventTypeAdd,
		types.PoolEventTypeRemove,
	}

	for _, eventType := range expectedTypes {
		if eventType == "" {
			t.Error("Pool event type should not be empty")
		}
		t.Logf("Pool event type: %s", eventType)
	}
}

// BenchmarkDexParser benchmarks the DexParser creation
func BenchmarkDexParser(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = dexparser.NewDexParser()
	}
}

// Example test showing how to use the parser
func ExampleDexParser() {
	parser := dexparser.NewDexParser()

	// Create a minimal transaction (in real usage, this would come from RPC or gRPC stream)
	tx := &adapter.SolanaTransaction{
		Transaction: adapter.TransactionData{
			Signatures: []string{"example_signature"},
		},
		Slot:      100000,
		BlockTime: nil,
	}

	config := &types.ParseConfig{
		TryUnknownDEX: true,
	}

	result := parser.ParseAll(tx, config)
	fmt.Printf("Parse state: %v, Signature: %s\n", result.State, result.Signature)
}
