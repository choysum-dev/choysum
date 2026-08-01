// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"github.com/choysum-dev/choysum/pkg/meta"
	"golang.org/x/net/html"
)

type UiResourceType string

type UiResourceIssueSeverity string
type UiResourceIssueCode string

const (
	UiResourceTypeRoute  UiResourceType = "ROUTE"
	UiResourceTypeMenu   UiResourceType = "MENU"
	UiResourceTypeAction UiResourceType = "ACTION"

	UiResourceIssueSeverityWarning UiResourceIssueSeverity = "WARNING"
	UiResourceIssueSeverityFatal   UiResourceIssueSeverity = "FATAL"

	// Parser/declaration-level issue codes. These diagnose authoring-shape violations
	// before extractor-level cross-file validation runs.
	UiResourceIssueCodeDeclIDNotLiteral                 UiResourceIssueCode = "UI_DECL_001"
	UiResourceIssueCodeDeclRequiresNotLiteral           UiResourceIssueCode = "UI_DECL_002"
	UiResourceIssueCodeDeclDefaultRolesNotLiteral       UiResourceIssueCode = "UI_DECL_003"
	UiResourceIssueCodeModelIDNotLiteral                UiResourceIssueCode = "UI_DECL_004"
	UiResourceIssueCodeModelIDInvalidFormat             UiResourceIssueCode = "UI_DECL_005"
	UiResourceIssueCodeModelIDEmptySegment              UiResourceIssueCode = "UI_DECL_006"
	UiResourceIssueCodeDeclIDNamingSuggested            UiResourceIssueCode = "UI_DECL_007"
	UiResourceIssueCodeRouteActionsNotLiteral           UiResourceIssueCode = "UI_DECL_008"
	UiResourceIssueCodeRouteActionEntryNotLiteral       UiResourceIssueCode = "UI_DECL_009"
	UiResourceIssueCodePublicRouteHasActions            UiResourceIssueCode = "UI_DECL_010"
	UiResourceIssueCodeParentMenuOnlyForMenu            UiResourceIssueCode = "UI_DECL_011"
	UiResourceIssueCodeModelActionEntityTitleNotLiteral UiResourceIssueCode = "UI_DECL_012"
	UiResourceIssueCodeModelActionTitlesInvalid         UiResourceIssueCode = "UI_DECL_013"
)

type UiResourceDeclIssue struct {
	Severity   UiResourceIssueSeverity
	Code       UiResourceIssueCode
	Factory    string
	ResourceID string
	Message    string
	SourcePath string
	Line       int
	Column     int
}

// UiResourceDecl is a per-file, parser-level intermediate declaration.
// Cross-file merge and normalization are handled by the web extractor stage.
type UiResourceDecl struct {
	ID           string
	Type         UiResourceType
	Title        string
	TitleText    *meta.TermReference
	Sequence     int
	Requires     []string
	ParentMenu   string
	Path         string
	Actions      []string
	DefaultRoles []string
	Override     bool
	SourcePath   string
	SourceLine   int
	SourceColumn int
}

type ParserResult struct {
	Path       string
	RawContent string
	Content    string

	Exports              map[string]*Export
	Imports              map[string]*Import
	Model                *meta.Model   // backend ts models
	ModelClassNode       *Class        // model class node
	ModelExtendsProperty *PropertyNode // extends property in model class

	// vue components
	VueAppImportTree       []string // vue app import tree,would be used to esbuild onResolve
	VueComponent           *meta.Component
	VueComponentsPropertys []*PropertyNode // components property in defineComponent
	VueExtendsProperty     *PropertyNode   // extends property in defineComponent

	RawScriptNode      *html.Node
	RawScriptSetupNode *html.Node

	ScriptNode      *html.Node
	ScriptSetupNode *html.Node

	RawTemplateNode *html.Node
	TemplateNode    *html.Node

	RawStyleNodes []*html.Node
	StyleNodes    []*html.Node

	// ui resources (parser-level intermediate declarations)
	UiResourceDecls      []*UiResourceDecl
	UiResourceDeclIssues []*UiResourceDeclIssue
}

type Parser interface {
	Parse(pathAlias map[string]string, path string, content string) (*ParserResult, error)
}
