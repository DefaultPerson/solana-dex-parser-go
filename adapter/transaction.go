package adapter

import (
	"encoding/binary"
	"math/big"
	"reflect"
	"sort"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/goccy/go-json"
	"github.com/mr-tron/base58"
)

// SolanaTransaction represents a generic Solana transaction interface
// This can be either a parsed transaction or a compiled/versioned transaction
type SolanaTransaction struct {
	Slot        uint64           `json:"slot"`
	BlockTime   *int64           `json:"blockTime"`
	Transaction TransactionData  `json:"transaction"`
	Meta        *TransactionMeta `json:"meta"`
	Version     interface{}      `json:"version"` // can be "legacy", 0, 1, or nil
}

// TransactionData contains the transaction message and signatures
type TransactionData struct {
	Signatures []string           `json:"signatures"`
	Message    TransactionMessage `json:"message"`
}

// TransactionMessage can be either legacy or v0 format
type TransactionMessage struct {
	// Legacy message fields
	AccountKeys []AccountKey `json:"accountKeys,omitempty"`

	// V0 message fields
	Header               *MessageHeader        `json:"header,omitempty"`
	StaticAccountKeys    []string              `json:"staticAccountKeys,omitempty"`
	CompiledInstructions []CompiledInstruction `json:"compiledInstructions,omitempty"`

	// Shared fields
	Instructions        []interface{}        `json:"instructions,omitempty"`
	AddressTableLookups []AddressTableLookup `json:"addressTableLookups,omitempty"`

	// TransactionConfig holds the compute budget of version 1 transactions,
	// which carry it in the message instead of ComputeBudget instructions.
	TransactionConfig *TransactionConfig `json:"transactionConfig,omitempty"`
}

// TransactionConfig is the message-level compute budget of a version 1
// transaction (nil fields were not set by the sender)
type TransactionConfig struct {
	ComputeUnitLimit            *uint64 `json:"computeUnitLimit,omitempty"`
	HeapSize                    *uint64 `json:"heapSize,omitempty"`
	LoadedAccountsDataSizeLimit *uint64 `json:"loadedAccountsDataSizeLimit,omitempty"`
	PriorityFee                 *uint64 `json:"priorityFee,omitempty"`
}

// MessageHeader contains message header information
type MessageHeader struct {
	NumRequiredSignatures       int `json:"numRequiredSignatures"`
	NumReadonlySignedAccounts   int `json:"numReadonlySignedAccounts"`
	NumReadonlyUnsignedAccounts int `json:"numReadonlyUnsignedAccounts"`
}

// AccountKey can be either a string or an object with pubkey and signer fields
type AccountKey struct {
	Pubkey   string `json:"pubkey,omitempty"`
	Signer   bool   `json:"signer,omitempty"`
	Writable bool   `json:"writable,omitempty"`
	// Source is set by the jsonParsed encoding: "transaction" or "lookupTable".
	Source string `json:"source,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling to handle a base58 string, a
// Buffer object or byte array, and the jsonParsed object format
func (a *AccountKey) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*a = AccountKey{Pubkey: s}
		return nil
	}

	// Try to unmarshal as object
	type accountKeyObj struct {
		Pubkey   json.RawMessage `json:"pubkey"`
		Signer   bool            `json:"signer"`
		Writable bool            `json:"writable"`
		Source   string          `json:"source"`
		Type     string          `json:"type"`
		Data     json.RawMessage `json:"data"`
	}
	var obj accountKeyObj
	if err := json.Unmarshal(data, &obj); err != nil {
		// Byte array
		key, kerr := keyFromJSON(data)
		if kerr != nil {
			return err
		}
		*a = AccountKey{Pubkey: key}
		return nil
	}
	if obj.Pubkey == nil && obj.Data != nil {
		// Buffer object
		key, err := keyFromJSON(data)
		if err != nil {
			return err
		}
		*a = AccountKey{Pubkey: key}
		return nil
	}
	key, err := keyFromJSON(obj.Pubkey)
	if err != nil {
		return err
	}
	*a = AccountKey{Pubkey: key, Signer: obj.Signer, Writable: obj.Writable, Source: obj.Source}
	return nil
}

// CompiledInstruction represents a compiled instruction
type CompiledInstruction struct {
	ProgramIdIndex    int    `json:"programIdIndex"`
	Accounts          []int  `json:"accounts,omitempty"`
	AccountKeyIndexes []int  `json:"accountKeyIndexes,omitempty"`
	Data              string `json:"data"` // base58
	// DataBytes holds the raw data when it was given as bytes (a Buffer or byte
	// array in JSON); it takes precedence over Data.
	DataBytes []byte `json:"-"`
	// StackHeight is the invocation depth (1 = outer instruction, 2 = its
	// direct CPI, ...); 0 when unknown
	StackHeight int `json:"stackHeight,omitempty"`
}

// ParsedInstruction represents a parsed instruction
type ParsedInstruction struct {
	ProgramId string      `json:"programId"`
	Program   string      `json:"program,omitempty"`
	Accounts  []string    `json:"accounts,omitempty"`
	Data      string      `json:"data,omitempty"`
	Parsed    *ParsedData `json:"parsed,omitempty"`
}

// ParsedData contains parsed instruction data
type ParsedData struct {
	Type string                 `json:"type"`
	Info map[string]interface{} `json:"info"`
}

// AddressTableLookup for address lookup tables
type AddressTableLookup struct {
	AccountKey      string `json:"accountKey"`
	WritableIndexes []int  `json:"writableIndexes"`
	ReadonlyIndexes []int  `json:"readonlyIndexes"`
}

// TransactionMeta contains transaction metadata
type TransactionMeta struct {
	Err                  interface{}           `json:"err"`
	Fee                  uint64                `json:"fee"`
	PreBalances          []uint64              `json:"preBalances"`
	PostBalances         []uint64              `json:"postBalances"`
	PreTokenBalances     []TokenBalance        `json:"preTokenBalances"`
	PostTokenBalances    []TokenBalance        `json:"postTokenBalances"`
	InnerInstructions    []InnerInstructionSet `json:"innerInstructions"`
	LogMessages          []string              `json:"logMessages"`
	LoadedAddresses      *LoadedAddresses      `json:"loadedAddresses"`
	ComputeUnitsConsumed *uint64               `json:"computeUnitsConsumed"`
}

// TokenBalance represents a token balance entry
type TokenBalance struct {
	AccountIndex  int               `json:"accountIndex"`
	Mint          string            `json:"mint"`
	Owner         string            `json:"owner"`
	ProgramId     string            `json:"programId,omitempty"`
	UiTokenAmount types.TokenAmount `json:"uiTokenAmount"`
}

// InnerInstructionSet contains inner instructions for an outer instruction
type InnerInstructionSet struct {
	Index        int           `json:"index"`
	Instructions []interface{} `json:"instructions"`
}

// LoadedAddresses contains loaded address lookup table addresses
type LoadedAddresses struct {
	Writable []string `json:"writable"`
	Readonly []string `json:"readonly"`
}

// TransactionAdapter provides unified access to transaction data
type TransactionAdapter struct {
	tx             *SolanaTransaction
	Config         *types.ParseConfig
	AccountKeys    []string
	SPLTokenMap    map[string]types.TokenInfo
	SPLDecimalsMap map[string]uint8

	// unresolvedLookups is set when address lookup table accounts of a v0
	// message could not be resolved (their AccountKeys entries are "")
	unresolvedLookups bool
	// guessedTokenAccounts marks SPLTokenMap entries whose mint was not known
	// and defaulted to SOL
	guessedTokenAccounts map[string]bool
	// warnings collects non-fatal problems (fetcher errors, unresolved
	// lookup table accounts)
	warnings []string

	// lookup tables built once per transaction
	instructions   []interface{}
	accountIndex   map[string]int
	tokenOwners    map[string]string
	postTokenByKey map[string]*types.TokenAmount
	preTokenByKey  map[string]*types.TokenAmount

	// caches
	parsedCache      map[uintptr]parsedEntry
	dataCache        map[string][]byte
	solChanges       [2]map[string]*types.BalanceChange
	tokenChanges     [2]map[string]map[string]*types.BalanceChange
	solChangesDone   [2]bool
	tokenChangesDone [2]bool
}

// NewTransactionAdapter creates a new TransactionAdapter
func NewTransactionAdapter(tx *SolanaTransaction, config *types.ParseConfig) *TransactionAdapter {
	if tx == nil {
		tx = &SolanaTransaction{}
	}
	adapter := &TransactionAdapter{
		tx:                   tx,
		Config:               config,
		SPLTokenMap:          make(map[string]types.TokenInfo, 32),
		SPLDecimalsMap:       make(map[string]uint8, 16),
		guessedTokenAccounts: make(map[string]bool),
		parsedCache:          make(map[uintptr]parsedEntry, 32),
		dataCache:            make(map[string][]byte, 16),
	}
	adapter.instructions = adapter.buildInstructions()
	adapter.AccountKeys = adapter.extractAccountKeys()
	adapter.buildIndexes()
	adapter.extractTokenInfo()
	return adapter
}

// HasUnresolvedAccounts reports whether some address lookup table accounts of
// a v0 message could not be resolved (no meta.loadedAddresses and no
// ParseConfig.AddressLookupTables / ALTsFetcher entry). Their AccountKeys
// entries are empty strings.
func (a *TransactionAdapter) HasUnresolvedAccounts() bool {
	return a.unresolvedLookups
}

// Warnings returns non-fatal problems met while building the adapter: errors
// returned by ParseConfig.ALTsFetcher or ParseConfig.TokenAccountsFetcher, and
// address lookup table accounts that stayed unresolved. The data affected by
// them is incomplete (empty account keys, token accounts guessed as SOL).
func (a *TransactionAdapter) Warnings() []string {
	return append([]string(nil), a.warnings...)
}

// buildIndexes builds the account index and token balance lookups
func (a *TransactionAdapter) buildIndexes() {
	a.accountIndex = make(map[string]int, len(a.AccountKeys))
	for i, key := range a.AccountKeys {
		if _, ok := a.accountIndex[key]; !ok && key != "" {
			a.accountIndex[key] = i
		}
	}

	// Token account owners from pre and post balances (post wins), so that
	// accounts closed within the transaction still resolve to their owner.
	a.tokenOwners = make(map[string]string)
	a.preTokenByKey = make(map[string]*types.TokenAmount)
	a.postTokenByKey = make(map[string]*types.TokenAmount)
	pre := a.PreTokenBalances()
	for i := range pre {
		key := a.GetAccountKey(pre[i].AccountIndex)
		if key == "" {
			continue
		}
		if pre[i].Owner != "" {
			a.tokenOwners[key] = pre[i].Owner
		}
		if _, ok := a.preTokenByKey[key]; !ok {
			amount := pre[i].UiTokenAmount
			a.preTokenByKey[key] = &amount
		}
	}
	post := a.PostTokenBalances()
	for i := range post {
		key := a.GetAccountKey(post[i].AccountIndex)
		if key == "" {
			continue
		}
		if post[i].Owner != "" {
			a.tokenOwners[key] = post[i].Owner
		}
		if _, ok := a.postTokenByKey[key]; !ok {
			amount := post[i].UiTokenAmount
			a.postTokenByKey[key] = &amount
		}
	}
}

// IsMessageV0 checks if the transaction uses MessageV0 format
func (a *TransactionAdapter) IsMessageV0() bool {
	msg := a.tx.Transaction.Message
	return msg.Header != nil && len(msg.StaticAccountKeys) > 0
}

// Slot returns the transaction slot
func (a *TransactionAdapter) Slot() uint64 {
	return a.tx.Slot
}

// BlockTime returns the transaction block time
func (a *TransactionAdapter) BlockTime() int64 {
	if a.tx.BlockTime != nil {
		return *a.tx.BlockTime
	}
	return 0
}

// Signature returns the transaction signature
func (a *TransactionAdapter) Signature() string {
	if len(a.tx.Transaction.Signatures) > 0 {
		return a.tx.Transaction.Signatures[0]
	}
	return ""
}

// Instructions returns all outer instructions
func (a *TransactionAdapter) Instructions() []interface{} {
	return a.instructions
}

func (a *TransactionAdapter) buildInstructions() []interface{} {
	msg := a.tx.Transaction.Message
	if len(msg.CompiledInstructions) > 0 {
		result := make([]interface{}, len(msg.CompiledInstructions))
		for i, ix := range msg.CompiledInstructions {
			result[i] = ix
		}
		return result
	}
	return msg.Instructions
}

// InstructionAt returns the outer instruction at index, or nil when out of range
func (a *TransactionAdapter) InstructionAt(index int) interface{} {
	if index >= 0 && index < len(a.instructions) {
		return a.instructions[index]
	}
	return nil
}

// InnerInstructions returns inner instructions
func (a *TransactionAdapter) InnerInstructions() []InnerInstructionSet {
	if a.tx.Meta != nil {
		return a.tx.Meta.InnerInstructions
	}
	return nil
}

// PreBalances returns pre-transaction SOL balances
func (a *TransactionAdapter) PreBalances() []uint64 {
	if a.tx.Meta != nil {
		return a.tx.Meta.PreBalances
	}
	return nil
}

// PostBalances returns post-transaction SOL balances
func (a *TransactionAdapter) PostBalances() []uint64 {
	if a.tx.Meta != nil {
		return a.tx.Meta.PostBalances
	}
	return nil
}

// PreTokenBalances returns pre-transaction token balances
func (a *TransactionAdapter) PreTokenBalances() []TokenBalance {
	if a.tx.Meta != nil {
		return a.tx.Meta.PreTokenBalances
	}
	return nil
}

// PostTokenBalances returns post-transaction token balances
func (a *TransactionAdapter) PostTokenBalances() []TokenBalance {
	if a.tx.Meta != nil {
		return a.tx.Meta.PostTokenBalances
	}
	return nil
}

// Signer returns the first signer account
func (a *TransactionAdapter) Signer() string {
	if len(a.AccountKeys) > 0 {
		return a.AccountKeys[0]
	}
	return ""
}

// Signers returns all signer accounts
func (a *TransactionAdapter) Signers() []string {
	msg := a.tx.Transaction.Message
	if msg.Header != nil {
		numSigners := msg.Header.NumRequiredSignatures
		if numSigners > 0 && numSigners <= len(a.AccountKeys) {
			return a.AccountKeys[:numSigners]
		}
	}

	// For legacy transactions, check signer field
	signers := make([]string, 0)
	for _, key := range msg.AccountKeys {
		if key.Signer {
			signers = append(signers, key.Pubkey)
		}
	}
	if len(signers) > 0 {
		return signers
	}

	return []string{a.Signer()}
}

// Fee returns the transaction fee
func (a *TransactionAdapter) Fee() types.TokenAmount {
	fee := uint64(0)
	if a.tx.Meta != nil {
		fee = a.tx.Meta.Fee
	}
	uiAmount := types.ConvertToUIAmount(new(big.Int).SetUint64(fee), 9)
	return types.TokenAmount{
		Amount:   new(big.Int).SetUint64(fee).String(),
		UIAmount: &uiAmount,
		Decimals: 9,
	}
}

// ComputeUnits returns the compute units consumed
func (a *TransactionAdapter) ComputeUnits() uint64 {
	if a.tx.Meta != nil && a.tx.Meta.ComputeUnitsConsumed != nil {
		return *a.tx.Meta.ComputeUnitsConsumed
	}
	return 0
}

// TxStatus returns the transaction status
func (a *TransactionAdapter) TxStatus() types.TransactionStatus {
	if a.tx.Meta == nil {
		return types.TransactionStatusUnknown
	}
	if a.tx.Meta.Err == nil {
		return types.TransactionStatusSuccess
	}
	return types.TransactionStatusFailed
}

// extractAccountKeys extracts all account keys from the transaction: static
// keys, then the writable and then the readonly addresses loaded from address
// lookup tables (meta.loadedAddresses, or resolved from the lookups)
func (a *TransactionAdapter) extractAccountKeys() []string {
	msg := a.tx.Transaction.Message
	var keys []string
	parsedWithLookups := false

	if a.IsMessageV0() {
		// V0 message
		keys = append(keys, msg.StaticAccountKeys...)
	} else {
		// Legacy message (or json/jsonParsed encoding of any version).
		// Keep empty entries so that indexes stay aligned.
		keys = make([]string, 0, len(msg.AccountKeys))
		for _, key := range msg.AccountKeys {
			keys = append(keys, key.Pubkey)
			if key.Source == "lookupTable" {
				parsedWithLookups = true
			}
		}
	}

	// jsonParsed already lists the lookup table addresses in accountKeys
	if parsedWithLookups {
		return keys
	}

	// Add loaded addresses
	if a.tx.Meta != nil && a.tx.Meta.LoadedAddresses != nil {
		keys = append(keys, a.tx.Meta.LoadedAddresses.Writable...)
		keys = append(keys, a.tx.Meta.LoadedAddresses.Readonly...)
		return keys
	}

	if len(msg.AddressTableLookups) > 0 {
		writable, readonly := a.resolveLookups(msg.AddressTableLookups, keys)
		keys = append(keys, writable...)
		keys = append(keys, readonly...)
	}

	return keys
}

// resolveLookups resolves the addresses of address lookup tables when the
// transaction carries no meta.loadedAddresses (pre-execution data such as
// shreds), from ParseConfig.AddressLookupTables first and ParseConfig.ALTsFetcher
// second. Unresolved positions are returned as "" so that later indexes stay
// aligned, and HasUnresolvedAccounts reports them.
func (a *TransactionAdapter) resolveLookups(lookups []AddressTableLookup, staticKeys []string) (writable, readonly []string) {
	var tables map[string][]string
	var fetcher *types.ALTsFetcher
	if a.Config != nil {
		tables = a.Config.AddressLookupTables
		fetcher = a.Config.ALTsFetcher
	}

	var fetched map[string]*types.LoadedAddresses
	if fetcher != nil && fetcher.Fetch != nil && a.fetchAllowed(fetcher.Filter, staticKeys) {
		var missing []types.AddressTableLookup
		for _, l := range lookups {
			if _, ok := tables[l.AccountKey]; !ok {
				missing = append(missing, types.AddressTableLookup{
					AccountKey:      l.AccountKey,
					WritableIndexes: l.WritableIndexes,
					ReadonlyIndexes: l.ReadonlyIndexes,
				})
			}
		}
		if len(missing) > 0 {
			if res, err := fetcher.Fetch(missing); err == nil {
				fetched = res
			} else {
				a.warnings = append(a.warnings, "ALTsFetcher: "+err.Error())
			}
		}
	}

	resolve := func(l AddressTableLookup, indexes []int, fromFetch func(*types.LoadedAddresses) []string) []string {
		out := make([]string, len(indexes))
		if contents, ok := tables[l.AccountKey]; ok {
			for i, idx := range indexes {
				if idx >= 0 && idx < len(contents) {
					out[i] = contents[idx]
				} else {
					a.unresolvedLookups = true
				}
			}
			return out
		}
		if la := fetched[l.AccountKey]; la != nil {
			if addrs := fromFetch(la); len(addrs) == len(indexes) {
				copy(out, addrs)
				return out
			}
		}
		if len(indexes) > 0 {
			a.unresolvedLookups = true
		}
		return out
	}

	for _, l := range lookups {
		writable = append(writable, resolve(l, l.WritableIndexes, func(la *types.LoadedAddresses) []string { return la.Writable })...)
	}
	for _, l := range lookups {
		readonly = append(readonly, resolve(l, l.ReadonlyIndexes, func(la *types.LoadedAddresses) []string { return la.Readonly })...)
	}
	if a.unresolvedLookups {
		a.warnings = append(a.warnings, "unresolved address lookup table accounts")
	}
	return writable, readonly
}

// fetchAllowed applies a fetcher's Filter: FetchFilterProgram requires one of
// ParseConfig.ProgramIds and FetchFilterAccount one of ParseConfig.AccountInclude
// among keys; FetchFilterAll (or empty) always fetches.
func (a *TransactionAdapter) fetchAllowed(filter types.FetchFilterType, keys []string) bool {
	var want []string
	switch filter {
	case types.FetchFilterProgram:
		if a.Config != nil {
			want = a.Config.ProgramIds
		}
	case types.FetchFilterAccount:
		if a.Config != nil {
			want = a.Config.AccountInclude
		}
	default:
		return true
	}
	for _, w := range want {
		for _, k := range keys {
			if w == k {
				return true
			}
		}
	}
	return false
}

// GetAccountKey returns the account key at the given index
func (a *TransactionAdapter) GetAccountKey(index int) string {
	if index >= 0 && index < len(a.AccountKeys) {
		return a.AccountKeys[index]
	}
	return ""
}

// GetAccountIndex returns the index of an account key
func (a *TransactionAdapter) GetAccountIndex(address string) int {
	if i, ok := a.accountIndex[address]; ok {
		return i
	}
	return -1
}

// GetInstruction returns unified instruction data.
// Results are cached per adapter; the returned value must not be modified.
func (a *TransactionAdapter) GetInstruction(instruction interface{}) *UnifiedInstruction {
	switch ix := instruction.(type) {
	case CompiledInstruction:
		return a.getCompiledInstruction(ix)
	case *CompiledInstruction:
		if ix == nil {
			return nil
		}
		return a.getCompiledInstruction(*ix)
	case map[string]interface{}:
		if ix == nil {
			return nil
		}
		key := reflect.ValueOf(ix).Pointer()
		if e, ok := a.parsedCache[key]; ok {
			return e.ui
		}
		ui := a.getParsedInstructionFromMap(ix)
		a.parsedCache[key] = parsedEntry{ix: ix, ui: ui}
		return ui
	default:
		return nil
	}
}

// parsedEntry caches the decoded form of an instruction map. It keeps a
// reference to the map so that the map's address, used as the cache key,
// cannot be reused by another map while the adapter is alive.
type parsedEntry struct {
	ix map[string]interface{}
	ui *UnifiedInstruction
}

// UnifiedInstruction represents a unified instruction format
type UnifiedInstruction struct {
	ProgramId string
	Accounts  []string
	Data      []byte
	Parsed    *ParsedData
	Program   string
}

// decodeData base58-decodes instruction data once per distinct string
func (a *TransactionAdapter) decodeData(data string) []byte {
	if data == "" {
		return nil
	}
	if b, ok := a.dataCache[data]; ok {
		return b
	}
	b, _ := base58.Decode(data)
	a.dataCache[data] = b
	return b
}

func (a *TransactionAdapter) getCompiledInstruction(ix CompiledInstruction) *UnifiedInstruction {
	programId := a.GetAccountKey(ix.ProgramIdIndex)

	// Get account indices
	accountIndices := ix.Accounts
	if len(accountIndices) == 0 {
		accountIndices = ix.AccountKeyIndexes
	}

	accounts := make([]string, len(accountIndices))
	for i, idx := range accountIndices {
		accounts[i] = a.GetAccountKey(idx)
	}

	data := ix.DataBytes
	if data == nil {
		data = a.decodeData(ix.Data)
	}

	return &UnifiedInstruction{
		ProgramId: programId,
		Accounts:  accounts,
		Data:      data,
	}
}

func (a *TransactionAdapter) getParsedInstructionFromMap(ix map[string]interface{}) *UnifiedInstruction {
	ui := &UnifiedInstruction{}

	// Check for programIdIndex (compiled instruction format)
	if programIdIndex, ok := ix["programIdIndex"]; ok {
		if idx, ok := intFromValue(programIdIndex); ok {
			ui.ProgramId = a.GetAccountKey(idx)
		}
	}

	// Check for programId (parsed instruction format)
	if programId, ok := ix["programId"]; ok {
		if key := keyFromValue(programId); key != "" {
			ui.ProgramId = key
		}
	}

	if program, ok := ix["program"].(string); ok {
		ui.Program = program
	}

	// Accounts: keys (jsonParsed) or indexes (json), as a JSON array, a Go
	// slice, a Buffer object or a base58 string of index bytes
	accounts, ok := ix["accounts"]
	if !ok {
		accounts = ix["accountKeyIndexes"]
	}
	switch v := accounts.(type) {
	case []interface{}:
		ui.Accounts = make([]string, len(v))
		for i, acc := range v {
			if s, ok := acc.(string); ok {
				ui.Accounts[i] = s
			} else if idx, ok := intFromValue(acc); ok {
				ui.Accounts[i] = a.GetAccountKey(idx)
			} else {
				ui.Accounts[i] = keyFromValue(acc)
			}
		}
	case []string:
		ui.Accounts = append([]string(nil), v...)
	case []int:
		ui.Accounts = make([]string, len(v))
		for i, idx := range v {
			ui.Accounts[i] = a.GetAccountKey(idx)
		}
	case string:
		if b, err := base58.Decode(v); err == nil {
			ui.Accounts = a.keysForIndexBytes(b)
		}
	default:
		if b, ok := bytesFromValue(v); ok {
			ui.Accounts = a.keysForIndexBytes(b)
		}
	}

	switch data := ix["data"].(type) {
	case string:
		ui.Data = a.decodeData(data)
	case nil:
	default:
		if b, ok := bytesFromValue(data); ok {
			ui.Data = b
		}
	}
	if parsed, ok := ix["parsed"].(map[string]interface{}); ok {
		ui.Parsed = &ParsedData{}
		if t, ok := parsed["type"].(string); ok {
			ui.Parsed.Type = t
		}
		if info, ok := parsed["info"].(map[string]interface{}); ok {
			ui.Parsed.Info = info
		}
	}

	return ui
}

func (a *TransactionAdapter) keysForIndexBytes(b []byte) []string {
	keys := make([]string, len(b))
	for i, idx := range b {
		keys[i] = a.GetAccountKey(int(idx))
	}
	return keys
}

// IsCompiledInstruction checks if an instruction is compiled
func (a *TransactionAdapter) IsCompiledInstruction(instruction interface{}) bool {
	switch ix := instruction.(type) {
	case CompiledInstruction:
		return true
	case map[string]interface{}:
		_, hasProgramIdIndex := ix["programIdIndex"]
		_, hasParsed := ix["parsed"]
		return hasProgramIdIndex && !hasParsed
	default:
		return false
	}
}

// GetInstructionProgramId returns the program ID from an instruction
func (a *TransactionAdapter) GetInstructionProgramId(instruction interface{}) string {
	ui := a.GetInstruction(instruction)
	if ui != nil {
		return ui.ProgramId
	}
	return ""
}

// GetInstructionAccounts returns the accounts from an instruction
func (a *TransactionAdapter) GetInstructionAccounts(instruction interface{}) []string {
	ui := a.GetInstruction(instruction)
	if ui != nil {
		return ui.Accounts
	}
	return nil
}

// GetInstructionData returns the data from an instruction
func (a *TransactionAdapter) GetInstructionData(instruction interface{}) []byte {
	ui := a.GetInstruction(instruction)
	if ui != nil {
		return ui.Data
	}
	return nil
}

// GetTokenAccountOwner returns the owner of a token account, from the post
// token balances, then the pre token balances (accounts closed in the
// transaction), then ParseConfig.TokenAccountsFetcher results
func (a *TransactionAdapter) GetTokenAccountOwner(accountKey string) string {
	return a.tokenOwners[accountKey]
}

// IsSupportedToken checks if a token is supported
func (a *TransactionAdapter) IsSupportedToken(mint string) bool {
	return constants.IsSOL(mint) || constants.IsStablecoin(mint)
}

// GetTokenDecimals returns the decimals for a token
func (a *TransactionAdapter) GetTokenDecimals(mint string) uint8 {
	if decimals, ok := a.SPLDecimalsMap[mint]; ok {
		return decimals
	}
	if decimals, ok := constants.TOKEN_DECIMALS[mint]; ok {
		return decimals
	}
	return 0
}

// GetSplTokenMint returns the mint address for a token account
func (a *TransactionAdapter) GetSplTokenMint(tokenAccount string) string {
	if info, ok := a.SPLTokenMap[tokenAccount]; ok {
		return info.Mint
	}
	return ""
}

// GetPoolEventBase creates a base pool event
func (a *TransactionAdapter) GetPoolEventBase(eventType types.PoolEventType, programId string) types.PoolEventBase {
	return types.PoolEventBase{
		User:      a.Signer(),
		Type:      eventType,
		ProgramId: programId,
		AMM:       constants.GetProgramName(programId),
		Slot:      a.Slot(),
		Timestamp: a.BlockTime(),
		Signature: a.Signature(),
	}
}

// extractTokenInfo extracts token information from the transaction
func (a *TransactionAdapter) extractTokenInfo() {
	a.extractTokenBalances()
	a.extractTokenFromInstructions()
	a.fetchTokenAccounts()

	// Add SOL if not exists
	if _, ok := a.SPLTokenMap[constants.TOKENS.SOL]; !ok {
		a.SPLTokenMap[constants.TOKENS.SOL] = types.TokenInfo{
			Mint:      constants.TOKENS.SOL,
			Amount:    0,
			AmountRaw: "0",
			Decimals:  9,
		}
	}
	if _, ok := a.SPLDecimalsMap[constants.TOKENS.SOL]; !ok {
		a.SPLDecimalsMap[constants.TOKENS.SOL] = 9
	}
}

// extractTokenBalances extracts token info from the post token balances, then
// from pre token balances of accounts closed within the transaction
func (a *TransactionAdapter) extractTokenBalances() {
	for _, balance := range a.PostTokenBalances() {
		a.setTokenInfoFromBalance(balance, true)
	}
	for _, balance := range a.PreTokenBalances() {
		a.setTokenInfoFromBalance(balance, false)
	}
}

func (a *TransactionAdapter) setTokenInfoFromBalance(balance TokenBalance, post bool) {
	if balance.Mint == "" {
		return
	}
	accountKey := a.GetAccountKey(balance.AccountIndex)
	if accountKey == "" {
		return
	}
	if _, ok := a.SPLTokenMap[accountKey]; !ok {
		info := types.TokenInfo{
			Mint:      balance.Mint,
			AmountRaw: "0",
			Decimals:  balance.UiTokenAmount.Decimals,
		}
		// SPLTokenMap amounts are post balances; a pre-only account was closed
		if post {
			info.AmountRaw = balance.UiTokenAmount.Amount
			if balance.UiTokenAmount.UIAmount != nil {
				info.Amount = *balance.UiTokenAmount.UIAmount
			}
		}
		a.SPLTokenMap[accountKey] = info
	}

	if _, ok := a.SPLDecimalsMap[balance.Mint]; !ok {
		a.SPLDecimalsMap[balance.Mint] = balance.UiTokenAmount.Decimals
	}
}

// extractTokenFromInstructions extracts token info from transfer instructions
func (a *TransactionAdapter) extractTokenFromInstructions() {
	for _, ix := range a.Instructions() {
		a.extractFromInstruction(ix)
	}

	for _, inner := range a.InnerInstructions() {
		for _, ix := range inner.Instructions {
			a.extractFromInstruction(ix)
		}
	}
}

func (a *TransactionAdapter) extractFromInstruction(ix interface{}) {
	ui := a.GetInstruction(ix)
	if ui == nil {
		return
	}

	// Only process token program instructions
	if ui.ProgramId != constants.TOKEN_PROGRAM_ID && ui.ProgramId != constants.TOKEN_2022_PROGRAM_ID {
		return
	}

	if ui.Parsed != nil {
		a.extractFromParsedInstruction(ui)
		return
	}

	if len(ui.Data) == 0 {
		return
	}

	instructionType := ui.Data[0]
	accounts := ui.Accounts

	var source, destination, mint, owner string
	var decimals uint8

	switch instructionType {
	case constants.SPLTokenTransfer:
		if len(accounts) >= 2 {
			source = accounts[0]
			destination = accounts[1]
		}
	case constants.SPLTokenTransferChecked:
		if len(accounts) >= 3 {
			source = accounts[0]
			mint = accounts[1]
			destination = accounts[2]
			if len(ui.Data) > 9 {
				decimals = ui.Data[9]
			}
		}
	case constants.SPLTokenMintTo:
		if len(accounts) >= 2 {
			mint = accounts[0]
			destination = accounts[1]
		}
	case constants.SPLTokenMintToChecked:
		if len(accounts) >= 2 {
			mint = accounts[0]
			destination = accounts[1]
			if len(ui.Data) > 9 {
				decimals = ui.Data[9]
			}
		}
	case constants.SPLTokenBurn:
		if len(accounts) >= 2 {
			source = accounts[0]
			mint = accounts[1]
		}
	case constants.SPLTokenBurnChecked:
		if len(accounts) >= 2 {
			source = accounts[0]
			mint = accounts[1]
			if len(ui.Data) > 9 {
				decimals = ui.Data[9]
			}
		}
	case constants.SPLTokenCloseAccount:
		// The destination of CloseAccount receives lamports, not tokens.
		if len(accounts) >= 1 {
			source = accounts[0]
		}
	case splTokenInitializeAccount:
		// accounts: account, mint, owner, rent
		if len(accounts) >= 3 {
			destination = accounts[0]
			mint = accounts[1]
			owner = accounts[2]
		}
	case splTokenInitializeAccount2, splTokenInitializeAccount3:
		// accounts: account, mint (, rent); data: owner pubkey
		if len(accounts) >= 2 {
			destination = accounts[0]
			mint = accounts[1]
			if len(ui.Data) >= 33 {
				owner = base58.Encode(ui.Data[1:33])
			}
		}
	case splTokenTransferFeeExtension:
		// TransferCheckedWithFee: accounts source, mint, destination, authority;
		// data 26, 1, amount u64, decimals u8, fee u64
		if len(ui.Data) >= 11 && ui.Data[1] == splTokenTransferCheckedWithFee && len(accounts) >= 3 {
			source = accounts[0]
			mint = accounts[1]
			destination = accounts[2]
			decimals = ui.Data[10]
		}
	}

	a.setTokenInfo(source, destination, mint, decimals)
	if owner != "" && destination != "" {
		if _, ok := a.tokenOwners[destination]; !ok {
			a.tokenOwners[destination] = owner
		}
	}
}

// SPL Token instruction tags used only by the adapter
const (
	splTokenInitializeAccount      = 1
	splTokenInitializeAccount2     = 16
	splTokenInitializeAccount3     = 18
	splTokenTransferFeeExtension   = 26
	splTokenTransferCheckedWithFee = 1
)

// extractFromParsedInstruction extracts token info from a jsonParsed token
// program instruction (program "spl-token", for Token and Token-2022)
func (a *TransactionAdapter) extractFromParsedInstruction(ui *UnifiedInstruction) {
	info := ui.Parsed.Info
	if info == nil {
		return
	}
	str := func(key string) string {
		v, _ := info[key].(string)
		return v
	}
	source, destination, mint := str("source"), str("destination"), str("mint")
	var decimals uint8
	if d, ok := intFromValue(info["decimals"]); ok && d >= 0 && d < 256 {
		decimals = uint8(d)
	} else if ta, ok := info["tokenAmount"].(map[string]interface{}); ok {
		if d, ok := intFromValue(ta["decimals"]); ok && d >= 0 && d < 256 {
			decimals = uint8(d)
		}
	}
	owner := ""
	switch ui.Parsed.Type {
	case "mintTo", "mintToChecked":
		destination = str("account")
	case "burn", "burnChecked":
		source = str("account")
	case "closeAccount":
		source, destination = str("account"), ""
	case "initializeAccount", "initializeAccount2", "initializeAccount3":
		destination = str("account")
		owner = str("owner")
	}
	if source == "" && destination == "" {
		return
	}
	a.setTokenInfo(source, destination, mint, decimals)
	if owner != "" && destination != "" {
		if _, ok := a.tokenOwners[destination]; !ok {
			a.tokenOwners[destination] = owner
		}
	}
}

func (a *TransactionAdapter) setTokenInfo(source, destination, mint string, decimals uint8) {
	a.setTokenAccountInfo(source, mint, decimals)
	a.setTokenAccountInfo(destination, mint, decimals)

	if mint != "" && decimals > 0 {
		if _, ok := a.SPLDecimalsMap[mint]; !ok {
			a.SPLDecimalsMap[mint] = decimals
		}
	}
}

// setTokenAccountInfo records the mint of a token account. Accounts whose mint
// is unknown default to SOL (typically temporary WSOL accounts) and are marked
// as guessed; a later instruction naming the mint replaces the guess. Entries
// from token balances or with a known mint are kept.
func (a *TransactionAdapter) setTokenAccountInfo(account, mint string, decimals uint8) {
	if account == "" {
		return
	}
	_, exists := a.SPLTokenMap[account]
	if exists && !a.guessedTokenAccounts[account] {
		return
	}
	if mint == "" {
		if !exists {
			a.SPLTokenMap[account] = types.TokenInfo{
				Mint:      constants.TOKENS.SOL,
				Amount:    0,
				AmountRaw: "0",
				Decimals:  9,
			}
			a.guessedTokenAccounts[account] = true
		}
		return
	}
	if decimals == 0 {
		if d, ok := a.SPLDecimalsMap[mint]; ok {
			decimals = d
		} else if d, ok := constants.TOKEN_DECIMALS[mint]; ok {
			decimals = d
		}
	}
	a.SPLTokenMap[account] = types.TokenInfo{
		Mint:      mint,
		Amount:    0,
		AmountRaw: "0",
		Decimals:  decimals,
	}
	delete(a.guessedTokenAccounts, account)
}

// IsGuessedTokenAccount reports whether the mint recorded for a token account
// in SPLTokenMap is a SOL default because the transaction does not reveal it
func (a *TransactionAdapter) IsGuessedTokenAccount(account string) bool {
	return a.guessedTokenAccounts[account]
}

// fetchTokenAccounts resolves guessed token accounts with
// ParseConfig.TokenAccountsFetcher
func (a *TransactionAdapter) fetchTokenAccounts() {
	if a.Config == nil || a.Config.TokenAccountsFetcher == nil || a.Config.TokenAccountsFetcher.Fetch == nil {
		return
	}
	if len(a.guessedTokenAccounts) == 0 || !a.fetchAllowed(a.Config.TokenAccountsFetcher.Filter, a.AccountKeys) {
		return
	}
	keys := make([]string, 0, len(a.guessedTokenAccounts))
	for key := range a.guessedTokenAccounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	infos, err := a.Config.TokenAccountsFetcher.Fetch(keys)
	if err != nil {
		a.warnings = append(a.warnings, "TokenAccountsFetcher: "+err.Error())
		return
	}
	for i, info := range infos {
		if i >= len(keys) || info == nil || info.Mint == "" {
			continue
		}
		key := keys[i]
		amount, ok := new(big.Int).SetString(info.Amount, 10)
		if !ok {
			amount = new(big.Int)
		}
		a.SPLTokenMap[key] = types.TokenInfo{
			Mint:      info.Mint,
			Amount:    types.ConvertToUIAmount(amount, info.Decimals),
			AmountRaw: amount.String(),
			Decimals:  info.Decimals,
		}
		delete(a.guessedTokenAccounts, key)
		if _, ok := a.SPLDecimalsMap[info.Mint]; !ok {
			a.SPLDecimalsMap[info.Mint] = info.Decimals
		}
		if _, ok := a.tokenOwners[key]; !ok && info.Owner != "" {
			a.tokenOwners[key] = info.Owner
		}
	}
}

// GetAccountSolBalanceChanges returns SOL balance changes for all accounts.
// With isOwner, token accounts are attributed to their owner and all accounts
// of an owner are summed. The returned map is cached per adapter and shared:
// callers must not modify it.
func (a *TransactionAdapter) GetAccountSolBalanceChanges(isOwner bool) map[string]*types.BalanceChange {
	slot := 0
	if isOwner {
		slot = 1
	}
	if a.solChangesDone[slot] {
		return a.solChanges[slot]
	}

	preBalances := a.PreBalances()
	postBalances := a.PostBalances()

	type sums struct{ pre, post *big.Int }
	totals := make(map[string]*sums)
	for i, key := range a.AccountKeys {
		if key == "" {
			continue
		}
		accountKey := key
		if isOwner {
			if owner := a.tokenOwners[key]; owner != "" {
				accountKey = owner
			}
		}

		preBalance := uint64(0)
		postBalance := uint64(0)
		if i < len(preBalances) {
			preBalance = preBalances[i]
		}
		if i < len(postBalances) {
			postBalance = postBalances[i]
		}

		t, ok := totals[accountKey]
		if !ok {
			t = &sums{pre: new(big.Int), post: new(big.Int)}
			totals[accountKey] = t
		}
		t.pre.Add(t.pre, new(big.Int).SetUint64(preBalance))
		t.post.Add(t.post, new(big.Int).SetUint64(postBalance))
	}

	changes := make(map[string]*types.BalanceChange)
	for accountKey, t := range totals {
		change := new(big.Int).Sub(t.post, t.pre)
		if change.Sign() != 0 {
			changes[accountKey] = newBalanceChange(t.pre, t.post, change, 9)
		}
	}

	a.solChanges[slot] = changes
	a.solChangesDone[slot] = true
	return changes
}

// GetAccountTokenBalanceChanges returns token balance changes for all accounts
// (account -> mint -> change). With isOwner, token accounts are attributed to
// their owner and all accounts of an owner and mint are summed. Accounts closed
// within the transaction (pre balance only) have post 0 and change -pre.
// Entries without change are omitted. The returned map is cached per adapter
// and shared: callers must not modify it.
func (a *TransactionAdapter) GetAccountTokenBalanceChanges(isOwner bool) map[string]map[string]*types.BalanceChange {
	slot := 0
	if isOwner {
		slot = 1
	}
	if a.tokenChangesDone[slot] {
		return a.tokenChanges[slot]
	}

	type sums struct {
		pre, post *big.Int
		decimals  uint8
	}
	totals := make(map[string]map[string]*sums)
	add := func(balance TokenBalance, isPost bool) {
		key := a.GetAccountKey(balance.AccountIndex)
		if key == "" || balance.Mint == "" {
			return
		}
		accountKey := key
		if isOwner {
			if owner := a.tokenOwners[key]; owner != "" {
				accountKey = owner
			}
		}
		amount, ok := new(big.Int).SetString(balance.UiTokenAmount.Amount, 10)
		if !ok {
			amount = new(big.Int)
		}
		byMint, ok := totals[accountKey]
		if !ok {
			byMint = make(map[string]*sums)
			totals[accountKey] = byMint
		}
		t, ok := byMint[balance.Mint]
		if !ok {
			t = &sums{pre: new(big.Int), post: new(big.Int), decimals: balance.UiTokenAmount.Decimals}
			byMint[balance.Mint] = t
		}
		if isPost {
			t.post.Add(t.post, amount)
		} else {
			t.pre.Add(t.pre, amount)
		}
	}
	for _, balance := range a.PreTokenBalances() {
		add(balance, false)
	}
	for _, balance := range a.PostTokenBalances() {
		add(balance, true)
	}

	changes := make(map[string]map[string]*types.BalanceChange)
	for accountKey, byMint := range totals {
		for mint, t := range byMint {
			change := new(big.Int).Sub(t.post, t.pre)
			if change.Sign() == 0 {
				continue
			}
			if _, ok := changes[accountKey]; !ok {
				changes[accountKey] = make(map[string]*types.BalanceChange)
			}
			changes[accountKey][mint] = newBalanceChange(t.pre, t.post, change, t.decimals)
		}
	}

	a.tokenChanges[slot] = changes
	a.tokenChangesDone[slot] = true
	return changes
}

func newBalanceChange(pre, post, change *big.Int, decimals uint8) *types.BalanceChange {
	amount := func(v *big.Int) types.TokenAmount {
		ui := types.ConvertToUIAmount(v, decimals)
		return types.TokenAmount{Amount: v.String(), UIAmount: &ui, Decimals: decimals}
	}
	return &types.BalanceChange{Pre: amount(pre), Post: amount(post), Change: amount(change)}
}

// GetInnerInstruction returns an inner instruction by indices
func (a *TransactionAdapter) GetInnerInstruction(outerIndex, innerIndex int) interface{} {
	for _, inner := range a.InnerInstructions() {
		if inner.Index == outerIndex && innerIndex >= 0 && innerIndex < len(inner.Instructions) {
			return inner.Instructions[innerIndex]
		}
	}
	return nil
}

// GetInstructionStackHeight returns the invocation depth of an instruction:
// 1 for an outer instruction (innerIndex < 0), the recorded stackHeight for an
// inner instruction, or 0 when it is unknown (older transactions, or input
// without stackHeight)
func (a *TransactionAdapter) GetInstructionStackHeight(outerIndex, innerIndex int) int {
	if innerIndex < 0 {
		if a.InstructionAt(outerIndex) == nil {
			return 0
		}
		return 1
	}
	return InstructionStackHeight(a.GetInnerInstruction(outerIndex, innerIndex))
}

// GetParentInstruction returns the instruction that invoked the inner
// instruction at (outerIndex, innerIndex), for example the swap instruction
// that emitted a CPI event: the nearest preceding instruction of the same outer
// group whose stack height is one less. parentInner is -1 when the parent is
// the outer instruction. ok is false for outer instructions, unknown indexes
// and inner instructions without stack height.
func (a *TransactionAdapter) GetParentInstruction(outerIndex, innerIndex int) (parent interface{}, parentInner int, ok bool) {
	height := a.GetInstructionStackHeight(outerIndex, innerIndex)
	if innerIndex < 0 || height < 2 {
		return nil, 0, false
	}
	if height == 2 {
		if ix := a.InstructionAt(outerIndex); ix != nil {
			return ix, -1, true
		}
		return nil, 0, false
	}
	for j := innerIndex - 1; j >= 0; j-- {
		ix := a.GetInnerInstruction(outerIndex, j)
		h := InstructionStackHeight(ix)
		if h == height-1 {
			return ix, j, true
		}
		if h == 0 || h < height-1 {
			// unknown height, or the parent's level was left: no parent
			return nil, 0, false
		}
	}
	return nil, 0, false
}

// InstructionStackHeight reads the stackHeight of an instruction (compiled,
// Go-typed or JSON map form), or 0 when it is absent
func InstructionStackHeight(ix interface{}) int {
	switch v := ix.(type) {
	case CompiledInstruction:
		return v.StackHeight
	case *CompiledInstruction:
		if v != nil {
			return v.StackHeight
		}
	case map[string]interface{}:
		if h, ok := intFromValue(v["stackHeight"]); ok {
			return h
		}
	}
	return 0
}

// GetTokenAccountBalance returns token balances for given accounts
func (a *TransactionAdapter) GetTokenAccountBalance(accountKeys []string) []*types.TokenAmount {
	return tokenAmounts(accountKeys, a.postTokenByKey)
}

// GetTokenAccountPreBalance returns pre-transaction token balances
func (a *TransactionAdapter) GetTokenAccountPreBalance(accountKeys []string) []*types.TokenAmount {
	return tokenAmounts(accountKeys, a.preTokenByKey)
}

func tokenAmounts(accountKeys []string, byKey map[string]*types.TokenAmount) []*types.TokenAmount {
	result := make([]*types.TokenAmount, len(accountKeys))
	for i, accountKey := range accountKeys {
		if amount, ok := byKey[accountKey]; ok && accountKey != "" {
			c := *amount
			result[i] = &c
		}
	}
	return result
}

// GetAccountBalance returns SOL balances for given accounts
func (a *TransactionAdapter) GetAccountBalance(accountKeys []string) []*types.TokenAmount {
	result := make([]*types.TokenAmount, len(accountKeys))
	postBalances := a.PostBalances()

	for i, accountKey := range accountKeys {
		if accountKey == "" {
			continue
		}
		index := a.GetAccountIndex(accountKey)
		if index >= 0 && index < len(postBalances) {
			amount := postBalances[index]
			uiAmount := types.ConvertToUIAmount(new(big.Int).SetUint64(amount), 9)
			result[i] = &types.TokenAmount{
				Amount:   new(big.Int).SetUint64(amount).String(),
				UIAmount: &uiAmount,
				Decimals: 9,
			}
		}
	}
	return result
}

// GetAccountPreBalance returns pre-transaction SOL balances
func (a *TransactionAdapter) GetAccountPreBalance(accountKeys []string) []*types.TokenAmount {
	result := make([]*types.TokenAmount, len(accountKeys))
	preBalances := a.PreBalances()

	for i, accountKey := range accountKeys {
		if accountKey == "" {
			continue
		}
		index := a.GetAccountIndex(accountKey)
		if index >= 0 && index < len(preBalances) {
			amount := preBalances[index]
			uiAmount := types.ConvertToUIAmount(new(big.Int).SetUint64(amount), 9)
			result[i] = &types.TokenAmount{
				Amount:   new(big.Int).SetUint64(amount).String(),
				UIAmount: &uiAmount,
				Decimals: 9,
			}
		}
	}
	return result
}

// LogMessages returns the transaction log messages
func (a *TransactionAdapter) LogMessages() []string {
	if a.tx.Meta != nil {
		return a.tx.Meta.LogMessages
	}
	return nil
}

// ParseTransferAmount parses the amount from transfer instruction data
func ParseTransferAmount(data []byte) uint64 {
	if len(data) < 9 {
		return 0
	}
	return binary.LittleEndian.Uint64(data[1:9])
}

// ParseTransferCheckedAmount parses the amount from transfer checked instruction data
func ParseTransferCheckedAmount(data []byte) uint64 {
	if len(data) < 9 {
		return 0
	}
	return binary.LittleEndian.Uint64(data[1:9])
}

// GetRawTransaction returns the underlying transaction
func (a *TransactionAdapter) GetRawTransaction() *SolanaTransaction {
	return a.tx
}
