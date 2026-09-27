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
