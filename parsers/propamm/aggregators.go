package propamm

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"strconv"

	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Aggregator route parsers.
//
// Titan and OKX DEX Router V2 execute a route by CPI into venue programs.
// Each hop is a trade of its venue (the venue parsers report it from the
// hop's own two transfers); the aggregator instruction itself must not be
// read as a group of transfers, since its CPI group holds the hops'
// transfers as well as the user <-> aggregator transfers (summing them was
// the Titan double count: 1000 USDT -> "1999 USDC").
//
// TitanParser and OKXV2Parser report the route instead: one trade per
// aggregator swap instruction, with the totals the aggregator itself emits
// (what the user paid and received, after the aggregator's fees). They are
// meant for the aggregate trade of a transaction, and DexParser registers
// them as route parsers (DexParser.RegisterRouteParser), not as trade
// parsers: next to the hop parsers in the same trade list they would count
// every route twice. The aggregate then takes the route trade in place of
// the hops it covers; without it (no event logged), the aggregate is
// utils.GetFinalSwap over the hops: the input of the first hop's input mint
// and the output of the last hop's output mint, before fees the aggregator
// keeps (Titan fee_c, OKX output-side commission).

// aggregatorParser holds what the route parsers share. Route trades name
// the aggregator as Route, so the DexInfo the constructors take (the
// TradeParserFactory signature) is not used.
type aggregatorParser struct {
	adapter                *adapter.TransactionAdapter
	transferActions        map[string][]types.TransferData
	classifiedInstructions []types.ClassifiedInstruction
	txUtils                *utils.TransactionUtils
}

func newAggregatorParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) aggregatorParser {
	return aggregatorParser{
		adapter:                adapter,
		transferActions:        transferActions,
		classifiedInstructions: classifiedInstructions,
		txUtils:                utils.NewTransactionUtils(adapter),
	}
}

// hopVenues returns, in execution order, the names of the known DEX programs
// that ci invokes directly (its hops)
func (p *aggregatorParser) hopVenues(ci types.ClassifiedInstruction) []string {
	_, instructions := cpiGroup(p.adapter, ci)
	height := p.adapter.GetInstructionStackHeight(ci.OuterIndex, ci.InnerIndex)
	var names []string
	for _, ix := range instructions {
		if h := adapter.InstructionStackHeight(ix); h > 0 && height > 0 && h != height+1 {
			continue // not invoked by ci itself
		}
		programId := p.adapter.GetInstructionProgramId(ix)
		if programId == ci.ProgramId {
			continue
		}
		name := constants.GetDexProgramByID(programId).Name
		if name == "" {
			continue
		}
		seen := false
		for _, n := range names {
			seen = seen || n == name
		}
		if !seen {
			names = append(names, name)
		}
	}
	return names
}

// tokenInfo returns the token details of amount of mint
func (p *aggregatorParser) tokenInfo(mint string, amount *big.Int) types.TokenInfo {
	decimals := p.adapter.GetTokenDecimals(mint)
	return types.TokenInfo{
		Mint:      mint,
		Amount:    types.ConvertToUIAmount(amount, decimals),
		AmountRaw: amount.String(),
		Decimals:  decimals,
	}
}

// feeInfo returns a fee of amount of mint that the aggregator program
// charged
func (p *aggregatorParser) feeInfo(program constants.DexProgram, mint string, amount *big.Int, feeType, recipient string) types.FeeInfo {
	decimals := p.adapter.GetTokenDecimals(mint)
	return types.FeeInfo{
		Mint:      mint,
		Amount:    types.ConvertToUIAmount(amount, decimals),
		AmountRaw: amount.String(),
		Decimals:  decimals,
		Dex:       program.Name,
		Type:      feeType,
		Recipient: recipient,
	}
}

// routeTrade completes a route trade and attaches balance changes
func (p *aggregatorParser) routeTrade(ci types.ClassifiedInstruction, program constants.DexProgram, user string, in, out types.TokenInfo) *types.TradeInfo {
	amms := p.hopVenues(ci)
	amm := program.Name
	if len(amms) > 0 {
		amm = amms[0]
	}
	trade := &types.TradeInfo{
		Type:        utils.GetTradeType(in.Mint, out.Mint),
		InputToken:  in,
		OutputToken: out,
		User:        user,
		ProgramId:   program.ID,
		AMM:         amm,
		AMMs:        amms,
		Route:       program.Name,
		Slot:        p.adapter.Slot(),
		Timestamp:   p.adapter.BlockTime(),
		Signature:   p.adapter.Signature(),
		Idx:         utils.FormatIdx(ci.OuterIndex, ci.InnerIndex),
	}
	return p.txUtils.AttachTokenTransferInfo(trade, p.transferActions)
}

// TitanParser reports Titan SwapRouteV3 routes from the swap event the Titan
// program logs once per route ("Program data", SWAP_EVENT: u64 in_amount,
// out_amount, quoted_out, fee_a, fee_b, fee_c). Input is in_amount of the
// mint of the user's source account (accounts[3], WSOL when it is a
// temporary account funded with SOL), gross of fee_a when fee_a is taken on
// the input side; output is out_amount of the mint of the destination
// account (accounts[4]), what the user received. User is accounts[1]. fee_a
// (the integrator's fee) is reported as Fee, Type "platform", when Titan
// itself (not a hop venue) made a transfer of that amount in the input or
// output mint to another owner; fee_c (kept in Titan's intermediate account,
// not a transfer) is not.
type TitanParser struct {
	aggregatorParser
}

// NewTitanParser creates a Titan route parser (see TitanParser)
func NewTitanParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *TitanParser {
	return &TitanParser{newAggregatorParser(adapter, transferActions, classifiedInstructions)}
}

// ProcessTrades returns one route trade per Titan SwapRouteV3 instruction
// that logged its swap event
func (p *TitanParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo
	var logs []utils.ProgramLog
	logsRead := false

	for _, ci := range p.classifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.TITAN.ID {
			continue
		}
		data := p.adapter.GetInstructionData(ci.Instruction)
		accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
		if !hasTag(data, disc.TITAN.SWAP_ROUTE_V3) || len(data) < 18 || len(accounts) < 5 {
			continue
		}
		if !logsRead {
			logs, logsRead = p.txUtils.GetProgramDataLogs(), true
		}

		var event []byte
		for _, l := range utils.FindProgramLogs(logs, ci.ProgramId, ci.OuterIndex, ci.InnerIndex) {
			if bytes.HasPrefix(l.Data, disc.TITAN.SWAP_EVENT) && len(l.Data) >= 56 {
				event = l.Data[8:]
				break
			}
		}
		if event == nil {
			continue
		}
		inAmount := new(big.Int).SetUint64(binary.LittleEndian.Uint64(event[0:8]))
		outAmount := new(big.Int).SetUint64(binary.LittleEndian.Uint64(event[8:16]))
		feeA := binary.LittleEndian.Uint64(event[24:32])

		transfers := groupTransfers(p.adapter, p.txUtils, ci, false)
		inMint := p.adapter.GetSplTokenMint(accounts[3])
		outMint := p.adapter.GetSplTokenMint(accounts[4])
		for _, t := range transfers {
			if inMint == "" && t.Info.Source == accounts[3] {
				inMint = t.Info.Mint
			}
			if t.Info.Destination == accounts[4] && outMint == "" {
				outMint = t.Info.Mint
			}
		}
		if inMint == "" || outMint == "" {
			continue
		}

		// fee_a is paid by a transfer Titan makes itself (a direct child of the
		// route instruction; hop transfers run one level deeper, inside the
		// venue) to an account outside Titan's own intermediate accounts
		// (owned by accounts[2]; the user's input goes there). When that
		// transfer goes to the user's own account (an integrator whose fee
		// account is the user's), the user kept it: it is part of what the
		// user received
		var fee *types.FeeInfo
		if feeA > 0 {
			amount := strconv.FormatUint(feeA, 10)
			height := p.adapter.GetInstructionStackHeight(ci.OuterIndex, ci.InnerIndex)
			for _, t := range transfers {
				if t.Info.TokenAmount.Amount != amount || (t.Info.Mint != inMint && t.Info.Mint != outMint) {
					continue
				}
				if h := p.adapter.GetInstructionStackHeight(utils.SplitIdx(t.Idx)); height > 0 && h > 0 && h != height+1 {
					continue // a hop's transfer
				}
				destinationOwner := t.Info.DestinationOwner
				if destinationOwner == "" {
					destinationOwner = p.adapter.GetTokenAccountOwner(t.Info.Destination)
				}
				if len(accounts) > 2 && destinationOwner == accounts[2] {
					continue // into Titan's intermediate account: the route's input
				}
				feeAmount := new(big.Int).SetUint64(feeA)
				if t.Info.Destination == accounts[4] || t.Info.DestinationOwner == accounts[1] {
					switch t.Info.Mint {
					case outMint:
						outAmount.Add(outAmount, feeAmount)
					case inMint:
						inAmount.Sub(inAmount, feeAmount)
					}
				} else {
					f := p.feeInfo(constants.DEX_PROGRAMS.TITAN, t.Info.Mint, feeAmount, "platform", t.Info.Destination)
					fee = &f
				}
				break
			}
		}

		trade := p.routeTrade(ci, constants.DEX_PROGRAMS.TITAN, accounts[1], p.tokenInfo(inMint, inAmount), p.tokenInfo(outMint, outAmount))
		if fee != nil {
			trade.Fee = fee
			trade.Fees = append(trade.Fees, *fee)
		}
		trades = append(trades, *trade)
	}

	return trades
}

// OKXV2Parser reports OKX DEX Router V2 swaps from the swap event the router
// emits (emit_cpi) at the end of each swap instruction. Input is
// source_token_change of source_mint (what left the user, including a
// commission charged on the input side), output is destination_token_change
// of destination_mint (what the user received), user is
// source_token_account_owner. The commission is reported as Fee on the side
// commission_direction names (input when set); platform, trim and charge
// fees are not, their side is not documented. The per-hop SwapEvent is not
// used: it shares Raydium CLMM's event discriminator and does not decode for
// every dex.
type OKXV2Parser struct {
	aggregatorParser
}

// NewOKXV2Parser creates an OKX DEX Router V2 route parser (see OKXV2Parser)
func NewOKXV2Parser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *OKXV2Parser {
	return &OKXV2Parser{newAggregatorParser(adapter, transferActions, classifiedInstructions)}
}

// okxEvent is the common part of the OKX V2 swap events
type okxEvent struct {
	sourceMint, destinationMint, user string
	sourceChange, destinationChange   uint64
	commissionOnSource                bool
	commission                        uint64
	commissionInSOL                   bool
}

// decodeOKXEvent decodes an OKX V2 swap event instruction; ok is false for
// any other data
func decodeOKXEvent(data []byte) (e okxEvent, ok bool) {
	d := disc.OKX_DEX_V2
	const common = 16 + 8 + 4*32 + 3*8 // prefix, order_id, 4 pubkeys, 3 amounts
	if len(data) < common {
		return e, false
	}
	u64 := func(off int) uint64 { return binary.LittleEndian.Uint64(data[off : off+8]) }
	e.sourceMint = base58.Encode(data[24:56])
	e.destinationMint = base58.Encode(data[56:88])
	e.user = base58.Encode(data[88:120])
	e.sourceChange = u64(160)
	e.destinationChange = u64(168)

	rest := data[common:]
	switch {
	case bytes.HasPrefix(data, d.SWAP_CPI_EVENT2):
	case bytes.HasPrefix(data, d.SWAP_WITH_FEES_CPI_EVENT2), bytes.HasPrefix(data, d.SWAP_WITH_FEES_CPI_EVENT_ENHANCED2):
		// commission_direction bool, commission_rate u32, commission_amount u64
		if len(rest) >= 13 {
			e.commissionOnSource = rest[0] == 1
			e.commission = binary.LittleEndian.Uint64(rest[5:13])
		}
	case bytes.HasPrefix(data, d.SWAP_TOB_V2_CPI_EVENT2), bytes.HasPrefix(data, d.SWAP_TOC_V2_CPI_EVENT2):
		// commission_direction bool, total_commission_rate u32,
		// parent_commission_rate u32, parent_commission_amount u64,
		// parent_commission_account, child_commission_rate u32,
		// child_commission_amount u64
		if len(rest) >= 1+4+4+8+32+4+8 {
			e.commissionOnSource = rest[0] == 1
			e.commission = binary.LittleEndian.Uint64(rest[9:17]) + binary.LittleEndian.Uint64(rest[53:61])
		}
	case bytes.HasPrefix(data, d.SWAP_WITH_FEE_CPI_EVENT_V3):
		// commission_direction bool, commission_paid_in_sol bool,
		// total_commission_rate u32, total_commission_amount u64
		if len(rest) >= 14 {
			e.commissionOnSource = rest[0] == 1
			e.commissionInSOL = rest[1] == 1
			e.commission = binary.LittleEndian.Uint64(rest[6:14])
		}
	default:
		return okxEvent{}, false
	}
	return e, true
}

// ProcessTrades returns one route trade per OKX V2 instruction that emitted
// a swap event
func (p *OKXV2Parser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	for _, ci := range p.classifiedInstructions {
		if ci.ProgramId != constants.DEX_PROGRAMS.OKX_DEX_V2.ID ||
			constants.IsAnchorEvent(p.adapter.GetInstructionData(ci.Instruction)) {
			continue
		}
		_, instructions := cpiGroup(p.adapter, ci)
		var event *okxEvent
		for _, ix := range instructions {
			if p.adapter.GetInstructionProgramId(ix) != ci.ProgramId {
				continue
			}
			if e, ok := decodeOKXEvent(p.adapter.GetInstructionData(ix)); ok {
				event = &e
			}
		}
		if event == nil {
			continue
		}

		in := p.tokenInfo(event.sourceMint, new(big.Int).SetUint64(event.sourceChange))
		out := p.tokenInfo(event.destinationMint, new(big.Int).SetUint64(event.destinationChange))
		trade := p.routeTrade(ci, constants.DEX_PROGRAMS.OKX_DEX_V2, event.user, in, out)
		if event.commission > 0 {
			mint := event.destinationMint
			if event.commissionInSOL {
				mint = constants.TOKENS.SOL
			} else if event.commissionOnSource {
				mint = event.sourceMint
			}
			fee := p.feeInfo(constants.DEX_PROGRAMS.OKX_DEX_V2, mint, new(big.Int).SetUint64(event.commission), "commission", "")
			trade.Fee = &fee
			trade.Fees = append(trade.Fees, fee)
		}
		trades = append(trades, *trade)
	}

	return trades
}
