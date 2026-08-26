// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv

import (
	"context"
	"encoding/csv"
	"errors"
	"testing"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestWrite_nilResult(t *testing.T) {
	w := Writer{}
	if err := w.Write(context.Background(), nil, exportplan.Plan{}, nil); err == nil {
		t.Fatal("expected nil result error")
	}
}

func TestWrite_missingHeaders(t *testing.T) {
	w := Writer{}
	result := &registry.Result{}
	if err := w.Write(context.Background(), nil, exportplan.Plan{}, result); err == nil {
		t.Fatal("expected missing headers error")
	}
}

func TestWrite_invalidUTF8(t *testing.T) {
	orig := validateUTF8
	t.Cleanup(func() { validateUTF8 = orig })
	validateUTF8 = func([]byte) error {
		return errors.New("bad utf8")
	}

	w := Writer{}
	result := &registry.Result{Headers: []string{"Name"}, Rows: [][]string{{"A"}}}
	if err := w.Write(context.Background(), nil, exportplan.Plan{Mode: exportpkg.ModeData}, result); err == nil {
		t.Fatal("expected utf8 validation error")
	}
}

func TestWrite_headerError(t *testing.T) {
	orig := writeCSVRecord
	t.Cleanup(func() { writeCSVRecord = orig })
	writeCSVRecord = func(*csv.Writer, []string) error {
		return errors.New("header failed")
	}

	w := Writer{}
	result := &registry.Result{Headers: []string{"Name"}, Rows: [][]string{{"A"}}}
	if err := w.Write(context.Background(), nil, exportplan.Plan{Mode: exportpkg.ModeData}, result); err == nil {
		t.Fatal("expected header write error")
	}
}

func TestWrite_rowError(t *testing.T) {
	calls := 0
	orig := writeCSVRecord
	t.Cleanup(func() { writeCSVRecord = orig })
	writeCSVRecord = func(w *csv.Writer, record []string) error {
		calls++
		if calls == 1 {
			return orig(w, record)
		}
		return errors.New("row failed")
	}

	w := Writer{}
	result := &registry.Result{Headers: []string{"Name"}, Rows: [][]string{{"A"}}}
	if err := w.Write(context.Background(), nil, exportplan.Plan{Mode: exportpkg.ModeData}, result); err == nil {
		t.Fatal("expected row write error")
	}
}

func TestWrite_flushError(t *testing.T) {
	orig := csvWriterError
	t.Cleanup(func() { csvWriterError = orig })
	csvWriterError = func(*csv.Writer) error {
		return errors.New("flush failed")
	}

	w := Writer{}
	result := &registry.Result{Headers: []string{"Name"}, Rows: [][]string{{"A"}}}
	if err := w.Write(context.Background(), nil, exportplan.Plan{Mode: exportpkg.ModeData}, result); err == nil {
		t.Fatal("expected flush error")
	}
}
