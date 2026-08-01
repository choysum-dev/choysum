package origin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

type upgradeSwitchStubCoordinator struct {
	fetchFn func(ctx context.Context, input string) (*meta.Module, error)
}

func (s upgradeSwitchStubCoordinator) Peek(context.Context, string) (*meta.Module, error) {
	return nil, nil
}

func (s upgradeSwitchStubCoordinator) ResolveInstallModule(context.Context, string) (*meta.Module, error) {
	return nil, nil
}

func (s upgradeSwitchStubCoordinator) Fetch(ctx context.Context, input string) (*meta.Module, error) {
	if s.fetchFn != nil {
		return s.fetchFn(ctx, input)
	}
	return &meta.Module{Name: "auth"}, nil
}

func (s upgradeSwitchStubCoordinator) Purge(context.Context, string) error {
	return nil
}

func TestPrepareUpgradeSwitchGuards(t *testing.T) {
	t.Parallel()

	snapshot, err := PrepareUpgradeSwitch(context.Background(), nil, t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), ParsedInput{Kind: InputKindLocal}, "auth", "")
	if err != nil {
		t.Fatalf("PrepareUpgradeSwitch(non-registry) error = %v", err)
	}
	if snapshot != nil {
		t.Fatalf("PrepareUpgradeSwitch(non-registry) = %#v, want nil", snapshot)
	}

	parsedRegistry := ParsedInput{Kind: InputKindRegistry, ModuleName: "auth", Version: "latest"}

	if _, err := PrepareUpgradeSwitch(context.Background(), upgradeSwitchStubCoordinator{}, t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), parsedRegistry, "   ", ""); err == nil || !strings.Contains(err.Error(), "module name is empty") {
		t.Fatalf("expected empty module name error, got %v", err)
	}

	if _, err := PrepareUpgradeSwitch(context.Background(), upgradeSwitchStubCoordinator{}, t.TempDir(), t.TempDir(), t.TempDir(), "", parsedRegistry, "auth", ""); err == nil || !strings.Contains(err.Error(), "scope config is not initialized") {
		t.Fatalf("expected missing defaultChoysumPath error, got %v", err)
	}

	if _, err := PrepareUpgradeSwitch(context.Background(), nil, t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir(), parsedRegistry, "auth", ""); err == nil || !strings.Contains(err.Error(), "origin coordinator is nil") {
		t.Fatalf("expected nil coordinator error, got %v", err)
	}
}

func TestUpgradeSwitchSnapshotCommitAndRollbackSafety(t *testing.T) {
	t.Parallel()

	var nilSnapshot *UpgradeSwitchSnapshot
	if err := nilSnapshot.Commit(); err != nil {
		t.Fatalf("nil snapshot Commit() error = %v", err)
	}
	if err := nilSnapshot.Rollback(); err != nil {
		t.Fatalf("nil snapshot Rollback() error = %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup")
	if err := os.MkdirAll(backupPath, 0o755); err != nil {
		t.Fatalf("mkdir backup path: %v", err)
	}
	snapshot := &UpgradeSwitchSnapshot{backupPath: backupPath}
	if err := snapshot.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatalf("expected backup path to be removed on Commit(), stat error=%v", err)
	}

	modulePath := filepath.Join(t.TempDir(), "module")
	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatalf("mkdir module path: %v", err)
	}
	snapshot = &UpgradeSwitchSnapshot{modulePath: modulePath}
	if err := snapshot.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if _, err := os.Stat(modulePath); !os.IsNotExist(err) {
		t.Fatalf("expected module path to be removed on Rollback(), stat error=%v", err)
	}
}

func TestNormalizeUpgradeBackupTmpRoot(t *testing.T) {
	t.Parallel()

	if _, err := normalizeUpgradeBackupTmpRoot(" "); err == nil || !strings.Contains(err.Error(), "tmpRoot is required") {
		t.Fatalf("expected tmpRoot required error, got %v", err)
	}

	if _, err := normalizeUpgradeBackupTmpRoot(string(filepath.Separator)); err == nil || !strings.Contains(err.Error(), "non-root") {
		t.Fatalf("expected non-root directory error, got %v", err)
	}

	root := t.TempDir()
	input := filepath.Join(root, "a", "..", "b")
	normalized, err := normalizeUpgradeBackupTmpRoot(input)
	if err != nil {
		t.Fatalf("normalizeUpgradeBackupTmpRoot(valid) error = %v", err)
	}
	if normalized != filepath.Clean(filepath.Join(root, "b")) {
		t.Fatalf("normalizeUpgradeBackupTmpRoot(valid) = %q, want %q", normalized, filepath.Clean(filepath.Join(root, "b")))
	}
}

func TestPrepareUpgradeSwitchSuccessAndFetchFailureRollback(t *testing.T) {
	t.Parallel()

	parsedRegistry := ParsedInput{Kind: InputKindRegistry, ModuleName: "auth", Version: "latest"}

	t.Run("success with existing module backup and commit", func(t *testing.T) {
		workspaceRoot := t.TempDir()
		modulesPath := t.TempDir()
		defaultChoysumPath := t.TempDir()
		tmpPath := filepath.Join(t.TempDir(), "tmp")

		modulePath := filepath.Join(modulesPath, "auth")
		if err := os.MkdirAll(modulePath, 0o755); err != nil {
			t.Fatalf("MkdirAll(modulePath) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(modulePath, "marker.txt"), []byte("before"), 0o644); err != nil {
			t.Fatalf("WriteFile(marker) error = %v", err)
		}

		snapshot, err := PrepareUpgradeSwitch(
			context.Background(),
			upgradeSwitchStubCoordinator{fetchFn: func(context.Context, string) (*meta.Module, error) { return &meta.Module{Name: "auth"}, nil }},
			workspaceRoot,
			modulesPath,
			tmpPath,
			defaultChoysumPath,
			parsedRegistry,
			"auth",
			"fixed-opid",
		)
		if err != nil {
			t.Fatalf("PrepareUpgradeSwitch(success) error = %v", err)
		}
		if snapshot == nil {
			t.Fatal("PrepareUpgradeSwitch(success) returned nil snapshot")
		}
		if !snapshot.hadModuleDir {
			t.Fatal("snapshot.hadModuleDir = false, want true")
		}
		if snapshot.backupPath == "" {
			t.Fatal("snapshot.backupPath should be non-empty when module dir existed")
		}
		if _, err := os.Stat(snapshot.backupPath); err != nil {
			t.Fatalf("backup path should exist before commit, stat error = %v", err)
		}

		if err := snapshot.Commit(); err != nil {
			t.Fatalf("Commit() error = %v", err)
		}
		if _, err := os.Stat(snapshot.backupPath); !os.IsNotExist(err) {
			t.Fatalf("backup path should be removed after commit, stat error = %v", err)
		}
	})

	t.Run("fetch failure restores previous module directory", func(t *testing.T) {
		workspaceRoot := t.TempDir()
		modulesPath := t.TempDir()
		defaultChoysumPath := t.TempDir()
		tmpPath := filepath.Join(t.TempDir(), "tmp")

		modulePath := filepath.Join(modulesPath, "auth")
		if err := os.MkdirAll(modulePath, 0o755); err != nil {
			t.Fatalf("MkdirAll(modulePath) error = %v", err)
		}
		markerPath := filepath.Join(modulePath, "marker.txt")
		if err := os.WriteFile(markerPath, []byte("before"), 0o644); err != nil {
			t.Fatalf("WriteFile(marker) error = %v", err)
		}

		_, err := PrepareUpgradeSwitch(
			context.Background(),
			upgradeSwitchStubCoordinator{fetchFn: func(context.Context, string) (*meta.Module, error) { return nil, errors.New("fetch failed") }},
			workspaceRoot,
			modulesPath,
			tmpPath,
			defaultChoysumPath,
			parsedRegistry,
			"auth",
			"fixed-opid",
		)
		if err == nil || !strings.Contains(err.Error(), "switch module origin binding for upgrade auth failed") {
			t.Fatalf("PrepareUpgradeSwitch(fetch failure) error = %v, want wrapped switch failure", err)
		}
		if _, statErr := os.Stat(markerPath); statErr != nil {
			t.Fatalf("marker file should be restored after rollback, stat error = %v", statErr)
		}
	})
}
