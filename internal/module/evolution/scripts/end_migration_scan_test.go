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
