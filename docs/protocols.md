---
layout: default
title: Supported Protocols
nav_order: 3
---

# Supported Protocols

## DEX Aggregators & Routers

| Protocol | Trades | Liquidity | Transfers | Notes |
|----------|--------|-----------|-----------|-------|
| **Jupiter** (All versions) | Yes | No | Yes | Priority parsing, aggregated trades |
| **OKX DEX** | Yes | No | Yes | Route aggregator |

## Major AMMs

| Protocol | Trades | Liquidity | Transfers | Notes |
|----------|--------|-----------|-----------|-------|
| **PumpSwap** | Yes | Yes | Yes | Pumpfun AMM |
| **Raydium V4** | Yes | Yes | Yes | Classic AMM |
| **Raydium CPMM** | Yes | Yes | Yes | Constant product |
| **Raydium CL** | Yes | Yes | Yes | Concentrated liquidity |
| **Orca Whirlpool** | Yes | Yes | Yes | CL pools |
| **Meteora DLMM** | Yes | Yes | Yes | Dynamic liquidity |
| **Meteora Pools** | Yes | Yes | Yes | Multi-token AMM |
| **Meteora DAMM V2** | Yes | Yes | Yes | Dynamic AMM |
| **Sanctum** | Yes | No | Yes | LST swaps |
| **Phoenix** | Yes | No | Yes | Order book DEX |
| **Lifinity** | Yes | No | Yes | Proactive market maker |

## Meme & Launch Platforms

| Protocol | Trades | Create | Migrate | Notes |
|----------|--------|--------|---------|-------|
| **Pumpfun** | Yes | Yes | Yes | Bonding curve |
| **Raydium Launchpad** | Yes | Yes | Yes | Meme launcher |
| **Meteora DBC** | Yes | Yes | Yes | Meme launcher |
| **Moonit** | Yes | Yes | Yes | Meme launcher |
| **Heaven.xyz** | Yes | Yes | Yes | Meme launcher |
| **Sugar.money** | Yes | Yes | Yes | Meme launcher |
| **Bonk** | Yes | Yes | Yes | Meme launcher |
| **BoopFun** | Yes | Yes | Yes | Meme launcher |

## Trading Bots

| Bot | Trades | Notes |
|-----|--------|-------|
| **BananaGun** | Yes | MEV bot |
| **Maestro** | Yes | Trading bot |
| **Nova** | Yes | Sniper bot |
| **Bloom** | Yes | Copy trading |
| **Mintech** | Yes | Trading bot |
| **Apepro** | Yes | Trading bot |

## Program IDs

All program IDs are available in `constants/programs.go`:

```go
import "github.com/solana-dex-parser-go/constants"

// Access program IDs
jupiterV6 := constants.DEX_PROGRAMS.JUPITER_V6.ID
raydiumV4 := constants.DEX_PROGRAMS.RAYDIUM_V4.ID
pumpfun := constants.DEX_PROGRAMS.PUMP_FUN.ID
```
