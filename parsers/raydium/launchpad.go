package raydium

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/parsers"
	"github.com/solana-dex-parser-go/types"
)

// RaydiumLaunchpadParser parses Raydium Launchpad transactions
type RaydiumLaunchpadParser struct {
	*parsers.BaseParser
	eventParser *RaydiumLaunchpadEventParser
}

// NewRaydiumLaunchpadParser creates a new Raydium Launchpad parser
func NewRaydiumLaunchpadParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *RaydiumLaunchpadParser {
	return &RaydiumLaunchpadParser{
		BaseParser:  parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
		eventParser: NewRaydiumLaunchpadEventParser(adapter, transferActions),
	}
}

// ProcessTrades parses Raydium Launchpad trades
func (p *RaydiumLaunchpadParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	events := p.eventParser.ParseInstructions(p.ClassifiedInstructions)

	for _, event := range events {
		if event.Type == types.TradeTypeBuy || event.Type == types.TradeTypeSell || event.Type == "SWAP" {
			trade := p.createTradeInfo(event)
			if trade != nil {
				trades = append(trades, *trade)
			}
		}
	}

	return trades
}

// createTradeInfo creates a TradeInfo from a MemeEvent
func (p *RaydiumLaunchpadParser) createTradeInfo(event *types.MemeEvent) *types.TradeInfo {
	isBuy := event.Type == types.TradeTypeBuy

	var inputToken, outputToken string
	var inputDecimal, outputDecimal uint8

	if isBuy {
		inputToken = event.QuoteMint
		inputDecimal = p.Adapter.GetTokenDecimals(event.QuoteMint)
		outputToken = event.BaseMint
		outputDecimal = p.Adapter.GetTokenDecimals(event.BaseMint)
	} else {
		inputToken = event.BaseMint
		inputDecimal = p.Adapter.GetTokenDecimals(event.BaseMint)
		outputToken = event.QuoteMint
		outputDecimal = p.Adapter.GetTokenDecimals(event.QuoteMint)
	}

	if inputToken == "" || outputToken == "" {
		return nil
	}

	trade := getRaydiumTradeInfo(
		event,
		inputToken, inputDecimal,
		outputToken, outputDecimal,
		p.Adapter.Slot(),
		p.Adapter.Signature(),
		p.Adapter.BlockTime(),
		event.Idx,
		p.DexInfo,
	)

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// getRaydiumTradeInfo creates a TradeInfo from event data
func getRaydiumTradeInfo(
	event *types.MemeEvent,
	inputMint string, inputDecimal uint8,
	outputMint string, outputDecimal uint8,
	slot uint64, signature string, timestamp int64, idx string,
	dexInfo types.DexInfo,
) *types.TradeInfo {
	isBuy := event.Type == types.TradeTypeBuy

	// Calculate total fee from float64 values
	var feeTotal float64
	if event.ProtocolFee != nil {
		feeTotal += *event.ProtocolFee
	}
	if event.CreatorFee != nil {
		feeTotal += *event.CreatorFee
	}
	if event.PlatformFee != nil {
		feeTotal += *event.PlatformFee
	}

	var feeMint string
	var feeDecimals uint8
	if isBuy {
		feeMint = inputMint
		feeDecimals = inputDecimal
	} else {
		feeMint = outputMint
		feeDecimals = outputDecimal
	}

	tradeType := types.TradeTypeSell
	if isBuy {
		tradeType = types.TradeTypeBuy
	}

	var pool []string
	if event.Pool != "" {
		pool = []string{event.Pool}
	}

	programId := dexInfo.ProgramId
	if programId == "" {
		programId = constants.DEX_PROGRAMS.RAYDIUM_LCP.ID
	}

	// Convert fee to raw amount
	feeBigInt := new(big.Int).SetUint64(uint64(feeTotal * float64(pow10(feeDecimals))))

	return &types.TradeInfo{
		Type:        tradeType,
		Pool:        pool,
		InputToken:  *event.InputToken,
		OutputToken: *event.OutputToken,
		Fee: &types.FeeInfo{
			Mint:      feeMint,
			Amount:    feeTotal,
			AmountRaw: feeBigInt.String(),
			Decimals:  feeDecimals,
		},
		User:      event.User,
		ProgramId: programId,
		AMM:       constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Route:     dexInfo.Route,
		Slot:      slot,
		Timestamp: timestamp,
		Signature: signature,
		Idx:       idx,
	}
}

func pow10(n uint8) uint64 {
	result := uint64(1)
	for i := uint8(0); i < n; i++ {
		result *= 10
	}
	return result
}

// RaydiumLaunchpadEventParser parses Raydium Launchpad events
type RaydiumLaunchpadEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
}

// NewRaydiumLaunchpadEventParser creates a new event parser
func NewRaydiumLaunchpadEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *RaydiumLaunchpadEventParser {
	return &RaydiumLaunchpadEventParser{
		adapter:         adapter,
		transferActions: transferActions,
	}
}

// ParseInstructions parses classified instructions into meme events
func (p *RaydiumLaunchpadEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	for _, ci := range instructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.RAYDIUM_LCP.ID {
			continue
		}

		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		// For outer instructions (InnerIndex = -1), track that state
		innerIdx := ci.InnerIndex
		effectiveInnerIdx := innerIdx
		if effectiveInnerIdx < 0 {
			effectiveInnerIdx = 0
		}

		var event *types.MemeEvent

		// Check for trade discriminators
		if len(data) >= 8 {
			disc := data[:8]
			if bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.BUY_EXACT_IN) ||
				bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.BUY_EXACT_OUT) ||
				bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.SELL_EXACT_IN) ||
				bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.SELL_EXACT_OUT) {
				event = p.decodeTradeInstruction(ci.Instruction, ci.OuterIndex, effectiveInnerIdx)
			}
		}

		// Check for create event
		if len(data) >= 16 {
			disc := data[:16]
			if bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.CREATE_EVENT) {
				event = p.decodeCreateEvent(data, ci.OuterIndex)
			}
		}

		// Check for migrate events
		if len(data) >= 8 {
			disc := data[:8]
			if bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.MIGRATE_TO_AMM) ||
				bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM_LCP.MIGRATE_TO_CPSWAP) {
				event = p.decodeCompleteInstruction(data, ci.Instruction)
			}
		}

		if event != nil {
			event.Signature = p.adapter.Signature()
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Idx = fmt.Sprintf("%d-%d", ci.OuterIndex, effectiveInnerIdx)
			events = append(events, event)
		}
	}

	return events
}

// decodeTradeInstruction decodes a trade instruction
func (p *RaydiumLaunchpadEventParser) decodeTradeInstruction(instruction interface{}, outerIndex int, innerIndex int) *types.MemeEvent {
	// Find inner instruction for event data
	eventInstruction := p.adapter.GetInnerInstruction(outerIndex, innerIndex+1)
	if eventInstruction == nil {
		return nil
	}

	eventData := p.adapter.GetInstructionData(eventInstruction)
	if len(eventData) < 16 {
		return nil
	}
	eventData = eventData[16:]

	// Determine version based on data length
	isNewVersion := len(eventData) > 130

	var evt *RaydiumLCPTradeEvent
	if isNewVersion {
		layout, err := ParseRaydiumLCPTradeV2Layout(eventData)
		if err != nil {
			return nil
		}
		evt = layout.ToObject()
	} else {
		layout, err := ParseRaydiumLCPTradeLayout(eventData)
		if err != nil {
			return nil
		}
		evt = layout.ToObject()
	}

	// Get instruction accounts
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 11 {
		return nil
	}
	evt.User = accounts[0]
	evt.BaseMint = accounts[9]
	evt.QuoteMint = accounts[10]

	var inputMint, outputMint string
	var inputAmount, outputAmount *big.Int
	var inputDecimals, outputDecimals uint8

	if evt.TradeDirection == TradeDirectionBuy {
		inputMint = evt.QuoteMint
		inputAmount = evt.AmountIn
		inputDecimals = 9
		outputMint = evt.BaseMint
		outputAmount = evt.AmountOut
		outputDecimals = 6
	} else {
		inputMint = evt.BaseMint
		inputAmount = evt.AmountIn
		inputDecimals = 6
		outputMint = evt.QuoteMint
		outputAmount = evt.AmountOut
		outputDecimals = 9
	}

	eventType := types.TradeTypeSell
	if evt.TradeDirection == TradeDirectionBuy {
		eventType = types.TradeTypeBuy
	}

	inputUIAmount := types.ConvertToUIAmount(inputAmount, inputDecimals)
	outputUIAmount := types.ConvertToUIAmount(outputAmount, outputDecimals)

	// Convert big.Int fees to float64
	protocolFee := bigIntToFloat64(evt.ProtocolFee, 9)
	platformFee := bigIntToFloat64(evt.PlatformFee, 9)
	shareFee := bigIntToFloat64(evt.ShareFee, 9)
	creatorFee := bigIntToFloat64(evt.CreatorFee, 9)

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:         eventType,
		BondingCurve: evt.PoolState,
		BaseMint:     evt.BaseMint,
		QuoteMint:    evt.QuoteMint,
		User:         evt.User,
		InputToken: &types.TokenInfo{
			Mint:      inputMint,
			AmountRaw: inputAmount.String(),
			Amount:    inputUIAmount,
			Decimals:  inputDecimals,
		},
		OutputToken: &types.TokenInfo{
			Mint:      outputMint,
			AmountRaw: outputAmount.String(),
			Amount:    outputUIAmount,
			Decimals:  outputDecimals,
		},
		ProtocolFee: &protocolFee,
		PlatformFee: &platformFee,
		ShareFee:    &shareFee,
		CreatorFee:  &creatorFee,
	}
}

func bigIntToFloat64(val *big.Int, decimals uint8) float64 {
	if val == nil {
		return 0
	}
	f := new(big.Float).SetInt(val)
	divisor := new(big.Float).SetUint64(pow10(decimals))
	f.Quo(f, divisor)
	result, _ := f.Float64()
	return result
}

// decodeCreateEvent decodes a create event
func (p *RaydiumLaunchpadEventParser) decodeCreateEvent(data []byte, outerIndex int) *types.MemeEvent {
	instructions := p.adapter.Instructions()
	if outerIndex >= len(instructions) {
		return nil
	}

	eventInstruction := instructions[outerIndex]

	// Parse event data
	eventData := data[16:]
	layout, err := ParsePoolCreateEventLayout(eventData)
	if err != nil {
		return nil
	}
	evt := layout.ToObject()

	// Get instruction accounts
	accounts := p.adapter.GetInstructionAccounts(eventInstruction)
	if len(accounts) < 8 {
		return nil
	}
	evt.BaseMint = accounts[6]
	evt.QuoteMint = accounts[7]

	decimals := evt.BaseMintParam.Decimals

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:         types.TradeTypeCreate,
		Timestamp:    p.adapter.BlockTime(),
		User:         evt.Creator,
		BaseMint:     evt.BaseMint,
		QuoteMint:    evt.QuoteMint,
		Name:         evt.BaseMintParam.Name,
		Symbol:       evt.BaseMintParam.Symbol,
		URI:          evt.BaseMintParam.URI,
		Decimals:     &decimals,
		BondingCurve: evt.PoolState,
		Creator:      evt.Creator,
	}
}

// decodeCompleteInstruction decodes a migrate instruction
func (p *RaydiumLaunchpadEventParser) decodeCompleteInstruction(data []byte, instruction interface{}) *types.MemeEvent {
	if len(data) < 8 {
		return nil
	}

	discriminator := data[:8]
	accounts := p.adapter.GetInstructionAccounts(instruction)

	var baseMint, quoteMint, poolMint string
	var amm string

	if bytes.Equal(discriminator, constants.DISCRIMINATORS.RAYDIUM_LCP.MIGRATE_TO_AMM) {
		if len(accounts) < 17 {
			return nil
		}
		baseMint = accounts[1]
		quoteMint = accounts[2]
		poolMint = accounts[13]
		amm = constants.DEX_PROGRAMS.RAYDIUM_V4.Name
	} else {
		if len(accounts) < 8 {
			return nil
		}
		baseMint = accounts[1]
		quoteMint = accounts[2]
		poolMint = accounts[5]
		amm = constants.DEX_PROGRAMS.RAYDIUM_CPMM.Name
	}

	return &types.MemeEvent{
		Protocol:  constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:      types.TradeTypeMigrate,
		Timestamp: p.adapter.BlockTime(),
		BaseMint:  baseMint,
		QuoteMint: quoteMint,
		Pool:      poolMint,
		PoolDex:   amm,
	}
}

// ProcessEvents implements the EventParser interface
func (p *RaydiumLaunchpadEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForProgramRaydiumLCP(p.adapter, constants.DEX_PROGRAMS.RAYDIUM_LCP.ID)
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}

// getAllInstructionsForProgramRaydiumLCP gets all instructions for Raydium Launchpad program
func getAllInstructionsForProgramRaydiumLCP(adapter *adapter.TransactionAdapter, programId string) []types.ClassifiedInstruction {
	var instructions []types.ClassifiedInstruction

	// Process outer instructions
	for i, ix := range adapter.Instructions() {
		ixProgramId := adapter.GetInstructionProgramId(ix)
		if ixProgramId == programId {
			instructions = append(instructions, types.ClassifiedInstruction{
				ProgramId:   ixProgramId,
				Instruction: ix,
				OuterIndex:  i,
				InnerIndex:  -1,
			})
		}
	}

	// Process inner instructions
	for _, innerSet := range adapter.InnerInstructions() {
		for j, innerIx := range innerSet.Instructions {
			ixProgramId := adapter.GetInstructionProgramId(innerIx)
			if ixProgramId == programId {
				instructions = append(instructions, types.ClassifiedInstruction{
					ProgramId:   ixProgramId,
					Instruction: innerIx,
					OuterIndex:  innerSet.Index,
					InnerIndex:  j,
				})
			}
		}
	}

	return instructions
}
