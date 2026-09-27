package orca

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
	"github.com/mr-tron/base58"
)

// OrcaParser parses Orca swap transactions
type OrcaParser struct {
	*parsers.BaseParser
	logs []utils.ProgramLog
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

// orcaSwap describes a Whirlpool swap instruction (whirlpool 0.9.0 IDL, as
// in the solana-streamer IDL; the account counts are the minimum an
// instruction of that kind carries)
type orcaSwap struct {
	minAccounts int
	pools       []int // whirlpool accounts, one per hop
	// token accounts to read the mints from: for each hop the vault of
	// token A and of token B (v1), or input and output mint accounts (v2)
	vaultsA, vaultsB []int
	mintIn, mintOut  int // two_hop_swap_v2: token_mint_input / token_mint_output
	mintA, mintB     int // swap_v2: token_mint_a / token_mint_b
}

// getSwap returns the swap layout of an Orca instruction, or nil when it is
// not a swap. Only swaps are trades: liquidity, fee and reward instructions
// also move tokens and must not become trades.
func getSwap(data []byte) *orcaSwap {
	if len(data) < 8 {
		return nil
	}
	disc := data[:8]
	orca := constants.DISCRIMINATORS.ORCA
	switch {
	case bytes.Equal(disc, orca.SWAP):
		// token_program 0, token_authority 1, whirlpool 2, token_owner_account_a 3,
		// token_vault_a 4, token_owner_account_b 5, token_vault_b 6, ...
		return &orcaSwap{minAccounts: 11, pools: []int{2}, vaultsA: []int{4}, vaultsB: []int{6}, mintIn: -1, mintOut: -1, mintA: -1, mintB: -1}
	case bytes.Equal(disc, orca.SWAP_V2):
		// token_program_a 0, token_program_b 1, memo 2, token_authority 3, whirlpool 4,
		// token_mint_a 5, token_mint_b 6, ...
		return &orcaSwap{minAccounts: 15, pools: []int{4}, mintIn: -1, mintOut: -1, mintA: 5, mintB: 6}
	case bytes.Equal(disc, orca.TWO_HOP_SWAP):
		// token_program 0, token_authority 1, whirlpool_one 2, whirlpool_two 3,
		// owner_one_a 4, vault_one_a 5, owner_one_b 6, vault_one_b 7,
		// owner_two_a 8, vault_two_a 9, owner_two_b 10, vault_two_b 11, ...
		return &orcaSwap{minAccounts: 20, pools: []int{2, 3}, vaultsA: []int{5, 9}, vaultsB: []int{7, 11}, mintIn: -1, mintOut: -1, mintA: -1, mintB: -1}
	case bytes.Equal(disc, orca.TWO_HOP_SWAP_V2):
		// whirlpool_one 0, whirlpool_two 1, token_mint_input 2,
		// token_mint_intermediate 3, token_mint_output 4, ...
		return &orcaSwap{minAccounts: 24, pools: []int{0, 1}, mintIn: 2, mintOut: 4, mintA: -1, mintB: -1}
	}
	return nil
}

// ProcessTrades parses Orca swap trades. Amounts and fees come from the
// Traded events the program logs (one per hop); without them (older program
// versions, truncated logs) the instruction's transfers are used.
func (p *OrcaParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.ORCA.ID {
			continue
		}
		swap := getSwap(p.Adapter.GetInstructionData(ci.Instruction))
		if swap == nil {
			continue
		}
		accounts := p.Adapter.GetInstructionAccounts(ci.Instruction)
		transfers := p.GetTransfersForInstruction(ci.ProgramId, ci.OuterIndex, ci.InnerIndex, nil)

		dexInfo := p.DexInfo
		if dexInfo.AMM == "" {
			dexInfo.AMM = constants.GetProgramName(ci.ProgramId)
		}

		idx := ci.GetIdx()
		if len(transfers) > 0 {
			idx = transfers[0].Idx
		}
		trade := p.tradeFromEvents(ci, swap, accounts, dexInfo, idx)
		if trade == nil {
			if len(transfers) < 2 {
				continue
			}
			trade = p.Utils.ProcessSwapData(transfers, dexInfo, false)
			if trade == nil {
				continue
			}
		}
		if len(accounts) >= swap.minAccounts {
			for _, i := range swap.pools {
				trade.Pool = append(trade.Pool, accounts[i])
			}
		}
		trade = p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
		trades = append(trades, *p.Utils.AttachInstructionTransfers(trade, transfers))
	}

	return trades
}

// tradedEvent is the Whirlpool Traded event: whirlpool, a_to_b,
// pre/post_sqrt_price (u128), input_amount, output_amount,
// input_transfer_fee, output_transfer_fee, lp_fee, protocol_fee
type tradedEvent struct {
	whirlpool         string
	aToB              bool
	inputAmount       uint64
	outputAmount      uint64
	inputTransferFee  uint64
	outputTransferFee uint64
	lpFee             uint64
	protocolFee       uint64
}

const tradedEventSize = 8 + 32 + 1 + 16 + 16 + 6*8

func decodeTradedEvent(data []byte) *tradedEvent {
	if len(data) < tradedEventSize || !bytes.Equal(data[:8], constants.DISCRIMINATORS.ORCA.TRADED_EVENT) {
		return nil
	}
	u64 := func(offset int) uint64 { return binary.LittleEndian.Uint64(data[offset : offset+8]) }
	const amounts = 8 + 32 + 1 + 32
	return &tradedEvent{
		whirlpool:         base58.Encode(data[8:40]),
		aToB:              data[40] != 0,
		inputAmount:       u64(amounts),
		outputAmount:      u64(amounts + 8),
		inputTransferFee:  u64(amounts + 16),
		outputTransferFee: u64(amounts + 24),
		lpFee:             u64(amounts + 32),
		protocolFee:       u64(amounts + 40),
	}
}

// tradeFromEvents builds the trade of a swap instruction from its Traded
// events. The user sends input_amount of the first hop (it includes the
// Token-2022 transfer fee) and receives output_amount minus
// output_transfer_fee of the last hop. Fee is the LP fee of a single-hop
// swap; the protocol fee, transfer fees and the fees of two-hop swaps are
// in Fees, each in the hop's input mint.
func (p *OrcaParser) tradeFromEvents(ci types.ClassifiedInstruction, swap *orcaSwap, accounts []string, dexInfo types.DexInfo, idx string) *types.TradeInfo {
	if len(accounts) < swap.minAccounts {
		return nil
	}
	if p.logs == nil {
		p.logs = p.Utils.GetProgramDataLogs()
	}
	var events []*tradedEvent
	for _, l := range utils.FindProgramLogs(p.logs, ci.ProgramId, ci.OuterIndex, ci.InnerIndex) {
		if ev := decodeTradedEvent(l.Data); ev != nil {
			events = append(events, ev)
		}
	}
	if len(events) != len(swap.pools) {
		return nil
	}
	for i, ev := range events {
		if ev.whirlpool != accounts[swap.pools[i]] {
			return nil // not the events of this instruction
		}
	}

	// input and output mint of each hop
	hopMints := make([][2]string, len(events))
	for i, ev := range events {
		var mintA, mintB string
		switch {
		case swap.mintA >= 0:
			mintA, mintB = accounts[swap.mintA], accounts[swap.mintB]
		case swap.vaultsA != nil:
			mintA = p.Adapter.GetSplTokenMint(accounts[swap.vaultsA[i]])
			mintB = p.Adapter.GetSplTokenMint(accounts[swap.vaultsB[i]])
		}
		if ev.aToB {
			hopMints[i] = [2]string{mintA, mintB}
		} else {
			hopMints[i] = [2]string{mintB, mintA}
		}
	}
	if swap.mintIn >= 0 {
		hopMints[0][0] = accounts[swap.mintIn]
		hopMints[len(hopMints)-1][1] = accounts[swap.mintOut]
	}
	inputMint, outputMint := hopMints[0][0], hopMints[len(hopMints)-1][1]
	if inputMint == "" || outputMint == "" {
		return nil
	}

	first, last := events[0], events[len(events)-1]
	output := last.outputAmount
	if last.outputTransferFee < output {
		output -= last.outputTransferFee
	}

	amm := dexInfo.AMM
	swapInfo := utils.EventSwap{
		InputMint:    inputMint,
		InputAmount:  new(big.Int).SetUint64(first.inputAmount),
		OutputMint:   outputMint,
		OutputAmount: new(big.Int).SetUint64(output),
	}
	for i, ev := range events {
		hopIn, hopOut := hopMints[i][0], hopMints[i][1]
		if len(events) == 1 {
			fee := p.Utils.NewEventFee(hopIn, ev.lpFee, "lp", amm)
			swapInfo.Fee = &fee
		} else if ev.lpFee > 0 {
			swapInfo.Fees = append(swapInfo.Fees, p.Utils.NewEventFee(hopIn, ev.lpFee, "lp", amm))
		}
		if ev.protocolFee > 0 {
			swapInfo.Fees = append(swapInfo.Fees, p.Utils.NewEventFee(hopIn, ev.protocolFee, "protocol", amm))
		}
		if i == 0 && ev.inputTransferFee > 0 && hopIn != "" {
			swapInfo.Fees = append(swapInfo.Fees, p.Utils.NewEventFee(hopIn, ev.inputTransferFee, "transferFee", amm))
		}
		if i == len(events)-1 && ev.outputTransferFee > 0 && hopOut != "" {
			swapInfo.Fees = append(swapInfo.Fees, p.Utils.NewEventFee(hopOut, ev.outputTransferFee, "transferFee", amm))
		}
	}
	return p.Utils.NewEventTrade(swapInfo, dexInfo, idx)
}
