package meme

import "fmt"

func formatIdx(outerIndex int, innerIndex int) string {
	return fmt.Sprintf("%d-%d", outerIndex, innerIndex)
}
