package utils

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
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
// Instruction positions are rebuilt from the "Program <id> invoke [<depth>]"
// lines: depth 1 starts the next outer instruction, deeper invokes are the
// next inner instruction of the current outer instruction (the order of
// meta.innerInstructions). "Program <id> success" and "Program <id> failed"
// return to the caller, so a payload written after a CPI returned belongs to
// the caller. outerProgramIds, when given, are the program ids of the outer
// instructions; they keep the outer index aligned when an outer instruction
// writes no invoke line. Lines that cannot be decoded are skipped, and
// decoding stops at "Log truncated": the payloads after it are lost.
func ParseProgramLogs(logs []string, outerProgramIds []string, prefix string) []ProgramLog {
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
			if outer < 0 || depth-1 > len(stack) {
				continue // no enclosing instruction: the log is not usable
			}
			inner++
			stack = append(stack[:depth-1], frame{programId: programId, innerIndex: inner})
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
	if !ok || strings.Contains(programId, " ") {
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
	if !ok {
		return "", false
	}
	if tail == "success" || strings.HasPrefix(tail, "failed") {
		return programId, true
	}
	return "", false
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

// GetProgramDataLogs returns the "Program data:" payloads of the transaction
// logs with the instruction that wrote each one (see ParseProgramLogs). It
// returns nil when the transaction carries no logs.
func (tu *TransactionUtils) GetProgramDataLogs() []ProgramLog {
	logs := tu.adapter.LogMessages()
	if len(logs) == 0 {
		return nil
	}
	return ParseProgramDataLogs(logs, outerProgramIds(tu.adapter))
}

// GetRayLogs returns the Raydium AMM v4 "ray_log:" payloads of the
// transaction logs with the instruction that wrote each one.
func (tu *TransactionUtils) GetRayLogs() []ProgramLog {
	logs := tu.adapter.LogMessages()
	if len(logs) == 0 {
		return nil
	}
	return ParseRayLogs(logs, outerProgramIds(tu.adapter))
}
