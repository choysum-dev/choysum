// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/orm"
	"github.com/choysum-dev/choysum/internal/import/plan"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	recordwriter "github.com/choysum-dev/choysum/internal/import/writer/record"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestResolveM2O_Validation(t *testing.T) {
	caller := &scriptedCaller{responses: map[string]any{
		"base.Currency.Search": []any{map[string]any{"Id": "cur-1"}},
	}}
	unit := recordplan.Unit{Index: 1, Model: "base.Country"}

	if _, err := recordwriter.ResolveM2O(context.Background(), caller, unit, "DefaultCurrencyId", "CNY"); err == nil {
		t.Fatal("expected invalid path")
	}
	if _, err := recordwriter.ResolveM2O(context.Background(), caller, unit, "Name/Code", "CNY"); err == nil {
		t.Fatal("expected unsupported M2O field")
	}
	if _, err := recordwriter.ResolveM2O(context.Background(), caller, unit, "DefaultCurrencyId/Name", "CNY"); err == nil {
		t.Fatal("expected unsupported lookup field")
	}
	id, err := recordwriter.ResolveM2O(context.Background(), caller, unit, "DefaultCurrencyId/id", "x")
	if err != nil || id != "cur-1" {
		t.Fatalf("id lookup: %q %v", id, err)
	}
	caller.responses["base.Currency.Search"] = []any{}
	if _, err := recordwriter.ResolveM2O(context.Background(), caller, unit, "DefaultCurrencyId/Code", "NOPE"); err == nil {
		t.Fatal("expected not found")
	}
	caller.fail = errors.New("unique constraint")
	_, err = recordwriter.ResolveM2O(context.Background(), caller, unit, "DefaultCurrencyId/Code", "CNY")
	imp, ok := importpkg.AsError(err)
	if !ok || imp.Code != importpkg.CodeDuplicateKey {
		t.Fatalf("error = %v", err)
	}
}

func TestUpsertCountry_ErrorPaths(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	unit := recordplan.Unit{
		Index:  1,
		Model:  "base.Country",
		Values: map[string]string{"Name": "X", "Code": "EP1", "IsActive": "true", "ZipRequired": "true", "StateRequired": "false"},
	}
	if err := recordwriter.UpsertCountry(context.Background(), runtimeScope, unit); err == nil {
		t.Fatal("expected missing orm caller")
	}

	ctx := orm.ContextWithCaller(context.Background(), &scriptedCaller{})
	if err := recordwriter.UpsertCountry(ctx, nil, unit); err == nil {
		t.Fatal("expected missing scope")
	}

	unit.Values = map[string]string{}
	if err := recordwriter.UpsertCountry(ctx, runtimeScope, unit); err == nil {
		t.Fatal("expected empty values")
	}

	unit.Values = map[string]string{"Name": "X", "Code": "EP1", "IsActive": "maybe"}
	if err := recordwriter.UpsertCountry(ctx, runtimeScope, unit); err == nil {
		t.Fatal("expected invalid bool")
	}

	unit.ExternalID = "bad."
	unit.Values = map[string]string{"Name": "X", "Code": "EP1", "IsActive": "true", "ZipRequired": "1", "StateRequired": "0"}
	if err := recordwriter.UpsertCountry(ctx, runtimeScope, unit); err == nil {
		t.Fatal("expected invalid external id")
	}

	unit.ExternalID = ""
	unit.Values = map[string]string{"Name": "X", "Partner.Name": "Y", "Code": "EP1", "IsActive": "true", "ZipRequired": "true", "StateRequired": "false"}
	if err := recordwriter.UpsertCountry(ctx, runtimeScope, unit); err == nil {
		t.Fatal("expected O2M unsupported")
	}

	unit.Values = map[string]string{"Name": "X", "Code": "EP1", "IsActive": "true", "ZipRequired": "true", "StateRequired": "false", "": "x"}
	caller := &scriptedCaller{responses: map[string]any{
		"base.Country.Search": []any{},
		"base.Country.Create": map[string]any{"Id": "n1"},
	}}
	ctx = orm.ContextWithCaller(context.Background(), caller)
	if err := recordwriter.UpsertCountry(ctx, runtimeScope, unit); err != nil {
		t.Fatalf("empty field path skip: %v", err)
	}
}

func TestWriter_UnsupportedModelUnit(t *testing.T) {
	runtimeScope := newCountryImportScope(t)
	ctx := orm.ContextWithCaller(context.Background(), &scriptedCaller{})
	err := recordwriter.Writer{}.Write(ctx, runtimeScope, []plan.Unit{recordplan.Unit{
		Index:  1,
		Model:  "base.Partner",
		Values: map[string]string{"Name": "X"},
	}})
	if err == nil {
		t.Fatal("expected unsupported model")
	}
}

func TestWithImportFileContext_AllBranches(t *testing.T) {
	if recordwriter.WithImportFileContext(nil) != nil {
		t.Fatal("nil scope")
	}
	runtimeScope := newExternalIDTestScope(t)
	marked := recordwriter.WithImportFileContext(runtimeScope)
	if !recordwriter.ImportFileFromContext(marked.Context()) {
		t.Fatal("expected import file marker")
	}
	again := recordwriter.WithImportFileContext(marked)
	if again != marked {
		t.Fatal("expected idempotent WithImportFileContext")
	}
}

type scriptedCaller struct {
	responses map[string]any
	fail      error
	calls     []string
}

func (c *scriptedCaller) Call(ctx context.Context, req orm.CallRequest) (any, error) {
	key := req.Model + "." + req.Method
	c.calls = append(c.calls, key)
	if c.fail != nil {
		return nil, c.fail
	}
	if c.responses == nil {
		return nil, fmt.Errorf("no response for %s", key)
	}
	v, ok := c.responses[key]
	if !ok {
		return nil, fmt.Errorf("unsupported %s", key)
	}
	return v, nil
}
