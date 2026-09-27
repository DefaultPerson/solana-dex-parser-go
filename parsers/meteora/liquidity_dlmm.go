package meteora

import (
	"bytes"
	"encoding/binary"
	"strconv"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
	"github.com/mr-tron/base58"
)

// MeteoraDLMMPoolParser parses Meteora DLMM pool events
type MeteoraDLMMPoolParser struct {
	*MeteoraLiquidityParserBase
}

// NewMeteoraDLMMPoolParser creates a new DLMM pool parser
func NewMeteoraDLMMPoolParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MeteoraDLMMPoolParser {
	return &MeteoraDLMMPoolParser{
		MeteoraLiquidityParserBase: NewMeteoraLiquidityParserBase(adapter, transferActions, classifiedInstructions),
	}
}

// GetPoolAction determines the pool action type from instruction data
func (p *MeteoraDLMMPoolParser) GetPoolAction(data []byte) interface{} {
	if len(data) < 8 {
		return nil
	}
	disc := data[:8]

	// Check ADD_LIQUIDITY discriminators
	for name, d := range constants.DISCRIMINATORS.METEORA_DLMM.ADD_LIQUIDITY {
		if bytes.Equal(disc, d) {
			return &PoolActionResult{Name: name, Type: types.PoolEventTypeAdd}
		}
	}

	// Check REMOVE_LIQUIDITY discriminators
	for name, d := range constants.DISCRIMINATORS.METEORA_DLMM.REMOVE_LIQUIDITY {
		if bytes.Equal(disc, d) {
			return &PoolActionResult{Name: name, Type: types.PoolEventTypeRemove}
		}
	}

	// Pair creation (initialize_lb_pair and its variants)
	for name, d := range constants.DISCRIMINATORS.METEORA_DLMM.CREATE {
		if bytes.Equal(disc, d) {
			return &PoolActionResult{Name: name, Type: types.PoolEventTypeCreate}
		}
	}

	return nil
}

// ProcessLiquidity parses liquidity events
func (p *MeteoraDLMMPoolParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.METEORA.ID {
			data := p.Adapter.GetInstructionData(ci.Instruction)
			if constants.MatchDiscriminator(data, constants.DISCRIMINATORS.METEORA_DLMM.OTHER["rebalanceLiquidity"]) {
				events = append(events, p.parseRebalance(ci)...)
				continue
			}
			event := p.ParseInstruction(ci.Instruction, ci.ProgramId, ci.OuterIndex, ci.InnerIndex, p)
			if event != nil {
				events = append(events, *event)
			}
		}
	}

	return events
}

// ParseAddLiquidityEvent parses add liquidity event. Accounts (lb_clmm IDL
// 0.12.0): lb_pair 1, token_x_mint 7, token_y_mint 8; the one-side adds name
// only the deposited mint (token_mint 5).
func (p *MeteoraDLMMPoolParser) ParseAddLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	programId := p.Adapter.GetInstructionProgramId(instruction)
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	mintXIndex, mintYIndex := dlmmAddLayout(data)

	event := &types.PoolEvent{
		PoolEventBase: p.Adapter.GetPoolEventBase(types.PoolEventTypeAdd, programId),
	}
	if len(accounts) > 1 {
		event.PoolId = accounts[1]
		event.PoolLpMint = accounts[1]
	}
	p.setSides(event, accountAt(accounts, mintXIndex), accountAt(accounts, mintYIndex), transfers)
	return event
}

// dlmmAddLayout gives the token_x_mint and token_y_mint account indices of a
// DLMM add instruction (lb_clmm IDL 0.12.0); a one-side add names only the
// deposited mint (token_mint 5), and mintYIndex is -1
func dlmmAddLayout(data []byte) (mintXIndex, mintYIndex int) {
	dlmm := constants.DISCRIMINATORS.METEORA_DLMM.ADD_LIQUIDITY
	for _, name := range []string{"addLiquidityOneSide", "addLiquidityByStrategyOneSide", "addLiquidityOneSidePrecise", "addLiquidityOneSidePrecise2"} {
		if constants.MatchDiscriminator(data, dlmm[name]) {
			return 5, -1
		}
	}
	return 7, 8
}

// accountAt returns accounts[i], "" when i is out of range
func accountAt(accounts []string, i int) string {
	if i >= 0 && i < len(accounts) {
		return accounts[i]
	}
	return ""
}

// dlmmRemoveLayout gives the lb_pair and token_x/y_mint account indices of a
// DLMM instruction that pays tokens out of the pool (lb_clmm IDL 0.12.0)
func dlmmRemoveLayout(data []byte) (poolIndex, mintXIndex, mintYIndex int) {
	dlmm := constants.DISCRIMINATORS.METEORA_DLMM.REMOVE_LIQUIDITY
	switch {
	case constants.MatchDiscriminator(data, dlmm["claimFee"]):
		// claim_fee: lb_pair 0, position 1, ..., token_x_mint 9, token_y_mint 10
		return 0, 9, 10
	case constants.MatchDiscriminator(data, dlmm["claimFeeV2"]):
		// claim_fee2: lb_pair 0, position 1, sender 2, reserves 3-4, user tokens 5-6, mints 7-8
		return 0, 7, 8
	}
	// remove_liquidity*, remove_all_liquidity: position 0, lb_pair 1, ..., mints 7-8
	return 1, 7, 8
}

// ParseRemoveLiquidityEvent parses remove liquidity event. Fee claims
// (claim_fee, claim_fee2) pay the position's fees out of the pool reserves
// and are reported as REMOVE events, as upstream does (DAMM v2
// claim_position_fee likewise); their LpAmount stays empty. Reward claims
// are not liquidity events.
func (p *MeteoraDLMMPoolParser) ParseRemoveLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	poolIndex, mintXIndex, mintYIndex := dlmmRemoveLayout(data)
	programId := p.Adapter.GetInstructionProgramId(instruction)

	event := &types.PoolEvent{
		PoolEventBase: p.Adapter.GetPoolEventBase(types.PoolEventTypeRemove, programId),
		PoolId:        accountAt(accounts, poolIndex),
	}
	event.PoolLpMint = event.PoolId
	p.setSides(event, accountAt(accounts, mintXIndex), accountAt(accounts, mintYIndex), transfers)
	return event
}

// setSides sets the tokens of a DLMM ADD or REMOVE event: the pair's mints
// mintX and mintY ("" when the instruction does not name it), each token
// transfer on the side of its mint, 0 for a side of a known mint that moved
// nothing (one-sided deposits and withdrawals), and, as for the pair's create
// event, the quote mint (utils.GetTradeType) as token1.
func (p *MeteoraDLMMPoolParser) setSides(event *types.PoolEvent, mintX, mintY string, transfers []types.TransferData) {
	mints := [2]string{mintX, mintY}
	var sides [2]*types.TransferData
	for _, t := range p.Utils.GetLPTransfers(transfers) {
		t := t
		for side := 0; side < 2; side++ {
			if sides[side] == nil && (mints[side] == t.Info.Mint || (mints[side] == "" && t.Info.Mint != mints[1-side])) {
				sides[side] = &t
				mints[side] = t.Info.Mint
				break
			}
		}
	}
	if utils.GetTradeType(mints[0], mints[1]) == types.TradeTypeBuy {
		mints[0], mints[1] = mints[1], mints[0]
		sides[0], sides[1] = sides[1], sides[0]
	}

	decimals0, decimals1 := p.Adapter.GetTokenDecimals(mints[0]), p.Adapter.GetTokenDecimals(mints[1])
	event.Token0Mint, event.Token1Mint = mints[0], mints[1]
	event.Token0Decimals, event.Token1Decimals = &decimals0, &decimals1
	amount := func(side int) (*float64, string) {
		if t := sides[side]; t != nil && t.Info.TokenAmount.UIAmount != nil {
			return t.Info.TokenAmount.UIAmount, t.Info.TokenAmount.Amount
		}
		if mints[side] == "" {
			return nil, ""
		}
		zero := float64(0)
		return &zero, "0"
	}
	event.Token0Amount, event.Token0AmountRaw = amount(0)
	event.Token1Amount, event.Token1AmountRaw = amount(1)
}

// ParseCreateLiquidityEvent parses the creation of a pair. Accounts
// (lb_clmm IDL 0.12.0): lb_pair 0, token_mint_x 2, token_mint_y 3;
// initialize_permission_lb_pair has base first (lb_pair 1, mints 3/4). A
// pair is created empty; any tokens the instruction moves are reported. As
// for the other pools, the quote mint (SOL or a stablecoin) is token1.
func (p *MeteoraDLMMPoolParser) ParseCreateLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	poolIndex := 0
	if constants.MatchDiscriminator(data, constants.DISCRIMINATORS.METEORA_DLMM.CREATE["initializePermissionLbPair"]) {
		poolIndex = 1
	}
	if len(accounts) <= poolIndex+3 {
		return nil
	}
	token0Mint, token1Mint := accounts[poolIndex+2], accounts[poolIndex+3]
	if utils.GetTradeType(token0Mint, token1Mint) == types.TradeTypeBuy {
		token0Mint, token1Mint = token1Mint, token0Mint
	}
	token0Decimals := p.Adapter.GetTokenDecimals(token0Mint)
	token1Decimals := p.Adapter.GetTokenDecimals(token1Mint)

	event := &types.PoolEvent{
		PoolEventBase:  p.Adapter.GetPoolEventBase(types.PoolEventTypeCreate, p.Adapter.GetInstructionProgramId(instruction)),
		PoolId:         accounts[poolIndex],
		PoolLpMint:     accounts[poolIndex],
		Token0Mint:     token0Mint,
		Token1Mint:     token1Mint,
		Token0Decimals: &token0Decimals,
		Token1Decimals: &token1Decimals,
	}
	for _, t := range p.Utils.GetLPTransfers(transfers) {
		if t.Info.TokenAmount.UIAmount == nil {
			continue
		}
		switch t.Info.Mint {
		case token0Mint:
			event.Token0Amount, event.Token0AmountRaw = t.Info.TokenAmount.UIAmount, t.Info.TokenAmount.Amount
		case token1Mint:
			event.Token1Amount, event.Token1AmountRaw = t.Info.TokenAmount.UIAmount, t.Info.TokenAmount.Amount
		}
	}
	return event
}

// parseRebalance parses rebalance_liquidity, which can withdraw from some
// bins and deposit into others in one instruction. Its Rebalancing self-CPI
// event (lb_pair, position, owner, active_bin_id i32, x_withdrawn_amount,
// x_added_amount, y_withdrawn_amount, y_added_amount, ...) gives both
// sides: a REMOVE event with the withdrawn and an ADD event with the added
// amounts, each only when it moved tokens. Accounts: lb_pair 1,
// token_x_mint 7, token_y_mint 8.
func (p *MeteoraDLMMPoolParser) parseRebalance(ci types.ClassifiedInstruction) []types.PoolEvent {
	accounts := p.Adapter.GetInstructionAccounts(ci.Instruction)
	if len(accounts) < 9 {
		return nil
	}
	data := findEvent(p.Adapter, followingEvents(p.Adapter, p.ClassifiedInstructions, ci.ProgramId, ci.OuterIndex, ci.InnerIndex),
		constants.DISCRIMINATORS.METEORA_DLMM.EVENTS["rebalancing"])
	const amounts = 16 + 3*32 + 4
	if len(data) < amounts+4*8 || base58.Encode(data[16:48]) != accounts[1] {
		return nil
	}
	u64 := func(offset int) uint64 { return binary.LittleEndian.Uint64(data[offset : offset+8]) }
	xWithdrawn, xAdded, yWithdrawn, yAdded := u64(amounts), u64(amounts+8), u64(amounts+16), u64(amounts+24)

	mintX, mintY := accounts[7], accounts[8]
	token0Mint, token1Mint := mintX, mintY
	if utils.GetTradeType(mintX, mintY) == types.TradeTypeBuy {
		token0Mint, token1Mint = mintY, mintX
	}
	decimals0, decimals1 := p.Adapter.GetTokenDecimals(token0Mint), p.Adapter.GetTokenDecimals(token1Mint)

	var events []types.PoolEvent
	for _, side := range []struct {
		eventType types.PoolEventType
		x, y      uint64
	}{{types.PoolEventTypeRemove, xWithdrawn, yWithdrawn}, {types.PoolEventTypeAdd, xAdded, yAdded}} {
		if side.x == 0 && side.y == 0 {
			continue
		}
		amount0, amount1 := side.x, side.y
		if token0Mint != mintX {
			amount0, amount1 = side.y, side.x
		}
		ui0 := types.ConvertToUIAmountUint64(amount0, decimals0)
		ui1 := types.ConvertToUIAmountUint64(amount1, decimals1)
		d0, d1 := decimals0, decimals1
		event := types.PoolEvent{
			PoolEventBase:   p.Adapter.GetPoolEventBase(side.eventType, ci.ProgramId),
			PoolId:          accounts[1],
			PoolLpMint:      accounts[1],
			Token0Mint:      token0Mint,
			Token0Amount:    &ui0,
			Token0AmountRaw: strconv.FormatUint(amount0, 10),
			Token0Decimals:  &d0,
			Token1Mint:      token1Mint,
			Token1Amount:    &ui1,
			Token1AmountRaw: strconv.FormatUint(amount1, 10),
			Token1Decimals:  &d1,
		}
		event.Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
		events = append(events, event)
	}
	return events
}
