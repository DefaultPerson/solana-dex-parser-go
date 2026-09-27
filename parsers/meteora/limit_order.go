package meteora

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// MeteoraDLMMLimitOrderParser reports Meteora DLMM limit orders as
// transfers. A limit order is neither a trade (a taker's swap fills it) nor
// a liquidity event, so the trade and liquidity parsers skip these
// instructions; the tokens they move are reported here, like Jupiter limit
// orders, each with its instruction's idx and a Type naming the action:
//   - placeLimitOrder: the owner's deposit into the pair's reserve
//   - cancelLimitOrder: what the reserves pay back to the owner (unfilled
//     and filled amounts)
//
// close_limit_order_if_empty moves no tokens and reports nothing.
type MeteoraDLMMLimitOrderParser struct {
	*parsers.BaseParser
}

// NewMeteoraDLMMLimitOrderParser creates a Meteora DLMM limit order parser
func NewMeteoraDLMMLimitOrderParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MeteoraDLMMLimitOrderParser {
	return &MeteoraDLMMLimitOrderParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// dlmmLimitOrderTypes maps the DLMM limit order instructions to their
// transfer types
var dlmmLimitOrderTypes = []struct {
	disc         []byte
	transferType string
}{
	{constants.DISCRIMINATORS.METEORA_DLMM.LIMIT_ORDER["placeLimitOrder"], "placeLimitOrder"},
	{constants.DISCRIMINATORS.METEORA_DLMM.LIMIT_ORDER["cancelLimitOrder"], "cancelLimitOrder"},
}

// ProcessTransfers returns the token transfers of the DLMM limit order
// instructions
func (p *MeteoraDLMMLimitOrderParser) ProcessTransfers() []types.TransferData {
	var transfers []types.TransferData
	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.METEORA.ID {
			continue
		}
		data := p.Adapter.GetInstructionData(ci.Instruction)
		for _, lo := range dlmmLimitOrderTypes {
			if !constants.MatchDiscriminator(data, lo.disc) {
				continue
			}
			for _, t := range p.Utils.CPIGroupTransfers(p.TransferActions, ci) {
				t.Type = lo.transferType
				t.ProgramId = ci.ProgramId
				t.Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
				transfers = append(transfers, t)
			}
			break
		}
	}
	return transfers
}
