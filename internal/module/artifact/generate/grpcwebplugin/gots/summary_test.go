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

const (
	updateGoldenEnv  = "GOTS_UPDATE_GOLDEN"
	refreshGoldenCmd = "GOTS_UPDATE_GOLDEN=1 go test ./internal/module/artifact/generate/grpcwebplugin/gots -run 'TestSummarizeGeneratedFile_AuthSubsetGolden|TestSummarizeGeneratedFile_ServiceNameVariantsGolden'"
)

func TestSummarizeGeneratedFile_MatchesGolden(t *testing.T) {
	tsPath := filepath.Join("testdata", "generated", "minimal_pb.ts")
	goldenPath := filepath.Join("testdata", "golden", "minimal_summary.json")

	content, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("read ts fixture: %v", err)
	}

	goldenBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var want FileSummary
	if err := json.Unmarshal(goldenBytes, &want); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}

	got := SummarizeGeneratedFile("minimal_pb.ts", string(content))
	diffs := DiffFileSummaries([]FileSummary{want}, []FileSummary{got})
	if len(diffs) != 0 {
		t.Fatalf("summary mismatch:\n%s", strings.Join(diffs, "\n"))
	}
}

func TestSummarizeGeneratedFile_ServiceMatchesGolden(t *testing.T) {
	tsPath := filepath.Join("testdata", "generated", "minimal_service_pb.ts")
	goldenPath := filepath.Join("testdata", "golden", "minimal_service_summary.json")

	content, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("read ts fixture: %v", err)
	}

	goldenBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var want FileSummary
	if err := json.Unmarshal(goldenBytes, &want); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}

	got := SummarizeGeneratedFile("minimal_service_pb.ts", string(content))
	diffs := DiffFileSummaries([]FileSummary{want}, []FileSummary{got})
	if len(diffs) != 0 {
		t.Fatalf("summary mismatch:\n%s", strings.Join(diffs, "\n"))
	}
}

func TestSummarizeGeneratedFile_AuthSubsetGolden(t *testing.T) {
	// Auth subset uses generator-driven snapshots (no handwritten TS fixtures).
	cases := []struct {
		name    string
		req     *pluginpb.CodeGeneratorRequest
		outName string
		golden  string
	}{
		{name: "auth session", req: authSubsetSessionRequest(), outName: "auth/session_pb.ts", golden: filepath.Join("testdata", "golden", "auth_session_summary.json")},
		{name: "auth account", req: authSubsetAccountRequest(), outName: "auth/account_pb.ts", golden: filepath.Join("testdata", "golden", "auth_account_summary.json")},
		{name: "gateway external messages", req: gatewaySubsetRequest(), outName: "gateway_pb.ts", golden: filepath.Join("testdata", "golden", "gateway_summary.json")},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp, err := NewGenerator().Generate(tc.req)
			if err != nil {
				t.Fatalf("generate failed: %v", err)
			}

			content, ok := findGeneratedContent(resp, tc.outName)
			if !ok {
				t.Fatalf("generated response missing file %q", tc.outName)
			}

			got := SummarizeGeneratedFile(tc.outName, content)
			assertSummaryGolden(t, tc.golden, got)
		})
	}
}

func TestSummarizeGeneratedFile_ServiceNameVariantsGolden(t *testing.T) {
	resp, err := NewGenerator().Generate(serviceNameVariantsRequest())
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	content, ok := findGeneratedContent(resp, "service_variants_pb.ts")
	if !ok {
		t.Fatalf("generated response missing file %q", "service_variants_pb.ts")
	}

	got := SummarizeGeneratedFile("service_variants_pb.ts", content)
	assertSummaryGolden(t, filepath.Join("testdata", "golden", "service_variants_summary.json"), got)
}

func assertSummaryGolden(t *testing.T, goldenPath string, got FileSummary) {
	t.Helper()

	if os.Getenv(updateGoldenEnv) == "1" {
		buf, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatalf("marshal golden update: %v", err)
		}
		buf = append(buf, '\n')
		if err := os.WriteFile(goldenPath, buf, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}

	goldenBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var want FileSummary
	if err := json.Unmarshal(goldenBytes, &want); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}

	diffs := DiffFileSummaries([]FileSummary{want}, []FileSummary{got})
	if len(diffs) != 0 {
		t.Fatalf("summary mismatch:\n%s\n\nrefresh: %s", strings.Join(diffs, "\n"), refreshGoldenCmd)
	}
}

func findGeneratedContent(resp *pluginpb.CodeGeneratorResponse, name string) (string, bool) {
	if resp == nil {
		return "", false
	}
	for _, f := range resp.GetFile() {
		if f.GetName() == name {
			return f.GetContent(), true
		}
	}
	return "", false
}

func authSubsetSessionRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("auth/session.proto"),
		Package: strPtr("auth.v1"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: strPtr("LoginRequest"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:   strPtr("username"),
				Number: int32Ptr(1),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			}},
		}},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: strPtr("SessionService"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:       strPtr("Login"),
				InputType:  strPtr(".auth.v1.LoginRequest"),
				OutputType: strPtr(".auth.v1.LoginRequest"),
			}},
		}},
	}
	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"auth/session.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func authSubsetAccountRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("auth/account.proto"),
		Package: strPtr("auth.v1"),
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: strPtr("AccountService"),
		}},
	}
	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"auth/account.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func gatewaySubsetRequest() *pluginpb.CodeGeneratorRequest {
	ext := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("external.proto"),
		Package: strPtr("sample.external"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("ExternalRequest")},
			{Name: strPtr("ExternalReply")},
		},
	}
	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("gateway.proto"),
		Package:    strPtr("sample.gateway"),
		Dependency: []string{"external.proto"},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: strPtr("GatewayService"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:       strPtr("Forward"),
				InputType:  strPtr(".sample.external.ExternalRequest"),
				OutputType: strPtr(".sample.external.ExternalReply"),
			}},
		}},
	}
	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"gateway.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{ext, main},
	}
}

func serviceNameVariantsRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("service_variants.proto"),
		Package: strPtr("sample.service"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("PingRequest")},
			{Name: strPtr("PingReply")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("user_session_service"),
				Method: []*descriptorpb.MethodDescriptorProto{{
					Name:       strPtr("Ping"),
					InputType:  strPtr(".sample.service.PingRequest"),
					OutputType: strPtr(".sample.service.PingReply"),
				}},
			},
			{
				Name: strPtr("AuditTrailService"),
				Method: []*descriptorpb.MethodDescriptorProto{{
					Name:       strPtr("Ping"),
					InputType:  strPtr(".sample.service.PingRequest"),
					OutputType: strPtr(".sample.service.PingReply"),
				}},
			},
		},
	}
	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"service_variants.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func TestDiffFileSummaries_ReportsCoreDifferences(t *testing.T) {
	want := []FileSummary{{
		Name:           "a_pb.ts",
		Imports:        []string{"./dep_pb"},
		Exports:        []string{"A", "ASchema"},
		SchemaExports:  []string{"ASchema"},
		ServiceExports: []string{"Svc"},
	}}
	got := []FileSummary{{
		Name:           "a_pb.ts",
		Imports:        []string{"./x_pb"},
		Exports:        []string{"A"},
		SchemaExports:  nil,
		ServiceExports: nil,
	}}

	diffs := DiffFileSummaries(want, got)
	if len(diffs) == 0 {
		t.Fatalf("expected diffs, got none")
	}
}

func TestSummarizeGeneratedFile_ServiceExports(t *testing.T) {
	content := "export const DemoService = serviceDesc(file_demo, 0);\nexport const user_session_service = serviceDesc(file_demo, 1);\nexport const DemoSchema = messageDesc(file_demo, 0);\n"
	s := SummarizeGeneratedFile("demo_pb.ts", content)
	if len(s.ServiceExports) != 2 || s.ServiceExports[0] != "DemoService" || s.ServiceExports[1] != "user_session_service" {
		t.Fatalf("unexpected service exports: %#v", s.ServiceExports)
	}
}

func TestSummarizeResponse_SortsByFileName(t *testing.T) {
	resp := &pluginpb.CodeGeneratorResponse{File: []*pluginpb.CodeGeneratorResponse_File{
		{Name: strPtr("b_pb.ts"), Content: strPtr("export type B = {};")},
		{Name: strPtr("a_pb.ts"), Content: strPtr("export type A = {};")},
	}}

	got := SummarizeResponse(resp)
	if len(got) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(got))
	}
	if got[0].Name != "a_pb.ts" || got[1].Name != "b_pb.ts" {
		t.Fatalf("expected summaries sorted by file name, got %#v", got)
	}
}

func TestDiffFileSummariesWithOptions_IgnoresWhitelistedDifferences(t *testing.T) {
	want := []FileSummary{{
		Name:          "a_pb.ts",
		Imports:       []string{"./dep_pb", "@bufbuild/protobuf/wkt"},
		Exports:       []string{"A", "ASchema", "Noise"},
		SchemaExports: []string{"ASchema", "NoiseSchema"},
	}, {
		Name:          "ignore_pb.ts",
		Imports:       []string{"./any_pb"},
		Exports:       []string{"Ignored"},
		SchemaExports: []string{"IgnoredSchema"},
	}}
	got := []FileSummary{{
		Name:          "a_pb.ts",
		Imports:       []string{"./x_pb", "@bufbuild/protobuf/wkt"},
		Exports:       []string{"A", "ASchema", "OtherNoise"},
		SchemaExports: []string{"ASchema", "OtherNoiseSchema"},
	}, {
		Name:          "ignore_pb.ts",
		Imports:       []string{"./diff_pb"},
		Exports:       []string{"Different"},
		SchemaExports: []string{"DifferentSchema"},
	}}

	diffs := DiffFileSummariesWithOptions(want, got, DiffOptions{
		IgnoreFiles:         []string{"ignore_pb.ts"},
		IgnoreImports:       []string{"./dep_pb", "./x_pb"},
		IgnoreExports:       []string{"Noise", "OtherNoise"},
		IgnoreSchemaExports: []string{"NoiseSchema", "OtherNoiseSchema"},
	})
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs after whitelist filtering, got:\n%s", strings.Join(diffs, "\n"))
	}
}

func TestDiffFileSummariesWithOptions_FileScopedWhitelist(t *testing.T) {
	want := []FileSummary{{
		Name:           "a_pb.ts",
		Imports:        []string{"./x_pb"},
		Exports:        []string{"A", "OnlyInA"},
		SchemaExports:  []string{"ASchema", "OnlyInASchema"},
		ServiceExports: []string{"OnlyServiceA"},
	}, {
		Name:           "b_pb.ts",
		Imports:        []string{"./x_pb"},
		Exports:        []string{"B", "OnlyInB"},
		SchemaExports:  []string{"BSchema", "OnlyInBSchema"},
		ServiceExports: []string{"OnlyServiceB"},
	}}
	got := []FileSummary{{
		Name:           "a_pb.ts",
		Imports:        []string{"./x_pb"},
		Exports:        []string{"A", "OtherA"},
		SchemaExports:  []string{"ASchema", "OtherASchema"},
		ServiceExports: []string{"OtherServiceA"},
	}, {
		Name:           "b_pb.ts",
		Imports:        []string{"./x_pb"},
		Exports:        []string{"B", "OtherB"},
		SchemaExports:  []string{"BSchema", "OtherBSchema"},
		ServiceExports: []string{"OtherServiceB"},
	}}

	diffs := DiffFileSummariesWithOptions(want, got, DiffOptions{
		IgnoreExportsByFile: map[string][]string{
			"a_pb.ts": {"OnlyInA", "OtherA"},
		},
		IgnoreSchemaExportsByFile: map[string][]string{
			"a_pb.ts": {"OnlyInASchema", "OtherASchema"},
		},
		IgnoreServiceExportsByFile: map[string][]string{
			"a_pb.ts": {"OnlyServiceA", "OtherServiceA"},
		},
	})
	if len(diffs) == 0 {
		t.Fatalf("expected diffs for non-whitelisted files, got none")
	}

	joined := strings.Join(diffs, "\n")
	if strings.Contains(joined, "a_pb.ts") {
		t.Fatalf("a_pb.ts should be fully filtered by file-scoped whitelist, got:\n%s", joined)
	}
	if !strings.Contains(joined, "b_pb.ts") {
		t.Fatalf("b_pb.ts should still report diffs, got:\n%s", joined)
	}
}

func TestDiffFileSummariesWithOptions_ErrorShowsFilteredValues(t *testing.T) {
	want := []FileSummary{{
		Name:    "a_pb.ts",
		Imports: []string{"./dep_pb", "./keep_pb"},
	}}
	got := []FileSummary{{
		Name:    "a_pb.ts",
		Imports: []string{"./x_pb", "./keep_pb"},
	}}

	diffs := DiffFileSummariesWithOptions(want, got, DiffOptions{
		IgnoreImports: []string{"./dep_pb", "./x_pb"},
	})
	if len(diffs) != 0 {
		t.Fatalf("expected no diffs after import filtering, got:\n%s", strings.Join(diffs, "\n"))
	}

	// Keep one side non-ignored to force mismatch and verify reported values are filtered.
	diffs = DiffFileSummariesWithOptions(want, got, DiffOptions{
		IgnoreImports: []string{"./dep_pb"},
	})
	if len(diffs) == 0 {
		t.Fatalf("expected import mismatch")
	}
	joined := strings.Join(diffs, "\n")
	if !strings.Contains(joined, "want=[./keep_pb] got=[./keep_pb ./x_pb]") {
		t.Fatalf("expected mismatch to report filtered values, got:\n%s", joined)
	}
}

func TestSummarizeGeneratedFile_MessageShapes(t *testing.T) {
	content := `export type Demo = Message<'sample.Demo'> & {
  id: string;
  count?: bigint;
  labels: Record<string, string>;
};

export type Other = Message<'sample.Other'> & {
  ok: boolean;
};`

	s := SummarizeGeneratedFile("demo_pb.ts", content)
	if len(s.MessageShapes) != 2 {
		t.Fatalf("expected 2 message shapes, got %#v", s.MessageShapes)
	}
	demo := s.MessageShapes["Demo"]
	if len(demo) != 3 {
		t.Fatalf("expected 3 Demo fields, got %#v", demo)
	}
	if !containsStringLocal(demo, "count?: bigint") {
		t.Fatalf("expected Demo shape to include optional bigint field, got %#v", demo)
	}
}

func TestDiffFileSummaries_MessageShapeMismatch(t *testing.T) {
	want := []FileSummary{{
		Name: "a_pb.ts",
		MessageShapes: map[string][]string{
			"Demo": {"id: string", "count: bigint"},
		},
	}}
	got := []FileSummary{{
		Name: "a_pb.ts",
		MessageShapes: map[string][]string{
			"Demo": {"id: string", "count: number"},
		},
	}}

	diffs := DiffFileSummaries(want, got)
	if len(diffs) == 0 {
		t.Fatalf("expected message shape mismatch, got none")
	}
	joined := strings.Join(diffs, "\n")
	if !strings.Contains(joined, "message shape mismatch in a_pb.ts/Demo") {
		t.Fatalf("expected message shape mismatch details, got:\n%s", joined)
	}
}

func containsStringLocal(items []string, target string) bool {
	for _, it := range items {
		if it == target {
			return true
		}
	}
	return false
}
