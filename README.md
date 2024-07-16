# Geyser Yellowstone SDK
[![GoDoc](https://pkg.go.dev/badge/github.com/weeaa/goyser?status.svg)](https://pkg.go.dev/github.com/weeaa/goyser?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/weeaa/goyser)](https://goreportcard.com/report/github.com/weeaa/goyser)
[![License](https://img.shields.io/badge/license-Apache_2.0-crimson)](https://opensource.org/license/apache-2-0)


This library contains tooling to interact with **[Yellowstone Geyser Plugin](https://docs.solanalabs.com/validator/geyser)**. Work in progress. 👷

<div align="center">
  <img src="https://github.com/weeaa/goyser/assets/108926252/601185b7-3f50-4542-ae94-16488a651467" alt="yellowstone" width="500" style="border-radius: 15px;"/>
</div>

## ❇️ Contents
- [Methods](#-methods)
- [How it works](#-how-it-works)
- [Installing](#-installing)
- [Examples](#-examples)
  - [Subscribe to Slots & Account](#subscribe-to-slots-and-account)
- [Support](#-support)
- [License](#-license)

## 📡 Methods
- `SubscribeAccounts`
  - `AppendAccounts`
  - `UnsubscribeAccounts`
  - `UnsubscribeAccountsByID`
  - `UnsubscribeAllAccounts`
- `SubscribeSlots`
  - `UnsubscribeSlots`
- `SubscribeTransaction`
  - `UnsubscribeTransaction`
- `SubscribeTransactionStatus`
  - `UnsubscribeTransactionStatus`
- `SubscribeBlocks`
  - `UnsubscribeBlocks`
- `SubscribeBlocksMeta`
  - `UnsubscribeBlocksMeta`
- `SubscribeEntry`
  - `UnsubscribeEntry`
- `SubscribeAccountDataSlice`
  - `UnsubscribeAccountDataSlice`

It also contains a feature to convert Goyser types to [github.com/gagliardetto/solana-go](https://github.com/gagliardetto/solana-go) types :)

## 🧠 How it works
Simple example on how to monitor an account for transactions with explanations.
```go
package main

import (
	"context"
	"github.com/weeaa/goyser"
	"github.com/weeaa/goyser/pb"
	"log"
	"os"
	"time"
)

func main() {
	ctx := context.Background()

	// get the geyser rpc address
	geyserRPC := os.Getenv("GEYSER_RPC")

	// create geyser client
	client, err := goyser.New(ctx, geyserRPC)
	if err != nil {
		log.Fatal(err)
	}

	// create a new subscribe client which is tied, for our example we will name it main
	// the created client is stored in client.Streams
	if err = client.NewSubscribeClient(ctx, "main"); err != nil {
		log.Fatal(err)
	}

	// get the stream client
	streamClient, ok := client.Streams["main"]
	if !ok {
		log.Fatal("client does not have a stream named main")
	}

	// subscribe to the account you want to see txns from and set a custom filter name to filter them out later
	if err = streamClient.SubscribeAccounts("accounts", &geyser_pb.SubscribeRequestFilterAccounts{
		Account: []string{"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"},
	}); err != nil {
		log.Fatal(err)
	}

	// loop through the stream and print the output
	for out := range streamClient.Ch {
		// u can filter the output by checking the filters
		go func() {
			filters := out.GetFilters()
			for _, filter := range filters {
				switch filter {
				case "accounts":
					log.Printf("account filter: %+v", out.GetAccount())
				default:
					log.Printf("unknown filter: %s", filter)
				}
			}
		}()
		break
	}

	// unsubscribe from the account
	if err = streamClient.UnsubscribeAccounts("accounts", ""); err != nil {
		log.Fatal(err)
	}

	time.Sleep(5 * time.Second)
}
```


## 💾 Installing

Go 1.22.0 or higher.
```shell
go get github.com/weeaa/goyser@latest
```

## 💻 Examples

### `Subscribe to Slots and Account`
```go
package main

import (
  "context"
  "github.com/joho/godotenv"
  "github.com/weeaa/goyser"
  "github.com/weeaa/goyser/geyser_pb"
  "log"
  "os"
)

func main() {
  ctx := context.Background()

  if err := godotenv.Load(); err != nil {
    log.Fatal(err)
  }

  rpcAddr, ok := os.LookupEnv("GEYSER_RPC")
  if !ok {
    log.Fatal("could not inf GEYSER_RPC in .env")
  }

  client, err := goyser.New(
    ctx,
    rpcAddr,
  )
  if err != nil {
    log.Fatal(err)
  }
  defer client.GrpcConn.Close()

  if err = client.NewSubscribeClient("main", ctx); err != nil {
    log.Fatal(err)
  }
  defer client.DefaultStreamClient.Geyser.CloseSend()

  stream := client.Streams["main"]
  defer client.DefaultStreamClient.Geyser.CloseSend()

  if err = stream.SubscribeSlots("slots", &geyser_pb.SubscribeRequestFilterSlots{}); err != nil {
    log.Fatal(err)
  }
  if err = stream.SubscribeAccounts("accounts", &geyser_pb.SubscribeRequestFilterAccounts{
    Account: []string{"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"},
  }); err != nil {
    log.Fatal(err)
  }

  for out := range stream.Ch {
    switch {
    case out.GetSlot() != nil:
      log.Printf("slot update: %+v", out.GetSlot())
    case out.GetTransaction() != nil:
      log.Printf("tx update: %+v", out.GetTransaction())
    default:
      log.Printf("other update: %+v", out)
    }
  }
}
```

## 🛟 Support
If my work has been useful in building your for-profit services/infra/bots/etc, consider donating at
`EcrHvqa5Vh4NhR3bitRZVrdcUGr1Z3o6bXHz7xgBU2FB` (SOL).

## 📃 License

[Apache-2.0 License](https://github.com/weeaa/jito-go/blob/main/LICENSE).
