// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

// Scope selects which roots Check includes.
type Scope int

const (
	// ScopeService checks modules/<app>/*.ts and modules/<app>/service/**/*.ts
	// plus shared .d.ts under the app (excluding web/ and test trees).
	ScopeService Scope = iota
	// ScopeNoVue checks ScopeService roots plus modules/<app>/web/**/*.{ts,tsx}
	// and web .d.ts. It skips .vue files.
	ScopeNoVue
)

// Options configures a Check run for a single application.
type Options struct {
	ModulesPath string // modules root (absolute preferred)
	RepoRoot    string
	App         string
	Scope       Scope // zero value is ScopeService
	KeepDir     string
	Overlays    map[string]string // optional slash-path → content overrides
}

// CodegenOptions is reserved for Vue service-script generation.
type CodegenOptions struct {
	CurrentDirectory string
}
