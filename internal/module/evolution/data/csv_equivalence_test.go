// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/runner"
	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
)

// JSON fixture ↔ CSV fixture → import.Run DB equivalence (PR-import-2csv / §9.12.4).
func TestImportRun_JSONAndCSV_Equivalent(t *testing.T) {
	importpkg.SetRun(runner.Run)

	jsonScope := dataloader.NewDefaultLoaderScopeForTest(t)
	csvScope := dataloader.NewDefaultLoaderScopeForTest(t)

	jsonDir := t.TempDir()
	dataloader.WriteDataFileForTest(t, jsonDir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "group_csv_eq", "application": "auth", "model": "group", "values": map[string]any{}},
		},
	})

	csvDir := t.TempDir()
	csvPath := filepath.Join(csvDir, "groups.csv")
	if err := os.WriteFile(csvPath, []byte("id,model\nauth.group_csv_eq,group\n"), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	ctx := context.Background()
	jsonMod := &meta.Module{Name: "auth", Path: jsonDir, ApplicationStr: "auth"}
	csvMod := &meta.Module{Name: "auth", Path: csvDir, ApplicationStr: "auth"}

	jsonSpec := importpkg.Spec{
		Profile: importpkg.ProfileInitdata,
		Caller:  importpkg.CallerE2E,
		Policy:  importpkg.PolicyAtomic,
		Module:  jsonMod.Name,
		Source: importpkg.Source{
			Format: "json",
			Path:   jsonMod.Path,
		},
		Options: importpkg.Options{InitdataFiles: []string{"data.json"}},
	}
	if _, err := importpkg.Run(ctx, jsonScope, jsonSpec); err != nil {
		t.Fatalf("import.Run json: %v", err)
	}

	csvSpec := importpkg.Spec{
		Profile:     importpkg.ProfileInitdata,
		Caller:      importpkg.CallerE2E,
		Policy:      importpkg.PolicyAtomic,
		Module:      csvMod.Name,
		Application: csvMod.ApplicationStr,
		Source: importpkg.Source{
			Format: "csv",
			Path:   csvMod.Path,
		},
		Options: importpkg.Options{InitdataFiles: []string{"groups.csv"}},
	}
	if _, err := importpkg.Run(ctx, csvScope, csvSpec); err != nil {
		t.Fatalf("import.Run csv: %v", err)
	}

	var jsonCount, csvCount int64
	if err := jsonScope.Session().Table("auth_group").Count(&jsonCount).Error; err != nil {
		t.Fatalf("count json: %v", err)
	}
	if err := csvScope.Session().Table("auth_group").Count(&csvCount).Error; err != nil {
		t.Fatalf("count csv: %v", err)
	}
	if jsonCount != csvCount || jsonCount != 1 {
		t.Fatalf("json count = %d, csv count = %d, want 1", jsonCount, csvCount)
	}
}
