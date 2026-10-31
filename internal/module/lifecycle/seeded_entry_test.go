// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/evolution/hooks"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestBuildSeededModuleImportsSkipsEmptyEntry(t *testing.T) {
	t.Parallel()
	mods := []*meta.Module{
		{Name: "web", ApplicationStr: "web", ServiceEntryPoint: ""},
		{Name: "document", ApplicationStr: "document", ServiceEntryPoint: "service/index.ts"},
	}
	got := buildSeededModuleImports(mods, "/mods")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ModuleName != "document" || got[0].AppName != "document" {
		t.Fatalf("got %#v", got[0])
	}
	wantEntry := filepath.Join("/mods", "document", "service/index.ts")
	if got[0].EntryPath != wantEntry {
		t.Fatalf("EntryPath = %q, want %q", got[0].EntryPath, wantEntry)
	}
	if got[0].SeedFile != "__seed_mod_document.ts" {
		t.Fatalf("SeedFile = %q", got[0].SeedFile)
	}
}

func TestWriteSeededBundlesEntryOrdersDispatcherSeedThenService(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	imports := []seededModuleImport{{
		AppName:    "document",
		ModuleName: "document",
		EntryPath:  "/abs/document/service/index.ts",
		SeedFile:   "__seed_mod_document.ts",
	}}
	entryPath, err := writeSeededBundlesEntry(dir, "__choysum_bundles_entry.ts", imports)
	if err != nil {
		t.Fatalf("writeSeededBundlesEntry: %v", err)
	}
	raw, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	text := string(raw)
	dispIdx := strings.Index(text, `import "./__choysum_hook_dispatcher.ts"`)
	seedIdx := strings.Index(text, `import "./__seed_mod_document.ts"`)
	svcIdx := strings.Index(text, `import "/abs/document/service/index.ts"`)
	if dispIdx < 0 || seedIdx < 0 || svcIdx < 0 {
		t.Fatalf("missing imports in entry:\n%s", text)
	}
	if !(dispIdx < seedIdx && seedIdx < svcIdx) {
		t.Fatalf("import order wrong: disp=%d seed=%d svc=%d\n%s", dispIdx, seedIdx, svcIdx, text)
	}

	seedRaw, err := os.ReadFile(filepath.Join(dir, "__seed_mod_document.ts"))
	if err != nil {
		t.Fatalf("read seed: %v", err)
	}
	seedText := string(seedRaw)
	if !strings.Contains(seedText, `globalThis.CHOYSUM_APP_NAME = "document"`) {
		t.Fatalf("seed missing APP_NAME:\n%s", seedText)
	}
	if !strings.Contains(seedText, `globalThis.CHOYSUM_MODULE_NAME = "document"`) {
		t.Fatalf("seed missing MODULE_NAME:\n%s", seedText)
	}

	dispRaw, err := os.ReadFile(filepath.Join(dir, "__choysum_hook_dispatcher.ts"))
	if err != nil {
		t.Fatalf("read dispatcher: %v", err)
	}
	if !strings.Contains(string(dispRaw), hooks.HookDispatcherScriptContent) {
		t.Fatal("dispatcher file missing HookDispatcherScriptContent")
	}
}

func TestWriteSeededBundlesEntryEmptyImportsStillEmitsDispatcher(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	entryPath, err := writeSeededBundlesEntry(dir, "__choysum_app_entry.ts", nil)
	if err != nil {
		t.Fatalf("writeSeededBundlesEntry: %v", err)
	}
	raw, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, `import "./__choysum_hook_dispatcher.ts"`) {
		t.Fatalf("expected dispatcher import, got %q", text)
	}
	if !strings.Contains(text, "export {};") {
		t.Fatalf("expected stub export, got %q", text)
	}
}

func TestSanitizeSeedFileToken(t *testing.T) {
	t.Parallel()
	if got := sanitizeSeedFileToken(""); got != "mod" {
		t.Fatalf("empty -> %q", got)
	}
	if got := sanitizeSeedFileToken("  "); got != "mod" {
		t.Fatalf("whitespace -> %q", got)
	}
	if got := sanitizeSeedFileToken("doc@v2"); got != "doc_v2" {
		t.Fatalf("special chars -> %q", got)
	}
	if got := sanitizeSeedFileToken("my-module"); got != "my-module" {
		t.Fatalf("hyphen preserved -> %q", got)
	}
}

func TestBuildSeededModuleImportsEdgeCases(t *testing.T) {
	t.Parallel()
	absEntry := filepath.Join(t.TempDir(), "service", "index.ts")
	mods := []*meta.Module{
		nil,
		{Name: "web", ServiceEntryPoint: absEntry, ApplicationStr: ""},
		{Name: "web", ServiceEntryPoint: "service/other.ts"},
		{Name: "same@name", ServiceEntryPoint: "service/a.ts"},
		{Name: "same@name", ServiceEntryPoint: "service/b.ts"},
	}
	got := buildSeededModuleImports(mods, "/mods")
	if len(got) != 4 {
		t.Fatalf("len=%d want 4", len(got))
	}
	if got[0].EntryPath != filepath.Clean(absEntry) {
		t.Fatalf("abs entry = %q", got[0].EntryPath)
	}
	if got[0].AppName != "web" {
		t.Fatalf("empty ApplicationStr defaults to module name, got %q", got[0].AppName)
	}
	if got[0].SeedFile != "__seed_mod_web.ts" {
		t.Fatalf("seed file = %q", got[0].SeedFile)
	}
	if got[3].SeedFile != "__seed_mod_same_name_2.ts" {
		t.Fatalf("duplicate token suffix = %q", got[3].SeedFile)
	}
}

func TestWriteSeededBundlesEntrySeedWriteFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "__seed_mod_document.ts"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := writeSeededBundlesEntry(dir, "__entry.ts", []seededModuleImport{{
		AppName: "document", ModuleName: "document", EntryPath: "/x", SeedFile: "__seed_mod_document.ts",
	}})
	if err == nil {
		t.Fatal("expected seed write failure when seed path is a file")
	}
}

func TestWriteSeededBundlesEntryFinalWriteFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	entryName := "__choysum_bundles_entry.ts"
	if err := os.Mkdir(filepath.Join(dir, entryName), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := writeSeededBundlesEntry(dir, entryName, nil)
	if err == nil {
		t.Fatal("expected entry write failure when entry path is a file")
	}
}

func TestWriteSeededBundlesEntryWriteFailure(t *testing.T) {
	t.Parallel()
	badDir := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(badDir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := writeSeededBundlesEntry(badDir, "__entry.ts", []seededModuleImport{{
		AppName: "a", ModuleName: "a", EntryPath: "/x", SeedFile: "__seed_mod_a.ts",
	}})
	if err == nil {
		t.Fatal("expected write error for bad entry dir")
	}
}
