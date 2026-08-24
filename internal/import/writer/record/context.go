// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"context"

	"github.com/choysum-dev/choysum/pkg/scope"
)

type importFileContextKey struct{}

// WithImportFileContext marks the scope context for ORM import writes.
func WithImportFileContext(txScope scope.Scope) scope.Scope {
	if txScope == nil {
		return txScope
	}
	ctx := txScope.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if ImportFileFromContext(ctx) {
		return txScope
	}
	return txScope.WithContext(context.WithValue(ctx, importFileContextKey{}, true))
}

// ImportFileFromContext reports whether import_file was set on the context.
func ImportFileFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, ok := ctx.Value(importFileContextKey{}).(bool)
	return ok && v
}
