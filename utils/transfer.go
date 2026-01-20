package utils

import (
	"encoding/binary"
	"math/big"
	"strconv"
	"strings"

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
	return (programId == constants.TOKEN_PROGRAM_ID || programId == constants.TOKEN_2022_PROGRAM_ID) &&
		strings.Contains(ix.Parsed.Type, "transferChecked")
}

// IsTransfer checks if instruction is a transfer instruction
func IsTransfer(ix *adapter.UnifiedInstruction) bool {
	if ix.Parsed == nil {
		return false
	}
	return ix.Program == "spl-token" &&
		ix.ProgramId == constants.TOKEN_PROGRAM_ID &&
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
		ix.ProgramId == constants.TOKEN_PROGRAM_ID &&
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

	// Get mint from token map
	var mint string
	if tokenInfo, ok := adapt.SPLTokenMap[destination]; ok {
		mint = tokenInfo.Mint
	}
	if mint == "" {
		if tokenInfo, ok := adapt.SPLTokenMap[source]; ok {
			mint = tokenInfo.Mint
		}
	}
	if mint == "" && ix.ProgramId == constants.TOKENS.NATIVE {
		mint = constants.TOKENS.SOL
	}
	if mint == "" {
		return nil
	}

	decimals := adapt.GetTokenDecimals(mint)
	if decimals == 0 {
		if d, ok := constants.TOKEN_DECIMALS[mint]; ok {
			decimals = d
		}
	}

	sourceBalances := adapt.GetTokenAccountBalance([]string{source})
	destinationBalances := adapt.GetTokenAccountBalance([]string{destination})
	sourcePreBalances := adapt.GetTokenAccountPreBalance([]string{source})
	destinationPreBalances := adapt.GetTokenAccountPreBalance([]string{destination})

	amountBig, _ := new(big.Int).SetString(amount, 10)
	uiAmount := types.ConvertToUIAmount(amountBig, decimals)

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

	amountBig, _ := new(big.Int).SetString(lamports, 10)
	uiAmount := types.ConvertToUIAmount(amountBig, decimals)

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
		if ui, ok := ta["uiAmount"].(float64); ok {
			tokenAmount.UIAmount = &ui
		}
		if d, ok := ta["decimals"].(float64); ok {
			tokenAmount.Decimals = uint8(d)
		}
	} else {
		amount := getStringFromMap(info, "amount")
		amountBig, _ := new(big.Int).SetString(amount, 10)
		uiAmount := types.ConvertToUIAmount(amountBig, decimals)
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
	amountBig, _ := new(big.Int).SetString(amount, 10)
	uiAmount := types.ConvertToUIAmount(amountBig, decimals)

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
func IsCompiledTransferCheck(ix *adapter.UnifiedInstruction) bool {
	if len(ix.Data) == 0 {
		return false
	}
	programId := ix.ProgramId
	return (programId == constants.TOKEN_PROGRAM_ID || programId == constants.TOKEN_2022_PROGRAM_ID) &&
		ix.Data[0] == constants.SPLTokenTransferChecked
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

	// Get mint from token map
	var mint string
	if tokenInfo, ok := adapt.SPLTokenMap[destination]; ok {
		mint = tokenInfo.Mint
	}
	if mint == "" {
		if tokenInfo, ok := adapt.SPLTokenMap[source]; ok {
			mint = tokenInfo.Mint
		}
	}
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

// ProcessCompiledTransferCheck processes a compiled transferChecked instruction
func ProcessCompiledTransferCheck(ix *adapter.UnifiedInstruction, idx string, adapt *adapter.TransactionAdapter) *types.TransferData {
	if len(ix.Data) < 10 || len(ix.Accounts) < 4 {
		return nil
	}

	source := ix.Accounts[0]
	mint := ix.Accounts[1]
	destination := ix.Accounts[2]
	authority := ix.Accounts[3]

	amount := binary.LittleEndian.Uint64(ix.Data[1:9])
	decimals := ix.Data[9]

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

// Helper to get string from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
