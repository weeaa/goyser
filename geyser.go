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
	Streams              map[string]*StreamClient
	DefaultStreamClient  *StreamClient
	DefaultAccountFilter *pb.SubscribeRequestFilterAccounts
	mu                   sync.Mutex

	ErrCh chan error
}

type StreamClient struct {
	Ctx     context.Context
	Geyser  pb.Geyser_SubscribeClient
	Request *pb.SubscribeRequest

	Ch    <-chan *pb.SubscribeUpdate
	ErrCh <-chan error
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
		Streams:  make(map[string]*StreamClient),
		DefaultStreamClient: &StreamClient{
			Geyser: subscribe,
			Ctx:    ctx,
			Ch:     make(chan *pb.SubscribeUpdate),
			ErrCh:  make(chan error),
		},
		ErrCh: chErr,
	}, nil
}

// NewSubscribeClient creates a new Geyser subscribe stream client.
func (c *Client) NewSubscribeClient(name string, ctx context.Context) error {
	stream, err := c.Geyser.Subscribe(ctx)
	if err != nil {
		return err
	}

	streamClient := &StreamClient{
		Ctx:    ctx,
		Geyser: stream,
		Request: &pb.SubscribeRequest{
			Accounts:           make(map[string]*pb.SubscribeRequestFilterAccounts),
			Slots:              make(map[string]*pb.SubscribeRequestFilterSlots),
			Transactions:       make(map[string]*pb.SubscribeRequestFilterTransactions),
			TransactionsStatus: make(map[string]*pb.SubscribeRequestFilterTransactions),
			Blocks:             make(map[string]*pb.SubscribeRequestFilterBlocks),
			BlocksMeta:         make(map[string]*pb.SubscribeRequestFilterBlocksMeta),
			Entry:              make(map[string]*pb.SubscribeRequestFilterEntry),
			AccountsDataSlice:  make([]*pb.SubscribeRequestAccountsDataSlice, 0),
		},
		Ch:    make(chan *pb.SubscribeUpdate),
		ErrCh: make(chan error),
	}

	c.Streams[name] = streamClient
	streamClient.listen()

	return nil
}

func (c *Client) SetDefaultSubscribeClient(client pb.Geyser_SubscribeClient) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DefaultStreamClient.Geyser = client
	return c
}

func (s *StreamClient) SubscribeAccounts(id string, req *pb.SubscribeRequestFilterAccounts) error {
	s.Request.Accounts[id] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) SubscribeSlots(id string, req *pb.SubscribeRequestFilterSlots) error {
	s.Request.Slots[id] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) SubscribeTransaction(id string, req *pb.SubscribeRequestFilterTransactions) error {
	s.Request.Transactions[id] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) SubscribeTransactionStatus(id string, req *pb.SubscribeRequestFilterTransactions) error {
	s.Request.TransactionsStatus[id] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) SubscribeBlocks(id string, req *pb.SubscribeRequestFilterBlocks) error {
	s.Request.Blocks[id] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) SubscribeBlocksMeta(id string, req *pb.SubscribeRequestFilterBlocksMeta) error {
	s.Request.BlocksMeta[id] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) SubscribeEntry(id string, req *pb.SubscribeRequestFilterEntry) error {
	s.Request.Entry[id] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) SubscribeAccountDataSlice(req []*pb.SubscribeRequestAccountsDataSlice) error {
	s.Request.AccountsDataSlice = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) listen() {
	s.Ch, s.ErrCh = s.Geyser.Response()
}
