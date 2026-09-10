// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	_ "embed"
)

//go:embed ambient/vite_client.d.ts
var viteClientDTS string

//go:embed ambient/vue_shim.d.ts
var vueShimDTS string

//go:embed ambient/vue_directives.d.ts
var vueDirectivesDTS string

//go:embed ambient/vue_module_stub.d.ts
var vueModuleStubDTS string

//go:embed ambient/playwright_test.d.ts
var playwrightTestDTS string

// ViteClientOverlay returns the relative ambient path (under the typecheck
// ambient root) and embedded vite/client declarations.
func ViteClientOverlay() (relPath, content string) {
	return "vite/client.d.ts", viteClientDTS
}

// PlaywrightTestOverlay returns a loose @playwright/test module declaration for
// legacy e2e specs still imported during the Playwright → @choysum/e2e cutover.
func PlaywrightTestOverlay() (relPath, content string) {
	return "playwright_test.d.ts", playwrightTestDTS
}

// VueShimOverlay returns vue/jsx-runtime ambient (does not declare module "vue").
func VueShimOverlay() (relPath, content string) {
	return "vue-shim.d.ts", vueShimDTS
}

// VueDirectivesOverlay returns vue GlobalComponents / GlobalDirectives
// augmentation used when type-fetch relative loads break upstream module
// augmentation (and when element-plus/global.d.ts is unavailable).
func VueDirectivesOverlay() (relPath, content string) {
	return "vue-directives.d.ts", vueDirectivesDTS
}

// VueModuleStubOverlay returns a minimal declare module "vue" for fixtures
// when real Vue package types are unavailable.
func VueModuleStubOverlay() (relPath, content string) {
	return "vue-module-stub.d.ts", vueModuleStubDTS
}
