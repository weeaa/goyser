package jito_geyser

import (
	"context"
	"fmt"
	jito_geyser_pb "github.com/weeaa/goyser/jito_geyser/pb"
	"github.com/weeaa/goyser/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"sync"
)

type Client struct {
	GrpcConn *grpc.ClientConn
	Ctx      context.Context
	Geyser   jito_geyser_pb.GeyserClient
	ErrCh    chan error
	mu       sync.Mutex
	Stream   *StreamClient
	s        *streamManager
}

type streamManager struct {
	geyser  *jito_geyser_pb.GeyserClient
	clients map[string]*StreamClient
	mu      sync.RWMutex
}

type StreamClient struct {
	Ctx    context.Context
	geyser jito_geyser_pb.GeyserClient

	Accounts        map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedAccountUpdate]
	PartialAccounts map[string]grpc.ServerStreamingClient[jito_geyser_pb.MaybePartialAccountUpdate]
	Slots           map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedSlotUpdate]
	SlotsEntry      map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedSlotEntryUpdate]
	Programs        map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedAccountUpdate]
	Blocks          map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedBlockUpdate]
	Transactions    map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedTransactionUpdate]

	ErrCh chan error
	mu    sync.RWMutex
}

// New creates a new Client instance.
func New(ctx context.Context, grpcDialURL string, md metadata.MD) (*Client, error) {
	ch := make(chan error)
	conn, err := pkg.CreateAndObserveGRPCConn(ctx, ch, grpcDialURL, md)
	if err != nil {
		return nil, err
	}

	geyserClient := jito_geyser_pb.NewGeyserClient(conn)
	if md != nil {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	return &Client{
		GrpcConn: conn,
		Ctx:      ctx,
		Geyser:   geyserClient,
		ErrCh:    ch,
		s:        &streamManager{},
	}, nil
}

func (c *Client) Close() error {
	c.Ctx.Done()
	return c.GrpcConn.Close()
}

// AddStreamClient creates a new Geyser subscribe stream client. You can retrieve it with GetStreamClient.
func (c *Client) AddStreamClient(ctx context.Context, streamName string, opts ...grpc.CallOption) error {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	if _, exists := c.s.clients[streamName]; exists {
		return fmt.Errorf("client with name %s already exists", streamName)
	}

	streamClient := StreamClient{
		Ctx:    ctx,
		geyser: c.Geyser,
		ErrCh:  make(chan error),
		mu:     sync.RWMutex{},

		Accounts:        make(map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedAccountUpdate]),
		PartialAccounts: make(map[string]grpc.ServerStreamingClient[jito_geyser_pb.MaybePartialAccountUpdate]),
		Slots:           make(map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedSlotUpdate]),
		SlotsEntry:      make(map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedSlotEntryUpdate]),
		Programs:        make(map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedAccountUpdate]),
		Blocks:          make(map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedBlockUpdate]),
		Transactions:    make(map[string]grpc.ServerStreamingClient[jito_geyser_pb.TimestampedTransactionUpdate]),
	}

	c.s.clients[streamName] = &streamClient

	return nil
}

// GetStreamClient returns a StreamClient for the given streamName from the client's map.
func (c *Client) GetStreamClient(streamName string) *StreamClient {
	defer c.s.mu.RUnlock()
	c.s.mu.RLock()
	return c.s.clients[streamName]
}

func (c *Client) GetFilters() []string {
	filters := make([]string, 0)
	for k := range c.s.clients {
		filters = append(filters, k)
	}
	return filters
}

// GetHeartbeatInterval returns
func (s *StreamClient) GetHeartbeatInterval(opts ...grpc.CallOption) (*jito_geyser_pb.GetHeartbeatIntervalResponse, error) {
	return s.geyser.GetHeartbeatInterval(s.Ctx, &jito_geyser_pb.EmptyRequest{}, opts...)
}

// SubscribeAccounts subscribes to account updates.
func (s *StreamClient) SubscribeAccounts(filterName string, accounts ...string) error {
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Accounts[filterName], err = s.geyser.SubscribeAccountUpdates(s.Ctx, &jito_geyser_pb.SubscribeAccountUpdatesRequest{
		Accounts: pkg.StrSliceToByteSlices(accounts),
	})
	return err
}

// UnsubscribeAccounts unsubscribes all accounts.
func (s *StreamClient) UnsubscribeAccounts(filterName string) {
	s.Accounts[filterName].Context().Done()
	delete(s.Accounts, filterName)
}

// SubscribeSlots subscribes to slot updates.
func (s *StreamClient) SubscribeSlots(filterName string) error {
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Slots[filterName], err = s.geyser.SubscribeSlotUpdates(s.Ctx, &jito_geyser_pb.SubscribeSlotUpdateRequest{})
	return err
}

// UnsubscribeSlots unsubscribes from slot updates.
func (s *StreamClient) UnsubscribeSlots(filterName string) {
	s.Slots[filterName].Context().Done()
	delete(s.Slots, filterName)
}

func (s *StreamClient) SubscribeSlotsEntry(filterName string) error {
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SlotsEntry[filterName], err = s.geyser.SubscribeSlotEntryUpdates(s.Ctx, &jito_geyser_pb.SubscribeSlotEntryUpdateRequest{})
	return err
}

// SubscribeTransaction subscribes to transaction updates.
func (s *StreamClient) SubscribeTransaction(filterName string) error {
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Transactions[filterName], err = s.geyser.SubscribeTransactionUpdates(s.Ctx, &jito_geyser_pb.SubscribeTransactionUpdatesRequest{})
	return err
}

// UnsubscribeTransaction unsubscribes from transaction updates.
func (s *StreamClient) UnsubscribeTransaction(filterName string) {
	s.Transactions[filterName].Context().Done()
	delete(s.Transactions, filterName)
}

// SubscribePartialAccount subscribes .
func (s *StreamClient) SubscribePartialAccount(filterName string, skipVoteAccounts bool) error {
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PartialAccounts[filterName], err = s.geyser.SubscribePartialAccountUpdates(s.Ctx, &jito_geyser_pb.SubscribePartialAccountUpdatesRequest{
		SkipVoteAccounts: skipVoteAccounts,
	})
	return err
}

func (s *StreamClient) UnsubscribePartialAccount(filterName string) {
	s.PartialAccounts[filterName].Context().Done()
	delete(s.PartialAccounts, filterName)
}

// SubscribeBlocks subscribes to block updates.
func (s *StreamClient) SubscribeBlocks(filterName string) error {
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Blocks[filterName], err = s.geyser.SubscribeBlockUpdates(s.Ctx, &jito_geyser_pb.SubscribeBlockUpdatesRequest{})
	return err
}

func (s *StreamClient) UnsubscribeBlocks(filterName string) {
	s.Blocks[filterName].Context().Done()
	delete(s.Blocks, filterName)
}

// SubscribePrograms subscribes to program updates.
func (s *StreamClient) SubscribePrograms(filterName string, programs ...string) error {
	var err error
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Programs[filterName], err = s.geyser.SubscribeProgramUpdates(s.Ctx, &jito_geyser_pb.SubscribeProgramsUpdatesRequest{
		Programs: pkg.StrSliceToByteSlices(programs),
	})
	return err
}

func (s *StreamClient) UnsubscribePrograms(filterName string) {
	s.Programs[filterName].Context().Done()
	delete(s.Programs, filterName)
}

/*
// ConvertTransaction converts a Geyser parsed transaction into an rpc.GetTransactionResult format.
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

	solTx = &solana.Transaction{
		Signatures: make([]solana.Signature, 0),
		Message: solana.Message{
			AccountKeys:         make(solana.PublicKeySlice, len(transaction.Message.AccountKeys)),
			Instructions:        make([]solana.CompiledInstruction, len(transaction.Message.Instructions)),
			AddressTableLookups: make(solana.MessageAddressTableLookupSlice, len(transaction.Message.AddressTableLookups)),
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

	return tx, nil
}

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
*/
