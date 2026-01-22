package jupiter

import (
	"bytes"
	"fmt"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// JupiterShredParser parses Jupiter V6 instructions from shred-stream
type JupiterShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewJupiterShredParser creates a new JupiterShredParser
func NewJupiterShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *JupiterShredParser {
	return &JupiterShredParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes Jupiter instructions and returns parsed results
func (p *JupiterShredParser) ProcessInstructions() []interface{} {
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.JUPITER.ID)
	return p.parseInstructions(instructions)
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *JupiterShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.JUPITER.ID)
	return p.parseTypedInstructions(instructions)
}

func (p *JupiterShredParser) parseInstructions(instructions []types.ClassifiedInstruction) []interface{} {
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
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.SHARE_ACCOUNTS_ROUTE):
			eventType = "shared_accounts_route"
			eventData = p.decodeShareAccountsRoute(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.SHARE_ACCOUNTS_EXACT_OUT_ROUTE):
			eventType = "shared_accounts_exact_out_route"
			eventData = p.decodeShareAccountsRoute(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.SHARE_ACCOUNTS_ROUTE_WITH_TOKEN_LEDGER):
			eventType = "shared_accounts_route_with_token_ledger"
			eventData = p.decodeShareAccountsRouteWithTokenLedger(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.ROUTE):
			eventType = "route"
			eventData = p.decodeRoute(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.ROUTE_EXACT_OUT):
			eventType = "route_exact_out"
			eventData = p.decodeRouteExactOut(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.ROUTE_WITH_TOKEN_LEDGER):
			eventType = "route_with_token_ledger"
			eventData = p.decodeRouteWithTokenLedger(ci.Instruction, payload)
		default:
			continue
		}

		if eventData != nil {
			event := &JupiterShredInstruction{
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

func (p *JupiterShredParser) parseTypedInstructions(instructions []types.ClassifiedInstruction) []types.ParsedShredInstruction {
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
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.SHARE_ACCOUNTS_ROUTE):
			eventType = "shared_accounts_route"
			trade = p.decodeShareAccountsRouteTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.SHARE_ACCOUNTS_EXACT_OUT_ROUTE):
			eventType = "shared_accounts_exact_out_route"
			trade = p.decodeShareAccountsRouteTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.SHARE_ACCOUNTS_ROUTE_WITH_TOKEN_LEDGER):
			eventType = "shared_accounts_route_with_token_ledger"
			trade = p.decodeShareAccountsRouteWithTokenLedgerTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.ROUTE):
			eventType = "route"
			trade = p.decodeRouteTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.ROUTE_EXACT_OUT):
			eventType = "route_exact_out"
			trade = p.decodeRouteExactOutTrade(ci.Instruction, payload)
		case bytes.Equal(disc, constants.DISCRIMINATORS.JUPITER.ROUTE_WITH_TOKEN_LEDGER):
			eventType = "route_with_token_ledger"
			trade = p.decodeRouteWithTokenLedgerTrade(ci.Instruction, payload)
		default:
			continue
		}

		if trade != nil {
			event := types.ParsedShredInstruction{
				ProgramID:   constants.DEX_PROGRAMS.JUPITER.ID,
				ProgramName: constants.DEX_PROGRAMS.JUPITER.Name,
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

// JupiterShredInstruction represents a parsed Jupiter instruction
type JupiterShredInstruction struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Slot      uint64      `json:"slot"`
	Timestamp int64       `json:"timestamp"`
	Signature string      `json:"signature"`
	Idx       string      `json:"idx"`
	Signer    []string    `json:"signer"`
}

// JupiterRouteData contains Jupiter route instruction data
type JupiterRouteData struct {
	User         string `json:"user"`
	InputMint    string `json:"inputMint"`
	OutputMint   string `json:"outputMint"`
	InputAmount  uint64 `json:"inputAmount"`
	OutputAmount uint64 `json:"outputAmount"`
	SlippageBps  uint16 `json:"slippageBps"`
}

func (p *JupiterShredParser) decodeShareAccountsRoute(instruction interface{}, data []byte) *JupiterRouteData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 9 {
		return nil
	}

	// Skip RoutePlan Vec to read amounts from the end (last 19 bytes: u64 + u64 + u16 + 1 padding)
	if len(data) < 19 {
		return nil
	}

	reader := utils.GetBinaryReader(data[len(data)-18:])
	defer reader.Release()

	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()
	slippageBps, _ := reader.ReadU16()

	if reader.HasError() {
		return nil
	}

	return &JupiterRouteData{
		User:         accounts[2],
		InputMint:    accounts[7],
		OutputMint:   accounts[8],
		InputAmount:  inputAmount,
		OutputAmount: outputAmount,
		SlippageBps:  slippageBps,
	}
}

func (p *JupiterShredParser) decodeShareAccountsRouteTrade(instruction interface{}, data []byte) *types.TradeInfo {
	routeData := p.decodeShareAccountsRoute(instruction, data)
	if routeData == nil {
		return nil
	}
	return p.buildTradeInfo(routeData)
}

func (p *JupiterShredParser) decodeShareAccountsRouteWithTokenLedger(instruction interface{}, data []byte) *JupiterRouteData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 9 {
		return nil
	}

	// Cannot get input amount for token ledger variants
	if len(data) < 11 {
		return nil
	}

	reader := utils.GetBinaryReader(data[len(data)-10:])
	defer reader.Release()

	outputAmount, _ := reader.ReadU64()
	slippageBps, _ := reader.ReadU16()

	if reader.HasError() {
		return nil
	}

	return &JupiterRouteData{
		User:         accounts[2],
		InputMint:    accounts[7],
		OutputMint:   accounts[8],
		InputAmount:  0, // Cannot get input amount for token ledger variants
		OutputAmount: outputAmount,
		SlippageBps:  slippageBps,
	}
}

func (p *JupiterShredParser) decodeShareAccountsRouteWithTokenLedgerTrade(instruction interface{}, data []byte) *types.TradeInfo {
	routeData := p.decodeShareAccountsRouteWithTokenLedger(instruction, data)
	if routeData == nil {
		return nil
	}
	return p.buildTradeInfo(routeData)
}

func (p *JupiterShredParser) decodeRoute(instruction interface{}, data []byte) *JupiterRouteData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 6 {
		return nil
	}

	if len(data) < 19 {
		return nil
	}

	reader := utils.GetBinaryReader(data[len(data)-18:])
	defer reader.Release()

	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()
	slippageBps, _ := reader.ReadU16()

	if reader.HasError() {
		return nil
	}

	inputMint := accounts[2]
	outputMint := accounts[5]

	// For route instruction, accounts[2] is userTokenAccount, need to resolve to mint
	// For now, use SOL as default if we can't resolve
	if inputMint == "" {
		inputMint = constants.TOKENS.SOL
	}

	return &JupiterRouteData{
		User:         accounts[1],
		InputMint:    inputMint,
		OutputMint:   outputMint,
		InputAmount:  inputAmount,
		OutputAmount: outputAmount,
		SlippageBps:  slippageBps,
	}
}

func (p *JupiterShredParser) decodeRouteTrade(instruction interface{}, data []byte) *types.TradeInfo {
	routeData := p.decodeRoute(instruction, data)
	if routeData == nil {
		return nil
	}
	return p.buildTradeInfo(routeData)
}

func (p *JupiterShredParser) decodeRouteExactOut(instruction interface{}, data []byte) *JupiterRouteData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 7 {
		return nil
	}

	if len(data) < 19 {
		return nil
	}

	reader := utils.GetBinaryReader(data[len(data)-18:])
	defer reader.Release()

	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()
	slippageBps, _ := reader.ReadU16()

	if reader.HasError() {
		return nil
	}

	return &JupiterRouteData{
		User:         accounts[1],
		InputMint:    accounts[5],
		OutputMint:   accounts[6],
		InputAmount:  inputAmount,
		OutputAmount: outputAmount,
		SlippageBps:  slippageBps,
	}
}

func (p *JupiterShredParser) decodeRouteExactOutTrade(instruction interface{}, data []byte) *types.TradeInfo {
	routeData := p.decodeRouteExactOut(instruction, data)
	if routeData == nil {
		return nil
	}
	return p.buildTradeInfo(routeData)
}

func (p *JupiterShredParser) decodeRouteWithTokenLedger(instruction interface{}, data []byte) *JupiterRouteData {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 6 {
		return nil
	}

	if len(data) < 11 {
		return nil
	}

	reader := utils.GetBinaryReader(data[len(data)-10:])
	defer reader.Release()

	outputAmount, _ := reader.ReadU64()
	slippageBps, _ := reader.ReadU16()

	if reader.HasError() {
		return nil
	}

	inputMint := accounts[2]
	if inputMint == "" {
		inputMint = constants.TOKENS.SOL
	}

	return &JupiterRouteData{
		User:         accounts[1],
		InputMint:    inputMint,
		OutputMint:   accounts[5],
		InputAmount:  0, // Cannot get input amount for token ledger variants
		OutputAmount: outputAmount,
		SlippageBps:  slippageBps,
	}
}

func (p *JupiterShredParser) decodeRouteWithTokenLedgerTrade(instruction interface{}, data []byte) *types.TradeInfo {
	routeData := p.decodeRouteWithTokenLedger(instruction, data)
	if routeData == nil {
		return nil
	}
	return p.buildTradeInfo(routeData)
}

func (p *JupiterShredParser) buildTradeInfo(data *JupiterRouteData) *types.TradeInfo {
	tradeType := utils.GetTradeType(data.InputMint, data.OutputMint)

	var inputDecimal, outputDecimal uint8 = 9, 6
	if tradeType == types.TradeTypeSell {
		inputDecimal, outputDecimal = 6, 9
	}

	slippageBps := int(data.SlippageBps)

	return &types.TradeInfo{
		Type: tradeType,
		Pool: []string{},
		User: data.User,
		InputToken: types.TokenInfo{
			Mint:      data.InputMint,
			Amount:    types.ConvertToUIAmountUint64(data.InputAmount, inputDecimal),
			AmountRaw: fmt.Sprintf("%d", data.InputAmount),
			Decimals:  inputDecimal,
		},
		OutputToken: types.TokenInfo{
			Mint:      data.OutputMint,
			Amount:    types.ConvertToUIAmountUint64(data.OutputAmount, outputDecimal),
			AmountRaw: fmt.Sprintf("%d", data.OutputAmount),
			Decimals:  outputDecimal,
		},
		ProgramId:   constants.DEX_PROGRAMS.JUPITER.ID,
		AMMs:        []string{},
		Route:       constants.DEX_PROGRAMS.JUPITER.Name,
		SlippageBps: &slippageBps,
	}
}
