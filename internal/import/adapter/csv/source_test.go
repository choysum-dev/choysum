// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/plan"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	stubplan "github.com/choysum-dev/choysum/internal/import/plan/stub"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestReadSourceBytes_Path(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.csv")
	if err := os.WriteFile(path, []byte("Name,Code\nA,1\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	raw, err := readSourceBytes(context.Background(), importpkg.Spec{
		Source: importpkg.Source{Path: path},
	})
	if err != nil {
		t.Fatalf("readSourceBytes: %v", err)
	}
	if string(raw) != "Name,Code\nA,1\n" {
		t.Fatalf("raw = %q", string(raw))
	}
}

func TestReadSourceBytes_DocumentRef(t *testing.T) {
	ctx := ContextWithSourceBytes(context.Background(), func(_ context.Context, documentRef string) ([]byte, error) {
		if documentRef != "doc-1" {
			t.Fatalf("documentRef = %q", documentRef)
		}
		return []byte("Name,Code\nB,2\n"), nil
	})
	raw, err := readSourceBytes(ctx, importpkg.Spec{
		Source: importpkg.Source{DocumentRef: "doc-1"},
	})
	if err != nil {
		t.Fatalf("readSourceBytes: %v", err)
	}
	if string(raw) != "Name,Code\nB,2\n" {
		t.Fatalf("raw = %q", string(raw))
	}
}

func TestReadSourceBytes_Errors(t *testing.T) {
	if _, err := readSourceBytes(context.Background(), importpkg.Spec{}); err == nil {
		t.Fatal("expected missing source error")
	}
	if _, err := readSourceBytes(context.Background(), importpkg.Spec{
		Source: importpkg.Source{DocumentRef: "doc-1"},
	}); err == nil {
		t.Fatal("expected missing loader error")
	}
}

func TestInjectCompanyID(t *testing.T) {
	p := plan.Plan{
		Units: []plan.Unit{
			recordplan.Unit{Model: "partner.Partner", Values: map[string]string{"Name": "A"}},
			recordplan.Unit{Model: "partner.Partner", Values: nil},
		},
	}
	got := injectCompanyID(p, "cmp-1")
	u0 := got.Units[0].(recordplan.Unit)
	if u0.Values["CompanyId"] != "cmp-1" {
		t.Fatalf("company id = %q", u0.Values["CompanyId"])
	}
	u1 := got.Units[1].(recordplan.Unit)
	if u1.Values["CompanyId"] != "cmp-1" {
		t.Fatalf("company id on nil map = %q", u1.Values["CompanyId"])
	}
	if got := injectCompanyID(plan.Plan{}, "cmp-1"); len(got.Units) != 0 {
		t.Fatal("empty plan unchanged")
	}
	unchanged := injectCompanyID(p, "")
	if len(unchanged.Units) != len(p.Units) {
		t.Fatal("empty company unchanged")
	}
}

func TestReadSourceBytes_LoaderError(t *testing.T) {
	ctx := ContextWithSourceBytes(context.Background(), func(_ context.Context, _ string) ([]byte, error) {
		return nil, errors.New("loader failed")
	})
	if _, err := readSourceBytes(ctx, importpkg.Spec{Source: importpkg.Source{DocumentRef: "doc-1"}}); err == nil {
		t.Fatal("expected loader error")
	}
}

func TestSourceBytesFromContextNilCtx(t *testing.T) {
	if _, ok := sourceBytesFromContext(nil); ok {
		t.Fatal("nil ctx should not have loader")
	}
}

func TestInjectCompanyIDSkipsNonRecordUnits(t *testing.T) {
	p := plan.Plan{
		Units: []plan.Unit{
			recordplan.Unit{Model: "partner.Partner", Values: map[string]string{"Name": "A"}},
			stubplan.Unit{Index: 2},
		},
	}
	got := injectCompanyID(p, "cmp-1")
	u0 := got.Units[0].(recordplan.Unit)
	if u0.Values["CompanyId"] != "cmp-1" {
		t.Fatalf("company id = %q", u0.Values["CompanyId"])
	}
}

func TestContextWithSourceBytes_NilLoader(t *testing.T) {
	if got := ContextWithSourceBytes(nil, nil); got == nil {
		t.Fatal("expected background context")
	}
	if got := ContextWithSourceBytes(context.Background(), nil); got != context.Background() {
		t.Fatal("nil loader returns original ctx")
	}
}

func TestBuilder_BuildFromDocumentRef(t *testing.T) {
	ctx := ContextWithSourceBytes(context.Background(), func(_ context.Context, documentRef string) ([]byte, error) {
		return []byte("Name,Code\nA,C1\n"), nil
	})
	builder := Builder{}
	plan, err := builder.Build(ctx, importpkg.Spec{
		Profile: importpkg.ProfileRecord,
		Model:   "base.Country",
		Source:  importpkg.Source{Format: "csv", DocumentRef: "doc-1"},
		Options: importpkg.Options{CompanyID: "cmp-1"},
	})
	if err != nil || len(plan.Units) != 1 {
		t.Fatalf("Build: %#v %v", plan, err)
	}
	unit := plan.Units[0].(recordplan.Unit)
	if unit.Values["CompanyId"] != "cmp-1" {
		t.Fatalf("company id = %q", unit.Values["CompanyId"])
	}
}

func TestBuildRecordPlan_DefaultRowNumber(t *testing.T) {
	plan, err := BuildRecordPlan("base.Country", []byte("Name\nA\n"), nil)
	if err != nil || len(plan.Units) != 1 {
		t.Fatalf("BuildRecordPlan: %v %#v", err, plan)
	}
	unit := plan.Units[0].(recordplan.Unit)
	if unit.RowNumber != 2 {
		t.Fatalf("row number = %d", unit.RowNumber)
	}
}
