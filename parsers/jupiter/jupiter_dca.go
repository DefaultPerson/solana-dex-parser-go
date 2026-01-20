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
		Fee: &types.FeeInfo{
			Mint:      event.FeeMint,
			Amount:    feeUIAmount,
			AmountRaw: event.Fee.String(),
			Decimals:  feeDecimal,
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

// getTransferActionKeys returns all transfer action keys
func (p *JupiterDCAParser) getTransferActionKeys() []string {
	keys := make([]string, 0, len(p.TransferActions))
	for k := range p.TransferActions {
		keys = append(keys, k)
	}
	return keys
}

// ProcessTransfers parses DCA transfer operations
func (p *JupiterDCAParser) ProcessTransfers() []types.TransferData {
	var transfers []types.TransferData

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.JUPITER_DCA.ID {
			data := p.Adapter.GetInstructionData(ci.Instruction)
			if len(data) < 8 {
				continue
			}

			innerIdx := ci.InnerIndex
			if innerIdx < 0 {
				innerIdx = 0
			}
			idx := fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx)

			discriminator := data[:8]
			if bytes.Equal(discriminator, constants.DISCRIMINATORS.JUPITER_DCA.CLOSE_DCA) {
				transfers = append(transfers, p.parseCloseDCA(ci.Instruction, ci.ProgramId, idx)...)
			} else if bytes.Equal(discriminator, constants.DISCRIMINATORS.JUPITER_DCA.OPEN_DCA) ||
				bytes.Equal(discriminator, constants.DISCRIMINATORS.JUPITER_DCA.OPEN_DCA_V2) {
				transfers = append(transfers, p.parseOpenDCA(ci.Instruction, ci.ProgramId, idx)...)
			}
		}
	}

	return transfers
}

// parseCloseDCA parses close DCA instruction
func (p *JupiterDCAParser) parseCloseDCA(instruction interface{}, programId string, idx string) []types.TransferData {
	var transfers []types.TransferData

	user := p.Adapter.Signer()
	balanceChanges := p.Adapter.GetAccountSolBalanceChanges(false)
	balance, ok := balanceChanges[user]
	if !ok || balance == nil {
		return transfers
	}

	accounts := p.Adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 2 {
		return transfers
	}

	uiAmount := float64(0)
	if balance.Change.UIAmount != nil {
		uiAmount = *balance.Change.UIAmount
	}

	transfers = append(transfers, types.TransferData{
		Type:      "CloseDca",
		ProgramId: programId,
		Info: types.TransferDataInfo{
			Authority:        p.Adapter.GetTokenAccountOwner(accounts[1]),
			Destination:      user,
			DestinationOwner: p.Adapter.GetTokenAccountOwner(user),
			Mint:             constants.TOKENS.SOL,
			Source:           accounts[1],
			TokenAmount: types.TokenAmount{
				Amount:   balance.Change.Amount,
				UIAmount: &uiAmount,
				Decimals: balance.Change.Decimals,
			},
			DestinationBalance:    &balance.Post,
			DestinationPreBalance: &balance.Pre,
		},
		Idx:       idx,
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
	})

	return transfers
}

// parseOpenDCA parses open DCA instruction
func (p *JupiterDCAParser) parseOpenDCA(instruction interface{}, programId string, idx string) []types.TransferData {
	var transfers []types.TransferData

	user := p.Adapter.Signer()
	balanceChanges := p.Adapter.GetAccountSolBalanceChanges(false)
	balance, ok := balanceChanges[user]
	if !ok || balance == nil {
		return transfers
	}

	accounts := p.Adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 1 {
		return transfers
	}

	uiAmount := float64(0)
	if balance.Change.UIAmount != nil {
		uiAmount = *balance.Change.UIAmount
	}

	transfers = append(transfers, types.TransferData{
		Type:      "OpenDca",
		ProgramId: programId,
		Info: types.TransferDataInfo{
			Authority:        p.Adapter.GetTokenAccountOwner(user),
			Source:           user,
			Destination:      accounts[0],
			DestinationOwner: p.Adapter.GetTokenAccountOwner(accounts[0]),
			Mint:             constants.TOKENS.SOL,
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

// GetAMMs extracts AMM names from transfer action keys
func GetAMMs(keys []string) []string {
	return utils.GetAMMs(keys)
}
