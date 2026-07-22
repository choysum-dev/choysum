// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webplugin

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/parser/vueparser"
	"github.com/choysum-dev/choysum/internal/vueplugin"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/rs/xid"
	"golang.org/x/net/html"
)

// WebPlugin wires web parser results into the shared esbuild plugin flow.
type WebPlugin struct {
	*esbplugins.BasePlugin
	EntryPointImports []string
	IndexHtmlOutFile  string
	parserFactory     func(scope.Scope, *meta.IrModule) parser.Parser
	runtimeOptions    runtimeOptions
}

const (
	entryPointImportsMarker = "// __choysum_entrypoint_imports__"
	entryPointMountMarker   = "// __choysum_entrypoint_mount__"
)

// BindRuntimeState refreshes the plugin's runtime scope, module, and parser state.
func (p *WebPlugin) BindRuntimeState(runtimeScope scope.Scope, module *meta.IrModule) {
	if p == nil || p.BasePlugin == nil {
		return
	}
	if runtimeScope != nil {
		p.Env = runtimeScope
	}
	if module != nil {
		p.Module = module
	}
	if p.parserFactory == nil {
		if p.Parser != nil {
			if p.Env != nil {
				p.runtimeOptions = runtimeOptionsFromScope(p.Env)
			}
			return
		}
		p.parserFactory = vueparser.NewVueParser
	}
	p.Parser = p.parserFactory(p.Env, p.Module)
	if p.Env != nil {
		p.runtimeOptions = runtimeOptionsFromScope(p.Env)
	}
}

// UseParser replaces the parser factory with a concrete parser instance.
func (p *WebPlugin) UseParser(parser parser.Parser) {
	p.parserFactory = nil
	p.Parser = parser
}

func (p *WebPlugin) isEntryPointPath(path string) bool {
	if p.SameFilePath(path, p.EntryPoint) {
		return true
	}

	current := strings.TrimSpace(path)
	entry := strings.TrimSpace(p.EntryPoint)
	if current == "" || entry == "" {
		return false
	}
	return filepath.ToSlash(filepath.Clean(current)) == filepath.ToSlash(filepath.Clean(entry))
}

// HandleParserResults drains and stores parser results in build order.
func (p *WebPlugin) HandleParserResults() []*parser.ParserResult {
	results := p.BasePlugin.HandleParserResults()
	return results
}

// TsPlugin returns the TypeScript loader plugin for web entry points.
func (p *WebPlugin) TsPlugin() api.Plugin {
	return api.Plugin{
		Name: "choysum-web-ts",
		Setup: func(build api.PluginBuild) {
			build.OnLoad(api.OnLoadOptions{Filter: `\.ts$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				p.Mu.Lock()
				defer p.Mu.Unlock()
				content, err := p.handleTsFile(args, build)
				if err != nil {
					return api.OnLoadResult{}, err
				}

				return api.OnLoadResult{
					Contents: &content,
					Loader:   api.LoaderTS,
				}, nil
			})

			vueAppfilter := `.*\/web(\/|\/index(\.ts)?)?$`
			build.OnResolve(api.OnResolveOptions{Filter: vueAppfilter}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				p.Mu.Lock()
				defer p.Mu.Unlock()

				pathAlias, err := parser.ParseTsconfigPathAlias(build.InitialOptions)
				if err != nil {
					return api.OnResolveResult{}, err
				}
				resolvePath := parser.ApplyPathAlias(pathAlias, args.Path)

				if !filepath.IsAbs(resolvePath) {
					resolvePath = filepath.Join(args.ResolveDir, resolvePath)
				}

				possiablePaths := []string{
					resolvePath,
					resolvePath + ".ts",
					filepath.Join(resolvePath, "index.ts"),
					resolvePath + ".tsx",
					filepath.Join(resolvePath, "index.tsx"),
					resolvePath + ".d.ts",
					filepath.Join(resolvePath, "index.d.ts"),
				}
				for _, possiablePath := range possiablePaths {
					if info, err := os.Stat(possiablePath); err == nil && !info.IsDir() {
						resolvePath = possiablePath
						break
					}
				}

				var importerModuleName string
				runtimeOptions := p.resolvedRuntimeOptions()
				args.Importer = strings.ReplaceAll(args.Importer, "\\", "/")
				importerModuleName = strings.Split(strings.TrimPrefix(args.Importer, runtimeOptions.modulesPath+"/"), "/")[0]

				if args.Importer != "" {
					for _, result := range p.ParserResults {
						if resolvePath == result.Path {
							if slices.Contains(result.VueAppImportTree, resolvePath) {
								for i := 0; i < len(result.VueAppImportTree); i++ {
									if result.VueAppImportTree[i] == resolvePath {
										if i > 0 && !strings.Contains(result.VueAppImportTree[i-1], filepath.Join(runtimeOptions.modulesPath, importerModuleName)) {
											return api.OnResolveResult{Path: result.VueAppImportTree[i-1]}, nil
										}
									}
								}
							}
						}
					}
				}
				return api.OnResolveResult{Path: resolvePath}, nil
			})
		},
	}
}

func (p *WebPlugin) handleTsFile(args api.OnLoadArgs, build api.PluginBuild) (string, error) {
	var content string
	var err error
	cachedParserResult := p.FindParserResultByPath(args.Path)

	if cachedParserResult != nil {
		if cachedParserResult.Content != "" {
			content = cachedParserResult.Content
		} else {
			content = cachedParserResult.RawContent
		}
	} else {
		content, err = p.ReadNormalizedTextFile(args.Path)
		if err != nil {
			return "", err
		}
	}

	pathAlias, err := parser.ParseTsconfigPathAlias(build.InitialOptions)
	if err != nil {
		return "", err
	}

	parserResult, err := p.Parser.Parse(pathAlias, args.Path, content)
	if err != nil {
		p.Env.Logger().Error("typescript file parsing failed", "error", err)
		return "", err
	}

	if p.isEntryPointPath(args.Path) {
		var contentChanged bool
		content, contentChanged, err = p.injectEntryPointContent(args.Path, content, parserResult)
		if err != nil {
			return "", err
		}

		if contentChanged {
			parserResult, err = p.Parser.Parse(pathAlias, args.Path, content)
			if err != nil {
				p.Env.Logger().Error("typescript file parsing failed", "error", err)
				return "", err
			}
		}
	}
	parserResult.Content = content

	p.PublishParserResult(parserResult)

	return content, nil
}

func (p *WebPlugin) injectEntryPointContent(path string, content string, parserResult *parser.ParserResult) (string, bool, error) {
	changed := false

	if !strings.Contains(content, entryPointImportsMarker) {
		stores, others := p.splitEntryPointImports()
		var b strings.Builder
		b.WriteString(entryPointImportsMarker)
		b.WriteByte('\n')
		// Only store side-effect imports are prepended. Module web entrypoints stay
		// after the app re-export: putting them first creates a circular graph where
		// `createApp` from core/web/application is still undefined when app.ts runs.
		for _, importPath := range stores {
			b.WriteString("import '")
			b.WriteString(importPath)
			b.WriteString("';\n")
		}
		b.WriteString(content)
		if len(content) > 0 && !strings.HasSuffix(content, "\n") && len(others) > 0 {
			b.WriteByte('\n')
		}
		for _, importPath := range others {
			b.WriteString("import '")
			b.WriteString(importPath)
			b.WriteString("';\n")
		}
		content = b.String()
		changed = true
	}

	if !strings.Contains(content, entryPointMountMarker) {
		v, ok := parserResult.Exports["default"]
		if !ok {
			return "", false, fmt.Errorf("default export not found in %s", path)
		}
		randStr := xid.New().String()
		content += fmt.Sprintf("\n%s\nimport %s from '%s'\n%s.mount('#app')", entryPointMountMarker, randStr, v.ModuleSpecPath, randStr)
		changed = true
	}

	return content, changed, nil
}

func isStoreEntryPointImport(importPath string) bool {
	normalized := strings.ReplaceAll(strings.TrimSpace(importPath), "\\", "/")
	if normalized == "" {
		return false
	}
	if strings.Contains(normalized, "/api/web/") && strings.Contains(normalized, "/stores/") {
		return true
	}
	normalized = strings.TrimSuffix(normalized, ".ts")
	return strings.HasSuffix(normalized, "/stores/index")
}

func (p *WebPlugin) splitEntryPointImports() (stores, others []string) {
	if len(p.EntryPointImports) == 0 {
		return nil, nil
	}
	stores = make([]string, 0, len(p.EntryPointImports))
	others = make([]string, 0, len(p.EntryPointImports))
	for _, importPath := range p.EntryPointImports {
		if isStoreEntryPointImport(importPath) {
			stores = append(stores, importPath)
			continue
		}
		others = append(others, importPath)
	}
	return stores, others
}

func (p *WebPlugin) prioritizedEntryPointImports() []string {
	stores, others := p.splitEntryPointImports()
	if len(stores) == 0 && len(others) == 0 {
		return nil
	}
	// Keep original relative order within each bucket, then concatenate:
	// stores first, other app entry imports after.
	ordered := make([]string, 0, len(stores)+len(others))
	ordered = append(ordered, stores...)
	ordered = append(ordered, others...)
	return ordered
}

func (p *WebPlugin) htmlIconProcessor(pathPrefix string) vueplugin.IndexHtmlProcessor {
	return func(doc *html.Node, result *api.BuildResult, opts *vueplugin.Options, build *api.PluginBuild) error {
		// 1. Find the <link rel="icon" ...> tag.
		iconNode := htmlquery.FindOne(doc, `//link[@rel="icon"]`)
		if iconNode == nil {
			p.Env.Logger().Warn("favicon processing skipped", "reason", "icon_link_tag_not_found")
			return nil
		}

		// 2. Read the href attribute.
		var href string
		var hrefAttrIndex int = -1
		for i, attr := range iconNode.Attr {
			if attr.Key == "href" {
				href = attr.Val
				hrefAttrIndex = i
				break
			}
		}
		if href == "" {
			p.Env.Logger().Warn("favicon processing skipped", "reason", "icon_link_tag_missing_href")
			return nil
		}

		// 3. Build the absolute path to the source file.
		htmlDir := filepath.Dir(opts.IndexHtmlOptions.SourceFile)
		iconSrcPath := filepath.Join(htmlDir, href)

		// Check whether the file exists.
		if _, err := os.Stat(iconSrcPath); err != nil {
			p.Env.Logger().Warn("favicon file does not exist", "path", iconSrcPath, "href", href)
			return nil
		}

		// 4. Copy the file into the output directory.
		outdir := build.InitialOptions.Outdir
		if outdir == "" {
			return fmt.Errorf("build.InitialOptions.Outdir is empty")
		}

		// Write the favicon into the output directory using its basename.
		iconDstPath := filepath.Join(outdir, filepath.Base(href))

		// Ensure the destination directory exists.
		if err := os.MkdirAll(filepath.Dir(iconDstPath), 0755); err != nil {
			return fmt.Errorf("failed to create favicon destination directory: %v", err)
		}

		// Read the source file.
		iconData, err := os.ReadFile(iconSrcPath)
		if err != nil {
			return fmt.Errorf("failed to read favicon file: %v", err)
		}

		// Write the destination file, overwriting it if it already exists.
		if err := os.WriteFile(iconDstPath, iconData, 0644); err != nil {
			return fmt.Errorf("failed to write favicon file: %v", err)
		}

		// 5. Rewrite the HTML href and prepend pathPrefix.
		newHref := pathPrefix + strings.TrimPrefix(href, "/")
		if hrefAttrIndex >= 0 {
			iconNode.Attr[hrefAttrIndex].Val = newHref
		}

		return nil
	}
}

// securityHtmlProcessor returns an HTML processor that injects security-related headers.
// It adjusts policy strength based on the runtime environment.
func (p *WebPlugin) securityHtmlProcessor() vueplugin.IndexHtmlProcessor {
	return func(doc *html.Node, result *api.BuildResult, opts *vueplugin.Options, build *api.PluginBuild) error {
		headNode := htmlquery.FindOne(doc, "//head")
		if headNode == nil {
			return fmt.Errorf("head tag not found in HTML document")
		}

		// Read the relevant settings from runtime configuration.
		runtimeOptions := p.resolvedRuntimeOptions()
		environment := runtimeOptions.serverEnvironment
		useHTTPS := runtimeOptions.serverEnabledTLS
		isProd := environment == "production" || environment == "prod"

		p.Env.Logger().Debug(
			"security html headers configuring",
			"environment", environment,
			"use_https", useHTTPS,
			"is_prod", isProd,
		)

		// Build the CSP policy.
		// frame-ancestors is omitted here because browsers ignore it in meta tags.
		var cspValue string

		if isProd {
			// Production uses the strictest CSP.
			if useHTTPS {
				// HTTPS production can also enable upgrade-insecure-requests and HSTS.
				cspValue = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
					"font-src 'self'; connect-src 'self'; media-src 'self'; object-src 'none'; " +
					"child-src 'none'; frame-src 'none'; worker-src 'self'; " +
					"form-action 'self'; upgrade-insecure-requests; block-all-mixed-content"

				// Add the HSTS header.
				hstsNode := &html.Node{
					Type: html.ElementNode,
					Data: "meta",
					Attr: []html.Attribute{
						{Key: "http-equiv", Val: "Strict-Transport-Security"},
						{Key: "content", Val: "max-age=31536000; includeSubDomains; preload"},
					},
				}
				headNode.AppendChild(hstsNode)
			} else {
				// Production without HTTPS.
				cspValue = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
					"font-src 'self'; connect-src 'self'; media-src 'self'; object-src 'none'; " +
					"child-src 'none'; frame-src 'none'; worker-src 'self'; " +
					"form-action 'self'"
			}
		} else {
			// Development keeps a looser CSP for hot reload and similar tooling.
			cspValue = "default-src 'self'; script-src 'self' 'unsafe-eval' 'unsafe-inline'; " +
				"style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; " +
				"connect-src 'self' ws: wss:; media-src 'self'; object-src 'none'; " +
				"child-src 'none'; frame-src 'none'; worker-src 'self' blob:; " +
				"form-action 'self'"
		}

		// Add the CSP header.
		cspNode := &html.Node{
			Type: html.ElementNode,
			Data: "meta",
			Attr: []html.Attribute{
				{Key: "http-equiv", Val: "Content-Security-Policy"},
				{Key: "content", Val: cspValue},
			},
		}
		headNode.AppendChild(cspNode)

		// Add the XSS protection header.
		xssNode := &html.Node{
			Type: html.ElementNode,
			Data: "meta",
			Attr: []html.Attribute{
				{Key: "http-equiv", Val: "X-XSS-Protection"},
				{Key: "content", Val: "1; mode=block"},
			},
		}
		headNode.AppendChild(xssNode)

		// Add the content type options header.
		ctNode := &html.Node{
			Type: html.ElementNode,
			Data: "meta",
			Attr: []html.Attribute{
				{Key: "http-equiv", Val: "X-Content-Type-Options"},
				{Key: "content", Val: "nosniff"},
			},
		}
		headNode.AppendChild(ctNode)

		// Add the Referrer-Policy header.
		var referrerValue string
		if isProd {
			referrerValue = "strict-origin-when-cross-origin"
		} else {
			referrerValue = "origin-when-cross-origin"
		}

		rpNode := &html.Node{
			Type: html.ElementNode,
			Data: "meta",
			Attr: []html.Attribute{
				{Key: "name", Val: "referrer"},
				{Key: "content", Val: referrerValue},
			},
		}
		headNode.AppendChild(rpNode)

		// Add the Permissions-Policy header in production.
		if isProd {
			ppNode := &html.Node{
				Type: html.ElementNode,
				Data: "meta",
				Attr: []html.Attribute{
					{Key: "http-equiv", Val: "Permissions-Policy"},
					{Key: "content", Val: "camera=(), microphone=(), geolocation=(), interest-cohort=()"},
				},
			}
			headNode.AppendChild(ppNode)
		}

		p.Env.Logger().Debug("security html headers configured",
			"environment", environment,
			"is_prod", isProd,
			"use_https", useHTTPS)

		return nil
	}
}

// VueResolveProcessor returns the Vue component resolve processor.
func (p *WebPlugin) VueResolveProcessor() vueplugin.OnVueResolveProcessor {
	return func(args *api.OnResolveArgs, buildOptions *api.BuildOptions) (*api.OnResolveResult, error) {
		p.Mu.Lock()
		defer p.Mu.Unlock()

		pathAlias, err := parser.ParseTsconfigPathAlias(buildOptions)
		if err != nil {
			return nil, err
		}
		args.Path = parser.ApplyPathAlias(pathAlias, args.Path)
		finalChildPath := parser.FindVueComponentFinalChild(p.ParserResults, args.Importer, args.Path)
		if finalChildPath != "" && finalChildPath != args.Path {
			args.Path = finalChildPath
		}

		return nil, nil
	}
}

// VueLoadProcessor returns the Vue component load processor.
func (p *WebPlugin) VueLoadProcessor() vueplugin.OnVueLoadProcessor {
	return func(content string, args api.OnLoadArgs, buildOptions *api.BuildOptions) (string, error) {
		p.Mu.Lock()
		defer p.Mu.Unlock()
		parserResult := p.FindParserResultByPath(args.Path)

		if parserResult != nil {
			if parserResult.Content != "" {
				content = parserResult.Content
			} else {
				content = parserResult.RawContent
			}
		}

		pathAlias, err := parser.ParseTsconfigPathAlias(buildOptions)
		if err != nil {
			return "", err
		}

		parserResult, err = p.Parser.Parse(pathAlias, args.Path, content)
		if err != nil {
			return "", err
		}

		p.PublishParserResult(parserResult)

		return content, nil
	}
}

// BuildEndProcessor returns the processor that removes stale build output files.
func (p *WebPlugin) BuildEndProcessor() vueplugin.OnEndProcessor {
	return func(result *api.BuildResult, buildOptions *api.BuildOptions) error {
		p.Mu.Lock()
		defer p.Mu.Unlock()
		if buildOptions == nil || !buildOptions.Write {
			return nil
		}
		if buildOptions.Outdir == "" {
			return nil
		}
		// clean Outdir
		outputFiles := make(map[string]bool)
		for _, outputFile := range result.OutputFiles {
			outputFiles[outputFile.Path] = true
		}

		err := filepath.Walk(buildOptions.Outdir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if !outputFiles[path] {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove file %s: %v", path, err)
				}
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("error cleaning Outdir: %v", err)
		}

		return nil
	}
}

// DefinePlugins returns the web plugin chain for the current runtime state.
func (p *WebPlugin) DefinePlugins(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.IrModule, options ...esbplugins.EsbPluginOptions) []api.Plugin {
	for _, opt := range options {
		if opt != nil {
			opt(p)
		}
	}
	p.BindRuntimeState(runtimeScope, module)
	runtimeOptions := p.resolvedRuntimeOptions()

	dist_path := runtimeOptions.distPath
	webBaseUrl := strings.TrimSuffix(runtimeOptions.webBaseURL, "/") + "/"
	htmlSourceFile := filepath.Join(runtimeOptions.modulesPath, "web", "web", "index.html")
	htmlOutFile := p.IndexHtmlOutFile
	if htmlOutFile == "" {
		htmlOutFile = filepath.Join(dist_path, "web", "index.html")
	}

	// Use the generic plugin constructor and replace the single processor with a chain.
	vuePlugin := vueplugin.NewPlugin(
		vueplugin.WithName("choysum-web-vue"),
		vueplugin.WithJsExecutor(jsExecutor),
		vueplugin.WithIndexHtmlOptions(vueplugin.IndexHtmlOptions{
			SourceFile:      htmlSourceFile,
			OutFile:         htmlOutFile,
			RemoveTagXPaths: []string{"//script[@src='/src/main.ts']"},
			// Apply the HTML processor chain: security headers first, then asset rewriting.
			IndexHtmlProcessors: []vueplugin.IndexHtmlProcessor{
				p.securityHtmlProcessor(),
				p.htmlIconProcessor(webBaseUrl),
				vueplugin.DefaultHtmlProcessor(&vueplugin.HtmlProcessorOptions{
					PathPrefix: webBaseUrl,
				}),
			},
		}),

		vueplugin.WithLogger(runtimeScope.Logger()),
		vueplugin.WithOnVueResolveProcessor(p.VueResolveProcessor()),
		vueplugin.WithOnVueLoadProcessor(p.VueLoadProcessor()),
		vueplugin.WithOnEndProcessor(p.BuildEndProcessor()),
	)

	return []api.Plugin{
		vuePlugin,
		p.TsPlugin(),
	}
}

func (p *WebPlugin) replaceModuleSpecReferenceIdent(parserResults []*parser.ParserResult) error {
	for _, parserResult := range parserResults {
		for _, componentsProperty := range parserResult.VueComponentsPropertys {
			moduleSpec, referenceIdent := p.FindModuleSpecAndReferenceIdent(componentsProperty.ModuleSpecPath, componentsProperty.ReferenceIdent)
			if moduleSpec != "" {
				componentsProperty.ModuleSpecPath = moduleSpec
				componentsProperty.ReferenceIdent = referenceIdent
			}
		}
		if parserResult.VueExtendsProperty != nil {
			moduleSpec, referenceIdent := p.FindModuleSpecAndReferenceIdent(parserResult.VueExtendsProperty.ModuleSpecPath, parserResult.VueExtendsProperty.ReferenceIdent)
			if moduleSpec != "" {
				parserResult.VueExtendsProperty.ModuleSpecPath = moduleSpec
				parserResult.VueExtendsProperty.ReferenceIdent = referenceIdent
				parserResult.VueComponent.RawExtends = moduleSpec
				parserResult.VueComponent.Extends = moduleSpec
			}
		}
	}
	return nil
}

// GetParserResults finalizes parser results after module-spec normalization.
func (p *WebPlugin) GetParserResults() ([]*parser.ParserResult, error) {
	results := p.HandleParserResults()
	err := p.replaceModuleSpecReferenceIdent(results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// NewWebPlugin creates a web esbuild plugin for a module entry point.
func NewWebPlugin(runtimeScope scope.Scope, module *meta.IrModule, entryPoint string, opts ...func(*WebPlugin)) esbplugins.EsbPlugin {
	p := &WebPlugin{
		BasePlugin:    esbplugins.NewBasePlugin(runtimeScope, module, entryPoint),
		parserFactory: vueparser.NewVueParser,
	}

	for _, opt := range opts {
		opt(p)
	}
	p.BindRuntimeState(runtimeScope, module)

	return p
}

// WithParser overrides the parser used by WebPlugin.
func WithParser(parser parser.Parser) func(*WebPlugin) {
	return func(p *WebPlugin) {
		p.UseParser(parser)
	}
}

// SetEntryPointImports stores imports that should be prepended to the web entry point.
func (p *WebPlugin) SetEntryPointImports(imports []string) {
	p.EntryPointImports = append([]string(nil), imports...)
}

// SetIndexHtmlOutFile overrides the generated index.html output path.
func (p *WebPlugin) SetIndexHtmlOutFile(outFile string) {
	p.IndexHtmlOutFile = outFile
}
