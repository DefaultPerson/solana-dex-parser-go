package propamm

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// SolFiParser parses SolFi (V1) DEX transactions. SolFi V1 is legacy: its
// recent transactions only reference it (see NewSolFiV2Parser).
type SolFiParser = VenueParser

// decodeSolFi: tag 0x07 (18 bytes in 2DaS55Tw...: u64 amount_in at 1,
// direction at 17). Accounts: 0 user authority, 1 pair, 2/3 pool vaults A/B,
// 4/5 user token accounts A/B, 6 token program, 7 sysvar instructions.
func decodeSolFi(data []byte, n int) *swapLayout {
	if !hasTag(data, disc.SOLFI.SWAP) || n < 8 {
		return nil
	}
	return &swapLayout{pool: 1, legs: bothDirections(4, 5, 2, 3)}
}

// NewSolFiParser creates a new SolFi parser
func NewSolFiParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *SolFiParser {
	return newVenueParser(constants.DEX_PROGRAMS.SOLFI, decodeSolFi, adapter, dexInfo, transferActions, classifiedInstructions)
}
