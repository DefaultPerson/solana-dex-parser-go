package photon

import (
	"bytes"
	"strconv"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// Photon (BSfD6SHZ...) has no published IDL. The pump_buy_v2, pump_sell_v2,
// collect_fee and moonshot_sell layouts below were verified against real
// transactions (arguments forwarded unchanged to the inner Pump.fun /
// Moonshot instruction). pump_buy, pump_sell and pump_amm_swap are no longer
// in the deployed program; their decoders, and the moonshot_buy and
// two_hop_swap decoders, are kept for historical transactions but their
// layouts could not be checked against a real transaction.

// PhotonShredParser parses Photon instructions from shred-stream
type PhotonShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewPhotonShredParser creates a new PhotonShredParser
func NewPhotonShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *PhotonShredParser {
	return &PhotonShredParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes Photon instructions and returns parsed results
func (p *PhotonShredParser) ProcessInstructions() []interface{} {
	events, _ := p.ProcessAll()
	return events
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *PhotonShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	_, typed := p.ProcessAll()
	return typed
}

// ProcessAll decodes the Photon instructions into legacy events and typed
// trades in a single pass
func (p *PhotonShredParser) ProcessAll() ([]interface{}, []types.ParsedShredInstruction) {
	var events []interface{}
	var typed []types.ParsedShredInstruction
	d := constants.DISCRIMINATORS.PHOTON

	for _, ci := range p.classifier.GetInstructions(constants.DEX_PROGRAMS.PHOTON.ID) {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}
		accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
		disc, payload := data[:8], data[8:]

		var eventType string
		var eventData interface{}
		var ins *types.ParsedShredInstruction

		switch {
		case bytes.Equal(disc, d.PUMPSWAP_TRADE):
			if swap := p.decodePhotonSwapData(accounts, payload); swap != nil {
				eventType, eventData, ins = "pumpswap_swap", swap, p.swapInstruction(swap)
			}
		case bytes.Equal(disc, d.PUMPFUN_BUY):
			if buy := p.decodePhotonPumpfunBuyData(accounts, payload); buy != nil {
				eventType, eventData, ins = "pumpfun_buy", buy, p.pumpfunInstruction(buy, types.ShredAmountUnknown, types.ShredAmountUnknown)
			}
		case bytes.Equal(disc, d.PUMPFUN_SELL):
			if sell := p.decodePhotonPumpfunSellData(accounts, payload); sell != nil {
				eventType, eventData, ins = "pumpfun_sell", sell, p.pumpfunInstruction(sell, types.ShredAmountUnknown, types.ShredAmountUnknown)
			}
		case bytes.Equal(disc, d.PUMPFUN_BUY_V2):
			if buy := p.decodePhotonPumpBuyV2Data(accounts, payload); buy != nil {
				eventType, eventData, ins = "pumpfun_buy_v2", buy, p.pumpfunInstruction(buy, types.ShredAmountExact, types.ShredAmountMin)
			}
		case bytes.Equal(disc, d.PUMPFUN_SELL_V2):
			if sell := p.decodePhotonPumpSellV2Data(accounts, payload); sell != nil {
				eventType, eventData, ins = "pumpfun_sell_v2", sell, p.pumpfunInstruction(sell, types.ShredAmountExact, types.ShredAmountMin)
			}
		case bytes.Equal(disc, d.MOONIT_BUY):
			if buy := p.decodePhotonMoonitData(accounts, payload, "buy"); buy != nil {
				// [unverified layout] both amounts are Moonshot trade parameters
				eventType, eventData, ins = "moonit_buy", buy, p.moonitInstruction(buy, types.ShredAmountQuote, types.ShredAmountQuote)
			}
		case bytes.Equal(disc, d.MOONIT_SELL):
			if sell := p.decodePhotonMoonitData(accounts, payload, "sell"); sell != nil {
				eventType, eventData, ins = "moonit_sell", sell, p.moonitInstruction(sell, types.ShredAmountExact, types.ShredAmountQuote)
			}
		case bytes.Equal(disc, d.HOP_TWO_SWAP):
			if hop := p.decodePhotonHopTwoSwapData(accounts, payload); hop != nil {
				eventType, eventData, ins = "hop_two_swap", hop, p.hopTwoSwapInstruction(hop)
			}
		case bytes.Equal(disc, d.COLLECT_FEE):
			if fee := p.decodePhotonCollectFeeData(accounts, payload); fee != nil {
				eventType, eventData = "collect_fee", fee
				ins = &types.ParsedShredInstruction{Action: eventType, Data: fee}
			}
		default:
			continue
		}

		if eventData == nil {
			continue
		}
		idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
		events = append(events, &PhotonInstruction{
			Type:               eventType,
			Data:               eventData,
			Slot:               p.adapter.Slot(),
			Timestamp:          p.adapter.BlockTime(),
			Signature:          p.adapter.Signature(),
			Idx:                idx,
			Signer:             p.adapter.Signers(),
			UnresolvedAccounts: types.HasUnresolvedAccount(accounts),
		})
		ins.ProgramID = constants.DEX_PROGRAMS.PHOTON.ID
		ins.ProgramName = constants.DEX_PROGRAMS.PHOTON.Name
		ins.Accounts = accounts
		ins.Idx = idx
		typed = append(typed, *ins)
	}

	return events, typed
}

// PhotonInstruction represents a parsed Photon instruction
type PhotonInstruction struct {
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

// PhotonSwapData contains Photon PumpSwap swap (pump_amm_swap) instruction
// data. TradeType is the pool side: BUY (quote -> base), SELL (base -> quote)
// or SWAP when the direction cannot be determined. The typed trade's Type is
// relative to SOL or a stablecoin instead (utils.GetPoolSideTradeType), which
// differs in pools whose base is WSOL.
type PhotonSwapData struct {
	Pool               string `json:"pool"`
	User               string `json:"user"`
	BaseMint           string `json:"baseMint"`
	QuoteMint          string `json:"quoteMint"`
	InputTokenAccount  string `json:"inputTokenAccount"`
	OutputTokenAccount string `json:"outputTokenAccount"`
	InputAmount        uint64 `json:"inputAmount"`
	OutputAmount       uint64 `json:"outputAmount"`
	TradeType          string `json:"tradeType"`
	TargetProgram      string `json:"targetProgram"`
}

// PhotonPumpfunData contains Photon Pump.fun instruction data.
//
// pump_buy_v2: InputAmount is the exact quote amount sent to Pump.fun
// (spendable_quote_in, excluding the Photon fee), OutputAmount the minimum
// tokens out, PhotonFee the Photon fee in lamports. pump_sell_v2:
// InputAmount is the exact token amount sold, OutputAmount the minimum quote
// output, PhotonFeeBps the Photon fee rate. The legacy pump_buy / pump_sell
// layout (timestamp first) is unverified.
type PhotonPumpfunData struct {
	Pool          string `json:"pool"`
	User          string `json:"user"`
	BaseMint      string `json:"baseMint"`
	Timestamp     int64  `json:"timestamp"`
	InputAmount   uint64 `json:"inputAmount"`
	OutputAmount  uint64 `json:"outputAmount"`
	TradeType     string `json:"tradeType"`
	TargetProgram string `json:"targetProgram"`
	// QuoteMint is the curve's quote mint (WSOL for the legacy instructions)
	QuoteMint string `json:"quoteMint,omitempty"`
	// PhotonFee is the Photon fee in lamports (pump_buy_v2)
	PhotonFee uint64 `json:"photonFee,omitempty"`
	// PhotonFeeBps is the Photon fee rate (pump_sell_v2)
	PhotonFeeBps uint64 `json:"photonFeeBps,omitempty"`
	// FeeVault is the Photon fee vault receiving the fee
	FeeVault string `json:"feeVault,omitempty"`
}

// PhotonMoonitData contains Photon Moonshot (Moonit) instruction data:
// timestamp, token_amount, collateral_amount (SOL), slippage_bps and
// photon_fee_bps. For moonshot_sell (verified) TokenAmount is the exact
// token input and CollateralAmount the quoted SOL output; the moonshot_buy
// layout is assumed to be the same (unverified).
type PhotonMoonitData struct {
	Pool      string `json:"pool"`
	User      string `json:"user"`
	BaseMint  string `json:"baseMint"`
	Timestamp int64  `json:"timestamp"`
	// InputAmount and OutputAmount are the trade's input and output amounts:
	// TokenAmount and CollateralAmount in trade direction
	InputAmount   uint64 `json:"inputAmount"`
	OutputAmount  uint64 `json:"outputAmount"`
	SlippageBps   uint64 `json:"slippageBps"`
	TradeType     string `json:"tradeType"`
	TargetProgram string `json:"targetProgram"`
	// TokenAmount and CollateralAmount are the Moonshot trade parameters
	TokenAmount      uint64 `json:"tokenAmount"`
	CollateralAmount uint64 `json:"collateralAmount"`
	// PhotonFeeBps is the Photon fee rate
	PhotonFeeBps uint64 `json:"photonFeeBps,omitempty"`
}

// PhotonHopTwoSwapData contains Photon hop two swap instruction data
type PhotonHopTwoSwapData struct {
	User         string   `json:"user"`
	Pools        []string `json:"pools"`
	InputMint    string   `json:"inputMint"`
	OutputMint   string   `json:"outputMint"`
	InputAmount  uint64   `json:"inputAmount"`
	OutputAmount uint64   `json:"outputAmount"`
	TradeType    string   `json:"tradeType"`
	Programs     []string `json:"programs"`
}

// PhotonCollectFeeData contains Photon collect_fee instruction data. The
// trade itself is a separate venue instruction of the same transaction.
// Mode 0: Amount is the fee in lamports; mode 1: Amount is a fee rate in bps
// of the SOL in the temporary WSOL account.
type PhotonCollectFeeData struct {
	User      string `json:"user"`
	FeeVault  string `json:"feeVault"`
	Timestamp int64  `json:"timestamp"`
	Amount    uint64 `json:"amount"`
	Mode      uint8  `json:"mode"`
}

// pumpBaseDecimals is the decimals of every Pump.fun bonding-curve mint
const pumpBaseDecimals = 6

func tokenInfo(mint string, amount uint64, decimals uint8) types.TokenInfo {
	return types.TokenInfo{
		Mint:      mint,
		Amount:    types.ConvertToUIAmountUint64(amount, decimals),
		AmountRaw: strconv.FormatUint(amount, 10),
		Decimals:  decimals,
	}
}

// decodePhotonSwapData decodes pump_amm_swap (input_amount, output_amount):
// accounts 0 pool, 1 user, 3 base_mint, 4 quote_mint, 5 input token account,
// 6 output token account, 16 PumpSwap program
func (p *PhotonShredParser) decodePhotonSwapData(accounts []string, data []byte) *PhotonSwapData {
	if len(accounts) < 17 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	swap := &PhotonSwapData{
		Pool:               accounts[0],
		User:               accounts[1],
		BaseMint:           accounts[3],
		QuoteMint:          accounts[4],
		InputTokenAccount:  accounts[5],
		OutputTokenAccount: accounts[6],
		InputAmount:        inputAmount,
		OutputAmount:       outputAmount,
		TargetProgram:      accounts[16],
	}
	swap.TradeType = string(p.swapDirection(swap))
	return swap
}

// swapDirection determines BUY (quote in) or SELL (base in) from the mints
// of the token accounts, then from the user's associated token account of
// the base mint; SWAP when neither decides
func (p *PhotonShredParser) swapDirection(swap *PhotonSwapData) types.TradeType {
	switch p.adapter.KnownTokenAccountMint(swap.InputTokenAccount) {
	case "":
	case swap.BaseMint:
		return types.TradeTypeSell
	case swap.QuoteMint:
		return types.TradeTypeBuy
	}
	switch p.adapter.KnownTokenAccountMint(swap.OutputTokenAccount) {
	case "":
	case swap.BaseMint:
		return types.TradeTypeBuy
	case swap.QuoteMint:
		return types.TradeTypeSell
	}
	if swap.User == "" || swap.BaseMint == "" {
		return types.TradeTypeSwap
	}
	return utils.GetAccountTradeType(swap.User, swap.BaseMint, swap.InputTokenAccount, swap.OutputTokenAccount)
}

func (p *PhotonShredParser) swapInstruction(swap *PhotonSwapData) *types.ParsedShredInstruction {
	// swap.TradeType is the pool side (BUY: quote in); the trade's type is
	// relative to SOL or a stablecoin, which differs in pools whose base is WSOL
	var inputMint, outputMint string
	tradeType := types.TradeType(swap.TradeType)
	switch tradeType {
	case types.TradeTypeSell:
		inputMint, outputMint = swap.BaseMint, swap.QuoteMint
		tradeType = utils.GetPoolSideTradeType(false, swap.BaseMint, swap.QuoteMint)
	case types.TradeTypeBuy:
		inputMint, outputMint = swap.QuoteMint, swap.BaseMint
		tradeType = utils.GetPoolSideTradeType(true, swap.BaseMint, swap.QuoteMint)
	}

	return &types.ParsedShredInstruction{
		Action: "pumpswap_swap",
		Trade: &types.TradeInfo{
			Type:        tradeType,
			Pool:        []string{swap.Pool},
			User:        swap.User,
			InputToken:  tokenInfo(inputMint, swap.InputAmount, p.adapter.GetTokenDecimals(inputMint)),
			OutputToken: tokenInfo(outputMint, swap.OutputAmount, p.adapter.GetTokenDecimals(outputMint)),
			ProgramId:   swap.TargetProgram,
			AMM:         utils.GetProgramName(swap.TargetProgram),
			AMMs:        []string{utils.GetProgramName(swap.TargetProgram)},
			Route:       constants.DEX_PROGRAMS.PHOTON.Name,
		},
		InputAmountKind:  types.ShredAmountUnknown,
		OutputAmountKind: types.ShredAmountUnknown,
	}
}

// decodePhotonPumpfunSellData decodes the legacy pump_sell (timestamp i64,
// input, output): accounts 2 mint, 3 bonding curve, 6 user, 9 Pump.fun
func (p *PhotonShredParser) decodePhotonPumpfunSellData(accounts []string, data []byte) *PhotonPumpfunData {
	if len(accounts) < 12 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	timestamp, _ := reader.ReadI64()
	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PhotonPumpfunData{
		Pool:          accounts[3],
		User:          accounts[6],
		BaseMint:      accounts[2],
		QuoteMint:     constants.TOKENS.SOL,
		Timestamp:     timestamp,
		InputAmount:   inputAmount,
		OutputAmount:  outputAmount,
		TradeType:     "sell",
		TargetProgram: accounts[9],
	}
}

// decodePhotonPumpfunBuyData decodes the legacy pump_buy (timestamp i64,
// input, output): accounts 3 mint, 4 bonding curve, 7 user, 9 Pump.fun
func (p *PhotonShredParser) decodePhotonPumpfunBuyData(accounts []string, data []byte) *PhotonPumpfunData {
	if len(accounts) < 12 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	timestamp, _ := reader.ReadI64()
	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PhotonPumpfunData{
		Pool:          accounts[4],
		User:          accounts[7],
		BaseMint:      accounts[3],
		QuoteMint:     constants.TOKENS.SOL,
		Timestamp:     timestamp,
		InputAmount:   inputAmount,
		OutputAmount:  outputAmount,
		TradeType:     "buy",
		TargetProgram: accounts[9],
	}
}

// decodePhotonPumpBuyV2Data decodes pump_buy_v2 (spendable_quote_in,
// min_tokens_out, photon_fee_lamports, flag u8): accounts 0 user, 6 Photon
// fee vault, 7 Pump.fun, 9 base_mint, 10 quote_mint, 16 bonding_curve
func (p *PhotonShredParser) decodePhotonPumpBuyV2Data(accounts []string, data []byte) *PhotonPumpfunData {
	if len(accounts) < 17 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	spendableQuoteIn, _ := reader.ReadU64()
	minTokensOut, _ := reader.ReadU64()
	photonFee, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PhotonPumpfunData{
		Pool:          accounts[16],
		User:          accounts[0],
		BaseMint:      accounts[9],
		QuoteMint:     accounts[10],
		InputAmount:   spendableQuoteIn,
		OutputAmount:  minTokensOut,
		TradeType:     "buy",
		TargetProgram: accounts[7],
		PhotonFee:     photonFee,
		FeeVault:      accounts[6],
	}
}

// decodePhotonPumpSellV2Data decodes pump_sell_v2 (amount, min_sol_output,
// photon_fee_bps, flag u8) with the pump_buy_v2 account layout
func (p *PhotonShredParser) decodePhotonPumpSellV2Data(accounts []string, data []byte) *PhotonPumpfunData {
	if len(accounts) < 17 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	amount, _ := reader.ReadU64()
	minSolOutput, _ := reader.ReadU64()
	photonFeeBps, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &PhotonPumpfunData{
		Pool:          accounts[16],
		User:          accounts[0],
		BaseMint:      accounts[9],
		QuoteMint:     accounts[10],
		InputAmount:   amount,
		OutputAmount:  minSolOutput,
		TradeType:     "sell",
		TargetProgram: accounts[7],
		PhotonFeeBps:  photonFeeBps,
		FeeVault:      accounts[6],
	}
}

func (p *PhotonShredParser) pumpfunInstruction(d *PhotonPumpfunData, inKind, outKind types.ShredAmountKind) *types.ParsedShredInstruction {
	quote := tokenInfo(d.QuoteMint, 0, p.adapter.GetTokenDecimals(d.QuoteMint))
	baseDecimals, ok := p.adapter.KnownDecimals(d.BaseMint)
	if !ok {
		baseDecimals = pumpBaseDecimals
	}
	base := tokenInfo(d.BaseMint, 0, baseDecimals)
	tradeType := types.TradeTypeBuy
	input, output := quote, base
	if d.TradeType == "sell" {
		tradeType = types.TradeTypeSell
		input, output = base, quote
	}
	input = tokenInfo(input.Mint, d.InputAmount, input.Decimals)
	output = tokenInfo(output.Mint, d.OutputAmount, output.Decimals)

	trade := &types.TradeInfo{
		Type:        tradeType,
		Pool:        []string{d.Pool},
		User:        d.User,
		InputToken:  input,
		OutputToken: output,
		ProgramId:   d.TargetProgram,
		AMM:         utils.GetProgramName(d.TargetProgram),
		AMMs:        []string{utils.GetProgramName(d.TargetProgram)},
		Route:       constants.DEX_PROGRAMS.PHOTON.Name,
		Timestamp:   d.Timestamp,
	}
	if d.PhotonFee > 0 {
		trade.Fee = &types.FeeInfo{
			Mint:      constants.TOKENS.SOL,
			Amount:    types.ConvertToUIAmountUint64(d.PhotonFee, 9),
			AmountRaw: strconv.FormatUint(d.PhotonFee, 10),
			Decimals:  9,
			Dex:       constants.DEX_PROGRAMS.PHOTON.Name,
			Type:      "platform",
			Recipient: d.FeeVault,
		}
	}

	return &types.ParsedShredInstruction{
		Action:           "pumpfun_" + d.TradeType,
		Trade:            trade,
		InputAmountKind:  inKind,
		OutputAmountKind: outKind,
	}
}

// decodePhotonMoonitData decodes moonshot_buy / moonshot_sell (timestamp i64,
// token_amount, collateral_amount, slippage_bps, photon_fee_bps): accounts
// 0 user, 2 curve, 7 mint, 9 Moonshot program
func (p *PhotonShredParser) decodePhotonMoonitData(accounts []string, data []byte, tradeType string) *PhotonMoonitData {
	if len(accounts) < 12 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	timestamp, _ := reader.ReadI64()
	tokenAmount, _ := reader.ReadU64()
	collateralAmount, _ := reader.ReadU64()
	slippageBps, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}
	photonFeeBps, _ := reader.ReadU64()

	moonit := &PhotonMoonitData{
		Pool:             accounts[2],
		User:             accounts[0],
		BaseMint:         accounts[7],
		Timestamp:        timestamp,
		SlippageBps:      slippageBps,
		TradeType:        tradeType,
		TargetProgram:    accounts[9],
		TokenAmount:      tokenAmount,
		CollateralAmount: collateralAmount,
		PhotonFeeBps:     photonFeeBps,
	}
	if tradeType == "buy" {
		moonit.InputAmount, moonit.OutputAmount = collateralAmount, tokenAmount
	} else {
		moonit.InputAmount, moonit.OutputAmount = tokenAmount, collateralAmount
	}
	return moonit
}

func (p *PhotonShredParser) moonitInstruction(d *PhotonMoonitData, inKind, outKind types.ShredAmountKind) *types.ParsedShredInstruction {
	sol := constants.TOKENS.SOL
	tradeType := types.TradeTypeBuy
	input := tokenInfo(sol, d.InputAmount, 9)
	output := tokenInfo(d.BaseMint, d.OutputAmount, p.adapter.GetTokenDecimals(d.BaseMint))
	if d.TradeType == "sell" {
		tradeType = types.TradeTypeSell
		input = tokenInfo(d.BaseMint, d.InputAmount, p.adapter.GetTokenDecimals(d.BaseMint))
		output = tokenInfo(sol, d.OutputAmount, 9)
	}
	slippageBps := int(d.SlippageBps)

	return &types.ParsedShredInstruction{
		Action: "moonit_" + d.TradeType,
		Trade: &types.TradeInfo{
			Type:        tradeType,
			Pool:        []string{d.Pool},
			User:        d.User,
			InputToken:  input,
			OutputToken: output,
			ProgramId:   d.TargetProgram,
			AMM:         utils.GetProgramName(d.TargetProgram),
			AMMs:        []string{utils.GetProgramName(d.TargetProgram)},
			Route:       constants.DEX_PROGRAMS.PHOTON.Name,
			Timestamp:   d.Timestamp,
			SlippageBps: &slippageBps,
		},
		InputAmountKind:  inKind,
		OutputAmountKind: outKind,
	}
}

// decodePhotonCollectFeeData decodes collect_fee (timestamp i64, amount u64,
// mode u8): accounts 0 user, 1 temporary WSOL account, 2 Photon fee vault
func (p *PhotonShredParser) decodePhotonCollectFeeData(accounts []string, data []byte) *PhotonCollectFeeData {
	if len(accounts) < 3 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	timestamp, _ := reader.ReadI64()
	amount, _ := reader.ReadU64()
	mode, _ := reader.ReadU8()
	if reader.HasError() {
		return nil
	}

	return &PhotonCollectFeeData{
		User:      accounts[0],
		FeeVault:  accounts[2],
		Timestamp: timestamp,
		Amount:    amount,
		Mode:      mode,
	}
}

// decodePhotonHopTwoSwapData decodes two_hop_swap (input, output) for the
// Raydium V4 -> Meteora and Meteora DBC -> Raydium V4 routes (unverified
// account layout)
func (p *PhotonShredParser) decodePhotonHopTwoSwapData(accounts []string, data []byte) *PhotonHopTwoSwapData {
	if len(accounts) < 12 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	inputAmount, _ := reader.ReadU64()
	outputAmount, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	userAccount := accounts[0]
	program1 := accounts[6]

	var inputMint, outputMint string
	var pools []string
	var programs []string
	var tradeType string

	switch program1 {
	case constants.DEX_PROGRAMS.RAYDIUM_V4.ID:
		// Raydium V4 -> Meteora (Buy)
		tradeType = "buy"
		program2 := accounts[11]
		programs = []string{utils.GetProgramName(program1), utils.GetProgramName(program2)}

		switch program2 {
		case constants.DEX_PROGRAMS.METEORA_DBC.ID:
			if len(accounts) < 18 {
				return nil
			}
			pools = []string{accounts[7], accounts[14]}
			inputMint = constants.TOKENS.SOL
			outputMint = accounts[17]
		case constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID:
			if len(accounts) < 17 {
				return nil
			}
			pools = []string{accounts[7], accounts[13]}
			inputMint = constants.TOKENS.SOL
			outputMint = accounts[16]
		case constants.DEX_PROGRAMS.METEORA.ID:
			if len(accounts) < 17 {
				return nil
			}
			pools = []string{accounts[7], accounts[12]}
			inputMint = constants.TOKENS.SOL
			outputMint = accounts[16]
		default:
			return nil
		}
	case constants.DEX_PROGRAMS.METEORA_DBC.ID:
		// Meteora DBC -> Raydium V4 (Sell)
		if len(accounts) < 18 {
			return nil
		}
		tradeType = "sell"
		pools = []string{accounts[9], accounts[17]}
		inputMint = accounts[12]
		outputMint = constants.TOKENS.SOL
		program2 := accounts[16]
		programs = []string{utils.GetProgramName(program1), utils.GetProgramName(program2)}
	default:
		return nil
	}

	return &PhotonHopTwoSwapData{
		User:         userAccount,
		Pools:        pools,
		InputMint:    inputMint,
		OutputMint:   outputMint,
		InputAmount:  inputAmount,
		OutputAmount: outputAmount,
		TradeType:    tradeType,
		Programs:     programs,
	}
}

func (p *PhotonShredParser) hopTwoSwapInstruction(swapData *PhotonHopTwoSwapData) *types.ParsedShredInstruction {
	tradeType := types.TradeTypeBuy
	if swapData.TradeType == "sell" {
		tradeType = types.TradeTypeSell
	}

	return &types.ParsedShredInstruction{
		Action: "hop_two_swap",
		Trade: &types.TradeInfo{
			Type:        tradeType,
			Pool:        swapData.Pools,
			User:        swapData.User,
			InputToken:  tokenInfo(swapData.InputMint, swapData.InputAmount, p.adapter.GetTokenDecimals(swapData.InputMint)),
			OutputToken: tokenInfo(swapData.OutputMint, swapData.OutputAmount, p.adapter.GetTokenDecimals(swapData.OutputMint)),
			ProgramId:   constants.DEX_PROGRAMS.PHOTON.ID,
			AMMs:        swapData.Programs,
			Route:       constants.DEX_PROGRAMS.PHOTON.Name,
		},
		InputAmountKind:  types.ShredAmountUnknown,
		OutputAmountKind: types.ShredAmountUnknown,
	}
}
