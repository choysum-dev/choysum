// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
)

func TestResolveI18nModulesExplicitArgs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "auth"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "auth", "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveI18nModules(root, false, []string{"auth"})
	if err != nil || len(got) != 1 || got[0] != "auth" {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if _, err := resolveI18nModules(root, false, []string{"../evil"}); err == nil {
		t.Fatal("expected invalid module name")
	}
	if _, err := resolveI18nModules(root, false, []string{"missing"}); err == nil {
		t.Fatal("expected missing module")
	}
	if _, err := resolveI18nModules(root, false, nil); err == nil {
		t.Fatal("expected no modules specified")
	}
	if _, err := resolveI18nModules("", true, nil); err == nil {
		t.Fatal("expected empty modules path")
	}
}

func TestI18nCmdArgsValidation(t *testing.T) {
	root := newI18nCmd(func() scope.Scope { return nil })

	extract := findI18nSub(t, root, "extract")
	if err := extract.Args(extract, nil); err == nil || !strings.Contains(err.Error(), "provide module name") {
		t.Fatalf("extract no args: %v", err)
	}
	_ = extract.Flags().Set("all", "true")
	if err := extract.Args(extract, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "--all cannot be used") {
		t.Fatalf("extract --all+args: %v", err)
	}

	sync := findI18nSub(t, root, "sync")
	if err := sync.Args(sync, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "--lang is required") {
		t.Fatalf("sync no lang: %v", err)
	}
	_ = sync.Flags().Set("lang", "bad lang!")
	if err := sync.Args(sync, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "invalid lang") {
		t.Fatalf("sync bad lang: %v", err)
	}

	status := findI18nSub(t, root, "status")
	if err := status.Args(status, []string{"auth"}); err == nil || !strings.Contains(err.Error(), "--lang is required") {
		t.Fatalf("status no lang: %v", err)
	}
}

func TestI18nExtractSyncStatusRunE(t *testing.T) {
	modulesPath := t.TempDir()
	mod := filepath.Join(modulesPath, "demo")
	srcDir := filepath.Join(mod, "web")
	i18nDir := filepath.Join(mod, "i18n")
	for _, dir := range []string{srcDir, i18nDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(mod, "package.json"), []byte(`{"name":"demo"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "hi.ts"), []byte("export const label = _t('Hello');\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "tsconfig.json"), []byte(`{"compilerOptions":{"paths":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := newCommandTestConfig(modulesPath)
	env := &commandTestScope{cfg: cfg}
	root := newI18nCmd(func() scope.Scope { return env })
	root.SilenceErrors = true
	root.SilenceUsage = true

	var extractOut bytes.Buffer
	root.SetOut(&extractOut)
	root.SetErr(&extractOut)
	root.SetArgs([]string{"extract", "demo"})
	if err := root.Execute(); err != nil {
		t.Fatalf("extract: %v\n%s", err, extractOut.String())
	}
	if !strings.Contains(extractOut.String(), "wrote") {
		t.Fatalf("extract output: %s", extractOut.String())
	}
	if _, err := os.Stat(filepath.Join(i18nDir, "demo.pot")); err != nil {
		t.Fatalf("expected pot: %v", err)
	}

	var syncOut bytes.Buffer
	root = newI18nCmd(func() scope.Scope { return env })
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetOut(&syncOut)
	root.SetErr(&syncOut)
	root.SetArgs([]string{"sync", "--lang", "zh_CN", "demo"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync: %v\n%s", err, syncOut.String())
	}
	if !strings.Contains(syncOut.String(), "synced") {
		t.Fatalf("sync output: %s", syncOut.String())
	}

	poPath := filepath.Join(i18nDir, "zh_CN.po")
	raw, err := os.ReadFile(poPath)
	if err != nil {
		t.Fatal(err)
	}
	filled := strings.Replace(string(raw), "msgid \"Hello\"\nmsgstr \"\"", "msgid \"Hello\"\nmsgstr \"你好\"", 1)
	if err := os.WriteFile(poPath, []byte(filled), 0o644); err != nil {
		t.Fatal(err)
	}

	var statusOut bytes.Buffer
	root = newI18nCmd(func() scope.Scope { return env })
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetOut(&statusOut)
	root.SetErr(&statusOut)
	root.SetArgs([]string{"status", "--lang", "zh_CN", "--json", "--skip-pot-check", "demo"})
	if err := root.Execute(); err != nil {
		t.Fatalf("status: %v\n%s", err, statusOut.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(statusOut.Bytes(), &payload); err != nil {
		t.Fatalf("status json: %v body=%s", err, statusOut.String())
	}
}

func TestI18nCmdRunEErrorPathsAndStrictOrphan(t *testing.T) {
	root := newI18nCmd(func() scope.Scope { return nil })
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{"extract", "demo"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "scope is not initialized") {
		t.Fatalf("nil scope: %v", err)
	}

	emptyCfg := newCommandTestConfig(t.TempDir())
	emptyCfg.ModulesPath = ""
	env := &commandTestScope{cfg: emptyCfg}
	root = newI18nCmd(func() scope.Scope { return env })
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{"extract", "demo"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "modules path is empty") {
		t.Fatalf("empty modules path: %v", err)
	}

	modulesPath := t.TempDir()
	mod := filepath.Join(modulesPath, "demo")
	i18nDir := filepath.Join(mod, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mod, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "tsconfig.json"), []byte(`{"compilerOptions":{"paths":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "demo.pot"), []byte(`msgid ""
msgstr ""

msgctxt "a@t"
msgid "Hello"
msgstr ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`msgid ""
msgstr ""

#~ msgctxt "a@t"
#~ msgid "Hello"
#~ msgstr "你好"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := newCommandTestConfig(modulesPath)
	env = &commandTestScope{cfg: cfg}
	root = newI18nCmd(func() scope.Scope { return env })
	root.SilenceErrors = true
	root.SilenceUsage = true
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"status", "--lang", "zh_CN", "--skip-pot-check", "--strict-orphan", "demo"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "blocking issue") {
		t.Fatalf("strict orphan: %v\n%s", err, out.String())
	}

	// --all discovery
	root = newI18nCmd(func() scope.Scope { return env })
	root.SilenceErrors = true
	root.SilenceUsage = true
	out.Reset()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"extract", "--all"})
	if err := root.Execute(); err != nil {
		t.Fatalf("extract --all: %v\n%s", err, out.String())
	}
}

func TestI18nExtractWarnsNonLiteral(t *testing.T) {
	modulesPath := t.TempDir()
	mod := filepath.Join(modulesPath, "demo")
	srcDir := filepath.Join(mod, "web")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mod, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "tsconfig.json"), []byte(`{"compilerOptions":{"paths":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "hi.ts"), []byte("_t(dynamic)\n_t('Ok')\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := newCommandTestConfig(modulesPath)
	env := &commandTestScope{cfg: cfg}
	root := newI18nCmd(func() scope.Scope { return env })
	root.SilenceErrors = true
	root.SilenceUsage = true
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"extract", "demo"})
	if err := root.Execute(); err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !strings.Contains(out.String(), "warn[") || !strings.Contains(out.String(), "warning(s)") {
		t.Fatalf("expected warnings in output: %s", out.String())
	}
}

func findI18nSub(t *testing.T, root *cobra.Command, name string) *cobra.Command {
	t.Helper()
	for _, c := range root.Commands() {
		if c.Name() == name {
			return c
		}
	}
	t.Fatalf("missing i18n subcommand %q", name)
	return nil
}
