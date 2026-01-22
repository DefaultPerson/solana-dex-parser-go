package constants

// DexProgram represents a DEX program configuration
type DexProgram struct {
	ID   string   // Program ID
	Name string   // Human-readable name
	Tags []string // Tags: "route", "amm", "bot", "vault"
}

// DEX_PROGRAMS contains all supported DEX program configurations
var DEX_PROGRAMS = struct {
	// DEX Aggregators
	JUPITER              DexProgram
	JUPITER_V2           DexProgram
	JUPITER_V4           DexProgram
	JUPITER_DCA          DexProgram
	JUPITER_DCA_KEEPER1  DexProgram
	JUPITER_DCA_KEEPER2  DexProgram
	JUPITER_DCA_KEEPER3  DexProgram
	JUPITER_LIMIT_ORDER  DexProgram
	JUPITER_LIMIT_ORDER_V2 DexProgram
	JUPITER_VA           DexProgram
	OKX_DEX              DexProgram
	OKX_ROUTER           DexProgram
	RAYDIUM_ROUTE        DexProgram
	SANCTUM              DexProgram
	PHOTON               DexProgram

	// Major DEX Protocols
	RAYDIUM_V4    DexProgram
	RAYDIUM_AMM   DexProgram
	RAYDIUM_CPMM  DexProgram
	RAYDIUM_CL    DexProgram
	RAYDIUM_LCP   DexProgram
	ORCA          DexProgram
	ORCA_V2       DexProgram
	ORCA_V1       DexProgram
	PHOENIX       DexProgram
	OPENBOOK      DexProgram
	METEORA       DexProgram
	METEORA_DAMM  DexProgram
	METEORA_DAMM_V2 DexProgram
	METEORA_DBC   DexProgram
	SERUM_V3      DexProgram

	// Vault Programs
	METEORA_VAULT DexProgram
	STABBEL_VAULT DexProgram

	// Trading Bot Programs
	BANANA_GUN DexProgram
	MINTECH    DexProgram
	BLOOM      DexProgram
	MAESTRO    DexProgram
	NOVA       DexProgram
	APEPRO     DexProgram

	// Other DEX Protocols
	ALDRIN         DexProgram
	ALDRIN_V2      DexProgram
	CREMA          DexProgram
	GOOSEFX        DexProgram
	LIFINITY       DexProgram
	LIFINITY_V2    DexProgram
	MERCURIAL      DexProgram
	MOONIT         DexProgram
	ONEDEX         DexProgram
	PUMP_FUN       DexProgram
	PUMP_SWAP      DexProgram
	SABER          DexProgram
	SAROS          DexProgram
	SOLFI          DexProgram
	STABBEL        DexProgram
	STABBEL_WEIGHT DexProgram
	BOOP_FUN       DexProgram
	ZERO_FI        DexProgram
	SUGAR          DexProgram
	HEAVEN         DexProgram
	HEAVEN_VAULT   DexProgram

	// Prop AMM Protocols (Dark Pools)
	GOONFI    DexProgram
	OBRIC_V2  DexProgram
	HUMIDIFI  DexProgram

	// Additional Aggregators
	DFLOW DexProgram
}{
	// DEX Aggregators
	JUPITER: DexProgram{
		ID:   "JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4",
		Name: "Jupiter",
		Tags: []string{"route"},
	},
	JUPITER_V2: DexProgram{
		ID:   "JUP2jxvXaqu7NQY1GmNF4m1vodw12LVXYxbFL2uJvfo",
		Name: "JupiterV2",
		Tags: []string{"route"},
	},
	JUPITER_V4: DexProgram{
		ID:   "JUP4Fb2cqiRUcaTHdrPC8h2gNsA2ETXiPDD33WcGuJB",
		Name: "JupiterV4",
		Tags: []string{"route"},
	},
	JUPITER_DCA: DexProgram{
		ID:   "DCA265Vj8a9CEuX1eb1LWRnDT7uK6q1xMipnNyatn23M",
		Name: "JupiterDCA",
		Tags: []string{"route"},
	},
	JUPITER_DCA_KEEPER1: DexProgram{
		ID:   "DCAKxn5PFNN1mBREPWGdk1RXg5aVH9rPErLfBFEi2Emb",
		Name: "JupiterDcaKeeper1",
		Tags: []string{"route"},
	},
	JUPITER_DCA_KEEPER2: DexProgram{
		ID:   "DCAKuApAuZtVNYLk3KTAVW9GLWVvPbnb5CxxRRmVgcTr",
		Name: "JupiterDcaKeeper2",
		Tags: []string{"route"},
	},
	JUPITER_DCA_KEEPER3: DexProgram{
		ID:   "DCAK36VfExkPdAkYUQg6ewgxyinvcEyPLyHjRbmveKFw",
		Name: "JupiterDcaKeeper3",
		Tags: []string{"route"},
	},
	JUPITER_LIMIT_ORDER: DexProgram{
		ID:   "jupoNjAxXgZ4rjzxzPMP4oxduvQsQtZzyknqvzYNrNu",
		Name: "JupiterLimit",
		Tags: []string{"route"},
	},
	JUPITER_LIMIT_ORDER_V2: DexProgram{
		ID:   "j1o2qRpjcyUwEvwtcfhEQefh773ZgjxcVRry7LDqg5X",
		Name: "JupiterLimitV2",
		Tags: []string{"route"},
	},
	JUPITER_VA: DexProgram{
		ID:   "VALaaymxQh2mNy2trH9jUqHT1mTow76wpTcGmSWSwJe",
		Name: "JupiterVA",
		Tags: []string{"route"},
	},
	OKX_DEX: DexProgram{
		ID:   "6m2CDdhRgxpH4WjvdzxAYbGxwdGUz5MziiL5jek2kBma",
		Name: "OKX",
		Tags: []string{"route"},
	},
	OKX_ROUTER: DexProgram{
		ID:   "HV1KXxWFaSeriyFvXyx48FqG9BoFbfinB8njCJonqP7K",
		Name: "OKXRouter",
		Tags: []string{"route"},
	},
	RAYDIUM_ROUTE: DexProgram{
		ID:   "routeUGWgWzqBWFcrCfv8tritsqukccJPu3q5GPP3xS",
		Name: "RaydiumRoute",
		Tags: []string{"route"},
	},
	SANCTUM: DexProgram{
		ID:   "stkitrT1Uoy18Dk1fTrgPw8W6MVzoCfYoAFT4MLsmhq",
		Name: "Sanctum",
		Tags: []string{"route"},
	},
	PHOTON: DexProgram{
		ID:   "BSfD6SHZigAfDWSjzD5Q41jw8LmKwtmjskPH9XW1mrRW",
		Name: "Photon",
		Tags: []string{"route"},
	},

	// Major DEX Protocols
	RAYDIUM_V4: DexProgram{
		ID:   "675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8",
		Name: "RaydiumV4",
		Tags: []string{"amm"},
	},
	RAYDIUM_AMM: DexProgram{
		ID:   "5quBtoiQqxF9Jv6KYKctB59NT3gtJD2Y65kdnB1Uev3h",
		Name: "RaydiumAMM",
		Tags: []string{"amm"},
	},
	RAYDIUM_CPMM: DexProgram{
		ID:   "CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C",
		Name: "RaydiumCPMM",
		Tags: []string{"amm"},
	},
	RAYDIUM_CL: DexProgram{
		ID:   "CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK",
		Name: "RaydiumCL",
		Tags: []string{"amm"},
	},
	RAYDIUM_LCP: DexProgram{
		ID:   "LanMV9sAd7wArD4vJFi2qDdfnVhFxYSUg6eADduJ3uj",
		Name: "RaydiumLaunchpad",
		Tags: []string{"amm"},
	},
	ORCA: DexProgram{
		ID:   "whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc",
		Name: "Orca",
		Tags: []string{"amm"},
	},
	ORCA_V2: DexProgram{
		ID:   "9W959DqEETiGZocYWCQPaJ6sBmUzgfxXfqGeTEdp3aQP",
		Name: "OrcaV2",
		Tags: []string{"amm"},
	},
	ORCA_V1: DexProgram{
		ID:   "DjVE6JNiYqPL2QXyCUUh8rNjHrbz9hXHNYt99MQ59qw1",
		Name: "OrcaV1",
		Tags: []string{"amm"},
	},
	PHOENIX: DexProgram{
		ID:   "PhoeNiXZ8ByJGLkxNfZRnkUfjvmuYqLR89jjFHGqdXY",
		Name: "Phoenix",
		Tags: []string{"route", "amm"},
	},
	OPENBOOK: DexProgram{
		ID:   "opnb2LAfJYbRMAHHvqjCwQxanZn7ReEHp1k81EohpZb",
		Name: "Openbook",
		Tags: []string{"amm"},
	},
	METEORA: DexProgram{
		ID:   "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo",
		Name: "MeteoraDLMM",
		Tags: []string{"amm"},
	},
	METEORA_DAMM: DexProgram{
		ID:   "Eo7WjKq67rjJQSZxS6z3YkapzY3eMj6Xy8X5EQVn5UaB",
		Name: "MeteoraDamm",
		Tags: []string{"amm"},
	},
	METEORA_DAMM_V2: DexProgram{
		ID:   "cpamdpZCGKUy5JxQXB4dcpGPiikHawvSWAd6mEn1sGG",
		Name: "MeteoraDammV2",
		Tags: []string{"amm"},
	},
	METEORA_DBC: DexProgram{
		ID:   "dbcij3LWUppWqq96dh6gJWwBifmcGfLSB5D4DuSMaqN",
		Name: "MeteoraDBC",
		Tags: []string{"amm"},
	},
	SERUM_V3: DexProgram{
		ID:   "9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin",
		Name: "SerumV3",
		Tags: []string{"amm", "vault"},
	},

	// Vault Programs
	METEORA_VAULT: DexProgram{
		ID:   "24Uqj9JCLxUeoC3hGfh5W3s9FM9uCHDS2SG3LYwBpyTi",
		Name: "MeteoraVault",
		Tags: []string{"vault"},
	},
	STABBEL_VAULT: DexProgram{
		ID:   "vo1tWgqZMjG61Z2T9qUaMYKqZ75CYzMuaZ2LZP1n7HV",
		Name: "StabbleVault",
		Tags: []string{"vault"},
	},

	// Trading Bot Programs
	BANANA_GUN: DexProgram{
		ID:   "BANANAjs7FJiPQqJTGFzkZJndT9o7UmKiYYGaJz6frGu",
		Name: "BananaGun",
		Tags: []string{"bot"},
	},
	MINTECH: DexProgram{
		ID:   "minTcHYRLVPubRK8nt6sqe2ZpWrGDLQoNLipDJCGocY",
		Name: "Mintech",
		Tags: []string{"bot"},
	},
	BLOOM: DexProgram{
		ID:   "b1oomGGqPKGD6errbyfbVMBuzSC8WtAAYo8MwNafWW1",
		Name: "Bloom",
		Tags: []string{"bot"},
	},
	MAESTRO: DexProgram{
		ID:   "MaestroAAe9ge5HTc64VbBQZ6fP77pwvrhM8i1XWSAx",
		Name: "Maestro",
		Tags: []string{"bot"},
	},
	NOVA: DexProgram{
		ID:   "NoVA1TmDUqksaj2hB1nayFkPysjJbFiU76dT4qPw2wm",
		Name: "Nova",
		Tags: []string{"bot"},
	},
	APEPRO: DexProgram{
		ID:   "JSW99DKmxNyREQM14SQLDykeBvEUG63TeohrvmofEiw",
		Name: "Apepro",
		Tags: []string{"bot"},
	},

	// Other DEX Protocols
	ALDRIN: DexProgram{
		ID:   "AMM55ShdkoGRB5jVYPjWziwk8m5MpwyDgsMWHaMSQWH6",
		Name: "Aldrin",
		Tags: []string{"amm"},
	},
	ALDRIN_V2: DexProgram{
		ID:   "CURVGoZn8zycx6FXwwevgBTB2gVvdbGTEpvMJDbgs2t4",
		Name: "Aldrin V2",
		Tags: []string{"amm"},
	},
	CREMA: DexProgram{
		ID:   "CLMM9tUoggJu2wagPkkqs9eFG4BWhVBZWkP1qv3Sp7tR",
		Name: "Crema",
		Tags: []string{"amm"},
	},
	GOOSEFX: DexProgram{
		ID:   "GAMMA7meSFWaBXF25oSUgmGRwaW6sCMFLmBNiMSdbHVT",
		Name: "GooseFX GAMMA",
		Tags: []string{"amm"},
	},
	LIFINITY: DexProgram{
		ID:   "EewxydAPCCVuNEyrVN68PuSYdQ7wKn27V9Gjeoi8dy3S",
		Name: "Lifinity",
		Tags: []string{"amm"},
	},
	LIFINITY_V2: DexProgram{
		ID:   "2wT8Yq49kHgDzXuPxZSaeLaH1qbmGXtEyPy64bL7aD3c",
		Name: "LifinityV2",
		Tags: []string{"amm"},
	},
	MERCURIAL: DexProgram{
		ID:   "MERLuDFBMmsHnsBPZw2sDQZHvXFMwp8EdjudcU2HKky",
		Name: "Mercurial",
		Tags: []string{"amm"},
	},
	MOONIT: DexProgram{
		ID:   "MoonCVVNZFSYkqNXP6bxHLPL6QQJiMagDL3qcqUQTrG",
		Name: "Moonit",
		Tags: []string{"amm"},
	},
	ONEDEX: DexProgram{
		ID:   "DEXYosS6oEGvk8uCDayvwEZz4qEyDJRf9nFgYCaqPMTm",
		Name: "1Dex",
		Tags: []string{"amm"},
	},
	PUMP_FUN: DexProgram{
		ID:   "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P",
		Name: "Pumpfun",
		Tags: []string{"amm"},
	},
	PUMP_SWAP: DexProgram{
		ID:   "pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA",
		Name: "Pumpswap",
		Tags: []string{"amm"},
	},
	SABER: DexProgram{
		ID:   "SSwpkEEcbUqx4vtoEByFjSkhKdCT862DNVb52nZg1UZ",
		Name: "Saber",
		Tags: []string{"amm"},
	},
	SAROS: DexProgram{
		ID:   "SSwapUtytfBdBn1b9NUGG6foMVPtcWgpRU32HToDUZr",
		Name: "Saros",
		Tags: []string{"amm"},
	},
	SOLFI: DexProgram{
		ID:   "SoLFiHG9TfgtdUXUjWAxi3LtvYuFyDLVhBWxdMZxyCe",
		Name: "SolFi",
		Tags: []string{"amm"},
	},
	STABBEL: DexProgram{
		ID:   "swapNyd8XiQwJ6ianp9snpu4brUqFxadzvHebnAXjJZ",
		Name: "Stabble",
		Tags: []string{"amm"},
	},
	STABBEL_WEIGHT: DexProgram{
		ID:   "swapFpHZwjELNnjvThjajtiVmkz3yPQEHjLtka2fwHW",
		Name: "StabbleWeight",
		Tags: []string{"amm"},
	},
	BOOP_FUN: DexProgram{
		ID:   "boop8hVGQGqehUK2iVEMEnMrL5RbjywRzHKBmBE7ry4",
		Name: "Boopfun",
		Tags: []string{"amm"},
	},
	ZERO_FI: DexProgram{
		ID:   "ZERor4xhbUycZ6gb9ntrhqscUcZmAbQDjEAtCf4hbZY",
		Name: "ZeroFi",
		Tags: []string{"amm"},
	},
	SUGAR: DexProgram{
		ID:   "deus4Bvftd5QKcEkE5muQaWGWDoma8GrySvPFrBPjhS",
		Name: "Sugar",
		Tags: []string{"amm"},
	},
	HEAVEN: DexProgram{
		ID:   "HEAVENoP2qxoeuF8Dj2oT1GHEnu49U5mJYkdeC8BAX2o",
		Name: "Heaven",
		Tags: []string{"amm"},
	},
	HEAVEN_VAULT: DexProgram{
		ID:   "HEvSKofvBgfaexv23kMabbYqxasxU3mQ4ibBMEmJWHny",
		Name: "HeavenStore",
		Tags: []string{"vault"},
	},

	// Prop AMM Protocols (Dark Pools)
	GOONFI: DexProgram{
		ID:   "goonERTdGsjnkZqWuVjs73BZ3Pb9qoCUdBUL17BnS5j",
		Name: "GoonFi",
		Tags: []string{"amm"},
	},
	OBRIC_V2: DexProgram{
		ID:   "obriQD1zbpyLz95G5n7nJe6a4DPjpFwa5XYPoNm113y",
		Name: "ObricV2",
		Tags: []string{"amm"},
	},
	HUMIDIFI: DexProgram{
		ID:   "9H6tua7jkLhdm3w8BvgpTn5LZNU7g4ZynDmCiNN3q6Rp",
		Name: "HumidiFi",
		Tags: []string{"amm"},
	},

	// Additional Aggregators
	DFLOW: DexProgram{
		ID:   "DF1ow4tspfHX9JwWJsAb9epbkA8hmpSEAtxXy1V27QBH",
		Name: "DFlow",
		Tags: []string{"route"},
	},
}

// DEX_PROGRAM_IDS is a list of all DEX program IDs
var DEX_PROGRAM_IDS = []string{
	DEX_PROGRAMS.JUPITER.ID,
	DEX_PROGRAMS.JUPITER_V2.ID,
	DEX_PROGRAMS.JUPITER_V4.ID,
	DEX_PROGRAMS.JUPITER_DCA.ID,
	DEX_PROGRAMS.JUPITER_DCA_KEEPER1.ID,
	DEX_PROGRAMS.JUPITER_DCA_KEEPER2.ID,
	DEX_PROGRAMS.JUPITER_DCA_KEEPER3.ID,
	DEX_PROGRAMS.JUPITER_LIMIT_ORDER.ID,
	DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID,
	DEX_PROGRAMS.JUPITER_VA.ID,
	DEX_PROGRAMS.OKX_DEX.ID,
	DEX_PROGRAMS.OKX_ROUTER.ID,
	DEX_PROGRAMS.RAYDIUM_ROUTE.ID,
	DEX_PROGRAMS.SANCTUM.ID,
	DEX_PROGRAMS.PHOTON.ID,
	DEX_PROGRAMS.RAYDIUM_V4.ID,
	DEX_PROGRAMS.RAYDIUM_AMM.ID,
	DEX_PROGRAMS.RAYDIUM_CPMM.ID,
	DEX_PROGRAMS.RAYDIUM_CL.ID,
	DEX_PROGRAMS.RAYDIUM_LCP.ID,
	DEX_PROGRAMS.ORCA.ID,
	DEX_PROGRAMS.ORCA_V2.ID,
	DEX_PROGRAMS.ORCA_V1.ID,
	DEX_PROGRAMS.PHOENIX.ID,
	DEX_PROGRAMS.OPENBOOK.ID,
	DEX_PROGRAMS.METEORA.ID,
	DEX_PROGRAMS.METEORA_DAMM.ID,
	DEX_PROGRAMS.METEORA_DAMM_V2.ID,
	DEX_PROGRAMS.METEORA_DBC.ID,
	DEX_PROGRAMS.SERUM_V3.ID,
	DEX_PROGRAMS.METEORA_VAULT.ID,
	DEX_PROGRAMS.STABBEL_VAULT.ID,
	DEX_PROGRAMS.BANANA_GUN.ID,
	DEX_PROGRAMS.MINTECH.ID,
	DEX_PROGRAMS.BLOOM.ID,
	DEX_PROGRAMS.MAESTRO.ID,
	DEX_PROGRAMS.NOVA.ID,
	DEX_PROGRAMS.APEPRO.ID,
	DEX_PROGRAMS.ALDRIN.ID,
	DEX_PROGRAMS.ALDRIN_V2.ID,
	DEX_PROGRAMS.CREMA.ID,
	DEX_PROGRAMS.GOOSEFX.ID,
	DEX_PROGRAMS.LIFINITY.ID,
	DEX_PROGRAMS.LIFINITY_V2.ID,
	DEX_PROGRAMS.MERCURIAL.ID,
	DEX_PROGRAMS.MOONIT.ID,
	DEX_PROGRAMS.ONEDEX.ID,
	DEX_PROGRAMS.PUMP_FUN.ID,
	DEX_PROGRAMS.PUMP_SWAP.ID,
	DEX_PROGRAMS.SABER.ID,
	DEX_PROGRAMS.SAROS.ID,
	DEX_PROGRAMS.SOLFI.ID,
	DEX_PROGRAMS.STABBEL.ID,
	DEX_PROGRAMS.STABBEL_WEIGHT.ID,
	DEX_PROGRAMS.BOOP_FUN.ID,
	DEX_PROGRAMS.ZERO_FI.ID,
	DEX_PROGRAMS.SUGAR.ID,
	DEX_PROGRAMS.HEAVEN.ID,
	DEX_PROGRAMS.HEAVEN_VAULT.ID,
	DEX_PROGRAMS.GOONFI.ID,
	DEX_PROGRAMS.OBRIC_V2.ID,
	DEX_PROGRAMS.HUMIDIFI.ID,
	DEX_PROGRAMS.DFLOW.ID,
}

// SYSTEM_PROGRAMS contains system program IDs that should be ignored
var SYSTEM_PROGRAMS = []string{
	"ComputeBudget111111111111111111111111111111",
	"11111111111111111111111111111111",
	"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
	"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb",
	"ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL",
}

// SKIP_PROGRAM_IDS contains program IDs that should be skipped
var SKIP_PROGRAM_IDS = []string{
	"pfeeUxB6jkeY1Hxd7CsFCAjcbHA9rWtchMGdZ6VojVZ", // Pumpswap Fee
}

// Token program constants
const (
	// System program ID (SOL transfers)
	SYSTEM_PROGRAM_ID = "11111111111111111111111111111111"
	// SPL Token program ID
	TOKEN_PROGRAM_ID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	// SPL Token 2022 program ID
	TOKEN_2022_PROGRAM_ID = "TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb"
	// Associated Token Account program ID
	ASSOCIATED_TOKEN_PROGRAM_ID = "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"
	// Metaplex Token Metadata program ID
	METAPLEX_PROGRAM_ID = "metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s"
	// Address Lookup Table program
	ALT_PROGRAM_ID = "AddressLookupTab1e1111111111111111111111111"
)

// PUMPFUN_MIGRATORS contains Pumpfun migrator addresses
var PUMPFUN_MIGRATORS = []string{
	"39azUYFWPz3VHgKCf3VChUwbpURdCHRxjWVowf5jUJjg",
}

// FEE_ACCOUNTS contains known fee account addresses
var FEE_ACCOUNTS = []string{
	// Jitotip accounts
	"96gYZGLnJYVFmbjzopPSU6QiEV5fGqZNyN9nmNhvrZU5",
	"HFqU5x63VTqvQss8hp11i4wVV8bD44PvwucfZ2bU7gRe",
	"Cw8CFyM9FkoMi7K7Crf6HNQqf4uEMzpKw6QNghXLvLkY",
	"ADaUMid9yfUytqMBgopwjb2DTLSokTSzL1zt6iGPaS49",
	"DfXygSm4jCyNCybVYYK6DwvWqjKee8pbDmJGcLWNDXjh",
	"ADuUkR4vqLUMWXxW9gh6D6L8pMSawimctcNZ5pGwDcEt",
	"DttWaMuVvTiduZRnguLF7jNxTgiMBZ1hyAumKUiL2KRL",
	"3AVi9Tg9Uo68tJfuvoKvqKNWKkC5wPdSSdeBnizKZ6jT",

	// Jupiter Partner Referral Fee Vault
	"45ruCyfdRkWpRNGEqWzjCiXRHkZs8WXCLQ67Pnpye7Hp",

	// Pumpfun
	"39azUYFWPz3VHgKCf3VChUwbpURdCHRxjWVowf5jUJjg",
	"FWsW1xNtWscwNmKv6wVsU1iTzRN6wmmk3MjxRP5tT7hz",
	"G5UZAVbAf46s7cKWoyKu8kYTip9DGTpbLZ2qa9Aq69dP",
	"7hTckgnGnLQR6sdH7YkqFTAA7VwTfYFaZ6EhEsU3saCX",
	"9rPYyANsfQZw3DnDmKE3YCQF5E8oD89UXoHn9JFEhJUz",
	"7VtfL8fvgNfhz17qKRMjzQEXgbdpnHHHQRh54R9jP2RJ",
	"AVmoTthdrX6tKt4nDjco2D775W2YK3sDhxPcMmzUAmTY",
	"62qc2CNXwrYqQScmEdiZFFAnJR262PxWEuNQtxfafNgV",
	"JCRGumoE9Qi5BBgULTgdgTLjSgkCMSbF62ZZfGs84JeU",
	"CebN5WGQ4jvEPvsVU4EoHEpgzq1VV7AbicfhtW4xC9iM",

	// Photon Fee Vault
	"AVUCZyuT35YSuj4RH7fwiyPu82Djn2Hfg7y2ND2XcnZH",

	// BonkSwap Fee
	"BUX7s2ef2htTGb2KKoPHWkmzxPj4nTWMWRgs5CSbQxf9",

	// Meteora Fee Vault
	"CdQTNULjDiTsvyR5UKjYBMqWvYpxXj6HY4m6atm2hErk",
}

// dexProgramMap is a map for quick lookup of DEX programs by ID
var dexProgramMap map[string]DexProgram

func init() {
	dexProgramMap = make(map[string]DexProgram)
	for _, id := range DEX_PROGRAM_IDS {
		dexProgramMap[id] = GetDexProgramByID(id)
	}
}

// GetDexProgramByID returns the DEX program configuration for a given ID
func GetDexProgramByID(id string) DexProgram {
	switch id {
	case DEX_PROGRAMS.JUPITER.ID:
		return DEX_PROGRAMS.JUPITER
	case DEX_PROGRAMS.JUPITER_V2.ID:
		return DEX_PROGRAMS.JUPITER_V2
	case DEX_PROGRAMS.JUPITER_V4.ID:
		return DEX_PROGRAMS.JUPITER_V4
	case DEX_PROGRAMS.JUPITER_DCA.ID:
		return DEX_PROGRAMS.JUPITER_DCA
	case DEX_PROGRAMS.JUPITER_DCA_KEEPER1.ID:
		return DEX_PROGRAMS.JUPITER_DCA_KEEPER1
	case DEX_PROGRAMS.JUPITER_DCA_KEEPER2.ID:
		return DEX_PROGRAMS.JUPITER_DCA_KEEPER2
	case DEX_PROGRAMS.JUPITER_DCA_KEEPER3.ID:
		return DEX_PROGRAMS.JUPITER_DCA_KEEPER3
	case DEX_PROGRAMS.JUPITER_LIMIT_ORDER.ID:
		return DEX_PROGRAMS.JUPITER_LIMIT_ORDER
	case DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID:
		return DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2
	case DEX_PROGRAMS.JUPITER_VA.ID:
		return DEX_PROGRAMS.JUPITER_VA
	case DEX_PROGRAMS.OKX_DEX.ID:
		return DEX_PROGRAMS.OKX_DEX
	case DEX_PROGRAMS.OKX_ROUTER.ID:
		return DEX_PROGRAMS.OKX_ROUTER
	case DEX_PROGRAMS.RAYDIUM_ROUTE.ID:
		return DEX_PROGRAMS.RAYDIUM_ROUTE
	case DEX_PROGRAMS.SANCTUM.ID:
		return DEX_PROGRAMS.SANCTUM
	case DEX_PROGRAMS.PHOTON.ID:
		return DEX_PROGRAMS.PHOTON
	case DEX_PROGRAMS.RAYDIUM_V4.ID:
		return DEX_PROGRAMS.RAYDIUM_V4
	case DEX_PROGRAMS.RAYDIUM_AMM.ID:
		return DEX_PROGRAMS.RAYDIUM_AMM
	case DEX_PROGRAMS.RAYDIUM_CPMM.ID:
		return DEX_PROGRAMS.RAYDIUM_CPMM
	case DEX_PROGRAMS.RAYDIUM_CL.ID:
		return DEX_PROGRAMS.RAYDIUM_CL
	case DEX_PROGRAMS.RAYDIUM_LCP.ID:
		return DEX_PROGRAMS.RAYDIUM_LCP
	case DEX_PROGRAMS.ORCA.ID:
		return DEX_PROGRAMS.ORCA
	case DEX_PROGRAMS.ORCA_V2.ID:
		return DEX_PROGRAMS.ORCA_V2
	case DEX_PROGRAMS.ORCA_V1.ID:
		return DEX_PROGRAMS.ORCA_V1
	case DEX_PROGRAMS.PHOENIX.ID:
		return DEX_PROGRAMS.PHOENIX
	case DEX_PROGRAMS.OPENBOOK.ID:
		return DEX_PROGRAMS.OPENBOOK
	case DEX_PROGRAMS.METEORA.ID:
		return DEX_PROGRAMS.METEORA
	case DEX_PROGRAMS.METEORA_DAMM.ID:
		return DEX_PROGRAMS.METEORA_DAMM
	case DEX_PROGRAMS.METEORA_DAMM_V2.ID:
		return DEX_PROGRAMS.METEORA_DAMM_V2
	case DEX_PROGRAMS.METEORA_DBC.ID:
		return DEX_PROGRAMS.METEORA_DBC
	case DEX_PROGRAMS.SERUM_V3.ID:
		return DEX_PROGRAMS.SERUM_V3
	case DEX_PROGRAMS.METEORA_VAULT.ID:
		return DEX_PROGRAMS.METEORA_VAULT
	case DEX_PROGRAMS.STABBEL_VAULT.ID:
		return DEX_PROGRAMS.STABBEL_VAULT
	case DEX_PROGRAMS.BANANA_GUN.ID:
		return DEX_PROGRAMS.BANANA_GUN
	case DEX_PROGRAMS.MINTECH.ID:
		return DEX_PROGRAMS.MINTECH
	case DEX_PROGRAMS.BLOOM.ID:
		return DEX_PROGRAMS.BLOOM
	case DEX_PROGRAMS.MAESTRO.ID:
		return DEX_PROGRAMS.MAESTRO
	case DEX_PROGRAMS.NOVA.ID:
		return DEX_PROGRAMS.NOVA
	case DEX_PROGRAMS.APEPRO.ID:
		return DEX_PROGRAMS.APEPRO
	case DEX_PROGRAMS.ALDRIN.ID:
		return DEX_PROGRAMS.ALDRIN
	case DEX_PROGRAMS.ALDRIN_V2.ID:
		return DEX_PROGRAMS.ALDRIN_V2
	case DEX_PROGRAMS.CREMA.ID:
		return DEX_PROGRAMS.CREMA
	case DEX_PROGRAMS.GOOSEFX.ID:
		return DEX_PROGRAMS.GOOSEFX
	case DEX_PROGRAMS.LIFINITY.ID:
		return DEX_PROGRAMS.LIFINITY
	case DEX_PROGRAMS.LIFINITY_V2.ID:
		return DEX_PROGRAMS.LIFINITY_V2
	case DEX_PROGRAMS.MERCURIAL.ID:
		return DEX_PROGRAMS.MERCURIAL
	case DEX_PROGRAMS.MOONIT.ID:
		return DEX_PROGRAMS.MOONIT
	case DEX_PROGRAMS.ONEDEX.ID:
		return DEX_PROGRAMS.ONEDEX
	case DEX_PROGRAMS.PUMP_FUN.ID:
		return DEX_PROGRAMS.PUMP_FUN
	case DEX_PROGRAMS.PUMP_SWAP.ID:
		return DEX_PROGRAMS.PUMP_SWAP
	case DEX_PROGRAMS.SABER.ID:
		return DEX_PROGRAMS.SABER
	case DEX_PROGRAMS.SAROS.ID:
		return DEX_PROGRAMS.SAROS
	case DEX_PROGRAMS.SOLFI.ID:
		return DEX_PROGRAMS.SOLFI
	case DEX_PROGRAMS.STABBEL.ID:
		return DEX_PROGRAMS.STABBEL
	case DEX_PROGRAMS.STABBEL_WEIGHT.ID:
		return DEX_PROGRAMS.STABBEL_WEIGHT
	case DEX_PROGRAMS.BOOP_FUN.ID:
		return DEX_PROGRAMS.BOOP_FUN
	case DEX_PROGRAMS.ZERO_FI.ID:
		return DEX_PROGRAMS.ZERO_FI
	case DEX_PROGRAMS.SUGAR.ID:
		return DEX_PROGRAMS.SUGAR
	case DEX_PROGRAMS.HEAVEN.ID:
		return DEX_PROGRAMS.HEAVEN
	case DEX_PROGRAMS.HEAVEN_VAULT.ID:
		return DEX_PROGRAMS.HEAVEN_VAULT
	case DEX_PROGRAMS.GOONFI.ID:
		return DEX_PROGRAMS.GOONFI
	case DEX_PROGRAMS.OBRIC_V2.ID:
		return DEX_PROGRAMS.OBRIC_V2
	case DEX_PROGRAMS.HUMIDIFI.ID:
		return DEX_PROGRAMS.HUMIDIFI
	case DEX_PROGRAMS.DFLOW.ID:
		return DEX_PROGRAMS.DFLOW
	default:
		return DexProgram{}
	}
}

// GetProgramName returns the human-readable name for a program ID
func GetProgramName(programId string) string {
	if prog, ok := dexProgramMap[programId]; ok {
		return prog.Name
	}
	return ""
}

// IsDexProgram checks if a program ID is a known DEX program
func IsDexProgram(programId string) bool {
	_, ok := dexProgramMap[programId]
	return ok
}

// IsSystemProgram checks if a program ID is a system program
func IsSystemProgram(programId string) bool {
	for _, p := range SYSTEM_PROGRAMS {
		if p == programId {
			return true
		}
	}
	return false
}

// IsFeeAccount checks if an account is a known fee account
func IsFeeAccount(account string) bool {
	for _, a := range FEE_ACCOUNTS {
		if a == account {
			return true
		}
	}
	return false
}
