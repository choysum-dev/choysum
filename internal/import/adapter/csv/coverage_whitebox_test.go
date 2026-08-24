// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv

import (
	"encoding/csv"
	"strings"
	"testing"
)

func TestCSVReadErrorText_Branches(t *testing.T) {
	if csvReadErrorText(assertErr{}) != "read CSV row" {
		t.Fatal("non-parse error")
	}
	pe := &csv.ParseError{StartLine: 0, Line: 7, Err: csv.ErrFieldCount}
	if got := csvReadErrorText(pe); got != "read CSV row 7" {
		t.Fatalf("got %q", got)
	}
	pe = &csv.ParseError{StartLine: 3, Line: 7, Err: csv.ErrFieldCount}
	if got := csvReadErrorText(pe); got != "read CSV row 3" {
		t.Fatalf("got %q", got)
	}
	pe = &csv.ParseError{StartLine: 0, Line: 0, Err: csv.ErrFieldCount}
	if got := csvReadErrorText(pe); got != "read CSV row" {
		t.Fatalf("got %q", got)
	}
}

func TestReadTable_HeaderReadError(t *testing.T) {
	// invalid UTF-8 already covered; force header parse error with only quotes
	_, err := ReadTable([]byte("\"unterminated"))
	if err == nil {
		t.Fatal("expected header error")
	}
}

func TestBuildRecordPlan_ReadTableError(t *testing.T) {
	_, err := BuildRecordPlan("base.Country", []byte{0xff}, nil)
	if err == nil {
		t.Fatal("expected encoding error")
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "x" }

func TestReadTable_EmptyLineSkippedViaFieldPos(t *testing.T) {
	// encoding/csv may treat bare newline as empty record
	raw := []byte("Name,Code\nA,1\n\nB,2\n")
	table, err := ReadTable(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("rows=%#v", table.Rows)
	}
	// physical lines: header1, A2, empty3?, B...
	if table.RowNumbers[0] != 2 {
		t.Fatalf("first=%d", table.RowNumbers[0])
	}
	_ = strings.Builder{}
}
