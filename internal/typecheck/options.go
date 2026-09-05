// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import "github.com/choysum-dev/choysum/internal/typecheck/vue"

// Scope selects which roots Check includes.
type Scope int

const (
	// ScopeService checks modules/<app>/*.ts and modules/<app>/service/**/*.ts
	// plus shared .d.ts under the app (excluding web/ and test trees).
	ScopeService Scope = iota
	// ScopeNoVue checks ScopeService roots plus modules/<app>/web/**/*.{ts,tsx}
	// and web .d.ts. It skips .vue files.
	ScopeNoVue
	// ScopeAll checks ScopeNoVue roots plus modules/<app>/web/**/*.vue
	// (Host serves language-core service scripts via Coder overlays).
	ScopeAll
)

// Options configures a Check run for a single application.
type Options struct {
	ModulesPath string // modules root (absolute preferred)
	RepoRoot    string
	App         string
	Scope       Scope // zero value is ScopeService
	KeepDir     string
	Overlays    map[string]string // optional slash-path → content overrides

	// Coder produces Vue service scripts for ScopeAll. When nil and ScopeAll,
	// Check constructs vue.NewGoldenCoder(VueGoldenDir).
	Coder vue.Coder
	// VueGoldenDir is the directory of committed *.vue.service.ts goldens.
	VueGoldenDir string
}

// CodegenOptions is reserved for Vue service-script generation.
type CodegenOptions struct {
	CurrentDirectory string
}
