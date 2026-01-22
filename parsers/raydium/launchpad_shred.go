package raydium

import (
	"bytes"
	"fmt"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// LaunchpadShredParser parses Raydium Launchpad (LCP) instructions from shred-stream
type LaunchpadShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewLaunchpadShredParser creates a new LaunchpadShredParser
func NewLaunchpadShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *LaunchpadShredParser {
	return &LaunchpadShredParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes Raydium LCP instructions and returns parsed results
func (p *LaunchpadShredParser) ProcessInstructions() []interface{} {
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.RAYDIUM_LCP.ID)
	return p.parseInstructions(instructions)
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *LaunchpadShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.RAYDIUM_LCP.ID)
	return p.parseTypedInstructions(instructions)
}

func (p *LaunchpadShredParser) parseInstructions(instructions []types.ClassifiedInstruction) []interface{} {
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

		payload := data[8:]

		switch {
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.INITIALIZE):
			eventType = "create"
			eventData = p.decodeCreateInstruction(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.BUY_EXACT_IN):
			eventType = "buy_exact_in"
			eventData = p.decodeBuyExactIn(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.BUY_EXACT_OUT):
			eventType = "buy_exact_out"
			eventData = p.decodeBuyExactOut(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.SELL_EXACT_IN):
			eventType = "sell_exact_in"
			eventData = p.decodeSellExactIn(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.SELL_EXACT_OUT):
			eventType = "sell_exact_out"
			eventData = p.decodeSellExactOut(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.MIGRATE_TO_AMM):
			eventType = "migrate_to_amm"
			eventData = p.decodeMigrateToAMM(ci.Instruction)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.MIGRATE_TO_CPSWAP):
			eventType = "migrate_to_cpswap"
			eventData = p.decodeMigrateToCPSwap(ci.Instruction)
		default:
			continue
		}

		if eventData != nil {
			event := &LaunchpadShredInstruction{
				Type:      eventType,
				Data:      eventData,
				Slot:      p.adapter.Slot(),
				Timestamp: p.adapter.BlockTime(),
				Signature: p.adapter.Signature(),
				Idx:       utils.FormatIdx(ci.OuterIndex, innerIdx),
				Signer:    p.adapter.Signers(),
			}
			events = append(events, event)
		}
	}

	return events
}

func (p *LaunchpadShredParser) parseTypedInstructions(instructions []types.ClassifiedInstruction) []types.ParsedShredInstruction {
	var events []types.ParsedShredInstruction

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
		var memeEvent *types.MemeEvent

		payload := data[8:]
		idx := utils.FormatIdx(ci.OuterIndex, innerIdx)

		switch {
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.INITIALIZE):
			eventType = "create"
			memeEvent = p.decodeCreateMemeEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.BUY_EXACT_IN):
			eventType = "buy_exact_in"
			memeEvent = p.decodeBuyExactInMemeEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.BUY_EXACT_OUT):
			eventType = "buy_exact_out"
			memeEvent = p.decodeBuyExactOutMemeEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.SELL_EXACT_IN):
			eventType = "sell_exact_in"
			memeEvent = p.decodeSellExactInMemeEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.SELL_EXACT_OUT):
			eventType = "sell_exact_out"
			memeEvent = p.decodeSellExactOutMemeEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.MIGRATE_TO_AMM):
			eventType = "migrate_to_amm"
			memeEvent = p.decodeMigrateToAMMMemeEvent(ci.Instruction)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.MIGRATE_TO_CPSWAP):
			eventType = "migrate_to_cpswap"
			memeEvent = p.decodeMigrateToCPSwapMemeEvent(ci.Instruction)
		default:
			continue
		}

		if memeEvent != nil {
			memeEvent.Signature = p.adapter.Signature()
			memeEvent.Slot = p.adapter.Slot()
			memeEvent.Timestamp = p.adapter.BlockTime()
			memeEvent.Idx = idx

			event := types.ParsedShredInstruction{
				ProgramID:   constants.DEX_PROGRAMS.RAYDIUM_LCP.ID,
				ProgramName: constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
				Action:      eventType,
				MemeEvent:   memeEvent,
				Accounts:    p.adapter.GetInstructionAccounts(ci.Instruction),
				Idx:         idx,
			}
			events = append(events, event)
		}
	}

	return events
}

// LaunchpadShredInstruction represents a parsed Raydium LCP instruction
type LaunchpadShredInstruction struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Slot      uint64      `json:"slot"`
	Timestamp int64       `json:"timestamp"`
	Signature string      `json:"signature"`
	Idx       string      `json:"idx"`
	Signer    []string    `json:"signer"`
}

// LaunchpadCreateData contains Raydium LCP create instruction data
type LaunchpadCreateData struct {
	User      string `json:"user"`
	Pool      string `json:"pool"`
	BaseMint  string `json:"baseMint"`
	QuoteMint string `json:"quoteMint"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	URI       string `json:"uri"`
}

// LaunchpadTradeData contains Raydium LCP trade instruction data
type LaunchpadTradeData struct {
	User           string `json:"user"`
	Pool           string `json:"pool"`
	BaseMint       string `json:"baseMint"`
	QuoteMint      string `json:"quoteMint"`
	InputMint      string `json:"inputMint"`
	OutputMint     string `json:"outputMint"`
	InputAmount    uint64 `json:"inputAmount"`
	OutputAmount   uint64 `json:"outputAmount"`
	TradeType      string `json:"tradeType"`
	PlatformConfig string `json:"platformConfig"`
}

// LaunchpadMigrateData contains Raydium LCP migrate instruction data
type LaunchpadMigrateData struct {
	BaseMint  string `json:"baseMint"`
	QuoteMint string `json:"quoteMint"`
	Pool      string `json:"pool"`
	PoolDex   string `json:"poolDex"`
}

func (p *LaunchpadShredParser) decodeCreateInstruction(instruction interface{}, data []byte) *LaunchpadCreateData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 8 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	// Read MintParams
	reader.ReadU8() // decimals
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

	return &LaunchpadCreateData{
		User:      accounts[1],
		Pool:      accounts[5],
		BaseMint:  accounts[6],
		QuoteMint: accounts[7],
		Name:      name,
		Symbol:    symbol,
		URI:       uri,
	}
}

func (p *LaunchpadShredParser) decodeCreateMemeEvent(instruction interface{}, data []byte) *types.MemeEvent {
	createData := p.decodeCreateInstruction(instruction, data)
	if createData == nil {
		return nil
	}

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:         types.TradeTypeCreate,
		User:         createData.User,
		BaseMint:     createData.BaseMint,
		QuoteMint:    createData.QuoteMint,
		Pool:         createData.Pool,
		BondingCurve: createData.Pool,
		Name:         createData.Name,
		Symbol:       createData.Symbol,
		URI:          createData.URI,
	}
}

func (p *LaunchpadShredParser) decodeBuyExactIn(instruction interface{}, data []byte) *LaunchpadTradeData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &LaunchpadTradeData{
		User:           accounts[0],
		Pool:           accounts[4],
		PlatformConfig: accounts[3],
		BaseMint:       accounts[9],
		QuoteMint:      accounts[10],
		InputMint:      accounts[10], // quoteMint
		OutputMint:     accounts[9],  // baseMint
		InputAmount:    inputAmount,
		OutputAmount:   outputAmount,
		TradeType:      "buy",
	}
}

func (p *LaunchpadShredParser) decodeBuyExactInMemeEvent(instruction interface{}, data []byte) *types.MemeEvent {
	tradeData := p.decodeBuyExactIn(instruction, data)
	if tradeData == nil {
		return nil
	}

	return p.buildMemeEvent(tradeData, types.TradeTypeBuy)
}

func (p *LaunchpadShredParser) decodeBuyExactOut(instruction interface{}, data []byte) *LaunchpadTradeData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	outputAmount, _ := reader.ReadU64()
	inputAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &LaunchpadTradeData{
		User:           accounts[0],
		Pool:           accounts[4],
		PlatformConfig: accounts[3],
		BaseMint:       accounts[9],
		QuoteMint:      accounts[10],
		InputMint:      accounts[10], // quoteMint
		OutputMint:     accounts[9],  // baseMint
		InputAmount:    inputAmount,
		OutputAmount:   outputAmount,
		TradeType:      "buy",
	}
}

func (p *LaunchpadShredParser) decodeBuyExactOutMemeEvent(instruction interface{}, data []byte) *types.MemeEvent {
	tradeData := p.decodeBuyExactOut(instruction, data)
	if tradeData == nil {
		return nil
	}

	return p.buildMemeEvent(tradeData, types.TradeTypeBuy)
}

func (p *LaunchpadShredParser) decodeSellExactIn(instruction interface{}, data []byte) *LaunchpadTradeData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &LaunchpadTradeData{
		User:           accounts[0],
		Pool:           accounts[4],
		PlatformConfig: accounts[3],
		BaseMint:       accounts[9],
		QuoteMint:      accounts[10],
		InputMint:      accounts[9],  // baseMint
		OutputMint:     accounts[10], // quoteMint
		InputAmount:    inputAmount,
		OutputAmount:   outputAmount,
		TradeType:      "sell",
	}
}

func (p *LaunchpadShredParser) decodeSellExactInMemeEvent(instruction interface{}, data []byte) *types.MemeEvent {
	tradeData := p.decodeSellExactIn(instruction, data)
	if tradeData == nil {
		return nil
	}

	return p.buildMemeEvent(tradeData, types.TradeTypeSell)
}

func (p *LaunchpadShredParser) decodeSellExactOut(instruction interface{}, data []byte) *LaunchpadTradeData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	outputAmount, _ := reader.ReadU64()
	inputAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &LaunchpadTradeData{
		User:           accounts[0],
		Pool:           accounts[4],
		PlatformConfig: accounts[3],
		BaseMint:       accounts[9],
		QuoteMint:      accounts[10],
		InputMint:      accounts[9],  // baseMint
		OutputMint:     accounts[10], // quoteMint
		InputAmount:    inputAmount,
		OutputAmount:   outputAmount,
		TradeType:      "sell",
	}
}

func (p *LaunchpadShredParser) decodeSellExactOutMemeEvent(instruction interface{}, data []byte) *types.MemeEvent {
	tradeData := p.decodeSellExactOut(instruction, data)
	if tradeData == nil {
		return nil
	}

	return p.buildMemeEvent(tradeData, types.TradeTypeSell)
}

func (p *LaunchpadShredParser) decodeMigrateToAMM(instruction interface{}) *LaunchpadMigrateData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 17 {
		return nil
	}

	return &LaunchpadMigrateData{
		BaseMint:  accounts[1],
		QuoteMint: accounts[2],
		Pool:      accounts[13],
		PoolDex:   constants.DEX_PROGRAMS.RAYDIUM_V4.Name,
	}
}

func (p *LaunchpadShredParser) decodeMigrateToAMMMemeEvent(instruction interface{}) *types.MemeEvent {
	migrateData := p.decodeMigrateToAMM(instruction)
	if migrateData == nil {
		return nil
	}

	return &types.MemeEvent{
		Protocol:  constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:      types.TradeTypeMigrate,
		BaseMint:  migrateData.BaseMint,
		QuoteMint: migrateData.QuoteMint,
		Pool:      migrateData.Pool,
		PoolDex:   migrateData.PoolDex,
	}
}

func (p *LaunchpadShredParser) decodeMigrateToCPSwap(instruction interface{}) *LaunchpadMigrateData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 17 {
		return nil
	}

	return &LaunchpadMigrateData{
		BaseMint:  accounts[1],
		QuoteMint: accounts[2],
		Pool:      accounts[5],
		PoolDex:   constants.DEX_PROGRAMS.RAYDIUM_CPMM.Name,
	}
}

func (p *LaunchpadShredParser) decodeMigrateToCPSwapMemeEvent(instruction interface{}) *types.MemeEvent {
	migrateData := p.decodeMigrateToCPSwap(instruction)
	if migrateData == nil {
		return nil
	}

	return &types.MemeEvent{
		Protocol:  constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:      types.TradeTypeMigrate,
		BaseMint:  migrateData.BaseMint,
		QuoteMint: migrateData.QuoteMint,
		Pool:      migrateData.Pool,
		PoolDex:   migrateData.PoolDex,
	}
}

func (p *LaunchpadShredParser) buildMemeEvent(data *LaunchpadTradeData, tradeType types.TradeType) *types.MemeEvent {
	var inputDecimal, outputDecimal uint8
	if tradeType == types.TradeTypeBuy {
		inputDecimal, outputDecimal = 9, 6
	} else {
		inputDecimal, outputDecimal = 6, 9
	}

	return &types.MemeEvent{
		Protocol:       constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:           tradeType,
		User:           data.User,
		BaseMint:       data.InputMint,
		QuoteMint:      data.OutputMint,
		BondingCurve:   data.Pool,
		Pool:           data.Pool,
		PlatformConfig: data.PlatformConfig,
		InputToken: &types.TokenInfo{
			Mint:      data.InputMint,
			Amount:    types.ConvertToUIAmountUint64(data.InputAmount, inputDecimal),
			AmountRaw: fmt.Sprintf("%d", data.InputAmount),
			Decimals:  inputDecimal,
		},
		OutputToken: &types.TokenInfo{
			Mint:      data.OutputMint,
			Amount:    types.ConvertToUIAmountUint64(data.OutputAmount, outputDecimal),
			AmountRaw: fmt.Sprintf("%d", data.OutputAmount),
			Decimals:  outputDecimal,
		},
	}
}
