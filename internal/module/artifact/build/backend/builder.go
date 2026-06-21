// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendbuilder

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	internalbackendplugin "github.com/choysum-dev/choysum/internal/esbplugins/backendplugin"
	"github.com/choysum-dev/choysum/internal/esmresolver"
	modulegenerator "github.com/choysum-dev/choysum/internal/module/artifact/generate"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/parser/backendtsparser"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

type ModuleBuilder struct {
	runtimeScope   scope.Scope
	runtimeOptions runtimeOptions
	jsExecutor     jsexecutor.ScriptExecutor
	module         *meta.IrModule
	entryPoint     string
	buildPlugin    esbplugins.EsbPlugin
	prebuildPlugin esbplugins.EsbPlugin
	publishDist    bool

	// Optional override for dist output file name (default: index.js).
	outFileName string

	// Optional override for esbuild IIFE global name (default: module.ApplicationStr).
	globalName string

	// Optional override for pipeline-managed staging.
	distAppDirOverride string

	// Cached parser and path alias for refresh reparsing.
	tsParser        parser.Parser
	tsParserFactory func(scope.Scope, *meta.IrModule) parser.Parser
	tsPathAlias     map[string]string
}

func (b *ModuleBuilder) bindRuntimeState(ctx context.Context) func() {
	if b == nil || b.runtimeScope == nil || ctx == nil {
		return func() {}
	}
	runtimeCtx := ctx
	if _, ok := scope.TransactionFromContext(runtimeCtx); !ok {
		if inheritedTx, ok := scope.TransactionFromContext(b.runtimeScope.Context()); ok && inheritedTx != nil {
			runtimeCtx = scope.ContextWithTransaction(runtimeCtx, inheritedTx)
		}
	}
	runtimeScope := b.runtimeScope.WithContext(runtimeCtx)
	if runtimeScope == nil {
		return func() {}
	}
	prevScope := b.runtimeScope
	prevParser := b.tsParser
	prevAlias := b.tsPathAlias
	b.runtimeScope = runtimeScope
	b.tsPathAlias = nil
	if b.tsParserFactory != nil {
		b.tsParser = b.tsParserFactory(b.runtimeScope, b.module)
	}
	return func() {
		b.runtimeScope = prevScope
		b.tsParser = prevParser
		b.tsPathAlias = prevAlias
	}
}

func (b *ModuleBuilder) buildOptions(prebuild bool) *api.BuildOptions {
	runtimeOptions := b.resolvedRuntimeOptions()
	modules_path := runtimeOptions.modulesPath
	dist_path := runtimeOptions.distPath
	tsconfigPath := filepath.Join(modules_path, ".", "tsconfig.json")
	if err := esmresolver.UpdateTsconfigPaths(tsconfigPath, nil); err != nil {
		if b.runtimeScope != nil {
			b.runtimeScope.Logger().Warn("backend build: ensure modules tsconfig failed", "path", tsconfigPath, "error", err)
		}
	}
	outName := strings.TrimSpace(b.outFileName)
	if outName == "" {
		outName = "index.js"
	}
	outFile := filepath.Join(dist_path, "apps", b.module.ApplicationStr, outName)
	if b.distAppDirOverride != "" {
		outFile = filepath.Join(b.distAppDirOverride, outName)
	}

	buildOptions := api.BuildOptions{
		EntryPoints: []string{b.entryPoint},
		Outfile:     outFile,
		Tsconfig:    tsconfigPath,
		Loader: map[string]api.Loader{
			".proto": api.LoaderText,
		},
		Format:     api.FormatIIFE,
		GlobalName: b.globalName,
		Platform:   api.PlatformBrowser,
		Bundle:     true,
		Write:      true,
		Metafile:   false,
	}

	backendEnv := map[string]any{}
	for k, v := range runtimeOptions.backendEnv {
		backendEnv[k] = v
	}

	backendEnv["CHOYSUM_GRPC_AUTHENTICATION_ENABLED"] = runtimeOptions.grpcAuthentication
	backendEnv["CHOYSUM_GRPC_METHOD_ACCESS_ENABLED"] = runtimeOptions.grpcMethodAccess
	backendEnv["CHOYSUM_GRPC_RECORD_RULE_ENABLED"] = runtimeOptions.grpcRecordRule
	backendEnv["CHOYSUM_GRPC_COMPANY_FILTER_ENABLED"] = runtimeOptions.grpcCompanyFilter
	backendEnv["CHOYSUM_GRPC_FIELD_RULE_ENABLED"] = runtimeOptions.grpcFieldRule
	backendEnv["CHOYSUM_AUTHZ_DECISION_LOG"] = runtimeOptions.authzDecisionLog
	backendEnv["CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED"] = runtimeOptions.authzDecisionAudit
	if runtimeOptions.taskDefaultMaxAttempt > 0 {
		backendEnv["CHOYSUM_TASK_DEFAULT_MAX_ATTEMPTS"] = runtimeOptions.taskDefaultMaxAttempt
	}

	jsonBackendEnv, err := json.Marshal(backendEnv)
	if err != nil {
		jsonBackendEnv = []byte("{}")
		b.runtimeScope.Logger().Warn("backend env marshal failed", "error", err)
	}

	buildOptions.Define = map[string]string{
		"import.meta.env": string(jsonBackendEnv),
	}

	// skip if no application string, like the choysum module
	if b.module.ApplicationStr == "" {
		buildOptions.Write = false
	}

	if runtimeOptions.compileSourceMap {
		buildOptions.Sourcemap = api.SourceMapInline
	} else {
		buildOptions.Sourcemap = api.SourceMapNone
	}

	buildOptions.MinifyWhitespace = runtimeOptions.compileMinify
	buildOptions.MinifyIdentifiers = runtimeOptions.compileMinify
	buildOptions.MinifySyntax = runtimeOptions.compileMinify

	if runtimeOptions.compileTreeShaking {
		buildOptions.TreeShaking = api.TreeShakingTrue
	} else {
		buildOptions.TreeShaking = api.TreeShakingFalse
	}

	esbOpts := []esbplugins.EsbPluginOptions{
		esbplugins.WithEntryPointImports(b.entryPointImports()),
	}

	if prebuild {
		basePlugins := b.prebuildPlugin.DefinePlugins(b.runtimeScope, b.jsExecutor, b.module, esbOpts...)
		// Prepend the ESM resolver so bare imports are intercepted before
		// other plugins process them.
		resolverOpts := []esmresolver.Option{
			esmresolver.WithCacheDir(runtimeOptions.defaultChoysumPath),
			esmresolver.WithTarget("es2020"),
			esmresolver.WithModulePath(b.module.Path),
		}
		if b.runtimeScope != nil {
			resolverOpts = append(resolverOpts, esmresolver.WithLogger(b.runtimeScope.Logger()))
		}
		if upstream := strings.TrimSpace(runtimeOptions.esmUpstreamURL); upstream != "" {
			resolverOpts = append(resolverOpts, esmresolver.WithUpstream(upstream))
		}
		buildOptions.Plugins = append([]api.Plugin{esmresolver.New(resolverOpts...).Plugin()}, basePlugins...)
		buildOptions.Write = false
	} else {
		basePlugins := b.buildPlugin.DefinePlugins(b.runtimeScope, b.jsExecutor, b.module, esbOpts...)
		// Prepend the ESM resolver so bare imports are intercepted before
		// other plugins process them.
		resolverOpts := []esmresolver.Option{
			esmresolver.WithCacheDir(runtimeOptions.defaultChoysumPath),
			esmresolver.WithTarget("es2020"),
			esmresolver.WithModulePath(b.module.Path),
		}
		if b.runtimeScope != nil {
			resolverOpts = append(resolverOpts, esmresolver.WithLogger(b.runtimeScope.Logger()))
		}
		if upstream := strings.TrimSpace(runtimeOptions.esmUpstreamURL); upstream != "" {
			resolverOpts = append(resolverOpts, esmresolver.WithUpstream(upstream))
		}
		buildOptions.Plugins = append([]api.Plugin{esmresolver.New(resolverOpts...).Plugin()}, basePlugins...)
		// Do not let esbuild write directly to dist; we publish outputs atomically.
		buildOptions.Write = false
	}

	return &buildOptions
}

func (b *ModuleBuilder) entryPointImports() []string {
	imports := make([]string, 0)
	if b == nil || b.runtimeScope == nil || b.runtimeScope.Session() == nil {
		return imports
	}
	runtimeOptions := b.resolvedRuntimeOptions()

	var installModules []*meta.IrModule
	modulePath := ""
	if b.module != nil {
		modulePath = b.module.Path
	}
	if result := b.runtimeScope.Session().
		Where("status = ?", meta.Installed).
		Order("id DESC").
		Find(&installModules); result.Error != nil {
		b.runtimeScope.Logger().Error("backend entrypoint import modules lookup failed", "module_path", modulePath, "error", result.Error)
		return imports
	}

	seen := make(map[string]struct{}, len(installModules))
	for _, m := range installModules {
		if m == nil {
			continue
		}
		if strings.TrimSpace(m.ServiceEntryPoint) == "" {
			continue
		}

		appName := strings.TrimSpace(m.ApplicationStr)
		if appName == "" {
			continue
		}

		_, _, workspaceServiceDir, err := modulegenerator.WorkspaceGeneratedAPITargets(runtimeOptions.modulesPath, appName, runtimeOptions.defaultChoysumPath)
		if err != nil {
			continue
		}
		workspaceServiceEntry := filepath.Join(workspaceServiceDir, "index.ts")
		if _, err := os.Stat(workspaceServiceEntry); err != nil {
			continue
		}

		importPath := workspaceServiceEntry
		if absPath, err := filepath.Abs(importPath); err == nil {
			importPath = absPath
		}
		importPath = filepath.ToSlash(filepath.Clean(importPath))
		if _, ok := seen[importPath]; ok {
			continue
		}
		seen[importPath] = struct{}{}
		imports = append(imports, importPath)
	}

	return imports
}

func (b *ModuleBuilder) getNewExtends(model *meta.IrModel) (*meta.IrModel, error) {
	if model.Extends == "" {
		return nil, nil
	}

	var extendsModels []*meta.IrModel
	if result := b.runtimeScope.Session().
		Where(&meta.IrModel{Name: model.Name}).
		Order("id DESC").
		Find(&extendsModels); result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, xfmt.Errorf("error getting last models: %w", result.Error)
		}
		return nil, nil
	}

	if len(extendsModels) > 0 {
		extendsModelPaths := make([]string, len(extendsModels))
		for i, m := range extendsModels {
			extendsModelPaths[i] = m.Path
		}

		if slices.Contains(extendsModelPaths, model.Extends) {
			lastModel := extendsModels[0]
			if lastModel.Path != model.Path && lastModel.Path != model.Extends {
				return lastModel, nil
			}
		}
	}

	return nil, nil

}

func (b *ModuleBuilder) updatePrebuildResult(buildResult *module.BuildResult) error {
	for _, parseResult := range module.ParserResults(buildResult) {
		if parseResult.Model == nil || parseResult.Model.Extends == "" {
			continue
		}

		extendedModel, err := b.getNewExtends(parseResult.Model)
		if err != nil {
			return xfmt.Errorf("error getting new extends: %w", err)
		}

		if extendedModel == nil {
			continue
		}

		if err := b.refreshModelExtendsProperty(parseResult); err != nil {
			return xfmt.Errorf("error refreshing model extends property: %w", err)
		}

		parseResult.Model.Extends = extendedModel.Path

		if err := b.updateModelExtends(parseResult, extendedModel); err != nil {
			return xfmt.Errorf("error updating model extends: %w", err)
		}
	}
	return nil
}

func (b *ModuleBuilder) refreshModelExtendsProperty(parseResult *parser.ParserResult) error {
	if parseResult == nil || parseResult.Model == nil || parseResult.Path == "" {
		return nil
	}
	if b.runtimeScope == nil {
		return nil
	}
	tsParser, tsPathAlias, err := b.getTsParserAndPathAlias()
	if err != nil {
		return xfmt.Errorf("prepare ts parser context for %s: %w", parseResult.Path, err)
	}

	content := parseResult.RawContent
	if parseResult.Content != "" {
		content = parseResult.Content
	}

	reparsed, err := tsParser.Parse(tsPathAlias, parseResult.Path, content)
	if err != nil {
		return xfmt.Errorf("reparse content for %s: %w", parseResult.Path, err)
	}
	if reparsed == nil {
		return xfmt.Errorf("reparse content for %s returned nil result", parseResult.Path)
	}
	if reparsed.ModelExtendsProperty == nil {
		return xfmt.Errorf("model extends property missing after reparse for %s", parseResult.Path)
	}

	parseResult.ModelExtendsProperty = reparsed.ModelExtendsProperty
	parseResult.Imports = reparsed.Imports
	return nil
}

func (b *ModuleBuilder) getTsParserAndPathAlias() (parser.Parser, map[string]string, error) {
	if b.tsParser == nil {
		factory := b.tsParserFactory
		if factory == nil {
			factory = backendtsparser.NewTsParser
		}
		b.tsParser = factory(b.runtimeScope, b.module)
	}

	if b.tsPathAlias == nil {
		runtimeOptions := b.resolvedRuntimeOptions()
		pathAlias, err := parser.ParseTsconfigPathAlias(&api.BuildOptions{
			Tsconfig: filepath.Join(runtimeOptions.modulesPath, ".", "tsconfig.json"),
		})
		if err != nil {
			return nil, nil, err
		}
		b.tsPathAlias = pathAlias
	}

	return b.tsParser, b.tsPathAlias, nil
}

func deterministicModelAlias(modelPath string, parentModelPath string) string {
	seed := strings.TrimSpace(modelPath) + "|" + strings.TrimSpace(parentModelPath)
	h := sha1.Sum([]byte(seed))
	return "model_" + hex.EncodeToString(h[:])[:12]
}

func normalizeModuleSpecPath(path string) string {
	return strings.TrimSuffix(strings.TrimSpace(path), ".ts")
}

func findDefaultImportIdentifierByModulePath(imports map[string]*parser.Import, moduleSpecPath string) (string, bool, bool) {
	if len(imports) == 0 {
		return "", false, false
	}

	normalizedTarget := normalizeModuleSpecPath(moduleSpecPath)
	foundModulePath := false
	for localIdent, imp := range imports {
		if imp == nil {
			continue
		}
		if normalizeModuleSpecPath(imp.ModuleSpecPath) != normalizedTarget {
			continue
		}
		foundModulePath = true
		if imp.ReferenceIdent == "default" && strings.TrimSpace(localIdent) != "" {
			return localIdent, true, true
		}
	}

	return "", foundModulePath, false
}

func insertImportIntoImportRegion(content string, imports map[string]*parser.Import, importStmt string) string {
	insertPos := 0
	for _, imp := range imports {
		if imp == nil {
			continue
		}
		if imp.End > insertPos && imp.End <= len(content) {
			insertPos = imp.End
		}
	}

	if insertPos <= 0 {
		if strings.HasPrefix(content, "\n") {
			return importStmt + content
		}
		return importStmt + "\n" + content
	}

	before := content[:insertPos]
	after := content[insertPos:]
	sepBefore := ""
	if !strings.HasSuffix(before, "\n") {
		sepBefore = "\n"
	}
	sepAfter := ""
	if !strings.HasPrefix(after, "\n") {
		sepAfter = "\n"
	}

	return before + sepBefore + importStmt + sepAfter + after
}

func (b *ModuleBuilder) updateModelExtends(parseResult *parser.ParserResult, extendedModel *meta.IrModel) error {
	// Skip if model extends is the same as raw extends
	if parseResult.Model.RawExtends == parseResult.Model.Extends {
		return nil
	}

	extendsClassName := deterministicModelAlias(parseResult.Path, extendedModel.Path)
	needAddImport := true
	if existingDefaultImportIdent, foundModulePath, hasDefaultImport := findDefaultImportIdentifierByModulePath(parseResult.Imports, extendedModel.Path); foundModulePath {
		if hasDefaultImport {
			extendsClassName = existingDefaultImportIdent
			needAddImport = false
		} else {
			return xfmt.Errorf("existing import from %s has no default identifier to reuse", extendedModel.Path)
		}
	}

	// Prepare new extends statement
	newExtendsStmt := fmt.Sprintf("extends %s", extendsClassName)
	newImportStmt := fmt.Sprintf("import %s from '%s';", extendsClassName, extendedModel.Path)

	if parseResult.ModelExtendsProperty == nil {
		return xfmt.Errorf("model extends property is nil for %s", parseResult.Path)
	}

	// Keep original content unchanged so parser source offsets remain valid.
	var content string
	if parseResult.Content != "" {
		content = parseResult.Content
	} else {
		content = parseResult.RawContent
	}

	replaced := false
	prop := parseResult.ModelExtendsProperty
	extendsText := strings.TrimSpace(prop.Text)

	// Primary path: replace by parser byte range for the extends node.
	if prop.Start >= 0 && prop.End > prop.Start && prop.End <= len(content) {
		rawSnippet := content[prop.Start:prop.End]
		snippet := strings.TrimSpace(rawSnippet)
		normalizeExtendsText := func(v string) string {
			return strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
		}
		// Offsets may be stale if content was edited after parse (e.g. decorator injection).
		// Require normalized snippet equality with parser-captured extends text to avoid false matches.
		if normalizeExtendsText(extendsText) != "" && normalizeExtendsText(snippet) == normalizeExtendsText(extendsText) {
			leadingWSLen := len(rawSnippet) - len(strings.TrimLeft(rawSnippet, " \t\r\n"))
			trailingWSLen := len(rawSnippet) - len(strings.TrimRight(rawSnippet, " \t\r\n"))
			replacement := rawSnippet[:leadingWSLen] + newExtendsStmt + rawSnippet[len(rawSnippet)-trailingWSLen:]
			content = content[:prop.Start] + replacement + content[prop.End:]
			replaced = true
		}
	}

	if !replaced {
		return xfmt.Errorf("failed to rewrite model extends for %s", parseResult.Path)
	}

	// Add import statement only when module path is not already imported.
	if needAddImport {
		content = insertImportIntoImportRegion(content, parseResult.Imports, newImportStmt)
	}

	parseResult.Content = content
	return nil
}

func (b *ModuleBuilder) prebuild() (*module.BuildResult, error) {
	if b.entryPoint == "" {
		return module.WithParserResults(&module.BuildResult{
			Module:        b.module,
			EsbuildResult: &api.BuildResult{},
		}, []*parser.ParserResult{}), nil
	}

	result := api.Build(*b.buildOptions(true))
	if len(result.Errors) > 0 {
		var err error
		for _, v := range result.Errors {
			if err == nil {
				if v.Location != nil {
					err = xfmt.Errorf("%s:%d:%d %s, `%s`", v.Location.File, v.Location.Line, v.Location.Column, v.Text, v.Location.LineText)
				} else if v.Text == "" {
					err = xfmt.Errorf("Unknown error")
				} else {
					err = xfmt.Errorf("%s", v.Text)
				}
			} else {
				err = xfmt.Errorf("%s : %w", v.Text, err)
			}
		}
		return nil, err
	}

	parserResults, err := b.prebuildPlugin.GetParserResults()
	if err != nil {
		return nil, xfmt.Errorf("Error getting parser results: %w", err)
	}
	return module.WithParserResults(&module.BuildResult{
		Module:        b.module,
		EsbuildResult: &result,
	}, parserResults), nil
}

func (b *ModuleBuilder) build(prebuildResult *module.BuildResult) (*module.BuildResult, error) {
	if b.entryPoint == "" {
		return prebuildResult, nil
	}

	b.buildPlugin.SetParserResults(module.ParserResults(prebuildResult))
	result := api.Build(*b.buildOptions(false))

	if len(result.Errors) > 0 {
		var err error
		for _, v := range result.Errors {
			if err == nil {
				if v.Location != nil {
					err = xfmt.Errorf("%s:%d:%d %s, `%s`", v.Location.File, v.Location.Line, v.Location.Column, v.Text, v.Location.LineText)
				} else if v.Text == "" {
					err = xfmt.Errorf("Unknown error")
				} else {
					err = xfmt.Errorf("%s", v.Text)
				}
			} else {
				err = xfmt.Errorf("%s : %w", v.Text, err)
			}
		}
		return nil, err
	}

	parserResults, err := b.buildPlugin.GetParserResults()
	if err != nil {
		return nil, xfmt.Errorf("Error getting parser results: %w", err)
	}

	// Publish build outputs atomically for application bundles.
	// NOTE: dist/apps/<app>/assets is reserved for proto files and is published separately.
	if b.publishDist && b.module.ApplicationStr != "" {
		for _, out := range result.OutputFiles {
			if out.Path == "" {
				continue
			}
			if err := staging.WriteFileAtomic(out.Path, out.Contents, 0o644); err != nil {
				return nil, xfmt.Errorf("publish build output %s: %w", out.Path, err)
			}
		}
		// Ensure dist/apps/<app> exists even if OutputFiles is empty (defensive).
		runtimeOptions := b.resolvedRuntimeOptions()
		distDir := filepath.Join(runtimeOptions.distPath, "apps", b.module.ApplicationStr)
		if b.distAppDirOverride != "" {
			distDir = b.distAppDirOverride
		}
		_ = os.MkdirAll(distDir, 0o755)
	}

	return module.WithParserResults(&module.BuildResult{
		Module:        b.module,
		EsbuildResult: &result,
	}, parserResults), nil
}

// Check inheritance relationship for models with the same name
func (b *ModuleBuilder) checkInheritanceChain(models []*meta.IrModel, pathModelMap map[string]*meta.IrModel) error {
	if len(models) <= 1 {
		return nil
	}

	byPath := make(map[string]*meta.IrModel, len(models))
	adj := make(map[string][]string, len(models))
	for _, m := range models {
		byPath[m.Path] = m
		adj[m.Path] = []string{}
	}

	for _, m := range models {
		if m.Extends == "" {
			continue
		}
		parent, ok := pathModelMap[m.Extends]
		if !ok || parent == nil || parent.Name != m.Name {
			continue
		}
		if _, ok := byPath[parent.Path]; !ok {
			continue
		}
		adj[m.Path] = append(adj[m.Path], parent.Path)
		adj[parent.Path] = append(adj[parent.Path], m.Path)
	}

	required := make(map[string]bool, len(models))
	for _, m := range models {
		if m.Extends != "" || len(adj[m.Path]) > 0 {
			required[m.Path] = true
		}
	}

	if len(required) <= 1 {
		return nil
	}

	visited := make(map[string]bool, len(models))
	start := ""
	for _, m := range models {
		if required[m.Path] {
			start = m.Path
			break
		}
	}
	if start == "" {
		return nil
	}
	queue := []string{start}
	visited[start] = true
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		for _, next := range adj[path] {
			if visited[next] {
				continue
			}
			visited[next] = true
			queue = append(queue, next)
		}
	}

	for _, m := range models {
		if required[m.Path] && !visited[m.Path] {
			return xfmt.Errorf("model %s(extends=%s) and %s(extends=%s) have same name but are not in the same inheritance component",
				start, byPath[start].Extends, m.Path, m.Extends)
		}
	}

	return nil
}

func (b *ModuleBuilder) checkCircularDependency(
	model *meta.IrModel,
	pathModelMap map[string]*meta.IrModel,
	visited map[string]bool,
) error {
	// If the model is nil or has no inheritance relationship, return nil
	if model == nil || model.Extends == "" {
		return nil
	}

	// Create a unique identifier for the model using the format "name:path"
	modelKey := fmt.Sprintf("%s:%s", model.Name, model.Path)

	// If the model has already been visited, a circular dependency is detected
	if visited[modelKey] {
		return xfmt.Errorf("circular dependency detected for model %s", model.Name)
	}

	// Mark the current model as visited
	visited[modelKey] = true

	// Find the parent model from pathModelMap
	parentModel := pathModelMap[model.Extends]
	if parentModel == nil {
		// If the parent model is not found, it is not in the current build result, so stop checking
		return nil
	}

	// Recursively check the parent model
	return b.checkCircularDependency(parentModel, pathModelMap, visited)
}

func (b *ModuleBuilder) validate(buildResult *module.BuildResult) error {
	// 1. Group models by name
	modelMap := make(map[string][]*meta.IrModel)
	// 2. Create a mapping from path to model for quick parent model lookup
	pathModelMap := make(map[string]*meta.IrModel)

	for _, result := range module.ParserResults(buildResult) {
		if result.Model != nil {
			modelMap[result.Model.Name] = append(modelMap[result.Model.Name], result.Model)
			pathModelMap[result.Model.Path] = result.Model
		}
	}

	// 2. Validate inheritance relationships
	for _, sameNameModels := range modelMap {
		// Check the inheritance chain for models with the same name
		if len(sameNameModels) > 1 {
			if err := b.checkInheritanceChain(sameNameModels, pathModelMap); err != nil {
				return xfmt.Errorf("invalid inheritance chain for models %v: %w", sameNameModels, err)
			}
		}

		// Check for circular dependencies
		for _, model := range sameNameModels {
			visited := make(map[string]bool)
			if err := b.checkCircularDependency(model, pathModelMap, visited); err != nil {
				return xfmt.Errorf("circular dependency detected for model %s: %w", model.Path, err)
			}
		}
	}

	return nil
}

func (b *ModuleBuilder) persist(buildResult *module.BuildResult) error {
	mod := buildResult.Module

	// update module application id
	if mod.ApplicationStr != "" {
		var app *meta.IrApplication
		if result := b.runtimeScope.Session().Where("name = ?", mod.ApplicationStr).Take(&app); result.Error != nil {
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return xfmt.Errorf("error getting application by name: %w", result.Error)
			}
		}

		if app.Id.String != "" {
			mod.ApplicationId = app.Id
			mod.Application = app
		} else {
			mod.Application = &meta.IrApplication{Name: mod.ApplicationStr}
		}
	}

	// save parser results
	for _, result := range module.ParserResults(buildResult) {
		if result.Model == nil {
			continue
		}
		// only save models that are in the same path as the module
		if strings.HasPrefix(result.Path, mod.Path) {
			mod.Models = append(mod.Models, result.Model)
		}
	}

	if err := b.materializeEffectiveModels(mod); err != nil {
		return xfmt.Errorf("error materializing effective meta: %w", err)
	}

	// save module
	// Avoid writing many2many join rows here; dependency graph is managed by ModuleManager.
	if result := b.runtimeScope.Session().Omit("Dependencies", "Dependents", "Models").Save(mod); result.Error != nil {
		return xfmt.Errorf("error saving module: %w", result.Error)
	}

	if mod.Id.Valid {
		if err := b.persistModuleModels(mod.Id.String, mod.Models); err != nil {
			return xfmt.Errorf("error persisting module models: %w", err)
		}
	}

	return nil
}

func (b *ModuleBuilder) persistModuleModels(moduleID string, models []*meta.IrModel) error {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return nil
	}

	orderedPaths := make([]string, 0, len(models))
	modelByPath := make(map[string]*meta.IrModel, len(models))
	for _, m := range models {
		if m == nil {
			continue
		}
		path := strings.TrimSpace(m.Path)
		if path == "" {
			continue
		}
		if _, exists := modelByPath[path]; !exists {
			orderedPaths = append(orderedPaths, path)
		}
		modelByPath[path] = m
	}

	rows := make([]*meta.IrModel, 0, len(orderedPaths))
	for _, path := range orderedPaths {
		m := modelByPath[path]
		if m == nil {
			continue
		}
		m.ModuleId = sql.NullString{String: moduleID, Valid: true}
		rows = append(rows, m)
	}

	if result := b.runtimeScope.Session().Unscoped().Where("module_id = ?", moduleID).Delete(&meta.IrModel{}); result.Error != nil {
		return result.Error
	}
	if len(rows) == 0 {
		return nil
	}
	if result := b.runtimeScope.Session().Create(&rows); result.Error != nil {
		return result.Error
	}
	return nil
}

type effectiveMeta struct {
	fields   []*meta.IrField
	services []*meta.IrService
}

func mergeOrderedFields(parentFields []*meta.IrField, childFields []*meta.IrField, parentPath string, childPath string) []*meta.IrField {
	result := make([]*meta.IrField, 0, len(parentFields)+len(childFields))
	indexByName := make(map[string]int)

	for _, pf := range parentFields {
		if pf == nil || pf.Name == "" {
			continue
		}
		if _, exists := indexByName[pf.Name]; exists {
			continue
		}
		origin := pf.OriginModelPath
		if origin == "" {
			origin = parentPath
		}
		nf := cloneField(pf)
		nf.OriginModelPath = origin
		indexByName[nf.Name] = len(result)
		result = append(result, nf)
	}

	for _, cf := range childFields {
		if cf == nil || cf.Name == "" {
			continue
		}
		nf := cloneField(cf)
		nf.OriginModelPath = childPath
		if idx, ok := indexByName[nf.Name]; ok {
			result[idx] = nf
			continue
		}
		indexByName[nf.Name] = len(result)
		result = append(result, nf)
	}

	return result
}

func mergeOrderedServices(parentServices []*meta.IrService, childServices []*meta.IrService, parentPath string, childPath string) []*meta.IrService {
	result := make([]*meta.IrService, 0, len(parentServices)+len(childServices))
	indexByName := make(map[string]int)

	for _, ps := range parentServices {
		if ps == nil || ps.Name == "" {
			continue
		}
		if _, exists := indexByName[ps.Name]; exists {
			continue
		}
		origin := ps.OriginModelPath
		if origin == "" {
			origin = parentPath
		}
		ns := cloneService(ps)
		ns.OriginModelPath = origin
		indexByName[ns.Name] = len(result)
		result = append(result, ns)
	}

	for _, cs := range childServices {
		if cs == nil || cs.Name == "" {
			continue
		}
		ns := cloneService(cs)
		ns.OriginModelPath = childPath
		if idx, ok := indexByName[ns.Name]; ok {
			result[idx] = ns
			continue
		}
		indexByName[ns.Name] = len(result)
		result = append(result, ns)
	}

	return result
}

func (b *ModuleBuilder) materializeEffectiveModels(module *meta.IrModule) error {
	localByPath := make(map[string]*meta.IrModel)
	for _, m := range module.Models {
		if m == nil {
			continue
		}
		localByPath[m.Path] = m
	}

	cache := make(map[string]*effectiveMeta)
	visiting := make(map[string]bool)
	for _, model := range module.Models {
		if model == nil {
			continue
		}
		eff, err := b.computeEffectiveMeta(model, localByPath, cache, visiting)
		if err != nil {
			return err
		}
		model.Fields = eff.fields
		model.Services = eff.services
	}
	return nil
}

func (b *ModuleBuilder) computeEffectiveMeta(
	model *meta.IrModel,
	localByPath map[string]*meta.IrModel,
	cache map[string]*effectiveMeta,
	visiting map[string]bool,
) (*effectiveMeta, error) {
	if model == nil {
		return &effectiveMeta{}, nil
	}
	if cached, ok := cache[model.Path]; ok {
		return cached, nil
	}
	if visiting[model.Path] {
		return nil, xfmt.Errorf("circular dependency detected while materializing: %s", model.Path)
	}
	visiting[model.Path] = true
	defer func() { visiting[model.Path] = false }()

	var parent *meta.IrModel
	var parentEff *effectiveMeta
	if model.Extends != "" {
		if localParent, ok := localByPath[model.Extends]; ok {
			parent = localParent
			var err error
			parentEff, err = b.computeEffectiveMeta(parent, localByPath, cache, visiting)
			if err != nil {
				return nil, err
			}
		} else {
			loaded, err := b.loadLatestModelByPath(model.Extends)
			if err != nil {
				return nil, err
			}
			parent = loaded
			if parent != nil {
				if b.isAlreadyMaterialized(parent) {
					parentEff = &effectiveMeta{fields: parent.Fields, services: parent.Services}
				} else {
					var err error
					parentEff, err = b.computeEffectiveMeta(parent, localByPath, cache, visiting)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	parentPath := ""
	if parent != nil {
		parentPath = parent.Path
	}
	var parentFields []*meta.IrField
	var parentServices []*meta.IrService
	if parentEff != nil {
		parentFields = parentEff.fields
		parentServices = parentEff.services
	}

	fields := mergeOrderedFields(parentFields, model.Fields, parentPath, model.Path)
	services := mergeOrderedServices(parentServices, model.Services, parentPath, model.Path)

	eff := &effectiveMeta{fields: fields, services: services}
	cache[model.Path] = eff
	return eff, nil
}

func (b *ModuleBuilder) loadLatestModelByPath(path string) (*meta.IrModel, error) {
	if path == "" {
		return nil, nil
	}
	var m meta.IrModel
	if result := b.runtimeScope.Session().
		Preload("Fields", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Preload("Fields.Decorators", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Preload("Fields.Decorators.Arguments", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Preload("Services", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Preload("Services.Decorators", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Preload("Services.Decorators.Arguments", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Preload("Services.TypeParameters", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Preload("Services.Parameters", func(db *gorm.DB) *gorm.DB { return db.Where("name != ?", "this").Order("id ASC") }).
		Where("path = ?", path).
		Order("id DESC").
		Take(&m); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, xfmt.Errorf("error loading parent model by path %s: %w", path, result.Error)
	}
	return &m, nil
}

func (b *ModuleBuilder) isAlreadyMaterialized(model *meta.IrModel) bool {
	for _, f := range model.Fields {
		if f != nil && f.OriginModelPath != "" {
			return true
		}
	}
	for _, s := range model.Services {
		if s != nil && s.OriginModelPath != "" {
			return true
		}
	}
	return false
}

func cloneField(src *meta.IrField) *meta.IrField {
	if src == nil {
		return nil
	}
	dst := *src
	dst.BaseModel = meta.BaseModel{}
	dst.ModelId = sql.NullString{}
	dst.Model = nil

	dst.Decorators = nil
	if len(src.Decorators) > 0 {
		dst.Decorators = make([]*meta.IrDecorator, 0, len(src.Decorators))
		for _, d := range src.Decorators {
			if d == nil {
				continue
			}
			dst.Decorators = append(dst.Decorators, cloneDecorator(d))
		}
	}
	return &dst
}

func cloneService(src *meta.IrService) *meta.IrService {
	if src == nil {
		return nil
	}
	dst := *src
	dst.BaseModel = meta.BaseModel{}
	dst.ModelId = sql.NullString{}
	dst.Model = nil

	dst.TypeParameters = nil
	if len(src.TypeParameters) > 0 {
		dst.TypeParameters = make([]*meta.IrTypeParameter, 0, len(src.TypeParameters))
		for _, tp := range src.TypeParameters {
			if tp == nil {
				continue
			}
			cloned := *tp
			cloned.BaseModel = meta.BaseModel{}
			cloned.ServiceId = sql.NullString{}
			cloned.Service = nil
			dst.TypeParameters = append(dst.TypeParameters, &cloned)
		}
	}

	dst.Parameters = nil
	if len(src.Parameters) > 0 {
		dst.Parameters = make([]*meta.IrParameter, 0, len(src.Parameters))
		for _, p := range src.Parameters {
			if p == nil {
				continue
			}
			cloned := *p
			cloned.BaseModel = meta.BaseModel{}
			cloned.ServiceId = sql.NullString{}
			cloned.Service = nil
			dst.Parameters = append(dst.Parameters, &cloned)
		}
	}

	dst.Decorators = nil
	if len(src.Decorators) > 0 {
		dst.Decorators = make([]*meta.IrDecorator, 0, len(src.Decorators))
		for _, d := range src.Decorators {
			if d == nil {
				continue
			}
			dst.Decorators = append(dst.Decorators, cloneDecorator(d))
		}
	}

	return &dst
}

func cloneDecorator(src *meta.IrDecorator) *meta.IrDecorator {
	if src == nil {
		return nil
	}
	dst := *src
	dst.BaseModel = meta.BaseModel{}
	dst.ModelId = sql.NullString{}
	dst.Model = nil
	dst.ServiceId = sql.NullString{}
	dst.Service = nil
	dst.FieldId = sql.NullString{}
	dst.Field = nil
	dst.ComponentId = sql.NullString{}
	dst.Component = nil

	dst.Arguments = nil
	if len(src.Arguments) > 0 {
		dst.Arguments = make([]*meta.IrArgument, 0, len(src.Arguments))
		for _, a := range src.Arguments {
			if a == nil {
				continue
			}
			cloned := *a
			cloned.BaseModel = meta.BaseModel{}
			cloned.DecoratorId = sql.NullString{}
			cloned.Decorator = nil
			dst.Arguments = append(dst.Arguments, &cloned)
		}
	}

	return &dst
}

func (b *ModuleBuilder) Build() (*module.BuildResult, error) {
	// 1. prebuild for parse original model extends
	prebuildResult, err := b.prebuild()
	if err != nil {
		return nil, xfmt.Errorf("error prebuilding: %w", err)
	}

	// 2. recompute and modify file content for model extends
	if err := b.updatePrebuildResult(prebuildResult); err != nil {
		return nil, xfmt.Errorf("error generating content: %w", err)
	}

	// 3. build for output
	buildResult, err := b.build(prebuildResult)
	if err != nil {
		return nil, xfmt.Errorf("error building: %w", err)
	}

	// 4. validate models
	if err := b.validate(buildResult); err != nil {
		return nil, xfmt.Errorf("error validating: %w", err)
	}

	// 5. persist build result
	if err := b.persist(buildResult); err != nil {
		return nil, xfmt.Errorf("error persisting build result: %w", err)
	}

	return buildResult, nil
}

// Bundle runs the build pipeline but does NOT validate/persist meta models.
// This is intended for application-stage bundling where DB/IR is already correct.
func (b *ModuleBuilder) Bundle() (*module.BuildResult, error) {
	prebuildResult, err := b.prebuild()
	if err != nil {
		return nil, xfmt.Errorf("error prebuilding: %w", err)
	}

	if err := b.updatePrebuildResult(prebuildResult); err != nil {
		return nil, xfmt.Errorf("error generating content: %w", err)
	}

	buildResult, err := b.build(prebuildResult)
	if err != nil {
		return nil, xfmt.Errorf("error building: %w", err)
	}

	return buildResult, nil
}

// BundleToDirCtx runs Bundle but writes dist outputs into distAppDir.
// Intended for pipeline-managed staging (pipeline will commit/replace dist/<app>).
func (b *ModuleBuilder) BundleToDirCtx(ctx context.Context, distAppDir string) (*module.BuildResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	restore := b.bindRuntimeState(ctx)
	defer restore()

	prev := b.distAppDirOverride
	b.distAppDirOverride = distAppDir
	defer func() { b.distAppDirOverride = prev }()
	return b.Bundle()
}

func NewModuleBuilder(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.IrModule, entryPoint string, opts ...func(*ModuleBuilder)) module.Builder {
	b := &ModuleBuilder{
		runtimeScope:    runtimeScope,
		runtimeOptions:  newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{}, false),
		jsExecutor:      jsExecutor,
		module:          module,
		entryPoint:      entryPoint,
		publishDist:     true,
		outFileName:     "index.js",
		globalName:      module.ApplicationStr,
		tsParserFactory: backendtsparser.NewTsParser,
	}

	for _, opt := range opts {
		opt(b)
	}

	if b.prebuildPlugin == nil {
		b.prebuildPlugin = internalbackendplugin.NewBackendPlugin(runtimeScope, module, entryPoint)
	}
	if b.buildPlugin == nil {
		b.buildPlugin = internalbackendplugin.NewBackendPlugin(runtimeScope, module, entryPoint)
	}
	if b.runtimeScope != nil {
		b.runtimeOptions = runtimeOptionsFromScope(b.runtimeScope)
	}

	return b
}

func WithPrebuildPlugin(plugin esbplugins.EsbPlugin) func(*ModuleBuilder) {
	return func(b *ModuleBuilder) {
		b.prebuildPlugin = plugin
	}
}

func WithBuildPlugin(plugin esbplugins.EsbPlugin) func(*ModuleBuilder) {
	return func(b *ModuleBuilder) {
		b.buildPlugin = plugin
	}
}

func WithPublishDist(publish bool) func(*ModuleBuilder) {
	return func(b *ModuleBuilder) {
		b.publishDist = publish
	}
}

func WithOutFileName(name string) func(*ModuleBuilder) {
	return func(b *ModuleBuilder) {
		b.outFileName = name
	}
}

func WithGlobalName(name string) func(*ModuleBuilder) {
	return func(b *ModuleBuilder) {
		b.globalName = name
	}
}
