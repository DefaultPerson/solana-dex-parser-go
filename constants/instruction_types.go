package constants

// SPL Token instruction types
const (
	SPLTokenInitializeMint           = 0
	SPLTokenInitializeAccount        = 1
	SPLTokenInitializeMultisig       = 2
	SPLTokenTransfer                 = 3
	SPLTokenApprove                  = 4
	SPLTokenRevoke                   = 5
	SPLTokenSetAuthority             = 6
	SPLTokenMintTo                   = 7
	SPLTokenBurn                     = 8
	SPLTokenCloseAccount             = 9
	SPLTokenFreezeAccount            = 10
	SPLTokenThawAccount              = 11
	SPLTokenTransferChecked          = 12
	SPLTokenApproveChecked           = 13
	SPLTokenMintToChecked            = 14
	SPLTokenBurnChecked              = 15
	SPLTokenInitializeAccount2       = 16
	SPLTokenSyncNative               = 17
	SPLTokenInitializeAccount3       = 18
	SPLTokenInitializeMultisig2      = 19
	SPLTokenInitializeMint2          = 20
	SPLTokenGetAccountDataSize       = 21
	SPLTokenInitializeImmutableOwner = 22
	SPLTokenAmountToUiAmount         = 23
	SPLTokenUiAmountToAmount         = 24
	// Token-2022 extensions
	SPLTokenInitializeMintCloseAuthority = 25
	SPLTokenTransferFeeExtension         = 26
)

// Token-2022 TransferFeeExtension (tag 26) sub-instruction types
const (
	TransferFeeInitializeConfig                = 0
	TransferFeeTransferCheckedWithFee          = 1
	TransferFeeWithdrawWithheldTokensFromMint  = 2
	TransferFeeWithdrawWithheldTokensFromAccts = 3
	TransferFeeHarvestWithheldTokensToMint     = 4
	TransferFeeSetTransferFee                  = 5
)

// System instruction types
const (
	SystemCreateAccount             = 0
	SystemAssign                    = 1
	SystemTransfer                  = 2
	SystemCreateAccountWithSeed     = 3
	SystemAdvanceNonceAccount       = 4
	SystemWithdrawNonceAccount      = 5
	SystemInitializeNonceAccount    = 6
	SystemAuthorizeNonceAccount     = 7
	SystemAllocate                  = 8
	SystemAllocateWithSeed          = 9
	SystemAssignWithSeed            = 10
	SystemTransferWithSeed          = 11
	SystemUpgradeNonceAccount       = 12
	SystemCreateAccountAllowPrefund = 13

	// Deprecated: the System program has no such instruction; tag 13 is
	// CreateAccountAllowPrefund. Kept for compatibility.
	SystemCreateAccountWithSeedChecked = 13
	// Deprecated: CreateIdempotent is Associated Token Account instruction 1
	// (AssociatedTokenCreateIdempotent), not a System instruction. Kept for compatibility.
	SystemCreateIdempotent = 14
)

// Associated Token Account program instruction types
const (
	AssociatedTokenCreate           = 0
	AssociatedTokenCreateIdempotent = 1
	AssociatedTokenRecoverNested    = 2
)

// IsSPLTransferInstruction checks if the instruction type is a transfer
func IsSPLTransferInstruction(instructionType uint8) bool {
	return instructionType == SPLTokenTransfer || instructionType == SPLTokenTransferChecked
}

// IsSPLMintInstruction checks if the instruction type is a mint operation
func IsSPLMintInstruction(instructionType uint8) bool {
	return instructionType == SPLTokenMintTo || instructionType == SPLTokenMintToChecked
}

// IsSPLBurnInstruction checks if the instruction type is a burn operation
func IsSPLBurnInstruction(instructionType uint8) bool {
	return instructionType == SPLTokenBurn || instructionType == SPLTokenBurnChecked
}
