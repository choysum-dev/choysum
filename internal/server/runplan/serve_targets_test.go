// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runplan

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/distmanifest"
)

func TestResolveServeTargets_UsesManifestDefaultsAndWebDir(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distRoot, "web"), 0o755); err != nil {
		t.Fatalf("mkdir web dir: %v", err)
	}
	manifest := &distmanifest.DistManifestV2{
		SchemaVersion:     distmanifest.SchemaVersion,
		CompileBundleMode: "application",
		HasWeb:            true,
		BackendTopoOrder:  []string{"base", "auth"},
	}

	bundleMode, targets, err := resolveServeTargets(distRoot, "bundle", nil, manifest, nil)
	if err != nil {
		t.Fatalf("resolveServeTargets() error = %v", err)
	}
	if bundleMode != "application" {
		t.Fatalf("bundle mode = %q, want application", bundleMode)
	}
	if got, want := strings.Join(targets, ","), "base,auth,web"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}
}

func TestResolveServeTargets_ExplicitArgsWarnsForUnknownAndMissingDeps(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	manifest := &distmanifest.DistManifestV2{
		SchemaVersion:    distmanifest.SchemaVersion,
		BackendTopoOrder: []string{"base", "auth"},
		Apps: map[string]distmanifest.DistManifestApp{
			"auth": {Deps: distmanifest.DistManifestAppDeps{Apps: []string{"base"}}},
		},
	}

	_, targets, err := resolveServeTargets(t.TempDir(), "bundle", logger, manifest, []string{" unknown ", "auth", "web", "auth"})
	if err != nil {
		t.Fatalf("resolveServeTargets() error = %v", err)
	}
	if got, want := strings.Join(targets, ","), "auth,unknown,web"; got != want {
		t.Fatalf("targets = %q, want %q", got, want)
	}

	logText := buf.String()
	if !strings.Contains(logText, "serve targets order fallback") || !strings.Contains(logText, "reason=requested_apps_missing_from_manifest") {
		t.Fatalf("expected unknown app warning, got %q", logText)
	}
	if !strings.Contains(logText, "serve targets dependency warning") || !strings.Contains(logText, "reason=requested_apps_missing_dependencies") || !strings.Contains(logText, "base") {
		t.Fatalf("expected missing dependency warning, got %q", logText)
	}
}

func TestResolveServeTargets_ManifestMissing_FallsBackToAlphabeticalAndWarns(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	_, targets, err := resolveServeTargets(t.TempDir(), "bundle", logger, nil, []string{" core ", "auth", "core"})
	if err != nil {
		t.Fatalf("resolveServeTargets: %v", err)
	}
	if want := []string{"auth", "core"}; strings.Join(targets, ",") != strings.Join(want, ",") {
		t.Fatalf("targets mismatch: want %v, got %v", want, targets)
	}

	logText := buf.String()
	if !strings.Contains(logText, "serve targets order fallback") || !strings.Contains(logText, "reason=dist_manifest_missing") {
		t.Fatalf("expected fallback warning in logs, got: %s", logText)
	}
}

func TestResolveServeTargets_DefaultManifestMissingDoesNotWarnBeforeBootstrapDecision(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	_, targets, err := resolveServeTargets(t.TempDir(), "bundle", logger, nil, nil)
	if err != nil {
		t.Fatalf("resolveServeTargets() error = %v", err)
	}
	if len(targets) != 0 {
		t.Fatalf("targets = %#v, want no runnable default targets", targets)
	}
	if logText := buf.String(); strings.Contains(logText, "serve targets default fallback") {
		t.Fatalf("expected no helper-level fallback warning before bootstrap decision, got %q", logText)
	}
}
