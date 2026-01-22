package dexparser

import (
	"fmt"
	"sync"

	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/classifier"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/alt"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/dflow"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/jupiter"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/meme"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/meteora"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/orca"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/propamm"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/pumpfun"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

// ParseCallback defines the callback function type for batch parsing
// index: the index of the transaction in the batch
// tx: the transaction being parsed (may be nil)
// result: the parse result
// err: any error that occurred during parsing
// returns: true to continue processing, false to stop early
type ParseCallback func(index int, tx *adapter.SolanaTransaction, result *types.ParseResult, err error) bool

// DexParser is the main parser class for Solana DEX transactions
type DexParser struct {
	// Trade parsers by program ID
	tradeParserFactories map[string]TradeParserFactory

	// Liquidity parsers by program ID
	liquidityParserFactories map[string]LiquidityParserFactory

	// Transfer parsers by program ID
	transferParserFactories map[string]TransferParserFactory

	// Meme event parsers by program ID
	memeEventParserFactories map[string]MemeEventParserFactory
}

// TradeParserFactory creates a trade parser
type TradeParserFactory func(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) parsers.TradeParser

// LiquidityParserFactory creates a liquidity parser
type LiquidityParserFactory func(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) parsers.LiquidityParser

// TransferParserFactory creates a transfer parser
type TransferParserFactory func(
	adapter *adapter.TransactionAdapter,
	dexInfo types.DexInfo,
	transferActions map[string][]types.TransferData,
	classifiedInstructions []types.ClassifiedInstruction,
) parsers.TransferParser

// MemeEventParserFactory creates a meme event parser
type MemeEventParserFactory func(
	adapter *adapter.TransactionAdapter,
	transferActions map[string][]types.TransferData,
) parsers.EventParser

// NewDexParser creates a new DexParser instance
func NewDexParser() *DexParser {
	dp := &DexParser{
		tradeParserFactories:     make(map[string]TradeParserFactory, 20),
		liquidityParserFactories: make(map[string]LiquidityParserFactory, 10),
		transferParserFactories:  make(map[string]TransferParserFactory, 5),
		memeEventParserFactories: make(map[string]MemeEventParserFactory, 10),
	}

	// Register default parsers
	dp.registerDefaultParsers()

	return dp
}

// registerDefaultParsers registers all default parsers
func (dp *DexParser) registerDefaultParsers() {
	// Trade parsers
	dp.tradeParserFactories[constants.DEX_PROGRAMS.JUPITER.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return jupiter.NewJupiterParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.JUPITER_DCA.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return jupiter.NewJupiterDCAParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.JUPITER_VA.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return jupiter.NewJupiterVAParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return jupiter.NewJupiterLimitOrderV2Parser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.PUMP_FUN.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return pumpfun.NewPumpfunParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.PUMP_SWAP.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return pumpfun.NewPumpswapParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.METEORA.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return meteora.NewMeteoraParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.METEORA_DAMM.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return meteora.NewMeteoraParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return meteora.NewMeteoraParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.METEORA_DBC.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return meteora.NewMeteoraDBCParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.RAYDIUM_ROUTE.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return raydium.NewRaydiumParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.RAYDIUM_CL.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return raydium.NewRaydiumParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return raydium.NewRaydiumParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.RAYDIUM_V4.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return raydium.NewRaydiumParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.RAYDIUM_AMM.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return raydium.NewRaydiumParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.RAYDIUM_LCP.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return raydium.NewRaydiumLaunchpadParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.ORCA.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return orca.NewOrcaParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.BOOP_FUN.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return meme.NewBoopfunParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.MOONIT.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return meme.NewMoonitParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.HEAVEN.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return meme.NewHeavenParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.SUGAR.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return meme.NewSugarParser(a, d, t, c)
	}

	// Prop AMM parsers
	dp.tradeParserFactories[constants.DEX_PROGRAMS.SOLFI.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewSolFiParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.GOONFI.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewGoonFiParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.OBRIC_V2.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewObricParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.HUMIDIFI.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewHumidiFiParser(a, d, t, c)
	}

	// Aggregator parsers
	dp.tradeParserFactories[constants.DEX_PROGRAMS.DFLOW.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return dflow.NewDFlowParser(a, d, t, c)
	}

	// Liquidity parsers
	dp.liquidityParserFactories[constants.DEX_PROGRAMS.METEORA.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.LiquidityParser {
		return meteora.NewMeteoraDLMMPoolParser(a, t, c)
	}
	dp.liquidityParserFactories[constants.DEX_PROGRAMS.METEORA_DAMM.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.LiquidityParser {
		return meteora.NewMeteoraPoolsParser(a, t, c)
	}
	dp.liquidityParserFactories[constants.DEX_PROGRAMS.METEORA_DAMM_V2.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.LiquidityParser {
		return meteora.NewMeteoraDAMMPoolParser(a, t, c)
	}
	dp.liquidityParserFactories[constants.DEX_PROGRAMS.RAYDIUM_V4.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.LiquidityParser {
		return raydium.NewRaydiumV4PoolParser(a, t, c)
	}
	dp.liquidityParserFactories[constants.DEX_PROGRAMS.RAYDIUM_CPMM.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.LiquidityParser {
		return raydium.NewRaydiumCPMMPoolParser(a, t, c)
	}
	dp.liquidityParserFactories[constants.DEX_PROGRAMS.RAYDIUM_CL.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.LiquidityParser {
		return raydium.NewRaydiumCLPoolParser(a, t, c)
	}
	dp.liquidityParserFactories[constants.DEX_PROGRAMS.ORCA.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.LiquidityParser {
		return orca.NewOrcaLiquidityParser(a, t, c)
	}
	dp.liquidityParserFactories[constants.DEX_PROGRAMS.PUMP_SWAP.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.LiquidityParser {
		return pumpfun.NewPumpswapLiquidityParser(a, t, c)
	}

	// Transfer parsers
	dp.transferParserFactories[constants.DEX_PROGRAMS.JUPITER_DCA.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TransferParser {
		return jupiter.NewJupiterDCAParser(a, d, t, c)
	}
	dp.transferParserFactories[constants.DEX_PROGRAMS.JUPITER_VA.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TransferParser {
		return jupiter.NewJupiterVAParser(a, d, t, c)
	}
	dp.transferParserFactories[constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TransferParser {
		return jupiter.NewJupiterLimitOrderParser(a, d, t, c)
	}
	dp.transferParserFactories[constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TransferParser {
		return jupiter.NewJupiterLimitOrderV2Parser(a, d, t, c)
	}

	// Meme event parsers
	dp.memeEventParserFactories[constants.DEX_PROGRAMS.PUMP_FUN.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData) parsers.EventParser {
		return pumpfun.NewPumpfunEventParser(a, t)
	}
	dp.memeEventParserFactories[constants.DEX_PROGRAMS.PUMP_SWAP.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData) parsers.EventParser {
		return pumpfun.NewPumpswapEventParser(a, t)
	}
	dp.memeEventParserFactories[constants.DEX_PROGRAMS.MOONIT.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData) parsers.EventParser {
		return meme.NewMoonitEventParser(a, t)
	}
	dp.memeEventParserFactories[constants.DEX_PROGRAMS.RAYDIUM_LCP.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData) parsers.EventParser {
		return raydium.NewRaydiumLaunchpadEventParser(a, t)
	}
	dp.memeEventParserFactories[constants.DEX_PROGRAMS.METEORA_DBC.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData) parsers.EventParser {
		return meteora.NewMeteoraDBCEventParser(a, t)
	}
	dp.memeEventParserFactories[constants.DEX_PROGRAMS.BOOP_FUN.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData) parsers.EventParser {
		return meme.NewBoopfunEventParser(a, t)
	}
	dp.memeEventParserFactories[constants.DEX_PROGRAMS.SUGAR.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData) parsers.EventParser {
		return meme.NewSugarEventParser(a, t)
	}
	dp.memeEventParserFactories[constants.DEX_PROGRAMS.HEAVEN.ID] = func(a *adapter.TransactionAdapter, t map[string][]types.TransferData) parsers.EventParser {
		return meme.NewHeavenEventParser(a, t)
	}
}

// RegisterTradeParser registers a trade parser for a program ID
func (dp *DexParser) RegisterTradeParser(programId string, factory TradeParserFactory) {
	dp.tradeParserFactories[programId] = factory
}

// RegisterLiquidityParser registers a liquidity parser for a program ID
func (dp *DexParser) RegisterLiquidityParser(programId string, factory LiquidityParserFactory) {
	dp.liquidityParserFactories[programId] = factory
}

// RegisterTransferParser registers a transfer parser for a program ID
func (dp *DexParser) RegisterTransferParser(programId string, factory TransferParserFactory) {
	dp.transferParserFactories[programId] = factory
}

// RegisterMemeEventParser registers a meme event parser for a program ID
func (dp *DexParser) RegisterMemeEventParser(programId string, factory MemeEventParserFactory) {
	dp.memeEventParserFactories[programId] = factory
}

// ParseTrades parses trades from a transaction
func (dp *DexParser) ParseTrades(tx *adapter.SolanaTransaction, config *types.ParseConfig) []types.TradeInfo {
	result := dp.parseWithClassifier(tx, config, "trades")
	return result.Trades
}

// ParseLiquidity parses liquidity events from a transaction
func (dp *DexParser) ParseLiquidity(tx *adapter.SolanaTransaction, config *types.ParseConfig) []types.PoolEvent {
	result := dp.parseWithClassifier(tx, config, "liquidity")
	return result.Liquidities
}

// ParseTransfers parses transfers from a transaction
func (dp *DexParser) ParseTransfers(tx *adapter.SolanaTransaction, config *types.ParseConfig) []types.TransferData {
	result := dp.parseWithClassifier(tx, config, "transfer")
	return result.Transfers
}

// ParseAll parses all data from a transaction
func (dp *DexParser) ParseAll(tx *adapter.SolanaTransaction, config *types.ParseConfig) *types.ParseResult {
	return dp.parseWithClassifier(tx, config, "all")
}

// ParseBatch parses multiple transactions concurrently
// maxWorkers: maximum number of concurrent workers, if <= 1, will use sequential processing
func (dp *DexParser) ParseBatch(txs []*adapter.SolanaTransaction, config *types.ParseConfig, maxWorkers int) []*types.ParseResult {
	return dp.ParseBatchWithCallback(txs, config, maxWorkers, nil)
}

// ParseBatchWithCallback parses multiple transactions with callback support
// maxWorkers: maximum number of concurrent workers, if <= 1, will use sequential processing
// callback: optional callback function called for each completed transaction
func (dp *DexParser) ParseBatchWithCallback(
	txs []*adapter.SolanaTransaction,
	config *types.ParseConfig,
	maxWorkers int,
	callback ParseCallback,
) []*types.ParseResult {
	if len(txs) == 0 {
		return []*types.ParseResult{}
	}

	// Optimize for single worker case
	if maxWorkers <= 1 {
		return dp.parseSequentiallyWithCallback(txs, config, callback)
	}

	return dp.parseConcurrentlyWithCallback(txs, config, maxWorkers, callback)
}

// parseSequentiallyWithCallback processes transactions one by one with callback
func (dp *DexParser) parseSequentiallyWithCallback(
	txs []*adapter.SolanaTransaction,
	config *types.ParseConfig,
	callback ParseCallback,
) []*types.ParseResult {
	results := make([]*types.ParseResult, len(txs))

	for i, tx := range txs {
		var result *types.ParseResult
		var err error

		// Handle panic gracefully
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic in transaction %d: %v", i, r)
					result = types.NewParseResult()
					result.State = false
					result.Msg = fmt.Sprintf("panic: %v", r)
				}
			}()

			result = dp.ParseAll(tx, config)
		}()

		results[i] = result

		// Call callback function if provided
		if callback != nil {
			if !callback(i, tx, result, err) {
				// Early termination requested
				break
			}
		}
	}

	return results
}

// parseConcurrentlyWithCallback processes transactions using goroutines with callback
func (dp *DexParser) parseConcurrentlyWithCallback(
	txs []*adapter.SolanaTransaction,
	config *types.ParseConfig,
	maxWorkers int,
	callback ParseCallback,
) []*types.ParseResult {
	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var callbackMu sync.Mutex

	// Pre-allocate results slice
	results := make([]*types.ParseResult, len(txs))
	var shouldStop bool

	for i, tx := range txs {
		wg.Add(1)
		go func(index int, transaction *adapter.SolanaTransaction) {
			defer wg.Done()

			// Check if we should stop early
			callbackMu.Lock()
			if shouldStop {
				callbackMu.Unlock()
				return
			}
			callbackMu.Unlock()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var result *types.ParseResult
			var err error

			// Handle panic gracefully
			func() {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("panic in transaction %d: %v", index, r)
						result = types.NewParseResult()
						result.State = false
						result.Msg = fmt.Sprintf("panic: %v", r)
					}
				}()

				result = dp.ParseAll(transaction, config)
			}()

			// Store result
			mu.Lock()
			results[index] = result
			mu.Unlock()

			// Call callback function if provided
			if callback != nil {
				callbackMu.Lock()
				if !shouldStop {
					if !callback(index, transaction, result, err) {
						// Early termination requested
						shouldStop = true
					}
				}
				callbackMu.Unlock()
			}
		}(i, tx)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return results
}

// parseWithClassifier is the main parsing logic
func (dp *DexParser) parseWithClassifier(tx *adapter.SolanaTransaction, config *types.ParseConfig, parseType string) *types.ParseResult {
	if config == nil {
		defaultConfig := types.DefaultParseConfig()
		config = &defaultConfig
	}

	result := types.NewParseResult()
	result.Slot = tx.Slot

	defer func() {
		if r := recover(); r != nil {
			if config.ThrowError {
				panic(r)
			}
			result.State = false
			result.Msg = fmt.Sprintf("Parse error: %v", r)
		}
	}()

	adapt := adapter.NewTransactionAdapter(tx, config)
	txUtils := utils.NewTransactionUtils(adapt)
	instrClassifier := classifier.NewInstructionClassifier(adapt)

	// Get DEX information
	dexInfo := txUtils.GetDexInfo(instrClassifier)
	allProgramIds := instrClassifier.GetAllProgramIds()

	result.Timestamp = adapt.BlockTime()
	result.Signature = adapt.Signature()
	result.Signer = adapt.Signers()
	result.ComputeUnits = adapt.ComputeUnits()
	result.TxStatus = adapt.TxStatus()

	// Check program ID filter
	if len(config.ProgramIds) > 0 {
		found := false
		for _, configProgramId := range config.ProgramIds {
			for _, programId := range allProgramIds {
				if configProgramId == programId {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			result.State = false
			result.Msg = "No matching program ids"
			return result
		}
	}

	// Check account include filter
	if len(config.AccountInclude) > 0 {
		found := false
		for _, includeAccount := range config.AccountInclude {
			for _, accountKey := range adapt.AccountKeys {
				if includeAccount == accountKey {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			result.State = false
			result.Msg = "No matching accounts include"
			return result
		}
	}

	// Check account exclude filter
	if len(config.AccountExclude) > 0 {
		for _, excludeAccount := range config.AccountExclude {
			for _, accountKey := range adapt.AccountKeys {
				if excludeAccount == accountKey {
					result.State = false
					result.Msg = "Account excluded"
					return result
				}
			}
		}
	}

	// Get transfer actions
	transferActions := txUtils.GetTransferActions([]string{"mintTo", "burn", "mintToChecked", "burnChecked"})

	// Process fee
	result.Fee = adapt.Fee()

	// Process balance changes
	result.SolBalanceChange = adapt.GetAccountSolBalanceChanges(false)[adapt.Signer()]
	tokenChanges := adapt.GetAccountTokenBalanceChanges(true)
	if userTokenChanges, ok := tokenChanges[adapt.Signer()]; ok {
		result.TokenBalanceChange = userTokenChanges
	}

	// Determine what to parse based on parseType and config.ParseType
	// Use GetEffectiveParseType for backward compatibility (defaults to all if not set)
	effectiveParseType := config.GetEffectiveParseType()
	shouldParseTrades := parseType == "trades" || (parseType == "all" && effectiveParseType.Trade)
	shouldParseLiquidity := parseType == "liquidity" || (parseType == "all" && effectiveParseType.Liquidity)
	shouldParseTransfers := parseType == "transfer" || (parseType == "all" && effectiveParseType.Transfer)
	shouldParseMemeEvents := parseType == "all" && effectiveParseType.MemeEvent
	shouldParseAltEvents := parseType == "all" && effectiveParseType.AltEvent

	// Try Jupiter-specific parsing first
	jupiterProgramIds := []string{
		constants.DEX_PROGRAMS.JUPITER.ID,
		constants.DEX_PROGRAMS.JUPITER_DCA.ID,
		constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER1.ID,
		constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER2.ID,
		constants.DEX_PROGRAMS.JUPITER_DCA_KEEPER3.ID,
		constants.DEX_PROGRAMS.JUPITER_VA.ID,
		constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID,
	}

	if dexInfo.ProgramId != "" && containsString(jupiterProgramIds, dexInfo.ProgramId) {
		if shouldParseTrades {
			jupiterInstructions := instrClassifier.GetInstructions(dexInfo.ProgramId)
			if factory, ok := dp.tradeParserFactories[dexInfo.ProgramId]; ok {
				dexInfoWithAMM := types.DexInfo{
					ProgramId: dexInfo.ProgramId,
					AMM:       constants.GetProgramName(dexInfo.ProgramId),
					Route:     dexInfo.Route,
				}
				parser := factory(adapt, dexInfoWithAMM, transferActions, jupiterInstructions)
				trades := parser.ProcessTrades()
				if len(trades) > 0 {
					shouldAggregate := config.ShouldAggregateTrades() || effectiveParseType.AggregateTrade
					if shouldAggregate {
						aggregateTrade := utils.GetFinalSwap(trades, &dexInfo)
						if aggregateTrade != nil {
							result.AggregateTrade = txUtils.AttachTradeFee(aggregateTrade)
						}
					} else {
						result.Trades = append(result.Trades, trades...)
					}
				}
			}
		}
		if len(result.Trades) > 0 || result.AggregateTrade != nil {
			return result
		}
	}

	// Process instructions for each program
	for _, programId := range allProgramIds {
		// Check program ID filters
		if len(config.ProgramIds) > 0 && !containsString(config.ProgramIds, programId) {
			continue
		}
		if len(config.IgnoreProgramIds) > 0 && containsString(config.IgnoreProgramIds, programId) {
			continue
		}

		classifiedInstructions := instrClassifier.GetInstructions(programId)

		// Process trades
		if shouldParseTrades {
			if factory, ok := dp.tradeParserFactories[programId]; ok {
				dexInfoForProgram := types.DexInfo{
					ProgramId: programId,
					AMM:       constants.GetProgramName(programId),
					Route:     dexInfo.Route,
				}
				parser := factory(adapt, dexInfoForProgram, transferActions, classifiedInstructions)
				result.Trades = append(result.Trades, parser.ProcessTrades()...)
			} else if config.TryUnknownDEX {
				// Try to parse unknown DEX programs
				for key, transfers := range transferActions {
					if len(transfers) >= 2 && keyStartsWith(key, programId) {
						hasSupported := false
						for _, t := range transfers {
							if adapt.IsSupportedToken(t.Info.Mint) {
								hasSupported = true
								break
							}
						}
						if hasSupported {
							dexInfoForProgram := types.DexInfo{
								ProgramId: programId,
								AMM:       constants.GetProgramName(programId),
								Route:     dexInfo.Route,
							}
							trade := txUtils.ProcessSwapData(transfers, dexInfoForProgram, true)
							if trade != nil {
								result.Trades = append(result.Trades, *txUtils.AttachTokenTransferInfo(trade, transferActions))
							}
						}
					}
				}
			}
		}

		// Process liquidity
		if shouldParseLiquidity {
			if factory, ok := dp.liquidityParserFactories[programId]; ok {
				parser := factory(adapt, transferActions, classifiedInstructions)
				liquidities := parser.ProcessLiquidity()
				result.Liquidities = append(result.Liquidities, txUtils.AttachUserBalanceToLPs(liquidities)...)
			}
		}

		// Process meme events
		if shouldParseMemeEvents {
			if factory, ok := dp.memeEventParserFactories[programId]; ok {
				parser := factory(adapt, transferActions)
				result.MemeEvents = append(result.MemeEvents, parser.ProcessEvents()...)
			}
		}
	}

	// Process ALT events
	if shouldParseAltEvents {
		altInstructions := instrClassifier.GetInstructions(constants.ALT_PROGRAM_ID)
		if len(altInstructions) > 0 {
			altParser := alt.NewAltEventParser(adapt, altInstructions)
			result.AltEvents = append(result.AltEvents, altParser.ProcessEvents()...)
		}
	}

	// Deduplicate trades
	if len(result.Trades) > 0 {
		result.Trades = deduplicateTrades(result.Trades)
		shouldAggregate := config.ShouldAggregateTrades() || effectiveParseType.AggregateTrade
		if shouldAggregate {
			aggregateTrade := utils.GetFinalSwap(result.Trades, &dexInfo)
			if aggregateTrade != nil {
				result.AggregateTrade = txUtils.AttachTradeFee(aggregateTrade)
			}
		}
	}

	// Process transfers if no trades and no liquidity
	if len(result.Trades) == 0 && len(result.Liquidities) == 0 {
		if shouldParseTransfers {
			if dexInfo.ProgramId != "" {
				classifiedInstructions := instrClassifier.GetInstructions(dexInfo.ProgramId)
				if factory, ok := dp.transferParserFactories[dexInfo.ProgramId]; ok {
					parser := factory(adapt, dexInfo, transferActions, classifiedInstructions)
					result.Transfers = append(result.Transfers, parser.ProcessTransfers()...)
				}
			}
			if len(result.Transfers) == 0 {
				// Add all transfers
				for _, transfers := range transferActions {
					result.Transfers = append(result.Transfers, transfers...)
				}
			}
		}
	}

	return result
}

// Helper functions

func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func keyStartsWith(key, prefix string) bool {
	return len(key) >= len(prefix) && key[:len(prefix)] == prefix
}

func deduplicateTrades(trades []types.TradeInfo) []types.TradeInfo {
	seen := make(map[string]bool, len(trades))
	result := make([]types.TradeInfo, 0, len(trades))
	for _, trade := range trades {
		key := utils.FormatDedupeKey(trade.Idx, trade.Signature)
		if !seen[key] {
			seen[key] = true
			result = append(result, trade)
		}
	}
	return result
}
