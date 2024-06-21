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
- `SubscribeSlots`
  - `UnsubscribeSlots`
- `SubscribeTransaction`
  - `UnsubscribeTransaction`
- `SubscribeTransactionStatus`
- `SubscribeBlocks`
- `SubscribeBlocksMeta`
- `SubscribeEntry`
- `SubscribeAccountDataSlice`

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
  "github.com/weeaa/goyser/pb"
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

  if err = stream.SubscribeSlots("slots", &pb.SubscribeRequestFilterSlots{}); err != nil {
    log.Fatal(err)
  }
  if err = stream.SubscribeAccounts("accounts", &pb.SubscribeRequestFilterAccounts{
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
