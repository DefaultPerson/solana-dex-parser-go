package propamm

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// GoonFiParser parses GoonFi DEX transactions
type GoonFiParser struct {
	adapter                *adapter.TransactionAdapter
	dexInfo                types.DexInfo
	transferActions        map[string][]types.TransferData
	classifiedInstructions []types.ClassifiedInstruction
	txUtils                *utils.TransactionUtils
}

// NewGoonFiParser creates a new GoonFi parser
func NewGoonFiParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *GoonFiParser {
	return &GoonFiParser{
		adapter:                adapter,
		dexInfo:                dexInfo,
		transferActions:        transferActions,
		classifiedInstructions: classifiedInstructions,
		txUtils:                utils.NewTransactionUtils(adapter),
	}
}

// ProcessTrades processes GoonFi trades
func (p *GoonFiParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.classifiedInstructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 1 {
			continue
		}

		disc := data[:1]

		// GoonFi swap discriminator is 0x02
		if !bytes.Equal(disc, constants.DISCRIMINATORS.GOONFI.SWAP) {
			continue
		}

		trade := p.parseSwap(ci)
		if trade != nil {
			trades = append(trades, *trade)
		}
	}

	return trades
}

// parseSwap parses a GoonFi swap instruction
func (p *GoonFiParser) parseSwap(ci types.ClassifiedInstruction) *types.TradeInfo {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	// GoonFi swap account layout:
	// 0: user (signer)
	// 1: market
	// 2: userTokenAccountA
	// 3: userTokenAccountB
	// 4: poolTokenAccountA
	// 5: poolTokenAccountB
	// 6: account
	// 7: sysvarInstructions
	// 8: tokenProgram
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
		ProgramId: constants.DEX_PROGRAMS.GOONFI.ID,
		AMM:       constants.DEX_PROGRAMS.GOONFI.Name,
		Route:     p.dexInfo.Route,
	}

	trade := p.txUtils.ProcessSwapData(transfers, dexInfo, false)
	if trade != nil {
		trade.Pool = []string{accounts[1]} // market account
		trade.Idx = utils.FormatIdx(ci.OuterIndex, innerIdx)
		trade = p.txUtils.AttachTokenTransferInfo(trade, p.transferActions)
	}

	return trade
}
