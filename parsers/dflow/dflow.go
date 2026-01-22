package dflow

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// DFlowParser parses DFlow aggregator transactions
type DFlowParser struct {
	adapter                *adapter.TransactionAdapter
	dexInfo                types.DexInfo
	transferActions        map[string][]types.TransferData
	classifiedInstructions []types.ClassifiedInstruction
	txUtils                *utils.TransactionUtils
}

// NewDFlowParser creates a new DFlow parser
func NewDFlowParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *DFlowParser {
	return &DFlowParser{
		adapter:                adapter,
		dexInfo:                dexInfo,
		transferActions:        transferActions,
		classifiedInstructions: classifiedInstructions,
		txUtils:                utils.NewTransactionUtils(adapter),
	}
}

// ProcessTrades processes DFlow trades
func (p *DFlowParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.classifiedInstructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		disc := data[:8]

		// Check for swap discriminators
		isSwap := bytes.Equal(disc, constants.DISCRIMINATORS.DFLOW.SWAP) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.DFLOW.SWAP2) ||
			bytes.Equal(disc, constants.DISCRIMINATORS.DFLOW.SWAP_WITH_DEST)

		isFillOrder := bytes.Equal(disc, constants.DISCRIMINATORS.DFLOW.FILL_ORDER)

		if !isSwap && !isFillOrder {
			continue
		}

		trade := p.parseSwap(ci)
		if trade != nil {
			trades = append(trades, *trade)
		}
	}

	return trades
}

// parseSwap parses a DFlow swap instruction
func (p *DFlowParser) parseSwap(ci types.ClassifiedInstruction) *types.TradeInfo {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	// DFlow swap account layout varies by instruction type
	// Minimum accounts needed
	if len(accounts) < 4 {
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
		ProgramId: constants.DEX_PROGRAMS.DFLOW.ID,
		AMM:       constants.DEX_PROGRAMS.DFLOW.Name,
		Route:     constants.DEX_PROGRAMS.DFLOW.Name,
	}

	trade := p.txUtils.ProcessSwapData(transfers, dexInfo, false)
	if trade != nil {
		trade.Route = constants.DEX_PROGRAMS.DFLOW.Name
		trade.Idx = utils.FormatIdx(ci.OuterIndex, innerIdx)
		trade = p.txUtils.AttachTokenTransferInfo(trade, p.transferActions)
	}

	return trade
}
