package propamm

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// ObricParser parses Obric V2 DEX transactions
type ObricParser = VenueParser

// decodeObric: Anchor swap and swap2 (25 bytes: is_x_to_y at 8, u64 amount_in
// at 9, u64 min_amount_out at 17; 2DaS55Tw... and 33VnDBtr...). Accounts: 0
// trading pair (vault authority), 1/2 oracle accounts, 3/4 pool vaults X/Y,
// 5/6 user token accounts X/Y, 7/8 unknown, 9 sysvar instructions, 10 user
// authority, 11 token program. DISCRIMINATORS.OBRIC.SWAP_X_TO_Y and
// SWAP_Y_TO_X were never seen in an Obric transaction (SWAP_X_TO_Y is
// Raydium CPMM's swap_base_input) and are not matched.
func decodeObric(data []byte, n int) *swapLayout {
	if n < 7 || !(constants.MatchDiscriminator(data, disc.OBRIC.SWAP) ||
		constants.MatchDiscriminator(data, disc.OBRIC.SWAP2)) {
		return nil
	}
	return &swapLayout{pool: 0, legs: bothDirections(5, 6, 3, 4)}
}

// NewObricParser creates a new Obric parser
func NewObricParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *ObricParser {
	return newVenueParser(constants.DEX_PROGRAMS.OBRIC_V2, decodeObric, adapter, dexInfo, transferActions, classifiedInstructions)
}
