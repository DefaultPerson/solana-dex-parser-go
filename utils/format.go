package utils

import (
	"strconv"
	"strings"
)

// FormatTransferKey formats a transfer key as "programId:outer" or "programId:outer-inner"
func FormatTransferKey(programId string, outer, inner int) string {
	if inner < 0 {
		return programId + ":" + strconv.Itoa(outer)
	}
	return programId + ":" + strconv.Itoa(outer) + "-" + strconv.Itoa(inner)
}

// FormatDedupeKey formats a deduplication key for trades as "idx-signature"
func FormatDedupeKey(idx, signature string) string {
	return idx + "-" + signature
}

// SplitIdx returns the outer and inner instruction index of an idx made by
// FormatIdx ("5" gives 5, -1; "5-3" gives 5, 3), or -1, -1 when it cannot be
// parsed
func SplitIdx(idx string) (outer, inner int) {
	outerStr, innerStr, hasInner := strings.Cut(idx, "-")
	outer, err := strconv.Atoi(outerStr)
	if err != nil {
		return -1, -1
	}
	if !hasInner {
		return outer, -1
	}
	inner, err = strconv.Atoi(innerStr)
	if err != nil {
		return -1, -1
	}
	return outer, inner
}
