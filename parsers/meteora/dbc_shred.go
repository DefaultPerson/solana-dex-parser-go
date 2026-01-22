package meteora

import (
	"bytes"
	"fmt"

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
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.METEORA_DBC.ID)
	return p.parseInstructions(instructions)
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *DBCShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.METEORA_DBC.ID)
	return p.parseTypedInstructions(instructions)
}

func (p *DBCShredParser) parseInstructions(instructions []types.ClassifiedInstruction) []interface{} {
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
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.SWAP):
			eventType = "swap"
			eventData = p.decodeSwapInstruction(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.SWAP_V2):
			eventType = "swap_v2"
			eventData = p.decodeSwapInstruction(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_SPL):
			eventType = "init_pool_spl"
			eventData = p.decodeInitPoolSplInstruction(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022):
			eventType = "init_pool_2022"
			eventData = p.decodeInitPool2022Instruction(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.METEORA_DBC_MIGRATE_DAMM):
			eventType = "migrate_damm"
			eventData = p.decodeMigrateDammInstruction(ci.Instruction)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.METEORA_DBC_MIGRATE_DAMM_V2):
			eventType = "migrate_damm_v2"
			eventData = p.decodeMigrateDammV2Instruction(ci.Instruction)
		default:
			continue
		}

		if eventData != nil {
			event := &DBCShredInstruction{
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

func (p *DBCShredParser) parseTypedInstructions(instructions []types.ClassifiedInstruction) []types.ParsedShredInstruction {
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
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.SWAP):
			eventType = "swap"
			memeEvent = p.decodeSwapMemeEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.SWAP_V2):
			eventType = "swap_v2"
			memeEvent = p.decodeSwapMemeEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_SPL):
			eventType = "init_pool_spl"
			memeEvent = p.decodeInitPoolSplMemeEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022):
			eventType = "init_pool_2022"
			memeEvent = p.decodeInitPool2022MemeEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.METEORA_DBC_MIGRATE_DAMM):
			eventType = "migrate_damm"
			memeEvent = p.decodeMigrateDammMemeEvent(ci.Instruction)
		case bytes.Equal(disc, constants.DISCRIMINATORS.METEORA_DBC.METEORA_DBC_MIGRATE_DAMM_V2):
			eventType = "migrate_damm_v2"
			memeEvent = p.decodeMigrateDammV2MemeEvent(ci.Instruction)
		default:
			continue
		}

		if memeEvent != nil {
			memeEvent.Protocol = constants.DEX_PROGRAMS.METEORA_DBC.Name
			memeEvent.Signature = p.adapter.Signature()
			memeEvent.Slot = p.adapter.Slot()
			memeEvent.Timestamp = p.adapter.BlockTime()
			memeEvent.Idx = idx

			event := types.ParsedShredInstruction{
				ProgramID:   constants.DEX_PROGRAMS.METEORA_DBC.ID,
				ProgramName: constants.DEX_PROGRAMS.METEORA_DBC.Name,
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

// DBCShredInstruction represents a parsed Meteora DBC instruction
type DBCShredInstruction struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Slot      uint64      `json:"slot"`
	Timestamp int64       `json:"timestamp"`
	Signature string      `json:"signature"`
	Idx       string      `json:"idx"`
	Signer    []string    `json:"signer"`
}

// DBCSwapData contains Meteora DBC swap instruction data
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

func (p *DBCShredParser) decodeSwapInstruction(instruction interface{}, data []byte) *DBCSwapData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 10 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	userAccount := accounts[9]
	baseMint := accounts[7]
	quoteMint := accounts[8]
	inputTokenAccount := accounts[3]
	outputTokenAccount := accounts[4]

	tradeType := utils.GetAccountTradeType(p.adapter.Signer(), baseMint, inputTokenAccount, outputTokenAccount)

	return &DBCSwapData{
		User:               userAccount,
		Pool:               accounts[2],
		BaseMint:           baseMint,
		QuoteMint:          quoteMint,
		InputTokenAccount:  inputTokenAccount,
		OutputTokenAccount: outputTokenAccount,
		InputAmount:        inputAmount,
		OutputAmount:       outputAmount,
		TradeType:          string(tradeType),
	}
}

func (p *DBCShredParser) decodeSwapMemeEvent(instruction interface{}, data []byte) *types.MemeEvent {
	swapData := p.decodeSwapInstruction(instruction, data)
	if swapData == nil {
		return nil
	}

	accounts := p.adapter.GetInstructionAccounts(instruction)

	var inputMint, outputMint string
	var tradeType types.TradeType

	if swapData.TradeType == "sell" {
		inputMint = swapData.BaseMint
		outputMint = swapData.QuoteMint
		tradeType = types.TradeTypeSell
	} else {
		inputMint = swapData.QuoteMint
		outputMint = swapData.BaseMint
		tradeType = types.TradeTypeBuy
	}

	return &types.MemeEvent{
		Type:         tradeType,
		User:         swapData.User,
		BaseMint:     swapData.BaseMint,
		QuoteMint:    swapData.QuoteMint,
		BondingCurve: accounts[2],
		Pool:         accounts[2],
		InputToken: &types.TokenInfo{
			Mint:      inputMint,
			Amount:    types.ConvertToUIAmountUint64(swapData.InputAmount, 9),
			AmountRaw: fmt.Sprintf("%d", swapData.InputAmount),
			Decimals:  9,
		},
		OutputToken: &types.TokenInfo{
			Mint:      outputMint,
			Amount:    types.ConvertToUIAmountUint64(swapData.OutputAmount, 9),
			AmountRaw: fmt.Sprintf("%d", swapData.OutputAmount),
			Decimals:  9,
		},
	}
}

func (p *DBCShredParser) decodeInitPoolSplInstruction(instruction interface{}, data []byte) *DBCInitPoolData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 10 {
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

func (p *DBCShredParser) decodeInitPoolSplMemeEvent(instruction interface{}, data []byte) *types.MemeEvent {
	initData := p.decodeInitPoolSplInstruction(instruction, data)
	if initData == nil {
		return nil
	}

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

func (p *DBCShredParser) decodeInitPool2022Instruction(instruction interface{}, data []byte) *DBCInitPoolData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 8 {
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

func (p *DBCShredParser) decodeInitPool2022MemeEvent(instruction interface{}, data []byte) *types.MemeEvent {
	initData := p.decodeInitPool2022Instruction(instruction, data)
	if initData == nil {
		return nil
	}

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

func (p *DBCShredParser) decodeMigrateDammInstruction(instruction interface{}) *DBCMigrateData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
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

func (p *DBCShredParser) decodeMigrateDammMemeEvent(instruction interface{}) *types.MemeEvent {
	migrateData := p.decodeMigrateDammInstruction(instruction)
	if migrateData == nil {
		return nil
	}

	return &types.MemeEvent{
		Type:         types.TradeTypeMigrate,
		BaseMint:     migrateData.BaseMint,
		QuoteMint:    migrateData.QuoteMint,
		BondingCurve: migrateData.BondingCurve,
		Pool:         migrateData.Pool,
		PoolDex:      migrateData.PoolDex,
	}
}

func (p *DBCShredParser) decodeMigrateDammV2Instruction(instruction interface{}) *DBCMigrateData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
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

func (p *DBCShredParser) decodeMigrateDammV2MemeEvent(instruction interface{}) *types.MemeEvent {
	migrateData := p.decodeMigrateDammV2Instruction(instruction)
	if migrateData == nil {
		return nil
	}

	return &types.MemeEvent{
		Type:         types.TradeTypeMigrate,
		BaseMint:     migrateData.BaseMint,
		QuoteMint:    migrateData.QuoteMint,
		BondingCurve: migrateData.BondingCurve,
		Pool:         migrateData.Pool,
		PoolDex:      migrateData.PoolDex,
	}
}
