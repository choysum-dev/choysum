// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/rs/xid"
	"github.com/zeebo/xxh3"
)

// toPosixPath converts Windows-style paths to POSIX-style paths by replacing backslashes with forward slashes.
// This ensures consistent path handling across different operating systems.
func toPosixPath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

// setupVueHandler registers all handlers for Vue Single File Components (.vue files).
// It sets up the complete processing pipeline including main entry, resolve, script, template, and style handlers.
func setupVueHandler(opts *Options, build *api.PluginBuild) {
	// Register main entry handler for .vue files
	registerMainEntryHandler(opts, build)

	// Register file resolve handler
	registerResolveHandler(opts, build)

	// Register handlers for Vue SFC parts
	registerScriptHandler(build)
	registerTemplateHandler(build)
	registerStyleHandler(build)
}

// registerMainEntryHandler processes .vue files and precompiles all SFC parts.
// This is the main entry point that orchestrates the compilation of Vue Single File Components.
// It reads the source, compiles it using the JS executor, and generates the final entry code.
func registerMainEntryHandler(opts *Options, build *api.PluginBuild) {
	// Determine production mode from build environment
	isProd := true
	if v, exists := parseImportMetaEnv(build.InitialOptions.Define, "PROD"); exists {
		_v, ok := v.(bool)
		if ok {
			isProd = _v
		}
	}

	// Determine SSR mode from build environment
	isSSR := false
	if v, exists := parseImportMetaEnv(build.InitialOptions.Define, "SSR"); exists {
		_v, ok := v.(bool)
		if ok {
			isSSR = _v
		}
	}

	build.OnLoad(api.OnLoadOptions{Filter: `\.vue$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		// Step 1: Read and preprocess the Vue source file
		source, err := readVueSource(args, opts, build)
		if err != nil {
			opts.logger.Error("vue file read failed", "error", err, "file", args.Path)
			return api.OnLoadResult{}, err
		}

		// Step 2: Generate unique component ID based on source content
		hashId := generateHashId(source)
		dataId := "data-v-" + hashId

		// Step 3: Compile SFC using the JavaScript executor
		jsResponse, err := opts.jsExecutor.Execute(context.Background(), &jsengine.JsRequest{
			Id:      xid.New().String(),
			Service: "sfc.vue.compileSFC",
			Args: []interface{}{
				hashId,
				toPosixPath(args.Path),
				source,
				map[string]interface{}{
					"sourceMap":         build.InitialOptions.Sourcemap > 0,
					"isProd":            isProd,
					"isSSR":             isSSR,
					"preprocessOptions": opts.StylePreprocessorOptions,
					"compilerOptions":   opts.TemplateCompilerOptions,
				},
			},
		})
		if err != nil {
			opts.logger.Error("vue sfc compilation failed", "error", err, "file", args.Path)
			return api.OnLoadResult{
				Errors: []api.Message{{
					Text: fmt.Sprintf("Vue SFC compilation failed: %v", err),
					Location: &api.Location{
						File: args.Path,
					},
				}},
			}, err
		}

		// Validate compilation result format
		compileResult, ok := jsResponse.Result.(map[string]interface{})
		if !ok {
			opts.logger.Error("vue sfc compilation result invalid", "result", jsResponse.Result, "file", args.Path)
			return api.OnLoadResult{
				Errors: []api.Message{{
					Text: fmt.Sprintf("Invalid Vue SFC compilation result: %v", jsResponse.Result),
					Location: &api.Location{
						File: args.Path,
					},
				}},
			}, err
		}

		// Step 4: Extract each SFC part from the compilation result
		var script map[string]interface{}
		scriptResult, ok := compileResult["script"]
		if scriptResult != nil && ok {
			script = scriptResult.(map[string]interface{})
		}

		var template map[string]interface{}
		if templateResult, ok := compileResult["template"].(map[string]interface{}); ok && templateResult != nil {
			template = templateResult
		}

		var styles []map[string]interface{}
		if stylesResult, ok := compileResult["styles"].([]interface{}); ok && len(stylesResult) > 0 {
			styles = make([]map[string]interface{}, len(stylesResult))
			for i, s := range stylesResult {
				if sMap, ok := s.(map[string]interface{}); ok {
					styles[i] = sMap
				}
			}
		}

		// Step 5: Generate entry JavaScript code that imports and combines all SFC parts
		contents, err := generateEntryContents(args.Path, dataId, isSSR, script, template, styles)
		if err != nil {
			opts.logger.Error("vue entry generation failed", "error", err, "file", args.Path)
			return api.OnLoadResult{
				Errors: []api.Message{{
					Text: fmt.Sprintf("Failed to generate Vue entry contents: %v", err),
					Location: &api.Location{
						File: args.Path,
					},
				}},
			}, err
		}

		// Step 6: Prepare plugin data for subsequent handlers
		pluginData := map[string]interface{}{
			"id":     dataId,
			"script": script,
		}

		if template != nil {
			pluginData["template"] = template
		}

		if styles != nil {
			pluginData["styles"] = styles
		}

		// Step 7: Extract script warnings and convert them to build warnings
		buildWarnings := make([]api.Message, 0)
		if warnings, ok := script["warnings"].([]interface{}); ok {
			for _, w := range warnings {
				if warn, isString := w.(string); isString {
					buildWarnings = append(buildWarnings, api.Message{Text: warn})
				}
			}
		}

		return api.OnLoadResult{
			Contents:   &contents,
			ResolveDir: filepath.Dir(args.Path),
			PluginData: pluginData,
			Warnings:   buildWarnings,
		}, nil
	})
}

// readVueSource reads and preprocesses the Vue source file.
// It normalizes line endings and executes any registered Vue load processor chain.
func readVueSource(args api.OnLoadArgs, opts *Options, build *api.PluginBuild) (string, error) {
	// Read file contents
	fbyte, err := os.ReadFile(args.Path)
	if err != nil {
		return "", err
	}

	// Normalize line endings to Unix-style
	source := string(fbyte)
	source = strings.ReplaceAll(source, "\r\n", "\n")

	// ResolveVueStylePath applies path alias and absolute path resolution for <style> blocks in Vue SFC content.
	pathAlias, err := ParseTsconfigPathAlias(build.InitialOptions)
	if err != nil {
		return "", fmt.Errorf("failed to parse TypeScript path aliases: %w", err)
	}
	source = ResolveVueStylePath(source, args.Path, pathAlias)

	// Execute Vue load processor chain if any processors are registered
	if len(opts.onVueLoadProcessors) > 0 {
		for _, processor := range opts.onVueLoadProcessors {
			var processorErr error
			source, processorErr = processor(source, args, build.InitialOptions)
			if processorErr != nil {
				return "", processorErr
			}
		}
	}

	return source, nil
}

// generateHashId generates a unique hash ID for the given source string.
// Uses xxh3 for fast and consistent hashing across builds.
func generateHashId(source string) string {
	return strconv.FormatUint(xxh3.HashString(source), 16)
}

// generateEntryContents generates the entry JavaScript code for a Vue Single File Component.
// It creates import statements and component setup code that combines the script, template, and styles.
// The generated code follows Vue 3's component structure and handles SSR/CSR rendering modes.
func generateEntryContents(filePath, dataId string, isSSR bool,
	sfcScript map[string]interface{}, sfcTemplate map[string]interface{},
	sfcStyles []map[string]interface{}) (string, error) {

	// Convert file path to relative POSIX path for consistent import statements
	relPath, err := filepath.Rel(".", filePath)
	if err != nil {
		relPath = filePath
	}
	relPath = toPosixPath(relPath)

	// Create template with helper functions for conditional code generation
	tpl := template.Must(template.New(filePath).Funcs(template.FuncMap{
		"SSR": func() bool {
			return isSSR
		},
		"dataId": func() string {
			return dataId
		},
		"hasScript": func() bool {
			return sfcScript != nil && sfcScript["content"] != nil
		},
		"hasTemplate": func() bool {
			return sfcTemplate != nil && sfcTemplate["code"] != nil
		},
		"someScoped": func() bool {
			for _, style := range sfcStyles {
				if scoped := style["scoped"].(bool); scoped {
					return true
				}
			}
			return false
		},
	}).Parse(`
{{ if hasScript }}
import script from '{{ .relPath }}?type=script'
{{ else }}
const script = {};
{{ end }}

{{ range $index, $_ := .styles }}
import '{{ $.relPath }}?type=style&index={{ $index }}'
{{ end }}

{{ if hasTemplate }}
import { {{ if SSR }}ssrRender{{ else }}render{{ end }} } from '{{ .relPath }}?type=template'
script.{{ if SSR }}ssrRender{{ else }}render{{ end }} = {{ if SSR }}ssrRender{{ else }}render{{ end }};
{{ end }}

script.__file = {{ .relPath | printf "%q" }};
{{ if someScoped }}
script.__scopeId = {{ printf "%q" (dataId) }};{{ end }}
{{ if SSR }}
script.__ssrInlineRender = true;
{{ end }}

{{ if hasScript }}
export * from '{{ .relPath }}?type=script'
{{ end }}
export default script;
`))

	// Prepare template data
	data := map[string]interface{}{
		"relPath": relPath,
		"styles":  sfcStyles,
	}

	// Execute template and generate final code
	contentsBuf := new(bytes.Buffer)
	if err := tpl.Execute(contentsBuf, data); err != nil {
		return "", fmt.Errorf("failed to execute Vue entry template: %w", err)
	}

	return contentsBuf.String(), nil
}

// registerResolveHandler registers the file resolution handler for .vue files.
// Handles path aliases, relative paths, URL parameters, and custom resolve processors.
// This handler determines how Vue file imports are resolved and which namespace they belong to.
func registerResolveHandler(opts *Options, build *api.PluginBuild) {
	build.OnResolve(api.OnResolveOptions{Filter: `\.vue(\?.*)?$`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		// Apply TypeScript path aliases if configured
		pathAlias, err := ParseTsconfigPathAlias(build.InitialOptions)
		if err != nil {
			return api.OnResolveResult{}, err
		}
		args.Path = ApplyPathAlias(pathAlias, args.Path)

		// Convert relative paths to absolute paths
		if !filepath.IsAbs(args.Path) {
			args.Path = filepath.Clean(filepath.Join(args.ResolveDir, args.Path))
		}

		// Execute custom Vue resolve processor chain if any processors are registered
		if len(opts.onVueResolveProcessors) > 0 {
			for _, processor := range opts.onVueResolveProcessors {
				result, err := processor(&args, build.InitialOptions)
				if err != nil {
					return api.OnResolveResult{}, err
				}
				if result != nil {
					return *result, nil
				}
			}
		}

		// Parse URL parameters to determine the SFC part type (script, template, style)
		pathUrl, _ := url.Parse(args.Path)
		params := pathUrl.Query()
		namespace := "file"
		if t := params.Get("type"); t != "" {
			namespace = "sfc-" + t
		}

		// Setup plugin data for passing information to subsequent handlers
		if args.PluginData == nil {
			args.PluginData = make(map[string]interface{})
		}

		// Extract style index from URL parameters for multi-style components
		if indexStr := params.Get("index"); indexStr != "" {
			if index, err := strconv.Atoi(indexStr); err == nil {
				args.PluginData.(map[string]interface{})["index"] = index
			}
		}

		return api.OnResolveResult{
			Path:       args.Path,
			Namespace:  namespace,
			PluginData: args.PluginData,
		}, nil
	})
}

// registerScriptHandler registers the script handler for Vue Single File Components.
// Loads the precompiled script part and optionally attaches sourcemap information.
// Determines the appropriate loader (JS/TS) based on the script language.
func registerScriptHandler(build *api.PluginBuild) {
	build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "sfc-script"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		pluginData := args.PluginData.(map[string]interface{})
		script := pluginData["script"].(map[string]interface{})

		content := script["content"].(string)

		// Append sourcemap as inline data URL if sourcemaps are enabled and available
		if build.InitialOptions.Sourcemap > 0 && script["map"] != nil {
			sourceMapJSON, err := json.Marshal(script["map"])
			if err != nil {
				return api.OnLoadResult{}, err
			}
			sourceMapBase64 := base64.StdEncoding.EncodeToString(sourceMapJSON)
			content += "\n\n//@ sourceMappingURL=data:application/json;charset=utf-8;base64," + sourceMapBase64
		}

		// Determine appropriate loader based on script language
		loader := api.LoaderJS
		if script["lang"] != nil && script["lang"].(string) == "ts" {
			loader = api.LoaderTS
		}

		return api.OnLoadResult{
			Contents:   &content,
			Loader:     loader,
			ResolveDir: filepath.Dir(args.Path),
			PluginData: pluginData,
		}, nil
	})
}

// registerTemplateHandler registers the template handler for Vue Single File Components.
// Loads the precompiled template part and converts template compilation tips to build warnings.
// Performs type-safe conversion of tips to prevent runtime panics.
func registerTemplateHandler(build *api.PluginBuild) {
	build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "sfc-template"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		pluginData := args.PluginData.(map[string]interface{})

		// Extract precompiled template result
		templateResult := pluginData["template"].(map[string]interface{})
		code := templateResult["code"].(string)

		// Convert template compilation tips to build warnings with type safety
		var mappedTips []api.Message
		if tips, ok := templateResult["tips"].([]interface{}); ok {
			for _, tip := range tips {
				// Type-safe conversion to string to prevent panics from invalid tip types
				if tipStr, isString := tip.(string); isString {
					mappedTips = append(mappedTips, api.Message{
						Text: tipStr,
					})
				}
				// Silently skip non-string tips to maintain build stability
			}
		}

		return api.OnLoadResult{
			Contents:   &code,
			Warnings:   mappedTips,
			Loader:     api.LoaderTS,
			ResolveDir: filepath.Dir(args.Path),
			PluginData: pluginData,
		}, nil
	})
}

// registerStyleHandler registers the style handler for Vue Single File Components.
// Loads the precompiled style part based on the index specified in URL parameters.
// Supports multiple style blocks within a single Vue component.
func registerStyleHandler(build *api.PluginBuild) {
	build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "sfc-style"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		pluginData := args.PluginData.(map[string]interface{})

		// Extract style index from URL query parameters
		parsedURL, _ := url.Parse(args.Path)
		params := parsedURL.Query()
		indexStr := params.Get("index")
		index, _ := strconv.Atoi(indexStr)

		// Load the specific style block by index
		styles := pluginData["styles"].([]map[string]interface{})
		styleResult := styles[index]
		code := styleResult["code"].(string)

		return api.OnLoadResult{
			Contents:   &code,
			Loader:     api.LoaderCSS,
			ResolveDir: filepath.Dir(args.Path),
			PluginData: pluginData,
		}, nil
	})
}
