// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

func (c *coordinator) defaultValidateRuntimeReady(ctx context.Context) error {
	_ = ctx
	opts := runtimeOptionsFromScope(c.runtimeScope)
	if strings.TrimSpace(opts.distPath) == "" {
		return newBootstrapError(bootstrapErrCodeRuntimeNotReady, "system configuration is not available", nil)
	}

	distRoot := strings.TrimSpace(opts.distPath)

	st, err := os.Stat(distRoot)
	if err != nil {
		return newBootstrapError(bootstrapErrCodeRuntimeNotReady, "required system files are not accessible", err)
	}
	if !st.IsDir() {
		return newBootstrapError(bootstrapErrCodeRuntimeNotReady, "required system files are misconfigured", nil)
	}

	webIndex := filepath.Join(distRoot, "web", "index.html")
	if _, err := os.Stat(webIndex); err != nil {
		return newBootstrapError(bootstrapErrCodeRuntimeNotReady, "web interface files are not ready", err)
	}

	return nil
}
