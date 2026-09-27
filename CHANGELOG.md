# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Behaviour changes

Read these before upgrading; results change for the same transaction.

- **Failed transactions** (meta.err set) no longer report trades, aggregate, liquidity, meme, ALT events, transfers or tip. Fee, signer, balance changes and `TxStatus=failed` are still filled, with `State=true` and `Msg="transaction failed"`. Set `ParseConfig.IncludeFailedTxs` to parse them as before. `ShredParser` skips them the same way.
- **Trades and aggregation**: `Trades` always holds the individual trades; `AggregateTrade` is computed in addition. With `ParseType` unset, `AggregateTrades: false` now really disables aggregation (a nil config still aggregates). `ParseType{AggregateTrade: true}` alone returns only the aggregate.
- **Jupiter** (v6, DCA, VA, Limit v1/v2, Jupiter Z) no longer returns early: liquidity, meme events, ALT events and transfers are kept. Jupiter trades cover their instructions, so nested AMM trades are not duplicated; when Jupiter runs by CPI inside another program only its own subtree is covered. `ProgramIds`/`IgnoreProgramIds` also filter Jupiter.
- **Jupiter v6 `*_v2` routes** (about half of current Jupiter traffic) give one Jupiter trade per hop from SwapsEvent (`ProgramId` JUP6, `AMM` = hop venue) instead of per-venue and unknown-DEX trades. A platform fee (FeeEvent) is a `platform` fee on the first or last hop.
- **Jupiter hops** (legacy and `*_v2` routes) take `Pool`, `Type` and the venue's fees from the venue's own parser run on the hop's AMM instruction (`JupiterParser.HopInstruction`); the amounts stay the route event's, and a venue that reports other amounts adds a line to `ParseResult.Warnings`. LaunchLab hops quoted in other tokens are BUY where the launchpad says so (they were SELL).
- **Titan and OKX DEX Router V2 routes**: `AggregateTrade` is the aggregator's route total, what the user sent and received, with `ProgramId`/`Route` the aggregator and its fee in `Fee` (Titan: Type `platform`, Dex `Titan`, Recipient the owner of the receiving account; OKX: Type `commission`, Dex `OKXV2`). `Trades` stay the venue hops, and the unknown-DEX fallback skips the route programs.
- **Jupiter hop user** is the route's `user_transfer_authority` (the token owner) instead of the fee payer: gasless and relayed routes report the real user. Keeper routes in DCA transactions keep the old choice.
- **Program order and DexInfo**: programs are taken in first-appearance order (outer, then inner). `DexInfo` is the first known DEX program that is not a vault or wallet, so `Route`/`AMM` labels can change: MeteoraVault and HeavenStore are never routes, and a router or bot program that executes first becomes the route.
- **Program events** (Anchor self-CPI events, `Program data` logs) belong to their stack-height parent instruction; the nearest-preceding rule is kept only for transactions without stack heights. Program logs are framed only on runtime invoke/success/failed lines and stop at a CPI that does not match the inner instructions.
- **Idx**: an outer instruction is `"N"` (meme events, Jupiter transfers and shred events used `"N-0"`); inner ones are `"N-M"`. Trades, liquidity and meme events, transfers and shred instructions come out sorted numerically by idx, and output is deterministic.
- **Fees**: `Fee`/`Fees` hold only fees reported by program events or paid by transfers flagged as fees. The "output minus balance change" estimate, the Raydium "third transfer is the fee" rule and the fixed 0.1% fees on Jupiter DCA hops and Limit v2 fills are gone. `AggregateTrade.Fees` lists each fee once: a `Fee` that is only the total of its `Fees` (Pump.fun, PumpSwap, LaunchLab, DBC, meme parsers), or that `Fees` already holds (Titan, OKX), is not listed again.
- **Trades come only from swap instructions** for Orca, Raydium (V4, CPMM, CLMM, route) and Meteora (DLMM, DAMM v1/v2): pool creations, DBC migrations, liquidity, fee and reward instructions no longer produce phantom trades. Amounts come from the programs' events or `ray_log` where they exist (Token-2022 transfer fees included on input, excluded on output).
- **Unknown programs** are named `"Unknown"` (as upstream TS) instead of `""`. The unknown-DEX fallback ignores native SOL (System) transfers and needs a non-native SOL or stablecoin leg, so trades built from rent payments disappear.
- **Bots**: a bot is attributed only when one of its fee accounts, or a token account it owns, receives at least 10000 lamports of SOL/WSOL (or 0.1% of the SOL leg if smaller), or at least 0.1% of a trade leg in that leg's token. Dust no longer tags bots; BONKbot and Fomo fees paid in the traded token or USDC now count.
- **Tips** to relay tip accounts are reported in `ParseResult.Tip`; they are never a trade fee or a swap leg, and transfers to tip accounts are no longer flagged `IsFee` (in `ShredParser` output too).
- **Typed transfers**: Jupiter DCA `WithdrawDca` and `DepositDca`, Limit v1 `cancelExpiredOrder`, Raydium CLMM `openLimitOrder`, `increaseLimitOrder`, `decreaseLimitOrder`, `settleLimitOrder`, Meteora DLMM `placeLimitOrder`, `cancelLimitOrder`, and the Pump.fun/PumpSwap fee payouts `collectCreatorFee`, `collectCoinCreatorFee`, `claimCashback`, `claimTokenIncentives`, `distributeCreatorFees`, `distributeFeeToHolders`, `transferCreatorFeesToPump`. Like the other transfers they are in `ParseTransfers`, and in `ParseAll` only when the transaction has no trades or liquidity events.
- **Token-2022 transfer-fee mints**: the output is the net amount received, and the withheld amount is a `transferFee` entry in `Fees`.
- **Pump.fun** trade amounts are what the user paid or received: buys add the protocol, creator and cashback fees reported by the TradeEvent, sells deduct them, and `Fee`/`Fees` are filled. Buys of the old program version (121-byte TradeEvent, no fee fields) add the user's transfer to the buy's fee_recipient as the `protocol` fee; the matching sells took the fee from the curve's lamports without a transfer, so their fee stays unknown. USDC and custom-quote pairs report the real quote mint and decimals.
- **PumpSwap** buy input is `user_quote_amount_in` (or `quote_amount_in` for buy_exact_quote_in); `Fee` is the exact sum of the fee components and buyback is its own `Fees` entry. boost_buy_and_burn is a `BUY_AND_BURN` meme event, not a BUY. Trade `Type` follows the SOL/stablecoin rule when either side is SOL or a stablecoin (a buy in a USD1- or PYUSD-quoted pool is BUY, it was SELL; USDC -> SOL in a WSOL-base pool is SELL, it was BUY), else the event's side (a buy in a token-quoted pool is BUY, it was SELL); Jupiter hops through PumpSwap copy it.
- **Meteora DBC** trades come from EvtSwap/EvtSwap2: sells are no longer BUY, amounts are actual (not min_out or 0).
- **LaunchLab** uses the real quote decimals (USD1 was 1000x off), an exact fee including share_fee, and `Pool` = pool_state.
- **Heaven, Boop.fun, Moonit, Sugar**: correct users, mints, pools and amounts (before: min_out, 0 or signer balance deltas). Boop tokens have 9 decimals.
- **Prop AMMs**: trades use the swap instruction's idx and always carry `Pool`. HumidiFi counts only its real swap forms and no longer produces empty-mint or reversed trades. Titan- and OKX-routed hops no longer include the aggregator's transfers (a SolFi V2 hop under Titan was reported as 1999694239 USDC instead of 999897114).
- **Liquidity**: `Idx` is `"outer"` or `"outer-inner"` (DLMM kept only the last digit). DLMM claim_fee events report the lb_pair and the token mints (they reported the position and token accounts). Raydium CLMM open_position events are ADD with the right pool.
- **Jupiter DCA, VA, Limit v2**: the `AMM` is the first venue in execution order (it varied between runs). DCA open/close transfers report the order's input mint and amount; VA transfers use the event amounts and include token withdrawals; CloseDca transfers carry the DCA program id. Limit v2 fills are the maker's trade and cancel_dust_order gives transfers.
- **Classification**: the Jupiter DCA keeper wallets and OKX_ROUTER are not DEX programs any more (`constants.KNOWN_AUTHORITIES`). OpenBook `srmqPv…` is a system program (no phantom SELL on Raydium V4 initialize2), SPL Memo and the Scorch pricing program `ojh19oja…` are skipped for transfer grouping, and Scorch hops report the swap program `SCoRcH8c…`. Meteora DLMM add_liquidity2, add_liquidity_by_weight2, add_liquidity_one_side_precise2 and remove_liquidity2 are liquidity events, not swaps. USDY is no longer a stablecoin; USDS, JupUSD, CASH and USDe are.
- **Fee accounts**: the Pump.fun/PumpSwap reserved and buyback fee recipients are flagged `IsFee`; the migrator `39azUY…` is not.
- **Token balance changes**: accounts closed in the transaction report post 0; several accounts of one owner and mint are summed per owner.
- **ShredParser**: mints are never guessed (`""` with decimals 0 when unknown; it defaulted to SOL or a token account), decimals come from the transaction or `TOKEN_DECIMALS` instead of hard-coded 9/6, and the trade type is `SWAP` when the direction cannot be determined (Jupiter and DFlow routes with an unknown or repeated mint). Programs with no decoded instructions are left out of `Instructions`. A Raydium V4 swap whose only known mint is a non-SOL, non-stablecoin input is SELL.
- `raydium.SwapDirectionCoinToPC` is 2 (the raydium-amm value; it was 0).
- `ParseAll` never returns nil; a recovered panic gives `State=false` with the signature in `Msg`.
- Fetcher warnings also redact JSON-escaped URLs and passwords containing `@`; transfer balance pointers are independent copies that do not point into `tx.Meta`.

### Added

- `ParseConfig.IncludeFailedTxs`, `ParseConfig.AddressLookupTables`; working `ALTsFetcher` and `TokenAccountsFetcher`.
- `ParseResult.Tip`, `ParseResult.Warnings` (fetcher errors with URLs cut to scheme and host, unresolved lookup-table accounts).
- Version 1 transactions: `adapter.TransactionMessage.TransactionConfig` decodes `message.transactionConfig`.
- `jsonParsed` input, Node.js Buffer JSON (signatures, keys, data), string slots; TransferCheckedWithFee and Token-2022 instructions in both encodings.
- Jupiter: SwapsEvent/FeeEvent decoding for the `*_v2` routes, a Jupiter Z (RFQ) parser, Limit Order v1 flash fills (historical), cancel_dust_order.
- New venue parsers: SolFi V2, GoonFi V2, BisonFi, TesseraV, AlphaQ, ZeroFi, Scorch, Quantum, Manifest, Byreal and Saros DLMM (`propamm.VenueParser` and `New…Parser` constructors). Route parsers `propamm.TitanParser` and `propamm.OKXV2Parser` decode the aggregators' swap events and are registered with the new `DexParser.RegisterRouteParser`.
- Known programs: `DEX_PROGRAM_IDS` grows from 63 to 150 entries (new named venues and aggregators, 63 venues of the Jupiter label list in `constants.JUPITER_LABEL_PROGRAMS`, bot routers), each confirmed executable; dormant ones are marked legacy. New `constants.KNOWN_AUTHORITIES`, `IsVaultProgram`, `IsKnownAuthority`.
- Bots: 14 bot router programs (`constants.BOT_ROUTER_PROGRAMS`, reported as `Route`) and the GMGN program; 10 more bot fee accounts (72 in total), among them two new bots, Fomo (a USDC fee wallet) and Nova (2 fee wallets).
- `constants.TIP_ACCOUNTS` (17 providers, 150 accounts; Jito verified), `IsTipAccount`, `GetTipProvider`, `IsTradeFeeAccount`.
- Liquidity: Raydium CLMM events (create_pool, create_customizable_pool, open_position variants), Orca decrease_liquidity_v2 and increase_liquidity_by_token_amounts_v2, CPMM initialize_with_permission, DLMM pair creation and rebalance_liquidity, DAMM v1 create variants, bootstrap_liquidity, remove_liquidity_single_side and claim_fee; DAMM v2 amounts from EvtLiquidityChange.
- Trade `Fee`/`Fees` from program events (protocol, creator, host, referral, limitOrder, compounding, partner, transferFee) for Orca, Raydium and Meteora; Orca trades carry the whirlpool(s); CLMM swap_router_base_in reports all pools.
- Meme events: Pump.fun v2 instructions and 160-byte migrate events, Meteora DBC swap2_with_transfer_hook and COMPLETE events, Heaven CREATE events (matched to the Metaplex create), Sugar migrate_to_radium (to Raydium CPMM, pSOL quote), PumpSwap `BUY_AND_BURN`. New `MemeEvent` fields: `IxName`, `TokenProgram`, `IsMayhemMode`, `IsCashbackEnabled`, `IsHolderReward`, `CreatorFeeBps`, `Fees`, virtual/real base/quote reserves, `BaseAmount`, `QuoteAmount`, `MigrationFee`, `Curve`; `Creator` on Pump.fun and PumpSwap trade events. Helpers `utils.SortInstructionsByExecution`, `utils.SumFeeAmounts`, `utils.TotalFee`.
- ShredParser: System, Token and Token-2022 transfers, a DFlow decoder, Pump.fun buy_exact_sol_in, create_v2, buy_v2, sell_v2, buy_exact_quote_in_v2 and migrate_v2, PumpSwap buy_exact_quote_in, Jupiter `*_v2` routes, Raydium V4 tags 11/16/17 and 8-account swaps, LaunchLab initialize_v2 and initialize_with_token_2022, Photon pump_buy_v2/pump_sell_v2/collect_fee, DBC transfer-hook swaps and pools. Typed output for Pump.fun and PumpSwap.
- `ParseShredResult.TxStatus`, `HasUnresolvedAccounts`; `ParsedShredInstruction.InputAmountKind`/`OutputAmountKind` (`exact`, `max`, `min`, `quote`, `unknown`) and `UnresolvedAccounts`; an `unresolvedAccounts` flag on every legacy shred event; `types.HasUnresolvedAccount`. New `JupiterRouteData` fields `PlatformFeeBps`, `PositiveSlippageBps`, `ExactOut`, `InputTokenAccount`, `OutputTokenAccount`; `PumpswapBuyExactQuoteInData`.
- Transfer parsers `raydium.RaydiumCLLimitOrderParser`, `meteora.MeteoraDLMMLimitOrderParser` and `pumpfun.PumpFeeClaimParser` (registered for Raydium CLMM, Meteora DLMM, Pump.fun and PumpSwap), and Jupiter DCA Withdraw/Deposit and Limit v1 cancel_expired_order transfers. `ParseShredResult.Warnings`.
- `DexParser.RegisterRouteParser`, `JupiterParser.HopInstruction`; `utils.FindEventEmitter`, `EmittedEvents`, `CPIGroup`, `SplitIdx`, `ProgramInstructions`, `FeeComponents`, `GetShredTradeType`, `TransactionUtils.CPIGroupTransfers`; `adapter.TransactionAdapter.KnownDecimals`, `KnownTokenAccountMint`; `constants.ANCHOR_EVENT_PREFIX`, `IsAnchorEvent`, `UnknownProgramName`.
- `DecodeShredEntries` decodes Jito ShredStream entries (legacy, v0 and v1 transactions).
- `ConvertGeyserTransaction` and `YellowstoneFromGeyser` accept the generated Yellowstone protobuf types (rpcpool and Helius LaserStream) without importing them; `YellowstoneTransaction` gained `NoMeta`, `Config`, `StackHeight`, flat `Loaded*Addresses` and `TokenBalance.ProgramId`.
- `ClassifiedInstruction.StackHeight`, `CompiledInstruction.StackHeight`, `adapter.GetParentInstruction`, `GetInstructionStackHeight`, `InstructionStackHeight`, `adapter.Warnings()`, `HasUnresolvedAccounts()`.
- `utils`: `FormatTransferKey`, `FormatIdx`, `CompareIdx`, `SortByIdx`, `SortTradesByIdx`, `FindProgramAddress`, `IsOnCurve`, `GetTipTotal`, `BotFeeMinLamports`, `BotFeeMinLegDivisor`, program log decoding (`ParseProgramLogs`, `ParseProgramDataLogs`, `ParseRayLogs`, `FindProgramLogs`, `EventSwap`), `TransactionUtils.GetProgramDataLogs`, `GetRayLogs`, `NewEventTrade`, `NewEventFee`, `AttachInstructionTransfers`. `BinaryReader.ReadVecLength`. `raydium.InitLog`. `types.TokenAmount.Copy`, `types.BalanceChange.Copy`.
- Discriminators and instruction types from the current on-chain IDLs (Jupiter, Pump.fun, PumpSwap, LaunchLab, Meteora, Orca, Raydium, DFlow, Photon, the new venues), SPL Token tags 16-26 and the Token-2022 transfer-fee sub-instructions.
- Tests run offline on about 400 stored mainnet transactions (`testdata/tx`); `SDP_FETCH_FIXTURES=1` fetches missing ones.
- Documentation: compilable examples (`example_test.go`), doc snippets compiled by the tests, a ShredParser page, the parse semantics and a NOTICE file with the MIT notices of solana-dex-parser and sol-parser-sdk.
- From the unreleased commit e9dbb7c: Photon, Padre, PepeBoost, STBot and MevX bot detection, 6 more Trojan and 3 more BananaGun fee accounts.

### Changed

- `propamm.SolFiParser`, `GoonFiParser`, `ObricParser` and `HumidiFiParser` are type aliases of `propamm.VenueParser`; constructors and `ProcessTrades` are unchanged.
- `constants.DEX_PROGRAMS.SCORCH` is the swap program `SCoRcH8c…`; the pricing program is `SCORCH_PRICING` (tag `pricing`, not a DEX program).
- `constants.GetProgramName` returns `"Unknown"` for unknown IDs.
- `MatchDiscriminator` and `MatchAnyDiscriminator` never match an empty discriminator.
- `ShredParser.ParseAll` never returns nil; `ParsedInstructions` span all programs in execution order.
- `ConvertYellowstoneTransaction` output parses exactly like RPC json: base58 keys and signatures, meta.err, loaded addresses, stack heights, v1 compute budget.
- `raydium.DecodeRaydiumLog` decodes InitLog and returns nil for short payloads.
- `BinaryReader` errors are sticky and negative lengths are rejected; untrusted counts no longer drive allocations (ALT ExtendLookupTable lists at most the addresses present).
- `ParseResult.SolBalanceChange`/`TokenBalanceChange` and trade balance pointers are independent copies.
- `ParseAll` allocates 23-43% less on the benchmark fixtures.

### Fixed

- README Quick Start did not compile (`dexparser.SolanaTransaction` does not exist; it is `adapter.SolanaTransaction`). Documentation examples fetched with `maxSupportedTransactionVersion` 0, which fails on version 1 transactions, sliced names and mints with `[:8]` (panics on "Jupiter" or an empty mint), passed the wrong types to the gRPC converter, and stated wrong config defaults.
- `utils.FindAssociatedTokenAddress` derived wrong addresses; `ClassifiedInstruction.GetIdx` was wrong for indices of 10 or more; `types.ConvertToUIAmount(nil)` panicked.
- Pump.fun `ProtocolFee` and `CreatorFee` were garbage (about 1e10 SOL); 121-byte TradeEvents report their real reserves; Pump.fun pools are the real bonding curve for v2 instructions and multi-instruction transactions.
- Sugar sell_exact_out (a 9-byte discriminator never matched), the DLMM RemoveLiquidity event and the Jupiter VA Close/Deposit events had wrong discriminators. System instruction tag 13 was misnamed.
- Jupiter VA withdraw data of 90-97 bytes panicked; ShredParser v1 Jupiter routes read amounts and slippage one byte off, and routes with extra trailing bytes gave garbage.
- Many shred decoders swapped mints or directions (DBC and Photon PumpSwap sells, LaunchLab buy mints, Raydium V4 token accounts reported as mints, Photon Moonit buys), and LaunchLab/Raydium V4 initialize and migrate fields were wrong.
- Meteora DLMM claim_reward/claim_reward2 were reported as REMOVE events; DAMM v1 LP amounts came from the vault LP and fallback amounts were swapped; CPMM create reported the discriminator as LP amount; Meteora liquidity counted native SOL rent transfers.
- Moonit collateral was USDC whenever USDC appeared in the transaction; Moonit trades lost their amounts when the program logged lines containing "failed" or "success".
- Trade legs of hops inside a route showed the previous hop's transfer; Token-2022 outputs with a transfer fee lost their account fields.

### Deprecated

- `ParseConfig.PoolInfoFetcher`, `types.PoolInfoFetcher` and `types.NewPoolInfoFetcher`: no parser uses pool data; the field has no effect.
- `constants.SystemCreateAccountWithSeedChecked` and `SystemCreateIdempotent` (tag 13 is CreateAccountAllowPrefund; tag 14 does not exist).
- PumpSwap create-pool `QuoteAmountOut` (use `QuoteAmountIn`) and the shred `PoolMint` fields (use `Pool`).

### Removed

- The `github.com/joho/godotenv` dependency.

### Dependencies and CI

- `github.com/goccy/go-json` 0.10.5 -> 0.10.6 -> 0.11.1; `github.com/mr-tron/base58` 1.2.0 -> 1.3.0.
- CI tests on Go 1.22 (the go.mod floor), oldstable and stable with `-race`, and a lint job runs `gofmt -l`, `go vet`, staticcheck 2026.2.1 and govulncheck 1.8.0.
- `docs.yml` needs only `contents: write` (Pages deploys from the gh-pages branch).
- From 5fd281c: the test workflow uses Go 1.22, has a least-privilege `contents: read` block and no longer references the dead RPC key.
- GitHub Actions `actions/checkout`, `actions/setup-go` and `actions/setup-python` 6 -> 7 (#6, #7, #8).

## [1.2.0] - 2026-01-22

### Fixed
- Raydium CL Token22 discriminator: moved from CREATE to ADD_LIQUIDITY
- Correct classification of Token22 NFT positions

### Added
- Raydium CL Swap discriminators (SWAP, SWAP_V2, SWAP_ROUTER_BASE_IN)
- Raydium CL CLOSE_POSITION discriminator
- Raydium CL Event discriminators (11 events: COLLECT_PERSONAL_FEE, COLLECT_PROTOCOL_FEE, etc.)

### Changed
- Documentation: removed "port" mentions from headers
- Documentation: updated protocol counts (35→55 routing detection)
- Documentation: expanded ShredParser examples and protocol support

## [1.1.0] - 2026-01-21

### Added
- Full documentation site with GitHub Pages
- Documentation for all APIs, types, and examples
- GitHub Actions workflow for docs deployment
- Test parity with TypeScript (33 integration tests)
- Route detection in trade parsing (BananaGun, Maestro, OKX, Jupiter, Bloom)
- Deterministic program ID ordering in classifier

### Changed
- README restructured with links to docs/
- GetDexInfo() now correctly detects both AMM and Route

### Fixed
- Non-deterministic Route detection due to map iteration order
- Float precision issue in transfer deduplication

### Performance
- JSON parsing: Replaced `encoding/json` with `goccy/go-json` (~2x faster)
- BinaryReader pooling with `sync.Pool` to reduce allocations
- String formatting optimized with `strconv` in hot paths
- Pre-allocation for maps and slices in critical paths

## [1.0.0] - 2026-01-20

### Added
- Initial Go port of `solana-dex-parser` TypeScript library
- Support for 18 DEX trade parsers (Jupiter, Raydium, Meteora, Orca, Pumpfun, etc.)
- Support for 8 liquidity pool parsers
- Support for 8 meme event parsers
- ShredParser for gRPC streams
- 28 integration tests with real Solana transactions
