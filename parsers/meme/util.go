package meme

import (
	"encoding/base64"
	"math/big"
	"sort"
	"strings"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

func formatIdx(outerIndex int, innerIndex int) string {
	return utils.FormatIdx(outerIndex, innerIndex)
}

// sortExecutionOrder sorts instructions in execution order in place: an
// outer instruction first, then its inner instructions by inner index
func sortExecutionOrder(instructions []types.ClassifiedInstruction) {
	sort.SliceStable(instructions, func(i, j int) bool {
		if instructions[i].OuterIndex != instructions[j].OuterIndex {
			return instructions[i].OuterIndex < instructions[j].OuterIndex
		}
		return instructions[i].InnerIndex < instructions[j].InnerIndex
	})
}

// instructionTransfers returns the transfer and transferChecked actions
// grouped under the instruction (outer instructions keep inner index -1)
func instructionTransfers(transferActions map[string][]types.TransferData, ci types.ClassifiedInstruction) []types.TransferData {
	var out []types.TransferData
	for _, t := range transferActions[utils.FormatTransferKey(ci.ProgramId, ci.OuterIndex, ci.InnerIndex)] {
		if t.Type == "transfer" || t.Type == "transferChecked" {
			out = append(out, t)
		}
	}
	return out
}

// userLegs finds what the user sent and received in an instruction's
// transfers: in sums the transfers of inMint the user authorised to accounts
// of others, out sums the transfers of outMint the user did not authorise
// into accounts the user owns (native SOL: to the user's wallet). Moves
// between the user's own accounts (wrapping SOL) count on neither side. When
// no destination owner is known, a single transfer of outMint not authorised
// by the user counts as the output. A nil result means the side was not
// found.
func userLegs(a *adapter.TransactionAdapter, transfers []types.TransferData, user, inMint, outMint string) (in, out *types.TokenInfo) {
	sum := func(mint string, keep func(t *types.TransferData) bool) *types.TokenInfo {
		total := new(big.Int)
		found := false
		var decimals uint8
		for i := range transfers {
			t := &transfers[i]
			if t.Info.Mint != mint || !keep(t) {
				continue
			}
			v, ok := new(big.Int).SetString(t.Info.TokenAmount.Amount, 10)
			if !ok {
				continue
			}
			total.Add(total, v)
			decimals = t.Info.TokenAmount.Decimals
			found = true
		}
		if !found {
			return nil
		}
		if d, ok := a.SPLDecimalsMap[mint]; ok {
			decimals = d
		} else if d, ok := constants.TOKEN_DECIMALS[mint]; ok {
			decimals = d
		}
		return &types.TokenInfo{
			Mint:      mint,
			AmountRaw: total.String(),
			Amount:    types.ConvertToUIAmount(total, decimals),
			Decimals:  decimals,
		}
	}
	owner := func(t *types.TransferData) string {
		if t.Info.DestinationOwner != "" {
			return t.Info.DestinationOwner
		}
		if o := a.GetTokenAccountOwner(t.Info.Destination); o != "" {
			return o
		}
		return ""
	}

	byUser := func(t *types.TransferData) bool { return t.Info.Authority == user || t.Info.Source == user }
	toUser := func(t *types.TransferData) bool { return t.Info.Destination == user || owner(t) == user }
	in = sum(inMint, func(t *types.TransferData) bool { return byUser(t) && !toUser(t) })
	out = sum(outMint, func(t *types.TransferData) bool { return toUser(t) && !byUser(t) })
	if out == nil {
		var candidates []*types.TransferData
		for i := range transfers {
			t := &transfers[i]
			if t.Info.Mint == outMint && t.Info.Authority != user && owner(t) == "" {
				candidates = append(candidates, t)
			}
		}
		if len(candidates) == 1 {
			c := candidates[0]
			out = sum(outMint, func(t *types.TransferData) bool { return t == c })
		}
	}
	return in, out
}

// tokenInfoFromRaw builds a token amount with the decimals known to the
// transaction (0 when unknown)
func tokenInfoFromRaw(a *adapter.TransactionAdapter, mint string, amount *big.Int) *types.TokenInfo {
	decimals := a.GetTokenDecimals(mint)
	return &types.TokenInfo{
		Mint:      mint,
		AmountRaw: amount.String(),
		Amount:    types.ConvertToUIAmount(amount, decimals),
		Decimals:  decimals,
	}
}

// programDataByInvocation returns, for each invocation of programId in log
// order, the payloads of the "Program data:" lines that invocation emitted
// itself (not its CPIs). It returns nil when the logs are missing, truncated
// or do not nest cleanly, so callers never attribute data to the wrong
// instruction. Invocations follow execution order, which is the order of
// the program's instructions sorted by (outer, inner) index.
func programDataByInvocation(logs []string, programId string) [][][]byte {
	if len(logs) == 0 {
		return nil
	}
	var stack []string
	var result [][][]byte
	var current []int // index into result for each stack frame of programId, -1 otherwise
	for _, line := range logs {
		switch {
		case strings.HasPrefix(line, "Program ") && strings.Contains(line, " invoke ["):
			id := strings.TrimPrefix(line, "Program ")
			id = id[:strings.Index(id, " ")]
			stack = append(stack, id)
			if id == programId {
				result = append(result, nil)
				current = append(current, len(result)-1)
			} else {
				current = append(current, -1)
			}
		case strings.HasPrefix(line, "Program data: "):
			if len(stack) == 0 {
				return nil
			}
			if k := current[len(current)-1]; k >= 0 {
				data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "Program data: "))
				if err != nil {
					return nil
				}
				result[k] = append(result[k], data)
			}
		case strings.HasPrefix(line, "Program ") && (strings.HasSuffix(line, " success") || strings.Contains(line, " failed")):
			if len(stack) == 0 {
				return nil
			}
			id := strings.TrimPrefix(line, "Program ")
			if i := strings.Index(id, " "); i < 0 || id[:i] != stack[len(stack)-1] {
				return nil
			}
			stack = stack[:len(stack)-1]
			current = current[:len(current)-1]
		case strings.HasPrefix(line, "Log truncated"):
			return nil
		}
	}
	return result
}
