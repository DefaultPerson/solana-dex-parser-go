package propamm

import (
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Swap layouts of the venues added in 2026-09. None of the prop AMMs has a
// published IDL: tags, data lengths and account roles were taken from mainnet
// swaps, where every recognised swap moved exactly the two transfers named
// here (see the tests). Tags collide across programs (0x07, 0x10, 0x02), so a
// decoder only applies to its own program's instructions.

// decodeSolFiV2 (SV2EYY...): tag 0x07, 18 or 25 bytes (u64 amount_in at 1,
// direction at 17). Accounts: 0 user authority, 1 pool, 2 config, 3 global
// state, 4/5 pool vaults A/B, 6/7 user token accounts A/B, 8/9 mints A/B,
// 10/11 token programs, 12 sysvar instructions.
func decodeSolFiV2(data []byte, n int) *swapLayout {
	if !hasTag(data, disc.SOLFI_V2.SWAP, 18, 25) || n < 13 {
		return nil
	}
	return &swapLayout{pool: 1, legs: bothDirections(6, 7, 4, 5)}
}

// decodeGoonFiV2 (goonud...): tag 0x01, 18 or 19 bytes (direction at 1, u64
// amount_in at 2). Accounts: 0 user authority, 1 pool, 2/3 user token
// accounts A/B, 4/5 pool vaults A/B, 6/7 mints A/B, 8 oracle, 9 global state,
// 10 sysvar instructions, 11/12 token programs.
func decodeGoonFiV2(data []byte, n int) *swapLayout {
	if !hasTag(data, disc.GOONFI_V2.SWAP, 18, 19) || n < 13 {
		return nil
	}
	return &swapLayout{pool: 1, legs: bothDirections(2, 3, 4, 5)}
}

// decodeBisonFi (BiSoNH...): tags 0x02 (18 bytes), 0x07 (19 bytes) and 0x13
// (153 bytes, signed quote), u64 amount_in at 1. Accounts: 0 user authority,
// 1 pool, 2/3 pool vaults A/B, 4/5 user token accounts A/B, 6/7 token
// programs. Tags 0x14 and 0x1b are quote updates.
func decodeBisonFi(data []byte, n int) *swapLayout {
	if n < 8 {
		return nil
	}
	if !hasTag(data, disc.BISONFI.SWAP, 18) && !hasTag(data, disc.BISONFI.SWAP_V2, 19) &&
		!hasTag(data, disc.BISONFI.SWAP_WITH_SIG, 153) {
		return nil
	}
	return &swapLayout{pool: 1, legs: bothDirections(4, 5, 2, 3)}
}

// decodeTesseraV (TessVd...): tag 0x10, 18 bytes (direction at 1, u64
// amount_in at 2). Accounts: 0 pool, 1 pool state, 2 user authority, 3/4 pool
// vaults A/B, 5/6 user token accounts A/B, 7/8 mints A/B, 9/10 token
// programs, 11 sysvar instructions. Tags 0x0d and 0x13 are quote updates.
func decodeTesseraV(data []byte, n int) *swapLayout {
	if !hasTag(data, disc.TESSERA_V.SWAP, 18) || n < 12 {
		return nil
	}
	return &swapLayout{pool: 0, legs: bothDirections(5, 6, 3, 4)}
}

// decodeAlphaQ (ALPHAQ...): tag 0x0c, 18 bytes (direction at 1, u64 amount_in
// at 2). Accounts: 0 user authority, 1 pair, 2 pool state, 3/4 user token
// accounts A/B, 5/6 pool vaults A/B (self-owned), 7-9 vaults again, 10 token
// program, 11 sysvar instructions. Tag 0x0b is an update.
func decodeAlphaQ(data []byte, n int) *swapLayout {
	if !hasTag(data, disc.ALPHAQ.SWAP, 18) || n < 12 {
		return nil
	}
	return &swapLayout{pool: 1, legs: bothDirections(3, 4, 5, 6)}
}

// decodeZeroFi (ZERor4...): tag 0x10 ("swap_v4"), 18 bytes (u64 amount_in at
// 1). The accounts are in input-first order: 0 pool, 1 config, 2 input vault
// info, 3 pool input vault, 4 output vault info, 5 pool output vault, 6 user
// input account, 7 user output account, 8 user authority, 9-12 token programs
// and mints, 13 sysvar instructions. Tag 0x11 is an update.
func decodeZeroFi(data []byte, n int) *swapLayout {
	if !hasTag(data, disc.ZERO_FI.SWAP, 18) || n < 13 {
		return nil
	}
	return &swapLayout{pool: 0, legs: []swapLegs{{userIn: 6, vaultIn: 3, vaultOut: 5, userOut: 7}}}
}

// decodeScorch (SCoRcH...): tag 0x02 (34 bytes, 17 accounts) or the compact
// tag 0x01 (34 bytes, 13-14 accounts), u64 amount_in at 18. Input-first
// accounts: 0 vault authority (shared by all pairs), 1 user authority, 2 user
// input account, 3 user output account, 4 pool input vault, 5 pool output
// vault; then mints and token programs (0x02 only), the Scorch pricing
// program and its accounts, the last of which is the per-pair account (15
// for 0x02, 11 for 0x01) used as the pool.
func decodeScorch(data []byte, n int) *swapLayout {
	legs := []swapLegs{{userIn: 2, vaultIn: 4, vaultOut: 5, userOut: 3}}
	switch {
	case hasTag(data, disc.SCORCH.SWAP, 34) && n >= 16:
		return &swapLayout{pool: 15, legs: legs}
	case hasTag(data, disc.SCORCH.SWAP_COMPACT, 34) && n >= 12:
		return &swapLayout{pool: 11, legs: legs}
	}
	return nil
}

// decodeQuantum (QuaNtZ...): tag 0x07, 18 bytes (u64 amount_in at 1,
// direction at 17). Accounts: 0 user authority, 1/2 user token accounts A/B,
// 3 pool, 4/5 pool vaults A/B, 6 global state, 7 token program, 8 sysvar
// clock, 9 sysvar instructions. Tags 0x0a and 0x14 are quote updates.
func decodeQuantum(data []byte, n int) *swapLayout {
	if !hasTag(data, disc.QUANTUM.SWAP, 18) || n < 10 {
		return nil
	}
	return &swapLayout{pool: 3, legs: bothDirections(1, 2, 4, 5)}
}

// decodeManifest (MNFSTq..., CKS-Systems/manifest instruction.rs): Swap (4)
// and SwapV2 (13), 19 bytes of SwapParams. Swap accounts: 0 payer, 1 market,
// 2 system program, 3/4 trader base/quote, 5/6 base/quote vault. SwapV2 adds
// the owner at 1, so market is 2, traders 4/5 and vaults 6/7. BatchUpdate and
// the other order-book instructions are not trades.
func decodeManifest(data []byte, n int) *swapLayout {
	switch {
	case hasTag(data, disc.MANIFEST.SWAP_V2, 19) && n >= 9:
		return &swapLayout{pool: 2, legs: bothDirections(4, 5, 6, 7)}
	case hasTag(data, disc.MANIFEST.SWAP, 19) && n >= 8:
		return &swapLayout{pool: 1, legs: bothDirections(3, 4, 5, 6)}
	}
	return nil
}

// decodeByreal (REALQq..., on-chain IDL byreal_clmm): swap, swap_v2 and
// swap_v3_dyn share the Raydium CLMM accounts: 0 payer, 1 amm_config, 2
// pool_state, 3 input_token_account, 4 output_token_account, 5 input_vault,
// 6 output_vault, then observation state, token programs and tick arrays.
func decodeByreal(data []byte, n int) *swapLayout {
	if n < 10 || !(constants.MatchDiscriminator(data, disc.BYREAL.SWAP) ||
		constants.MatchDiscriminator(data, disc.BYREAL.SWAP_V2) ||
		constants.MatchDiscriminator(data, disc.BYREAL.SWAP_V3_DYN)) {
		return nil
	}
	return &swapLayout{pool: 2, legs: []swapLegs{{userIn: 3, vaultIn: 5, vaultOut: 6, userOut: 4}}}
}

// decodeSarosDLMM (1qbkdr..., IDL liquidity_book): swap, 26 bytes. Accounts:
// 0 pair, 1/2 mints X/Y, 3/4 bin arrays, 5/6 pool vaults X/Y, 7/8 user token
// accounts X/Y, 9 user, 10-16 programs and event authority. Its emit_cpi
// event instructions are not swaps.
func decodeSarosDLMM(data []byte, n int) *swapLayout {
	if !hasTag(data, disc.SAROS_DLMM.SWAP, 26) || n < 17 {
		return nil
	}
	return &swapLayout{pool: 0, legs: bothDirections(7, 8, 5, 6)}
}

// NewSolFiV2Parser creates a SolFi V2 trade parser
func NewSolFiV2Parser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.SOLFI_V2, decodeSolFiV2, a, d, t, c)
}

// NewGoonFiV2Parser creates a GoonFi V2 trade parser
func NewGoonFiV2Parser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.GOONFI_V2, decodeGoonFiV2, a, d, t, c)
}

// NewBisonFiParser creates a BisonFi trade parser
func NewBisonFiParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.BISONFI, decodeBisonFi, a, d, t, c)
}

// NewTesseraVParser creates a TesseraV trade parser
func NewTesseraVParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.TESSERA_V, decodeTesseraV, a, d, t, c)
}

// NewAlphaQParser creates an AlphaQ trade parser
func NewAlphaQParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.ALPHAQ, decodeAlphaQ, a, d, t, c)
}

// NewZeroFiParser creates a ZeroFi trade parser
func NewZeroFiParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.ZERO_FI, decodeZeroFi, a, d, t, c)
}

// NewScorchParser creates a Scorch trade parser. Scorch swaps run in
// DEX_PROGRAMS.SCORCH; its pricing program SCORCH_PRICING moves no tokens.
func NewScorchParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.SCORCH, decodeScorch, a, d, t, c)
}

// NewQuantumParser creates a Quantum trade parser
func NewQuantumParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.QUANTUM, decodeQuantum, a, d, t, c)
}

// NewManifestParser creates a Manifest (order book) trade parser
func NewManifestParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.MANIFEST, decodeManifest, a, d, t, c)
}

// NewByrealParser creates a Byreal CLMM trade parser
func NewByrealParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.BYREAL, decodeByreal, a, d, t, c)
}

// NewSarosDLMMParser creates a Saros DLMM trade parser
func NewSarosDLMMParser(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) *VenueParser {
	return newVenueParser(constants.DEX_PROGRAMS.SAROS_DLMM, decodeSarosDLMM, a, d, t, c)
}
