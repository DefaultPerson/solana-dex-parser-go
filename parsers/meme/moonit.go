package meme

import (
	"bytes"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// MoonitParser parses Moonit (MoonShot) transactions
type MoonitParser struct {
	*parsers.BaseParser
}

// NewMoonitParser creates a new Moonit parser
func NewMoonitParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MoonitParser {
	return &MoonitParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// ProcessTrades parses Moonit trades
func (p *MoonitParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if p.isTradeInstruction(ci.Instruction, ci.ProgramId) {
			innerIdx := ci.InnerIndex
			if innerIdx < 0 {
				innerIdx = 0
			}
			idx := formatIdx(ci.OuterIndex, innerIdx)
			trade := p.parseTradeInstruction(ci.Instruction, idx)
			if trade != nil {
				trades = append(trades, *trade)
			}
		}
	}

	return trades
}

func (p *MoonitParser) isTradeInstruction(instruction interface{}, programId string) bool {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	return programId == constants.DEX_PROGRAMS.MOONIT.ID && len(accounts) == 11
}

func (p *MoonitParser) parseTradeInstruction(instruction interface{}, idx string) *types.TradeInfo {
	data := p.Adapter.GetInstructionData(instruction)
	if len(data) < 8 {
		return nil
	}

	disc := data[:8]
	var tradeType types.TradeType

	if bytes.Equal(disc, constants.DISCRIMINATORS.MOONIT.BUY) {
		tradeType = types.TradeTypeBuy
	} else if bytes.Equal(disc, constants.DISCRIMINATORS.MOONIT.SELL) {
		tradeType = types.TradeTypeSell
	} else {
		return nil
	}

	accounts := p.Adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 7 {
		return nil
	}

	moonitTokenMint := accounts[6]
	accountKeys := p.Adapter.AccountKeys
	collateralMint := p.detectCollateralMint(accountKeys)

	tokenAmount, collateralAmount := p.calculateAmounts(moonitTokenMint, collateralMint)

	var inputMint, outputMint string
	var inputAmt, outputAmt types.TokenAmount
	if tradeType == types.TradeTypeBuy {
		inputMint = collateralMint
		inputAmt = collateralAmount
		outputMint = moonitTokenMint
		outputAmt = tokenAmount
	} else {
		inputMint = moonitTokenMint
		inputAmt = tokenAmount
		outputMint = collateralMint
		outputAmt = collateralAmount
	}

	inputUIAmount := float64(0)
	if inputAmt.UIAmount != nil {
		inputUIAmount = *inputAmt.UIAmount
	}
	outputUIAmount := float64(0)
	if outputAmt.UIAmount != nil {
		outputUIAmount = *outputAmt.UIAmount
	}

	trade := &types.TradeInfo{
		Type: tradeType,
		Pool: []string{accountKeys[2]},
		InputToken: types.TokenInfo{
			Mint:      inputMint,
			Amount:    inputUIAmount,
			AmountRaw: inputAmt.Amount,
			Decimals:  inputAmt.Decimals,
		},
		OutputToken: types.TokenInfo{
			Mint:      outputMint,
			Amount:    outputUIAmount,
			AmountRaw: outputAmt.Amount,
			Decimals:  outputAmt.Decimals,
		},
		User:      p.Adapter.Signer(),
		ProgramId: constants.DEX_PROGRAMS.MOONIT.ID,
		AMM:       constants.DEX_PROGRAMS.MOONIT.Name,
		Route:     p.DexInfo.Route,
		Slot:      p.Adapter.Slot(),
		Timestamp: p.Adapter.BlockTime(),
		Signature: p.Adapter.Signature(),
		Idx:       idx,
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

func (p *MoonitParser) detectCollateralMint(accountKeys []string) string {
	for _, key := range accountKeys {
		if key == constants.TOKENS.USDC {
			return constants.TOKENS.USDC
		}
		if key == constants.TOKENS.USDT {
			return constants.TOKENS.USDT
		}
	}
	return constants.TOKENS.SOL
}

func (p *MoonitParser) calculateAmounts(tokenMint, collateralMint string) (types.TokenAmount, types.TokenAmount) {
	tokenBalanceChange := p.getTokenBalanceChanges(tokenMint)
	collateralBalanceChange := p.getTokenBalanceChanges(collateralMint)

	return p.createTokenAmount(absInt64(tokenBalanceChange), tokenMint),
		p.createTokenAmount(absInt64(collateralBalanceChange), collateralMint)
}

func (p *MoonitParser) getTokenBalanceChanges(mint string) int64 {
	signer := p.Adapter.Signer()

	if mint == constants.TOKENS.SOL {
		preBalances := p.Adapter.PreBalances()
		postBalances := p.Adapter.PostBalances()
		if len(preBalances) > 0 && len(postBalances) > 0 {
			return int64(postBalances[0]) - int64(preBalances[0])
		}
		return 0
	}

	var preAmount, postAmount int64

	for _, preBalance := range p.Adapter.PreTokenBalances() {
		if preBalance.Mint == mint && preBalance.Owner == signer {
			if amt, ok := new(big.Int).SetString(preBalance.UiTokenAmount.Amount, 10); ok {
				preAmount = amt.Int64()
			}
		}
	}

	for _, postBalance := range p.Adapter.PostTokenBalances() {
		if postBalance.Mint == mint && postBalance.Owner == signer {
			if amt, ok := new(big.Int).SetString(postBalance.UiTokenAmount.Amount, 10); ok {
				postAmount = amt.Int64()
			}
		}
	}

	return postAmount - preAmount
}

func (p *MoonitParser) createTokenAmount(amount int64, mint string) types.TokenAmount {
	decimals := p.Adapter.GetTokenDecimals(mint)
	amtBig := new(big.Int).SetInt64(amount)
	uiAmount := types.ConvertToUIAmount(amtBig, decimals)
	return types.TokenAmount{
		Amount:   amtBig.String(),
		UIAmount: &uiAmount,
		Decimals: decimals,
	}
}

func absInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
