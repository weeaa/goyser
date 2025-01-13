// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v4.25.3
// source: geyser.proto

package jito_pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Geyser_GetHeartbeatInterval_FullMethodName           = "/solana.geyser.Geyser/GetHeartbeatInterval"
	Geyser_SubscribeAccountUpdates_FullMethodName        = "/solana.geyser.Geyser/SubscribeAccountUpdates"
	Geyser_SubscribeProgramUpdates_FullMethodName        = "/solana.geyser.Geyser/SubscribeProgramUpdates"
	Geyser_SubscribePartialAccountUpdates_FullMethodName = "/solana.geyser.Geyser/SubscribePartialAccountUpdates"
	Geyser_SubscribeSlotUpdates_FullMethodName           = "/solana.geyser.Geyser/SubscribeSlotUpdates"
	Geyser_SubscribeTransactionUpdates_FullMethodName    = "/solana.geyser.Geyser/SubscribeTransactionUpdates"
	Geyser_SubscribeBlockUpdates_FullMethodName          = "/solana.geyser.Geyser/SubscribeBlockUpdates"
	Geyser_SubscribeSlotEntryUpdates_FullMethodName      = "/solana.geyser.Geyser/SubscribeSlotEntryUpdates"
)

// GeyserClient is the client API for Geyser service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// The following __must__ be assumed:
//   - Clients may receive data for slots out of order.
//   - Clients may receive account updates for a given slot out of order.
type GeyserClient interface {
	// Invoke to get the expected heartbeat interval.
	GetHeartbeatInterval(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*GetHeartbeatIntervalResponse, error)
	// Subscribes to account updates in the accounts database; additionally pings clients with empty heartbeats.
	// Upon initially connecting the client can expect a `highest_write_slot` set in the http headers.
	// Subscribe to account updates
	SubscribeAccountUpdates(ctx context.Context, in *SubscribeAccountUpdatesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedAccountUpdate], error)
	// Subscribes to updates given a list of program IDs. When an account update comes in that's owned by a provided
	// program id, one will receive an update
	SubscribeProgramUpdates(ctx context.Context, in *SubscribeProgramsUpdatesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedAccountUpdate], error)
	// Functions similarly to `SubscribeAccountUpdates`, but consumes less bandwidth.
	// Returns the highest slot seen thus far in the http headers named `highest-write-slot`.
	SubscribePartialAccountUpdates(ctx context.Context, in *SubscribePartialAccountUpdatesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[MaybePartialAccountUpdate], error)
	// Subscribes to slot updates.
	// Returns the highest slot seen thus far in the http headers named `highest-write-slot`.
	SubscribeSlotUpdates(ctx context.Context, in *SubscribeSlotUpdateRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedSlotUpdate], error)
	// Subscribes to transaction updates.
	SubscribeTransactionUpdates(ctx context.Context, in *SubscribeTransactionUpdatesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedTransactionUpdate], error)
	// Subscribes to block updates.
	SubscribeBlockUpdates(ctx context.Context, in *SubscribeBlockUpdatesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedBlockUpdate], error)
	// Subscribes to entry updates.
	// Returns the highest slot seen thus far and the entry index corresponding to the tick
	SubscribeSlotEntryUpdates(ctx context.Context, in *SubscribeSlotEntryUpdateRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedSlotEntryUpdate], error)
}

type geyserClient struct {
	cc grpc.ClientConnInterface
}

func NewGeyserClient(cc grpc.ClientConnInterface) GeyserClient {
	return &geyserClient{cc}
}

func (c *geyserClient) GetHeartbeatInterval(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*GetHeartbeatIntervalResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetHeartbeatIntervalResponse)
	err := c.cc.Invoke(ctx, Geyser_GetHeartbeatInterval_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *geyserClient) SubscribeAccountUpdates(ctx context.Context, in *SubscribeAccountUpdatesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedAccountUpdate], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Geyser_ServiceDesc.Streams[0], Geyser_SubscribeAccountUpdates_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[SubscribeAccountUpdatesRequest, TimestampedAccountUpdate]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeAccountUpdatesClient = grpc.ServerStreamingClient[TimestampedAccountUpdate]

func (c *geyserClient) SubscribeProgramUpdates(ctx context.Context, in *SubscribeProgramsUpdatesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedAccountUpdate], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Geyser_ServiceDesc.Streams[1], Geyser_SubscribeProgramUpdates_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[SubscribeProgramsUpdatesRequest, TimestampedAccountUpdate]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeProgramUpdatesClient = grpc.ServerStreamingClient[TimestampedAccountUpdate]

func (c *geyserClient) SubscribePartialAccountUpdates(ctx context.Context, in *SubscribePartialAccountUpdatesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[MaybePartialAccountUpdate], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Geyser_ServiceDesc.Streams[2], Geyser_SubscribePartialAccountUpdates_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[SubscribePartialAccountUpdatesRequest, MaybePartialAccountUpdate]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribePartialAccountUpdatesClient = grpc.ServerStreamingClient[MaybePartialAccountUpdate]

func (c *geyserClient) SubscribeSlotUpdates(ctx context.Context, in *SubscribeSlotUpdateRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedSlotUpdate], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Geyser_ServiceDesc.Streams[3], Geyser_SubscribeSlotUpdates_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[SubscribeSlotUpdateRequest, TimestampedSlotUpdate]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeSlotUpdatesClient = grpc.ServerStreamingClient[TimestampedSlotUpdate]

func (c *geyserClient) SubscribeTransactionUpdates(ctx context.Context, in *SubscribeTransactionUpdatesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedTransactionUpdate], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Geyser_ServiceDesc.Streams[4], Geyser_SubscribeTransactionUpdates_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[SubscribeTransactionUpdatesRequest, TimestampedTransactionUpdate]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeTransactionUpdatesClient = grpc.ServerStreamingClient[TimestampedTransactionUpdate]

func (c *geyserClient) SubscribeBlockUpdates(ctx context.Context, in *SubscribeBlockUpdatesRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedBlockUpdate], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Geyser_ServiceDesc.Streams[5], Geyser_SubscribeBlockUpdates_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[SubscribeBlockUpdatesRequest, TimestampedBlockUpdate]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeBlockUpdatesClient = grpc.ServerStreamingClient[TimestampedBlockUpdate]

func (c *geyserClient) SubscribeSlotEntryUpdates(ctx context.Context, in *SubscribeSlotEntryUpdateRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[TimestampedSlotEntryUpdate], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Geyser_ServiceDesc.Streams[6], Geyser_SubscribeSlotEntryUpdates_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[SubscribeSlotEntryUpdateRequest, TimestampedSlotEntryUpdate]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeSlotEntryUpdatesClient = grpc.ServerStreamingClient[TimestampedSlotEntryUpdate]

// GeyserServer is the server API for Geyser service.
// All implementations must embed UnimplementedGeyserServer
// for forward compatibility.
//
// The following __must__ be assumed:
//   - Clients may receive data for slots out of order.
//   - Clients may receive account updates for a given slot out of order.
type GeyserServer interface {
	// Invoke to get the expected heartbeat interval.
	GetHeartbeatInterval(context.Context, *EmptyRequest) (*GetHeartbeatIntervalResponse, error)
	// Subscribes to account updates in the accounts database; additionally pings clients with empty heartbeats.
	// Upon initially connecting the client can expect a `highest_write_slot` set in the http headers.
	// Subscribe to account updates
	SubscribeAccountUpdates(*SubscribeAccountUpdatesRequest, grpc.ServerStreamingServer[TimestampedAccountUpdate]) error
	// Subscribes to updates given a list of program IDs. When an account update comes in that's owned by a provided
	// program id, one will receive an update
	SubscribeProgramUpdates(*SubscribeProgramsUpdatesRequest, grpc.ServerStreamingServer[TimestampedAccountUpdate]) error
	// Functions similarly to `SubscribeAccountUpdates`, but consumes less bandwidth.
	// Returns the highest slot seen thus far in the http headers named `highest-write-slot`.
	SubscribePartialAccountUpdates(*SubscribePartialAccountUpdatesRequest, grpc.ServerStreamingServer[MaybePartialAccountUpdate]) error
	// Subscribes to slot updates.
	// Returns the highest slot seen thus far in the http headers named `highest-write-slot`.
	SubscribeSlotUpdates(*SubscribeSlotUpdateRequest, grpc.ServerStreamingServer[TimestampedSlotUpdate]) error
	// Subscribes to transaction updates.
	SubscribeTransactionUpdates(*SubscribeTransactionUpdatesRequest, grpc.ServerStreamingServer[TimestampedTransactionUpdate]) error
	// Subscribes to block updates.
	SubscribeBlockUpdates(*SubscribeBlockUpdatesRequest, grpc.ServerStreamingServer[TimestampedBlockUpdate]) error
	// Subscribes to entry updates.
	// Returns the highest slot seen thus far and the entry index corresponding to the tick
	SubscribeSlotEntryUpdates(*SubscribeSlotEntryUpdateRequest, grpc.ServerStreamingServer[TimestampedSlotEntryUpdate]) error
	mustEmbedUnimplementedGeyserServer()
}

// UnimplementedGeyserServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedGeyserServer struct{}

func (UnimplementedGeyserServer) GetHeartbeatInterval(context.Context, *EmptyRequest) (*GetHeartbeatIntervalResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetHeartbeatInterval not implemented")
}
func (UnimplementedGeyserServer) SubscribeAccountUpdates(*SubscribeAccountUpdatesRequest, grpc.ServerStreamingServer[TimestampedAccountUpdate]) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeAccountUpdates not implemented")
}
func (UnimplementedGeyserServer) SubscribeProgramUpdates(*SubscribeProgramsUpdatesRequest, grpc.ServerStreamingServer[TimestampedAccountUpdate]) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeProgramUpdates not implemented")
}
func (UnimplementedGeyserServer) SubscribePartialAccountUpdates(*SubscribePartialAccountUpdatesRequest, grpc.ServerStreamingServer[MaybePartialAccountUpdate]) error {
	return status.Errorf(codes.Unimplemented, "method SubscribePartialAccountUpdates not implemented")
}
func (UnimplementedGeyserServer) SubscribeSlotUpdates(*SubscribeSlotUpdateRequest, grpc.ServerStreamingServer[TimestampedSlotUpdate]) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeSlotUpdates not implemented")
}
func (UnimplementedGeyserServer) SubscribeTransactionUpdates(*SubscribeTransactionUpdatesRequest, grpc.ServerStreamingServer[TimestampedTransactionUpdate]) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeTransactionUpdates not implemented")
}
func (UnimplementedGeyserServer) SubscribeBlockUpdates(*SubscribeBlockUpdatesRequest, grpc.ServerStreamingServer[TimestampedBlockUpdate]) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeBlockUpdates not implemented")
}
func (UnimplementedGeyserServer) SubscribeSlotEntryUpdates(*SubscribeSlotEntryUpdateRequest, grpc.ServerStreamingServer[TimestampedSlotEntryUpdate]) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeSlotEntryUpdates not implemented")
}
func (UnimplementedGeyserServer) mustEmbedUnimplementedGeyserServer() {}
func (UnimplementedGeyserServer) testEmbeddedByValue()                {}

// UnsafeGeyserServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GeyserServer will
// result in compilation errors.
type UnsafeGeyserServer interface {
	mustEmbedUnimplementedGeyserServer()
}

func RegisterGeyserServer(s grpc.ServiceRegistrar, srv GeyserServer) {
	// If the following call pancis, it indicates UnimplementedGeyserServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Geyser_ServiceDesc, srv)
}

func _Geyser_GetHeartbeatInterval_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmptyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GeyserServer).GetHeartbeatInterval(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Geyser_GetHeartbeatInterval_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GeyserServer).GetHeartbeatInterval(ctx, req.(*EmptyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Geyser_SubscribeAccountUpdates_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeAccountUpdatesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GeyserServer).SubscribeAccountUpdates(m, &grpc.GenericServerStream[SubscribeAccountUpdatesRequest, TimestampedAccountUpdate]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeAccountUpdatesServer = grpc.ServerStreamingServer[TimestampedAccountUpdate]

func _Geyser_SubscribeProgramUpdates_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeProgramsUpdatesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GeyserServer).SubscribeProgramUpdates(m, &grpc.GenericServerStream[SubscribeProgramsUpdatesRequest, TimestampedAccountUpdate]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeProgramUpdatesServer = grpc.ServerStreamingServer[TimestampedAccountUpdate]

func _Geyser_SubscribePartialAccountUpdates_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribePartialAccountUpdatesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GeyserServer).SubscribePartialAccountUpdates(m, &grpc.GenericServerStream[SubscribePartialAccountUpdatesRequest, MaybePartialAccountUpdate]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribePartialAccountUpdatesServer = grpc.ServerStreamingServer[MaybePartialAccountUpdate]

func _Geyser_SubscribeSlotUpdates_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeSlotUpdateRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GeyserServer).SubscribeSlotUpdates(m, &grpc.GenericServerStream[SubscribeSlotUpdateRequest, TimestampedSlotUpdate]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeSlotUpdatesServer = grpc.ServerStreamingServer[TimestampedSlotUpdate]

func _Geyser_SubscribeTransactionUpdates_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeTransactionUpdatesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GeyserServer).SubscribeTransactionUpdates(m, &grpc.GenericServerStream[SubscribeTransactionUpdatesRequest, TimestampedTransactionUpdate]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeTransactionUpdatesServer = grpc.ServerStreamingServer[TimestampedTransactionUpdate]

func _Geyser_SubscribeBlockUpdates_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeBlockUpdatesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GeyserServer).SubscribeBlockUpdates(m, &grpc.GenericServerStream[SubscribeBlockUpdatesRequest, TimestampedBlockUpdate]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeBlockUpdatesServer = grpc.ServerStreamingServer[TimestampedBlockUpdate]

func _Geyser_SubscribeSlotEntryUpdates_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeSlotEntryUpdateRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GeyserServer).SubscribeSlotEntryUpdates(m, &grpc.GenericServerStream[SubscribeSlotEntryUpdateRequest, TimestampedSlotEntryUpdate]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Geyser_SubscribeSlotEntryUpdatesServer = grpc.ServerStreamingServer[TimestampedSlotEntryUpdate]

// Geyser_ServiceDesc is the grpc.ServiceDesc for Geyser service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Geyser_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "solana.geyser.Geyser",
	HandlerType: (*GeyserServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetHeartbeatInterval",
			Handler:    _Geyser_GetHeartbeatInterval_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SubscribeAccountUpdates",
			Handler:       _Geyser_SubscribeAccountUpdates_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SubscribeProgramUpdates",
			Handler:       _Geyser_SubscribeProgramUpdates_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SubscribePartialAccountUpdates",
			Handler:       _Geyser_SubscribePartialAccountUpdates_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SubscribeSlotUpdates",
			Handler:       _Geyser_SubscribeSlotUpdates_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SubscribeTransactionUpdates",
			Handler:       _Geyser_SubscribeTransactionUpdates_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SubscribeBlockUpdates",
			Handler:       _Geyser_SubscribeBlockUpdates_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SubscribeSlotEntryUpdates",
			Handler:       _Geyser_SubscribeSlotEntryUpdates_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "geyser.proto",
}
