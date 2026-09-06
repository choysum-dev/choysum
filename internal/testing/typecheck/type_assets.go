// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	moddeps "github.com/choysum-dev/choysum/internal/testing/moddeps"
	noderuntime "github.com/choysum-dev/choysum/internal/testing/noderuntime"
	gonative "github.com/choysum-dev/choysum/internal/typecheck"
	"github.com/tailscale/hujson"
	xfmt "golang.org/x/exp/errors/fmt"
)

func formatTypecheckFailureWithGuidance(app string, runErr error, output string, warnedMissingTypeAssets bool) error {
	baseErr := xfmt.Errorf("typecheck failed for %s: %w", app, runErr)
	if !warnedMissingTypeAssets && !shouldSuggestTypeFetchFromOutput(output) {
		return baseErr
	}

	app = strings.TrimSpace(app)
	if app == "" {
		app = "<app>"
	}
	return xfmt.Errorf(
		"%w\nrecommended action:\n  go run . type-fetch %s",
		baseErr,
		app,
	)
}

func shouldSuggestTypeFetchFromOutput(output string) bool {
	output = strings.ToLower(strings.TrimSpace(output))
	if output == "" {
		return false
	}
	if strings.Contains(output, "ts2307") || strings.Contains(output, "ts7016") {
		return true
	}
	if strings.Contains(output, "cannot find module") && strings.Contains(output, "type declaration") {
		return true
	}
	return false
}

func warnMissingTypeAssetsPrecheck(stderr io.Writer, modulesRoot string, app string) (bool, error) {
	if stderr == nil {
		return false, nil
	}

	externalModules, err := moddeps.CollectExternalModuleDependencies(modulesRoot, []string{app}, true)
	if err != nil {
		return false, xfmt.Errorf("typecheck: collect module dependencies: %w", err)
	}
	if len(externalModules) == 0 {
		return false, nil
	}

	missingModules := missingTypeAssetModules(modulesRoot, externalModules)
	if len(missingModules) == 0 {
		return false, nil
	}

	preview := strings.Join(missingModules, ", ")
	if len(missingModules) > 3 {
		preview = strings.Join(missingModules[:3], ", ") + ", ..."
	}

	_, _ = fmt.Fprintf(
		stderr,
		"Warning: type declarations may be incomplete for %s.\nmissing %d module(s) (sample: %s)\nrecommended action:\n  go run . type-fetch %s\n\n",
		app,
		len(missingModules),
		preview,
		app,
	)
	return true, nil
}

func missingTypeAssetModules(modulesRoot string, expectedModules []string) []string {
	pathsByModule := readModuleTSConfigPaths(modulesRoot)
	if len(pathsByModule) == 0 {
		return nil
	}

	missing := make([]string, 0)
	for _, moduleName := range noderuntime.NormalizeStringList(expectedModules) {
		pathEntries, hasPathMapping := resolveTypePathEntries(pathsByModule, moduleName)
		if !hasPathMapping {
			continue
		}
		if hasAnyExistingTypeAsset(pathEntries, modulesRoot) {
			continue
		}
		missing = append(missing, moduleName)
	}
	return missing
}

func resolveTypePathEntries(pathsByModule map[string][]string, moduleName string) ([]string, bool) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return nil, false
	}

	entries, ok := pathsByModule[moduleName]
	if ok {
		return entries, true
	}

	resolved := make([]string, 0, 4)
	hasMapping := false
	for key, valueEntries := range pathsByModule {
		replacements, matched := matchTSConfigPathPattern(strings.TrimSpace(key), moduleName)
		if !matched {
			continue
		}
		hasMapping = true
		for _, entry := range valueEntries {
			resolved = append(resolved, applyPathPatternReplacements(entry, replacements))
		}
	}

	return resolved, hasMapping
}

func matchTSConfigPathPattern(pattern string, moduleName string) ([]string, bool) {
	if pattern == "" {
		return nil, false
	}
	if !strings.Contains(pattern, "*") {
		return nil, pattern == moduleName
	}

	parts := strings.Split(pattern, "*")
	starMatches := make([]string, 0, len(parts)-1)
	remain := moduleName

	if !strings.HasPrefix(remain, parts[0]) {
		return nil, false
	}
	remain = strings.TrimPrefix(remain, parts[0])

	for i := 1; i < len(parts)-1; i++ {
		segment := parts[i]
		if segment == "" {
			starMatches = append(starMatches, "")
			continue
		}

		index := strings.Index(remain, segment)
		if index < 0 {
			return nil, false
		}
		starMatches = append(starMatches, remain[:index])
		remain = remain[index+len(segment):]
	}

	segment := parts[len(parts)-1]
	if segment == "" {
		starMatches = append(starMatches, remain)
		return starMatches, true
	}
	if !strings.HasSuffix(remain, segment) {
		return nil, false
	}
	starMatches = append(starMatches, remain[:len(remain)-len(segment)])
	return starMatches, true
}

func applyPathPatternReplacements(pathPattern string, replacements []string) string {
	if len(replacements) == 0 || !strings.Contains(pathPattern, "*") {
		return pathPattern
	}

	var b strings.Builder
	replacementIndex := 0
	for _, ch := range pathPattern {
		if ch != '*' {
			b.WriteRune(ch)
			continue
		}
		if replacementIndex < len(replacements) {
			b.WriteString(replacements[replacementIndex])
			replacementIndex++
			continue
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func readModuleTSConfigPaths(modulesRoot string) map[string][]string {
	tsconfigPath := filepath.Join(modulesRoot, "tsconfig.json")
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return nil
	}

	var tsconfig struct {
		CompilerOptions struct {
			Paths map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(data, &tsconfig); err != nil {
		// tsconfig.json is often JSONC (comments/trailing commas).
		// Use hujson to standardize into strict JSON before unmarshaling.
		jsoncValue, parseErr := hujson.Parse(data)
		if parseErr != nil {
			return nil
		}
		jsoncValue.Standardize()
		if err := json.Unmarshal(jsoncValue.Pack(), &tsconfig); err != nil {
			return nil
		}
	}
	return tsconfig.CompilerOptions.Paths
}

func hasAnyExistingTypeAsset(pathEntries []string, modulesRoot string) bool {
	for _, entry := range pathEntries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		resolvedPath := entry
		if !filepath.IsAbs(resolvedPath) {
			resolvedPath = filepath.Join(modulesRoot, filepath.FromSlash(resolvedPath))
		}
		resolvedPath = gonative.RewriteChoysumTypesPath(filepath.ToSlash(resolvedPath))

		if strings.ContainsAny(resolvedPath, "*?[]") {
			matches, err := filepath.Glob(filepath.FromSlash(resolvedPath))
			if err != nil {
				continue
			}
			for _, match := range matches {
				if st, err := os.Stat(match); err == nil && !st.IsDir() {
					return true
				}
			}
			continue
		}

		if st, err := os.Stat(filepath.FromSlash(resolvedPath)); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}
