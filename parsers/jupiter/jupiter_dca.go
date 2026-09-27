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

// JupiterDCAParser parses Jupiter DCA transactions
type JupiterDCAParser struct {
	*parsers.BaseParser
}

// NewJupiterDCAParser creates a new Jupiter DCA parser
func NewJupiterDCAParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *JupiterDCAParser {
	return &JupiterDCAParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// ProcessTrades parses Jupiter DCA trades
func (p *JupiterDCAParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.JUPITER_DCA.ID {
			data := p.Adapter.GetInstructionData(ci.Instruction)
			if len(data) >= 16 && bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_DCA.FILLED) {
				trade := p.parseFullFilled(ci.Instruction, utils.FormatIdx(ci.OuterIndex, ci.InnerIndex))
				if trade != nil {
					trades = append(trades, *trade)
				}
			}
		}
	}

	return trades
}

// parseFullFilled parses DCA filled event
func (p *JupiterDCAParser) parseFullFilled(instruction interface{}, idx string) *types.TradeInfo {
	data := p.Adapter.GetInstructionData(instruction)
	if len(data) < 16 {
		return nil
	}

	eventData := data[16:]
	layout, err := ParseJupiterDCAFilledLayout(eventData)
	if err != nil {
		return nil
	}

	event := layout.ToObject()

	inputDecimal := p.Adapter.GetTokenDecimals(event.InputMint)
	outputDecimal := p.Adapter.GetTokenDecimals(event.OutputMint)
	feeDecimal := p.Adapter.GetTokenDecimals(event.FeeMint)

	inUIAmount := types.ConvertToUIAmount(event.InAmount, inputDecimal)
	outUIAmount := types.ConvertToUIAmount(event.OutAmount, outputDecimal)
	feeUIAmount := types.ConvertToUIAmount(event.Fee, feeDecimal)

	trade := &types.TradeInfo{
		Type: utils.GetTradeType(event.InputMint, event.OutputMint),
		InputToken: types.TokenInfo{
			Mint:      event.InputMint,
			Amount:    inUIAmount,
			AmountRaw: event.InAmount.String(),
			Decimals:  inputDecimal,
		},
		OutputToken: types.TokenInfo{
			Mint:      event.OutputMint,
			Amount:    outUIAmount,
			AmountRaw: event.OutAmount.String(),
			Decimals:  outputDecimal,
		},
		// The DCA program's fee, taken from the output (Filled.fee)
		Fee: &types.FeeInfo{
			Mint:      event.FeeMint,
			Amount:    feeUIAmount,
			AmountRaw: event.Fee.String(),
			Decimals:  feeDecimal,
			Dex:       constants.DEX_PROGRAMS.JUPITER_DCA.Name,
			Type:      "protocol",
		},
		User:      event.UserKey,
		ProgramId: constants.DEX_PROGRAMS.JUPITER_DCA.ID,
		AMM:       p.getAMM(),
		Route:     p.DexInfo.Route,
		Slot:      p.Adapter.Slot(),
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
		Idx:       idx,
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// getAMM gets the AMM name from transfer actions
func (p *JupiterDCAParser) getAMM() string {
	amms := utils.GetAMMs(p.getTransferActionKeys())
	if len(amms) > 0 {
		return amms[0]
	}
	if p.DexInfo.AMM != "" {
		return p.DexInfo.AMM
	}
	return constants.DEX_PROGRAMS.JUPITER_DCA.Name
}

// getTransferActionKeys returns the transfer action keys in execution order,
// so the AMM chosen from them does not depend on map iteration order
func (p *JupiterDCAParser) getTransferActionKeys() []string {
	return utils.SortedTransferKeys(p.TransferActions)
}

// ProcessTransfers reports DCA (Recurring) deposits, withdrawals and refunds
// from the DCA program's events. Opened (open_dca, open_dca_v2) is the user's
// deposit of in_deposited input tokens, Deposit (deposit) a later top-up.
// Withdraw pays tokens out of the DCA to its user: the output of a fill
// (transfer, by the keeper) or what the user's withdraw instruction takes.
// Closed (close_dca by the user, end_and_close by a keeper) returns what was
// left in the DCA account: the close instruction's token transfers of the
// input or output mint, or, when none is found, the event's unfilled input
// amount.
func (p *JupiterDCAParser) ProcessTransfers() []types.TransferData {
	var transfers []types.TransferData

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.JUPITER_DCA.ID {
			continue
		}
		data := p.Adapter.GetInstructionData(ci.Instruction)
		if len(data) < 16 {
			continue
		}
		switch {
		case bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_DCA.OPENED_EVENT):
			if event, err := ParseJupiterDCAOpenedEvent(data[16:]); err == nil {
				transfers = append(transfers, p.parseOpened(event, ci)...)
			}
		case bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_DCA.CLOSED_EVENT):
			if event, err := ParseJupiterDCAClosedEvent(data[16:]); err == nil {
				transfers = append(transfers, p.parseClosed(event, ci)...)
			}
		case bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_DCA.WITHDRAW_EVENT):
			if event, err := ParseJupiterDCAWithdrawEvent(data[16:]); err == nil {
				transfers = append(transfers, p.parseWithdraw(event, ci)...)
			}
		case bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_DCA.DEPOSIT_EVENT):
			if event, err := ParseJupiterDCADepositEvent(data[16:]); err == nil {
				transfers = append(transfers, p.parseDeposit(event, ci)...)
			}
		}
	}

	return transfers
}

// orderTransfers returns the token transfers of the DCA instruction that
// emitted the event ci
func (p *JupiterDCAParser) orderTransfers(ci types.ClassifiedInstruction) (*types.ClassifiedInstruction, []types.TransferData) {
	order := findEmittingInstruction(p.Adapter, p.ClassifiedInstructions, ci)
	if order == nil {
		return nil, nil
	}
	return order, p.GetTransfersForInstruction(order.ProgramId, order.OuterIndex, order.InnerIndex, nil)
}

// parseOpened reports the deposit of an Opened event: in_deposited of the
// input mint, from the user into the DCA's input account
func (p *JupiterDCAParser) parseOpened(event *JupiterDCAOrderEvent, ci types.ClassifiedInstruction) []types.TransferData {
	order, instTransfers := p.orderTransfers(ci)
	idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	if order != nil {
		idx = utils.FormatIdx(order.OuterIndex, order.InnerIndex)
	}

	transfer := p.eventTransfer("OpenDca", event.InputMint, event.InDeposited, event.UserKey, event.DCAKey, idx)
	transfer.Info.Authority = event.UserKey
	for i := range instTransfers {
		if t := instTransfers[i]; t.Info.Mint == event.InputMint && t.Info.TokenAmount.Amount == event.InDeposited.String() {
			transfer.Info.Source = t.Info.Source
			transfer.Info.Destination = t.Info.Destination
			transfer.Info.DestinationOwner = t.Info.DestinationOwner
			transfer.Info.SourceBalance = t.Info.SourceBalance
			transfer.Info.SourcePreBalance = t.Info.SourcePreBalance
			transfer.Info.DestinationBalance = t.Info.DestinationBalance
			transfer.Info.DestinationPreBalance = t.Info.DestinationPreBalance
			break
		}
	}
	return []types.TransferData{transfer}
}

// parseClosed reports what a close returned to the user: the close
// instruction's transfers of the order's input or output mint, else the
// event's unfilled input amount
func (p *JupiterDCAParser) parseClosed(event *JupiterDCAOrderEvent, ci types.ClassifiedInstruction) []types.TransferData {
	var transfers []types.TransferData
	order, instTransfers := p.orderTransfers(ci)
	idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	if order != nil {
		idx = utils.FormatIdx(order.OuterIndex, order.InnerIndex)
	}

	for _, t := range instTransfers {
		if t.Info.Mint != event.InputMint && t.Info.Mint != event.OutputMint {
			continue
		}
		// Reported as a DCA transfer like OpenDca: the order program's id,
		// not the token program's
		t.Type = "CloseDca"
		t.ProgramId = constants.DEX_PROGRAMS.JUPITER_DCA.ID
		t.Idx = idx
		transfers = append(transfers, t)
	}
	if len(transfers) == 0 && event.UnfilledAmount.Sign() > 0 {
		transfer := p.eventTransfer("CloseDca", event.InputMint, event.UnfilledAmount, event.DCAKey, event.UserKey, idx)
		transfer.Info.Authority = event.DCAKey
		transfers = append(transfers, transfer)
	}
	return transfers
}

// parseWithdraw reports a Withdraw event: in_amount of the input mint and
// out_amount of the output mint paid from the DCA to its user. Each side is
// the emitting instruction's token transfer of that amount (its accounts,
// authority and balances); without one (e.g. SOL unwrapped to the user) it
// is built from the event, with the mint from the instruction's accounts
// (withdraw: user 0, input_mint 2, output_mint 3; transfer: user 2,
// output_mint 3).
func (p *JupiterDCAParser) parseWithdraw(event *JupiterDCAWithdrawEvent, ci types.ClassifiedInstruction) []types.TransferData {
	order, instTransfers := p.orderTransfers(ci)
	idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	var user, inputMint, outputMint string
	if order != nil {
		idx = utils.FormatIdx(order.OuterIndex, order.InnerIndex)
		accounts := p.Adapter.GetInstructionAccounts(order.Instruction)
		data := p.Adapter.GetInstructionData(order.Instruction)
		switch {
		case constants.MatchDiscriminator(data, constants.DISCRIMINATORS.JUPITER_DCA.WITHDRAW) && len(accounts) > 3:
			user, inputMint, outputMint = accounts[0], accounts[2], accounts[3]
		case constants.MatchDiscriminator(data, constants.DISCRIMINATORS.JUPITER_DCA.TRANSFER) && len(accounts) > 3:
			user, outputMint = accounts[2], accounts[3]
		}
	}

	var transfers []types.TransferData
	for _, side := range []struct {
		amount *big.Int
		mint   string
	}{{event.InAmount, inputMint}, {event.OutAmount, outputMint}} {
		if side.amount.Sign() <= 0 {
			continue
		}
		if t := dcaTransferOf(instTransfers, side.mint, side.amount); t != nil {
			transfer := *t
			transfer.Type = "WithdrawDca"
			transfer.ProgramId = constants.DEX_PROGRAMS.JUPITER_DCA.ID
			transfer.Idx = idx
			transfers = append(transfers, transfer)
		} else if side.mint != "" {
			transfer := p.eventTransfer("WithdrawDca", side.mint, side.amount, event.DCAKey, user, idx)
			transfer.Info.Authority = event.DCAKey
			transfers = append(transfers, transfer)
		}
	}
	return transfers
}

// parseDeposit reports a Deposit event: amount of the input mint the user
// added to the DCA (deposit accounts: user 0, dca 1, in_ata 2, user_in_ata 3)
func (p *JupiterDCAParser) parseDeposit(event *JupiterDCADepositEvent, ci types.ClassifiedInstruction) []types.TransferData {
	if event.Amount.Sign() <= 0 {
		return nil
	}
	order, instTransfers := p.orderTransfers(ci)
	idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	var user, inAta string
	if order != nil {
		idx = utils.FormatIdx(order.OuterIndex, order.InnerIndex)
		if accounts := p.Adapter.GetInstructionAccounts(order.Instruction); len(accounts) > 2 {
			user, inAta = accounts[0], accounts[2]
		}
	}
	if t := dcaTransferOf(instTransfers, "", event.Amount); t != nil {
		transfer := *t
		transfer.Type = "DepositDca"
		transfer.ProgramId = constants.DEX_PROGRAMS.JUPITER_DCA.ID
		transfer.Idx = idx
		return []types.TransferData{transfer}
	}
	mint := p.Adapter.KnownTokenAccountMint(inAta)
	if mint == "" {
		return nil
	}
	transfer := p.eventTransfer("DepositDca", mint, event.Amount, user, event.DCAKey, idx)
	transfer.Info.Authority = user
	return []types.TransferData{transfer}
}

// dcaTransferOf returns the first token transfer of amount (and of mint,
// when given) among transfers, or nil
func dcaTransferOf(transfers []types.TransferData, mint string, amount *big.Int) *types.TransferData {
	for i := range transfers {
		t := &transfers[i]
		if t.Info.TokenAmount.Amount == amount.String() && (mint == "" || t.Info.Mint == mint) &&
			t.ProgramId != constants.SYSTEM_PROGRAM_ID {
			return t
		}
	}
	return nil
}

// eventTransfer builds a transfer of amount of mint from source to destination
func (p *JupiterDCAParser) eventTransfer(transferType, mint string, amount *big.Int, source, destination, idx string) types.TransferData {
	decimals := p.Adapter.GetTokenDecimals(mint)
	uiAmount := types.ConvertToUIAmount(amount, decimals)
	return types.TransferData{
		Type:      transferType,
		ProgramId: constants.DEX_PROGRAMS.JUPITER_DCA.ID,
		Info: types.TransferDataInfo{
			Source:      source,
			Destination: destination,
			Mint:        mint,
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
}

// GetAMMs extracts AMM names from transfer action keys
func GetAMMs(keys []string) []string {
	return utils.GetAMMs(keys)
}
