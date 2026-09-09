// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
)

func TestWriteE2EEntry(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "entry.js")
	spec := filepath.Join(dir, "a.spec.ts")
	if err := os.WriteFile(spec, []byte("export {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteE2EEntry(entry, []string{spec}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(entry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "import ") || !strings.Contains(string(raw), "a.spec.ts") {
		t.Fatalf("unexpected entry: %s", raw)
	}
}

func TestWriteE2EEntryErrors(t *testing.T) {
	if err := WriteE2EEntry("", nil); err == nil || !strings.Contains(err.Error(), "empty entry path") {
		t.Fatalf("empty path: %v", err)
	}

	oldMkdir := e2eOsMkdirAll
	e2eOsMkdirAll = func(path string, perm os.FileMode) error { return errors.New("mkdir fail") }
	defer func() { e2eOsMkdirAll = oldMkdir }()
	if err := WriteE2EEntry(filepath.Join(t.TempDir(), "e.js"), nil); err == nil || !strings.Contains(err.Error(), "mkdir entry") {
		t.Fatalf("mkdir: %v", err)
	}
	e2eOsMkdirAll = oldMkdir

	oldAbs := e2eFilepathAbs
	e2eFilepathAbs = func(path string) (string, error) { return "", errors.New("abs fail") }
	defer func() { e2eFilepathAbs = oldAbs }()
	if err := WriteE2EEntry(filepath.Join(t.TempDir(), "e.js"), []string{"x.ts"}); err == nil || !strings.Contains(err.Error(), "resolve spec path") {
		t.Fatalf("abs: %v", err)
	}
	e2eFilepathAbs = oldAbs

	oldMarshal := e2eJSONMarshal
	e2eJSONMarshal = func(v any) ([]byte, error) { return nil, errors.New("marshal fail") }
	defer func() { e2eJSONMarshal = oldMarshal }()
	if err := WriteE2EEntry(filepath.Join(t.TempDir(), "e.js"), []string{filepath.Join(t.TempDir(), "a.ts")}); err == nil || !strings.Contains(err.Error(), "encode import path") {
		t.Fatalf("marshal: %v", err)
	}
	e2eJSONMarshal = oldMarshal

	oldWrite := e2eOsWriteFile
	e2eOsWriteFile = func(name string, data []byte, perm os.FileMode) error { return errors.New("write fail") }
	defer func() { e2eOsWriteFile = oldWrite }()
	if err := WriteE2EEntry(filepath.Join(t.TempDir(), "e.js"), []string{filepath.Join(t.TempDir(), "a.ts")}); err == nil || !strings.Contains(err.Error(), "write entry") {
		t.Fatalf("write: %v", err)
	}
}

func TestBuildE2EBundleAlias(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	dir := t.TempDir()
	spec := filepath.Join(dir, "smoke.spec.ts")
	if err := os.WriteFile(spec, []byte(`
import { test, expect, page, runtime } from '@choysum/e2e';
test('x', async () => { void page; void runtime; void expect; });
`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "entry.js")
	if err := WriteE2EEntry(entry, []string{spec}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "bundle.js")
	res, err := BuildE2EBundle(E2EBundleOptions{
		RepoRoot:   repoRoot,
		EntryPath:  entry,
		Outfile:    out,
		WorkingDir: repoRoot,
		RunDir:     dir,
	})
	if err != nil {
		t.Fatalf("BuildE2EBundle: %v", err)
	}
	if res == nil || res.JS == "" {
		t.Fatal("empty bundle")
	}
	if !strings.Contains(res.JS, "__choysum_e2e_host__") && !strings.Contains(res.JS, "getHost") {
		t.Fatalf("bundle missing e2e facade markers")
	}
}

func TestBuildE2EBundleSmoke(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	dir := t.TempDir()
	spec := filepath.Join(dir, "ok.spec.ts")
	if err := os.WriteFile(spec, []byte(`
import { test, expect, page, runtime } from '@choysum/e2e';
test('ok', async () => { expect(runtime).toBeTruthy(); });
`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "entry.js")
	if err := WriteE2EEntry(entry, []string{spec}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "bundle.js")
	res, err := BuildE2EBundle(E2EBundleOptions{
		RepoRoot:   repoRoot,
		EntryPath:  entry,
		Outfile:    out,
		WorkingDir: repoRoot,
		RunDir:     dir,
		CacheDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("BuildE2EBundle: %v", err)
	}
	if res == nil || res.JS == "" {
		t.Fatal("empty bundle")
	}
}

func TestBuildE2EBundleErrorPaths(t *testing.T) {
	if _, err := BuildE2EBundle(E2EBundleOptions{}); err == nil || !strings.Contains(err.Error(), "empty repo root") {
		t.Fatalf("repo root: %v", err)
	}
	if _, err := BuildE2EBundle(E2EBundleOptions{RepoRoot: "/tmp"}); err == nil || !strings.Contains(err.Error(), "empty entry path") {
		t.Fatalf("entry: %v", err)
	}

	oldPath := e2eChoysumE2EPath
	defer func() { e2eChoysumE2EPath = oldPath }()

	e2eChoysumE2EPath = func() (string, error) { return "", errors.New("path boom") }
	if _, err := BuildE2EBundle(E2EBundleOptions{RepoRoot: "/tmp", EntryPath: "/tmp/e.js"}); err == nil || !strings.Contains(err.Error(), "choysume2e path") {
		t.Fatalf("path: %v", err)
	}

	oldAbs := e2eFilepathAbs
	e2eChoysumE2EPath = func() (string, error) { return "/x/choysume2e.js", nil }
	e2eFilepathAbs = func(path string) (string, error) { return "", errors.New("abs boom") }
	if _, err := BuildE2EBundle(E2EBundleOptions{RepoRoot: "/tmp", EntryPath: "/tmp/e.js"}); err == nil || !strings.Contains(err.Error(), "abs choysume2e") {
		t.Fatalf("abs: %v", err)
	}
	e2eFilepathAbs = oldAbs

	oldHome := e2eOsUserHome
	t.Setenv("CHOYSUM_HOME", "")
	e2eOsUserHome = func() (string, error) { return "", errors.New("home boom") }
	e2ePathFile := filepath.Join(t.TempDir(), "choysume2e.js")
	if err := os.WriteFile(e2ePathFile, []byte("export {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	e2eChoysumE2EPath = func() (string, error) { return e2ePathFile, nil }
	if _, err := BuildE2EBundle(E2EBundleOptions{RepoRoot: t.TempDir(), EntryPath: filepath.Join(t.TempDir(), "e.js")}); err == nil || !strings.Contains(err.Error(), "cache dir") {
		t.Fatalf("home: %v", err)
	}
	e2eOsUserHome = oldHome

	oldMkdir := e2eOsMkdirAll
	e2eOsMkdirAll = func(path string, perm os.FileMode) error { return errors.New("outdir fail") }
	e2eChoysumE2EPath = func() (string, error) { return e2ePathFile, nil }
	if _, err := BuildE2EBundle(E2EBundleOptions{RepoRoot: t.TempDir(), EntryPath: filepath.Join(t.TempDir(), "e.js"), CacheDir: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "mkdir") {
		t.Fatalf("mkdir: %v", err)
	}
	e2eOsMkdirAll = oldMkdir

	oldBuild := e2eEsbuildBuild
	e2eEsbuildBuild = func(api.BuildOptions) api.BuildResult {
		return api.BuildResult{Errors: []api.Message{{Text: "esbuild fail"}}}
	}
	if _, err := BuildE2EBundle(E2EBundleOptions{RepoRoot: t.TempDir(), EntryPath: filepath.Join(t.TempDir(), "e.js"), CacheDir: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "esbuild") {
		t.Fatalf("esbuild: %v", err)
	}

	e2eEsbuildBuild = func(opts api.BuildOptions) api.BuildResult {
		return api.BuildResult{}
	}
	oldRead := e2eOsReadFile
	e2eOsReadFile = func(name string) ([]byte, error) { return nil, errors.New("read fail") }
	defer func() {
		e2eEsbuildBuild = oldBuild
		e2eOsReadFile = oldRead
	}()
	if _, err := BuildE2EBundle(E2EBundleOptions{RepoRoot: t.TempDir(), EntryPath: filepath.Join(t.TempDir(), "e.js"), CacheDir: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "read outfile") {
		t.Fatalf("read: %v", err)
	}
}

func TestBuildE2EBundleExtensionlessPbImport(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	runDir := t.TempDir()
	genWeb := filepath.Join(runDir, ".choysum", "generated", "web")
	if err := os.MkdirAll(genWeb, 0o755); err != nil {
		t.Fatal(err)
	}
	// Extensionless import `./foo_pb` must resolve to foo_pb.ts via baseNoExt+".ts".
	if err := os.WriteFile(filepath.Join(genWeb, "foo_pb.ts"), []byte("export const Foo = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Extra sibling so WalkDir (if used) also visits a non-matching file (return nil).
	if err := os.WriteFile(filepath.Join(genWeb, "other_pb.ts"), []byte("export const Other = 2;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	spec := filepath.Join(dir, "extless_pb.spec.ts")
	if err := os.WriteFile(spec, []byte(`
import { test } from '@choysum/e2e';
import { Foo } from './foo_pb';
test('extless', async () => { void Foo; });
`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "entry.js")
	if err := WriteE2EEntry(entry, []string{spec}); err != nil {
		t.Fatal(err)
	}
	res, err := BuildE2EBundle(E2EBundleOptions{
		RepoRoot:   repoRoot,
		EntryPath:  entry,
		Outfile:    filepath.Join(dir, "bundle.js"),
		WorkingDir: dir,
		RunDir:     runDir,
		CacheDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("BuildE2EBundle: %v", err)
	}
	if res == nil || res.JS == "" {
		t.Fatal("empty bundle")
	}
}

func TestBuildE2EBundleGeneratedPbWalkError(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	runDir := t.TempDir()
	genRoot := filepath.Join(runDir, ".choysum", "generated")
	if err := os.MkdirAll(genRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	// Unreadable directory forces WalkDir callback err path in the pb plugin.
	blocked := filepath.Join(genRoot, "blocked")
	if err := os.MkdirAll(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })

	dir := t.TempDir()
	spec := filepath.Join(dir, "walk_pb.spec.ts")
	if err := os.WriteFile(spec, []byte(`
import { test } from '@choysum/e2e';
import { X } from 'walk_pb.ts';
test('x', async () => { void X; });
`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "entry.js")
	if err := WriteE2EEntry(entry, []string{spec}); err != nil {
		t.Fatal(err)
	}
	_, err := BuildE2EBundle(E2EBundleOptions{
		RepoRoot:   repoRoot,
		EntryPath:  entry,
		Outfile:    filepath.Join(dir, "bundle.js"),
		WorkingDir: dir,
		RunDir:     runDir,
		CacheDir:   t.TempDir(),
	})
	// Bundle fails to resolve; walk err path is still exercised.
	if err == nil {
		t.Fatal("expected bundle failure")
	}
}

func TestBuildE2EBundleGeneratedPbNotFound(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	runDir := t.TempDir()
	// Empty generated tree → plugin WalkDir finds nothing and returns empty resolve.
	if err := os.MkdirAll(filepath.Join(runDir, ".choysum", "generated"), 0o755); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	spec := filepath.Join(dir, "missing_pb.spec.ts")
	// Import a bare relative path that matches the _pb filter but does not exist;
	// esbuild will fail, but the plugin's empty OnResolveResult path is exercised first.
	if err := os.WriteFile(spec, []byte(`
import { test } from '@choysum/e2e';
import { Missing } from './gone_pb.ts';
test('x', async () => { void Missing; });
`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "entry.js")
	if err := WriteE2EEntry(entry, []string{spec}); err != nil {
		t.Fatal(err)
	}
	_, err := BuildE2EBundle(E2EBundleOptions{
		RepoRoot:   repoRoot,
		EntryPath:  entry,
		Outfile:    filepath.Join(dir, "bundle.js"),
		WorkingDir: dir,
		RunDir:     runDir,
		CacheDir:   t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected esbuild failure for missing pb")
	}
}

func TestBuildE2EBundleGeneratedPbPlugin(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	runDir := t.TempDir()
	genWeb := filepath.Join(runDir, ".choysum", "generated", "web")
	if err := os.MkdirAll(genWeb, 0o755); err != nil {
		t.Fatal(err)
	}
	pbWeb := filepath.Join(genWeb, "demo_pb.ts")
	if err := os.WriteFile(pbWeb, []byte("export const Demo = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Nested copy for WalkDir fallback (different base name).
	nested := filepath.Join(runDir, ".choysum", "generated", "other")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "nested_pb.ts"), []byte("export const Nested = 2;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	spec := filepath.Join(dir, "pb.spec.ts")
	if err := os.WriteFile(spec, []byte(`
import { test } from '@choysum/e2e';
import { Demo } from 'demo_pb.ts';
import { Nested } from 'nested_pb.ts';
test('pb', async () => { void Demo; void Nested; });
`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "entry.js")
	if err := WriteE2EEntry(entry, []string{spec}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "bundle.js")
	res, err := BuildE2EBundle(E2EBundleOptions{
		RepoRoot:   repoRoot,
		EntryPath:  entry,
		Outfile:    out,
		WorkingDir: repoRoot,
		RunDir:     runDir,
		CacheDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("BuildE2EBundle: %v", err)
	}
	if res == nil || res.JS == "" {
		t.Fatal("empty bundle")
	}
}
