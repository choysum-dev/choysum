// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

// BuildOptions configures BuildPlan behavior beyond the op/root/resolver inputs.
type BuildOptions struct {
	// SkipWebShell disables auto-including the web shell when a planned module
	// declares entryPoints.web (CLI --no-web).
	SkipWebShell bool
}

// BuildOption mutates BuildOptions.
type BuildOption func(*BuildOptions)

// WithSkipWebShell sets SkipWebShell.
func WithSkipWebShell(skip bool) BuildOption {
	return func(o *BuildOptions) {
		if o == nil {
			return
		}
		o.SkipWebShell = skip
	}
}

func applyBuildOptions(opts []BuildOption) BuildOptions {
	out := BuildOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&out)
		}
	}
	return out
}
