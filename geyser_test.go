package goyser

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/weeaa/goyser/pb"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	_, filename, _, _ := runtime.Caller(0)
	godotenv.Load(filepath.Join(filepath.Dir(filename), "..", "..", "..", "goyser", ".env"))
	os.Exit(m.Run())
}

func Test_GeyserClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	rpcAddr, ok := os.LookupEnv("GEYSER_RPC")
	if !assert.True(t, ok, "getting GEYSER_RPC from .env") {
		t.FailNow()
	}

	if !assert.NotEqualf(t, "", rpcAddr, "GEYSER_RPC shouldn't be equal to [%s]", rpcAddr) {
		t.FailNow()
	}

	client, err := New(
		ctx,
		rpcAddr,
	)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer client.GrpcConn.Close()
	subscription := &pb.SubscribeRequest{}

	subscription.Accounts = make(map[string]*pb.SubscribeRequestFilterAccounts)
	subscription.Slots = make(map[string]*pb.SubscribeRequestFilterSlots)
	subscription.Accounts["feur"] = &pb.SubscribeRequestFilterAccounts{
		Account: []string{"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"},
	}
	subscription.Slots["slots"] = &pb.SubscribeRequestFilterSlots{}

	if err = client.DefaultStreamClient.Geyser.Send(subscription); err != nil {
		t.Fatal(err)
	}

	var hb int32 = 0
	for {
		r, err := client.DefaultStreamClient.Geyser.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if err == io.EOF {
			break
		}

		if r.GetPing() != nil {
			hb++
			client.DefaultStreamClient.Geyser.Send(&pb.SubscribeRequest{
				Ping: &pb.SubscribeRequestPing{
					Id: hb,
				},
			})
		}

		log.Printf("%+v", r)
	}
}
