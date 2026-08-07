// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/pkg/config"
)

func TestBootstrapValidateRuntimeReadyUsesManifestCompileBundleMode(t *testing.T) {
	distRoot := t.TempDir()

	manifest := `{"schemaVersion":2,"compileBundleMode":"application","apps":{"auth":{"deps":{"apps":[]},"dev":{"modules":[]}}}}`
	if err := os.WriteFile(filepath.Join(distRoot, distmanifest.DistManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(distRoot, "web"), 0o755); err != nil {
		t.Fatalf("mkdir web: %v", err)
	}
	seedApplicationModeWebBackendDist(t, distRoot)

	if err := os.MkdirAll(filepath.Join(distRoot, "apps", "auth", "assets"), 0o755); err != nil {
		t.Fatalf("mkdir app assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "apps", "auth", "index.js"), []byte("console.log('auth')"), 0o644); err != nil {
		t.Fatalf("write app index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "apps", "auth", "assets", "proto.bin"), []byte("proto"), 0o644); err != nil {
		t.Fatalf("write app asset: %v", err)
	}

	runtimeScope := newServerTestScope()
	runtimeScope.cfg.DistPath = distRoot
	runtimeScope.cfg.Compile = &config.CompileConfig{BundleMode: "bundle"}

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	if err := srv.bootstrapValidateRuntimeReady(context.Background()); err != nil {
		t.Fatalf("bootstrapValidateRuntimeReady() error = %v, want nil", err)
	}
}

func TestBootstrapValidateRuntimeReadyFailsWhenAuthRuntimeMissing(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distRoot, "web"), 0o755); err != nil {
		t.Fatalf("mkdir web: %v", err)
	}

	runtimeScope := newServerTestScope()
	runtimeScope.cfg.DistPath = distRoot
	runtimeScope.cfg.Compile = &config.CompileConfig{BundleMode: "application"}

	srv := &GRPCWebServer{runtimeScope: runtimeScope}
	err := srv.bootstrapValidateRuntimeReady(context.Background())
	if err == nil {
		t.Fatal("expected runtime readiness error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "app index missing") {
		t.Fatalf("error = %v, want message containing app index missing", err)
	}
}
