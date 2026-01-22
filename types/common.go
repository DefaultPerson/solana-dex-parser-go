package types

// ClassifiedInstruction represents a classified instruction with its context information
type ClassifiedInstruction struct {
	// Instruction is the raw instruction data
	Instruction interface{} `json:"instruction"`

	// ProgramId is the program ID that owns this instruction
	ProgramId string `json:"programId"`

	// OuterIndex is the outer instruction index in the transaction
	OuterIndex int `json:"outerIndex"`

	// InnerIndex is the inner instruction index (for CPI calls), -1 if not inner
	InnerIndex int `json:"innerIndex"`
}

// GetIdx returns the instruction index as string in format "outer-inner" or just "outer"
func (c *ClassifiedInstruction) GetIdx() string {
	if c.InnerIndex >= 0 {
		return formatIdx(c.OuterIndex, c.InnerIndex)
	}
	return formatIdxSingle(c.OuterIndex)
}

func formatIdx(outer, inner int) string {
	return string(rune('0'+outer)) + "-" + string(rune('0'+inner))
}

func formatIdxSingle(outer int) string {
	return string(rune('0' + outer))
}

// BalanceChange represents token balance changes before and after transaction execution
type BalanceChange struct {
	Pre    TokenAmount `json:"pre"`    // Token balance before transaction execution
	Post   TokenAmount `json:"post"`   // Token balance after transaction execution
	Change TokenAmount `json:"change"` // Net change in token balance (post - pre)
}

// TransactionStatus represents the transaction execution status
type TransactionStatus string

const (
	TransactionStatusUnknown TransactionStatus = "unknown"
	TransactionStatusSuccess TransactionStatus = "success"
	TransactionStatusFailed  TransactionStatus = "failed"
)

// AltEvent contains Address Lookup Table event data
type AltEvent struct {
	// Type is the ALT operation type (CreateLookupTable/FreezeLookupTable/ExtendLookupTable/DeactivateLookupTable/CloseLookupTable)
	Type string `json:"type"`

	// AltAccount is the Address Lookup Table account address
	AltAccount string `json:"altAccount"`

	// AltAuthority is the ALT authority (owner) address
	AltAuthority string `json:"altAuthority"`

	// Recipient is the recipient account for CloseLookupTable
	Recipient string `json:"recipient,omitempty"`

	// NewAddresses contains newly added addresses for ExtendLookupTable
	NewAddresses []string `json:"newAddresses,omitempty"`

	// PayerAccount is the fee payer for CreateLookupTable/ExtendLookupTable
	PayerAccount string `json:"payerAccount,omitempty"`

	// RecentSlot is the recent slot for CreateLookupTable
	RecentSlot uint64 `json:"recentSlot,omitempty"`

	// Idx is the instruction index
	Idx string `json:"idx,omitempty"`
}

// ParseResult contains complete parsing result with all extracted transaction data
type ParseResult struct {
	// State indicates parsing success status - true if parsing completed successfully
	State bool `json:"state"`

	// Fee is the transaction gas fee paid in SOL
	Fee TokenAmount `json:"fee"`

	// AggregateTrade contains aggregated trade information combining multiple related trades
	AggregateTrade *TradeInfo `json:"aggregateTrade,omitempty"`

	// Trades contains array of individual trade transactions found in the transaction
	Trades []TradeInfo `json:"trades"`

	// Liquidities contains array of liquidity operations (add/remove/create pool)
	Liquidities []PoolEvent `json:"liquidities"`

	// Transfers contains array of token transfer operations not related to trades
	Transfers []TransferData `json:"transfers"`

	// SolBalanceChange contains SOL balance change for the transaction signer
	SolBalanceChange *BalanceChange `json:"solBalanceChange,omitempty"`

	// TokenBalanceChange contains token balance changes mapped by token mint address
	TokenBalanceChange map[string]*BalanceChange `json:"tokenBalanceChange,omitempty"`

	// MemeEvents contains meme platform events (create/buy/sell/migrate/complete)
	MemeEvents []MemeEvent `json:"memeEvents"`

	// AltEvents contains Address Lookup Table events
	AltEvents []AltEvent `json:"altEvents,omitempty"`

	// Slot is the Solana slot number where the transaction was included
	Slot uint64 `json:"slot"`

	// Timestamp is the Unix timestamp when the transaction was processed
	Timestamp int64 `json:"timestamp"`

	// Signature is the unique transaction signature identifier
	Signature string `json:"signature"`

	// Signer contains array of public keys that signed this transaction
	Signer []string `json:"signer"`

	// ComputeUnits indicates compute units consumed by the transaction execution
	ComputeUnits uint64 `json:"computeUnits"`

	// TxStatus indicates final execution status of the transaction
	TxStatus TransactionStatus `json:"txStatus"`

	// Msg contains optional error or status message
	Msg string `json:"msg,omitempty"`

	// Extras contains additional parser-specific data
	Extras interface{} `json:"extras,omitempty"`
}

// NewParseResult creates a new ParseResult with default values
func NewParseResult() *ParseResult {
	return &ParseResult{
		State:       true,
		Fee:         TokenAmount{Amount: "0", Decimals: 9},
		Trades:      make([]TradeInfo, 0),
		Liquidities: make([]PoolEvent, 0),
		Transfers:   make([]TransferData, 0),
		MemeEvents:  make([]MemeEvent, 0),
		AltEvents:   make([]AltEvent, 0),
		Signer:      make([]string, 0),
		TxStatus:    TransactionStatusUnknown,
	}
}

// IsArbitrage returns true if the aggregated trade is an arbitrage
// (same input and output token mint addresses)
func (r *ParseResult) IsArbitrage() bool {
	if r.AggregateTrade != nil {
		return r.AggregateTrade.InputToken.Mint == r.AggregateTrade.OutputToken.Mint
	}
	return false
}

// ParseShredResult contains parsing result for shred-stream data (pre-execution instruction analysis)
type ParseShredResult struct {
	// State indicates parsing success status - true if shred parsing completed successfully
	State bool `json:"state"`

	// Signature is the transaction signature being analyzed
	Signature string `json:"signature"`

	// Instructions contains parsed instructions grouped by AMM/DEX name (legacy format)
	Instructions map[string][]interface{} `json:"instructions"`

	// ParsedInstructions contains typed parsed instructions (new format)
	ParsedInstructions []ParsedShredInstruction `json:"parsedInstructions,omitempty"`

	// Slot is the block slot number
	Slot uint64 `json:"slot,omitempty"`

	// Timestamp is the Unix timestamp
	Timestamp int64 `json:"timestamp,omitempty"`

	// Signer contains transaction signers
	Signer []string `json:"signer,omitempty"`

	// Msg contains optional error or status message
	Msg string `json:"msg,omitempty"`
}

// ParsedShredInstruction represents a typed parsed shred instruction
type ParsedShredInstruction struct {
	// ProgramID is the program that owns this instruction
	ProgramID string `json:"programId"`

	// ProgramName is the human-readable name of the program
	ProgramName string `json:"programName"`

	// Action is the instruction action type (e.g., "pumpswap_swap", "transfer")
	Action string `json:"action"`

	// Trade contains trade data if this is a trade instruction
	Trade *TradeInfo `json:"trade,omitempty"`

	// Liquidity contains liquidity data if this is a liquidity instruction
	Liquidity *PoolEvent `json:"liquidity,omitempty"`

	// Transfer contains transfer data if this is a transfer instruction
	Transfer *TransferData `json:"transfer,omitempty"`

	// MemeEvent contains meme event data if this is a meme event instruction
	MemeEvent *MemeEvent `json:"memeEvent,omitempty"`

	// Data contains additional instruction-specific data
	Data interface{} `json:"data,omitempty"`

	// Accounts contains the account addresses involved in this instruction
	Accounts []string `json:"accounts"`

	// Idx is the instruction index in format "outer" or "outer-inner"
	Idx string `json:"idx"`
}

// EventParser is a generic event parser configuration for single discriminator events
type EventParser[T any] struct {
	// Discriminator is the unique byte sequence identifying this event type
	Discriminator []byte

	// Decode decodes raw event data into typed object
	Decode func(data []byte) (T, error)
}

// EventsParser is a generic event parser configuration for multiple discriminator events
type EventsParser[T any] struct {
	// Discriminators are byte sequences identifying related event types
	Discriminators [][]byte

	// Slice is the number of bytes to slice from the beginning of data
	Slice int

	// Decode decodes raw event data with additional options
	Decode func(data []byte, options interface{}) (T, error)
}

// InstructionParser is a generic instruction parser configuration
type InstructionParser[T any] struct {
	// Discriminator is the unique byte sequence identifying this instruction type
	Discriminator []byte

	// Decode decodes instruction data with additional options
	Decode func(instruction interface{}, options interface{}) (T, error)
}
