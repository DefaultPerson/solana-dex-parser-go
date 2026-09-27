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
	JUPITER                DexProgram
	JUPITER_V2             DexProgram
	JUPITER_V4             DexProgram
	JUPITER_DCA            DexProgram
	JUPITER_DCA_KEEPER1    DexProgram
	JUPITER_DCA_KEEPER2    DexProgram
	JUPITER_DCA_KEEPER3    DexProgram
	JUPITER_LIMIT_ORDER    DexProgram
	JUPITER_LIMIT_ORDER_V2 DexProgram
	JUPITER_VA             DexProgram
	OKX_DEX                DexProgram
	OKX_ROUTER             DexProgram
	RAYDIUM_ROUTE          DexProgram
	SANCTUM                DexProgram
	PHOTON                 DexProgram

	// Major DEX Protocols
	RAYDIUM_V4      DexProgram
	RAYDIUM_AMM     DexProgram
	RAYDIUM_CPMM    DexProgram
	RAYDIUM_CL      DexProgram
	RAYDIUM_LCP     DexProgram
	ORCA            DexProgram
	ORCA_V2         DexProgram
	ORCA_V1         DexProgram
	PHOENIX         DexProgram
	OPENBOOK        DexProgram
	METEORA         DexProgram
	METEORA_DAMM    DexProgram
	METEORA_DAMM_V2 DexProgram
	METEORA_DBC     DexProgram
	SERUM_V3        DexProgram

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
	GOONFI   DexProgram
	OBRIC_V2 DexProgram
	HUMIDIFI DexProgram

	// Additional Aggregators
	DFLOW DexProgram

	// Added 2026-09 (audit). IDs and names from the Jupiter program-id-to-label list
	// (lite-api.jup.ag/swap/v1/program-id-to-label, fetched 2026-09-27) unless noted;
	// every ID was confirmed executable with getMultipleAccounts.
	SOLFI_V2   DexProgram
	GOONFI_V2  DexProgram
	BISONFI    DexProgram
	TESSERA_V  DexProgram
	ALPHAQ     DexProgram
	SCORCH     DexProgram
	QUANTUM    DexProgram
	MANIFEST   DexProgram
	BYREAL     DexProgram
	SAROS_DLMM DexProgram
	TITAN      DexProgram
	OKX_DEX_V2 DexProgram
	JUPITER_Z  DexProgram
	GMGN       DexProgram
	// Scorch swap program (SCoRcH8c...); SCORCH (ojh19oja..., the Jupiter-labelled ID) is
	// its pricing program, invoked by CPI without token transfers of its own
	SCORCH_SWAP DexProgram
}{
	// DEX Aggregators
	JUPITER: DexProgram{
		ID:   "JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4",
		Name: "Jupiter",
		Tags: []string{"route"},
	},
	// Legacy: last transaction referencing the program 2025-09-23 (checked 2026-09-27)
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
	// Legacy: last transaction referencing the program 2026-07-28 (checked 2026-09-27)
	JUPITER_VA: DexProgram{
		ID:   "VALaaymxQh2mNy2trH9jUqHT1mTow76wpTcGmSWSwJe",
		Name: "JupiterVA",
		Tags: []string{"route"},
	},
	// Legacy: last transaction referencing the program 2026-08-02 (checked 2026-09-27); see OKX_DEX_V2
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
	// Legacy: last transaction referencing the program 2026-08-07 (checked 2026-09-27)
	NOVA: DexProgram{
		ID:   "NoVA1TmDUqksaj2hB1nayFkPysjJbFiU76dT4qPw2wm",
		Name: "Nova",
		Tags: []string{"bot"},
	},
	// Legacy: last transaction referencing the program 2026-07-30 (checked 2026-09-27)
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
	// Legacy: last transaction referencing the program 2026-08-30 (checked 2026-09-27)
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
	// Legacy: not in the Jupiter label list fetched 2026-09-27; last transaction referencing the program 2026-07-28
	LIFINITY: DexProgram{
		ID:   "EewxydAPCCVuNEyrVN68PuSYdQ7wKn27V9Gjeoi8dy3S",
		Name: "Lifinity",
		Tags: []string{"amm"},
	},
	// Legacy: not in the Jupiter label list fetched 2026-09-27
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
	// Near-idle: 5 transactions between 2026-09-17 and 2026-09-26 (audit, checked 2026-09-27)
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
	// Legacy (GoonFi V1): last transaction referencing the program 2026-08-02 (checked 2026-09-27); see GOONFI_V2
	GOONFI: DexProgram{
		ID:   "goonERTdGsjnkZqWuVjs73BZ3Pb9qoCUdBUL17BnS5j",
		Name: "GoonFi",
		Tags: []string{"amm"},
	},
	// Legacy: last swap found 2024-11-27; later transactions only reference the program or run admin instructions (checked 2026-09-27)
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

	// Added 2026-09 (audit)
	SOLFI_V2: DexProgram{
		ID:   "SV2EYYJyRz2YhfXwXnhNAevDEui5Q6yrfyo13WtupPF",
		Name: "SolFiV2",
		Tags: []string{"amm"},
	},
	GOONFI_V2: DexProgram{
		ID:   "goonuddtQRrWqqn5nFyczVKaie28f3kDkHWkHtURSLE",
		Name: "GoonFiV2",
		Tags: []string{"amm"},
	},
	BISONFI: DexProgram{
		ID:   "BiSoNHVpsVZW2F7rx2eQ59yQwKxzU5NvBcmKshCSUypi",
		Name: "BisonFi",
		Tags: []string{"amm"},
	},
	TESSERA_V: DexProgram{
		ID:   "TessVdML9pBGgG9yGks7o4HewRaXVAMuoVj4x83GLQH",
		Name: "TesseraV",
		Tags: []string{"amm"},
	},
	ALPHAQ: DexProgram{
		ID:   "ALPHAQmeA7bjrVuccPsYPiCvsi428SNwte66Srvs4pHA",
		Name: "AlphaQ",
		Tags: []string{"amm"},
	},
	SCORCH: DexProgram{
		ID:   "ojh19ojaKduoJZuaJADhcVGp4xt1TcdAvZmpVsCorch",
		Name: "Scorch",
		Tags: []string{"amm"},
	},
	QUANTUM: DexProgram{
		ID:   "QuaNtZsgYRe5Z9Bk4LZ4cTD9tbkVoyCNf1R2BN9bBDv",
		Name: "Quantum",
		Tags: []string{"amm"},
	},
	MANIFEST: DexProgram{
		ID:   "MNFSTqtC93rEfYHB6hF82sKdZpUDFWkViLByLd1k1Ms",
		Name: "Manifest",
		Tags: []string{"amm"},
	},
	BYREAL: DexProgram{
		ID:   "REALQqNEomY6cQGZJUGwywTBD2UmDT32rZcNnfxQ5N2",
		Name: "Byreal",
		Tags: []string{"amm"},
	},
	SAROS_DLMM: DexProgram{
		ID:   "1qbkdrr3z4ryLA7pZykqxvxWPoeifcVKo6ZG9CfkvVE",
		Name: "SarosDLMM",
		Tags: []string{"amm"},
	},
	// Titan aggregator (not in the Jupiter label list; seen as the outer router of
	// SolFi V2 swaps, e.g. 4C2p65nu...)
	TITAN: DexProgram{
		ID:   "T1TANpTeScyeqVzzgNViGDNrkQ6qHz9KrSBS4aNXvGT",
		Name: "Titan",
		Tags: []string{"route"},
	},
	// OKX DEX Router V2 (on-chain IDL metadata name "OKX: DEX Router")
	OKX_DEX_V2: DexProgram{
		ID:   "proVF4pMXVaYqmy4NjniPh4pqKNfMmsihgd4wdkCX3u",
		Name: "OKXV2",
		Tags: []string{"route"},
	},
	// Jupiter Z (RFQ order_engine, instruction "fill")
	JUPITER_Z: DexProgram{
		ID:   "61DFfeTKM7trxYcPQCM78bJ794ddZprZpAwAnLiwTpYH",
		Name: "JupiterZ",
		Tags: []string{"route"},
	},
	// GMGN trading bot router, invoked by GMGN fee wallet 7sHXjs1j... transactions
	GMGN: DexProgram{
		ID:   "GMgnVFR8Jb39LoXsEVzb3DvBy3ywCmdmJquHUy1Lrkqb",
		Name: "GMGN",
		Tags: []string{"bot"},
	},
	SCORCH_SWAP: DexProgram{
		ID:   "SCoRcH8c2dpjvcJD6FiPbCSQyQgu3PcUAWj2Xxx3mqn",
		Name: "Scorch",
		Tags: []string{"amm"},
	},
}

// JUPITER_LABEL_PROGRAMS lists the remaining venues from the Jupiter
// program-id-to-label list (fetched 2026-09-27) that have no named entry in
// DEX_PROGRAMS. Jupiter routes swaps through all of them, so they are tagged
// "amm". Names are the Jupiter labels. Every ID was confirmed executable.
var JUPITER_LABEL_PROGRAMS = []DexProgram{
	{ID: "AQU1FRd7papthgdrwPTTq5JacJh8YtwEXaBfKU3bTz45", Name: "Aquifer", Tags: []string{"amm"}},
	{ID: "Archer8kgiavM61GyusMzaaS2ft5sALtNsD1HxkUPMhy", Name: "Archer", Tags: []string{"amm"}},
	{ID: "B72M6nyCLFgWiJtAN4naUTminMiTmyGcEqQHXwVeRdht", Name: "BinaryFi", Tags: []string{"amm"}},
	{ID: "2DNbzPochEcyCcWMbL4d9S3u9QqQEj5bbe6cSZFvKsbh", Name: "BisonFi Predict", Tags: []string{"amm"}},
	{ID: "BSwp6bEBihVLdqJRKGgzjcGLHkcTuzmSo1TQkHepzH8p", Name: "Bonkswap", Tags: []string{"amm"}},
	{ID: "CarrotwivhMpDnm27EHmRLeQ683Z1PufuqEmBZvD282s", Name: "Carrot", Tags: []string{"amm"}},
	{ID: "H8W3ctz92svYg6mkn1UtGfu2aQr2fnUFHM1RhScEtQDt", Name: "Cropper", Tags: []string{"amm"}},
	{ID: "fUSioN9YKKSa3CUC2YUc4tPkHJ5Y6XW1yz8y6F7qWz9", Name: "DefiTuna", Tags: []string{"amm"}},
	{ID: "DNL1tgEj3nJovHw9jtyCCQD3arssCJzkmpDizknwzey4", Name: "Denali", Tags: []string{"amm"}},
	{ID: "DRVSpZ2YUYYKgZP8XtLhAGtT1zYSCKzeHfb4DgRnrgqD", Name: "Deriverse", Tags: []string{"amm"}},
	{ID: "DSwpgjMvXhtGn6BsbqmacdBZyfLj6jSWf3HJpdJtmg6N", Name: "DexLab", Tags: []string{"amm"}},
	{ID: "FLiNTXPwppyoJabCoxc2uiiRygAHpmMXajiDXo2Ub1z", Name: "Flint", Tags: []string{"amm"}},
	{ID: "FLUX6xBayGxLX9UcimVRxXFMHH6q43mAbRvDzSpCsvfK", Name: "Flux", Tags: []string{"amm"}},
	{ID: "FLUXubRmkEi2q6K3Y9kBPg9248ggaZVsoSFhtJHSrm1X", Name: "FluxBeam", Tags: []string{"amm"}},
	{ID: "gatorLx9aC1e5ZWAXscv5QRKiLXnLPLXjftVc81h1Hr", Name: "GatorSwap", Tags: []string{"amm"}},
	{ID: "srAMMzfVHVAtgSJc8iH6CfKzuWuUTzLHVCE81QU1rgi", Name: "Gavel", Tags: []string{"amm"}},
	{ID: "Gswppe6ERWKpUTXvRPfXdzHhiCyJvLadVvXGfdpBqcE1", Name: "Guacswap", Tags: []string{"amm"}},
	{ID: "HADRoNbLovyqhCsocfYQYB7QdfCAAinN9HTePvBCVDQ8", Name: "Hadron", Tags: []string{"amm"}},
	{ID: "treaf4wWBBty3fHdyBpo35Mz84M8k3heKXmjmi9vFt5", Name: "Helium Network", Tags: []string{"amm"}},
	{ID: "HumaXepHnjaRCpjYTokxY4UtaJcmx41prQ8cxGmFC5fn", Name: "Huma", Tags: []string{"amm"}},
	{ID: "HYEXCHtHkBagdStcJCp3xbbb9B7sdMdWXFNj6mdsG4hn", Name: "Hylo Exchange", Tags: []string{"amm"}},
	{ID: "HyaB3W9q6XdA5xwpU4XnSZV94htfmbmqJXZcEbRaJutt", Name: "Invariant", Tags: []string{"amm"}},
	{ID: "jup3YeL8QhtSx1e253b2FDvsMNC87fDrgQZivbrndc9", Name: "Jupiter Lend Earn", Tags: []string{"amm"}},
	{ID: "fd3nMFYTQjX1yr5ER8u7tPdHJB7qt8RpDpNtLQX2Br5", Name: "JupiterRfqV2", Tags: []string{"amm"}},
	{ID: "jupZ4m2GqUCJ5iueMfzQf8khFfH31d4XAQt3RzCT9Vd", Name: "JupLend AMM", Tags: []string{"amm"}},
	{ID: "3TK9D8aoBFYjYZtKCjciPrVrRStsnvo7KmpcJqDavpaU", Name: "Kipseli", Tags: []string{"amm"}},
	{ID: "BQEJZUB4CzoT6UhRffoCkqCyqQNrCPCSGHcPEmsdbEsX", Name: "LemmingsFi", Tags: []string{"amm"}},
	{ID: "MSwapi3WhNKMUGm9YrxGhypgUEt7wYQH3ZgG32XoWzH", Name: "M Swap", Tags: []string{"amm"}},
	{ID: "FUTARELBfJfQ8RDGhg1wdhddq1odMAJUePHFuBYfUxKq", Name: "MetaDAO", Tags: []string{"amm"}},
	{ID: "Bvs46DPFxiFE6YHxLDLD6QAUcmy51FyRVPZJusPxLk3j", Name: "Metric", Tags: []string{"amm"}},
	{ID: "HBVw6bZtcCaezhcBrmfyXBSBRWCdv72271xQ4GPvms2z", Name: "Obsidian", Tags: []string{"amm"}},
	{ID: "omnixgS8fnqHfCcTGKWj6JtKjzpJZ1Y5y9pyFkQDkYE", Name: "Omnipair", Tags: []string{"amm"}},
	{ID: "HpNfyc2Saw7RKkQd8nEL4khUcuPhQ7WwY1B2qjx8jxFq", Name: "PancakeSwap", Tags: []string{"amm"}},
	{ID: "PSwapMdSai8tjrEXcxFeQth87xC4rRsa4VA5mhGhXkP", Name: "Penguin", Tags: []string{"amm"}},
	{ID: "NUMERUNsFCP3kuNmWZuXtm1AaQCPj9uw6Guv2Ekoi5P", Name: "Perena", Tags: []string{"amm"}},
	{ID: "save8RQVPMWNTzU18t3GBvBkN9hT7jsGjiCQ28FpD9H", Name: "Perena Star V2", Tags: []string{"amm"}},
	{ID: "PERPHjGBqRHArX4DySjwM6UJHiR3sWAatqfdBS2qQJu", Name: "Perps", Tags: []string{"amm"}},
	{ID: "QUayE6nexQWYNZAEqfN8FxoNwQDSu3CAzT2qq9J1ArG", Name: "Quay", Tags: []string{"amm"}},
	{ID: "riptK81hDxhe5pW5jSzSM9iRA8azgEgLJ4dXkPtBS7j", Name: "Riptide", Tags: []string{"amm"}},
	{ID: "runnrXXdsSRkdueCRYxKDvSWfv6nAnrG5dcM29qj1HA", Name: "RunnerRodeo", Tags: []string{"amm"}},
	{ID: "DecZY86MU5Gj7kppfUCEmd4LbXXuyZH1yHaP2NTqdiZB", Name: "Saber (Decimals)", Tags: []string{"amm"}},
	{ID: "SP12tWFxD9oJsVWNavTTBZvMbA6gkAmxtVgxdqvyvhY", Name: "Sanctum", Tags: []string{"amm"}},
	{ID: "SPMBzsVUuoHA4Jm6KunbsotaahvVikZs1JyTW6iJvbn", Name: "Sanctum", Tags: []string{"amm"}},
	{ID: "SPoo1Ku8WFXoNDMHPsrGSTSG1Y47rzgn41SLUNakuHy", Name: "Sanctum", Tags: []string{"amm"}},
	{ID: "5ocnV1qiCgaQR8Jb8xWnVbApfaygJ8tNoZfgPwsgx9kx", Name: "Sanctum Infinity", Tags: []string{"amm"}},
	{ID: "pegVkBpfR9GFi5Jaa9YXAHACyM9CzCNvhc5bf8GWNyW", Name: "Sanctum Prop S", Tags: []string{"amm"}},
	{ID: "so1f7APRw5pJ5iNJrM9g5X9tQgXzk8kMUnXxDNdt99b", Name: "Sanctum SOLS", Tags: []string{"amm"}},
	{ID: "SCALEwAvEK5gtkdHiFzXfPgtk2YwJxPDzaV3aDmR7tA", Name: "Scale Amm", Tags: []string{"amm"}},
	{ID: "SCALEWoRSpVZpMRqHEcDfNvBh3nUSe34jDr9r689gLa", Name: "Scale Vmm", Tags: []string{"amm"}},
	{ID: "endoLNCKTqDn8gSVnN2hDdpgACUPWHZTwoYnnMybpAT", Name: "Solayer", Tags: []string{"amm"}},
	{ID: "6dMXqGZ3ga2dikrYS9ovDXgHGh5RUsb2RTUj6hrQXhk6", Name: "Stabble CLMM", Tags: []string{"amm"}},
	{ID: "ghosty4ZU1Qk1HN7Ymz4pZ15QfspzJZgSYFkdKN6ZLK", Name: "Stableswap", Tags: []string{"amm"}},
	{ID: "Dooar9JkhdZ7J3LHN3A7YCuoGRUggXhQaG4kijfLGU2j", Name: "StepN", Tags: []string{"amm"}},
	{ID: "9VX8EKBg6vM6tA68xaDsPkbrx26XConZjkQmhVApUptc", Name: "TaurusFi", Tags: []string{"amm"}},
	{ID: "SwaPpA9LAaLfeLi3a68M4DjnLqgtticKg6CnyNwgAC8", Name: "Token Swap", Tags: []string{"amm"}},
	{ID: "8ZoPDsvLWXthQaZz6aJJEWNFBZKk2ePuygi6LHT7F7Hh", Name: "Trench", Tags: []string{"amm"}},
	{ID: "CURVEmPpijXDTNdqrA9PGP1io2rkgiVXH26xdXVGLLfz", Name: "Trends", Tags: []string{"amm"}},
	{ID: "2rU1oCHtQ7WJUvy15tKtFvxdYNNSc3id7AzUcjeFSddo", Name: "VaultLiquidUnstake", Tags: []string{"amm"}},
	{ID: "5U3EU2ubXtK84QcRjWVmYt9RaDyA8gKxdUrPFXmZyaki", Name: "Virtuals", Tags: []string{"amm"}},
	{ID: "vVoLTRjQmtFpiYoegx285Ze4gsLJ8ZxgFKVcuvmG1a8", Name: "Voltr", Tags: []string{"amm"}},
	{ID: "FW6zUqn4iKRaeopwwhwsquTY6ABWLLgjxtrC3VPnaWBf", Name: "WhaleStreet", Tags: []string{"amm"}},
	{ID: "WooFif76YGRNjk1pA8wCsN67aQsD9f9iLsz4NcJ1AVb", Name: "Woofi", Tags: []string{"amm"}},
	{ID: "StaKE6XNKVVhG8Qu9hDJBqCW3eRe7MDGLz17nJZetLT", Name: "XOrca", Tags: []string{"amm"}},
}

// DEX_PROGRAM_IDS is a list of all DEX program IDs
// Wallet entries (JUPITER_DCA_KEEPER1-3, OKX_ROUTER) are not programs and are listed in
// KNOWN_AUTHORITIES instead.
var DEX_PROGRAM_IDS = append([]string{
	DEX_PROGRAMS.JUPITER.ID,
	DEX_PROGRAMS.JUPITER_V2.ID,
	DEX_PROGRAMS.JUPITER_V4.ID,
	DEX_PROGRAMS.JUPITER_DCA.ID,
	DEX_PROGRAMS.JUPITER_LIMIT_ORDER.ID,
	DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID,
	DEX_PROGRAMS.JUPITER_VA.ID,
	DEX_PROGRAMS.OKX_DEX.ID,
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
	DEX_PROGRAMS.SOLFI_V2.ID,
	DEX_PROGRAMS.GOONFI_V2.ID,
	DEX_PROGRAMS.BISONFI.ID,
	DEX_PROGRAMS.TESSERA_V.ID,
	DEX_PROGRAMS.ALPHAQ.ID,
	DEX_PROGRAMS.SCORCH.ID,
	DEX_PROGRAMS.QUANTUM.ID,
	DEX_PROGRAMS.MANIFEST.ID,
	DEX_PROGRAMS.BYREAL.ID,
	DEX_PROGRAMS.SAROS_DLMM.ID,
	DEX_PROGRAMS.TITAN.ID,
	DEX_PROGRAMS.OKX_DEX_V2.ID,
	DEX_PROGRAMS.JUPITER_Z.ID,
	DEX_PROGRAMS.GMGN.ID,
	DEX_PROGRAMS.SCORCH_SWAP.ID,
}, jupiterLabelProgramIDs()...)

func jupiterLabelProgramIDs() []string {
	ids := make([]string, 0, len(JUPITER_LABEL_PROGRAMS))
	for _, p := range JUPITER_LABEL_PROGRAMS {
		ids = append(ids, p.ID)
	}
	return ids
}

// KNOWN_AUTHORITIES contains wallets (system-owned, not executable) that act as
// authorities in DEX flows. They never appear as program IDs, so they are not DEX programs.
var KNOWN_AUTHORITIES = []string{
	DEX_PROGRAMS.JUPITER_DCA_KEEPER1.ID,
	DEX_PROGRAMS.JUPITER_DCA_KEEPER2.ID,
	DEX_PROGRAMS.JUPITER_DCA_KEEPER3.ID,
	DEX_PROGRAMS.OKX_ROUTER.ID,
}

// SYSTEM_PROGRAMS contains system program IDs that should be ignored
var SYSTEM_PROGRAMS = []string{
	"ComputeBudget111111111111111111111111111111",
	"11111111111111111111111111111111",
	"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
	"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb",
	"ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL",
	"srmqPvymJeFKQ4zGQed1GFppgkRHL9kaELCbyksJtPX", // OpenBook (Serum fork used by Raydium V4 pools), as upstream TS
}

// SKIP_PROGRAM_IDS contains program IDs that should be skipped
var SKIP_PROGRAM_IDS = []string{
	"pfeeUxB6jkeY1Hxd7CsFCAjcbHA9rWtchMGdZ6VojVZ", // Pumpswap Fee
	"MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr", // SPL Memo (CPI'd by Whirlpool *_v2 before transfers)
	"Memo1UhkJRfHyvLMcVucJwxXeuD728EqVDDwQDxFMNo", // SPL Memo v1
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

	// Pumpfun (39azUY... is the migrator / withdraw_authority, see PUMPFUN_MIGRATORS)
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

	// Pump.fun Global (4wTV1Ymi...) and PumpSwap GlobalConfig (ADyA8hde...), decoded on-chain
	// 2026-09-27: reserved (mayhem mode) fee recipients, identical in both accounts
	"GesfTA3X2arioaHp8bbKdjG9vJtskViWACZoYvxp4twS",
	"4budycTjhs9fD6xw62VBducVTNgMgJJ5BgtKq7mAZwn6",
	"8SBKzEQU4nLSzcwF4a74F2iaUDQyTfjGndn6qUWBnrpR",
	"4UQeTP1T39KZ9Sfxzo3WR5skgsaP6NZa87BAkuazLEKH",
	"8sNeir4QsLsJdYpc9RZacohhK1Y5FLU3nC5LXgYB4aa6",
	"Fh9HmeLNUMVCvejxCtCL2DbYaRyBFVJ5xrWkLnMH6fdk",
	"463MEnMeGyJekNZFQSTUABBEbLnvMTALbT6ZmsxAbAdq",
	"6AUH3WEHucYZyC61hqpqYUWVto5qA5hjHuNQ32GNnNxA",
	// buyback fee recipients, identical in both accounts
	"5YxQFdt3Tr9zJLvkFccqXVUwhdTWJQc1fFg2YPbxvxeD",
	"9M4giFFMxmFGXtc3feFzRai56WbBqehoSeRE5GK7gf7",
	"GXPFM2caqTtQYC2cJ5yJRi9VDkpsYZXzYdwYpGnLmtDL",
	"3BpXnfJaUTiwXnJNe7Ej1rcbzqTTQUvLShZaWazebsVR",
	"5cjcW9wExnJJiqgLjq7DEG75Pm6JBgE1hNv4B2vHXUW6",
	"EHAAiTxcdDwQ3U4bU6YcMsQGaekdzLS3B5SmYo46kJtL",
	"5eHhjP8JaYkz83CWwvGU2uMUXefd3AazWGx4gpcuEEYD",
	"A7hAgCzFw14fejgCp387JUJRMNyz4j89JKnhtKU8piqW",
}

// dexProgramMap is a map for quick lookup of DEX programs by ID
var dexProgramMap map[string]DexProgram

func init() {
	dexProgramMap = make(map[string]DexProgram)
	for _, p := range JUPITER_LABEL_PROGRAMS {
		dexProgramMap[p.ID] = p
	}
	for _, id := range DEX_PROGRAM_IDS {
		if p := namedDexProgram(id); p.ID != "" {
			dexProgramMap[id] = p
		}
	}
}

// GetDexProgramByID returns the DEX program configuration for a given ID,
// or an empty DexProgram if the ID is not a known DEX program.
func GetDexProgramByID(id string) DexProgram {
	return dexProgramMap[id]
}

// namedDexProgram returns the DEX_PROGRAMS entry for an ID in DEX_PROGRAM_IDS
func namedDexProgram(id string) DexProgram {
	switch id {
	case DEX_PROGRAMS.JUPITER.ID:
		return DEX_PROGRAMS.JUPITER
	case DEX_PROGRAMS.JUPITER_V2.ID:
		return DEX_PROGRAMS.JUPITER_V2
	case DEX_PROGRAMS.JUPITER_V4.ID:
		return DEX_PROGRAMS.JUPITER_V4
	case DEX_PROGRAMS.JUPITER_DCA.ID:
		return DEX_PROGRAMS.JUPITER_DCA
	case DEX_PROGRAMS.JUPITER_LIMIT_ORDER.ID:
		return DEX_PROGRAMS.JUPITER_LIMIT_ORDER
	case DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID:
		return DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2
	case DEX_PROGRAMS.JUPITER_VA.ID:
		return DEX_PROGRAMS.JUPITER_VA
	case DEX_PROGRAMS.OKX_DEX.ID:
		return DEX_PROGRAMS.OKX_DEX
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
	case DEX_PROGRAMS.SOLFI_V2.ID:
		return DEX_PROGRAMS.SOLFI_V2
	case DEX_PROGRAMS.GOONFI_V2.ID:
		return DEX_PROGRAMS.GOONFI_V2
	case DEX_PROGRAMS.BISONFI.ID:
		return DEX_PROGRAMS.BISONFI
	case DEX_PROGRAMS.TESSERA_V.ID:
		return DEX_PROGRAMS.TESSERA_V
	case DEX_PROGRAMS.ALPHAQ.ID:
		return DEX_PROGRAMS.ALPHAQ
	case DEX_PROGRAMS.SCORCH.ID:
		return DEX_PROGRAMS.SCORCH
	case DEX_PROGRAMS.QUANTUM.ID:
		return DEX_PROGRAMS.QUANTUM
	case DEX_PROGRAMS.MANIFEST.ID:
		return DEX_PROGRAMS.MANIFEST
	case DEX_PROGRAMS.BYREAL.ID:
		return DEX_PROGRAMS.BYREAL
	case DEX_PROGRAMS.SAROS_DLMM.ID:
		return DEX_PROGRAMS.SAROS_DLMM
	case DEX_PROGRAMS.TITAN.ID:
		return DEX_PROGRAMS.TITAN
	case DEX_PROGRAMS.OKX_DEX_V2.ID:
		return DEX_PROGRAMS.OKX_DEX_V2
	case DEX_PROGRAMS.JUPITER_Z.ID:
		return DEX_PROGRAMS.JUPITER_Z
	case DEX_PROGRAMS.GMGN.ID:
		return DEX_PROGRAMS.GMGN
	case DEX_PROGRAMS.SCORCH_SWAP.ID:
		return DEX_PROGRAMS.SCORCH_SWAP
	default:
		return DexProgram{}
	}
}

// GetProgramName returns the human-readable name for a program ID,
// or "Unknown" if the ID is not a known DEX program (same as upstream TS).
func GetProgramName(programId string) string {
	if prog, ok := dexProgramMap[programId]; ok {
		return prog.Name
	}
	return "Unknown"
}

// IsDexProgram checks if a program ID is a known DEX program
func IsDexProgram(programId string) bool {
	_, ok := dexProgramMap[programId]
	return ok
}

// IsVaultProgram checks if a program ID is a known program tagged "vault"
func IsVaultProgram(programId string) bool {
	for _, tag := range dexProgramMap[programId].Tags {
		if tag == "vault" {
			return true
		}
	}
	return false
}

// IsKnownAuthority checks if an account is a wallet listed in KNOWN_AUTHORITIES
func IsKnownAuthority(account string) bool {
	for _, a := range KNOWN_AUTHORITIES {
		if a == account {
			return true
		}
	}
	return false
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
