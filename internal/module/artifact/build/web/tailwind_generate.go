// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	tw "github.com/dhamidi/tailwind-go"
)

// ChoyTailwindBudget is the soft wall-clock budget for a representative kit generate.
const ChoyTailwindBudget = 500 * time.Millisecond

const choyTailwindGeneratedCSSName = "choy-tailwind.generated.css"

// choyGalleryRootSelector scopes generated theme + utilities to the gallery root
// so product ERP pages are not affected when the gallery stylesheet is imported.
const choyGalleryRootSelector = ".choy-gallery-root"

// maxTailwindCandidateLen rejects scanner leaks from multi-line TS/JS literals.
// Long arbitrary-value utilities (urls, gradients, shadows) are legitimate; keep
// the cap generous. Whitespace/brace/semicolon checks already reject real leaks.
const maxTailwindCandidateLen = 512

// ChoyTailwindGenerateResult is the outcome of a Go Tailwind generate pass.
type ChoyTailwindGenerateResult struct {
	CSS            string
	Duration       time.Duration
	CandidateCount int
	DialectHash    string
	ContentHash    string
	OutputPath     string
}

// ScanTailwindCandidates walks roots for Vue/TS/JS/CSS/HTML text and returns unique class-like tokens.
func ScanTailwindCandidates(roots []string) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			if !shouldScanTailwindPath(root) {
				continue
			}
			if err := scanTailwindFile(root, seen, &out); err != nil {
				return nil, err
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				name := d.Name()
				if name == "node_modules" || name == "dist" || name == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if !shouldScanTailwindPath(path) {
				return nil
			}
			return scanTailwindFile(path, seen, &out)
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

// ScanChoyKitTailwindCandidates walks a module web/ tree but only files that belong
// to the Choy kit surface. Dual-stack product web also ships O*/Element Plus trees;
// scanning those would inflate candidates and break the soft generate budget.
func ScanChoyKitTailwindCandidates(webRoot string) ([]string, error) {
	webRoot = strings.TrimSpace(webRoot)
	if webRoot == "" {
		return nil, nil
	}
	seen := map[string]struct{}{}
	var out []string
	info, err := os.Stat(webRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}
	err = filepath.WalkDir(webRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == "dist" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !shouldScanTailwindPath(path) || !isChoyKitTailwindInputPath(webRoot, path) {
			return nil
		}
		return scanTailwindFile(path, seen, &out)
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// isChoyKitTailwindInputPath reports whether path under webRoot is Choy kit input
// (vendor/ui, internal engines, Choy* SFCs/helpers, gallery/dogfood pages, tokens CSS).
func isChoyKitTailwindInputPath(webRoot, path string) bool {
	rel, err := filepath.Rel(webRoot, path)
	if err != nil {
		return false
	}
	slash := filepath.ToSlash(rel)
	base := filepath.Base(path)

	switch {
	case strings.HasPrefix(slash, "styles/"):
		// Product EP SCSS lives beside Choy tokens; only scan Choy CSS dialect siblings.
		lower := strings.ToLower(base)
		return strings.HasSuffix(lower, ".css") &&
			!strings.HasSuffix(lower, ".generated.css") &&
			lower != choyTailwindGeneratedCSSName
	case strings.HasPrefix(slash, "components/vendor/"),
		strings.HasPrefix(slash, "components/internal/"),
		strings.HasPrefix(slash, "lib/"):
		return true
	case strings.HasPrefix(slash, "composables/"):
		return strings.Contains(base, "Choy") || strings.Contains(base, "choy")
	case strings.HasPrefix(slash, "pages/"):
		return base == "Gallery.vue" ||
			strings.HasPrefix(base, "Dogfood") ||
			strings.HasPrefix(base, "partnerDetail") ||
			strings.HasPrefix(base, "Choy")
	case strings.HasPrefix(slash, "components/layout/"),
		strings.HasPrefix(slash, "components/view/"),
		strings.HasPrefix(slash, "components/field/"),
		strings.HasPrefix(slash, "components/chatter/"):
		if strings.HasPrefix(base, "O") {
			return false
		}
		return strings.HasPrefix(base, "Choy") ||
			strings.Contains(base, "Helpers") ||
			strings.Contains(base, "Types") ||
			strings.Contains(base, "Adapter") ||
			strings.HasPrefix(base, "merge") ||
			strings.HasPrefix(base, "chart") ||
			strings.HasPrefix(base, "html") ||
			strings.HasPrefix(base, "json") ||
			strings.HasPrefix(base, "properties") ||
			strings.HasPrefix(base, "chatter") ||
			strings.HasPrefix(base, "kanban") ||
			strings.HasPrefix(base, "pagination") ||
			strings.HasPrefix(base, "search")
	default:
		// Unknown directories may still hold kit files; Choy* naming is the safety
		// net so new kit paths are not silently dropped from candidate scanning.
		return strings.HasPrefix(base, "Choy")
	}
}

func shouldScanTailwindPath(path string) bool {
	// Keep JS/TS module variants aligned with hashWebSourceTreeOpts so classes
	// authored only in .mts/.cts/.mjs/.cjs still produce utilities.
	switch strings.ToLower(filepath.Ext(path)) {
	case ".vue", ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs", ".css", ".html":
	default:
		return false
	}
	base := filepath.Base(path)
	if base == choyTailwindGeneratedCSSName || strings.HasSuffix(base, ".generated.css") {
		return false
	}
	// theme.css is LoadCSS dialect input, not class candidate source.
	if base == "theme.css" {
		return false
	}
	// Test/spec sources never render; class strings there only bloat utilities.
	if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") ||
		strings.Contains(filepath.ToSlash(path), "__tests__/") {
		return false
	}
	return true
}

func scanTailwindFile(path string, seen map[string]struct{}, out *[]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		// A file that vanished between WalkDir and ReadFile is not a build
		// input; mirror hashFile's NotExist tolerance instead of failing.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	eng := tw.New()
	// tw.New() engines have no passthrough writer; Write never returns an error.
	_, _ = eng.Write(data)
	eng.Flush()
	for _, c := range eng.Candidates() {
		c = strings.TrimSpace(c)
		if !isPlausibleTailwindCandidate(c) {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		*out = append(*out, c)
	}
	return nil
}

// isPlausibleTailwindCandidate drops scanner leaks from multi-line TS/JS arrays
// and object literals (which produce invalid CSS when treated as class names).
func isPlausibleTailwindCandidate(c string) bool {
	if c == "" || len(c) > maxTailwindCandidateLen {
		return false
	}
	if strings.ContainsAny(c, "\n\r\t") || strings.Contains(c, " ") {
		return false
	}
	if strings.ContainsAny(c, "{};") {
		return false
	}
	for _, r := range c {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// GenerateTailwindCSS loads dialect CSS, scans candidates, and returns theme +
// utility CSS scoped to .choy-gallery-root (no Tailwind FullCSS preflight).
func GenerateTailwindCSS(dialectCSS string, candidates []string) (css string, dur time.Duration, err error) {
	start := time.Now()
	eng := tw.New()
	if strings.TrimSpace(dialectCSS) != "" {
		if err := choyLoadCSS(eng, []byte(dialectCSS)); err != nil {
			return "", time.Since(start), fmt.Errorf("load Tailwind dialect: %w", err)
		}
	}
	if len(candidates) > 0 {
		payload := []byte(strings.Join(candidates, " "))
		// tw.New() engines have no passthrough writer; Write never returns an error.
		_, _ = eng.Write(payload)
	}
	eng.Flush()
	// ThemeCSS emits --color-* aliases from @theme; CSS() is utilities only.
	// Scope both under the gallery root so imports do not restyle product pages.
	theme := scopeChoyThemeCSS(eng.ThemeCSS())
	utilities := scopeChoyUtilityCSS(eng.CSS(), choyGalleryRootSelector)
	css = theme + utilities
	return css, time.Since(start), nil
}

// choyLoadCSS wraps Engine.LoadCSS so tests can force parse failures.
var choyLoadCSS = func(eng *tw.Engine, css []byte) error {
	return eng.LoadCSS(css)
}

// generateTailwindCSS is GenerateTailwindCSS; tests replace it to force empty output.
var generateTailwindCSS = GenerateTailwindCSS

// ensureChoyTailwindCSS is EnsureChoyTailwindCSS; tests replace it to force budget-warn paths.
var ensureChoyTailwindCSS = EnsureChoyTailwindCSS

// scopeChoyThemeCSS rebinds ThemeCSS :root/:host tokens onto the gallery root.
func scopeChoyThemeCSS(theme string) string {
	theme = strings.NewReplacer(
		":root, :host", choyGalleryRootSelector,
		":host, :root", choyGalleryRootSelector,
		":root,:host", choyGalleryRootSelector,
		":host,:root", choyGalleryRootSelector,
	).Replace(theme)
	return replaceChoyThemeSelectors(theme, choyGalleryRootSelector)
}

// replaceChoyThemeSelectors rebinds standalone :root / :host selectors to
// scope and leaves functional forms such as :host-context(...) or :host(.x)
// untouched.
func replaceChoyThemeSelectors(theme, scope string) string {
	var out strings.Builder
	for i := 0; i < len(theme); {
		switch {
		case strings.HasPrefix(theme[i:], ":root") && choyThemeSelectorBoundary(theme, i+len(":root")):
			out.WriteString(scope)
			i += len(":root")
		case strings.HasPrefix(theme[i:], ":host") && choyThemeSelectorBoundary(theme, i+len(":host")):
			out.WriteString(scope)
			i += len(":host")
		default:
			out.WriteByte(theme[i])
			i++
		}
	}
	return out.String()
}

// choyThemeSelectorBoundary reports whether a :root / :host token ends at i.
func choyThemeSelectorBoundary(s string, i int) bool {
	if i >= len(s) {
		return true
	}
	c := s[i]
	// Only standalone :root / :host should be rebound; reject identifier
	// continuations (e.g. :rooted, :hostname) and functional forms such as
	// :host(...) / :host-context(...).
	if c == '-' || c == '(' {
		return false
	}
	if c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
		return false
	}
	return true
}

// scopeChoyUtilityCSS prefixes top-level class selectors as descendants of scope.
// Nested selector blocks inside at-rules are scoped by default; declaration-body
// at-rules (@property/@keyframes/@font-face/…) keep their blocks verbatim.
func scopeChoyUtilityCSS(css, scope string) string {
	if strings.TrimSpace(css) == "" || strings.TrimSpace(scope) == "" {
		return css
	}
	var out strings.Builder
	rest := css
	for len(rest) > 0 {
		trimmed := strings.TrimLeft(rest, " \t\r\n")
		leading := rest[:len(rest)-len(trimmed)]
		out.WriteString(leading)
		rest = trimmed
		if rest == "" {
			break
		}
		if strings.HasPrefix(rest, "@") {
			// Block-less at-rules (@import/@charset/@layer a, b;) end at ';'.
			// Without this, indexCSSBlockEnd jumps to a later rule's '}' and
			// that rule is copied through unscoped.
			if semi := indexCSSBareAtRuleSemi(rest); semi >= 0 {
				out.WriteString(rest[:semi+1])
				rest = rest[semi+1:]
				continue
			}
			end := indexCSSBlockEnd(rest)
			if end < 0 {
				out.WriteString(rest)
				break
			}
			name := strings.TrimPrefix(rest, "@")
			if idx := strings.IndexAny(name, " \t\r\n({;"); idx >= 0 {
				name = name[:idx]
			}
			switch strings.ToLower(name) {
			// Declaration-body at-rules keep their blocks verbatim.
			case "property", "keyframes", "-webkit-keyframes", "font-face", "counter-style", "font-feature-values", "page", "viewport":
			default:
				if brace := indexCSSOpenBrace(rest, end); brace >= 0 {
					out.WriteString(rest[:brace+1])
					out.WriteString(scopeChoyUtilityCSS(rest[brace+1:end-1], scope))
					out.WriteString("}")
					rest = rest[end:]
					continue
				}
			}
			out.WriteString(rest[:end])
			rest = rest[end:]
			continue
		}
		// A leading comment may contain '{'; emit it verbatim so it is not
		// mistaken for the selector boundary and prefixed with the scope.
		if strings.HasPrefix(rest, "/*") {
			end := strings.Index(rest, "*/")
			if end < 0 {
				out.WriteString(rest)
				break
			}
			out.WriteString(rest[:end+2])
			rest = rest[end+2:]
			continue
		}
		brace := strings.IndexByte(rest, '{')
		if brace < 0 {
			out.WriteString(rest)
			break
		}
		selectors := rest[:brace]
		block := rest[brace:]
		end := indexCSSBlockEnd(block)
		if end < 0 {
			out.WriteString(rest)
			break
		}
		out.WriteString(prefixCSSSelectorList(selectors, scope))
		out.WriteString(block[:end])
		rest = block[end:]
	}
	return out.String()
}

func prefixCSSSelectorList(selectors, scope string) string {
	parts := splitTopLevelSelectors(selectors)
	scoped := make([]string, 0, len(parts))
	for _, part := range parts {
		trim := strings.TrimSpace(part)
		if trim == "" {
			continue
		}
		// Preserve leading whitespace/newlines around each selector.
		lead := part[:len(part)-len(strings.TrimLeft(part, " \t\r\n"))]
		// `:root` / `:host` can never match as a descendant of the scope; rebind
		// them to the gallery root instead of emitting a never-matching selector.
		if trim == ":root" || trim == ":host" {
			scoped = append(scoped, lead+scope)
			continue
		}
		scoped = append(scoped, lead+scope+" "+trim)
	}
	return strings.Join(scoped, ",")
}

// splitTopLevelSelectors splits a selector list on commas that are not nested
// inside (), [] , quoted strings, or /* */ comments, so functional pseudo-classes
// such as :where(.dark, .dark *) stay intact.
func splitTopLevelSelectors(selectors string) []string {
	var parts []string
	depth := 0
	var quote byte
	start := 0
	for i := 0; i < len(selectors); i++ {
		c := selectors[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '/' && i+1 < len(selectors) && selectors[i+1] == '*':
			end := strings.Index(selectors[i+2:], "*/")
			if end < 0 {
				return append(parts, selectors[start:])
			}
			i += end + 3
		case c == '\'' || c == '"':
			quote = c
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			if depth > 0 {
				depth--
			}
		case c == ',' && depth == 0:
			parts = append(parts, selectors[start:i])
			start = i + 1
		}
	}
	return append(parts, selectors[start:])
}

// indexCSSBareAtRuleSemi returns the index of the terminating ';' for a
// block-less at-rule, ignoring ';' inside quotes, /* */ comments, or nested ().
// Returns -1 when a top-level '{' appears first (block at-rule) or no terminator is found.
func indexCSSBareAtRuleSemi(rest string) int {
	depth := 0
	var quote byte
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '/' && i+1 < len(rest) && rest[i+1] == '*':
			end := strings.Index(rest[i+2:], "*/")
			if end < 0 {
				return -1
			}
			i += end + 3
		case c == '(':
			depth++
		case c == ')':
			if depth > 0 {
				depth--
			}
		case c == ';' && depth == 0:
			return i
		case c == '{' && depth == 0:
			return -1
		}
	}
	return -1
}

// indexCSSOpenBrace returns the index of the first '{' in css[:limit] that is
// outside quotes and /* */ comments. Returns -1 when none is found.
func indexCSSOpenBrace(css string, limit int) int {
	if limit < 0 || limit > len(css) {
		limit = len(css)
	}
	var quote byte
	for i := 0; i < limit; i++ {
		c := css[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '/' && i+1 < limit && css[i+1] == '*':
			end := strings.Index(css[i+2:limit], "*/")
			if end < 0 {
				return -1
			}
			i += end + 3
		case c == '{':
			return i
		}
	}
	return -1
}

// indexCSSBlockEnd returns the index after the first top-level closing `}`,
// accounting for nested braces, quotes, and /* */ comments. Returns -1 when unbalanced.
func indexCSSBlockEnd(css string) int {
	depth := 0
	var quote byte
	for i := 0; i < len(css); i++ {
		c := css[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '/' && i+1 < len(css) && css[i+1] == '*':
			end := strings.Index(css[i+2:], "*/")
			if end < 0 {
				return -1
			}
			i += end + 3
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

// GenerateChoyTailwindForModule scans a Choy kit module root and writes generated utilities CSS.
// Emits theme aliases + utilities scoped to .choy-gallery-root (never FullCSS preflight).
func GenerateChoyTailwindForModule(moduleRoot string) (*ChoyTailwindGenerateResult, error) {
	moduleRoot = strings.TrimSpace(moduleRoot)
	if moduleRoot == "" {
		return nil, fmt.Errorf("choy kit module root is empty")
	}
	webRoot := filepath.Join(moduleRoot, "web")
	dialectPath := filepath.Join(webRoot, "styles", "theme.css")
	dialectBytes, err := os.ReadFile(dialectPath)
	if err != nil {
		return nil, fmt.Errorf("read dialect %s: %w", dialectPath, err)
	}
	dialectHash := sha256Hex(dialectBytes)

	candidates, err := ScanChoyKitTailwindCandidates(webRoot)
	if err != nil {
		return nil, err
	}
	contentHash := sha256Hex([]byte(strings.Join(candidates, "\n")))

	css, dur, err := generateTailwindCSS(string(dialectBytes), candidates)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(css) == "" {
		return nil, fmt.Errorf("generated Tailwind CSS for %s is empty; refusing to write an unstyled kit", moduleRoot)
	}

	// Header omits wall-clock duration so identical inputs rewrite a stable file.
	// Duration is returned on ChoyTailwindGenerateResult for gates. The output path
	// is gitignored; callers must run this before bundling Gallery CSS imports.
	header := fmt.Sprintf(
		"/* Generated by choysum web build (tailwind-go). Do not edit.\n"+
			" * dialect=%s content=%s engine=%s candidates=%d\n"+
			" * Isolation: theme+utilities scoped to %s (no Tailwind preflight).\n"+
			" */\n",
		dialectHash[:12], contentHash[:12], choyTailwindGoModuleVersion(), len(candidates), choyGalleryRootSelector,
	)
	outPath := filepath.Join(webRoot, "styles", choyTailwindGeneratedCSSName)
	if err := writeFileAtomicIfChanged(outPath, header+css); err != nil {
		return nil, err
	}
	return &ChoyTailwindGenerateResult{
		CSS:            css,
		Duration:       dur,
		CandidateCount: len(candidates),
		DialectHash:    dialectHash,
		ContentHash:    contentHash,
		OutputPath:     outPath,
	}, nil
}

func writeFileAtomicIfChanged(path, content string) error {
	if prev, err := os.ReadFile(path); err == nil && string(prev) == content {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".choy-tailwind-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			// Close before Remove so a failed Close cannot leave the fd open
			// (and block Remove on Windows).
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()
	if err := writeAtomicTemp(tmp, content); err != nil {
		return err
	}
	if err := atomicWriteClose(tmp); err != nil {
		return err
	}
	if err := atomicWriteRename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func writeAtomicTemp(tmp *os.File, content string) error {
	if err := atomicWriteWriteString(tmp, content); err != nil {
		return err
	}
	if err := atomicWriteChmod(tmp, 0o644); err != nil {
		return err
	}
	if err := atomicWriteSync(tmp); err != nil {
		return err
	}
	return nil
}

// Atomic write helpers are vars so tests can force OS-level failure paths.
var (
	atomicWriteWriteString = func(tmp *os.File, content string) error {
		_, err := tmp.WriteString(content)
		return err
	}
	atomicWriteChmod = func(tmp *os.File, mode os.FileMode) error {
		return tmp.Chmod(mode)
	}
	atomicWriteSync = func(tmp *os.File) error {
		return tmp.Sync()
	}
	atomicWriteClose = func(tmp *os.File) error {
		return tmp.Close()
	}
	atomicWriteRename = os.Rename
)

// EnsureChoyTailwindCSS finds the Choy kit under modulesPath and regenerates CSS.
// Prefers modules/web when styles/theme.css is present; falls back to modules/choy_ui.
// No-op when neither kit root owns a dialect file.
func EnsureChoyTailwindCSS(modulesPath string) (*ChoyTailwindGenerateResult, error) {
	modulesPath = strings.TrimSpace(modulesPath)
	if modulesPath == "" {
		return nil, nil
	}
	root, err := resolveChoyKitModuleRoot(modulesPath)
	if err != nil {
		return nil, err
	}
	if root == "" {
		return nil, nil
	}
	return GenerateChoyTailwindForModule(root)
}

// resolveChoyKitModuleRoot returns the module root that owns Choy styles/theme.css.
// Prefers web when it has a dialect file; falls back to choy_ui.
func resolveChoyKitModuleRoot(modulesPath string) (string, error) {
	for _, name := range []string{"web", "choy_ui"} {
		root := filepath.Join(modulesPath, name)
		webRoot := filepath.Join(root, "web")
		st, err := os.Stat(webRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if !st.IsDir() {
			continue
		}
		dialectPath := filepath.Join(webRoot, "styles", "theme.css")
		st, err = os.Stat(dialectPath)
		if err != nil {
			if os.IsNotExist(err) {
				// A module that already hosts the kit tree but lost its dialect is a
				// broken install, not a host that never carried the kit.
				if kitSt, kitErr := os.Stat(filepath.Join(webRoot, "components", "vendor", "ui")); kitErr == nil && kitSt.IsDir() {
					return "", fmt.Errorf("%s module present but dialect %s is missing", name, dialectPath)
				}
				continue
			}
			return "", err
		}
		if st.IsDir() {
			return "", fmt.Errorf("%s dialect %s is a directory, not a file", name, dialectPath)
		}
		return root, nil
	}
	return "", nil
}

// TailwindInputDigest returns stable dialect and candidate hashes for the Choy kit under modulesPath.
// Empty strings when the kit module is absent (no Tailwind inputs to invalidate).
func TailwindInputDigest(modulesPath string) (dialectHash, contentHash string, err error) {
	modulesPath = strings.TrimSpace(modulesPath)
	if modulesPath == "" {
		return "", "", nil
	}
	root, err := resolveChoyKitModuleRoot(modulesPath)
	if err != nil {
		return "", "", err
	}
	if root == "" {
		return "", "", nil
	}
	webRoot := filepath.Join(root, "web")
	dialectPath := filepath.Join(webRoot, "styles", "theme.css")
	dialectBytes, err := os.ReadFile(dialectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", nil
		}
		return "", "", err
	}
	candidates, err := ScanChoyKitTailwindCandidates(webRoot)
	if err != nil {
		return "", "", err
	}
	return sha256Hex(dialectBytes), sha256Hex([]byte(strings.Join(candidates, "\n"))), nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
