package pumpfun

import (
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// tradeInfoParams holds parameters for creating trade info
type tradeInfoParams struct {
	Slot      uint64
	Signature string
	Timestamp int64
	Idx       string
	DexInfo   types.DexInfo
}

// getPumpfunTradeInfo creates a TradeInfo from a MemeEvent
func getPumpfunTradeInfo(event *types.MemeEvent, info tradeInfoParams) types.TradeInfo {
	var pool []string
	if event.BondingCurve != "" {
		pool = []string{event.BondingCurve}
	}

	amm := info.DexInfo.AMM
	if amm == "" {
		amm = constants.DEX_PROGRAMS.PUMP_FUN.Name
	}

	return types.TradeInfo{
		Type:        event.Type,
		Pool:        pool,
		InputToken:  *event.InputToken,
		OutputToken: *event.OutputToken,
		User:        event.User,
		ProgramId:   constants.DEX_PROGRAMS.PUMP_FUN.ID,
		AMM:         amm,
		Route:       info.DexInfo.Route,
		Slot:        info.Slot,
		Timestamp:   info.Timestamp,
		Signature:   info.Signature,
		Idx:         info.Idx,
	}
}

// getPumpswapBuyInfo creates a TradeInfo from a Pumpswap buy event
func getPumpswapBuyInfo(
	event *PumpswapBuyEventData,
	inputToken tokenInfo,
	outputToken tokenInfo,
	feeToken tokenInfo,
	info tradeInfoParams,
) types.TradeInfo {
	feeAmt := new(big.Int).Add(
		new(big.Int).SetUint64(event.ProtocolFee),
		new(big.Int).SetUint64(event.CoinCreatorFee),
	)

	tradeType := getTradeType(inputToken.Mint, outputToken.Mint)
	inputUIAmount := types.ConvertToUIAmountUint64(event.QuoteAmountInWithLpFee, inputToken.Decimals)
	outputUIAmount := types.ConvertToUIAmountUint64(event.BaseAmountOut, outputToken.Decimals)
	feeUIAmount := types.ConvertToUIAmountUint64(feeAmt.Uint64(), feeToken.Decimals)
	protocolFeeUIAmount := types.ConvertToUIAmountUint64(event.ProtocolFee, feeToken.Decimals)

	programId := info.DexInfo.ProgramId
	if programId == "" {
		programId = constants.DEX_PROGRAMS.PUMP_SWAP.ID
	}

	trade := types.TradeInfo{
		Type: tradeType,
		Pool: []string{event.Pool},
		InputToken: types.TokenInfo{
			Mint:      inputToken.Mint,
			Amount:    inputUIAmount,
			AmountRaw: uint64ToString(event.QuoteAmountInWithLpFee),
			Decimals:  inputToken.Decimals,
		},
		OutputToken: types.TokenInfo{
			Mint:      outputToken.Mint,
			Amount:    outputUIAmount,
			AmountRaw: uint64ToString(event.BaseAmountOut),
			Decimals:  outputToken.Decimals,
		},
		Fee: &types.FeeInfo{
			Mint:      feeToken.Mint,
			Amount:    feeUIAmount,
			AmountRaw: feeAmt.String(),
			Decimals:  feeToken.Decimals,
		},
		Fees: []types.FeeInfo{
			{
				Mint:      feeToken.Mint,
				Amount:    protocolFeeUIAmount,
				AmountRaw: uint64ToString(event.ProtocolFee),
				Decimals:  feeToken.Decimals,
				Dex:       constants.DEX_PROGRAMS.PUMP_SWAP.Name,
				Type:      "protocol",
				Recipient: event.ProtocolFeeRecipient,
			},
		},
		User:      event.User,
		ProgramId: programId,
		AMM:       constants.DEX_PROGRAMS.PUMP_SWAP.Name,
		Route:     info.DexInfo.Route,
		Slot:      info.Slot,
		Timestamp: info.Timestamp,
		Signature: info.Signature,
		Idx:       info.Idx,
	}

	// Add creator fee if present
	if event.CoinCreatorFee > 0 {
		creatorFeeUIAmount := types.ConvertToUIAmountUint64(event.CoinCreatorFee, feeToken.Decimals)
		trade.Fees = append(trade.Fees, types.FeeInfo{
			Mint:      feeToken.Mint,
			Amount:    creatorFeeUIAmount,
			AmountRaw: uint64ToString(event.CoinCreatorFee),
			Decimals:  feeToken.Decimals,
			Dex:       constants.DEX_PROGRAMS.PUMP_SWAP.Name,
			Type:      "coinCreator",
			Recipient: event.CoinCreator,
		})
	}

	return trade
}

// getPumpswapSellInfo creates a TradeInfo from a Pumpswap sell event
func getPumpswapSellInfo(
	event *PumpswapSellEventData,
	inputToken tokenInfo,
	outputToken tokenInfo,
	feeToken tokenInfo,
	info tradeInfoParams,
) types.TradeInfo {
	feeAmt := new(big.Int).Add(
		new(big.Int).SetUint64(event.ProtocolFee),
		new(big.Int).SetUint64(event.CoinCreatorFee),
	)

	tradeType := getTradeType(inputToken.Mint, outputToken.Mint)
	inputUIAmount := types.ConvertToUIAmountUint64(event.BaseAmountIn, inputToken.Decimals)
	outputUIAmount := types.ConvertToUIAmountUint64(event.UserQuoteAmountOut, outputToken.Decimals)
	feeUIAmount := types.ConvertToUIAmountUint64(feeAmt.Uint64(), feeToken.Decimals)
	protocolFeeUIAmount := types.ConvertToUIAmountUint64(event.ProtocolFee, feeToken.Decimals)

	programId := info.DexInfo.ProgramId
	if programId == "" {
		programId = constants.DEX_PROGRAMS.PUMP_SWAP.ID
	}

	trade := types.TradeInfo{
		Type: tradeType,
		Pool: []string{event.Pool},
		InputToken: types.TokenInfo{
			Mint:      inputToken.Mint,
			Amount:    inputUIAmount,
			AmountRaw: uint64ToString(event.BaseAmountIn),
			Decimals:  inputToken.Decimals,
		},
		OutputToken: types.TokenInfo{
			Mint:      outputToken.Mint,
			Amount:    outputUIAmount,
			AmountRaw: uint64ToString(event.UserQuoteAmountOut),
			Decimals:  outputToken.Decimals,
		},
		Fee: &types.FeeInfo{
			Mint:      feeToken.Mint,
			Amount:    feeUIAmount,
			AmountRaw: uint64ToString(event.ProtocolFee),
			Decimals:  feeToken.Decimals,
			Dex:       constants.DEX_PROGRAMS.PUMP_SWAP.Name,
		},
		Fees: []types.FeeInfo{
			{
				Mint:      feeToken.Mint,
				Amount:    protocolFeeUIAmount,
				AmountRaw: uint64ToString(event.ProtocolFee),
				Decimals:  feeToken.Decimals,
				Dex:       constants.DEX_PROGRAMS.PUMP_SWAP.Name,
				Type:      "protocol",
				Recipient: event.ProtocolFeeRecipient,
			},
		},
		User:      event.User,
		ProgramId: programId,
		AMM:       constants.DEX_PROGRAMS.PUMP_SWAP.Name,
		Route:     info.DexInfo.Route,
		Slot:      info.Slot,
		Timestamp: info.Timestamp,
		Signature: info.Signature,
		Idx:       info.Idx,
	}

	// Add creator fee if present
	if event.CoinCreatorFee > 0 {
		creatorFeeUIAmount := types.ConvertToUIAmountUint64(event.CoinCreatorFee, feeToken.Decimals)
		trade.Fees = append(trade.Fees, types.FeeInfo{
			Mint:      feeToken.Mint,
			Amount:    creatorFeeUIAmount,
			AmountRaw: uint64ToString(event.CoinCreatorFee),
			Decimals:  feeToken.Decimals,
			Dex:       constants.DEX_PROGRAMS.PUMP_SWAP.Name,
			Type:      "coinCreator",
			Recipient: event.CoinCreator,
		})
	}

	return trade
}

// tokenInfo holds token information
type tokenInfo struct {
	Mint     string
	Decimals uint8
}

// getTradeType determines trade type from mints
func getTradeType(inputMint, outputMint string) types.TradeType {
	if inputMint == constants.TOKENS.SOL || inputMint == constants.TOKENS.USDC || inputMint == constants.TOKENS.USDT {
		return types.TradeTypeBuy
	}
	return types.TradeTypeSell
}

func uint64ToString(v uint64) string {
	return new(big.Int).SetUint64(v).String()
}
