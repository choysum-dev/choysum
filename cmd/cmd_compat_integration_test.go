// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	clicompat "github.com/choysum-dev/choysum/internal/cli/compat"
	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newCLICompatTestRuntimeOptions(t *testing.T, indexURL string) cliruntime.Options {
	t.Helper()

	workspaceRoot := t.TempDir()
	runtimeOptions := cliruntime.Options{
		DefaultChoysumPath:    filepath.Join(workspaceRoot, ".choysum"),
		ModulesPath:           filepath.Join(workspaceRoot, "modules"),
		TmpPath:               filepath.Join(workspaceRoot, "tmp"),
		ModuleCatalogIndexURL: indexURL,
	}
	if err := runtimeOptions.Validate(); err != nil {
		t.Fatalf("runtimeOptions.Validate() error = %v", err)
	}
	return runtimeOptions
}

func newCLICompatTestScope(runtimeOptions cliruntime.Options) *commandTestScope {
	return &commandTestScope{
		ctx: context.Background(),
		cfg: &config.Config{
			DefaultChoysumPath: runtimeOptions.DefaultChoysumPath,
			ModulesPath:        runtimeOptions.ModulesPath,
			TmpPath:            runtimeOptions.TmpPath,
			ConfigPath:         filepath.Join(filepath.Dir(runtimeOptions.ModulesPath), "config.yaml"),
		},
	}
}

func resolveCLICompatFromCommand(cmd *cobra.Command, flagValue string) (clicompat.ResolvedCLICompatVersion, error) {
	runtimeVersion := ""
	if cmd != nil && cmd.Root() != nil {
		runtimeVersion = strings.TrimSpace(cmd.Root().Version)
	}
	return clicompat.ResolveCLICompatVersion(flagValue, runtimeVersion, strings.TrimSpace(os.Getenv(clicompat.CLICompatVersionEnv)))
}

func resolveRegistryBackedUpgradeInputForTest(ctx context.Context, runtimeScope *commandTestScope, runtimeOptions cliruntime.Options, moduleName, cliVersion string) (string, bool, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return "", false, xfmt.Errorf("module name is empty")
	}
	if err := runtimeOptions.Validate(); err != nil {
		return "", false, err
	}

	registryBacked, err := clicompat.HasRegistryOriginBinding(runtimeScope, runtimeOptions.DefaultChoysumPath, moduleName)
	if err != nil {
		return "", false, err
	}
	if !registryBacked {
		return moduleName, false, nil
	}

	indexURL, err := resolveModuleCatalogIndexURL(runtimeOptions)
	if err != nil {
		return "", false, err
	}
	compatibleVersion, err := clicompat.ResolveCompatibleRegistryLatestVersion(ctx, runtimeScope, indexURL, moduleName, cliVersion)
	if err != nil {
		return "", true, err
	}
	return moduleName + "@" + compatibleVersion, true, nil
}

func TestResolveCLICompatFromCommand(t *testing.T) {
	t.Run("reads root command version", func(t *testing.T) {
		t.Setenv(clicompat.CLICompatVersionEnv, "")

		root := &cobra.Command{Use: "root", Version: "v1.7.0"}
		child := &cobra.Command{Use: "child"}
		root.AddCommand(child)

		resolved, err := resolveCLICompatFromCommand(child, "")
		if err != nil {
			t.Fatalf("resolveCLICompatFromCommand() error = %v", err)
		}
		if resolved.Version != "v1.7.0" || resolved.Source != "runtime" {
			t.Fatalf("resolveCLICompatFromCommand() = %#v, want runtime v1.7.0", resolved)
		}
	})

	t.Run("invalid flag still fails", func(t *testing.T) {
		cmd := &cobra.Command{Use: "root", Version: "v1.7.0"}
		if _, err := resolveCLICompatFromCommand(cmd, "bad"); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("resolveCLICompatFromCommand(invalid flag) error = %v, want invalid-version error", err)
		}
	})

	t.Run("resolve from nil command", func(t *testing.T) {
		t.Setenv(clicompat.CLICompatVersionEnv, "")
		resolved, err := resolveCLICompatFromCommand(nil, "")
		if err != nil {
			t.Fatalf("resolveCLICompatFromCommand(nil) error = %v", err)
		}
		if resolved != (clicompat.ResolvedCLICompatVersion{}) {
			t.Fatalf("resolveCLICompatFromCommand(nil) = %#v, want empty", resolved)
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
	indexURL, err := resolveModuleCatalogIndexURL(runtimeOptions)
	if err != nil {
		t.Fatalf("resolveModuleCatalogIndexURL() error = %v", err)
	}

	version, err := clicompat.ResolveCompatibleRegistryLatestVersion(context.Background(), runtimeScope, indexURL, "demo", "v1.6.0")
	if err != nil {
		t.Fatalf("ResolveCompatibleRegistryLatestVersion() error = %v", err)
	}
	if version != "v1.5.0" {
		t.Fatalf("ResolveCompatibleRegistryLatestVersion() = %q, want %q", version, "v1.5.0")
	}

	if _, err := clicompat.ResolveCompatibleRegistryLatestVersion(context.Background(), runtimeScope, indexURL, "missing", "v1.6.0"); err == nil || !strings.Contains(err.Error(), "query remote module info failed") {
		t.Fatalf("ResolveCompatibleRegistryLatestVersion(missing) error = %v, want wrapped query error", err)
	}

	invalidOptions := runtimeOptions
	invalidOptions.ModuleCatalogIndexURL = "https://index.acme.dev/v1/catalog.json"
	if _, err := resolveModuleCatalogIndexURL(invalidOptions); err == nil || !strings.Contains(err.Error(), "index.json") {
		t.Fatalf("resolveModuleCatalogIndexURL(invalid url) error = %v, want invalid index url", err)
	}

	if _, err := clicompat.ResolveCompatibleRegistryLatestVersion(context.Background(), runtimeScope, indexURL, "demo", "bad"); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
		t.Fatalf("ResolveCompatibleRegistryLatestVersion(invalid cli version) error = %v, want invalid-version error", err)
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
	store := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(runtimeOptions.DefaultChoysumPath))

	if _, err := clicompat.HasRegistryOriginBinding(runtimeScope, runtimeOptions.DefaultChoysumPath, ""); err == nil || !strings.Contains(err.Error(), "module name is empty") {
		t.Fatalf("HasRegistryOriginBinding(empty) error = %v, want empty module error", err)
	}

	if _, err := clicompat.HasRegistryOriginBinding(runtimeScope, "", "demo"); err == nil || !strings.Contains(err.Error(), "defaultChoysumPath is required") {
		t.Fatalf("HasRegistryOriginBinding(invalid options) error = %v, want validation error", err)
	}

	registryBacked, err := clicompat.HasRegistryOriginBinding(runtimeScope, runtimeOptions.DefaultChoysumPath, "demo")
	if err != nil {
		t.Fatalf("HasRegistryOriginBinding(no binding) error = %v", err)
	}
	if registryBacked {
		t.Fatal("HasRegistryOriginBinding(no binding) = true, want false")
	}

	if err := store.UpsertBinding(workspaceRoot, internalorigin.Binding{ModuleName: "demo", OriginType: internalorigin.OriginTypeLocal, OriginRef: "demo", LocalPath: filepath.Join(runtimeOptions.ModulesPath, "demo")}); err != nil {
		t.Fatalf("UpsertBinding(local) error = %v", err)
	}
	registryBacked, err = clicompat.HasRegistryOriginBinding(runtimeScope, runtimeOptions.DefaultChoysumPath, "demo")
	if err != nil {
		t.Fatalf("HasRegistryOriginBinding(local) error = %v", err)
	}
	if registryBacked {
		t.Fatal("HasRegistryOriginBinding(local) = true, want false")
	}

	if err := store.UpsertBinding(workspaceRoot, internalorigin.Binding{ModuleName: "demo", OriginType: internalorigin.OriginTypeRegistry, OriginRef: "demo@latest", ResolvedVersion: "v2.0.0", LocalPath: filepath.Join(runtimeOptions.ModulesPath, "demo")}); err != nil {
		t.Fatalf("UpsertBinding(registry) error = %v", err)
	}
	registryBacked, err = clicompat.HasRegistryOriginBinding(runtimeScope, runtimeOptions.DefaultChoysumPath, "demo")
	if err != nil {
		t.Fatalf("HasRegistryOriginBinding(registry) error = %v", err)
	}
	if !registryBacked {
		t.Fatal("HasRegistryOriginBinding(registry) = false, want true")
	}

	resolvedInput, backed, err := resolveRegistryBackedUpgradeInputForTest(context.Background(), runtimeScope, runtimeOptions, "demo", "v1.6.0")
	if err != nil {
		t.Fatalf("resolveRegistryBackedUpgradeInputForTest() error = %v", err)
	}
	if !backed || resolvedInput != "demo@v1.5.0" {
		t.Fatalf("resolveRegistryBackedUpgradeInputForTest() = (%q,%v), want (%q,true)", resolvedInput, backed, "demo@v1.5.0")
	}

	if _, backed, err := resolveRegistryBackedUpgradeInputForTest(context.Background(), runtimeScope, runtimeOptions, "demo", "v3.0.0"); err == nil || !backed || !strings.Contains(err.Error(), "ERR_MODULE_NO_COMPATIBLE_VERSION") {
		t.Fatalf("resolveRegistryBackedUpgradeInputForTest(no compatible) = (%v,%v), want backed with no-compatible error", err, backed)
	}

	nonRegistryInput, backed, err := resolveRegistryBackedUpgradeInputForTest(context.Background(), runtimeScope, runtimeOptions, "unknown", "v1.6.0")
	if err != nil {
		t.Fatalf("resolveRegistryBackedUpgradeInputForTest(non registry) error = %v", err)
	}
	if backed || nonRegistryInput != "unknown" {
		t.Fatalf("resolveRegistryBackedUpgradeInputForTest(non registry) = (%q,%v), want (%q,false)", nonRegistryInput, backed, "unknown")
	}

	if _, _, err := resolveRegistryBackedUpgradeInputForTest(context.Background(), runtimeScope, runtimeOptions, "", "v1.6.0"); err == nil || !strings.Contains(err.Error(), "module name is empty") {
		t.Fatalf("resolveRegistryBackedUpgradeInputForTest(empty) error = %v, want empty module error", err)
	}

	if _, _, err := resolveRegistryBackedUpgradeInputForTest(context.Background(), runtimeScope, cliruntime.Options{}, "demo", "v1.6.0"); err == nil || !strings.Contains(err.Error(), "defaultChoysumPath is required") {
		t.Fatalf("resolveRegistryBackedUpgradeInputForTest(invalid options) error = %v, want validation error", err)
	}
}
