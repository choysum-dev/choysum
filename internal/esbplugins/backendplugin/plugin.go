// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendplugin

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/internal/parser/backendtsparser"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
	"github.com/evanw/esbuild/pkg/api"
	xfmt "golang.org/x/exp/errors/fmt"
)

// BackendPlugin wires backend parser results into the shared esbuild plugin flow.
type BackendPlugin struct {
	*esbplugins.BasePlugin
	EntryPointImports []string
	parserFactory     func(scope.Scope, *meta.IrModule) parser.Parser
	runtimeOptions    runtimeOptions
}

func (p *BackendPlugin) bindRuntimeState(runtimeScope scope.Scope, module *meta.IrModule) {
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
		p.parserFactory = backendtsparser.NewTsParser
	}
	p.Parser = p.parserFactory(p.Env, p.Module)
	if p.Env != nil {
		p.runtimeOptions = runtimeOptionsFromScope(p.Env)
	}
}

// SetEntryPointImports stores imports that should be prepended to the backend entry point.
func (p *BackendPlugin) SetEntryPointImports(imports []string) {
	p.EntryPointImports = append([]string(nil), imports...)
}

func normalizeBackendPluginImportPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	normalized := filepath.ToSlash(filepath.Clean(trimmed))
	return strings.TrimSuffix(normalized, ".ts")
}

func normalizeBackendPluginPath(path string) string {
	return esbplugins.NormalizePath(path)
}

func backendPluginPathWithinRoot(path string, root string) bool {
	return esbplugins.PathWithinRoot(path, root)
}

func normalizeBackendPluginModuleSpecPath(path string) string {
	return esbplugins.NormalizeModuleSpecPath(path)
}

func backendPluginSameModuleSpecPath(a string, b string) bool {
	normalizedA := normalizeBackendPluginModuleSpecPath(a)
	normalizedB := normalizeBackendPluginModuleSpecPath(b)
	if normalizedA == "" || normalizedB == "" {
		return false
	}
	return filepath.ToSlash(normalizedA) == filepath.ToSlash(normalizedB)
}

func newBackendPluginModuleSpecPathComparer() func(a string, b string) bool {
	normalizedCache := make(map[string]string)
	normalizedModuleSpecPath := func(path string) string {
		path = strings.TrimSpace(path)
		if path == "" {
			return ""
		}
		if normalized, ok := normalizedCache[path]; ok {
			return normalized
		}
		normalized := normalizeBackendPluginModuleSpecPath(path)
		normalizedCache[path] = normalized
		return normalized
	}
	return func(a string, b string) bool {
		normalizedA := normalizedModuleSpecPath(a)
		normalizedB := normalizedModuleSpecPath(b)
		if normalizedA == "" || normalizedB == "" {
			return false
		}
		return filepath.ToSlash(normalizedA) == filepath.ToSlash(normalizedB)
	}
}

func isBackendPluginIrModelTableMissingError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table: meta_ir_model") || strings.Contains(msg, `relation "meta_ir_model" does not exist`)
}

func (p *BackendPlugin) isEntryPointPath(path string) bool {
	if p.SameFilePath(path, p.EntryPoint) {
		return true
	}

	current := normalizeBackendPluginImportPath(path)
	if current == "" {
		return false
	}

	entry := normalizeBackendPluginImportPath(p.EntryPoint)
	if entry == "" {
		return false
	}
	if current == entry {
		return true
	}

	if p.Module == nil {
		return false
	}
	modulePath := strings.TrimSpace(p.Module.Path)
	if modulePath == "" {
		return false
	}
	relEntry := strings.TrimPrefix(entry, "./")
	relEntry = strings.TrimPrefix(relEntry, ".\\")
	joinedRaw := filepath.Join(modulePath, filepath.FromSlash(relEntry))
	if p.SameFilePath(path, joinedRaw) {
		return true
	}
	joined := normalizeBackendPluginImportPath(joinedRaw)
	return current == joined
}

func (p *BackendPlugin) appendEntryPointImports(content string) string {
	if len(p.EntryPointImports) == 0 {
		return content
	}
	missing := make([]string, 0, len(p.EntryPointImports))
	seenMissing := make(map[string]struct{}, len(p.EntryPointImports))
	for _, importPath := range p.EntryPointImports {
		importPath = strings.TrimSpace(importPath)
		if importPath == "" {
			continue
		}
		singleQuoteStmt := fmt.Sprintf("import '%s';", importPath)
		if _, ok := seenMissing[singleQuoteStmt]; ok {
			continue
		}
		doubleQuoteStmt := fmt.Sprintf("import \"%s\";", importPath)
		if strings.Contains(content, singleQuoteStmt) || strings.Contains(content, doubleQuoteStmt) {
			continue
		}
		missing = append(missing, singleQuoteStmt)
		seenMissing[singleQuoteStmt] = struct{}{}
	}
	if len(missing) == 0 {
		return content
	}
	prefix := strings.Join(missing, "\n") + "\n"
	return prefix + content
}

func (p *BackendPlugin) replaceModuleSpecReferenceIdent(parserResults []*parser.ParserResult) error {
	// Use the new Field decorator only.
	fieldDecoratorModuleSpec, fieldDecoratorReferenceIdent := meta.FieldDecoratorModuleSpec(p.Env)
	modelDecoratorModuleSpec, modelDecoratorReferenceIdent := meta.ModelDecoratorModuleSpec(p.Env)
	sameModuleSpecPath := newBackendPluginModuleSpecPathComparer()

	moduleAbsPath := p.Module.Path
	for _, parserResult := range parserResults {
		if !backendPluginPathWithinRoot(parserResult.Path, moduleAbsPath) {
			continue
		}

		model := parserResult.Model
		if model == nil {
			continue
		}

		model.Path = normalizeBackendPluginPath(model.Path)

		if parserResult.ModelExtendsProperty != nil {
			moduleSpec, referenceIdent := p.FindModuleSpecAndReferenceIdent(parserResult.ModelExtendsProperty.ModuleSpecPath, parserResult.ModelExtendsProperty.ReferenceIdent)
			if moduleSpec != "" {
				parserResult.ModelExtendsProperty.ModuleSpecPath = moduleSpec
				parserResult.ModelExtendsProperty.ReferenceIdent = referenceIdent
			}

			if parserResult.ModelExtendsProperty.ReferenceIdent != "default" {
				return xfmt.Errorf("model should extend default: path=%s module_spec=%s reference_ident=%s", parserResult.Path, parserResult.ModelExtendsProperty.ModuleSpecPath, parserResult.ModelExtendsProperty.ReferenceIdent)
			}

			model.RawExtends = parserResult.ModelExtendsProperty.ModuleSpecPath + ".ts"
			model.Extends = model.RawExtends
		}

		model.Abstract = true
		model.Application = p.Module.ApplicationStr
		for _, decorator := range model.Decorators {
			moduleSpec, referenceIdent := p.FindModuleSpecAndReferenceIdent(decorator.ModuleSpecPath, decorator.ReferenceIdent)
			if moduleSpec != "" {
				decorator.ModuleSpecPath = moduleSpec
				decorator.ReferenceIdent = referenceIdent
			}

			if sameModuleSpecPath(decorator.ModuleSpecPath, modelDecoratorModuleSpec) && decorator.ReferenceIdent == modelDecoratorReferenceIdent {
				appName := model.Application
				if len(decorator.Arguments) > 0 {
					arg := decorator.Arguments[0]
					if arg.Type == "Literal" {
						model.Name = strings.Trim(arg.Value, "'\"")
						model.Abstract = false
					}
					if len(decorator.Arguments) > 1 {
						arg = decorator.Arguments[1]
						if arg.Type == "ObjectLiteral" {
							var options map[string]interface{}
							if err := json.Unmarshal([]byte(arg.Value), &options); err != nil {
								return xfmt.Errorf("error unmarshal model decorator options: %w", err)
							}
							if tableName, ok := options["tableName"].(string); ok && tableName != "" {
								model.ModelTable = tableName
							}
							if application, ok := options["application"].(string); ok && application != "" {
								appName = application
								model.Application = application
							}
							if autoMigrate, ok := options["autoMigrate"].(bool); ok {
								model.AutoMigrate = &autoMigrate
							}
							if readonly, ok := options["readonly"].(bool); ok {
								model.Readonly = readonly
							}
						}
					}
					if model.ModelTable == "" && model.Name != "" {
						model.ModelTable = strcase.ToSnake(fmt.Sprintf("%s_%s", appName, model.Name))
					}

				}
			}
		}

		modelServices := make([]*meta.IrService, 0)
		for _, service := range model.Services {

			for _, decorator := range service.Decorators {
				moduleSpec, referenceIdent := p.FindModuleSpecAndReferenceIdent(decorator.ModuleSpecPath, decorator.ReferenceIdent)
				if moduleSpec != "" {
					decorator.ModuleSpecPath = moduleSpec
					decorator.ReferenceIdent = referenceIdent
				}
			}
			for _, typeParam := range service.TypeParameters {
				moduleSpec, referenceIdent := p.FindModuleSpecAndReferenceIdent(typeParam.ModuleSpecPath, typeParam.ReferenceIdent)
				if moduleSpec != "" {
					typeParam.ModuleSpecPath = moduleSpec
					typeParam.ReferenceIdent = referenceIdent
				}
			}

			if meta.IsConventionalModelService(service.AccessibilityModifier, service.IsStatic, service.Name) {
				modelServices = append(modelServices, service)
			}
		}
		model.Services = modelServices

		modelFields := make([]*meta.IrField, 0)
		for _, field := range model.Fields {
			moduleSpec, referenceIdent := p.FindModuleSpecAndReferenceIdent(field.ModuleSpecPath, field.ReferenceIdent)
			field.ModuleSpecPath = moduleSpec
			field.ReferenceIdent = referenceIdent

			for _, decorator := range field.Decorators {
				moduleSpec, referenceIdent := p.FindModuleSpecAndReferenceIdent(decorator.ModuleSpecPath, decorator.ReferenceIdent)
				if moduleSpec != "" {
					decorator.ModuleSpecPath = moduleSpec
					decorator.ReferenceIdent = referenceIdent
				}
				for _, arg := range decorator.Arguments {
					moduleSpec, referenceIdent := p.FindModuleSpecAndReferenceIdent(arg.ModuleSpecPath, arg.ReferenceIdent)
					if moduleSpec != "" {
						arg.ModuleSpecPath = moduleSpec
						arg.ReferenceIdent = referenceIdent
					}
				}
			}

			// Recognize the new Field decorator only.
			isField := false
			for _, decorator := range field.Decorators {
				if decorator.Name == "Field" && sameModuleSpecPath(decorator.ModuleSpecPath, fieldDecoratorModuleSpec) && decorator.ReferenceIdent == fieldDecoratorReferenceIdent {
					isField = true
					break
				}
			}
			if isField {
				modelFields = append(modelFields, field)
			}
		}
		model.Fields = modelFields
	}

	return nil
}

func (p *BackendPlugin) injectModelApplication(parserResults []*parser.ParserResult) error {
	modelDecoratorModuleSpec, modelDecoratorReferenceIdent := meta.ModelDecoratorModuleSpec(p.Env)
	moduleAbsPath := p.Module.Path
	sameModuleSpecPath := newBackendPluginModuleSpecPathComparer()

	type contentEdit struct {
		rawPos int
		delta  int
	}
	type contentState struct {
		content string
		edits   []contentEdit
	}
	contentStateByPath := make(map[string]*contentState)
	getContentState := func(result *parser.ParserResult) *contentState {
		state, ok := contentStateByPath[result.Path]
		if ok {
			return state
		}

		raw := result.RawContent
		content := result.Content
		if content == "" {
			content = raw
		}

		state = &contentState{
			content: content,
			edits:   make([]contentEdit, 0),
		}
		contentStateByPath[result.Path] = state
		return state
	}
	mapRawOffsetToCurrent := func(state *contentState, rawPos int) int {
		currentPos := rawPos
		for _, edit := range state.edits {
			if rawPos >= edit.rawPos {
				currentPos += edit.delta
			}
		}
		return currentPos
	}
	applyRawInsertion := func(state *contentState, rawPos int, text string) bool {
		insertPos := mapRawOffsetToCurrent(state, rawPos)
		if insertPos <= 0 || insertPos > len(state.content) {
			return false
		}

		state.content = state.content[:insertPos] + text + state.content[insertPos:]
		state.edits = append(state.edits, contentEdit{rawPos: rawPos, delta: len(text)})
		return true
	}
	applyRawReplacement := func(state *contentState, rawStart int, rawEnd int, replacement string) bool {
		if rawStart < 0 || rawEnd <= rawStart {
			return false
		}

		start := mapRawOffsetToCurrent(state, rawStart)
		end := mapRawOffsetToCurrent(state, rawEnd)
		if start < 0 || end <= start || end > len(state.content) {
			return false
		}

		state.content = state.content[:start] + replacement + state.content[end:]
		state.edits = append(state.edits, contentEdit{rawPos: rawEnd, delta: len(replacement) - (rawEnd - rawStart)})
		return true
	}

	// 1. Collect external paths
	var externalPaths []string
	for _, result := range parserResults {
		if result.ModelClassNode == nil {
			continue
		}
		if !backendPluginPathWithinRoot(result.Path, moduleAbsPath) {
			externalPaths = append(externalPaths, result.Path)
		}
	}

	// 2. Batch query external applications
	externalAppMap := make(map[string]string)
	externalModuleAppMap := make(map[string]string)
	runtimeOptions := p.resolvedRuntimeOptions()
	normalizedModulesPath := normalizeBackendPluginPath(runtimeOptions.modulesPath)
	moduleNameFromPath := func(path string) string {
		if normalizedModulesPath == "" {
			return ""
		}
		normalizedPath := normalizeBackendPluginPath(path)
		if normalizedPath == "" {
			return ""
		}
		rel, err := filepath.Rel(normalizedModulesPath, normalizedPath)
		if err != nil {
			return ""
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "../") || rel == ".." {
			return ""
		}
		parts := strings.Split(rel, "/")
		if len(parts) == 0 {
			return ""
		}
		return strings.TrimSpace(parts[0])
	}

	if len(externalPaths) > 0 {
		var session *scope.Session
		if p.Env != nil {
			session = p.Env.Session()
		}

		queryPathSet := make(map[string]struct{})
		moduleNameSet := make(map[string]struct{})
		var queryPaths []string
		var moduleNames []string
		addQueryPath := func(path string) {
			path = strings.TrimSpace(path)
			if path == "" {
				return
			}
			if _, ok := queryPathSet[path]; ok {
				return
			}
			queryPathSet[path] = struct{}{}
			queryPaths = append(queryPaths, path)
		}

		for _, path := range externalPaths {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			if moduleName := moduleNameFromPath(path); moduleName != "" {
				if _, ok := moduleNameSet[moduleName]; !ok {
					moduleNameSet[moduleName] = struct{}{}
					moduleNames = append(moduleNames, moduleName)
				}
			}
			addQueryPath(path)
			if strings.HasSuffix(path, ".ts") {
				addQueryPath(strings.TrimSuffix(path, ".ts"))
			} else {
				addQueryPath(path + ".ts")
			}
		}

		if session != nil && len(moduleNames) > 0 {
			var modules []meta.IrModule
			if err := session.Where("name IN ?", moduleNames).Find(&modules).Error; err == nil {
				for _, mod := range modules {
					name := strings.TrimSpace(mod.Name)
					app := strings.TrimSpace(mod.ApplicationStr)
					if name != "" && app != "" {
						externalModuleAppMap[name] = app
					}
				}
			}
		}

		if session != nil && len(queryPaths) > 0 {
			var models []meta.IrModel
			if err := session.
				Preload("Module.Application").
				Where("path IN ?", queryPaths).
				Find(&models).Error; err != nil {
				if !isBackendPluginIrModelTableMissingError(err) {
					return xfmt.Errorf("failed to batch load external models: %w", err)
				}
				models = nil
			}

			for _, m := range models {
				if m.Module != nil && m.Module.Application != nil {
					appName := m.Module.Application.Name
					externalAppMap[m.Path] = appName
					if strings.HasSuffix(m.Path, ".ts") {
						externalAppMap[strings.TrimSuffix(m.Path, ".ts")] = appName
					} else {
						externalAppMap[m.Path+".ts"] = appName
					}
				}
			}
		}
	}

	getAppForPath := func(path string) (string, error) {
		if backendPluginPathWithinRoot(path, moduleAbsPath) {
			return p.Module.ApplicationStr, nil
		}
		if app, ok := externalAppMap[path]; ok {
			return app, nil
		}
		if moduleName := moduleNameFromPath(path); moduleName != "" {
			if app, ok := externalModuleAppMap[moduleName]; ok {
				return app, nil
			}
		}
		return "", fmt.Errorf("application not found for model path: %s", path)
	}

	for _, result := range parserResults {
		if result.ModelClassNode == nil {
			continue
		}

		state := getContentState(result)
		result.Content = state.content

		// Find @Model decorator
		var modelDecorator *parser.Decorator
		for _, d := range result.ModelClassNode.Decorators {
			moduleSpec, referenceIdent := p.FindModuleSpecAndReferenceIdent(d.ModuleSpecPath, d.ReferenceIdent)
			if sameModuleSpecPath(moduleSpec, modelDecoratorModuleSpec) && referenceIdent == modelDecoratorReferenceIdent {
				modelDecorator = d
				break
			}
		}

		if modelDecorator == nil {
			continue
		}

		// If the decorator already specifies application, skip injection.
		// This avoids warnings for external models not registered in meta yet.
		hasApplication := false
		if len(modelDecorator.Arguments) >= 2 {
			arg := modelDecorator.Arguments[1]
			if arg.Type == "ObjectLiteral" {
				for _, prop := range arg.ObjectProperties {
					if prop == nil {
						continue
					}
					if prop.Name == "application" {
						hasApplication = true
						break
					}
				}
			}
		}
		if hasApplication {
			continue
		}

		appName, err := getAppForPath(result.Path)
		if err != nil {
			return xfmt.Errorf("failed to determine application for model path %s: %w", result.Path, err)
		}

		// Inject application: 'appName'
		// Case 1: @Model('Name') -> @Model('Name', { application: 'appName' })
		// Case 2: @Model('Name', { ... }) -> @Model('Name', { ..., application: 'appName' })

		if len(modelDecorator.Arguments) == 0 {
			continue
		}

		// Modify content by mapping AST raw byte offsets to the current content.
		// This keeps offsets stable even when multiple model injections happen in one file.

		if len(modelDecorator.Arguments) == 1 {
			// Case 1: Insert second argument
			arg := modelDecorator.Arguments[0]
			injection := fmt.Sprintf(", { application: '%s' }", appName)
			if !applyRawInsertion(state, arg.End, injection) {
				continue
			}
		} else if len(modelDecorator.Arguments) >= 2 {
			// Case 2: Inject into existing object literal (Overwrite user provided application)
			arg := modelDecorator.Arguments[1]
			if arg.Type == "ObjectLiteral" {
				replaced := false
				for _, prop := range arg.ObjectProperties {
					if prop == nil || prop.Name != "application" {
						continue
					}

					start := mapRawOffsetToCurrent(state, prop.ValueStart)
					end := mapRawOffsetToCurrent(state, prop.ValueEnd)
					if start < 0 || end <= start || end > len(state.content) {
						continue
					}

					litText := state.content[start:end]
					quote := "'"
					if strings.HasPrefix(litText, "\"") {
						quote = "\""
					}
					newLit := fmt.Sprintf("%s%s%s", quote, appName, quote)
					replaced = applyRawReplacement(state, prop.ValueStart, prop.ValueEnd, newLit)
					if !replaced {
						continue
					}
					break
				}

				if replaced {
					continue
				}

				rawEnd := arg.End
				if rawEnd == 0 {
					continue
				}
				insertPos := mapRawOffsetToCurrent(state, rawEnd) - 1 // position of '}'
				if insertPos < 0 || insertPos > len(state.content) {
					continue
				}

				// Determine if we need a leading comma (if last non-ws before '}' isn't already ',' or '{')
				needsComma := true
				for i := insertPos - 1; i >= 0; i-- {
					ch := state.content[i]
					if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
						continue
					}
					if ch == ',' {
						needsComma = false
					}
					if ch == '{' {
						needsComma = false
					}
					break
				}

				// Compute indentation based on the line of the closing brace
				indent := ""
				for i := insertPos - 1; i >= 0; i-- {
					if state.content[i] == '\n' {
						indent = state.content[i+1 : insertPos]
						break
					}
				}
				// Trim indent to spaces/tabs only
				trimmed := ""
				for j := 0; j < len(indent); j++ {
					if indent[j] == ' ' || indent[j] == '\t' {
						trimmed += string(indent[j])
					}
				}
				indent = trimmed

				prefix := "\n" + indent
				if needsComma {
					prefix = "," + prefix
				}

				injection := fmt.Sprintf("%sapplication: '%s'", prefix, appName)
				if !applyRawInsertion(state, rawEnd-1, injection) {
					continue
				}
			}
		}

		result.Content = state.content

		continue
	}

	for _, result := range parserResults {
		if state, ok := contentStateByPath[result.Path]; ok {
			result.Content = state.content
		}
	}
	return nil
}

func (p *BackendPlugin) setFieldMeta(parserResults []*parser.ParserResult) error {
	// Use the new Field decorator only.
	fieldDecoratorModuleSpec, fieldDecoratorReferenceIdent := meta.FieldDecoratorModuleSpec(p.Env)
	modelDecoratorModuleSpec, modelDecoratorReferenceIdent := meta.ModelDecoratorModuleSpec(p.Env)
	moduleAbsPath := p.Module.Path
	sameModuleSpecPath := newBackendPluginModuleSpecPathComparer()

	findModelName := func(modelPath string) string {
		moduleName := ""
		for _, parserResult := range parserResults {
			model := parserResult.Model
			if model == nil {
				continue
			}
			if model.Path == modelPath || model.Path == modelPath+".ts" {
				moduleName = fmt.Sprintf("%s.%s", p.Module.ApplicationStr, model.Name)
			}
		}
		if moduleName == "" {
			var model *meta.IrModel
			if result := p.Env.Session().Preload("Module.Application").Where("path = ? OR path = ?", modelPath, modelPath+".ts").Take(&model); result.Error != nil {
				return ""
			}
			if model != nil && model.Module != nil && model.Module.Application != nil {
				moduleName = fmt.Sprintf("%s.%s", model.Module.Application.Name, model.Name)
			}
		}
		return moduleName
	}
	// Resolve the target model by name from the parsed results.
	findModelByName := func(name string) *meta.IrModel {
		for _, pr := range parserResults {
			if pr.Model != nil && pr.Model.Name == name {
				return pr.Model
			}
		}
		return nil
	}
	// Read the target model's parentField option when it is present.
	getParentFieldFromModel := func(m *meta.IrModel) string {
		if m == nil {
			return ""
		}
		for _, d := range m.Decorators {
			if d.Name != "Model" || !sameModuleSpecPath(d.ModuleSpecPath, modelDecoratorModuleSpec) || d.ReferenceIdent != modelDecoratorReferenceIdent {
				continue
			}
			for _, arg := range d.Arguments {
				if arg.Type != "ObjectLiteral" || arg.Value == "" {
					continue
				}
				var opts map[string]interface{}
				if err := json.Unmarshal([]byte(arg.Value), &opts); err != nil {
					continue
				}
				if pf, ok := opts["parentField"].(string); ok && pf != "" {
					return pf
				}
			}
		}
		return ""
	}

	asInt := func(v interface{}) int {
		if f, ok := v.(float64); ok {
			return int(f)
		}
		return 0
	}
	asBool := func(v interface{}) bool {
		if b, ok := v.(bool); ok {
			return b
		}
		return false
	}
	// Normalize round to a string value and accept both numeric and string inputs.
	toRoundStr := func(v interface{}) *string {
		if v == nil {
			return nil
		}
		// Numbers arrive from JSON decoding as float64.
		if f, ok := v.(float64); ok {
			num := int(f)
			num2name := map[int]string{
				0: "ROUND_UP",
				1: "ROUND_DOWN",
				2: "ROUND_CEIL",
				3: "ROUND_FLOOR",
				4: "ROUND_HALF_UP",
				5: "ROUND_HALF_DOWN",
				6: "ROUND_HALF_EVEN",
				7: "ROUND_HALF_CEIL",
				8: "ROUND_HALF_FLOOR",
			}
			if name, ok := num2name[num]; ok {
				return &name
			}
			return nil
		}
		// Strings accept variants like "HALF_UP", "ROUND_HALF_UP", or "half_up".
		if s, ok := v.(string); ok {
			key := strings.ToUpper(strings.TrimSpace(s))
			if key == "" {
				return nil
			}
			if !strings.HasPrefix(key, "ROUND_") {
				key = "ROUND_" + key
			}
			// Validate the normalized value against the allowed constants.
			switch key {
			case "ROUND_UP", "ROUND_DOWN", "ROUND_CEIL", "ROUND_FLOOR",
				"ROUND_HALF_UP", "ROUND_HALF_DOWN", "ROUND_HALF_EVEN",
				"ROUND_HALF_CEIL", "ROUND_HALF_FLOOR":
				return &key
			}
		}
		return nil
	}

	isRelationType := func(t string) bool {
		return t == "ManyToOne" || t == "OneToMany" || t == "ManyToMany"
	}

	needsStructure := func(t string) (needSize bool) {
		switch t {
		case "char", "varchar":
			return true
		default:
			// decimal/monetary: DDL uses NUMERIC(38,18); business precision/scale are optional.
			return false
		}
	}

	for _, parserResult := range parserResults {
		if !backendPluginPathWithinRoot(parserResult.Path, moduleAbsPath) {
			continue
		}
		model := parserResult.Model
		if model == nil {
			continue
		}

		for _, field := range model.Fields {
			for _, decorator := range field.Decorators {
				if decorator.Name != "Field" || !sameModuleSpecPath(decorator.ModuleSpecPath, fieldDecoratorModuleSpec) || decorator.ReferenceIdent != fieldDecoratorReferenceIdent {
					continue
				}
				if len(decorator.Arguments) == 0 {
					continue
				}

				var options map[string]interface{}
				if err := json.Unmarshal([]byte(decorator.Arguments[0].Value), &options); err != nil {
					return fmt.Errorf("parse Field options failed: %v", err)
				}

				// Type
				ftype, _ := options["type"].(string)
				field.FieldType = ftype

				// Selection must stay on the TsParser-resolved field.Selection
				// (msgid labels + optional labelText). Do not overwrite from the
				// raw decorator ObjectLiteral — call expressions like _lt(...)
				// are stored as source text there and break web store codegen.

				// Mark fields with select expressions as read-only.
				if _, ok := options["select"]; ok {
					field.IsReadonly = true
				}

				// Column metadata.
				if col, ok := options["column"].(map[string]interface{}); ok {
					// Computed columns are read-only.
					if _, ok := col["compute"]; ok {
						field.IsReadonly = true
					}
					field.NotNull = asBool(col["notNull"])
					if size, ok := col["size"]; ok {
						field.Size = asInt(size)
					}
					if prec, ok := col["precision"]; ok {
						field.Precision = asInt(prec)
					}
					if scale, ok := col["scale"]; ok {
						field.Scale = asInt(scale)
					}
					// Pass through scaleField.
					if v, ok := col["scaleField"]; ok {
						if s, ok2 := v.(string); ok2 && s != "" {
							field.ScaleField = s
						}
					}
					// Pass through currencyField (monetary).
					if v, ok := col["currencyField"]; ok {
						if s, ok2 := v.(string); ok2 && s != "" {
							field.CurrencyField = s
						}
					}
					// Store round as a normalized string.
					if r := toRoundStr(col["round"]); r != nil {
						field.Round = r
					}
					if asBool(col["primaryKey"]) {
						field.Indexed = true
					}
					// Index and uniqueness flags.
					for _, k := range []string{"index", "uniqueIndex", "unique"} {
						if v, ok := col[k]; ok {
							switch vv := v.(type) {
							case bool:
								if vv {
									field.Indexed = true
								}
							case string:
								if vv != "" {
									field.Indexed = true
								}
							}
						}
					}
				}

				// Use select to fill structural metadata only when column is absent.
				if sel, ok := options["select"].(map[string]interface{}); ok {
					if _, hasCol := options["column"]; !hasCol {
						if size, ok := sel["size"]; ok && field.Size == 0 {
							field.Size = asInt(size)
						}
						if prec, ok := sel["precision"]; ok && field.Precision == 0 {
							field.Precision = asInt(prec)
						}
						if scale, ok := sel["scale"]; ok && field.Scale == 0 {
							field.Scale = asInt(scale)
						}
						// Allow scaleField from select when column is absent.
						if v, ok := sel["scaleField"]; ok && field.ScaleField == "" {
							if s, ok2 := v.(string); ok2 && s != "" {
								field.ScaleField = s
							}
						}
						if v, ok := sel["currencyField"]; ok && field.CurrencyField == "" {
							if s, ok2 := v.(string); ok2 && s != "" {
								field.CurrencyField = s
							}
						}
						// If column did not define round, allow it to come from select.
						if field.Round == nil {
							if r := toRoundStr(sel["round"]); r != nil {
								field.Round = r
							}
						}
					}
				}

				// Flat @Field options (PR-1): top-level scaleField / currencyField.
				if v, ok := options["scaleField"].(string); ok && strings.TrimSpace(v) != "" && field.ScaleField == "" {
					field.ScaleField = strings.TrimSpace(v)
				}
				if v, ok := options["currencyField"].(string); ok && strings.TrimSpace(v) != "" && field.CurrencyField == "" {
					field.CurrencyField = strings.TrimSpace(v)
				}
				if size, ok := options["size"]; ok && field.Size == 0 {
					field.Size = asInt(size)
				}
				if prec, ok := options["precision"]; ok && field.Precision == 0 {
					field.Precision = asInt(prec)
				}
				if scale, ok := options["scale"]; ok && field.Scale == 0 {
					field.Scale = asInt(scale)
				}
				if field.Round == nil {
					if r := toRoundStr(options["round"]); r != nil {
						field.Round = r
					}
				}
				if asBool(options["notNull"]) || asBool(options["required"]) {
					field.NotNull = true
				}

				// Validate required structure only for scalar fields that create physical columns.
				if ftype != "" && !isRelationType(ftype) {
					if (model.AutoMigrate != nil && !*model.AutoMigrate) || model.Readonly {
						continue
					}
					needSize := needsStructure(ftype)
					_, hasSelect := options["select"]
					if !hasSelect { // Treat select as a virtual column that does not require physical structure.
						if needSize && field.Size == 0 {
							return fmt.Errorf("model %s field %s(type=%s) missing required size (provide column.size or select.size)", model.Name, field.Name, ftype)
						}
						// Decimal/monetary no longer require precision/scale: DDL uses NUMERIC(38,18).
					}

				}

				// Relation fields.
				switch ftype {
				case "ManyToOneRef":
					field.Relation = "ManyToOne"
					if tm, ok := options["targetModel"].(string); ok {
						field.RelationModel = tm
					}
				case "ManyToManyRef":
					field.Relation = "ManyToMany"
					if tm, ok := options["targetModel"].(string); ok {
						field.RelationModel = tm
					}
				case "ManyToOne":
					field.Relation = "ManyToOne"
					field.RelationModel = findModelName(field.ModuleSpecPath)
					// Propagate the target model's parentField when it is present.
					if field.RelationModel != "" {
						if mm := findModelByName(field.RelationModel); mm != nil {
							if pf := getParentFieldFromModel(mm); pf != "" {
								field.RelationModelParentField = pf
							}
						}
					}
				case "OneToMany":
					field.Relation = "OneToMany"
					field.RelationModel = findModelName(field.ModuleSpecPath)
					if rel, ok := options["relation"].(map[string]interface{}); ok {
						if v, ok := rel["inverseField"]; v != nil && ok {
							field.RelationInverseField = fmt.Sprintf("%v", v)
						}
					}
					// Collection relations also propagate parentField for tree-style selectors in the UI.
					if field.RelationModel != "" {
						if mm := findModelByName(field.RelationModel); mm != nil {
							if pf := getParentFieldFromModel(mm); pf != "" {
								field.RelationModelParentField = pf
							}
						}
					}
				case "ManyToMany":
					field.Relation = "ManyToMany"
					field.RelationModel = findModelName(field.ModuleSpecPath)
					if rel, ok := options["relation"].(map[string]interface{}); ok {
						if v, ok := rel["joinField"]; v != nil && ok {
							field.RelationJoinField = fmt.Sprintf("%v", v)
						}
						if v, ok := rel["inverseJoinField"]; v != nil && ok {
							field.RelationInverseJoinField = fmt.Sprintf("%v", v)
						}
						if v, ok := rel["joinModel"]; v != nil && ok {
							field.RelationJoinModel = fmt.Sprintf("%v", v)
						}
					}
					// Propagate the target model's parentField here as well.
					if field.RelationModel != "" {
						if mm := findModelByName(field.RelationModel); mm != nil {
							if pf := getParentFieldFromModel(mm); pf != "" {
								field.RelationModelParentField = pf
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// DefinePlugins returns the backend esbuild plugins for the current runtime state.
func (p *BackendPlugin) DefinePlugins(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.IrModule, options ...esbplugins.EsbPluginOptions) []api.Plugin {
	for _, opt := range options {
		if opt != nil {
			opt(p)
		}
	}
	p.bindRuntimeState(runtimeScope, module)

	return []api.Plugin{{
		Name: "choysum-backend-inherit",
		Setup: func(build api.PluginBuild) {
			// modulesPath := "<modules-path>"
			// build.OnResolve(api.OnResolveOptions{Filter: `.*`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			// 	p.Mu.Lock()
			// 	defer p.Mu.Unlock()

			// 	pathAlias, err := parser.ParseTsconfigPathAlias(build.InitialOptions)
			// 	if err != nil {
			// 		return api.OnResolveResult{}, err
			// 	}
			// 	resolvePath := parser.ApplyPathAlias(pathAlias, args.Path)

			// 	if !filepath.IsAbs(resolvePath) {
			// 		resolvePath = filepath.Join(args.ResolveDir, resolvePath)
			// 	}

			// 	if strings.HasPrefix(resolvePath, modulesPath) {
			// 		fullPath := ""
			// 		possiablePaths := []string{
			// 			resolvePath,
			// 			resolvePath + ".ts",
			// 			filepath.Join(resolvePath, "index.ts"),
			// 			resolvePath + ".tsx",
			// 			filepath.Join(resolvePath, "index.tsx"),
			// 			resolvePath + ".d.ts",
			// 			filepath.Join(resolvePath, "index.d.ts"),
			// 		}
			// 		for _, possiablePath := range possiablePaths {
			// 			if info, err := os.Stat(possiablePath); err == nil && !info.IsDir() {
			// 				fullPath = possiablePath
			// 				break
			// 			}
			// 		}

			// 		if fullPath != "" {
			// 			finalChildPath := parser.FindModelFinalChild(p.ParserResults, fullPath, args.Importer)
			// 			if finalChildPath != "" {
			// 				fullPath = finalChildPath
			// 			}
			// 			if args.Importer == "/Users/wangbuke/choysum/modules/sale2/service/models/order.ts" {
			// 				fmt.Printf("fullPath: %s args: %+s\n", fullPath, args.Importer)
			// 			}
			// 			return api.OnResolveResult{Path: fullPath}, nil
			// 		}
			// 	}

			// 	return api.OnResolveResult{}, nil
			// })
			build.OnLoad(api.OnLoadOptions{Filter: `\.ts$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				p.Mu.Lock()
				defer p.Mu.Unlock()

				var content string
				var err error
				// Check if the file is already parsed
				parserResult := p.FindParserResultByPath(args.Path)
				if parserResult != nil {
					// If the file is already parsed, use the parsed content
					if parserResult.Content != "" {
						content = parserResult.Content
					} else {
						content = parserResult.RawContent
					}
				} else {
					content, err = p.ReadNormalizedTextFile(args.Path)
					if err != nil {
						return api.OnLoadResult{}, err
					}
				}

				if p.isEntryPointPath(args.Path) {
					content = p.appendEntryPointImports(content)
				}

				pathAlias, err := parser.ParseTsconfigPathAlias(build.InitialOptions)
				if err != nil {
					return api.OnLoadResult{}, err
				}
				parserResult, err = p.Parser.Parse(pathAlias, args.Path, content)
				if err != nil {
					p.Env.Logger().Error("typescript file parsing failed", "error", err)
					return api.OnLoadResult{}, err
				}

				p.PublishParserResult(parserResult)

				return api.OnLoadResult{
					Contents: &content,
					Loader:   api.LoaderTS,
				}, nil
			})
		},
	}}
}

// GetParserResults finalizes parser results after metadata normalization.
func (p *BackendPlugin) GetParserResults() ([]*parser.ParserResult, error) {
	results := p.HandleParserResults()
	if err := p.replaceModuleSpecReferenceIdent(results); err != nil {
		return nil, err
	}

	if err := p.injectModelApplication(results); err != nil {
		return nil, err
	}

	if err := p.setFieldMeta(results); err != nil {
		return nil, err
	}
	return results, nil
}

// NewBackendPlugin creates a backend esbuild plugin for a module entry point.
func NewBackendPlugin(runtimeScope scope.Scope, module *meta.IrModule, entryPoint string, opts ...func(*BackendPlugin)) esbplugins.EsbPlugin {
	p := &BackendPlugin{
		BasePlugin:    esbplugins.NewBasePlugin(runtimeScope, module, entryPoint),
		parserFactory: backendtsparser.NewTsParser,
	}

	for _, opt := range opts {
		opt(p)
	}
	p.bindRuntimeState(runtimeScope, module)

	return p
}

// WithParser overrides the parser used by BackendPlugin.
func WithParser(parser parser.Parser) func(*BackendPlugin) {
	return func(p *BackendPlugin) {
		p.parserFactory = nil
		p.Parser = parser
	}
}
