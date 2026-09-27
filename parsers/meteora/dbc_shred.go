package meteora

import (
	"bytes"
	"strconv"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// DBCShredParser parses Meteora DBC instructions from shred-stream
type DBCShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewDBCShredParser creates a new DBCShredParser
func NewDBCShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *DBCShredParser {
	return &DBCShredParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes Meteora DBC instructions and returns parsed results
func (p *DBCShredParser) ProcessInstructions() []interface{} {
	events, _ := p.ProcessAll()
	return events
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *DBCShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	_, typed := p.ProcessAll()
	return typed
}

// DBC swap2 swap modes (SwapParameters2.swap_mode)
const (
	dbcSwapModeExactIn     = 0
	dbcSwapModePartialFill = 1
	dbcSwapModeExactOut    = 2
)

// ProcessAll decodes the DBC instructions into legacy events and typed meme
// events in a single pass
func (p *DBCShredParser) ProcessAll() ([]interface{}, []types.ParsedShredInstruction) {
	var events []interface{}
	var typed []types.ParsedShredInstruction
	d := constants.DISCRIMINATORS.METEORA_DBC

	for _, ci := range p.classifier.GetInstructions(constants.DEX_PROGRAMS.METEORA_DBC.ID) {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}
		accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
		disc, payload := data[:8], data[8:]

		var eventType string
		var eventData interface{}
		var memeEvent *types.MemeEvent
		var inKind, outKind types.ShredAmountKind

		switch {
		case bytes.Equal(disc, d.SWAP):
			if swap := p.decodeSwapInstruction(accounts, payload, false); swap != nil {
				eventType, eventData, memeEvent = "swap", swap, p.swapMemeEvent(swap)
				inKind, outKind = dbcSwapAmountKinds(swap)
			}
		case bytes.Equal(disc, d.SWAP_V2), bytes.Equal(disc, d.SWAP2_WITH_TRANSFER_HOOK):
			if swap := p.decodeSwapInstruction(accounts, payload, true); swap != nil {
				eventType, eventData, memeEvent = "swap_v2", swap, p.swapMemeEvent(swap)
				inKind, outKind = dbcSwapAmountKinds(swap)
			}
		case bytes.Equal(disc, d.INITIALIZE_VIRTUAL_POOL_WITH_SPL):
			if init := p.decodeInitPool(accounts, payload); init != nil {
				eventType, eventData, memeEvent = "init_pool_spl", init, dbcShredInitMemeEvent(init)
			}
		case bytes.Equal(disc, d.INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022), bytes.Equal(disc, d.INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022_TRANSFER_HOOK):
			if init := p.decodeInitPool(accounts, payload); init != nil {
				eventType, eventData, memeEvent = "init_pool_2022", init, dbcShredInitMemeEvent(init)
			}
		case bytes.Equal(disc, d.METEORA_DBC_MIGRATE_DAMM):
			if migrate := p.decodeMigrateDammInstruction(accounts); migrate != nil {
				eventType, eventData, memeEvent = "migrate_damm", migrate, dbcShredMigrateMemeEvent(migrate)
			}
		case bytes.Equal(disc, d.METEORA_DBC_MIGRATE_DAMM_V2):
			if migrate := p.decodeMigrateDammV2Instruction(accounts); migrate != nil {
				eventType, eventData, memeEvent = "migrate_damm_v2", migrate, dbcShredMigrateMemeEvent(migrate)
			}
		default:
			continue
		}

		if eventData == nil {
			continue
		}
		idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
		events = append(events, &DBCShredInstruction{
			Type:               eventType,
			Data:               eventData,
			Slot:               p.adapter.Slot(),
			Timestamp:          p.adapter.BlockTime(),
			Signature:          p.adapter.Signature(),
			Idx:                idx,
			Signer:             p.adapter.Signers(),
			UnresolvedAccounts: types.HasUnresolvedAccount(accounts),
		})

		memeEvent.Protocol = constants.DEX_PROGRAMS.METEORA_DBC.Name
		memeEvent.Signature = p.adapter.Signature()
		memeEvent.Slot = p.adapter.Slot()
		memeEvent.Timestamp = p.adapter.BlockTime()
		memeEvent.Idx = idx
		typed = append(typed, types.ParsedShredInstruction{
			ProgramID:        constants.DEX_PROGRAMS.METEORA_DBC.ID,
			ProgramName:      constants.DEX_PROGRAMS.METEORA_DBC.Name,
			Action:           eventType,
			MemeEvent:        memeEvent,
			Accounts:         accounts,
			Idx:              idx,
			InputAmountKind:  inKind,
			OutputAmountKind: outKind,
		})
	}

	return events, typed
}

// DBCShredInstruction represents a parsed Meteora DBC instruction
type DBCShredInstruction struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Slot      uint64      `json:"slot"`
	Timestamp int64       `json:"timestamp"`
	Signature string      `json:"signature"`
	Idx       string      `json:"idx"`
	Signer    []string    `json:"signer"`
	// UnresolvedAccounts is true when some of the instruction's accounts are
	// address lookup table entries that could not be resolved (empty
	// strings); the decoded data leaves them empty
	UnresolvedAccounts bool `json:"unresolvedAccounts,omitempty"`
}

// DBCSwapData contains Meteora DBC swap instruction data. The amounts are
// instruction arguments: in exact-in mode (swap, and swap2 with SwapMode 0)
// InputAmount is the input and OutputAmount the minimum output; in
// partial-fill mode (SwapMode 1) InputAmount is the maximum input (the
// program takes less when the pool reaches its migration threshold) and
// OutputAmount the minimum output; in exact-out mode (swap2 with SwapMode 2,
// ExactOut) OutputAmount is the exact output and InputAmount the maximum
// input.
//
// TradeType is BUY (quote -> base), SELL (base -> quote) or SWAP when the
// direction cannot be determined; the input and output mints are then empty.
// The direction comes from the mints of the input/output token accounts when
// the transaction reveals them, otherwise from the payer's associated token
// accounts.
type DBCSwapData struct {
	User               string `json:"user"`
	Pool               string `json:"pool"`
	BaseMint           string `json:"baseMint"`
	QuoteMint          string `json:"quoteMint"`
	InputTokenAccount  string `json:"inputTokenAccount"`
	OutputTokenAccount string `json:"outputTokenAccount"`
	InputAmount        uint64 `json:"inputAmount"`
	OutputAmount       uint64 `json:"outputAmount"`
	TradeType          string `json:"tradeType"`
	InputMint          string `json:"inputMint,omitempty"`
	OutputMint         string `json:"outputMint,omitempty"`
	// SwapMode is the swap2 swap_mode (0 exact in, 1 partial fill, 2 exact out); 0 for swap
	SwapMode uint8 `json:"swapMode"`
	// ExactOut is true in exact-out mode
	ExactOut bool `json:"exactOut,omitempty"`
}

// DBCInitPoolData contains Meteora DBC init pool instruction data
type DBCInitPoolData struct {
	User           string `json:"user"`
	Pool           string `json:"pool"`
	BaseMint       string `json:"baseMint"`
	QuoteMint      string `json:"quoteMint"`
	PlatformConfig string `json:"platformConfig"`
	Name           string `json:"name"`
	Symbol         string `json:"symbol"`
	URI            string `json:"uri"`
}

// DBCMigrateData contains Meteora DBC migrate instruction data
type DBCMigrateData struct {
	BaseMint     string `json:"baseMint"`
	QuoteMint    string `json:"quoteMint"`
	BondingCurve string `json:"bondingCurve"`
	Pool         string `json:"pool"`
	PoolDex      string `json:"poolDex"`
}

// decodeSwapInstruction decodes swap (amount_in, minimum_amount_out) and
// swap2 (amount_0, amount_1, swap_mode). Accounts: 2 pool,
// 3 input_token_account, 4 output_token_account, 7 base_mint, 8 quote_mint,
// 9 payer.
func (p *DBCShredParser) decodeSwapInstruction(accounts []string, data []byte, swap2 bool) *DBCSwapData {
	if len(accounts) < 10 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	amount0, _ := reader.ReadU64()
	amount1, _ := reader.ReadU64()
	var swapMode uint8
	if swap2 {
		swapMode, _ = reader.ReadU8()
	}
	if reader.HasError() {
		return nil
	}
	if swapMode > dbcSwapModeExactOut {
		return nil
	}

	swap := &DBCSwapData{
		User:               accounts[9],
		Pool:               accounts[2],
		BaseMint:           accounts[7],
		QuoteMint:          accounts[8],
		InputTokenAccount:  accounts[3],
		OutputTokenAccount: accounts[4],
		SwapMode:           swapMode,
		ExactOut:           swapMode == dbcSwapModeExactOut,
	}
	if swap.ExactOut {
		swap.OutputAmount, swap.InputAmount = amount0, amount1
	} else {
		swap.InputAmount, swap.OutputAmount = amount0, amount1
	}

	tradeType := p.swapDirection(swap)
	swap.TradeType = string(tradeType)
	switch tradeType {
	case types.TradeTypeBuy:
		swap.InputMint, swap.OutputMint = swap.QuoteMint, swap.BaseMint
	case types.TradeTypeSell:
		swap.InputMint, swap.OutputMint = swap.BaseMint, swap.QuoteMint
	}
	return swap
}

// swapDirection determines BUY (quote in) or SELL (base in) from the mints
// of the swap's token accounts, then from the payer's associated token
// accounts; SWAP when neither decides
func (p *DBCShredParser) swapDirection(swap *DBCSwapData) types.TradeType {
	if swap.BaseMint == "" || swap.QuoteMint == "" {
		return types.TradeTypeSwap
	}
	switch p.adapter.KnownTokenAccountMint(swap.InputTokenAccount) {
	case swap.BaseMint:
		return types.TradeTypeSell
	case swap.QuoteMint:
		return types.TradeTypeBuy
	}
	switch p.adapter.KnownTokenAccountMint(swap.OutputTokenAccount) {
	case swap.BaseMint:
		return types.TradeTypeBuy
	case swap.QuoteMint:
		return types.TradeTypeSell
	}
	if swap.User == "" {
		return types.TradeTypeSwap
	}
	if tradeType := utils.GetAccountTradeType(swap.User, swap.BaseMint, swap.InputTokenAccount, swap.OutputTokenAccount); tradeType != types.TradeTypeSwap {
		return tradeType
	}
	// The payer's quote ATA pays for a buy and receives a sell
	switch utils.GetAccountTradeType(swap.User, swap.QuoteMint, swap.InputTokenAccount, swap.OutputTokenAccount) {
	case types.TradeTypeSell:
		return types.TradeTypeBuy
	case types.TradeTypeBuy:
		return types.TradeTypeSell
	}
	return types.TradeTypeSwap
}

// dbcSwapAmountKinds returns the kinds of a swap's input and output amounts.
// In partial-fill mode the program may take less than amount_0 when the pool
// reaches its migration threshold (EvtSwap2's amount_left), so the input is a
// maximum.
func dbcSwapAmountKinds(swap *DBCSwapData) (types.ShredAmountKind, types.ShredAmountKind) {
	switch {
	case swap.ExactOut:
		return types.ShredAmountMax, types.ShredAmountExact
	case swap.SwapMode == dbcSwapModePartialFill:
		return types.ShredAmountMax, types.ShredAmountMin
	}
	return types.ShredAmountExact, types.ShredAmountMin
}

func (p *DBCShredParser) swapMemeEvent(swap *DBCSwapData) *types.MemeEvent {
	inDecimals, outDecimals := p.adapter.GetTokenDecimals(swap.InputMint), p.adapter.GetTokenDecimals(swap.OutputMint)
	return &types.MemeEvent{
		Type:         types.TradeType(swap.TradeType),
		User:         swap.User,
		BaseMint:     swap.BaseMint,
		QuoteMint:    swap.QuoteMint,
		BondingCurve: swap.Pool,
		Pool:         swap.Pool,
		InputToken: &types.TokenInfo{
			Mint:      swap.InputMint,
			Amount:    types.ConvertToUIAmountUint64(swap.InputAmount, inDecimals),
			AmountRaw: strconv.FormatUint(swap.InputAmount, 10),
			Decimals:  inDecimals,
			Source:    swap.InputTokenAccount,
		},
		OutputToken: &types.TokenInfo{
			Mint:        swap.OutputMint,
			Amount:      types.ConvertToUIAmountUint64(swap.OutputAmount, outDecimals),
			AmountRaw:   strconv.FormatUint(swap.OutputAmount, 10),
			Decimals:    outDecimals,
			Destination: swap.OutputTokenAccount,
		},
	}
}

// decodeInitPool decodes InitializePoolParameters (name, symbol, uri) of the
// initialize_virtual_pool_* instructions: accounts 0 config, 2 creator,
// 3 base_mint, 4 quote_mint, 5 pool
func (p *DBCShredParser) decodeInitPool(accounts []string, data []byte) *DBCInitPoolData {
	if len(accounts) < 6 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	name, err := reader.ReadString()
	if err != nil {
		return nil
	}
	symbol, err := reader.ReadString()
	if err != nil {
		return nil
	}
	uri, err := reader.ReadString()
	if err != nil {
		return nil
	}

	return &DBCInitPoolData{
		User:           accounts[2],
		Pool:           accounts[5],
		BaseMint:       accounts[3],
		QuoteMint:      accounts[4],
		PlatformConfig: accounts[0],
		Name:           name,
		Symbol:         symbol,
		URI:            uri,
	}
}

func dbcShredInitMemeEvent(initData *DBCInitPoolData) *types.MemeEvent {
	return &types.MemeEvent{
		Type:           types.TradeTypeCreate,
		User:           initData.User,
		BaseMint:       initData.BaseMint,
		QuoteMint:      initData.QuoteMint,
		Pool:           initData.Pool,
		BondingCurve:   initData.Pool,
		PlatformConfig: initData.PlatformConfig,
		Name:           initData.Name,
		Symbol:         initData.Symbol,
		URI:            initData.URI,
	}
}

// decodeMigrateDammInstruction decodes migrate_meteora_damm: accounts
// 0 virtual_pool, 4 pool, 7 token_a_mint, 8 token_b_mint
func (p *DBCShredParser) decodeMigrateDammInstruction(accounts []string) *DBCMigrateData {
	if len(accounts) < 10 {
		return nil
	}

	return &DBCMigrateData{
		BaseMint:     accounts[7],
		QuoteMint:    accounts[8],
		BondingCurve: accounts[0],
		Pool:         accounts[4],
		PoolDex:      constants.DEX_PROGRAMS.METEORA_DAMM.Name,
	}
}

// decodeMigrateDammV2Instruction decodes migration_damm_v2: accounts
// 0 virtual_pool, 4 pool, 13 base_mint, 14 quote_mint
func (p *DBCShredParser) decodeMigrateDammV2Instruction(accounts []string) *DBCMigrateData {
	if len(accounts) < 15 {
		return nil
	}

	return &DBCMigrateData{
		BaseMint:     accounts[13],
		QuoteMint:    accounts[14],
		BondingCurve: accounts[0],
		Pool:         accounts[4],
		PoolDex:      constants.DEX_PROGRAMS.METEORA_DAMM_V2.Name,
	}
}

func dbcShredMigrateMemeEvent(migrateData *DBCMigrateData) *types.MemeEvent {
	return &types.MemeEvent{
		Type:         types.TradeTypeMigrate,
		BaseMint:     migrateData.BaseMint,
		QuoteMint:    migrateData.QuoteMint,
		BondingCurve: migrateData.BondingCurve,
		Pool:         migrateData.Pool,
		PoolDex:      migrateData.PoolDex,
	}
}
