// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/choysum-dev/choysum/pkg/scope"
)

// IsConventionalServiceName returns true when a service method name starts with
// an upper-case rune.
func IsConventionalServiceName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}

	r, size := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError && size == 1 {
		return false
	}

	return unicode.IsUpper(r)
}

// IsConventionalModelService applies the shared model-service convention used by
// parser/backend plugin/generator.
func IsConventionalModelService(accessibilityModifier string, isStatic bool, serviceName string) bool {
	if strings.TrimSpace(accessibilityModifier) != "public" || !isStatic {
		return false
	}

	return IsConventionalServiceName(serviceName)
}

func IsCoreModule(moduleName string) bool {
	return moduleName == "core"
}

func ModelDecoratorModuleSpec(runtimeScope scope.Scope) (string, string) {
	addonsPath := runtimeOptionsFromScope(runtimeScope).addonsPath
	return filepath.Join(addonsPath, "core", "service", "orm", "decorator", "model"), "Model"
}

func FieldDecoratorModuleSpec(runtimeScope scope.Scope) (string, string) {
	addonsPath := runtimeOptionsFromScope(runtimeScope).addonsPath
	return filepath.Join(addonsPath, "core", "service", "orm", "decorator", "field"), "Field"
}

func ServiceDecoratorModuleSpec(runtimeScope scope.Scope) (string, string) {
	addonsPath := runtimeOptionsFromScope(runtimeScope).addonsPath
	return filepath.Join(addonsPath, "core", "service", "orm", "decorator", "service"), "Service"
}

func XpathComponentModuleSpec(runtimeScope scope.Scope) (string, string) {
	addonsPath := runtimeOptionsFromScope(runtimeScope).addonsPath
	return filepath.Join(addonsPath, "core", "web", "component", "xpath.vue"), "default"
}

func BaseModelModuleSpec(runtimeScope scope.Scope) (string, string) {
	addonsPath := runtimeOptionsFromScope(runtimeScope).addonsPath

	return filepath.Join(addonsPath, "core", "service", "orm", "model", "model"), "BaseModel"
}
