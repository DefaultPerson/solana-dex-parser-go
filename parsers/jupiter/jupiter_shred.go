package jupiter

import (
	"bytes"
	"strconv"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// JupiterShredParser parses Jupiter V6 instructions from shred-stream
type JupiterShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewJupiterShredParser creates a new JupiterShredParser
func NewJupiterShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *JupiterShredParser {
	return &JupiterShredParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes Jupiter instructions and returns parsed results
func (p *JupiterShredParser) ProcessInstructions() []interface{} {
	events, _ := p.ProcessAll()
	return events
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *JupiterShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	_, typed := p.ProcessAll()
	return typed
}

// jupiterRoute describes the argument and account layout of a route instruction
type jupiterRoute struct {
	name string
	// shared routes start with an id u8
	shared bool
	// v2 routes put the amounts at the head of the arguments, v1 routes at the tail
	v2 bool
	// exactOut: the first amount is the exact output, the second the quoted input
	exactOut bool
	// tokenLedger: the input amount is read from the token ledger at execution
	tokenLedger bool
	// account indexes; sourceMint < 0 means the instruction has no source mint
	// account and the mint is resolved from the source token account
	user, sourceAccount, destinationAccount, sourceMint, destinationMint int
}

func (r jupiterRoute) minAccounts() int {
	n := r.user
	for _, i := range []int{r.sourceAccount, r.destinationAccount, r.sourceMint, r.destinationMint} {
		if i > n {
			n = i
		}
	}
	return n + 1
}

// jupiterRouteFor returns the layout of a route instruction (on-chain JUP6 IDL)
func jupiterRouteFor(disc []byte) (jupiterRoute, bool) {
	d := constants.DISCRIMINATORS.JUPITER
	switch {
	case bytes.Equal(disc, d.ROUTE):
		return jupiterRoute{name: "route", user: 1, sourceAccount: 2, destinationAccount: 3, sourceMint: -1, destinationMint: 5}, true
	case bytes.Equal(disc, d.ROUTE_WITH_TOKEN_LEDGER):
		return jupiterRoute{name: "route_with_token_ledger", tokenLedger: true, user: 1, sourceAccount: 2, destinationAccount: 3, sourceMint: -1, destinationMint: 5}, true
	case bytes.Equal(disc, d.ROUTE_EXACT_OUT):
		return jupiterRoute{name: "route_exact_out", exactOut: true, user: 1, sourceAccount: 2, destinationAccount: 3, sourceMint: 5, destinationMint: 6}, true
	case bytes.Equal(disc, d.SHARE_ACCOUNTS_ROUTE):
		return jupiterRoute{name: "shared_accounts_route", shared: true, user: 2, sourceAccount: 3, destinationAccount: 6, sourceMint: 7, destinationMint: 8}, true
	case bytes.Equal(disc, d.SHARE_ACCOUNTS_EXACT_OUT_ROUTE):
		return jupiterRoute{name: "shared_accounts_exact_out_route", shared: true, exactOut: true, user: 2, sourceAccount: 3, destinationAccount: 6, sourceMint: 7, destinationMint: 8}, true
	case bytes.Equal(disc, d.SHARE_ACCOUNTS_ROUTE_WITH_TOKEN_LEDGER):
		return jupiterRoute{name: "shared_accounts_route_with_token_ledger", shared: true, tokenLedger: true, user: 2, sourceAccount: 3, destinationAccount: 6, sourceMint: 7, destinationMint: 8}, true
	case bytes.Equal(disc, d.ROUTE_V2):
		return jupiterRoute{name: "route_v2", v2: true, user: 0, sourceAccount: 1, destinationAccount: 2, sourceMint: 3, destinationMint: 4}, true
	case bytes.Equal(disc, d.EXACT_OUT_ROUTE_V2):
		return jupiterRoute{name: "exact_out_route_v2", v2: true, exactOut: true, user: 0, sourceAccount: 1, destinationAccount: 2, sourceMint: 3, destinationMint: 4}, true
	case bytes.Equal(disc, d.SHARED_ACCOUNTS_ROUTE_V2):
		return jupiterRoute{name: "shared_accounts_route_v2", v2: true, shared: true, user: 1, sourceAccount: 2, destinationAccount: 5, sourceMint: 6, destinationMint: 7}, true
	case bytes.Equal(disc, d.SHARED_ACCOUNTS_EXACT_OUT_ROUTE_V2):
		return jupiterRoute{name: "shared_accounts_exact_out_route_v2", v2: true, shared: true, exactOut: true, user: 1, sourceAccount: 2, destinationAccount: 5, sourceMint: 6, destinationMint: 7}, true
	}
	return jupiterRoute{}, false
}

// ProcessAll decodes the Jupiter route instructions into legacy events and
// typed trades in a single pass
func (p *JupiterShredParser) ProcessAll() ([]interface{}, []types.ParsedShredInstruction) {
	var events []interface{}
	var typed []types.ParsedShredInstruction

	for _, ci := range p.classifier.GetInstructions(constants.DEX_PROGRAMS.JUPITER.ID) {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}
		route, ok := jupiterRouteFor(data[:8])
		if !ok {
			continue
		}
		accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
		routeData := p.decodeRoute(route, accounts, data[8:])
		if routeData == nil {
			continue
		}

		idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
		events = append(events, &JupiterShredInstruction{
			Type:      route.name,
			Data:      routeData,
			Slot:      p.adapter.Slot(),
			Timestamp: p.adapter.BlockTime(),
			Signature: p.adapter.Signature(),
			Idx:       idx,
			Signer:    p.adapter.Signers(),
		})

		inKind, outKind := types.ShredAmountExact, types.ShredAmountQuote
		if route.exactOut {
			inKind, outKind = types.ShredAmountQuote, types.ShredAmountExact
		}
		if route.tokenLedger {
			inKind = types.ShredAmountUnknown
		}
		typed = append(typed, types.ParsedShredInstruction{
			ProgramID:        constants.DEX_PROGRAMS.JUPITER.ID,
			ProgramName:      constants.DEX_PROGRAMS.JUPITER.Name,
			Action:           route.name,
			Trade:            p.buildTradeInfo(routeData),
			Accounts:         accounts,
			Idx:              idx,
			InputAmountKind:  inKind,
			OutputAmountKind: outKind,
		})
	}

	return events, typed
}

// JupiterShredInstruction represents a parsed Jupiter instruction
type JupiterShredInstruction struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Slot      uint64      `json:"slot"`
	Timestamp int64       `json:"timestamp"`
	Signature string      `json:"signature"`
	Idx       string      `json:"idx"`
	Signer    []string    `json:"signer"`
}

// JupiterRouteData contains Jupiter route instruction data. The amounts are
// instruction arguments, not executed amounts: for exact-in routes
// InputAmount is the exact in_amount and OutputAmount the quoted_out_amount
// (the router's quote; the minimum output follows from SlippageBps); for
// exact-out routes (ExactOut) OutputAmount is the exact out_amount and
// InputAmount the quoted_in_amount (the maximum input follows from
// SlippageBps). Token-ledger routes read the input amount from the token
// ledger at execution, so InputAmount is 0.
//
// InputMint is empty when it cannot be determined: route and
// route_with_token_ledger name only the source token account, whose mint is
// known only when the transaction reveals it (token balances, instructions);
// mints loaded from an unresolved address lookup table are empty too.
type JupiterRouteData struct {
	User         string `json:"user"`
	InputMint    string `json:"inputMint"`
	OutputMint   string `json:"outputMint"`
	InputAmount  uint64 `json:"inputAmount"`
	OutputAmount uint64 `json:"outputAmount"`
	SlippageBps  uint16 `json:"slippageBps"`
	// PlatformFeeBps is the platform fee argument (u8 in v1 routes, u16 in v2)
	PlatformFeeBps uint16 `json:"platformFeeBps"`
	// PositiveSlippageBps is the v2 positive_slippage_bps argument
	PositiveSlippageBps uint16 `json:"positiveSlippageBps,omitempty"`
	// ExactOut is true for the exact-out routes
	ExactOut bool `json:"exactOut,omitempty"`
	// InputTokenAccount and OutputTokenAccount are the user's source and
	// destination token accounts
	InputTokenAccount  string `json:"inputTokenAccount,omitempty"`
	OutputTokenAccount string `json:"outputTokenAccount,omitempty"`
}

// decodeRoute decodes the amounts and accounts of a route instruction.
// v1 routes end with (in u64, out u64, slippage_bps u16, platform_fee_bps u8)
// after the route plan, token-ledger routes with (quoted_out u64,
// slippage_bps u16, platform_fee_bps u8); v2 routes start with (in u64,
// out u64, slippage_bps u16, platform_fee_bps u16, positive_slippage_bps u16)
// followed by the route plan. Shared routes have an id u8 first.
func (p *JupiterShredParser) decodeRoute(route jupiterRoute, accounts []string, data []byte) *JupiterRouteData {
	if len(accounts) < route.minAccounts() {
		return nil
	}

	var first, second uint64
	var slippageBps, platformFeeBps, positiveSlippageBps uint16

	switch {
	case route.v2:
		reader := utils.GetBinaryReader(data)
		defer reader.Release()
		if route.shared {
			reader.Skip(1)
		}
		first, _ = reader.ReadU64()
		second, _ = reader.ReadU64()
		slippageBps, _ = reader.ReadU16()
		platformFeeBps, _ = reader.ReadU16()
		positiveSlippageBps, _ = reader.ReadU16()
		if _, err := reader.ReadU32(); err != nil { // route plan length
			return nil
		}
	case route.tokenLedger:
		const tail = 8 + 2 + 1
		if len(data) < tail+4 {
			return nil
		}
		reader := utils.GetBinaryReader(data[len(data)-tail:])
		defer reader.Release()
		second, _ = reader.ReadU64()
		slippageBps, _ = reader.ReadU16()
		fee, _ := reader.ReadU8()
		platformFeeBps = uint16(fee)
		if reader.HasError() {
			return nil
		}
	default:
		const tail = 8 + 8 + 2 + 1
		if len(data) < tail+4 {
			return nil
		}
		reader := utils.GetBinaryReader(data[len(data)-tail:])
		defer reader.Release()
		first, _ = reader.ReadU64()
		second, _ = reader.ReadU64()
		slippageBps, _ = reader.ReadU16()
		fee, _ := reader.ReadU8()
		platformFeeBps = uint16(fee)
		if reader.HasError() {
			return nil
		}
	}

	routeData := &JupiterRouteData{
		User:                accounts[route.user],
		InputTokenAccount:   accounts[route.sourceAccount],
		OutputTokenAccount:  accounts[route.destinationAccount],
		OutputMint:          accounts[route.destinationMint],
		SlippageBps:         slippageBps,
		PlatformFeeBps:      platformFeeBps,
		PositiveSlippageBps: positiveSlippageBps,
		ExactOut:            route.exactOut,
	}
	if route.sourceMint >= 0 {
		routeData.InputMint = accounts[route.sourceMint]
	} else {
		routeData.InputMint = p.tokenAccountMint(routeData.InputTokenAccount)
	}
	if route.exactOut {
		routeData.OutputAmount, routeData.InputAmount = first, second
	} else {
		routeData.InputAmount, routeData.OutputAmount = first, second
	}
	return routeData
}

// tokenAccountMint returns the mint of a token account when the transaction
// reveals it, "" otherwise (never a guess)
func (p *JupiterShredParser) tokenAccountMint(account string) string {
	if account == "" || p.adapter.IsGuessedTokenAccount(account) {
		return ""
	}
	return p.adapter.GetSplTokenMint(account)
}

// decimals returns the decimals of mint when known, 0 (unknown) otherwise
func (p *JupiterShredParser) decimals(mint string) uint8 {
	if mint == "" {
		return 0
	}
	if d, ok := p.adapter.SPLDecimalsMap[mint]; ok {
		return d
	}
	return constants.TOKEN_DECIMALS[mint]
}

// shredTradeType is utils.GetTradeType for mints that may be unknown (""):
// the direction is SWAP unless a known side is SOL or a stablecoin
func shredTradeType(inMint, outMint string) types.TradeType {
	if inMint != "" && outMint != "" {
		return utils.GetTradeType(inMint, outMint)
	}
	if constants.IsQuoteToken(inMint) {
		return types.TradeTypeBuy
	}
	if constants.IsQuoteToken(outMint) {
		return types.TradeTypeSell
	}
	return types.TradeTypeSwap
}

func (p *JupiterShredParser) buildTradeInfo(data *JupiterRouteData) *types.TradeInfo {
	slippageBps := int(data.SlippageBps)
	inDecimals, outDecimals := p.decimals(data.InputMint), p.decimals(data.OutputMint)

	return &types.TradeInfo{
		Type: shredTradeType(data.InputMint, data.OutputMint),
		Pool: []string{},
		User: data.User,
		InputToken: types.TokenInfo{
			Mint:      data.InputMint,
			Amount:    types.ConvertToUIAmountUint64(data.InputAmount, inDecimals),
			AmountRaw: strconv.FormatUint(data.InputAmount, 10),
			Decimals:  inDecimals,
			Source:    data.InputTokenAccount,
		},
		OutputToken: types.TokenInfo{
			Mint:        data.OutputMint,
			Amount:      types.ConvertToUIAmountUint64(data.OutputAmount, outDecimals),
			AmountRaw:   strconv.FormatUint(data.OutputAmount, 10),
			Decimals:    outDecimals,
			Destination: data.OutputTokenAccount,
		},
		ProgramId:   constants.DEX_PROGRAMS.JUPITER.ID,
		AMMs:        []string{},
		Route:       constants.DEX_PROGRAMS.JUPITER.Name,
		SlippageBps: &slippageBps,
	}
}
