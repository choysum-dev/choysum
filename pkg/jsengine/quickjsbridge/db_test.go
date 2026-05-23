// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsbridge

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
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
