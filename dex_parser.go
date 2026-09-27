package dexparser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
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
	dp.tradeParserFactories[constants.DEX_PROGRAMS.JUPITER_Z.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return jupiter.NewJupiterZParser(a, d, t, c)
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
	dp.tradeParserFactories[constants.DEX_PROGRAMS.SOLFI_V2.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewSolFiV2Parser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.GOONFI_V2.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewGoonFiV2Parser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.BISONFI.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewBisonFiParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.TESSERA_V.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewTesseraVParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.ALPHAQ.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewAlphaQParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.ZERO_FI.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewZeroFiParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.SCORCH.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewScorchParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.QUANTUM.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewQuantumParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.MANIFEST.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewManifestParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.BYREAL.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewByrealParser(a, d, t, c)
	}
	dp.tradeParserFactories[constants.DEX_PROGRAMS.SAROS_DLMM.ID] = func(a *adapter.TransactionAdapter, d types.DexInfo, t map[string][]types.TransferData, c []types.ClassifiedInstruction) parsers.TradeParser {
		return propamm.NewSarosDLMMParser(a, d, t, c)
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
	if result == nil {
		return nil
	}
	return result.Trades
}

// ParseLiquidity parses liquidity events from a transaction
func (dp *DexParser) ParseLiquidity(tx *adapter.SolanaTransaction, config *types.ParseConfig) []types.PoolEvent {
	result := dp.parseWithClassifier(tx, config, "liquidity")
	if result == nil {
		return nil
	}
	return result.Liquidities
}

// ParseTransfers parses transfers from a transaction
func (dp *DexParser) ParseTransfers(tx *adapter.SolanaTransaction, config *types.ParseConfig) []types.TransferData {
	result := dp.parseWithClassifier(tx, config, "transfer")
	if result == nil {
		return nil
	}
	return result.Transfers
}

// ParseAll parses all data from a transaction. It never returns nil: parse
// errors (including a nil tx) yield State=false and Msg, unless
// config.ThrowError is set, in which case the panic propagates.
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

// jupiterOrderProgramIds lists the Jupiter order programs (DCA/Recurring,
// Value Average, Limit v2/Trigger). A keeper fills their orders by routing
// through Jupiter v6 in the same transaction.
var jupiterOrderProgramIds = []string{
	constants.DEX_PROGRAMS.JUPITER_DCA.ID,
	constants.DEX_PROGRAMS.JUPITER_VA.ID,
	constants.DEX_PROGRAMS.JUPITER_LIMIT_ORDER_V2.ID,
}

// jupiterProgramIds lists the Jupiter programs whose trade parsers are
// authoritative for the outer instructions they cover: the order programs
// first, then the v6 aggregator
var jupiterProgramIds = append(append([]string{}, jupiterOrderProgramIds...), constants.DEX_PROGRAMS.JUPITER.ID)

// parseWithClassifier is the main parsing logic.
//
// Trades: Jupiter parsers (v6, DCA, VA, Limit v2) run first whenever their
// program appears in the transaction; their trades are authoritative for the
// outer instructions they cover, so the other trade parsers and the
// unknown-DEX fallback skip those outer instructions. When an order program
// (DCA, VA, Limit v2) reports a fill, the Jupiter v6 route in the same
// transaction is the keeper's execution of that fill: its trades are dropped
// and its outer instructions stay covered. Trades are returned in
// execution order (numeric idx) with duplicates of the same idx removed (first
// kept). The aggregate trade is computed in addition to the individual trades.
// Liquidity, meme, ALT events and transfers are always processed for the whole
// transaction. ProgramIds/IgnoreProgramIds apply to every program.
func (dp *DexParser) parseWithClassifier(tx *adapter.SolanaTransaction, config *types.ParseConfig, parseType string) (result *types.ParseResult) {
	if config == nil {
		defaultConfig := types.DefaultParseConfig()
		config = &defaultConfig
	}

	result = types.NewParseResult()

	defer func() {
		if r := recover(); r != nil {
			if config.ThrowError {
				panic(r)
			}
			result.State = false
			result.Msg = fmt.Sprintf("Parse error: %s %v", txSignature(tx), r)
		}
	}()

	if tx == nil {
		result.State = false
		result.Msg = "nil transaction"
		return result
	}
	result.Slot = tx.Slot

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
	result.Warnings = adapt.Warnings()

	// Check program ID filter
	if len(config.ProgramIds) > 0 {
		found := false
		for _, configProgramId := range config.ProgramIds {
			if containsString(allProgramIds, configProgramId) {
				found = true
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
			if adapt.GetAccountIndex(includeAccount) >= 0 {
				found = true
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
	for _, excludeAccount := range config.AccountExclude {
		if adapt.GetAccountIndex(excludeAccount) >= 0 {
			result.State = false
			result.Msg = "Account excluded"
			return result
		}
	}

	// Process fee
	result.Fee = adapt.Fee()

	// Process balance changes (copies: the adapter's maps are cached and shared)
	result.SolBalanceChange = adapt.GetAccountSolBalanceChanges(false)[adapt.Signer()].Copy()
	tokenChanges := adapt.GetAccountTokenBalanceChanges(true)
	if userTokenChanges, ok := tokenChanges[adapt.Signer()]; ok {
		result.TokenBalanceChange = make(map[string]*types.BalanceChange, len(userTokenChanges))
		for mint, change := range userTokenChanges {
			result.TokenBalanceChange[mint] = change.Copy()
		}
	}

	// A failed transaction reverted everything its instructions did
	if result.TxStatus == types.TransactionStatusFailed && !config.IncludeFailedTxs {
		result.Msg = "transaction failed"
		return result
	}

	// Get transfer actions
	transferActions := txUtils.GetTransferActions([]string{"mintTo", "burn", "mintToChecked", "burnChecked"})
	result.Tip = utils.GetTipTotal(transferActions)

	// Determine what to parse based on parseType and config.ParseType
	effectiveParseType := config.GetEffectiveParseType()
	returnTrades := parseType == "trades" || (parseType == "all" && effectiveParseType.Trade)
	shouldAggregate := parseType == "all" && effectiveParseType.AggregateTrade
	shouldParseTrades := returnTrades || shouldAggregate
	shouldParseLiquidity := parseType == "liquidity" || (parseType == "all" && effectiveParseType.Liquidity)
	shouldParseTransfers := parseType == "transfer" || (parseType == "all" && effectiveParseType.Transfer)
	shouldParseMemeEvents := parseType == "all" && effectiveParseType.MemeEvent
	shouldParseAltEvents := parseType == "all" && effectiveParseType.AltEvent

	programAllowed := func(programId string) bool {
		if len(config.ProgramIds) > 0 && !containsString(config.ProgramIds, programId) {
			return false
		}
		return !containsString(config.IgnoreProgramIds, programId)
	}
	dexInfoFor := func(programId string) types.DexInfo {
		return types.DexInfo{
			ProgramId: programId,
			AMM:       constants.GetProgramName(programId),
			Route:     dexInfo.Route,
		}
	}

	var trades []types.TradeInfo

	// Jupiter parsers first, order programs before v6. coveredBy maps an
	// outer instruction index to the Jupiter program whose trades cover it; the
	// first program to cover an outer instruction keeps it.
	coveredBy := make(map[int]string)
	if shouldParseTrades {
		orderFilled := false
		for _, programId := range jupiterProgramIds {
			if !instrClassifier.HasProgram(programId) || !programAllowed(programId) {
				continue
			}
			factory, ok := dp.tradeParserFactories[programId]
			if !ok {
				continue
			}
			isOrderProgram := containsString(jupiterOrderProgramIds, programId)
			parser := factory(adapt, dexInfoFor(programId), transferActions, instrClassifier.GetInstructions(programId))
			for _, trade := range parser.ProcessTrades() {
				outer := outerIndexOf(trade.Idx)
				if owner, ok := coveredBy[outer]; ok && owner != programId {
					continue
				}
				coveredBy[outer] = programId
				if isOrderProgram {
					orderFilled = true
				} else if orderFilled {
					continue // keeper route of an order fill
				}
				trades = append(trades, trade)
			}
		}
	}
	isCovered := func(outer int) bool {
		_, ok := coveredBy[outer]
		return ok
	}

	// Process instructions for each program
	for _, programId := range allProgramIds {
		if !programAllowed(programId) {
			continue
		}

		classifiedInstructions := instrClassifier.GetInstructions(programId)

		// Process trades
		if shouldParseTrades && !containsString(jupiterProgramIds, programId) {
			if factory, ok := dp.tradeParserFactories[programId]; ok {
				instructions := classifiedInstructions
				if len(coveredBy) > 0 {
					instructions = make([]types.ClassifiedInstruction, 0, len(classifiedInstructions))
					for _, ci := range classifiedInstructions {
						if !isCovered(ci.OuterIndex) {
							instructions = append(instructions, ci)
						}
					}
				}
				if len(instructions) > 0 {
					parser := factory(adapt, dexInfoFor(programId), transferActions, instructions)
					for _, trade := range parser.ProcessTrades() {
						if !isCovered(outerIndexOf(trade.Idx)) {
							trades = append(trades, trade)
						}
					}
				}
			} else if config.TryUnknownDEX {
				// Try to parse unknown DEX programs from their transfer groups
				prefix := programId + ":"
				for _, key := range utils.SortedTransferKeys(transferActions) {
					if !strings.HasPrefix(key, prefix) || isCovered(outerIndexOf(key[len(prefix):])) {
						continue
					}
					transfers := transferActions[key]
					if len(transfers) < 2 {
						continue
					}
					// A SOL or stablecoin leg is required; native SOL
					// transfers do not count because ProcessSwapData skips them
					hasSupported := false
					for _, t := range transfers {
						if t.ProgramId != constants.SYSTEM_PROGRAM_ID && adapt.IsSupportedToken(t.Info.Mint) {
							hasSupported = true
							break
						}
					}
					if !hasSupported {
						continue
					}
					trade := txUtils.ProcessSwapData(transfers, dexInfoFor(programId), true)
					if trade != nil {
						trades = append(trades, *txUtils.AttachTokenTransferInfo(trade, transferActions))
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
	sortByIdx(result.Liquidities, func(e types.PoolEvent) string { return e.Idx })
	sortByIdx(result.MemeEvents, func(e types.MemeEvent) string { return e.Idx })

	// Process ALT events
	if shouldParseAltEvents && programAllowed(constants.ALT_PROGRAM_ID) {
		altInstructions := instrClassifier.GetInstructions(constants.ALT_PROGRAM_ID)
		if len(altInstructions) > 0 {
			altParser := alt.NewAltEventParser(adapt, altInstructions)
			result.AltEvents = append(result.AltEvents, altParser.ProcessEvents()...)
		}
	}

	// Trades in execution order, duplicates removed
	if len(trades) > 0 {
		trades = deduplicateTrades(utils.SortTradesByIdx(trades))
		allTransfers := utils.SortedTransfers(transferActions)
		for i := range trades {
			txUtils.ApplyToken2022TransferFee(&trades[i], allTransfers)
		}
		if returnTrades {
			result.Trades = trades
		}
		if shouldAggregate {
			aggregateTrade := utils.GetFinalSwap(trades, &dexInfo)
			if aggregateTrade != nil {
				result.AggregateTrade = txUtils.AttachTradeFee(aggregateTrade)
			}
		}
	}

	// Process transfers if no trades and no liquidity
	if len(trades) == 0 && len(result.Liquidities) == 0 && shouldParseTransfers {
		if dexInfo.ProgramId != "" && programAllowed(dexInfo.ProgramId) {
			if factory, ok := dp.transferParserFactories[dexInfo.ProgramId]; ok {
				classifiedInstructions := instrClassifier.GetInstructions(dexInfo.ProgramId)
				parser := factory(adapt, dexInfo, transferActions, classifiedInstructions)
				result.Transfers = append(result.Transfers, parser.ProcessTransfers()...)
			}
		}
		if len(result.Transfers) == 0 {
			// Add all transfers, in execution order, except those grouped
			// under a program excluded by ProgramIds/IgnoreProgramIds
			allowed := transferActions
			if len(config.ProgramIds) > 0 || len(config.IgnoreProgramIds) > 0 {
				allowed = make(map[string][]types.TransferData, len(transferActions))
				for key, transfers := range transferActions {
					if i := strings.IndexByte(key, ':'); i < 0 || programAllowed(key[:i]) {
						allowed[key] = transfers
					}
				}
			}
			result.Transfers = append(result.Transfers, utils.SortedTransfers(allowed)...)
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

// outerIndexOf returns the outer instruction index of an idx ("5" or "5-3"),
// or -1 when it cannot be parsed
func outerIndexOf(idx string) int {
	if i := strings.IndexByte(idx, '-'); i >= 0 {
		idx = idx[:i]
	}
	n, err := strconv.Atoi(idx)
	if err != nil {
		return -1
	}
	return n
}

// txSignature returns the first signature of tx for error messages
func txSignature(tx *adapter.SolanaTransaction) string {
	if tx == nil || len(tx.Transaction.Signatures) == 0 {
		return ""
	}
	return tx.Transaction.Signatures[0]
}

// sortByIdx sorts events stably by numeric idx (execution order)
func sortByIdx[T any](items []T, idx func(T) string) {
	sort.SliceStable(items, func(i, j int) bool {
		return utils.CompareIdx(idx(items[i]), idx(items[j])) < 0
	})
}

// deduplicateTrades removes trades with the same idx and signature, keeping
// the first occurrence
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
