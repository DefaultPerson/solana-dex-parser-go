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

// RaydiumV4ShredParser parses Raydium V4 instructions from shred-stream
type RaydiumV4ShredParser struct {
	adapter    *adapter.TransactionAdapter
	classifier *classifier.InstructionClassifier
}

// NewRaydiumV4ShredParser creates a new RaydiumV4ShredParser
func NewRaydiumV4ShredParser(adapter *adapter.TransactionAdapter, classifier *classifier.InstructionClassifier) *RaydiumV4ShredParser {
	return &RaydiumV4ShredParser{
		adapter:    adapter,
		classifier: classifier,
	}
}

// ProcessInstructions processes Raydium V4 instructions and returns parsed results
func (p *RaydiumV4ShredParser) ProcessInstructions() []interface{} {
	events, _ := p.ProcessAll()
	return events
}

// ProcessTypedInstructions returns typed ParsedShredInstruction results
func (p *RaydiumV4ShredParser) ProcessTypedInstructions() []types.ParsedShredInstruction {
	_, typed := p.ProcessAll()
	return typed
}

// ProcessAll decodes the Raydium V4 instructions into legacy events and typed
// instructions in a single pass
func (p *RaydiumV4ShredParser) ProcessAll() ([]interface{}, []types.ParsedShredInstruction) {
	var events []interface{}
	var typed []types.ParsedShredInstruction
	d := constants.DISCRIMINATORS.RAYDIUM

	for _, ci := range p.classifier.GetInstructions(constants.DEX_PROGRAMS.RAYDIUM_V4.ID) {
		data := p.adapter.GetInstructionData(ci.Instruction)
		if len(data) < 1 {
			continue
		}
		accounts := p.adapter.GetInstructionAccounts(ci.Instruction)
		disc, payload := data[:1], data[1:]

		var eventType string
		var eventData interface{}
		var ins *types.ParsedShredInstruction

		switch {
		case bytes.Equal(disc, d.SWAP), bytes.Equal(disc, d.SWAP_EXACT_OUT), bytes.Equal(disc, d.SWAP_V2), bytes.Equal(disc, d.SWAP_EXACT_OUT_V2):
			exactOut := bytes.Equal(disc, d.SWAP_EXACT_OUT) || bytes.Equal(disc, d.SWAP_EXACT_OUT_V2)
			v2 := bytes.Equal(disc, d.SWAP_V2) || bytes.Equal(disc, d.SWAP_EXACT_OUT_V2)
			if swap := p.decodeSwap(accounts, payload, v2, exactOut); swap != nil {
				eventType, eventData = swapEventType(v2, exactOut), swap
				ins = p.swapInstruction(eventType, swap)
			}
		case bytes.Equal(disc, d.CREATE):
			if create := p.decodeCreate(accounts, payload); create != nil {
				eventType, eventData = "create", create
				ins = p.liquidityInstruction(eventType, types.PoolEventTypeCreate, create, types.ShredAmountExact, types.ShredAmountUnknown)
			}
		case bytes.Equal(disc, d.ADD_LIQUIDITY):
			if add := p.decodeAddLiquidity(accounts, payload); add != nil {
				eventType, eventData = "add_liquidity", add
				ins = p.liquidityInstruction(eventType, types.PoolEventTypeAdd, add, types.ShredAmountMax, types.ShredAmountUnknown)
			}
		case bytes.Equal(disc, d.REMOVE_LIQUIDITY):
			if remove := p.decodeRemoveLiquidity(accounts, payload); remove != nil {
				eventType, eventData = "remove_liquidity", remove
				outKind := types.ShredAmountMin
				if !remove.HasMinAmounts {
					outKind = types.ShredAmountUnknown
				}
				ins = p.liquidityInstruction(eventType, types.PoolEventTypeRemove, remove, types.ShredAmountExact, outKind)
			}
		default:
			continue
		}

		if eventData == nil {
			continue
		}
		idx := utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
		events = append(events, &RaydiumV4ShredInstruction{
			Type:               eventType,
			Data:               eventData,
			Slot:               p.adapter.Slot(),
			Timestamp:          p.adapter.BlockTime(),
			Signature:          p.adapter.Signature(),
			Idx:                idx,
			Signer:             p.adapter.Signers(),
			UnresolvedAccounts: types.HasUnresolvedAccount(accounts),
		})
		ins.ProgramID = constants.DEX_PROGRAMS.RAYDIUM_V4.ID
		ins.ProgramName = constants.DEX_PROGRAMS.RAYDIUM_V4.Name
		ins.Accounts = accounts
		ins.Idx = idx
		typed = append(typed, *ins)
	}

	return events, typed
}

func swapEventType(v2, exactOut bool) string {
	switch {
	case v2 && exactOut:
		return "swap_base_out_v2"
	case v2:
		return "swap_v2"
	case exactOut:
		return "swap_base_out"
	}
	return "swap"
}

// RaydiumV4ShredInstruction represents a parsed Raydium V4 instruction
type RaydiumV4ShredInstruction struct {
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

// RaydiumV4SwapData contains Raydium V4 swap instruction data. The amounts
// are instruction arguments: for swap_base_in (tags 9, 16) InputAmount is the
// exact input and OutputAmount the minimum output; for swap_base_out (tags
// 11, 17, ExactOut) InputAmount is the maximum input and OutputAmount the
// exact output. InputMint and OutputMint are empty when the transaction does
// not reveal the mints of the user's token accounts.
type RaydiumV4SwapData struct {
	Pool               string `json:"pool"`
	User               string `json:"user"`
	InputTokenAccount  string `json:"inputTokenAccount"`
	OutputTokenAccount string `json:"outputTokenAccount"`
	InputAmount        uint64 `json:"inputAmount"`
	OutputAmount       uint64 `json:"outputAmount"`
	InputMint          string `json:"inputMint,omitempty"`
	OutputMint         string `json:"outputMint,omitempty"`
	CoinVault          string `json:"coinVault,omitempty"`
	PcVault            string `json:"pcVault,omitempty"`
	ExactOut           bool   `json:"exactOut,omitempty"`
}

// RaydiumV4LiquidityData contains Raydium V4 liquidity instruction data.
// Base is the pool's coin side, quote its pc side. Amounts are arguments:
// initialize2 amounts are the exact initial reserves, deposit amounts are
// maximums and withdraw amounts minimums (absent in old withdraw
// instructions: HasMinAmounts false). BaseMint and QuoteMint are empty when
// the transaction does not reveal the mints of the pool vaults.
type RaydiumV4LiquidityData struct {
	Pool        string `json:"pool"`
	User        string `json:"user"`
	BaseMint    string `json:"baseMint"`
	QuoteMint   string `json:"quoteMint"`
	LpMint      string `json:"lpMint"`
	BaseAmount  uint64 `json:"baseAmount"`
	QuoteAmount uint64 `json:"quoteAmount"`
	LpAmount    uint64 `json:"lpAmount,omitempty"`
	// BaseVault and QuoteVault are the pool's coin and pc token accounts
	BaseVault  string `json:"baseVault,omitempty"`
	QuoteVault string `json:"quoteVault,omitempty"`
	// HasMinAmounts is true when a withdraw carries min_coin/min_pc amounts
	HasMinAmounts bool `json:"hasMinAmounts,omitempty"`
}

// decodeSwap decodes swap_base_in / swap_base_out. Accounts: 0 token_program,
// 1 amm, then either the v1 layout with (18 accounts) or without (17, current
// SDK) amm_target_orders at 4, whose last three accounts are the user source,
// destination and owner, or the v2 layout (8 accounts): 3 coin vault, 4 pc
// vault, 5 source, 6 destination, 7 owner.
func (p *RaydiumV4ShredParser) decodeSwap(accounts []string, data []byte, v2, exactOut bool) *RaydiumV4SwapData {
	var coinVault, pcVault, source, destination, owner int
	switch {
	case v2 && len(accounts) >= 8:
		coinVault, pcVault, source, destination, owner = 3, 4, 5, 6, 7
	case !v2 && len(accounts) >= 18:
		coinVault, pcVault, source, destination, owner = 5, 6, 15, 16, 17
	case !v2 && len(accounts) == 17:
		coinVault, pcVault, source, destination, owner = 4, 5, 14, 15, 16
	default:
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	first, _ := reader.ReadU64()
	second, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	swap := &RaydiumV4SwapData{
		Pool:               accounts[1],
		User:               accounts[owner],
		InputTokenAccount:  accounts[source],
		OutputTokenAccount: accounts[destination],
		InputAmount:        first,
		OutputAmount:       second,
		CoinVault:          accounts[coinVault],
		PcVault:            accounts[pcVault],
		ExactOut:           exactOut,
	}

	// Mints of the user's accounts; when one side is unknown but both vault
	// mints are, it is the vault mint the other side does not use
	swap.InputMint = p.adapter.KnownTokenAccountMint(swap.InputTokenAccount)
	swap.OutputMint = p.adapter.KnownTokenAccountMint(swap.OutputTokenAccount)
	coinMint, pcMint := p.adapter.KnownTokenAccountMint(swap.CoinVault), p.adapter.KnownTokenAccountMint(swap.PcVault)
	if coinMint != "" && pcMint != "" {
		other := func(m string) string {
			switch m {
			case coinMint:
				return pcMint
			case pcMint:
				return coinMint
			}
			return ""
		}
		if swap.InputMint == "" {
			swap.InputMint = other(swap.OutputMint)
		}
		if swap.OutputMint == "" {
			swap.OutputMint = other(swap.InputMint)
		}
	}
	return swap
}

// shredTradeType is utils.GetShredTradeType for one pool, whose two mints
// differ, so one known side can decide what utils.GetTradeType would say
// whatever the other mint is: WSOL in is a BUY, WSOL out a SELL, and an input
// that is neither SOL nor a stablecoin a SELL. Any other single known side
// does not (USDC in is a BUY against a token but a SELL against SOL) and
// gives SWAP.
func shredTradeType(inMint, outMint string) types.TradeType {
	if inMint != "" && outMint != "" {
		return utils.GetShredTradeType(inMint, outMint)
	}
	switch {
	case inMint == constants.TOKENS.SOL:
		return types.TradeTypeBuy
	case outMint == constants.TOKENS.SOL:
		return types.TradeTypeSell
	case inMint != "" && !constants.IsSOL(inMint) && !constants.IsStablecoin(inMint):
		return types.TradeTypeSell
	}
	return types.TradeTypeSwap
}

func (p *RaydiumV4ShredParser) swapInstruction(action string, swap *RaydiumV4SwapData) *types.ParsedShredInstruction {
	inDecimals, outDecimals := p.adapter.GetTokenDecimals(swap.InputMint), p.adapter.GetTokenDecimals(swap.OutputMint)
	inKind, outKind := types.ShredAmountExact, types.ShredAmountMin
	if swap.ExactOut {
		inKind, outKind = types.ShredAmountMax, types.ShredAmountExact
	}

	return &types.ParsedShredInstruction{
		Action: action,
		Trade: &types.TradeInfo{
			Type: shredTradeType(swap.InputMint, swap.OutputMint),
			Pool: []string{swap.Pool},
			User: swap.User,
			InputToken: types.TokenInfo{
				Mint:      swap.InputMint,
				Amount:    types.ConvertToUIAmountUint64(swap.InputAmount, inDecimals),
				AmountRaw: strconv.FormatUint(swap.InputAmount, 10),
				Decimals:  inDecimals,
				Source:    swap.InputTokenAccount,
			},
			OutputToken: types.TokenInfo{
				Mint:        swap.OutputMint,
				Amount:      types.ConvertToUIAmountUint64(swap.OutputAmount, outDecimals),
				AmountRaw:   strconv.FormatUint(swap.OutputAmount, 10),
				Decimals:    outDecimals,
				Destination: swap.OutputTokenAccount,
			},
			ProgramId: constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
			AMM:       constants.DEX_PROGRAMS.RAYDIUM_V4.Name,
			AMMs:      []string{constants.DEX_PROGRAMS.RAYDIUM_V4.Name},
			Extras: map[string]interface{}{
				"amm":       swap.Pool,
				"coinVault": swap.CoinVault,
				"pcVault":   swap.PcVault,
			},
		},
		InputAmountKind:  inKind,
		OutputAmountKind: outKind,
	}
}

// decodeCreate decodes initialize2 (nonce u8, open_time u64, init_pc_amount
// u64, init_coin_amount u64): accounts 4 amm, 7 lp_mint, 8 coin_mint,
// 9 pc_mint, 10 coin vault, 11 pc vault, 17 user_wallet
func (p *RaydiumV4ShredParser) decodeCreate(accounts []string, data []byte) *RaydiumV4LiquidityData {
	if len(accounts) < 18 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	reader.ReadU8()  // nonce
	reader.ReadU64() // open_time
	pcAmount, _ := reader.ReadU64()
	coinAmount, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &RaydiumV4LiquidityData{
		Pool:        accounts[4],
		User:        accounts[17],
		BaseMint:    accounts[8],
		QuoteMint:   accounts[9],
		LpMint:      accounts[7],
		BaseVault:   accounts[10],
		QuoteVault:  accounts[11],
		BaseAmount:  coinAmount,
		QuoteAmount: pcAmount,
	}
}

// decodeAddLiquidity decodes deposit (max_coin_amount, max_pc_amount,
// base_side, [other_amount_min]): accounts 1 amm, 5 lp_mint, 6 coin vault,
// 7 pc vault, 12 user_owner
func (p *RaydiumV4ShredParser) decodeAddLiquidity(accounts []string, data []byte) *RaydiumV4LiquidityData {
	if len(accounts) < 13 {
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	maxCoinAmount, _ := reader.ReadU64()
	maxPcAmount, _ := reader.ReadU64()
	if reader.HasError() {
		return nil
	}

	return &RaydiumV4LiquidityData{
		Pool:        accounts[1],
		User:        accounts[12],
		BaseMint:    p.adapter.KnownTokenAccountMint(accounts[6]),
		QuoteMint:   p.adapter.KnownTokenAccountMint(accounts[7]),
		LpMint:      accounts[5],
		BaseVault:   accounts[6],
		QuoteVault:  accounts[7],
		BaseAmount:  maxCoinAmount,
		QuoteAmount: maxPcAmount,
	}
}

// decodeRemoveLiquidity decodes withdraw (amount, [min_coin_amount,
// min_pc_amount]): accounts 1 amm, 5 lp_mint, 6 coin vault, 7 pc vault and
// the owner at 18 (22-account layout with the withdraw queue and temp LP
// accounts) or 16 (current 20/21-account layout)
func (p *RaydiumV4ShredParser) decodeRemoveLiquidity(accounts []string, data []byte) *RaydiumV4LiquidityData {
	owner := 16
	switch {
	case len(accounts) >= 22:
		owner = 18
	case len(accounts) < 17:
		return nil
	}

	reader := utils.GetBinaryReader(data)
	defer reader.Release()
	lpAmount, err := reader.ReadU64()
	if err != nil {
		return nil
	}

	remove := &RaydiumV4LiquidityData{
		Pool:       accounts[1],
		User:       accounts[owner],
		BaseMint:   p.adapter.KnownTokenAccountMint(accounts[6]),
		QuoteMint:  p.adapter.KnownTokenAccountMint(accounts[7]),
		LpMint:     accounts[5],
		BaseVault:  accounts[6],
		QuoteVault: accounts[7],
		LpAmount:   lpAmount,
	}
	if reader.Remaining() >= 16 {
		remove.BaseAmount, _ = reader.ReadU64()
		remove.QuoteAmount, _ = reader.ReadU64()
		remove.HasMinAmounts = true
	}
	return remove
}

// liquidityInstruction builds a PoolEvent; for liquidity the input side is
// what the user deposits (token amounts for CREATE and ADD, LP for REMOVE)
func (p *RaydiumV4ShredParser) liquidityInstruction(action string, eventType types.PoolEventType, l *RaydiumV4LiquidityData, inKind, outKind types.ShredAmountKind) *types.ParsedShredInstruction {
	baseDecimals, quoteDecimals := p.adapter.GetTokenDecimals(l.BaseMint), p.adapter.GetTokenDecimals(l.QuoteMint)
	baseAmount := types.ConvertToUIAmountUint64(l.BaseAmount, baseDecimals)
	quoteAmount := types.ConvertToUIAmountUint64(l.QuoteAmount, quoteDecimals)

	event := &types.PoolEvent{
		PoolEventBase: types.PoolEventBase{
			Type:      eventType,
			ProgramId: constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
			AMM:       constants.DEX_PROGRAMS.RAYDIUM_V4.Name,
			User:      l.User,
		},
		PoolId:         l.Pool,
		PoolLpMint:     l.LpMint,
		Token0Mint:     l.BaseMint,
		Token0Decimals: &baseDecimals,
		Token1Mint:     l.QuoteMint,
		Token1Decimals: &quoteDecimals,
	}
	if eventType != types.PoolEventTypeRemove || l.HasMinAmounts {
		event.Token0Amount = &baseAmount
		event.Token0AmountRaw = strconv.FormatUint(l.BaseAmount, 10)
		event.Token1Amount = &quoteAmount
		event.Token1AmountRaw = strconv.FormatUint(l.QuoteAmount, 10)
	}
	if eventType == types.PoolEventTypeRemove {
		lpAmount := types.ConvertToUIAmountUint64(l.LpAmount, p.adapter.GetTokenDecimals(l.LpMint))
		event.LpAmount = &lpAmount
		event.LpAmountRaw = strconv.FormatUint(l.LpAmount, 10)
	}

	return &types.ParsedShredInstruction{
		Action:           action,
		Liquidity:        event,
		InputAmountKind:  inKind,
		OutputAmountKind: outKind,
	}
}
