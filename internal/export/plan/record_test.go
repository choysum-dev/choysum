// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan_test

import (
	"testing"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
)

func TestSearchCondition_IdsPrecedence(t *testing.T) {
	cond, err := exportplan.SearchCondition(exportplan.Plan{
		Ids:    []string{"a", "b"},
		Domain: `[{"And":[["Code","=","X"]]}]`,
	})
	if err != nil {
		t.Fatalf("SearchCondition: %v", err)
	}
	and, _ := cond["And"].([]any)
	tuple, _ := and[0].([]any)
	if tuple[0] != "Id" || tuple[1] != "in" {
		t.Fatalf("cond = %#v", cond)
	}
}

func TestResolveExportFields_Defaults(t *testing.T) {
	fields, err := exportplan.ResolveExportFields(exportplan.Plan{Model: "base.Country"}, func(model string) ([]string, error) {
		if model != "base.Country" {
			t.Fatalf("model = %q", model)
		}
		return []string{"Name", "Code"}, nil
	})
	if err != nil || len(fields) != 2 {
		t.Fatalf("fields = %#v, err = %v", fields, err)
	}
}

func TestSearchCondition_InvalidDomain(t *testing.T) {
	_, err := exportplan.SearchCondition(exportplan.Plan{Domain: "not-json"})
	if err == nil {
		t.Fatal("expected domain error")
	}
}

func TestSearchCondition_EmptyDomain(t *testing.T) {
	cond, err := exportplan.SearchCondition(exportplan.Plan{})
	if err != nil {
		t.Fatalf("SearchCondition: %v", err)
	}
	and, _ := cond["And"].([]any)
	if len(and) != 0 {
		t.Fatalf("cond = %#v", cond)
	}
}

func TestSearchCondition_TupleDomain(t *testing.T) {
	cond, err := exportplan.SearchCondition(exportplan.Plan{Domain: `["Code","=","X"]`})
	if err != nil {
		t.Fatalf("SearchCondition: %v", err)
	}
	and, _ := cond["And"].([]any)
	tuple, _ := and[0].([]any)
	if tuple[0] != "Code" {
		t.Fatalf("cond = %#v", cond)
	}
}

func TestSearchCondition_InvalidDomainShape(t *testing.T) {
	_, err := exportplan.SearchCondition(exportplan.Plan{Domain: `"string"`})
	if err == nil {
		t.Fatal("expected invalid domain shape error")
	}
}

func TestResolveExportFields_Explicit(t *testing.T) {
	fields, err := exportplan.ResolveExportFields(exportplan.Plan{Fields: []string{"Name"}}, nil)
	if err != nil || len(fields) != 1 || fields[0] != "Name" {
		t.Fatalf("fields = %#v, err = %v", fields, err)
	}
}

func TestSearchCondition_ObjectDomain(t *testing.T) {
	cond, err := exportplan.SearchCondition(exportplan.Plan{Domain: `{"Or":[["Code","=","X"]]}`})
	if err != nil {
		t.Fatalf("SearchCondition: %v", err)
	}
	or, _ := cond["Or"].([]any)
	if len(or) != 1 {
		t.Fatalf("cond = %#v", cond)
	}
}

func TestSearchCondition_NullDomain(t *testing.T) {
	cond, err := exportplan.SearchCondition(exportplan.Plan{Domain: `null`})
	if err != nil {
		t.Fatalf("SearchCondition: %v", err)
	}
	and, _ := cond["And"].([]any)
	if len(and) != 0 {
		t.Fatalf("cond = %#v", cond)
	}
}

func TestResolveExportFields_NilDefault(t *testing.T) {
	_, err := exportplan.ResolveExportFields(exportplan.Plan{Model: "base.Country"}, nil)
	if err == nil {
		t.Fatal("expected nil defaultFields error")
	}
}
