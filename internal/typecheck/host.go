// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"path/filepath"
	"strings"

	"github.com/buke/typescript-go-internal/v7/pkg/bundled"
	"github.com/buke/typescript-go-internal/v7/pkg/compiler"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs/osvfs"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs/wrapvfs"
)

func newTypecheckFS(overlays map[string]string) vfs.FS {
	base := osvfs.FS()
	caseSensitive := base.UseCaseSensitiveFileNames()
	normalized := normalizeOverlayMap(overlays)
	overlay := wrapvfs.Wrap(base, wrapvfs.Replacements{
		FileExists: func(p string) bool {
			if _, ok := lookupOverlay(normalized, p, caseSensitive); ok {
				return true
			}
			return base.FileExists(p)
		},
		ReadFile: func(p string) (string, bool) {
			if content, ok := lookupOverlay(normalized, p, caseSensitive); ok {
				return content, true
			}
			return base.ReadFile(p)
		},
	})
	return bundled.WrapFS(overlay)
}

func newHost(currentDir string, fs vfs.FS) compiler.CompilerHost {
	currentDir = filepath.ToSlash(filepath.Clean(currentDir))
	if currentDir == "" || currentDir == "." {
		currentDir = "/"
	}
	return compiler.NewCompilerHost(currentDir, fs, bundled.LibPath(), nil, nil)
}

func normalizeOverlayMap(overlays map[string]string) map[string]string {
	if len(overlays) == 0 {
		return nil
	}
	out := make(map[string]string, len(overlays))
	for k, v := range overlays {
		key := normalizePathKey(k)
		if key == "" {
			continue
		}
		out[key] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func lookupOverlay(overlays map[string]string, requestPath string, caseSensitive bool) (string, bool) {
	if len(overlays) == 0 {
		return "", false
	}
	key := normalizePathKey(requestPath)
	if key == "" {
		return "", false
	}
	if content, ok := overlays[key]; ok {
		return content, true
	}
	if caseSensitive {
		return "", false
	}
	for k, v := range overlays {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
}

func normalizePathKey(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := absPath(path); err == nil {
		path = abs
	}
	return filepath.ToSlash(filepath.Clean(path))
}
