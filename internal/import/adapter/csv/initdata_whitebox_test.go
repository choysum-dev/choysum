// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv

import (
	"testing"

	initdataplan "github.com/choysum-dev/choysum/internal/import/plan/initdata"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestBuildInitdataPlanFromCSV_ReadTableError(t *testing.T) {
	_, err := BuildInitdataPlanFromCSV([]byte{0xff}, "auth")
	if err == nil {
		t.Fatal("expected encoding error")
	}
}

func TestBuildInitdataPlanFromCSV_RowErrorUsesRowNumber(t *testing.T) {
	raw := []byte("id,model\n,group\n")
	_, err := BuildInitdataPlanFromCSV(raw, "auth")
	if err == nil {
		t.Fatal("expected empty id error")
	}
	if !containsImportCode(err, importpkg.CodeInvalidFormat) {
		t.Fatalf("error = %v", err)
	}
}

func TestMapInitdataHeaders_Branches(t *testing.T) {
	if _, err := mapInitdataHeaders([]string{"", "model"}); err == nil {
		t.Fatal("empty header")
	}
	if _, err := mapInitdataHeaders([]string{"id", "ID", "model"}); err == nil {
		t.Fatal("duplicate header")
	}
	if _, err := mapInitdataHeaders([]string{"id", "Code"}); err == nil {
		t.Fatal("missing model")
	}
	idx, err := mapInitdataHeaders([]string{" ID ", "Model", "Code"})
	if err != nil {
		t.Fatal(err)
	}
	if idx["id"] != 0 || idx["model"] != 1 || idx["code"] != 2 {
		t.Fatalf("idx=%#v", idx)
	}
}

func TestRecordFromCSVRow_Branches(t *testing.T) {
	headers := []string{"id", "model", "application", "module", "noupdate", "Code", "Skip"}
	col, err := mapInitdataHeaders(headers)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := recordFromCSVRow(headers, []string{"", "group", "", "", "", "", ""}, col, "auth"); err == nil {
		t.Fatal("empty id")
	}
	if _, err := recordFromCSVRow(headers, []string{".onlyname", "group", "", "", "", "", ""}, col, "auth"); err == nil {
		t.Fatal("invalid id module empty")
	}
	if _, err := recordFromCSVRow(headers, []string{"auth.", "group", "", "", "", "", ""}, col, "auth"); err == nil {
		t.Fatal("invalid id name empty")
	}
	if _, err := recordFromCSVRow(headers, []string{"auth.a.b", "group", "", "", "", "", ""}, col, "auth"); err == nil {
		t.Fatal("invalid id extra dot")
	}
	if _, err := recordFromCSVRow(headers, []string{"bare", "group", "", "", "", "", ""}, col, ""); err == nil {
		t.Fatal("bare id without applying module")
	}
	if _, err := recordFromCSVRow(headers, []string{"auth.g1", "", "", "", "", "", ""}, col, "auth"); err == nil {
		t.Fatal("empty model")
	}
	if _, err := recordFromCSVRow(headers, []string{"auth.g1", "group", "", "", "maybe", "", ""}, col, "auth"); err == nil {
		t.Fatal("bad noupdate")
	}

	rec, err := recordFromCSVRow(headers, []string{"bare", "group", "auth", "auth", "0", "C1", ""}, col, "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Module != "auth" || rec.Name != "bare" || rec.Application != "auth" {
		t.Fatalf("rec=%#v", rec)
	}
	if rec.NoUpdate == nil || *rec.NoUpdate {
		t.Fatalf("noupdate=%v", rec.NoUpdate)
	}
	if rec.Values["Code"] != "C1" {
		t.Fatalf("values=%#v", rec.Values)
	}
	if _, ok := rec.Values["Skip"]; ok {
		t.Fatal("empty value column should be skipped")
	}

	// explicit module column overrides id module; empty module column ignored
	rec2, err := recordFromCSVRow(headers, []string{"other.g2", "group", "", "", "yes", "", ""}, col, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if rec2.Module != "other" || rec2.NoUpdate == nil || !*rec2.NoUpdate {
		t.Fatalf("rec2=%#v", rec2)
	}
	rec3, err := recordFromCSVRow(headers, []string{"other.g3", "group", "", "  ", "n", "", ""}, col, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if rec3.Module != "other" || rec3.NoUpdate == nil || *rec3.NoUpdate {
		t.Fatalf("rec3=%#v", rec3)
	}
}

func TestSplitExternalID_EmptyID(t *testing.T) {
	if _, _, err := splitExternalID("   ", "auth"); err == nil {
		t.Fatal("expected empty id error")
	}
}

func TestParseCSVBool_AllBranches(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"1", true}, {"TRUE", true}, {"Yes", true}, {"y", true},
		{"0", false}, {"FALSE", false}, {"No", false}, {"n", false},
	}
	for _, tc := range cases {
		got, err := parseCSVBool(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("%q => %v %v, want %v", tc.in, got, err, tc.want)
		}
	}
	// strconv.ParseBool path for values not in the switch after ToLower — use a
	// string ParseBool accepts that our switch already covers via ToLower; force
	// default error instead:
	if _, err := parseCSVBool("not-a-bool"); err == nil {
		t.Fatal("expected invalid bool")
	}
	// Cover ParseBool success in default: "t" / "f" are accepted by strconv but
	// not listed in the switch cases above as separate arms after ToLower.
	if got, err := parseCSVBool("t"); err != nil || !got {
		t.Fatalf("t => %v %v", got, err)
	}
	if got, err := parseCSVBool("f"); err != nil || got {
		t.Fatalf("f => %v %v", got, err)
	}
}

func TestBuildInitdataFilePlan_Branches(t *testing.T) {
	if _, err := buildInitdataFilePlan(importpkg.Spec{Module: "auth"}); err == nil {
		t.Fatal("missing path")
	}
	if _, err := buildInitdataFilePlan(importpkg.Spec{Source: importpkg.Source{Path: "/m"}}); err == nil {
		t.Fatal("missing module")
	}

	p, err := buildInitdataFilePlan(importpkg.Spec{
		Module:      "auth",
		Application: "auth",
		Source:      importpkg.Source{Path: "/m"},
		Options: importpkg.Options{
			InitdataFiles: []string{"a.csv", " ", ""},
			WithDemo:      true,
			DemoFiles:     []string{"d.csv", ""},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Units) != 2 {
		t.Fatalf("units=%d", len(p.Units))
	}
	u0 := p.Units[0].(initdataplan.Unit)
	if len(u0.Files) != 1 || u0.Files[0] != "a.csv" {
		t.Fatalf("init files=%#v", u0.Files)
	}
	u1 := p.Units[1].(initdataplan.Unit)
	if len(u1.Files) != 1 || u1.Files[0] != "d.csv" || u1.Index != 2 {
		t.Fatalf("demo unit=%#v", u1)
	}

	p2, err := buildInitdataFilePlan(importpkg.Spec{
		Module: "auth",
		Source: importpkg.Source{Path: "/m"},
		Options: importpkg.Options{
			WithDemo:  true,
			DemoFiles: nil,
		},
	})
	if err != nil || len(p2.Units) != 0 {
		t.Fatalf("empty plan=%#v err=%v", p2, err)
	}
}

func TestCleanInitdataPaths_Branches(t *testing.T) {
	if cleanInitdataPaths(nil) != nil {
		t.Fatal("nil")
	}
	if cleanInitdataPaths([]string{}) != nil {
		t.Fatal("empty")
	}
	got := cleanInitdataPaths([]string{" ", "a.csv", ""})
	if len(got) != 1 || got[0] != "a.csv" {
		t.Fatalf("got=%#v", got)
	}
}

func containsImportCode(err error, code string) bool {
	impErr, ok := importpkg.AsError(err)
	return ok && impErr.Code == code
}
