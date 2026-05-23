// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package coverage

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

// ReadGoModulePath reads the module path from go.mod under repoRoot.
func ReadGoModulePath(repoRoot string) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		repoRoot = FindRepoRootFromCwd()
	}
	file, err := os.Open(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return "", xfmt.Errorf("open go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if modulePath == "" {
				return "", xfmt.Errorf("empty module path in go.mod")
			}
			return modulePath, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", xfmt.Errorf("scan go.mod: %w", err)
	}
	return "", xfmt.Errorf("module directive not found in go.mod")
}

// FilterHandwrittenGoCoverProfile removes generated Go files from a coverprofile.
// It excludes *.pb.go and any Go source file whose header contains common generated markers.
func FilterHandwrittenGoCoverProfile(repoRoot string, rawProfile []byte) ([]byte, []string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		repoRoot = FindRepoRootFromCwd()
	}
	modulePath, err := ReadGoModulePath(repoRoot)
	if err != nil {
		return nil, nil, err
	}

	normalized := bytes.ReplaceAll(rawProfile, []byte("\r\n"), []byte("\n"))
	lines := strings.Split(strings.TrimRight(string(normalized), "\n"), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "mode: ") {
		return nil, nil, xfmt.Errorf("invalid coverprofile: missing mode header")
	}

	filtered := make([]string, 0, len(lines))
	filtered = append(filtered, lines[0])
	excludedSet := map[string]struct{}{}
	generatedCache := map[string]bool{}

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		filePart, _, ok := strings.Cut(line, ":")
		if !ok {
			return nil, nil, xfmt.Errorf("invalid coverprofile line: %s", line)
		}
		relPath := normalizeCoverageFilePath(filePart, modulePath)
		generated, ok := generatedCache[relPath]
		if !ok {
			generated = isGeneratedGoCoverageFile(repoRoot, relPath)
			generatedCache[relPath] = generated
		}
		if generated {
			excludedSet[relPath] = struct{}{}
			continue
		}
		filtered = append(filtered, line)
	}

	excluded := make([]string, 0, len(excludedSet))
	for relPath := range excludedSet {
		excluded = append(excluded, relPath)
	}
	sort.Strings(excluded)

	return []byte(strings.Join(filtered, "\n") + "\n"), excluded, nil
}

func normalizeCoverageFilePath(filePath string, modulePath string) string {
	filePath = strings.TrimSpace(filePath)
	modulePrefix := strings.TrimSpace(modulePath)
	if modulePrefix != "" {
		modulePrefix += "/"
		filePath = strings.TrimPrefix(filePath, modulePrefix)
	}
	if filepath.IsAbs(filePath) {
		return filepath.ToSlash(filepath.Clean(filePath))
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(filePath)))
}

func isGeneratedGoCoverageFile(repoRoot string, relPath string) bool {
	if !strings.HasSuffix(relPath, ".go") {
		return false
	}
	if strings.HasSuffix(relPath, ".pb.go") {
		return true
	}
	if strings.Contains(relPath, "/vendor/") {
		return false
	}
	absPath := relPath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(repoRoot, filepath.FromSlash(relPath))
	}
	header, err := readFileHeader(absPath, 2048)
	if err != nil {
		return false
	}
	return strings.Contains(header, "Code generated") || strings.Contains(header, "DO NOT EDIT")
}

func readFileHeader(path string, maxBytes int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, maxBytes)
	n, err := file.Read(buf)
	if err != nil && err.Error() != "EOF" {
		if n == 0 {
			return "", err
		}
	}
	return string(buf[:n]), nil
}
