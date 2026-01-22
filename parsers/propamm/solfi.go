package propamm

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// SolFiParser parses SolFi DEX transactions
type SolFiParser struct {
	adapter                *adapter.TransactionAdapter
	dexInfo                types.DexInfo
	transferActions        map[string][]types.TransferData
	classifiedInstructions []types.ClassifiedInstruction
	txUtils                *utils.TransactionUtils
}

// NewSolFiParser creates a new SolFi parser
func NewSolFiParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *SolFiParser {
	return &SolFiParser{
		adapter:                adapter,
		dexInfo:                dexInfo,
		transferActions:        transferActions,
		classifiedInstructions: classifiedInstructions,
		txUtils:                utils.NewTransactionUtils(adapter),
	}
}

// ProcessTrades processes SolFi trades
func (p *SolFiParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.classifiedInstructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 1 {
			continue
		}

		disc := data[:1]

		// SolFi swap discriminator is 0x07
		if !bytes.Equal(disc, constants.DISCRIMINATORS.SOLFI.SWAP) {
			continue
		}

		trade := p.parseSwap(ci)
		if trade != nil {
			trades = append(trades, *trade)
		}
	}

	return trades
}

// parseSwap parses a SolFi swap instruction
func (p *SolFiParser) parseSwap(ci types.ClassifiedInstruction) *types.TradeInfo {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	// SolFi swap account layout:
	// 0: user (signer)
	// 1: pair
	// 2: poolTokenAccountA
	// 3: poolTokenAccountB
	// 4: userTokenAccountA
	// 5: userTokenAccountB
	// 6: tokenProgram
	// 7: sysvarInstructions
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
		ProgramId: constants.DEX_PROGRAMS.SOLFI.ID,
		AMM:       constants.DEX_PROGRAMS.SOLFI.Name,
		Route:     p.dexInfo.Route,
	}

	trade := p.txUtils.ProcessSwapData(transfers, dexInfo, false)
	if trade != nil {
		trade.Pool = []string{accounts[1]} // pair account
		trade.Idx = utils.FormatIdx(ci.OuterIndex, innerIdx)
		trade = p.txUtils.AttachTokenTransferInfo(trade, p.transferActions)
	}

	return trade
}
