package adapter

import (
	"encoding/binary"
	"encoding/json"
	"math/big"

	"github.com/mr-tron/base58"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/types"
)

// SolanaTransaction represents a generic Solana transaction interface
// This can be either a parsed transaction or a compiled/versioned transaction
type SolanaTransaction struct {
	Slot        uint64                 `json:"slot"`
	BlockTime   *int64                 `json:"blockTime"`
	Transaction TransactionData        `json:"transaction"`
	Meta        *TransactionMeta       `json:"meta"`
	Version     interface{}            `json:"version"` // can be "legacy", 0, or nil
}

// TransactionData contains the transaction message and signatures
type TransactionData struct {
	Signatures []string        `json:"signatures"`
	Message    TransactionMessage `json:"message"`
}

// TransactionMessage can be either legacy or v0 format
type TransactionMessage struct {
	// Legacy message fields
	AccountKeys     []AccountKey `json:"accountKeys,omitempty"`

	// V0 message fields
	Header              *MessageHeader      `json:"header,omitempty"`
	StaticAccountKeys   []string            `json:"staticAccountKeys,omitempty"`
	CompiledInstructions []CompiledInstruction `json:"compiledInstructions,omitempty"`

	// Shared fields
	Instructions        []interface{}       `json:"instructions,omitempty"`
	AddressTableLookups []AddressTableLookup `json:"addressTableLookups,omitempty"`
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
}

// UnmarshalJSON implements custom unmarshaling to handle both string and object formats
func (a *AccountKey) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		a.Pubkey = s
		return nil
	}

	// Try to unmarshal as object
	type accountKeyObj struct {
		Pubkey   string `json:"pubkey"`
		Signer   bool   `json:"signer"`
		Writable bool   `json:"writable"`
	}
	var obj accountKeyObj
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	a.Pubkey = obj.Pubkey
	a.Signer = obj.Signer
	a.Writable = obj.Writable
	return nil
}

// CompiledInstruction represents a compiled instruction
type CompiledInstruction struct {
	ProgramIdIndex  int    `json:"programIdIndex"`
	Accounts        []int  `json:"accounts,omitempty"`
	AccountKeyIndexes []int `json:"accountKeyIndexes,omitempty"`
	Data            string `json:"data"`
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
	Err               interface{}           `json:"err"`
	Fee               uint64                `json:"fee"`
	PreBalances       []uint64              `json:"preBalances"`
	PostBalances      []uint64              `json:"postBalances"`
	PreTokenBalances  []TokenBalance        `json:"preTokenBalances"`
	PostTokenBalances []TokenBalance        `json:"postTokenBalances"`
	InnerInstructions []InnerInstructionSet `json:"innerInstructions"`
	LogMessages       []string              `json:"logMessages"`
	LoadedAddresses   *LoadedAddresses      `json:"loadedAddresses"`
	ComputeUnitsConsumed *uint64            `json:"computeUnitsConsumed"`
}

// TokenBalance represents a token balance entry
type TokenBalance struct {
	AccountIndex  int              `json:"accountIndex"`
	Mint          string           `json:"mint"`
	Owner         string           `json:"owner"`
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
}

// NewTransactionAdapter creates a new TransactionAdapter
func NewTransactionAdapter(tx *SolanaTransaction, config *types.ParseConfig) *TransactionAdapter {
	adapter := &TransactionAdapter{
		tx:             tx,
		Config:         config,
		SPLTokenMap:    make(map[string]types.TokenInfo),
		SPLDecimalsMap: make(map[string]uint8),
	}
	adapter.AccountKeys = adapter.extractAccountKeys()
	adapter.extractTokenInfo()
	return adapter
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
		Amount:   big.NewInt(int64(fee)).String(),
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

// extractAccountKeys extracts all account keys from the transaction
func (a *TransactionAdapter) extractAccountKeys() []string {
	msg := a.tx.Transaction.Message
	var keys []string

	if a.IsMessageV0() {
		// V0 message
		keys = append(keys, msg.StaticAccountKeys...)
	} else {
		// Legacy message
		for _, key := range msg.AccountKeys {
			if key.Pubkey != "" {
				keys = append(keys, key.Pubkey)
			}
		}
	}

	// Add loaded addresses
	if a.tx.Meta != nil && a.tx.Meta.LoadedAddresses != nil {
		keys = append(keys, a.tx.Meta.LoadedAddresses.Writable...)
		keys = append(keys, a.tx.Meta.LoadedAddresses.Readonly...)
	}

	return keys
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
	for i, key := range a.AccountKeys {
		if key == address {
			return i
		}
	}
	return -1
}

// GetInstruction returns unified instruction data
func (a *TransactionAdapter) GetInstruction(instruction interface{}) *UnifiedInstruction {
	switch ix := instruction.(type) {
	case CompiledInstruction:
		return a.getCompiledInstruction(ix)
	case map[string]interface{}:
		return a.getParsedInstructionFromMap(ix)
	default:
		return nil
	}
}

// UnifiedInstruction represents a unified instruction format
type UnifiedInstruction struct {
	ProgramId string
	Accounts  []string
	Data      []byte
	Parsed    *ParsedData
	Program   string
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

	data, _ := base58.Decode(ix.Data)

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
		var idx int
		switch v := programIdIndex.(type) {
		case float64:
			idx = int(v)
		case int:
			idx = v
		}
		ui.ProgramId = a.GetAccountKey(idx)
	}

	// Check for programId (parsed instruction format)
	if programId, ok := ix["programId"].(string); ok {
		ui.ProgramId = programId
	}

	if program, ok := ix["program"].(string); ok {
		ui.Program = program
	}

	// Handle accounts - can be array of strings or array of integers (indexes)
	if accounts, ok := ix["accounts"].([]interface{}); ok {
		ui.Accounts = make([]string, len(accounts))
		for i, acc := range accounts {
			switch v := acc.(type) {
			case string:
				ui.Accounts[i] = v
			case float64:
				ui.Accounts[i] = a.GetAccountKey(int(v))
			case int:
				ui.Accounts[i] = a.GetAccountKey(v)
			}
		}
	}

	if data, ok := ix["data"].(string); ok {
		ui.Data, _ = base58.Decode(data)
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

// GetTokenAccountOwner returns the owner of a token account
func (a *TransactionAdapter) GetTokenAccountOwner(accountKey string) string {
	for _, balance := range a.PostTokenBalances() {
		if a.AccountKeys[balance.AccountIndex] == accountKey {
			return balance.Owner
		}
	}
	return ""
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

// extractTokenBalances extracts token balances from transaction metadata
func (a *TransactionAdapter) extractTokenBalances() {
	for _, balance := range a.PostTokenBalances() {
		if balance.Mint == "" {
			continue
		}

		accountKey := a.AccountKeys[balance.AccountIndex]
		if _, ok := a.SPLTokenMap[accountKey]; !ok {
			uiAmount := float64(0)
			if balance.UiTokenAmount.UIAmount != nil {
				uiAmount = *balance.UiTokenAmount.UIAmount
			}
			a.SPLTokenMap[accountKey] = types.TokenInfo{
				Mint:      balance.Mint,
				Amount:    uiAmount,
				AmountRaw: balance.UiTokenAmount.Amount,
				Decimals:  balance.UiTokenAmount.Decimals,
			}
		}

		if _, ok := a.SPLDecimalsMap[balance.Mint]; !ok {
			a.SPLDecimalsMap[balance.Mint] = balance.UiTokenAmount.Decimals
		}
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

	if len(ui.Data) == 0 {
		return
	}

	instructionType := ui.Data[0]
	accounts := ui.Accounts

	var source, destination, mint string
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
		if len(accounts) >= 2 {
			source = accounts[0]
			destination = accounts[1]
		}
	}

	a.setTokenInfo(source, destination, mint, decimals)
}

func (a *TransactionAdapter) setTokenInfo(source, destination, mint string, decimals uint8) {
	if source != "" {
		if _, ok := a.SPLTokenMap[source]; !ok {
			m := mint
			if m == "" {
				m = constants.TOKENS.SOL
			}
			d := decimals
			if d == 0 {
				d = 9
			}
			a.SPLTokenMap[source] = types.TokenInfo{
				Mint:      m,
				Amount:    0,
				AmountRaw: "0",
				Decimals:  d,
			}
		}
	}

	if destination != "" {
		if _, ok := a.SPLTokenMap[destination]; !ok {
			m := mint
			if m == "" {
				m = constants.TOKENS.SOL
			}
			d := decimals
			if d == 0 {
				d = 9
			}
			a.SPLTokenMap[destination] = types.TokenInfo{
				Mint:      m,
				Amount:    0,
				AmountRaw: "0",
				Decimals:  d,
			}
		}
	}

	if mint != "" && decimals > 0 {
		if _, ok := a.SPLDecimalsMap[mint]; !ok {
			a.SPLDecimalsMap[mint] = decimals
		}
	}
}

// GetAccountSolBalanceChanges returns SOL balance changes for all accounts
func (a *TransactionAdapter) GetAccountSolBalanceChanges(isOwner bool) map[string]*types.BalanceChange {
	changes := make(map[string]*types.BalanceChange)

	preBalances := a.PreBalances()
	postBalances := a.PostBalances()

	for i, key := range a.AccountKeys {
		accountKey := key
		if isOwner {
			if owner := a.GetTokenAccountOwner(key); owner != "" {
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

		change := int64(postBalance) - int64(preBalance)
		if change != 0 {
			preUI := types.ConvertToUIAmount(new(big.Int).SetUint64(preBalance), 9)
			postUI := types.ConvertToUIAmount(new(big.Int).SetUint64(postBalance), 9)
			changeUI := types.ConvertToUIAmount(new(big.Int).SetInt64(change), 9)

			changes[accountKey] = &types.BalanceChange{
				Pre: types.TokenAmount{
					Amount:   new(big.Int).SetUint64(preBalance).String(),
					UIAmount: &preUI,
					Decimals: 9,
				},
				Post: types.TokenAmount{
					Amount:   new(big.Int).SetUint64(postBalance).String(),
					UIAmount: &postUI,
					Decimals: 9,
				},
				Change: types.TokenAmount{
					Amount:   new(big.Int).SetInt64(change).String(),
					UIAmount: &changeUI,
					Decimals: 9,
				},
			}
		}
	}

	return changes
}

// GetAccountTokenBalanceChanges returns token balance changes for all accounts
func (a *TransactionAdapter) GetAccountTokenBalanceChanges(isOwner bool) map[string]map[string]*types.BalanceChange {
	changes := make(map[string]map[string]*types.BalanceChange)

	// Process pre token balances
	for _, balance := range a.PreTokenBalances() {
		key := a.AccountKeys[balance.AccountIndex]
		accountKey := key
		if isOwner {
			if owner := a.GetTokenAccountOwner(key); owner != "" {
				accountKey = owner
			}
		}

		mint := balance.Mint
		if mint == "" {
			continue
		}

		if _, ok := changes[accountKey]; !ok {
			changes[accountKey] = make(map[string]*types.BalanceChange)
		}

		zeroUI := float64(0)
		changes[accountKey][mint] = &types.BalanceChange{
			Pre: balance.UiTokenAmount,
			Post: types.TokenAmount{
				Amount:   "0",
				UIAmount: &zeroUI,
				Decimals: balance.UiTokenAmount.Decimals,
			},
			Change: types.TokenAmount{
				Amount:   "0",
				UIAmount: &zeroUI,
				Decimals: balance.UiTokenAmount.Decimals,
			},
		}
	}

	// Process post token balances
	for _, balance := range a.PostTokenBalances() {
		key := a.AccountKeys[balance.AccountIndex]
		accountKey := key
		if isOwner {
			if owner := a.GetTokenAccountOwner(key); owner != "" {
				accountKey = owner
			}
		}

		mint := balance.Mint
		if mint == "" {
			continue
		}

		if _, ok := changes[accountKey]; !ok {
			changes[accountKey] = make(map[string]*types.BalanceChange)
		}

		if existing, ok := changes[accountKey][mint]; ok {
			// Update post balance and calculate change
			existing.Post = balance.UiTokenAmount

			preAmount, _ := new(big.Int).SetString(existing.Pre.Amount, 10)
			postAmount, _ := new(big.Int).SetString(balance.UiTokenAmount.Amount, 10)
			changeAmount := new(big.Int).Sub(postAmount, preAmount)

			preUI := float64(0)
			postUI := float64(0)
			if existing.Pre.UIAmount != nil {
				preUI = *existing.Pre.UIAmount
			}
			if balance.UiTokenAmount.UIAmount != nil {
				postUI = *balance.UiTokenAmount.UIAmount
			}
			changeUI := postUI - preUI

			existing.Change = types.TokenAmount{
				Amount:   changeAmount.String(),
				UIAmount: &changeUI,
				Decimals: balance.UiTokenAmount.Decimals,
			}

			if changeAmount.Sign() == 0 {
				delete(changes[accountKey], mint)
				if len(changes[accountKey]) == 0 {
					delete(changes, accountKey)
				}
			}
		} else {
			// No pre-balance, set pre to zero
			zeroUI := float64(0)
			changes[accountKey][mint] = &types.BalanceChange{
				Pre: types.TokenAmount{
					Amount:   "0",
					UIAmount: &zeroUI,
					Decimals: balance.UiTokenAmount.Decimals,
				},
				Post:   balance.UiTokenAmount,
				Change: balance.UiTokenAmount,
			}
		}
	}

	return changes
}

// GetInnerInstruction returns an inner instruction by indices
func (a *TransactionAdapter) GetInnerInstruction(outerIndex, innerIndex int) interface{} {
	for _, inner := range a.InnerInstructions() {
		if inner.Index == outerIndex && innerIndex < len(inner.Instructions) {
			return inner.Instructions[innerIndex]
		}
	}
	return nil
}

// GetTokenAccountBalance returns token balances for given accounts
func (a *TransactionAdapter) GetTokenAccountBalance(accountKeys []string) []*types.TokenAmount {
	result := make([]*types.TokenAmount, len(accountKeys))
	for i, accountKey := range accountKeys {
		if accountKey == "" {
			continue
		}
		for _, balance := range a.PostTokenBalances() {
			if a.AccountKeys[balance.AccountIndex] == accountKey {
				result[i] = &balance.UiTokenAmount
				break
			}
		}
	}
	return result
}

// GetTokenAccountPreBalance returns pre-transaction token balances
func (a *TransactionAdapter) GetTokenAccountPreBalance(accountKeys []string) []*types.TokenAmount {
	result := make([]*types.TokenAmount, len(accountKeys))
	for i, accountKey := range accountKeys {
		if accountKey == "" {
			continue
		}
		for _, balance := range a.PreTokenBalances() {
			if a.AccountKeys[balance.AccountIndex] == accountKey {
				result[i] = &balance.UiTokenAmount
				break
			}
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

// Helper to create TokenAmount from amount and decimals
func createTokenAmount(amount uint64, decimals uint8) types.TokenAmount {
	amountBig := new(big.Int).SetUint64(amount)
	uiAmount := types.ConvertToUIAmount(amountBig, decimals)
	return types.TokenAmount{
		Amount:   amountBig.String(),
		UIAmount: &uiAmount,
		Decimals: decimals,
	}
}

