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

// ProcessTrades parses Jupiter Limit Order V2 (Trigger) fills from their
// TradeEvent. The user is the order's maker: the maker sold making_amount of
// the input mint and received the output the program paid out, after the fee.
func (p *JupiterLimitOrderV2Parser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID {
			continue
		}
		data := p.Adapter.GetInstructionData(ci.Instruction)
		if len(data) >= 16 && bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER_V2.TRADE_EVENT) {
			if trade := p.parseFlashFilled(data, ci); trade != nil {
				trades = append(trades, *trade)
			}
		}
	}

	return trades
}

// orderInstruction returns the Limit Order V2 instruction that emitted the
// event ci: the last non-event instruction of the program before ci in the same
// outer instruction (the outer instruction itself when the program is called
// directly)
func (p *JupiterLimitOrderV2Parser) orderInstruction(ci types.ClassifiedInstruction) *types.ClassifiedInstruction {
	return findEmittingInstruction(p.Adapter, p.ClassifiedInstructions, ci)
}

// parseFlashFilled parses the TradeEvent of a fill_order or flash_fill_order
func (p *JupiterLimitOrderV2Parser) parseFlashFilled(data []byte, ci types.ClassifiedInstruction) *types.TradeInfo {
	layout, err := ParseJupiterLimitOrderV2TradeLayout(data[16:])
	if err != nil {
		return nil
	}
	event := layout.ToObject()

	order := p.orderInstruction(ci)
	if order == nil {
		return nil
	}
	accounts := p.Adapter.GetInstructionAccounts(order.Instruction)
	orderData := p.Adapter.GetInstructionData(order.Instruction)

	// Accounts per the limit_order_2 IDL:
	// fill_order: taker, maker, order, taker_input_mint_account,
	//   taker_output_mint_account, maker_output_mint_account, fee_account,
	//   order_input_mint_account, input_mint, input_token_program, output_mint, ...
	// flash_fill_order: taker, maker, order, input_mint_reserve,
	//   maker_output_mint_account, taker_output_mint_account, fee_account,
	//   input_token_program, output_mint, ...
	var maker, inputMint, outputMint, makerOutput, feeAccount string
	if parsers.MatchDiscriminator(orderData, constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER_V2.UNKNOWN) {
		if len(accounts) < 11 {
			return nil
		}
		maker, makerOutput, feeAccount = accounts[1], accounts[5], accounts[6]
		inputMint, outputMint = accounts[8], accounts[10]
	} else {
		if len(accounts) < 9 {
			return nil
		}
		maker, makerOutput, feeAccount = accounts[1], accounts[4], accounts[6]
		inputMint, outputMint = p.Adapter.GetSplTokenMint(accounts[3]), accounts[8]
		if tokenInfo, ok := p.Adapter.SPLTokenMap[accounts[3]]; ok && tokenInfo.Mint != "" {
			inputMint = tokenInfo.Mint
		}
	}
	if inputMint == "" || outputMint == "" {
		return nil
	}

	// The program pays taking_amount minus the order's fee (fee_bps) to the
	// maker and the fee to fee_account; both transfers are in the fill
	received, fee := p.fillPayouts(order, ci, outputMint, maker, makerOutput, feeAccount)
	outAmount := received
	if outAmount == nil {
		outAmount = new(big.Int).Set(event.TakingAmount)
		if fee != nil && fee.Cmp(outAmount) <= 0 {
			outAmount.Sub(outAmount, fee)
		}
	}

	inputDecimals := p.Adapter.GetTokenDecimals(inputMint)
	outputDecimals := p.Adapter.GetTokenDecimals(outputMint)

	trade := &types.TradeInfo{
		Type: utils.GetTradeType(inputMint, outputMint),
		InputToken: types.TokenInfo{
			Mint:      inputMint,
			Amount:    types.ConvertToUIAmount(event.MakingAmount, inputDecimals),
			AmountRaw: event.MakingAmount.String(),
			Decimals:  inputDecimals,
		},
		OutputToken: types.TokenInfo{
			Mint:      outputMint,
			Amount:    types.ConvertToUIAmount(outAmount, outputDecimals),
			AmountRaw: outAmount.String(),
			Decimals:  outputDecimals,
		},
		User:      maker,
		ProgramId: constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID,
		AMM:       p.getAMM(),
		Route:     p.DexInfo.Route,
		Slot:      p.Adapter.Slot(),
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
		Idx:       utils.FormatIdx(ci.OuterIndex, ci.InnerIndex),
	}
	if fee != nil && fee.Sign() > 0 {
		trade.Fee = &types.FeeInfo{
			Mint:      outputMint,
			Amount:    types.ConvertToUIAmount(fee, outputDecimals),
			AmountRaw: fee.String(),
			Decimals:  outputDecimals,
			Dex:       constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.Name,
			Type:      "protocol",
			Recipient: feeAccount,
		}
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// fillPayouts sums the output-mint transfers of a fill (between the order
// instruction and its event) to the maker's output account and to the fee
// account. A nil result means no such transfer was found.
func (p *JupiterLimitOrderV2Parser) fillPayouts(order *types.ClassifiedInstruction, event types.ClassifiedInstruction, outputMint, maker, makerOutput, feeAccount string) (*big.Int, *big.Int) {
	from := utils.FormatIdx(order.OuterIndex, order.InnerIndex)
	to := utils.FormatIdx(event.OuterIndex, event.InnerIndex)
	var received, fee *big.Int
	add := func(sum *big.Int, amount string) *big.Int {
		v, ok := new(big.Int).SetString(amount, 10)
		if !ok {
			return sum
		}
		if sum == nil {
			return v
		}
		return sum.Add(sum, v)
	}
	for _, t := range utils.SortedTransfers(p.TransferActions) {
		if outer, _ := utils.SplitIdx(t.Idx); outer != event.OuterIndex || utils.CompareIdx(t.Idx, from) <= 0 || utils.CompareIdx(t.Idx, to) >= 0 {
			continue
		}
		if t.Info.Mint != outputMint {
			continue
		}
		switch t.Info.Destination {
		case makerOutput, maker:
			received = add(received, t.Info.TokenAmount.Amount)
		case feeAccount:
			fee = add(fee, t.Info.TokenAmount.Amount)
		}
	}
	return received, fee
}

// getAMM gets the AMM name: the first AMM (in execution order) that moved tokens
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

// getTransferActionKeys returns the transfer action keys in execution order,
// so the AMM chosen from them does not depend on map iteration order
func (p *JupiterLimitOrderV2Parser) getTransferActionKeys() []string {
	return utils.SortedTransferKeys(p.TransferActions)
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

			idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)

			if len(data) >= 16 && bytes.Equal(data[:16], constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER_V2.CREATE_ORDER_EVENT) {
				transfers = append(transfers, p.parseInitializeOrder(data, ci, idx)...)
			} else if bytes.Equal(data[:8], constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER_V2.CANCEL_ORDER) {
				transfers = append(transfers, p.parseCancelOrder(ci.Instruction, ci.ProgramId, ci.OuterIndex, ci.InnerIndex, "cancelOrder")...)
			} else if bytes.Equal(data[:8], constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER_V2.CANCEL_DUST_ORDER) {
				// cancel_dust_order has the account layout of cancel_order
				transfers = append(transfers, p.parseCancelOrder(ci.Instruction, ci.ProgramId, ci.OuterIndex, ci.InnerIndex, "cancelDustOrder")...)
			}
		}
	}

	// Deduplicate transfers
	if len(transfers) > 1 {
		seen := make(map[string]bool)
		var unique []types.TransferData
		for _, t := range transfers {
			key := fmt.Sprintf("%s-%s-%s=%v", t.Idx, t.Signature, t.Info.Mint, t.IsFee)
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
func (p *JupiterLimitOrderV2Parser) parseInitializeOrder(data []byte, ci types.ClassifiedInstruction, idx string) []types.TransferData {
	var transfers []types.TransferData
	programId := ci.ProgramId

	order := p.orderInstruction(ci)
	if order == nil {
		return transfers
	}
	eventInstruction := order.Instruction

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
			SourceBalance:    balance.Post.Copy(),
			SourcePreBalance: balance.Pre.Copy(),
		},
		Idx:       idx,
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
	})

	return transfers
}

// parseCancelOrder parses cancel order instruction
func (p *JupiterLimitOrderV2Parser) parseCancelOrder(instruction interface{}, programId string, outerIndex int, innerIndex int, transferType string) []types.TransferData {
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

	idx := utils.FormatIdx(outerIndex, innerIndex)

	instTransfers := p.GetTransfersForInstruction(programId, outerIndex, innerIndex, nil)
	var transfer *types.TransferData
	for i := range instTransfers {
		if instTransfers[i].Info.Mint == mint {
			transfer = &instTransfers[i]
			break
		}
	}

	// The returned tokens: the instruction's transfer, else the lamports of
	// a SOL order's closed order and WSOL reserve accounts, else (no inner
	// instructions) the maker's balance change when it is a credit
	decimals := uint8(0)
	tokenAmount := balance.Change.Amount
	if transfer != nil {
		decimals = transfer.Info.TokenAmount.Decimals
		tokenAmount = transfer.Info.TokenAmount.Amount
	} else {
		decimals = p.Adapter.GetTokenDecimals(mint)
		if mint == constants.TOKENS.SOL {
			tokenAmount = closedAccountsLamports(p.Adapter, accounts[2], accounts[3]).String()
		}
		if change, ok := new(big.Int).SetString(tokenAmount, 10); !ok || change.Sign() <= 0 {
			tokenAmount = ""
		}
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

	if tokenAmount != "" {
		transfers = append(transfers, types.TransferData{
			Type:      transferType,
			ProgramId: programId,
			Info: types.TransferDataInfo{
				Authority:             authorityStr,
				Source:                sourceStr,
				Destination:           destinationStr,
				DestinationOwner:      p.Adapter.GetTokenAccountOwner(destination),
				Mint:                  mint,
				TokenAmount:           types.TokenAmount{Amount: tokenAmount, UIAmount: &uiAmount, Decimals: decimals},
				DestinationBalance:    balance.Post.Copy(),
				DestinationPreBalance: balance.Pre.Copy(),
			},
			Idx:       idx,
			Timestamp: p.Adapter.BlockTime(),
			Signature: p.Adapter.Signature(),
		})
	}

	// A token order's closed order and reserve accounts return their rent
	// to the maker: a refund of this instruction, not a fee
	if mint != constants.TOKENS.SOL {
		if refund := closedAccountsLamports(p.Adapter, accounts[2], accounts[3]); refund.Sign() > 0 {
			var post, pre *types.TokenAmount
			if solBalance := p.Adapter.GetAccountSolBalanceChanges(false)[user]; solBalance != nil {
				post, pre = solBalance.Post.Copy(), solBalance.Pre.Copy()
			}
			solUIAmount := types.ConvertToUIAmount(refund, 9)
			transfers = append(transfers, types.TransferData{
				Type:      transferType,
				ProgramId: programId,
				Info: types.TransferDataInfo{
					Authority:             authorityStr,
					Source:                sourceStr,
					Destination:           user,
					Mint:                  constants.TOKENS.SOL,
					TokenAmount:           types.TokenAmount{Amount: refund.String(), UIAmount: &solUIAmount, Decimals: 9},
					DestinationBalance:    post,
					DestinationPreBalance: pre,
				},
				Idx:       idx,
				Timestamp: p.Adapter.BlockTime(),
				Signature: p.Adapter.Signature(),
			})
		}
	}

	return transfers
}
