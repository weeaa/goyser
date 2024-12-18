// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.25.3
// source: yellowstone-log.proto

package yellowstone_geyser_pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	YellowstoneLog_CreateStaticConsumerGroup_FullMethodName = "/yellowstone.log.YellowstoneLog/CreateStaticConsumerGroup"
	YellowstoneLog_Consume_FullMethodName                   = "/yellowstone.log.YellowstoneLog/Consume"
)

// YellowstoneLogClient is the client API for YellowstoneLog service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type YellowstoneLogClient interface {
	CreateStaticConsumerGroup(ctx context.Context, in *CreateStaticConsumerGroupRequest, opts ...grpc.CallOption) (*CreateStaticConsumerGroupResponse, error)
	Consume(ctx context.Context, in *ConsumeRequest, opts ...grpc.CallOption) (YellowstoneLog_ConsumeClient, error)
}

type yellowstoneLogClient struct {
	cc grpc.ClientConnInterface
}

func NewYellowstoneLogClient(cc grpc.ClientConnInterface) YellowstoneLogClient {
	return &yellowstoneLogClient{cc}
}

func (c *yellowstoneLogClient) CreateStaticConsumerGroup(ctx context.Context, in *CreateStaticConsumerGroupRequest, opts ...grpc.CallOption) (*CreateStaticConsumerGroupResponse, error) {
	out := new(CreateStaticConsumerGroupResponse)
	err := c.cc.Invoke(ctx, YellowstoneLog_CreateStaticConsumerGroup_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yellowstoneLogClient) Consume(ctx context.Context, in *ConsumeRequest, opts ...grpc.CallOption) (YellowstoneLog_ConsumeClient, error) {
	stream, err := c.cc.NewStream(ctx, &YellowstoneLog_ServiceDesc.Streams[0], YellowstoneLog_Consume_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &yellowstoneLogConsumeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type YellowstoneLog_ConsumeClient interface {
	Recv() (*SubscribeUpdate, error)
	grpc.ClientStream
}

type yellowstoneLogConsumeClient struct {
	grpc.ClientStream
}

func (x *yellowstoneLogConsumeClient) Recv() (*SubscribeUpdate, error) {
	m := new(SubscribeUpdate)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// YellowstoneLogServer is the server API for YellowstoneLog service.
// All implementations must embed UnimplementedYellowstoneLogServer
// for forward compatibility
type YellowstoneLogServer interface {
	CreateStaticConsumerGroup(context.Context, *CreateStaticConsumerGroupRequest) (*CreateStaticConsumerGroupResponse, error)
	Consume(*ConsumeRequest, YellowstoneLog_ConsumeServer) error
	mustEmbedUnimplementedYellowstoneLogServer()
}

// UnimplementedYellowstoneLogServer must be embedded to have forward compatible implementations.
type UnimplementedYellowstoneLogServer struct {
}

func (UnimplementedYellowstoneLogServer) CreateStaticConsumerGroup(context.Context, *CreateStaticConsumerGroupRequest) (*CreateStaticConsumerGroupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateStaticConsumerGroup not implemented")
}
func (UnimplementedYellowstoneLogServer) Consume(*ConsumeRequest, YellowstoneLog_ConsumeServer) error {
	return status.Errorf(codes.Unimplemented, "method Consume not implemented")
}
func (UnimplementedYellowstoneLogServer) mustEmbedUnimplementedYellowstoneLogServer() {}

// UnsafeYellowstoneLogServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to YellowstoneLogServer will
// result in compilation errors.
type UnsafeYellowstoneLogServer interface {
	mustEmbedUnimplementedYellowstoneLogServer()
}

func RegisterYellowstoneLogServer(s grpc.ServiceRegistrar, srv YellowstoneLogServer) {
	s.RegisterService(&YellowstoneLog_ServiceDesc, srv)
}

func _YellowstoneLog_CreateStaticConsumerGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateStaticConsumerGroupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YellowstoneLogServer).CreateStaticConsumerGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: YellowstoneLog_CreateStaticConsumerGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YellowstoneLogServer).CreateStaticConsumerGroup(ctx, req.(*CreateStaticConsumerGroupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _YellowstoneLog_Consume_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ConsumeRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(YellowstoneLogServer).Consume(m, &yellowstoneLogConsumeServer{stream})
}

type YellowstoneLog_ConsumeServer interface {
	Send(*SubscribeUpdate) error
	grpc.ServerStream
}

type yellowstoneLogConsumeServer struct {
	grpc.ServerStream
}

func (x *yellowstoneLogConsumeServer) Send(m *SubscribeUpdate) error {
	return x.ServerStream.SendMsg(m)
}

// YellowstoneLog_ServiceDesc is the grpc.ServiceDesc for YellowstoneLog service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var YellowstoneLog_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "yellowstone.log.YellowstoneLog",
	HandlerType: (*YellowstoneLogServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateStaticConsumerGroup",
			Handler:    _YellowstoneLog_CreateStaticConsumerGroup_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Consume",
			Handler:       _YellowstoneLog_Consume_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "yellowstone-log.proto",
}
