// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package vue provides Vue SFC service-script codegen and diagnostic remap
// for Go-native typecheck. PR-3 uses GoldenCoder (committed goldens); QuickJS
// language-core lands in a later PR.
package vue

// HelperVirtualRoot is the stable VFS prefix for vendored language-core helpers.
const HelperVirtualRoot = "/choysum-vue-virtual/types"

// HelperTemplatePath is the rewritten reference target for template-helpers.d.ts.
const HelperTemplatePath = HelperVirtualRoot + "/template-helpers.d.ts"

// HelperPropsPath is the rewritten reference target for props-fallback.d.ts.
const HelperPropsPath = HelperVirtualRoot + "/props-fallback.d.ts"

// SpanMapping maps a generated service-script span back to the source .vue file.
type SpanMapping struct {
	SourceFile     string `json:"sourceFile"`
	SourceStart    int    `json:"sourceStart"`
	SourceEnd      int    `json:"sourceEnd"`
	GeneratedStart int    `json:"generatedStart"`
	GeneratedEnd   int    `json:"generatedEnd"`
	Verification   any    `json:"verification,omitempty"`
}

// ServiceScript is the language-core service embedded for one .vue file.
type ServiceScript struct {
	EmbeddedID string
	ScriptKind string // ts|tsx|js|jsx
	Content    string
	Mappings   []SpanMapping
}

// CodegenOptions configures CreateServiceScript.
type CodegenOptions struct {
	CurrentDirectory string
}

// Coder produces a service script for a Vue SFC.
type Coder interface {
	CreateServiceScript(path, source string, opts CodegenOptions) (ServiceScript, error)
}
