package meteora

import (
	"encoding/binary"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
	"github.com/mr-tron/base58"
)

// MeteoraParser parses Meteora swap transactions (DLMM, DAMM v1 and DAMM v2)
type MeteoraParser struct {
	*parsers.BaseParser
	logs     []utils.ProgramLog
	logsRead bool
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

// ProcessTrades parses Meteora swap trades. Only swap instructions are
// trades: liquidity, fee, reward and pool-creation instructions (and the
// self-CPI events they emit, e.g. DAMM v2 EvtCreatePosition on pool creation
// or a DBC migration) also move tokens and must not become trades. Amounts
// and fees come from the programs' swap events (DLMM Swap/Swap2Evt and DAMM
// v2 EvtSwap2/EvtSwap self-CPIs, the DAMM v1 Swap log); without them the
// instruction's transfers are used.
func (p *MeteoraParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		if !isMeteoraProgram(ci.ProgramId) || !p.isSwap(ci.ProgramId, p.Adapter.GetInstructionData(ci.Instruction)) {
			continue
		}
		events := followingEvents(p.Adapter, p.ClassifiedInstructions, ci.ProgramId, ci.OuterIndex, ci.InnerIndex)
		// Transfers made after a self-CPI event are grouped under the event
		transfers := p.GetTransfersForInstruction(ci.ProgramId, ci.OuterIndex, ci.InnerIndex, nil)
		for _, e := range events {
			transfers = append(transfers, p.GetTransfersForInstruction(e.ProgramId, e.OuterIndex, e.InnerIndex, nil)...)
		}
		accounts := p.Adapter.GetInstructionAccounts(ci.Instruction)

		dexInfo := p.DexInfo
		if dexInfo.AMM == "" {
			dexInfo.AMM = constants.GetProgramName(ci.ProgramId)
		}
		idx := ci.GetIdx()
		if len(transfers) > 0 {
			idx = transfers[0].Idx
		}

		var trade *types.TradeInfo
		switch ci.ProgramId {
		case constants.DEX_PROGRAMS.METEORA.ID:
			trade = p.dlmmTradeFromEvent(events, accounts, dexInfo, idx)
		case constants.DEX_PROGRAMS.METEORA_DAMM.ID:
			trade = p.dammV1TradeFromEvent(ci, accounts, dexInfo, idx)
		case constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID:
			trade = p.dammV2TradeFromEvent(events, accounts, dexInfo, idx)
		}

		if trade == nil {
			if len(transfers) < 2 {
				continue
			}
			// For METEORA (DLMM), take only first 2 transfers
			if ci.ProgramId == constants.DEX_PROGRAMS.METEORA.ID {
				transfers = transfers[:2]
			}
			trade = p.Utils.ProcessSwapData(transfers, dexInfo, false)
			if trade == nil {
				continue
			}
		}
		if pool := p.getPoolAddress(ci.Instruction, ci.ProgramId); pool != "" {
			trade.Pool = []string{pool}
		}
		trade = p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
		trades = append(trades, *p.Utils.AttachInstructionTransfers(trade, transfers))
	}

	return trades
}

// isMeteoraProgram checks if programId is a Meteora program
func isMeteoraProgram(programId string) bool {
	return programId == constants.DEX_PROGRAMS.METEORA.ID ||
		programId == constants.DEX_PROGRAMS.METEORA_DAMM.ID ||
		programId == constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID
}

// isSwap reports whether the instruction data is a swap of programId
func (p *MeteoraParser) isSwap(programId string, data []byte) bool {
	switch programId {
	case constants.DEX_PROGRAMS.METEORA.ID:
		_, ok := constants.MatchAnyDiscriminator(data, constants.DISCRIMINATORS.METEORA_DLMM.SWAP)
		return ok
	case constants.DEX_PROGRAMS.METEORA_DAMM.ID:
		return constants.MatchDiscriminator(data, constants.DISCRIMINATORS.METEORA_DAMM.SWAP)
	case constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID:
		return constants.MatchDiscriminator(data, constants.DISCRIMINATORS.METEORA_DAMM_V2.SWAP) ||
			constants.MatchDiscriminator(data, constants.DISCRIMINATORS.METEORA_DAMM_V2.SWAP2)
	}
	return false
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

// dlmmTradeFromEvent builds a DLMM swap from its Swap or Swap2Evt event.
// Accounts (swap, swap2, swap_exact_out(2), swap_with_price_impact(2)):
// lb_pair 0, token_x_mint 6, token_y_mint 7.
//
// Swap: lb_pair, from, start/end_bin_id (i32), amount_in, amount_out,
// swap_for_y, fee, protocol_fee, fee_bps (u128), host_fee. Fees are charged
// on the input token; fee includes the protocol fee (fee - protocol_fee
// equals the mm_fee of the Swap2Evt the program emits with it).
//
// Swap2Evt: lb_pair, from, start/end_bin_id, swap_for_y, fee_bps (u128),
// amount_in, amount_left, amount_out, mm_fee, protocol_fee,
// limit_order_fee, host_fee, fees_on_input, fees_on_token_x.
func (p *MeteoraParser) dlmmTradeFromEvent(instructionEvents []types.ClassifiedInstruction, accounts []string, dexInfo types.DexInfo, idx string) *types.TradeInfo {
	if len(accounts) < 8 {
		return nil
	}
	events := constants.DISCRIMINATORS.METEORA_DLMM.EVENTS
	data := findEvent(p.Adapter, instructionEvents, events["swap2Evt"])
	if data == nil {
		data = findEvent(p.Adapter, instructionEvents, events["swap"])
	}
	if len(data) < 48 {
		return nil
	}
	u64 := func(offset int) uint64 { return binary.LittleEndian.Uint64(data[offset : offset+8]) }
	mintX, mintY := accounts[6], accounts[7]
	amm := dexInfo.AMM
	const body = 16 + 32 + 32 + 4 + 4 // prefix, lb_pair, from, start_bin_id, end_bin_id
	if base58.Encode(data[16:48]) != accounts[0] {
		return nil
	}

	swap := utils.EventSwap{}
	if constants.MatchDiscriminator(data, events["swap"]) {
		if len(data) < body+8+8+1+8+8+16+8 {
			return nil
		}
		swapForY := data[body+16] != 0
		swap.InputMint, swap.OutputMint = mintY, mintX
		if swapForY {
			swap.InputMint, swap.OutputMint = mintX, mintY
		}
		swap.InputAmount = new(big.Int).SetUint64(u64(body))
		swap.OutputAmount = new(big.Int).SetUint64(u64(body + 8))
		fee, protocolFee, hostFee := u64(body+17), u64(body+25), u64(body+49)
		// fee includes the protocol fee; Fee is the LP share
		if protocolFee <= fee {
			fee -= protocolFee
		}
		f := p.Utils.NewEventFee(swap.InputMint, fee, "lp", amm)
		swap.Fee = &f
		if protocolFee > 0 {
			swap.Fees = append(swap.Fees, p.Utils.NewEventFee(swap.InputMint, protocolFee, "protocol", amm))
		}
		if hostFee > 0 {
			swap.Fees = append(swap.Fees, p.Utils.NewEventFee(swap.InputMint, hostFee, "host", amm))
		}
		return p.Utils.NewEventTrade(swap, dexInfo, idx)
	}

	// Swap2Evt
	if len(data) < body+1+16+7*8+2 {
		return nil
	}
	swapForY := data[body] != 0
	swap.InputMint, swap.OutputMint = mintY, mintX
	if swapForY {
		swap.InputMint, swap.OutputMint = mintX, mintY
	}
	amounts := body + 1 + 16
	swap.InputAmount = new(big.Int).SetUint64(u64(amounts))       // amount_in: total the user transfers
	swap.OutputAmount = new(big.Int).SetUint64(u64(amounts + 16)) // amount_out: total transferred to the user
	mmFee, protocolFee, limitOrderFee, hostFee := u64(amounts+24), u64(amounts+32), u64(amounts+40), u64(amounts+48)
	feesOnTokenX := data[amounts+57] != 0
	feeMint := mintY
	if feesOnTokenX {
		feeMint = mintX
	}
	f := p.Utils.NewEventFee(feeMint, mmFee, "lp", amm)
	swap.Fee = &f
	for _, extra := range []struct {
		amount  uint64
		feeType string
	}{{protocolFee, "protocol"}, {limitOrderFee, "limitOrder"}, {hostFee, "host"}} {
		if extra.amount > 0 {
			swap.Fees = append(swap.Fees, p.Utils.NewEventFee(feeMint, extra.amount, extra.feeType, amm))
		}
	}
	return p.Utils.NewEventTrade(swap, dexInfo, idx)
}

// dammV1TradeFromEvent builds a DAMM v1 swap from its Swap log (in_amount,
// out_amount, trade_fee, protocol_fee, host_fee; fees in the input token,
// trade_fee is the LP share: the protocol and host fees are taken out of it).
// Accounts: pool 0, user_source_token 1, user_destination_token 2.
func (p *MeteoraParser) dammV1TradeFromEvent(ci types.ClassifiedInstruction, accounts []string, dexInfo types.DexInfo, idx string) *types.TradeInfo {
	if len(accounts) < 3 {
		return nil
	}
	if !p.logsRead {
		p.logsRead = true
		p.logs = p.Utils.GetProgramDataLogs()
	}
	var data []byte
	for _, l := range utils.FindProgramLogs(p.logs, ci.ProgramId, ci.OuterIndex, ci.InnerIndex) {
		if len(l.Data) >= 8+5*8 && constants.MatchDiscriminator(l.Data, constants.DISCRIMINATORS.METEORA_DAMM.SWAP_EVENT) {
			data = l.Data
		}
	}
	if data == nil {
		return nil
	}
	u64 := func(offset int) uint64 { return binary.LittleEndian.Uint64(data[offset : offset+8]) }
	inputMint := p.Adapter.GetSplTokenMint(accounts[1])
	outputMint := p.Adapter.GetSplTokenMint(accounts[2])
	amm := dexInfo.AMM
	swap := utils.EventSwap{
		InputMint:    inputMint,
		InputAmount:  new(big.Int).SetUint64(u64(8)),
		OutputMint:   outputMint,
		OutputAmount: new(big.Int).SetUint64(u64(16)),
	}
	f := p.Utils.NewEventFee(inputMint, u64(24), "lp", amm)
	swap.Fee = &f
	if protocolFee := u64(32); protocolFee > 0 {
		swap.Fees = append(swap.Fees, p.Utils.NewEventFee(inputMint, protocolFee, "protocol", amm))
	}
	if hostFee := u64(40); hostFee > 0 {
		swap.Fees = append(swap.Fees, p.Utils.NewEventFee(inputMint, hostFee, "host", amm))
	}
	return p.Utils.NewEventTrade(swap, dexInfo, idx)
}

// dammV2TradeFromEvent builds a DAMM v2 swap from its EvtSwap2 (or legacy
// EvtSwap) event. Accounts (swap, swap2): pool 1, output_token_account 3,
// token_a_mint 6, token_b_mint 7, referral_token_account 11.
// trade_direction 0 is A to B.
//
// EvtSwap2: pool, trade_direction, collect_fee_mode, has_referral,
// params {amount_0, amount_1, swap_mode}, swap_result
// {included_fee_input_amount, excluded_fee_input_amount, amount_left,
// output_amount, next_sqrt_price (u128), claiming_fee, protocol_fee,
// compounding_fee, referral_fee}, included_transfer_fee_amount_in,
// included_transfer_fee_amount_out, excluded_transfer_fee_amount_out, ...
// The user sends included_transfer_fee_amount_in and receives
// excluded_transfer_fee_amount_out. The fees are on the input when
// included_fee_input_amount exceeds excluded_fee_input_amount, else on the
// output.
//
// Legacy EvtSwap: pool, trade_direction, has_referral, amount_in,
// minimum_amount_out, output_amount, next_sqrt_price (u128), lp_fee,
// protocol_fee, partner_fee, referral_fee, actual_amount_in, ... The user
// sends amount_in and receives output_amount.
func (p *MeteoraParser) dammV2TradeFromEvent(instructionEvents []types.ClassifiedInstruction, accounts []string, dexInfo types.DexInfo, idx string) *types.TradeInfo {
	if len(accounts) < 12 {
		return nil
	}
	v2 := constants.DISCRIMINATORS.METEORA_DAMM_V2
	// some program versions emit both; EvtSwap2 is the current one
	data := findEvent(p.Adapter, instructionEvents, v2.EVT_SWAP2)
	if data == nil {
		data = findEvent(p.Adapter, instructionEvents, v2.EVT_SWAP)
	}
	if data == nil || len(data) < 16+32+2 || base58.Encode(data[16:48]) != accounts[1] {
		return nil
	}
	u64 := func(offset int) uint64 { return binary.LittleEndian.Uint64(data[offset : offset+8]) }
	mintA, mintB := accounts[6], accounts[7]
	inputMint, outputMint := mintA, mintB
	if data[48] != 0 {
		inputMint, outputMint = mintB, mintA
	}
	amm := dexInfo.AMM
	swap := utils.EventSwap{InputMint: inputMint, OutputMint: outputMint}

	type fee struct {
		amount  uint64
		feeType string
	}
	var fees []fee
	feeMint := outputMint
	if constants.MatchDiscriminator(data, v2.EVT_SWAP2) {
		const result = 16 + 32 + 3 + 17 // swap_result offset
		if len(data) < result+4*8+16+4*8+3*8 {
			return nil
		}
		includedFeeInput, excludedFeeInput := u64(result), u64(result+8)
		fees = []fee{{u64(result + 48), "lp"}, {u64(result + 56), "protocol"}, {u64(result + 64), "compounding"}, {u64(result + 72), "referral"}}
		if includedFeeInput > excludedFeeInput {
			feeMint = inputMint
		}
		swap.InputAmount = new(big.Int).SetUint64(u64(result + 80))  // included_transfer_fee_amount_in
		swap.OutputAmount = new(big.Int).SetUint64(u64(result + 96)) // excluded_transfer_fee_amount_out
	} else {
		const body = 16 + 32 + 2
		if len(data) < body+3*8+16+5*8 {
			return nil
		}
		amountIn, actualAmountIn := u64(body), u64(body+72)
		fees = []fee{{u64(body + 40), "lp"}, {u64(body + 48), "protocol"}, {u64(body + 56), "partner"}, {u64(body + 64), "referral"}}
		if actualAmountIn < amountIn {
			feeMint = inputMint
		}
		swap.InputAmount = new(big.Int).SetUint64(amountIn)
		swap.OutputAmount = new(big.Int).SetUint64(u64(body + 16))
	}
	// A referral fee paid to the user's own output account (self-referral)
	// is received by the user as well
	if referral := fees[len(fees)-1].amount; referral > 0 && feeMint == outputMint && accounts[11] == accounts[3] {
		swap.OutputAmount.Add(swap.OutputAmount, new(big.Int).SetUint64(referral))
	}
	for j, f := range fees {
		if j == 0 {
			info := p.Utils.NewEventFee(feeMint, f.amount, f.feeType, amm)
			swap.Fee = &info
			continue
		}
		if f.amount > 0 {
			swap.Fees = append(swap.Fees, p.Utils.NewEventFee(feeMint, f.amount, f.feeType, amm))
		}
	}
	return p.Utils.NewEventTrade(swap, dexInfo, idx)
}
