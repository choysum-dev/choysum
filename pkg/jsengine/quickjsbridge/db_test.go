// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/shopspring/decimal"
)

type bridgeRecord struct {
	ID   int `gorm:"primaryKey"`
	Name string
}

func quickjsBridgeTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func quickjsBridgeTestConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Db: &config.DbConfig{
			Dialect:         "sqlite",
			DSN:             filepath.Join(t.TempDir(), "quickjsbridge.db"),
			MaxIdleConns:    7,
			MaxOpenConns:    9,
			ConnMaxLifetime: 60,
		},
	}
}

func TestNormalizeDBParams_PlainObjectIsJSONString(t *testing.T) {
	in := []interface{}{map[string]interface{}{"a": float64(1), "b": "x"}}

	out, err := normalizeDBParams(in)
	if err != nil {
		t.Fatalf("normalizeDBParams returned error: %v", err)
	}

	serialized, ok := out[0].(string)
	if !ok {
		t.Fatalf("expected string, got %T (%v)", out[0], out[0])
	}

	var got map[string]interface{}
	if err := json.Unmarshal([]byte(serialized), &got); err != nil {
		t.Fatalf("json.Unmarshal serialized param failed: %v", err)
	}

	if got["a"] != float64(1) {
		t.Fatalf("expected a=1, got %v", got["a"])
	}
	if got["b"] != "x" {
		t.Fatalf("expected b=x, got %v", got["b"])
	}
}

func TestNormalizeDBParams_BigIntMarker(t *testing.T) {
	in := []interface{}{map[string]interface{}{"$bigint": "42"}}

	out, err := normalizeDBParams(in)
	if err != nil {
		t.Fatalf("normalizeDBParams returned error: %v", err)
	}

	value, ok := out[0].(int64)
	if !ok {
		t.Fatalf("expected int64, got %T (%v)", out[0], out[0])
	}
	if value != 42 {
		t.Fatalf("expected 42, got %d", value)
	}
}

func TestNormalizeDBParams_BigDecimalMarker(t *testing.T) {
	in := []interface{}{map[string]interface{}{"$bigdecimal": "12.34"}}

	out, err := normalizeDBParams(in)
	if err != nil {
		t.Fatalf("normalizeDBParams returned error: %v", err)
	}

	value, ok := out[0].(decimal.Decimal)
	if !ok {
		t.Fatalf("expected decimal.Decimal, got %T (%v)", out[0], out[0])
	}

	expected, err := decimal.NewFromString("12.34")
	if err != nil {
		t.Fatalf("decimal.NewFromString failed: %v", err)
	}
	if !value.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected.String(), value.String())
	}
}

func TestNormalizeDBParams_BytesBase64Marker(t *testing.T) {
	in := []interface{}{map[string]interface{}{"$bytesBase64": "aGVsbG8="}}

	out, err := normalizeDBParams(in)
	if err != nil {
		t.Fatalf("normalizeDBParams returned error: %v", err)
	}

	value, ok := out[0].([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T (%v)", out[0], out[0])
	}
	if string(value) != "hello" {
		t.Fatalf("expected hello, got %q", string(value))
	}
}

func TestNormalizeDBParams_BytesBase64MarkerInvalid(t *testing.T) {
	in := []interface{}{map[string]interface{}{"$bytesBase64": "%%%not-base64%%%"}}

	_, err := normalizeDBParams(in)
	if err == nil {
		t.Fatal("expected error for invalid $bytesBase64")
	}
}

func TestWithDbSavepointSmoke(t *testing.T) {
	logger := quickjsBridgeTestLogger()
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(quickjsBridgeTestConfig(t)), logger)
	if err := runtimeScope.Session().AutoMigrate(&bridgeRecord{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	engine := newTestQuickjsEngine(t, WithDb("sqlite", logger))
	if err := engine.Load([]*jsengine.JsScript{{
		FileName: "db-savepoint-smoke.js",
		Content: `
			globalThis.$choysum.__rpc__ = async function(req) {
				await $choysum.db.savepoint('sp_bridge');
				await $choysum.db.execute('INSERT INTO bridge_record (name) VALUES (?)', JSON.stringify(['rolled-back']));
				await $choysum.db.rollbackToSavepoint('sp_bridge');
				await $choysum.db.releaseSavepoint('sp_bridge');
				await $choysum.db.execute('INSERT INTO bridge_record (name) VALUES (?)', JSON.stringify(['kept']));
				const rows = JSON.parse(await $choysum.db.query('SELECT name FROM bridge_record ORDER BY name', '[]'));
				return {
					id: req.id,
					result: { rows: rows.map((row) => row.name) },
					context: {},
				};
			};
		`,
	}}); err != nil {
		t.Fatalf("engine.Load: %v", err)
	}

	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		resp, err := engine.Execute(tx.Context(), &jsengine.JsRequest{Id: "savepoint-smoke", Service: "db"})
		if err != nil {
			return err
		}

		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("response result type = %T", resp.Result)
		}
		rows, ok := result["rows"].([]interface{})
		if !ok {
			t.Fatalf("rows type = %T", result["rows"])
		}
		if len(rows) != 1 || rows[0] != "kept" {
			t.Fatalf("rows = %#v, want [kept]", rows)
		}

		var rolledBackCount int64
		if err := txScope.Session().Model(&bridgeRecord{}).Where("name = ?", "rolled-back").Count(&rolledBackCount).Error; err != nil {
			return err
		}
		if rolledBackCount != 0 {
			t.Fatalf("rolled-back row count = %d, want 0", rolledBackCount)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Transactor.Required: %v", err)
	}

	var keptCount int64
	if err := runtimeScope.Session().Model(&bridgeRecord{}).Where("name = ?", "kept").Count(&keptCount).Error; err != nil {
		t.Fatalf("Count kept rows: %v", err)
	}
	if keptCount != 1 {
		t.Fatalf("kept row count = %d, want 1", keptCount)
	}
}

func TestIsUniqueConstraintErr(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"UNIQUE constraint failed: base_uo_m.category_id, base_uo_m.reference_slot_key", true},
		{"unique constraint failed: document_attachment_mutation_ledger.action", true},
		{"ERROR: duplicate key value violates unique constraint \"uidx\" (SQLSTATE 23505)", true},
		{"Duplicate entry 'x' for key 'PRIMARY'", true},
		{"cannot drop unique constraint uidx", false},
		{"failed to add unique index uidx", false},
		{"no such table: main.fd2test_field_default", false},
		{"database is locked", false},
		{"", false},
	}
	for _, tc := range cases {
		var err error
		if tc.msg != "" {
			err = errors.New(tc.msg)
		}
		if got := isUniqueConstraintErr(err); got != tc.want {
			t.Fatalf("isUniqueConstraintErr(%q)=%v want %v", tc.msg, got, tc.want)
		}
	}
	if isUniqueConstraintErr(nil) {
		t.Fatal("nil error must not be unique constraint")
	}
}

func TestWithDbQueryEmptyResultReturnsJSONArray(t *testing.T) {
	logger := quickjsBridgeTestLogger()
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(quickjsBridgeTestConfig(t)), logger)
	if err := runtimeScope.Session().AutoMigrate(&bridgeRecord{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	engine := newTestQuickjsEngine(t, WithDb("sqlite", logger))
	if err := engine.Load([]*jsengine.JsScript{{
		FileName: "db-empty-query.js",
		Content: `
			globalThis.$choysum.__rpc__ = async function(req) {
				const raw = await $choysum.db.query(
					"SELECT name FROM bridge_record WHERE name = 'missing-row'",
					'[]'
				);
				return { id: req.id, result: { raw: raw, parsed: JSON.parse(raw) }, context: {} };
			};
		`,
	}}); err != nil {
		t.Fatalf("engine.Load: %v", err)
	}

	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		resp, err := engine.Execute(tx.Context(), &jsengine.JsRequest{Id: "empty-query", Service: "db"})
		if err != nil {
			return err
		}
		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("result type = %T", resp.Result)
		}
		if raw, _ := result["raw"].(string); raw != "[]" {
			t.Fatalf("raw = %q, want []", raw)
		}
		parsed, ok := result["parsed"].([]interface{})
		if !ok {
			t.Fatalf("parsed type = %T (must be array, not null)", result["parsed"])
		}
		if len(parsed) != 0 {
			t.Fatalf("parsed len = %d", len(parsed))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Transactor.Required: %v", err)
	}
}

func TestWithDbQueryAndExecuteFailuresReachLogDBOpFailure(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(quickjsBridgeTestConfig(t)), logger)
	engine := newTestQuickjsEngine(t, WithDb("sqlite", logger))
	if err := engine.Load([]*jsengine.JsScript{{
		FileName: "db-fail-paths.js",
		Content: `
			globalThis.$choysum.__rpc__ = async function(req) {
				// Swallow rejections so the RPC can finish; coverage is asserted via Go logs.
				try { await $choysum.db.query('SELECT 1 FROM definitely_missing_table_xyz', '[]'); } catch (err) {}
				try {
					await $choysum.db.execute(
						'CREATE INDEX IF NOT EXISTS uidx_x ON definitely_missing_table_xyz (id)',
						'[]'
					);
				} catch (err) {}
				return { id: req.id, result: { ok: true }, context: {} };
			};
		`,
	}}); err != nil {
		t.Fatalf("engine.Load: %v", err)
	}

	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		resp, err := engine.Execute(tx.Context(), &jsengine.JsRequest{Id: "fail-paths", Service: "db"})
		if err != nil {
			return err
		}
		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("result type = %T", resp.Result)
		}
		if okFlag, _ := result["ok"].(bool); !okFlag {
			t.Fatalf("expected ok result, got %v", result)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Transactor.Required: %v", err)
	}
	logged := buf.String()
	if !strings.Contains(logged, "level=ERROR") {
		t.Fatalf("missing-table failures should log at ERROR, got %q", logged)
	}
	if !strings.Contains(logged, "db query failed") || !strings.Contains(logged, "db execute failed") {
		t.Fatalf("expected both query and execute failure logs, got %q", logged)
	}
}

func TestLogDBOpFailureLevels(t *testing.T) {
	logDBOpFailure(nil, "db query failed", errors.New("boom"), "") // must not panic

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	logDBOpFailure(logger, "db query failed", errors.New("UNIQUE constraint failed: t.c"), "INSERT INTO t (c) VALUES (1)")
	if !strings.Contains(buf.String(), "level=WARN") || strings.Contains(buf.String(), "level=ERROR") {
		t.Fatalf("DML unique violation should log exactly once at WARN, got %q", buf.String())
	}

	buf.Reset()
	logDBOpFailure(logger, "db execute failed", errors.New("index uidx already exists"), "CREATE UNIQUE INDEX uidx ON t (c)")
	if !strings.Contains(buf.String(), "level=WARN") || strings.Contains(buf.String(), "level=ERROR") {
		t.Fatalf("index-name collision should log exactly once at WARN, got %q", buf.String())
	}

	buf.Reset()
	logDBOpFailure(logger, "db execute failed", errors.New("UNIQUE constraint failed: t.c"), "CREATE UNIQUE INDEX IF NOT EXISTS uidx ON t (c)")
	if !strings.Contains(buf.String(), "level=ERROR") || strings.Contains(buf.String(), "level=WARN") {
		t.Fatalf("CREATE UNIQUE INDEX duplicate-row failure should log at ERROR, got %q", buf.String())
	}

	buf.Reset()
	logDBOpFailure(logger, "db execute failed", errors.New("duplicate key value violates unique constraint"), "ALTER TABLE t ADD CONSTRAINT uq UNIQUE (c)")
	if !strings.Contains(buf.String(), "level=ERROR") || strings.Contains(buf.String(), "level=WARN") {
		t.Fatalf("ALTER TABLE unique failure on duplicate data should log at ERROR, got %q", buf.String())
	}

	buf.Reset()
	logDBOpFailure(logger, "db query failed", errors.New("no such table: t"), "SELECT 1 FROM t")
	if !strings.Contains(buf.String(), "level=ERROR") {
		t.Fatalf("other failures should log at ERROR, got %q", buf.String())
	}
}

func TestIsIndexDDLAndIsDDLStmt(t *testing.T) {
	if !isIndexDDL("CREATE UNIQUE INDEX uidx ON t (c)") {
		t.Fatal("CREATE UNIQUE INDEX")
	}
	if !isIndexDDL("create index idx on t (c)") {
		t.Fatal("CREATE INDEX")
	}
	if isIndexDDL("CREATE TABLE t (a int, index int)") {
		t.Fatal("CREATE TABLE with index column must not count as index DDL")
	}
	if !isDDLStmt("ALTER TABLE t ADD CONSTRAINT uq UNIQUE (c)") {
		t.Fatal("ALTER TABLE is DDL")
	}
	if isDDLStmt("INSERT INTO t (c) VALUES (1)") {
		t.Fatal("INSERT is not DDL")
	}
}

func TestWithDbCreateUniqueIndexDuplicateRowsLogsError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	runtimeScope := defaultscope.NewDefaultScope(context.Background(), scopetest.FactoryInputFromConfig(quickjsBridgeTestConfig(t)), logger)
	session := runtimeScope.Session()
	if err := session.Exec(`CREATE TABLE uniq_dup_t (id INTEGER PRIMARY KEY, c TEXT)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := session.Exec(`INSERT INTO uniq_dup_t (id, c) VALUES (1, 'same'), (2, 'same')`).Error; err != nil {
		t.Fatalf("insert duplicates: %v", err)
	}

	engine := newTestQuickjsEngine(t, WithDb("sqlite", logger))
	if err := engine.Load([]*jsengine.JsScript{{
		FileName: "db-uniq-index-dup.js",
		Content: `
			globalThis.$choysum.__rpc__ = async function(req) {
				try {
					await $choysum.db.execute(
						'CREATE UNIQUE INDEX uidx_uniq_dup_t_c ON uniq_dup_t (c)',
						'[]'
					);
				} catch (err) {}
				return { id: req.id, result: { ok: true }, context: {} };
			};
		`,
	}}); err != nil {
		t.Fatalf("engine.Load: %v", err)
	}

	err := runtimeScope.Transactor().Required(context.Background(), func(txScope scope.Scope, tx scope.Transaction) error {
		resp, err := engine.Execute(tx.Context(), &jsengine.JsRequest{Id: "uniq-index-dup", Service: "db"})
		if err != nil {
			return err
		}
		if resp == nil || resp.Result == nil {
			t.Fatal("expected RPC result")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Transactor.Required: %v", err)
	}
	logged := buf.String()
	found := false
	for _, line := range strings.Split(logged, "\n") {
		if !strings.Contains(line, "db execute failed") {
			continue
		}
		found = true
		if !strings.Contains(line, "level=ERROR") {
			t.Fatalf("duplicate-row index DDL should log the failure at ERROR, got %q", line)
		}
	}
	if !found {
		t.Fatalf("expected execute failure log, got %q", logged)
	}
}
