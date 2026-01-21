# Supported Protocols

## DEX Aggregators & Routers

| Protocol | Trades | Liquidity | Transfers | Notes |
|----------|:------:|:---------:|:---------:|-------|
| **Jupiter** (All versions) | ✅ | ❌ | ✅ | Priority parsing, aggregated trades |
| **OKX DEX** | ✅ | ❌ | ✅ | Route aggregator |

## Major AMMs

| Protocol | Trades | Liquidity | Transfers | Notes |
|----------|:------:|:---------:|:---------:|-------|
| **PumpSwap** | ✅ | ✅ | ✅ | Pumpfun AMM |
| **Raydium V4** | ✅ | ✅ | ✅ | Classic AMM |
| **Raydium CPMM** | ✅ | ✅ | ✅ | Constant product |
| **Raydium CL** | ✅ | ✅ | ✅ | Concentrated liquidity |
| **Orca Whirlpool** | ✅ | ✅ | ✅ | CL pools |
| **Meteora DLMM** | ✅ | ✅ | ✅ | Dynamic liquidity |
| **Meteora Pools** | ✅ | ✅ | ✅ | Multi-token AMM |
| **Meteora DAMM V2** | ✅ | ✅ | ✅ | Dynamic AMM |
| **Sanctum** | ✅ | ❌ | ✅ | LST swaps |
| **Phoenix** | ✅ | ❌ | ✅ | Order book DEX |
| **Lifinity** | ✅ | ❌ | ✅ | Proactive market maker |

## Meme & Launch Platforms

| Protocol | Trades | Create | Migrate | Notes |
|----------|:------:|:------:|:-------:|-------|
| **Pumpfun** | ✅ | ✅ | ✅ | Bonding curve |
| **Raydium Launchpad** | ✅ | ✅ | ✅ | Meme launcher |
| **Meteora DBC** | ✅ | ✅ | ✅ | Meme launcher |
| **Moonit** | ✅ | ✅ | ✅ | Meme launcher |
| **Heaven.xyz** | ✅ | ✅ | ✅ | Meme launcher |
| **Sugar.money** | ✅ | ✅ | ✅ | Meme launcher |
| **Bonk** | ✅ | ✅ | ✅ | Meme launcher |
| **BoopFun** | ✅ | ✅ | ✅ | Meme launcher |

## Trading Bots

| Bot | Trades | Notes |
|-----|:------:|-------|
| **BananaGun** | ✅ | MEV bot |
| **Maestro** | ✅ | Trading bot |
| **Nova** | ✅ | Sniper bot |
| **Bloom** | ✅ | Copy trading |
| **Mintech** | ✅ | Trading bot |
| **Apepro** | ✅ | Trading bot |

## Program IDs

All program IDs are available in `constants/programs.go`:

```go
import "github.com/solana-dex-parser-go/constants"

// Access program IDs
jupiterV6 := constants.DEX_PROGRAMS.JUPITER_V6.ID
raydiumV4 := constants.DEX_PROGRAMS.RAYDIUM_V4.ID
pumpfun := constants.DEX_PROGRAMS.PUMP_FUN.ID
```
