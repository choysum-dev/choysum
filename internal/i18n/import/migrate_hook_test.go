// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18nimport

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateTranslationTermTableIfMissingPropagatesMigrateError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "hook.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	rs := &hookScope{
		ctx:     context.Background(),
		session: &scope.Session{DB: db},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	old := migrateTranslationTermTable
	migrateTranslationTermTable = func(scope.Scope, string) error {
		return errors.New("migrate auth_translation_term: forced")
	}
	t.Cleanup(func() { migrateTranslationTermTable = old })

	_, err = UpsertPackagedTerms(rs, nil, "auth", "auth", "zh_CN", []byte(`
msgctxt "web/a@new"
msgid "Hello"
msgstr "你好"
`))
	if err == nil || !strings.Contains(err.Error(), "migrate auth_translation_term") {
		t.Fatalf("expected migrate error, got %v", err)
	}
}

type hookScope struct {
	ctx     context.Context
	session *scope.Session
	logger  *slog.Logger
}

func (s *hookScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *hookScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(s)
}
func (s *hookScope) Session() *scope.Session { return s.session }
func (s *hookScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = s.ctx
	}
	return &hookScope{ctx: ctx, logger: s.logger, session: s.session}
}
func (s *hookScope) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
func (s *hookScope) Logger() *slog.Logger { return s.logger }
