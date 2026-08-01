// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type catalogTestScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *catalogTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *catalogTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *catalogTestScope) Session() *scope.Session { return s.session }
func (s *catalogTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &catalogTestScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *catalogTestScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *catalogTestScope) Logger() *slog.Logger { return s.logger }

func TestInstalledModulesByAppIncludesFrameworkModule(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "catalog.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatal(err)
	}
	rs := &catalogTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	rows := []meta.Module{
		{Name: "core", ApplicationStr: "core", Status: meta.Installed},
		{Name: "auth", ApplicationStr: "auth", Status: meta.Installed},
		{Name: "web", ApplicationStr: "web", Status: meta.Installed},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	byApp, err := installedModulesByApp(rs)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := byApp["core"]; ok {
		t.Fatal("must not expose application core")
	}
	for _, app := range []string{"auth", "web"} {
		mods := byApp[app]
		if !slices.Contains(mods, "core") {
			t.Fatalf("%s modules = %v, want core included", app, mods)
		}
		if !slices.Contains(mods, app) {
			t.Fatalf("%s modules = %v, want own module", app, mods)
		}
	}
}
