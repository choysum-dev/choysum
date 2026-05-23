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

// Test helper functions
func createTestEntry(t *testing.T, tmpDir string) string {
	t.Helper()
	entryFile := filepath.Join(tmpDir, "main.js")
	if err := os.WriteFile(entryFile, []byte(`console.log('test');`), 0644); err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}
	return entryFile
}

func buildWithTestPlugin(t *testing.T, tmpDir string, options ...OptionFunc) api.BuildResult {
	t.Helper()

	entryFile := createTestEntry(t, tmpDir)

	// Ensure jsExecutor is provided
	hasJsExecutor := false
	for _, opt := range options {
		testOpts := newOptions()
		opt(testOpts)
		if testOpts.jsExecutor != nil {
			hasJsExecutor = true
			break
		}
	}

	if !hasJsExecutor {
		jsExec := createTestExecutor(t)
		options = append(options, WithJsExecutor(jsExec))
	}

	plugin := NewPlugin(options...)

	return api.Build(api.BuildOptions{
		EntryPoints:    []string{entryFile},
		Bundle:         true,
		Write:          false,
		LogLevel:       api.LogLevelError,
		Plugins:        []api.Plugin{plugin},
		Outdir:         tmpDir,
		AbsWorkingDir:  tmpDir,
		AllowOverwrite: true,
	})
}

// Unit tests for NewPlugin
func TestNewPlugin(t *testing.T) {
	t.Run("panic_without_jsExecutor", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				expectedMsg := "jsExecutor is required, please set it using WithJsExecutor()"
				if r.(string) != expectedMsg {
					t.Errorf("Expected panic message '%s', got '%v'", expectedMsg, r)
				}
			} else {
				t.Error("Expected panic when jsExecutor is nil, but no panic occurred")
			}
		}()
		NewPlugin()
	})

	t.Run("with_jsExecutor", func(t *testing.T) {
		jsExec := createTestExecutor(t)
		plugin := NewPlugin(WithJsExecutor(jsExec))

		if plugin.Name != "vue-plugin" {
			t.Errorf("Expected plugin name to be 'vue-plugin', got '%s'", plugin.Name)
		}
		if plugin.Setup == nil {
			t.Error("Expected plugin.Setup to be non-nil")
		}
	})

	t.Run("with_custom_name", func(t *testing.T) {
		jsExec := createTestExecutor(t)
		plugin := NewPlugin(
			WithJsExecutor(jsExec),
			WithName("custom-vue-plugin"),
		)

		if plugin.Name != "custom-vue-plugin" {
			t.Errorf("Expected plugin name to be 'custom-vue-plugin', got '%s'", plugin.Name)
		}
	})

	t.Run("with_all_options", func(t *testing.T) {
		jsExec := createTestExecutor(t)
		plugin := NewPlugin(
			WithJsExecutor(jsExec),
			WithName("test-plugin"),
			WithTemplateCompilerOptions(map[string]any{"test": "value"}),
			WithStylePreprocessorOptions(map[string]any{"sass": "compressed"}),
			WithOnStartProcessor(func(*api.BuildOptions) error { return nil }),
			WithOnEndProcessor(func(*api.BuildResult, *api.BuildOptions) error { return nil }),
			WithOnDisposeProcessor(func(*api.BuildOptions) {}),
		)

		if plugin.Name != "test-plugin" {
			t.Errorf("Expected plugin name to be 'test-plugin', got '%s'", plugin.Name)
		}
	})
}

// Test processor error handling
func TestProcessorErrors(t *testing.T) {
	t.Run("start_processor_error", func(t *testing.T) {
		tmpDir := t.TempDir()
		failingProcessor := func(*api.BuildOptions) error {
			return fmt.Errorf("start processor failed: custom error")
		}

		result := buildWithTestPlugin(t, tmpDir, WithOnStartProcessor(failingProcessor))

		if len(result.Errors) == 0 {
			t.Error("Expected build errors due to start processor failure, got none")
		}

		foundError := false
		for _, err := range result.Errors {
			if strings.Contains(err.Text, "start processor failed: custom error") {
				foundError = true
				break
			}
		}
		if !foundError {
			t.Errorf("Expected start processor error, got: %v", result.Errors)
		}
	})

	t.Run("end_processor_error", func(t *testing.T) {
		tmpDir := t.TempDir()
		failingProcessor := func(*api.BuildResult, *api.BuildOptions) error {
			return fmt.Errorf("end processor failed: custom error")
		}

		result := buildWithTestPlugin(t, tmpDir, WithOnEndProcessor(failingProcessor))

		if len(result.Errors) == 0 {
			t.Error("Expected build errors due to end processor failure, got none")
		}

		foundError := false
		for _, err := range result.Errors {
			if strings.Contains(err.Text, "end processor failed: custom error") {
				foundError = true
				break
			}
		}
		if !foundError {
			t.Errorf("Expected end processor error, got: %v", result.Errors)
		}
	})
}

// Test processor success scenarios
func TestProcessorSuccess(t *testing.T) {
	t.Run("start_processor_success", func(t *testing.T) {
		tmpDir := t.TempDir()
		startProcessorCalled := false
		successfulProcessor := func(buildOptions *api.BuildOptions) error {
			startProcessorCalled = true
			if buildOptions.Define == nil {
				buildOptions.Define = make(map[string]string)
			}
			buildOptions.Define["TEST_START_PROCESSOR"] = "true"
			return nil
		}

		result := buildWithTestPlugin(t, tmpDir, WithOnStartProcessor(successfulProcessor))

		if len(result.Errors) > 0 {
			t.Errorf("Expected successful build, got errors: %v", result.Errors)
		}
		if !startProcessorCalled {
			t.Error("Expected start processor to be called")
		}
	})

	t.Run("end_processor_success", func(t *testing.T) {
		tmpDir := t.TempDir()
		endProcessorCalled := false
		successfulProcessor := func(result *api.BuildResult, buildOptions *api.BuildOptions) error {
			endProcessorCalled = true
			if result == nil {
				return fmt.Errorf("expected non-nil build result")
			}
			return nil
		}

		result := buildWithTestPlugin(t, tmpDir, WithOnEndProcessor(successfulProcessor))

		if len(result.Errors) > 0 {
			t.Errorf("Expected successful build, got errors: %v", result.Errors)
		}
		if !endProcessorCalled {
			t.Error("Expected end processor to be called")
		}
	})

	t.Run("dispose_processor_registration", func(t *testing.T) {
		tmpDir := t.TempDir()
		disposeProcessor := func(buildOptions *api.BuildOptions) {
			if buildOptions == nil {
				return
			}
		}

		result := buildWithTestPlugin(t, tmpDir, WithOnDisposeProcessor(disposeProcessor))

		if len(result.Errors) > 0 {
			t.Errorf("Expected successful build, got errors: %v", result.Errors)
		}
		// Note: OnDispose is called internally by esbuild during context disposal
		t.Log("Dispose processor registered successfully")
	})
}

// Test multiple processors
func TestMultipleProcessors(t *testing.T) {
	tmpDir := t.TempDir()

	var callOrder []string

	processors := []OptionFunc{
		WithOnStartProcessor(func(*api.BuildOptions) error {
			callOrder = append(callOrder, "start1")
			return nil
		}),
		WithOnStartProcessor(func(*api.BuildOptions) error {
			callOrder = append(callOrder, "start2")
			return nil
		}),
		WithOnEndProcessor(func(*api.BuildResult, *api.BuildOptions) error {
			callOrder = append(callOrder, "end1")
			return nil
		}),
		WithOnEndProcessor(func(*api.BuildResult, *api.BuildOptions) error {
			callOrder = append(callOrder, "end2")
			return nil
		}),
	}

	result := buildWithTestPlugin(t, tmpDir, processors...)

	if len(result.Errors) > 0 {
		t.Errorf("Expected successful build, got errors: %v", result.Errors)
	}

	expectedOrder := []string{"start1", "start2", "end1", "end2"}
	if len(callOrder) != len(expectedOrder) {
		t.Errorf("Expected %d processor calls, got %d: %v", len(expectedOrder), len(callOrder), callOrder)
	}

	for i, expected := range expectedOrder {
		if i >= len(callOrder) || callOrder[i] != expected {
			t.Errorf("Expected processor call order %v, got %v", expectedOrder, callOrder)
			break
		}
	}
}

// Test options normalization
func TestOptionsNormalization(t *testing.T) {
	tmpDir := t.TempDir()
	optionsNormalized := false

	startProcessor := func(buildOptions *api.BuildOptions) error {
		// Check if options were normalized
		if buildOptions.Define != nil {
			if _, exists := buildOptions.Define["import.meta.env"]; exists {
				optionsNormalized = true
			}
		}
		if buildOptions.Metafile {
			optionsNormalized = true
		}
		return nil
	}

	result := buildWithTestPlugin(t, tmpDir, WithOnStartProcessor(startProcessor))

	if len(result.Errors) > 0 {
		t.Errorf("Expected successful build, got errors: %v", result.Errors)
	}
	if !optionsNormalized {
		t.Error("Expected esbuild options to be normalized")
	}
}

// Integration test with Vue file
func TestVueIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	vueFile := filepath.Join(tmpDir, "test.vue")
	entryFile := filepath.Join(tmpDir, "main.js")

	// Create Vue file
	vueContent := `
<template>
  <div class="test">Hello Vue Plugin!</div>
</template>
<script>
export default { name: 'TestComponent' }
</script>
<style scoped>
.test { color: red; }
</style>`

	if err := os.WriteFile(vueFile, []byte(vueContent), 0644); err != nil {
		t.Fatalf("Failed to create Vue file: %v", err)
	}

	// Create entry file
	entryContent := `import TestComponent from './test.vue'; console.log(TestComponent);`
	if err := os.WriteFile(entryFile, []byte(entryContent), 0644); err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}

	// Create JS executor with mock engine
	jsExec, err := newTestExecutor(&MockEngineConfig{
		Script:   &MockScriptConfig{Content: "export default { name: 'TestComponent' }", Lang: "js"},
		Template: &MockTemplateConfig{Code: "function render() { return h('div', 'Hello Vue Plugin!'); }"},
		Styles: []*MockStyleConfig{
			{Code: ".test { color: red; }", Scoped: true},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	plugin := NewPlugin(WithJsExecutor(jsExec))

	result := api.Build(api.BuildOptions{
		EntryPoints:    []string{entryFile},
		Bundle:         true,
		Write:          false,
		LogLevel:       api.LogLevelError,
		Plugins:        []api.Plugin{plugin},
		Outdir:         tmpDir,
		AbsWorkingDir:  tmpDir,
		AllowOverwrite: true,
	})

	if len(result.Errors) > 0 {
		t.Errorf("Expected successful build, got errors: %v", result.Errors)
	}

	if len(result.OutputFiles) == 0 {
		t.Fatal("Expected output files, got none")
	}

	mainOutput := string(result.OutputFiles[0].Contents)
	if !strings.Contains(mainOutput, "TestComponent") {
		t.Error("Expected output to contain Vue component content")
	}
}

// Test plugin setup handlers registration
func TestPluginSetupHandlers(t *testing.T) {
	jsExec := createTestExecutor(t)
	plugin := NewPlugin(WithJsExecutor(jsExec))

	if plugin.Setup == nil {
		t.Fatal("Expected Setup function to be non-nil")
	}

	// Test that Setup can be called without errors
	var onStartCallback func() (api.OnStartResult, error)
	var onEndCallback func(*api.BuildResult) (api.OnEndResult, error)
	var onDisposeCallback func()

	mockBuild := api.PluginBuild{
		InitialOptions: &api.BuildOptions{},
		OnStart: func(callback func() (api.OnStartResult, error)) {
			onStartCallback = callback
		},
		OnEnd: func(callback func(*api.BuildResult) (api.OnEndResult, error)) {
			onEndCallback = callback
		},
		OnDispose: func(callback func()) {
			onDisposeCallback = callback
		},
		OnResolve: func(api.OnResolveOptions, func(api.OnResolveArgs) (api.OnResolveResult, error)) {
			// Mock OnResolve registration
		},
		OnLoad: func(api.OnLoadOptions, func(api.OnLoadArgs) (api.OnLoadResult, error)) {
			// Mock OnLoad registration
		},
	}

	// Call Setup
	plugin.Setup(mockBuild)

	// Verify callbacks were registered
	if onStartCallback == nil {
		t.Error("Expected OnStart callback to be registered")
	}
	if onEndCallback == nil {
		t.Error("Expected OnEnd callback to be registered")
	}
	if onDisposeCallback == nil {
		t.Error("Expected OnDispose callback to be registered")
	}

	// Test callbacks execution
	if onStartCallback != nil {
		if _, err := onStartCallback(); err != nil {
			t.Errorf("Expected OnStart callback to succeed, got error: %v", err)
		}
	}

	if onEndCallback != nil {
		if _, err := onEndCallback(&api.BuildResult{}); err != nil {
			t.Errorf("Expected OnEnd callback to succeed, got error: %v", err)
		}
	}

	if onDisposeCallback != nil {
		onDisposeCallback() // Should not panic
	}
}
