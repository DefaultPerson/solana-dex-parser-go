# Supported Protocols

## DEX Aggregators & Routers

| Protocol | Trades | Liquidity | Transfers | Notes |
|----------|:------:|:---------:|:---------:|-------|
| **Jupiter** (All versions) | :white_check_mark: | :x: | :white_check_mark: | Priority parsing, aggregated trades |
| **OKX DEX** | :white_check_mark: | :x: | :white_check_mark: | Route aggregator |

## Major AMMs

| Protocol | Trades | Liquidity | Transfers | Notes |
|----------|:------:|:---------:|:---------:|-------|
| **PumpSwap** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Pumpfun AMM |
| **Raydium V4** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Classic AMM |
| **Raydium CPMM** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Constant product |
| **Raydium CL** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Concentrated liquidity |
| **Orca Whirlpool** | :white_check_mark: | :white_check_mark: | :white_check_mark: | CL pools |
| **Meteora DLMM** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Dynamic liquidity |
| **Meteora Pools** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Multi-token AMM |
| **Meteora DAMM V2** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Dynamic AMM |
| **Sanctum** | :white_check_mark: | :x: | :white_check_mark: | LST swaps |
| **Phoenix** | :white_check_mark: | :x: | :white_check_mark: | Order book DEX |
| **Lifinity** | :white_check_mark: | :x: | :white_check_mark: | Proactive market maker |

## Meme & Launch Platforms

| Protocol | Trades | Create | Migrate | Notes |
|----------|:------:|:------:|:-------:|-------|
| **Pumpfun** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Bonding curve |
| **Raydium Launchpad** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Meme launcher |
| **Meteora DBC** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Meme launcher |
| **Moonit** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Meme launcher |
| **Heaven.xyz** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Meme launcher |
| **Sugar.money** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Meme launcher |
| **Bonk** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Meme launcher |
| **BoopFun** | :white_check_mark: | :white_check_mark: | :white_check_mark: | Meme launcher |

## Trading Bots

| Bot | Trades | Notes |
|-----|:------:|-------|
| **BananaGun** | :white_check_mark: | MEV bot |
| **Maestro** | :white_check_mark: | Trading bot |
| **Nova** | :white_check_mark: | Sniper bot |
| **Bloom** | :white_check_mark: | Copy trading |
| **Mintech** | :white_check_mark: | Trading bot |
| **Apepro** | :white_check_mark: | Trading bot |

## Program IDs

All program IDs are available in `constants/programs.go`:

```go
import "github.com/solana-dex-parser-go/constants"

// Access program IDs
jupiterV6 := constants.DEX_PROGRAMS.JUPITER_V6.ID
raydiumV4 := constants.DEX_PROGRAMS.RAYDIUM_V4.ID
pumpfun := constants.DEX_PROGRAMS.PUMP_FUN.ID
```
