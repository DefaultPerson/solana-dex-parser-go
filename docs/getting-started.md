# Getting Started

## Prerequisites

- Go 1.21 or higher

## Installation

```bash
go get github.com/DefaultPerson/solana-dex-parser-go
```

## Quick Start

### Parse All (Trades, Liquidity and Transfers)

Parse all types of transactions including DEX trades, liquidity operations, and token transfers.

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"

    dexparser "github.com/DefaultPerson/solana-dex-parser-go"
    "github.com/DefaultPerson/solana-dex-parser-go/adapter"
)

func main() {
    // Get transaction from RPC
    signature := "4Cod1cNGv6RboJ7rSB79yeVCR4Lfd25rFgLY3eiPJfTJjTGyYP1r2i1upAYZHQsWDqUbGd1bhTRm1bpSQcpWMnEz"
    tx, _ := getTransaction(signature, "https://api.mainnet-beta.solana.com")

    // Parse all types of transactions in one call
    parser := dexparser.NewDexParser()
    result := parser.ParseAll(tx, nil)

    fmt.Printf("Trades: %d\n", len(result.Trades))
    fmt.Printf("Liquidities: %d\n", len(result.Liquidities))
    fmt.Printf("Transfers: %d\n", len(result.Transfers))
    fmt.Printf("MemeEvents: %d\n", len(result.MemeEvents))
}

// getTransaction fetches transaction from Solana RPC
func getTransaction(sig, rpc string) (*adapter.SolanaTransaction, error) {
    payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"getTransaction","params":["%s",{"encoding":"jsonParsed","maxSupportedTransactionVersion":0}]}`, sig)
    resp, err := http.Post(rpc, "application/json", strings.NewReader(payload))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    var rpcResp struct {
        Result *adapter.SolanaTransaction `json:"result"`
    }
    json.Unmarshal(body, &rpcResp)
    return rpcResp.Result, nil
}
```

### Configuration Options

```go
type ParseConfig struct {
    TryUnknownDEX    bool     // Try unknown DEX programs (default: true)
    ProgramIds       []string // Only parse specific program IDs
    IgnoreProgramIds []string // Ignore specific program IDs
    AggregateTrades  bool     // Aggregate multiple trades into one
}
```

### Filter by Program

```go
import "github.com/DefaultPerson/solana-dex-parser-go/constants"

config := &types.ParseConfig{
    ProgramIds: []string{
        constants.DEX_PROGRAMS.PUMP_FUN.ID,
        constants.DEX_PROGRAMS.RAYDIUM_V4.ID,
    },
}
result := parser.ParseAll(tx, config)
```

## Yellowstone gRPC (Real-time Streaming)

For high-performance real-time parsing, use Yellowstone gRPC (Helius Laserstream/Triton):

```go
package main

import (
    "context"
    "fmt"
    "log"

    pb "github.com/rpcpool/yellowstone-grpc/grpc/go"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"

    dexparser "github.com/DefaultPerson/solana-dex-parser-go"
    "github.com/DefaultPerson/solana-dex-parser-go/constants"
    "github.com/DefaultPerson/solana-dex-parser-go/types"
)

func main() {
    // Connect to Yellowstone gRPC
    conn, err := grpc.Dial(
        "laserstream-mainnet-fra.helius-rpc.com:443",
        grpc.WithTransportCredentials(credentials.NewTLS(nil)),
        grpc.WithPerRPCCredentials(&tokenAuth{"YOUR_API_KEY"}),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := pb.NewGeyserClient(conn)
    stream, err := client.Subscribe(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Subscribe to Pumpfun transactions
    stream.Send(&pb.SubscribeRequest{
        Transactions: map[string]*pb.SubscribeRequestFilterTransactions{
            "pumpfun": {
                AccountInclude: []string{constants.DEX_PROGRAMS.PUMP_FUN.ID},
            },
        },
        Commitment: pb.CommitmentLevel_CONFIRMED.Enum(),
    })

    // Parse incoming transactions
    parser := dexparser.NewDexParser()
    config := &types.ParseConfig{
        ProgramIds: []string{constants.DEX_PROGRAMS.PUMP_FUN.ID},
    }

    for {
        resp, err := stream.Recv()
        if err != nil {
            log.Fatal(err)
        }

        if tx := resp.GetTransaction(); tx != nil {
            // Convert gRPC transaction to SolanaTransaction format
            // See grpc_utils.go for ConvertYellowstoneTransaction helper
            solTx := dexparser.ConvertYellowstoneTransaction(tx.Transaction, resp.GetSlot(), 0)
            result := parser.ParseAll(solTx, config)

            for _, trade := range result.Trades {
                fmt.Printf("[%s] %s: %s -> %s\n",
                    trade.AMM, trade.Type,
                    trade.InputToken.Mint[:8],
                    trade.OutputToken.Mint[:8])
            }
        }
    }
}

type tokenAuth struct{ token string }

func (t *tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
    return map[string]string{"x-token": t.token}, nil
}

func (t *tokenAuth) RequireTransportSecurity() bool { return true }
```

### Dependencies for gRPC

```bash
go get github.com/rpcpool/yellowstone-grpc/grpc/go
go get google.golang.org/grpc
```

## Next Steps

- [Examples](examples/index.md) - Code examples for specific use cases
- [Development](development.md) - Contributing and testing
