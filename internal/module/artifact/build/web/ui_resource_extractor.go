// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

type uiResourceValidationWarning struct {
	resourceID string
	sourcePath string
	line       int
	column     int
	code       string
	hint       string
	message    string
}

type uiResourceMenuRouteRef struct {
	MenuName  string
	RouteName string
}

type uiResourceRouteActionRef struct {
	RouteName  string
	ActionName string
}

type collectedUiDecls struct {
	orderedIDs []string
	declByID   map[string]*parser.UiResourceDecl
	warnings   []uiResourceValidationWarning
}

const (
	uiRuleDuplicateWithoutOverride   = "UI_VAL_002"
	uiRuleOverrideIncompatible       = "UI_VAL_003"
	uiRuleActionDefaultRolesRequires = "UI_VAL_004"
	uiRuleDanglingParentFallback     = "UI_VAL_005"
	uiRuleMenuPathNoRoute            = "UI_VAL_006"
	uiRuleRouteActionTargetInvalid   = "UI_VAL_007"
	uiRuleMenuPathAmbiguousRoute     = "UI_VAL_008"
)

func collectFinalUiResourceDecls(parseResults []*parser.ParserResult) (*collectedUiDecls, error) {
	if len(parseResults) == 0 {
		return &collectedUiDecls{
			orderedIDs: nil,
			declByID:   map[string]*parser.UiResourceDecl{},
			warnings:   nil,
		}, nil
	}

	declByID := make(map[string]*parser.UiResourceDecl)
	orderedIDs := make([]string, 0)
	warnings := make([]uiResourceValidationWarning, 0)
	fatalIssues := make([]string, 0)

	for _, pr := range parseResults {
		if pr == nil {
			continue
		}

		for _, issue := range pr.UiResourceDeclIssues {
			if issue == nil {
				continue
			}
			formatted := formatUiIssue(issue)
			hint := hintForParserIssueCode(issue.Code)
			if issue.Severity == parser.UiResourceIssueSeverityFatal {
				fatalIssues = append(fatalIssues, formatted)
				continue
			}
			warnings = append(warnings, uiResourceValidationWarning{
				resourceID: strings.TrimSpace(issue.ResourceID),
				sourcePath: strings.TrimSpace(issue.SourcePath),
				line:       issue.Line,
				column:     issue.Column,
				code:       string(issue.Code),
				hint:       hint,
				message:    formatted,
			})
		}

		if len(pr.UiResourceDecls) == 0 {
			continue
		}
		for _, decl := range pr.UiResourceDecls {
			if decl == nil {
				continue
			}
			id := strings.TrimSpace(decl.ID)
			if id == "" {
				continue
			}

			if prev, exists := declByID[id]; exists {
				if equivalentUiDecl(prev, decl) {
					continue
				}
				if !prev.Override && !decl.Override {
					return nil, newUiRuleError(
						uiRuleDuplicateWithoutOverride,
						decl,
						id,
						fmt.Sprintf("duplicate ui resource id %q without override=true", id),
						"set override: true on the later declaration if intentional, otherwise rename one of the duplicated resource ids",
					)
				}
				if err := validateOverrideCompatibility(id, prev, decl); err != nil {
					return nil, newUiRuleError(
						uiRuleOverrideIncompatible,
						decl,
						id,
						err.Error(),
						"keep resource type unchanged when override=true; only override mutable display or relation fields",
					)
				}
			}

			if _, exists := declByID[id]; !exists {
				orderedIDs = append(orderedIDs, id)
			}
			declByID[id] = decl
		}
	}

	if len(fatalIssues) > 0 {
		sort.Strings(fatalIssues)
		return nil, fmt.Errorf("%s", strings.Join(fatalIssues, "; "))
	}

	return &collectedUiDecls{
		orderedIDs: orderedIDs,
		declByID:   declByID,
		warnings:   warnings,
	}, nil
}

func extractUiResources(module *meta.IrModule, parseResults []*parser.ParserResult) ([]*meta.IrUiResource, []uiResourceValidationWarning, error) {
	if module == nil || len(parseResults) == 0 {
		return nil, nil, nil
	}

	decls, err := collectFinalUiResourceDecls(parseResults)
	if err != nil {
		return nil, nil, err
	}
	return buildUiResources(module, decls)
}

func buildUiResources(module *meta.IrModule, decls *collectedUiDecls) ([]*meta.IrUiResource, []uiResourceValidationWarning, error) {
	if module == nil || decls == nil {
		return nil, nil, nil
	}
	if len(decls.orderedIDs) == 0 {
		return nil, append([]uiResourceValidationWarning(nil), decls.warnings...), nil
	}

	resourceByID := make(map[string]*meta.IrUiResource)
	warnings := append([]uiResourceValidationWarning(nil), decls.warnings...)
	moduleApp := resolveModuleApplication(module)

	for _, id := range decls.orderedIDs {
		decl := decls.declByID[id]
		if decl == nil {
			continue
		}

		requires := mustJSON(decl.Requires)
		defaultRoles := mustJSON(decl.DefaultRoles)

		v := &meta.IrUiResource{
			Name:               id,
			Type:               meta.UiResourceType(decl.Type),
			Title:              strings.TrimSpace(decl.Title),
			Sequence:           decl.Sequence,
			Requires:           requires,
			Module:             strings.TrimSpace(module.Name),
			Path:               strings.TrimSpace(decl.SourcePath),
			ParentResourceName: strings.TrimSpace(decl.ParentMenu),
			IrApplicationId:    moduleApp,
			UiPath:             strings.TrimSpace(decl.Path),
			DefaultRoles:       defaultRoles,
		}
		if module.Id.Valid {
			v.ModuleId = module.Id
		}

		if v.Type == meta.UiResourceTypeAction {
			hasDefaultRoles := len(parseJSONStrings(v.DefaultRoles)) > 0
			hasRequires := len(parseJSONStrings(v.Requires)) > 0
			if hasDefaultRoles && hasRequires {
				return nil, nil, newUiRuleError(
					uiRuleActionDefaultRolesRequires,
					decl,
					v.Name,
					fmt.Sprintf("action %q with non-empty requires cannot declare defaultRoles", v.Name),
					"remove defaultRoles or make requires empty for pure-frontend actions",
				)
			}
		}

		resourceByID[id] = v
	}

	menuIDs := make(map[string]bool)
	menuChildren := make(map[string]int)
	routePathSet := make(map[string]bool)
	for _, id := range decls.orderedIDs {
		r := resourceByID[id]
		if r == nil {
			continue
		}
		if r.Type == meta.UiResourceTypeMenu {
			menuIDs[r.Name] = true
		}
		if r.Type == meta.UiResourceTypeRoute {
			p := normalizeUiPath(r.UiPath)
			if p != "" {
				routePathSet[p] = true
			}
		}
	}
	for _, id := range decls.orderedIDs {
		r := resourceByID[id]
		if r == nil || r.ParentResourceName == "" {
			continue
		}
		if _, ok := resourceByID[r.ParentResourceName]; ok {
			menuChildren[r.ParentResourceName]++
		}
		if !menuIDs[r.ParentResourceName] {
			decl := decls.declByID[r.Name]
			sourcePath, sourceLine, sourceColumn := declLocation(decl)
			hint := "fix parentMenu to an existing MENU resource id or keep it empty to make this node top-level"
			warnings = append(warnings, uiResourceValidationWarning{
				resourceID: r.Name,
				sourcePath: sourcePath,
				line:       sourceLine,
				column:     sourceColumn,
				code:       uiRuleDanglingParentFallback,
				hint:       hint,
				message:    formatUiRuleDiagnostic(uiRuleDanglingParentFallback, decl, r.Name, fmt.Sprintf("parentMenu %q not found or not MENU; falling back to top-level", r.ParentResourceName), hint),
			})
			r.ParentResourceName = ""
		}
	}

	for _, id := range decls.orderedIDs {
		r := resourceByID[id]
		if r == nil || r.Type != meta.UiResourceTypeMenu {
			continue
		}
		if strings.TrimSpace(r.UiPath) == "" {
			continue
		}
		if menuChildren[r.Name] > 0 {
			continue
		}
		if isExternalLink(r.UiPath) {
			continue
		}
		norm := normalizeUiPath(r.UiPath)
		if norm == "" {
			continue
		}
		if !routePathSet[norm] {
			decl := decls.declByID[r.Name]
			return nil, warnings, newUiRuleError(
				uiRuleMenuPathNoRoute,
				decl,
				r.Name,
				fmt.Sprintf("menu %q path %q does not match any route path", r.Name, r.UiPath),
				"add or fix a matching defineRoute path, or mark this menu as external link when it is an outbound URL",
			)
		}
	}

	out := make([]*meta.IrUiResource, 0, len(decls.orderedIDs))
	for _, id := range decls.orderedIDs {
		out = append(out, resourceByID[id])
	}

	sort.SliceStable(warnings, func(i, j int) bool {
		if warnings[i].resourceID == warnings[j].resourceID {
			return warnings[i].message < warnings[j].message
		}
		return warnings[i].resourceID < warnings[j].resourceID
	})

	return out, warnings, nil
}

func extractUiResourceRelations(uiResources []*meta.IrUiResource, parseResults []*parser.ParserResult) ([]uiResourceMenuRouteRef, []uiResourceRouteActionRef, error) {
	decls, err := collectFinalUiResourceDecls(parseResults)
	if err != nil {
		return nil, nil, err
	}
	return buildUiResourceRelations(decls, uiResources)
}

func buildUiResourceRelations(decls *collectedUiDecls, uiResources []*meta.IrUiResource) ([]uiResourceMenuRouteRef, []uiResourceRouteActionRef, error) {
	if decls == nil || len(decls.orderedIDs) == 0 || len(uiResources) == 0 {
		return nil, nil, nil
	}

	resourceByName := make(map[string]*meta.IrUiResource, len(uiResources))
	routeNamesByPath := make(map[string][]string)
	for _, resource := range uiResources {
		if resource == nil {
			continue
		}
		name := strings.TrimSpace(resource.Name)
		if name == "" {
			continue
		}
		resourceByName[name] = resource
		if resource.Type != meta.UiResourceTypeRoute {
			continue
		}
		path := normalizeUiPath(resource.UiPath)
		if path == "" {
			continue
		}
		existing := routeNamesByPath[path]
		if !slices.Contains(existing, name) {
			routeNamesByPath[path] = append(existing, name)
		}
	}

	menuRoutes := make([]uiResourceMenuRouteRef, 0)
	routeActions := make([]uiResourceRouteActionRef, 0)
	seenMenuRoutes := make(map[string]bool)
	seenRouteActions := make(map[string]bool)

	for _, id := range decls.orderedIDs {
		decl := decls.declByID[id]
		if decl == nil {
			continue
		}

		switch decl.Type {
		case parser.UiResourceTypeMenu:
			resource := resourceByName[id]
			if resource == nil || resource.Type != meta.UiResourceTypeMenu {
				continue
			}
			if isExternalLink(resource.UiPath) {
				continue
			}
			path := normalizeUiPath(resource.UiPath)
			if path == "" {
				continue
			}
			routeNames := append([]string(nil), routeNamesByPath[path]...)
			if len(routeNames) == 0 {
				continue
			}
			sort.Strings(routeNames)
			if len(routeNames) > 1 {
				return nil, nil, newUiRuleError(
					uiRuleMenuPathAmbiguousRoute,
					decl,
					id,
					fmt.Sprintf("menu %q path %q matches multiple route resources: %s", id, resource.UiPath, strings.Join(routeNames, ", ")),
					"keep only one ROUTE with the same normalized path, or change the menu path so MENU -> ROUTE inference is unambiguous",
				)
			}
			key := id + "/" + routeNames[0]
			if seenMenuRoutes[key] {
				continue
			}
			seenMenuRoutes[key] = true
			menuRoutes = append(menuRoutes, uiResourceMenuRouteRef{MenuName: id, RouteName: routeNames[0]})
		case parser.UiResourceTypeRoute:
			resource := resourceByName[id]
			if resource == nil || resource.Type != meta.UiResourceTypeRoute {
				continue
			}
			for _, raw := range decl.Actions {
				actionName := strings.TrimSpace(raw)
				if actionName == "" {
					continue
				}
				key := id + "/" + actionName
				if seenRouteActions[key] {
					continue
				}
				seenRouteActions[key] = true
				routeActions = append(routeActions, uiResourceRouteActionRef{RouteName: id, ActionName: actionName})
			}
		}
	}

	return menuRoutes, routeActions, nil
}

func equivalentUiDecl(a *parser.UiResourceDecl, b *parser.UiResourceDecl) bool {
	if a == nil || b == nil {
		return false
	}
	if strings.TrimSpace(a.ID) != strings.TrimSpace(b.ID) {
		return false
	}
	if a.Type != b.Type {
		return false
	}
	if strings.TrimSpace(a.Title) != strings.TrimSpace(b.Title) {
		return false
	}
	if a.Sequence != b.Sequence {
		return false
	}
	if strings.TrimSpace(a.ParentMenu) != strings.TrimSpace(b.ParentMenu) {
		return false
	}
	if strings.TrimSpace(a.Path) != strings.TrimSpace(b.Path) {
		return false
	}
	if !equalTrimmedStringSlices(a.Requires, b.Requires) {
		return false
	}
	if !equalTrimmedStringSlices(a.Actions, b.Actions) {
		return false
	}
	if !equalTrimmedStringSlices(a.DefaultRoles, b.DefaultRoles) {
		return false
	}

	return true
}

func equalTrimmedStringSlices(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}

func resolveModuleApplication(module *meta.IrModule) string {
	if module == nil {
		return ""
	}
	if module.ApplicationId.Valid {
		if id := strings.TrimSpace(module.ApplicationId.String); id != "" {
			return id
		}
	}
	if module.Application != nil {
		if id := strings.TrimSpace(module.Application.Id.String); id != "" {
			return id
		}
	}
	if v := strings.TrimSpace(module.ApplicationStr); v != "" {
		if strings.EqualFold(v, "web") {
			return ""
		}
		return v
	}
	if module.Application != nil {
		if v := strings.TrimSpace(module.Application.Name); v != "" {
			return v
		}
	}
	if name := strings.TrimSpace(module.Name); name != "" {
		parts := strings.Split(name, "_")
		prefix := strings.TrimSpace(parts[0])
		if prefix != "" {
			return prefix
		}
	}
	return ""
}

func formatUiIssue(issue *parser.UiResourceDeclIssue) string {
	if issue == nil {
		return ""
	}
	parts := make([]string, 0, 3)
	if issue.SourcePath != "" {
		location := issue.SourcePath
		if issue.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, issue.Line)
			if issue.Column > 0 {
				location = fmt.Sprintf("%s:%d", location, issue.Column)
			}
		}
		parts = append(parts, location)
	}
	if issue.Factory != "" {
		parts = append(parts, issue.Factory)
	}
	if issue.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", issue.Code))
	}
	if issue.ResourceID != "" {
		parts = append(parts, fmt.Sprintf("resourceId=%s", issue.ResourceID))
	}
	prefix := ""
	if len(parts) > 0 {
		prefix = strings.Join(parts, " ") + " "
	}
	message := strings.TrimSpace(prefix + issue.Message)
	hint := hintForParserIssueCode(issue.Code)
	if hint != "" {
		message = fmt.Sprintf("%s; hint: %s", message, hint)
	}
	return message
}

func hintForParserIssueCode(code parser.UiResourceIssueCode) string {
	switch code {
	case parser.UiResourceIssueCodeDeclIDNotLiteral:
		return "use a string literal id in defineRoute/defineMenu/defineAction"
	case parser.UiResourceIssueCodeDeclRequiresNotLiteral:
		return "use a static object array for requires, for example [{ model: 'auth.User' }] or [{ model: 'auth.User', method: 'Browse' }]"
	case parser.UiResourceIssueCodeDeclDefaultRolesNotLiteral:
		return "use a static string array for defaultRoles, for example ['base.user']"
	case parser.UiResourceIssueCodeDeclIDNamingSuggested:
		return "prefer [application].[type].[name] with lowercase snake_case segments, for example auth.route.user_list"
	case parser.UiResourceIssueCodeModelIDNotLiteral:
		return "use a literal model id like auth.User in defineModelActions"
	case parser.UiResourceIssueCodeModelIDInvalidFormat, parser.UiResourceIssueCodeModelIDEmptySegment:
		return "model id must follow [application].[ModelName], for example auth.User"
	case parser.UiResourceIssueCodeRouteActionsNotLiteral, parser.UiResourceIssueCodeRouteActionEntryNotLiteral:
		return "use a static string array for actions, for example ['auth.action.user_edit']"
	case parser.UiResourceIssueCodePublicRouteHasActions:
		return "remove actions from public routes; only authenticated routes can bind ACTION resources"
	case parser.UiResourceIssueCodeParentMenuOnlyForMenu:
		return "keep parentMenu only on defineMenu; use menu path inference and route.actions for other relations"
	default:
		return ""
	}
}

func formatUiRuleDiagnostic(code string, decl *parser.UiResourceDecl, resourceID string, detail string, hint string) string {
	parts := make([]string, 0, 4)
	if strings.TrimSpace(code) != "" {
		parts = append(parts, fmt.Sprintf("[%s]", strings.TrimSpace(code)))
	}
	if decl != nil && strings.TrimSpace(decl.SourcePath) != "" {
		location := strings.TrimSpace(decl.SourcePath)
		if decl.SourceLine > 0 {
			location = fmt.Sprintf("%s:%d", location, decl.SourceLine)
			if decl.SourceColumn > 0 {
				location = fmt.Sprintf("%s:%d", location, decl.SourceColumn)
			}
		}
		parts = append(parts, location)
	}
	rid := strings.TrimSpace(resourceID)
	if rid != "" {
		parts = append(parts, fmt.Sprintf("resourceId=%s", rid))
	}
	parts = append(parts, strings.TrimSpace(detail))

	msg := strings.TrimSpace(strings.Join(parts, " "))
	if strings.TrimSpace(hint) != "" {
		msg = fmt.Sprintf("%s; hint: %s", msg, strings.TrimSpace(hint))
	}
	return msg
}

func newUiRuleError(code string, decl *parser.UiResourceDecl, resourceID string, detail string, hint string) error {
	return fmt.Errorf("%s", formatUiRuleDiagnostic(code, decl, resourceID, detail, hint))
}

func declLocation(decl *parser.UiResourceDecl) (string, int, int) {
	if decl == nil {
		return "", 0, 0
	}
	return strings.TrimSpace(decl.SourcePath), decl.SourceLine, decl.SourceColumn
}

func normalizeUiPath(p string) string {
	v := strings.TrimSpace(p)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "/") {
		v = "/" + v
	}
	v = strings.TrimRight(v, "/")
	if v == "" {
		return "/"
	}
	return v
}

func isExternalLink(p string) bool {
	v := strings.ToLower(strings.TrimSpace(p))
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")
}

func mustJSON(value any) []byte {
	if value == nil {
		return []byte("[]")
	}
	b, err := json.Marshal(value)
	if err != nil {
		return []byte("[]")
	}
	return b
}

func validateOverrideCompatibility(id string, prev *parser.UiResourceDecl, next *parser.UiResourceDecl) error {
	if prev == nil || next == nil {
		return nil
	}
	if prev.Type != next.Type {
		return fmt.Errorf("override for %q cannot change type from %s to %s", id, prev.Type, next.Type)
	}
	return nil
}
