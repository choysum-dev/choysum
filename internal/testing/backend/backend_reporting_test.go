// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
)

func TestResolveServiceEntryPoint(t *testing.T) {
	addonsPath := t.TempDir()
	app := "auth"
	appDir := filepath.Join(addonsPath, app)
	if err := os.MkdirAll(filepath.Join(appDir, "service"), 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}

	runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath}}

	entryRel := "service/custom.ts"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.json"), []byte(`{"entryPoints":{"service":"`+entryRel+`"}}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	got, err := resolveServiceEntryPoint(runtimeScope, app)
	if err != nil {
		t.Fatalf("resolveServiceEntryPoint error: %v", err)
	}
	if got != filepath.Join(appDir, entryRel) {
		t.Fatalf("unexpected entry path: %q", got)
	}

	absEntry := filepath.Join(appDir, "service", "abs.ts")
	if err := os.WriteFile(filepath.Join(appDir, "manifest.json"), []byte(`{"entryPoints":{"service":"`+absEntry+`"}}`), 0o644); err != nil {
		t.Fatalf("rewrite manifest: %v", err)
	}
	got, err = resolveServiceEntryPoint(runtimeScope, app)
	if err != nil || got != absEntry {
		t.Fatalf("expected absolute entry path, got %q err=%v", got, err)
	}

	if err := os.WriteFile(filepath.Join(appDir, "manifest.json"), []byte(`{"entryPoints":{}}`), 0o644); err != nil {
		t.Fatalf("rewrite manifest: %v", err)
	}
	fallback := filepath.Join(appDir, "service", "index.ts")
	if err := os.WriteFile(fallback, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("write fallback index: %v", err)
	}
	got, err = resolveServiceEntryPoint(runtimeScope, app)
	if err != nil || got != fallback {
		t.Fatalf("expected fallback entry path, got %q err=%v", got, err)
	}

	if err := os.Remove(fallback); err != nil {
		t.Fatalf("remove fallback index: %v", err)
	}
	_, err = resolveServiceEntryPoint(runtimeScope, app)
	if err == nil || !strings.Contains(err.Error(), "service entry point not found") {
		t.Fatalf("expected missing entry error, got %v", err)
	}

	_, err = resolveServiceEntryPoint(runtimeScope, "missing")
	if err == nil || !strings.Contains(err.Error(), "manifest not found") {
		t.Fatalf("expected missing manifest error, got %v", err)
	}
}

func TestParseReport(t *testing.T) {
	_, err := parseReport("bad")
	if err == nil || !strings.Contains(err.Error(), "unexpected result type") {
		t.Fatalf("expected type error, got %v", err)
	}

	report, err := parseReport(map[string]any{
		"total":        float64(2),
		"passed":       int64(1),
		"failed":       int(1),
		"coverageJSON": "{\"ok\":true}",
		"cases": []any{
			map[string]any{"name": "case1", "ok": true, "durationMs": int64(120)},
			map[string]any{"name": "case2", "ok": false, "durationMs": float64(30), "error": map[string]any{"message": "boom", "stack": "stack line"}},
		},
	})
	if err != nil {
		t.Fatalf("parseReport error: %v", err)
	}
	if report.total != 2 || report.passed != 1 || report.failed != 1 {
		t.Fatalf("unexpected totals: %#v", report)
	}
	if report.coverageJSON == "" || len(report.cases) != 2 {
		t.Fatalf("unexpected parsed report fields: %#v", report)
	}
	if report.cases[1].errMsg != "boom" || report.cases[1].errStack != "stack line" {
		t.Fatalf("unexpected error fields: %#v", report.cases[1])
	}
}

func TestWriteTAPAndJUnit(t *testing.T) {
	report := parsedReport{
		total:  2,
		passed: 1,
		failed: 1,
		cases: []parsedCase{
			{name: "ok case", ok: true, durationMs: 10},
			{name: "fail case", ok: false, durationMs: 20, errMsg: "first line\nsecond line", errStack: "trace"},
		},
	}

	tapPath := filepath.Join(t.TempDir(), "out.tap")
	f, err := os.Create(tapPath)
	if err != nil {
		t.Fatalf("create tap file: %v", err)
	}
	writeTAP(f, report)
	if err := f.Close(); err != nil {
		t.Fatalf("close tap file: %v", err)
	}
	tap, err := os.ReadFile(tapPath)
	if err != nil {
		t.Fatalf("read tap file: %v", err)
	}
	tapText := string(tap)
	checks := []string{"TAP version 13", "ok 1 - ok case", "not ok 2 - fail case", "# first line", "# trace", "1..2"}
	for _, s := range checks {
		if !strings.Contains(tapText, s) {
			t.Fatalf("tap output missing %q: %s", s, tapText)
		}
	}

	failed, err := writeJUnitIfNeeded("auth", report, "")
	if err != nil || !failed {
		t.Fatalf("expected failed=true without junit path, failed=%v err=%v", failed, err)
	}

	junitPath := filepath.Join(t.TempDir(), "report.xml")
	failed, err = writeJUnitIfNeeded("auth", report, junitPath)
	if err != nil || !failed {
		t.Fatalf("expected failed junit write, failed=%v err=%v", failed, err)
	}
	xmlRaw, err := os.ReadFile(junitPath)
	if err != nil {
		t.Fatalf("read junit: %v", err)
	}
	xmlText := string(xmlRaw)
	xmlChecks := []string{"<testsuites", "<testsuite", `name="auth"`, `tests="2"`, `failures="1"`, `classname="auth"`, "<failure", "first line"}
	for _, s := range xmlChecks {
		if !strings.Contains(xmlText, s) {
			t.Fatalf("junit output missing %q: %s", s, xmlText)
		}
	}
}
