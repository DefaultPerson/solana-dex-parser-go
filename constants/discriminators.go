package constants

// Discriminator byte slices for instruction identification
var DISCRIMINATORS = struct {
	JUPITER            JupiterDiscriminators
	JUPITER_DCA        JupiterDCADiscriminators
	JUPITER_LIMIT_ORDER JupiterLimitOrderDiscriminators
	JUPITER_LIMIT_ORDER_V2 JupiterLimitOrderV2Discriminators
	JUPITER_VA         JupiterVADiscriminators
	PUMPFUN            PumpfunDiscriminators
	PUMPSWAP           PumpswapDiscriminators
	MOONIT             MoonitDiscriminators
	RAYDIUM            RaydiumDiscriminators
	RAYDIUM_CL         RaydiumCLDiscriminators
	RAYDIUM_CPMM       RaydiumCPMMDiscriminators
	RAYDIUM_LCP        RaydiumLCPDiscriminators
	METEORA_DLMM       MeteoraDLMMDiscriminators
	METEORA_DAMM       MeteoraDAMMDiscriminators
	METEORA_DAMM_V2    MeteoraDAMMV2Discriminators
	METEORA_DBC        MeteoraDBCDiscriminators
	ORCA               OrcaDiscriminators
	BOOPFUN            BoopfunDiscriminators
	HEAVEN             HeavenDiscriminators
	METAPLEX           MetaplexDiscriminators
	SUGAR              SugarDiscriminators
	PHOTON             PhotonDiscriminators
	SOLFI              SolFiDiscriminators
	GOONFI             GoonFiDiscriminators
	OBRIC              ObricDiscriminators
	DFLOW              DFlowDiscriminators
	HUMIDIFI           HumidiFiDiscriminators
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
	},
	JUPITER_VA: JupiterVADiscriminators{
		FILL_EVENT:     []byte{228, 69, 165, 46, 81, 203, 154, 29, 78, 225, 199, 154, 86, 219, 224, 169},
		OPEN_EVENT:     []byte{228, 69, 165, 46, 81, 203, 154, 29, 104, 220, 224, 191, 87, 241, 132, 61},
		CLOSE_EVENT:    []byte{},
		DEPOSIT_EVENT:  []byte{},
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
	},
	PUMPSWAP: PumpswapDiscriminators{
		CREATE_POOL:           []byte{233, 146, 209, 142, 207, 104, 64, 188},
		ADD_LIQUIDITY:         []byte{242, 35, 198, 137, 82, 225, 242, 182},
		REMOVE_LIQUIDITY:      []byte{183, 18, 70, 156, 148, 109, 161, 34},
		BUY:                   []byte{102, 6, 61, 18, 1, 218, 235, 234},
		SELL:                  []byte{51, 230, 133, 164, 1, 127, 131, 173},
		CREATE_POOL_EVENT:     []byte{228, 69, 165, 46, 81, 203, 154, 29, 177, 49, 12, 210, 160, 118, 167, 116},
		ADD_LIQUIDITY_EVENT:   []byte{228, 69, 165, 46, 81, 203, 154, 29, 120, 248, 61, 83, 31, 142, 107, 144},
		REMOVE_LIQUIDITY_EVENT: []byte{228, 69, 165, 46, 81, 203, 154, 29, 22, 9, 133, 26, 160, 44, 71, 192},
		BUY_EVENT:             []byte{228, 69, 165, 46, 81, 203, 154, 29, 103, 244, 82, 31, 44, 245, 119, 119},
		SELL_EVENT:            []byte{228, 69, 165, 46, 81, 203, 154, 29, 62, 47, 55, 10, 165, 3, 220, 42},
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
	},
	RAYDIUM_CL: RaydiumCLDiscriminators{
		CREATE: RaydiumCLCreateDiscriminators{
			OPEN_POSITION:    []byte{135, 128, 47, 77, 15, 152, 240, 49},
			OPEN_POSITION_V2: []byte{77, 184, 74, 214, 112, 86, 241, 199},
			CREATE_POOL:      []byte{233, 146, 209, 142, 207, 104, 64, 188},
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
		},
	},
	RAYDIUM_CPMM: RaydiumCPMMDiscriminators{
		CREATE:           []byte{175, 175, 109, 31, 13, 152, 155, 237},
		ADD_LIQUIDITY:    []byte{242, 35, 198, 137, 82, 225, 242, 182},
		REMOVE_LIQUIDITY: []byte{183, 18, 70, 156, 148, 109, 161, 34},
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
	},
	METEORA_DLMM: MeteoraDLMMDiscriminators{
		ADD_LIQUIDITY: map[string][]byte{
			"addLiquidity":                 {181, 157, 89, 67, 143, 182, 52, 72},
			"addLiquidityByStrategy":       {7, 3, 150, 127, 148, 40, 61, 200},
			"addLiquidityByStrategy2":      {3, 221, 149, 218, 111, 141, 118, 213},
			"addLiquidityByStrategyOneSide": {41, 5, 238, 175, 100, 225, 6, 205},
			"addLiquidityOneSide":          {94, 155, 103, 151, 70, 95, 220, 165},
			"addLiquidityOneSidePrecise":   {161, 194, 103, 84, 171, 71, 250, 154},
			"addLiquidityByWeight":         {28, 140, 238, 99, 231, 162, 21, 149},
		},
		REMOVE_LIQUIDITY: map[string][]byte{
			"removeLiquidity":        {80, 85, 209, 72, 24, 206, 177, 108},
			"removeLiquidityByRange":  {26, 82, 102, 152, 240, 74, 105, 26},
			"removeLiquidityByRange2": {204, 2, 195, 145, 53, 145, 145, 205},
			"removeAllLiquidity":     {10, 51, 61, 35, 112, 105, 24, 85},
			"claimFee":               {169, 32, 79, 137, 136, 232, 70, 137},
			"claimFeeV2":             {112, 191, 101, 171, 28, 144, 127, 187},
		},
		LIQUIDITY_EVENT: map[string][]byte{
			"compositionFeeEvent":  {228, 69, 165, 46, 81, 203, 154, 29, 128, 151, 123, 106, 17, 102, 113, 142},
			"addLiquidityEvent":    {228, 69, 165, 46, 81, 203, 154, 29, 31, 94, 125, 90, 227, 52, 61, 186},
			"removeLiquidityEvent": {228, 69, 165, 46, 81, 203, 154, 29, 151, 113, 115, 164, 224, 159, 112, 193},
		},
	},
	METEORA_DAMM: MeteoraDAMMDiscriminators{
		CREATE:                 []byte{7, 166, 138, 171, 206, 171, 236, 244},
		ADD_LIQUIDITY:          []byte{168, 227, 50, 62, 189, 171, 84, 176},
		REMOVE_LIQUIDITY:       []byte{133, 109, 44, 179, 56, 238, 114, 33},
		ADD_IMBALANCE_LIQUIDITY: []byte{79, 35, 122, 84, 173, 15, 93, 191},
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
	},
	METEORA_DBC: MeteoraDBCDiscriminators{
		SWAP:                               []byte{248, 198, 158, 145, 225, 117, 135, 200},
		SWAP_V2:                            []byte{65, 75, 63, 76, 235, 91, 91, 136},
		INITIALIZE_VIRTUAL_POOL_WITH_SPL:   []byte{140, 85, 215, 176, 102, 54, 104, 79},
		INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022: []byte{169, 118, 51, 78, 145, 110, 220, 155},
		METEORA_DBC_MIGRATE_DAMM:           []byte{27, 1, 48, 22, 180, 63, 118, 217},
		METEORA_DBC_MIGRATE_DAMM_V2:        []byte{156, 169, 230, 103, 53, 228, 80, 64},
	},
	ORCA: OrcaDiscriminators{
		CREATE:           []byte{242, 29, 134, 48, 58, 110, 14, 60},
		CREATE2:          []byte{212, 47, 95, 92, 114, 102, 131, 250},
		ADD_LIQUIDITY:    []byte{46, 156, 243, 118, 13, 205, 251, 178},
		ADD_LIQUIDITY2:   []byte{133, 29, 89, 223, 69, 238, 176, 10},
		REMOVE_LIQUIDITY: []byte{160, 38, 208, 111, 104, 91, 44, 1},
		OTHER1:           []byte{164, 152, 207, 99, 30, 186, 19, 182},
		OTHER2:           []byte{70, 5, 132, 87, 86, 235, 177, 34},
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
		SELL_EXACT_OUT:    []byte{149, 95, 200, 71, 34, 8, 9, 11, 166},
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
	},

	// Prop AMM discriminators
	SOLFI: SolFiDiscriminators{
		SWAP: []byte{0x07},
	},
	GOONFI: GoonFiDiscriminators{
		SWAP: []byte{0x02},
	},
	OBRIC: ObricDiscriminators{
		SWAP:        []byte{248, 198, 158, 145, 225, 117, 135, 200}, // Anchor: swap
		SWAP_X_TO_Y: []byte{143, 190, 90, 218, 196, 30, 51, 222},    // Anchor: swapXToY
		SWAP_Y_TO_X: []byte{220, 117, 232, 239, 48, 247, 211, 180},  // Anchor: swapYToX
	},
	DFLOW: DFlowDiscriminators{
		SWAP:           []byte{248, 198, 158, 145, 225, 117, 135, 200},
		SWAP2:          []byte{65, 75, 63, 76, 235, 91, 91, 136},
		SWAP_WITH_DEST: []byte{168, 172, 24, 77, 197, 156, 135, 101},
		OPEN_ORDER:     []byte{206, 88, 88, 143, 38, 136, 50, 224},
		FILL_ORDER:     []byte{232, 122, 115, 25, 199, 143, 136, 162},
		CLOSE_ORDER:    []byte{90, 103, 209, 28, 7, 63, 168, 4},
	},
	HUMIDIFI: HumidiFiDiscriminators{
		// HumidiFi uses XOR encryption, discriminator after decryption
		SWAP: []byte{248, 198, 158, 145, 225, 117, 135, 200},
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
}

type PumpswapDiscriminators struct {
	CREATE_POOL           []byte
	ADD_LIQUIDITY         []byte
	REMOVE_LIQUIDITY      []byte
	BUY                   []byte
	SELL                  []byte
	CREATE_POOL_EVENT     []byte
	ADD_LIQUIDITY_EVENT   []byte
	REMOVE_LIQUIDITY_EVENT []byte
	BUY_EVENT             []byte
	SELL_EVENT            []byte
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
}

type RaydiumCLDiscriminators struct {
	CREATE           RaydiumCLCreateDiscriminators
	ADD_LIQUIDITY    RaydiumCLAddLiquidityDiscriminators
	REMOVE_LIQUIDITY RaydiumCLRemoveLiquidityDiscriminators
	SWAP             RaydiumCLSwapDiscriminators
	EVENTS           RaydiumCLEventDiscriminators
}

type RaydiumCLCreateDiscriminators struct {
	OPEN_POSITION    []byte
	OPEN_POSITION_V2 []byte
	CREATE_POOL      []byte
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
}

type RaydiumCPMMDiscriminators struct {
	CREATE           []byte
	ADD_LIQUIDITY    []byte
	REMOVE_LIQUIDITY []byte
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
}

type MeteoraDLMMDiscriminators struct {
	ADD_LIQUIDITY    map[string][]byte
	REMOVE_LIQUIDITY map[string][]byte
	LIQUIDITY_EVENT  map[string][]byte
}

type MeteoraDAMMDiscriminators struct {
	CREATE                 []byte
	ADD_LIQUIDITY          []byte
	REMOVE_LIQUIDITY       []byte
	ADD_IMBALANCE_LIQUIDITY []byte
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
}

type MeteoraDBCDiscriminators struct {
	SWAP                               []byte
	SWAP_V2                            []byte
	INITIALIZE_VIRTUAL_POOL_WITH_SPL   []byte
	INITIALIZE_VIRTUAL_POOL_WITH_TOKEN2022 []byte
	METEORA_DBC_MIGRATE_DAMM           []byte
	METEORA_DBC_MIGRATE_DAMM_V2        []byte
}

type OrcaDiscriminators struct {
	CREATE           []byte
	CREATE2          []byte
	ADD_LIQUIDITY    []byte
	ADD_LIQUIDITY2   []byte
	REMOVE_LIQUIDITY []byte
	OTHER1           []byte
	OTHER2           []byte
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
}

type HumidiFiDiscriminators struct {
	SWAP []byte
}

// MatchDiscriminator checks if data starts with the given discriminator
func MatchDiscriminator(data []byte, discriminator []byte) bool {
	if len(data) < len(discriminator) {
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
