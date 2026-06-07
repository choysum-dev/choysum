// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLockStoreUpsertLookupDelete(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspaceRoot := t.TempDir()
	defaultChoysumPath := t.TempDir()
	store := NewLockStore(WithLockStoreDefaultChoysumPath(defaultChoysumPath))

	if err := store.UpsertBinding(workspaceRoot, Binding{
		ModuleName:      "auth",
		OriginType:      OriginTypeRegistry,
		OriginRef:       "official/auth@v1.2.3",
		ResolvedVersion: "v1.2.3",
		Integrity:       "sha512-auth-v1.2.3",
		LocalPath:       "/tmp/addons/auth",
	}); err != nil {
		t.Fatalf("UpsertBinding() error = %v", err)
	}

	binding, ok, err := store.LookupBinding(workspaceRoot, "auth")
	if err != nil {
		t.Fatalf("LookupBinding() error = %v", err)
	}
	if !ok || binding.OriginRef != "official/auth@v1.2.3" {
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
		OriginRef:       "corp/auth@v2.0.0",
		ResolvedVersion: "v2.0.0",
		Integrity:       "sha512-auth-v2",
		LocalPath:       "/tmp/addons/auth",
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
