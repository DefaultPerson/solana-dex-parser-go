package types

import (
	"math"
	"math/big"
)

// TradeType represents the direction/type of a trade
type TradeType string

const (
	TradeTypeBuy      TradeType = "BUY"
	TradeTypeSell     TradeType = "SELL"
	TradeTypeSwap     TradeType = "SWAP"
	TradeTypeCreate   TradeType = "CREATE"
	TradeTypeMigrate  TradeType = "MIGRATE"
	TradeTypeComplete TradeType = "COMPLETE"
	TradeTypeAdd      TradeType = "ADD"
	TradeTypeRemove   TradeType = "REMOVE"
	TradeTypeLock     TradeType = "LOCK"
	TradeTypeBurn     TradeType = "BURN"
)

// ParseConfig contains configuration options for transaction parsing
type ParseConfig struct {
	// TryUnknownDEX if true, will try to parse unknown DEXes (results may be inaccurate)
	TryUnknownDEX bool `json:"tryUnknownDEX,omitempty"`

	// ProgramIds if set, will only parse transactions from these program IDs
	ProgramIds []string `json:"programIds,omitempty"`

	// IgnoreProgramIds if set, will ignore transactions from these program IDs
	IgnoreProgramIds []string `json:"ignoreProgramIds,omitempty"`

	// ThrowError if true, will return error if parsing fails
	ThrowError bool `json:"throwError,omitempty"`

	// AggregateTrades if true, will return the finalSwap record instead of detail route trades
	AggregateTrades bool `json:"aggregateTrades,omitempty"`
}

// DefaultParseConfig returns default parsing configuration
func DefaultParseConfig() ParseConfig {
	return ParseConfig{
		TryUnknownDEX:   true,
		AggregateTrades: true,
	}
}

// DexInfo contains basic DEX protocol information
type DexInfo struct {
	ProgramId string `json:"programId,omitempty"` // DEX program ID on Solana
	AMM       string `json:"amm,omitempty"`       // Automated Market Maker name
	Route     string `json:"route,omitempty"`     // Router or aggregator name
}

// TokenAmount represents a standard token amount format
type TokenAmount struct {
	Amount   string   `json:"amount"`           // Raw token amount
	UIAmount *float64 `json:"uiAmount"`         // Human-readable amount (can be null)
	Decimals uint8    `json:"decimals"`         // Token decimals
}

// TokenInfo contains token information including balances and accounts
type TokenInfo struct {
	Mint                  string       `json:"mint"`                            // Token mint address
	Amount                float64      `json:"amount"`                          // Token uiAmount
	AmountRaw             string       `json:"amountRaw"`                       // Raw token amount
	Decimals              uint8        `json:"decimals"`                        // Token decimals
	Authority             string       `json:"authority,omitempty"`             // Token authority (if applicable)
	Destination           string       `json:"destination,omitempty"`           // Destination token account
	DestinationOwner      string       `json:"destinationOwner,omitempty"`      // Owner of destination account
	DestinationBalance    *TokenAmount `json:"destinationBalance,omitempty"`    // Balance after transfer
	DestinationPreBalance *TokenAmount `json:"destinationPreBalance,omitempty"` // Balance before transfer
	Source                string       `json:"source,omitempty"`                // Source token account
	SourceBalance         *TokenAmount `json:"sourceBalance,omitempty"`         // Source balance after transfer
	SourcePreBalance      *TokenAmount `json:"sourcePreBalance,omitempty"`      // Source balance before transfer
	BalanceChange         string       `json:"balanceChange,omitempty"`         // Raw user balance change amount
}

// TransferInfo contains transfer information for tracking token movements
type TransferInfo struct {
	Type      string    `json:"type"`      // Transfer direction: TRANSFER_IN or TRANSFER_OUT
	Token     TokenInfo `json:"token"`     // Token details
	From      string    `json:"from"`      // Source address
	To        string    `json:"to"`        // Destination address
	Timestamp int64     `json:"timestamp"` // Unix timestamp
	Signature string    `json:"signature"` // Transaction signature
}

// TransferDataInfo contains detailed transfer data
type TransferDataInfo struct {
	Authority             string       `json:"authority,omitempty"`             // Transfer authority
	Destination           string       `json:"destination"`                     // Destination account
	DestinationOwner      string       `json:"destinationOwner,omitempty"`      // Owner of destination account
	Mint                  string       `json:"mint"`                            // Token mint address
	Source                string       `json:"source"`                          // Source account
	TokenAmount           TokenAmount  `json:"tokenAmount"`                     // Amount details
	SourceBalance         *TokenAmount `json:"sourceBalance,omitempty"`         // Source balance after transfer
	SourcePreBalance      *TokenAmount `json:"sourcePreBalance,omitempty"`      // Source balance before transfer
	DestinationBalance    *TokenAmount `json:"destinationBalance,omitempty"`    // Balance after transfer
	DestinationPreBalance *TokenAmount `json:"destinationPreBalance,omitempty"` // Balance before transfer
	SolBalanceChange      string       `json:"solBalanceChange,omitempty"`      // Raw SOL balance change amount
}

// TransferData contains detailed transfer data including account information
type TransferData struct {
	Type      string           `json:"type"`      // Transfer instruction type
	ProgramId string           `json:"programId"` // Token program ID
	Info      TransferDataInfo `json:"info"`      // Transfer details
	Idx       string           `json:"idx"`       // Instruction index
	Timestamp int64            `json:"timestamp"` // Unix timestamp
	Signature string           `json:"signature"` // Transaction signature
	IsFee     bool             `json:"isFee,omitempty"` // Whether it's a fee transfer
}

// FeeInfo contains fee information
type FeeInfo struct {
	Mint      string  `json:"mint"`                // Fee token mint address
	Amount    float64 `json:"amount"`              // Fee amount in UI format
	AmountRaw string  `json:"amountRaw"`           // Raw fee amount
	Decimals  uint8   `json:"decimals"`            // Fee token decimals
	Dex       string  `json:"dex,omitempty"`       // DEX name (e.g., 'Raydium', 'Meteora')
	Type      string  `json:"type,omitempty"`      // Fee type (e.g., 'protocol', 'coinCreator')
	Recipient string  `json:"recipient,omitempty"` // Fee recipient account
}

// TradeInfo contains comprehensive trade information
type TradeInfo struct {
	User        string     `json:"user"`                  // Signer address (trader)
	Type        TradeType  `json:"type"`                  // Trade direction (BUY/SELL/SWAP)
	Pool        []string   `json:"Pool"`                  // Pool addresses
	InputToken  TokenInfo  `json:"inputToken"`            // Token being sold
	OutputToken TokenInfo  `json:"outputToken"`           // Token being bought
	SlippageBps *int       `json:"slippageBps,omitempty"` // Slippage in basis points
	Fee         *FeeInfo   `json:"fee,omitempty"`         // Fee information (if applicable)
	Fees        []FeeInfo  `json:"fees,omitempty"`        // List of fees (if multiple)
	ProgramId   string     `json:"programId,omitempty"`   // DEX program ID
	AMM         string     `json:"amm,omitempty"`         // AMM type (e.g., 'RaydiumV4', 'Meteora')
	AMMs        []string   `json:"amms,omitempty"`        // List of AMMs (if multiple)
	Route       string     `json:"route,omitempty"`       // Router or Bot name
	Slot        uint64     `json:"slot"`                  // Block slot number
	Timestamp   int64      `json:"timestamp"`             // Unix timestamp
	Signature   string     `json:"signature"`             // Transaction signature
	Idx         string     `json:"idx"`                   // Instruction indexes
	Signer      []string   `json:"signer,omitempty"`      // Original signer
}

// ConvertToUIAmount converts raw token amount to human-readable format
func ConvertToUIAmount(amount *big.Int, decimals uint8) float64 {
	if decimals == 0 {
		f, _ := new(big.Float).SetInt(amount).Float64()
		return f
	}
	divisor := new(big.Float).SetFloat64(math.Pow10(int(decimals)))
	result := new(big.Float).SetInt(amount)
	result.Quo(result, divisor)
	f, _ := result.Float64()
	return f
}

// ConvertToUIAmountString converts raw token amount string to human-readable format
func ConvertToUIAmountString(amountStr string, decimals uint8) float64 {
	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return 0
	}
	return ConvertToUIAmount(amount, decimals)
}

// ConvertToUIAmountUint64 converts uint64 raw amount to human-readable format
func ConvertToUIAmountUint64(amount uint64, decimals uint8) float64 {
	return ConvertToUIAmount(new(big.Int).SetUint64(amount), decimals)
}
