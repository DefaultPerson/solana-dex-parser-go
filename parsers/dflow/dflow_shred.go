package dflow

import (
	"bytes"
	"encoding/binary"
	"math"
	"strconv"

	"github.com/mr-tron/base58"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// DFlowShredParser decodes the DFlow swap_orchestrator swap instructions
// (swap, swap2 and their _with_destination and _with_destination_native
// variants) from their arguments
type DFlowShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewDFlowShredParser creates a new DFlowShredParser
func NewDFlowShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *DFlowShredParser {
	return &DFlowShredParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// DFlowShredInstruction represents a parsed DFlow instruction
type DFlowShredInstruction struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Slot      uint64      `json:"slot"`
	Timestamp int64       `json:"timestamp"`
	Signature string      `json:"signature"`
	Idx       string      `json:"idx"`
	Signer    []string    `json:"signer"`
	// UnresolvedAccounts is true when some of the instruction's accounts are
	// address lookup table entries that could not be resolved (empty
	// strings); the decoded data leaves them empty
	UnresolvedAccounts bool `json:"unresolvedAccounts,omitempty"`
}

// DFlowSwapData contains DFlow swap instruction data. InputAmount is the
// amount of the first swap action (0 when the action spends the whole balance,
// InputAmountUnknown); QuotedOutAmount is the quoted output after the platform
// fee, the minimum output follows from SlippageBps. The instruction names no
// input mint: InputMint comes from the first SwapEvent of an executed
// transaction and is empty otherwise; OutputMint is the destination mint
// (SOL for the _native variants), the destination token account's mint or
// the last SwapEvent's output mint.
type DFlowSwapData struct {
	// User is the user_token_authority (the trader; sponsored transactions
	// have another fee payer)
	User                        string   `json:"user"`
	InputMint                   string   `json:"inputMint"`
	OutputMint                  string   `json:"outputMint"`
	InputAmount                 uint64   `json:"inputAmount"`
	InputAmountUnknown          bool     `json:"inputAmountUnknown,omitempty"`
	QuotedOutAmount             uint64   `json:"quotedOutAmount"`
	SlippageBps                 uint16   `json:"slippageBps"`
	PlatformFeeBps              uint16   `json:"platformFeeBps"`
	PositiveSlippageFeeLimitPct uint8    `json:"positiveSlippageFeeLimitPct,omitempty"`
	DestinationAccount          string   `json:"destinationAccount,omitempty"`
	Actions                     []string `json:"actions"`
}

// dflowSwapLayout describes one swap instruction variant
type dflowSwapLayout struct {
	name string
	// swap2: Swap2Params (with positive_slippage_fee_limit_pct)
	swap2 bool
	// destination: destination token account index, -1 for none
	destination int
	// destinationMint: destination mint index, -1 for none
	destinationMint int
	// native: the destination receives native SOL
	native bool
}

func dflowSwapLayoutFor(disc []byte) (dflowSwapLayout, bool) {
	d := constants.DISCRIMINATORS.DFLOW
	switch {
	case bytes.Equal(disc, d.SWAP):
		return dflowSwapLayout{name: "swap", destination: -1, destinationMint: -1}, true
	case bytes.Equal(disc, d.SWAP2):
		return dflowSwapLayout{name: "swap2", swap2: true, destination: -1, destinationMint: -1}, true
	case bytes.Equal(disc, d.SWAP_WITH_DEST):
		return dflowSwapLayout{name: "swap_with_destination", destination: 4, destinationMint: 6}, true
	case bytes.Equal(disc, d.SWAP2_WITH_DEST):
		return dflowSwapLayout{name: "swap2_with_destination", swap2: true, destination: 4, destinationMint: 6}, true
	case bytes.Equal(disc, d.SWAP_WITH_DEST_NATIVE):
		return dflowSwapLayout{name: "swap_with_destination_native", destination: 4, destinationMint: -1, native: true}, true
	case bytes.Equal(disc, d.SWAP2_WITH_DEST_NATIVE):
		return dflowSwapLayout{name: "swap2_with_destination_native", swap2: true, destination: 4, destinationMint: -1, native: true}, true
	}
	return dflowSwapLayout{}, false
}

// ProcessInstructions returns the decoded swaps as legacy events
func (p *DFlowShredParser) ProcessInstructions() []interface{} {
	events, _ := p.ProcessAll()
	return events
}

// ProcessTypedInstructions returns the decoded swaps as typed trades
func (p *DFlowShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	_, typed := p.ProcessAll()
	return typed
}

// dflowSwapEvent is one leg of an executed swap (SwapEvent self-CPI)
type dflowSwapEvent struct {
	inputMint, outputMint string
}

// dflowSwapEventDisc is the emit_cpi prefix + event:SwapEvent
var dflowSwapEventDisc = []byte{228, 69, 165, 46, 81, 203, 154, 29, 64, 198, 205, 232, 38, 8, 113, 226}

// ProcessAll decodes the DFlow swaps into legacy events and typed trades
func (p *DFlowShredParser) ProcessAll() ([]interface{}, []types.ParsedShredInstruction) {
	var events []interface{}
	var typed []types.ParsedShredInstruction

	instructions := p.classifier.GetInstructions(constants.DEX_PROGRAMS.DFLOW.ID)
	legs := p.swapEvents(instructions)

	for _, ci := range instructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}
		layout, ok := dflowSwapLayoutFor(data[:8])
		if !ok {
			continue
		}
		accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
		swap := decodeDFlowSwap(layout, accounts, data[8:])
		if swap == nil {
			continue
		}
		if swap.OutputMint == "" && swap.DestinationAccount != "" {
			swap.OutputMint = p.tokenAccountMint(swap.DestinationAccount)
		}
		if l := legs[swapKey(ci)]; len(l) > 0 {
			swap.InputMint = l[0].inputMint
			if swap.OutputMint == "" {
				swap.OutputMint = l[len(l)-1].outputMint
			}
		}

		idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
		events = append(events, &DFlowShredInstruction{
			Type:               layout.name,
			Data:               swap,
			Slot:               p.adapter.Slot(),
			Timestamp:          p.adapter.BlockTime(),
			Signature:          p.adapter.Signature(),
			Idx:                idx,
			Signer:             p.adapter.Signers(),
			UnresolvedAccounts: types.HasUnresolvedAccount(accounts),
		})

		inKind := types.ShredAmountExact
		if swap.InputAmountUnknown {
			inKind = types.ShredAmountUnknown
		}
		typed = append(typed, types.ParsedShredInstruction{
			ProgramID:        constants.DEX_PROGRAMS.DFLOW.ID,
			ProgramName:      constants.DEX_PROGRAMS.DFLOW.Name,
			Action:           layout.name,
			Trade:            p.buildTradeInfo(swap),
			Accounts:         accounts,
			Idx:              idx,
			InputAmountKind:  inKind,
			OutputAmountKind: types.ShredAmountQuote,
		})
	}

	return events, typed
}

type dflowSwapKey struct{ outer, inner int }

func swapKey(ci types.ClassifiedInstruction) dflowSwapKey {
	return dflowSwapKey{ci.OuterIndex, ci.InnerIndex}
}

// swapEvents groups the SwapEvent self-CPIs of an executed transaction by the
// swap instruction that emitted them: the nearest preceding DFlow swap of the
// same outer instruction
func (p *DFlowShredParser) swapEvents(instructions []types.ClassifiedInstruction) map[dflowSwapKey][]dflowSwapEvent {
	legs := make(map[dflowSwapKey][]dflowSwapEvent)
	swaps := make(map[int][]int) // outer index -> inner indexes of swaps (-1 = outer)
	for _, ci := range instructions {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) >= 8 {
			if _, ok := dflowSwapLayoutFor(data[:8]); ok {
				swaps[ci.OuterIndex] = append(swaps[ci.OuterIndex], ci.InnerIndex)
			}
		}
	}
	for _, ci := range instructions {
		if ci.InnerIndex < 0 {
			continue
		}
		data := p.adapter.GetInstructionData(ci.Instruction)
		// event: amm, input_mint, input_amount, output_mint, output_amount
		if len(data) < 16+32+32+8+32+8 || !bytes.Equal(data[:16], dflowSwapEventDisc) {
			continue
		}
		owner, found := -2, false
		for _, inner := range swaps[ci.OuterIndex] {
			if inner < ci.InnerIndex && (!found || inner > owner) {
				owner, found = inner, true
			}
		}
		if !found {
			continue
		}
		body := data[16:]
		key := dflowSwapKey{ci.OuterIndex, owner}
		legs[key] = append(legs[key], dflowSwapEvent{
			inputMint:  base58.Encode(body[32:64]),
			outputMint: base58.Encode(body[72:104]),
		})
	}
	return legs
}

// decodeDFlowSwap decodes SwapParams (actions, quoted_out_amount u64,
// slippage_bps u16, platform_fee_bps u16) or Swap2Params (the same and
// positive_slippage_fee_limit_pct u8). Accounts: 3 user_token_authority,
// then for the destination variants 4 destination (token) account and, for
// the non-native ones, 6 destination_mint. Trailing bytes are ignored as the
// program does.
func decodeDFlowSwap(layout dflowSwapLayout, accounts []string, data []byte) *DFlowSwapData {
	minAccounts := 4
	if layout.destinationMint >= 0 {
		minAccounts = layout.destinationMint + 1
	} else if layout.destination >= 0 {
		minAccounts = layout.destination + 1
	}
	if len(accounts) < minAccounts {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	names, amount, hasAmount, ok := decodeDFlowActions(reader)
	if !ok {
		return nil
	}
	swap := &DFlowSwapData{
		User:    accounts[3],
		Actions: names,
	}
	swap.QuotedOutAmount, _ = reader.ReadU64()
	swap.SlippageBps, _ = reader.ReadU16()
	swap.PlatformFeeBps, _ = reader.ReadU16()
	if reader.HasError() {
		return nil
	}
	if layout.swap2 {
		swap.PositiveSlippageFeeLimitPct, _ = reader.ReadU8()
	}

	if hasAmount && amount != math.MaxUint64 {
		swap.InputAmount = amount
	} else {
		swap.InputAmountUnknown = true
	}
	if layout.destination >= 0 {
		swap.DestinationAccount = accounts[layout.destination]
	}
	switch {
	case layout.native:
		swap.OutputMint = constants.TOKENS.SOL
	case layout.destinationMint >= 0 && accounts[layout.destinationMint] != constants.DEX_PROGRAMS.DFLOW.ID:
		// clients pass the program id for an omitted account
		swap.OutputMint = accounts[layout.destinationMint]
	}
	return swap
}

// decodeDFlowActions reads the Vec<Action> of the swap arguments and returns
// the action names and the amount of the first swap action. Every action
// variant has its own layout, so an unknown variant stops the decode (ok
// false).
func decodeDFlowActions(reader *utils.BinaryReader) (names []string, firstAmount uint64, hasAmount bool, ok bool) {
	count, err := reader.ReadVecLength(1)
	if err != nil {
		return nil, 0, false, false
	}
	names = make([]string, 0, count)
	for i := 0; i < count; i++ {
		tag, err := reader.ReadU8()
		if err != nil || int(tag) >= len(dflowActions) {
			return nil, 0, false, false
		}
		action := dflowActions[tag]
		names = append(names, action.name)
		for pi, part := range action.parts {
			switch {
			case part.fixed > 0:
				if pi == action.amountPart && !hasAmount && part.fixed >= action.amountOff+8 {
					b, err := reader.Slice(part.fixed)
					if err != nil {
						return nil, 0, false, false
					}
					firstAmount = binary.LittleEndian.Uint64(b[action.amountOff:])
					hasAmount = true
				}
				if reader.Skip(part.fixed) != nil {
					return nil, 0, false, false
				}
			case part.vecElem > 0:
				n, err := reader.ReadVecLength(part.vecElem)
				if err != nil || reader.Skip(n*part.vecElem) != nil {
					return nil, 0, false, false
				}
			case len(part.vecEnum) > 0:
				n, err := reader.ReadVecLength(1)
				if err != nil {
					return nil, 0, false, false
				}
				for j := 0; j < n; j++ {
					t, err := reader.ReadU8()
					if err != nil || int(t) >= len(part.vecEnum) || reader.Skip(part.vecEnum[t]) != nil {
						return nil, 0, false, false
					}
				}
			}
		}
	}
	return names, firstAmount, hasAmount, true
}

// tokenAccountMint returns the mint of a token account when the transaction
// reveals it, "" otherwise (never a guess)
func (p *DFlowShredParser) tokenAccountMint(account string) string {
	if p.adapter.IsGuessedTokenAccount(account) {
		return ""
	}
	return p.adapter.GetSplTokenMint(account)
}

// decimals returns the decimals of mint when known, 0 (unknown) otherwise
func (p *DFlowShredParser) decimals(mint string) uint8 {
	if mint == "" {
		return 0
	}
	if d, ok := p.adapter.SPLDecimalsMap[mint]; ok {
		return d
	}
	return constants.TOKEN_DECIMALS[mint]
}

func (p *DFlowShredParser) buildTradeInfo(swap *DFlowSwapData) *types.TradeInfo {
	slippageBps := int(swap.SlippageBps)
	inDecimals, outDecimals := p.decimals(swap.InputMint), p.decimals(swap.OutputMint)

	tradeType := types.TradeTypeSwap
	switch {
	case swap.InputMint != "" && swap.InputMint == swap.OutputMint:
		// a circular (arbitrage) route buys nothing: SWAP
	case swap.InputMint != "" && swap.OutputMint != "":
		tradeType = utils.GetTradeType(swap.InputMint, swap.OutputMint)
	case constants.IsQuoteToken(swap.InputMint):
		tradeType = types.TradeTypeBuy
	case constants.IsQuoteToken(swap.OutputMint):
		tradeType = types.TradeTypeSell
	}

	return &types.TradeInfo{
		Type: tradeType,
		Pool: []string{},
		User: swap.User,
		InputToken: types.TokenInfo{
			Mint:      swap.InputMint,
			Amount:    types.ConvertToUIAmountUint64(swap.InputAmount, inDecimals),
			AmountRaw: strconv.FormatUint(swap.InputAmount, 10),
			Decimals:  inDecimals,
		},
		OutputToken: types.TokenInfo{
			Mint:        swap.OutputMint,
			Amount:      types.ConvertToUIAmountUint64(swap.QuotedOutAmount, outDecimals),
			AmountRaw:   strconv.FormatUint(swap.QuotedOutAmount, 10),
			Decimals:    outDecimals,
			Destination: swap.DestinationAccount,
		},
		ProgramId:   constants.DEX_PROGRAMS.DFLOW.ID,
		AMM:         constants.DEX_PROGRAMS.DFLOW.Name,
		Route:       constants.DEX_PROGRAMS.DFLOW.Name,
		SlippageBps: &slippageBps,
	}
}

// dflowPart is a piece of an action's Borsh layout
type dflowPart struct {
	// fixed is a fixed number of bytes
	fixed int
	// vecElem > 0 is a Vec (u32 length) of fixed-size elements
	vecElem int
	// vecEnum is a Vec of enums; the payload size of each variant by tag
	vecEnum []int
}

func dflowFixed(n int) dflowPart          { return dflowPart{fixed: n} }
func dflowVec(elem int) dflowPart         { return dflowPart{vecElem: elem} }
func dflowVecEnum(sizes ...int) dflowPart { return dflowPart{vecEnum: sizes} }

// dflowAction is the layout of one Action variant. amountPart is the index of
// the fixed part holding the swap's input amount (u64 at amountOff); -1 for
// actions that are not swaps.
type dflowAction struct {
	name       string
	parts      []dflowPart
	amountPart int
	amountOff  int
}

// dflowActions lists the Action variants by tag, generated from the on-chain
// swap_orchestrator IDL (account Cp2dCjxCWdktak2JiSrh87X6sz31EnDVKoTGtsHJvhYq,
// version 0.1.0, fetched 2026-09-27)
var dflowActions = []dflowAction{
	{"WhirlpoolsSwap", []dflowPart{dflowFixed(10)}, 0, 0},                                                          // 0
	{"ClearpoolsSwap", []dflowPart{dflowFixed(10)}, 0, 0},                                                          // 1
	{"RaydiumAmmSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                           // 2
	{"LifinityV2Swap", []dflowPart{dflowFixed(9)}, 0, 0},                                                           // 3
	{"MeteoraDlmmSwap", []dflowPart{dflowFixed(10)}, 0, 0},                                                         // 4
	{"RaydiumClmmSwap", []dflowPart{dflowFixed(10)}, 0, 0},                                                         // 5
	{"RaydiumClmmSwapV2", []dflowPart{dflowFixed(10)}, 0, 0},                                                       // 6
	{"PhoenixSwap", []dflowPart{dflowFixed(10)}, 0, 0},                                                             // 7
	{"PumpFunBuy", []dflowPart{dflowFixed(9)}, 0, 0},                                                               // 8
	{"PumpFunSell", []dflowPart{dflowFixed(9)}, 0, 0},                                                              // 9
	{"GammaSwap", []dflowPart{dflowFixed(10)}, 0, 0},                                                               // 10
	{"ObricV2Swap", []dflowPart{dflowFixed(9)}, 0, 0},                                                              // 11
	{"PumpFunAmmBuy", []dflowPart{dflowFixed(9)}, 0, 0},                                                            // 12
	{"PumpFunAmmSell", []dflowPart{dflowFixed(9)}, 0, 0},                                                           // 13
	{"SolFiSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                                // 14
	{"RubiconSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                              // 15
	{"MeteoraDammV1Swap", []dflowPart{dflowFixed(9)}, 0, 0},                                                        // 16
	{"RaydiumCpSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                            // 17
	{"StabbleStableSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                        // 18
	{"TesseraVSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                             // 19
	{"MeteoraDammV2Swap", []dflowPart{dflowFixed(9)}, 0, 0},                                                        // 20
	{"RaydiumLaunchlabSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                     // 21
	{"MeteoraDbcSwap", []dflowPart{dflowFixed(10)}, 0, 0},                                                          // 22
	{"HumidiFiSwap", []dflowPart{dflowFixed(17)}, 0, 0},                                                            // 23
	{"WhirlpoolsSwapV2", []dflowPart{dflowFixed(10)}, 0, 0},                                                        // 24
	{"MeteoraDlmmSwapV2", []dflowPart{dflowFixed(10)}, 0, 0},                                                       // 25
	{"ZeroFiSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                               // 26
	{"AlphaQSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                               // 27
	{"TokenSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                                // 28
	{"SolFiV2Swap", []dflowPart{dflowFixed(9)}, 0, 0},                                                              // 29
	{"MozartSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                               // 30
	{"DFlowDynamicRouteV1", []dflowPart{dflowVecEnum(0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 1, 0, 1), dflowFixed(9)}, 1, 0}, // 31
	{"HeavenSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                               // 32
	{"NexusSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                                // 33
	{"SarosDlmmSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                            // 34
	{"TransferFee", []dflowPart{dflowFixed(8)}, -1, 0},                                                             // 35
	{"TransferFeeWithMint", []dflowPart{dflowFixed(8)}, -1, 0},                                                     // 36
	{"RecordId", []dflowPart{dflowFixed(76)}, -1, 0},                                                               // 37
	{"RecordId2", []dflowPart{dflowFixed(4)}, -1, 0},                                                               // 38
	{"ManifestSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                             // 39
	{"BisonFiSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                              // 40
	{"SanctumInfinitySwap", []dflowPart{dflowFixed(19)}, 0, 0},                                                     // 41
	{"SanctumInfinityLiquidity", []dflowPart{dflowFixed(14)}, -1, 0},                                               // 42
	{"OpenPredictionsOrder", []dflowPart{dflowFixed(53)}, -1, 0},                                                   // 43
	{"ScorchSwap", []dflowPart{dflowFixed(25)}, 0, 0},                                                              // 44
	{"IncludeAccount", []dflowPart{}, -1, 0},                                                                       // 45
	{"StabbleWeightedSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                      // 46
	{"VertigoSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                              // 47
	{"SetMinimumLegOutputs", []dflowPart{dflowVec(8)}, -1, 0},                                                      // 48
	{"SetMinimumLegPrices", []dflowPart{dflowVec(9)}, -1, 0},                                                       // 49
	{"SetSponsor", []dflowPart{}, -1, 0},                                                                           // 50
	{"WrapSol", []dflowPart{dflowFixed(8)}, -1, 0},                                                                 // 51
	{"UnwrapSol", []dflowPart{}, -1, 0},                                                                            // 52
	{"KDEXSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                                 // 53
	{"DeriverseSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                            // 54
	{"VaultSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                                // 55
	{"MetaDaoSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                              // 56
	{"D", []dflowPart{dflowVec(2), dflowFixed(13)}, 1, 0},                                                          // 57
	{"XoCashExchangeSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                       // 58
	{"LemmingsFiSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                           // 59
	{"InitAtaIdempotent", []dflowPart{}, -1, 0},                                                                    // 60
	{"SetMaxUnderconsumptionBps", []dflowPart{dflowFixed(2)}, -1, 0},                                               // 61
	{"GhostSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                                // 62
	{"TransferFeeV2", []dflowPart{dflowFixed(8)}, -1, 0},                                                           // 63
	{"TransferFeeWithMintV2", []dflowPart{dflowFixed(8)}, -1, 0},                                                   // 64
	{"PumpFunBuyV2", []dflowPart{dflowFixed(9)}, 0, 0},                                                             // 65
	{"PumpFunSellV2", []dflowPart{dflowFixed(9)}, 0, 0},                                                            // 66
	{"DopplerLaunchSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                        // 67
	{"DopplerCpmmSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                          // 68
	{"ByrealClmmSwapV3Dyn", []dflowPart{dflowFixed(10)}, 0, 0},                                                     // 69
	{"SanctumStakedexStakeWSOL", []dflowPart{dflowFixed(9)}, 0, 0},                                                 // 70
	{"SanctumStakedexWithdrawWSOL", []dflowPart{dflowFixed(9)}, 0, 0},                                              // 71
	{"SanctumStakedexPrefund", []dflowPart{dflowFixed(13)}, 0, 0},                                                  // 72
	{"SanctumStakedexReserve", []dflowPart{dflowFixed(13)}, 0, 0},                                                  // 73
	{"SuperisSwap", []dflowPart{dflowFixed(9)}, 0, 0},                                                              // 74
	{"GatorSwapSwap", []dflowPart{dflowFixed(82)}, 0, 0},                                                           // 75
	{"JupLendAmmSwap", []dflowPart{dflowFixed(10)}, 0, 0},                                                          // 76
	{"C", []dflowPart{dflowFixed(16)}, -1, 0},                                                                      // 77
	{"SetSponsorIntermediateCloseAuthority", []dflowPart{}, -1, 0},                                                 // 78
}
