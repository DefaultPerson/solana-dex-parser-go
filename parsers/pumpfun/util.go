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

	trade := types.TradeInfo{
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

	// Fees from the TradeEvent (protocol, buyback, creator, cashback), all in
	// the quote mint; Fee is their sum
	if len(event.Fees) > 0 {
		trade.Fees = append([]types.FeeInfo(nil), event.Fees...)
		trade.Fee = types.TotalFee(event.Fees)
	}

	return trade
}

// getPumpswapBuyInfo creates a TradeInfo from a Pumpswap buy event. The input
// is what the user paid: the pool input with the LP fee plus the protocol and
// coin-creator fees (user_quote_amount_in, or quote_amount_in for
// buy_exact_quote_in). Fee is the sum of the listed fee components.
func getPumpswapBuyInfo(
	event *PumpswapBuyEventData,
	inputToken tokenInfo,
	outputToken tokenInfo,
	feeToken tokenInfo,
	info tradeInfoParams,
) types.TradeInfo {
	fees := pumpswapFees(pumpswapFeeParams{
		protocolFee: event.ProtocolFee, buybackFee: event.BuybackFee, coinCreatorFee: event.CoinCreatorFee, cashback: event.Cashback,
		protocolRecipient: event.ProtocolFeeRecipient, coinCreator: event.CoinCreator, user: event.User,
	}, feeToken.Mint, feeToken.Decimals)

	return pumpswapTradeInfo(
		getTradeType(inputToken.Mint, outputToken.Mint), event.Pool, event.User,
		inputToken, event.UserQuoteIn(),
		outputToken, event.BaseAmountOut,
		feeToken, fees, info,
	)
}

// getPumpswapSellInfo creates a TradeInfo from a Pumpswap sell event. The
// output is what the user received (user_quote_amount_out, after all fees).
func getPumpswapSellInfo(
	event *PumpswapSellEventData,
	inputToken tokenInfo,
	outputToken tokenInfo,
	feeToken tokenInfo,
	info tradeInfoParams,
) types.TradeInfo {
	fees := pumpswapFees(pumpswapFeeParams{
		protocolFee: event.ProtocolFee, buybackFee: event.BuybackFee, coinCreatorFee: event.CoinCreatorFee, cashback: event.Cashback,
		protocolRecipient: event.ProtocolFeeRecipient, coinCreator: event.CoinCreator, user: event.User,
	}, feeToken.Mint, feeToken.Decimals)

	return pumpswapTradeInfo(
		getTradeType(inputToken.Mint, outputToken.Mint), event.Pool, event.User,
		inputToken, event.BaseAmountIn,
		outputToken, event.UserQuoteAmountOut,
		feeToken, fees, info,
	)
}

// pumpswapTradeInfo assembles a PumpSwap TradeInfo; Fee is the exact sum of
// fees
func pumpswapTradeInfo(
	tradeType types.TradeType, pool, user string,
	inputToken tokenInfo, inputAmount uint64,
	outputToken tokenInfo, outputAmount uint64,
	feeToken tokenInfo, fees []types.FeeInfo,
	info tradeInfoParams,
) types.TradeInfo {
	programId := info.DexInfo.ProgramId
	if programId == "" {
		programId = constants.DEX_PROGRAMS.PUMP_SWAP.ID
	}

	total := types.SumFeeAmounts(fees)
	return types.TradeInfo{
		Type: tradeType,
		Pool: []string{pool},
		InputToken: types.TokenInfo{
			Mint:      inputToken.Mint,
			Amount:    types.ConvertToUIAmountUint64(inputAmount, inputToken.Decimals),
			AmountRaw: uint64ToString(inputAmount),
			Decimals:  inputToken.Decimals,
		},
		OutputToken: types.TokenInfo{
			Mint:      outputToken.Mint,
			Amount:    types.ConvertToUIAmountUint64(outputAmount, outputToken.Decimals),
			AmountRaw: uint64ToString(outputAmount),
			Decimals:  outputToken.Decimals,
		},
		Fee: &types.FeeInfo{
			Mint:      feeToken.Mint,
			Amount:    types.ConvertToUIAmount(total, feeToken.Decimals),
			AmountRaw: total.String(),
			Decimals:  feeToken.Decimals,
			Dex:       constants.DEX_PROGRAMS.PUMP_SWAP.Name,
		},
		Fees:      fees,
		User:      user,
		ProgramId: programId,
		AMM:       constants.DEX_PROGRAMS.PUMP_SWAP.Name,
		Route:     info.DexInfo.Route,
		Slot:      info.Slot,
		Timestamp: info.Timestamp,
		Signature: info.Signature,
		Idx:       info.Idx,
	}
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
