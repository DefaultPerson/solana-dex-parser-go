package utils

import (
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// SumFeeAmounts returns the exact sum of the raw amounts of fee components
// (an amount that is not an integer counts as 0).
func SumFeeAmounts(fees []types.FeeInfo) *big.Int {
	total := new(big.Int)
	for _, f := range fees {
		if v, ok := new(big.Int).SetString(f.AmountRaw, 10); ok {
			total.Add(total, v)
		}
	}
	return total
}

// TotalFee returns fee components charged in one mint as a single fee: the
// exact sum of their raw amounts, with the mint, decimals and dex of the
// first component. It returns nil when there are no components.
func TotalFee(fees []types.FeeInfo) *types.FeeInfo {
	if len(fees) == 0 {
		return nil
	}
	total := SumFeeAmounts(fees)
	first := fees[0]
	return &types.FeeInfo{
		Mint:      first.Mint,
		Amount:    types.ConvertToUIAmount(total, first.Decimals),
		AmountRaw: total.String(),
		Decimals:  first.Decimals,
		Dex:       first.Dex,
	}
}

// FeeComponents returns the distinct fees of a trade: its Fee and Fees,
// except a Fee that is only the total of the Fees charged in its mint or
// that Fees already lists. Three conventions exist: the Pump.fun, PumpSwap,
// LaunchLab, Meteora DBC and meme parsers set Fee to the total of their fee
// components (as upstream does), the AMM parsers set Fee to the LP fee and
// list the other fees in Fees, and the aggregator route trades (Titan, OKX)
// list their Fee in Fees as well. Token-2022 transfer fees ("transferFee")
// never count toward a total.
func FeeComponents(trade *types.TradeInfo) []types.FeeInfo {
	if trade == nil {
		return nil
	}
	var fees []types.FeeInfo
	if trade.Fee != nil && !isFeeTotal(*trade.Fee, trade.Fees) && !containsFee(trade.Fees, *trade.Fee) {
		fees = append(fees, *trade.Fee)
	}
	return append(fees, trade.Fees...)
}

// containsFee reports whether fees holds fee
func containsFee(fees []types.FeeInfo, fee types.FeeInfo) bool {
	for _, f := range fees {
		if f == fee {
			return true
		}
	}
	return false
}

// isFeeTotal reports whether fee is the sum of the components (other than
// transfer fees) of its mint
func isFeeTotal(fee types.FeeInfo, components []types.FeeInfo) bool {
	var same []types.FeeInfo
	for _, f := range components {
		if f.Mint == fee.Mint && f.Type != "transferFee" {
			same = append(same, f)
		}
	}
	amount, ok := new(big.Int).SetString(fee.AmountRaw, 10)
	return ok && len(same) > 0 && fee.Type == "" && SumFeeAmounts(same).Cmp(amount) == 0
}
