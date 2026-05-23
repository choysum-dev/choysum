// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
)

// TestParseTsconfigPathAliasWithTsconfigRaw tests parsing path aliases from raw tsconfig JSON.
func TestParseTsconfigPathAliasWithTsconfigRaw(t *testing.T) {
	buildOptions := &api.BuildOptions{
		TsconfigRaw:   `{"compilerOptions":{"paths":{"@/*":["./src/*"],"@utils/*":["./src/utils/*"]}}}`,
		AbsWorkingDir: "/test/project",
	}

	pathAlias, err := ParseTsconfigPathAlias(buildOptions)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedAlias := filepath.Join("/test/project", "./src/*")
	if pathAlias["@/*"] != expectedAlias {
		t.Errorf("Expected alias '@/*' to be '%s', got '%s'", expectedAlias, pathAlias["@/*"])
	}

	expectedUtilsAlias := filepath.Join("/test/project", "./src/utils/*")
	if pathAlias["@utils/*"] != expectedUtilsAlias {
		t.Errorf("Expected alias '@utils/*' to be '%s', got '%s'", expectedUtilsAlias, pathAlias["@utils/*"])
	}
}

// TestParseTsconfigPathAliasWithInvalidRawJSON tests parsing with invalid raw JSON.
func TestParseTsconfigPathAliasWithInvalidRawJSON(t *testing.T) {
	buildOptions := &api.BuildOptions{
		TsconfigRaw: `{invalid json}`,
	}

	_, err := ParseTsconfigPathAlias(buildOptions)
	if err == nil {
		t.Errorf("Expected error with invalid JSON, got nil")
	}
}

// TestParseTsconfigPathAliasWithTsconfigRawNoAbsWorkingDir tests raw JSON without AbsWorkingDir.
func TestParseTsconfigPathAliasWithTsconfigRawNoAbsWorkingDir(t *testing.T) {
	buildOptions := &api.BuildOptions{
		TsconfigRaw: `{"compilerOptions":{"paths":{"@/*":["./src/*"]}}}`,
		// AbsWorkingDir is empty, should use executable path
	}

	pathAlias, err := ParseTsconfigPathAlias(buildOptions)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should have some path set (using executable path)
	if len(pathAlias) == 0 {
		t.Errorf("Expected some path aliases to be parsed")
	}
}

// TestParseTsconfigPathAliasWithTsconfigFile tests parsing from tsconfig file.
func TestParseTsconfigPathAliasWithTsconfigFile(t *testing.T) {
	// Create a temporary tsconfig file
	tmpDir := t.TempDir()
	tsconfigPath := filepath.Join(tmpDir, "tsconfig.json")
	tsconfigContent := `{
        "compilerOptions": {
            "paths": {
                "@/*": ["./src/*"],
                "@components/*": ["./src/components/*"]
            }
        }
    }`

	err := os.WriteFile(tsconfigPath, []byte(tsconfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test tsconfig file: %v", err)
	}

	buildOptions := &api.BuildOptions{
		Tsconfig: tsconfigPath,
	}

	pathAlias, err := ParseTsconfigPathAlias(buildOptions)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedAlias := filepath.Join(tmpDir, "./src/*")
	if pathAlias["@/*"] != expectedAlias {
		t.Errorf("Expected alias '@/*' to be '%s', got '%s'", expectedAlias, pathAlias["@/*"])
	}

	expectedComponentsAlias := filepath.Join(tmpDir, "./src/components/*")
	if pathAlias["@components/*"] != expectedComponentsAlias {
		t.Errorf("Expected alias '@components/*' to be '%s', got '%s'", expectedComponentsAlias, pathAlias["@components/*"])
	}
}

// TestParseTsconfigPathAliasWithNonexistentFile tests parsing with nonexistent tsconfig file.
func TestParseTsconfigPathAliasWithNonexistentFile(t *testing.T) {
	buildOptions := &api.BuildOptions{
		Tsconfig: "/nonexistent/tsconfig.json",
	}

	_, err := ParseTsconfigPathAlias(buildOptions)
	if err == nil {
		t.Errorf("Expected error with nonexistent file, got nil")
	}
}

// TestParseTsconfigPathAliasWithInvalidFileJSON tests parsing with invalid JSON in file.
func TestParseTsconfigPathAliasWithInvalidFileJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tsconfigPath := filepath.Join(tmpDir, "tsconfig.json")
	invalidContent := `{invalid json}`

	err := os.WriteFile(tsconfigPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test tsconfig file: %v", err)
	}

	buildOptions := &api.BuildOptions{
		Tsconfig: tsconfigPath,
	}

	_, err = ParseTsconfigPathAlias(buildOptions)
	if err == nil {
		t.Errorf("Expected error with invalid JSON file, got nil")
	}
}

// TestParseTsconfigPathAliasEmpty tests parsing with empty build options.
func TestParseTsconfigPathAliasEmpty(t *testing.T) {
	buildOptions := &api.BuildOptions{}

	pathAlias, err := ParseTsconfigPathAlias(buildOptions)
	if err != nil {
		t.Errorf("Expected no error with empty options, got: %v", err)
	}

	if len(pathAlias) != 0 {
		t.Errorf("Expected empty path alias map, got %d entries", len(pathAlias))
	}
}

// TestApplyPathAliasWithWildcard tests path alias application with wildcard.
func TestApplyPathAliasWithWildcard(t *testing.T) {
	pathAlias := map[string]string{
		"@/*":        "/src/*",
		"@utils/*":   "/src/utils/*",
		"@exact":     "/src/exact",
		"@special/*": "/very/long/path/to/special/*",
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"@/components/Button.vue", "/src/components/Button.vue"},
		{"@utils/helper.ts", "/src/utils/helper.ts"},
		{"@exact", "/src/exact"},
		{"@special/deep/nested/file.ts", "/very/long/path/to/special/deep/nested/file.ts"},
		{"normal/path", "normal/path"},     // No alias match
		{"@unknown/path", "@unknown/path"}, // No alias match
	}

	for _, test := range tests {
		result := ApplyPathAlias(pathAlias, test.input)
		if result != test.expected {
			t.Errorf("applyPathAlias(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

// TestApplyPathAliasWithoutWildcard tests path alias application without wildcard.
func TestApplyPathAliasWithoutWildcard(t *testing.T) {
	pathAlias := map[string]string{
		"@exact": "/src/exact",
		"@lib":   "/node_modules/lib",
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"@exact", "/src/exact"},
		{"@lib", "/node_modules/lib"},
		{"@exact/sub", "@exact/sub"}, // No match for exact alias with sub path
		{"normal", "normal"},         // No alias match
	}

	for _, test := range tests {
		result := ApplyPathAlias(pathAlias, test.input)
		if result != test.expected {
			t.Errorf("applyPathAlias(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

// TestApplyPathAliasEmptyMap tests path alias application with empty alias map.
func TestApplyPathAliasEmptyMap(t *testing.T) {
	pathAlias := map[string]string{}
	input := "@/components/Button.vue"

	result := ApplyPathAlias(pathAlias, input)
	if result != input {
		t.Errorf("Expected unchanged path with empty alias map, got: %s", result)
	}
}

// TestParseTsconfigPathAliasWithComplexPaths tests parsing with complex path configurations.
func TestParseTsconfigPathAliasWithComplexPaths(t *testing.T) {
	buildOptions := &api.BuildOptions{
		TsconfigRaw: `{
            "compilerOptions": {
                "paths": {
                    "@/*": ["./src/*"],
                    "#/*": ["./types/*"],
                    "utils": ["./src/utils/index.ts"],
                    "empty": [],
                    "multiple": ["./first/*", "./second/*"],
                    "invalid": "not_an_array"
                }
            }
        }`,
		AbsWorkingDir: "/project",
	}

	pathAlias, err := ParseTsconfigPathAlias(buildOptions)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should parse valid entries
	expectedSrc := filepath.Join("/project", "./src/*")
	if pathAlias["@/*"] != expectedSrc {
		t.Errorf("Expected '@/*' alias, got: %s", pathAlias["@/*"])
	}

	expectedTypes := filepath.Join("/project", "./types/*")
	if pathAlias["#/*"] != expectedTypes {
		t.Errorf("Expected '#/*' alias, got: %s", pathAlias["#/*"])
	}

	// Should take first entry for multiple paths
	expectedUtils := filepath.Join("/project", "./src/utils/index.ts")
	if pathAlias["utils"] != expectedUtils {
		t.Errorf("Expected 'utils' alias, got: %s", pathAlias["utils"])
	}

	expectedMultiple := filepath.Join("/project", "./first/*")
	if pathAlias["multiple"] != expectedMultiple {
		t.Errorf("Expected 'multiple' alias to use first path, got: %s", pathAlias["multiple"])
	}

	// Should skip empty arrays and invalid entries
	if _, exists := pathAlias["empty"]; exists {
		t.Errorf("Expected 'empty' alias to be skipped")
	}
	if _, exists := pathAlias["invalid"]; exists {
		t.Errorf("Expected 'invalid' alias to be skipped")
	}
}

// TestParseTsconfigPathAliasWithoutCompilerOptions tests parsing without compilerOptions.
func TestParseTsconfigPathAliasWithoutCompilerOptions(t *testing.T) {
	buildOptions := &api.BuildOptions{
		TsconfigRaw:   `{"extends": "./base.json"}`,
		AbsWorkingDir: "/project",
	}

	pathAlias, err := ParseTsconfigPathAlias(buildOptions)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(pathAlias) != 0 {
		t.Errorf("Expected empty path alias map without compilerOptions, got %d entries", len(pathAlias))
	}
}

// TestParseTsconfigPathAliasWithoutPaths tests parsing without paths in compilerOptions.
func TestParseTsconfigPathAliasWithoutPaths(t *testing.T) {
	buildOptions := &api.BuildOptions{
		TsconfigRaw: `{
            "compilerOptions": {
                "target": "ES2020",
                "module": "ESNext"
            }
        }`,
		AbsWorkingDir: "/project",
	}

	pathAlias, err := ParseTsconfigPathAlias(buildOptions)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(pathAlias) != 0 {
		t.Errorf("Expected empty path alias map without paths, got %d entries", len(pathAlias))
	}
}

func TestReplaceStylePaths(t *testing.T) {
	// Create OS-specific absolute paths to ensure tests are reliable on all platforms, including Windows.
	absFoo, _ := filepath.Abs(filepath.FromSlash("/abs/foo"))
	absStar, _ := filepath.Abs(filepath.FromSlash("/abs/*"))
	baseFile, _ := filepath.Abs(filepath.FromSlash("/project/src/app.vue"))
	pathAlias := map[string]string{
		"@/foo": absFoo,
		"@/*":   absStar,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "import with alias",
			input:    `@import "@/foo";`,
			expected: `@import "../../abs/foo";`,
		},
		{
			name:     "import with wildcard alias",
			input:    `@import "@/bar";`,
			expected: `@import "../../abs/bar";`,
		},
		{
			name:     "import with relative path",
			input:    `@import "./local.scss";`,
			expected: `@import "./local.scss";`,
		},
		{
			name:     "use with alias",
			input:    `@use "@/foo" as *;`,
			expected: `@use "../../abs/foo" as *;`,
		},
		{
			name:     "forward with alias",
			input:    `@forward "@/foo";`,
			expected: `@forward "../../abs/foo";`,
		},
		{
			name:     "url with alias",
			input:    `url("@/foo")`,
			expected: `url("../../abs/foo")`,
		},
		{
			name:     "url with relative path",
			input:    `url("./img.png")`,
			expected: `url("./img.png")`,
		},
		{
			name:     "no match",
			input:    `@import "plain.css";`,
			expected: "", // will set below
		},
		{
			name:     "invalid match",
			input:    `@import ;`,
			expected: `@import ;`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := replaceStylePaths(test.input, baseFile, pathAlias)
			if test.name == "no match" {
				expectedAbs, _ := filepath.Abs("plain.css")
				relPath, _ := filepath.Rel(filepath.Dir(baseFile), expectedAbs)
				relPath = strings.ReplaceAll(relPath, `\`, `/`)
				if !strings.HasPrefix(relPath, ".") {
					relPath = "./" + relPath
				}
				test.expected = `@import "` + relPath + `";`
			}
			// Normalize to POSIX style for cross-platform compatibility
			got = strings.ReplaceAll(got, `\`, `/`)
			test.expected = strings.ReplaceAll(test.expected, `\`, `/`)
			if got != test.expected {
				t.Errorf("replaceStylePaths: got %q, want %q", got, test.expected)
			}
		})
	}
}

func TestResolveVueStylePath(t *testing.T) {
	absFoo, _ := filepath.Abs(filepath.FromSlash("/abs/foo"))
	vueFile, _ := filepath.Abs(filepath.FromSlash("/project/src/app.vue"))
	pathAlias := map[string]string{
		"@/foo": absFoo,
	}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single style block",
			input:    `<style>@import "@/foo";</style>`,
			expected: `<style>@import "../../abs/foo";</style>`,
		},
		{
			name:     "style block with use and forward",
			input:    `<style>@use "@/foo" as *; @forward "@/foo";</style>`,
			expected: `<style>@use "../../abs/foo" as *; @forward "../../abs/foo";</style>`,
		},
		{
			name:     "multiple style blocks",
			input:    `<style>@import "@/foo";</style><style>@import "@/foo";</style>`,
			expected: `<style>@import "../../abs/foo";</style><style>@import "../../abs/foo";</style>`,
		},
		{
			name:     "style block with no match",
			input:    `<style>body { color: red; }</style>`,
			expected: `<style>body { color: red; }</style>`,
		},
		{
			name:     "no style block",
			input:    `<template><div></div></template>`,
			expected: `<template><div></div></template>`,
		},
		{
			name:     "invalid style block",
			input:    `<style></style>`,
			expected: `<style></style>`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ResolveVueStylePath(test.input, vueFile, pathAlias)
			got = strings.ReplaceAll(got, `\`, `/`)
			test.expected = strings.ReplaceAll(test.expected, `\`, `/`)
			if got != test.expected {
				t.Errorf("ResolveVueStylePath: got %q, want %q", got, test.expected)
			}
		})
	}
}

func TestResolveScssPath(t *testing.T) {
	absFoo, _ := filepath.Abs(filepath.FromSlash("/abs/foo"))
	scssFile, _ := filepath.Abs(filepath.FromSlash("/project/src/app.scss"))
	pathAlias := map[string]string{
		"@/foo": absFoo,
	}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "import with alias",
			input:    `@import "@/foo";`,
			expected: `@import "../../abs/foo";`,
		},
		{
			name:     "import with relative path",
			input:    `@import "./local.scss";`,
			expected: `@import "./local.scss";`,
		},
		{
			name:     "use with alias",
			input:    `@use "@/foo" as *;`,
			expected: `@use "../../abs/foo" as *;`,
		},
		{
			name:     "forward with alias",
			input:    `@forward "@/foo";`,
			expected: `@forward "../../abs/foo";`,
		},
		{
			name:     "url with alias",
			input:    `url("@/foo")`,
			expected: `url("../../abs/foo")`,
		},
		{
			name:     "url with relative path",
			input:    `url("./img.png")`,
			expected: `url("./img.png")`,
		},
		{
			name:     "no match",
			input:    `@import "plain.css";`,
			expected: "", // will set below
		},
		{
			name:     "invalid match",
			input:    `@import ;`,
			expected: `@import ;`,
		},
		{
			name:     "import with empty path",
			input:    `@import "";`,
			expected: `@import "";`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ResolveScssPath(test.input, scssFile, pathAlias)
			if test.name == "no match" {
				expectedAbs, _ := filepath.Abs("plain.css")
				relPath, _ := filepath.Rel(filepath.Dir(scssFile), expectedAbs)
				relPath = strings.ReplaceAll(relPath, `\`, `/`)
				if !strings.HasPrefix(relPath, ".") {
					relPath = "./" + relPath
				}
				test.expected = `@import "` + relPath + `";`
			}
			// Normalize to POSIX style for cross-platform compatibility
			got = strings.ReplaceAll(got, `\`, `/`)
			test.expected = strings.ReplaceAll(test.expected, `\`, `/`)
			if got != test.expected {
				t.Errorf("ResolveScssPath: got %q, want %q", got, test.expected)
			}
		})
	}
}
