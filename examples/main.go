package examples

import (
	"context"
	"github.com/weeaa/goyser"
	"log"
	"os"
)

func main() {
	ctx := context.Background()
	block := make(chan struct{})

	client, err := goyser.New(ctx, os.Getenv("GEYSER_RPC"))
	if err != nil {
		log.Fatal(err)
	}

	streamClient, err := client.NewSubscribeClient("main", ctx)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for stream := range streamClient.Ch {
			log.Printf("received > %+v", stream)
		}
	}()

	client.SetDefaultSubscribeClient(streamClient.Geyser)
	if err = streamClient.AddAccounts("winners", "0x...", "0x..."); err != nil {
		log.Fatal(err)
	}

	<-block
}
