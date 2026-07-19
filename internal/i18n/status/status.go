// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package status reports terminology catalog quality for CI and local gates.
package status

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/internal/i18n/extract"
	"github.com/choysum-dev/choysum/internal/i18n/langcode"
	"github.com/choysum-dev/choysum/internal/i18n/po"
)

// IssueKind classifies one status finding.
type IssueKind string

const (
	IssueMissing  IssueKind = "missing"
	IssueFuzzy    IssueKind = "fuzzy"
	IssueOrphan   IssueKind = "orphan"    // obsolete (#~) still in PO (D12a)
	IssuePotDirty IssueKind = "pot-dirty" // committed .pot drifts from extract
	IssueNoPo     IssueKind = "no-po"     // pot exists but lang.po missing
)

// Issue is one finding for a module/lang (or pot-only for pot-dirty).
type Issue struct {
	Module string    `json:"module"`
	Lang   string    `json:"lang,omitempty"`
	Kind   IssueKind `json:"kind"`
	Scope  string    `json:"scope,omitempty"`
	Src    string    `json:"src,omitempty"`
	Detail string    `json:"detail,omitempty"`
}

// Report aggregates findings across modules.
type Report struct {
	Lang   string  `json:"lang"`
	Issues []Issue `json:"issues"`
}

// Options configures StatusReport.
type Options struct {
	ModulesPath string
	Modules     []string
	Lang        string
	PathAlias   map[string]string
	// SkipPotCheck disables extract-vs-committed .pot drift detection.
	SkipPotCheck bool
}

// ExitOptions controls which issue kinds fail the process.
type ExitOptions struct {
	// StrictOrphan makes obsolete/orphan findings non-zero (default: report only).
	StrictOrphan bool
}

// StatusReport scans module PO/POT catalogs for missing/fuzzy/orphan/pot-dirty.
// It is filesystem-only (no DB) so CI can run with lightweight CLI scope.
func StatusReport(opts Options) (*Report, error) {
	lang := strings.TrimSpace(opts.Lang)
	if lang == "" {
		return nil, fmt.Errorf("lang is required")
	}
	if !langcode.Valid(lang) {
		return nil, fmt.Errorf("invalid lang format")
	}
	modulesPath := strings.TrimSpace(opts.ModulesPath)
	if modulesPath == "" {
		return nil, fmt.Errorf("modules path is empty")
	}
	modules := opts.Modules
	if len(modules) == 0 {
		return nil, fmt.Errorf("no modules specified")
	}

	report := &Report{Lang: lang}
	for _, moduleName := range modules {
		moduleName = strings.TrimSpace(moduleName)
		if moduleName == "" {
			continue
		}
		moduleRoot := filepath.Join(modulesPath, moduleName)
		issues, err := checkModule(moduleRoot, moduleName, lang, opts.PathAlias, opts.SkipPotCheck)
		if err != nil {
			return nil, fmt.Errorf("status %s: %w", moduleName, err)
		}
		report.Issues = append(report.Issues, issues...)
	}
	sort.Slice(report.Issues, func(i, j int) bool {
		a, b := report.Issues[i], report.Issues[j]
		if a.Module != b.Module {
			return a.Module < b.Module
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		return a.Src < b.Src
	})
	return report, nil
}

// ExitCode returns 0 when clean of blocking issues, 1 when any blocking issue exists.
// Orphan findings are informational by default (obsolete PO history is expected per D12a);
// pass StrictOrphan to treat them as failures.
func ExitCode(report *Report, opts ...ExitOptions) int {
	if report == nil || len(report.Issues) == 0 {
		return 0
	}
	strictOrphan := false
	if len(opts) > 0 {
		strictOrphan = opts[0].StrictOrphan
	}
	for _, issue := range report.Issues {
		switch issue.Kind {
		case IssueOrphan:
			if strictOrphan {
				return 1
			}
		default:
			return 1
		}
	}
	return 0
}

func checkModule(moduleRoot, moduleName, lang string, pathAlias map[string]string, skipPot bool) ([]Issue, error) {
	i18nDir := filepath.Join(moduleRoot, "i18n")
	potPath := filepath.Join(i18nDir, moduleName+".pot")
	poPath := filepath.Join(i18nDir, lang+".po")

	if _, err := os.Stat(i18nDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no catalog for this module
		}
		return nil, err
	}

	var issues []Issue

	if !skipPot {
		if potIssues, err := checkPotDirty(moduleRoot, moduleName, potPath, pathAlias); err != nil {
			return nil, err
		} else {
			issues = append(issues, potIssues...)
		}
	}

	potKeys, err := loadPotKeys(potPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No pot yet — skip po quality unless po exists alone.
			potKeys = map[string]struct{}{}
		} else {
			return nil, err
		}
	}

	if _, err := os.Stat(poPath); err != nil {
		if os.IsNotExist(err) {
			if len(potKeys) > 0 {
				issues = append(issues, Issue{
					Module: moduleName,
					Lang:   lang,
					Kind:   IssueNoPo,
					Detail: fmt.Sprintf("missing %s (run: choysum i18n sync %s --lang %s)", filepath.Base(poPath), moduleName, lang),
				})
			}
			return issues, nil
		}
		return nil, err
	}

	raw, err := os.ReadFile(poPath)
	if err != nil {
		return nil, err
	}
	entries, err := po.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", poPath, err)
	}

	for _, e := range entries {
		if e.Msgid == "" && e.Msgctxt == "" {
			continue // header
		}
		if e.Obsolete {
			issues = append(issues, Issue{
				Module: moduleName,
				Lang:   lang,
				Kind:   IssueOrphan,
				Scope:  e.Msgctxt,
				Src:    e.Msgid,
				Detail: "obsolete entry retained in PO (D12a)",
			})
			continue
		}
		if hasFlag(e.Flags, "fuzzy") {
			issues = append(issues, Issue{
				Module: moduleName,
				Lang:   lang,
				Kind:   IssueFuzzy,
				Scope:  e.Msgctxt,
				Src:    e.Msgid,
			})
		}
		if strings.TrimSpace(e.Msgstr) == "" {
			issues = append(issues, Issue{
				Module: moduleName,
				Lang:   lang,
				Kind:   IssueMissing,
				Scope:  e.Msgctxt,
				Src:    e.Msgid,
			})
		}
		// Entries present in PO but absent from pot (and not obsolete) are also orphan-like.
		if len(potKeys) > 0 {
			key := e.Key()
			if _, ok := potKeys[key]; !ok {
				issues = append(issues, Issue{
					Module: moduleName,
					Lang:   lang,
					Kind:   IssueOrphan,
					Scope:  e.Msgctxt,
					Src:    e.Msgid,
					Detail: "present in PO but not in pot (run sync)",
				})
			}
		}
	}
	return issues, nil
}

func checkPotDirty(moduleRoot, moduleName, potPath string, pathAlias map[string]string) ([]Issue, error) {
	result, err := extract.ExtractModule(moduleRoot, moduleName, pathAlias, false)
	if err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}
	liveKeys := map[string]struct{}{}
	for _, e := range result.Entries {
		kind := e.Kind
		if kind == "" {
			kind = "literal"
		}
		liveKeys[entryKey(e.Msgctxt, e.Msgid, kind)] = struct{}{}
	}

	committedKeys, err := loadPotKeys(potPath)
	if err != nil {
		if os.IsNotExist(err) {
			if len(liveKeys) == 0 {
				return nil, nil
			}
			return []Issue{{
				Module: moduleName,
				Kind:   IssuePotDirty,
				Detail: fmt.Sprintf("missing %s (run: choysum i18n extract %s)", filepath.Base(potPath), moduleName),
			}}, nil
		}
		return nil, err
	}

	var issues []Issue
	for key := range liveKeys {
		if _, ok := committedKeys[key]; !ok {
			scope, src := splitKey(key)
			issues = append(issues, Issue{
				Module: moduleName,
				Kind:   IssuePotDirty,
				Scope:  scope,
				Src:    src,
				Detail: "in source but not in committed pot",
			})
		}
	}
	for key := range committedKeys {
		if _, ok := liveKeys[key]; !ok {
			scope, src := splitKey(key)
			issues = append(issues, Issue{
				Module: moduleName,
				Kind:   IssuePotDirty,
				Scope:  scope,
				Src:    src,
				Detail: "in committed pot but not in source",
			})
		}
	}
	return issues, nil
}

func loadPotKeys(potPath string) (map[string]struct{}, error) {
	raw, err := os.ReadFile(potPath)
	if err != nil {
		return nil, err
	}
	entries, err := po.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", potPath, err)
	}
	keys := map[string]struct{}{}
	for _, e := range entries {
		if e.Msgid == "" && e.Msgctxt == "" {
			continue
		}
		if e.Obsolete {
			continue
		}
		keys[e.Key()] = struct{}{}
	}
	return keys, nil
}

func entryKey(scope, src, kind string) string {
	if strings.TrimSpace(kind) == "" {
		kind = "literal"
	}
	return scope + "\x00" + src + "\x00" + kind
}

func splitKey(key string) (scope, src string) {
	parts := strings.SplitN(key, "\x00", 3)
	if len(parts) < 2 {
		return "", key
	}
	return parts[0], parts[1]
}

func hasFlag(flags []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, f := range flags {
		if strings.ToLower(strings.TrimSpace(f)) == want {
			return true
		}
	}
	return false
}

// FormatText writes a human-readable summary.
func FormatText(report *Report) string {
	if report == nil {
		return "i18n status: clean\n"
	}
	if len(report.Issues) == 0 {
		return fmt.Sprintf("i18n status: clean (lang=%s)\n", report.Lang)
	}
	var b strings.Builder
	blocking := ExitCode(report) != 0
	orphanOnly := !blocking
	if orphanOnly {
		fmt.Fprintf(&b, "i18n status: clean (lang=%s; %d informational orphan(s))\n", report.Lang, len(report.Issues))
	} else {
		fmt.Fprintf(&b, "i18n status: %d issue(s) (lang=%s)\n", len(report.Issues), report.Lang)
	}
	counts := map[IssueKind]int{}
	for _, issue := range report.Issues {
		counts[issue.Kind]++
		loc := issue.Module
		if issue.Scope != "" || issue.Src != "" {
			loc = fmt.Sprintf("%s %s %q", issue.Module, issue.Scope, issue.Src)
		}
		detail := issue.Detail
		if detail != "" {
			detail = " — " + detail
		}
		fmt.Fprintf(&b, "  [%s] %s%s\n", issue.Kind, loc, detail)
	}
	fmt.Fprintf(&b, "summary: missing=%d fuzzy=%d orphan=%d pot-dirty=%d no-po=%d\n",
		counts[IssueMissing], counts[IssueFuzzy], counts[IssueOrphan], counts[IssuePotDirty], counts[IssueNoPo])
	return b.String()
}
