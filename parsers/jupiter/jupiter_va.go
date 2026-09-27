package jupiter

import (
	"bytes"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
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
				trade := p.parseFullFilled(ci.Instruction, utils.FormatIdx(ci.OuterIndex, ci.InnerIndex))
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

// getAMM gets the AMM name: the first AMM (in execution order) that moved tokens
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

// getTransferActionKeys returns the transfer action keys in execution order,
// so the AMM chosen from them does not depend on map iteration order
func (p *JupiterVAParser) getTransferActionKeys() []string {
	return utils.SortedTransferKeys(p.TransferActions)
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

			idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)

			discriminator := data[:16]
			if bytes.Equal(discriminator, constants.DISCRIMINATORS.JUPITER_VA.OPEN_EVENT) {
				transfers = append(transfers, p.parseOpen(data, ci, idx)...)
			} else if bytes.Equal(discriminator, constants.DISCRIMINATORS.JUPITER_VA.WITHDRAW_EVENT) {
				transfers = append(transfers, p.parseWithdraw(data, ci, idx)...)
			}
		}
	}

	return transfers
}

// parseOpen parses VA open event
func (p *JupiterVAParser) parseOpen(data []byte, ci types.ClassifiedInstruction, idx string) []types.TransferData {
	var transfers []types.TransferData
	programId := ci.ProgramId

	// The VA instruction that emitted the event
	order := findEmittingInstruction(p.Adapter, p.ClassifiedInstructions, ci)
	if order == nil {
		return transfers
	}
	eventInstruction := order.Instruction

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

	// The deposit is the event's amount; the balances are the user's
	// (owner-level, so token accounts of the user are found)
	balance := p.userBalanceChange(user, event.InputMint)
	decimals := p.Adapter.GetTokenDecimals(event.InputMint)
	amount := new(big.Int).SetUint64(event.Deposit)
	uiAmount := types.ConvertToUIAmount(amount, decimals)

	transfer := types.TransferData{
		Type:      "open",
		ProgramId: programId,
		Info: types.TransferDataInfo{
			Authority:        user,
			Source:           source,
			Destination:      destination,
			DestinationOwner: p.Adapter.GetTokenAccountOwner(destination),
			Mint:             event.InputMint,
			TokenAmount: types.TokenAmount{
				Amount:   amount.String(),
				UIAmount: &uiAmount,
				Decimals: decimals,
			},
		},
		Idx:       idx,
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
	}
	if balance != nil {
		transfer.Info.SourceBalance = &balance.Post
		transfer.Info.SourcePreBalance = &balance.Pre
	}
	transfers = append(transfers, transfer)

	return transfers
}

// parseWithdraw parses VA withdraw event
func (p *JupiterVAParser) parseWithdraw(data []byte, ci types.ClassifiedInstruction, idx string) []types.TransferData {
	var transfers []types.TransferData
	programId := ci.ProgramId

	// The VA instruction that emitted the event
	order := findEmittingInstruction(p.Adapter, p.ClassifiedInstructions, ci)
	if order == nil {
		return transfers
	}
	eventInstruction := order.Instruction

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

	// withdraw accounts: payer, user, value_average, input_mint, output_mint,
	// value_average_vault, user_input_account, user_output_account,
	// intermediate_account (the program id when absent), ...
	user := accounts[1]
	source := accounts[5]

	// The withdrawn amount is the event's amount; the balances are the user's
	// (owner-level, so token accounts of the user are found)
	balance := p.userBalanceChange(user, event.Mint)
	decimals := p.Adapter.GetTokenDecimals(event.Mint)
	amount := new(big.Int).SetUint64(event.Amount)
	uiAmount := types.ConvertToUIAmount(amount, decimals)

	transfer := types.TransferData{
		Type:      "withdraw",
		ProgramId: programId,
		Info: types.TransferDataInfo{
			Authority:        p.Adapter.GetTokenAccountOwner(source),
			Source:           source,
			Destination:      user,
			DestinationOwner: p.Adapter.GetTokenAccountOwner(user),
			Mint:             event.Mint,
			TokenAmount: types.TokenAmount{
				Amount:   amount.String(),
				UIAmount: &uiAmount,
				Decimals: decimals,
			},
		},
		Idx:       idx,
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
	}
	if balance != nil {
		transfer.Info.DestinationBalance = &balance.Post
		transfer.Info.DestinationPreBalance = &balance.Pre
	}
	transfers = append(transfers, transfer)

	return transfers
}

// userBalanceChange returns the balance change of wallet user in mint: SOL
// lamports for SOL, else the sum over the user's token accounts of mint
func (p *JupiterVAParser) userBalanceChange(user, mint string) *types.BalanceChange {
	if mint == constants.TOKENS.SOL {
		if balance := p.Adapter.GetAccountSolBalanceChanges(false)[user]; balance != nil {
			return balance
		}
	}
	if userTokens, ok := p.Adapter.GetAccountTokenBalanceChanges(true)[user]; ok {
		return userTokens[mint]
	}
	return nil
}
