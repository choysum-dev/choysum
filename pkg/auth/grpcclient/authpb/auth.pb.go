// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package authpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	structpb "google.golang.org/protobuf/types/known/structpb"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Language_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Browse_Req) Reset() {
	*x = Language_Browse_Req{}
	mi := &file_auth_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Browse_Req) ProtoMessage() {}

func (x *Language_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Browse_Req.ProtoReflect.Descriptor instead.
func (*Language_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{0}
}

func (x *Language_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Language_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Language_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Browse_Resp) Reset() {
	*x = Language_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Browse_Resp) ProtoMessage() {}

func (x *Language_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Browse_Resp.ProtoReflect.Descriptor instead.
func (*Language_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{1}
}

func (x *Language_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Language_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_BrowseMany_Req) Reset() {
	*x = Language_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_BrowseMany_Req) ProtoMessage() {}

func (x *Language_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*Language_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{2}
}

func (x *Language_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *Language_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Language_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_BrowseMany_Resp) Reset() {
	*x = Language_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_BrowseMany_Resp) ProtoMessage() {}

func (x *Language_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*Language_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{3}
}

func (x *Language_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Language_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Count_Req) Reset() {
	*x = Language_Count_Req{}
	mi := &file_auth_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Count_Req) ProtoMessage() {}

func (x *Language_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Count_Req.ProtoReflect.Descriptor instead.
func (*Language_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{4}
}

func (x *Language_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Language_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Count_Resp) Reset() {
	*x = Language_Count_Resp{}
	mi := &file_auth_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Count_Resp) ProtoMessage() {}

func (x *Language_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Count_Resp.ProtoReflect.Descriptor instead.
func (*Language_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{5}
}

func (x *Language_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Language_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Create_Req) Reset() {
	*x = Language_Create_Req{}
	mi := &file_auth_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Create_Req) ProtoMessage() {}

func (x *Language_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Create_Req.ProtoReflect.Descriptor instead.
func (*Language_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{6}
}

func (x *Language_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *Language_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Language_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Create_Resp) Reset() {
	*x = Language_Create_Resp{}
	mi := &file_auth_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Create_Resp) ProtoMessage() {}

func (x *Language_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Create_Resp.ProtoReflect.Descriptor instead.
func (*Language_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{7}
}

func (x *Language_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Language_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_CreateMany_Req) Reset() {
	*x = Language_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_CreateMany_Req) ProtoMessage() {}

func (x *Language_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*Language_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{8}
}

func (x *Language_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Language_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Language_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_CreateMany_Resp) Reset() {
	*x = Language_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_CreateMany_Resp) ProtoMessage() {}

func (x *Language_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*Language_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{9}
}

func (x *Language_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Language_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_DefaultGet_Req) Reset() {
	*x = Language_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_DefaultGet_Req) ProtoMessage() {}

func (x *Language_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*Language_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{10}
}

func (x *Language_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type Language_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_DefaultGet_Resp) Reset() {
	*x = Language_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_DefaultGet_Resp) ProtoMessage() {}

func (x *Language_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*Language_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{11}
}

func (x *Language_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Language_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Delete_Req) Reset() {
	*x = Language_Delete_Req{}
	mi := &file_auth_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Delete_Req) ProtoMessage() {}

func (x *Language_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Delete_Req.ProtoReflect.Descriptor instead.
func (*Language_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{12}
}

func (x *Language_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Language_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Delete_Resp) Reset() {
	*x = Language_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Delete_Resp) ProtoMessage() {}

func (x *Language_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Delete_Resp.ProtoReflect.Descriptor instead.
func (*Language_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{13}
}

func (x *Language_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Language_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_DeleteById_Req) Reset() {
	*x = Language_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[14]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_DeleteById_Req) ProtoMessage() {}

func (x *Language_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[14]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*Language_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{14}
}

func (x *Language_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type Language_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_DeleteById_Resp) Reset() {
	*x = Language_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[15]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_DeleteById_Resp) ProtoMessage() {}

func (x *Language_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[15]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*Language_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{15}
}

func (x *Language_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Language_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Onchange_Req) Reset() {
	*x = Language_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[16]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Onchange_Req) ProtoMessage() {}

func (x *Language_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[16]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Onchange_Req.ProtoReflect.Descriptor instead.
func (*Language_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{16}
}

func (x *Language_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *Language_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *Language_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type Language_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Onchange_Resp) Reset() {
	*x = Language_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[17]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Onchange_Resp) ProtoMessage() {}

func (x *Language_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[17]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*Language_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{17}
}

func (x *Language_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Language_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_ReadGroup_Req) Reset() {
	*x = Language_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[18]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_ReadGroup_Req) ProtoMessage() {}

func (x *Language_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[18]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*Language_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{18}
}

func (x *Language_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Language_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Language_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Language_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_ReadGroup_Resp) Reset() {
	*x = Language_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[19]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_ReadGroup_Resp) ProtoMessage() {}

func (x *Language_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[19]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*Language_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{19}
}

func (x *Language_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Language_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_ReadGroupCount_Req) Reset() {
	*x = Language_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[20]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_ReadGroupCount_Req) ProtoMessage() {}

func (x *Language_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[20]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*Language_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{20}
}

func (x *Language_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Language_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Language_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Language_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_ReadGroupCount_Resp) Reset() {
	*x = Language_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[21]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_ReadGroupCount_Resp) ProtoMessage() {}

func (x *Language_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[21]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*Language_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{21}
}

func (x *Language_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Language_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Search_Req) Reset() {
	*x = Language_Search_Req{}
	mi := &file_auth_proto_msgTypes[22]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Search_Req) ProtoMessage() {}

func (x *Language_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[22]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Search_Req.ProtoReflect.Descriptor instead.
func (*Language_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{22}
}

func (x *Language_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Language_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Language_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Search_Resp) Reset() {
	*x = Language_Search_Resp{}
	mi := &file_auth_proto_msgTypes[23]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Search_Resp) ProtoMessage() {}

func (x *Language_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[23]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Search_Resp.ProtoReflect.Descriptor instead.
func (*Language_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{23}
}

func (x *Language_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Language_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Update_Req) Reset() {
	*x = Language_Update_Req{}
	mi := &file_auth_proto_msgTypes[24]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Update_Req) ProtoMessage() {}

func (x *Language_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[24]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Update_Req.ProtoReflect.Descriptor instead.
func (*Language_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{24}
}

func (x *Language_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Language_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Language_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Language_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_Update_Resp) Reset() {
	*x = Language_Update_Resp{}
	mi := &file_auth_proto_msgTypes[25]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_Update_Resp) ProtoMessage() {}

func (x *Language_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[25]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_Update_Resp.ProtoReflect.Descriptor instead.
func (*Language_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{25}
}

func (x *Language_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Language_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_UpdateById_Req) Reset() {
	*x = Language_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[26]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_UpdateById_Req) ProtoMessage() {}

func (x *Language_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[26]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*Language_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{26}
}

func (x *Language_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Language_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Language_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Language_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Language_UpdateById_Resp) Reset() {
	*x = Language_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[27]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Language_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Language_UpdateById_Resp) ProtoMessage() {}

func (x *Language_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[27]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Language_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*Language_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{27}
}

func (x *Language_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Browse_Req) Reset() {
	*x = Location_Browse_Req{}
	mi := &file_auth_proto_msgTypes[28]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Browse_Req) ProtoMessage() {}

func (x *Location_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[28]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Browse_Req.ProtoReflect.Descriptor instead.
func (*Location_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{28}
}

func (x *Location_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Location_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Location_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Browse_Resp) Reset() {
	*x = Location_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[29]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Browse_Resp) ProtoMessage() {}

func (x *Location_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[29]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Browse_Resp.ProtoReflect.Descriptor instead.
func (*Location_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{29}
}

func (x *Location_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_BrowseMany_Req) Reset() {
	*x = Location_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[30]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_BrowseMany_Req) ProtoMessage() {}

func (x *Location_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[30]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*Location_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{30}
}

func (x *Location_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *Location_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Location_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_BrowseMany_Resp) Reset() {
	*x = Location_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[31]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_BrowseMany_Resp) ProtoMessage() {}

func (x *Location_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[31]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*Location_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{31}
}

func (x *Location_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Count_Req) Reset() {
	*x = Location_Count_Req{}
	mi := &file_auth_proto_msgTypes[32]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Count_Req) ProtoMessage() {}

func (x *Location_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[32]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Count_Req.ProtoReflect.Descriptor instead.
func (*Location_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{32}
}

func (x *Location_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Location_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Count_Resp) Reset() {
	*x = Location_Count_Resp{}
	mi := &file_auth_proto_msgTypes[33]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Count_Resp) ProtoMessage() {}

func (x *Location_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[33]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Count_Resp.ProtoReflect.Descriptor instead.
func (*Location_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{33}
}

func (x *Location_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Location_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Create_Req) Reset() {
	*x = Location_Create_Req{}
	mi := &file_auth_proto_msgTypes[34]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Create_Req) ProtoMessage() {}

func (x *Location_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[34]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Create_Req.ProtoReflect.Descriptor instead.
func (*Location_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{34}
}

func (x *Location_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *Location_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Location_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Create_Resp) Reset() {
	*x = Location_Create_Resp{}
	mi := &file_auth_proto_msgTypes[35]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Create_Resp) ProtoMessage() {}

func (x *Location_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[35]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Create_Resp.ProtoReflect.Descriptor instead.
func (*Location_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{35}
}

func (x *Location_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_CreateMany_Req) Reset() {
	*x = Location_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[36]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_CreateMany_Req) ProtoMessage() {}

func (x *Location_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[36]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*Location_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{36}
}

func (x *Location_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Location_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Location_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_CreateMany_Resp) Reset() {
	*x = Location_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[37]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_CreateMany_Resp) ProtoMessage() {}

func (x *Location_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[37]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*Location_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{37}
}

func (x *Location_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_DefaultGet_Req) Reset() {
	*x = Location_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[38]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_DefaultGet_Req) ProtoMessage() {}

func (x *Location_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[38]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*Location_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{38}
}

func (x *Location_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type Location_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_DefaultGet_Resp) Reset() {
	*x = Location_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[39]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_DefaultGet_Resp) ProtoMessage() {}

func (x *Location_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[39]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*Location_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{39}
}

func (x *Location_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Delete_Req) Reset() {
	*x = Location_Delete_Req{}
	mi := &file_auth_proto_msgTypes[40]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Delete_Req) ProtoMessage() {}

func (x *Location_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[40]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Delete_Req.ProtoReflect.Descriptor instead.
func (*Location_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{40}
}

func (x *Location_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Location_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Delete_Resp) Reset() {
	*x = Location_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[41]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Delete_Resp) ProtoMessage() {}

func (x *Location_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[41]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Delete_Resp.ProtoReflect.Descriptor instead.
func (*Location_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{41}
}

func (x *Location_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Location_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_DeleteById_Req) Reset() {
	*x = Location_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[42]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_DeleteById_Req) ProtoMessage() {}

func (x *Location_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[42]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*Location_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{42}
}

func (x *Location_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type Location_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_DeleteById_Resp) Reset() {
	*x = Location_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[43]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_DeleteById_Resp) ProtoMessage() {}

func (x *Location_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[43]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*Location_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{43}
}

func (x *Location_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Location_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Onchange_Req) Reset() {
	*x = Location_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[44]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Onchange_Req) ProtoMessage() {}

func (x *Location_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[44]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Onchange_Req.ProtoReflect.Descriptor instead.
func (*Location_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{44}
}

func (x *Location_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *Location_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *Location_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type Location_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Onchange_Resp) Reset() {
	*x = Location_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[45]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Onchange_Resp) ProtoMessage() {}

func (x *Location_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[45]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*Location_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{45}
}

func (x *Location_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_ReadGroup_Req) Reset() {
	*x = Location_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[46]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_ReadGroup_Req) ProtoMessage() {}

func (x *Location_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[46]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*Location_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{46}
}

func (x *Location_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Location_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Location_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Location_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_ReadGroup_Resp) Reset() {
	*x = Location_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[47]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_ReadGroup_Resp) ProtoMessage() {}

func (x *Location_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[47]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*Location_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{47}
}

func (x *Location_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_ReadGroupCount_Req) Reset() {
	*x = Location_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[48]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_ReadGroupCount_Req) ProtoMessage() {}

func (x *Location_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[48]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*Location_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{48}
}

func (x *Location_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Location_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Location_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Location_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_ReadGroupCount_Resp) Reset() {
	*x = Location_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[49]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_ReadGroupCount_Resp) ProtoMessage() {}

func (x *Location_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[49]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*Location_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{49}
}

func (x *Location_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Location_Register_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Register_Req) Reset() {
	*x = Location_Register_Req{}
	mi := &file_auth_proto_msgTypes[50]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Register_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Register_Req) ProtoMessage() {}

func (x *Location_Register_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[50]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Register_Req.ProtoReflect.Descriptor instead.
func (*Location_Register_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{50}
}

func (x *Location_Register_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type Location_Register_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Register_Resp) Reset() {
	*x = Location_Register_Resp{}
	mi := &file_auth_proto_msgTypes[51]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Register_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Register_Resp) ProtoMessage() {}

func (x *Location_Register_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[51]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Register_Resp.ProtoReflect.Descriptor instead.
func (*Location_Register_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{51}
}

func (x *Location_Register_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Search_Req) Reset() {
	*x = Location_Search_Req{}
	mi := &file_auth_proto_msgTypes[52]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Search_Req) ProtoMessage() {}

func (x *Location_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[52]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Search_Req.ProtoReflect.Descriptor instead.
func (*Location_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{52}
}

func (x *Location_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Location_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Location_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Search_Resp) Reset() {
	*x = Location_Search_Resp{}
	mi := &file_auth_proto_msgTypes[53]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Search_Resp) ProtoMessage() {}

func (x *Location_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[53]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Search_Resp.ProtoReflect.Descriptor instead.
func (*Location_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{53}
}

func (x *Location_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Update_Req) Reset() {
	*x = Location_Update_Req{}
	mi := &file_auth_proto_msgTypes[54]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Update_Req) ProtoMessage() {}

func (x *Location_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[54]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Update_Req.ProtoReflect.Descriptor instead.
func (*Location_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{54}
}

func (x *Location_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Location_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Location_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Location_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_Update_Resp) Reset() {
	*x = Location_Update_Resp{}
	mi := &file_auth_proto_msgTypes[55]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_Update_Resp) ProtoMessage() {}

func (x *Location_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[55]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_Update_Resp.ProtoReflect.Descriptor instead.
func (*Location_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{55}
}

func (x *Location_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Location_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_UpdateById_Req) Reset() {
	*x = Location_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[56]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_UpdateById_Req) ProtoMessage() {}

func (x *Location_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[56]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*Location_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{56}
}

func (x *Location_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Location_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Location_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Location_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location_UpdateById_Resp) Reset() {
	*x = Location_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[57]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location_UpdateById_Resp) ProtoMessage() {}

func (x *Location_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[57]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*Location_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{57}
}

func (x *Location_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Order_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Browse_Req) Reset() {
	*x = Order_Browse_Req{}
	mi := &file_auth_proto_msgTypes[58]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Browse_Req) ProtoMessage() {}

func (x *Order_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[58]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Browse_Req.ProtoReflect.Descriptor instead.
func (*Order_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{58}
}

func (x *Order_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Order_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Order_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Browse_Resp) Reset() {
	*x = Order_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[59]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Browse_Resp) ProtoMessage() {}

func (x *Order_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[59]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Browse_Resp.ProtoReflect.Descriptor instead.
func (*Order_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{59}
}

func (x *Order_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Order_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_BrowseMany_Req) Reset() {
	*x = Order_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[60]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_BrowseMany_Req) ProtoMessage() {}

func (x *Order_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[60]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*Order_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{60}
}

func (x *Order_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *Order_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Order_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_BrowseMany_Resp) Reset() {
	*x = Order_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[61]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_BrowseMany_Resp) ProtoMessage() {}

func (x *Order_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[61]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*Order_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{61}
}

func (x *Order_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Order_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Count_Req) Reset() {
	*x = Order_Count_Req{}
	mi := &file_auth_proto_msgTypes[62]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Count_Req) ProtoMessage() {}

func (x *Order_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[62]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Count_Req.ProtoReflect.Descriptor instead.
func (*Order_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{62}
}

func (x *Order_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Order_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Count_Resp) Reset() {
	*x = Order_Count_Resp{}
	mi := &file_auth_proto_msgTypes[63]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Count_Resp) ProtoMessage() {}

func (x *Order_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[63]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Count_Resp.ProtoReflect.Descriptor instead.
func (*Order_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{63}
}

func (x *Order_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Order_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Create_Req) Reset() {
	*x = Order_Create_Req{}
	mi := &file_auth_proto_msgTypes[64]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Create_Req) ProtoMessage() {}

func (x *Order_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[64]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Create_Req.ProtoReflect.Descriptor instead.
func (*Order_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{64}
}

func (x *Order_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *Order_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Order_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Create_Resp) Reset() {
	*x = Order_Create_Resp{}
	mi := &file_auth_proto_msgTypes[65]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Create_Resp) ProtoMessage() {}

func (x *Order_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[65]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Create_Resp.ProtoReflect.Descriptor instead.
func (*Order_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{65}
}

func (x *Order_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Order_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_CreateMany_Req) Reset() {
	*x = Order_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[66]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_CreateMany_Req) ProtoMessage() {}

func (x *Order_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[66]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*Order_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{66}
}

func (x *Order_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Order_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Order_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_CreateMany_Resp) Reset() {
	*x = Order_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[67]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_CreateMany_Resp) ProtoMessage() {}

func (x *Order_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[67]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*Order_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{67}
}

func (x *Order_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Order_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_DefaultGet_Req) Reset() {
	*x = Order_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[68]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_DefaultGet_Req) ProtoMessage() {}

func (x *Order_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[68]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*Order_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{68}
}

func (x *Order_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type Order_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_DefaultGet_Resp) Reset() {
	*x = Order_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[69]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_DefaultGet_Resp) ProtoMessage() {}

func (x *Order_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[69]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*Order_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{69}
}

func (x *Order_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Order_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Delete_Req) Reset() {
	*x = Order_Delete_Req{}
	mi := &file_auth_proto_msgTypes[70]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Delete_Req) ProtoMessage() {}

func (x *Order_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[70]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Delete_Req.ProtoReflect.Descriptor instead.
func (*Order_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{70}
}

func (x *Order_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Order_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Delete_Resp) Reset() {
	*x = Order_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[71]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Delete_Resp) ProtoMessage() {}

func (x *Order_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[71]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Delete_Resp.ProtoReflect.Descriptor instead.
func (*Order_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{71}
}

func (x *Order_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Order_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_DeleteById_Req) Reset() {
	*x = Order_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[72]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_DeleteById_Req) ProtoMessage() {}

func (x *Order_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[72]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*Order_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{72}
}

func (x *Order_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type Order_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_DeleteById_Resp) Reset() {
	*x = Order_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[73]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_DeleteById_Resp) ProtoMessage() {}

func (x *Order_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[73]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*Order_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{73}
}

func (x *Order_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Order_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Onchange_Req) Reset() {
	*x = Order_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[74]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Onchange_Req) ProtoMessage() {}

func (x *Order_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[74]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Onchange_Req.ProtoReflect.Descriptor instead.
func (*Order_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{74}
}

func (x *Order_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *Order_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *Order_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type Order_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Onchange_Resp) Reset() {
	*x = Order_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[75]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Onchange_Resp) ProtoMessage() {}

func (x *Order_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[75]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*Order_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{75}
}

func (x *Order_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Order_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_ReadGroup_Req) Reset() {
	*x = Order_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[76]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_ReadGroup_Req) ProtoMessage() {}

func (x *Order_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[76]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*Order_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{76}
}

func (x *Order_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Order_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Order_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Order_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_ReadGroup_Resp) Reset() {
	*x = Order_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[77]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_ReadGroup_Resp) ProtoMessage() {}

func (x *Order_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[77]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*Order_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{77}
}

func (x *Order_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Order_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_ReadGroupCount_Req) Reset() {
	*x = Order_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[78]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_ReadGroupCount_Req) ProtoMessage() {}

func (x *Order_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[78]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*Order_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{78}
}

func (x *Order_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Order_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Order_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Order_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_ReadGroupCount_Resp) Reset() {
	*x = Order_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[79]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_ReadGroupCount_Resp) ProtoMessage() {}

func (x *Order_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[79]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*Order_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{79}
}

func (x *Order_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Order_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Search_Req) Reset() {
	*x = Order_Search_Req{}
	mi := &file_auth_proto_msgTypes[80]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Search_Req) ProtoMessage() {}

func (x *Order_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[80]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Search_Req.ProtoReflect.Descriptor instead.
func (*Order_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{80}
}

func (x *Order_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Order_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Order_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Search_Resp) Reset() {
	*x = Order_Search_Resp{}
	mi := &file_auth_proto_msgTypes[81]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Search_Resp) ProtoMessage() {}

func (x *Order_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[81]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Search_Resp.ProtoReflect.Descriptor instead.
func (*Order_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{81}
}

func (x *Order_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Order_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Update_Req) Reset() {
	*x = Order_Update_Req{}
	mi := &file_auth_proto_msgTypes[82]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Update_Req) ProtoMessage() {}

func (x *Order_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[82]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Update_Req.ProtoReflect.Descriptor instead.
func (*Order_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{82}
}

func (x *Order_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Order_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Order_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Order_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_Update_Resp) Reset() {
	*x = Order_Update_Resp{}
	mi := &file_auth_proto_msgTypes[83]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_Update_Resp) ProtoMessage() {}

func (x *Order_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[83]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_Update_Resp.ProtoReflect.Descriptor instead.
func (*Order_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{83}
}

func (x *Order_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Order_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_UpdateById_Req) Reset() {
	*x = Order_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[84]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_UpdateById_Req) ProtoMessage() {}

func (x *Order_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[84]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*Order_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{84}
}

func (x *Order_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Order_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Order_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Order_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Order_UpdateById_Resp) Reset() {
	*x = Order_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[85]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Order_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order_UpdateById_Resp) ProtoMessage() {}

func (x *Order_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[85]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*Order_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{85}
}

func (x *Order_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type OrderLine_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Browse_Req) Reset() {
	*x = OrderLine_Browse_Req{}
	mi := &file_auth_proto_msgTypes[86]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Browse_Req) ProtoMessage() {}

func (x *OrderLine_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[86]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Browse_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{86}
}

func (x *OrderLine_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *OrderLine_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type OrderLine_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Browse_Resp) Reset() {
	*x = OrderLine_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[87]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Browse_Resp) ProtoMessage() {}

func (x *OrderLine_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[87]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Browse_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{87}
}

func (x *OrderLine_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type OrderLine_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_BrowseMany_Req) Reset() {
	*x = OrderLine_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[88]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_BrowseMany_Req) ProtoMessage() {}

func (x *OrderLine_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[88]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{88}
}

func (x *OrderLine_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *OrderLine_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type OrderLine_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_BrowseMany_Resp) Reset() {
	*x = OrderLine_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[89]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_BrowseMany_Resp) ProtoMessage() {}

func (x *OrderLine_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[89]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{89}
}

func (x *OrderLine_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type OrderLine_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Count_Req) Reset() {
	*x = OrderLine_Count_Req{}
	mi := &file_auth_proto_msgTypes[90]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Count_Req) ProtoMessage() {}

func (x *OrderLine_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[90]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Count_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{90}
}

func (x *OrderLine_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type OrderLine_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Count_Resp) Reset() {
	*x = OrderLine_Count_Resp{}
	mi := &file_auth_proto_msgTypes[91]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Count_Resp) ProtoMessage() {}

func (x *OrderLine_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[91]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Count_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{91}
}

func (x *OrderLine_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type OrderLine_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Create_Req) Reset() {
	*x = OrderLine_Create_Req{}
	mi := &file_auth_proto_msgTypes[92]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Create_Req) ProtoMessage() {}

func (x *OrderLine_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[92]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Create_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{92}
}

func (x *OrderLine_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *OrderLine_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type OrderLine_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Create_Resp) Reset() {
	*x = OrderLine_Create_Resp{}
	mi := &file_auth_proto_msgTypes[93]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Create_Resp) ProtoMessage() {}

func (x *OrderLine_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[93]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Create_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{93}
}

func (x *OrderLine_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type OrderLine_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_CreateMany_Req) Reset() {
	*x = OrderLine_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[94]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_CreateMany_Req) ProtoMessage() {}

func (x *OrderLine_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[94]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{94}
}

func (x *OrderLine_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *OrderLine_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type OrderLine_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_CreateMany_Resp) Reset() {
	*x = OrderLine_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[95]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_CreateMany_Resp) ProtoMessage() {}

func (x *OrderLine_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[95]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{95}
}

func (x *OrderLine_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type OrderLine_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_DefaultGet_Req) Reset() {
	*x = OrderLine_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[96]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_DefaultGet_Req) ProtoMessage() {}

func (x *OrderLine_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[96]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{96}
}

func (x *OrderLine_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type OrderLine_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_DefaultGet_Resp) Reset() {
	*x = OrderLine_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[97]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_DefaultGet_Resp) ProtoMessage() {}

func (x *OrderLine_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[97]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{97}
}

func (x *OrderLine_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type OrderLine_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Delete_Req) Reset() {
	*x = OrderLine_Delete_Req{}
	mi := &file_auth_proto_msgTypes[98]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Delete_Req) ProtoMessage() {}

func (x *OrderLine_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[98]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Delete_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{98}
}

func (x *OrderLine_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type OrderLine_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Delete_Resp) Reset() {
	*x = OrderLine_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[99]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Delete_Resp) ProtoMessage() {}

func (x *OrderLine_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[99]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Delete_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{99}
}

func (x *OrderLine_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type OrderLine_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_DeleteById_Req) Reset() {
	*x = OrderLine_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[100]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_DeleteById_Req) ProtoMessage() {}

func (x *OrderLine_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[100]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{100}
}

func (x *OrderLine_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type OrderLine_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_DeleteById_Resp) Reset() {
	*x = OrderLine_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[101]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_DeleteById_Resp) ProtoMessage() {}

func (x *OrderLine_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[101]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{101}
}

func (x *OrderLine_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type OrderLine_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Onchange_Req) Reset() {
	*x = OrderLine_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[102]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Onchange_Req) ProtoMessage() {}

func (x *OrderLine_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[102]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Onchange_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{102}
}

func (x *OrderLine_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *OrderLine_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *OrderLine_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type OrderLine_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Onchange_Resp) Reset() {
	*x = OrderLine_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[103]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Onchange_Resp) ProtoMessage() {}

func (x *OrderLine_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[103]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{103}
}

func (x *OrderLine_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type OrderLine_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_ReadGroup_Req) Reset() {
	*x = OrderLine_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[104]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_ReadGroup_Req) ProtoMessage() {}

func (x *OrderLine_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[104]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{104}
}

func (x *OrderLine_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *OrderLine_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *OrderLine_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type OrderLine_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_ReadGroup_Resp) Reset() {
	*x = OrderLine_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[105]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_ReadGroup_Resp) ProtoMessage() {}

func (x *OrderLine_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[105]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{105}
}

func (x *OrderLine_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type OrderLine_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_ReadGroupCount_Req) Reset() {
	*x = OrderLine_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[106]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_ReadGroupCount_Req) ProtoMessage() {}

func (x *OrderLine_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[106]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{106}
}

func (x *OrderLine_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *OrderLine_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *OrderLine_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type OrderLine_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_ReadGroupCount_Resp) Reset() {
	*x = OrderLine_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[107]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_ReadGroupCount_Resp) ProtoMessage() {}

func (x *OrderLine_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[107]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{107}
}

func (x *OrderLine_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type OrderLine_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Search_Req) Reset() {
	*x = OrderLine_Search_Req{}
	mi := &file_auth_proto_msgTypes[108]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Search_Req) ProtoMessage() {}

func (x *OrderLine_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[108]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Search_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{108}
}

func (x *OrderLine_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *OrderLine_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type OrderLine_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Search_Resp) Reset() {
	*x = OrderLine_Search_Resp{}
	mi := &file_auth_proto_msgTypes[109]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Search_Resp) ProtoMessage() {}

func (x *OrderLine_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[109]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Search_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{109}
}

func (x *OrderLine_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type OrderLine_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Update_Req) Reset() {
	*x = OrderLine_Update_Req{}
	mi := &file_auth_proto_msgTypes[110]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Update_Req) ProtoMessage() {}

func (x *OrderLine_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[110]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Update_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{110}
}

func (x *OrderLine_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *OrderLine_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *OrderLine_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type OrderLine_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_Update_Resp) Reset() {
	*x = OrderLine_Update_Resp{}
	mi := &file_auth_proto_msgTypes[111]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_Update_Resp) ProtoMessage() {}

func (x *OrderLine_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[111]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_Update_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{111}
}

func (x *OrderLine_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type OrderLine_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_UpdateById_Req) Reset() {
	*x = OrderLine_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[112]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_UpdateById_Req) ProtoMessage() {}

func (x *OrderLine_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[112]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*OrderLine_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{112}
}

func (x *OrderLine_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *OrderLine_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *OrderLine_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type OrderLine_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *OrderLine_UpdateById_Resp) Reset() {
	*x = OrderLine_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[113]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrderLine_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrderLine_UpdateById_Resp) ProtoMessage() {}

func (x *OrderLine_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[113]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrderLine_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*OrderLine_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{113}
}

func (x *OrderLine_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Role_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Browse_Req) Reset() {
	*x = Role_Browse_Req{}
	mi := &file_auth_proto_msgTypes[114]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Browse_Req) ProtoMessage() {}

func (x *Role_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[114]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Browse_Req.ProtoReflect.Descriptor instead.
func (*Role_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{114}
}

func (x *Role_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Role_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Role_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Browse_Resp) Reset() {
	*x = Role_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[115]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Browse_Resp) ProtoMessage() {}

func (x *Role_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[115]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Browse_Resp.ProtoReflect.Descriptor instead.
func (*Role_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{115}
}

func (x *Role_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Role_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_BrowseMany_Req) Reset() {
	*x = Role_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[116]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_BrowseMany_Req) ProtoMessage() {}

func (x *Role_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[116]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*Role_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{116}
}

func (x *Role_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *Role_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Role_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_BrowseMany_Resp) Reset() {
	*x = Role_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[117]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_BrowseMany_Resp) ProtoMessage() {}

func (x *Role_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[117]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*Role_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{117}
}

func (x *Role_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Role_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Count_Req) Reset() {
	*x = Role_Count_Req{}
	mi := &file_auth_proto_msgTypes[118]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Count_Req) ProtoMessage() {}

func (x *Role_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[118]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Count_Req.ProtoReflect.Descriptor instead.
func (*Role_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{118}
}

func (x *Role_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Role_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Count_Resp) Reset() {
	*x = Role_Count_Resp{}
	mi := &file_auth_proto_msgTypes[119]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Count_Resp) ProtoMessage() {}

func (x *Role_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[119]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Count_Resp.ProtoReflect.Descriptor instead.
func (*Role_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{119}
}

func (x *Role_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Role_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Create_Req) Reset() {
	*x = Role_Create_Req{}
	mi := &file_auth_proto_msgTypes[120]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Create_Req) ProtoMessage() {}

func (x *Role_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[120]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Create_Req.ProtoReflect.Descriptor instead.
func (*Role_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{120}
}

func (x *Role_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *Role_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Role_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Create_Resp) Reset() {
	*x = Role_Create_Resp{}
	mi := &file_auth_proto_msgTypes[121]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Create_Resp) ProtoMessage() {}

func (x *Role_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[121]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Create_Resp.ProtoReflect.Descriptor instead.
func (*Role_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{121}
}

func (x *Role_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Role_CreateIfNotExists_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	RoleData      *structpb.Value        `protobuf:"bytes,1,opt,name=roleData,proto3" json:"roleData,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_CreateIfNotExists_Req) Reset() {
	*x = Role_CreateIfNotExists_Req{}
	mi := &file_auth_proto_msgTypes[122]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_CreateIfNotExists_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_CreateIfNotExists_Req) ProtoMessage() {}

func (x *Role_CreateIfNotExists_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[122]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_CreateIfNotExists_Req.ProtoReflect.Descriptor instead.
func (*Role_CreateIfNotExists_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{122}
}

func (x *Role_CreateIfNotExists_Req) GetRoleData() *structpb.Value {
	if x != nil {
		return x.RoleData
	}
	return nil
}

type Role_CreateIfNotExists_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        string                 `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_CreateIfNotExists_Resp) Reset() {
	*x = Role_CreateIfNotExists_Resp{}
	mi := &file_auth_proto_msgTypes[123]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_CreateIfNotExists_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_CreateIfNotExists_Resp) ProtoMessage() {}

func (x *Role_CreateIfNotExists_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[123]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_CreateIfNotExists_Resp.ProtoReflect.Descriptor instead.
func (*Role_CreateIfNotExists_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{123}
}

func (x *Role_CreateIfNotExists_Resp) GetResult() string {
	if x != nil {
		return x.Result
	}
	return ""
}

type Role_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_CreateMany_Req) Reset() {
	*x = Role_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[124]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_CreateMany_Req) ProtoMessage() {}

func (x *Role_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[124]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*Role_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{124}
}

func (x *Role_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Role_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Role_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_CreateMany_Resp) Reset() {
	*x = Role_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[125]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_CreateMany_Resp) ProtoMessage() {}

func (x *Role_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[125]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*Role_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{125}
}

func (x *Role_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Role_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_DefaultGet_Req) Reset() {
	*x = Role_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[126]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_DefaultGet_Req) ProtoMessage() {}

func (x *Role_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[126]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*Role_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{126}
}

func (x *Role_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type Role_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_DefaultGet_Resp) Reset() {
	*x = Role_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[127]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_DefaultGet_Resp) ProtoMessage() {}

func (x *Role_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[127]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*Role_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{127}
}

func (x *Role_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Role_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Delete_Req) Reset() {
	*x = Role_Delete_Req{}
	mi := &file_auth_proto_msgTypes[128]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Delete_Req) ProtoMessage() {}

func (x *Role_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[128]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Delete_Req.ProtoReflect.Descriptor instead.
func (*Role_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{128}
}

func (x *Role_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Role_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Delete_Resp) Reset() {
	*x = Role_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[129]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Delete_Resp) ProtoMessage() {}

func (x *Role_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[129]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Delete_Resp.ProtoReflect.Descriptor instead.
func (*Role_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{129}
}

func (x *Role_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Role_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_DeleteById_Req) Reset() {
	*x = Role_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[130]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_DeleteById_Req) ProtoMessage() {}

func (x *Role_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[130]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*Role_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{130}
}

func (x *Role_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type Role_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_DeleteById_Resp) Reset() {
	*x = Role_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[131]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_DeleteById_Resp) ProtoMessage() {}

func (x *Role_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[131]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*Role_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{131}
}

func (x *Role_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Role_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Onchange_Req) Reset() {
	*x = Role_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[132]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Onchange_Req) ProtoMessage() {}

func (x *Role_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[132]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Onchange_Req.ProtoReflect.Descriptor instead.
func (*Role_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{132}
}

func (x *Role_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *Role_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *Role_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type Role_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Onchange_Resp) Reset() {
	*x = Role_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[133]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Onchange_Resp) ProtoMessage() {}

func (x *Role_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[133]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*Role_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{133}
}

func (x *Role_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Role_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_ReadGroup_Req) Reset() {
	*x = Role_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[134]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_ReadGroup_Req) ProtoMessage() {}

func (x *Role_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[134]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*Role_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{134}
}

func (x *Role_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Role_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Role_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Role_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_ReadGroup_Resp) Reset() {
	*x = Role_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[135]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_ReadGroup_Resp) ProtoMessage() {}

func (x *Role_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[135]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*Role_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{135}
}

func (x *Role_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Role_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_ReadGroupCount_Req) Reset() {
	*x = Role_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[136]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_ReadGroupCount_Req) ProtoMessage() {}

func (x *Role_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[136]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*Role_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{136}
}

func (x *Role_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Role_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Role_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Role_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_ReadGroupCount_Resp) Reset() {
	*x = Role_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[137]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_ReadGroupCount_Resp) ProtoMessage() {}

func (x *Role_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[137]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*Role_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{137}
}

func (x *Role_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Role_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Search_Req) Reset() {
	*x = Role_Search_Req{}
	mi := &file_auth_proto_msgTypes[138]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Search_Req) ProtoMessage() {}

func (x *Role_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[138]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Search_Req.ProtoReflect.Descriptor instead.
func (*Role_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{138}
}

func (x *Role_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Role_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Role_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Search_Resp) Reset() {
	*x = Role_Search_Resp{}
	mi := &file_auth_proto_msgTypes[139]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Search_Resp) ProtoMessage() {}

func (x *Role_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[139]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Search_Resp.ProtoReflect.Descriptor instead.
func (*Role_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{139}
}

func (x *Role_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Role_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Update_Req) Reset() {
	*x = Role_Update_Req{}
	mi := &file_auth_proto_msgTypes[140]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Update_Req) ProtoMessage() {}

func (x *Role_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[140]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Update_Req.ProtoReflect.Descriptor instead.
func (*Role_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{140}
}

func (x *Role_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Role_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Role_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Role_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_Update_Resp) Reset() {
	*x = Role_Update_Resp{}
	mi := &file_auth_proto_msgTypes[141]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_Update_Resp) ProtoMessage() {}

func (x *Role_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[141]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_Update_Resp.ProtoReflect.Descriptor instead.
func (*Role_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{141}
}

func (x *Role_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Role_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_UpdateById_Req) Reset() {
	*x = Role_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[142]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_UpdateById_Req) ProtoMessage() {}

func (x *Role_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[142]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*Role_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{142}
}

func (x *Role_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Role_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Role_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Role_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Role_UpdateById_Resp) Reset() {
	*x = Role_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[143]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Role_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Role_UpdateById_Resp) ProtoMessage() {}

func (x *Role_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[143]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Role_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*Role_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{143}
}

func (x *Role_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleFieldRule_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Browse_Req) Reset() {
	*x = RoleFieldRule_Browse_Req{}
	mi := &file_auth_proto_msgTypes[144]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Browse_Req) ProtoMessage() {}

func (x *RoleFieldRule_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[144]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Browse_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{144}
}

func (x *RoleFieldRule_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *RoleFieldRule_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type RoleFieldRule_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Browse_Resp) Reset() {
	*x = RoleFieldRule_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[145]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Browse_Resp) ProtoMessage() {}

func (x *RoleFieldRule_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[145]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Browse_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{145}
}

func (x *RoleFieldRule_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleFieldRule_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_BrowseMany_Req) Reset() {
	*x = RoleFieldRule_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[146]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_BrowseMany_Req) ProtoMessage() {}

func (x *RoleFieldRule_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[146]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{146}
}

func (x *RoleFieldRule_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *RoleFieldRule_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type RoleFieldRule_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_BrowseMany_Resp) Reset() {
	*x = RoleFieldRule_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[147]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_BrowseMany_Resp) ProtoMessage() {}

func (x *RoleFieldRule_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[147]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{147}
}

func (x *RoleFieldRule_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleFieldRule_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Count_Req) Reset() {
	*x = RoleFieldRule_Count_Req{}
	mi := &file_auth_proto_msgTypes[148]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Count_Req) ProtoMessage() {}

func (x *RoleFieldRule_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[148]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Count_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{148}
}

func (x *RoleFieldRule_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type RoleFieldRule_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Count_Resp) Reset() {
	*x = RoleFieldRule_Count_Resp{}
	mi := &file_auth_proto_msgTypes[149]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Count_Resp) ProtoMessage() {}

func (x *RoleFieldRule_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[149]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Count_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{149}
}

func (x *RoleFieldRule_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleFieldRule_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Create_Req) Reset() {
	*x = RoleFieldRule_Create_Req{}
	mi := &file_auth_proto_msgTypes[150]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Create_Req) ProtoMessage() {}

func (x *RoleFieldRule_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[150]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Create_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{150}
}

func (x *RoleFieldRule_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *RoleFieldRule_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleFieldRule_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Create_Resp) Reset() {
	*x = RoleFieldRule_Create_Resp{}
	mi := &file_auth_proto_msgTypes[151]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Create_Resp) ProtoMessage() {}

func (x *RoleFieldRule_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[151]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Create_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{151}
}

func (x *RoleFieldRule_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleFieldRule_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_CreateMany_Req) Reset() {
	*x = RoleFieldRule_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[152]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_CreateMany_Req) ProtoMessage() {}

func (x *RoleFieldRule_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[152]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{152}
}

func (x *RoleFieldRule_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleFieldRule_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleFieldRule_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_CreateMany_Resp) Reset() {
	*x = RoleFieldRule_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[153]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_CreateMany_Resp) ProtoMessage() {}

func (x *RoleFieldRule_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[153]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{153}
}

func (x *RoleFieldRule_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleFieldRule_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_DefaultGet_Req) Reset() {
	*x = RoleFieldRule_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[154]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_DefaultGet_Req) ProtoMessage() {}

func (x *RoleFieldRule_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[154]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{154}
}

func (x *RoleFieldRule_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type RoleFieldRule_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_DefaultGet_Resp) Reset() {
	*x = RoleFieldRule_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[155]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_DefaultGet_Resp) ProtoMessage() {}

func (x *RoleFieldRule_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[155]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{155}
}

func (x *RoleFieldRule_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleFieldRule_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Delete_Req) Reset() {
	*x = RoleFieldRule_Delete_Req{}
	mi := &file_auth_proto_msgTypes[156]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Delete_Req) ProtoMessage() {}

func (x *RoleFieldRule_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[156]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Delete_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{156}
}

func (x *RoleFieldRule_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type RoleFieldRule_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Delete_Resp) Reset() {
	*x = RoleFieldRule_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[157]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Delete_Resp) ProtoMessage() {}

func (x *RoleFieldRule_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[157]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Delete_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{157}
}

func (x *RoleFieldRule_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleFieldRule_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_DeleteById_Req) Reset() {
	*x = RoleFieldRule_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[158]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_DeleteById_Req) ProtoMessage() {}

func (x *RoleFieldRule_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[158]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{158}
}

func (x *RoleFieldRule_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type RoleFieldRule_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_DeleteById_Resp) Reset() {
	*x = RoleFieldRule_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[159]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_DeleteById_Resp) ProtoMessage() {}

func (x *RoleFieldRule_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[159]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{159}
}

func (x *RoleFieldRule_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleFieldRule_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Onchange_Req) Reset() {
	*x = RoleFieldRule_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[160]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Onchange_Req) ProtoMessage() {}

func (x *RoleFieldRule_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[160]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Onchange_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{160}
}

func (x *RoleFieldRule_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *RoleFieldRule_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *RoleFieldRule_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type RoleFieldRule_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Onchange_Resp) Reset() {
	*x = RoleFieldRule_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[161]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Onchange_Resp) ProtoMessage() {}

func (x *RoleFieldRule_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[161]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{161}
}

func (x *RoleFieldRule_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleFieldRule_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_ReadGroup_Req) Reset() {
	*x = RoleFieldRule_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[162]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_ReadGroup_Req) ProtoMessage() {}

func (x *RoleFieldRule_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[162]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{162}
}

func (x *RoleFieldRule_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *RoleFieldRule_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleFieldRule_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleFieldRule_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_ReadGroup_Resp) Reset() {
	*x = RoleFieldRule_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[163]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_ReadGroup_Resp) ProtoMessage() {}

func (x *RoleFieldRule_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[163]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{163}
}

func (x *RoleFieldRule_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleFieldRule_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_ReadGroupCount_Req) Reset() {
	*x = RoleFieldRule_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[164]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_ReadGroupCount_Req) ProtoMessage() {}

func (x *RoleFieldRule_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[164]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{164}
}

func (x *RoleFieldRule_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *RoleFieldRule_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleFieldRule_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleFieldRule_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_ReadGroupCount_Resp) Reset() {
	*x = RoleFieldRule_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[165]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_ReadGroupCount_Resp) ProtoMessage() {}

func (x *RoleFieldRule_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[165]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{165}
}

func (x *RoleFieldRule_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleFieldRule_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Search_Req) Reset() {
	*x = RoleFieldRule_Search_Req{}
	mi := &file_auth_proto_msgTypes[166]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Search_Req) ProtoMessage() {}

func (x *RoleFieldRule_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[166]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Search_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{166}
}

func (x *RoleFieldRule_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleFieldRule_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleFieldRule_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Search_Resp) Reset() {
	*x = RoleFieldRule_Search_Resp{}
	mi := &file_auth_proto_msgTypes[167]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Search_Resp) ProtoMessage() {}

func (x *RoleFieldRule_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[167]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Search_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{167}
}

func (x *RoleFieldRule_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleFieldRule_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Update_Req) Reset() {
	*x = RoleFieldRule_Update_Req{}
	mi := &file_auth_proto_msgTypes[168]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Update_Req) ProtoMessage() {}

func (x *RoleFieldRule_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[168]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Update_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{168}
}

func (x *RoleFieldRule_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleFieldRule_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleFieldRule_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleFieldRule_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_Update_Resp) Reset() {
	*x = RoleFieldRule_Update_Resp{}
	mi := &file_auth_proto_msgTypes[169]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_Update_Resp) ProtoMessage() {}

func (x *RoleFieldRule_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[169]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_Update_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{169}
}

func (x *RoleFieldRule_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleFieldRule_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_UpdateById_Req) Reset() {
	*x = RoleFieldRule_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[170]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_UpdateById_Req) ProtoMessage() {}

func (x *RoleFieldRule_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[170]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{170}
}

func (x *RoleFieldRule_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *RoleFieldRule_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleFieldRule_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleFieldRule_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleFieldRule_UpdateById_Resp) Reset() {
	*x = RoleFieldRule_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[171]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleFieldRule_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleFieldRule_UpdateById_Resp) ProtoMessage() {}

func (x *RoleFieldRule_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[171]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleFieldRule_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*RoleFieldRule_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{171}
}

func (x *RoleFieldRule_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleInheritance_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Browse_Req) Reset() {
	*x = RoleInheritance_Browse_Req{}
	mi := &file_auth_proto_msgTypes[172]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Browse_Req) ProtoMessage() {}

func (x *RoleInheritance_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[172]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Browse_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{172}
}

func (x *RoleInheritance_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *RoleInheritance_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type RoleInheritance_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Browse_Resp) Reset() {
	*x = RoleInheritance_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[173]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Browse_Resp) ProtoMessage() {}

func (x *RoleInheritance_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[173]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Browse_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{173}
}

func (x *RoleInheritance_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleInheritance_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_BrowseMany_Req) Reset() {
	*x = RoleInheritance_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[174]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_BrowseMany_Req) ProtoMessage() {}

func (x *RoleInheritance_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[174]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{174}
}

func (x *RoleInheritance_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *RoleInheritance_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type RoleInheritance_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_BrowseMany_Resp) Reset() {
	*x = RoleInheritance_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[175]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_BrowseMany_Resp) ProtoMessage() {}

func (x *RoleInheritance_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[175]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{175}
}

func (x *RoleInheritance_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleInheritance_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Count_Req) Reset() {
	*x = RoleInheritance_Count_Req{}
	mi := &file_auth_proto_msgTypes[176]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Count_Req) ProtoMessage() {}

func (x *RoleInheritance_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[176]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Count_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{176}
}

func (x *RoleInheritance_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type RoleInheritance_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Count_Resp) Reset() {
	*x = RoleInheritance_Count_Resp{}
	mi := &file_auth_proto_msgTypes[177]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Count_Resp) ProtoMessage() {}

func (x *RoleInheritance_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[177]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Count_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{177}
}

func (x *RoleInheritance_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleInheritance_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Create_Req) Reset() {
	*x = RoleInheritance_Create_Req{}
	mi := &file_auth_proto_msgTypes[178]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Create_Req) ProtoMessage() {}

func (x *RoleInheritance_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[178]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Create_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{178}
}

func (x *RoleInheritance_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *RoleInheritance_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleInheritance_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Create_Resp) Reset() {
	*x = RoleInheritance_Create_Resp{}
	mi := &file_auth_proto_msgTypes[179]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Create_Resp) ProtoMessage() {}

func (x *RoleInheritance_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[179]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Create_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{179}
}

func (x *RoleInheritance_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleInheritance_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_CreateMany_Req) Reset() {
	*x = RoleInheritance_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[180]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_CreateMany_Req) ProtoMessage() {}

func (x *RoleInheritance_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[180]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{180}
}

func (x *RoleInheritance_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleInheritance_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleInheritance_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_CreateMany_Resp) Reset() {
	*x = RoleInheritance_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[181]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_CreateMany_Resp) ProtoMessage() {}

func (x *RoleInheritance_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[181]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{181}
}

func (x *RoleInheritance_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleInheritance_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_DefaultGet_Req) Reset() {
	*x = RoleInheritance_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[182]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_DefaultGet_Req) ProtoMessage() {}

func (x *RoleInheritance_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[182]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{182}
}

func (x *RoleInheritance_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type RoleInheritance_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_DefaultGet_Resp) Reset() {
	*x = RoleInheritance_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[183]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_DefaultGet_Resp) ProtoMessage() {}

func (x *RoleInheritance_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[183]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{183}
}

func (x *RoleInheritance_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleInheritance_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Delete_Req) Reset() {
	*x = RoleInheritance_Delete_Req{}
	mi := &file_auth_proto_msgTypes[184]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Delete_Req) ProtoMessage() {}

func (x *RoleInheritance_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[184]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Delete_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{184}
}

func (x *RoleInheritance_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type RoleInheritance_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Delete_Resp) Reset() {
	*x = RoleInheritance_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[185]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Delete_Resp) ProtoMessage() {}

func (x *RoleInheritance_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[185]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Delete_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{185}
}

func (x *RoleInheritance_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleInheritance_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_DeleteById_Req) Reset() {
	*x = RoleInheritance_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[186]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_DeleteById_Req) ProtoMessage() {}

func (x *RoleInheritance_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[186]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{186}
}

func (x *RoleInheritance_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type RoleInheritance_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_DeleteById_Resp) Reset() {
	*x = RoleInheritance_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[187]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_DeleteById_Resp) ProtoMessage() {}

func (x *RoleInheritance_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[187]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{187}
}

func (x *RoleInheritance_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleInheritance_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Onchange_Req) Reset() {
	*x = RoleInheritance_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[188]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Onchange_Req) ProtoMessage() {}

func (x *RoleInheritance_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[188]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Onchange_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{188}
}

func (x *RoleInheritance_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *RoleInheritance_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *RoleInheritance_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type RoleInheritance_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Onchange_Resp) Reset() {
	*x = RoleInheritance_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[189]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Onchange_Resp) ProtoMessage() {}

func (x *RoleInheritance_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[189]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{189}
}

func (x *RoleInheritance_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleInheritance_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_ReadGroup_Req) Reset() {
	*x = RoleInheritance_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[190]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_ReadGroup_Req) ProtoMessage() {}

func (x *RoleInheritance_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[190]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{190}
}

func (x *RoleInheritance_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *RoleInheritance_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleInheritance_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleInheritance_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_ReadGroup_Resp) Reset() {
	*x = RoleInheritance_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[191]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_ReadGroup_Resp) ProtoMessage() {}

func (x *RoleInheritance_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[191]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{191}
}

func (x *RoleInheritance_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleInheritance_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_ReadGroupCount_Req) Reset() {
	*x = RoleInheritance_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[192]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_ReadGroupCount_Req) ProtoMessage() {}

func (x *RoleInheritance_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[192]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{192}
}

func (x *RoleInheritance_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *RoleInheritance_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleInheritance_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleInheritance_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_ReadGroupCount_Resp) Reset() {
	*x = RoleInheritance_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[193]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_ReadGroupCount_Resp) ProtoMessage() {}

func (x *RoleInheritance_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[193]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{193}
}

func (x *RoleInheritance_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleInheritance_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Search_Req) Reset() {
	*x = RoleInheritance_Search_Req{}
	mi := &file_auth_proto_msgTypes[194]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Search_Req) ProtoMessage() {}

func (x *RoleInheritance_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[194]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Search_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{194}
}

func (x *RoleInheritance_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleInheritance_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleInheritance_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Search_Resp) Reset() {
	*x = RoleInheritance_Search_Resp{}
	mi := &file_auth_proto_msgTypes[195]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Search_Resp) ProtoMessage() {}

func (x *RoleInheritance_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[195]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Search_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{195}
}

func (x *RoleInheritance_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleInheritance_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Update_Req) Reset() {
	*x = RoleInheritance_Update_Req{}
	mi := &file_auth_proto_msgTypes[196]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Update_Req) ProtoMessage() {}

func (x *RoleInheritance_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[196]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Update_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{196}
}

func (x *RoleInheritance_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleInheritance_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleInheritance_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleInheritance_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_Update_Resp) Reset() {
	*x = RoleInheritance_Update_Resp{}
	mi := &file_auth_proto_msgTypes[197]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_Update_Resp) ProtoMessage() {}

func (x *RoleInheritance_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[197]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_Update_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{197}
}

func (x *RoleInheritance_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleInheritance_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_UpdateById_Req) Reset() {
	*x = RoleInheritance_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[198]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_UpdateById_Req) ProtoMessage() {}

func (x *RoleInheritance_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[198]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*RoleInheritance_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{198}
}

func (x *RoleInheritance_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *RoleInheritance_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleInheritance_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleInheritance_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleInheritance_UpdateById_Resp) Reset() {
	*x = RoleInheritance_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[199]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleInheritance_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleInheritance_UpdateById_Resp) ProtoMessage() {}

func (x *RoleInheritance_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[199]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleInheritance_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*RoleInheritance_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{199}
}

func (x *RoleInheritance_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleMethodAccess_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Browse_Req) Reset() {
	*x = RoleMethodAccess_Browse_Req{}
	mi := &file_auth_proto_msgTypes[200]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Browse_Req) ProtoMessage() {}

func (x *RoleMethodAccess_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[200]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Browse_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{200}
}

func (x *RoleMethodAccess_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *RoleMethodAccess_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type RoleMethodAccess_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Browse_Resp) Reset() {
	*x = RoleMethodAccess_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[201]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Browse_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[201]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Browse_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{201}
}

func (x *RoleMethodAccess_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleMethodAccess_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_BrowseMany_Req) Reset() {
	*x = RoleMethodAccess_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[202]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_BrowseMany_Req) ProtoMessage() {}

func (x *RoleMethodAccess_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[202]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{202}
}

func (x *RoleMethodAccess_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *RoleMethodAccess_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type RoleMethodAccess_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_BrowseMany_Resp) Reset() {
	*x = RoleMethodAccess_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[203]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_BrowseMany_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[203]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{203}
}

func (x *RoleMethodAccess_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleMethodAccess_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Count_Req) Reset() {
	*x = RoleMethodAccess_Count_Req{}
	mi := &file_auth_proto_msgTypes[204]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Count_Req) ProtoMessage() {}

func (x *RoleMethodAccess_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[204]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Count_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{204}
}

func (x *RoleMethodAccess_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type RoleMethodAccess_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Count_Resp) Reset() {
	*x = RoleMethodAccess_Count_Resp{}
	mi := &file_auth_proto_msgTypes[205]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Count_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[205]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Count_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{205}
}

func (x *RoleMethodAccess_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleMethodAccess_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Create_Req) Reset() {
	*x = RoleMethodAccess_Create_Req{}
	mi := &file_auth_proto_msgTypes[206]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Create_Req) ProtoMessage() {}

func (x *RoleMethodAccess_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[206]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Create_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{206}
}

func (x *RoleMethodAccess_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *RoleMethodAccess_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleMethodAccess_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Create_Resp) Reset() {
	*x = RoleMethodAccess_Create_Resp{}
	mi := &file_auth_proto_msgTypes[207]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Create_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[207]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Create_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{207}
}

func (x *RoleMethodAccess_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleMethodAccess_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_CreateMany_Req) Reset() {
	*x = RoleMethodAccess_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[208]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_CreateMany_Req) ProtoMessage() {}

func (x *RoleMethodAccess_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[208]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{208}
}

func (x *RoleMethodAccess_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleMethodAccess_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleMethodAccess_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_CreateMany_Resp) Reset() {
	*x = RoleMethodAccess_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[209]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_CreateMany_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[209]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{209}
}

func (x *RoleMethodAccess_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleMethodAccess_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_DefaultGet_Req) Reset() {
	*x = RoleMethodAccess_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[210]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_DefaultGet_Req) ProtoMessage() {}

func (x *RoleMethodAccess_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[210]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{210}
}

func (x *RoleMethodAccess_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type RoleMethodAccess_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_DefaultGet_Resp) Reset() {
	*x = RoleMethodAccess_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[211]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_DefaultGet_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[211]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{211}
}

func (x *RoleMethodAccess_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleMethodAccess_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Delete_Req) Reset() {
	*x = RoleMethodAccess_Delete_Req{}
	mi := &file_auth_proto_msgTypes[212]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Delete_Req) ProtoMessage() {}

func (x *RoleMethodAccess_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[212]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Delete_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{212}
}

func (x *RoleMethodAccess_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type RoleMethodAccess_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Delete_Resp) Reset() {
	*x = RoleMethodAccess_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[213]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Delete_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[213]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Delete_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{213}
}

func (x *RoleMethodAccess_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleMethodAccess_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_DeleteById_Req) Reset() {
	*x = RoleMethodAccess_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[214]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_DeleteById_Req) ProtoMessage() {}

func (x *RoleMethodAccess_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[214]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{214}
}

func (x *RoleMethodAccess_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type RoleMethodAccess_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_DeleteById_Resp) Reset() {
	*x = RoleMethodAccess_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[215]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_DeleteById_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[215]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{215}
}

func (x *RoleMethodAccess_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleMethodAccess_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Onchange_Req) Reset() {
	*x = RoleMethodAccess_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[216]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Onchange_Req) ProtoMessage() {}

func (x *RoleMethodAccess_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[216]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Onchange_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{216}
}

func (x *RoleMethodAccess_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *RoleMethodAccess_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *RoleMethodAccess_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type RoleMethodAccess_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Onchange_Resp) Reset() {
	*x = RoleMethodAccess_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[217]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Onchange_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[217]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{217}
}

func (x *RoleMethodAccess_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleMethodAccess_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_ReadGroup_Req) Reset() {
	*x = RoleMethodAccess_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[218]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_ReadGroup_Req) ProtoMessage() {}

func (x *RoleMethodAccess_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[218]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{218}
}

func (x *RoleMethodAccess_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *RoleMethodAccess_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleMethodAccess_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleMethodAccess_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_ReadGroup_Resp) Reset() {
	*x = RoleMethodAccess_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[219]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_ReadGroup_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[219]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{219}
}

func (x *RoleMethodAccess_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleMethodAccess_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_ReadGroupCount_Req) Reset() {
	*x = RoleMethodAccess_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[220]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_ReadGroupCount_Req) ProtoMessage() {}

func (x *RoleMethodAccess_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[220]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{220}
}

func (x *RoleMethodAccess_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *RoleMethodAccess_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleMethodAccess_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleMethodAccess_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_ReadGroupCount_Resp) Reset() {
	*x = RoleMethodAccess_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[221]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_ReadGroupCount_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[221]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{221}
}

func (x *RoleMethodAccess_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleMethodAccess_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Search_Req) Reset() {
	*x = RoleMethodAccess_Search_Req{}
	mi := &file_auth_proto_msgTypes[222]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Search_Req) ProtoMessage() {}

func (x *RoleMethodAccess_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[222]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Search_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{222}
}

func (x *RoleMethodAccess_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleMethodAccess_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleMethodAccess_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Search_Resp) Reset() {
	*x = RoleMethodAccess_Search_Resp{}
	mi := &file_auth_proto_msgTypes[223]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Search_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[223]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Search_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{223}
}

func (x *RoleMethodAccess_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleMethodAccess_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Update_Req) Reset() {
	*x = RoleMethodAccess_Update_Req{}
	mi := &file_auth_proto_msgTypes[224]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Update_Req) ProtoMessage() {}

func (x *RoleMethodAccess_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[224]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Update_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{224}
}

func (x *RoleMethodAccess_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleMethodAccess_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleMethodAccess_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleMethodAccess_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_Update_Resp) Reset() {
	*x = RoleMethodAccess_Update_Resp{}
	mi := &file_auth_proto_msgTypes[225]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_Update_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[225]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_Update_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{225}
}

func (x *RoleMethodAccess_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleMethodAccess_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_UpdateById_Req) Reset() {
	*x = RoleMethodAccess_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[226]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_UpdateById_Req) ProtoMessage() {}

func (x *RoleMethodAccess_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[226]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{226}
}

func (x *RoleMethodAccess_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *RoleMethodAccess_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleMethodAccess_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleMethodAccess_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleMethodAccess_UpdateById_Resp) Reset() {
	*x = RoleMethodAccess_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[227]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleMethodAccess_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleMethodAccess_UpdateById_Resp) ProtoMessage() {}

func (x *RoleMethodAccess_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[227]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleMethodAccess_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*RoleMethodAccess_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{227}
}

func (x *RoleMethodAccess_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleRecordRule_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Browse_Req) Reset() {
	*x = RoleRecordRule_Browse_Req{}
	mi := &file_auth_proto_msgTypes[228]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Browse_Req) ProtoMessage() {}

func (x *RoleRecordRule_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[228]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Browse_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{228}
}

func (x *RoleRecordRule_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *RoleRecordRule_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type RoleRecordRule_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Browse_Resp) Reset() {
	*x = RoleRecordRule_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[229]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Browse_Resp) ProtoMessage() {}

func (x *RoleRecordRule_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[229]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Browse_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{229}
}

func (x *RoleRecordRule_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleRecordRule_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_BrowseMany_Req) Reset() {
	*x = RoleRecordRule_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[230]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_BrowseMany_Req) ProtoMessage() {}

func (x *RoleRecordRule_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[230]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{230}
}

func (x *RoleRecordRule_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *RoleRecordRule_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type RoleRecordRule_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_BrowseMany_Resp) Reset() {
	*x = RoleRecordRule_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[231]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_BrowseMany_Resp) ProtoMessage() {}

func (x *RoleRecordRule_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[231]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{231}
}

func (x *RoleRecordRule_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleRecordRule_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Count_Req) Reset() {
	*x = RoleRecordRule_Count_Req{}
	mi := &file_auth_proto_msgTypes[232]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Count_Req) ProtoMessage() {}

func (x *RoleRecordRule_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[232]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Count_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{232}
}

func (x *RoleRecordRule_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type RoleRecordRule_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Count_Resp) Reset() {
	*x = RoleRecordRule_Count_Resp{}
	mi := &file_auth_proto_msgTypes[233]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Count_Resp) ProtoMessage() {}

func (x *RoleRecordRule_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[233]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Count_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{233}
}

func (x *RoleRecordRule_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleRecordRule_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Create_Req) Reset() {
	*x = RoleRecordRule_Create_Req{}
	mi := &file_auth_proto_msgTypes[234]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Create_Req) ProtoMessage() {}

func (x *RoleRecordRule_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[234]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Create_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{234}
}

func (x *RoleRecordRule_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *RoleRecordRule_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleRecordRule_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Create_Resp) Reset() {
	*x = RoleRecordRule_Create_Resp{}
	mi := &file_auth_proto_msgTypes[235]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Create_Resp) ProtoMessage() {}

func (x *RoleRecordRule_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[235]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Create_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{235}
}

func (x *RoleRecordRule_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleRecordRule_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_CreateMany_Req) Reset() {
	*x = RoleRecordRule_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[236]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_CreateMany_Req) ProtoMessage() {}

func (x *RoleRecordRule_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[236]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{236}
}

func (x *RoleRecordRule_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleRecordRule_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleRecordRule_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_CreateMany_Resp) Reset() {
	*x = RoleRecordRule_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[237]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_CreateMany_Resp) ProtoMessage() {}

func (x *RoleRecordRule_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[237]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{237}
}

func (x *RoleRecordRule_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleRecordRule_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_DefaultGet_Req) Reset() {
	*x = RoleRecordRule_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[238]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_DefaultGet_Req) ProtoMessage() {}

func (x *RoleRecordRule_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[238]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{238}
}

func (x *RoleRecordRule_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type RoleRecordRule_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_DefaultGet_Resp) Reset() {
	*x = RoleRecordRule_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[239]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_DefaultGet_Resp) ProtoMessage() {}

func (x *RoleRecordRule_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[239]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{239}
}

func (x *RoleRecordRule_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleRecordRule_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Delete_Req) Reset() {
	*x = RoleRecordRule_Delete_Req{}
	mi := &file_auth_proto_msgTypes[240]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Delete_Req) ProtoMessage() {}

func (x *RoleRecordRule_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[240]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Delete_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{240}
}

func (x *RoleRecordRule_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type RoleRecordRule_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Delete_Resp) Reset() {
	*x = RoleRecordRule_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[241]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Delete_Resp) ProtoMessage() {}

func (x *RoleRecordRule_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[241]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Delete_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{241}
}

func (x *RoleRecordRule_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleRecordRule_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_DeleteById_Req) Reset() {
	*x = RoleRecordRule_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[242]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_DeleteById_Req) ProtoMessage() {}

func (x *RoleRecordRule_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[242]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{242}
}

func (x *RoleRecordRule_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type RoleRecordRule_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_DeleteById_Resp) Reset() {
	*x = RoleRecordRule_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[243]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_DeleteById_Resp) ProtoMessage() {}

func (x *RoleRecordRule_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[243]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{243}
}

func (x *RoleRecordRule_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleRecordRule_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Onchange_Req) Reset() {
	*x = RoleRecordRule_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[244]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Onchange_Req) ProtoMessage() {}

func (x *RoleRecordRule_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[244]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Onchange_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{244}
}

func (x *RoleRecordRule_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *RoleRecordRule_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *RoleRecordRule_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type RoleRecordRule_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Onchange_Resp) Reset() {
	*x = RoleRecordRule_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[245]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Onchange_Resp) ProtoMessage() {}

func (x *RoleRecordRule_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[245]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{245}
}

func (x *RoleRecordRule_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleRecordRule_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_ReadGroup_Req) Reset() {
	*x = RoleRecordRule_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[246]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_ReadGroup_Req) ProtoMessage() {}

func (x *RoleRecordRule_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[246]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{246}
}

func (x *RoleRecordRule_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *RoleRecordRule_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleRecordRule_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleRecordRule_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_ReadGroup_Resp) Reset() {
	*x = RoleRecordRule_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[247]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_ReadGroup_Resp) ProtoMessage() {}

func (x *RoleRecordRule_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[247]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{247}
}

func (x *RoleRecordRule_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleRecordRule_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_ReadGroupCount_Req) Reset() {
	*x = RoleRecordRule_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[248]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_ReadGroupCount_Req) ProtoMessage() {}

func (x *RoleRecordRule_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[248]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{248}
}

func (x *RoleRecordRule_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *RoleRecordRule_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleRecordRule_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleRecordRule_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_ReadGroupCount_Resp) Reset() {
	*x = RoleRecordRule_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[249]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_ReadGroupCount_Resp) ProtoMessage() {}

func (x *RoleRecordRule_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[249]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{249}
}

func (x *RoleRecordRule_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type RoleRecordRule_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Search_Req) Reset() {
	*x = RoleRecordRule_Search_Req{}
	mi := &file_auth_proto_msgTypes[250]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Search_Req) ProtoMessage() {}

func (x *RoleRecordRule_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[250]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Search_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{250}
}

func (x *RoleRecordRule_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleRecordRule_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type RoleRecordRule_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Search_Resp) Reset() {
	*x = RoleRecordRule_Search_Resp{}
	mi := &file_auth_proto_msgTypes[251]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Search_Resp) ProtoMessage() {}

func (x *RoleRecordRule_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[251]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Search_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{251}
}

func (x *RoleRecordRule_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleRecordRule_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Update_Req) Reset() {
	*x = RoleRecordRule_Update_Req{}
	mi := &file_auth_proto_msgTypes[252]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Update_Req) ProtoMessage() {}

func (x *RoleRecordRule_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[252]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Update_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{252}
}

func (x *RoleRecordRule_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *RoleRecordRule_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleRecordRule_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleRecordRule_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_Update_Resp) Reset() {
	*x = RoleRecordRule_Update_Resp{}
	mi := &file_auth_proto_msgTypes[253]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_Update_Resp) ProtoMessage() {}

func (x *RoleRecordRule_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[253]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_Update_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{253}
}

func (x *RoleRecordRule_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type RoleRecordRule_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_UpdateById_Req) Reset() {
	*x = RoleRecordRule_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[254]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_UpdateById_Req) ProtoMessage() {}

func (x *RoleRecordRule_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[254]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{254}
}

func (x *RoleRecordRule_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *RoleRecordRule_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *RoleRecordRule_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type RoleRecordRule_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RoleRecordRule_UpdateById_Resp) Reset() {
	*x = RoleRecordRule_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[255]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RoleRecordRule_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RoleRecordRule_UpdateById_Resp) ProtoMessage() {}

func (x *RoleRecordRule_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[255]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RoleRecordRule_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*RoleRecordRule_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{255}
}

func (x *RoleRecordRule_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Browse_Req) Reset() {
	*x = Session_Browse_Req{}
	mi := &file_auth_proto_msgTypes[256]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Browse_Req) ProtoMessage() {}

func (x *Session_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[256]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Browse_Req.ProtoReflect.Descriptor instead.
func (*Session_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{256}
}

func (x *Session_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Session_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Session_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Browse_Resp) Reset() {
	*x = Session_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[257]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Browse_Resp) ProtoMessage() {}

func (x *Session_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[257]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Browse_Resp.ProtoReflect.Descriptor instead.
func (*Session_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{257}
}

func (x *Session_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_BrowseMany_Req) Reset() {
	*x = Session_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[258]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_BrowseMany_Req) ProtoMessage() {}

func (x *Session_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[258]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*Session_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{258}
}

func (x *Session_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *Session_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Session_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_BrowseMany_Resp) Reset() {
	*x = Session_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[259]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_BrowseMany_Resp) ProtoMessage() {}

func (x *Session_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[259]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*Session_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{259}
}

func (x *Session_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_CleanExpiredSessions_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_CleanExpiredSessions_Resp) Reset() {
	*x = Session_CleanExpiredSessions_Resp{}
	mi := &file_auth_proto_msgTypes[260]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_CleanExpiredSessions_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_CleanExpiredSessions_Resp) ProtoMessage() {}

func (x *Session_CleanExpiredSessions_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[260]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_CleanExpiredSessions_Resp.ProtoReflect.Descriptor instead.
func (*Session_CleanExpiredSessions_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{260}
}

func (x *Session_CleanExpiredSessions_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Session_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Count_Req) Reset() {
	*x = Session_Count_Req{}
	mi := &file_auth_proto_msgTypes[261]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Count_Req) ProtoMessage() {}

func (x *Session_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[261]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Count_Req.ProtoReflect.Descriptor instead.
func (*Session_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{261}
}

func (x *Session_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Session_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Count_Resp) Reset() {
	*x = Session_Count_Resp{}
	mi := &file_auth_proto_msgTypes[262]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Count_Resp) ProtoMessage() {}

func (x *Session_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[262]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Count_Resp.ProtoReflect.Descriptor instead.
func (*Session_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{262}
}

func (x *Session_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Session_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Create_Req) Reset() {
	*x = Session_Create_Req{}
	mi := &file_auth_proto_msgTypes[263]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Create_Req) ProtoMessage() {}

func (x *Session_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[263]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Create_Req.ProtoReflect.Descriptor instead.
func (*Session_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{263}
}

func (x *Session_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *Session_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Session_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Create_Resp) Reset() {
	*x = Session_Create_Resp{}
	mi := &file_auth_proto_msgTypes[264]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Create_Resp) ProtoMessage() {}

func (x *Session_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[264]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Create_Resp.ProtoReflect.Descriptor instead.
func (*Session_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{264}
}

func (x *Session_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_CreateMany_Req) Reset() {
	*x = Session_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[265]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_CreateMany_Req) ProtoMessage() {}

func (x *Session_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[265]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*Session_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{265}
}

func (x *Session_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Session_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Session_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_CreateMany_Resp) Reset() {
	*x = Session_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[266]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_CreateMany_Resp) ProtoMessage() {}

func (x *Session_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[266]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*Session_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{266}
}

func (x *Session_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_DefaultGet_Req) Reset() {
	*x = Session_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[267]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_DefaultGet_Req) ProtoMessage() {}

func (x *Session_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[267]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*Session_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{267}
}

func (x *Session_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type Session_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_DefaultGet_Resp) Reset() {
	*x = Session_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[268]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_DefaultGet_Resp) ProtoMessage() {}

func (x *Session_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[268]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*Session_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{268}
}

func (x *Session_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Delete_Req) Reset() {
	*x = Session_Delete_Req{}
	mi := &file_auth_proto_msgTypes[269]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Delete_Req) ProtoMessage() {}

func (x *Session_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[269]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Delete_Req.ProtoReflect.Descriptor instead.
func (*Session_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{269}
}

func (x *Session_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Session_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Delete_Resp) Reset() {
	*x = Session_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[270]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Delete_Resp) ProtoMessage() {}

func (x *Session_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[270]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Delete_Resp.ProtoReflect.Descriptor instead.
func (*Session_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{270}
}

func (x *Session_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Session_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_DeleteById_Req) Reset() {
	*x = Session_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[271]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_DeleteById_Req) ProtoMessage() {}

func (x *Session_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[271]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*Session_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{271}
}

func (x *Session_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type Session_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_DeleteById_Resp) Reset() {
	*x = Session_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[272]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_DeleteById_Resp) ProtoMessage() {}

func (x *Session_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[272]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*Session_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{272}
}

func (x *Session_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Session_GetActiveSessionsForUser_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_GetActiveSessionsForUser_Req) Reset() {
	*x = Session_GetActiveSessionsForUser_Req{}
	mi := &file_auth_proto_msgTypes[273]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_GetActiveSessionsForUser_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_GetActiveSessionsForUser_Req) ProtoMessage() {}

func (x *Session_GetActiveSessionsForUser_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[273]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_GetActiveSessionsForUser_Req.ProtoReflect.Descriptor instead.
func (*Session_GetActiveSessionsForUser_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{273}
}

func (x *Session_GetActiveSessionsForUser_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

type Session_GetActiveSessionsForUser_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_GetActiveSessionsForUser_Resp) Reset() {
	*x = Session_GetActiveSessionsForUser_Resp{}
	mi := &file_auth_proto_msgTypes[274]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_GetActiveSessionsForUser_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_GetActiveSessionsForUser_Resp) ProtoMessage() {}

func (x *Session_GetActiveSessionsForUser_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[274]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_GetActiveSessionsForUser_Resp.ProtoReflect.Descriptor instead.
func (*Session_GetActiveSessionsForUser_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{274}
}

func (x *Session_GetActiveSessionsForUser_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Onchange_Req) Reset() {
	*x = Session_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[275]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Onchange_Req) ProtoMessage() {}

func (x *Session_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[275]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Onchange_Req.ProtoReflect.Descriptor instead.
func (*Session_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{275}
}

func (x *Session_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *Session_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *Session_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type Session_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Onchange_Resp) Reset() {
	*x = Session_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[276]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Onchange_Resp) ProtoMessage() {}

func (x *Session_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[276]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*Session_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{276}
}

func (x *Session_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_ReadGroup_Req) Reset() {
	*x = Session_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[277]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_ReadGroup_Req) ProtoMessage() {}

func (x *Session_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[277]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*Session_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{277}
}

func (x *Session_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Session_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Session_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Session_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_ReadGroup_Resp) Reset() {
	*x = Session_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[278]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_ReadGroup_Resp) ProtoMessage() {}

func (x *Session_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[278]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*Session_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{278}
}

func (x *Session_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_ReadGroupCount_Req) Reset() {
	*x = Session_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[279]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_ReadGroupCount_Req) ProtoMessage() {}

func (x *Session_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[279]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*Session_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{279}
}

func (x *Session_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Session_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Session_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Session_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_ReadGroupCount_Resp) Reset() {
	*x = Session_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[280]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_ReadGroupCount_Resp) ProtoMessage() {}

func (x *Session_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[280]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*Session_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{280}
}

func (x *Session_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Session_RevokeAllForUser_Req struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	UserId          string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	ExceptSessionId string                 `protobuf:"bytes,2,opt,name=exceptSessionId,proto3" json:"exceptSessionId,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *Session_RevokeAllForUser_Req) Reset() {
	*x = Session_RevokeAllForUser_Req{}
	mi := &file_auth_proto_msgTypes[281]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_RevokeAllForUser_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_RevokeAllForUser_Req) ProtoMessage() {}

func (x *Session_RevokeAllForUser_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[281]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_RevokeAllForUser_Req.ProtoReflect.Descriptor instead.
func (*Session_RevokeAllForUser_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{281}
}

func (x *Session_RevokeAllForUser_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *Session_RevokeAllForUser_Req) GetExceptSessionId() string {
	if x != nil {
		return x.ExceptSessionId
	}
	return ""
}

type Session_RevokeAllForUser_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_RevokeAllForUser_Resp) Reset() {
	*x = Session_RevokeAllForUser_Resp{}
	mi := &file_auth_proto_msgTypes[282]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_RevokeAllForUser_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_RevokeAllForUser_Resp) ProtoMessage() {}

func (x *Session_RevokeAllForUser_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[282]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_RevokeAllForUser_Resp.ProtoReflect.Descriptor instead.
func (*Session_RevokeAllForUser_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{282}
}

func (x *Session_RevokeAllForUser_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Session_RevokeSession_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	SessionId     string                 `protobuf:"bytes,1,opt,name=sessionId,proto3" json:"sessionId,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_RevokeSession_Req) Reset() {
	*x = Session_RevokeSession_Req{}
	mi := &file_auth_proto_msgTypes[283]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_RevokeSession_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_RevokeSession_Req) ProtoMessage() {}

func (x *Session_RevokeSession_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[283]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_RevokeSession_Req.ProtoReflect.Descriptor instead.
func (*Session_RevokeSession_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{283}
}

func (x *Session_RevokeSession_Req) GetSessionId() string {
	if x != nil {
		return x.SessionId
	}
	return ""
}

type Session_RevokeSession_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        bool                   `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_RevokeSession_Resp) Reset() {
	*x = Session_RevokeSession_Resp{}
	mi := &file_auth_proto_msgTypes[284]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_RevokeSession_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_RevokeSession_Resp) ProtoMessage() {}

func (x *Session_RevokeSession_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[284]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_RevokeSession_Resp.ProtoReflect.Descriptor instead.
func (*Session_RevokeSession_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{284}
}

func (x *Session_RevokeSession_Resp) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type Session_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Search_Req) Reset() {
	*x = Session_Search_Req{}
	mi := &file_auth_proto_msgTypes[285]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Search_Req) ProtoMessage() {}

func (x *Session_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[285]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Search_Req.ProtoReflect.Descriptor instead.
func (*Session_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{285}
}

func (x *Session_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Session_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Session_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Search_Resp) Reset() {
	*x = Session_Search_Resp{}
	mi := &file_auth_proto_msgTypes[286]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Search_Resp) ProtoMessage() {}

func (x *Session_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[286]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Search_Resp.ProtoReflect.Descriptor instead.
func (*Session_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{286}
}

func (x *Session_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Update_Req) Reset() {
	*x = Session_Update_Req{}
	mi := &file_auth_proto_msgTypes[287]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Update_Req) ProtoMessage() {}

func (x *Session_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[287]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Update_Req.ProtoReflect.Descriptor instead.
func (*Session_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{287}
}

func (x *Session_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Session_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Session_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Session_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_Update_Resp) Reset() {
	*x = Session_Update_Resp{}
	mi := &file_auth_proto_msgTypes[288]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_Update_Resp) ProtoMessage() {}

func (x *Session_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[288]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_Update_Resp.ProtoReflect.Descriptor instead.
func (*Session_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{288}
}

func (x *Session_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_UpdateById_Req) Reset() {
	*x = Session_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[289]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_UpdateById_Req) ProtoMessage() {}

func (x *Session_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[289]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*Session_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{289}
}

func (x *Session_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Session_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Session_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Session_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_UpdateById_Resp) Reset() {
	*x = Session_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[290]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_UpdateById_Resp) ProtoMessage() {}

func (x *Session_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[290]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*Session_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{290}
}

func (x *Session_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Session_ValidateToken_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Token         string                 `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_ValidateToken_Req) Reset() {
	*x = Session_ValidateToken_Req{}
	mi := &file_auth_proto_msgTypes[291]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_ValidateToken_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_ValidateToken_Req) ProtoMessage() {}

func (x *Session_ValidateToken_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[291]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_ValidateToken_Req.ProtoReflect.Descriptor instead.
func (*Session_ValidateToken_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{291}
}

func (x *Session_ValidateToken_Req) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

type Session_ValidateToken_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Session_ValidateToken_Resp) Reset() {
	*x = Session_ValidateToken_Resp{}
	mi := &file_auth_proto_msgTypes[292]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Session_ValidateToken_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Session_ValidateToken_Resp) ProtoMessage() {}

func (x *Session_ValidateToken_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[292]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Session_ValidateToken_Resp.ProtoReflect.Descriptor instead.
func (*Session_ValidateToken_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{292}
}

func (x *Session_ValidateToken_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Browse_Req) Reset() {
	*x = Token_Browse_Req{}
	mi := &file_auth_proto_msgTypes[293]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Browse_Req) ProtoMessage() {}

func (x *Token_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[293]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Browse_Req.ProtoReflect.Descriptor instead.
func (*Token_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{293}
}

func (x *Token_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Token_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Token_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Browse_Resp) Reset() {
	*x = Token_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[294]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Browse_Resp) ProtoMessage() {}

func (x *Token_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[294]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Browse_Resp.ProtoReflect.Descriptor instead.
func (*Token_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{294}
}

func (x *Token_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_BrowseMany_Req) Reset() {
	*x = Token_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[295]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_BrowseMany_Req) ProtoMessage() {}

func (x *Token_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[295]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*Token_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{295}
}

func (x *Token_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *Token_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type Token_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_BrowseMany_Resp) Reset() {
	*x = Token_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[296]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_BrowseMany_Resp) ProtoMessage() {}

func (x *Token_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[296]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*Token_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{296}
}

func (x *Token_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_CleanExpiredTokens_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_CleanExpiredTokens_Resp) Reset() {
	*x = Token_CleanExpiredTokens_Resp{}
	mi := &file_auth_proto_msgTypes[297]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_CleanExpiredTokens_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_CleanExpiredTokens_Resp) ProtoMessage() {}

func (x *Token_CleanExpiredTokens_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[297]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_CleanExpiredTokens_Resp.ProtoReflect.Descriptor instead.
func (*Token_CleanExpiredTokens_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{297}
}

func (x *Token_CleanExpiredTokens_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Token_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Count_Req) Reset() {
	*x = Token_Count_Req{}
	mi := &file_auth_proto_msgTypes[298]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Count_Req) ProtoMessage() {}

func (x *Token_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[298]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Count_Req.ProtoReflect.Descriptor instead.
func (*Token_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{298}
}

func (x *Token_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Token_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Count_Resp) Reset() {
	*x = Token_Count_Resp{}
	mi := &file_auth_proto_msgTypes[299]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Count_Resp) ProtoMessage() {}

func (x *Token_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[299]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Count_Resp.ProtoReflect.Descriptor instead.
func (*Token_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{299}
}

func (x *Token_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Token_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Create_Req) Reset() {
	*x = Token_Create_Req{}
	mi := &file_auth_proto_msgTypes[300]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Create_Req) ProtoMessage() {}

func (x *Token_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[300]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Create_Req.ProtoReflect.Descriptor instead.
func (*Token_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{300}
}

func (x *Token_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *Token_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Token_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Create_Resp) Reset() {
	*x = Token_Create_Resp{}
	mi := &file_auth_proto_msgTypes[301]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Create_Resp) ProtoMessage() {}

func (x *Token_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[301]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Create_Resp.ProtoReflect.Descriptor instead.
func (*Token_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{301}
}

func (x *Token_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_CreateMany_Req) Reset() {
	*x = Token_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[302]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_CreateMany_Req) ProtoMessage() {}

func (x *Token_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[302]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*Token_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{302}
}

func (x *Token_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Token_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Token_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_CreateMany_Resp) Reset() {
	*x = Token_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[303]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_CreateMany_Resp) ProtoMessage() {}

func (x *Token_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[303]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*Token_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{303}
}

func (x *Token_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_CreateTokenPair_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	Metadata      *structpb.Value        `protobuf:"bytes,2,opt,name=metadata,proto3" json:"metadata,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_CreateTokenPair_Req) Reset() {
	*x = Token_CreateTokenPair_Req{}
	mi := &file_auth_proto_msgTypes[304]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_CreateTokenPair_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_CreateTokenPair_Req) ProtoMessage() {}

func (x *Token_CreateTokenPair_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[304]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_CreateTokenPair_Req.ProtoReflect.Descriptor instead.
func (*Token_CreateTokenPair_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{304}
}

func (x *Token_CreateTokenPair_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *Token_CreateTokenPair_Req) GetMetadata() *structpb.Value {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type Token_CreateTokenPair_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_CreateTokenPair_Resp) Reset() {
	*x = Token_CreateTokenPair_Resp{}
	mi := &file_auth_proto_msgTypes[305]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_CreateTokenPair_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_CreateTokenPair_Resp) ProtoMessage() {}

func (x *Token_CreateTokenPair_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[305]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_CreateTokenPair_Resp.ProtoReflect.Descriptor instead.
func (*Token_CreateTokenPair_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{305}
}

func (x *Token_CreateTokenPair_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_DefaultGet_Req) Reset() {
	*x = Token_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[306]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_DefaultGet_Req) ProtoMessage() {}

func (x *Token_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[306]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*Token_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{306}
}

func (x *Token_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type Token_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_DefaultGet_Resp) Reset() {
	*x = Token_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[307]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_DefaultGet_Resp) ProtoMessage() {}

func (x *Token_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[307]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*Token_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{307}
}

func (x *Token_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Delete_Req) Reset() {
	*x = Token_Delete_Req{}
	mi := &file_auth_proto_msgTypes[308]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Delete_Req) ProtoMessage() {}

func (x *Token_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[308]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Delete_Req.ProtoReflect.Descriptor instead.
func (*Token_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{308}
}

func (x *Token_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type Token_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Delete_Resp) Reset() {
	*x = Token_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[309]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Delete_Resp) ProtoMessage() {}

func (x *Token_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[309]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Delete_Resp.ProtoReflect.Descriptor instead.
func (*Token_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{309}
}

func (x *Token_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Token_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_DeleteById_Req) Reset() {
	*x = Token_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[310]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_DeleteById_Req) ProtoMessage() {}

func (x *Token_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[310]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*Token_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{310}
}

func (x *Token_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type Token_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_DeleteById_Resp) Reset() {
	*x = Token_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[311]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_DeleteById_Resp) ProtoMessage() {}

func (x *Token_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[311]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*Token_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{311}
}

func (x *Token_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Token_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Onchange_Req) Reset() {
	*x = Token_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[312]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Onchange_Req) ProtoMessage() {}

func (x *Token_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[312]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Onchange_Req.ProtoReflect.Descriptor instead.
func (*Token_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{312}
}

func (x *Token_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *Token_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *Token_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type Token_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Onchange_Resp) Reset() {
	*x = Token_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[313]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Onchange_Resp) ProtoMessage() {}

func (x *Token_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[313]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*Token_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{313}
}

func (x *Token_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_ReadGroup_Req) Reset() {
	*x = Token_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[314]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_ReadGroup_Req) ProtoMessage() {}

func (x *Token_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[314]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*Token_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{314}
}

func (x *Token_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Token_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Token_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Token_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_ReadGroup_Resp) Reset() {
	*x = Token_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[315]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_ReadGroup_Resp) ProtoMessage() {}

func (x *Token_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[315]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*Token_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{315}
}

func (x *Token_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_ReadGroupCount_Req) Reset() {
	*x = Token_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[316]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_ReadGroupCount_Req) ProtoMessage() {}

func (x *Token_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[316]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*Token_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{316}
}

func (x *Token_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *Token_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Token_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Token_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_ReadGroupCount_Resp) Reset() {
	*x = Token_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[317]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_ReadGroupCount_Resp) ProtoMessage() {}

func (x *Token_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[317]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*Token_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{317}
}

func (x *Token_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Token_RefreshTokens_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	RefreshToken  string                 `protobuf:"bytes,1,opt,name=refreshToken,proto3" json:"refreshToken,omitempty"`
	Metadata      *structpb.Value        `protobuf:"bytes,2,opt,name=metadata,proto3" json:"metadata,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_RefreshTokens_Req) Reset() {
	*x = Token_RefreshTokens_Req{}
	mi := &file_auth_proto_msgTypes[318]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_RefreshTokens_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_RefreshTokens_Req) ProtoMessage() {}

func (x *Token_RefreshTokens_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[318]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_RefreshTokens_Req.ProtoReflect.Descriptor instead.
func (*Token_RefreshTokens_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{318}
}

func (x *Token_RefreshTokens_Req) GetRefreshToken() string {
	if x != nil {
		return x.RefreshToken
	}
	return ""
}

func (x *Token_RefreshTokens_Req) GetMetadata() *structpb.Value {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type Token_RefreshTokens_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_RefreshTokens_Resp) Reset() {
	*x = Token_RefreshTokens_Resp{}
	mi := &file_auth_proto_msgTypes[319]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_RefreshTokens_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_RefreshTokens_Resp) ProtoMessage() {}

func (x *Token_RefreshTokens_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[319]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_RefreshTokens_Resp.ProtoReflect.Descriptor instead.
func (*Token_RefreshTokens_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{319}
}

func (x *Token_RefreshTokens_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_RevokeAllUserTokens_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	ExceptTokenId string                 `protobuf:"bytes,2,opt,name=exceptTokenId,proto3" json:"exceptTokenId,omitempty"`
	Reason        string                 `protobuf:"bytes,3,opt,name=reason,proto3" json:"reason,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_RevokeAllUserTokens_Req) Reset() {
	*x = Token_RevokeAllUserTokens_Req{}
	mi := &file_auth_proto_msgTypes[320]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_RevokeAllUserTokens_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_RevokeAllUserTokens_Req) ProtoMessage() {}

func (x *Token_RevokeAllUserTokens_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[320]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_RevokeAllUserTokens_Req.ProtoReflect.Descriptor instead.
func (*Token_RevokeAllUserTokens_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{320}
}

func (x *Token_RevokeAllUserTokens_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *Token_RevokeAllUserTokens_Req) GetExceptTokenId() string {
	if x != nil {
		return x.ExceptTokenId
	}
	return ""
}

func (x *Token_RevokeAllUserTokens_Req) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

type Token_RevokeAllUserTokens_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_RevokeAllUserTokens_Resp) Reset() {
	*x = Token_RevokeAllUserTokens_Resp{}
	mi := &file_auth_proto_msgTypes[321]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_RevokeAllUserTokens_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_RevokeAllUserTokens_Resp) ProtoMessage() {}

func (x *Token_RevokeAllUserTokens_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[321]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_RevokeAllUserTokens_Resp.ProtoReflect.Descriptor instead.
func (*Token_RevokeAllUserTokens_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{321}
}

func (x *Token_RevokeAllUserTokens_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Token_RevokeToken_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Token         string                 `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
	Reason        string                 `protobuf:"bytes,2,opt,name=reason,proto3" json:"reason,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_RevokeToken_Req) Reset() {
	*x = Token_RevokeToken_Req{}
	mi := &file_auth_proto_msgTypes[322]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_RevokeToken_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_RevokeToken_Req) ProtoMessage() {}

func (x *Token_RevokeToken_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[322]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_RevokeToken_Req.ProtoReflect.Descriptor instead.
func (*Token_RevokeToken_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{322}
}

func (x *Token_RevokeToken_Req) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *Token_RevokeToken_Req) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

type Token_RevokeToken_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        bool                   `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_RevokeToken_Resp) Reset() {
	*x = Token_RevokeToken_Resp{}
	mi := &file_auth_proto_msgTypes[323]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_RevokeToken_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_RevokeToken_Resp) ProtoMessage() {}

func (x *Token_RevokeToken_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[323]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_RevokeToken_Resp.ProtoReflect.Descriptor instead.
func (*Token_RevokeToken_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{323}
}

func (x *Token_RevokeToken_Resp) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type Token_RevokeUserAccessTokens_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	Reason        string                 `protobuf:"bytes,2,opt,name=reason,proto3" json:"reason,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_RevokeUserAccessTokens_Req) Reset() {
	*x = Token_RevokeUserAccessTokens_Req{}
	mi := &file_auth_proto_msgTypes[324]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_RevokeUserAccessTokens_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_RevokeUserAccessTokens_Req) ProtoMessage() {}

func (x *Token_RevokeUserAccessTokens_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[324]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_RevokeUserAccessTokens_Req.ProtoReflect.Descriptor instead.
func (*Token_RevokeUserAccessTokens_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{324}
}

func (x *Token_RevokeUserAccessTokens_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *Token_RevokeUserAccessTokens_Req) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

type Token_RevokeUserAccessTokens_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_RevokeUserAccessTokens_Resp) Reset() {
	*x = Token_RevokeUserAccessTokens_Resp{}
	mi := &file_auth_proto_msgTypes[325]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_RevokeUserAccessTokens_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_RevokeUserAccessTokens_Resp) ProtoMessage() {}

func (x *Token_RevokeUserAccessTokens_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[325]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_RevokeUserAccessTokens_Resp.ProtoReflect.Descriptor instead.
func (*Token_RevokeUserAccessTokens_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{325}
}

func (x *Token_RevokeUserAccessTokens_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type Token_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Search_Req) Reset() {
	*x = Token_Search_Req{}
	mi := &file_auth_proto_msgTypes[326]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Search_Req) ProtoMessage() {}

func (x *Token_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[326]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Search_Req.ProtoReflect.Descriptor instead.
func (*Token_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{326}
}

func (x *Token_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Token_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type Token_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Search_Resp) Reset() {
	*x = Token_Search_Resp{}
	mi := &file_auth_proto_msgTypes[327]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Search_Resp) ProtoMessage() {}

func (x *Token_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[327]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Search_Resp.ProtoReflect.Descriptor instead.
func (*Token_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{327}
}

func (x *Token_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Update_Req) Reset() {
	*x = Token_Update_Req{}
	mi := &file_auth_proto_msgTypes[328]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Update_Req) ProtoMessage() {}

func (x *Token_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[328]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Update_Req.ProtoReflect.Descriptor instead.
func (*Token_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{328}
}

func (x *Token_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *Token_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Token_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Token_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_Update_Resp) Reset() {
	*x = Token_Update_Resp{}
	mi := &file_auth_proto_msgTypes[329]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_Update_Resp) ProtoMessage() {}

func (x *Token_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[329]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_Update_Resp.ProtoReflect.Descriptor instead.
func (*Token_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{329}
}

func (x *Token_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_UpdateById_Req) Reset() {
	*x = Token_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[330]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_UpdateById_Req) ProtoMessage() {}

func (x *Token_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[330]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*Token_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{330}
}

func (x *Token_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Token_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *Token_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type Token_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_UpdateById_Resp) Reset() {
	*x = Token_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[331]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_UpdateById_Resp) ProtoMessage() {}

func (x *Token_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[331]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*Token_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{331}
}

func (x *Token_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type Token_ValidateToken_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Token         string                 `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
	TokenType     string                 `protobuf:"bytes,2,opt,name=tokenType,proto3" json:"tokenType,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_ValidateToken_Req) Reset() {
	*x = Token_ValidateToken_Req{}
	mi := &file_auth_proto_msgTypes[332]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_ValidateToken_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_ValidateToken_Req) ProtoMessage() {}

func (x *Token_ValidateToken_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[332]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_ValidateToken_Req.ProtoReflect.Descriptor instead.
func (*Token_ValidateToken_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{332}
}

func (x *Token_ValidateToken_Req) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *Token_ValidateToken_Req) GetTokenType() string {
	if x != nil {
		return x.TokenType
	}
	return ""
}

type Token_ValidateToken_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Token_ValidateToken_Resp) Reset() {
	*x = Token_ValidateToken_Resp{}
	mi := &file_auth_proto_msgTypes[333]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Token_ValidateToken_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token_ValidateToken_Resp) ProtoMessage() {}

func (x *Token_ValidateToken_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[333]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token_ValidateToken_Resp.ProtoReflect.Descriptor instead.
func (*Token_ValidateToken_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{333}
}

func (x *Token_ValidateToken_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_AssignRoles_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	RoleIds       *structpb.Value        `protobuf:"bytes,2,opt,name=roleIds,proto3" json:"roleIds,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_AssignRoles_Req) Reset() {
	*x = User_AssignRoles_Req{}
	mi := &file_auth_proto_msgTypes[334]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_AssignRoles_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_AssignRoles_Req) ProtoMessage() {}

func (x *User_AssignRoles_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[334]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_AssignRoles_Req.ProtoReflect.Descriptor instead.
func (*User_AssignRoles_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{334}
}

func (x *User_AssignRoles_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *User_AssignRoles_Req) GetRoleIds() *structpb.Value {
	if x != nil {
		return x.RoleIds
	}
	return nil
}

type User_AssignRoles_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        bool                   `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_AssignRoles_Resp) Reset() {
	*x = User_AssignRoles_Resp{}
	mi := &file_auth_proto_msgTypes[335]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_AssignRoles_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_AssignRoles_Resp) ProtoMessage() {}

func (x *User_AssignRoles_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[335]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_AssignRoles_Resp.ProtoReflect.Descriptor instead.
func (*User_AssignRoles_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{335}
}

func (x *User_AssignRoles_Resp) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type User_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Browse_Req) Reset() {
	*x = User_Browse_Req{}
	mi := &file_auth_proto_msgTypes[336]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Browse_Req) ProtoMessage() {}

func (x *User_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[336]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Browse_Req.ProtoReflect.Descriptor instead.
func (*User_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{336}
}

func (x *User_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *User_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type User_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Browse_Resp) Reset() {
	*x = User_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[337]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Browse_Resp) ProtoMessage() {}

func (x *User_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[337]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Browse_Resp.ProtoReflect.Descriptor instead.
func (*User_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{337}
}

func (x *User_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_BrowseMany_Req) Reset() {
	*x = User_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[338]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_BrowseMany_Req) ProtoMessage() {}

func (x *User_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[338]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*User_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{338}
}

func (x *User_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *User_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type User_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_BrowseMany_Resp) Reset() {
	*x = User_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[339]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_BrowseMany_Resp) ProtoMessage() {}

func (x *User_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[339]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*User_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{339}
}

func (x *User_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_ChangePassword_Req struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	UserId          string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	CurrentPassword string                 `protobuf:"bytes,2,opt,name=currentPassword,proto3" json:"currentPassword,omitempty"`
	NewPassword     string                 `protobuf:"bytes,3,opt,name=newPassword,proto3" json:"newPassword,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *User_ChangePassword_Req) Reset() {
	*x = User_ChangePassword_Req{}
	mi := &file_auth_proto_msgTypes[340]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_ChangePassword_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_ChangePassword_Req) ProtoMessage() {}

func (x *User_ChangePassword_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[340]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_ChangePassword_Req.ProtoReflect.Descriptor instead.
func (*User_ChangePassword_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{340}
}

func (x *User_ChangePassword_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *User_ChangePassword_Req) GetCurrentPassword() string {
	if x != nil {
		return x.CurrentPassword
	}
	return ""
}

func (x *User_ChangePassword_Req) GetNewPassword() string {
	if x != nil {
		return x.NewPassword
	}
	return ""
}

type User_ChangePassword_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        bool                   `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_ChangePassword_Resp) Reset() {
	*x = User_ChangePassword_Resp{}
	mi := &file_auth_proto_msgTypes[341]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_ChangePassword_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_ChangePassword_Resp) ProtoMessage() {}

func (x *User_ChangePassword_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[341]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_ChangePassword_Resp.ProtoReflect.Descriptor instead.
func (*User_ChangePassword_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{341}
}

func (x *User_ChangePassword_Resp) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type User_CheckMethodAccess_Req struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	CompanyId       string                 `protobuf:"bytes,1,opt,name=companyId,proto3" json:"companyId,omitempty"`
	ServiceFullName string                 `protobuf:"bytes,2,opt,name=serviceFullName,proto3" json:"serviceFullName,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *User_CheckMethodAccess_Req) Reset() {
	*x = User_CheckMethodAccess_Req{}
	mi := &file_auth_proto_msgTypes[342]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_CheckMethodAccess_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_CheckMethodAccess_Req) ProtoMessage() {}

func (x *User_CheckMethodAccess_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[342]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_CheckMethodAccess_Req.ProtoReflect.Descriptor instead.
func (*User_CheckMethodAccess_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{342}
}

func (x *User_CheckMethodAccess_Req) GetCompanyId() string {
	if x != nil {
		return x.CompanyId
	}
	return ""
}

func (x *User_CheckMethodAccess_Req) GetServiceFullName() string {
	if x != nil {
		return x.ServiceFullName
	}
	return ""
}

type User_CheckMethodAccess_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        bool                   `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_CheckMethodAccess_Resp) Reset() {
	*x = User_CheckMethodAccess_Resp{}
	mi := &file_auth_proto_msgTypes[343]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_CheckMethodAccess_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_CheckMethodAccess_Resp) ProtoMessage() {}

func (x *User_CheckMethodAccess_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[343]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_CheckMethodAccess_Resp.ProtoReflect.Descriptor instead.
func (*User_CheckMethodAccess_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{343}
}

func (x *User_CheckMethodAccess_Resp) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type User_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Count_Req) Reset() {
	*x = User_Count_Req{}
	mi := &file_auth_proto_msgTypes[344]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Count_Req) ProtoMessage() {}

func (x *User_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[344]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Count_Req.ProtoReflect.Descriptor instead.
func (*User_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{344}
}

func (x *User_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type User_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Count_Resp) Reset() {
	*x = User_Count_Resp{}
	mi := &file_auth_proto_msgTypes[345]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Count_Resp) ProtoMessage() {}

func (x *User_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[345]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Count_Resp.ProtoReflect.Descriptor instead.
func (*User_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{345}
}

func (x *User_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type User_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Create_Req) Reset() {
	*x = User_Create_Req{}
	mi := &file_auth_proto_msgTypes[346]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Create_Req) ProtoMessage() {}

func (x *User_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[346]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Create_Req.ProtoReflect.Descriptor instead.
func (*User_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{346}
}

func (x *User_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *User_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type User_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Create_Resp) Reset() {
	*x = User_Create_Resp{}
	mi := &file_auth_proto_msgTypes[347]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Create_Resp) ProtoMessage() {}

func (x *User_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[347]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Create_Resp.ProtoReflect.Descriptor instead.
func (*User_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{347}
}

func (x *User_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_CreateMany_Req) Reset() {
	*x = User_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[348]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_CreateMany_Req) ProtoMessage() {}

func (x *User_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[348]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*User_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{348}
}

func (x *User_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *User_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type User_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_CreateMany_Resp) Reset() {
	*x = User_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[349]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_CreateMany_Resp) ProtoMessage() {}

func (x *User_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[349]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*User_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{349}
}

func (x *User_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_DefaultGet_Req) Reset() {
	*x = User_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[350]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_DefaultGet_Req) ProtoMessage() {}

func (x *User_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[350]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*User_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{350}
}

func (x *User_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type User_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_DefaultGet_Resp) Reset() {
	*x = User_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[351]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_DefaultGet_Resp) ProtoMessage() {}

func (x *User_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[351]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*User_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{351}
}

func (x *User_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Delete_Req) Reset() {
	*x = User_Delete_Req{}
	mi := &file_auth_proto_msgTypes[352]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Delete_Req) ProtoMessage() {}

func (x *User_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[352]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Delete_Req.ProtoReflect.Descriptor instead.
func (*User_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{352}
}

func (x *User_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type User_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Delete_Resp) Reset() {
	*x = User_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[353]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Delete_Resp) ProtoMessage() {}

func (x *User_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[353]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Delete_Resp.ProtoReflect.Descriptor instead.
func (*User_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{353}
}

func (x *User_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type User_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_DeleteById_Req) Reset() {
	*x = User_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[354]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_DeleteById_Req) ProtoMessage() {}

func (x *User_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[354]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*User_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{354}
}

func (x *User_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type User_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_DeleteById_Resp) Reset() {
	*x = User_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[355]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_DeleteById_Resp) ProtoMessage() {}

func (x *User_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[355]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*User_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{355}
}

func (x *User_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type User_GetRecordRuleCondition_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Model         string                 `protobuf:"bytes,1,opt,name=model,proto3" json:"model,omitempty"`
	Op            string                 `protobuf:"bytes,2,opt,name=op,proto3" json:"op,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_GetRecordRuleCondition_Req) Reset() {
	*x = User_GetRecordRuleCondition_Req{}
	mi := &file_auth_proto_msgTypes[356]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_GetRecordRuleCondition_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_GetRecordRuleCondition_Req) ProtoMessage() {}

func (x *User_GetRecordRuleCondition_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[356]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_GetRecordRuleCondition_Req.ProtoReflect.Descriptor instead.
func (*User_GetRecordRuleCondition_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{356}
}

func (x *User_GetRecordRuleCondition_Req) GetModel() string {
	if x != nil {
		return x.Model
	}
	return ""
}

func (x *User_GetRecordRuleCondition_Req) GetOp() string {
	if x != nil {
		return x.Op
	}
	return ""
}

type User_GetRecordRuleCondition_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_GetRecordRuleCondition_Resp) Reset() {
	*x = User_GetRecordRuleCondition_Resp{}
	mi := &file_auth_proto_msgTypes[357]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_GetRecordRuleCondition_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_GetRecordRuleCondition_Resp) ProtoMessage() {}

func (x *User_GetRecordRuleCondition_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[357]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_GetRecordRuleCondition_Resp.ProtoReflect.Descriptor instead.
func (*User_GetRecordRuleCondition_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{357}
}

func (x *User_GetRecordRuleCondition_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_HasPermission_Req struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	UserId         string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	PermissionCode string                 `protobuf:"bytes,2,opt,name=permissionCode,proto3" json:"permissionCode,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *User_HasPermission_Req) Reset() {
	*x = User_HasPermission_Req{}
	mi := &file_auth_proto_msgTypes[358]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_HasPermission_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_HasPermission_Req) ProtoMessage() {}

func (x *User_HasPermission_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[358]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_HasPermission_Req.ProtoReflect.Descriptor instead.
func (*User_HasPermission_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{358}
}

func (x *User_HasPermission_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *User_HasPermission_Req) GetPermissionCode() string {
	if x != nil {
		return x.PermissionCode
	}
	return ""
}

type User_HasPermission_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        bool                   `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_HasPermission_Resp) Reset() {
	*x = User_HasPermission_Resp{}
	mi := &file_auth_proto_msgTypes[359]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_HasPermission_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_HasPermission_Resp) ProtoMessage() {}

func (x *User_HasPermission_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[359]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_HasPermission_Resp.ProtoReflect.Descriptor instead.
func (*User_HasPermission_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{359}
}

func (x *User_HasPermission_Resp) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type User_HasRole_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	RoleCode      string                 `protobuf:"bytes,2,opt,name=roleCode,proto3" json:"roleCode,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_HasRole_Req) Reset() {
	*x = User_HasRole_Req{}
	mi := &file_auth_proto_msgTypes[360]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_HasRole_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_HasRole_Req) ProtoMessage() {}

func (x *User_HasRole_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[360]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_HasRole_Req.ProtoReflect.Descriptor instead.
func (*User_HasRole_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{360}
}

func (x *User_HasRole_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *User_HasRole_Req) GetRoleCode() string {
	if x != nil {
		return x.RoleCode
	}
	return ""
}

type User_HasRole_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        bool                   `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_HasRole_Resp) Reset() {
	*x = User_HasRole_Resp{}
	mi := &file_auth_proto_msgTypes[361]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_HasRole_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_HasRole_Resp) ProtoMessage() {}

func (x *User_HasRole_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[361]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_HasRole_Resp.ProtoReflect.Descriptor instead.
func (*User_HasRole_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{361}
}

func (x *User_HasRole_Resp) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type User_Login_Req struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	UsernameOrEmail string                 `protobuf:"bytes,1,opt,name=usernameOrEmail,proto3" json:"usernameOrEmail,omitempty"`
	Password        string                 `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
	IpAddress       string                 `protobuf:"bytes,3,opt,name=ipAddress,proto3" json:"ipAddress,omitempty"`
	DeviceInfo      string                 `protobuf:"bytes,4,opt,name=deviceInfo,proto3" json:"deviceInfo,omitempty"`
	RememberMe      bool                   `protobuf:"varint,5,opt,name=rememberMe,proto3" json:"rememberMe,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *User_Login_Req) Reset() {
	*x = User_Login_Req{}
	mi := &file_auth_proto_msgTypes[362]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Login_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Login_Req) ProtoMessage() {}

func (x *User_Login_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[362]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Login_Req.ProtoReflect.Descriptor instead.
func (*User_Login_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{362}
}

func (x *User_Login_Req) GetUsernameOrEmail() string {
	if x != nil {
		return x.UsernameOrEmail
	}
	return ""
}

func (x *User_Login_Req) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *User_Login_Req) GetIpAddress() string {
	if x != nil {
		return x.IpAddress
	}
	return ""
}

func (x *User_Login_Req) GetDeviceInfo() string {
	if x != nil {
		return x.DeviceInfo
	}
	return ""
}

func (x *User_Login_Req) GetRememberMe() bool {
	if x != nil {
		return x.RememberMe
	}
	return false
}

type User_Login_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Login_Resp) Reset() {
	*x = User_Login_Resp{}
	mi := &file_auth_proto_msgTypes[363]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Login_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Login_Resp) ProtoMessage() {}

func (x *User_Login_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[363]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Login_Resp.ProtoReflect.Descriptor instead.
func (*User_Login_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{363}
}

func (x *User_Login_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_Logout_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Token         string                 `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
	AllDevices    bool                   `protobuf:"varint,2,opt,name=allDevices,proto3" json:"allDevices,omitempty"`
	DeviceInfo    string                 `protobuf:"bytes,3,opt,name=deviceInfo,proto3" json:"deviceInfo,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Logout_Req) Reset() {
	*x = User_Logout_Req{}
	mi := &file_auth_proto_msgTypes[364]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Logout_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Logout_Req) ProtoMessage() {}

func (x *User_Logout_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[364]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Logout_Req.ProtoReflect.Descriptor instead.
func (*User_Logout_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{364}
}

func (x *User_Logout_Req) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *User_Logout_Req) GetAllDevices() bool {
	if x != nil {
		return x.AllDevices
	}
	return false
}

func (x *User_Logout_Req) GetDeviceInfo() string {
	if x != nil {
		return x.DeviceInfo
	}
	return ""
}

type User_Logout_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        bool                   `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Logout_Resp) Reset() {
	*x = User_Logout_Resp{}
	mi := &file_auth_proto_msgTypes[365]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Logout_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Logout_Resp) ProtoMessage() {}

func (x *User_Logout_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[365]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Logout_Resp.ProtoReflect.Descriptor instead.
func (*User_Logout_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{365}
}

func (x *User_Logout_Resp) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type User_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Onchange_Req) Reset() {
	*x = User_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[366]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Onchange_Req) ProtoMessage() {}

func (x *User_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[366]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Onchange_Req.ProtoReflect.Descriptor instead.
func (*User_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{366}
}

func (x *User_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *User_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *User_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type User_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Onchange_Resp) Reset() {
	*x = User_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[367]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Onchange_Resp) ProtoMessage() {}

func (x *User_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[367]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*User_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{367}
}

func (x *User_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_ReadGroup_Req) Reset() {
	*x = User_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[368]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_ReadGroup_Req) ProtoMessage() {}

func (x *User_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[368]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*User_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{368}
}

func (x *User_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *User_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *User_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type User_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_ReadGroup_Resp) Reset() {
	*x = User_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[369]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_ReadGroup_Resp) ProtoMessage() {}

func (x *User_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[369]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*User_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{369}
}

func (x *User_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_ReadGroupCount_Req) Reset() {
	*x = User_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[370]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_ReadGroupCount_Req) ProtoMessage() {}

func (x *User_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[370]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*User_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{370}
}

func (x *User_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *User_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *User_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type User_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_ReadGroupCount_Resp) Reset() {
	*x = User_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[371]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_ReadGroupCount_Resp) ProtoMessage() {}

func (x *User_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[371]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*User_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{371}
}

func (x *User_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type User_RefreshTokens_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	RefreshToken  string                 `protobuf:"bytes,1,opt,name=refreshToken,proto3" json:"refreshToken,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_RefreshTokens_Req) Reset() {
	*x = User_RefreshTokens_Req{}
	mi := &file_auth_proto_msgTypes[372]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_RefreshTokens_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_RefreshTokens_Req) ProtoMessage() {}

func (x *User_RefreshTokens_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[372]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_RefreshTokens_Req.ProtoReflect.Descriptor instead.
func (*User_RefreshTokens_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{372}
}

func (x *User_RefreshTokens_Req) GetRefreshToken() string {
	if x != nil {
		return x.RefreshToken
	}
	return ""
}

type User_RefreshTokens_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_RefreshTokens_Resp) Reset() {
	*x = User_RefreshTokens_Resp{}
	mi := &file_auth_proto_msgTypes[373]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_RefreshTokens_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_RefreshTokens_Resp) ProtoMessage() {}

func (x *User_RefreshTokens_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[373]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_RefreshTokens_Resp.ProtoReflect.Descriptor instead.
func (*User_RefreshTokens_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{373}
}

func (x *User_RefreshTokens_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_Register_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserData      *structpb.Value        `protobuf:"bytes,1,opt,name=userData,proto3" json:"userData,omitempty"`
	Password      string                 `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Register_Req) Reset() {
	*x = User_Register_Req{}
	mi := &file_auth_proto_msgTypes[374]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Register_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Register_Req) ProtoMessage() {}

func (x *User_Register_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[374]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Register_Req.ProtoReflect.Descriptor instead.
func (*User_Register_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{374}
}

func (x *User_Register_Req) GetUserData() *structpb.Value {
	if x != nil {
		return x.UserData
	}
	return nil
}

func (x *User_Register_Req) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type User_Register_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        string                 `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Register_Resp) Reset() {
	*x = User_Register_Resp{}
	mi := &file_auth_proto_msgTypes[375]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Register_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Register_Resp) ProtoMessage() {}

func (x *User_Register_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[375]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Register_Resp.ProtoReflect.Descriptor instead.
func (*User_Register_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{375}
}

func (x *User_Register_Resp) GetResult() string {
	if x != nil {
		return x.Result
	}
	return ""
}

type User_RemoveRoles_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	RoleIds       *structpb.Value        `protobuf:"bytes,2,opt,name=roleIds,proto3" json:"roleIds,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_RemoveRoles_Req) Reset() {
	*x = User_RemoveRoles_Req{}
	mi := &file_auth_proto_msgTypes[376]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_RemoveRoles_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_RemoveRoles_Req) ProtoMessage() {}

func (x *User_RemoveRoles_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[376]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_RemoveRoles_Req.ProtoReflect.Descriptor instead.
func (*User_RemoveRoles_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{376}
}

func (x *User_RemoveRoles_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *User_RemoveRoles_Req) GetRoleIds() *structpb.Value {
	if x != nil {
		return x.RoleIds
	}
	return nil
}

type User_RemoveRoles_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        bool                   `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_RemoveRoles_Resp) Reset() {
	*x = User_RemoveRoles_Resp{}
	mi := &file_auth_proto_msgTypes[377]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_RemoveRoles_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_RemoveRoles_Resp) ProtoMessage() {}

func (x *User_RemoveRoles_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[377]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_RemoveRoles_Resp.ProtoReflect.Descriptor instead.
func (*User_RemoveRoles_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{377}
}

func (x *User_RemoveRoles_Resp) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type User_ResetPassword_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        string                 `protobuf:"bytes,1,opt,name=userId,proto3" json:"userId,omitempty"`
	NewPassword   string                 `protobuf:"bytes,2,opt,name=newPassword,proto3" json:"newPassword,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_ResetPassword_Req) Reset() {
	*x = User_ResetPassword_Req{}
	mi := &file_auth_proto_msgTypes[378]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_ResetPassword_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_ResetPassword_Req) ProtoMessage() {}

func (x *User_ResetPassword_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[378]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_ResetPassword_Req.ProtoReflect.Descriptor instead.
func (*User_ResetPassword_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{378}
}

func (x *User_ResetPassword_Req) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *User_ResetPassword_Req) GetNewPassword() string {
	if x != nil {
		return x.NewPassword
	}
	return ""
}

type User_ResetPassword_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        bool                   `protobuf:"varint,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_ResetPassword_Resp) Reset() {
	*x = User_ResetPassword_Resp{}
	mi := &file_auth_proto_msgTypes[379]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_ResetPassword_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_ResetPassword_Resp) ProtoMessage() {}

func (x *User_ResetPassword_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[379]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_ResetPassword_Resp.ProtoReflect.Descriptor instead.
func (*User_ResetPassword_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{379}
}

func (x *User_ResetPassword_Resp) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type User_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Search_Req) Reset() {
	*x = User_Search_Req{}
	mi := &file_auth_proto_msgTypes[380]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Search_Req) ProtoMessage() {}

func (x *User_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[380]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Search_Req.ProtoReflect.Descriptor instead.
func (*User_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{380}
}

func (x *User_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *User_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type User_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Search_Resp) Reset() {
	*x = User_Search_Resp{}
	mi := &file_auth_proto_msgTypes[381]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Search_Resp) ProtoMessage() {}

func (x *User_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[381]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Search_Resp.ProtoReflect.Descriptor instead.
func (*User_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{381}
}

func (x *User_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Update_Req) Reset() {
	*x = User_Update_Req{}
	mi := &file_auth_proto_msgTypes[382]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Update_Req) ProtoMessage() {}

func (x *User_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[382]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Update_Req.ProtoReflect.Descriptor instead.
func (*User_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{382}
}

func (x *User_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *User_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *User_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type User_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_Update_Resp) Reset() {
	*x = User_Update_Resp{}
	mi := &file_auth_proto_msgTypes[383]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_Update_Resp) ProtoMessage() {}

func (x *User_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[383]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_Update_Resp.ProtoReflect.Descriptor instead.
func (*User_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{383}
}

func (x *User_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type User_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_UpdateById_Req) Reset() {
	*x = User_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[384]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_UpdateById_Req) ProtoMessage() {}

func (x *User_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[384]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*User_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{384}
}

func (x *User_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *User_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *User_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type User_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User_UpdateById_Resp) Reset() {
	*x = User_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[385]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User_UpdateById_Resp) ProtoMessage() {}

func (x *User_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[385]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*User_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{385}
}

func (x *User_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type UserRole_Browse_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Browse_Req) Reset() {
	*x = UserRole_Browse_Req{}
	mi := &file_auth_proto_msgTypes[386]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Browse_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Browse_Req) ProtoMessage() {}

func (x *UserRole_Browse_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[386]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Browse_Req.ProtoReflect.Descriptor instead.
func (*UserRole_Browse_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{386}
}

func (x *UserRole_Browse_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *UserRole_Browse_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type UserRole_Browse_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Browse_Resp) Reset() {
	*x = UserRole_Browse_Resp{}
	mi := &file_auth_proto_msgTypes[387]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Browse_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Browse_Resp) ProtoMessage() {}

func (x *UserRole_Browse_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[387]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Browse_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_Browse_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{387}
}

func (x *UserRole_Browse_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type UserRole_BrowseMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           *structpb.Value        `protobuf:"bytes,1,opt,name=ids,proto3" json:"ids,omitempty"`
	Fields        *structpb.Value        `protobuf:"bytes,2,opt,name=fields,proto3" json:"fields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_BrowseMany_Req) Reset() {
	*x = UserRole_BrowseMany_Req{}
	mi := &file_auth_proto_msgTypes[388]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_BrowseMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_BrowseMany_Req) ProtoMessage() {}

func (x *UserRole_BrowseMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[388]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_BrowseMany_Req.ProtoReflect.Descriptor instead.
func (*UserRole_BrowseMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{388}
}

func (x *UserRole_BrowseMany_Req) GetIds() *structpb.Value {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *UserRole_BrowseMany_Req) GetFields() *structpb.Value {
	if x != nil {
		return x.Fields
	}
	return nil
}

type UserRole_BrowseMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_BrowseMany_Resp) Reset() {
	*x = UserRole_BrowseMany_Resp{}
	mi := &file_auth_proto_msgTypes[389]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_BrowseMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_BrowseMany_Resp) ProtoMessage() {}

func (x *UserRole_BrowseMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[389]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_BrowseMany_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_BrowseMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{389}
}

func (x *UserRole_BrowseMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type UserRole_Count_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Count_Req) Reset() {
	*x = UserRole_Count_Req{}
	mi := &file_auth_proto_msgTypes[390]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Count_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Count_Req) ProtoMessage() {}

func (x *UserRole_Count_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[390]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Count_Req.ProtoReflect.Descriptor instead.
func (*UserRole_Count_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{390}
}

func (x *UserRole_Count_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type UserRole_Count_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Count_Resp) Reset() {
	*x = UserRole_Count_Resp{}
	mi := &file_auth_proto_msgTypes[391]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Count_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Count_Resp) ProtoMessage() {}

func (x *UserRole_Count_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[391]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Count_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_Count_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{391}
}

func (x *UserRole_Count_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type UserRole_Create_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Create_Req) Reset() {
	*x = UserRole_Create_Req{}
	mi := &file_auth_proto_msgTypes[392]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Create_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Create_Req) ProtoMessage() {}

func (x *UserRole_Create_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[392]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Create_Req.ProtoReflect.Descriptor instead.
func (*UserRole_Create_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{392}
}

func (x *UserRole_Create_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *UserRole_Create_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type UserRole_Create_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Create_Resp) Reset() {
	*x = UserRole_Create_Resp{}
	mi := &file_auth_proto_msgTypes[393]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Create_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Create_Resp) ProtoMessage() {}

func (x *UserRole_Create_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[393]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Create_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_Create_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{393}
}

func (x *UserRole_Create_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type UserRole_CreateMany_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        *structpb.Value        `protobuf:"bytes,1,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,2,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_CreateMany_Req) Reset() {
	*x = UserRole_CreateMany_Req{}
	mi := &file_auth_proto_msgTypes[394]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_CreateMany_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_CreateMany_Req) ProtoMessage() {}

func (x *UserRole_CreateMany_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[394]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_CreateMany_Req.ProtoReflect.Descriptor instead.
func (*UserRole_CreateMany_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{394}
}

func (x *UserRole_CreateMany_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *UserRole_CreateMany_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type UserRole_CreateMany_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_CreateMany_Resp) Reset() {
	*x = UserRole_CreateMany_Resp{}
	mi := &file_auth_proto_msgTypes[395]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_CreateMany_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_CreateMany_Resp) ProtoMessage() {}

func (x *UserRole_CreateMany_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[395]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_CreateMany_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_CreateMany_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{395}
}

func (x *UserRole_CreateMany_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type UserRole_DefaultGet_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Value         *structpb.Value        `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_DefaultGet_Req) Reset() {
	*x = UserRole_DefaultGet_Req{}
	mi := &file_auth_proto_msgTypes[396]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_DefaultGet_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_DefaultGet_Req) ProtoMessage() {}

func (x *UserRole_DefaultGet_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[396]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_DefaultGet_Req.ProtoReflect.Descriptor instead.
func (*UserRole_DefaultGet_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{396}
}

func (x *UserRole_DefaultGet_Req) GetValue() *structpb.Value {
	if x != nil {
		return x.Value
	}
	return nil
}

type UserRole_DefaultGet_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_DefaultGet_Resp) Reset() {
	*x = UserRole_DefaultGet_Resp{}
	mi := &file_auth_proto_msgTypes[397]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_DefaultGet_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_DefaultGet_Resp) ProtoMessage() {}

func (x *UserRole_DefaultGet_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[397]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_DefaultGet_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_DefaultGet_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{397}
}

func (x *UserRole_DefaultGet_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type UserRole_Delete_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Delete_Req) Reset() {
	*x = UserRole_Delete_Req{}
	mi := &file_auth_proto_msgTypes[398]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Delete_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Delete_Req) ProtoMessage() {}

func (x *UserRole_Delete_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[398]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Delete_Req.ProtoReflect.Descriptor instead.
func (*UserRole_Delete_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{398}
}

func (x *UserRole_Delete_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

type UserRole_Delete_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Delete_Resp) Reset() {
	*x = UserRole_Delete_Resp{}
	mi := &file_auth_proto_msgTypes[399]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Delete_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Delete_Resp) ProtoMessage() {}

func (x *UserRole_Delete_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[399]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Delete_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_Delete_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{399}
}

func (x *UserRole_Delete_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type UserRole_DeleteById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_DeleteById_Req) Reset() {
	*x = UserRole_DeleteById_Req{}
	mi := &file_auth_proto_msgTypes[400]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_DeleteById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_DeleteById_Req) ProtoMessage() {}

func (x *UserRole_DeleteById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[400]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_DeleteById_Req.ProtoReflect.Descriptor instead.
func (*UserRole_DeleteById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{400}
}

func (x *UserRole_DeleteById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type UserRole_DeleteById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_DeleteById_Resp) Reset() {
	*x = UserRole_DeleteById_Resp{}
	mi := &file_auth_proto_msgTypes[401]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_DeleteById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_DeleteById_Resp) ProtoMessage() {}

func (x *UserRole_DeleteById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[401]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_DeleteById_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_DeleteById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{401}
}

func (x *UserRole_DeleteById_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type UserRole_Onchange_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Draft         *structpb.Value        `protobuf:"bytes,1,opt,name=draft,proto3" json:"draft,omitempty"`
	Changed       *structpb.Value        `protobuf:"bytes,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Opts          *structpb.Value        `protobuf:"bytes,3,opt,name=opts,proto3" json:"opts,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Onchange_Req) Reset() {
	*x = UserRole_Onchange_Req{}
	mi := &file_auth_proto_msgTypes[402]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Onchange_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Onchange_Req) ProtoMessage() {}

func (x *UserRole_Onchange_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[402]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Onchange_Req.ProtoReflect.Descriptor instead.
func (*UserRole_Onchange_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{402}
}

func (x *UserRole_Onchange_Req) GetDraft() *structpb.Value {
	if x != nil {
		return x.Draft
	}
	return nil
}

func (x *UserRole_Onchange_Req) GetChanged() *structpb.Value {
	if x != nil {
		return x.Changed
	}
	return nil
}

func (x *UserRole_Onchange_Req) GetOpts() *structpb.Value {
	if x != nil {
		return x.Opts
	}
	return nil
}

type UserRole_Onchange_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Onchange_Resp) Reset() {
	*x = UserRole_Onchange_Resp{}
	mi := &file_auth_proto_msgTypes[403]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Onchange_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Onchange_Resp) ProtoMessage() {}

func (x *UserRole_Onchange_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[403]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Onchange_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_Onchange_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{403}
}

func (x *UserRole_Onchange_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type UserRole_ReadGroup_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_ReadGroup_Req) Reset() {
	*x = UserRole_ReadGroup_Req{}
	mi := &file_auth_proto_msgTypes[404]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_ReadGroup_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_ReadGroup_Req) ProtoMessage() {}

func (x *UserRole_ReadGroup_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[404]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_ReadGroup_Req.ProtoReflect.Descriptor instead.
func (*UserRole_ReadGroup_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{404}
}

func (x *UserRole_ReadGroup_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *UserRole_ReadGroup_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *UserRole_ReadGroup_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type UserRole_ReadGroup_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_ReadGroup_Resp) Reset() {
	*x = UserRole_ReadGroup_Resp{}
	mi := &file_auth_proto_msgTypes[405]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_ReadGroup_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_ReadGroup_Resp) ProtoMessage() {}

func (x *UserRole_ReadGroup_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[405]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_ReadGroup_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_ReadGroup_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{405}
}

func (x *UserRole_ReadGroup_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type UserRole_ReadGroupCount_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Groupby       *structpb.Value        `protobuf:"bytes,1,opt,name=groupby,proto3" json:"groupby,omitempty"`
	Condition     *structpb.Value        `protobuf:"bytes,2,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,3,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_ReadGroupCount_Req) Reset() {
	*x = UserRole_ReadGroupCount_Req{}
	mi := &file_auth_proto_msgTypes[406]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_ReadGroupCount_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_ReadGroupCount_Req) ProtoMessage() {}

func (x *UserRole_ReadGroupCount_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[406]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_ReadGroupCount_Req.ProtoReflect.Descriptor instead.
func (*UserRole_ReadGroupCount_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{406}
}

func (x *UserRole_ReadGroupCount_Req) GetGroupby() *structpb.Value {
	if x != nil {
		return x.Groupby
	}
	return nil
}

func (x *UserRole_ReadGroupCount_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *UserRole_ReadGroupCount_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type UserRole_ReadGroupCount_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        float64                `protobuf:"fixed64,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_ReadGroupCount_Resp) Reset() {
	*x = UserRole_ReadGroupCount_Resp{}
	mi := &file_auth_proto_msgTypes[407]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_ReadGroupCount_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_ReadGroupCount_Resp) ProtoMessage() {}

func (x *UserRole_ReadGroupCount_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[407]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_ReadGroupCount_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_ReadGroupCount_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{407}
}

func (x *UserRole_ReadGroupCount_Resp) GetResult() float64 {
	if x != nil {
		return x.Result
	}
	return 0
}

type UserRole_Search_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Options       *structpb.Value        `protobuf:"bytes,2,opt,name=options,proto3" json:"options,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Search_Req) Reset() {
	*x = UserRole_Search_Req{}
	mi := &file_auth_proto_msgTypes[408]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Search_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Search_Req) ProtoMessage() {}

func (x *UserRole_Search_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[408]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Search_Req.ProtoReflect.Descriptor instead.
func (*UserRole_Search_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{408}
}

func (x *UserRole_Search_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *UserRole_Search_Req) GetOptions() *structpb.Value {
	if x != nil {
		return x.Options
	}
	return nil
}

type UserRole_Search_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Search_Resp) Reset() {
	*x = UserRole_Search_Resp{}
	mi := &file_auth_proto_msgTypes[409]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Search_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Search_Resp) ProtoMessage() {}

func (x *UserRole_Search_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[409]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Search_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_Search_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{409}
}

func (x *UserRole_Search_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type UserRole_Update_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Condition     *structpb.Value        `protobuf:"bytes,1,opt,name=condition,proto3" json:"condition,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Update_Req) Reset() {
	*x = UserRole_Update_Req{}
	mi := &file_auth_proto_msgTypes[410]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Update_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Update_Req) ProtoMessage() {}

func (x *UserRole_Update_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[410]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Update_Req.ProtoReflect.Descriptor instead.
func (*UserRole_Update_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{410}
}

func (x *UserRole_Update_Req) GetCondition() *structpb.Value {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *UserRole_Update_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *UserRole_Update_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type UserRole_Update_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_Update_Resp) Reset() {
	*x = UserRole_Update_Resp{}
	mi := &file_auth_proto_msgTypes[411]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_Update_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_Update_Resp) ProtoMessage() {}

func (x *UserRole_Update_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[411]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_Update_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_Update_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{411}
}

func (x *UserRole_Update_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

type UserRole_UpdateById_Req struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Values        *structpb.Value        `protobuf:"bytes,2,opt,name=values,proto3" json:"values,omitempty"`
	ReturnFields  *structpb.Value        `protobuf:"bytes,3,opt,name=returnFields,proto3" json:"returnFields,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_UpdateById_Req) Reset() {
	*x = UserRole_UpdateById_Req{}
	mi := &file_auth_proto_msgTypes[412]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_UpdateById_Req) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_UpdateById_Req) ProtoMessage() {}

func (x *UserRole_UpdateById_Req) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[412]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_UpdateById_Req.ProtoReflect.Descriptor instead.
func (*UserRole_UpdateById_Req) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{412}
}

func (x *UserRole_UpdateById_Req) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *UserRole_UpdateById_Req) GetValues() *structpb.Value {
	if x != nil {
		return x.Values
	}
	return nil
}

func (x *UserRole_UpdateById_Req) GetReturnFields() *structpb.Value {
	if x != nil {
		return x.ReturnFields
	}
	return nil
}

type UserRole_UpdateById_Resp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Result        *structpb.Value        `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserRole_UpdateById_Resp) Reset() {
	*x = UserRole_UpdateById_Resp{}
	mi := &file_auth_proto_msgTypes[413]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserRole_UpdateById_Resp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserRole_UpdateById_Resp) ProtoMessage() {}

func (x *UserRole_UpdateById_Resp) ProtoReflect() protoreflect.Message {
	mi := &file_auth_proto_msgTypes[413]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserRole_UpdateById_Resp.ProtoReflect.Descriptor instead.
func (*UserRole_UpdateById_Resp) Descriptor() ([]byte, []int) {
	return file_auth_proto_rawDescGZIP(), []int{413}
}

func (x *UserRole_UpdateById_Resp) GetResult() *structpb.Value {
	if x != nil {
		return x.Result
	}
	return nil
}

var File_auth_proto protoreflect.FileDescriptor

const file_auth_proto_rawDesc = "" +
	"\n" +
	"\n" +
	"auth.proto\x12\x04auth\x1a\x1cgoogle/protobuf/struct.proto\x1a\x1bgoogle/protobuf/empty.proto\"U\n" +
	"\x13Language_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"F\n" +
	"\x14Language_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"s\n" +
	"\x17Language_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"J\n" +
	"\x18Language_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"J\n" +
	"\x12Language_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"-\n" +
	"\x13Language_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x7f\n" +
	"\x13Language_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"F\n" +
	"\x14Language_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x85\x01\n" +
	"\x17Language_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"J\n" +
	"\x18Language_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"G\n" +
	"\x17Language_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"J\n" +
	"\x18Language_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"K\n" +
	"\x13Language_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\".\n" +
	"\x14Language_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\")\n" +
	"\x17Language_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"2\n" +
	"\x18Language_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\xa3\x01\n" +
	"\x15Language_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"H\n" +
	"\x16Language_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb2\x01\n" +
	"\x16Language_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"I\n" +
	"\x17Language_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb7\x01\n" +
	"\x1bLanguage_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"6\n" +
	"\x1cLanguage_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"}\n" +
	"\x13Language_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"F\n" +
	"\x14Language_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb7\x01\n" +
	"\x13Language_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"F\n" +
	"\x14Language_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x95\x01\n" +
	"\x17Language_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"J\n" +
	"\x18Language_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"U\n" +
	"\x13Location_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"F\n" +
	"\x14Location_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"s\n" +
	"\x17Location_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"J\n" +
	"\x18Location_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"J\n" +
	"\x12Location_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"-\n" +
	"\x13Location_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x7f\n" +
	"\x13Location_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"F\n" +
	"\x14Location_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x85\x01\n" +
	"\x17Location_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"J\n" +
	"\x18Location_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"G\n" +
	"\x17Location_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"J\n" +
	"\x18Location_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"K\n" +
	"\x13Location_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\".\n" +
	"\x14Location_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\")\n" +
	"\x17Location_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"2\n" +
	"\x18Location_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\xa3\x01\n" +
	"\x15Location_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"H\n" +
	"\x16Location_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb2\x01\n" +
	"\x16Location_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"I\n" +
	"\x17Location_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb7\x01\n" +
	"\x1bLocation_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"6\n" +
	"\x1cLocation_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"'\n" +
	"\x15Location_Register_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"H\n" +
	"\x16Location_Register_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"}\n" +
	"\x13Location_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"F\n" +
	"\x14Location_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb7\x01\n" +
	"\x13Location_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"F\n" +
	"\x14Location_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x95\x01\n" +
	"\x17Location_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"J\n" +
	"\x18Location_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"R\n" +
	"\x10Order_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"C\n" +
	"\x11Order_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"p\n" +
	"\x14Order_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"G\n" +
	"\x15Order_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"G\n" +
	"\x0fOrder_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"*\n" +
	"\x10Order_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"|\n" +
	"\x10Order_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"C\n" +
	"\x11Order_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x82\x01\n" +
	"\x14Order_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"G\n" +
	"\x15Order_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"D\n" +
	"\x14Order_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"G\n" +
	"\x15Order_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"H\n" +
	"\x10Order_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"+\n" +
	"\x11Order_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"&\n" +
	"\x14Order_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"/\n" +
	"\x15Order_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\xa0\x01\n" +
	"\x12Order_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"E\n" +
	"\x13Order_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xaf\x01\n" +
	"\x13Order_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"F\n" +
	"\x14Order_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb4\x01\n" +
	"\x18Order_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"3\n" +
	"\x19Order_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"z\n" +
	"\x10Order_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"C\n" +
	"\x11Order_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb4\x01\n" +
	"\x10Order_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"C\n" +
	"\x11Order_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x92\x01\n" +
	"\x14Order_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"G\n" +
	"\x15Order_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"V\n" +
	"\x14OrderLine_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"G\n" +
	"\x15OrderLine_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"t\n" +
	"\x18OrderLine_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"K\n" +
	"\x19OrderLine_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"K\n" +
	"\x13OrderLine_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\".\n" +
	"\x14OrderLine_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x80\x01\n" +
	"\x14OrderLine_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"G\n" +
	"\x15OrderLine_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x86\x01\n" +
	"\x18OrderLine_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"K\n" +
	"\x19OrderLine_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"H\n" +
	"\x18OrderLine_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"K\n" +
	"\x19OrderLine_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"L\n" +
	"\x14OrderLine_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"/\n" +
	"\x15OrderLine_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"*\n" +
	"\x18OrderLine_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"3\n" +
	"\x19OrderLine_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\xa4\x01\n" +
	"\x16OrderLine_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"I\n" +
	"\x17OrderLine_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb3\x01\n" +
	"\x17OrderLine_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"J\n" +
	"\x18OrderLine_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb8\x01\n" +
	"\x1cOrderLine_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"7\n" +
	"\x1dOrderLine_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"~\n" +
	"\x14OrderLine_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"G\n" +
	"\x15OrderLine_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb8\x01\n" +
	"\x14OrderLine_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"G\n" +
	"\x15OrderLine_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x96\x01\n" +
	"\x18OrderLine_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"K\n" +
	"\x19OrderLine_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"Q\n" +
	"\x0fRole_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"B\n" +
	"\x10Role_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"o\n" +
	"\x13Role_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"F\n" +
	"\x14Role_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"F\n" +
	"\x0eRole_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\")\n" +
	"\x0fRole_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"{\n" +
	"\x0fRole_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"B\n" +
	"\x10Role_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"P\n" +
	"\x1aRole_CreateIfNotExists_Req\x122\n" +
	"\broleData\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\broleData\"5\n" +
	"\x1bRole_CreateIfNotExists_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\tR\x06result\"\x81\x01\n" +
	"\x13Role_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"F\n" +
	"\x14Role_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"C\n" +
	"\x13Role_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"F\n" +
	"\x14Role_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"G\n" +
	"\x0fRole_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"*\n" +
	"\x10Role_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"%\n" +
	"\x13Role_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\".\n" +
	"\x14Role_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x9f\x01\n" +
	"\x11Role_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"D\n" +
	"\x12Role_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xae\x01\n" +
	"\x12Role_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"E\n" +
	"\x13Role_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb3\x01\n" +
	"\x17Role_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"2\n" +
	"\x18Role_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"y\n" +
	"\x0fRole_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"B\n" +
	"\x10Role_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb3\x01\n" +
	"\x0fRole_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"B\n" +
	"\x10Role_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x91\x01\n" +
	"\x13Role_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"F\n" +
	"\x14Role_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"Z\n" +
	"\x18RoleFieldRule_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"K\n" +
	"\x19RoleFieldRule_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"x\n" +
	"\x1cRoleFieldRule_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"O\n" +
	"\x1dRoleFieldRule_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"O\n" +
	"\x17RoleFieldRule_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"2\n" +
	"\x18RoleFieldRule_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x84\x01\n" +
	"\x18RoleFieldRule_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"K\n" +
	"\x19RoleFieldRule_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x8a\x01\n" +
	"\x1cRoleFieldRule_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"O\n" +
	"\x1dRoleFieldRule_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"L\n" +
	"\x1cRoleFieldRule_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"O\n" +
	"\x1dRoleFieldRule_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"P\n" +
	"\x18RoleFieldRule_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"3\n" +
	"\x19RoleFieldRule_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\".\n" +
	"\x1cRoleFieldRule_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"7\n" +
	"\x1dRoleFieldRule_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\xa8\x01\n" +
	"\x1aRoleFieldRule_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"M\n" +
	"\x1bRoleFieldRule_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb7\x01\n" +
	"\x1bRoleFieldRule_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"N\n" +
	"\x1cRoleFieldRule_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xbc\x01\n" +
	" RoleFieldRule_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\";\n" +
	"!RoleFieldRule_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x82\x01\n" +
	"\x18RoleFieldRule_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"K\n" +
	"\x19RoleFieldRule_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xbc\x01\n" +
	"\x18RoleFieldRule_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"K\n" +
	"\x19RoleFieldRule_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x9a\x01\n" +
	"\x1cRoleFieldRule_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"O\n" +
	"\x1dRoleFieldRule_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\\\n" +
	"\x1aRoleInheritance_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"M\n" +
	"\x1bRoleInheritance_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"z\n" +
	"\x1eRoleInheritance_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"Q\n" +
	"\x1fRoleInheritance_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"Q\n" +
	"\x19RoleInheritance_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"4\n" +
	"\x1aRoleInheritance_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x86\x01\n" +
	"\x1aRoleInheritance_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"M\n" +
	"\x1bRoleInheritance_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x8c\x01\n" +
	"\x1eRoleInheritance_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"Q\n" +
	"\x1fRoleInheritance_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"N\n" +
	"\x1eRoleInheritance_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"Q\n" +
	"\x1fRoleInheritance_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"R\n" +
	"\x1aRoleInheritance_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"5\n" +
	"\x1bRoleInheritance_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"0\n" +
	"\x1eRoleInheritance_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"9\n" +
	"\x1fRoleInheritance_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\xaa\x01\n" +
	"\x1cRoleInheritance_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"O\n" +
	"\x1dRoleInheritance_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb9\x01\n" +
	"\x1dRoleInheritance_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"P\n" +
	"\x1eRoleInheritance_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xbe\x01\n" +
	"\"RoleInheritance_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"=\n" +
	"#RoleInheritance_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x84\x01\n" +
	"\x1aRoleInheritance_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"M\n" +
	"\x1bRoleInheritance_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xbe\x01\n" +
	"\x1aRoleInheritance_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"M\n" +
	"\x1bRoleInheritance_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x9c\x01\n" +
	"\x1eRoleInheritance_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"Q\n" +
	"\x1fRoleInheritance_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"]\n" +
	"\x1bRoleMethodAccess_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"N\n" +
	"\x1cRoleMethodAccess_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"{\n" +
	"\x1fRoleMethodAccess_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"R\n" +
	" RoleMethodAccess_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"R\n" +
	"\x1aRoleMethodAccess_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"5\n" +
	"\x1bRoleMethodAccess_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x87\x01\n" +
	"\x1bRoleMethodAccess_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"N\n" +
	"\x1cRoleMethodAccess_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x8d\x01\n" +
	"\x1fRoleMethodAccess_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"R\n" +
	" RoleMethodAccess_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"O\n" +
	"\x1fRoleMethodAccess_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"R\n" +
	" RoleMethodAccess_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"S\n" +
	"\x1bRoleMethodAccess_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"6\n" +
	"\x1cRoleMethodAccess_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"1\n" +
	"\x1fRoleMethodAccess_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\":\n" +
	" RoleMethodAccess_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\xab\x01\n" +
	"\x1dRoleMethodAccess_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"P\n" +
	"\x1eRoleMethodAccess_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xba\x01\n" +
	"\x1eRoleMethodAccess_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"Q\n" +
	"\x1fRoleMethodAccess_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xbf\x01\n" +
	"#RoleMethodAccess_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\">\n" +
	"$RoleMethodAccess_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x85\x01\n" +
	"\x1bRoleMethodAccess_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"N\n" +
	"\x1cRoleMethodAccess_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xbf\x01\n" +
	"\x1bRoleMethodAccess_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"N\n" +
	"\x1cRoleMethodAccess_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x9d\x01\n" +
	"\x1fRoleMethodAccess_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"R\n" +
	" RoleMethodAccess_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"[\n" +
	"\x19RoleRecordRule_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"L\n" +
	"\x1aRoleRecordRule_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"y\n" +
	"\x1dRoleRecordRule_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"P\n" +
	"\x1eRoleRecordRule_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"P\n" +
	"\x18RoleRecordRule_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"3\n" +
	"\x19RoleRecordRule_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x85\x01\n" +
	"\x19RoleRecordRule_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"L\n" +
	"\x1aRoleRecordRule_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x8b\x01\n" +
	"\x1dRoleRecordRule_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"P\n" +
	"\x1eRoleRecordRule_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"M\n" +
	"\x1dRoleRecordRule_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"P\n" +
	"\x1eRoleRecordRule_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"Q\n" +
	"\x19RoleRecordRule_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"4\n" +
	"\x1aRoleRecordRule_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"/\n" +
	"\x1dRoleRecordRule_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"8\n" +
	"\x1eRoleRecordRule_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\xa9\x01\n" +
	"\x1bRoleRecordRule_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"N\n" +
	"\x1cRoleRecordRule_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb8\x01\n" +
	"\x1cRoleRecordRule_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"O\n" +
	"\x1dRoleRecordRule_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xbd\x01\n" +
	"!RoleRecordRule_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"<\n" +
	"\"RoleRecordRule_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x83\x01\n" +
	"\x19RoleRecordRule_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"L\n" +
	"\x1aRoleRecordRule_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xbd\x01\n" +
	"\x19RoleRecordRule_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"L\n" +
	"\x1aRoleRecordRule_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x9b\x01\n" +
	"\x1dRoleRecordRule_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"P\n" +
	"\x1eRoleRecordRule_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"T\n" +
	"\x12Session_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"E\n" +
	"\x13Session_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"r\n" +
	"\x16Session_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"I\n" +
	"\x17Session_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\";\n" +
	"!Session_CleanExpiredSessions_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"I\n" +
	"\x11Session_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\",\n" +
	"\x12Session_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"~\n" +
	"\x12Session_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"E\n" +
	"\x13Session_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x84\x01\n" +
	"\x16Session_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"I\n" +
	"\x17Session_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"F\n" +
	"\x16Session_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"I\n" +
	"\x17Session_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"J\n" +
	"\x12Session_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"-\n" +
	"\x13Session_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"(\n" +
	"\x16Session_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"1\n" +
	"\x17Session_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\">\n" +
	"$Session_GetActiveSessionsForUser_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\"W\n" +
	"%Session_GetActiveSessionsForUser_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xa2\x01\n" +
	"\x14Session_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"G\n" +
	"\x15Session_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb1\x01\n" +
	"\x15Session_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"H\n" +
	"\x16Session_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb6\x01\n" +
	"\x1aSession_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"5\n" +
	"\x1bSession_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"`\n" +
	"\x1cSession_RevokeAllForUser_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x12(\n" +
	"\x0fexceptSessionId\x18\x02 \x01(\tR\x0fexceptSessionId\"7\n" +
	"\x1dSession_RevokeAllForUser_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"9\n" +
	"\x19Session_RevokeSession_Req\x12\x1c\n" +
	"\tsessionId\x18\x01 \x01(\tR\tsessionId\"4\n" +
	"\x1aSession_RevokeSession_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\bR\x06result\"|\n" +
	"\x12Session_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"E\n" +
	"\x13Session_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb6\x01\n" +
	"\x12Session_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"E\n" +
	"\x13Session_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x94\x01\n" +
	"\x16Session_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"I\n" +
	"\x17Session_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"1\n" +
	"\x19Session_ValidateToken_Req\x12\x14\n" +
	"\x05token\x18\x01 \x01(\tR\x05token\"L\n" +
	"\x1aSession_ValidateToken_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"R\n" +
	"\x10Token_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"C\n" +
	"\x11Token_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"p\n" +
	"\x14Token_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"G\n" +
	"\x15Token_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"7\n" +
	"\x1dToken_CleanExpiredTokens_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"G\n" +
	"\x0fToken_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"*\n" +
	"\x10Token_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"|\n" +
	"\x10Token_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"C\n" +
	"\x11Token_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x82\x01\n" +
	"\x14Token_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"G\n" +
	"\x15Token_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"g\n" +
	"\x19Token_CreateTokenPair_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x122\n" +
	"\bmetadata\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\bmetadata\"L\n" +
	"\x1aToken_CreateTokenPair_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"D\n" +
	"\x14Token_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"G\n" +
	"\x15Token_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"H\n" +
	"\x10Token_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"+\n" +
	"\x11Token_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"&\n" +
	"\x14Token_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"/\n" +
	"\x15Token_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\xa0\x01\n" +
	"\x12Token_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"E\n" +
	"\x13Token_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xaf\x01\n" +
	"\x13Token_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"F\n" +
	"\x14Token_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb4\x01\n" +
	"\x18Token_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"3\n" +
	"\x19Token_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"q\n" +
	"\x17Token_RefreshTokens_Req\x12\"\n" +
	"\frefreshToken\x18\x01 \x01(\tR\frefreshToken\x122\n" +
	"\bmetadata\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\bmetadata\"J\n" +
	"\x18Token_RefreshTokens_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"u\n" +
	"\x1dToken_RevokeAllUserTokens_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x12$\n" +
	"\rexceptTokenId\x18\x02 \x01(\tR\rexceptTokenId\x12\x16\n" +
	"\x06reason\x18\x03 \x01(\tR\x06reason\"8\n" +
	"\x1eToken_RevokeAllUserTokens_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"E\n" +
	"\x15Token_RevokeToken_Req\x12\x14\n" +
	"\x05token\x18\x01 \x01(\tR\x05token\x12\x16\n" +
	"\x06reason\x18\x02 \x01(\tR\x06reason\"0\n" +
	"\x16Token_RevokeToken_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\bR\x06result\"R\n" +
	" Token_RevokeUserAccessTokens_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x12\x16\n" +
	"\x06reason\x18\x02 \x01(\tR\x06reason\";\n" +
	"!Token_RevokeUserAccessTokens_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"z\n" +
	"\x10Token_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"C\n" +
	"\x11Token_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb4\x01\n" +
	"\x10Token_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"C\n" +
	"\x11Token_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x92\x01\n" +
	"\x14Token_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"G\n" +
	"\x15Token_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"M\n" +
	"\x17Token_ValidateToken_Req\x12\x14\n" +
	"\x05token\x18\x01 \x01(\tR\x05token\x12\x1c\n" +
	"\ttokenType\x18\x02 \x01(\tR\ttokenType\"J\n" +
	"\x18Token_ValidateToken_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"`\n" +
	"\x14User_AssignRoles_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x120\n" +
	"\aroleIds\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aroleIds\"/\n" +
	"\x15User_AssignRoles_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\bR\x06result\"Q\n" +
	"\x0fUser_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"B\n" +
	"\x10User_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"o\n" +
	"\x13User_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"F\n" +
	"\x14User_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"}\n" +
	"\x17User_ChangePassword_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x12(\n" +
	"\x0fcurrentPassword\x18\x02 \x01(\tR\x0fcurrentPassword\x12 \n" +
	"\vnewPassword\x18\x03 \x01(\tR\vnewPassword\"2\n" +
	"\x18User_ChangePassword_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\bR\x06result\"d\n" +
	"\x1aUser_CheckMethodAccess_Req\x12\x1c\n" +
	"\tcompanyId\x18\x01 \x01(\tR\tcompanyId\x12(\n" +
	"\x0fserviceFullName\x18\x02 \x01(\tR\x0fserviceFullName\"5\n" +
	"\x1bUser_CheckMethodAccess_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\bR\x06result\"F\n" +
	"\x0eUser_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\")\n" +
	"\x0fUser_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"{\n" +
	"\x0fUser_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"B\n" +
	"\x10User_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x81\x01\n" +
	"\x13User_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"F\n" +
	"\x14User_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"C\n" +
	"\x13User_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"F\n" +
	"\x14User_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"G\n" +
	"\x0fUser_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"*\n" +
	"\x10User_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"%\n" +
	"\x13User_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\".\n" +
	"\x14User_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"G\n" +
	"\x1fUser_GetRecordRuleCondition_Req\x12\x14\n" +
	"\x05model\x18\x01 \x01(\tR\x05model\x12\x0e\n" +
	"\x02op\x18\x02 \x01(\tR\x02op\"R\n" +
	" User_GetRecordRuleCondition_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"X\n" +
	"\x16User_HasPermission_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x12&\n" +
	"\x0epermissionCode\x18\x02 \x01(\tR\x0epermissionCode\"1\n" +
	"\x17User_HasPermission_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\bR\x06result\"F\n" +
	"\x10User_HasRole_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x12\x1a\n" +
	"\broleCode\x18\x02 \x01(\tR\broleCode\"+\n" +
	"\x11User_HasRole_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\bR\x06result\"\xb4\x01\n" +
	"\x0eUser_Login_Req\x12(\n" +
	"\x0fusernameOrEmail\x18\x01 \x01(\tR\x0fusernameOrEmail\x12\x1a\n" +
	"\bpassword\x18\x02 \x01(\tR\bpassword\x12\x1c\n" +
	"\tipAddress\x18\x03 \x01(\tR\tipAddress\x12\x1e\n" +
	"\n" +
	"deviceInfo\x18\x04 \x01(\tR\n" +
	"deviceInfo\x12\x1e\n" +
	"\n" +
	"rememberMe\x18\x05 \x01(\bR\n" +
	"rememberMe\"A\n" +
	"\x0fUser_Login_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"g\n" +
	"\x0fUser_Logout_Req\x12\x14\n" +
	"\x05token\x18\x01 \x01(\tR\x05token\x12\x1e\n" +
	"\n" +
	"allDevices\x18\x02 \x01(\bR\n" +
	"allDevices\x12\x1e\n" +
	"\n" +
	"deviceInfo\x18\x03 \x01(\tR\n" +
	"deviceInfo\"*\n" +
	"\x10User_Logout_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\bR\x06result\"\x9f\x01\n" +
	"\x11User_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"D\n" +
	"\x12User_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xae\x01\n" +
	"\x12User_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"E\n" +
	"\x13User_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb3\x01\n" +
	"\x17User_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"2\n" +
	"\x18User_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"<\n" +
	"\x16User_RefreshTokens_Req\x12\"\n" +
	"\frefreshToken\x18\x01 \x01(\tR\frefreshToken\"I\n" +
	"\x17User_RefreshTokens_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"c\n" +
	"\x11User_Register_Req\x122\n" +
	"\buserData\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\buserData\x12\x1a\n" +
	"\bpassword\x18\x02 \x01(\tR\bpassword\",\n" +
	"\x12User_Register_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\tR\x06result\"`\n" +
	"\x14User_RemoveRoles_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x120\n" +
	"\aroleIds\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aroleIds\"/\n" +
	"\x15User_RemoveRoles_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\bR\x06result\"R\n" +
	"\x16User_ResetPassword_Req\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\tR\x06userId\x12 \n" +
	"\vnewPassword\x18\x02 \x01(\tR\vnewPassword\"1\n" +
	"\x17User_ResetPassword_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\bR\x06result\"y\n" +
	"\x0fUser_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"B\n" +
	"\x10User_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb3\x01\n" +
	"\x0fUser_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"B\n" +
	"\x10User_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x91\x01\n" +
	"\x13User_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"F\n" +
	"\x14User_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"U\n" +
	"\x13UserRole_Browse_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"F\n" +
	"\x14UserRole_Browse_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"s\n" +
	"\x17UserRole_BrowseMany_Req\x12(\n" +
	"\x03ids\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x03ids\x12.\n" +
	"\x06fields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06fields\"J\n" +
	"\x18UserRole_BrowseMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"J\n" +
	"\x12UserRole_Count_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\"-\n" +
	"\x13UserRole_Count_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\x7f\n" +
	"\x13UserRole_Create_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"F\n" +
	"\x14UserRole_Create_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x85\x01\n" +
	"\x17UserRole_CreateMany_Req\x12.\n" +
	"\x06values\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"J\n" +
	"\x18UserRole_CreateMany_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"G\n" +
	"\x17UserRole_DefaultGet_Req\x12,\n" +
	"\x05value\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05value\"J\n" +
	"\x18UserRole_DefaultGet_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"K\n" +
	"\x13UserRole_Delete_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\".\n" +
	"\x14UserRole_Delete_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\")\n" +
	"\x17UserRole_DeleteById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\"2\n" +
	"\x18UserRole_DeleteById_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"\xa3\x01\n" +
	"\x15UserRole_Onchange_Req\x12,\n" +
	"\x05draft\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x05draft\x120\n" +
	"\achanged\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\achanged\x12*\n" +
	"\x04opts\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\x04opts\"H\n" +
	"\x16UserRole_Onchange_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb2\x01\n" +
	"\x16UserRole_ReadGroup_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"I\n" +
	"\x17UserRole_ReadGroup_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb7\x01\n" +
	"\x1bUserRole_ReadGroupCount_Req\x120\n" +
	"\agroupby\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\agroupby\x124\n" +
	"\tcondition\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"6\n" +
	"\x1cUserRole_ReadGroupCount_Resp\x12\x16\n" +
	"\x06result\x18\x01 \x01(\x01R\x06result\"}\n" +
	"\x13UserRole_Search_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x120\n" +
	"\aoptions\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\aoptions\"F\n" +
	"\x14UserRole_Search_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\xb7\x01\n" +
	"\x13UserRole_Update_Req\x124\n" +
	"\tcondition\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\tcondition\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"F\n" +
	"\x14UserRole_Update_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result\"\x95\x01\n" +
	"\x17UserRole_UpdateById_Req\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12.\n" +
	"\x06values\x18\x02 \x01(\v2\x16.google.protobuf.ValueR\x06values\x12:\n" +
	"\freturnFields\x18\x03 \x01(\v2\x16.google.protobuf.ValueR\freturnFields\"J\n" +
	"\x18UserRole_UpdateById_Resp\x12.\n" +
	"\x06result\x18\x01 \x01(\v2\x16.google.protobuf.ValueR\x06result2\x94\b\n" +
	"\bLanguage\x12A\n" +
	"\x06Browse\x12\x19.auth.Language_Browse_Req\x1a\x1a.auth.Language_Browse_Resp\"\x00\x12M\n" +
	"\n" +
	"BrowseMany\x12\x1d.auth.Language_BrowseMany_Req\x1a\x1e.auth.Language_BrowseMany_Resp\"\x00\x12>\n" +
	"\x05Count\x12\x18.auth.Language_Count_Req\x1a\x19.auth.Language_Count_Resp\"\x00\x12A\n" +
	"\x06Create\x12\x19.auth.Language_Create_Req\x1a\x1a.auth.Language_Create_Resp\"\x00\x12M\n" +
	"\n" +
	"CreateMany\x12\x1d.auth.Language_CreateMany_Req\x1a\x1e.auth.Language_CreateMany_Resp\"\x00\x12M\n" +
	"\n" +
	"DefaultGet\x12\x1d.auth.Language_DefaultGet_Req\x1a\x1e.auth.Language_DefaultGet_Resp\"\x00\x12A\n" +
	"\x06Delete\x12\x19.auth.Language_Delete_Req\x1a\x1a.auth.Language_Delete_Resp\"\x00\x12M\n" +
	"\n" +
	"DeleteById\x12\x1d.auth.Language_DeleteById_Req\x1a\x1e.auth.Language_DeleteById_Resp\"\x00\x12G\n" +
	"\bOnchange\x12\x1b.auth.Language_Onchange_Req\x1a\x1c.auth.Language_Onchange_Resp\"\x00\x12J\n" +
	"\tReadGroup\x12\x1c.auth.Language_ReadGroup_Req\x1a\x1d.auth.Language_ReadGroup_Resp\"\x00\x12Y\n" +
	"\x0eReadGroupCount\x12!.auth.Language_ReadGroupCount_Req\x1a\".auth.Language_ReadGroupCount_Resp\"\x00\x12A\n" +
	"\x06Search\x12\x19.auth.Language_Search_Req\x1a\x1a.auth.Language_Search_Resp\"\x00\x12A\n" +
	"\x06Update\x12\x19.auth.Language_Update_Req\x1a\x1a.auth.Language_Update_Resp\"\x00\x12M\n" +
	"\n" +
	"UpdateById\x12\x1d.auth.Language_UpdateById_Req\x1a\x1e.auth.Language_UpdateById_Resp\"\x002\xdd\b\n" +
	"\bLocation\x12A\n" +
	"\x06Browse\x12\x19.auth.Location_Browse_Req\x1a\x1a.auth.Location_Browse_Resp\"\x00\x12M\n" +
	"\n" +
	"BrowseMany\x12\x1d.auth.Location_BrowseMany_Req\x1a\x1e.auth.Location_BrowseMany_Resp\"\x00\x12>\n" +
	"\x05Count\x12\x18.auth.Location_Count_Req\x1a\x19.auth.Location_Count_Resp\"\x00\x12A\n" +
	"\x06Create\x12\x19.auth.Location_Create_Req\x1a\x1a.auth.Location_Create_Resp\"\x00\x12M\n" +
	"\n" +
	"CreateMany\x12\x1d.auth.Location_CreateMany_Req\x1a\x1e.auth.Location_CreateMany_Resp\"\x00\x12M\n" +
	"\n" +
	"DefaultGet\x12\x1d.auth.Location_DefaultGet_Req\x1a\x1e.auth.Location_DefaultGet_Resp\"\x00\x12A\n" +
	"\x06Delete\x12\x19.auth.Location_Delete_Req\x1a\x1a.auth.Location_Delete_Resp\"\x00\x12M\n" +
	"\n" +
	"DeleteById\x12\x1d.auth.Location_DeleteById_Req\x1a\x1e.auth.Location_DeleteById_Resp\"\x00\x12G\n" +
	"\bOnchange\x12\x1b.auth.Location_Onchange_Req\x1a\x1c.auth.Location_Onchange_Resp\"\x00\x12J\n" +
	"\tReadGroup\x12\x1c.auth.Location_ReadGroup_Req\x1a\x1d.auth.Location_ReadGroup_Resp\"\x00\x12Y\n" +
	"\x0eReadGroupCount\x12!.auth.Location_ReadGroupCount_Req\x1a\".auth.Location_ReadGroupCount_Resp\"\x00\x12G\n" +
	"\bRegister\x12\x1b.auth.Location_Register_Req\x1a\x1c.auth.Location_Register_Resp\"\x00\x12A\n" +
	"\x06Search\x12\x19.auth.Location_Search_Req\x1a\x1a.auth.Location_Search_Resp\"\x00\x12A\n" +
	"\x06Update\x12\x19.auth.Location_Update_Req\x1a\x1a.auth.Location_Update_Resp\"\x00\x12M\n" +
	"\n" +
	"UpdateById\x12\x1d.auth.Location_UpdateById_Req\x1a\x1e.auth.Location_UpdateById_Resp\"\x002\xbd\a\n" +
	"\x05Order\x12;\n" +
	"\x06Browse\x12\x16.auth.Order_Browse_Req\x1a\x17.auth.Order_Browse_Resp\"\x00\x12G\n" +
	"\n" +
	"BrowseMany\x12\x1a.auth.Order_BrowseMany_Req\x1a\x1b.auth.Order_BrowseMany_Resp\"\x00\x128\n" +
	"\x05Count\x12\x15.auth.Order_Count_Req\x1a\x16.auth.Order_Count_Resp\"\x00\x12;\n" +
	"\x06Create\x12\x16.auth.Order_Create_Req\x1a\x17.auth.Order_Create_Resp\"\x00\x12G\n" +
	"\n" +
	"CreateMany\x12\x1a.auth.Order_CreateMany_Req\x1a\x1b.auth.Order_CreateMany_Resp\"\x00\x12G\n" +
	"\n" +
	"DefaultGet\x12\x1a.auth.Order_DefaultGet_Req\x1a\x1b.auth.Order_DefaultGet_Resp\"\x00\x12;\n" +
	"\x06Delete\x12\x16.auth.Order_Delete_Req\x1a\x17.auth.Order_Delete_Resp\"\x00\x12G\n" +
	"\n" +
	"DeleteById\x12\x1a.auth.Order_DeleteById_Req\x1a\x1b.auth.Order_DeleteById_Resp\"\x00\x12A\n" +
	"\bOnchange\x12\x18.auth.Order_Onchange_Req\x1a\x19.auth.Order_Onchange_Resp\"\x00\x12D\n" +
	"\tReadGroup\x12\x19.auth.Order_ReadGroup_Req\x1a\x1a.auth.Order_ReadGroup_Resp\"\x00\x12S\n" +
	"\x0eReadGroupCount\x12\x1e.auth.Order_ReadGroupCount_Req\x1a\x1f.auth.Order_ReadGroupCount_Resp\"\x00\x12;\n" +
	"\x06Search\x12\x16.auth.Order_Search_Req\x1a\x17.auth.Order_Search_Resp\"\x00\x12;\n" +
	"\x06Update\x12\x16.auth.Order_Update_Req\x1a\x17.auth.Order_Update_Resp\"\x00\x12G\n" +
	"\n" +
	"UpdateById\x12\x1a.auth.Order_UpdateById_Req\x1a\x1b.auth.Order_UpdateById_Resp\"\x002\xb1\b\n" +
	"\tOrderLine\x12C\n" +
	"\x06Browse\x12\x1a.auth.OrderLine_Browse_Req\x1a\x1b.auth.OrderLine_Browse_Resp\"\x00\x12O\n" +
	"\n" +
	"BrowseMany\x12\x1e.auth.OrderLine_BrowseMany_Req\x1a\x1f.auth.OrderLine_BrowseMany_Resp\"\x00\x12@\n" +
	"\x05Count\x12\x19.auth.OrderLine_Count_Req\x1a\x1a.auth.OrderLine_Count_Resp\"\x00\x12C\n" +
	"\x06Create\x12\x1a.auth.OrderLine_Create_Req\x1a\x1b.auth.OrderLine_Create_Resp\"\x00\x12O\n" +
	"\n" +
	"CreateMany\x12\x1e.auth.OrderLine_CreateMany_Req\x1a\x1f.auth.OrderLine_CreateMany_Resp\"\x00\x12O\n" +
	"\n" +
	"DefaultGet\x12\x1e.auth.OrderLine_DefaultGet_Req\x1a\x1f.auth.OrderLine_DefaultGet_Resp\"\x00\x12C\n" +
	"\x06Delete\x12\x1a.auth.OrderLine_Delete_Req\x1a\x1b.auth.OrderLine_Delete_Resp\"\x00\x12O\n" +
	"\n" +
	"DeleteById\x12\x1e.auth.OrderLine_DeleteById_Req\x1a\x1f.auth.OrderLine_DeleteById_Resp\"\x00\x12I\n" +
	"\bOnchange\x12\x1c.auth.OrderLine_Onchange_Req\x1a\x1d.auth.OrderLine_Onchange_Resp\"\x00\x12L\n" +
	"\tReadGroup\x12\x1d.auth.OrderLine_ReadGroup_Req\x1a\x1e.auth.OrderLine_ReadGroup_Resp\"\x00\x12[\n" +
	"\x0eReadGroupCount\x12\".auth.OrderLine_ReadGroupCount_Req\x1a#.auth.OrderLine_ReadGroupCount_Resp\"\x00\x12C\n" +
	"\x06Search\x12\x1a.auth.OrderLine_Search_Req\x1a\x1b.auth.OrderLine_Search_Resp\"\x00\x12C\n" +
	"\x06Update\x12\x1a.auth.OrderLine_Update_Req\x1a\x1b.auth.OrderLine_Update_Resp\"\x00\x12O\n" +
	"\n" +
	"UpdateById\x12\x1e.auth.OrderLine_UpdateById_Req\x1a\x1f.auth.OrderLine_UpdateById_Resp\"\x002\xfc\a\n" +
	"\x04Role\x129\n" +
	"\x06Browse\x12\x15.auth.Role_Browse_Req\x1a\x16.auth.Role_Browse_Resp\"\x00\x12E\n" +
	"\n" +
	"BrowseMany\x12\x19.auth.Role_BrowseMany_Req\x1a\x1a.auth.Role_BrowseMany_Resp\"\x00\x126\n" +
	"\x05Count\x12\x14.auth.Role_Count_Req\x1a\x15.auth.Role_Count_Resp\"\x00\x129\n" +
	"\x06Create\x12\x15.auth.Role_Create_Req\x1a\x16.auth.Role_Create_Resp\"\x00\x12Z\n" +
	"\x11CreateIfNotExists\x12 .auth.Role_CreateIfNotExists_Req\x1a!.auth.Role_CreateIfNotExists_Resp\"\x00\x12E\n" +
	"\n" +
	"CreateMany\x12\x19.auth.Role_CreateMany_Req\x1a\x1a.auth.Role_CreateMany_Resp\"\x00\x12E\n" +
	"\n" +
	"DefaultGet\x12\x19.auth.Role_DefaultGet_Req\x1a\x1a.auth.Role_DefaultGet_Resp\"\x00\x129\n" +
	"\x06Delete\x12\x15.auth.Role_Delete_Req\x1a\x16.auth.Role_Delete_Resp\"\x00\x12E\n" +
	"\n" +
	"DeleteById\x12\x19.auth.Role_DeleteById_Req\x1a\x1a.auth.Role_DeleteById_Resp\"\x00\x12?\n" +
	"\bOnchange\x12\x17.auth.Role_Onchange_Req\x1a\x18.auth.Role_Onchange_Resp\"\x00\x12B\n" +
	"\tReadGroup\x12\x18.auth.Role_ReadGroup_Req\x1a\x19.auth.Role_ReadGroup_Resp\"\x00\x12Q\n" +
	"\x0eReadGroupCount\x12\x1d.auth.Role_ReadGroupCount_Req\x1a\x1e.auth.Role_ReadGroupCount_Resp\"\x00\x129\n" +
	"\x06Search\x12\x15.auth.Role_Search_Req\x1a\x16.auth.Role_Search_Resp\"\x00\x129\n" +
	"\x06Update\x12\x15.auth.Role_Update_Req\x1a\x16.auth.Role_Update_Resp\"\x00\x12E\n" +
	"\n" +
	"UpdateById\x12\x19.auth.Role_UpdateById_Req\x1a\x1a.auth.Role_UpdateById_Resp\"\x002\xa5\t\n" +
	"\rRoleFieldRule\x12K\n" +
	"\x06Browse\x12\x1e.auth.RoleFieldRule_Browse_Req\x1a\x1f.auth.RoleFieldRule_Browse_Resp\"\x00\x12W\n" +
	"\n" +
	"BrowseMany\x12\".auth.RoleFieldRule_BrowseMany_Req\x1a#.auth.RoleFieldRule_BrowseMany_Resp\"\x00\x12H\n" +
	"\x05Count\x12\x1d.auth.RoleFieldRule_Count_Req\x1a\x1e.auth.RoleFieldRule_Count_Resp\"\x00\x12K\n" +
	"\x06Create\x12\x1e.auth.RoleFieldRule_Create_Req\x1a\x1f.auth.RoleFieldRule_Create_Resp\"\x00\x12W\n" +
	"\n" +
	"CreateMany\x12\".auth.RoleFieldRule_CreateMany_Req\x1a#.auth.RoleFieldRule_CreateMany_Resp\"\x00\x12W\n" +
	"\n" +
	"DefaultGet\x12\".auth.RoleFieldRule_DefaultGet_Req\x1a#.auth.RoleFieldRule_DefaultGet_Resp\"\x00\x12K\n" +
	"\x06Delete\x12\x1e.auth.RoleFieldRule_Delete_Req\x1a\x1f.auth.RoleFieldRule_Delete_Resp\"\x00\x12W\n" +
	"\n" +
	"DeleteById\x12\".auth.RoleFieldRule_DeleteById_Req\x1a#.auth.RoleFieldRule_DeleteById_Resp\"\x00\x12Q\n" +
	"\bOnchange\x12 .auth.RoleFieldRule_Onchange_Req\x1a!.auth.RoleFieldRule_Onchange_Resp\"\x00\x12T\n" +
	"\tReadGroup\x12!.auth.RoleFieldRule_ReadGroup_Req\x1a\".auth.RoleFieldRule_ReadGroup_Resp\"\x00\x12c\n" +
	"\x0eReadGroupCount\x12&.auth.RoleFieldRule_ReadGroupCount_Req\x1a'.auth.RoleFieldRule_ReadGroupCount_Resp\"\x00\x12K\n" +
	"\x06Search\x12\x1e.auth.RoleFieldRule_Search_Req\x1a\x1f.auth.RoleFieldRule_Search_Resp\"\x00\x12K\n" +
	"\x06Update\x12\x1e.auth.RoleFieldRule_Update_Req\x1a\x1f.auth.RoleFieldRule_Update_Resp\"\x00\x12W\n" +
	"\n" +
	"UpdateById\x12\".auth.RoleFieldRule_UpdateById_Req\x1a#.auth.RoleFieldRule_UpdateById_Resp\"\x002\xdf\t\n" +
	"\x0fRoleInheritance\x12O\n" +
	"\x06Browse\x12 .auth.RoleInheritance_Browse_Req\x1a!.auth.RoleInheritance_Browse_Resp\"\x00\x12[\n" +
	"\n" +
	"BrowseMany\x12$.auth.RoleInheritance_BrowseMany_Req\x1a%.auth.RoleInheritance_BrowseMany_Resp\"\x00\x12L\n" +
	"\x05Count\x12\x1f.auth.RoleInheritance_Count_Req\x1a .auth.RoleInheritance_Count_Resp\"\x00\x12O\n" +
	"\x06Create\x12 .auth.RoleInheritance_Create_Req\x1a!.auth.RoleInheritance_Create_Resp\"\x00\x12[\n" +
	"\n" +
	"CreateMany\x12$.auth.RoleInheritance_CreateMany_Req\x1a%.auth.RoleInheritance_CreateMany_Resp\"\x00\x12[\n" +
	"\n" +
	"DefaultGet\x12$.auth.RoleInheritance_DefaultGet_Req\x1a%.auth.RoleInheritance_DefaultGet_Resp\"\x00\x12O\n" +
	"\x06Delete\x12 .auth.RoleInheritance_Delete_Req\x1a!.auth.RoleInheritance_Delete_Resp\"\x00\x12[\n" +
	"\n" +
	"DeleteById\x12$.auth.RoleInheritance_DeleteById_Req\x1a%.auth.RoleInheritance_DeleteById_Resp\"\x00\x12U\n" +
	"\bOnchange\x12\".auth.RoleInheritance_Onchange_Req\x1a#.auth.RoleInheritance_Onchange_Resp\"\x00\x12X\n" +
	"\tReadGroup\x12#.auth.RoleInheritance_ReadGroup_Req\x1a$.auth.RoleInheritance_ReadGroup_Resp\"\x00\x12g\n" +
	"\x0eReadGroupCount\x12(.auth.RoleInheritance_ReadGroupCount_Req\x1a).auth.RoleInheritance_ReadGroupCount_Resp\"\x00\x12O\n" +
	"\x06Search\x12 .auth.RoleInheritance_Search_Req\x1a!.auth.RoleInheritance_Search_Resp\"\x00\x12O\n" +
	"\x06Update\x12 .auth.RoleInheritance_Update_Req\x1a!.auth.RoleInheritance_Update_Resp\"\x00\x12[\n" +
	"\n" +
	"UpdateById\x12$.auth.RoleInheritance_UpdateById_Req\x1a%.auth.RoleInheritance_UpdateById_Resp\"\x002\xfc\t\n" +
	"\x10RoleMethodAccess\x12Q\n" +
	"\x06Browse\x12!.auth.RoleMethodAccess_Browse_Req\x1a\".auth.RoleMethodAccess_Browse_Resp\"\x00\x12]\n" +
	"\n" +
	"BrowseMany\x12%.auth.RoleMethodAccess_BrowseMany_Req\x1a&.auth.RoleMethodAccess_BrowseMany_Resp\"\x00\x12N\n" +
	"\x05Count\x12 .auth.RoleMethodAccess_Count_Req\x1a!.auth.RoleMethodAccess_Count_Resp\"\x00\x12Q\n" +
	"\x06Create\x12!.auth.RoleMethodAccess_Create_Req\x1a\".auth.RoleMethodAccess_Create_Resp\"\x00\x12]\n" +
	"\n" +
	"CreateMany\x12%.auth.RoleMethodAccess_CreateMany_Req\x1a&.auth.RoleMethodAccess_CreateMany_Resp\"\x00\x12]\n" +
	"\n" +
	"DefaultGet\x12%.auth.RoleMethodAccess_DefaultGet_Req\x1a&.auth.RoleMethodAccess_DefaultGet_Resp\"\x00\x12Q\n" +
	"\x06Delete\x12!.auth.RoleMethodAccess_Delete_Req\x1a\".auth.RoleMethodAccess_Delete_Resp\"\x00\x12]\n" +
	"\n" +
	"DeleteById\x12%.auth.RoleMethodAccess_DeleteById_Req\x1a&.auth.RoleMethodAccess_DeleteById_Resp\"\x00\x12W\n" +
	"\bOnchange\x12#.auth.RoleMethodAccess_Onchange_Req\x1a$.auth.RoleMethodAccess_Onchange_Resp\"\x00\x12Z\n" +
	"\tReadGroup\x12$.auth.RoleMethodAccess_ReadGroup_Req\x1a%.auth.RoleMethodAccess_ReadGroup_Resp\"\x00\x12i\n" +
	"\x0eReadGroupCount\x12).auth.RoleMethodAccess_ReadGroupCount_Req\x1a*.auth.RoleMethodAccess_ReadGroupCount_Resp\"\x00\x12Q\n" +
	"\x06Search\x12!.auth.RoleMethodAccess_Search_Req\x1a\".auth.RoleMethodAccess_Search_Resp\"\x00\x12Q\n" +
	"\x06Update\x12!.auth.RoleMethodAccess_Update_Req\x1a\".auth.RoleMethodAccess_Update_Resp\"\x00\x12]\n" +
	"\n" +
	"UpdateById\x12%.auth.RoleMethodAccess_UpdateById_Req\x1a&.auth.RoleMethodAccess_UpdateById_Resp\"\x002\xc2\t\n" +
	"\x0eRoleRecordRule\x12M\n" +
	"\x06Browse\x12\x1f.auth.RoleRecordRule_Browse_Req\x1a .auth.RoleRecordRule_Browse_Resp\"\x00\x12Y\n" +
	"\n" +
	"BrowseMany\x12#.auth.RoleRecordRule_BrowseMany_Req\x1a$.auth.RoleRecordRule_BrowseMany_Resp\"\x00\x12J\n" +
	"\x05Count\x12\x1e.auth.RoleRecordRule_Count_Req\x1a\x1f.auth.RoleRecordRule_Count_Resp\"\x00\x12M\n" +
	"\x06Create\x12\x1f.auth.RoleRecordRule_Create_Req\x1a .auth.RoleRecordRule_Create_Resp\"\x00\x12Y\n" +
	"\n" +
	"CreateMany\x12#.auth.RoleRecordRule_CreateMany_Req\x1a$.auth.RoleRecordRule_CreateMany_Resp\"\x00\x12Y\n" +
	"\n" +
	"DefaultGet\x12#.auth.RoleRecordRule_DefaultGet_Req\x1a$.auth.RoleRecordRule_DefaultGet_Resp\"\x00\x12M\n" +
	"\x06Delete\x12\x1f.auth.RoleRecordRule_Delete_Req\x1a .auth.RoleRecordRule_Delete_Resp\"\x00\x12Y\n" +
	"\n" +
	"DeleteById\x12#.auth.RoleRecordRule_DeleteById_Req\x1a$.auth.RoleRecordRule_DeleteById_Resp\"\x00\x12S\n" +
	"\bOnchange\x12!.auth.RoleRecordRule_Onchange_Req\x1a\".auth.RoleRecordRule_Onchange_Resp\"\x00\x12V\n" +
	"\tReadGroup\x12\".auth.RoleRecordRule_ReadGroup_Req\x1a#.auth.RoleRecordRule_ReadGroup_Resp\"\x00\x12e\n" +
	"\x0eReadGroupCount\x12'.auth.RoleRecordRule_ReadGroupCount_Req\x1a(.auth.RoleRecordRule_ReadGroupCount_Resp\"\x00\x12M\n" +
	"\x06Search\x12\x1f.auth.RoleRecordRule_Search_Req\x1a .auth.RoleRecordRule_Search_Resp\"\x00\x12M\n" +
	"\x06Update\x12\x1f.auth.RoleRecordRule_Update_Req\x1a .auth.RoleRecordRule_Update_Resp\"\x00\x12Y\n" +
	"\n" +
	"UpdateById\x12#.auth.RoleRecordRule_UpdateById_Req\x1a$.auth.RoleRecordRule_UpdateById_Resp\"\x002\xd4\v\n" +
	"\aSession\x12?\n" +
	"\x06Browse\x12\x18.auth.Session_Browse_Req\x1a\x19.auth.Session_Browse_Resp\"\x00\x12K\n" +
	"\n" +
	"BrowseMany\x12\x1c.auth.Session_BrowseMany_Req\x1a\x1d.auth.Session_BrowseMany_Resp\"\x00\x12Y\n" +
	"\x14CleanExpiredSessions\x12\x16.google.protobuf.Empty\x1a'.auth.Session_CleanExpiredSessions_Resp\"\x00\x12<\n" +
	"\x05Count\x12\x17.auth.Session_Count_Req\x1a\x18.auth.Session_Count_Resp\"\x00\x12?\n" +
	"\x06Create\x12\x18.auth.Session_Create_Req\x1a\x19.auth.Session_Create_Resp\"\x00\x12K\n" +
	"\n" +
	"CreateMany\x12\x1c.auth.Session_CreateMany_Req\x1a\x1d.auth.Session_CreateMany_Resp\"\x00\x12K\n" +
	"\n" +
	"DefaultGet\x12\x1c.auth.Session_DefaultGet_Req\x1a\x1d.auth.Session_DefaultGet_Resp\"\x00\x12?\n" +
	"\x06Delete\x12\x18.auth.Session_Delete_Req\x1a\x19.auth.Session_Delete_Resp\"\x00\x12K\n" +
	"\n" +
	"DeleteById\x12\x1c.auth.Session_DeleteById_Req\x1a\x1d.auth.Session_DeleteById_Resp\"\x00\x12u\n" +
	"\x18GetActiveSessionsForUser\x12*.auth.Session_GetActiveSessionsForUser_Req\x1a+.auth.Session_GetActiveSessionsForUser_Resp\"\x00\x12E\n" +
	"\bOnchange\x12\x1a.auth.Session_Onchange_Req\x1a\x1b.auth.Session_Onchange_Resp\"\x00\x12H\n" +
	"\tReadGroup\x12\x1b.auth.Session_ReadGroup_Req\x1a\x1c.auth.Session_ReadGroup_Resp\"\x00\x12W\n" +
	"\x0eReadGroupCount\x12 .auth.Session_ReadGroupCount_Req\x1a!.auth.Session_ReadGroupCount_Resp\"\x00\x12]\n" +
	"\x10RevokeAllForUser\x12\".auth.Session_RevokeAllForUser_Req\x1a#.auth.Session_RevokeAllForUser_Resp\"\x00\x12T\n" +
	"\rRevokeSession\x12\x1f.auth.Session_RevokeSession_Req\x1a .auth.Session_RevokeSession_Resp\"\x00\x12?\n" +
	"\x06Search\x12\x18.auth.Session_Search_Req\x1a\x19.auth.Session_Search_Resp\"\x00\x12?\n" +
	"\x06Update\x12\x18.auth.Session_Update_Req\x1a\x19.auth.Session_Update_Resp\"\x00\x12K\n" +
	"\n" +
	"UpdateById\x12\x1c.auth.Session_UpdateById_Req\x1a\x1d.auth.Session_UpdateById_Resp\"\x00\x12T\n" +
	"\rValidateToken\x12\x1f.auth.Session_ValidateToken_Req\x1a .auth.Session_ValidateToken_Resp\"\x002\xab\f\n" +
	"\x05Token\x12;\n" +
	"\x06Browse\x12\x16.auth.Token_Browse_Req\x1a\x17.auth.Token_Browse_Resp\"\x00\x12G\n" +
	"\n" +
	"BrowseMany\x12\x1a.auth.Token_BrowseMany_Req\x1a\x1b.auth.Token_BrowseMany_Resp\"\x00\x12S\n" +
	"\x12CleanExpiredTokens\x12\x16.google.protobuf.Empty\x1a#.auth.Token_CleanExpiredTokens_Resp\"\x00\x128\n" +
	"\x05Count\x12\x15.auth.Token_Count_Req\x1a\x16.auth.Token_Count_Resp\"\x00\x12;\n" +
	"\x06Create\x12\x16.auth.Token_Create_Req\x1a\x17.auth.Token_Create_Resp\"\x00\x12G\n" +
	"\n" +
	"CreateMany\x12\x1a.auth.Token_CreateMany_Req\x1a\x1b.auth.Token_CreateMany_Resp\"\x00\x12V\n" +
	"\x0fCreateTokenPair\x12\x1f.auth.Token_CreateTokenPair_Req\x1a .auth.Token_CreateTokenPair_Resp\"\x00\x12G\n" +
	"\n" +
	"DefaultGet\x12\x1a.auth.Token_DefaultGet_Req\x1a\x1b.auth.Token_DefaultGet_Resp\"\x00\x12;\n" +
	"\x06Delete\x12\x16.auth.Token_Delete_Req\x1a\x17.auth.Token_Delete_Resp\"\x00\x12G\n" +
	"\n" +
	"DeleteById\x12\x1a.auth.Token_DeleteById_Req\x1a\x1b.auth.Token_DeleteById_Resp\"\x00\x12A\n" +
	"\bOnchange\x12\x18.auth.Token_Onchange_Req\x1a\x19.auth.Token_Onchange_Resp\"\x00\x12D\n" +
	"\tReadGroup\x12\x19.auth.Token_ReadGroup_Req\x1a\x1a.auth.Token_ReadGroup_Resp\"\x00\x12S\n" +
	"\x0eReadGroupCount\x12\x1e.auth.Token_ReadGroupCount_Req\x1a\x1f.auth.Token_ReadGroupCount_Resp\"\x00\x12P\n" +
	"\rRefreshTokens\x12\x1d.auth.Token_RefreshTokens_Req\x1a\x1e.auth.Token_RefreshTokens_Resp\"\x00\x12b\n" +
	"\x13RevokeAllUserTokens\x12#.auth.Token_RevokeAllUserTokens_Req\x1a$.auth.Token_RevokeAllUserTokens_Resp\"\x00\x12J\n" +
	"\vRevokeToken\x12\x1b.auth.Token_RevokeToken_Req\x1a\x1c.auth.Token_RevokeToken_Resp\"\x00\x12k\n" +
	"\x16RevokeUserAccessTokens\x12&.auth.Token_RevokeUserAccessTokens_Req\x1a'.auth.Token_RevokeUserAccessTokens_Resp\"\x00\x12;\n" +
	"\x06Search\x12\x16.auth.Token_Search_Req\x1a\x17.auth.Token_Search_Resp\"\x00\x12;\n" +
	"\x06Update\x12\x16.auth.Token_Update_Req\x1a\x17.auth.Token_Update_Resp\"\x00\x12G\n" +
	"\n" +
	"UpdateById\x12\x1a.auth.Token_UpdateById_Req\x1a\x1b.auth.Token_UpdateById_Resp\"\x00\x12P\n" +
	"\rValidateToken\x12\x1d.auth.Token_ValidateToken_Req\x1a\x1e.auth.Token_ValidateToken_Resp\"\x002\xb0\x0e\n" +
	"\x04User\x12H\n" +
	"\vAssignRoles\x12\x1a.auth.User_AssignRoles_Req\x1a\x1b.auth.User_AssignRoles_Resp\"\x00\x129\n" +
	"\x06Browse\x12\x15.auth.User_Browse_Req\x1a\x16.auth.User_Browse_Resp\"\x00\x12E\n" +
	"\n" +
	"BrowseMany\x12\x19.auth.User_BrowseMany_Req\x1a\x1a.auth.User_BrowseMany_Resp\"\x00\x12Q\n" +
	"\x0eChangePassword\x12\x1d.auth.User_ChangePassword_Req\x1a\x1e.auth.User_ChangePassword_Resp\"\x00\x12Z\n" +
	"\x11CheckMethodAccess\x12 .auth.User_CheckMethodAccess_Req\x1a!.auth.User_CheckMethodAccess_Resp\"\x00\x126\n" +
	"\x05Count\x12\x14.auth.User_Count_Req\x1a\x15.auth.User_Count_Resp\"\x00\x129\n" +
	"\x06Create\x12\x15.auth.User_Create_Req\x1a\x16.auth.User_Create_Resp\"\x00\x12E\n" +
	"\n" +
	"CreateMany\x12\x19.auth.User_CreateMany_Req\x1a\x1a.auth.User_CreateMany_Resp\"\x00\x12E\n" +
	"\n" +
	"DefaultGet\x12\x19.auth.User_DefaultGet_Req\x1a\x1a.auth.User_DefaultGet_Resp\"\x00\x129\n" +
	"\x06Delete\x12\x15.auth.User_Delete_Req\x1a\x16.auth.User_Delete_Resp\"\x00\x12E\n" +
	"\n" +
	"DeleteById\x12\x19.auth.User_DeleteById_Req\x1a\x1a.auth.User_DeleteById_Resp\"\x00\x12i\n" +
	"\x16GetRecordRuleCondition\x12%.auth.User_GetRecordRuleCondition_Req\x1a&.auth.User_GetRecordRuleCondition_Resp\"\x00\x12N\n" +
	"\rHasPermission\x12\x1c.auth.User_HasPermission_Req\x1a\x1d.auth.User_HasPermission_Resp\"\x00\x12<\n" +
	"\aHasRole\x12\x16.auth.User_HasRole_Req\x1a\x17.auth.User_HasRole_Resp\"\x00\x126\n" +
	"\x05Login\x12\x14.auth.User_Login_Req\x1a\x15.auth.User_Login_Resp\"\x00\x129\n" +
	"\x06Logout\x12\x15.auth.User_Logout_Req\x1a\x16.auth.User_Logout_Resp\"\x00\x12?\n" +
	"\bOnchange\x12\x17.auth.User_Onchange_Req\x1a\x18.auth.User_Onchange_Resp\"\x00\x12B\n" +
	"\tReadGroup\x12\x18.auth.User_ReadGroup_Req\x1a\x19.auth.User_ReadGroup_Resp\"\x00\x12Q\n" +
	"\x0eReadGroupCount\x12\x1d.auth.User_ReadGroupCount_Req\x1a\x1e.auth.User_ReadGroupCount_Resp\"\x00\x12N\n" +
	"\rRefreshTokens\x12\x1c.auth.User_RefreshTokens_Req\x1a\x1d.auth.User_RefreshTokens_Resp\"\x00\x12?\n" +
	"\bRegister\x12\x17.auth.User_Register_Req\x1a\x18.auth.User_Register_Resp\"\x00\x12H\n" +
	"\vRemoveRoles\x12\x1a.auth.User_RemoveRoles_Req\x1a\x1b.auth.User_RemoveRoles_Resp\"\x00\x12N\n" +
	"\rResetPassword\x12\x1c.auth.User_ResetPassword_Req\x1a\x1d.auth.User_ResetPassword_Resp\"\x00\x129\n" +
	"\x06Search\x12\x15.auth.User_Search_Req\x1a\x16.auth.User_Search_Resp\"\x00\x129\n" +
	"\x06Update\x12\x15.auth.User_Update_Req\x1a\x16.auth.User_Update_Resp\"\x00\x12E\n" +
	"\n" +
	"UpdateById\x12\x19.auth.User_UpdateById_Req\x1a\x1a.auth.User_UpdateById_Resp\"\x002\x94\b\n" +
	"\bUserRole\x12A\n" +
	"\x06Browse\x12\x19.auth.UserRole_Browse_Req\x1a\x1a.auth.UserRole_Browse_Resp\"\x00\x12M\n" +
	"\n" +
	"BrowseMany\x12\x1d.auth.UserRole_BrowseMany_Req\x1a\x1e.auth.UserRole_BrowseMany_Resp\"\x00\x12>\n" +
	"\x05Count\x12\x18.auth.UserRole_Count_Req\x1a\x19.auth.UserRole_Count_Resp\"\x00\x12A\n" +
	"\x06Create\x12\x19.auth.UserRole_Create_Req\x1a\x1a.auth.UserRole_Create_Resp\"\x00\x12M\n" +
	"\n" +
	"CreateMany\x12\x1d.auth.UserRole_CreateMany_Req\x1a\x1e.auth.UserRole_CreateMany_Resp\"\x00\x12M\n" +
	"\n" +
	"DefaultGet\x12\x1d.auth.UserRole_DefaultGet_Req\x1a\x1e.auth.UserRole_DefaultGet_Resp\"\x00\x12A\n" +
	"\x06Delete\x12\x19.auth.UserRole_Delete_Req\x1a\x1a.auth.UserRole_Delete_Resp\"\x00\x12M\n" +
	"\n" +
	"DeleteById\x12\x1d.auth.UserRole_DeleteById_Req\x1a\x1e.auth.UserRole_DeleteById_Resp\"\x00\x12G\n" +
	"\bOnchange\x12\x1b.auth.UserRole_Onchange_Req\x1a\x1c.auth.UserRole_Onchange_Resp\"\x00\x12J\n" +
	"\tReadGroup\x12\x1c.auth.UserRole_ReadGroup_Req\x1a\x1d.auth.UserRole_ReadGroup_Resp\"\x00\x12Y\n" +
	"\x0eReadGroupCount\x12!.auth.UserRole_ReadGroupCount_Req\x1a\".auth.UserRole_ReadGroupCount_Resp\"\x00\x12A\n" +
	"\x06Search\x12\x19.auth.UserRole_Search_Req\x1a\x1a.auth.UserRole_Search_Resp\"\x00\x12A\n" +
	"\x06Update\x12\x19.auth.UserRole_Update_Req\x1a\x1a.auth.UserRole_Update_Resp\"\x00\x12M\n" +
	"\n" +
	"UpdateById\x12\x1d.auth.UserRole_UpdateById_Req\x1a\x1e.auth.UserRole_UpdateById_Resp\"\x00BBZ@github.com/choysum-dev/choysum/pkg/auth/grpcclient/authpb;authpbb\x06proto3"

var (
	file_auth_proto_rawDescOnce sync.Once
	file_auth_proto_rawDescData []byte
)

func file_auth_proto_rawDescGZIP() []byte {
	file_auth_proto_rawDescOnce.Do(func() {
		file_auth_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_auth_proto_rawDesc), len(file_auth_proto_rawDesc)))
	})
	return file_auth_proto_rawDescData
}

var file_auth_proto_msgTypes = make([]protoimpl.MessageInfo, 414)
var file_auth_proto_goTypes = []any{
	(*Language_Browse_Req)(nil),                   // 0: auth.Language_Browse_Req
	(*Language_Browse_Resp)(nil),                  // 1: auth.Language_Browse_Resp
	(*Language_BrowseMany_Req)(nil),               // 2: auth.Language_BrowseMany_Req
	(*Language_BrowseMany_Resp)(nil),              // 3: auth.Language_BrowseMany_Resp
	(*Language_Count_Req)(nil),                    // 4: auth.Language_Count_Req
	(*Language_Count_Resp)(nil),                   // 5: auth.Language_Count_Resp
	(*Language_Create_Req)(nil),                   // 6: auth.Language_Create_Req
	(*Language_Create_Resp)(nil),                  // 7: auth.Language_Create_Resp
	(*Language_CreateMany_Req)(nil),               // 8: auth.Language_CreateMany_Req
	(*Language_CreateMany_Resp)(nil),              // 9: auth.Language_CreateMany_Resp
	(*Language_DefaultGet_Req)(nil),               // 10: auth.Language_DefaultGet_Req
	(*Language_DefaultGet_Resp)(nil),              // 11: auth.Language_DefaultGet_Resp
	(*Language_Delete_Req)(nil),                   // 12: auth.Language_Delete_Req
	(*Language_Delete_Resp)(nil),                  // 13: auth.Language_Delete_Resp
	(*Language_DeleteById_Req)(nil),               // 14: auth.Language_DeleteById_Req
	(*Language_DeleteById_Resp)(nil),              // 15: auth.Language_DeleteById_Resp
	(*Language_Onchange_Req)(nil),                 // 16: auth.Language_Onchange_Req
	(*Language_Onchange_Resp)(nil),                // 17: auth.Language_Onchange_Resp
	(*Language_ReadGroup_Req)(nil),                // 18: auth.Language_ReadGroup_Req
	(*Language_ReadGroup_Resp)(nil),               // 19: auth.Language_ReadGroup_Resp
	(*Language_ReadGroupCount_Req)(nil),           // 20: auth.Language_ReadGroupCount_Req
	(*Language_ReadGroupCount_Resp)(nil),          // 21: auth.Language_ReadGroupCount_Resp
	(*Language_Search_Req)(nil),                   // 22: auth.Language_Search_Req
	(*Language_Search_Resp)(nil),                  // 23: auth.Language_Search_Resp
	(*Language_Update_Req)(nil),                   // 24: auth.Language_Update_Req
	(*Language_Update_Resp)(nil),                  // 25: auth.Language_Update_Resp
	(*Language_UpdateById_Req)(nil),               // 26: auth.Language_UpdateById_Req
	(*Language_UpdateById_Resp)(nil),              // 27: auth.Language_UpdateById_Resp
	(*Location_Browse_Req)(nil),                   // 28: auth.Location_Browse_Req
	(*Location_Browse_Resp)(nil),                  // 29: auth.Location_Browse_Resp
	(*Location_BrowseMany_Req)(nil),               // 30: auth.Location_BrowseMany_Req
	(*Location_BrowseMany_Resp)(nil),              // 31: auth.Location_BrowseMany_Resp
	(*Location_Count_Req)(nil),                    // 32: auth.Location_Count_Req
	(*Location_Count_Resp)(nil),                   // 33: auth.Location_Count_Resp
	(*Location_Create_Req)(nil),                   // 34: auth.Location_Create_Req
	(*Location_Create_Resp)(nil),                  // 35: auth.Location_Create_Resp
	(*Location_CreateMany_Req)(nil),               // 36: auth.Location_CreateMany_Req
	(*Location_CreateMany_Resp)(nil),              // 37: auth.Location_CreateMany_Resp
	(*Location_DefaultGet_Req)(nil),               // 38: auth.Location_DefaultGet_Req
	(*Location_DefaultGet_Resp)(nil),              // 39: auth.Location_DefaultGet_Resp
	(*Location_Delete_Req)(nil),                   // 40: auth.Location_Delete_Req
	(*Location_Delete_Resp)(nil),                  // 41: auth.Location_Delete_Resp
	(*Location_DeleteById_Req)(nil),               // 42: auth.Location_DeleteById_Req
	(*Location_DeleteById_Resp)(nil),              // 43: auth.Location_DeleteById_Resp
	(*Location_Onchange_Req)(nil),                 // 44: auth.Location_Onchange_Req
	(*Location_Onchange_Resp)(nil),                // 45: auth.Location_Onchange_Resp
	(*Location_ReadGroup_Req)(nil),                // 46: auth.Location_ReadGroup_Req
	(*Location_ReadGroup_Resp)(nil),               // 47: auth.Location_ReadGroup_Resp
	(*Location_ReadGroupCount_Req)(nil),           // 48: auth.Location_ReadGroupCount_Req
	(*Location_ReadGroupCount_Resp)(nil),          // 49: auth.Location_ReadGroupCount_Resp
	(*Location_Register_Req)(nil),                 // 50: auth.Location_Register_Req
	(*Location_Register_Resp)(nil),                // 51: auth.Location_Register_Resp
	(*Location_Search_Req)(nil),                   // 52: auth.Location_Search_Req
	(*Location_Search_Resp)(nil),                  // 53: auth.Location_Search_Resp
	(*Location_Update_Req)(nil),                   // 54: auth.Location_Update_Req
	(*Location_Update_Resp)(nil),                  // 55: auth.Location_Update_Resp
	(*Location_UpdateById_Req)(nil),               // 56: auth.Location_UpdateById_Req
	(*Location_UpdateById_Resp)(nil),              // 57: auth.Location_UpdateById_Resp
	(*Order_Browse_Req)(nil),                      // 58: auth.Order_Browse_Req
	(*Order_Browse_Resp)(nil),                     // 59: auth.Order_Browse_Resp
	(*Order_BrowseMany_Req)(nil),                  // 60: auth.Order_BrowseMany_Req
	(*Order_BrowseMany_Resp)(nil),                 // 61: auth.Order_BrowseMany_Resp
	(*Order_Count_Req)(nil),                       // 62: auth.Order_Count_Req
	(*Order_Count_Resp)(nil),                      // 63: auth.Order_Count_Resp
	(*Order_Create_Req)(nil),                      // 64: auth.Order_Create_Req
	(*Order_Create_Resp)(nil),                     // 65: auth.Order_Create_Resp
	(*Order_CreateMany_Req)(nil),                  // 66: auth.Order_CreateMany_Req
	(*Order_CreateMany_Resp)(nil),                 // 67: auth.Order_CreateMany_Resp
	(*Order_DefaultGet_Req)(nil),                  // 68: auth.Order_DefaultGet_Req
	(*Order_DefaultGet_Resp)(nil),                 // 69: auth.Order_DefaultGet_Resp
	(*Order_Delete_Req)(nil),                      // 70: auth.Order_Delete_Req
	(*Order_Delete_Resp)(nil),                     // 71: auth.Order_Delete_Resp
	(*Order_DeleteById_Req)(nil),                  // 72: auth.Order_DeleteById_Req
	(*Order_DeleteById_Resp)(nil),                 // 73: auth.Order_DeleteById_Resp
	(*Order_Onchange_Req)(nil),                    // 74: auth.Order_Onchange_Req
	(*Order_Onchange_Resp)(nil),                   // 75: auth.Order_Onchange_Resp
	(*Order_ReadGroup_Req)(nil),                   // 76: auth.Order_ReadGroup_Req
	(*Order_ReadGroup_Resp)(nil),                  // 77: auth.Order_ReadGroup_Resp
	(*Order_ReadGroupCount_Req)(nil),              // 78: auth.Order_ReadGroupCount_Req
	(*Order_ReadGroupCount_Resp)(nil),             // 79: auth.Order_ReadGroupCount_Resp
	(*Order_Search_Req)(nil),                      // 80: auth.Order_Search_Req
	(*Order_Search_Resp)(nil),                     // 81: auth.Order_Search_Resp
	(*Order_Update_Req)(nil),                      // 82: auth.Order_Update_Req
	(*Order_Update_Resp)(nil),                     // 83: auth.Order_Update_Resp
	(*Order_UpdateById_Req)(nil),                  // 84: auth.Order_UpdateById_Req
	(*Order_UpdateById_Resp)(nil),                 // 85: auth.Order_UpdateById_Resp
	(*OrderLine_Browse_Req)(nil),                  // 86: auth.OrderLine_Browse_Req
	(*OrderLine_Browse_Resp)(nil),                 // 87: auth.OrderLine_Browse_Resp
	(*OrderLine_BrowseMany_Req)(nil),              // 88: auth.OrderLine_BrowseMany_Req
	(*OrderLine_BrowseMany_Resp)(nil),             // 89: auth.OrderLine_BrowseMany_Resp
	(*OrderLine_Count_Req)(nil),                   // 90: auth.OrderLine_Count_Req
	(*OrderLine_Count_Resp)(nil),                  // 91: auth.OrderLine_Count_Resp
	(*OrderLine_Create_Req)(nil),                  // 92: auth.OrderLine_Create_Req
	(*OrderLine_Create_Resp)(nil),                 // 93: auth.OrderLine_Create_Resp
	(*OrderLine_CreateMany_Req)(nil),              // 94: auth.OrderLine_CreateMany_Req
	(*OrderLine_CreateMany_Resp)(nil),             // 95: auth.OrderLine_CreateMany_Resp
	(*OrderLine_DefaultGet_Req)(nil),              // 96: auth.OrderLine_DefaultGet_Req
	(*OrderLine_DefaultGet_Resp)(nil),             // 97: auth.OrderLine_DefaultGet_Resp
	(*OrderLine_Delete_Req)(nil),                  // 98: auth.OrderLine_Delete_Req
	(*OrderLine_Delete_Resp)(nil),                 // 99: auth.OrderLine_Delete_Resp
	(*OrderLine_DeleteById_Req)(nil),              // 100: auth.OrderLine_DeleteById_Req
	(*OrderLine_DeleteById_Resp)(nil),             // 101: auth.OrderLine_DeleteById_Resp
	(*OrderLine_Onchange_Req)(nil),                // 102: auth.OrderLine_Onchange_Req
	(*OrderLine_Onchange_Resp)(nil),               // 103: auth.OrderLine_Onchange_Resp
	(*OrderLine_ReadGroup_Req)(nil),               // 104: auth.OrderLine_ReadGroup_Req
	(*OrderLine_ReadGroup_Resp)(nil),              // 105: auth.OrderLine_ReadGroup_Resp
	(*OrderLine_ReadGroupCount_Req)(nil),          // 106: auth.OrderLine_ReadGroupCount_Req
	(*OrderLine_ReadGroupCount_Resp)(nil),         // 107: auth.OrderLine_ReadGroupCount_Resp
	(*OrderLine_Search_Req)(nil),                  // 108: auth.OrderLine_Search_Req
	(*OrderLine_Search_Resp)(nil),                 // 109: auth.OrderLine_Search_Resp
	(*OrderLine_Update_Req)(nil),                  // 110: auth.OrderLine_Update_Req
	(*OrderLine_Update_Resp)(nil),                 // 111: auth.OrderLine_Update_Resp
	(*OrderLine_UpdateById_Req)(nil),              // 112: auth.OrderLine_UpdateById_Req
	(*OrderLine_UpdateById_Resp)(nil),             // 113: auth.OrderLine_UpdateById_Resp
	(*Role_Browse_Req)(nil),                       // 114: auth.Role_Browse_Req
	(*Role_Browse_Resp)(nil),                      // 115: auth.Role_Browse_Resp
	(*Role_BrowseMany_Req)(nil),                   // 116: auth.Role_BrowseMany_Req
	(*Role_BrowseMany_Resp)(nil),                  // 117: auth.Role_BrowseMany_Resp
	(*Role_Count_Req)(nil),                        // 118: auth.Role_Count_Req
	(*Role_Count_Resp)(nil),                       // 119: auth.Role_Count_Resp
	(*Role_Create_Req)(nil),                       // 120: auth.Role_Create_Req
	(*Role_Create_Resp)(nil),                      // 121: auth.Role_Create_Resp
	(*Role_CreateIfNotExists_Req)(nil),            // 122: auth.Role_CreateIfNotExists_Req
	(*Role_CreateIfNotExists_Resp)(nil),           // 123: auth.Role_CreateIfNotExists_Resp
	(*Role_CreateMany_Req)(nil),                   // 124: auth.Role_CreateMany_Req
	(*Role_CreateMany_Resp)(nil),                  // 125: auth.Role_CreateMany_Resp
	(*Role_DefaultGet_Req)(nil),                   // 126: auth.Role_DefaultGet_Req
	(*Role_DefaultGet_Resp)(nil),                  // 127: auth.Role_DefaultGet_Resp
	(*Role_Delete_Req)(nil),                       // 128: auth.Role_Delete_Req
	(*Role_Delete_Resp)(nil),                      // 129: auth.Role_Delete_Resp
	(*Role_DeleteById_Req)(nil),                   // 130: auth.Role_DeleteById_Req
	(*Role_DeleteById_Resp)(nil),                  // 131: auth.Role_DeleteById_Resp
	(*Role_Onchange_Req)(nil),                     // 132: auth.Role_Onchange_Req
	(*Role_Onchange_Resp)(nil),                    // 133: auth.Role_Onchange_Resp
	(*Role_ReadGroup_Req)(nil),                    // 134: auth.Role_ReadGroup_Req
	(*Role_ReadGroup_Resp)(nil),                   // 135: auth.Role_ReadGroup_Resp
	(*Role_ReadGroupCount_Req)(nil),               // 136: auth.Role_ReadGroupCount_Req
	(*Role_ReadGroupCount_Resp)(nil),              // 137: auth.Role_ReadGroupCount_Resp
	(*Role_Search_Req)(nil),                       // 138: auth.Role_Search_Req
	(*Role_Search_Resp)(nil),                      // 139: auth.Role_Search_Resp
	(*Role_Update_Req)(nil),                       // 140: auth.Role_Update_Req
	(*Role_Update_Resp)(nil),                      // 141: auth.Role_Update_Resp
	(*Role_UpdateById_Req)(nil),                   // 142: auth.Role_UpdateById_Req
	(*Role_UpdateById_Resp)(nil),                  // 143: auth.Role_UpdateById_Resp
	(*RoleFieldRule_Browse_Req)(nil),              // 144: auth.RoleFieldRule_Browse_Req
	(*RoleFieldRule_Browse_Resp)(nil),             // 145: auth.RoleFieldRule_Browse_Resp
	(*RoleFieldRule_BrowseMany_Req)(nil),          // 146: auth.RoleFieldRule_BrowseMany_Req
	(*RoleFieldRule_BrowseMany_Resp)(nil),         // 147: auth.RoleFieldRule_BrowseMany_Resp
	(*RoleFieldRule_Count_Req)(nil),               // 148: auth.RoleFieldRule_Count_Req
	(*RoleFieldRule_Count_Resp)(nil),              // 149: auth.RoleFieldRule_Count_Resp
	(*RoleFieldRule_Create_Req)(nil),              // 150: auth.RoleFieldRule_Create_Req
	(*RoleFieldRule_Create_Resp)(nil),             // 151: auth.RoleFieldRule_Create_Resp
	(*RoleFieldRule_CreateMany_Req)(nil),          // 152: auth.RoleFieldRule_CreateMany_Req
	(*RoleFieldRule_CreateMany_Resp)(nil),         // 153: auth.RoleFieldRule_CreateMany_Resp
	(*RoleFieldRule_DefaultGet_Req)(nil),          // 154: auth.RoleFieldRule_DefaultGet_Req
	(*RoleFieldRule_DefaultGet_Resp)(nil),         // 155: auth.RoleFieldRule_DefaultGet_Resp
	(*RoleFieldRule_Delete_Req)(nil),              // 156: auth.RoleFieldRule_Delete_Req
	(*RoleFieldRule_Delete_Resp)(nil),             // 157: auth.RoleFieldRule_Delete_Resp
	(*RoleFieldRule_DeleteById_Req)(nil),          // 158: auth.RoleFieldRule_DeleteById_Req
	(*RoleFieldRule_DeleteById_Resp)(nil),         // 159: auth.RoleFieldRule_DeleteById_Resp
	(*RoleFieldRule_Onchange_Req)(nil),            // 160: auth.RoleFieldRule_Onchange_Req
	(*RoleFieldRule_Onchange_Resp)(nil),           // 161: auth.RoleFieldRule_Onchange_Resp
	(*RoleFieldRule_ReadGroup_Req)(nil),           // 162: auth.RoleFieldRule_ReadGroup_Req
	(*RoleFieldRule_ReadGroup_Resp)(nil),          // 163: auth.RoleFieldRule_ReadGroup_Resp
	(*RoleFieldRule_ReadGroupCount_Req)(nil),      // 164: auth.RoleFieldRule_ReadGroupCount_Req
	(*RoleFieldRule_ReadGroupCount_Resp)(nil),     // 165: auth.RoleFieldRule_ReadGroupCount_Resp
	(*RoleFieldRule_Search_Req)(nil),              // 166: auth.RoleFieldRule_Search_Req
	(*RoleFieldRule_Search_Resp)(nil),             // 167: auth.RoleFieldRule_Search_Resp
	(*RoleFieldRule_Update_Req)(nil),              // 168: auth.RoleFieldRule_Update_Req
	(*RoleFieldRule_Update_Resp)(nil),             // 169: auth.RoleFieldRule_Update_Resp
	(*RoleFieldRule_UpdateById_Req)(nil),          // 170: auth.RoleFieldRule_UpdateById_Req
	(*RoleFieldRule_UpdateById_Resp)(nil),         // 171: auth.RoleFieldRule_UpdateById_Resp
	(*RoleInheritance_Browse_Req)(nil),            // 172: auth.RoleInheritance_Browse_Req
	(*RoleInheritance_Browse_Resp)(nil),           // 173: auth.RoleInheritance_Browse_Resp
	(*RoleInheritance_BrowseMany_Req)(nil),        // 174: auth.RoleInheritance_BrowseMany_Req
	(*RoleInheritance_BrowseMany_Resp)(nil),       // 175: auth.RoleInheritance_BrowseMany_Resp
	(*RoleInheritance_Count_Req)(nil),             // 176: auth.RoleInheritance_Count_Req
	(*RoleInheritance_Count_Resp)(nil),            // 177: auth.RoleInheritance_Count_Resp
	(*RoleInheritance_Create_Req)(nil),            // 178: auth.RoleInheritance_Create_Req
	(*RoleInheritance_Create_Resp)(nil),           // 179: auth.RoleInheritance_Create_Resp
	(*RoleInheritance_CreateMany_Req)(nil),        // 180: auth.RoleInheritance_CreateMany_Req
	(*RoleInheritance_CreateMany_Resp)(nil),       // 181: auth.RoleInheritance_CreateMany_Resp
	(*RoleInheritance_DefaultGet_Req)(nil),        // 182: auth.RoleInheritance_DefaultGet_Req
	(*RoleInheritance_DefaultGet_Resp)(nil),       // 183: auth.RoleInheritance_DefaultGet_Resp
	(*RoleInheritance_Delete_Req)(nil),            // 184: auth.RoleInheritance_Delete_Req
	(*RoleInheritance_Delete_Resp)(nil),           // 185: auth.RoleInheritance_Delete_Resp
	(*RoleInheritance_DeleteById_Req)(nil),        // 186: auth.RoleInheritance_DeleteById_Req
	(*RoleInheritance_DeleteById_Resp)(nil),       // 187: auth.RoleInheritance_DeleteById_Resp
	(*RoleInheritance_Onchange_Req)(nil),          // 188: auth.RoleInheritance_Onchange_Req
	(*RoleInheritance_Onchange_Resp)(nil),         // 189: auth.RoleInheritance_Onchange_Resp
	(*RoleInheritance_ReadGroup_Req)(nil),         // 190: auth.RoleInheritance_ReadGroup_Req
	(*RoleInheritance_ReadGroup_Resp)(nil),        // 191: auth.RoleInheritance_ReadGroup_Resp
	(*RoleInheritance_ReadGroupCount_Req)(nil),    // 192: auth.RoleInheritance_ReadGroupCount_Req
	(*RoleInheritance_ReadGroupCount_Resp)(nil),   // 193: auth.RoleInheritance_ReadGroupCount_Resp
	(*RoleInheritance_Search_Req)(nil),            // 194: auth.RoleInheritance_Search_Req
	(*RoleInheritance_Search_Resp)(nil),           // 195: auth.RoleInheritance_Search_Resp
	(*RoleInheritance_Update_Req)(nil),            // 196: auth.RoleInheritance_Update_Req
	(*RoleInheritance_Update_Resp)(nil),           // 197: auth.RoleInheritance_Update_Resp
	(*RoleInheritance_UpdateById_Req)(nil),        // 198: auth.RoleInheritance_UpdateById_Req
	(*RoleInheritance_UpdateById_Resp)(nil),       // 199: auth.RoleInheritance_UpdateById_Resp
	(*RoleMethodAccess_Browse_Req)(nil),           // 200: auth.RoleMethodAccess_Browse_Req
	(*RoleMethodAccess_Browse_Resp)(nil),          // 201: auth.RoleMethodAccess_Browse_Resp
	(*RoleMethodAccess_BrowseMany_Req)(nil),       // 202: auth.RoleMethodAccess_BrowseMany_Req
	(*RoleMethodAccess_BrowseMany_Resp)(nil),      // 203: auth.RoleMethodAccess_BrowseMany_Resp
	(*RoleMethodAccess_Count_Req)(nil),            // 204: auth.RoleMethodAccess_Count_Req
	(*RoleMethodAccess_Count_Resp)(nil),           // 205: auth.RoleMethodAccess_Count_Resp
	(*RoleMethodAccess_Create_Req)(nil),           // 206: auth.RoleMethodAccess_Create_Req
	(*RoleMethodAccess_Create_Resp)(nil),          // 207: auth.RoleMethodAccess_Create_Resp
	(*RoleMethodAccess_CreateMany_Req)(nil),       // 208: auth.RoleMethodAccess_CreateMany_Req
	(*RoleMethodAccess_CreateMany_Resp)(nil),      // 209: auth.RoleMethodAccess_CreateMany_Resp
	(*RoleMethodAccess_DefaultGet_Req)(nil),       // 210: auth.RoleMethodAccess_DefaultGet_Req
	(*RoleMethodAccess_DefaultGet_Resp)(nil),      // 211: auth.RoleMethodAccess_DefaultGet_Resp
	(*RoleMethodAccess_Delete_Req)(nil),           // 212: auth.RoleMethodAccess_Delete_Req
	(*RoleMethodAccess_Delete_Resp)(nil),          // 213: auth.RoleMethodAccess_Delete_Resp
	(*RoleMethodAccess_DeleteById_Req)(nil),       // 214: auth.RoleMethodAccess_DeleteById_Req
	(*RoleMethodAccess_DeleteById_Resp)(nil),      // 215: auth.RoleMethodAccess_DeleteById_Resp
	(*RoleMethodAccess_Onchange_Req)(nil),         // 216: auth.RoleMethodAccess_Onchange_Req
	(*RoleMethodAccess_Onchange_Resp)(nil),        // 217: auth.RoleMethodAccess_Onchange_Resp
	(*RoleMethodAccess_ReadGroup_Req)(nil),        // 218: auth.RoleMethodAccess_ReadGroup_Req
	(*RoleMethodAccess_ReadGroup_Resp)(nil),       // 219: auth.RoleMethodAccess_ReadGroup_Resp
	(*RoleMethodAccess_ReadGroupCount_Req)(nil),   // 220: auth.RoleMethodAccess_ReadGroupCount_Req
	(*RoleMethodAccess_ReadGroupCount_Resp)(nil),  // 221: auth.RoleMethodAccess_ReadGroupCount_Resp
	(*RoleMethodAccess_Search_Req)(nil),           // 222: auth.RoleMethodAccess_Search_Req
	(*RoleMethodAccess_Search_Resp)(nil),          // 223: auth.RoleMethodAccess_Search_Resp
	(*RoleMethodAccess_Update_Req)(nil),           // 224: auth.RoleMethodAccess_Update_Req
	(*RoleMethodAccess_Update_Resp)(nil),          // 225: auth.RoleMethodAccess_Update_Resp
	(*RoleMethodAccess_UpdateById_Req)(nil),       // 226: auth.RoleMethodAccess_UpdateById_Req
	(*RoleMethodAccess_UpdateById_Resp)(nil),      // 227: auth.RoleMethodAccess_UpdateById_Resp
	(*RoleRecordRule_Browse_Req)(nil),             // 228: auth.RoleRecordRule_Browse_Req
	(*RoleRecordRule_Browse_Resp)(nil),            // 229: auth.RoleRecordRule_Browse_Resp
	(*RoleRecordRule_BrowseMany_Req)(nil),         // 230: auth.RoleRecordRule_BrowseMany_Req
	(*RoleRecordRule_BrowseMany_Resp)(nil),        // 231: auth.RoleRecordRule_BrowseMany_Resp
	(*RoleRecordRule_Count_Req)(nil),              // 232: auth.RoleRecordRule_Count_Req
	(*RoleRecordRule_Count_Resp)(nil),             // 233: auth.RoleRecordRule_Count_Resp
	(*RoleRecordRule_Create_Req)(nil),             // 234: auth.RoleRecordRule_Create_Req
	(*RoleRecordRule_Create_Resp)(nil),            // 235: auth.RoleRecordRule_Create_Resp
	(*RoleRecordRule_CreateMany_Req)(nil),         // 236: auth.RoleRecordRule_CreateMany_Req
	(*RoleRecordRule_CreateMany_Resp)(nil),        // 237: auth.RoleRecordRule_CreateMany_Resp
	(*RoleRecordRule_DefaultGet_Req)(nil),         // 238: auth.RoleRecordRule_DefaultGet_Req
	(*RoleRecordRule_DefaultGet_Resp)(nil),        // 239: auth.RoleRecordRule_DefaultGet_Resp
	(*RoleRecordRule_Delete_Req)(nil),             // 240: auth.RoleRecordRule_Delete_Req
	(*RoleRecordRule_Delete_Resp)(nil),            // 241: auth.RoleRecordRule_Delete_Resp
	(*RoleRecordRule_DeleteById_Req)(nil),         // 242: auth.RoleRecordRule_DeleteById_Req
	(*RoleRecordRule_DeleteById_Resp)(nil),        // 243: auth.RoleRecordRule_DeleteById_Resp
	(*RoleRecordRule_Onchange_Req)(nil),           // 244: auth.RoleRecordRule_Onchange_Req
	(*RoleRecordRule_Onchange_Resp)(nil),          // 245: auth.RoleRecordRule_Onchange_Resp
	(*RoleRecordRule_ReadGroup_Req)(nil),          // 246: auth.RoleRecordRule_ReadGroup_Req
	(*RoleRecordRule_ReadGroup_Resp)(nil),         // 247: auth.RoleRecordRule_ReadGroup_Resp
	(*RoleRecordRule_ReadGroupCount_Req)(nil),     // 248: auth.RoleRecordRule_ReadGroupCount_Req
	(*RoleRecordRule_ReadGroupCount_Resp)(nil),    // 249: auth.RoleRecordRule_ReadGroupCount_Resp
	(*RoleRecordRule_Search_Req)(nil),             // 250: auth.RoleRecordRule_Search_Req
	(*RoleRecordRule_Search_Resp)(nil),            // 251: auth.RoleRecordRule_Search_Resp
	(*RoleRecordRule_Update_Req)(nil),             // 252: auth.RoleRecordRule_Update_Req
	(*RoleRecordRule_Update_Resp)(nil),            // 253: auth.RoleRecordRule_Update_Resp
	(*RoleRecordRule_UpdateById_Req)(nil),         // 254: auth.RoleRecordRule_UpdateById_Req
	(*RoleRecordRule_UpdateById_Resp)(nil),        // 255: auth.RoleRecordRule_UpdateById_Resp
	(*Session_Browse_Req)(nil),                    // 256: auth.Session_Browse_Req
	(*Session_Browse_Resp)(nil),                   // 257: auth.Session_Browse_Resp
	(*Session_BrowseMany_Req)(nil),                // 258: auth.Session_BrowseMany_Req
	(*Session_BrowseMany_Resp)(nil),               // 259: auth.Session_BrowseMany_Resp
	(*Session_CleanExpiredSessions_Resp)(nil),     // 260: auth.Session_CleanExpiredSessions_Resp
	(*Session_Count_Req)(nil),                     // 261: auth.Session_Count_Req
	(*Session_Count_Resp)(nil),                    // 262: auth.Session_Count_Resp
	(*Session_Create_Req)(nil),                    // 263: auth.Session_Create_Req
	(*Session_Create_Resp)(nil),                   // 264: auth.Session_Create_Resp
	(*Session_CreateMany_Req)(nil),                // 265: auth.Session_CreateMany_Req
	(*Session_CreateMany_Resp)(nil),               // 266: auth.Session_CreateMany_Resp
	(*Session_DefaultGet_Req)(nil),                // 267: auth.Session_DefaultGet_Req
	(*Session_DefaultGet_Resp)(nil),               // 268: auth.Session_DefaultGet_Resp
	(*Session_Delete_Req)(nil),                    // 269: auth.Session_Delete_Req
	(*Session_Delete_Resp)(nil),                   // 270: auth.Session_Delete_Resp
	(*Session_DeleteById_Req)(nil),                // 271: auth.Session_DeleteById_Req
	(*Session_DeleteById_Resp)(nil),               // 272: auth.Session_DeleteById_Resp
	(*Session_GetActiveSessionsForUser_Req)(nil),  // 273: auth.Session_GetActiveSessionsForUser_Req
	(*Session_GetActiveSessionsForUser_Resp)(nil), // 274: auth.Session_GetActiveSessionsForUser_Resp
	(*Session_Onchange_Req)(nil),                  // 275: auth.Session_Onchange_Req
	(*Session_Onchange_Resp)(nil),                 // 276: auth.Session_Onchange_Resp
	(*Session_ReadGroup_Req)(nil),                 // 277: auth.Session_ReadGroup_Req
	(*Session_ReadGroup_Resp)(nil),                // 278: auth.Session_ReadGroup_Resp
	(*Session_ReadGroupCount_Req)(nil),            // 279: auth.Session_ReadGroupCount_Req
	(*Session_ReadGroupCount_Resp)(nil),           // 280: auth.Session_ReadGroupCount_Resp
	(*Session_RevokeAllForUser_Req)(nil),          // 281: auth.Session_RevokeAllForUser_Req
	(*Session_RevokeAllForUser_Resp)(nil),         // 282: auth.Session_RevokeAllForUser_Resp
	(*Session_RevokeSession_Req)(nil),             // 283: auth.Session_RevokeSession_Req
	(*Session_RevokeSession_Resp)(nil),            // 284: auth.Session_RevokeSession_Resp
	(*Session_Search_Req)(nil),                    // 285: auth.Session_Search_Req
	(*Session_Search_Resp)(nil),                   // 286: auth.Session_Search_Resp
	(*Session_Update_Req)(nil),                    // 287: auth.Session_Update_Req
	(*Session_Update_Resp)(nil),                   // 288: auth.Session_Update_Resp
	(*Session_UpdateById_Req)(nil),                // 289: auth.Session_UpdateById_Req
	(*Session_UpdateById_Resp)(nil),               // 290: auth.Session_UpdateById_Resp
	(*Session_ValidateToken_Req)(nil),             // 291: auth.Session_ValidateToken_Req
	(*Session_ValidateToken_Resp)(nil),            // 292: auth.Session_ValidateToken_Resp
	(*Token_Browse_Req)(nil),                      // 293: auth.Token_Browse_Req
	(*Token_Browse_Resp)(nil),                     // 294: auth.Token_Browse_Resp
	(*Token_BrowseMany_Req)(nil),                  // 295: auth.Token_BrowseMany_Req
	(*Token_BrowseMany_Resp)(nil),                 // 296: auth.Token_BrowseMany_Resp
	(*Token_CleanExpiredTokens_Resp)(nil),         // 297: auth.Token_CleanExpiredTokens_Resp
	(*Token_Count_Req)(nil),                       // 298: auth.Token_Count_Req
	(*Token_Count_Resp)(nil),                      // 299: auth.Token_Count_Resp
	(*Token_Create_Req)(nil),                      // 300: auth.Token_Create_Req
	(*Token_Create_Resp)(nil),                     // 301: auth.Token_Create_Resp
	(*Token_CreateMany_Req)(nil),                  // 302: auth.Token_CreateMany_Req
	(*Token_CreateMany_Resp)(nil),                 // 303: auth.Token_CreateMany_Resp
	(*Token_CreateTokenPair_Req)(nil),             // 304: auth.Token_CreateTokenPair_Req
	(*Token_CreateTokenPair_Resp)(nil),            // 305: auth.Token_CreateTokenPair_Resp
	(*Token_DefaultGet_Req)(nil),                  // 306: auth.Token_DefaultGet_Req
	(*Token_DefaultGet_Resp)(nil),                 // 307: auth.Token_DefaultGet_Resp
	(*Token_Delete_Req)(nil),                      // 308: auth.Token_Delete_Req
	(*Token_Delete_Resp)(nil),                     // 309: auth.Token_Delete_Resp
	(*Token_DeleteById_Req)(nil),                  // 310: auth.Token_DeleteById_Req
	(*Token_DeleteById_Resp)(nil),                 // 311: auth.Token_DeleteById_Resp
	(*Token_Onchange_Req)(nil),                    // 312: auth.Token_Onchange_Req
	(*Token_Onchange_Resp)(nil),                   // 313: auth.Token_Onchange_Resp
	(*Token_ReadGroup_Req)(nil),                   // 314: auth.Token_ReadGroup_Req
	(*Token_ReadGroup_Resp)(nil),                  // 315: auth.Token_ReadGroup_Resp
	(*Token_ReadGroupCount_Req)(nil),              // 316: auth.Token_ReadGroupCount_Req
	(*Token_ReadGroupCount_Resp)(nil),             // 317: auth.Token_ReadGroupCount_Resp
	(*Token_RefreshTokens_Req)(nil),               // 318: auth.Token_RefreshTokens_Req
	(*Token_RefreshTokens_Resp)(nil),              // 319: auth.Token_RefreshTokens_Resp
	(*Token_RevokeAllUserTokens_Req)(nil),         // 320: auth.Token_RevokeAllUserTokens_Req
	(*Token_RevokeAllUserTokens_Resp)(nil),        // 321: auth.Token_RevokeAllUserTokens_Resp
	(*Token_RevokeToken_Req)(nil),                 // 322: auth.Token_RevokeToken_Req
	(*Token_RevokeToken_Resp)(nil),                // 323: auth.Token_RevokeToken_Resp
	(*Token_RevokeUserAccessTokens_Req)(nil),      // 324: auth.Token_RevokeUserAccessTokens_Req
	(*Token_RevokeUserAccessTokens_Resp)(nil),     // 325: auth.Token_RevokeUserAccessTokens_Resp
	(*Token_Search_Req)(nil),                      // 326: auth.Token_Search_Req
	(*Token_Search_Resp)(nil),                     // 327: auth.Token_Search_Resp
	(*Token_Update_Req)(nil),                      // 328: auth.Token_Update_Req
	(*Token_Update_Resp)(nil),                     // 329: auth.Token_Update_Resp
	(*Token_UpdateById_Req)(nil),                  // 330: auth.Token_UpdateById_Req
	(*Token_UpdateById_Resp)(nil),                 // 331: auth.Token_UpdateById_Resp
	(*Token_ValidateToken_Req)(nil),               // 332: auth.Token_ValidateToken_Req
	(*Token_ValidateToken_Resp)(nil),              // 333: auth.Token_ValidateToken_Resp
	(*User_AssignRoles_Req)(nil),                  // 334: auth.User_AssignRoles_Req
	(*User_AssignRoles_Resp)(nil),                 // 335: auth.User_AssignRoles_Resp
	(*User_Browse_Req)(nil),                       // 336: auth.User_Browse_Req
	(*User_Browse_Resp)(nil),                      // 337: auth.User_Browse_Resp
	(*User_BrowseMany_Req)(nil),                   // 338: auth.User_BrowseMany_Req
	(*User_BrowseMany_Resp)(nil),                  // 339: auth.User_BrowseMany_Resp
	(*User_ChangePassword_Req)(nil),               // 340: auth.User_ChangePassword_Req
	(*User_ChangePassword_Resp)(nil),              // 341: auth.User_ChangePassword_Resp
	(*User_CheckMethodAccess_Req)(nil),            // 342: auth.User_CheckMethodAccess_Req
	(*User_CheckMethodAccess_Resp)(nil),           // 343: auth.User_CheckMethodAccess_Resp
	(*User_Count_Req)(nil),                        // 344: auth.User_Count_Req
	(*User_Count_Resp)(nil),                       // 345: auth.User_Count_Resp
	(*User_Create_Req)(nil),                       // 346: auth.User_Create_Req
	(*User_Create_Resp)(nil),                      // 347: auth.User_Create_Resp
	(*User_CreateMany_Req)(nil),                   // 348: auth.User_CreateMany_Req
	(*User_CreateMany_Resp)(nil),                  // 349: auth.User_CreateMany_Resp
	(*User_DefaultGet_Req)(nil),                   // 350: auth.User_DefaultGet_Req
	(*User_DefaultGet_Resp)(nil),                  // 351: auth.User_DefaultGet_Resp
	(*User_Delete_Req)(nil),                       // 352: auth.User_Delete_Req
	(*User_Delete_Resp)(nil),                      // 353: auth.User_Delete_Resp
	(*User_DeleteById_Req)(nil),                   // 354: auth.User_DeleteById_Req
	(*User_DeleteById_Resp)(nil),                  // 355: auth.User_DeleteById_Resp
	(*User_GetRecordRuleCondition_Req)(nil),       // 356: auth.User_GetRecordRuleCondition_Req
	(*User_GetRecordRuleCondition_Resp)(nil),      // 357: auth.User_GetRecordRuleCondition_Resp
	(*User_HasPermission_Req)(nil),                // 358: auth.User_HasPermission_Req
	(*User_HasPermission_Resp)(nil),               // 359: auth.User_HasPermission_Resp
	(*User_HasRole_Req)(nil),                      // 360: auth.User_HasRole_Req
	(*User_HasRole_Resp)(nil),                     // 361: auth.User_HasRole_Resp
	(*User_Login_Req)(nil),                        // 362: auth.User_Login_Req
	(*User_Login_Resp)(nil),                       // 363: auth.User_Login_Resp
	(*User_Logout_Req)(nil),                       // 364: auth.User_Logout_Req
	(*User_Logout_Resp)(nil),                      // 365: auth.User_Logout_Resp
	(*User_Onchange_Req)(nil),                     // 366: auth.User_Onchange_Req
	(*User_Onchange_Resp)(nil),                    // 367: auth.User_Onchange_Resp
	(*User_ReadGroup_Req)(nil),                    // 368: auth.User_ReadGroup_Req
	(*User_ReadGroup_Resp)(nil),                   // 369: auth.User_ReadGroup_Resp
	(*User_ReadGroupCount_Req)(nil),               // 370: auth.User_ReadGroupCount_Req
	(*User_ReadGroupCount_Resp)(nil),              // 371: auth.User_ReadGroupCount_Resp
	(*User_RefreshTokens_Req)(nil),                // 372: auth.User_RefreshTokens_Req
	(*User_RefreshTokens_Resp)(nil),               // 373: auth.User_RefreshTokens_Resp
	(*User_Register_Req)(nil),                     // 374: auth.User_Register_Req
	(*User_Register_Resp)(nil),                    // 375: auth.User_Register_Resp
	(*User_RemoveRoles_Req)(nil),                  // 376: auth.User_RemoveRoles_Req
	(*User_RemoveRoles_Resp)(nil),                 // 377: auth.User_RemoveRoles_Resp
	(*User_ResetPassword_Req)(nil),                // 378: auth.User_ResetPassword_Req
	(*User_ResetPassword_Resp)(nil),               // 379: auth.User_ResetPassword_Resp
	(*User_Search_Req)(nil),                       // 380: auth.User_Search_Req
	(*User_Search_Resp)(nil),                      // 381: auth.User_Search_Resp
	(*User_Update_Req)(nil),                       // 382: auth.User_Update_Req
	(*User_Update_Resp)(nil),                      // 383: auth.User_Update_Resp
	(*User_UpdateById_Req)(nil),                   // 384: auth.User_UpdateById_Req
	(*User_UpdateById_Resp)(nil),                  // 385: auth.User_UpdateById_Resp
	(*UserRole_Browse_Req)(nil),                   // 386: auth.UserRole_Browse_Req
	(*UserRole_Browse_Resp)(nil),                  // 387: auth.UserRole_Browse_Resp
	(*UserRole_BrowseMany_Req)(nil),               // 388: auth.UserRole_BrowseMany_Req
	(*UserRole_BrowseMany_Resp)(nil),              // 389: auth.UserRole_BrowseMany_Resp
	(*UserRole_Count_Req)(nil),                    // 390: auth.UserRole_Count_Req
	(*UserRole_Count_Resp)(nil),                   // 391: auth.UserRole_Count_Resp
	(*UserRole_Create_Req)(nil),                   // 392: auth.UserRole_Create_Req
	(*UserRole_Create_Resp)(nil),                  // 393: auth.UserRole_Create_Resp
	(*UserRole_CreateMany_Req)(nil),               // 394: auth.UserRole_CreateMany_Req
	(*UserRole_CreateMany_Resp)(nil),              // 395: auth.UserRole_CreateMany_Resp
	(*UserRole_DefaultGet_Req)(nil),               // 396: auth.UserRole_DefaultGet_Req
	(*UserRole_DefaultGet_Resp)(nil),              // 397: auth.UserRole_DefaultGet_Resp
	(*UserRole_Delete_Req)(nil),                   // 398: auth.UserRole_Delete_Req
	(*UserRole_Delete_Resp)(nil),                  // 399: auth.UserRole_Delete_Resp
	(*UserRole_DeleteById_Req)(nil),               // 400: auth.UserRole_DeleteById_Req
	(*UserRole_DeleteById_Resp)(nil),              // 401: auth.UserRole_DeleteById_Resp
	(*UserRole_Onchange_Req)(nil),                 // 402: auth.UserRole_Onchange_Req
	(*UserRole_Onchange_Resp)(nil),                // 403: auth.UserRole_Onchange_Resp
	(*UserRole_ReadGroup_Req)(nil),                // 404: auth.UserRole_ReadGroup_Req
	(*UserRole_ReadGroup_Resp)(nil),               // 405: auth.UserRole_ReadGroup_Resp
	(*UserRole_ReadGroupCount_Req)(nil),           // 406: auth.UserRole_ReadGroupCount_Req
	(*UserRole_ReadGroupCount_Resp)(nil),          // 407: auth.UserRole_ReadGroupCount_Resp
	(*UserRole_Search_Req)(nil),                   // 408: auth.UserRole_Search_Req
	(*UserRole_Search_Resp)(nil),                  // 409: auth.UserRole_Search_Resp
	(*UserRole_Update_Req)(nil),                   // 410: auth.UserRole_Update_Req
	(*UserRole_Update_Resp)(nil),                  // 411: auth.UserRole_Update_Resp
	(*UserRole_UpdateById_Req)(nil),               // 412: auth.UserRole_UpdateById_Req
	(*UserRole_UpdateById_Resp)(nil),              // 413: auth.UserRole_UpdateById_Resp
	(*structpb.Value)(nil),                        // 414: google.protobuf.Value
	(*emptypb.Empty)(nil),                         // 415: google.protobuf.Empty
}
var file_auth_proto_depIdxs = []int32{
	414, // 0: auth.Language_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 1: auth.Language_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 2: auth.Language_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 3: auth.Language_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 4: auth.Language_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 5: auth.Language_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 6: auth.Language_Create_Req.value:type_name -> google.protobuf.Value
	414, // 7: auth.Language_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 8: auth.Language_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 9: auth.Language_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 10: auth.Language_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 11: auth.Language_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 12: auth.Language_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 13: auth.Language_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 14: auth.Language_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 15: auth.Language_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 16: auth.Language_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 17: auth.Language_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 18: auth.Language_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 19: auth.Language_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 20: auth.Language_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 21: auth.Language_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 22: auth.Language_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 23: auth.Language_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 24: auth.Language_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 25: auth.Language_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 26: auth.Language_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 27: auth.Language_Search_Req.options:type_name -> google.protobuf.Value
	414, // 28: auth.Language_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 29: auth.Language_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 30: auth.Language_Update_Req.values:type_name -> google.protobuf.Value
	414, // 31: auth.Language_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 32: auth.Language_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 33: auth.Language_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 34: auth.Language_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 35: auth.Language_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 36: auth.Location_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 37: auth.Location_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 38: auth.Location_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 39: auth.Location_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 40: auth.Location_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 41: auth.Location_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 42: auth.Location_Create_Req.value:type_name -> google.protobuf.Value
	414, // 43: auth.Location_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 44: auth.Location_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 45: auth.Location_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 46: auth.Location_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 47: auth.Location_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 48: auth.Location_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 49: auth.Location_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 50: auth.Location_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 51: auth.Location_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 52: auth.Location_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 53: auth.Location_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 54: auth.Location_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 55: auth.Location_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 56: auth.Location_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 57: auth.Location_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 58: auth.Location_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 59: auth.Location_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 60: auth.Location_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 61: auth.Location_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 62: auth.Location_Register_Resp.result:type_name -> google.protobuf.Value
	414, // 63: auth.Location_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 64: auth.Location_Search_Req.options:type_name -> google.protobuf.Value
	414, // 65: auth.Location_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 66: auth.Location_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 67: auth.Location_Update_Req.values:type_name -> google.protobuf.Value
	414, // 68: auth.Location_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 69: auth.Location_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 70: auth.Location_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 71: auth.Location_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 72: auth.Location_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 73: auth.Order_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 74: auth.Order_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 75: auth.Order_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 76: auth.Order_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 77: auth.Order_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 78: auth.Order_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 79: auth.Order_Create_Req.value:type_name -> google.protobuf.Value
	414, // 80: auth.Order_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 81: auth.Order_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 82: auth.Order_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 83: auth.Order_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 84: auth.Order_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 85: auth.Order_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 86: auth.Order_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 87: auth.Order_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 88: auth.Order_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 89: auth.Order_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 90: auth.Order_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 91: auth.Order_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 92: auth.Order_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 93: auth.Order_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 94: auth.Order_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 95: auth.Order_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 96: auth.Order_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 97: auth.Order_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 98: auth.Order_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 99: auth.Order_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 100: auth.Order_Search_Req.options:type_name -> google.protobuf.Value
	414, // 101: auth.Order_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 102: auth.Order_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 103: auth.Order_Update_Req.values:type_name -> google.protobuf.Value
	414, // 104: auth.Order_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 105: auth.Order_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 106: auth.Order_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 107: auth.Order_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 108: auth.Order_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 109: auth.OrderLine_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 110: auth.OrderLine_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 111: auth.OrderLine_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 112: auth.OrderLine_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 113: auth.OrderLine_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 114: auth.OrderLine_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 115: auth.OrderLine_Create_Req.value:type_name -> google.protobuf.Value
	414, // 116: auth.OrderLine_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 117: auth.OrderLine_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 118: auth.OrderLine_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 119: auth.OrderLine_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 120: auth.OrderLine_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 121: auth.OrderLine_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 122: auth.OrderLine_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 123: auth.OrderLine_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 124: auth.OrderLine_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 125: auth.OrderLine_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 126: auth.OrderLine_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 127: auth.OrderLine_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 128: auth.OrderLine_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 129: auth.OrderLine_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 130: auth.OrderLine_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 131: auth.OrderLine_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 132: auth.OrderLine_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 133: auth.OrderLine_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 134: auth.OrderLine_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 135: auth.OrderLine_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 136: auth.OrderLine_Search_Req.options:type_name -> google.protobuf.Value
	414, // 137: auth.OrderLine_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 138: auth.OrderLine_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 139: auth.OrderLine_Update_Req.values:type_name -> google.protobuf.Value
	414, // 140: auth.OrderLine_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 141: auth.OrderLine_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 142: auth.OrderLine_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 143: auth.OrderLine_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 144: auth.OrderLine_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 145: auth.Role_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 146: auth.Role_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 147: auth.Role_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 148: auth.Role_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 149: auth.Role_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 150: auth.Role_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 151: auth.Role_Create_Req.value:type_name -> google.protobuf.Value
	414, // 152: auth.Role_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 153: auth.Role_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 154: auth.Role_CreateIfNotExists_Req.roleData:type_name -> google.protobuf.Value
	414, // 155: auth.Role_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 156: auth.Role_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 157: auth.Role_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 158: auth.Role_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 159: auth.Role_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 160: auth.Role_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 161: auth.Role_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 162: auth.Role_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 163: auth.Role_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 164: auth.Role_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 165: auth.Role_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 166: auth.Role_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 167: auth.Role_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 168: auth.Role_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 169: auth.Role_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 170: auth.Role_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 171: auth.Role_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 172: auth.Role_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 173: auth.Role_Search_Req.options:type_name -> google.protobuf.Value
	414, // 174: auth.Role_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 175: auth.Role_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 176: auth.Role_Update_Req.values:type_name -> google.protobuf.Value
	414, // 177: auth.Role_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 178: auth.Role_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 179: auth.Role_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 180: auth.Role_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 181: auth.Role_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 182: auth.RoleFieldRule_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 183: auth.RoleFieldRule_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 184: auth.RoleFieldRule_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 185: auth.RoleFieldRule_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 186: auth.RoleFieldRule_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 187: auth.RoleFieldRule_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 188: auth.RoleFieldRule_Create_Req.value:type_name -> google.protobuf.Value
	414, // 189: auth.RoleFieldRule_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 190: auth.RoleFieldRule_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 191: auth.RoleFieldRule_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 192: auth.RoleFieldRule_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 193: auth.RoleFieldRule_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 194: auth.RoleFieldRule_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 195: auth.RoleFieldRule_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 196: auth.RoleFieldRule_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 197: auth.RoleFieldRule_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 198: auth.RoleFieldRule_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 199: auth.RoleFieldRule_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 200: auth.RoleFieldRule_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 201: auth.RoleFieldRule_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 202: auth.RoleFieldRule_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 203: auth.RoleFieldRule_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 204: auth.RoleFieldRule_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 205: auth.RoleFieldRule_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 206: auth.RoleFieldRule_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 207: auth.RoleFieldRule_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 208: auth.RoleFieldRule_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 209: auth.RoleFieldRule_Search_Req.options:type_name -> google.protobuf.Value
	414, // 210: auth.RoleFieldRule_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 211: auth.RoleFieldRule_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 212: auth.RoleFieldRule_Update_Req.values:type_name -> google.protobuf.Value
	414, // 213: auth.RoleFieldRule_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 214: auth.RoleFieldRule_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 215: auth.RoleFieldRule_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 216: auth.RoleFieldRule_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 217: auth.RoleFieldRule_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 218: auth.RoleInheritance_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 219: auth.RoleInheritance_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 220: auth.RoleInheritance_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 221: auth.RoleInheritance_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 222: auth.RoleInheritance_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 223: auth.RoleInheritance_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 224: auth.RoleInheritance_Create_Req.value:type_name -> google.protobuf.Value
	414, // 225: auth.RoleInheritance_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 226: auth.RoleInheritance_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 227: auth.RoleInheritance_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 228: auth.RoleInheritance_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 229: auth.RoleInheritance_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 230: auth.RoleInheritance_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 231: auth.RoleInheritance_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 232: auth.RoleInheritance_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 233: auth.RoleInheritance_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 234: auth.RoleInheritance_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 235: auth.RoleInheritance_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 236: auth.RoleInheritance_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 237: auth.RoleInheritance_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 238: auth.RoleInheritance_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 239: auth.RoleInheritance_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 240: auth.RoleInheritance_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 241: auth.RoleInheritance_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 242: auth.RoleInheritance_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 243: auth.RoleInheritance_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 244: auth.RoleInheritance_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 245: auth.RoleInheritance_Search_Req.options:type_name -> google.protobuf.Value
	414, // 246: auth.RoleInheritance_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 247: auth.RoleInheritance_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 248: auth.RoleInheritance_Update_Req.values:type_name -> google.protobuf.Value
	414, // 249: auth.RoleInheritance_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 250: auth.RoleInheritance_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 251: auth.RoleInheritance_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 252: auth.RoleInheritance_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 253: auth.RoleInheritance_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 254: auth.RoleMethodAccess_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 255: auth.RoleMethodAccess_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 256: auth.RoleMethodAccess_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 257: auth.RoleMethodAccess_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 258: auth.RoleMethodAccess_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 259: auth.RoleMethodAccess_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 260: auth.RoleMethodAccess_Create_Req.value:type_name -> google.protobuf.Value
	414, // 261: auth.RoleMethodAccess_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 262: auth.RoleMethodAccess_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 263: auth.RoleMethodAccess_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 264: auth.RoleMethodAccess_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 265: auth.RoleMethodAccess_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 266: auth.RoleMethodAccess_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 267: auth.RoleMethodAccess_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 268: auth.RoleMethodAccess_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 269: auth.RoleMethodAccess_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 270: auth.RoleMethodAccess_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 271: auth.RoleMethodAccess_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 272: auth.RoleMethodAccess_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 273: auth.RoleMethodAccess_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 274: auth.RoleMethodAccess_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 275: auth.RoleMethodAccess_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 276: auth.RoleMethodAccess_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 277: auth.RoleMethodAccess_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 278: auth.RoleMethodAccess_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 279: auth.RoleMethodAccess_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 280: auth.RoleMethodAccess_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 281: auth.RoleMethodAccess_Search_Req.options:type_name -> google.protobuf.Value
	414, // 282: auth.RoleMethodAccess_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 283: auth.RoleMethodAccess_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 284: auth.RoleMethodAccess_Update_Req.values:type_name -> google.protobuf.Value
	414, // 285: auth.RoleMethodAccess_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 286: auth.RoleMethodAccess_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 287: auth.RoleMethodAccess_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 288: auth.RoleMethodAccess_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 289: auth.RoleMethodAccess_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 290: auth.RoleRecordRule_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 291: auth.RoleRecordRule_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 292: auth.RoleRecordRule_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 293: auth.RoleRecordRule_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 294: auth.RoleRecordRule_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 295: auth.RoleRecordRule_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 296: auth.RoleRecordRule_Create_Req.value:type_name -> google.protobuf.Value
	414, // 297: auth.RoleRecordRule_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 298: auth.RoleRecordRule_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 299: auth.RoleRecordRule_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 300: auth.RoleRecordRule_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 301: auth.RoleRecordRule_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 302: auth.RoleRecordRule_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 303: auth.RoleRecordRule_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 304: auth.RoleRecordRule_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 305: auth.RoleRecordRule_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 306: auth.RoleRecordRule_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 307: auth.RoleRecordRule_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 308: auth.RoleRecordRule_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 309: auth.RoleRecordRule_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 310: auth.RoleRecordRule_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 311: auth.RoleRecordRule_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 312: auth.RoleRecordRule_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 313: auth.RoleRecordRule_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 314: auth.RoleRecordRule_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 315: auth.RoleRecordRule_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 316: auth.RoleRecordRule_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 317: auth.RoleRecordRule_Search_Req.options:type_name -> google.protobuf.Value
	414, // 318: auth.RoleRecordRule_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 319: auth.RoleRecordRule_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 320: auth.RoleRecordRule_Update_Req.values:type_name -> google.protobuf.Value
	414, // 321: auth.RoleRecordRule_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 322: auth.RoleRecordRule_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 323: auth.RoleRecordRule_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 324: auth.RoleRecordRule_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 325: auth.RoleRecordRule_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 326: auth.Session_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 327: auth.Session_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 328: auth.Session_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 329: auth.Session_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 330: auth.Session_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 331: auth.Session_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 332: auth.Session_Create_Req.value:type_name -> google.protobuf.Value
	414, // 333: auth.Session_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 334: auth.Session_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 335: auth.Session_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 336: auth.Session_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 337: auth.Session_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 338: auth.Session_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 339: auth.Session_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 340: auth.Session_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 341: auth.Session_GetActiveSessionsForUser_Resp.result:type_name -> google.protobuf.Value
	414, // 342: auth.Session_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 343: auth.Session_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 344: auth.Session_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 345: auth.Session_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 346: auth.Session_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 347: auth.Session_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 348: auth.Session_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 349: auth.Session_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 350: auth.Session_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 351: auth.Session_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 352: auth.Session_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 353: auth.Session_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 354: auth.Session_Search_Req.options:type_name -> google.protobuf.Value
	414, // 355: auth.Session_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 356: auth.Session_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 357: auth.Session_Update_Req.values:type_name -> google.protobuf.Value
	414, // 358: auth.Session_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 359: auth.Session_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 360: auth.Session_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 361: auth.Session_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 362: auth.Session_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 363: auth.Session_ValidateToken_Resp.result:type_name -> google.protobuf.Value
	414, // 364: auth.Token_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 365: auth.Token_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 366: auth.Token_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 367: auth.Token_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 368: auth.Token_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 369: auth.Token_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 370: auth.Token_Create_Req.value:type_name -> google.protobuf.Value
	414, // 371: auth.Token_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 372: auth.Token_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 373: auth.Token_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 374: auth.Token_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 375: auth.Token_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 376: auth.Token_CreateTokenPair_Req.metadata:type_name -> google.protobuf.Value
	414, // 377: auth.Token_CreateTokenPair_Resp.result:type_name -> google.protobuf.Value
	414, // 378: auth.Token_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 379: auth.Token_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 380: auth.Token_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 381: auth.Token_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 382: auth.Token_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 383: auth.Token_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 384: auth.Token_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 385: auth.Token_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 386: auth.Token_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 387: auth.Token_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 388: auth.Token_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 389: auth.Token_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 390: auth.Token_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 391: auth.Token_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 392: auth.Token_RefreshTokens_Req.metadata:type_name -> google.protobuf.Value
	414, // 393: auth.Token_RefreshTokens_Resp.result:type_name -> google.protobuf.Value
	414, // 394: auth.Token_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 395: auth.Token_Search_Req.options:type_name -> google.protobuf.Value
	414, // 396: auth.Token_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 397: auth.Token_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 398: auth.Token_Update_Req.values:type_name -> google.protobuf.Value
	414, // 399: auth.Token_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 400: auth.Token_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 401: auth.Token_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 402: auth.Token_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 403: auth.Token_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 404: auth.Token_ValidateToken_Resp.result:type_name -> google.protobuf.Value
	414, // 405: auth.User_AssignRoles_Req.roleIds:type_name -> google.protobuf.Value
	414, // 406: auth.User_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 407: auth.User_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 408: auth.User_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 409: auth.User_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 410: auth.User_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 411: auth.User_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 412: auth.User_Create_Req.value:type_name -> google.protobuf.Value
	414, // 413: auth.User_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 414: auth.User_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 415: auth.User_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 416: auth.User_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 417: auth.User_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 418: auth.User_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 419: auth.User_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 420: auth.User_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 421: auth.User_GetRecordRuleCondition_Resp.result:type_name -> google.protobuf.Value
	414, // 422: auth.User_Login_Resp.result:type_name -> google.protobuf.Value
	414, // 423: auth.User_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 424: auth.User_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 425: auth.User_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 426: auth.User_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 427: auth.User_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 428: auth.User_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 429: auth.User_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 430: auth.User_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 431: auth.User_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 432: auth.User_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 433: auth.User_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 434: auth.User_RefreshTokens_Resp.result:type_name -> google.protobuf.Value
	414, // 435: auth.User_Register_Req.userData:type_name -> google.protobuf.Value
	414, // 436: auth.User_RemoveRoles_Req.roleIds:type_name -> google.protobuf.Value
	414, // 437: auth.User_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 438: auth.User_Search_Req.options:type_name -> google.protobuf.Value
	414, // 439: auth.User_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 440: auth.User_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 441: auth.User_Update_Req.values:type_name -> google.protobuf.Value
	414, // 442: auth.User_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 443: auth.User_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 444: auth.User_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 445: auth.User_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 446: auth.User_UpdateById_Resp.result:type_name -> google.protobuf.Value
	414, // 447: auth.UserRole_Browse_Req.fields:type_name -> google.protobuf.Value
	414, // 448: auth.UserRole_Browse_Resp.result:type_name -> google.protobuf.Value
	414, // 449: auth.UserRole_BrowseMany_Req.ids:type_name -> google.protobuf.Value
	414, // 450: auth.UserRole_BrowseMany_Req.fields:type_name -> google.protobuf.Value
	414, // 451: auth.UserRole_BrowseMany_Resp.result:type_name -> google.protobuf.Value
	414, // 452: auth.UserRole_Count_Req.condition:type_name -> google.protobuf.Value
	414, // 453: auth.UserRole_Create_Req.value:type_name -> google.protobuf.Value
	414, // 454: auth.UserRole_Create_Req.returnFields:type_name -> google.protobuf.Value
	414, // 455: auth.UserRole_Create_Resp.result:type_name -> google.protobuf.Value
	414, // 456: auth.UserRole_CreateMany_Req.values:type_name -> google.protobuf.Value
	414, // 457: auth.UserRole_CreateMany_Req.returnFields:type_name -> google.protobuf.Value
	414, // 458: auth.UserRole_CreateMany_Resp.result:type_name -> google.protobuf.Value
	414, // 459: auth.UserRole_DefaultGet_Req.value:type_name -> google.protobuf.Value
	414, // 460: auth.UserRole_DefaultGet_Resp.result:type_name -> google.protobuf.Value
	414, // 461: auth.UserRole_Delete_Req.condition:type_name -> google.protobuf.Value
	414, // 462: auth.UserRole_Onchange_Req.draft:type_name -> google.protobuf.Value
	414, // 463: auth.UserRole_Onchange_Req.changed:type_name -> google.protobuf.Value
	414, // 464: auth.UserRole_Onchange_Req.opts:type_name -> google.protobuf.Value
	414, // 465: auth.UserRole_Onchange_Resp.result:type_name -> google.protobuf.Value
	414, // 466: auth.UserRole_ReadGroup_Req.groupby:type_name -> google.protobuf.Value
	414, // 467: auth.UserRole_ReadGroup_Req.condition:type_name -> google.protobuf.Value
	414, // 468: auth.UserRole_ReadGroup_Req.options:type_name -> google.protobuf.Value
	414, // 469: auth.UserRole_ReadGroup_Resp.result:type_name -> google.protobuf.Value
	414, // 470: auth.UserRole_ReadGroupCount_Req.groupby:type_name -> google.protobuf.Value
	414, // 471: auth.UserRole_ReadGroupCount_Req.condition:type_name -> google.protobuf.Value
	414, // 472: auth.UserRole_ReadGroupCount_Req.options:type_name -> google.protobuf.Value
	414, // 473: auth.UserRole_Search_Req.condition:type_name -> google.protobuf.Value
	414, // 474: auth.UserRole_Search_Req.options:type_name -> google.protobuf.Value
	414, // 475: auth.UserRole_Search_Resp.result:type_name -> google.protobuf.Value
	414, // 476: auth.UserRole_Update_Req.condition:type_name -> google.protobuf.Value
	414, // 477: auth.UserRole_Update_Req.values:type_name -> google.protobuf.Value
	414, // 478: auth.UserRole_Update_Req.returnFields:type_name -> google.protobuf.Value
	414, // 479: auth.UserRole_Update_Resp.result:type_name -> google.protobuf.Value
	414, // 480: auth.UserRole_UpdateById_Req.values:type_name -> google.protobuf.Value
	414, // 481: auth.UserRole_UpdateById_Req.returnFields:type_name -> google.protobuf.Value
	414, // 482: auth.UserRole_UpdateById_Resp.result:type_name -> google.protobuf.Value
	0,   // 483: auth.Language.Browse:input_type -> auth.Language_Browse_Req
	2,   // 484: auth.Language.BrowseMany:input_type -> auth.Language_BrowseMany_Req
	4,   // 485: auth.Language.Count:input_type -> auth.Language_Count_Req
	6,   // 486: auth.Language.Create:input_type -> auth.Language_Create_Req
	8,   // 487: auth.Language.CreateMany:input_type -> auth.Language_CreateMany_Req
	10,  // 488: auth.Language.DefaultGet:input_type -> auth.Language_DefaultGet_Req
	12,  // 489: auth.Language.Delete:input_type -> auth.Language_Delete_Req
	14,  // 490: auth.Language.DeleteById:input_type -> auth.Language_DeleteById_Req
	16,  // 491: auth.Language.Onchange:input_type -> auth.Language_Onchange_Req
	18,  // 492: auth.Language.ReadGroup:input_type -> auth.Language_ReadGroup_Req
	20,  // 493: auth.Language.ReadGroupCount:input_type -> auth.Language_ReadGroupCount_Req
	22,  // 494: auth.Language.Search:input_type -> auth.Language_Search_Req
	24,  // 495: auth.Language.Update:input_type -> auth.Language_Update_Req
	26,  // 496: auth.Language.UpdateById:input_type -> auth.Language_UpdateById_Req
	28,  // 497: auth.Location.Browse:input_type -> auth.Location_Browse_Req
	30,  // 498: auth.Location.BrowseMany:input_type -> auth.Location_BrowseMany_Req
	32,  // 499: auth.Location.Count:input_type -> auth.Location_Count_Req
	34,  // 500: auth.Location.Create:input_type -> auth.Location_Create_Req
	36,  // 501: auth.Location.CreateMany:input_type -> auth.Location_CreateMany_Req
	38,  // 502: auth.Location.DefaultGet:input_type -> auth.Location_DefaultGet_Req
	40,  // 503: auth.Location.Delete:input_type -> auth.Location_Delete_Req
	42,  // 504: auth.Location.DeleteById:input_type -> auth.Location_DeleteById_Req
	44,  // 505: auth.Location.Onchange:input_type -> auth.Location_Onchange_Req
	46,  // 506: auth.Location.ReadGroup:input_type -> auth.Location_ReadGroup_Req
	48,  // 507: auth.Location.ReadGroupCount:input_type -> auth.Location_ReadGroupCount_Req
	50,  // 508: auth.Location.Register:input_type -> auth.Location_Register_Req
	52,  // 509: auth.Location.Search:input_type -> auth.Location_Search_Req
	54,  // 510: auth.Location.Update:input_type -> auth.Location_Update_Req
	56,  // 511: auth.Location.UpdateById:input_type -> auth.Location_UpdateById_Req
	58,  // 512: auth.Order.Browse:input_type -> auth.Order_Browse_Req
	60,  // 513: auth.Order.BrowseMany:input_type -> auth.Order_BrowseMany_Req
	62,  // 514: auth.Order.Count:input_type -> auth.Order_Count_Req
	64,  // 515: auth.Order.Create:input_type -> auth.Order_Create_Req
	66,  // 516: auth.Order.CreateMany:input_type -> auth.Order_CreateMany_Req
	68,  // 517: auth.Order.DefaultGet:input_type -> auth.Order_DefaultGet_Req
	70,  // 518: auth.Order.Delete:input_type -> auth.Order_Delete_Req
	72,  // 519: auth.Order.DeleteById:input_type -> auth.Order_DeleteById_Req
	74,  // 520: auth.Order.Onchange:input_type -> auth.Order_Onchange_Req
	76,  // 521: auth.Order.ReadGroup:input_type -> auth.Order_ReadGroup_Req
	78,  // 522: auth.Order.ReadGroupCount:input_type -> auth.Order_ReadGroupCount_Req
	80,  // 523: auth.Order.Search:input_type -> auth.Order_Search_Req
	82,  // 524: auth.Order.Update:input_type -> auth.Order_Update_Req
	84,  // 525: auth.Order.UpdateById:input_type -> auth.Order_UpdateById_Req
	86,  // 526: auth.OrderLine.Browse:input_type -> auth.OrderLine_Browse_Req
	88,  // 527: auth.OrderLine.BrowseMany:input_type -> auth.OrderLine_BrowseMany_Req
	90,  // 528: auth.OrderLine.Count:input_type -> auth.OrderLine_Count_Req
	92,  // 529: auth.OrderLine.Create:input_type -> auth.OrderLine_Create_Req
	94,  // 530: auth.OrderLine.CreateMany:input_type -> auth.OrderLine_CreateMany_Req
	96,  // 531: auth.OrderLine.DefaultGet:input_type -> auth.OrderLine_DefaultGet_Req
	98,  // 532: auth.OrderLine.Delete:input_type -> auth.OrderLine_Delete_Req
	100, // 533: auth.OrderLine.DeleteById:input_type -> auth.OrderLine_DeleteById_Req
	102, // 534: auth.OrderLine.Onchange:input_type -> auth.OrderLine_Onchange_Req
	104, // 535: auth.OrderLine.ReadGroup:input_type -> auth.OrderLine_ReadGroup_Req
	106, // 536: auth.OrderLine.ReadGroupCount:input_type -> auth.OrderLine_ReadGroupCount_Req
	108, // 537: auth.OrderLine.Search:input_type -> auth.OrderLine_Search_Req
	110, // 538: auth.OrderLine.Update:input_type -> auth.OrderLine_Update_Req
	112, // 539: auth.OrderLine.UpdateById:input_type -> auth.OrderLine_UpdateById_Req
	114, // 540: auth.Role.Browse:input_type -> auth.Role_Browse_Req
	116, // 541: auth.Role.BrowseMany:input_type -> auth.Role_BrowseMany_Req
	118, // 542: auth.Role.Count:input_type -> auth.Role_Count_Req
	120, // 543: auth.Role.Create:input_type -> auth.Role_Create_Req
	122, // 544: auth.Role.CreateIfNotExists:input_type -> auth.Role_CreateIfNotExists_Req
	124, // 545: auth.Role.CreateMany:input_type -> auth.Role_CreateMany_Req
	126, // 546: auth.Role.DefaultGet:input_type -> auth.Role_DefaultGet_Req
	128, // 547: auth.Role.Delete:input_type -> auth.Role_Delete_Req
	130, // 548: auth.Role.DeleteById:input_type -> auth.Role_DeleteById_Req
	132, // 549: auth.Role.Onchange:input_type -> auth.Role_Onchange_Req
	134, // 550: auth.Role.ReadGroup:input_type -> auth.Role_ReadGroup_Req
	136, // 551: auth.Role.ReadGroupCount:input_type -> auth.Role_ReadGroupCount_Req
	138, // 552: auth.Role.Search:input_type -> auth.Role_Search_Req
	140, // 553: auth.Role.Update:input_type -> auth.Role_Update_Req
	142, // 554: auth.Role.UpdateById:input_type -> auth.Role_UpdateById_Req
	144, // 555: auth.RoleFieldRule.Browse:input_type -> auth.RoleFieldRule_Browse_Req
	146, // 556: auth.RoleFieldRule.BrowseMany:input_type -> auth.RoleFieldRule_BrowseMany_Req
	148, // 557: auth.RoleFieldRule.Count:input_type -> auth.RoleFieldRule_Count_Req
	150, // 558: auth.RoleFieldRule.Create:input_type -> auth.RoleFieldRule_Create_Req
	152, // 559: auth.RoleFieldRule.CreateMany:input_type -> auth.RoleFieldRule_CreateMany_Req
	154, // 560: auth.RoleFieldRule.DefaultGet:input_type -> auth.RoleFieldRule_DefaultGet_Req
	156, // 561: auth.RoleFieldRule.Delete:input_type -> auth.RoleFieldRule_Delete_Req
	158, // 562: auth.RoleFieldRule.DeleteById:input_type -> auth.RoleFieldRule_DeleteById_Req
	160, // 563: auth.RoleFieldRule.Onchange:input_type -> auth.RoleFieldRule_Onchange_Req
	162, // 564: auth.RoleFieldRule.ReadGroup:input_type -> auth.RoleFieldRule_ReadGroup_Req
	164, // 565: auth.RoleFieldRule.ReadGroupCount:input_type -> auth.RoleFieldRule_ReadGroupCount_Req
	166, // 566: auth.RoleFieldRule.Search:input_type -> auth.RoleFieldRule_Search_Req
	168, // 567: auth.RoleFieldRule.Update:input_type -> auth.RoleFieldRule_Update_Req
	170, // 568: auth.RoleFieldRule.UpdateById:input_type -> auth.RoleFieldRule_UpdateById_Req
	172, // 569: auth.RoleInheritance.Browse:input_type -> auth.RoleInheritance_Browse_Req
	174, // 570: auth.RoleInheritance.BrowseMany:input_type -> auth.RoleInheritance_BrowseMany_Req
	176, // 571: auth.RoleInheritance.Count:input_type -> auth.RoleInheritance_Count_Req
	178, // 572: auth.RoleInheritance.Create:input_type -> auth.RoleInheritance_Create_Req
	180, // 573: auth.RoleInheritance.CreateMany:input_type -> auth.RoleInheritance_CreateMany_Req
	182, // 574: auth.RoleInheritance.DefaultGet:input_type -> auth.RoleInheritance_DefaultGet_Req
	184, // 575: auth.RoleInheritance.Delete:input_type -> auth.RoleInheritance_Delete_Req
	186, // 576: auth.RoleInheritance.DeleteById:input_type -> auth.RoleInheritance_DeleteById_Req
	188, // 577: auth.RoleInheritance.Onchange:input_type -> auth.RoleInheritance_Onchange_Req
	190, // 578: auth.RoleInheritance.ReadGroup:input_type -> auth.RoleInheritance_ReadGroup_Req
	192, // 579: auth.RoleInheritance.ReadGroupCount:input_type -> auth.RoleInheritance_ReadGroupCount_Req
	194, // 580: auth.RoleInheritance.Search:input_type -> auth.RoleInheritance_Search_Req
	196, // 581: auth.RoleInheritance.Update:input_type -> auth.RoleInheritance_Update_Req
	198, // 582: auth.RoleInheritance.UpdateById:input_type -> auth.RoleInheritance_UpdateById_Req
	200, // 583: auth.RoleMethodAccess.Browse:input_type -> auth.RoleMethodAccess_Browse_Req
	202, // 584: auth.RoleMethodAccess.BrowseMany:input_type -> auth.RoleMethodAccess_BrowseMany_Req
	204, // 585: auth.RoleMethodAccess.Count:input_type -> auth.RoleMethodAccess_Count_Req
	206, // 586: auth.RoleMethodAccess.Create:input_type -> auth.RoleMethodAccess_Create_Req
	208, // 587: auth.RoleMethodAccess.CreateMany:input_type -> auth.RoleMethodAccess_CreateMany_Req
	210, // 588: auth.RoleMethodAccess.DefaultGet:input_type -> auth.RoleMethodAccess_DefaultGet_Req
	212, // 589: auth.RoleMethodAccess.Delete:input_type -> auth.RoleMethodAccess_Delete_Req
	214, // 590: auth.RoleMethodAccess.DeleteById:input_type -> auth.RoleMethodAccess_DeleteById_Req
	216, // 591: auth.RoleMethodAccess.Onchange:input_type -> auth.RoleMethodAccess_Onchange_Req
	218, // 592: auth.RoleMethodAccess.ReadGroup:input_type -> auth.RoleMethodAccess_ReadGroup_Req
	220, // 593: auth.RoleMethodAccess.ReadGroupCount:input_type -> auth.RoleMethodAccess_ReadGroupCount_Req
	222, // 594: auth.RoleMethodAccess.Search:input_type -> auth.RoleMethodAccess_Search_Req
	224, // 595: auth.RoleMethodAccess.Update:input_type -> auth.RoleMethodAccess_Update_Req
	226, // 596: auth.RoleMethodAccess.UpdateById:input_type -> auth.RoleMethodAccess_UpdateById_Req
	228, // 597: auth.RoleRecordRule.Browse:input_type -> auth.RoleRecordRule_Browse_Req
	230, // 598: auth.RoleRecordRule.BrowseMany:input_type -> auth.RoleRecordRule_BrowseMany_Req
	232, // 599: auth.RoleRecordRule.Count:input_type -> auth.RoleRecordRule_Count_Req
	234, // 600: auth.RoleRecordRule.Create:input_type -> auth.RoleRecordRule_Create_Req
	236, // 601: auth.RoleRecordRule.CreateMany:input_type -> auth.RoleRecordRule_CreateMany_Req
	238, // 602: auth.RoleRecordRule.DefaultGet:input_type -> auth.RoleRecordRule_DefaultGet_Req
	240, // 603: auth.RoleRecordRule.Delete:input_type -> auth.RoleRecordRule_Delete_Req
	242, // 604: auth.RoleRecordRule.DeleteById:input_type -> auth.RoleRecordRule_DeleteById_Req
	244, // 605: auth.RoleRecordRule.Onchange:input_type -> auth.RoleRecordRule_Onchange_Req
	246, // 606: auth.RoleRecordRule.ReadGroup:input_type -> auth.RoleRecordRule_ReadGroup_Req
	248, // 607: auth.RoleRecordRule.ReadGroupCount:input_type -> auth.RoleRecordRule_ReadGroupCount_Req
	250, // 608: auth.RoleRecordRule.Search:input_type -> auth.RoleRecordRule_Search_Req
	252, // 609: auth.RoleRecordRule.Update:input_type -> auth.RoleRecordRule_Update_Req
	254, // 610: auth.RoleRecordRule.UpdateById:input_type -> auth.RoleRecordRule_UpdateById_Req
	256, // 611: auth.Session.Browse:input_type -> auth.Session_Browse_Req
	258, // 612: auth.Session.BrowseMany:input_type -> auth.Session_BrowseMany_Req
	415, // 613: auth.Session.CleanExpiredSessions:input_type -> google.protobuf.Empty
	261, // 614: auth.Session.Count:input_type -> auth.Session_Count_Req
	263, // 615: auth.Session.Create:input_type -> auth.Session_Create_Req
	265, // 616: auth.Session.CreateMany:input_type -> auth.Session_CreateMany_Req
	267, // 617: auth.Session.DefaultGet:input_type -> auth.Session_DefaultGet_Req
	269, // 618: auth.Session.Delete:input_type -> auth.Session_Delete_Req
	271, // 619: auth.Session.DeleteById:input_type -> auth.Session_DeleteById_Req
	273, // 620: auth.Session.GetActiveSessionsForUser:input_type -> auth.Session_GetActiveSessionsForUser_Req
	275, // 621: auth.Session.Onchange:input_type -> auth.Session_Onchange_Req
	277, // 622: auth.Session.ReadGroup:input_type -> auth.Session_ReadGroup_Req
	279, // 623: auth.Session.ReadGroupCount:input_type -> auth.Session_ReadGroupCount_Req
	281, // 624: auth.Session.RevokeAllForUser:input_type -> auth.Session_RevokeAllForUser_Req
	283, // 625: auth.Session.RevokeSession:input_type -> auth.Session_RevokeSession_Req
	285, // 626: auth.Session.Search:input_type -> auth.Session_Search_Req
	287, // 627: auth.Session.Update:input_type -> auth.Session_Update_Req
	289, // 628: auth.Session.UpdateById:input_type -> auth.Session_UpdateById_Req
	291, // 629: auth.Session.ValidateToken:input_type -> auth.Session_ValidateToken_Req
	293, // 630: auth.Token.Browse:input_type -> auth.Token_Browse_Req
	295, // 631: auth.Token.BrowseMany:input_type -> auth.Token_BrowseMany_Req
	415, // 632: auth.Token.CleanExpiredTokens:input_type -> google.protobuf.Empty
	298, // 633: auth.Token.Count:input_type -> auth.Token_Count_Req
	300, // 634: auth.Token.Create:input_type -> auth.Token_Create_Req
	302, // 635: auth.Token.CreateMany:input_type -> auth.Token_CreateMany_Req
	304, // 636: auth.Token.CreateTokenPair:input_type -> auth.Token_CreateTokenPair_Req
	306, // 637: auth.Token.DefaultGet:input_type -> auth.Token_DefaultGet_Req
	308, // 638: auth.Token.Delete:input_type -> auth.Token_Delete_Req
	310, // 639: auth.Token.DeleteById:input_type -> auth.Token_DeleteById_Req
	312, // 640: auth.Token.Onchange:input_type -> auth.Token_Onchange_Req
	314, // 641: auth.Token.ReadGroup:input_type -> auth.Token_ReadGroup_Req
	316, // 642: auth.Token.ReadGroupCount:input_type -> auth.Token_ReadGroupCount_Req
	318, // 643: auth.Token.RefreshTokens:input_type -> auth.Token_RefreshTokens_Req
	320, // 644: auth.Token.RevokeAllUserTokens:input_type -> auth.Token_RevokeAllUserTokens_Req
	322, // 645: auth.Token.RevokeToken:input_type -> auth.Token_RevokeToken_Req
	324, // 646: auth.Token.RevokeUserAccessTokens:input_type -> auth.Token_RevokeUserAccessTokens_Req
	326, // 647: auth.Token.Search:input_type -> auth.Token_Search_Req
	328, // 648: auth.Token.Update:input_type -> auth.Token_Update_Req
	330, // 649: auth.Token.UpdateById:input_type -> auth.Token_UpdateById_Req
	332, // 650: auth.Token.ValidateToken:input_type -> auth.Token_ValidateToken_Req
	334, // 651: auth.User.AssignRoles:input_type -> auth.User_AssignRoles_Req
	336, // 652: auth.User.Browse:input_type -> auth.User_Browse_Req
	338, // 653: auth.User.BrowseMany:input_type -> auth.User_BrowseMany_Req
	340, // 654: auth.User.ChangePassword:input_type -> auth.User_ChangePassword_Req
	342, // 655: auth.User.CheckMethodAccess:input_type -> auth.User_CheckMethodAccess_Req
	344, // 656: auth.User.Count:input_type -> auth.User_Count_Req
	346, // 657: auth.User.Create:input_type -> auth.User_Create_Req
	348, // 658: auth.User.CreateMany:input_type -> auth.User_CreateMany_Req
	350, // 659: auth.User.DefaultGet:input_type -> auth.User_DefaultGet_Req
	352, // 660: auth.User.Delete:input_type -> auth.User_Delete_Req
	354, // 661: auth.User.DeleteById:input_type -> auth.User_DeleteById_Req
	356, // 662: auth.User.GetRecordRuleCondition:input_type -> auth.User_GetRecordRuleCondition_Req
	358, // 663: auth.User.HasPermission:input_type -> auth.User_HasPermission_Req
	360, // 664: auth.User.HasRole:input_type -> auth.User_HasRole_Req
	362, // 665: auth.User.Login:input_type -> auth.User_Login_Req
	364, // 666: auth.User.Logout:input_type -> auth.User_Logout_Req
	366, // 667: auth.User.Onchange:input_type -> auth.User_Onchange_Req
	368, // 668: auth.User.ReadGroup:input_type -> auth.User_ReadGroup_Req
	370, // 669: auth.User.ReadGroupCount:input_type -> auth.User_ReadGroupCount_Req
	372, // 670: auth.User.RefreshTokens:input_type -> auth.User_RefreshTokens_Req
	374, // 671: auth.User.Register:input_type -> auth.User_Register_Req
	376, // 672: auth.User.RemoveRoles:input_type -> auth.User_RemoveRoles_Req
	378, // 673: auth.User.ResetPassword:input_type -> auth.User_ResetPassword_Req
	380, // 674: auth.User.Search:input_type -> auth.User_Search_Req
	382, // 675: auth.User.Update:input_type -> auth.User_Update_Req
	384, // 676: auth.User.UpdateById:input_type -> auth.User_UpdateById_Req
	386, // 677: auth.UserRole.Browse:input_type -> auth.UserRole_Browse_Req
	388, // 678: auth.UserRole.BrowseMany:input_type -> auth.UserRole_BrowseMany_Req
	390, // 679: auth.UserRole.Count:input_type -> auth.UserRole_Count_Req
	392, // 680: auth.UserRole.Create:input_type -> auth.UserRole_Create_Req
	394, // 681: auth.UserRole.CreateMany:input_type -> auth.UserRole_CreateMany_Req
	396, // 682: auth.UserRole.DefaultGet:input_type -> auth.UserRole_DefaultGet_Req
	398, // 683: auth.UserRole.Delete:input_type -> auth.UserRole_Delete_Req
	400, // 684: auth.UserRole.DeleteById:input_type -> auth.UserRole_DeleteById_Req
	402, // 685: auth.UserRole.Onchange:input_type -> auth.UserRole_Onchange_Req
	404, // 686: auth.UserRole.ReadGroup:input_type -> auth.UserRole_ReadGroup_Req
	406, // 687: auth.UserRole.ReadGroupCount:input_type -> auth.UserRole_ReadGroupCount_Req
	408, // 688: auth.UserRole.Search:input_type -> auth.UserRole_Search_Req
	410, // 689: auth.UserRole.Update:input_type -> auth.UserRole_Update_Req
	412, // 690: auth.UserRole.UpdateById:input_type -> auth.UserRole_UpdateById_Req
	1,   // 691: auth.Language.Browse:output_type -> auth.Language_Browse_Resp
	3,   // 692: auth.Language.BrowseMany:output_type -> auth.Language_BrowseMany_Resp
	5,   // 693: auth.Language.Count:output_type -> auth.Language_Count_Resp
	7,   // 694: auth.Language.Create:output_type -> auth.Language_Create_Resp
	9,   // 695: auth.Language.CreateMany:output_type -> auth.Language_CreateMany_Resp
	11,  // 696: auth.Language.DefaultGet:output_type -> auth.Language_DefaultGet_Resp
	13,  // 697: auth.Language.Delete:output_type -> auth.Language_Delete_Resp
	15,  // 698: auth.Language.DeleteById:output_type -> auth.Language_DeleteById_Resp
	17,  // 699: auth.Language.Onchange:output_type -> auth.Language_Onchange_Resp
	19,  // 700: auth.Language.ReadGroup:output_type -> auth.Language_ReadGroup_Resp
	21,  // 701: auth.Language.ReadGroupCount:output_type -> auth.Language_ReadGroupCount_Resp
	23,  // 702: auth.Language.Search:output_type -> auth.Language_Search_Resp
	25,  // 703: auth.Language.Update:output_type -> auth.Language_Update_Resp
	27,  // 704: auth.Language.UpdateById:output_type -> auth.Language_UpdateById_Resp
	29,  // 705: auth.Location.Browse:output_type -> auth.Location_Browse_Resp
	31,  // 706: auth.Location.BrowseMany:output_type -> auth.Location_BrowseMany_Resp
	33,  // 707: auth.Location.Count:output_type -> auth.Location_Count_Resp
	35,  // 708: auth.Location.Create:output_type -> auth.Location_Create_Resp
	37,  // 709: auth.Location.CreateMany:output_type -> auth.Location_CreateMany_Resp
	39,  // 710: auth.Location.DefaultGet:output_type -> auth.Location_DefaultGet_Resp
	41,  // 711: auth.Location.Delete:output_type -> auth.Location_Delete_Resp
	43,  // 712: auth.Location.DeleteById:output_type -> auth.Location_DeleteById_Resp
	45,  // 713: auth.Location.Onchange:output_type -> auth.Location_Onchange_Resp
	47,  // 714: auth.Location.ReadGroup:output_type -> auth.Location_ReadGroup_Resp
	49,  // 715: auth.Location.ReadGroupCount:output_type -> auth.Location_ReadGroupCount_Resp
	51,  // 716: auth.Location.Register:output_type -> auth.Location_Register_Resp
	53,  // 717: auth.Location.Search:output_type -> auth.Location_Search_Resp
	55,  // 718: auth.Location.Update:output_type -> auth.Location_Update_Resp
	57,  // 719: auth.Location.UpdateById:output_type -> auth.Location_UpdateById_Resp
	59,  // 720: auth.Order.Browse:output_type -> auth.Order_Browse_Resp
	61,  // 721: auth.Order.BrowseMany:output_type -> auth.Order_BrowseMany_Resp
	63,  // 722: auth.Order.Count:output_type -> auth.Order_Count_Resp
	65,  // 723: auth.Order.Create:output_type -> auth.Order_Create_Resp
	67,  // 724: auth.Order.CreateMany:output_type -> auth.Order_CreateMany_Resp
	69,  // 725: auth.Order.DefaultGet:output_type -> auth.Order_DefaultGet_Resp
	71,  // 726: auth.Order.Delete:output_type -> auth.Order_Delete_Resp
	73,  // 727: auth.Order.DeleteById:output_type -> auth.Order_DeleteById_Resp
	75,  // 728: auth.Order.Onchange:output_type -> auth.Order_Onchange_Resp
	77,  // 729: auth.Order.ReadGroup:output_type -> auth.Order_ReadGroup_Resp
	79,  // 730: auth.Order.ReadGroupCount:output_type -> auth.Order_ReadGroupCount_Resp
	81,  // 731: auth.Order.Search:output_type -> auth.Order_Search_Resp
	83,  // 732: auth.Order.Update:output_type -> auth.Order_Update_Resp
	85,  // 733: auth.Order.UpdateById:output_type -> auth.Order_UpdateById_Resp
	87,  // 734: auth.OrderLine.Browse:output_type -> auth.OrderLine_Browse_Resp
	89,  // 735: auth.OrderLine.BrowseMany:output_type -> auth.OrderLine_BrowseMany_Resp
	91,  // 736: auth.OrderLine.Count:output_type -> auth.OrderLine_Count_Resp
	93,  // 737: auth.OrderLine.Create:output_type -> auth.OrderLine_Create_Resp
	95,  // 738: auth.OrderLine.CreateMany:output_type -> auth.OrderLine_CreateMany_Resp
	97,  // 739: auth.OrderLine.DefaultGet:output_type -> auth.OrderLine_DefaultGet_Resp
	99,  // 740: auth.OrderLine.Delete:output_type -> auth.OrderLine_Delete_Resp
	101, // 741: auth.OrderLine.DeleteById:output_type -> auth.OrderLine_DeleteById_Resp
	103, // 742: auth.OrderLine.Onchange:output_type -> auth.OrderLine_Onchange_Resp
	105, // 743: auth.OrderLine.ReadGroup:output_type -> auth.OrderLine_ReadGroup_Resp
	107, // 744: auth.OrderLine.ReadGroupCount:output_type -> auth.OrderLine_ReadGroupCount_Resp
	109, // 745: auth.OrderLine.Search:output_type -> auth.OrderLine_Search_Resp
	111, // 746: auth.OrderLine.Update:output_type -> auth.OrderLine_Update_Resp
	113, // 747: auth.OrderLine.UpdateById:output_type -> auth.OrderLine_UpdateById_Resp
	115, // 748: auth.Role.Browse:output_type -> auth.Role_Browse_Resp
	117, // 749: auth.Role.BrowseMany:output_type -> auth.Role_BrowseMany_Resp
	119, // 750: auth.Role.Count:output_type -> auth.Role_Count_Resp
	121, // 751: auth.Role.Create:output_type -> auth.Role_Create_Resp
	123, // 752: auth.Role.CreateIfNotExists:output_type -> auth.Role_CreateIfNotExists_Resp
	125, // 753: auth.Role.CreateMany:output_type -> auth.Role_CreateMany_Resp
	127, // 754: auth.Role.DefaultGet:output_type -> auth.Role_DefaultGet_Resp
	129, // 755: auth.Role.Delete:output_type -> auth.Role_Delete_Resp
	131, // 756: auth.Role.DeleteById:output_type -> auth.Role_DeleteById_Resp
	133, // 757: auth.Role.Onchange:output_type -> auth.Role_Onchange_Resp
	135, // 758: auth.Role.ReadGroup:output_type -> auth.Role_ReadGroup_Resp
	137, // 759: auth.Role.ReadGroupCount:output_type -> auth.Role_ReadGroupCount_Resp
	139, // 760: auth.Role.Search:output_type -> auth.Role_Search_Resp
	141, // 761: auth.Role.Update:output_type -> auth.Role_Update_Resp
	143, // 762: auth.Role.UpdateById:output_type -> auth.Role_UpdateById_Resp
	145, // 763: auth.RoleFieldRule.Browse:output_type -> auth.RoleFieldRule_Browse_Resp
	147, // 764: auth.RoleFieldRule.BrowseMany:output_type -> auth.RoleFieldRule_BrowseMany_Resp
	149, // 765: auth.RoleFieldRule.Count:output_type -> auth.RoleFieldRule_Count_Resp
	151, // 766: auth.RoleFieldRule.Create:output_type -> auth.RoleFieldRule_Create_Resp
	153, // 767: auth.RoleFieldRule.CreateMany:output_type -> auth.RoleFieldRule_CreateMany_Resp
	155, // 768: auth.RoleFieldRule.DefaultGet:output_type -> auth.RoleFieldRule_DefaultGet_Resp
	157, // 769: auth.RoleFieldRule.Delete:output_type -> auth.RoleFieldRule_Delete_Resp
	159, // 770: auth.RoleFieldRule.DeleteById:output_type -> auth.RoleFieldRule_DeleteById_Resp
	161, // 771: auth.RoleFieldRule.Onchange:output_type -> auth.RoleFieldRule_Onchange_Resp
	163, // 772: auth.RoleFieldRule.ReadGroup:output_type -> auth.RoleFieldRule_ReadGroup_Resp
	165, // 773: auth.RoleFieldRule.ReadGroupCount:output_type -> auth.RoleFieldRule_ReadGroupCount_Resp
	167, // 774: auth.RoleFieldRule.Search:output_type -> auth.RoleFieldRule_Search_Resp
	169, // 775: auth.RoleFieldRule.Update:output_type -> auth.RoleFieldRule_Update_Resp
	171, // 776: auth.RoleFieldRule.UpdateById:output_type -> auth.RoleFieldRule_UpdateById_Resp
	173, // 777: auth.RoleInheritance.Browse:output_type -> auth.RoleInheritance_Browse_Resp
	175, // 778: auth.RoleInheritance.BrowseMany:output_type -> auth.RoleInheritance_BrowseMany_Resp
	177, // 779: auth.RoleInheritance.Count:output_type -> auth.RoleInheritance_Count_Resp
	179, // 780: auth.RoleInheritance.Create:output_type -> auth.RoleInheritance_Create_Resp
	181, // 781: auth.RoleInheritance.CreateMany:output_type -> auth.RoleInheritance_CreateMany_Resp
	183, // 782: auth.RoleInheritance.DefaultGet:output_type -> auth.RoleInheritance_DefaultGet_Resp
	185, // 783: auth.RoleInheritance.Delete:output_type -> auth.RoleInheritance_Delete_Resp
	187, // 784: auth.RoleInheritance.DeleteById:output_type -> auth.RoleInheritance_DeleteById_Resp
	189, // 785: auth.RoleInheritance.Onchange:output_type -> auth.RoleInheritance_Onchange_Resp
	191, // 786: auth.RoleInheritance.ReadGroup:output_type -> auth.RoleInheritance_ReadGroup_Resp
	193, // 787: auth.RoleInheritance.ReadGroupCount:output_type -> auth.RoleInheritance_ReadGroupCount_Resp
	195, // 788: auth.RoleInheritance.Search:output_type -> auth.RoleInheritance_Search_Resp
	197, // 789: auth.RoleInheritance.Update:output_type -> auth.RoleInheritance_Update_Resp
	199, // 790: auth.RoleInheritance.UpdateById:output_type -> auth.RoleInheritance_UpdateById_Resp
	201, // 791: auth.RoleMethodAccess.Browse:output_type -> auth.RoleMethodAccess_Browse_Resp
	203, // 792: auth.RoleMethodAccess.BrowseMany:output_type -> auth.RoleMethodAccess_BrowseMany_Resp
	205, // 793: auth.RoleMethodAccess.Count:output_type -> auth.RoleMethodAccess_Count_Resp
	207, // 794: auth.RoleMethodAccess.Create:output_type -> auth.RoleMethodAccess_Create_Resp
	209, // 795: auth.RoleMethodAccess.CreateMany:output_type -> auth.RoleMethodAccess_CreateMany_Resp
	211, // 796: auth.RoleMethodAccess.DefaultGet:output_type -> auth.RoleMethodAccess_DefaultGet_Resp
	213, // 797: auth.RoleMethodAccess.Delete:output_type -> auth.RoleMethodAccess_Delete_Resp
	215, // 798: auth.RoleMethodAccess.DeleteById:output_type -> auth.RoleMethodAccess_DeleteById_Resp
	217, // 799: auth.RoleMethodAccess.Onchange:output_type -> auth.RoleMethodAccess_Onchange_Resp
	219, // 800: auth.RoleMethodAccess.ReadGroup:output_type -> auth.RoleMethodAccess_ReadGroup_Resp
	221, // 801: auth.RoleMethodAccess.ReadGroupCount:output_type -> auth.RoleMethodAccess_ReadGroupCount_Resp
	223, // 802: auth.RoleMethodAccess.Search:output_type -> auth.RoleMethodAccess_Search_Resp
	225, // 803: auth.RoleMethodAccess.Update:output_type -> auth.RoleMethodAccess_Update_Resp
	227, // 804: auth.RoleMethodAccess.UpdateById:output_type -> auth.RoleMethodAccess_UpdateById_Resp
	229, // 805: auth.RoleRecordRule.Browse:output_type -> auth.RoleRecordRule_Browse_Resp
	231, // 806: auth.RoleRecordRule.BrowseMany:output_type -> auth.RoleRecordRule_BrowseMany_Resp
	233, // 807: auth.RoleRecordRule.Count:output_type -> auth.RoleRecordRule_Count_Resp
	235, // 808: auth.RoleRecordRule.Create:output_type -> auth.RoleRecordRule_Create_Resp
	237, // 809: auth.RoleRecordRule.CreateMany:output_type -> auth.RoleRecordRule_CreateMany_Resp
	239, // 810: auth.RoleRecordRule.DefaultGet:output_type -> auth.RoleRecordRule_DefaultGet_Resp
	241, // 811: auth.RoleRecordRule.Delete:output_type -> auth.RoleRecordRule_Delete_Resp
	243, // 812: auth.RoleRecordRule.DeleteById:output_type -> auth.RoleRecordRule_DeleteById_Resp
	245, // 813: auth.RoleRecordRule.Onchange:output_type -> auth.RoleRecordRule_Onchange_Resp
	247, // 814: auth.RoleRecordRule.ReadGroup:output_type -> auth.RoleRecordRule_ReadGroup_Resp
	249, // 815: auth.RoleRecordRule.ReadGroupCount:output_type -> auth.RoleRecordRule_ReadGroupCount_Resp
	251, // 816: auth.RoleRecordRule.Search:output_type -> auth.RoleRecordRule_Search_Resp
	253, // 817: auth.RoleRecordRule.Update:output_type -> auth.RoleRecordRule_Update_Resp
	255, // 818: auth.RoleRecordRule.UpdateById:output_type -> auth.RoleRecordRule_UpdateById_Resp
	257, // 819: auth.Session.Browse:output_type -> auth.Session_Browse_Resp
	259, // 820: auth.Session.BrowseMany:output_type -> auth.Session_BrowseMany_Resp
	260, // 821: auth.Session.CleanExpiredSessions:output_type -> auth.Session_CleanExpiredSessions_Resp
	262, // 822: auth.Session.Count:output_type -> auth.Session_Count_Resp
	264, // 823: auth.Session.Create:output_type -> auth.Session_Create_Resp
	266, // 824: auth.Session.CreateMany:output_type -> auth.Session_CreateMany_Resp
	268, // 825: auth.Session.DefaultGet:output_type -> auth.Session_DefaultGet_Resp
	270, // 826: auth.Session.Delete:output_type -> auth.Session_Delete_Resp
	272, // 827: auth.Session.DeleteById:output_type -> auth.Session_DeleteById_Resp
	274, // 828: auth.Session.GetActiveSessionsForUser:output_type -> auth.Session_GetActiveSessionsForUser_Resp
	276, // 829: auth.Session.Onchange:output_type -> auth.Session_Onchange_Resp
	278, // 830: auth.Session.ReadGroup:output_type -> auth.Session_ReadGroup_Resp
	280, // 831: auth.Session.ReadGroupCount:output_type -> auth.Session_ReadGroupCount_Resp
	282, // 832: auth.Session.RevokeAllForUser:output_type -> auth.Session_RevokeAllForUser_Resp
	284, // 833: auth.Session.RevokeSession:output_type -> auth.Session_RevokeSession_Resp
	286, // 834: auth.Session.Search:output_type -> auth.Session_Search_Resp
	288, // 835: auth.Session.Update:output_type -> auth.Session_Update_Resp
	290, // 836: auth.Session.UpdateById:output_type -> auth.Session_UpdateById_Resp
	292, // 837: auth.Session.ValidateToken:output_type -> auth.Session_ValidateToken_Resp
	294, // 838: auth.Token.Browse:output_type -> auth.Token_Browse_Resp
	296, // 839: auth.Token.BrowseMany:output_type -> auth.Token_BrowseMany_Resp
	297, // 840: auth.Token.CleanExpiredTokens:output_type -> auth.Token_CleanExpiredTokens_Resp
	299, // 841: auth.Token.Count:output_type -> auth.Token_Count_Resp
	301, // 842: auth.Token.Create:output_type -> auth.Token_Create_Resp
	303, // 843: auth.Token.CreateMany:output_type -> auth.Token_CreateMany_Resp
	305, // 844: auth.Token.CreateTokenPair:output_type -> auth.Token_CreateTokenPair_Resp
	307, // 845: auth.Token.DefaultGet:output_type -> auth.Token_DefaultGet_Resp
	309, // 846: auth.Token.Delete:output_type -> auth.Token_Delete_Resp
	311, // 847: auth.Token.DeleteById:output_type -> auth.Token_DeleteById_Resp
	313, // 848: auth.Token.Onchange:output_type -> auth.Token_Onchange_Resp
	315, // 849: auth.Token.ReadGroup:output_type -> auth.Token_ReadGroup_Resp
	317, // 850: auth.Token.ReadGroupCount:output_type -> auth.Token_ReadGroupCount_Resp
	319, // 851: auth.Token.RefreshTokens:output_type -> auth.Token_RefreshTokens_Resp
	321, // 852: auth.Token.RevokeAllUserTokens:output_type -> auth.Token_RevokeAllUserTokens_Resp
	323, // 853: auth.Token.RevokeToken:output_type -> auth.Token_RevokeToken_Resp
	325, // 854: auth.Token.RevokeUserAccessTokens:output_type -> auth.Token_RevokeUserAccessTokens_Resp
	327, // 855: auth.Token.Search:output_type -> auth.Token_Search_Resp
	329, // 856: auth.Token.Update:output_type -> auth.Token_Update_Resp
	331, // 857: auth.Token.UpdateById:output_type -> auth.Token_UpdateById_Resp
	333, // 858: auth.Token.ValidateToken:output_type -> auth.Token_ValidateToken_Resp
	335, // 859: auth.User.AssignRoles:output_type -> auth.User_AssignRoles_Resp
	337, // 860: auth.User.Browse:output_type -> auth.User_Browse_Resp
	339, // 861: auth.User.BrowseMany:output_type -> auth.User_BrowseMany_Resp
	341, // 862: auth.User.ChangePassword:output_type -> auth.User_ChangePassword_Resp
	343, // 863: auth.User.CheckMethodAccess:output_type -> auth.User_CheckMethodAccess_Resp
	345, // 864: auth.User.Count:output_type -> auth.User_Count_Resp
	347, // 865: auth.User.Create:output_type -> auth.User_Create_Resp
	349, // 866: auth.User.CreateMany:output_type -> auth.User_CreateMany_Resp
	351, // 867: auth.User.DefaultGet:output_type -> auth.User_DefaultGet_Resp
	353, // 868: auth.User.Delete:output_type -> auth.User_Delete_Resp
	355, // 869: auth.User.DeleteById:output_type -> auth.User_DeleteById_Resp
	357, // 870: auth.User.GetRecordRuleCondition:output_type -> auth.User_GetRecordRuleCondition_Resp
	359, // 871: auth.User.HasPermission:output_type -> auth.User_HasPermission_Resp
	361, // 872: auth.User.HasRole:output_type -> auth.User_HasRole_Resp
	363, // 873: auth.User.Login:output_type -> auth.User_Login_Resp
	365, // 874: auth.User.Logout:output_type -> auth.User_Logout_Resp
	367, // 875: auth.User.Onchange:output_type -> auth.User_Onchange_Resp
	369, // 876: auth.User.ReadGroup:output_type -> auth.User_ReadGroup_Resp
	371, // 877: auth.User.ReadGroupCount:output_type -> auth.User_ReadGroupCount_Resp
	373, // 878: auth.User.RefreshTokens:output_type -> auth.User_RefreshTokens_Resp
	375, // 879: auth.User.Register:output_type -> auth.User_Register_Resp
	377, // 880: auth.User.RemoveRoles:output_type -> auth.User_RemoveRoles_Resp
	379, // 881: auth.User.ResetPassword:output_type -> auth.User_ResetPassword_Resp
	381, // 882: auth.User.Search:output_type -> auth.User_Search_Resp
	383, // 883: auth.User.Update:output_type -> auth.User_Update_Resp
	385, // 884: auth.User.UpdateById:output_type -> auth.User_UpdateById_Resp
	387, // 885: auth.UserRole.Browse:output_type -> auth.UserRole_Browse_Resp
	389, // 886: auth.UserRole.BrowseMany:output_type -> auth.UserRole_BrowseMany_Resp
	391, // 887: auth.UserRole.Count:output_type -> auth.UserRole_Count_Resp
	393, // 888: auth.UserRole.Create:output_type -> auth.UserRole_Create_Resp
	395, // 889: auth.UserRole.CreateMany:output_type -> auth.UserRole_CreateMany_Resp
	397, // 890: auth.UserRole.DefaultGet:output_type -> auth.UserRole_DefaultGet_Resp
	399, // 891: auth.UserRole.Delete:output_type -> auth.UserRole_Delete_Resp
	401, // 892: auth.UserRole.DeleteById:output_type -> auth.UserRole_DeleteById_Resp
	403, // 893: auth.UserRole.Onchange:output_type -> auth.UserRole_Onchange_Resp
	405, // 894: auth.UserRole.ReadGroup:output_type -> auth.UserRole_ReadGroup_Resp
	407, // 895: auth.UserRole.ReadGroupCount:output_type -> auth.UserRole_ReadGroupCount_Resp
	409, // 896: auth.UserRole.Search:output_type -> auth.UserRole_Search_Resp
	411, // 897: auth.UserRole.Update:output_type -> auth.UserRole_Update_Resp
	413, // 898: auth.UserRole.UpdateById:output_type -> auth.UserRole_UpdateById_Resp
	691, // [691:899] is the sub-list for method output_type
	483, // [483:691] is the sub-list for method input_type
	483, // [483:483] is the sub-list for extension type_name
	483, // [483:483] is the sub-list for extension extendee
	0,   // [0:483] is the sub-list for field type_name
}

func init() { file_auth_proto_init() }
func file_auth_proto_init() {
	if File_auth_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_auth_proto_rawDesc), len(file_auth_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   414,
			NumExtensions: 0,
			NumServices:   13,
		},
		GoTypes:           file_auth_proto_goTypes,
		DependencyIndexes: file_auth_proto_depIdxs,
		MessageInfos:      file_auth_proto_msgTypes,
	}.Build()
	File_auth_proto = out.File
	file_auth_proto_goTypes = nil
	file_auth_proto_depIdxs = nil
}
