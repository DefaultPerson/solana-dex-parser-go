package tests

import (
	"math/big"
	"strings"
	"testing"

	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/propamm"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// venueProgram maps a venue of the tests to its program and parser
type venueProgram struct {
	program constants.DexProgram
	parser  func(*adapter.TransactionAdapter, types.DexInfo, map[string][]types.TransferData, []types.ClassifiedInstruction) parsers.TradeParser
}

var venuePrograms = map[string]venueProgram{
	"SolFiV2": {constants.DEX_PROGRAMS.SOLFI_V2, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewSolFiV2Parser(a, d, t, c)
	}},
	"SolFi": {constants.DEX_PROGRAMS.SOLFI, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewSolFiParser(a, d, t, c)
	}},
	"GoonFiV2": {constants.DEX_PROGRAMS.GOONFI_V2, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewGoonFiV2Parser(a, d, t, c)
	}},
	"GoonFi": {constants.DEX_PROGRAMS.GOONFI, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewGoonFiParser(a, d, t, c)
	}},
	"BisonFi": {constants.DEX_PROGRAMS.BISONFI, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewBisonFiParser(a, d, t, c)
	}},
	"TesseraV": {constants.DEX_PROGRAMS.TESSERA_V, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewTesseraVParser(a, d, t, c)
	}},
	"AlphaQ": {constants.DEX_PROGRAMS.ALPHAQ, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewAlphaQParser(a, d, t, c)
	}},
	"ZeroFi": {constants.DEX_PROGRAMS.ZERO_FI, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewZeroFiParser(a, d, t, c)
	}},
	"Scorch": {constants.DEX_PROGRAMS.SCORCH, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewScorchParser(a, d, t, c)
	}},
	"Quantum": {constants.DEX_PROGRAMS.QUANTUM, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewQuantumParser(a, d, t, c)
	}},
	"Manifest": {constants.DEX_PROGRAMS.MANIFEST, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewManifestParser(a, d, t, c)
	}},
	"Byreal": {constants.DEX_PROGRAMS.BYREAL, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewByrealParser(a, d, t, c)
	}},
	"SarosDLMM": {constants.DEX_PROGRAMS.SAROS_DLMM, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewSarosDLMMParser(a, d, t, c)
	}},
	"HumidiFi": {constants.DEX_PROGRAMS.HUMIDIFI, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewHumidiFiParser(a, d, t, c)
	}},
	"ObricV2": {constants.DEX_PROGRAMS.OBRIC_V2, func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewObricParser(a, d, t, c)
	}},
}

// venueSwapCase is one swap instruction of a venue in a mainnet fixture. The
// expected pool and amounts were read from the raw transaction (the swap
// instruction's accounts and the two token transfers of its CPI group) and
// agree with research/venues.md; the test also checks them against the pool
// vault balance deltas.
type venueSwapCase struct {
	venue, sig, idx, pool string
	inMint, inAmount      string
	outMint, outAmount    string
}

var venueSwapCases = []venueSwapCase{
	// SolFi V2 via Titan (the audit's double-count case), unknown router, OKX
	{"SolFiV2", "4C2p65nuttUqBv5VhLNXHq6cNEaexT5i4YGbGSH2hodRGLBQyJBzRCkr2VXPKPNof8Lut8WSm51tkxVv3mvUkpuZ", "2-1", "FkEB6uvyzuoaGpgs4yRtFtxC4WJxhejNFbUkj5R6wR32", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "1000011964", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "999897114"},
	{"SolFiV2", "3nMGDt51vDAJZa96QEDNgND5ARY5xf8L8P4TpHLZDmJFJZW2mUGVGiVSwe1NNzMEXm9hpTT7JmnwKwWsx31wGu5M", "3-12", "FkEB6uvyzuoaGpgs4yRtFtxC4WJxhejNFbUkj5R6wR32", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "1001224", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "1001098"},
	{"SolFiV2", "5PrZGSv25nELLAXnGg6N7WL45UVJ6ZPPku4FWo4fgmew3qPDUEzBi8BNNujkxSKykjHKw9St6L4cvRfYFqzenr6D", "2-5", "FkEB6uvyzuoaGpgs4yRtFtxC4WJxhejNFbUkj5R6wR32", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "85558490", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "85547819"},
	// SolFi V1 (legacy), Jupiter route of 2024-11
	{"SolFi", "2DaS55TwMuryadgsirCLRJywZ7rbBoLugXnicbJR2YFSSgn8PAvGN8Mhr9p4Ux8SEtgPzrRRuqV9T7XS7WWtdxvR", "3-5", "B2e5QH2GDkRg6hj3y34GCUURBMGskPfX5RAAXe8RBRjS", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "4842762", "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263", "11348229108"},
	// GoonFi V2: 14 and 13 accounts
	{"GoonFiV2", "2MRXWhAaFQKgJmyxHdqddwZRyZWWbvU4RERznPPq8ghwGV5ATy1kReSUEHGSoFbxqkX3XgWDAyo5iKChWC8SMvAv", "3-2", "8TxrtAxqA5PA2Y1d2pxzCz9SoDhjrBcYqNAQKVv6p443", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "229977000", "cbbtcf3aa214zXHbiAZQwf4122FBYbraNdFqgw4iMij", "270511"},
	{"GoonFiV2", "3LpCQBijkmyVv1t4F5iBA3YkDjcQy1DiLJmUdHKATUZ1E2nVff7PvXxVn5oqCkn6pcz8MrTzA4dQ3DmProNMbVe9", "2-3", "8TDBxPXyGvxcaoHZoY5D4X2vePuhMYekKpTEhEhaQX5b", "98sMhvDwXj1RQi5c5Mndm3vPe9cBqPrbLaufMXFNMh5g", "630113149", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "58740222"},
	{"GoonFiV2", "5PrZGSv25nELLAXnGg6N7WL45UVJ6ZPPku4FWo4fgmew3qPDUEzBi8BNNujkxSKykjHKw9St6L4cvRfYFqzenr6D", "2-1", "4rJggoVMajEUtipev1XhSMjESYk8Zibz6CDHPtUe1mem", "So11111111111111111111111111111111111111112", "690000000", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "85558490"},
	// GoonFi V1 (legacy), one of its last swaps before the vaults were closed (2026-02-06)
	{"GoonFi", "4zQF9pQbHntrkTQAQr8dRm5CegKB7XejFCAjVyzaVc1nvWGVxatLZ5WtBc7PTj6rLpPr3YKAs81AzVr4rW5Pwp2X", "1-1", "4uWuh9fC7rrZKrN8ZdJf69MN1e2S7FPpMqcsyY1aof6K", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "500000000", "So11111111111111111111111111111111111111112", "6217548216"},
	// BisonFi: tags 0x02, 0x07 and 0x13 (signed quote)
	{"BisonFi", "22VFxreHs9cEvHoYiooU62iLsK76vJo6E1WGXsjFRcM7PwCrUUhNiBhvUMztPCdxCgyfcDCB79TUaRDEC2VPLYAF", "2-0", "8FnX3xo2yYw3EUE6w3nQA4GfXGS9wpK6oj3veJpbFzLo", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "248815402", "So11111111111111111111111111111111111111112", "2003709082"},
	{"BisonFi", "2AbymqA3xMNTPgcPu6qh9EZWUnXtudS3JNmvzAU4c4s5ziPhwVwUQx5PZsmQahfqigvzcRg99SLCvRCrVvNzGCvm", "3-0", "8FnX3xo2yYw3EUE6w3nQA4GfXGS9wpK6oj3veJpbFzLo", "So11111111111111111111111111111111111111112", "122990681", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "15245220"},
	{"BisonFi", "4e8T9Hnh2AbXV482VcwLr1FWcZeCtQNKW5qsw3uJRt5oQpvyp6qRK2aoxP7JYHaQRScEzFWid1krx7ezDUNAyx1X", "3-8", "6b5LxeDVxqCGAhZjjjgieGP71c5GBt2cBwiafCFX6NMU", "So11111111111111111111111111111111111111112", "33756628", "USD1ttGY1N17NEEHLmELoaybftRBUSErhqYiQzvEmuB", "4097114"},
	{"BisonFi", "262n1pEgfLq9G87vRNBsb5xKdmADSQxFGWXXb2qXkxqtsPViu5ifNWC2UsGCFpo1aXaagpsetjxNZKoR2mG6D3xw", "5-2", "8FnX3xo2yYw3EUE6w3nQA4GfXGS9wpK6oj3veJpbFzLo", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "26773035", "So11111111111111111111111111111111111111112", "215496676"},
	// TesseraV: pool at account 0, user authority at 2
	{"TesseraV", "YfyJgF3Ph3GhxoEFpZR9MUnr2EGspamfUZh2VYhQEFP2caVqrwQbxP6a4aqTPufCz2kNRXiF1tX1HFCHvYGToz2", "4-8", "8ekCy2jHHUbW2yeNGFWYJT9Hm9FW7SvZcZK66dSZCDiF", "So11111111111111111111111111111111111111112", "789792", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "166493"},
	{"TesseraV", "29b7yFfvur5vt9VQkvu1aPDZFUgastzdXpqQYAZ41XYgvW25UDz48EYMTZEHxtWQ3P8SsZ9SSALedVzTcGLvbo96", "3-1", "8ekCy2jHHUbW2yeNGFWYJT9Hm9FW7SvZcZK66dSZCDiF", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "99050000", "So11111111111111111111111111111111111111112", "797306100"},
	// AlphaQ: self-owned vaults; 12 and 14 accounts
	{"AlphaQ", "23RNqCyLYWXqPjEWednSdDiXHoie64EdyP94NpVJn1ic2HnxyaDmHLt5314j1HyYBDq2hDKw8Z1S8jw7U6duFhy4", "2-1", "Pi9nzTjPxD8DsRfRBGfKYzmefJoJM8TcXu2jyaQjSHm", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "42996568", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "42991563"},
	{"AlphaQ", "38iesCRDJKFGDFCnYPpuZiJd4xFTArXoy33LqvyZNTpvqRnp9CTYqZbqtEDiwMboRdMM6kMN86Shjga9NJ1QyrKg", "2-5", "2YR8bXXn4tTnq8nVjvpYnBiQ7ZKjN3G16wxE8ShL3KaB", "CASHx9KJUStyftLFWGvEVf59SGeG9sh5FfcnZMVPCASH", "99992438", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "99960237"},
	// ZeroFi: input-first accounts, both directions on one pool
	{"ZeroFi", "2fbxB1hkkbwvTj2kZLRVmWZuGA4poHXmsmQDDzCRQEV3gkqvnLDoLiL9o4smRK1eMUUXWTgH3oKsTCnAGxcmvUET", "3-3", "A59MtPWamLSFb4o6fLWprNAsmJbMSxQ3yKBoZcsXDCmk", "7vfCXTUXx5WJV5JADk17DUJ4ksgau7utNKj4b963voxs", "6525536", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "177096524"},
	{"ZeroFi", "3hWuUyvWrBuqgqizgeUcsCLNeQdW9TBVv6Hywve68dXsSRitgxxQbViURZfkSTTggcVCKBKiABQULRWjCmhAKrwC", "9-4", "A59MtPWamLSFb4o6fLWprNAsmJbMSxQ3yKBoZcsXDCmk", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "434641443", "7vfCXTUXx5WJV5JADk17DUJ4ksgau7utNKj4b963voxs", "16007517"},
	// Scorch: tag 0x02 (17 accounts), compact tag 0x01 (14 and 13 accounts)
	{"Scorch", "3tHnRiFwCdzB92qnpui67qviFD24YUuVcu7VMNhrUEYVL2xdVsmQy9v4of3bA9oEPxmMWckKicXtESKqY6zemP73", "2-0", "FnhxUP3dcQbypCUmGWw55ijxPBxifPT558UQSCYDfcCU", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "309365132", "So11111111111111111111111111111111111111112", "2490560000"},
	{"Scorch", "5dnD4akxGFE31hLnSfybpL24geVkFDHNVcyCsDwVLwig2HPwL26EjKRkwEm8CKajGamYRDV1DN4BkFDDWMnJntEv", "2-15", "382i8DWxUuNLYXrYqnbK151otneeqjsJgrtVpdKArtm6", "cbbtcf3aa214zXHbiAZQwf4122FBYbraNdFqgw4iMij", "255014", "So11111111111111111111111111111111111111112", "1742210000"},
	{"Scorch", "4v27ccyrAgpCdCHLvjvn8smFn4Fb4HGcRVTSt952eNcF5jg5niA5bKRLPoGrzxXZdZULEZujgA5TXdESNbwmFYE8", "2-19", "8aAuC8E1Sc72gniUHAJ3utwWzzLKP1YLzMF3hPgfaSaC", "Dfh5DzRgSvvCFDoYc2ciTkMrbDfRKybA4SoFbPmApump", "6631746", "SKRbvo6Gf7GondiT3BbTfuRDPqLWei4j2Qy2NPGZhW3", "15650000"},
	// Quantum: user accounts before the pool
	{"Quantum", "24P4ArQNW3Mq73FUgsD6HPB8d6FP12CdpCSPgcCXtU98Y2oDU7wcpT2zY8syZ9nX1176ujEbuiunTAjXDWjzPeQR", "2-0", "2SixkjMjaLqXcEEJZfeth7qjH711R2uvp6TrUaHNMB65", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "288797470", "So11111111111111111111111111111111111111112", "2324688699"},
	{"Quantum", "3yeYKP1NRVK8ARRVTu77dA8E6rpRg3x68p73wBSkiQ3aUzMVqN26ybUQfFteAZetmZE6h8hr7FHbfWoqXhuMJ5AE", "1-0", "8R5qdXKMn2KcfHBy9rEpi43KScHewqvRAcpFyqoL3wap", "So11111111111111111111111111111111111111112", "1842968968", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "228904965"},
	// Manifest: SwapV2 (13) and Swap (4)
	{"Manifest", "1T2XZsyrPudVFb6xKCocTq6Yv93Kxhy6kWYSE4244jeSix9WZDL1sbRcF1BTs8FmU4UiEWFcE2SL4p25NnWXcKn", "3-1", "8sjV1AqBFvFuADBCQHhotaRq5DFFYSjjg1jMyVWMqXvZ", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "2442100", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "2442405"},
	{"Manifest", "2Xgfh9FHC2EAjBErQgv52cfMurnsVc9kHpfsSws7SoYBaJD8Cgh47T9qBfhgksgvTAMgz1M48PiuRm1ynAW4QU84", "3-14", "2QLNw13jbu9EfiwQv579hL4qngWHHvj3yCwdqfmkuMWv", "J1yxV53EmVRmt9RUj6PPufMfdAYUbtgwXCuNYTBTsYzJ", "14541579714", "So11111111111111111111111111111111111111112", "177562935"},
	// Byreal swap_v3_dyn
	{"Byreal", "232QdyFyTFbhFFePUaPHfGmdrZc8X1bMbW3NSSxdA4bo6XFsoKdUTTtjjLqtP5AFkBUiU4H5oSatMdfJfmmVfACN", "5-5", "9GTj99g9tbz9U6UYDsX6YeRTgUnkYG6GTnHv3qLa5aXq", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "3150772", "So11111111111111111111111111111111111111112", "25368479"},
	{"Byreal", "4CLdJQBTakrQquPdqtfXF6SLFw6a54HPX33u4kdDhDH6MQR7BzjSyot8kfSTPM5vQzPwDZo3RbfqCmpoRfBavVcK", "2-5", "DyzGYEhdgSn5EEUt4XXviavZ7v7SV2YGrYV8HX3Aw5XT", "So11111111111111111111111111111111111111112", "87643", "USD1ttGY1N17NEEHLmELoaybftRBUSErhqYiQzvEmuB", "10887"},
	// Saros DLMM, both directions
	{"SarosDLMM", "2Tp9YKLWwkMterDpysv59pikgApqVNCpzyGbAM3nhMUQva5YqtR8U1dpPqEwfmYKof1RxP9P4mF8TGbm4D1xEwo8", "2-3", "CBFkUHqZs1EhaL1wfLP3UAYaVU12tVYZYj221bP3UKNT", "CPcf58MNikQw2G23kTVWQevRDeFDpdxMH7KkR7Lhpump", "5891122505", "SarosY6Vscao718M4A778z4CGtvcwcGef5M9MEH1LGL", "5628347640"},
	{"SarosDLMM", "5PjAGmBpcE6E598Ro6FYUbYnFjtQUHH1Ym4wbkz3dgJSZB6kEzaQGZQYMoBTrp32Ns9ycUhYextTdjDchpde1cRM", "1-7", "CBFkUHqZs1EhaL1wfLP3UAYaVU12tVYZYj221bP3UKNT", "SarosY6Vscao718M4A778z4CGtvcwcGef5M9MEH1LGL", "16274294859", "CPcf58MNikQw2G23kTVWQevRDeFDpdxMH7KkR7Lhpump", "17058850037"},
	// HumidiFi: 25-byte B in; output transfer before the input transfer
	// (the old parser reported this one reversed); 113-byte signed quote,
	// which also carries a System transfer to ignore; 25-byte A in
	{"HumidiFi", "2YDhCqd2KUQ9SHVZQFwiKFzWuP6HkfxxA3Wsu5k7xoMPY3NW36B6h55MSZYyH9sfW1dCjPxhGtf2U6mRZGhs8Cj4", "2-4", "8sKQHfjNhvmAw94PhfvfMcytmqW6jmxvwieYyzXCCPu", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "358053005", "So11111111111111111111111111111111111111112", "2886321962"},
	{"HumidiFi", "2kCAkYGndEdQZemihC8r5yeFfa2DCo2msxtLHZU8RqVb8SArQKyhAthgwgMEWc2xeBc5tXdh7N2bkRRfXqND6apM", "0-6", "8sKQHfjNhvmAw94PhfvfMcytmqW6jmxvwieYyzXCCPu", "So11111111111111111111111111111111111111112", "2003133373", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "248176935"},
	{"HumidiFi", "4tzaGfjUtBPNKaSJcZWcWbUeVyMj3aEsQ6XXpq341dJWUEYAFK7MX5jCcDKkCNuZRNLqBhEBMMVf8APdRXHtX4rT", "6-22", "8sKQHfjNhvmAw94PhfvfMcytmqW6jmxvwieYyzXCCPu", "So11111111111111111111111111111111111111112", "7092795050", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "879077963"},
	{"HumidiFi", "3dhDZRhpSNK5Naku3R9CKERXP4MXEYecwkBTpN4SqtJZcDCDsh3ob4J2jdMkAbNyT6KmkamqsBomTEQNFHrLkZ9u", "2-4", "8sKQHfjNhvmAw94PhfvfMcytmqW6jmxvwieYyzXCCPu", "So11111111111111111111111111111111111111112", "4043218367", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "502093499"},
	// Obric V2: Anchor swap (pool = trading pair at account 0) and swap2
	{"ObricV2", "2DaS55TwMuryadgsirCLRJywZ7rbBoLugXnicbJR2YFSSgn8PAvGN8Mhr9p4Ux8SEtgPzrRRuqV9T7XS7WWtdxvR", "3-1", "2Qee5WoA7Pr99DkGsNfgZRnfHM4fTfYPQ4d82HmrCKJ9", "CzLSujWBLFsSjncfkh59rUFqvafWcY5tzedWJSuypump", "5646888", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", "4842762"},
	{"ObricV2", "33VnDBtrFawBRYwDqomdsH57GL83B7eWTQN5mnga9F1whyMzcpdmURnPkAjqDte8Ja9EcsGcejhDYcUKkA9sE4HG", "4-7", "D94tFiBfJzdZmcH6GtV39iXexWyVpNfwEH3CxEbqvsvr", "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", "40791626", "So11111111111111111111111111111111111111112", "291882456"},
}

// venueTrades runs the venue's parser on the venue program's instructions
func venueTrades(t *testing.T, venue string, tx *adapter.SolanaTransaction) []types.TradeInfo {
	t.Helper()
	vp, ok := venuePrograms[venue]
	if !ok {
		t.Fatalf("unknown venue %s", venue)
	}
	ctx := newParseContext(tx, nil)
	return vp.parser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions, ctx.Classifier.GetInstructions(vp.program.ID)).ProcessTrades()
}

// transfersTouching counts the transfers of the transaction from or to account
func transfersTouching(transferActions map[string][]types.TransferData, account string) int {
	n := 0
	for _, tr := range utils.SortedTransfers(transferActions) {
		if tr.Info.Source == account || tr.Info.Destination == account {
			n++
		}
	}
	return n
}

// Every venue swap yields one trade at the swap instruction with the amounts
// of its own two transfers (meme-9, constants-11, meme-10, constants-7).
func TestVenuesSwapHops(t *testing.T) {
	for _, tc := range venueSwapCases {
		t.Run(tc.venue+"/"+tc.sig[:8]+"/"+tc.idx, func(t *testing.T) {
			tx := loadFixture(t, tc.sig)
			program := venuePrograms[tc.venue].program
			var found []types.TradeInfo
			for _, tr := range venueTrades(t, tc.venue, tx) {
				if tr.Idx == tc.idx {
					found = append(found, tr)
				}
			}
			if len(found) != 1 {
				t.Fatalf("want 1 trade at %s, got %d", tc.idx, len(found))
			}
			tr := found[0]
			if tr.ProgramId != program.ID || tr.AMM != program.Name {
				t.Errorf("program %s/%s, want %s/%s", tr.ProgramId, tr.AMM, program.ID, program.Name)
			}
			if len(tr.Pool) != 1 || tr.Pool[0] != tc.pool {
				t.Errorf("pool %v, want %s", tr.Pool, tc.pool)
			}
			if tr.InputToken.Mint != tc.inMint || tr.InputToken.AmountRaw != tc.inAmount {
				t.Errorf("input %s %s, want %s %s", tr.InputToken.AmountRaw, tr.InputToken.Mint, tc.inAmount, tc.inMint)
			}
			if tr.OutputToken.Mint != tc.outMint || tr.OutputToken.AmountRaw != tc.outAmount {
				t.Errorf("output %s %s, want %s %s", tr.OutputToken.AmountRaw, tr.OutputToken.Mint, tc.outAmount, tc.outMint)
			}
			if tr.Type != utils.GetTradeType(tc.inMint, tc.outMint) {
				t.Errorf("type %s", tr.Type)
			}

			// The pool's vaults moved exactly these amounts, unless the route
			// used the same vault more than once
			ctx := newParseContext(tx, nil)
			vaultIn, vaultOut := tr.InputToken.Destination, tr.OutputToken.Source
			if transfersTouching(ctx.TransferActions, vaultIn) == 1 {
				if d := accountTokenDelta(tx, vaultIn, tc.inMint); d.String() != tc.inAmount {
					t.Errorf("input vault %s delta %s, want %s", vaultIn, d, tc.inAmount)
				}
			}
			if transfersTouching(ctx.TransferActions, vaultOut) == 1 {
				if d := accountTokenDelta(tx, vaultOut, tc.outMint); new(big.Int).Neg(d).String() != tc.outAmount {
					t.Errorf("output vault %s delta %s, want -%s", vaultOut, d, tc.outAmount)
				}
			}
		})
	}
}

// Through DexParser.ParseAll a venue hop is reported with its own amounts,
// unless a Jupiter route covers its outer instruction; no trade at that idx
// carries other amounts (the unknown-DEX fallback used to sum the hop's
// transfers with the aggregator's).
func TestVenuesParseAllHops(t *testing.T) {
	parser := dexparser.NewDexParser()
	for _, tc := range venueSwapCases {
		t.Run(tc.venue+"/"+tc.sig[:8]+"/"+tc.idx, func(t *testing.T) {
			result := parser.ParseAll(loadFixture(t, tc.sig), nil)
			if !result.State {
				t.Fatalf("parse failed: %s", result.Msg)
			}
			outer := strings.SplitN(tc.idx, "-", 2)[0]
			hop, covered := false, false
			for _, tr := range result.Trades {
				if strings.SplitN(tr.Idx, "-", 2)[0] == outer && tr.ProgramId == constants.DEX_PROGRAMS.JUPITER.ID {
					covered = true
				}
				if tr.Idx != tc.idx {
					continue
				}
				if tr.InputToken.AmountRaw != tc.inAmount || tr.OutputToken.AmountRaw != tc.outAmount ||
					tr.InputToken.Mint != tc.inMint || tr.OutputToken.Mint != tc.outMint {
					t.Errorf("trade at %s: %s %s -> %s %s, want %s %s -> %s %s", tr.Idx,
						tr.InputToken.AmountRaw, tr.InputToken.Mint, tr.OutputToken.AmountRaw, tr.OutputToken.Mint,
						tc.inAmount, tc.inMint, tc.outAmount, tc.outMint)
				}
				hop = true
			}
			if !hop && !covered {
				t.Errorf("no trade at %s and no Jupiter route covering it", tc.idx)
			}
		})
	}
}

// Quote/oracle updates, admin and order-book instructions of the venues move
// no tokens to or from a user and are never trades.
func TestVenuesNonSwapInstructions(t *testing.T) {
	cases := []struct {
		venue, sig, what string
	}{
		{"BisonFi", "2BjRrnMpdkukeaBS7MYuSFZaz2rXnumCYpFMgork4cF3V2LHxMfXLa18SojPjohSJkScR2QMtGv5BdY11HULBsQg", "tag 0x14 quote update"},
		{"BisonFi", "Edj1GzHvKdrFxSNkjhH1mACJ8kNsGjddSFBGNgEhQQbXG9eHp1u4ZioPrESdyekXVS59429QjqiByGkfsmC731U", "tag 0x1b quote update"},
		{"TesseraV", "4xhkuVhwTfm7Cr2Upr96CqmhAD4U6i9vazgftbTNRETPGEQDvvF9Ht4pMqUdJXgbHJEUtWWSuEvxoG63DSQq47EY", "tag 0x0d quote update"},
		{"TesseraV", "5vUfFi6RmU6nQYyZkQd7UfcqqncRJ2cqRREHZKUA7Wd5GSp7JhzM81voVXZn3FZeyC8H6ifWRLJNmeGKbcPp6ahc", "tag 0x13 quote update"},
		{"AlphaQ", "ZHWxQqC2QPaBTcTZUyGzhVa3t1NdBPuKsowQzb8Sz1eSYSxmYfnqciZyj8r2wJWKFDkdVYDpjF25iu325kybXf3", "tag 0x0b update, 3 accounts"},
		{"AlphaQ", "22XBYjtSgKuJjcDCi5uNuz4nmzoPBzUQ8HQNn9Di9BLCd2bTEHU7tqKwRkVPY2gouR7rHWUV9dNbFyMfAgwDqBsD", "tag 0x0b update, 15 accounts"},
		{"ZeroFi", "3YSct1tKTMTceYEs4fpCxJLqpjWcYkX6zufDmQnCtA4K2XZbKmis1cTcdv2iG6aNTL2CaE6VdUKTgFGFNGj66B16", "tag 0x11 update"},
		{"Scorch", "23bHd9VasKUhimfGv5Qs7cu8d1yhVETq5vzC68NArW1i9fvYN7LjGtFvgvFzBoyimQiNSBpXGqYzKxYHsmgMYiNF", "pricing program quote update 0x38"},
		{"Scorch", "2LsMq39RFsmEMYa1CdJA12AQW8p3vYuKsMzG1ouFZHjPEdoddDwRk7pMh5qvLRvKCMvyzFo8eopXFRdZKfj3v7A1", "pricing program quote update 0x37"},
		{"Quantum", "2yGEqqGeNFkbKsCPp5gJu8UAhedgzEbyYpdr6HPLAxj6x8kR413Bv6dpoK89guLLmTLtznwnDmnAk4DsTyFyy4Rt", "tag 0x0a quote update"},
		{"Quantum", "3udzrJELcQJN5SUWxnbbH9Hmkko2rGU42S9JnXX7kjXTD6i5w7pUdyBwrpU8GQU6t22ZUdeHbYToC3QNxTYCyth4", "tag 0x14 quote update"},
		{"Manifest", "4M87aipMDPVRWZ93hdkfhNnK5paxe6ZF7nyx9yPCuKtWiije7ZLrjJQV6tPVg14dvmWCkfaZt8dzpQ5sX92f5iWJ", "BatchUpdate"},
		{"HumidiFi", "275Z7BNmhGqM9jsh5vYxT5jdKF6jusj2hR4eBRQyqp9TskWwjWMYCRq656RRumbi41Su2nntTdkRyS8uEb7Xyh12", "65-byte oracle update"},
		{"GoonFi", "3F62U2dzZdCgHbyvgMvHPAAwawmTW11hdzKZdh5t9yHG8tp4iLFUBU9aRNVpdLpEYJ4q8kG8ck6iFwvuNASZdAbG", "tag 0x04 admin vault withdrawal (2 transfers)"},
		{"GoonFi", "28JTWSqBCq3qpQjH7KFYy3bvFVoDy7WxPEs6txXJh4697B2paoVKMoYtyb8HoQy5VZQ1LbhMRcpE58Dw1zVsGQGt", "tag 0x08 quote update"},
	}
	parser := dexparser.NewDexParser()
	for _, tc := range cases {
		t.Run(tc.venue+"/"+tc.what, func(t *testing.T) {
			tx := loadFixture(t, tc.sig)
			if trades := venueTrades(t, tc.venue, tx); len(trades) != 0 {
				t.Errorf("parser returned %d trades", len(trades))
			}
			program := venuePrograms[tc.venue].program
			for _, tr := range parser.ParseAll(tx, nil).Trades {
				if tr.ProgramId == program.ID || tr.AMM == program.Name {
					t.Errorf("ParseAll trade %s by %s", tr.Idx, tr.AMM)
				}
			}
		})
	}
}

// The 1-byte tags collide across programs: a venue parser only reads its own
// program's instructions.
func TestVenuesDispatchByProgramId(t *testing.T) {
	cases := []struct {
		venue, other, sig string
	}{
		{"ZeroFi", "TesseraV", "29b7yFfvur5vt9VQkvu1aPDZFUgastzdXpqQYAZ41XYgvW25UDz48EYMTZEHxtWQ3P8SsZ9SSALedVzTcGLvbo96"}, // 0x10, 18 bytes
		{"Quantum", "SolFiV2", "4C2p65nuttUqBv5VhLNXHq6cNEaexT5i4YGbGSH2hodRGLBQyJBzRCkr2VXPKPNof8Lut8WSm51tkxVv3mvUkpuZ"}, // 0x07, 18 bytes
		{"SolFi", "SolFiV2", "4C2p65nuttUqBv5VhLNXHq6cNEaexT5i4YGbGSH2hodRGLBQyJBzRCkr2VXPKPNof8Lut8WSm51tkxVv3mvUkpuZ"},
		{"Scorch", "BisonFi", "22VFxreHs9cEvHoYiooU62iLsK76vJo6E1WGXsjFRcM7PwCrUUhNiBhvUMztPCdxCgyfcDCB79TUaRDEC2VPLYAF"}, // 0x02
	}
	for _, tc := range cases {
		t.Run(tc.venue+"<-"+tc.other, func(t *testing.T) {
			ctx := newParseContext(loadFixture(t, tc.sig), nil)
			instructions := ctx.Classifier.GetInstructions(venuePrograms[tc.other].program.ID)
			if len(instructions) == 0 {
				t.Fatalf("fixture has no %s instruction", tc.other)
			}
			p := venuePrograms[tc.venue].parser(ctx.Adapter, ctx.DexInfo, ctx.TransferActions, instructions)
			if trades := p.ProcessTrades(); len(trades) != 0 {
				t.Errorf("%s parser read %d %s instructions as trades", tc.venue, len(trades), tc.other)
			}
		})
	}
}

// innerInstructionMap returns inner instruction outer-inner of a fixture as
// its decoded JSON object
func innerInstructionMap(t *testing.T, tx *adapter.SolanaTransaction, outer, inner int) map[string]interface{} {
	t.Helper()
	for _, set := range tx.Meta.InnerInstructions {
		if set.Index == outer {
			m, ok := set.Instructions[inner].(map[string]interface{})
			if !ok {
				t.Fatalf("inner instruction %d-%d is %T", outer, inner, set.Instructions[inner])
			}
			return m
		}
	}
	t.Fatalf("no inner instructions for %d", outer)
	return nil
}

// meme-10 / constants-15: a HumidiFi instruction whose transfers are missing
// is not a trade (the old parser built one with empty mints and 9 decimals
// from the accounts). Synthetic: built from the real swap 3dhDZRhp... 2-4 by
// removing its two transfers (2-5, 2-6); no real transaction of this shape
// exists.
func TestVenuesHumidiFiNoTransfersNoTrade(t *testing.T) {
	tx := loadFixture(t, "3dhDZRhpSNK5Naku3R9CKERXP4MXEYecwkBTpN4SqtJZcDCDsh3ob4J2jdMkAbNyT6KmkamqsBomTEQNFHrLkZ9u")
	for i, set := range tx.Meta.InnerInstructions {
		if set.Index == 2 {
			ixs := append([]interface{}{}, set.Instructions[:5]...)
			tx.Meta.InnerInstructions[i].Instructions = append(ixs, set.Instructions[7:]...)
		}
	}
	for _, tr := range venueTrades(t, "HumidiFi", tx) {
		t.Errorf("trade %s: %s %s -> %s %s", tr.Idx, tr.InputToken.AmountRaw, tr.InputToken.Mint, tr.OutputToken.AmountRaw, tr.OutputToken.Mint)
	}
}

// meme-10: the 25-byte swap carries its direction as a u64 0/1 after
// deobfuscation; other values are not swaps. Synthetic: the real swap
// 3dhDZRhp... 2-4 with the direction field set to 2.
func TestVenuesHumidiFiDirectionGate(t *testing.T) {
	tx := loadFixture(t, "3dhDZRhpSNK5Naku3R9CKERXP4MXEYecwkBTpN4SqtJZcDCDsh3ob4J2jdMkAbNyT6KmkamqsBomTEQNFHrLkZ9u")
	ix := innerInstructionMap(t, tx, 2, 4)
	data, err := base58.Decode(ix["data"].(string))
	if err != nil || len(data) != 25 {
		t.Fatalf("data %d bytes, %v", len(data), err)
	}
	data[16] ^= 2 // deobfuscated direction 0 -> 2
	ix["data"] = base58.Encode(data)
	if trades := venueTrades(t, "HumidiFi", tx); len(trades) != 0 {
		t.Errorf("got %d trades", len(trades))
	}
}

// constants-7: only Obric's Anchor swap and swap2 are swaps. The legacy
// SWAP_X_TO_Y value is Raydium CPMM's swap_base_input and never occurred in
// an Obric transaction. Synthetic: the real Obric swap 2DaS55Tw... 3-1 with
// its discriminator replaced by SWAP_X_TO_Y.
func TestVenuesObricOnlyAnchorSwaps(t *testing.T) {
	tx := loadFixture(t, "2DaS55TwMuryadgsirCLRJywZ7rbBoLugXnicbJR2YFSSgn8PAvGN8Mhr9p4Ux8SEtgPzrRRuqV9T7XS7WWtdxvR")
	ix := innerInstructionMap(t, tx, 3, 1)
	data, err := base58.Decode(ix["data"].(string))
	if err != nil {
		t.Fatal(err)
	}
	copy(data, constants.DISCRIMINATORS.OBRIC.SWAP_X_TO_Y)
	ix["data"] = base58.Encode(data)
	for _, tr := range venueTrades(t, "ObricV2", tx) {
		t.Errorf("trade %s from a SWAP_X_TO_Y instruction", tr.Idx)
	}
}

// The venue parsers read "jsonParsed" transactions like "json" ones
// (NeF1UiWX...: BisonFi hop at 3-8 of a Jupiter route).
func TestVenuesJSONParsedEncoding(t *testing.T) {
	const sig = "NeF1UiWXKUbuswNNw14gJ2uup7yrV6KyXijQ7dsjLYanj9dnsBSeqGPDVZ3wP3NfhXQ84rJncRo5XwbrbSdWVWW"
	fromJSON := venueTrades(t, "BisonFi", loadFixture(t, sig))
	fromParsed := venueTrades(t, "BisonFi", loadParsedFixture(t, sig))
	if len(fromJSON) != 1 || len(fromParsed) != 1 {
		t.Fatalf("json: %d trades, jsonParsed: %d trades, want 1 each", len(fromJSON), len(fromParsed))
	}
	a, b := fromJSON[0], fromParsed[0]
	if a.Idx != "3-8" || a.Idx != b.Idx || a.Pool[0] != b.Pool[0] ||
		a.InputToken.Mint != b.InputToken.Mint || a.InputToken.AmountRaw != b.InputToken.AmountRaw ||
		a.OutputToken.Mint != b.OutputToken.Mint || a.OutputToken.AmountRaw != b.OutputToken.AmountRaw {
		t.Errorf("json %s %s %s -> %s %s, jsonParsed %s %s %s -> %s %s",
			a.Idx, a.InputToken.AmountRaw, a.InputToken.Mint, a.OutputToken.AmountRaw, a.OutputToken.Mint,
			b.Idx, b.InputToken.AmountRaw, b.InputToken.Mint, b.OutputToken.AmountRaw, b.OutputToken.Mint)
	}
}

// Transactions from before stack heights were recorded: the CPI group of a
// swap then ends at the next instruction of the same program. Synthetic: real
// swaps with every stackHeight removed.
func TestVenuesWithoutStackHeights(t *testing.T) {
	for _, tc := range venueSwapCases {
		switch tc.sig[:8] {
		case "4C2p65nu", "2Tp9YKLW", "3tHnRiFw", "5PrZGSv2", "2kCAkYGn":
		default:
			continue
		}
		t.Run(tc.venue+"/"+tc.sig[:8]+"/"+tc.idx, func(t *testing.T) {
			tx := loadFixture(t, tc.sig)
			for _, set := range tx.Meta.InnerInstructions {
				for _, ix := range set.Instructions {
					delete(ix.(map[string]interface{}), "stackHeight")
				}
			}
			n := 0
			for _, tr := range venueTrades(t, tc.venue, tx) {
				if tr.Idx != tc.idx {
					continue
				}
				n++
				if tr.InputToken.AmountRaw != tc.inAmount || tr.OutputToken.AmountRaw != tc.outAmount {
					t.Errorf("%s -> %s, want %s -> %s", tr.InputToken.AmountRaw, tr.OutputToken.AmountRaw, tc.inAmount, tc.outAmount)
				}
			}
			if n != 1 {
				t.Errorf("want 1 trade at %s, got %d", tc.idx, n)
			}
		})
	}
}
