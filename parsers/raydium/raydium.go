package raydium

import (
	"bytes"

	"github.com/solana-dex-parser-go/adapter"
	"github.com/solana-dex-parser-go/constants"
	"github.com/solana-dex-parser-go/parsers"
	"github.com/solana-dex-parser-go/types"
)

// RaydiumParser parses Raydium swap transactions
type RaydiumParser struct {
	*parsers.BaseParser
}

// NewRaydiumParser creates a new Raydium parser
func NewRaydiumParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *RaydiumParser {
	return &RaydiumParser{
		BaseParser: parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
	}
}

// ProcessTrades parses Raydium swap trades
func (p *RaydiumParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if p.notLiquidityEvent(ci.Instruction) {
			// For outer instructions (InnerIndex = -1), don't set to 0 to match the groupKey
			transfers := p.GetTransfersForInstruction(ci.ProgramId, ci.OuterIndex, ci.InnerIndex, nil)

			if len(transfers) >= 2 {
				dexInfo := p.DexInfo
				if dexInfo.AMM == "" {
					dexInfo.AMM = constants.GetProgramName(ci.ProgramId)
				}

				trade := p.Utils.ProcessSwapData(transfers[:2], dexInfo, false)

				if trade != nil {
					pool := p.getPoolAddress(ci.Instruction, ci.ProgramId)
					if pool != "" {
						trade.Pool = []string{pool}
					}
					if len(transfers) > 2 {
						tokenInfo := p.Utils.GetTransferTokenInfo(&transfers[2])
						if tokenInfo != nil {
							trade.Fee = &types.FeeInfo{
								Mint:      tokenInfo.Mint,
								Amount:    tokenInfo.Amount,
								AmountRaw: tokenInfo.AmountRaw,
								Decimals:  tokenInfo.Decimals,
							}
						}
					}
					trades = append(trades, *p.Utils.AttachTokenTransferInfo(trade, p.TransferActions))
				}
			}
		}
	}

	return trades
}

// getPoolAddress gets pool address from instruction accounts
func (p *RaydiumParser) getPoolAddress(instruction interface{}, programId string) string {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	if len(accounts) > 5 {
		switch programId {
		case constants.DEX_PROGRAMS.RAYDIUM_V4.ID, constants.DEX_PROGRAMS.RAYDIUM_AMM.ID:
			return accounts[1]
		case constants.DEX_PROGRAMS.RAYDIUM_CL.ID:
			return accounts[2]
		case constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID:
			return accounts[3]
		}
	}
	return ""
}

// notLiquidityEvent checks if instruction is NOT a liquidity event
func (p *RaydiumParser) notLiquidityEvent(instruction interface{}) bool {
	data := p.Adapter.GetInstructionData(instruction)
	if len(data) == 0 {
		return true
	}

	// Check Raydium V4 discriminators (1 byte)
	if len(data) >= 1 {
		disc := data[:1]
		if bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.CREATE) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.ADD_LIQUIDITY) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.RAYDIUM.REMOVE_LIQUIDITY) {
			return false
		}
	}

	// Check Raydium CL discriminators (8 bytes)
	if len(data) >= 8 {
		disc8 := data[:8]
		// CREATE discriminators
		for _, d := range [][]byte{
			constants.DISCRIMINATORS.RAYDIUM_CL.CREATE.OPEN_POSITION,
			constants.DISCRIMINATORS.RAYDIUM_CL.CREATE.OPEN_POSITION_V2,
			constants.DISCRIMINATORS.RAYDIUM_CL.CREATE.CREATE_POOL,
			constants.DISCRIMINATORS.RAYDIUM_CL.CREATE.INITIALIZE,
		} {
			if bytes.Equal(disc8, d) {
				return false
			}
		}
		// ADD_LIQUIDITY discriminators
		for _, d := range [][]byte{
			constants.DISCRIMINATORS.RAYDIUM_CL.ADD_LIQUIDITY.INCREASE_LIQUIDITY,
			constants.DISCRIMINATORS.RAYDIUM_CL.ADD_LIQUIDITY.INCREASE_LIQUIDITY_V2,
		} {
			if bytes.Equal(disc8, d) {
				return false
			}
		}
		// REMOVE_LIQUIDITY discriminators
		for _, d := range [][]byte{
			constants.DISCRIMINATORS.RAYDIUM_CL.REMOVE_LIQUIDITY.DECREASE_LIQUIDITY,
			constants.DISCRIMINATORS.RAYDIUM_CL.REMOVE_LIQUIDITY.DECREASE_LIQUIDITY_V2,
		} {
			if bytes.Equal(disc8, d) {
				return false
			}
		}

		// CPMM discriminators
		if bytes.Equal(disc8, constants.DISCRIMINATORS.RAYDIUM_CPMM.CREATE) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.RAYDIUM_CPMM.ADD_LIQUIDITY) ||
			bytes.Equal(disc8, constants.DISCRIMINATORS.RAYDIUM_CPMM.REMOVE_LIQUIDITY) {
			return false
		}
	}

	return true
}
