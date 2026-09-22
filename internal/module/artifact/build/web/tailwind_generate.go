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

// maxTailwindCandidateLen rejects scanner leaks from multi-line TS/JS literals.
const maxTailwindCandidateLen = 128

// ChoyTailwindGenerateResult is the outcome of a Go Tailwind generate pass.
type ChoyTailwindGenerateResult struct {
	CSS            string
	Duration       time.Duration
	CandidateCount int
	DialectHash    string
	ContentHash    string
	OutputPath     string
}

// ScanTailwindCandidates walks roots for .vue/.ts/.tsx/.css text and returns unique class-like tokens.
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

func shouldScanTailwindPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".vue", ".ts", ".tsx", ".js", ".jsx", ".css", ".html":
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
	return true
}

func scanTailwindFile(path string, seen map[string]struct{}, out *[]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
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

// GenerateTailwindCSS loads dialect CSS, scans candidates, and returns utility CSS (no preflight).
func GenerateTailwindCSS(dialectCSS string, candidates []string) (css string, dur time.Duration, err error) {
	start := time.Now()
	eng := tw.New()
	if strings.TrimSpace(dialectCSS) != "" {
		eng.LoadCSS([]byte(dialectCSS))
	}
	if len(candidates) > 0 {
		payload := []byte(strings.Join(candidates, " "))
		// tw.New() engines have no passthrough writer; Write never returns an error.
		_, _ = eng.Write(payload)
	}
	eng.Flush()
	css = eng.CSS()
	return css, time.Since(start), nil
}

// GenerateChoyTailwindForModule scans a choy_ui module root and writes generated utilities CSS.
// Uses eng.CSS() (not FullCSS) so product EP pages are not hit by Tailwind preflight.
func GenerateChoyTailwindForModule(moduleRoot string) (*ChoyTailwindGenerateResult, error) {
	moduleRoot = strings.TrimSpace(moduleRoot)
	if moduleRoot == "" {
		return nil, fmt.Errorf("choy_ui module root is empty")
	}
	webRoot := filepath.Join(moduleRoot, "web")
	dialectPath := filepath.Join(webRoot, "styles", "theme.css")
	dialectBytes, err := os.ReadFile(dialectPath)
	if err != nil {
		return nil, fmt.Errorf("read dialect %s: %w", dialectPath, err)
	}
	dialectHash := sha256Hex(dialectBytes)

	candidates, err := ScanTailwindCandidates([]string{webRoot})
	if err != nil {
		return nil, err
	}
	contentHash := sha256Hex([]byte(strings.Join(candidates, "\n")))

	// GenerateTailwindCSS only fails if the engine Write fails; tw.New() never does.
	css, dur, _ := GenerateTailwindCSS(string(dialectBytes), candidates)

	// Header omits wall-clock duration so identical inputs rewrite a stable file.
	// Duration is returned on ChoyTailwindGenerateResult for gates. The output path
	// is gitignored; callers must run this before bundling Gallery CSS imports.
	header := fmt.Sprintf(
		"/* Generated by choysum web build (tailwind-go). Do not edit.\n"+
			" * dialect=%s content=%s candidates=%d\n"+
			" * Isolation: utilities only (no Tailwind preflight).\n"+
			" */\n",
		dialectHash[:12], contentHash[:12], len(candidates),
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
			_ = os.Remove(tmpName)
		}
	}()
	if err := writeAtomicTemp(tmp, content); err != nil {
		_ = tmp.Close()
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

// EnsureChoyTailwindCSS finds an installed/local choy_ui module under modulesPath and regenerates CSS.
// No-op when the module is absent.
func EnsureChoyTailwindCSS(modulesPath string) (*ChoyTailwindGenerateResult, error) {
	modulesPath = strings.TrimSpace(modulesPath)
	if modulesPath == "" {
		return nil, nil
	}
	root := filepath.Join(modulesPath, "choy_ui")
	st, err := os.Stat(filepath.Join(root, "web", "styles", "theme.css"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if st.IsDir() {
		return nil, nil
	}
	return GenerateChoyTailwindForModule(root)
}

// TailwindInputDigest returns stable dialect and candidate hashes for choy_ui under modulesPath.
// Empty strings when the kit module is absent (no Tailwind inputs to invalidate).
func TailwindInputDigest(modulesPath string) (dialectHash, contentHash string, err error) {
	modulesPath = strings.TrimSpace(modulesPath)
	if modulesPath == "" {
		return "", "", nil
	}
	root := filepath.Join(modulesPath, "choy_ui")
	webRoot := filepath.Join(root, "web")
	dialectPath := filepath.Join(webRoot, "styles", "theme.css")
	dialectBytes, err := os.ReadFile(dialectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", nil
		}
		return "", "", err
	}
	candidates, err := ScanTailwindCandidates([]string{webRoot})
	if err != nil {
		return "", "", err
	}
	return sha256Hex(dialectBytes), sha256Hex([]byte(strings.Join(candidates, "\n"))), nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
