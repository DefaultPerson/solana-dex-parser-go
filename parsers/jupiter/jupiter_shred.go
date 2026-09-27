package jupiter

import (
	"bytes"
	"errors"
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
// v1 routes are (route_plan Vec<RoutePlanStep>, in u64, out u64,
// slippage_bps u16, platform_fee_bps u8), token-ledger routes the same
// without the input amount; v2 routes start with (in u64, out u64,
// slippage_bps u16, platform_fee_bps u16, positive_slippage_bps u16)
// followed by the route plan. Shared routes have an id u8 first.
func (p *JupiterShredParser) decodeRoute(route jupiterRoute, accounts []string, data []byte) *JupiterRouteData {
	if len(accounts) < route.minAccounts() {
		return nil
	}

	var first, second uint64
	var slippageBps, platformFeeBps, positiveSlippageBps uint16

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	if route.shared {
		reader.Skip(1)
	}

	if route.v2 {
		first, _ = reader.ReadU64()
		second, _ = reader.ReadU64()
		slippageBps, _ = reader.ReadU16()
		platformFeeBps, _ = reader.ReadU16()
		positiveSlippageBps, _ = reader.ReadU16()
		reader.ReadU32() // route plan length
		if reader.HasError() {
			return nil
		}
	} else {
		// The arguments follow the route plan. Anchor ignores bytes after
		// the last argument and some clients send one, so the plan is walked
		// forward instead of reading the arguments from the end of the data.
		args := reader
		if err := skipJupiterBorsh(reader, jupiterRoutePlan); err != nil {
			if !errors.Is(err, errJupiterUnknownVariant) {
				return nil
			}
			// A Swap variant newer than jupiterSwapVariants: read the
			// arguments from the end, which is right unless the data carries
			// trailing bytes
			tail := 8 + 2 + 1
			if !route.tokenLedger {
				tail += 8
			}
			if len(data) < tail+4 {
				return nil
			}
			args = utils.GetBinaryReader(data[len(data)-tail:])
			defer args.Release()
		}
		if !route.tokenLedger {
			first, _ = args.ReadU64()
		}
		second, _ = args.ReadU64()
		slippageBps, _ = args.ReadU16()
		fee, _ := args.ReadU8()
		platformFeeBps = uint16(fee)
		if args.HasError() {
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
// the direction is SWAP unless a known side is SOL or a stablecoin. A trade
// that ends in the mint it starts with (a circular arbitrage route) is a
// SWAP too.
func shredTradeType(inMint, outMint string) types.TradeType {
	if inMint != "" && inMint == outMint {
		return types.TradeTypeSwap
	}
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

// jupiterBorshKind is the kind of a jupiterBorsh layout node
type jupiterBorshKind uint8

const (
	jupiterBorshFixed jupiterBorshKind = iota
	jupiterBorshVec
	jupiterBorshOption
	jupiterBorshEnum
)

// jupiterBorsh is a node of a Borsh layout, enough to skip over a route
// plan: fixed-size bytes, a Vec or an Option of a sequence of nodes, or an
// enum (u8 tag, then the fields of that variant)
type jupiterBorsh struct {
	kind jupiterBorshKind
	// size is the byte count of a fixed node and the minimum element size
	// of a Vec
	size int
	// elem is the element of a Vec or the value of an Option
	elem []jupiterBorsh
	// variants are the fields of each enum variant by tag
	variants [][]jupiterBorsh
}

func jbFixed(n int) jupiterBorsh { return jupiterBorsh{kind: jupiterBorshFixed, size: n} }

func jbVec(minElem int, elem ...jupiterBorsh) jupiterBorsh {
	return jupiterBorsh{kind: jupiterBorshVec, size: minElem, elem: elem}
}

func jbOption(value ...jupiterBorsh) jupiterBorsh {
	return jupiterBorsh{kind: jupiterBorshOption, elem: value}
}

func jbEnum(variants [][]jupiterBorsh) jupiterBorsh {
	return jupiterBorsh{kind: jupiterBorshEnum, variants: variants}
}

// errJupiterUnknownVariant is returned for an enum tag the layout tables do
// not know, i.e. a variant added to the program after they were generated
var errJupiterUnknownVariant = errors.New("jupiter: unknown enum variant")

var errJupiterInvalidOption = errors.New("jupiter: invalid option tag")

// skipJupiterBorsh advances r over a value with the given layout. Vec
// lengths are checked against the remaining data before looping.
func skipJupiterBorsh(r *utils.BinaryReader, layout []jupiterBorsh) error {
	for _, node := range layout {
		switch node.kind {
		case jupiterBorshFixed:
			if err := r.Skip(node.size); err != nil {
				return err
			}
		case jupiterBorshVec:
			n, err := r.ReadVecLength(node.size)
			if err != nil {
				return err
			}
			for i := 0; i < n; i++ {
				if err := skipJupiterBorsh(r, node.elem); err != nil {
					return err
				}
			}
		case jupiterBorshOption:
			tag, err := r.ReadU8()
			if err != nil {
				return err
			}
			if tag > 1 {
				return errJupiterInvalidOption
			}
			if tag == 1 {
				if err := skipJupiterBorsh(r, node.elem); err != nil {
					return err
				}
			}
		case jupiterBorshEnum:
			tag, err := r.ReadU8()
			if err != nil {
				return err
			}
			if int(tag) >= len(node.variants) {
				return errJupiterUnknownVariant
			}
			if err := skipJupiterBorsh(r, node.variants[tag]); err != nil {
				return err
			}
		}
	}
	return nil
}

// jupiterRemainingAccountsInfo is RemainingAccountsInfo: a Vec of
// (accounts_type u8, length u8) slices
var jupiterRemainingAccountsInfo = jbVec(2, jbFixed(2))

// jupiterCandidateSwap is the CandidateSwap enum
var jupiterCandidateSwap = jbEnum(jupiterCandidateSwapVariants)

// jupiterRoutePlan is the v1 route_plan: a Vec of RoutePlanStep (swap Swap,
// percent u8, input_index u8, output_index u8)
var jupiterRoutePlan = []jupiterBorsh{jbVec(4, jbEnum(jupiterSwapVariants), jbFixed(3))}

// jupiterCandidateSwapVariants lists the payload layouts of the CandidateSwap
// variants (the candidates of a DynamicV1/DynamicV2 step) by tag
var jupiterCandidateSwapVariants = [][]jupiterBorsh{
	{jbFixed(9)}, // 0 HumidiFi
	{jbFixed(1)}, // 1 TesseraV
	{jbFixed(9)}, // 2 HumidiFiV2
	nil,          // 3 RaydiumV2
	nil,          // 4 RaydiumClmm
	{jbFixed(1)}, // 5 Whirlpool
	nil,          // 6 ZeroFi
	{jbFixed(1)}, // 7 BisonFiV2
	{jbFixed(1)}, // 8 GoonFiV2
	{jbFixed(1)}, // 9 GoonFiV3
	{jbFixed(1), jbOption(jupiterRemainingAccountsInfo)}, // 10 WhirlpoolV2
	nil,           // 11 ZeroFiSwapV2
	{jbFixed(1)},  // 12 BisonFiMarketBacked
	nil,           // 13 RaydiumClmmV2
	{jbFixed(1)},  // 14 TesseraVV2
	{jbFixed(57)}, // 15 HumidiFiRouter
	{jbFixed(65)}, // 16 HumidiFiRouterV2
}

// jupiterSwapVariants lists the payload layouts of the Swap variants (one
// route plan step) by tag, generated from the on-chain JUP6 IDL (IDL account
// C88XWfp26heEmDkmfSzeXP7Fd7GQJ2j9dDTUsyiZbUTa, fetched 2026-09-27). nil is a
// variant without fields.
var jupiterSwapVariants = [][]jupiterBorsh{
	nil,           // 0 Saber
	nil,           // 1 SaberAddDecimalsDeposit
	nil,           // 2 SaberAddDecimalsWithdraw
	nil,           // 3 TokenSwap
	nil,           // 4 Sencha
	nil,           // 5 Step
	nil,           // 6 Cropper
	nil,           // 7 Raydium
	{jbFixed(1)},  // 8 Crema
	nil,           // 9 Lifinity
	nil,           // 10 Mercurial
	nil,           // 11 Cykura
	{jbFixed(1)},  // 12 Serum
	nil,           // 13 MarinadeDeposit
	nil,           // 14 MarinadeUnstake
	{jbFixed(1)},  // 15 Aldrin
	{jbFixed(1)},  // 16 AldrinV2
	{jbFixed(1)},  // 17 Whirlpool
	{jbFixed(1)},  // 18 Invariant
	nil,           // 19 Meteora
	nil,           // 20 GooseFX
	{jbFixed(1)},  // 21 DeltaFi
	nil,           // 22 Balansol
	{jbFixed(1)},  // 23 MarcoPolo
	{jbFixed(1)},  // 24 Dradex
	nil,           // 25 LifinityV2
	nil,           // 26 RaydiumClmm
	{jbFixed(1)},  // 27 Openbook
	{jbFixed(1)},  // 28 Phoenix
	{jbFixed(16)}, // 29 Symmetry
	nil,           // 30 TokenSwapV2
	nil,           // 31 HeliumTreasuryManagementRedeemV0
	nil,           // 32 StakeDexStakeWrappedSol
	{jbFixed(4)},  // 33 StakeDexSwapViaStake
	nil,           // 34 GooseFXV2
	nil,           // 35 Perps
	nil,           // 36 PerpsAddLiquidity
	nil,           // 37 PerpsRemoveLiquidity
	nil,           // 38 MeteoraDlmm
	{jbFixed(1)},  // 39 OpenBookV2
	nil,           // 40 RaydiumClmmV2
	{jbFixed(4)},  // 41 StakeDexPrefundWithdrawStakeAndDepositStake
	{jbFixed(3)},  // 42 Clone
	{jbFixed(10)}, // 43 SanctumS
	{jbFixed(5)},  // 44 SanctumSAddLiquidity
	{jbFixed(5)},  // 45 SanctumSRemoveLiquidity
	nil,           // 46 RaydiumCP
	{jbFixed(1), jbOption(jupiterRemainingAccountsInfo)}, // 47 WhirlpoolSwapV2
	nil,                            // 48 OneIntro
	nil,                            // 49 PumpWrappedBuy
	nil,                            // 50 PumpWrappedSell
	nil,                            // 51 PerpsV2
	nil,                            // 52 PerpsV2AddLiquidity
	nil,                            // 53 PerpsV2RemoveLiquidity
	nil,                            // 54 MoonshotWrappedBuy
	nil,                            // 55 MoonshotWrappedSell
	nil,                            // 56 StabbleStableSwap
	nil,                            // 57 StabbleWeightedSwap
	{jbFixed(1)},                   // 58 Obric
	nil,                            // 59 FoxBuyFromEstimatedCost
	{jbFixed(1)},                   // 60 FoxClaimPartial
	{jbFixed(1)},                   // 61 SolFi
	nil,                            // 62 SolayerDelegateNoInit
	nil,                            // 63 SolayerUndelegateNoInit
	{jbFixed(1)},                   // 64 TokenMill
	nil,                            // 65 DaosFunBuy
	nil,                            // 66 DaosFunSell
	nil,                            // 67 ZeroFi
	nil,                            // 68 StakeDexWithdrawWrappedSol
	nil,                            // 69 VirtualsBuy
	nil,                            // 70 VirtualsSell
	{jbFixed(2)},                   // 71 Perena
	nil,                            // 72 PumpSwapBuy
	nil,                            // 73 PumpSwapSell
	nil,                            // 74 Gamma
	{jupiterRemainingAccountsInfo}, // 75 MeteoraDlmmSwapV2
	nil,                            // 76 Woofi
	nil,                            // 77 MeteoraDammV2
	nil,                            // 78 MeteoraDynamicBondingCurveSwap
	nil,                            // 79 StabbleStableSwapV2
	nil,                            // 80 StabbleWeightedSwapV2
	{jbFixed(8)},                   // 81 RaydiumLaunchlabBuy
	{jbFixed(8)},                   // 82 RaydiumLaunchlabSell
	nil,                            // 83 BoopdotfunWrappedBuy
	nil,                            // 84 BoopdotfunWrappedSell
	{jbFixed(1)},                   // 85 Plasma
	{jbFixed(2)},                   // 86 GoonFi
	{jbFixed(9)},                   // 87 HumidiFi
	nil,                            // 88 MeteoraDynamicBondingCurveSwapWithRemainingAccounts
	{jbFixed(1)},                   // 89 TesseraV
	nil,                            // 90 PumpWrappedBuyV2
	nil,                            // 91 PumpWrappedSellV2
	nil,                            // 92 PumpSwapBuyV2
	nil,                            // 93 PumpSwapSellV2
	{jbFixed(1)},                   // 94 Heaven
	{jbFixed(1)},                   // 95 SolFiV2
	nil,                            // 96 Aquifer
	nil,                            // 97 PumpWrappedBuyV3
	nil,                            // 98 PumpWrappedSellV3
	nil,                            // 99 PumpSwapBuyV3
	nil,                            // 100 PumpSwapSellV3
	nil,                            // 101 JupiterLendDeposit
	nil,                            // 102 JupiterLendRedeem
	{jbFixed(1), jbOption(jupiterRemainingAccountsInfo)}, // 103 DefiTuna
	{jbFixed(1)}, // 104 AlphaQ
	nil,          // 105 RaydiumV2
	{jbFixed(1)}, // 106 SarosDlmm
	{jbFixed(1)}, // 107 Futarchy
	nil,          // 108 MeteoraDammV2WithRemainingAccounts
	nil,          // 109 Obsidian
	{jbFixed(1)}, // 110 WhaleStreet
	{jbVec(1, jupiterCandidateSwap), jbOption(jbFixed(1))}, // 111 DynamicV1
	nil,                                // 112 PumpWrappedBuyV4
	nil,                                // 113 PumpWrappedSellV4
	nil,                                // 114 CarrotIssue
	nil,                                // 115 CarrotRedeem
	{jbFixed(1)},                       // 116 Manifest
	{jbFixed(1)},                       // 117 BisonFi
	{jbFixed(9)},                       // 118 HumidiFiV2
	{jbFixed(1)},                       // 119 PerenaStar
	{jbFixed(1), jbVec(1, jbFixed(1))}, // 120 JupiterRfqV2
	{jbFixed(1)},                       // 121 GoonFiV2
	{jbFixed(16)},                      // 122 Scorch
	{jbFixed(48)},                      // 123 VaultLiquidUnstake
	nil,                                // 124 XOrca
	{jbFixed(1)},                       // 125 Quantum
	{jbFixed(17)},                      // 126 WhaleStreetV2
	{jbFixed(1)},                       // 127 Riptide
	nil,                                // 128 RunnerRodeo
	{jbFixed(1)},                       // 129 TaurusFi
	nil,                                // 130 Omnipair
	nil,                                // 131 MSwap
	{jbFixed(1)},                       // 132 Hylo
	nil,                                // 133 VoltrDeposit
	nil,                                // 134 VoltrWithdraw
	{jbFixed(10)},                      // 135 SanctumSV2
	{jbFixed(1)},                       // 136 LemmingsFi
	nil,                                // 137 ScaleVmmBuy
	nil,                                // 138 ScaleVmmSell
	nil,                                // 139 ScaleAmmBuy
	nil,                                // 140 ScaleAmmSell
	{jbFixed(1)},                       // 141 BisonFiV2
	nil,                                // 142 Trends
	nil,                                // 143 HumaDeposit
	nil,                                // 144 HumaInstantWithdraw
	{jbFixed(1)},                       // 145 Kipseli
	{jbVec(5, jupiterCandidateSwap, jbFixed(4)), jbFixed(2)}, // 146 DynamicV2
	nil,                                // 147 PumpSwapBuyV3WithCashbackClaim
	nil,                                // 148 PumpSwapSellV3WithCashbackClaim
	nil,                                // 149 PumpWrappedBuyV4WithCashbackClaim
	nil,                                // 150 PumpWrappedSellV4WithCashbackClaim
	{jbFixed(1)},                       // 151 GoonFiV3
	{jbFixed(1)},                       // 152 PumpWrappedBuyV5
	{jbFixed(1)},                       // 153 PumpWrappedSellV5
	nil,                                // 154 ZeroFiSwapV2
	{jbFixed(2)},                       // 155 BisonFiPredict
	nil,                                // 156 ByrealDynamicV3
	{jbFixed(9)},                       // 157 Flux
	nil,                                // 158 VaultLiquidSellLst
	{jbFixed(8)},                       // 159 VaultLiquidBuyLst
	{jbFixed(1)},                       // 160 KipseliV2
	{jbFixed(5)},                       // 161 Deriverse
	{jbFixed(1)},                       // 162 Hadron
	nil,                                // 163 BinaryFi
	{jbFixed(1)},                       // 164 Metric
	{jbFixed(1)},                       // 165 JupiterLendDexSwap
	{jbFixed(1)},                       // 166 Gatorswap
	{jbFixed(2)},                       // 167 Flint
	{jbFixed(1)},                       // 168 Denali
	nil,                                // 169 PerenaStarV2Deposit
	{jbFixed(1)},                       // 170 PerenaStarV2WithdrawFromExternal
	{jbFixed(1)},                       // 171 SanctumSols
	{jbFixed(1)},                       // 172 HyloV2
	nil,                                // 173 SanctumPamm
	{jbFixed(1)},                       // 174 Archer
	nil,                                // 175 TrenchWrappedBuy
	nil,                                // 176 TrenchWrappedSell
	{jbFixed(1)},                       // 177 BisonFiMarketBacked
	{jbFixed(1)},                       // 178 TesseraVV2
	nil,                                // 179 Stableswap
	nil,                                // 180 BinaryFiV2
	{jbFixed(1)},                       // 181 KipseliV3
	{jbFixed(1)},                       // 182 PerenaStarV2TrancheDeposit
	{jbFixed(1), jbOption(jbFixed(1))}, // 183 PerenaStarV2TrancheWithdrawFromExternal
	{jbFixed(1)},                       // 184 Quay
	{jbFixed(57)},                      // 185 HumidiFiRouter
	{jbFixed(65)},                      // 186 HumidiFiRouterV2
	{jbFixed(2)},                       // 187 PumpWrappedBuyV6
	nil,                                // 188 HyloRouter
}
