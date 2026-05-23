// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package staging

import (
	"crypto/sha1"
	"encoding/hex"
	"path/filepath"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

// WorkspaceStagingRoot returns the centralized staging root under the configured
// tmp root.
// The root is always <default-tmp-root>/staging.
func WorkspaceStagingRoot(path string, tmpRoot string) (string, error) {
	workspaceTmpRoot, err := WorkspaceTmpRoot(path, tmpRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(workspaceTmpRoot, "staging"), nil
}

// WorkspaceTmpRoot returns the centralized tmp root for staging.
// The root is always <default-tmp-root>.
func WorkspaceTmpRoot(path string, tmpRoot string) (string, error) {
	_ = path
	return normalizeTmpRoot(tmpRoot)
}

func normalizeTmpRoot(tmpRoot string) (string, error) {
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

func stagingEntryName(kind string, targetPath string) string {
	cleaned := filepath.Clean(strings.TrimSpace(targetPath))
	base := sanitizeStagingEntry(filepath.Base(cleaned))
	if base == "" {
		base = kind
	}
	hash := sha1.Sum([]byte(kind + ":" + cleaned))
	return base + "-" + hex.EncodeToString(hash[:6])
}

func sanitizeStagingEntry(name string) string {
	if name == "" || name == "." || name == string(filepath.Separator) {
		return ""
	}

	var b strings.Builder
	b.Grow(len(name))
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '_' || ch == '-' {
			b.WriteByte(ch)
			continue
		}
		b.WriteByte('_')
	}

	return strings.Trim(b.String(), "._-")
}
