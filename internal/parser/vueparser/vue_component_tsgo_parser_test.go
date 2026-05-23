// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"path/filepath"
	"strings"
	"testing"

	tsast "github.com/buke/typescript-go-internal/pkg/ast"
)

func TestParseVueComponentWithTSGoCollectsExtendsAndComponents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "component.ts")
	content := `
import { defineComponent } from 'vue'
import BaseView from './BaseView.vue'
import ChildView from './ChildView.vue'
import { NamedChild } from './named'

export default defineComponent({
	extends: BaseView,
	components: {
		ChildView,
		AliasChild: NamedChild,
		Ignored: makeChild(),
	},
})
`

	result, err := parseVueComponentWithTSGo(nil, path, content)
	if err != nil {
		t.Fatalf("parseVueComponentWithTSGo() error = %v", err)
	}
	if result.extendsProperty == nil || result.rawExtends == "" {
		t.Fatalf("expected extends property, got %#v", result)
	}
	if result.extendsProperty.ReferenceIdent != "default" || !strings.HasSuffix(result.extendsProperty.ModuleSpecPath, "/BaseView.vue") {
		t.Fatalf("unexpected extends property: %#v", result.extendsProperty)
	}
	if len(result.componentPropertys) != 2 {
		t.Fatalf("component property count = %d, want 2", len(result.componentPropertys))
	}

	var shorthandComponentFound bool
	var namedComponentFound bool
	for _, property := range result.componentPropertys {
		switch {
		case property.ValueText == "ChildView":
			shorthandComponentFound = property.ReferenceIdent == "default" && strings.HasSuffix(property.ModuleSpecPath, "/ChildView.vue")
		case property.Name == "AliasChild":
			namedComponentFound = property.ReferenceIdent == "NamedChild" && strings.HasSuffix(property.ModuleSpecPath, "/named")
		}
	}
	if !shorthandComponentFound || !namedComponentFound {
		t.Fatalf("unexpected component properties: %#v", result.componentPropertys)
	}
}

func TestIsDefineComponentCallWithOptionsSupportsNamespaceImports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "namespace_component.ts")
	content := `
import * as Vue from 'vue'

export default Vue.defineComponent({})
`

	ctx, err := parseTSGoCtx(nil, path, content)
	if err != nil {
		t.Fatalf("parseTSGoCtx() error = %v", err)
	}
	callExprNode := findFirstNodeInSourceFile(ctx.source, tsast.KindCallExpression)
	if callExprNode == nil {
		t.Fatal("expected call expression node")
	}
	expr := callExprNode.AsCallExpression().Expression
	if isDefineComponentCall(ctx, expr) {
		t.Fatal("isDefineComponentCall() should reject namespace form")
	}
	if !isDefineComponentCallWithOptions(ctx, expr, true) {
		t.Fatal("isDefineComponentCallWithOptions() should accept namespace form when enabled")
	}
}

func TestParseVueComponentWithTSGoSupportsPropertyAccessExtendsAndQuotedNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quoted_component.ts")
	content := `
import { defineComponent } from 'vue'
import * as BaseView from './BaseView'
import { NamedChild } from './named'

export default defineComponent({
	extends: BaseView.default,
	components: {
		'QuotedChild': NamedChild,
		Ignored: NamedChild.factory,
	},
})
`

	result, err := parseVueComponentWithTSGo(nil, path, content)
	if err != nil {
		t.Fatalf("parseVueComponentWithTSGo() error = %v", err)
	}
	if result.extendsProperty == nil || result.rawExtends == "" {
		t.Fatalf("expected extends property, got %#v", result)
	}
	if result.extendsProperty.ReferenceIdent != "default" || !strings.HasSuffix(result.extendsProperty.ModuleSpecPath, "/BaseView") {
		t.Fatalf("unexpected property-access extends property: %#v", result.extendsProperty)
	}
	if len(result.componentPropertys) != 1 {
		t.Fatalf("component property count = %d, want 1", len(result.componentPropertys))
	}
	component := result.componentPropertys[0]
	if component.Name != "QuotedChild" || component.ReferenceIdent != "NamedChild" || !strings.HasSuffix(component.ModuleSpecPath, "/named") {
		t.Fatalf("unexpected quoted-name component property: %#v", component)
	}
}
