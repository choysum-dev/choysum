// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv_test

import (
	"context"
	"os"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/adapter/csv"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestBuildRecordPlan_ColumnMapping(t *testing.T) {
	raw := []byte("Name,Code,Currency\nAlpha,AL1,CNY\n")
	plan, err := csv.BuildRecordPlan("base.Country", raw, map[string]string{
		"Currency": "DefaultCurrencyId/Code",
	})
	if err != nil {
		t.Fatalf("BuildRecordPlan: %v", err)
	}
	if len(plan.Units) != 1 {
		t.Fatalf("unit count = %d, want 1", len(plan.Units))
	}
	unit, ok := plan.Units[0].(recordplan.Unit)
	if !ok {
		t.Fatalf("unit type = %T", plan.Units[0])
	}
	if unit.Values["DefaultCurrencyId/Code"] != "CNY" {
		t.Fatalf("mapped value = %q", unit.Values["DefaultCurrencyId/Code"])
	}
	if unit.Values["Name"] != "Alpha" || unit.Values["Code"] != "AL1" {
		t.Fatalf("unexpected values: %#v", unit.Values)
	}
}

func TestReadTable_BOM(t *testing.T) {
	raw := append([]byte{0xEF, 0xBB, 0xBF}, []byte("Name\nX\n")...)
	table, err := csv.ReadTable(raw)
	if err != nil {
		t.Fatalf("ReadTable: %v", err)
	}
	if len(table.Headers) != 1 || table.Headers[0] != "Name" {
		t.Fatalf("headers = %#v", table.Headers)
	}
	if len(table.Rows) != 1 || table.Rows[0][0] != "X" {
		t.Fatalf("rows = %#v", table.Rows)
	}
}

func TestReadTable_WhitespaceOnlyRowSkipped(t *testing.T) {
	raw := []byte("Name,Code\nAlpha,A1\n   ,  \nBeta,B2\n")
	table, err := csv.ReadTable(raw)
	if err != nil {
		t.Fatalf("ReadTable: %v", err)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("rows = %#v, want 2 non-blank", table.Rows)
	}
	if len(table.RowNumbers) != 2 || table.RowNumbers[0] != 2 || table.RowNumbers[1] != 4 {
		t.Fatalf("RowNumbers = %#v, want [2 4]", table.RowNumbers)
	}
}

func TestBuildRecordPlan_DuplicateHeaderRejected(t *testing.T) {
	raw := []byte("Name,Name\nAlpha,Beta\n")
	_, err := csv.BuildRecordPlan("base.Country", raw, nil)
	if err == nil {
		t.Fatal("expected duplicate header error")
	}
}

func TestBuildRecordPlan_DuplicateMappedFieldRejected(t *testing.T) {
	raw := []byte("ColA,ColB\nAlpha,Beta\n")
	_, err := csv.BuildRecordPlan("base.Country", raw, map[string]string{
		"ColA": "Code",
		"ColB": "Code",
	})
	if err == nil {
		t.Fatal("expected duplicate mapped field error")
	}
}

func TestBuildRecordPlan_PreservesSourceLineNumbers(t *testing.T) {
	raw := []byte("Name,Code\nAlpha,A1\n   ,  \nBeta,B2\n")
	plan, err := csv.BuildRecordPlan("base.Country", raw, nil)
	if err != nil {
		t.Fatalf("BuildRecordPlan: %v", err)
	}
	if len(plan.Units) != 2 {
		t.Fatalf("unit count = %d", len(plan.Units))
	}
	u0, _ := plan.Units[0].(recordplan.Unit)
	u1, _ := plan.Units[1].(recordplan.Unit)
	if u0.RowNumber != 2 || u1.RowNumber != 4 {
		t.Fatalf("RowNumbers = %d,%d want 2,4", u0.RowNumber, u1.RowNumber)
	}
}

func TestReadTable_EmptyAndParseErrors(t *testing.T) {
	if _, err := csv.ReadTable(nil); err == nil {
		t.Fatal("expected empty CSV error")
	}
	if _, err := csv.ReadTable([]byte("Name,Code\n\"unterminated")); err == nil {
		t.Fatal("expected parse error")
	}
	if _, err := csv.ReadTable([]byte("Name,Code\nOnlyOne\n")); err == nil {
		t.Fatal("expected column count error")
	}
}

func TestReadTable_MultilineQuotedRecord(t *testing.T) {
	raw := []byte("Name,Code\n\"Line1\nLine2\",M1\nNext,M2\n")
	table, err := csv.ReadTable(raw)
	if err != nil {
		t.Fatalf("ReadTable: %v", err)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("rows=%#v", table.Rows)
	}
	if table.Rows[0][0] != "Line1\nLine2" {
		t.Fatalf("multiline cell=%q", table.Rows[0][0])
	}
	if table.RowNumbers[0] != 2 || table.RowNumbers[1] != 4 {
		t.Fatalf("RowNumbers=%#v want [2 4]", table.RowNumbers)
	}
}

func TestBuildRecordPlan_ExternalIDAndSkippedCells(t *testing.T) {
	raw := []byte("id,Name,Code,SkipMe\nimport.x,Alpha,,keep\n")
	plan, err := csv.BuildRecordPlan("base.Country", raw, map[string]string{"SkipMe": ""})
	if err != nil {
		t.Fatalf("BuildRecordPlan: %v", err)
	}
	unit := plan.Units[0].(recordplan.Unit)
	if unit.ExternalID != "import.x" {
		t.Fatalf("external id=%q", unit.ExternalID)
	}
	if _, ok := unit.Values["Code"]; ok {
		t.Fatalf("empty Code should be omitted: %#v", unit.Values)
	}
	if _, ok := unit.Values["SkipMe"]; ok {
		t.Fatalf("mapped-away header should be omitted: %#v", unit.Values)
	}
}

func TestBuildRecordPlan_EmptyHeaderRejected(t *testing.T) {
	if _, err := csv.BuildRecordPlan("base.Country", []byte(",Code\nA,B\n"), nil); err == nil {
		t.Fatal("expected empty header error")
	}
	if _, err := csv.BuildRecordPlan("  ", []byte("Name\nX\n"), nil); err == nil {
		t.Fatal("expected empty model error")
	}
}

func TestBuilder_BuildFromFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/ok.csv"
	if err := os.WriteFile(path, []byte("Name,Code\nA,C1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	builder := csv.Builder{}
	plan, err := builder.Build(context.Background(), importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv", Path: path},
	})
	if err != nil || len(plan.Units) != 1 {
		t.Fatalf("Build: %#v %v", plan, err)
	}
	if _, err := builder.Build(context.Background(), importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Module:  "auth",
		Source:  importpkg.Source{Path: dir},
		Options: importpkg.Options{InitdataFiles: []string{"ok.csv"}},
	}); err != nil {
		t.Fatalf("initdata Build: %v", err)
	}
	if _, err := builder.Build(context.Background(), importpkg.Spec{Profile: importpkg.ProfileTerminology, Model: "base.Country", Source: importpkg.Source{Path: path}}); err == nil {
		t.Fatal("expected profile error")
	}
	if _, err := builder.Build(context.Background(), importpkg.Spec{Profile: importpkg.ProfileRecord, Source: importpkg.Source{Path: path}}); err == nil {
		t.Fatal("expected model error")
	}
	if _, err := builder.Build(context.Background(), importpkg.Spec{Profile: importpkg.ProfileRecord, Model: "base.Country"}); err == nil {
		t.Fatal("expected path error")
	}
	if _, err := builder.Build(context.Background(), importpkg.Spec{Profile: importpkg.ProfileRecord, Model: "base.Country", Source: importpkg.Source{Path: path + ".missing"}}); err == nil {
		t.Fatal("expected read error")
	}
}
