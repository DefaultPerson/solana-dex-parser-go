package orca

import (
	"bytes"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// OrcaLiquidityParser parses Orca liquidity operations
type OrcaLiquidityParser struct {
	*parsers.BaseLiquidityParser
}

// NewOrcaLiquidityParser creates a new Orca liquidity parser
func NewOrcaLiquidityParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *OrcaLiquidityParser {
	return &OrcaLiquidityParser{
		BaseLiquidityParser: parsers.NewBaseLiquidityParser(adapter, transferActions, classifiedInstructions),
	}
}

// ProcessLiquidity parses liquidity events
func (p *OrcaLiquidityParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.ORCA.ID {
			event := p.parseInstruction(ci.Instruction, ci.ProgramId, ci.OuterIndex, ci.InnerIndex)
			if event != nil {
				events = append(events, *event)
			}
		}
	}

	return events
}

// orcaLiquidityAction describes a Whirlpool liquidity instruction
type orcaLiquidityAction struct {
	eventType types.PoolEventType
	// mintIndexA/B are the token_mint_a/b accounts of the *_v2 instructions,
	// -1 when the instruction has none (v1)
	mintIndexA, mintIndexB int
	// hasLiquidityArg reports whether the args start with liquidity_amount u128
	hasLiquidityArg bool
}

// getLiquidityAction determines the pool action from instruction data.
// Accounts (whirlpool 0.9.0 IDL): increase/decrease_liquidity have no mint
// accounts; the *_v2 variants have token_mint_a 7 and token_mint_b 8.
// increase_liquidity_by_token_amounts_v2 takes token amounts instead of a
// liquidity amount. collect_fees(_v2) and collect_reward(_v2) are not
// liquidity events (upstream parity).
func (p *OrcaLiquidityParser) getLiquidityAction(data []byte) *orcaLiquidityAction {
	if len(data) < 8 {
		return nil
	}
	disc := data[:8]
	orca := constants.DISCRIMINATORS.ORCA

	switch {
	case bytes.Equal(disc, orca.ADD_LIQUIDITY):
		return &orcaLiquidityAction{types.PoolEventTypeAdd, -1, -1, true}
	case bytes.Equal(disc, orca.ADD_LIQUIDITY2):
		return &orcaLiquidityAction{types.PoolEventTypeAdd, 7, 8, true}
	case bytes.Equal(disc, orca.ADD_LIQUIDITY_BY_TOKEN_AMOUNTS_V2):
		return &orcaLiquidityAction{types.PoolEventTypeAdd, 7, 8, false}
	case bytes.Equal(disc, orca.REMOVE_LIQUIDITY):
		return &orcaLiquidityAction{types.PoolEventTypeRemove, -1, -1, true}
	case bytes.Equal(disc, orca.REMOVE_LIQUIDITY_V2):
		return &orcaLiquidityAction{types.PoolEventTypeRemove, 7, 8, true}
	}
	return nil
}

// parseInstruction parses a single instruction. innerIndex is -1 for an
// outer instruction.
func (p *OrcaLiquidityParser) parseInstruction(instruction interface{}, programId string, outerIndex int, innerIndex int) *types.PoolEvent {
	data := p.Adapter.GetInstructionData(instruction)
	action := p.getLiquidityAction(data)
	if action == nil {
		return nil
	}

	transfers := p.GetTransfersForInstruction(programId, outerIndex, innerIndex, nil)
	event := p.parseLiquidityEvent(instruction, action, data, transfers)
	if event != nil {
		event.Idx = utils.FormatIdx(outerIndex, innerIndex)
	}
	return event
}

// parseLiquidityEvent builds the event of a liquidity instruction from its
// transfers. The pool (and, as upstream, PoolLpMint) is the whirlpool,
// account 0.
func (p *OrcaLiquidityParser) parseLiquidityEvent(instruction interface{}, action *orcaLiquidityAction, data []byte, transfers []types.TransferData) *types.PoolEvent {
	lpTransfers := p.Utils.GetLPTransfers(transfers)
	var token0, token1 *types.TransferData
	if len(lpTransfers) > 0 {
		token0 = &lpTransfers[0]
	}
	if len(lpTransfers) > 1 {
		token1 = &lpTransfers[1]
	}

	programId := p.Adapter.GetInstructionProgramId(instruction)
	accounts := p.Adapter.GetInstructionAccounts(instruction)

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
	}
	// Without transfers (nothing moved on one side) the *_v2 accounts name the mints
	if token0Mint == "" && token1Mint == "" && action.mintIndexB >= 0 && action.mintIndexB < len(accounts) {
		token0Mint, token1Mint = accounts[action.mintIndexA], accounts[action.mintIndexB]
		if utils.GetTradeType(token0Mint, token1Mint) == types.TradeTypeBuy {
			token0Mint, token1Mint = token1Mint, token0Mint
		}
	}

	token0Decimals := p.Adapter.GetTokenDecimals(token0Mint)
	token1Decimals := p.Adapter.GetTokenDecimals(token1Mint)

	event := &types.PoolEvent{
		PoolEventBase:  p.Adapter.GetPoolEventBase(action.eventType, programId),
		Token0Mint:     token0Mint,
		Token1Mint:     token1Mint,
		Token0Decimals: &token0Decimals,
		Token1Decimals: &token1Decimals,
	}

	if len(accounts) > 0 {
		event.PoolId = accounts[0]
		event.PoolLpMint = accounts[0]
	}

	if token0 != nil && token0.Info.TokenAmount.UIAmount != nil {
		event.Token0Amount = token0.Info.TokenAmount.UIAmount
		event.Token0AmountRaw = token0.Info.TokenAmount.Amount
	}
	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	}

	// LP amount: the liquidity_amount u128 argument (liquidity has no decimals)
	if action.hasLiquidityArg && len(data) >= 24 {
		liquidity := new(big.Int).SetBytes(reverseBytes(data[8:24]))
		uiAmt := types.ConvertToUIAmount(liquidity, 0)
		event.LpAmount = &uiAmt
		event.LpAmountRaw = liquidity.String()
	}

	return event
}

// reverseBytes returns a reversed copy of b (little endian to big endian)
func reverseBytes(b []byte) []byte {
	r := make([]byte, len(b))
	for i := range b {
		r[len(b)-1-i] = b[i]
	}
	return r
}
