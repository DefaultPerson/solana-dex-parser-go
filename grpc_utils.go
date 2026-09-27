package dexparser

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Yellowstone gRPC (geyser.proto of github.com/rpcpool/yellowstone-grpc, also
// served by Helius LaserStream and Triton) delivers executed transactions as
// SubscribeUpdate.GetTransaction() -> SubscribeUpdateTransaction{Transaction
// *SubscribeUpdateTransactionInfo, Slot uint64}. Transaction updates carry no
// block time.
//
// The generated Go types (github.com/rpcpool/yellowstone-grpc/examples/golang/proto,
// or github.com/helius-labs/laserstream-sdk/go/proto, which also has the v1
// Message.Config) require a newer Go than this module, so it does not import
// them. Pass the generated *SubscribeUpdateTransactionInfo to
// ConvertGeyserTransaction, or fill YellowstoneTransaction yourself:
//
//	SubscribeUpdateTransactionInfo     YellowstoneTransaction
//	  Signature, IsVote                  Signature, IsVote
//	  Transaction.Signatures             Transaction.Signatures
//	  Transaction.Message.Header         Transaction.Message.Header (uint32 -> int)
//	  Message.AccountKeys                Transaction.Message.AccountKeys (static keys)
//	  Message.RecentBlockhash            Transaction.Message.RecentBlockhash
//	  Message.Instructions[i]            Transaction.Message.Instructions[i]
//	    ProgramIdIndex, Accounts, Data     ProgramIdIndex (uint32 -> int), Accounts, Data (raw bytes)
//	  Message.Versioned                  Transaction.Message.Versioned
//	  Message.AddressTableLookups        Transaction.Message.AddressTableLookups
//	  Message.Config (v1, LaserStream)   Transaction.Message.Config
//	  Meta.Err (*TransactionError)       Meta.Err (nil = success, any other value = failed)
//	  Meta.Fee, PreBalances, PostBalances, LogMessages
//	  Meta.InnerInstructions[i]          Meta.InnerInstructions[i] (Index uint32 -> int)
//	    Instructions[j] (+StackHeight)     Instructions[j] (+StackHeight)
//	  Meta.Pre/PostTokenBalances[i]      Meta.Pre/PostTokenBalances[i]
//	    AccountIndex, Mint, Owner, ProgramId, UiTokenAmount{Amount, Decimals, UiAmount, UiAmountString}
//	  Meta.LoadedWritableAddresses       Meta.LoadedWritableAddresses
//	  Meta.LoadedReadonlyAddresses       Meta.LoadedReadonlyAddresses
//	  Meta.ComputeUnitsConsumed (*uint64) Meta.ComputeUnitsConsumed
//
// A Yellowstone transaction is version 1 when Message.Config is set, 0 when
// Message.Versioned is set, legacy otherwise.

// YellowstoneTransaction represents a transaction from Yellowstone gRPC
// (SubscribeUpdateTransactionInfo; Helius LaserStream, Triton). Keys,
// signatures and instruction data are raw bytes as in the protobuf message.
type YellowstoneTransaction struct {
	Signature   []byte
	IsVote      bool
	Transaction YellowstoneTransactionData
	Meta        YellowstoneTransactionMeta
	// NoMeta marks an update without TransactionStatusMeta (pre-execution
	// data such as deshred updates): the converted transaction has no meta,
	// as ShredParser expects for pre-execution input
	NoMeta bool
	// LoadedWritableAddresses and LoadedReadonlyAddresses are the lookup
	// table addresses of an update without meta (deshred updates carry them
	// at the top level)
	LoadedWritableAddresses [][]byte
	LoadedReadonlyAddresses [][]byte
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
	// Config is the message-level compute budget of a version 1 transaction
	// (nil for legacy and v0 transactions)
	Config *YellowstoneTransactionConfig
}

// YellowstoneTransactionConfig is the v1 transaction config (SIMD-0385)
type YellowstoneTransactionConfig struct {
	PriorityFee                 *uint64
	ComputeUnitLimit            *uint32
	LoadedAccountsDataSizeLimit *uint32
	HeapSize                    *uint32
}

// YellowstoneHeader represents the message header from gRPC
type YellowstoneHeader struct {
	NumRequiredSignatures       int
	NumReadonlySignedAccounts   int
	NumReadonlyUnsignedAccounts int
}

// YellowstoneInstruction represents an instruction from gRPC: account indexes
// one per byte, raw data
type YellowstoneInstruction struct {
	ProgramIdIndex int
	Accounts       []byte
	Data           []byte
	// StackHeight is the invocation depth of an inner instruction (optional)
	StackHeight *uint32
}

// YellowstoneAddressTableLookup represents address table lookup from gRPC
type YellowstoneAddressTableLookup struct {
	AccountKey      []byte
	WritableIndexes []byte
	ReadonlyIndexes []byte
}

// YellowstoneTransactionMeta represents the transaction metadata from gRPC
type YellowstoneTransactionMeta struct {
	// Err is nil for a successful transaction; any other value (for example
	// the protobuf *TransactionError) marks it failed
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
	// LoadedWritableAddresses and LoadedReadonlyAddresses are the protobuf's
	// flat lookup table address lists; they take precedence over
	// LoadedAddresses when set
	LoadedWritableAddresses [][]byte
	LoadedReadonlyAddresses [][]byte
}

// YellowstoneTokenBalance represents token balance from gRPC
type YellowstoneTokenBalance struct {
	AccountIndex  int
	Mint          string
	Owner         string
	UiTokenAmount YellowstoneTokenAmount
	// ProgramId is the token program owning the account
	ProgramId string
}

// YellowstoneTokenAmount represents token amount from gRPC. Amount (raw) is
// authoritative; the protobuf UiAmount is 0 where RPC reports null.
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

// ConvertYellowstoneTransaction converts a Yellowstone gRPC transaction to
// the SolanaTransaction form of the JSON-RPC "json" encoding (base58 keys and
// signatures), usable with DexParser.ParseAll() or ShredParser.ParseAll().
// slot is SubscribeUpdateTransaction.Slot; transaction updates have no block
// time, pass 0 when unknown.
func ConvertYellowstoneTransaction(grpc *YellowstoneTransaction, slot uint64, blockTime int64) *adapter.SolanaTransaction {
	if grpc == nil {
		return nil
	}

	msg := grpc.Transaction.Message

	signatures := make([]string, len(grpc.Transaction.Signatures))
	for i, sig := range grpc.Transaction.Signatures {
		signatures[i] = base58.Encode(sig)
	}
	if len(signatures) == 0 && len(grpc.Signature) > 0 {
		signatures = []string{base58.Encode(grpc.Signature)}
	}

	accountKeys := make([]adapter.AccountKey, len(msg.AccountKeys))
	for i, key := range msg.AccountKeys {
		accountKeys[i] = adapter.AccountKey{Pubkey: base58.Encode(key)}
	}

	instructions := make([]interface{}, len(msg.Instructions))
	for i, inst := range msg.Instructions {
		instructions[i] = yellowstoneInstructionMap(inst)
	}

	var lookups []adapter.AddressTableLookup
	for _, l := range msg.AddressTableLookups {
		lookups = append(lookups, adapter.AddressTableLookup{
			AccountKey:      base58.Encode(l.AccountKey),
			WritableIndexes: byteIndexes(l.WritableIndexes),
			ReadonlyIndexes: byteIndexes(l.ReadonlyIndexes),
		})
	}

	tx := &adapter.SolanaTransaction{
		Transaction: adapter.TransactionData{
			Signatures: signatures,
			Message: adapter.TransactionMessage{
				Header: &adapter.MessageHeader{
					NumRequiredSignatures:       msg.Header.NumRequiredSignatures,
					NumReadonlySignedAccounts:   msg.Header.NumReadonlySignedAccounts,
					NumReadonlyUnsignedAccounts: msg.Header.NumReadonlyUnsignedAccounts,
				},
				AccountKeys:         accountKeys,
				Instructions:        instructions,
				AddressTableLookups: lookups,
			},
		},
		Slot:      slot,
		BlockTime: &blockTime,
	}

	switch {
	case msg.Config != nil:
		tx.Version = 1
		tx.Transaction.Message.TransactionConfig = &adapter.TransactionConfig{
			PriorityFee:                 msg.Config.PriorityFee,
			ComputeUnitLimit:            widen(msg.Config.ComputeUnitLimit),
			LoadedAccountsDataSizeLimit: widen(msg.Config.LoadedAccountsDataSizeLimit),
			HeapSize:                    widen(msg.Config.HeapSize),
		}
	case msg.Versioned:
		tx.Version = 0
	default:
		tx.Version = "legacy"
	}

	if grpc.NoMeta {
		// Without meta the adapter reads lookup table addresses from the
		// message (the jsonParsed convention) when the update provides them
		for _, addr := range grpc.LoadedWritableAddresses {
			tx.Transaction.Message.AccountKeys = append(tx.Transaction.Message.AccountKeys, adapter.AccountKey{Pubkey: base58.Encode(addr), Writable: true, Source: "lookupTable"})
		}
		for _, addr := range grpc.LoadedReadonlyAddresses {
			tx.Transaction.Message.AccountKeys = append(tx.Transaction.Message.AccountKeys, adapter.AccountKey{Pubkey: base58.Encode(addr), Source: "lookupTable"})
		}
		return tx
	}

	meta := grpc.Meta
	innerInstructions := make([]adapter.InnerInstructionSet, 0, len(meta.InnerInstructions))
	for _, innerSet := range meta.InnerInstructions {
		innerInsts := make([]interface{}, len(innerSet.Instructions))
		for j, inst := range innerSet.Instructions {
			innerInsts[j] = yellowstoneInstructionMap(inst)
		}
		innerInstructions = append(innerInstructions, adapter.InnerInstructionSet{
			Index:        innerSet.Index,
			Instructions: innerInsts,
		})
	}

	writable, readonly := meta.LoadedWritableAddresses, meta.LoadedReadonlyAddresses
	if len(writable) == 0 && len(readonly) == 0 {
		writable, readonly = meta.LoadedAddresses.Writable, meta.LoadedAddresses.Readonly
	}
	computeUnits := meta.ComputeUnitsConsumed

	tx.Meta = &adapter.TransactionMeta{
		Err:                  yellowstoneError(meta.Err),
		Fee:                  meta.Fee,
		PreBalances:          meta.PreBalances,
		PostBalances:         meta.PostBalances,
		PreTokenBalances:     yellowstoneTokenBalances(meta.PreTokenBalances),
		PostTokenBalances:    yellowstoneTokenBalances(meta.PostTokenBalances),
		InnerInstructions:    innerInstructions,
		LogMessages:          meta.LogMessages,
		LoadedAddresses:      &adapter.LoadedAddresses{Writable: base58List(writable), Readonly: base58List(readonly)},
		ComputeUnitsConsumed: &computeUnits,
	}
	return tx
}

// yellowstoneInstructionMap builds an instruction in the form of the "json"
// encoding, with index and data bytes the adapter decodes directly
func yellowstoneInstructionMap(inst YellowstoneInstruction) map[string]interface{} {
	data := inst.Data
	if data == nil {
		data = []byte{}
	}
	ix := map[string]interface{}{
		"programIdIndex": inst.ProgramIdIndex,
		"accounts":       byteIndexes(inst.Accounts),
		"data":           data,
	}
	if inst.StackHeight != nil {
		ix["stackHeight"] = int(*inst.StackHeight)
	}
	return ix
}

// yellowstoneError returns nil for a nil error, including typed nil pointers
// such as a nil *TransactionError stored in the interface
func yellowstoneError(err interface{}) interface{} {
	if err == nil {
		return nil
	}
	v := reflect.ValueOf(err)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface:
		if v.IsNil() {
			return nil
		}
	}
	return err
}

func yellowstoneTokenBalances(balances []YellowstoneTokenBalance) []adapter.TokenBalance {
	out := make([]adapter.TokenBalance, len(balances))
	for i, bal := range balances {
		out[i] = adapter.TokenBalance{
			AccountIndex: bal.AccountIndex,
			Mint:         bal.Mint,
			Owner:        bal.Owner,
			ProgramId:    bal.ProgramId,
			UiTokenAmount: types.TokenAmount{
				Amount:   bal.UiTokenAmount.Amount,
				Decimals: uint8(bal.UiTokenAmount.Decimals),
				UIAmount: bal.UiTokenAmount.UiAmount,
			},
		}
	}
	return out
}

func byteIndexes(b []byte) []int {
	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}
	return out
}

func base58List(keys [][]byte) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = base58.Encode(k)
	}
	return out
}

func widen(v *uint32) *uint64 {
	if v == nil {
		return nil
	}
	w := uint64(*v)
	return &w
}

// ConvertGeyserTransaction converts a transaction of the generated Yellowstone
// protobuf types to SolanaTransaction: a *SubscribeUpdateTransactionInfo
// (executed transaction, with meta) or a deshred transaction info (no meta,
// lookup table addresses at the top level). It reads the generated Go field
// names, so the rpcpool and Helius LaserStream packages both work without
// this module importing them.
func ConvertGeyserTransaction(info interface{}, slot uint64, blockTime int64) (*adapter.SolanaTransaction, error) {
	yt, err := YellowstoneFromGeyser(info)
	if err != nil {
		return nil, err
	}
	return ConvertYellowstoneTransaction(yt, slot, blockTime), nil
}

// YellowstoneFromGeyser copies a generated Yellowstone protobuf transaction
// info (see ConvertGeyserTransaction) into YellowstoneTransaction
func YellowstoneFromGeyser(info interface{}) (*YellowstoneTransaction, error) {
	root := derefValue(reflect.ValueOf(info))
	if !root.IsValid() || root.Kind() != reflect.Struct {
		return nil, errors.New("geyser transaction: expected a (pointer to a) SubscribeUpdateTransactionInfo struct")
	}
	txv := structField(root, "Transaction")
	if !txv.IsValid() {
		return nil, errors.New("geyser transaction: no Transaction")
	}
	msg := structField(txv, "Message")
	if !msg.IsValid() {
		return nil, errors.New("geyser transaction: no Transaction.Message")
	}

	yt := &YellowstoneTransaction{
		Signature: bytesField(root, "Signature"),
		IsVote:    boolField(root, "IsVote"),
	}
	yt.Transaction.Signatures = bytesListField(txv, "Signatures")

	header := structField(msg, "Header")
	yt.Transaction.Message.Header = YellowstoneHeader{
		NumRequiredSignatures:       int(uintField(header, "NumRequiredSignatures")),
		NumReadonlySignedAccounts:   int(uintField(header, "NumReadonlySignedAccounts")),
		NumReadonlyUnsignedAccounts: int(uintField(header, "NumReadonlyUnsignedAccounts")),
	}
	yt.Transaction.Message.AccountKeys = bytesListField(msg, "AccountKeys")
	yt.Transaction.Message.RecentBlockhash = bytesField(msg, "RecentBlockhash")
	yt.Transaction.Message.Versioned = boolField(msg, "Versioned")
	forEachElem(structField(msg, "Instructions"), func(ix reflect.Value) {
		yt.Transaction.Message.Instructions = append(yt.Transaction.Message.Instructions, geyserInstruction(ix))
	})
	forEachElem(structField(msg, "AddressTableLookups"), func(l reflect.Value) {
		yt.Transaction.Message.AddressTableLookups = append(yt.Transaction.Message.AddressTableLookups, YellowstoneAddressTableLookup{
			AccountKey:      bytesField(l, "AccountKey"),
			WritableIndexes: bytesField(l, "WritableIndexes"),
			ReadonlyIndexes: bytesField(l, "ReadonlyIndexes"),
		})
	})
	if cfg := structField(msg, "Config"); cfg.IsValid() {
		yt.Transaction.Message.Config = &YellowstoneTransactionConfig{
			PriorityFee:                 optUint64Field(cfg, "PriorityFee"),
			ComputeUnitLimit:            optUint32Field(cfg, "ComputeUnitLimit"),
			LoadedAccountsDataSizeLimit: optUint32Field(cfg, "LoadedAccountsDataSizeLimit"),
			HeapSize:                    optUint32Field(cfg, "HeapSize"),
		}
	}

	meta := structField(root, "Meta")
	if !meta.IsValid() {
		yt.NoMeta = true
		yt.LoadedWritableAddresses = bytesListField(root, "LoadedWritableAddresses")
		yt.LoadedReadonlyAddresses = bytesListField(root, "LoadedReadonlyAddresses")
		return yt, nil
	}

	if errv := structField(meta, "Err"); errv.IsValid() {
		yt.Meta.Err = geyserError{Err: bytesField(errv, "Err")}
	}
	yt.Meta.Fee = uintField(meta, "Fee")
	yt.Meta.PreBalances = uint64ListField(meta, "PreBalances")
	yt.Meta.PostBalances = uint64ListField(meta, "PostBalances")
	if v := structField(meta, "LogMessages"); v.IsValid() {
		if logs, ok := v.Interface().([]string); ok {
			yt.Meta.LogMessages = logs
		}
	}
	forEachElem(structField(meta, "InnerInstructions"), func(set reflect.Value) {
		inner := YellowstoneInnerInstructionSet{Index: int(uintField(set, "Index"))}
		forEachElem(structField(set, "Instructions"), func(ix reflect.Value) {
			inner.Instructions = append(inner.Instructions, geyserInstruction(ix))
		})
		yt.Meta.InnerInstructions = append(yt.Meta.InnerInstructions, inner)
	})
	yt.Meta.PreTokenBalances = geyserTokenBalances(structField(meta, "PreTokenBalances"))
	yt.Meta.PostTokenBalances = geyserTokenBalances(structField(meta, "PostTokenBalances"))
	yt.Meta.LoadedWritableAddresses = bytesListField(meta, "LoadedWritableAddresses")
	yt.Meta.LoadedReadonlyAddresses = bytesListField(meta, "LoadedReadonlyAddresses")
	if cu := optUint64Field(meta, "ComputeUnitsConsumed"); cu != nil {
		yt.Meta.ComputeUnitsConsumed = *cu
	}
	return yt, nil
}

// geyserError is the TransactionError of a failed geyser transaction: the
// serialized Solana TransactionError
type geyserError struct {
	Err []byte `json:"err"`
}

func (e geyserError) Error() string {
	return fmt.Sprintf("transaction error %x", e.Err)
}

func geyserInstruction(ix reflect.Value) YellowstoneInstruction {
	return YellowstoneInstruction{
		ProgramIdIndex: int(uintField(ix, "ProgramIdIndex")),
		Accounts:       bytesField(ix, "Accounts"),
		Data:           bytesField(ix, "Data"),
		StackHeight:    optUint32Field(ix, "StackHeight"),
	}
}

func geyserTokenBalances(list reflect.Value) []YellowstoneTokenBalance {
	var out []YellowstoneTokenBalance
	forEachElem(list, func(b reflect.Value) {
		bal := YellowstoneTokenBalance{
			AccountIndex: int(uintField(b, "AccountIndex")),
			Mint:         stringField(b, "Mint"),
			Owner:        stringField(b, "Owner"),
			ProgramId:    stringField(b, "ProgramId"),
		}
		if amount := structField(b, "UiTokenAmount"); amount.IsValid() {
			bal.UiTokenAmount = YellowstoneTokenAmount{
				Amount:         stringField(amount, "Amount"),
				Decimals:       int(uintField(amount, "Decimals")),
				UiAmountString: stringField(amount, "UiAmountString"),
			}
			// RPC reports a zero balance as uiAmount null
			if v := structField(amount, "UiAmount"); v.IsValid() && v.Kind() == reflect.Float64 && bal.UiTokenAmount.Amount != "0" {
				f := v.Float()
				bal.UiTokenAmount.UiAmount = &f
			}
		}
		out = append(out, bal)
	})
	return out
}

// derefValue follows pointers and interfaces; it returns an invalid Value for nil
func derefValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// structField returns the named field of a struct value, dereferenced; an
// invalid Value when v is not a struct, the field is missing or nil
func structField(v reflect.Value, name string) reflect.Value {
	v = derefValue(v)
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	return derefValue(v.FieldByName(name))
}

func bytesField(v reflect.Value, name string) []byte {
	f := structField(v, name)
	if f.IsValid() && f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.Uint8 {
		return f.Bytes()
	}
	return nil
}

func bytesListField(v reflect.Value, name string) [][]byte {
	var out [][]byte
	forEachElem(structField(v, name), func(e reflect.Value) {
		if e.Kind() == reflect.Slice && e.Type().Elem().Kind() == reflect.Uint8 {
			out = append(out, e.Bytes())
		}
	})
	return out
}

func uint64ListField(v reflect.Value, name string) []uint64 {
	var out []uint64
	forEachElem(structField(v, name), func(e reflect.Value) {
		if n, ok := uintValue(e); ok {
			out = append(out, n)
		}
	})
	return out
}

func boolField(v reflect.Value, name string) bool {
	f := structField(v, name)
	return f.IsValid() && f.Kind() == reflect.Bool && f.Bool()
}

func stringField(v reflect.Value, name string) string {
	f := structField(v, name)
	if f.IsValid() && f.Kind() == reflect.String {
		return f.String()
	}
	return ""
}

func uintField(v reflect.Value, name string) uint64 {
	n, _ := uintValue(structField(v, name))
	return n
}

func optUint64Field(v reflect.Value, name string) *uint64 {
	if n, ok := uintValue(structField(v, name)); ok {
		return &n
	}
	return nil
}

func optUint32Field(v reflect.Value, name string) *uint32 {
	if n, ok := uintValue(structField(v, name)); ok {
		m := uint32(n)
		return &m
	}
	return nil
}

func uintValue(v reflect.Value) (uint64, bool) {
	if !v.IsValid() {
		return 0, false
	}
	switch v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() >= 0 {
			return uint64(v.Int()), true
		}
	}
	return 0, false
}

// forEachElem calls fn with each (dereferenced, non-nil) element of a slice
func forEachElem(v reflect.Value, fn func(reflect.Value)) {
	if !v.IsValid() || v.Kind() != reflect.Slice {
		return
	}
	for i := 0; i < v.Len(); i++ {
		if e := derefValue(v.Index(i)); e.IsValid() {
			fn(e)
		}
	}
}
