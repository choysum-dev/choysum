// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/grpc/loader"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// serviceCodec owns protobuf descriptor lookup and message conversion for internal/service.
var serviceCodec descriptorCodec

type descriptorCodec struct{}

// ProtobufMessage -> JavaScript Object
// ---------------------------------------

// messageToAny converts a protobuf message to a JavaScript compatible object
func (descriptorCodec) messageToAny(msg protoreflect.Message) (interface{}, error) {
	return converter.MessageToAny(msg)
}

// messageToMap converts a protobuf message into a map representation.
func (descriptorCodec) messageToMap(msg protoreflect.Message) (map[string]any, error) {
	return converter.MessageToMap(msg)
}

// JavaScript Object -> ProtobufMessage
// ---------------------------------------

// anyToMessage converts a JavaScript object to a protobuf message
func (descriptorCodec) anyToMessage(v interface{}, msg *dynamicpb.Message) error {
	return converter.AnyToMessage(v, msg)
}

// mapToMessage converts a map to a message
func (descriptorCodec) mapToMessage(m map[string]interface{}, msg *dynamicpb.Message) error {
	return converter.MapToMessage(m, msg)
}

// sliceToMessage converts a slice to a message, suitable for repeated fields
func (descriptorCodec) sliceToMessage(slice []interface{}, msg *dynamicpb.Message) error {
	return converter.SliceToMessage(slice, msg)
}

// convertToProtoValue converts a Go value to protoreflect.Value
func (descriptorCodec) convertToProtoValue(v interface{}, field protoreflect.FieldDescriptor) (protoreflect.Value, error) {
	return converter.ConvertToProtoValue(v, field)
}

// newMessage creates a dynamic protobuf message for desc.
func (descriptorCodec) newMessage(desc protoreflect.MessageDescriptor) *dynamicpb.Message {
	return dynamicpb.NewMessage(desc)
}

// methodDescriptor resolves a method descriptor from the global loader.
func (descriptorCodec) methodDescriptor(fullMethod string) (protoreflect.MethodDescriptor, error) {
	return loader.Global().GetMethodDescriptor(fullMethod)
}

// taskWorker returns the cached task worker descriptor set.
func (descriptorCodec) taskWorker(appName string) (protoreflect.MethodDescriptor, protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, error) {
	return taskWorkerDescriptors(appName)
}
