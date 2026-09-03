// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package converter

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testDescriptors(t *testing.T) (protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, protoreflect.EnumDescriptor) {
	t.Helper()
	file := &descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("converter_test.proto"),
		Package: proto.String("convertertest"),
		EnumType: []*descriptorpb.EnumDescriptorProto{{
			Name: proto.String("Status"),
			Value: []*descriptorpb.EnumValueDescriptorProto{
				{Name: proto.String("STATUS_UNKNOWN"), Number: proto.Int32(0)},
				{Name: proto.String("STATUS_READY"), Number: proto.Int32(1)},
			},
		}},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Nested"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   proto.String("label"),
					Number: proto.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}},
			},
			{
				Name: proto.String("Container"),
				NestedType: []*descriptorpb.DescriptorProto{{
					Name: proto.String("AttributesEntry"),
					Options: &descriptorpb.MessageOptions{
						MapEntry: proto.Bool(true),
					},
					Field: []*descriptorpb.FieldDescriptorProto{
						{Name: proto.String("key"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
						{Name: proto.String("value"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".convertertest.Nested")},
					},
				}},
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("name"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: proto.String("count"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()},
					{Name: proto.String("tags"), Number: proto.Int32(3), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: proto.String("nested"), Number: proto.Int32(4), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".convertertest.Nested")},
					{Name: proto.String("enabled"), Number: proto.Int32(5), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()},
					{Name: proto.String("ratio"), Number: proto.Int32(6), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum()},
					{Name: proto.String("score"), Number: proto.Int32(7), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum()},
					{Name: proto.String("raw"), Number: proto.Int32(8), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum()},
					{Name: proto.String("total"), Number: proto.Int32(9), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum()},
					{Name: proto.String("status"), Number: proto.Int32(10), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: proto.String(".convertertest.Status")},
					{Name: proto.String("big_count"), Number: proto.Int32(11), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum()},
					{Name: proto.String("small_count"), Number: proto.Int32(12), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum()},
					{Name: proto.String("items"), Number: proto.Int32(13), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".convertertest.Nested")},
					{Name: proto.String("attributes"), Number: proto.Int32(14), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: proto.String(".convertertest.Container.AttributesEntry")},
				},
			},
			{
				Name: proto.String("NoList"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   proto.String("name"),
					Number: proto.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}},
			},
			{
				Name: proto.String("Empty"),
			},
			{
				Name: proto.String("NumberFirst"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   proto.String("count"),
					Number: proto.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				}},
			},
			{
				Name: proto.String("RepeatedNestedOnly"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:     proto.String("items"),
					Number:   proto.Int32(1),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".convertertest.Nested"),
				}},
			},
		},
	}

	fd, err := protodesc.NewFile(file, nil)
	if err != nil {
		t.Fatalf("protodesc.NewFile: %v", err)
	}
	return fd.Messages().ByName("Container"), fd.Messages().ByName("NoList"), fd.Messages().ByName("Empty"), fd.Messages().ByName("NumberFirst"), fd.Messages().ByName("RepeatedNestedOnly"), fd.Enums().ByName("Status")
}

func TestMessageToAnyAndMapCoverWellKnownAndGenericMessages(t *testing.T) {
	containerDesc, _, _, _, _, _ := testDescriptors(t)
	container := dynamicpb.NewMessage(containerDesc)
	if err := AnyToMessage(map[string]interface{}{
		"name":    "choysum",
		"count":   float64(3),
		"tags":    []interface{}{"a", "b"},
		"nested":  map[string]interface{}{"label": "child"},
		"enabled": true,
	}, container); err != nil {
		t.Fatalf("AnyToMessage(container): %v", err)
	}

	converted, err := MessageToAny(container)
	if err != nil {
		t.Fatalf("MessageToAny(container): %v", err)
	}
	resultMap, ok := converted.(map[string]interface{})
	if !ok {
		t.Fatalf("MessageToAny(container) type = %T", converted)
	}
	if resultMap["name"] != "choysum" || resultMap["count"].(float64) != 3 || resultMap["enabled"] != true {
		t.Fatalf("unexpected converted map: %#v", resultMap)
	}

	mapped, err := MessageToMap(container)
	if err != nil {
		t.Fatalf("MessageToMap(container): %v", err)
	}
	if mapped["name"] != "choysum" {
		t.Fatalf("unexpected mapped result: %#v", mapped)
	}

	structMsg, err := structpb.NewStruct(map[string]interface{}{"name": "choysum", "count": float64(2), "flag": true})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}
	converted, err = MessageToAny(structMsg.ProtoReflect())
	if err != nil {
		t.Fatalf("MessageToAny(struct): %v", err)
	}
	structMap := converted.(map[string]interface{})
	if structMap["name"] != "choysum" || structMap["flag"] != true {
		t.Fatalf("unexpected struct conversion: %#v", structMap)
	}

	valueMsg, err := structpb.NewValue("text")
	if err != nil {
		t.Fatalf("structpb.NewValue: %v", err)
	}
	converted, err = MessageToAny(valueMsg.ProtoReflect())
	if err != nil || converted != "text" {
		t.Fatalf("MessageToAny(value) = %#v, %v", converted, err)
	}
	if _, err := MessageToMap(valueMsg.ProtoReflect()); err == nil || !strings.Contains(err.Error(), "not a map") {
		t.Fatalf("expected MessageToMap(value) error, got %v", err)
	}
}

func TestAnyToMessageSliceAndWellKnownHelpers(t *testing.T) {
	containerDesc, noListDesc, emptyDesc, _, repeatedNestedDesc, _ := testDescriptors(t)
	container := dynamicpb.NewMessage(containerDesc)
	if err := SliceToMessage([]interface{}{"a", 2}, container); err != nil {
		t.Fatalf("SliceToMessage(container): %v", err)
	}
	tagsField := containerDesc.Fields().ByName("tags")
	tags := container.Get(tagsField).List()
	if tags.Len() != 2 || tags.Get(0).String() != "a" || tags.Get(1).String() != "2" {
		t.Fatalf("unexpected tags list: %#v", tags)
	}

	noList := dynamicpb.NewMessage(noListDesc)
	if err := SliceToMessage([]interface{}{"x"}, noList); err == nil || !strings.Contains(err.Error(), "no suitable repeated field") {
		t.Fatalf("expected no repeated field error, got %v", err)
	}

	repeatedNested := dynamicpb.NewMessage(repeatedNestedDesc)
	if err := SliceToMessage([]interface{}{map[string]interface{}{"label": "first"}}, repeatedNested); err != nil {
		t.Fatalf("SliceToMessage(repeated message): %v", err)
	}
	itemsField := repeatedNestedDesc.Fields().ByName("items")
	items := repeatedNested.Get(itemsField).List()
	if items.Len() != 1 || items.Get(0).Message().Get(items.Get(0).Message().Descriptor().Fields().ByName("label")).String() != "first" {
		t.Fatalf("unexpected repeated message items: %#v", items)
	}

	empty := dynamicpb.NewMessage(emptyDesc)
	if err := AnyToMessage("ignored", empty); err != nil {
		t.Fatalf("AnyToMessage(empty): %v", err)
	}
	if empty.Range(func(protoreflect.FieldDescriptor, protoreflect.Value) bool { return false }); false {
		t.Fatal("unreachable")
	}

	valueDyn := dynamicpb.NewMessage((&structpb.Value{}).ProtoReflect().Descriptor())
	if err := AnyToMessage(map[string]interface{}{"name": "choysum"}, valueDyn); err != nil {
		t.Fatalf("AnyToMessage(value): %v", err)
	}
	got, err := extractProtoValue(valueDyn)
	if err != nil {
		t.Fatalf("extractProtoValue: %v", err)
	}
	valueMap, ok := got.(map[string]interface{})
	if !ok || valueMap["name"] != "choysum" {
		t.Fatalf("unexpected extracted struct value: %#v", got)
	}

	listDyn := dynamicpb.NewMessage((&structpb.ListValue{}).ProtoReflect().Descriptor())
	if err := AnyToMessage([]interface{}{true, "x", float64(3)}, listDyn); err != nil {
		t.Fatalf("AnyToMessage(list): %v", err)
	}
	got, err = extractProtoList(listDyn)
	if err != nil {
		t.Fatalf("extractProtoList: %v", err)
	}
	list := got.([]interface{})
	if len(list) != 3 || list[0] != true || list[1] != "x" || list[2].(float64) != 3 {
		t.Fatalf("unexpected extracted list: %#v", list)
	}

	anyValue, err := anypb.New(structpb.NewStringValue("hello"))
	if err != nil {
		t.Fatalf("anypb.New: %v", err)
	}
	got, err = extractProtoAny(anyValue)
	if err != nil {
		t.Fatalf("extractProtoAny: %v", err)
	}
	anyMap, ok := got.(map[string]interface{})
	if !ok || !strings.Contains(anyMap["@type"].(string), "google.protobuf.Value") {
		t.Fatalf("unexpected extracted any: %#v", got)
	}

	structMsg, err := structpb.NewStruct(map[string]interface{}{"flag": true})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}
	if got, err = extractWellKnownType(structMsg.ProtoReflect()); err != nil {
		t.Fatalf("extractWellKnownType(struct): %v", err)
	} else if got.(map[string]interface{})["flag"] != true {
		t.Fatalf("unexpected extractWellKnownType(struct): %#v", got)
	}

	listMsg, err := structpb.NewList([]interface{}{float64(1), "two"})
	if err != nil {
		t.Fatalf("structpb.NewList: %v", err)
	}
	if got, err = extractWellKnownType(listMsg.ProtoReflect()); err != nil {
		t.Fatalf("extractWellKnownType(list): %v", err)
	} else if len(got.([]interface{})) != 2 {
		t.Fatalf("unexpected extractWellKnownType(list): %#v", got)
	}

	custom := dynamicpb.NewMessage(containerDesc)
	if err := AnyToMessage(map[string]interface{}{"name": "fallback"}, custom); err != nil {
		t.Fatalf("AnyToMessage(custom): %v", err)
	}
	if got, err = extractWellKnownType(custom); err != nil {
		t.Fatalf("extractWellKnownType(custom fallback): %v", err)
	} else if got.(map[string]interface{})["name"] != "fallback" {
		t.Fatalf("unexpected custom fallback conversion: %#v", got)
	}
}

func TestSetProtoHelpersAndConvertToProtoValue(t *testing.T) {
	containerDesc, _, _, _, _, statusEnum := testDescriptors(t)
	valueDyn := dynamicpb.NewMessage((&structpb.Value{}).ProtoReflect().Descriptor())
	if err := setProtoValue(nil, valueDyn); err != nil {
		t.Fatalf("setProtoValue(nil): %v", err)
	}
	got, err := extractProtoValue(valueDyn)
	if err != nil || got != nil {
		t.Fatalf("extractProtoValue(nil) = %#v, %v", got, err)
	}
	if err := setProtoValue(struct{}{}, valueDyn); err == nil || !strings.Contains(err.Error(), "unsupported value type") {
		t.Fatalf("expected unsupported setProtoValue error, got %v", err)
	}
	if err := setProtoValue(true, dynamicpb.NewMessage(containerDesc)); err == nil || !strings.Contains(err.Error(), "expected google.protobuf.Value") {
		t.Fatalf("expected wrong message type error, got %v", err)
	}
	for _, check := range []struct {
		name string
		in   interface{}
		want interface{}
	}{
		{name: "bool", in: true, want: true},
		{name: "float32", in: float32(1.5), want: float64(1.5)},
		{name: "int", in: int32(3), want: float64(3)},
		{name: "string", in: "hello", want: "hello"},
		{name: "struct", in: map[string]interface{}{"x": "y"}, want: map[string]interface{}{"x": "y"}},
		{name: "list", in: []interface{}{true, float64(2)}, want: []interface{}{true, float64(2)}},
	} {
		msg := dynamicpb.NewMessage((&structpb.Value{}).ProtoReflect().Descriptor())
		if err := setProtoValue(check.in, msg); err != nil {
			t.Fatalf("setProtoValue(%s): %v", check.name, err)
		}
		got, err := extractProtoValue(msg)
		if err != nil {
			t.Fatalf("extractProtoValue(%s): %v", check.name, err)
		}
		if !reflect.DeepEqual(got, check.want) {
			t.Fatalf("%s extracted value = %#v, want %#v", check.name, got, check.want)
		}
	}

	structDyn := dynamicpb.NewMessage((&structpb.Struct{}).ProtoReflect().Descriptor())
	if err := setProtoStruct(map[string]interface{}{"name": "choysum", "enabled": true}, structDyn); err != nil {
		t.Fatalf("setProtoStruct: %v", err)
	}
	got, err = extractProtoStruct(structDyn)
	if err != nil {
		t.Fatalf("extractProtoStruct: %v", err)
	}
	if !reflect.DeepEqual(got, map[string]interface{}{"enabled": true, "name": "choysum"}) {
		t.Fatalf("unexpected extracted struct: %#v", got)
	}
	if err := setProtoStruct([]interface{}{"bad"}, structDyn); err == nil || !strings.Contains(err.Error(), "expected map[string]interface{}") {
		t.Fatalf("expected setProtoStruct type error, got %v", err)
	}
	if err := setProtoList("bad", dynamicpb.NewMessage((&structpb.ListValue{}).ProtoReflect().Descriptor())); err == nil || !strings.Contains(err.Error(), "expected []interface{}") {
		t.Fatalf("expected setProtoList type error, got %v", err)
	}

	fields := containerDesc.Fields()
	checks := []struct {
		name string
		in   interface{}
		want interface{}
	}{
		{name: "enabled", in: true, want: true},
		{name: "count", in: int64(7), want: int32(7)},
		{name: "big_count", in: int32(8), want: int64(8)},
		{name: "small_count", in: uint(5), want: uint64(5)},
		{name: "ratio", in: int32(2), want: float32(2)},
		{name: "score", in: int64(9), want: float64(9)},
		{name: "raw", in: []byte("ab"), want: []byte("ab")},
		{name: "total", in: float64(11), want: uint64(11)},
		{name: "name", in: 123, want: "123"},
		{name: "status", in: int32(1), want: protoreflect.EnumNumber(1)},
		{name: "status", in: "STATUS_READY", want: statusEnum.Values().ByName("STATUS_READY").Number()},
	}
	for _, check := range checks {
		field := fields.ByName(protoreflect.Name(check.name))
		value, err := ConvertToProtoValue(check.in, field)
		if err != nil {
			t.Fatalf("ConvertToProtoValue(%s): %v", check.name, err)
		}
		switch want := check.want.(type) {
		case bool:
			if value.Bool() != want {
				t.Fatalf("%s bool = %v, want %v", check.name, value.Bool(), want)
			}
		case int32:
			if value.Int() != int64(want) {
				t.Fatalf("%s int = %d, want %d", check.name, value.Int(), want)
			}
		case float32:
			if value.Float() != float64(want) {
				t.Fatalf("%s float = %v, want %v", check.name, value.Float(), want)
			}
		case float64:
			if value.Float() != want {
				t.Fatalf("%s double = %v, want %v", check.name, value.Float(), want)
			}
		case []byte:
			if !reflect.DeepEqual(value.Bytes(), want) {
				t.Fatalf("%s bytes = %v, want %v", check.name, value.Bytes(), want)
			}
		case uint64:
			if value.Uint() != want {
				t.Fatalf("%s uint = %d, want %d", check.name, value.Uint(), want)
			}
		case string:
			if value.String() != want {
				t.Fatalf("%s string = %q, want %q", check.name, value.String(), want)
			}
		case protoreflect.EnumNumber:
			if value.Enum() != want {
				t.Fatalf("%s enum = %d, want %d", check.name, value.Enum(), want)
			}
		}
	}

	if _, err := ConvertToProtoValue(nil, fields.ByName("name")); err == nil || !strings.Contains(err.Error(), "nil value") {
		t.Fatalf("expected nil conversion error, got %v", err)
	}
	if _, err := ConvertToProtoValue("bad", fields.ByName("enabled")); err == nil || !strings.Contains(err.Error(), "cannot convert string to bool") {
		t.Fatalf("expected bool conversion error, got %v", err)
	}
	if _, err := ConvertToProtoValue("bad", fields.ByName("count")); err == nil || !strings.Contains(err.Error(), "cannot convert string to int32") {
		t.Fatalf("expected int32 conversion error, got %v", err)
	}
	if _, err := ConvertToProtoValue(int64(math.MaxInt32)+1, fields.ByName("count")); err == nil || !strings.Contains(err.Error(), "out of int32 range") {
		t.Fatalf("expected int32 range error, got %v", err)
	}
	if _, err := ConvertToProtoValue(float64(math.MaxInt32)+1, fields.ByName("count")); err == nil || !strings.Contains(err.Error(), "out of int32 range") {
		t.Fatalf("expected float64->int32 range error, got %v", err)
	}
	if _, err := ConvertToProtoValue("bad", fields.ByName("big_count")); err == nil || !strings.Contains(err.Error(), "cannot convert string to int64") {
		t.Fatalf("expected int64 conversion error, got %v", err)
	}
	if _, err := ConvertToProtoValue("bad", fields.ByName("small_count")); err == nil || !strings.Contains(err.Error(), "cannot convert string to uint32") {
		t.Fatalf("expected uint32 conversion error, got %v", err)
	}
	if _, err := ConvertToProtoValue(-1, fields.ByName("small_count")); err == nil || !strings.Contains(err.Error(), "out of uint32 range") {
		t.Fatalf("expected uint32 range error, got %v", err)
	}
	if _, err := ConvertToProtoValue(float64(math.MaxUint32)+1, fields.ByName("small_count")); err == nil || !strings.Contains(err.Error(), "out of uint32 range") {
		t.Fatalf("expected float64->uint32 range error, got %v", err)
	}
	if _, err := ConvertToProtoValue(int64(math.MaxInt32)+1, fields.ByName("status")); err == nil || !strings.Contains(err.Error(), "out of enum range") {
		t.Fatalf("expected enum range error, got %v", err)
	}
	if _, err := ConvertToProtoValue("bad", fields.ByName("score")); err == nil || !strings.Contains(err.Error(), "cannot convert string to float64") {
		t.Fatalf("expected float64 conversion error, got %v", err)
	}
	if _, err := ConvertToProtoValue("bad", fields.ByName("raw")); err == nil || !strings.Contains(err.Error(), "cannot convert string to bytes") {
		t.Fatalf("expected bytes conversion error, got %v", err)
	}
	if _, err := ConvertToProtoValue("UNKNOWN_ENUM", fields.ByName("status")); err == nil || !strings.Contains(err.Error(), "cannot convert string to enum") {
		t.Fatalf("expected enum conversion error, got %v", err)
	}
	if !isWellKnownType((&structpb.Value{}).ProtoReflect().Descriptor()) {
		t.Fatal("expected Value descriptor to be treated as well-known type")
	}
	if isWellKnownType(containerDesc) {
		t.Fatal("did not expect custom descriptor to be treated as well-known type")
	}
}

func TestMapToMessageHandlesRepeatedMessagesMapsAndIgnoredFields(t *testing.T) {
	containerDesc, _, _, _, _, _ := testDescriptors(t)
	container := dynamicpb.NewMessage(containerDesc)
	if err := MapToMessage(map[string]interface{}{
		"missing":    "ignored",
		"name":       nil,
		"count":      "bad",
		"items":      []interface{}{map[string]interface{}{"label": "first"}, map[string]interface{}{"label": "second"}},
		"attributes": map[string]interface{}{"primary": map[string]interface{}{"label": "core"}},
	}, container); err != nil {
		t.Fatalf("MapToMessage: %v", err)
	}

	countField := containerDesc.Fields().ByName("count")
	if container.Has(countField) {
		t.Fatal("expected invalid basic conversion to be ignored")
	}

	itemsField := containerDesc.Fields().ByName("items")
	items := container.Get(itemsField).List()
	if items.Len() != 2 {
		t.Fatalf("items len = %d, want 2", items.Len())
	}
	firstItem := items.Get(0).Message()
	if firstItem.Get(firstItem.Descriptor().Fields().ByName("label")).String() != "first" {
		t.Fatalf("unexpected first item: %v", firstItem.Interface())
	}

	attrsField := containerDesc.Fields().ByName("attributes")
	attrs := container.Get(attrsField).Map()
	if attrs.Len() != 1 {
		t.Fatalf("attributes len = %d, want 1", attrs.Len())
	}
	primary := attrs.Get(protoreflect.ValueOfString("primary").MapKey()).Message()
	if primary.Get(primary.Descriptor().Fields().ByName("label")).String() != "core" {
		t.Fatalf("unexpected map value: %v", primary.Interface())
	}
}

func TestAnyToMessageSimpleValueAndMessageToAnyForAny(t *testing.T) {
	containerDesc, _, _, numberFirstDesc, _, _ := testDescriptors(t)
	container := dynamicpb.NewMessage(containerDesc)
	if err := AnyToMessage("choysum", container); err != nil {
		t.Fatalf("AnyToMessage(simple): %v", err)
	}
	nameField := containerDesc.Fields().ByName("name")
	if container.Get(nameField).String() != "choysum" {
		t.Fatalf("unexpected first-field assignment: %v", container.Get(nameField))
	}

	anyValue, err := anypb.New(structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{"x": structpb.NewStringValue("y")}}))
	if err != nil {
		t.Fatalf("anypb.New struct: %v", err)
	}
	converted, err := MessageToAny(anyValue.ProtoReflect())
	if err != nil {
		t.Fatalf("MessageToAny(any): %v", err)
	}
	anyMap, ok := converted.(map[string]interface{})
	if !ok || anyMap["@type"] == nil {
		t.Fatalf("unexpected MessageToAny(any) result: %#v", converted)
	}

	numberFirst := dynamicpb.NewMessage(numberFirstDesc)
	if err := AnyToMessage("bad", numberFirst); err == nil || !strings.Contains(err.Error(), "cannot convert string to int32") {
		t.Fatalf("expected AnyToMessage numeric conversion error, got %v", err)
	}
}

func TestTimestampWellKnownRoundTrip(t *testing.T) {
	desc := (&timestamppb.Timestamp{}).ProtoReflect().Descriptor()
	msg := dynamicpb.NewMessage(desc)

	if err := AnyToMessage("2020-01-02T03:04:05.006Z", msg); err != nil {
		t.Fatalf("AnyToMessage(string): %v", err)
	}
	got, err := MessageToAny(msg)
	if err != nil {
		t.Fatalf("MessageToAny: %v", err)
	}
	if got != "2020-01-02T03:04:05.006Z" {
		t.Fatalf("round-trip got %#v", got)
	}

	msgRFC := dynamicpb.NewMessage(desc)
	if err := AnyToMessage("2020-01-02T03:04:05Z", msgRFC); err != nil {
		t.Fatalf("AnyToMessage(RFC3339): %v", err)
	}
	gotRFC, err := MessageToAny(msgRFC)
	if err != nil || gotRFC != "2020-01-02T03:04:05Z" {
		t.Fatalf("RFC3339 round-trip %#v %v", gotRFC, err)
	}

	msg2 := dynamicpb.NewMessage(desc)
	if err := AnyToMessage(map[string]interface{}{"seconds": float64(1), "nanos": float64(2)}, msg2); err != nil {
		t.Fatalf("AnyToMessage(map): %v", err)
	}
	msg2b := dynamicpb.NewMessage(desc)
	if err := AnyToMessage(map[string]interface{}{"seconds": int64(3), "nanos": int32(4)}, msg2b); err != nil {
		t.Fatalf("AnyToMessage(map typed): %v", err)
	}
	msg2c := dynamicpb.NewMessage(desc)
	if err := AnyToMessage(map[string]interface{}{"seconds": int(5), "nanos": int(6)}, msg2c); err != nil {
		t.Fatalf("AnyToMessage(map int): %v", err)
	}
	msg2d := dynamicpb.NewMessage(desc)
	if err := AnyToMessage(map[string]interface{}{"seconds": json.Number("7"), "nanos": int64(8)}, msg2d); err != nil {
		t.Fatalf("AnyToMessage(map json.Number): %v", err)
	}
	if err := AnyToMessage(map[string]interface{}{"seconds": true}, dynamicpb.NewMessage(desc)); err == nil {
		t.Fatal("expected bad seconds type")
	}
	if err := AnyToMessage(map[string]interface{}{"nanos": true}, dynamicpb.NewMessage(desc)); err == nil {
		t.Fatal("expected bad nanos type")
	}
	if err := AnyToMessage(map[string]interface{}{"seconds": json.Number("x")}, dynamicpb.NewMessage(desc)); err == nil {
		t.Fatal("expected bad json.Number seconds")
	}

	msg3 := dynamicpb.NewMessage(desc)
	if err := AnyToMessage(time.Unix(10, 0).UTC(), msg3); err != nil {
		t.Fatalf("AnyToMessage(time): %v", err)
	}
	msg4 := dynamicpb.NewMessage(desc)
	if err := AnyToMessage(float64(11.5), msg4); err != nil {
		t.Fatalf("AnyToMessage(float): %v", err)
	}
	msg4b := dynamicpb.NewMessage(desc)
	if err := AnyToMessage(float64(-11.25), msg4b); err != nil {
		t.Fatalf("AnyToMessage(neg float): %v", err)
	}
	msg5 := dynamicpb.NewMessage(desc)
	if err := AnyToMessage(12, msg5); err != nil {
		t.Fatalf("AnyToMessage(int): %v", err)
	}
	msg6 := dynamicpb.NewMessage(desc)
	if err := AnyToMessage(int64(13), msg6); err != nil {
		t.Fatalf("AnyToMessage(int64): %v", err)
	}
	if err := AnyToMessage(nil, dynamicpb.NewMessage(desc)); err != nil {
		t.Fatalf("AnyToMessage(nil): %v", err)
	}
	if err := AnyToMessage("not-a-time", dynamicpb.NewMessage(desc)); err == nil {
		t.Fatal("expected invalid timestamp string error")
	}
	if err := AnyToMessage([]byte("x"), dynamicpb.NewMessage(desc)); err == nil {
		t.Fatal("expected unsupported type error")
	}
	if err := setProtoTimestamp(nil, nil); err == nil {
		t.Fatal("expected nil msg error")
	}
	if _, err := extractProtoTimestamp(nil); err == nil {
		t.Fatal("expected nil extract error")
	}
}
