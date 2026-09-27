package utils

import (
	"math/big"
	"sort"
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

// GetDexInfo extracts DEX information from transaction.
// It returns the first program (in first-appearance order, see
// InstructionClassifier.GetAllProgramIds) that is a known DEX program, is not a
// vault and is not a wallet/authority entry. Its name becomes the Route unless
// the program is tagged "amm", in which case it becomes the AMM. Without a
// known program, ProgramId is the first program of the transaction.
func (tu *TransactionUtils) GetDexInfo(classifier *classifier.InstructionClassifier) types.DexInfo {
	programIds := classifier.GetAllProgramIds()
	if len(programIds) == 0 {
		return types.DexInfo{}
	}

	for _, programId := range programIds {
		if isAuthorityEntry(programId) {
			continue
		}
		prog := constants.GetDexProgramByID(programId)
		if prog.Name == "" || hasTag(prog, "vault") {
			continue
		}
		if hasTag(prog, "amm") {
			return types.DexInfo{ProgramId: prog.ID, AMM: prog.Name}
		}
		return types.DexInfo{ProgramId: prog.ID, Route: prog.Name}
	}

	return types.DexInfo{ProgramId: programIds[0]}
}

// hasTag reports whether a DEX program carries the given tag
func hasTag(prog constants.DexProgram, tag string) bool {
	for _, t := range prog.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// isAuthorityEntry reports whether id is listed in DEX_PROGRAMS as a wallet or
// authority (Jupiter DCA keepers, OKX router authority) rather than a program
func isAuthorityEntry(id string) bool {
	switch id {
	case constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER1.ID,
		constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER2.ID,
		constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER3.ID,
		constants.DEX_PROGRAMS.OKX_ROUTER.ID:
		return true
	}
	return false
}

// vaultPrograms lists the DEX programs tagged "vault"
var vaultPrograms = func() map[string]bool {
	m := make(map[string]bool)
	for _, id := range constants.DEX_PROGRAM_IDS {
		if prog := constants.GetDexProgramByID(id); hasTag(prog, "vault") {
			m[id] = true
		}
	}
	return m
}()

// GetTransferActions extracts transfer actions from transaction
func (tu *TransactionUtils) GetTransferActions(extraTypes []string) map[string][]types.TransferData {
	actions := make(map[string][]types.TransferData)
	innerInstructions := tu.adapter.InnerInstructions()

	groupKey := ""

	// Process transfers of program instructions
	for _, set := range innerInstructions {
		outerIndex := set.Index
		outerInstruction := tu.adapter.InstructionAt(outerIndex)
		if outerInstruction == nil {
			continue
		}
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
	for _, actionType := range extraTypes {
		if IsExtraAction(ix, actionType) {
			return ProcessExtraAction(ix, idx, tu.adapter, actionType)
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
	for _, actionType := range extraTypes {
		if IsCompiledExtraAction(ix, actionType) {
			return ProcessCompiledExtraAction(ix, idx, tu.adapter, actionType)
		}
	}
	return nil
}

// isIgnoredProgram checks if program should be ignored for grouping: skipped
// programs and every program tagged "vault" (their CPIs belong to the caller)
func (tu *TransactionUtils) isIgnoredProgram(programId string) bool {
	for _, p := range constants.SKIP_PROGRAM_IDS {
		if p == programId {
			return true
		}
	}
	return vaultPrograms[programId]
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
	inputToken, outputToken, feeTransfer := tu.calculateTokenAmounts(signer, transfers, uniqueTokens, skipNative)

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
		feeUIAmount := float64(0)
		if feeTransfer.Info.TokenAmount.UIAmount != nil {
			feeUIAmount = *feeTransfer.Info.TokenAmount.UIAmount
		}
		trade.Fee = &types.FeeInfo{
			Mint:      feeTransfer.Info.Mint,
			Amount:    feeUIAmount,
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
		// Native SOL transfers (System program: fees, tips, account funding)
		// are not swap legs; SOL moved by the pool travels as WSOL.
		if skipNative && transfer.ProgramId == constants.SYSTEM_PROGRAM_ID {
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
func (tu *TransactionUtils) calculateTokenAmounts(signer string, transfers []types.TransferData, uniqueTokens []types.TokenInfo, skipNative bool) (types.TokenInfo, types.TokenInfo, *types.TransferData) {
	inputToken := uniqueTokens[0]
	outputToken := uniqueTokens[len(uniqueTokens)-1]

	// Check if tokens should be swapped
	if (outputToken.Source == signer || outputToken.Authority == signer) ||
		(outputToken.Source == constants.DEX_PROGRAMS.OKX_ROUTER.ID || outputToken.Authority == constants.DEX_PROGRAMS.OKX_ROUTER.ID) {
		inputToken, outputToken = outputToken, inputToken
	}

	inputAmountRaw, outputAmountRaw, feeTransfer := tu.sumTokenAmounts(transfers, inputToken.Mint, outputToken.Mint, signer, skipNative)

	inputToken.AmountRaw = inputAmountRaw.String()
	inputToken.Amount = types.ConvertToUIAmount(inputAmountRaw, inputToken.Decimals)
	outputToken.AmountRaw = outputAmountRaw.String()
	outputToken.Amount = types.ConvertToUIAmount(outputAmountRaw, outputToken.Decimals)

	return inputToken, outputToken, feeTransfer
}

// sumTokenAmounts sums the raw amounts of the transfers of the input and output
// mints. A transfer that forwards the same mint and amount from (or to) an
// account an already counted transfer ended at (or started from) is a
// pass-through leg and is counted once; independent transfers of equal size
// (split legs, equal fee splits) are all counted. With skipNative, System
// program transfers (fees, tips, rent) are ignored.
func (tu *TransactionUtils) sumTokenAmounts(transfers []types.TransferData, inputMint, outputMint, signer string, skipNative bool) (*big.Int, *big.Int, *types.TransferData) {
	var counted []*types.TransferData
	inputAmountRaw := big.NewInt(0)
	outputAmountRaw := big.NewInt(0)
	var feeTransfer *types.TransferData

	isPassThrough := func(t *types.TransferData) bool {
		for _, c := range counted {
			if c.Info.Mint == t.Info.Mint && c.Info.TokenAmount.Amount == t.Info.TokenAmount.Amount &&
				((t.Info.Source != "" && t.Info.Source == c.Info.Destination) ||
					(t.Info.Destination != "" && t.Info.Destination == c.Info.Source)) {
				return true
			}
		}
		return false
	}

	for i := range transfers {
		transfer := &transfers[i]
		if skipNative && transfer.ProgramId == constants.SYSTEM_PROGRAM_ID {
			continue
		}

		destination := transfer.Info.DestinationOwner
		if destination == "" {
			destination = transfer.Info.Destination
		}
		if constants.IsFeeAccount(destination) {
			feeTransfer = transfer
			continue
		}
		if transfer.Info.Authority == constants.DEX_PROGRAMS.OKX_ROUTER.ID && destination == signer {
			continue
		}

		if isPassThrough(transfer) {
			continue
		}
		counted = append(counted, transfer)

		amount := parseAmount(transfer.Info.TokenAmount.Amount)
		if transfer.Info.Mint == inputMint {
			inputAmountRaw.Add(inputAmountRaw, amount)
		}
		if transfer.Info.Mint == outputMint {
			outputAmountRaw.Add(outputAmountRaw, amount)
		}
	}

	return inputAmountRaw, outputAmountRaw, feeTransfer
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

// AttachTokenTransferInfo attaches token transfer info to trade. The input and
// output transfers are the first transfers, in execution order, whose mint and
// amount match the trade.
func (tu *TransactionUtils) AttachTokenTransferInfo(trade *types.TradeInfo, transferActions map[string][]types.TransferData) *types.TradeInfo {
	if trade == nil {
		return nil
	}

	// Find input and output transfers
	var inputTransfer, outputTransfer *types.TransferData
	for _, key := range SortedTransferKeys(transferActions) {
		transfers := transferActions[key]
		for i := range transfers {
			t := &transfers[i]
			if inputTransfer == nil && t.Info.Mint == trade.InputToken.Mint && t.Info.TokenAmount.Amount == trade.InputToken.AmountRaw {
				inputTransfer = t
			}
			if outputTransfer == nil && t.Info.Mint == trade.OutputToken.Mint && t.Info.TokenAmount.Amount == trade.OutputToken.AmountRaw {
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

// AttachTradeFee completes a trade (typically the aggregate trade) after
// parsing. It does not infer fees: Fee stays what the parsers reported from
// protocol fee events or fee transfers. For SOL input it sets
// InputToken.BalanceChange to the user's SOL change (SOL and token account
// lamports of the user, summed) when that exceeds the input amount, fills
// Signer when it is empty, and attributes a trading bot (DetectBot). Like
// AttachTokenTransferInfo, the input BalanceChange is the absolute amount spent.
func (tu *TransactionUtils) AttachTradeFee(trade *types.TradeInfo) *types.TradeInfo {
	if trade == nil {
		return nil
	}

	if trade.InputToken.Mint == constants.TOKENS.SOL {
		token := tu.adapter.GetAccountSolBalanceChanges(true)[trade.User]
		if token != nil && token.Change.Amount != "" && trade.InputToken.AmountRaw != "" {
			change, ok1 := new(big.Int).SetString(token.Change.Amount, 10)
			input, ok2 := new(big.Int).SetString(trade.InputToken.AmountRaw, 10)
			if ok1 && ok2 && change.Sign() < 0 && new(big.Int).Abs(change).Cmp(input) > 0 {
				trade.InputToken.BalanceChange = new(big.Int).Abs(change).String()
			}
		}
	}

	if len(trade.Signer) == 0 {
		trade.Signer = tu.adapter.Signers()
	}

	// Detect trading bot from fee transfers
	tu.DetectBot(trade)

	return trade
}

// BotFeeMinLamports is the smallest SOL/WSOL amount a bot fee account must
// receive in a transaction for DetectBot to attribute the trade to that bot.
// Smaller credits are dust (address poisoning) and do not count.
const BotFeeMinLamports = 10000

// BotFeeMinLegDivisor sets the smallest credit in a traded mint that counts as
// a bot fee: 1/BotFeeMinLegDivisor (0.1%) of the trade leg in that mint.
const BotFeeMinLegDivisor = 1000

// DetectBot attributes a trading bot when one of its fee accounts
// (constants.BOT_FEE_ACCOUNTS), or a token account owned by one, receives in
// the transaction either at least BotFeeMinLamports of SOL or WSOL, or an
// amount of the trade's input or output mint that is at least 0.1% of that
// trade leg (some bots, e.g. BONKbot, take their fee in the traded token).
// Credits are balance deltas, so fees moved by a program without a transfer
// instruction count too; the owner of a token account comes from the token
// balances, so the fee wallet itself need not be in the account keys. Presence
// of a fee account alone never counts, and the trade's user and the signers are
// never fee receivers. For SOL/WSOL only the absolute threshold applies, so
// dust never counts on small trades. Accounts are checked in account-key order;
// the first match wins.
func (tu *TransactionUtils) DetectBot(trade *types.TradeInfo) {
	if trade == nil || trade.Bot != "" {
		return
	}

	solChanges := tu.adapter.GetAccountSolBalanceChanges(false)
	byAccount := tu.adapter.GetAccountTokenBalanceChanges(false)
	byOwner := tu.adapter.GetAccountTokenBalanceChanges(true)

	// Minimum credit per mint: BotFeeMinLamports for SOL/WSOL, 0.1% (rounded
	// up, at least 1) of the smaller trade leg for the traded mints
	minFee := map[string]*big.Int{constants.TOKENS.SOL: big.NewInt(BotFeeMinLamports)}
	for _, leg := range []types.TokenInfo{trade.InputToken, trade.OutputToken} {
		if leg.Mint == "" || leg.Mint == constants.TOKENS.SOL {
			continue
		}
		amount, ok := new(big.Int).SetString(leg.AmountRaw, 10)
		if !ok || amount.Sign() <= 0 {
			continue
		}
		min := new(big.Int).Add(amount, big.NewInt(BotFeeMinLegDivisor-1))
		min.Quo(min, big.NewInt(BotFeeMinLegDivisor))
		if cur, ok := minFee[leg.Mint]; !ok || min.Cmp(cur) < 0 {
			minFee[leg.Mint] = min
		}
	}
	received := func(change *types.BalanceChange, min *big.Int) bool {
		if change == nil {
			return false
		}
		v, ok := new(big.Int).SetString(change.Change.Amount, 10)
		return ok && v.Sign() > 0 && v.Cmp(min) >= 0
	}
	receivedToken := func(byMint map[string]*types.BalanceChange) bool {
		for mint, min := range minFee {
			if received(byMint[mint], min) {
				return true
			}
		}
		return false
	}

	// The trader's own accounts are never fee receivers
	excluded := map[string]bool{trade.User: true}
	for _, signer := range tu.adapter.Signers() {
		excluded[signer] = true
	}

	checkedOwners := make(map[string]bool)
	for _, key := range tu.adapter.AccountKeys {
		if key == "" || excluded[key] {
			continue
		}
		// A listed wallet, or a listed token account (e.g. a WSOL account)
		if bot := constants.GetBotName(key); bot != "" {
			checkedOwners[key] = true
			if received(solChanges[key], minFee[constants.TOKENS.SOL]) || receivedToken(byAccount[key]) || receivedToken(byOwner[key]) {
				trade.Bot = bot
				return
			}
		}
		// A token account owned by a listed wallet
		owner := tu.adapter.GetTokenAccountOwner(key)
		if owner == "" || excluded[owner] || checkedOwners[owner] {
			continue
		}
		checkedOwners[owner] = true
		if bot := constants.GetBotName(owner); bot != "" && receivedToken(byOwner[owner]) {
			trade.Bot = bot
			return
		}
	}
}

// ApplyToken2022TransferFee makes the output of a trade what the user
// actually received when it was delivered by a Token-2022 transfer that
// withheld a transfer fee (TransferFee extension): the destination account is
// credited less than the transferred amount. transfers are all transfers of the
// transaction (SortedTransfers). The withheld amount is derived from the
// destination's balances, incoming and outgoing transfers, applied only when
// the delivering transfer is the account's only credit, and reported in Fees
// with Type "transferFee".
func (tu *TransactionUtils) ApplyToken2022TransferFee(trade *types.TradeInfo, transfers []types.TransferData) {
	if trade == nil || trade.OutputToken.Mint == "" || trade.OutputToken.Mint == constants.TOKENS.SOL {
		return
	}
	mint := trade.OutputToken.Mint

	var delivered *types.TransferData
	for i := range transfers {
		t := &transfers[i]
		if t.ProgramId == constants.TOKEN_2022_PROGRAM_ID && t.Info.Mint == mint &&
			t.Info.TokenAmount.Amount == trade.OutputToken.AmountRaw &&
			tu.adapter.GetTokenAccountOwner(t.Info.Destination) == trade.User {
			delivered = t
			break
		}
	}
	if delivered == nil {
		return
	}
	dest := delivered.Info.Destination
	post := tu.adapter.GetTokenAccountBalance([]string{dest})[0]
	if post == nil {
		return // closed or unknown: cannot account for it
	}
	preAmount := new(big.Int)
	if pre := tu.adapter.GetTokenAccountPreBalance([]string{dest})[0]; pre != nil {
		preAmount = parseAmount(pre.Amount)
	}

	incoming, outgoing := new(big.Int), new(big.Int)
	for _, t := range transfers {
		if t.Info.Mint != mint {
			continue
		}
		amount := parseAmount(t.Info.TokenAmount.Amount)
		if t.Info.Destination == dest {
			incoming.Add(incoming, amount)
		}
		if t.Info.Source == dest {
			outgoing.Add(outgoing, amount)
		}
	}
	amount := parseAmount(delivered.Info.TokenAmount.Amount)
	if incoming.Cmp(amount) != 0 {
		return // other credits: the fee cannot be attributed
	}
	// withheld = incoming - outgoing - (post - pre)
	withheld := new(big.Int).Sub(incoming, outgoing)
	withheld.Sub(withheld, new(big.Int).Sub(parseAmount(post.Amount), preAmount))
	if withheld.Sign() <= 0 || withheld.Cmp(amount) >= 0 {
		return
	}

	received := new(big.Int).Sub(amount, withheld)
	decimals := trade.OutputToken.Decimals
	trade.OutputToken.AmountRaw = received.String()
	trade.OutputToken.Amount = types.ConvertToUIAmount(received, decimals)
	trade.Fees = append(trade.Fees, types.FeeInfo{
		Mint:      mint,
		Amount:    types.ConvertToUIAmount(withheld, decimals),
		AmountRaw: withheld.String(),
		Decimals:  decimals,
		Type:      "transferFee",
	})
}

// SortedTransferKeys returns the keys of transferActions in execution order of
// their first transfer (numeric idx), so that iteration is deterministic.
func SortedTransferKeys(transferActions map[string][]types.TransferData) []string {
	keys := make([]string, 0, len(transferActions))
	for key := range transferActions {
		keys = append(keys, key)
	}
	firstIdx := func(key string) string {
		if transfers := transferActions[key]; len(transfers) > 0 {
			return transfers[0].Idx
		}
		return ""
	}
	sort.Slice(keys, func(i, j int) bool {
		if c := compareIdx(firstIdx(keys[i]), firstIdx(keys[j])); c != 0 {
			return c < 0
		}
		return keys[i] < keys[j]
	})
	return keys
}

// SortedTransfers returns all transfers of transferActions in execution order
// (numeric idx).
func SortedTransfers(transferActions map[string][]types.TransferData) []types.TransferData {
	var all []types.TransferData
	for _, key := range SortedTransferKeys(transferActions) {
		all = append(all, transferActions[key]...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return compareIdx(all[i].Idx, all[j].Idx) < 0
	})
	return all
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
