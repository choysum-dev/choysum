// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package po

import (
	"bytes"
	"context"

	"github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	i18npo "github.com/choysum-dev/choysum/internal/i18n/po"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Writer serializes terminology export entries as PO text.
type Writer struct{}

// Write implements registry.Sink.
func (Writer) Write(ctx context.Context, runtimeScope scope.Scope, p plan.Plan, result *registry.Result) error {
	_ = ctx
	_ = runtimeScope
	_ = p
	if result == nil {
		return exportpkg.Errorf(exportpkg.CodeInvalidFormat, "export result is required")
	}
	if len(result.POEntries) == 0 {
		return exportpkg.Errorf(exportpkg.CodeInvalidFormat, "export PO entries are required")
	}

	var buf bytes.Buffer
	if err := i18npo.Write(&buf, result.POEntries); err != nil {
		return exportpkg.ErrorfWrap(exportpkg.CodeInvalidFormat, "write po", err)
	}
	result.POBytes = buf.Bytes()
	return nil
}
