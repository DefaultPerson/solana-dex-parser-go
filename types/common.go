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
		Signer:      make([]string, 0),
		TxStatus:    TransactionStatusUnknown,
	}
}

// ParseShredResult contains parsing result for shred-stream data (pre-execution instruction analysis)
type ParseShredResult struct {
	// State indicates parsing success status - true if shred parsing completed successfully
	State bool `json:"state"`

	// Signature is the transaction signature being analyzed
	Signature string `json:"signature"`

	// Instructions contains parsed instructions grouped by AMM/DEX name
	Instructions map[string][]interface{} `json:"instructions"`

	// Msg contains optional error or status message
	Msg string `json:"msg,omitempty"`
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
