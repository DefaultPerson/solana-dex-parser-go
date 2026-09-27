package tests

import (
	"reflect"
	"testing"

	"github.com/goccy/go-json"
	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Yellowstone gRPC conversion. Each test converts real RPC ("json") fixtures
// into the byte form of the Yellowstone protobuf message (raw keys,
// signatures and instruction data, flat loaded addresses), converts them back
// with the library and requires the same parse results as the RPC form.
// shred-2, shred-19, shred-28, core-4, parity-6.

// grpcFixtures covers legacy, v0 (lookup tables) and v1 transactions, inner
// instructions with stack heights, token balances, failed transactions and
// the main shred programs
var grpcFixtures = []string{
	sigJupRouteV1,      // v0 Jupiter route, lookup tables
	sigJupRouteV2,      // v0 route_v2
	sigPumpBuyLegacy,   // legacy Pump.fun buy
	sigPswapSell,       // PumpSwap
	sigDBCSwap2Sell,    // Meteora DBC
	sigRayWithdraw,     // Raydium V4
	sigLcpInit2022,     // v1 transaction (message transactionConfig)
	sigPhotonPumpBuyV2, // Photon
	sigSystemTransfers, // System transfers
	sigPumpMigrate,     // Pump.fun migrate + PumpSwap create_pool
	"4b6jX4P8vqKUPJLLFWLxQ3gWH25qgy3uxQqQ98FW7tmUnk29PqWXjNwDeCb3tkP4w2MDtLWjqQYRK42abVnfz55m", // legacy PumpSwap buy (audit repro)
	"2nGCisM2rPEsCiYSJf9nF3gTT2TAPwE8HnTZdF5hTTU4HdQPgAga5rbWp15ex2gjhGHrUh5WQoFEhtS7e6tMzDaP", // failed Jupiter route
}

// toYellowstone builds the Yellowstone form of a "json" fixture
func toYellowstone(t *testing.T, tx *adapter.SolanaTransaction) *dexparser.YellowstoneTransaction {
	t.Helper()
	dec := func(s string) []byte {
		if s == "" {
			return []byte{}
		}
		b, err := base58.Decode(s)
		if err != nil {
			t.Fatalf("base58 %q: %v", s, err)
		}
		return b
	}
	instruction := func(v interface{}) dexparser.YellowstoneInstruction {
		m := v.(map[string]interface{})
		var accounts []byte
		for _, a := range m["accounts"].([]interface{}) {
			accounts = append(accounts, byte(jsonInt(a)))
		}
		ix := dexparser.YellowstoneInstruction{ProgramIdIndex: jsonInt(m["programIdIndex"]), Accounts: accounts, Data: dec(m["data"].(string))}
		if h, ok := m["stackHeight"]; ok && h != nil {
			sh := uint32(jsonInt(h))
			ix.StackHeight = &sh
		}
		return ix
	}

	yt := &dexparser.YellowstoneTransaction{}
	for _, s := range tx.Transaction.Signatures {
		yt.Transaction.Signatures = append(yt.Transaction.Signatures, dec(s))
	}
	yt.Signature = yt.Transaction.Signatures[0]
	msg := tx.Transaction.Message
	yt.Transaction.Message.Header = dexparser.YellowstoneHeader{
		NumRequiredSignatures:       msg.Header.NumRequiredSignatures,
		NumReadonlySignedAccounts:   msg.Header.NumReadonlySignedAccounts,
		NumReadonlyUnsignedAccounts: msg.Header.NumReadonlyUnsignedAccounts,
	}
	for _, k := range msg.AccountKeys {
		yt.Transaction.Message.AccountKeys = append(yt.Transaction.Message.AccountKeys, dec(k.Pubkey))
	}
	for _, ix := range msg.Instructions {
		yt.Transaction.Message.Instructions = append(yt.Transaction.Message.Instructions, instruction(ix))
	}
	yt.Transaction.Message.Versioned = tx.Version != "legacy"
	for _, l := range msg.AddressTableLookups {
		lookup := dexparser.YellowstoneAddressTableLookup{AccountKey: dec(l.AccountKey)}
		for _, i := range l.WritableIndexes {
			lookup.WritableIndexes = append(lookup.WritableIndexes, byte(i))
		}
		for _, i := range l.ReadonlyIndexes {
			lookup.ReadonlyIndexes = append(lookup.ReadonlyIndexes, byte(i))
		}
		yt.Transaction.Message.AddressTableLookups = append(yt.Transaction.Message.AddressTableLookups, lookup)
	}
	if cfg := msg.TransactionConfig; cfg != nil {
		narrow := func(v *uint64) *uint32 {
			if v == nil {
				return nil
			}
			n := uint32(*v)
			return &n
		}
		yt.Transaction.Message.Config = &dexparser.YellowstoneTransactionConfig{
			PriorityFee:                 cfg.PriorityFee,
			ComputeUnitLimit:            narrow(cfg.ComputeUnitLimit),
			LoadedAccountsDataSizeLimit: narrow(cfg.LoadedAccountsDataSizeLimit),
			HeapSize:                    narrow(cfg.HeapSize),
		}
	} else if jsonInt(tx.Version) == 1 {
		yt.Transaction.Message.Config = &dexparser.YellowstoneTransactionConfig{}
	}

	meta := tx.Meta
	if meta.Err != nil {
		// the protobuf carries the serialized TransactionError
		yt.Meta.Err = &struct{ Err []byte }{Err: []byte{8, 0, 0, 0}}
	}
	yt.Meta.Fee = meta.Fee
	yt.Meta.PreBalances = meta.PreBalances
	yt.Meta.PostBalances = meta.PostBalances
	yt.Meta.LogMessages = meta.LogMessages
	if meta.ComputeUnitsConsumed != nil {
		yt.Meta.ComputeUnitsConsumed = *meta.ComputeUnitsConsumed
	}
	balances := func(list []adapter.TokenBalance) []dexparser.YellowstoneTokenBalance {
		var out []dexparser.YellowstoneTokenBalance
		for _, b := range list {
			out = append(out, dexparser.YellowstoneTokenBalance{
				AccountIndex: b.AccountIndex, Mint: b.Mint, Owner: b.Owner, ProgramId: b.ProgramId,
				UiTokenAmount: dexparser.YellowstoneTokenAmount{Amount: b.UiTokenAmount.Amount, Decimals: int(b.UiTokenAmount.Decimals), UiAmount: b.UiTokenAmount.UIAmount},
			})
		}
		return out
	}
	yt.Meta.PreTokenBalances = balances(meta.PreTokenBalances)
	yt.Meta.PostTokenBalances = balances(meta.PostTokenBalances)
	for _, set := range meta.InnerInstructions {
		inner := dexparser.YellowstoneInnerInstructionSet{Index: set.Index}
		for _, ix := range set.Instructions {
			inner.Instructions = append(inner.Instructions, instruction(ix))
		}
		yt.Meta.InnerInstructions = append(yt.Meta.InnerInstructions, inner)
	}
	if la := meta.LoadedAddresses; la != nil {
		for _, a := range la.Writable {
			yt.Meta.LoadedWritableAddresses = append(yt.Meta.LoadedWritableAddresses, dec(a))
		}
		for _, a := range la.Readonly {
			yt.Meta.LoadedReadonlyAddresses = append(yt.Meta.LoadedReadonlyAddresses, dec(a))
		}
	}
	return yt
}

func jsonOf(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestGRPCRoundTrip: the converter emitted base64 signatures, hex keys,
// base64 data and []int accounts, so nothing parsed (0 trades, 0 programs).
func TestGRPCRoundTrip(t *testing.T) {
	cfg := &types.ParseConfig{ParseType: types.ParseAll(), TryUnknownDEX: true, IncludeFailedTxs: true}
	for _, sig := range grpcFixtures {
		t.Run(sig[:8], func(t *testing.T) {
			tx := loadFixture(t, sig)
			converted := dexparser.ConvertYellowstoneTransaction(toYellowstone(t, tx), tx.Slot, *tx.BlockTime)

			if converted.Transaction.Signatures[0] != sig {
				t.Fatalf("signature %s, want %s", converted.Transaction.Signatures[0], sig)
			}
			if got, want := jsonOf(t, rawAccountKeys(converted)), jsonOf(t, rawAccountKeys(tx)); got != want {
				t.Fatalf("account keys differ:\n got %s\nwant %s", got, want)
			}

			want := dexparser.NewDexParser().ParseAll(tx, cfg)
			got := dexparser.NewDexParser().ParseAll(converted, cfg)
			if jsonOf(t, got) != jsonOf(t, want) {
				t.Errorf("DexParser results differ:\n got %s\nwant %s", jsonOf(t, got), jsonOf(t, want))
			}
			if want.TxStatus != got.TxStatus || (tx.Meta.Err != nil) != (got.TxStatus == types.TransactionStatusFailed) {
				t.Errorf("TxStatus %s, want %s", got.TxStatus, want.TxStatus)
			}

			wantShred := dexparser.NewShredParser().ParseAll(tx, cfg)
			gotShred := dexparser.NewShredParser().ParseAll(converted, cfg)
			if jsonOf(t, gotShred) != jsonOf(t, wantShred) {
				t.Errorf("ShredParser results differ:\n got %s\nwant %s", jsonOf(t, gotShred), jsonOf(t, wantShred))
			}
			if len(wantShred.ParsedInstructions) == 0 {
				t.Errorf("fixture decodes nothing")
			}
		})
	}
}

// TestGRPCFailedTransaction: meta.err was dropped, so failed transactions
// were reported as successful trades. shred-19.
func TestGRPCFailedTransaction(t *testing.T) {
	tx := loadFixture(t, sigJupRouteV1)
	yt := toYellowstone(t, tx)
	yt.Meta.Err = &struct{ Err []byte }{Err: []byte{1}}
	converted := dexparser.ConvertYellowstoneTransaction(yt, tx.Slot, 0)
	res := dexparser.NewDexParser().ParseAll(converted, nil)
	if res.TxStatus != types.TransactionStatusFailed || len(res.Trades) != 0 {
		t.Errorf("failed tx: status %s trades %d, want failed and no trades", res.TxStatus, len(res.Trades))
	}
	shred := dexparser.NewShredParser().ParseAll(converted, nil)
	if shred.TxStatus != types.TransactionStatusFailed || len(shred.ParsedInstructions) != 0 || shred.Msg != "transaction failed" {
		t.Errorf("shred failed tx: status %s instructions %d msg %q", shred.TxStatus, len(shred.ParsedInstructions), shred.Msg)
	}

	// A typed nil error pointer means success
	yt.Meta.Err = (*struct{ Err []byte })(nil)
	if res := dexparser.NewDexParser().ParseAll(dexparser.ConvertYellowstoneTransaction(yt, tx.Slot, 0), nil); res.TxStatus != types.TransactionStatusSuccess {
		t.Errorf("nil *TransactionError: status %s, want success", res.TxStatus)
	}
}

// Mirrors of the generated Yellowstone Go types (field names and types of
// github.com/rpcpool/yellowstone-grpc/examples/golang/proto and the Helius
// LaserStream SDK; protobuf internals omitted), used to test
// ConvertGeyserTransaction without importing them.
type (
	pbSubscribeUpdateTransactionInfo struct {
		Signature   []byte
		IsVote      bool
		Transaction *pbTransaction
		Meta        *pbTransactionStatusMeta
		Index       uint64
	}
	pbSubscribeUpdateDeshredTransactionInfo struct {
		Signature               []byte
		IsVote                  bool
		Transaction             *pbTransaction
		LoadedWritableAddresses [][]byte
		LoadedReadonlyAddresses [][]byte
	}
	pbTransaction struct {
		Signatures [][]byte
		Message    *pbMessage
	}
	pbMessage struct {
		Header              *pbMessageHeader
		AccountKeys         [][]byte
		RecentBlockhash     []byte
		Instructions        []*pbCompiledInstruction
		Versioned           bool
		AddressTableLookups []*pbMessageAddressTableLookup
		Config              *pbTransactionConfig
	}
	pbTransactionConfig struct {
		PriorityFee                 *uint64
		ComputeUnitLimit            *uint32
		LoadedAccountsDataSizeLimit *uint32
		HeapSize                    *uint32
	}
	pbMessageHeader struct {
		NumRequiredSignatures, NumReadonlySignedAccounts, NumReadonlyUnsignedAccounts uint32
	}
	pbCompiledInstruction struct {
		ProgramIdIndex uint32
		Accounts       []byte
		Data           []byte
	}
	pbMessageAddressTableLookup struct {
		AccountKey, WritableIndexes, ReadonlyIndexes []byte
	}
	pbTransactionStatusMeta struct {
		Err                     *pbTransactionError
		Fee                     uint64
		PreBalances             []uint64
		PostBalances            []uint64
		InnerInstructions       []*pbInnerInstructions
		InnerInstructionsNone   bool
		LogMessages             []string
		LogMessagesNone         bool
		PreTokenBalances        []*pbTokenBalance
		PostTokenBalances       []*pbTokenBalance
		LoadedWritableAddresses [][]byte
		LoadedReadonlyAddresses [][]byte
		ReturnDataNone          bool
		ComputeUnitsConsumed    *uint64
		CostUnits               *uint64
	}
	pbTransactionError  struct{ Err []byte }
	pbInnerInstructions struct {
		Index        uint32
		Instructions []*pbInnerInstruction
	}
	pbInnerInstruction struct {
		ProgramIdIndex uint32
		Accounts       []byte
		Data           []byte
		StackHeight    *uint32
	}
	pbTokenBalance struct {
		AccountIndex  uint32
		Mint          string
		UiTokenAmount *pbUiTokenAmount
		Owner         string
		ProgramId     string
	}
	pbUiTokenAmount struct {
		UiAmount       float64
		Decimals       uint32
		Amount         string
		UiAmountString string
	}
)

// toGeyserProto builds the generated-type form of a Yellowstone transaction
func toGeyserProto(yt *dexparser.YellowstoneTransaction) *pbSubscribeUpdateTransactionInfo {
	m := yt.Transaction.Message
	msg := &pbMessage{
		Header:          &pbMessageHeader{uint32(m.Header.NumRequiredSignatures), uint32(m.Header.NumReadonlySignedAccounts), uint32(m.Header.NumReadonlyUnsignedAccounts)},
		AccountKeys:     m.AccountKeys,
		RecentBlockhash: make([]byte, 32),
		Versioned:       m.Versioned,
	}
	for _, ix := range m.Instructions {
		msg.Instructions = append(msg.Instructions, &pbCompiledInstruction{uint32(ix.ProgramIdIndex), ix.Accounts, ix.Data})
	}
	for _, l := range m.AddressTableLookups {
		msg.AddressTableLookups = append(msg.AddressTableLookups, &pbMessageAddressTableLookup{l.AccountKey, l.WritableIndexes, l.ReadonlyIndexes})
	}
	if c := m.Config; c != nil {
		msg.Config = &pbTransactionConfig{c.PriorityFee, c.ComputeUnitLimit, c.LoadedAccountsDataSizeLimit, c.HeapSize}
	}
	meta := &pbTransactionStatusMeta{
		Fee: yt.Meta.Fee, PreBalances: yt.Meta.PreBalances, PostBalances: yt.Meta.PostBalances, LogMessages: yt.Meta.LogMessages,
		LoadedWritableAddresses: yt.Meta.LoadedWritableAddresses, LoadedReadonlyAddresses: yt.Meta.LoadedReadonlyAddresses,
	}
	if yt.Meta.Err != nil && !reflect.ValueOf(yt.Meta.Err).IsNil() {
		meta.Err = &pbTransactionError{Err: []byte{1}}
	}
	cu := yt.Meta.ComputeUnitsConsumed
	meta.ComputeUnitsConsumed = &cu
	for _, set := range yt.Meta.InnerInstructions {
		inner := &pbInnerInstructions{Index: uint32(set.Index)}
		for _, ix := range set.Instructions {
			inner.Instructions = append(inner.Instructions, &pbInnerInstruction{uint32(ix.ProgramIdIndex), ix.Accounts, ix.Data, ix.StackHeight})
		}
		meta.InnerInstructions = append(meta.InnerInstructions, inner)
	}
	balances := func(list []dexparser.YellowstoneTokenBalance) []*pbTokenBalance {
		var out []*pbTokenBalance
		for _, b := range list {
			amount := &pbUiTokenAmount{Decimals: uint32(b.UiTokenAmount.Decimals), Amount: b.UiTokenAmount.Amount, UiAmountString: b.UiTokenAmount.UiAmountString}
			if b.UiTokenAmount.UiAmount != nil {
				amount.UiAmount = *b.UiTokenAmount.UiAmount
			}
			out = append(out, &pbTokenBalance{uint32(b.AccountIndex), b.Mint, amount, b.Owner, b.ProgramId})
		}
		return out
	}
	meta.PreTokenBalances = balances(yt.Meta.PreTokenBalances)
	meta.PostTokenBalances = balances(yt.Meta.PostTokenBalances)
	return &pbSubscribeUpdateTransactionInfo{Signature: yt.Signature, Transaction: &pbTransaction{Signatures: yt.Transaction.Signatures, Message: msg}, Meta: meta}
}

// TestGRPCGeyserTypes: the helper did not accept the generated
// SubscribeUpdateTransactionInfo (the docs example did not compile);
// ConvertGeyserTransaction reads it by field name. shred-19.
func TestGRPCGeyserTypes(t *testing.T) {
	cfg := &types.ParseConfig{ParseType: types.ParseAll(), TryUnknownDEX: true, IncludeFailedTxs: true}
	for _, sig := range grpcFixtures {
		tx := loadFixture(t, sig)
		converted, err := dexparser.ConvertGeyserTransaction(toGeyserProto(toYellowstone(t, tx)), tx.Slot, *tx.BlockTime)
		if err != nil {
			t.Fatalf("%s: %v", sig[:8], err)
		}
		want := dexparser.NewDexParser().ParseAll(tx, cfg)
		got := dexparser.NewDexParser().ParseAll(converted, cfg)
		if jsonOf(t, got) != jsonOf(t, want) {
			t.Errorf("%s: DexParser results differ:\n got %s\nwant %s", sig[:8], jsonOf(t, got), jsonOf(t, want))
		}
		wantShred := dexparser.NewShredParser().ParseAll(tx, cfg)
		gotShred := dexparser.NewShredParser().ParseAll(converted, cfg)
		if jsonOf(t, gotShred) != jsonOf(t, wantShred) {
			t.Errorf("%s: ShredParser results differ", sig[:8])
		}
	}
	if _, err := dexparser.ConvertGeyserTransaction(42, 0, 0); err == nil {
		t.Error("non-struct input accepted")
	}
	if _, err := dexparser.ConvertGeyserTransaction((*pbSubscribeUpdateTransactionInfo)(nil), 0, 0); err == nil {
		t.Error("nil input accepted")
	}
}

// TestGRPCDeshredWithoutMeta: deshred updates carry no meta and the lookup
// table addresses at the top level; the converted transaction has no meta,
// resolved accounts and the same shred results as the executed
// transaction's outer instructions.
func TestGRPCDeshredWithoutMeta(t *testing.T) {
	tx := loadFixture(t, sigJupRouteV1)
	full := toGeyserProto(toYellowstone(t, tx))
	deshred := &pbSubscribeUpdateDeshredTransactionInfo{
		Signature: full.Signature, Transaction: full.Transaction,
		LoadedWritableAddresses: full.Meta.LoadedWritableAddresses, LoadedReadonlyAddresses: full.Meta.LoadedReadonlyAddresses,
	}
	converted, err := dexparser.ConvertGeyserTransaction(deshred, tx.Slot, *tx.BlockTime)
	if err != nil {
		t.Fatal(err)
	}
	if converted.Meta != nil {
		t.Fatal("deshred conversion has meta")
	}
	got := parseShred(t, converted, nil)
	want := parseShred(t, preExec(t, tx), &types.ParseConfig{AddressLookupTables: lookupTables(t, tx)})
	if got.HasUnresolvedAccounts || got.TxStatus != types.TransactionStatusUnknown {
		t.Errorf("unresolved %v status %s", got.HasUnresolvedAccounts, got.TxStatus)
	}
	if jsonOf(t, got.ParsedInstructions) != jsonOf(t, want.ParsedInstructions) {
		t.Errorf("deshred results differ:\n got %s\nwant %s", jsonOf(t, got.ParsedInstructions), jsonOf(t, want.ParsedInstructions))
	}
}

// TestGRPCRoundTripAllFixtures repeats the round trip for every executed
// fixture. DexParser output is not deterministic for some Jupiter
// transactions (AMM choice by map order), so a differing converted result is
// accepted when repeated parses of the RPC form produce it too.
func TestGRPCRoundTripAllFixtures(t *testing.T) {
	cfg := &types.ParseConfig{ParseType: types.ParseAll(), TryUnknownDEX: true, IncludeFailedTxs: true}
	checked := 0
	for _, sig := range fixtureSignatures(t, "json") {
		tx := loadFixture(t, sig)
		if tx.Meta == nil || tx.BlockTime == nil {
			continue
		}
		converted := dexparser.ConvertYellowstoneTransaction(toYellowstone(t, tx), tx.Slot, *tx.BlockTime)
		if jsonOf(t, dexparser.NewShredParser().ParseAll(converted, cfg)) != jsonOf(t, dexparser.NewShredParser().ParseAll(tx, cfg)) {
			t.Errorf("%s: ShredParser results differ", sig)
		}
		got := jsonOf(t, dexparser.NewDexParser().ParseAll(converted, cfg))
		same := false
		for i := 0; i < 300 && !same; i++ {
			same = got == jsonOf(t, dexparser.NewDexParser().ParseAll(tx, cfg))
		}
		if !same {
			t.Errorf("%s: DexParser results differ", sig)
		}
		checked++
	}
	if checked < 200 {
		t.Errorf("only %d fixtures compared", checked)
	}
}
