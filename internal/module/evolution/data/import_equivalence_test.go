// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader_test

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/runner"
	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestImportRun_MatchesApplyFiles(t *testing.T) {
	t.Cleanup(func() { importpkg.SetRun(nil) })
	importpkg.SetRun(runner.Run)

	loaderScope := dataloader.NewDefaultLoaderScopeForTest(t)
	importScope := dataloader.NewDefaultLoaderScopeForTest(t)

	dir := t.TempDir()
	dataloader.WriteDataFileForTest(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "group_import", "application": "auth", "model": "group", "values": map[string]any{}},
		},
	})
	mod := &meta.Module{Name: "auth", Path: dir, ApplicationStr: "auth"}

	ctx := context.Background()
	if err := dataloader.New(loaderScope).ApplyFiles(ctx, mod, []string{"data.json"}); err != nil {
		t.Fatalf("ApplyFiles: %v", err)
	}
	var loaderCount int64
	if err := loaderScope.Session().Table("auth_group").Count(&loaderCount).Error; err != nil {
		t.Fatalf("count loader rows: %v", err)
	}

	spec := importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Caller:  importpkg.CallerE2E,
		Policy:  importpkg.PolicyAtomic,
		Module:  mod.Name,
		Source: importpkg.Source{
			Format: "json",
			Path:   mod.Path,
		},
		Options: importpkg.Options{InitdataFiles: []string{"data.json"}},
	}
	if _, err := importpkg.Run(ctx, importScope, spec); err != nil {
		t.Fatalf("import.Run: %v", err)
	}
	var importCount int64
	if err := importScope.Session().Table("auth_group").Count(&importCount).Error; err != nil {
		t.Fatalf("count import rows: %v", err)
	}
	if loaderCount != importCount || loaderCount != 1 {
		t.Fatalf("loader count = %d, import count = %d, want 1", loaderCount, importCount)
	}
}
