// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
)

// OnStartProcessor is a function type for processing logic before the build starts.
// Receives the esbuild BuildOptions as input and can perform pre-build initialization,
// configuration validation, or environment setup.
// Returns an error if the processing fails, which will abort the build.
type OnStartProcessor func(buildOptions *api.BuildOptions) error

// OnVueResolveProcessor is a function type for custom Vue file resolution logic.
// Receives the OnResolveArgs and BuildOptions, returns an optional OnResolveResult and error.
// If the processor returns a non-nil OnResolveResult, it will be used as the final result.
// If nil is returned, the default resolution logic will be used.
type OnVueResolveProcessor func(args *api.OnResolveArgs, buildOptions *api.BuildOptions) (*api.OnResolveResult, error)

// OnVueLoadProcessor is a function type for custom Vue file loading logic.
// Receives the file content, OnLoadArgs, and BuildOptions, returns the (possibly transformed) content and error.
// This allows for content transformation, preprocessing, or dynamic content injection before Vue compilation.
type OnVueLoadProcessor func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error)

// OnSassLoadProcessor is a function type for custom Sass file loading logic.
// Receives the file content, OnLoadArgs, and BuildOptions, returns the (possibly transformed) content and error.
// If content is returned, it will be used instead of reading from the file system.
// Return empty string to fallback to default file reading behavior.
type OnSassLoadProcessor func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error)

// OnEndProcessor is a function type for processing logic after the build ends.
// Receives the BuildResult and BuildOptions as input and can perform post-build processing,
// asset manipulation, file copying, or result analysis.
// Returns an error if the processing fails.
type OnEndProcessor func(result *api.BuildResult, buildOptions *api.BuildOptions) error

// OnDisposeProcessor is a function type for cleanup logic after the build is disposed.
// Receives the BuildOptions as input and should perform resource cleanup,
// temporary file removal, or connection closure.
// Note: Dispose processors should not return errors as cleanup should be best-effort.
type OnDisposeProcessor func(buildOptions *api.BuildOptions)

// IndexHtmlProcessor is a function type for processing HTML files after build.
// Receives the HTML document node, BuildResult, plugin options, and PluginBuild context.
// Can modify the HTML DOM, inject scripts/styles, or perform other HTML transformations.
// Returns an error if the processing fails.
type IndexHtmlProcessor func(doc *html.Node, result *api.BuildResult, opts *Options, build *api.PluginBuild) error

// IndexHtmlOptions holds configuration options for HTML file processing.
// Used to configure how HTML files are processed and transformed during the build.
type IndexHtmlOptions struct {
	SourceFile          string               // Source HTML file path to process
	OutFile             string               // Output HTML file path after processing
	RemoveTagXPaths     []string             // XPath expressions for removing specific HTML nodes
	IndexHtmlProcessors []IndexHtmlProcessor // Custom processors for HTML transformation
}

// options holds all plugin configuration and processor chains.
// This is the internal configuration structure used by the plugin to manage
// all settings, compiler options, and processor chains.
type Options struct {
	Name                     string           // Plugin name for identification
	TemplateCompilerOptions  map[string]any   // Vue template compiler configuration
	StylePreprocessorOptions map[string]any   // Style preprocessor configuration (Sass, Less, etc.)
	IndexHtmlOptions         IndexHtmlOptions // HTML processing configuration

	// Processor chains for plugin extension points
	onStartProcessors      []OnStartProcessor      // Executed before build starts
	onVueResolveProcessors []OnVueResolveProcessor // Custom Vue file resolution logic
	onVueLoadProcessors    []OnVueLoadProcessor    // Custom Vue file loading/preprocessing
	onSassLoadProcessors   []OnSassLoadProcessor   // Custom Sass file loading/preprocessing
	onEndProcessors        []OnEndProcessor        // Executed after build completes
	onDisposeProcessors    []OnDisposeProcessor    // Executed during cleanup

	jsExecutor jsexecutor.ScriptExecutor // JavaScript executor for Vue compilation
	logger     *slog.Logger              // Logger for plugin messages
}

// OptionFunc is a function type for configuring plugin options using the functional options pattern.
// Each option function receives the options struct and modifies it to customize plugin behavior.
type OptionFunc func(*Options)

// newOptions creates a new options struct with sensible default values.
// Initializes empty maps for compiler options and sets up default logger.
func newOptions() *Options {
	return &Options{
		Name:                     "vue-plugin",         // Default plugin name
		TemplateCompilerOptions:  make(map[string]any), // Empty template options
		StylePreprocessorOptions: make(map[string]any), // Empty style options
		logger:                   slog.Default(),       // Use default structured logger
	}
}

// WithName sets a custom plugin name for identification in esbuild logs and error messages.
// Useful when running multiple instances of the plugin with different configurations.
func WithName(name string) OptionFunc {
	return func(opts *Options) {
		opts.Name = name
	}
}

// WithTemplateCompilerOptions sets the Vue template compiler options.
// These options are passed directly to the Vue compiler and can include:
// - sourceMap: boolean - Generate source maps for templates
// - compilerOptions: object - Vue compiler specific options
// - transformAssetUrls: object - Asset URL transformation rules
func WithTemplateCompilerOptions(templateCompilerOptions map[string]any) OptionFunc {
	return func(opts *Options) {
		opts.TemplateCompilerOptions = templateCompilerOptions
	}
}

// WithStylePreprocessorOptions sets the style preprocessor options.
// These options are passed to style preprocessors (Sass, Less, Stylus) and can include:
// - includePaths: []string - Additional paths for @import resolution
// - outputStyle: string - CSS output style (expanded, compressed, etc.)
// - sourceMap: boolean - Generate source maps for styles
func WithStylePreprocessorOptions(stylePreprocessorOptions map[string]any) OptionFunc {
	return func(opts *Options) {
		opts.StylePreprocessorOptions = stylePreprocessorOptions
	}
}

// WithIndexHtmlOptions sets the HTML processing options.
// Configures how HTML files are processed, including source/output paths and custom processors.
func WithIndexHtmlOptions(indexHtmlOptions IndexHtmlOptions) OptionFunc {
	return func(opts *Options) {
		opts.IndexHtmlOptions = indexHtmlOptions
	}
}

// WithOnStartProcessor adds an OnStartProcessor to the processor chain.
// Start processors are executed before the build begins and can perform setup tasks,
// validation, or environment preparation.
func WithOnStartProcessor(processor OnStartProcessor) OptionFunc {
	return func(opts *Options) {
		opts.onStartProcessors = append(opts.onStartProcessors, processor)
	}
}

// WithOnVueResolveProcessor adds an OnVueResolveProcessor to the processor chain.
// Resolve processors can customize how Vue file imports are resolved,
// enabling custom path mapping, virtual modules, or dynamic resolution logic.
func WithOnVueResolveProcessor(processor OnVueResolveProcessor) OptionFunc {
	return func(opts *Options) {
		opts.onVueResolveProcessors = append(opts.onVueResolveProcessors, processor)
	}
}

// WithOnVueLoadProcessor adds an OnVueLoadProcessor to the processor chain.
// Load processors can transform Vue file content before compilation,
// enabling preprocessing, content injection, or dynamic code generation.
func WithOnVueLoadProcessor(processor OnVueLoadProcessor) OptionFunc {
	return func(opts *Options) {
		opts.onVueLoadProcessors = append(opts.onVueLoadProcessors, processor)
	}
}

// WithOnSassLoadProcessor adds an OnSassLoadProcessor to the processor chain.
// Sass load processors can provide custom content for Sass files,
// enabling dynamic imports, content generation, or preprocessing.
func WithOnSassLoadProcessor(processor OnSassLoadProcessor) OptionFunc {
	return func(opts *Options) {
		opts.onSassLoadProcessors = append(opts.onSassLoadProcessors, processor)
	}
}

// WithOnEndProcessor adds an OnEndProcessor to the processor chain.
// End processors are executed after the build completes and can perform post-processing,
// file copying, asset manipulation, or build result analysis.
func WithOnEndProcessor(processor OnEndProcessor) OptionFunc {
	return func(opts *Options) {
		opts.onEndProcessors = append(opts.onEndProcessors, processor)
	}
}

// WithOnDisposeProcessor adds an OnDisposeProcessor to the processor chain.
// Dispose processors are executed during cleanup and should handle resource cleanup,
// temporary file removal, or connection closure.
func WithOnDisposeProcessor(processor OnDisposeProcessor) OptionFunc {
	return func(opts *Options) {
		opts.onDisposeProcessors = append(opts.onDisposeProcessors, processor)
	}
}

// WithJsExecutor sets the JavaScript executor for Vue compilation.
// The JS executor is required and handles communication with the Vue compiler running in a JavaScript context.
// It's used for compiling Vue Single File Components and processing style files.
func WithJsExecutor(jsExecutor jsexecutor.ScriptExecutor) OptionFunc {
	return func(opts *Options) {
		opts.jsExecutor = jsExecutor
	}
}

// WithLogger sets a custom logger for the plugin.
// The logger is used for debug information, warnings, and error messages throughout the plugin.
// Defaults to slog.Default() if not specified.
func WithLogger(logger *slog.Logger) OptionFunc {
	return func(opts *Options) {
		opts.logger = logger
	}
}

// parseImportMetaEnv parses import.meta.env values from esbuild Define map.
// Supports both individual env variable definitions and nested env object definitions.
// This function handles the complexity of esbuild's Define format for environment variables.
func parseImportMetaEnv(defineMap map[string]string, key string) (any, bool) {
	// First, try to find the specific env variable (e.g., "import.meta.env.NODE_ENV")
	if v, ok := defineMap[fmt.Sprintf("import.meta.env.%s", key)]; ok {
		var value interface{}
		// Unmarshal the JSON-encoded value
		json.Unmarshal([]byte(v), &value)
		return value, true
	}

	// Second, try to find it within the env object (e.g., "import.meta.env": "{\"NODE_ENV\": \"production\"}")
	if v, ok := defineMap["import.meta.env"]; ok {
		var value interface{}
		json.Unmarshal([]byte(v), &value)
		if envMap, ok := value.(map[string]interface{}); ok {
			if v, ok := envMap[key]; ok {
				return v, true
			}
		}
	}

	return nil, false
}

// normalizeEsbuildOptions sets default values and ensures esbuild options are valid for Vue development.
// This function configures essential environment variables and Vue-specific defines that are
// commonly needed for Vue applications to work correctly.
func normalizeEsbuildOptions(initialOptions *api.BuildOptions) {
	// Initialize Define map if not present
	if initialOptions.Define == nil {
		initialOptions.Define = make(map[string]string)
	}

	// Set default import.meta.env object if not defined
	if _, ok := initialOptions.Define["import.meta.env"]; !ok {
		initialOptions.Define["import.meta.env"] = "{}"
	}

	// Configure standard Vite-compatible environment variables
	if _, exists := parseImportMetaEnv(initialOptions.Define, "MODE"); !exists {
		initialOptions.Define["import.meta.env.MODE"] = "'production'"
	}
	if _, exists := parseImportMetaEnv(initialOptions.Define, "PROD"); !exists {
		initialOptions.Define["import.meta.env.PROD"] = "true"
	}
	if _, exists := parseImportMetaEnv(initialOptions.Define, "DEV"); !exists {
		initialOptions.Define["import.meta.env.DEV"] = "false"
	}
	if _, exists := parseImportMetaEnv(initialOptions.Define, "SSR"); !exists {
		initialOptions.Define["import.meta.env.SSR"] = "false"
	}
	if _, exists := parseImportMetaEnv(initialOptions.Define, "BASE_URL"); !exists {
		initialOptions.Define["import.meta.env.BASE_URL"] = "'/'"
	}

	// Configure Vue-specific feature flags for optimal bundle size and behavior
	if _, ok := initialOptions.Define["__VUE_OPTIONS_API__"]; !ok {
		initialOptions.Define["__VUE_OPTIONS_API__"] = fmt.Sprintf(`%t`, true)
	}
	if _, ok := initialOptions.Define["__VUE_PROD_DEVTOOLS__"]; !ok {
		initialOptions.Define["__VUE_PROD_DEVTOOLS__"] = fmt.Sprintf(`%t`, false)
	}
	if _, ok := initialOptions.Define["__VUE_PROD_HYDRATION_MISMATCH_DETAILS__"]; !ok {
		initialOptions.Define["__VUE_PROD_HYDRATION_MISMATCH_DETAILS__"] = fmt.Sprintf(`%t`, false)
	}

	// Enable metafile generation for build analysis and HTML processing
	initialOptions.Metafile = true
}

// SimpleCopy returns an OnEndProcessor that copies files from fileMap after build completion.
// Each key-value pair in fileMap represents srcFile -> outFile mapping.
// This is a utility function for common file copying operations in build workflows.
//
// Example usage:
//
//	processor := SimpleCopy(map[string]string{
//	  "src/assets/favicon.ico": "dist/favicon.ico",
//	  "public/robots.txt": "dist/robots.txt",
//	})
func SimpleCopy(fileMap map[string]string) OnEndProcessor {
	return func(result *api.BuildResult, initialOptions *api.BuildOptions) error {
		// Process each file mapping in the provided map
		for srcFile, outFile := range fileMap {
			// Step 1: Open the source file for reading
			src, err := os.Open(srcFile)
			if err != nil {
				return fmt.Errorf("failed to open source file %s: %w", srcFile, err)
			}
			defer src.Close()

			// Step 2: Ensure the output directory structure exists
			if err := os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
				return fmt.Errorf("failed to create output dir for %s: %w", outFile, err)
			}

			// Step 3: Create the destination file
			dst, err := os.Create(outFile)
			if err != nil {
				return fmt.Errorf("failed to create output file %s: %w", outFile, err)
			}
			defer dst.Close()

			// Step 4: Copy the file contents efficiently
			if _, err := io.Copy(dst, src); err != nil {
				return fmt.Errorf("failed to copy from %s to %s: %w", srcFile, outFile, err)
			}
		}
		return nil
	}
}
