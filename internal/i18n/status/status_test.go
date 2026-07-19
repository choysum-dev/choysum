// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package status_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/i18n/status"
)

func TestStatusReportCleanTree(t *testing.T) {
	root := t.TempDir()
	moduleRoot := filepath.Join(root, "demo")
	i18nDir := filepath.Join(moduleRoot, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Source with matching _t so pot-dirty stays clean when we skip extract of empty tree:
	// write pot + po in sync; skip pot check to avoid needing TS fixtures for clean case.
	pot := `msgid ""
msgstr ""

msgctxt "a@t"
msgid "Hello"
msgstr ""
`
	poBody := `msgid ""
msgstr "Language: zh_CN\n"

msgctxt "a@t"
msgid "Hello"
msgstr "你好"
`
	if err := os.WriteFile(filepath.Join(i18nDir, "demo.pot"), []byte(pot), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(poBody), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := status.StatusReport(status.Options{
		ModulesPath:  root,
		Modules:      []string{"demo"},
		Lang:         "zh_CN",
		SkipPotCheck: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExitCode(report) != 0 {
		t.Fatalf("expected clean, got:\n%s", status.FormatText(report))
	}
}

func TestStatusReportMissingFuzzyOrphan(t *testing.T) {
	root := t.TempDir()
	moduleRoot := filepath.Join(root, "demo")
	i18nDir := filepath.Join(moduleRoot, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pot := `msgid ""
msgstr ""

msgctxt "a@ok"
msgid "OK"
msgstr ""

msgctxt "a@hi"
msgid "Hello"
msgstr ""
`
	poBody := `msgid ""
msgstr ""

msgctxt "a@ok"
msgid "OK"
msgstr ""

#, fuzzy
msgctxt "a@hi"
msgid "Hello"
msgstr "你好"

#~ msgctxt "a@gone"
#~ msgid "Gone"
#~ msgstr "没了"
`
	if err := os.WriteFile(filepath.Join(i18nDir, "demo.pot"), []byte(pot), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(poBody), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := status.StatusReport(status.Options{
		ModulesPath:  root,
		Modules:      []string{"demo"},
		Lang:         "zh_CN",
		SkipPotCheck: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExitCode(report) != 1 {
		t.Fatal("expected non-zero exit")
	}
	kinds := map[status.IssueKind]int{}
	for _, issue := range report.Issues {
		kinds[issue.Kind]++
	}
	if kinds[status.IssueMissing] < 1 {
		t.Fatalf("expected missing, got %#v", report.Issues)
	}
	if kinds[status.IssueFuzzy] < 1 {
		t.Fatalf("expected fuzzy, got %#v", report.Issues)
	}
	if kinds[status.IssueOrphan] < 1 {
		t.Fatalf("expected orphan obsolete, got %#v", report.Issues)
	}
	// Orphans alone should not fail by default.
	orphanOnly := &status.Report{Lang: "zh_CN"}
	for _, issue := range report.Issues {
		if issue.Kind == status.IssueOrphan {
			orphanOnly.Issues = append(orphanOnly.Issues, issue)
		}
	}
	if status.ExitCode(orphanOnly) != 0 {
		t.Fatal("orphan-only should be exit 0 by default")
	}
	if status.ExitCode(orphanOnly, status.ExitOptions{StrictOrphan: true}) != 1 {
		t.Fatal("strict orphan should be exit 1")
	}
}

func TestStatusReportPotDirty(t *testing.T) {
	root := t.TempDir()
	moduleRoot := filepath.Join(root, "demo")
	srcDir := filepath.Join(moduleRoot, "service")
	i18nDir := filepath.Join(moduleRoot, "i18n")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a.ts"), []byte(`
export function f() {
  return _t('Hello', { scope: 'web/a@title' });
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Committed pot is empty of the Hello entry → pot-dirty.
	if err := os.WriteFile(filepath.Join(i18nDir, "demo.pot"), []byte(`msgid ""
msgstr ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(`msgid ""
msgstr ""
`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := status.StatusReport(status.Options{
		ModulesPath: root,
		Modules:     []string{"demo"},
		Lang:        "zh_CN",
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Kind == status.IssuePotDirty {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected pot-dirty, got:\n%s", status.FormatText(report))
	}
	if status.ExitCode(report) != 1 {
		t.Fatal("expected non-zero")
	}
}

func TestStatusReportNoPo(t *testing.T) {
	root := t.TempDir()
	moduleRoot := filepath.Join(root, "demo")
	i18nDir := filepath.Join(moduleRoot, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "demo.pot"), []byte(`msgid ""
msgstr ""

msgctxt "a@t"
msgid "Hello"
msgstr ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := status.StatusReport(status.Options{
		ModulesPath:  root,
		Modules:      []string{"demo"},
		Lang:         "zh_CN",
		SkipPotCheck: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 1 || report.Issues[0].Kind != status.IssueNoPo {
		t.Fatalf("expected no-po, got %#v", report.Issues)
	}
}

func TestStatusReportValidationAndFormatText(t *testing.T) {
	if _, err := status.StatusReport(status.Options{}); err == nil {
		t.Fatal("expected lang required")
	}
	if _, err := status.StatusReport(status.Options{Lang: "bad lang!"}); err == nil {
		t.Fatal("expected invalid lang")
	}
	if _, err := status.StatusReport(status.Options{Lang: "zh_CN"}); err == nil {
		t.Fatal("expected modules path required")
	}
	if _, err := status.StatusReport(status.Options{Lang: "zh_CN", ModulesPath: t.TempDir()}); err == nil {
		t.Fatal("expected modules required")
	}
	if status.ExitCode(nil) != 0 {
		t.Fatal("nil report exit 0")
	}

	root := t.TempDir()
	moduleRoot := filepath.Join(root, "demo")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := status.StatusReport(status.Options{
		ModulesPath:  root,
		Modules:      []string{"demo"},
		Lang:         "zh_CN",
		SkipPotCheck: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := status.FormatText(report)
	if text == "" {
		t.Fatal("expected format text")
	}
}
