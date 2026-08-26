// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	csvsink "github.com/choysum-dev/choysum/internal/export/sink/csv"
	importcsv "github.com/choysum-dev/choysum/internal/import/adapter/csv"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestCSV_BOM(t *testing.T) {
	result := &registry.Result{
		Headers: []string{"Name", "Code"},
		Rows:    [][]string{{"Alpha", "A1"}},
	}
	w := csvsink.Writer{}
	if err := w.Write(context.Background(), nil, exportplan.Plan{Mode: exportpkg.ModeData}, result); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !bytes.HasPrefix(result.CSVBytes, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("expected UTF-8 BOM prefix")
	}
	table, err := csv.NewReader(strings.NewReader(string(importcsv.StripUTF8BOM(result.CSVBytes)))).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(table) != 2 || table[0][0] != "Name" || table[1][1] != "A1" {
		t.Fatalf("table = %#v", table)
	}
}

func TestModeTemplate_HeaderOnly(t *testing.T) {
	result := &registry.Result{
		Headers: []string{"Name", "Code"},
		Rows:    [][]string{{"should", "skip"}},
	}
	w := csvsink.Writer{}
	if err := w.Write(context.Background(), nil, exportplan.Plan{Mode: exportpkg.ModeTemplate}, result); err != nil {
		t.Fatalf("Write: %v", err)
	}
	table, err := csv.NewReader(strings.NewReader(string(importcsv.StripUTF8BOM(result.CSVBytes)))).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(table) != 1 || table[0][0] != "Name" {
		t.Fatalf("template table = %#v", table)
	}
}
