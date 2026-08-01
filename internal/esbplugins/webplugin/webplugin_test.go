// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webplugin

import (
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestHandleParserResultsBuildsOrderedResultsAndExports(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{})
	runtimeOpts := runtimeOptionsFromScope(plugin.Env)
	firstPath := filepath.Join(runtimeOpts.modulesPath, "auth", "web", "components", "first.ts")
	indexPath := filepath.Join(runtimeOpts.modulesPath, "auth", "web", "components", "index.ts")

	plugin.ParserResultChan <- &parser.ParserResult{
		Path: firstPath,
		Exports: map[string]*parser.Export{
			"First": {ModuleSpecPath: firstPath, ReferenceIdent: "First"},
		},
	}
	plugin.ParserResultChan <- &parser.ParserResult{
		Path: indexPath,
		Exports: map[string]*parser.Export{
			"default": {ModuleSpecPath: indexPath, ReferenceIdent: "default"},
		},
	}

	results := plugin.HandleParserResults()
	if len(results) != 2 {
		t.Fatalf("expected 2 parser results, got %d", len(results))
	}
	if results[0].Path != indexPath || results[1].Path != firstPath {
		t.Fatalf("unexpected parser result order: %#v", []string{results[0].Path, results[1].Path})
	}
	indexModuleSpec := indexPath[:len(indexPath)-len(filepath.Ext(indexPath))]
	if plugin.TsExports[indexModuleSpec]["default"] == nil {
		t.Fatalf("expected ts exports for module spec %q, got %#v", indexModuleSpec, plugin.TsExports)
	}
	if plugin.TsExports[filepath.Dir(indexPath)]["default"] == nil {
		t.Fatalf("expected ts exports to alias index directory %q, got %#v", filepath.Dir(indexPath), plugin.TsExports)
	}
}

func TestGetParserResultsRewritesVueComponentReferences(t *testing.T) {
	plugin := newPluginForTest(t, fakeParser{})
	runtimeOpts := runtimeOptionsFromScope(plugin.Env)
	componentPath := filepath.Join(runtimeOpts.modulesPath, "auth", "web", "components", "base.ts")
	componentModuleSpec := componentPath[:len(componentPath)-len(filepath.Ext(componentPath))]
	consumerPath := filepath.Join(runtimeOpts.modulesPath, "auth", "web", "pages", "home.ts")

	plugin.ParserResultChan <- &parser.ParserResult{
		Path: componentPath,
		Exports: map[string]*parser.Export{
			"default":   {ModuleSpecPath: componentPath, ReferenceIdent: "default"},
			"NamedCard": {ModuleSpecPath: componentPath, ReferenceIdent: "CardView"},
		},
	}
	plugin.ParserResultChan <- &parser.ParserResult{
		Path: consumerPath,
		VueComponent: &meta.Component{
			Path: consumerPath,
		},
		VueComponentsPropertys: []*parser.PropertyNode{
			{ModuleSpecPath: componentModuleSpec, ReferenceIdent: "NamedCard"},
		},
		VueExtendsProperty: &parser.PropertyNode{ModuleSpecPath: componentModuleSpec, ReferenceIdent: "default"},
	}

	results, err := plugin.GetParserResults()
	if err != nil {
		t.Fatalf("GetParserResults() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 parser results, got %d", len(results))
	}

	var consumer *parser.ParserResult
	for _, result := range results {
		if result.Path == consumerPath {
			consumer = result
			break
		}
	}
	if consumer == nil {
		t.Fatalf("expected consumer parser result in %#v", results)
	}
	if consumer.VueComponentsPropertys[0].ModuleSpecPath != componentPath || consumer.VueComponentsPropertys[0].ReferenceIdent != "CardView" {
		t.Fatalf("unexpected component property rewrite: %#v", consumer.VueComponentsPropertys[0])
	}
	if consumer.VueExtendsProperty.ModuleSpecPath != componentPath || consumer.VueExtendsProperty.ReferenceIdent != "default" {
		t.Fatalf("unexpected extends rewrite: %#v", consumer.VueExtendsProperty)
	}
	if consumer.VueComponent.RawExtends != componentPath || consumer.VueComponent.Extends != componentPath {
		t.Fatalf("unexpected vue component extends fields: raw=%q extends=%q", consumer.VueComponent.RawExtends, consumer.VueComponent.Extends)
	}
}
