// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtimeapi

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	xfmt "golang.org/x/exp/errors/fmt"
)

// GeneratedProtoRootFromDist returns <choysum_root>/generated/proto next to dist.
func GeneratedProtoRootFromDist(distRoot string) string {
	return filepath.Join(filepath.Dir(distRoot), "generated", "proto")
}

// SyncMissingProtos restores bundle-mode runtime protobuf dirs under
// <choysum_root>/api/<app>/proto from <choysum_root>/generated/proto/<app>
// when the runtime dir is missing.
//
// This is an explicit opt-in recovery helper. Do not call it from cold-start
// planning or upgrade success paths; those must surface missing runtime protos
// so the underlying publish bug can be reproduced and fixed.
func SyncMissingProtos(distRoot string, apps []string) ([]string, error) {
	distRoot = filepath.Clean(strings.TrimSpace(distRoot))
	if distRoot == "" || distRoot == "." {
		return nil, xfmt.Errorf("dist root is empty")
	}

	generatedRoot := GeneratedProtoRootFromDist(distRoot)
	synced := make([]string, 0)
	seen := map[string]bool{}
	for _, app := range apps {
		app = strings.TrimSpace(app)
		if app == "" || strings.EqualFold(app, "web") || seen[app] || strings.ContainsAny(app, `/\`) || strings.Contains(app, "..") || app != filepath.Base(app) {
			continue
		}
		seen[app] = true

		dst := config.APIAppProtoDir(distRoot, app)
		if st, err := os.Stat(dst); err == nil && st.IsDir() {
			continue
		} else if err != nil && !os.IsNotExist(err) {
			return synced, xfmt.Errorf("stat runtime api proto %s: %w", dst, err)
		}

		src := filepath.Join(generatedRoot, app)
		if st, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(dst, 0o755); err != nil {
					return synced, xfmt.Errorf("create empty runtime api proto %s: %w", dst, err)
				}
				synced = append(synced, app)
				continue
			}
			return synced, xfmt.Errorf("stat generated proto %s: %w", src, err)
		} else if !st.IsDir() {
			return synced, xfmt.Errorf("generated proto is not a directory: %s", src)
		}

		if err := copyDirContents(src, dst); err != nil {
			return synced, xfmt.Errorf("restore runtime api proto %s from %s: %w", dst, src, err)
		}
		synced = append(synced, app)
	}
	return synced, nil
}

func copyDirContents(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
