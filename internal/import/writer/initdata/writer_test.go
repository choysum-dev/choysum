// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initdata

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/plan"
	initdataplan "github.com/choysum-dev/choysum/internal/import/plan/initdata"
	planstub "github.com/choysum-dev/choysum/internal/import/plan/stub"
	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestWriter_Write(t *testing.T) {
	runtimeScope := dataloader.BootstrapTestScope(t)
	dir := t.TempDir()
	dataloader.WriteDataFileForTest(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "group_writer", "application": "auth", "model": "group", "values": map[string]any{}},
		},
	})

	unit := initdataplan.Unit{
		Index:       1,
		ModuleName:  "auth",
		ModulePath:  dir,
		Application: "auth",
		Files:       []string{"data.json"},
	}
	writer := Writer{}
	if err := writer.Write(context.Background(), runtimeScope, []plan.Unit{unit}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	var count int64
	if err := runtimeScope.Session().Table("auth_group").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("row count = %d, want 1", count)
	}
}

func TestWriter_WriteUnexpectedUnitType(t *testing.T) {
	writer := Writer{}
	err := writer.Write(context.Background(), nil, []plan.Unit{planstub.Unit{Index: 1}})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("Write() error = %v, want CodeInvalidFormat", err)
	}
}

func TestWriter_WriteLoadErrorMapped(t *testing.T) {
	runtimeScope := dataloader.BootstrapTestScope(t)
	dir := t.TempDir()
	dataloader.WriteDataFileForTest(t, dir, map[string]any{
		"records": []any{
			map[string]any{"module": "auth", "name": "bad", "application": "auth", "model": "MissingModel", "values": map[string]any{}},
		},
	})

	unit := initdataplan.Unit{
		ModuleName:  "auth",
		ModulePath:  dir,
		Application: "auth",
		Files:       []string{"data.json"},
	}
	writer := Writer{}
	err := writer.Write(context.Background(), runtimeScope, []plan.Unit{unit})
	if err == nil {
		t.Fatal("expected load error")
	}
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) {
		t.Fatalf("error = %T, want *importpkg.Error", err)
	}
	if impErr.Code == "" || impErr.Text == "" {
		t.Fatalf("mapped error = %+v", impErr)
	}
}

func TestWriter_WritePlainError(t *testing.T) {
	runtimeScope := dataloader.BootstrapTestScope(t)
	unit := initdataplan.Unit{
		ModuleName: "auth",
		ModulePath: "/missing/path",
		Files:      []string{"data.json"},
	}
	writer := Writer{}
	err := writer.Write(context.Background(), runtimeScope, []plan.Unit{unit})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	var impErr *importpkg.Error
	if errors.As(err, &impErr) {
		t.Fatalf("plain filesystem error should not be wrapped as import error: %v", err)
	}
}

func TestWriter_WriteNilUnits(t *testing.T) {
	writer := Writer{}
	if err := writer.Write(context.Background(), nil, nil); err != nil {
		t.Fatalf("Write(nil units): %v", err)
	}
}

func TestMapLoaderError(t *testing.T) {
	loadErr := &dataloader.LoadError{Code: dataloader.LoadErrorCodeMissingName, Message: "missing name"}
	err := mapLoaderError(loadErr)
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != dataloader.LoadErrorCodeMissingName {
		t.Fatalf("mapLoaderError(loadErr) = %v", err)
	}
	plain := errors.New("plain")
	if mapLoaderError(plain) != plain {
		t.Fatal("non-LoadError should pass through unchanged")
	}
}
