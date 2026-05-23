// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package oerrors

import (
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// ErrorInfo stores structured error details.
type ErrorInfo struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Error ID used for tracing and log correlation.
	ErrorId string `protobuf:"bytes,1,opt,name=error_id,json=errorId,proto3" json:"error_id,omitempty"`
	// System or service domain where the error occurred.
	// For example: "auth.choysum", "sale.choysum".
	Domain string `protobuf:"bytes,2,opt,name=domain,proto3" json:"domain,omitempty"`
	// Machine-readable static error code.
	// For example: "USERNAME_TAKEN", "EMAIL_ALREADY_EXISTS".
	Code string `protobuf:"bytes,3,opt,name=code,proto3" json:"code,omitempty"`
	// Detailed error message suitable for user display.
	// It may contain formatted content such as "username {username} is already in use".
	Message string `protobuf:"bytes,4,opt,name=message,proto3" json:"message,omitempty"`
	// gRPC status code mapped directly to google.golang.org/grpc/codes.
	GrpcCode int32 `protobuf:"varint,5,opt,name=grpc_code,json=grpcCode,proto3" json:"grpc_code,omitempty"`
	// Key-value metadata associated with the error.
	// For example: {"username": "user123", "field": "username"}.
	Metadata      map[string]string `protobuf:"bytes,6,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ErrorInfo) Reset() {
	*x = ErrorInfo{}
	mi := &file_error_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ErrorInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ErrorInfo) ProtoMessage() {}

func (x *ErrorInfo) ProtoReflect() protoreflect.Message {
	mi := &file_error_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ErrorInfo.ProtoReflect.Descriptor instead.
func (*ErrorInfo) Descriptor() ([]byte, []int) {
	return file_error_proto_rawDescGZIP(), []int{0}
}

func (x *ErrorInfo) GetErrorId() string {
	if x != nil {
		return x.ErrorId
	}
	return ""
}

func (x *ErrorInfo) GetDomain() string {
	if x != nil {
		return x.Domain
	}
	return ""
}

func (x *ErrorInfo) GetCode() string {
	if x != nil {
		return x.Code
	}
	return ""
}

func (x *ErrorInfo) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *ErrorInfo) GetGrpcCode() int32 {
	if x != nil {
		return x.GrpcCode
	}
	return 0
}

func (x *ErrorInfo) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

var File_error_proto protoreflect.FileDescriptor

const file_error_proto_rawDesc = "" +
	"\n" +
	"\verror.proto\x12\aoerrors\"\x84\x02\n" +
	"\tErrorInfo\x12\x19\n" +
	"\berror_id\x18\x01 \x01(\tR\aerrorId\x12\x16\n" +
	"\x06domain\x18\x02 \x01(\tR\x06domain\x12\x12\n" +
	"\x04code\x18\x03 \x01(\tR\x04code\x12\x18\n" +
	"\amessage\x18\x04 \x01(\tR\amessage\x12\x1b\n" +
	"\tgrpc_code\x18\x05 \x01(\x05R\bgrpcCode\x12<\n" +
	"\bmetadata\x18\x06 \x03(\v2 .oerrors.ErrorInfo.MetadataEntryR\bmetadata\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01B4Z2github.com/choysum-dev/choysum/pkg/oerrors;oerrorsb\x06proto3"

var (
	file_error_proto_rawDescOnce sync.Once
	file_error_proto_rawDescData []byte
)

func file_error_proto_rawDescGZIP() []byte {
	file_error_proto_rawDescOnce.Do(func() {
		file_error_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_error_proto_rawDesc), len(file_error_proto_rawDesc)))
	})
	return file_error_proto_rawDescData
}

var file_error_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_error_proto_goTypes = []any{
	(*ErrorInfo)(nil), // 0: oerrors.ErrorInfo
	nil,               // 1: oerrors.ErrorInfo.MetadataEntry
}
var file_error_proto_depIdxs = []int32{
	1, // 0: oerrors.ErrorInfo.metadata:type_name -> oerrors.ErrorInfo.MetadataEntry
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_error_proto_init() }
func file_error_proto_init() {
	if File_error_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_error_proto_rawDesc), len(file_error_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_error_proto_goTypes,
		DependencyIndexes: file_error_proto_depIdxs,
		MessageInfos:      file_error_proto_msgTypes,
	}.Build()
	File_error_proto = out.File
	file_error_proto_goTypes = nil
	file_error_proto_depIdxs = nil
}
