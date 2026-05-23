// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gots

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGeneratorGenerate_NilRequest(t *testing.T) {
	g := NewGenerator()
	_, err := g.Generate(nil)
	if err == nil {
		t.Fatalf("expected error for nil request")
	}
	if !strings.Contains(err.Error(), "nil request") {
		t.Fatalf("expected nil request error, got %v", err)
	}
}

func TestGeneratorGenerate_UnsupportedParameter(t *testing.T) {
	g := NewGenerator()
	_, err := g.Generate(&pluginpb.CodeGeneratorRequest{Parameter: strPtr("target=js")})
	if err == nil {
		t.Fatalf("expected error for unsupported parameter")
	}
	if !strings.Contains(err.Error(), "unsupported parameter") {
		t.Fatalf("expected unsupported parameter error, got %v", err)
	}
}

func TestGeneratorGenerate_UnsupportedDocumentedOutOfScopeParameters(t *testing.T) {
	g := NewGenerator()
	cases := []string{
		"target=ts,target=dts",
		"target=ts,js_import_style=legacy_commonjs",
		"target=ts,import_extension=js",
		"target=ts,import_extension=ts",
		"target=ts,json_types=true",
		"target=ts,valid_types=foo.bar.Baz",
	}

	for _, parameter := range cases {
		t.Run(parameter, func(t *testing.T) {
			_, err := g.Generate(&pluginpb.CodeGeneratorRequest{Parameter: strPtr(parameter)})
			if err == nil {
				t.Fatalf("expected unsupported parameter error for %q", parameter)
			}
			if !strings.Contains(err.Error(), "unsupported parameter") {
				t.Fatalf("expected unsupported parameter error, got %v", err)
			}
		})
	}
}

func TestGeneratorGenerate_MinimalSuccess(t *testing.T) {
	g := NewGenerator()
	resp, err := g.Generate(minimalRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetFile()) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(resp.GetFile()))
	}
	if resp.GetFile()[0].GetName() != "minimal_pb.ts" {
		t.Fatalf("unexpected generated file name: %s", resp.GetFile()[0].GetName())
	}
	if !strings.Contains(resp.GetFile()[0].GetContent(), "export const file_minimal") {
		t.Fatalf("generated content missing file descriptor const")
	}
	if !strings.Contains(resp.GetFile()[0].GetContent(), "// Message Demo") {
		t.Fatalf("generated content missing message comment")
	}
}

func TestGeneratorGenerate_Int64MapsToBigint(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("int64.proto"),
		Package: strPtr("sample"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Counter"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   strPtr("seq"),
						Number: int32Ptr(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
					},
				},
			},
		},
	}
	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"int64.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "seq: bigint;") {
		t.Fatalf("expected int64 field to map to bigint, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2Presence(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_presence.proto"),
		Package: strPtr("sample"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("PresenceDemo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   strPtr("opt_name"),
						Number: int32Ptr(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:   strPtr("req_id"),
						Number: int32Ptr(2),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
					},
					{
						Name:   strPtr("tags"),
						Number: int32Ptr(3),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_presence.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "optName?: string;") {
		t.Fatalf("expected proto2 optional field to be optional, got:\n%s", content)
	}
	if !strings.Contains(content, "reqId: number;") {
		t.Fatalf("expected proto2 required field to stay required, got:\n%s", content)
	}
	if !strings.Contains(content, "tags: string[];") {
		t.Fatalf("expected repeated field to stay non-optional array, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2DefaultsShape(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_defaults.proto"),
		Package: strPtr("sample"),
		Syntax:  strPtr("proto2"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Mode"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("MODE_UNSPECIFIED"), Number: int32Ptr(0)},
					{Name: strPtr("MODE_FAST"), Number: int32Ptr(1)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("DefaultsDemo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("name"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), DefaultValue: strPtr("guest")},
					{Name: strPtr("retries"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("3")},
					{Name: strPtr("mode"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.Mode"), DefaultValue: strPtr("MODE_FAST")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_defaults.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "name?: string;") {
		t.Fatalf("expected proto2 optional default field to stay optional, got:\n%s", content)
	}
	if !strings.Contains(content, "retries: number;") {
		t.Fatalf("expected proto2 required default field to stay required, got:\n%s", content)
	}
	if !strings.Contains(content, "mode?: Mode;") {
		t.Fatalf("expected proto2 optional enum default field to stay optional enum type, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2CrossFileEnumDefaultShape(t *testing.T) {
	g := NewGenerator()
	dep := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_dep.proto"),
		Package: strPtr("sample.dep"),
		Syntax:  strPtr("proto2"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Mode"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("MODE_UNSPECIFIED"), Number: int32Ptr(0)},
					{Name: strPtr("MODE_FAST"), Number: int32Ptr(1)},
				},
			},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("proto2_main.proto"),
		Package:    strPtr("sample.main"),
		Syntax:     strPtr("proto2"),
		Dependency: []string{"proto2_dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("CrossDefaults"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("mode"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.dep.Mode"), DefaultValue: strPtr("MODE_FAST")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "import type { Mode } from './proto2_dep_pb';") {
		t.Fatalf("expected cross-file enum type import for default field, got:\n%s", content)
	}
	if !strings.Contains(content, "mode?: Mode;") {
		t.Fatalf("expected proto2 cross-file optional enum default field to stay optional enum type, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2BytesDefaultsShape(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_bytes_defaults.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("BytesDefaultsDemo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("token"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(), DefaultValue: strPtr("abc")},
					{Name: strPtr("nonce"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum(), DefaultValue: strPtr("xyz")},
					{Name: strPtr("chunks"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum()},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_bytes_defaults.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "token?: Uint8Array;") {
		t.Fatalf("expected proto2 optional bytes default field to stay optional Uint8Array, got:\n%s", content)
	}
	if !strings.Contains(content, "nonce: Uint8Array;") {
		t.Fatalf("expected proto2 required bytes default field to stay required Uint8Array, got:\n%s", content)
	}
	if !strings.Contains(content, "chunks: Uint8Array[];") {
		t.Fatalf("expected repeated bytes field to stay non-optional Uint8Array array, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2OneofPresenceShape(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_oneof_presence.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:      strPtr("Proto2OneofDemo"),
				OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: strPtr("choice")}},
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("name"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("req_id"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()},
					{Name: strPtr("choice_text"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("choice_num"), Number: int32Ptr(4), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("tags"), Number: int32Ptr(5), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_oneof_presence.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "name?: string;") {
		t.Fatalf("expected proto2 optional field to stay optional, got:\n%s", content)
	}
	if !strings.Contains(content, "reqId: number;") {
		t.Fatalf("expected proto2 required field to stay required, got:\n%s", content)
	}
	if !strings.Contains(content, "choiceText?: string;") {
		t.Fatalf("expected proto2 oneof string field to stay optional, got:\n%s", content)
	}
	if !strings.Contains(content, "choiceNum?: number;") {
		t.Fatalf("expected proto2 oneof int32 field to stay optional, got:\n%s", content)
	}
	if !strings.Contains(content, "tags: string[];") {
		t.Fatalf("expected repeated field to stay non-optional array, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2MessageDefaultToleranceShape(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_message_default_tolerance.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Inner"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
			{
				Name: strPtr("Outer"),
				Field: []*descriptorpb.FieldDescriptorProto{
					// Message fields should not have defaults in valid proto2 schemas.
					// Keep generator robust if descriptor payload contains unexpected default values.
					{Name: strPtr("inner"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.proto2.Inner"), DefaultValue: strPtr("{id:'x'}")},
					{Name: strPtr("req_inner"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.proto2.Inner"), DefaultValue: strPtr("{id:'y'}")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_message_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "inner?: Inner;") {
		t.Fatalf("expected proto2 optional message field with unexpected default to stay optional message type, got:\n%s", content)
	}
	if !strings.Contains(content, "reqInner: Inner;") {
		t.Fatalf("expected proto2 required message field with unexpected default to stay required message type, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2EnumInvalidDefaultToleranceShape(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_enum_invalid_default_tolerance.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Mode"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("MODE_UNSPECIFIED"), Number: int32Ptr(0)},
					{Name: strPtr("MODE_FAST"), Number: int32Ptr(1)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("EnumDefaultTolerance"),
				Field: []*descriptorpb.FieldDescriptorProto{
					// Invalid enum default token should not affect output type shape.
					{Name: strPtr("mode"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.proto2.Mode"), DefaultValue: strPtr("MODE_GHOST")},
					{Name: strPtr("req_mode"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.proto2.Mode"), DefaultValue: strPtr("MODE_MISSING")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_enum_invalid_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "mode?: Mode;") {
		t.Fatalf("expected optional enum field with invalid default to stay optional Mode, got:\n%s", content)
	}
	if !strings.Contains(content, "reqMode: Mode;") {
		t.Fatalf("expected required enum field with invalid default to stay required Mode, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2NumericInvalidDefaultToleranceShape(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_numeric_invalid_default_tolerance.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("NumericDefaultTolerance"),
				Field: []*descriptorpb.FieldDescriptorProto{
					// Invalid numeric defaults should not affect output type shape.
					{Name: strPtr("opt_i32"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("2147483648")},
					{Name: strPtr("req_u32"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(), DefaultValue: strPtr("-1")},
					{Name: strPtr("opt_token"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("not_a_number")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_numeric_invalid_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "optI32?: number;") {
		t.Fatalf("expected optional int32 field with invalid default to stay optional number, got:\n%s", content)
	}
	if !strings.Contains(content, "reqU32: number;") {
		t.Fatalf("expected required uint32 field with invalid default to stay required number, got:\n%s", content)
	}
	if !strings.Contains(content, "optToken?: number;") {
		t.Fatalf("expected optional int32 field with invalid token default to stay optional number, got:\n%s", content)
	}
}

func TestGeneratorGenerate_CrossFileServiceProto2DefaultsShape(t *testing.T) {
	g := NewGenerator()
	dep := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("service_defaults_dep.proto"),
		Package: strPtr("sample.dep"),
		Syntax:  strPtr("proto2"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Mode"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("MODE_UNSPECIFIED"), Number: int32Ptr(0)},
					{Name: strPtr("MODE_FAST"), Number: int32Ptr(1)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("DepRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
			{
				Name: strPtr("DepReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("ok"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()},
				},
			},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("service_defaults_main.proto"),
		Package:    strPtr("sample.main"),
		Syntax:     strPtr("proto2"),
		Dependency: []string{"service_defaults_dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Query"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("mode"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.dep.Mode"), DefaultValue: strPtr("MODE_FAST")},
					{Name: strPtr("req_id"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("7")},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("GatewayService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: strPtr("Fetch"), InputType: strPtr(".sample.dep.DepRequest"), OutputType: strPtr(".sample.dep.DepReply")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"service_defaults_main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "import type { DepReply, DepRequest, Mode } from './service_defaults_dep_pb';") {
		t.Fatalf("expected cross-file service/default imports, got:\n%s", content)
	}
	if !strings.Contains(content, "mode?: Mode;") {
		t.Fatalf("expected proto2 optional enum default field to stay optional Mode, got:\n%s", content)
	}
	if !strings.Contains(content, "reqId: number;") {
		t.Fatalf("expected proto2 required scalar default field to stay required number, got:\n%s", content)
	}
	if !strings.Contains(content, "GatewayService") {
		t.Fatalf("expected service export to be present, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2FloatInvalidDefaultToleranceShape(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_float_invalid_default_tolerance.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("FloatDefaultTolerance"),
				Field: []*descriptorpb.FieldDescriptorProto{
					// Invalid/edge floating defaults should not affect output type shape.
					{Name: strPtr("opt_f32"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum(), DefaultValue: strPtr("NaN")},
					{Name: strPtr("req_f64"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), DefaultValue: strPtr("Inf")},
					{Name: strPtr("opt_huge"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), DefaultValue: strPtr("1e9999")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_float_invalid_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "optF32?: number;") {
		t.Fatalf("expected optional float field with invalid default to stay optional number, got:\n%s", content)
	}
	if !strings.Contains(content, "reqF64: number;") {
		t.Fatalf("expected required double field with invalid default to stay required number, got:\n%s", content)
	}
	if !strings.Contains(content, "optHuge?: number;") {
		t.Fatalf("expected optional double field with huge scientific default to stay optional number, got:\n%s", content)
	}
}

func TestGeneratorGenerate_CrossFileServiceOneofProto2DefaultsShape(t *testing.T) {
	g := NewGenerator()
	dep := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("service_oneof_defaults_dep.proto"),
		Package: strPtr("sample.dep"),
		Syntax:  strPtr("proto2"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Mode"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("MODE_UNSPECIFIED"), Number: int32Ptr(0)},
					{Name: strPtr("MODE_FAST"), Number: int32Ptr(1)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("DepRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
			{
				Name: strPtr("DepReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("ok"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()},
				},
			},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("service_oneof_defaults_main.proto"),
		Package:    strPtr("sample.main"),
		Syntax:     strPtr("proto2"),
		Dependency: []string{"service_oneof_defaults_dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:      strPtr("Query"),
				OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: strPtr("choice")}},
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("mode"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.dep.Mode"), DefaultValue: strPtr("MODE_FAST")},
					{Name: strPtr("req_id"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("7")},
					{Name: strPtr("choice_text"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("choice_num"), Number: int32Ptr(4), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), OneofIndex: int32Ptr(0)},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("GatewayService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: strPtr("Fetch"), InputType: strPtr(".sample.dep.DepRequest"), OutputType: strPtr(".sample.dep.DepReply")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"service_oneof_defaults_main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "import type { DepReply, DepRequest, Mode } from './service_oneof_defaults_dep_pb';") {
		t.Fatalf("expected cross-file service/default imports, got:\n%s", content)
	}
	if !strings.Contains(content, "mode?: Mode;") {
		t.Fatalf("expected proto2 optional enum default field to stay optional Mode, got:\n%s", content)
	}
	if !strings.Contains(content, "reqId: number;") {
		t.Fatalf("expected proto2 required scalar default field to stay required number, got:\n%s", content)
	}
	if !strings.Contains(content, "choiceText?: string;") || !strings.Contains(content, "choiceNum?: number;") {
		t.Fatalf("expected oneof fields to stay optional in combined scenario, got:\n%s", content)
	}
	if !strings.Contains(content, "GatewayService") {
		t.Fatalf("expected service export to be present, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2NumericHexOctInvalidDefaultToleranceShape(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_numeric_hex_oct_invalid_default_tolerance.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("NumericHexOctTolerance"),
				Field: []*descriptorpb.FieldDescriptorProto{
					// Non-decimal token styles should not affect output type shape.
					{Name: strPtr("opt_hex"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("0xFF")},
					{Name: strPtr("req_oct"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(), DefaultValue: strPtr("077")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_numeric_hex_oct_invalid_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "optHex?: number;") {
		t.Fatalf("expected optional int32 field with hex default token to stay optional number, got:\n%s", content)
	}
	if !strings.Contains(content, "reqOct: number;") {
		t.Fatalf("expected required uint32 field with octal default token to stay required number, got:\n%s", content)
	}
}

func TestGeneratorGenerate_CrossFileMapOneofProto2DefaultsShape(t *testing.T) {
	g := NewGenerator()
	dep := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("map_oneof_defaults_dep.proto"),
		Package: strPtr("sample.dep"),
		Syntax:  strPtr("proto2"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Mode"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("MODE_UNSPECIFIED"), Number: int32Ptr(0)},
					{Name: strPtr("MODE_FAST"), Number: int32Ptr(1)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("DepRequest"), Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}},
			{Name: strPtr("DepReply"), Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("ok"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()}}},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("map_oneof_defaults_main.proto"),
		Package:    strPtr("sample.main"),
		Syntax:     strPtr("proto2"),
		Dependency: []string{"map_oneof_defaults_dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Query"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("ModeByKeyEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{Name: strPtr("key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
							{Name: strPtr("value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.dep.Mode")},
						},
						Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
					},
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: strPtr("choice")}},
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("mode"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.dep.Mode"), DefaultValue: strPtr("MODE_FAST")},
					{Name: strPtr("req_id"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("7")},
					{Name: strPtr("choice_text"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("choice_num"), Number: int32Ptr(4), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("mode_by_key"), Number: int32Ptr(5), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.main.Query.ModeByKeyEntry")},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: strPtr("GatewayService"), Method: []*descriptorpb.MethodDescriptorProto{{Name: strPtr("Fetch"), InputType: strPtr(".sample.dep.DepRequest"), OutputType: strPtr(".sample.dep.DepReply")}}}},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"map_oneof_defaults_main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "import type { DepReply, DepRequest, Mode } from './map_oneof_defaults_dep_pb';") {
		t.Fatalf("expected cross-file imports for map/oneof/defaults scenario, got:\n%s", content)
	}
	if !strings.Contains(content, "mode?: Mode;") || !strings.Contains(content, "reqId: number;") {
		t.Fatalf("expected proto2 defaults field signatures to stay stable, got:\n%s", content)
	}
	if !strings.Contains(content, "choiceText?: string;") || !strings.Contains(content, "choiceNum?: number;") {
		t.Fatalf("expected oneof fields to stay optional, got:\n%s", content)
	}
	if !strings.Contains(content, "modeByKey: Record<string, Mode>;") {
		t.Fatalf("expected map enum field to stay Record<string, Mode>, got:\n%s", content)
	}
	if !strings.Contains(content, "GatewayService") {
		t.Fatalf("expected service export to be present, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2FloatSignedTokenDefaultToleranceShape(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_float_signed_token_default_tolerance.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("FloatSignedTokenTolerance"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("opt_neg_zero"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum(), DefaultValue: strPtr("-0")},
					{Name: strPtr("req_pos_sci"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), DefaultValue: strPtr("+1e-3")},
					{Name: strPtr("opt_neg_sci"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), DefaultValue: strPtr("-2E+5")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_float_signed_token_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "optNegZero?: number;") {
		t.Fatalf("expected optional float field with -0 default token to stay optional number, got:\n%s", content)
	}
	if !strings.Contains(content, "reqPosSci: number;") {
		t.Fatalf("expected required double field with +scientific default token to stay required number, got:\n%s", content)
	}
	if !strings.Contains(content, "optNegSci?: number;") {
		t.Fatalf("expected optional double field with -scientific default token to stay optional number, got:\n%s", content)
	}
}

func TestGeneratorGenerate_Proto2FloatExtremeSignedTokenDefaultToleranceShape(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_float_extreme_signed_token_default_tolerance.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("FloatExtremeSignedTokenTolerance"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("opt_neg_zero_decimal"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum(), DefaultValue: strPtr("-0.0")},
					{Name: strPtr("req_pos_zero_sci"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), DefaultValue: strPtr("+0e0")},
					{Name: strPtr("opt_neg_underflow"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), DefaultValue: strPtr("-1e-9999")},
					{Name: strPtr("opt_pos_dot_zero"), Number: int32Ptr(4), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum(), DefaultValue: strPtr("+.0")},
					{Name: strPtr("opt_neg_dot_zero"), Number: int32Ptr(5), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), DefaultValue: strPtr("-.0")},
					{Name: strPtr("req_pos_sci_dot"), Number: int32Ptr(6), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), DefaultValue: strPtr("+1.E-3")},
				},
			},
		},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_float_extreme_signed_token_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "optNegZeroDecimal?: number;") {
		t.Fatalf("expected optional float field with -0.0 default token to stay optional number, got:\n%s", content)
	}
	if !strings.Contains(content, "reqPosZeroSci: number;") {
		t.Fatalf("expected required double field with +0e0 default token to stay required number, got:\n%s", content)
	}
	if !strings.Contains(content, "optNegUnderflow?: number;") {
		t.Fatalf("expected optional double field with underflow scientific default token to stay optional number, got:\n%s", content)
	}
	if !strings.Contains(content, "optPosDotZero?: number;") {
		t.Fatalf("expected optional float field with +.0 default token to stay optional number, got:\n%s", content)
	}
	if !strings.Contains(content, "optNegDotZero?: number;") {
		t.Fatalf("expected optional double field with -.0 default token to stay optional number, got:\n%s", content)
	}
	if !strings.Contains(content, "reqPosSciDot: number;") {
		t.Fatalf("expected required double field with +1.E-3 default token to stay required number, got:\n%s", content)
	}
}

func TestGeneratorGenerate_CrossFileServiceReturnMapEnumTopology(t *testing.T) {
	g := NewGenerator()
	dep := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("service_return_map_enum_dep.proto"),
		Package: strPtr("sample.dep"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Status"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("STATUS_UNSPECIFIED"), Number: int32Ptr(0)},
					{Name: strPtr("STATUS_ACTIVE"), Number: int32Ptr(1)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("DepRequest"), Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("service_return_map_enum_main.proto"),
		Package:    strPtr("sample.main"),
		Dependency: []string{"service_return_map_enum_dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Reply"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("StatusByKeyEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{Name: strPtr("key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
							{Name: strPtr("value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.dep.Status")},
						},
						Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
					},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("status_by_key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.main.Reply.StatusByKeyEntry")},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: strPtr("GatewayService"), Method: []*descriptorpb.MethodDescriptorProto{{Name: strPtr("Fetch"), InputType: strPtr(".sample.dep.DepRequest"), OutputType: strPtr(".sample.main.Reply")}}}},
	}

	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"service_return_map_enum_main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := resp.GetFile()[0].GetContent()
	if !strings.Contains(content, "import type { DepRequest, Status } from './service_return_map_enum_dep_pb';") {
		t.Fatalf("expected cross-file import topology for service return map(enum), got:\n%s", content)
	}
	if !strings.Contains(content, "statusByKey: Record<string, Status>;") {
		t.Fatalf("expected map(enum) return message field to stay Record<string, Status>, got:\n%s", content)
	}
	if !strings.Contains(content, "GatewayService") {
		t.Fatalf("expected service export to be present, got:\n%s", content)
	}
}

func TestGeneratorGenerate_CrossFileEnumServiceAndWKT(t *testing.T) {
	g := NewGenerator()
	resp, err := g.Generate(featureRequest())
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(resp.GetFile()) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(resp.GetFile()))
	}
	content := resp.GetFile()[0].GetContent()

	mustContain := []string{
		"import { file_common } from './common_pb';",
		"import { file_google_protobuf_any, file_google_protobuf_duration, file_google_protobuf_empty, file_google_protobuf_field_mask, file_google_protobuf_struct, file_google_protobuf_timestamp, file_google_protobuf_wrappers } from '@bufbuild/protobuf/wkt';",
		"fileDesc('",
		"[file_common, file_google_protobuf_any, file_google_protobuf_duration, file_google_protobuf_empty, file_google_protobuf_field_mask, file_google_protobuf_struct, file_google_protobuf_timestamp, file_google_protobuf_wrappers]",
		"import type { Other } from './common_pb';",
		"import type { Any, BoolValue, BytesValue, DoubleValue, Duration, Empty, FieldMask, FloatValue, Int32Value, Int64Value, ListValue, NullValue, StringValue, Struct, Timestamp, UInt32Value, UInt64Value, Value } from '@bufbuild/protobuf/wkt';",
		"export enum Status",
		"export enum Demo_NestedStatus",
		"export type Demo_Payload",
		"export const StatusSchema = enumDesc(file_main, 0);",
		"export const DemoSchema: GenMessage<Demo> = messageDesc(file_main, 0);",
		"export const Demo_NestedStatusSchema = enumDesc(file_main, 0, 0);",
		"export const Demo_PayloadSchema: GenMessage<Demo_Payload> = messageDesc(file_main, 0, 0);",
		"export const DemoService = serviceDesc(file_main, 0);",
		"other: Other;",
		"payload: Struct;",
		"nested?: Demo_Payload;",
		"nestedStatus: Demo_NestedStatus;",
		"choiceText?: string;",
		"choiceNum?: number;",
		"empty: Empty;",
		"value: Value;",
		"list: ListValue;",
		"nullKind: NullValue;",
		"details: Any;",
		"createdAt: Timestamp;",
		"timeout: Duration;",
		"fieldMask: FieldMask;",
		"doubleValue: DoubleValue;",
		"floatValue: FloatValue;",
		"int64Value: Int64Value;",
		"uint64Value: UInt64Value;",
		"int32Value: Int32Value;",
		"uint32Value: UInt32Value;",
		"boolValue: BoolValue;",
		"stringValue: StringValue;",
		"bytesValue: BytesValue;",
	}
	for _, s := range mustContain {
		if !strings.Contains(content, s) {
			t.Fatalf("generated content missing %q\n----\n%s", s, content)
		}
	}
}

func TestGeneratorGenerate_StructProto_DoesNotSelfImportWKTTypes(t *testing.T) {
	g := NewGenerator()
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"google/protobuf/struct.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    strPtr("google/protobuf/struct.proto"),
			Package: strPtr("google.protobuf"),
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name:  strPtr("NullValue"),
				Value: []*descriptorpb.EnumValueDescriptorProto{{Name: strPtr("NULL_VALUE"), Number: int32Ptr(0)}},
			}},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name:  strPtr("Struct"),
				Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("fields"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Value")}},
			}, {
				Name:  strPtr("Value"),
				Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("null_value"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".google.protobuf.NullValue")}, {Name: strPtr("struct_value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Struct")}, {Name: strPtr("list_value"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.ListValue")}},
			}, {
				Name:  strPtr("ListValue"),
				Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("values"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Value")}},
			}},
		}},
	}

	resp, err := g.Generate(req)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(resp.GetFile()) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(resp.GetFile()))
	}
	content := resp.GetFile()[0].GetContent()
	if strings.Contains(content, "import type { ListValue, NullValue, Struct, Value } from '@bufbuild/protobuf/wkt';") {
		t.Fatalf("struct.proto must not self-import WKT peer symbols from @bufbuild/protobuf/wkt:\n%s", content)
	}
	for _, s := range []string{"export enum NullValue", "export type Struct = Message<'google.protobuf.Struct'>", "fields: Value[];", "nullValue: NullValue;", "structValue: Struct;", "listValue: ListValue;", "export type ListValue = Message<'google.protobuf.ListValue'>"} {
		if !strings.Contains(content, s) {
			t.Fatalf("generated content missing %q\n----\n%s", s, content)
		}
	}
}

func TestGeneratorGenerate_DeepNestedSchemaPath(t *testing.T) {
	g := NewGenerator()
	req := &pluginpb.CodeGeneratorRequest{FileToGenerate: []string{"nested.proto"}, Parameter: strPtr("target=ts"), ProtoFile: []*descriptorpb.FileDescriptorProto{{Name: strPtr("nested.proto"), Package: strPtr("sample"), MessageType: []*descriptorpb.DescriptorProto{{Name: strPtr("Outer"), NestedType: []*descriptorpb.DescriptorProto{{Name: strPtr("Inner"), NestedType: []*descriptorpb.DescriptorProto{{Name: strPtr("Leaf")}}, EnumType: []*descriptorpb.EnumDescriptorProto{{Name: strPtr("InnerStatus"), Value: []*descriptorpb.EnumValueDescriptorProto{{Name: strPtr("INNER_STATUS_UNSPECIFIED"), Number: int32Ptr(0)}}}}}}, EnumType: []*descriptorpb.EnumDescriptorProto{{Name: strPtr("OuterStatus"), Value: []*descriptorpb.EnumValueDescriptorProto{{Name: strPtr("OUTER_STATUS_UNSPECIFIED"), Number: int32Ptr(0)}}}}}}}}}
	resp, err := g.Generate(req)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(resp.GetFile()) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(resp.GetFile()))
	}
	content := resp.GetFile()[0].GetContent()
	for _, s := range []string{"export const OuterSchema: GenMessage<Outer> = messageDesc(file_nested, 0);", "export const Outer_OuterStatusSchema = enumDesc(file_nested, 0, 0);", "export const Outer_InnerSchema: GenMessage<Outer_Inner> = messageDesc(file_nested, 0, 0);", "export const Outer_Inner_InnerStatusSchema = enumDesc(file_nested, 0, 0, 0);", "export const Outer_Inner_LeafSchema: GenMessage<Outer_Inner_Leaf> = messageDesc(file_nested, 0, 0, 0);"} {
		if !strings.Contains(content, s) {
			t.Fatalf("generated content missing %q\n----\n%s", s, content)
		}
	}
}

func TestGeneratorGenerate_AuthLikeNestedEnumAndMap(t *testing.T) {
	g := NewGenerator()
	resp, err := g.Generate(authLikeRequest())
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(resp.GetFile()) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(resp.GetFile()))
	}
	content := resp.GetFile()[0].GetContent()
	for _, s := range []string{"import { file_auth_common } from './common_pb';", "fileDesc('", "[file_auth_common]", "import type { RequestMeta } from './common_pb';", "export type LoginRequest = Message<'auth.v1.LoginRequest'> & {", "mfaKind: LoginRequest_MfaKind;", "metaLabels: Record<string, string>;", "otpCode?: string;", "totpToken?: string;", "rememberMe?: boolean;", "export enum LoginRequest_MfaKind", "export const LoginRequest_MfaKindSchema = enumDesc(file_auth_session, 0, 0);", "export const SessionService = serviceDesc(file_auth_session, 0);"} {
		if !strings.Contains(content, s) {
			t.Fatalf("generated content missing %q\n----\n%s", s, content)
		}
	}
}

func TestGeneratorGenerate_CrossFileEnumAndMapMessageValue(t *testing.T) {
	g := NewGenerator()
	resp, err := g.Generate(crossFileEnumMapRequest())
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(resp.GetFile()) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(resp.GetFile()))
	}
	content := resp.GetFile()[0].GetContent()
	for _, s := range []string{"import { file_dep } from './dep_pb';", "[file_dep]", "import type { Item, Status } from './dep_pb';", "export type Catalog = Message<'sample.main.Catalog'> & {", "status: Status;", "itemByKey: Record<string, Item>;", "export const CatalogSchema: GenMessage<Catalog> = messageDesc(file_main, 0);"} {
		if !strings.Contains(content, s) {
			t.Fatalf("generated content missing %q\n----\n%s", s, content)
		}
	}
}

func TestGeneratorGenerate_CrossFileMapEnumValue(t *testing.T) {
	g := NewGenerator()
	resp, err := g.Generate(crossFileMapEnumRequest())
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(resp.GetFile()) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(resp.GetFile()))
	}
	content := resp.GetFile()[0].GetContent()
	for _, s := range []string{"import { file_dep } from './dep_pb';", "[file_dep]", "import type { Status } from './dep_pb';", "export type Catalog = Message<'sample.main.Catalog'> & {", "statusByKey: Record<string, Status>;", "export const CatalogSchema: GenMessage<Catalog> = messageDesc(file_main_enum_map, 0);"} {
		if !strings.Contains(content, s) {
			t.Fatalf("generated content missing %q\n----\n%s", s, content)
		}
	}
}

func TestGeneratorGenerate_CrossFileServiceExternalMessagesImport(t *testing.T) {
	g := NewGenerator()
	resp, err := g.Generate(crossFileServiceExternalMessagesRequest())
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(resp.GetFile()) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(resp.GetFile()))
	}
	content := resp.GetFile()[0].GetContent()
	for _, s := range []string{"import { file_external } from './external_pb';", "[file_external]", "import type { ExternalReply, ExternalRequest } from './external_pb';", "export const GatewayService = serviceDesc(file_gateway, 0);"} {
		if !strings.Contains(content, s) {
			t.Fatalf("generated content missing %q\n----\n%s", s, content)
		}
	}
}

func TestGeneratorGenerate_MapKeyTypesForRecord(t *testing.T) {
	g := NewGenerator()
	fd := &descriptorpb.FileDescriptorProto{Name: strPtr("map_keys.proto"), Package: strPtr("sample.mapkeys"), MessageType: []*descriptorpb.DescriptorProto{{Name: strPtr("KeyCases"), NestedType: []*descriptorpb.DescriptorProto{{Name: strPtr("BoolByKeyEntry"), Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()}, {Name: strPtr("value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}, Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)}}, {Name: strPtr("I64ByKeyEntry"), Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum()}, {Name: strPtr("value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}, Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)}}, {Name: strPtr("I32ByKeyEntry"), Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()}, {Name: strPtr("value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}, Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)}}}, Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("bool_by_key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.mapkeys.KeyCases.BoolByKeyEntry")}, {Name: strPtr("i64_by_key"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.mapkeys.KeyCases.I64ByKeyEntry")}, {Name: strPtr("i32_by_key"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.mapkeys.KeyCases.I32ByKeyEntry")}}}}}
	resp, err := g.Generate(&pluginpb.CodeGeneratorRequest{FileToGenerate: []string{"map_keys.proto"}, Parameter: strPtr("target=ts"), ProtoFile: []*descriptorpb.FileDescriptorProto{fd}})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	content := resp.GetFile()[0].GetContent()
	for _, s := range []string{"boolByKey: Record<string, string>;", "i64ByKey: Record<string, string>;", "i32ByKey: Record<number, string>;"} {
		if !strings.Contains(content, s) {
			t.Fatalf("generated content missing %q\n----\n%s", s, content)
		}
	}
}

func TestGeneratorGenerate_MinimalSemanticGolden(t *testing.T) {
	req := minimalRequest()
	g := NewGenerator()
	resp, err := g.Generate(req)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	got := SummarizeResponse(resp)
	goldenPath := filepath.Join("testdata", "golden", "minimal_summary.json")
	goldenBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var want FileSummary
	if err := json.Unmarshal(goldenBytes, &want); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}
	diffs := DiffFileSummaries([]FileSummary{want}, got)
	if len(diffs) != 0 {
		t.Fatalf("semantic golden mismatch:\n%s", strings.Join(diffs, "\n"))
	}
}

func TestWKTFixtureProto_IncludesExtendedTypes(t *testing.T) {
	fixture := filepath.Join("testdata", "proto", "wkt_extended.proto")
	content, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixture, err)
	}
	text := string(content)
	for _, item := range []string{"import \"google/protobuf/any.proto\";", "import \"google/protobuf/timestamp.proto\";", "import \"google/protobuf/duration.proto\";", "import \"google/protobuf/field_mask.proto\";", "import \"google/protobuf/wrappers.proto\";", "google.protobuf.Any details = 6;", "google.protobuf.Timestamp created_at = 7;", "google.protobuf.Duration timeout = 8;", "google.protobuf.FieldMask field_mask = 9;", "google.protobuf.DoubleValue double_value = 10;", "google.protobuf.FloatValue float_value = 11;", "google.protobuf.Int64Value int64_value = 12;", "google.protobuf.UInt64Value uint64_value = 13;", "google.protobuf.Int32Value int32_value = 14;", "google.protobuf.UInt32Value uint32_value = 15;", "google.protobuf.BoolValue bool_value = 16;", "google.protobuf.StringValue string_value = 17;", "google.protobuf.BytesValue bytes_value = 18;"} {
		if !strings.Contains(text, item) {
			t.Fatalf("fixture missing %q", item)
		}
	}
}

func featureRequest() *pluginpb.CodeGeneratorRequest {
	common := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("common.proto"),
		Package: strPtr("sample"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Other"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   strPtr("id"),
						Number: int32Ptr(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("main.proto"),
		Package:    strPtr("sample"),
		Dependency: []string{"common.proto", "google/protobuf/any.proto", "google/protobuf/duration.proto", "google/protobuf/empty.proto", "google/protobuf/field_mask.proto", "google/protobuf/struct.proto", "google/protobuf/timestamp.proto", "google/protobuf/wrappers.proto"},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Status"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("STATUS_UNSPECIFIED"), Number: int32Ptr(0)},
					{Name: strPtr("STATUS_ACTIVE"), Number: int32Ptr(1)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Demo"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("Payload"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{Name: strPtr("note"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
						},
					},
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: strPtr("NestedStatus"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{Name: strPtr("NESTED_UNSPECIFIED"), Number: int32Ptr(0)},
							{Name: strPtr("NESTED_READY"), Number: int32Ptr(1)},
						},
					},
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: strPtr("choice")}},
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("other"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.Other")},
					{Name: strPtr("status"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.Status")},
					{Name: strPtr("payload"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Struct")},
					{Name: strPtr("empty"), Number: int32Ptr(4), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Empty")},
					{Name: strPtr("value"), Number: int32Ptr(5), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Value")},
					{Name: strPtr("list"), Number: int32Ptr(6), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.ListValue")},
					{Name: strPtr("null_kind"), Number: int32Ptr(7), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".google.protobuf.NullValue")},
					{Name: strPtr("nested"), Number: int32Ptr(8), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.Demo.Payload"), Proto3Optional: boolPtr(true)},
					{Name: strPtr("nested_status"), Number: int32Ptr(9), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.Demo.NestedStatus")},
					{Name: strPtr("choice_text"), Number: int32Ptr(10), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("choice_num"), Number: int32Ptr(11), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("details"), Number: int32Ptr(12), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Any")},
					{Name: strPtr("created_at"), Number: int32Ptr(13), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Timestamp")},
					{Name: strPtr("timeout"), Number: int32Ptr(14), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Duration")},
					{Name: strPtr("field_mask"), Number: int32Ptr(15), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.FieldMask")},
					{Name: strPtr("double_value"), Number: int32Ptr(16), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.DoubleValue")},
					{Name: strPtr("float_value"), Number: int32Ptr(17), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.FloatValue")},
					{Name: strPtr("int64_value"), Number: int32Ptr(18), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Int64Value")},
					{Name: strPtr("uint64_value"), Number: int32Ptr(19), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.UInt64Value")},
					{Name: strPtr("int32_value"), Number: int32Ptr(20), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.Int32Value")},
					{Name: strPtr("uint32_value"), Number: int32Ptr(21), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.UInt32Value")},
					{Name: strPtr("bool_value"), Number: int32Ptr(22), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.BoolValue")},
					{Name: strPtr("string_value"), Number: int32Ptr(23), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.StringValue")},
					{Name: strPtr("bytes_value"), Number: int32Ptr(24), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".google.protobuf.BytesValue")},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: strPtr("DemoService")}},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{common, main},
	}
}

func authLikeRequest() *pluginpb.CodeGeneratorRequest {
	common := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("auth/common.proto"),
		Package: strPtr("auth.v1"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("RequestMeta"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   strPtr("trace_id"),
					Number: int32Ptr(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}},
			},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("auth/session.proto"),
		Package:    strPtr("auth.v1"),
		Dependency: []string{"auth/common.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("LoginRequest"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("MetaLabelsEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{Name: strPtr("key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
							{Name: strPtr("value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
						},
						Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
					},
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{{
					Name: strPtr("MfaKind"),
					Value: []*descriptorpb.EnumValueDescriptorProto{
						{Name: strPtr("MFA_KIND_UNSPECIFIED"), Number: int32Ptr(0)},
						{Name: strPtr("MFA_KIND_TOTP"), Number: int32Ptr(1)},
					},
				}},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: strPtr("auth_factor")}, {Name: strPtr("_remember_me")}},
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("username"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("password"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("meta"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".auth.v1.RequestMeta")},
					{Name: strPtr("otp_code"), Number: int32Ptr(4), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("totp_token"), Number: int32Ptr(5), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("remember_me"), Number: int32Ptr(6), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(), Proto3Optional: boolPtr(true), OneofIndex: int32Ptr(1)},
					{Name: strPtr("mfa_kind"), Number: int32Ptr(7), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".auth.v1.LoginRequest.MfaKind")},
					{Name: strPtr("meta_labels"), Number: int32Ptr(8), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".auth.v1.LoginRequest.MetaLabelsEntry")},
				},
			},
			{
				Name:  strPtr("LoginReply"),
				Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("access_token"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name:   strPtr("SessionService"),
			Method: []*descriptorpb.MethodDescriptorProto{{Name: strPtr("Login"), InputType: strPtr(".auth.v1.LoginRequest"), OutputType: strPtr(".auth.v1.LoginReply")}},
		}},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"auth/session.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{common, main},
	}
}

func crossFileEnumMapRequest() *pluginpb.CodeGeneratorRequest {
	dep := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("dep.proto"),
		Package: strPtr("sample.dep"),
		EnumType: []*descriptorpb.EnumDescriptorProto{{
			Name: strPtr("Status"),
			Value: []*descriptorpb.EnumValueDescriptorProto{
				{Name: strPtr("STATUS_UNSPECIFIED"), Number: int32Ptr(0)},
				{Name: strPtr("STATUS_ACTIVE"), Number: int32Ptr(1)},
			},
		}},
		MessageType: []*descriptorpb.DescriptorProto{{
			Name:  strPtr("Item"),
			Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}},
		}},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("main.proto"),
		Package:    strPtr("sample.main"),
		Dependency: []string{"dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: strPtr("Catalog"),
			NestedType: []*descriptorpb.DescriptorProto{{
				Name: strPtr("ItemByKeyEntry"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.dep.Item")},
				},
				Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
			}},
			Field: []*descriptorpb.FieldDescriptorProto{
				{Name: strPtr("status"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.dep.Status")},
				{Name: strPtr("item_by_key"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.main.Catalog.ItemByKeyEntry")},
			},
		}},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	}
}

func crossFileMapEnumRequest() *pluginpb.CodeGeneratorRequest {
	dep := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("dep.proto"),
		Package: strPtr("sample.dep"),
		EnumType: []*descriptorpb.EnumDescriptorProto{{
			Name: strPtr("Status"),
			Value: []*descriptorpb.EnumValueDescriptorProto{
				{Name: strPtr("STATUS_UNSPECIFIED"), Number: int32Ptr(0)},
				{Name: strPtr("STATUS_ACTIVE"), Number: int32Ptr(1)},
			},
		}},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("main_enum_map.proto"),
		Package:    strPtr("sample.main"),
		Dependency: []string{"dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: strPtr("Catalog"),
			NestedType: []*descriptorpb.DescriptorProto{{
				Name: strPtr("StatusByKeyEntry"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.dep.Status")},
				},
				Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
			}},
			Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("status_by_key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.main.Catalog.StatusByKeyEntry")}},
		}},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"main_enum_map.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	}
}

func crossFileServiceExternalMessagesRequest() *pluginpb.CodeGeneratorRequest {
	ext := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("external.proto"),
		Package: strPtr("sample.external"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("ExternalRequest"), Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()}}},
			{Name: strPtr("ExternalReply"), Field: []*descriptorpb.FieldDescriptorProto{{Name: strPtr("ok"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()}}},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("gateway.proto"),
		Package:    strPtr("sample.gateway"),
		Dependency: []string{"external.proto"},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name:   strPtr("GatewayService"),
			Method: []*descriptorpb.MethodDescriptorProto{{Name: strPtr("Forward"), InputType: strPtr(".sample.external.ExternalRequest"), OutputType: strPtr(".sample.external.ExternalReply")}},
		}},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"gateway.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{ext, main},
	}
}

func minimalRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("minimal.proto"),
		Package: strPtr("sample"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Demo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   strPtr("id"),
						Number: int32Ptr(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"minimal.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func strPtr(s string) *string {
	return &s
}

func int32Ptr(v int32) *int32 {
	return &v
}

func boolPtr(v bool) *bool { return &v }
