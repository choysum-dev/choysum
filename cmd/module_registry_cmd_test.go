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

func TestNewModuleCmd_SubcommandsAndWorkflow(t *testing.T) {
	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "auth", `{
		"name": "@choysum/module-auth",
		"version": "1.2.3",
		"description": "test module",
		"license": "Apache-2.0",
		"author": "test",
		"type": "module",
		"main": "index.ts",
		"choysum": {
			"moduleName": "auth",
			"application": "auth",
			"category": "test",
			"depends": [],
			"entryPoints": {"service": "./service/index.ts", "web": "./web/index.ts"}
		}
	}`)

	defaultChoysumPath := t.TempDir()
	cfg := &config.Config{
		ModulesPath:        modulesPath,
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
	if _, err := os.Stat(filepath.Join(modulesPath, "auth")); !os.IsNotExist(err) {
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
