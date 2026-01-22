package propamm

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// ObricParser parses Obric V2 DEX transactions
type ObricParser struct {
	adapter                *adapter.TransactionAdapter
	dexInfo                types.DexInfo
	transferActions        map[string][]types.TransferData
	classifiedInstructions []types.ClassifiedInstruction
	txUtils                *utils.TransactionUtils
}

// NewObricParser creates a new Obric parser
func NewObricParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *ObricParser {
	return &ObricParser{
		adapter:                adapter,
		dexInfo:                dexInfo,
		transferActions:        transferActions,
		classifiedInstructions: classifiedInstructions,
		txUtils:                utils.NewTransactionUtils(adapter),
	}
}

// ProcessTrades processes Obric trades
func (p *ObricParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.classifiedInstructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		disc := data[:8]

		// Check for swap discriminators
		isSwap := bytes.Equal(disc, constants.DISCRIMINATORS.OBRIC.SWAP) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.OBRIC.SWAP_X_TO_Y) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.OBRIC.SWAP_Y_TO_X)

		if !isSwap {
			continue
		}

		trade := p.parseSwap(ci)
		if trade != nil {
			trades = append(trades, *trade)
		}
	}

	return trades
}

// parseSwap parses an Obric swap instruction
func (p *ObricParser) parseSwap(ci types.ClassifiedInstruction) *types.TradeInfo {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	// Obric swap account layout (based on IDL):
	// 0: user (signer)
	// 1: tradingPair
	// 2: userTokenAccountX
	// 3: userTokenAccountY
	// 4: poolTokenAccountX
	// 5: poolTokenAccountY
	// 6: tokenProgram
	if len(accounts) < 6 {
		return nil
	}

	innerIdx := ci.InnerIndex
	if innerIdx < 0 {
		innerIdx = 0
	}

	// Get transfers for this instruction
	transfers := p.txUtils.GetTransfersForInstruction(
		p.transferActions,
		ci.ProgramId,
		ci.OuterIndex,
		ci.InnerIndex,
		nil,
	)

	if len(transfers) < 2 {
		return nil
	}

	dexInfo := types.DexInfo{
		ProgramId: constants.DEX_PROGRAMS.OBRIC_V2.ID,
		AMM:       constants.DEX_PROGRAMS.OBRIC_V2.Name,
		Route:     p.dexInfo.Route,
	}

	trade := p.txUtils.ProcessSwapData(transfers, dexInfo, false)
	if trade != nil {
		trade.Pool = []string{accounts[1]} // tradingPair account
		trade.Idx = utils.FormatIdx(ci.OuterIndex, innerIdx)
		trade = p.txUtils.AttachTokenTransferInfo(trade, p.transferActions)
	}

	return trade
}
