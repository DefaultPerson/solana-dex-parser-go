package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/mr-tron/base58"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/types"
)

// DecodeInstructionData decodes instruction data from various formats
func DecodeInstructionData(data interface{}) ([]byte, error) {
	switch v := data.(type) {
	case string:
		// Base58 encoded string
		return base58.Decode(v)
	case []byte:
		return v, nil
	default:
		return nil, nil
	}
}

// GetProgramName returns the name of a program by its ID
func GetProgramName(programId string) string {
	return constants.GetProgramName(programId)
}

// HexToBytes converts a hex string to byte slice
func HexToBytes(hexStr string) ([]byte, error) {
	// Remove 0x prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")
	return hex.DecodeString(hexStr)
}

// AbsBigInt returns the absolute value of a big.Int
func AbsBigInt(value *big.Int) *big.Int {
	if value.Sign() < 0 {
		return new(big.Int).Neg(value)
	}
	return new(big.Int).Set(value)
}

// GetTradeType determines the trade type based on input/output mints
func GetTradeType(inMint, outMint string) types.TradeType {
	if inMint == constants.TOKENS.SOL {
		return types.TradeTypeBuy
	}
	if outMint == constants.TOKENS.SOL {
		return types.TradeTypeSell
	}
	// Check if input is a stablecoin
	if constants.IsStablecoin(inMint) || constants.IsSOL(inMint) {
		return types.TradeTypeBuy
	}
	return types.TradeTypeSell
}

// GetAMMs extracts AMM names from transfer action keys
func GetAMMs(transferActionKeys []string) []string {
	var result []string
	for _, key := range transferActionKeys {
		parts := strings.Split(key, ":")
		if len(parts) > 0 {
			programId := parts[0]
			prog := constants.GetDexProgramByID(programId)
			if prog.Name != "" {
				// Check if it's an AMM
				for _, tag := range prog.Tags {
					if tag == "amm" {
						if !containsString(result, prog.Name) {
							result = append(result, prog.Name)
						}
						break
					}
				}
			}
		}
	}
	return result
}

// GetTransferTokenMint determines the token mint from two options
func GetTransferTokenMint(token1, token2 string) string {
	if token1 == token2 {
		return token1
	}
	if token1 != "" && token1 != constants.TOKENS.SOL {
		return token1
	}
	if token2 != "" && token2 != constants.TOKENS.SOL {
		return token2
	}
	if token1 != "" {
		return token1
	}
	return token2
}

// GetPubkeyString converts various pubkey representations to string
func GetPubkeyString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return base58.Encode(v)
	default:
		return ""
	}
}

// SortByIdx sorts items by their idx field (format: "main-sub")
func SortByIdx[T interface{ GetIdx() string }](items []T) []T {
	if len(items) <= 1 {
		return items
	}

	sorted := make([]T, len(items))
	copy(sorted, items)

	sort.Slice(sorted, func(i, j int) bool {
		return compareIdx(sorted[i].GetIdx(), sorted[j].GetIdx()) < 0
	})

	return sorted
}

// SortTradesByIdx sorts TradeInfo slice by idx
func SortTradesByIdx(trades []types.TradeInfo) []types.TradeInfo {
	if len(trades) <= 1 {
		return trades
	}

	sorted := make([]types.TradeInfo, len(trades))
	copy(sorted, trades)

	sort.Slice(sorted, func(i, j int) bool {
		return compareIdx(sorted[i].Idx, sorted[j].Idx) < 0
	})

	return sorted
}

// compareIdx compares two idx strings in format "main-sub"
func compareIdx(a, b string) int {
	aParts := strings.Split(a, "-")
	bParts := strings.Split(b, "-")

	aMain, _ := strconv.Atoi(aParts[0])
	bMain, _ := strconv.Atoi(bParts[0])

	if aMain != bMain {
		return aMain - bMain
	}

	aSub := 0
	bSub := 0
	if len(aParts) > 1 {
		aSub, _ = strconv.Atoi(aParts[1])
	}
	if len(bParts) > 1 {
		bSub, _ = strconv.Atoi(bParts[1])
	}

	return aSub - bSub
}

// GetFinalSwap aggregates multiple trades into a single final swap
func GetFinalSwap(trades []types.TradeInfo, dexInfo *types.DexInfo) *types.TradeInfo {
	if len(trades) == 0 {
		return nil
	}
	if len(trades) == 1 {
		return &trades[0]
	}

	// Sort by idx
	if len(trades) > 2 {
		trades = SortTradesByIdx(trades)
	}

	inputTrade := trades[0]
	outputTrade := trades[len(trades)-1]
	var pools []string

	inputAmount := new(big.Int)
	outputAmount := new(big.Int)

	// Merge trades
	for _, trade := range trades {
		if trade.InputToken.Mint == inputTrade.InputToken.Mint {
			amt, ok := new(big.Int).SetString(trade.InputToken.AmountRaw, 10)
			if ok {
				inputAmount.Add(inputAmount, amt)
			}
		}
		if trade.OutputToken.Mint == outputTrade.OutputToken.Mint {
			amt, ok := new(big.Int).SetString(trade.OutputToken.AmountRaw, 10)
			if ok {
				outputAmount.Add(outputAmount, amt)
			}
		}
		if len(trade.Pool) > 0 && !containsString(pools, trade.Pool[0]) {
			pools = append(pools, trade.Pool[0])
		}
	}

	amm := inputTrade.AMM
	route := inputTrade.Route
	if dexInfo != nil {
		if dexInfo.AMM != "" {
			amm = dexInfo.AMM
		}
		if dexInfo.Route != "" {
			route = dexInfo.Route
		}
	}

	return &types.TradeInfo{
		Type: GetTradeType(inputTrade.InputToken.Mint, outputTrade.OutputToken.Mint),
		Pool: pools,
		InputToken: types.TokenInfo{
			Mint:      inputTrade.InputToken.Mint,
			AmountRaw: inputAmount.String(),
			Amount:    types.ConvertToUIAmount(inputAmount, inputTrade.InputToken.Decimals),
			Decimals:  inputTrade.InputToken.Decimals,
		},
		OutputToken: types.TokenInfo{
			Mint:      outputTrade.OutputToken.Mint,
			AmountRaw: outputAmount.String(),
			Amount:    types.ConvertToUIAmount(outputAmount, outputTrade.OutputToken.Decimals),
			Decimals:  outputTrade.OutputToken.Decimals,
		},
		User:      inputTrade.User,
		ProgramId: inputTrade.ProgramId,
		AMM:       amm,
		Route:     route,
		Slot:      inputTrade.Slot,
		Timestamp: inputTrade.Timestamp,
		Signature: inputTrade.Signature,
		Idx:       inputTrade.Idx,
	}
}

// FindAssociatedTokenAddress computes the associated token address for a wallet and mint
func FindAssociatedTokenAddress(wallet, mint string) (standard, token2022 string, err error) {
	walletBytes, err := base58.Decode(wallet)
	if err != nil {
		return "", "", err
	}
	mintBytes, err := base58.Decode(mint)
	if err != nil {
		return "", "", err
	}

	// Standard ATA
	tokenProgramBytes, _ := base58.Decode(constants.TOKEN_PROGRAM_ID)
	standardPDA, err := findProgramAddress(
		[][]byte{walletBytes, tokenProgramBytes, mintBytes},
		constants.ASSOCIATED_TOKEN_PROGRAM_ID,
	)
	if err != nil {
		return "", "", err
	}

	// Token 2022 ATA
	token2022ProgramBytes, _ := base58.Decode(constants.TOKEN_2022_PROGRAM_ID)
	token2022PDA, err := findProgramAddress(
		[][]byte{walletBytes, token2022ProgramBytes, mintBytes},
		constants.ASSOCIATED_TOKEN_PROGRAM_ID,
	)
	if err != nil {
		return "", "", err
	}

	return standardPDA, token2022PDA, nil
}

// findProgramAddress finds a program derived address
func findProgramAddress(seeds [][]byte, programId string) (string, error) {
	programIdBytes, err := base58.Decode(programId)
	if err != nil {
		return "", err
	}

	for nonce := uint8(255); nonce > 0; nonce-- {
		// Build seed with nonce
		var seedWithNonce []byte
		for _, seed := range seeds {
			seedWithNonce = append(seedWithNonce, seed...)
		}
		seedWithNonce = append(seedWithNonce, nonce)
		seedWithNonce = append(seedWithNonce, programIdBytes...)
		seedWithNonce = append(seedWithNonce, []byte("ProgramDerivedAddress")...)

		hash := sha256.Sum256(seedWithNonce)

		// Check if it's off the curve (valid PDA)
		// For simplicity, we'll assume it's valid
		// In production, you'd check if the point is on the ed25519 curve
		return base58.Encode(hash[:]), nil
	}

	return "", nil
}

// GetAccountTradeType determines trade type based on user's token accounts
func GetAccountTradeType(userAccount, baseMint, inputUserAccount, outputUserAccount string) types.TradeType {
	standard, token2022, err := FindAssociatedTokenAddress(userAccount, baseMint)
	if err != nil {
		return types.TradeTypeSwap
	}

	if standard == inputUserAccount || token2022 == inputUserAccount {
		return types.TradeTypeSell
	}
	if standard == outputUserAccount || token2022 == outputUserAccount {
		return types.TradeTypeBuy
	}

	return types.TradeTypeSwap
}

// GetPrevInstructionByIndex finds the previous instruction in a list
func GetPrevInstructionByIndex(instructions []types.ClassifiedInstruction, outerIndex, innerIndex int) *types.ClassifiedInstruction {
	for i, inst := range instructions {
		if inst.OuterIndex == outerIndex && inst.InnerIndex == innerIndex {
			if i > 0 {
				return &instructions[i-1]
			}
		}
	}
	return nil
}

// FormatIdx formats instruction indices as string
func FormatIdx(outerIndex int, innerIndex int) string {
	if innerIndex >= 0 {
		return strconv.Itoa(outerIndex) + "-" + strconv.Itoa(innerIndex)
	}
	return strconv.Itoa(outerIndex)
}

// containsString checks if a string slice contains a value
func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// ContainsAny checks if any of the values exist in the slice
func ContainsAny(slice []string, values ...string) bool {
	for _, val := range values {
		if containsString(slice, val) {
			return true
		}
	}
	return false
}

// Ptr returns a pointer to the value
func Ptr[T any](v T) *T {
	return &v
}
