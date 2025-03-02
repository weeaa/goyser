package fumarole_geyser

import (
	"context"
	"errors"
	"fmt"
	"github.com/weeaa/goyser/fumarole_geyser/pb"
	"github.com/weeaa/goyser/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"sync"
	"time"
)

// Client represents a client for the Fumarole service.
type Client struct {
	Ctx      context.Context
	Cancel   context.CancelFunc
	GrpcConn *grpc.ClientConn
	Fumarole pb.FumaroleClient
	ErrCh    chan error
	s        *streamManager
}

// streamManager manages multiple stream clients.
type streamManager struct {
	clients map[string]*StreamClient
	mu      sync.RWMutex
}

// StreamClient represents a client for a specific stream.
type StreamClient struct {
	Ctx          context.Context
	Cancel       context.CancelFunc
	GrpcConn     *grpc.ClientConn
	streamName   string
	fumarole     pb.Fumarole_SubscribeClient
	fumaroleConn pb.FumaroleClient
	request      *pb.SubscribeRequest
	Ch           chan *pb.SubscribeUpdate
	ErrCh        chan error
	mu           sync.RWMutex
	countMu      sync.RWMutex
	count        int32
	latestCount  time.Time
}

// New creates a new Client instance.
func New(ctx context.Context, grpcDialURL string, md metadata.MD, opts ...grpc.DialOption) (*Client, error) {
	ch := make(chan error)

	if md != nil {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	conn, err := pkg.CreateAndObserveGRPCConn(ctx, ch, grpcDialURL, opts...)
	if err != nil {
		return nil, err
	}

	fumaroleClient := pb.NewFumaroleClient(conn)
	if fumaroleClient == nil {
		return nil, errors.New("fumarole client is equal to nil")
	}

	client := &Client{
		GrpcConn: conn,
		Ctx:      ctx,
		Fumarole: fumaroleClient,
		ErrCh:    ch,
		s: &streamManager{
			clients: make(map[string]*StreamClient),
			mu:      sync.RWMutex{},
		},
	}

	return client, nil
}

// Close closes the client and all the streams.
func (c *Client) Close() error {
	for _, sc := range c.s.clients {
		sc.Stop()
	}
	close(c.ErrCh)
	return c.GrpcConn.Close()
}

// AddStreamClient creates a new Fumarole subscribe stream client.
func (c *Client) AddStreamClient(ctx context.Context, streamName string, opts ...grpc.CallOption) (*StreamClient, error) {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	if _, exists := c.s.clients[streamName]; exists {
		return nil, fmt.Errorf("client with name %s already exists", streamName)
	}

	opts = append(opts, grpc.MaxCallRecvMsgSize(100*1024*1024))
	stream, err := c.Fumarole.Subscribe(ctx, opts...)
	if err != nil {
		return nil, err
	}

	streamClient := StreamClient{
		Ctx:          ctx,
		GrpcConn:     c.GrpcConn,
		streamName:   streamName,
		fumarole:     stream,
		fumaroleConn: c.Fumarole,
		request:      &pb.SubscribeRequest{},
		Ch:           make(chan *pb.SubscribeUpdate),
		ErrCh:        make(chan error),
		mu:           sync.RWMutex{},
	}

	c.s.clients[streamName] = &streamClient
	go streamClient.listen()

	return &streamClient, nil
}

func (s *StreamClient) Stop() {
	s.Ctx.Done()
	close(s.Ch)
	close(s.ErrCh)
}

func (s *StreamClient) listen() {
	for {
		select {
		case <-s.Ctx.Done():
			return
		default:
			recv, err := s.fumarole.Recv()
			if err != nil {
				s.ErrCh <- err
				continue
			}

			s.Ch <- recv
		}
	}
}

func (s *StreamClient) GetConsumerGroupInfo(label string) (*pb.ConsumerGroupInfo, error) {
	return s.fumaroleConn.GetConsumerGroupInfo(s.Ctx, &pb.GetConsumerGroupInfoRequest{
		ConsumerGroupLabel: label,
	})
}

func (s *StreamClient) CreateStaticConsumerGroup(
	label string,
	memberCount uint32,
	initialOffsetPolicy pb.InitialOffsetPolicy,
	commitmentLevel pb.CommitmentLevel,
	eventSubPolicy pb.EventSubscriptionPolicy,
	atSlot int64,
) (*pb.CreateStaticConsumerGroupResponse, error) {
	return s.fumaroleConn.CreateStaticConsumerGroup(s.Ctx, &pb.CreateStaticConsumerGroupRequest{
		ConsumerGroupLabel:      label,
		MemberCount:             &memberCount,
		InitialOffsetPolicy:     initialOffsetPolicy,
		CommitmentLevel:         commitmentLevel,
		EventSubscriptionPolicy: eventSubPolicy,
		AtSlot:                  &atSlot,
	})
}

func (s *StreamClient) DeleteConsumerGroup(label string) (*pb.DeleteConsumerGroupResponse, error) {
	return s.fumaroleConn.DeleteConsumerGroup(s.Ctx, &pb.DeleteConsumerGroupRequest{
		ConsumerGroupLabel: label,
	})
}

func (s *StreamClient) ListConsumerGroups() (*pb.ListConsumerGroupsResponse, error) {
	return s.fumaroleConn.ListConsumerGroups(s.Ctx, &pb.ListConsumerGroupsRequest{})
}

func (s *StreamClient) GetOldestSlot(commitmentLevel pb.CommitmentLevel) (*pb.GetOldestSlotResponse, error) {
	return s.fumaroleConn.GetOldestSlot(s.Ctx, &pb.GetOldestSlotRequest{
		CommitmentLevel: commitmentLevel,
	})
}

func (s *StreamClient) GetSlotLagInfo(label string) (*pb.GetSlotLagInfoResponse, error) {
	return s.fumaroleConn.GetSlotLagInfo(s.Ctx, &pb.GetSlotLagInfoRequest{
		ConsumerGroupLabel: label,
	})
}

func (s *StreamClient) ListAvailableCommitmentLevels() (*pb.ListAvailableCommitmentLevelsResponse, error) {
	return s.fumaroleConn.ListAvailableCommitmentLevels(s.Ctx, &pb.ListAvailableCommitmentLevelsRequest{})
}

func (s *StreamClient) Subscribe(req *pb.SubscribeRequest) error {
	return s.fumarole.Send(req)
}
