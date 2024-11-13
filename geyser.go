package goyser

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/weeaa/goyser/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io"
	"slices"
	"strconv"
	"sync"
)

type Client struct {
	grpcConn *grpc.ClientConn
	Ctx      context.Context
	Geyser   geyser_pb.GeyserClient
	ErrCh    chan error
	s        *streamManager
}

type streamManager struct {
	clients map[string]*StreamClient
	mu      sync.RWMutex
}

type StreamClient struct {
	Ctx     context.Context
	geyser  geyser_pb.Geyser_SubscribeClient
	request *geyser_pb.SubscribeRequest
	Ch      chan *geyser_pb.SubscribeUpdate
	ErrCh   chan error
	mu      sync.RWMutex
}

// New creates a new Client instance.
func New(ctx context.Context, grpcDialURL string, md metadata.MD) (*Client, error) {
	ch := make(chan error)
	conn, err := createAndObserveGRPCConn(ctx, ch, grpcDialURL, md)
	if err != nil {
		return nil, err
	}

	geyserClient := geyser_pb.NewGeyserClient(conn)

	if md != nil {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	return &Client{
		grpcConn: conn,
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
	return c.grpcConn.Close()
}

func (c *Client) Ping(count int32) (*geyser_pb.PongResponse, error) {
	return c.Geyser.Ping(c.Ctx, &geyser_pb.PingRequest{Count: count})
}

// AddStreamClient creates a new Geyser subscribe stream client. You can then retrieve it with GetStreamClient
func (c *Client) AddStreamClient(ctx context.Context, streamName string, opts ...grpc.CallOption) error {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	if _, exists := c.s.clients[streamName]; exists {
		return fmt.Errorf("client with name %s already exists", streamName)
	}

	stream, err := c.Geyser.Subscribe(ctx, opts...)
	if err != nil {
		return err
	}

	streamClient := &StreamClient{
		Ctx:    ctx,
		geyser: stream,
		request: &geyser_pb.SubscribeRequest{
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
		mu:    sync.RWMutex{},
	}

	c.s.clients[streamName] = streamClient
	go streamClient.listen()

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

// SetRequest sets a custom request to be used across all methods.
func (s *StreamClient) SetRequest(req *geyser_pb.SubscribeRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.request = req
}

// NewRequest creates a new empty *geyser_pb.SubscribeRequest.
func (s *StreamClient) NewRequest() *geyser_pb.SubscribeRequest {
	return &geyser_pb.SubscribeRequest{
		Accounts:           make(map[string]*geyser_pb.SubscribeRequestFilterAccounts),
		Slots:              make(map[string]*geyser_pb.SubscribeRequestFilterSlots),
		Transactions:       make(map[string]*geyser_pb.SubscribeRequestFilterTransactions),
		TransactionsStatus: make(map[string]*geyser_pb.SubscribeRequestFilterTransactions),
		Blocks:             make(map[string]*geyser_pb.SubscribeRequestFilterBlocks),
		BlocksMeta:         make(map[string]*geyser_pb.SubscribeRequestFilterBlocksMeta),
		Entry:              make(map[string]*geyser_pb.SubscribeRequestFilterEntry),
		AccountsDataSlice:  make([]*geyser_pb.SubscribeRequestAccountsDataSlice, 0),
	}
}

// SendCustomRequest sends a custom *geyser_pb.SubscribeRequest using the Geyser client.
func (s *StreamClient) SendCustomRequest(request *geyser_pb.SubscribeRequest) error {
	return s.geyser.Send(request)
}

func (s *StreamClient) sendRequest() error {
	return s.geyser.Send(s.request)
}

// SubscribeAccounts subscribes to account updates.
// Note: This will overwrite existing subscriptions for the given ID.
// To add new accounts without overwriting, use AppendAccounts.
func (s *StreamClient) SubscribeAccounts(filterName string, req *geyser_pb.SubscribeRequestFilterAccounts) error {
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

// UnsubscribeAccountsByFilterName unsubscribes from account updates by filter name.
func (s *StreamClient) UnsubscribeAccountsByFilterName(filterName string) error {
	delete(s.request.Accounts, filterName)
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
func (s *StreamClient) SubscribeSlots(filterName string, req *geyser_pb.SubscribeRequestFilterSlots) error {
	s.request.Slots[filterName] = req
	return s.geyser.Send(s.request)
}

// UnsubscribeSlots unsubscribes from slot updates.
func (s *StreamClient) UnsubscribeSlots(filterName string) error {
	delete(s.request.Slots, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeTransaction subscribes to transaction updates.
func (s *StreamClient) SubscribeTransaction(filterName string, req *geyser_pb.SubscribeRequestFilterTransactions) error {
	s.request.Transactions[filterName] = req
	return s.geyser.Send(s.request)
}

// UnsubscribeTransaction unsubscribes from transaction updates.
func (s *StreamClient) UnsubscribeTransaction(filterName string) error {
	delete(s.request.Transactions, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeTransactionStatus subscribes to transaction status updates.
func (s *StreamClient) SubscribeTransactionStatus(filterName string, req *geyser_pb.SubscribeRequestFilterTransactions) error {
	s.request.TransactionsStatus[filterName] = req
	return s.geyser.Send(s.request)
}

func (s *StreamClient) UnsubscribeTransactionStatus(filterName string) error {
	delete(s.request.TransactionsStatus, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeBlocks subscribes to block updates.
func (s *StreamClient) SubscribeBlocks(filterName string, req *geyser_pb.SubscribeRequestFilterBlocks) error {
	s.request.Blocks[filterName] = req
	return s.geyser.Send(s.request)
}

func (s *StreamClient) UnsubscribeBlocks(filterName string) error {
	delete(s.request.Blocks, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeBlocksMeta subscribes to block metadata updates.
func (s *StreamClient) SubscribeBlocksMeta(filterName string, req *geyser_pb.SubscribeRequestFilterBlocksMeta) error {
	s.request.BlocksMeta[filterName] = req
	return s.geyser.Send(s.request)
}

func (s *StreamClient) UnsubscribeBlocksMeta(filterName string) error {
	delete(s.request.BlocksMeta, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeEntry subscribes to entry updates.
func (s *StreamClient) SubscribeEntry(filterName string, req *geyser_pb.SubscribeRequestFilterEntry) error {
	s.request.Entry[filterName] = req
	return s.geyser.Send(s.request)
}

func (s *StreamClient) UnsubscribeEntry(filterName string) error {
	delete(s.request.Entry, filterName)
	return s.geyser.Send(s.request)
}

// SubscribeAccountDataSlice subscribes to account data slice updates.
func (s *StreamClient) SubscribeAccountDataSlice(req []*geyser_pb.SubscribeRequestAccountsDataSlice) error {
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

// ConvertTransaction converts a Geyser parsed transaction into an rpc.GetTransactionResult format.
func ConvertTransaction(geyserTx *geyser_pb.SubscribeUpdateTransaction) (*rpc.GetTransactionResult, error) {
	var tx rpc.GetTransactionResult

	meta := geyserTx.Transaction.Meta
	transaction := geyserTx.Transaction.Transaction

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
		tx.Meta.InnerInstructions[i].Index = uint16(innerInst.Index)
		for x, inst := range innerInst.Instructions {
			accounts, err := bytesToUint16Slice(inst.Accounts)
			if err != nil {
				return nil, err
			}

			tx.Meta.InnerInstructions[i].Instructions[x].Accounts = accounts
			tx.Meta.InnerInstructions[i].Instructions[x].ProgramIDIndex = uint16(inst.ProgramIdIndex)
			if err = tx.Meta.InnerInstructions[i].Instructions[x].Data.UnmarshalJSON(inst.Data); err != nil {
				return nil, err
			}
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

	solTx, err := tx.Transaction.GetTransaction()
	if err != nil {
		return nil, err
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

	return &tx, nil
}

/*
// Deprecated: Use ConvertParsedTransaction instead.
func ConvertTransaction(geyserTx *geyser_pb.SubscribeUpdateTransaction) *solana.Transaction {
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

	// instructions
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

	return tx
}
*/

func BatchConvertTransaction(geyserTxns ...*geyser_pb.SubscribeUpdateTransaction) []*rpc.GetTransactionResult {
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

// ConvertBlockHash converts a Geyser type block to a github.com/gagliardetto/solana-go Solana block.
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

func BatchConvertBlockHash(geyserBlocks ...*geyser_pb.SubscribeUpdateBlock) []*rpc.GetBlockResult {
	blocks := make([]*rpc.GetBlockResult, len(geyserBlocks), 0)
	for _, block := range geyserBlocks {
		blocks = append(blocks, ConvertBlockHash(block))
	}
	return blocks
}

func bytesToUint16Slice(data []byte) ([]uint16, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("length of byte slice must be even to convert to uint16 slice")
	}

	uint16s := make([]uint16, len(data)/2)

	for i := 0; i < len(data); i += 2 {
		uint16s[i/2] = binary.LittleEndian.Uint16(data[i : i+2])
	}

	return uint16s, nil
}
