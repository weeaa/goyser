package goyser

import (
	"context"
	"fmt"
	"github.com/weeaa/goyser/pb"
	"google.golang.org/grpc"
	"sync"
)

type Client struct {
	GrpcConn *grpc.ClientConn
	Ctx      context.Context

	Geyser               pb.GeyserClient
	DefaultStreamClient  *StreamClient
	DefaultAccountFilter *pb.SubscribeRequestFilterAccounts
	mu                   sync.Mutex

	ErrCh chan error
}

type StreamClient struct {
	Ctx     context.Context
	Geyser  pb.Geyser_SubscribeClient
	Request *pb.SubscribeRequest
	Ch      chan *pb.SubscribeUpdate
	ErrCh   chan<- error
}

func New(ctx context.Context, grpcDialURL string) (*Client, error) {
	chErr := make(chan error)
	conn, err := createAndObserveGRPCConn(ctx, chErr, grpcDialURL)
	if err != nil {
		return nil, err
	}

	geyserClient := pb.NewGeyserClient(conn)
	subscribe, err := geyserClient.Subscribe(
		ctx,
		grpc.MaxCallRecvMsgSize(16*1024*1024),
		grpc.MaxCallSendMsgSize(16*1024*1024),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating default subscribe client: %w", err)
	}

	return &Client{
		GrpcConn: conn,
		Ctx:      ctx,
		Geyser:   geyserClient,
		DefaultStreamClient: &StreamClient{
			Geyser: subscribe,
			Ctx:    ctx,
			Ch:     make(chan *pb.SubscribeUpdate),
			ErrCh:  make(chan error),
		},
		ErrCh: chErr,
	}, nil
}

// NewSubscribeClient creates a new Geyser stream client.
func (c *Client) NewSubscribeClient(name string, ctx context.Context) (*StreamClient, error) {
	stream, err := c.Geyser.Subscribe(ctx)
	if err != nil {
		return nil, err
	}

	return &StreamClient{
		Ctx:    ctx,
		Geyser: stream,
		Ch:     make(chan *pb.SubscribeUpdate),
	}, nil
}

func (c *Client) SetDefaultSubscribeClient(client pb.Geyser_SubscribeClient) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DefaultStreamClient.Geyser = client
	return c
}

func (s *StreamClient) AddAccounts(id string, accounts ...string) error {
	s.Request.Accounts[id] = &pb.SubscribeRequestFilterAccounts{Account: accounts}
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) Listen() {
	for {
		select {
		case <-s.Ctx.Done():
			s.ErrCh <- s.Ctx.Err()
		default:
			recv, err := s.Geyser.Recv()
			if err != nil {
				s.ErrCh <- err
				continue
			}

			go s.dispatch(recv)
		}
	}
}

func (s *StreamClient) dispatch(update *pb.SubscribeUpdate) {
	s.Ch <- update
}
