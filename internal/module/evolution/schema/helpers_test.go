// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	dynamicstruct "github.com/Chise1/dynamic-struct"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type schemaTestScope struct {
	ctx     context.Context
	cfg     *config.Config
	logger  *slog.Logger
	session *scope.Session
}

func (e *schemaTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }

func (e *schemaTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *schemaTestScope) Session() *scope.Session { return e.session }

func (e *schemaTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = e.ctx
	}
	return &schemaTestScope{ctx: ctx, cfg: e.cfg, logger: e.logger, session: e.session}
}

func (e *schemaTestScope) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}

func (e *schemaTestScope) Logger() *slog.Logger { return e.logger }

func (e *schemaTestScope) Config() *config.Config { return e.cfg }

func (e *schemaTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newSchemaTestScope(t *testing.T) *schemaTestScope {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "schema.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &schemaTestScope{
		ctx:     context.Background(),
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "sqlite"}, Server: config.NewDefaultServerConfig(), Log: config.NewDefaultLogConfig()},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
}

func migrateSchemaMetaTables(t *testing.T, session *scope.Session) {
	t.Helper()
	if err := session.AutoMigrate(&meta.IrModel{}, &meta.IrField{}, &meta.IrDecorator{}, &meta.IrArgument{}, &meta.IrModule{}); err != nil {
		t.Fatalf("migrate schema meta tables: %v", err)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func newFieldWithOptions(t *testing.T, name string, options string) *meta.IrField {
	t.Helper()
	return &meta.IrField{
		Name: name,
		Decorators: []*meta.IrDecorator{{
			Name:      "Field",
			Arguments: []*meta.IrArgument{{Type: "ObjectLiteral", Value: options}},
		}},
	}
}

func newRelationField(name string, moduleSpecPath string, options string) *meta.IrField {
	field := &meta.IrField{
		Name:           name,
		ModuleSpecPath: moduleSpecPath,
		Decorators: []*meta.IrDecorator{{
			Name:      "Field",
			Arguments: []*meta.IrArgument{{Type: "ObjectLiteral", Value: options}},
		}},
	}
	field.ModuleSpecPath = moduleSpecPath
	return field
}

func dynamicStructBuilder() dynamicstruct.Builder {
	return dynamicstruct.NewStruct()
}
