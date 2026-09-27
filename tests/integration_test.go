package tests

import (
	"math/big"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// Integration tests run offline on the mainnet fixtures in testdata/tx (see
// fixture_helper_test.go).

// IntegrationTestCase defines a test case
type IntegrationTestCase struct {
	Name               string
	Signature          string
	ExpectedAMM        string
	ExpectedType       string
	ExpectedInputMint  string
	ExpectedOutputMint string
	ExpectedRoute      string
	// AmountCheckSkip, when set, says why the matched trade's token-side
	// amount is not compared with the user's token balance delta
	AmountCheckSkip string
}

var integrationTestCases = []IntegrationTestCase{
	// Pumpfun trades
	{
		Name:              "Pumpfun BUY",
		Signature:         "4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz",
		ExpectedAMM:       "Pumpfun",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	{
		Name:              "Pumpfun BUY v8s",
		Signature:         "v8s37Srj6QPMtRC1HfJcrSenCHvYebHiGkHVuFFiQ6UviqHnoVx4U77M3TZhQQXewXadHYh5t35LkesJi3ztPZZ",
		ExpectedAMM:       "Pumpfun",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	// Orca trades
	{
		Name:              "Orca BUY",
		Signature:         "2kAW5GAhPZjM3NoSrhJVHdEpwjmq9neWtckWnjopCfsmCGB27e3v2ZyMM79FdsL4VWGEtYSFi1sF1Zhs7bqdoaVT",
		ExpectedAMM:       "Orca",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	// Raydium V4 trades
	{
		Name:          "BananaGun SELL via RaydiumV4",
		Signature:     "oXUd22GQ1d45a6XNzfdpHAX6NfFEfFa9o2Awn2oimY89Rms3PmXL1uBJx3CnTYjULJw6uim174b3PLBFkaAxKzK",
		ExpectedAMM:   "RaydiumV4",
		ExpectedType:  "SELL",
		ExpectedRoute: "BananaGun",
	},
	{
		Name:               "Raydium V4 SELL 5ka",
		Signature:          "5kaAWK5X9DdMmsWm6skaUXLd6prFisuYJavd9B62A941nRGcrmwvncg3tRtUfn7TcMLsrrmjCChdEjK3sjxS6YG9",
		ExpectedAMM:        "RaydiumV4",
		ExpectedType:       "SELL",
		ExpectedOutputMint: "So11111111111111111111111111111111111111112",
	},
	// Meteora DLMM trades
	{
		Name:              "Meteora DLMM BUY",
		Signature:         "125MRda3h1pwGZpPRwSRdesTPiETaKvy4gdiizyc3SWAik4cECqKGw2gggwyA1sb2uekQVkupA2X9S4vKjbstxx3",
		ExpectedAMM:       "MeteoraDLMM",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	// Meteora DAMM trades
	{
		Name:              "Meteora DAMM BUY",
		Signature:         "4uuw76SPksFw6PvxLFkG9jRyReV1F4EyPYNc3DdSECip8tM22ewqGWJUaRZ1SJEZpuLJz1qPTEPb2es8Zuegng9Z",
		ExpectedAMM:       "MeteoraDamm",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
		AmountCheckSkip:   "the user pays 171348057 of the output token to BONKbot (ZG98FU) in a later outer instruction",
	},
	// Raydium CPMM trades
	{
		Name:              "Raydium CPMM BUY",
		Signature:         "51nj5GtAmDC23QkeyfCNfTJ6Pdgwx7eq4BARfq1sMmeEaPeLsx9stFA3Dzt9MeLV5xFujBgvghLGcayC3ZevaQYi",
		ExpectedAMM:       "RaydiumCPMM",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
		ExpectedRoute:     "RaydiumRoute",
	},
	{
		Name:              "Raydium CPMM BUY afU",
		Signature:         "afUCiFQ6amxuxx2AAwsghLt7Q9GYqHfZiF4u3AHhAzs8p1ThzmrtSUFMbcdJy8UnQNTa35Fb1YqxR6F9JMZynYp",
		ExpectedAMM:       "RaydiumCPMM",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	// Raydium CL trades
	{
		Name:               "Raydium CL SELL",
		Signature:          "2durZHGFkK4vjpWFGc5GWh5miDs8ke8nWkuee8AUYJA8F9qqT2Um76Q5jGsbK3w2MMgqwZKbnENTLWZoi3d6o2Ds",
		ExpectedAMM:        "RaydiumCL",
		ExpectedType:       "SELL",
		ExpectedOutputMint: "So11111111111111111111111111111111111111112",
	},
	// Jupiter-related trades (via Jupiter DCA)
	{
		Name:              "Jupiter DCA BUY via MeteoraDLMM",
		Signature:         "2euJJaq2LCagFjjENTUdn7n6LgobDvBdsfPLovCXnZ9pknMnKtcHo1Vfw8c8kghGzzHRcYuWoEyQFAtuCm9TXGr1",
		ExpectedAMM:       "MeteoraDLMM",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
		ExpectedRoute:     "JupiterDCA",
	},
	// Moonit trades
	{
		Name:              "Moonit BUY",
		Signature:         "AhiFQX1Z3VYbkKQH64ryPDRwxUv8oEPzQVjSvT7zY58UYDm4Yvkkt2Ee9VtSXtF6fJz8fXmb5j3xYVDF17Gr9CG",
		ExpectedAMM:       "Moonit",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	{
		Name:               "Moonit SELL",
		Signature:          "2XYu86VrUXiwNNj8WvngcXGytrCsSrpay69Rt3XBz9YZvCQcZJLjvDfh9UWETFtFW47vi4xG2CkiarRJwSe6VekE",
		ExpectedAMM:        "Moonit",
		ExpectedType:       "SELL",
		ExpectedOutputMint: "So11111111111111111111111111111111111111112",
	},
	// Pumpswap trades
	{
		Name:              "Pumpswap BUY",
		Signature:         "2W1ScejYBFe6kS4VnTmj14qaEmeqiV1Rf6TXfQU9PZcJEW5ERqY19kSWWgLtdfVJKx1PCMBGvXiJWc65o59VNAtf",
		ExpectedAMM:       "Pumpswap",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	{
		Name:               "Pumpswap SELL",
		Signature:          "36Q2tYo1CPa42GF51bzA493nYQCG8fPbpQJEzRhZQURYuBcRKpj97HWBCLCzDwgQJ8tnVrW9fDZKWaPBdADEsxTE",
		ExpectedAMM:        "Pumpswap",
		ExpectedType:       "SELL",
		ExpectedOutputMint: "So11111111111111111111111111111111111111112",
	},
	// Boopfun trades (boop8hVGQGqehUK2iVEMEnMrL5RbjywRzHKBmBE7ry4)
	{
		Name:              "Boopfun BUY",
		Signature:         "28S2MakapF1zTrnqYHdMxdnN9uqAfKV2fa5ez9HpE466L3xWz8AXwsz4eKXXnpvX8p49Ckbp26doG5fgW5f6syk9",
		ExpectedAMM:       "Boopfun",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	{
		Name:               "Boopfun SELL",
		Signature:          "3Lyh3wAPkcLGKydqT6VdjMsorLUJqEuDeppxh79sQjGxuLiMqMgB75aSJyZsM3y3jJRqdLJYZhNUBaLeKQ8vL4An",
		ExpectedAMM:        "Boopfun",
		ExpectedType:       "SELL",
		ExpectedOutputMint: "So11111111111111111111111111111111111111112",
	},
	// Jupiter V6 trades (JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4)
	{
		Name:              "Jupiter V6 via Moonit",
		Signature:         "3TZKJLxy4H2wQiYenuSVoQ2ox7xveRoG5bxr7yfmEdtMPqKmdHWcf5Q9B8uUBi6ystp2gQsZdP5qxiYK4JnUpm7",
		ExpectedAMM:       "Moonit", // Jupiter routes through Moonit
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
		ExpectedRoute:     "Jupiter",
	},
	// OKX trades (6m2CDdhRgxpH4WjvdzxAYbGxwdGUz5MziiL5jek2kBma)
	{
		Name:          "OKX via RaydiumV4",
		Signature:     "5xaT2SXQUyvyLGsnyyoKMwsDoHrx1enCKofkdRMdNaL5MW26gjQBM3AWebwjTJ49uqEqnFu5d9nXJek6gUSGCqbL",
		ExpectedAMM:   "RaydiumV4", // OKX routes through Raydium V4
		ExpectedRoute: "OKX",
	},
	{
		Name:               "OKX SELL",
		Signature:          "53tdwmNWEp9KsyegiDk7Z3DXVfSQoBXpAJfZbpAUTwzCtDkfrbdCN17ksQnKdH2p9yBTrYHGhTvHrckaPCSshBkU",
		ExpectedType:       "SELL",
		ExpectedOutputMint: "So11111111111111111111111111111111111111112",
		ExpectedRoute:      "OKX",
	},
	// Raydium Launchpad trades
	{
		Name:              "Raydium Launchpad BUY",
		Signature:         "61AN23VGPknSqskF6CvtZqrD4LtL2CNGYKeyFc5nLVcfaUaHV5LsQe6HTnRFM6pNX8qf7fkqZ5tEZfnNEF73H8MX",
		ExpectedAMM:       "RaydiumLaunchpad",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	{
		Name:              "Raydium Launchpad BUY_EXACT_IN",
		Signature:         "Gi44zBwsd8eUGEVPS1jstts457hKLbm8SSMLrRVHVK2McrhJjosiszb65U1LdrjsF1WfCXoesLMhm8RX3dchx4s",
		ExpectedAMM:       "RaydiumLaunchpad",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	{
		Name:               "Raydium Launchpad SELL_EXACT_IN",
		Signature:          "36n8GMHRMSyX8kRSgaUfcE5jpjWNWhjAu7YPeYFX2fMVzirJT4YhvYMo4dS5VoCVj5H47qZ8FzSEDLc6ui78HcAh",
		ExpectedAMM:        "RaydiumLaunchpad",
		ExpectedType:       "SELL",
		ExpectedOutputMint: "So11111111111111111111111111111111111111112",
	},
	// Additional coverage tests - SELL operations
	{
		Name:               "Meteora DLMM SELL",
		Signature:          "7YPF21r7JBDeoXuMJn6KSqDVYGrm821U87Cnje3xPvZpMUVaAEAvCGJPP6va2b5oMLAzGku5s3TcNAsN6zdXPRn",
		ExpectedAMM:        "MeteoraDLMM",
		ExpectedType:       "SELL",
		ExpectedOutputMint: "So11111111111111111111111111111111111111112",
		AmountCheckSkip:    "the same transaction also removes liquidity (outer 10, 11) and buys back (outer 17)",
	},
	{
		Name:          "Jupiter DCA SELL via RaydiumV4",
		Signature:     "4mxr44yo5Qi7Rabwbknkh8MNUEWAMKmzFQEmqUVdx5JpHEEuh59TrqiMCjZ7mgZMozRK1zW8me34w8Myi8Qi1tWP",
		ExpectedAMM:   "RaydiumV4",
		ExpectedType:  "SELL",
		ExpectedRoute: "JupiterDCA",
	},
	// Additional coverage tests - BUY operations
	{
		Name:              "Maestro BUY via RaydiumV4",
		Signature:         "mWaH4FELcPj4zeY4Cgk5gxUirQDM7yE54VgMEVaqiUDQjStyzwNrxLx4FMEaKEHQoYsgCRhc1YdmBvhGDRVgRrq",
		ExpectedAMM:       "RaydiumV4",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
		ExpectedRoute:     "Maestro",
	},
	{
		Name:              "Raydium CL BUY",
		Signature:         "4MSVpVBwxnYTQSF3bSrAB99a3pVr6P6bgoCRDsrBbDMA77WeQqoBDDDXqEh8WpnUy5U4GeotdCG9xyExjNTjYE1u",
		ExpectedAMM:       "RaydiumCL",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	{
		Name:          "Meteora DLMM SELL via OKX",
		Signature:     "33VnDBtrFawBRYwDqomdsH57GL83B7eWTQN5mnga9F1whyMzcpdmURnPkAjqDte8Ja9EcsGcejhDYcUKkA9sE4HG",
		ExpectedAMM:   "MeteoraDLMM",
		ExpectedType:  "SELL",
		ExpectedRoute: "OKX",
	},
	// Trading Bots - new tests
	{
		Name:              "Bloom BUY via Pumpfun",
		Signature:         "U9K99asspxi8WzTHhzmvBZZ5BtaZsFijgrgW3zYLSkjGTEXLzHjtwffe3X85LPxqRQ8NvSdS3trhag5qptFASRj",
		ExpectedAMM:       "Pumpfun",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
		ExpectedRoute:     "Bloom",
	},
	// OKX Routes - additional tests
	{
		Name:              "Pumpfun BUY via OKX",
		Signature:         "648cwSysqKXnb3XLPy577Lu4oBk7jimaY8p95JGfS9QUNabYar5pzfcRdu518TWw3dbopquJnMne9qx22xuf8xqn",
		ExpectedAMM:       "Pumpfun",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
		ExpectedRoute:     "OKX",
	},
	{
		Name:          "OKX SELL via RaydiumV4 3rE",
		Signature:     "3rEob1PiezEtzhjPJcDJ9menwWeUBmF19FfYysHP5v6DRQe6PVrWcLRBvAGDbB9Ubn8PF8FVKjQYVxDjA2wAwSgn",
		ExpectedAMM:   "RaydiumV4",
		ExpectedType:  "SELL",
		ExpectedRoute: "OKX",
	},
	// Additional coverage - new tests
	{
		Name:              "Pumpfun BUY 5fu",
		Signature:         "5fuBdjC7G3ABez84ZBppPz4SXg1EmhKFToXhfPpr8qGddVTnUaAvrgZ5UPfN3PXJdAXEnWAeoZfDXjG23u25trZB",
		ExpectedAMM:       "Pumpfun",
		ExpectedType:      "BUY",
		ExpectedInputMint: "So11111111111111111111111111111111111111112",
	},
	{
		Name:               "RaydiumV4 SELL 4qU",
		Signature:          "4qUyABFnkT7wesZehkrYXYvUVtoS5XERm397ZUXAn7TRrXgrupFtEoPLZnzqh91SW8ZZZhaiQWxb4eVWftNhPmmC",
		ExpectedAMM:        "RaydiumV4",
		ExpectedType:       "SELL",
		ExpectedOutputMint: "So11111111111111111111111111111111111111112",
	},
}

func TestIntegrationParseTrades(t *testing.T) {
	parser := dexparser.NewDexParser()

	for _, tc := range integrationTestCases {
		t.Run(tc.Name, func(t *testing.T) {
			tx := loadFixture(t, tc.Signature)

			config := &types.ParseConfig{TryUnknownDEX: true}
			trades := parser.ParseTrades(tx, config)
			if len(trades) == 0 {
				t.Fatalf("Expected at least 1 trade, got 0")
			}

			// Find the best matching trade
			// Priority: 1) matches AMM + Type, 2) matches AMM only, 3) first trade
			var matchingTrade *types.TradeInfo
			var ammOnlyMatch *types.TradeInfo
			for i := range trades {
				trade := &trades[i]
				ammMatches := tc.ExpectedAMM == "" || trade.AMM == tc.ExpectedAMM
				typeMatches := tc.ExpectedType == "" || string(trade.Type) == tc.ExpectedType

				if ammMatches && typeMatches && matchingTrade == nil {
					matchingTrade = trade
				} else if ammMatches && ammOnlyMatch == nil {
					ammOnlyMatch = trade
				}
			}
			// Fallback: AMM-only match, then first trade
			if matchingTrade == nil {
				matchingTrade = ammOnlyMatch
			}
			if matchingTrade == nil {
				matchingTrade = &trades[0]
			}

			trade := matchingTrade

			if tc.ExpectedAMM != "" && trade.AMM != tc.ExpectedAMM {
				t.Errorf("Expected AMM %s, got %s", tc.ExpectedAMM, trade.AMM)
			}

			if tc.ExpectedType != "" && string(trade.Type) != tc.ExpectedType {
				t.Errorf("Expected type %s, got %s", tc.ExpectedType, trade.Type)
			}

			if tc.ExpectedInputMint != "" && trade.InputToken.Mint != tc.ExpectedInputMint {
				t.Errorf("Expected input mint %s, got %s", tc.ExpectedInputMint, trade.InputToken.Mint)
			}

			if tc.ExpectedOutputMint != "" && trade.OutputToken.Mint != tc.ExpectedOutputMint {
				t.Errorf("Expected output mint %s, got %s", tc.ExpectedOutputMint, trade.OutputToken.Mint)
			}

			if tc.ExpectedRoute != "" && trade.Route != tc.ExpectedRoute {
				t.Errorf("Expected route %s, got %s", tc.ExpectedRoute, trade.Route)
			}

			// The token side of the trade must match the user's token balance
			// change (independent of the parser) whenever the user's balance of
			// that mint changed
			if tc.AmountCheckSkip != "" {
				return
			}
			if trade.OutputToken.Mint != solMint {
				delta := ownerTokenDelta(tx, trade.User, trade.OutputToken.Mint)
				if delta.Sign() != 0 && bigStr(trade.OutputToken.AmountRaw).Cmp(delta) != 0 {
					t.Errorf("output %s %s, user's balance change %s", trade.OutputToken.Mint, trade.OutputToken.AmountRaw, delta)
				}
			}
			if trade.InputToken.Mint != solMint {
				delta := ownerTokenDelta(tx, trade.User, trade.InputToken.Mint)
				if delta.Sign() != 0 && bigStr(trade.InputToken.AmountRaw).Cmp(new(big.Int).Neg(delta)) != 0 {
					t.Errorf("input %s %s, user's balance change %s", trade.InputToken.Mint, trade.InputToken.AmountRaw, delta)
				}
			}
		})
	}
}

func TestIntegrationParseAll(t *testing.T) {
	parser := dexparser.NewDexParser()

	// Pump.fun create + buy
	signature := "4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz"
	tx := loadFixture(t, signature)

	config := &types.ParseConfig{TryUnknownDEX: true}
	result := parser.ParseAll(tx, config)

	if result == nil {
		t.Fatal("ParseAll returned nil")
	}
	if !result.State {
		t.Fatalf("Parse failed: %s", result.Msg)
	}
	if result.Signature != signature {
		t.Errorf("Signature = %s", result.Signature)
	}

	// one Pump.fun buy, and the create + buy meme events
	if len(result.Trades) != 1 || result.Trades[0].AMM != "Pumpfun" || result.Trades[0].Type != types.TradeTypeBuy {
		t.Fatalf("Trades = %v, want one Pump.fun BUY", result.Trades)
	}
	user := tx.Transaction.Message.AccountKeys[0].Pubkey
	trade := result.Trades[0]
	if trade.User != user || bigStr(trade.OutputToken.AmountRaw).Cmp(ownerTokenDelta(tx, user, trade.OutputToken.Mint)) != 0 {
		t.Errorf("trade %s, want output = user's balance change %s", tradeKey(trade), ownerTokenDelta(tx, user, trade.OutputToken.Mint))
	}
	if len(result.MemeEvents) != 2 || result.MemeEvents[0].Type != types.TradeTypeCreate || result.MemeEvents[1].Type != types.TradeTypeBuy {
		t.Errorf("MemeEvents = %+v, want CREATE then BUY", result.MemeEvents)
	}
	if result.Fee.Amount != "80285" {
		t.Errorf("Fee = %s, want 80285 (meta.fee)", result.Fee.Amount)
	}
}
