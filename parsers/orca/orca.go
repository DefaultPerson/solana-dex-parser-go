package orca

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// OrcaParser parses Orca swap transactions
type OrcaParser struct {
	*parsers.BaseParser
}

// NewOrcaParser creates a new Orca parser
func NewOrcaParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *OrcaParser {
	return &OrcaParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// ProcessTrades parses Orca swap trades
func (p *OrcaParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.ORCA.ID && p.notLiquidityEvent(ci.Instruction) {
			// For outer instructions (InnerIndex = -1), don't set to 0 to match the groupKey
			transfers := p.GetTransfersForInstruction(ci.ProgramId, ci.OuterIndex, ci.InnerIndex, nil)

			if len(transfers) >= 2 {
				dexInfo := p.DexInfo
				if dexInfo.AMM == "" {
					dexInfo.AMM = constants.GetProgramName(ci.ProgramId)
				}

				trade := p.Utils.ProcessSwapData(transfers, dexInfo, false)
				if trade != nil {
					trades = append(trades, *p.Utils.AttachTokenTransferInfo(trade, p.TransferActions))
				}
			}
		}
	}

	return trades
}

// notLiquidityEvent checks if instruction is NOT a liquidity event
func (p *OrcaParser) notLiquidityEvent(instruction interface{}) bool {
	data := p.Adapter.GetInstructionData(instruction)
	if len(data) < 8 {
		return true
	}

	disc := data[:8]

	// Check all Orca discriminators
	orcaDiscs := [][]byte{
		constants.DISCRIMINATORS.ORCA.CREATE,
		constants.DISCRIMINATORS.ORCA.CREATE2,
		constants.DISCRIMINATORS.ORCA.ADD_LIQUIDITY,
		constants.DISCRIMINATORS.ORCA.ADD_LIQUIDITY2,
		constants.DISCRIMINATORS.ORCA.REMOVE_LIQUIDITY,
		constants.DISCRIMINATORS.ORCA.OTHER1,
		constants.DISCRIMINATORS.ORCA.OTHER2,
	}

	for _, d := range orcaDiscs {
		if bytes.Equal(disc, d) {
			return false
		}
	}

	return true
}
