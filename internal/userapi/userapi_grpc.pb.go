// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: internal/userapi/userapi.proto

package userapi

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
	UserAPI_SetManifest_FullMethodName  = "/edgelesssys.contrast.userapi.UserAPI/SetManifest"
	UserAPI_GetManifests_FullMethodName = "/edgelesssys.contrast.userapi.UserAPI/GetManifests"
	UserAPI_Recover_FullMethodName      = "/edgelesssys.contrast.userapi.UserAPI/Recover"
)

// UserAPIClient is the client API for UserAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type UserAPIClient interface {
	SetManifest(ctx context.Context, in *SetManifestRequest, opts ...grpc.CallOption) (*SetManifestResponse, error)
	GetManifests(ctx context.Context, in *GetManifestsRequest, opts ...grpc.CallOption) (*GetManifestsResponse, error)
	Recover(ctx context.Context, in *RecoverRequest, opts ...grpc.CallOption) (*RecoverResponse, error)
}

type userAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewUserAPIClient(cc grpc.ClientConnInterface) UserAPIClient {
	return &userAPIClient{cc}
}

func (c *userAPIClient) SetManifest(ctx context.Context, in *SetManifestRequest, opts ...grpc.CallOption) (*SetManifestResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SetManifestResponse)
	err := c.cc.Invoke(ctx, UserAPI_SetManifest_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userAPIClient) GetManifests(ctx context.Context, in *GetManifestsRequest, opts ...grpc.CallOption) (*GetManifestsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetManifestsResponse)
	err := c.cc.Invoke(ctx, UserAPI_GetManifests_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userAPIClient) Recover(ctx context.Context, in *RecoverRequest, opts ...grpc.CallOption) (*RecoverResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RecoverResponse)
	err := c.cc.Invoke(ctx, UserAPI_Recover_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UserAPIServer is the server API for UserAPI service.
// All implementations must embed UnimplementedUserAPIServer
// for forward compatibility.
type UserAPIServer interface {
	SetManifest(context.Context, *SetManifestRequest) (*SetManifestResponse, error)
	GetManifests(context.Context, *GetManifestsRequest) (*GetManifestsResponse, error)
	Recover(context.Context, *RecoverRequest) (*RecoverResponse, error)
	mustEmbedUnimplementedUserAPIServer()
}

// UnimplementedUserAPIServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedUserAPIServer struct{}

func (UnimplementedUserAPIServer) SetManifest(context.Context, *SetManifestRequest) (*SetManifestResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetManifest not implemented")
}
func (UnimplementedUserAPIServer) GetManifests(context.Context, *GetManifestsRequest) (*GetManifestsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetManifests not implemented")
}
func (UnimplementedUserAPIServer) Recover(context.Context, *RecoverRequest) (*RecoverResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Recover not implemented")
}
func (UnimplementedUserAPIServer) mustEmbedUnimplementedUserAPIServer() {}
func (UnimplementedUserAPIServer) testEmbeddedByValue()                 {}

// UnsafeUserAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to UserAPIServer will
// result in compilation errors.
type UnsafeUserAPIServer interface {
	mustEmbedUnimplementedUserAPIServer()
}

func RegisterUserAPIServer(s grpc.ServiceRegistrar, srv UserAPIServer) {
	// If the following call pancis, it indicates UnimplementedUserAPIServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&UserAPI_ServiceDesc, srv)
}

func _UserAPI_SetManifest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetManifestRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserAPIServer).SetManifest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserAPI_SetManifest_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserAPIServer).SetManifest(ctx, req.(*SetManifestRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserAPI_GetManifests_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetManifestsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserAPIServer).GetManifests(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserAPI_GetManifests_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserAPIServer).GetManifests(ctx, req.(*GetManifestsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserAPI_Recover_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RecoverRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserAPIServer).Recover(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserAPI_Recover_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserAPIServer).Recover(ctx, req.(*RecoverRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// UserAPI_ServiceDesc is the grpc.ServiceDesc for UserAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var UserAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "edgelesssys.contrast.userapi.UserAPI",
	HandlerType: (*UserAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SetManifest",
			Handler:    _UserAPI_SetManifest_Handler,
		},
		{
			MethodName: "GetManifests",
			Handler:    _UserAPI_GetManifests_Handler,
		},
		{
			MethodName: "Recover",
			Handler:    _UserAPI_Recover_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "internal/userapi/userapi.proto",
}
