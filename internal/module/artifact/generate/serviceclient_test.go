// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestServiceClientGenerate(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	protoDir := t.TempDir()
	serviceDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(protoDir, "google", "protobuf"), 0o755); err != nil {
		t.Fatalf("mkdir google proto dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(protoDir, "partner.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatalf("write partner proto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(protoDir, "google", "protobuf", "empty.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatalf("write google proto: %v", err)
	}

	gen := &serviceClientGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesProtoDir: protoDir, modulesServiceDir: serviceDir}
	protoFiles, err := gen.collectProtoFiles("crm")
	if err != nil {
		t.Fatalf("collectProtoFiles() error = %v", err)
	}
	if len(protoFiles) != 2 || protoFiles[0].RegisterPath != "crm/partner.proto" || protoFiles[1].RegisterPath != "google/protobuf/empty.proto" {
		t.Fatalf("unexpected proto files: %#v", protoFiles)
	}
	if got := resolveProtoRegisterPath("crm", filepath.Join("google", "protobuf", "empty.proto")); got != "google/protobuf/empty.proto" {
		t.Fatalf("unexpected google register path: %q", got)
	}
	if got := resolveProtoRegisterPath("crm", filepath.Join("sub", "partner.proto")); got != "crm/sub/partner.proto" {
		t.Fatalf("unexpected app register path: %q", got)
	}
	if NewServiceClientGenerator(runtimeScope, &meta.IrModule{Name: "base"}) == nil {
		t.Fatal("expected service client generator constructor to return non-nil")
	}

	results, err := gen.generate(context.Background(), testApp())
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	if len(results) != 1 || results[0].Name != "serviceclient" || len(results[0].OutPaths) != 2 {
		t.Fatalf("unexpected generation results: %#v", results)
	}
	content, err := os.ReadFile(filepath.Join(serviceDir, "service.ts"))
	if err != nil {
		t.Fatalf("read generated service.ts: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "crm/partner.proto") || !strings.Contains(text, "google/protobuf/empty.proto") || !strings.Contains(text, "CreateServerApiService") || !strings.Contains(text, "registerServiceFactory") || !strings.Contains(text, "registerServiceFactory('crm.Partner'") || !strings.Contains(text, "@/core/service/rpc") {
		t.Fatalf("unexpected service.ts content: %s", text)
	}
	if _, err := os.Stat(filepath.Join(serviceDir, "index.ts")); err != nil {
		t.Fatalf("expected index.ts: %v", err)
	}
}

func TestServiceClientGenerateEdgeCases(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	gen := &serviceClientGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesProtoDir: t.TempDir(), modulesServiceDir: t.TempDir()}

	results, err := gen.generate(context.Background(), &meta.IrApplication{Name: "crm"})
	if err != nil {
		t.Fatalf("generate(empty app) error = %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results for app without models, got %#v", results)
	}

	_, err = gen.generate(context.Background(), testApp())
	if err == nil || !strings.Contains(err.Error(), "no proto files found") {
		t.Fatalf("expected missing proto files error, got %v", err)
	}
}
