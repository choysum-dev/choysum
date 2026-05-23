// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package authpb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Language_Browse_FullMethodName         = "/auth.Language/Browse"
	Language_BrowseMany_FullMethodName     = "/auth.Language/BrowseMany"
	Language_Count_FullMethodName          = "/auth.Language/Count"
	Language_Create_FullMethodName         = "/auth.Language/Create"
	Language_CreateMany_FullMethodName     = "/auth.Language/CreateMany"
	Language_DefaultGet_FullMethodName     = "/auth.Language/DefaultGet"
	Language_Delete_FullMethodName         = "/auth.Language/Delete"
	Language_DeleteById_FullMethodName     = "/auth.Language/DeleteById"
	Language_Onchange_FullMethodName       = "/auth.Language/Onchange"
	Language_ReadGroup_FullMethodName      = "/auth.Language/ReadGroup"
	Language_ReadGroupCount_FullMethodName = "/auth.Language/ReadGroupCount"
	Language_Search_FullMethodName         = "/auth.Language/Search"
	Language_Update_FullMethodName         = "/auth.Language/Update"
	Language_UpdateById_FullMethodName     = "/auth.Language/UpdateById"
)

// LanguageClient is the client API for Language service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: Language
type LanguageClient interface {
	Browse(ctx context.Context, in *Language_Browse_Req, opts ...grpc.CallOption) (*Language_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *Language_BrowseMany_Req, opts ...grpc.CallOption) (*Language_BrowseMany_Resp, error)
	Count(ctx context.Context, in *Language_Count_Req, opts ...grpc.CallOption) (*Language_Count_Resp, error)
	Create(ctx context.Context, in *Language_Create_Req, opts ...grpc.CallOption) (*Language_Create_Resp, error)
	CreateMany(ctx context.Context, in *Language_CreateMany_Req, opts ...grpc.CallOption) (*Language_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *Language_DefaultGet_Req, opts ...grpc.CallOption) (*Language_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *Language_Delete_Req, opts ...grpc.CallOption) (*Language_Delete_Resp, error)
	DeleteById(ctx context.Context, in *Language_DeleteById_Req, opts ...grpc.CallOption) (*Language_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *Language_Onchange_Req, opts ...grpc.CallOption) (*Language_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *Language_ReadGroup_Req, opts ...grpc.CallOption) (*Language_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *Language_ReadGroupCount_Req, opts ...grpc.CallOption) (*Language_ReadGroupCount_Resp, error)
	Search(ctx context.Context, in *Language_Search_Req, opts ...grpc.CallOption) (*Language_Search_Resp, error)
	Update(ctx context.Context, in *Language_Update_Req, opts ...grpc.CallOption) (*Language_Update_Resp, error)
	UpdateById(ctx context.Context, in *Language_UpdateById_Req, opts ...grpc.CallOption) (*Language_UpdateById_Resp, error)
}

type languageClient struct {
	cc grpc.ClientConnInterface
}

func NewLanguageClient(cc grpc.ClientConnInterface) LanguageClient {
	return &languageClient{cc}
}

func (c *languageClient) Browse(ctx context.Context, in *Language_Browse_Req, opts ...grpc.CallOption) (*Language_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_Browse_Resp)
	err := c.cc.Invoke(ctx, Language_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) BrowseMany(ctx context.Context, in *Language_BrowseMany_Req, opts ...grpc.CallOption) (*Language_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, Language_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) Count(ctx context.Context, in *Language_Count_Req, opts ...grpc.CallOption) (*Language_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_Count_Resp)
	err := c.cc.Invoke(ctx, Language_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) Create(ctx context.Context, in *Language_Create_Req, opts ...grpc.CallOption) (*Language_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_Create_Resp)
	err := c.cc.Invoke(ctx, Language_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) CreateMany(ctx context.Context, in *Language_CreateMany_Req, opts ...grpc.CallOption) (*Language_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_CreateMany_Resp)
	err := c.cc.Invoke(ctx, Language_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) DefaultGet(ctx context.Context, in *Language_DefaultGet_Req, opts ...grpc.CallOption) (*Language_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, Language_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) Delete(ctx context.Context, in *Language_Delete_Req, opts ...grpc.CallOption) (*Language_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_Delete_Resp)
	err := c.cc.Invoke(ctx, Language_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) DeleteById(ctx context.Context, in *Language_DeleteById_Req, opts ...grpc.CallOption) (*Language_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_DeleteById_Resp)
	err := c.cc.Invoke(ctx, Language_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) Onchange(ctx context.Context, in *Language_Onchange_Req, opts ...grpc.CallOption) (*Language_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_Onchange_Resp)
	err := c.cc.Invoke(ctx, Language_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) ReadGroup(ctx context.Context, in *Language_ReadGroup_Req, opts ...grpc.CallOption) (*Language_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, Language_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) ReadGroupCount(ctx context.Context, in *Language_ReadGroupCount_Req, opts ...grpc.CallOption) (*Language_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, Language_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) Search(ctx context.Context, in *Language_Search_Req, opts ...grpc.CallOption) (*Language_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_Search_Resp)
	err := c.cc.Invoke(ctx, Language_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) Update(ctx context.Context, in *Language_Update_Req, opts ...grpc.CallOption) (*Language_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_Update_Resp)
	err := c.cc.Invoke(ctx, Language_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *languageClient) UpdateById(ctx context.Context, in *Language_UpdateById_Req, opts ...grpc.CallOption) (*Language_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Language_UpdateById_Resp)
	err := c.cc.Invoke(ctx, Language_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LanguageServer is the server API for Language service.
// All implementations must embed UnimplementedLanguageServer
// for forward compatibility.
//
// Model: Language
type LanguageServer interface {
	Browse(context.Context, *Language_Browse_Req) (*Language_Browse_Resp, error)
	BrowseMany(context.Context, *Language_BrowseMany_Req) (*Language_BrowseMany_Resp, error)
	Count(context.Context, *Language_Count_Req) (*Language_Count_Resp, error)
	Create(context.Context, *Language_Create_Req) (*Language_Create_Resp, error)
	CreateMany(context.Context, *Language_CreateMany_Req) (*Language_CreateMany_Resp, error)
	DefaultGet(context.Context, *Language_DefaultGet_Req) (*Language_DefaultGet_Resp, error)
	Delete(context.Context, *Language_Delete_Req) (*Language_Delete_Resp, error)
	DeleteById(context.Context, *Language_DeleteById_Req) (*Language_DeleteById_Resp, error)
	Onchange(context.Context, *Language_Onchange_Req) (*Language_Onchange_Resp, error)
	ReadGroup(context.Context, *Language_ReadGroup_Req) (*Language_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *Language_ReadGroupCount_Req) (*Language_ReadGroupCount_Resp, error)
	Search(context.Context, *Language_Search_Req) (*Language_Search_Resp, error)
	Update(context.Context, *Language_Update_Req) (*Language_Update_Resp, error)
	UpdateById(context.Context, *Language_UpdateById_Req) (*Language_UpdateById_Resp, error)
	mustEmbedUnimplementedLanguageServer()
}

// UnimplementedLanguageServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedLanguageServer struct{}

func (UnimplementedLanguageServer) Browse(context.Context, *Language_Browse_Req) (*Language_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedLanguageServer) BrowseMany(context.Context, *Language_BrowseMany_Req) (*Language_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedLanguageServer) Count(context.Context, *Language_Count_Req) (*Language_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedLanguageServer) Create(context.Context, *Language_Create_Req) (*Language_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedLanguageServer) CreateMany(context.Context, *Language_CreateMany_Req) (*Language_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedLanguageServer) DefaultGet(context.Context, *Language_DefaultGet_Req) (*Language_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedLanguageServer) Delete(context.Context, *Language_Delete_Req) (*Language_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedLanguageServer) DeleteById(context.Context, *Language_DeleteById_Req) (*Language_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedLanguageServer) Onchange(context.Context, *Language_Onchange_Req) (*Language_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedLanguageServer) ReadGroup(context.Context, *Language_ReadGroup_Req) (*Language_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedLanguageServer) ReadGroupCount(context.Context, *Language_ReadGroupCount_Req) (*Language_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedLanguageServer) Search(context.Context, *Language_Search_Req) (*Language_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedLanguageServer) Update(context.Context, *Language_Update_Req) (*Language_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedLanguageServer) UpdateById(context.Context, *Language_UpdateById_Req) (*Language_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedLanguageServer) mustEmbedUnimplementedLanguageServer() {}
func (UnimplementedLanguageServer) testEmbeddedByValue()                  {}

// UnsafeLanguageServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LanguageServer will
// result in compilation errors.
type UnsafeLanguageServer interface {
	mustEmbedUnimplementedLanguageServer()
}

func RegisterLanguageServer(s grpc.ServiceRegistrar, srv LanguageServer) {
	// If the following call panics, it indicates UnimplementedLanguageServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Language_ServiceDesc, srv)
}

func _Language_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).Browse(ctx, req.(*Language_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).BrowseMany(ctx, req.(*Language_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).Count(ctx, req.(*Language_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).Create(ctx, req.(*Language_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).CreateMany(ctx, req.(*Language_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).DefaultGet(ctx, req.(*Language_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).Delete(ctx, req.(*Language_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).DeleteById(ctx, req.(*Language_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).Onchange(ctx, req.(*Language_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).ReadGroup(ctx, req.(*Language_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).ReadGroupCount(ctx, req.(*Language_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).Search(ctx, req.(*Language_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).Update(ctx, req.(*Language_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Language_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Language_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LanguageServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Language_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LanguageServer).UpdateById(ctx, req.(*Language_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// Language_ServiceDesc is the grpc.ServiceDesc for Language service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Language_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.Language",
	HandlerType: (*LanguageServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _Language_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _Language_BrowseMany_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _Language_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _Language_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _Language_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _Language_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _Language_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _Language_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _Language_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _Language_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _Language_ReadGroupCount_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _Language_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _Language_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _Language_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	Location_Browse_FullMethodName         = "/auth.Location/Browse"
	Location_BrowseMany_FullMethodName     = "/auth.Location/BrowseMany"
	Location_Count_FullMethodName          = "/auth.Location/Count"
	Location_Create_FullMethodName         = "/auth.Location/Create"
	Location_CreateMany_FullMethodName     = "/auth.Location/CreateMany"
	Location_DefaultGet_FullMethodName     = "/auth.Location/DefaultGet"
	Location_Delete_FullMethodName         = "/auth.Location/Delete"
	Location_DeleteById_FullMethodName     = "/auth.Location/DeleteById"
	Location_Onchange_FullMethodName       = "/auth.Location/Onchange"
	Location_ReadGroup_FullMethodName      = "/auth.Location/ReadGroup"
	Location_ReadGroupCount_FullMethodName = "/auth.Location/ReadGroupCount"
	Location_Register_FullMethodName       = "/auth.Location/Register"
	Location_Search_FullMethodName         = "/auth.Location/Search"
	Location_Update_FullMethodName         = "/auth.Location/Update"
	Location_UpdateById_FullMethodName     = "/auth.Location/UpdateById"
)

// LocationClient is the client API for Location service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: Location
type LocationClient interface {
	Browse(ctx context.Context, in *Location_Browse_Req, opts ...grpc.CallOption) (*Location_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *Location_BrowseMany_Req, opts ...grpc.CallOption) (*Location_BrowseMany_Resp, error)
	Count(ctx context.Context, in *Location_Count_Req, opts ...grpc.CallOption) (*Location_Count_Resp, error)
	Create(ctx context.Context, in *Location_Create_Req, opts ...grpc.CallOption) (*Location_Create_Resp, error)
	CreateMany(ctx context.Context, in *Location_CreateMany_Req, opts ...grpc.CallOption) (*Location_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *Location_DefaultGet_Req, opts ...grpc.CallOption) (*Location_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *Location_Delete_Req, opts ...grpc.CallOption) (*Location_Delete_Resp, error)
	DeleteById(ctx context.Context, in *Location_DeleteById_Req, opts ...grpc.CallOption) (*Location_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *Location_Onchange_Req, opts ...grpc.CallOption) (*Location_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *Location_ReadGroup_Req, opts ...grpc.CallOption) (*Location_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *Location_ReadGroupCount_Req, opts ...grpc.CallOption) (*Location_ReadGroupCount_Resp, error)
	Register(ctx context.Context, in *Location_Register_Req, opts ...grpc.CallOption) (*Location_Register_Resp, error)
	Search(ctx context.Context, in *Location_Search_Req, opts ...grpc.CallOption) (*Location_Search_Resp, error)
	Update(ctx context.Context, in *Location_Update_Req, opts ...grpc.CallOption) (*Location_Update_Resp, error)
	UpdateById(ctx context.Context, in *Location_UpdateById_Req, opts ...grpc.CallOption) (*Location_UpdateById_Resp, error)
}

type locationClient struct {
	cc grpc.ClientConnInterface
}

func NewLocationClient(cc grpc.ClientConnInterface) LocationClient {
	return &locationClient{cc}
}

func (c *locationClient) Browse(ctx context.Context, in *Location_Browse_Req, opts ...grpc.CallOption) (*Location_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_Browse_Resp)
	err := c.cc.Invoke(ctx, Location_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) BrowseMany(ctx context.Context, in *Location_BrowseMany_Req, opts ...grpc.CallOption) (*Location_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, Location_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) Count(ctx context.Context, in *Location_Count_Req, opts ...grpc.CallOption) (*Location_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_Count_Resp)
	err := c.cc.Invoke(ctx, Location_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) Create(ctx context.Context, in *Location_Create_Req, opts ...grpc.CallOption) (*Location_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_Create_Resp)
	err := c.cc.Invoke(ctx, Location_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) CreateMany(ctx context.Context, in *Location_CreateMany_Req, opts ...grpc.CallOption) (*Location_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_CreateMany_Resp)
	err := c.cc.Invoke(ctx, Location_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) DefaultGet(ctx context.Context, in *Location_DefaultGet_Req, opts ...grpc.CallOption) (*Location_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, Location_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) Delete(ctx context.Context, in *Location_Delete_Req, opts ...grpc.CallOption) (*Location_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_Delete_Resp)
	err := c.cc.Invoke(ctx, Location_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) DeleteById(ctx context.Context, in *Location_DeleteById_Req, opts ...grpc.CallOption) (*Location_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_DeleteById_Resp)
	err := c.cc.Invoke(ctx, Location_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) Onchange(ctx context.Context, in *Location_Onchange_Req, opts ...grpc.CallOption) (*Location_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_Onchange_Resp)
	err := c.cc.Invoke(ctx, Location_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) ReadGroup(ctx context.Context, in *Location_ReadGroup_Req, opts ...grpc.CallOption) (*Location_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, Location_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) ReadGroupCount(ctx context.Context, in *Location_ReadGroupCount_Req, opts ...grpc.CallOption) (*Location_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, Location_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) Register(ctx context.Context, in *Location_Register_Req, opts ...grpc.CallOption) (*Location_Register_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_Register_Resp)
	err := c.cc.Invoke(ctx, Location_Register_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) Search(ctx context.Context, in *Location_Search_Req, opts ...grpc.CallOption) (*Location_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_Search_Resp)
	err := c.cc.Invoke(ctx, Location_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) Update(ctx context.Context, in *Location_Update_Req, opts ...grpc.CallOption) (*Location_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_Update_Resp)
	err := c.cc.Invoke(ctx, Location_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *locationClient) UpdateById(ctx context.Context, in *Location_UpdateById_Req, opts ...grpc.CallOption) (*Location_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Location_UpdateById_Resp)
	err := c.cc.Invoke(ctx, Location_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LocationServer is the server API for Location service.
// All implementations must embed UnimplementedLocationServer
// for forward compatibility.
//
// Model: Location
type LocationServer interface {
	Browse(context.Context, *Location_Browse_Req) (*Location_Browse_Resp, error)
	BrowseMany(context.Context, *Location_BrowseMany_Req) (*Location_BrowseMany_Resp, error)
	Count(context.Context, *Location_Count_Req) (*Location_Count_Resp, error)
	Create(context.Context, *Location_Create_Req) (*Location_Create_Resp, error)
	CreateMany(context.Context, *Location_CreateMany_Req) (*Location_CreateMany_Resp, error)
	DefaultGet(context.Context, *Location_DefaultGet_Req) (*Location_DefaultGet_Resp, error)
	Delete(context.Context, *Location_Delete_Req) (*Location_Delete_Resp, error)
	DeleteById(context.Context, *Location_DeleteById_Req) (*Location_DeleteById_Resp, error)
	Onchange(context.Context, *Location_Onchange_Req) (*Location_Onchange_Resp, error)
	ReadGroup(context.Context, *Location_ReadGroup_Req) (*Location_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *Location_ReadGroupCount_Req) (*Location_ReadGroupCount_Resp, error)
	Register(context.Context, *Location_Register_Req) (*Location_Register_Resp, error)
	Search(context.Context, *Location_Search_Req) (*Location_Search_Resp, error)
	Update(context.Context, *Location_Update_Req) (*Location_Update_Resp, error)
	UpdateById(context.Context, *Location_UpdateById_Req) (*Location_UpdateById_Resp, error)
	mustEmbedUnimplementedLocationServer()
}

// UnimplementedLocationServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedLocationServer struct{}

func (UnimplementedLocationServer) Browse(context.Context, *Location_Browse_Req) (*Location_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedLocationServer) BrowseMany(context.Context, *Location_BrowseMany_Req) (*Location_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedLocationServer) Count(context.Context, *Location_Count_Req) (*Location_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedLocationServer) Create(context.Context, *Location_Create_Req) (*Location_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedLocationServer) CreateMany(context.Context, *Location_CreateMany_Req) (*Location_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedLocationServer) DefaultGet(context.Context, *Location_DefaultGet_Req) (*Location_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedLocationServer) Delete(context.Context, *Location_Delete_Req) (*Location_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedLocationServer) DeleteById(context.Context, *Location_DeleteById_Req) (*Location_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedLocationServer) Onchange(context.Context, *Location_Onchange_Req) (*Location_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedLocationServer) ReadGroup(context.Context, *Location_ReadGroup_Req) (*Location_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedLocationServer) ReadGroupCount(context.Context, *Location_ReadGroupCount_Req) (*Location_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedLocationServer) Register(context.Context, *Location_Register_Req) (*Location_Register_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Register not implemented")
}
func (UnimplementedLocationServer) Search(context.Context, *Location_Search_Req) (*Location_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedLocationServer) Update(context.Context, *Location_Update_Req) (*Location_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedLocationServer) UpdateById(context.Context, *Location_UpdateById_Req) (*Location_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedLocationServer) mustEmbedUnimplementedLocationServer() {}
func (UnimplementedLocationServer) testEmbeddedByValue()                  {}

// UnsafeLocationServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LocationServer will
// result in compilation errors.
type UnsafeLocationServer interface {
	mustEmbedUnimplementedLocationServer()
}

func RegisterLocationServer(s grpc.ServiceRegistrar, srv LocationServer) {
	// If the following call panics, it indicates UnimplementedLocationServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Location_ServiceDesc, srv)
}

func _Location_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).Browse(ctx, req.(*Location_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).BrowseMany(ctx, req.(*Location_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).Count(ctx, req.(*Location_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).Create(ctx, req.(*Location_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).CreateMany(ctx, req.(*Location_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).DefaultGet(ctx, req.(*Location_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).Delete(ctx, req.(*Location_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).DeleteById(ctx, req.(*Location_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).Onchange(ctx, req.(*Location_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).ReadGroup(ctx, req.(*Location_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).ReadGroupCount(ctx, req.(*Location_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_Register_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_Register_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).Register(ctx, req.(*Location_Register_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).Search(ctx, req.(*Location_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).Update(ctx, req.(*Location_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Location_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Location_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocationServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Location_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocationServer).UpdateById(ctx, req.(*Location_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// Location_ServiceDesc is the grpc.ServiceDesc for Location service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Location_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.Location",
	HandlerType: (*LocationServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _Location_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _Location_BrowseMany_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _Location_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _Location_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _Location_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _Location_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _Location_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _Location_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _Location_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _Location_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _Location_ReadGroupCount_Handler,
		},
		{
			MethodName: "Register",
			Handler:    _Location_Register_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _Location_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _Location_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _Location_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	Order_Browse_FullMethodName         = "/auth.Order/Browse"
	Order_BrowseMany_FullMethodName     = "/auth.Order/BrowseMany"
	Order_Count_FullMethodName          = "/auth.Order/Count"
	Order_Create_FullMethodName         = "/auth.Order/Create"
	Order_CreateMany_FullMethodName     = "/auth.Order/CreateMany"
	Order_DefaultGet_FullMethodName     = "/auth.Order/DefaultGet"
	Order_Delete_FullMethodName         = "/auth.Order/Delete"
	Order_DeleteById_FullMethodName     = "/auth.Order/DeleteById"
	Order_Onchange_FullMethodName       = "/auth.Order/Onchange"
	Order_ReadGroup_FullMethodName      = "/auth.Order/ReadGroup"
	Order_ReadGroupCount_FullMethodName = "/auth.Order/ReadGroupCount"
	Order_Search_FullMethodName         = "/auth.Order/Search"
	Order_Update_FullMethodName         = "/auth.Order/Update"
	Order_UpdateById_FullMethodName     = "/auth.Order/UpdateById"
)

// OrderClient is the client API for Order service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: Order
type OrderClient interface {
	Browse(ctx context.Context, in *Order_Browse_Req, opts ...grpc.CallOption) (*Order_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *Order_BrowseMany_Req, opts ...grpc.CallOption) (*Order_BrowseMany_Resp, error)
	Count(ctx context.Context, in *Order_Count_Req, opts ...grpc.CallOption) (*Order_Count_Resp, error)
	Create(ctx context.Context, in *Order_Create_Req, opts ...grpc.CallOption) (*Order_Create_Resp, error)
	CreateMany(ctx context.Context, in *Order_CreateMany_Req, opts ...grpc.CallOption) (*Order_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *Order_DefaultGet_Req, opts ...grpc.CallOption) (*Order_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *Order_Delete_Req, opts ...grpc.CallOption) (*Order_Delete_Resp, error)
	DeleteById(ctx context.Context, in *Order_DeleteById_Req, opts ...grpc.CallOption) (*Order_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *Order_Onchange_Req, opts ...grpc.CallOption) (*Order_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *Order_ReadGroup_Req, opts ...grpc.CallOption) (*Order_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *Order_ReadGroupCount_Req, opts ...grpc.CallOption) (*Order_ReadGroupCount_Resp, error)
	Search(ctx context.Context, in *Order_Search_Req, opts ...grpc.CallOption) (*Order_Search_Resp, error)
	Update(ctx context.Context, in *Order_Update_Req, opts ...grpc.CallOption) (*Order_Update_Resp, error)
	UpdateById(ctx context.Context, in *Order_UpdateById_Req, opts ...grpc.CallOption) (*Order_UpdateById_Resp, error)
}

type orderClient struct {
	cc grpc.ClientConnInterface
}

func NewOrderClient(cc grpc.ClientConnInterface) OrderClient {
	return &orderClient{cc}
}

func (c *orderClient) Browse(ctx context.Context, in *Order_Browse_Req, opts ...grpc.CallOption) (*Order_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_Browse_Resp)
	err := c.cc.Invoke(ctx, Order_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) BrowseMany(ctx context.Context, in *Order_BrowseMany_Req, opts ...grpc.CallOption) (*Order_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, Order_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) Count(ctx context.Context, in *Order_Count_Req, opts ...grpc.CallOption) (*Order_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_Count_Resp)
	err := c.cc.Invoke(ctx, Order_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) Create(ctx context.Context, in *Order_Create_Req, opts ...grpc.CallOption) (*Order_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_Create_Resp)
	err := c.cc.Invoke(ctx, Order_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) CreateMany(ctx context.Context, in *Order_CreateMany_Req, opts ...grpc.CallOption) (*Order_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_CreateMany_Resp)
	err := c.cc.Invoke(ctx, Order_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) DefaultGet(ctx context.Context, in *Order_DefaultGet_Req, opts ...grpc.CallOption) (*Order_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, Order_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) Delete(ctx context.Context, in *Order_Delete_Req, opts ...grpc.CallOption) (*Order_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_Delete_Resp)
	err := c.cc.Invoke(ctx, Order_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) DeleteById(ctx context.Context, in *Order_DeleteById_Req, opts ...grpc.CallOption) (*Order_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_DeleteById_Resp)
	err := c.cc.Invoke(ctx, Order_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) Onchange(ctx context.Context, in *Order_Onchange_Req, opts ...grpc.CallOption) (*Order_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_Onchange_Resp)
	err := c.cc.Invoke(ctx, Order_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) ReadGroup(ctx context.Context, in *Order_ReadGroup_Req, opts ...grpc.CallOption) (*Order_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, Order_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) ReadGroupCount(ctx context.Context, in *Order_ReadGroupCount_Req, opts ...grpc.CallOption) (*Order_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, Order_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) Search(ctx context.Context, in *Order_Search_Req, opts ...grpc.CallOption) (*Order_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_Search_Resp)
	err := c.cc.Invoke(ctx, Order_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) Update(ctx context.Context, in *Order_Update_Req, opts ...grpc.CallOption) (*Order_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_Update_Resp)
	err := c.cc.Invoke(ctx, Order_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderClient) UpdateById(ctx context.Context, in *Order_UpdateById_Req, opts ...grpc.CallOption) (*Order_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Order_UpdateById_Resp)
	err := c.cc.Invoke(ctx, Order_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// OrderServer is the server API for Order service.
// All implementations must embed UnimplementedOrderServer
// for forward compatibility.
//
// Model: Order
type OrderServer interface {
	Browse(context.Context, *Order_Browse_Req) (*Order_Browse_Resp, error)
	BrowseMany(context.Context, *Order_BrowseMany_Req) (*Order_BrowseMany_Resp, error)
	Count(context.Context, *Order_Count_Req) (*Order_Count_Resp, error)
	Create(context.Context, *Order_Create_Req) (*Order_Create_Resp, error)
	CreateMany(context.Context, *Order_CreateMany_Req) (*Order_CreateMany_Resp, error)
	DefaultGet(context.Context, *Order_DefaultGet_Req) (*Order_DefaultGet_Resp, error)
	Delete(context.Context, *Order_Delete_Req) (*Order_Delete_Resp, error)
	DeleteById(context.Context, *Order_DeleteById_Req) (*Order_DeleteById_Resp, error)
	Onchange(context.Context, *Order_Onchange_Req) (*Order_Onchange_Resp, error)
	ReadGroup(context.Context, *Order_ReadGroup_Req) (*Order_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *Order_ReadGroupCount_Req) (*Order_ReadGroupCount_Resp, error)
	Search(context.Context, *Order_Search_Req) (*Order_Search_Resp, error)
	Update(context.Context, *Order_Update_Req) (*Order_Update_Resp, error)
	UpdateById(context.Context, *Order_UpdateById_Req) (*Order_UpdateById_Resp, error)
	mustEmbedUnimplementedOrderServer()
}

// UnimplementedOrderServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedOrderServer struct{}

func (UnimplementedOrderServer) Browse(context.Context, *Order_Browse_Req) (*Order_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedOrderServer) BrowseMany(context.Context, *Order_BrowseMany_Req) (*Order_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedOrderServer) Count(context.Context, *Order_Count_Req) (*Order_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedOrderServer) Create(context.Context, *Order_Create_Req) (*Order_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedOrderServer) CreateMany(context.Context, *Order_CreateMany_Req) (*Order_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedOrderServer) DefaultGet(context.Context, *Order_DefaultGet_Req) (*Order_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedOrderServer) Delete(context.Context, *Order_Delete_Req) (*Order_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedOrderServer) DeleteById(context.Context, *Order_DeleteById_Req) (*Order_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedOrderServer) Onchange(context.Context, *Order_Onchange_Req) (*Order_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedOrderServer) ReadGroup(context.Context, *Order_ReadGroup_Req) (*Order_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedOrderServer) ReadGroupCount(context.Context, *Order_ReadGroupCount_Req) (*Order_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedOrderServer) Search(context.Context, *Order_Search_Req) (*Order_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedOrderServer) Update(context.Context, *Order_Update_Req) (*Order_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedOrderServer) UpdateById(context.Context, *Order_UpdateById_Req) (*Order_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedOrderServer) mustEmbedUnimplementedOrderServer() {}
func (UnimplementedOrderServer) testEmbeddedByValue()               {}

// UnsafeOrderServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to OrderServer will
// result in compilation errors.
type UnsafeOrderServer interface {
	mustEmbedUnimplementedOrderServer()
}

func RegisterOrderServer(s grpc.ServiceRegistrar, srv OrderServer) {
	// If the following call panics, it indicates UnimplementedOrderServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Order_ServiceDesc, srv)
}

func _Order_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).Browse(ctx, req.(*Order_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).BrowseMany(ctx, req.(*Order_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).Count(ctx, req.(*Order_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).Create(ctx, req.(*Order_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).CreateMany(ctx, req.(*Order_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).DefaultGet(ctx, req.(*Order_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).Delete(ctx, req.(*Order_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).DeleteById(ctx, req.(*Order_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).Onchange(ctx, req.(*Order_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).ReadGroup(ctx, req.(*Order_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).ReadGroupCount(ctx, req.(*Order_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).Search(ctx, req.(*Order_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).Update(ctx, req.(*Order_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Order_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Order_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Order_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderServer).UpdateById(ctx, req.(*Order_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// Order_ServiceDesc is the grpc.ServiceDesc for Order service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Order_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.Order",
	HandlerType: (*OrderServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _Order_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _Order_BrowseMany_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _Order_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _Order_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _Order_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _Order_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _Order_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _Order_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _Order_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _Order_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _Order_ReadGroupCount_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _Order_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _Order_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _Order_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	OrderLine_Browse_FullMethodName         = "/auth.OrderLine/Browse"
	OrderLine_BrowseMany_FullMethodName     = "/auth.OrderLine/BrowseMany"
	OrderLine_Count_FullMethodName          = "/auth.OrderLine/Count"
	OrderLine_Create_FullMethodName         = "/auth.OrderLine/Create"
	OrderLine_CreateMany_FullMethodName     = "/auth.OrderLine/CreateMany"
	OrderLine_DefaultGet_FullMethodName     = "/auth.OrderLine/DefaultGet"
	OrderLine_Delete_FullMethodName         = "/auth.OrderLine/Delete"
	OrderLine_DeleteById_FullMethodName     = "/auth.OrderLine/DeleteById"
	OrderLine_Onchange_FullMethodName       = "/auth.OrderLine/Onchange"
	OrderLine_ReadGroup_FullMethodName      = "/auth.OrderLine/ReadGroup"
	OrderLine_ReadGroupCount_FullMethodName = "/auth.OrderLine/ReadGroupCount"
	OrderLine_Search_FullMethodName         = "/auth.OrderLine/Search"
	OrderLine_Update_FullMethodName         = "/auth.OrderLine/Update"
	OrderLine_UpdateById_FullMethodName     = "/auth.OrderLine/UpdateById"
)

// OrderLineClient is the client API for OrderLine service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: OrderLine
type OrderLineClient interface {
	Browse(ctx context.Context, in *OrderLine_Browse_Req, opts ...grpc.CallOption) (*OrderLine_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *OrderLine_BrowseMany_Req, opts ...grpc.CallOption) (*OrderLine_BrowseMany_Resp, error)
	Count(ctx context.Context, in *OrderLine_Count_Req, opts ...grpc.CallOption) (*OrderLine_Count_Resp, error)
	Create(ctx context.Context, in *OrderLine_Create_Req, opts ...grpc.CallOption) (*OrderLine_Create_Resp, error)
	CreateMany(ctx context.Context, in *OrderLine_CreateMany_Req, opts ...grpc.CallOption) (*OrderLine_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *OrderLine_DefaultGet_Req, opts ...grpc.CallOption) (*OrderLine_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *OrderLine_Delete_Req, opts ...grpc.CallOption) (*OrderLine_Delete_Resp, error)
	DeleteById(ctx context.Context, in *OrderLine_DeleteById_Req, opts ...grpc.CallOption) (*OrderLine_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *OrderLine_Onchange_Req, opts ...grpc.CallOption) (*OrderLine_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *OrderLine_ReadGroup_Req, opts ...grpc.CallOption) (*OrderLine_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *OrderLine_ReadGroupCount_Req, opts ...grpc.CallOption) (*OrderLine_ReadGroupCount_Resp, error)
	Search(ctx context.Context, in *OrderLine_Search_Req, opts ...grpc.CallOption) (*OrderLine_Search_Resp, error)
	Update(ctx context.Context, in *OrderLine_Update_Req, opts ...grpc.CallOption) (*OrderLine_Update_Resp, error)
	UpdateById(ctx context.Context, in *OrderLine_UpdateById_Req, opts ...grpc.CallOption) (*OrderLine_UpdateById_Resp, error)
}

type orderLineClient struct {
	cc grpc.ClientConnInterface
}

func NewOrderLineClient(cc grpc.ClientConnInterface) OrderLineClient {
	return &orderLineClient{cc}
}

func (c *orderLineClient) Browse(ctx context.Context, in *OrderLine_Browse_Req, opts ...grpc.CallOption) (*OrderLine_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_Browse_Resp)
	err := c.cc.Invoke(ctx, OrderLine_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) BrowseMany(ctx context.Context, in *OrderLine_BrowseMany_Req, opts ...grpc.CallOption) (*OrderLine_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, OrderLine_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) Count(ctx context.Context, in *OrderLine_Count_Req, opts ...grpc.CallOption) (*OrderLine_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_Count_Resp)
	err := c.cc.Invoke(ctx, OrderLine_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) Create(ctx context.Context, in *OrderLine_Create_Req, opts ...grpc.CallOption) (*OrderLine_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_Create_Resp)
	err := c.cc.Invoke(ctx, OrderLine_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) CreateMany(ctx context.Context, in *OrderLine_CreateMany_Req, opts ...grpc.CallOption) (*OrderLine_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_CreateMany_Resp)
	err := c.cc.Invoke(ctx, OrderLine_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) DefaultGet(ctx context.Context, in *OrderLine_DefaultGet_Req, opts ...grpc.CallOption) (*OrderLine_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, OrderLine_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) Delete(ctx context.Context, in *OrderLine_Delete_Req, opts ...grpc.CallOption) (*OrderLine_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_Delete_Resp)
	err := c.cc.Invoke(ctx, OrderLine_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) DeleteById(ctx context.Context, in *OrderLine_DeleteById_Req, opts ...grpc.CallOption) (*OrderLine_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_DeleteById_Resp)
	err := c.cc.Invoke(ctx, OrderLine_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) Onchange(ctx context.Context, in *OrderLine_Onchange_Req, opts ...grpc.CallOption) (*OrderLine_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_Onchange_Resp)
	err := c.cc.Invoke(ctx, OrderLine_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) ReadGroup(ctx context.Context, in *OrderLine_ReadGroup_Req, opts ...grpc.CallOption) (*OrderLine_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, OrderLine_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) ReadGroupCount(ctx context.Context, in *OrderLine_ReadGroupCount_Req, opts ...grpc.CallOption) (*OrderLine_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, OrderLine_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) Search(ctx context.Context, in *OrderLine_Search_Req, opts ...grpc.CallOption) (*OrderLine_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_Search_Resp)
	err := c.cc.Invoke(ctx, OrderLine_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) Update(ctx context.Context, in *OrderLine_Update_Req, opts ...grpc.CallOption) (*OrderLine_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_Update_Resp)
	err := c.cc.Invoke(ctx, OrderLine_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *orderLineClient) UpdateById(ctx context.Context, in *OrderLine_UpdateById_Req, opts ...grpc.CallOption) (*OrderLine_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(OrderLine_UpdateById_Resp)
	err := c.cc.Invoke(ctx, OrderLine_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// OrderLineServer is the server API for OrderLine service.
// All implementations must embed UnimplementedOrderLineServer
// for forward compatibility.
//
// Model: OrderLine
type OrderLineServer interface {
	Browse(context.Context, *OrderLine_Browse_Req) (*OrderLine_Browse_Resp, error)
	BrowseMany(context.Context, *OrderLine_BrowseMany_Req) (*OrderLine_BrowseMany_Resp, error)
	Count(context.Context, *OrderLine_Count_Req) (*OrderLine_Count_Resp, error)
	Create(context.Context, *OrderLine_Create_Req) (*OrderLine_Create_Resp, error)
	CreateMany(context.Context, *OrderLine_CreateMany_Req) (*OrderLine_CreateMany_Resp, error)
	DefaultGet(context.Context, *OrderLine_DefaultGet_Req) (*OrderLine_DefaultGet_Resp, error)
	Delete(context.Context, *OrderLine_Delete_Req) (*OrderLine_Delete_Resp, error)
	DeleteById(context.Context, *OrderLine_DeleteById_Req) (*OrderLine_DeleteById_Resp, error)
	Onchange(context.Context, *OrderLine_Onchange_Req) (*OrderLine_Onchange_Resp, error)
	ReadGroup(context.Context, *OrderLine_ReadGroup_Req) (*OrderLine_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *OrderLine_ReadGroupCount_Req) (*OrderLine_ReadGroupCount_Resp, error)
	Search(context.Context, *OrderLine_Search_Req) (*OrderLine_Search_Resp, error)
	Update(context.Context, *OrderLine_Update_Req) (*OrderLine_Update_Resp, error)
	UpdateById(context.Context, *OrderLine_UpdateById_Req) (*OrderLine_UpdateById_Resp, error)
	mustEmbedUnimplementedOrderLineServer()
}

// UnimplementedOrderLineServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedOrderLineServer struct{}

func (UnimplementedOrderLineServer) Browse(context.Context, *OrderLine_Browse_Req) (*OrderLine_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedOrderLineServer) BrowseMany(context.Context, *OrderLine_BrowseMany_Req) (*OrderLine_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedOrderLineServer) Count(context.Context, *OrderLine_Count_Req) (*OrderLine_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedOrderLineServer) Create(context.Context, *OrderLine_Create_Req) (*OrderLine_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedOrderLineServer) CreateMany(context.Context, *OrderLine_CreateMany_Req) (*OrderLine_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedOrderLineServer) DefaultGet(context.Context, *OrderLine_DefaultGet_Req) (*OrderLine_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedOrderLineServer) Delete(context.Context, *OrderLine_Delete_Req) (*OrderLine_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedOrderLineServer) DeleteById(context.Context, *OrderLine_DeleteById_Req) (*OrderLine_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedOrderLineServer) Onchange(context.Context, *OrderLine_Onchange_Req) (*OrderLine_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedOrderLineServer) ReadGroup(context.Context, *OrderLine_ReadGroup_Req) (*OrderLine_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedOrderLineServer) ReadGroupCount(context.Context, *OrderLine_ReadGroupCount_Req) (*OrderLine_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedOrderLineServer) Search(context.Context, *OrderLine_Search_Req) (*OrderLine_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedOrderLineServer) Update(context.Context, *OrderLine_Update_Req) (*OrderLine_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedOrderLineServer) UpdateById(context.Context, *OrderLine_UpdateById_Req) (*OrderLine_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedOrderLineServer) mustEmbedUnimplementedOrderLineServer() {}
func (UnimplementedOrderLineServer) testEmbeddedByValue()                   {}

// UnsafeOrderLineServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to OrderLineServer will
// result in compilation errors.
type UnsafeOrderLineServer interface {
	mustEmbedUnimplementedOrderLineServer()
}

func RegisterOrderLineServer(s grpc.ServiceRegistrar, srv OrderLineServer) {
	// If the following call panics, it indicates UnimplementedOrderLineServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&OrderLine_ServiceDesc, srv)
}

func _OrderLine_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).Browse(ctx, req.(*OrderLine_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).BrowseMany(ctx, req.(*OrderLine_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).Count(ctx, req.(*OrderLine_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).Create(ctx, req.(*OrderLine_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).CreateMany(ctx, req.(*OrderLine_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).DefaultGet(ctx, req.(*OrderLine_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).Delete(ctx, req.(*OrderLine_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).DeleteById(ctx, req.(*OrderLine_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).Onchange(ctx, req.(*OrderLine_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).ReadGroup(ctx, req.(*OrderLine_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).ReadGroupCount(ctx, req.(*OrderLine_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).Search(ctx, req.(*OrderLine_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).Update(ctx, req.(*OrderLine_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _OrderLine_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OrderLine_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OrderLineServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: OrderLine_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OrderLineServer).UpdateById(ctx, req.(*OrderLine_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// OrderLine_ServiceDesc is the grpc.ServiceDesc for OrderLine service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var OrderLine_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.OrderLine",
	HandlerType: (*OrderLineServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _OrderLine_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _OrderLine_BrowseMany_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _OrderLine_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _OrderLine_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _OrderLine_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _OrderLine_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _OrderLine_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _OrderLine_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _OrderLine_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _OrderLine_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _OrderLine_ReadGroupCount_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _OrderLine_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _OrderLine_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _OrderLine_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	Role_Browse_FullMethodName            = "/auth.Role/Browse"
	Role_BrowseMany_FullMethodName        = "/auth.Role/BrowseMany"
	Role_Count_FullMethodName             = "/auth.Role/Count"
	Role_Create_FullMethodName            = "/auth.Role/Create"
	Role_CreateIfNotExists_FullMethodName = "/auth.Role/CreateIfNotExists"
	Role_CreateMany_FullMethodName        = "/auth.Role/CreateMany"
	Role_DefaultGet_FullMethodName        = "/auth.Role/DefaultGet"
	Role_Delete_FullMethodName            = "/auth.Role/Delete"
	Role_DeleteById_FullMethodName        = "/auth.Role/DeleteById"
	Role_Onchange_FullMethodName          = "/auth.Role/Onchange"
	Role_ReadGroup_FullMethodName         = "/auth.Role/ReadGroup"
	Role_ReadGroupCount_FullMethodName    = "/auth.Role/ReadGroupCount"
	Role_Search_FullMethodName            = "/auth.Role/Search"
	Role_Update_FullMethodName            = "/auth.Role/Update"
	Role_UpdateById_FullMethodName        = "/auth.Role/UpdateById"
)

// RoleClient is the client API for Role service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: Role
type RoleClient interface {
	Browse(ctx context.Context, in *Role_Browse_Req, opts ...grpc.CallOption) (*Role_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *Role_BrowseMany_Req, opts ...grpc.CallOption) (*Role_BrowseMany_Resp, error)
	Count(ctx context.Context, in *Role_Count_Req, opts ...grpc.CallOption) (*Role_Count_Resp, error)
	Create(ctx context.Context, in *Role_Create_Req, opts ...grpc.CallOption) (*Role_Create_Resp, error)
	CreateIfNotExists(ctx context.Context, in *Role_CreateIfNotExists_Req, opts ...grpc.CallOption) (*Role_CreateIfNotExists_Resp, error)
	CreateMany(ctx context.Context, in *Role_CreateMany_Req, opts ...grpc.CallOption) (*Role_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *Role_DefaultGet_Req, opts ...grpc.CallOption) (*Role_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *Role_Delete_Req, opts ...grpc.CallOption) (*Role_Delete_Resp, error)
	DeleteById(ctx context.Context, in *Role_DeleteById_Req, opts ...grpc.CallOption) (*Role_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *Role_Onchange_Req, opts ...grpc.CallOption) (*Role_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *Role_ReadGroup_Req, opts ...grpc.CallOption) (*Role_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *Role_ReadGroupCount_Req, opts ...grpc.CallOption) (*Role_ReadGroupCount_Resp, error)
	Search(ctx context.Context, in *Role_Search_Req, opts ...grpc.CallOption) (*Role_Search_Resp, error)
	Update(ctx context.Context, in *Role_Update_Req, opts ...grpc.CallOption) (*Role_Update_Resp, error)
	UpdateById(ctx context.Context, in *Role_UpdateById_Req, opts ...grpc.CallOption) (*Role_UpdateById_Resp, error)
}

type roleClient struct {
	cc grpc.ClientConnInterface
}

func NewRoleClient(cc grpc.ClientConnInterface) RoleClient {
	return &roleClient{cc}
}

func (c *roleClient) Browse(ctx context.Context, in *Role_Browse_Req, opts ...grpc.CallOption) (*Role_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_Browse_Resp)
	err := c.cc.Invoke(ctx, Role_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) BrowseMany(ctx context.Context, in *Role_BrowseMany_Req, opts ...grpc.CallOption) (*Role_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, Role_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) Count(ctx context.Context, in *Role_Count_Req, opts ...grpc.CallOption) (*Role_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_Count_Resp)
	err := c.cc.Invoke(ctx, Role_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) Create(ctx context.Context, in *Role_Create_Req, opts ...grpc.CallOption) (*Role_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_Create_Resp)
	err := c.cc.Invoke(ctx, Role_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) CreateIfNotExists(ctx context.Context, in *Role_CreateIfNotExists_Req, opts ...grpc.CallOption) (*Role_CreateIfNotExists_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_CreateIfNotExists_Resp)
	err := c.cc.Invoke(ctx, Role_CreateIfNotExists_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) CreateMany(ctx context.Context, in *Role_CreateMany_Req, opts ...grpc.CallOption) (*Role_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_CreateMany_Resp)
	err := c.cc.Invoke(ctx, Role_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) DefaultGet(ctx context.Context, in *Role_DefaultGet_Req, opts ...grpc.CallOption) (*Role_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, Role_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) Delete(ctx context.Context, in *Role_Delete_Req, opts ...grpc.CallOption) (*Role_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_Delete_Resp)
	err := c.cc.Invoke(ctx, Role_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) DeleteById(ctx context.Context, in *Role_DeleteById_Req, opts ...grpc.CallOption) (*Role_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_DeleteById_Resp)
	err := c.cc.Invoke(ctx, Role_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) Onchange(ctx context.Context, in *Role_Onchange_Req, opts ...grpc.CallOption) (*Role_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_Onchange_Resp)
	err := c.cc.Invoke(ctx, Role_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) ReadGroup(ctx context.Context, in *Role_ReadGroup_Req, opts ...grpc.CallOption) (*Role_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, Role_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) ReadGroupCount(ctx context.Context, in *Role_ReadGroupCount_Req, opts ...grpc.CallOption) (*Role_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, Role_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) Search(ctx context.Context, in *Role_Search_Req, opts ...grpc.CallOption) (*Role_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_Search_Resp)
	err := c.cc.Invoke(ctx, Role_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) Update(ctx context.Context, in *Role_Update_Req, opts ...grpc.CallOption) (*Role_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_Update_Resp)
	err := c.cc.Invoke(ctx, Role_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleClient) UpdateById(ctx context.Context, in *Role_UpdateById_Req, opts ...grpc.CallOption) (*Role_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Role_UpdateById_Resp)
	err := c.cc.Invoke(ctx, Role_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RoleServer is the server API for Role service.
// All implementations must embed UnimplementedRoleServer
// for forward compatibility.
//
// Model: Role
type RoleServer interface {
	Browse(context.Context, *Role_Browse_Req) (*Role_Browse_Resp, error)
	BrowseMany(context.Context, *Role_BrowseMany_Req) (*Role_BrowseMany_Resp, error)
	Count(context.Context, *Role_Count_Req) (*Role_Count_Resp, error)
	Create(context.Context, *Role_Create_Req) (*Role_Create_Resp, error)
	CreateIfNotExists(context.Context, *Role_CreateIfNotExists_Req) (*Role_CreateIfNotExists_Resp, error)
	CreateMany(context.Context, *Role_CreateMany_Req) (*Role_CreateMany_Resp, error)
	DefaultGet(context.Context, *Role_DefaultGet_Req) (*Role_DefaultGet_Resp, error)
	Delete(context.Context, *Role_Delete_Req) (*Role_Delete_Resp, error)
	DeleteById(context.Context, *Role_DeleteById_Req) (*Role_DeleteById_Resp, error)
	Onchange(context.Context, *Role_Onchange_Req) (*Role_Onchange_Resp, error)
	ReadGroup(context.Context, *Role_ReadGroup_Req) (*Role_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *Role_ReadGroupCount_Req) (*Role_ReadGroupCount_Resp, error)
	Search(context.Context, *Role_Search_Req) (*Role_Search_Resp, error)
	Update(context.Context, *Role_Update_Req) (*Role_Update_Resp, error)
	UpdateById(context.Context, *Role_UpdateById_Req) (*Role_UpdateById_Resp, error)
	mustEmbedUnimplementedRoleServer()
}

// UnimplementedRoleServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedRoleServer struct{}

func (UnimplementedRoleServer) Browse(context.Context, *Role_Browse_Req) (*Role_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedRoleServer) BrowseMany(context.Context, *Role_BrowseMany_Req) (*Role_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedRoleServer) Count(context.Context, *Role_Count_Req) (*Role_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedRoleServer) Create(context.Context, *Role_Create_Req) (*Role_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedRoleServer) CreateIfNotExists(context.Context, *Role_CreateIfNotExists_Req) (*Role_CreateIfNotExists_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateIfNotExists not implemented")
}
func (UnimplementedRoleServer) CreateMany(context.Context, *Role_CreateMany_Req) (*Role_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedRoleServer) DefaultGet(context.Context, *Role_DefaultGet_Req) (*Role_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedRoleServer) Delete(context.Context, *Role_Delete_Req) (*Role_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedRoleServer) DeleteById(context.Context, *Role_DeleteById_Req) (*Role_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedRoleServer) Onchange(context.Context, *Role_Onchange_Req) (*Role_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedRoleServer) ReadGroup(context.Context, *Role_ReadGroup_Req) (*Role_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedRoleServer) ReadGroupCount(context.Context, *Role_ReadGroupCount_Req) (*Role_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedRoleServer) Search(context.Context, *Role_Search_Req) (*Role_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedRoleServer) Update(context.Context, *Role_Update_Req) (*Role_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedRoleServer) UpdateById(context.Context, *Role_UpdateById_Req) (*Role_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedRoleServer) mustEmbedUnimplementedRoleServer() {}
func (UnimplementedRoleServer) testEmbeddedByValue()              {}

// UnsafeRoleServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RoleServer will
// result in compilation errors.
type UnsafeRoleServer interface {
	mustEmbedUnimplementedRoleServer()
}

func RegisterRoleServer(s grpc.ServiceRegistrar, srv RoleServer) {
	// If the following call panics, it indicates UnimplementedRoleServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Role_ServiceDesc, srv)
}

func _Role_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).Browse(ctx, req.(*Role_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).BrowseMany(ctx, req.(*Role_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).Count(ctx, req.(*Role_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).Create(ctx, req.(*Role_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_CreateIfNotExists_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_CreateIfNotExists_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).CreateIfNotExists(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_CreateIfNotExists_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).CreateIfNotExists(ctx, req.(*Role_CreateIfNotExists_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).CreateMany(ctx, req.(*Role_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).DefaultGet(ctx, req.(*Role_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).Delete(ctx, req.(*Role_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).DeleteById(ctx, req.(*Role_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).Onchange(ctx, req.(*Role_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).ReadGroup(ctx, req.(*Role_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).ReadGroupCount(ctx, req.(*Role_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).Search(ctx, req.(*Role_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).Update(ctx, req.(*Role_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Role_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Role_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Role_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleServer).UpdateById(ctx, req.(*Role_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// Role_ServiceDesc is the grpc.ServiceDesc for Role service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Role_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.Role",
	HandlerType: (*RoleServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _Role_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _Role_BrowseMany_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _Role_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _Role_Create_Handler,
		},
		{
			MethodName: "CreateIfNotExists",
			Handler:    _Role_CreateIfNotExists_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _Role_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _Role_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _Role_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _Role_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _Role_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _Role_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _Role_ReadGroupCount_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _Role_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _Role_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _Role_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	RoleFieldRule_Browse_FullMethodName         = "/auth.RoleFieldRule/Browse"
	RoleFieldRule_BrowseMany_FullMethodName     = "/auth.RoleFieldRule/BrowseMany"
	RoleFieldRule_Count_FullMethodName          = "/auth.RoleFieldRule/Count"
	RoleFieldRule_Create_FullMethodName         = "/auth.RoleFieldRule/Create"
	RoleFieldRule_CreateMany_FullMethodName     = "/auth.RoleFieldRule/CreateMany"
	RoleFieldRule_DefaultGet_FullMethodName     = "/auth.RoleFieldRule/DefaultGet"
	RoleFieldRule_Delete_FullMethodName         = "/auth.RoleFieldRule/Delete"
	RoleFieldRule_DeleteById_FullMethodName     = "/auth.RoleFieldRule/DeleteById"
	RoleFieldRule_Onchange_FullMethodName       = "/auth.RoleFieldRule/Onchange"
	RoleFieldRule_ReadGroup_FullMethodName      = "/auth.RoleFieldRule/ReadGroup"
	RoleFieldRule_ReadGroupCount_FullMethodName = "/auth.RoleFieldRule/ReadGroupCount"
	RoleFieldRule_Search_FullMethodName         = "/auth.RoleFieldRule/Search"
	RoleFieldRule_Update_FullMethodName         = "/auth.RoleFieldRule/Update"
	RoleFieldRule_UpdateById_FullMethodName     = "/auth.RoleFieldRule/UpdateById"
)

// RoleFieldRuleClient is the client API for RoleFieldRule service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: RoleFieldRule
type RoleFieldRuleClient interface {
	Browse(ctx context.Context, in *RoleFieldRule_Browse_Req, opts ...grpc.CallOption) (*RoleFieldRule_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *RoleFieldRule_BrowseMany_Req, opts ...grpc.CallOption) (*RoleFieldRule_BrowseMany_Resp, error)
	Count(ctx context.Context, in *RoleFieldRule_Count_Req, opts ...grpc.CallOption) (*RoleFieldRule_Count_Resp, error)
	Create(ctx context.Context, in *RoleFieldRule_Create_Req, opts ...grpc.CallOption) (*RoleFieldRule_Create_Resp, error)
	CreateMany(ctx context.Context, in *RoleFieldRule_CreateMany_Req, opts ...grpc.CallOption) (*RoleFieldRule_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *RoleFieldRule_DefaultGet_Req, opts ...grpc.CallOption) (*RoleFieldRule_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *RoleFieldRule_Delete_Req, opts ...grpc.CallOption) (*RoleFieldRule_Delete_Resp, error)
	DeleteById(ctx context.Context, in *RoleFieldRule_DeleteById_Req, opts ...grpc.CallOption) (*RoleFieldRule_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *RoleFieldRule_Onchange_Req, opts ...grpc.CallOption) (*RoleFieldRule_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *RoleFieldRule_ReadGroup_Req, opts ...grpc.CallOption) (*RoleFieldRule_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *RoleFieldRule_ReadGroupCount_Req, opts ...grpc.CallOption) (*RoleFieldRule_ReadGroupCount_Resp, error)
	Search(ctx context.Context, in *RoleFieldRule_Search_Req, opts ...grpc.CallOption) (*RoleFieldRule_Search_Resp, error)
	Update(ctx context.Context, in *RoleFieldRule_Update_Req, opts ...grpc.CallOption) (*RoleFieldRule_Update_Resp, error)
	UpdateById(ctx context.Context, in *RoleFieldRule_UpdateById_Req, opts ...grpc.CallOption) (*RoleFieldRule_UpdateById_Resp, error)
}

type roleFieldRuleClient struct {
	cc grpc.ClientConnInterface
}

func NewRoleFieldRuleClient(cc grpc.ClientConnInterface) RoleFieldRuleClient {
	return &roleFieldRuleClient{cc}
}

func (c *roleFieldRuleClient) Browse(ctx context.Context, in *RoleFieldRule_Browse_Req, opts ...grpc.CallOption) (*RoleFieldRule_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_Browse_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) BrowseMany(ctx context.Context, in *RoleFieldRule_BrowseMany_Req, opts ...grpc.CallOption) (*RoleFieldRule_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) Count(ctx context.Context, in *RoleFieldRule_Count_Req, opts ...grpc.CallOption) (*RoleFieldRule_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_Count_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) Create(ctx context.Context, in *RoleFieldRule_Create_Req, opts ...grpc.CallOption) (*RoleFieldRule_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_Create_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) CreateMany(ctx context.Context, in *RoleFieldRule_CreateMany_Req, opts ...grpc.CallOption) (*RoleFieldRule_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_CreateMany_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) DefaultGet(ctx context.Context, in *RoleFieldRule_DefaultGet_Req, opts ...grpc.CallOption) (*RoleFieldRule_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) Delete(ctx context.Context, in *RoleFieldRule_Delete_Req, opts ...grpc.CallOption) (*RoleFieldRule_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_Delete_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) DeleteById(ctx context.Context, in *RoleFieldRule_DeleteById_Req, opts ...grpc.CallOption) (*RoleFieldRule_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_DeleteById_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) Onchange(ctx context.Context, in *RoleFieldRule_Onchange_Req, opts ...grpc.CallOption) (*RoleFieldRule_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_Onchange_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) ReadGroup(ctx context.Context, in *RoleFieldRule_ReadGroup_Req, opts ...grpc.CallOption) (*RoleFieldRule_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) ReadGroupCount(ctx context.Context, in *RoleFieldRule_ReadGroupCount_Req, opts ...grpc.CallOption) (*RoleFieldRule_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) Search(ctx context.Context, in *RoleFieldRule_Search_Req, opts ...grpc.CallOption) (*RoleFieldRule_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_Search_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) Update(ctx context.Context, in *RoleFieldRule_Update_Req, opts ...grpc.CallOption) (*RoleFieldRule_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_Update_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleFieldRuleClient) UpdateById(ctx context.Context, in *RoleFieldRule_UpdateById_Req, opts ...grpc.CallOption) (*RoleFieldRule_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleFieldRule_UpdateById_Resp)
	err := c.cc.Invoke(ctx, RoleFieldRule_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RoleFieldRuleServer is the server API for RoleFieldRule service.
// All implementations must embed UnimplementedRoleFieldRuleServer
// for forward compatibility.
//
// Model: RoleFieldRule
type RoleFieldRuleServer interface {
	Browse(context.Context, *RoleFieldRule_Browse_Req) (*RoleFieldRule_Browse_Resp, error)
	BrowseMany(context.Context, *RoleFieldRule_BrowseMany_Req) (*RoleFieldRule_BrowseMany_Resp, error)
	Count(context.Context, *RoleFieldRule_Count_Req) (*RoleFieldRule_Count_Resp, error)
	Create(context.Context, *RoleFieldRule_Create_Req) (*RoleFieldRule_Create_Resp, error)
	CreateMany(context.Context, *RoleFieldRule_CreateMany_Req) (*RoleFieldRule_CreateMany_Resp, error)
	DefaultGet(context.Context, *RoleFieldRule_DefaultGet_Req) (*RoleFieldRule_DefaultGet_Resp, error)
	Delete(context.Context, *RoleFieldRule_Delete_Req) (*RoleFieldRule_Delete_Resp, error)
	DeleteById(context.Context, *RoleFieldRule_DeleteById_Req) (*RoleFieldRule_DeleteById_Resp, error)
	Onchange(context.Context, *RoleFieldRule_Onchange_Req) (*RoleFieldRule_Onchange_Resp, error)
	ReadGroup(context.Context, *RoleFieldRule_ReadGroup_Req) (*RoleFieldRule_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *RoleFieldRule_ReadGroupCount_Req) (*RoleFieldRule_ReadGroupCount_Resp, error)
	Search(context.Context, *RoleFieldRule_Search_Req) (*RoleFieldRule_Search_Resp, error)
	Update(context.Context, *RoleFieldRule_Update_Req) (*RoleFieldRule_Update_Resp, error)
	UpdateById(context.Context, *RoleFieldRule_UpdateById_Req) (*RoleFieldRule_UpdateById_Resp, error)
	mustEmbedUnimplementedRoleFieldRuleServer()
}

// UnimplementedRoleFieldRuleServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedRoleFieldRuleServer struct{}

func (UnimplementedRoleFieldRuleServer) Browse(context.Context, *RoleFieldRule_Browse_Req) (*RoleFieldRule_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedRoleFieldRuleServer) BrowseMany(context.Context, *RoleFieldRule_BrowseMany_Req) (*RoleFieldRule_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedRoleFieldRuleServer) Count(context.Context, *RoleFieldRule_Count_Req) (*RoleFieldRule_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedRoleFieldRuleServer) Create(context.Context, *RoleFieldRule_Create_Req) (*RoleFieldRule_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedRoleFieldRuleServer) CreateMany(context.Context, *RoleFieldRule_CreateMany_Req) (*RoleFieldRule_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedRoleFieldRuleServer) DefaultGet(context.Context, *RoleFieldRule_DefaultGet_Req) (*RoleFieldRule_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedRoleFieldRuleServer) Delete(context.Context, *RoleFieldRule_Delete_Req) (*RoleFieldRule_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedRoleFieldRuleServer) DeleteById(context.Context, *RoleFieldRule_DeleteById_Req) (*RoleFieldRule_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedRoleFieldRuleServer) Onchange(context.Context, *RoleFieldRule_Onchange_Req) (*RoleFieldRule_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedRoleFieldRuleServer) ReadGroup(context.Context, *RoleFieldRule_ReadGroup_Req) (*RoleFieldRule_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedRoleFieldRuleServer) ReadGroupCount(context.Context, *RoleFieldRule_ReadGroupCount_Req) (*RoleFieldRule_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedRoleFieldRuleServer) Search(context.Context, *RoleFieldRule_Search_Req) (*RoleFieldRule_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedRoleFieldRuleServer) Update(context.Context, *RoleFieldRule_Update_Req) (*RoleFieldRule_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedRoleFieldRuleServer) UpdateById(context.Context, *RoleFieldRule_UpdateById_Req) (*RoleFieldRule_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedRoleFieldRuleServer) mustEmbedUnimplementedRoleFieldRuleServer() {}
func (UnimplementedRoleFieldRuleServer) testEmbeddedByValue()                       {}

// UnsafeRoleFieldRuleServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RoleFieldRuleServer will
// result in compilation errors.
type UnsafeRoleFieldRuleServer interface {
	mustEmbedUnimplementedRoleFieldRuleServer()
}

func RegisterRoleFieldRuleServer(s grpc.ServiceRegistrar, srv RoleFieldRuleServer) {
	// If the following call panics, it indicates UnimplementedRoleFieldRuleServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&RoleFieldRule_ServiceDesc, srv)
}

func _RoleFieldRule_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).Browse(ctx, req.(*RoleFieldRule_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).BrowseMany(ctx, req.(*RoleFieldRule_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).Count(ctx, req.(*RoleFieldRule_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).Create(ctx, req.(*RoleFieldRule_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).CreateMany(ctx, req.(*RoleFieldRule_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).DefaultGet(ctx, req.(*RoleFieldRule_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).Delete(ctx, req.(*RoleFieldRule_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).DeleteById(ctx, req.(*RoleFieldRule_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).Onchange(ctx, req.(*RoleFieldRule_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).ReadGroup(ctx, req.(*RoleFieldRule_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).ReadGroupCount(ctx, req.(*RoleFieldRule_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).Search(ctx, req.(*RoleFieldRule_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).Update(ctx, req.(*RoleFieldRule_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleFieldRule_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleFieldRule_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleFieldRuleServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleFieldRule_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleFieldRuleServer).UpdateById(ctx, req.(*RoleFieldRule_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// RoleFieldRule_ServiceDesc is the grpc.ServiceDesc for RoleFieldRule service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RoleFieldRule_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.RoleFieldRule",
	HandlerType: (*RoleFieldRuleServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _RoleFieldRule_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _RoleFieldRule_BrowseMany_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _RoleFieldRule_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _RoleFieldRule_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _RoleFieldRule_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _RoleFieldRule_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _RoleFieldRule_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _RoleFieldRule_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _RoleFieldRule_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _RoleFieldRule_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _RoleFieldRule_ReadGroupCount_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _RoleFieldRule_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _RoleFieldRule_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _RoleFieldRule_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	RoleInheritance_Browse_FullMethodName         = "/auth.RoleInheritance/Browse"
	RoleInheritance_BrowseMany_FullMethodName     = "/auth.RoleInheritance/BrowseMany"
	RoleInheritance_Count_FullMethodName          = "/auth.RoleInheritance/Count"
	RoleInheritance_Create_FullMethodName         = "/auth.RoleInheritance/Create"
	RoleInheritance_CreateMany_FullMethodName     = "/auth.RoleInheritance/CreateMany"
	RoleInheritance_DefaultGet_FullMethodName     = "/auth.RoleInheritance/DefaultGet"
	RoleInheritance_Delete_FullMethodName         = "/auth.RoleInheritance/Delete"
	RoleInheritance_DeleteById_FullMethodName     = "/auth.RoleInheritance/DeleteById"
	RoleInheritance_Onchange_FullMethodName       = "/auth.RoleInheritance/Onchange"
	RoleInheritance_ReadGroup_FullMethodName      = "/auth.RoleInheritance/ReadGroup"
	RoleInheritance_ReadGroupCount_FullMethodName = "/auth.RoleInheritance/ReadGroupCount"
	RoleInheritance_Search_FullMethodName         = "/auth.RoleInheritance/Search"
	RoleInheritance_Update_FullMethodName         = "/auth.RoleInheritance/Update"
	RoleInheritance_UpdateById_FullMethodName     = "/auth.RoleInheritance/UpdateById"
)

// RoleInheritanceClient is the client API for RoleInheritance service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: RoleInheritance
type RoleInheritanceClient interface {
	Browse(ctx context.Context, in *RoleInheritance_Browse_Req, opts ...grpc.CallOption) (*RoleInheritance_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *RoleInheritance_BrowseMany_Req, opts ...grpc.CallOption) (*RoleInheritance_BrowseMany_Resp, error)
	Count(ctx context.Context, in *RoleInheritance_Count_Req, opts ...grpc.CallOption) (*RoleInheritance_Count_Resp, error)
	Create(ctx context.Context, in *RoleInheritance_Create_Req, opts ...grpc.CallOption) (*RoleInheritance_Create_Resp, error)
	CreateMany(ctx context.Context, in *RoleInheritance_CreateMany_Req, opts ...grpc.CallOption) (*RoleInheritance_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *RoleInheritance_DefaultGet_Req, opts ...grpc.CallOption) (*RoleInheritance_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *RoleInheritance_Delete_Req, opts ...grpc.CallOption) (*RoleInheritance_Delete_Resp, error)
	DeleteById(ctx context.Context, in *RoleInheritance_DeleteById_Req, opts ...grpc.CallOption) (*RoleInheritance_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *RoleInheritance_Onchange_Req, opts ...grpc.CallOption) (*RoleInheritance_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *RoleInheritance_ReadGroup_Req, opts ...grpc.CallOption) (*RoleInheritance_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *RoleInheritance_ReadGroupCount_Req, opts ...grpc.CallOption) (*RoleInheritance_ReadGroupCount_Resp, error)
	Search(ctx context.Context, in *RoleInheritance_Search_Req, opts ...grpc.CallOption) (*RoleInheritance_Search_Resp, error)
	Update(ctx context.Context, in *RoleInheritance_Update_Req, opts ...grpc.CallOption) (*RoleInheritance_Update_Resp, error)
	UpdateById(ctx context.Context, in *RoleInheritance_UpdateById_Req, opts ...grpc.CallOption) (*RoleInheritance_UpdateById_Resp, error)
}

type roleInheritanceClient struct {
	cc grpc.ClientConnInterface
}

func NewRoleInheritanceClient(cc grpc.ClientConnInterface) RoleInheritanceClient {
	return &roleInheritanceClient{cc}
}

func (c *roleInheritanceClient) Browse(ctx context.Context, in *RoleInheritance_Browse_Req, opts ...grpc.CallOption) (*RoleInheritance_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_Browse_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) BrowseMany(ctx context.Context, in *RoleInheritance_BrowseMany_Req, opts ...grpc.CallOption) (*RoleInheritance_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) Count(ctx context.Context, in *RoleInheritance_Count_Req, opts ...grpc.CallOption) (*RoleInheritance_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_Count_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) Create(ctx context.Context, in *RoleInheritance_Create_Req, opts ...grpc.CallOption) (*RoleInheritance_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_Create_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) CreateMany(ctx context.Context, in *RoleInheritance_CreateMany_Req, opts ...grpc.CallOption) (*RoleInheritance_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_CreateMany_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) DefaultGet(ctx context.Context, in *RoleInheritance_DefaultGet_Req, opts ...grpc.CallOption) (*RoleInheritance_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) Delete(ctx context.Context, in *RoleInheritance_Delete_Req, opts ...grpc.CallOption) (*RoleInheritance_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_Delete_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) DeleteById(ctx context.Context, in *RoleInheritance_DeleteById_Req, opts ...grpc.CallOption) (*RoleInheritance_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_DeleteById_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) Onchange(ctx context.Context, in *RoleInheritance_Onchange_Req, opts ...grpc.CallOption) (*RoleInheritance_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_Onchange_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) ReadGroup(ctx context.Context, in *RoleInheritance_ReadGroup_Req, opts ...grpc.CallOption) (*RoleInheritance_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) ReadGroupCount(ctx context.Context, in *RoleInheritance_ReadGroupCount_Req, opts ...grpc.CallOption) (*RoleInheritance_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) Search(ctx context.Context, in *RoleInheritance_Search_Req, opts ...grpc.CallOption) (*RoleInheritance_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_Search_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) Update(ctx context.Context, in *RoleInheritance_Update_Req, opts ...grpc.CallOption) (*RoleInheritance_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_Update_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleInheritanceClient) UpdateById(ctx context.Context, in *RoleInheritance_UpdateById_Req, opts ...grpc.CallOption) (*RoleInheritance_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleInheritance_UpdateById_Resp)
	err := c.cc.Invoke(ctx, RoleInheritance_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RoleInheritanceServer is the server API for RoleInheritance service.
// All implementations must embed UnimplementedRoleInheritanceServer
// for forward compatibility.
//
// Model: RoleInheritance
type RoleInheritanceServer interface {
	Browse(context.Context, *RoleInheritance_Browse_Req) (*RoleInheritance_Browse_Resp, error)
	BrowseMany(context.Context, *RoleInheritance_BrowseMany_Req) (*RoleInheritance_BrowseMany_Resp, error)
	Count(context.Context, *RoleInheritance_Count_Req) (*RoleInheritance_Count_Resp, error)
	Create(context.Context, *RoleInheritance_Create_Req) (*RoleInheritance_Create_Resp, error)
	CreateMany(context.Context, *RoleInheritance_CreateMany_Req) (*RoleInheritance_CreateMany_Resp, error)
	DefaultGet(context.Context, *RoleInheritance_DefaultGet_Req) (*RoleInheritance_DefaultGet_Resp, error)
	Delete(context.Context, *RoleInheritance_Delete_Req) (*RoleInheritance_Delete_Resp, error)
	DeleteById(context.Context, *RoleInheritance_DeleteById_Req) (*RoleInheritance_DeleteById_Resp, error)
	Onchange(context.Context, *RoleInheritance_Onchange_Req) (*RoleInheritance_Onchange_Resp, error)
	ReadGroup(context.Context, *RoleInheritance_ReadGroup_Req) (*RoleInheritance_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *RoleInheritance_ReadGroupCount_Req) (*RoleInheritance_ReadGroupCount_Resp, error)
	Search(context.Context, *RoleInheritance_Search_Req) (*RoleInheritance_Search_Resp, error)
	Update(context.Context, *RoleInheritance_Update_Req) (*RoleInheritance_Update_Resp, error)
	UpdateById(context.Context, *RoleInheritance_UpdateById_Req) (*RoleInheritance_UpdateById_Resp, error)
	mustEmbedUnimplementedRoleInheritanceServer()
}

// UnimplementedRoleInheritanceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedRoleInheritanceServer struct{}

func (UnimplementedRoleInheritanceServer) Browse(context.Context, *RoleInheritance_Browse_Req) (*RoleInheritance_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedRoleInheritanceServer) BrowseMany(context.Context, *RoleInheritance_BrowseMany_Req) (*RoleInheritance_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedRoleInheritanceServer) Count(context.Context, *RoleInheritance_Count_Req) (*RoleInheritance_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedRoleInheritanceServer) Create(context.Context, *RoleInheritance_Create_Req) (*RoleInheritance_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedRoleInheritanceServer) CreateMany(context.Context, *RoleInheritance_CreateMany_Req) (*RoleInheritance_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedRoleInheritanceServer) DefaultGet(context.Context, *RoleInheritance_DefaultGet_Req) (*RoleInheritance_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedRoleInheritanceServer) Delete(context.Context, *RoleInheritance_Delete_Req) (*RoleInheritance_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedRoleInheritanceServer) DeleteById(context.Context, *RoleInheritance_DeleteById_Req) (*RoleInheritance_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedRoleInheritanceServer) Onchange(context.Context, *RoleInheritance_Onchange_Req) (*RoleInheritance_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedRoleInheritanceServer) ReadGroup(context.Context, *RoleInheritance_ReadGroup_Req) (*RoleInheritance_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedRoleInheritanceServer) ReadGroupCount(context.Context, *RoleInheritance_ReadGroupCount_Req) (*RoleInheritance_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedRoleInheritanceServer) Search(context.Context, *RoleInheritance_Search_Req) (*RoleInheritance_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedRoleInheritanceServer) Update(context.Context, *RoleInheritance_Update_Req) (*RoleInheritance_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedRoleInheritanceServer) UpdateById(context.Context, *RoleInheritance_UpdateById_Req) (*RoleInheritance_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedRoleInheritanceServer) mustEmbedUnimplementedRoleInheritanceServer() {}
func (UnimplementedRoleInheritanceServer) testEmbeddedByValue()                         {}

// UnsafeRoleInheritanceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RoleInheritanceServer will
// result in compilation errors.
type UnsafeRoleInheritanceServer interface {
	mustEmbedUnimplementedRoleInheritanceServer()
}

func RegisterRoleInheritanceServer(s grpc.ServiceRegistrar, srv RoleInheritanceServer) {
	// If the following call panics, it indicates UnimplementedRoleInheritanceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&RoleInheritance_ServiceDesc, srv)
}

func _RoleInheritance_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).Browse(ctx, req.(*RoleInheritance_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).BrowseMany(ctx, req.(*RoleInheritance_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).Count(ctx, req.(*RoleInheritance_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).Create(ctx, req.(*RoleInheritance_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).CreateMany(ctx, req.(*RoleInheritance_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).DefaultGet(ctx, req.(*RoleInheritance_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).Delete(ctx, req.(*RoleInheritance_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).DeleteById(ctx, req.(*RoleInheritance_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).Onchange(ctx, req.(*RoleInheritance_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).ReadGroup(ctx, req.(*RoleInheritance_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).ReadGroupCount(ctx, req.(*RoleInheritance_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).Search(ctx, req.(*RoleInheritance_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).Update(ctx, req.(*RoleInheritance_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleInheritance_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleInheritance_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleInheritanceServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleInheritance_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleInheritanceServer).UpdateById(ctx, req.(*RoleInheritance_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// RoleInheritance_ServiceDesc is the grpc.ServiceDesc for RoleInheritance service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RoleInheritance_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.RoleInheritance",
	HandlerType: (*RoleInheritanceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _RoleInheritance_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _RoleInheritance_BrowseMany_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _RoleInheritance_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _RoleInheritance_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _RoleInheritance_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _RoleInheritance_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _RoleInheritance_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _RoleInheritance_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _RoleInheritance_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _RoleInheritance_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _RoleInheritance_ReadGroupCount_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _RoleInheritance_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _RoleInheritance_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _RoleInheritance_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	RoleMethodAccess_Browse_FullMethodName         = "/auth.RoleMethodAccess/Browse"
	RoleMethodAccess_BrowseMany_FullMethodName     = "/auth.RoleMethodAccess/BrowseMany"
	RoleMethodAccess_Count_FullMethodName          = "/auth.RoleMethodAccess/Count"
	RoleMethodAccess_Create_FullMethodName         = "/auth.RoleMethodAccess/Create"
	RoleMethodAccess_CreateMany_FullMethodName     = "/auth.RoleMethodAccess/CreateMany"
	RoleMethodAccess_DefaultGet_FullMethodName     = "/auth.RoleMethodAccess/DefaultGet"
	RoleMethodAccess_Delete_FullMethodName         = "/auth.RoleMethodAccess/Delete"
	RoleMethodAccess_DeleteById_FullMethodName     = "/auth.RoleMethodAccess/DeleteById"
	RoleMethodAccess_Onchange_FullMethodName       = "/auth.RoleMethodAccess/Onchange"
	RoleMethodAccess_ReadGroup_FullMethodName      = "/auth.RoleMethodAccess/ReadGroup"
	RoleMethodAccess_ReadGroupCount_FullMethodName = "/auth.RoleMethodAccess/ReadGroupCount"
	RoleMethodAccess_Search_FullMethodName         = "/auth.RoleMethodAccess/Search"
	RoleMethodAccess_Update_FullMethodName         = "/auth.RoleMethodAccess/Update"
	RoleMethodAccess_UpdateById_FullMethodName     = "/auth.RoleMethodAccess/UpdateById"
)

// RoleMethodAccessClient is the client API for RoleMethodAccess service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: RoleMethodAccess
type RoleMethodAccessClient interface {
	Browse(ctx context.Context, in *RoleMethodAccess_Browse_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *RoleMethodAccess_BrowseMany_Req, opts ...grpc.CallOption) (*RoleMethodAccess_BrowseMany_Resp, error)
	Count(ctx context.Context, in *RoleMethodAccess_Count_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Count_Resp, error)
	Create(ctx context.Context, in *RoleMethodAccess_Create_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Create_Resp, error)
	CreateMany(ctx context.Context, in *RoleMethodAccess_CreateMany_Req, opts ...grpc.CallOption) (*RoleMethodAccess_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *RoleMethodAccess_DefaultGet_Req, opts ...grpc.CallOption) (*RoleMethodAccess_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *RoleMethodAccess_Delete_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Delete_Resp, error)
	DeleteById(ctx context.Context, in *RoleMethodAccess_DeleteById_Req, opts ...grpc.CallOption) (*RoleMethodAccess_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *RoleMethodAccess_Onchange_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *RoleMethodAccess_ReadGroup_Req, opts ...grpc.CallOption) (*RoleMethodAccess_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *RoleMethodAccess_ReadGroupCount_Req, opts ...grpc.CallOption) (*RoleMethodAccess_ReadGroupCount_Resp, error)
	Search(ctx context.Context, in *RoleMethodAccess_Search_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Search_Resp, error)
	Update(ctx context.Context, in *RoleMethodAccess_Update_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Update_Resp, error)
	UpdateById(ctx context.Context, in *RoleMethodAccess_UpdateById_Req, opts ...grpc.CallOption) (*RoleMethodAccess_UpdateById_Resp, error)
}

type roleMethodAccessClient struct {
	cc grpc.ClientConnInterface
}

func NewRoleMethodAccessClient(cc grpc.ClientConnInterface) RoleMethodAccessClient {
	return &roleMethodAccessClient{cc}
}

func (c *roleMethodAccessClient) Browse(ctx context.Context, in *RoleMethodAccess_Browse_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_Browse_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) BrowseMany(ctx context.Context, in *RoleMethodAccess_BrowseMany_Req, opts ...grpc.CallOption) (*RoleMethodAccess_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) Count(ctx context.Context, in *RoleMethodAccess_Count_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_Count_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) Create(ctx context.Context, in *RoleMethodAccess_Create_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_Create_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) CreateMany(ctx context.Context, in *RoleMethodAccess_CreateMany_Req, opts ...grpc.CallOption) (*RoleMethodAccess_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_CreateMany_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) DefaultGet(ctx context.Context, in *RoleMethodAccess_DefaultGet_Req, opts ...grpc.CallOption) (*RoleMethodAccess_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) Delete(ctx context.Context, in *RoleMethodAccess_Delete_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_Delete_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) DeleteById(ctx context.Context, in *RoleMethodAccess_DeleteById_Req, opts ...grpc.CallOption) (*RoleMethodAccess_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_DeleteById_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) Onchange(ctx context.Context, in *RoleMethodAccess_Onchange_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_Onchange_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) ReadGroup(ctx context.Context, in *RoleMethodAccess_ReadGroup_Req, opts ...grpc.CallOption) (*RoleMethodAccess_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) ReadGroupCount(ctx context.Context, in *RoleMethodAccess_ReadGroupCount_Req, opts ...grpc.CallOption) (*RoleMethodAccess_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) Search(ctx context.Context, in *RoleMethodAccess_Search_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_Search_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) Update(ctx context.Context, in *RoleMethodAccess_Update_Req, opts ...grpc.CallOption) (*RoleMethodAccess_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_Update_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleMethodAccessClient) UpdateById(ctx context.Context, in *RoleMethodAccess_UpdateById_Req, opts ...grpc.CallOption) (*RoleMethodAccess_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleMethodAccess_UpdateById_Resp)
	err := c.cc.Invoke(ctx, RoleMethodAccess_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RoleMethodAccessServer is the server API for RoleMethodAccess service.
// All implementations must embed UnimplementedRoleMethodAccessServer
// for forward compatibility.
//
// Model: RoleMethodAccess
type RoleMethodAccessServer interface {
	Browse(context.Context, *RoleMethodAccess_Browse_Req) (*RoleMethodAccess_Browse_Resp, error)
	BrowseMany(context.Context, *RoleMethodAccess_BrowseMany_Req) (*RoleMethodAccess_BrowseMany_Resp, error)
	Count(context.Context, *RoleMethodAccess_Count_Req) (*RoleMethodAccess_Count_Resp, error)
	Create(context.Context, *RoleMethodAccess_Create_Req) (*RoleMethodAccess_Create_Resp, error)
	CreateMany(context.Context, *RoleMethodAccess_CreateMany_Req) (*RoleMethodAccess_CreateMany_Resp, error)
	DefaultGet(context.Context, *RoleMethodAccess_DefaultGet_Req) (*RoleMethodAccess_DefaultGet_Resp, error)
	Delete(context.Context, *RoleMethodAccess_Delete_Req) (*RoleMethodAccess_Delete_Resp, error)
	DeleteById(context.Context, *RoleMethodAccess_DeleteById_Req) (*RoleMethodAccess_DeleteById_Resp, error)
	Onchange(context.Context, *RoleMethodAccess_Onchange_Req) (*RoleMethodAccess_Onchange_Resp, error)
	ReadGroup(context.Context, *RoleMethodAccess_ReadGroup_Req) (*RoleMethodAccess_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *RoleMethodAccess_ReadGroupCount_Req) (*RoleMethodAccess_ReadGroupCount_Resp, error)
	Search(context.Context, *RoleMethodAccess_Search_Req) (*RoleMethodAccess_Search_Resp, error)
	Update(context.Context, *RoleMethodAccess_Update_Req) (*RoleMethodAccess_Update_Resp, error)
	UpdateById(context.Context, *RoleMethodAccess_UpdateById_Req) (*RoleMethodAccess_UpdateById_Resp, error)
	mustEmbedUnimplementedRoleMethodAccessServer()
}

// UnimplementedRoleMethodAccessServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedRoleMethodAccessServer struct{}

func (UnimplementedRoleMethodAccessServer) Browse(context.Context, *RoleMethodAccess_Browse_Req) (*RoleMethodAccess_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedRoleMethodAccessServer) BrowseMany(context.Context, *RoleMethodAccess_BrowseMany_Req) (*RoleMethodAccess_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedRoleMethodAccessServer) Count(context.Context, *RoleMethodAccess_Count_Req) (*RoleMethodAccess_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedRoleMethodAccessServer) Create(context.Context, *RoleMethodAccess_Create_Req) (*RoleMethodAccess_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedRoleMethodAccessServer) CreateMany(context.Context, *RoleMethodAccess_CreateMany_Req) (*RoleMethodAccess_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedRoleMethodAccessServer) DefaultGet(context.Context, *RoleMethodAccess_DefaultGet_Req) (*RoleMethodAccess_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedRoleMethodAccessServer) Delete(context.Context, *RoleMethodAccess_Delete_Req) (*RoleMethodAccess_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedRoleMethodAccessServer) DeleteById(context.Context, *RoleMethodAccess_DeleteById_Req) (*RoleMethodAccess_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedRoleMethodAccessServer) Onchange(context.Context, *RoleMethodAccess_Onchange_Req) (*RoleMethodAccess_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedRoleMethodAccessServer) ReadGroup(context.Context, *RoleMethodAccess_ReadGroup_Req) (*RoleMethodAccess_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedRoleMethodAccessServer) ReadGroupCount(context.Context, *RoleMethodAccess_ReadGroupCount_Req) (*RoleMethodAccess_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedRoleMethodAccessServer) Search(context.Context, *RoleMethodAccess_Search_Req) (*RoleMethodAccess_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedRoleMethodAccessServer) Update(context.Context, *RoleMethodAccess_Update_Req) (*RoleMethodAccess_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedRoleMethodAccessServer) UpdateById(context.Context, *RoleMethodAccess_UpdateById_Req) (*RoleMethodAccess_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedRoleMethodAccessServer) mustEmbedUnimplementedRoleMethodAccessServer() {}
func (UnimplementedRoleMethodAccessServer) testEmbeddedByValue()                          {}

// UnsafeRoleMethodAccessServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RoleMethodAccessServer will
// result in compilation errors.
type UnsafeRoleMethodAccessServer interface {
	mustEmbedUnimplementedRoleMethodAccessServer()
}

func RegisterRoleMethodAccessServer(s grpc.ServiceRegistrar, srv RoleMethodAccessServer) {
	// If the following call panics, it indicates UnimplementedRoleMethodAccessServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&RoleMethodAccess_ServiceDesc, srv)
}

func _RoleMethodAccess_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).Browse(ctx, req.(*RoleMethodAccess_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).BrowseMany(ctx, req.(*RoleMethodAccess_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).Count(ctx, req.(*RoleMethodAccess_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).Create(ctx, req.(*RoleMethodAccess_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).CreateMany(ctx, req.(*RoleMethodAccess_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).DefaultGet(ctx, req.(*RoleMethodAccess_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).Delete(ctx, req.(*RoleMethodAccess_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).DeleteById(ctx, req.(*RoleMethodAccess_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).Onchange(ctx, req.(*RoleMethodAccess_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).ReadGroup(ctx, req.(*RoleMethodAccess_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).ReadGroupCount(ctx, req.(*RoleMethodAccess_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).Search(ctx, req.(*RoleMethodAccess_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).Update(ctx, req.(*RoleMethodAccess_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleMethodAccess_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleMethodAccess_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleMethodAccessServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleMethodAccess_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleMethodAccessServer).UpdateById(ctx, req.(*RoleMethodAccess_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// RoleMethodAccess_ServiceDesc is the grpc.ServiceDesc for RoleMethodAccess service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RoleMethodAccess_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.RoleMethodAccess",
	HandlerType: (*RoleMethodAccessServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _RoleMethodAccess_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _RoleMethodAccess_BrowseMany_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _RoleMethodAccess_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _RoleMethodAccess_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _RoleMethodAccess_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _RoleMethodAccess_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _RoleMethodAccess_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _RoleMethodAccess_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _RoleMethodAccess_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _RoleMethodAccess_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _RoleMethodAccess_ReadGroupCount_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _RoleMethodAccess_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _RoleMethodAccess_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _RoleMethodAccess_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	RoleRecordRule_Browse_FullMethodName         = "/auth.RoleRecordRule/Browse"
	RoleRecordRule_BrowseMany_FullMethodName     = "/auth.RoleRecordRule/BrowseMany"
	RoleRecordRule_Count_FullMethodName          = "/auth.RoleRecordRule/Count"
	RoleRecordRule_Create_FullMethodName         = "/auth.RoleRecordRule/Create"
	RoleRecordRule_CreateMany_FullMethodName     = "/auth.RoleRecordRule/CreateMany"
	RoleRecordRule_DefaultGet_FullMethodName     = "/auth.RoleRecordRule/DefaultGet"
	RoleRecordRule_Delete_FullMethodName         = "/auth.RoleRecordRule/Delete"
	RoleRecordRule_DeleteById_FullMethodName     = "/auth.RoleRecordRule/DeleteById"
	RoleRecordRule_Onchange_FullMethodName       = "/auth.RoleRecordRule/Onchange"
	RoleRecordRule_ReadGroup_FullMethodName      = "/auth.RoleRecordRule/ReadGroup"
	RoleRecordRule_ReadGroupCount_FullMethodName = "/auth.RoleRecordRule/ReadGroupCount"
	RoleRecordRule_Search_FullMethodName         = "/auth.RoleRecordRule/Search"
	RoleRecordRule_Update_FullMethodName         = "/auth.RoleRecordRule/Update"
	RoleRecordRule_UpdateById_FullMethodName     = "/auth.RoleRecordRule/UpdateById"
)

// RoleRecordRuleClient is the client API for RoleRecordRule service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: RoleRecordRule
type RoleRecordRuleClient interface {
	Browse(ctx context.Context, in *RoleRecordRule_Browse_Req, opts ...grpc.CallOption) (*RoleRecordRule_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *RoleRecordRule_BrowseMany_Req, opts ...grpc.CallOption) (*RoleRecordRule_BrowseMany_Resp, error)
	Count(ctx context.Context, in *RoleRecordRule_Count_Req, opts ...grpc.CallOption) (*RoleRecordRule_Count_Resp, error)
	Create(ctx context.Context, in *RoleRecordRule_Create_Req, opts ...grpc.CallOption) (*RoleRecordRule_Create_Resp, error)
	CreateMany(ctx context.Context, in *RoleRecordRule_CreateMany_Req, opts ...grpc.CallOption) (*RoleRecordRule_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *RoleRecordRule_DefaultGet_Req, opts ...grpc.CallOption) (*RoleRecordRule_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *RoleRecordRule_Delete_Req, opts ...grpc.CallOption) (*RoleRecordRule_Delete_Resp, error)
	DeleteById(ctx context.Context, in *RoleRecordRule_DeleteById_Req, opts ...grpc.CallOption) (*RoleRecordRule_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *RoleRecordRule_Onchange_Req, opts ...grpc.CallOption) (*RoleRecordRule_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *RoleRecordRule_ReadGroup_Req, opts ...grpc.CallOption) (*RoleRecordRule_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *RoleRecordRule_ReadGroupCount_Req, opts ...grpc.CallOption) (*RoleRecordRule_ReadGroupCount_Resp, error)
	Search(ctx context.Context, in *RoleRecordRule_Search_Req, opts ...grpc.CallOption) (*RoleRecordRule_Search_Resp, error)
	Update(ctx context.Context, in *RoleRecordRule_Update_Req, opts ...grpc.CallOption) (*RoleRecordRule_Update_Resp, error)
	UpdateById(ctx context.Context, in *RoleRecordRule_UpdateById_Req, opts ...grpc.CallOption) (*RoleRecordRule_UpdateById_Resp, error)
}

type roleRecordRuleClient struct {
	cc grpc.ClientConnInterface
}

func NewRoleRecordRuleClient(cc grpc.ClientConnInterface) RoleRecordRuleClient {
	return &roleRecordRuleClient{cc}
}

func (c *roleRecordRuleClient) Browse(ctx context.Context, in *RoleRecordRule_Browse_Req, opts ...grpc.CallOption) (*RoleRecordRule_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_Browse_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) BrowseMany(ctx context.Context, in *RoleRecordRule_BrowseMany_Req, opts ...grpc.CallOption) (*RoleRecordRule_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) Count(ctx context.Context, in *RoleRecordRule_Count_Req, opts ...grpc.CallOption) (*RoleRecordRule_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_Count_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) Create(ctx context.Context, in *RoleRecordRule_Create_Req, opts ...grpc.CallOption) (*RoleRecordRule_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_Create_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) CreateMany(ctx context.Context, in *RoleRecordRule_CreateMany_Req, opts ...grpc.CallOption) (*RoleRecordRule_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_CreateMany_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) DefaultGet(ctx context.Context, in *RoleRecordRule_DefaultGet_Req, opts ...grpc.CallOption) (*RoleRecordRule_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) Delete(ctx context.Context, in *RoleRecordRule_Delete_Req, opts ...grpc.CallOption) (*RoleRecordRule_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_Delete_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) DeleteById(ctx context.Context, in *RoleRecordRule_DeleteById_Req, opts ...grpc.CallOption) (*RoleRecordRule_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_DeleteById_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) Onchange(ctx context.Context, in *RoleRecordRule_Onchange_Req, opts ...grpc.CallOption) (*RoleRecordRule_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_Onchange_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) ReadGroup(ctx context.Context, in *RoleRecordRule_ReadGroup_Req, opts ...grpc.CallOption) (*RoleRecordRule_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) ReadGroupCount(ctx context.Context, in *RoleRecordRule_ReadGroupCount_Req, opts ...grpc.CallOption) (*RoleRecordRule_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) Search(ctx context.Context, in *RoleRecordRule_Search_Req, opts ...grpc.CallOption) (*RoleRecordRule_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_Search_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) Update(ctx context.Context, in *RoleRecordRule_Update_Req, opts ...grpc.CallOption) (*RoleRecordRule_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_Update_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *roleRecordRuleClient) UpdateById(ctx context.Context, in *RoleRecordRule_UpdateById_Req, opts ...grpc.CallOption) (*RoleRecordRule_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RoleRecordRule_UpdateById_Resp)
	err := c.cc.Invoke(ctx, RoleRecordRule_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RoleRecordRuleServer is the server API for RoleRecordRule service.
// All implementations must embed UnimplementedRoleRecordRuleServer
// for forward compatibility.
//
// Model: RoleRecordRule
type RoleRecordRuleServer interface {
	Browse(context.Context, *RoleRecordRule_Browse_Req) (*RoleRecordRule_Browse_Resp, error)
	BrowseMany(context.Context, *RoleRecordRule_BrowseMany_Req) (*RoleRecordRule_BrowseMany_Resp, error)
	Count(context.Context, *RoleRecordRule_Count_Req) (*RoleRecordRule_Count_Resp, error)
	Create(context.Context, *RoleRecordRule_Create_Req) (*RoleRecordRule_Create_Resp, error)
	CreateMany(context.Context, *RoleRecordRule_CreateMany_Req) (*RoleRecordRule_CreateMany_Resp, error)
	DefaultGet(context.Context, *RoleRecordRule_DefaultGet_Req) (*RoleRecordRule_DefaultGet_Resp, error)
	Delete(context.Context, *RoleRecordRule_Delete_Req) (*RoleRecordRule_Delete_Resp, error)
	DeleteById(context.Context, *RoleRecordRule_DeleteById_Req) (*RoleRecordRule_DeleteById_Resp, error)
	Onchange(context.Context, *RoleRecordRule_Onchange_Req) (*RoleRecordRule_Onchange_Resp, error)
	ReadGroup(context.Context, *RoleRecordRule_ReadGroup_Req) (*RoleRecordRule_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *RoleRecordRule_ReadGroupCount_Req) (*RoleRecordRule_ReadGroupCount_Resp, error)
	Search(context.Context, *RoleRecordRule_Search_Req) (*RoleRecordRule_Search_Resp, error)
	Update(context.Context, *RoleRecordRule_Update_Req) (*RoleRecordRule_Update_Resp, error)
	UpdateById(context.Context, *RoleRecordRule_UpdateById_Req) (*RoleRecordRule_UpdateById_Resp, error)
	mustEmbedUnimplementedRoleRecordRuleServer()
}

// UnimplementedRoleRecordRuleServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedRoleRecordRuleServer struct{}

func (UnimplementedRoleRecordRuleServer) Browse(context.Context, *RoleRecordRule_Browse_Req) (*RoleRecordRule_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedRoleRecordRuleServer) BrowseMany(context.Context, *RoleRecordRule_BrowseMany_Req) (*RoleRecordRule_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedRoleRecordRuleServer) Count(context.Context, *RoleRecordRule_Count_Req) (*RoleRecordRule_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedRoleRecordRuleServer) Create(context.Context, *RoleRecordRule_Create_Req) (*RoleRecordRule_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedRoleRecordRuleServer) CreateMany(context.Context, *RoleRecordRule_CreateMany_Req) (*RoleRecordRule_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedRoleRecordRuleServer) DefaultGet(context.Context, *RoleRecordRule_DefaultGet_Req) (*RoleRecordRule_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedRoleRecordRuleServer) Delete(context.Context, *RoleRecordRule_Delete_Req) (*RoleRecordRule_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedRoleRecordRuleServer) DeleteById(context.Context, *RoleRecordRule_DeleteById_Req) (*RoleRecordRule_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedRoleRecordRuleServer) Onchange(context.Context, *RoleRecordRule_Onchange_Req) (*RoleRecordRule_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedRoleRecordRuleServer) ReadGroup(context.Context, *RoleRecordRule_ReadGroup_Req) (*RoleRecordRule_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedRoleRecordRuleServer) ReadGroupCount(context.Context, *RoleRecordRule_ReadGroupCount_Req) (*RoleRecordRule_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedRoleRecordRuleServer) Search(context.Context, *RoleRecordRule_Search_Req) (*RoleRecordRule_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedRoleRecordRuleServer) Update(context.Context, *RoleRecordRule_Update_Req) (*RoleRecordRule_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedRoleRecordRuleServer) UpdateById(context.Context, *RoleRecordRule_UpdateById_Req) (*RoleRecordRule_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedRoleRecordRuleServer) mustEmbedUnimplementedRoleRecordRuleServer() {}
func (UnimplementedRoleRecordRuleServer) testEmbeddedByValue()                        {}

// UnsafeRoleRecordRuleServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RoleRecordRuleServer will
// result in compilation errors.
type UnsafeRoleRecordRuleServer interface {
	mustEmbedUnimplementedRoleRecordRuleServer()
}

func RegisterRoleRecordRuleServer(s grpc.ServiceRegistrar, srv RoleRecordRuleServer) {
	// If the following call panics, it indicates UnimplementedRoleRecordRuleServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&RoleRecordRule_ServiceDesc, srv)
}

func _RoleRecordRule_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).Browse(ctx, req.(*RoleRecordRule_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).BrowseMany(ctx, req.(*RoleRecordRule_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).Count(ctx, req.(*RoleRecordRule_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).Create(ctx, req.(*RoleRecordRule_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).CreateMany(ctx, req.(*RoleRecordRule_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).DefaultGet(ctx, req.(*RoleRecordRule_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).Delete(ctx, req.(*RoleRecordRule_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).DeleteById(ctx, req.(*RoleRecordRule_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).Onchange(ctx, req.(*RoleRecordRule_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).ReadGroup(ctx, req.(*RoleRecordRule_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).ReadGroupCount(ctx, req.(*RoleRecordRule_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).Search(ctx, req.(*RoleRecordRule_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).Update(ctx, req.(*RoleRecordRule_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _RoleRecordRule_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RoleRecordRule_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoleRecordRuleServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoleRecordRule_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoleRecordRuleServer).UpdateById(ctx, req.(*RoleRecordRule_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// RoleRecordRule_ServiceDesc is the grpc.ServiceDesc for RoleRecordRule service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RoleRecordRule_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.RoleRecordRule",
	HandlerType: (*RoleRecordRuleServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _RoleRecordRule_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _RoleRecordRule_BrowseMany_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _RoleRecordRule_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _RoleRecordRule_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _RoleRecordRule_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _RoleRecordRule_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _RoleRecordRule_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _RoleRecordRule_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _RoleRecordRule_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _RoleRecordRule_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _RoleRecordRule_ReadGroupCount_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _RoleRecordRule_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _RoleRecordRule_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _RoleRecordRule_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	Session_Browse_FullMethodName                   = "/auth.Session/Browse"
	Session_BrowseMany_FullMethodName               = "/auth.Session/BrowseMany"
	Session_CleanExpiredSessions_FullMethodName     = "/auth.Session/CleanExpiredSessions"
	Session_Count_FullMethodName                    = "/auth.Session/Count"
	Session_Create_FullMethodName                   = "/auth.Session/Create"
	Session_CreateMany_FullMethodName               = "/auth.Session/CreateMany"
	Session_DefaultGet_FullMethodName               = "/auth.Session/DefaultGet"
	Session_Delete_FullMethodName                   = "/auth.Session/Delete"
	Session_DeleteById_FullMethodName               = "/auth.Session/DeleteById"
	Session_GetActiveSessionsForUser_FullMethodName = "/auth.Session/GetActiveSessionsForUser"
	Session_Onchange_FullMethodName                 = "/auth.Session/Onchange"
	Session_ReadGroup_FullMethodName                = "/auth.Session/ReadGroup"
	Session_ReadGroupCount_FullMethodName           = "/auth.Session/ReadGroupCount"
	Session_RevokeAllForUser_FullMethodName         = "/auth.Session/RevokeAllForUser"
	Session_RevokeSession_FullMethodName            = "/auth.Session/RevokeSession"
	Session_Search_FullMethodName                   = "/auth.Session/Search"
	Session_Update_FullMethodName                   = "/auth.Session/Update"
	Session_UpdateById_FullMethodName               = "/auth.Session/UpdateById"
	Session_ValidateToken_FullMethodName            = "/auth.Session/ValidateToken"
)

// SessionClient is the client API for Session service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: Session
type SessionClient interface {
	Browse(ctx context.Context, in *Session_Browse_Req, opts ...grpc.CallOption) (*Session_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *Session_BrowseMany_Req, opts ...grpc.CallOption) (*Session_BrowseMany_Resp, error)
	CleanExpiredSessions(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Session_CleanExpiredSessions_Resp, error)
	Count(ctx context.Context, in *Session_Count_Req, opts ...grpc.CallOption) (*Session_Count_Resp, error)
	Create(ctx context.Context, in *Session_Create_Req, opts ...grpc.CallOption) (*Session_Create_Resp, error)
	CreateMany(ctx context.Context, in *Session_CreateMany_Req, opts ...grpc.CallOption) (*Session_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *Session_DefaultGet_Req, opts ...grpc.CallOption) (*Session_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *Session_Delete_Req, opts ...grpc.CallOption) (*Session_Delete_Resp, error)
	DeleteById(ctx context.Context, in *Session_DeleteById_Req, opts ...grpc.CallOption) (*Session_DeleteById_Resp, error)
	GetActiveSessionsForUser(ctx context.Context, in *Session_GetActiveSessionsForUser_Req, opts ...grpc.CallOption) (*Session_GetActiveSessionsForUser_Resp, error)
	Onchange(ctx context.Context, in *Session_Onchange_Req, opts ...grpc.CallOption) (*Session_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *Session_ReadGroup_Req, opts ...grpc.CallOption) (*Session_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *Session_ReadGroupCount_Req, opts ...grpc.CallOption) (*Session_ReadGroupCount_Resp, error)
	RevokeAllForUser(ctx context.Context, in *Session_RevokeAllForUser_Req, opts ...grpc.CallOption) (*Session_RevokeAllForUser_Resp, error)
	RevokeSession(ctx context.Context, in *Session_RevokeSession_Req, opts ...grpc.CallOption) (*Session_RevokeSession_Resp, error)
	Search(ctx context.Context, in *Session_Search_Req, opts ...grpc.CallOption) (*Session_Search_Resp, error)
	Update(ctx context.Context, in *Session_Update_Req, opts ...grpc.CallOption) (*Session_Update_Resp, error)
	UpdateById(ctx context.Context, in *Session_UpdateById_Req, opts ...grpc.CallOption) (*Session_UpdateById_Resp, error)
	ValidateToken(ctx context.Context, in *Session_ValidateToken_Req, opts ...grpc.CallOption) (*Session_ValidateToken_Resp, error)
}

type sessionClient struct {
	cc grpc.ClientConnInterface
}

func NewSessionClient(cc grpc.ClientConnInterface) SessionClient {
	return &sessionClient{cc}
}

func (c *sessionClient) Browse(ctx context.Context, in *Session_Browse_Req, opts ...grpc.CallOption) (*Session_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_Browse_Resp)
	err := c.cc.Invoke(ctx, Session_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) BrowseMany(ctx context.Context, in *Session_BrowseMany_Req, opts ...grpc.CallOption) (*Session_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, Session_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) CleanExpiredSessions(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Session_CleanExpiredSessions_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_CleanExpiredSessions_Resp)
	err := c.cc.Invoke(ctx, Session_CleanExpiredSessions_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) Count(ctx context.Context, in *Session_Count_Req, opts ...grpc.CallOption) (*Session_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_Count_Resp)
	err := c.cc.Invoke(ctx, Session_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) Create(ctx context.Context, in *Session_Create_Req, opts ...grpc.CallOption) (*Session_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_Create_Resp)
	err := c.cc.Invoke(ctx, Session_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) CreateMany(ctx context.Context, in *Session_CreateMany_Req, opts ...grpc.CallOption) (*Session_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_CreateMany_Resp)
	err := c.cc.Invoke(ctx, Session_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) DefaultGet(ctx context.Context, in *Session_DefaultGet_Req, opts ...grpc.CallOption) (*Session_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, Session_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) Delete(ctx context.Context, in *Session_Delete_Req, opts ...grpc.CallOption) (*Session_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_Delete_Resp)
	err := c.cc.Invoke(ctx, Session_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) DeleteById(ctx context.Context, in *Session_DeleteById_Req, opts ...grpc.CallOption) (*Session_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_DeleteById_Resp)
	err := c.cc.Invoke(ctx, Session_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) GetActiveSessionsForUser(ctx context.Context, in *Session_GetActiveSessionsForUser_Req, opts ...grpc.CallOption) (*Session_GetActiveSessionsForUser_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_GetActiveSessionsForUser_Resp)
	err := c.cc.Invoke(ctx, Session_GetActiveSessionsForUser_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) Onchange(ctx context.Context, in *Session_Onchange_Req, opts ...grpc.CallOption) (*Session_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_Onchange_Resp)
	err := c.cc.Invoke(ctx, Session_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) ReadGroup(ctx context.Context, in *Session_ReadGroup_Req, opts ...grpc.CallOption) (*Session_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, Session_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) ReadGroupCount(ctx context.Context, in *Session_ReadGroupCount_Req, opts ...grpc.CallOption) (*Session_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, Session_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) RevokeAllForUser(ctx context.Context, in *Session_RevokeAllForUser_Req, opts ...grpc.CallOption) (*Session_RevokeAllForUser_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_RevokeAllForUser_Resp)
	err := c.cc.Invoke(ctx, Session_RevokeAllForUser_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) RevokeSession(ctx context.Context, in *Session_RevokeSession_Req, opts ...grpc.CallOption) (*Session_RevokeSession_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_RevokeSession_Resp)
	err := c.cc.Invoke(ctx, Session_RevokeSession_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) Search(ctx context.Context, in *Session_Search_Req, opts ...grpc.CallOption) (*Session_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_Search_Resp)
	err := c.cc.Invoke(ctx, Session_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) Update(ctx context.Context, in *Session_Update_Req, opts ...grpc.CallOption) (*Session_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_Update_Resp)
	err := c.cc.Invoke(ctx, Session_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) UpdateById(ctx context.Context, in *Session_UpdateById_Req, opts ...grpc.CallOption) (*Session_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_UpdateById_Resp)
	err := c.cc.Invoke(ctx, Session_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *sessionClient) ValidateToken(ctx context.Context, in *Session_ValidateToken_Req, opts ...grpc.CallOption) (*Session_ValidateToken_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Session_ValidateToken_Resp)
	err := c.cc.Invoke(ctx, Session_ValidateToken_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SessionServer is the server API for Session service.
// All implementations must embed UnimplementedSessionServer
// for forward compatibility.
//
// Model: Session
type SessionServer interface {
	Browse(context.Context, *Session_Browse_Req) (*Session_Browse_Resp, error)
	BrowseMany(context.Context, *Session_BrowseMany_Req) (*Session_BrowseMany_Resp, error)
	CleanExpiredSessions(context.Context, *emptypb.Empty) (*Session_CleanExpiredSessions_Resp, error)
	Count(context.Context, *Session_Count_Req) (*Session_Count_Resp, error)
	Create(context.Context, *Session_Create_Req) (*Session_Create_Resp, error)
	CreateMany(context.Context, *Session_CreateMany_Req) (*Session_CreateMany_Resp, error)
	DefaultGet(context.Context, *Session_DefaultGet_Req) (*Session_DefaultGet_Resp, error)
	Delete(context.Context, *Session_Delete_Req) (*Session_Delete_Resp, error)
	DeleteById(context.Context, *Session_DeleteById_Req) (*Session_DeleteById_Resp, error)
	GetActiveSessionsForUser(context.Context, *Session_GetActiveSessionsForUser_Req) (*Session_GetActiveSessionsForUser_Resp, error)
	Onchange(context.Context, *Session_Onchange_Req) (*Session_Onchange_Resp, error)
	ReadGroup(context.Context, *Session_ReadGroup_Req) (*Session_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *Session_ReadGroupCount_Req) (*Session_ReadGroupCount_Resp, error)
	RevokeAllForUser(context.Context, *Session_RevokeAllForUser_Req) (*Session_RevokeAllForUser_Resp, error)
	RevokeSession(context.Context, *Session_RevokeSession_Req) (*Session_RevokeSession_Resp, error)
	Search(context.Context, *Session_Search_Req) (*Session_Search_Resp, error)
	Update(context.Context, *Session_Update_Req) (*Session_Update_Resp, error)
	UpdateById(context.Context, *Session_UpdateById_Req) (*Session_UpdateById_Resp, error)
	ValidateToken(context.Context, *Session_ValidateToken_Req) (*Session_ValidateToken_Resp, error)
	mustEmbedUnimplementedSessionServer()
}

// UnimplementedSessionServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedSessionServer struct{}

func (UnimplementedSessionServer) Browse(context.Context, *Session_Browse_Req) (*Session_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedSessionServer) BrowseMany(context.Context, *Session_BrowseMany_Req) (*Session_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedSessionServer) CleanExpiredSessions(context.Context, *emptypb.Empty) (*Session_CleanExpiredSessions_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CleanExpiredSessions not implemented")
}
func (UnimplementedSessionServer) Count(context.Context, *Session_Count_Req) (*Session_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedSessionServer) Create(context.Context, *Session_Create_Req) (*Session_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedSessionServer) CreateMany(context.Context, *Session_CreateMany_Req) (*Session_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedSessionServer) DefaultGet(context.Context, *Session_DefaultGet_Req) (*Session_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedSessionServer) Delete(context.Context, *Session_Delete_Req) (*Session_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedSessionServer) DeleteById(context.Context, *Session_DeleteById_Req) (*Session_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedSessionServer) GetActiveSessionsForUser(context.Context, *Session_GetActiveSessionsForUser_Req) (*Session_GetActiveSessionsForUser_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method GetActiveSessionsForUser not implemented")
}
func (UnimplementedSessionServer) Onchange(context.Context, *Session_Onchange_Req) (*Session_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedSessionServer) ReadGroup(context.Context, *Session_ReadGroup_Req) (*Session_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedSessionServer) ReadGroupCount(context.Context, *Session_ReadGroupCount_Req) (*Session_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedSessionServer) RevokeAllForUser(context.Context, *Session_RevokeAllForUser_Req) (*Session_RevokeAllForUser_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method RevokeAllForUser not implemented")
}
func (UnimplementedSessionServer) RevokeSession(context.Context, *Session_RevokeSession_Req) (*Session_RevokeSession_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method RevokeSession not implemented")
}
func (UnimplementedSessionServer) Search(context.Context, *Session_Search_Req) (*Session_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedSessionServer) Update(context.Context, *Session_Update_Req) (*Session_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedSessionServer) UpdateById(context.Context, *Session_UpdateById_Req) (*Session_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedSessionServer) ValidateToken(context.Context, *Session_ValidateToken_Req) (*Session_ValidateToken_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ValidateToken not implemented")
}
func (UnimplementedSessionServer) mustEmbedUnimplementedSessionServer() {}
func (UnimplementedSessionServer) testEmbeddedByValue()                 {}

// UnsafeSessionServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SessionServer will
// result in compilation errors.
type UnsafeSessionServer interface {
	mustEmbedUnimplementedSessionServer()
}

func RegisterSessionServer(s grpc.ServiceRegistrar, srv SessionServer) {
	// If the following call panics, it indicates UnimplementedSessionServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Session_ServiceDesc, srv)
}

func _Session_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).Browse(ctx, req.(*Session_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).BrowseMany(ctx, req.(*Session_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_CleanExpiredSessions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).CleanExpiredSessions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_CleanExpiredSessions_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).CleanExpiredSessions(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).Count(ctx, req.(*Session_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).Create(ctx, req.(*Session_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).CreateMany(ctx, req.(*Session_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).DefaultGet(ctx, req.(*Session_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).Delete(ctx, req.(*Session_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).DeleteById(ctx, req.(*Session_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_GetActiveSessionsForUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_GetActiveSessionsForUser_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).GetActiveSessionsForUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_GetActiveSessionsForUser_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).GetActiveSessionsForUser(ctx, req.(*Session_GetActiveSessionsForUser_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).Onchange(ctx, req.(*Session_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).ReadGroup(ctx, req.(*Session_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).ReadGroupCount(ctx, req.(*Session_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_RevokeAllForUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_RevokeAllForUser_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).RevokeAllForUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_RevokeAllForUser_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).RevokeAllForUser(ctx, req.(*Session_RevokeAllForUser_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_RevokeSession_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_RevokeSession_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).RevokeSession(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_RevokeSession_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).RevokeSession(ctx, req.(*Session_RevokeSession_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).Search(ctx, req.(*Session_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).Update(ctx, req.(*Session_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).UpdateById(ctx, req.(*Session_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Session_ValidateToken_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Session_ValidateToken_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SessionServer).ValidateToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Session_ValidateToken_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SessionServer).ValidateToken(ctx, req.(*Session_ValidateToken_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// Session_ServiceDesc is the grpc.ServiceDesc for Session service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Session_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.Session",
	HandlerType: (*SessionServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _Session_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _Session_BrowseMany_Handler,
		},
		{
			MethodName: "CleanExpiredSessions",
			Handler:    _Session_CleanExpiredSessions_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _Session_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _Session_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _Session_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _Session_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _Session_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _Session_DeleteById_Handler,
		},
		{
			MethodName: "GetActiveSessionsForUser",
			Handler:    _Session_GetActiveSessionsForUser_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _Session_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _Session_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _Session_ReadGroupCount_Handler,
		},
		{
			MethodName: "RevokeAllForUser",
			Handler:    _Session_RevokeAllForUser_Handler,
		},
		{
			MethodName: "RevokeSession",
			Handler:    _Session_RevokeSession_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _Session_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _Session_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _Session_UpdateById_Handler,
		},
		{
			MethodName: "ValidateToken",
			Handler:    _Session_ValidateToken_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	Token_Browse_FullMethodName                 = "/auth.Token/Browse"
	Token_BrowseMany_FullMethodName             = "/auth.Token/BrowseMany"
	Token_CleanExpiredTokens_FullMethodName     = "/auth.Token/CleanExpiredTokens"
	Token_Count_FullMethodName                  = "/auth.Token/Count"
	Token_Create_FullMethodName                 = "/auth.Token/Create"
	Token_CreateMany_FullMethodName             = "/auth.Token/CreateMany"
	Token_CreateTokenPair_FullMethodName        = "/auth.Token/CreateTokenPair"
	Token_DefaultGet_FullMethodName             = "/auth.Token/DefaultGet"
	Token_Delete_FullMethodName                 = "/auth.Token/Delete"
	Token_DeleteById_FullMethodName             = "/auth.Token/DeleteById"
	Token_Onchange_FullMethodName               = "/auth.Token/Onchange"
	Token_ReadGroup_FullMethodName              = "/auth.Token/ReadGroup"
	Token_ReadGroupCount_FullMethodName         = "/auth.Token/ReadGroupCount"
	Token_RefreshTokens_FullMethodName          = "/auth.Token/RefreshTokens"
	Token_RevokeAllUserTokens_FullMethodName    = "/auth.Token/RevokeAllUserTokens"
	Token_RevokeToken_FullMethodName            = "/auth.Token/RevokeToken"
	Token_RevokeUserAccessTokens_FullMethodName = "/auth.Token/RevokeUserAccessTokens"
	Token_Search_FullMethodName                 = "/auth.Token/Search"
	Token_Update_FullMethodName                 = "/auth.Token/Update"
	Token_UpdateById_FullMethodName             = "/auth.Token/UpdateById"
	Token_ValidateToken_FullMethodName          = "/auth.Token/ValidateToken"
)

// TokenClient is the client API for Token service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: Token
type TokenClient interface {
	Browse(ctx context.Context, in *Token_Browse_Req, opts ...grpc.CallOption) (*Token_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *Token_BrowseMany_Req, opts ...grpc.CallOption) (*Token_BrowseMany_Resp, error)
	CleanExpiredTokens(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Token_CleanExpiredTokens_Resp, error)
	Count(ctx context.Context, in *Token_Count_Req, opts ...grpc.CallOption) (*Token_Count_Resp, error)
	Create(ctx context.Context, in *Token_Create_Req, opts ...grpc.CallOption) (*Token_Create_Resp, error)
	CreateMany(ctx context.Context, in *Token_CreateMany_Req, opts ...grpc.CallOption) (*Token_CreateMany_Resp, error)
	CreateTokenPair(ctx context.Context, in *Token_CreateTokenPair_Req, opts ...grpc.CallOption) (*Token_CreateTokenPair_Resp, error)
	DefaultGet(ctx context.Context, in *Token_DefaultGet_Req, opts ...grpc.CallOption) (*Token_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *Token_Delete_Req, opts ...grpc.CallOption) (*Token_Delete_Resp, error)
	DeleteById(ctx context.Context, in *Token_DeleteById_Req, opts ...grpc.CallOption) (*Token_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *Token_Onchange_Req, opts ...grpc.CallOption) (*Token_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *Token_ReadGroup_Req, opts ...grpc.CallOption) (*Token_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *Token_ReadGroupCount_Req, opts ...grpc.CallOption) (*Token_ReadGroupCount_Resp, error)
	RefreshTokens(ctx context.Context, in *Token_RefreshTokens_Req, opts ...grpc.CallOption) (*Token_RefreshTokens_Resp, error)
	RevokeAllUserTokens(ctx context.Context, in *Token_RevokeAllUserTokens_Req, opts ...grpc.CallOption) (*Token_RevokeAllUserTokens_Resp, error)
	RevokeToken(ctx context.Context, in *Token_RevokeToken_Req, opts ...grpc.CallOption) (*Token_RevokeToken_Resp, error)
	RevokeUserAccessTokens(ctx context.Context, in *Token_RevokeUserAccessTokens_Req, opts ...grpc.CallOption) (*Token_RevokeUserAccessTokens_Resp, error)
	Search(ctx context.Context, in *Token_Search_Req, opts ...grpc.CallOption) (*Token_Search_Resp, error)
	Update(ctx context.Context, in *Token_Update_Req, opts ...grpc.CallOption) (*Token_Update_Resp, error)
	UpdateById(ctx context.Context, in *Token_UpdateById_Req, opts ...grpc.CallOption) (*Token_UpdateById_Resp, error)
	ValidateToken(ctx context.Context, in *Token_ValidateToken_Req, opts ...grpc.CallOption) (*Token_ValidateToken_Resp, error)
}

type tokenClient struct {
	cc grpc.ClientConnInterface
}

func NewTokenClient(cc grpc.ClientConnInterface) TokenClient {
	return &tokenClient{cc}
}

func (c *tokenClient) Browse(ctx context.Context, in *Token_Browse_Req, opts ...grpc.CallOption) (*Token_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_Browse_Resp)
	err := c.cc.Invoke(ctx, Token_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) BrowseMany(ctx context.Context, in *Token_BrowseMany_Req, opts ...grpc.CallOption) (*Token_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, Token_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) CleanExpiredTokens(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*Token_CleanExpiredTokens_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_CleanExpiredTokens_Resp)
	err := c.cc.Invoke(ctx, Token_CleanExpiredTokens_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) Count(ctx context.Context, in *Token_Count_Req, opts ...grpc.CallOption) (*Token_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_Count_Resp)
	err := c.cc.Invoke(ctx, Token_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) Create(ctx context.Context, in *Token_Create_Req, opts ...grpc.CallOption) (*Token_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_Create_Resp)
	err := c.cc.Invoke(ctx, Token_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) CreateMany(ctx context.Context, in *Token_CreateMany_Req, opts ...grpc.CallOption) (*Token_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_CreateMany_Resp)
	err := c.cc.Invoke(ctx, Token_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) CreateTokenPair(ctx context.Context, in *Token_CreateTokenPair_Req, opts ...grpc.CallOption) (*Token_CreateTokenPair_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_CreateTokenPair_Resp)
	err := c.cc.Invoke(ctx, Token_CreateTokenPair_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) DefaultGet(ctx context.Context, in *Token_DefaultGet_Req, opts ...grpc.CallOption) (*Token_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, Token_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) Delete(ctx context.Context, in *Token_Delete_Req, opts ...grpc.CallOption) (*Token_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_Delete_Resp)
	err := c.cc.Invoke(ctx, Token_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) DeleteById(ctx context.Context, in *Token_DeleteById_Req, opts ...grpc.CallOption) (*Token_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_DeleteById_Resp)
	err := c.cc.Invoke(ctx, Token_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) Onchange(ctx context.Context, in *Token_Onchange_Req, opts ...grpc.CallOption) (*Token_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_Onchange_Resp)
	err := c.cc.Invoke(ctx, Token_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) ReadGroup(ctx context.Context, in *Token_ReadGroup_Req, opts ...grpc.CallOption) (*Token_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, Token_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) ReadGroupCount(ctx context.Context, in *Token_ReadGroupCount_Req, opts ...grpc.CallOption) (*Token_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, Token_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) RefreshTokens(ctx context.Context, in *Token_RefreshTokens_Req, opts ...grpc.CallOption) (*Token_RefreshTokens_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_RefreshTokens_Resp)
	err := c.cc.Invoke(ctx, Token_RefreshTokens_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) RevokeAllUserTokens(ctx context.Context, in *Token_RevokeAllUserTokens_Req, opts ...grpc.CallOption) (*Token_RevokeAllUserTokens_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_RevokeAllUserTokens_Resp)
	err := c.cc.Invoke(ctx, Token_RevokeAllUserTokens_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) RevokeToken(ctx context.Context, in *Token_RevokeToken_Req, opts ...grpc.CallOption) (*Token_RevokeToken_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_RevokeToken_Resp)
	err := c.cc.Invoke(ctx, Token_RevokeToken_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) RevokeUserAccessTokens(ctx context.Context, in *Token_RevokeUserAccessTokens_Req, opts ...grpc.CallOption) (*Token_RevokeUserAccessTokens_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_RevokeUserAccessTokens_Resp)
	err := c.cc.Invoke(ctx, Token_RevokeUserAccessTokens_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) Search(ctx context.Context, in *Token_Search_Req, opts ...grpc.CallOption) (*Token_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_Search_Resp)
	err := c.cc.Invoke(ctx, Token_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) Update(ctx context.Context, in *Token_Update_Req, opts ...grpc.CallOption) (*Token_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_Update_Resp)
	err := c.cc.Invoke(ctx, Token_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) UpdateById(ctx context.Context, in *Token_UpdateById_Req, opts ...grpc.CallOption) (*Token_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_UpdateById_Resp)
	err := c.cc.Invoke(ctx, Token_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tokenClient) ValidateToken(ctx context.Context, in *Token_ValidateToken_Req, opts ...grpc.CallOption) (*Token_ValidateToken_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Token_ValidateToken_Resp)
	err := c.cc.Invoke(ctx, Token_ValidateToken_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TokenServer is the server API for Token service.
// All implementations must embed UnimplementedTokenServer
// for forward compatibility.
//
// Model: Token
type TokenServer interface {
	Browse(context.Context, *Token_Browse_Req) (*Token_Browse_Resp, error)
	BrowseMany(context.Context, *Token_BrowseMany_Req) (*Token_BrowseMany_Resp, error)
	CleanExpiredTokens(context.Context, *emptypb.Empty) (*Token_CleanExpiredTokens_Resp, error)
	Count(context.Context, *Token_Count_Req) (*Token_Count_Resp, error)
	Create(context.Context, *Token_Create_Req) (*Token_Create_Resp, error)
	CreateMany(context.Context, *Token_CreateMany_Req) (*Token_CreateMany_Resp, error)
	CreateTokenPair(context.Context, *Token_CreateTokenPair_Req) (*Token_CreateTokenPair_Resp, error)
	DefaultGet(context.Context, *Token_DefaultGet_Req) (*Token_DefaultGet_Resp, error)
	Delete(context.Context, *Token_Delete_Req) (*Token_Delete_Resp, error)
	DeleteById(context.Context, *Token_DeleteById_Req) (*Token_DeleteById_Resp, error)
	Onchange(context.Context, *Token_Onchange_Req) (*Token_Onchange_Resp, error)
	ReadGroup(context.Context, *Token_ReadGroup_Req) (*Token_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *Token_ReadGroupCount_Req) (*Token_ReadGroupCount_Resp, error)
	RefreshTokens(context.Context, *Token_RefreshTokens_Req) (*Token_RefreshTokens_Resp, error)
	RevokeAllUserTokens(context.Context, *Token_RevokeAllUserTokens_Req) (*Token_RevokeAllUserTokens_Resp, error)
	RevokeToken(context.Context, *Token_RevokeToken_Req) (*Token_RevokeToken_Resp, error)
	RevokeUserAccessTokens(context.Context, *Token_RevokeUserAccessTokens_Req) (*Token_RevokeUserAccessTokens_Resp, error)
	Search(context.Context, *Token_Search_Req) (*Token_Search_Resp, error)
	Update(context.Context, *Token_Update_Req) (*Token_Update_Resp, error)
	UpdateById(context.Context, *Token_UpdateById_Req) (*Token_UpdateById_Resp, error)
	ValidateToken(context.Context, *Token_ValidateToken_Req) (*Token_ValidateToken_Resp, error)
	mustEmbedUnimplementedTokenServer()
}

// UnimplementedTokenServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedTokenServer struct{}

func (UnimplementedTokenServer) Browse(context.Context, *Token_Browse_Req) (*Token_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedTokenServer) BrowseMany(context.Context, *Token_BrowseMany_Req) (*Token_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedTokenServer) CleanExpiredTokens(context.Context, *emptypb.Empty) (*Token_CleanExpiredTokens_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CleanExpiredTokens not implemented")
}
func (UnimplementedTokenServer) Count(context.Context, *Token_Count_Req) (*Token_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedTokenServer) Create(context.Context, *Token_Create_Req) (*Token_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedTokenServer) CreateMany(context.Context, *Token_CreateMany_Req) (*Token_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedTokenServer) CreateTokenPair(context.Context, *Token_CreateTokenPair_Req) (*Token_CreateTokenPair_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateTokenPair not implemented")
}
func (UnimplementedTokenServer) DefaultGet(context.Context, *Token_DefaultGet_Req) (*Token_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedTokenServer) Delete(context.Context, *Token_Delete_Req) (*Token_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedTokenServer) DeleteById(context.Context, *Token_DeleteById_Req) (*Token_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedTokenServer) Onchange(context.Context, *Token_Onchange_Req) (*Token_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedTokenServer) ReadGroup(context.Context, *Token_ReadGroup_Req) (*Token_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedTokenServer) ReadGroupCount(context.Context, *Token_ReadGroupCount_Req) (*Token_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedTokenServer) RefreshTokens(context.Context, *Token_RefreshTokens_Req) (*Token_RefreshTokens_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method RefreshTokens not implemented")
}
func (UnimplementedTokenServer) RevokeAllUserTokens(context.Context, *Token_RevokeAllUserTokens_Req) (*Token_RevokeAllUserTokens_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method RevokeAllUserTokens not implemented")
}
func (UnimplementedTokenServer) RevokeToken(context.Context, *Token_RevokeToken_Req) (*Token_RevokeToken_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method RevokeToken not implemented")
}
func (UnimplementedTokenServer) RevokeUserAccessTokens(context.Context, *Token_RevokeUserAccessTokens_Req) (*Token_RevokeUserAccessTokens_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method RevokeUserAccessTokens not implemented")
}
func (UnimplementedTokenServer) Search(context.Context, *Token_Search_Req) (*Token_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedTokenServer) Update(context.Context, *Token_Update_Req) (*Token_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedTokenServer) UpdateById(context.Context, *Token_UpdateById_Req) (*Token_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedTokenServer) ValidateToken(context.Context, *Token_ValidateToken_Req) (*Token_ValidateToken_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ValidateToken not implemented")
}
func (UnimplementedTokenServer) mustEmbedUnimplementedTokenServer() {}
func (UnimplementedTokenServer) testEmbeddedByValue()               {}

// UnsafeTokenServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TokenServer will
// result in compilation errors.
type UnsafeTokenServer interface {
	mustEmbedUnimplementedTokenServer()
}

func RegisterTokenServer(s grpc.ServiceRegistrar, srv TokenServer) {
	// If the following call panics, it indicates UnimplementedTokenServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Token_ServiceDesc, srv)
}

func _Token_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).Browse(ctx, req.(*Token_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).BrowseMany(ctx, req.(*Token_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_CleanExpiredTokens_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).CleanExpiredTokens(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_CleanExpiredTokens_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).CleanExpiredTokens(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).Count(ctx, req.(*Token_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).Create(ctx, req.(*Token_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).CreateMany(ctx, req.(*Token_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_CreateTokenPair_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_CreateTokenPair_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).CreateTokenPair(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_CreateTokenPair_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).CreateTokenPair(ctx, req.(*Token_CreateTokenPair_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).DefaultGet(ctx, req.(*Token_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).Delete(ctx, req.(*Token_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).DeleteById(ctx, req.(*Token_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).Onchange(ctx, req.(*Token_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).ReadGroup(ctx, req.(*Token_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).ReadGroupCount(ctx, req.(*Token_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_RefreshTokens_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_RefreshTokens_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).RefreshTokens(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_RefreshTokens_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).RefreshTokens(ctx, req.(*Token_RefreshTokens_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_RevokeAllUserTokens_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_RevokeAllUserTokens_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).RevokeAllUserTokens(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_RevokeAllUserTokens_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).RevokeAllUserTokens(ctx, req.(*Token_RevokeAllUserTokens_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_RevokeToken_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_RevokeToken_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).RevokeToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_RevokeToken_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).RevokeToken(ctx, req.(*Token_RevokeToken_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_RevokeUserAccessTokens_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_RevokeUserAccessTokens_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).RevokeUserAccessTokens(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_RevokeUserAccessTokens_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).RevokeUserAccessTokens(ctx, req.(*Token_RevokeUserAccessTokens_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).Search(ctx, req.(*Token_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).Update(ctx, req.(*Token_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).UpdateById(ctx, req.(*Token_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _Token_ValidateToken_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Token_ValidateToken_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TokenServer).ValidateToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Token_ValidateToken_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TokenServer).ValidateToken(ctx, req.(*Token_ValidateToken_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// Token_ServiceDesc is the grpc.ServiceDesc for Token service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Token_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.Token",
	HandlerType: (*TokenServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _Token_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _Token_BrowseMany_Handler,
		},
		{
			MethodName: "CleanExpiredTokens",
			Handler:    _Token_CleanExpiredTokens_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _Token_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _Token_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _Token_CreateMany_Handler,
		},
		{
			MethodName: "CreateTokenPair",
			Handler:    _Token_CreateTokenPair_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _Token_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _Token_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _Token_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _Token_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _Token_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _Token_ReadGroupCount_Handler,
		},
		{
			MethodName: "RefreshTokens",
			Handler:    _Token_RefreshTokens_Handler,
		},
		{
			MethodName: "RevokeAllUserTokens",
			Handler:    _Token_RevokeAllUserTokens_Handler,
		},
		{
			MethodName: "RevokeToken",
			Handler:    _Token_RevokeToken_Handler,
		},
		{
			MethodName: "RevokeUserAccessTokens",
			Handler:    _Token_RevokeUserAccessTokens_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _Token_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _Token_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _Token_UpdateById_Handler,
		},
		{
			MethodName: "ValidateToken",
			Handler:    _Token_ValidateToken_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	User_AssignRoles_FullMethodName            = "/auth.User/AssignRoles"
	User_Browse_FullMethodName                 = "/auth.User/Browse"
	User_BrowseMany_FullMethodName             = "/auth.User/BrowseMany"
	User_ChangePassword_FullMethodName         = "/auth.User/ChangePassword"
	User_CheckMethodAccess_FullMethodName      = "/auth.User/CheckMethodAccess"
	User_Count_FullMethodName                  = "/auth.User/Count"
	User_Create_FullMethodName                 = "/auth.User/Create"
	User_CreateMany_FullMethodName             = "/auth.User/CreateMany"
	User_DefaultGet_FullMethodName             = "/auth.User/DefaultGet"
	User_Delete_FullMethodName                 = "/auth.User/Delete"
	User_DeleteById_FullMethodName             = "/auth.User/DeleteById"
	User_GetRecordRuleCondition_FullMethodName = "/auth.User/GetRecordRuleCondition"
	User_HasPermission_FullMethodName          = "/auth.User/HasPermission"
	User_HasRole_FullMethodName                = "/auth.User/HasRole"
	User_Login_FullMethodName                  = "/auth.User/Login"
	User_Logout_FullMethodName                 = "/auth.User/Logout"
	User_Onchange_FullMethodName               = "/auth.User/Onchange"
	User_ReadGroup_FullMethodName              = "/auth.User/ReadGroup"
	User_ReadGroupCount_FullMethodName         = "/auth.User/ReadGroupCount"
	User_RefreshTokens_FullMethodName          = "/auth.User/RefreshTokens"
	User_Register_FullMethodName               = "/auth.User/Register"
	User_RemoveRoles_FullMethodName            = "/auth.User/RemoveRoles"
	User_ResetPassword_FullMethodName          = "/auth.User/ResetPassword"
	User_Search_FullMethodName                 = "/auth.User/Search"
	User_Update_FullMethodName                 = "/auth.User/Update"
	User_UpdateById_FullMethodName             = "/auth.User/UpdateById"
)

// UserClient is the client API for User service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: User
type UserClient interface {
	AssignRoles(ctx context.Context, in *User_AssignRoles_Req, opts ...grpc.CallOption) (*User_AssignRoles_Resp, error)
	Browse(ctx context.Context, in *User_Browse_Req, opts ...grpc.CallOption) (*User_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *User_BrowseMany_Req, opts ...grpc.CallOption) (*User_BrowseMany_Resp, error)
	ChangePassword(ctx context.Context, in *User_ChangePassword_Req, opts ...grpc.CallOption) (*User_ChangePassword_Resp, error)
	CheckMethodAccess(ctx context.Context, in *User_CheckMethodAccess_Req, opts ...grpc.CallOption) (*User_CheckMethodAccess_Resp, error)
	Count(ctx context.Context, in *User_Count_Req, opts ...grpc.CallOption) (*User_Count_Resp, error)
	Create(ctx context.Context, in *User_Create_Req, opts ...grpc.CallOption) (*User_Create_Resp, error)
	CreateMany(ctx context.Context, in *User_CreateMany_Req, opts ...grpc.CallOption) (*User_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *User_DefaultGet_Req, opts ...grpc.CallOption) (*User_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *User_Delete_Req, opts ...grpc.CallOption) (*User_Delete_Resp, error)
	DeleteById(ctx context.Context, in *User_DeleteById_Req, opts ...grpc.CallOption) (*User_DeleteById_Resp, error)
	GetRecordRuleCondition(ctx context.Context, in *User_GetRecordRuleCondition_Req, opts ...grpc.CallOption) (*User_GetRecordRuleCondition_Resp, error)
	HasPermission(ctx context.Context, in *User_HasPermission_Req, opts ...grpc.CallOption) (*User_HasPermission_Resp, error)
	HasRole(ctx context.Context, in *User_HasRole_Req, opts ...grpc.CallOption) (*User_HasRole_Resp, error)
	Login(ctx context.Context, in *User_Login_Req, opts ...grpc.CallOption) (*User_Login_Resp, error)
	Logout(ctx context.Context, in *User_Logout_Req, opts ...grpc.CallOption) (*User_Logout_Resp, error)
	Onchange(ctx context.Context, in *User_Onchange_Req, opts ...grpc.CallOption) (*User_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *User_ReadGroup_Req, opts ...grpc.CallOption) (*User_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *User_ReadGroupCount_Req, opts ...grpc.CallOption) (*User_ReadGroupCount_Resp, error)
	RefreshTokens(ctx context.Context, in *User_RefreshTokens_Req, opts ...grpc.CallOption) (*User_RefreshTokens_Resp, error)
	Register(ctx context.Context, in *User_Register_Req, opts ...grpc.CallOption) (*User_Register_Resp, error)
	RemoveRoles(ctx context.Context, in *User_RemoveRoles_Req, opts ...grpc.CallOption) (*User_RemoveRoles_Resp, error)
	ResetPassword(ctx context.Context, in *User_ResetPassword_Req, opts ...grpc.CallOption) (*User_ResetPassword_Resp, error)
	Search(ctx context.Context, in *User_Search_Req, opts ...grpc.CallOption) (*User_Search_Resp, error)
	Update(ctx context.Context, in *User_Update_Req, opts ...grpc.CallOption) (*User_Update_Resp, error)
	UpdateById(ctx context.Context, in *User_UpdateById_Req, opts ...grpc.CallOption) (*User_UpdateById_Resp, error)
}

type userClient struct {
	cc grpc.ClientConnInterface
}

func NewUserClient(cc grpc.ClientConnInterface) UserClient {
	return &userClient{cc}
}

func (c *userClient) AssignRoles(ctx context.Context, in *User_AssignRoles_Req, opts ...grpc.CallOption) (*User_AssignRoles_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_AssignRoles_Resp)
	err := c.cc.Invoke(ctx, User_AssignRoles_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Browse(ctx context.Context, in *User_Browse_Req, opts ...grpc.CallOption) (*User_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_Browse_Resp)
	err := c.cc.Invoke(ctx, User_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) BrowseMany(ctx context.Context, in *User_BrowseMany_Req, opts ...grpc.CallOption) (*User_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, User_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) ChangePassword(ctx context.Context, in *User_ChangePassword_Req, opts ...grpc.CallOption) (*User_ChangePassword_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_ChangePassword_Resp)
	err := c.cc.Invoke(ctx, User_ChangePassword_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) CheckMethodAccess(ctx context.Context, in *User_CheckMethodAccess_Req, opts ...grpc.CallOption) (*User_CheckMethodAccess_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_CheckMethodAccess_Resp)
	err := c.cc.Invoke(ctx, User_CheckMethodAccess_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Count(ctx context.Context, in *User_Count_Req, opts ...grpc.CallOption) (*User_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_Count_Resp)
	err := c.cc.Invoke(ctx, User_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Create(ctx context.Context, in *User_Create_Req, opts ...grpc.CallOption) (*User_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_Create_Resp)
	err := c.cc.Invoke(ctx, User_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) CreateMany(ctx context.Context, in *User_CreateMany_Req, opts ...grpc.CallOption) (*User_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_CreateMany_Resp)
	err := c.cc.Invoke(ctx, User_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) DefaultGet(ctx context.Context, in *User_DefaultGet_Req, opts ...grpc.CallOption) (*User_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, User_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Delete(ctx context.Context, in *User_Delete_Req, opts ...grpc.CallOption) (*User_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_Delete_Resp)
	err := c.cc.Invoke(ctx, User_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) DeleteById(ctx context.Context, in *User_DeleteById_Req, opts ...grpc.CallOption) (*User_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_DeleteById_Resp)
	err := c.cc.Invoke(ctx, User_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) GetRecordRuleCondition(ctx context.Context, in *User_GetRecordRuleCondition_Req, opts ...grpc.CallOption) (*User_GetRecordRuleCondition_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_GetRecordRuleCondition_Resp)
	err := c.cc.Invoke(ctx, User_GetRecordRuleCondition_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) HasPermission(ctx context.Context, in *User_HasPermission_Req, opts ...grpc.CallOption) (*User_HasPermission_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_HasPermission_Resp)
	err := c.cc.Invoke(ctx, User_HasPermission_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) HasRole(ctx context.Context, in *User_HasRole_Req, opts ...grpc.CallOption) (*User_HasRole_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_HasRole_Resp)
	err := c.cc.Invoke(ctx, User_HasRole_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Login(ctx context.Context, in *User_Login_Req, opts ...grpc.CallOption) (*User_Login_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_Login_Resp)
	err := c.cc.Invoke(ctx, User_Login_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Logout(ctx context.Context, in *User_Logout_Req, opts ...grpc.CallOption) (*User_Logout_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_Logout_Resp)
	err := c.cc.Invoke(ctx, User_Logout_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Onchange(ctx context.Context, in *User_Onchange_Req, opts ...grpc.CallOption) (*User_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_Onchange_Resp)
	err := c.cc.Invoke(ctx, User_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) ReadGroup(ctx context.Context, in *User_ReadGroup_Req, opts ...grpc.CallOption) (*User_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, User_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) ReadGroupCount(ctx context.Context, in *User_ReadGroupCount_Req, opts ...grpc.CallOption) (*User_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, User_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) RefreshTokens(ctx context.Context, in *User_RefreshTokens_Req, opts ...grpc.CallOption) (*User_RefreshTokens_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_RefreshTokens_Resp)
	err := c.cc.Invoke(ctx, User_RefreshTokens_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Register(ctx context.Context, in *User_Register_Req, opts ...grpc.CallOption) (*User_Register_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_Register_Resp)
	err := c.cc.Invoke(ctx, User_Register_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) RemoveRoles(ctx context.Context, in *User_RemoveRoles_Req, opts ...grpc.CallOption) (*User_RemoveRoles_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_RemoveRoles_Resp)
	err := c.cc.Invoke(ctx, User_RemoveRoles_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) ResetPassword(ctx context.Context, in *User_ResetPassword_Req, opts ...grpc.CallOption) (*User_ResetPassword_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_ResetPassword_Resp)
	err := c.cc.Invoke(ctx, User_ResetPassword_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Search(ctx context.Context, in *User_Search_Req, opts ...grpc.CallOption) (*User_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_Search_Resp)
	err := c.cc.Invoke(ctx, User_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Update(ctx context.Context, in *User_Update_Req, opts ...grpc.CallOption) (*User_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_Update_Resp)
	err := c.cc.Invoke(ctx, User_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) UpdateById(ctx context.Context, in *User_UpdateById_Req, opts ...grpc.CallOption) (*User_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(User_UpdateById_Resp)
	err := c.cc.Invoke(ctx, User_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UserServer is the server API for User service.
// All implementations must embed UnimplementedUserServer
// for forward compatibility.
//
// Model: User
type UserServer interface {
	AssignRoles(context.Context, *User_AssignRoles_Req) (*User_AssignRoles_Resp, error)
	Browse(context.Context, *User_Browse_Req) (*User_Browse_Resp, error)
	BrowseMany(context.Context, *User_BrowseMany_Req) (*User_BrowseMany_Resp, error)
	ChangePassword(context.Context, *User_ChangePassword_Req) (*User_ChangePassword_Resp, error)
	CheckMethodAccess(context.Context, *User_CheckMethodAccess_Req) (*User_CheckMethodAccess_Resp, error)
	Count(context.Context, *User_Count_Req) (*User_Count_Resp, error)
	Create(context.Context, *User_Create_Req) (*User_Create_Resp, error)
	CreateMany(context.Context, *User_CreateMany_Req) (*User_CreateMany_Resp, error)
	DefaultGet(context.Context, *User_DefaultGet_Req) (*User_DefaultGet_Resp, error)
	Delete(context.Context, *User_Delete_Req) (*User_Delete_Resp, error)
	DeleteById(context.Context, *User_DeleteById_Req) (*User_DeleteById_Resp, error)
	GetRecordRuleCondition(context.Context, *User_GetRecordRuleCondition_Req) (*User_GetRecordRuleCondition_Resp, error)
	HasPermission(context.Context, *User_HasPermission_Req) (*User_HasPermission_Resp, error)
	HasRole(context.Context, *User_HasRole_Req) (*User_HasRole_Resp, error)
	Login(context.Context, *User_Login_Req) (*User_Login_Resp, error)
	Logout(context.Context, *User_Logout_Req) (*User_Logout_Resp, error)
	Onchange(context.Context, *User_Onchange_Req) (*User_Onchange_Resp, error)
	ReadGroup(context.Context, *User_ReadGroup_Req) (*User_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *User_ReadGroupCount_Req) (*User_ReadGroupCount_Resp, error)
	RefreshTokens(context.Context, *User_RefreshTokens_Req) (*User_RefreshTokens_Resp, error)
	Register(context.Context, *User_Register_Req) (*User_Register_Resp, error)
	RemoveRoles(context.Context, *User_RemoveRoles_Req) (*User_RemoveRoles_Resp, error)
	ResetPassword(context.Context, *User_ResetPassword_Req) (*User_ResetPassword_Resp, error)
	Search(context.Context, *User_Search_Req) (*User_Search_Resp, error)
	Update(context.Context, *User_Update_Req) (*User_Update_Resp, error)
	UpdateById(context.Context, *User_UpdateById_Req) (*User_UpdateById_Resp, error)
	mustEmbedUnimplementedUserServer()
}

// UnimplementedUserServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedUserServer struct{}

func (UnimplementedUserServer) AssignRoles(context.Context, *User_AssignRoles_Req) (*User_AssignRoles_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method AssignRoles not implemented")
}
func (UnimplementedUserServer) Browse(context.Context, *User_Browse_Req) (*User_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedUserServer) BrowseMany(context.Context, *User_BrowseMany_Req) (*User_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedUserServer) ChangePassword(context.Context, *User_ChangePassword_Req) (*User_ChangePassword_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ChangePassword not implemented")
}
func (UnimplementedUserServer) CheckMethodAccess(context.Context, *User_CheckMethodAccess_Req) (*User_CheckMethodAccess_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CheckMethodAccess not implemented")
}
func (UnimplementedUserServer) Count(context.Context, *User_Count_Req) (*User_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedUserServer) Create(context.Context, *User_Create_Req) (*User_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedUserServer) CreateMany(context.Context, *User_CreateMany_Req) (*User_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedUserServer) DefaultGet(context.Context, *User_DefaultGet_Req) (*User_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedUserServer) Delete(context.Context, *User_Delete_Req) (*User_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedUserServer) DeleteById(context.Context, *User_DeleteById_Req) (*User_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedUserServer) GetRecordRuleCondition(context.Context, *User_GetRecordRuleCondition_Req) (*User_GetRecordRuleCondition_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method GetRecordRuleCondition not implemented")
}
func (UnimplementedUserServer) HasPermission(context.Context, *User_HasPermission_Req) (*User_HasPermission_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method HasPermission not implemented")
}
func (UnimplementedUserServer) HasRole(context.Context, *User_HasRole_Req) (*User_HasRole_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method HasRole not implemented")
}
func (UnimplementedUserServer) Login(context.Context, *User_Login_Req) (*User_Login_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Login not implemented")
}
func (UnimplementedUserServer) Logout(context.Context, *User_Logout_Req) (*User_Logout_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Logout not implemented")
}
func (UnimplementedUserServer) Onchange(context.Context, *User_Onchange_Req) (*User_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedUserServer) ReadGroup(context.Context, *User_ReadGroup_Req) (*User_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedUserServer) ReadGroupCount(context.Context, *User_ReadGroupCount_Req) (*User_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedUserServer) RefreshTokens(context.Context, *User_RefreshTokens_Req) (*User_RefreshTokens_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method RefreshTokens not implemented")
}
func (UnimplementedUserServer) Register(context.Context, *User_Register_Req) (*User_Register_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Register not implemented")
}
func (UnimplementedUserServer) RemoveRoles(context.Context, *User_RemoveRoles_Req) (*User_RemoveRoles_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method RemoveRoles not implemented")
}
func (UnimplementedUserServer) ResetPassword(context.Context, *User_ResetPassword_Req) (*User_ResetPassword_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ResetPassword not implemented")
}
func (UnimplementedUserServer) Search(context.Context, *User_Search_Req) (*User_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedUserServer) Update(context.Context, *User_Update_Req) (*User_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedUserServer) UpdateById(context.Context, *User_UpdateById_Req) (*User_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedUserServer) mustEmbedUnimplementedUserServer() {}
func (UnimplementedUserServer) testEmbeddedByValue()              {}

// UnsafeUserServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to UserServer will
// result in compilation errors.
type UnsafeUserServer interface {
	mustEmbedUnimplementedUserServer()
}

func RegisterUserServer(s grpc.ServiceRegistrar, srv UserServer) {
	// If the following call panics, it indicates UnimplementedUserServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&User_ServiceDesc, srv)
}

func _User_AssignRoles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_AssignRoles_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).AssignRoles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_AssignRoles_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).AssignRoles(ctx, req.(*User_AssignRoles_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Browse(ctx, req.(*User_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).BrowseMany(ctx, req.(*User_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_ChangePassword_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_ChangePassword_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).ChangePassword(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_ChangePassword_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).ChangePassword(ctx, req.(*User_ChangePassword_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_CheckMethodAccess_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_CheckMethodAccess_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).CheckMethodAccess(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_CheckMethodAccess_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).CheckMethodAccess(ctx, req.(*User_CheckMethodAccess_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Count(ctx, req.(*User_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Create(ctx, req.(*User_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).CreateMany(ctx, req.(*User_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).DefaultGet(ctx, req.(*User_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Delete(ctx, req.(*User_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).DeleteById(ctx, req.(*User_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_GetRecordRuleCondition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_GetRecordRuleCondition_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).GetRecordRuleCondition(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_GetRecordRuleCondition_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).GetRecordRuleCondition(ctx, req.(*User_GetRecordRuleCondition_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_HasPermission_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_HasPermission_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).HasPermission(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_HasPermission_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).HasPermission(ctx, req.(*User_HasPermission_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_HasRole_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_HasRole_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).HasRole(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_HasRole_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).HasRole(ctx, req.(*User_HasRole_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Login_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_Login_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Login(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_Login_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Login(ctx, req.(*User_Login_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Logout_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_Logout_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Logout(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_Logout_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Logout(ctx, req.(*User_Logout_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Onchange(ctx, req.(*User_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).ReadGroup(ctx, req.(*User_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).ReadGroupCount(ctx, req.(*User_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_RefreshTokens_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_RefreshTokens_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).RefreshTokens(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_RefreshTokens_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).RefreshTokens(ctx, req.(*User_RefreshTokens_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_Register_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_Register_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Register(ctx, req.(*User_Register_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_RemoveRoles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_RemoveRoles_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).RemoveRoles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_RemoveRoles_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).RemoveRoles(ctx, req.(*User_RemoveRoles_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_ResetPassword_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_ResetPassword_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).ResetPassword(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_ResetPassword_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).ResetPassword(ctx, req.(*User_ResetPassword_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Search(ctx, req.(*User_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Update(ctx, req.(*User_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(User_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: User_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).UpdateById(ctx, req.(*User_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// User_ServiceDesc is the grpc.ServiceDesc for User service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var User_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.User",
	HandlerType: (*UserServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AssignRoles",
			Handler:    _User_AssignRoles_Handler,
		},
		{
			MethodName: "Browse",
			Handler:    _User_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _User_BrowseMany_Handler,
		},
		{
			MethodName: "ChangePassword",
			Handler:    _User_ChangePassword_Handler,
		},
		{
			MethodName: "CheckMethodAccess",
			Handler:    _User_CheckMethodAccess_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _User_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _User_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _User_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _User_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _User_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _User_DeleteById_Handler,
		},
		{
			MethodName: "GetRecordRuleCondition",
			Handler:    _User_GetRecordRuleCondition_Handler,
		},
		{
			MethodName: "HasPermission",
			Handler:    _User_HasPermission_Handler,
		},
		{
			MethodName: "HasRole",
			Handler:    _User_HasRole_Handler,
		},
		{
			MethodName: "Login",
			Handler:    _User_Login_Handler,
		},
		{
			MethodName: "Logout",
			Handler:    _User_Logout_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _User_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _User_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _User_ReadGroupCount_Handler,
		},
		{
			MethodName: "RefreshTokens",
			Handler:    _User_RefreshTokens_Handler,
		},
		{
			MethodName: "Register",
			Handler:    _User_Register_Handler,
		},
		{
			MethodName: "RemoveRoles",
			Handler:    _User_RemoveRoles_Handler,
		},
		{
			MethodName: "ResetPassword",
			Handler:    _User_ResetPassword_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _User_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _User_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _User_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}

const (
	UserRole_Browse_FullMethodName         = "/auth.UserRole/Browse"
	UserRole_BrowseMany_FullMethodName     = "/auth.UserRole/BrowseMany"
	UserRole_Count_FullMethodName          = "/auth.UserRole/Count"
	UserRole_Create_FullMethodName         = "/auth.UserRole/Create"
	UserRole_CreateMany_FullMethodName     = "/auth.UserRole/CreateMany"
	UserRole_DefaultGet_FullMethodName     = "/auth.UserRole/DefaultGet"
	UserRole_Delete_FullMethodName         = "/auth.UserRole/Delete"
	UserRole_DeleteById_FullMethodName     = "/auth.UserRole/DeleteById"
	UserRole_Onchange_FullMethodName       = "/auth.UserRole/Onchange"
	UserRole_ReadGroup_FullMethodName      = "/auth.UserRole/ReadGroup"
	UserRole_ReadGroupCount_FullMethodName = "/auth.UserRole/ReadGroupCount"
	UserRole_Search_FullMethodName         = "/auth.UserRole/Search"
	UserRole_Update_FullMethodName         = "/auth.UserRole/Update"
	UserRole_UpdateById_FullMethodName     = "/auth.UserRole/UpdateById"
)

// UserRoleClient is the client API for UserRole service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Model: UserRole
type UserRoleClient interface {
	Browse(ctx context.Context, in *UserRole_Browse_Req, opts ...grpc.CallOption) (*UserRole_Browse_Resp, error)
	BrowseMany(ctx context.Context, in *UserRole_BrowseMany_Req, opts ...grpc.CallOption) (*UserRole_BrowseMany_Resp, error)
	Count(ctx context.Context, in *UserRole_Count_Req, opts ...grpc.CallOption) (*UserRole_Count_Resp, error)
	Create(ctx context.Context, in *UserRole_Create_Req, opts ...grpc.CallOption) (*UserRole_Create_Resp, error)
	CreateMany(ctx context.Context, in *UserRole_CreateMany_Req, opts ...grpc.CallOption) (*UserRole_CreateMany_Resp, error)
	DefaultGet(ctx context.Context, in *UserRole_DefaultGet_Req, opts ...grpc.CallOption) (*UserRole_DefaultGet_Resp, error)
	Delete(ctx context.Context, in *UserRole_Delete_Req, opts ...grpc.CallOption) (*UserRole_Delete_Resp, error)
	DeleteById(ctx context.Context, in *UserRole_DeleteById_Req, opts ...grpc.CallOption) (*UserRole_DeleteById_Resp, error)
	Onchange(ctx context.Context, in *UserRole_Onchange_Req, opts ...grpc.CallOption) (*UserRole_Onchange_Resp, error)
	ReadGroup(ctx context.Context, in *UserRole_ReadGroup_Req, opts ...grpc.CallOption) (*UserRole_ReadGroup_Resp, error)
	ReadGroupCount(ctx context.Context, in *UserRole_ReadGroupCount_Req, opts ...grpc.CallOption) (*UserRole_ReadGroupCount_Resp, error)
	Search(ctx context.Context, in *UserRole_Search_Req, opts ...grpc.CallOption) (*UserRole_Search_Resp, error)
	Update(ctx context.Context, in *UserRole_Update_Req, opts ...grpc.CallOption) (*UserRole_Update_Resp, error)
	UpdateById(ctx context.Context, in *UserRole_UpdateById_Req, opts ...grpc.CallOption) (*UserRole_UpdateById_Resp, error)
}

type userRoleClient struct {
	cc grpc.ClientConnInterface
}

func NewUserRoleClient(cc grpc.ClientConnInterface) UserRoleClient {
	return &userRoleClient{cc}
}

func (c *userRoleClient) Browse(ctx context.Context, in *UserRole_Browse_Req, opts ...grpc.CallOption) (*UserRole_Browse_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_Browse_Resp)
	err := c.cc.Invoke(ctx, UserRole_Browse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) BrowseMany(ctx context.Context, in *UserRole_BrowseMany_Req, opts ...grpc.CallOption) (*UserRole_BrowseMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_BrowseMany_Resp)
	err := c.cc.Invoke(ctx, UserRole_BrowseMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) Count(ctx context.Context, in *UserRole_Count_Req, opts ...grpc.CallOption) (*UserRole_Count_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_Count_Resp)
	err := c.cc.Invoke(ctx, UserRole_Count_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) Create(ctx context.Context, in *UserRole_Create_Req, opts ...grpc.CallOption) (*UserRole_Create_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_Create_Resp)
	err := c.cc.Invoke(ctx, UserRole_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) CreateMany(ctx context.Context, in *UserRole_CreateMany_Req, opts ...grpc.CallOption) (*UserRole_CreateMany_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_CreateMany_Resp)
	err := c.cc.Invoke(ctx, UserRole_CreateMany_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) DefaultGet(ctx context.Context, in *UserRole_DefaultGet_Req, opts ...grpc.CallOption) (*UserRole_DefaultGet_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_DefaultGet_Resp)
	err := c.cc.Invoke(ctx, UserRole_DefaultGet_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) Delete(ctx context.Context, in *UserRole_Delete_Req, opts ...grpc.CallOption) (*UserRole_Delete_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_Delete_Resp)
	err := c.cc.Invoke(ctx, UserRole_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) DeleteById(ctx context.Context, in *UserRole_DeleteById_Req, opts ...grpc.CallOption) (*UserRole_DeleteById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_DeleteById_Resp)
	err := c.cc.Invoke(ctx, UserRole_DeleteById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) Onchange(ctx context.Context, in *UserRole_Onchange_Req, opts ...grpc.CallOption) (*UserRole_Onchange_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_Onchange_Resp)
	err := c.cc.Invoke(ctx, UserRole_Onchange_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) ReadGroup(ctx context.Context, in *UserRole_ReadGroup_Req, opts ...grpc.CallOption) (*UserRole_ReadGroup_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_ReadGroup_Resp)
	err := c.cc.Invoke(ctx, UserRole_ReadGroup_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) ReadGroupCount(ctx context.Context, in *UserRole_ReadGroupCount_Req, opts ...grpc.CallOption) (*UserRole_ReadGroupCount_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_ReadGroupCount_Resp)
	err := c.cc.Invoke(ctx, UserRole_ReadGroupCount_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) Search(ctx context.Context, in *UserRole_Search_Req, opts ...grpc.CallOption) (*UserRole_Search_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_Search_Resp)
	err := c.cc.Invoke(ctx, UserRole_Search_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) Update(ctx context.Context, in *UserRole_Update_Req, opts ...grpc.CallOption) (*UserRole_Update_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_Update_Resp)
	err := c.cc.Invoke(ctx, UserRole_Update_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userRoleClient) UpdateById(ctx context.Context, in *UserRole_UpdateById_Req, opts ...grpc.CallOption) (*UserRole_UpdateById_Resp, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UserRole_UpdateById_Resp)
	err := c.cc.Invoke(ctx, UserRole_UpdateById_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UserRoleServer is the server API for UserRole service.
// All implementations must embed UnimplementedUserRoleServer
// for forward compatibility.
//
// Model: UserRole
type UserRoleServer interface {
	Browse(context.Context, *UserRole_Browse_Req) (*UserRole_Browse_Resp, error)
	BrowseMany(context.Context, *UserRole_BrowseMany_Req) (*UserRole_BrowseMany_Resp, error)
	Count(context.Context, *UserRole_Count_Req) (*UserRole_Count_Resp, error)
	Create(context.Context, *UserRole_Create_Req) (*UserRole_Create_Resp, error)
	CreateMany(context.Context, *UserRole_CreateMany_Req) (*UserRole_CreateMany_Resp, error)
	DefaultGet(context.Context, *UserRole_DefaultGet_Req) (*UserRole_DefaultGet_Resp, error)
	Delete(context.Context, *UserRole_Delete_Req) (*UserRole_Delete_Resp, error)
	DeleteById(context.Context, *UserRole_DeleteById_Req) (*UserRole_DeleteById_Resp, error)
	Onchange(context.Context, *UserRole_Onchange_Req) (*UserRole_Onchange_Resp, error)
	ReadGroup(context.Context, *UserRole_ReadGroup_Req) (*UserRole_ReadGroup_Resp, error)
	ReadGroupCount(context.Context, *UserRole_ReadGroupCount_Req) (*UserRole_ReadGroupCount_Resp, error)
	Search(context.Context, *UserRole_Search_Req) (*UserRole_Search_Resp, error)
	Update(context.Context, *UserRole_Update_Req) (*UserRole_Update_Resp, error)
	UpdateById(context.Context, *UserRole_UpdateById_Req) (*UserRole_UpdateById_Resp, error)
	mustEmbedUnimplementedUserRoleServer()
}

// UnimplementedUserRoleServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedUserRoleServer struct{}

func (UnimplementedUserRoleServer) Browse(context.Context, *UserRole_Browse_Req) (*UserRole_Browse_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Browse not implemented")
}
func (UnimplementedUserRoleServer) BrowseMany(context.Context, *UserRole_BrowseMany_Req) (*UserRole_BrowseMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method BrowseMany not implemented")
}
func (UnimplementedUserRoleServer) Count(context.Context, *UserRole_Count_Req) (*UserRole_Count_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Count not implemented")
}
func (UnimplementedUserRoleServer) Create(context.Context, *UserRole_Create_Req) (*UserRole_Create_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedUserRoleServer) CreateMany(context.Context, *UserRole_CreateMany_Req) (*UserRole_CreateMany_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateMany not implemented")
}
func (UnimplementedUserRoleServer) DefaultGet(context.Context, *UserRole_DefaultGet_Req) (*UserRole_DefaultGet_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DefaultGet not implemented")
}
func (UnimplementedUserRoleServer) Delete(context.Context, *UserRole_Delete_Req) (*UserRole_Delete_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedUserRoleServer) DeleteById(context.Context, *UserRole_DeleteById_Req) (*UserRole_DeleteById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method DeleteById not implemented")
}
func (UnimplementedUserRoleServer) Onchange(context.Context, *UserRole_Onchange_Req) (*UserRole_Onchange_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Onchange not implemented")
}
func (UnimplementedUserRoleServer) ReadGroup(context.Context, *UserRole_ReadGroup_Req) (*UserRole_ReadGroup_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroup not implemented")
}
func (UnimplementedUserRoleServer) ReadGroupCount(context.Context, *UserRole_ReadGroupCount_Req) (*UserRole_ReadGroupCount_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method ReadGroupCount not implemented")
}
func (UnimplementedUserRoleServer) Search(context.Context, *UserRole_Search_Req) (*UserRole_Search_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Search not implemented")
}
func (UnimplementedUserRoleServer) Update(context.Context, *UserRole_Update_Req) (*UserRole_Update_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedUserRoleServer) UpdateById(context.Context, *UserRole_UpdateById_Req) (*UserRole_UpdateById_Resp, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateById not implemented")
}
func (UnimplementedUserRoleServer) mustEmbedUnimplementedUserRoleServer() {}
func (UnimplementedUserRoleServer) testEmbeddedByValue()                  {}

// UnsafeUserRoleServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to UserRoleServer will
// result in compilation errors.
type UnsafeUserRoleServer interface {
	mustEmbedUnimplementedUserRoleServer()
}

func RegisterUserRoleServer(s grpc.ServiceRegistrar, srv UserRoleServer) {
	// If the following call panics, it indicates UnimplementedUserRoleServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&UserRole_ServiceDesc, srv)
}

func _UserRole_Browse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_Browse_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).Browse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_Browse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).Browse(ctx, req.(*UserRole_Browse_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_BrowseMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_BrowseMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).BrowseMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_BrowseMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).BrowseMany(ctx, req.(*UserRole_BrowseMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_Count_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_Count_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).Count(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_Count_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).Count(ctx, req.(*UserRole_Count_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_Create_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).Create(ctx, req.(*UserRole_Create_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_CreateMany_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_CreateMany_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).CreateMany(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_CreateMany_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).CreateMany(ctx, req.(*UserRole_CreateMany_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_DefaultGet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_DefaultGet_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).DefaultGet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_DefaultGet_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).DefaultGet(ctx, req.(*UserRole_DefaultGet_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_Delete_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).Delete(ctx, req.(*UserRole_Delete_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_DeleteById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_DeleteById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).DeleteById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_DeleteById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).DeleteById(ctx, req.(*UserRole_DeleteById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_Onchange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_Onchange_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).Onchange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_Onchange_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).Onchange(ctx, req.(*UserRole_Onchange_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_ReadGroup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_ReadGroup_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).ReadGroup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_ReadGroup_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).ReadGroup(ctx, req.(*UserRole_ReadGroup_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_ReadGroupCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_ReadGroupCount_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).ReadGroupCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_ReadGroupCount_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).ReadGroupCount(ctx, req.(*UserRole_ReadGroupCount_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_Search_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_Search_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).Search(ctx, req.(*UserRole_Search_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_Update_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_Update_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).Update(ctx, req.(*UserRole_Update_Req))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserRole_UpdateById_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UserRole_UpdateById_Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserRoleServer).UpdateById(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: UserRole_UpdateById_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserRoleServer).UpdateById(ctx, req.(*UserRole_UpdateById_Req))
	}
	return interceptor(ctx, in, info, handler)
}

// UserRole_ServiceDesc is the grpc.ServiceDesc for UserRole service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var UserRole_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.UserRole",
	HandlerType: (*UserRoleServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Browse",
			Handler:    _UserRole_Browse_Handler,
		},
		{
			MethodName: "BrowseMany",
			Handler:    _UserRole_BrowseMany_Handler,
		},
		{
			MethodName: "Count",
			Handler:    _UserRole_Count_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _UserRole_Create_Handler,
		},
		{
			MethodName: "CreateMany",
			Handler:    _UserRole_CreateMany_Handler,
		},
		{
			MethodName: "DefaultGet",
			Handler:    _UserRole_DefaultGet_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _UserRole_Delete_Handler,
		},
		{
			MethodName: "DeleteById",
			Handler:    _UserRole_DeleteById_Handler,
		},
		{
			MethodName: "Onchange",
			Handler:    _UserRole_Onchange_Handler,
		},
		{
			MethodName: "ReadGroup",
			Handler:    _UserRole_ReadGroup_Handler,
		},
		{
			MethodName: "ReadGroupCount",
			Handler:    _UserRole_ReadGroupCount_Handler,
		},
		{
			MethodName: "Search",
			Handler:    _UserRole_Search_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _UserRole_Update_Handler,
		},
		{
			MethodName: "UpdateById",
			Handler:    _UserRole_UpdateById_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}
