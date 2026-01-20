package jupiter

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
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

// ProcessTrades returns empty trades for limit order V1 (no immediate trades)
func (p *JupiterLimitOrderParser) ProcessTrades() []types.TradeInfo {
	return []types.TradeInfo{}
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
