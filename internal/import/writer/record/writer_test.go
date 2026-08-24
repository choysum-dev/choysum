// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record_test

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/plan"
	planstub "github.com/choysum-dev/choysum/internal/import/plan/stub"
	recordwriter "github.com/choysum-dev/choysum/internal/import/writer/record"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestWriter_WriteUnexpectedUnitType(t *testing.T) {
	err := recordwriter.Writer{}.Write(context.Background(), nil, []plan.Unit{planstub.Unit{Index: 1}})
	impErr, ok := importpkg.AsError(err)
	if !ok || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("Write() error = %v, want CodeInvalidFormat", err)
	}
}

func TestWithImportFileContext(t *testing.T) {
	if recordwriter.ImportFileFromContext(nil) {
		t.Fatal("expected false for nil context")
	}
}
