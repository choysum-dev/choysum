// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package output

import "strings"

func ModuleCommandFailureAttrs(operation string) []any {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return nil
	}
	return []any{"operation", operation}
}

func normalizedRequestedValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func CurrentOrRequestedAttr(singular string, plural string, current string, requested []string) []any {
	current = strings.TrimSpace(current)
	if current != "" {
		return []any{singular, current}
	}
	values := normalizedRequestedValues(requested)
	if len(values) == 0 {
		return nil
	}
	if len(values) == 1 {
		return []any{singular, values[0]}
	}
	return []any{plural, values}
}

func ModuleInstallFailureAttrs(input string, moduleName string) []any {
	attrs := CurrentOrRequestedAttr("input", "inputs", input, nil)
	moduleName = strings.TrimSpace(moduleName)
	input = strings.TrimSpace(input)
	if moduleName == "" || moduleName == input {
		return attrs
	}
	return append(attrs, "module", moduleName)
}
