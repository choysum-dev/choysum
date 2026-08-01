// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package manifest

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/distmanifest"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type distManifestTestScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *distManifestTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *distManifestTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *distManifestTestScope) Session() *scope.Session { return s.session }
func (s *distManifestTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &distManifestTestScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *distManifestTestScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *distManifestTestScope) Logger() *slog.Logger { return s.logger }

func TestWriteDistManifestBuildsInstalledModuleManifest(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dist-manifest.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("auto migrate modules: %v", err)
	}
	for _, row := range []meta.Module{
		{Name: "auth", ApplicationStr: "auth", Status: meta.Installed, DependsStr: []byte(`["core"]`)},
		{Name: "crm", ApplicationStr: "crm", Status: meta.Installed, DependsStr: []byte(`["auth"]`)},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed module %q: %v", row.Name, err)
		}
	}

	rs := &distManifestTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	outPath := filepath.Join(t.TempDir(), "dist", distmanifest.DistManifestFileName)
	if err := WriteDistManifest(nil, rs, "bundle", outPath); err != nil {
		t.Fatalf("WriteDistManifest() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var got distmanifest.DistManifestV2
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if len(got.BackendTopoOrder) != 2 || got.BackendTopoOrder[0] != "auth" || got.BackendTopoOrder[1] != "crm" {
		t.Fatalf("unexpected backend topo order: %#v", got.BackendTopoOrder)
	}
	authApp, ok := got.Apps["auth"]
	if !ok || len(authApp.Dev.Modules) != 1 || authApp.Dev.Modules[0] != "auth" {
		t.Fatalf("unexpected auth app manifest: %#v", authApp)
	}
}

func TestWriteDistManifestListInstalledModulesError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dist-manifest-error.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&meta.Module{}); err != nil {
		t.Fatalf("auto migrate modules: %v", err)
	}
	if err := db.Migrator().DropTable(&meta.Module{}); err != nil {
		t.Fatalf("drop meta_module: %v", err)
	}

	rs := &distManifestTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	err = WriteDistManifest(context.Background(), rs, "bundle", filepath.Join(t.TempDir(), "manifest.json"))
	if err == nil || !strings.Contains(err.Error(), "list installed modules") {
		t.Fatalf("WriteDistManifest() error = %v, want list installed modules failure", err)
	}
}
