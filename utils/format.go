package utils

import "strconv"

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
