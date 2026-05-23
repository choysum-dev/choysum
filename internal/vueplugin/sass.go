// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/rs/xid"
)

// setupSassHandler registers handlers for Sass files (.scss and .sass).
// It sets up the complete processing pipeline including path resolution and compilation.
func setupSassHandler(opts *Options, build *api.PluginBuild) {
	// Register resolve handler for Sass files
	registerSassResolveHandler(opts, build)

	// Register load and compile handler for Sass files
	registerSassLoadHandler(opts, build)
}

// registerSassResolveHandler registers the path resolution handler for Sass files.
// It handles TypeScript path aliases and converts relative paths to absolute paths.
// Resolved Sass files are assigned to the "sass-loader" namespace for further processing.
func registerSassResolveHandler(opts *Options, build *api.PluginBuild) {
	build.OnResolve(api.OnResolveOptions{Filter: `\.s[ac]ss$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		// Parse TypeScript path aliases from tsconfig to support project-wide path mapping
		pathAlias, err := ParseTsconfigPathAlias(build.InitialOptions)
		if err != nil {
			opts.logger.Error("tsconfig path alias parsing failed", "error", err)
			return api.OnResolveResult{}, err
		}

		// Apply path aliases to support imports like @/styles/main.scss
		args.Path = ApplyPathAlias(pathAlias, args.Path)

		// Convert relative paths to absolute paths for consistent file resolution
		path := args.Path
		if !filepath.IsAbs(args.Path) {
			path = filepath.Clean(filepath.Join(args.ResolveDir, args.Path))
		}

		return api.OnResolveResult{
			Path:      path,
			Namespace: "sass-loader", // Assign to sass-loader namespace for compilation
		}, nil
	})
}

// registerSassLoadHandler registers the handler to load and compile Sass files.
// It reads the source content, processes it through any registered processors,
// and compiles it to CSS using the Vue compiler's Sass service.
func registerSassLoadHandler(opts *Options, build *api.PluginBuild) {
	build.OnLoad(api.OnLoadOptions{Filter: `\.s[ac]ss$`, Namespace: "sass-loader"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		// Step 1: Read the Sass source content (with optional preprocessing)
		source, err := readSassSource(args, opts, build)
		if err != nil {
			opts.logger.Error("sass file read failed", "error", err, "file", args.Path)
			return api.OnLoadResult{
				Errors: []api.Message{{
					Text: err.Error(),
					Location: &api.Location{
						File: args.Path,
					},
				}},
			}, err
		}

		// Step 2: Compile Sass to CSS using the Vue compiler's integrated Sass service
		css, err := compileSass(args.Path, source, opts.jsExecutor)
		if err != nil {
			opts.logger.Error("sass compilation failed", "error", err, "file", args.Path)
			return api.OnLoadResult{
				Errors: []api.Message{{
					Text: err.Error(),
					Location: &api.Location{
						File: args.Path,
					},
				}},
			}, err
		}

		// Step 3: Return compiled CSS with appropriate loader
		return api.OnLoadResult{
			Contents: &css,
			Loader:   api.LoaderCSS, // Use CSS loader for the compiled output
		}, nil
	})
}

// readSassSource reads the Sass source file content with optional preprocessing.
// If any Sass load processor is registered, it will be executed first to allow
// custom content transformation or dynamic content generation.
// If no processor returns content, the file will be read from disk.
func readSassSource(args api.OnLoadArgs, opts *Options, build *api.PluginBuild) (string, error) {
	// Read file contents
	fbyte, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read sass file: %w", err)
	}

	// Convert to string
	source := string(fbyte)

	source = strings.ReplaceAll(source, "\r\n", "\n")
	// ResolveVueStylePath applies path alias and absolute path resolution for <style> blocks in Vue SFC content.
	pathAlias, err := ParseTsconfigPathAlias(build.InitialOptions)
	if err != nil {
		return "", fmt.Errorf("failed to parse TypeScript path aliases: %w", err)
	}
	source = ResolveScssPath(source, args.Path, pathAlias)

	// Execute Sass load processor chain if present
	// Each processor can transform the source content
	if len(opts.onSassLoadProcessors) > 0 {
		for _, processor := range opts.onSassLoadProcessors {
			var processorErr error
			source, processorErr = processor(source, args, build.InitialOptions)
			if processorErr != nil {
				return "", fmt.Errorf("sass processor failed: %w", processorErr)
			}
		}
	}

	return source, nil
}

// compileSass compiles Sass to CSS using the Vue compiler via the JS executor.
// It uses the integrated Sass compiler service that supports both .scss and .sass syntax.
// The compilation includes dependency resolution and supports Sass features like imports,
// variables, mixins, and functions.
func compileSass(filePath, source string, jsExecutor jsexecutor.ScriptExecutor) (string, error) {
	// Extract directory path for Sass import resolution
	location := filepath.Dir(filePath)

	// Execute Sass compilation via the Vue compiler's Sass service
	jsResponse, err := jsExecutor.Execute(context.Background(), &jsengine.JsRequest{
		Id:      xid.New().String(),
		Service: "sfc.sass.renderSync", // Vue compiler's integrated Sass service
		Args: []interface{}{map[string]interface{}{
			"data":         source,     // Sass source code to compile
			"sasslocation": location,   // Base directory for resolving @import statements
			"sourceMap":    false,      // Disable source maps for production builds
			"style":        "expanded", // Output style: expanded, compressed, etc.
		}},
	})

	if err != nil {
		return "", fmt.Errorf("sass compilation service failed: %w", err)
	}

	// Extract and validate compilation result
	result, ok := jsResponse.Result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response from sass compilation service")
	}

	// Extract the compiled CSS code from the result
	code, ok := result["css"].(string)
	if !ok {
		return "", fmt.Errorf("failed to extract CSS from compilation result")
	}

	return code, nil
}
