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

// RaydiumV4ShredParser parses Raydium V4 instructions from shred-stream
type RaydiumV4ShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewRaydiumV4ShredParser creates a new RaydiumV4ShredParser
func NewRaydiumV4ShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *RaydiumV4ShredParser {
	return &RaydiumV4ShredParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes Raydium V4 instructions and returns parsed results
func (p *RaydiumV4ShredParser) ProcessInstructions() []interface{} {
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.RAYDIUM_V4.ID)
	return p.parseInstructions(instructions)
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *RaydiumV4ShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.RAYDIUM_V4.ID)
	return p.parseTypedInstructions(instructions)
}

func (p *RaydiumV4ShredParser) parseInstructions(instructions []types.ClassifiedInstruction) []interface{} {
	var events []interface{}

	for _, ci := range instructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 1 {
			continue
		}

		disc := data[:1]
		innerIdx := ci.InnerIndex
		if innerIdx < 0 {
			innerIdx = 0
		}

		var eventType string
		var eventData interface{}

		payload := data[1:]

		switch {
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.SWAP):
			eventType = "swap"
			eventData = p.decodeSwapInstruction(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.CREATE):
			eventType = "create"
			eventData = p.decodeCreateInstruction(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.ADD_LIQUIDITY):
			eventType = "add_liquidity"
			eventData = p.decodeAddLiquidityInstruction(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.REMOVE_LIQUIDITY):
			eventType = "remove_liquidity"
			eventData = p.decodeRemoveLiquidityInstruction(ci.Instruction, payload)
		default:
			continue
		}

		if eventData != nil {
			event := &RaydiumV4ShredInstruction{
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

func (p *RaydiumV4ShredParser) parseTypedInstructions(instructions []types.ClassifiedInstruction) []types.ParsedShredInstruction {
	var events []types.ParsedShredInstruction

	for _, ci := range instructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 1 {
			continue
		}

		disc := data[:1]
		innerIdx := ci.InnerIndex
		if innerIdx < 0 {
			innerIdx = 0
		}

		var eventType string
		var trade *types.TradeInfo
		var liquidity *types.PoolEvent

		payload := data[1:]

		switch {
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.SWAP):
			eventType = "swap"
			trade = p.decodeSwapTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.CREATE):
			eventType = "create"
			liquidity = p.decodeCreatePoolEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.ADD_LIQUIDITY):
			eventType = "add_liquidity"
			liquidity = p.decodeAddLiquidityPoolEvent(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.REMOVE_LIQUIDITY):
			eventType = "remove_liquidity"
			liquidity = p.decodeRemoveLiquidityPoolEvent(ci.Instruction, payload)
		default:
			continue
		}

		if trade != nil || liquidity != nil {
			event := types.ParsedShredInstruction{
				ProgramID:   constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
				ProgramName: constants.DEX_PROGRAMS.RAYDIUM_V4.Name,
				Action:      eventType,
				Trade:       trade,
				Liquidity:   liquidity,
				Accounts:    p.adapter.GetInstructionAccounts(ci.Instruction),
				Idx:         utils.FormatIdx(ci.OuterIndex, innerIdx),
			}
			events = append(events, event)
		}
	}

	return events
}

// RaydiumV4ShredInstruction represents a parsed Raydium V4 instruction
type RaydiumV4ShredInstruction struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Slot      uint64      `json:"slot"`
	Timestamp int64       `json:"timestamp"`
	Signature string      `json:"signature"`
	Idx       string      `json:"idx"`
	Signer    []string    `json:"signer"`
}

// RaydiumV4SwapData contains Raydium V4 swap instruction data
type RaydiumV4SwapData struct {
	Pool               string `json:"pool"`
	User               string `json:"user"`
	InputTokenAccount  string `json:"inputTokenAccount"`
	OutputTokenAccount string `json:"outputTokenAccount"`
	InputAmount        uint64 `json:"inputAmount"`
	OutputAmount       uint64 `json:"outputAmount"`
}

// RaydiumV4LiquidityData contains Raydium V4 liquidity instruction data
type RaydiumV4LiquidityData struct {
	Pool        string `json:"pool"`
	User        string `json:"user"`
	BaseMint    string `json:"baseMint"`
	QuoteMint   string `json:"quoteMint"`
	LpMint      string `json:"lpMint"`
	BaseAmount  uint64 `json:"baseAmount"`
	QuoteAmount uint64 `json:"quoteAmount"`
	LpAmount    uint64 `json:"lpAmount,omitempty"`
}

func (p *RaydiumV4ShredParser) decodeSwapInstruction(instruction interface{}, data []byte) *RaydiumV4SwapData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 18 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &RaydiumV4SwapData{
		Pool:               accounts[1],
		User:               accounts[17],
		InputTokenAccount:  accounts[15],
		OutputTokenAccount: accounts[16],
		InputAmount:        inputAmount,
		OutputAmount:       outputAmount,
	}
}

func (p *RaydiumV4ShredParser) decodeSwapTrade(instruction interface{}, data []byte) *types.TradeInfo {
	swapData := p.decodeSwapInstruction(instruction, data)
	if swapData == nil {
		return nil
	}

	accounts := p.adapter.GetInstructionAccounts(instruction)

	// For Raydium V4 swap, we need to determine direction from accounts
	// Input/output token accounts are at positions 15 and 16
	// We use default decimals since we don't have mint info from instruction

	return &types.TradeInfo{
		Type: types.TradeTypeSwap,
		Pool: []string{swapData.Pool},
		User: swapData.User,
		InputToken: types.TokenInfo{
			Mint:      swapData.InputTokenAccount,
			Amount:    types.ConvertToUIAmountUint64(swapData.InputAmount, 9),
			AmountRaw: fmt.Sprintf("%d", swapData.InputAmount),
			Decimals:  9,
		},
		OutputToken: types.TokenInfo{
			Mint:      swapData.OutputTokenAccount,
			Amount:    types.ConvertToUIAmountUint64(swapData.OutputAmount, 9),
			AmountRaw: fmt.Sprintf("%d", swapData.OutputAmount),
			Decimals:  9,
		},
		ProgramId: constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
		AMMs:      []string{constants.DEX_PROGRAMS.RAYDIUM_V4.Name},
		Extras: map[string]interface{}{
			"amm":          accounts[1],
			"ammAuthority": accounts[2],
			"coinVault":    accounts[5],
			"pcVault":      accounts[6],
		},
	}
}

func (p *RaydiumV4ShredParser) decodeCreateInstruction(instruction interface{}, data []byte) *RaydiumV4LiquidityData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 18 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	reader.ReadU8()  // Skip nonce
	reader.ReadU64() // Skip init pc amount
	reader.ReadU64() // Skip init coin amount
	quoteAmount, _ := reader.ReadU64()
	baseAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &RaydiumV4LiquidityData{
		Pool:        accounts[4],
		User:        accounts[17],
		BaseMint:    accounts[8],
		QuoteMint:   accounts[9],
		LpMint:      accounts[7],
		BaseAmount:  baseAmount,
		QuoteAmount: quoteAmount,
	}
}

func (p *RaydiumV4ShredParser) decodeCreatePoolEvent(instruction interface{}, data []byte) *types.PoolEvent {
	createData := p.decodeCreateInstruction(instruction, data)
	if createData == nil {
		return nil
	}

	baseAmount := types.ConvertToUIAmountUint64(createData.BaseAmount, 9)
	quoteAmount := types.ConvertToUIAmountUint64(createData.QuoteAmount, 9)
	decimals := uint8(9)

	return &types.PoolEvent{
		PoolEventBase: types.PoolEventBase{
			Type:      types.PoolEventTypeCreate,
			ProgramId: constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
			AMM:       constants.DEX_PROGRAMS.RAYDIUM_V4.Name,
			User:      createData.User,
			Slot:      p.adapter.Slot(),
			Timestamp: p.adapter.BlockTime(),
			Signature: p.adapter.Signature(),
		},
		PoolId:          createData.Pool,
		PoolLpMint:      createData.LpMint,
		Token0Mint:      createData.BaseMint,
		Token0Amount:    &baseAmount,
		Token0AmountRaw: fmt.Sprintf("%d", createData.BaseAmount),
		Token0Decimals:  &decimals,
		Token1Mint:      createData.QuoteMint,
		Token1Amount:    &quoteAmount,
		Token1AmountRaw: fmt.Sprintf("%d", createData.QuoteAmount),
		Token1Decimals:  &decimals,
	}
}

func (p *RaydiumV4ShredParser) decodeAddLiquidityInstruction(instruction interface{}, data []byte) *RaydiumV4LiquidityData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 14 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	baseAmount, _ := reader.ReadU64()
	quoteAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &RaydiumV4LiquidityData{
		Pool:        accounts[1],
		User:        accounts[12],
		BaseMint:    accounts[6],
		QuoteMint:   accounts[7],
		LpMint:      accounts[5],
		BaseAmount:  baseAmount,
		QuoteAmount: quoteAmount,
	}
}

func (p *RaydiumV4ShredParser) decodeAddLiquidityPoolEvent(instruction interface{}, data []byte) *types.PoolEvent {
	addData := p.decodeAddLiquidityInstruction(instruction, data)
	if addData == nil {
		return nil
	}

	baseAmount := types.ConvertToUIAmountUint64(addData.BaseAmount, 9)
	quoteAmount := types.ConvertToUIAmountUint64(addData.QuoteAmount, 9)
	decimals := uint8(9)

	return &types.PoolEvent{
		PoolEventBase: types.PoolEventBase{
			Type:      types.PoolEventTypeAdd,
			ProgramId: constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
			AMM:       constants.DEX_PROGRAMS.RAYDIUM_V4.Name,
			User:      addData.User,
			Slot:      p.adapter.Slot(),
			Timestamp: p.adapter.BlockTime(),
			Signature: p.adapter.Signature(),
		},
		PoolId:          addData.Pool,
		PoolLpMint:      addData.LpMint,
		Token0Mint:      addData.BaseMint,
		Token0Amount:    &baseAmount,
		Token0AmountRaw: fmt.Sprintf("%d", addData.BaseAmount),
		Token0Decimals:  &decimals,
		Token1Mint:      addData.QuoteMint,
		Token1Amount:    &quoteAmount,
		Token1AmountRaw: fmt.Sprintf("%d", addData.QuoteAmount),
		Token1Decimals:  &decimals,
	}
}

func (p *RaydiumV4ShredParser) decodeRemoveLiquidityInstruction(instruction interface{}, data []byte) *RaydiumV4LiquidityData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 18 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	lpAmount, err := reader.ReadU64()
	if err != nil || reader.HasError() {
		return nil
	}

	return &RaydiumV4LiquidityData{
		Pool:      accounts[1],
		User:      accounts[16],
		BaseMint:  accounts[6],
		QuoteMint: accounts[7],
		LpMint:    accounts[5],
		LpAmount:  lpAmount,
	}
}

func (p *RaydiumV4ShredParser) decodeRemoveLiquidityPoolEvent(instruction interface{}, data []byte) *types.PoolEvent {
	removeData := p.decodeRemoveLiquidityInstruction(instruction, data)
	if removeData == nil {
		return nil
	}

	lpAmount := types.ConvertToUIAmountUint64(removeData.LpAmount, 6)
	decimals := uint8(9)

	return &types.PoolEvent{
		PoolEventBase: types.PoolEventBase{
			Type:      types.PoolEventTypeRemove,
			ProgramId: constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
			AMM:       constants.DEX_PROGRAMS.RAYDIUM_V4.Name,
			User:      removeData.User,
			Slot:      p.adapter.Slot(),
			Timestamp: p.adapter.BlockTime(),
			Signature: p.adapter.Signature(),
		},
		PoolId:         removeData.Pool,
		PoolLpMint:     removeData.LpMint,
		Token0Mint:     removeData.BaseMint,
		Token0Decimals: &decimals,
		Token1Mint:     removeData.QuoteMint,
		Token1Decimals: &decimals,
		LpAmount:       &lpAmount,
		LpAmountRaw:    fmt.Sprintf("%d", removeData.LpAmount),
	}
}
