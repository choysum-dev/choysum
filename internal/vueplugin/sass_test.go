// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/evanw/esbuild/pkg/api"
)

// Test utility functions

func createTempSassFile(t *testing.T, content string, ext string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test_*."+ext)
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

func buildSassTest(t *testing.T, entryFile string, jsExec jsexecutor.JsExecutor, additionalOptions ...func(*api.BuildOptions)) api.BuildResult {
	t.Helper()

	vuePlugin := NewPlugin(WithJsExecutor(jsExec))
	outDir := t.TempDir()

	options := api.BuildOptions{
		EntryPoints:   []string{entryFile},
		Bundle:        true,
		Write:         false,
		LogLevel:      api.LogLevelError,
		Plugins:       []api.Plugin{vuePlugin},
		Outdir:        outDir,
		AbsWorkingDir: filepath.Dir(entryFile),
	}

	for _, fn := range additionalOptions {
		fn(&options)
	}

	return api.Build(options)
}

// Unit tests

func TestSetupSassHandler(t *testing.T) {
	jsExec, err := newTestExecutor(&MockEngineConfig{
		Sass: &MockSassConfig{CSS: ".test { color: #333; }"},
	})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	opts := newOptions()
	opts.jsExecutor = jsExec

	build := &api.PluginBuild{
		InitialOptions: &api.BuildOptions{},
		OnResolve: func(options api.OnResolveOptions, callback func(api.OnResolveArgs) (api.OnResolveResult, error)) {
			t.Logf("OnResolve called with filter: %s", options.Filter)
		},
		OnLoad: func(options api.OnLoadOptions, callback func(api.OnLoadArgs) (api.OnLoadResult, error)) {
			t.Logf("OnLoad called with filter: %s, namespace: %s", options.Filter, options.Namespace)
		},
	}

	setupSassHandler(opts, build)
	t.Log("setupSassHandler completed without errors")
}

func TestReadSassSource(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		processor   func(string, api.OnLoadArgs, *api.BuildOptions) (string, error)
		expectError bool
		expected    string
	}{
		{
			name:        "valid_scss_file",
			content:     "$primary: #333;\n.button { color: $primary; }",
			expectError: false,
			expected:    "$primary: #333;\n.button { color: $primary; }",
		},
		{
			name:    "with_processor",
			content: "$color: red;\n.test { color: $color; }",
			processor: func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error) {
				return "$color: blue;\n.test { color: $color; }", nil
			},
			expectError: false,
			expected:    "$color: blue;\n.test { color: $color; }",
		},
		{
			name:    "processor_error",
			content: "$color: red;",
			processor: func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error) {
				return "", fmt.Errorf("processor error")
			},
			expectError: true,
		},
		{
			name:    "processor_returns_empty",
			content: "$color: green;\n.test { color: $color; }",
			processor: func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error) {
				return "", nil // Empty return, should fall back to file reading
			},
			expectError: false,
			expected:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpFile := createTempSassFile(t, test.content, "scss")
			args := api.OnLoadArgs{Path: tmpFile}
			opts := newOptions()
			if test.processor != nil {
				opts.onSassLoadProcessors = append(opts.onSassLoadProcessors, test.processor)
			}
			build := &api.PluginBuild{InitialOptions: &api.BuildOptions{}}

			result, err := readSassSource(args, opts, build)

			if test.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
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

func TestReadSassSourceFileNotFound(t *testing.T) {
	args := api.OnLoadArgs{Path: "/nonexistent/file.scss"}
	opts := newOptions()
	build := &api.PluginBuild{InitialOptions: &api.BuildOptions{}}

	_, err := readSassSource(args, opts, build)
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read sass file") {
		t.Errorf("Expected 'failed to read sass file' error, got: %v", err)
	}
}

func TestCompileSass(t *testing.T) {
	tests := []struct {
		name        string
		mockConfig  *MockEngineConfig
		expectError bool
		expected    string
	}{
		{
			name: "successful_compilation",
			mockConfig: &MockEngineConfig{
				Sass: &MockSassConfig{
					CSS: ".btn { color: #333; }",
					Stats: &MockSassStatsConfig{
						Entry:         "/test/app.scss",
						IncludedFiles: []string{"/test/app.scss"},
					},
				},
			},
			expectError: false,
			expected:    ".btn { color: #333; }",
		},
		{
			name: "compilation_error",
			mockConfig: &MockEngineConfig{
				Sass: &MockSassConfig{CompileError: true},
			},
			expectError: true,
		},
		{
			name:        "invalid_result_type",
			mockConfig:  &MockEngineConfig{InvalidResult: true},
			expectError: true,
		},
		{
			name: "missing_css_in_result",
			mockConfig: &MockEngineConfig{
				ServiceResponses: map[string]interface{}{
					"sfc.sass.renderSync": map[string]interface{}{
						"map":   "",
						"stats": map[string]interface{}{},
						// Missing "css" field
					},
				},
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jsExec, err := newTestExecutor(test.mockConfig)
			if err != nil {
				t.Fatalf("Failed to create JS executor: %v", err)
			}
			if err := jsExec.Start(); err != nil {
				t.Fatalf("Failed to start JS executor: %v", err)
			}
			defer jsExec.Stop()

			result, err := compileSass("/test/app.scss", "$primary: #333;", jsExec)

			if test.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if result != test.expected {
					t.Errorf("Expected CSS '%s', got '%s'", test.expected, result)
				}
			}
		})
	}
}

// Integration tests

func TestSassCompilationWithMockEngine(t *testing.T) {
	tests := []struct {
		name        string
		mockConfig  *MockEngineConfig
		expectError string
		checkOutput bool
	}{
		{
			name: "successful_compilation",
			mockConfig: &MockEngineConfig{
				Sass: &MockSassConfig{
					CSS: ".button { color: #333; }",
					Stats: &MockSassStatsConfig{
						Entry:         "test.scss",
						IncludedFiles: []string{"test.scss"},
					},
				},
			},
			checkOutput: true,
		},
		{
			name: "compilation_error",
			mockConfig: &MockEngineConfig{
				Sass: &MockSassConfig{CompileError: true},
			},
			expectError: "sass compilation service failed",
		},
		{
			name:        "invalid_result",
			mockConfig:  &MockEngineConfig{InvalidResult: true},
			expectError: "invalid response from sass compilation service",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpFile := createTempSassFile(t, "$primary: #333;\n.button { color: $primary; }", "scss")
			entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
			entryContent := fmt.Sprintf(`import '%s';`, filepath.Base(tmpFile))
			err := os.WriteFile(entryFile, []byte(entryContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create entry file: %v", err)
			}
			defer os.Remove(entryFile)

			jsExec, err := newTestExecutor(test.mockConfig)
			if err != nil {
				t.Fatalf("Failed to create JS executor: %v", err)
			}
			if err := jsExec.Start(); err != nil {
				t.Fatalf("Failed to start JS executor: %v", err)
			}
			defer jsExec.Stop()

			result := buildSassTest(t, entryFile, jsExec)

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
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Expected successful build, got errors: %v", result.Errors)
				}
				if test.checkOutput {
					foundCSS := false
					for _, file := range result.OutputFiles {
						if strings.HasSuffix(file.Path, ".css") {
							foundCSS = true
							cssContent := string(file.Contents)
							if !strings.Contains(cssContent, ".button") {
								t.Errorf("Expected CSS to contain '.button', got: %s", cssContent)
							}
							break
						}
					}
					if !foundCSS {
						t.Error("Expected CSS output file, got none")
					}
				}
			}
		})
	}
}

func TestSassWithProcessors(t *testing.T) {
	tmpFile := createTempSassFile(t, "$color: red;\n.test { color: $color; }", "scss")
	entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
	entryContent := fmt.Sprintf(`import '%s';`, filepath.Base(tmpFile))
	err := os.WriteFile(entryFile, []byte(entryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}
	defer os.Remove(entryFile)

	jsExec, err := newTestExecutor(&MockEngineConfig{
		Sass: &MockSassConfig{CSS: ".test { color: blue; }"},
	})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	processorCalled := false
	processor := func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error) {
		processorCalled = true
		return "$color: blue;\n.test { color: $color; }", nil
	}

	vuePlugin := NewPlugin(
		WithJsExecutor(jsExec),
		WithOnSassLoadProcessor(processor),
	)

	result := api.Build(api.BuildOptions{
		EntryPoints:   []string{entryFile},
		Bundle:        true,
		Write:         false,
		LogLevel:      api.LogLevelError,
		Plugins:       []api.Plugin{vuePlugin},
		Outdir:        t.TempDir(),
		AbsWorkingDir: filepath.Dir(tmpFile),
	})

	if len(result.Errors) > 0 {
		t.Errorf("Expected successful build, got errors: %v", result.Errors)
	}
	if !processorCalled {
		t.Error("Expected Sass processor to be called")
	}
}

func TestSassFileExtensions(t *testing.T) {
	extensions := []string{"scss", "sass"}

	for _, ext := range extensions {
		t.Run("extension_"+ext, func(t *testing.T) {
			content := "$primary: #333;\n.button { color: $primary; }"
			if ext == "sass" {
				content = "$primary: #333\n.button\n  color: $primary"
			}

			tmpFile := createTempSassFile(t, content, ext)
			entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
			entryContent := fmt.Sprintf(`import '%s';`, filepath.Base(tmpFile))
			err := os.WriteFile(entryFile, []byte(entryContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create entry file: %v", err)
			}
			defer os.Remove(entryFile)

			jsExec, err := newTestExecutor(&MockEngineConfig{
				Sass: &MockSassConfig{CSS: ".button { color: #333; }"},
			})
			if err != nil {
				t.Fatalf("Failed to create JS executor: %v", err)
			}
			if err := jsExec.Start(); err != nil {
				t.Fatalf("Failed to start JS executor: %v", err)
			}
			defer jsExec.Stop()

			result := buildSassTest(t, entryFile, jsExec)

			if len(result.Errors) > 0 {
				t.Errorf("Expected successful build for .%s file, got errors: %v", ext, result.Errors)
			}
		})
	}
}

// Error handling tests

func TestSassResolveError(t *testing.T) {
	tmpFile := createTempSassFile(t, "$color: red;", "scss")
	entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
	entryContent := fmt.Sprintf(`import '%s';`, filepath.Base(tmpFile))
	err := os.WriteFile(entryFile, []byte(entryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}
	defer os.Remove(entryFile)

	jsExec, err := newTestExecutor(&MockEngineConfig{
		Sass: &MockSassConfig{CSS: ".test { color: red; }"},
	})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	result := buildSassTest(t, entryFile, jsExec, func(options *api.BuildOptions) {
		options.TsconfigRaw = `{invalid json}` // Trigger tsconfig parsing error
	})

	if len(result.Errors) == 0 {
		t.Error("Expected build errors due to invalid tsconfig, got none")
	}

	foundError := false
	for _, err := range result.Errors {
		if strings.Contains(err.Text, "invalid character") {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Errorf("Expected tsconfig parsing error, got: %v", result.Errors)
	}
}

func TestSassCompilationServiceError(t *testing.T) {
	jsExec, err := newTestExecutor(&MockEngineConfig{
		ExecuteError: fmt.Errorf("JS executor service failed"),
	})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	_, err = compileSass("/test/error.scss", "$color: red;", jsExec)
	if err == nil {
		t.Error("Expected error from JS executor service, got nil")
	}
	if !strings.Contains(err.Error(), "sass compilation service failed") {
		t.Errorf("Expected service error, got: %v", err)
	}
}

func TestSassLoadHandlerReadError(t *testing.T) {
	tmpDir := t.TempDir()
	entryFile := filepath.Join(tmpDir, "entry.js")
	nonExistentSassFile := filepath.Join(tmpDir, "nonexistent.scss")

	entryContent := fmt.Sprintf(`import '%s';`, filepath.Base(nonExistentSassFile))
	err := os.WriteFile(entryFile, []byte(entryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}

	jsExec, err := newTestExecutor(&MockEngineConfig{
		Sass: &MockSassConfig{CSS: ".test { color: red; }"},
	})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	result := buildSassTest(t, entryFile, jsExec)

	if len(result.Errors) == 0 {
		t.Error("Expected build errors due to readSassSource failure, got none")
	}

	foundReadError := false
	for _, err := range result.Errors {
		if strings.Contains(err.Text, "failed to read sass file") ||
			strings.Contains(err.Text, "no such file or directory") {
			foundReadError = true
			break
		}
	}
	if !foundReadError {
		t.Errorf("Expected error containing 'failed to read sass file', got: %v", result.Errors)
	}
}

func TestSassLoadHandlerProcessorError(t *testing.T) {
	tmpFile := createTempSassFile(t, "$color: red;\n.test { color: $color; }", "scss")
	entryFile := filepath.Join(filepath.Dir(tmpFile), "entry.js")
	entryContent := fmt.Sprintf(`import '%s';`, filepath.Base(tmpFile))
	err := os.WriteFile(entryFile, []byte(entryContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}
	defer os.Remove(entryFile)

	jsExec, err := newTestExecutor(&MockEngineConfig{
		Sass: &MockSassConfig{CSS: ".test { color: red; }"},
	})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	// Failing processor: returns error
	failingProcessor := func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error) {
		return "", fmt.Errorf("sass processor failed: custom error")
	}

	vuePlugin := NewPlugin(
		WithJsExecutor(jsExec),
		WithOnSassLoadProcessor(failingProcessor),
	)

	result := api.Build(api.BuildOptions{
		EntryPoints:   []string{entryFile},
		Bundle:        true,
		Write:         false,
		LogLevel:      api.LogLevelError,
		Plugins:       []api.Plugin{vuePlugin},
		Outdir:        t.TempDir(),
		AbsWorkingDir: filepath.Dir(tmpFile),
	})

	if len(result.Errors) == 0 {
		t.Error("Expected build errors due to processor failure, got none")
	}

	foundProcessorError := false
	for _, err := range result.Errors {
		if strings.Contains(err.Text, "sass processor failed: custom error") {
			foundProcessorError = true
			break
		}
	}
	if !foundProcessorError {
		t.Errorf("Expected error containing 'sass processor failed: custom error', got: %v", result.Errors)
	}
}

func TestReadSassSource_ParseTsconfigPathAliasError(t *testing.T) {
	tmpFile := createTempSassFile(t, "$primary: #333;\n.button { color: $primary; }", "scss")
	args := api.OnLoadArgs{Path: tmpFile}
	opts := newOptions()
	build := &api.PluginBuild{
		InitialOptions: &api.BuildOptions{
			TsconfigRaw: "{invalid json}",
		},
	}

	_, err := readSassSource(args, opts, build)
	if err == nil || !strings.Contains(err.Error(), "failed to parse TypeScript path aliases") {
		t.Errorf("Expected parse TypeScript path aliases error, got: %v", err)
	}
}
