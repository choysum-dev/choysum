// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/choysum-dev/choysum/internal/esbplugins"
	internalwebplugin "github.com/choysum-dev/choysum/internal/esbplugins/webplugin"
	"github.com/choysum-dev/choysum/internal/esbplugins/webprebuildplugin"
	"github.com/choysum-dev/choysum/internal/esmresolver"
	modulegenerator "github.com/choysum-dev/choysum/internal/module/artifact/generate"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/parser/vueparser"
	"github.com/choysum-dev/choysum/internal/parser/vueparser/vuesfchtmlparser"
	"github.com/choysum-dev/choysum/internal/vueplugin"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
	"github.com/evanw/esbuild/pkg/api"
	xfmt "golang.org/x/exp/errors/fmt"
	"golang.org/x/net/html"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WebModuleBuilder struct {
	runtimeScope   scope.Scope
	runtimeOptions runtimeOptions
	jsExecutor     jsexecutor.ScriptExecutor
	module         *meta.IrModule
	entryPoint     string
	buildPlugin    esbplugins.EsbPlugin
	prebuildPlugin esbplugins.EsbPlugin
	parser         parser.Parser
	parserFactory  func(scope.Scope, *meta.IrModule) parser.Parser
	publishDist    bool

	// Optional override for pipeline-managed staging.
	// When set, build() writes into this directory directly and does not commit.
	distWebDirOverride string
}

func (b *WebModuleBuilder) bindRuntimeState(ctx context.Context) func() {
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
	prevParser := b.parser
	b.runtimeScope = runtimeScope
	if b.parserFactory != nil {
		b.parser = b.parserFactory(b.runtimeScope, b.module)
	}
	return func() {
		b.runtimeScope = prevScope
		b.parser = prevParser
	}
}

var sassModuleDirectiveRe = regexp.MustCompile(`(?m)^[ \t]*@(charset|use|forward)\b[^;]*;[ \t]*$`)

type roleCodeRow struct {
	Id   string
	Code string
}

type roleUiResourceGrantRow struct {
	meta.BaseModel
	RoleId          sql.NullString `gorm:"type:char(20);not null;index"`
	Mode            string         `gorm:"type:varchar(16);not null;default:allow;index"`
	IrApplicationId sql.NullString `gorm:"type:char(20);index"`
	IrUiResourceId  sql.NullString `gorm:"type:char(20);index"`
}

func (roleUiResourceGrantRow) TableName() string {
	return "auth_role_ui_resource"
}

func (b *WebModuleBuilder) Build() (*module.BuildResult, error) {
	return b.BuildCtx(context.Background())
}

func (b *WebModuleBuilder) BuildCtx(ctx context.Context) (*module.BuildResult, error) {
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

	// 1. prebuild for parse original model extends
	prebuildResult, err := b.prebuild()
	if err != nil {
		return nil, xfmt.Errorf("Error prebuilding: %w", err)
	}

	// 2. upate component content
	if err := b.updatePrebuildResult(prebuildResult); err != nil {
		return nil, xfmt.Errorf("Error generating component content: %w", err)
	}

	// 3. build for output
	buildResult, err := b.build(ctx, prebuildResult)
	if err != nil {
		return nil, xfmt.Errorf("Error building: %w", err)
	}

	// 4. validate components
	if err := b.validate(buildResult); err != nil {
		return nil, xfmt.Errorf("Error validating: %w", err)
	}

	// 5. persist build result
	if err := b.persist(buildResult); err != nil {
		return nil, xfmt.Errorf("Error persisting build result: %w", err)
	}

	return buildResult, nil
}

// BuildToDirCtx builds global web assets into distWebDir.
// Intended for pipeline-managed staging (pipeline will commit/replace dist/web).
func (b *WebModuleBuilder) BuildToDirCtx(ctx context.Context, distWebDir string) (*module.BuildResult, error) {
	prev := b.distWebDirOverride
	b.distWebDirOverride = distWebDir
	defer func() { b.distWebDirOverride = prev }()
	return b.BuildCtx(ctx)
}

func (b *WebModuleBuilder) prebuild() (*module.BuildResult, error) {
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

// Check inheritance relationship for components with the same name
func (b *WebModuleBuilder) checkInheritanceChain(components []*meta.IrComponent, pathComponentMap map[string]*meta.IrComponent) error {
	getInheritancePath := func(component *meta.IrComponent) []string {
		var path []string
		current := component
		for current != nil && current.Extends != "" {
			path = append(path, current.Path)
			// Get parent component from pathComponentMap
			current = pathComponentMap[current.Extends]
			// Stop if parent not found
			if current == nil {
				break
			}
		}
		return path
	}

	// Check if child component is in parent's inheritance chain
	isInInheritanceChain := func(child *meta.IrComponent, parent *meta.IrComponent) bool {
		current := child
		for current != nil && current.Extends != "" {
			if current.Extends == parent.Path {
				return true
			}
			// Get parent from component map
			current = pathComponentMap[current.Extends]
			// Stop if parent not found
			if current == nil {
				break
			}
		}
		return false
	}

	// Sort by inheritance path length to ensure deepest subclasses are first
	sort.Slice(components, func(i, j int) bool {
		return len(getInheritancePath(components[i])) > len(getInheritancePath(components[j]))
	})

	// Verify inheritance chain
	for i := 0; i < len(components); i++ {
		for j := i + 1; j < len(components); j++ {
			if !isInInheritanceChain(components[i], components[j]) {
				return xfmt.Errorf("component %s and %s have same name but not in inheritance chain",
					components[i].Path, components[j].Path)
			}
		}
	}
	return nil
}

// Check circular dependencies
func (b *WebModuleBuilder) checkCircularDependency(
	component *meta.IrComponent,
	pathComponentMap map[string]*meta.IrComponent,
	visited map[string]bool,
) error {
	// Return nil if component is nil or has no inheritance
	if component == nil || component.Extends == "" {
		return nil
	}

	// Create unique identifier using "name:path" format
	componentKey := fmt.Sprintf("%s:%s", component.Name, component.Path)

	// Circular dependency detected if component was visited
	if visited[componentKey] {
		return xfmt.Errorf("circular dependency detected for component %s", component.Name)
	}

	// Mark component as visited
	visited[componentKey] = true

	// Get parent component from map
	parentComponent := pathComponentMap[component.Extends]
	if parentComponent == nil {
		// Parent not in current build result, stop checking
		return nil
	}

	// Recursively check parent component
	return b.checkCircularDependency(parentComponent, pathComponentMap, visited)
}

func (b *WebModuleBuilder) validate(buildResult *module.BuildResult) error {
	// 1. Group components by name
	componentMap := make(map[string][]*meta.IrComponent)
	// 2. Create path to component mapping for quick parent lookup
	pathComponentMap := make(map[string]*meta.IrComponent)

	for _, result := range module.ParserResults(buildResult) {
		if result.VueComponent != nil {
			componentMap[result.VueComponent.Name] = append(componentMap[result.VueComponent.Name], result.VueComponent)
			pathComponentMap[result.VueComponent.Path] = result.VueComponent
		}
	}

	// Validate inheritance relationships
	for _, sameNameComponents := range componentMap {
		// Check inheritance chain for components with same name
		if len(sameNameComponents) > 1 {
			if err := b.checkInheritanceChain(sameNameComponents, pathComponentMap); err != nil {
				return xfmt.Errorf("invalid inheritance chain for components %v: %w", sameNameComponents, err)
			}
		}

		// Check for circular dependencies
		for _, component := range sameNameComponents {
			visited := make(map[string]bool)
			if err := b.checkCircularDependency(component, pathComponentMap, visited); err != nil {
				return xfmt.Errorf("circular dependency detected for component %s: %w", component.Path, err)
			}
		}
	}

	return nil
}

func (b *WebModuleBuilder) build(ctx context.Context, prebuildResult *module.BuildResult) (*module.BuildResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	b.buildPlugin.SetParserResults(module.ParserResults(prebuildResult))

	var result api.BuildResult

	buildTo := func(webDir string, publish bool) error {
		assetsDir := filepath.Join(webDir, "assets")
		indexHtml := filepath.Join(webDir, "index.html")

		buildOptions := b.buildOptions(false, esbplugins.WithIndexHtmlOutFile(indexHtml))
		buildOptions.Outdir = assetsDir
		// When publishing (to staging or final dir), allow esbuild/plugins to write
		// to disk so index.html processors (OnEnd hooks) can run.
		buildOptions.Write = publish

		result = api.Build(*buildOptions)
		if len(result.Errors) > 0 {
			var buildErr error
			for _, v := range result.Errors {
				if buildErr == nil {
					if v.Location != nil {
						buildErr = xfmt.Errorf("%s:%d:%d %s", v.Location.File, v.Location.Line, v.Location.Column, v.Text)
					} else if v.Text == "" {
						buildErr = xfmt.Errorf("Unknown error")
					} else {
						buildErr = xfmt.Errorf("%s", v.Text)
					}
				} else {
					buildErr = xfmt.Errorf("%s : %w", v.Text, buildErr)
				}
			}
			return buildErr
		}

		if publish {
			if _, err := os.Stat(indexHtml); err != nil {
				return xfmt.Errorf("index.html not generated: %w", err)
			}
		}

		return nil
	}

	if b.publishDist {
		if b.distWebDirOverride != "" {
			if err := buildTo(b.distWebDirOverride, true); err != nil {
				return nil, err
			}
		} else {
			runtimeOptions := b.resolvedRuntimeOptions()
			distWebDir := filepath.Join(runtimeOptions.distPath, "web")
			err := staging.WithStagingDir(ctx, distWebDir, func(stagingWebDir string) error {
				return buildTo(stagingWebDir, true)
			})
			if err != nil {
				return nil, err
			}
		}
	} else {
		tmpDir, err := os.MkdirTemp("", "choysum-web-build-*")
		if err != nil {
			return nil, xfmt.Errorf("create temp web build dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		// Build and run processors into a temp directory, but do not publish dist/web.
		if err := buildTo(tmpDir, false); err != nil {
			return nil, err
		}
	}

	parserResults, err := b.buildPlugin.GetParserResults()
	if err != nil {
		return nil, xfmt.Errorf("Error getting parser results: %w", err)
	}
	return module.WithParserResults(&module.BuildResult{
		Module:        b.module,
		EsbuildResult: &result,
	}, parserResults), nil
}

func (b *WebModuleBuilder) updatePrebuildResult(buildResult *module.BuildResult) error {
	pathAlias, err := parser.ParseTsconfigPathAlias(b.buildOptions(false))
	if err != nil {
		return xfmt.Errorf("Error parsing tsconfig path alias: %w", err)
	}

	var VueAppImportTree []string
	if result := b.runtimeScope.Session().
		Model(&meta.IrModule{}).
		Where("web_entry_point != ?", "").
		Where("status = ?", "installed").
		Order("id DESC").
		Pluck("web_entry_point", &VueAppImportTree); result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return xfmt.Errorf("Error finding module web entry points: %w", result.Error)
		}
	}

	for _, result := range module.ParserResults(buildResult) {
		if err := b.updateComponent(buildResult, pathAlias, result.Path); err != nil {
			return err
		}

		if slices.Contains(VueAppImportTree, result.Path) {
			result.VueAppImportTree = VueAppImportTree
		}
	}

	return nil
}

func (b *WebModuleBuilder) updateComponent(buildResult *module.BuildResult, pathAlias map[string]string, extends string) error {
	var result *parser.ParserResult
	// find parser result
	for _, r := range module.ParserResults(buildResult) {
		if r.Path == extends {
			result = r
			break
		}
	}

	// try to load component if not found
	if result == nil {
		byteContent, err := os.ReadFile(extends)
		if err != nil {
			return xfmt.Errorf("Error reading file: %w", err)
		}
		content := string(byteContent)
		// Resolve the path alias for the Vue file
		content = vueplugin.ResolveVueStylePath(content, extends, pathAlias)
		result, err = b.parser.Parse(pathAlias, extends, content)
		if err != nil {
			return xfmt.Errorf("Error parsing component: %w", err)
		}
		module.AppendParserResult(buildResult, result)
	}

	// check if component is already updated
	if result.Content != "" || result.VueComponent == nil {
		return nil
	}

	// if component does not extend anything, return raw content
	if result.VueComponent.Extends == "" {
		result.Content = result.RawContent
		return nil
	}

	// update extends
	extendsComponent, err := b.getNewExtends(buildResult, result.VueComponent)
	if err != nil {
		return xfmt.Errorf("Error getting new extends component: %w", err)
	}
	if extendsComponent != nil {
		result.VueComponent.Extends = extendsComponent.Path
	}

	// load extends component
	if err := b.updateComponent(buildResult, pathAlias, result.VueComponent.Extends); err != nil {
		return xfmt.Errorf("Error loading extends component: %w", err)
	}

	// get extendsResult
	var extendsResult *parser.ParserResult
	for _, r := range module.ParserResults(buildResult) {
		if r.Path == result.VueComponent.Extends {
			extendsResult = r
			break
		}
	}

	// update script node
	scriptNode, err := b.getScriptNode(result, extendsResult)
	if err != nil {
		return xfmt.Errorf("Error getting script node: %w", err)
	}
	result.ScriptNode = scriptNode

	// update template node
	templateNode, err := b.getTemplateNode(result, extendsResult)
	if err != nil {
		return xfmt.Errorf("Error getting template node: %w", err)
	}
	result.TemplateNode = templateNode

	// update style nodes
	styleNodes, err := b.getStyleNodes(result, extendsResult)
	if err != nil {
		return xfmt.Errorf("Error getting style nodes: %w", err)
	}
	result.StyleNodes = styleNodes

	// render component
	content, err := b.renderComponent(result)
	if err != nil {
		return xfmt.Errorf("Error rendering component: %w", err)
	}
	result.Content = content

	// reparse component
	reParseResult, err := b.parser.Parse(pathAlias, result.Path, content)
	if err != nil {
		return xfmt.Errorf("Error parsing extends component: %w", err)
	}
	result.VueComponent = reParseResult.VueComponent
	result.VueComponentsPropertys = reParseResult.VueComponentsPropertys
	result.VueExtendsProperty = reParseResult.VueExtendsProperty
	result.Imports = reParseResult.Imports
	result.Exports = reParseResult.Exports
	result.RawScriptNode = reParseResult.RawScriptNode
	result.RawScriptSetupNode = reParseResult.RawScriptSetupNode
	result.RawTemplateNode = reParseResult.RawTemplateNode
	result.RawStyleNodes = reParseResult.RawStyleNodes
	result.ScriptNode = reParseResult.ScriptNode
	result.ScriptSetupNode = reParseResult.ScriptSetupNode
	result.TemplateNode = reParseResult.TemplateNode
	result.StyleNodes = reParseResult.StyleNodes

	return nil
}

func (b *WebModuleBuilder) getTemplateImportComponents(extendsResult *parser.ParserResult) (map[string]*parser.Import, error) {
	if extendsResult == nil || extendsResult.RawTemplateNode == nil || len(extendsResult.Imports) == 0 {
		return nil, nil
	}

	templateImports := make(map[string]*parser.Import)

	// Iterate through all imported components
	for importName, importInfo := range extendsResult.Imports {
		found := templateUsesImportAsTag(extendsResult.RawTemplateNode, importName)

		if found {
			templateImports[importName] = importInfo
			continue
		}

		// Dynamic bindings can reference imported components without using tag names,
		// e.g. <component :is="QuestionFilled" />.
		if templateUsesImportInDynamicBinding(extendsResult.RawTemplateNode, importName) {
			templateImports[importName] = importInfo
		}
	}

	return templateImports, nil
}

func normalizeTemplateComponentName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return name
}

func templateUsesImportAsTag(root *html.Node, importName string) bool {
	importName = strings.TrimSpace(importName)
	if root == nil || importName == "" {
		return false
	}

	targets := map[string]struct{}{}
	for _, candidate := range []string{importName, strcase.ToKebab(importName), strings.ToLower(importName)} {
		normalized := normalizeTemplateComponentName(candidate)
		if normalized != "" {
			targets[normalized] = struct{}{}
		}
	}
	if len(targets) == 0 {
		return false
	}

	var walk func(node *html.Node) bool
	walk = func(node *html.Node) bool {
		if node == nil {
			return false
		}

		if node.Type == html.ElementNode {
			normalizedNodeName := normalizeTemplateComponentName(node.Data)
			if _, ok := targets[normalizedNodeName]; ok {
				return true
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if walk(child) {
				return true
			}
		}

		return false
	}

	return walk(root)
}

func templateUsesImportInDynamicBinding(root *html.Node, importName string) bool {
	importName = strings.TrimSpace(importName)
	if root == nil || importName == "" {
		return false
	}

	identPattern := regexp.MustCompile(`(^|[^A-Za-z0-9_$])` + regexp.QuoteMeta(importName) + `([^A-Za-z0-9_$]|$)`)

	var walk func(node *html.Node) bool
	walk = func(node *html.Node) bool {
		if node == nil {
			return false
		}

		if node.Type == html.ElementNode {
			for _, attr := range node.Attr {
				key := strings.TrimSpace(attr.Key)
				value := strings.TrimSpace(attr.Val)
				if key == "" || value == "" {
					continue
				}

				isDynamicBinding := strings.HasPrefix(key, ":") || strings.HasPrefix(key, "v-") || strings.HasPrefix(key, "@") || key == "is"
				if !isDynamicBinding {
					continue
				}

				if identPattern.MatchString(value) {
					return true
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if walk(child) {
				return true
			}
		}

		return false
	}

	return walk(root)
}

func (b *WebModuleBuilder) getScriptNode(result *parser.ParserResult, extendsResult *parser.ParserResult) (*html.Node, error) {
	// Create new script node
	scriptNode := &html.Node{
		Type:      result.RawScriptNode.Type,
		Data:      result.RawScriptNode.Data,
		DataAtom:  result.RawScriptNode.DataAtom,
		Namespace: result.RawScriptNode.Namespace,
		Attr:      result.RawScriptNode.Attr,
	}

	// Keep the original script text so parser byte ranges remain valid.
	scriptContent := htmlquery.InnerText(result.RawScriptNode)

	// Find xpath component node and apply component-list replacement first,
	// before any import rewrites that might reshape line layout.
	var xpathPropertyNode *parser.PropertyNode
	xpathModuleSpec, xpathReferenceIdent := meta.XpathComponentModuleSpec(b.runtimeScope)
	runtimeOptions := b.resolvedRuntimeOptions()
	normalizeModuleSpec := func(path string) string {
		p := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		return strings.TrimSuffix(p, "/")
	}

	canonicalXPathModule := normalizeModuleSpec(xpathModuleSpec)
	canonicalCoreWebModule := normalizeModuleSpec(filepath.Join(runtimeOptions.modulesPath, "core", "web"))

	for _, node := range result.VueComponentsPropertys {
		if node == nil {
			continue
		}
		if normalizeModuleSpec(node.ModuleSpecPath) == canonicalXPathModule && node.ReferenceIdent == xpathReferenceIdent {
			xpathPropertyNode = node
			break
		}
	}

	if xpathPropertyNode == nil {
		for _, node := range result.VueComponentsPropertys {
			if node == nil {
				continue
			}

			moduleSpec := normalizeModuleSpec(node.ModuleSpecPath)
			isCoreWebModule := moduleSpec == canonicalCoreWebModule ||
				strings.HasSuffix(moduleSpec, "/core/web") ||
				strings.HasSuffix(moduleSpec, "/core/web/index") ||
				strings.HasSuffix(moduleSpec, "/core/web/component") ||
				strings.HasSuffix(moduleSpec, "/core/web/component/index") ||
				strings.HasSuffix(moduleSpec, "/core/web/component/xpath.vue")

			if isCoreWebModule && node.ReferenceIdent == "Xpath" {
				xpathPropertyNode = node
				break
			}

			if isCoreWebModule && (node.ReferenceIdent == "default" || strings.TrimSpace(node.ValueText) == "Xpath") {
				xpathPropertyNode = node
				break
			}
		}
	}

	var extendsTemplateImportComponents map[string]*parser.Import
	if xpathPropertyNode != nil {
		var err error
		extendsTemplateImportComponents, err = b.getTemplateImportComponents(extendsResult)
		if err != nil {
			return nil, xfmt.Errorf("Error getting template imports: %w", err)
		}

		// Replace xpath component list using parser-provided source ranges.
		scriptContent, err = b.replaceXPathComponents(scriptContent, xpathPropertyNode, extendsTemplateImportComponents, result.Imports)
		if err != nil {
			return nil, err
		}
	}

	// When the resolved extends path changes, keep the original extends identifier
	// and only rewrite its import module path. This preserves parent references used
	// elsewhere (e.g. ParentComp?.setup?.(...)) and avoids accidental self-reference.
	if result.VueComponent.RawExtends != result.VueComponent.Extends && result.VueExtendsProperty != nil {
		importUpdated := false

		rewriteImportModuleSpec := func(imp *parser.Import) bool {
			if imp == nil {
				return false
			}

			oldImportStmt := imp.Text
			oldModuleSpec := imp.ModuleSpecText
			if oldImportStmt == "" || oldModuleSpec == "" {
				return false
			}

			canonicalImportStmt := regexp.MustCompile(`from\s*\n\s*`).ReplaceAllString(oldImportStmt, "from ")
			if canonicalImportStmt == "" {
				canonicalImportStmt = oldImportStmt
			}

			quote := "'"
			if strings.HasPrefix(oldModuleSpec, "\"") {
				quote = "\""
			}
			newModuleSpec := fmt.Sprintf("%s%s%s", quote, result.VueComponent.Extends, quote)
			newImportStmt := strings.Replace(canonicalImportStmt, oldModuleSpec, newModuleSpec, 1)
			if newImportStmt != canonicalImportStmt {
				rewritten := strings.Replace(scriptContent, oldImportStmt, newImportStmt, 1)
				if rewritten != scriptContent {
					scriptContent = rewritten
					return true
				}
			}

			// Fallback: rewrite only module spec literal text captured from AST.
			rewrittenBySpec := strings.Replace(scriptContent, oldModuleSpec, newModuleSpec, 1)
			if rewrittenBySpec == scriptContent {
				return false
			}
			scriptContent = rewrittenBySpec
			return true
		}

		for _, imp := range result.Imports {
			if imp == nil {
				continue
			}
			if imp.ModuleSpecPath != result.VueComponent.RawExtends {
				continue
			}
			if rewriteImportModuleSpec(imp) {
				importUpdated = true
				break
			}
		}

		if !importUpdated {
			extendsRef := strings.TrimSpace(result.VueExtendsProperty.ValueText)
			if extendsRef == "" {
				extendsRef = strings.TrimSpace(result.VueExtendsProperty.Text)
			}
			if idx := strings.Index(extendsRef, "."); idx > 0 {
				extendsRef = strings.TrimSpace(extendsRef[:idx])
			}
			if idx := strings.Index(extendsRef, " "); idx > 0 {
				extendsRef = strings.TrimSpace(extendsRef[:idx])
			}
			if extendsRef != "" {
				if imp, ok := result.Imports[extendsRef]; ok {
					importUpdated = rewriteImportModuleSpec(imp)
				}

				if !importUpdated {
					// Final targeted fallback: rewrite the import line that imports extendsRef.
					// This avoids broad regex replacement while keeping behavior robust when AST
					// node text cannot be mapped back to the current script content verbatim.
					pattern := fmt.Sprintf(`(?m)^\s*import\s+(?:type\s+)?(?:[^\n]*\b%s\b[^\n]*)\s+from\s+['"][^'"]+['"]\s*;?`, regexp.QuoteMeta(extendsRef))
					re := regexp.MustCompile(pattern)
					if stmt := re.FindString(scriptContent); stmt != "" {
						reFrom := regexp.MustCompile(`from\s+['"][^'"]+['"]`)
						rewrittenStmt := reFrom.ReplaceAllString(stmt, fmt.Sprintf("from '%s'", result.VueComponent.Extends))
						rewritten := strings.Replace(scriptContent, stmt, rewrittenStmt, 1)
						if rewritten != scriptContent {
							scriptContent = rewritten
							importUpdated = true
						}
					}
				}
			}
		}

		if !importUpdated {
			return nil, xfmt.Errorf("failed to rewrite extends import path from %s to %s", result.VueComponent.RawExtends, result.VueComponent.Extends)
		}
	}

	// If xpath component exists, append missing imports after all in-place rewrites.
	if xpathPropertyNode != nil {
		// Add new import statements
		var err error
		scriptContent, err = b.appendNewImports(scriptContent, extendsTemplateImportComponents, result.Imports)
		if err != nil {
			return nil, err
		}

	}

	// Update script node content
	scriptNode.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: scriptContent,
	})
	return scriptNode, nil
}

// Replace xpath component list
func (b *WebModuleBuilder) replaceXPathComponents(scriptContent string, xpathNode *parser.PropertyNode, extendsImports map[string]*parser.Import, currentImports map[string]*parser.Import) (string, error) {
	if xpathNode == nil {
		return "", xfmt.Errorf("xpath node is nil")
	}

	// Get new component list
	var newComponents []string
	for componentName := range extendsImports {
		if _, exists := currentImports[componentName]; !exists {
			newComponents = append(newComponents, componentName)
		}
	}
	sort.Strings(newComponents)
	if len(newComponents) == 0 {
		return scriptContent, nil
	}

	componentsStr := strings.Join(newComponents, ",\n    ")

	// Primary path: replace by parser byte range for the property node.
	if xpathNode.Start >= 0 && xpathNode.End > xpathNode.Start && xpathNode.End <= len(scriptContent) {
		return scriptContent[:xpathNode.Start] + componentsStr + scriptContent[xpathNode.End:], nil
	}

	joined := scriptContent
	replaced := false

	// Fallback for cases where source ranges are unavailable.
	symbol := strings.TrimSpace(xpathNode.ValueText)
	if symbol == "" {
		symbol = strings.TrimSpace(xpathNode.Name)
	}
	if symbol == "" {
		symbol = strings.TrimSpace(xpathNode.Text)
	}
	if symbol == "" {
		symbol = "Xpath"
	}

	// Prefer parser-backed reparse fallback over regex guessing for complex component blocks.
	reparsedNode, err := b.reparseXPathComponentsPropertyNode(scriptContent, symbol)
	if err == nil && reparsedNode != nil {
		if reparsedNode.Start >= 0 && reparsedNode.End > reparsedNode.Start && reparsedNode.End <= len(scriptContent) {
			joined = scriptContent[:reparsedNode.Start] + componentsStr + scriptContent[reparsedNode.End:]
			replaced = true
		}
	}

	if !replaced {
		re := regexp.MustCompile(`(?s)(components\s*:\s*\{[^}]*?)\b` + regexp.QuoteMeta(symbol) + `\b`)
		if loc := re.FindStringSubmatchIndex(joined); loc != nil {
			symbolStart := loc[3]
			symbolEnd := loc[1]
			joined = joined[:symbolStart] + componentsStr + joined[symbolEnd:]
			replaced = true
		}
	}

	if !replaced {
		return "", xfmt.Errorf("failed to replace xpath components for symbol %s", symbol)
	}

	return joined, nil
}

func (b *WebModuleBuilder) reparseXPathComponentsPropertyNode(scriptContent string, symbol string) (*parser.PropertyNode, error) {
	if strings.TrimSpace(scriptContent) == "" {
		return nil, xfmt.Errorf("script content is empty")
	}
	if b == nil || b.runtimeScope == nil || b.module == nil {
		return nil, xfmt.Errorf("web builder context is not available for reparse fallback")
	}
	runtimeOptions := b.resolvedRuntimeOptions()

	pathAlias := map[string]string{}
	resolvedAlias, err := parser.ParseTsconfigPathAlias(&api.BuildOptions{
		Tsconfig: filepath.Join(runtimeOptions.modulesPath, "tsconfig.json"),
	})
	if err == nil {
		pathAlias = resolvedAlias
	}

	vueParser := vueparser.NewVueParser(b.runtimeScope, b.module)
	virtualPath := filepath.Join(runtimeOptions.modulesPath, ".choysum_internal", "xpath_fallback.vue")
	wrapped := "<template><div/></template>\n<script lang=\"ts\">\n" + scriptContent + "\n</script>\n"
	reparsed, err := vueParser.Parse(pathAlias, virtualPath, wrapped)
	if err != nil {
		return nil, xfmt.Errorf("reparse script fallback: %w", err)
	}
	if reparsed == nil || len(reparsed.VueComponentsPropertys) == 0 {
		return nil, xfmt.Errorf("no component properties found in reparsed script")
	}

	trimmedSymbol := strings.TrimSpace(symbol)
	for _, node := range reparsed.VueComponentsPropertys {
		if node == nil {
			continue
		}
		if trimmedSymbol != "" {
			if strings.TrimSpace(node.Name) == trimmedSymbol || strings.TrimSpace(node.ValueText) == trimmedSymbol || strings.TrimSpace(node.Text) == trimmedSymbol {
				return node, nil
			}
		}
		if strings.TrimSpace(node.ValueText) == "Xpath" || strings.TrimSpace(node.Name) == "Xpath" || strings.TrimSpace(node.ReferenceIdent) == "Xpath" {
			return node, nil
		}
	}

	return nil, xfmt.Errorf("xpath component symbol %s not found after reparse", trimmedSymbol)
}

// Add new import statements
func (b *WebModuleBuilder) appendNewImports(scriptContent string, extendsImports map[string]*parser.Import, currentImports map[string]*parser.Import) (string, error) {
	// Group by ModuleSpecPath
	importGroups := make(map[string]struct {
		defaultImport string
		namedImports  []string
	})

	// Collect components to be imported
	for componentName, importInfo := range extendsImports {
		if _, exists := currentImports[componentName]; !exists {
			group := importGroups[importInfo.ModuleSpecPath]
			if importInfo.ReferenceIdent == "default" {
				group.defaultImport = componentName
			} else {
				group.namedImports = append(group.namedImports, componentName)
			}
			importGroups[importInfo.ModuleSpecPath] = group
		}
	}

	// Generate import statements
	var newImports []string
	paths := make([]string, 0, len(importGroups))
	for path := range importGroups {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		group := importGroups[path]
		sort.Strings(group.namedImports)
		if group.defaultImport != "" && len(group.namedImports) > 0 {
			newImports = append(newImports, fmt.Sprintf("\nimport %s, { %s } from '%s';",
				group.defaultImport,
				strings.Join(group.namedImports, ", "),
				path))
		} else if group.defaultImport != "" {
			newImports = append(newImports, fmt.Sprintf("\nimport %s from '%s';",
				group.defaultImport, path))
		} else if len(group.namedImports) > 0 {
			newImports = append(newImports, fmt.Sprintf("\nimport { %s } from '%s';",
				strings.Join(group.namedImports, ", "),
				path))
		}
	}

	if len(newImports) > 0 {
		scriptContent += strings.Join(newImports, "") + "\n"
	}

	return scriptContent, nil
}

func (b *WebModuleBuilder) getTemplateNode(result *parser.ParserResult, extendsResult *parser.ParserResult) (*html.Node, error) {
	// apply xpath to template
	extendsTemplateNode := extendsResult.RawTemplateNode
	if extendsResult.TemplateNode != nil {
		extendsTemplateNode = extendsResult.TemplateNode
	}

	templateNode, err := vuesfchtmlparser.ApplyXPathToTemplate(extendsTemplateNode, result.RawTemplateNode)
	if err != nil {
		return nil, xfmt.Errorf("Error applying XPath to template: %w", err)
	}

	return templateNode, nil
}

func (b *WebModuleBuilder) getStyleNodes(result *parser.ParserResult, extendsResult *parser.ParserResult) ([]*html.Node, error) {
	getStyleSignature := func(node *html.Node) string {
		var attrs []string
		for _, attr := range node.Attr {
			attrs = append(attrs, attr.Key+"="+attr.Val)
		}
		sort.Strings(attrs)
		return strings.Join(attrs, ";")
	}
	setStyleNodeText := func(node *html.Node, text string) {
		for child := node.FirstChild; child != nil; {
			next := child.NextSibling
			node.RemoveChild(child)
			child = next
		}
		textNode := &html.Node{Type: html.TextNode, Data: text}
		node.AppendChild(textNode)
	}
	rewriteStyleContentPaths := func(content string, fromPath string, toPath string) string {
		pathRe := regexp.MustCompile(`@(import|use|forward)\s+(?:url\()?['"]([^'"]*)['"]\)?|url\(['"]([^'"]*)['"]\)`)
		return pathRe.ReplaceAllStringFunc(content, func(match string) string {
			submatches := pathRe.FindStringSubmatch(match)
			var origPath string
			if submatches[2] != "" {
				origPath = submatches[2]
			} else if submatches[3] != "" {
				origPath = submatches[3]
			} else {
				return match
			}

			fromAbsPath := origPath
			if !filepath.IsAbs(origPath) {
				fromAbsPath, _ = filepath.Abs(filepath.Join(filepath.Dir(fromPath), origPath))
			}
			relPath, _ := filepath.Rel(filepath.Dir(toPath), fromAbsPath)
			relPath = strings.ReplaceAll(relPath, `\`, `/`)
			if !strings.HasPrefix(relPath, ".") {
				relPath = "./" + relPath
			}
			return strings.Replace(match, origPath, relPath, 1)
		})
	}
	extractHoistableDirectives := func(content string) ([]string, string) {
		matches := sassModuleDirectiveRe.FindAllStringIndex(content, -1)
		if len(matches) == 0 {
			return nil, content
		}

		var remaining strings.Builder
		last := 0
		var charsets []string
		var directives []string
		for _, match := range matches {
			remaining.WriteString(content[last:match[0]])
			stmt := strings.TrimSpace(content[match[0]:match[1]])
			if stmt != "" {
				if strings.HasPrefix(strings.ToLower(stmt), "@charset") {
					charsets = append(charsets, stmt)
				} else {
					directives = append(directives, stmt)
				}
			}
			last = match[1]
		}
		remaining.WriteString(content[last:])
		return append(charsets, directives...), strings.Trim(remaining.String(), "\n")
	}
	mergeStyleContents := func(nodes []*html.Node) string {
		seenDirectives := map[string]struct{}{}
		var hoistedDirectives []string
		var bodies []string
		for _, node := range nodes {
			content := htmlquery.InnerText(node)
			if content == "" {
				continue
			}
			directives, body := extractHoistableDirectives(content)
			for _, directive := range directives {
				if _, ok := seenDirectives[directive]; ok {
					continue
				}
				seenDirectives[directive] = struct{}{}
				hoistedDirectives = append(hoistedDirectives, directive)
			}
			if strings.TrimSpace(body) != "" {
				bodies = append(bodies, body)
			}
		}

		var combined []string
		if len(hoistedDirectives) > 0 {
			combined = append(combined, strings.Join(hoistedDirectives, "\n"))
		}
		combined = append(combined, bodies...)
		return strings.Join(combined, "\n")
	}
	type styleGroup struct {
		signature string
		nodes     []*html.Node
	}

	var groups []*styleGroup
	groupBySignature := make(map[string]*styleGroup)
	appendToGroup := func(node *html.Node) {
		sig := getStyleSignature(node)
		group := groupBySignature[sig]
		if group == nil {
			group = &styleGroup{signature: sig}
			groupBySignature[sig] = group
			groups = append(groups, group)
		}
		group.nodes = append(group.nodes, node)
	}

	if result.VueComponent.Extends != "" && extendsResult != nil && extendsResult.RawStyleNodes != nil {
		for _, rawNode := range extendsResult.RawStyleNodes {
			node := vuesfchtmlparser.CloneNode(rawNode)
			setStyleNodeText(node, rewriteStyleContentPaths(htmlquery.InnerText(node), extendsResult.Path, result.Path))
			appendToGroup(node)
		}
	}

	for _, rawNode := range result.RawStyleNodes {
		appendToGroup(vuesfchtmlparser.CloneNode(rawNode))
	}

	mergedStyleNodes := make([]*html.Node, 0, len(groups))
	for _, group := range groups {
		if len(group.nodes) == 1 {
			mergedStyleNodes = append(mergedStyleNodes, vuesfchtmlparser.CloneNode(group.nodes[0]))
			continue
		}

		mergedNode := vuesfchtmlparser.CloneNode(group.nodes[0])
		setStyleNodeText(mergedNode, mergeStyleContents(group.nodes))
		mergedStyleNodes = append(mergedStyleNodes, mergedNode)
	}

	return mergedStyleNodes, nil
}

func (b *WebModuleBuilder) renderComponent(result *parser.ParserResult) (string, error) {
	var doc *html.Node
	if result.RawScriptNode != nil {
		doc = result.RawScriptNode.Parent
	} else {
		return "", xfmt.Errorf("script node node is nil")
	}

	// insert ScriptNode and remove RawScriptNode
	if result.ScriptNode != nil {
		if result.RawScriptNode != nil {
			doc.InsertBefore(result.ScriptNode, result.RawScriptNode)
			doc.RemoveChild(result.RawScriptNode)
		} else {
			emptyLine := &html.Node{
				Type: html.TextNode,
				Data: "\n\n",
			}
			doc.InsertBefore(emptyLine, result.RawScriptSetupNode)
			doc.InsertBefore(result.ScriptNode, emptyLine)

		}
	}

	// insert TemplateNode and remove RawTemplateNode
	if result.TemplateNode != nil {
		if result.RawTemplateNode != nil {
			doc.InsertBefore(result.TemplateNode, result.RawTemplateNode)
			doc.RemoveChild(result.RawTemplateNode)
		} else {
			doc.AppendChild(result.TemplateNode)
		}
	}

	// insert StyleNodes
	if len(result.StyleNodes) > 0 {
		if len(result.RawStyleNodes) > 0 {
			// Insert before first RawStyleNode if exists
			firstStyleNode := result.RawStyleNodes[0]
			for i := len(result.StyleNodes) - 1; i >= 0; i-- {
				doc.InsertBefore(result.StyleNodes[i], firstStyleNode)
				firstStyleNode = result.StyleNodes[i]
			}
		} else {
			// Append to end of document if no RawStyleNodes
			for _, styleNode := range result.StyleNodes {
				doc.AppendChild(styleNode)
			}
		}
	}

	// Remove RawStyleNodes if they exist
	if result.RawStyleNodes != nil {
		for _, node := range result.RawStyleNodes {
			doc.RemoveChild(node)
		}
	}

	sfcContent, err := vuesfchtmlparser.RenderVueSfcFromHtmlNode(doc)
	if err != nil {
		return "", xfmt.Errorf("Error rendering vue sfc from html node: %w", err)
	}

	return sfcContent, nil
}

func (b *WebModuleBuilder) getNewExtends(buildResult *module.BuildResult, component *meta.IrComponent) (*meta.IrComponent, error) {
	var extendsComponents []*meta.IrComponent
	if result := b.runtimeScope.Session().
		Where(&meta.IrComponent{Name: component.Name}).
		Order("id DESC").
		Find(&extendsComponents); result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, xfmt.Errorf("error getting last components: %w", result.Error)
		}
	}

	if component.Extends == "" {
		return nil, nil
	}

	// During a single build run, same-name extension layers may not be persisted
	// yet. Prefer already-resolved in-memory components so chaining remains stable.
	if buildResult != nil {
		existingPaths := make(map[string]bool, len(extendsComponents))
		for _, c := range extendsComponents {
			existingPaths[c.Path] = true
		}

		parserResults := module.ParserResults(buildResult)
		for i := len(parserResults) - 1; i >= 0; i-- {
			r := parserResults[i]
			if r == nil || r.VueComponent == nil || r.Content == "" {
				continue
			}
			if r.VueComponent.Name != component.Name {
				continue
			}
			if existingPaths[r.VueComponent.Path] {
				continue
			}
			extendsComponents = append([]*meta.IrComponent{r.VueComponent}, extendsComponents...)
			existingPaths[r.VueComponent.Path] = true
		}
	}

	if len(extendsComponents) == 0 {
		return nil, nil
	}

	adj := make(map[string][]string, len(extendsComponents))
	for _, c := range extendsComponents {
		if c.Extends == "" {
			continue
		}
		adj[c.Path] = append(adj[c.Path], c.Extends)
		adj[c.Extends] = append(adj[c.Extends], c.Path)
	}

	visited := map[string]bool{component.Extends: true}
	queue := []string{component.Extends}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range adj[cur] {
			if visited[next] {
				continue
			}
			visited[next] = true
			queue = append(queue, next)
		}
	}

	for _, candidate := range extendsComponents {
		if candidate.Path == component.Path || candidate.Path == component.Extends {
			continue
		}
		if visited[candidate.Path] {
			return candidate, nil
		}
	}

	return nil, nil
}

func (b *WebModuleBuilder) persist(buildResult *module.BuildResult) error {
	mod := buildResult.Module
	parserResults := module.ParserResults(buildResult)
	uiResources, warnings, err := extractUiResources(mod, parserResults)
	if err != nil {
		return xfmt.Errorf("ui resource validation failed: %w", err)
	}
	menuRouteRefs, routeActionRefs, err := extractUiResourceRelations(uiResources, parserResults)
	if err != nil {
		return xfmt.Errorf("ui resource relation validation failed: %w", err)
	}
	for _, w := range warnings {
		b.runtimeScope.Logger().Warn(
			"ui resource validation warning",
			"code", w.code,
			"resource_id", w.resourceID,
			"source_path", w.sourcePath,
			"line", w.line,
			"column", w.column,
			"hint", w.hint,
			"message", w.message,
		)
	}
	if err := b.validateUiResourceDependencies(uiResources); err != nil {
		return xfmt.Errorf("ui resource dependency validation failed: %w", err)
	}
	mod.UiResources = uiResources

	for _, result := range parserResults {
		if result.VueComponent == nil {
			continue
		}
		if strings.HasPrefix(result.VueComponent.Path, mod.Path) {
			mod.Components = append(mod.Components, result.VueComponent)
		}
	}

	if mod.Id.Valid {
		if err := b.persistModuleComponents(mod.Id.String, mod.Components); err != nil {
			return xfmt.Errorf("error persisting module components: %w", err)
		}
	}

	if mod.Id.Valid {
		if err := b.persistModuleUiResources(mod.Id.String, uiResources, menuRouteRefs, routeActionRefs); err != nil {
			return xfmt.Errorf("error persisting module UI resources: %w", err)
		}

		if len(uiResources) > 0 {
			if err := b.persistUiResourceDefaultRoles(uiResources); err != nil {
				return xfmt.Errorf("error creating role UI resource default roles: %w", err)
			}
		}
	}

	return nil
}

func (b *WebModuleBuilder) persistModuleComponents(moduleID string, components []*meta.IrComponent) error {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return nil
	}

	orderedPaths := make([]string, 0, len(components))
	componentByPath := make(map[string]*meta.IrComponent, len(components))
	for _, c := range components {
		if c == nil {
			continue
		}
		path := strings.TrimSpace(c.Path)
		if path == "" {
			continue
		}
		if _, exists := componentByPath[path]; !exists {
			orderedPaths = append(orderedPaths, path)
		}
		componentByPath[path] = c
	}

	rows := make([]*meta.IrComponent, 0, len(orderedPaths))
	for _, path := range orderedPaths {
		c := componentByPath[path]
		if c == nil {
			continue
		}
		c.ModuleId = sql.NullString{String: moduleID, Valid: true}
		rows = append(rows, c)
	}

	if result := b.runtimeScope.Session().Unscoped().Where("module_id = ?", moduleID).Delete(&meta.IrComponent{}); result.Error != nil {
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

func (b *WebModuleBuilder) persistModuleUiResources(moduleID string, uiResources []*meta.IrUiResource, menuRoutes []uiResourceMenuRouteRef, routeActions []uiResourceRouteActionRef) error {
	if strings.TrimSpace(moduleID) == "" {
		return nil
	}

	for _, r := range uiResources {
		if r == nil {
			continue
		}
		r.ModuleId = sql.NullString{String: moduleID, Valid: true}
		r.ParentId = sql.NullString{}
		r.ParentPath = ""
	}

	if len(uiResources) > 0 {
		if result := b.runtimeScope.Session().
			Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"type",
					"title",
					"sequence",
					"requires",
					"module",
					"path",
					"parent_id",
					"parent_path",
					"ir_application_id",
					"ui_path",
					"default_roles",
					"module_id",
					"updated_at",
				}),
			}).
			Create(&uiResources); result.Error != nil {
			return result.Error
		}
	}

	names := make([]string, 0, len(uiResources))
	for _, r := range uiResources {
		if r == nil {
			continue
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}

	if len(names) == 0 {
		existingRows := make([]*meta.IrUiResource, 0)
		if result := b.runtimeScope.Session().
			Model(&meta.IrUiResource{}).
			Where("module_id = ?", moduleID).
			Find(&existingRows); result.Error != nil {
			return result.Error
		}
		if err := b.deleteUiResourceRelationsByIDs(collectUiResourceIDs(existingRows)); err != nil {
			return err
		}
		if result := b.runtimeScope.Session().
			Where("module_id = ?", moduleID).
			Delete(&meta.IrUiResource{}); result.Error != nil {
			return result.Error
		}
		return nil
	}

	rows := make([]*meta.IrUiResource, 0)
	if result := b.runtimeScope.Session().
		Model(&meta.IrUiResource{}).
		Where("module_id = ?", moduleID).
		Where("name IN ?", names).
		Find(&rows); result.Error != nil {
		return result.Error
	}

	idByName := make(map[string]string, len(rows))
	for _, row := range rows {
		if row == nil || !row.Id.Valid {
			continue
		}
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		idByName[name] = row.Id.String
	}

	menuIDByName := make(map[string]string, len(uiResources))
	for _, r := range uiResources {
		if r == nil {
			continue
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			continue
		}
		if r.Type != meta.UiResourceTypeMenu {
			continue
		}
		id := strings.TrimSpace(idByName[name])
		if id == "" {
			continue
		}
		menuIDByName[name] = id
	}

	parentNameByName := make(map[string]string, len(menuIDByName))
	for _, r := range uiResources {
		if r == nil || r.Type != meta.UiResourceTypeMenu {
			continue
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			continue
		}
		parentName := strings.TrimSpace(r.ParentResourceName)
		if parentName != "" {
			if _, ok := menuIDByName[parentName]; !ok {
				parentName = ""
			}
		}
		parentNameByName[name] = parentName
	}

	parentPathByName, err := buildUiResourceParentPathByName(menuIDByName, parentNameByName)
	if err != nil {
		return err
	}

	for _, r := range uiResources {
		if r == nil {
			continue
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			continue
		}
		id := strings.TrimSpace(idByName[name])
		if id == "" {
			continue
		}

		updates := map[string]any{
			"parent_id":   nil,
			"parent_path": nil,
		}
		parentID := ""
		parentPath := ""
		if r.Type == meta.UiResourceTypeMenu {
			if parentName := parentNameByName[name]; parentName != "" {
				parentID = strings.TrimSpace(idByName[parentName])
			}
			parentPath = strings.TrimSpace(parentPathByName[name])
			if parentPath == "" {
				parentPath = id + "/"
			}
			updates["parent_path"] = parentPath
			if parentID != "" {
				updates["parent_id"] = parentID
			}
		}
		if result := b.runtimeScope.Session().
			Model(&meta.IrUiResource{}).
			Where("id = ?", id).
			Updates(updates); result.Error != nil {
			return result.Error
		}

		if parentID != "" {
			r.ParentId = sql.NullString{String: parentID, Valid: true}
		} else {
			r.ParentId = sql.NullString{}
		}
		r.ParentPath = parentPath
	}

	moduleRows := make([]*meta.IrUiResource, 0)
	if result := b.runtimeScope.Session().
		Model(&meta.IrUiResource{}).
		Where("module_id = ?", moduleID).
		Find(&moduleRows); result.Error != nil {
		return result.Error
	}
	moduleResourceIDs := collectUiResourceIDs(moduleRows)
	if err := b.replaceUiResourceRelations(moduleResourceIDs, menuRoutes, routeActions); err != nil {
		return err
	}

	if result := b.runtimeScope.Session().
		Where("module_id = ?", moduleID).
		Where("name NOT IN ?", names).
		Delete(&meta.IrUiResource{}); result.Error != nil {
		return result.Error
	}

	return nil
}

func buildUiResourceParentPathByName(idByName map[string]string, parentNameByName map[string]string) (map[string]string, error) {
	parentPathByName := make(map[string]string, len(idByName))
	visiting := make(map[string]bool, len(idByName))

	var resolve func(name string) (string, error)
	resolve = func(name string) (string, error) {
		if path, ok := parentPathByName[name]; ok {
			return path, nil
		}
		if visiting[name] {
			return "", xfmt.Errorf("ui resource parent cycle detected at %s", name)
		}
		visiting[name] = true
		defer delete(visiting, name)

		id := strings.TrimSpace(idByName[name])
		if id == "" {
			return "", xfmt.Errorf("ui resource id not found for name %s", name)
		}

		parentName := strings.TrimSpace(parentNameByName[name])
		if parentName == "" {
			path := id + "/"
			parentPathByName[name] = path
			return path, nil
		}
		if _, ok := idByName[parentName]; !ok {
			path := id + "/"
			parentPathByName[name] = path
			return path, nil
		}

		parentPath, err := resolve(parentName)
		if err != nil {
			return "", err
		}
		path := parentPath + id + "/"
		parentPathByName[name] = path
		return path, nil
	}

	for name := range idByName {
		if _, err := resolve(name); err != nil {
			return nil, err
		}
	}

	return parentPathByName, nil
}

func parentIDFromPath(parentPath string, selfID string) string {
	path := strings.TrimSpace(parentPath)
	if path == "" {
		return ""
	}
	partsRaw := strings.Split(path, "/")
	parts := make([]string, 0, len(partsRaw))
	for _, p := range partsRaw {
		v := strings.TrimSpace(p)
		if v != "" {
			parts = append(parts, v)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if last == selfID {
		if len(parts) >= 2 {
			return parts[len(parts)-2]
		}
		return ""
	}
	if last == "" || last == selfID {
		return ""
	}
	return last
}

func collectUiResourceIDs(rows []*meta.IrUiResource) []string {
	ids := make([]string, 0, len(rows))
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row == nil || !row.Id.Valid {
			continue
		}
		id := strings.TrimSpace(row.Id.String)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func (b *WebModuleBuilder) deleteUiResourceRelationsByIDs(resourceIDs []string) error {
	if len(resourceIDs) == 0 {
		return nil
	}
	if result := b.runtimeScope.Session().
		Unscoped().
		Where("menu_ui_resource_id IN ? OR route_ui_resource_id IN ?", resourceIDs, resourceIDs).
		Delete(&meta.IrUiResourceMenuRoute{}); result.Error != nil {
		return result.Error
	}
	if result := b.runtimeScope.Session().
		Unscoped().
		Where("route_ui_resource_id IN ? OR action_ui_resource_id IN ?", resourceIDs, resourceIDs).
		Delete(&meta.IrUiResourceRouteAction{}); result.Error != nil {
		return result.Error
	}
	return nil
}

func (b *WebModuleBuilder) replaceUiResourceRelations(moduleResourceIDs []string, menuRoutes []uiResourceMenuRouteRef, routeActions []uiResourceRouteActionRef) error {
	if err := b.deleteUiResourceRelationsByIDs(moduleResourceIDs); err != nil {
		return err
	}
	if len(menuRoutes) == 0 && len(routeActions) == 0 {
		return nil
	}

	rows := make([]*meta.IrUiResource, 0)
	if result := b.runtimeScope.Session().
		Model(&meta.IrUiResource{}).
		Where("id IN ?", moduleResourceIDs).
		Find(&rows); result.Error != nil {
		return result.Error
	}

	idByName := make(map[string]string, len(rows))
	typeByName := make(map[string]meta.UiResourceType, len(rows))
	for _, row := range rows {
		if row == nil || !row.Id.Valid {
			continue
		}
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		idByName[name] = strings.TrimSpace(row.Id.String)
		typeByName[name] = row.Type
	}

	menuRouteRows := make([]*meta.IrUiResourceMenuRoute, 0, len(menuRoutes))
	seenMenuRoutes := make(map[string]bool, len(menuRoutes))
	for _, ref := range menuRoutes {
		menuName := strings.TrimSpace(ref.MenuName)
		routeName := strings.TrimSpace(ref.RouteName)
		if menuName == "" || routeName == "" {
			continue
		}
		menuID := strings.TrimSpace(idByName[menuName])
		routeID := strings.TrimSpace(idByName[routeName])
		if menuID == "" || routeID == "" {
			continue
		}
		if typeByName[menuName] != meta.UiResourceTypeMenu || typeByName[routeName] != meta.UiResourceTypeRoute {
			continue
		}
		key := menuID + "/" + routeID
		if seenMenuRoutes[key] {
			continue
		}
		seenMenuRoutes[key] = true
		menuRouteRows = append(menuRouteRows, &meta.IrUiResourceMenuRoute{
			MenuUiResourceId:  sql.NullString{String: menuID, Valid: true},
			RouteUiResourceId: sql.NullString{String: routeID, Valid: true},
		})
	}

	routeActionRows := make([]*meta.IrUiResourceRouteAction, 0, len(routeActions))
	seenRouteActions := make(map[string]bool, len(routeActions))
	for _, ref := range routeActions {
		routeName := strings.TrimSpace(ref.RouteName)
		actionName := strings.TrimSpace(ref.ActionName)
		if routeName == "" || actionName == "" {
			continue
		}
		routeID := strings.TrimSpace(idByName[routeName])
		actionID := strings.TrimSpace(idByName[actionName])
		if routeID == "" || actionID == "" {
			continue
		}
		if typeByName[routeName] != meta.UiResourceTypeRoute || typeByName[actionName] != meta.UiResourceTypeAction {
			continue
		}
		key := routeID + "/" + actionID
		if seenRouteActions[key] {
			continue
		}
		seenRouteActions[key] = true
		routeActionRows = append(routeActionRows, &meta.IrUiResourceRouteAction{
			RouteUiResourceId:  sql.NullString{String: routeID, Valid: true},
			ActionUiResourceId: sql.NullString{String: actionID, Valid: true},
		})
	}

	if len(menuRouteRows) > 0 {
		if result := b.runtimeScope.Session().Create(&menuRouteRows); result.Error != nil {
			return result.Error
		}
	}
	if len(routeActionRows) > 0 {
		if result := b.runtimeScope.Session().Create(&routeActionRows); result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func (b *WebModuleBuilder) validateUiResourceDependencies(uiResources []*meta.IrUiResource) error {
	if len(uiResources) == 0 {
		return nil
	}

	type requirementRef struct {
		resourceID string
		requireRaw string
		modelKey   string
		method     string
	}

	modelRefSet := make(map[string]bool)
	requirements := make([]requirementRef, 0)

	for _, r := range uiResources {
		if r == nil {
			continue
		}
		requires := parseJSONStrings(r.Requires)
		for _, raw := range requires {
			modelKey, method, ok := parseRpcRequire(raw)
			if !ok {
				return xfmt.Errorf("resource %s has invalid requires entry %q", r.Name, raw)
			}
			modelRefSet[modelKey] = true
			requirements = append(requirements, requirementRef{
				resourceID: r.Name,
				requireRaw: raw,
				modelKey:   modelKey,
				method:     method,
			})
		}
	}

	if len(modelRefSet) == 0 {
		return nil
	}

	models := make([]*meta.IrModel, 0)
	if result := b.runtimeScope.Session().
		Model(&meta.IrModel{}).
		Select("id", "application", "name").
		Find(&models); result.Error != nil {
		return xfmt.Errorf("query meta models failed: %w", result.Error)
	}

	modelIDByKey := make(map[string]string)
	for _, m := range models {
		if m == nil || !m.Id.Valid {
			continue
		}
		k := strings.TrimSpace(m.Application) + "." + strings.TrimSpace(m.Name)
		if strings.TrimSpace(k) == "." {
			continue
		}
		modelIDByKey[k] = m.Id.String
	}

	for mk := range modelRefSet {
		if _, ok := modelIDByKey[mk]; !ok {
			return xfmt.Errorf("referenced model %q not found in meta.IrModel", mk)
		}
	}

	modelIDs := make([]string, 0, len(modelIDByKey))
	for mk := range modelRefSet {
		modelIDs = append(modelIDs, modelIDByKey[mk])
	}

	services := make([]*meta.IrService, 0)
	if result := b.runtimeScope.Session().
		Model(&meta.IrService{}).
		Select("model_id", "name").
		Where("model_id in ?", modelIDs).
		Find(&services); result.Error != nil {
		return xfmt.Errorf("query meta services failed: %w", result.Error)
	}

	modelKeyByID := make(map[string]string)
	for k, id := range modelIDByKey {
		modelKeyByID[id] = k
	}

	serviceKeySet := make(map[string]bool)
	for _, s := range services {
		if s == nil || !s.ModelId.Valid {
			continue
		}
		mk := modelKeyByID[s.ModelId.String]
		if mk == "" {
			continue
		}
		method := strings.ToLower(strings.TrimSpace(s.Name))
		if method == "" {
			continue
		}
		serviceKeySet[mk+"/"+method] = true
	}

	for _, req := range requirements {
		if req.method == "*" {
			continue
		}
		if !serviceKeySet[req.modelKey+"/"+strings.ToLower(req.method)] {
			return xfmt.Errorf("resource %s requires %q but service method not found", req.resourceID, req.requireRaw)
		}
	}

	return nil
}

func normalizeModelKey(v string) (string, bool) {
	s := strings.TrimSpace(v)
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return "", false
	}
	app := strings.TrimSpace(parts[0])
	model := strings.TrimSpace(parts[1])
	if app == "" || model == "" {
		return "", false
	}
	return app + "." + model, true
}

func parseRpcRequire(raw string) (modelKey string, method string, ok bool) {
	v := strings.TrimSpace(raw)
	if strings.HasPrefix(v, "service:/") {
		v = "rpc:/" + strings.TrimPrefix(v, "service:/")
	}
	if !strings.HasPrefix(v, "rpc:/") {
		return "", "", false
	}
	body := strings.TrimPrefix(v, "rpc:/")
	parts := strings.Split(body, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	modelKey, ok = normalizeModelKey(parts[0])
	if !ok {
		return "", "", false
	}
	method = strings.TrimSpace(parts[1])
	if method == "" {
		return "", "", false
	}
	return modelKey, method, true
}

func parseJSONStrings(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	norm := make([]string, 0, len(out))
	for _, v := range out {
		s := strings.TrimSpace(v)
		if s == "" {
			continue
		}
		norm = append(norm, s)
	}
	return norm
}

func collectUiResourceDefaultRoleRows(
	uiResources []*meta.IrUiResource,
	roleIDByCode map[string]string,
) ([]roleUiResourceGrantRow, error) {
	if len(uiResources) == 0 {
		return nil, nil
	}

	rows := make([]roleUiResourceGrantRow, 0)
	seen := make(map[string]bool)
	for _, resource := range uiResources {
		if resource == nil || !resource.Id.Valid {
			continue
		}
		for _, roleCode := range parseJSONStrings(resource.DefaultRoles) {
			code := strings.TrimSpace(roleCode)
			if code == "" {
				continue
			}
			roleID := strings.TrimSpace(roleIDByCode[code])
			if roleID == "" {
				return nil, xfmt.Errorf("defaultRoles role %q not found", code)
			}

			grantKey := roleID + "/" + resource.Id.String
			if seen[grantKey] {
				continue
			}
			seen[grantKey] = true

			rows = append(rows, roleUiResourceGrantRow{
				RoleId:         sql.NullString{String: roleID, Valid: true},
				Mode:           "allow",
				IrUiResourceId: sql.NullString{String: resource.Id.String, Valid: true},
			})
		}
	}

	return rows, nil
}

func (b *WebModuleBuilder) persistUiResourceDefaultRoles(uiResources []*meta.IrUiResource) error {
	roleCodeSet := make(map[string]bool)
	for _, resource := range uiResources {
		if resource == nil {
			continue
		}
		for _, roleCode := range parseJSONStrings(resource.DefaultRoles) {
			code := strings.TrimSpace(roleCode)
			if code == "" {
				continue
			}
			roleCodeSet[code] = true
		}
	}

	if len(roleCodeSet) == 0 {
		return nil
	}

	session := b.runtimeScope.Session()
	if session == nil || session.DB == nil {
		return nil
	}
	if !session.Migrator().HasTable("auth_role") || !session.Migrator().HasTable("auth_role_ui_resource") {
		return nil
	}

	roleCodes := make([]string, 0, len(roleCodeSet))
	for roleCode := range roleCodeSet {
		roleCodes = append(roleCodes, roleCode)
	}

	roleRows := make([]roleCodeRow, 0, len(roleCodes))
	if result := session.
		Table("auth_role").
		Select("id", "code").
		Where("code in ?", roleCodes).
		Find(&roleRows); result.Error != nil {
		return xfmt.Errorf("query defaultRoles failed: %w", result.Error)
	}

	roleIDByCode := make(map[string]string, len(roleRows))
	for _, row := range roleRows {
		code := strings.TrimSpace(row.Code)
		id := strings.TrimSpace(row.Id)
		if code == "" || id == "" {
			continue
		}
		roleIDByCode[code] = id
	}

	grantRows, err := collectUiResourceDefaultRoleRows(uiResources, roleIDByCode)
	if err != nil {
		return err
	}
	if len(grantRows) == 0 {
		return nil
	}

	if result := session.
		Table("auth_role_ui_resource").
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&grantRows); result.Error != nil {
		return xfmt.Errorf("insert defaultRoles role UI resources failed: %w", result.Error)
	}

	return nil
}

func (b *WebModuleBuilder) buildOptions(prebuild bool, extraEsbOpts ...esbplugins.EsbPluginOptions) *api.BuildOptions {
	runtimeOptions := b.resolvedRuntimeOptions()
	modules_path := runtimeOptions.modulesPath
	dist_path := runtimeOptions.distPath
	webBaseUrl := strings.TrimSuffix(runtimeOptions.webBaseURL, "/") + "/"

	buildOptions := api.BuildOptions{
		EntryPoints: []string{b.entryPoint},
		PublicPath:  webBaseUrl + "assets",
		Tsconfig:    filepath.Join(modules_path, "tsconfig.json"),
		Loader: map[string]api.Loader{
			".png":  api.LoaderFile,
			".scss": api.LoaderCSS,
			".sass": api.LoaderCSS,
			".css":  api.LoaderCSS,
			".svg":  api.LoaderFile,
		},
		Target:     api.ES2020,
		Platform:   api.PlatformBrowser,
		Format:     api.FormatESModule,
		Bundle:     true,
		Outdir:     filepath.Join(dist_path, "web", "assets"),
		EntryNames: "[dir]/[name]-[hash]",
		Splitting:  true,
		Metafile:   true,
		Write:      true,
	}

	jsonFrontendEnv, err := json.Marshal(runtimeOptions.frontendEnv)
	if err != nil {
		jsonFrontendEnv = []byte("{}")
		b.runtimeScope.Logger().Warn("frontend env marshal failed", "error", err)
	}

	buildOptions.Define = map[string]string{
		"import.meta.env": string(jsonFrontendEnv),
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
	if len(extraEsbOpts) > 0 {
		esbOpts = append(esbOpts, extraEsbOpts...)
	}

	if prebuild {
		basePlugins := b.prebuildPlugin.DefinePlugins(b.runtimeScope, b.jsExecutor, b.module, esbOpts...)
		// Prepend the ESM resolver so bare imports are intercepted before
		// other plugins process them.
		webResolverOpts := []esmresolver.Option{
			esmresolver.WithCacheDir(runtimeOptions.defaultChoysumPath),
			esmresolver.WithModulePath(b.module.Path),
		}
		if upstream := strings.TrimSpace(runtimeOptions.esmUpstreamURL); upstream != "" {
			webResolverOpts = append(webResolverOpts, esmresolver.WithUpstream(upstream))
		}
		buildOptions.Plugins = append([]api.Plugin{esmresolver.New(webResolverOpts...).Plugin()}, basePlugins...)
		buildOptions.Write = false
	} else {
		basePlugins := b.buildPlugin.DefinePlugins(b.runtimeScope, b.jsExecutor, b.module, esbOpts...)
		// Prepend the ESM resolver so bare imports are intercepted before
		// other plugins process them.
		webResolverOpts := []esmresolver.Option{
			esmresolver.WithCacheDir(runtimeOptions.defaultChoysumPath),
			esmresolver.WithModulePath(b.module.Path),
		}
		if upstream := strings.TrimSpace(runtimeOptions.esmUpstreamURL); upstream != "" {
			webResolverOpts = append(webResolverOpts, esmresolver.WithUpstream(upstream))
		}
		buildOptions.Plugins = append([]api.Plugin{esmresolver.New(webResolverOpts...).Plugin()}, basePlugins...)
		buildOptions.Write = false
	}

	return &buildOptions
}

func (b *WebModuleBuilder) entryPointImports() []string {
	moduleImports := make([]string, 0)
	storeImports := make([]string, 0)
	seen := map[string]struct{}{}
	runtimeOptions := b.resolvedRuntimeOptions()

	var installModules []*meta.IrModule
	if result := b.runtimeScope.Session().
		Where("status = ?", "installed").
		Order("id DESC").
		Find(&installModules); result.Error != nil {
		b.runtimeScope.Logger().Error("web entrypoint import modules lookup failed", "module_path", b.module.Path, "error", result.Error)
		return moduleImports
	}
	for _, m := range installModules {
		if m.WebEntryPoint == "" {
			continue
		}
		// Avoid importing the web module entrypoint into itself.
		if strings.EqualFold(strings.TrimSpace(m.Name), strings.TrimSpace(b.module.Name)) {
			continue
		}

		ep := strings.TrimSpace(m.WebEntryPoint)
		if ep == "" {
			continue
		}
		ep = strings.TrimSuffix(ep, ".ts")

		// Normalize entrypoint to an import path resolvable by tsconfig paths.
		// - If absolute: convert to modules-relative then prefix with @/
		// - If relative (manifest style like "./web/index.ts"): join with module name
		var importPath string
		if filepath.IsAbs(ep) {
			rel, err := filepath.Rel(runtimeOptions.modulesPath, ep)
			if err == nil {
				importPath = "@/" + filepath.ToSlash(rel)
			} else {
				importPath = filepath.ToSlash(ep)
			}
		} else {
			ep = strings.TrimPrefix(ep, "./")
			ep = strings.TrimPrefix(ep, ".\\")
			importPath = "@/" + filepath.ToSlash(filepath.Join(m.Name, ep))
		}
		if _, ok := seen[importPath]; ok {
			continue
		}
		seen[importPath] = struct{}{}
		moduleImports = append(moduleImports, importPath)
	}

	var installApplications []*meta.IrApplication
	if result := b.runtimeScope.Session().
		Order("id DESC").
		Find(&installApplications); result.Error != nil {
		b.runtimeScope.Logger().Error("web entrypoint import applications lookup failed", "module_path", b.module.Path, "error", result.Error)
		return moduleImports
	}
	for _, app := range installApplications {
		_, workspaceWebDir, _, err := modulegenerator.WorkspaceGeneratedAPITargets(runtimeOptions.modulesPath, app.Name, runtimeOptions.defaultChoysumPath)
		if err != nil {
			continue
		}
		workspaceStorePath := filepath.Join(workspaceWebDir, "stores", "index.ts")
		if _, err := os.Stat(workspaceStorePath); err != nil {
			continue
		}

		importPath := workspaceStorePath
		if absPath, err := filepath.Abs(importPath); err == nil {
			importPath = absPath
		}
		importPath = filepath.ToSlash(filepath.Clean(importPath))
		if _, ok := seen[importPath]; ok {
			continue
		}
		seen[importPath] = struct{}{}
		storeImports = append(storeImports, importPath)
	}

	// Store side-effect imports must run before module entrypoints so factory
	// registration is available to top-level createStoreByModel(...) calls.
	return append(storeImports, moduleImports...)
}

func NewWebBuilder(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.IrModule, entryPoint string, opts ...func(*WebModuleBuilder)) module.Builder {
	b := &WebModuleBuilder{
		runtimeScope:   runtimeScope,
		runtimeOptions: newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.RuntimeEnvironmentOptions{}, false, scope.CompileRuntimeOptions{}, false),
		module:         module,
		jsExecutor:     jsExecutor,
		entryPoint:     entryPoint,
		publishDist:    true,
		parserFactory:  vueparser.NewVueParser,
	}

	for _, opt := range opts {
		opt(b)
	}

	if b.prebuildPlugin == nil {
		b.prebuildPlugin = webprebuildplugin.NewWebPrebuildPlugin(runtimeScope, module, entryPoint)
	}

	if b.buildPlugin == nil {
		b.buildPlugin = internalwebplugin.NewWebPlugin(runtimeScope, module, entryPoint)
	}

	if b.parser == nil {
		b.parser = b.parserFactory(runtimeScope, module)
	}
	if b.runtimeScope != nil {
		b.runtimeOptions = runtimeOptionsFromScope(b.runtimeScope)
	}

	return b
}

func WithPublishDist(publish bool) func(*WebModuleBuilder) {
	return func(b *WebModuleBuilder) {
		b.publishDist = publish
	}
}

func WithBuildPlugin(buildPlugin esbplugins.EsbPlugin) func(*WebModuleBuilder) {
	return func(b *WebModuleBuilder) {
		b.buildPlugin = buildPlugin
	}
}

func WithPrebuildPlugin(prebuildPlugin esbplugins.EsbPlugin) func(*WebModuleBuilder) {
	return func(b *WebModuleBuilder) {
		b.prebuildPlugin = prebuildPlugin
	}
}
