// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	_ "embed"
	"path/filepath"
)

//go:embed ambient/vite_client.d.ts
var viteClientDTS string

// ViteClientOverlay returns the relative ambient path (under the typecheck
// ambient root) and embedded vite/client declarations.
func ViteClientOverlay() (relPath, content string) {
	return filepath.ToSlash(filepath.Join("vite", "client.d.ts")), viteClientDTS
}
