package utils

import (
	"encoding/base64"
	"math/big"
	"strconv"
	"strings"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// ProgramLog is a payload a program wrote to the transaction log, attributed
// to the instruction that was executing when it was written.
type ProgramLog struct {
	ProgramId  string // program that wrote the log line
	OuterIndex int    // outer instruction index
	InnerIndex int    // inner instruction index, -1 for the outer instruction itself
	Depth      int    // invoke depth: 1 for an outer instruction, 2+ for CPIs
	Data       []byte // decoded payload
}

// Log line prefixes
const (
	programDataPrefix = "Program data: "
	rayLogPrefix      = "Program log: ray_log: "
	logTruncated      = "Log truncated"
)

// ParseProgramDataLogs decodes the "Program data: <base64>" lines (Anchor
// emit! events and similar) of a transaction. See ParseProgramLogs.
func ParseProgramDataLogs(logs []string, outerProgramIds []string) []ProgramLog {
	return ParseProgramLogs(logs, outerProgramIds, programDataPrefix)
}

// ParseRayLogs decodes the "Program log: ray_log: <base64>" lines written by
// the Raydium AMM v4 program. See ParseProgramLogs.
func ParseRayLogs(logs []string, outerProgramIds []string) []ProgramLog {
	return ParseProgramLogs(logs, outerProgramIds, rayLogPrefix)
}

// ParseProgramLogs decodes the log lines that start with prefix and carry a
// base64 payload, and attributes each one to the instruction that wrote it.
//
// Instruction positions are rebuilt from the runtime's own lines only:
// "Program <id> invoke [<depth>]" opens a frame (depth 1 starts the next
// outer instruction, deeper invokes are the next inner instruction of the
// current outer instruction, in the order of meta.innerInstructions), and
// "Program <id> success" / "Program <id> failed: ..." return to the caller,
// so a payload written after a CPI returned belongs to the caller. <id> must
// be a base58 token: what programs write themselves ("Program log: ...",
// "Program data: ...", "Program return: ...", "Program consumption: ...")
// never opens or closes a frame, whatever its text, and neither do
// "consumed ... compute units" lines. outerProgramIds, when given, are the
// program ids of the outer instructions; they keep the outer index aligned
// when an outer instruction writes no invoke line. Lines that cannot be
// decoded are skipped, and decoding stops at "Log truncated": the payloads
// after it are lost.
func ParseProgramLogs(logs []string, outerProgramIds []string, prefix string) []ProgramLog {
	return parseProgramLogs(logs, outerProgramIds, nil, prefix)
}

// parseProgramLogs is ParseProgramLogs. When innerProgramIds (the program ids
// of each outer instruction's inner instructions, by outer index) is given,
// every CPI invoke line must name the program of the inner instruction it
// stands for; decoding stops at the first one that does not (logs that do
// not belong to these instructions), keeping what was attributed before.
func parseProgramLogs(logs []string, outerProgramIds []string, innerProgramIds map[int][]string, prefix string) []ProgramLog {
	type frame struct {
		programId  string
		innerIndex int
	}
	var result []ProgramLog
	var stack []frame
	outer, inner := -1, -1

	for _, line := range logs {
		if strings.HasPrefix(line, logTruncated) {
			break
		}
		if strings.HasPrefix(line, prefix) {
			if len(stack) == 0 || outer < 0 {
				continue
			}
			data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(line[len(prefix):]))
			if err != nil {
				continue
			}
			top := stack[len(stack)-1]
			result = append(result, ProgramLog{
				ProgramId:  top.programId,
				OuterIndex: outer,
				InnerIndex: top.innerIndex,
				Depth:      len(stack),
				Data:       data,
			})
			continue
		}
		if programId, depth, ok := parseInvokeLine(line); ok {
			if depth == 1 {
				outer = nextOuterIndex(outer, programId, outerProgramIds)
				inner = -1
				stack = append(stack[:0], frame{programId: programId, innerIndex: -1})
				continue
			}
			if outer < 0 {
				continue // no enclosing outer instruction
			}
			inner++
			if innerProgramIds != nil {
				if ids := innerProgramIds[outer]; inner >= len(ids) || ids[inner] != programId {
					return result
				}
			}
			if depth-1 < len(stack) {
				stack = stack[:depth-1]
			}
			stack = append(stack, frame{programId: programId, innerIndex: inner})
			continue
		}
		if programId, ok := parseReturnLine(line); ok {
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].programId == programId {
					stack = stack[:i]
					break
				}
			}
		}
	}
	return result
}

// nextOuterIndex returns the index of the outer instruction that programId
// starts after outer. Without outerProgramIds it is outer+1; with them it is
// the next outer instruction of programId, so that instructions that write
// no invoke line are skipped.
func nextOuterIndex(outer int, programId string, outerProgramIds []string) int {
	if len(outerProgramIds) == 0 {
		return outer + 1
	}
	for i := outer + 1; i < len(outerProgramIds); i++ {
		if outerProgramIds[i] == programId {
			return i
		}
	}
	return outer + 1
}

// parseInvokeLine parses "Program <id> invoke [<depth>]"
func parseInvokeLine(line string) (string, int, bool) {
	rest, ok := strings.CutPrefix(line, "Program ")
	if !ok {
		return "", 0, false
	}
	programId, tail, ok := strings.Cut(rest, " invoke [")
	if !ok || !isBase58Token(programId) {
		return "", 0, false
	}
	depthStr, ok := strings.CutSuffix(tail, "]")
	if !ok {
		return "", 0, false
	}
	depth, err := strconv.Atoi(depthStr)
	if err != nil || depth < 1 {
		return "", 0, false
	}
	return programId, depth, true
}

// parseReturnLine parses "Program <id> success" and "Program <id> failed: ..."
func parseReturnLine(line string) (string, bool) {
	rest, ok := strings.CutPrefix(line, "Program ")
	if !ok {
		return "", false
	}
	programId, tail, ok := strings.Cut(rest, " ")
	if !ok || !isBase58Token(programId) {
		return "", false
	}
	if tail == "success" || strings.HasPrefix(tail, "failed") {
		return programId, true
	}
	return "", false
}

// isBase58Token reports whether s is a non-empty string of base58 characters,
// the form of a program id. The "log:", "data:", "return:" and
// "consumption:" words of program-written lines are not.
func isBase58Token(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '1' && c <= '9', c >= 'A' && c <= 'H', c >= 'J' && c <= 'N', c >= 'P' && c <= 'Z',
			c >= 'a' && c <= 'k', c >= 'm' && c <= 'z':
		default:
			return false
		}
	}
	return true
}

// FindProgramLogs returns the logs written by programId while it executed the
// instruction at (outerIndex, innerIndex), in log order.
func FindProgramLogs(logs []ProgramLog, programId string, outerIndex, innerIndex int) []ProgramLog {
	var result []ProgramLog
	for _, l := range logs {
		if l.OuterIndex == outerIndex && l.InnerIndex == innerIndex && l.ProgramId == programId {
			result = append(result, l)
		}
	}
	return result
}

// outerProgramIds returns the program id of each outer instruction
func outerProgramIds(adapt *adapter.TransactionAdapter) []string {
	instructions := adapt.Instructions()
	ids := make([]string, len(instructions))
	for i, ix := range instructions {
		ids[i] = adapt.GetInstructionProgramId(ix)
	}
	return ids
}

// innerProgramIds returns the program ids of the inner instructions of each
// outer instruction, or nil when the transaction records none (no CPI, or
// old transactions without meta.innerInstructions)
func innerProgramIds(adapt *adapter.TransactionAdapter) map[int][]string {
	sets := adapt.InnerInstructions()
	if len(sets) == 0 {
		return nil
	}
	ids := make(map[int][]string, len(sets))
	for _, set := range sets {
		list := make([]string, len(set.Instructions))
		for i, ix := range set.Instructions {
			list[i] = adapt.GetInstructionProgramId(ix)
		}
		ids[set.Index] = list
	}
	return ids
}

// GetProgramDataLogs returns the "Program data:" payloads of the transaction
// logs with the instruction that wrote each one (see ParseProgramLogs). Each
// CPI invoke line is checked against the recorded inner instructions. It
// returns nil when the transaction carries no logs.
func (tu *TransactionUtils) GetProgramDataLogs() []ProgramLog {
	return tu.programLogs(programDataPrefix)
}

// GetRayLogs returns the Raydium AMM v4 "ray_log:" payloads of the
// transaction logs with the instruction that wrote each one.
func (tu *TransactionUtils) GetRayLogs() []ProgramLog {
	return tu.programLogs(rayLogPrefix)
}

// programLogs decodes the transaction's log lines that start with prefix
func (tu *TransactionUtils) programLogs(prefix string) []ProgramLog {
	logs := tu.adapter.LogMessages()
	if len(logs) == 0 {
		return nil
	}
	return parseProgramLogs(logs, outerProgramIds(tu.adapter), innerProgramIds(tu.adapter), prefix)
}

// EventSwap is a swap as a program reports it in its own event: the amounts
// the user actually sent and received (transfer fees of Token-2022 mints
// included in the input and excluded from the output) and the fees the
// program charged.
type EventSwap struct {
	InputMint    string
	InputAmount  *big.Int
	OutputMint   string
	OutputAmount *big.Int
	Fee          *types.FeeInfo  // main trading fee, if reported
	Fees         []types.FeeInfo // further fees (protocol, host, referral, transfer fees)
}

// NewEventTrade builds the trade of a swap reported by a program event.
// Decimals come from the transaction's token balances.
func (tu *TransactionUtils) NewEventTrade(swap EventSwap, dexInfo types.DexInfo, idx string) *types.TradeInfo {
	if swap.InputMint == "" || swap.OutputMint == "" || swap.InputAmount == nil || swap.OutputAmount == nil {
		return nil
	}
	inDecimals := tu.adapter.GetTokenDecimals(swap.InputMint)
	outDecimals := tu.adapter.GetTokenDecimals(swap.OutputMint)
	return &types.TradeInfo{
		Type: GetTradeType(swap.InputMint, swap.OutputMint),
		InputToken: types.TokenInfo{
			Mint:      swap.InputMint,
			Amount:    types.ConvertToUIAmount(swap.InputAmount, inDecimals),
			AmountRaw: swap.InputAmount.String(),
			Decimals:  inDecimals,
		},
		OutputToken: types.TokenInfo{
			Mint:      swap.OutputMint,
			Amount:    types.ConvertToUIAmount(swap.OutputAmount, outDecimals),
			AmountRaw: swap.OutputAmount.String(),
			Decimals:  outDecimals,
		},
		Fee:       swap.Fee,
		Fees:      swap.Fees,
		User:      tu.getSwapSigner(),
		ProgramId: dexInfo.ProgramId,
		AMM:       dexInfo.AMM,
		Route:     dexInfo.Route,
		Slot:      tu.adapter.Slot(),
		Timestamp: tu.adapter.BlockTime(),
		Signature: tu.adapter.Signature(),
		Idx:       idx,
	}
}

// AttachInstructionTransfers sets the token accounts, authority and
// balances of each trade leg from the swap instruction's own transfers.
// Call it after AttachTokenTransferInfo: that one takes the first transfer
// of the whole transaction with the leg's mint and amount, which in a route
// is the previous hop's output (the same amount) or, for a Token-2022
// output with a transfer fee, the next hop's input (the net amount), and
// leaves a leg bare when no transfer carries its amount.
func (tu *TransactionUtils) AttachInstructionTransfers(trade *types.TradeInfo, transfers []types.TransferData) *types.TradeInfo {
	if trade == nil {
		return nil
	}
	// The output transfer carries the amount before the Token-2022
	// transfer fee is withheld from the user
	output := parseAmount(trade.OutputToken.AmountRaw)
	grossOutput := new(big.Int).Set(output)
	for _, f := range trade.Fees {
		if f.Type == "transferFee" && f.Mint == trade.OutputToken.Mint {
			grossOutput.Add(grossOutput, parseAmount(f.AmountRaw))
		}
	}
	setLegTransfer(&trade.InputToken, legTransfer(transfers, trade.InputToken.Mint, parseAmount(trade.InputToken.AmountRaw)))
	setLegTransfer(&trade.OutputToken, legTransfer(transfers, trade.OutputToken.Mint, output, grossOutput))
	return trade
}

// legTransfer returns the transfer that carried a trade leg: the first
// transfer of mint whose amount is one of amounts, else the largest
// transfer of mint. The amount a program reports can be split over several
// transfers, e.g. a DAMM v1 input over the host fee and the vault deposit,
// or a DAMM v2 output over the swap output and a self-referral fee; the
// fee part is the smaller one. Native SOL transfers of the System program
// (rent, tips) are not swap legs.
func legTransfer(transfers []types.TransferData, mint string, amounts ...*big.Int) *types.TransferData {
	var largest *types.TransferData
	var largestAmount *big.Int
	for i := range transfers {
		t := &transfers[i]
		if t.Info.Mint != mint || t.ProgramId == constants.SYSTEM_PROGRAM_ID {
			continue
		}
		for _, amount := range amounts {
			if t.Info.TokenAmount.Amount == amount.String() {
				return t
			}
		}
		if amount := parseAmount(t.Info.TokenAmount.Amount); largest == nil || amount.Cmp(largestAmount) > 0 {
			largest, largestAmount = t, amount
		}
	}
	return largest
}

// setLegTransfer copies the token accounts, authority and balances of the
// transfer that carried a trade leg to the leg
func setLegTransfer(token *types.TokenInfo, transfer *types.TransferData) {
	if transfer == nil {
		return
	}
	token.Authority = transfer.Info.Authority
	token.Source = transfer.Info.Source
	token.Destination = transfer.Info.Destination
	token.DestinationOwner = transfer.Info.DestinationOwner
	token.DestinationBalance = transfer.Info.DestinationBalance
	token.DestinationPreBalance = transfer.Info.DestinationPreBalance
	token.SourceBalance = transfer.Info.SourceBalance
	token.SourcePreBalance = transfer.Info.SourcePreBalance
}

// NewEventFee returns the FeeInfo of a fee a program reported in an event
func (tu *TransactionUtils) NewEventFee(mint string, amount uint64, feeType, dex string) types.FeeInfo {
	decimals := tu.adapter.GetTokenDecimals(mint)
	raw := new(big.Int).SetUint64(amount)
	return types.FeeInfo{
		Mint:      mint,
		Amount:    types.ConvertToUIAmount(raw, decimals),
		AmountRaw: raw.String(),
		Decimals:  decimals,
		Dex:       dex,
		Type:      feeType,
	}
}
