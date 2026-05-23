// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package web

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

const (
	// EnvBootstrapWebSource selects whether bootstrap web assets are served from embedded files or disk.
	EnvBootstrapWebSource = "CHOYSUM_BOOTSTRAP_WEB_SOURCE"
	// EnvBootstrapWebDistDir overrides the disk dist directory used when EnvBootstrapWebSource is set to "disk".
	EnvBootstrapWebDistDir = "CHOYSUM_BOOTSTRAP_WEB_DIST_DIR"
)

var (
	//go:embed dist
	embeddedDistFS embed.FS
)

// LoadDistFS loads the bootstrap web dist filesystem and reports which source was selected.
func LoadDistFS(source string) (fs.FS, string, string, error) {
	normalizedSource := strings.ToLower(strings.TrimSpace(source))
	if normalizedSource == "" {
		normalizedSource = "embed"
	}

	switch normalizedSource {
	case "embed":
		distFS, err := fs.Sub(embeddedDistFS, "dist")
		if err != nil {
			return nil, "", "", xfmt.Errorf("failed to load embedded bootstrap web dist: %w", err)
		}
		return distFS, "embed", "embedded:dist", nil
	case "disk":
		distDir := strings.TrimSpace(os.Getenv(EnvBootstrapWebDistDir))
		if distDir == "" {
			distDir = defaultDiskDistDir()
		}

		st, err := os.Stat(distDir)
		if err != nil {
			return nil, "", "", xfmt.Errorf("bootstrap web disk dist missing: %s: %w", distDir, err)
		}
		if !st.IsDir() {
			return nil, "", "", xfmt.Errorf("bootstrap web disk dist is not a directory: %s", distDir)
		}

		return os.DirFS(distDir), "disk", distDir, nil
	default:
		return nil, "", "", xfmt.Errorf("invalid %s value %q (allowed: embed | disk)", EnvBootstrapWebSource, source)
	}
}

func defaultDiskDistDir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Clean(filepath.FromSlash("internal/bootstrap/web/dist"))
	}
	return filepath.Join(filepath.Dir(thisFile), "dist")
}
