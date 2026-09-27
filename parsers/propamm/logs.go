package propamm

import (
	"encoding/base64"
	"strings"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
)

// programDataLog is a "Program data:" log line and the instruction that
// wrote it
type programDataLog struct {
	outer, inner int // inner is -1 for an outer instruction
	programId    string
	data         []byte
}

// programDataLogs attributes the "Program data:" lines of the transaction's
// logs to the instruction that was executing, by replaying the "Program X
// invoke [n]" and "Program X success|failed" lines against the outer and inner
// instructions. It stops at the first line it cannot align (truncated logs,
// logs of another transaction) and returns what it attributed until then.
func programDataLogs(a *adapter.TransactionAdapter) []programDataLog {
	type frame struct {
		outer, inner int
		programId    string
	}

	instructions := a.Instructions()
	innerSets := make(map[int][]interface{})
	for _, set := range a.InnerInstructions() {
		innerSets[set.Index] = set.Instructions
	}

	var result []programDataLog
	var stack []frame
	outer, innerCount := -1, 0

	for _, line := range a.LogMessages() {
		if data, ok := strings.CutPrefix(line, "Program data: "); ok {
			if len(stack) == 0 {
				return result
			}
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(data))
			if err != nil {
				continue
			}
			top := stack[len(stack)-1]
			result = append(result, programDataLog{outer: top.outer, inner: top.inner, programId: top.programId, data: decoded})
			continue
		}
		rest, ok := strings.CutPrefix(line, "Program ")
		if !ok {
			if strings.HasPrefix(line, "Log truncated") {
				return result
			}
			continue
		}
		programId, action, found := strings.Cut(rest, " ")
		if !found {
			continue
		}
		switch {
		case action == "invoke [1]":
			// Skip outer instructions that wrote no logs
			outer++
			for outer < len(instructions) && a.GetInstructionProgramId(instructions[outer]) != programId {
				outer++
			}
			if outer >= len(instructions) {
				return result
			}
			innerCount = 0
			stack = append(stack[:0], frame{outer: outer, inner: -1, programId: programId})
		case strings.HasPrefix(action, "invoke ["):
			set := innerSets[outer]
			if len(stack) == 0 || innerCount >= len(set) || a.GetInstructionProgramId(set[innerCount]) != programId {
				return result
			}
			stack = append(stack, frame{outer: outer, inner: innerCount, programId: programId})
			innerCount++
		case action == "success" || strings.HasPrefix(action, "failed"):
			if len(stack) == 0 || stack[len(stack)-1].programId != programId {
				return result
			}
			stack = stack[:len(stack)-1]
		}
	}

	return result
}
