package meteora

import (
	"bytes"
	"encoding/binary"
	"strconv"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// MeteoraPoolsParser parses Meteora DAMM pool events
type MeteoraPoolsParser struct {
	*MeteoraLiquidityParserBase
}

// NewMeteoraPoolsParser creates a new Meteora pools parser
func NewMeteoraPoolsParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *MeteoraPoolsParser {
	return &MeteoraPoolsParser{
		MeteoraLiquidityParserBase: NewMeteoraLiquidityParserBase(adapter, transferActions, classifiedInstructions),
	}
}

// dammV1Layout describes a DAMM v1 (Meteora pools, amm 0.5.2 IDL) liquidity
// instruction: the account indices of the pool LP mint and token mints, and
// the u64 offsets of the token amounts in the args (-1 when the args hold no
// fixed-position amount). Data amounts are used only when the transfers are
// missing; for add/remove they are slippage limits.
type dammV1Layout struct {
	name           string
	eventType      types.PoolEventType
	lpMintIndex    int
	mintAIndex     int
	mintBIndex     int
	amountAOffset  int
	amountBOffset  int
	lpAmountOffset int
}

// dammV1Layouts lists the DAMM v1 liquidity instructions. Pool is account 0
// in all of them. claim_fee withdraws the fees of a lock escrow from the
// pool (REMOVE, like the DLMM and DAMM v2 fee claims); partner_claim_fee
// pays protocol fees to a partner and is not a liquidity event.
func dammV1Layouts() []struct {
	disc   []byte
	layout dammV1Layout
} {
	d := constants.DISCRIMINATORS.METEORA_DAMM
	create := types.PoolEventTypeCreate
	add := types.PoolEventTypeAdd
	remove := types.PoolEventTypeRemove
	return []struct {
		disc   []byte
		layout dammV1Layout
	}{
		// pool 0, config 1, lp_mint 2, token_a_mint 3, token_b_mint 4; args token_a_amount, token_b_amount
		{d.CREATE, dammV1Layout{"initializePermissionlessConstantProductPoolWithConfig", create, 2, 3, 4, 8, 16, -1}},
		{d.CREATE_WITH_CONFIG2, dammV1Layout{"initializePermissionlessConstantProductPoolWithConfig2", create, 2, 3, 4, 8, 16, -1}},
		// pool 0, lp_mint 1, token_a_mint 2, token_b_mint 3
		{d.CREATE_CUSTOMIZABLE, dammV1Layout{"initializeCustomizablePermissionlessConstantProductPool", create, 1, 2, 3, 8, 16, -1}},
		// args start with a curve_type enum of variable size: no data amounts
		{d.CREATE_PERMISSIONLESS_POOL, dammV1Layout{"initializePermissionlessPool", create, 1, 2, 3, -1, -1, -1}},
		{d.CREATE_PERMISSIONLESS_POOL_WITH_FEE_TIER, dammV1Layout{"initializePermissionlessPoolWithFeeTier", create, 1, 2, 3, -1, -1, -1}},
		{d.CREATE_PERMISSIONED_POOL, dammV1Layout{"initializePermissionedPool", create, 1, 2, 3, -1, -1, -1}},
		// pool 0, lp_mint 1; add_balance_liquidity args: pool_token_amount, maximum_token_a_amount, maximum_token_b_amount
		{d.ADD_LIQUIDITY, dammV1Layout{"addBalanceLiquidity", add, 1, -1, -1, 16, 24, 8}},
		// add_imbalance_liquidity args: minimum_pool_token_amount, token_a_amount, token_b_amount
		{d.ADD_IMBALANCE_LIQUIDITY, dammV1Layout{"addImbalanceLiquidity", add, 1, -1, -1, 16, 24, -1}},
		// bootstrap_liquidity args: token_a_amount, token_b_amount
		{d.BOOTSTRAP_LIQUIDITY, dammV1Layout{"bootstrapLiquidity", add, 1, -1, -1, 8, 16, -1}},
		// remove_balance_liquidity args: pool_token_amount, minimum_a_token_out, minimum_b_token_out
		{d.REMOVE_LIQUIDITY, dammV1Layout{"removeBalanceLiquidity", remove, 1, -1, -1, 16, 24, 8}},
		// remove_liquidity_single_side args: pool_token_amount, minimum_out_amount (one token)
		{d.REMOVE_LIQUIDITY_SINGLE_SIDE, dammV1Layout{"removeLiquiditySingleSide", remove, 1, -1, -1, -1, -1, 8}},
		// claim_fee: pool 0, lp_mint 1; args max_amount
		{d.CLAIM_FEE, dammV1Layout{"claimFee", remove, 1, -1, -1, -1, -1, -1}},
	}
}

// getLayout returns the layout of a DAMM v1 liquidity instruction
func (p *MeteoraPoolsParser) getLayout(data []byte) *dammV1Layout {
	if len(data) < 8 {
		return nil
	}
	for _, l := range dammV1Layouts() {
		if bytes.Equal(data[:8], l.disc) {
			layout := l.layout
			return &layout
		}
	}
	return nil
}

// GetPoolAction determines the pool action type from instruction data
func (p *MeteoraPoolsParser) GetPoolAction(data []byte) interface{} {
	if layout := p.getLayout(data); layout != nil {
		return &PoolActionResult{Name: layout.name, Type: layout.eventType}
	}
	return nil
}

// ProcessLiquidity parses liquidity events
func (p *MeteoraPoolsParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.METEORA_DAMM.ID {
			event := p.ParseInstruction(ci.Instruction, ci.ProgramId, ci.OuterIndex, ci.InnerIndex, p)
			if event != nil {
				events = append(events, *event)
			}
		}
	}

	return events
}

// ParseCreateLiquidityEvent parses create pool event
func (p *MeteoraPoolsParser) ParseCreateLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	return p.parseEvent(instruction, index, data, transfers, types.PoolEventTypeCreate)
}

// ParseAddLiquidityEvent parses add liquidity event
func (p *MeteoraPoolsParser) ParseAddLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	return p.parseEvent(instruction, index, data, transfers, types.PoolEventTypeAdd)
}

// ParseRemoveLiquidityEvent parses remove liquidity event
func (p *MeteoraPoolsParser) ParseRemoveLiquidityEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
) *types.PoolEvent {
	return p.parseEvent(instruction, index, data, transfers, types.PoolEventTypeRemove)
}

// parseEvent builds a DAMM v1 liquidity event. Token amounts come from the
// transfers (the vault program CPIs are grouped with the pool instruction);
// the LP amount from the mintTo (burn for REMOVE) of the pool LP mint.
func (p *MeteoraPoolsParser) parseEvent(
	instruction interface{},
	index int,
	data []byte,
	transfers []types.TransferData,
	eventType types.PoolEventType,
) *types.PoolEvent {
	layout := p.getLayout(data)
	if layout == nil {
		layout = &dammV1Layout{eventType: eventType, lpMintIndex: 1, mintAIndex: -1, mintBIndex: -1, amountAOffset: -1, amountBOffset: -1, lpAmountOffset: -1}
	}
	accounts := p.Adapter.GetInstructionAccounts(instruction)
	accountAt := func(i int) string {
		if i >= 0 && i < len(accounts) {
			return accounts[i]
		}
		return ""
	}
	lpMint := accountAt(layout.lpMintIndex)

	lpTransfers := p.Utils.GetLPTransfers(transfers)
	var token0, token1, lpToken *types.TransferData
	if len(lpTransfers) > 0 {
		token0 = &lpTransfers[0]
	}
	if len(lpTransfers) > 1 {
		token1 = &lpTransfers[1]
	}

	lpType := "mintTo"
	if eventType == types.PoolEventTypeRemove {
		lpType = "burn"
	}
	for i := range transfers {
		if transfers[i].Type == lpType && (lpMint == "" || transfers[i].Info.Mint == lpMint) {
			lpToken = &transfers[i]
			break
		}
	}

	var token0Mint, token1Mint string
	if token0 != nil {
		token0Mint = token0.Info.Mint
	} else {
		token0Mint = accountAt(layout.mintAIndex)
	}
	if token1 != nil {
		token1Mint = token1.Info.Mint
	} else {
		token1Mint = accountAt(layout.mintBIndex)
	}

	programId := p.Adapter.GetInstructionProgramId(instruction)
	token0Decimals := p.Adapter.GetTokenDecimals(token0Mint)
	token1Decimals := p.Adapter.GetTokenDecimals(token1Mint)

	event := &types.PoolEvent{
		PoolEventBase:  p.Adapter.GetPoolEventBase(eventType, programId),
		PoolId:         accountAt(0),
		PoolLpMint:     lpMint,
		Token0Mint:     token0Mint,
		Token1Mint:     token1Mint,
		Token0Decimals: &token0Decimals,
		Token1Decimals: &token1Decimals,
	}
	event.Idx = strconv.Itoa(index)

	// Token amounts from transfers, else from the instruction args (token A
	// and token B when the mints come from the accounts in A/B order)
	readU64 := func(offset int) (uint64, bool) {
		if offset < 0 || offset+8 > len(data) {
			return 0, false
		}
		return binary.LittleEndian.Uint64(data[offset : offset+8]), true
	}
	if token0 != nil && token0.Info.TokenAmount.UIAmount != nil {
		event.Token0Amount = token0.Info.TokenAmount.UIAmount
		event.Token0AmountRaw = token0.Info.TokenAmount.Amount
	} else if amt, ok := readU64(layout.amountAOffset); ok && token1 == nil {
		uiAmt := types.ConvertToUIAmountUint64(amt, token0Decimals)
		event.Token0Amount = &uiAmt
		event.Token0AmountRaw = strconv.FormatUint(amt, 10)
	}
	if token1 != nil && token1.Info.TokenAmount.UIAmount != nil {
		event.Token1Amount = token1.Info.TokenAmount.UIAmount
		event.Token1AmountRaw = token1.Info.TokenAmount.Amount
	} else if amt, ok := readU64(layout.amountBOffset); ok && token0 == nil {
		uiAmt := types.ConvertToUIAmountUint64(amt, token1Decimals)
		event.Token1Amount = &uiAmt
		event.Token1AmountRaw = strconv.FormatUint(amt, 10)
	}

	lpDecimals := p.Adapter.GetTokenDecimals(lpMint)
	if lpToken != nil && lpToken.Info.TokenAmount.UIAmount != nil {
		event.LpAmount = lpToken.Info.TokenAmount.UIAmount
		event.LpAmountRaw = lpToken.Info.TokenAmount.Amount
	} else if amt, ok := readU64(layout.lpAmountOffset); ok {
		uiAmt := types.ConvertToUIAmountUint64(amt, lpDecimals)
		event.LpAmount = &uiAmt
		event.LpAmountRaw = strconv.FormatUint(amt, 10)
	}

	return event
}
