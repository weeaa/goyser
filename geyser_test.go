package goyser

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/weeaa/goyser/pb"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	_, filename, _, _ := runtime.Caller(0)
	godotenv.Load(filepath.Join(filepath.Dir(filename), "..", "..", "..", "goyser", ".env"))
	os.Exit(m.Run())
}

func Test_GeyserClient(t *testing.T) {
	ctx := context.Background()

	client, err := New(
		ctx,
		"http://185.164.138.184:8901/",
		nil,
	)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer client.Close()

	streamName := "main"
	commitmentLevel := geyser_pb.CommitmentLevel_PROCESSED
	if err = client.AddStreamClient(ctx, streamName, &commitmentLevel); err != nil {
		t.Fatal(err)
	}

	stream := client.GetStreamClient(streamName)

	if err = stream.SubscribeAccounts("accounts", &geyser_pb.SubscribeRequestFilterAccounts{
		Account: []string{"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"},
	}); err != nil {
		t.Fatal(err)
	}

	for out := range stream.Ch {
		log.Printf("%+v", out)
		return
	}
}
