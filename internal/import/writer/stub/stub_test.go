// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package stub_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/import/plan"
	planstub "github.com/choysum-dev/choysum/internal/import/plan/stub"
	"github.com/choysum-dev/choysum/internal/import/writer/stub"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func testScope(t *testing.T) scope.Scope {
	t.Helper()
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "stub-writer.db"),
		},
	}
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err := runtimeScope.Session().AutoMigrate(&stub.Row{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return runtimeScope
}

func TestWriter_Write(t *testing.T) {
	writer := stub.Writer{}

	if err := writer.Write(context.Background(), nil, []plan.Unit{planstub.Unit{Index: 1}}); err != nil {
		t.Fatalf("Write(nil scope): %v", err)
	}

	var impErr *importpkg.Error
	if err := writer.Write(context.Background(), nil, []plan.Unit{planstub.Unit{Index: 2, Fail: true}}); !errors.As(err, &impErr) {
		t.Fatalf("Write(fail unit) error = %v, want *importpkg.Error", err)
	}

	if err := writer.Write(context.Background(), nil, []plan.Unit{fakeUnit{}}); err != nil {
		t.Fatalf("Write(non-stub unit): %v", err)
	}

	runtimeScope := testScope(t)
	if err := writer.Write(context.Background(), runtimeScope, []plan.Unit{planstub.Unit{Index: 3}}); err != nil {
		t.Fatalf("Write(persist): %v", err)
	}
	var count int64
	if err := runtimeScope.Session().Model(&stub.Row{}).Count(&count).Error; err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("row count = %d, want 1", count)
	}
}

func TestWriter_WriteDBError(t *testing.T) {
	runtimeScope := testScope(t)
	if err := runtimeScope.Session().Migrator().DropTable(&stub.Row{}); err != nil {
		t.Fatalf("DropTable: %v", err)
	}

	err := stub.Writer{}.Write(context.Background(), runtimeScope, []plan.Unit{planstub.Unit{Index: 1}})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("Write(missing table) error = %v, want CodeInvalidFormat", err)
	}
}

type fakeUnit struct{}

func (fakeUnit) UnitIndex() int { return 0 }
