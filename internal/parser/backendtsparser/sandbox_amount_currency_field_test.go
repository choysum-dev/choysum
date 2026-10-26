// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestBuildFieldResolvedSpec_ScaleAndCurrencyField(t *testing.T) {
	field := &meta.Field{
		Name: "Amount",
		Decorators: []*meta.Decorator{{
			Name: "Field",
			Arguments: []*meta.Argument{{
				Type:  "ObjectLiteral",
				Value: `{"type":"Decimal","scaleField":" Scale ","currencyField":" CurrencyId "}`,
			}},
		}},
	}
	if _, err := buildFieldResolvedSpec(field, nil, nil, "demo", "demo.scope", nil); err != nil {
		t.Fatalf("buildFieldResolvedSpec: %v", err)
	}
	if field.ScaleField != "Scale" {
		t.Fatalf("ScaleField=%q want Scale", field.ScaleField)
	}
	if field.CurrencyField != "CurrencyId" {
		t.Fatalf("CurrencyField=%q want CurrencyId", field.CurrencyField)
	}
	// Whitespace-only companions must not overwrite.
	blank := &meta.Field{
		Name: "Amount",
		Decorators: []*meta.Decorator{{
			Name: "Field",
			Arguments: []*meta.Argument{{
				Type:  "ObjectLiteral",
				Value: `{"type":"Decimal","scaleField":"  ","currencyField":"  "}`,
			}},
		}},
	}
	if _, err := buildFieldResolvedSpec(blank, nil, nil, "demo", "demo.scope", nil); err != nil {
		t.Fatalf("blank companions: %v", err)
	}
	if blank.ScaleField != "" || blank.CurrencyField != "" {
		t.Fatalf("blank companions must stay empty, got scale=%q currency=%q", blank.ScaleField, blank.CurrencyField)
	}
}

func TestParseSandboxAmountCurrencyField(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../../.."))
	path := filepath.Join(root, "modules/sandbox/service/models/sandbox_record.ts")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	runtimeScope := newBackendParserTestScope()
	module := &meta.Module{Path: filepath.Join(root, "modules/sandbox"), ApplicationStr: "sandbox"}
	p := NewTsParser(runtimeScope, module)
	res, err := p.Parse(map[string]string{}, path, string(content))
	if err != nil {
		t.Fatal(err)
	}
	if res.Model == nil {
		t.Fatal("expected model")
	}
	for _, f := range res.Model.Fields {
		if f.Name != "Amount" {
			continue
		}
		for _, d := range f.Decorators {
			if d.Name != "Field" {
				continue
			}
			for _, a := range d.Arguments {
				var opts map[string]any
				if err := json.Unmarshal([]byte(a.Value), &opts); err != nil {
					t.Fatalf("unmarshal: %v value=%s", err, a.Value)
				}
				t.Logf("Amount options currencyField=%v keys=%v", opts["currencyField"], opts)
				if opts["currencyField"] != "CurrencyId" {
					t.Fatalf("expected currencyField CurrencyId in decorator JSON, got %#v full=%s", opts["currencyField"], a.Value)
				}
			}
		}
		if f.CurrencyField != "CurrencyId" {
			t.Fatalf("Amount.CurrencyField=%q want CurrencyId", f.CurrencyField)
		}
		return
	}
	t.Fatal("Amount field not found")
}


func TestParseSandboxCurrencyIdRelationModel(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../../.."))
	path := filepath.Join(root, "modules/sandbox/service/models/sandbox_record.ts")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	runtimeScope := newBackendParserTestScope()
	module := &meta.Module{Path: filepath.Join(root, "modules/sandbox"), ApplicationStr: "sandbox"}
	p := NewTsParser(runtimeScope, module)
	res, err := p.Parse(map[string]string{}, path, string(content))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Model.Fields {
		if f.Name != "CurrencyId" {
			continue
		}
		t.Logf("CurrencyId FieldType=%q RelationModel=%q", f.FieldType, f.RelationModel)
		if f.FieldType != "ManyToOneRef" {
			t.Fatalf("FieldType=%q", f.FieldType)
		}
		if f.RelationModel != "base.Currency" {
			t.Fatalf("RelationModel=%q want base.Currency", f.RelationModel)
		}
		return
	}
	t.Fatal("CurrencyId not found")
}
