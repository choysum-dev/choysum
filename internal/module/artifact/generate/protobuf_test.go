// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestProtobufGenerateWritesEmbeddedAssetsAndAppProto(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	protoDir := t.TempDir()
	distAppDir := filepath.Join(t.TempDir(), "apps", "crm")
	gen := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, addonsProtoDir: protoDir, distAppDir: distAppDir}

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
	runtimeScope.cfg.AddonsPath = t.TempDir()
	runtimeScope.cfg.DistPath = t.TempDir()
	gen := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}}
	ctx := staging.WithTmpRoot(context.Background(), t.TempDir())

	results, err := gen.generate(ctx, testApp())
	if err != nil {
		t.Fatalf("generate(staging) error = %v", err)
	}
	if len(results) != 1 || len(results[0].OutPaths) == 0 {
		t.Fatalf("unexpected staging results: %#v", results)
	}
	generatedRoot, err := WorkspaceGeneratedAPIRoot(runtimeScope.cfg.AddonsPath, runtimeScope.cfg.DefaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPIRoot() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(generatedRoot, "proto", "crm", "crm.proto")); err != nil {
		t.Fatalf("expected staged addons proto output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(runtimeScope.cfg.DistPath, "apps", "crm", "assets", "crm.proto")); err != nil {
		t.Fatalf("expected staged dist proto output: %v", err)
	}
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

	g := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "auth"}}
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

	g := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "auth"}, distAppDir: filepath.Join(distRoot, "apps", "auth")}
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
		g := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "auth"}}
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
		g := &protobufGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "auth"}, distAppDir: filepath.Join(distRoot, "apps", "auth")}
		if err := g.syncProtoToDistDirect(ctx, src); err != nil {
			t.Fatalf("syncProtoToDistDirect: %v", err)
		}
		if _, err := os.Stat(filepath.Join(distRoot, "apps")); err == nil {
			t.Fatalf("expected dist/apps to not exist in bundle mode")
		}
	})
}
