package tests

import (
	"math"
	"testing"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
)

// liquidityCase is one expected liquidity event of a real mainnet transaction.
// Expected values come from the upstream TypeScript test suite
// (solana-dex-parser src/__tests__/liquidity-*.test.ts, v2.6.7), corrected
// where upstream's own later fixes changed the result (see comments).
type liquidityCase struct {
	suite   string
	sig     string
	typ     types.PoolEventType
	pool    string
	mint0   string
	amount0 float64
	mint1   string
	amount1 float64
}

var upstreamLiquidityCases = []liquidityCase{
	{"meteora-pools", "2GWLwbEjyR7moFYK5JfapbsDBrBz3298BVWsAebhUECPaXjLbTZ6DkbEN34BF57jdGot7GkwDnrzszFB3H9AJxmS", types.PoolEventTypeCreate, "BCXjm4FfSoquZQJV5Wcje1g1pSHW2hFMU9wDE98Nyatb", "STrikemJEk2tFVYpg7SMo9nGPrnJ56fHnS1K7PV2fPw", 100000000.0, "So11111111111111111111111111111111111111112", 2740.0},       // Meteora Pools Program: initializePermissionlessConstantProductPoolWithConfig
	{"meteora-pools", "LaocVd6PpfdH1KTdQuRTf5WwnUzmyf3gdAy16xro747nzrhpgXg1oxFrpgBk31tPh24ksVAyiSkNW7vncoKTGyH", types.PoolEventTypeAdd, "BCXjm4FfSoquZQJV5Wcje1g1pSHW2hFMU9wDE98Nyatb", "STrikemJEk2tFVYpg7SMo9nGPrnJ56fHnS1K7PV2fPw", 720.780405, "So11111111111111111111111111111111111111112", 0.029097874},       // Meteora Pools Program: addBalanceLiquidity
	{"meteora-pools", "2xEAewTjtSHgpEHHzaNjHiuoMHNZQXz5vySXHCSL8omujjvaxq9JsfGWjusz43ndmzcu5riESKm1UH4riWDX9v1v", types.PoolEventTypeRemove, "BCXjm4FfSoquZQJV5Wcje1g1pSHW2hFMU9wDE98Nyatb", "STrikemJEk2tFVYpg7SMo9nGPrnJ56fHnS1K7PV2fPw", 14938.609562, "So11111111111111111111111111111111111111112", 0.578534516}, // Meteora Pools Program: removeBalanceLiquidity
	{"meteora", "2vkT747Y9udxkiCD6bqGTME65G47xnrQ9mtvwoHXBwPNQJ2he6XBCGemxiD55oDeKcZ4vHbfgTYiX8ofr1a4phD6", types.PoolEventTypeAdd, "GHfVqcnXhGLEHhvUZuTKyE68cHwsvnvYMatapWETgASC", "7w4XU7wWKoCB3bfM7Vi7zmHVSokoPP7Nb2oQuceiWb3s", 8.240658, "So11111111111111111111111111111111111111112", 0.0},                     // Meteora DLMM Program: addLiquidity
	{"meteora", "5oMPfyxsWgDnCSpxriQa4QR3jCuW6SUqCix1aD4T1a9As36uoxURH5KiZS1xTbRrwVRsLxrtW8hY1655LiVREBkL", types.PoolEventTypeAdd, "7q1WMhzQyqTT49xiSF6AXMvUNjFxTcS81nxuQQUyqp1P", "jz4nRUM5ScvhzCEdRxtJ2fQt4zyBgbvn7VYemHiKrYC", 250109.689531, "So11111111111111111111111111111111111111112", 0.0},                 // Meteora DLMM Program: addLiquidity
	// Meteora DLMM Program: addLiquidityByStrategyOneSide. Upstream reports the
	// USDC deposit as token0; the quote mint is token1 (types.PoolEvent, as
	// for the pair's other events), and the instruction does not name the
	// pair's other mint (final review amm-4)
	{"meteora", "4HAaBUNWYGQZsEf1VJAvzAtF6sLL8pb7CEMeBxNdivrWwP6XhNVNBH446NpoVtVyFoGuT1bu8jSpVHa8xe1yXtf", types.PoolEventTypeAdd, "ARwi1S4DaiTG5DX7S4M4ZsrXqpMD1MrTmbu9ue2tpmEq", "", 0.0, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 3000.405339},
	{"meteora", "4BAG7T3kBo2immawL7eHecG7wretTnJ3C7Xcg3ouftZSVf7KcHPaLm93yD7KUgzKeqzkErg15UWV92kD3wSo6NUj", types.PoolEventTypeAdd, "27M7AnaFpW68thenG1oVAc7TCVnjPGM3LeZr3HixmQRG", "", 0.0, "So11111111111111111111111111111111111111112", 4.999999982},                                                                // Meteora DLMM Program: addLiquidityByStrategyOneSide
	{"meteora", "65iaEBLYVvv6vrxTTsPhi6vhtYi5XhsuXBqbugRL8fZ6PQXZ4L7ocHZxXxBNAz2LsXCqBk56Gb6vGP2BA5PvmqJ1", types.PoolEventTypeAdd, "5fU1WwLVRkDg8RiDkzvcETSSLqxcd3ry4tFmRW6oRcNN", "", 0.0, "So11111111111111111111111111111111111111112", 7.99999999},                                                                 // Meteora DLMM Program: addLiquidityOneSide
	{"meteora", "2japn1cjrYtPjS5RZj8kE4cuCZKSq2ihP6ZbyaA3x4CpuYyJtbbfx8mctCpCg7hRpNrwNcxiGYkrveDmcQLuBGP2", types.PoolEventTypeAdd, "4FCeBszj9XeS2whpRbVKjxi7Z6AEws9hoaFDu7ctnjkR", "5jctYwusTPnD3PQoanMuqDstini9RnUPPP4NKbzWwgY", 847432538, "", 0},                                                                    // Meteora DLMM Program: addLiquidityOneSidePrecise
	{"meteora", "2japn1cjrYtPjS5RZj8kE4cuCZKSq2ihP6ZbyaA3x4CpuYyJtbbfx8mctCpCg7hRpNrwNcxiGYkrveDmcQLuBGP2", types.PoolEventTypeAdd, "4FCeBszj9XeS2whpRbVKjxi7Z6AEws9hoaFDu7ctnjkR", "5jctYwusTPnD3PQoanMuqDstini9RnUPPP4NKbzWwgY", 47, "", 0},                                                                           // Meteora DLMM Program: addLiquidityOneSide (second instruction; one-sided, no token1)
	{"meteora", "PZbntZdTgAh99WvNZ2emugv6wV8FwT94vZhP1FjBaqBm8bHuSpJZ5tS3vqi53jVErHWZ1MNMLwtDgnr7VNn2dbf", types.PoolEventTypeAdd, "BoeMUkCLHchTD31HdXsbDExuZZfcUppSLpYtV3LZTH6U", "J1toso1uCk3RLmjorhTtrVwY9HJ7X8V9yYac6Y7kGCPn", 243.51585425, "So11111111111111111111111111111111111111112", 60.589950682},           // Meteora DLMM Program: addLiquidityByWeight
	{"meteora", "4AZBcYYZfHTsrPXttv8UCAqKLBj8j8roEHW5evbpCYdx34e6Sxsb5P1VGiPMjScKy1caTtwGEfQH9rFUhvDdJxMU", types.PoolEventTypeAdd, "8Bn5BW27CSPosSWqzBywXruKoBLjGgSfdys15uP9RavL", "1Qf8gESP4i6CFNWerUSDdLKJ9U1LpqTYvjJ2MM4pain", 0.0, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 69730.4961},                     // Meteora DLMM Program: addLiquidityByStrategy
	{"meteora", "3b536UNQGGwrfHsDA9tLTKHwcy5G2eMZMSPxDqWhGbQbNrFSd5QaPHMzuufotiv5sBQWNvLvfcmToHGLoLoU1jz5", types.PoolEventTypeAdd, "9d9mb8kooFfaD3SctgZtkxQypkshx6ezhbKio89ixyy2", "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN", 0.0, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 619500.976753},                 // Meteora DLMM Program: addLiquidityByStrategy
	{"meteora", "sYGH1o3g8crh5dcgzn3WBD29wPxaKKXhPL1Baf1tac79yUTWmQ484vXbyf48Gr85RAgTHFc8scjPvyMaJnW24NA", types.PoolEventTypeRemove, "8Bn5BW27CSPosSWqzBywXruKoBLjGgSfdys15uP9RavL", "1Qf8gESP4i6CFNWerUSDdLKJ9U1LpqTYvjJ2MM4pain", 0.0, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 69730.496031},                 // Meteora DLMM Program: removeLiquidityByRange
	{"meteora", "38HUTGqsFeMzQkcqcV1DsmPfs6N4JhJEoa3vESHs7Zic4KyW3HkRTNyWmhCbw1YBub12sRcQ2VAjkUJcGVQpzYXg", types.PoolEventTypeRemove, "9d9mb8kooFfaD3SctgZtkxQypkshx6ezhbKio89ixyy2", "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN", 20.919301, "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 110.714287},           // Meteora DLMM Program: removeLiquidityByRange
	{"meteora", "2mNBApzw9HXF3TJXsLoBtSxc3DqC6XVYbtwchmyUA9mvt9oVamYzpHa9axF62y9aNnXMzyKTzpfCcsHNsLgUcSnE", types.PoolEventTypeRemove, "5rCf1DM8LjKTw4YqhnoLcngyZYeNnQqztScTogYHAS6", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 0.108148, "So11111111111111111111111111111111111111112", 0.000233718},             // Meteora DLMM Program: removeAllLiquidity
	{"meteora", "9Wr5VreyNGZQ2vEqmnbZ1sAWPFrHKaxfM9m1qMiTcSzwvz6tJVWt9NWHuNeRoWxvWoZi3bDko1C6kS1jHVwSf9H", types.PoolEventTypeRemove, "6ehEi3xc5DX3SVPiBpnRwPW3nQJBTzDynA26ce3AnYPp", "h5NciPdMZ5QCB5BYETJMYBMpVx9ZuitR6HcVjyBhood", 3342.210433, "So11111111111111111111111111111111111111112", 1.384101939},           // Meteora DLMM Program: removeAllLiquidity
	{"meteora", "Cj2c5dEmHvmMWwkMa4QMauQE6aBbyRz5mn4fEYARez2bHqukkJ3nbYAdst9ixQsAMh9G9tUNntAxEXpgrz5T1Qi", types.PoolEventTypeRemove, "BoeMUkCLHchTD31HdXsbDExuZZfcUppSLpYtV3LZTH6U", "J1toso1uCk3RLmjorhTtrVwY9HJ7X8V9yYac6Y7kGCPn", 18.504862033, "So11111111111111111111111111111111111111112", 6.074752463},         // Meteora DLMM Program: removeLiquidity
	{"orca", "3mZcyeDJysgs79nLcvtN4XQ6iepyERqG93P2F2ZYgUX4ZF1Yr1XFBMKR8DHd7z4gN2EmvAqMc3KhQTQpGMbtvhF7", types.PoolEventTypeAdd, "C1MgLojNLWBKADvu9BHdtgzz1oZX4dZ5zGdGcgvvW8Wz", "JUPyiwrYJFskUPiHa7hkeR8VUtAeFoSYbKedZNsDvCN", 6931.015285, "So11111111111111111111111111111111111111112", 45.407996732},               // Whirlpools Program: increaseLiquidity
	{"orca", "4Kv6gQgdSsCPSxRApiCNMHFE1dKKGVugrJTzdzSYX5a2aXho4o7jaQDSHLH3RTsr5aVwpkzWL1o5mSCyDtHeZKZr", types.PoolEventTypeAdd, "Djf5NYkwhdipTW3ZScPeUkjf7BLtDdxLvEGtNyWMtw3d", "STrikemJEk2tFVYpg7SMo9nGPrnJ56fHnS1K7PV2fPw", 2000000.0, "So11111111111111111111111111111111111111112", 6.315579644},                  // Whirlpools Program: increaseLiquidityV2
	{"orca", "23zkGAorUC3aHSk7zJYiUvvs6gEXPPzp8xRiWfACzkrqBEaQrKiH9QCgrmwSTD6hxKKEjryEGbEvurt6xSBpuBMC", types.PoolEventTypeRemove, "C1MgLojNLWBKADvu9BHdtgzz1oZX4dZ5zGdGcgvvW8Wz", "JUPyiwrYJFskUPiHa7hkeR8VUtAeFoSYbKedZNsDvCN", 106.470976, "So11111111111111111111111111111111111111112", 0.475676716},              // Whirlpools Program: decreaseLiquidity
	{"raydium-cl", "5BEKeMuUfah3wFkCvMmaGDq5JDTat2nokaqMURYZubNDm9WdQrMmwWK4YcL7nksuq94k62wxgbbbwUf5LCgtXU4J", types.PoolEventTypeAdd, "DFX9AHEnoU8caagtFFGiv7xnsEJ3DTh5TQo8XktJHoTN", "1Qf8gESP4i6CFNWerUSDdLKJ9U1LpqTYvjJ2MM4pain", 127.164994, "So11111111111111111111111111111111111111112", 7.199999997},           // Raydium Concentrated Liquidity: openPositionWithToken22Nft
	{"raydium-cl", "4Vv9ZWLizvRE7um22gF8bUWvD5UfK1TMXsP4hF8TVF4gc2BNmmPG8kFu7Dyod9Zw5x16xAsGeDJnUznCwaKXim5n", types.PoolEventTypeCreate, "GQsPr4RJk9AZkkfWHud7v4MtotcxhaYzZHdsPCg9vNvW", "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN", 0, "So11111111111111111111111111111111111111112", 0},                          // Raydium Concentrated Liquidity: createPool (corrected, see clmmCreateCases)
	{"raydium-cl", "54t2sbzBxmejGNYmttn5nr4fDeRHdZC6CF4eiAm3Was7WKrvw6LEH51gb2RGw5Wom2y12vW11o2hwaHBE3W4Rgx9", types.PoolEventTypeAdd, "5MczDZ1DYBpCyjXWhyLXAW4AofywKizDF8w4cLZqvpuV", "kH6hPcpdJqeMAATYU7W4rzqZuzYTkYr6QqGYTLkpump", 2796.229978, "So11111111111111111111111111111111111111112", 0.004266144},          // Raydium Concentrated Liquidity: openPosition
	{"raydium-cl", "2Bm6Xh3UQYCYywCQPEJ1tCQKrSPHRPGukN4qTmutCYnR45PKMBhTuBjMdbT7x3fsLnqvduAeiTQeVRQ76NA2NMGN", types.PoolEventTypeAdd, "8sLbNZoA1cfnvMJLPfp98ZLAnFSYCFApfJKMbiXNLwxj", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 41677.305735, "So11111111111111111111111111111111111111112", 215.71233501},       // Raydium Concentrated Liquidity: openPosition
	{"raydium-cl", "61auheJ9MhRbQeXqiAMitgWWv2yxkDSCqb6qauMr77UaENwR7yUfc5LCH8pExkjo1QqnqYLvRu9vNXi5QRZST19S", types.PoolEventTypeAdd, "EYy5nwcFQHfSCKrrALgMB2DbU2vhszWHjuEdDFgAvfpu", "9McvH6w97oewLmPxqQEoHUAv3u5iYMyQ9AeZZhguYf1T", 9.526157579, "So11111111111111111111111111111111111111112", 0.471506154},         // Raydium Concentrated Liquidity: openPositionV2 (corrected: ADD)
	{"raydium-cl", "54TqXAJT3JcuJe37TMT6nYcZzuRgYMkcLinZt4vNoJF3aVmFTU5yD1fWCFuqygSH4tBtyUJnb4fxUBA8XtGYsDRC", types.PoolEventTypeAdd, "H6aoNRGBnzMfpAfSkaS2uXh7PsT3F9nzj1HFCmBgGTgY", "1Qf8gESP4i6CFNWerUSDdLKJ9U1LpqTYvjJ2MM4pain", 4.848942, "So11111111111111111111111111111111111111112", 0.215266532},             // Raydium Concentrated Liquidity: increaseLiquidityV2
	{"raydium-cl", "4hVaQHrCJrzfPTP7XsbtUnwW7qWEM2fwF68WJQnvNfNVEU3Va5RAeZHACF93xy4vYeoxT52nK61hDk8twCW99BJc", types.PoolEventTypeAdd, "DFX9AHEnoU8caagtFFGiv7xnsEJ3DTh5TQo8XktJHoTN", "1Qf8gESP4i6CFNWerUSDdLKJ9U1LpqTYvjJ2MM4pain", 4.279968, "So11111111111111111111111111111111111111112", 0.278012566},             // Raydium Concentrated Liquidity: increaseLiquidityV2
	{"raydium-cl", "2Uc81FAsAsmgSmwNk4xLdnzPQ1ALJrCou4SjTm6C3C5gXDMEYFmWL9a8VUcop5AJvWpTFa9vVxSMxgvNpumhCKnM", types.PoolEventTypeAdd, "GQsPr4RJk9AZkkfWHud7v4MtotcxhaYzZHdsPCg9vNvW", "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN", 415.62333, "So11111111111111111111111111111111111111112", 5.658872061},           // Raydium Concentrated Liquidity: increaseLiquidityV2
	{"raydium-cl", "4VpDFKjjyBNjS3amzzqW22mx1aLNQnYwm1EyZT4kNkHrDpfNCfuZWjm8UL4br5ReNdVUjTmFdqgoUBX2eKuhVndt", types.PoolEventTypeAdd, "BZtgQEyS6eXUXicYPHecYQ7PybqodXQMvkjUbP4R8mUU", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 544.16166, "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", 468.509899},           // Raydium Concentrated Liquidity: increaseLiquidity
	{"raydium-cl", "48oGGt6rsBqbiyj7xyzWD8oRXk3sGhE6Kt6LzRG5QofL1LJqwtcfp4uKXeinDn4a24uyWJVGDPTaAFac2R5eNX1w", types.PoolEventTypeRemove, "DFX9AHEnoU8caagtFFGiv7xnsEJ3DTh5TQo8XktJHoTN", "1Qf8gESP4i6CFNWerUSDdLKJ9U1LpqTYvjJ2MM4pain", 13.5631, "So11111111111111111111111111111111111111112", 0.196747275},           // Raydium Concentrated Liquidity: decreaseLiquidityV2
	{"raydium-cl", "44Ccr2ftN83ryWyCLjspAUZXSLy4fxWsLhCP3xmBftHN3MhWBd1k3ctFjwvJBFQGRtZo2uof4qM3NazQkn9gn5UZ", types.PoolEventTypeRemove, "GQsPr4RJk9AZkkfWHud7v4MtotcxhaYzZHdsPCg9vNvW", "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN", 91.825081, "So11111111111111111111111111111111111111112", 3.376022161},        // Raydium Concentrated Liquidity: decreaseLiquidityV2
	{"raydium-cl", "7JX8oo13G3q812rQ7FwarAKGHzHBKS4JqQSfkpso826HeFifupXgBo6yhnkXE8Yt8xfAG2Qv6SwvrNgRYWiEy8K", types.PoolEventTypeRemove, "BZtgQEyS6eXUXicYPHecYQ7PybqodXQMvkjUbP4R8mUU", "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", 74.3064, "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", 63.893602},            // Raydium Concentrated Liquidity: decreaseLiquidity
	{"raydium-cpmm", "xZmKodPHYxesDJzPLHfjqG4VvKJcQpwuo3fTa2T64LY9nePfAa97xtSmzCvSWJ5EFzYjAURLD666zqV8oJL8kTp", types.PoolEventTypeCreate, "CXgcuECqdaBpvJWH5cwEir9Y5FY9SKTjhGutMc95bGy3", "znv3FZt2HFAvzYf5LxzVyryh3mBXWuTRRng25gEZAjh", 1000000000.0, "So11111111111111111111111111111111111111112", 20.0},            // Raydium CPMM: initialize
	{"raydium-cpmm", "55JBLGRP6Zcd4t8gxsQ27EbLN3kzzMMnwxA9iD1CUTsax2C2cJLNbW3nuRg1gCH56ojJ4YNpRcuzpDL1PbnRvh18", types.PoolEventTypeCreate, "HKuJrP5tYQLbEUdjKwjgnHs2957QKjR2iWhJKTtMa1xs", "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN", 423.320786, "So11111111111111111111111111111111111111112", 0.352896784},     // Raydium CPMM: initialize
	{"raydium-cpmm", "2uVjffn9bwatwRnGZYRpJk5fhxTcGcGfvtc971FBdt1CgUhxESrG8qPN3DiahgaLDULC4JTqc1whSCfvZ5gzJRmL", types.PoolEventTypeAdd, "CXgcuECqdaBpvJWH5cwEir9Y5FY9SKTjhGutMc95bGy3", "znv3FZt2HFAvzYf5LxzVyryh3mBXWuTRRng25gEZAjh", 928.616834, "So11111111111111111111111111111111111111112", 0.064818705},         // Raydium CPMM: deposit
	{"raydium-cpmm", "38Bf6HotXPuGcErvexLS4tCQVnLSuD1PtvhgNuCmhbbNwwDKXNSw8Gc6MQwgWwpU5FNUJwGac1ziSS6kKDJdW1Rr", types.PoolEventTypeAdd, "HKuJrP5tYQLbEUdjKwjgnHs2957QKjR2iWhJKTtMa1xs", "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN", 972.595817, "So11111111111111111111111111111111111111112", 95.855640586},       // Raydium CPMM: deposit
	{"raydium-cpmm", "5sYGW9tkSuh4UHUNGLwtLRSuAGHTTKQMqgaGtF7jB4r8EzADkVmq3nPfiWiQ7UhPn4vj2bpqNXvcKpqgMX1uWc97", types.PoolEventTypeRemove, "CXgcuECqdaBpvJWH5cwEir9Y5FY9SKTjhGutMc95bGy3", "znv3FZt2HFAvzYf5LxzVyryh3mBXWuTRRng25gEZAjh", 1735.661744, "So11111111111111111111111111111111111111112", 0.03501825},      // Raydium CPMM: withdraw
	{"raydium-cpmm", "5eVe9LMAHgz6Ze7VrUn1XHgoaWJXinM2uoEvRciVoJsADCDc8v4HrewQorp4JUx3jAzBSw5p3RrSAYFqd95udWxR", types.PoolEventTypeRemove, "HKuJrP5tYQLbEUdjKwjgnHs2957QKjR2iWhJKTtMa1xs", "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN", 977.537752, "So11111111111111111111111111111111111111112", 95.417034698},    // Raydium CPMM: withdraw
	{"raydium", "2YxPyAJNfnBLrVpBwMx7qMVNPSvBDhxiquwJGhBjwXhkP6i6AbooUg4b4wpi15bQq2Qs4t7BpL1UVvTMcXL8P4uS", types.PoolEventTypeCreate, "GGJZ7ayp5urhWaTikqFvm3ahggobmaEtz7wgettXrr31", "3ZmaEAKC75ereKNTxZeaDxxHLNWag2rL43UY19Rw6v7D", 50000000.0, "So11111111111111111111111111111111111111112", 1.0},                  // Singer > Raydium Liquidity Pool V4: initialize2
	{"raydium", "FHz3LurEFNnWREXSfqpenJTRuzybQrmxsNjndmMDw36XUwUofSQ1vRQwVi6uT422Xe2Whb7gfFY3tHGqvEBg6dF", types.PoolEventTypeCreate, "GmF1c1CSwm56Hx2mfJnjK8WT5eNCuDyHrHDbBYiM3U5f", "56dtyR73VF1MKHLVusZgTkAvEeDYkQqneqZqkusLgyb4", 50000000.0, "So11111111111111111111111111111111111111112", 1.0},                   // Singer > Raydium Liquidity Pool V4: initialize2
	{"raydium", "5MRWUUoCWpaFm8B9jSoLp4w6B19p46jcJMQrm26SHHGRZQpAtG3mdEcqGVDwsgUEGKRmC1R6JMFS5NhzmXUnR3X1", types.PoolEventTypeCreate, "GHrKkupnWVxQokfCfrYtmQMKnVE5s9873gJCg3wGrcTw", "591irbGXZLfqDneLqwDHLJFgS9UxxVzBmyFkWPeTmoon", 187674338.64251745, "So11111111111111111111111111111111111111112", 84.116339594}, // Moonit > Raydium Liquidity Pool V4: initialize2
	{"raydium", "4998xRsghpLWWcZsFFtfN8UBss9SVkN8sMUPkqdBzYS6eRVuxve3RoT89pqBaYvHDgqPSsFt9GDGQc6Us3pwPX5v", types.PoolEventTypeCreate, "Bx5pAzBqbuYH5yT93XCoShDka6d6vz9AzBddG7zE9syh", "7oBYdEhV4GkXC19ZfgAvXpJWp2Rn9pm1Bx2cVNxFpump", 206900000.0, "So11111111111111111111111111111111111111112", 79.005359057},        // Pumpfun > Raydium Liquidity Pool V4: initialize2
	{"raydium", "49ejTjaCdqV3hwAs1GomGsTLqZNUWztHduHRCNr8m3Dogfyii29qkNPNauAr9VfEh9j5m7QHq8HYRd6FiaAACCf", types.PoolEventTypeCreate, "Fidv7mwYxFgRVL3qHFJrGQmrhiwn6bvso7pUqU1r1db8", "npvXPTd172PepW5HMdgayiNuqKqbSsd3DArCeeTpump", 206900000.0, "So11111111111111111111111111111111111111112", 79.005359118},          // Pumpfun > Raydium Liquidity Pool V4: initialize2
	{"raydium", "4zFeXoUVaaQ18chkY899hvvTYBRMAJ6CaNfWxbchMDDEY1L56maqwAdY19Faif5LBoTxwBPeEamcEsh76b39fjia", types.PoolEventTypeAdd, "Bx5pAzBqbuYH5yT93XCoShDka6d6vz9AzBddG7zE9syh", "7oBYdEhV4GkXC19ZfgAvXpJWp2Rn9pm1Bx2cVNxFpump", 1950.837707, "So11111111111111111111111111111111111111112", 0.147754335},            // Singer > Raydium Liquidity Pool V4: raydium:add-liquidity
	{"raydium", "2S4DdkD4FpqazTn5qHd4x9X5bAHu9g5Ry3jCLWkE44UCmW8bwkcet9eeAsHbzyR7HWtiE3gV268MtsBNrc4X6KpL", types.PoolEventTypeAdd, "3bC2e2RxcfvF9oP22LvbaNsVwoS2T98q6ErCRoayQYdq", "FeR8VBqNRSUD5NtXAj2n3j1dAHkZHfyDktKuLXD4pump", 10279267.251303, "So11111111111111111111111111111111111111112", 370.892851447},      // Singer > Raydium Liquidity Pool V4: raydium:add-liquidity
	{"raydium", "2MvpoPWEY3gnEE5WxsQATRRa15Go6p8HxBbuxATiTUMxCRJeVQsBCPnCRdrM5YModikwpTiKu1iZbPBeTdHkg3uv", types.PoolEventTypeRemove, "Bx5pAzBqbuYH5yT93XCoShDka6d6vz9AzBddG7zE9syh", "7oBYdEhV4GkXC19ZfgAvXpJWp2Rn9pm1Bx2cVNxFpump", 696.848377, "So11111111111111111111111111111111111111112", 0.055709162},          // Singer > Raydium Liquidity Pool V4: raydium:add-liquidity
	{"raydium", "33HpWkDo8tr4r3jQCnML7ojpL3pFLHiE5yLZsbJDGe1mQzdg7aeJ9EKyD6WayEY4Q1hEivca93bP82TzsVcLzkgf", types.PoolEventTypeRemove, "3bC2e2RxcfvF9oP22LvbaNsVwoS2T98q6ErCRoayQYdq", "FeR8VBqNRSUD5NtXAj2n3j1dAHkZHfyDktKuLXD4pump", 12152.01771, "So11111111111111111111111111111111111111112", 0.375727077},         // Singer > Raydium Liquidity Pool V4: raydium:remove-liquidity
}

// Raydium CLMM corrections to the upstream expectations: upstream commit
// 60d4b47 (issue #85, RaydiumCLPoolV2Parser) turned open_position* into ADD
// events and made create_pool a CREATE event without amounts, but did not
// update liquidity-raydium-cl.test.ts. Truth per the raydium_clmm IDL:
//   - 61auheJ9 opens a position (open_position_v2) in an existing pool: ADD.
//   - 4Vv9ZWLi creates the pool (create_pool, outer 2) and opens the first
//     position (open_position_with_token22_nft, outer 5): CREATE without
//     amounts plus an ADD with the deposited amounts.
var clmmCreateCases = []liquidityCase{
	{"raydium-cl", "4Vv9ZWLizvRE7um22gF8bUWvD5UfK1TMXsP4hF8TVF4gc2BNmmPG8kFu7Dyod9Zw5x16xAsGeDJnUznCwaKXim5n", types.PoolEventTypeAdd, "GQsPr4RJk9AZkkfWHud7v4MtotcxhaYzZHdsPCg9vNvW", "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN", 46, "So11111111111111111111111111111111111111112", 0.459999999},
	{"raydium-cl", "PGYHMva3trxE1BR2Qr4H8DZDcSLeHdHcCgJT2iAarvrE9xovBu5dtrNiti3xXxUKoNgT9WeifFPA6zX8DjsASdP", types.PoolEventTypeCreate, "FjQygBZbvzUd8DC1H83JZApYKSJctv5mKecjqN8eiDAz", "54Srar54dHLayGxHxPswBLaseTdCGcV4c8GaHuR3C4Xv", 0, "So11111111111111111111111111111111111111112", 0},
}

func liquidityConfig() *types.ParseConfig {
	return &types.ParseConfig{ParseType: types.ParseType{Liquidity: true}}
}

func amountEqual(got *float64, want float64) bool {
	if got == nil {
		return want == 0
	}
	return math.Abs(*got-want) <= 1e-9*math.Max(1, math.Abs(want))
}

func matchLiquidity(e types.PoolEvent, c liquidityCase) bool {
	return e.Type == c.typ && e.PoolId == c.pool && e.Token0Mint == c.mint0 && e.Token1Mint == c.mint1 &&
		amountEqual(e.Token0Amount, c.amount0) && amountEqual(e.Token1Amount, c.amount1)
}

func checkLiquidityCases(t *testing.T, cases []liquidityCase) {
	t.Helper()
	parser := dexparser.NewDexParser()
	for _, c := range cases {
		c := c
		t.Run(c.suite+"/"+c.sig[:12], func(t *testing.T) {
			events := parser.ParseLiquidity(loadFixture(t, c.sig), liquidityConfig())
			for _, e := range events {
				if matchLiquidity(e, c) {
					return
				}
			}
			t.Errorf("want %s pool=%s %s %v / %s %v", c.typ, c.pool, c.mint0, c.amount0, c.mint1, c.amount1)
			for _, e := range events {
				t.Logf("got %s pool=%s idx=%s %s %v / %s %v", e.Type, e.PoolId, e.Idx, e.Token0Mint, deref(e.Token0Amount), e.Token1Mint, deref(e.Token1Amount))
			}
		})
	}
}

func deref(f *float64) interface{} {
	if f == nil {
		return nil
	}
	return *f
}

// TestLiquidityUpstreamFixtures runs the upstream liquidity fixtures of all
// supported pools (Raydium V4/CPMM/CLMM, Orca, Meteora DLMM/DAMM v1) through
// DexParser.ParseLiquidity (amm-4, amm-5, amm-10, amm-13, amm-20, parity-3,
// parity-8).
func TestLiquidityUpstreamFixtures(t *testing.T) {
	checkLiquidityCases(t, upstreamLiquidityCases)
	checkLiquidityCases(t, clmmCreateCases)
}

// dammV2LiquidityCase is an expected Meteora DAMM v2 liquidity event with raw
// amounts and idx, from upstream liquidity-meteora-damm.test.ts.
type dammV2LiquidityCase struct {
	sig        string
	idx        string
	typ        types.PoolEventType
	pool       string
	mint0      string
	amount0Raw string
	mint1      string
	amount1Raw string
}

// TestLiquidityDammV2UpstreamFixtures checks the upstream DAMM v2 fixtures.
// The pool creations (initialize_pool) deposit after the EvtCreatePosition
// self-CPI; 5S7ikDVh claims the position fees (idx 1) and then removes all
// liquidity (idx 2).
func TestLiquidityDammV2UpstreamFixtures(t *testing.T) {
	const sol = "So11111111111111111111111111111111111111112"
	cases := []dammV2LiquidityCase{
		{"3874qjiBkmSNk3rRMEst2fAfwSx9jPNNi3sCcFBxETzEYxpPeRnU9emKz26M2x3ttxJGJmjV4ctZziQMFmDgKBkZ", "3", types.PoolEventTypeCreate, "65xR8hF9DVBZ2V7CoWNmAQ5YRE9XWJE6HGomHcLQnQ7e", "ArP293mAg3ons5UQtAnesqCrTBGSuUyxuE6hZWfPpump", "737317749263", sol, "502886096"},
		{"3qiyKhA1zXD6Mvu7sxDfi8CpY5pfGhTJbRSjSPdsTmmr5kapM4EMYfqyWvAWxVLeUa53BtbxxxXMEar3wbDtaD9r", "4", types.PoolEventTypeCreate, "8g9K8u5U1HxhvM7M6Mv8jq8p8NpVGRLUB6Gi6D46QbPG", "GVDaS4pKd7GAmtPd8oDy98ksqfuKbKd29HiTTwRoiMzr", "197482850000000014", sol, "78993139600"},
		{"5A4t1CD7GwU6yPyzG9tDbyKkBAKTQpWaWL1Lc7LaXvh5ypAWoqmLuRM8TFJ1GkZhvveYvFc3HeGntNSGqQmZhoTg", "2", types.PoolEventTypeCreate, "6Xzp9hP9mwabA7bWvRXwWuKeZ6YrLgUbtoY4sA87zmbx", "6zrwCY2XE9RLL78gTQmAQfczT1XsVXmB9WQW2TawBunj", "1000000000", sol, "0"},
		{"3qCwS4QxAF9E3SCiMGk8j2wWsTve4AYVbwcXUzYxB4H4xmyGJsahDUmbAY4oo16YSQXozvY4VsnyTZ8dmm4jAaso", "1", types.PoolEventTypeAdd, "DiQWNk7aLpfF5XNwwfWfWVoSvzsSonz9B4unwW7CFsAB", "Cf1ZjYZi5UPbAyC7LhLkJYvebxrwam4AWVacymaBbonk", "23190961833072", "CDBdbNqmrLu1PcgjrFG52yxg71QnFhBZcUE6PSFdbonk", "67012807"},
		{"59mLJKYYhqauwRCrTRqoQGfF3aucjtf7MZ9MvbxTRuV7Nt1eBGXn3XdrcYMqqvYfSY3wSHHUVg918rUiDJ2vLqEk", "5", types.PoolEventTypeAdd, "8g9K8u5U1HxhvM7M6Mv8jq8p8NpVGRLUB6Gi6D46QbPG", "GVDaS4pKd7GAmtPd8oDy98ksqfuKbKd29HiTTwRoiMzr", "104044154629", sol, "440912"},
		{"5S7ikDVhmBxHiRoVxCECYr9NZzXgoUaJKxSNmqcvfDTVh2FVnCQfyo8sQoUcFdAjBgGNsSHYCBD8vadhf7k2kQ3w", "1", types.PoolEventTypeRemove, "7U2CcTGM2qPXfhRi2XytcWPXYP6nRkNaNyFaB93uBXxb", "7C5SPivzYtk2umHScjLXN7WZnstLFf1r5TCsMNaTpump", "10385838519", sol, "56694087"},
		{"5S7ikDVhmBxHiRoVxCECYr9NZzXgoUaJKxSNmqcvfDTVh2FVnCQfyo8sQoUcFdAjBgGNsSHYCBD8vadhf7k2kQ3w", "2", types.PoolEventTypeRemove, "7U2CcTGM2qPXfhRi2XytcWPXYP6nRkNaNyFaB93uBXxb", "7C5SPivzYtk2umHScjLXN7WZnstLFf1r5TCsMNaTpump", "28880994227987", sol, "36532330648"},
	}
	parser := dexparser.NewDexParser()
	for _, c := range cases {
		c := c
		t.Run(c.sig[:12]+"/"+c.idx, func(t *testing.T) {
			events := parser.ParseLiquidity(loadFixture(t, c.sig), liquidityConfig())
			for _, e := range events {
				if e.Idx == c.idx && e.Type == c.typ && e.PoolId == c.pool && e.Token0Mint == c.mint0 && e.Token1Mint == c.mint1 &&
					rawOrZero(e.Token0AmountRaw) == c.amount0Raw && rawOrZero(e.Token1AmountRaw) == c.amount1Raw {
					return
				}
			}
			t.Errorf("want %s idx=%s pool=%s %s %s / %s %s", c.typ, c.idx, c.pool, c.mint0, c.amount0Raw, c.mint1, c.amount1Raw)
			for _, e := range events {
				t.Logf("got %s idx=%s pool=%s %s %s / %s %s", e.Type, e.Idx, e.PoolId, e.Token0Mint, e.Token0AmountRaw, e.Token1Mint, e.Token1AmountRaw)
			}
		})
	}
}

func rawOrZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}
