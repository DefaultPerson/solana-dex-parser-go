package meteora

import (
	"bytes"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/parsers"
	"github.com/solana-dex-parser-go/types"
)

// MeteoraParser parses Meteora swap transactions
type MeteoraParser struct {
	*parsers.BaseParser
}

// NewMeteoraParser creates a new Meteora parser
func NewMeteoraParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MeteoraParser {
	return &MeteoraParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// ProcessTrades parses Meteora swap trades
func (p *MeteoraParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if isMeteoraProgram(ci.ProgramId) && p.notLiquidityEvent(ci.Instruction) {
			// For outer instructions (InnerIndex = -1), don't set to 0 to match the groupKey
			transfers := p.GetTransfersForInstruction(ci.ProgramId, ci.OuterIndex, ci.InnerIndex, nil)

			if len(transfers) >= 2 {
				// For METEORA (DLMM), take only first 2 transfers
				if ci.ProgramId == constants.DEX_PROGRAMS.METEORA.ID {
					transfers = transfers[:2]
				}

				dexInfo := p.DexInfo
				if dexInfo.AMM == "" {
					dexInfo.AMM = constants.GetProgramName(ci.ProgramId)
				}

				trade := p.Utils.ProcessSwapData(transfers, dexInfo, false)
				if trade != nil {
					pool := p.getPoolAddress(ci.Instruction, ci.ProgramId)
					if pool != "" {
						trade.Pool = []string{pool}
					}
					trades = append(trades, *p.Utils.AttachTokenTransferInfo(trade, p.TransferActions))
				}
			}
		}
	}

	return trades
}

// isMeteoraProgram checks if programId is a Meteora program
func isMeteoraProgram(programId string) bool {
	return programId == constants.DEX_PROGRAMS.METEORA.ID ||
		programId == constants.DEX_PROGRAMS.METEORA_DAMM.ID ||
		programId == constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID
}

// getPoolAddress gets pool address from instruction accounts
func (p *MeteoraParser) getPoolAddress(instruction interface{}, programId string) string {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	if len(accounts) > 5 {
		switch programId {
		case constants.DEX_PROGRAMS.METEORA_DAMM.ID, constants.DEX_PROGRAMS.METEORA.ID:
			return accounts[0]
		case constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID:
			return accounts[1]
		}
	}
	return ""
}

// notLiquidityEvent checks if instruction is NOT a liquidity event
func (p *MeteoraParser) notLiquidityEvent(instruction interface{}) bool {
	data := p.Adapter.GetInstructionData(instruction)
	if len(data) == 0 {
		return true
	}

	// Check DLMM liquidity discriminators
	if len(data) >= 8 {
		disc8 := data[:8]

		// Check ADD_LIQUIDITY discriminators
		for _, d := range constants.DISCRIMINATORS.METEORA_DLMM.ADD_LIQUIDITY {
			if bytes.Equal(disc8, d) {
				return false
			}
		}

		// Check REMOVE_LIQUIDITY discriminators
		for _, d := range constants.DISCRIMINATORS.METEORA_DLMM.REMOVE_LIQUIDITY {
			if bytes.Equal(disc8, d) {
				return false
			}
		}

		// Check DAMM discriminators
		if bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM.CREATE) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM.ADD_LIQUIDITY) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM.ADD_IMBALANCE_LIQUIDITY) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM.REMOVE_LIQUIDITY) {
			return false
		}

		// Check DAMM_V2 discriminators
		if bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM_V2.INITIALIZE_POOL) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM_V2.INITIALIZE_CUSTOM_POOL) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM_V2.INITIALIZE_POOL_WITH_DYNAMIC_CONFIG) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM_V2.ADD_LIQUIDITY) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM_V2.CLAIM_POSITION_FEE) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM_V2.REMOVE_LIQUIDITY) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.METEORA_DAMM_V2.REMOVE_ALL_LIQUIDITY) {
			return false
		}
	}

	return true
}
