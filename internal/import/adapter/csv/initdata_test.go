// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv_test

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/import/adapter/csv"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestBuildInitdataPlan_IdDefaultModule(t *testing.T) {
	raw := []byte("id,model,Code\ngroup_import,group,G1\n")
	recs, err := csv.BuildInitdataPlanFromCSV(raw, "auth")
	if err != nil {
		t.Fatalf("BuildInitdataPlanFromCSV: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("len = %d, want 1", len(recs))
	}
	if recs[0].Module != "auth" || recs[0].Name != "group_import" {
		t.Fatalf("id split = %s.%s, want auth.group_import", recs[0].Module, recs[0].Name)
	}
	if recs[0].Model != "group" {
		t.Fatalf("model = %q", recs[0].Model)
	}
	if recs[0].Values["Code"] != "G1" {
		t.Fatalf("values = %#v", recs[0].Values)
	}
	if recs[0].Values == nil {
		t.Fatal("values must be non-nil")
	}
}

func TestBuildInitdataPlan_ModelColumn(t *testing.T) {
	raw := []byte("id,model,noupdate\nauth.group_import,group,true\n")
	recs, err := csv.BuildInitdataPlanFromCSV(raw, "ignored")
	if err != nil {
		t.Fatalf("BuildInitdataPlanFromCSV: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("len = %d", len(recs))
	}
	if recs[0].Module != "auth" || recs[0].Name != "group_import" {
		t.Fatalf("got %s.%s", recs[0].Module, recs[0].Name)
	}
	if recs[0].NoUpdate == nil || !*recs[0].NoUpdate {
		t.Fatalf("noupdate = %#v, want true", recs[0].NoUpdate)
	}
}

func TestBuildInitdataPlan_RequiresIdAndModel(t *testing.T) {
	_, err := csv.BuildInitdataPlanFromCSV([]byte("name,model\nx,group\n"), "auth")
	if err == nil {
		t.Fatal("expected missing id column error")
	}
	impErr, ok := importpkg.AsError(err)
	if !ok || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("error = %v", err)
	}
}

func TestReadTable_InvalidUTF8(t *testing.T) {
	_, err := csv.ReadTable([]byte{0xE9, ',', 'm', '\n', 'a', ',', 'b', '\n'})
	if err == nil {
		t.Fatal("expected invalid UTF-8 error")
	}
	impErr, ok := importpkg.AsError(err)
	if !ok || impErr.Code != importpkg.CodeInvalidEncoding {
		t.Fatalf("error = %v, want CodeInvalidEncoding", err)
	}
}
