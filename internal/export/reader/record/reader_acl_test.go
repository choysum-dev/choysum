// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record_test

import (
	"context"
	"testing"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/reader/record"
	importcaller "github.com/choysum-dev/choysum/internal/import/caller"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

type aclCaller struct{}

func (c *aclCaller) Call(ctx context.Context, req importcaller.CallRequest) (any, error) {
	if req.Model+"."+req.Method != "base.Country.Search" {
		return nil, nil
	}
	return []any{}, nil
}

func TestRecordReader_RespectsRecordRule(t *testing.T) {
	ctx := importcaller.ContextWithCaller(context.Background(), &aclCaller{})
	result, err := record.Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile: exportpkg.ProfileRecord,
		Model:   "base.Country",
		Mode:    exportpkg.ModeData,
		Format:  "csv",
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if result.Outcomes.Total != 0 || len(result.Rows) != 0 {
		t.Fatalf("expected empty export under record rule, got %+v", result)
	}
}
