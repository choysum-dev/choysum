// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package database

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"gorm.io/gorm"
)

type fakeCompatCol struct {
	name, dbType  string
	nullable      bool
	nullableOK    bool
	length        int64
	lengthOK      bool
	primaryKey    bool
	primaryKeyOK  bool
	primaryKeySet bool
}

func (f fakeCompatCol) Name() string               { return f.name }
func (f fakeCompatCol) DatabaseTypeName() string   { return f.dbType }
func (f fakeCompatCol) ColumnType() (string, bool) { return f.dbType, true }
func (f fakeCompatCol) PrimaryKey() (bool, bool) {
	if f.primaryKeySet {
		return f.primaryKey, f.primaryKeyOK
	}
	return false, true
}
func (f fakeCompatCol) AutoIncrement() (bool, bool)       { return false, true }
func (f fakeCompatCol) Length() (int64, bool)             { return f.length, f.lengthOK }
func (f fakeCompatCol) DecimalSize() (int64, int64, bool) { return 0, 0, false }
func (f fakeCompatCol) Nullable() (bool, bool)            { return f.nullable, f.nullableOK }
func (f fakeCompatCol) Unique() (bool, bool)              { return false, true }
func (f fakeCompatCol) ScanType() reflect.Type            { return reflect.TypeOf("") }
func (f fakeCompatCol) Comment() (string, bool)           { return "", false }
func (f fakeCompatCol) DefaultValue() (string, bool)      { return "", false }

func restoreAuthTokenCompatHooks(t *testing.T) {
	t.Helper()
	origDialect := authTokenDialectName
	origCols := authTokenColumnTypes
	origCount := authTokenTableCount
	origNulls := authTokenNullCount
	origDrop := authTokenDropAndRecreate
	origExec := authTokenExecSQL
	t.Cleanup(func() {
		authTokenDialectName = origDialect
		authTokenColumnTypes = origCols
		authTokenTableCount = origCount
		authTokenNullCount = origNulls
		authTokenDropAndRecreate = origDrop
		authTokenExecSQL = origExec
	})
}

func TestEnsureAuthTokenSchemaCompatibility_Coverage(t *testing.T) {
	restoreAuthTokenCompatHooks(t)

	if err := ensureAuthTokenSchemaCompatibility(nil, "auth_token"); err != nil {
		t.Fatalf("nil db: %v", err)
	}

	session := newDatabaseTestSession(t, "compat-cov.db")
	authTokenColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return nil, errors.New("column types boom")
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "column types boom") {
		t.Fatalf("column types error: %v", err)
	}

	authTokenColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		return []gorm.ColumnType{
			fakeCompatCol{name: "id", dbType: "char", nullableOK: true},
			fakeCompatCol{name: "user_id", dbType: "char", nullableOK: true},
			fakeCompatCol{name: "token_id", dbType: "varchar", nullable: false, nullableOK: true},
			fakeCompatCol{name: "token_type", dbType: "varchar", nullable: false, nullableOK: true},
			fakeCompatCol{name: "expires_at", dbType: "datetime", nullable: false, nullableOK: true},
		}, nil
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err != nil {
		t.Fatalf("compatible schema: %v", err)
	}

	legacyCols := []gorm.ColumnType{
		fakeCompatCol{name: "id", dbType: "varchar", nullableOK: true},
		fakeCompatCol{name: "user_id", dbType: "integer", nullableOK: true},
		fakeCompatCol{name: "token_id", dbType: "varchar", nullable: true, nullableOK: true},
		fakeCompatCol{name: "token_type", dbType: "varchar", nullable: true, nullableOK: true},
		fakeCompatCol{name: "expires_at", dbType: "datetime", nullable: true, nullableOK: true},
	}
	authTokenColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) { return legacyCols, nil }
	authTokenTableCount = func(*gorm.DB, string) (int64, error) { return 0, errors.New("count boom") }
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "count boom") {
		t.Fatalf("count error: %v", err)
	}

	counts := 0
	authTokenTableCount = func(*gorm.DB, string) (int64, error) {
		counts++
		if counts == 1 {
			return 0, nil
		}
		return 0, errors.New("recheck boom")
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "recheck boom") {
		t.Fatalf("recheck error: %v", err)
	}

	counts = 0
	authTokenTableCount = func(*gorm.DB, string) (int64, error) { return 0, nil }
	authTokenDropAndRecreate = func(*gorm.DB, string) error { return errors.New("drop boom") }
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "drop boom") {
		t.Fatalf("drop error: %v", err)
	}
	authTokenDropAndRecreate = func(db *gorm.DB, tableName string) error {
		if err := db.Migrator().DropTable(tableName); err != nil && !strings.Contains(err.Error(), "no such table") {
			return err
		}
		return db.AutoMigrate(&revokedTokenRecord{})
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err != nil {
		t.Fatalf("empty recreate: %v", err)
	}

	// Empty check then race: second count sees rows → fall through to non-postgres error.
	authTokenColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) { return legacyCols, nil }
	counts = 0
	authTokenTableCount = func(*gorm.DB, string) (int64, error) {
		counts++
		if counts == 1 {
			return 0, nil
		}
		return 2, nil
	}
	authTokenDialectName = func(*gorm.DB) string { return "sqlite" }
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("non-empty non-postgres: %v", err)
	}

	authTokenDialectName = func(*gorm.DB) string { return "postgres" }
	authTokenTableCount = func(*gorm.DB, string) (int64, error) { return 3, nil }
	var execs []string
	authTokenExecSQL = func(_ *gorm.DB, sql string) error {
		execs = append(execs, sql)
		if strings.Contains(sql, "DROP DEFAULT") && strings.Contains(sql, "id") {
			return errors.New("id drop default boom")
		}
		return nil
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "id drop default boom") {
		t.Fatalf("id drop default: %v", err)
	}

	execs = nil
	authTokenExecSQL = func(_ *gorm.DB, sql string) error {
		execs = append(execs, sql)
		if strings.Contains(sql, "TYPE char(20)") && strings.Contains(sql, " id ") {
			return errors.New("id type boom")
		}
		return nil
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "id type boom") {
		t.Fatalf("id type: %v", err)
	}

	execs = nil
	authTokenExecSQL = func(_ *gorm.DB, sql string) error {
		execs = append(execs, sql)
		if strings.Contains(sql, "user_id DROP DEFAULT") {
			return errors.New("user_id drop default boom")
		}
		return nil
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "user_id drop default boom") {
		t.Fatalf("user_id drop default: %v", err)
	}

	execs = nil
	authTokenExecSQL = func(_ *gorm.DB, sql string) error {
		execs = append(execs, sql)
		if strings.Contains(sql, "user_id TYPE") {
			return errors.New("user_id type boom")
		}
		return nil
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "user_id type boom") {
		t.Fatalf("user_id type: %v", err)
	}

	authTokenExecSQL = func(*gorm.DB, string) error { return nil }
	authTokenNullCount = func(*gorm.DB, string, string) (int64, error) { return 0, errors.New("null count boom") }
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "null count boom") {
		t.Fatalf("null count: %v", err)
	}

	authTokenNullCount = func(*gorm.DB, string, string) (int64, error) { return 2, nil }
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "NULL row") {
		t.Fatalf("null rows: %v", err)
	}

	authTokenNullCount = func(*gorm.DB, string, string) (int64, error) { return 0, nil }
	authTokenExecSQL = func(_ *gorm.DB, sql string) error {
		if strings.Contains(sql, "SET NOT NULL") {
			return errors.New("set not null boom")
		}
		return nil
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "set not null boom") {
		t.Fatalf("set not null: %v", err)
	}

	authTokenExecSQL = func(*gorm.DB, string) error { return nil }
	colCalls := 0
	authTokenColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		colCalls++
		if colCalls == 1 {
			return legacyCols, nil
		}
		return nil, errors.New("repaired types boom")
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "repaired types boom") {
		t.Fatalf("repaired types: %v", err)
	}

	colCalls = 0
	authTokenColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		colCalls++
		if colCalls == 1 {
			return legacyCols, nil
		}
		return legacyCols, nil // still mismatched
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err == nil || !strings.Contains(err.Error(), "remains incompatible") {
		t.Fatalf("still mismatched: %v", err)
	}

	colCalls = 0
	execs = nil
	authTokenColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		colCalls++
		if colCalls == 1 {
			return legacyCols, nil
		}
		return []gorm.ColumnType{
			fakeCompatCol{name: "id", dbType: "char", nullableOK: true},
			fakeCompatCol{name: "user_id", dbType: "char", nullableOK: true},
			fakeCompatCol{name: "token_id", dbType: "varchar", nullable: false, nullableOK: true},
			fakeCompatCol{name: "token_type", dbType: "varchar", nullable: false, nullableOK: true},
			fakeCompatCol{name: "expires_at", dbType: "datetime", nullable: false, nullableOK: true},
			// missing optional columns exercise !ok continue
		}, nil
	}
	authTokenExecSQL = func(_ *gorm.DB, sql string) error {
		execs = append(execs, sql)
		return nil
	}
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err != nil {
		t.Fatalf("postgres repair success: %v", err)
	}
	joined := strings.Join(execs, "\n")
	for _, want := range []string{
		`ALTER TABLE "auth_token" ALTER COLUMN id DROP DEFAULT`,
		`ALTER TABLE "auth_token" ALTER COLUMN id TYPE char(20) USING id::text`,
		`ALTER TABLE "auth_token" ALTER COLUMN user_id DROP DEFAULT`,
		`ALTER TABLE "auth_token" ALTER COLUMN user_id TYPE char(20) USING user_id::text`,
		`ALTER TABLE "auth_token" ALTER COLUMN token_id SET NOT NULL`,
		`ALTER TABLE "auth_token" ALTER COLUMN token_type SET NOT NULL`,
		`ALTER TABLE "auth_token" ALTER COLUMN expires_at SET NOT NULL`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing SQL %q in %#v", want, execs)
		}
	}

	// Nullable unknown (!ok) skips SET NOT NULL; repair still succeeds when repaired types match.
	colCalls = 0
	authTokenColumnTypes = func(*gorm.DB, string) ([]gorm.ColumnType, error) {
		colCalls++
		if colCalls == 1 {
			return []gorm.ColumnType{
				fakeCompatCol{name: "id", dbType: "varchar"},
				fakeCompatCol{name: "token_id", dbType: "varchar", nullableOK: false},
				fakeCompatCol{name: "token_type", dbType: "varchar", nullable: false, nullableOK: true},
			}, nil
		}
		return []gorm.ColumnType{
			fakeCompatCol{name: "id", dbType: "char"},
			fakeCompatCol{name: "user_id", dbType: "char"},
			fakeCompatCol{name: "token_id", dbType: "varchar", nullable: false, nullableOK: true},
			fakeCompatCol{name: "token_type", dbType: "varchar", nullable: false, nullableOK: true},
			fakeCompatCol{name: "expires_at", dbType: "datetime", nullable: false, nullableOK: true},
		}, nil
	}
	authTokenTableCount = func(*gorm.DB, string) (int64, error) { return 1, nil }
	authTokenExecSQL = func(*gorm.DB, string) error { return nil }
	if err := ensureAuthTokenSchemaCompatibility(session.DB, "auth_token"); err != nil {
		t.Fatalf("nullable unknown skip: %v", err)
	}
}

func TestAuthTokenCompatDefaultHookBodies(t *testing.T) {
	// Call named defaults directly so coverage hits store.go bodies even when
	// other tests have swapped the package vars.
	session := newDatabaseTestSession(t, "default-hooks.db")
	if got := defaultAuthTokenDialectName(session.DB); got != "sqlite" {
		t.Fatalf("dialect=%q", got)
	}
	if err := session.AutoMigrate(&revokedTokenRecord{}); err != nil {
		t.Fatal(err)
	}
	if _, err := defaultAuthTokenColumnTypes(session.DB, "auth_token"); err != nil {
		t.Fatal(err)
	}
	if n, err := defaultAuthTokenTableCount(session.DB, "auth_token"); err != nil || n != 0 {
		t.Fatalf("count=%d err=%v", n, err)
	}
	if n, err := defaultAuthTokenNullCount(session.DB, "auth_token", "token_id"); err != nil || n != 0 {
		t.Fatalf("nulls=%d err=%v", n, err)
	}
	if err := defaultAuthTokenExecSQL(session.DB, "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	if err := defaultAuthTokenDropAndRecreate(session.DB, "auth_token"); err != nil {
		t.Fatalf("drop+recreate: %v", err)
	}

	// Error paths on a closed connection.
	sqlDB, err := session.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	if err := defaultAuthTokenDropAndRecreate(session.DB, "auth_token"); err == nil {
		t.Fatal("expected DropTable error on closed db")
	}
	if err := defaultAuthTokenExecSQL(session.DB, "SELECT 1"); err == nil {
		t.Fatal("expected Exec error on closed db")
	}
	if _, err := defaultAuthTokenNullCount(session.DB, "auth_token", "token_id"); err == nil {
		t.Fatal("expected NullCount error on closed db")
	}
}

func TestAuthTokenSchemaMismatch_PartialColumns(t *testing.T) {
	if authTokenSchemaMismatch(nil) {
		t.Fatal("empty should not mismatch")
	}
	if authTokenSchemaMismatch([]gorm.ColumnType{
		fakeCompatCol{name: "other", dbType: "varchar"},
	}) {
		t.Fatal("unrelated columns")
	}
	if !authTokenSchemaMismatch([]gorm.ColumnType{
		fakeCompatCol{name: "id", dbType: "integer"},
	}) {
		t.Fatal("integer id should mismatch")
	}
	if !authTokenSchemaMismatch([]gorm.ColumnType{
		fakeCompatCol{name: "expires_at", dbType: "datetime", nullable: true, nullableOK: true},
	}) {
		t.Fatal("nullable expires_at should mismatch")
	}
}

func TestIsVarcharAndIntegerLikeWhitespace(t *testing.T) {
	t.Parallel()
	if !isVarcharDBType("varchar (20)") {
		t.Fatal("varchar (20)")
	}
	if !isIntegerLikeDBType("numeric (10, 2)") {
		t.Fatal("numeric (10, 2)")
	}
	if !isIntegerLikeDBType("INT4") {
		t.Fatal("INT4")
	}
	if authTokenIDNeedsTypeRepair("char(20)") {
		t.Fatal("char should not need repair")
	}
	if !authTokenIDNeedsTypeRepair("varchar (20)") {
		t.Fatal("varchar (20) needs repair")
	}
}
