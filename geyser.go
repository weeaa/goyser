package goyser

import (
	"context"
	//"encoding/binary"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/weeaa/goyser/pb"
	"google.golang.org/grpc"
	"slices"
	"strconv"
	"sync"
)

type Client struct {
	GrpcConn            *grpc.ClientConn         // gRPC connection
	Ctx                 context.Context          // Context for cancellation and deadlines
	Geyser              geyser_pb.GeyserClient   // Geyser client
	Streams             map[string]*StreamClient // Active stream clients
	DefaultStreamClient *StreamClient            // Default stream client
	mu                  sync.Mutex               // Mutex for thread safety
	ErrCh               chan error               // Channel for errors
}

type StreamClient struct {
	Ctx     context.Context                  // Context for cancellation and deadlines
	Geyser  geyser_pb.Geyser_SubscribeClient // Geyser subscribe client
	Request *geyser_pb.SubscribeRequest      // Subscribe request
	Ch      chan *geyser_pb.SubscribeUpdate  // Channel for updates
	ErrCh   chan error                       // Channel for errors
}

func New(ctx context.Context, grpcDialURL string) (*Client, error) {
	chErr := make(chan error)
	conn, err := createAndObserveGRPCConn(ctx, chErr, grpcDialURL)
	if err != nil {
		return nil, err
	}

	geyserClient := geyser_pb.NewGeyserClient(conn)
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
			Ch:     make(chan *geyser_pb.SubscribeUpdate),
			ErrCh:  make(chan error),
		},
		ErrCh: chErr,
	}, nil
}

// NewSubscribeClient creates a new Geyser subscribe stream client.
func (c *Client) NewSubscribeClient(ctx context.Context, clientName string) error {

	stream, err := c.Geyser.Subscribe(ctx)
	if err != nil {
		return err
	}

	streamClient := &StreamClient{
		Ctx:    ctx,
		Geyser: stream,
		Request: &geyser_pb.SubscribeRequest{
			Accounts:           make(map[string]*geyser_pb.SubscribeRequestFilterAccounts),
			Slots:              make(map[string]*geyser_pb.SubscribeRequestFilterSlots),
			Transactions:       make(map[string]*geyser_pb.SubscribeRequestFilterTransactions),
			TransactionsStatus: make(map[string]*geyser_pb.SubscribeRequestFilterTransactions),
			Blocks:             make(map[string]*geyser_pb.SubscribeRequestFilterBlocks),
			BlocksMeta:         make(map[string]*geyser_pb.SubscribeRequestFilterBlocksMeta),
			Entry:              make(map[string]*geyser_pb.SubscribeRequestFilterEntry),
			AccountsDataSlice:  make([]*geyser_pb.SubscribeRequestAccountsDataSlice, 0),
		},
		Ch:    make(chan *geyser_pb.SubscribeUpdate),
		ErrCh: make(chan error),
	}

	c.Streams[clientName] = streamClient
	go streamClient.listen()

	return nil
}

func (c *Client) BatchNewSubscribeClient(ctx context.Context, clientNames ...string) error {
	for _, clientName := range clientNames {
		if err := c.NewSubscribeClient(ctx, clientName); err != nil {
			return err
		}
	}
	return nil
}

// SetDefaultSubscribeClient sets the default subscribe client.
func (c *Client) SetDefaultSubscribeClient(client geyser_pb.Geyser_SubscribeClient) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.DefaultStreamClient.Geyser = client
	return c
}

// SubscribeAccounts subscribes to account updates.
// Note: This will overwrite existing subscriptions for the given ID.
// To add new accounts without overwriting, use AppendAccounts.
func (s *StreamClient) SubscribeAccounts(filterName string, req *geyser_pb.SubscribeRequestFilterAccounts) error {
	s.Request.Accounts[filterName] = req
	return s.Geyser.Send(s.Request)
}

// AppendAccounts appends accounts to an existing subscription and sends the request.
func (s *StreamClient) AppendAccounts(filterName string, accounts ...string) error {
	s.Request.Accounts[filterName].Account = append(s.Request.Accounts[filterName].Account, accounts...)
	return s.Geyser.Send(s.Request)
}

// UnsubscribeAccountsByID unsubscribes from account updates by ID.
func (s *StreamClient) UnsubscribeAccountsByID(filterName string) error {
	delete(s.Request.Accounts, filterName)
	return s.Geyser.Send(s.Request)
}

// UnsubscribeAccounts unsubscribes specific accounts.
func (s *StreamClient) UnsubscribeAccounts(filterName string, accounts ...string) error {
	for _, account := range accounts {
		s.Request.Accounts[filterName].Account = slices.DeleteFunc(s.Request.Accounts[filterName].Account, func(a string) bool {
			return a == account
		})
	}
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) UnsubscribeAllAccounts(filterName string) error {
	delete(s.Request.Accounts, filterName)
	return s.Geyser.Send(s.Request)
}

// SubscribeSlots subscribes to slot updates.
func (s *StreamClient) SubscribeSlots(filterName string, req *geyser_pb.SubscribeRequestFilterSlots) error {
	s.Request.Slots[filterName] = req
	return s.Geyser.Send(s.Request)
}

// UnsubscribeSlots unsubscribes from slot updates.
func (s *StreamClient) UnsubscribeSlots(filterName string) error {
	delete(s.Request.Slots, filterName)
	return s.Geyser.Send(s.Request)
}

// SubscribeTransaction subscribes to transaction updates.
func (s *StreamClient) SubscribeTransaction(filterName string, req *geyser_pb.SubscribeRequestFilterTransactions) error {
	s.Request.Transactions[filterName] = req
	return s.Geyser.Send(s.Request)
}

// UnsubscribeTransaction unsubscribes from transaction updates.
func (s *StreamClient) UnsubscribeTransaction(filterName string) error {
	delete(s.Request.Transactions, filterName)
	return s.Geyser.Send(s.Request)
}

// SubscribeTransactionStatus subscribes to transaction status updates.
func (s *StreamClient) SubscribeTransactionStatus(filterName string, req *geyser_pb.SubscribeRequestFilterTransactions) error {
	s.Request.TransactionsStatus[filterName] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) UnsubscribeTransactionStatus(filterName string) error {
	delete(s.Request.TransactionsStatus, filterName)
	return s.Geyser.Send(s.Request)
}

// SubscribeBlocks subscribes to block updates.
func (s *StreamClient) SubscribeBlocks(filterName string, req *geyser_pb.SubscribeRequestFilterBlocks) error {
	s.Request.Blocks[filterName] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) UnsubscribeBlocks(filterName string) error {
	delete(s.Request.Blocks, filterName)
	return s.Geyser.Send(s.Request)
}

// SubscribeBlocksMeta subscribes to block metadata updates.
func (s *StreamClient) SubscribeBlocksMeta(filterName string, req *geyser_pb.SubscribeRequestFilterBlocksMeta) error {
	s.Request.BlocksMeta[filterName] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) UnsubscribeBlocksMeta(filterName string) error {
	delete(s.Request.BlocksMeta, filterName)
	return s.Geyser.Send(s.Request)
}

// SubscribeEntry subscribes to entry updates.
func (s *StreamClient) SubscribeEntry(filterName string, req *geyser_pb.SubscribeRequestFilterEntry) error {
	s.Request.Entry[filterName] = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) UnsubscribeEntry(filterName string) error {
	delete(s.Request.Entry, filterName)
	return s.Geyser.Send(s.Request)
}

// SubscribeAccountDataSlice subscribes to account data slice updates.
func (s *StreamClient) SubscribeAccountDataSlice(req []*geyser_pb.SubscribeRequestAccountsDataSlice) error {
	s.Request.AccountsDataSlice = req
	return s.Geyser.Send(s.Request)
}

func (s *StreamClient) UnsubscribeAccountDataSlice() error {
	s.Request.AccountsDataSlice = nil
	return s.Geyser.Send(s.Request)
}

// listen starts listening for responses and errors.
func (s *StreamClient) listen() {
	for {
		select {
		case <-s.Ctx.Done():
			return
		default:
			recv, err := s.Geyser.Recv()
			if err != nil {
				s.ErrCh <- err
			}
			s.Ch <- recv
		}
	}
}

// ConvertTransaction converts a Geyser Transaction to github.com/gagliardetto/solana-go Solana Transaction. // , []rpc.InnerInstruction
func ConvertTransaction(geyserTx *geyser_pb.SubscribeUpdateTransaction) (*solana.Transaction) {
	tx := new(solana.Transaction)

	tx.Signatures = []solana.Signature{solana.SignatureFromBytes(geyserTx.Transaction.Signature)}
	if geyserTx.Transaction.Transaction.Message.Versioned {
		tx.Message.SetVersion(1)
	}
	// header
	tx.Message.Header.NumRequiredSignatures = uint8(geyserTx.Transaction.Transaction.Message.Header.NumRequiredSignatures)
	tx.Message.Header.NumReadonlySignedAccounts = uint8(geyserTx.Transaction.Transaction.Message.Header.NumReadonlySignedAccounts)
	tx.Message.Header.NumReadonlyUnsignedAccounts = uint8(geyserTx.Transaction.Transaction.Message.Header.NumReadonlyUnsignedAccounts)

	// account keys
	accountKeys := solana.PublicKeySlice{}
	for _, accountKey := range geyserTx.Transaction.Transaction.Message.AccountKeys {
		accountKeys.Append(solana.PublicKeyFromBytes(accountKey))
	}

	tx.Message.AccountKeys = accountKeys

	//// instructions
	for _, instruction := range geyserTx.Transaction.Transaction.Message.Instructions {
		accounts := make([]uint16, len(instruction.Accounts))
		for i, account := range instruction.Accounts {
			accounts[i] = uint16(account)
		}
		tx.Message.Instructions = append(tx.Message.Instructions, solana.CompiledInstruction{
			ProgramIDIndex: uint16(instruction.ProgramIdIndex),
			Accounts:       accounts,
			Data:           instruction.Data,
		})
	}

	// address table lookup
	for _, atl := range geyserTx.Transaction.Transaction.Message.AddressTableLookups {
		tx.Message.AddressTableLookups = append(tx.Message.AddressTableLookups, solana.MessageAddressTableLookup{
			AccountKey:      solana.PublicKeyFromBytes(atl.AccountKey),
			WritableIndexes: atl.WritableIndexes,
			ReadonlyIndexes: atl.ReadonlyIndexes,
		})
	}

	tx.Message.RecentBlockhash = solana.Hash(solana.PublicKeyFromBytes(geyserTx.Transaction.Transaction.Message.RecentBlockhash).Bytes())

	return tx//,innerInstructions
}


// ConvertBlockHash converts a Geyser block to a github.com/gagliardetto/solana-go Solana block.
func ConvertBlockHash(geyserBlock *geyser_pb.SubscribeUpdateBlock) *rpc.GetBlockResult {
	block := new(rpc.GetBlockResult)

	blockTime := solana.UnixTimeSeconds(geyserBlock.BlockTime.Timestamp)
	block.BlockTime = &blockTime
	block.BlockHeight = &geyserBlock.BlockHeight.BlockHeight
	block.Blockhash = solana.Hash{[]byte(geyserBlock.Blockhash)[32]}
	block.ParentSlot = geyserBlock.ParentSlot

	for _, reward := range geyserBlock.Rewards.Rewards {
		commission, err := strconv.ParseUint(reward.Commission, 10, 8)
		if err != nil {
			return nil
		}

		var rewardType rpc.RewardType
		switch reward.RewardType {
		case 1:
			rewardType = rpc.RewardTypeFee
		case 2:
			rewardType = rpc.RewardTypeRent
		case 3:
			rewardType = rpc.RewardTypeStaking
		case 4:
			rewardType = rpc.RewardTypeVoting
		}

		comm := uint8(commission)
		block.Rewards = append(block.Rewards, rpc.BlockReward{
			Pubkey:      solana.MustPublicKeyFromBase58(reward.Pubkey),
			Lamports:    reward.Lamports,
			PostBalance: reward.PostBalance,
			Commission:  &comm,
			RewardType:  rewardType,
		})
	}

	return block
}
