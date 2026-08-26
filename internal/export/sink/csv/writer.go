// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv

import (
	"bytes"
	"context"
	"encoding/csv"

	"github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	importcsv "github.com/choysum-dev/choysum/internal/import/adapter/csv"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Writer serializes record export rows as UTF-8 CSV with BOM.
type Writer struct{}

// Write implements registry.Sink.
func (Writer) Write(ctx context.Context, runtimeScope scope.Scope, p plan.Plan, result *registry.Result) error {
	_ = ctx
	_ = runtimeScope
	if result == nil {
		return exportpkg.Errorf(exportpkg.CodeInvalidFormat, "export result is required")
	}
	if len(result.Headers) == 0 {
		return exportpkg.Errorf(exportpkg.CodeInvalidFormat, "export headers are required")
	}

	var buf bytes.Buffer
	w := newCSVWriter(&buf)
	if err := writeCSVRecord(w, result.Headers); err != nil {
		return exportpkg.ErrorfWrap(exportpkg.CodeInvalidFormat, "write csv header", err)
	}
	if p.Mode != exportpkg.ModeTemplate {
		for _, row := range result.Rows {
			if err := writeCSVRecord(w, sanitizeCSVRecord(row)); err != nil {
				return exportpkg.ErrorfWrap(exportpkg.CodeInvalidFormat, "write csv row", err)
			}
		}
	}
	flushCSVWriter(w)
	if err := csvWriterError(w); err != nil {
		return exportpkg.ErrorfWrap(exportpkg.CodeInvalidFormat, "flush csv", err)
	}

	raw := buf.Bytes()
	if err := validateUTF8(raw); err != nil {
		return err
	}
	result.CSVBytes = importcsv.PrependUTF8BOM(raw)
	return nil
}

var (
	validateUTF8   = importcsv.ValidateUTF8
	newCSVWriter   = csv.NewWriter
	writeCSVRecord = func(w *csv.Writer, record []string) error { return w.Write(record) }
	flushCSVWriter = func(w *csv.Writer) { w.Flush() }
	csvWriterError = func(w *csv.Writer) error { return w.Error() }
)

func sanitizeCSVRecord(record []string) []string {
	out := make([]string, len(record))
	for i, cell := range record {
		out[i] = importcsv.SanitizeSpreadsheetCell(cell)
	}
	return out
}
