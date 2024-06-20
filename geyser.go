package goyser

import (
	"context"
	"fmt"
	"github.com/weeaa/goyser/pb"
	"google.golang.org/grpc"
	"slices"
	"sync"
)

type Client struct {
	GrpcConn            *grpc.ClientConn         // gRPC connection
	Ctx                 context.Context          // Context for cancellation and deadlines
	Geyser              pb.GeyserClient          // Geyser client
	Streams             map[string]*StreamClient // Active stream clients
	DefaultStreamClient *StreamClient            // Default stream client
	mu                  sync.Mutex               // Mutex for thread safety
	ErrCh               chan error               // Channel for errors
}

type StreamClient struct {
	Ctx     context.Context            // Context for cancellation and deadlines
	Geyser  pb.Geyser_SubscribeClient  // Geyser subscribe client
	Request *pb.SubscribeRequest       // Subscribe request
	Ch      <-chan *pb.SubscribeUpdate // Channel for updates
	ErrCh   <-chan error               // Channel for errors
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

// SetDefaultSubscribeClient sets the default subscribe client.
func (c *Client) SetDefaultSubscribeClient(client pb.Geyser_SubscribeClient) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DefaultStreamClient.Geyser = client
	return c
}

// SubscribeAccounts subscribes to account updates.
// Note: This will overwrite existing subscriptions for the given ID.
// To add new accounts without overwriting, use AppendAccounts.
func (s *StreamClient) SubscribeAccounts(id string, req *pb.SubscribeRequestFilterAccounts) error {
	s.Request.Accounts[id] = req
	return s.Geyser.Send(s.Request)
}

// AppendAccounts appends accounts to an existing subscription and sends the request.
func (s *StreamClient) AppendAccounts(id string, accounts ...string) error {
	s.Request.Accounts[id].Account = append(s.Request.Accounts[id].Account, accounts...)
	return s.Geyser.Send(s.Request)
}

// UnsubscribeAccountsByID unsubscribes from account updates by ID.
func (s *StreamClient) UnsubscribeAccountsByID(id string) {
	delete(s.Request.Accounts, id)
}

// UnsubscribeAccounts unsubscribes specific accounts.
func (s *StreamClient) UnsubscribeAccounts(id string, accounts ...string) {
	for _, account := range accounts {
		s.Request.Accounts[id].Account = slices.DeleteFunc(s.Request.Accounts[id].Account, func(a string) bool {
			return a == account
		})
	}
}

// SubscribeSlots subscribes to slot updates.
func (s *StreamClient) SubscribeSlots(id string, req *pb.SubscribeRequestFilterSlots) error {
	s.Request.Slots[id] = req
	return s.Geyser.Send(s.Request)
}

// UnsubscribeSlots unsubscribes from slot updates.
func (s *StreamClient) UnsubscribeSlots(id string) {
	delete(s.Request.Slots, id)
}

// SubscribeTransaction subscribes to transaction updates.
func (s *StreamClient) SubscribeTransaction(id string, req *pb.SubscribeRequestFilterTransactions) error {
	s.Request.Transactions[id] = req
	return s.Geyser.Send(s.Request)
}

// UnsubscribeTransaction unsubscribes from transaction updates.
func (s *StreamClient) UnsubscribeTransaction(id string) {
	delete(s.Request.Transactions, id)
}

// SubscribeTransactionStatus subscribes to transaction status updates.
func (s *StreamClient) SubscribeTransactionStatus(id string, req *pb.SubscribeRequestFilterTransactions) error {
	s.Request.TransactionsStatus[id] = req
	return s.Geyser.Send(s.Request)
}

// SubscribeBlocks subscribes to block updates.
func (s *StreamClient) SubscribeBlocks(id string, req *pb.SubscribeRequestFilterBlocks) error {
	s.Request.Blocks[id] = req
	return s.Geyser.Send(s.Request)
}

// SubscribeBlocksMeta subscribes to block metadata updates.
func (s *StreamClient) SubscribeBlocksMeta(id string, req *pb.SubscribeRequestFilterBlocksMeta) error {
	s.Request.BlocksMeta[id] = req
	return s.Geyser.Send(s.Request)
}

// SubscribeEntry subscribes to entry updates.
func (s *StreamClient) SubscribeEntry(id string, req *pb.SubscribeRequestFilterEntry) error {
	s.Request.Entry[id] = req
	return s.Geyser.Send(s.Request)
}

// SubscribeAccountDataSlice subscribes to account data slice updates.
func (s *StreamClient) SubscribeAccountDataSlice(req []*pb.SubscribeRequestAccountsDataSlice) error {
	s.Request.AccountsDataSlice = req
	return s.Geyser.Send(s.Request)
}

// listen starts listening for responses and errors.
func (s *StreamClient) listen() {
	s.Ch, s.ErrCh = s.Geyser.Response()
}
