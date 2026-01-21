package utils

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// TransactionUtils provides utility functions for transaction processing
type TransactionUtils struct {
	adapter *adapter.TransactionAdapter
}

// NewTransactionUtils creates a new TransactionUtils instance
func NewTransactionUtils(adapter *adapter.TransactionAdapter) *TransactionUtils {
	return &TransactionUtils{adapter: adapter}
}

// GetDexInfo extracts DEX information from transaction
func (tu *TransactionUtils) GetDexInfo(classifier *classifier.InstructionClassifier) types.DexInfo {
	programIds := classifier.GetAllProgramIds()
	if len(programIds) == 0 {
		return types.DexInfo{}
	}

	info := types.DexInfo{}

	for _, programId := range programIds {
		prog := constants.GetDexProgramByID(programId)
		if prog.Name == "" {
			continue
		}

		hasAmmTag := false
		for _, tag := range prog.Tags {
			if tag == "amm" {
				hasAmmTag = true
				break
			}
		}

		if hasAmmTag {
			if info.AMM == "" {
				info.AMM = prog.Name
				if info.ProgramId == "" {
					info.ProgramId = prog.ID
				}
			}
		} else {
			if info.Route == "" {
				info.Route = prog.Name
				info.ProgramId = prog.ID
			}
		}
	}

	if info.ProgramId == "" && len(programIds) > 0 {
		info.ProgramId = programIds[0]
	}

	return info
}

// GetTransferActions extracts transfer actions from transaction
func (tu *TransactionUtils) GetTransferActions(extraTypes []string) map[string][]types.TransferData {
	actions := make(map[string][]types.TransferData)
	innerInstructions := tu.adapter.InnerInstructions()

	groupKey := ""

	// Process transfers of program instructions
	for _, set := range innerInstructions {
		outerIndex := set.Index
		outerInstruction := tu.adapter.Instructions()[outerIndex]
		outerProgramId := tu.adapter.GetInstructionProgramId(outerInstruction)

		if constants.IsSystemProgram(outerProgramId) {
			continue
		}
		groupKey = FormatTransferKey(outerProgramId, outerIndex, -1)

		for innerIndex, ix := range set.Instructions {
			innerProgramId := tu.adapter.GetInstructionProgramId(ix)

			// Special case for meteora vault
			if !constants.IsSystemProgram(innerProgramId) && !tu.isIgnoredProgram(innerProgramId) {
				groupKey = FormatTransferKey(innerProgramId, outerIndex, innerIndex)
				continue
			}

			idx := FormatIdx(outerIndex, innerIndex)
			transferData := tu.ParseInstructionAction(ix, idx, extraTypes)
			if transferData != nil {
				if constants.IsFeeAccount(transferData.Info.Destination) ||
					constants.IsFeeAccount(transferData.Info.DestinationOwner) {
					transferData.IsFee = true
				}
				actions[groupKey] = append(actions[groupKey], *transferData)
			}
		}
	}

	// Process transfers without program
	groupKey = "transfer"
	for outerIndex, ix := range tu.adapter.Instructions() {
		idx := strconv.Itoa(outerIndex)
		transferData := tu.ParseInstructionAction(ix, idx, extraTypes)
		if transferData != nil {
			actions[groupKey] = append(actions[groupKey], *transferData)
		}
	}

	return actions
}

// ParseInstructionAction parses instruction action for transfers
func (tu *TransactionUtils) ParseInstructionAction(instruction interface{}, idx string, extraTypes []string) *types.TransferData {
	ix := tu.adapter.GetInstruction(instruction)
	if ix == nil {
		return nil
	}

	// Handle parsed instruction
	if ix.Parsed != nil {
		return tu.parseParsedInstructionAction(ix, idx, extraTypes)
	}

	// Handle compiled instruction
	return tu.parseCompiledInstructionAction(ix, idx, extraTypes)
}

// parseParsedInstructionAction parses a parsed instruction
func (tu *TransactionUtils) parseParsedInstructionAction(ix *adapter.UnifiedInstruction, idx string, extraTypes []string) *types.TransferData {
	if IsTransfer(ix) {
		return ProcessTransfer(ix, idx, tu.adapter)
	}
	if IsNativeTransfer(ix) {
		return ProcessNativeTransfer(ix, idx, tu.adapter)
	}
	if IsTransferCheck(ix) {
		return ProcessTransferCheck(ix, idx, tu.adapter)
	}
	if extraTypes != nil {
		for _, actionType := range extraTypes {
			if IsExtraAction(ix, actionType) {
				return ProcessExtraAction(ix, idx, tu.adapter, actionType)
			}
		}
	}
	return nil
}

// parseCompiledInstructionAction parses a compiled instruction
func (tu *TransactionUtils) parseCompiledInstructionAction(ix *adapter.UnifiedInstruction, idx string, extraTypes []string) *types.TransferData {
	if IsCompiledTransfer(ix) {
		return ProcessCompiledTransfer(ix, idx, tu.adapter)
	}
	if IsCompiledNativeTransfer(ix) {
		return ProcessCompiledNativeTransfer(ix, idx, tu.adapter)
	}
	if IsCompiledTransferCheck(ix) {
		return ProcessCompiledTransferCheck(ix, idx, tu.adapter)
	}
	if extraTypes != nil {
		for _, actionType := range extraTypes {
			if IsCompiledExtraAction(ix, actionType) {
				return ProcessCompiledExtraAction(ix, idx, tu.adapter, actionType)
			}
		}
	}
	return nil
}

// isIgnoredProgram checks if program should be ignored for grouping
func (tu *TransactionUtils) isIgnoredProgram(programId string) bool {
	for _, p := range constants.SKIP_PROGRAM_IDS {
		if p == programId {
			return true
		}
	}
	// Check vault programs
	vaultPrograms := []string{
		constants.DEX_PROGRAMS.METEORA_VAULT.ID,
		constants.DEX_PROGRAMS.STABBEL_VAULT.ID,
		constants.DEX_PROGRAMS.HEAVEN_VAULT.ID,
	}
	for _, p := range vaultPrograms {
		if p == programId {
			return true
		}
	}
	return false
}

// ProcessSwapData processes swap data from transfers
func (tu *TransactionUtils) ProcessSwapData(transfers []types.TransferData, dexInfo types.DexInfo, skipNative bool) *types.TradeInfo {
	if len(transfers) == 0 {
		return nil
	}

	uniqueTokens := tu.extractUniqueTokens(transfers, skipNative)
	if len(uniqueTokens) < 2 {
		return nil
	}

	signer := tu.getSwapSigner()
	inputToken, outputToken, feeTransfer := tu.calculateTokenAmounts(signer, transfers, uniqueTokens)

	trade := &types.TradeInfo{
		Type:        GetTradeType(inputToken.Mint, outputToken.Mint),
		InputToken:  inputToken,
		OutputToken: outputToken,
		User:        signer,
		ProgramId:   dexInfo.ProgramId,
		AMM:         dexInfo.AMM,
		Route:       dexInfo.Route,
		Slot:        tu.adapter.Slot(),
		Timestamp:   tu.adapter.BlockTime(),
		Signature:   tu.adapter.Signature(),
		Idx:         transfers[0].Idx,
	}

	if feeTransfer != nil {
		trade.Fee = &types.FeeInfo{
			Mint:      feeTransfer.Info.Mint,
			Amount:    *feeTransfer.Info.TokenAmount.UIAmount,
			AmountRaw: feeTransfer.Info.TokenAmount.Amount,
			Decimals:  feeTransfer.Info.TokenAmount.Decimals,
		}
	}

	return trade
}

// getSwapSigner gets the signer for swap transaction
func (tu *TransactionUtils) getSwapSigner() string {
	defaultSigner := tu.adapter.Signer()

	// Check for Jupiter DCA program
	for _, key := range tu.adapter.AccountKeys {
		if key == constants.DEX_PROGRAMS.JUPITER_DCA.ID {
			if len(tu.adapter.AccountKeys) > 2 {
				return tu.adapter.AccountKeys[2]
			}
		}
	}

	return defaultSigner
}

// extractUniqueTokens extracts unique tokens from transfers
func (tu *TransactionUtils) extractUniqueTokens(transfers []types.TransferData, skipNative bool) []types.TokenInfo {
	var uniqueTokens []types.TokenInfo
	seenTokens := make(map[string]bool)

	for _, transfer := range transfers {
		if skipNative && transfer.Info.Mint == constants.TOKENS.NATIVE {
			continue
		}
		tokenInfo := tu.GetTransferTokenInfo(&transfer)
		if tokenInfo != nil && !seenTokens[tokenInfo.Mint] {
			uniqueTokens = append(uniqueTokens, *tokenInfo)
			seenTokens[tokenInfo.Mint] = true
		}
	}

	return uniqueTokens
}

// calculateTokenAmounts calculates token amounts for swap
func (tu *TransactionUtils) calculateTokenAmounts(signer string, transfers []types.TransferData, uniqueTokens []types.TokenInfo) (types.TokenInfo, types.TokenInfo, *types.TransferData) {
	inputToken := uniqueTokens[0]
	outputToken := uniqueTokens[len(uniqueTokens)-1]

	// Check if tokens should be swapped
	if (outputToken.Source == signer || outputToken.Authority == signer) ||
		(outputToken.Source == constants.DEX_PROGRAMS.OKX_ROUTER.ID || outputToken.Authority == constants.DEX_PROGRAMS.OKX_ROUTER.ID) {
		inputToken, outputToken = outputToken, inputToken
	}

	inputAmount, inputAmountRaw, outputAmount, outputAmountRaw, feeTransfer := tu.sumTokenAmounts(transfers, inputToken.Mint, outputToken.Mint, signer)

	inputToken.Amount = inputAmount
	inputToken.AmountRaw = inputAmountRaw.String()
	outputToken.Amount = outputAmount
	outputToken.AmountRaw = outputAmountRaw.String()

	return inputToken, outputToken, feeTransfer
}

// sumTokenAmounts sums token amounts from transfers
func (tu *TransactionUtils) sumTokenAmounts(transfers []types.TransferData, inputMint, outputMint, signer string) (float64, *big.Int, float64, *big.Int, *types.TransferData) {
	seenTransfers := make(map[string]bool)
	var inputAmount, outputAmount float64
	inputAmountRaw := big.NewInt(0)
	outputAmountRaw := big.NewInt(0)
	var feeTransfer *types.TransferData

	for i := range transfers {
		transfer := &transfers[i]
		tokenInfo := tu.GetTransferTokenInfo(transfer)
		if tokenInfo == nil {
			continue
		}

		destination := tokenInfo.DestinationOwner
		if destination == "" {
			destination = tokenInfo.Destination
		}
		if constants.IsFeeAccount(destination) {
			feeTransfer = transfer
			continue
		}
		if tokenInfo.Authority == constants.DEX_PROGRAMS.OKX_ROUTER.ID && destination == signer {
			continue
		}

		key := tokenInfo.AmountRaw + "-" + tokenInfo.Mint
		if seenTransfers[key] {
			continue
		}
		seenTransfers[key] = true

		if tokenInfo.Mint == inputMint {
			inputAmount += tokenInfo.Amount
			amt, _ := new(big.Int).SetString(tokenInfo.AmountRaw, 10)
			inputAmountRaw.Add(inputAmountRaw, amt)
		}
		if tokenInfo.Mint == outputMint {
			outputAmount += tokenInfo.Amount
			amt, _ := new(big.Int).SetString(tokenInfo.AmountRaw, 10)
			outputAmountRaw.Add(outputAmountRaw, amt)
		}
	}

	return inputAmount, inputAmountRaw, outputAmount, outputAmountRaw, feeTransfer
}

// GetTransferTokenInfo gets token info from transfer data
func (tu *TransactionUtils) GetTransferTokenInfo(transfer *types.TransferData) *types.TokenInfo {
	if transfer == nil {
		return nil
	}
	uiAmount := float64(0)
	if transfer.Info.TokenAmount.UIAmount != nil {
		uiAmount = *transfer.Info.TokenAmount.UIAmount
	}
	return &types.TokenInfo{
		Mint:                  transfer.Info.Mint,
		Amount:                uiAmount,
		AmountRaw:             transfer.Info.TokenAmount.Amount,
		Decimals:              transfer.Info.TokenAmount.Decimals,
		Authority:             transfer.Info.Authority,
		Destination:           transfer.Info.Destination,
		DestinationOwner:      transfer.Info.DestinationOwner,
		DestinationBalance:    transfer.Info.DestinationBalance,
		DestinationPreBalance: transfer.Info.DestinationPreBalance,
		Source:                transfer.Info.Source,
		SourceBalance:         transfer.Info.SourceBalance,
		SourcePreBalance:      transfer.Info.SourcePreBalance,
	}
}

// GetLPTransfers sorts and gets LP tokens
func (tu *TransactionUtils) GetLPTransfers(transfers []types.TransferData) []types.TransferData {
	var tokens []types.TransferData
	for _, t := range transfers {
		if strings.Contains(t.Type, "transfer") {
			tokens = append(tokens, t)
		}
	}
	if len(tokens) >= 2 {
		if tokens[0].Info.Mint == constants.TOKENS.SOL ||
			(tu.adapter.IsSupportedToken(tokens[0].Info.Mint) && !tu.adapter.IsSupportedToken(tokens[1].Info.Mint)) {
			return []types.TransferData{tokens[1], tokens[0]}
		}
	}
	return tokens
}

// AttachTokenTransferInfo attaches token transfer info to trade
func (tu *TransactionUtils) AttachTokenTransferInfo(trade *types.TradeInfo, transferActions map[string][]types.TransferData) *types.TradeInfo {
	if trade == nil {
		return nil
	}

	// Find input and output transfers
	var inputTransfer, outputTransfer *types.TransferData
	for _, transfers := range transferActions {
		for i := range transfers {
			t := &transfers[i]
			if t.Info.Mint == trade.InputToken.Mint && t.Info.TokenAmount.Amount == trade.InputToken.AmountRaw {
				inputTransfer = t
			}
			if t.Info.Mint == trade.OutputToken.Mint && t.Info.TokenAmount.Amount == trade.OutputToken.AmountRaw {
				outputTransfer = t
			}
		}
	}

	solChanges := tu.adapter.GetAccountSolBalanceChanges(false)
	tokenChanges := tu.adapter.GetAccountTokenBalanceChanges(true)

	// Input token balance change
	var inputAmt *types.BalanceChange
	if trade.InputToken.Mint == constants.TOKENS.SOL {
		inputAmt = solChanges[trade.User]
	} else if userTokens, ok := tokenChanges[trade.User]; ok {
		inputAmt = userTokens[trade.InputToken.Mint]
	}

	// Output token balance change
	var outputAmt *types.BalanceChange
	if trade.OutputToken.Mint == constants.TOKENS.SOL {
		outputAmt = solChanges[trade.User]
	} else if userTokens, ok := tokenChanges[trade.User]; ok {
		outputAmt = userTokens[trade.OutputToken.Mint]
	}

	// Set balance changes
	if inputAmt != nil && inputAmt.Change.Amount != "" {
		trade.InputToken.BalanceChange = strings.TrimPrefix(inputAmt.Change.Amount, "-")
	} else {
		trade.InputToken.BalanceChange = trade.InputToken.AmountRaw
	}

	if outputAmt != nil && outputAmt.Change.Amount != "" {
		trade.OutputToken.BalanceChange = outputAmt.Change.Amount
	} else {
		trade.OutputToken.BalanceChange = trade.OutputToken.AmountRaw
	}

	// Attach transfer info
	if inputTransfer != nil {
		trade.InputToken.Authority = inputTransfer.Info.Authority
		trade.InputToken.Source = inputTransfer.Info.Source
		trade.InputToken.Destination = inputTransfer.Info.Destination
		trade.InputToken.DestinationOwner = inputTransfer.Info.DestinationOwner
		trade.InputToken.DestinationBalance = inputTransfer.Info.DestinationBalance
		trade.InputToken.DestinationPreBalance = inputTransfer.Info.DestinationPreBalance
		trade.InputToken.SourceBalance = inputTransfer.Info.SourceBalance
		trade.InputToken.SourcePreBalance = inputTransfer.Info.SourcePreBalance
	} else if inputAmt != nil {
		trade.InputToken.SourceBalance = &inputAmt.Post
		trade.InputToken.SourcePreBalance = &inputAmt.Pre
	}

	if outputTransfer != nil {
		trade.OutputToken.Authority = outputTransfer.Info.Authority
		trade.OutputToken.Source = outputTransfer.Info.Source
		trade.OutputToken.Destination = outputTransfer.Info.Destination
		trade.OutputToken.DestinationOwner = outputTransfer.Info.DestinationOwner
		trade.OutputToken.DestinationBalance = outputTransfer.Info.DestinationBalance
		trade.OutputToken.DestinationPreBalance = outputTransfer.Info.DestinationPreBalance
		trade.OutputToken.SourceBalance = outputTransfer.Info.SourceBalance
		trade.OutputToken.SourcePreBalance = outputTransfer.Info.SourcePreBalance
	} else if outputAmt != nil {
		trade.OutputToken.DestinationBalance = &outputAmt.Post
		trade.OutputToken.DestinationPreBalance = &outputAmt.Pre
	}

	trade.Signer = tu.adapter.Signers()

	return trade
}

// AttachUserBalanceToLPs attaches user balance changes to liquidities
func (tu *TransactionUtils) AttachUserBalanceToLPs(liquidities []types.PoolEvent) []types.PoolEvent {
	for i := range liquidities {
		lp := &liquidities[i]
		solChanges := tu.adapter.GetAccountSolBalanceChanges(false)
		tokenChanges := tu.adapter.GetAccountTokenBalanceChanges(true)

		solAmt := solChanges[lp.User]

		var token0Amt, token1Amt *types.BalanceChange
		if lp.Token0Mint != "" && lp.Token0Mint == constants.TOKENS.SOL {
			token0Amt = solAmt
		} else if lp.Token0Mint != "" {
			if userTokens, ok := tokenChanges[lp.User]; ok {
				token0Amt = userTokens[lp.Token0Mint]
			}
		}

		if lp.Token1Mint != "" && lp.Token1Mint == constants.TOKENS.SOL {
			token1Amt = solAmt
		} else if lp.Token1Mint != "" {
			if userTokens, ok := tokenChanges[lp.User]; ok {
				token1Amt = userTokens[lp.Token1Mint]
			}
		}

		if token0Amt != nil && token0Amt.Change.Amount != "" {
			lp.Token0BalanceChange = token0Amt.Change.Amount
		} else if lp.Token0AmountRaw != "" {
			lp.Token0BalanceChange = lp.Token0AmountRaw
		}

		if token1Amt != nil && token1Amt.Change.Amount != "" {
			lp.Token1BalanceChange = token1Amt.Change.Amount
		} else if lp.Token1AmountRaw != "" {
			lp.Token1BalanceChange = lp.Token1AmountRaw
		}

		lp.Signer = tu.adapter.Signers()
	}

	return liquidities
}

// AttachTradeFee attaches fee information to trade
func (tu *TransactionUtils) AttachTradeFee(trade *types.TradeInfo) *types.TradeInfo {
	if trade == nil {
		return nil
	}

	if trade.Fee == nil {
		mint := trade.OutputToken.Mint

		var token *types.BalanceChange
		if mint == constants.TOKENS.SOL {
			token = tu.adapter.GetAccountSolBalanceChanges(true)[trade.User]
		} else {
			if userTokens, ok := tu.adapter.GetAccountTokenBalanceChanges(true)[trade.User]; ok {
				token = userTokens[mint]
			}
		}

		if token != nil {
			outputAmount, _ := new(big.Int).SetString(trade.OutputToken.AmountRaw, 10)
			changeAmount, _ := new(big.Int).SetString(token.Change.Amount, 10)
			feeAmount := new(big.Int).Sub(outputAmount, changeAmount)

			if feeAmount.Sign() > 0 {
				feeUIAmount := types.ConvertToUIAmount(feeAmount, trade.OutputToken.Decimals)
				trade.Fee = &types.FeeInfo{
					Mint:      mint,
					Amount:    feeUIAmount,
					AmountRaw: feeAmount.String(),
					Decimals:  trade.OutputToken.Decimals,
				}
				trade.OutputToken.BalanceChange = token.Change.Amount
			}
		}
	}

	if trade.InputToken.Mint == constants.TOKENS.SOL {
		token := tu.adapter.GetAccountSolBalanceChanges(true)[trade.User]
		if token != nil {
			if token.Change.UIAmount != nil {
				changeAbs := *token.Change.UIAmount
				if changeAbs < 0 {
					changeAbs = -changeAbs
				}
				if changeAbs > trade.InputToken.Amount {
					trade.InputToken.BalanceChange = token.Change.Amount
				}
			}
		}
	}

	return trade
}

// GetTransfersForInstruction gets transfers for a specific instruction
func (tu *TransactionUtils) GetTransfersForInstruction(transferActions map[string][]types.TransferData, programId string, outerIndex int, innerIndex int, extraTypes []string) []types.TransferData {
	defaultTypes := []string{"transfer", "transferChecked"}
	if extraTypes != nil {
		defaultTypes = append(defaultTypes, extraTypes...)
	}
	return tu.FilterTransfersForInstruction(transferActions, programId, outerIndex, innerIndex, defaultTypes)
}

// FilterTransfersForInstruction filters transfers for a specific instruction
func (tu *TransactionUtils) FilterTransfersForInstruction(transferActions map[string][]types.TransferData, programId string, outerIndex int, innerIndex int, filterTypes []string) []types.TransferData {
	key := FormatTransferKey(programId, outerIndex, innerIndex)

	transfers, ok := transferActions[key]
	if !ok {
		return nil
	}

	if len(filterTypes) == 0 {
		return transfers
	}

	var result []types.TransferData
	for _, transfer := range transfers {
		for _, filterType := range filterTypes {
			if transfer.Type == filterType {
				result = append(result, transfer)
				break
			}
		}
	}

	return result
}

// ProcessTransferInstructions processes transfer instructions for an outer index
func (tu *TransactionUtils) ProcessTransferInstructions(outerIndex int, extraTypes []string) []types.TransferData {
	innerInstructions := tu.adapter.InnerInstructions()
	if len(innerInstructions) == 0 {
		return nil
	}

	var result []types.TransferData
	for _, set := range innerInstructions {
		if set.Index != outerIndex {
			continue
		}
		for idx, instruction := range set.Instructions {
			idxStr := FormatIdx(outerIndex, idx)
			transferData := tu.ParseInstructionAction(instruction, idxStr, extraTypes)
			if transferData != nil {
				result = append(result, *transferData)
			}
		}
	}

	return result
}

// GetAdapter returns the underlying adapter
func (tu *TransactionUtils) GetAdapter() *adapter.TransactionAdapter {
	return tu.adapter
}

// GetTransferInfo converts TransferData to TransferInfo format
func (tu *TransactionUtils) GetTransferInfo(transferData types.TransferData, timestamp int64, signature string) *types.TransferInfo {
	info := transferData.Info
	if info.TokenAmount.Amount == "" {
		return nil
	}

	var uiAmount float64
	if info.TokenAmount.UIAmount != nil {
		uiAmount = *info.TokenAmount.UIAmount
	}

	tokenInfo := types.TokenInfo{
		Mint:      info.Mint,
		Amount:    uiAmount,
		AmountRaw: info.TokenAmount.Amount,
		Decimals:  info.TokenAmount.Decimals,
	}

	transferType := "TRANSFER_IN"
	if info.Source == info.Authority {
		transferType = "TRANSFER_OUT"
	}

	return &types.TransferInfo{
		Type:      transferType,
		Token:     tokenInfo,
		From:      info.Source,
		To:        info.Destination,
		Timestamp: timestamp,
		Signature: signature,
	}
}

// GetTransferInfoList converts a list of TransferData to TransferInfo list
func (tu *TransactionUtils) GetTransferInfoList(transferDataList []types.TransferData) []types.TransferInfo {
	timestamp := tu.adapter.BlockTime()
	signature := tu.adapter.Signature()

	var result []types.TransferInfo
	for _, data := range transferDataList {
		info := tu.GetTransferInfo(data, timestamp, signature)
		if info != nil {
			result = append(result, *info)
		}
	}
	return result
}

// ProcessMemeTransferData processes transfer data for meme token events
func (tu *TransactionUtils) ProcessMemeTransferData(
	ci types.ClassifiedInstruction,
	event *types.MemeEvent,
	baseMint string,
	skipNative bool,
	transferStartIdx int,
	transferActions map[string][]types.TransferData,
) *types.MemeEvent {
	transfers := tu.GetTransfersForInstruction(transferActions, ci.ProgramId, ci.OuterIndex, ci.InnerIndex, nil)

	if len(transfers) < 2 {
		return event
	}

	dexInfo := types.DexInfo{
		ProgramId: ci.ProgramId,
		AMM:       constants.GetProgramName(ci.ProgramId),
		Route:     "",
	}

	// Process only transfers starting from transferStartIdx
	if transferStartIdx < len(transfers) {
		transfers = transfers[transferStartIdx:]
	} else {
		return event
	}

	trade := tu.ProcessSwapData(transfers, dexInfo, skipNative)
	if trade == nil {
		return event
	}

	// Validate trade direction and token mints match
	isBuy := event.Type == types.TradeTypeBuy
	if isBuy && trade.InputToken.Mint != event.QuoteMint {
		return event
	}
	if !isBuy && trade.InputToken.Mint != baseMint {
		return event
	}

	tu.UpdateMemeTokenInfo(event, trade)
	return event
}

// UpdateMemeTokenInfo updates meme event token information from trade data
func (tu *TransactionUtils) UpdateMemeTokenInfo(event *types.MemeEvent, trade *types.TradeInfo) {
	if event.InputToken == nil {
		event.InputToken = &types.TokenInfo{
			Mint:      "",
			Amount:    0,
			AmountRaw: "0",
			Decimals:  0,
		}
	}
	if event.OutputToken == nil {
		event.OutputToken = &types.TokenInfo{
			Mint:      "",
			Amount:    0,
			AmountRaw: "0",
			Decimals:  0,
		}
	}

	// Update input token info
	event.InputToken.Mint = trade.InputToken.Mint
	event.InputToken.Amount = trade.InputToken.Amount
	event.InputToken.AmountRaw = trade.InputToken.AmountRaw
	event.InputToken.Decimals = trade.InputToken.Decimals
	event.InputToken.Authority = trade.InputToken.Authority
	event.InputToken.Source = trade.InputToken.Source
	event.InputToken.Destination = trade.InputToken.Destination

	// Update output token info
	event.OutputToken.Mint = trade.OutputToken.Mint
	event.OutputToken.Amount = trade.OutputToken.Amount
	event.OutputToken.AmountRaw = trade.OutputToken.AmountRaw
	event.OutputToken.Decimals = trade.OutputToken.Decimals
	event.OutputToken.Authority = trade.OutputToken.Authority
	event.OutputToken.Source = trade.OutputToken.Source
	event.OutputToken.Destination = trade.OutputToken.Destination

	// Update fee info if available
	if trade.Fee != nil {
		feeAmount := trade.Fee.Amount
		event.ProtocolFee = &feeAmount
	}
}
