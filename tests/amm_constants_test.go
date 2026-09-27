package tests

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/DefaultPerson/solana-dex-parser-go/constants"
)

// TestAmmEventDiscriminators recomputes the event and position discriminators
// added for the amm work package from their IDL names: 8-byte
// sha256("event:<Name>") for emit! log events, the Anchor event-instruction
// tag plus that hash for self-CPI events, sha256("global:<name>") for
// instructions.
func TestAmmEventDiscriminators(t *testing.T) {
	logEv := func(name string) []byte {
		h := sha256.Sum256([]byte("event:" + name))
		return h[:8]
	}
	cpiEv := func(name string) []byte {
		return append([]byte{228, 69, 165, 46, 81, 203, 154, 29}, logEv(name)...)
	}
	ix := func(name string) []byte {
		h := sha256.Sum256([]byte("global:" + name))
		return h[:8]
	}
	d := constants.DISCRIMINATORS
	cases := map[string]struct{ got, want []byte }{
		"RAYDIUM_CPMM.SWAP_EVENT":      {d.RAYDIUM_CPMM.SWAP_EVENT, logEv("SwapEvent")},
		"RAYDIUM_CPMM.LP_CHANGE_EVENT": {d.RAYDIUM_CPMM.LP_CHANGE_EVENT, logEv("LpChangeEvent")},

		"METEORA_DLMM.EVENTS[swap]":                  {d.METEORA_DLMM.EVENTS["swap"], cpiEv("Swap")},
		"METEORA_DLMM.EVENTS[swap2Evt]":              {d.METEORA_DLMM.EVENTS["swap2Evt"], cpiEv("Swap2Evt")},
		"METEORA_DLMM.EVENTS[claimFee]":              {d.METEORA_DLMM.EVENTS["claimFee"], cpiEv("ClaimFee")},
		"METEORA_DLMM.EVENTS[claimFee2]":             {d.METEORA_DLMM.EVENTS["claimFee2"], cpiEv("ClaimFee2")},
		"METEORA_DLMM.EVENTS[positionCreate]":        {d.METEORA_DLMM.EVENTS["positionCreate"], cpiEv("PositionCreate")},
		"METEORA_DLMM.EVENTS[positionClose]":         {d.METEORA_DLMM.EVENTS["positionClose"], cpiEv("PositionClose")},
		"METEORA_DLMM.EVENTS[lbPairCreate]":          {d.METEORA_DLMM.EVENTS["lbPairCreate"], cpiEv("LbPairCreate")},
		"METEORA_DLMM.EVENTS[rebalancing]":           {d.METEORA_DLMM.EVENTS["rebalancing"], cpiEv("Rebalancing")},
		"METEORA_DLMM.POSITION[initializePosition]":  {d.METEORA_DLMM.POSITION["initializePosition"], ix("initialize_position")},
		"METEORA_DLMM.POSITION[initializePosition2]": {d.METEORA_DLMM.POSITION["initializePosition2"], ix("initialize_position2")},
		"METEORA_DLMM.POSITION[closePosition]":       {d.METEORA_DLMM.POSITION["closePosition"], ix("close_position")},
		"METEORA_DLMM.POSITION[closePosition2]":      {d.METEORA_DLMM.POSITION["closePosition2"], ix("close_position2")},

		"METEORA_DAMM.SWAP_EVENT":                {d.METEORA_DAMM.SWAP_EVENT, logEv("Swap")},
		"METEORA_DAMM.ADD_LIQUIDITY_EVENT":       {d.METEORA_DAMM.ADD_LIQUIDITY_EVENT, logEv("AddLiquidity")},
		"METEORA_DAMM.REMOVE_LIQUIDITY_EVENT":    {d.METEORA_DAMM.REMOVE_LIQUIDITY_EVENT, logEv("RemoveLiquidity")},
		"METEORA_DAMM.BOOTSTRAP_LIQUIDITY_EVENT": {d.METEORA_DAMM.BOOTSTRAP_LIQUIDITY_EVENT, logEv("BootstrapLiquidity")},
		"METEORA_DAMM.POOL_CREATED_EVENT":        {d.METEORA_DAMM.POOL_CREATED_EVENT, logEv("PoolCreated")},
		"METEORA_DAMM.SET_POOL_FEES_EVENT":       {d.METEORA_DAMM.SET_POOL_FEES_EVENT, logEv("SetPoolFees")},
		"METEORA_DAMM.CLAIM_FEE_EVENT":           {d.METEORA_DAMM.CLAIM_FEE_EVENT, logEv("ClaimFee")},

		"METEORA_DAMM_V2.EVT_SWAP":                           {d.METEORA_DAMM_V2.EVT_SWAP, cpiEv("EvtSwap")},
		"METEORA_DAMM_V2.EVT_SWAP2":                          {d.METEORA_DAMM_V2.EVT_SWAP2, cpiEv("EvtSwap2")},
		"METEORA_DAMM_V2.EVT_LIQUIDITY_CHANGE":               {d.METEORA_DAMM_V2.EVT_LIQUIDITY_CHANGE, cpiEv("EvtLiquidityChange")},
		"METEORA_DAMM_V2.EVT_ADD_LIQUIDITY":                  {d.METEORA_DAMM_V2.EVT_ADD_LIQUIDITY, cpiEv("EvtAddLiquidity")},
		"METEORA_DAMM_V2.EVT_REMOVE_LIQUIDITY":               {d.METEORA_DAMM_V2.EVT_REMOVE_LIQUIDITY, cpiEv("EvtRemoveLiquidity")},
		"METEORA_DAMM_V2.EVT_INITIALIZE_POOL":                {d.METEORA_DAMM_V2.EVT_INITIALIZE_POOL, cpiEv("EvtInitializePool")},
		"METEORA_DAMM_V2.EVT_CLAIM_POSITION_FEE":             {d.METEORA_DAMM_V2.EVT_CLAIM_POSITION_FEE, cpiEv("EvtClaimPositionFee")},
		"METEORA_DAMM_V2.EVT_CLOSE_POSITION":                 {d.METEORA_DAMM_V2.EVT_CLOSE_POSITION, cpiEv("EvtClosePosition")},
		"METEORA_DAMM_V2.EVT_CLAIM_REWARD":                   {d.METEORA_DAMM_V2.EVT_CLAIM_REWARD, cpiEv("EvtClaimReward")},
		"METEORA_DAMM_V2.EVT_FUND_REWARD":                    {d.METEORA_DAMM_V2.EVT_FUND_REWARD, cpiEv("EvtFundReward")},
		"METEORA_DAMM_V2.EVT_INITIALIZE_REWARD":              {d.METEORA_DAMM_V2.EVT_INITIALIZE_REWARD, cpiEv("EvtInitializeReward")},
		"METEORA_DAMM_V2.EVT_CREATE_CONFIG":                  {d.METEORA_DAMM_V2.EVT_CREATE_CONFIG, cpiEv("EvtCreateConfig")},
		"METEORA_DAMM_V2.EVT_CREATE_DYNAMIC_CONFIG":          {d.METEORA_DAMM_V2.EVT_CREATE_DYNAMIC_CONFIG, cpiEv("EvtCreateDynamicConfig")},
		"METEORA_DAMM_V2.EVT_UPDATE_DELEGATE_PERMISSION":     {d.METEORA_DAMM_V2.EVT_UPDATE_DELEGATE_PERMISSION, cpiEv("EvtUpdateDelegatePermission")},
		"METEORA_DAMM_V2.EVT_WITHDRAW_DEAD_LIQUIDITY_REWARD": {d.METEORA_DAMM_V2.EVT_WITHDRAW_DEAD_LIQUIDITY_REWARD, cpiEv("EvtWithdrawDeadLiquidityReward")},
		"METEORA_DAMM_V2.EVT_CLAIM_PROTOCOL_FEE2":            {d.METEORA_DAMM_V2.EVT_CLAIM_PROTOCOL_FEE2, cpiEv("EvtClaimProtocolFee2")},

		"ORCA.POOL_INITIALIZED_EVENT":       {d.ORCA.POOL_INITIALIZED_EVENT, logEv("PoolInitialized")},
		"ORCA.LIQUIDITY_REPOSITIONED_EVENT": {d.ORCA.LIQUIDITY_REPOSITIONED_EVENT, logEv("LiquidityRepositioned")},
		"ORCA.POSITION_OPENED_EVENT":        {d.ORCA.POSITION_OPENED_EVENT, logEv("PositionOpened")},
	}
	for name, c := range cases {
		if !bytes.Equal(c.got, c.want) {
			t.Errorf("%s = %v, want %v", name, c.got, c.want)
		}
	}
}
