package raydium

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

// RaydiumParser parses Raydium swap transactions (AMM v4, CPMM, CLMM and
// the Raydium route program)
type RaydiumParser struct {
	*parsers.BaseParser
	dataLogs []utils.ProgramLog
	rayLogs  []utils.ProgramLog
	logsRead bool
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

// raydiumSwapKind identifies a Raydium swap instruction
type raydiumSwapKind int

const (
	swapNone       raydiumSwapKind = iota
	swapV4                         // AMM v4 swap_base_in/out (tags 9, 11) and their V2 forms (16, 17)
	swapCPMM                       // CPMM swap_base_input / swap_base_output
	swapCLMM                       // CLMM swap / swap_v2
	swapCLMMRouter                 // CLMM swap_router_base_in (multi-hop)
	swapRoute                      // Raydium route program swap
)

// getSwapKind returns the kind of a Raydium swap instruction, or swapNone.
// Only swaps are trades: pool creation, deposits, withdrawals, fee
// collection and limit orders also move tokens and must not become trades.
func getSwapKind(programId string, data []byte) raydiumSwapKind {
	switch programId {
	case constants.DEX_PROGRAMS.RAYDIUM_V4.ID, constants.DEX_PROGRAMS.RAYDIUM_AMM.ID:
		if len(data) == 0 {
			return swapNone
		}
		r := constants.DISCRIMINATORS.RAYDIUM
		tag := data[:1]
		if bytes.Equal(tag, r.SWAP) || bytes.Equal(tag, r.SWAP_EXACT_OUT) ||
			bytes.Equal(tag, r.SWAP_V2) || bytes.Equal(tag, r.SWAP_EXACT_OUT_V2) {
			return swapV4
		}
	case constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID:
		c := constants.DISCRIMINATORS.RAYDIUM_CPMM
		if constants.MatchDiscriminator(data, c.SWAP_BASE_INPUT) || constants.MatchDiscriminator(data, c.SWAP_BASE_OUTPUT) {
			return swapCPMM
		}
	case constants.DEX_PROGRAMS.RAYDIUM_CL.ID:
		s := constants.DISCRIMINATORS.RAYDIUM_CL.SWAP
		if constants.MatchDiscriminator(data, s.SWAP) || constants.MatchDiscriminator(data, s.SWAP_V2) {
			return swapCLMM
		}
		if constants.MatchDiscriminator(data, s.SWAP_ROUTER_BASE_IN) {
			return swapCLMMRouter
		}
	case constants.DEX_PROGRAMS.RAYDIUM_ROUTE.ID:
		// Route swaps seen on mainnet: tag 0 (multi-pool route) and 8
		// (process_route_swap_base_in). Tags 5 and 6 wrap and unwrap SOL.
		if len(data) > 0 && (data[0] == 0 || data[0] == 8) {
			return swapRoute
		}
	}
	return swapNone
}

// ProcessTrades parses Raydium swap trades. Amounts come from the programs'
// own logs where they write them (AMM v4 ray_log, CPMM and CLMM SwapEvent),
// otherwise from the instruction's transfers. Only fees the programs report
// are attached (CPMM trade and creator fees, transfer fees).
func (p *RaydiumParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.ClassifiedInstructions {
		data := p.Adapter.GetInstructionData(ci.Instruction)
		kind := getSwapKind(ci.ProgramId, data)
		if kind == swapNone {
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

		var trade *types.TradeInfo
		var pools []string
		switch kind {
		case swapV4:
			trade = p.v4TradeFromLog(ci, data, accounts, dexInfo, idx)
		case swapCPMM:
			trade = p.cpmmTradeFromEvent(ci, accounts, dexInfo, idx)
		case swapCLMM, swapCLMMRouter:
			trade, pools = p.clmmTradeFromEvents(ci, dexInfo, idx)
		}

		if trade == nil {
			if kind == swapCLMMRouter {
				// all hops: first input and last output
				if len(transfers) >= 2 {
					trade = p.Utils.ProcessSwapData(transfers, dexInfo, false)
				}
			} else if len(transfers) >= 2 {
				// a swap moves two tokens; later transfers of the group belong
				// to the calling program
				trade = p.Utils.ProcessSwapData(transfers[:2], dexInfo, false)
			}
		}
		if trade == nil {
			continue
		}

		if len(pools) > 0 {
			trade.Pool = pools
		} else if pool := p.getPoolAddress(kind, accounts); pool != "" {
			trade.Pool = []string{pool}
		}
		trade = p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
		trades = append(trades, *p.Utils.AttachInstructionTransfers(trade, transfers))
	}

	return trades
}

// getPoolAddress gets pool address from instruction accounts
func (p *RaydiumParser) getPoolAddress(kind raydiumSwapKind, accounts []string) string {
	switch kind {
	case swapV4:
		if len(accounts) > 1 {
			return accounts[1] // amm
		}
	case swapCPMM:
		if len(accounts) > 3 {
			return accounts[3] // pool_state
		}
	case swapCLMM:
		if len(accounts) > 2 {
			return accounts[2] // pool_state
		}
	case swapCLMMRouter:
		// payer, input_token_account, input_token_mint, token_program,
		// token_program_2022, memo_program, then per hop amm_config,
		// pool_state, ...: the first hop's pool
		if len(accounts) > 7 {
			return accounts[7]
		}
	}
	return ""
}

// readLogs reads the transaction logs once
func (p *RaydiumParser) readLogs() {
	if !p.logsRead {
		p.logsRead = true
		p.dataLogs = p.Utils.GetProgramDataLogs()
		p.rayLogs = p.Utils.GetRayLogs()
	}
}

// v4TradeFromLog builds an AMM v4 swap from its ray_log. SwapBaseIn gives
// amount_in and out_amount, SwapBaseOut deduct_in and amount_out; direction
// 1 is pc to coin, 2 coin to pc. Mints are those of the pool vaults:
// swap_base_in/out have pool_coin/pool_pc at 5/6 (4/5 without the target
// orders account, 17 accounts), the V2 forms (8 accounts) at 3/4.
func (p *RaydiumParser) v4TradeFromLog(ci types.ClassifiedInstruction, data []byte, accounts []string, dexInfo types.DexInfo, idx string) *types.TradeInfo {
	p.readLogs()
	var in, out *big.Int
	var direction uint64
	for _, l := range utils.FindProgramLogs(p.rayLogs, ci.ProgramId, ci.OuterIndex, ci.InnerIndex) {
		switch log := DecodeRaydiumLog(l.Data).(type) {
		case *SwapBaseInLog:
			in, out, direction = log.AmountIn, log.OutAmount, log.Direction.Uint64()
		case *SwapBaseOutLog:
			in, out, direction = log.DeductIn, log.AmountOut, log.Direction.Uint64()
		}
	}
	if in == nil {
		return nil
	}

	coinVault, pcVault := 5, 6
	switch {
	case len(data) > 0 && (data[0] == constants.DISCRIMINATORS.RAYDIUM.SWAP_V2[0] || data[0] == constants.DISCRIMINATORS.RAYDIUM.SWAP_EXACT_OUT_V2[0]):
		coinVault, pcVault = 3, 4
	case len(accounts) == 17:
		coinVault, pcVault = 4, 5
	}
	if len(accounts) <= pcVault {
		return nil
	}
	coinMint := p.Adapter.GetSplTokenMint(accounts[coinVault])
	pcMint := p.Adapter.GetSplTokenMint(accounts[pcVault])

	swap := utils.EventSwap{InputAmount: in, OutputAmount: out}
	switch direction {
	case SwapDirectionPCToCoin:
		swap.InputMint, swap.OutputMint = pcMint, coinMint
	case SwapDirectionCoinToPC:
		swap.InputMint, swap.OutputMint = coinMint, pcMint
	default:
		return nil
	}
	return p.Utils.NewEventTrade(swap, dexInfo, idx)
}

// cpmmSwapEvent is the CPMM SwapEvent log: pool_id, input_vault_before,
// output_vault_before, input_amount, output_amount, input_transfer_fee,
// output_transfer_fee, base_input; newer program versions append
// input_mint, output_mint, trade_fee, creator_fee, creator_fee_on_input.
type cpmmSwapEvent struct {
	pool                                string
	inputAmount, outputAmount           uint64
	inputTransferFee, outputTransferFee uint64
	hasFees                             bool
	inputMint, outputMint               string
	tradeFee, creatorFee                uint64
	creatorFeeOnInput                   bool
}

func decodeCPMMSwapEvent(data []byte) *cpmmSwapEvent {
	const legacySize = 8 + 32 + 6*8 + 1
	if len(data) < legacySize || !bytes.Equal(data[:8], constants.DISCRIMINATORS.RAYDIUM_CPMM.SWAP_EVENT) {
		return nil
	}
	u64 := func(offset int) uint64 { return binary.LittleEndian.Uint64(data[offset : offset+8]) }
	ev := &cpmmSwapEvent{
		pool:              base58.Encode(data[8:40]),
		inputAmount:       u64(56),
		outputAmount:      u64(64),
		inputTransferFee:  u64(72),
		outputTransferFee: u64(80),
	}
	if len(data) >= legacySize+32+32+8+8+1 {
		ev.hasFees = true
		ev.inputMint = base58.Encode(data[89:121])
		ev.outputMint = base58.Encode(data[121:153])
		ev.tradeFee = u64(153)
		ev.creatorFee = u64(161)
		ev.creatorFeeOnInput = data[169] != 0
	}
	return ev
}

// cpmmTradeFromEvent builds a CPMM swap from its SwapEvent. The user sends
// input_amount plus the input transfer fee and receives output_amount minus
// the output transfer fee. Fee is the trade fee (in the input mint), the
// creator fee is in Fees. Accounts: pool_state 3, input_token_mint 10,
// output_token_mint 11.
func (p *RaydiumParser) cpmmTradeFromEvent(ci types.ClassifiedInstruction, accounts []string, dexInfo types.DexInfo, idx string) *types.TradeInfo {
	if len(accounts) < 12 {
		return nil
	}
	p.readLogs()
	var ev *cpmmSwapEvent
	for _, l := range utils.FindProgramLogs(p.dataLogs, ci.ProgramId, ci.OuterIndex, ci.InnerIndex) {
		if e := decodeCPMMSwapEvent(l.Data); e != nil && e.pool == accounts[3] {
			ev = e
		}
	}
	if ev == nil {
		return nil
	}
	inputMint, outputMint := accounts[10], accounts[11]
	if ev.hasFees && (ev.inputMint != inputMint || ev.outputMint != outputMint) {
		return nil
	}
	output := ev.outputAmount
	if ev.outputTransferFee < output {
		output -= ev.outputTransferFee
	}
	amm := dexInfo.AMM
	swap := utils.EventSwap{
		InputMint:    inputMint,
		InputAmount:  new(big.Int).Add(new(big.Int).SetUint64(ev.inputAmount), new(big.Int).SetUint64(ev.inputTransferFee)),
		OutputMint:   outputMint,
		OutputAmount: new(big.Int).SetUint64(output),
	}
	if ev.hasFees {
		fee := p.Utils.NewEventFee(inputMint, ev.tradeFee, "trade", amm)
		swap.Fee = &fee
		if ev.creatorFee > 0 {
			creatorMint := outputMint
			if ev.creatorFeeOnInput {
				creatorMint = inputMint
			}
			swap.Fees = append(swap.Fees, p.Utils.NewEventFee(creatorMint, ev.creatorFee, "creator", amm))
		}
	}
	if ev.inputTransferFee > 0 {
		swap.Fees = append(swap.Fees, p.Utils.NewEventFee(inputMint, ev.inputTransferFee, "transferFee", amm))
	}
	if ev.outputTransferFee > 0 {
		swap.Fees = append(swap.Fees, p.Utils.NewEventFee(outputMint, ev.outputTransferFee, "transferFee", amm))
	}
	return p.Utils.NewEventTrade(swap, dexInfo, idx)
}

// clmmSwapEvent is the CLMM SwapEvent log: pool_state, sender,
// token_account_0, token_account_1, amount_0, transfer_fee_0, amount_1,
// transfer_fee_1, zero_for_one, sqrt_price_x64, liquidity, tick
type clmmSwapEvent struct {
	pool                         string
	tokenAccount0, tokenAccount1 string
	amount0, transferFee0        uint64
	amount1, transferFee1        uint64
	zeroForOne                   bool
}

func decodeCLMMSwapEvent(data []byte) *clmmSwapEvent {
	const size = 8 + 4*32 + 4*8 + 1
	if len(data) < size || !bytes.Equal(data[:8], constants.DISCRIMINATORS.RAYDIUM_CL.EVENTS.SWAP) {
		return nil
	}
	u64 := func(offset int) uint64 { return binary.LittleEndian.Uint64(data[offset : offset+8]) }
	return &clmmSwapEvent{
		pool:          base58.Encode(data[8:40]),
		tokenAccount0: base58.Encode(data[72:104]),
		tokenAccount1: base58.Encode(data[104:136]),
		amount0:       u64(136),
		transferFee0:  u64(144),
		amount1:       u64(152),
		transferFee1:  u64(160),
		zeroForOne:    data[168] != 0,
	}
}

// clmmTradeFromEvents builds a CLMM swap from its SwapEvents, one per hop
// (swap_router_base_in swaps through several pools). The input is the first
// hop's input and the output the last hop's output; the mints are those of
// the user token accounts the events name. The CLMM events report no
// trading fee.
func (p *RaydiumParser) clmmTradeFromEvents(ci types.ClassifiedInstruction, dexInfo types.DexInfo, idx string) (*types.TradeInfo, []string) {
	p.readLogs()
	var events []*clmmSwapEvent
	for _, l := range utils.FindProgramLogs(p.dataLogs, ci.ProgramId, ci.OuterIndex, ci.InnerIndex) {
		if e := decodeCLMMSwapEvent(l.Data); e != nil {
			events = append(events, e)
		}
	}
	if len(events) == 0 {
		return nil, nil
	}
	type leg struct {
		mint      string
		amount    uint64
		transfFee uint64
	}
	legs := func(e *clmmSwapEvent) (in, out leg) {
		in = leg{p.Adapter.GetSplTokenMint(e.tokenAccount0), e.amount0, e.transferFee0}
		out = leg{p.Adapter.GetSplTokenMint(e.tokenAccount1), e.amount1, e.transferFee1}
		if !e.zeroForOne {
			in, out = out, in
		}
		return in, out
	}
	in, _ := legs(events[0])
	_, out := legs(events[len(events)-1])
	if in.mint == "" || out.mint == "" {
		return nil, nil
	}
	pools := make([]string, 0, len(events))
	for _, e := range events {
		pools = append(pools, e.pool)
	}
	// amount_0/amount_1 are the pool-side amounts: the user sends the input
	// plus its transfer fee and receives the output minus its transfer fee
	output := out.amount
	if out.transfFee < output {
		output -= out.transfFee
	}
	amm := dexInfo.AMM
	swap := utils.EventSwap{
		InputMint:    in.mint,
		InputAmount:  new(big.Int).Add(new(big.Int).SetUint64(in.amount), new(big.Int).SetUint64(in.transfFee)),
		OutputMint:   out.mint,
		OutputAmount: new(big.Int).SetUint64(output),
	}
	if in.transfFee > 0 {
		swap.Fees = append(swap.Fees, p.Utils.NewEventFee(in.mint, in.transfFee, "transferFee", amm))
	}
	if out.transfFee > 0 {
		swap.Fees = append(swap.Fees, p.Utils.NewEventFee(out.mint, out.transfFee, "transferFee", amm))
	}
	return p.Utils.NewEventTrade(swap, dexInfo, idx), pools
}
