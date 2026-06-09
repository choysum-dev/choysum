// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"path/filepath"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

func normalizedDefaultChoysumRoot(defaultChoysumPath string) (string, error) {
	root := strings.TrimSpace(defaultChoysumPath)
	if root == "" {
		return "", xfmt.Errorf("defaultChoysumPath is required")
	}
	if absRoot, err := filepath.Abs(root); err == nil {
		root = absRoot
	}
	root = filepath.Clean(root)
	if root == "." || root == string(filepath.Separator) {
		return "", xfmt.Errorf("defaultChoysumPath must be a non-root directory")
	}
	return root, nil
}

// WorkspaceGeneratedAPIRoot returns the generated API root under
// <default-choysum-root>/generated.
func WorkspaceGeneratedAPIRoot(modulesPath string, defaultChoysumPath string) (string, error) {
	_ = modulesPath
	root, err := normalizedDefaultChoysumRoot(defaultChoysumPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "generated"), nil
}

func workspaceGeneratedAPIProtoDir(modulesPath, appName, defaultChoysumPath string) (string, error) {
	root, err := WorkspaceGeneratedAPIRoot(modulesPath, defaultChoysumPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "proto", appName), nil
}

func workspaceGeneratedAPIWebDir(modulesPath, appName, defaultChoysumPath string) (string, error) {
	root, err := WorkspaceGeneratedAPIRoot(modulesPath, defaultChoysumPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "web", appName), nil
}

func workspaceGeneratedAPIServiceDir(modulesPath, appName, defaultChoysumPath string) (string, error) {
	root, err := WorkspaceGeneratedAPIRoot(modulesPath, defaultChoysumPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "service", appName), nil
}

// WorkspaceGeneratedAPITargets returns per-app proto/web/service output directories
// under <default-choysum-root>/generated.
func WorkspaceGeneratedAPITargets(modulesPath, appName, defaultChoysumPath string) (protoDir, webDir, serviceDir string, err error) {
	root, err := WorkspaceGeneratedAPIRoot(modulesPath, defaultChoysumPath)
	if err != nil {
		return "", "", "", err
	}
	return filepath.Join(root, "proto", appName), filepath.Join(root, "web", appName), filepath.Join(root, "service", appName), nil
}
