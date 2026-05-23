// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcwebplugin

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/artifact/generate/grpcwebplugin/gots"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	parityScopeExport       = "export"
	parityScopeSchemaExport = "schema_export"
	parityScopeMessageShape = "message_shape"
	// Increase only with an attached parity issue that documents why new items are semantically equivalent.
	parityWhitelistItemMax = 16
)

type parityWhitelistEntry struct {
	file    string
	scope   string
	items   []string
	issue   string
	reason  string
	expires string
}

type parityProtocGenerator struct {
	protocPath string
	pluginPath string
}

func (g *parityProtocGenerator) Generate(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}

	workDir, err := os.MkdirTemp("", "choysum-parity-protoc-")
	if err != nil {
		return nil, fmt.Errorf("create parity temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	descSetPath := filepath.Join(workDir, "input.pb")
	outDir := filepath.Join(workDir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create parity output dir: %w", err)
	}

	descSet := &descriptorpb.FileDescriptorSet{File: req.GetProtoFile()}
	descBytes, err := proto.Marshal(descSet)
	if err != nil {
		return nil, fmt.Errorf("marshal descriptor set: %w", err)
	}
	if err := os.WriteFile(descSetPath, descBytes, 0o644); err != nil {
		return nil, fmt.Errorf("write descriptor set: %w", err)
	}

	args := []string{
		"--descriptor_set_in=" + descSetPath,
		"--plugin=protoc-gen-es=" + g.pluginPath,
		"--es_out=" + outDir,
	}
	if p := strings.TrimSpace(req.GetParameter()); p != "" {
		args = append(args, "--es_opt="+p)
	}
	args = append(args, req.GetFileToGenerate()...)

	cmd := exec.Command(g.protocPath, args...)
	cmd.Dir = workDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run protoc parity generator: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	files := make([]*pluginpb.CodeGeneratorResponse_File, 0)
	err = filepath.WalkDir(outDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(outDir, path)
		if relErr != nil {
			return relErr
		}
		name := filepath.ToSlash(rel)
		body := string(content)
		files = append(files, &pluginpb.CodeGeneratorResponse_File{Name: &name, Content: &body})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect protoc parity outputs: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].GetName() < files[j].GetName() })

	return &pluginpb.CodeGeneratorResponse{File: files}, nil
}

func newParityProtocGenerator() (*parityProtocGenerator, error) {
	protocPath, err := exec.LookPath("protoc")
	if err != nil {
		return nil, fmt.Errorf("protoc not found in PATH")
	}
	pluginPath, err := resolveParityProtocGenESPlugin()
	if err != nil {
		return nil, err
	}
	return &parityProtocGenerator{protocPath: protocPath, pluginPath: pluginPath}, nil
}

func resolveParityProtocGenESPlugin() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("CHOYSUM_PARITY_PROTOC_GEN_ES_PLUGIN")); explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("CHOYSUM_PARITY_PROTOC_GEN_ES_PLUGIN invalid: %w", err)
		}
		return explicit, nil
	}

	repoRoot := parityRepoRoot()
	pluginSource := filepath.Join(repoRoot, "node_modules", "@bufbuild", "protoc-gen-es", "dist", "cjs", "src", "protoc-gen-es-plugin.js")
	if _, err := os.Stat(pluginSource); err != nil {
		return "", fmt.Errorf("resolve protoc-gen-es plugin failed: %w", err)
	}
	return writeParityProtocGenESWrapper(repoRoot, pluginSource)
}

func parityRepoRoot() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic(fmt.Sprintf("resolve repo root from %s: go.mod not found", thisFile))
		}
		dir = parent
	}
}

func writeParityProtocGenESWrapper(repoRoot string, pluginSource string) (string, error) {
	wrapperPath := filepath.Join(repoRoot, "node_modules", ".bin", "choysum-protoc-gen-es-parity")
	content := strings.Join([]string{
		"#!/usr/bin/env node",
		"if (typeof globalThis.localStorage !== 'object' || globalThis.localStorage === null || typeof globalThis.localStorage.getItem !== 'function') {",
		"  globalThis.localStorage = { getItem() { return undefined; }, setItem() {}, removeItem() {}, clear() {} };",
		"}",
		"const { runNodeJs } = require('@bufbuild/protoplugin');",
		"const { protocGenEs } = require(" + strconv.Quote(pluginSource) + ");",
		"runNodeJs(protocGenEs);",
		"",
	}, "\n")
	if err := os.WriteFile(wrapperPath, []byte(content), 0o755); err != nil {
		return "", fmt.Errorf("write protoc-gen-es parity wrapper: %w", err)
	}
	return wrapperPath, nil
}

func parityWhitelistEntries() []parityWhitelistEntry {
	// Default target is zero whitelist. Add entries only when parity proves a
	// semantically equivalent mismatch and attach full audit metadata.
	return nil
}

func parityDiffOptions() gots.DiffOptions {
	// Keep this whitelist minimal and temporary.
	// Add only entries that are proven to be semantically equivalent.
	opts := gots.DiffOptions{
		IgnoreExportsByFile:       map[string][]string{},
		IgnoreSchemaExportsByFile: map[string][]string{},
		IgnoreMessageShapesByFile: map[string][]string{},
	}
	for _, e := range parityWhitelistEntries() {
		switch e.scope {
		case parityScopeExport:
			opts.IgnoreExportsByFile[e.file] = append(opts.IgnoreExportsByFile[e.file], e.items...)
		case parityScopeSchemaExport:
			opts.IgnoreSchemaExportsByFile[e.file] = append(opts.IgnoreSchemaExportsByFile[e.file], e.items...)
		case parityScopeMessageShape:
			opts.IgnoreMessageShapesByFile[e.file] = append(opts.IgnoreMessageShapesByFile[e.file], e.items...)
		}
	}
	return opts
}

func TestParityWhitelistEntries_Auditable(t *testing.T) {
	entries := parityWhitelistEntries()
	if len(entries) == 0 {
		return
	}
	seen := map[string]int{}
	totalItems := 0
	for i, e := range entries {
		if e.file == "" {
			t.Fatalf("entry[%d] missing file", i)
		}
		if e.scope == "" {
			t.Fatalf("entry[%d] missing scope", i)
		}
		if e.scope != parityScopeExport && e.scope != parityScopeSchemaExport && e.scope != parityScopeMessageShape {
			t.Fatalf("entry[%d] has invalid scope %q", i, e.scope)
		}
		if len(e.items) == 0 {
			t.Fatalf("entry[%d] has no items", i)
		}
		if strings.TrimSpace(e.issue) == "" {
			t.Fatalf("entry[%d] missing issue", i)
		}
		if strings.TrimSpace(e.reason) == "" {
			t.Fatalf("entry[%d] missing reason", i)
		}
		if strings.TrimSpace(e.expires) == "" {
			t.Fatalf("entry[%d] missing expires", i)
		}
		if !isActionableExpiry(e.expires) {
			t.Fatalf("entry[%d] has non-actionable expires value %q", i, e.expires)
		}
		for _, item := range e.items {
			totalItems++
			if strings.TrimSpace(item) == "" {
				t.Fatalf("entry[%d] contains empty item", i)
			}
			k := e.file + "|" + e.scope + "|" + item
			seen[k]++
			if seen[k] > 1 {
				t.Fatalf("duplicated whitelist item: file=%s scope=%s item=%s", e.file, e.scope, item)
			}
		}
	}
	if totalItems > parityWhitelistItemMax {
		t.Fatalf("parity whitelist item budget exceeded: total=%d max=%d", totalItems, parityWhitelistItemMax)
	}
}

func isActionableExpiry(expires string) bool {
	v := strings.ToLower(strings.TrimSpace(expires))
	if v == "" {
		return false
	}
	keywords := []string{"before", "phase", "milestone", "switch"}
	for _, kw := range keywords {
		if strings.Contains(v, kw) {
			return true
		}
	}
	return false
}

func TestParityDiffOptions_FromEntries(t *testing.T) {
	opts := parityDiffOptions()

	exports := opts.IgnoreExportsByFile["nested_pb.ts"]
	schemaExports := opts.IgnoreSchemaExportsByFile["nested_pb.ts"]
	if len(parityWhitelistEntries()) == 0 {
		if len(exports) != 0 || len(schemaExports) != 0 {
			t.Fatalf("expected empty diff options when no whitelist entries are configured")
		}
		return
	}

	if len(exports) == 0 {
		t.Fatalf("expected nested_pb.ts export whitelist to be populated")
	}
	if len(schemaExports) == 0 {
		t.Fatalf("expected nested_pb.ts schema export whitelist to be populated")
	}

	if !containsString(exports, "Outer_Status") {
		t.Fatalf("expected export whitelist to contain Outer_Status")
	}
	if containsString(schemaExports, "Outer_Status") {
		t.Fatalf("schema export whitelist should not contain non-schema symbol Outer_Status")
	}
	if !containsString(schemaExports, "Outer_StatusSchema") {
		t.Fatalf("expected schema export whitelist to contain Outer_StatusSchema")
	}
}

func containsString(items []string, target string) bool {
	for _, it := range items {
		if it == target {
			return true
		}
	}
	return false
}

func TestIsActionableExpiry(t *testing.T) {
	tests := []struct {
		name    string
		expires string
		want    bool
	}{
		{name: "before keyword", expires: "before default switch to Go generator", want: true},
		{name: "phase keyword", expires: "phase 3 parity cleanup", want: true},
		{name: "milestone keyword", expires: "milestone: go-default", want: true},
		{name: "empty", expires: "", want: false},
		{name: "vague text", expires: "later", want: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := isActionableExpiry(tt.expires); got != tt.want {
				t.Fatalf("isActionableExpiry(%q) = %v, want %v", tt.expires, got, tt.want)
			}
		})
	}
}

func TestParity_JSAndGoTargetTS_Fixtures(t *testing.T) {
	jsGen, err := newParityProtocGenerator()
	if err != nil {
		t.Skipf("skip parity: protoc/protoc-gen-es unavailable: %v", err)
	}

	goGen := gots.NewGenerator()

	// Fixture coverage matrix:
	// - minimal: basic message + schema export baseline
	// - nested: nested message/enum naming and schema path
	// - wkt: minimum WKT import/type support
	// - auth_like: cross-file deps + service + oneof + proto3 optional + map(message) + nested enum
	// - int64_oneof_map: 64-bit scalar semantics + oneof + map(message) + cross-file service IO
	// - nested_oneof_map_enum_i64: nested oneof + map(enum value) + mixed int64/uint64 in nested message
	// - proto2_presence: proto2 optional/required/repeated presence semantics
	// - proto2_oneof_presence: proto2 + oneof combined presence semantics
	// - proto2_defaults: proto2 scalar/enum default values shape compatibility
	// - proto2_bytes_defaults: proto2 bytes default values type/presence compatibility
	// - proto2_cross_file_enum_default: proto2 cross-file enum default import/type compatibility
	// - proto2_float_invalid_default_tolerance: robustness when float/double fields carry NaN/Inf/huge scientific default tokens
	// - proto2_numeric_hex_oct_invalid_default_tolerance: robustness when numeric fields carry hex/octal default token styles
	// - cross_file_service_proto2_defaults: cross-file service IO + proto2 defaults import/type/signature compatibility
	// - cross_file_service_oneof_proto2_defaults: cross-file service IO + oneof + proto2 defaults linked stability
	// - cross_file_map_oneof_proto2_defaults: cross-file import topology + oneof + map(enum) + proto2 defaults combined stability
	// - cross_file_service_return_map_enum: cross-file service output message shape with map(enum value) import topology stability
	// - cross_file_map_enum: cross-file enum + map(enum value)
	// - service_name_variants: underscore/multi-word service names and summary extraction consistency
	tests := []struct {
		name string
		req  *pluginpb.CodeGeneratorRequest
	}{
		{name: "minimal", req: parityMinimalRequest()},
		{name: "nested", req: parityNestedRequest()},
		{name: "wkt", req: parityWKTRequest()},
		{name: "auth_like", req: parityAuthLikeRequest()},
		{name: "int64_oneof_map", req: parityInt64OneofMapRequest()},
		{name: "nested_oneof_map_enum_i64", req: parityNestedOneofMapEnumI64Request()},
		{name: "proto2_presence", req: parityProto2PresenceRequest()},
		{name: "proto2_oneof_presence", req: parityProto2OneofPresenceRequest()},
		{name: "proto2_defaults", req: parityProto2DefaultsRequest()},
		{name: "proto2_bytes_defaults", req: parityProto2BytesDefaultsRequest()},
		{name: "proto2_cross_file_enum_default", req: parityProto2CrossFileEnumDefaultRequest()},
		{name: "proto2_float_invalid_default_tolerance", req: parityProto2FloatInvalidDefaultToleranceRequest()},
		{name: "proto2_numeric_hex_oct_invalid_default_tolerance", req: parityProto2NumericHexOctInvalidDefaultToleranceRequest()},
		{name: "cross_file_service_proto2_defaults", req: parityCrossFileServiceProto2DefaultsRequest()},
		{name: "cross_file_service_oneof_proto2_defaults", req: parityCrossFileServiceOneofProto2DefaultsRequest()},
		{name: "cross_file_map_oneof_proto2_defaults", req: parityCrossFileMapOneofProto2DefaultsRequest()},
		{name: "cross_file_service_return_map_enum", req: parityCrossFileServiceReturnMapEnumRequest()},
		{name: "cross_file_map_enum", req: parityCrossFileMapEnumRequest()},
		{name: "cross_file_service_external_messages", req: parityCrossFileServiceExternalMessagesRequest()},
		{name: "service_name_variants", req: parityServiceNameVariantsRequest()},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			jsResp, err := jsGen.Generate(tt.req)
			if err != nil {
				t.Fatalf("js generator failed: %v", err)
			}
			goResp, err := goGen.Generate(tt.req)
			if err != nil {
				t.Fatalf("go generator failed: %v", err)
			}

			jsSummary := gots.SummarizeResponse(jsResp)
			goSummary := gots.SummarizeResponse(goResp)
			diffs := gots.DiffFileSummariesWithOptions(
				jsSummary,
				goSummary,
				parityDiffOptions(),
			)
			if len(diffs) != 0 {
				scopes := diffScopes(diffs)
				t.Fatalf(
					"parity mismatch:\n%s\n\ntroubleshooting:\n%s\n\njs summary:\n%s\n\ngo summary:\n%s",
					formatDiffsByScope(diffs),
					parityTroubleshootingHint(tt.name, scopes),
					formatSummaries(jsSummary),
					formatSummaries(goSummary),
				)
			}
		})
	}
}

func formatDiffsByScope(diffs []string) string {
	if len(diffs) == 0 {
		return ""
	}
	groups := map[string][]string{}
	order := []string{"file", "import", "export", "schema_export", "service_export", "message_shape", "other"}
	for _, d := range diffs {
		scope := diffScope(d)
		groups[scope] = append(groups[scope], d)
	}
	for _, scope := range order {
		sort.Strings(groups[scope])
	}

	var b strings.Builder
	for _, scope := range order {
		items := groups[scope]
		if len(items) == 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[")
		b.WriteString(scope)
		b.WriteString("]")
		for _, it := range items {
			b.WriteString("\n")
			b.WriteString(it)
		}
	}
	return b.String()
}

func diffScope(diff string) string {
	switch {
	case strings.HasPrefix(diff, "missing file:"), strings.HasPrefix(diff, "unexpected file:"):
		return "file"
	case strings.HasPrefix(diff, "import mismatch"):
		return "import"
	case strings.HasPrefix(diff, "export mismatch"):
		return "export"
	case strings.HasPrefix(diff, "schema export mismatch"):
		return "schema_export"
	case strings.HasPrefix(diff, "service export mismatch"):
		return "service_export"
	case strings.HasPrefix(diff, "message shape "):
		return "message_shape"
	default:
		return "other"
	}
}

func diffScopes(diffs []string) []string {
	seen := map[string]struct{}{}
	for _, d := range diffs {
		seen[diffScope(d)] = struct{}{}
	}
	order := []string{"file", "import", "export", "schema_export", "service_export", "message_shape", "other"}
	out := make([]string, 0, len(seen))
	for _, s := range order {
		if _, ok := seen[s]; ok {
			out = append(out, s)
		}
	}
	return out
}

func parityTroubleshootingHint(fixtureName string, scopes []string) string {
	if len(scopes) == 0 {
		return "no scopes detected"
	}
	steps := []string{fmt.Sprintf("fixture=%s", fixtureName)}
	for _, s := range scopes {
		switch s {
		case "file":
			steps = append(steps, "1) Verify that fileToGenerate matches the output file name mapping")
		case "import":
			steps = append(steps, "2) Check cross-file type references and relative import topology")
		case "export":
			steps = append(steps, "3) Compare exported symbol sets, including naming and nesting")
		case "schema_export":
			steps = append(steps, "4) Compare message and enum schema exports")
		case "service_export":
			steps = append(steps, "5) Validate service constant exports and method signature references")
		case "message_shape":
			steps = append(steps, "6) Validate field signatures, especially optionality, 64-bit integers, and oneof")
		case "other":
			steps = append(steps, "7) Review remaining differences manually for semantic equivalence")
		}
		if owner, ok := scopeOwnerHint(s); ok {
			steps = append(steps, fmt.Sprintf("   owner: %s", owner))
		}
	}
	return strings.Join(steps, "\n")
}

func scopeOwnerHint(scope string) (string, bool) {
	switch scope {
	case "file", "import", "export", "schema_export", "service_export", "message_shape":
		return "internal/module/artifact/generate/grpcwebplugin/gots/generator.go", true
	case "other":
		return "internal/module/artifact/generate/grpcwebplugin/parity_test.go", true
	default:
		return "", false
	}
}

func TestFormatDiffsByScope(t *testing.T) {
	diffs := []string{
		"schema export mismatch in a_pb.ts: want=[ASchema] got=[]",
		"import mismatch in a_pb.ts: want=[./dep_pb] got=[./x_pb]",
		"message shape mismatch in a_pb.ts/Demo: want=[id: string] got=[id: number]",
		"unexpected file: z_pb.ts",
		"export mismatch in a_pb.ts: want=[A] got=[B]",
	}
	got := formatDiffsByScope(diffs)
	wantSections := []string{
		"[file]",
		"unexpected file: z_pb.ts",
		"[import]",
		"import mismatch in a_pb.ts: want=[./dep_pb] got=[./x_pb]",
		"[export]",
		"export mismatch in a_pb.ts: want=[A] got=[B]",
		"[schema_export]",
		"schema export mismatch in a_pb.ts: want=[ASchema] got=[]",
		"[message_shape]",
		"message shape mismatch in a_pb.ts/Demo: want=[id: string] got=[id: number]",
	}
	for _, w := range wantSections {
		if !strings.Contains(got, w) {
			t.Fatalf("formatted diff missing %q, got:\n%s", w, got)
		}
	}
}

func TestDiffScopes_OrderAndUniq(t *testing.T) {
	diffs := []string{
		"message shape mismatch in a_pb.ts/Demo: want=[id: string] got=[id: number]",
		"import mismatch in a_pb.ts: want=[./dep_pb] got=[./x_pb]",
		"import mismatch in b_pb.ts: want=[./dep_pb] got=[./x_pb]",
		"unexpected file: z_pb.ts",
	}
	got := diffScopes(diffs)
	want := []string{"file", "import", "message_shape"}
	if len(got) != len(want) {
		t.Fatalf("diffScopes len=%d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("diffScopes[%d]=%q, want %q (got=%v)", i, got[i], want[i], got)
		}
	}
}

func TestParityTroubleshootingHint(t *testing.T) {
	scopes := []string{"import", "message_shape"}
	got := parityTroubleshootingHint("auth_like", scopes)
	wants := []string{
		"fixture=auth_like",
		"Check cross-file type references and relative import topology",
		"Validate field signatures",
		"owner: internal/module/artifact/generate/grpcwebplugin/gots/generator.go",
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Fatalf("hint missing %q, got:\n%s", w, got)
		}
	}
}

func TestScopeOwnerHint(t *testing.T) {
	owner, ok := scopeOwnerHint("message_shape")
	if !ok {
		t.Fatalf("expected scope owner hint for message_shape")
	}
	if owner != "internal/module/artifact/generate/grpcwebplugin/gots/generator.go" {
		t.Fatalf("unexpected owner: %q", owner)
	}
}

func formatSummaries(items []gots.FileSummary) string {
	if len(items) == 0 {
		return "(empty)"
	}
	var b strings.Builder
	for i, s := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("- %s", s.Name))
		b.WriteString(fmt.Sprintf("\n  imports=%v", s.Imports))
		b.WriteString(fmt.Sprintf("\n  exports=%v", s.Exports))
		b.WriteString(fmt.Sprintf("\n  schemaExports=%v", s.SchemaExports))
		b.WriteString(fmt.Sprintf("\n  serviceExports=%v", s.ServiceExports))
		if len(s.MessageShapes) > 0 {
			b.WriteString(fmt.Sprintf("\n  messageShapes=%s", formatMessageShapes(s.MessageShapes)))
		}
	}
	return b.String()
}

func formatMessageShapes(shapes map[string][]string) string {
	if len(shapes) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(shapes))
	for k := range shapes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		fields := append([]string(nil), shapes[k]...)
		sort.Strings(fields)
		parts = append(parts, fmt.Sprintf("%s=%v", k, fields))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func parityMinimalRequest() *pluginpb.CodeGeneratorRequest {
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

func parityNestedRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("nested.proto"),
		Package: strPtr("sample"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Outer"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("Inner"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   strPtr("name"),
								Number: int32Ptr(1),
								Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
						},
					},
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: strPtr("Status"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{Name: strPtr("STATUS_UNSPECIFIED"), Number: int32Ptr(0)},
							{Name: strPtr("STATUS_READY"), Number: int32Ptr(1)},
						},
					},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("inner"),
						Number:   int32Ptr(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".sample.Outer.Inner"),
					},
					{
						Name:     strPtr("status"),
						Number:   int32Ptr(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: strPtr(".sample.Outer.Status"),
					},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"nested.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityWKTRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("wkt.proto"),
		Package:    strPtr("sample"),
		Dependency: []string{"google/protobuf/any.proto", "google/protobuf/duration.proto", "google/protobuf/empty.proto", "google/protobuf/field_mask.proto", "google/protobuf/struct.proto", "google/protobuf/timestamp.proto", "google/protobuf/wrappers.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("WktDemo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("empty"),
						Number:   int32Ptr(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.Empty"),
					},
					{
						Name:     strPtr("value"),
						Number:   int32Ptr(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.Value"),
					},
					{
						Name:     strPtr("list"),
						Number:   int32Ptr(3),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.ListValue"),
					},
					{
						Name:     strPtr("data"),
						Number:   int32Ptr(4),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.Struct"),
					},
					{
						Name:     strPtr("null_kind"),
						Number:   int32Ptr(5),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: strPtr(".google.protobuf.NullValue"),
					},
					{
						Name:     strPtr("details"),
						Number:   int32Ptr(6),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.Any"),
					},
					{
						Name:     strPtr("created_at"),
						Number:   int32Ptr(7),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.Timestamp"),
					},
					{
						Name:     strPtr("timeout"),
						Number:   int32Ptr(8),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.Duration"),
					},
					{
						Name:     strPtr("field_mask"),
						Number:   int32Ptr(9),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.FieldMask"),
					},
					{
						Name:     strPtr("double_value"),
						Number:   int32Ptr(10),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.DoubleValue"),
					},
					{
						Name:     strPtr("float_value"),
						Number:   int32Ptr(11),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.FloatValue"),
					},
					{
						Name:     strPtr("int64_value"),
						Number:   int32Ptr(12),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.Int64Value"),
					},
					{
						Name:     strPtr("uint64_value"),
						Number:   int32Ptr(13),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.UInt64Value"),
					},
					{
						Name:     strPtr("int32_value"),
						Number:   int32Ptr(14),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.Int32Value"),
					},
					{
						Name:     strPtr("uint32_value"),
						Number:   int32Ptr(15),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.UInt32Value"),
					},
					{
						Name:     strPtr("bool_value"),
						Number:   int32Ptr(16),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.BoolValue"),
					},
					{
						Name:     strPtr("string_value"),
						Number:   int32Ptr(17),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.StringValue"),
					},
					{
						Name:     strPtr("bytes_value"),
						Number:   int32Ptr(18),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".google.protobuf.BytesValue"),
					},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"wkt.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      append(parityWKTDependencyProtos(), fd),
	}
}

func parityWKTDependencyProtos() []*descriptorpb.FileDescriptorProto {
	anypb := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("google/protobuf/any.proto"),
		Package: strPtr("google.protobuf"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("Any")},
		},
	}
	durationpb := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("google/protobuf/duration.proto"),
		Package: strPtr("google.protobuf"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("Duration")},
		},
	}
	fieldMaskpb := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("google/protobuf/field_mask.proto"),
		Package: strPtr("google.protobuf"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("FieldMask")},
		},
	}
	empty := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("google/protobuf/empty.proto"),
		Package: strPtr("google.protobuf"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("Empty")},
		},
	}
	structpb := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("google/protobuf/struct.proto"),
		Package: strPtr("google.protobuf"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("Struct")},
			{Name: strPtr("Value")},
			{Name: strPtr("ListValue")},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("NullValue"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("NULL_VALUE"), Number: int32Ptr(0)},
				},
			},
		},
	}
	timestamppb := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("google/protobuf/timestamp.proto"),
		Package: strPtr("google.protobuf"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("Timestamp")},
		},
	}
	wrapperspb := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("google/protobuf/wrappers.proto"),
		Package: strPtr("google.protobuf"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("DoubleValue")},
			{Name: strPtr("FloatValue")},
			{Name: strPtr("Int64Value")},
			{Name: strPtr("UInt64Value")},
			{Name: strPtr("Int32Value")},
			{Name: strPtr("UInt32Value")},
			{Name: strPtr("BoolValue")},
			{Name: strPtr("StringValue")},
			{Name: strPtr("BytesValue")},
		},
	}
	return []*descriptorpb.FileDescriptorProto{anypb, durationpb, empty, fieldMaskpb, structpb, timestamppb, wrapperspb}
}

func parityAuthLikeRequest() *pluginpb.CodeGeneratorRequest {
	common := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("auth/common.proto"),
		Syntax:  strPtr("proto3"),
		Package: strPtr("auth.v1"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("RequestMeta"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   strPtr("trace_id"),
						Number: int32Ptr(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("auth/session.proto"),
		Syntax:     strPtr("proto3"),
		Package:    strPtr("auth.v1"),
		Dependency: []string{"auth/common.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("LoginRequest"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("MetaLabelsEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   strPtr("key"),
								Number: int32Ptr(1),
								Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
							{
								Name:   strPtr("value"),
								Number: int32Ptr(2),
								Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
						},
						Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
					},
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: strPtr("MfaKind"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{Name: strPtr("MFA_KIND_UNSPECIFIED"), Number: int32Ptr(0)},
							{Name: strPtr("MFA_KIND_TOTP"), Number: int32Ptr(1)},
						},
					},
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{Name: strPtr("auth_factor")},
					{Name: strPtr("_remember_me")},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   strPtr("username"),
						Number: int32Ptr(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:   strPtr("password"),
						Number: int32Ptr(2),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:     strPtr("meta"),
						Number:   int32Ptr(3),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".auth.v1.RequestMeta"),
					},
					{
						Name:       strPtr("otp_code"),
						Number:     int32Ptr(4),
						Label:      descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:       descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						OneofIndex: int32Ptr(0),
					},
					{
						Name:       strPtr("totp_token"),
						Number:     int32Ptr(5),
						Label:      descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:       descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						OneofIndex: int32Ptr(0),
					},
					{
						Name:           strPtr("remember_me"),
						Number:         int32Ptr(6),
						Label:          descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:           descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
						Proto3Optional: boolPtr(true),
						OneofIndex:     int32Ptr(1),
					},
					{
						Name:     strPtr("mfa_kind"),
						Number:   int32Ptr(7),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: strPtr(".auth.v1.LoginRequest.MfaKind"),
					},
					{
						Name:     strPtr("meta_labels"),
						Number:   int32Ptr(8),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: strPtr(".auth.v1.LoginRequest.MetaLabelsEntry"),
					},
				},
			},
			{
				Name: strPtr("LoginReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   strPtr("access_token"),
						Number: int32Ptr(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("SessionService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("Login"),
						InputType:  strPtr(".auth.v1.LoginRequest"),
						OutputType: strPtr(".auth.v1.LoginReply"),
					},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"auth/session.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{common, main},
	}
}

func parityCrossFileMapEnumRequest() *pluginpb.CodeGeneratorRequest {
	dep := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("dep.proto"),
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
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("main_enum_map.proto"),
		Package:    strPtr("sample.main"),
		Dependency: []string{"dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Catalog"),
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
					{Name: strPtr("status_by_key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.main.Catalog.StatusByKeyEntry")},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"main_enum_map.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	}
}

func parityNestedOneofMapEnumI64Request() *pluginpb.CodeGeneratorRequest {
	dep := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("dep_mode.proto"),
		Package: strPtr("sample.dep2"),
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
		Name:       strPtr("nested_mix.proto"),
		Package:    strPtr("sample.mix"),
		Dependency: []string{"dep_mode.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Container"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("Item"),
						NestedType: []*descriptorpb.DescriptorProto{
							{
								Name: strPtr("ModeByKeyEntry"),
								Field: []*descriptorpb.FieldDescriptorProto{
									{Name: strPtr("key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
									{Name: strPtr("value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.dep2.Mode")},
								},
								Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
							},
						},
						OneofDecl: []*descriptorpb.OneofDescriptorProto{
							{Name: strPtr("selector")},
						},
						Field: []*descriptorpb.FieldDescriptorProto{
							{Name: strPtr("legacy_id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(), OneofIndex: int32Ptr(0)},
							{Name: strPtr("alias"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), OneofIndex: int32Ptr(0)},
							{Name: strPtr("version"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum()},
							{Name: strPtr("mode_by_key"), Number: int32Ptr(4), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.mix.Container.Item.ModeByKeyEntry")},
						},
					},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("items"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.mix.Container.Item")},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("MixedService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: strPtr("Upsert"), InputType: strPtr(".sample.mix.Container"), OutputType: strPtr(".sample.mix.Container")},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"nested_mix.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	}
}

func parityInt64OneofMapRequest() *pluginpb.CodeGeneratorRequest {
	dep := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("ledger_dep.proto"),
		Package: strPtr("sample.ledgerdep"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Meta"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("source"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("ledger.proto"),
		Package:    strPtr("sample.ledger"),
		Dependency: []string{"ledger_dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("LedgerEntry"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: strPtr("MetaByKeyEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{Name: strPtr("key"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
							{Name: strPtr("value"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.ledgerdep.Meta")},
						},
						Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
					},
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{Name: strPtr("id_selector")},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("entry_id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum()},
					{Name: strPtr("seq_no"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum()},
					{Name: strPtr("delta"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_SINT64.Enum()},
					{Name: strPtr("fixed_counter"), Number: int32Ptr(4), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_FIXED64.Enum()},
					{Name: strPtr("signed_fixed_counter"), Number: int32Ptr(5), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_SFIXED64.Enum()},
					{Name: strPtr("legacy_id"), Number: int32Ptr(6), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("display_id"), Number: int32Ptr(7), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), OneofIndex: int32Ptr(0)},
					{Name: strPtr("meta_by_key"), Number: int32Ptr(8), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.ledger.LedgerEntry.MetaByKeyEntry")},
				},
			},
			{
				Name: strPtr("LedgerQuery"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("entry_id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum()},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("LedgerService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: strPtr("GetEntry"), InputType: strPtr(".sample.ledger.LedgerQuery"), OutputType: strPtr(".sample.ledger.LedgerEntry")},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"ledger.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	}
}

func parityProto2PresenceRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_presence.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("PresenceDemo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("opt_name"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("req_id"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum()},
					{Name: strPtr("tags"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_presence.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityProto2OneofPresenceRequest() *pluginpb.CodeGeneratorRequest {
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

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_oneof_presence.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityProto2DefaultsRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_defaults.proto"),
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
				Name: strPtr("DefaultsDemo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("name"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), DefaultValue: strPtr("guest")},
					{Name: strPtr("retries"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("3")},
					{Name: strPtr("mode"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.proto2.Mode"), DefaultValue: strPtr("MODE_FAST")},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_defaults.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityProto2BytesDefaultsRequest() *pluginpb.CodeGeneratorRequest {
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

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_bytes_defaults.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityProto2CrossFileEnumDefaultRequest() *pluginpb.CodeGeneratorRequest {
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

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	}
}

func parityProto2MessageDefaultToleranceRequest() *pluginpb.CodeGeneratorRequest {
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
					{Name: strPtr("inner"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.proto2.Inner"), DefaultValue: strPtr("{id:'x'}")},
					{Name: strPtr("req_inner"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".sample.proto2.Inner"), DefaultValue: strPtr("{id:'y'}")},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_message_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityProto2EnumInvalidDefaultToleranceRequest() *pluginpb.CodeGeneratorRequest {
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
					{Name: strPtr("mode"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.proto2.Mode"), DefaultValue: strPtr("MODE_GHOST")},
					{Name: strPtr("req_mode"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: strPtr(".sample.proto2.Mode"), DefaultValue: strPtr("MODE_MISSING")},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_enum_invalid_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityProto2NumericInvalidDefaultToleranceRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_numeric_invalid_default_tolerance.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("NumericDefaultTolerance"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("opt_i32"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("2147483648")},
					{Name: strPtr("req_u32"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(), DefaultValue: strPtr("-1")},
					{Name: strPtr("opt_token"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("not_a_number")},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_numeric_invalid_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityProto2FloatInvalidDefaultToleranceRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_float_invalid_default_tolerance.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("FloatDefaultTolerance"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("opt_f32"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum(), DefaultValue: strPtr("NaN")},
					{Name: strPtr("req_f64"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), DefaultValue: strPtr("Inf")},
					{Name: strPtr("opt_huge"), Number: int32Ptr(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum(), DefaultValue: strPtr("1e9999")},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_float_invalid_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityProto2NumericHexOctInvalidDefaultToleranceRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("proto2_numeric_hex_oct_invalid_default_tolerance.proto"),
		Package: strPtr("sample.proto2"),
		Syntax:  strPtr("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("NumericHexOctTolerance"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("opt_hex"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), DefaultValue: strPtr("0xFF")},
					{Name: strPtr("req_oct"), Number: int32Ptr(2), Label: descriptorpb.FieldDescriptorProto_LABEL_REQUIRED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_UINT32.Enum(), DefaultValue: strPtr("077")},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_numeric_hex_oct_invalid_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityProto2FloatSignedTokenDefaultToleranceRequest() *pluginpb.CodeGeneratorRequest {
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

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_float_signed_token_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityProto2FloatExtremeSignedTokenDefaultToleranceRequest() *pluginpb.CodeGeneratorRequest {
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

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"proto2_float_extreme_signed_token_default_tolerance.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fd},
	}
}

func parityCrossFileServiceProto2DefaultsRequest() *pluginpb.CodeGeneratorRequest {
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

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"service_defaults_main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	}
}

func parityCrossFileServiceOneofProto2DefaultsRequest() *pluginpb.CodeGeneratorRequest {
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

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"service_oneof_defaults_main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	}
}

func parityCrossFileMapOneofProto2DefaultsRequest() *pluginpb.CodeGeneratorRequest {
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

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"map_oneof_defaults_main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	}
}

func parityCrossFileServiceReturnMapEnumRequest() *pluginpb.CodeGeneratorRequest {
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

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"service_return_map_enum_main.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{dep, main},
	}
}

func parityCrossFileServiceExternalMessagesRequest() *pluginpb.CodeGeneratorRequest {
	ext := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("external.proto"),
		Package: strPtr("sample.external"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("ExternalRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
			{
				Name: strPtr("ExternalReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("ok"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()},
				},
			},
		},
	}

	main := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("gateway.proto"),
		Package:    strPtr("sample.gateway"),
		Dependency: []string{"external.proto"},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("GatewayService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("Forward"),
						InputType:  strPtr(".sample.external.ExternalRequest"),
						OutputType: strPtr(".sample.external.ExternalReply"),
					},
				},
			},
		},
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"gateway.proto"},
		Parameter:      strPtr("target=ts"),
		ProtoFile:      []*descriptorpb.FileDescriptorProto{ext, main},
	}
}

func parityServiceNameVariantsRequest() *pluginpb.CodeGeneratorRequest {
	fd := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("service_variants.proto"),
		Package: strPtr("sample.service"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("PingRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("id"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
			{
				Name: strPtr("PingReply"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("ok"), Number: int32Ptr(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()},
				},
			},
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

func strPtr(s string) *string { return &s }
func int32Ptr(v int32) *int32 { return &v }
func boolPtr(v bool) *bool    { return &v }
