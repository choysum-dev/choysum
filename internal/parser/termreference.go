// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
)

var (
	termReferenceCallPattern     = regexp.MustCompile(`(?s)^([A-Za-z_$][\w$]*)\s*\(\s*("(?:\\.|[^"])*"|'(?:\\.|[^'])*'|` + "`[^`]*`" + `)(?:\s*,\s*\{(.*?)\})?\s*\)$`)
	referenceScopePattern        = regexp.MustCompile(`\bscope\s*:\s*("(?:\\.|[^"])*"|'(?:\\.|[^'])*'|` + "`[^`]*`" + `)`)
	referencePathPattern         = regexp.MustCompile(`\bpath\s*:\s*("(?:\\.|[^"])*"|'(?:\\.|[^'])*'|` + "`[^`]*`" + `)`)
	referenceLocationPattern     = regexp.MustCompile(`\blocation\s*:\s*("(?:\\.|[^"])*"|'(?:\\.|[^'])*'|` + "`[^`]*`" + `)`)
	createTranslateModulePattern = regexp.MustCompile(`(?s)^\s*("(?:\\.|[^"])*"|'(?:\\.|[^'])*')`)
)

type TranslateBinding struct {
	Module          string
	DefaultScope    string
	ReferenceOutput bool
}

// DeriveOwnerModuleFromSourcePath returns the manifest module name from a modules/<name>/ path.
func DeriveOwnerModuleFromSourcePath(sourcePath string) string {
	normalized := filepath.ToSlash(sourcePath)
	parts := strings.Split(normalized, "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "modules" && i+1 < len(parts) {
			return strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}

func parseTextCallLiteral(value string) (string, bool) {
	if strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`") {
		return strings.TrimSuffix(strings.TrimPrefix(value, "`"), "`"), true
	}
	parsed, err := ParseJSStringLiteral(value)
	return parsed, err == nil
}

func parseFactoryStringOption(options string, pattern *regexp.Regexp) string {
	match := pattern.FindStringSubmatch(options)
	if len(match) != 2 {
		return ""
	}
	raw := strings.TrimSpace(match[1])
	if strings.HasPrefix(raw, "`") && strings.HasSuffix(raw, "`") {
		return strings.TrimSuffix(strings.TrimPrefix(raw, "`"), "`")
	}
	parsed, err := ParseJSStringLiteral(raw)
	if err != nil {
		return ""
	}
	return parsed
}

func parseCreateTranslateOptions(options string) (defaultScope string) {
	options = strings.TrimSpace(options)
	if options == "" {
		return ""
	}
	if scopeValue := parseFactoryStringOption(options, referenceScopePattern); strings.TrimSpace(scopeValue) != "" {
		return scopeValue
	}
	pathValue := strings.TrimSpace(parseFactoryStringOption(options, referencePathPattern))
	locationValue := strings.TrimSpace(parseFactoryStringOption(options, referenceLocationPattern))
	if pathValue == "" {
		return ""
	}
	defaultScope = pathValue
	if locationValue != "" {
		defaultScope += "@" + locationValue
	}
	return defaultScope
}

// ParseTranslateBindings scans source for createTranslate destructuring bindings.
// `_t` aliases are text helpers; `_lt` aliases are TermReference helpers.
func ParseTranslateBindings(source string) map[string]TranslateBinding {
	bindings := map[string]TranslateBinding{}
	source = strings.TrimSpace(source)
	if source == "" {
		return bindings
	}

	createPattern := regexp.MustCompile(`(?s)\bconst\s*\{([^}]+)\}\s*=\s*createTranslate\s*\(`)
	for _, loc := range createPattern.FindAllStringSubmatchIndex(source, -1) {
		if len(loc) < 4 {
			continue
		}
		destructure := strings.TrimSpace(source[loc[2]:loc[3]])
		args, ok := parseBalancedCallArguments(source, loc[1]-1)
		if !ok {
			continue
		}
		moduleMatch := createTranslateModulePattern.FindStringSubmatch(args)
		if len(moduleMatch) != 2 {
			continue
		}
		moduleName, ok := parseTextCallLiteral(moduleMatch[1])
		if !ok || strings.TrimSpace(moduleName) == "" {
			continue
		}
		options := ""
		if idx := strings.Index(args, ","); idx >= 0 {
			options = strings.TrimSpace(args[idx+1:])
			if strings.HasPrefix(options, "{") && strings.HasSuffix(options, "}") {
				options = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(options, "{"), "}"))
			}
		}
		defaultScope := parseCreateTranslateOptions(options)

		for _, part := range strings.Split(destructure, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			propName := part
			localName := part
			if strings.Contains(part, ":") {
				segments := strings.SplitN(part, ":", 2)
				propName = strings.TrimSpace(segments[0])
				localName = strings.TrimSpace(segments[1])
			}
			propName = strings.TrimSpace(propName)
			localName = strings.TrimSpace(localName)
			if localName == "" || (propName != "_t" && propName != "_lt") {
				continue
			}
			bindings[localName] = TranslateBinding{
				Module:          strings.TrimSpace(moduleName),
				DefaultScope:    defaultScope,
				ReferenceOutput: propName == "_lt",
			}
		}
	}

	return bindings
}

func parseBalancedCallArguments(source string, openParenIndex int) (string, bool) {
	if openParenIndex < 0 || openParenIndex >= len(source) || source[openParenIndex] != '(' {
		return "", false
	}
	depth := 0
	inQuote := byte(0)
	escaped := false
	for i := openParenIndex; i < len(source); i++ {
		ch := source[i]
		if inQuote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			inQuote = ch
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return strings.TrimSpace(source[openParenIndex+1 : i]), true
			}
		}
	}
	return "", false
}

// ParseTermReferenceCall parses `_lt('literal'[, opts])` (or an `_lt` alias) into a TermReference.
func ParseTermReferenceCall(raw string, ownerModule string, binding TranslateBinding) (*meta.TermReference, bool) {
	match := termReferenceCallPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(match) != 4 || strings.TrimSpace(ownerModule) == "" {
		return nil, false
	}

	callee := strings.TrimSpace(match[1])
	referenceOutput := binding.ReferenceOutput || callee == "_lt"
	if !referenceOutput {
		return nil, false
	}

	src, srcOK := parseTextCallLiteral(match[2])
	scope := strings.TrimSpace(binding.DefaultScope)
	if strings.TrimSpace(match[3]) != "" {
		if explicitScope := strings.TrimSpace(parseFactoryStringOption(match[3], referenceScopePattern)); explicitScope != "" {
			scope = explicitScope
		} else {
			pathValue := strings.TrimSpace(parseFactoryStringOption(match[3], referencePathPattern))
			locationValue := strings.TrimSpace(parseFactoryStringOption(match[3], referenceLocationPattern))
			if pathValue != "" {
				scope = pathValue
				if locationValue != "" {
					scope += "@" + locationValue
				}
			}
		}
	}
	if !srcOK || strings.TrimSpace(scope) == "" || strings.TrimSpace(src) == "" {
		return nil, false
	}

	moduleName := strings.TrimSpace(binding.Module)
	if moduleName == "" {
		moduleName = strings.TrimSpace(ownerModule)
	}
	reference := meta.NewTermReference(moduleName, scope, src, "literal")
	return &reference, true
}

// ParseResourceTitleExpr parses a string literal or `_lt(...)` expression.
func ParseResourceTitleExpr(raw string, ownerModule string, bindings map[string]TranslateBinding) (title string, titleText *meta.TermReference, ok bool) {
	expr := strings.TrimSpace(raw)
	if expr == "" {
		return "", nil, false
	}
	if literal, err := ParseJSStringLiteral(expr); err == nil && strings.TrimSpace(literal) != "" {
		return strings.TrimSpace(literal), nil, true
	}

	match := termReferenceCallPattern.FindStringSubmatch(expr)
	if len(match) != 4 {
		return "", nil, false
	}
	binding, exists := bindings[strings.TrimSpace(match[1])]
	if !exists {
		binding = TranslateBinding{Module: ownerModule}
	}
	reference, parsed := ParseTermReferenceCall(expr, ownerModule, binding)
	if !parsed || reference == nil || strings.TrimSpace(reference.Src) == "" {
		return "", nil, false
	}
	return strings.TrimSpace(reference.Src), reference, true
}

// CloneTermReferenceWithSrc returns a copy of reference with a new src/key.
func CloneTermReferenceWithSrc(reference *meta.TermReference, src string) *meta.TermReference {
	if reference == nil {
		return nil
	}
	src = strings.TrimSpace(src)
	if src == "" {
		return nil
	}
	clone := meta.NewTermReference(reference.Module, reference.Scope, src, reference.Kind)
	return &clone
}
