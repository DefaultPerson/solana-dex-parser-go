package meme

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/types"
	"github.com/solana-dex-parser-go/utils"
)

// MoonitEventParser parses Moonit meme coin events
type MoonitEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
	utils           *utils.TransactionUtils
}

// NewMoonitEventParser creates a new Moonit event parser
func NewMoonitEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *MoonitEventParser {
	return &MoonitEventParser{
		adapter:         adapter,
		transferActions: transferActions,
		utils:           utils.NewTransactionUtils(adapter),
	}
}

// ProcessEvents implements the EventParser interface
func (p *MoonitEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForMultiPrograms(p.adapter, []string{
		constants.DEX_PROGRAMS.MOONIT.ID,
		constants.METAPLEX_PROGRAM_ID,
	})
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}

// ParseInstructions parses classified instructions into meme events
func (p *MoonitEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	for _, ci := range instructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		disc := data[:8]
		innerIdx := ci.InnerIndex
		if innerIdx < 0 {
			innerIdx = 0
		}

		var event *types.MemeEvent

		switch {
		case bytes.Equal(disc, constants.DISCRIMINATORS.MOONIT.BUY):
			event = p.decodeBuyEvent(data[8:], ci)
		case bytes.Equal(disc, constants.DISCRIMINATORS.MOONIT.SELL):
			event = p.decodeSellEvent(data[8:], ci)
		case bytes.Equal(disc, constants.DISCRIMINATORS.MOONIT.CREATE):
			event = p.decodeCreateEvent(data[8:], ci)
		case bytes.Equal(disc, constants.DISCRIMINATORS.MOONIT.MIGRATE):
			event = p.decodeMigrateEvent(data[8:], ci)
		}

		if event != nil {
			event.Signature = p.adapter.Signature()
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Idx = fmt.Sprintf("%d-%d", ci.OuterIndex, innerIdx)
			events = append(events, event)
		}
	}

	return events
}

func (p *MoonitEventParser) decodeBuyEvent(data []byte, ci types.ClassifiedInstruction) *types.MemeEvent {
	if len(data) < 16 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 13 {
		return nil
	}

	outputAmount, _ := reader.ReadU64()
	inputAmount, _ := reader.ReadU64()

	baseMint := accounts[6]
	pool := accounts[2]
	user := accounts[0]
	inputMint := constants.TOKENS.SOL
	outputMint := baseMint

	inputDecimals := p.adapter.GetTokenDecimals(inputMint)
	outputDecimals := p.adapter.GetTokenDecimals(outputMint)

	inputAmountBig := new(big.Int).SetUint64(inputAmount)
	outputAmountBig := new(big.Int).SetUint64(outputAmount)

	event := &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.MOONIT.Name,
		Type:         types.TradeTypeBuy,
		BaseMint:     outputMint,
		QuoteMint:    inputMint,
		BondingCurve: pool,
		Pool:         pool,
		User:         user,
		InputToken: &types.TokenInfo{
			Mint:      inputMint,
			AmountRaw: inputAmountBig.String(),
			Amount:    types.ConvertToUIAmount(inputAmountBig, inputDecimals),
			Decimals:  inputDecimals,
		},
		OutputToken: &types.TokenInfo{
			Mint:      outputMint,
			AmountRaw: outputAmountBig.String(),
			Amount:    types.ConvertToUIAmount(outputAmountBig, outputDecimals),
			Decimals:  outputDecimals,
		},
	}

	// Attach transfer data
	return p.processMemeTransferData(ci, event)
}

func (p *MoonitEventParser) decodeSellEvent(data []byte, ci types.ClassifiedInstruction) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 7 {
		return nil
	}

	user := accounts[0]
	pool := accounts[2]
	dexFeeMint := accounts[4]
	helioFeeMint := accounts[5]
	baseMint := accounts[6]

	collateralMint := p.detectCollateralMint(p.adapter.AccountKeys)
	tokenAmount, collateralAmount, dexFeeAmount, _ := p.calculateAmounts(baseMint, collateralMint, dexFeeMint, helioFeeMint)

	event := &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.MOONIT.Name,
		Type:         types.TradeTypeSell,
		BaseMint:     baseMint,
		QuoteMint:    collateralMint,
		BondingCurve: pool,
		Pool:         pool,
		User:         user,
		InputToken: &types.TokenInfo{
			Mint:      baseMint,
			AmountRaw: tokenAmount.Amount,
			Amount:    getUIAmount(tokenAmount.UIAmount),
			Decimals:  tokenAmount.Decimals,
		},
		OutputToken: &types.TokenInfo{
			Mint:      collateralMint,
			AmountRaw: collateralAmount.Amount,
			Amount:    getUIAmount(collateralAmount.UIAmount),
			Decimals:  collateralAmount.Decimals,
		},
	}

	// Add fee info
	if dexFeeAmount.Amount != "0" {
		feeAmt := getUIAmount(dexFeeAmount.UIAmount)
		event.ProtocolFee = &feeAmt
	}

	return event
}

func (p *MoonitEventParser) decodeCreateEvent(data []byte, ci types.ClassifiedInstruction) *types.MemeEvent {
	if len(data) < 10 {
		return nil
	}

	reader := utils.NewBinaryReader(data)
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 4 {
		return nil
	}

	name, _ := reader.ReadString()
	symbol, _ := reader.ReadString()
	uri, _ := reader.ReadString()
	decimals, _ := reader.ReadU8()
	reader.ReadU8() // skip
	totalSupply, _ := reader.ReadU64()

	pool := accounts[2]
	baseMint := accounts[3]
	user := accounts[0]

	totalSupplyFloat := types.ConvertToUIAmountUint64(totalSupply, decimals)

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.MOONIT.Name,
		Type:         types.TradeTypeCreate,
		Timestamp:    p.adapter.BlockTime(),
		Pool:         pool,
		BondingCurve: pool,
		User:         user,
		Creator:      user,
		BaseMint:     baseMint,
		QuoteMint:    constants.TOKENS.SOL,
		Name:         name,
		Symbol:       symbol,
		URI:          uri,
		Decimals:     &decimals,
		TotalSupply:  &totalSupplyFloat,
	}
}

func (p *MoonitEventParser) decodeMigrateEvent(data []byte, ci types.ClassifiedInstruction) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 6 {
		return nil
	}

	bondingCurve := accounts[2]
	baseMint := accounts[5]

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.MOONIT.Name,
		Type:         types.TradeTypeMigrate,
		Timestamp:    p.adapter.BlockTime(),
		BondingCurve: bondingCurve,
		BaseMint:     baseMint,
		QuoteMint:    constants.TOKENS.SOL,
	}
}

func (p *MoonitEventParser) detectCollateralMint(accountKeys []string) string {
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

func (p *MoonitEventParser) calculateAmounts(tokenMint, collateralMint, dexFeeMint, helioFeeMint string) (types.TokenAmount, types.TokenAmount, types.TokenAmount, types.TokenAmount) {
	tokenBalanceChange := p.getTokenBalanceChanges(tokenMint)
	collateralBalanceChange := p.getTokenBalanceChanges(collateralMint)
	dexFeeBalanceChange := p.getTokenBalanceChanges(dexFeeMint)
	helioFeeBalanceChange := p.getTokenBalanceChanges(helioFeeMint)

	return p.createTokenAmount(absInt64(tokenBalanceChange), tokenMint),
		p.createTokenAmount(absInt64(collateralBalanceChange), collateralMint),
		p.createTokenAmount(absInt64(dexFeeBalanceChange), dexFeeMint),
		p.createTokenAmount(absInt64(helioFeeBalanceChange), helioFeeMint)
}

func (p *MoonitEventParser) getTokenBalanceChanges(mint string) int64 {
	signer := p.adapter.Signer()

	if mint == constants.TOKENS.SOL {
		preBalances := p.adapter.PreBalances()
		postBalances := p.adapter.PostBalances()
		if len(preBalances) > 0 && len(postBalances) > 0 {
			return int64(postBalances[0]) - int64(preBalances[0])
		}
		return 0
	}

	var preAmount, postAmount int64

	for _, preBalance := range p.adapter.PreTokenBalances() {
		if preBalance.Mint == mint && preBalance.Owner == signer {
			if amt, ok := new(big.Int).SetString(preBalance.UiTokenAmount.Amount, 10); ok {
				preAmount = amt.Int64()
			}
		}
	}

	for _, postBalance := range p.adapter.PostTokenBalances() {
		if postBalance.Mint == mint && postBalance.Owner == signer {
			if amt, ok := new(big.Int).SetString(postBalance.UiTokenAmount.Amount, 10); ok {
				postAmount = amt.Int64()
			}
		}
	}

	return postAmount - preAmount
}

func (p *MoonitEventParser) createTokenAmount(amount int64, mint string) types.TokenAmount {
	decimals := p.adapter.GetTokenDecimals(mint)
	amtBig := new(big.Int).SetInt64(amount)
	uiAmount := types.ConvertToUIAmount(amtBig, decimals)
	return types.TokenAmount{
		Amount:   amtBig.String(),
		UIAmount: &uiAmount,
		Decimals: decimals,
	}
}

func (p *MoonitEventParser) processMemeTransferData(ci types.ClassifiedInstruction, event *types.MemeEvent) *types.MemeEvent {
	innerIdx := ci.InnerIndex
	if innerIdx < 0 {
		innerIdx = 0
	}

	key := fmt.Sprintf("%s:%d-%d", ci.ProgramId, ci.OuterIndex, innerIdx)
	transfers, ok := p.transferActions[key]
	if !ok || len(transfers) < 2 {
		return event
	}

	// Attach transfer info to input/output tokens
	for _, transfer := range transfers {
		if event.InputToken != nil && transfer.Info.Mint == event.InputToken.Mint {
			event.InputToken.Authority = transfer.Info.Authority
			event.InputToken.Source = transfer.Info.Source
			event.InputToken.Destination = transfer.Info.Destination
		}
		if event.OutputToken != nil && transfer.Info.Mint == event.OutputToken.Mint {
			event.OutputToken.Authority = transfer.Info.Authority
			event.OutputToken.Source = transfer.Info.Source
			event.OutputToken.Destination = transfer.Info.Destination
		}
	}

	return event
}

// getAllInstructionsForMultiPrograms gets all instructions for multiple program IDs
func getAllInstructionsForMultiPrograms(adapter *adapter.TransactionAdapter, programIds []string) []types.ClassifiedInstruction {
	var instructions []types.ClassifiedInstruction

	// Process outer instructions
	for i, ix := range adapter.Instructions() {
		programId := adapter.GetInstructionProgramId(ix)
		for _, pid := range programIds {
			if programId == pid {
				instructions = append(instructions, types.ClassifiedInstruction{
					ProgramId:   programId,
					Instruction: ix,
					OuterIndex:  i,
					InnerIndex:  -1,
				})
				break
			}
		}
	}

	// Process inner instructions
	for _, innerSet := range adapter.InnerInstructions() {
		for j, innerIx := range innerSet.Instructions {
			programId := adapter.GetInstructionProgramId(innerIx)
			for _, pid := range programIds {
				if programId == pid {
					instructions = append(instructions, types.ClassifiedInstruction{
						ProgramId:   programId,
						Instruction: innerIx,
						OuterIndex:  innerSet.Index,
						InnerIndex:  j,
					})
					break
				}
			}
		}
	}

	return instructions
}

// getUIAmount safely extracts float64 from *float64
func getUIAmount(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
