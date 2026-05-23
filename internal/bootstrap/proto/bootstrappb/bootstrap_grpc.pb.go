// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bootstrappb

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
	Workspace_Initialize_FullMethodName              = "/bootstrap.Workspace/Initialize"
	Workspace_GetInitializationStatus_FullMethodName = "/bootstrap.Workspace/GetInitializationStatus"
)

// WorkspaceClient is the client API for Workspace service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WorkspaceClient interface {
	Initialize(ctx context.Context, in *Workspace_Initialize_Req, opts ...grpc.CallOption) (*Workspace_Initialize_Resp, error)
	GetInitializationStatus(ctx context.Context, in *Workspace_GetInitializationStatus_Req, opts ...grpc.CallOption) (*Workspace_GetInitializationStatus_Resp, error)
}

type workspaceClient struct {
	cc grpc.ClientConnInterface
}

func NewWorkspaceClient(cc grpc.ClientConnInterface) WorkspaceClient {
	return &workspaceClient{cc}
}

func (c *workspaceClient) Initialize(ctx context.Context, in *Workspace_Initialize_Req, opts ...grpc.CallOption) (*Workspace_Initialize_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Workspace_Initialize_Resp)
	err := c.cc.Invoke(ctx, Workspace_Initialize_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workspaceClient) GetInitializationStatus(ctx context.Context, in *Workspace_GetInitializationStatus_Req, opts ...grpc.CallOption) (*Workspace_GetInitializationStatus_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Workspace_GetInitializationStatus_Resp)
	err := c.cc.Invoke(ctx, Workspace_GetInitializationStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WorkspaceServer is the server API for Workspace service.
// All implementations must embed UnimplementedWorkspaceServer
// for forward compatibility.
type WorkspaceServer interface {
	Initialize(context.Context, *Workspace_Initialize_Req) (*Workspace_Initialize_Resp, error)
	GetInitializationStatus(context.Context, *Workspace_GetInitializationStatus_Req) (*Workspace_GetInitializationStatus_Resp, error)
	mustEmbedUnimplementedWorkspaceServer()
}

// UnimplementedWorkspaceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedWorkspaceServer struct{}

func (UnimplementedWorkspaceServer) Initialize(context.Context, *Workspace_Initialize_Req) (*Workspace_Initialize_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Initialize not implemented")
}
func (UnimplementedWorkspaceServer) GetInitializationStatus(context.Context, *Workspace_GetInitializationStatus_Req) (*Workspace_GetInitializationStatus_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method GetInitializationStatus not implemented")
}
func (UnimplementedWorkspaceServer) mustEmbedUnimplementedWorkspaceServer() {}
func (UnimplementedWorkspaceServer) testEmbeddedByValue()                   {}

// UnsafeWorkspaceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WorkspaceServer will
// result in compilation errors.
type UnsafeWorkspaceServer interface {
	mustEmbedUnimplementedWorkspaceServer()
}

func RegisterWorkspaceServer(s grpc.ServiceRegistrar, srv WorkspaceServer) {
	// If the following call panics, it indicates UnimplementedWorkspaceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Workspace_ServiceDesc, srv)
}

func _Workspace_Initialize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Workspace_Initialize_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkspaceServer).Initialize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Workspace_Initialize_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkspaceServer).Initialize(ctx, req.(*Workspace_Initialize_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Workspace_GetInitializationStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Workspace_GetInitializationStatus_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkspaceServer).GetInitializationStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Workspace_GetInitializationStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkspaceServer).GetInitializationStatus(ctx, req.(*Workspace_GetInitializationStatus_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// Workspace_ServiceDesc is the grpc.ServiceDesc for Workspace service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Workspace_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "bootstrap.Workspace",
	HandlerType: (*WorkspaceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Initialize",
			Handler:    _Workspace_Initialize_Handler,
		},
		{
			MethodName: "GetInitializationStatus",
			Handler:    _Workspace_GetInitializationStatus_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "bootstrap.proto",
}
