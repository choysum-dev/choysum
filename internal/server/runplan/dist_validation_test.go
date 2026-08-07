// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runplan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
)

func TestValidateDistForTargets_BundleMode_BundlesDirMissing(t *testing.T) {
	distRoot := t.TempDir()
	err := ValidateDistForTargets("bundle", distRoot, []string{"auth"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "bundles dir missing") {
		t.Fatalf("expected bundles dir missing error, got: %v", err)
	}
}

func TestValidateDistForTargets_BundleMode_BundlesIndexMissing(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distRoot, "bundles"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	err := ValidateDistForTargets("bundle", distRoot, []string{"auth"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "bundles index missing") {
		t.Fatalf("expected bundles index missing error, got: %v", err)
	}
}

func TestValidateDistForTargets_WebOnly_RequiresBundlesAndProto(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distRoot, "web"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	err := ValidateDistForTargets("bundle", distRoot, []string{"web"})
	if err == nil || !strings.Contains(err.Error(), "bundles dir missing") {
		t.Fatalf("expected bundles dir missing for web-only, got %v", err)
	}

	if err := os.MkdirAll(filepath.Join(distRoot, "bundles"), 0o755); err != nil {
		t.Fatalf("mkdir bundles: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "bundles", "index.js"), []byte("export {}\n"), 0o644); err != nil {
		t.Fatalf("write bundles index: %v", err)
	}
	err = ValidateDistForTargets("bundle", distRoot, []string{"web"})
	if err == nil || !strings.Contains(err.Error(), "api proto assets missing") {
		t.Fatalf("expected api proto missing for web-only, got %v", err)
	}

	webProto := config.APIAppProtoDir(distRoot, "web")
	if err := os.MkdirAll(webProto, 0o755); err != nil {
		t.Fatalf("mkdir web proto: %v", err)
	}
	if err := ValidateDistForTargets("bundle", distRoot, []string{"web"}); err != nil {
		t.Fatalf("expected nil error with web+bundles+proto, got %v", err)
	}
}

func TestValidateDistForTargets_BundleMode_BundlesAssetsMissingForTarget(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distRoot, "bundles"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "bundles", "index.js"), []byte("// bundles"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := ValidateDistForTargets("bundle", distRoot, []string{"auth"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "api proto assets missing") {
		t.Fatalf("expected api proto assets missing error, got: %v", err)
	}
}

func TestValidateDistForTargets_DefaultBundleMode_SucceedsWithBackendAndWeb(t *testing.T) {
	distRoot := t.TempDir()
	for _, dir := range []string{
		filepath.Join(distRoot, "bundles"),
		config.APIAppProtoDir(distRoot, "auth"),
		config.APIAppProtoDir(distRoot, "web"),
		filepath.Join(distRoot, "web"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(distRoot, "bundles", "index.js"), []byte("// bundles"), 0o644); err != nil {
		t.Fatalf("write bundles index: %v", err)
	}

	if err := ValidateDistForTargets(" ", distRoot, []string{" auth ", "web"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidateDistForTargets_ApplicationMode_AppIndexMissing(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distRoot, "apps", "auth", "assets"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "apps", "auth", "assets", "a.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := ValidateDistForTargets("application", distRoot, []string{"auth"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "app index missing") {
		t.Fatalf("expected app index missing error, got: %v", err)
	}
}

func TestValidateDistForTargets_ApplicationMode_AppAssetsMissingOrEmpty(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distRoot, "apps", "auth"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "apps", "auth", "index.js"), []byte("// app"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	err := ValidateDistForTargets("application", distRoot, []string{"auth"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "app proto assets missing") {
		t.Fatalf("expected app proto assets missing error, got: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(distRoot, "apps", "auth", "assets"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	err = ValidateDistForTargets("application", distRoot, []string{"auth"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "app proto assets missing") {
		t.Fatalf("expected app proto assets missing error, got: %v", err)
	}
}

func TestValidateDistForTargets_ApplicationMode_SucceedsWithAssetsAndWeb(t *testing.T) {
	distRoot := t.TempDir()
	for _, dir := range []string{
		filepath.Join(distRoot, "apps", "auth", "assets"),
		filepath.Join(distRoot, "apps", "base", "assets"),
		filepath.Join(distRoot, "apps", "web"),
		filepath.Join(distRoot, "web"),
		config.APIAppProtoDir(distRoot, "web"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(distRoot, "apps", "auth", "index.js"), []byte("// auth"), 0o644); err != nil {
		t.Fatalf("write auth index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "apps", "base", "index.js"), []byte("// base"), 0o644); err != nil {
		t.Fatalf("write base index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "apps", "web", "index.js"), []byte("// web"), 0o644); err != nil {
		t.Fatalf("write web index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "apps", "auth", "assets", "a.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatalf("write auth asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "apps", "base", "assets", "b.proto"), []byte("syntax = \"proto3\";"), 0o644); err != nil {
		t.Fatalf("write base asset: %v", err)
	}

	if err := ValidateDistForTargets("application", distRoot, []string{"base", "web", "auth"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidateDistForTargets_InvalidBundleMode(t *testing.T) {
	err := ValidateDistForTargets("broken", t.TempDir(), []string{"auth"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid compile.bundleMode") {
		t.Fatalf("expected invalid compile.bundleMode error, got: %v", err)
	}
}

func TestValidateDistForTargets_WebMissingWhenRequested(t *testing.T) {
	distRoot := t.TempDir()
	err := ValidateDistForTargets("bundle", distRoot, []string{"web"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "web dist missing") {
		t.Fatalf("expected web dist missing error, got: %v", err)
	}
}

func TestResolveDefaultTargetsFromDist_BundleMode_EnumeratesAssetsAndWeb(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(config.APIAppProtoDir(distRoot, "auth"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(distRoot, "bundles"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distRoot, "bundles", "index.js"), []byte("// bundles"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(distRoot, "web"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	targets, err := resolveDefaultTargetsFromDist("bundle", distRoot)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(targets) != 2 || targets[0] != "auth" || targets[1] != "web" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}

func TestResolveDefaultTargetsFromDist_BundleMode_WebOnlyWhenNoAssets(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(distRoot, "web"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	targets, err := resolveDefaultTargetsFromDist("bundle", distRoot)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(targets) != 1 || targets[0] != "web" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}

func TestResolveDefaultTargetsFromDist_ApplicationMode_EnumeratesAppsAndWeb(t *testing.T) {
	distRoot := t.TempDir()
	for _, dir := range []string{
		filepath.Join(distRoot, "apps", "auth"),
		filepath.Join(distRoot, "apps", "base"),
		filepath.Join(distRoot, "web"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(distRoot, "apps", "README.txt"), []byte("ignore me"), 0o644); err != nil {
		t.Fatalf("write non-dir entry: %v", err)
	}

	targets, err := resolveDefaultTargetsFromDist("application", distRoot)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(targets) != 3 || targets[0] != "auth" || targets[1] != "base" || targets[2] != "web" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}

func TestResolveDefaultTargetsFromDist_ApplicationMode_ReadDirError(t *testing.T) {
	distRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(distRoot, "apps"), []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write apps path: %v", err)
	}

	_, err := resolveDefaultTargetsFromDist("application", distRoot)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "read apps dist dir") {
		t.Fatalf("expected read apps dist dir error, got: %v", err)
	}
}

func TestResolveDefaultTargetsFromDist_BundleMode_ReadDirError(t *testing.T) {
	distRoot := t.TempDir()
	apiRoot := config.APIRootFromDist(distRoot)
	if err := os.MkdirAll(filepath.Dir(apiRoot), 0o755); err != nil {
		t.Fatalf("mkdir api parent dir: %v", err)
	}
	if err := os.WriteFile(apiRoot, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write api root path: %v", err)
	}

	_, err := resolveDefaultTargetsFromDist("bundle", distRoot)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "read api root dir") {
		t.Fatalf("expected read api root dir error, got: %v", err)
	}
}

func TestResolveDefaultTargetsFromDist_InvalidBundleMode(t *testing.T) {
	_, err := resolveDefaultTargetsFromDist("broken", t.TempDir())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid compile.bundleMode") {
		t.Fatalf("expected invalid compile.bundleMode error, got: %v", err)
	}
}
