package fumarole_geyser

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/weeaa/goyser/fumarole_geyser/pb"
	"google.golang.org/grpc/metadata"
	"log"
	"os"
	"testing"
	"time"
)

func TestGeyser(t *testing.T) {
	var ctx = context.Background()

	fumaroleDialURL, ok := os.LookupEnv("FUMAROLE_GRPC")
	assert.True(t, ok)
	assert.NotEqual(t, fumaroleDialURL, "")

	fumaroleAuth, ok := os.LookupEnv("FUMAROLE_AUTH")
	assert.True(t, ok)
	assert.NotEqual(t, fumaroleAuth, "")

	md := metadata.New(map[string]string{"x-token": fumaroleAuth})
	ctx = metadata.NewOutgoingContext(ctx, md)

	client, err := New(ctx, fumaroleDialURL, nil)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	stream, err := client.AddStreamClient(ctx, "main")
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	groupLabel := uuid.NewString()
	grp, err := stream.CreateStaticConsumerGroup(groupLabel, 1, pb.InitialOffsetPolicy_LATEST, pb.CommitmentLevel_CONFIRMED, pb.EventSubscriptionPolicy_BOTH, 0)
	assert.NoError(t, err)
	assert.NotNil(t, grp)

	err = stream.Subscribe(&pb.SubscribeRequest{
		ConsumerGroupLabel: groupLabel,
		Accounts: map[string]*pb.SubscribeRequestFilterAccounts{
			"USDC": {
				Account: []string{"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"},
			},
		},
	})
	assert.NoError(t, err)

	for data := range stream.Ch {
		log.Println(time.Now().Unix(), data)
	}
}
