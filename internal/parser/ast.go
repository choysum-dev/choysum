// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
	tscore "github.com/buke/typescript-go-internal/pkg/core"
	tsparser "github.com/buke/typescript-go-internal/pkg/parser"
	tspath "github.com/buke/typescript-go-internal/pkg/tspath"
)

type AstTree = tsast.SourceFile

func ParseTSTree(ctx context.Context, path string, content string) (*AstTree, error) {
	_ = ctx
	normalized := tspath.NormalizePath(path)
	if !filepath.IsAbs(normalized) {
		absPath, err := filepath.Abs(normalized)
		if err != nil {
			return nil, err
		}
		normalized = tspath.NormalizePath(absPath)
	}

	scriptKind := tscore.GetScriptKindFromFileName(normalized)
	source := tsparser.ParseSourceFile(tsast.SourceFileParseOptions{
		FileName: normalized,
		Path:     tspath.ToPath(normalized, "", true),
	}, content, scriptKind)

	return source, nil
}

type PropertyNode struct {
	ReferenceIdent string
	ModuleSpecPath string
	Text           string
	Start          int
	End            int
	Line           int
	Column         int
	Name           string
	ValueKind      string
	ValueText      string
	ValueStart     int
	ValueEnd       int
}

type Import struct {
	ReferenceIdent  string
	ModuleSpecPath  string
	Text            string
	Start           int
	End             int
	Line            int
	Column          int
	ModuleSpecText  string
	ModuleSpecStart int
	ModuleSpecEnd   int
}

type Export struct {
	ReferenceIdent string
	ModuleSpecPath string
	Text           string
	Start          int
	End            int
	Line           int
	Column         int

	Wildcard []*Export // only for Wildcard export like export * from "./test1"; export * from "./test2";
}

type Class struct {
	Text          string
	Start         int
	End           int
	Line          int
	Column        int
	Name          string
	Abstract      bool
	Decorators    []*Decorator
	Extends       *Extends
	MemberVars    []*MemberVar
	MemberMethods []*MemberMethod
}

type Extends struct {
	Text           string
	Start          int
	End            int
	Line           int
	Column         int
	Name           string
	ReferenceIdent string
	ModuleSpecPath string
}

type Decorator struct {
	Text           string
	Start          int
	End            int
	Line           int
	Column         int
	Name           string
	ReferenceIdent string
	ModuleSpecPath string
	Arguments      []*Argument
}

type ObjectProperty struct {
	Name       string
	ValueKind  string
	ValueText  string
	Start      int
	End        int
	Line       int
	Column     int
	ValueStart int
	ValueEnd   int
}

type Argument struct {
	Text             string
	Start            int
	End              int
	Line             int
	Column           int
	Type             string
	Value            string
	ReferenceIdent   string
	ModuleSpecPath   string
	ObjectProperties []*ObjectProperty
}

type MemberVar struct {
	Text           string
	Start          int
	End            int
	Line           int
	Column         int
	Name           string
	TypeAnnotation string

	TsTypeReference string
	ReferenceIdent  string
	ModuleSpecPath  string

	AccessibilityModifier string
	IsStatic              bool
	IsReadonly            bool
	Decorators            []*Decorator
}

type MemberMethod struct {
	Text                  string
	Start                 int
	End                   int
	Line                  int
	Column                int
	Name                  string
	TypeAnnotation        string
	AccessibilityModifier string
	IsStatic              bool
	Decorators            []*Decorator
	Parameters            []*Parameter
	TypeParameters        []*TypeParameter
}

type Parameter struct {
	Text           string
	Start          int
	End            int
	Line           int
	Column         int
	Name           string
	TypeAnnotation string
}

type TypeParameter struct {
	Text           string
	Start          int
	End            int
	Line           int
	Column         int
	Name           string
	ReferenceIdent string
	ModuleSpecPath string
}

type TsParser struct {
	Path       string
	Content    string
	Context    context.Context
	PathAlias  map[string]string
	ImportsMap map[string]*Import
	ExportsMap map[string]*Export
}

type tsgoImportExportCtx struct {
	path      string
	pathAlias map[string]string
	source    *tsast.SourceFile
	imports   map[string]*Import
	exports   map[string]*Export
	lineMap   []tscore.TextPos
}

func (p *TsParser) parseTSGoImportExportCtx() (*tsgoImportExportCtx, error) {
	if strings.TrimSpace(p.Path) == "" || strings.TrimSpace(p.Content) == "" {
		return nil, fmt.Errorf("path or content is empty")
	}

	normalized := tspath.NormalizePath(p.Path)
	if !filepath.IsAbs(normalized) {
		absPath, err := filepath.Abs(normalized)
		if err != nil {
			return nil, err
		}
		normalized = tspath.NormalizePath(absPath)
	}

	scriptKind := tscore.GetScriptKindFromFileName(normalized)
	source := tsparser.ParseSourceFile(tsast.SourceFileParseOptions{
		FileName: normalized,
		Path:     tspath.ToPath(normalized, "", true),
	}, p.Content, scriptKind)

	ctx := &tsgoImportExportCtx{
		path:      normalized,
		pathAlias: p.PathAlias,
		source:    source,
		imports:   make(map[string]*Import),
		exports:   make(map[string]*Export),
		lineMap:   source.ECMALineMap(),
	}

	for _, stmt := range source.Statements.Nodes {
		if stmt == nil {
			continue
		}
		if stmt.Kind == tsast.KindImportDeclaration || stmt.Kind == tsast.KindJSImportDeclaration {
			ctx.parseImport(stmt)
		}
	}

	for _, stmt := range source.Statements.Nodes {
		if stmt == nil {
			continue
		}
		ctx.parseExport(stmt)
	}

	return ctx, nil
}

func (c *tsgoImportExportCtx) currentModuleSpecPath() string {
	if filepath.Ext(c.path) == ".ts" {
		return strings.TrimSuffix(c.path, ".ts")
	}
	return c.path
}

func (c *tsgoImportExportCtx) resolveModuleSpec(moduleSpecifier string) string {
	moduleSpecifier = strings.Trim(moduleSpecifier, `"'`)
	if strings.HasPrefix(moduleSpecifier, "./") || strings.HasPrefix(moduleSpecifier, "../") {
		moduleSpecifier = filepath.Join(filepath.Dir(c.path), moduleSpecifier)
	}
	moduleSpecifier = ApplyPathAlias(c.pathAlias, moduleSpecifier)
	moduleSpecifier = strings.TrimSuffix(moduleSpecifier, ".ts")
	return moduleSpecifier
}

func (c *tsgoImportExportCtx) nodeText(node *tsast.Node) string {
	if node == nil {
		return ""
	}
	start, end := node.Pos(), node.End()
	if start < 0 || end < start || end > len(c.source.Text()) {
		return ""
	}
	return c.source.Text()[start:end]
}

func (c *tsgoImportExportCtx) lineColumn(pos int) (int, int) {
	line, col := tscore.PositionToLineAndByteOffset(pos, c.lineMap)
	return line + 1, col + 1
}

func (c *tsgoImportExportCtx) parseImport(stmt *tsast.Node) {
	decl := stmt.AsImportDeclaration()
	moduleSpecifier := strings.Trim(decl.ModuleSpecifier.Text(), `"'`)
	if moduleSpecifier == "" {
		return
	}

	moduleSpecPath := c.resolveModuleSpec(moduleSpecifier)
	line, col := c.lineColumn(stmt.Pos())
	moduleSpecText := strings.TrimSpace(c.nodeText(decl.ModuleSpecifier))
	importText := strings.TrimSpace(c.nodeText(stmt))

	makeImport := func(referenceIdent string) *Import {
		return &Import{
			ReferenceIdent:  referenceIdent,
			ModuleSpecPath:  moduleSpecPath,
			Text:            importText,
			Start:           stmt.Pos(),
			End:             stmt.End(),
			Line:            line,
			Column:          col,
			ModuleSpecText:  moduleSpecText,
			ModuleSpecStart: decl.ModuleSpecifier.Pos(),
			ModuleSpecEnd:   decl.ModuleSpecifier.End(),
		}
	}

	if decl.ImportClause == nil {
		return
	}

	importClause := decl.ImportClause.AsImportClause()
	if defaultName := importClause.Name(); defaultName != nil {
		localName := defaultName.Text()
		if localName != "" {
			c.imports[localName] = makeImport("default")
		}
	}

	namedBindings := importClause.NamedBindings
	if namedBindings == nil {
		return
	}

	if namedBindings.Kind == tsast.KindNamespaceImport {
		alias := namedBindings.AsNamespaceImport().Name().Text()
		if alias != "" {
			c.imports[alias] = makeImport("*")
		}
		return
	}

	if namedBindings.Kind == tsast.KindNamedImports {
		for _, node := range namedBindings.AsNamedImports().Elements.Nodes {
			if node == nil || node.Kind != tsast.KindImportSpecifier {
				continue
			}
			spec := node.AsImportSpecifier()
			localName := spec.Name().Text()
			if localName == "" {
				continue
			}
			referenceIdent := localName
			if spec.PropertyName != nil {
				referenceIdent = spec.PropertyName.Text()
			}
			c.imports[localName] = makeImport(referenceIdent)
		}
	}
}

func (c *tsgoImportExportCtx) convertReferenceWithModuleSpec(referenceIdent string) (string, string) {
	if tsImport, ok := c.imports[referenceIdent]; ok {
		if tsImport.ReferenceIdent == "*" {
			return tsImport.ModuleSpecPath, ""
		}
		return tsImport.ModuleSpecPath, tsImport.ReferenceIdent
	}
	if defaultExport, ok := c.exports["default"]; ok {
		if defaultExport.ReferenceIdent == referenceIdent {
			return defaultExport.ModuleSpecPath, "default"
		}
	}
	return c.currentModuleSpecPath(), referenceIdent
}

func (c *tsgoImportExportCtx) parseExport(stmt *tsast.Node) {
	line, col := c.lineColumn(stmt.Pos())
	exportText := c.nodeText(stmt)
	newExport := func(referenceIdent string, moduleSpecPath string) *Export {
		return &Export{
			ReferenceIdent: referenceIdent,
			ModuleSpecPath: moduleSpecPath,
			Text:           exportText,
			Start:          stmt.Pos(),
			End:            stmt.End(),
			Line:           line,
			Column:         col,
		}
	}

	if stmt.Kind == tsast.KindExportAssignment {
		expr := stmt.AsExportAssignment().Expression
		if expr != nil {
			switch expr.Kind {
			case tsast.KindIdentifier:
				moduleSpec, ref := c.convertReferenceWithModuleSpec(expr.Text())
				c.exports["default"] = newExport(ref, moduleSpec)
				return
			case tsast.KindPropertyAccessExpression:
				pa := expr.AsPropertyAccessExpression()
				moduleSpec, _ := c.convertReferenceWithModuleSpec(pa.Expression.Text())
				c.exports["default"] = newExport(pa.Name().Text(), moduleSpec)
				return
			}
		}
		c.exports["default"] = newExport("default", c.currentModuleSpecPath())
		return
	}

	if stmt.Kind == tsast.KindExportDeclaration {
		decl := stmt.AsExportDeclaration()
		moduleSpecPath := ""
		if decl.ModuleSpecifier != nil {
			moduleSpecPath = c.resolveModuleSpec(decl.ModuleSpecifier.Text())
		}

		if decl.ExportClause != nil {
			if decl.ExportClause.Kind == tsast.KindNamedExports {
				for _, node := range decl.ExportClause.AsNamedExports().Elements.Nodes {
					if node == nil || node.Kind != tsast.KindExportSpecifier {
						continue
					}
					spec := node.AsExportSpecifier()
					name := spec.Name().Text()
					if name == "" {
						continue
					}
					referenceIdent := name
					if spec.PropertyName != nil {
						referenceIdent = spec.PropertyName.Text()
					}
					if moduleSpecPath == "" {
						resolvedModuleSpec, resolvedRef := c.convertReferenceWithModuleSpec(referenceIdent)
						c.exports[name] = newExport(resolvedRef, resolvedModuleSpec)
					} else {
						c.exports[name] = newExport(referenceIdent, moduleSpecPath)
					}
				}
				return
			}

			if decl.ExportClause.Kind == tsast.KindNamespaceExport {
				name := decl.ExportClause.AsNamespaceExport().Name().Text()
				if name != "" {
					c.exports[name] = newExport(name, moduleSpecPath)
				}
				return
			}
		}

		if moduleSpecPath != "" {
			wildcard := newExport("*", moduleSpecPath)
			if existing, ok := c.exports["*"]; ok {
				existing.Wildcard = append(existing.Wildcard, wildcard)
			} else {
				c.exports["*"] = &Export{Wildcard: []*Export{wildcard}}
			}
		}
		return
	}

	if tsgoIsExportDefaultDeclaration(stmt) {
		referenceIdent := "default"
		if name := tsgoExportDeclarationName(stmt); name != "" {
			referenceIdent = name
		}
		c.exports["default"] = newExport(referenceIdent, c.currentModuleSpecPath())
		return
	}

	if tsgoIsExportDeclaration(stmt) {
		if name := tsgoExportDeclarationName(stmt); name != "" {
			c.exports[name] = newExport(name, c.currentModuleSpecPath())
		}
	}
}

func tsgoExportDeclarationName(stmt *tsast.Node) string {
	if stmt == nil {
		return ""
	}
	name := stmt.Name()
	if name != nil {
		return name.Text()
	}
	if stmt.Kind == tsast.KindVariableStatement {
		decls := stmt.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes
		if len(decls) > 0 {
			decl := decls[0]
			if decl != nil {
				return decl.AsVariableDeclaration().Name().Text()
			}
		}
	}
	return ""
}

func tsgoHasModifier(node *tsast.Node, kind tsast.Kind) bool {
	if node == nil {
		return false
	}
	mods := node.Modifiers()
	if mods == nil {
		return false
	}
	for _, m := range mods.Nodes {
		if m != nil && m.Kind == kind {
			return true
		}
	}
	return false
}

func tsgoIsExportDefaultDeclaration(node *tsast.Node) bool {
	return tsgoHasModifier(node, tsast.KindExportKeyword) && tsgoHasModifier(node, tsast.KindDefaultKeyword)
}

func tsgoIsExportDeclaration(node *tsast.Node) bool {
	return tsgoHasModifier(node, tsast.KindExportKeyword)
}

func (p *TsParser) ParseImport(_ any) error {
	if p.ImportsMap == nil {
		p.ImportsMap = make(map[string]*Import)
	}
	ctx, err := p.parseTSGoImportExportCtx()
	if err != nil {
		return err
	}
	p.ImportsMap = ctx.imports
	return nil
}

func (p *TsParser) ConvertReferenceWithModuleSpec(referenceIdent string) (string, string) {
	if tsImport, ok := p.ImportsMap[referenceIdent]; ok {
		if tsImport.ReferenceIdent == "*" {
			return tsImport.ModuleSpecPath, ""
		} else {
			return tsImport.ModuleSpecPath, tsImport.ReferenceIdent
		}
	} else {
		if defaultExport, ok := p.ExportsMap["default"]; ok {
			if defaultExport.ReferenceIdent == referenceIdent {
				return defaultExport.ModuleSpecPath, "default"
			}
		}
		return strings.TrimSuffix(p.Path, ".ts"), referenceIdent
	}
}

func (p *TsParser) ParseExport(_ any) error {
	if p.ExportsMap == nil {
		p.ExportsMap = make(map[string]*Export)
	}
	ctx, err := p.parseTSGoImportExportCtx()
	if err != nil {
		return err
	}
	p.ExportsMap = ctx.exports
	return nil
}

func tsgoNodeStableInfo(ctx *tsgoImportExportCtx, node *tsast.Node) (text string, start, end, line, column int) {
	if ctx == nil || node == nil {
		return "", 0, 0, 0, 0
	}
	line, column = ctx.lineColumn(node.Pos())
	return ctx.nodeText(node), node.Pos(), node.End(), line, column
}

func tsgoNodeTypeName(node *tsast.Node) string {
	if node == nil {
		return ""
	}
	switch node.Kind {
	case tsast.KindObjectLiteralExpression:
		return "ObjectLiteral"
	case tsast.KindArrayLiteralExpression:
		return "ArrayLiteral"
	case tsast.KindIdentifier:
		return "IdentExpr"
	case tsast.KindStringLiteral, tsast.KindNumericLiteral, tsast.KindBigIntLiteral, tsast.KindTrueKeyword, tsast.KindFalseKeyword, tsast.KindNullKeyword:
		return "Literal"
	default:
		return node.Kind.String()
	}
}

func tsgoIsLiteralNode(node *tsast.Node) bool {
	if node == nil {
		return false
	}
	switch node.Kind {
	case tsast.KindStringLiteral, tsast.KindNumericLiteral, tsast.KindBigIntLiteral, tsast.KindTrueKeyword, tsast.KindFalseKeyword, tsast.KindNullKeyword:
		return true
	default:
		return false
	}
}

func tsgoPropertyName(ctx *tsgoImportExportCtx, nameNode *tsast.Node) string {
	if ctx == nil || nameNode == nil {
		return ""
	}
	name := strings.TrimSpace(ctx.nodeText(nameNode))
	if name == "" {
		name = strings.TrimSpace(nameNode.Text())
	}
	if len(name) >= 2 {
		if (strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"")) || (strings.HasPrefix(name, "'") && strings.HasSuffix(name, "'")) {
			name = strings.Trim(name, "\"'")
		}
	}
	return strings.TrimSpace(name)
}

func tsgoCollectReferenceIdents(expr *tsast.Node) []string {
	if expr == nil {
		return nil
	}
	switch expr.Kind {
	case tsast.KindIdentifier:
		if text := strings.TrimSpace(expr.Text()); text != "" {
			return []string{text}
		}
		return nil
	case tsast.KindPropertyAccessExpression:
		pa := expr.AsPropertyAccessExpression()
		out := tsgoCollectReferenceIdents(pa.Expression)
		if pa.Name() != nil {
			out = append(out, strings.TrimSpace(pa.Name().Text()))
		}
		return out
	default:
		return nil
	}
}

func (p *TsParser) tsgoConvertLiteralToInterface(ctx *tsgoImportExportCtx, literal *tsast.Node) (interface{}, error) {
	vText := strings.TrimSpace(ctx.nodeText(literal))
	vText = strings.Trim(vText, `"'`)
	var value interface{}
	err := json.Unmarshal([]byte(vText), &value)
	if err != nil {
		value = vText
	}
	return value, nil
}

func (p *TsParser) tsgoConvertArrayLiteralToArray(ctx *tsgoImportExportCtx, arrayLiteral *tsast.Node) ([]interface{}, error) {
	arrayLiteralArray := make([]interface{}, 0)
	if arrayLiteral == nil || arrayLiteral.Kind != tsast.KindArrayLiteralExpression {
		return arrayLiteralArray, nil
	}

	for _, element := range arrayLiteral.AsArrayLiteralExpression().Elements.Nodes {
		if element == nil {
			continue
		}
		if element.Kind == tsast.KindObjectLiteralExpression {
			value, err := p.tsgoConvertObjectLiteralToMap(ctx, element)
			if err != nil {
				return nil, err
			}
			arrayLiteralArray = append(arrayLiteralArray, value)
		} else if tsgoIsLiteralNode(element) {
			value, err := p.tsgoConvertLiteralToInterface(ctx, element)
			if err != nil {
				return nil, err
			}
			arrayLiteralArray = append(arrayLiteralArray, value)
		} else if element.Kind == tsast.KindArrayLiteralExpression {
			value, err := p.tsgoConvertArrayLiteralToArray(ctx, element)
			if err != nil {
				return nil, err
			}
			arrayLiteralArray = append(arrayLiteralArray, value)
		} else {
			arrayLiteralArray = append(arrayLiteralArray, ctx.nodeText(element))
		}
	}

	return arrayLiteralArray, nil
}

func (p *TsParser) tsgoConvertObjectLiteralToMap(ctx *tsgoImportExportCtx, objectLiteral *tsast.Node) (map[string]interface{}, error) {
	objectLiteralMap := make(map[string]interface{})
	if objectLiteral == nil || objectLiteral.Kind != tsast.KindObjectLiteralExpression {
		return objectLiteralMap, nil
	}

	for _, prop := range objectLiteral.AsObjectLiteralExpression().Properties.Nodes {
		if prop == nil {
			continue
		}
		switch prop.Kind {
		case tsast.KindPropertyAssignment:
			assignment := prop.AsPropertyAssignment()
			key := tsgoPropertyName(ctx, assignment.Name())
			init := assignment.Initializer
			if key == "" || init == nil {
				continue
			}

			if init.Kind == tsast.KindObjectLiteralExpression {
				value, err := p.tsgoConvertObjectLiteralToMap(ctx, init)
				if err != nil {
					return nil, err
				}
				objectLiteralMap[key] = value
			} else if tsgoIsLiteralNode(init) {
				value, err := p.tsgoConvertLiteralToInterface(ctx, init)
				if err != nil {
					return nil, err
				}
				objectLiteralMap[key] = value
			} else if init.Kind == tsast.KindArrayLiteralExpression {
				value, err := p.tsgoConvertArrayLiteralToArray(ctx, init)
				if err != nil {
					return nil, err
				}
				objectLiteralMap[key] = value
			} else {
				objectLiteralMap[key] = ctx.nodeText(init)
			}
		case tsast.KindShorthandPropertyAssignment:
			name := tsgoPropertyName(ctx, prop.AsShorthandPropertyAssignment().Name())
			if name != "" {
				objectLiteralMap[name] = name
			}
		}
	}

	return objectLiteralMap, nil
}

func (p *TsParser) tsgoExtractObjectProperties(ctx *tsgoImportExportCtx, objectLiteral *tsast.Node) []*ObjectProperty {
	properties := make([]*ObjectProperty, 0)
	if objectLiteral == nil || objectLiteral.Kind != tsast.KindObjectLiteralExpression {
		return properties
	}

	for _, prop := range objectLiteral.AsObjectLiteralExpression().Properties.Nodes {
		if prop == nil || prop.Kind != tsast.KindPropertyAssignment {
			continue
		}
		assignment := prop.AsPropertyAssignment()
		name := tsgoPropertyName(ctx, assignment.Name())
		_, start, end, line, column := tsgoNodeStableInfo(ctx, prop)
		valueKind := tsgoNodeTypeName(assignment.Initializer)
		valueText := ""
		valueStart, valueEnd := 0, 0
		if assignment.Initializer != nil {
			valueText = ctx.nodeText(assignment.Initializer)
			valueStart = assignment.Initializer.Pos()
			valueEnd = assignment.Initializer.End()
		}

		properties = append(properties, &ObjectProperty{
			Name:       name,
			ValueKind:  valueKind,
			ValueText:  valueText,
			Start:      start,
			End:        end,
			Line:       line,
			Column:     column,
			ValueStart: valueStart,
			ValueEnd:   valueEnd,
		})
	}

	return properties
}

func (p *TsParser) tsgoParseDecorator(ctx *tsgoImportExportCtx, decoratorNode *tsast.Node) (*Decorator, error) {
	if ctx == nil || decoratorNode == nil || decoratorNode.Kind != tsast.KindDecorator {
		return nil, nil
	}

	decoratorText, decoratorStart, decoratorEnd, decoratorLine, decoratorColumn := tsgoNodeStableInfo(ctx, decoratorNode)
	decorator := &Decorator{
		Text:      decoratorText,
		Start:     decoratorStart,
		End:       decoratorEnd,
		Line:      decoratorLine,
		Column:    decoratorColumn,
		Arguments: make([]*Argument, 0),
	}

	expr := decoratorNode.AsDecorator().Expression
	callee := expr
	var args []*tsast.Node
	if expr != nil && expr.Kind == tsast.KindCallExpression {
		call := expr.AsCallExpression()
		callee = call.Expression
		if call.Arguments != nil {
			args = call.Arguments.Nodes
		}
	}

	referenceIdents := tsgoCollectReferenceIdents(callee)
	referenceIdent := strings.Join(referenceIdents, ".")
	if len(referenceIdents) > 0 {
		decorator.Name = referenceIdents[len(referenceIdents)-1]
		decorator.ReferenceIdent = decorator.Name
	}

	if len(referenceIdents) > 1 {
		namespaceModuleSpec, namespaceReferenceIdent := p.ConvertReferenceWithModuleSpec(referenceIdents[0])
		decorator.ReferenceIdent = namespaceReferenceIdent
		decorator.ModuleSpecPath = namespaceModuleSpec
	} else if len(referenceIdents) == 1 {
		namespaceModuleSpec, _ := p.ConvertReferenceWithModuleSpec(referenceIdent)
		decorator.ModuleSpecPath = namespaceModuleSpec
	}

	for _, a := range args {
		if a == nil {
			continue
		}
		argText, argStart, argEnd, argLine, argColumn := tsgoNodeStableInfo(ctx, a)
		argument := &Argument{
			Type:   tsgoNodeTypeName(a),
			Text:   argText,
			Start:  argStart,
			End:    argEnd,
			Line:   argLine,
			Column: argColumn,
		}

		if a.Kind == tsast.KindObjectLiteralExpression {
			argument.ObjectProperties = p.tsgoExtractObjectProperties(ctx, a)
			value, err := p.tsgoConvertObjectLiteralToMap(ctx, a)
			if err != nil {
				return nil, err
			}
			jsonValue, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			argument.Value = string(jsonValue)
		} else if tsgoIsLiteralNode(a) {
			value, err := p.tsgoConvertLiteralToInterface(ctx, a)
			if err != nil {
				return nil, err
			}
			jsonValue, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			argument.Value = string(jsonValue)
		} else if a.Kind == tsast.KindArrayLiteralExpression {
			value, err := p.tsgoConvertArrayLiteralToArray(ctx, a)
			if err != nil {
				return nil, err
			}
			jsonValue, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			argument.Value = string(jsonValue)
		} else if a.Kind == tsast.KindIdentifier {
			argument.Value = a.Text()
			moduleSpec, referenceIdent := p.ConvertReferenceWithModuleSpec(a.Text())
			argument.ReferenceIdent = referenceIdent
			argument.ModuleSpecPath = moduleSpec
		} else {
			argument.Value = ctx.nodeText(a)
		}

		decorator.Arguments = append(decorator.Arguments, argument)
	}

	return decorator, nil
}

func (p *TsParser) tsgoParseExtends(ctx *tsgoImportExportCtx, classNode *tsast.Node) (*Extends, error) {
	if ctx == nil || classNode == nil || classNode.Kind != tsast.KindClassDeclaration {
		return nil, nil
	}
	classDecl := classNode.AsClassDeclaration()
	if classDecl.HeritageClauses == nil {
		return nil, nil
	}

	for _, heritageNode := range classDecl.HeritageClauses.Nodes {
		if heritageNode == nil || heritageNode.Kind != tsast.KindHeritageClause {
			continue
		}
		heritage := heritageNode.AsHeritageClause()
		if heritage.Token != tsast.KindExtendsKeyword || heritage.Types == nil || len(heritage.Types.Nodes) == 0 {
			continue
		}

		extendsExprNode := heritage.Types.Nodes[0]
		if extendsExprNode != nil && extendsExprNode.Kind == tsast.KindExpressionWithTypeArguments {
			extendsExprNode = extendsExprNode.AsExpressionWithTypeArguments().Expression
		}
		if extendsExprNode == nil {
			continue
		}

		extendsText, extendsStart, extendsEnd, extendsLine, extendsColumn := tsgoNodeStableInfo(ctx, heritageNode)
		extendsText = strings.TrimSpace(extendsText)
		extends := &Extends{
			Text:   extendsText,
			Start:  extendsStart,
			End:    extendsEnd,
			Line:   extendsLine,
			Column: extendsColumn,
		}

		switch extendsExprNode.Kind {
		case tsast.KindIdentifier:
			extends.Name = extendsExprNode.Text()
			moduleSpec, referenceIdent := p.ConvertReferenceWithModuleSpec(extendsExprNode.Text())
			extends.ReferenceIdent = referenceIdent
			extends.ModuleSpecPath = moduleSpec
		case tsast.KindPropertyAccessExpression:
			pa := extendsExprNode.AsPropertyAccessExpression()
			extends.Name = pa.Name().Text()
			moduleSpec, _ := p.ConvertReferenceWithModuleSpec(pa.Expression.Text())
			extends.ReferenceIdent = pa.Name().Text()
			extends.ModuleSpecPath = moduleSpec
		default:
			extends.Name = strings.TrimSpace(ctx.nodeText(extendsExprNode))
			extends.ReferenceIdent = extends.Name
			extends.ModuleSpecPath = strings.TrimSuffix(p.Path, ".ts")
		}

		return extends, nil
	}

	return nil, nil
}

func (p *TsParser) tsgoParseMemberVar(ctx *tsgoImportExportCtx, memberVarNode *tsast.Node) (*MemberVar, error) {
	if ctx == nil || memberVarNode == nil || memberVarNode.Kind != tsast.KindPropertyDeclaration {
		return nil, nil
	}

	memberVarText, memberVarStart, memberVarEnd, memberVarLine, memberVarColumn := tsgoNodeStableInfo(ctx, memberVarNode)
	propDecl := memberVarNode.AsPropertyDeclaration()
	memberVar := &MemberVar{
		Decorators: make([]*Decorator, 0),
		Text:       memberVarText,
		Start:      memberVarStart,
		End:        memberVarEnd,
		Line:       memberVarLine,
		Column:     memberVarColumn,
		Name:       tsgoPropertyName(ctx, propDecl.Name()),
	}

	if memberVar.Name == "" {
		return nil, nil
	}

	typeAnnotationNode := propDecl.Type
	if typeAnnotationNode != nil && typeAnnotationNode.Kind == tsast.KindTypeLiteral {
		memberVar.TypeAnnotation = "jsonobject"
	} else if typeAnnotationNode != nil {
		memberVar.TypeAnnotation = strings.TrimSpace(ctx.nodeText(typeAnnotationNode))
	}

	tsTypeReferenceNode := typeAnnotationNode
	if typeAnnotationNode != nil && typeAnnotationNode.Kind == tsast.KindArrayType {
		tsTypeReferenceNode = typeAnnotationNode.AsArrayTypeNode().ElementType
	}

	if typeAnnotationNode != nil && typeAnnotationNode.Kind == tsast.KindTypeLiteral {
		memberVar.TsTypeReference = "jsonobject"
	} else if tsTypeReferenceNode != nil {
		memberVar.TsTypeReference = strings.TrimSpace(ctx.nodeText(tsTypeReferenceNode))
	}

	moduleSpec, referenceIdent := p.ConvertReferenceWithModuleSpec(memberVar.TsTypeReference)
	memberVar.ReferenceIdent = referenceIdent
	memberVar.ModuleSpecPath = moduleSpec

	for _, d := range memberVarNode.Decorators() {
		decorator, err := p.tsgoParseDecorator(ctx, d)
		if err != nil {
			return nil, err
		}
		if decorator != nil {
			memberVar.Decorators = append(memberVar.Decorators, decorator)
		}
	}

	accessibilityModifier := "public"
	if tsast.HasSyntacticModifier(memberVarNode, tsast.ModifierFlagsPrivate) {
		accessibilityModifier = "private"
	} else if tsast.HasSyntacticModifier(memberVarNode, tsast.ModifierFlagsProtected) {
		accessibilityModifier = "protected"
	} else if tsast.HasSyntacticModifier(memberVarNode, tsast.ModifierFlagsPublic) {
		accessibilityModifier = "public"
	}
	memberVar.AccessibilityModifier = accessibilityModifier
	if strings.HasPrefix(memberVar.Name, "#") {
		memberVar.AccessibilityModifier = "private"
	}

	memberVar.IsStatic = tsast.HasSyntacticModifier(memberVarNode, tsast.ModifierFlagsStatic)
	memberVar.IsReadonly = tsast.HasSyntacticModifier(memberVarNode, tsast.ModifierFlagsReadonly)

	return memberVar, nil
}

func (p *TsParser) tsgoParseMemberMethod(ctx *tsgoImportExportCtx, memberMethodNode *tsast.Node) (*MemberMethod, error) {
	if ctx == nil || memberMethodNode == nil || memberMethodNode.Kind != tsast.KindMethodDeclaration {
		return nil, nil
	}
	if !tsast.HasSyntacticModifier(memberMethodNode, tsast.ModifierFlagsAsync) {
		return nil, nil
	}

	memberMethodText, memberMethodStart, memberMethodEnd, memberMethodLine, memberMethodColumn := tsgoNodeStableInfo(ctx, memberMethodNode)
	methodDecl := memberMethodNode.AsMethodDeclaration()
	memberMethod := &MemberMethod{
		Decorators:     make([]*Decorator, 0),
		Parameters:     make([]*Parameter, 0),
		TypeParameters: make([]*TypeParameter, 0),
		Text:           memberMethodText,
		Start:          memberMethodStart,
		End:            memberMethodEnd,
		Line:           memberMethodLine,
		Column:         memberMethodColumn,
	}

	for _, d := range memberMethodNode.Decorators() {
		decorator, err := p.tsgoParseDecorator(ctx, d)
		if err != nil {
			return nil, err
		}
		if decorator != nil {
			memberMethod.Decorators = append(memberMethod.Decorators, decorator)
		}
	}

	memberMethod.Name = tsgoPropertyName(ctx, methodDecl.Name())
	if memberMethod.Name == "" || memberMethod.Name == "constructor" {
		return nil, nil
	}

	accessibilityModifier := "public"
	if tsast.HasSyntacticModifier(memberMethodNode, tsast.ModifierFlagsPrivate) {
		accessibilityModifier = "private"
	} else if tsast.HasSyntacticModifier(memberMethodNode, tsast.ModifierFlagsProtected) {
		accessibilityModifier = "protected"
	} else if tsast.HasSyntacticModifier(memberMethodNode, tsast.ModifierFlagsPublic) {
		accessibilityModifier = "public"
	}
	memberMethod.AccessibilityModifier = accessibilityModifier
	if strings.HasPrefix(memberMethod.Name, "#") {
		memberMethod.AccessibilityModifier = "private"
	}

	memberMethod.IsStatic = tsast.HasSyntacticModifier(memberMethodNode, tsast.ModifierFlagsStatic)

	if methodDecl.TypeParameters != nil {
		for _, typeParameterNode := range methodDecl.TypeParameters.Nodes {
			if typeParameterNode == nil || typeParameterNode.Kind != tsast.KindTypeParameter {
				continue
			}
			typeParamText, typeParamStart, typeParamEnd, typeParamLine, typeParamColumn := tsgoNodeStableInfo(ctx, typeParameterNode)
			typeParamDecl := typeParameterNode.AsTypeParameterDeclaration()
			parameter := &TypeParameter{
				Name:   tsgoPropertyName(ctx, typeParamDecl.Name()),
				Text:   typeParamText,
				Start:  typeParamStart,
				End:    typeParamEnd,
				Line:   typeParamLine,
				Column: typeParamColumn,
			}

			if typeParamDecl.Constraint != nil && typeParamDecl.Constraint.Kind == tsast.KindTypeReference {
				typeName := strings.TrimSpace(ctx.nodeText(typeParamDecl.Constraint.AsTypeReferenceNode().TypeName))
				if typeName != "" {
					moduleSpec, referenceIdent := p.ConvertReferenceWithModuleSpec(typeName)
					parameter.ModuleSpecPath = moduleSpec
					parameter.ReferenceIdent = referenceIdent
				}
			}

			memberMethod.TypeParameters = append(memberMethod.TypeParameters, parameter)
		}
	}

	if methodDecl.Type != nil {
		memberMethod.TypeAnnotation = strings.TrimSpace(ctx.nodeText(methodDecl.Type))
	}

	if methodDecl.Parameters != nil {
		for _, paramNode := range methodDecl.Parameters.Nodes {
			if paramNode == nil || paramNode.Kind != tsast.KindParameter {
				continue
			}
			paramText, paramStart, paramEnd, paramLine, paramColumn := tsgoNodeStableInfo(ctx, paramNode)
			paramDecl := paramNode.AsParameterDeclaration()
			parameter := &Parameter{
				Text:   paramText,
				Start:  paramStart,
				End:    paramEnd,
				Line:   paramLine,
				Column: paramColumn,
			}
			if paramDecl.Name() != nil {
				if tsast.IsThisIdentifier(paramDecl.Name()) {
					parameter.Name = "this"
				} else {
					parameter.Name = tsgoPropertyName(ctx, paramDecl.Name())
				}
			}
			if paramDecl.Type != nil {
				parameter.TypeAnnotation = strings.TrimSpace(ctx.nodeText(paramDecl.Type))
			}
			memberMethod.Parameters = append(memberMethod.Parameters, parameter)
		}
	}

	return memberMethod, nil
}

func (p *TsParser) tsgoFindClassNode(ctx *tsgoImportExportCtx, targetName string) *tsast.Node {
	if ctx == nil || ctx.source == nil {
		return nil
	}

	classesByName := make(map[string]*tsast.Node)
	defaultExportRef := ""

	for _, stmt := range ctx.source.Statements.Nodes {
		if stmt == nil {
			continue
		}
		if stmt.Kind == tsast.KindClassDeclaration {
			name := ""
			if stmt.Name() != nil {
				name = strings.TrimSpace(stmt.Name().Text())
			}
			if name != "" {
				classesByName[name] = stmt
			}
			if tsgoIsExportDefaultDeclaration(stmt) {
				if targetName == "" || targetName == name {
					return stmt
				}
			}
		}
		if stmt.Kind == tsast.KindExportAssignment {
			expr := stmt.AsExportAssignment().Expression
			if expr != nil && expr.Kind == tsast.KindIdentifier {
				defaultExportRef = strings.TrimSpace(expr.Text())
			}
		}
	}

	if targetName != "" {
		if classNode, ok := classesByName[targetName]; ok {
			return classNode
		}
	}

	if defaultExportRef != "" {
		if classNode, ok := classesByName[defaultExportRef]; ok {
			return classNode
		}
	}

	return nil
}

func (p *TsParser) tsgoParseClassNode(targetName string) (*Class, error) {
	ctx, err := p.parseTSGoImportExportCtx()
	if err != nil {
		return nil, err
	}
	// Keep ParseClassNode self-contained: extends/decorator resolution should not
	// depend on callers invoking ParseImport/ParseExport beforehand.
	p.ImportsMap = ctx.imports
	p.ExportsMap = ctx.exports

	classNode := p.tsgoFindClassNode(ctx, targetName)
	if classNode == nil || classNode.Kind != tsast.KindClassDeclaration {
		return nil, nil
	}

	classText, classStart, classEnd, classLine, classColumn := tsgoNodeStableInfo(ctx, classNode)
	class := &Class{
		Text:          classText,
		Start:         classStart,
		End:           classEnd,
		Line:          classLine,
		Column:        classColumn,
		Decorators:    make([]*Decorator, 0),
		MemberVars:    make([]*MemberVar, 0),
		MemberMethods: make([]*MemberMethod, 0),
		Abstract:      tsast.HasSyntacticModifier(classNode, tsast.ModifierFlagsAbstract),
	}

	if classNode.Name() != nil {
		class.Name = strings.TrimSpace(classNode.Name().Text())
	}

	for _, d := range classNode.Decorators() {
		decorator, err := p.tsgoParseDecorator(ctx, d)
		if err != nil {
			return nil, err
		}
		if decorator != nil {
			class.Decorators = append(class.Decorators, decorator)
		}
	}

	extends, err := p.tsgoParseExtends(ctx, classNode)
	if err != nil {
		return nil, err
	}
	class.Extends = extends

	classDecl := classNode.AsClassDeclaration()
	if classDecl.Members != nil {
		for _, memberNode := range classDecl.Members.Nodes {
			if memberNode == nil {
				continue
			}
			if memberNode.Kind == tsast.KindPropertyDeclaration {
				memberVar, err := p.tsgoParseMemberVar(ctx, memberNode)
				if err != nil {
					return nil, err
				}
				if memberVar != nil {
					class.MemberVars = append(class.MemberVars, memberVar)
				}
				continue
			}
			if memberNode.Kind == tsast.KindMethodDeclaration {
				memberMethod, err := p.tsgoParseMemberMethod(ctx, memberNode)
				if err != nil {
					return nil, err
				}
				if memberMethod != nil {
					class.MemberMethods = append(class.MemberMethods, memberMethod)
				}
			}
		}
	}

	return class, nil
}

func (p *TsParser) ParseClassNode(_ any, _ any) (*Class, error) {
	return p.tsgoParseClassNode("")
}

// func (p *TsParser) ParseClasses(root *ast.Node) ([]*Class, error) {
// 	classes := make([]*Class, 0)
// 	for _, c := range root.Children(selector.OneOf([]js.NodeType{js.ExportDefault, js.ExportDecl, js.Class}...)) {
// 		var classNode *ast.Node
// 		var decoratorNodes []*ast.Node
// 		if c.Type() == js.ExportDefault || c.Type() == js.ExportDecl {
// 			classNode = c.Child(selector.Class)
// 			decoratorNodes = c.Children(selector.Decorator)
// 		} else {
// 			classNode = c
// 			decoratorNodes = classNode.Children(selector.Decorator)
// 		}
// 		if classNode == nil {
// 			continue
// 		}
// 		class, err := p.ParseClassNode(decoratorNodes, classNode)
// 		if err != nil {
// 			return nil, err
// 		}
// 		classes = append(classes, class)
// 	}
// 	return classes, nil
// }
