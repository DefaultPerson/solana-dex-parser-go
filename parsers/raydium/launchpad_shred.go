package raydium

import (
	"bytes"
	"strconv"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// LaunchpadShredParser parses Raydium Launchpad (LCP) instructions from shred-stream
type LaunchpadShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewLaunchpadShredParser creates a new LaunchpadShredParser
func NewLaunchpadShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *LaunchpadShredParser {
	return &LaunchpadShredParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes Raydium LCP instructions and returns parsed results
func (p *LaunchpadShredParser) ProcessInstructions() []interface{} {
	events, _ := p.ProcessAll()
	return events
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *LaunchpadShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	_, typed := p.ProcessAll()
	return typed
}

// ProcessAll decodes the LaunchLab instructions into legacy events and typed
// meme events in a single pass
func (p *LaunchpadShredParser) ProcessAll() ([]interface{}, []types.ParsedShredInstruction) {
	var events []interface{}
	var typed []types.ParsedShredInstruction
	d := constants.DISCRIMINATORS.RAYDIUM_LCP

	for _, ci := range p.classifier.GetInstructions(constants.DEX_PROGRAMS.RAYDIUM_LCP.ID) {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 8 {
			continue
		}
		accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
		disc, payload := data[:8], data[8:]

		var eventType string
		var eventData interface{}
		var memeEvent *types.MemeEvent
		var inKind, outKind types.ShredAmountKind

		switch {
		case bytes.Equal(disc, d.INITIALIZE), bytes.Equal(disc, d.INITIALIZE_V2), bytes.Equal(disc, d.INITIALIZE_WITH_TOKEN_2022):
			if create := p.decodeCreateInstruction(accounts, payload); create != nil {
				eventType, eventData, memeEvent = "create", create, p.createMemeEvent(create)
			}
		case bytes.Equal(disc, d.BUY_EXACT_IN):
			if trade := p.decodeTrade(accounts, payload, true, false); trade != nil {
				eventType, eventData, memeEvent = "buy_exact_in", trade, p.buildMemeEvent(trade)
				inKind, outKind = types.ShredAmountExact, types.ShredAmountMin
			}
		case bytes.Equal(disc, d.BUY_EXACT_OUT):
			if trade := p.decodeTrade(accounts, payload, true, true); trade != nil {
				eventType, eventData, memeEvent = "buy_exact_out", trade, p.buildMemeEvent(trade)
				inKind, outKind = types.ShredAmountMax, types.ShredAmountExact
			}
		case bytes.Equal(disc, d.SELL_EXACT_IN):
			if trade := p.decodeTrade(accounts, payload, false, false); trade != nil {
				eventType, eventData, memeEvent = "sell_exact_in", trade, p.buildMemeEvent(trade)
				inKind, outKind = types.ShredAmountExact, types.ShredAmountMin
			}
		case bytes.Equal(disc, d.SELL_EXACT_OUT):
			if trade := p.decodeTrade(accounts, payload, false, true); trade != nil {
				eventType, eventData, memeEvent = "sell_exact_out", trade, p.buildMemeEvent(trade)
				inKind, outKind = types.ShredAmountMax, types.ShredAmountExact
			}
		case bytes.Equal(disc, d.MIGRATE_TO_AMM):
			if migrate := p.decodeMigrateToAMM(accounts, payload); migrate != nil {
				eventType, eventData, memeEvent = "migrate_to_amm", migrate, p.migrateMemeEvent(migrate)
			}
		case bytes.Equal(disc, d.MIGRATE_TO_CPSWAP):
			if migrate := p.decodeMigrateToCPSwap(accounts); migrate != nil {
				eventType, eventData, memeEvent = "migrate_to_cpswap", migrate, p.migrateMemeEvent(migrate)
			}
		default:
			continue
		}

		if eventData == nil {
			continue
		}
		idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
		events = append(events, &LaunchpadShredInstruction{
			Type:               eventType,
			Data:               eventData,
			Slot:               p.adapter.Slot(),
			Timestamp:          p.adapter.BlockTime(),
			Signature:          p.adapter.Signature(),
			Idx:                idx,
			Signer:             p.adapter.Signers(),
			UnresolvedAccounts: types.HasUnresolvedAccount(accounts),
		})

		memeEvent.Signature = p.adapter.Signature()
		memeEvent.Slot = p.adapter.Slot()
		memeEvent.Timestamp = p.adapter.BlockTime()
		memeEvent.Idx = idx
		typed = append(typed, types.ParsedShredInstruction{
			ProgramID:        constants.DEX_PROGRAMS.RAYDIUM_LCP.ID,
			ProgramName:      constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
			Action:           eventType,
			MemeEvent:        memeEvent,
			Accounts:         accounts,
			Idx:              idx,
			InputAmountKind:  inKind,
			OutputAmountKind: outKind,
		})
	}

	return events, typed
}

// LaunchpadShredInstruction represents a parsed Raydium LCP instruction
type LaunchpadShredInstruction struct {
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

// LaunchpadCreateData contains Raydium LCP create instruction data
// (initialize, initialize_v2, initialize_with_token_2022)
type LaunchpadCreateData struct {
	User      string `json:"user"`
	Pool      string `json:"pool"`
	BaseMint  string `json:"baseMint"`
	QuoteMint string `json:"quoteMint"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	URI       string `json:"uri"`
	// Decimals is the base mint decimals argument
	Decimals uint8 `json:"decimals"`
	// Payer pays for the pool creation; User is the creator
	Payer string `json:"payer,omitempty"`
	// PlatformConfig is the launch platform's config account
	PlatformConfig string `json:"platformConfig,omitempty"`
}

// LaunchpadTradeData contains Raydium LCP trade instruction data. The amounts
// are instruction arguments: for the exact-in instructions InputAmount is
// exact and OutputAmount the minimum output; for the exact-out instructions
// (ExactOut) OutputAmount is exact and InputAmount the maximum input. Buys
// spend the quote mint (WSOL, USD1, ...) for the base mint, sells the reverse.
type LaunchpadTradeData struct {
	User           string `json:"user"`
	Pool           string `json:"pool"`
	BaseMint       string `json:"baseMint"`
	QuoteMint      string `json:"quoteMint"`
	InputMint      string `json:"inputMint"`
	OutputMint     string `json:"outputMint"`
	InputAmount    uint64 `json:"inputAmount"`
	OutputAmount   uint64 `json:"outputAmount"`
	TradeType      string `json:"tradeType"`
	PlatformConfig string `json:"platformConfig"`
	// ShareFeeRate is the share_fee_rate argument
	ShareFeeRate uint64 `json:"shareFeeRate,omitempty"`
	// ExactOut is true for buy_exact_out and sell_exact_out
	ExactOut bool `json:"exactOut,omitempty"`
}

// LaunchpadMigrateData contains Raydium LCP migrate instruction data: Pool
// is the new AMM / CPMM pool, BondingCurve the LaunchLab pool_state
type LaunchpadMigrateData struct {
	BaseMint     string `json:"baseMint"`
	QuoteMint    string `json:"quoteMint"`
	Pool         string `json:"pool"`
	PoolDex      string `json:"poolDex"`
	BondingCurve string `json:"bondingCurve,omitempty"`
}

// decodeCreateInstruction decodes MintParams (decimals u8, name, symbol,
// uri), the first argument of initialize, initialize_v2 and
// initialize_with_token_2022. Accounts: 0 payer, 1 creator,
// 3 platform_config, 5 pool_state, 6 base_mint, 7 quote_mint.
func (p *LaunchpadShredParser) decodeCreateInstruction(accounts []string, data []byte) *LaunchpadCreateData {
	if len(accounts) < 8 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()

	decimals, _ := reader.ReadU8()
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

	return &LaunchpadCreateData{
		User:           accounts[1],
		Payer:          accounts[0],
		PlatformConfig: accounts[3],
		Pool:           accounts[5],
		BaseMint:       accounts[6],
		QuoteMint:      accounts[7],
		Name:           name,
		Symbol:         symbol,
		URI:            uri,
		Decimals:       decimals,
	}
}

func (p *LaunchpadShredParser) createMemeEvent(createData *LaunchpadCreateData) *types.MemeEvent {
	decimals := createData.Decimals
	return &types.MemeEvent{
		Protocol:       constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:           types.TradeTypeCreate,
		User:           createData.User,
		BaseMint:       createData.BaseMint,
		QuoteMint:      createData.QuoteMint,
		Pool:           createData.Pool,
		BondingCurve:   createData.Pool,
		PlatformConfig: createData.PlatformConfig,
		Name:           createData.Name,
		Symbol:         createData.Symbol,
		URI:            createData.URI,
		Decimals:       &decimals,
	}
}

// decodeTrade decodes buy_exact_in (amount_in, minimum_amount_out),
// buy_exact_out (amount_out, maximum_amount_in), sell_exact_in and
// sell_exact_out, each followed by share_fee_rate. Accounts: 0 payer,
// 3 platform_config, 4 pool_state, 9 base_token_mint, 10 quote_token_mint.
func (p *LaunchpadShredParser) decodeTrade(accounts []string, data []byte, buy, exactOut bool) *LaunchpadTradeData {
	if len(accounts) < 11 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	first, _ := reader.ReadU64()
	second, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}
	shareFeeRate, _ := reader.ReadU64()

	trade := &LaunchpadTradeData{
		User:           accounts[0],
		Pool:           accounts[4],
		PlatformConfig: accounts[3],
		BaseMint:       accounts[9],
		QuoteMint:      accounts[10],
		ShareFeeRate:   shareFeeRate,
		ExactOut:       exactOut,
	}
	if buy {
		trade.InputMint, trade.OutputMint, trade.TradeType = trade.QuoteMint, trade.BaseMint, "buy"
	} else {
		trade.InputMint, trade.OutputMint, trade.TradeType = trade.BaseMint, trade.QuoteMint, "sell"
	}
	if exactOut {
		trade.OutputAmount, trade.InputAmount = first, second
	} else {
		trade.InputAmount, trade.OutputAmount = first, second
	}
	return trade
}

// decodeMigrateToAMM decodes migrate_to_amm. The current program takes no
// arguments and 23 accounts (1 base_mint, 2 quote_mint, 5 amm_pool,
// 14 pool_state); the earlier OpenBook layout takes (base_lot_size,
// quote_lot_size, market_vault_signer_nonce) and 32 accounts (13 amm_pool,
// 23 pool_state).
func (p *LaunchpadShredParser) decodeMigrateToAMM(accounts []string, data []byte) *LaunchpadMigrateData {
	pool, poolState := 5, 14
	if len(data) >= 8+8+1 {
		pool, poolState = 13, 23
	}
	if len(accounts) <= poolState {
		return nil
	}

	return &LaunchpadMigrateData{
		BaseMint:     accounts[1],
		QuoteMint:    accounts[2],
		Pool:         accounts[pool],
		BondingCurve: accounts[poolState],
		PoolDex:      constants.DEX_PROGRAMS.RAYDIUM_V4.Name,
	}
}

// decodeMigrateToCPSwap decodes migrate_to_cpswap: accounts 1 base_mint,
// 2 quote_mint, 5 cpswap_pool, 17 pool_state
func (p *LaunchpadShredParser) decodeMigrateToCPSwap(accounts []string) *LaunchpadMigrateData {
	if len(accounts) < 17 {
		return nil
	}

	migrate := &LaunchpadMigrateData{
		BaseMint:  accounts[1],
		QuoteMint: accounts[2],
		Pool:      accounts[5],
		PoolDex:   constants.DEX_PROGRAMS.RAYDIUM_CPMM.Name,
	}
	if len(accounts) > 17 {
		migrate.BondingCurve = accounts[17]
	}
	return migrate
}

func (p *LaunchpadShredParser) migrateMemeEvent(migrateData *LaunchpadMigrateData) *types.MemeEvent {
	return &types.MemeEvent{
		Protocol:     constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:         types.TradeTypeMigrate,
		BaseMint:     migrateData.BaseMint,
		QuoteMint:    migrateData.QuoteMint,
		Pool:         migrateData.Pool,
		PoolDex:      migrateData.PoolDex,
		BondingCurve: migrateData.BondingCurve,
	}
}

// buildMemeEvent reports a trade; decimals come from the transaction or
// TOKEN_DECIMALS, 0 when unknown (LaunchLab base mints choose their decimals)
func (p *LaunchpadShredParser) buildMemeEvent(data *LaunchpadTradeData) *types.MemeEvent {
	tradeType := types.TradeTypeBuy
	if data.TradeType == "sell" {
		tradeType = types.TradeTypeSell
	}
	inputDecimal, outputDecimal := shredKnownDecimals(p.adapter, data.InputMint), shredKnownDecimals(p.adapter, data.OutputMint)

	return &types.MemeEvent{
		Protocol:       constants.DEX_PROGRAMS.RAYDIUM_LCP.Name,
		Type:           tradeType,
		User:           data.User,
		BaseMint:       data.BaseMint,
		QuoteMint:      data.QuoteMint,
		BondingCurve:   data.Pool,
		Pool:           data.Pool,
		PlatformConfig: data.PlatformConfig,
		InputToken: &types.TokenInfo{
			Mint:      data.InputMint,
			Amount:    types.ConvertToUIAmountUint64(data.InputAmount, inputDecimal),
			AmountRaw: strconv.FormatUint(data.InputAmount, 10),
			Decimals:  inputDecimal,
		},
		OutputToken: &types.TokenInfo{
			Mint:      data.OutputMint,
			Amount:    types.ConvertToUIAmountUint64(data.OutputAmount, outputDecimal),
			AmountRaw: strconv.FormatUint(data.OutputAmount, 10),
			Decimals:  outputDecimal,
		},
	}
}
