// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type raceTestScope struct {
	ctx     context.Context
	logger  *slog.Logger
	session *scope.Session
}

func (s *raceTestScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *raceTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *raceTestScope) Session() *scope.Session { return s.session }
func (s *raceTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &raceTestScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *raceTestScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *raceTestScope) Logger() *slog.Logger { return s.logger }

func TestWarmLanguageSkipsStaleSnapshotAfterOverride(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "race.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rs := &raceTestScope{
		ctx:     context.Background(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
	if err := i18nmodels.EnsureTranslationTermTable(rs, "auth"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	table := rs.Session().Table("auth_translation_term")
	if err := table.Create(&i18nmodels.TranslationTerm{
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
		Scope:       "web/a@t",
		Src:         "Hello",
		Value:       "你好",
		Kind:        i18nmodels.KindLiteral,
		Source:      i18nmodels.SourcePackaged,
	}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ts := NewTermStore(rs, "auth")
	if err := ts.WarmLanguage("zh_CN"); err != nil {
		t.Fatalf("warm: %v", err)
	}

	// Inject a concurrent override after WarmLanguage has read the DB snapshot
	// but before it installs the cache.
	prev := warmAfterLoadHook
	t.Cleanup(func() { warmAfterLoadHook = prev })
	warmAfterLoadHook = func(lang string) {
		if lang != "zh_CN" {
			return
		}
		if _, err := ts.UpsertOverride("auth", "zh_CN", "web/a@t", "Hello", "", "您好"); err != nil {
			t.Errorf("upsert during warm: %v", err)
		}
	}

	if err := ts.WarmLanguage("zh_CN"); err != nil {
		t.Fatalf("warm with barrier: %v", err)
	}
	got, ok := ts.Lookup("auth", "zh_CN", "web/a@t", "Hello", "")
	if !ok || got != "您好" {
		t.Fatalf("Lookup = %q ok=%v, want override kept over stale warm", got, ok)
	}
}
