// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/scope"
)

type originPathsScope struct {
	ctx   context.Context
	input scope.FactoryInput
}

func (s *originPathsScope) Run(fn func(scope.Scope) error) error {
	if fn == nil {
		return nil
	}
	return fn(s)
}

func (s *originPathsScope) Session() *scope.Session { return nil }
func (s *originPathsScope) Transactor() scope.Transactor {
	return nil
}

func (s *originPathsScope) WithContext(ctx context.Context) scope.Scope {
	clone := *s
	clone.ctx = ctx
	return &clone
}

func (s *originPathsScope) Context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *originPathsScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (s *originPathsScope) FactoryInput() scope.FactoryInput {
	return s.input
}

type originPathsInput struct {
	modulesPath string
	configPath  string
}

func (i originPathsInput) Environment() string        { return "" }
func (i originPathsInput) ModulesPath() string        { return i.modulesPath }
func (i originPathsInput) DistPath() string           { return "" }
func (i originPathsInput) TmpPath() string            { return "" }
func (i originPathsInput) DefaultChoysumPath() string { return "" }
func (i originPathsInput) ConfigPath() string         { return i.configPath }

func TestWorkspaceRootPrefersConfigPath(t *testing.T) {
	workspaceRoot := t.TempDir()
	configPath := filepath.Join(workspaceRoot, "configs", "dev.yaml")
	modulesPath := filepath.Join(workspaceRoot, "modules")

	runtimeScope := &originPathsScope{ctx: context.Background(), input: originPathsInput{configPath: configPath, modulesPath: modulesPath}}
	got := WorkspaceRoot(runtimeScope)

	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", configPath, err)
	}
	want := filepath.Dir(absConfigPath)
	if got != want {
		t.Fatalf("WorkspaceRoot() = %q, want %q", got, want)
	}
}

func TestWorkspaceRootFallsBackToModulesPathAndCWD(t *testing.T) {
	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")

	runtimeScope := &originPathsScope{ctx: context.Background(), input: originPathsInput{modulesPath: modulesPath}}
	got := WorkspaceRoot(runtimeScope)
	absModulesPath, err := filepath.Abs(modulesPath)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", modulesPath, err)
	}
	want := filepath.Dir(absModulesPath)
	if got != want {
		t.Fatalf("WorkspaceRoot() with modules path = %q, want %q", got, want)
	}

	originCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(originCWD)
	}()
	fallbackCWD := t.TempDir()
	if err := os.Chdir(fallbackCWD); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", fallbackCWD, err)
	}

	fallbackScope := &originPathsScope{ctx: context.Background(), input: originPathsInput{}}
	gotFallback := WorkspaceRoot(fallbackScope)
	resolvedGot := gotFallback
	if eval, err := filepath.EvalSymlinks(gotFallback); err == nil {
		resolvedGot = eval
	}
	resolvedWant := fallbackCWD
	if eval, err := filepath.EvalSymlinks(fallbackCWD); err == nil {
		resolvedWant = eval
	}
	if resolvedGot != resolvedWant {
		t.Fatalf("WorkspaceRoot() cwd fallback = %q (resolved %q), want %q (resolved %q)", gotFallback, resolvedGot, fallbackCWD, resolvedWant)
	}
}

func TestWorkspaceRootModulesPathDirFallbackWhenAbsFails(t *testing.T) {
	originCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(originCWD)
	}()

	brokenRoot := t.TempDir()
	brokenCWD := filepath.Join(brokenRoot, "gone")
	if err := os.MkdirAll(brokenCWD, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", brokenCWD, err)
	}
	if err := os.Chdir(brokenCWD); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", brokenCWD, err)
	}
	if err := os.RemoveAll(brokenRoot); err != nil {
		t.Fatalf("os.RemoveAll(%q) error = %v", brokenRoot, err)
	}

	if _, err := filepath.Abs("relative/modules"); err == nil {
		t.Skip("platform did not trigger filepath.Abs failure from deleted cwd")
	}

	runtimeScope := &originPathsScope{ctx: context.Background(), input: originPathsInput{modulesPath: "relative/modules"}}
	if got := WorkspaceRoot(runtimeScope); got != "relative" {
		t.Fatalf("WorkspaceRoot() = %q, want %q", got, "relative")
	}
}

func TestWorkspaceChoysumDirAndLockPaths(t *testing.T) {
	workspaceRoot := t.TempDir()

	if _, err := workspaceChoysumDir(workspaceRoot, ""); err == nil {
		t.Fatal("expected workspaceChoysumDir to reject empty defaultChoysumPath")
	}
	if _, err := workspaceChoysumDir(workspaceRoot, string(filepath.Separator)); err == nil {
		t.Fatal("expected workspaceChoysumDir to reject root path")
	}

	override := filepath.Join(workspaceRoot, ".choysum")
	choysumDir, err := workspaceChoysumDir(workspaceRoot, override)
	if err != nil {
		t.Fatalf("workspaceChoysumDir() error = %v", err)
	}
	absOverride, err := filepath.Abs(override)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", override, err)
	}
	if choysumDir != filepath.Clean(absOverride) {
		t.Fatalf("workspaceChoysumDir() = %q, want %q", choysumDir, filepath.Clean(absOverride))
	}

	lockPath, err := modulesLockFilePath(workspaceRoot, override)
	if err != nil {
		t.Fatalf("modulesLockFilePath() error = %v", err)
	}
	if lockPath != filepath.Join(choysumDir, "modules.lock.json") {
		t.Fatalf("modulesLockFilePath() = %q, want %q", lockPath, filepath.Join(choysumDir, "modules.lock.json"))
	}

	leasePath, err := modulesLockLeasePath(workspaceRoot, override)
	if err != nil {
		t.Fatalf("modulesLockLeasePath() error = %v", err)
	}
	if leasePath != lockPath+".lock" {
		t.Fatalf("modulesLockLeasePath() = %q, want %q", leasePath, lockPath+".lock")
	}
}

func TestLockStoreUpsertLookupDelete(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	defaultChoysumPath := t.TempDir()
	store := NewLockStore(WithLockStoreDefaultChoysumPath(defaultChoysumPath))

	if err := store.UpsertBinding(workspaceRoot, Binding{
		ModuleName:      "auth",
		OriginType:      OriginTypeRegistry,
		OriginRef:       "auth@v1.2.3",
		ResolvedVersion: "v1.2.3",
		Integrity:       "sha512-auth-v1.2.3",
		LocalPath:       "/tmp/modules/auth",
	}); err != nil {
		t.Fatalf("UpsertBinding() error = %v", err)
	}

	binding, ok, err := store.LookupBinding(workspaceRoot, "auth")
	if err != nil {
		t.Fatalf("LookupBinding() error = %v", err)
	}
	if !ok || binding.OriginRef != "auth@v1.2.3" {
		t.Fatalf("unexpected binding after upsert: ok=%v binding=%#v", ok, binding)
	}
	if binding.Integrity != "sha512-auth-v1.2.3" {
		t.Fatalf("unexpected integrity after upsert: %#v", binding)
	}

	if err := store.DeleteBinding(workspaceRoot, "auth"); err != nil {
		t.Fatalf("DeleteBinding() error = %v", err)
	}
	_, ok, err = store.LookupBinding(workspaceRoot, "auth")
	if err != nil {
		t.Fatalf("LookupBinding(after delete) error = %v", err)
	}
	if ok {
		t.Fatal("expected binding to be deleted")
	}
}

func TestLockStoreUpsertBindingSameContentDoesNotRewriteFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	defaultChoysumPath := t.TempDir()
	store := NewLockStore(WithLockStoreDefaultChoysumPath(defaultChoysumPath))

	binding := Binding{
		ModuleName:      "auth",
		OriginType:      OriginTypeRegistry,
		OriginRef:       "auth@v2.0.0",
		ResolvedVersion: "v2.0.0",
		Integrity:       "sha512-auth-v2",
		LocalPath:       "/tmp/modules/auth",
	}
	if err := store.UpsertBinding(workspaceRoot, binding); err != nil {
		t.Fatalf("first UpsertBinding() error = %v", err)
	}

	lockPath, err := modulesLockFilePath(workspaceRoot, defaultChoysumPath)
	if err != nil {
		t.Fatalf("modulesLockFilePath() error = %v", err)
	}
	before, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock file (before): %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	if err := store.UpsertBinding(workspaceRoot, binding); err != nil {
		t.Fatalf("second UpsertBinding() error = %v", err)
	}

	after, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock file (after): %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("expected identical lockfile after idempotent upsert\nbefore=%s\nafter=%s", string(before), string(after))
	}
}

func TestLockStoreDeleteMissingBindingDoesNotCreateFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	defaultChoysumPath := t.TempDir()
	store := NewLockStore(WithLockStoreDefaultChoysumPath(defaultChoysumPath))

	if err := store.DeleteBinding(workspaceRoot, "missing"); err != nil {
		t.Fatalf("DeleteBinding(missing) error = %v", err)
	}

	lockPath, err := modulesLockFilePath(workspaceRoot, defaultChoysumPath)
	if err != nil {
		t.Fatalf("modulesLockFilePath() error = %v", err)
	}
	if _, statErr := os.Stat(lockPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected lock file to stay absent, stat err = %v", statErr)
	}
}

func TestLockStoreUpsertBindingConcurrentStress(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	defaultChoysumPath := t.TempDir()
	store := NewLockStore(WithLockStoreDefaultChoysumPath(defaultChoysumPath))

	lockPath, err := modulesLockLeasePath(workspaceRoot, defaultChoysumPath)
	if err != nil {
		t.Fatalf("modulesLockLeasePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("mkdir lock dir: %v", err)
	}
	busy := ModulesLockLease{
		Owner:         "holder",
		PID:           12345,
		StartedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		TTL:           (2 * time.Second).String(),
		Operation:     "stress",
		LastHeartbeat: time.Now().UTC().Format(time.RFC3339Nano),
		ErrorCode:     LockConflictErrorCodeDefault,
		RetryBackoff:  LockRetryBackoffInitialDefault.String(),
	}
	if err := writeLeaseFileExclusive(lockPath, busy); err != nil {
		t.Fatalf("write initial busy lease failed: %v", err)
	}
	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = os.Remove(lockPath)
	}()

	const workers = 16
	const iterations = 20
	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	var conflictCount atomic.Int64

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			moduleName := fmt.Sprintf("auth_%02d", worker%4)
			for i := 0; i < iterations; i++ {
				binding := Binding{
					ModuleName:      moduleName,
					OriginType:      OriginTypeLocal,
					OriginRef:       moduleName,
					ResolvedVersion: fmt.Sprintf("v1.0.%d", i),
					LocalPath:       fmt.Sprintf("/tmp/%s", moduleName),
				}

				deadline := time.Now().Add(5 * time.Second)
				lastErr := error(nil)
				for {
					err := store.UpsertBinding(workspaceRoot, binding)
					if err == nil {
						lastErr = nil
						break
					}
					var conflictErr *LockConflictError
					if errors.As(err, &conflictErr) {
						conflictCount.Add(1)
						lastErr = err
						if time.Now().After(deadline) {
							break
						}
						time.Sleep(2 * time.Millisecond)
						continue
					}
					lastErr = err
					break
				}

				if lastErr != nil {
					errCh <- fmt.Errorf("worker %d iteration %d upsert failed: %w", worker, i, lastErr)
					return
				}
			}
		}(worker)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	if conflictCount.Load() == 0 {
		t.Fatal("expected at least one lock conflict under concurrent stress")
	}

	lock, err := store.Read(workspaceRoot)
	if err != nil {
		t.Fatalf("Read() after stress error = %v", err)
	}
	if len(lock.Modules) == 0 {
		t.Fatalf("expected non-empty lock modules after stress, got %#v", lock)
	}
}

func TestAcquireModulesLockLeaseConflictAndStaleRecovery(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	defaultChoysumPath := t.TempDir()
	lockPath, err := modulesLockLeasePath(workspaceRoot, defaultChoysumPath)
	if err != nil {
		t.Fatalf("modulesLockLeasePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("mkdir lock dir: %v", err)
	}

	busy := ModulesLockLease{
		Owner:         "holder",
		PID:           12345,
		StartedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		TTL:           (10 * time.Minute).String(),
		Operation:     "busy",
		LastHeartbeat: time.Now().UTC().Format(time.RFC3339Nano),
		ErrorCode:     LockConflictErrorCodeDefault,
		RetryBackoff:  LockRetryBackoffInitialDefault.String(),
	}
	if err := writeLeaseFileExclusive(lockPath, busy); err != nil {
		t.Fatalf("write busy lease failed: %v", err)
	}

	if _, err := AcquireModulesLockLease(workspaceRoot, "op", defaultChoysumPath); err == nil {
		t.Fatal("expected AcquireModulesLockLease conflict error")
	}

	if err := os.Remove(lockPath); err != nil {
		t.Fatalf("remove busy lock failed: %v", err)
	}
	stale := busy
	stale.TTL = (100 * time.Millisecond).String()
	stale.StartedAt = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano)
	stale.LastHeartbeat = stale.StartedAt
	if err := writeLeaseFileExclusive(lockPath, stale); err != nil {
		t.Fatalf("write stale lease failed: %v", err)
	}

	release, err := AcquireModulesLockLease(workspaceRoot, "op", defaultChoysumPath)
	if err != nil {
		t.Fatalf("AcquireModulesLockLease(stale) error = %v", err)
	}
	if err := release(); err != nil {
		t.Fatalf("release lock failed: %v", err)
	}
}
