package propamm

import (
	"encoding/binary"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// HumidiFi XOR encryption key
var humidiFiXorKey = []byte{58, 255, 47, 255, 226, 186, 235, 195}

// HumidiFiParser parses HumidiFi DEX transactions
type HumidiFiParser = VenueParser

// deobfuscateHumidiFi decrypts XOR-obfuscated HumidiFi instruction data
func deobfuscateHumidiFi(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	result := make([]byte, len(data))
	copy(result, data)

	pos := 0
	for i := 0; i < len(result); i += 8 {
		chunkSize := 8
		if len(result)-i < 8 {
			chunkSize = len(result) - i
		}

		// Create position mask
		posMask := make([]byte, 8)
		for j := 0; j < 8; j += 2 {
			binary.LittleEndian.PutUint16(posMask[j:], uint16(pos))
		}

		// XOR each byte
		for j := 0; j < chunkSize; j++ {
			result[i+j] ^= humidiFiXorKey[j] ^ posMask[j]
		}
		pos++
	}

	return result
}

// decodeHumidiFi recognises the two swap forms; the data has no plain tag.
//   - 25 bytes, 9 or more accounts. Deobfuscated: 0-8 caller nonce, 8-16 u64
//     amount_in, 16-24 u64 direction (0 = A in, 1 = B in), 24 a flag byte.
//   - 113 bytes, 12 or more accounts (signed quote). Deobfuscated: 0-8 u64
//     amount_in, 8-16 u64 flags, 16-24 u64 quote expiry, then the signature.
//
// Accounts: 0 user authority, 1 pool, 2/3 pool vaults A/B, 4/5 user token
// accounts A/B, 6 sysvar clock, 7/8 token programs, 9 sysvar instructions.
// The 65-byte, 3-account instructions are oracle/quote updates.
// DISCRIMINATORS.HUMIDIFI.SWAP (Anchor "swap") does not occur in HumidiFi data.
func decodeHumidiFi(data []byte, n int) *swapLayout {
	switch {
	case len(data) == 25 && n >= 9:
		if binary.LittleEndian.Uint64(deobfuscateHumidiFi(data)[16:24]) > 1 {
			return nil
		}
	case len(data) == 113 && n >= 12:
	default:
		return nil
	}
	return &swapLayout{pool: 1, legs: bothDirections(4, 5, 2, 3)}
}

// NewHumidiFiParser creates a new HumidiFi parser
func NewHumidiFiParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *HumidiFiParser {
	return newVenueParser(constants.DEX_PROGRAMS.HUMIDIFI, decodeHumidiFi, adapter, dexInfo, transferActions, classifiedInstructions)
}
