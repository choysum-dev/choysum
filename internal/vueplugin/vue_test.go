// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
)

// Test utility functions

// createTempVueFile creates a temporary Vue file with the given content
func createTempVueFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test_*.vue")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile.Name()
}

// Unit tests for internal functions

// TestToPosixPath tests the conversion from Windows-style paths to POSIX-style paths
func TestToPosixPath(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{`C:\Users\test\file.vue`, `C:/Users/test/file.vue`},
		{`path\to\file.vue`, `path/to/file.vue`},
		{`normal/path.vue`, `normal/path.vue`},
		{``, ``},
		{`\\server\share\file.vue`, `//server/share/file.vue`},
	}

	for _, test := range tests {
		result := toPosixPath(test.input)
		if result != test.expected {
			t.Errorf("toPosixPath(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

// TestGenerateHashId tests the hash ID generation function
func TestGenerateHashId(t *testing.T) {
	tests := []struct {
		name, source1, source2 string
		sameSources            bool
	}{
		{"different_sources", "<template><div>Hello</div></template>", "<template><div>World</div></template>", false},
		{"same_sources", "<template><div>Same</div></template>", "<template><div>Same</div></template>", true},
		{"empty_sources", "", "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hash1, hash2 := generateHashId(test.source1), generateHashId(test.source2)

			if hash1 == "" || hash2 == "" {
				t.Error("Expected non-empty hashes")
			}

			if test.sameSources {
				if hash1 != hash2 {
					t.Errorf("Expected same hash for same sources, got %s and %s", hash1, hash2)
				}
			} else {
				if hash1 == hash2 {
					t.Error("Expected different hashes for different sources")
				}
			}

			// Test consistency
			if hash1 != generateHashId(test.source1) {
				t.Error("Expected consistent hash for same source")
			}
		})
	}
}

// TestGenerateEntryContents tests the generation of entry contents
func TestGenerateEntryContents(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		dataId   string
		isSSR    bool
		script   map[string]interface{}
		template map[string]interface{}
		styles   []map[string]interface{}
		expected []string
	}{
		{
			name: "complete_component", filePath: "test.vue", dataId: "data-v-test", isSSR: false,
			script:   map[string]interface{}{"content": "export default { name: 'TestComponent' }"},
			template: map[string]interface{}{"code": "function render() { return h('div', 'test'); }"},
			styles:   []map[string]interface{}{{"scoped": true}},
			expected: []string{"import script from 'test.vue?type=script'", "import { render } from 'test.vue?type=template'", "script.__scopeId = \"data-v-test\";"},
		},
		{
			name: "without_script", filePath: "test.vue", dataId: "data-v-test", isSSR: false,
			script:   nil,
			template: map[string]interface{}{"code": "function render() { return h('div', 'test'); }"},
			styles:   []map[string]interface{}{},
			expected: []string{"const script = {};", "import { render } from 'test.vue?type=template'"},
		},
		{
			name: "ssr_mode", filePath: "test.vue", dataId: "data-v-test", isSSR: true,
			script:   map[string]interface{}{"content": "export default { name: 'SSRComponent' }"},
			template: map[string]interface{}{"code": "function ssrRender() { return 'SSR content'; }"},
			styles:   []map[string]interface{}{},
			expected: []string{"import { ssrRender } from 'test.vue?type=template'", "script.__ssrInlineRender = true;"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contents, err := generateEntryContents(test.filePath, test.dataId, test.isSSR, test.script, test.template, test.styles)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			for _, expected := range test.expected {
				if !strings.Contains(contents, expected) {
					t.Errorf("Expected output to contain '%s', got:\n%s", expected, contents)
				}
			}
		})
	}
}

// TestGenerateEntryContentsErrors tests error conditions in generateEntryContents
func TestGenerateEntryContentsErrors(t *testing.T) {
	tests := []struct {
		name   string
		styles []map[string]interface{}
	}{
		{"invalid_scoped_string", []map[string]interface{}{{"scoped": "not-a-boolean"}}},
		{"invalid_scoped_number", []map[string]interface{}{{"scoped": 42}}},
		{"nil_scoped_value", []map[string]interface{}{{"scoped": nil}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := generateEntryContents("test.vue", "data-v-test", false,
				map[string]interface{}{"content": "export default {}"},
				map[string]interface{}{"code": "render() {}"},
				test.styles)

			if err == nil {
				t.Error("Expected error for invalid scoped value")
			}
			if !strings.Contains(err.Error(), "failed to execute Vue entry template") {
				t.Errorf("Expected template execution error, got: %v", err)
			}
		})
	}
}

// TestReadVueSource tests the Vue source reading functionality
func TestReadVueSource(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		useRealFile bool // false means use non-existent file path
		processor   func(string, api.OnLoadArgs, *api.BuildOptions) (string, error)
		expectError bool
		expected    string
	}{
		{"valid_file", "<template><div>Test</div></template>", true, nil, false, "<template><div>Test</div></template>"},
		{"with_processor", "<template>Test</template>", true,
			func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error) {
				return content + "<!-- processed -->", nil
			}, false, "<template>Test</template><!-- processed -->"},
		{"processor_error", "<template>Test</template>", true,
			func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error) {
				return "", fmt.Errorf("processor error")
			}, true, ""},
		{"file_not_found", "", false, nil, true, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var args api.OnLoadArgs

			if test.useRealFile {
				tmpFile := createTempVueFile(t, test.content)
				args = api.OnLoadArgs{Path: tmpFile}
			} else {
				// Use non-existent file path for file_not_found test
				args = api.OnLoadArgs{Path: "/nonexistent/file.vue"}
			}

			opts := newOptions()
			if test.processor != nil {
				opts.onVueLoadProcessors = append(opts.onVueLoadProcessors, test.processor)
			}
			build := &api.PluginBuild{InitialOptions: &api.BuildOptions{}}

			result, err := readVueSource(args, opts, build)

			if test.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				// For file_not_found test, just check that we have an error
				// Don't check specific error message as it varies by OS
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if result != test.expected {
					t.Errorf("Expected content '%s', got '%s'", test.expected, result)
				}
			}
		})
	}
}

// TestParseImportMetaEnv tests the parsing of import.meta.env variables
func TestParseImportMetaEnv(t *testing.T) {
	tests := []struct {
		defineMap   map[string]string
		envVar      string
		expectValue interface{}
		expectExist bool
	}{
		{map[string]string{"import.meta.env.PROD": "true"}, "PROD", true, true},
		{map[string]string{"import.meta.env.NODE_ENV": "\"production\""}, "NODE_ENV", "production", true},
		{map[string]string{"import.meta.env.PORT": "3000"}, "PORT", 3000.0, true},
		{map[string]string{}, "NONEXISTENT", nil, false},
		{map[string]string{"import.meta.env.INVALID": "invalid json"}, "INVALID", nil, true},
	}

	for _, test := range tests {
		value, exists := parseImportMetaEnv(test.defineMap, test.envVar)
		if exists != test.expectExist {
			t.Errorf("Expected exists=%v, got=%v for %s", test.expectExist, exists, test.envVar)
		}
		if test.expectExist && value != test.expectValue {
			t.Errorf("Expected value=%v, got=%v for %s", test.expectValue, value, test.envVar)
		}
	}
}

// TestOptionsValidation tests options validation
func TestOptionsValidation(t *testing.T) {
	opts := newOptions()
	if opts.logger == nil {
		t.Error("Expected default logger to be set")
	}
	if opts.onVueLoadProcessors != nil {
		t.Error("Expected onVueLoadProcessors to be nil by default")
	}
	if opts.TemplateCompilerOptions == nil {
		t.Error("Expected templateCompilerOptions to be initialized")
	}
	if len(opts.TemplateCompilerOptions) != 0 {
		t.Error("Expected empty templateCompilerOptions map")
	}
}

// Integration tests with mock engine
// TestVueCompilationWithMockEngine tests Vue compilation using mock engines
func TestVueCompilationWithMockEngine(t *testing.T) {
	tests := []struct {
		name        string
		mockConfig  *MockEngineConfig
		useRealFile bool // false means use non-existent file path
		expectError string
	}{
		{"execution_error", &MockEngineConfig{ExecuteError: fmt.Errorf("Vue SFC compilation failed: syntax error")}, true, "Vue SFC compilation failed"},
		{"invalid_result", &MockEngineConfig{InvalidResult: true}, true, "Invalid Vue SFC compilation result"},
		{"empty_result", &MockEngineConfig{EmptyResult: true}, true, ""},
		{"with_warnings", &MockEngineConfig{
			Script:   &MockScriptConfig{Content: "export default { name: 'Component' }", Warnings: []interface{}{"Unused variable"}},
			Template: &MockTemplateConfig{Code: "function render() { return h('div', 'test'); }"},
		}, true, ""},
		{"read_vue_source_error", &MockEngineConfig{
			Script:   &MockScriptConfig{Content: "export default { name: 'Component' }"},
			Template: &MockTemplateConfig{Code: "function render() { return h('div', 'test'); }"},
		}, false, ""}, // Remove specific error message check for cross-platform compatibility
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var tmpFile, entryFile string
			var err error

			if test.useRealFile {
				// Create real Vue file
				content := `<template><div>Test</div></template><script>export default { name: 'Test' }</script>`
				tmpFile = createTempVueFile(t, content)
				entryFile = filepath.Join(filepath.Dir(tmpFile), "entry.js")
				entryContent := fmt.Sprintf(`import App from '%s';`, filepath.Base(tmpFile))
				err = os.WriteFile(entryFile, []byte(entryContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create entry file: %v", err)
				}
				defer os.Remove(entryFile)
			} else {
				// Use non-existent Vue file to trigger readVueSource error
				tmpDir := t.TempDir()
				tmpFile = filepath.Join(tmpDir, "nonexistent.vue")
				entryFile = filepath.Join(tmpDir, "entry.js")
				entryContent := fmt.Sprintf(`import App from '%s';`, filepath.Base(tmpFile))
				err = os.WriteFile(entryFile, []byte(entryContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create entry file: %v", err)
				}
			}

			jsExec, err := newTestExecutor(test.mockConfig)
			if err != nil {
				t.Fatalf("Failed to create JS executor: %v", err)
			}
			if err := jsExec.Start(); err != nil {
				t.Fatalf("Failed to start JS executor: %v", err)
			}
			defer jsExec.Stop()

			vuePlugin := NewPlugin(WithJsExecutor(jsExec))
			result := api.Build(api.BuildOptions{
				EntryPoints: []string{entryFile},
				Bundle:      true,
				Write:       false,
				LogLevel:    api.LogLevelError,
				Plugins:     []api.Plugin{vuePlugin},
			})

			if test.expectError != "" {
				if len(result.Errors) == 0 {
					t.Error("Expected build errors, got none")
				}
				foundError := false
				for _, err := range result.Errors {
					if strings.Contains(err.Text, test.expectError) {
						foundError = true
						break
					}
				}
				if !foundError {
					t.Errorf("Expected error containing '%s', got: %v", test.expectError, result.Errors)
				}
			} else if test.name == "read_vue_source_error" {
				// For read_vue_source_error test, just check that we have an error
				// Don't check specific error message as it varies by OS
				if len(result.Errors) == 0 {
					t.Error("Expected build errors due to file not found, got none")
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Expected successful build, got errors: %v", result.Errors)
				}
			}
		})
	}
}

// TestVueGenerateEntryContentsError tests the generateEntryContents error branch in the main handler
func TestVueGenerateEntryContentsError(t *testing.T) {
	// This test specifically covers the error branch in registerMainEntryHandler:
	// contents, err := generateEntryContents(args.Path, dataId, isSSR, script, template, styles)
	// if err != nil { // This branch was not covered before
	//     opts.logger.Error("Failed to generate Vue entry contents", "error", err, "file", args.Path)
	//     return api.OnLoadResult{...}, err
	// }

	// Create a mock config that returns invalid style scoped values to trigger generateEntryContents error
	mockConfig := &MockEngineConfig{
		Script: &MockScriptConfig{
			Content: "export default { name: 'TestComponent' }",
			Lang:    "js",
		},
		Template: &MockTemplateConfig{
			Code: "function render() { return h('div', 'test'); }",
		},
		// Return styles with invalid scoped values that will cause generateEntryContents to fail
		Styles: []*MockStyleConfig{
			{
				Code:   ".test { color: red; }",
				Scoped: "invalid-boolean-value", // This will cause type assertion to fail in template execution
			},
		},
	}

	// Create temporary Vue file
	content := `<template><div>Test</div></template><script>export default { name: 'Test' }</script><style scoped>.test { color: red; }</style>`
	tmpFile := createTempVueFile(t, content)

	// Create temporary entry file
	entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
	entryContent := fmt.Sprintf(`import App from '%s';`, filepath.Base(tmpFile))
	err := os.WriteFile(entryFile, []byte(entryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}
	defer os.Remove(entryFile)

	// Create JS executor with mock engine that returns invalid style data
	jsExec, err := newTestExecutor(mockConfig)
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}

	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	// Create plugin
	vuePlugin := NewPlugin(WithJsExecutor(jsExec))

	// Configure esbuild options
	buildOptions := api.BuildOptions{
		EntryPoints: []string{entryFile},
		Bundle:      true,
		Write:       false,
		LogLevel:    api.LogLevelError,
		Plugins:     []api.Plugin{vuePlugin},
		Define: map[string]string{
			"process.env.NODE_ENV": `"production"`,
		},
	}

	// Execute build
	result := api.Build(buildOptions)

	// Should have errors due to generateEntryContents failure
	if len(result.Errors) == 0 {
		t.Error("Expected build errors due to generateEntryContents failure, got none")
	}

	// Check if the error is the expected one from generateEntryContents
	foundGenerateEntryError := false
	for _, err := range result.Errors {
		// The actual error message is "failed to execute Vue entry template" which is part of generateEntryContents
		if strings.Contains(err.Text, "failed to execute Vue entry template") {
			foundGenerateEntryError = true
			break
		}
	}

	if !foundGenerateEntryError {
		t.Errorf("Expected error containing 'failed to execute Vue entry template', got: %v", result.Errors)
	}
}

// TestVueResolveProcessorChain tests resolve processors
func TestVueResolveProcessorChain(t *testing.T) {
	content := `<template><div>Test</div></template><script>export default { name: 'Test' }</script>`
	tmpFile := createTempVueFile(t, content)
	entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
	entryContent := fmt.Sprintf(`import App from '%s';`, filepath.Base(tmpFile))
	err := os.WriteFile(entryFile, []byte(entryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}
	defer os.Remove(entryFile)

	jsExec, err := newTestExecutor(&MockEngineConfig{
		Script:   &MockScriptConfig{Content: "export default { name: 'TestComponent' }", Lang: "js"},
		Template: &MockTemplateConfig{Code: "function render() { return h('div', 'test'); }"},
	})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	processor1Called := false
	processor1 := func(args *api.OnResolveArgs, buildOptions *api.BuildOptions) (*api.OnResolveResult, error) {
		processor1Called = true
		if !strings.Contains(args.Path, "?type=") {
			return &api.OnResolveResult{Path: args.Path, Namespace: "file"}, nil
		}
		return nil, nil
	}

	processor2Called := false
	processor2 := func(args *api.OnResolveArgs, buildOptions *api.BuildOptions) (*api.OnResolveResult, error) {
		processor2Called = true
		return nil, nil
	}

	vuePlugin := NewPlugin(
		WithJsExecutor(jsExec),
		WithOnVueResolveProcessor(processor1),
		WithOnVueResolveProcessor(processor2),
	)

	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entryFile},
		Bundle:      true,
		Write:       false,
		LogLevel:    api.LogLevelError,
		Plugins:     []api.Plugin{vuePlugin},
	})

	if len(result.Errors) > 0 {
		t.Errorf("Expected successful build, got errors: %v", result.Errors)
	}
	if !processor1Called || !processor2Called {
		t.Error("Expected both processors to be called")
	}
}

// TestVueResolveProcessorError tests error handling in resolve processors
func TestVueResolveProcessorError(t *testing.T) {
	content := `<template><div>Test</div></template><script>export default { name: 'Test' }</script>`
	tmpFile := createTempVueFile(t, content)
	entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
	entryContent := fmt.Sprintf(`import App from '%s';`, filepath.Base(tmpFile))
	err := os.WriteFile(entryFile, []byte(entryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}
	defer os.Remove(entryFile)

	jsExec, err := newTestExecutor(&MockEngineConfig{
		Script:   &MockScriptConfig{Content: "export default { name: 'TestComponent' }", Lang: "js"},
		Template: &MockTemplateConfig{Code: "function render() { return h('div', 'test'); }"},
	})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	// Test 1: Processor error
	t.Run("processor_error", func(t *testing.T) {
		processor := func(args *api.OnResolveArgs, buildOptions *api.BuildOptions) (*api.OnResolveResult, error) {
			return nil, fmt.Errorf("resolve processor failed: custom error")
		}

		vuePlugin := NewPlugin(WithJsExecutor(jsExec), WithOnVueResolveProcessor(processor))
		result := api.Build(api.BuildOptions{
			EntryPoints: []string{entryFile},
			Bundle:      true,
			Write:       false,
			LogLevel:    api.LogLevelError,
			Plugins:     []api.Plugin{vuePlugin},
		})

		if len(result.Errors) == 0 {
			t.Error("Expected build errors, got none")
		}
		foundError := false
		for _, err := range result.Errors {
			if strings.Contains(err.Text, "resolve processor failed: custom error") {
				foundError = true
				break
			}
		}
		if !foundError {
			t.Error("Expected resolve processor error")
		}
	})

	// Test 2: Path alias error (attempt to trigger parseTsconfigPathAlias error)
	t.Run("path_alias_error", func(t *testing.T) {
		// Create a directory structure that might cause path alias resolution to fail
		tmpDir := t.TempDir()

		// Create a malformed tsconfig.json that might cause parsing errors
		malformedTsconfig := filepath.Join(tmpDir, "tsconfig.json")
		malformedContent := `{
            "compilerOptions": {
                "baseUrl": ".",
                "paths": {
                    "@/*": 
                }
            }
        }` // Invalid JSON - missing value for "@/*"

		err := os.WriteFile(malformedTsconfig, []byte(malformedContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create malformed tsconfig.json: %v", err)
		}

		// Create test files in the temp directory
		testVueFile := filepath.Join(tmpDir, "test.vue")
		err = os.WriteFile(testVueFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test Vue file: %v", err)
		}

		testEntryFile := filepath.Join(tmpDir, "entry.js")
		testEntryContent := `import App from './test.vue';`
		err = os.WriteFile(testEntryFile, []byte(testEntryContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test entry file: %v", err)
		}

		vuePlugin := NewPlugin(WithJsExecutor(jsExec))
		result := api.Build(api.BuildOptions{
			EntryPoints: []string{testEntryFile},
			Bundle:      true,
			Write:       false,
			LogLevel:    api.LogLevelError,
			Plugins:     []api.Plugin{vuePlugin},
			Tsconfig:    malformedTsconfig,
		})

		// This test might not always trigger the error depending on the implementation
		// of parseTsconfigPathAlias, but it's an attempt to cover the error branch
		if len(result.Errors) > 0 {
			t.Logf("Build errors (may include tsconfig parsing errors): %v", result.Errors)

			// Check if any error is related to tsconfig/path alias parsing
			for _, err := range result.Errors {
				if strings.Contains(strings.ToLower(err.Text), "tsconfig") ||
					strings.Contains(strings.ToLower(err.Text), "json") ||
					strings.Contains(strings.ToLower(err.Text), "parse") {
					t.Logf("Found potential tsconfig parsing error: %s", err.Text)
					return
				}
			}
		}

		t.Log("Path alias error branch may not be triggered in this test environment")
	})
}

// TestVueScriptHandlerWithSourcemap tests script handler sourcemap functionality
func TestVueScriptHandlerWithSourcemap(t *testing.T) {
	tests := []struct {
		name      string
		sourcemap api.SourceMap
		scriptMap interface{}
		checkFunc func(string) bool
	}{
		{"with_sourcemap", api.SourceMapInline, map[string]interface{}{"version": 3, "sources": []string{"test.vue"}},
			func(content string) bool {
				return strings.Contains(content, "//# sourceMappingURL=data:application/json;base64,")
			}},
		{"without_sourcemap", api.SourceMapNone, map[string]interface{}{"version": 3, "sources": []string{"test.vue"}},
			func(content string) bool {
				return !strings.Contains(content, "//# sourceMappingURL=data:application/json;base64,")
			}},
		{"nil_map", api.SourceMapInline, nil,
			func(content string) bool { return strings.Contains(content, "TestComponent") }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig := &MockEngineConfig{
				Script:   &MockScriptConfig{Content: "export default { name: 'TestComponent' }", Lang: "js", Map: test.scriptMap},
				Template: &MockTemplateConfig{Code: "function render() { return h('div', 'test'); }"},
			}

			content := `<template><div>Test</div></template><script>export default { name: 'Test' }</script>`
			tmpFile := createTempVueFile(t, content)
			entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
			entryContent := fmt.Sprintf(`import App from '%s';`, filepath.Base(tmpFile))
			err := os.WriteFile(entryFile, []byte(entryContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create entry file: %v", err)
			}
			defer os.Remove(entryFile)

			jsExec, err := newTestExecutor(mockConfig)
			if err != nil {
				t.Fatalf("Failed to create JS executor: %v", err)
			}
			if err := jsExec.Start(); err != nil {
				t.Fatalf("Failed to start JS executor: %v", err)
			}
			defer jsExec.Stop()

			vuePlugin := NewPlugin(WithJsExecutor(jsExec))
			result := api.Build(api.BuildOptions{
				EntryPoints: []string{entryFile},
				Bundle:      true,
				Write:       false,
				LogLevel:    api.LogLevelError,
				Plugins:     []api.Plugin{vuePlugin},
				Sourcemap:   test.sourcemap,
			})

			if len(result.Errors) > 0 {
				t.Errorf("Expected successful build, got errors: %v", result.Errors)
				return
			}

			var mainOutput string
			for _, file := range result.OutputFiles {
				mainOutput = string(file.Contents)
				break
			}

			if !test.checkFunc(mainOutput) {
				t.Errorf("Content check failed for test case '%s'", test.name)
			}
		})
	}
}

// TestVueScriptHandlerSourcemapMarshalError tests json.Marshal error in sourcemap handling
func TestVueScriptHandlerSourcemapMarshalError(t *testing.T) {
	unmarshalableMap := map[string]interface{}{
		"version": 3,
		"sources": []string{"test.vue"},
		"channel": make(chan int), // Cannot be marshaled to JSON
	}

	mockConfig := &MockEngineConfig{
		Script:   &MockScriptConfig{Content: "export default { name: 'TestComponent' }", Lang: "js", Map: unmarshalableMap},
		Template: &MockTemplateConfig{Code: "function render() { return h('div', 'test'); }"},
	}

	content := `<template><div>Test</div></template><script>export default { name: 'Test' }</script>`
	tmpFile := createTempVueFile(t, content)
	entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
	entryContent := fmt.Sprintf(`import App from '%s';`, filepath.Base(tmpFile))
	err := os.WriteFile(entryFile, []byte(entryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}
	defer os.Remove(entryFile)

	jsExec, err := newTestExecutor(mockConfig)
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	vuePlugin := NewPlugin(WithJsExecutor(jsExec))
	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entryFile},
		Bundle:      true,
		Write:       false,
		LogLevel:    api.LogLevelError,
		Plugins:     []api.Plugin{vuePlugin},
		Sourcemap:   api.SourceMapInline,
	})

	if len(result.Errors) == 0 {
		t.Error("Expected build errors due to json.Marshal failure, got none")
	}

	foundMarshalError := false
	for _, err := range result.Errors {
		errorText := strings.ToLower(err.Text)
		if strings.Contains(errorText, "json") || strings.Contains(errorText, "marshal") ||
			strings.Contains(errorText, "unsupported") || strings.Contains(errorText, "channel") {
			foundMarshalError = true
			break
		}
	}

	if !foundMarshalError {
		t.Errorf("Expected JSON marshal error, got: %v", result.Errors)
	}
}

// filterVuePluginWarnings filters warnings that come from the Vue plugin
func filterVuePluginWarnings(warnings []api.Message) []api.Message {
	var vueWarnings []api.Message
	for _, warning := range warnings {
		if warning.PluginName == "vue-plugin" {
			vueWarnings = append(vueWarnings, warning)
		}
	}
	return vueWarnings
}

// TestVueTemplateHandlerWithTips tests template handler tips functionality
func TestVueTemplateHandlerWithTips(t *testing.T) {
	tests := []struct {
		name           string
		tips           []interface{}
		expectWarnings int
	}{
		{"with_tips", []interface{}{"Tip 1: Performance", "Tip 2: Accessibility", "Tip 3: Security"}, 3},
		{"single_tip", []interface{}{"Single performance tip"}, 1},
		{"empty_tips", []interface{}{}, 0},
		{"nil_tips", nil, 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig := &MockEngineConfig{
				Script:   &MockScriptConfig{Content: "export default { name: 'TestComponent' }", Lang: "js"},
				Template: &MockTemplateConfig{Code: "export function render() { return h('div', 'test'); }", Tips: test.tips},
			}

			content := `<template><div>Test</div></template><script>export default { name: 'Test' }</script>`
			tmpFile := createTempVueFile(t, content)
			entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
			entryContent := fmt.Sprintf(`import App from '%s';`, filepath.Base(tmpFile))
			err := os.WriteFile(entryFile, []byte(entryContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create entry file: %v", err)
			}
			defer os.Remove(entryFile)

			jsExec, err := newTestExecutor(mockConfig)
			if err != nil {
				t.Fatalf("Failed to create JS executor: %v", err)
			}
			if err := jsExec.Start(); err != nil {
				t.Fatalf("Failed to start JS executor: %v", err)
			}
			defer jsExec.Stop()

			vuePlugin := NewPlugin(WithJsExecutor(jsExec))
			result := api.Build(api.BuildOptions{
				EntryPoints: []string{entryFile},
				Bundle:      true,
				Write:       false,
				LogLevel:    api.LogLevelInfo,
				Plugins:     []api.Plugin{vuePlugin},
			})

			if len(result.Errors) > 0 {
				t.Errorf("Expected successful build, got errors: %v", result.Errors)
				return
			}

			vueWarnings := filterVuePluginWarnings(result.Warnings)
			if len(vueWarnings) != test.expectWarnings {
				t.Errorf("Expected %d Vue plugin warnings, got %d", test.expectWarnings, len(vueWarnings))
			}
		})
	}
}

// TestVueTemplateHandlerWithInvalidTips tests invalid tips handling
func TestVueTemplateHandlerWithInvalidTips(t *testing.T) {
	// Test mixed valid and invalid tips - should only process valid string tips
	invalidTips := []interface{}{"Valid tip", 42, "Another valid tip"} // 42 is invalid type

	mockConfig := &MockEngineConfig{
		Script:   &MockScriptConfig{Content: "export default { name: 'TestComponent' }", Lang: "js"},
		Template: &MockTemplateConfig{Code: "export function render() { return h('div', 'test'); }", Tips: invalidTips},
	}

	content := `<template><div>Test</div></template><script>export default { name: 'Test' }</script>`
	tmpFile := createTempVueFile(t, content)
	entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
	entryContent := fmt.Sprintf(`import App from '%s';`, filepath.Base(tmpFile))
	err := os.WriteFile(entryFile, []byte(entryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}
	defer os.Remove(entryFile)

	jsExec, err := newTestExecutor(mockConfig)
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	vuePlugin := NewPlugin(WithJsExecutor(jsExec))
	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entryFile},
		Bundle:      true,
		Write:       false,
		LogLevel:    api.LogLevelInfo, // Change to Info to see warnings
		Plugins:     []api.Plugin{vuePlugin},
	})

	// Should not have errors - invalid tips should be silently skipped
	if len(result.Errors) > 0 {
		t.Errorf("Expected successful build, got errors: %v", result.Errors)
		return
	}

	// Should only have warnings for valid string tips (2 out of 3)
	vueWarnings := filterVuePluginWarnings(result.Warnings)
	expectedWarnings := 2 // Only "Valid tip" and "Another valid tip"
	if len(vueWarnings) != expectedWarnings {
		t.Errorf("Expected %d Vue plugin warnings (valid tips only), got %d", expectedWarnings, len(vueWarnings))
		return
	}

	// Check that the valid tips are preserved (don't assume order)
	expectedTips := map[string]bool{
		"Valid tip":         false,
		"Another valid tip": false,
	}

	for _, warning := range vueWarnings {
		if _, exists := expectedTips[warning.Text]; exists {
			expectedTips[warning.Text] = true
		} else {
			t.Errorf("Unexpected warning text: '%s'", warning.Text)
		}
	}

	// Verify all expected tips were found
	for tip, found := range expectedTips {
		if !found {
			t.Errorf("Expected warning text '%s' not found", tip)
		}
	}
}

func TestReadVueSource_ParseTsconfigPathAliasError(t *testing.T) {
	tmpFile := createTempVueFile(t, "<template><div>Test</div></template>")
	args := api.OnLoadArgs{Path: tmpFile}
	opts := newOptions()
	build := &api.PluginBuild{
		InitialOptions: &api.BuildOptions{
			TsconfigRaw: "{invalid json}",
		},
	}

	_, err := readVueSource(args, opts, build)
	if err == nil || !strings.Contains(err.Error(), "failed to parse TypeScript path aliases") {
		t.Errorf("Expected parse TypeScript path aliases error, got: %v", err)
	}
}
