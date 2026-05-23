// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueparser

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type vueParserTestScope struct {
	ctx context.Context
	cfg *config.Config
}

func (e *vueParserTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *vueParserTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *vueParserTestScope) Session() *scope.Session { return nil }
func (e *vueParserTestScope) WithContext(ctx context.Context) scope.Scope {
	return &vueParserTestScope{ctx: ctx, cfg: e.cfg}
}
func (e *vueParserTestScope) Context() context.Context { return e.ctx }
func (e *vueParserTestScope) Logger() *slog.Logger     { return slog.Default() }
func (e *vueParserTestScope) Config() *config.Config   { return e.cfg }
func (e *vueParserTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

func newVueParserTestScope() scope.Scope {
	return &vueParserTestScope{
		ctx: context.Background(),
		cfg: &config.Config{AddonsPath: "/virtual/addons"},
	}
}

func TestVueParser_ParseVueComponentWithTSGo(t *testing.T) {
	runtimeScope := newVueParserTestScope()
	module := &meta.IrModule{Path: "/virtual/addons/auth", ApplicationStr: "auth"}
	p := NewVueParser(runtimeScope, module)

	path := "/virtual/addons/auth/web/views/ChildView.vue"
	content := `<template><div /></template>
<script lang="ts">
import { defineComponent } from 'vue';
import BaseView from './BaseView.vue';
import Xpath from '@/core/web/component/xpath.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
  components: { Xpath },
});
</script>`

	r, err := p.Parse(map[string]string{"@": "/virtual/addons"}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r == nil || r.VueComponent == nil {
		t.Fatalf("expected vue component parse result")
	}

	if r.VueExtendsProperty == nil {
		t.Fatalf("expected vue extends property")
	}
	if strings.TrimSpace(r.VueExtendsProperty.Text) != "extends: BaseView" {
		t.Fatalf("unexpected extends property text: %q", r.VueExtendsProperty.Text)
	}
	if r.VueComponent.RawExtends != "/virtual/addons/auth/web/views/BaseView.vue" {
		t.Fatalf("unexpected vue raw extends: %s", r.VueComponent.RawExtends)
	}

	xpathModuleSpec, xpathReferenceIdent := meta.XpathComponentModuleSpec(runtimeScope)
	foundXpath := false
	for _, node := range r.VueComponentsPropertys {
		if node == nil {
			continue
		}
		if node.ModuleSpecPath == xpathModuleSpec && node.ReferenceIdent == xpathReferenceIdent {
			foundXpath = true
			if strings.TrimSpace(node.Name) != "" {
				t.Fatalf("expected shorthand component name to match legacy empty value, got %q", node.Name)
			}
			if node.Line <= 0 || strings.TrimSpace(node.Text) == "" {
				t.Fatalf("expected stable line/text for xpath component, got line=%d text=%q", node.Line, node.Text)
			}
		}
	}
	if !foundXpath {
		t.Fatalf("expected xpath component property in parsed vue components")
	}
}

func TestVueParser_ParseImportsExportsWithTSGoAcrossScriptBlocks(t *testing.T) {
	runtimeScope := newVueParserTestScope()
	module := &meta.IrModule{Path: "/virtual/addons/auth", ApplicationStr: "auth"}
	p := NewVueParser(runtimeScope, module)

	path := "/virtual/addons/auth/web/views/ChildView.vue"
	content := `<template><div /></template>
<script lang="ts">
import { defineComponent } from 'vue';
import BaseView from './BaseView.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
});
</script>
<script setup lang="ts">
import { ref as localRef } from 'vue';

const state = localRef(1);
void state;
</script>`

	r, err := p.Parse(map[string]string{"@": "/virtual/addons"}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if r == nil {
		t.Fatalf("expected parse result")
	}
	if len(r.Imports) < 3 {
		t.Fatalf("expected merged imports from script+setup, got %d", len(r.Imports))
	}

	if _, ok := r.Imports["defineComponent"]; !ok {
		t.Fatalf("expected defineComponent import")
	}
	if _, ok := r.Imports["BaseView"]; !ok {
		t.Fatalf("expected BaseView import")
	}
	if _, ok := r.Imports["localRef"]; !ok {
		t.Fatalf("expected localRef import from script setup")
	}

	if r.Imports["BaseView"].ModuleSpecPath != "/virtual/addons/auth/web/views/BaseView.vue" {
		t.Fatalf("unexpected BaseView module spec path: %s", r.Imports["BaseView"].ModuleSpecPath)
	}

	defaultExport := r.Exports["default"]
	if defaultExport == nil {
		t.Fatalf("expected default export")
	}
	if defaultExport.ModuleSpecPath != path {
		t.Fatalf("unexpected default export module spec path: %s", defaultExport.ModuleSpecPath)
	}
}
