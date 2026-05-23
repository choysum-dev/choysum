// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"github.com/evanw/esbuild/pkg/api"
)

// NewPlugin creates a new esbuild plugin for Vue SFC support.
// It accepts a list of OptionFunc to customize plugin behavior such as:
// - JavaScript executor configuration
// - Template compiler options
// - Style preprocessor options
// - Custom processor chains for various build phases
//
// The plugin handles Vue Single File Components (.vue), Sass files (.scss/.sass),
// and HTML files with comprehensive build integration.
//
// Example usage:
//
//	plugin := NewPlugin(
//	  WithJsExecutor(jsExec),
//	  WithTemplateCompilerOptions(map[string]interface{}{"sourceMap": true}),
//	)
//
// Panics if jsExecutor is not provided, as it's required for Vue compilation.
func NewPlugin(optsFunc ...OptionFunc) api.Plugin {
	// Initialize default options
	opts := newOptions()

	// Apply all provided option functions to configure the plugin
	for _, fn := range optsFunc {
		fn(opts)
	}

	// Validate required dependencies - jsExecutor is mandatory for Vue compilation
	if opts.jsExecutor == nil {
		panic("jsExecutor is required, please set it using WithJsExecutor()")
	}

	return api.Plugin{
		Name: opts.Name, // Plugin name for identification in esbuild logs
		Setup: func(build api.PluginBuild) {
			// Step 1: Normalize and validate esbuild options for compatibility
			normalizeEsbuildOptions(build.InitialOptions)

			// Step 2: Register start processor chain - executed before build starts
			// This allows for pre-build initialization, configuration validation, etc.
			build.OnStart(func() (api.OnStartResult, error) {
				// Execute all registered start processors in sequence
				for _, processor := range opts.onStartProcessors {
					if err := processor(build.InitialOptions); err != nil {
						opts.logger.Error("start processor failed", "error", err)
						return api.OnStartResult{}, err
					}
				}
				return api.OnStartResult{}, nil
			})

			// Step 3: Register all file type handlers for comprehensive support
			setupVueHandler(opts, &build)  // Handle .vue Single File Components
			setupSassHandler(opts, &build) // Handle .scss/.sass style files
			setupHtmlHandler(opts, &build) // Handle .html template files

			// Step 4: Register end processor chain - executed after all processing is done
			// This allows for post-build processing, asset manipulation, cleanup, etc.
			build.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
				// Execute all registered end processors with build results
				for _, processor := range opts.onEndProcessors {
					if err := processor(result, build.InitialOptions); err != nil {
						opts.logger.Error("end processor failed", "error", err)
						return api.OnEndResult{}, err
					}
				}
				return api.OnEndResult{}, nil
			})

			// Step 5: Register dispose processor chain - cleanup after build completion
			// This handles resource cleanup, temporary file removal, connection closure, etc.
			build.OnDispose(func() {
				// Execute all registered dispose processors for cleanup
				for _, processor := range opts.onDisposeProcessors {
					// Note: Dispose processors don't return errors as cleanup should be best-effort
					processor(build.InitialOptions)
				}
			})
		},
	}
}
