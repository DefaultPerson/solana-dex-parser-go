package photon

import (
	"bytes"
	"fmt"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// PhotonShredParser parses Photon instructions from shred-stream
type PhotonShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewPhotonShredParser creates a new PhotonShredParser
func NewPhotonShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *PhotonShredParser {
	return &PhotonShredParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes Photon instructions and returns parsed results
func (p *PhotonShredParser) ProcessInstructions() []interface{} {
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.PHOTON.ID)
	return p.parseInstructions(instructions)
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *PhotonShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.PHOTON.ID)
	return p.parseTypedInstructions(instructions)
}

func (p *PhotonShredParser) parseInstructions(instructions []types.ClassifiedInstruction) []interface{} {
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
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.PUMPSWAP_TRADE):
			eventType = "pumpswap_swap"
			eventData = p.decodePhotonSwapData(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.PUMPFUN_BUY):
			eventType = "pumpfun_buy"
			eventData = p.decodePhotonPumpfunBuyData(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.PUMPFUN_SELL):
			eventType = "pumpfun_sell"
			eventData = p.decodePhotonPumpfunSellData(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.MOONIT_BUY):
			eventType = "moonit_buy"
			eventData = p.decodePhotonMoonitBuyData(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.MOONIT_SELL):
			eventType = "moonit_sell"
			eventData = p.decodePhotonMoonitSellData(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.HOP_TWO_SWAP):
			eventType = "hop_two_swap"
			eventData = p.decodePhotonHopTwoSwapData(ci.Instruction, payload)
		default:
			continue
		}

		if eventData != nil {
			event := &PhotonInstruction{
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

func (p *PhotonShredParser) parseTypedInstructions(instructions []types.ClassifiedInstruction) []types.ParsedShredInstruction {
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
		var trade *types.TradeInfo

		payload := data[8:]

		switch {
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.PUMPSWAP_TRADE):
			eventType = "pumpswap_swap"
			trade = p.decodePhotonSwapTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.PUMPFUN_BUY):
			eventType = "pumpfun_buy"
			trade = p.decodePhotonPumpfunBuyTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.PUMPFUN_SELL):
			eventType = "pumpfun_sell"
			trade = p.decodePhotonPumpfunSellTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.MOONIT_BUY):
			eventType = "moonit_buy"
			trade = p.decodePhotonMoonitBuyTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.MOONIT_SELL):
			eventType = "moonit_sell"
			trade = p.decodePhotonMoonitSellTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.PHOTON.HOP_TWO_SWAP):
			eventType = "hop_two_swap"
			trade = p.decodePhotonHopTwoSwapTrade(ci.Instruction, payload)
		default:
			continue
		}

		if trade != nil {
			event := types.ParsedShredInstruction{
				ProgramID:   constants.DEX_PROGRAMS.PHOTON.ID,
				ProgramName: constants.DEX_PROGRAMS.PHOTON.Name,
				Action:      eventType,
				Trade:       trade,
				Accounts:    p.adapter.GetInstructionAccounts(ci.Instruction),
				Idx:         utils.FormatIdx(ci.OuterIndex, innerIdx),
			}
			events = append(events, event)
		}
	}

	return events
}

// PhotonInstruction represents a parsed Photon instruction
type PhotonInstruction struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Slot      uint64      `json:"slot"`
	Timestamp int64       `json:"timestamp"`
	Signature string      `json:"signature"`
	Idx       string      `json:"idx"`
	Signer    []string    `json:"signer"`
}

// PhotonSwapData contains Photon swap instruction data
type PhotonSwapData struct {
	Pool               string `json:"pool"`
	User               string `json:"user"`
	BaseMint           string `json:"baseMint"`
	QuoteMint          string `json:"quoteMint"`
	InputTokenAccount  string `json:"inputTokenAccount"`
	OutputTokenAccount string `json:"outputTokenAccount"`
	InputAmount        uint64 `json:"inputAmount"`
	OutputAmount       uint64 `json:"outputAmount"`
	TradeType          string `json:"tradeType"`
	TargetProgram      string `json:"targetProgram"`
}

// PhotonPumpfunData contains Photon Pumpfun instruction data
type PhotonPumpfunData struct {
	Pool          string `json:"pool"`
	User          string `json:"user"`
	BaseMint      string `json:"baseMint"`
	Timestamp     int64  `json:"timestamp"`
	InputAmount   uint64 `json:"inputAmount"`
	OutputAmount  uint64 `json:"outputAmount"`
	TradeType     string `json:"tradeType"`
	TargetProgram string `json:"targetProgram"`
}

// PhotonMoonitData contains Photon Moonit instruction data
type PhotonMoonitData struct {
	Pool          string `json:"pool"`
	User          string `json:"user"`
	BaseMint      string `json:"baseMint"`
	Timestamp     int64  `json:"timestamp"`
	InputAmount   uint64 `json:"inputAmount"`
	OutputAmount  uint64 `json:"outputAmount"`
	SlippageBps   uint64 `json:"slippageBps"`
	TradeType     string `json:"tradeType"`
	TargetProgram string `json:"targetProgram"`
}

// PhotonHopTwoSwapData contains Photon hop two swap instruction data
type PhotonHopTwoSwapData struct {
	User         string   `json:"user"`
	Pools        []string `json:"pools"`
	InputMint    string   `json:"inputMint"`
	OutputMint   string   `json:"outputMint"`
	InputAmount  uint64   `json:"inputAmount"`
	OutputAmount uint64   `json:"outputAmount"`
	TradeType    string   `json:"tradeType"`
	Programs     []string `json:"programs"`
}

func (p *PhotonShredParser) decodePhotonSwapData(instruction interface{}, data []byte) *PhotonSwapData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 17 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	userAccount := accounts[1]
	baseMint := accounts[3]
	quoteMint := accounts[4]
	inputTokenAccount := accounts[5]
	outputTokenAccount := accounts[6]

	tradeType := utils.GetAccountTradeType(userAccount, baseMint, inputTokenAccount, outputTokenAccount)

	return &PhotonSwapData{
		Pool:               accounts[0],
		User:               userAccount,
		BaseMint:           baseMint,
		QuoteMint:          quoteMint,
		InputTokenAccount:  inputTokenAccount,
		OutputTokenAccount: outputTokenAccount,
		InputAmount:        inputAmount,
		OutputAmount:       outputAmount,
		TradeType:          string(tradeType),
		TargetProgram:      accounts[16],
	}
}

func (p *PhotonShredParser) decodePhotonSwapTrade(instruction interface{}, data []byte) *types.TradeInfo {
	swapData := p.decodePhotonSwapData(instruction, data)
	if swapData == nil {
		return nil
	}

	accounts := p.adapter.GetInstructionAccounts(instruction)

	var inputMint, outputMint string
	var inputDecimal, outputDecimal uint8 = 6, 9
	var tradeType types.TradeType

	if swapData.TradeType == "sell" {
		inputMint = swapData.BaseMint
		outputMint = swapData.QuoteMint
		tradeType = types.TradeTypeSell
	} else {
		inputMint = swapData.QuoteMint
		outputMint = swapData.BaseMint
		inputDecimal, outputDecimal = 9, 6
		tradeType = types.TradeTypeBuy
	}

	return &types.TradeInfo{
		Type: tradeType,
		Pool: []string{swapData.Pool},
		User: swapData.User,
		InputToken: types.TokenInfo{
			Mint:      inputMint,
			Amount:    types.ConvertToUIAmountUint64(swapData.InputAmount, inputDecimal),
			AmountRaw: fmt.Sprintf("%d", swapData.InputAmount),
			Decimals:  inputDecimal,
		},
		OutputToken: types.TokenInfo{
			Mint:      outputMint,
			Amount:    types.ConvertToUIAmountUint64(swapData.OutputAmount, outputDecimal),
			AmountRaw: fmt.Sprintf("%d", swapData.OutputAmount),
			Decimals:  outputDecimal,
		},
		ProgramId: accounts[16],
		AMMs:      []string{utils.GetProgramName(accounts[16])},
		Route:     constants.DEX_PROGRAMS.PHOTON.Name,
	}
}

func (p *PhotonShredParser) decodePhotonPumpfunSellData(instruction interface{}, data []byte) *PhotonPumpfunData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 12 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	timestamp, _ := reader.ReadI64()
	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &PhotonPumpfunData{
		Pool:          accounts[3],
		User:          accounts[6],
		BaseMint:      accounts[2],
		Timestamp:     timestamp,
		InputAmount:   inputAmount,
		OutputAmount:  outputAmount,
		TradeType:     "sell",
		TargetProgram: accounts[9],
	}
}

func (p *PhotonShredParser) decodePhotonPumpfunSellTrade(instruction interface{}, data []byte) *types.TradeInfo {
	sellData := p.decodePhotonPumpfunSellData(instruction, data)
	if sellData == nil {
		return nil
	}

	return &types.TradeInfo{
		Type: types.TradeTypeSell,
		Pool: []string{sellData.Pool},
		User: sellData.User,
		InputToken: types.TokenInfo{
			Mint:      sellData.BaseMint,
			Amount:    types.ConvertToUIAmountUint64(sellData.InputAmount, 6),
			AmountRaw: fmt.Sprintf("%d", sellData.InputAmount),
			Decimals:  6,
		},
		OutputToken: types.TokenInfo{
			Mint:      constants.TOKENS.SOL,
			Amount:    types.ConvertToUIAmountUint64(sellData.OutputAmount, 9),
			AmountRaw: fmt.Sprintf("%d", sellData.OutputAmount),
			Decimals:  9,
		},
		ProgramId: sellData.TargetProgram,
		AMMs:      []string{utils.GetProgramName(sellData.TargetProgram)},
		Route:     constants.DEX_PROGRAMS.PHOTON.Name,
		Timestamp: sellData.Timestamp,
	}
}

func (p *PhotonShredParser) decodePhotonPumpfunBuyData(instruction interface{}, data []byte) *PhotonPumpfunData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 12 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	timestamp, _ := reader.ReadI64()
	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &PhotonPumpfunData{
		Pool:          accounts[4],
		User:          accounts[7],
		BaseMint:      accounts[3],
		Timestamp:     timestamp,
		InputAmount:   inputAmount,
		OutputAmount:  outputAmount,
		TradeType:     "buy",
		TargetProgram: accounts[9],
	}
}

func (p *PhotonShredParser) decodePhotonPumpfunBuyTrade(instruction interface{}, data []byte) *types.TradeInfo {
	buyData := p.decodePhotonPumpfunBuyData(instruction, data)
	if buyData == nil {
		return nil
	}

	return &types.TradeInfo{
		Type: types.TradeTypeBuy,
		Pool: []string{buyData.Pool},
		User: buyData.User,
		InputToken: types.TokenInfo{
			Mint:      constants.TOKENS.SOL,
			Amount:    types.ConvertToUIAmountUint64(buyData.InputAmount, 9),
			AmountRaw: fmt.Sprintf("%d", buyData.InputAmount),
			Decimals:  9,
		},
		OutputToken: types.TokenInfo{
			Mint:      buyData.BaseMint,
			Amount:    types.ConvertToUIAmountUint64(buyData.OutputAmount, 6),
			AmountRaw: fmt.Sprintf("%d", buyData.OutputAmount),
			Decimals:  6,
		},
		ProgramId: buyData.TargetProgram,
		AMMs:      []string{utils.GetProgramName(buyData.TargetProgram)},
		Route:     constants.DEX_PROGRAMS.PHOTON.Name,
		Timestamp: buyData.Timestamp,
	}
}

func (p *PhotonShredParser) decodePhotonMoonitBuyData(instruction interface{}, data []byte) *PhotonMoonitData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 12 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	timestamp, _ := reader.ReadI64()
	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()
	slippageBps, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &PhotonMoonitData{
		Pool:          accounts[2],
		User:          accounts[0],
		BaseMint:      accounts[7],
		Timestamp:     timestamp,
		InputAmount:   inputAmount,
		OutputAmount:  outputAmount,
		SlippageBps:   slippageBps,
		TradeType:     "buy",
		TargetProgram: accounts[9],
	}
}

func (p *PhotonShredParser) decodePhotonMoonitBuyTrade(instruction interface{}, data []byte) *types.TradeInfo {
	buyData := p.decodePhotonMoonitBuyData(instruction, data)
	if buyData == nil {
		return nil
	}

	slippageBps := int(buyData.SlippageBps)

	return &types.TradeInfo{
		Type: types.TradeTypeBuy,
		Pool: []string{buyData.Pool},
		User: buyData.User,
		InputToken: types.TokenInfo{
			Mint:      buyData.BaseMint,
			Amount:    types.ConvertToUIAmountUint64(buyData.InputAmount, 9),
			AmountRaw: fmt.Sprintf("%d", buyData.InputAmount),
			Decimals:  9,
		},
		OutputToken: types.TokenInfo{
			Mint:      constants.TOKENS.SOL,
			Amount:    types.ConvertToUIAmountUint64(buyData.OutputAmount, 6),
			AmountRaw: fmt.Sprintf("%d", buyData.OutputAmount),
			Decimals:  6,
		},
		ProgramId:   buyData.TargetProgram,
		AMMs:        []string{utils.GetProgramName(buyData.TargetProgram)},
		Route:       constants.DEX_PROGRAMS.PHOTON.Name,
		Timestamp:   buyData.Timestamp,
		SlippageBps: &slippageBps,
	}
}

func (p *PhotonShredParser) decodePhotonMoonitSellData(instruction interface{}, data []byte) *PhotonMoonitData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 12 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	timestamp, _ := reader.ReadI64()
	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()
	slippageBps, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	return &PhotonMoonitData{
		Pool:          accounts[2],
		User:          accounts[0],
		BaseMint:      accounts[7],
		Timestamp:     timestamp,
		InputAmount:   inputAmount,
		OutputAmount:  outputAmount,
		SlippageBps:   slippageBps,
		TradeType:     "sell",
		TargetProgram: accounts[9],
	}
}

func (p *PhotonShredParser) decodePhotonMoonitSellTrade(instruction interface{}, data []byte) *types.TradeInfo {
	sellData := p.decodePhotonMoonitSellData(instruction, data)
	if sellData == nil {
		return nil
	}

	slippageBps := int(sellData.SlippageBps)

	return &types.TradeInfo{
		Type: types.TradeTypeSell,
		Pool: []string{sellData.Pool},
		User: sellData.User,
		InputToken: types.TokenInfo{
			Mint:      sellData.BaseMint,
			Amount:    types.ConvertToUIAmountUint64(sellData.InputAmount, 6),
			AmountRaw: fmt.Sprintf("%d", sellData.InputAmount),
			Decimals:  6,
		},
		OutputToken: types.TokenInfo{
			Mint:      constants.TOKENS.SOL,
			Amount:    types.ConvertToUIAmountUint64(sellData.OutputAmount, 9),
			AmountRaw: fmt.Sprintf("%d", sellData.OutputAmount),
			Decimals:  9,
		},
		ProgramId:   sellData.TargetProgram,
		AMMs:        []string{utils.GetProgramName(sellData.TargetProgram)},
		Route:       constants.DEX_PROGRAMS.PHOTON.Name,
		Timestamp:   sellData.Timestamp,
		SlippageBps: &slippageBps,
	}
}

func (p *PhotonShredParser) decodePhotonHopTwoSwapData(instruction interface{}, data []byte) *PhotonHopTwoSwapData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 17 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()

	if reader.HasError() {
		return nil
	}

	userAccount := accounts[0]
	program1 := accounts[6]

	var inputMint, outputMint string
	var pools []string
	var programs []string
	var tradeType string

	// Raydium V4 -> Meteora (Buy)
	if program1 == constants.DEX_PROGRAMS.RAYDIUM_V4.ID {
		tradeType = "buy"
		program2 := accounts[11]
		programs = []string{utils.GetProgramName(program1), utils.GetProgramName(program2)}

		switch program2 {
		case constants.DEX_PROGRAMS.METEORA_DBC.ID:
			pools = []string{accounts[7], accounts[14]}
			inputMint = constants.TOKENS.SOL
			outputMint = accounts[17]
		case constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID:
			pools = []string{accounts[7], accounts[13]}
			inputMint = constants.TOKENS.SOL
			outputMint = accounts[16]
		case constants.DEX_PROGRAMS.METEORA.ID:
			pools = []string{accounts[7], accounts[12]}
			inputMint = constants.TOKENS.SOL
			outputMint = accounts[16]
		default:
			return nil
		}
	} else if program1 == constants.DEX_PROGRAMS.METEORA_DBC.ID {
		// Meteora DBC -> Raydium V4 (Sell)
		tradeType = "sell"
		pools = []string{accounts[9], accounts[17]}
		inputMint = accounts[12]
		outputMint = constants.TOKENS.SOL
		program2 := accounts[16]
		programs = []string{utils.GetProgramName(program1), utils.GetProgramName(program2)}
	} else {
		return nil
	}

	return &PhotonHopTwoSwapData{
		User:         userAccount,
		Pools:        pools,
		InputMint:    inputMint,
		OutputMint:   outputMint,
		InputAmount:  inputAmount,
		OutputAmount: outputAmount,
		TradeType:    tradeType,
		Programs:     programs,
	}
}

func (p *PhotonShredParser) decodePhotonHopTwoSwapTrade(instruction interface{}, data []byte) *types.TradeInfo {
	swapData := p.decodePhotonHopTwoSwapData(instruction, data)
	if swapData == nil {
		return nil
	}

	var inputDecimal, outputDecimal uint8
	var tradeType types.TradeType
	if swapData.TradeType == "buy" {
		inputDecimal, outputDecimal = 9, 6
		tradeType = types.TradeTypeBuy
	} else {
		inputDecimal, outputDecimal = 6, 9
		tradeType = types.TradeTypeSell
	}

	return &types.TradeInfo{
		Type: tradeType,
		Pool: swapData.Pools,
		User: swapData.User,
		InputToken: types.TokenInfo{
			Mint:      swapData.InputMint,
			Amount:    types.ConvertToUIAmountUint64(swapData.InputAmount, inputDecimal),
			AmountRaw: fmt.Sprintf("%d", swapData.InputAmount),
			Decimals:  inputDecimal,
		},
		OutputToken: types.TokenInfo{
			Mint:      swapData.OutputMint,
			Amount:    types.ConvertToUIAmountUint64(swapData.OutputAmount, outputDecimal),
			AmountRaw: fmt.Sprintf("%d", swapData.OutputAmount),
			Decimals:  outputDecimal,
		},
		ProgramId: constants.DEX_PROGRAMS.PHOTON.ID,
		AMMs:      swapData.Programs,
		Route:     constants.DEX_PROGRAMS.PHOTON.Name,
	}
}
