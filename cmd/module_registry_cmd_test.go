// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	sourceregistry "github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
)

func executeCommandForTest(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestNewRegistryCmd_SubcommandsAndWorkflow(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	sharedCfg := newCommandTestConfig(t.TempDir())

	envGetter := func() scope.Scope {
		return &commandTestScope{cfg: sharedCfg}
	}
	runtimeOptionsGetter := func() cliRuntimeOptions {
		return newCliRuntimeOptionsFromScopeInputOptions(newScopeInputConfigOptions(snapshot.New(sharedCfg)))
	}

	registryCmd := newRegistryCmd(envGetter, runtimeOptionsGetter)
	output, err := executeCommandForTest(t, registryCmd, "add", "corp", "https://example.com/registry")
	if err != nil {
		t.Fatalf("registry add failed: %v", err)
	}
	if !strings.Contains(output, `Registry "corp" -> https://example.com/registry`) {
		t.Fatalf("unexpected add output: %q", output)
	}

	if _, err := executeCommandForTest(t, registryCmd, "login", "corp", "--auth-ref", "token://corp"); err != nil {
		t.Fatalf("registry login failed: %v", err)
	}

	output, err = executeCommandForTest(t, registryCmd, "list")
	if err != nil {
		t.Fatalf("registry list failed: %v", err)
	}
	if !strings.Contains(output, "corp") || !strings.Contains(output, "token://corp") {
		t.Fatalf("expected corp registry in list output, got %q", output)
	}
	if !strings.Contains(output, sourceregistry.DefaultRegistryAlias) {
		t.Fatalf("expected default registry alias in list output, got %q", output)
	}

	if _, err := executeCommandForTest(t, registryCmd, "remove", "corp"); err != nil {
		t.Fatalf("registry remove failed: %v", err)
	}
	store := sourceregistry.NewStore(sourceregistry.WithHomeDir(homeDir), sourceregistry.WithDefaultChoysumPath(filepath.Join(homeDir, ".choysum")))
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("load registry store after remove failed: %v", err)
	}
	if _, ok := cfg.Registries["corp"]; ok {
		t.Fatal("expected registry alias corp to be removed")
	}
}

func TestNewRegistryCmd_ValidationPaths(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	sharedCfg := newCommandTestConfig(t.TempDir())
	envGetter := func() scope.Scope {
		return &commandTestScope{cfg: sharedCfg}
	}
	runtimeOptionsGetter := func() cliRuntimeOptions {
		return newCliRuntimeOptionsFromScopeInputOptions(newScopeInputConfigOptions(snapshot.New(sharedCfg)))
	}
	registryCmd := newRegistryCmd(envGetter, runtimeOptionsGetter)

	if _, err := executeCommandForTest(t, registryCmd, "add", "bad/alias", "https://example.com/registry"); err == nil || !strings.Contains(err.Error(), "invalid registry alias") {
		t.Fatalf("expected invalid alias error, got %v", err)
	}

	if _, err := executeCommandForTest(t, registryCmd, "add", "corp", "ftp://example.com/registry"); err == nil || !strings.Contains(err.Error(), "only http/https are supported") {
		t.Fatalf("expected invalid url error, got %v", err)
	}

	if _, err := executeCommandForTest(t, registryCmd, "remove", sourceregistry.DefaultRegistryAlias); err == nil || !strings.Contains(err.Error(), "cannot remove default registry alias") {
		t.Fatalf("expected protected default alias error, got %v", err)
	}
}

func TestNewModuleCmd_SubcommandsAndWorkflow(t *testing.T) {
	workspaceRoot := t.TempDir()
	addonsPath := filepath.Join(workspaceRoot, "addons")
	if err := os.MkdirAll(addonsPath, 0o755); err != nil {
		t.Fatalf("create addons path: %v", err)
	}

	writeCommandManifest(t, addonsPath, "auth", `{
		"name": "auth",
		"description": "test module",
		"application": "auth",
		"category": "test",
		"depends": [],
		"externalDependencies": {"application": {}, "node_module": {}, "binary": {}},
		"entryPoints": {"service": "./service/index.ts", "web": "./web/index.ts"},
		"version": "v1.2.3",
		"license": "Apache 2.0",
		"author": "test"
	}`)

	defaultChoysumPath := t.TempDir()
	cfg := &config.Config{
		AddonsPath:         addonsPath,
		NpmPath:            filepath.Join(workspaceRoot, "node_modules"),
		TmpPath:            filepath.Join(defaultChoysumPath, "tmp"),
		ConfigPath:         filepath.Join(workspaceRoot, "config.yaml"),
		DefaultChoysumPath: defaultChoysumPath,
	}
	envGetter := func() scope.Scope {
		return &commandTestScope{ctx: context.Background(), cfg: cfg}
	}
	runtimeOptionsGetter := func() cliRuntimeOptions {
		return newCliRuntimeOptionsFromScopeInputOptions(newScopeInputConfigOptions(snapshot.New(cfg)))
	}

	moduleCmd := newModuleCmd(envGetter, runtimeOptionsGetter)
	output, err := executeCommandForTest(t, moduleCmd, "fetch", "auth")
	if err != nil {
		t.Fatalf("module fetch failed: %v", err)
	}
	if !strings.Contains(output, "Fetched module auth@v1.2.3") {
		t.Fatalf("unexpected fetch output: %q", output)
	}

	output, err = executeCommandForTest(t, moduleCmd, "info", "auth")
	if err != nil {
		t.Fatalf("module info failed: %v", err)
	}
	if !strings.Contains(output, `"module_name": "auth"`) || !strings.Contains(output, `"version": "v1.2.3"`) {
		t.Fatalf("unexpected info output: %q", output)
	}

	output, err = executeCommandForTest(t, moduleCmd, "list")
	if err != nil {
		t.Fatalf("module list failed: %v", err)
	}
	if !strings.Contains(output, "auth") || !strings.Contains(output, "local") {
		t.Fatalf("unexpected list output: %q", output)
	}

	output, err = executeCommandForTest(t, moduleCmd, "search", "au")
	if err != nil {
		t.Fatalf("module search failed: %v", err)
	}
	if !strings.Contains(output, "auth") {
		t.Fatalf("expected module search output to include auth, got %q", output)
	}

	if _, err := executeCommandForTest(t, moduleCmd, "purge", "auth"); err != nil {
		t.Fatalf("module purge failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(addonsPath, "auth")); !os.IsNotExist(err) {
		t.Fatalf("expected purged module directory to be removed, stat err=%v", err)
	}

	store := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(cfg.DefaultChoysumPath))
	lock, err := store.Read(workspaceRoot)
	if err != nil {
		t.Fatalf("read lock file after purge: %v", err)
	}
	if _, ok := lock.Modules["auth"]; ok {
		t.Fatal("expected auth binding to be removed after purge")
	}
}

func TestNewModuleCmd_RequiresRuntimeScope(t *testing.T) {
	moduleCmd := newModuleCmd(func() scope.Scope { return nil }, func() cliRuntimeOptions { return cliRuntimeOptions{} })
	if _, err := executeCommandForTest(t, moduleCmd, "fetch", "auth"); err == nil || !strings.Contains(err.Error(), "scope is not initialized") {
		t.Fatalf("expected environment initialization error, got %v", err)
	}
}
