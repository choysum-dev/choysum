// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/evanw/esbuild/pkg/api"
)

// ModuleResult is the extract output for one module.
type ModuleResult struct {
	ModuleName string
	Terms      []TermOccurrence
	Issues     []ExtractIssue
	Entries    []PotEntry
	PotPath    string
}

// ExtractModule scans a module tree for explicit `_t` / `_lt` / `_tr` literals
// and optionally writes pot.
func ExtractModule(moduleRoot string, moduleName string, pathAlias map[string]string, writePot bool) (*ModuleResult, error) {
	moduleRoot = filepath.Clean(moduleRoot)
	if moduleName == "" {
		moduleName = filepath.Base(moduleRoot)
	}

	var terms []TermOccurrence
	var issues []ExtractIssue

	err := filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.IsDir() {
			switch name {
			case "node_modules", "dist", ".git", "i18n":
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".ts", ".tsx", ".vue":
		default:
			return nil
		}
		if strings.HasSuffix(strings.ToLower(name), ".d.ts") {
			return nil
		}

		rel, err := filepath.Rel(moduleRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		collectOpts := CollectOptions{
			ModuleName: moduleName,
			RelPath:    rel,
			PathAlias:  pathAlias,
		}
		var fileTerms []TermOccurrence
		var fileIssues []ExtractIssue
		switch ext {
		case ".vue":
			fileTerms, fileIssues = CollectVue(collectOpts, string(content))
		default:
			fileTerms, fileIssues = CollectScript(collectOpts, string(content))
		}
		terms = append(terms, fileTerms...)
		issues = append(issues, fileIssues...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	entries := DedupeOccurrences(terms)
	result := &ModuleResult{
		ModuleName: moduleName,
		Terms:      terms,
		Issues:     issues,
		Entries:    entries,
		PotPath:    filepath.Join(moduleRoot, "i18n", moduleName+".pot"),
	}

	if writePot {
		if err := os.MkdirAll(filepath.Dir(result.PotPath), 0o755); err != nil {
			return result, fmt.Errorf("create i18n dir: %w", err)
		}
		f, err := os.Create(result.PotPath)
		if err != nil {
			return result, fmt.Errorf("create pot: %w", err)
		}
		defer f.Close()
		if err := WritePot(f, moduleName, entries); err != nil {
			return result, err
		}
	}
	return result, nil
}

// LoadPathAliasFromModulesTsconfig loads path aliases from modules/tsconfig.json when present.
func LoadPathAliasFromModulesTsconfig(modulesPath string) (map[string]string, error) {
	tsconfigPath := filepath.Join(modulesPath, "tsconfig.json")
	if _, err := os.Stat(tsconfigPath); err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	return parser.ParseTsconfigPathAlias(&api.BuildOptions{Tsconfig: tsconfigPath})
}
