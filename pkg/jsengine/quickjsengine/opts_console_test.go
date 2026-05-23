// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func decodeConsoleRecords(t *testing.T, logged string) []map[string]any {
	t.Helper()
	logged = strings.TrimSpace(logged)
	if logged == "" {
		return nil
	}
	lines := strings.Split(logged, "\n")
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("unmarshal log line %q: %v", line, err)
		}
		records = append(records, record)
	}
	return records
}

func consoleRecordByLevel(t *testing.T, records []map[string]any, consoleLevel string) map[string]any {
	t.Helper()
	for _, record := range records {
		if fmt.Sprint(record["console_level"]) == consoleLevel {
			return record
		}
	}
	t.Fatalf("console level %q not found in %#v", consoleLevel, records)
	return nil
}

func TestWithConsoleLogsStructuredEnvelopeAcrossLevels(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	engine := newTestQuickjsEngine(t, WithConsole(logger))

	_ = evalString(t, engine, `
		(() => {
			console.trace("trace", { a: 1 });
			console.debug("debug", new Map([["key", 1]]));
			console.info("info");
			console.log("log");
			console.warn("warn");
			console.error("error", new Error("boom"));
			return "done";
		})()
	`)

	records := decodeConsoleRecords(t, buffer.String())
	if len(records) != 6 {
		t.Fatalf("record count = %d, want 6; logs=%q", len(records), buffer.String())
	}

	wantLevels := map[string]string{
		"trace": "DEBUG",
		"debug": "DEBUG",
		"info":  "INFO",
		"log":   "INFO",
		"warn":  "WARN",
		"error": "ERROR",
	}
	wantFragments := map[string]string{
		"trace": "trace",
		"debug": "dataType",
		"info":  "info",
		"log":   "log",
		"warn":  "warn",
		"error": "boom",
	}
	for consoleLevel, wantLevel := range wantLevels {
		record := consoleRecordByLevel(t, records, consoleLevel)
		if got := fmt.Sprint(record["msg"]); got != jsConsoleMessage {
			t.Fatalf("message for %s = %q, want %q", consoleLevel, got, jsConsoleMessage)
		}
		if got := fmt.Sprint(record["level"]); got != wantLevel {
			t.Fatalf("level for %s = %q, want %q", consoleLevel, got, wantLevel)
		}
		if got := fmt.Sprint(record["emitter"]); got != "js_console" {
			t.Fatalf("emitter for %s = %q, want js_console", consoleLevel, got)
		}
		if got, ok := record["passthrough"].(bool); !ok || !got {
			t.Fatalf("passthrough for %s = %#v, want true", consoleLevel, record["passthrough"])
		}
		if got := fmt.Sprint(record["console_text"]); !strings.Contains(got, wantFragments[consoleLevel]) {
			t.Fatalf("console_text for %s = %q, want fragment %q", consoleLevel, got, wantFragments[consoleLevel])
		}
	}

	errorRecord := consoleRecordByLevel(t, records, "error")
	if got := fmt.Sprint(errorRecord["error"]); !strings.Contains(got, "boom") {
		t.Fatalf("error field = %q, want boom fragment", got)
	}
}

func TestWithConsoleDeprecationWarningsFollowNormalErrorPath(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	engine := newTestQuickjsEngine(t, WithConsole(logger))

	_ = evalString(t, engine, `
		(() => {
			console.error("DEPRECATION WARNING", "legacy api");
			return "done";
		})()
	`)

	records := decodeConsoleRecords(t, buffer.String())
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1; logs=%q", len(records), buffer.String())
	}
	record := records[0]
	if got := fmt.Sprint(record["msg"]); got != jsConsoleMessage {
		t.Fatalf("message = %q, want %q", got, jsConsoleMessage)
	}
	if got := fmt.Sprint(record["level"]); got != "ERROR" {
		t.Fatalf("level = %q, want ERROR", got)
	}
	if got := fmt.Sprint(record["console_level"]); got != "error" {
		t.Fatalf("console_level = %q, want error", got)
	}
	if _, ok := record["policy_reason"]; ok {
		t.Fatalf("policy_reason should be absent, got %#v", record["policy_reason"])
	}
	if got := fmt.Sprint(record["console_text"]); !strings.Contains(got, "DEPRECATION WARNING") {
		t.Fatalf("console_text = %q, want deprecation warning fragment", got)
	}
}
