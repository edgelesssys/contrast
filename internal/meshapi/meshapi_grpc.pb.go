// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: meshapi.proto

package meshapi

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
	MeshAPI_NewMeshCert_FullMethodName = "/meshapi.MeshAPI/NewMeshCert"
)

// MeshAPIClient is the client API for MeshAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MeshAPIClient interface {
	NewMeshCert(ctx context.Context, in *NewMeshCertRequest, opts ...grpc.CallOption) (*NewMeshCertResponse, error)
}

type meshAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewMeshAPIClient(cc grpc.ClientConnInterface) MeshAPIClient {
	return &meshAPIClient{cc}
}

func (c *meshAPIClient) NewMeshCert(ctx context.Context, in *NewMeshCertRequest, opts ...grpc.CallOption) (*NewMeshCertResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(NewMeshCertResponse)
	err := c.cc.Invoke(ctx, MeshAPI_NewMeshCert_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MeshAPIServer is the server API for MeshAPI service.
// All implementations must embed UnimplementedMeshAPIServer
// for forward compatibility.
type MeshAPIServer interface {
	NewMeshCert(context.Context, *NewMeshCertRequest) (*NewMeshCertResponse, error)
	mustEmbedUnimplementedMeshAPIServer()
}

// UnimplementedMeshAPIServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedMeshAPIServer struct{}

func (UnimplementedMeshAPIServer) NewMeshCert(context.Context, *NewMeshCertRequest) (*NewMeshCertResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NewMeshCert not implemented")
}
func (UnimplementedMeshAPIServer) mustEmbedUnimplementedMeshAPIServer() {}
func (UnimplementedMeshAPIServer) testEmbeddedByValue()                 {}

// UnsafeMeshAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MeshAPIServer will
// result in compilation errors.
type UnsafeMeshAPIServer interface {
	mustEmbedUnimplementedMeshAPIServer()
}

func RegisterMeshAPIServer(s grpc.ServiceRegistrar, srv MeshAPIServer) {
	// If the following call pancis, it indicates UnimplementedMeshAPIServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&MeshAPI_ServiceDesc, srv)
}

func _MeshAPI_NewMeshCert_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NewMeshCertRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MeshAPIServer).NewMeshCert(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MeshAPI_NewMeshCert_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MeshAPIServer).NewMeshCert(ctx, req.(*NewMeshCertRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// MeshAPI_ServiceDesc is the grpc.ServiceDesc for MeshAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MeshAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "meshapi.MeshAPI",
	HandlerType: (*MeshAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "NewMeshCert",
			Handler:    _MeshAPI_NewMeshCert_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "meshapi.proto",
}
