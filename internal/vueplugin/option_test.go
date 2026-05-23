// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/jsexecutortest"
	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
)

// TestWithName verifies that WithName sets the plugin name correctly.
func TestWithName(t *testing.T) {
	opts := newOptions()
	WithName("test-plugin")(opts)
	if opts.Name != "test-plugin" {
		t.Errorf("Expected name to be 'test-plugin', got %s", opts.Name)
	}
}

// TestWithTemplateCompilerOptions verifies that WithTemplateCompilerOptions sets template options.
func TestWithTemplateCompilerOptions(t *testing.T) {
	opts := newOptions()
	templateOpts := map[string]any{"test": "value"}
	WithTemplateCompilerOptions(templateOpts)(opts)
	if opts.TemplateCompilerOptions["test"] != "value" {
		t.Errorf("Expected templateCompilerOptions to contain test option")
	}
}

// TestWithStylePreprocessorOptions verifies that WithStylePreprocessorOptions sets style options.
func TestWithStylePreprocessorOptions(t *testing.T) {
	opts := newOptions()
	styleOpts := map[string]any{"sass": "compressed"}
	WithStylePreprocessorOptions(styleOpts)(opts)
	if opts.StylePreprocessorOptions["sass"] != "compressed" {
		t.Errorf("Expected stylePreprocessorOptions to contain sass option")
	}
}

// TestWithOnStartProcessor verifies that WithOnStartProcessor adds a processor.
func TestWithOnStartProcessor(t *testing.T) {
	opts := newOptions()
	processor := func(buildOptions *api.BuildOptions) error { return nil }
	WithOnStartProcessor(processor)(opts)
	if len(opts.onStartProcessors) != 1 {
		t.Errorf("Expected 1 start processor, got %d", len(opts.onStartProcessors))
	}
}

// TestWithOnVueLoadProcessor verifies that WithOnVueLoadProcessor adds a processor and works.
func TestWithOnVueLoadProcessor(t *testing.T) {
	opts := newOptions()
	processor := func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error) {
		return content + "_processed", nil
	}
	WithOnVueLoadProcessor(processor)(opts)
	if len(opts.onVueLoadProcessors) != 1 {
		t.Errorf("Expected 1 vue load processor, got %d", len(opts.onVueLoadProcessors))
	}
	out, err := opts.onVueLoadProcessors[0]("abc", api.OnLoadArgs{}, &api.BuildOptions{})
	if err != nil || out != "abc_processed" {
		t.Errorf("Processor did not work as expected")
	}
}

// TestWithOnVueResolveProcessor verifies that WithOnVueResolveProcessor adds a processor.
func TestWithOnVueResolveProcessor(t *testing.T) {
	opts := newOptions()
	processor := func(args *api.OnResolveArgs, buildOptions *api.BuildOptions) (*api.OnResolveResult, error) {
		return nil, nil
	}
	WithOnVueResolveProcessor(processor)(opts)
	if len(opts.onVueResolveProcessors) != 1 {
		t.Errorf("Expected 1 vue resolve processor, got %d", len(opts.onVueResolveProcessors))
	}
}

// TestWithOnSassLoadProcessor verifies that WithOnSassLoadProcessor adds a processor.
func TestWithOnSassLoadProcessor(t *testing.T) {
	opts := newOptions()
	processor := func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error) {
		return content + "_sass_processed", nil
	}
	WithOnSassLoadProcessor(processor)(opts)
	if len(opts.onSassLoadProcessors) != 1 {
		t.Errorf("Expected 1 sass load processor, got %d", len(opts.onSassLoadProcessors))
	}
	out, err := opts.onSassLoadProcessors[0]("abc", api.OnLoadArgs{}, &api.BuildOptions{})
	if err != nil || out != "abc_sass_processed" {
		t.Errorf("Processor did not work as expected, got out=%s err=%v", out, err)
	}
}

// TestWithOnEndProcessor verifies that WithOnEndProcessor adds a processor.
func TestWithOnEndProcessor(t *testing.T) {
	opts := newOptions()
	processor := func(result *api.BuildResult, buildOptions *api.BuildOptions) error { return nil }
	WithOnEndProcessor(processor)(opts)
	if len(opts.onEndProcessors) != 1 {
		t.Errorf("Expected 1 end processor, got %d", len(opts.onEndProcessors))
	}
}

// TestWithOnDisposeProcessor verifies that WithOnDisposeProcessor adds a processor.
func TestWithOnDisposeProcessor(t *testing.T) {
	opts := newOptions()
	processor := func(buildOptions *api.BuildOptions) {}
	WithOnDisposeProcessor(processor)(opts)
	if len(opts.onDisposeProcessors) != 1 {
		t.Errorf("Expected 1 dispose processor, got %d", len(opts.onDisposeProcessors))
	}
}

// TestWithJsExecutor verifies that WithJsExecutor sets the JS executor.
func TestWithJsExecutor(t *testing.T) {
	opts := newOptions()
	executor := jsexecutortest.NewUninitializedExecutor()
	WithJsExecutor(executor)(opts)
	if opts.jsExecutor != executor {
		t.Errorf("Expected jsExecutor to be set")
	}
}

// TestWithLogger verifies that WithLogger sets the logger.
func TestWithLogger(t *testing.T) {
	opts := newOptions()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	WithLogger(logger)(opts)
	if opts.logger != logger {
		t.Errorf("Expected logger to be set")
	}
}

// TestParseImportMetaEnvWithDirectKey checks direct key lookup in parseImportMetaEnv.
func TestParseImportMetaEnvWithDirectKey(t *testing.T) {
	defineMap := map[string]string{
		"import.meta.env.CUSTOM": `"test_value"`,
	}
	value, exists := parseImportMetaEnv(defineMap, "CUSTOM")
	if !exists {
		t.Errorf("Expected CUSTOM to exist in import.meta.env")
	}
	if value != "test_value" {
		t.Errorf("Expected value to be 'test_value', got %v", value)
	}
}

// TestParseImportMetaEnvWithNestedEnv checks nested env object lookup in parseImportMetaEnv.
func TestParseImportMetaEnvWithNestedEnv(t *testing.T) {
	defineMap := map[string]string{
		"import.meta.env": `{"NESTED_KEY": "nested_value"}`,
	}
	value, exists := parseImportMetaEnv(defineMap, "NESTED_KEY")
	if !exists {
		t.Errorf("Expected NESTED_KEY to exist in import.meta.env")
	}
	if value != "nested_value" {
		t.Errorf("Expected value to be 'nested_value', got %v", value)
	}
}

// TestParseImportMetaEnvWithInvalidJSON checks parseImportMetaEnv with invalid JSON.
func TestParseImportMetaEnvWithInvalidJSON(t *testing.T) {
	defineMap := map[string]string{
		"import.meta.env": `{invalid json}`,
	}
	_, exists := parseImportMetaEnv(defineMap, "MISSING_KEY")
	if exists {
		t.Errorf("Expected MISSING_KEY to not exist with invalid JSON")
	}
}

// TestParseImportMetaEnvNotFound checks parseImportMetaEnv for missing key.
func TestParseImportMetaEnvNotFound(t *testing.T) {
	defineMap := map[string]string{
		"other.key": "value",
	}
	_, exists := parseImportMetaEnv(defineMap, "MISSING_KEY")
	if exists {
		t.Errorf("Expected MISSING_KEY to not exist")
	}
}

// TestNormalizeEsbuildOptionsWithNilDefine checks normalization with nil Define.
func TestNormalizeEsbuildOptionsWithNilDefine(t *testing.T) {
	buildOptions := &api.BuildOptions{}
	normalizeEsbuildOptions(buildOptions)
	if buildOptions.Define == nil {
		t.Errorf("Expected Define to be initialized")
	}
	if !buildOptions.Metafile {
		t.Errorf("Expected Metafile to be true")
	}
}

// TestNormalizeEsbuildOptionsWithExistingDefine checks normalization with existing Define.
func TestNormalizeEsbuildOptionsWithExistingDefine(t *testing.T) {
	buildOptions := &api.BuildOptions{
		Define: map[string]string{
			"import.meta.env.MODE": `"development"`,
		},
	}
	normalizeEsbuildOptions(buildOptions)
	if buildOptions.Define["import.meta.env.MODE"] != `"development"` {
		t.Errorf("Expected existing MODE value to be preserved")
	}
}

// TestSimpleCopyFileNotFound checks SimpleCopy when the source file does not exist.
func TestSimpleCopyFileNotFound(t *testing.T) {
	copyProcessor := SimpleCopy(map[string]string{
		"nonexistent_file.txt": "output.txt",
	})
	err := copyProcessor(&api.BuildResult{}, &api.BuildOptions{})
	if err == nil {
		t.Errorf("Expected error when copying nonexistent file")
	}
}

// TestSimpleCopyCreateDirAndCopyFile checks SimpleCopy for normal file copy and directory creation.
func TestSimpleCopyCreateDirAndCopyFile(t *testing.T) {
	tmpSrc := "test_source.txt"
	tmpDst := "test_output/test_dest.txt"
	err := os.WriteFile(tmpSrc, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test source file: %v", err)
	}
	defer os.Remove(tmpSrc)
	defer os.RemoveAll("test_output")
	copyProcessor := SimpleCopy(map[string]string{
		tmpSrc: tmpDst,
	})
	err = copyProcessor(&api.BuildResult{}, &api.BuildOptions{})
	if err != nil {
		t.Errorf("Expected no error when copying file, got: %v", err)
	}
	content, err := os.ReadFile(tmpDst)
	if err != nil {
		t.Errorf("Failed to read copied file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Expected copied content to be 'test content', got: %s", string(content))
	}
}

// TestWithIndexHtmlOptionsAndProcessors checks WithIndexHtmlOptions and processor assignment.
func TestWithIndexHtmlOptionsAndProcessors(t *testing.T) {
	opts := newOptions()
	processor := func(doc *html.Node, result *api.BuildResult, opts *Options, build *api.PluginBuild) error {
		return nil
	}
	htmlOptions := IndexHtmlOptions{
		SourceFile:          "test.html",
		OutFile:             "output.html",
		RemoveTagXPaths:     []string{"//script"},
		IndexHtmlProcessors: []IndexHtmlProcessor{processor},
	}
	WithIndexHtmlOptions(htmlOptions)(opts)
	if opts.IndexHtmlOptions.SourceFile != "test.html" {
		t.Errorf("Expected SourceFile to be 'test.html', got %s", opts.IndexHtmlOptions.SourceFile)
	}
	if opts.IndexHtmlOptions.OutFile != "output.html" {
		t.Errorf("Expected OutFile to be 'output.html', got %s", opts.IndexHtmlOptions.OutFile)
	}
	if len(opts.IndexHtmlOptions.RemoveTagXPaths) != 1 {
		t.Errorf("Expected 1 RemoveTagXPath, got %d", len(opts.IndexHtmlOptions.RemoveTagXPaths))
	}
	if len(opts.IndexHtmlOptions.IndexHtmlProcessors) != 1 {
		t.Errorf("Expected 1 IndexHtmlProcessor, got %d", len(opts.IndexHtmlOptions.IndexHtmlProcessors))
	}
}

// TestSimpleCopyMkdirAllFail checks SimpleCopy when MkdirAll fails.
func TestSimpleCopyMkdirAllFail(t *testing.T) {
	srcFile := "test_source.txt"
	var dstFile string

	// Use different invalid paths for different operating systems
	switch runtime.GOOS {
	case "windows":
		// On Windows, use reserved device name or invalid characters
		dstFile = "CON/foo.txt" // CON is a reserved device name on Windows
	default:
		// On Unix-like systems, use /dev/null as a file instead of directory
		dstFile = "/dev/null/foo.txt"
	}

	err := os.WriteFile(srcFile, []byte("x"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	defer os.Remove(srcFile)

	copyProcessor := SimpleCopy(map[string]string{
		srcFile: dstFile,
	})
	err = copyProcessor(&api.BuildResult{}, &api.BuildOptions{})
	if err == nil {
		t.Error("Expected directory creation error, got none")
	}
	if !strings.Contains(err.Error(), "failed to create output dir") {
		t.Errorf("Expected directory creation error, got: %v", err)
	}
}

// TestSimpleCopyCreateDirFail checks SimpleCopy with an invalid output path.
func TestSimpleCopyCreateDirFail(t *testing.T) {
	copyProcessor := SimpleCopy(map[string]string{
		"test_source.txt": string([]byte{0}), // Invalid path
	})
	_ = os.WriteFile("test_source.txt", []byte("x"), 0644)
	defer os.Remove("test_source.txt")
	err := copyProcessor(&api.BuildResult{}, &api.BuildOptions{})
	if err == nil ||
		!(strings.Contains(err.Error(), "failed to create output dir") ||
			strings.Contains(err.Error(), "failed to create output file") ||
			strings.Contains(err.Error(), "invalid argument")) {
		t.Errorf("Expected directory or file creation error, got: %v", err)
	}
}

// TestSimpleCopyCopyFail checks SimpleCopy when io.Copy fails (using a directory as source).
func TestSimpleCopyCopyFail(t *testing.T) {
	srcDir := "test_src_dir"
	dstFile := filepath.Join(t.TempDir(), "test_output.txt")
	err := os.Mkdir(srcDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)
	copyProcessor := SimpleCopy(map[string]string{
		srcDir: dstFile,
	})
	err = copyProcessor(&api.BuildResult{}, &api.BuildOptions{})
	if err == nil || !strings.Contains(err.Error(), "failed to copy from") {
		t.Errorf("Expected copy error, got: %v", err)
	}
}
