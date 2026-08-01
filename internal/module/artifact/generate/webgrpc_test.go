// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/meta"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

type fakeGrpcPlugin struct {
	name string
}

func (p fakeGrpcPlugin) Name() string { return p.name }

func (p fakeGrpcPlugin) Generate(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	if len(req.GetFileToGenerate()) == 0 {
		return &pluginpb.CodeGeneratorResponse{}, nil
	}
	return &pluginpb.CodeGeneratorResponse{File: []*pluginpb.CodeGeneratorResponse_File{{
		Name:    proto.String("crm_pb.ts"),
		Content: proto.String("export const generated = true;\n"),
	}}}, nil
}

func TestWebGrpcGenerate(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	webGrpc := NewWebGrpcGenerator(runtimeScope, &meta.Module{Name: "base"})
	if webGrpc == nil || webGrpc.buildParameters() != "target=ts" || len(webGrpc.plugins) == 0 {
		t.Fatalf("unexpected web grpc generator constructor state: %#v", webGrpc)
	}

	protoDir := t.TempDir()
	outDir := t.TempDir()
	protoPath := filepath.Join(protoDir, "crm.proto")
	protoContent := "syntax = \"proto3\"; package crm; message PingRequest {} message PingReply {} service Partner { rpc Ping(PingRequest) returns (PingReply); }"
	if err := os.WriteFile(protoPath, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("write proto file: %v", err)
	}

	gen := &webGrpcGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "crm"}, plugins: []GrpcPlugin{fakeGrpcPlugin{name: "fake-grpc"}}, modulesProtoDir: protoDir, modulesWebDir: outDir}
	request, err := gen.buildCodeGeneratorRequest([]string{protoPath})
	if err != nil {
		t.Fatalf("buildCodeGeneratorRequest() error = %v", err)
	}
	if len(request.GetFileToGenerate()) != 1 || request.GetFileToGenerate()[0] != "crm.proto" || request.GetParameter() != "target=ts" {
		t.Fatalf("unexpected request: %#v", request)
	}

	results, err := gen.generate(context.Background(), []*module.GeneratorResult{{Name: "protobuf", OutPaths: []string{protoPath}}})
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected original result plus plugin result, got %#v", results)
	}
	generatedPath := filepath.Join(outDir, "pb", "crm_pb.ts")
	content, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("read generated grpc file: %v", err)
	}
	if !strings.Contains(string(content), "generated = true") {
		t.Fatalf("unexpected generated grpc content: %s", string(content))
	}
}

func TestWebGrpcGenerateWithoutProtoResultsReturnsInput(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	gen := &webGrpcGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "crm"}, plugins: []GrpcPlugin{fakeGrpcPlugin{name: "fake-grpc"}}, modulesWebDir: t.TempDir()}
	input := []*module.GeneratorResult{{Name: "webservice", OutPaths: []string{"service.ts"}}}

	results, err := gen.generate(context.Background(), input)
	if err != nil {
		t.Fatalf("generate(no proto) error = %v", err)
	}
	if len(results) != 1 || results[0].Name != "webservice" {
		t.Fatalf("expected original results to pass through, got %#v", results)
	}
}

func TestWebGrpcGenerate_UsesWorkspaceGeneratedTargets(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	protoDir, webDir, _, err := WorkspaceGeneratedAPITargets(runtimeScope.cfg.ModulesPath, "crm", runtimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets() error = %v", err)
	}
	protoPath := filepath.Join(protoDir, "crm.proto")
	if err := os.MkdirAll(filepath.Dir(protoPath), 0o755); err != nil {
		t.Fatalf("mkdir proto dir: %v", err)
	}
	protoContent := "syntax = \"proto3\"; package crm; message PingRequest {} message PingReply {} service Partner { rpc Ping(PingRequest) returns (PingReply); }"
	if err := os.WriteFile(protoPath, []byte(protoContent), 0o644); err != nil {
		t.Fatalf("write proto file: %v", err)
	}

	gen := &webGrpcGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "crm"}, plugins: []GrpcPlugin{fakeGrpcPlugin{name: "fake-grpc"}}}
	ctx := staging.WithTmpRoot(context.Background(), t.TempDir())
	results, err := gen.generate(ctx, []*module.GeneratorResult{{Name: "protobuf", OutPaths: []string{protoPath}}})
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected original result plus plugin result, got %#v", results)
	}
	if _, err := os.Stat(filepath.Join(webDir, "pb", "crm_pb.ts")); err != nil {
		t.Fatalf("expected generated grpc file in workspace target: %v", err)
	}
}

func TestWebGrpcGenerate_WorkspaceTargetsRequireDefaultChoysumPath(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	runtimeScope.cfg.DefaultChoysumPath = ""

	gen := &webGrpcGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "crm"}, plugins: []GrpcPlugin{fakeGrpcPlugin{name: "fake-grpc"}}}
	_, err := gen.generate(context.Background(), []*module.GeneratorResult{{Name: "protobuf", OutPaths: []string{"crm.proto"}}})
	if err == nil || !strings.Contains(err.Error(), "resolve workspace generated api targets") {
		t.Fatalf("expected workspace target resolution error, got %v", err)
	}
}
