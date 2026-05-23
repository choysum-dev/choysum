// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runplan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/distmanifest"
)

func TestBuildRunDecision_DefaultNoTargetsFallsBackToBootstrap(t *testing.T) {
	distRoot := t.TempDir()

	decision, err := buildRunDecision(distRoot, "bundle", nil, nil, nil)
	if err != nil {
		t.Fatalf("buildRunDecision() error = %v", err)
	}
	if decision.RunMode != RunModeBootstrap {
		t.Fatalf("buildRunDecision() run mode = %q, want %q", decision.RunMode, RunModeBootstrap)
	}
	if decision.Reason != "no app is ready to serve yet" {
		t.Fatalf("buildRunDecision() reason = %q, want %q", decision.Reason, "no app is ready to serve yet")
	}
}

func TestBuildRunDecision_ExplicitMissingWebKeepsFailFast(t *testing.T) {
	distRoot := t.TempDir()

	_, err := buildRunDecision(distRoot, "bundle", nil, nil, []string{"web"})
	if err == nil {
		t.Fatal("expected explicit web run to fail when web dist is missing")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "web dist missing") {
		t.Fatalf("buildRunDecision() error = %v, want web dist missing", err)
	}
}

func TestBuildRunDecision_DefaultBackendValidationFailureFallsBackToBootstrap(t *testing.T) {
	distRoot := t.TempDir()
	manifest := &distmanifest.DistManifestV2{
		SchemaVersion:     distmanifest.SchemaVersion,
		CompileBundleMode: "bundle",
		BackendTopoOrder:  []string{"auth"},
		Apps:              map[string]distmanifest.DistManifestApp{"auth": {}},
	}

	decision, err := buildRunDecision(distRoot, "bundle", nil, manifest, nil)
	if err != nil {
		t.Fatalf("buildRunDecision() error = %v", err)
	}
	if decision.RunMode != RunModeBootstrap {
		t.Fatalf("buildRunDecision() run mode = %q, want %q", decision.RunMode, RunModeBootstrap)
	}
	if decision.Reason != "required app assets are not ready yet" {
		t.Fatalf("buildRunDecision() reason = %q, want %q", decision.Reason, "required app assets are not ready yet")
	}
}

func TestBuildRunDecision_ExplicitBackendValidationFailureIsError(t *testing.T) {
	distRoot := t.TempDir()
	manifest := &distmanifest.DistManifestV2{
		SchemaVersion:     distmanifest.SchemaVersion,
		CompileBundleMode: "bundle",
		BackendTopoOrder:  []string{"auth"},
		Apps:              map[string]distmanifest.DistManifestApp{"auth": {}},
	}

	_, err := buildRunDecision(distRoot, "bundle", nil, manifest, []string{"auth"})
	if err == nil {
		t.Fatal("expected explicit backend run to fail when bundles are missing")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "bundles dir missing") {
		t.Fatalf("buildRunDecision() error = %v, want bundles dir missing", err)
	}
}

func TestBuildRunDecision_DefaultWebOnlyCanRunApplication(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distRoot, "web"), 0o755); err != nil {
		t.Fatalf("MkdirAll(web) error = %v", err)
	}

	decision, err := buildRunDecision(distRoot, "bundle", nil, nil, nil)
	if err != nil {
		t.Fatalf("buildRunDecision() error = %v", err)
	}
	if decision.RunMode != RunModeApplication {
		t.Fatalf("buildRunDecision() run mode = %q, want %q", decision.RunMode, RunModeApplication)
	}
	if len(decision.ServeTargets) != 1 || decision.ServeTargets[0] != "web" {
		t.Fatalf("buildRunDecision() targets = %#v, want [web]", decision.ServeTargets)
	}
}
