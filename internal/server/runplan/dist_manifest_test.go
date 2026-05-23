// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runplan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/distmanifest"
)

func TestLoadDistManifest_MissingFile_ReturnsNil(t *testing.T) {
	distRoot := t.TempDir()

	manifest, err := LoadDistManifest(distRoot)
	if err != nil {
		t.Fatalf("LoadDistManifest: %v", err)
	}
	if manifest != nil {
		t.Fatalf("expected nil manifest when missing, got %#v", manifest)
	}
}

func TestLoadDistManifest_InvalidSchemaVersion_ReturnsError(t *testing.T) {
	distRoot := t.TempDir()
	path := filepath.Join(distRoot, distmanifest.DistManifestFileName)

	b, err := json.Marshal(map[string]any{"schemaVersion": 1, "compileBundleMode": "bundle"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := LoadDistManifest(distRoot); err == nil {
		t.Fatalf("expected error for unsupported schemaVersion")
	}
}

func TestLoadDistManifest_MissingCompileBundleMode_ReturnsError(t *testing.T) {
	distRoot := t.TempDir()
	path := filepath.Join(distRoot, distmanifest.DistManifestFileName)

	b, err := json.Marshal(map[string]any{"schemaVersion": distmanifest.SchemaVersion})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err = LoadDistManifest(distRoot)
	if err == nil {
		t.Fatal("expected error for missing compileBundleMode")
	}
	if !strings.Contains(err.Error(), "compileBundleMode is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadDistManifest_NormalizesFields(t *testing.T) {
	distRoot := t.TempDir()
	b, err := json.Marshal(distmanifest.DistManifestV2{
		SchemaVersion:     distmanifest.SchemaVersion,
		CompileBundleMode: " Application ",
		BackendTopoOrder:  []string{" auth ", "", "core", "auth"},
		Apps:              nil,
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, distmanifest.DistManifestFileName), b, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	manifest, err := LoadDistManifest(distRoot)
	if err != nil {
		t.Fatalf("LoadDistManifest() error = %v", err)
	}
	if manifest.CompileBundleMode != "application" {
		t.Fatalf("unexpected compile bundle mode: %q", manifest.CompileBundleMode)
	}
	if len(manifest.BackendTopoOrder) != 2 || manifest.BackendTopoOrder[0] != "auth" || manifest.BackendTopoOrder[1] != "core" {
		t.Fatalf("unexpected topo order: %#v", manifest.BackendTopoOrder)
	}
	if manifest.Apps == nil {
		t.Fatal("expected Apps map to be initialized")
	}

	got := normalizeStringList([]string{" auth ", "", "core", "auth", "core"})
	if len(got) != 2 || got[0] != "auth" || got[1] != "core" {
		t.Fatalf("unexpected normalized list: %#v", got)
	}
}

func TestOrderServeTargetsByTopo_OrdersAndAppendsUnknownAlphabetically(t *testing.T) {
	topo := []string{"base", "core", "auth"}
	targets := []string{"auth", "unknown_b", "core", "unknown_a", "auth"}

	ordered, unknown := orderServeTargetsByTopo(topo, targets)
	if want := []string{"unknown_a", "unknown_b"}; !reflect.DeepEqual(unknown, want) {
		t.Fatalf("unknown mismatch: want %v, got %v", want, unknown)
	}
	if want := []string{"core", "auth", "unknown_a", "unknown_b"}; !reflect.DeepEqual(ordered, want) {
		t.Fatalf("ordered mismatch: want %v, got %v", want, ordered)
	}
}

func TestComputeMissingAppDeps_UsesTransitiveClosure(t *testing.T) {
	manifest := &distmanifest.DistManifestV2{
		SchemaVersion: distmanifest.SchemaVersion,
		Apps: map[string]distmanifest.DistManifestApp{
			"auth": {Deps: distmanifest.DistManifestAppDeps{Apps: []string{"core"}}},
			"core": {Deps: distmanifest.DistManifestAppDeps{Apps: []string{"base"}}},
			"base": {Deps: distmanifest.DistManifestAppDeps{Apps: nil}},
		},
	}

	missing := computeMissingAppDeps(manifest, []string{"auth"})
	if want := []string{"base", "core"}; !reflect.DeepEqual(missing, want) {
		t.Fatalf("missing mismatch: want %v, got %v", want, missing)
	}

	missing = computeMissingAppDeps(manifest, []string{"auth", "core", "base"})
	if len(missing) != 0 {
		t.Fatalf("expected no missing deps, got %v", missing)
	}
}
