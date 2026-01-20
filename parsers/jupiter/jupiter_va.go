package jupiter

import (
	"bytes"
	"fmt"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/parsers"
	"github.com/solana-dex-parser-go/types"
	"github.com/solana-dex-parser-go/utils"
)

// JupiterVAParser parses Jupiter VA (Value Average) transactions
type JupiterVAParser struct {
	*parsers.BaseParser
}

// NewJupiterVAParser creates a new Jupiter VA parser
func NewJupiterVAParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *JupiterVAParser {
	return &JupiterVAParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// ProcessTrades parses Jupiter VA trades
func (p *JupiterVAParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.JUPITER_VA.ID {
			data := p.Adapter.GetInstructionData(ci.Instruction)
			if len(data) >= 16 && bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_VA.FILL_EVENT) {
				innerIdx := ci.InnerIndex
				if innerIdx < 0 {
					innerIdx = 0
				}
				trade := p.parseFullFilled(ci.Instruction, fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx))
				if trade != nil {
					trades = append(trades, *trade)
				}
			}
		}
	}

	return trades
}

// parseFullFilled parses VA fill event
func (p *JupiterVAParser) parseFullFilled(instruction interface{}, idx string) *types.TradeInfo {
	data := p.Adapter.GetInstructionData(instruction)
	if len(data) < 16 {
		return nil
	}

	eventData := data[16:]
	layout, err := ParseJupiterVAFillLayout(eventData)
	if err != nil {
		return nil
	}

	event := layout.ToObject()

	inputDecimal := p.Adapter.GetTokenDecimals(event.InputMint)
	outputDecimal := p.Adapter.GetTokenDecimals(event.OutputMint)

	inUIAmount := types.ConvertToUIAmount(event.InputAmount, inputDecimal)
	outUIAmount := types.ConvertToUIAmount(event.OutputAmount, outputDecimal)
	feeUIAmount := types.ConvertToUIAmount(event.Fee, outputDecimal)

	trade := &types.TradeInfo{
		Type: utils.GetTradeType(event.InputMint, event.OutputMint),
		InputToken: types.TokenInfo{
			Mint:      event.InputMint,
			Amount:    inUIAmount,
			AmountRaw: event.InputAmount.String(),
			Decimals:  inputDecimal,
		},
		OutputToken: types.TokenInfo{
			Mint:      event.OutputMint,
			Amount:    outUIAmount,
			AmountRaw: event.OutputAmount.String(),
			Decimals:  outputDecimal,
		},
		Fee: &types.FeeInfo{
			Mint:      event.OutputMint,
			Amount:    feeUIAmount,
			AmountRaw: event.Fee.String(),
			Decimals:  outputDecimal,
		},
		User:      event.User,
		ProgramId: constants.DEX_PROGRAMS.JUPITER_VA.ID,
		AMM:       p.getAMM(),
		Route:     p.DexInfo.Route,
		Slot:      p.Adapter.Slot(),
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
		Idx:       idx,
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// getAMM gets the AMM name
func (p *JupiterVAParser) getAMM() string {
	amms := utils.GetAMMs(p.getTransferActionKeys())
	if len(amms) > 0 {
		return amms[0]
	}
	if p.DexInfo.AMM != "" {
		return p.DexInfo.AMM
	}
	return constants.DEX_PROGRAMS.JUPITER_VA.Name
}

// getTransferActionKeys returns all transfer action keys
func (p *JupiterVAParser) getTransferActionKeys() []string {
	keys := make([]string, 0, len(p.TransferActions))
	for k := range p.TransferActions {
		keys = append(keys, k)
	}
	return keys
}

// ProcessTransfers parses VA transfer operations
func (p *JupiterVAParser) ProcessTransfers() []types.TransferData {
	var transfers []types.TransferData

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.JUPITER_VA.ID {
			data := p.Adapter.GetInstructionData(ci.Instruction)
			if len(data) < 16 {
				continue
			}

			innerIdx := ci.InnerIndex
			if innerIdx < 0 {
				innerIdx = 0
			}
			idx := fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx)

			discriminator := data[:16]
			if bytes.Equal(discriminator, constants.DISCRIMINATORS.JUPITER_VA.OPEN_EVENT) {
				transfers = append(transfers, p.parseOpen(data, ci.ProgramId, ci.OuterIndex, idx)...)
			} else if bytes.Equal(discriminator, constants.DISCRIMINATORS.JUPITER_VA.WITHDRAW_EVENT) {
				transfers = append(transfers, p.parseWithdraw(data, ci.ProgramId, ci.OuterIndex, idx)...)
			}
		}
	}

	return transfers
}

// parseOpen parses VA open event
func (p *JupiterVAParser) parseOpen(data []byte, programId string, outerIndex int, idx string) []types.TransferData {
	var transfers []types.TransferData

	instructions := p.Adapter.Instructions()
	if outerIndex >= len(instructions) {
		return transfers
	}

	eventInstruction := instructions[outerIndex]

	// Parse event data
	eventData := data[16:]
	layout, err := ParseJupiterVAOpenLayout(eventData)
	if err != nil {
		return transfers
	}
	event := layout.ToObject()

	// Get outer instruction accounts
	accounts := p.Adapter.GetInstructionAccounts(eventInstruction)
	if len(accounts) < 7 {
		return transfers
	}

	user := event.User
	source := accounts[5]
	destination := accounts[6]

	var balance *types.BalanceChange
	if event.InputMint == constants.TOKENS.SOL {
		balanceChanges := p.Adapter.GetAccountSolBalanceChanges(false)
		balance = balanceChanges[user]
	} else {
		tokenChanges := p.Adapter.GetAccountTokenBalanceChanges(false)
		if userTokens, ok := tokenChanges[user]; ok {
			balance = userTokens[event.InputMint]
		}
	}

	if balance == nil {
		return transfers
	}

	uiAmount := float64(0)
	if balance.Change.UIAmount != nil {
		uiAmount = *balance.Change.UIAmount
	}

	transfers = append(transfers, types.TransferData{
		Type:      "open",
		ProgramId: programId,
		Info: types.TransferDataInfo{
			Authority:        user,
			Source:           source,
			Destination:      destination,
			DestinationOwner: p.Adapter.GetTokenAccountOwner(destination),
			Mint:             event.InputMint,
			TokenAmount: types.TokenAmount{
				Amount:   balance.Change.Amount,
				UIAmount: &uiAmount,
				Decimals: balance.Change.Decimals,
			},
			SourceBalance:    &balance.Post,
			SourcePreBalance: &balance.Pre,
		},
		Idx:       idx,
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
	})

	return transfers
}

// parseWithdraw parses VA withdraw event
func (p *JupiterVAParser) parseWithdraw(data []byte, programId string, outerIndex int, idx string) []types.TransferData {
	var transfers []types.TransferData

	instructions := p.Adapter.Instructions()
	if outerIndex >= len(instructions) {
		return transfers
	}

	eventInstruction := instructions[outerIndex]

	// Parse event data
	eventData := data[16:]
	layout, err := ParseJupiterVAWithdrawLayout(eventData)
	if err != nil {
		return transfers
	}
	event := layout.ToObject()

	// Get outer instruction accounts
	accounts := p.Adapter.GetInstructionAccounts(eventInstruction)
	if len(accounts) < 9 {
		return transfers
	}

	user := accounts[1]
	source := accounts[8]

	var balance *types.BalanceChange
	if event.Mint == constants.TOKENS.SOL {
		balanceChanges := p.Adapter.GetAccountSolBalanceChanges(false)
		balance = balanceChanges[user]
	} else {
		tokenChanges := p.Adapter.GetAccountTokenBalanceChanges(false)
		if userTokens, ok := tokenChanges[user]; ok {
			balance = userTokens[event.Mint]
		}
	}

	if balance == nil {
		return transfers
	}

	uiAmount := float64(0)
	if balance.Change.UIAmount != nil {
		uiAmount = *balance.Change.UIAmount
	}

	transfers = append(transfers, types.TransferData{
		Type:      "withdraw",
		ProgramId: programId,
		Info: types.TransferDataInfo{
			Authority:        p.Adapter.GetTokenAccountOwner(source),
			Source:           source,
			Destination:      user,
			DestinationOwner: p.Adapter.GetTokenAccountOwner(user),
			Mint:             event.Mint,
			TokenAmount: types.TokenAmount{
				Amount:   balance.Change.Amount,
				UIAmount: &uiAmount,
				Decimals: balance.Change.Decimals,
			},
			SourceBalance:    &balance.Post,
			SourcePreBalance: &balance.Pre,
		},
		Idx:       idx,
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
	})

	return transfers
}
