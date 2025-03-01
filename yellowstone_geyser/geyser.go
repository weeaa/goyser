package yellowstone_geyser

import (
	"context"
	"errors"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/weeaa/goyser/pkg"
	"github.com/weeaa/goyser/yellowstone_geyser/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"io"
	"reflect"
	"slices"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

type Client struct {
	Ctx            context.Context
	GrpcConn       *grpc.ClientConn
	KeepAliveCount int32
	Geyser         yellowstone_geyser_pb.GeyserClient
	ErrCh          chan error
	s              *streamManager
}

type streamManager struct {
	clients map[string]*StreamClient
	mu      sync.RWMutex
}

type StreamClient struct {
	Ctx        context.Context
	GrpcConn   *grpc.ClientConn
	streamName string
	geyserConn yellowstone_geyser_pb.GeyserClient
	geyser     yellowstone_geyser_pb.Geyser_SubscribeClient
	request    *yellowstone_geyser_pb.SubscribeRequest
	Ch         chan *yellowstone_geyser_pb.SubscribeUpdate
	ErrCh      chan error
	mu         sync.RWMutex

	countMu     sync.RWMutex
	count       int32
	latestCount time.Time
}

// New creates a new Client instance.
func New(ctx context.Context, grpcDialURL string, md metadata.MD) (*Client, error) {
	ch := make(chan error)

	if md != nil {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	conn, err := pkg.CreateAndObserveGRPCConn(ctx, ch, grpcDialURL)
	if err != nil {
		return nil, err
	}

	geyserClient := yellowstone_geyser_pb.NewGeyserClient(conn)

	return &Client{
		GrpcConn: conn,
		Ctx:      ctx,
		Geyser:   geyserClient,
		ErrCh:    ch,
		s: &streamManager{
			clients: make(map[string]*StreamClient),
			mu:      sync.RWMutex{},
		},
	}, nil
}

// Close closes the client and all the streams.
func (c *Client) Close() error {
	for _, sc := range c.s.clients {
		sc.Stop()
	}
	close(c.ErrCh)
	return c.GrpcConn.Close()
}

func (c *Client) Ping(count int32) (*yellowstone_geyser_pb.PongResponse, error) {
	return c.Geyser.Ping(c.Ctx, &yellowstone_geyser_pb.PingRequest{Count: count})
}

// AddStreamClient creates a new Geyser subscribe stream client. You can retrieve it with GetStreamClient.
func (c *Client) AddStreamClient(ctx context.Context, streamName string, commitmentLevel yellowstone_geyser_pb.CommitmentLevel, opts ...grpc.CallOption) error {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	if _, exists := c.s.clients[streamName]; exists {
		return fmt.Errorf("client with name %s already exists", streamName)
	}

	stream, err := c.Geyser.Subscribe(ctx, opts...)
	if err != nil {
		return err
	}

	streamClient := StreamClient{
		Ctx:        ctx,
		GrpcConn:   c.GrpcConn,
		streamName: streamName,
		geyser:     stream,
		geyserConn: c.Geyser,
		request: &yellowstone_geyser_pb.SubscribeRequest{
			Accounts:           make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterAccounts),
			Slots:              make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterSlots),
			Transactions:       make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterTransactions),
			TransactionsStatus: make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterTransactions),
			Blocks:             make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterBlocks),
			BlocksMeta:         make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterBlocksMeta),
			Entry:              make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterEntry),
			AccountsDataSlice:  make([]*yellowstone_geyser_pb.SubscribeRequestAccountsDataSlice, 0),
			Commitment:         &commitmentLevel,
		},
		Ch:    make(chan *yellowstone_geyser_pb.SubscribeUpdate),
		ErrCh: make(chan error),
		mu:    sync.RWMutex{},
	}

	c.s.clients[streamName] = &streamClient
	go streamClient.listen()
	go streamClient.keepAlive()

	return nil
}

func (s *StreamClient) Stop() {
	s.Ctx.Done()
	close(s.Ch)
	close(s.ErrCh)
}

// GetStreamClient returns a StreamClient for the given streamName from the client's map.
func (c *Client) GetStreamClient(streamName string) *StreamClient {
	defer c.s.mu.RUnlock()
	c.s.mu.RLock()
	return c.s.clients[streamName]
}

func (s *StreamClient) GetStreamName() string {
	return s.streamName
}

// SetRequest sets a custom request to be used across all methods.
func (s *StreamClient) SetRequest(req *yellowstone_geyser_pb.SubscribeRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.request = req
}

// SetCommitmentLevel modifies the commitment level of the stream's request.
func (s *StreamClient) SetCommitmentLevel(commitmentLevel yellowstone_geyser_pb.CommitmentLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.request.Commitment = &commitmentLevel
}

// NewRequest creates a new empty *geyser_pb.SubscribeRequest.
func (s *StreamClient) NewRequest() *yellowstone_geyser_pb.SubscribeRequest {
	return &yellowstone_geyser_pb.SubscribeRequest{
		Accounts:           make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterAccounts),
		Slots:              make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterSlots),
		Transactions:       make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterTransactions),
		TransactionsStatus: make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterTransactions),
		Blocks:             make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterBlocks),
		BlocksMeta:         make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterBlocksMeta),
		Entry:              make(map[string]*yellowstone_geyser_pb.SubscribeRequestFilterEntry),
		AccountsDataSlice:  make([]*yellowstone_geyser_pb.SubscribeRequestAccountsDataSlice, 0),
	}
}

// SendCustomRequest sends a custom *geyser_pb.SubscribeRequest using the Geyser client.
func (s *StreamClient) SendCustomRequest(request *yellowstone_geyser_pb.SubscribeRequest) error {
	return s.geyser.Send(request)
}

func (s *StreamClient) sendRequest() error {
	return s.geyser.Send(s.request)
}

// SubscribeAccounts subscribes to account updates.
// Note: This will overwrite existing subscriptions for the given ID.
// To add new accounts without overwriting, use AppendAccounts.
func (s *StreamClient) SubscribeAccounts(filterName string, req *yellowstone_geyser_pb.SubscribeRequestFilterAccounts) error {
	s.mu.Lock()
	s.request.Accounts[filterName] = req
	s.mu.Unlock()
	return s.geyser.Send(s.request)
}

// GetAccounts returns all account addresses for the given filter name.
func (s *StreamClient) GetAccounts(filterName string) []string {
	defer s.mu.RUnlock()
	s.mu.RLock()
	return s.request.Accounts[filterName].Account
}

// AppendAccounts appends accounts to an existing subscription and sends the request.
func (s *StreamClient) AppendAccounts(filterName string, accounts ...string) error {
	s.request.Accounts[filterName].Account = append(s.request.Accounts[filterName].Account, accounts...)
	return s.geyser.Send(s.request)
}

// UnsubscribeAccounts unsubscribes specific accounts.
func (s *StreamClient) UnsubscribeAccounts(filterName string, accounts ...string) error {
	defer s.mu.Unlock()
	s.mu.Lock()
	if filter, exists := s.request.Accounts[filterName]; exists {
		filter.Account = slices.DeleteFunc(filter.Account, func(a string) bool {
			return slices.Contains(accounts, a)
		})
	}
	return s.sendRequest()
}

func (s *StreamClient) UnsubscribeAllAccounts(filterName string) error {
	delete(s.request.Accounts, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeSlots subscribes to slot updates.
func (s *StreamClient) SubscribeSlots(filterName string, req *yellowstone_geyser_pb.SubscribeRequestFilterSlots) error {
	s.request.Slots[filterName] = req
	return s.geyser.Send(s.request)
}

// UnsubscribeSlots unsubscribes from slot updates.
func (s *StreamClient) UnsubscribeSlots(filterName string) error {
	delete(s.request.Slots, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeTransaction subscribes to transaction updates.
func (s *StreamClient) SubscribeTransaction(filterName string, req *yellowstone_geyser_pb.SubscribeRequestFilterTransactions) error {
	s.request.Transactions[filterName] = req
	return s.geyser.Send(s.request)
}

// UnsubscribeTransaction unsubscribes from transaction updates.
func (s *StreamClient) UnsubscribeTransaction(filterName string) error {
	delete(s.request.Transactions, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeTransactionStatus subscribes to transaction status updates.
func (s *StreamClient) SubscribeTransactionStatus(filterName string, req *yellowstone_geyser_pb.SubscribeRequestFilterTransactions) error {
	s.request.TransactionsStatus[filterName] = req
	return s.geyser.Send(s.request)
}

func (s *StreamClient) UnsubscribeTransactionStatus(filterName string) error {
	delete(s.request.TransactionsStatus, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeBlocks subscribes to block updates.
func (s *StreamClient) SubscribeBlocks(filterName string, req *yellowstone_geyser_pb.SubscribeRequestFilterBlocks) error {
	s.request.Blocks[filterName] = req
	return s.geyser.Send(s.request)
}

func (s *StreamClient) UnsubscribeBlocks(filterName string) error {
	delete(s.request.Blocks, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeBlocksMeta subscribes to block metadata updates.
func (s *StreamClient) SubscribeBlocksMeta(filterName string, req *yellowstone_geyser_pb.SubscribeRequestFilterBlocksMeta) error {
	s.request.BlocksMeta[filterName] = req
	return s.geyser.Send(s.request)
}

func (s *StreamClient) UnsubscribeBlocksMeta(filterName string) error {
	delete(s.request.BlocksMeta, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeEntry subscribes to entry updates.
func (s *StreamClient) SubscribeEntry(filterName string, req *yellowstone_geyser_pb.SubscribeRequestFilterEntry) error {
	s.request.Entry[filterName] = req
	return s.geyser.Send(s.request)
}

func (s *StreamClient) UnsubscribeEntry(filterName string) error {
	delete(s.request.Entry, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeAccountDataSlice subscribes to account data slice updates.
func (s *StreamClient) SubscribeAccountDataSlice(req []*yellowstone_geyser_pb.SubscribeRequestAccountsDataSlice) error {
	s.request.AccountsDataSlice = req
	return s.geyser.Send(s.request)
}

func (s *StreamClient) UnsubscribeAccountDataSlice() error {
	s.request.AccountsDataSlice = nil
	return s.geyser.Send(s.request)
}

// listen starts listening for responses and errors.
func (s *StreamClient) listen() {
	defer close(s.Ch)
	defer close(s.ErrCh)

	for {
		select {
		case <-s.Ctx.Done():
			if err := s.Ctx.Err(); err != nil {
				s.ErrCh <- fmt.Errorf("stream context cancelled: %w", err)
			}
			return
		default:
			recv, err := s.geyser.Recv()
			if err != nil {
				if err == io.EOF {
					s.ErrCh <- errors.New("stream cancelled: EOF")
					return
				}
				select {
				case s.ErrCh <- fmt.Errorf("error receiving from stream: %w", err):
				case <-s.Ctx.Done():
					if err = s.Ctx.Err(); err != nil {
						s.ErrCh <- fmt.Errorf("stream context cancelled: %w", err)
					}
					return
				}
				return
			}
			select {
			case s.Ch <- recv:
			case <-s.Ctx.Done():
				return
			}
		}
	}
}

// keepAlive sends every 10 second a ping to the gRPC conn in order to keep it alive.
func (s *StreamClient) keepAlive() {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-s.Ctx.Done():
			return
		case <-ticker.C:
			s.count++
			s.latestCount = time.Now()

			state := s.GrpcConn.GetState()
			if state == connectivity.Idle || state == connectivity.Ready {
				if _, err := s.geyserConn.Ping(s.Ctx,
					&yellowstone_geyser_pb.PingRequest{
						Count: s.count,
					},
				); err != nil {
					s.ErrCh <- err
				}
			} else {
				s.ErrCh <- fmt.Errorf("%s: error keeping alive conn: expected %s or %s, got %s", s.streamName, connectivity.Idle.String(), connectivity.Ready.String(), state.String())
			}
		}
	}
}

// GetKeepAliveCountAndTimestamp returns the
func (s *StreamClient) GetKeepAliveCountAndTimestamp() (int32, time.Time) {
	return s.count, s.latestCount
}

// ConvertTransaction converts a Geyser parsed transaction into *rpc.GetTransactionResult.
func ConvertTransaction(geyserTx *yellowstone_geyser_pb.SubscribeUpdateTransaction) (*rpc.GetTransactionResult, error) {
	meta := geyserTx.Transaction.Meta
	transaction := geyserTx.Transaction.Transaction

	tx := &rpc.GetTransactionResult{
		Transaction: &rpc.TransactionResultEnvelope{},
		Meta: &rpc.TransactionMeta{
			InnerInstructions: make([]rpc.InnerInstruction, 0),
			LogMessages:       make([]string, 0),
			PostBalances:      make([]uint64, 0),
			PostTokenBalances: make([]rpc.TokenBalance, 0),
			PreBalances:       make([]uint64, 0),
			PreTokenBalances:  make([]rpc.TokenBalance, 0),
			Rewards:           make([]rpc.BlockReward, 0),
			LoadedAddresses: rpc.LoadedAddresses{
				ReadOnly: make([]solana.PublicKey, 0),
				Writable: make([]solana.PublicKey, 0),
			},
		},
		Slot: geyserTx.Slot,
	}

	tx.Meta.PreBalances = meta.PreBalances
	tx.Meta.PostBalances = meta.PostBalances
	tx.Meta.Err = meta.Err
	tx.Meta.Fee = meta.Fee
	tx.Meta.ComputeUnitsConsumed = meta.ComputeUnitsConsumed
	tx.Meta.LogMessages = meta.LogMessages

	for _, preTokenBalance := range meta.PreTokenBalances {
		owner := solana.MustPublicKeyFromBase58(preTokenBalance.Owner)
		tx.Meta.PreTokenBalances = append(tx.Meta.PreTokenBalances, rpc.TokenBalance{
			AccountIndex: uint16(preTokenBalance.AccountIndex),
			Owner:        &owner,
			Mint:         solana.MustPublicKeyFromBase58(preTokenBalance.Mint),
			UiTokenAmount: &rpc.UiTokenAmount{
				Amount:         preTokenBalance.UiTokenAmount.Amount,
				Decimals:       uint8(preTokenBalance.UiTokenAmount.Decimals),
				UiAmount:       &preTokenBalance.UiTokenAmount.UiAmount,
				UiAmountString: preTokenBalance.UiTokenAmount.UiAmountString,
			},
		})
	}

	for _, postTokenBalance := range meta.PostTokenBalances {
		owner := solana.MustPublicKeyFromBase58(postTokenBalance.Owner)
		tx.Meta.PostTokenBalances = append(tx.Meta.PostTokenBalances, rpc.TokenBalance{
			AccountIndex: uint16(postTokenBalance.AccountIndex),
			Owner:        &owner,
			Mint:         solana.MustPublicKeyFromBase58(postTokenBalance.Mint),
			UiTokenAmount: &rpc.UiTokenAmount{
				Amount:         postTokenBalance.UiTokenAmount.Amount,
				Decimals:       uint8(postTokenBalance.UiTokenAmount.Decimals),
				UiAmount:       &postTokenBalance.UiTokenAmount.UiAmount,
				UiAmountString: postTokenBalance.UiTokenAmount.UiAmountString,
			},
		})
	}

	for i, innerInst := range meta.InnerInstructions {
		tx.Meta.InnerInstructions = append(tx.Meta.InnerInstructions, rpc.InnerInstruction{})
		tx.Meta.InnerInstructions[i].Index = uint16(innerInst.Index)
		for x, inst := range innerInst.Instructions {
			tx.Meta.InnerInstructions[i].Instructions = append(tx.Meta.InnerInstructions[i].Instructions, solana.CompiledInstruction{})
			accounts, err := bytesToUint16Slice(inst.Accounts)
			if err != nil {
				return nil, err
			}

			tx.Meta.InnerInstructions[i].Instructions[x].Accounts = accounts
			tx.Meta.InnerInstructions[i].Instructions[x].ProgramIDIndex = uint16(inst.ProgramIdIndex)
			tx.Meta.InnerInstructions[i].Instructions[x].Data = inst.Data
			// if err = tx.Meta.InnerInstructions[i].Instructions[x].Data.UnmarshalJSON(inst.Data); err != nil {
			// 	return nil, err
			// }
		}
	}

	for _, reward := range meta.Rewards {
		comm, _ := strconv.ParseUint(reward.Commission, 10, 64)
		commission := uint8(comm)
		tx.Meta.Rewards = append(tx.Meta.Rewards, rpc.BlockReward{
			Pubkey:      solana.MustPublicKeyFromBase58(reward.Pubkey),
			Lamports:    reward.Lamports,
			PostBalance: reward.PostBalance,
			RewardType:  rpc.RewardType(reward.RewardType.String()),
			Commission:  &commission,
		})
	}

	for _, readOnlyAddress := range meta.LoadedReadonlyAddresses {
		tx.Meta.LoadedAddresses.ReadOnly = append(tx.Meta.LoadedAddresses.ReadOnly, solana.PublicKeyFromBytes(readOnlyAddress))
	}

	for _, writableAddress := range meta.LoadedWritableAddresses {
		tx.Meta.LoadedAddresses.Writable = append(tx.Meta.LoadedAddresses.Writable, solana.PublicKeyFromBytes(writableAddress))
	}

	// solTx, err := tx.Transaction.GetTransaction()
	// if err != nil {
	// 	return nil, err
	// }

	solTx := &solana.Transaction{
		Signatures: make([]solana.Signature, 0),
		Message: solana.Message{
			AccountKeys:         make(solana.PublicKeySlice, 0),
			Instructions:        make([]solana.CompiledInstruction, 0),
			AddressTableLookups: make(solana.MessageAddressTableLookupSlice, 0),
		},
	}

	if transaction.Message.Versioned {
		solTx.Message.SetVersion(1)
	}

	solTx.Message.RecentBlockhash = solana.HashFromBytes(transaction.Message.RecentBlockhash)
	solTx.Message.Header = solana.MessageHeader{
		NumRequiredSignatures:       uint8(transaction.Message.Header.NumRequiredSignatures),
		NumReadonlySignedAccounts:   uint8(transaction.Message.Header.NumReadonlySignedAccounts),
		NumReadonlyUnsignedAccounts: uint8(transaction.Message.Header.NumReadonlyUnsignedAccounts),
	}

	for _, sig := range transaction.Signatures {
		solTx.Signatures = append(solTx.Signatures, solana.SignatureFromBytes(sig))
	}

	for _, table := range transaction.Message.AddressTableLookups {
		solTx.Message.AddressTableLookups = append(solTx.Message.AddressTableLookups, solana.MessageAddressTableLookup{
			AccountKey:      solana.PublicKeyFromBytes(table.AccountKey),
			WritableIndexes: table.WritableIndexes,
			ReadonlyIndexes: table.ReadonlyIndexes,
		})
	}

	for _, key := range transaction.Message.AccountKeys {
		solTx.Message.AccountKeys = append(solTx.Message.AccountKeys, solana.PublicKeyFromBytes(key))
	}

	for _, inst := range transaction.Message.Instructions {
		accounts, err := bytesToUint16Slice(inst.Accounts)
		if err != nil {
			return nil, err
		}

		solTx.Message.Instructions = append(solTx.Message.Instructions, solana.CompiledInstruction{
			ProgramIDIndex: uint16(inst.ProgramIdIndex),
			Accounts:       accounts,
			Data:           inst.Data,
		})
	}

	func(obj interface{}, fieldName string, value interface{}) {
		v := reflect.ValueOf(obj).Elem()
		f := v.FieldByName(fieldName)
		ptr := unsafe.Pointer(f.UnsafeAddr())
		reflect.NewAt(f.Type(), ptr).Elem().Set(reflect.ValueOf(value))
	}(tx.Transaction, "asParsedTransaction", solTx)

	return tx, nil
}

/*
// ConvertTransactionf converts a Geyser parsed transaction into an rpc.GetTransactionResult format.
func ConvertTransactionf(geyserTx *yellowstone_geyser_pb.SubscribeUpdateTransaction) (*rpc.GetParsedTransactionResult, error) {
	meta := geyserTx.Transaction.Meta
	transaction := geyserTx.Transaction.Transaction

	tx := &rpc.GetParsedTransactionResult{
		Transaction: &rpc.ParsedTransaction{
			Signatures: make([]solana.Signature, 0),
			Message: rpc.ParsedMessage{
				AccountKeys:     make([]rpc.ParsedMessageAccount, 0),
				Instructions:    make([]*rpc.ParsedInstruction, 0),
				RecentBlockHash: string(transaction.Message.RecentBlockhash),
			},
		},
		Meta: &rpc.ParsedTransactionMeta{
			InnerInstructions: make([]rpc.ParsedInnerInstruction, 0),
			LogMessages:       meta.LogMessages,
			PostBalances:      make([]uint64, 0),
			PostTokenBalances: make([]rpc.TokenBalance, 0),
			PreBalances:       make([]uint64, 0),
			PreTokenBalances:  make([]rpc.TokenBalance, 0),
			//Rewards:           make([]rpc.BlockReward, 0),
		},
		Slot: geyserTx.Slot,
		//Version: transaction.Message.Versioned, todo
	}

	tx.Meta.PreBalances = meta.PreBalances
	tx.Meta.PostBalances = meta.PostBalances
	tx.Meta.Err = meta.Err
	tx.Meta.Fee = meta.Fee
	//tx.Meta.ComputeUnitsConsumed = meta.ComputeUnitsConsumed
	tx.Meta.LogMessages = meta.LogMessages

	for _, preTokenBalance := range meta.PreTokenBalances {
		owner := solana.MustPublicKeyFromBase58(preTokenBalance.Owner)
		tx.Meta.PreTokenBalances = append(tx.Meta.PreTokenBalances, rpc.TokenBalance{
			AccountIndex: uint16(preTokenBalance.AccountIndex),
			Owner:        &owner,
			Mint:         solana.MustPublicKeyFromBase58(preTokenBalance.Mint),
			UiTokenAmount: &rpc.UiTokenAmount{
				Amount:         preTokenBalance.UiTokenAmount.Amount,
				Decimals:       uint8(preTokenBalance.UiTokenAmount.Decimals),
				UiAmount:       &preTokenBalance.UiTokenAmount.UiAmount,
				UiAmountString: preTokenBalance.UiTokenAmount.UiAmountString,
			},
		})
	}

	for _, postTokenBalance := range meta.PostTokenBalances {
		owner := solana.MustPublicKeyFromBase58(postTokenBalance.Owner)
		tx.Meta.PostTokenBalances = append(tx.Meta.PostTokenBalances, rpc.TokenBalance{
			AccountIndex: uint16(postTokenBalance.AccountIndex),
			Owner:        &owner,
			Mint:         solana.MustPublicKeyFromBase58(postTokenBalance.Mint),
			UiTokenAmount: &rpc.UiTokenAmount{
				Amount:         postTokenBalance.UiTokenAmount.Amount,
				Decimals:       uint8(postTokenBalance.UiTokenAmount.Decimals),
				UiAmount:       &postTokenBalance.UiTokenAmount.UiAmount,
				UiAmountString: postTokenBalance.UiTokenAmount.UiAmountString,
			},
		})
	}

	for i, innerInst := range meta.InnerInstructions {
		tx.Meta.InnerInstructions = append(tx.Meta.InnerInstructions, rpc.ParsedInnerInstruction{})
		tx.Meta.InnerInstructions[i].Index = uint64(innerInst.Index)
		for x, inst := range innerInst.Instructions {
			tx.Meta.InnerInstructions[i].Instructions = append(tx.Meta.InnerInstructions[i].Instructions, &rpc.ParsedInstruction{})
			accounts, err := bytesToUint16Slice(inst.Accounts)
			if err != nil {
				return nil, err
			}

			//tx.Meta.InnerInstructions[i].Instructions[x].Parsed

			tx.Meta.InnerInstructions[i].Instructions[x].Accounts = accounts
			tx.Meta.InnerInstructions[i].Instructions[x].ProgramIDIndex = uint16(inst.ProgramIdIndex)
			tx.Meta.InnerInstructions[i].Instructions[x].Data = inst.Data
			tx.Meta.InnerInstructions[i].Instructions[x].Program = inst.
			// if err = tx.Meta.InnerInstructions[i].Instructions[x].Data.UnmarshalJSON(inst.Data); err != nil {
			// 	return nil, err
			// }
		}
	}

	for _, reward := range meta.Rewards {
		comm, _ := strconv.ParseUint(reward.Commission, 10, 64)
		commission := uint8(comm)
		tx.Meta.Rewards = append(tx.Meta.Rewards, rpc.BlockReward{
			Pubkey:      solana.MustPublicKeyFromBase58(reward.Pubkey),
			Lamports:    reward.Lamports,
			PostBalance: reward.PostBalance,
			RewardType:  rpc.RewardType(reward.RewardType.String()),
			Commission:  &commission,
		})
	}

	for _, readOnlyAddress := range meta.LoadedReadonlyAddresses {
		tx.Meta.LoadedAddresses.ReadOnly = append(tx.Meta.LoadedAddresses.ReadOnly, solana.PublicKeyFromBytes(readOnlyAddress))
	}

	for _, writableAddress := range meta.LoadedWritableAddresses {
		tx.Meta.LoadedAddresses.ReadOnly = append(tx.Meta.LoadedAddresses.ReadOnly, solana.PublicKeyFromBytes(writableAddress))
	}

	// solTx, err := tx.Transaction.GetTransaction()
	// if err != nil {
	// 	return nil, err
	// }

	solTx := &solana.Transaction{
		Signatures: make([]solana.Signature, 0),
		Message: solana.Message{
			AccountKeys:         make(solana.PublicKeySlice, 0),
			Instructions:        make([]solana.CompiledInstruction, 0),
			AddressTableLookups: make(solana.MessageAddressTableLookupSlice, 0),
		},
	}

	if transaction.Message.Versioned {
		solTx.Message.SetVersion(1)
	}

	solTx.Message.RecentBlockhash = solana.HashFromBytes(transaction.Message.RecentBlockhash)
	solTx.Message.Header = solana.MessageHeader{
		NumRequiredSignatures:       uint8(transaction.Message.Header.NumRequiredSignatures),
		NumReadonlySignedAccounts:   uint8(transaction.Message.Header.NumReadonlySignedAccounts),
		NumReadonlyUnsignedAccounts: uint8(transaction.Message.Header.NumReadonlyUnsignedAccounts),
	}

	for _, sig := range transaction.Signatures {
		solTx.Signatures = append(solTx.Signatures, solana.SignatureFromBytes(sig))
	}

	for _, table := range transaction.Message.AddressTableLookups {
		solTx.Message.AddressTableLookups = append(solTx.Message.AddressTableLookups, solana.MessageAddressTableLookup{
			AccountKey:      solana.PublicKeyFromBytes(table.AccountKey),
			WritableIndexes: table.WritableIndexes,
			ReadonlyIndexes: table.ReadonlyIndexes,
		})
	}

	for _, key := range transaction.Message.AccountKeys {
		solTx.Message.AccountKeys = append(solTx.Message.AccountKeys, solana.PublicKeyFromBytes(key))
	}

	for _, inst := range transaction.Message.Instructions {
		accounts, err := bytesToUint16Slice(inst.Accounts)
		if err != nil {
			return nil, err
		}

		solTx.Message.Instructions = append(solTx.Message.Instructions, solana.CompiledInstruction{
			ProgramIDIndex: uint16(inst.ProgramIdIndex),
			Accounts:       accounts,
			Data:           inst.Data,
		})
	}

	func(obj interface{}, fieldName string, value interface{}) {
		v := reflect.ValueOf(obj).Elem()
		f := v.FieldByName(fieldName)
		ptr := unsafe.Pointer(f.UnsafeAddr())
		reflect.NewAt(f.Type(), ptr).Elem().Set(reflect.ValueOf(value))
	}(tx.Transaction, "asParsedTransaction", solTx)

	return tx, nil
}
*/

/*
func BatchConvertTransaction(geyserTxns ...*yellowstone_geyser_pb.SubscribeUpdateTransaction) []*rpc.GetTransactionResult {
	txns := make([]*rpc.GetTransactionResult, len(geyserTxns), 0)
	for _, tx := range geyserTxns {
		txn, err := ConvertTransaction(tx)
		if err != nil {
			continue
		}
		txns = append(txns, txn)
	}
	return txns
}
*/

// ConvertBlockHash converts a Geyser type block to a github.com/gagliardetto/solana-go Solana block.
func ConvertBlockHash(geyserBlock *yellowstone_geyser_pb.SubscribeUpdateBlock) *rpc.GetBlockResult {
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

func BatchConvertBlockHash(geyserBlocks ...*yellowstone_geyser_pb.SubscribeUpdateBlock) []*rpc.GetBlockResult {
	blocks := make([]*rpc.GetBlockResult, len(geyserBlocks), 0)
	for _, block := range geyserBlocks {
		blocks = append(blocks, ConvertBlockHash(block))
	}
	return blocks
}

// func bytesToUint16Slice(data []byte) ([]uint16, error) {
// 	if len(data)%2 != 0 {
// 		return nil, fmt.Errorf("length of byte slice must be even to convert to uint16 slice")
// 	}

// 	uint16s := make([]uint16, len(data)/2)

// 	for i := 0; i < len(data); i += 2 {
// 		uint16s[i/2] = binary.LittleEndian.Uint16(data[i : i+2])
// 	}

// 	return uint16s, nil
// }

func bytesToUint16Slice(data []byte) ([]uint16, error) {
	uint16s := make([]uint16, len(data))

	for i := 0; i < len(data); i += 1 {
		uint16s[i] = uint16(data[i])
	}

	return uint16s, nil
}
