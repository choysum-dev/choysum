// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestModuleSourceDeclaresEndMigration(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := filepath.Join(root, "service", "migrations.ts")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(src, []byte(`@Migration({ version: '1.0.0', constraints: { x: 1 }, phase: 'end', name: 'done' })
export function done() {}
`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	testSrc := filepath.Join(root, "service", "orm", "decorator", "lifecycle.test.ts")
	if err := os.MkdirAll(filepath.Dir(testSrc), 0o755); err != nil {
		t.Fatalf("mkdir test: %v", err)
	}
	if err := os.WriteFile(testSrc, []byte(`@Migration({ version: '2.0.0', phase: 'end', name: 'fixture' })
export function fixture() {}
`), 0o644); err != nil {
		t.Fatalf("write test: %v", err)
	}
	if !moduleSourceDeclaresEndMigration(&meta.Module{Path: root}) {
		t.Fatal("expected end migration declaration")
	}
	onlyTests := t.TempDir()
	onlyTestSrc := filepath.Join(onlyTests, "lifecycle.test.ts")
	if err := os.WriteFile(onlyTestSrc, []byte(`@Migration({ phase: 'end', name: 'x' })
export function x() {}
`), 0o644); err != nil {
		t.Fatalf("write only test: %v", err)
	}
	if moduleSourceDeclaresEndMigration(&meta.Module{Path: onlyTests}) {
		t.Fatal("*.test.ts fixtures must not declare production end migrations")
	}
	if !moduleSourceDeclaresEndMigration(&meta.Module{Path: filepath.Join(root, "missing")}) {
		t.Fatal("missing path should fail open (maybe has end migrations)")
	}
	if !moduleSourceDeclaresEndMigration(&meta.Module{Path: ""}) {
		t.Fatal("empty path should fail open so runtime PhaseEnd scripts can load")
	}
	if moduleSourceDeclaresEndMigration(nil) {
		t.Fatal("nil module should not declare end migration")
	}
	typed := t.TempDir()
	typedSrc := filepath.Join(typed, "service", "end.ts")
	if err := os.MkdirAll(filepath.Dir(typedSrc), 0o755); err != nil {
		t.Fatalf("mkdir typed: %v", err)
	}
	if err := os.WriteFile(typedSrc, []byte(`@Migration({ version: '1.0.0', phase: MigrationPhase.End, name: 'done' })
export function done() {}
`), 0o644); err != nil {
		t.Fatalf("write typed: %v", err)
	}
	if !moduleSourceDeclaresEndMigration(&meta.Module{Path: typed}) {
		t.Fatal("expected non-literal phase: MigrationPhase.End to fail open")
	}

	// Object-based options fail open so PhaseEnd is not skipped.
	spread := t.TempDir()
	spreadSrc := filepath.Join(spread, "service", "spread.ts")
	if err := os.MkdirAll(filepath.Dir(spreadSrc), 0o755); err != nil {
		t.Fatalf("mkdir spread: %v", err)
	}
	if err := os.WriteFile(spreadSrc, []byte(`const endPhase = { phase: 'end' as const }
@Migration({ ...endPhase, name: 'done' })
export function done() {}
`), 0o644); err != nil {
		t.Fatalf("write spread: %v", err)
	}
	if !moduleSourceDeclaresEndMigration(&meta.Module{Path: spread}) {
		t.Fatal("spread options should fail open for PhaseEnd")
	}

	quoted := t.TempDir()
	quotedSrc := filepath.Join(quoted, "service", "quoted.ts")
	if err := os.MkdirAll(filepath.Dir(quotedSrc), 0o755); err != nil {
		t.Fatalf("mkdir quoted: %v", err)
	}
	if err := os.WriteFile(quotedSrc, []byte(`@Migration({ 'phase': 'end', name: 'done' })
export function done() {}
`), 0o644); err != nil {
		t.Fatalf("write quoted: %v", err)
	}
	if !moduleSourceDeclaresEndMigration(&meta.Module{Path: quoted}) {
		t.Fatal("quoted phase key should declare end migration")
	}

	bracketed := t.TempDir()
	bracketSrc := filepath.Join(bracketed, "service", "bracket.ts")
	if err := os.MkdirAll(filepath.Dir(bracketSrc), 0o755); err != nil {
		t.Fatalf("mkdir bracket: %v", err)
	}
	if err := os.WriteFile(bracketSrc, []byte(`@Migration({ ['phase']: 'end', name: 'done' })
export function done() {}
`), 0o644); err != nil {
		t.Fatalf("write bracket: %v", err)
	}
	if !moduleSourceDeclaresEndMigration(&meta.Module{Path: bracketed}) {
		t.Fatal("bracket-quoted phase key should declare end migration")
	}

	variableArg := t.TempDir()
	variableSrc := filepath.Join(variableArg, "service", "var.ts")
	if err := os.MkdirAll(filepath.Dir(variableSrc), 0o755); err != nil {
		t.Fatalf("mkdir variable: %v", err)
	}
	if err := os.WriteFile(variableSrc, []byte(`const migrationOptions = { phase: 'end' as const, name: 'done' }
@Migration(migrationOptions)
export function done() {}
`), 0o644); err != nil {
		t.Fatalf("write variable: %v", err)
	}
	if !moduleSourceDeclaresEndMigration(&meta.Module{Path: variableArg}) {
		t.Fatal("variable Migration options should fail open")
	}

	// Non-directory path fails open.
	filePath := filepath.Join(t.TempDir(), "not-a-dir.ts")
	if err := os.WriteFile(filePath, []byte("export {}\n"), 0o644); err != nil {
		t.Fatalf("write file path: %v", err)
	}
	if !moduleSourceDeclaresEndMigration(&meta.Module{Path: filePath}) {
		t.Fatal("non-directory path should fail open")
	}

	// Skipped dirs / non-ts extensions / __tests__ do not count; empty tree => false.
	emptyTree := t.TempDir()
	if err := os.MkdirAll(filepath.Join(emptyTree, "__tests__"), 0o755); err != nil {
		t.Fatalf("mkdir __tests__: %v", err)
	}
	if err := os.WriteFile(filepath.Join(emptyTree, "__tests__", "x.ts"), []byte(`@Migration({ phase: 'end' })
export function x() {}
`), 0o644); err != nil {
		t.Fatalf("write tests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(emptyTree, "readme.md"), []byte("no"), 0o644); err != nil {
		t.Fatalf("write md: %v", err)
	}
	if moduleSourceDeclaresEndMigration(&meta.Module{Path: emptyTree}) {
		t.Fatal("empty production sources should report no end migrations")
	}

	// Unreadable file inside tree fails open.
	t.Run("unreadable source file", func(t *testing.T) {
		blocked := t.TempDir()
		blockedFile := filepath.Join(blocked, "service", "secret.ts")
		if err := os.MkdirAll(filepath.Dir(blockedFile), 0o755); err != nil {
			t.Fatalf("mkdir blocked: %v", err)
		}
		if err := os.WriteFile(blockedFile, []byte("export {}\n"), 0o600); err != nil {
			t.Fatalf("write blocked: %v", err)
		}
		if err := os.Chmod(blockedFile, 0); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(blockedFile, 0o644) })
		if _, err := os.ReadFile(blockedFile); err == nil {
			t.Skip("filesystem permits read despite mode 0")
		}
		if !moduleSourceDeclaresEndMigration(&meta.Module{Path: blocked}) {
			t.Fatal("unreadable source should fail open")
		}
	})

	// Unreadable directory entry fails open via WalkDir walkErr.
	t.Run("unreadable directory", func(t *testing.T) {
		blockedDir := t.TempDir()
		secretDir := filepath.Join(blockedDir, "secret")
		if err := os.MkdirAll(secretDir, 0o755); err != nil {
			t.Fatalf("mkdir secret: %v", err)
		}
		if err := os.WriteFile(filepath.Join(secretDir, "x.ts"), []byte("export {}\n"), 0o644); err != nil {
			t.Fatalf("write secret file: %v", err)
		}
		if err := os.Chmod(secretDir, 0); err != nil {
			t.Fatalf("chmod secret dir: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(secretDir, 0o755) })
		if _, err := os.ReadDir(secretDir); err == nil {
			t.Skip("filesystem permits readdir despite mode 0")
		}
		if !moduleSourceDeclaresEndMigration(&meta.Module{Path: blockedDir}) {
			t.Fatal("unreadable directory should fail open")
		}
	})
}

func TestRunPhase_EndNoScripts_NoExecutorLoad(t *testing.T) {
	testRuntimeScope := newScriptsTestScope(t)
	writeScriptsRuntimeBundle(t, testRuntimeScope, "console.log('bundle')")
	prepareRunnerModuleSource(t, testRuntimeScope, "base", "service/index.ts", "export const migration = {}\n")

	engine := &scriptsSelectiveEngine{execute: func(req *jsengine.JsRequest, _ []*jsengine.JsScript) (*jsengine.JsResponse, error) {
		t.Fatalf("unexpected executor execute for PhaseEnd with no end migrations: %s", req.Service)
		return nil, nil
	}}
	baseExecutor := newScriptsTestExecutorWithEngine(t, testRuntimeScope, engine)
	countingExecutor := &reloadCountingExecutor{inner: baseExecutor}

	runner := NewRunner(testRuntimeScope, countingExecutor, &meta.Module{
		Name: "base", ApplicationStr: "core", Version: "1.2.0", ServiceEntryPoint: "service/index.ts",
		Path: filepath.Join(testRuntimeScope.cfg.ModulesPath, "base"),
	})
	if err := runner.RunPhase(context.Background(), RunOptions{
		FromVersion: "1.0.0", ToVersion: "1.2.0", Phase: PhaseEnd, ReuseExecutorScripts: true,
	}); err != nil {
		t.Fatalf("RunPhase() error = %v", err)
	}
	if countingExecutor.reloadCalls != 0 {
		t.Fatalf("expected no executor reload for empty PhaseEnd, got %d", countingExecutor.reloadCalls)
	}
}
