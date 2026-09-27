package meme

import (
	"bytes"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// MoonitEventParser parses Moonit meme coin events
type MoonitEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
	utils           *utils.TransactionUtils
	logEvents       map[[2]int]*moonitTradeEvent
}

// NewMoonitEventParser creates a new Moonit event parser
func NewMoonitEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *MoonitEventParser {
	return &MoonitEventParser{
		adapter:         adapter,
		transferActions: transferActions,
		utils:           utils.NewTransactionUtils(adapter),
	}
}

// ProcessEvents implements the EventParser interface
func (p *MoonitEventParser) ProcessEvents() []types.MemeEvent {
	instructions := utils.ProgramInstructions(p.adapter,
		constants.DEX_PROGRAMS.MOONIT.ID,
		constants.METAPLEX_PROGRAM_ID,
	)
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}

// ParseInstructions parses classified instructions into meme events, in
// execution order
func (p *MoonitEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	ordered := append([]types.ClassifiedInstruction(nil), instructions...)
	utils.SortInstructionsByExecution(ordered)

	for _, ci := range ordered {
		if ci.ProgramId != constants.DEX_PROGRAMS.MOONIT.ID {
			continue
		}
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		disc := data[:8]
		var event *types.MemeEvent

		switch {
		case bytes.Equal(disc, constants.DISCRIMINATORS.MOONIT.BUY):
			event = p.decodeTradeEvent(data[8:], ci, types.TradeTypeBuy)
		case bytes.Equal(disc, constants.DISCRIMINATORS.MOONIT.SELL):
			event = p.decodeTradeEvent(data[8:], ci, types.TradeTypeSell)
		case bytes.Equal(disc, constants.DISCRIMINATORS.MOONIT.CREATE):
			event = p.decodeCreateEvent(data[8:], ci)
		case bytes.Equal(disc, constants.DISCRIMINATORS.MOONIT.MIGRATE):
			event = p.decodeMigrateEvent(data[8:], ci)
		}

		if event != nil {
			event.Signature = p.adapter.Signature()
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
			events = append(events, event)
		}
	}

	return events
}

// moonitTradeEvent is Moonit's TradeEvent (token_launchpad IDL), emitted as
// a "Program data:" log only: amount, collateral_amount, dex_fee, helio_fee,
// allocation, curve, cost_token, sender, type (0 buy, 1 sell), label
type moonitTradeEvent struct {
	Amount           uint64
	CollateralAmount uint64
	DexFee           uint64
	HelioFee         uint64
	Curve            string
	CostToken        string
	Sender           string
	IsSell           bool
}

// moonitTradeEventDisc is sha256("event:TradeEvent")[:8]
var moonitTradeEventDisc = []byte{189, 219, 127, 211, 78, 230, 97, 238}

// tradeEvents returns the TradeEvent logged by each Moonit instruction, keyed
// by its (outer, inner) index (utils.GetProgramDataLogs attributes each
// "Program data:" line to the instruction that wrote it; truncated or
// misaligned logs lose the events after the break).
func (p *MoonitEventParser) tradeEvents() map[[2]int]*moonitTradeEvent {
	if p.logEvents != nil {
		return p.logEvents
	}
	p.logEvents = map[[2]int]*moonitTradeEvent{}

	for _, l := range p.utils.GetProgramDataLogs() {
		data := l.Data
		if l.ProgramId != constants.DEX_PROGRAMS.MOONIT.ID || len(data) < 8 || !bytes.Equal(data[:8], moonitTradeEventDisc) {
			continue
		}
		reader := utils.GetBinaryReader(data[8:])
		evt := &moonitTradeEvent{}
		evt.Amount, _ = reader.ReadU64()
		evt.CollateralAmount, _ = reader.ReadU64()
		evt.DexFee, _ = reader.ReadU64()
		evt.HelioFee, _ = reader.ReadU64()
		reader.Skip(8) // allocation
		evt.Curve, _ = reader.ReadPubkey()
		evt.CostToken, _ = reader.ReadPubkey()
		evt.Sender, _ = reader.ReadPubkey()
		tradeType, _ := reader.ReadU8()
		evt.IsSell = tradeType == 1
		ok := !reader.HasError()
		reader.Release()
		if ok {
			p.logEvents[[2]int{l.OuterIndex, l.InnerIndex}] = evt
		}
	}
	return p.logEvents
}

// decodeTradeEvent decodes buy and sell. Accounts (IDL): 0 sender, 2 curve,
// 4 dex fee, 5 helio fee, 6 mint; later layouts append remaining accounts.
// Amounts come from the TradeEvent the instruction logs: a buyer pays the
// collateral plus both fees, a seller receives the collateral minus both
// fees, in the event's cost token. Without the log the token side comes from
// the instruction's transfers (else the token_amount arg) and the collateral
// side from the user's transfers (else the collateral_amount arg, which is
// the slippage-bounded quote, not the executed amount).
func (p *MoonitEventParser) decodeTradeEvent(data []byte, ci types.ClassifiedInstruction, tradeType types.TradeType) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	argTokens := reader.ReadU64AsBigInt()
	argCollateral := reader.ReadU64AsBigInt()
	if reader.HasError() {
		return nil
	}

	user := accounts[0]
	pool := accounts[2]
	baseMint := accounts[6]
	collateralMint := constants.TOKENS.SOL

	event := &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.MOONIT.Name,
		Type:         tradeType,
		BaseMint:     baseMint,
		BondingCurve: pool,
		Pool:         pool,
		User:         user,
	}

	var base, collateral *types.TokenInfo
	evt := p.tradeEvents()[[2]int{ci.OuterIndex, ci.InnerIndex}]
	if evt != nil && evt.Curve == pool && evt.IsSell == (tradeType == types.TradeTypeSell) {
		if evt.CostToken != "" && evt.CostToken != systemProgramID {
			collateralMint = evt.CostToken
		}
		fees := new(big.Int).Add(new(big.Int).SetUint64(evt.DexFee), new(big.Int).SetUint64(evt.HelioFee))
		amount := new(big.Int).SetUint64(evt.CollateralAmount)
		if tradeType == types.TradeTypeBuy {
			amount.Add(amount, fees)
		} else if amount.Cmp(fees) >= 0 {
			amount.Sub(amount, fees)
		}
		base = tokenInfoFromRaw(p.adapter, baseMint, new(big.Int).SetUint64(evt.Amount))
		collateral = tokenInfoFromRaw(p.adapter, collateralMint, amount)
		event.Fees = p.fees(collateralMint, evt, accounts)
		if len(event.Fees) > 0 {
			protocolFee := types.ConvertToUIAmount(fees, collateral.Decimals)
			event.ProtocolFee = &protocolFee
		}
	} else {
		transfers := instructionTransfers(p.transferActions, ci)
		if tradeType == types.TradeTypeBuy {
			collateral, base = userLegs(p.adapter, transfers, user, collateralMint, baseMint)
		} else {
			base, collateral = userLegs(p.adapter, transfers, user, baseMint, collateralMint)
		}
		if base == nil {
			base = tokenInfoFromRaw(p.adapter, baseMint, argTokens)
		}
		if collateral == nil {
			collateral = tokenInfoFromRaw(p.adapter, collateralMint, argCollateral)
		}
	}

	event.QuoteMint = collateralMint
	event.InputToken, event.OutputToken = collateral, base
	if tradeType == types.TradeTypeSell {
		event.InputToken, event.OutputToken = base, collateral
	}
	return event
}

// systemProgramID is Pubkey::default in base58
const systemProgramID = "11111111111111111111111111111111"

// fees lists the dex and helio fees of a Moonit trade in the collateral mint
func (p *MoonitEventParser) fees(mint string, evt *moonitTradeEvent, accounts []string) []types.FeeInfo {
	decimals := p.adapter.GetTokenDecimals(mint)
	var fees []types.FeeInfo
	for _, f := range []struct {
		amount    uint64
		feeType   string
		recipient string
	}{
		{evt.DexFee, "dex", accounts[4]},
		{evt.HelioFee, "helio", accounts[5]},
	} {
		if f.amount == 0 {
			continue
		}
		v := new(big.Int).SetUint64(f.amount)
		fees = append(fees, types.FeeInfo{
			Mint:      mint,
			Amount:    types.ConvertToUIAmount(v, decimals),
			AmountRaw: v.String(),
			Decimals:  decimals,
			Dex:       constants.DEX_PROGRAMS.MOONIT.Name,
			Type:      f.feeType,
			Recipient: f.recipient,
		})
	}
	return fees
}

func (p *MoonitEventParser) decodeCreateEvent(data []byte, ci types.ClassifiedInstruction) *types.MemeEvent {
	if len(data) < 10 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 4 {
		return nil
	}

	name, _ := reader.ReadString()
	symbol, _ := reader.ReadString()
	uri, _ := reader.ReadString()
	decimals, _ := reader.ReadU8()
	reader.ReadU8() // skip
	totalSupply, _ := reader.ReadU64()

	pool := accounts[2]
	baseMint := accounts[3]
	user := accounts[0]

	totalSupplyFloat := types.ConvertToUIAmountUint64(totalSupply, decimals)

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.MOONIT.Name,
		Type:         types.TradeTypeCreate,
		Timestamp:    p.adapter.BlockTime(),
		Pool:         pool,
		BondingCurve: pool,
		User:         user,
		Creator:      user,
		BaseMint:     baseMint,
		QuoteMint:    constants.TOKENS.SOL,
		Name:         name,
		Symbol:       symbol,
		URI:          uri,
		Decimals:     &decimals,
		TotalSupply:  &totalSupplyFloat,
	}
}

func (p *MoonitEventParser) decodeMigrateEvent(data []byte, ci types.ClassifiedInstruction) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 6 {
		return nil
	}

	bondingCurve := accounts[2]
	baseMint := accounts[5]

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.MOONIT.Name,
		Type:         types.TradeTypeMigrate,
		Timestamp:    p.adapter.BlockTime(),
		BondingCurve: bondingCurve,
		BaseMint:     baseMint,
		QuoteMint:    constants.TOKENS.SOL,
	}
}
