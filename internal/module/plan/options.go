// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import (
	"fmt"
	"strings"
)

// WebShellProfile selects optional modules Ensure'd after the web SPA shell.
//
//	minimal  — web depends only (core/auth/meta); no document/message/audit
//	standard — also Ensure document (default for CLI install/upgrade)
//	full     — also Ensure document + message + audit
type WebShellProfile string

const (
	WebShellProfileMinimal  WebShellProfile = "minimal"
	WebShellProfileStandard WebShellProfile = "standard"
	WebShellProfileFull     WebShellProfile = "full"
)

// BuildOptions configures BuildPlan behavior beyond the op/root/resolver inputs.
type BuildOptions struct {
	// SkipWebShell disables auto-including the web SPA shell when a planned module
	// declares entryPoints.web (CLI --no-web). When web is not in ModuleOrder it
	// also clears NeedsGlobalWebBuild; installing module "web" itself still builds dist/web.
	SkipWebShell bool
	// WebShellProfile selects optional shell extras (document / collaboration).
	// Empty means standard.
	WebShellProfile WebShellProfile
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

// WithWebShellProfile sets WebShellProfile (empty / unknown values are rejected by BuildPlan).
func WithWebShellProfile(profile WebShellProfile) BuildOption {
	return func(o *BuildOptions) {
		if o == nil {
			return
		}
		o.WebShellProfile = profile
	}
}

// ParseWebShellProfile accepts minimal|standard|full (case-insensitive).
// Empty string means standard.
func ParseWebShellProfile(raw string) (WebShellProfile, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(WebShellProfileStandard):
		return WebShellProfileStandard, nil
	case string(WebShellProfileMinimal):
		return WebShellProfileMinimal, nil
	case string(WebShellProfileFull):
		return WebShellProfileFull, nil
	default:
		return "", fmt.Errorf("invalid web-shell-profile %q (want minimal|standard|full)", raw)
	}
}

func normalizeWebShellProfile(profile WebShellProfile) (WebShellProfile, error) {
	return ParseWebShellProfile(string(profile))
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
