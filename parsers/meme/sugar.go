package meme

import (
	"bytes"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// SugarParser parses Sugar transactions
type SugarParser struct {
	*parsers.BaseParser
	eventParser *SugarEventParser
}

// NewSugarParser creates a new Sugar parser
func NewSugarParser(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *SugarParser {
	return &SugarParser{
		BaseParser:  parsers.NewBaseParser(adapter, dexInfo, transferActions, classifiedInstructions),
		eventParser: NewSugarEventParser(adapter, transferActions),
	}
}

// ProcessTrades parses Sugar trades
func (p *SugarParser) ProcessTrades() []types.TradeInfo {
	var trades []types.TradeInfo

	events := p.eventParser.ParseInstructions(p.ClassifiedInstructions)

	for _, event := range events {
		if event.Type == types.TradeTypeBuy || event.Type == types.TradeTypeSell || event.Type == "SWAP" {
			trade := p.createTradeInfo(event)
			if trade != nil {
				trades = append(trades, *trade)
			}
		}
	}

	return trades
}

func (p *SugarParser) createTradeInfo(event *types.MemeEvent) *types.TradeInfo {
	if event.InputToken == nil || event.OutputToken == nil {
		return nil
	}

	var pool []string
	if event.BondingCurve != "" {
		pool = []string{event.BondingCurve}
	}

	amm := p.DexInfo.AMM
	if amm == "" {
		amm = constants.DEX_PROGRAMS.SUGAR.Name
	}

	trade := &types.TradeInfo{
		Type:        event.Type,
		Pool:        pool,
		InputToken:  *event.InputToken,
		OutputToken: *event.OutputToken,
		User:        event.User,
		ProgramId:   constants.DEX_PROGRAMS.SUGAR.ID,
		AMM:         amm,
		Route:       p.DexInfo.Route,
		Slot:        p.Adapter.Slot(),
		Timestamp:   event.Timestamp,
		Signature:   p.Adapter.Signature(),
		Idx:         event.Idx,
	}

	return p.Utils.AttachTokenTransferInfo(trade, p.TransferActions)
}

// SugarEventParser parses Sugar events
type SugarEventParser struct {
	adapter         *adapter.TransactionAdapter
	transferActions map[string][]types.TransferData
	utils           *utils.TransactionUtils
}

// NewSugarEventParser creates a new event parser
func NewSugarEventParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) *SugarEventParser {
	return &SugarEventParser{
		adapter:         adapter,
		transferActions: transferActions,
		utils:           utils.NewTransactionUtils(adapter),
	}
}

// ParseInstructions parses classified instructions into meme events, in
// execution order
func (p *SugarEventParser) ParseInstructions(instructions []types.ClassifiedInstruction) []*types.MemeEvent {
	var events []*types.MemeEvent

	ordered := append([]types.ClassifiedInstruction(nil), instructions...)
	utils.SortInstructionsByExecution(ordered)

	for _, ci := range ordered {
		if ci.ProgramId != constants.DEX_PROGRAMS.SUGAR.ID {
			continue
		}

		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}

		disc := data[:8]
		var event *types.MemeEvent

		switch {
		case bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.BUY_EXACT_IN),
			bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.BUY_EXACT_OUT),
			bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.BUY_MAX_OUT):
			event = p.decodeTradeEvent(data[8:], ci, types.TradeTypeBuy)
		case bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.SELL_EXACT_IN),
			bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.SELL_EXACT_OUT):
			event = p.decodeTradeEvent(data[8:], ci, types.TradeTypeSell)
		case bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.CREATE):
			event = p.decodeCreateEvent(data[8:], ci.Instruction)
		case bytes.Equal(disc, constants.DISCRIMINATORS.SUGAR.MIGRATE_TO_RADIUM):
			event = p.decodeMigrateEvent(ci.Instruction)
		}

		if event != nil {
			event.Signature = p.adapter.Signature()
			event.Slot = p.adapter.Slot()
			event.Timestamp = p.adapter.BlockTime()
			event.Idx = formatIdx(ci.OuterIndex, ci.InnerIndex)
			events = append(events, event)
		}
	}

	return events
}

// decodeTradeEvent decodes the buy and sell instructions. Accounts (checked
// on mainnet sell_exact_in, as in upstream): 1 base mint, 2 pool, 6 user;
// the quote is native SOL. Amounts are the user's transfers in the
// instruction (the SOL fee goes to a separate recipient); without them the
// args (u16, amount in, minimum out) are used.
func (p *SugarEventParser) decodeTradeEvent(data []byte, ci types.ClassifiedInstruction, tradeType types.TradeType) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	reader.Skip(2)
	argIn := reader.ReadU64AsBigInt()
	argOut := reader.ReadU64AsBigInt()
	if reader.HasError() {
		return nil
	}

	baseMint := accounts[1]
	pool := accounts[2]
	user := accounts[6]
	quoteMint := constants.TOKENS.SOL
	inMint, outMint := quoteMint, baseMint
	if tradeType == types.TradeTypeSell {
		inMint, outMint = baseMint, quoteMint
	}

	in, out := userLegs(p.adapter, instructionTransfers(p.transferActions, ci), user, inMint, outMint)
	if in == nil {
		in = tokenInfoFromRaw(p.adapter, inMint, argIn)
	}
	if out == nil {
		out = tokenInfoFromRaw(p.adapter, outMint, argOut)
	}

	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.SUGAR.Name,
		Type:         tradeType,
		BaseMint:     baseMint,
		QuoteMint:    quoteMint,
		BondingCurve: pool,
		Pool:         pool,
		User:         user,
		InputToken:   in,
		OutputToken:  out,
	}
}

// decodeCreateEvent decodes create. Accounts (checked on mainnet 3nFeuZaT,
// as in upstream): 1 metadata, 2 pool, 3 base mint, 6 creator; the args are
// name, symbol, uri.
func (p *SugarEventParser) decodeCreateEvent(data []byte, instruction interface{}) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 7 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	name, err := reader.ReadString()
	if err != nil {
		return nil
	}
	symbol, err := reader.ReadString()
	if err != nil {
		return nil
	}
	uri, err := reader.ReadString()
	if err != nil {
		return nil
	}

	pool := accounts[2]
	baseMint := accounts[3]
	creator := accounts[6]
	event := &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.SUGAR.Name,
		Type:         types.TradeTypeCreate,
		Timestamp:    p.adapter.BlockTime(),
		User:         creator,
		BaseMint:     baseMint,
		QuoteMint:    constants.TOKENS.SOL,
		Name:         name,
		Symbol:       symbol,
		URI:          uri,
		BondingCurve: pool,
		Pool:         pool,
		Creator:      creator,
	}
	if d, ok := p.adapter.SPLDecimalsMap[baseMint]; ok {
		event.Decimals = &d
	}
	return event
}

// decodeMigrateEvent decodes migrate_to_radium. Accounts (checked on mainnet
// 2kWg1Xif and 67JQpQfx, as in upstream): 1 base mint, 2 quote mint of the
// new pool (pSOL), 3 bonding curve, 11 Raydium CPMM program, 12 migrator,
// 15 CPMM pool.
func (p *SugarEventParser) decodeMigrateEvent(instruction interface{}) *types.MemeEvent {
	accounts := p.adapter.GetInstructionAccounts(instruction)
	if len(accounts) < 16 {
		return nil
	}

	event := &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.SUGAR.Name,
		Type:         types.TradeTypeMigrate,
		Timestamp:    p.adapter.BlockTime(),
		User:         accounts[12],
		BaseMint:     accounts[1],
		QuoteMint:    accounts[2],
		BondingCurve: accounts[3],
		Pool:         accounts[15],
	}
	if accounts[11] == constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID {
		event.PoolDex = constants.DEX_PROGRAMS.RAYDIUM_CPMM.Name
	}
	return event
}

// ProcessEvents implements the EventParser interface
func (p *SugarEventParser) ProcessEvents() []types.MemeEvent {
	instructions := getAllInstructionsForMultiPrograms(p.adapter, []string{constants.DEX_PROGRAMS.SUGAR.ID})
	events := p.ParseInstructions(instructions)

	result := make([]types.MemeEvent, 0, len(events))
	for _, e := range events {
		if e != nil {
			result = append(result, *e)
		}
	}
	return result
}
