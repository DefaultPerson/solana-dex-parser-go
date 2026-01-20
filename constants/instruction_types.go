package constants

// SPL Token instruction types
const (
	SPLTokenInitializeMint    = 0
	SPLTokenInitializeAccount = 1
	SPLTokenInitializeMultisig = 2
	SPLTokenTransfer          = 3
	SPLTokenApprove           = 4
	SPLTokenRevoke            = 5
	SPLTokenSetAuthority      = 6
	SPLTokenMintTo            = 7
	SPLTokenBurn              = 8
	SPLTokenCloseAccount      = 9
	SPLTokenFreezeAccount     = 10
	SPLTokenThawAccount       = 11
	SPLTokenTransferChecked   = 12
	SPLTokenApproveChecked    = 13
	SPLTokenMintToChecked     = 14
	SPLTokenBurnChecked       = 15
)

// System instruction types
const (
	SystemCreateAccount            = 0
	SystemAssign                   = 1
	SystemTransfer                 = 2
	SystemCreateAccountWithSeed    = 3
	SystemAdvanceNonceAccount      = 4
	SystemWithdrawNonceAccount     = 5
	SystemInitializeNonceAccount   = 6
	SystemAuthorizeNonceAccount    = 7
	SystemAllocate                 = 8
	SystemAllocateWithSeed         = 9
	SystemAssignWithSeed           = 10
	SystemTransferWithSeed         = 11
	SystemUpgradeNonceAccount      = 12
	SystemCreateAccountWithSeedChecked = 13
	SystemCreateIdempotent         = 14
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
