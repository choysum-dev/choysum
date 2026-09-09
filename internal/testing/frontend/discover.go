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
	osStat          = os.Stat
	filepathAbs     = filepath.Abs
	filepathWalkDir = filepath.WalkDir
)

// ScanMode controls whether illegal FE marks fail the scan.
type ScanMode int

const (
	// ScanModeWarn reports legacy Node/VTU inventory hits without failing (migration).
	ScanModeWarn ScanMode = iota
	// ScanModeError treats configured illegal patterns as a hard failure (FE hard-cut).
	ScanModeError
)

// IllegalKind identifies a banned or inventory FE unit-test pattern.
// After FE hard-cut, ScanModeError rejects vitest/Node DOM packages; choysumMount + .vue imports are allowed.
type IllegalKind string

const (
	IllegalDOMEnvironment IllegalKind = "dom-environment"
	IllegalVTU            IllegalKind = "vue-test-utils"
	IllegalVueImport      IllegalKind = "vue-sfc-import"
	IllegalDOMPackage     IllegalKind = "dom-package"
	// IllegalCoverageProbe is a fake lcov sampling SFC / import (banned; mount real business pages).
	IllegalCoverageProbe IllegalKind = "coverage-probe"
	// IllegalVitestImport is a bare vitest import or vi.mock (banned after FE hard-cut).
	IllegalVitestImport IllegalKind = "vitest-import"
)

// IllegalMark is one FE unit-test scan hit (see IllegalKind).
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
	reDOMPackage   = regexp.MustCompile("(?m)(?:\\bfrom\\s+|import\\s*(?:\\(\\s*)?|require\\s*\\(\\s*)['\"`](happy-dom|jsdom)(?:/[^'\"`]*)?['\"`]")
	reVTUImport    = regexp.MustCompile("(?m)(?:\\bfrom\\s+|import\\s*(?:\\(\\s*)?|require\\s*\\(\\s*)['\"`]@vue/test-utils(?:/[^'\"`]*)?['\"`]")
	reVitestImport = regexp.MustCompile("(?m)(?:\\bfrom\\s+|import\\s*(?:\\(\\s*)?|require\\s*\\(\\s*)['\"`]vitest(?:/[^'\"`]*)?['\"`]")
	reViMock       = regexp.MustCompile(`(?m)\bvi\.mock\s*\(`)
	// Suppress IllegalVTU for mount/shallowMount only when those names are imported from choysumMount.
	reChoysumMountBinding = regexp.MustCompile(`(?m)import\s*\{[^}]*\b(?:mount|shallowMount)\b[^}]*\}\s*from\s*['"\x60]@choysum/test-utils(?:/[^'"\x60]*)?['"\x60]`)
	reMountCall           = regexp.MustCompile(`(?:^|[^\.\w])(?:shallowMount|mount)\s*\(`)
	// Matches from '...vue', side-effect/dynamic/require imports, optional Vite query (?raw), and backticks.
	reVueImport = regexp.MustCompile("(?m)(?:\\bfrom\\s+|import\\s*(?:\\(\\s*)?|require\\s*\\(\\s*)['\"`][^'\"`]+\\.vue(?:\\?[^'\"`]*)?['\"`]")
	// CoverageProbe sampling SFC imports (banned). Basename must be exactly CoverageProbe.vue.
	reCoverageProbeImport = regexp.MustCompile("(?m)(?:\\bfrom\\s+|import\\s*(?:\\(\\s*)?|require\\s*\\(\\s*)['\"`](?:[^'\"`]*[/\\\\])?CoverageProbe\\.vue(?:\\?[^'\"`]*)?['\"`]")
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
	if strings.HasSuffix(lower, ".d.ts") {
		return false
	}
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
	hits, err := ScanIllegalFrontendMarks(files)
	if err != nil {
		return nil, err
	}
	probeHits, err := scanCoverageProbeFiles(repoRoot, app)
	if err != nil {
		return nil, err
	}
	hits = append(hits, probeHits...)
	return hits, nil
}

// scanCoverageProbeFiles flags modules/<app>/web/**/CoverageProbe.vue on disk.
func scanCoverageProbeFiles(repoRoot, app string) ([]IllegalMark, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	app = strings.TrimSpace(app)
	webRoot := filepath.Join(repoRoot, "modules", app, "web")
	st, err := osStat(webRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, xfmt.Errorf("frontend discover: coverage-probe walk: %w", err)
	}
	if !st.IsDir() {
		return nil, nil
	}
	var hits []IllegalMark
	err = filepathWalkDir(webRoot, func(path string, d fs.DirEntry, walkErr error) error {
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
		if d.Name() != "CoverageProbe.vue" {
			return nil
		}
		abs, absErr := filepathAbs(path)
		if absErr != nil {
			abs = path
		}
		hits = append(hits, IllegalMark{
			Path:    abs,
			Line:    1,
			Kind:    IllegalCoverageProbe,
			Snippet: "CoverageProbe.vue",
		})
		return nil
	})
	if err != nil {
		return nil, xfmt.Errorf("frontend discover: coverage-probe walk: %w", err)
	}
	return hits, nil
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
			// choysumMount: suppress only when mount/shallowMount are imported from @choysum/test-utils.
			// Binding regex uses [^}]* so multiline named imports are recognized against full content.
			if !reChoysumMountBinding.MatchString(content) {
				add(lineNo, IllegalVTU, line)
			}
		}
	}

	// vi.mock: scan code with comments/strings blanked so docs/fixtures do not false-fail --fail.
	code := blankJSCommentsAndStrings(content)
	codeLines := strings.Split(code, "\n")
	for i, line := range codeLines {
		if reViMock.MatchString(line) {
			snippet := line
			if i < len(lines) {
				snippet = lines[i]
			}
			add(i+1, IllegalVitestImport, snippet)
		}
	}

	// Match-span based reporting for import forms that may span lines.
	addRegexHits(content, lines, reVueImport, IllegalVueImport, add)
	addRegexHits(content, lines, reVTUImport, IllegalVTU, add)
	addRegexHits(content, lines, reVitestImport, IllegalVitestImport, add)
	addRegexHits(content, lines, reDOMPackage, IllegalDOMPackage, add)
	addRegexHits(content, lines, reCoverageProbeImport, IllegalCoverageProbe, add)
	return hits
}

// blankJSCommentsAndStrings replaces // and /* */ comments plus ', ", and ` string
// literals with spaces (newlines preserved) so inventory regexes can ignore noise.
func blankJSCommentsAndStrings(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				b.WriteByte(' ')
				i++
			}
			continue
		}
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			b.WriteByte(' ')
			b.WriteByte(' ')
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				if s[i] == '\n' {
					b.WriteByte('\n')
				} else {
					b.WriteByte(' ')
				}
				i++
			}
			if i+1 < len(s) {
				b.WriteByte(' ')
				b.WriteByte(' ')
				i += 2
			}
			continue
		}
		if s[i] == '\'' || s[i] == '"' || s[i] == '`' {
			quote := s[i]
			b.WriteByte(' ')
			i++
			for i < len(s) {
				if s[i] == '\\' && i+1 < len(s) {
					b.WriteByte(' ')
					b.WriteByte(' ')
					i += 2
					continue
				}
				ch := s[i]
				if ch == quote {
					b.WriteByte(' ')
					i++
					break
				}
				if ch == '\n' {
					b.WriteByte('\n')
				} else {
					b.WriteByte(' ')
				}
				i++
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
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
	b.WriteString(fmt.Sprintf("choysum test: %d illegal FE unit mark(s):\n", len(hits)))
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

// IsHardCutFailKind reports whether a mark fails ScanModeError / unit-fe-illegal --fail.
// IllegalVueImport stays inventory-only (warn): .vue imports are allowed at FE hard-cut.
// IllegalCoverageProbe always fails hard-cut (banned sampling SFCs).
func IsHardCutFailKind(kind IllegalKind) bool {
	return kind != IllegalVueImport
}

// FilterHardCutFailHits keeps marks that should fail hard-cut mode.
func FilterHardCutFailHits(hits []IllegalMark) []IllegalMark {
	out := make([]IllegalMark, 0, len(hits))
	for _, h := range hits {
		if IsHardCutFailKind(h.Kind) {
			out = append(out, h)
		}
	}
	return out
}

// CheckIllegalFrontendMarks scans and, in error mode, returns an error when hard-cut hits exist.
// All hits (including IllegalVueImport inventory) are still returned for warn formatting.
func CheckIllegalFrontendMarks(repoRoot, app string, mode ScanMode) ([]IllegalMark, error) {
	hits, err := ScanAppIllegalFrontendMarks(repoRoot, app)
	if err != nil {
		return nil, err
	}
	if mode == ScanModeError {
		failHits := FilterHardCutFailHits(hits)
		if len(failHits) > 0 {
			return hits, xfmt.Errorf("frontend illegal scan: %d illegal mark(s)\n%s", len(failHits), FormatIllegalMarksWarn(failHits, repoRoot))
		}
	}
	return hits, nil
}
