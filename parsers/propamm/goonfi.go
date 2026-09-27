package propamm

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// GoonFiParser parses GoonFi (V1) DEX transactions. GoonFi V1 is legacy: its
// vaults were emptied and closed in 2026-02 (see NewGoonFiV2Parser).
type GoonFiParser = VenueParser

// decodeGoonFi: tag 0x02 (19-20 bytes, direction at 1, u64 amount_in at 2;
// e.g. 4zQF9pQb...). Accounts: 0 user authority, 1 pool, 2/3 user token
// accounts A/B, 4/5 pool vaults A/B, 6 oracle, 7 sysvar instructions, 8 token
// program. Tag 0x04 (admin vault withdrawal, 2 transfers), 0x08 (quote
// update) and 0x09 (close) are not swaps.
func decodeGoonFi(data []byte, n int) *swapLayout {
	if !hasTag(data, disc.GOONFI.SWAP) || n < 6 {
		return nil
	}
	return &swapLayout{pool: 1, legs: bothDirections(2, 3, 4, 5)}
}

// NewGoonFiParser creates a new GoonFi parser
func NewGoonFiParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *GoonFiParser {
	return newVenueParser(constants.DEX_PROGRAMS.GOONFI, decodeGoonFi, adapter, dexInfo, transferActions, classifiedInstructions)
}
