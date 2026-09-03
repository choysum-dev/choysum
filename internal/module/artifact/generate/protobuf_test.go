// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestProtobufGenerateUsesProtobufTypeForEmptyReturns(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	protoDir := t.TempDir()
	distAppDir := filepath.Join(t.TempDir(), "apps", "crm")
	gen := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "crm"}, modulesProtoDir: protoDir, distAppDir: distAppDir}

	app := &meta.Application{
		Name: "crm",
		Models: []*meta.Model{{
			Name: "Partner",
			Services: []*meta.Service{{
				Name:                  "Create",
				AccessibilityModifier: "public",
				IsStatic:              true,
				// Text annotation still looks non-void; ProtobufType is the source of truth.
				TsTypeAnnotation: "Promise<void>",
				ProtobufType:     "google.protobuf.Empty",
			}},
		}},
	}

	if _, err := gen.generate(context.Background(), app); err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(protoDir, "crm.proto"))
	if err != nil {
		t.Fatalf("read crm.proto: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "rpc Create ( google.protobuf.Empty ) returns ( google.protobuf.Empty )") {
		t.Fatalf("expected Empty return rpc, got:\n%s", text)
	}
	if strings.Contains(text, "Partner_Create_Resp") {
		t.Fatalf("did not expect Resp message for Empty return, got:\n%s", text)
	}
}

func TestProtobufGenerateRepeatedAndTimestamp(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	protoDir := t.TempDir()
	distAppDir := filepath.Join(t.TempDir(), "apps", "crm")
	gen := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "crm"}, modulesProtoDir: protoDir, distAppDir: distAppDir}

	app := &meta.Application{
		Name: "crm",
		Models: []*meta.Model{{
			Name: "Partner",
			Services: []*meta.Service{
				{
					Name:                  "ListTags",
					AccessibilityModifier: "public",
					IsStatic:              true,
					ProtobufType:          "repeated string",
					Parameters: []*meta.Parameter{{
						Name:         "tags",
						ProtobufType: "repeated string",
					}},
				},
				{
					Name:                  "Touch",
					AccessibilityModifier: "public",
					IsStatic:              true,
					ProtobufType:          "google.protobuf.Timestamp",
					Parameters: []*meta.Parameter{{
						Name:         "at",
						ProtobufType: "google.protobuf.Timestamp",
					}},
				},
			},
		}},
	}

	if _, err := gen.generate(context.Background(), app); err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(protoDir, "crm.proto"))
	if err != nil {
		t.Fatalf("read crm.proto: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, `import "google/protobuf/timestamp.proto";`) {
		t.Fatalf("expected timestamp import, got:\n%s", text)
	}
	if !strings.Contains(text, "repeated string tags = 1;") {
		t.Fatalf("expected repeated string param, got:\n%s", text)
	}
	if !strings.Contains(text, "repeated string result = 1;") {
		t.Fatalf("expected repeated string result, got:\n%s", text)
	}
	if !strings.Contains(text, "google.protobuf.Timestamp at = 1;") {
		t.Fatalf("expected Timestamp param, got:\n%s", text)
	}
	if _, err := os.Stat(filepath.Join(protoDir, "google", "protobuf", "timestamp.proto")); err != nil {
		t.Fatalf("expected embedded timestamp.proto copied: %v", err)
	}
}

func TestProtobufGenerateOmitsTimestampImportWhenUnused(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	protoDir := t.TempDir()
	distAppDir := filepath.Join(t.TempDir(), "apps", "crm")
	gen := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "crm"}, modulesProtoDir: protoDir, distAppDir: distAppDir}

	app := &meta.Application{
		Name: "crm",
		Models: []*meta.Model{{
			Name: "Partner",
			Services: []*meta.Service{{
				Name:                  "Echo",
				AccessibilityModifier: "public",
				IsStatic:              true,
				ProtobufType:          "string",
				Parameters:            []*meta.Parameter{{Name: "v", ProtobufType: "string"}},
			}},
		}},
	}
	if _, err := gen.generate(context.Background(), app); err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(protoDir, "crm.proto"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(content), "timestamp.proto") {
		t.Fatalf("did not expect timestamp import:\n%s", content)
	}
	if _, err := os.Stat(filepath.Join(protoDir, "google", "protobuf", "timestamp.proto")); !os.IsNotExist(err) {
		t.Fatalf("expected timestamp.proto not copied when unused, err=%v", err)
	}
}

func TestProtobufNeedsTimestamp(t *testing.T) {
	if protobufNeedsTimestamp(nil) {
		t.Fatal("nil app")
	}
	if protobufNeedsTimestamp(&meta.Application{Models: []*meta.Model{nil, {Name: "M", Services: []*meta.Service{nil}}}}) {
		t.Fatal("nil model/service")
	}
	if !protobufNeedsTimestamp(&meta.Application{Models: []*meta.Model{{
		Services: []*meta.Service{{
			ProtobufType: "string",
			Parameters:   []*meta.Parameter{nil, {ProtobufType: "google.protobuf.Timestamp"}},
		}},
	}}}) {
		t.Fatal("expected param Timestamp")
	}
	if !protobufNeedsTimestamp(&meta.Application{Models: []*meta.Model{{
		Services: []*meta.Service{{ProtobufType: "google.protobuf.Timestamp"}},
	}}}) {
		t.Fatal("expected return Timestamp")
	}
}

func TestProtobufGenerateWritesEmbeddedAssetsAndAppProto(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	protoDir := t.TempDir()
	distAppDir := filepath.Join(t.TempDir(), "apps", "crm")
	gen := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "crm"}, modulesProtoDir: protoDir, distAppDir: distAppDir}

	results, err := gen.generate(context.Background(), testApp())
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	if len(results) != 1 || results[0].Name != "protobuf" {
		t.Fatalf("unexpected protobuf results: %#v", results)
	}
	if len(results[0].OutPaths) < 2 {
		t.Fatalf("expected app proto plus embedded proto assets, got %#v", results[0].OutPaths)
	}

	appProtoPath := filepath.Join(protoDir, "crm.proto")
	content, err := os.ReadFile(appProtoPath)
	if err != nil {
		t.Fatalf("read generated crm.proto: %v", err)
	}
	if text := string(content); !strings.Contains(text, "service Partner") || !strings.Contains(text, "message Partner_CreatePartner_Req") {
		t.Fatalf("unexpected crm.proto content: %s", text)
	}

	protoCount := 0
	err = filepath.WalkDir(protoDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".proto" {
			protoCount++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk generated proto dir: %v", err)
	}
	if protoCount < 2 {
		t.Fatalf("expected embedded proto assets to be copied, protoCount = %d", protoCount)
	}
	if _, err := os.Stat(filepath.Join(distAppDir, "assets", "crm.proto")); err != nil {
		t.Fatalf("expected dist app proto sync: %v", err)
	}
}

func TestProtobufGenerateUsesStagingWhenNoExplicitTargets(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	runtimeScope.cfg.ModulesPath = t.TempDir()
	runtimeScope.cfg.DistPath = t.TempDir()
	gen := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "crm"}}
	ctx := staging.WithTmpRoot(context.Background(), t.TempDir())

	results, err := gen.generate(ctx, testApp())
	if err != nil {
		t.Fatalf("generate(staging) error = %v", err)
	}
	if len(results) != 1 || len(results[0].OutPaths) == 0 {
		t.Fatalf("unexpected staging results: %#v", results)
	}
	generatedRoot, err := WorkspaceGeneratedAPIRoot(runtimeScope.cfg.ModulesPath, runtimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(generatedRoot, "proto", "crm", "crm.proto")); err != nil {
		t.Fatalf("expected staged modules proto output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runtimeScope.cfg.DistPath, "apps", "crm", "assets", "crm.proto")); err != nil {
		t.Fatalf("expected staged dist proto output: %v", err)
	}
}

func TestProtobufGenerateModulesProtoDirBranchErrorPaths(t *testing.T) {
	t.Run("writeAll error is propagated", func(t *testing.T) {
		runtimeScope := newGeneratorScope(t)
		blockedPath := filepath.Join(t.TempDir(), "blocked")
		if err := os.WriteFile(blockedPath, []byte("file blocks mkdir"), 0o644); err != nil {
			t.Fatalf("write blocked path file: %v", err)
		}

		gen := &protobufGenerator{
			runtimeScope:    runtimeScope,
			module:          &meta.Module{ApplicationStr: "crm"},
			modulesProtoDir: blockedPath,
		}
		if _, err := gen.generate(context.Background(), testApp()); err == nil {
			t.Fatal("expected generate() to fail when modulesProtoDir cannot be created")
		}
	})

	t.Run("syncProtoToDistDirect error is propagated", func(t *testing.T) {
		runtimeScope := newGeneratorScope(t)
		gen := &protobufGenerator{
			runtimeScope:    runtimeScope,
			module:          &meta.Module{ApplicationStr: "crm"},
			modulesProtoDir: t.TempDir(),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := gen.generate(ctx, testApp())
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("generate(canceled) error = %v, want context.Canceled", err)
		}
	})
}

func TestSyncProtoToDist_ApplicationCopies_ToAppsAssets_NoLegacyFallback(t *testing.T) {
	ctx := staging.WithTmpRoot(context.Background(), t.TempDir())
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatalf("write a.proto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatalf("write b.proto: %v", err)
	}

	distRoot := t.TempDir()
	runtimeScope := newGeneratorScope(t)
	runtimeScope.cfg.DistPath = distRoot
	runtimeScope.cfg.Compile.BundleMode = string(config.BundleModeApplication)

	g := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "auth"}}
	if err := g.syncProtoToDist(ctx, src); err != nil {
		t.Fatalf("syncProtoToDist: %v", err)
	}

	newA := filepath.Join(distRoot, "apps", "auth", "assets", "a.proto")
	newB := filepath.Join(distRoot, "apps", "auth", "assets", "sub", "b.proto")
	if _, err := os.Stat(newA); err != nil {
		t.Fatalf("expected %s to exist: %v", newA, err)
	}
	if _, err := os.Stat(newB); err != nil {
		t.Fatalf("expected %s to exist: %v", newB, err)
	}
	if _, err := os.Stat(filepath.Join(distRoot, "auth")); err == nil {
		t.Fatalf("expected legacy dist directory to not exist")
	}
}

func TestSyncProtoToDist_BundleMode_NoOp(t *testing.T) {
	ctx := context.Background()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatalf("write a.proto: %v", err)
	}

	distRoot := t.TempDir()
	runtimeScope := newGeneratorScope(t)
	runtimeScope.cfg.DistPath = distRoot
	runtimeScope.cfg.Compile.BundleMode = string(config.BundleModeBundle)

	g := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "auth"}, distAppDir: filepath.Join(distRoot, "apps", "auth")}
	if err := g.syncProtoToDist(ctx, src); err != nil {
		t.Fatalf("syncProtoToDist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(distRoot, "apps")); err == nil {
		t.Fatalf("expected dist/apps to not exist in bundle mode")
	}
	if _, err := os.Stat(filepath.Join(distRoot, "auth")); err == nil {
		t.Fatalf("expected legacy dist/<app> to not exist in bundle mode")
	}
}

func TestSyncProtoToDistDirect_ApplicationCopies_BundleSkips(t *testing.T) {
	ctx := context.Background()
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatalf("write a.proto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatalf("write b.proto: %v", err)
	}

	t.Run("application copies to dist/apps/<app>/assets", func(t *testing.T) {
		distRoot := t.TempDir()
		runtimeScope := newGeneratorScope(t)
		runtimeScope.cfg.DistPath = distRoot
		runtimeScope.cfg.Compile.BundleMode = string(config.BundleModeApplication)
		g := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "auth"}}
		if err := g.syncProtoToDistDirect(ctx, src); err != nil {
			t.Fatalf("syncProtoToDistDirect: %v", err)
		}
		wantA := filepath.Join(distRoot, "apps", "auth", "assets", "a.proto")
		wantB := filepath.Join(distRoot, "apps", "auth", "assets", "sub", "b.proto")
		if _, err := os.Stat(wantA); err != nil {
			t.Fatalf("expected %s to exist: %v", wantA, err)
		}
		if _, err := os.Stat(wantB); err != nil {
			t.Fatalf("expected %s to exist: %v", wantB, err)
		}
	})

	t.Run("bundle mode skips per-app dist write", func(t *testing.T) {
		distRoot := t.TempDir()
		runtimeScope := newGeneratorScope(t)
		runtimeScope.cfg.DistPath = distRoot
		runtimeScope.cfg.Compile.BundleMode = string(config.BundleModeBundle)
		g := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.Module{ApplicationStr: "auth"}, distAppDir: filepath.Join(distRoot, "apps", "auth")}
		if err := g.syncProtoToDistDirect(ctx, src); err != nil {
			t.Fatalf("syncProtoToDistDirect: %v", err)
		}
		if _, err := os.Stat(filepath.Join(distRoot, "apps")); err == nil {
			t.Fatalf("expected dist/apps to not exist in bundle mode")
		}
	})
}
