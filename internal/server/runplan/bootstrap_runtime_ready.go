// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runplan

import (
	"context"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

func ValidateBootstrapRuntimeReady(ctx context.Context, distRoot string, compileBundleMode string) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return xfmt.Errorf("bootstrap runtime readiness canceled: %w", err)
		}
	}

	distRoot = strings.TrimSpace(distRoot)
	if distRoot == "" {
		return xfmt.Errorf("dist path is not configured")
	}

	manifest, err := LoadDistManifest(distRoot)
	if err != nil {
		return xfmt.Errorf("load dist manifest: %w", err)
	}
	if manifest != nil && strings.TrimSpace(manifest.CompileBundleMode) != "" {
		compileBundleMode = manifest.CompileBundleMode
	}

	return ValidateDistForTargets(compileBundleMode, distRoot, []string{"auth", "web"})
}
