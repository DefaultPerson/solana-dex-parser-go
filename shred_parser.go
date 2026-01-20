package dexparser

import (
	"fmt"
	"sort"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/classifier"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/types"
	"github.com/solana-dex-parser-go/utils"
)

// ShredInstructionParser interface for instruction parsers
type ShredInstructionParser interface {
	ProcessInstructions() []interface{}
}

// ShredParser parses Solana Shred transactions (pre-execution instruction analysis)
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

// parseWithClassifier parses transaction with specific type
func (p *ShredParser) parseWithClassifier(tx *adapter.SolanaTransaction, config *types.ParseConfig) *types.ParseShredResult {
	if config == nil {
		config = &types.ParseConfig{TryUnknownDEX: true}
	}

	result := &types.ParseShredResult{
		State:        true,
		Signature:    "",
		Instructions: make(map[string][]interface{}),
	}

	defer func() {
		if r := recover(); r != nil {
			if config.ThrowError {
				panic(r)
			}
			sig := ""
			if len(tx.Transaction.Signatures) > 0 {
				sig = tx.Transaction.Signatures[0]
			}
			result.State = false
			result.Msg = fmt.Sprintf("Parse error: %s %v", sig, r)
		}
	}()

	txAdapter := adapter.NewTransactionAdapter(tx, config)
	instructionClassifier := classifier.NewInstructionClassifier(txAdapter)

	allProgramIds := instructionClassifier.GetAllProgramIds()
	result.Signature = txAdapter.Signature()

	// Filter by programIds if specified
	if len(config.ProgramIds) > 0 {
		found := false
		for _, id := range config.ProgramIds {
			for _, pid := range allProgramIds {
				if id == pid {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return result
		}
	}

	// Process instructions for each program
	for _, programId := range allProgramIds {
		// Check programIds filter
		if len(config.ProgramIds) > 0 {
			found := false
			for _, id := range config.ProgramIds {
				if id == programId {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check ignoreProgramIds filter
		if len(config.IgnoreProgramIds) > 0 {
			ignored := false
			for _, id := range config.IgnoreProgramIds {
				if id == programId {
					ignored = true
					break
				}
			}
			if ignored {
				continue
			}
		}

		var parser ShredInstructionParser
		switch programId {
		case constants.DEX_PROGRAMS.PUMP_FUN.ID:
			parser = NewPumpfunInstructionParser(txAdapter, instructionClassifier)
		case constants.DEX_PROGRAMS.PUMP_SWAP.ID:
			parser = NewPumpswapInstructionParser(txAdapter, instructionClassifier)
		}

		if parser != nil {
			programName := utils.GetProgramName(programId)
			result.Instructions[programName] = parser.ProcessInstructions()
		}
	}

	return result
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
}

// PumpfunBuyData contains buy instruction data
type PumpfunBuyData struct {
	Mint         string `json:"mint"`
	BondingCurve string `json:"bondingCurve"`
	TokenAmount  uint64 `json:"tokenAmount"`
	SolAmount    uint64 `json:"solAmount"`
	User         string `json:"user"`
}

// PumpfunSellData contains sell instruction data
type PumpfunSellData struct {
	Mint         string `json:"mint"`
	BondingCurve string `json:"bondingCurve"`
	TokenAmount  uint64 `json:"tokenAmount"`
	SolAmount    uint64 `json:"solAmount"`
	User         string `json:"user"`
}

// PumpfunCreateData contains create instruction data
type PumpfunCreateData struct {
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	URI          string `json:"uri"`
	Mint         string `json:"mint"`
	BondingCurve string `json:"bondingCurve"`
	User         string `json:"user"`
}

// PumpfunMigrateData contains migrate instruction data
type PumpfunMigrateData struct {
	Mint                  string `json:"mint"`
	BondingCurve          string `json:"bondingCurve"`
	User                  string `json:"user"`
	PoolMint              string `json:"poolMint"`
	QuoteMint             string `json:"quoteMint"`
	LpMint                string `json:"lpMint"`
	UserPoolTokenAccount  string `json:"userPoolTokenAccount"`
	PoolBaseTokenAccount  string `json:"poolBaseTokenAccount"`
	PoolQuoteTokenAccount string `json:"poolQuoteTokenAccount"`
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
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.PUMP_FUN.ID)
	return p.parseInstructions(instructions)
}

func (p *PumpfunInstructionParser) parseInstructions(instructions []types.ClassifiedInstruction) []interface{} {
	var events []interface{}

	for _, ci := range instructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		disc := data[:8]
		innerIdx := ci.InnerIndex
		if innerIdx < 0 {
			innerIdx = 0
		}

		var eventType string
		var eventData interface{}

		// Check discriminators
		if bytesEqual(disc, constants.DISCRIMINATORS.PUMPFUN.CREATE) {
			eventType = "CREATE"
			eventData = p.decodeCreateInstruction(ci.Instruction, data[8:])
		} else if bytesEqual(disc, constants.DISCRIMINATORS.PUMPFUN.MIGRATE) {
			eventType = "MIGRATE"
			eventData = p.decodeMigrateInstruction(ci.Instruction)
		} else if bytesEqual(disc, constants.DISCRIMINATORS.PUMPFUN.BUY) {
			eventType = "BUY"
			eventData = p.decodeBuyInstruction(ci.Instruction, data[8:])
		} else if bytesEqual(disc, constants.DISCRIMINATORS.PUMPFUN.SELL) {
			eventType = "SELL"
			eventData = p.decodeSellInstruction(ci.Instruction, data[8:])
		}

		if eventData != nil {
			event := &PumpfunInstruction{
				Type:      eventType,
				Data:      eventData,
				Slot:      p.adapter.Slot(),
				Timestamp: p.adapter.BlockTime(),
				Signature: p.adapter.Signature(),
				Idx:       fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx),
				Signer:    p.adapter.Signers(),
			}
			events = append(events, event)
		}
	}

	// Sort by idx
	sort.Slice(events, func(i, j int) bool {
		ei := events[i].(*PumpfunInstruction)
		ej := events[j].(*PumpfunInstruction)
		return ei.Idx < ej.Idx
	})

	return events
}

func (p *PumpfunInstructionParser) decodeBuyInstruction(instruction interface{}, data []byte) *PumpfunBuyData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
	tokenAmount, _ := reader.ReadU64()
	solAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &PumpfunBuyData{
		Mint:         accounts[2],
		BondingCurve: accounts[3],
		TokenAmount:  tokenAmount,
		SolAmount:    solAmount,
		User:         accounts[6],
	}
}

func (p *PumpfunInstructionParser) decodeSellInstruction(instruction interface{}, data []byte) *PumpfunSellData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
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
	}
}

func (p *PumpfunInstructionParser) decodeCreateInstruction(instruction interface{}, data []byte) *PumpfunCreateData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 8 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
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

	return &PumpfunCreateData{
		Name:         name,
		Symbol:       symbol,
		URI:          uri,
		Mint:         accounts[0],
		BondingCurve: accounts[2],
		User:         accounts[7],
	}
}

func (p *PumpfunInstructionParser) decodeMigrateInstruction(instruction interface{}) *PumpfunMigrateData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 19 {
		return nil
	}

	return &PumpfunMigrateData{
		Mint:                  accounts[2],
		BondingCurve:          accounts[3],
		User:                  accounts[5],
		PoolMint:              accounts[9],
		QuoteMint:             accounts[4],
		LpMint:                accounts[15],
		UserPoolTokenAccount:  accounts[16],
		PoolBaseTokenAccount:  accounts[17],
		PoolQuoteTokenAccount: accounts[18],
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
}

// PumpswapBuyInstructionData contains buy instruction data
type PumpswapBuyInstructionData struct {
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
}

// PumpswapSellInstructionData contains sell instruction data
type PumpswapSellInstructionData struct {
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
}

// PumpswapAddLiquidityData contains add liquidity instruction data
type PumpswapAddLiquidityData struct {
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
}

// PumpswapRemoveLiquidityData contains remove liquidity instruction data
type PumpswapRemoveLiquidityData struct {
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
}

// PumpswapCreatePoolInstructionData contains create pool instruction data
type PumpswapCreatePoolInstructionData struct {
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
	QuoteAmountOut        uint64 `json:"quoteAmountOut"`
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
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.PUMP_SWAP.ID)
	return p.parseInstructions(instructions)
}

func (p *PumpswapInstructionParser) parseInstructions(instructions []types.ClassifiedInstruction) []interface{} {
	var events []interface{}

	for _, ci := range instructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		disc := data[:8]
		innerIdx := ci.InnerIndex
		if innerIdx < 0 {
			innerIdx = 0
		}

		var eventType string
		var eventData interface{}

		// Check discriminators
		if bytesEqual(disc, constants.DISCRIMINATORS.PUMPSWAP.CREATE_POOL) {
			eventType = "CREATE"
			eventData = p.decodeCreateInstruction(ci.Instruction, data[8:])
		} else if bytesEqual(disc, constants.DISCRIMINATORS.PUMPSWAP.ADD_LIQUIDITY) {
			eventType = "ADD"
			eventData = p.decodeAddLiquidityInstruction(ci.Instruction, data[8:])
		} else if bytesEqual(disc, constants.DISCRIMINATORS.PUMPSWAP.REMOVE_LIQUIDITY) {
			eventType = "REMOVE"
			eventData = p.decodeRemoveLiquidityInstruction(ci.Instruction, data[8:])
		} else if bytesEqual(disc, constants.DISCRIMINATORS.PUMPSWAP.BUY) {
			eventType = "BUY"
			eventData = p.decodeBuyInstruction(ci.Instruction, data[8:])
		} else if bytesEqual(disc, constants.DISCRIMINATORS.PUMPSWAP.SELL) {
			eventType = "SELL"
			eventData = p.decodeSellInstruction(ci.Instruction, data[8:])
		}

		if eventData != nil {
			event := &PumpswapInstruction{
				Type:      eventType,
				Data:      eventData,
				Slot:      p.adapter.Slot(),
				Timestamp: p.adapter.BlockTime(),
				Signature: p.adapter.Signature(),
				Idx:       fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx),
				Signer:    p.adapter.Signers(),
			}
			events = append(events, event)
		}
	}

	// Sort by idx
	sort.Slice(events, func(i, j int) bool {
		ei := events[i].(*PumpswapInstruction)
		ej := events[j].(*PumpswapInstruction)
		return ei.Idx < ej.Idx
	})

	return events
}

func (p *PumpswapInstructionParser) decodeBuyInstruction(instruction interface{}, data []byte) *PumpswapBuyInstructionData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 9 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
	baseAmountOut, _ := reader.ReadU64()
	maxQuoteAmountIn, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &PumpswapBuyInstructionData{
		PoolMint:              accounts[0],
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

func (p *PumpswapInstructionParser) decodeSellInstruction(instruction interface{}, data []byte) *PumpswapSellInstructionData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 9 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
	baseAmountIn, _ := reader.ReadU64()
	minQuoteAmountOut, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &PumpswapSellInstructionData{
		PoolMint:              accounts[0],
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

func (p *PumpswapInstructionParser) decodeAddLiquidityInstruction(instruction interface{}, data []byte) *PumpswapAddLiquidityData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
	lpTokenAmountOut, _ := reader.ReadU64()
	maxBaseAmountIn, _ := reader.ReadU64()
	maxQuoteAmountIn, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &PumpswapAddLiquidityData{
		PoolMint:              accounts[0],
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

func (p *PumpswapInstructionParser) decodeRemoveLiquidityInstruction(instruction interface{}, data []byte) *PumpswapRemoveLiquidityData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
	lpTokenAmountIn, _ := reader.ReadU64()
	minBaseAmountOut, _ := reader.ReadU64()
	minQuoteAmountOut, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &PumpswapRemoveLiquidityData{
		PoolMint:              accounts[0],
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

func (p *PumpswapInstructionParser) decodeCreateInstruction(instruction interface{}, data []byte) *PumpswapCreatePoolInstructionData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
	reader.Skip(2) // skip first u16 (index)
	baseAmountIn, _ := reader.ReadU64()
	quoteAmountOut, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &PumpswapCreatePoolInstructionData{
		PoolMint:              accounts[0],
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
		QuoteAmountOut:        quoteAmountOut,
	}
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
