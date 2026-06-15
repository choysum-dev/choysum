// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	sourceregistry "github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/spf13/cobra"
)

func newCLICompatTestRuntimeOptions(t *testing.T, indexURL string) cliRuntimeOptions {
	t.Helper()

	workspaceRoot := t.TempDir()
	runtimeOptions := cliRuntimeOptions{
		defaultChoysumPath:    filepath.Join(workspaceRoot, ".choysum"),
		modulesPath:           filepath.Join(workspaceRoot, "modules"),
		npmPath:               filepath.Join(workspaceRoot, "node_modules"),
		tmpPath:               filepath.Join(workspaceRoot, "tmp"),
		moduleCatalogIndexURL: indexURL,
	}
	if err := runtimeOptions.Validate(); err != nil {
		t.Fatalf("runtimeOptions.Validate() error = %v", err)
	}
	return runtimeOptions
}

func newCLICompatTestScope(runtimeOptions cliRuntimeOptions) *commandTestScope {
	return &commandTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			DefaultChoysumPath: runtimeOptions.defaultChoysumPath,
			ModulesPath:        runtimeOptions.modulesPath,
			NpmPath:            runtimeOptions.npmPath,
			TmpPath:            runtimeOptions.tmpPath,
			ConfigPath:         filepath.Join(filepath.Dir(runtimeOptions.modulesPath), "config.yaml"),
		},
	}
}

func TestNormalizeAndParseCLICompatVersion(t *testing.T) {
	t.Run("normalize", func(t *testing.T) {
		tests := []struct {
			name   string
			input  string
			want   string
			wantOK bool
		}{
			{name: "trim and add prefix", input: " 1.7.0 ", want: "v1.7.0", wantOK: true},
			{name: "keep prefixed", input: "v1.7.0", want: "v1.7.0", wantOK: true},
			{name: "invalid", input: "foo", want: "", wantOK: false},
			{name: "empty", input: "   ", want: "", wantOK: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, ok := normalizeCLICompatVersion(tt.input)
				if got != tt.want || ok != tt.wantOK {
					t.Fatalf("normalizeCLICompatVersion(%q) = (%q,%v), want (%q,%v)", tt.input, got, ok, tt.want, tt.wantOK)
				}
			})
		}
	})

	t.Run("parse rejects invalid", func(t *testing.T) {
		if _, err := parseCLICompatVersion("not-semver"); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("parseCLICompatVersion(invalid) error = %v, want invalid-version error", err)
		}
	})
}

func TestResolveCLICompatVersion(t *testing.T) {
	t.Run("flag takes precedence", func(t *testing.T) {
		t.Setenv(cliCompatVersionEnv, "v1.6.0")

		resolved, err := resolveCLICompatVersion("1.7.0", "v1.8.0")
		if err != nil {
			t.Fatalf("resolveCLICompatVersion() error = %v", err)
		}
		if resolved.Version != "v1.7.0" || resolved.Source != "flag" {
			t.Fatalf("resolveCLICompatVersion() = %#v, want flag v1.7.0", resolved)
		}
	})

	t.Run("env used when flag missing", func(t *testing.T) {
		t.Setenv(cliCompatVersionEnv, "1.6.0")

		resolved, err := resolveCLICompatVersion("", "v1.8.0")
		if err != nil {
			t.Fatalf("resolveCLICompatVersion() error = %v", err)
		}
		if resolved.Version != "v1.6.0" || resolved.Source != "env" {
			t.Fatalf("resolveCLICompatVersion() = %#v, want env v1.6.0", resolved)
		}
	})

	t.Run("runtime used when flag env missing", func(t *testing.T) {
		t.Setenv(cliCompatVersionEnv, "")

		resolved, err := resolveCLICompatVersion("", "1.8.0")
		if err != nil {
			t.Fatalf("resolveCLICompatVersion() error = %v", err)
		}
		if resolved.Version != "v1.8.0" || resolved.Source != "runtime" {
			t.Fatalf("resolveCLICompatVersion() = %#v, want runtime v1.8.0", resolved)
		}
	})

	t.Run("returns empty when unresolved", func(t *testing.T) {
		t.Setenv(cliCompatVersionEnv, "")

		resolved, err := resolveCLICompatVersion("", "invalid")
		if err != nil {
			t.Fatalf("resolveCLICompatVersion() error = %v", err)
		}
		if resolved != (resolvedCLICompatVersion{}) {
			t.Fatalf("resolveCLICompatVersion() = %#v, want empty", resolved)
		}
	})

	t.Run("invalid flag fails", func(t *testing.T) {
		if _, err := resolveCLICompatVersion("bad", "v1.8.0"); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("resolveCLICompatVersion(flag invalid) error = %v, want invalid-version error", err)
		}
	})

	t.Run("invalid env fails", func(t *testing.T) {
		t.Setenv(cliCompatVersionEnv, "bad")
		if _, err := resolveCLICompatVersion("", "v1.8.0"); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("resolveCLICompatVersion(env invalid) error = %v, want invalid-version error", err)
		}
	})
}

func TestResolveCLICompatVersionForCommand(t *testing.T) {
	t.Run("reads root command version", func(t *testing.T) {
		t.Setenv(cliCompatVersionEnv, "")

		root := &cobra.Command{Use: "root", Version: "v1.7.0"}
		child := &cobra.Command{Use: "child"}
		root.AddCommand(child)

		resolved, err := resolveCLICompatVersionForCommand(child, "")
		if err != nil {
			t.Fatalf("resolveCLICompatVersionForCommand() error = %v", err)
		}
		if resolved.Version != "v1.7.0" || resolved.Source != "runtime" {
			t.Fatalf("resolveCLICompatVersionForCommand() = %#v, want runtime v1.7.0", resolved)
		}
	})

	t.Run("invalid flag still fails", func(t *testing.T) {
		cmd := &cobra.Command{Use: "root", Version: "v1.7.0"}
		if _, err := resolveCLICompatVersionForCommand(cmd, "bad"); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("resolveCLICompatVersionForCommand(invalid flag) error = %v, want invalid-version error", err)
		}
	})
}

func TestCatalogVersionHelpers(t *testing.T) {
	versions := catalogCandidateVersions(&sourceregistry.CatalogModule{Versions: []string{" v1.0.0 ", "", "v1.2.0"}})
	if !reflect.DeepEqual(versions, []string{"v1.0.0", "v1.2.0"}) {
		t.Fatalf("catalogCandidateVersions() = %#v", versions)
	}

	latestOnly := catalogCandidateVersions(&sourceregistry.CatalogModule{LatestVersion: " v2.0.0 "})
	if !reflect.DeepEqual(latestOnly, []string{"v2.0.0"}) {
		t.Fatalf("catalogCandidateVersions(latest only) = %#v", latestOnly)
	}

	if !containsCatalogVersion([]string{"v1.2.3"}, "1.2.3") {
		t.Fatal("containsCatalogVersion() should match equivalent semver with or without v prefix")
	}
	if !containsCatalogVersion([]string{"snapshot"}, "snapshot") {
		t.Fatal("containsCatalogVersion() should match non-semver values by exact string")
	}
	if containsCatalogVersion([]string{"v1.2.3"}, "") {
		t.Fatal("containsCatalogVersion() should reject empty target")
	}
}

func TestCompatibleCatalogVersionsAndSelection(t *testing.T) {
	t.Run("nil module", func(t *testing.T) {
		if _, err := compatibleCatalogVersions(nil, "v1.0.0"); err == nil || !strings.Contains(err.Error(), "remote module is nil") {
			t.Fatalf("compatibleCatalogVersions(nil) error = %v, want nil-module error", err)
		}
	})

	t.Run("invalid cli version", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{Name: "demo", Versions: []string{"v1.0.0"}, VersionCLIRanges: map[string]string{"v1.0.0": ">=1.0.0 <2.0.0"}}
		if _, err := compatibleCatalogVersions(item, "bad"); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("compatibleCatalogVersions(invalid cli) error = %v, want invalid-version error", err)
		}
	})

	t.Run("missing range", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{Versions: []string{"v1.0.0"}}
		if _, err := compatibleCatalogVersions(item, "v1.0.0"); err == nil || !strings.Contains(err.Error(), "ERR_MODULE_CLI_RANGE_MISSING") {
			t.Fatalf("compatibleCatalogVersions(missing range) error = %v, want missing-range error", err)
		}
	})

	t.Run("invalid range", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{Name: "demo", Versions: []string{"v1.0.0"}, VersionCLIRanges: map[string]string{"v1.0.0": "not-a-range"}}
		if _, err := compatibleCatalogVersions(item, "v1.0.0"); err == nil || !strings.Contains(err.Error(), "ERR_MODULE_CLI_RANGE_INVALID") {
			t.Fatalf("compatibleCatalogVersions(invalid range) error = %v, want invalid-range error", err)
		}
	})

	t.Run("no compatible version", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{Name: "demo", Versions: []string{"v2.0.0"}, VersionCLIRanges: map[string]string{"v2.0.0": ">=2.0.0 <3.0.0"}}
		if _, err := compatibleCatalogVersions(item, "v1.0.0"); err == nil || !strings.Contains(err.Error(), "ERR_MODULE_NO_COMPATIBLE_VERSION") {
			t.Fatalf("compatibleCatalogVersions(no compatible) error = %v, want no-compatible error", err)
		}
	})

	t.Run("returns compatible versions and latest", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{
			Name:     "demo",
			Versions: []string{"v1.0.0", "v1.3.0", "v2.0.0"},
			VersionCLIRanges: map[string]string{
				"v1.0.0": ">=1.0.0 <2.0.0",
				"v1.3.0": ">=1.2.0 <2.0.0",
				"v2.0.0": ">=2.0.0 <3.0.0",
			},
		}

		compatible, err := compatibleCatalogVersions(item, "v1.5.0")
		if err != nil {
			t.Fatalf("compatibleCatalogVersions() error = %v", err)
		}
		if !reflect.DeepEqual(compatible, []string{"v1.0.0", "v1.3.0"}) {
			t.Fatalf("compatibleCatalogVersions() = %#v, want [v1.0.0 v1.3.0]", compatible)
		}

		latest, err := selectLatestCompatibleCatalogVersion(item, "v1.5.0")
		if err != nil {
			t.Fatalf("selectLatestCompatibleCatalogVersion() error = %v", err)
		}
		if latest != "v1.3.0" {
			t.Fatalf("selectLatestCompatibleCatalogVersion() = %q, want %q", latest, "v1.3.0")
		}
	})
}

func TestResolveCompatibleRegistryLatestVersion(t *testing.T) {
	catalogServer := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{
		{
			Name:          "demo",
			LatestVersion: "v2.0.0",
			Versions:      []string{"v1.0.0", "v1.5.0", "v2.0.0"},
			VersionCLIRanges: map[string]string{
				"v1.0.0": ">=1.0.0 <2.0.0",
				"v1.5.0": ">=1.4.0 <2.0.0",
				"v2.0.0": ">=2.0.0 <3.0.0",
			},
		},
	})
	defer catalogServer.Close()

	runtimeOptions := newCLICompatTestRuntimeOptions(t, catalogServer.URL+"/v1/index.json")
	runtimeScope := newCLICompatTestScope(runtimeOptions)

	version, err := resolveCompatibleRegistryLatestVersion(context.Background(), runtimeScope, runtimeOptions, "demo", "v1.6.0")
	if err != nil {
		t.Fatalf("resolveCompatibleRegistryLatestVersion() error = %v", err)
	}
	if version != "v1.5.0" {
		t.Fatalf("resolveCompatibleRegistryLatestVersion() = %q, want %q", version, "v1.5.0")
	}

	if _, err := resolveCompatibleRegistryLatestVersion(context.Background(), runtimeScope, runtimeOptions, "missing", "v1.6.0"); err == nil || !strings.Contains(err.Error(), "query remote module info failed") {
		t.Fatalf("resolveCompatibleRegistryLatestVersion(missing) error = %v, want wrapped query error", err)
	}

	invalidOptions := runtimeOptions
	invalidOptions.moduleCatalogIndexURL = "https://index.acme.dev/v1/catalog.json"
	if _, err := resolveCompatibleRegistryLatestVersion(context.Background(), runtimeScope, invalidOptions, "demo", "v1.6.0"); err == nil || !strings.Contains(err.Error(), "index.json") {
		t.Fatalf("resolveCompatibleRegistryLatestVersion(invalid url) error = %v, want invalid index url", err)
	}
}

func TestRegistryBindingHelpers(t *testing.T) {
	catalogServer := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{
		{
			Name:          "demo",
			LatestVersion: "v2.0.0",
			Versions:      []string{"v1.0.0", "v1.5.0", "v2.0.0"},
			VersionCLIRanges: map[string]string{
				"v1.0.0": ">=1.0.0 <2.0.0",
				"v1.5.0": ">=1.4.0 <2.0.0",
				"v2.0.0": ">=2.0.0 <3.0.0",
			},
		},
	})
	defer catalogServer.Close()

	runtimeOptions := newCLICompatTestRuntimeOptions(t, catalogServer.URL+"/v1/index.json")
	runtimeScope := newCLICompatTestScope(runtimeOptions)
	workspaceRoot := internalorigin.WorkspaceRoot(runtimeScope)
	store := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(runtimeOptions.defaultChoysumPath))

	if _, err := hasRegistryOriginBinding(runtimeScope, runtimeOptions, ""); err == nil || !strings.Contains(err.Error(), "module name is empty") {
		t.Fatalf("hasRegistryOriginBinding(empty) error = %v, want empty module error", err)
	}

	if _, err := hasRegistryOriginBinding(runtimeScope, cliRuntimeOptions{}, "demo"); err == nil || !strings.Contains(err.Error(), "defaultChoysumPath is required") {
		t.Fatalf("hasRegistryOriginBinding(invalid options) error = %v, want validation error", err)
	}

	registryBacked, err := hasRegistryOriginBinding(runtimeScope, runtimeOptions, "demo")
	if err != nil {
		t.Fatalf("hasRegistryOriginBinding(no binding) error = %v", err)
	}
	if registryBacked {
		t.Fatal("hasRegistryOriginBinding(no binding) = true, want false")
	}

	if err := store.UpsertBinding(workspaceRoot, internalorigin.Binding{ModuleName: "demo", OriginType: internalorigin.OriginTypeLocal, OriginRef: "demo", LocalPath: filepath.Join(runtimeOptions.modulesPath, "demo")}); err != nil {
		t.Fatalf("UpsertBinding(local) error = %v", err)
	}
	registryBacked, err = hasRegistryOriginBinding(runtimeScope, runtimeOptions, "demo")
	if err != nil {
		t.Fatalf("hasRegistryOriginBinding(local) error = %v", err)
	}
	if registryBacked {
		t.Fatal("hasRegistryOriginBinding(local) = true, want false")
	}

	if err := store.UpsertBinding(workspaceRoot, internalorigin.Binding{ModuleName: "demo", OriginType: internalorigin.OriginTypeRegistry, OriginRef: "demo@latest", ResolvedVersion: "v2.0.0", LocalPath: filepath.Join(runtimeOptions.modulesPath, "demo")}); err != nil {
		t.Fatalf("UpsertBinding(registry) error = %v", err)
	}
	registryBacked, err = hasRegistryOriginBinding(runtimeScope, runtimeOptions, "demo")
	if err != nil {
		t.Fatalf("hasRegistryOriginBinding(registry) error = %v", err)
	}
	if !registryBacked {
		t.Fatal("hasRegistryOriginBinding(registry) = false, want true")
	}

	resolvedInput, backed, err := resolveRegistryBackedUpgradeInput(context.Background(), runtimeScope, runtimeOptions, "demo", "v1.6.0")
	if err != nil {
		t.Fatalf("resolveRegistryBackedUpgradeInput() error = %v", err)
	}
	if !backed || resolvedInput != "demo@v1.5.0" {
		t.Fatalf("resolveRegistryBackedUpgradeInput() = (%q,%v), want (%q,true)", resolvedInput, backed, "demo@v1.5.0")
	}

	if _, backed, err := resolveRegistryBackedUpgradeInput(context.Background(), runtimeScope, runtimeOptions, "demo", "v3.0.0"); err == nil || !backed || !strings.Contains(err.Error(), "ERR_MODULE_NO_COMPATIBLE_VERSION") {
		t.Fatalf("resolveRegistryBackedUpgradeInput(no compatible) = (%v,%v), want backed with no-compatible error", err, backed)
	}

	nonRegistryInput, backed, err := resolveRegistryBackedUpgradeInput(context.Background(), runtimeScope, runtimeOptions, "unknown", "v1.6.0")
	if err != nil {
		t.Fatalf("resolveRegistryBackedUpgradeInput(non registry) error = %v", err)
	}
	if backed || nonRegistryInput != "unknown" {
		t.Fatalf("resolveRegistryBackedUpgradeInput(non registry) = (%q,%v), want (%q,false)", nonRegistryInput, backed, "unknown")
	}

	if _, _, err := resolveRegistryBackedUpgradeInput(context.Background(), runtimeScope, runtimeOptions, "", "v1.6.0"); err == nil || !strings.Contains(err.Error(), "module name is empty") {
		t.Fatalf("resolveRegistryBackedUpgradeInput(empty) error = %v, want empty module error", err)
	}

	if _, _, err := resolveRegistryBackedUpgradeInput(context.Background(), runtimeScope, cliRuntimeOptions{}, "demo", "v1.6.0"); err == nil || !strings.Contains(err.Error(), "defaultChoysumPath is required") {
		t.Fatalf("resolveRegistryBackedUpgradeInput(invalid options) error = %v, want validation error", err)
	}
}

func TestCLICompatFilterSkippedWarning(t *testing.T) {
	warning := cliCompatFilterSkippedWarning()
	if !strings.Contains(warning, "WARN_CLI_COMPAT_FILTER_SKIPPED") {
		t.Fatalf("cliCompatFilterSkippedWarning() = %q, want warning code", warning)
	}
}
