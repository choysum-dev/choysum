// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package choysume2e

import (
	_ "embed"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

//go:embed choysume2e.js
var ChoysumE2EScript string

var (
	materializeMu   sync.Mutex
	materializedDir string
	osMkdirTemp     = os.MkdirTemp
	runtimeCaller   = runtime.Caller
)

// SourcePath returns a filesystem path to choysume2e.js for esbuild aliasing.
// Prefers the source-tree file next to this package; otherwise materializes the embed.
func SourcePath() (string, error) {
	if p, err := packageFile("choysume2e.js"); err == nil {
		return p, nil
	}
	return materializeFile("choysume2e.js", ChoysumE2EScript)
}

func packageFile(name string) (string, error) {
	_, thisFile, _, ok := runtimeCaller(0)
	if !ok {
		return "", os.ErrNotExist
	}
	p := filepath.Join(filepath.Dir(thisFile), name)
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}

func materializeFile(name, content string) (string, error) {
	materializeMu.Lock()
	defer materializeMu.Unlock()
	if materializedDir == "" {
		dir, err := osMkdirTemp("", "choysume2e-*")
		if err != nil {
			return "", err
		}
		materializedDir = dir
	}
	p := filepath.Join(materializedDir, name)
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return "", err
	}
	return p, nil
}
