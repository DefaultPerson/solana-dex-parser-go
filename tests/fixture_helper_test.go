package tests

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Fixtures are getTransaction results of real mainnet transactions stored as
// testdata/tx/<signature>.json.gz ("json" encoding) or
// testdata/tx/<signature>.parsed.json.gz ("jsonParsed" encoding).
// Tests run offline. Set SDP_FETCH_FIXTURES=1 to download missing fixtures from
// SOLANA_RPC_URL (default: the public mainnet RPC) and store them.
const fixtureDir = "../testdata/tx"

const defaultFixtureRPC = "https://api.mainnet-beta.solana.com"

// loadFixture returns the "json"-encoded fixture for sig.
func loadFixture(t testing.TB, sig string) *adapter.SolanaTransaction {
	t.Helper()
	return loadFixtureEncoding(t, sig, "json")
}

// loadParsedFixture returns the "jsonParsed"-encoded fixture for sig.
func loadParsedFixture(t testing.TB, sig string) *adapter.SolanaTransaction {
	t.Helper()
	return loadFixtureEncoding(t, sig, "jsonParsed")
}

func fixturePath(sig, encoding string) string {
	if encoding == "jsonParsed" {
		return filepath.Join(fixtureDir, sig+".parsed.json.gz")
	}
	return filepath.Join(fixtureDir, sig+".json.gz")
}

// fixtureSignatures lists the signatures of all stored fixtures for encoding
// ("json" or "jsonParsed"), sorted.
func fixtureSignatures(t testing.TB, encoding string) []string {
	t.Helper()
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read %s: %v", fixtureDir, err)
	}
	var sigs []string
	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasSuffix(name, ".parsed.json.gz"):
			if encoding == "jsonParsed" {
				sigs = append(sigs, strings.TrimSuffix(name, ".parsed.json.gz"))
			}
		case strings.HasSuffix(name, ".json.gz"):
			if encoding == "json" {
				sigs = append(sigs, strings.TrimSuffix(name, ".json.gz"))
			}
		}
	}
	sort.Strings(sigs)
	return sigs
}

func loadFixtureEncoding(t testing.TB, sig, encoding string) *adapter.SolanaTransaction {
	t.Helper()
	raw, err := readFixture(sig, encoding)
	if err != nil && os.IsNotExist(err) && os.Getenv("SDP_FETCH_FIXTURES") == "1" {
		raw, err = fetchFixture(sig, encoding)
		if err == nil {
			err = writeFixture(sig, encoding, raw)
		}
	}
	if err != nil {
		t.Fatalf("fixture %s (%s): %v", sig, encoding, err)
	}
	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(raw, &tx); err != nil {
		t.Fatalf("fixture %s (%s): unmarshal: %v", sig, encoding, err)
	}
	return &tx
}

func readFixture(sig, encoding string) ([]byte, error) {
	f, err := os.Open(fixturePath(sig, encoding))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func writeFixture(sig, encoding string, raw []byte) error {
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(fixturePath(sig, encoding), buf.Bytes(), 0o644)
}

// fetchFixture downloads a transaction with maxSupportedTransactionVersion 1,
// retrying on rate limits.
func fetchFixture(sig, encoding string) ([]byte, error) {
	rpcURL := os.Getenv("SOLANA_RPC_URL")
	if rpcURL == "" {
		rpcURL = defaultFixtureRPC
	}
	reqBody, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getTransaction",
		"params": []interface{}{sig, map[string]interface{}{
			"encoding":                       encoding,
			"commitment":                     "confirmed",
			"maxSupportedTransactionVersion": 1,
		}},
	})
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}
		resp, err := client.Post(rpcURL, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited")
			continue
		}
		var rpcResp struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &rpcResp); err != nil {
			return nil, fmt.Errorf("decode RPC response: %w", err)
		}
		if rpcResp.Error != nil {
			return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
		}
		if len(rpcResp.Result) == 0 || string(rpcResp.Result) == "null" {
			return nil, fmt.Errorf("transaction not found")
		}
		return rpcResp.Result, nil
	}
	return nil, lastErr
}

// parseContext bundles what parsers receive inside DexParser, for parser-level tests.
type parseContext struct {
	Adapter         *adapter.TransactionAdapter
	Classifier      *classifier.InstructionClassifier
	Utils           *utils.TransactionUtils
	TransferActions map[string][]types.TransferData
	DexInfo         types.DexInfo
}

// newParseContext builds the adapter, classifier and transfer actions the same way DexParser does.
func newParseContext(tx *adapter.SolanaTransaction, cfg *types.ParseConfig) *parseContext {
	if cfg == nil {
		c := types.DefaultParseConfig()
		cfg = &c
	}
	a := adapter.NewTransactionAdapter(tx, cfg)
	u := utils.NewTransactionUtils(a)
	c := classifier.NewInstructionClassifier(a)
	return &parseContext{
		Adapter:         a,
		Classifier:      c,
		Utils:           u,
		TransferActions: u.GetTransferActions([]string{"mintTo", "burn", "mintToChecked", "burnChecked"}),
		DexInfo:         u.GetDexInfo(c),
	}
}
