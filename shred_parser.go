package dexparser

import (
	"fmt"
	"strconv"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/dflow"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/jupiter"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/meteora"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/photon"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/systoken"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// ShredInstructionParser interface for instruction parsers
type ShredInstructionParser interface {
	ProcessInstructions() []interface{}
}

// shredProgramParser decodes the instructions of one program in a single
// pass: the legacy events (ParseShredResult.Instructions) and the typed
// instructions (ParseShredResult.ParsedInstructions)
type shredProgramParser interface {
	ProcessAll() ([]interface{}, []types.ParsedShredInstruction)
}

// ShredParser decodes DEX instructions from their arguments, without relying
// on execution results.
//
// Its input can be a pre-execution transaction (a ShredStream entry decoded
// with DecodeShredEntries, or any transaction without meta), where only the
// outer instructions exist and CPI calls made by other programs (bots,
// routers) are invisible, or an executed transaction (RPC, or a Yellowstone
// gRPC update converted with ConvertYellowstoneTransaction), where inner
// instructions are decoded too. Amounts are the signed instruction arguments,
// often slippage limits (see types.ParsedShredInstruction's amount kinds);
// DexParser reports executed amounts. Failed transactions (meta.err set) are
// skipped unless ParseConfig.IncludeFailedTxs is set.
type ShredParser struct {
}

// NewShredParser creates a new ShredParser
func NewShredParser() *ShredParser {
	return &ShredParser{}
}

// ParseAll parses both trades and liquidity events from transaction
func (p *ShredParser) ParseAll(tx *adapter.SolanaTransaction, config *types.ParseConfig) *types.ParseShredResult {
	return p.parseWithClassifier(tx, config)
}

// parseWithClassifier decodes the instructions of every supported program.
// It never returns nil: a nil transaction or a recovered panic gives
// State=false with a message (ThrowError re-panics).
func (p *ShredParser) parseWithClassifier(tx *adapter.SolanaTransaction, config *types.ParseConfig) (result *types.ParseShredResult) {
	if config == nil {
		config = &types.ParseConfig{TryUnknownDEX: true}
	}

	result = &types.ParseShredResult{
		State:              true,
		Signature:          "",
		Instructions:       make(map[string][]interface{}),
		ParsedInstructions: make([]types.ParsedShredInstruction, 0),
	}

	defer func() {
		if r := recover(); r != nil {
			if config.ThrowError {
				panic(r)
			}
			result.State = false
			result.Msg = fmt.Sprintf("Parse error: %s %v", txSignature(tx), r)
		}
	}()

	if tx == nil {
		result.State = false
		result.Msg = "nil transaction"
		return result
	}

	txAdapter := adapter.NewTransactionAdapter(tx, config)
	instructionClassifier := classifier.NewInstructionClassifier(txAdapter)

	programIds := shredProgramIds(instructionClassifier)
	result.Signature = txAdapter.Signature()
	result.Slot = txAdapter.Slot()
	result.Timestamp = txAdapter.BlockTime()
	result.Signer = txAdapter.Signers()
	result.TxStatus = txAdapter.TxStatus()
	result.HasUnresolvedAccounts = txAdapter.HasUnresolvedAccounts()
	result.Warnings = txAdapter.Warnings()

	// Filter by programIds if specified
	if len(config.ProgramIds) > 0 {
		found := false
		for _, id := range config.ProgramIds {
			if containsString(programIds, id) {
				found = true
				break
			}
		}
		if !found {
			return result
		}
	}

	// A failed transaction reverted everything its instructions did
	if result.TxStatus == types.TransactionStatusFailed && !config.IncludeFailedTxs {
		result.Msg = "transaction failed"
		return result
	}

	// Process instructions for each program
	for _, programId := range programIds {
		if len(config.ProgramIds) > 0 && !containsString(config.ProgramIds, programId) {
			continue
		}
		if containsString(config.IgnoreProgramIds, programId) {
			continue
		}

		programName, parser := shredParserFor(programId, txAdapter, instructionClassifier)
		if parser == nil {
			continue
		}
		events, typed := parser.ProcessAll()
		if len(events) > 0 {
			result.Instructions[programName] = events
		}
		result.ParsedInstructions = append(result.ParsedInstructions, typed...)
	}

	fillShredContext(txAdapter, result.ParsedInstructions)
	sortByIdx(result.ParsedInstructions, func(ins types.ParsedShredInstruction) string { return ins.Idx })

	return result
}

// shredProgramIds lists the programs of the transaction in first-appearance
// order (outer, then inner instructions), followed by the System, Token and
// Token-2022 programs when they are used (the classifier leaves them out of
// GetAllProgramIds)
func shredProgramIds(ic *classifier.InstructionClassifier) []string {
	programIds := ic.GetAllProgramIds()
	for _, id := range []string{constants.SYSTEM_PROGRAM_ID, constants.TOKEN_PROGRAM_ID, constants.TOKEN_2022_PROGRAM_ID} {
		if ic.HasProgram(id) && !containsString(programIds, id) {
			programIds = append(programIds, id)
		}
	}
	return programIds
}

// shredParserFor returns the result key and the decoder of a program, or a
// nil parser for programs ShredParser does not decode
func shredParserFor(programId string, a *adapter.TransactionAdapter, ic *classifier.InstructionClassifier) (string, shredProgramParser) {
	switch programId {
	case constants.DEX_PROGRAMS.PUMP_FUN.ID:
		return utils.GetProgramName(programId), NewPumpfunInstructionParser(a, ic)
	case constants.DEX_PROGRAMS.PUMP_SWAP.ID:
		return utils.GetProgramName(programId), NewPumpswapInstructionParser(a, ic)
	case constants.DEX_PROGRAMS.PHOTON.ID:
		return "Photon", photon.NewPhotonShredParser(a, ic)
	case constants.SYSTEM_PROGRAM_ID:
		return "System", systoken.NewSystemTokenShredParser(a, ic).ForProgram(programId)
	case constants.TOKEN_PROGRAM_ID:
		return "Token", systoken.NewSystemTokenShredParser(a, ic).ForProgram(programId)
	case constants.TOKEN_2022_PROGRAM_ID:
		return "Token2022", systoken.NewSystemTokenShredParser(a, ic).ForProgram(programId)
	case constants.DEX_PROGRAMS.JUPITER.ID:
		return constants.DEX_PROGRAMS.JUPITER.Name, jupiter.NewJupiterShredParser(a, ic)
	case constants.DEX_PROGRAMS.RAYDIUM_V4.ID:
		return constants.DEX_PROGRAMS.RAYDIUM_V4.Name, raydium.NewRaydiumV4ShredParser(a, ic)
	case constants.DEX_PROGRAMS.RAYDIUM_LCP.ID:
		return constants.DEX_PROGRAMS.RAYDIUM_LCP.Name, raydium.NewLaunchpadShredParser(a, ic)
	case constants.DEX_PROGRAMS.METEORA_DBC.ID:
		return constants.DEX_PROGRAMS.METEORA_DBC.Name, meteora.NewDBCShredParser(a, ic)
	case constants.DEX_PROGRAMS.DFLOW.ID:
		return constants.DEX_PROGRAMS.DFLOW.Name, dflow.NewDFlowShredParser(a, ic)
	}
	return "", nil
}

// fillShredContext completes typed instructions with the transaction context
// (signature, slot, time, idx) and flags unresolved lookup table accounts
func fillShredContext(a *adapter.TransactionAdapter, instructions []types.ParsedShredInstruction) {
	for i := range instructions {
		ins := &instructions[i]
		if types.HasUnresolvedAccount(ins.Accounts) {
			ins.UnresolvedAccounts = true
		}
		if t := ins.Trade; t != nil {
			if t.Signature == "" {
				t.Signature = a.Signature()
			}
			if t.Slot == 0 {
				t.Slot = a.Slot()
			}
			if t.Timestamp == 0 {
				t.Timestamp = a.BlockTime()
			}
			if t.Idx == "" {
				t.Idx = ins.Idx
			}
			if t.Signer == nil {
				t.Signer = a.Signers()
			}
		}
		if l := ins.Liquidity; l != nil {
			if l.Signature == "" {
				l.Signature = a.Signature()
			}
			if l.Slot == 0 {
				l.Slot = a.Slot()
			}
			if l.Timestamp == 0 {
				l.Timestamp = a.BlockTime()
			}
			if l.Idx == "" {
				l.Idx = ins.Idx
			}
		}
		if m := ins.MemeEvent; m != nil {
			if m.Signature == "" {
				m.Signature = a.Signature()
			}
			if m.Slot == 0 {
				m.Slot = a.Slot()
			}
			if m.Timestamp == 0 {
				m.Timestamp = a.BlockTime()
			}
			if m.Idx == "" {
				m.Idx = ins.Idx
			}
		}
	}
}

// shredToken builds a TokenInfo from a raw instruction amount
func shredToken(mint string, amount uint64, decimals uint8) types.TokenInfo {
	return types.TokenInfo{
		Mint:      mint,
		Amount:    types.ConvertToUIAmountUint64(amount, decimals),
		AmountRaw: strconv.FormatUint(amount, 10),
		Decimals:  decimals,
	}
}

// pumpBaseDecimals is the decimals of every Pump.fun bonding-curve mint
const pumpBaseDecimals = 6

// pumpBaseMintDecimals returns the decimals of a Pump.fun bonding-curve mint:
// the known ones (adapter.KnownDecimals), else pumpBaseDecimals
func pumpBaseMintDecimals(a *adapter.TransactionAdapter, mint string) uint8 {
	if decimals, ok := a.KnownDecimals(mint); ok {
		return decimals
	}
	return pumpBaseDecimals
}

// quoteOrSOL returns the quote mint named by an instruction, SOL (WSOL) when
// the instruction has no quote mint account
func quoteOrSOL(quote string, present bool) string {
	if !present {
		return constants.TOKENS.SOL
	}
	return quote
}

// PumpfunInstruction represents a parsed Pumpfun instruction
type PumpfunInstruction struct {
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

// PumpfunBuyData contains buy instruction data. The amounts are instruction
// arguments, not executed amounts: for buy and buy_v2, TokenAmount is the
// exact number of tokens bought and SolAmount the maximum quote cost (a
// slippage limit); for buy_exact_sol_in and buy_exact_quote_in_v2
// (ExactQuoteIn), SolAmount is the exact quote amount spent and TokenAmount
// the minimum number of tokens accepted. SolAmount is in QuoteMint units
// (lamports for SOL curves).
type PumpfunBuyData struct {
	Mint         string `json:"mint"`
	BondingCurve string `json:"bondingCurve"`
	TokenAmount  uint64 `json:"tokenAmount"`
	SolAmount    uint64 `json:"solAmount"`
	User         string `json:"user"`
	// QuoteMint is the curve's quote mint (WSOL for SOL curves)
	QuoteMint string `json:"quoteMint,omitempty"`
	// ExactQuoteIn is true when SolAmount is the exact quote amount spent
	ExactQuoteIn bool `json:"exactQuoteIn,omitempty"`
	// Instruction is the instruction name (buy, buy_exact_sol_in, buy_v2, buy_exact_quote_in_v2)
	Instruction string `json:"instruction,omitempty"`
}

// PumpfunSellData contains sell instruction data: TokenAmount is the exact
// number of tokens sold and SolAmount the minimum quote output (a slippage
// limit), in QuoteMint units
type PumpfunSellData struct {
	Mint         string `json:"mint"`
	BondingCurve string `json:"bondingCurve"`
	TokenAmount  uint64 `json:"tokenAmount"`
	SolAmount    uint64 `json:"solAmount"`
	User         string `json:"user"`
	// QuoteMint is the curve's quote mint (WSOL for SOL curves)
	QuoteMint string `json:"quoteMint,omitempty"`
	// Instruction is the instruction name (sell, sell_v2)
	Instruction string `json:"instruction,omitempty"`
}

// PumpfunCreateData contains create instruction data
type PumpfunCreateData struct {
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	URI          string `json:"uri"`
	Mint         string `json:"mint"`
	BondingCurve string `json:"bondingCurve"`
	User         string `json:"user"`
	// Creator is the creator argument (the coin creator, may differ from User)
	Creator string `json:"creator,omitempty"`
	// QuoteMint is the curve's quote mint: WSOL, or the quote_mint remaining
	// account of create_v2
	QuoteMint string `json:"quoteMint,omitempty"`
	// IsMayhemMode, IsCashbackEnabled, CreatorFeeBps and IsHolderReward are
	// the create_v2 flags (fixed-size OptionBool/OptionU64 values; older
	// create_v2 instructions end after the first ones and leave the rest 0)
	IsMayhemMode      bool   `json:"isMayhemMode,omitempty"`
	IsCashbackEnabled bool   `json:"isCashbackEnabled,omitempty"`
	CreatorFeeBps     uint64 `json:"creatorFeeBps,omitempty"`
	IsHolderReward    bool   `json:"isHolderReward,omitempty"`
	// Instruction is the instruction name (create, create_v2)
	Instruction string `json:"instruction,omitempty"`
}

// PumpfunMigrateData contains migrate instruction data
type PumpfunMigrateData struct {
	Mint         string `json:"mint"`
	BondingCurve string `json:"bondingCurve"`
	User         string `json:"user"`
	// PoolMint holds the PumpSwap pool address (not a mint); same as Pool.
	//
	// Deprecated: use Pool.
	PoolMint              string `json:"poolMint"`
	QuoteMint             string `json:"quoteMint"`
	LpMint                string `json:"lpMint"`
	UserPoolTokenAccount  string `json:"userPoolTokenAccount"`
	PoolBaseTokenAccount  string `json:"poolBaseTokenAccount"`
	PoolQuoteTokenAccount string `json:"poolQuoteTokenAccount"`
	// Pool is the PumpSwap pool created by the migration
	Pool string `json:"pool,omitempty"`
	// Instruction is the instruction name (migrate, migrate_v2)
	Instruction string `json:"instruction,omitempty"`
}

// PumpfunInstructionParser parses Pumpfun instructions
type PumpfunInstructionParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewPumpfunInstructionParser creates a new Pumpfun instruction parser
func NewPumpfunInstructionParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *PumpfunInstructionParser {
	return &PumpfunInstructionParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes all Pumpfun instructions
func (p *PumpfunInstructionParser) ProcessInstructions() []interface{} {
	events, _ := p.ProcessAll()
	return events
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *PumpfunInstructionParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	_, typed := p.ProcessAll()
	return typed
}

// ProcessAll decodes all Pumpfun instructions into legacy events and typed
// instructions, in execution order
func (p *PumpfunInstructionParser) ProcessAll() ([]interface{}, []types.ParsedShredInstruction) {
	var events []interface{}
	var typed []types.ParsedShredInstruction
	disc := constants.DISCRIMINATORS.PUMPFUN

	for _, ci := range p.classifier.GetInstructions(constants.DEX_PROGRAMS.PUMP_FUN.ID) {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}
		accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
		d, args := data[:8], data[8:]

		var eventType string
		var eventData interface{}
		var ins *types.ParsedShredInstruction

		switch {
		case bytesEqual(d, disc.CREATE):
			if c := p.decodeCreate(accounts, args); c != nil {
				eventType, eventData, ins = "CREATE", c, p.createInstruction(c)
			}
		case bytesEqual(d, disc.CREATE_V2):
			if c := p.decodeCreateV2(accounts, args); c != nil {
				eventType, eventData, ins = "CREATE", c, p.createInstruction(c)
			}
		case bytesEqual(d, disc.MIGRATE):
			if m := p.decodeMigrate(accounts); m != nil {
				eventType, eventData, ins = "MIGRATE", m, p.migrateInstruction(m)
			}
		case bytesEqual(d, disc.MIGRATE_V2):
			if m := p.decodeMigrateV2(accounts); m != nil {
				eventType, eventData, ins = "MIGRATE", m, p.migrateInstruction(m)
			}
		case bytesEqual(d, disc.BUY):
			if b := p.decodeBuy(accounts, args, "buy", false); b != nil {
				eventType, eventData, ins = "BUY", b, p.buyInstruction(b)
			}
		case bytesEqual(d, disc.BUY_EXACT_SOL_IN):
			if b := p.decodeBuy(accounts, args, "buy_exact_sol_in", true); b != nil {
				eventType, eventData, ins = "BUY", b, p.buyInstruction(b)
			}
		case bytesEqual(d, disc.BUY_V2):
			if b := p.decodeBuyV2(accounts, args, "buy_v2", false); b != nil {
				eventType, eventData, ins = "BUY", b, p.buyInstruction(b)
			}
		case bytesEqual(d, disc.BUY_EXACT_QUOTE_IN_V2):
			if b := p.decodeBuyV2(accounts, args, "buy_exact_quote_in_v2", true); b != nil {
				eventType, eventData, ins = "BUY", b, p.buyInstruction(b)
			}
		case bytesEqual(d, disc.SELL):
			if s := p.decodeSell(accounts, args); s != nil {
				eventType, eventData, ins = "SELL", s, p.sellInstruction(s)
			}
		case bytesEqual(d, disc.SELL_V2):
			if s := p.decodeSellV2(accounts, args); s != nil {
				eventType, eventData, ins = "SELL", s, p.sellInstruction(s)
			}
		}

		if eventData == nil {
			continue
		}
		idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
		events = append(events, &PumpfunInstruction{
			Type:               eventType,
			Data:               eventData,
			Slot:               p.adapter.Slot(),
			Timestamp:          p.adapter.BlockTime(),
			Signature:          p.adapter.Signature(),
			Idx:                idx,
			Signer:             p.adapter.Signers(),
			UnresolvedAccounts: types.HasUnresolvedAccount(accounts),
		})
		if ins != nil {
			ins.ProgramID = constants.DEX_PROGRAMS.PUMP_FUN.ID
			ins.ProgramName = constants.DEX_PROGRAMS.PUMP_FUN.Name
			ins.Accounts = accounts
			ins.Idx = idx
			typed = append(typed, *ins)
		}
	}

	sortByIdx(events, func(e interface{}) string { return e.(*PumpfunInstruction).Idx })
	return events, typed
}

// decodeBuy decodes buy (amount, max_sol_cost) and buy_exact_sol_in
// (spendable_sol_in, min_tokens_out): accounts 2 mint, 3 bonding_curve, 6 user
func (p *PumpfunInstructionParser) decodeBuy(accounts []string, data []byte, name string, exactIn bool) *PumpfunBuyData {
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	first, _ := reader.ReadU64()
	second, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	buy := &PumpfunBuyData{
		Mint:         accounts[2],
		BondingCurve: accounts[3],
		User:         accounts[6],
		QuoteMint:    constants.TOKENS.SOL,
		ExactQuoteIn: exactIn,
		Instruction:  name,
	}
	if exactIn {
		buy.SolAmount, buy.TokenAmount = first, second
	} else {
		buy.TokenAmount, buy.SolAmount = first, second
	}
	return buy
}

// decodeBuyV2 decodes buy_v2 (amount, max_sol_cost) and buy_exact_quote_in_v2
// (spendable_quote_in, min_tokens_out): accounts 1 base_mint, 2 quote_mint,
// 10 bonding_curve, 13 user
func (p *PumpfunInstructionParser) decodeBuyV2(accounts []string, data []byte, name string, exactIn bool) *PumpfunBuyData {
	if len(accounts) < 14 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	first, _ := reader.ReadU64()
	second, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	buy := &PumpfunBuyData{
		Mint:         accounts[1],
		BondingCurve: accounts[10],
		User:         accounts[13],
		QuoteMint:    accounts[2],
		ExactQuoteIn: exactIn,
		Instruction:  name,
	}
	if exactIn {
		buy.SolAmount, buy.TokenAmount = first, second
	} else {
		buy.TokenAmount, buy.SolAmount = first, second
	}
	return buy
}

// decodeSell decodes sell (amount, min_sol_output): accounts 2 mint,
// 3 bonding_curve, 6 user
func (p *PumpfunInstructionParser) decodeSell(accounts []string, data []byte) *PumpfunSellData {
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	tokenAmount, _ := reader.ReadU64()
	solAmount, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PumpfunSellData{
		Mint:         accounts[2],
		BondingCurve: accounts[3],
		TokenAmount:  tokenAmount,
		SolAmount:    solAmount,
		User:         accounts[6],
		QuoteMint:    constants.TOKENS.SOL,
		Instruction:  "sell",
	}
}

// decodeSellV2 decodes sell_v2 (amount, min_sol_output): accounts
// 1 base_mint, 2 quote_mint, 10 bonding_curve, 13 user
func (p *PumpfunInstructionParser) decodeSellV2(accounts []string, data []byte) *PumpfunSellData {
	if len(accounts) < 14 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	tokenAmount, _ := reader.ReadU64()
	solAmount, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PumpfunSellData{
		Mint:         accounts[1],
		BondingCurve: accounts[10],
		TokenAmount:  tokenAmount,
		SolAmount:    solAmount,
		User:         accounts[13],
		QuoteMint:    accounts[2],
		Instruction:  "sell_v2",
	}
}

// readCreateArgs reads name, symbol, uri and the creator pubkey (absent in
// early create instructions)
func readCreateArgs(reader *utils.BinaryReader) (name, symbol, uri, creator string, ok bool) {
	var err error
	if name, err = reader.ReadString(); err != nil {
		return
	}
	if symbol, err = reader.ReadString(); err != nil {
		return
	}
	if uri, err = reader.ReadString(); err != nil {
		return
	}
	if reader.Remaining() >= 32 {
		creator, _ = reader.ReadPubkey()
	}
	return name, symbol, uri, creator, true
}

// decodeCreate decodes create: accounts 0 mint, 2 bonding_curve, 7 user
func (p *PumpfunInstructionParser) decodeCreate(accounts []string, data []byte) *PumpfunCreateData {
	if len(accounts) < 8 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	name, symbol, uri, creator, ok := readCreateArgs(reader)
	if !ok {
		return nil
	}

	return &PumpfunCreateData{
		Name:         name,
		Symbol:       symbol,
		URI:          uri,
		Mint:         accounts[0],
		BondingCurve: accounts[2],
		User:         accounts[7],
		Creator:      creator,
		QuoteMint:    constants.TOKENS.SOL,
		Instruction:  "create",
	}
}

// decodeCreateV2 decodes create_v2: accounts 0 mint, 2 bonding_curve, 5 user;
// args name, symbol, uri, creator, is_mayhem_mode and the fixed-size
// is_cashback_enabled, creator_fee_bps and is_holder_reward. A non-SOL curve passes quote_mint, associated_quote_bonding_curve and
// quote_token_program as remaining accounts 16-18; without them the quote is
// SOL.
func (p *PumpfunInstructionParser) decodeCreateV2(accounts []string, data []byte) *PumpfunCreateData {
	if len(accounts) < 6 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	name, symbol, uri, creator, ok := readCreateArgs(reader)
	if !ok {
		return nil
	}
	create := &PumpfunCreateData{
		Name:         name,
		Symbol:       symbol,
		URI:          uri,
		Mint:         accounts[0],
		BondingCurve: accounts[2],
		User:         accounts[5],
		Creator:      creator,
		QuoteMint:    quoteOrSOL(accountAt(accounts, 16), len(accounts) >= 19),
		Instruction:  "create_v2",
	}
	if reader.Remaining() >= 1 {
		create.IsMayhemMode, _ = reader.ReadBool()
	}
	if reader.Remaining() >= 1 {
		create.IsCashbackEnabled, _ = reader.ReadBool()
	}
	if reader.Remaining() >= 8 {
		create.CreatorFeeBps, _ = reader.ReadU64()
	}
	if reader.Remaining() >= 1 {
		create.IsHolderReward, _ = reader.ReadBool()
	}
	return create
}

// decodeMigrate decodes migrate: accounts 2 mint, 3 bonding_curve, 5 user,
// 9 pool, 14 wsol_mint, 15 lp_mint, 16-18 pool token accounts
func (p *PumpfunInstructionParser) decodeMigrate(accounts []string) *PumpfunMigrateData {
	if len(accounts) < 19 {
		return nil
	}

	return &PumpfunMigrateData{
		Mint:                  accounts[2],
		BondingCurve:          accounts[3],
		User:                  accounts[5],
		PoolMint:              accounts[9],
		Pool:                  accounts[9],
		QuoteMint:             accounts[14],
		LpMint:                accounts[15],
		UserPoolTokenAccount:  accounts[16],
		PoolBaseTokenAccount:  accounts[17],
		PoolQuoteTokenAccount: accounts[18],
		Instruction:           "migrate",
	}
}

// decodeMigrateV2 decodes migrate_v2: accounts 2 base_mint, 3 quote_mint,
// 4 bonding_curve, 7 user, 10 pool, 15 lp_mint, 16-18 pool token accounts
func (p *PumpfunInstructionParser) decodeMigrateV2(accounts []string) *PumpfunMigrateData {
	if len(accounts) < 19 {
		return nil
	}

	return &PumpfunMigrateData{
		Mint:                  accounts[2],
		BondingCurve:          accounts[4],
		User:                  accounts[7],
		PoolMint:              accounts[10],
		Pool:                  accounts[10],
		QuoteMint:             accounts[3],
		LpMint:                accounts[15],
		UserPoolTokenAccount:  accounts[16],
		PoolBaseTokenAccount:  accounts[17],
		PoolQuoteTokenAccount: accounts[18],
		Instruction:           "migrate_v2",
	}
}

func (p *PumpfunInstructionParser) buyInstruction(b *PumpfunBuyData) *types.ParsedShredInstruction {
	input := shredToken(b.QuoteMint, b.SolAmount, p.adapter.GetTokenDecimals(b.QuoteMint))
	output := shredToken(b.Mint, b.TokenAmount, pumpBaseMintDecimals(p.adapter, b.Mint))
	inKind, outKind := types.ShredAmountMax, types.ShredAmountExact
	if b.ExactQuoteIn {
		inKind, outKind = types.ShredAmountExact, types.ShredAmountMin
	}
	return p.tradeInstruction("buy", types.TradeTypeBuy, b.User, b.BondingCurve, b.Mint, b.QuoteMint, input, output, inKind, outKind)
}

func (p *PumpfunInstructionParser) sellInstruction(s *PumpfunSellData) *types.ParsedShredInstruction {
	input := shredToken(s.Mint, s.TokenAmount, pumpBaseMintDecimals(p.adapter, s.Mint))
	output := shredToken(s.QuoteMint, s.SolAmount, p.adapter.GetTokenDecimals(s.QuoteMint))
	return p.tradeInstruction("sell", types.TradeTypeSell, s.User, s.BondingCurve, s.Mint, s.QuoteMint, input, output, types.ShredAmountExact, types.ShredAmountMin)
}

// tradeInstruction reports a Pump.fun trade both as a Trade and as a
// MemeEvent, as DexParser does
func (p *PumpfunInstructionParser) tradeInstruction(action string, tradeType types.TradeType, user, curve, baseMint, quoteMint string, input, output types.TokenInfo, inKind, outKind types.ShredAmountKind) *types.ParsedShredInstruction {
	in, out := input, output
	return &types.ParsedShredInstruction{
		Action: action,
		Trade: &types.TradeInfo{
			Type:        tradeType,
			Pool:        []string{curve},
			User:        user,
			InputToken:  input,
			OutputToken: output,
			ProgramId:   constants.DEX_PROGRAMS.PUMP_FUN.ID,
			AMM:         constants.DEX_PROGRAMS.PUMP_FUN.Name,
		},
		MemeEvent: &types.MemeEvent{
			Protocol:     constants.DEX_PROGRAMS.PUMP_FUN.Name,
			Type:         tradeType,
			User:         user,
			BaseMint:     baseMint,
			QuoteMint:    quoteMint,
			BondingCurve: curve,
			Pool:         curve,
			InputToken:   &in,
			OutputToken:  &out,
		},
		InputAmountKind:  inKind,
		OutputAmountKind: outKind,
	}
}

func (p *PumpfunInstructionParser) createInstruction(c *PumpfunCreateData) *types.ParsedShredInstruction {
	decimals := uint8(pumpBaseDecimals)
	return &types.ParsedShredInstruction{
		Action: "create",
		MemeEvent: &types.MemeEvent{
			Protocol:     constants.DEX_PROGRAMS.PUMP_FUN.Name,
			Type:         types.TradeTypeCreate,
			User:         c.User,
			BaseMint:     c.Mint,
			QuoteMint:    c.QuoteMint,
			Name:         c.Name,
			Symbol:       c.Symbol,
			URI:          c.URI,
			Decimals:     &decimals,
			Creator:      c.Creator,
			BondingCurve: c.BondingCurve,
			Pool:         c.BondingCurve,
		},
	}
}

func (p *PumpfunInstructionParser) migrateInstruction(m *PumpfunMigrateData) *types.ParsedShredInstruction {
	return &types.ParsedShredInstruction{
		Action: "migrate",
		MemeEvent: &types.MemeEvent{
			Protocol:     constants.DEX_PROGRAMS.PUMP_FUN.Name,
			Type:         types.TradeTypeMigrate,
			User:         m.User,
			BaseMint:     m.Mint,
			QuoteMint:    m.QuoteMint,
			BondingCurve: m.BondingCurve,
			Pool:         m.Pool,
			PoolDex:      constants.DEX_PROGRAMS.PUMP_SWAP.Name,
		},
	}
}

// PumpswapInstruction represents a parsed Pumpswap instruction
type PumpswapInstruction struct {
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

// PumpswapBuyInstructionData contains buy instruction data: BaseAmountOut is
// the exact base amount bought and MaxQuoteAmountIn the maximum quote cost (a
// slippage limit)
type PumpswapBuyInstructionData struct {
	// PoolMint holds the pool address (not a mint); same as Pool.
	//
	// Deprecated: use Pool.
	PoolMint              string `json:"poolMint"`
	User                  string `json:"user"`
	BaseMint              string `json:"baseMint"`
	QuoteMint             string `json:"quoteMint"`
	UserBaseTokenAccount  string `json:"userBaseTokenAccount"`
	UserQuoteTokenAccount string `json:"userQuoteTokenAccount"`
	PoolBaseTokenAccount  string `json:"poolBaseTokenAccount"`
	PoolQuoteTokenAccount string `json:"poolQuoteTokenAccount"`
	BaseAmountOut         uint64 `json:"baseAmountOut"`
	MaxQuoteAmountIn      uint64 `json:"maxQuoteAmountIn"`
	// Pool is the pool address
	Pool string `json:"pool,omitempty"`
}

// PumpswapBuyExactQuoteInData contains buy_exact_quote_in instruction data:
// SpendableQuoteIn is the exact quote amount spent and MinBaseAmountOut the
// minimum base amount accepted (a slippage limit)
type PumpswapBuyExactQuoteInData struct {
	Pool                  string `json:"pool"`
	User                  string `json:"user"`
	BaseMint              string `json:"baseMint"`
	QuoteMint             string `json:"quoteMint"`
	UserBaseTokenAccount  string `json:"userBaseTokenAccount"`
	UserQuoteTokenAccount string `json:"userQuoteTokenAccount"`
	PoolBaseTokenAccount  string `json:"poolBaseTokenAccount"`
	PoolQuoteTokenAccount string `json:"poolQuoteTokenAccount"`
	SpendableQuoteIn      uint64 `json:"spendableQuoteIn"`
	MinBaseAmountOut      uint64 `json:"minBaseAmountOut"`
}

// PumpswapSellInstructionData contains sell instruction data: BaseAmountIn is
// the exact base amount sold and MinQuoteAmountOut the minimum quote output
// (a slippage limit)
type PumpswapSellInstructionData struct {
	// PoolMint holds the pool address (not a mint); same as Pool.
	//
	// Deprecated: use Pool.
	PoolMint              string `json:"poolMint"`
	User                  string `json:"user"`
	BaseMint              string `json:"baseMint"`
	QuoteMint             string `json:"quoteMint"`
	UserBaseTokenAccount  string `json:"userBaseTokenAccount"`
	UserQuoteTokenAccount string `json:"userQuoteTokenAccount"`
	PoolBaseTokenAccount  string `json:"poolBaseTokenAccount"`
	PoolQuoteTokenAccount string `json:"poolQuoteTokenAccount"`
	BaseAmountIn          uint64 `json:"baseAmountIn"`
	MinQuoteAmountOut     uint64 `json:"minQuoteAmountOut"`
	// Pool is the pool address
	Pool string `json:"pool,omitempty"`
}

// PumpswapAddLiquidityData contains deposit instruction data: LpTokenAmountOut
// is exact, the base and quote amounts are maximums
type PumpswapAddLiquidityData struct {
	// PoolMint holds the pool address (not a mint); same as Pool.
	//
	// Deprecated: use Pool.
	PoolMint              string `json:"poolMint"`
	User                  string `json:"user"`
	BaseMint              string `json:"baseMint"`
	QuoteMint             string `json:"quoteMint"`
	LpMint                string `json:"lpMint"`
	UserBaseTokenAccount  string `json:"userBaseTokenAccount"`
	UserQuoteTokenAccount string `json:"userQuoteTokenAccount"`
	UserPoolTokenAccount  string `json:"userPoolTokenAccount"`
	PoolBaseTokenAccount  string `json:"poolBaseTokenAccount"`
	PoolQuoteTokenAccount string `json:"poolQuoteTokenAccount"`
	LpTokenAmountOut      uint64 `json:"lpTokenAmountOut"`
	MaxBaseAmountIn       uint64 `json:"maxBaseAmountIn"`
	MaxQuoteAmountIn      uint64 `json:"maxQuoteAmountIn"`
	// Pool is the pool address
	Pool string `json:"pool,omitempty"`
}

// PumpswapRemoveLiquidityData contains withdraw instruction data:
// LpTokenAmountIn is exact, the base and quote amounts are minimums
type PumpswapRemoveLiquidityData struct {
	// PoolMint holds the pool address (not a mint); same as Pool.
	//
	// Deprecated: use Pool.
	PoolMint              string `json:"poolMint"`
	User                  string `json:"user"`
	BaseMint              string `json:"baseMint"`
	QuoteMint             string `json:"quoteMint"`
	LpMint                string `json:"lpMint"`
	UserBaseTokenAccount  string `json:"userBaseTokenAccount"`
	UserQuoteTokenAccount string `json:"userQuoteTokenAccount"`
	UserPoolTokenAccount  string `json:"userPoolTokenAccount"`
	PoolBaseTokenAccount  string `json:"poolBaseTokenAccount"`
	PoolQuoteTokenAccount string `json:"poolQuoteTokenAccount"`
	LpTokenAmountIn       uint64 `json:"lpTokenAmountIn"`
	MinBaseAmountOut      uint64 `json:"minBaseAmountOut"`
	MinQuoteAmountOut     uint64 `json:"minQuoteAmountOut"`
	// Pool is the pool address
	Pool string `json:"pool,omitempty"`
}

// PumpswapCreatePoolInstructionData contains create_pool instruction data;
// BaseAmountIn and QuoteAmountIn are the exact initial reserves
type PumpswapCreatePoolInstructionData struct {
	// PoolMint holds the pool address (not a mint); same as Pool.
	//
	// Deprecated: use Pool.
	PoolMint              string `json:"poolMint"`
	User                  string `json:"user"`
	BaseMint              string `json:"baseMint"`
	QuoteMint             string `json:"quoteMint"`
	LpMint                string `json:"lpMint"`
	UserBaseTokenAccount  string `json:"userBaseTokenAccount"`
	UserQuoteTokenAccount string `json:"userQuoteTokenAccount"`
	UserPoolTokenAccount  string `json:"userPoolTokenAccount"`
	PoolBaseTokenAccount  string `json:"poolBaseTokenAccount"`
	PoolQuoteTokenAccount string `json:"poolQuoteTokenAccount"`
	BaseAmountIn          uint64 `json:"baseAmountIn"`
	// QuoteAmountOut holds the quote_amount_in argument; same as QuoteAmountIn.
	//
	// Deprecated: use QuoteAmountIn.
	QuoteAmountOut uint64 `json:"quoteAmountOut"`
	// QuoteAmountIn is the initial quote reserve deposited by the creator
	QuoteAmountIn uint64 `json:"quoteAmountIn"`
	// Pool is the pool address
	Pool string `json:"pool,omitempty"`
}

// PumpswapInstructionParser parses Pumpswap instructions
type PumpswapInstructionParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewPumpswapInstructionParser creates a new Pumpswap instruction parser
func NewPumpswapInstructionParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *PumpswapInstructionParser {
	return &PumpswapInstructionParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes all Pumpswap instructions
func (p *PumpswapInstructionParser) ProcessInstructions() []interface{} {
	events, _ := p.ProcessAll()
	return events
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *PumpswapInstructionParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	_, typed := p.ProcessAll()
	return typed
}

// ProcessAll decodes all Pumpswap instructions into legacy events and typed
// instructions, in execution order
func (p *PumpswapInstructionParser) ProcessAll() ([]interface{}, []types.ParsedShredInstruction) {
	var events []interface{}
	var typed []types.ParsedShredInstruction
	disc := constants.DISCRIMINATORS.PUMPSWAP

	for _, ci := range p.classifier.GetInstructions(constants.DEX_PROGRAMS.PUMP_SWAP.ID) {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}
		accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
		d, args := data[:8], data[8:]

		var eventType string
		var eventData interface{}
		var ins *types.ParsedShredInstruction

		switch {
		case bytesEqual(d, disc.CREATE_POOL):
			if c := p.decodeCreate(accounts, args); c != nil {
				eventType, eventData = "CREATE", c
				ins = p.liquidityInstruction("create", types.PoolEventTypeCreate, c.User, c.Pool, c.LpMint, c.BaseMint, c.QuoteMint, c.BaseAmountIn, c.QuoteAmountIn, 0, types.ShredAmountExact, types.ShredAmountUnknown)
			}
		case bytesEqual(d, disc.ADD_LIQUIDITY):
			if a := p.decodeAddLiquidity(accounts, args); a != nil {
				eventType, eventData = "ADD", a
				ins = p.liquidityInstruction("add_liquidity", types.PoolEventTypeAdd, a.User, a.Pool, a.LpMint, a.BaseMint, a.QuoteMint, a.MaxBaseAmountIn, a.MaxQuoteAmountIn, a.LpTokenAmountOut, types.ShredAmountMax, types.ShredAmountExact)
			}
		case bytesEqual(d, disc.REMOVE_LIQUIDITY):
			if r := p.decodeRemoveLiquidity(accounts, args); r != nil {
				eventType, eventData = "REMOVE", r
				ins = p.liquidityInstruction("remove_liquidity", types.PoolEventTypeRemove, r.User, r.Pool, r.LpMint, r.BaseMint, r.QuoteMint, r.MinBaseAmountOut, r.MinQuoteAmountOut, r.LpTokenAmountIn, types.ShredAmountExact, types.ShredAmountMin)
			}
		case bytesEqual(d, disc.BUY):
			if b := p.decodeBuy(accounts, args); b != nil {
				eventType, eventData = "BUY", b
				ins = p.tradeInstruction("buy", types.TradeTypeBuy, b.User, b.Pool, b.QuoteMint, b.MaxQuoteAmountIn, b.BaseMint, b.BaseAmountOut, types.ShredAmountMax, types.ShredAmountExact)
			}
		case bytesEqual(d, disc.BUY_EXACT_QUOTE_IN):
			if b := p.decodeBuyExactQuoteIn(accounts, args); b != nil {
				eventType, eventData = "BUY", b
				ins = p.tradeInstruction("buy_exact_quote_in", types.TradeTypeBuy, b.User, b.Pool, b.QuoteMint, b.SpendableQuoteIn, b.BaseMint, b.MinBaseAmountOut, types.ShredAmountExact, types.ShredAmountMin)
			}
		case bytesEqual(d, disc.SELL):
			if s := p.decodeSell(accounts, args); s != nil {
				eventType, eventData = "SELL", s
				ins = p.tradeInstruction("sell", types.TradeTypeSell, s.User, s.Pool, s.BaseMint, s.BaseAmountIn, s.QuoteMint, s.MinQuoteAmountOut, types.ShredAmountExact, types.ShredAmountMin)
			}
		}

		if eventData == nil {
			continue
		}
		idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
		events = append(events, &PumpswapInstruction{
			Type:               eventType,
			Data:               eventData,
			Slot:               p.adapter.Slot(),
			Timestamp:          p.adapter.BlockTime(),
			Signature:          p.adapter.Signature(),
			Idx:                idx,
			Signer:             p.adapter.Signers(),
			UnresolvedAccounts: types.HasUnresolvedAccount(accounts),
		})
		ins.ProgramID = constants.DEX_PROGRAMS.PUMP_SWAP.ID
		ins.ProgramName = constants.DEX_PROGRAMS.PUMP_SWAP.Name
		ins.Accounts = accounts
		ins.Idx = idx
		typed = append(typed, *ins)
	}

	sortByIdx(events, func(e interface{}) string { return e.(*PumpswapInstruction).Idx })
	return events, typed
}

func (p *PumpswapInstructionParser) tradeInstruction(action string, tradeType types.TradeType, user, pool, inMint string, inAmount uint64, outMint string, outAmount uint64, inKind, outKind types.ShredAmountKind) *types.ParsedShredInstruction {
	return &types.ParsedShredInstruction{
		Action: action,
		Trade: &types.TradeInfo{
			Type:        tradeType,
			Pool:        []string{pool},
			User:        user,
			InputToken:  shredToken(inMint, inAmount, p.adapter.GetTokenDecimals(inMint)),
			OutputToken: shredToken(outMint, outAmount, p.adapter.GetTokenDecimals(outMint)),
			ProgramId:   constants.DEX_PROGRAMS.PUMP_SWAP.ID,
			AMM:         constants.DEX_PROGRAMS.PUMP_SWAP.Name,
		},
		InputAmountKind:  inKind,
		OutputAmountKind: outKind,
	}
}

// liquidityInstruction builds a PoolEvent; for liquidity the input side is
// what the user deposits (base/quote for CREATE and ADD, LP for REMOVE)
func (p *PumpswapInstructionParser) liquidityInstruction(action string, eventType types.PoolEventType, user, pool, lpMint, baseMint, quoteMint string, baseAmount, quoteAmount, lpAmount uint64, inKind, outKind types.ShredAmountKind) *types.ParsedShredInstruction {
	base := shredToken(baseMint, baseAmount, p.adapter.GetTokenDecimals(baseMint))
	quote := shredToken(quoteMint, quoteAmount, p.adapter.GetTokenDecimals(quoteMint))
	event := &types.PoolEvent{
		PoolEventBase: types.PoolEventBase{
			User:      user,
			Type:      eventType,
			ProgramId: constants.DEX_PROGRAMS.PUMP_SWAP.ID,
			AMM:       constants.DEX_PROGRAMS.PUMP_SWAP.Name,
		},
		PoolId:          pool,
		PoolLpMint:      lpMint,
		Token0Mint:      baseMint,
		Token0Amount:    &base.Amount,
		Token0AmountRaw: base.AmountRaw,
		Token0Decimals:  &base.Decimals,
		Token1Mint:      quoteMint,
		Token1Amount:    &quote.Amount,
		Token1AmountRaw: quote.AmountRaw,
		Token1Decimals:  &quote.Decimals,
	}
	if eventType != types.PoolEventTypeCreate {
		lp := types.ConvertToUIAmountUint64(lpAmount, p.adapter.GetTokenDecimals(lpMint))
		event.LpAmount = &lp
		event.LpAmountRaw = strconv.FormatUint(lpAmount, 10)
	}
	return &types.ParsedShredInstruction{
		Action:           action,
		Liquidity:        event,
		InputAmountKind:  inKind,
		OutputAmountKind: outKind,
	}
}

// decodeBuy decodes buy (base_amount_out, max_quote_amount_in): accounts
// 0 pool, 1 user, 3 base_mint, 4 quote_mint, 5-8 token accounts
func (p *PumpswapInstructionParser) decodeBuy(accounts []string, data []byte) *PumpswapBuyInstructionData {
	if len(accounts) < 9 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	baseAmountOut, _ := reader.ReadU64()
	maxQuoteAmountIn, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PumpswapBuyInstructionData{
		PoolMint:              accounts[0],
		Pool:                  accounts[0],
		User:                  accounts[1],
		BaseMint:              accounts[3],
		QuoteMint:             accounts[4],
		UserBaseTokenAccount:  accounts[5],
		UserQuoteTokenAccount: accounts[6],
		PoolBaseTokenAccount:  accounts[7],
		PoolQuoteTokenAccount: accounts[8],
		BaseAmountOut:         baseAmountOut,
		MaxQuoteAmountIn:      maxQuoteAmountIn,
	}
}

// decodeBuyExactQuoteIn decodes buy_exact_quote_in (spendable_quote_in,
// min_base_amount_out) with the buy account layout
func (p *PumpswapInstructionParser) decodeBuyExactQuoteIn(accounts []string, data []byte) *PumpswapBuyExactQuoteInData {
	if len(accounts) < 9 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	spendableQuoteIn, _ := reader.ReadU64()
	minBaseAmountOut, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PumpswapBuyExactQuoteInData{
		Pool:                  accounts[0],
		User:                  accounts[1],
		BaseMint:              accounts[3],
		QuoteMint:             accounts[4],
		UserBaseTokenAccount:  accounts[5],
		UserQuoteTokenAccount: accounts[6],
		PoolBaseTokenAccount:  accounts[7],
		PoolQuoteTokenAccount: accounts[8],
		SpendableQuoteIn:      spendableQuoteIn,
		MinBaseAmountOut:      minBaseAmountOut,
	}
}

// decodeSell decodes sell (base_amount_in, min_quote_amount_out) with the buy
// account layout
func (p *PumpswapInstructionParser) decodeSell(accounts []string, data []byte) *PumpswapSellInstructionData {
	if len(accounts) < 9 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	baseAmountIn, _ := reader.ReadU64()
	minQuoteAmountOut, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PumpswapSellInstructionData{
		PoolMint:              accounts[0],
		Pool:                  accounts[0],
		User:                  accounts[1],
		BaseMint:              accounts[3],
		QuoteMint:             accounts[4],
		UserBaseTokenAccount:  accounts[5],
		UserQuoteTokenAccount: accounts[6],
		PoolBaseTokenAccount:  accounts[7],
		PoolQuoteTokenAccount: accounts[8],
		BaseAmountIn:          baseAmountIn,
		MinQuoteAmountOut:     minQuoteAmountOut,
	}
}

// decodeAddLiquidity decodes deposit (lp_token_amount_out,
// max_base_amount_in, max_quote_amount_in): accounts 0 pool, 2 user,
// 3 base_mint, 4 quote_mint, 5 lp_mint, 6-10 token accounts
func (p *PumpswapInstructionParser) decodeAddLiquidity(accounts []string, data []byte) *PumpswapAddLiquidityData {
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	lpTokenAmountOut, _ := reader.ReadU64()
	maxBaseAmountIn, _ := reader.ReadU64()
	maxQuoteAmountIn, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PumpswapAddLiquidityData{
		PoolMint:              accounts[0],
		Pool:                  accounts[0],
		User:                  accounts[2],
		BaseMint:              accounts[3],
		QuoteMint:             accounts[4],
		LpMint:                accounts[5],
		UserBaseTokenAccount:  accounts[6],
		UserQuoteTokenAccount: accounts[7],
		UserPoolTokenAccount:  accounts[8],
		PoolBaseTokenAccount:  accounts[9],
		PoolQuoteTokenAccount: accounts[10],
		LpTokenAmountOut:      lpTokenAmountOut,
		MaxBaseAmountIn:       maxBaseAmountIn,
		MaxQuoteAmountIn:      maxQuoteAmountIn,
	}
}

// decodeRemoveLiquidity decodes withdraw (lp_token_amount_in,
// min_base_amount_out, min_quote_amount_out) with the deposit account layout
func (p *PumpswapInstructionParser) decodeRemoveLiquidity(accounts []string, data []byte) *PumpswapRemoveLiquidityData {
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	lpTokenAmountIn, _ := reader.ReadU64()
	minBaseAmountOut, _ := reader.ReadU64()
	minQuoteAmountOut, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PumpswapRemoveLiquidityData{
		PoolMint:              accounts[0],
		Pool:                  accounts[0],
		User:                  accounts[2],
		BaseMint:              accounts[3],
		QuoteMint:             accounts[4],
		LpMint:                accounts[5],
		UserBaseTokenAccount:  accounts[6],
		UserQuoteTokenAccount: accounts[7],
		UserPoolTokenAccount:  accounts[8],
		PoolBaseTokenAccount:  accounts[9],
		PoolQuoteTokenAccount: accounts[10],
		LpTokenAmountIn:       lpTokenAmountIn,
		MinBaseAmountOut:      minBaseAmountOut,
		MinQuoteAmountOut:     minQuoteAmountOut,
	}
}

// decodeCreate decodes create_pool (index u16, base_amount_in,
// quote_amount_in, ...): accounts 0 pool, 2 creator, 3 base_mint,
// 4 quote_mint, 5 lp_mint, 6-10 token accounts
func (p *PumpswapInstructionParser) decodeCreate(accounts []string, data []byte) *PumpswapCreatePoolInstructionData {
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	reader.Skip(2) // index u16
	baseAmountIn, _ := reader.ReadU64()
	quoteAmountIn, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PumpswapCreatePoolInstructionData{
		PoolMint:              accounts[0],
		Pool:                  accounts[0],
		User:                  accounts[2],
		BaseMint:              accounts[3],
		QuoteMint:             accounts[4],
		LpMint:                accounts[5],
		UserBaseTokenAccount:  accounts[6],
		UserQuoteTokenAccount: accounts[7],
		UserPoolTokenAccount:  accounts[8],
		PoolBaseTokenAccount:  accounts[9],
		PoolQuoteTokenAccount: accounts[10],
		BaseAmountIn:          baseAmountIn,
		QuoteAmountOut:        quoteAmountIn,
		QuoteAmountIn:         quoteAmountIn,
	}
}

// accountAt returns accounts[i], or "" when the instruction has fewer accounts
func accountAt(accounts []string, i int) string {
	if i >= 0 && i < len(accounts) {
		return accounts[i]
	}
	return ""
}

// bytesEqual compares two byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
