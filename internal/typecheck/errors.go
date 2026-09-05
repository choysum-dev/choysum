// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"errors"
	"strings"
)

var (
	ErrModulesPathRequired = errors.New("typecheck: modules_path is required")
	ErrRepoRootRequired    = errors.New("typecheck: repo_root is required")
	ErrAppRequired         = errors.New("typecheck: app is required")
	ErrNoRootFiles         = errors.New("typecheck: no checkable TypeScript roots")
	ErrUnsupportedScope    = errors.New("typecheck: unsupported scope")
)

func validateOptions(opts Options) error {
	if strings.TrimSpace(opts.ModulesPath) == "" {
		return ErrModulesPathRequired
	}
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return ErrRepoRootRequired
	}
	if strings.TrimSpace(opts.App) == "" {
		return ErrAppRequired
	}
	return nil
}
