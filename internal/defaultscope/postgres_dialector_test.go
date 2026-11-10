// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultscope

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type stubTypeAliasMigrator struct {
	gorm.Migrator
	aliases map[string][]string
}

func (m stubTypeAliasMigrator) GetTypeAliases(databaseTypeName string) []string {
	if m.aliases == nil {
		return nil
	}
	return m.aliases[databaseTypeName]
}

func TestAppendCharAliasesForBPChar(t *testing.T) {
	t.Parallel()

	got := appendCharAliasesForBPChar(nil)
	if !containsFold(got, "char") || !containsFold(got, "character") {
		t.Fatalf("nil aliases: got %v, want char+character", got)
	}

	got = appendCharAliasesForBPChar([]string{"char", "character"})
	if len(got) != 2 {
		t.Fatalf("already present: got %v, want unchanged length 2", got)
	}
}

type nonConcretePostgresDialector struct{}

func (nonConcretePostgresDialector) Name() string                                       { return "postgres" }
func (nonConcretePostgresDialector) Initialize(*gorm.DB) error                          { return nil }
func (nonConcretePostgresDialector) Migrator(*gorm.DB) gorm.Migrator                  { return nil }
func (nonConcretePostgresDialector) DataTypeOf(*schema.Field) string                    { return "" }
func (nonConcretePostgresDialector) DefaultValueOf(*schema.Field) clause.Expression     { return clause.Expr{} }
func (nonConcretePostgresDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v any) {
}
func (nonConcretePostgresDialector) QuoteTo(writer clause.Writer, str string)           {}
func (nonConcretePostgresDialector) Explain(sql string, vars ...any) string             { return sql }

func TestOpenPostgresFallbackWhenConcreteTypeMissing(t *testing.T) {
	// Not parallel: mutates package-level postgresOpen shared with other dialector tests.
	prev := postgresOpen
	t.Cleanup(func() { postgresOpen = prev })
	calls := 0
	postgresOpen = func(string) gorm.Dialector {
		calls++
		if calls == 1 {
			return nonConcretePostgresDialector{}
		}
		inner, ok := postgres.Open("host=127.0.0.1 user=wangbuke dbname=choysum sslmode=disable").(*postgres.Dialector)
		if !ok {
			t.Fatal("expected concrete postgres dialector on fallback")
		}
		return inner
	}
	d := openPostgres("dsn")
	if _, ok := d.(*postgres.Dialector); !ok {
		t.Fatalf("fallback dialector type %T", d)
	}
}

func TestOpenPostgresKeepsOptionalDialectorInterfaces(t *testing.T) {
	t.Parallel()

	d := openPostgres("host=127.0.0.1 user=wangbuke dbname=choysum sslmode=disable")
	if _, ok := d.(gorm.SavePointerDialectorInterface); !ok {
		t.Fatal("openPostgres dialector must implement SavePointerDialectorInterface")
	}
	if _, ok := d.(gorm.ErrorTranslator); !ok {
		t.Fatal("openPostgres dialector must implement ErrorTranslator")
	}
}

func TestBPCharAwareMigratorGetTypeAliases(t *testing.T) {
	t.Parallel()

	inner := stubTypeAliasMigrator{aliases: map[string][]string{
		"varchar": {"character varying"},
	}}
	m := bpcharAwareMigrator{Migrator: inner}

	got := m.GetTypeAliases("bpchar")
	if !containsFold(got, "char") || !containsFold(got, "character") {
		t.Fatalf("bpchar aliases = %v, want char+character", got)
	}

	got = m.GetTypeAliases("varchar")
	if len(got) != 1 || got[0] != "character varying" {
		t.Fatalf("varchar aliases = %v, want passthrough", got)
	}
}

func containsFold(values []string, want string) bool {
	for _, v := range values {
		if strings.EqualFold(v, want) {
			return true
		}
	}
	return false
}

func TestPostgresAutoMigrateSkipsBPCharCharRewrite(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("CHOYSUM_POSTGRES_DSN"))
	if dsn == "" {
		dsn = "host=127.0.0.1 user=wangbuke dbname=choysum sslmode=disable"
	}

	var sqlLog strings.Builder
	db, err := gorm.Open(openPostgres(dsn), &gorm.Config{
		Logger: logger.New(
			slog.NewLogLogger(slog.NewTextHandler(&sqlLog, nil), slog.LevelInfo),
			logger.Config{LogLevel: logger.Info},
		),
	})
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("postgres db handle: %v", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		t.Skipf("postgres ping: %v", err)
	}

	// Create once so the second AutoMigrate hits existing-column comparison (bpchar vs char).
	if err := db.AutoMigrate(&meta.Application{}); err != nil {
		t.Fatalf("AutoMigrate Application (create): %v", err)
	}
	sqlLog.Reset()
	if err := db.AutoMigrate(&meta.Application{}); err != nil {
		t.Fatalf("AutoMigrate Application (existing): %v", err)
	}
	logged := strings.ToLower(sqlLog.String())
	if strings.Contains(logged, `alter column "created_uid" type`) ||
		strings.Contains(logged, `alter column "updated_uid" type`) ||
		strings.Contains(logged, `alter column "deleted_uid" type`) ||
		strings.Contains(logged, `alter column "id" type`) {
		t.Fatalf("AutoMigrate still rewrote char/bpchar columns:\n%s", sqlLog.String())
	}
}
