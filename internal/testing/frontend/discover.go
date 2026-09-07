// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	xfmt "golang.org/x/exp/errors/fmt"
)

// Test seams for rare OS failures (overridden in unit tests).
var (
	osStat      = os.Stat
	filepathAbs = filepath.Abs
)

// ScanMode controls whether illegal FE marks fail the scan.
type ScanMode int

const (
	// ScanModeWarn reports hits without failing (corpus migration).
	ScanModeWarn ScanMode = iota
	// ScanModeError treats any hit as a hard failure (FE hard-cut).
	ScanModeError
)

// IllegalKind identifies a banned FE unit-test pattern.
type IllegalKind string

const (
	IllegalDOMEnvironment IllegalKind = "dom-environment"
	IllegalVTU            IllegalKind = "vue-test-utils"
	IllegalVueImport      IllegalKind = "vue-sfc-import"
	IllegalDOMPackage     IllegalKind = "dom-package"
)

// IllegalMark is one illegal FE unit-test hit.
type IllegalMark struct {
	Path    string
	Line    int
	Kind    IllegalKind
	Snippet string
}

var (
	reVitestEnvHappy = regexp.MustCompile(`(?i)@(?:vitest|jest)-environment\s+(happy-dom|jsdom)\b`)
	// Match the from/import/require clause itself so line numbers stay on the package specifier
	// (avoids [\s\S]*? spanning back to an earlier unrelated import/export).
	// Quote class includes backticks for dynamic import()/require() template literals.
	reDOMPackage = regexp.MustCompile("(?m)(?:\\bfrom\\s+|import\\s*(?:\\(\\s*)?|require\\s*\\(\\s*)['\"`](happy-dom|jsdom)['\"`]")
	reVTUImport  = regexp.MustCompile("(?m)(?:\\bfrom\\s+|import\\s*(?:\\(\\s*)?|require\\s*\\(\\s*)['\"`]@vue/test-utils['\"`]")
	reMountCall  = regexp.MustCompile(`\b(?:shallowMount|mount)\s*\(`)
	// Matches from '...vue', side-effect/dynamic/require imports, optional Vite query (?raw), and backticks.
	reVueImport = regexp.MustCompile("(?m)(?:\\bfrom\\s+|import\\s*(?:\\(\\s*)?|require\\s*\\(\\s*)['\"`][^'\"`]+\\.vue(?:\\?[^'\"`]*)?['\"`]")
)

// DiscoverFrontendTests lists FE unit files under modules/<app>/web.
func DiscoverFrontendTests(repoRoot, app string) ([]string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	app = strings.TrimSpace(app)
	if repoRoot == "" {
		return nil, xfmt.Errorf("frontend discover: empty repo root")
	}
	if app == "" {
		return nil, xfmt.Errorf("frontend discover: empty app")
	}
	webRoot := filepath.Join(repoRoot, "modules", app, "web")
	st, err := osStat(webRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, xfmt.Errorf("frontend discover: stat %s: %w", webRoot, err)
	}
	if !st.IsDir() {
		return nil, nil
	}

	var out []string
	err = filepath.WalkDir(webRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "dist" || name == ".choysum" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isFrontendUnitTestFile(d.Name()) {
			return nil
		}
		abs, err := filepathAbs(path)
		if err != nil {
			return err
		}
		out = append(out, abs)
		return nil
	})
	if err != nil {
		return nil, xfmt.Errorf("frontend discover: walk %s: %w", webRoot, err)
	}
	sort.Strings(out)
	return out, nil
}

func isFrontendUnitTestFile(name string) bool {
	lower := strings.ToLower(name)
	if !strings.Contains(lower, ".test.") && !strings.Contains(lower, ".spec.") {
		return false
	}
	for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// ScanIllegalFrontendMarks scans FE unit files for banned patterns.
// Importing from 'vitest' is allowed during the migration period.
func ScanIllegalFrontendMarks(paths []string) ([]IllegalMark, error) {
	var hits []IllegalMark
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, xfmt.Errorf("frontend illegal scan: read %s: %w", path, err)
		}
		hits = append(hits, scanIllegalContent(path, string(raw))...)
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Path != hits[j].Path {
			return hits[i].Path < hits[j].Path
		}
		if hits[i].Line != hits[j].Line {
			return hits[i].Line < hits[j].Line
		}
		return hits[i].Kind < hits[j].Kind
	})
	return hits, nil
}

// ScanAppIllegalFrontendMarks discovers and scans one app's FE unit tests.
func ScanAppIllegalFrontendMarks(repoRoot, app string) ([]IllegalMark, error) {
	files, err := DiscoverFrontendTests(repoRoot, app)
	if err != nil {
		return nil, err
	}
	return ScanIllegalFrontendMarks(files)
}

func scanIllegalContent(path, content string) []IllegalMark {
	lines := strings.Split(content, "\n")
	var hits []IllegalMark
	seen := map[string]bool{}

	add := func(line int, kind IllegalKind, snippet string) {
		key := fmt.Sprintf("%s:%d:%s", path, line, kind)
		if seen[key] {
			return
		}
		seen[key] = true
		hits = append(hits, IllegalMark{
			Path:    path,
			Line:    line,
			Kind:    kind,
			Snippet: truncateSnippet(snippet),
		})
	}

	for i, line := range lines {
		lineNo := i + 1
		if reVitestEnvHappy.MatchString(line) {
			add(lineNo, IllegalDOMEnvironment, line)
		}
		if reMountCall.MatchString(line) {
			add(lineNo, IllegalVTU, line)
		}
	}

	// Match-span based reporting for import forms that may span lines.
	addRegexHits(content, lines, reVueImport, IllegalVueImport, add)
	addRegexHits(content, lines, reVTUImport, IllegalVTU, add)
	addRegexHits(content, lines, reDOMPackage, IllegalDOMPackage, add)
	return hits
}

func addRegexHits(content string, lines []string, re *regexp.Regexp, kind IllegalKind, add func(int, IllegalKind, string)) {
	matches := re.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return
	}
	lineNo := 1
	lastIdx := 0
	for _, loc := range matches {
		lineNo += strings.Count(content[lastIdx:loc[0]], "\n")
		lastIdx = loc[0]
		snippet := content[loc[0]:loc[1]]
		if lineNo >= 1 && lineNo <= len(lines) {
			snippet = lines[lineNo-1]
		}
		add(lineNo, kind, snippet)
	}
}

func truncateSnippet(snippet string) string {
	snippet = strings.TrimSpace(snippet)
	if utf8.RuneCountInString(snippet) <= 120 {
		return snippet
	}
	runes := []rune(snippet)
	return string(runes[:117]) + "..."
}

// FormatIllegalMarksWarn formats hits for stderr (human-readable).
func FormatIllegalMarksWarn(hits []IllegalMark, repoRoot string) string {
	if len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("choysum test: %d illegal FE unit mark(s) (warn; hard-cut will fail):\n", len(hits)))
	for _, hit := range hits {
		rel := relativizeRepoPath(repoRoot, hit.Path)
		fmt.Fprintf(&b, "  - %s:%d [%s] %s\n", rel, hit.Line, hit.Kind, hit.Snippet)
	}
	return b.String()
}

// FormatIllegalMarksGitHubAnnotations emits GitHub Actions warning annotations.
func FormatIllegalMarksGitHubAnnotations(hits []IllegalMark, repoRoot string) string {
	if len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	for _, hit := range hits {
		rel := relativizeRepoPath(repoRoot, hit.Path)
		rel = strings.ReplaceAll(rel, "\\", "/")
		rel = strings.ReplaceAll(rel, "%", "%25")
		rel = strings.ReplaceAll(rel, ",", "%2C")
		msg := fmt.Sprintf("illegal FE unit mark [%s]: %s", hit.Kind, hit.Snippet)
		msg = strings.ReplaceAll(msg, "\n", " ")
		msg = strings.ReplaceAll(msg, "%", "%25")
		msg = strings.ReplaceAll(msg, "\r", "")
		fmt.Fprintf(&b, "::warning file=%s,line=%d::%s\n", rel, hit.Line, msg)
	}
	return b.String()
}

func relativizeRepoPath(repoRoot, path string) string {
	path = filepath.Clean(path)
	repoRoot = filepath.Clean(strings.TrimSpace(repoRoot))
	if repoRoot == "" || repoRoot == "." {
		return filepath.ToSlash(path)
	}
	absRoot, err1 := filepathAbs(repoRoot)
	absPath, err2 := filepathAbs(path)
	if err1 == nil && err2 == nil {
		if rel, err := filepath.Rel(absRoot, absPath); err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
		return filepath.ToSlash(absPath)
	}
	if rel, err := filepath.Rel(repoRoot, path); err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

// CheckIllegalFrontendMarks scans and, in error mode, returns an error when hits exist.
func CheckIllegalFrontendMarks(repoRoot, app string, mode ScanMode) ([]IllegalMark, error) {
	hits, err := ScanAppIllegalFrontendMarks(repoRoot, app)
	if err != nil {
		return nil, err
	}
	if mode == ScanModeError && len(hits) > 0 {
		return hits, xfmt.Errorf("frontend illegal scan: %d illegal mark(s)\n%s", len(hits), FormatIllegalMarksWarn(hits, repoRoot))
	}
	return hits, nil
}
