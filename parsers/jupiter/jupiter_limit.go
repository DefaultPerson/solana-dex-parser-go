package jupiter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// JupiterLimitOrderParser parses Jupiter Limit Order transactions (V1)
type JupiterLimitOrderParser struct {
	*parsers.BaseParser
}

// NewJupiterLimitOrderParser creates a new Jupiter Limit Order parser
func NewJupiterLimitOrderParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *JupiterLimitOrderParser {
	return &JupiterLimitOrderParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// ProcessTrades parses Limit Order v1 fills (flash_fill_order; legacy: the
// program no longer fills orders, the parser serves historical transactions).
// The user is the order's maker: the maker sold the making_amount of the input
// mint that the paired pre_flash_fill_order took from the order's reserve, and
// received the output the fill paid to the maker's output account, after the
// program fee (the Fee). The v1 TradeEvent is only logged ("Program data"), so
// the amounts come from the instructions and their transfers.
func (p *JupiterLimitOrderParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo
	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER.ID {
			continue
		}
		data := p.Adapter.GetInstructionData(ci.Instruction)
		if len(data) >= 8 && bytes.Equal(data[:8], constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER.FLASH_FILL_ORDER) {
			if trade := p.parseFlashFill(ci); trade != nil {
				trades = append(trades, *trade)
			}
		}
	}
	return trades
}

// parseFlashFill parses a flash_fill_order. Accounts per the v1 IDL: order,
// reserve, maker, taker, maker_output_account, taker_input_account,
// fee_authority, program_fee_account, referral, input_mint,
// input_mint_token_program, output_mint, ...
func (p *JupiterLimitOrderParser) parseFlashFill(ci types.ClassifiedInstruction) *types.TradeInfo {
	accounts := p.Adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 12 {
		return nil
	}
	order, maker, makerOutput, feeAccount := accounts[0], accounts[2], accounts[4], accounts[7]
	inputMint, outputMint := accounts[9], accounts[11]

	making := p.preFlashFillMaking(order, ci)
	if making == nil {
		return nil
	}

	// The fill pays the maker (maker_output_account, or the maker itself for
	// native SOL) and the program fee account in the output mint
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
	for _, t := range p.GetTransfersForInstruction(ci.ProgramId, ci.OuterIndex, ci.InnerIndex, nil) {
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
	if received == nil {
		return nil
	}

	inputDecimals := p.Adapter.GetTokenDecimals(inputMint)
	outputDecimals := p.Adapter.GetTokenDecimals(outputMint)
	trade := &types.TradeInfo{
		Type: utils.GetTradeType(inputMint, outputMint),
		InputToken: types.TokenInfo{
			Mint:      inputMint,
			Amount:    types.ConvertToUIAmount(making, inputDecimals),
			AmountRaw: making.String(),
			Decimals:  inputDecimals,
		},
		OutputToken: types.TokenInfo{
			Mint:      outputMint,
			Amount:    types.ConvertToUIAmount(received, outputDecimals),
			AmountRaw: received.String(),
			Decimals:  outputDecimals,
		},
		User:      maker,
		ProgramId: constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER.ID,
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
			Dex:       constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER.Name,
			Type:      "protocol",
			Recipient: feeAccount,
		}
	}
	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// preFlashFillMaking returns the making_amount argument of the last
// pre_flash_fill_order of order before the flash fill ci, or nil
func (p *JupiterLimitOrderParser) preFlashFillMaking(order string, ci types.ClassifiedInstruction) *big.Int {
	fillIdx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	var making *big.Int
	for _, pre := range p.ClassifiedInstructions {
		if pre.ProgramId != ci.ProgramId || utils.CompareIdx(utils.FormatIdx(pre.OuterIndex, pre.InnerIndex), fillIdx) >= 0 {
			continue
		}
		data := p.Adapter.GetInstructionData(pre.Instruction)
		accounts := p.Adapter.GetInstructionAccounts(pre.Instruction)
		if len(data) < 16 || !bytes.Equal(data[:8], constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER.PRE_FLASH_FILL_ORDER) || len(accounts) == 0 || accounts[0] != order {
			continue
		}
		making = new(big.Int).SetUint64(binary.LittleEndian.Uint64(data[8:16]))
	}
	return making
}

// getAMM returns the first AMM (in execution order) that moved tokens, i.e.
// the venue of the keeper's route, else the program name
func (p *JupiterLimitOrderParser) getAMM() string {
	if amms := utils.GetAMMs(utils.SortedTransferKeys(p.TransferActions)); len(amms) > 0 {
		return amms[0]
	}
	return constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER.Name
}

// ProcessTransfers parses limit order transfer operations
func (p *JupiterLimitOrderParser) ProcessTransfers() []types.TransferData {
	var transfers []types.TransferData

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER.ID {
			data := p.Adapter.GetInstructionData(ci.Instruction)
			if len(data) < 8 {
				continue
			}

			discriminator := data[:8]
			innerIdx := ci.InnerIndex
			if bytes.Equal(discriminator, constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER.CREATE_ORDER) {
				transfers = append(transfers, p.parseInitializeOrder(ci.Instruction, ci.ProgramId, ci.OuterIndex, innerIdx)...)
			} else if bytes.Equal(discriminator, constants.DISCRIMINATORS.JUPITER_LIMIT_ORDER.CANCEL_ORDER) {
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

// parseInitializeOrder parses initialize order instruction
func (p *JupiterLimitOrderParser) parseInitializeOrder(instruction interface{}, programId string, outerIndex int, innerIndex int) []types.TransferData {
	var transfers []types.TransferData

	accounts := p.Adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 6 {
		return transfers
	}

	user := accounts[1]
	mint := accounts[5]
	source := accounts[4]
	destination := accounts[3]
	if mint == constants.TOKENS.SOL {
		destination = user
	}

	var balance *types.BalanceChange
	if mint == constants.TOKENS.SOL {
		balanceChanges := p.Adapter.GetAccountSolBalanceChanges(true)
		balance = balanceChanges[user]
	} else {
		tokenChanges := p.Adapter.GetAccountTokenBalanceChanges(false)
		if userTokens, ok := tokenChanges[source]; ok {
			balance = userTokens[mint]
		}
	}

	solBalanceChanges := p.Adapter.GetAccountSolBalanceChanges(true)
	solBalance := solBalanceChanges[user]

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

	solBalanceChange := "0"
	if solBalance != nil {
		solBalanceChange = solBalance.Change.Amount
	}

	transfers = append(transfers, types.TransferData{
		Type:      "initializeOrder",
		ProgramId: programId,
		Info: types.TransferDataInfo{
			Authority:        p.Adapter.GetTokenAccountOwner(source),
			Source:           source,
			Destination:      destination,
			DestinationOwner: p.Adapter.GetTokenAccountOwner(source),
			Mint:             mint,
			TokenAmount: types.TokenAmount{
				Amount:   tokenAmount,
				UIAmount: &uiAmount,
				Decimals: decimals,
			},
			SourceBalance:    &balance.Post,
			SourcePreBalance: &balance.Pre,
			SolBalanceChange: solBalanceChange,
		},
		Idx:       idx,
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
	})

	return transfers
}

// parseCancelOrder parses cancel order instruction
func (p *JupiterLimitOrderParser) parseCancelOrder(instruction interface{}, programId string, outerIndex int, innerIndex int) []types.TransferData {
	var transfers []types.TransferData

	accounts := p.Adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 7 {
		return transfers
	}

	user := accounts[2]
	mint := accounts[6]
	source := accounts[1]
	authority := accounts[0]
	destination := accounts[3]
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
