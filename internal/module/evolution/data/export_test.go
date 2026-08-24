// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/scope"
)

// NewDefaultLoaderScopeForTest exposes the loader integration test scope to external packages.
func NewDefaultLoaderScopeForTest(t *testing.T) scope.Scope {
	t.Helper()
	return newDefaultLoaderScope(t)
}

// WriteDataFileForTest writes a bootstrap JSON file for loader/import equivalence tests.
func WriteDataFileForTest(t *testing.T, dir string, df any) {
	t.Helper()
	writeDataFile(t, dir, df)
}
