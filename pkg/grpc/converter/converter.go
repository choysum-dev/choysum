// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package converter

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"

	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/structpb"
)

// MessageToMap converts a dynamicpb.Message to a map[string]interface{}
// This is a helper wrapper around MessageToAny that asserts the result is a map.
func MessageToMap(msg protoreflect.Message) (map[string]interface{}, error) {
	res, err := MessageToAny(msg)
	if err != nil {
		return nil, err
	}
	if m, ok := res.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("result is not a map, got %T", res)
}

// MessageToAny converts a protobuf message to a JavaScript compatible object
func MessageToAny(msg protoreflect.Message) (interface{}, error) {
	// Check for Well-Known Types
	if isWellKnownType(msg.Descriptor()) {
		return extractWellKnownType(msg)
	}

	// For non-special types, use standard JSON conversion
	dynMsg := dynamicpb.NewMessage(msg.Descriptor())
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		dynMsg.Set(fd, v)
		return true
	})

	// Use protojson to convert message to JSON
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}

	jsonBytes, err := marshaler.Marshal(dynMsg)
	if err != nil {
		return nil, xfmt.Errorf("marshal message to JSON failed: %w", err)
	}

	// Parse as generic interface{}
	var result interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, xfmt.Errorf("unmarshal JSON to interface failed: %w", err)
	}

	return result, nil
}

// extractWellKnownType handles special processing for Well-Known Types
func extractWellKnownType(msg protoreflect.Message) (interface{}, error) {
	// Get message type full name
	fullName := string(msg.Descriptor().FullName())

	protoMsg := msg.Interface()
	if protoMsg == nil {
		dynMsg := dynamicpb.NewMessage(msg.Descriptor())
		msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			dynMsg.Set(fd, v)
			return true
		})
		protoMsg = dynMsg
	}

	// Select appropriate handler function based on message type
	switch fullName {
	case "google.protobuf.Value":
		return extractProtoValue(protoMsg)
	case "google.protobuf.Struct":
		return extractProtoStruct(protoMsg)
	case "google.protobuf.ListValue":
		return extractProtoList(protoMsg)
	case "google.protobuf.Any":
		return extractProtoAny(protoMsg)
	default:
		// Use generic JSON conversion as fallback
		return messageToAnyGeneric(protoMsg)
	}
}

// extractProtoValue extracts a native value from google.protobuf.Value
func extractProtoValue(msg protoreflect.ProtoMessage) (interface{}, error) {
	fields := msg.ProtoReflect().Descriptor().Fields()

	// Try to get each possible field value
	var fieldValue protoreflect.Value

	// Check null_value
	if f := fields.ByName("null_value"); f != nil && msg.ProtoReflect().Has(f) {
		return nil, nil
	}

	// Check number_value
	if f := fields.ByName("number_value"); f != nil && msg.ProtoReflect().Has(f) {
		fieldValue = msg.ProtoReflect().Get(f)
		return fieldValue.Float(), nil
	}

	// Check string_value
	if f := fields.ByName("string_value"); f != nil && msg.ProtoReflect().Has(f) {
		fieldValue = msg.ProtoReflect().Get(f)
		return fieldValue.String(), nil
	}

	// Check bool_value
	if f := fields.ByName("bool_value"); f != nil && msg.ProtoReflect().Has(f) {
		fieldValue = msg.ProtoReflect().Get(f)
		return fieldValue.Bool(), nil
	}

	// Check struct_value
	if f := fields.ByName("struct_value"); f != nil && msg.ProtoReflect().Has(f) {
		fieldValue = msg.ProtoReflect().Get(f)
		return extractProtoStruct(fieldValue.Message().Interface())
	}

	// Check list_value
	if f := fields.ByName("list_value"); f != nil && msg.ProtoReflect().Has(f) {
		fieldValue = msg.ProtoReflect().Get(f)
		return extractProtoList(fieldValue.Message().Interface())
	}

	// Allow unset values to be treated as null instead of returning an error.
	return nil, nil
}

// extractProtoStruct extracts a map from google.protobuf.Struct
func extractProtoStruct(msg protoreflect.ProtoMessage) (interface{}, error) {
	result := make(map[string]interface{})

	fields := msg.ProtoReflect().Descriptor().Fields()
	fieldsField := fields.ByName("fields")

	if fieldsField == nil || !msg.ProtoReflect().Has(fieldsField) {
		return result, nil
	}

	fieldsMap := msg.ProtoReflect().Get(fieldsField).Map()
	if fieldsMap.Len() == 0 {
		return result, nil
	}

	fieldsMap.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
		valueMsg := value.Message().Interface()
		extractedValue, err := extractProtoValue(valueMsg)
		if err != nil {
			return true // continue
		}
		result[key.String()] = extractedValue
		return true
	})

	return result, nil
}

// extractProtoList extracts an array from google.protobuf.ListValue
func extractProtoList(msg protoreflect.ProtoMessage) (interface{}, error) {
	result := make([]interface{}, 0)

	fields := msg.ProtoReflect().Descriptor().Fields()
	valuesField := fields.ByName("values")

	if valuesField == nil || !msg.ProtoReflect().Has(valuesField) {
		return result, nil
	}

	valuesList := msg.ProtoReflect().Get(valuesField).List()
	if valuesList.Len() == 0 {
		return result, nil
	}

	for i := 0; i < valuesList.Len(); i++ {
		valueMsg := valuesList.Get(i).Message().Interface()
		extractedValue, err := extractProtoValue(valueMsg)
		if err != nil {
			continue
		}
		result = append(result, extractedValue)
	}

	return result, nil
}

// extractProtoAny extracts value from google.protobuf.Any
func extractProtoAny(msg protoreflect.ProtoMessage) (interface{}, error) {
	// Implementation would depend on specific Any type handling
	// Using JSON conversion for simplicity
	return messageToAnyGeneric(msg)
}

// messageToAnyGeneric provides a generic JSON conversion method
func messageToAnyGeneric(msg protoreflect.ProtoMessage) (interface{}, error) {
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}
	jsonBytes, err := marshaler.Marshal(msg)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// JavaScript Object -> ProtobufMessage
// ---------------------------------------

// AnyToMessage converts a JavaScript object to a protobuf message
func AnyToMessage(v interface{}, msg *dynamicpb.Message) error {
	// Check for Well-Known Types
	if msg.Descriptor().FullName() == "google.protobuf.Value" {
		return setProtoValue(v, msg)
	} else if msg.Descriptor().FullName() == "google.protobuf.Struct" {
		return setProtoStruct(v, msg)
	} else if msg.Descriptor().FullName() == "google.protobuf.ListValue" {
		return setProtoList(v, msg)
	}

	// Process based on value type
	switch val := v.(type) {
	case map[string]interface{}:
		return MapToMessage(val, msg)
	case []interface{}:
		return SliceToMessage(val, msg)
	default:
		// For simple value types, try to set to the first field
		if msg.Descriptor().Fields().Len() > 0 {
			field := msg.Descriptor().Fields().Get(0)
			protoValue, err := ConvertToProtoValue(v, field)
			if err != nil {
				return err
			}
			msg.Set(field, protoValue)
		}
		return nil
	}
}

// MapToMessage converts a map to a message
func MapToMessage(m map[string]interface{}, msg *dynamicpb.Message) error {
	for k, v := range m {
		if v == nil {
			continue
		}

		field := msg.Descriptor().Fields().ByTextName(k)
		if field == nil {
			continue // Skip unknown fields
		}

		if field.IsList() {
			if slice, ok := v.([]interface{}); ok {
				listValue := msg.Mutable(field).List()
				for _, item := range slice {
					if field.Message() != nil {
						nestedMsg := dynamicpb.NewMessage(field.Message())
						if err := AnyToMessage(item, nestedMsg); err != nil {
							return err
						}
						listValue.Append(protoreflect.ValueOf(nestedMsg))
						continue
					}

					protoValue, err := ConvertToProtoValue(item, field)
					if err != nil {
						continue
					}
					listValue.Append(protoValue)
				}
			}
		} else if field.IsMap() {
			// Handle map fields
			if mapValue, ok := v.(map[string]interface{}); ok {
				fieldMap := msg.Mutable(field).Map()
				valueDesc := field.MapValue().Message()
				for mapKey, mapVal := range mapValue {
					valueMsg := dynamicpb.NewMessage(valueDesc)
					if err := AnyToMessage(mapVal, valueMsg); err != nil {
						return err
					}

					var keyValue protoreflect.Value
					switch field.MapKey().Kind() {
					case protoreflect.StringKind:
						keyValue = protoreflect.ValueOfString(mapKey)
					default:
						continue
					}

					fieldMap.Set(protoreflect.MapKey(keyValue), protoreflect.ValueOf(valueMsg))
				}
			}
		} else if field.Message() != nil {
			// Regular message fields
			nestedMsg := dynamicpb.NewMessage(field.Message())
			if err := AnyToMessage(v, nestedMsg); err != nil {
				return err
			}
			msg.Set(field, protoreflect.ValueOf(nestedMsg))
		} else {
			// Basic type fields
			protoValue, err := ConvertToProtoValue(v, field)
			if err != nil {
				continue // Ignore fields with conversion errors
			}
			msg.Set(field, protoValue)
		}
	}

	return nil
}

// SliceToMessage converts a slice to a message, suitable for repeated fields
func SliceToMessage(slice []interface{}, msg *dynamicpb.Message) error {
	// Find the first list field
	var field protoreflect.FieldDescriptor
	for i := 0; i < msg.Descriptor().Fields().Len(); i++ {
		f := msg.Descriptor().Fields().Get(i)
		if f.IsList() && !f.IsMap() {
			field = f
			break
		}
	}

	if field == nil {
		return xfmt.Errorf("no suitable repeated field found in message %s", msg.Descriptor().FullName())
	}

	// Populate list field
	list := msg.Mutable(field).List()
	for _, item := range slice {
		if field.Message() != nil {
			nestedMsg := dynamicpb.NewMessage(field.Message())
			if err := AnyToMessage(item, nestedMsg); err != nil {
				return err
			}
			list.Append(protoreflect.ValueOf(nestedMsg))
		} else {
			protoValue, err := ConvertToProtoValue(item, field)
			if err != nil {
				continue
			}
			list.Append(protoValue)
		}
	}

	return nil
}

// Well-Known Types Handling
// ---------------------------------------

// setProtoValue sets a value to google.protobuf.Value
func setProtoValue(v interface{}, msg *dynamicpb.Message) error {
	// Ensure we're handling google.protobuf.Value
	if msg.Descriptor().FullName() != "google.protobuf.Value" {
		return xfmt.Errorf("expected google.protobuf.Value, got %s", msg.Descriptor().FullName())
	}

	// Set field based on value type
	switch val := v.(type) {
	case nil:
		// null_value
		nullField := msg.Descriptor().Fields().ByName("null_value")
		if nullField != nil {
			// Fix: use ValueOfEnum instead of ValueOf.
			msg.Set(nullField, protoreflect.ValueOfEnum(protoreflect.EnumNumber(structpb.NullValue_NULL_VALUE)))
		}

	// Other type handlers remain unchanged.
	case bool:
		// bool_value
		boolField := msg.Descriptor().Fields().ByName("bool_value")
		if boolField != nil {
			msg.Set(boolField, protoreflect.ValueOf(val))
		}

	case float64:
		// number_value
		numField := msg.Descriptor().Fields().ByName("number_value")
		if numField != nil {
			msg.Set(numField, protoreflect.ValueOf(val))
		}

	case float32:
		// number_value
		numField := msg.Descriptor().Fields().ByName("number_value")
		if numField != nil {
			msg.Set(numField, protoreflect.ValueOf(float64(val)))
		}

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// number_value (convert integers to float)
		numField := msg.Descriptor().Fields().ByName("number_value")
		if numField != nil {
			floatVal := reflect.ValueOf(val).Convert(reflect.TypeOf(float64(0))).Float()
			msg.Set(numField, protoreflect.ValueOf(floatVal))
		}

	case string:
		// string_value
		strField := msg.Descriptor().Fields().ByName("string_value")
		if strField != nil {
			msg.Set(strField, protoreflect.ValueOf(val))
		}

	case map[string]interface{}:
		// struct_value
		structField := msg.Descriptor().Fields().ByName("struct_value")
		if structField != nil {
			structMsg := dynamicpb.NewMessage(structField.Message())
			if err := setProtoStruct(val, structMsg); err != nil {
				return err
			}
			msg.Set(structField, protoreflect.ValueOf(structMsg))
		}

	case []interface{}:
		// list_value
		listField := msg.Descriptor().Fields().ByName("list_value")
		if listField != nil {
			listMsg := dynamicpb.NewMessage(listField.Message())
			if err := setProtoList(val, listMsg); err != nil {
				return err
			}
			msg.Set(listField, protoreflect.ValueOf(listMsg))
		}

	default:
		return xfmt.Errorf("unsupported value type for google.protobuf.Value: %T", v)
	}

	return nil
}

// setProtoStruct sets a map to google.protobuf.Struct
func setProtoStruct(v interface{}, msg *dynamicpb.Message) error {
	// Ensure we're handling google.protobuf.Struct
	if msg.Descriptor().FullName() != "google.protobuf.Struct" {
		return xfmt.Errorf("expected google.protobuf.Struct, got %s", msg.Descriptor().FullName())
	}

	mapValue, ok := v.(map[string]interface{})
	if !ok {
		return xfmt.Errorf("expected map[string]interface{}, got %T", v)
	}

	// Get fields field
	fieldsField := msg.Descriptor().Fields().ByName("fields")
	if fieldsField == nil {
		return xfmt.Errorf("fields field not found in google.protobuf.Struct")
	}

	// Create and populate fields Map
	fieldsMap := msg.Mutable(fieldsField).Map()
	for key, val := range mapValue {
		// Create Value message for each value
		valueDesc := fieldsField.MapValue().Message()
		valueMsg := dynamicpb.NewMessage(valueDesc)

		if err := setProtoValue(val, valueMsg); err != nil {
			return err
		}

		fieldsMap.Set(protoreflect.ValueOf(key).MapKey(), protoreflect.ValueOf(valueMsg))
	}

	return nil
}

// setProtoList sets an array to google.protobuf.ListValue
func setProtoList(v interface{}, msg *dynamicpb.Message) error {
	// Ensure we're handling google.protobuf.ListValue
	if msg.Descriptor().FullName() != "google.protobuf.ListValue" {
		return xfmt.Errorf("expected google.protobuf.ListValue, got %s", msg.Descriptor().FullName())
	}

	sliceValue, ok := v.([]interface{})
	if !ok {
		return xfmt.Errorf("expected []interface{}, got %T", v)
	}

	// Get values field
	valuesField := msg.Descriptor().Fields().ByName("values")
	if valuesField == nil {
		return xfmt.Errorf("values field not found in google.protobuf.ListValue")
	}

	// Create and populate values List
	valuesList := msg.Mutable(valuesField).List()
	for _, val := range sliceValue {
		// Create Value message for each value
		valueDesc := valuesField.Message()
		valueMsg := dynamicpb.NewMessage(valueDesc)

		if err := setProtoValue(val, valueMsg); err != nil {
			return err
		}

		valuesList.Append(protoreflect.ValueOf(valueMsg))
	}

	return nil
}

// Helper Functions
// ---------------------------------------

// isWellKnownType checks if a descriptor is for a Well-Known Type
func isWellKnownType(desc protoreflect.MessageDescriptor) bool {
	fullName := string(desc.FullName())
	return fullName == "google.protobuf.Value" ||
		fullName == "google.protobuf.Struct" ||
		fullName == "google.protobuf.ListValue" ||
		fullName == "google.protobuf.Any"
}

// ConvertToProtoValue converts a Go value to protoreflect.Value
func ConvertToProtoValue(v interface{}, field protoreflect.FieldDescriptor) (protoreflect.Value, error) {
	if v == nil {
		return protoreflect.Value{}, xfmt.Errorf("nil value for field %s", field.FullName())
	}

	switch field.Kind() {
	case protoreflect.BoolKind:
		if b, ok := v.(bool); ok {
			return protoreflect.ValueOfBool(b), nil
		}

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		i32, err := convertToInt32(v)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(i32), nil

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		var i64 int64
		switch val := v.(type) {
		case int:
			i64 = int64(val)
		case int32:
			i64 = int64(val)
		case int64:
			i64 = val
		case float64:
			i64 = int64(val)
		default:
			return protoreflect.Value{}, xfmt.Errorf("cannot convert %T to int64", v)
		}
		return protoreflect.ValueOfInt64(i64), nil

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		u32, err := convertToUint32(v)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint32(u32), nil

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		var u64 uint64
		switch val := v.(type) {
		case int:
			if val < 0 {
				return protoreflect.Value{}, xfmt.Errorf("int %d out of uint64 range", val)
			}
			u64 = uint64(val)
		case uint:
			u64 = uint64(val)
		case uint64:
			u64 = val
		case float64:
			if val < 0 {
				return protoreflect.Value{}, xfmt.Errorf("float64 %v out of uint64 range", val)
			}
			u64 = uint64(val)
		default:
			return protoreflect.Value{}, xfmt.Errorf("cannot convert %T to uint64", v)
		}
		return protoreflect.ValueOfUint64(u64), nil

	case protoreflect.FloatKind:
		var f32 float32
		switch val := v.(type) {
		case float32:
			f32 = val
		case float64:
			f32 = float32(val)
		case int:
			f32 = float32(val)
		case int32:
			f32 = float32(val)
		case int64:
			f32 = float32(val)
		default:
			return protoreflect.Value{}, xfmt.Errorf("cannot convert %T to float32", v)
		}
		return protoreflect.ValueOfFloat32(f32), nil

	case protoreflect.DoubleKind:
		var f64 float64
		switch val := v.(type) {
		case float32:
			f64 = float64(val)
		case float64:
			f64 = val
		case int:
			f64 = float64(val)
		case int32:
			f64 = float64(val)
		case int64:
			f64 = float64(val)
		default:
			return protoreflect.Value{}, xfmt.Errorf("cannot convert %T to float64", v)
		}
		return protoreflect.ValueOfFloat64(f64), nil

	case protoreflect.StringKind:
		if s, ok := v.(string); ok {
			return protoreflect.ValueOfString(s), nil
		}
		// Try to convert non-string values to string
		return protoreflect.ValueOfString(fmt.Sprintf("%v", v)), nil

	case protoreflect.BytesKind:
		if b, ok := v.([]byte); ok {
			return protoreflect.ValueOfBytes(b), nil
		}

	case protoreflect.EnumKind:
		// Handle enum values, typically mapping integers or strings to enums
		switch val := v.(type) {
		case int, int32, int64:
			enumNumber, err := convertToEnumNumber(val)
			if err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfEnum(enumNumber), nil
		case string:
			// Map from enum name to enum value
			enumType := field.Enum()
			for i := 0; i < enumType.Values().Len(); i++ {
				enumVal := enumType.Values().Get(i)
				if string(enumVal.Name()) == val {
					return protoreflect.ValueOfEnum(enumVal.Number()), nil
				}
			}
		}
	}

	// If unable to handle, return error
	return protoreflect.Value{}, xfmt.Errorf("cannot convert %T to %s", v, field.Kind())
}

func convertToInt32(v interface{}) (int32, error) {
	switch val := v.(type) {
	case int:
		if val < math.MinInt32 || val > math.MaxInt32 {
			return 0, xfmt.Errorf("int %d out of int32 range", val)
		}
		return int32(val), nil
	case int32:
		return val, nil
	case int64:
		if val < math.MinInt32 || val > math.MaxInt32 {
			return 0, xfmt.Errorf("int64 %d out of int32 range", val)
		}
		return int32(val), nil
	case float64:
		if val < float64(math.MinInt32) || val > float64(math.MaxInt32) {
			return 0, xfmt.Errorf("float64 %v out of int32 range", val)
		}
		return int32(val), nil
	default:
		return 0, xfmt.Errorf("cannot convert %T to int32", v)
	}
}

func convertToUint32(v interface{}) (uint32, error) {
	switch val := v.(type) {
	case int:
		if val < 0 || uint64(val) > math.MaxUint32 {
			return 0, xfmt.Errorf("int %d out of uint32 range", val)
		}
		return uint32(val), nil
	case uint:
		if uint64(val) > math.MaxUint32 {
			return 0, xfmt.Errorf("uint %d out of uint32 range", val)
		}
		return uint32(val), nil
	case uint32:
		return val, nil
	case float64:
		if val < 0 || val > float64(math.MaxUint32) {
			return 0, xfmt.Errorf("float64 %v out of uint32 range", val)
		}
		return uint32(val), nil
	default:
		return 0, xfmt.Errorf("cannot convert %T to uint32", v)
	}
}

func convertToEnumNumber(v interface{}) (protoreflect.EnumNumber, error) {
	switch val := v.(type) {
	case int:
		if val < math.MinInt32 || val > math.MaxInt32 {
			return 0, xfmt.Errorf("int %d out of enum range", val)
		}
		return protoreflect.EnumNumber(val), nil
	case int32:
		return protoreflect.EnumNumber(val), nil
	case int64:
		if val < math.MinInt32 || val > math.MaxInt32 {
			return 0, xfmt.Errorf("int64 %d out of enum range", val)
		}
		return protoreflect.EnumNumber(val), nil
	default:
		return 0, xfmt.Errorf("cannot convert %T to enum", v)
	}
}
