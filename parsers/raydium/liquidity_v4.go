package raydium

import (
	"bytes"
	"math/big"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// RaydiumV4PoolParser parses Raydium V4 liquidity operations
type RaydiumV4PoolParser struct {
	*RaydiumLiquidityParserBase
}

// NewRaydiumV4PoolParser creates a new Raydium V4 pool parser
func NewRaydiumV4PoolParser(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) *RaydiumV4PoolParser {
	return &RaydiumV4PoolParser{
		RaydiumLiquidityParserBase: NewRaydiumLiquidityParserBase(adapter, transferActions, classifiedInstructions),
	}
}

// GetPoolAction gets the pool action type from instruction data
func (p *RaydiumV4PoolParser) GetPoolAction(data []byte) interface{} {
	if len(data) < 1 {
		return nil
	}
	instructionType := data[:1]
	if bytes.Equal(instructionType, constants.DISCRIMINATORS.RAYDIUM.CREATE) {
		return types.PoolEventTypeCreate
	}
	if bytes.Equal(instructionType, constants.DISCRIMINATORS.RAYDIUM.ADD_LIQUIDITY) {
		return types.PoolEventTypeAdd
	}
	if bytes.Equal(instructionType, constants.DISCRIMINATORS.RAYDIUM.REMOVE_LIQUIDITY) {
		return types.PoolEventTypeRemove
	}
	return nil
}

// GetEventConfig gets the event configuration for a pool event type
func (p *RaydiumV4PoolParser) GetEventConfig(eventType types.PoolEventType, instructionType interface{}) *ParseEventConfig {
	configs := map[types.PoolEventType]*ParseEventConfig{
		types.PoolEventTypeCreate: {EventType: types.PoolEventTypeCreate, PoolIdIndex: 4, LpMintIndex: 7},
		types.PoolEventTypeAdd:    {EventType: types.PoolEventTypeAdd, PoolIdIndex: 1, LpMintIndex: 5},
		types.PoolEventTypeRemove: {EventType: types.PoolEventTypeRemove, PoolIdIndex: 1, LpMintIndex: 5},
	}
	return configs[eventType]
}

// ProcessLiquidity parses liquidity events. AMM v4 amounts come from the
// ray_log the instruction wrote (see eventFromRayLog), the instruction's
// transfers are the fallback when the log is missing or truncated, and the
// only source for the RaydiumAMM program, whose logs no fixture shows.
func (p *RaydiumV4PoolParser) ProcessLiquidity() []types.PoolEvent {
	var events []types.PoolEvent
	var rayLogs []utils.ProgramLog
	logsRead := false

	for _, ci := range p.ClassifiedInstructions {
		if ci.ProgramId == constants.DEX_PROGRAMS.RAYDIUM_V4.ID ||
			ci.ProgramId == constants.DEX_PROGRAMS.RAYDIUM_AMM.ID {
			var event *types.PoolEvent
			if ci.ProgramId == constants.DEX_PROGRAMS.RAYDIUM_V4.ID {
				if !logsRead {
					rayLogs, logsRead = p.Utils.GetRayLogs(), true
				}
				event = p.eventFromRayLog(ci, rayLogs)
			}
			if event == nil {
				event = p.ParseRaydiumInstruction(ci.Instruction, ci.ProgramId, ci.OuterIndex, ci.InnerIndex, p)
			}
			if event != nil {
				events = append(events, *event)
			}
		}
	}

	return events
}

// eventFromRayLog builds the liquidity event of an AMM v4 instruction from
// its ray_log (raydium-amm log.rs): InitLog of initialize2 (coin_amount,
// pc_amount; the LP minted to the user is not in the log and comes from the
// mintTo transfer), DepositLog (deduct_coin, deduct_pc, mint_lp) or
// WithdrawLog (out_coin, out_pc, withdraw_lp). initialize2 has lp_mint,
// coin_mint and pc_mint at 7/8/9; deposit and withdraw have lp_mint,
// coin_vault and pc_vault at 5/6/7. As for transfers (GetLPTransfers),
// token1 is the quote side. It returns nil when the instruction wrote no
// such log.
func (p *RaydiumV4PoolParser) eventFromRayLog(ci types.ClassifiedInstruction, rayLogs []utils.ProgramLog) *types.PoolEvent {
	var eventType types.PoolEventType
	var coin, pc, lp *big.Int
	for _, l := range utils.FindProgramLogs(rayLogs, ci.ProgramId, ci.OuterIndex, ci.InnerIndex) {
		switch log := DecodeRaydiumLog(l.Data).(type) {
		case *InitLog:
			eventType, coin, pc = types.PoolEventTypeCreate, log.CoinAmount, log.PcAmount
		case *DepositLog:
			eventType, coin, pc, lp = types.PoolEventTypeAdd, log.DeductCoin, log.DeductPc, log.MintLp
		case *WithdrawLog:
			eventType, coin, pc, lp = types.PoolEventTypeRemove, log.OutCoin, log.OutPc, log.WithdrawLp
		}
	}
	if coin == nil || p.GetPoolAction(p.Adapter.GetInstructionData(ci.Instruction)) != eventType {
		return nil
	}
	accounts := p.Adapter.GetInstructionAccounts(ci.Instruction)
	config := p.GetEventConfig(eventType, eventType)
	var coinMint, pcMint string
	if eventType == types.PoolEventTypeCreate {
		if len(accounts) > 9 {
			coinMint, pcMint = accounts[8], accounts[9]
		}
	} else if len(accounts) > 7 {
		coinMint, pcMint = p.Adapter.GetSplTokenMint(accounts[6]), p.Adapter.GetSplTokenMint(accounts[7])
	}
	if coinMint == "" || pcMint == "" || config.PoolIdIndex >= len(accounts) || config.LpMintIndex >= len(accounts) {
		return nil
	}
	mint0, mint1, amount0, amount1 := coinMint, pcMint, coin, pc
	if mint0 == constants.TOKENS.SOL || (p.Adapter.IsSupportedToken(mint0) && !p.Adapter.IsSupportedToken(mint1)) {
		mint0, mint1, amount0, amount1 = mint1, mint0, amount1, amount0
	}
	lpMint := accounts[config.LpMintIndex]
	if lp == nil {
		lp = new(big.Int)
		for _, t := range p.GetTransfersForInstruction(ci.ProgramId, ci.OuterIndex, ci.InnerIndex, nil) {
			if t.Type == "mintTo" && t.Info.Mint == lpMint {
				lp.SetString(t.Info.TokenAmount.Amount, 10)
				break
			}
		}
	}

	base := p.Adapter.GetPoolEventBase(eventType, ci.ProgramId)
	base.Idx = utils.FormatIdx(ci.OuterIndex, ci.InnerIndex)
	token0Decimals := p.Adapter.GetTokenDecimals(mint0)
	token1Decimals := p.Adapter.GetTokenDecimals(mint1)
	token0Amount := types.ConvertToUIAmount(amount0, token0Decimals)
	token1Amount := types.ConvertToUIAmount(amount1, token1Decimals)
	lpAmount := types.ConvertToUIAmount(lp, p.Adapter.GetTokenDecimals(lpMint))
	return &types.PoolEvent{
		PoolEventBase:   base,
		PoolId:          accounts[config.PoolIdIndex],
		PoolLpMint:      lpMint,
		Token0Mint:      mint0,
		Token0Amount:    &token0Amount,
		Token0AmountRaw: amount0.String(),
		Token0Decimals:  &token0Decimals,
		Token1Mint:      mint1,
		Token1Amount:    &token1Amount,
		Token1AmountRaw: amount1.String(),
		Token1Decimals:  &token1Decimals,
		LpAmount:        &lpAmount,
		LpAmountRaw:     lp.String(),
	}
}
