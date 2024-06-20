# Geyser Yellowstone SDK
[![GoDoc](https://pkg.go.dev/badge/github.com/weeaa/goyser?status.svg)](https://pkg.go.dev/github.com/weeaa/goyser?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/weeaa/goyser)](https://goreportcard.com/report/github.com/weeaa/goyser)
[![License](https://img.shields.io/badge/license-Apache_2.0-crimson)](https://opensource.org/license/apache-2-0)

This library contains tooling to interact with **[Yellowstone Geyser Plugin](https://docs.solanalabs.com/validator/geyser)**. Work in progress. 👷

![yellowstone](https://files.oaiusercontent.com/file-cg9YkbHcSC5kXatlKK6WLG7i?se=2024-06-20T11%3A48%3A12Z&sp=r&sv=2023-11-03&sr=b&rscc=max-age%3D31536000%2C%20immutable&rscd=attachment%3B%20filename%3D198eb09f-2fb3-4e19-9ff2-f9c813cddeb9.webp&sig=19er0erTrnglRhYISe/nJLOoubqswjgo71so/JVadyI%3D)

## ❇️ Contents
- [Methods](#-methods)
- [Installing](#-installing)
- [Examples](#-examples)
  - [Subscribe to Slots & Account](#subscribe-to-slots-and-account)
- [Disclaimer](#-disclaimer)
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
      log.Printf("slot updata: %+v", out.GetSlot())
    case out.GetTransaction() != nil:
      log.Printf("tx updata: %+v", out.GetTransaction())
    default:
      log.Printf("other updqte: %+v", out)
    }
  }
}
```

## 🚨 Disclaimer

**This library is not affiliated with Jito Labs**. It is a community project and is not officially supported by Jito Labs. Use at your own risk.

## 🛟 Support
If my work has been useful in building your for-profit services/infra/bots/etc, consider donating at
`EcrHvqa5Vh4NhR3bitRZVrdcUGr1Z3o6bXHz7xgBU2FB` (SOL).

## 📃 License

[Apache-2.0 License](https://github.com/weeaa/jito-go/blob/main/LICENSE).