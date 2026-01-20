package tests

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	dexparser "github.com/solana-dex-parser-go"
	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/types"
)

// GRPCTransaction represents a transaction from Helius Laserstream/gRPC
type GRPCTransaction struct {
	Signature   []byte
	IsVote      bool
	Transaction GRPCTransactionData
	Meta        GRPCTransactionMeta
}

type GRPCTransactionData struct {
	Signatures [][]byte
	Message    GRPCMessageData
}

type GRPCMessageData struct {
	Header              GRPCHeader
	AccountKeys         [][]byte
	RecentBlockhash     []byte
	Instructions        []GRPCInstruction
	Versioned           bool
	AddressTableLookups []GRPCAddressTableLookup
}

type GRPCHeader struct {
	NumRequiredSignatures       int
	NumReadonlySignedAccounts   int
	NumReadonlyUnsignedAccounts int
}

type GRPCInstruction struct {
	ProgramIdIndex int
	Accounts       []byte
	Data           []byte
}

type GRPCAddressTableLookup struct {
	AccountKey      []byte
	WritableIndexes []byte
	ReadonlyIndexes []byte
}

type GRPCTransactionMeta struct {
	Err                  interface{}
	Fee                  uint64
	PreBalances          []uint64
	PostBalances         []uint64
	PreTokenBalances     []GRPCTokenBalance
	PostTokenBalances    []GRPCTokenBalance
	InnerInstructions    []GRPCInnerInstructionSet
	LogMessages          []string
	LoadedAddresses      GRPCLoadedAddresses
	ReturnData           *GRPCReturnData
	ComputeUnitsConsumed uint64
}

type GRPCTokenBalance struct {
	AccountIndex  int
	Mint          string
	Owner         string
	UiTokenAmount GRPCTokenAmount
}

type GRPCTokenAmount struct {
	Amount         string
	Decimals       int
	UiAmount       *float64
	UiAmountString string
}

type GRPCInnerInstructionSet struct {
	Index        int
	Instructions []GRPCInstruction
}

type GRPCLoadedAddresses struct {
	Writable [][]byte
	Readonly [][]byte
}

type GRPCReturnData struct {
	ProgramId []byte
	Data      []byte
}

// ConvertGRPCToSolanaTransaction converts gRPC format to SolanaTransaction
func ConvertGRPCToSolanaTransaction(grpc *GRPCTransaction, slot uint64, blockTime int64) *adapter.SolanaTransaction {
	// Convert signatures
	signatures := make([]string, len(grpc.Transaction.Signatures))
	for i, sig := range grpc.Transaction.Signatures {
		signatures[i] = base64.StdEncoding.EncodeToString(sig)
	}

	// Convert account keys
	accountKeys := make([]adapter.AccountKey, len(grpc.Transaction.Message.AccountKeys))
	for i, key := range grpc.Transaction.Message.AccountKeys {
		// In real implementation, you'd use base58 encoding
		accountKeys[i] = adapter.AccountKey{
			Pubkey: hex.EncodeToString(key),
		}
	}

	// Convert instructions
	instructions := make([]interface{}, len(grpc.Transaction.Message.Instructions))
	for i, inst := range grpc.Transaction.Message.Instructions {
		accounts := make([]int, len(inst.Accounts))
		for j, acc := range inst.Accounts {
			accounts[j] = int(acc)
		}
		instructions[i] = map[string]interface{}{
			"programIdIndex": inst.ProgramIdIndex,
			"accounts":       accounts,
			"data":           base64.StdEncoding.EncodeToString(inst.Data),
		}
	}

	// Convert inner instructions
	var innerInstructions []adapter.InnerInstructionSet
	for _, innerSet := range grpc.Meta.InnerInstructions {
		innerInsts := make([]interface{}, len(innerSet.Instructions))
		for j, inst := range innerSet.Instructions {
			accounts := make([]int, len(inst.Accounts))
			for k, acc := range inst.Accounts {
				accounts[k] = int(acc)
			}
			innerInsts[j] = map[string]interface{}{
				"programIdIndex": inst.ProgramIdIndex,
				"accounts":       accounts,
				"data":           base64.StdEncoding.EncodeToString(inst.Data),
			}
		}
		innerInstructions = append(innerInstructions, adapter.InnerInstructionSet{
			Index:        innerSet.Index,
			Instructions: innerInsts,
		})
	}

	// Convert token balances
	preTokenBalances := make([]adapter.TokenBalance, len(grpc.Meta.PreTokenBalances))
	for i, bal := range grpc.Meta.PreTokenBalances {
		preTokenBalances[i] = adapter.TokenBalance{
			AccountIndex: bal.AccountIndex,
			Mint:         bal.Mint,
			Owner:        bal.Owner,
			UiTokenAmount: types.TokenAmount{
				Amount:   bal.UiTokenAmount.Amount,
				Decimals: uint8(bal.UiTokenAmount.Decimals),
				UIAmount: bal.UiTokenAmount.UiAmount,
			},
		}
	}

	postTokenBalances := make([]adapter.TokenBalance, len(grpc.Meta.PostTokenBalances))
	for i, bal := range grpc.Meta.PostTokenBalances {
		postTokenBalances[i] = adapter.TokenBalance{
			AccountIndex: bal.AccountIndex,
			Mint:         bal.Mint,
			Owner:        bal.Owner,
			UiTokenAmount: types.TokenAmount{
				Amount:   bal.UiTokenAmount.Amount,
				Decimals: uint8(bal.UiTokenAmount.Decimals),
				UIAmount: bal.UiTokenAmount.UiAmount,
			},
		}
	}

	// Convert loaded addresses
	var loadedAddresses *adapter.LoadedAddresses
	if len(grpc.Meta.LoadedAddresses.Writable) > 0 || len(grpc.Meta.LoadedAddresses.Readonly) > 0 {
		writable := make([]string, len(grpc.Meta.LoadedAddresses.Writable))
		for i, addr := range grpc.Meta.LoadedAddresses.Writable {
			writable[i] = hex.EncodeToString(addr)
		}
		readonly := make([]string, len(grpc.Meta.LoadedAddresses.Readonly))
		for i, addr := range grpc.Meta.LoadedAddresses.Readonly {
			readonly[i] = hex.EncodeToString(addr)
		}
		loadedAddresses = &adapter.LoadedAddresses{
			Writable: writable,
			Readonly: readonly,
		}
	}

	computeUnits := grpc.Meta.ComputeUnitsConsumed

	return &adapter.SolanaTransaction{
		Transaction: adapter.TransactionData{
			Signatures: signatures,
			Message: adapter.TransactionMessage{
				Header: &adapter.MessageHeader{
					NumRequiredSignatures:       grpc.Transaction.Message.Header.NumRequiredSignatures,
					NumReadonlySignedAccounts:   grpc.Transaction.Message.Header.NumReadonlySignedAccounts,
					NumReadonlyUnsignedAccounts: grpc.Transaction.Message.Header.NumReadonlyUnsignedAccounts,
				},
				AccountKeys:  accountKeys,
				Instructions: instructions,
			},
		},
		Meta: &adapter.TransactionMeta{
			Fee:                  grpc.Meta.Fee,
			PreBalances:          grpc.Meta.PreBalances,
			PostBalances:         grpc.Meta.PostBalances,
			PreTokenBalances:     preTokenBalances,
			PostTokenBalances:    postTokenBalances,
			InnerInstructions:    innerInstructions,
			LogMessages:          grpc.Meta.LogMessages,
			LoadedAddresses:      loadedAddresses,
			ComputeUnitsConsumed: &computeUnits,
		},
		Slot:      slot,
		BlockTime: &blockTime,
	}
}

func TestGRPCConversion(t *testing.T) {
	// Test conversion of gRPC format to SolanaTransaction
	grpcTx := &GRPCTransaction{
		Signature: []byte{1, 2, 3, 4, 5, 6, 7, 8},
		IsVote:    false,
		Transaction: GRPCTransactionData{
			Signatures: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8}},
			Message: GRPCMessageData{
				Header: GRPCHeader{
					NumRequiredSignatures:       1,
					NumReadonlySignedAccounts:   0,
					NumReadonlyUnsignedAccounts: 2,
				},
				AccountKeys: [][]byte{
					{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					{6, 166, 68, 89, 133, 141, 127, 214, 75, 172, 145, 254, 38, 110, 10, 79, 90, 115, 117, 180, 203, 46, 23, 97, 47, 148, 59, 14, 35, 75, 246, 56},
				},
				RecentBlockhash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				Instructions: []GRPCInstruction{
					{
						ProgramIdIndex: 1,
						Accounts:       []byte{0},
						Data:           []byte{1, 2, 3, 4},
					},
				},
			},
		},
		Meta: GRPCTransactionMeta{
			Fee:          5000,
			PreBalances:  []uint64{1000000000, 0},
			PostBalances: []uint64{999995000, 0},
		},
	}

	tx := ConvertGRPCToSolanaTransaction(grpcTx, 12345, 1699999999)

	if tx == nil {
		t.Fatal("Conversion returned nil")
	}

	if len(tx.Transaction.Signatures) != 1 {
		t.Errorf("Expected 1 signature, got %d", len(tx.Transaction.Signatures))
	}

	if tx.Slot != 12345 {
		t.Errorf("Expected slot 12345, got %d", tx.Slot)
	}

	if tx.Meta.Fee != 5000 {
		t.Errorf("Expected fee 5000, got %d", tx.Meta.Fee)
	}
}

func TestParseGRPCTransaction(t *testing.T) {
	grpcTx := &GRPCTransaction{
		Signature: []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Transaction: GRPCTransactionData{
			Signatures: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8}},
			Message: GRPCMessageData{
				Header: GRPCHeader{
					NumRequiredSignatures: 1,
				},
				AccountKeys: [][]byte{
					make([]byte, 32), // signer
					make([]byte, 32), // token program
				},
				Instructions: []GRPCInstruction{
					{
						ProgramIdIndex: 1,
						Accounts:       []byte{0},
						Data:           []byte{1, 2, 3, 4},
					},
				},
			},
		},
		Meta: GRPCTransactionMeta{
			Fee:          5000,
			PreBalances:  []uint64{1000000000},
			PostBalances: []uint64{999995000},
		},
	}

	tx := ConvertGRPCToSolanaTransaction(grpcTx, 12345, 1699999999)

	parser := dexparser.NewDexParser()
	config := &types.ParseConfig{
		TryUnknownDEX: true,
	}

	result := parser.ParseAll(tx, config)

	if result == nil {
		t.Fatal("ParseAll returned nil")
	}

	if !result.State {
		t.Logf("Parse message: %s", result.Msg)
	}

	t.Logf("Signature: %s", result.Signature)
	t.Logf("Trades: %d", len(result.Trades))
	t.Logf("Liquidities: %d", len(result.Liquidities))
}

func TestShredParseGRPCTransaction(t *testing.T) {
	grpcTx := &GRPCTransaction{
		Signature: []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Transaction: GRPCTransactionData{
			Signatures: [][]byte{{1, 2, 3, 4, 5, 6, 7, 8}},
			Message: GRPCMessageData{
				Header: GRPCHeader{
					NumRequiredSignatures: 1,
				},
				AccountKeys: [][]byte{
					make([]byte, 32), // signer
					make([]byte, 32), // program
				},
				Instructions: []GRPCInstruction{
					{
						ProgramIdIndex: 1,
						Accounts:       []byte{0},
						Data:           []byte{1, 2, 3, 4},
					},
				},
			},
		},
		Meta: GRPCTransactionMeta{
			Fee: 5000,
		},
	}

	tx := ConvertGRPCToSolanaTransaction(grpcTx, 12345, 1699999999)

	parser := dexparser.NewShredParser()
	config := &types.ParseConfig{
		TryUnknownDEX: true,
	}

	result := parser.ParseAll(tx, config)

	if result == nil {
		t.Fatal("ParseAll returned nil")
	}

	if !result.State {
		t.Logf("Parse message: %s", result.Msg)
	}

	t.Logf("Signature: %s", result.Signature)
	t.Logf("Instructions: %d programs", len(result.Instructions))
}
