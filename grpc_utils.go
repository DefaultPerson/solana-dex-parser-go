package dexparser

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// YellowstoneTransaction represents a transaction from Yellowstone gRPC (Helius Laserstream/Triton)
type YellowstoneTransaction struct {
	Signature   []byte
	IsVote      bool
	Transaction YellowstoneTransactionData
	Meta        YellowstoneTransactionMeta
}

// YellowstoneTransactionData represents the transaction data from gRPC
type YellowstoneTransactionData struct {
	Signatures [][]byte
	Message    YellowstoneMessageData
}

// YellowstoneMessageData represents the message data from gRPC
type YellowstoneMessageData struct {
	Header              YellowstoneHeader
	AccountKeys         [][]byte
	RecentBlockhash     []byte
	Instructions        []YellowstoneInstruction
	Versioned           bool
	AddressTableLookups []YellowstoneAddressTableLookup
}

// YellowstoneHeader represents the message header from gRPC
type YellowstoneHeader struct {
	NumRequiredSignatures       int
	NumReadonlySignedAccounts   int
	NumReadonlyUnsignedAccounts int
}

// YellowstoneInstruction represents an instruction from gRPC
type YellowstoneInstruction struct {
	ProgramIdIndex int
	Accounts       []byte
	Data           []byte
}

// YellowstoneAddressTableLookup represents address table lookup from gRPC
type YellowstoneAddressTableLookup struct {
	AccountKey      []byte
	WritableIndexes []byte
	ReadonlyIndexes []byte
}

// YellowstoneTransactionMeta represents the transaction metadata from gRPC
type YellowstoneTransactionMeta struct {
	Err                  interface{}
	Fee                  uint64
	PreBalances          []uint64
	PostBalances         []uint64
	PreTokenBalances     []YellowstoneTokenBalance
	PostTokenBalances    []YellowstoneTokenBalance
	InnerInstructions    []YellowstoneInnerInstructionSet
	LogMessages          []string
	LoadedAddresses      YellowstoneLoadedAddresses
	ReturnData           *YellowstoneReturnData
	ComputeUnitsConsumed uint64
}

// YellowstoneTokenBalance represents token balance from gRPC
type YellowstoneTokenBalance struct {
	AccountIndex  int
	Mint          string
	Owner         string
	UiTokenAmount YellowstoneTokenAmount
}

// YellowstoneTokenAmount represents token amount from gRPC
type YellowstoneTokenAmount struct {
	Amount         string
	Decimals       int
	UiAmount       *float64
	UiAmountString string
}

// YellowstoneInnerInstructionSet represents inner instructions from gRPC
type YellowstoneInnerInstructionSet struct {
	Index        int
	Instructions []YellowstoneInstruction
}

// YellowstoneLoadedAddresses represents loaded addresses from gRPC
type YellowstoneLoadedAddresses struct {
	Writable [][]byte
	Readonly [][]byte
}

// YellowstoneReturnData represents return data from gRPC
type YellowstoneReturnData struct {
	ProgramId []byte
	Data      []byte
}

// ConvertYellowstoneTransaction converts a Yellowstone gRPC transaction to SolanaTransaction format
// that can be used with DexParser.ParseAll() or ShredParser.ParseAll()
func ConvertYellowstoneTransaction(grpc *YellowstoneTransaction, slot uint64, blockTime int64) *adapter.SolanaTransaction {
	if grpc == nil {
		return nil
	}

	// Convert signatures
	signatures := make([]string, len(grpc.Transaction.Signatures))
	for i, sig := range grpc.Transaction.Signatures {
		signatures[i] = base64.StdEncoding.EncodeToString(sig)
	}

	// Convert account keys
	accountKeys := make([]adapter.AccountKey, len(grpc.Transaction.Message.AccountKeys))
	for i, key := range grpc.Transaction.Message.AccountKeys {
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
