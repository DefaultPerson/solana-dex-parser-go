package jupiter

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// JupiterLimitOrderV2Parser parses Jupiter Limit Order V2 transactions
type JupiterLimitOrderV2Parser struct {
	*parsers.BaseParser
}

// NewJupiterLimitOrderV2Parser creates a new Jupiter Limit Order V2 parser
func NewJupiterLimitOrderV2Parser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *JupiterLimitOrderV2Parser {
	return &JupiterLimitOrderV2Parser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// ProcessTrades parses Jupiter Limit Order V2 trades
func (p *JupiterLimitOrderV2Parser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID {
			data := p.Adapter.GetInstructionData(ci.Instruction)
			if len(data) >= 16 && bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER_V2.TRADE_EVENT) {
				innerIdx := ci.InnerIndex
				if innerIdx < 0 {
					innerIdx = 0
				}
				trade := p.parseFlashFilled(data, ci.OuterIndex, fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx))
				if trade != nil {
					trades = append(trades, *trade)
				}
			}
		}
	}

	return trades
}

// parseFlashFilled parses flash filled trade event
func (p *JupiterLimitOrderV2Parser) parseFlashFilled(data []byte, outerIndex int, idx string) *types.TradeInfo {
	instructions := p.Adapter.Instructions()
	if outerIndex >= len(instructions) {
		return nil
	}

	eventInstruction := instructions[outerIndex]

	// Parse event data
	eventData := data[16:]
	layout, err := ParseJupiterLimitOrderV2TradeLayout(eventData)
	if err != nil {
		return nil
	}
	event := layout.ToObject()

	// Get outer instruction accounts
	accounts := p.Adapter.GetInstructionAccounts(eventInstruction)
	outerData := p.Adapter.GetInstructionData(eventInstruction)

	var inputToken, outputToken *types.TokenInfo

	// Determine token info based on instruction discriminator
	if len(outerData) >= 8 && bytes.Equal(outerData[:8], constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER_V2.UNKNOWN) {
		// Unknown instruction
		if len(accounts) > 4 {
			if tokenInfo, ok := p.Adapter.SPLTokenMap[accounts[3]]; ok {
				inputToken = &tokenInfo
			}
			if tokenInfo, ok := p.Adapter.SPLTokenMap[accounts[4]]; ok {
				outputToken = &tokenInfo
			}
		}
	} else {
		// FlashFillOrder instruction
		if len(accounts) > 8 {
			if tokenInfo, ok := p.Adapter.SPLTokenMap[accounts[3]]; ok {
				inputToken = &tokenInfo
			}
			outputToken = &types.TokenInfo{
				Mint:     accounts[8],
				Decimals: p.Adapter.GetTokenDecimals(accounts[8]),
			}
		}
	}

	if inputToken == nil || outputToken == nil {
		return nil
	}

	// Jupiter fee 0.1%
	feeAmount := new(big.Int).Div(event.TakingAmount, big.NewInt(1000))
	outAmount := new(big.Int).Sub(event.TakingAmount, feeAmount)

	inUIAmount := types.ConvertToUIAmount(event.MakingAmount, inputToken.Decimals)
	outUIAmount := types.ConvertToUIAmount(outAmount, outputToken.Decimals)
	feeUIAmount := types.ConvertToUIAmount(feeAmount, outputToken.Decimals)

	trade := &types.TradeInfo{
		Type: utils.GetTradeType(inputToken.Mint, outputToken.Mint),
		InputToken: types.TokenInfo{
			Mint:      inputToken.Mint,
			Amount:    inUIAmount,
			AmountRaw: event.MakingAmount.String(),
			Decimals:  inputToken.Decimals,
		},
		OutputToken: types.TokenInfo{
			Mint:      outputToken.Mint,
			Amount:    outUIAmount,
			AmountRaw: outAmount.String(),
			Decimals:  outputToken.Decimals,
		},
		Fee: &types.FeeInfo{
			Mint:      outputToken.Mint,
			Amount:    feeUIAmount,
			AmountRaw: feeAmount.String(),
			Decimals:  outputToken.Decimals,
		},
		User:      event.Taker,
		ProgramId: constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID,
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
func (p *JupiterLimitOrderV2Parser) getAMM() string {
	amms := utils.GetAMMs(p.getTransferActionKeys())
	if len(amms) > 0 {
		return amms[0]
	}
	if p.DexInfo.AMM != "" {
		return p.DexInfo.AMM
	}
	return constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.Name
}

// getTransferActionKeys returns all transfer action keys
func (p *JupiterLimitOrderV2Parser) getTransferActionKeys() []string {
	keys := make([]string, 0, len(p.TransferActions))
	for k := range p.TransferActions {
		keys = append(keys, k)
	}
	return keys
}

// ProcessTransfers parses Limit Order V2 transfer operations
func (p *JupiterLimitOrderV2Parser) ProcessTransfers() []types.TransferData {
	var transfers []types.TransferData

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID {
			data := p.Adapter.GetInstructionData(ci.Instruction)
			if len(data) < 8 {
				continue
			}

			innerIdx := ci.InnerIndex
			if innerIdx < 0 {
				innerIdx = 0
			}
			idx := fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx)

			if len(data) >= 16 && bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER_V2.CREATE_ORDER_EVENT) {
				transfers = append(transfers, p.parseInitializeOrder(data, ci.ProgramId, ci.OuterIndex, idx)...)
			} else if bytes.Equal(data[:8], constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER_V2.CANCEL_ORDER) {
				transfers = append(transfers, p.parseCancelOrder(ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx)...)
			}
		}
	}

	// Deduplicate transfers
	if len(transfers) > 1 {
		seen := make(map[string]bool)
		var unique []types.TransferData
		for _, t := range transfers {
			key := fmt.Sprintf("%s-%s=%v", t.Idx, t.Signature, t.IsFee)
			if !seen[key] {
				seen[key] = true
				unique = append(unique, t)
			}
		}
		return unique
	}

	return transfers
}

// parseInitializeOrder parses create order event
func (p *JupiterLimitOrderV2Parser) parseInitializeOrder(data []byte, programId string, outerIndex int, idx string) []types.TransferData {
	var transfers []types.TransferData

	instructions := p.Adapter.Instructions()
	if outerIndex >= len(instructions) {
		return transfers
	}

	eventInstruction := instructions[outerIndex]

	// Parse event data
	eventData := data[16:]
	layout, err := ParseJupiterLimitOrderV2CreateOrderLayout(eventData)
	if err != nil {
		return transfers
	}
	event := layout.ToObject()

	// Get outer instruction accounts
	accounts := p.Adapter.GetInstructionAccounts(eventInstruction)
	if len(accounts) < 5 {
		return transfers
	}

	user := event.Maker
	source := accounts[4]
	destination := accounts[3]

	var balance *types.BalanceChange
	if event.InputMint == constants.TOKENS.SOL {
		balanceChanges := p.Adapter.GetAccountSolBalanceChanges(false)
		balance = balanceChanges[user]
	} else {
		tokenChanges := p.Adapter.GetAccountTokenBalanceChanges(true)
		if userTokens, ok := tokenChanges[user]; ok {
			balance = userTokens[event.InputMint]
		}
	}

	if balance == nil {
		return transfers
	}

	decimals := p.Adapter.GetTokenDecimals(event.InputMint)
	uiAmount := types.ConvertToUIAmount(event.MakingAmount, decimals)

	transfers = append(transfers, types.TransferData{
		Type:      "initializeOrder",
		ProgramId: programId,
		Info: types.TransferDataInfo{
			Authority:        p.Adapter.GetTokenAccountOwner(source),
			Source:           source,
			Destination:      destination,
			DestinationOwner: p.Adapter.GetTokenAccountOwner(source),
			Mint:             event.InputMint,
			TokenAmount: types.TokenAmount{
				Amount:   event.MakingAmount.String(),
				UIAmount: &uiAmount,
				Decimals: decimals,
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

// parseCancelOrder parses cancel order instruction
func (p *JupiterLimitOrderV2Parser) parseCancelOrder(instruction interface{}, programId string, outerIndex int, innerIndex int) []types.TransferData {
	var transfers []types.TransferData

	accounts := p.Adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 6 {
		return transfers
	}

	user := accounts[1]
	mint := accounts[5]
	source := accounts[3]
	authority := accounts[2]
	destination := accounts[4]
	if mint == constants.TOKENS.SOL {
		destination = user
	}

	var balance *types.BalanceChange
	if mint == constants.TOKENS.SOL {
		balanceChanges := p.Adapter.GetAccountSolBalanceChanges(false)
		balance = balanceChanges[destination]
	} else {
		tokenChanges := p.Adapter.GetAccountTokenBalanceChanges(false)
		if destTokens, ok := tokenChanges[destination]; ok {
			balance = destTokens[mint]
		}
	}

	if balance == nil {
		return transfers
	}

	innerIdx := innerIndex
	if innerIdx < 0 {
		innerIdx = 0
	}
	idx := fmt.Sprintf("%d-%d", outerIndex, innerIdx)

	instTransfers := p.GetTransfersForInstruction(programId, outerIndex, innerIdx, nil)
	var transfer *types.TransferData
	for i := range instTransfers {
		if instTransfers[i].Info.Mint == mint {
			transfer = &instTransfers[i]
			break
		}
	}

	decimals := uint8(0)
	tokenAmount := balance.Change.Amount
	if transfer != nil {
		decimals = transfer.Info.TokenAmount.Decimals
		tokenAmount = transfer.Info.TokenAmount.Amount
	} else {
		decimals = p.Adapter.GetTokenDecimals(mint)
	}

	uiAmount := types.ConvertToUIAmount(new(big.Int).SetUint64(0), decimals)
	if tokenAmount != "" {
		amt, _ := new(big.Int).SetString(tokenAmount, 10)
		if amt != nil {
			uiAmount = types.ConvertToUIAmount(amt, decimals)
		}
	}

	authorityStr := authority
	sourceStr := source
	destinationStr := destination
	if transfer != nil {
		if transfer.Info.Authority != "" {
			authorityStr = transfer.Info.Authority
		}
		if transfer.Info.Source != "" {
			sourceStr = transfer.Info.Source
		}
		if transfer.Info.Destination != "" {
			destinationStr = transfer.Info.Destination
		}
	}

	transfers = append(transfers, types.TransferData{
		Type:      "cancelOrder",
		ProgramId: programId,
		Info: types.TransferDataInfo{
			Authority:             authorityStr,
			Source:                sourceStr,
			Destination:           destinationStr,
			DestinationOwner:      p.Adapter.GetTokenAccountOwner(destination),
			Mint:                  mint,
			TokenAmount:           types.TokenAmount{Amount: tokenAmount, UIAmount: &uiAmount, Decimals: decimals},
			DestinationBalance:    &balance.Post,
			DestinationPreBalance: &balance.Pre,
		},
		Idx:       idx,
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
	})

	// Add SOL balance change if not SOL order
	if mint != constants.TOKENS.SOL {
		solBalanceChanges := p.Adapter.GetAccountSolBalanceChanges(false)
		if solBalance, ok := solBalanceChanges[user]; ok && solBalance != nil {
			solUIAmount := float64(0)
			if solBalance.Change.UIAmount != nil {
				solUIAmount = *solBalance.Change.UIAmount
			}
			transfers = append(transfers, types.TransferData{
				Type:      "cancelOrder",
				ProgramId: programId,
				Info: types.TransferDataInfo{
					Authority:             authorityStr,
					Source:                sourceStr,
					Destination:           user,
					Mint:                  constants.TOKENS.SOL,
					TokenAmount:           types.TokenAmount{Amount: solBalance.Change.Amount, UIAmount: &solUIAmount, Decimals: solBalance.Change.Decimals},
					DestinationBalance:    &solBalance.Post,
					DestinationPreBalance: &solBalance.Pre,
				},
				Idx:       idx,
				Timestamp: p.Adapter.BlockTime(),
				Signature: p.Adapter.Signature(),
				IsFee:     true,
			})
		}
	}

	return transfers
}
