package origin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/xid"
	xfmt "golang.org/x/exp/errors/fmt"
)

type UpgradeSwitchSnapshot struct {
	workspaceRoot   string
	moduleName      string
	modulePath      string
	backupPath      string
	hadModuleDir    bool
	lockStore       *LockStore
	hadBinding      bool
	previousBinding Binding
}

func PrepareUpgradeSwitch(ctx context.Context, coordinator Service, workspaceRoot string, addonsPath string, tmpPath string, defaultChoysumPath string, parsed ParsedInput, moduleName string, opid string) (*UpgradeSwitchSnapshot, error) {
	if parsed.Kind != InputKindRegistry {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return nil, xfmt.Errorf("module name is empty")
	}
	defaultChoysumPath = strings.TrimSpace(defaultChoysumPath)
	if defaultChoysumPath == "" {
		return nil, xfmt.Errorf("scope config is not initialized")
	}
	if coordinator == nil {
		return nil, xfmt.Errorf("origin coordinator is nil")
	}

	lockStore := NewLockStore(WithLockStoreDefaultChoysumPath(defaultChoysumPath))
	previousBinding, hadBinding, err := lockStore.LookupBinding(workspaceRoot, moduleName)
	if err != nil {
		return nil, xfmt.Errorf("lookup previous module origin binding %s failed: %w", moduleName, err)
	}

	modulePath := filepath.Join(strings.TrimSpace(addonsPath), moduleName)
	hadModuleDir := false
	if info, statErr := os.Stat(modulePath); statErr == nil {
		if !info.IsDir() {
			return nil, xfmt.Errorf("module path %s is not a directory", modulePath)
		}
		hadModuleDir = true
	} else if !os.IsNotExist(statErr) {
		return nil, xfmt.Errorf("stat module path %s failed: %w", modulePath, statErr)
	}

	snapshot := &UpgradeSwitchSnapshot{
		workspaceRoot:   workspaceRoot,
		moduleName:      moduleName,
		modulePath:      modulePath,
		hadModuleDir:    hadModuleDir,
		lockStore:       lockStore,
		hadBinding:      hadBinding,
		previousBinding: previousBinding,
	}

	if hadModuleDir {
		resolvedTmpRoot := strings.TrimSpace(tmpPath)
		if resolvedTmpRoot == "" {
			resolvedTmpRoot = filepath.Join(defaultChoysumPath, "tmp")
		}
		workspaceTmpRoot, err := normalizeUpgradeBackupTmpRoot(resolvedTmpRoot)
		if err != nil {
			return nil, xfmt.Errorf("resolve workspace tmp root failed: %w", err)
		}
		backupRoot := filepath.Join(workspaceTmpRoot, "upgrade-origin-backups")
		if err := os.MkdirAll(backupRoot, 0o755); err != nil {
			return nil, xfmt.Errorf("create upgrade backup dir %s failed: %w", backupRoot, err)
		}
		backupID := strings.TrimSpace(opid)
		if backupID == "" {
			backupID = xid.New().String()
		}
		snapshot.backupPath = filepath.Join(backupRoot, moduleName+"."+backupID+"."+xid.New().String())
		if err := os.Rename(modulePath, snapshot.backupPath); err != nil {
			return nil, xfmt.Errorf("backup current module origin %s failed: %w", modulePath, err)
		}
	}

	if _, err := coordinator.Fetch(ctx, parsed.CanonicalRef()); err != nil {
		if rollbackErr := snapshot.Rollback(); rollbackErr != nil {
			return nil, xfmt.Errorf("switch module origin binding for upgrade %s failed: %w; rollback failed: %v", moduleName, err, rollbackErr)
		}
		return nil, xfmt.Errorf("switch module origin binding for upgrade %s failed: %w", moduleName, err)
	}

	return snapshot, nil
}

func (s *UpgradeSwitchSnapshot) Rollback() error {
	if s == nil {
		return nil
	}

	var errs []error
	if modulePath := strings.TrimSpace(s.modulePath); modulePath != "" {
		if err := os.RemoveAll(modulePath); err != nil && !os.IsNotExist(err) {
			errs = append(errs, xfmt.Errorf("remove switched module path %s failed: %w", modulePath, err))
		}
	}

	if s.hadModuleDir {
		backupPath := strings.TrimSpace(s.backupPath)
		if backupPath != "" {
			if _, err := os.Stat(backupPath); err == nil {
				if err := os.Rename(backupPath, s.modulePath); err != nil {
					errs = append(errs, xfmt.Errorf("restore previous module path %s failed: %w", s.modulePath, err))
				}
			} else if !os.IsNotExist(err) {
				errs = append(errs, xfmt.Errorf("check backup module path %s failed: %w", backupPath, err))
			}
		}
	}

	if s.lockStore != nil {
		if s.hadBinding {
			if err := s.lockStore.UpsertBinding(s.workspaceRoot, s.previousBinding); err != nil {
				errs = append(errs, xfmt.Errorf("restore previous module origin binding %s failed: %w", s.moduleName, err))
			}
		} else {
			if err := s.lockStore.DeleteBinding(s.workspaceRoot, s.moduleName); err != nil {
				errs = append(errs, xfmt.Errorf("remove module origin binding %s failed: %w", s.moduleName, err))
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func (s *UpgradeSwitchSnapshot) Commit() error {
	if s == nil {
		return nil
	}
	backupPath := strings.TrimSpace(s.backupPath)
	if backupPath == "" {
		return nil
	}
	if err := os.RemoveAll(backupPath); err != nil && !os.IsNotExist(err) {
		return xfmt.Errorf("remove upgrade backup path %s failed: %w", backupPath, err)
	}
	return nil
}

func normalizeUpgradeBackupTmpRoot(tmpRoot string) (string, error) {
	tmpRoot = strings.TrimSpace(tmpRoot)
	if tmpRoot == "" {
		return "", xfmt.Errorf("tmpRoot is required")
	}
	if absTmpRoot, err := filepath.Abs(tmpRoot); err == nil {
		tmpRoot = absTmpRoot
	}
	tmpRoot = filepath.Clean(tmpRoot)
	if tmpRoot == "." || tmpRoot == string(filepath.Separator) {
		return "", xfmt.Errorf("tmpRoot must be a non-root directory")
	}
	return tmpRoot, nil
}
