package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	gojson "github.com/goccy/go-json"
	"github.com/mr-tron/base58"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// constIxDisc is the Anchor instruction discriminator sha256("global:<name>")[:8].
func constIxDisc(name string) []byte {
	h := sha256.Sum256([]byte("global:" + name))
	return h[:8]
}

// constLogEventDisc is the Anchor event discriminator sha256("event:<Name>")[:8],
// as found in "Program data:" log lines (emit!).
func constLogEventDisc(name string) []byte {
	h := sha256.Sum256([]byte("event:" + name))
	return h[:8]
}

// constCpiEventDisc is the discriminator of an Anchor self-CPI event (emit_cpi!):
// the event-instruction tag e445a52e51cb9a1d followed by the event discriminator.
func constCpiEventDisc(name string) []byte {
	return append([]byte{0xe4, 0x45, 0xa5, 0x2e, 0x51, 0xcb, 0x9a, 0x1d}, constLogEventDisc(name)...)
}

// constAnchorTable maps every Anchor discriminator in constants.DISCRIMINATORS (by field
// path, as produced by walkDiscriminators) to the value recomputed from its IDL name.
// Names are taken from the on-chain IDLs (camelCase names of older Anchor IDLs are hashed
// in snake_case, as Anchor does).
func constAnchorTable() map[string][]byte {
	ix, logEv, cpiEv := constIxDisc, constLogEventDisc, constCpiEventDisc
	return map[string][]byte{
		// amm WP: event and position discriminators
		"RAYDIUM_CPMM.SWAP_EVENT": logEv("SwapEvent"), "RAYDIUM_CPMM.LP_CHANGE_EVENT": logEv("LpChangeEvent"),
		"METEORA_DLMM.EVENTS[swap]": cpiEv("Swap"), "METEORA_DLMM.EVENTS[swap2Evt]": cpiEv("Swap2Evt"), "METEORA_DLMM.EVENTS[claimFee]": cpiEv("ClaimFee"), "METEORA_DLMM.EVENTS[claimFee2]": cpiEv("ClaimFee2"), "METEORA_DLMM.EVENTS[positionCreate]": cpiEv("PositionCreate"), "METEORA_DLMM.EVENTS[positionClose]": cpiEv("PositionClose"), "METEORA_DLMM.EVENTS[lbPairCreate]": cpiEv("LbPairCreate"), "METEORA_DLMM.EVENTS[rebalancing]": cpiEv("Rebalancing"),
		"METEORA_DLMM.POSITION[initializePosition]": ix("initialize_position"), "METEORA_DLMM.POSITION[initializePosition2]": ix("initialize_position2"), "METEORA_DLMM.POSITION[closePosition]": ix("close_position"), "METEORA_DLMM.POSITION[closePosition2]": ix("close_position2"),
		"METEORA_DAMM.SWAP_EVENT": logEv("Swap"), "METEORA_DAMM.ADD_LIQUIDITY_EVENT": logEv("AddLiquidity"), "METEORA_DAMM.REMOVE_LIQUIDITY_EVENT": logEv("RemoveLiquidity"), "METEORA_DAMM.BOOTSTRAP_LIQUIDITY_EVENT": logEv("BootstrapLiquidity"), "METEORA_DAMM.POOL_CREATED_EVENT": logEv("PoolCreated"), "METEORA_DAMM.SET_POOL_FEES_EVENT": logEv("SetPoolFees"), "METEORA_DAMM.CLAIM_FEE_EVENT": logEv("ClaimFee"),
		"METEORA_DAMM_V2.EVT_SWAP": cpiEv("EvtSwap"), "METEORA_DAMM_V2.EVT_SWAP2": cpiEv("EvtSwap2"), "METEORA_DAMM_V2.EVT_LIQUIDITY_CHANGE": cpiEv("EvtLiquidityChange"), "METEORA_DAMM_V2.EVT_ADD_LIQUIDITY": cpiEv("EvtAddLiquidity"), "METEORA_DAMM_V2.EVT_REMOVE_LIQUIDITY": cpiEv("EvtRemoveLiquidity"), "METEORA_DAMM_V2.EVT_INITIALIZE_POOL": cpiEv("EvtInitializePool"), "METEORA_DAMM_V2.EVT_CLAIM_POSITION_FEE": cpiEv("EvtClaimPositionFee"), "METEORA_DAMM_V2.EVT_CLOSE_POSITION": cpiEv("EvtClosePosition"), "METEORA_DAMM_V2.EVT_CLAIM_REWARD": cpiEv("EvtClaimReward"), "METEORA_DAMM_V2.EVT_FUND_REWARD": cpiEv("EvtFundReward"), "METEORA_DAMM_V2.EVT_INITIALIZE_REWARD": cpiEv("EvtInitializeReward"), "METEORA_DAMM_V2.EVT_CREATE_CONFIG": cpiEv("EvtCreateConfig"), "METEORA_DAMM_V2.EVT_CREATE_DYNAMIC_CONFIG": cpiEv("EvtCreateDynamicConfig"), "METEORA_DAMM_V2.EVT_UPDATE_DELEGATE_PERMISSION": cpiEv("EvtUpdateDelegatePermission"), "METEORA_DAMM_V2.EVT_WITHDRAW_DEAD_LIQUIDITY_REWARD": cpiEv("EvtWithdrawDeadLiquidityReward"), "METEORA_DAMM_V2.EVT_CLAIM_PROTOCOL_FEE2": cpiEv("EvtClaimProtocolFee2"),
		"ORCA.POOL_INITIALIZED_EVENT": logEv("PoolInitialized"), "ORCA.LIQUIDITY_REPOSITIONED_EVENT": logEv("LiquidityRepositioned"), "ORCA.POSITION_OPENED_EVENT": logEv("PositionOpened"),
		// Jupiter V6
		"JUPITER.ROUTE_EVENT":                            cpiEv("SwapEvent"),
		"PUMPFUN.MIGRATE_BONDING_CURVE_CREATOR":          ix("migrate_bonding_curve_creator"),
		"PUMPSWAP.BOOST_BUY_AND_BURN":                    ix("boost_buy_and_burn"),
		"PUMPSWAP.BOOST_BUY_AND_BURN_EVENT":              cpiEv("BoostBuyAndBurnEvent"),
		"RAYDIUM_LCP.CREATE_EVENT_LOG":                   logEv("PoolCreateEvent"),
		"RAYDIUM_LCP.CLAIM_VESTED_EVENT":                 cpiEv("ClaimVestedEvent"),
		"RAYDIUM_LCP.CREATE_VESTING_EVENT":               cpiEv("CreateVestingEvent"),
		"JUPITER.ROUTE":                                  ix("route"),
		"JUPITER.ROUTE_EXACT_OUT":                        ix("exact_out_route"),
		"JUPITER.ROUTE_WITH_TOKEN_LEDGER":                ix("route_with_token_ledger"),
		"JUPITER.SHARE_ACCOUNTS_ROUTE":                   ix("shared_accounts_route"),
		"JUPITER.SHARE_ACCOUNTS_EXACT_OUT_ROUTE":         ix("shared_accounts_exact_out_route"),
		"JUPITER.SHARE_ACCOUNTS_ROUTE_WITH_TOKEN_LEDGER": ix("shared_accounts_route_with_token_ledger"),
		"JUPITER.ROUTE_V2":                               ix("route_v2"),
		"JUPITER.SHARED_ACCOUNTS_ROUTE_V2":               ix("shared_accounts_route_v2"),
		"JUPITER.EXACT_OUT_ROUTE_V2":                     ix("exact_out_route_v2"),
		"JUPITER.SHARED_ACCOUNTS_EXACT_OUT_ROUTE_V2":     ix("shared_accounts_exact_out_route_v2"),
		"JUPITER.SWAPS_EVENT":                            cpiEv("SwapsEvent"),
		"JUPITER.FEE_EVENT":                              cpiEv("FeeEvent"),
		// Jupiter DCA
		"JUPITER_DCA.FILLED":      cpiEv("Filled"),
		"JUPITER_DCA.CLOSE_DCA":   ix("close_dca"),
		"JUPITER_DCA.OPEN_DCA":    ix("open_dca"),
		"JUPITER_DCA.OPEN_DCA_V2": ix("open_dca_v2"),
		// Jupiter Limit Order v1 / v2
		"JUPITER_LIMIT_ORDER.CANCEL_ORDER":               ix("cancel_order"),
		"JUPITER_LIMIT_ORDER.CREATE_ORDER":               ix("initialize_order"),
		"JUPITER_LIMIT_ORDER.TRADE_EVENT":                cpiEv("TradeEvent"),
		"JUPITER_LIMIT_ORDER.UNKNOWN":                    ix("fill_order"),
		"JUPITER_LIMIT_ORDER.FLASH_FILL_ORDER":           ix("flash_fill_order"),
		"JUPITER_LIMIT_ORDER_V2.CANCEL_ORDER":            ix("cancel_order"),
		"JUPITER_LIMIT_ORDER_V2.CREATE_ORDER_EVENT":      cpiEv("CreateOrderEvent"),
		"JUPITER_LIMIT_ORDER_V2.TRADE_EVENT":             cpiEv("TradeEvent"),
		"JUPITER_LIMIT_ORDER_V2.UNKNOWN":                 ix("fill_order"),
		"JUPITER_LIMIT_ORDER_V2.FLASH_FILL_ORDER":        ix("flash_fill_order"),
		"JUPITER_LIMIT_ORDER_V2.CANCEL_DUST_ORDER":       ix("cancel_dust_order"),
		"JUPITER_LIMIT_ORDER_V2.CANCEL_ORDER_EVENT":      cpiEv("CancelOrderEvent"),
		"JUPITER_LIMIT_ORDER_V2.CANCEL_DUST_ORDER_EVENT": cpiEv("CancelDustOrderEvent"),
		// Jupiter Value Average
		"JUPITER_VA.FILL_EVENT":     cpiEv("Fill"),
		"JUPITER_VA.OPEN_EVENT":     cpiEv("Open"),
		"JUPITER_VA.CLOSE_EVENT":    cpiEv("Close"),
		"JUPITER_VA.DEPOSIT_EVENT":  cpiEv("Deposit"),
		"JUPITER_VA.WITHDRAW_EVENT": cpiEv("Withdraw"),
		// Jupiter Z
		"JUPITER_Z.FILL": ix("fill"),
		// Pump.fun
		"PUMPFUN.CREATE":                ix("create"),
		"PUMPFUN.MIGRATE":               ix("migrate"),
		"PUMPFUN.BUY":                   ix("buy"),
		"PUMPFUN.SELL":                  ix("sell"),
		"PUMPFUN.TRADE_EVENT":           cpiEv("TradeEvent"),
		"PUMPFUN.CREATE_EVENT":          cpiEv("CreateEvent"),
		"PUMPFUN.COMPLETE_EVENT":        cpiEv("CompleteEvent"),
		"PUMPFUN.MIGRATE_EVENT":         cpiEv("CompletePumpAmmMigrationEvent"),
		"PUMPFUN.BUY_V2":                ix("buy_v2"),
		"PUMPFUN.SELL_V2":               ix("sell_v2"),
		"PUMPFUN.BUY_EXACT_SOL_IN":      ix("buy_exact_sol_in"),
		"PUMPFUN.BUY_EXACT_QUOTE_IN_V2": ix("buy_exact_quote_in_v2"),
		"PUMPFUN.CREATE_V2":             ix("create_v2"),
		"PUMPFUN.MIGRATE_V2":            ix("migrate_v2"),
		// PumpSwap
		"PUMPSWAP.CREATE_POOL":            ix("create_pool"),
		"PUMPSWAP.ADD_LIQUIDITY":          ix("deposit"),
		"PUMPSWAP.REMOVE_LIQUIDITY":       ix("withdraw"),
		"PUMPSWAP.BUY":                    ix("buy"),
		"PUMPSWAP.SELL":                   ix("sell"),
		"PUMPSWAP.CREATE_POOL_EVENT":      cpiEv("CreatePoolEvent"),
		"PUMPSWAP.ADD_LIQUIDITY_EVENT":    cpiEv("DepositEvent"),
		"PUMPSWAP.REMOVE_LIQUIDITY_EVENT": cpiEv("WithdrawEvent"),
		"PUMPSWAP.BUY_EVENT":              cpiEv("BuyEvent"),
		"PUMPSWAP.SELL_EVENT":             cpiEv("SellEvent"),
		"PUMPSWAP.BUY_EXACT_QUOTE_IN":     ix("buy_exact_quote_in"),
		// Moonit (token_launchpad IDL)
		"MOONIT.BUY":     ix("buy"),
		"MOONIT.SELL":    ix("sell"),
		"MOONIT.CREATE":  ix("token_mint"),
		"MOONIT.MIGRATE": ix("migrate_funds"),
		// Raydium CLMM
		"RAYDIUM_CL.CREATE.OPEN_POSITION":                     ix("open_position"),
		"RAYDIUM_CL.CREATE.OPEN_POSITION_V2":                  ix("open_position_v2"),
		"RAYDIUM_CL.CREATE.CREATE_POOL":                       ix("create_pool"),
		"RAYDIUM_CL.CREATE.CREATE_CUSTOMIZABLE_POOL":          ix("create_customizable_pool"),
		"RAYDIUM_CL.ADD_LIQUIDITY.INCREASE_LIQUIDITY":         ix("increase_liquidity"),
		"RAYDIUM_CL.ADD_LIQUIDITY.INCREASE_LIQUIDITY_V2":      ix("increase_liquidity_v2"),
		"RAYDIUM_CL.ADD_LIQUIDITY.OPEN_POSITION_WITH_TOKEN22": ix("open_position_with_token22_nft"),
		"RAYDIUM_CL.REMOVE_LIQUIDITY.DECREASE_LIQUIDITY":      ix("decrease_liquidity"),
		"RAYDIUM_CL.REMOVE_LIQUIDITY.DECREASE_LIQUIDITY_V2":   ix("decrease_liquidity_v2"),
		"RAYDIUM_CL.REMOVE_LIQUIDITY.CLOSE_POSITION":          ix("close_position"),
		"RAYDIUM_CL.SWAP.SWAP":                                ix("swap"),
		"RAYDIUM_CL.SWAP.SWAP_V2":                             ix("swap_v2"),
		"RAYDIUM_CL.SWAP.SWAP_ROUTER_BASE_IN":                 ix("swap_router_base_in"),
		"RAYDIUM_CL.EVENTS.COLLECT_PERSONAL_FEE":              logEv("CollectPersonalFeeEvent"),
		"RAYDIUM_CL.EVENTS.COLLECT_PROTOCOL_FEE":              logEv("CollectProtocolFeeEvent"),
		"RAYDIUM_CL.EVENTS.CONFIG_CHANGE":                     logEv("ConfigChangeEvent"),
		"RAYDIUM_CL.EVENTS.CREATE_PERSONAL_POSITION":          logEv("CreatePersonalPositionEvent"),
		"RAYDIUM_CL.EVENTS.DECREASE_LIQUIDITY":                logEv("DecreaseLiquidityEvent"),
		"RAYDIUM_CL.EVENTS.INCREASE_LIQUIDITY":                logEv("IncreaseLiquidityEvent"),
		"RAYDIUM_CL.EVENTS.LIQUIDITY_CALCULATE":               logEv("LiquidityCalculateEvent"),
		"RAYDIUM_CL.EVENTS.LIQUIDITY_CHANGE":                  logEv("LiquidityChangeEvent"),
		"RAYDIUM_CL.EVENTS.POOL_CREATED":                      logEv("PoolCreatedEvent"),
		"RAYDIUM_CL.EVENTS.SWAP":                              logEv("SwapEvent"),
		"RAYDIUM_CL.EVENTS.UPDATE_REWARD_INFOS":               logEv("UpdateRewardInfosEvent"),
		"RAYDIUM_CL.EVENTS.OPEN_LIMIT_ORDER":                  logEv("OpenLimitOrderEvent"),
		"RAYDIUM_CL.EVENTS.INCREASE_LIMIT_ORDER":              logEv("IncreaseLimitOrderEvent"),
		"RAYDIUM_CL.EVENTS.DECREASE_LIMIT_ORDER":              logEv("DecreaseLimitOrderEvent"),
		"RAYDIUM_CL.EVENTS.SETTLE_LIMIT_ORDER":                logEv("SettleLimitOrderEvent"),
		"RAYDIUM_CL.LIMIT_ORDER.OPEN_LIMIT_ORDER":             ix("open_limit_order"),
		"RAYDIUM_CL.LIMIT_ORDER.INCREASE_LIMIT_ORDER":         ix("increase_limit_order"),
		"RAYDIUM_CL.LIMIT_ORDER.DECREASE_LIMIT_ORDER":         ix("decrease_limit_order"),
		"RAYDIUM_CL.LIMIT_ORDER.SETTLE_LIMIT_ORDER":           ix("settle_limit_order"),
		"RAYDIUM_CL.LIMIT_ORDER.CLOSE_LIMIT_ORDER":            ix("close_limit_order"),
		"RAYDIUM_CL.OTHER.COLLECT_FUND_FEE":                   ix("collect_fund_fee"),
		"RAYDIUM_CL.OTHER.COLLECT_PROTOCOL_FEE":               ix("collect_protocol_fee"),
		"RAYDIUM_CL.OTHER.COLLECT_REMAINING_REWARDS":          ix("collect_remaining_rewards"),
		"RAYDIUM_CL.OTHER.INITIALIZE_REWARD":                  ix("initialize_reward"),
		"RAYDIUM_CL.OTHER.SET_REWARD_PARAMS":                  ix("set_reward_params"),
		// Raydium CPMM
		"RAYDIUM_CPMM.CREATE":                     ix("initialize"),
		"RAYDIUM_CPMM.ADD_LIQUIDITY":              ix("deposit"),
		"RAYDIUM_CPMM.REMOVE_LIQUIDITY":           ix("withdraw"),
		"RAYDIUM_CPMM.INITIALIZE_WITH_PERMISSION": ix("initialize_with_permission"),
		"RAYDIUM_CPMM.SWAP_BASE_INPUT":            ix("swap_base_input"),
		"RAYDIUM_CPMM.SWAP_BASE_OUTPUT":           ix("swap_base_output"),
		"RAYDIUM_CPMM.COLLECT_CREATOR_FEE":        ix("collect_creator_fee"),
		"RAYDIUM_CPMM.COLLECT_FUND_FEE":           ix("collect_fund_fee"),
		"RAYDIUM_CPMM.COLLECT_PROTOCOL_FEE":       ix("collect_protocol_fee"),
		// Raydium LaunchLab
		"RAYDIUM_LCP.CREATE_EVENT":               cpiEv("PoolCreateEvent"),
		"RAYDIUM_LCP.TRADE_EVENT":                cpiEv("TradeEvent"),
		"RAYDIUM_LCP.MIGRATE_TO_AMM":             ix("migrate_to_amm"),
		"RAYDIUM_LCP.MIGRATE_TO_CPSWAP":          ix("migrate_to_cpswap"),
		"RAYDIUM_LCP.BUY_EXACT_IN":               ix("buy_exact_in"),
		"RAYDIUM_LCP.BUY_EXACT_OUT":              ix("buy_exact_out"),
		"RAYDIUM_LCP.SELL_EXACT_IN":              ix("sell_exact_in"),
		"RAYDIUM_LCP.SELL_EXACT_OUT":             ix("sell_exact_out"),
		"RAYDIUM_LCP.INITIALIZE":                 ix("initialize"),
		"RAYDIUM_LCP.INITIALIZE_V2":              ix("initialize_v2"),
		"RAYDIUM_LCP.INITIALIZE_WITH_TOKEN_2022": ix("initialize_with_token_2022"),
		// Meteora DLMM (lb_clmm 0.12.0)
		"METEORA_DLMM.ADD_LIQUIDITY[addLiquidity]":                         ix("add_liquidity"),
		"METEORA_DLMM.ADD_LIQUIDITY[addLiquidityByStrategy]":               ix("add_liquidity_by_strategy"),
		"METEORA_DLMM.ADD_LIQUIDITY[addLiquidityByStrategy2]":              ix("add_liquidity_by_strategy2"),
		"METEORA_DLMM.ADD_LIQUIDITY[addLiquidityByStrategyOneSide]":        ix("add_liquidity_by_strategy_one_side"),
		"METEORA_DLMM.ADD_LIQUIDITY[addLiquidityOneSide]":                  ix("add_liquidity_one_side"),
		"METEORA_DLMM.ADD_LIQUIDITY[addLiquidityOneSidePrecise]":           ix("add_liquidity_one_side_precise"),
		"METEORA_DLMM.ADD_LIQUIDITY[addLiquidityByWeight]":                 ix("add_liquidity_by_weight"),
		"METEORA_DLMM.ADD_LIQUIDITY[addLiquidity2]":                        ix("add_liquidity2"),
		"METEORA_DLMM.ADD_LIQUIDITY[addLiquidityByWeight2]":                ix("add_liquidity_by_weight2"),
		"METEORA_DLMM.ADD_LIQUIDITY[addLiquidityOneSidePrecise2]":          ix("add_liquidity_one_side_precise2"),
		"METEORA_DLMM.REMOVE_LIQUIDITY[removeLiquidity]":                   ix("remove_liquidity"),
		"METEORA_DLMM.REMOVE_LIQUIDITY[removeLiquidityByRange]":            ix("remove_liquidity_by_range"),
		"METEORA_DLMM.REMOVE_LIQUIDITY[removeLiquidityByRange2]":           ix("remove_liquidity_by_range2"),
		"METEORA_DLMM.REMOVE_LIQUIDITY[removeAllLiquidity]":                ix("remove_all_liquidity"),
		"METEORA_DLMM.REMOVE_LIQUIDITY[claimFee]":                          ix("claim_fee"),
		"METEORA_DLMM.REMOVE_LIQUIDITY[claimFeeV2]":                        ix("claim_fee2"),
		"METEORA_DLMM.REMOVE_LIQUIDITY[removeLiquidity2]":                  ix("remove_liquidity2"),
		"METEORA_DLMM.OTHER[claimReward]":                                  ix("claim_reward"),
		"METEORA_DLMM.OTHER[claimReward2]":                                 ix("claim_reward2"),
		"METEORA_DLMM.LIQUIDITY_EVENT[compositionFeeEvent]":                cpiEv("CompositionFee"),
		"METEORA_DLMM.LIQUIDITY_EVENT[addLiquidityEvent]":                  cpiEv("AddLiquidity"),
		"METEORA_DLMM.LIQUIDITY_EVENT[removeLiquidityEvent]":               cpiEv("RemoveLiquidity"),
		"METEORA_DLMM.SWAP[swap]":                                          ix("swap"),
		"METEORA_DLMM.SWAP[swap2]":                                         ix("swap2"),
		"METEORA_DLMM.SWAP[swapExactOut]":                                  ix("swap_exact_out"),
		"METEORA_DLMM.SWAP[swapExactOut2]":                                 ix("swap_exact_out2"),
		"METEORA_DLMM.SWAP[swapWithPriceImpact]":                           ix("swap_with_price_impact"),
		"METEORA_DLMM.SWAP[swapWithPriceImpact2]":                          ix("swap_with_price_impact2"),
		"METEORA_DLMM.CREATE[initializeLbPair]":                            ix("initialize_lb_pair"),
		"METEORA_DLMM.CREATE[initializeLbPair2]":                           ix("initialize_lb_pair2"),
		"METEORA_DLMM.CREATE[initializePermissionLbPair]":                  ix("initialize_permission_lb_pair"),
		"METEORA_DLMM.CREATE[initializeCustomizablePermissionlessLbPair]":  ix("initialize_customizable_permissionless_lb_pair"),
		"METEORA_DLMM.CREATE[initializeCustomizablePermissionlessLbPair2]": ix("initialize_customizable_permissionless_lb_pair2"),
		"METEORA_DLMM.LIMIT_ORDER[placeLimitOrder]":                        ix("place_limit_order"),
		"METEORA_DLMM.LIMIT_ORDER[cancelLimitOrder]":                       ix("cancel_limit_order"),
		"METEORA_DLMM.LIMIT_ORDER[closeLimitOrderIfEmpty]":                 ix("close_limit_order_if_empty"),
		"METEORA_DLMM.OTHER[rebalanceLiquidity]":                           ix("rebalance_liquidity"),
		"METEORA_DLMM.OTHER[fundReward]":                                   ix("fund_reward"),
		"METEORA_DLMM.OTHER[withdrawIneligibleReward]":                     ix("withdraw_ineligible_reward"),
		"METEORA_DLMM.OTHER[withdrawProtocolFee]":                          ix("withdraw_protocol_fee"),
		"METEORA_DLMM.OTHER[zapProtocolFee]":                               ix("zap_protocol_fee"),
		// Meteora DAMM v1 (amm 0.5.2, camelCase IDL names)
		"METEORA_DAMM.CREATE":                                   ix("initialize_permissionless_constant_product_pool_with_config"),
		"METEORA_DAMM.ADD_LIQUIDITY":                            ix("add_balance_liquidity"),
		"METEORA_DAMM.REMOVE_LIQUIDITY":                         ix("remove_balance_liquidity"),
		"METEORA_DAMM.ADD_IMBALANCE_LIQUIDITY":                  ix("add_imbalance_liquidity"),
		"METEORA_DAMM.SWAP":                                     ix("swap"),
		"METEORA_DAMM.CREATE_PERMISSIONED_POOL":                 ix("initialize_permissioned_pool"),
		"METEORA_DAMM.CREATE_PERMISSIONLESS_POOL":               ix("initialize_permissionless_pool"),
		"METEORA_DAMM.CREATE_PERMISSIONLESS_POOL_WITH_FEE_TIER": ix("initialize_permissionless_pool_with_fee_tier"),
		"METEORA_DAMM.CREATE_WITH_CONFIG2":                      ix("initialize_permissionless_constant_product_pool_with_config2"),
		"METEORA_DAMM.CREATE_CUSTOMIZABLE":                      ix("initialize_customizable_permissionless_constant_product_pool"),
		"METEORA_DAMM.REMOVE_LIQUIDITY_SINGLE_SIDE":             ix("remove_liquidity_single_side"),
		"METEORA_DAMM.BOOTSTRAP_LIQUIDITY":                      ix("bootstrap_liquidity"),
		"METEORA_DAMM.CLAIM_FEE":                                ix("claim_fee"),
		"METEORA_DAMM.PARTNER_CLAIM_FEE":                        ix("partner_claim_fee"),
		"METEORA_DAMM.WITHDRAW_PROTOCOL_FEES":                   ix("withdraw_protocol_fees"),
		"METEORA_DAMM.LOCK":                                     ix("lock"),
		// Meteora DAMM v2 (cp_amm 0.2.0)
		"METEORA_DAMM_V2.INITIALIZE_POOL":                     ix("initialize_pool"),
		"METEORA_DAMM_V2.INITIALIZE_CUSTOM_POOL":              ix("initialize_customizable_pool"),
		"METEORA_DAMM_V2.INITIALIZE_POOL_WITH_DYNAMIC_CONFIG": ix("initialize_pool_with_dynamic_config"),
		"METEORA_DAMM_V2.ADD_LIQUIDITY":                       ix("add_liquidity"),
		"METEORA_DAMM_V2.CLAIM_POSITION_FEE":                  ix("claim_position_fee"),
		"METEORA_DAMM_V2.REMOVE_LIQUIDITY":                    ix("remove_liquidity"),
		"METEORA_DAMM_V2.REMOVE_ALL_LIQUIDITY":                ix("remove_all_liquidity"),
		"METEORA_DAMM_V2.CREATE_POSITION_EVENT":               cpiEv("EvtCreatePosition"),
		"METEORA_DAMM_V2.SWAP":                                ix("swap"),
		"METEORA_DAMM_V2.SWAP2":                               ix("swap2"),
		"METEORA_DAMM_V2.CLAIM_REWARD":                        ix("claim_reward"),
		"METEORA_DAMM_V2.SPLIT_POSITION":                      ix("split_position"),
		"METEORA_DAMM_V2.CLAIM_PROTOCOL_FEE":                  ix("claim_protocol_fee"),
		"METEORA_DAMM_V2.FUND_REWARD":                         ix("fund_reward"),
		"METEORA_DAMM_V2.WITHDRAW_INELIGIBLE_REWARD":          ix("withdraw_ineligible_reward"),
		"METEORA_DAMM_V2.ZAP_PROTOCOL_FEE":                    ix("zap_protocol_fee"),
		"METEORA_DAMM_V2.SPLIT_POSITION2":                     ix("split_position2"),
		// Meteora DBC (dynamic_bonding_curve 0.2.1)
		"METEORA_DBC.SWAP":                                                 ix("swap"),
		"METEORA_DBC.SWAP_V2":                                              ix("swap2"),
		"METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_SPL":                     ix("initialize_virtual_pool_with_spl_token"),
		"METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022":               ix("initialize_virtual_pool_with_token2022"),
		"METEORA_DBC.METEORA_DBC_MIGRATE_DAMM":                             ix("migrate_meteora_damm"),
		"METEORA_DBC.METEORA_DBC_MIGRATE_DAMM_V2":                          ix("migration_damm_v2"),
		"METEORA_DBC.SWAP2_WITH_TRANSFER_HOOK":                             ix("swap2_with_transfer_hook"),
		"METEORA_DBC.INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022_TRANSFER_HOOK": ix("initialize_virtual_pool_with_token2022_transfer_hook"),
		"METEORA_DBC.EVT_SWAP":                                             cpiEv("EvtSwap"),
		"METEORA_DBC.EVT_SWAP2":                                            cpiEv("EvtSwap2"),
		"METEORA_DBC.EVT_SWAP2_WITH_TRANSFER_HOOK":                         cpiEv("EvtSwap2WithTransferHook"),
		"METEORA_DBC.EVT_CURVE_COMPLETE":                                   cpiEv("EvtCurveComplete"),
		"METEORA_DBC.EVT_CURVE_COMPLETE_WITH_TRANSFER_HOOK":                cpiEv("EvtCurveCompleteWithTransferHook"),
		"METEORA_DBC.EVT_INITIALIZE_POOL":                                  cpiEv("EvtInitializePool"),
		"METEORA_DBC.EVT_INITIALIZE_POOL_WITH_TRANSFER_HOOK":               cpiEv("EvtInitializePoolWithTransferHook"),
		// Orca Whirlpool 0.9.0
		"ORCA.CREATE":                            ix("open_position_with_metadata"),
		"ORCA.CREATE2":                           ix("open_position_with_token_extensions"),
		"ORCA.ADD_LIQUIDITY":                     ix("increase_liquidity"),
		"ORCA.ADD_LIQUIDITY2":                    ix("increase_liquidity_v2"),
		"ORCA.REMOVE_LIQUIDITY":                  ix("decrease_liquidity"),
		"ORCA.OTHER1":                            ix("collect_fees"),
		"ORCA.OTHER2":                            ix("collect_reward"),
		"ORCA.SWAP":                              ix("swap"),
		"ORCA.SWAP_V2":                           ix("swap_v2"),
		"ORCA.TWO_HOP_SWAP":                      ix("two_hop_swap"),
		"ORCA.TWO_HOP_SWAP_V2":                   ix("two_hop_swap_v2"),
		"ORCA.REMOVE_LIQUIDITY_V2":               ix("decrease_liquidity_v2"),
		"ORCA.ADD_LIQUIDITY_BY_TOKEN_AMOUNTS_V2": ix("increase_liquidity_by_token_amounts_v2"),
		"ORCA.REPOSITION_LIQUIDITY_V2":           ix("reposition_liquidity_v2"),
		"ORCA.COLLECT_FEES_V2":                   ix("collect_fees_v2"),
		"ORCA.COLLECT_REWARD_V2":                 ix("collect_reward_v2"),
		"ORCA.COLLECT_PROTOCOL_FEES":             ix("collect_protocol_fees"),
		"ORCA.COLLECT_PROTOCOL_FEES_V2":          ix("collect_protocol_fees_v2"),
		"ORCA.TRADED_EVENT":                      logEv("Traded"),
		"ORCA.LIQUIDITY_INCREASED_EVENT":         logEv("LiquidityIncreased"),
		"ORCA.LIQUIDITY_DECREASED_EVENT":         logEv("LiquidityDecreased"),
		// Boop.fun 0.3.0
		"BOOPFUN.CREATE":   ix("create_token"),
		"BOOPFUN.DEPLOY":   ix("deploy_bonding_curve"),
		"BOOPFUN.COMPLETE": ix("graduate"),
		"BOOPFUN.BUY":      ix("buy_token"),
		"BOOPFUN.SELL":     ix("sell_token"),
		// Heaven (no public IDL; names resolved by hash)
		"HEAVEN.BUY":         ix("buy"),
		"HEAVEN.SELL":        ix("sell"),
		"HEAVEN.CREATE_POOL": ix("create_standard_liquidity_pool"),
		// Sugar (mastermind IDL)
		"SUGAR.BUY_EXACT_IN":      ix("buy_exact_in"),
		"SUGAR.BUY_EXACT_OUT":     ix("buy_exact_out"),
		"SUGAR.BUY_MAX_OUT":       ix("buy_max_out"),
		"SUGAR.SELL_EXACT_IN":     ix("sell_exact_in"),
		"SUGAR.SELL_EXACT_OUT":    ix("sell_exact_out"),
		"SUGAR.CREATE":            ix("create"),
		"SUGAR.INITIALIZE":        ix("initialize"),
		"SUGAR.MIGRATE_TO_RADIUM": ix("migrate_to_radium"),
		// Photon (no public IDL; names resolved by hash, v2 ones observed on mainnet)
		"PHOTON.PUMPSWAP_TRADE":  ix("pump_amm_swap"),
		"PHOTON.PUMPFUN_BUY":     ix("pump_buy"),
		"PHOTON.PUMPFUN_SELL":    ix("pump_sell"),
		"PHOTON.MOONIT_BUY":      ix("moonshot_buy"),
		"PHOTON.MOONIT_SELL":     ix("moonshot_sell"),
		"PHOTON.HOP_TWO_SWAP":    ix("two_hop_swap"),
		"PHOTON.PUMPFUN_BUY_V2":  ix("pump_buy_v2"),
		"PHOTON.PUMPFUN_SELL_V2": ix("pump_sell_v2"),
		"PHOTON.COLLECT_FEE":     ix("collect_fee"),
		// Obric V2
		"OBRIC.SWAP": ix("swap"),
		// DFlow (swap_orchestrator IDL)
		"DFLOW.SWAP":                   ix("swap"),
		"DFLOW.SWAP2":                  ix("swap2"),
		"DFLOW.SWAP_WITH_DEST":         ix("swap_with_destination"),
		"DFLOW.OPEN_ORDER":             ix("open_order"),
		"DFLOW.FILL_ORDER":             ix("fill_order"),
		"DFLOW.CLOSE_ORDER":            ix("close_order"),
		"DFLOW.SWAP2_WITH_DEST":        ix("swap2_with_destination"),
		"DFLOW.SWAP_WITH_DEST_NATIVE":  ix("swap_with_destination_native"),
		"DFLOW.SWAP2_WITH_DEST_NATIVE": ix("swap2_with_destination_native"),
		// HumidiFi: Anchor "swap" hash, unused by the (non-Anchor) parser
		"HUMIDIFI.SWAP": ix("swap"),
		// Venues (2026-09): Byreal on-chain IDL, Saros liquidity_book IDL, OKX
		// "OKX: DEX Router" on-chain IDL, Obric swap2 seen in 33VnDBtr...
		"BYREAL.SWAP":                                   ix("swap"),
		"BYREAL.SWAP_V2":                                ix("swap_v2"),
		"BYREAL.SWAP_V3_DYN":                            ix("swap_v3_dyn"),
		"SAROS_DLMM.SWAP":                               ix("swap"),
		"OBRIC.SWAP2":                                   ix("swap2"),
		"OKX_DEX_V2.SWAP_CPI_EVENT2":                    cpiEv("SwapCpiEvent2"),
		"OKX_DEX_V2.SWAP_WITH_FEES_CPI_EVENT2":          cpiEv("SwapWithFeesCpiEvent2"),
		"OKX_DEX_V2.SWAP_WITH_FEES_CPI_EVENT_ENHANCED2": cpiEv("SwapWithFeesCpiEventEnhanced2"),
		"OKX_DEX_V2.SWAP_TOB_V2_CPI_EVENT2":             cpiEv("SwapTobV2CpiEvent2"),
		"OKX_DEX_V2.SWAP_TOC_V2_CPI_EVENT2":             cpiEv("SwapTocV2CpiEvent2"),
		"OKX_DEX_V2.SWAP_WITH_FEE_CPI_EVENT_V3":         cpiEv("SwapWithFeeCpiEventV3"),
	}
}

// constNonAnchorDiscriminators lists discriminators that are not Anchor hashes.
var constNonAnchorDiscriminators = map[string]string{
	"RAYDIUM.CREATE":            "Raydium V4 instruction tag",
	"RAYDIUM.ADD_LIQUIDITY":     "Raydium V4 instruction tag",
	"RAYDIUM.REMOVE_LIQUIDITY":  "Raydium V4 instruction tag",
	"RAYDIUM.SWAP":              "Raydium V4 instruction tag",
	"RAYDIUM.SWAP_EXACT_OUT":    "Raydium V4 instruction tag",
	"RAYDIUM.SWAP_V2":           "Raydium V4 instruction tag",
	"RAYDIUM.SWAP_EXACT_OUT_V2": "Raydium V4 instruction tag",
	"METAPLEX.CREATE_MINT":      "Metaplex instruction tag",
	"SOLFI.SWAP":                "native program tag",
	"GOONFI.SWAP":               "native program tag",
	"OBRIC.SWAP_X_TO_Y":         "unverified legacy value (equals Raydium CPMM swap_base_input)",
	"OBRIC.SWAP_Y_TO_X":         "unverified legacy value",
	"SOLFI_V2.SWAP":             "native program tag",
	"GOONFI_V2.SWAP":            "native program tag",
	"BISONFI.SWAP":              "native program tag",
	"BISONFI.SWAP_V2":           "native program tag",
	"BISONFI.SWAP_WITH_SIG":     "native program tag",
	"TESSERA_V.SWAP":            "native program tag",
	"ALPHAQ.SWAP":               "native program tag",
	"ZERO_FI.SWAP":              "native program tag",
	"SCORCH.SWAP":               "native program tag",
	"SCORCH.SWAP_COMPACT":       "native program tag",
	"QUANTUM.SWAP":              "native program tag",
	"MANIFEST.SWAP":             "Manifest (shank) instruction tag",
	"MANIFEST.SWAP_V2":          "Manifest (shank) instruction tag",
	"TITAN.SWAP_ROUTE_V3":       "native program tag",
	"TITAN.SWAP_EVENT":          "Titan log event tag (no IDL)",
}

// walkDiscriminators returns every []byte in constants.DISCRIMINATORS keyed by field path
// ("PROGRAM.FIELD", "PROGRAM.GROUP.FIELD" or "PROGRAM.MAP[key]").
func walkDiscriminators() map[string][]byte {
	out := map[string][]byte{}
	var walk func(path string, v reflect.Value)
	walk = func(path string, v reflect.Value) {
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				name := v.Type().Field(i).Name
				if path != "" {
					name = path + "." + name
				}
				walk(name, v.Field(i))
			}
		case reflect.Map:
			for _, k := range v.MapKeys() {
				walk(path+"["+k.String()+"]", v.MapIndex(k))
			}
		case reflect.Slice:
			out[path] = v.Bytes()
		}
	}
	walk("", reflect.ValueOf(constants.DISCRIMINATORS))
	return out
}

// Every Anchor discriminator equals the hash of its IDL name (constants-6, meme-18,
// constants-17, and all discriminators added for the 2026-09 audit).
func TestConstantsAnchorDiscriminators(t *testing.T) {
	table := constAnchorTable()
	all := walkDiscriminators()

	var paths []string
	for p := range all {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		got := all[p]
		if len(got) == 0 {
			t.Errorf("%s is empty", p)
			continue
		}
		want, ok := table[p]
		if !ok {
			if _, skip := constNonAnchorDiscriminators[p]; !skip {
				t.Errorf("%s (% x) has no IDL name in the test table", p, got)
			}
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s = %v (len %d), want %v", p, got, len(got), want)
		}
	}
	for p := range table {
		if _, ok := all[p]; !ok {
			t.Errorf("test table entry %s has no constant", p)
		}
	}
}

// constants-17: empty discriminators must never match; before the fix
// MatchDiscriminator(x, []byte{}) returned true for any x.
func TestConstantsEmptyDiscriminatorNeverMatches(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if constants.MatchDiscriminator(data, []byte{}) || constants.MatchDiscriminator(data, nil) {
		t.Error("MatchDiscriminator matches an empty discriminator")
	}
	if name, ok := constants.MatchAnyDiscriminator(data, map[string][]byte{"empty": {}}); ok {
		t.Errorf("MatchAnyDiscriminator matched empty discriminator %q", name)
	}
	if !constants.MatchDiscriminator(data, []byte{1, 2}) {
		t.Error("MatchDiscriminator does not match a real prefix")
	}
	if constants.MatchDiscriminator(data, constants.DISCRIMINATORS.JUPITER_VA.CLOSE_EVENT) {
		t.Error("JUPITER_VA.CLOSE_EVENT matches arbitrary data")
	}
}

// constants-6 / meme-18: a real sell_exact_out instruction must match SUGAR.SELL_EXACT_OUT
// (it was 9 bytes with a stray leading 149 and could never match).
func TestConstantsSugarSellExactOut(t *testing.T) {
	data := append(constIxDisc("sell_exact_out"), make([]byte, 24)...)
	if !constants.MatchDiscriminator(data, constants.DISCRIMINATORS.SUGAR.SELL_EXACT_OUT) {
		t.Errorf("sell_exact_out data does not match SUGAR.SELL_EXACT_OUT (%d bytes)", len(constants.DISCRIMINATORS.SUGAR.SELL_EXACT_OUT))
	}
	if len(constants.DISCRIMINATORS.SUGAR.SELL_EXACT_OUT) != 8 {
		t.Errorf("SUGAR.SELL_EXACT_OUT has %d bytes, want 8", len(constants.DISCRIMINATORS.SUGAR.SELL_EXACT_OUT))
	}
}

// constFixtureInstructionData returns the data of the instruction at idx ("N" outer,
// "N-M" inner) and its program ID.
func constFixtureInstructionData(t *testing.T, tx *adapter.SolanaTransaction, idx string) (string, []byte) {
	t.Helper()
	a := adapter.NewTransactionAdapter(tx, nil)
	var outer, inner int
	if strings.Contains(idx, "-") {
		if _, err := fmt.Sscanf(idx, "%d-%d", &outer, &inner); err != nil {
			t.Fatalf("bad idx %s", idx)
		}
		for _, set := range a.InnerInstructions() {
			if set.Index == outer && inner < len(set.Instructions) {
				ix := set.Instructions[inner]
				return a.GetInstructionProgramId(ix), a.GetInstructionData(ix)
			}
		}
		t.Fatalf("inner instruction %s not found", idx)
	}
	if _, err := fmt.Sscanf(idx, "%d", &outer); err != nil || outer >= len(a.Instructions()) {
		t.Fatalf("bad idx %s", idx)
	}
	ix := a.Instructions()[outer]
	return a.GetInstructionProgramId(ix), a.GetInstructionData(ix)
}

// New and fixed discriminators match the instruction data of real mainnet transactions
// (shred-9, shred-17, constants-1, constants-4, constants-10, parity-14, parity-15,
// constants-2, constants-7, amm-7 program constant).
func TestConstantsDiscriminatorsMatchMainnetFixtures(t *testing.T) {
	d := constants.DISCRIMINATORS
	p := constants.DEX_PROGRAMS
	cases := []struct {
		name, sig, idx, program string
		want                    []byte
	}{
		{"Jupiter route_v2", "3Nb6n7meLAN6SB2EgbFbRYxvLGnj42TbQh2eA7Awr5h8euTt1icygUfAnd1pc19nAUA1513BPZdaFNNW6LAEouNV", "2", p.JUPITER.ID, d.JUPITER.ROUTE_V2},
		{"Jupiter shared_accounts_route_v2", "1V8cnQVrAApjfY61PdnG1mCoj9ejqNDj5xKyEnNWUrzQtA1VN129zCVYmkUqTkuiTQzrmWY9FWQenQaBrXDWEXS", "2", p.JUPITER.ID, d.JUPITER.SHARED_ACCOUNTS_ROUTE_V2},
		{"Jupiter SwapsEvent", "1V8cnQVrAApjfY61PdnG1mCoj9ejqNDj5xKyEnNWUrzQtA1VN129zCVYmkUqTkuiTQzrmWY9FWQenQaBrXDWEXS", "2-18", p.JUPITER.ID, d.JUPITER.SWAPS_EVENT},
		{"Jupiter FeeEvent", "3TZKJLxy4H2wQiYenuSVoQ2ox7xveRoG5bxr7yfmEdtMPqKmdHWcf5Q9B8uUBi6ystp2gQsZdP5qxiYK4JnUpm7", "5-1", p.JUPITER.ID, d.JUPITER.FEE_EVENT},
		{"JupiterZ fill", "4tke4p2A2RqXWW4RgMMp3wELrcjn4gZyhUbS21QqJmxVNhEFhip6a7VcQCc5eEN1dztTSfd7FZvYuPT9bhrYEe1w", "3", p.JUPITER_Z.ID, d.JUPITER_Z.FILL},
		{"Pump.fun create_v2", "5zMo16ECF3UwYWzj8kG7ccYavPjs3oMMJGkcDuvgiyUsqHxFG1suvzyF6T4CoWrgbWryiiFw4oSitkdmUsa8mZae", "2", p.PUMP_FUN.ID, d.PUMPFUN.CREATE_V2},
		{"Pump.fun buy_v2", "5zMo16ECF3UwYWzj8kG7ccYavPjs3oMMJGkcDuvgiyUsqHxFG1suvzyF6T4CoWrgbWryiiFw4oSitkdmUsa8mZae", "4", p.PUMP_FUN.ID, d.PUMPFUN.BUY_V2},
		{"Pump.fun sell_v2", "2DdZiRsXJWVkD1nD31bcBQpZPKmmmd1gG1T4GAFcLDw6f7Zpxao876E6pPcreUqMmKLWjmxo5FRvq6WynU2pPXaW", "3-0", p.PUMP_FUN.ID, d.PUMPFUN.SELL_V2},
		{"Pump.fun buy_exact_quote_in_v2", "4teKKBqKNVBfAmsPzbHRSgV1FtNWKtXFe45Ps6fJ7uibAGtCwnnRBEM852VTKfG4AfsdgacbS6a5t5SqgnKDUNvS", "6-1", p.PUMP_FUN.ID, d.PUMPFUN.BUY_EXACT_QUOTE_IN_V2},
		{"Pump.fun migrate_v2", "3ck1kPo6VzPsahWM1wpotwCfFR3yW5B8ATwDPptfVEVQCjmXRWtm44XiAgaojJHwgnEyckrX2YHkFCEz3MMVUXwM", "1", p.PUMP_FUN.ID, d.PUMPFUN.MIGRATE_V2},
		{"PumpSwap buy_exact_quote_in", "28DurhrCQrkkrM6MeJki2yQr76Hfx2pUMRZnzzjbfXJSW17PaZVbBMs7LpxdNt56FbxtJuSgyHJVbDxRYYtFKYJ1", "4-0", p.PUMP_SWAP.ID, d.PUMPSWAP.BUY_EXACT_QUOTE_IN},
		{"Photon pump_buy_v2", "61tXHLrQufhFiHbjq8z4Ugm1TDY8ig65MfESi7sQCFDRN18gszH8crcNbaMHVxvEHrtY8o5xMddJ58X9gvp7GcQA", "2", p.PHOTON.ID, d.PHOTON.PUMPFUN_BUY_V2},
		{"Photon pump_sell_v2", "5p8FZiX6Q5C9WuTwQaH72UJuUShRJnBs6LGziQFUrkzrE2SyDHtXybp8RtXwYWRHKwvXt9cCMLCz2tqATwAi2vSJ", "2", p.PHOTON.ID, d.PHOTON.PUMPFUN_SELL_V2},
		{"Photon collect_fee", "afUCiFQ6amxuxx2AAwsghLt7Q9GYqHfZiF4u3AHhAzs8p1ThzmrtSUFMbcdJy8UnQNTa35Fb1YqxR6F9JMZynYp", "5", p.PHOTON.ID, d.PHOTON.COLLECT_FEE},
		{"Orca collect_fees_v2", "5AzH3HApZUEnGECG5Xk26jgUpbiRAzRsAqTRHiTj7Jf6bX3jfSXZCj5zqREvgnYwzgD2mUXw9g6FrN2hVtJZF5JN", "6", p.ORCA.ID, d.ORCA.COLLECT_FEES_V2},
		{"Raydium V4 SwapBaseInV2 (tag 16)", "5NXJqrtPPaAzcTYMSqM2jPjQtnpELWzoctDQNRwqCfyMkxz9Qeko87LsQJGkgamtacbseCY6Rf6w4rfxvpTkk4td", "2-0", p.RAYDIUM_V4.ID, d.RAYDIUM.SWAP_V2},
		{"Meteora DLMM RemoveLiquidity event", "7YPF21r7JBDeoXuMJn6KSqDVYGrm821U87Cnje3xPvZpMUVaAEAvCGJPP6va2b5oMLAzGku5s3TcNAsN6zdXPRn", "10-2", p.METEORA.ID, d.METEORA_DLMM.LIQUIDITY_EVENT["removeLiquidityEvent"]},
		{"Meteora DBC EvtSwap", "128TrBNx8icLgpsCY1Tkk6WXv4QwyJDEBTGuqfRQaT7YG3CFdYTuuw3aVCu36Wqxruu3YdE5HuhiciAooKCw9nHJ", "2-8", p.METEORA_DBC.ID, d.METEORA_DBC.EVT_SWAP},
		{"Meteora DBC EvtSwap2", "128TrBNx8icLgpsCY1Tkk6WXv4QwyJDEBTGuqfRQaT7YG3CFdYTuuw3aVCu36Wqxruu3YdE5HuhiciAooKCw9nHJ", "2-9", p.METEORA_DBC.ID, d.METEORA_DBC.EVT_SWAP2},
		{"Meteora DBC EvtCurveComplete", "ZADe3kHFeqgpN96HQKoxBPdsfq5y9hnPdt1Ue523D8biaeCogpZDz7gdkoNtLLUDfWL7bz142vQwvc9uX44zQdo", "1-4", p.METEORA_DBC.ID, d.METEORA_DBC.EVT_CURVE_COMPLETE},
		{"Meteora DBC EvtInitializePool", "3uwWqXt9wgrkfp1bwLJjMApxLW9TYxriEPtan4aWouH7ExebphFD3LthPmzXmP6eeZ5nKdtrUZodpAZiWR5P8DNV", "0-14", p.METEORA_DBC.ID, d.METEORA_DBC.EVT_INITIALIZE_POOL},
		{"Obric V2 swap (Jupiter CPI)", "2DaS55TwMuryadgsirCLRJywZ7rbBoLugXnicbJR2YFSSgn8PAvGN8Mhr9p4Ux8SEtgPzrRRuqV9T7XS7WWtdxvR", "3-1", p.OBRIC_V2.ID, d.OBRIC.SWAP},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tx := loadFixture(t, c.sig)
			prog, data := constFixtureInstructionData(t, tx, c.idx)
			if prog != c.program {
				t.Fatalf("instruction %s is program %s, want %s", c.idx, prog, c.program)
			}
			if len(c.want) == 0 || !bytes.HasPrefix(data, c.want) {
				n := len(c.want)
				if n == 0 || n > len(data) {
					n = len(data)
				}
				t.Errorf("instruction %s starts with %v, constant is %v", c.idx, data[:n], c.want)
			}
		})
	}
}

// Whirlpool events are emitted as "Program data:" log lines with 8-byte discriminators.
func TestConstantsOrcaTradedEventInLogs(t *testing.T) {
	tx := loadFixture(t, "5JfxWaTCJcLu8H8WcFcxxLSLyHDfdP2A2VUTKGsi3VHkBHbEtUQdu26PzsYZGD8UvgW2FsfNXAaMNx4Mmnv4vQDv")
	found := false
	for _, l := range tx.Meta.LogMessages {
		if !strings.HasPrefix(l, "Program data: ") {
			continue
		}
		b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(l, "Program data: "))
		if err == nil && bytes.HasPrefix(b, constants.DISCRIMINATORS.ORCA.TRADED_EVENT) {
			found = true
		}
	}
	if !found {
		t.Error("no Traded event log matches ORCA.TRADED_EVENT")
	}
}

// constants-18: SPL Token / Token-2022 and System instruction tags (token-2022 interface
// instruction.rs, solana system-interface instruction.rs, associated-token-account interface).
func TestConstantsInstructionTypes(t *testing.T) {
	cases := map[string][2]int{
		"SyncNative":                   {constants.SPLTokenSyncNative, 17},
		"InitializeAccount3":           {constants.SPLTokenInitializeAccount3, 18},
		"InitializeMint2":              {constants.SPLTokenInitializeMint2, 20},
		"InitializeImmutableOwner":     {constants.SPLTokenInitializeImmutableOwner, 22},
		"TransferFeeExtension":         {constants.SPLTokenTransferFeeExtension, 26},
		"TransferCheckedWithFee":       {constants.TransferFeeTransferCheckedWithFee, 1},
		"CreateAccountAllowPrefund":    {constants.SystemCreateAccountAllowPrefund, 13},
		"AssociatedTokenCreateIdempot": {constants.AssociatedTokenCreateIdempotent, 1},
	}
	for name, c := range cases {
		if c[0] != c[1] {
			t.Errorf("%s = %d, want %d", name, c[0], c[1])
		}
	}
}

// constSyntheticDLMMClaimReward2 is a SYNTHETIC transaction: no real claim_reward2 fixture
// exists, so it is built from the real DLMM transaction h3sGiri... (outer instruction 2 is
// a router CPI into lb_clmm). The router's inner lb_clmm call is replaced by a claim_reward2
// instruction with the on-chain lb_clmm IDL account layout [lb_pair, position, sender,
// reward_vault, reward_mint, user_token_account, token_program, memo_program,
// event_authority, program] and one reward transferChecked from the reward vault to the
// user. The later outer instructions are dropped.
func constSyntheticDLMMClaimReward2(t *testing.T) *adapter.SolanaTransaction {
	t.Helper()
	raw, err := readFixture("h3sGiriW4jCGgbnNF8DaEKnsWhjtcH5ZM1dkyZFidgD2fx9aDNatray38yRmkxaWez3g5qFNpyXhE8ho716vjgp", "json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	var m map[string]interface{}
	if err := gojson.Unmarshal(raw, &m); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	msg := m["transaction"].(map[string]interface{})["message"].(map[string]interface{})
	msg["instructions"] = msg["instructions"].([]interface{})[:3]
	meta := m["meta"].(map[string]interface{})
	inner := meta["innerInstructions"].([]interface{})[0].(map[string]interface{})
	if idx := inner["index"].(float64); idx != 2 {
		t.Fatalf("fixture changed: first inner group index %v", idx)
	}
	ixs := inner["instructions"].([]interface{})
	// Account indexes: 16 lb_pair 6t5fJw..., 2 position, 12 sender, 17 reward vault,
	// 22 reward mint, 3 user token account, 20 token program (also as memo), 14 event
	// authority, 15 lb_clmm.
	data := append(constIxDisc("claim_reward2"), make([]byte, 12)...) // reward_index u64, min/max bin i32 unused here
	transfer := []byte{12, 1, 0, 0, 0, 0, 0, 0, 0, 6}                 // transferChecked 1 raw unit, 6 decimals
	inner["instructions"] = []interface{}{
		ixs[0],
		map[string]interface{}{"accounts": []interface{}{16, 2, 12, 17, 22, 3, 20, 20, 14, 15}, "data": base58.Encode(data), "programIdIndex": 15, "stackHeight": 3},
		map[string]interface{}{"accounts": []interface{}{17, 22, 3, 16}, "data": base58.Encode(transfer), "programIdIndex": 20, "stackHeight": 4},
	}
	meta["innerInstructions"] = []interface{}{inner}
	out, err := gojson.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var tx adapter.SolanaTransaction
	if err := gojson.Unmarshal(out, &tx); err != nil {
		t.Fatal(err)
	}
	return &tx
}

// R2-G1: claim_reward and claim_reward2 pay one reward token and have no token_x/token_y
// mint accounts (lb_clmm IDL), so they are not REMOVE_LIQUIDITY. While they were in that
// map, ParseRemoveLiquidityEvent reported the position as the pool and the event authority
// as token1.
func TestConstantsDLMMClaimRewardIsNotLiquidity(t *testing.T) {
	d := constants.DISCRIMINATORS.METEORA_DLMM
	for _, name := range []string{"claim_reward", "claim_reward2"} {
		disc := constIxDisc(name)
		for mapName, m := range map[string]map[string][]byte{"ADD_LIQUIDITY": d.ADD_LIQUIDITY, "REMOVE_LIQUIDITY": d.REMOVE_LIQUIDITY, "SWAP": d.SWAP} {
			for k, v := range m {
				if bytes.Equal(v, disc) {
					t.Errorf("%s is in METEORA_DLMM.%s[%s]", name, mapName, k)
				}
			}
		}
	}

	res := dexparser.NewDexParser().ParseAll(constSyntheticDLMMClaimReward2(t), nil)
	for _, l := range res.Liquidities {
		t.Errorf("claim_reward2 produced a liquidity event: %s pool=%s t0=%s t1=%s idx=%s", l.Type, l.PoolId, l.Token0Mint, l.Token1Mint, l.Idx)
	}
	for _, tr := range res.Trades {
		t.Errorf("claim_reward2 produced a trade: %s %s idx=%s", tr.Type, tr.ProgramId, tr.Idx)
	}
}
