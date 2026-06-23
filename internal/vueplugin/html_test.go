// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antchfx/htmlquery"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
)

// Test helper functions
func createTestFiles(t *testing.T, tmpDir string) (entryFile, htmlSourceFile, htmlOutFile string) {
	t.Helper()

	entryFile = filepath.Join(tmpDir, "main.js")
	htmlSourceFile = filepath.Join(tmpDir, "index.html")
	htmlOutFile = filepath.Join(tmpDir, "dist", "index.html")

	// Create entry file
	entryContent := `console.log('test');`
	if err := os.WriteFile(entryFile, []byte(entryContent), 0644); err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}

	// Create HTML source file
	htmlContent := `<!DOCTYPE html><html><head><title>Test</title></head><body></body></html>`
	if err := os.WriteFile(htmlSourceFile, []byte(htmlContent), 0644); err != nil {
		t.Fatalf("Failed to create HTML file: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(filepath.Dir(htmlOutFile), 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	return entryFile, htmlSourceFile, htmlOutFile
}

func createTestExecutor(t *testing.T) jsexecutor.JsExecutor {
	t.Helper()

	jsExec, err := newTestExecutor(&MockEngineConfig{})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	t.Cleanup(func() { jsExec.Stop() })

	return jsExec
}

func buildWithPlugin(t *testing.T, entryFile string, jsExec jsexecutor.JsExecutor, htmlOptions IndexHtmlOptions, additionalOptions ...func(*api.BuildOptions)) api.BuildResult {
	t.Helper()

	vuePlugin := NewPlugin(
		WithJsExecutor(jsExec),
		WithIndexHtmlOptions(htmlOptions),
	)

	// Use a separate output directory to avoid overwriting input files
	outDir := filepath.Join(filepath.Dir(entryFile), "dist")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	buildOptions := api.BuildOptions{
		EntryPoints:    []string{entryFile},
		Bundle:         true,
		Write:          true,
		LogLevel:       api.LogLevelError,
		Plugins:        []api.Plugin{vuePlugin},
		Outdir:         outDir,
		AbsWorkingDir:  filepath.Dir(entryFile),
		Metafile:       true,
		AllowOverwrite: true, // Allow overwriting files
	}

	// Apply additional options
	for _, fn := range additionalOptions {
		fn(&buildOptions)
	}

	return api.Build(buildOptions)
}

// Unit tests for HtmlProcessor
func TestNewHtmlProcessor(t *testing.T) {
	tests := []struct {
		name    string
		options HtmlProcessorOptions
	}{
		{
			name:    "empty_options",
			options: HtmlProcessorOptions{},
		},
		{
			name: "custom_script_builder",
			options: HtmlProcessorOptions{
				ScriptAttrBuilder: func(filename string, htmlSourceFile string) []html.Attribute {
					return []html.Attribute{{Key: "src", Val: "custom-" + filename}}
				},
			},
		},
		{
			name: "custom_css_builder",
			options: HtmlProcessorOptions{
				CssAttrBuilder: func(filename string, htmlSourceFile string) []html.Attribute {
					return []html.Attribute{{Key: "href", Val: "custom-" + filename}}
				},
			},
		},
		{
			name: "with_path_prefix",
			options: HtmlProcessorOptions{
				PathPrefix: "/static/assets",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			processor := DefaultHtmlProcessor(&test.options)
			if processor == nil {
				t.Fatal("Expected non-nil processor")
			}

			// Test processor execution
			htmlContent := `<html><head></head><body></body></html>`
			doc, err := html.Parse(strings.NewReader(htmlContent))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			opts := newOptions()
			opts.IndexHtmlOptions.OutFile = "/test/index.html"

			build := &api.PluginBuild{
				InitialOptions: &api.BuildOptions{
					EntryPoints: []string{"/test/entry.js"},
				},
			}

			result := &api.BuildResult{
				OutputFiles: []api.OutputFile{
					{Path: "/test/entry-abc123.js"},
					{Path: "/test/entry-def456.css"},
				},
			}

			if err := processor(doc, result, opts, build); err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			// Additional check for PathPrefix
			if test.name == "with_path_prefix" {
				var buf bytes.Buffer
				if err := html.Render(&buf, doc); err != nil {
					t.Fatalf("Failed to render HTML: %v", err)
				}
				htmlResult := buf.String()
				if !strings.Contains(htmlResult, `src="/static/assets/entry-abc123.js"`) {
					t.Error("Expected script tag src to contain path prefix")
				}
				if !strings.Contains(htmlResult, `href="/static/assets/entry-def456.css"`) {
					t.Error("Expected link tag href to contain path prefix")
				}
			}
		})
	}
}

func TestHtmlProcessorFileFiltering(t *testing.T) {
	processor := DefaultHtmlProcessor(nil)

	htmlContent := `<html><head></head><body></body></html>`
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	opts := newOptions()
	opts.IndexHtmlOptions.OutFile = "/test/index.html"

	build := &api.PluginBuild{
		InitialOptions: &api.BuildOptions{
			EntryPoints: []string{"/test/entry.js"},
		},
	}

	result := &api.BuildResult{
		OutputFiles: []api.OutputFile{
			{Path: "/test/entry-abc123.js"},  // Should be included
			{Path: "/test/entry-def456.css"}, // Should be included
			{Path: "/test/other-xyz789.js"},  // Should be skipped
			{Path: "/test/another-uvw.css"},  // Should be skipped
		},
	}

	if err := processor(doc, result, opts, build); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}

	htmlResult := buf.String()
	if !strings.Contains(htmlResult, "entry-abc123.js") {
		t.Error("Expected HTML to contain entry-abc123.js")
	}
	if !strings.Contains(htmlResult, "entry-def456.css") {
		t.Error("Expected HTML to contain entry-def456.css")
	}
	if strings.Contains(htmlResult, "other-xyz789.js") {
		t.Error("Expected HTML to not contain other-xyz789.js")
	}
}

func TestHtmlProcessorUsesMetafileWhenOutputFilesEmpty(t *testing.T) {
	processor := DefaultHtmlProcessor(nil)

	htmlContent := `<html><head></head><body></body></html>`
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	opts := newOptions()
	opts.IndexHtmlOptions.OutFile = "/test/dist/index.html"

	build := &api.PluginBuild{
		InitialOptions: &api.BuildOptions{
			EntryPoints:    []string{"/test/src/main.ts"},
			Outfile:        "/test/dist/index.js",
			AbsWorkingDir:  "/test",
			AllowOverwrite: true,
		},
	}

	result := &api.BuildResult{
		Metafile: `{"outputs":{"dist/index.js":{"entryPoint":"src/main.ts"},"dist/index.css":{"bytes":128},"dist/chunk-abc.js":{"bytes":64}}}`,
	}

	if err := processor(doc, result, opts, build); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}

	htmlResult := buf.String()
	if !strings.Contains(htmlResult, `src="index.js"`) {
		t.Error("Expected HTML to contain script tag for index.js")
	}
	if !strings.Contains(htmlResult, `href="index.css"`) {
		t.Error("Expected HTML to contain link tag for index.css")
	}
	if strings.Contains(htmlResult, "chunk-abc.js") {
		t.Error("Expected HTML to not contain non-entry chunk script")
	}
}

func TestHtmlProcessorRemoveTagXPaths(t *testing.T) {
	processor := DefaultHtmlProcessor(nil)

	htmlContent := `<html><head><title>Test</title><meta name="remove-me" content="test"></head><body></body></html>`
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	opts := newOptions()
	opts.IndexHtmlOptions.OutFile = "/test/index.html"
	opts.IndexHtmlOptions.RemoveTagXPaths = []string{"//meta[@name='remove-me']"}

	build := &api.PluginBuild{
		InitialOptions: &api.BuildOptions{
			EntryPoints: []string{"/test/entry.js"},
		},
	}

	result := &api.BuildResult{OutputFiles: []api.OutputFile{}}

	if err := processor(doc, result, opts, build); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}

	htmlResult := buf.String()
	if strings.Contains(htmlResult, `name="remove-me"`) {
		t.Error("Expected meta tag to be removed")
	}
	if !strings.Contains(htmlResult, "<title>Test</title>") {
		t.Error("Expected title tag to remain")
	}
}

// Unit tests for detectAndConvertToUTF8
func TestDetectAndConvertToUTF8(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "utf8_content",
			content:  "<html><head><title>UTF-8 Test</title></head></html>",
			expected: "<html><head><title>UTF-8 Test</title></head></html>",
		},
		{
			name:     "empty_content",
			content:  "",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := strings.NewReader(test.content)
			utf8Reader, err := detectAndConvertToUTF8(reader)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			result, err := io.ReadAll(utf8Reader)
			if err != nil {
				t.Errorf("Expected no error reading, got: %v", err)
			}

			if string(result) != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, string(result))
			}
		})
	}
}

func TestDetectAndConvertToUTF8ReadError(t *testing.T) {
	failingReader := &failingReader{}
	_, err := detectAndConvertToUTF8(failingReader)
	if err == nil {
		t.Error("Expected error from failing reader, got nil")
	}
}

// failingReader is a test helper that always returns an error
type failingReader struct{}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

// Integration tests
func TestHtmlHandlerWithVueFile(t *testing.T) {
	tmpDir := t.TempDir()
	vueFile := filepath.Join(tmpDir, "App.vue")
	entryFile := filepath.Join(tmpDir, "main.js")
	htmlSourceFile := filepath.Join(tmpDir, "index.html")
	htmlOutFile := filepath.Join(tmpDir, "dist", "index.html")

	// Create Vue file
	vueContent := `
<template>
  <div>Hello World</div>
</template>
<script>
export default {
  name: 'App'
}
</script>
<style>
.app { color: red; }
</style>
`
	if err := os.WriteFile(vueFile, []byte(vueContent), 0644); err != nil {
		t.Fatalf("Failed to create Vue file: %v", err)
	}

	// Create entry file
	entryContent := `import App from './App.vue';`
	if err := os.WriteFile(entryFile, []byte(entryContent), 0644); err != nil {
		t.Fatalf("Failed to create entry file: %v", err)
	}

	// Create HTML source file
	htmlContent := `<!DOCTYPE html>
<html>
<head>
    <title>Test App</title>
</head>
<body>
    <div id="app"></div>
</body>
</html>`
	if err := os.WriteFile(htmlSourceFile, []byte(htmlContent), 0644); err != nil {
		t.Fatalf("Failed to create HTML file: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(filepath.Dir(htmlOutFile), 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Create JS executor with mock engine
	jsExec, err := newTestExecutor(&MockEngineConfig{
		Script:   &MockScriptConfig{Content: "export default { name: 'App' }", Lang: "js"},
		Template: &MockTemplateConfig{Code: "function render() { return h('div', 'Hello World'); }"},
		Styles: []*MockStyleConfig{
			{Code: ".app { color: red; }", Scoped: false},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create JS executor: %v", err)
	}
	if err := jsExec.Start(); err != nil {
		t.Fatalf("Failed to start JS executor: %v", err)
	}
	defer jsExec.Stop()

	// Build with HTML options
	result := buildWithPlugin(t, entryFile, jsExec, IndexHtmlOptions{
		SourceFile: htmlSourceFile,
		OutFile:    htmlOutFile,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("Expected successful build, got errors: %v", result.Errors)
	}

	// Verify HTML file was created
	if _, err := os.Stat(htmlOutFile); os.IsNotExist(err) {
		t.Fatal("Expected HTML output file to be created")
	}

	// Verify HTML content
	htmlBytes, err := os.ReadFile(htmlOutFile)
	if err != nil {
		t.Fatalf("Failed to read HTML output: %v", err)
	}

	htmlResult := string(htmlBytes)
	if !strings.Contains(htmlResult, "<script") {
		t.Error("Expected HTML to contain script tag")
	}
}

func TestHtmlHandlerWithCustomProcessor(t *testing.T) {
	tmpDir := t.TempDir()
	entryFile, htmlSourceFile, htmlOutFile := createTestFiles(t, tmpDir)
	jsExec := createTestExecutor(t)

	// Create custom processor
	customProcessorCalled := false
	customProcessor := func(doc *html.Node, result *api.BuildResult, opts *Options, build *api.PluginBuild) error {
		customProcessorCalled = true
		headNode := htmlquery.FindOne(doc, "//head")
		if headNode != nil {
			comment := &html.Node{
				Type: html.CommentNode,
				Data: " Custom processor was here ",
			}
			headNode.AppendChild(comment)
		}
		return nil
	}

	// Build with custom processor
	result := buildWithPlugin(t, entryFile, jsExec, IndexHtmlOptions{
		SourceFile:          htmlSourceFile,
		OutFile:             htmlOutFile,
		IndexHtmlProcessors: []IndexHtmlProcessor{customProcessor},
	})

	if len(result.Errors) > 0 {
		t.Errorf("Expected successful build, got errors: %v", result.Errors)
	}

	if !customProcessorCalled {
		t.Error("Expected custom processor to be called")
	}

	// Verify custom comment was added
	htmlBytes, err := os.ReadFile(htmlOutFile)
	if err != nil {
		t.Errorf("Failed to read HTML output: %v", err)
	} else {
		htmlResult := string(htmlBytes)
		if !strings.Contains(htmlResult, "Custom processor was here") {
			t.Error("Expected HTML to contain custom comment")
		}
	}
}

// Error handling tests
func TestHtmlHandlerErrorConditions(t *testing.T) {
	tests := []struct {
		name          string
		setupOptions  func(*Options, string)
		setupBuild    func(*api.BuildOptions)
		expectError   bool
		errorContains string
	}{
		{
			name: "no_source_file",
			setupOptions: func(opts *Options, tmpDir string) {
				opts.IndexHtmlOptions.SourceFile = ""
			},
			setupBuild: func(buildOpts *api.BuildOptions) {
				buildOpts.Write = true
				buildOpts.Metafile = true
			},
			expectError: false, // Should skip processing
		},
		{
			name: "write_false",
			setupOptions: func(opts *Options, tmpDir string) {
				opts.IndexHtmlOptions.SourceFile = "/test/index.html"
			},
			setupBuild: func(buildOpts *api.BuildOptions) {
				buildOpts.Write = false
				buildOpts.Metafile = true
			},
			expectError: false, // Should skip processing
		},
		{
			name: "no_out_file",
			setupOptions: func(opts *Options, tmpDir string) {
				// Create a real HTML file
				htmlSourceFile := filepath.Join(tmpDir, "index.html")
				htmlContent := `<!DOCTYPE html><html><head><title>Test</title></head><body></body></html>`
				if err := os.WriteFile(htmlSourceFile, []byte(htmlContent), 0644); err != nil {
					panic(fmt.Sprintf("Failed to create HTML file: %v", err))
				}
				opts.IndexHtmlOptions.SourceFile = htmlSourceFile
				opts.IndexHtmlOptions.OutFile = "" // This should trigger the outFile error
			},
			setupBuild: func(buildOpts *api.BuildOptions) {
				buildOpts.Write = true
				buildOpts.Metafile = true
			},
			expectError:   true,
			errorContains: "outFile or sourceFile is nil",
		},
		{
			name: "nonexistent_source_file",
			setupOptions: func(opts *Options, tmpDir string) {
				opts.IndexHtmlOptions.SourceFile = "/nonexistent/index.html"
				// Create the output directory to avoid directory creation errors
				distDir := filepath.Join(tmpDir, "dist")
				if err := os.MkdirAll(distDir, 0755); err != nil {
					panic(fmt.Sprintf("Failed to create dist directory: %v", err))
				}
				opts.IndexHtmlOptions.OutFile = filepath.Join(distDir, "index.html")
			},
			setupBuild: func(buildOpts *api.BuildOptions) {
				buildOpts.Write = true
				buildOpts.Metafile = true
			},
			expectError:   true,
			errorContains: "failed to open source file",
		},
		{
			name: "write_to_nonexistent_directory",
			setupOptions: func(opts *Options, tmpDir string) {
				// Create a real HTML file
				htmlSourceFile := filepath.Join(tmpDir, "index.html")
				htmlContent := `<!DOCTYPE html><html><head><title>Test</title></head><body></body></html>`
				if err := os.WriteFile(htmlSourceFile, []byte(htmlContent), 0644); err != nil {
					panic(fmt.Sprintf("Failed to create HTML file: %v", err))
				}
				opts.IndexHtmlOptions.SourceFile = htmlSourceFile
				// Set output file to a non-existent directory
				opts.IndexHtmlOptions.OutFile = filepath.Join(tmpDir, "nonexistent", "index.html")
			},
			setupBuild: func(buildOpts *api.BuildOptions) {
				buildOpts.Write = true
				buildOpts.Metafile = true
			},
			expectError:   true,
			errorContains: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create temporary directory and files
			tmpDir := t.TempDir()
			entryFile := filepath.Join(tmpDir, "main.js")

			// Create entry file
			entryContent := `console.log('test');`
			err := os.WriteFile(entryFile, []byte(entryContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create entry file: %v", err)
			}

			// Create JS executor
			jsExec := createTestExecutor(t)

			// Create plugin with test options
			opts := newOptions()
			test.setupOptions(opts, tmpDir)

			vuePlugin := NewPlugin(
				WithJsExecutor(jsExec),
				WithIndexHtmlOptions(opts.IndexHtmlOptions),
			)

			// Use separate output directory to avoid overwriting input files
			outDir := filepath.Join(tmpDir, "build_output")
			if err := os.MkdirAll(outDir, 0755); err != nil {
				t.Fatalf("Failed to create output directory: %v", err)
			}

			// Build with test options
			buildOptions := api.BuildOptions{
				EntryPoints:    []string{entryFile},
				Bundle:         true,
				LogLevel:       api.LogLevelError,
				Plugins:        []api.Plugin{vuePlugin},
				Outdir:         outDir,
				AbsWorkingDir:  tmpDir,
				AllowOverwrite: true,
			}
			test.setupBuild(&buildOptions)

			result := api.Build(buildOptions)

			if test.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected build errors, got none")
				}
				if test.errorContains != "" {
					found := false
					for _, err := range result.Errors {
						if strings.Contains(err.Text, test.errorContains) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing '%s', got: %v", test.errorContains, result.Errors)
					}
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", result.Errors)
				}
			}
		})
	}
}

func TestHtmlHandlerProcessorError(t *testing.T) {
	tmpDir := t.TempDir()
	entryFile, htmlSourceFile, htmlOutFile := createTestFiles(t, tmpDir)
	jsExec := createTestExecutor(t)

	// Create failing processor
	failingProcessor := func(doc *html.Node, result *api.BuildResult, opts *Options, build *api.PluginBuild) error {
		return fmt.Errorf("processor failed: custom error")
	}

	result := buildWithPlugin(t, entryFile, jsExec, IndexHtmlOptions{
		SourceFile:          htmlSourceFile,
		OutFile:             htmlOutFile,
		IndexHtmlProcessors: []IndexHtmlProcessor{failingProcessor},
	})

	if len(result.Errors) == 0 {
		t.Error("Expected build errors due to processor failure, got none")
	}

	foundProcessorError := false
	for _, err := range result.Errors {
		if strings.Contains(err.Text, "processor failed: custom error") {
			foundProcessorError = true
			break
		}
	}
	if !foundProcessorError {
		t.Errorf("Expected processor error, got: %v", result.Errors)
	}
}

func TestHtmlHandlerReadOnlyOutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	entryFile, htmlSourceFile, htmlOutFile := createTestFiles(t, tmpDir)
	jsExec := createTestExecutor(t)

	// Create read-only output file
	if err := os.WriteFile(htmlOutFile, []byte("existing content"), 0444); err != nil {
		t.Fatalf("Failed to create read-only HTML file: %v", err)
	}
	t.Cleanup(func() {
		os.Chmod(htmlOutFile, 0644)
		os.Remove(htmlOutFile)
	})

	result := buildWithPlugin(t, entryFile, jsExec, IndexHtmlOptions{
		SourceFile: htmlSourceFile,
		OutFile:    htmlOutFile,
	})

	if len(result.Errors) == 0 {
		t.Error("Expected build errors due to write permission, got none")
	}
}

func TestSetupHtmlHandlerMetafileNil(t *testing.T) {
	tmpDir := t.TempDir()
	_, htmlSourceFile, htmlOutFile := createTestFiles(t, tmpDir)

	opts := newOptions()
	opts.IndexHtmlOptions.SourceFile = htmlSourceFile
	opts.IndexHtmlOptions.OutFile = htmlOutFile

	var onEndCallback func(result *api.BuildResult) (api.OnEndResult, error)
	build := &api.PluginBuild{
		InitialOptions: &api.BuildOptions{Write: true},
		OnEnd: func(callback func(result *api.BuildResult) (api.OnEndResult, error)) {
			onEndCallback = callback
		},
	}

	setupHtmlHandler(opts, build)

	result := &api.BuildResult{Metafile: ""}
	_, err := onEndCallback(result)
	if err == nil {
		t.Error("Expected error for empty metafile, got none")
	}
	if !strings.Contains(err.Error(), "metafile is nil") {
		t.Errorf("Expected error containing 'metafile is nil', got: %v", err)
	}
}

func TestSetupHtmlHandlerDetectAndConvertToUTF8Error(t *testing.T) {
	tmpDir := t.TempDir()
	entryFile, htmlSourceFile, htmlOutFile := createTestFiles(t, tmpDir)
	jsExec := createTestExecutor(t)

	// Replace HTML file with directory to trigger io.ReadAll error
	if err := os.Remove(htmlSourceFile); err != nil {
		t.Fatalf("Failed to remove HTML file: %v", err)
	}
	if err := os.Mkdir(htmlSourceFile, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	result := buildWithPlugin(t, entryFile, jsExec, IndexHtmlOptions{
		SourceFile: htmlSourceFile,
		OutFile:    htmlOutFile,
	})

	if len(result.Errors) == 0 {
		t.Error("Expected build errors due to UTF-8 conversion failure, got none")
	}

	foundUTF8Error := false
	for _, err := range result.Errors {
		if strings.Contains(err.Text, "failed to convert source file to UTF-8") {
			foundUTF8Error = true
			break
		}
	}
	if !foundUTF8Error {
		t.Errorf("Expected UTF-8 conversion error, got: %v", result.Errors)
	}
}

func TestSetupHtmlHandlerRenderErrorWithErrorNode(t *testing.T) {
	tmpDir := t.TempDir()
	entryFile, htmlSourceFile, htmlOutFile := createTestFiles(t, tmpDir)
	jsExec := createTestExecutor(t)

	// Create processor that adds ErrorNode
	errorNodeProcessor := func(doc *html.Node, result *api.BuildResult, opts *Options, build *api.PluginBuild) error {
		headNode := htmlquery.FindOne(doc, "//head")
		if headNode == nil {
			return fmt.Errorf("head node not found")
		}

		errorNode := &html.Node{
			Type: html.ErrorNode,
			Data: "error-node-data",
		}
		headNode.AppendChild(errorNode)
		return nil
	}

	result := buildWithPlugin(t, entryFile, jsExec, IndexHtmlOptions{
		SourceFile:          htmlSourceFile,
		OutFile:             htmlOutFile,
		IndexHtmlProcessors: []IndexHtmlProcessor{errorNodeProcessor},
	})

	if len(result.Errors) == 0 {
		t.Error("Expected build errors due to ErrorNode rendering, got none")
	}

	foundRenderError := false
	for _, buildErr := range result.Errors {
		if strings.Contains(buildErr.Text, "cannot render an ErrorNode node") {
			foundRenderError = true
			break
		}
	}
	if !foundRenderError {
		t.Errorf("Expected error containing 'cannot render an ErrorNode node', got: %v", result.Errors)
	}
}

// Direct unit tests for html.Render error conditions
func TestHtmlRenderErrorConditions(t *testing.T) {
	var buf bytes.Buffer

	t.Run("ErrorNode", func(t *testing.T) {
		buf.Reset()
		errorNode := &html.Node{
			Type: html.ErrorNode,
			Data: "error",
		}
		err := html.Render(&buf, errorNode)
		if err == nil {
			t.Error("Expected error when rendering ErrorNode, got nil")
		}
		if !strings.Contains(err.Error(), "cannot render an ErrorNode node") {
			t.Errorf("Expected error message about ErrorNode, got: %v", err)
		}
	})

	t.Run("VoidElementWithChildren", func(t *testing.T) {
		buf.Reset()
		brNode := &html.Node{
			Type: html.ElementNode,
			Data: "br",
		}
		childNode := &html.Node{
			Type: html.TextNode,
			Data: "child",
		}
		brNode.AppendChild(childNode)

		err := html.Render(&buf, brNode)
		if err == nil {
			t.Error("Expected error when rendering void element with children, got nil")
		}
		if !strings.Contains(err.Error(), "void element") || !strings.Contains(err.Error(), "has child nodes") {
			t.Errorf("Expected error message about void element with children, got: %v", err)
		}
	})

	t.Run("UnknownNodeType", func(t *testing.T) {
		buf.Reset()
		unknownNode := &html.Node{
			Type: html.NodeType(999),
			Data: "unknown",
		}
		err := html.Render(&buf, unknownNode)
		if err == nil {
			t.Error("Expected error when rendering unknown node type, got nil")
		}
		if !strings.Contains(err.Error(), "unknown node type") {
			t.Errorf("Expected error message about unknown node type, got: %v", err)
		}
	})
}
