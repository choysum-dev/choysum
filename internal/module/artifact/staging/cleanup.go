// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package staging

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

// CleanupCrashArtifacts removes leftovers that can be left behind if the process
// crashes between CommitKeepBackup and Finalize/Rollback.
//
// It is intended to be called before starting a new module operation while
// holding the module manager lease.
//
// roots should include workspace paths related to operation targets,
// e.g. <dist_root>/ and <default_choysum_path>/generated/.
func CleanupCrashArtifacts(ctx context.Context, roots []string, currentOpID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	checkCtx := func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	normalizedRoots := make([]string, 0, len(roots))
	stagingRoots := map[string]struct{}{}
	tmpRoot, _ := tmpRootFromContext(ctx)
	for _, root := range roots {
		if err := checkCtx(); err != nil {
			return err
		}
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "" || root == "." {
			continue
		}
		normalizedRoots = append(normalizedRoots, root)
		stagingRoot, err := WorkspaceStagingRoot(root, tmpRoot)
		if err != nil {
			return xfmt.Errorf("resolve staging root for %s: %w", root, err)
		}
		stagingRoots[stagingRoot] = struct{}{}
	}

	// 1) Clean staging op dirs: <tmpRoot>/staging/<opid>/...
	for stagingRoot := range stagingRoots {
		if err := checkCtx(); err != nil {
			return err
		}
		if entries, err := os.ReadDir(stagingRoot); err == nil {
			for _, ent := range entries {
				if err := checkCtx(); err != nil {
					return err
				}
				if !ent.IsDir() {
					continue
				}
				name := ent.Name()
				if name == "" || name == currentOpID {
					continue
				}
				if err := os.RemoveAll(filepath.Join(stagingRoot, name)); err != nil {
					return xfmt.Errorf("cleanup staging op dir %s: %w", filepath.Join(stagingRoot, name), err)
				}
			}
			_ = os.Remove(stagingRoot)
		} else if !os.IsNotExist(err) {
			return xfmt.Errorf("read staging root %s: %w", stagingRoot, err)
		}
	}

	for _, root := range normalizedRoots {
		if err := checkCtx(); err != nil {
			return err
		}

		// 2) Clean leftover backup/failed dirs/files next to targets:
		//    <target>.old.<opid> and <target>.failed.<opid>
		if entries, err := os.ReadDir(root); err == nil {
			for _, ent := range entries {
				if err := checkCtx(); err != nil {
					return err
				}
				name := ent.Name()
				if name == "" {
					continue
				}
				if strings.Contains(name, ".old.") || strings.Contains(name, ".failed.") {
					if err := os.RemoveAll(filepath.Join(root, name)); err != nil {
						return xfmt.Errorf("cleanup leftover artifact %s: %w", filepath.Join(root, name), err)
					}
				}
			}
		} else if !os.IsNotExist(err) {
			return xfmt.Errorf("read root %s: %w", root, err)
		}
	}

	return nil
}
