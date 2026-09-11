// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"os"
	"regexp"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

// Illegal marks that must not appear in QJS-only e2e specs.
var (
	illegalPlaywrightImportRE = regexp.MustCompile(
		`(?m)(?:from\s+|import\s*\(|require\s*\()\s*['"]@playwright/test['"]|import\s+['"]@playwright/test['"]`,
	)
	illegalNodeBuiltinRE = regexp.MustCompile(
		`(?m)(?:from\s+|import\s*\(|require\s*\()\s*['"]node:(?:fs|path|crypto)['"]|import\s+['"]node:(?:fs|path|crypto)['"]`,
	)
)

// IllegalE2EMark describes one forbidden import/require found in a spec file.
type IllegalE2EMark struct {
	Path string
	Kind string // "playwright" or "node-builtin"
	Line string
}

// ScanIllegalE2EMarks reports forbidden Playwright / Node builtin usages in specs.
func ScanIllegalE2EMarks(specFiles []string) ([]IllegalE2EMark, error) {
	var out []IllegalE2EMark
	for _, path := range specFiles {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, xfmt.Errorf("read %s: %w", path, err)
		}
		source := string(raw)
		for _, line := range strings.Split(source, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			switch {
			case illegalPlaywrightImportRE.MatchString(line):
				out = append(out, IllegalE2EMark{Path: path, Kind: "playwright", Line: trimmed})
			case illegalNodeBuiltinRE.MatchString(line):
				out = append(out, IllegalE2EMark{Path: path, Kind: "node-builtin", Line: trimmed})
			}
		}
	}
	return out, nil
}

// CheckIllegalE2EMarks fails when any scanned spec contains illegal marks.
func CheckIllegalE2EMarks(specFiles []string) error {
	marks, err := ScanIllegalE2EMarks(specFiles)
	if err != nil {
		return err
	}
	if len(marks) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("e2e: illegal marks in specs (QJS-only; no @playwright/test or node:fs|path|crypto):")
	for _, m := range marks {
		b.WriteString("\n  ")
		b.WriteString(m.Path)
		b.WriteString(" [")
		b.WriteString(m.Kind)
		b.WriteString("]: ")
		b.WriteString(m.Line)
	}
	return xfmt.Errorf("%s", b.String())
}
