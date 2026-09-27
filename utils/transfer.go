package utils

import (
	"encoding/binary"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/goccy/go-json"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// IsTransferCheck checks if instruction is a transferChecked instruction
func IsTransferCheck(ix *adapter.UnifiedInstruction) bool {
	if ix.Parsed == nil {
		return false
	}
	programId := ix.ProgramId
	// also matches transferCheckedWithFee (Token-2022 transfer fee extension)
	return isTokenProgram(programId) &&
		strings.Contains(ix.Parsed.Type, "transferChecked")
}

// isTokenProgram reports whether programId is the SPL Token or Token-2022
// program (jsonParsed labels both as program "spl-token")
func isTokenProgram(programId string) bool {
	return programId == constants.TOKEN_PROGRAM_ID || programId == constants.TOKEN_2022_PROGRAM_ID
}

// IsTransfer checks if instruction is a transfer instruction
func IsTransfer(ix *adapter.UnifiedInstruction) bool {
	if ix.Parsed == nil {
		return false
	}
	return ix.Program == "spl-token" &&
		isTokenProgram(ix.ProgramId) &&
		ix.Parsed.Type == "transfer"
}

// IsNativeTransfer checks if instruction is a native SOL transfer
func IsNativeTransfer(ix *adapter.UnifiedInstruction) bool {
	if ix.Parsed == nil {
		return false
	}
	return ix.Program == "system" &&
		ix.ProgramId == constants.TOKENS.NATIVE &&
		ix.Parsed.Type == "transfer"
}

// IsExtraAction checks if instruction is an extra action type
func IsExtraAction(ix *adapter.UnifiedInstruction, actionType string) bool {
	if ix.Parsed == nil {
		return false
	}
	return ix.Program == "spl-token" &&
		isTokenProgram(ix.ProgramId) &&
		ix.Parsed.Type == actionType
}

// ProcessTransfer processes a parsed transfer instruction
func ProcessTransfer(ix *adapter.UnifiedInstruction, idx string, adapt *adapter.TransactionAdapter) *types.TransferData {
	if ix.Parsed == nil || ix.Parsed.Info == nil {
		return nil
	}

	info := ix.Parsed.Info
	source := getStringFromMap(info, "source")
	destination := getStringFromMap(info, "destination")
	authority := getStringFromMap(info, "authority")
	amount := getStringFromMap(info, "amount")

	// Get mint from token map, preferring a non-SOL mint of either side
	mint := transferTokenMint(adapt, source, destination)
	if mint == "" && ix.ProgramId == constants.TOKENS.NATIVE {
		mint = constants.TOKENS.SOL
	}
	if mint == "" {
		return nil
	}

	decimals := adapt.GetTokenDecimals(mint)

	sourceBalances := adapt.GetTokenAccountBalance([]string{source})
	destinationBalances := adapt.GetTokenAccountBalance([]string{destination})
	sourcePreBalances := adapt.GetTokenAccountPreBalance([]string{source})
	destinationPreBalances := adapt.GetTokenAccountPreBalance([]string{destination})

	uiAmount := types.ConvertToUIAmount(parseAmount(amount), decimals)

	return &types.TransferData{
		Type:      "transfer",
		ProgramId: ix.ProgramId,
		Info: types.TransferDataInfo{
			Authority:             authority,
			Destination:           destination,
			DestinationOwner:      adapt.GetTokenAccountOwner(destination),
			Mint:                  mint,
			Source:                source,
			TokenAmount:           types.TokenAmount{Amount: amount, UIAmount: &uiAmount, Decimals: decimals},
			SourceBalance:         sourceBalances[0],
			SourcePreBalance:      sourcePreBalances[0],
			DestinationBalance:    destinationBalances[0],
			DestinationPreBalance: destinationPreBalances[0],
		},
		Idx:       idx,
		Timestamp: adapt.BlockTime(),
		Signature: adapt.Signature(),
	}
}

// ProcessNativeTransfer processes a native SOL transfer instruction
func ProcessNativeTransfer(ix *adapter.UnifiedInstruction, idx string, adapt *adapter.TransactionAdapter) *types.TransferData {
	if ix.Parsed == nil || ix.Parsed.Info == nil {
		return nil
	}

	info := ix.Parsed.Info
	source := getStringFromMap(info, "source")
	destination := getStringFromMap(info, "destination")
	lamports := getStringFromMap(info, "lamports")

	mint := constants.TOKENS.SOL
	var decimals uint8 = 9

	sourceBalances := adapt.GetAccountBalance([]string{source})
	destinationBalances := adapt.GetAccountBalance([]string{destination})
	sourcePreBalances := adapt.GetAccountPreBalance([]string{source})
	destinationPreBalances := adapt.GetAccountPreBalance([]string{destination})

	if lamports == "" {
		return nil
	}
	uiAmount := types.ConvertToUIAmount(parseAmount(lamports), decimals)

	return &types.TransferData{
		Type:      "transfer",
		ProgramId: ix.ProgramId,
		Info: types.TransferDataInfo{
			Destination:           destination,
			DestinationOwner:      adapt.GetTokenAccountOwner(destination),
			Mint:                  mint,
			Source:                source,
			TokenAmount:           types.TokenAmount{Amount: lamports, UIAmount: &uiAmount, Decimals: decimals},
			SourceBalance:         sourceBalances[0],
			SourcePreBalance:      sourcePreBalances[0],
			DestinationBalance:    destinationBalances[0],
			DestinationPreBalance: destinationPreBalances[0],
		},
		Idx:       idx,
		Timestamp: adapt.BlockTime(),
		Signature: adapt.Signature(),
	}
}

// ProcessTransferCheck processes a transferChecked instruction
func ProcessTransferCheck(ix *adapter.UnifiedInstruction, idx string, adapt *adapter.TransactionAdapter) *types.TransferData {
	if ix.Parsed == nil || ix.Parsed.Info == nil {
		return nil
	}

	info := ix.Parsed.Info
	source := getStringFromMap(info, "source")
	destination := getStringFromMap(info, "destination")
	authority := getStringFromMap(info, "authority")
	mint := getStringFromMap(info, "mint")

	decimals := adapt.GetTokenDecimals(mint)

	sourceBalances := adapt.GetTokenAccountBalance([]string{source})
	destinationBalances := adapt.GetTokenAccountBalance([]string{destination})
	sourcePreBalances := adapt.GetTokenAccountPreBalance([]string{source})
	destinationPreBalances := adapt.GetTokenAccountPreBalance([]string{destination})

	// Get tokenAmount from parsed info
	var tokenAmount types.TokenAmount
	if ta, ok := info["tokenAmount"].(map[string]interface{}); ok {
		tokenAmount.Amount = getStringFromMap(ta, "amount")
		tokenAmount.Decimals = decimals
		if d, ok := getUint8FromMap(ta, "decimals"); ok {
			tokenAmount.Decimals = d
		}
		ui, ok := getFloatFromMap(ta, "uiAmount")
		if !ok {
			ui = types.ConvertToUIAmount(parseAmount(tokenAmount.Amount), tokenAmount.Decimals)
		}
		tokenAmount.UIAmount = &ui
	} else {
		amount := getStringFromMap(info, "amount")
		uiAmount := types.ConvertToUIAmount(parseAmount(amount), decimals)
		tokenAmount = types.TokenAmount{Amount: amount, UIAmount: &uiAmount, Decimals: decimals}
	}

	return &types.TransferData{
		Type:      "transferChecked",
		ProgramId: ix.ProgramId,
		Info: types.TransferDataInfo{
			Authority:             authority,
			Destination:           destination,
			DestinationOwner:      adapt.GetTokenAccountOwner(destination),
			Mint:                  mint,
			Source:                source,
			TokenAmount:           tokenAmount,
			SourceBalance:         sourceBalances[0],
			SourcePreBalance:      sourcePreBalances[0],
			DestinationBalance:    destinationBalances[0],
			DestinationPreBalance: destinationPreBalances[0],
		},
		Idx:       idx,
		Timestamp: adapt.BlockTime(),
		Signature: adapt.Signature(),
	}
}

// ProcessExtraAction processes extra actions like mintTo, burn, etc.
func ProcessExtraAction(ix *adapter.UnifiedInstruction, idx string, adapt *adapter.TransactionAdapter, actionType string) *types.TransferData {
	if ix.Parsed == nil || ix.Parsed.Info == nil {
		return nil
	}

	info := ix.Parsed.Info
	source := getStringFromMap(info, "source")
	destination := getStringFromMap(info, "destination")
	// jsonParsed names the token account "account": the destination of
	// mintTo and the source of burn
	switch actionType {
	case "mintTo", "mintToChecked":
		if destination == "" {
			destination = getStringFromMap(info, "account")
		}
	case "burn", "burnChecked":
		if source == "" {
			source = getStringFromMap(info, "account")
		}
	}
	authority := getStringFromMap(info, "authority")
	if authority == "" {
		authority = getStringFromMap(info, "mintAuthority")
	}
	mint := getStringFromMap(info, "mint")

	if mint == "" {
		if tokenInfo, ok := adapt.SPLTokenMap[destination]; ok {
			mint = tokenInfo.Mint
		}
	}
	if mint == "" {
		return nil
	}

	decimals := adapt.GetTokenDecimals(mint)

	amount := getStringFromMap(info, "amount")
	if amount == "" {
		if ta, ok := info["tokenAmount"].(map[string]interface{}); ok {
			amount = getStringFromMap(ta, "amount")
		}
	}
	uiAmount := types.ConvertToUIAmount(parseAmount(amount), decimals)

	sourceBalances := adapt.GetTokenAccountBalance([]string{source})
	destinationBalances := adapt.GetTokenAccountBalance([]string{destination})
	sourcePreBalances := adapt.GetTokenAccountPreBalance([]string{source})
	destinationPreBalances := adapt.GetTokenAccountPreBalance([]string{destination})

	return &types.TransferData{
		Type:      actionType,
		ProgramId: ix.ProgramId,
		Info: types.TransferDataInfo{
			Authority:             authority,
			Destination:           destination,
			DestinationOwner:      adapt.GetTokenAccountOwner(destination),
			Mint:                  mint,
			Source:                source,
			TokenAmount:           types.TokenAmount{Amount: amount, UIAmount: &uiAmount, Decimals: decimals},
			SourceBalance:         sourceBalances[0],
			SourcePreBalance:      sourcePreBalances[0],
			DestinationBalance:    destinationBalances[0],
			DestinationPreBalance: destinationPreBalances[0],
		},
		Idx:       idx,
		Timestamp: adapt.BlockTime(),
		Signature: adapt.Signature(),
	}
}

// IsCompiledTransfer checks if a compiled instruction is a transfer
func IsCompiledTransfer(ix *adapter.UnifiedInstruction) bool {
	if len(ix.Data) == 0 {
		return false
	}
	programId := ix.ProgramId
	return (programId == constants.TOKEN_PROGRAM_ID || programId == constants.TOKEN_2022_PROGRAM_ID) &&
		ix.Data[0] == constants.SPLTokenTransfer
}

// IsCompiledTransferCheck checks if a compiled instruction is a transferChecked
// (or a Token-2022 TransferCheckedWithFee)
func IsCompiledTransferCheck(ix *adapter.UnifiedInstruction) bool {
	if len(ix.Data) == 0 {
		return false
	}
	return (isTokenProgram(ix.ProgramId) && ix.Data[0] == constants.SPLTokenTransferChecked) ||
		isCompiledTransferCheckedWithFee(ix)
}

// IsCompiledNativeTransfer checks if a compiled instruction is a native SOL transfer
func IsCompiledNativeTransfer(ix *adapter.UnifiedInstruction) bool {
	if len(ix.Data) == 0 {
		return false
	}
	return ix.ProgramId == constants.TOKENS.NATIVE && ix.Data[0] == constants.SystemTransfer
}

// IsCompiledExtraAction checks if a compiled instruction is an extra action
func IsCompiledExtraAction(ix *adapter.UnifiedInstruction, actionType string) bool {
	if len(ix.Data) == 0 {
		return false
	}
	programId := ix.ProgramId
	if programId != constants.TOKEN_PROGRAM_ID && programId != constants.TOKEN_2022_PROGRAM_ID {
		return false
	}

	switch actionType {
	case "mintTo":
		return ix.Data[0] == constants.SPLTokenMintTo
	case "mintToChecked":
		return ix.Data[0] == constants.SPLTokenMintToChecked
	case "burn":
		return ix.Data[0] == constants.SPLTokenBurn
	case "burnChecked":
		return ix.Data[0] == constants.SPLTokenBurnChecked
	default:
		return false
	}
}

// ProcessCompiledTransfer processes a compiled transfer instruction
func ProcessCompiledTransfer(ix *adapter.UnifiedInstruction, idx string, adapt *adapter.TransactionAdapter) *types.TransferData {
	if len(ix.Data) < 9 || len(ix.Accounts) < 3 {
		return nil
	}

	source := ix.Accounts[0]
	destination := ix.Accounts[1]
	authority := ix.Accounts[2]

	amount := binary.LittleEndian.Uint64(ix.Data[1:9])

	// Get mint from token map, preferring a non-SOL mint of either side
	mint := transferTokenMint(adapt, source, destination)
	if mint == "" {
		return nil
	}

	decimals := adapt.GetTokenDecimals(mint)

	sourceBalances := adapt.GetTokenAccountBalance([]string{source})
	destinationBalances := adapt.GetTokenAccountBalance([]string{destination})
	sourcePreBalances := adapt.GetTokenAccountPreBalance([]string{source})
	destinationPreBalances := adapt.GetTokenAccountPreBalance([]string{destination})

	amountStr := strconv.FormatUint(amount, 10)
	uiAmount := types.ConvertToUIAmount(new(big.Int).SetUint64(amount), decimals)

	return &types.TransferData{
		Type:      "transfer",
		ProgramId: ix.ProgramId,
		Info: types.TransferDataInfo{
			Authority:             authority,
			Destination:           destination,
			DestinationOwner:      adapt.GetTokenAccountOwner(destination),
			Mint:                  mint,
			Source:                source,
			TokenAmount:           types.TokenAmount{Amount: amountStr, UIAmount: &uiAmount, Decimals: decimals},
			SourceBalance:         sourceBalances[0],
			SourcePreBalance:      sourcePreBalances[0],
			DestinationBalance:    destinationBalances[0],
			DestinationPreBalance: destinationPreBalances[0],
		},
		Idx:       idx,
		Timestamp: adapt.BlockTime(),
		Signature: adapt.Signature(),
	}
}

// ProcessCompiledNativeTransfer processes a compiled native SOL transfer
func ProcessCompiledNativeTransfer(ix *adapter.UnifiedInstruction, idx string, adapt *adapter.TransactionAdapter) *types.TransferData {
	if len(ix.Data) < 12 || len(ix.Accounts) < 2 {
		return nil
	}

	source := ix.Accounts[0]
	destination := ix.Accounts[1]

	lamports := binary.LittleEndian.Uint64(ix.Data[4:12])

	mint := constants.TOKENS.SOL
	var decimals uint8 = 9

	sourceBalances := adapt.GetAccountBalance([]string{source})
	destinationBalances := adapt.GetAccountBalance([]string{destination})
	sourcePreBalances := adapt.GetAccountPreBalance([]string{source})
	destinationPreBalances := adapt.GetAccountPreBalance([]string{destination})

	amountStr := strconv.FormatUint(lamports, 10)
	uiAmount := types.ConvertToUIAmount(new(big.Int).SetUint64(lamports), decimals)

	return &types.TransferData{
		Type:      "transfer",
		ProgramId: ix.ProgramId,
		Info: types.TransferDataInfo{
			Destination:           destination,
			DestinationOwner:      adapt.GetTokenAccountOwner(destination),
			Mint:                  mint,
			Source:                source,
			TokenAmount:           types.TokenAmount{Amount: amountStr, UIAmount: &uiAmount, Decimals: decimals},
			SourceBalance:         sourceBalances[0],
			SourcePreBalance:      sourcePreBalances[0],
			DestinationBalance:    destinationBalances[0],
			DestinationPreBalance: destinationPreBalances[0],
		},
		Idx:       idx,
		Timestamp: adapt.BlockTime(),
		Signature: adapt.Signature(),
	}
}

// ProcessCompiledTransferCheck processes a compiled transferChecked
// instruction, or a Token-2022 TransferCheckedWithFee (26/1) whose amount is
// the gross amount debited from the source
func ProcessCompiledTransferCheck(ix *adapter.UnifiedInstruction, idx string, adapt *adapter.TransactionAdapter) *types.TransferData {
	if len(ix.Accounts) < 4 {
		return nil
	}

	var amount uint64
	var decimals uint8
	switch {
	case len(ix.Data) >= 10 && ix.Data[0] == constants.SPLTokenTransferChecked:
		amount = binary.LittleEndian.Uint64(ix.Data[1:9])
		decimals = ix.Data[9]
	case isCompiledTransferCheckedWithFee(ix):
		amount = binary.LittleEndian.Uint64(ix.Data[2:10])
		decimals = ix.Data[10]
	default:
		return nil
	}

	source := ix.Accounts[0]
	mint := ix.Accounts[1]
	destination := ix.Accounts[2]
	authority := ix.Accounts[3]

	sourceBalances := adapt.GetTokenAccountBalance([]string{source})
	destinationBalances := adapt.GetTokenAccountBalance([]string{destination})
	sourcePreBalances := adapt.GetTokenAccountPreBalance([]string{source})
	destinationPreBalances := adapt.GetTokenAccountPreBalance([]string{destination})

	amountStr := strconv.FormatUint(amount, 10)
	uiAmount := types.ConvertToUIAmount(new(big.Int).SetUint64(amount), decimals)

	return &types.TransferData{
		Type:      "transferChecked",
		ProgramId: ix.ProgramId,
		Info: types.TransferDataInfo{
			Authority:             authority,
			Destination:           destination,
			DestinationOwner:      adapt.GetTokenAccountOwner(destination),
			Mint:                  mint,
			Source:                source,
			TokenAmount:           types.TokenAmount{Amount: amountStr, UIAmount: &uiAmount, Decimals: decimals},
			SourceBalance:         sourceBalances[0],
			SourcePreBalance:      sourcePreBalances[0],
			DestinationBalance:    destinationBalances[0],
			DestinationPreBalance: destinationPreBalances[0],
		},
		Idx:       idx,
		Timestamp: adapt.BlockTime(),
		Signature: adapt.Signature(),
	}
}

// isCompiledTransferCheckedWithFee checks for the Token-2022 TransferFeeExtension
// (26) sub-instruction TransferCheckedWithFee (1): data 26, 1, amount u64,
// decimals u8, fee u64; accounts source, mint, destination, authority
func isCompiledTransferCheckedWithFee(ix *adapter.UnifiedInstruction) bool {
	return ix.ProgramId == constants.TOKEN_2022_PROGRAM_ID &&
		len(ix.Data) >= 19 && ix.Data[0] == splTokenTransferFeeExtension && ix.Data[1] == splTokenTransferCheckedWithFee
}

// SPL Token-2022 TransferFeeExtension instruction tag and its
// TransferCheckedWithFee sub-instruction
const (
	splTokenTransferFeeExtension   = 26
	splTokenTransferCheckedWithFee = 1
)

// ProcessCompiledExtraAction processes compiled extra actions
func ProcessCompiledExtraAction(ix *adapter.UnifiedInstruction, idx string, adapt *adapter.TransactionAdapter, actionType string) *types.TransferData {
	if len(ix.Data) < 9 || len(ix.Accounts) < 2 {
		return nil
	}

	var source, destination, mint, authority string
	var decimals uint8

	switch actionType {
	case "mintTo":
		if len(ix.Accounts) >= 3 {
			mint = ix.Accounts[0]
			destination = ix.Accounts[1]
			authority = ix.Accounts[2]
		}
	case "mintToChecked":
		if len(ix.Accounts) >= 3 && len(ix.Data) >= 10 {
			mint = ix.Accounts[0]
			destination = ix.Accounts[1]
			authority = ix.Accounts[2]
			decimals = ix.Data[9]
		}
	case "burn":
		if len(ix.Accounts) >= 3 {
			source = ix.Accounts[0]
			mint = ix.Accounts[1]
			authority = ix.Accounts[2]
		}
	case "burnChecked":
		if len(ix.Accounts) >= 3 && len(ix.Data) >= 10 {
			source = ix.Accounts[0]
			mint = ix.Accounts[1]
			authority = ix.Accounts[2]
			decimals = ix.Data[9]
		}
	default:
		return nil
	}

	if decimals == 0 {
		decimals = adapt.GetTokenDecimals(mint)
	}

	amount := binary.LittleEndian.Uint64(ix.Data[1:9])

	sourceBalances := adapt.GetTokenAccountBalance([]string{source})
	destinationBalances := adapt.GetTokenAccountBalance([]string{destination})
	sourcePreBalances := adapt.GetTokenAccountPreBalance([]string{source})
	destinationPreBalances := adapt.GetTokenAccountPreBalance([]string{destination})

	amountStr := strconv.FormatUint(amount, 10)
	uiAmount := types.ConvertToUIAmount(new(big.Int).SetUint64(amount), decimals)

	return &types.TransferData{
		Type:      actionType,
		ProgramId: ix.ProgramId,
		Info: types.TransferDataInfo{
			Authority:             authority,
			Destination:           destination,
			DestinationOwner:      adapt.GetTokenAccountOwner(destination),
			Mint:                  mint,
			Source:                source,
			TokenAmount:           types.TokenAmount{Amount: amountStr, UIAmount: &uiAmount, Decimals: decimals},
			SourceBalance:         sourceBalances[0],
			SourcePreBalance:      sourcePreBalances[0],
			DestinationBalance:    destinationBalances[0],
			DestinationPreBalance: destinationPreBalances[0],
		},
		Idx:       idx,
		Timestamp: adapt.BlockTime(),
		Signature: adapt.Signature(),
	}
}

// Helper to get string from map. JSON numbers (json.Number, float64) and Go
// integers are formatted as exact decimal strings, so numeric fields such as
// jsonParsed system transfer lamports are read as well.
func getStringFromMap(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch n := v.(type) {
	case string:
		return n
	case json.Number:
		return n.String()
	case float64:
		if n >= 0 && n == math.Trunc(n) {
			return strconv.FormatFloat(n, 'f', -1, 64)
		}
	case int:
		return strconv.Itoa(n)
	case int64:
		return strconv.FormatInt(n, 10)
	case uint64:
		return strconv.FormatUint(n, 10)
	}
	return ""
}

// getUint8FromMap reads a small JSON number (e.g. decimals)
func getUint8FromMap(m map[string]interface{}, key string) (uint8, bool) {
	s := getStringFromMap(m, key)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 8)
	return uint8(v), err == nil
}

// getFloatFromMap reads a JSON number as float64 (UI amounts)
func getFloatFromMap(m map[string]interface{}, key string) (float64, bool) {
	switch n := m[key].(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// transferTokenMint returns the mint of a token transfer from the token
// accounts on both sides, preferring a non-SOL mint (a side whose mint is a
// SOL default is typically a temporary account)
func transferTokenMint(adapt *adapter.TransactionAdapter, source, destination string) string {
	var destMint, srcMint string
	if info, ok := adapt.SPLTokenMap[destination]; ok {
		destMint = info.Mint
	}
	if info, ok := adapt.SPLTokenMap[source]; ok {
		srcMint = info.Mint
	}
	return GetTransferTokenMint(destMint, srcMint)
}

// parseAmount parses a raw amount string; invalid input yields 0
func parseAmount(amount string) *big.Int {
	v, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return new(big.Int)
	}
	return v
}
