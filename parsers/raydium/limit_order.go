package raydium

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// RaydiumCLLimitOrderParser reports Raydium CLMM limit orders as transfers.
// A limit order is neither a trade (a taker's swap fills it) nor a liquidity
// event, so the trade and liquidity parsers skip these instructions; the
// tokens they move are reported here, like Jupiter limit orders, each with
// its instruction's idx and a Type naming the action:
//   - openLimitOrder, increaseLimitOrder: the owner's deposit into the input
//     vault
//   - decreaseLimitOrder: the unfilled input and the filled output paid back
//     to the owner
//   - settleLimitOrder: the filled output paid to the owner (by the owner or a
//     keeper)
//
// close_limit_order moves no tokens (it closes the order account) and
// reports nothing.
type RaydiumCLLimitOrderParser struct {
	*parsers.BaseParser
}

// NewRaydiumCLLimitOrderParser creates a Raydium CLMM limit order parser
func NewRaydiumCLLimitOrderParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *RaydiumCLLimitOrderParser {
	return &RaydiumCLLimitOrderParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// clLimitOrderTypes maps the CLMM limit order instructions to their
// transfer types
var clLimitOrderTypes = []struct {
	disc         []byte
	transferType string
}{
	{constants.DISCRIMINATORS.RAYDIUM_CL.LIMIT_ORDER.OPEN_LIMIT_ORDER, "openLimitOrder"},
	{constants.DISCRIMINATORS.RAYDIUM_CL.LIMIT_ORDER.INCREASE_LIMIT_ORDER, "increaseLimitOrder"},
	{constants.DISCRIMINATORS.RAYDIUM_CL.LIMIT_ORDER.DECREASE_LIMIT_ORDER, "decreaseLimitOrder"},
	{constants.DISCRIMINATORS.RAYDIUM_CL.LIMIT_ORDER.SETTLE_LIMIT_ORDER, "settleLimitOrder"},
}

// ProcessTransfers returns the token transfers of the CLMM limit order
// instructions
func (p *RaydiumCLLimitOrderParser) ProcessTransfers() []types.TransferData {
	var transfers []types.TransferData
	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.RAYDIUM_CL.ID {
			continue
		}
		data := p.Adapter.GetInstructionData(ci.Instruction)
		for _, lo := range clLimitOrderTypes {
			if constants.MatchDiscriminator(data, lo.disc) {
				transfers = append(transfers, limitOrderTransfers(p.Utils, p.TransferActions, ci, lo.transferType)...)
				break
			}
		}
	}
	return transfers
}

// limitOrderTransfers returns the token transfers made inside the limit
// order instruction ci, labelled with transferType, the program and ci's idx
func limitOrderTransfers(tu *utils.TransactionUtils, transferActions map[string][]types.TransferData, ci types.ClassifiedInstruction, transferType string) []types.TransferData {
	transfers := tu.CPIGroupTransfers(transferActions, ci)
	for i := range transfers {
		transfers[i].Type = transferType
		transfers[i].ProgramId = ci.ProgramId
		transfers[i].Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	}
	return transfers
}
