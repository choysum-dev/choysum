// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/buke/typescript-go-internal/v7/pkg/bundled"
	"github.com/buke/typescript-go-internal/v7/pkg/compiler"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs/osvfs"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs/wrapvfs"
)

// esm.sh type-fetch .d.ts files rewrite module augmentations to
// declare module 'https://esm.sh/<pkg>@ver/...' which does not merge into
// package-name modules. Map those IDs back carefully:
//
//	https://esm.sh/@vue/runtime-core@3.5.35/dist/runtime-core.d.ts → @vue/runtime-core
//	https://esm.sh/dayjs@1.11.21/locale/* → dayjs/locale/*  (keep subpath!)
var esmShDeclareModuleRE = regexp.MustCompile(
	`declare module ['"]https://esm\.sh/(?:v[0-9]+/)?((?:@[^/'"@]+/)?[^/'"@]+)(?:@[^/'"]+)?(/[^'"]*)?['"]`,
)

func rewriteEsmShDeclareModules(content string) string {
	if !strings.Contains(content, "https://esm.sh/") || !strings.Contains(content, "declare module") {
		return content
	}
	return esmShDeclareModuleRE.ReplaceAllStringFunc(content, func(full string) string {
		m := esmShDeclareModuleRE.FindStringSubmatch(full)
		pkg, sub := m[1], m[2]
		mod := esmShURLToModuleID(pkg, sub)
		return "declare module '" + mod + "'"
	})
}

// esmShURLToModuleID maps an esm.sh declare-module URL to a TypeScript module id.
func esmShURLToModuleID(pkg, sub string) string {
	if sub == "" || sub == "/" {
		return pkg
	}
	if isEsmShPackageMainTypePath(pkg, sub) {
		return pkg
	}
	mod := pkg + sub
	mod = strings.TrimSuffix(mod, ".d.ts")
	mod = strings.TrimSuffix(mod, ".d.mts")
	return mod
}

func isEsmShPackageMainTypePath(pkg, sub string) bool {
	clean := path.Clean(sub)
	dir := path.Dir(clean)
	base := path.Base(clean)
	name := strings.TrimSuffix(strings.TrimSuffix(base, ".d.ts"), ".d.mts")
	if name == "*" || name == "" {
		return false
	}
	pkgBase := pkg
	if i := strings.LastIndex(pkg, "/"); i >= 0 {
		pkgBase = pkg[i+1:]
	}
	if name == pkgBase {
		return true
	}
	// Only treat package-root index files as the main entry — nested
	// …/locale/index.d.ts must keep its subpath module id.
	if name == "index" {
		switch dir {
		case "/", "/dist", "/types", "/lib", "/dist/types":
			return true
		}
	}
	return false
}

func isEsmShTypeFetchPath(p string) bool {
	p = filepath.ToSlash(p)
	// Any type-fetch cache file under pkg/types (esm.sh_* entries and package
	// caches such as vue@ver.d.ts) may contain declare module 'https://esm.sh/…'.
	return strings.Contains(p, "/pkg/types/")
}

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
				return rewriteEsmShDeclareModules(content), true
			}
			content, ok := base.ReadFile(p)
			if !ok {
				return "", false
			}
			if isEsmShTypeFetchPath(p) {
				content = rewriteEsmShDeclareModules(content)
			}
			return content, true
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
