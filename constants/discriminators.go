package constants

// Discriminator byte slices for instruction identification
var DISCRIMINATORS = struct {
	JUPITER                JupiterDiscriminators
	JUPITER_DCA            JupiterDCADiscriminators
	JUPITER_LIMIT_ORDER    JupiterLimitOrderDiscriminators
	JUPITER_LIMIT_ORDER_V2 JupiterLimitOrderV2Discriminators
	JUPITER_VA             JupiterVADiscriminators
	PUMPFUN                PumpfunDiscriminators
	PUMPSWAP               PumpswapDiscriminators
	MOONIT                 MoonitDiscriminators
	RAYDIUM                RaydiumDiscriminators
	RAYDIUM_CL             RaydiumCLDiscriminators
	RAYDIUM_CPMM           RaydiumCPMMDiscriminators
	RAYDIUM_LCP            RaydiumLCPDiscriminators
	METEORA_DLMM           MeteoraDLMMDiscriminators
	METEORA_DAMM           MeteoraDAMMDiscriminators
	METEORA_DAMM_V2        MeteoraDAMMV2Discriminators
	METEORA_DBC            MeteoraDBCDiscriminators
	ORCA                   OrcaDiscriminators
	BOOPFUN                BoopfunDiscriminators
	HEAVEN                 HeavenDiscriminators
	METAPLEX               MetaplexDiscriminators
	SUGAR                  SugarDiscriminators
	PHOTON                 PhotonDiscriminators
	SOLFI                  SolFiDiscriminators
	GOONFI                 GoonFiDiscriminators
	OBRIC                  ObricDiscriminators
	DFLOW                  DFlowDiscriminators
	HUMIDIFI               HumidiFiDiscriminators

	// Added 2026-09 (audit): Jupiter Z (order_engine RFQ)
	JUPITER_Z JupiterZDiscriminators
}{
	JUPITER: JupiterDiscriminators{
		ROUTE_EVENT: []byte{228, 69, 165, 46, 81, 203, 154, 29, 64, 198, 205, 232, 38, 8, 113, 226},
		// Jupiter V6 shred discriminators
		ROUTE:                                  []byte{229, 23, 203, 151, 122, 227, 173, 42},
		ROUTE_EXACT_OUT:                        []byte{208, 51, 239, 151, 123, 43, 237, 92},
		ROUTE_WITH_TOKEN_LEDGER:                []byte{150, 86, 71, 116, 167, 93, 14, 104},
		SHARE_ACCOUNTS_ROUTE:                   []byte{193, 32, 155, 51, 65, 214, 156, 129},
		SHARE_ACCOUNTS_EXACT_OUT_ROUTE:         []byte{176, 209, 105, 168, 154, 125, 69, 62},
		SHARE_ACCOUNTS_ROUTE_WITH_TOKEN_LEDGER: []byte{230, 121, 143, 80, 119, 159, 106, 170},

		// Jupiter V6 *_v2 routes (on-chain IDL). They emit SwapsEvent instead of SwapEvent.
		ROUTE_V2:                           []byte{187, 100, 250, 204, 49, 196, 175, 20},
		SHARED_ACCOUNTS_ROUTE_V2:           []byte{209, 152, 83, 147, 124, 254, 216, 233},
		EXACT_OUT_ROUTE_V2:                 []byte{157, 138, 184, 82, 21, 244, 243, 36},
		SHARED_ACCOUNTS_EXACT_OUT_ROUTE_V2: []byte{53, 96, 229, 202, 216, 187, 250, 24},
		// Self-CPI events (anchor event prefix + event:SwapsEvent / event:FeeEvent)
		SWAPS_EVENT: []byte{228, 69, 165, 46, 81, 203, 154, 29, 152, 47, 78, 235, 192, 96, 110, 106},
		FEE_EVENT:   []byte{228, 69, 165, 46, 81, 203, 154, 29, 73, 79, 78, 127, 184, 213, 13, 220},
	},
	JUPITER_DCA: JupiterDCADiscriminators{
		FILLED:      []byte{228, 69, 165, 46, 81, 203, 154, 29, 134, 4, 17, 63, 221, 45, 177, 173},
		CLOSE_DCA:   []byte{22, 7, 33, 98, 168, 183, 34, 243},
		OPEN_DCA:    []byte{36, 65, 185, 54, 1, 210, 100, 163},
		OPEN_DCA_V2: []byte{142, 119, 43, 109, 162, 52, 11, 177},
	},
	JUPITER_LIMIT_ORDER: JupiterLimitOrderDiscriminators{
		CANCEL_ORDER:     []byte{95, 129, 237, 240, 8, 49, 223, 132},
		CREATE_ORDER:     []byte{133, 110, 74, 175, 112, 159, 245, 159},
		TRADE_EVENT:      []byte{228, 69, 165, 46, 81, 203, 154, 29, 189, 219, 127, 211, 78, 230, 97, 238},
		UNKNOWN:          []byte{232, 122, 115, 25, 199, 143, 136, 162},
		FLASH_FILL_ORDER: []byte{252, 104, 18, 134, 164, 78, 18, 140},
	},
	JUPITER_LIMIT_ORDER_V2: JupiterLimitOrderV2Discriminators{
		CANCEL_ORDER:       []byte{95, 129, 237, 240, 8, 49, 223, 132},
		CREATE_ORDER_EVENT: []byte{228, 69, 165, 46, 81, 203, 154, 29, 49, 142, 72, 166, 230, 29, 84, 84},
		TRADE_EVENT:        []byte{228, 69, 165, 46, 81, 203, 154, 29, 189, 219, 127, 211, 78, 230, 97, 238},
		UNKNOWN:            []byte{232, 122, 115, 25, 199, 143, 136, 162},
		FLASH_FILL_ORDER:   []byte{252, 104, 18, 134, 164, 78, 18, 140},

		CANCEL_DUST_ORDER:       []byte{197, 112, 189, 164, 79, 48, 23, 246},
		CANCEL_ORDER_EVENT:      []byte{228, 69, 165, 46, 81, 203, 154, 29, 174, 66, 141, 17, 4, 224, 162, 77},
		CANCEL_DUST_ORDER_EVENT: []byte{228, 69, 165, 46, 81, 203, 154, 29, 124, 190, 74, 28, 177, 40, 200, 220},
	},
	JUPITER_VA: JupiterVADiscriminators{
		FILL_EVENT:     []byte{228, 69, 165, 46, 81, 203, 154, 29, 78, 225, 199, 154, 86, 219, 224, 169},
		OPEN_EVENT:     []byte{228, 69, 165, 46, 81, 203, 154, 29, 104, 220, 224, 191, 87, 241, 132, 61},
		CLOSE_EVENT:    []byte{228, 69, 165, 46, 81, 203, 154, 29, 255, 220, 12, 202, 144, 201, 67, 237},
		DEPOSIT_EVENT:  []byte{228, 69, 165, 46, 81, 203, 154, 29, 62, 205, 242, 175, 244, 169, 136, 52},
		WITHDRAW_EVENT: []byte{228, 69, 165, 46, 81, 203, 154, 29, 192, 241, 201, 217, 70, 150, 90, 247},
	},
	PUMPFUN: PumpfunDiscriminators{
		CREATE:         []byte{24, 30, 200, 40, 5, 28, 7, 119},
		MIGRATE:        []byte{155, 234, 231, 146, 236, 158, 162, 30},
		BUY:            []byte{102, 6, 61, 18, 1, 218, 235, 234},
		SELL:           []byte{51, 230, 133, 164, 1, 127, 131, 173},
		TRADE_EVENT:    []byte{228, 69, 165, 46, 81, 203, 154, 29, 189, 219, 127, 211, 78, 230, 97, 238},
		CREATE_EVENT:   []byte{228, 69, 165, 46, 81, 203, 154, 29, 27, 114, 169, 77, 222, 235, 99, 118},
		COMPLETE_EVENT: []byte{228, 69, 165, 46, 81, 203, 154, 29, 95, 114, 97, 156, 212, 46, 152, 8},
		MIGRATE_EVENT:  []byte{228, 69, 165, 46, 81, 203, 154, 29, 189, 233, 93, 185, 92, 148, 234, 148},

		// Current on-chain IDL instructions (quote-mint aware v2 layouts, exact-SOL buy)
		BUY_V2:                []byte{184, 23, 238, 97, 103, 197, 211, 61},
		SELL_V2:               []byte{93, 246, 130, 60, 231, 233, 64, 178},
		BUY_EXACT_SOL_IN:      []byte{56, 252, 116, 8, 158, 223, 205, 95},
		BUY_EXACT_QUOTE_IN_V2: []byte{194, 171, 28, 70, 104, 77, 91, 47},
		CREATE_V2:             []byte{214, 144, 76, 236, 95, 139, 49, 180},
		MIGRATE_V2:            []byte{187, 203, 18, 31, 206, 237, 254, 41},
	},
	PUMPSWAP: PumpswapDiscriminators{
		CREATE_POOL:            []byte{233, 146, 209, 142, 207, 104, 64, 188},
		ADD_LIQUIDITY:          []byte{242, 35, 198, 137, 82, 225, 242, 182},
		REMOVE_LIQUIDITY:       []byte{183, 18, 70, 156, 148, 109, 161, 34},
		BUY:                    []byte{102, 6, 61, 18, 1, 218, 235, 234},
		SELL:                   []byte{51, 230, 133, 164, 1, 127, 131, 173},
		CREATE_POOL_EVENT:      []byte{228, 69, 165, 46, 81, 203, 154, 29, 177, 49, 12, 210, 160, 118, 167, 116},
		ADD_LIQUIDITY_EVENT:    []byte{228, 69, 165, 46, 81, 203, 154, 29, 120, 248, 61, 83, 31, 142, 107, 144},
		REMOVE_LIQUIDITY_EVENT: []byte{228, 69, 165, 46, 81, 203, 154, 29, 22, 9, 133, 26, 160, 44, 71, 192},
		BUY_EVENT:              []byte{228, 69, 165, 46, 81, 203, 154, 29, 103, 244, 82, 31, 44, 245, 119, 119},
		SELL_EVENT:             []byte{228, 69, 165, 46, 81, 203, 154, 29, 62, 47, 55, 10, 165, 3, 220, 42},

		BUY_EXACT_QUOTE_IN: []byte{198, 46, 21, 82, 180, 217, 232, 112},
	},
	MOONIT: MoonitDiscriminators{
		BUY:     []byte{102, 6, 61, 18, 1, 218, 235, 234},
		SELL:    []byte{51, 230, 133, 164, 1, 127, 131, 173},
		CREATE:  []byte{3, 44, 164, 184, 123, 13, 245, 179},
		MIGRATE: []byte{42, 229, 10, 231, 189, 62, 193, 174},
	},
	RAYDIUM: RaydiumDiscriminators{
		CREATE:           []byte{1},
		ADD_LIQUIDITY:    []byte{3},
		REMOVE_LIQUIDITY: []byte{4},
		SWAP:             []byte{9},
		SWAP_EXACT_OUT:   []byte{11},

		// SwapBaseInV2 / SwapBaseOutV2 (raydium-amm program/src/instruction.rs)
		SWAP_V2:           []byte{16},
		SWAP_EXACT_OUT_V2: []byte{17},
	},
	RAYDIUM_CL: RaydiumCLDiscriminators{
		CREATE: RaydiumCLCreateDiscriminators{
			OPEN_POSITION:    []byte{135, 128, 47, 77, 15, 152, 240, 49},
			OPEN_POSITION_V2: []byte{77, 184, 74, 214, 112, 86, 241, 199},
			CREATE_POOL:      []byte{233, 146, 209, 142, 207, 104, 64, 188},

			CREATE_CUSTOMIZABLE_POOL: []byte{43, 68, 212, 167, 89, 47, 164, 1},
		},
		ADD_LIQUIDITY: RaydiumCLAddLiquidityDiscriminators{
			INCREASE_LIQUIDITY:         []byte{46, 156, 243, 118, 13, 205, 251, 178},
			INCREASE_LIQUIDITY_V2:      []byte{133, 29, 89, 223, 69, 238, 176, 10},
			OPEN_POSITION_WITH_TOKEN22: []byte{77, 255, 174, 82, 125, 29, 201, 46},
		},
		REMOVE_LIQUIDITY: RaydiumCLRemoveLiquidityDiscriminators{
			DECREASE_LIQUIDITY:    []byte{160, 38, 208, 111, 104, 91, 44, 1},
			DECREASE_LIQUIDITY_V2: []byte{58, 127, 188, 62, 79, 82, 196, 96},
			CLOSE_POSITION:        []byte{123, 134, 81, 0, 49, 68, 98, 98},
		},
		SWAP: RaydiumCLSwapDiscriminators{
			SWAP:                []byte{248, 198, 158, 145, 225, 117, 135, 200},
			SWAP_V2:             []byte{43, 4, 237, 11, 26, 201, 30, 98},
			SWAP_ROUTER_BASE_IN: []byte{69, 125, 115, 218, 245, 186, 242, 196},
		},
		EVENTS: RaydiumCLEventDiscriminators{
			COLLECT_PERSONAL_FEE:     []byte{166, 174, 105, 192, 81, 161, 83, 105},
			COLLECT_PROTOCOL_FEE:     []byte{206, 87, 17, 79, 45, 41, 213, 61},
			CONFIG_CHANGE:            []byte{247, 189, 7, 119, 106, 112, 95, 151},
			CREATE_PERSONAL_POSITION: []byte{100, 30, 87, 249, 196, 223, 154, 206},
			DECREASE_LIQUIDITY:       []byte{58, 222, 86, 58, 68, 50, 85, 56},
			INCREASE_LIQUIDITY:       []byte{49, 79, 105, 212, 32, 34, 30, 84},
			LIQUIDITY_CALCULATE:      []byte{237, 112, 148, 230, 57, 84, 180, 162},
			LIQUIDITY_CHANGE:         []byte{126, 240, 175, 206, 158, 88, 153, 107},
			POOL_CREATED:             []byte{25, 94, 75, 47, 112, 99, 53, 63},
			SWAP:                     []byte{64, 198, 205, 232, 38, 8, 113, 226},
			UPDATE_REWARD_INFOS:      []byte{109, 127, 186, 78, 114, 65, 37, 236},

			OPEN_LIMIT_ORDER:     []byte{106, 24, 71, 85, 57, 169, 158, 216},
			INCREASE_LIMIT_ORDER: []byte{11, 120, 13, 204, 199, 87, 19, 200},
			DECREASE_LIMIT_ORDER: []byte{70, 48, 40, 221, 219, 237, 212, 163},
			SETTLE_LIMIT_ORDER:   []byte{88, 119, 77, 164, 125, 124, 10, 194},
		},

		// Limit orders (official raydium_clmm IDL); they move tokens but are not swaps
		LIMIT_ORDER: RaydiumCLLimitOrderDiscriminators{
			OPEN_LIMIT_ORDER:     []byte{157, 32, 218, 183, 71, 29, 18, 147},
			INCREASE_LIMIT_ORDER: []byte{177, 144, 89, 236, 250, 186, 125, 99},
			DECREASE_LIMIT_ORDER: []byte{117, 157, 60, 103, 66, 49, 163, 0},
			SETTLE_LIMIT_ORDER:   []byte{205, 78, 116, 33, 92, 105, 26, 96},
			CLOSE_LIMIT_ORDER:    []byte{76, 124, 128, 15, 213, 87, 37, 250},
		},
		// Fee and reward collection; they move tokens but are not swaps
		OTHER: RaydiumCLOtherDiscriminators{
			COLLECT_FUND_FEE:          []byte{167, 138, 78, 149, 223, 194, 6, 126},
			COLLECT_PROTOCOL_FEE:      []byte{136, 136, 252, 221, 194, 66, 126, 89},
			COLLECT_REMAINING_REWARDS: []byte{18, 237, 166, 197, 34, 16, 213, 144},
		},
	},
	RAYDIUM_CPMM: RaydiumCPMMDiscriminators{
		CREATE:           []byte{175, 175, 109, 31, 13, 152, 155, 237},
		ADD_LIQUIDITY:    []byte{242, 35, 198, 137, 82, 225, 242, 182},
		REMOVE_LIQUIDITY: []byte{183, 18, 70, 156, 148, 109, 161, 34},

		INITIALIZE_WITH_PERMISSION: []byte{63, 55, 254, 65, 49, 178, 89, 121},
		SWAP_BASE_INPUT:            []byte{143, 190, 90, 218, 196, 30, 51, 222},
		SWAP_BASE_OUTPUT:           []byte{55, 217, 98, 86, 163, 74, 180, 173},
		// Fee collection; moves tokens but is not a swap
		COLLECT_CREATOR_FEE:  []byte{20, 22, 86, 123, 198, 28, 219, 132},
		COLLECT_FUND_FEE:     []byte{167, 138, 78, 149, 223, 194, 6, 126},
		COLLECT_PROTOCOL_FEE: []byte{136, 136, 252, 221, 194, 66, 126, 89},
	},
	RAYDIUM_LCP: RaydiumLCPDiscriminators{
		CREATE_EVENT:      []byte{228, 69, 165, 46, 81, 203, 154, 29, 151, 215, 226, 9, 118, 161, 115, 174},
		TRADE_EVENT:       []byte{228, 69, 165, 46, 81, 203, 154, 29, 189, 219, 127, 211, 78, 230, 97, 238},
		MIGRATE_TO_AMM:    []byte{207, 82, 192, 145, 254, 207, 145, 223},
		MIGRATE_TO_CPSWAP: []byte{136, 92, 200, 103, 28, 218, 144, 140},
		BUY_EXACT_IN:      []byte{250, 234, 13, 123, 213, 156, 19, 236},
		BUY_EXACT_OUT:     []byte{24, 211, 116, 40, 105, 3, 153, 56},
		SELL_EXACT_IN:     []byte{149, 39, 222, 155, 211, 124, 152, 26},
		SELL_EXACT_OUT:    []byte{95, 200, 71, 34, 8, 9, 11, 166},
		INITIALIZE:        []byte{175, 175, 109, 31, 13, 152, 155, 237},

		// LaunchLab IDL v0.2.0 create variants (same accounts 0-7 as initialize)
		INITIALIZE_V2:              []byte{67, 153, 175, 39, 218, 16, 38, 32},
		INITIALIZE_WITH_TOKEN_2022: []byte{37, 190, 126, 222, 44, 154, 171, 17},
	},
	METEORA_DLMM: MeteoraDLMMDiscriminators{
		ADD_LIQUIDITY: map[string][]byte{
			"addLiquidity":                  {181, 157, 89, 67, 143, 182, 52, 72},
			"addLiquidityByStrategy":        {7, 3, 150, 127, 148, 40, 61, 200},
			"addLiquidityByStrategy2":       {3, 221, 149, 218, 111, 141, 118, 213},
			"addLiquidityByStrategyOneSide": {41, 5, 238, 175, 100, 225, 6, 205},
			"addLiquidityOneSide":           {94, 155, 103, 151, 70, 95, 220, 165},
			"addLiquidityOneSidePrecise":    {161, 194, 103, 84, 171, 71, 250, 154},
			"addLiquidityByWeight":          {28, 140, 238, 99, 231, 162, 21, 149},
			// lb_clmm 0.12.0
			"addLiquidity2":               {228, 162, 78, 28, 70, 219, 116, 115},
			"addLiquidityByWeight2":       {209, 59, 63, 91, 111, 200, 153, 228},
			"addLiquidityOneSidePrecise2": {33, 51, 163, 201, 117, 98, 125, 231},
		},
		REMOVE_LIQUIDITY: map[string][]byte{
			"removeLiquidity":         {80, 85, 209, 72, 24, 206, 177, 108},
			"removeLiquidityByRange":  {26, 82, 102, 152, 240, 74, 105, 26},
			"removeLiquidityByRange2": {204, 2, 195, 145, 53, 145, 145, 205},
			"removeAllLiquidity":      {10, 51, 61, 35, 112, 105, 24, 85},
			"claimFee":                {169, 32, 79, 137, 136, 232, 70, 137},
			"claimFeeV2":              {112, 191, 101, 171, 28, 144, 127, 187},
			// lb_clmm 0.12.0
			"removeLiquidity2": {230, 215, 82, 127, 241, 101, 227, 146},
			"claimReward":      {149, 95, 181, 242, 94, 90, 158, 162},
			"claimReward2":     {190, 3, 127, 119, 178, 87, 157, 183},
		},
		LIQUIDITY_EVENT: map[string][]byte{
			"compositionFeeEvent":  {228, 69, 165, 46, 81, 203, 154, 29, 128, 151, 123, 106, 17, 102, 113, 142},
			"addLiquidityEvent":    {228, 69, 165, 46, 81, 203, 154, 29, 31, 94, 125, 90, 227, 52, 61, 186},
			"removeLiquidityEvent": {228, 69, 165, 46, 81, 203, 154, 29, 116, 244, 97, 232, 103, 31, 152, 58},
		},

		// lb_clmm 0.12.0 instructions that are not add/remove liquidity
		SWAP: map[string][]byte{
			"swap":                 {248, 198, 158, 145, 225, 117, 135, 200},
			"swap2":                {65, 75, 63, 76, 235, 91, 91, 136},
			"swapExactOut":         {250, 73, 101, 33, 38, 207, 75, 184},
			"swapExactOut2":        {43, 215, 247, 132, 137, 60, 243, 81},
			"swapWithPriceImpact":  {56, 173, 230, 208, 173, 228, 156, 205},
			"swapWithPriceImpact2": {74, 98, 192, 214, 177, 51, 75, 51},
		},
		CREATE: map[string][]byte{
			"initializeLbPair":                            {45, 154, 237, 210, 221, 15, 166, 92},
			"initializeLbPair2":                           {73, 59, 36, 120, 237, 83, 108, 198},
			"initializePermissionLbPair":                  {108, 102, 213, 85, 251, 3, 53, 21},
			"initializeCustomizablePermissionlessLbPair":  {46, 39, 41, 135, 111, 183, 200, 64},
			"initializeCustomizablePermissionlessLbPair2": {243, 73, 129, 126, 51, 19, 241, 107},
		},
		// Limit orders move tokens but are not swaps
		LIMIT_ORDER: map[string][]byte{
			"placeLimitOrder":        {108, 176, 33, 186, 146, 229, 1, 197},
			"cancelLimitOrder":       {132, 156, 132, 31, 67, 40, 232, 97},
			"closeLimitOrderIfEmpty": {57, 124, 36, 155, 126, 249, 93, 171},
		},
		// Other instructions that move tokens but are not swaps. rebalance_liquidity can
		// deposit and withdraw in the same instruction, so it is neither ADD nor REMOVE.
		OTHER: map[string][]byte{
			"rebalanceLiquidity": {92, 4, 176, 193, 119, 185, 83, 9},
		},
	},
	METEORA_DAMM: MeteoraDAMMDiscriminators{
		CREATE:                  []byte{7, 166, 138, 171, 206, 171, 236, 244},
		ADD_LIQUIDITY:           []byte{168, 227, 50, 62, 189, 171, 84, 176},
		REMOVE_LIQUIDITY:        []byte{133, 109, 44, 179, 56, 238, 114, 33},
		ADD_IMBALANCE_LIQUIDITY: []byte{79, 35, 122, 84, 173, 15, 93, 191},

		// amm 0.5.2 IDL (camelCase names hashed as snake_case)
		SWAP:                                     []byte{248, 198, 158, 145, 225, 117, 135, 200},
		CREATE_PERMISSIONED_POOL:                 []byte{77, 85, 178, 157, 50, 48, 212, 126},
		CREATE_PERMISSIONLESS_POOL:               []byte{118, 173, 41, 157, 173, 72, 97, 103},
		CREATE_PERMISSIONLESS_POOL_WITH_FEE_TIER: []byte{6, 135, 68, 147, 229, 82, 169, 113},
		CREATE_WITH_CONFIG2:                      []byte{48, 149, 220, 130, 61, 11, 9, 178},
		CREATE_CUSTOMIZABLE:                      []byte{145, 24, 172, 194, 219, 125, 3, 190},
		REMOVE_LIQUIDITY_SINGLE_SIDE:             []byte{84, 84, 177, 66, 254, 185, 10, 251},
		BOOTSTRAP_LIQUIDITY:                      []byte{4, 228, 215, 71, 225, 253, 119, 206},
		CLAIM_FEE:                                []byte{169, 32, 79, 137, 136, 232, 70, 137},
		PARTNER_CLAIM_FEE:                        []byte{57, 53, 176, 30, 123, 70, 52, 64},
	},
	METEORA_DAMM_V2: MeteoraDAMMV2Discriminators{
		INITIALIZE_POOL:                     []byte{95, 180, 10, 172, 84, 174, 232, 40},
		INITIALIZE_CUSTOM_POOL:              []byte{20, 161, 241, 24, 189, 221, 180, 2},
		INITIALIZE_POOL_WITH_DYNAMIC_CONFIG: []byte{149, 82, 72, 197, 253, 252, 68, 15},
		ADD_LIQUIDITY:                       []byte{181, 157, 89, 67, 143, 182, 52, 72},
		CLAIM_POSITION_FEE:                  []byte{180, 38, 154, 17, 133, 33, 162, 211},
		REMOVE_LIQUIDITY:                    []byte{80, 85, 209, 72, 24, 206, 177, 108},
		REMOVE_ALL_LIQUIDITY:                []byte{10, 51, 61, 35, 112, 105, 24, 85},
		CREATE_POSITION_EVENT:               []byte{228, 69, 165, 46, 81, 203, 154, 29, 156, 15, 119, 198, 29, 181, 221, 55},

		// cp_amm 0.2.0 IDL
		SWAP:            []byte{248, 198, 158, 145, 225, 117, 135, 200},
		SWAP2:           []byte{65, 75, 63, 76, 235, 91, 91, 136},
		CLAIM_REWARD:    []byte{149, 95, 181, 242, 94, 90, 158, 162},
		SPLIT_POSITION:  []byte{172, 241, 221, 138, 161, 29, 253, 42},
		SPLIT_POSITION2: []byte{221, 147, 228, 207, 140, 212, 17, 119},
	},
	METEORA_DBC: MeteoraDBCDiscriminators{
		SWAP:                                   []byte{248, 198, 158, 145, 225, 117, 135, 200},
		SWAP_V2:                                []byte{65, 75, 63, 76, 235, 91, 91, 136},
		INITIALIZE_VIRTUAL_POOL_WITH_SPL:       []byte{140, 85, 215, 176, 102, 54, 104, 79},
		INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022: []byte{169, 118, 51, 78, 145, 110, 220, 155},
		METEORA_DBC_MIGRATE_DAMM:               []byte{27, 1, 48, 22, 180, 63, 118, 217},
		METEORA_DBC_MIGRATE_DAMM_V2:            []byte{156, 169, 230, 103, 53, 228, 80, 64},

		// dynamic_bonding_curve IDL 0.2.1
		SWAP2_WITH_TRANSFER_HOOK:                             []byte{183, 93, 153, 40, 24, 230, 194, 151},
		INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022_TRANSFER_HOOK: []byte{182, 13, 233, 177, 42, 145, 135, 2},
		// Self-CPI events (anchor event prefix + event:<Name>)
		EVT_SWAP:                               []byte{228, 69, 165, 46, 81, 203, 154, 29, 27, 60, 21, 213, 138, 170, 187, 147},
		EVT_SWAP2:                              []byte{228, 69, 165, 46, 81, 203, 154, 29, 189, 66, 51, 168, 38, 80, 117, 153},
		EVT_SWAP2_WITH_TRANSFER_HOOK:           []byte{228, 69, 165, 46, 81, 203, 154, 29, 134, 59, 168, 120, 94, 51, 114, 231},
		EVT_CURVE_COMPLETE:                     []byte{228, 69, 165, 46, 81, 203, 154, 29, 229, 231, 86, 84, 156, 134, 75, 24},
		EVT_CURVE_COMPLETE_WITH_TRANSFER_HOOK:  []byte{228, 69, 165, 46, 81, 203, 154, 29, 59, 47, 109, 205, 13, 31, 44, 159},
		EVT_INITIALIZE_POOL:                    []byte{228, 69, 165, 46, 81, 203, 154, 29, 228, 50, 246, 85, 203, 66, 134, 37},
		EVT_INITIALIZE_POOL_WITH_TRANSFER_HOOK: []byte{228, 69, 165, 46, 81, 203, 154, 29, 213, 137, 164, 53, 193, 74, 15, 110},
	},
	ORCA: OrcaDiscriminators{
		CREATE:           []byte{242, 29, 134, 48, 58, 110, 14, 60},
		CREATE2:          []byte{212, 47, 95, 92, 114, 102, 131, 250},
		ADD_LIQUIDITY:    []byte{46, 156, 243, 118, 13, 205, 251, 178},
		ADD_LIQUIDITY2:   []byte{133, 29, 89, 223, 69, 238, 176, 10},
		REMOVE_LIQUIDITY: []byte{160, 38, 208, 111, 104, 91, 44, 1},
		OTHER1:           []byte{164, 152, 207, 99, 30, 186, 19, 182}, // collect_fees
		OTHER2:           []byte{70, 5, 132, 87, 86, 235, 177, 34},    // collect_reward

		// whirlpool 0.9.0 IDL
		SWAP:                              []byte{248, 198, 158, 145, 225, 117, 135, 200},
		SWAP_V2:                           []byte{43, 4, 237, 11, 26, 201, 30, 98},
		TWO_HOP_SWAP:                      []byte{195, 96, 237, 108, 68, 162, 219, 230},
		TWO_HOP_SWAP_V2:                   []byte{186, 143, 209, 29, 254, 2, 194, 117},
		REMOVE_LIQUIDITY_V2:               []byte{58, 127, 188, 62, 79, 82, 196, 96},  // decrease_liquidity_v2
		ADD_LIQUIDITY_BY_TOKEN_AMOUNTS_V2: []byte{239, 251, 9, 124, 210, 198, 53, 43}, // increase_liquidity_by_token_amounts_v2
		REPOSITION_LIQUIDITY_V2:           []byte{191, 169, 224, 11, 131, 19, 158, 253},
		COLLECT_FEES_V2:                   []byte{207, 117, 95, 191, 229, 180, 226, 15},
		COLLECT_REWARD_V2:                 []byte{177, 107, 37, 180, 160, 19, 49, 209},
		COLLECT_PROTOCOL_FEES:             []byte{22, 67, 23, 98, 150, 178, 70, 220},
		COLLECT_PROTOCOL_FEES_V2:          []byte{103, 128, 222, 134, 114, 200, 22, 200},
		// Whirlpool events are emitted with emit! ("Program data:" log lines): 8 bytes, no CPI prefix
		TRADED_EVENT:              []byte{225, 202, 73, 175, 147, 43, 160, 150},
		LIQUIDITY_INCREASED_EVENT: []byte{30, 7, 144, 181, 102, 254, 155, 161},
		LIQUIDITY_DECREASED_EVENT: []byte{166, 1, 36, 71, 112, 202, 181, 171},
	},
	BOOPFUN: BoopfunDiscriminators{
		CREATE:   []byte{84, 52, 204, 228, 24, 140, 234, 75},
		DEPLOY:   []byte{180, 89, 199, 76, 168, 236, 217, 138},
		COMPLETE: []byte{45, 235, 225, 181, 17, 218, 64, 130},
		BUY:      []byte{138, 127, 14, 91, 38, 87, 115, 105},
		SELL:     []byte{109, 61, 40, 187, 230, 176, 135, 174},
	},
	HEAVEN: HeavenDiscriminators{
		BUY:         []byte{102, 6, 61, 18, 1, 218, 235, 234},
		SELL:        []byte{51, 230, 133, 164, 1, 127, 131, 173},
		CREATE_POOL: []byte{42, 43, 126, 56, 231, 10, 208, 53},
	},
	METAPLEX: MetaplexDiscriminators{
		CREATE_MINT: []byte{42},
	},
	SUGAR: SugarDiscriminators{
		BUY_EXACT_IN:      []byte{250, 234, 13, 123, 213, 156, 19, 236},
		BUY_EXACT_OUT:     []byte{24, 211, 116, 40, 105, 3, 153, 56},
		BUY_MAX_OUT:       []byte{96, 177, 203, 117, 183, 65, 196, 177},
		SELL_EXACT_IN:     []byte{149, 39, 222, 155, 211, 124, 152, 26},
		SELL_EXACT_OUT:    []byte{95, 200, 71, 34, 8, 9, 11, 166},
		CREATE:            []byte{24, 30, 200, 40, 5, 28, 7, 119},
		INITIALIZE:        []byte{175, 175, 109, 31, 13, 152, 155, 237},
		MIGRATE_TO_RADIUM: []byte{96, 230, 91, 140, 139, 40, 235, 142},
	},
	PHOTON: PhotonDiscriminators{
		PUMPSWAP_TRADE: []byte{44, 119, 175, 218, 199, 77, 196, 235},
		PUMPFUN_BUY:    []byte{82, 225, 119, 231, 78, 29, 45, 70},
		PUMPFUN_SELL:   []byte{93, 88, 60, 34, 91, 18, 86, 197},
		MOONIT_BUY:     []byte{61, 220, 193, 108, 173, 62, 69, 176},
		MOONIT_SELL:    []byte{206, 188, 188, 107, 32, 145, 81, 150},
		HOP_TWO_SWAP:   []byte{195, 96, 237, 108, 68, 162, 219, 230},

		// Observed as outer Photon instructions in 2026-09 mainnet txs
		PUMPFUN_BUY_V2:  []byte{27, 79, 2, 101, 40, 156, 35, 179},   // global:pump_buy_v2
		PUMPFUN_SELL_V2: []byte{65, 127, 9, 177, 231, 236, 105, 80}, // global:pump_sell_v2
		COLLECT_FEE:     []byte{60, 173, 247, 103, 4, 93, 130, 48},  // global:collect_fee
	},

	// Prop AMM discriminators
	SOLFI: SolFiDiscriminators{
		SWAP: []byte{0x07},
	},
	GOONFI: GoonFiDiscriminators{
		SWAP: []byte{0x02},
	},
	OBRIC: ObricDiscriminators{
		// Anchor: swap. The only swap instruction seen in real Obric V2 txs (Jupiter CPI,
		// 25 bytes, 12 accounts; e.g. 2DaS55Tw..., 2024-11-27).
		SWAP: []byte{248, 198, 158, 145, 225, 117, 135, 200},
		// Unverified legacy values, never observed in Obric V2 txs. SWAP_X_TO_Y equals
		// sha256("global:swap_base_input")[:8] (Raydium CPMM), not "swap_x_to_y";
		// SWAP_Y_TO_X matches no known name. Kept for compatibility.
		SWAP_X_TO_Y: []byte{143, 190, 90, 218, 196, 30, 51, 222},
		SWAP_Y_TO_X: []byte{220, 117, 232, 239, 48, 247, 211, 180},
	},
	DFLOW: DFlowDiscriminators{
		SWAP:           []byte{248, 198, 158, 145, 225, 117, 135, 200},
		SWAP2:          []byte{65, 75, 63, 76, 235, 91, 91, 136},
		SWAP_WITH_DEST: []byte{168, 172, 24, 77, 197, 156, 135, 101},
		OPEN_ORDER:     []byte{206, 88, 88, 143, 38, 136, 50, 224},
		FILL_ORDER:     []byte{232, 122, 115, 25, 199, 143, 136, 162},
		CLOSE_ORDER:    []byte{90, 103, 209, 28, 7, 63, 168, 4},

		// swap_orchestrator IDL
		SWAP2_WITH_DEST:        []byte{95, 123, 213, 246, 122, 1, 86, 231},
		SWAP_WITH_DEST_NATIVE:  []byte{205, 77, 127, 108, 241, 32, 196, 195},
		SWAP2_WITH_DEST_NATIVE: []byte{222, 100, 184, 146, 186, 196, 105, 165},
	},
	HUMIDIFI: HumidiFiDiscriminators{
		// HumidiFi uses XOR encryption, discriminator after decryption
		SWAP: []byte{248, 198, 158, 145, 225, 117, 135, 200},
	},

	JUPITER_Z: JupiterZDiscriminators{
		FILL: []byte{168, 96, 183, 163, 92, 10, 40, 160}, // order_engine IDL: fill
	},
}

// Discriminator type definitions
type JupiterDiscriminators struct {
	ROUTE_EVENT []byte
	// Jupiter V6 shred discriminators
	ROUTE                                  []byte
	ROUTE_EXACT_OUT                        []byte
	ROUTE_WITH_TOKEN_LEDGER                []byte
	SHARE_ACCOUNTS_ROUTE                   []byte
	SHARE_ACCOUNTS_EXACT_OUT_ROUTE         []byte
	SHARE_ACCOUNTS_ROUTE_WITH_TOKEN_LEDGER []byte

	ROUTE_V2                           []byte
	SHARED_ACCOUNTS_ROUTE_V2           []byte
	EXACT_OUT_ROUTE_V2                 []byte
	SHARED_ACCOUNTS_EXACT_OUT_ROUTE_V2 []byte
	SWAPS_EVENT                        []byte
	FEE_EVENT                          []byte
}

type JupiterDCADiscriminators struct {
	FILLED      []byte
	CLOSE_DCA   []byte
	OPEN_DCA    []byte
	OPEN_DCA_V2 []byte
}

type JupiterLimitOrderDiscriminators struct {
	CANCEL_ORDER     []byte
	CREATE_ORDER     []byte
	TRADE_EVENT      []byte
	UNKNOWN          []byte
	FLASH_FILL_ORDER []byte
}

type JupiterLimitOrderV2Discriminators struct {
	CANCEL_ORDER       []byte
	CREATE_ORDER_EVENT []byte
	TRADE_EVENT        []byte
	UNKNOWN            []byte
	FLASH_FILL_ORDER   []byte

	CANCEL_DUST_ORDER       []byte
	CANCEL_ORDER_EVENT      []byte
	CANCEL_DUST_ORDER_EVENT []byte
}

type JupiterVADiscriminators struct {
	FILL_EVENT     []byte
	OPEN_EVENT     []byte
	CLOSE_EVENT    []byte
	DEPOSIT_EVENT  []byte
	WITHDRAW_EVENT []byte
}

type PumpfunDiscriminators struct {
	CREATE         []byte
	MIGRATE        []byte
	BUY            []byte
	SELL           []byte
	TRADE_EVENT    []byte
	CREATE_EVENT   []byte
	COMPLETE_EVENT []byte
	MIGRATE_EVENT  []byte

	BUY_V2                []byte
	SELL_V2               []byte
	BUY_EXACT_SOL_IN      []byte
	BUY_EXACT_QUOTE_IN_V2 []byte
	CREATE_V2             []byte
	MIGRATE_V2            []byte
}

type PumpswapDiscriminators struct {
	CREATE_POOL            []byte
	ADD_LIQUIDITY          []byte
	REMOVE_LIQUIDITY       []byte
	BUY                    []byte
	SELL                   []byte
	CREATE_POOL_EVENT      []byte
	ADD_LIQUIDITY_EVENT    []byte
	REMOVE_LIQUIDITY_EVENT []byte
	BUY_EVENT              []byte
	SELL_EVENT             []byte

	BUY_EXACT_QUOTE_IN []byte
}

type MoonitDiscriminators struct {
	BUY     []byte
	SELL    []byte
	CREATE  []byte
	MIGRATE []byte
}

type RaydiumDiscriminators struct {
	CREATE           []byte
	ADD_LIQUIDITY    []byte
	REMOVE_LIQUIDITY []byte
	SWAP             []byte
	SWAP_EXACT_OUT   []byte

	SWAP_V2           []byte
	SWAP_EXACT_OUT_V2 []byte
}

type RaydiumCLDiscriminators struct {
	CREATE           RaydiumCLCreateDiscriminators
	ADD_LIQUIDITY    RaydiumCLAddLiquidityDiscriminators
	REMOVE_LIQUIDITY RaydiumCLRemoveLiquidityDiscriminators
	SWAP             RaydiumCLSwapDiscriminators
	EVENTS           RaydiumCLEventDiscriminators

	LIMIT_ORDER RaydiumCLLimitOrderDiscriminators
	OTHER       RaydiumCLOtherDiscriminators
}

type RaydiumCLCreateDiscriminators struct {
	OPEN_POSITION    []byte
	OPEN_POSITION_V2 []byte
	CREATE_POOL      []byte

	CREATE_CUSTOMIZABLE_POOL []byte
}

type RaydiumCLAddLiquidityDiscriminators struct {
	INCREASE_LIQUIDITY         []byte
	INCREASE_LIQUIDITY_V2      []byte
	OPEN_POSITION_WITH_TOKEN22 []byte
}

type RaydiumCLRemoveLiquidityDiscriminators struct {
	DECREASE_LIQUIDITY    []byte
	DECREASE_LIQUIDITY_V2 []byte
	CLOSE_POSITION        []byte
}

type RaydiumCLSwapDiscriminators struct {
	SWAP                []byte
	SWAP_V2             []byte
	SWAP_ROUTER_BASE_IN []byte
}

type RaydiumCLEventDiscriminators struct {
	COLLECT_PERSONAL_FEE     []byte
	COLLECT_PROTOCOL_FEE     []byte
	CONFIG_CHANGE            []byte
	CREATE_PERSONAL_POSITION []byte
	DECREASE_LIQUIDITY       []byte
	INCREASE_LIQUIDITY       []byte
	LIQUIDITY_CALCULATE      []byte
	LIQUIDITY_CHANGE         []byte
	POOL_CREATED             []byte
	SWAP                     []byte
	UPDATE_REWARD_INFOS      []byte

	OPEN_LIMIT_ORDER     []byte
	INCREASE_LIMIT_ORDER []byte
	DECREASE_LIMIT_ORDER []byte
	SETTLE_LIMIT_ORDER   []byte
}

type RaydiumCPMMDiscriminators struct {
	CREATE           []byte
	ADD_LIQUIDITY    []byte
	REMOVE_LIQUIDITY []byte

	INITIALIZE_WITH_PERMISSION []byte
	SWAP_BASE_INPUT            []byte
	SWAP_BASE_OUTPUT           []byte
	COLLECT_CREATOR_FEE        []byte
	COLLECT_FUND_FEE           []byte
	COLLECT_PROTOCOL_FEE       []byte
}

type RaydiumLCPDiscriminators struct {
	CREATE_EVENT      []byte
	TRADE_EVENT       []byte
	MIGRATE_TO_AMM    []byte
	MIGRATE_TO_CPSWAP []byte
	BUY_EXACT_IN      []byte
	BUY_EXACT_OUT     []byte
	SELL_EXACT_IN     []byte
	SELL_EXACT_OUT    []byte
	INITIALIZE        []byte

	INITIALIZE_V2              []byte
	INITIALIZE_WITH_TOKEN_2022 []byte
}

type MeteoraDLMMDiscriminators struct {
	ADD_LIQUIDITY    map[string][]byte
	REMOVE_LIQUIDITY map[string][]byte
	LIQUIDITY_EVENT  map[string][]byte

	SWAP        map[string][]byte
	CREATE      map[string][]byte
	LIMIT_ORDER map[string][]byte
	OTHER       map[string][]byte
}

type MeteoraDAMMDiscriminators struct {
	CREATE                  []byte
	ADD_LIQUIDITY           []byte
	REMOVE_LIQUIDITY        []byte
	ADD_IMBALANCE_LIQUIDITY []byte

	SWAP                                     []byte
	CREATE_PERMISSIONED_POOL                 []byte
	CREATE_PERMISSIONLESS_POOL               []byte
	CREATE_PERMISSIONLESS_POOL_WITH_FEE_TIER []byte
	CREATE_WITH_CONFIG2                      []byte
	CREATE_CUSTOMIZABLE                      []byte
	REMOVE_LIQUIDITY_SINGLE_SIDE             []byte
	BOOTSTRAP_LIQUIDITY                      []byte
	CLAIM_FEE                                []byte
	PARTNER_CLAIM_FEE                        []byte
}

type MeteoraDAMMV2Discriminators struct {
	INITIALIZE_POOL                     []byte
	INITIALIZE_CUSTOM_POOL              []byte
	INITIALIZE_POOL_WITH_DYNAMIC_CONFIG []byte
	ADD_LIQUIDITY                       []byte
	CLAIM_POSITION_FEE                  []byte
	REMOVE_LIQUIDITY                    []byte
	REMOVE_ALL_LIQUIDITY                []byte
	CREATE_POSITION_EVENT               []byte

	SWAP            []byte
	SWAP2           []byte
	CLAIM_REWARD    []byte
	SPLIT_POSITION  []byte
	SPLIT_POSITION2 []byte
}

type MeteoraDBCDiscriminators struct {
	SWAP                                   []byte
	SWAP_V2                                []byte
	INITIALIZE_VIRTUAL_POOL_WITH_SPL       []byte
	INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022 []byte
	METEORA_DBC_MIGRATE_DAMM               []byte
	METEORA_DBC_MIGRATE_DAMM_V2            []byte

	SWAP2_WITH_TRANSFER_HOOK                             []byte
	INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022_TRANSFER_HOOK []byte
	EVT_SWAP                                             []byte
	EVT_SWAP2                                            []byte
	EVT_SWAP2_WITH_TRANSFER_HOOK                         []byte
	EVT_CURVE_COMPLETE                                   []byte
	EVT_CURVE_COMPLETE_WITH_TRANSFER_HOOK                []byte
	EVT_INITIALIZE_POOL                                  []byte
	EVT_INITIALIZE_POOL_WITH_TRANSFER_HOOK               []byte
}

type OrcaDiscriminators struct {
	CREATE           []byte
	CREATE2          []byte
	ADD_LIQUIDITY    []byte
	ADD_LIQUIDITY2   []byte
	REMOVE_LIQUIDITY []byte
	OTHER1           []byte
	OTHER2           []byte

	SWAP                              []byte
	SWAP_V2                           []byte
	TWO_HOP_SWAP                      []byte
	TWO_HOP_SWAP_V2                   []byte
	REMOVE_LIQUIDITY_V2               []byte
	ADD_LIQUIDITY_BY_TOKEN_AMOUNTS_V2 []byte
	REPOSITION_LIQUIDITY_V2           []byte
	COLLECT_FEES_V2                   []byte
	COLLECT_REWARD_V2                 []byte
	COLLECT_PROTOCOL_FEES             []byte
	COLLECT_PROTOCOL_FEES_V2          []byte
	TRADED_EVENT                      []byte
	LIQUIDITY_INCREASED_EVENT         []byte
	LIQUIDITY_DECREASED_EVENT         []byte
}

type BoopfunDiscriminators struct {
	CREATE   []byte
	DEPLOY   []byte
	COMPLETE []byte
	BUY      []byte
	SELL     []byte
}

type HeavenDiscriminators struct {
	BUY         []byte
	SELL        []byte
	CREATE_POOL []byte
}

type MetaplexDiscriminators struct {
	CREATE_MINT []byte
}

type SugarDiscriminators struct {
	BUY_EXACT_IN      []byte
	BUY_EXACT_OUT     []byte
	BUY_MAX_OUT       []byte
	SELL_EXACT_IN     []byte
	SELL_EXACT_OUT    []byte
	CREATE            []byte
	INITIALIZE        []byte
	MIGRATE_TO_RADIUM []byte
}

type PhotonDiscriminators struct {
	PUMPSWAP_TRADE []byte
	PUMPFUN_BUY    []byte
	PUMPFUN_SELL   []byte
	MOONIT_BUY     []byte
	MOONIT_SELL    []byte
	HOP_TWO_SWAP   []byte

	PUMPFUN_BUY_V2  []byte
	PUMPFUN_SELL_V2 []byte
	COLLECT_FEE     []byte
}

type SolFiDiscriminators struct {
	SWAP []byte
}

type GoonFiDiscriminators struct {
	SWAP []byte
}

type ObricDiscriminators struct {
	SWAP        []byte
	SWAP_X_TO_Y []byte
	SWAP_Y_TO_X []byte
}

type DFlowDiscriminators struct {
	SWAP           []byte
	SWAP2          []byte
	SWAP_WITH_DEST []byte
	OPEN_ORDER     []byte
	FILL_ORDER     []byte
	CLOSE_ORDER    []byte

	SWAP2_WITH_DEST        []byte
	SWAP_WITH_DEST_NATIVE  []byte
	SWAP2_WITH_DEST_NATIVE []byte
}

type HumidiFiDiscriminators struct {
	SWAP []byte
}

type JupiterZDiscriminators struct {
	FILL []byte
}

type RaydiumCLLimitOrderDiscriminators struct {
	OPEN_LIMIT_ORDER     []byte
	INCREASE_LIMIT_ORDER []byte
	DECREASE_LIMIT_ORDER []byte
	SETTLE_LIMIT_ORDER   []byte
	CLOSE_LIMIT_ORDER    []byte
}

type RaydiumCLOtherDiscriminators struct {
	COLLECT_FUND_FEE          []byte
	COLLECT_PROTOCOL_FEE      []byte
	COLLECT_REMAINING_REWARDS []byte
}

// MatchDiscriminator checks if data starts with the given discriminator.
// An empty discriminator never matches.
func MatchDiscriminator(data []byte, discriminator []byte) bool {
	if len(discriminator) == 0 || len(data) < len(discriminator) {
		return false
	}
	for i, b := range discriminator {
		if data[i] != b {
			return false
		}
	}
	return true
}

// MatchAnyDiscriminator checks if data matches any of the given discriminators
func MatchAnyDiscriminator(data []byte, discriminators map[string][]byte) (string, bool) {
	for name, disc := range discriminators {
		if MatchDiscriminator(data, disc) {
			return name, true
		}
	}
	return "", false
}
