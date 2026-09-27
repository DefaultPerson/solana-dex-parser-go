package pumpfun

import (
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// defaultPubkey is Pubkey::default() in base58; Pump.fun events use it for
// "no quote mint" (SOL-paired coins)
const defaultPubkey = "11111111111111111111111111111111"

// tailReader reads the optional trailing fields of an event. Events grow by
// appending fields, so older (shorter) events simply end early: every read
// checks the remaining bytes first, and after the first missing field ok is
// false and all later reads return zero values without touching the buffer.
type tailReader struct {
	r  *utils.BinaryReader
	ok bool
}

func newTailReader(r *utils.BinaryReader) *tailReader {
	return &tailReader{r: r, ok: !r.HasError()}
}

func (t *tailReader) has(n int) bool {
	if t.ok && t.r.Remaining() < n {
		t.ok = false
	}
	return t.ok
}

func (t *tailReader) u64() uint64 {
	if !t.has(8) {
		return 0
	}
	v, _ := t.r.ReadU64()
	return v
}

func (t *tailReader) u16() uint16 {
	if !t.has(2) {
		return 0
	}
	v, _ := t.r.ReadU16()
	return v
}

func (t *tailReader) boolean() bool {
	if !t.has(1) {
		return false
	}
	v, _ := t.r.ReadBool()
	return v
}

func (t *tailReader) pubkey() string {
	if !t.has(32) {
		return ""
	}
	v, _ := t.r.ReadPubkey()
	return v
}

// i128 reads a signed 128-bit integer
func (t *tailReader) i128() *big.Int {
	if !t.has(16) {
		return nil
	}
	lo, hi, _ := t.r.ReadI128()
	v := new(big.Int).SetInt64(hi)
	v.Lsh(v, 64)
	return v.Add(v, new(big.Int).SetUint64(lo))
}

// str reads a Borsh string; a length beyond the buffer ends the tail
func (t *tailReader) str() string {
	if !t.has(4) {
		return ""
	}
	s, err := t.r.ReadString()
	if err != nil {
		t.ok = false
		return ""
	}
	return s
}

// skipVec skips a Borsh vector of elemSize-byte elements; the element count is
// checked against the remaining bytes before anything is skipped
func (t *tailReader) skipVec(elemSize int) {
	if !t.has(4) {
		return
	}
	n, err := t.r.ReadVecLength(elemSize)
	if err != nil {
		t.ok = false
		return
	}
	_ = t.r.Skip(n * elemSize)
}

// isEventData reports whether data is an Anchor self-CPI event
func isEventData(data []byte) bool {
	return constants.IsAnchorEvent(data)
}

// executionOrder returns a copy of instructions sorted in execution order: an
// outer instruction first, then its inner instructions by inner index.
func executionOrder(instructions []types.ClassifiedInstruction) []types.ClassifiedInstruction {
	ordered := make([]types.ClassifiedInstruction, len(instructions))
	copy(ordered, instructions)
	utils.SortInstructionsByExecution(ordered)
	return ordered
}

// findParentInstruction returns the instruction of programId that emitted
// the event at position pos of ordered and for which match returns true (see
// utils.FindEventEmitter: the event's parent when stack heights are known,
// else the nearest preceding matching instruction in the same outer group),
// or nil
func findParentInstruction(
	a *adapter.TransactionAdapter,
	ordered []types.ClassifiedInstruction,
	pos int,
	programId string,
	match func(data []byte, accounts []string) bool,
) *types.ClassifiedInstruction {
	if pos < 0 || pos >= len(ordered) || ordered[pos].ProgramId != programId {
		return nil
	}
	return utils.FindEventEmitter(a, ordered, ordered[pos], match)
}

// tokenDecimals returns the decimals of mint known to the transaction (token
// balances, instructions), then the constants table, else fallback
func tokenDecimals(a *adapter.TransactionAdapter, mint string, fallback uint8) uint8 {
	if d, ok := a.SPLDecimalsMap[mint]; ok {
		return d
	}
	if d, ok := constants.TOKEN_DECIMALS[mint]; ok {
		return d
	}
	return fallback
}

// normalizeQuoteMint maps Pubkey::default and wrapped SOL to the SOL mint
func normalizeQuoteMint(mint string) string {
	if mint == "" || mint == defaultPubkey {
		return constants.TOKENS.SOL
	}
	return mint
}

// feeInfo builds a fee component with exact raw and UI amounts
func feeInfo(mint string, amount *big.Int, decimals uint8, dex, feeType, recipient string) types.FeeInfo {
	return types.FeeInfo{
		Mint:      mint,
		Amount:    types.ConvertToUIAmount(amount, decimals),
		AmountRaw: amount.String(),
		Decimals:  decimals,
		Dex:       dex,
		Type:      feeType,
		Recipient: recipient,
	}
}

func u64(v uint64) *big.Int {
	return new(big.Int).SetUint64(v)
}

func uiPtr(amount *big.Int, decimals uint8) *float64 {
	v := types.ConvertToUIAmount(amount, decimals)
	return &v
}
