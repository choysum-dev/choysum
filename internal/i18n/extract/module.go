// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package extract

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// ExtractModuleOptions configures ExtractModule.
type ExtractModuleOptions struct {
	PathAlias map[string]string
	WritePot  bool
	// Metadata enables S7 implicit extract (field/menu/selection). Default true when using ExtractModule.
	Metadata bool
}

// ExtractModule scans a module tree for `_t` / `_lt` literals (and metadata when enabled)
// and optionally writes pot. Metadata defaults to on (S7); pass ExtractModuleWithOptions to disable.
func ExtractModule(moduleRoot string, moduleName string, pathAlias map[string]string, writePot bool) (*ModuleResult, error) {
	return ExtractModuleWithOptions(moduleRoot, moduleName, ExtractModuleOptions{
		PathAlias: pathAlias,
		WritePot:  writePot,
		Metadata:  true,
	})
}

// ExtractModuleWithOptions is the configurable extract entrypoint.
func ExtractModuleWithOptions(moduleRoot string, moduleName string, opts ExtractModuleOptions) (*ModuleResult, error) {
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
			PathAlias:  opts.PathAlias,
		}
		var fileTerms []TermOccurrence
		var fileIssues []ExtractIssue
		switch ext {
		case ".vue":
			fileTerms, fileIssues = CollectVue(collectOpts, string(content))
			if opts.Metadata {
				mt, mi := CollectVueMetadata(collectOpts, string(content))
				fileTerms = append(fileTerms, mt...)
				fileIssues = append(fileIssues, mi...)
			}
		default:
			fileTerms, fileIssues = CollectScript(collectOpts, string(content))
			if opts.Metadata {
				mt, mi := CollectScriptMetadata(collectOpts, string(content))
				fileTerms = append(fileTerms, mt...)
				fileIssues = append(fileIssues, mi...)
			}
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

	if opts.WritePot {
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

// CollectScriptMetadata runs S7 AST collectors on a TS/TSX (or Vue script) file.
func CollectScriptMetadata(opts CollectOptions, content string) ([]TermOccurrence, []ExtractIssue) {
	var terms []TermOccurrence
	var issues []ExtractIssue
	t, i := CollectMetadataSelection(opts, content)
	terms = append(terms, t...)
	issues = append(issues, i...)
	t, i = CollectMetadataResources(opts, content)
	terms = append(terms, t...)
	issues = append(issues, i...)
	return terms, issues
}

// CollectVueMetadata runs S7 collectors on a Vue SFC (template labels + script selection/resources).
func CollectVueMetadata(opts CollectOptions, content string) ([]TermOccurrence, []ExtractIssue) {
	scriptContents, templateHTML, err := splitVueSFC(content)
	if err != nil {
		// Template-only SFCs (or parser requiring <script>) still need label extract.
		if tmpl, ok := extractTemplateBodyFallback(content); ok {
			return CollectMetadataVueTemplate(opts, tmpl)
		}
		return nil, []ExtractIssue{{
			Severity:   IssueSeverityWarn,
			Code:       "vue_parse_error",
			Message:    err.Error(),
			SourcePath: opts.RelPath,
			Line:       1,
			Col:        1,
		}}
	}
	var terms []TermOccurrence
	var issues []ExtractIssue
	for _, scriptContent := range scriptContents {
		if strings.TrimSpace(scriptContent) == "" {
			continue
		}
		t, i := CollectScriptMetadata(opts, scriptContent)
		terms = append(terms, t...)
		issues = append(issues, i...)
	}
	if templateHTML != "" {
		t, i := CollectMetadataVueTemplate(opts, templateHTML)
		terms = append(terms, t...)
		issues = append(issues, i...)
	}
	return terms, issues
}

func extractTemplateBodyFallback(content string) (string, bool) {
	re := regexp.MustCompile(`(?is)<template\b[^>]*>([\s\S]*?)</template>`)
	m := re.FindStringSubmatch(content)
	if len(m) < 2 {
		return "", false
	}
	return m[1], true
}

// MetadataEnabledFromEnv returns false when CHOYSUM_I18N_EXTRACT_METADATA=0/false/off.
func MetadataEnabledFromEnv() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("CHOYSUM_I18N_EXTRACT_METADATA")))
	switch v {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
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
