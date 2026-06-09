// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package origin

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

func WorkspaceRoot(runtimeScope scope.Scope) string {
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	if configPath := strings.TrimSpace(runtimeOpts.configPath); configPath != "" {
		if absConfigPath, err := filepath.Abs(configPath); err == nil && strings.TrimSpace(absConfigPath) != "" {
			configPath = absConfigPath
		}
		return filepath.Dir(configPath)
	}
	if modulesPath := strings.TrimSpace(runtimeOpts.modulesPath); modulesPath != "" {
		if absModulesPath, err := filepath.Abs(modulesPath); err == nil && strings.TrimSpace(absModulesPath) != "" {
			modulesPath = absModulesPath
		}
		return filepath.Dir(modulesPath)
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

func workspaceChoysumDir(workspaceRoot string, defaultChoysumPath string) (string, error) {
	_ = workspaceRoot
	override := strings.TrimSpace(defaultChoysumPath)
	if override == "" {
		return "", xfmt.Errorf("defaultChoysumPath is required")
	}
	if absOverride, absErr := filepath.Abs(override); absErr == nil {
		override = absOverride
	}
	override = filepath.Clean(override)
	if override == "." || override == string(filepath.Separator) {
		return "", xfmt.Errorf("defaultChoysumPath must be a non-root directory")
	}
	return override, nil
}

func modulesLockFilePath(workspaceRoot string, defaultChoysumPath string) (string, error) {
	choysumDir, err := workspaceChoysumDir(workspaceRoot, defaultChoysumPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(choysumDir, "modules.lock.json"), nil
}

func modulesLockLeasePath(workspaceRoot string, defaultChoysumPath string) (string, error) {
	lockPath, err := modulesLockFilePath(workspaceRoot, defaultChoysumPath)
	if err != nil {
		return "", err
	}
	return lockPath + ".lock", nil
}
