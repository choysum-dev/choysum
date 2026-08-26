// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"context"
	"errors"
	"testing"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

type stubCaller struct {
	result any
	err    error
}

func (c *stubCaller) Call(context.Context, importcaller.CallRequest) (any, error) {
	return c.result, c.err
}

func TestReader_TemplateMode(t *testing.T) {
	result, err := Reader{}.Read(context.Background(), nil, exportplan.Plan{
		Profile: exportpkg.ProfileRecord,
		Model:   "base.Country",
		Mode:    exportpkg.ModeTemplate,
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(result.Headers) == 0 {
		t.Fatal("expected template headers")
	}
}

func TestReader_RequiresCaller(t *testing.T) {
	_, err := Reader{}.Read(context.Background(), nil, exportplan.Plan{
		Profile: exportpkg.ProfileRecord,
		Model:   "base.Country",
		Mode:    exportpkg.ModeData,
	})
	if err == nil {
		t.Fatal("expected caller error")
	}
}

func TestReader_SearchError(t *testing.T) {
	ctx := importcaller.ContextWithCaller(context.Background(), &stubCaller{err: errors.New("boom")})
	_, err := Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile: exportpkg.ProfileRecord,
		Model:   "base.Country",
		Mode:    exportpkg.ModeData,
	})
	if err == nil {
		t.Fatal("expected search error")
	}
}

func TestReader_WithLimitOffset(t *testing.T) {
	ctx := importcaller.ContextWithCaller(context.Background(), &stubCaller{
		result: []any{map[string]any{"Name": "A", "Code": "C1"}},
	})
	result, err := Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile: exportpkg.ProfileRecord,
		Model:   "base.Country",
		Mode:    exportpkg.ModeData,
		Limit:   1,
		Offset:  2,
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if result.Outcomes.Total != 1 {
		t.Fatalf("total = %d", result.Outcomes.Total)
	}
}

func TestReader_InvalidModelFields(t *testing.T) {
	ctx := importcaller.ContextWithCaller(context.Background(), &stubCaller{})
	_, err := Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile: exportpkg.ProfileRecord,
		Model:   "base.Partner",
		Mode:    exportpkg.ModeData,
	})
	if err == nil {
		t.Fatal("expected unsupported model error")
	}
}

func TestReader_InvalidSearchResult(t *testing.T) {
	ctx := importcaller.ContextWithCaller(context.Background(), &stubCaller{result: "bad"})
	_, err := Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile: exportpkg.ProfileRecord,
		Model:   "base.Country",
		Mode:    exportpkg.ModeData,
	})
	if err == nil {
		t.Fatal("expected search result type error")
	}
}

func TestReader_InvalidDomain(t *testing.T) {
	ctx := importcaller.ContextWithCaller(context.Background(), &stubCaller{})
	_, err := Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile: exportpkg.ProfileRecord,
		Model:   "base.Country",
		Mode:    exportpkg.ModeData,
		Domain:  "not-json",
	})
	if err == nil {
		t.Fatal("expected domain error")
	}
}
