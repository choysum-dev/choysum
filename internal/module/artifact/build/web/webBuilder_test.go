// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package webmodulebuilder

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/antchfx/htmlquery"
	"github.com/choysum-dev/choysum/internal/esbplugins"
	defaultesbplugins "github.com/choysum-dev/choysum/internal/esbplugins/webprebuildplugin"
	"github.com/choysum-dev/choysum/internal/esmresolver"
	modulegenerator "github.com/choysum-dev/choysum/internal/module/artifact/generate"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/artifact/staging"
	"github.com/choysum-dev/choysum/internal/parser"
	defaultparser "github.com/choysum-dev/choysum/internal/parser/vueparser"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/internal/vueplugin"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/evanw/esbuild/pkg/api"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testScope struct {
	ctx context.Context
	cfg *config.Config
	db  *gorm.DB
	log *slog.Logger
}

type webBuilderTestTransaction struct {
	ctx     context.Context
	session *scope.Session
}

func withParserResults(result *module.BuildResult, parserResults ...*parser.ParserResult) *module.BuildResult {
	return module.WithParserResults(result, parserResults)
}

func parserResultsOf(result *module.BuildResult) []*parser.ParserResult {
	return module.ParserResults(result)
}

func (e *testScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *testScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *testScope) Session() *scope.Session {
	if e == nil {
		return nil
	}
	if tx, ok := scope.TransactionFromContext(e.ctx); ok {
		if sess := tx.Session(); sess != nil {
			return sess
		}
	}
	if sess, ok := scope.SessionFromContext(e.ctx); ok {
		return sess
	}
	if e.db == nil {
		return nil
	}
	return &scope.Session{DB: e.db}
}
func (e *testScope) WithContext(ctx context.Context) scope.Scope {
	return &testScope{ctx: ctx, cfg: e.cfg, db: e.db}
}
func (e *testScope) Context() context.Context { return e.ctx }
func (e *testScope) Logger() *slog.Logger {
	if e.log != nil {
		return e.log
	}
	return slog.Default()
}
func (e *testScope) Config() *config.Config { return e.cfg }

func (e *testScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func (tx *webBuilderTestTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *webBuilderTestTransaction) Session() *scope.Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *webBuilderTestTransaction) Savepoint(string) error           { return nil }
func (tx *webBuilderTestTransaction) RollbackToSavepoint(string) error { return nil }
func (tx *webBuilderTestTransaction) ReleaseSavepoint(string) error    { return nil }

func newTestScope() scope.Scope {
	return &testScope{
		ctx: context.Background(),
		cfg: &config.Config{ModulesPath: "/virtual/modules", DefaultChoysumPath: "/virtual/.choysum"},
	}
}

func newTestScopeWithDB(t *testing.T) scope.Scope {
	t.Helper()

	dsn := "file:web_builder_" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	return &testScope{
		ctx: context.Background(),
		cfg: &config.Config{ModulesPath: "/virtual/modules", DefaultChoysumPath: t.TempDir()},
		db:  db,
	}
}

func normalizeAbsImportPath(path string) string {
	absPath := path
	if resolved, err := filepath.Abs(path); err == nil {
		absPath = resolved
	}
	return filepath.ToSlash(filepath.Clean(absPath))
}

func mustExec(t *testing.T, db *gorm.DB, query string, args ...any) {
	t.Helper()
	if err := db.Exec(query, args...).Error; err != nil {
		t.Fatalf("exec failed: %v, query=%s", err, query)
	}
}

func parseVueResult(t *testing.T, runtimeScope scope.Scope, path string, content string) *parser.ParserResult {
	t.Helper()
	p := defaultparser.NewVueParser(runtimeScope, &meta.IrModule{Path: "/virtual/modules/test"})
	parsed, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed.VueComponent == nil {
		t.Fatalf("expected VueComponent to be parsed")
	}
	if parsed.VueExtendsProperty == nil {
		t.Fatalf("expected VueExtendsProperty to be parsed")
	}
	if parsed.RawScriptNode == nil {
		t.Fatalf("expected RawScriptNode to be parsed")
	}
	return parsed
}

type buildTestPlugin struct {
	parserResults     []*parser.ParserResult
	storedResults     []*parser.ParserResult
	getErr            error
	definePlugins     []api.Plugin
	defineCalls       int
	entryPointImports []string
	indexHtmlOutFile  string
	writeIndexHTML    bool
}

type fixedParser struct {
	result *parser.ParserResult
	err    error
}

func (p *buildTestPlugin) DefinePlugins(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.IrModule, options ...esbplugins.EsbPluginOptions) []api.Plugin {
	p.defineCalls++
	for _, opt := range options {
		opt(p)
	}
	plugins := append([]api.Plugin(nil), p.definePlugins...)
	if p.writeIndexHTML {
		plugins = append(plugins, api.Plugin{
			Name: "write-index-html",
			Setup: func(build api.PluginBuild) {
				build.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
					if strings.TrimSpace(p.indexHtmlOutFile) == "" {
						return api.OnEndResult{}, nil
					}
					if err := os.MkdirAll(filepath.Dir(p.indexHtmlOutFile), 0o755); err != nil {
						return api.OnEndResult{}, err
					}
					if err := os.WriteFile(p.indexHtmlOutFile, []byte("<html><body>ok</body></html>"), 0o644); err != nil {
						return api.OnEndResult{}, err
					}
					return api.OnEndResult{}, nil
				})
			},
		})
	}
	return plugins
}

func (p *buildTestPlugin) GetParserResults() ([]*parser.ParserResult, error) {
	if p.getErr != nil {
		return nil, p.getErr
	}
	if p.parserResults != nil {
		return p.parserResults, nil
	}
	return p.storedResults, nil
}

func (p *buildTestPlugin) SetParserResults(parserResults []*parser.ParserResult) error {
	p.storedResults = parserResults
	return nil
}

func (p *buildTestPlugin) SetEntryPointImports(imports []string) {
	p.entryPointImports = append([]string(nil), imports...)
}

func (p *buildTestPlugin) SetIndexHtmlOutFile(outFile string) {
	p.indexHtmlOutFile = outFile
}

func (p fixedParser) Parse(pathAlias map[string]string, path string, content string) (*parser.ParserResult, error) {
	return p.result, p.err
}

func setupBuildPipelineTestFiles(t *testing.T, testRuntimeScope *testScope, entryContent string) (*meta.IrModule, string) {
	t.Helper()

	root := t.TempDir()
	modulesPath := filepath.Join(root, "modules")
	modulePath := filepath.Join(modulesPath, "auth")
	webPath := filepath.Join(modulePath, "web")
	entryPoint := filepath.Join(webPath, "index.ts")

	for _, dir := range []string{modulesPath, webPath, filepath.Join(root, "dist")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s failed: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(modulesPath, "tsconfig.json"), []byte(`{
	"compilerOptions": {
		"target": "ES2020",
		"module": "ESNext",
		"moduleResolution": "bundler"
	}
}`), 0o644); err != nil {
		t.Fatalf("write tsconfig failed: %v", err)
	}
	if err := os.WriteFile(entryPoint, []byte(entryContent), 0o644); err != nil {
		t.Fatalf("write entry point failed: %v", err)
	}

	testRuntimeScope.cfg.ModulesPath = modulesPath
	testRuntimeScope.cfg.DistPath = filepath.Join(root, "dist")
	testRuntimeScope.cfg.Compile = config.NewDefaultCompileConfig()
	testRuntimeScope.cfg.Server = config.NewDefaultServerConfig()
	testRuntimeScope.cfg.FrontendEnv = map[string]any{"CHOYSUM_APP_NAME": "build-pipeline-test"}

	return &meta.IrModule{Name: "auth", Path: modulePath}, entryPoint
}

func TestGetScriptNode_RewritesExtendsImport_SingleLineImport(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	path := "/virtual/modules/test/web/views/ChildView.vue"
	newExtendsPath := "/virtual/modules/ext/web/views/BaseView.vue"

	sfc := `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from './BaseView.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
  setup(props, ctx) {
    const baseSetup = BaseView?.setup?.(props, ctx) || {};
    return {
      ...baseSetup,
    };
  },
});
</script>`

	parsed := parseVueResult(t, testRuntimeScope, path, sfc)
	oldExtendsPath := parsed.VueComponent.RawExtends
	if oldExtendsPath == "" {
		t.Fatalf("expected old extends path to be parsed")
	}
	parsed.VueComponent.Extends = newExtendsPath

	scriptNode, err := b.getScriptNode(parsed, nil)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)

	if !strings.Contains(content, "extends: BaseView") {
		t.Fatalf("expected extends identifier to remain BaseView, got:\n%s", content)
	}
	if !strings.Contains(content, "BaseView?.setup?.(props, ctx)") {
		t.Fatalf("expected setup to keep BaseView parent call, got:\n%s", content)
	}
	if !strings.Contains(content, "import BaseView from '"+newExtendsPath+"'") {
		t.Fatalf("expected import path to be rewritten from %s to %s, got:\n%s", oldExtendsPath, newExtendsPath, content)
	}
}

func TestGetScriptNode_RewritesExtendsImport_MultiLineImport(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	path := "/virtual/modules/test/web/views/ChildView.vue"
	newExtendsPath := filepath.Join("/virtual/modules/ext/web/views", "BaseView.vue")

	sfc := `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from
	'./BaseView.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
  setup(props, ctx) {
    const baseSetup = BaseView?.setup?.(props, ctx) || {};
    return {
      ...baseSetup,
    };
  },
});
</script>`

	parsed := parseVueResult(t, testRuntimeScope, path, sfc)
	if parsed.VueComponent.RawExtends == "" {
		t.Fatalf("expected old extends path to be parsed")
	}
	parsed.VueComponent.Extends = newExtendsPath

	scriptNode, err := b.getScriptNode(parsed, nil)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)

	if !strings.Contains(content, "extends: BaseView") {
		t.Fatalf("expected extends identifier to remain BaseView, got:\n%s", content)
	}
	if !strings.Contains(content, "BaseView?.setup?.(props, ctx)") {
		t.Fatalf("expected setup to keep BaseView parent call, got:\n%s", content)
	}
	if !strings.Contains(content, "from '"+newExtendsPath+"'") {
		t.Fatalf("expected multiline import from path to be rewritten to %s, got:\n%s", newExtendsPath, content)
	}
}

func TestGetScriptNode_MultiLineExtendsImport_WithXPathReplacement(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	childPath := "/virtual/modules/test/web/views/ChildView.vue"
	parentPath := "/virtual/modules/test/web/views/BaseView.vue"
	newExtendsPath := "/virtual/modules/ext/web/views/BaseView.vue"

	childSFC := `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from
	'./BaseView.vue';
import Xpath from '/virtual/modules/core/web/component/xpath.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
  components: { Xpath },
});
</script>`

	parentSFC := `<template><div><ParentBadge /></div></template>
<script lang="ts" _name="BaseView">
import { defineComponent } from 'vue';
import ParentBadge from './ParentBadge.vue';
import LegacyOnly from './LegacyOnly.vue';

export default defineComponent({
  name: 'BaseView',
	extends: LegacyOnly,
  components: { ParentBadge },
});
</script>`

	childParsed := parseVueResult(t, testRuntimeScope, childPath, childSFC)
	parentParsed := parseVueResult(t, testRuntimeScope, parentPath, parentSFC)
	childParsed.VueComponent.Extends = newExtendsPath

	scriptNode, err := b.getScriptNode(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)

	if !strings.Contains(content, "from '"+newExtendsPath+"'") {
		t.Fatalf("expected multiline extends import to be rewritten, got:\n%s", content)
	}
	if strings.Contains(content, "components: { Xpath }") || strings.Contains(content, "components: {Xpath}") {
		t.Fatalf("expected xpath placeholder to be replaced, got:\n%s", content)
	}
	if !(strings.Contains(content, "components: {ParentBadge") || strings.Contains(content, "components: { ParentBadge")) {
		t.Fatalf("expected parent component to replace xpath placeholder, got:\n%s", content)
	}
	if !strings.Contains(content, "import ParentBadge from '/virtual/modules/test/web/views/ParentBadge.vue';") {
		t.Fatalf("expected missing parent import to be appended, got:\n%s", content)
	}
}

func TestGetScriptNode_RewritesExtendsImport_WithStableFieldsOnly(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	path := "/virtual/modules/test/web/views/ChildView.vue"
	newExtendsPath := "/virtual/modules/ext/web/views/BaseView.vue"

	sfc := `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from './BaseView.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
});
</script>`

	parsed := parseVueResult(t, testRuntimeScope, path, sfc)
	parsed.VueComponent.Extends = newExtendsPath

	// Legacy AstNode fields were removed; stable fields must be enough.

	scriptNode, err := b.getScriptNode(parsed, nil)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)

	if !strings.Contains(content, "extends: BaseView") {
		t.Fatalf("expected extends identifier to remain BaseView, got:\n%s", content)
	}
	if !strings.Contains(content, "import BaseView from '"+newExtendsPath+"'") {
		t.Fatalf("expected import path to be rewritten with stable fields, got:\n%s", content)
	}
}

func TestGetScriptNode_MultipleRewrites_WithMultibyteContent(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	childPath := "/virtual/modules/test/web/views/ChildView.vue"
	parentPath := "/virtual/modules/test/web/views/BaseView.vue"
	newExtendsPath := "/virtual/modules/ext/web/views/BaseView.vue"

	childSFC := `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from './BaseView.vue';
import Xpath from '/virtual/modules/core/web/component/xpath.vue';

// emoji😀 keep offset
export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
  components: { Xpath },
});
</script>`

	parentSFC := `<template><div><ParentBadge /></div></template>
<script lang="ts" _name="BaseView">
import { defineComponent } from 'vue';
import ParentBadge from './ParentBadge.vue';
import LegacyOnly from './LegacyOnly.vue';

export default defineComponent({
  name: 'BaseView',
  extends: LegacyOnly,
  components: { ParentBadge },
});
</script>`

	childParsed := parseVueResult(t, testRuntimeScope, childPath, childSFC)
	parentParsed := parseVueResult(t, testRuntimeScope, parentPath, parentSFC)
	childParsed.VueComponent.Extends = newExtendsPath

	scriptNode, err := b.getScriptNode(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)

	if !strings.Contains(content, "// emoji😀 keep offset") {
		t.Fatalf("expected multibyte content to remain unchanged, got:\n%s", content)
	}
	if !strings.Contains(content, "extends: BaseView") {
		t.Fatalf("expected extends identifier to remain BaseView, got:\n%s", content)
	}
	if !strings.Contains(content, "import BaseView from '"+newExtendsPath+"'") {
		t.Fatalf("expected extends import path to be rewritten, got:\n%s", content)
	}
	if strings.Contains(content, "./BaseView.vue") {
		t.Fatalf("expected old extends import path to be removed, got:\n%s", content)
	}
	if !(strings.Contains(content, "components: {ParentBadge") || strings.Contains(content, "components: { ParentBadge")) {
		t.Fatalf("expected xpath component placeholder to be replaced by parent template imports, got:\n%s", content)
	}
	if !strings.Contains(content, "import ParentBadge from '/virtual/modules/test/web/views/ParentBadge.vue';") {
		t.Fatalf("expected missing parent component import to be appended, got:\n%s", content)
	}
}

func TestGetScriptNode_MultipleRewrites_DeterministicAcrossRuns(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	childPath := "/virtual/modules/test/web/views/ChildView.vue"
	parentPath := "/virtual/modules/test/web/views/BaseView.vue"
	newExtendsPath := "/virtual/modules/ext/web/views/BaseView.vue"

	childSFC := `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from './BaseView.vue';
import Xpath from '/virtual/modules/core/web/component/xpath.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
  components: { Xpath },
});
</script>`

	parentSFC := `<template><div><ParentBadge /><AncillaryPanel /><AlphaWidget /><HelperWidget /></div></template>
<script lang="ts" _name="BaseView">
import { defineComponent } from 'vue';
import ParentBadge from './ParentBadge.vue';
import AncillaryPanel from '../panel/AncillaryPanel.vue';
import { HelperWidget, AlphaWidget } from './widgets';
import LegacyOnly from './LegacyOnly.vue';

export default defineComponent({
  name: 'BaseView',
  extends: LegacyOnly,
  components: { ParentBadge, AncillaryPanel, HelperWidget, AlphaWidget },
});
</script>`

	buildOutput := func() string {
		childParsed := parseVueResult(t, testRuntimeScope, childPath, childSFC)
		parentParsed := parseVueResult(t, testRuntimeScope, parentPath, parentSFC)
		childParsed.VueComponent.Extends = newExtendsPath

		scriptNode, err := b.getScriptNode(childParsed, parentParsed)
		if err != nil {
			t.Fatalf("getScriptNode failed: %v", err)
		}
		return htmlquery.InnerText(scriptNode)
	}

	first := buildOutput()
	second := buildOutput()

	if first != second {
		t.Fatalf("expected deterministic output across repeated runs\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	if idxA, idxB := strings.Index(first, "AlphaWidget"), strings.Index(first, "AncillaryPanel"); idxA < 0 || idxB < 0 || idxA > idxB {
		t.Fatalf("expected stable sorted component order in xpath replacement, got:\n%s", first)
	}

	if !strings.Contains(first, "import AncillaryPanel from '/virtual/modules/test/web/panel/AncillaryPanel.vue';") {
		t.Fatalf("expected AncillaryPanel import to be appended, got:\n%s", first)
	}
	if !strings.Contains(first, "import ParentBadge from '/virtual/modules/test/web/views/ParentBadge.vue';") {
		t.Fatalf("expected ParentBadge import to be appended, got:\n%s", first)
	}
	if !strings.Contains(first, "import { AlphaWidget, HelperWidget } from '/virtual/modules/test/web/views/widgets';") {
		t.Fatalf("expected named imports to be sorted deterministically, got:\n%s", first)
	}

	ancillaryImportIdx := strings.Index(first, "import AncillaryPanel from '/virtual/modules/test/web/panel/AncillaryPanel.vue';")
	parentImportIdx := strings.Index(first, "import ParentBadge from '/virtual/modules/test/web/views/ParentBadge.vue';")
	widgetsImportIdx := strings.Index(first, "import { AlphaWidget, HelperWidget } from '/virtual/modules/test/web/views/widgets';")
	if ancillaryImportIdx < 0 || parentImportIdx < 0 || widgetsImportIdx < 0 || !(ancillaryImportIdx < parentImportIdx && parentImportIdx < widgetsImportIdx) {
		t.Fatalf("expected deterministic import group order by module path, got:\n%s", first)
	}
}

func TestGetTemplateImportComponents_CollectsScriptSetupTemplateImports(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	parentPath := "/virtual/modules/test/web/components/layout/OHeader.vue"
	parentSFC := `<template>
  <div>
    <el-icon><QuestionFilled /></el-icon>
  </div>
</template>
<script setup lang="ts">
import { QuestionFilled } from '@element-plus/icons-vue';
</script>`

	parentParsed, err := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/test"}).Parse(map[string]string{"@": "/virtual/modules"}, parentPath, parentSFC)
	if err != nil {
		t.Fatalf("parse parent failed: %v", err)
	}

	imports, err := b.getTemplateImportComponents(parentParsed)
	if err != nil {
		t.Fatalf("getTemplateImportComponents failed: %v", err)
	}

	if _, ok := imports["QuestionFilled"]; !ok {
		t.Fatalf("expected QuestionFilled to be collected from script setup template usage, got %+v", imports)
	}
}

func TestGetScriptNode_AppendsScriptSetupParentTemplateIconImports(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	childPath := "/virtual/modules/test/web/components/layout/OHeader.vue"
	parentPath := "/virtual/modules/web/web/components/layout/OHeader.vue"

	childSFC := `<template>
  <Xpath expr="//div[@class='o-header__actions-primary']" position="inside" />
</template>
<script lang="ts" _name="OHeader">
import { defineComponent } from 'vue';
import { Xpath } from '@/core/web';
import OHeader from '@/web/web/components/layout/OHeader.vue';

export default defineComponent({
  extends: OHeader,
  components: { Xpath },
});
</script>`

	parentSFC := `<template>
  <div class="o-header__actions-primary">
    <el-icon><QuestionFilled /></el-icon>
  </div>
</template>
<script setup lang="ts">
import { QuestionFilled } from '@element-plus/icons-vue';
</script>`

	childParsed := parseVueResult(t, testRuntimeScope, childPath, childSFC)
	parentParsed, err := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/web"}).Parse(map[string]string{"@": "/virtual/modules"}, parentPath, parentSFC)
	if err != nil {
		t.Fatalf("parse parent failed: %v", err)
	}

	scriptNode, err := b.getScriptNode(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)

	if !strings.Contains(content, "QuestionFilled") || strings.Contains(content, "components: { Xpath }") {
		t.Fatalf("expected xpath placeholder to be replaced by parent script-setup icon import, got:\n%s", content)
	}
	if !strings.Contains(content, "import { QuestionFilled } from") || !strings.Contains(content, "icons-vue") {
		t.Fatalf("expected QuestionFilled import to be appended, got:\n%s", content)
	}
}

func TestGetScriptNode_AppendsQuestionFilledImport_ForRealAuthOHeader(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	childPath := "/virtual/modules/auth/web/components/layout/OHeader.vue"
	parentPath := "/virtual/modules/web/web/components/layout/OHeader.vue"

	repoRoot, err := findRepoRootFromWD()
	if err != nil {
		t.Fatalf("locate repo root failed: %v", err)
	}

	childContent, err := os.ReadFile(filepath.Join(repoRoot, "modules", "auth", "web", "components", "layout", "OHeader.vue"))
	if err != nil {
		t.Fatalf("read child OHeader failed: %v", err)
	}
	parentContent, err := os.ReadFile(filepath.Join(repoRoot, "modules", "web", "web", "components", "layout", "OHeader.vue"))
	if err != nil {
		t.Fatalf("read parent OHeader failed: %v", err)
	}

	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})
	alias := map[string]string{"@": "/virtual/modules"}

	childParsed, err := p.Parse(alias, childPath, string(childContent))
	if err != nil {
		t.Fatalf("parse child OHeader failed: %v", err)
	}
	parentParsed, err := p.Parse(alias, parentPath, string(parentContent))
	if err != nil {
		t.Fatalf("parse parent OHeader failed: %v", err)
	}

	if childParsed == nil || childParsed.VueComponent == nil {
		t.Fatalf("expected child VueComponent")
	}
	if parentParsed == nil || parentParsed.VueComponent == nil {
		t.Fatalf("expected parent VueComponent")
	}

	childParsed.VueComponent.Extends = parentPath

	scriptNode, err := b.getScriptNode(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)
	var firstComp any
	if len(childParsed.VueComponentsPropertys) > 0 && childParsed.VueComponentsPropertys[0] != nil {
		firstComp = *childParsed.VueComponentsPropertys[0]
	}

	if !strings.Contains(content, "QuestionFilled") {
		t.Fatalf("expected merged script to include QuestionFilled component, firstComp=%#v imports=%+v got:\n%s", firstComp, childParsed.Imports, content)
	}
	if !strings.Contains(content, "icons-vue") {
		t.Fatalf("expected merged script to include icons-vue import for QuestionFilled, got:\n%s", content)
	}
}

func TestGetScriptNode_AppendsQuestionFilledImport_WithRelativeModulesPath(t *testing.T) {
	repoRoot, err := findRepoRootFromWD()
	if err != nil {
		t.Fatalf("locate repo root failed: %v", err)
	}

	testRuntimeScope := &testScope{
		ctx: context.Background(),
		cfg: &config.Config{ModulesPath: filepath.Join(repoRoot, "modules")},
	}
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	childPath := filepath.Join(repoRoot, "modules", "auth", "web", "components", "layout", "OHeader.vue")
	parentPath := filepath.Join(repoRoot, "modules", "web", "web", "components", "layout", "OHeader.vue")

	childContent, err := os.ReadFile(childPath)
	if err != nil {
		t.Fatalf("read child OHeader failed: %v", err)
	}
	parentContent, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatalf("read parent OHeader failed: %v", err)
	}

	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: filepath.Join(repoRoot, "modules", "auth")})
	alias := map[string]string{"@": filepath.Join(repoRoot, "modules")}

	childParsed, err := p.Parse(alias, childPath, string(childContent))
	if err != nil {
		t.Fatalf("parse child OHeader failed: %v", err)
	}
	parentParsed, err := p.Parse(alias, parentPath, string(parentContent))
	if err != nil {
		t.Fatalf("parse parent OHeader failed: %v", err)
	}

	if childParsed == nil || childParsed.VueComponent == nil {
		t.Fatalf("expected child VueComponent")
	}
	childParsed.VueComponent.Extends = parentPath

	scriptNode, err := b.getScriptNode(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)

	if !strings.Contains(content, "QuestionFilled") {
		t.Fatalf("expected merged script to include QuestionFilled component, got:\n%s", content)
	}
	if !strings.Contains(content, "icons-vue") {
		t.Fatalf("expected merged script to include icons-vue import for QuestionFilled, got:\n%s", content)
	}
}

func TestGetScriptNode_AppendsQuestionFilledImport_ResolvesAliasViaTsconfig(t *testing.T) {
	repoRoot, err := findRepoRootFromWD()
	if err != nil {
		t.Fatalf("locate repo root failed: %v", err)
	}

	testRuntimeScope := &testScope{
		ctx: context.Background(),
		cfg: &config.Config{ModulesPath: "./modules"},
	}
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	childPath := filepath.Join(repoRoot, "modules", "auth", "web", "components", "layout", "OHeader.vue")
	parentPath := filepath.Join(repoRoot, "modules", "web", "web", "components", "layout", "OHeader.vue")

	childContent, err := os.ReadFile(childPath)
	if err != nil {
		t.Fatalf("read child OHeader failed: %v", err)
	}
	parentContent, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatalf("read parent OHeader failed: %v", err)
	}

	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: filepath.Join(repoRoot, "modules", "auth")})

	// Intentionally do not pass runtime path aliases; getScriptNode should resolve
	// '@/core/web' using ParseTsconfigPathAlias + ApplyPathAlias.
	childParsed, err := p.Parse(map[string]string{}, childPath, string(childContent))
	if err != nil {
		t.Fatalf("parse child OHeader failed: %v", err)
	}
	parentParsed, err := p.Parse(map[string]string{}, parentPath, string(parentContent))
	if err != nil {
		t.Fatalf("parse parent OHeader failed: %v", err)
	}

	if childParsed == nil || childParsed.VueComponent == nil {
		t.Fatalf("expected child VueComponent")
	}
	childParsed.VueComponent.Extends = parentPath

	scriptNode, err := b.getScriptNode(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)

	if !strings.Contains(content, "QuestionFilled") {
		t.Fatalf("expected merged script to include QuestionFilled component, got:\n%s", content)
	}
	if !strings.Contains(content, "icons-vue") {
		t.Fatalf("expected merged script to include icons-vue import for QuestionFilled, got:\n%s", content)
	}
}

func TestGetScriptNode_AppendsQuestionFilledImport_WithRuntimeTsconfigAliasMap(t *testing.T) {
	repoRoot, err := findRepoRootFromWD()
	if err != nil {
		t.Fatalf("locate repo root failed: %v", err)
	}

	testRuntimeScope := &testScope{
		ctx: context.Background(),
		cfg: &config.Config{ModulesPath: "./modules"},
	}
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	childPath := filepath.Join(repoRoot, "modules", "auth", "web", "components", "layout", "OHeader.vue")
	parentPath := filepath.Join(repoRoot, "modules", "web", "web", "components", "layout", "OHeader.vue")

	childContent, err := os.ReadFile(childPath)
	if err != nil {
		t.Fatalf("read child OHeader failed: %v", err)
	}
	parentContent, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatalf("read parent OHeader failed: %v", err)
	}

	tsconfigPath := filepath.Join(repoRoot, "modules", "tsconfig.json")
	if err := esmresolver.UpdateTsconfigPaths(tsconfigPath, nil); err != nil {
		t.Fatalf("ensure modules tsconfig failed: %v", err)
	}

	pathAlias, err := parser.ParseTsconfigPathAlias(&api.BuildOptions{Tsconfig: tsconfigPath})
	if err != nil {
		t.Fatalf("parse tsconfig path alias failed: %v", err)
	}

	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: filepath.Join(repoRoot, "modules", "auth")})
	childParsed, err := p.Parse(pathAlias, childPath, string(childContent))
	if err != nil {
		t.Fatalf("parse child OHeader failed: %v", err)
	}
	parentParsed, err := p.Parse(pathAlias, parentPath, string(parentContent))
	if err != nil {
		t.Fatalf("parse parent OHeader failed: %v", err)
	}

	if childParsed == nil || childParsed.VueComponent == nil {
		t.Fatalf("expected child VueComponent")
	}
	childParsed.VueComponent.Extends = parentPath

	scriptNode, err := b.getScriptNode(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)

	if !strings.Contains(content, "QuestionFilled") {
		t.Fatalf("expected merged script to include QuestionFilled component, got:\n%s", content)
	}
	if !strings.Contains(content, "icons-vue") {
		t.Fatalf("expected merged script to include icons-vue import for QuestionFilled, got:\n%s", content)
	}
}

func TestUpdateComponent_InjectsQuestionFilled_ForRealAuthOHeader(t *testing.T) {
	repoRoot, err := findRepoRootFromWD()
	if err != nil {
		t.Fatalf("locate repo root failed: %v", err)
	}

	modulesPath := filepath.Join(repoRoot, "modules")
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	testRuntimeScope.cfg.ModulesPath = modulesPath
	testRuntimeScope.cfg.DefaultChoysumPath = filepath.Join(repoRoot, ".choysum")
	if err := testRuntimeScope.db.AutoMigrate(&meta.IrComponent{}); err != nil {
		t.Fatalf("auto migrate components failed: %v", err)
	}
	mod := &meta.IrModule{Path: filepath.Join(modulesPath, "auth")}
	b := &WebModuleBuilder{
		runtimeScope: testRuntimeScope,
		module:       mod,
		parser:       defaultparser.NewVueParser(testRuntimeScope, mod),
	}

	tsconfigPath := filepath.Join(modulesPath, "tsconfig.json")
	if err := esmresolver.UpdateTsconfigPaths(tsconfigPath, nil); err != nil {
		t.Fatalf("ensure modules tsconfig failed: %v", err)
	}

	pathAlias, err := parser.ParseTsconfigPathAlias(&api.BuildOptions{Tsconfig: tsconfigPath})
	if err != nil {
		t.Fatalf("parse tsconfig alias failed: %v", err)
	}

	childPath := filepath.Join(modulesPath, "auth", "web", "components", "layout", "OHeader.vue")
	parentPath := filepath.Join(modulesPath, "web", "web", "components", "layout", "OHeader.vue")

	childContentBytes, err := os.ReadFile(childPath)
	if err != nil {
		t.Fatalf("read child OHeader failed: %v", err)
	}
	parentContentBytes, err := os.ReadFile(parentPath)
	if err != nil {
		t.Fatalf("read parent OHeader failed: %v", err)
	}

	childContent := vueplugin.ResolveVueStylePath(string(childContentBytes), childPath, pathAlias)
	parentContent := vueplugin.ResolveVueStylePath(string(parentContentBytes), parentPath, pathAlias)

	childParsed, err := b.parser.Parse(pathAlias, childPath, childContent)
	if err != nil {
		t.Fatalf("parse child OHeader failed: %v", err)
	}
	parentParsed, err := b.parser.Parse(pathAlias, parentPath, parentContent)
	if err != nil {
		t.Fatalf("parse parent OHeader failed: %v", err)
	}

	buildResult := withParserResults(&module.BuildResult{}, childParsed, parentParsed)
	if err := b.updateComponent(buildResult, pathAlias, childPath); err != nil {
		t.Fatalf("updateComponent failed: %v", err)
	}

	if childParsed.RawScriptNode == nil {
		t.Fatal("expected child raw script node after update")
	}
	scriptText := htmlquery.InnerText(childParsed.RawScriptNode)
	if !strings.Contains(scriptText, "QuestionFilled") {
		t.Fatalf("expected merged child script to include QuestionFilled, got:\n%s", scriptText)
	}
	if !strings.Contains(scriptText, "icons-vue") {
		t.Fatalf("expected merged child script to include icons-vue import, got:\n%s", scriptText)
	}

	foundQuestionFilledComp := false
	componentNames := make([]string, 0, len(childParsed.VueComponentsPropertys))
	for _, node := range childParsed.VueComponentsPropertys {
		if node == nil {
			continue
		}
		componentNames = append(componentNames, strings.TrimSpace(node.ValueText))
		if strings.TrimSpace(node.Name) == "QuestionFilled" || strings.TrimSpace(node.ValueText) == "QuestionFilled" {
			foundQuestionFilledComp = true
			break
		}
	}
	if !foundQuestionFilledComp {
		t.Fatalf("expected merged child components to include QuestionFilled, got names=%v script:\n%s", componentNames, scriptText)
	}
}

func TestPrebuildUpdatePrebuildResult_RealAuthOHeaderContainsInjectedQuestionFilled(t *testing.T) {
	repoRoot, err := findRepoRootFromWD()
	if err != nil {
		t.Fatalf("locate repo root failed: %v", err)
	}

	modulesPath := filepath.Join(repoRoot, "modules")
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	testRuntimeScope.cfg.ModulesPath = modulesPath
	testRuntimeScope.cfg.DistPath = filepath.Join(repoRoot, ".choysum", "dist")
	testRuntimeScope.cfg.DefaultChoysumPath = filepath.Join(repoRoot, ".choysum")
	testRuntimeScope.cfg.Server = &config.ServerConfig{WebBaseURL: "/web"}
	testRuntimeScope.cfg.Compile = config.NewDefaultCompileConfig()
	testRuntimeScope.cfg.FrontendEnv = map[string]any{}
	if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrComponent{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	moduleRef := &meta.IrModule{Name: "auth", Path: filepath.Join(modulesPath, "auth")}
	entryPoint := filepath.Join(modulesPath, "auth", "web", "index.ts")
	builder, ok := NewWebBuilder(testRuntimeScope, nil, moduleRef, entryPoint).(*WebModuleBuilder)
	if !ok {
		t.Fatal("expected NewWebBuilder to return *WebModuleBuilder")
	}
	builder.buildPlugin = &buildTestPlugin{}

	prebuildResult, err := builder.prebuild()
	if err != nil {
		t.Fatalf("prebuild failed: %v", err)
	}

	childPath := filepath.Join(modulesPath, "auth", "web", "components", "layout", "OHeader.vue")
	var beforeChild *parser.ParserResult
	for _, r := range parserResultsOf(prebuildResult) {
		if r != nil && r.Path == childPath {
			beforeChild = r
			break
		}
	}
	if beforeChild == nil {
		t.Fatalf("expected prebuild result for %s", childPath)
	}

	if err := builder.updatePrebuildResult(prebuildResult); err != nil {
		t.Fatalf("updatePrebuildResult failed: %v", err)
	}

	var childResult *parser.ParserResult
	for _, r := range parserResultsOf(prebuildResult) {
		if r != nil && r.Path == childPath {
			childResult = r
			break
		}
	}
	if childResult == nil {
		t.Fatalf("expected prebuild result for %s", childPath)
	}
	if strings.TrimSpace(childResult.Content) == "" {
		t.Fatalf("expected merged content for %s", childPath)
	}

	if !strings.Contains(childResult.Content, "QuestionFilled") {
		t.Fatalf("expected merged content to include QuestionFilled, got:\n%s", childResult.Content)
	}
	if strings.Contains(childResult.Content, "components: {\n    Xpath,") || strings.Contains(childResult.Content, "components: { Xpath") {
		t.Fatalf("expected xpath placeholder to be replaced in merged content, got:\n%s", childResult.Content)
	}
}

func TestGetScriptNode_RecognizesCoreWebDefaultImportFallback(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	childParsed := parseVueResult(t, testRuntimeScope, "/virtual/modules/test/web/views/ChildView.vue", `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from './BaseView.vue';
import Xpath from '/virtual/modules/core/web/index';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
  components: { Xpath },
});
</script>`)
	parentParsed := parseVueResult(t, testRuntimeScope, "/virtual/modules/test/web/views/BaseView.vue", `<template><div><ParentBadge /></div></template>
<script lang="ts" _name="BaseView">
import { defineComponent } from 'vue';
import ParentBadge from './ParentBadge.vue';
import LegacyOnly from './LegacyOnly.vue';

export default defineComponent({
  name: 'BaseView',
  extends: LegacyOnly,
  components: { ParentBadge },
});
</script>`)

	scriptNode, err := b.getScriptNode(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	content := htmlquery.InnerText(scriptNode)

	if strings.Contains(content, "components: { Xpath }") || strings.Contains(content, "components: {Xpath}") {
		t.Fatalf("expected fallback xpath placeholder to be replaced, got:\n%s", content)
	}
	if !strings.Contains(content, "ParentBadge") {
		t.Fatalf("expected parent template import to be injected, got:\n%s", content)
	}
	if !strings.Contains(content, "import ParentBadge from '/virtual/modules/test/web/views/ParentBadge.vue';") {
		t.Fatalf("expected parent import to be appended, got:\n%s", content)
	}
}

func TestGetScriptNode_ErrorPaths(t *testing.T) {
	t.Run("returns replace error when xpath fallback cannot locate symbol", func(t *testing.T) {
		testRuntimeScope := newTestScope()
		b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

		childParsed := parseVueResult(t, testRuntimeScope, "/virtual/modules/test/web/views/ChildView.vue", `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from './BaseView.vue';
import Xpath from '/virtual/modules/core/web/component/xpath.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
  components: { Xpath },
});
</script>`)
		parentParsed := parseVueResult(t, testRuntimeScope, "/virtual/modules/test/web/views/BaseView.vue", `<template><div><ParentBadge /></div></template>
<script lang="ts" _name="BaseView">
import { defineComponent } from 'vue';
import ParentBadge from './ParentBadge.vue';

export default defineComponent({
  name: 'BaseView',
  extends: ParentBadge,
  components: { ParentBadge },
});
</script>`)

		childParsed.VueComponentsPropertys = []*parser.PropertyNode{{
			ModuleSpecPath: "/virtual/modules/core/web/component/xpath.vue",
			ReferenceIdent: "Xpath",
			ValueText:      "MissingXpathSymbol",
			Start:          -1,
			End:            -1,
		}}

		if _, err := b.getScriptNode(childParsed, parentParsed); err == nil || !strings.Contains(err.Error(), "failed to replace xpath components for symbol MissingXpathSymbol") {
			t.Fatalf("expected xpath replacement failure, got %v", err)
		}
	})

	t.Run("returns extends rewrite error when no matching import can be found", func(t *testing.T) {
		testRuntimeScope := newTestScope()
		b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

		parsed := parseVueResult(t, testRuntimeScope, "/virtual/modules/test/web/views/ChildView.vue", `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from './BaseView.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
});
</script>`)
		parsed.VueComponent.Extends = "/virtual/modules/ext/web/views/BaseView.vue"
		parsed.Imports = map[string]*parser.Import{}
		parsed.VueExtendsProperty.ValueText = "MissingRef.setup"

		if _, err := b.getScriptNode(parsed, nil); err == nil || !strings.Contains(err.Error(), "failed to rewrite extends import path") {
			t.Fatalf("expected extends rewrite failure, got %v", err)
		}
	})
}

func findRepoRootFromWD() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	cur := wd
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("go.mod not found from working directory %s", wd)
		}
		cur = parent
	}
}

func TestWebRuntime_NoLegacyPermissionAnyOfKeys(t *testing.T) {
	repoRoot, err := findRepoRootFromWD()
	if err != nil {
		t.Fatalf("locate repo root failed: %v", err)
	}
	webRoot := filepath.Join(repoRoot, "modules")
	legacyAnyOfPattern := regexp.MustCompile(`(?s)permission\s*:\s*\{[^\}]*?anyOf\s*:\s*\[[^\]]*(rpc:/|role:)`)

	var violations []string
	err = filepath.WalkDir(webRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "tests" || name == "test" || name == "e2e" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		clean := filepath.ToSlash(path)
		if !strings.Contains(clean, "/web/") {
			return nil
		}
		if strings.HasSuffix(clean, ".test.ts") || strings.HasSuffix(clean, ".spec.ts") {
			return nil
		}
		if !(strings.HasSuffix(clean, ".ts") || strings.HasSuffix(clean, ".tsx") || strings.HasSuffix(clean, ".vue")) {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if legacyAnyOfPattern.MatchString(string(data)) {
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				rel = path
			}
			violations = append(violations, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan web runtime files failed: %v", err)
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Fatalf("legacy permission.anyOf key usage found in runtime web files: %v", violations)
	}
}

func TestReplaceXPathComponents_UsesStableFieldsWithoutAstNode(t *testing.T) {
	b := &WebModuleBuilder{}
	scriptContent := "components: {\n  __XPATH__\n}\n"
	xpathNode := &parser.PropertyNode{
		Line: 2,
		Text: "__XPATH__",
	}

	extendsImports := map[string]*parser.Import{
		"CompA": {ModuleSpecPath: "/virtual/a"},
		"CompB": {ModuleSpecPath: "/virtual/b"},
	}
	currentImports := map[string]*parser.Import{}

	updated, err := b.replaceXPathComponents(scriptContent, xpathNode, extendsImports, currentImports)
	if err != nil {
		t.Fatalf("replaceXPathComponents failed: %v", err)
	}

	if !strings.Contains(updated, "CompA") || !strings.Contains(updated, "CompB") {
		t.Fatalf("expected xpath replacement to inject new components, got:\n%s", updated)
	}
	if strings.Contains(updated, "__XPATH__") {
		t.Fatalf("expected xpath placeholder to be replaced, got:\n%s", updated)
	}
}

func TestReplaceXPathComponents_DeterministicComponentOrder(t *testing.T) {
	b := &WebModuleBuilder{}
	scriptContent := "components: {\n  __XPATH__\n}\n"
	xpathNode := &parser.PropertyNode{Line: 2, Text: "__XPATH__"}

	extendsImports := map[string]*parser.Import{
		"CompB": {ModuleSpecPath: "/virtual/b"},
		"CompA": {ModuleSpecPath: "/virtual/a"},
	}

	updated, err := b.replaceXPathComponents(scriptContent, xpathNode, extendsImports, map[string]*parser.Import{})
	if err != nil {
		t.Fatalf("replaceXPathComponents failed: %v", err)
	}

	if !strings.Contains(updated, "CompA,\n    CompB") {
		t.Fatalf("expected deterministic sorted component order, got:\n%s", updated)
	}
}

func TestReplaceXPathComponents_ReturnsErrorWhenXpathNodeIsNil(t *testing.T) {
	b := &WebModuleBuilder{}
	scriptContent := "components: {\n  __XPATH__\n}\n"

	extendsImports := map[string]*parser.Import{
		"CompA": {ModuleSpecPath: "/virtual/a"},
	}

	_, err := b.replaceXPathComponents(scriptContent, nil, extendsImports, map[string]*parser.Import{})
	if err == nil {
		t.Fatalf("expected error when xpath node is nil")
	}
}

func TestReplaceXPathComponents_ReturnsErrorWhenNoReplacementOccurs(t *testing.T) {
	b := &WebModuleBuilder{}
	scriptContent := "components: {\n  Xpath\n}\n"
	xpathNode := &parser.PropertyNode{Line: 2, Text: "__MISSING_SYMBOL__"}

	extendsImports := map[string]*parser.Import{
		"CompA": {ModuleSpecPath: "/virtual/a"},
	}

	_, err := b.replaceXPathComponents(scriptContent, xpathNode, extendsImports, map[string]*parser.Import{})
	if err == nil {
		t.Fatalf("expected error when xpath replacement target is not found")
	}
}

func TestReplaceXPathComponents_ReparseFallbackReplacesPropertyAssignment(t *testing.T) {
	b := &WebModuleBuilder{runtimeScope: newTestScope(), module: &meta.IrModule{Path: "/virtual/modules/test"}}
	scriptContent := "import { defineComponent } from 'vue';\n" +
		"import Xpath from '/virtual/modules/core/web/component/xpath.vue';\n\n" +
		"export default defineComponent({\n" +
		"  components: {\n" +
		"    Xpath: Xpath,\n" +
		"  },\n" +
		"});\n"
	xpathNode := &parser.PropertyNode{ValueText: "Xpath"}

	extendsImports := map[string]*parser.Import{
		"CompA": {ModuleSpecPath: "/virtual/a"},
	}

	updated, err := b.replaceXPathComponents(scriptContent, xpathNode, extendsImports, map[string]*parser.Import{})
	if err != nil {
		t.Fatalf("replaceXPathComponents failed: %v", err)
	}
	if !strings.Contains(updated, "CompA") {
		t.Fatalf("expected reparse fallback to inject replacement component, got:\n%s", updated)
	}
	if strings.Contains(updated, "Xpath: Xpath") {
		t.Fatalf("expected full property assignment to be replaced via reparse fallback, got:\n%s", updated)
	}
}

func TestAppendNewImports_DeterministicOrderAndNamedSorting(t *testing.T) {
	b := &WebModuleBuilder{}
	scriptContent := "import { defineComponent } from 'vue';"

	extendsImports := map[string]*parser.Import{
		"ZComp": {
			ModuleSpecPath: "/virtual/pkg/base",
			ReferenceIdent: "named",
		},
		"AComp": {
			ModuleSpecPath: "/virtual/pkg/base",
			ReferenceIdent: "named",
		},
		"BaseDefault": {
			ModuleSpecPath: "/virtual/pkg/base",
			ReferenceIdent: "default",
		},
		"Widget": {
			ModuleSpecPath: "/virtual/pkg/widgets",
			ReferenceIdent: "default",
		},
	}

	updated, err := b.appendNewImports(scriptContent, extendsImports, map[string]*parser.Import{})
	if err != nil {
		t.Fatalf("appendNewImports failed: %v", err)
	}

	expectedSuffix := "\nimport BaseDefault, { AComp, ZComp } from '/virtual/pkg/base';\nimport Widget from '/virtual/pkg/widgets';\n"
	if !strings.HasSuffix(updated, expectedSuffix) {
		t.Fatalf("expected deterministic import output, got:\n%s", updated)
	}
}

func TestTsParser_CollectsUiResourceDecls(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/route/routes.ts"
	content := `
import { defineRoute, defineAction } from '@/core/web/app/resource';

export const userListRoute = defineRoute('auth.route.user_list', {
	actions: ['auth.action.user_export'],
	requires: [{ model: 'auth.User', method: 'List' }],
  path: '/auth/users',
  meta: { pageTitle: 'User List' }
});

export const exportAction = defineAction('auth.action.user_export', {
	title: 'Export Users',
	requires: [{ kind: 'rpc', model: 'auth.User' }]
});
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDecls) != 2 {
		t.Fatalf("expected 2 ui resource decls, got %d", len(r.UiResourceDecls))
	}

	ids := []string{r.UiResourceDecls[0].ID, r.UiResourceDecls[1].ID}
	if !slices.Contains(ids, "auth.route.user_list") || !slices.Contains(ids, "auth.action.user_export") {
		t.Fatalf("unexpected ui resource ids: %v", ids)
	}

	for _, decl := range r.UiResourceDecls {
		switch decl.ID {
		case "auth.route.user_list":
			if !slices.Equal(decl.Requires, []string{"rpc:/auth.User/List"}) {
				t.Fatalf("unexpected route requires: %v", decl.Requires)
			}
			if !slices.Equal(decl.Actions, []string{"auth.action.user_export"}) {
				t.Fatalf("unexpected route actions: %v", decl.Actions)
			}
		case "auth.action.user_export":
			if !slices.Equal(decl.Requires, []string{"rpc:/auth.User/*"}) {
				t.Fatalf("unexpected action requires: %v", decl.Requires)
			}
		}
	}
}

func TestVueParser_CollectsUiResourceDeclsFromScriptSetup(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/views/UserListView.vue"
	content := `<template><div /></template>
<script setup lang="ts">
import { defineMenu, defineModelActions } from '@/core/web/app/resource';

const menus = [
	defineMenu('auth.menu.user_list', {
		title: 'User List',
		path: '/auth/users',
	})
];

const userActions = defineModelActions('auth.User', {
	entityTitle: 'User',
	titles: { delete: 'Deactivate User' },
	exclude: ['copy']
});
</script>`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDecls) != 4 {
		t.Fatalf("expected 4 ui resource decls (1 menu + 3 actions), got %d", len(r.UiResourceDecls))
	}

	declByID := make(map[string]*parser.UiResourceDecl)
	for _, decl := range r.UiResourceDecls {
		if decl == nil {
			continue
		}
		declByID[decl.ID] = decl
	}

	if _, ok := declByID["auth.menu.user_list"]; !ok {
		t.Fatalf("expected auth.menu.user_list to be collected")
	}

	expected := map[string]struct {
		requires []string
		title    string
	}{
		"auth.action.user_create": {requires: []string{"rpc:/auth.User/Create"}, title: "Create User"},
		"auth.action.user_edit":   {requires: []string{"rpc:/auth.User/Update"}, title: "Edit User"},
		"auth.action.user_delete": {requires: []string{"rpc:/auth.User/Delete"}, title: "Deactivate User"},
	}
	for id, expectedDecl := range expected {
		decl, ok := declByID[id]
		if !ok {
			t.Fatalf("expected %s to be collected", id)
		}
		if strings.TrimSpace(decl.ParentMenu) != "" {
			t.Fatalf("expected %s parentMenu empty, got %q", id, decl.ParentMenu)
		}
		if !slices.Equal(decl.Requires, expectedDecl.requires) {
			t.Fatalf("unexpected requires for %s: %v", id, decl.Requires)
		}
		if strings.TrimSpace(decl.Title) != expectedDecl.title {
			t.Fatalf("unexpected title for %s: %q", id, decl.Title)
		}
	}

	if _, ok := declByID["auth.action.user_copy"]; ok {
		t.Fatalf("did not expect excluded action auth.action.user_copy")
	}
}

func TestVueParser_CollectsUiResourceDeclsFromScript(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/views/RouteActionView.vue"
	content := `<template><div /></template>
<script lang="ts">
import { defineRoute, defineAction } from '@/core/web/app/resource';

export const userListRoute = defineRoute('auth.route.user_list', {
	actions: ['auth.action.user_export'],
	requires: [{ model: 'auth.User', method: 'List' }],
	path: '/auth/users',
	meta: { pageTitle: 'User List' }
});

export const exportAction = defineAction('auth.action.user_export', {
	title: 'Export Users',
	requires: [{ model: 'auth.User' }]
});
</script>`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDecls) != 2 {
		t.Fatalf("expected 2 ui resource decls, got %d", len(r.UiResourceDecls))
	}

	declByID := make(map[string]*parser.UiResourceDecl)
	for _, decl := range r.UiResourceDecls {
		if decl == nil {
			continue
		}
		declByID[decl.ID] = decl
	}

	routeDecl, ok := declByID["auth.route.user_list"]
	if !ok {
		t.Fatalf("expected auth.route.user_list to be collected")
	}
	if !slices.Equal(routeDecl.Requires, []string{"rpc:/auth.User/List"}) {
		t.Fatalf("unexpected route requires: %v", routeDecl.Requires)
	}
	if !slices.Equal(routeDecl.Actions, []string{"auth.action.user_export"}) {
		t.Fatalf("unexpected route actions: %v", routeDecl.Actions)
	}

	actionDecl, ok := declByID["auth.action.user_export"]
	if !ok {
		t.Fatalf("expected auth.action.user_export to be collected")
	}
	if !slices.Equal(actionDecl.Requires, []string{"rpc:/auth.User/*"}) {
		t.Fatalf("unexpected action requires: %v", actionDecl.Requires)
	}
}

func TestTsParser_InheritsParentMenuFromNestedDefineMenuChildren(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/menu/menus.ts"
	content := `
import { defineMenu } from '@/core/web/app/resource';

export const menus = [
  defineMenu('auth.menu.root', {
    title: 'Root',
    children: [
      defineMenu('auth.menu.user_list', {
        title: 'Users',
        path: '/auth/users'
      }),
      defineMenu('auth.menu.role_list', {
        title: 'Roles',
        path: '/auth/roles',
        parentMenu: 'auth.menu.manual_override'
      }),
      defineMenu('auth.menu.profile_root', {
        title: 'Profile',
        children: [
          defineMenu('auth.menu.profile_security', {
            title: 'Security',
            path: '/auth/profile/security'
          })
        ]
      })
    ]
  })
];
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDecls) != 5 {
		t.Fatalf("expected 5 ui resource decls, got %d", len(r.UiResourceDecls))
	}

	declByID := make(map[string]*parser.UiResourceDecl)
	for _, decl := range r.UiResourceDecls {
		if decl == nil {
			continue
		}
		declByID[decl.ID] = decl
	}

	if got := strings.TrimSpace(declByID["auth.menu.root"].ParentMenu); got != "" {
		t.Fatalf("expected root parentMenu empty, got %q", got)
	}
	if got := strings.TrimSpace(declByID["auth.menu.user_list"].ParentMenu); got != "auth.menu.root" {
		t.Fatalf("expected user_list parentMenu=auth.menu.root, got %q", got)
	}
	if got := strings.TrimSpace(declByID["auth.menu.role_list"].ParentMenu); got != "auth.menu.manual_override" {
		t.Fatalf("expected explicit parentMenu to be preserved, got %q", got)
	}
	if got := strings.TrimSpace(declByID["auth.menu.profile_root"].ParentMenu); got != "auth.menu.root" {
		t.Fatalf("expected profile_root parentMenu=auth.menu.root, got %q", got)
	}
	if got := strings.TrimSpace(declByID["auth.menu.profile_security"].ParentMenu); got != "auth.menu.profile_root" {
		t.Fatalf("expected profile_security parentMenu=auth.menu.profile_root, got %q", got)
	}
}

func TestExtractUiResources_UsesInheritedParentMenuFromNestedMenus(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/menu/menus.ts"
	content := `
import { defineMenu } from '@/core/web/app/resource';

export const menus = [
  defineMenu('auth.menu.root', {
    title: 'Root',
    children: [
      defineMenu('auth.menu.user_list', {
        title: 'Users',
        path: '/auth/users',
        children: [
          defineMenu('auth.menu.user_detail', {
						title: 'User Detail'
          })
        ]
      })
    ]
  })
];
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	module := &meta.IrModule{Name: "auth"}
	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{r})
	if err != nil {
		t.Fatalf("extract ui resources failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
	if len(resources) != 3 {
		t.Fatalf("expected 3 extracted resources, got %d", len(resources))
	}

	parentByName := make(map[string]string)
	for _, res := range resources {
		if res == nil {
			continue
		}
		parentByName[res.Name] = strings.TrimSpace(res.ParentResourceName)
	}

	if got := parentByName["auth.menu.root"]; got != "" {
		t.Fatalf("expected root parent empty, got %q", got)
	}
	if got := parentByName["auth.menu.user_list"]; got != "auth.menu.root" {
		t.Fatalf("expected user_list parent auth.menu.root, got %q", got)
	}
	if got := parentByName["auth.menu.user_detail"]; got != "auth.menu.user_list" {
		t.Fatalf("expected user_detail parent auth.menu.user_list, got %q", got)
	}
}

func TestTsParser_DefaultsMissingRequireMethodToWildcard(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/views/actions.ts"
	content := `
import { defineAction } from '@/core/web/app/resource';

export const actionExport = defineAction('auth.action.user_export', {
  requires: [{ model: 'auth.User' }],
	title: 'Export Users'
});
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDecls) != 1 {
		t.Fatalf("expected 1 ui resource decl, got %d", len(r.UiResourceDecls))
	}
	if !slices.Equal(r.UiResourceDecls[0].Requires, []string{"rpc:/auth.User/*"}) {
		t.Fatalf("expected missing method to normalize to wildcard require, got %v", r.UiResourceDecls[0].Requires)
	}
}

func TestTsParser_SkipsPublicRouteRequiresAuthFalse(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/route/routes.ts"
	content := `
import { defineRoute } from '@/core/web/app/resource';

export const loginRoute = defineRoute('auth.route.login', {
  path: '/login',
	meta: { requiresAuth: false, pageTitle: 'Login' }
});

export const userListRoute = defineRoute('auth.route.user_list', {
  path: '/auth/users',
	requires: [{ model: 'auth.User', method: 'Browse' }],
	meta: { requiresAuth: true, pageTitle: 'User List' }
});
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDecls) != 1 {
		t.Fatalf("expected 1 ui resource decl after skipping public route, got %d", len(r.UiResourceDecls))
	}
	if r.UiResourceDecls[0].ID != "auth.route.user_list" {
		t.Fatalf("unexpected route id: %s", r.UiResourceDecls[0].ID)
	}
}

func TestTsParser_ReportsFatalForNonLiteralResourceID(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/route/routes.ts"
	content := `
import { defineRoute } from '@/core/web/app/resource';

const routeId = 'auth.route.user_list';

export const userListRoute = defineRoute(routeId, {
  path: '/auth/users',
	meta: { pageTitle: 'User List' }
});
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDecls) != 0 {
		t.Fatalf("expected no extracted decls for non-literal id, got %d", len(r.UiResourceDecls))
	}
	if len(r.UiResourceDeclIssues) == 0 {
		t.Fatalf("expected parser issues for non-literal id")
	}
	issue := r.UiResourceDeclIssues[0]
	if issue.Severity != parser.UiResourceIssueSeverityFatal {
		t.Fatalf("expected fatal severity, got %s", issue.Severity)
	}
	if issue.Code != parser.UiResourceIssueCodeDeclIDNotLiteral {
		t.Fatalf("expected issue code %s, got %s", parser.UiResourceIssueCodeDeclIDNotLiteral, issue.Code)
	}
	if !strings.Contains(issue.Message, "string literal resource id") {
		t.Fatalf("unexpected issue message: %s", issue.Message)
	}
	if issue.Line <= 0 || issue.Column <= 0 {
		t.Fatalf("expected issue location, got line=%d col=%d", issue.Line, issue.Column)
	}

	module := &meta.IrModule{Name: "auth"}
	_, _, extractErr := extractUiResources(module, []*parser.ParserResult{r})
	if extractErr == nil {
		t.Fatalf("expected extractor to fail on parser fatal issue")
	}
	if !strings.Contains(extractErr.Error(), path) {
		t.Fatalf("expected error to contain source path, got: %v", extractErr)
	}
	if !strings.Contains(extractErr.Error(), string(parser.UiResourceIssueCodeDeclIDNotLiteral)) {
		t.Fatalf("expected error to contain issue code, got: %v", extractErr)
	}
}

func TestTsParser_ReportsFatalForDynamicRequires(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/views/actions.ts"
	content := `
import { defineAction } from '@/core/web/app/resource';

const req = ['rpc:/auth.User/Export'];

export const actionExport = defineAction('auth.action.user_export', {
  requires: req,
	title: 'Export Users'
});
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDeclIssues) == 0 {
		t.Fatalf("expected parser issues for dynamic requires")
	}
	var found bool
	for _, issue := range r.UiResourceDeclIssues {
		if issue.Severity == parser.UiResourceIssueSeverityFatal && issue.Code == parser.UiResourceIssueCodeDeclRequiresNotLiteral && strings.Contains(issue.Message, "requires must be an object-literal array") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected fatal requires issue, got: %#v", r.UiResourceDeclIssues)
	}
}

func TestTsParser_ReportsFatalForLegacyStringRequires(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/views/actions.ts"
	content := `
import { defineAction } from '@/core/web/app/resource';

export const actionExport = defineAction('auth.action.user_export', {
  requires: ['rpc:/auth.User/Export'],
	title: 'Export Users'
});
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDeclIssues) == 0 {
		t.Fatalf("expected parser issues for legacy string requires")
	}
	var found bool
	for _, issue := range r.UiResourceDeclIssues {
		if issue.Severity == parser.UiResourceIssueSeverityFatal && issue.Code == parser.UiResourceIssueCodeDeclRequiresNotLiteral && strings.Contains(issue.Message, "requires must be an object-literal array") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected fatal requires issue for legacy string syntax, got: %#v", r.UiResourceDeclIssues)
	}
}

func TestTsParser_ReportsFatalForEmptyRequireMethod(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/views/actions.ts"
	content := `
import { defineAction } from '@/core/web/app/resource';

export const actionExport = defineAction('auth.action.user_export', {
  requires: [{ model: 'auth.User', method: '' }],
	title: 'Export Users'
});
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDeclIssues) == 0 {
		t.Fatalf("expected parser issues for empty require method")
	}
	var found bool
	for _, issue := range r.UiResourceDeclIssues {
		if issue.Severity == parser.UiResourceIssueSeverityFatal && issue.Code == parser.UiResourceIssueCodeDeclRequiresNotLiteral && strings.Contains(issue.Message, "property \"method\" must not be empty") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected fatal requires issue for empty method, got: %#v", r.UiResourceDeclIssues)
	}
}

func TestExtractUiResources_ExpandsModelActions(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/route/routes.ts"
	content := `
import { defineMenu, defineModelActions, defineRoute } from '@/core/web/app/resource';

export const menus = [
	defineMenu('auth.menu.user_list', {
		title: 'User List',
		path: '/auth/users',
	})
];

export const routes = [
	defineRoute('auth.route.user_list', {
		path: '/auth/users'
	})
];

export const userActions = defineModelActions('auth.User', {
	entityTitle: 'User',
	titles: { delete: 'Deactivate User' },
	exclude: ['copy']
});
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDecls) != 5 {
		t.Fatalf("expected 5 ui resource decls (1 menu + 1 route + 3 actions), got %d", len(r.UiResourceDecls))
	}

	module := &meta.IrModule{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "m1", Valid: true}},
		Name:      "auth",
	}
	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{r})
	if err != nil {
		t.Fatalf("extract ui resources failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
	if len(resources) != 5 {
		t.Fatalf("expected 5 extracted ui resources (1 menu + 1 route + 3 actions), got %d", len(resources))
	}

	resourceIDs := []string{}
	actionCount := 0
	titleByName := map[string]string{}
	for _, v := range resources {
		resourceIDs = append(resourceIDs, v.Name)
		titleByName[v.Name] = v.Title
		if v.Type == meta.UiResourceTypeAction {
			actionCount++
		}
	}
	if actionCount != 3 {
		t.Fatalf("expected 3 action resources, got %d", actionCount)
	}
	if !slices.Contains(resourceIDs, "auth.menu.user_list") {
		t.Fatalf("expected parent menu resource present, got IDs: %v", resourceIDs)
	}

	if !slices.Contains(resourceIDs, "auth.action.user_create") ||
		!slices.Contains(resourceIDs, "auth.action.user_edit") ||
		!slices.Contains(resourceIDs, "auth.action.user_delete") {
		t.Fatalf("unexpected resource IDs: %v", resourceIDs)
	}
	if titleByName["auth.action.user_create"] != "Create User" || titleByName["auth.action.user_edit"] != "Edit User" || titleByName["auth.action.user_delete"] != "Deactivate User" {
		t.Fatalf("unexpected model action titles: %#v", titleByName)
	}
}

func TestTsParser_RejectsInvalidDefineModelActionDisplayOptions(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/views/UserListView.vue"
	content := `<template><div /></template>
<script setup lang="ts">
const entityTitle = 'User';
const userActions = defineModelActions('auth.User', {
	entityTitle: entityTitle,
	titles: { create: makeTitle() }
});
</script>`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDeclIssues) == 0 {
		t.Fatalf("expected parser issues for invalid defineModelActions display options")
	}
	var hasEntityTitleIssue bool
	var hasTitlesIssue bool
	for _, issue := range r.UiResourceDeclIssues {
		if issue == nil || issue.Severity != parser.UiResourceIssueSeverityFatal {
			continue
		}
		if issue.Code == parser.UiResourceIssueCodeModelActionEntityTitleNotLiteral {
			hasEntityTitleIssue = true
		}
		if issue.Code == parser.UiResourceIssueCodeModelActionTitlesInvalid {
			hasTitlesIssue = true
		}
	}
	if !hasEntityTitleIssue || !hasTitlesIssue {
		t.Fatalf("expected UI_DECL_012 and UI_DECL_013, got %#v", r.UiResourceDeclIssues)
	}
}

func TestTsParser_AcceptsDefineModelActionTitlesWithTrailingComma(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/meta"})

	path := "/virtual/modules/meta/web/views/ModuleListView.vue"
	content := `<template><div /></template>
<script setup lang="ts">
const moduleActions = defineModelActions('meta.IrModuleIndex', {
	entityTitle: 'Module Index',
	titles: {
		edit: 'Edit Module Index',
		copy: 'Copy Module Index',
		delete: 'Delete Module Index',
	},
});
</script>`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	for _, issue := range r.UiResourceDeclIssues {
		if issue != nil && issue.Code == parser.UiResourceIssueCodeModelActionTitlesInvalid {
			t.Fatalf("expected trailing comma in titles to be accepted, got issue: %#v", issue)
		}
	}

	declByID := make(map[string]*parser.UiResourceDecl, len(r.UiResourceDecls))
	for _, decl := range r.UiResourceDecls {
		if decl == nil {
			continue
		}
		declByID[decl.ID] = decl
	}

	if got := strings.TrimSpace(declByID["meta.action.ir_module_index_edit"].Title); got != "Edit Module Index" {
		t.Fatalf("unexpected edit title: %q", got)
	}
	if got := strings.TrimSpace(declByID["meta.action.ir_module_index_copy"].Title); got != "Copy Module Index" {
		t.Fatalf("unexpected copy title: %q", got)
	}
	if got := strings.TrimSpace(declByID["meta.action.ir_module_index_delete"].Title); got != "Delete Module Index" {
		t.Fatalf("unexpected delete title: %q", got)
	}
}

func TestTsParser_RejectsParentMenuOutsideDefineMenu(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/route/routes.ts"
	content := `
import { defineRoute } from '@/core/web/app/resource';

export const userListRoute = defineRoute('auth.route.user_list', {
  parentMenu: 'auth.menu.user_list',
  path: '/auth/users'
});
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDeclIssues) == 0 {
		t.Fatalf("expected parser issue for legacy route parentMenu")
	}
	var found bool
	for _, issue := range r.UiResourceDeclIssues {
		if issue.Severity == parser.UiResourceIssueSeverityFatal && issue.Code == parser.UiResourceIssueCodeParentMenuOnlyForMenu {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %s issue, got %#v", parser.UiResourceIssueCodeParentMenuOnlyForMenu, r.UiResourceDeclIssues)
	}
}

func TestExtractUiResourceRelations_InfersMenuRouteAndRouteActions(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}
	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu, Path: "/auth/users"},
		{ID: "auth.route.user_list", Type: parser.UiResourceTypeRoute, Path: "/auth/users", Actions: []string{"auth.action.user_create", "auth.action.user_edit"}},
		{ID: "auth.action.user_create", Type: parser.UiResourceTypeAction},
		{ID: "auth.action.user_edit", Type: parser.UiResourceTypeAction},
	}}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("extract ui resources failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}

	menuRoutes, routeActions, err := extractUiResourceRelations(resources, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("extract ui resource relations failed: %v", err)
	}
	if len(menuRoutes) != 1 {
		t.Fatalf("expected 1 menu_route relation, got %d", len(menuRoutes))
	}
	if menuRoutes[0].MenuName != "auth.menu.user_list" || menuRoutes[0].RouteName != "auth.route.user_list" {
		t.Fatalf("unexpected menu_route relation: %#v", menuRoutes[0])
	}
	if len(routeActions) != 2 {
		t.Fatalf("expected 2 route_action relations, got %d", len(routeActions))
	}
	if routeActions[0].RouteName != "auth.route.user_list" || routeActions[0].ActionName != "auth.action.user_create" {
		t.Fatalf("unexpected first route_action relation: %#v", routeActions[0])
	}
	if routeActions[1].RouteName != "auth.route.user_list" || routeActions[1].ActionName != "auth.action.user_edit" {
		t.Fatalf("unexpected second route_action relation: %#v", routeActions[1])
	}
}

func TestPersistModuleUiResources_PersistsUiResourceRelations(t *testing.T) {
	testRuntimeScope := newTestScopeWithDB(t)
	tenv, ok := testRuntimeScope.(*testScope)
	if !ok {
		t.Fatalf("unexpected env type")
	}
	if err := tenv.db.AutoMigrate(&meta.IrUiResource{}, &meta.IrUiResourceMenuRoute{}, &meta.IrUiResourceRouteAction{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}

	b := &WebModuleBuilder{runtimeScope: tenv}
	module := &meta.IrModule{BaseModel: meta.BaseModel{Id: sql.NullString{String: "m_auth", Valid: true}}, Name: "auth"}
	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu, Path: "/auth/users"},
		{ID: "auth.route.user_list", Type: parser.UiResourceTypeRoute, Path: "/auth/users", Actions: []string{"auth.action.user_create"}},
		{ID: "auth.action.user_create", Type: parser.UiResourceTypeAction},
	}}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("extract ui resources failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
	menuRoutes, routeActions, err := extractUiResourceRelations(resources, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("extract ui resource relations failed: %v", err)
	}

	if err := b.persistModuleUiResources(module.Id.String, resources, menuRoutes, routeActions); err != nil {
		t.Fatalf("persist module ui resources failed: %v", err)
	}
	if err := b.persistModuleUiResources(module.Id.String, resources, menuRoutes, routeActions); err != nil {
		t.Fatalf("persist module ui resources second time failed: %v", err)
	}

	var menuRouteCount int64
	if err := tenv.db.Model(&meta.IrUiResourceMenuRoute{}).Count(&menuRouteCount).Error; err != nil {
		t.Fatalf("count menu_route failed: %v", err)
	}
	if menuRouteCount != 1 {
		t.Fatalf("expected 1 menu_route row, got %d", menuRouteCount)
	}

	var routeActionCount int64
	if err := tenv.db.Model(&meta.IrUiResourceRouteAction{}).Count(&routeActionCount).Error; err != nil {
		t.Fatalf("count route_action failed: %v", err)
	}
	if routeActionCount != 1 {
		t.Fatalf("expected 1 route_action row, got %d", routeActionCount)
	}

	rows := make([]*meta.IrUiResource, 0)
	if err := tenv.db.
		Where("name IN ?", []string{"auth.menu.user_list", "auth.route.user_list", "auth.action.user_create"}).
		Find(&rows).Error; err != nil {
		t.Fatalf("load ui resource rows failed: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 ui resource rows, got %d", len(rows))
	}

	rowByName := make(map[string]*meta.IrUiResource, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		rowByName[row.Name] = row
	}

	menuRow := rowByName["auth.menu.user_list"]
	if menuRow == nil || !menuRow.Id.Valid {
		t.Fatalf("expected persisted menu row with id")
	}
	if menuRow.ParentPath != menuRow.Id.String+"/" {
		t.Fatalf("expected root menu parent_path %q, got %q", menuRow.Id.String+"/", menuRow.ParentPath)
	}

	routeRow := rowByName["auth.route.user_list"]
	if routeRow == nil {
		t.Fatalf("expected persisted route row")
	}
	if routeRow.ParentPath != "" {
		t.Fatalf("expected route parent_path empty, got %q", routeRow.ParentPath)
	}

	actionRow := rowByName["auth.action.user_create"]
	if actionRow == nil {
		t.Fatalf("expected persisted action row")
	}
	if actionRow.ParentPath != "" {
		t.Fatalf("expected action parent_path empty, got %q", actionRow.ParentPath)
	}
}

func TestPersistModuleUiResources_CleansModuleRowsWhenNamesEmpty(t *testing.T) {
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	if err := testRuntimeScope.db.AutoMigrate(&meta.IrUiResource{}, &meta.IrUiResourceMenuRoute{}, &meta.IrUiResourceRouteAction{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	seedRows := []*meta.IrUiResource{
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "menu_mod1", Valid: true}}, Name: "auth.menu.root", Type: meta.UiResourceTypeMenu, ModuleId: sql.NullString{String: "mod1", Valid: true}},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "route_mod1", Valid: true}}, Name: "auth.route.root", Type: meta.UiResourceTypeRoute, ModuleId: sql.NullString{String: "mod1", Valid: true}},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "action_mod1", Valid: true}}, Name: "auth.action.root", Type: meta.UiResourceTypeAction, ModuleId: sql.NullString{String: "mod1", Valid: true}},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "menu_mod2", Valid: true}}, Name: "other.menu.root", Type: meta.UiResourceTypeMenu, ModuleId: sql.NullString{String: "mod2", Valid: true}},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "route_mod2", Valid: true}}, Name: "other.route.root", Type: meta.UiResourceTypeRoute, ModuleId: sql.NullString{String: "mod2", Valid: true}},
	}
	for _, row := range seedRows {
		if err := testRuntimeScope.db.Create(row).Error; err != nil {
			t.Fatalf("seed ui resource %s: %v", row.Name, err)
		}
	}
	if err := testRuntimeScope.db.Create(&meta.IrUiResourceMenuRoute{
		MenuUiResourceId:  sql.NullString{String: "menu_mod1", Valid: true},
		RouteUiResourceId: sql.NullString{String: "route_mod1", Valid: true},
	}).Error; err != nil {
		t.Fatalf("seed mod1 menu route: %v", err)
	}
	if err := testRuntimeScope.db.Create(&meta.IrUiResourceRouteAction{
		RouteUiResourceId:  sql.NullString{String: "route_mod1", Valid: true},
		ActionUiResourceId: sql.NullString{String: "action_mod1", Valid: true},
	}).Error; err != nil {
		t.Fatalf("seed mod1 route action: %v", err)
	}
	if err := testRuntimeScope.db.Create(&meta.IrUiResourceMenuRoute{
		MenuUiResourceId:  sql.NullString{String: "menu_mod2", Valid: true},
		RouteUiResourceId: sql.NullString{String: "route_mod2", Valid: true},
	}).Error; err != nil {
		t.Fatalf("seed mod2 menu route: %v", err)
	}

	if err := b.persistModuleUiResources("mod1", nil, nil, nil); err != nil {
		t.Fatalf("persistModuleUiResources cleanup failed: %v", err)
	}

	var mod1Count int64
	if err := testRuntimeScope.db.Model(&meta.IrUiResource{}).Where("module_id = ?", "mod1").Count(&mod1Count).Error; err != nil {
		t.Fatalf("count mod1 ui resources failed: %v", err)
	}
	if mod1Count != 0 {
		t.Fatalf("expected mod1 resources to be deleted, got %d", mod1Count)
	}

	var mod2Count int64
	if err := testRuntimeScope.db.Model(&meta.IrUiResource{}).Where("module_id = ?", "mod2").Count(&mod2Count).Error; err != nil {
		t.Fatalf("count mod2 ui resources failed: %v", err)
	}
	if mod2Count != 2 {
		t.Fatalf("expected mod2 resources to remain, got %d", mod2Count)
	}

	var menuRoutes []meta.IrUiResourceMenuRoute
	if err := testRuntimeScope.db.Order("menu_ui_resource_id").Find(&menuRoutes).Error; err != nil {
		t.Fatalf("query menu routes failed: %v", err)
	}
	if len(menuRoutes) != 1 || menuRoutes[0].MenuUiResourceId.String != "menu_mod2" {
		t.Fatalf("expected only unrelated menu route to remain, got %#v", menuRoutes)
	}

	var routeActions []meta.IrUiResourceRouteAction
	if err := testRuntimeScope.db.Find(&routeActions).Error; err != nil {
		t.Fatalf("query route actions failed: %v", err)
	}
	if len(routeActions) != 0 {
		t.Fatalf("expected mod1 route actions to be deleted, got %#v", routeActions)
	}

	if err := b.persistModuleUiResources("   ", []*meta.IrUiResource{{Name: "ignored"}}, nil, nil); err != nil {
		t.Fatalf("blank module id fast path failed: %v", err)
	}
	if err := b.persistModuleUiResources("mod1", []*meta.IrUiResource{{Name: "   "}}, nil, nil); err != nil {
		t.Fatalf("blank resource name cleanup failed: %v", err)
	}
}

func TestBuildUiResourceParentPathByName(t *testing.T) {
	t.Run("builds nested paths and falls back for missing parent", func(t *testing.T) {
		paths, err := buildUiResourceParentPathByName(
			map[string]string{"root": "menu_root", "child": "menu_child", "orphan": "menu_orphan"},
			map[string]string{"root": "", "child": "root", "orphan": "missing"},
		)
		if err != nil {
			t.Fatalf("buildUiResourceParentPathByName returned error: %v", err)
		}
		if paths["root"] != "menu_root/" {
			t.Fatalf("expected root path, got %q", paths["root"])
		}
		if paths["child"] != "menu_root/menu_child/" {
			t.Fatalf("expected nested child path, got %q", paths["child"])
		}
		if paths["orphan"] != "menu_orphan/" {
			t.Fatalf("expected missing parent to fall back to root path, got %q", paths["orphan"])
		}
	})

	t.Run("detects parent cycles", func(t *testing.T) {
		_, err := buildUiResourceParentPathByName(
			map[string]string{"a": "id_a", "b": "id_b"},
			map[string]string{"a": "b", "b": "a"},
		)
		if err == nil || !strings.Contains(err.Error(), "parent cycle detected") {
			t.Fatalf("expected cycle error, got %v", err)
		}
	})

	t.Run("fails when resource id is missing", func(t *testing.T) {
		_, err := buildUiResourceParentPathByName(
			map[string]string{"broken": "   "},
			map[string]string{"broken": ""},
		)
		if err == nil || !strings.Contains(err.Error(), "id not found") {
			t.Fatalf("expected missing id error, got %v", err)
		}
	})
}

func TestPersist_IntegratesUiResourcesComponentsAndWarnings(t *testing.T) {
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	var logs bytes.Buffer
	testRuntimeScope.log = slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if err := testRuntimeScope.db.AutoMigrate(&meta.IrComponent{}, &meta.IrUiResource{}, &meta.IrUiResourceMenuRoute{}, &meta.IrUiResourceRouteAction{}, &meta.IrModel{}, &meta.IrService{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	mustExec(t, testRuntimeScope.db, `CREATE TABLE auth_role (id TEXT PRIMARY KEY, code TEXT)`)
	mustExec(t, testRuntimeScope.db, `CREATE TABLE auth_role_ui_resource (id TEXT, role_id TEXT, mode TEXT, ir_application_id TEXT, ir_ui_resource_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`)
	mustExec(t, testRuntimeScope.db, `CREATE UNIQUE INDEX idx_auth_role_ui_resource_pair ON auth_role_ui_resource(role_id, ir_ui_resource_id)`)
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	if err := testRuntimeScope.db.Create(&meta.IrModel{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "model_lead", Valid: true}},
		Application: "crm",
		Name:        "Lead",
		Path:        "/virtual/modules/crm/models/lead.ts",
	}).Error; err != nil {
		t.Fatalf("seed model failed: %v", err)
	}
	if err := testRuntimeScope.db.Create(&meta.IrService{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "svc_read", Valid: true}},
		ModelId:   sql.NullString{String: "model_lead", Valid: true},
		Name:      "Read",
	}).Error; err != nil {
		t.Fatalf("seed service failed: %v", err)
	}
	if err := testRuntimeScope.db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?)`, "role_user", "base.user").Error; err != nil {
		t.Fatalf("seed auth_role failed: %v", err)
	}

	mod := &meta.IrModule{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "mod_auth", Valid: true}},
		Name:      "auth",
		Path:      "/virtual/modules/auth",
	}
	insidePath := filepath.Join(mod.Path, "web", "views", "DashboardView.vue")
	outsidePath := "/virtual/modules/other/web/views/ForeignView.vue"
	buildResult := withParserResults(
		&module.BuildResult{Module: mod},
		&parser.ParserResult{
			Path: insidePath,
			VueComponent: &meta.IrComponent{
				Name: "DashboardView",
				Path: insidePath,
			},
			UiResourceDecls: []*parser.UiResourceDecl{
				{ID: "auth.menu.dashboard", Type: parser.UiResourceTypeMenu, Path: "/auth/dashboard", DefaultRoles: []string{"base.user"}},
				{ID: "auth.route.dashboard", Type: parser.UiResourceTypeRoute, Path: "/auth/dashboard", Actions: []string{"auth.action.dashboard_read"}},
				{ID: "auth.action.dashboard_read", Type: parser.UiResourceTypeAction, Requires: []string{"rpc:/crm.Lead/read"}},
			},
			UiResourceDeclIssues: []*parser.UiResourceDeclIssue{{
				Severity:   parser.UiResourceIssueSeverityWarning,
				Code:       parser.UiResourceIssueCodeDeclIDNamingSuggested,
				Factory:    "defineRoute",
				ResourceID: "auth.route.dashboard",
				Message:    "prefer module-scoped route ids",
				SourcePath: insidePath,
				Line:       12,
				Column:     4,
			}},
		},
		&parser.ParserResult{
			Path: outsidePath,
			VueComponent: &meta.IrComponent{
				Name: "ForeignView",
				Path: outsidePath,
			},
		},
	)

	if err := b.persist(buildResult); err != nil {
		t.Fatalf("persist returned error: %v", err)
	}

	if len(mod.Components) != 1 || mod.Components[0].Path != insidePath {
		t.Fatalf("expected only in-module component to be collected, got %#v", mod.Components)
	}
	if len(mod.UiResources) != 3 {
		t.Fatalf("expected 3 ui resources on module, got %d", len(mod.UiResources))
	}

	var componentRows []*meta.IrComponent
	if err := testRuntimeScope.db.Where("module_id = ?", mod.Id.String).Find(&componentRows).Error; err != nil {
		t.Fatalf("query components failed: %v", err)
	}
	if len(componentRows) != 1 || componentRows[0].Path != insidePath {
		t.Fatalf("expected one persisted in-module component, got %#v", componentRows)
	}

	uiRows := make([]*meta.IrUiResource, 0)
	if err := testRuntimeScope.db.Where("module_id = ?", mod.Id.String).Order("name").Find(&uiRows).Error; err != nil {
		t.Fatalf("query ui resources failed: %v", err)
	}
	if len(uiRows) != 3 {
		t.Fatalf("expected 3 persisted ui resources, got %#v", uiRows)
	}
	uiByName := make(map[string]*meta.IrUiResource, len(uiRows))
	for _, row := range uiRows {
		uiByName[row.Name] = row
	}
	if uiByName["auth.menu.dashboard"] == nil || uiByName["auth.menu.dashboard"].ParentPath == "" {
		t.Fatalf("expected menu row with root parent path, got %#v", uiByName["auth.menu.dashboard"])
	}

	var menuRouteCount int64
	if err := testRuntimeScope.db.Model(&meta.IrUiResourceMenuRoute{}).Count(&menuRouteCount).Error; err != nil {
		t.Fatalf("count menu routes failed: %v", err)
	}
	if menuRouteCount != 1 {
		t.Fatalf("expected 1 menu route, got %d", menuRouteCount)
	}

	var routeActionCount int64
	if err := testRuntimeScope.db.Model(&meta.IrUiResourceRouteAction{}).Count(&routeActionCount).Error; err != nil {
		t.Fatalf("count route actions failed: %v", err)
	}
	if routeActionCount != 1 {
		t.Fatalf("expected 1 route action, got %d", routeActionCount)
	}

	var grants []roleUiResourceGrantRow
	if err := testRuntimeScope.db.Table("auth_role_ui_resource").Find(&grants).Error; err != nil {
		t.Fatalf("query default role grants failed: %v", err)
	}
	if len(grants) != 1 || grants[0].RoleId.String != "role_user" {
		t.Fatalf("expected one default role grant for base.user, got %#v", grants)
	}

	logOutput := logs.String()
	if !strings.Contains(logOutput, "ui resource validation warning") || !strings.Contains(logOutput, string(parser.UiResourceIssueCodeDeclIDNamingSuggested)) {
		t.Fatalf("expected warning log to include code, got %q", logOutput)
	}
}

func TestExtractUiResources_DuplicateWithoutOverrideFails(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu, Title: "User List"},
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu, Title: "Users"},
	}}

	_, _, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err == nil {
		t.Fatalf("expected duplicate id error")
	}
	if !strings.Contains(err.Error(), "[UI_VAL_002]") {
		t.Fatalf("expected diagnostic code UI_VAL_002, got: %v", err)
	}
}

func TestExtractUiResources_DuplicateEquivalentDeclsAreDeduped(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{
			ID:         "auth.action.session_create",
			Type:       parser.UiResourceTypeAction,
			Requires:   []string{"rpc:/auth.Session/Create"},
			SourcePath: "/modules/auth/web/views/SessionListView.vue",
			SourceLine: 10,
		},
		{
			ID:         "auth.action.session_create",
			Type:       parser.UiResourceTypeAction,
			Requires:   []string{"rpc:/auth.Session/Create"},
			SourcePath: "/modules/auth/web/views/SessionFormView.vue",
			SourceLine: 35,
		},
	}}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("expected equivalent duplicates to be deduped, got error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 deduped resource, got %d", len(resources))
	}
	if resources[0].Name != "auth.action.session_create" {
		t.Fatalf("unexpected deduped resource: %s", resources[0].Name)
	}
}

func TestWebBuilder_IntegratesCrossFileDuplicateModelActionsFromVue(t *testing.T) {
	tmpDir := t.TempDir()
	modulesPath := filepath.Join(tmpDir, "modules")
	distPath := filepath.Join(tmpDir, "dist")
	modulePath := filepath.Join(modulesPath, "auth")
	viewsPath := filepath.Join(modulePath, "web", "views")
	entryPoint := filepath.Join(modulePath, "web", "index.ts")

	mustMkdirAll := func(path string) {
		t.Helper()
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
	}

	mustWrite := func(path string, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write file failed: %v", err)
		}
	}

	mustMkdirAll(modulesPath)
	mustMkdirAll(viewsPath)
	mustMkdirAll(distPath)

	mustWrite(filepath.Join(modulesPath, "tsconfig.json"), `{
	"compilerOptions": {
		"target": "ES2020",
		"module": "ESNext",
		"moduleResolution": "bundler",
		"baseUrl": ".",
		"paths": {
			"@/*": ["*"]
		}
	}
}`)

	mustWrite(entryPoint, `import './views/SessionListView.vue';
import './views/SessionFormView.vue';

export default {
	mount() {}
};
`)

	mustWrite(filepath.Join(viewsPath, "SessionListView.vue"), `<template><div>list</div></template>
<script setup lang="ts">
const listActions = defineModelActions('auth.Session', {
	exclude: ['create', 'edit', 'delete']
});
</script>
`)

	mustWrite(filepath.Join(viewsPath, "SessionFormView.vue"), `<template><div>form</div></template>
<script setup lang="ts">
const formActions = defineModelActions('auth.Session', {
	exclude: ['create', 'edit', 'delete']
});
</script>
`)

	testRuntimeScope := newTestScopeWithDB(t)
	tenv, ok := testRuntimeScope.(*testScope)
	if !ok {
		t.Fatalf("unexpected env type")
	}
	tenv.cfg = &config.Config{
		ModulesPath: modulesPath,
		DistPath:    distPath,
		Compile:     config.NewDefaultCompileConfig(),
		Server:      config.NewDefaultServerConfig(),
		FrontendEnv: map[string]any{},
		BackendEnv:  map[string]any{},
		Log:         config.NewDefaultLogConfig(),
		Db:          config.NewDefaultDbConfig(),
		Auth:        config.NewDefaultAuthConfig(),
		Task:        config.NewDefaultTaskConfig(),
	}

	if err := tenv.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}, &meta.IrComponent{}, &meta.IrUiResource{}, &meta.IrUiResourceMenuRoute{}, &meta.IrUiResourceRouteAction{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}

	module := &meta.IrModule{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "m_auth_builder", Valid: true}},
		Name:      "auth",
		Path:      modulePath,
		Status:    meta.Installed,
	}

	prebuildPlugin := defaultesbplugins.NewWebPrebuildPlugin(tenv, module, entryPoint)
	buildPlugin := defaultesbplugins.NewWebPrebuildPlugin(tenv, module, entryPoint)
	b := NewWebBuilder(
		tenv,
		nil,
		module,
		entryPoint,
		WithPublishDist(false),
		WithPrebuildPlugin(prebuildPlugin),
		WithBuildPlugin(buildPlugin),
	)

	buildResult, err := b.Build()
	if err != nil {
		t.Fatalf("builder build failed: %v", err)
	}
	if buildResult == nil {
		t.Fatalf("expected non-nil build result")
	}

	var count int64
	if err := tenv.db.
		Table("meta_ir_ui_resource").
		Where("name = ? AND type = ?", "auth.action.session_copy", "ACTION").
		Count(&count).Error; err != nil {
		t.Fatalf("count ACTION rows failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 persisted ACTION row for duplicated model-actions copy id, got %d", count)
	}
}

func TestWebBuilder_IntegratesCrossFileConflictingRouteActionsFromVueFailsUIVal002(t *testing.T) {
	tmpDir := t.TempDir()
	modulesPath := filepath.Join(tmpDir, "modules")
	distPath := filepath.Join(tmpDir, "dist")
	modulePath := filepath.Join(modulesPath, "auth")
	viewsPath := filepath.Join(modulePath, "web", "views")
	entryPoint := filepath.Join(modulePath, "web", "index.ts")

	mustMkdirAll := func(path string) {
		t.Helper()
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
	}

	mustWrite := func(path string, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write file failed: %v", err)
		}
	}

	mustMkdirAll(modulesPath)
	mustMkdirAll(viewsPath)
	mustMkdirAll(distPath)

	mustWrite(filepath.Join(modulesPath, "tsconfig.json"), `{
	"compilerOptions": {
		"target": "ES2020",
		"module": "ESNext",
		"moduleResolution": "bundler",
		"baseUrl": ".",
		"paths": {
			"@/*": ["*"]
		}
	}
}`)

	mustWrite(entryPoint, `import './views/SessionListView.vue';
import './views/SessionFormView.vue';

export default {
	mount() {}
};
`)

	mustWrite(filepath.Join(viewsPath, "SessionListView.vue"), `<template><div>list</div></template>
<script setup lang="ts">
const listRoute = defineRoute('auth.route.session_list', {
	path: '/auth/sessions',
	actions: ['auth.action.session_copy']
});
</script>
`)

	mustWrite(filepath.Join(viewsPath, "SessionFormView.vue"), `<template><div>form</div></template>
<script setup lang="ts">
const formRoute = defineRoute('auth.route.session_list', {
	path: '/auth/sessions',
	actions: ['auth.action.session_delete']
});
</script>
`)

	testRuntimeScope := newTestScopeWithDB(t)
	tenv, ok := testRuntimeScope.(*testScope)
	if !ok {
		t.Fatalf("unexpected env type")
	}
	tenv.cfg = &config.Config{
		ModulesPath: modulesPath,
		DistPath:    distPath,
		Compile:     config.NewDefaultCompileConfig(),
		Server:      config.NewDefaultServerConfig(),
		FrontendEnv: map[string]any{},
		BackendEnv:  map[string]any{},
		Log:         config.NewDefaultLogConfig(),
		Db:          config.NewDefaultDbConfig(),
		Auth:        config.NewDefaultAuthConfig(),
		Task:        config.NewDefaultTaskConfig(),
	}

	if err := tenv.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}, &meta.IrComponent{}, &meta.IrUiResource{}, &meta.IrUiResourceMenuRoute{}, &meta.IrUiResourceRouteAction{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}

	module := &meta.IrModule{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "m_auth_conflict", Valid: true}},
		Name:      "auth",
		Path:      modulePath,
		Status:    meta.Installed,
	}

	prebuildPlugin := defaultesbplugins.NewWebPrebuildPlugin(tenv, module, entryPoint)
	buildPlugin := defaultesbplugins.NewWebPrebuildPlugin(tenv, module, entryPoint)
	b := NewWebBuilder(
		tenv,
		nil,
		module,
		entryPoint,
		WithPublishDist(false),
		WithPrebuildPlugin(prebuildPlugin),
		WithBuildPlugin(buildPlugin),
	)

	_, err := b.Build()
	if err == nil {
		t.Fatalf("expected build to fail with UI_VAL_002 for conflicting duplicate model actions")
	}
	if !strings.Contains(err.Error(), "[UI_VAL_002]") {
		t.Fatalf("expected UI_VAL_002 in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "auth.route.session_list") {
		t.Fatalf("expected conflicting route id in error, got: %v", err)
	}
}

func TestExtractUiResources_DiagnosticIncludesLocationAndHintForDuplicate(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu, Title: "User List", SourcePath: "/modules/auth/web/menu/menus.ts", SourceLine: 12, SourceColumn: 5},
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu, Title: "Users", SourcePath: "/modules/auth/web/menu/override.ts", SourceLine: 20, SourceColumn: 9},
	}}

	_, _, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err == nil {
		t.Fatalf("expected duplicate id error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "[UI_VAL_002]") {
		t.Fatalf("expected rule code in diagnostic, got: %s", msg)
	}
	if !strings.Contains(msg, "/modules/auth/web/menu/override.ts:20:9") {
		t.Fatalf("expected source location in diagnostic, got: %s", msg)
	}
	if !strings.Contains(msg, "hint:") {
		t.Fatalf("expected remediation hint in diagnostic, got: %s", msg)
	}
}

func TestTsParser_WarnsForNonRecommendedResourceIDNaming(t *testing.T) {
	testRuntimeScope := newTestScope()
	p := defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Path: "/virtual/modules/auth"})

	path := "/virtual/modules/auth/web/route/routes.ts"
	content := `
import { defineRoute } from '@/core/web/app/resource';

export const userListRoute = defineRoute('UserList', {
  path: '/auth/users',
	meta: { pageTitle: 'User List' }
});
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(r.UiResourceDecls) != 1 {
		t.Fatalf("expected 1 ui resource decl, got %d", len(r.UiResourceDecls))
	}
	if len(r.UiResourceDeclIssues) == 0 {
		t.Fatalf("expected parser warning for non-recommended id naming")
	}
	issue := r.UiResourceDeclIssues[0]
	if issue.Severity != parser.UiResourceIssueSeverityWarning {
		t.Fatalf("expected warning severity, got %s", issue.Severity)
	}
	if issue.Code != parser.UiResourceIssueCodeDeclIDNamingSuggested {
		t.Fatalf("expected issue code %s, got %s", parser.UiResourceIssueCodeDeclIDNamingSuggested, issue.Code)
	}
	if !strings.Contains(issue.Message, "recommended naming form") {
		t.Fatalf("unexpected issue message: %s", issue.Message)
	}

	resources, warnings, err := extractUiResources(&meta.IrModule{Name: "auth"}, []*parser.ParserResult{r})
	if err != nil {
		t.Fatalf("expected extractor to accept non-recommended id naming, got: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if len(warnings) == 0 {
		t.Fatalf("expected propagated warning")
	}
	if warnings[0].code != string(parser.UiResourceIssueCodeDeclIDNamingSuggested) {
		t.Fatalf("expected warning code %s, got %s", parser.UiResourceIssueCodeDeclIDNamingSuggested, warnings[0].code)
	}
}

func TestExtractUiResources_OverrideTypeChangeFails(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu},
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeRoute, Override: true},
	}}

	_, _, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err == nil {
		t.Fatalf("expected override type change error")
	}
}

func TestExtractUiResources_OverrideSameIdentityPasses(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu, Title: "A", Sequence: 10},
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu, Title: "B", Sequence: 20, Override: true},
	}}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("expected override with same identity to pass, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].Title != "B" || resources[0].Sequence != 20 {
		t.Fatalf("expected overridden fields to take effect, got title=%q sequence=%d", resources[0].Title, resources[0].Sequence)
	}
}

func TestExtractUiResources_DanglingParentWarnsAndFallsBack(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "auth.route.user_list", Type: parser.UiResourceTypeRoute, Path: "/auth/users"},
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu, Path: "/auth/users", ParentMenu: "auth.menu.missing"},
	}}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("extract ui resources failed: %v", err)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected dangling parent warning")
	}
	if !strings.Contains(warnings[0].message, "[UI_VAL_005]") {
		t.Fatalf("expected warning code UI_VAL_005, got: %s", warnings[0].message)
	}
	if warnings[0].code != "UI_VAL_005" {
		t.Fatalf("expected structured warning code UI_VAL_005, got: %s", warnings[0].code)
	}
	if strings.TrimSpace(warnings[0].hint) == "" {
		t.Fatalf("expected structured warning hint")
	}
	if !strings.Contains(warnings[0].message, "hint:") {
		t.Fatalf("expected warning hint in message, got: %s", warnings[0].message)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	for _, r := range resources {
		if r.Name == "auth.menu.user_list" && r.ParentResourceName != "" {
			t.Fatalf("expected dangling parent to be cleared, got %q", r.ParentResourceName)
		}
	}
}

func TestExtractUiResources_ParserWarningCarriesCodeAndHint(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{
		UiResourceDecls: []*parser.UiResourceDecl{
			{ID: "auth.route.user_list", Type: parser.UiResourceTypeRoute, SourcePath: "/modules/auth/web/route/routes.ts", SourceLine: 10, SourceColumn: 2},
		},
		UiResourceDeclIssues: []*parser.UiResourceDeclIssue{
			{
				Severity:   parser.UiResourceIssueSeverityWarning,
				Code:       parser.UiResourceIssueCodeDeclRequiresNotLiteral,
				Factory:    "defineRoute",
				ResourceID: "auth.route.user_list",
				Message:    "requires must be an object-literal array like [{ model: 'auth.User' }] or [{ model: 'auth.User', method: 'Browse' }]",
				SourcePath: "/modules/auth/web/route/routes.ts",
				Line:       11,
				Column:     6,
			},
		},
	}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("extract ui resources failed: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if len(warnings) == 0 {
		t.Fatalf("expected parser warning to be propagated")
	}
	if warnings[0].code != string(parser.UiResourceIssueCodeDeclRequiresNotLiteral) {
		t.Fatalf("expected warning code %s, got %s", parser.UiResourceIssueCodeDeclRequiresNotLiteral, warnings[0].code)
	}
	if strings.TrimSpace(warnings[0].hint) == "" {
		t.Fatalf("expected propagated warning hint")
	}
	if !strings.Contains(warnings[0].message, "hint:") {
		t.Fatalf("expected warning message to include hint, got: %s", warnings[0].message)
	}
}

func TestExtractUiResources_MenuPathMustMatchRoute(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "auth.route.user_list", Type: parser.UiResourceTypeRoute, Path: "/auth/users"},
		{ID: "auth.menu.user_list", Type: parser.UiResourceTypeMenu, Path: "/auth/unknown"},
	}}

	_, _, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err == nil {
		t.Fatalf("expected menu path mismatch error")
	}
	if !strings.Contains(err.Error(), "[UI_VAL_006]") {
		t.Fatalf("expected UI_VAL_006 in error, got: %v", err)
	}
}

func TestExtractUiResources_ExternalLeafMenuDoesNotRequireRouteMatch(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "auth.menu.docs", Type: parser.UiResourceTypeMenu, Path: "https://docs.choysum.example"},
	}}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("expected external leaf menu to pass, got error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
}

func TestExtractUiResources_ActionWithRequiresCannotDeclareDefaultRoles(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{
			ID:           "auth.action.user_export",
			Type:         parser.UiResourceTypeAction,
			Requires:     []string{"rpc:/auth.User/Export"},
			DefaultRoles: []string{"base.user"},
		},
	}}

	_, _, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err == nil {
		t.Fatalf("expected action defaultRoles validation error")
	}
}

func TestExtractUiResources_ActionWithoutRequiresCanDeclareDefaultRoles(t *testing.T) {
	module := &meta.IrModule{Name: "auth"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{
			ID:           "auth.action.user_tour",
			Type:         parser.UiResourceTypeAction,
			Requires:     []string{},
			DefaultRoles: []string{"base.user"},
		},
	}}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("expected action defaultRoles without requires to pass, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
}

func TestExtractUiResources_MenuAndRoutePreserveBaselineDefaultRoles(t *testing.T) {
	module := &meta.IrModule{Name: "web"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{
			ID:           "web.menu.home",
			Type:         parser.UiResourceTypeMenu,
			Path:         "/home",
			DefaultRoles: []string{"base.user"},
		},
		{
			ID:           "web.route.home",
			Type:         parser.UiResourceTypeRoute,
			Path:         "home",
			DefaultRoles: []string{"base.user"},
		},
	}}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("expected baseline menu/route defaultRoles to pass, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}

	defaultRolesByName := map[string][]string{}
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		defaultRolesByName[resource.Name] = parseJSONStrings(resource.DefaultRoles)
	}

	if !slices.Equal(defaultRolesByName["web.menu.home"], []string{"base.user"}) {
		t.Fatalf("unexpected menu defaultRoles: %v", defaultRolesByName["web.menu.home"])
	}
	if !slices.Equal(defaultRolesByName["web.route.home"], []string{"base.user"}) {
		t.Fatalf("unexpected route defaultRoles: %v", defaultRolesByName["web.route.home"])
	}

	menuRoutes, routeActions, err := extractUiResourceRelations(resources, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("extract ui resource relations failed: %v", err)
	}
	if len(routeActions) != 0 {
		t.Fatalf("expected no route actions, got %d", len(routeActions))
	}
	if len(menuRoutes) != 1 {
		t.Fatalf("expected 1 menu_route relation, got %d", len(menuRoutes))
	}
	if menuRoutes[0].MenuName != "web.menu.home" || menuRoutes[0].RouteName != "web.route.home" {
		t.Fatalf("unexpected menu_route relation: %#v", menuRoutes[0])
	}
}

func TestExtractUiResources_InjectsApplicationFromModuleOwnership(t *testing.T) {
	module := &meta.IrModule{
		Name:           "sale_marketing",
		ApplicationId:  sql.NullString{String: "app_sale_001", Valid: true},
		ApplicationStr: "sale",
	}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "whatever-id", Type: parser.UiResourceTypeMenu, SourcePath: "/modules/sale_marketing/web/menu/menus.ts", SourceLine: 9, SourceColumn: 3},
	}}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("expected module ownership injection to pass, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(warnings))
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].IrApplicationId != "app_sale_001" {
		t.Fatalf("expected module application id to be injected, got %q", resources[0].IrApplicationId)
	}
}

func TestExtractUiResources_CrossModuleAppendPassesWithoutNamespaceProtocol(t *testing.T) {
	module := &meta.IrModule{Name: "sale_marketing", ApplicationStr: "sale"}

	pr := &parser.ParserResult{UiResourceDecls: []*parser.UiResourceDecl{
		{ID: "dashboard-route", Type: parser.UiResourceTypeRoute, Path: "/sale/marketing/dashboard"},
		{
			ID:         "marketing-menu",
			Type:       parser.UiResourceTypeMenu,
			ParentMenu: "base.menu.settings",
			Path:       "/sale/marketing/dashboard",
		},
	}}

	resources, warnings, err := extractUiResources(module, []*parser.ParserResult{pr})
	if err != nil {
		t.Fatalf("expected same-application append to pass, got: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if len(warnings) == 0 {
		t.Fatalf("expected dangling parent warning due external parent id")
	}
	if warnings[0].code != "UI_VAL_005" {
		t.Fatalf("expected UI_VAL_005 warning, got %s", warnings[0].code)
	}
	for _, resource := range resources {
		if resource.IrApplicationId != "sale" {
			t.Fatalf("expected module application fallback to be injected, got %q", resource.IrApplicationId)
		}
	}
}

func TestCollectUiResourceDefaultRoleRows_BuildsDistinctRows(t *testing.T) {
	uiResources := []*meta.IrUiResource{
		{
			BaseModel: meta.BaseModel{Id: sql.NullString{String: "ui_1", Valid: true}},
			Name:      "auth.menu.root",
			DefaultRoles: []byte(`[
"base.user",
"base.user",
"sys.admin"
]`),
		},
		{
			BaseModel:    meta.BaseModel{Id: sql.NullString{String: "ui_2", Valid: true}},
			Name:         "auth.route.home",
			DefaultRoles: []byte(`["base.user"]`),
		},
	}

	rows, err := collectUiResourceDefaultRoleRows(uiResources, map[string]string{
		"base.user": "role_base_user",
		"sys.admin": "role_sys_admin",
	})
	if err != nil {
		t.Fatalf("collectUiResourceDefaultRoleRows returned error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	pairSet := map[string]bool{}
	for _, row := range rows {
		pairSet[row.RoleId.String+"/"+row.IrUiResourceId.String] = true
		if strings.TrimSpace(row.Mode) != "allow" {
			t.Fatalf("expected defaultRoles seed mode allow, got %q", row.Mode)
		}
	}

	if !pairSet["role_base_user/ui_1"] {
		t.Fatalf("expected base.user grant for ui_1")
	}
	if !pairSet["role_sys_admin/ui_1"] {
		t.Fatalf("expected sys.admin grant for ui_1")
	}
	if !pairSet["role_base_user/ui_2"] {
		t.Fatalf("expected base.user grant for ui_2")
	}
}

func TestCollectUiResourceDefaultRoleRows_MissingRoleFails(t *testing.T) {
	uiResources := []*meta.IrUiResource{
		{
			BaseModel:    meta.BaseModel{Id: sql.NullString{String: "ui_1", Valid: true}},
			Name:         "auth.menu.root",
			DefaultRoles: []byte(`["base.user"]`),
		},
	}

	_, err := collectUiResourceDefaultRoleRows(uiResources, map[string]string{})
	if err == nil {
		t.Fatalf("expected missing role error")
	}
	if !strings.Contains(err.Error(), `defaultRoles role "base.user" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTemplateStyleAndRenderComponentChain(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	parentPath := "/virtual/modules/test/web/base/BaseView.vue"
	childPath := "/virtual/modules/test/web/views/ChildView.vue"

	parentSFC := `<template><div id="root"><p id="target">base</p></div></template>
<script lang="ts" _name="BaseView">
import { defineComponent } from 'vue';
import LegacyView from './LegacyView.vue';

export default defineComponent({
  name: 'BaseView',
  extends: LegacyView,
});
</script>
<style scoped>@import "./base.css";
.base { color: red; }</style>
<style>.shared { background: url("./img/bg.png"); }</style>`

	childSFC := `<template><xpath expr="//*[@id='target']" position="replace"><strong id="new">child</strong></xpath></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from '../base/BaseView.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
});
</script>
<style scoped>.child { color: blue; }</style>
<style module>.module { display: block; }</style>`

	parentParsed := parseVueResult(t, testRuntimeScope, parentPath, parentSFC)
	childParsed := parseVueResult(t, testRuntimeScope, childPath, childSFC)

	templateNode, err := b.getTemplateNode(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getTemplateNode failed: %v", err)
	}
	if htmlquery.FindOne(templateNode, "//*[@id='new']") == nil {
		t.Fatalf("expected xpath replacement node in merged template")
	}
	if htmlquery.FindOne(templateNode, "//*[@id='target']") != nil {
		t.Fatalf("expected target node to be replaced in merged template")
	}

	styleNodes, err := b.getStyleNodes(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getStyleNodes failed: %v", err)
	}
	if len(styleNodes) != 3 {
		t.Fatalf("expected 3 merged style nodes, got %d", len(styleNodes))
	}
	var foundMergedScoped bool
	var foundShared bool
	var foundModule bool
	for _, node := range styleNodes {
		content := htmlquery.InnerText(node)
		attrs := map[string]string{}
		for _, attr := range node.Attr {
			attrs[attr.Key] = attr.Val
		}
		if _, ok := attrs["scoped"]; ok {
			foundMergedScoped = strings.Contains(content, ".child { color: blue; }") && strings.Contains(content, "../base/base.css")
		}
		if _, ok := attrs["module"]; ok && strings.Contains(content, ".module { display: block; }") {
			foundModule = true
		}
		if strings.Contains(content, "../base/img/bg.png") {
			foundShared = true
		}
	}
	if !foundMergedScoped || !foundShared || !foundModule {
		t.Fatalf("unexpected merged style nodes: mergedScoped=%v shared=%v module=%v", foundMergedScoped, foundShared, foundModule)
	}

	scriptNode, err := b.getScriptNode(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getScriptNode failed: %v", err)
	}
	childParsed.ScriptNode = scriptNode
	childParsed.TemplateNode = templateNode
	childParsed.StyleNodes = styleNodes

	rendered, err := b.renderComponent(childParsed)
	if err != nil {
		t.Fatalf("renderComponent failed: %v", err)
	}
	for _, want := range []string{"<strong id=\"new\">child</strong>", "../base/base.css", "../base/img/bg.png", ".module { display: block; }"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered component to contain %q, got:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "<xpath") {
		t.Fatalf("expected xpath directives to be consumed during render, got:\n%s", rendered)
	}

	t.Run("renderComponent fails without raw script node", func(t *testing.T) {
		_, err := b.renderComponent(&parser.ParserResult{})
		if err == nil || !strings.Contains(err.Error(), "script node node is nil") {
			t.Fatalf("expected missing raw script node error, got %v", err)
		}
	})
}

func TestGetStyleNodes_HoistsSassModuleDirectivesAfterPathRewrite(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	parentPath := "/virtual/modules/test/web/base/BaseView.vue"
	childPath := "/virtual/modules/test/web/views/ChildView.vue"

	parentSFC := `<template><div/></template>
<script lang="ts" _name="BaseView">
import { defineComponent } from 'vue';
import LegacyView from './LegacyView.vue';

export default defineComponent({
  name: 'BaseView',
  extends: LegacyView,
});
</script>
<style scoped lang="scss">@charset "UTF-8";
@use "./theme.scss" as *;
@forward "./tokens.scss";
.base { color: red; }</style>
<style>.shared { background: url("./img/bg.png"); }</style>`

	childSFC := `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from '../base/BaseView.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
});
</script>
<style scoped lang="scss">@use "../base/theme.scss" as *;
@forward "../base/tokens.scss";
.child { color: blue; }</style>
<style scoped lang="scss">@use "./local.scss" as *;
.child-2 { color: green; }</style>`

	parentParsed := parseVueResult(t, testRuntimeScope, parentPath, parentSFC)
	childParsed := parseVueResult(t, testRuntimeScope, childPath, childSFC)

	styleNodes, err := b.getStyleNodes(childParsed, parentParsed)
	if err != nil {
		t.Fatalf("getStyleNodes failed: %v", err)
	}
	if len(styleNodes) != 2 {
		t.Fatalf("expected 2 merged style nodes, got %d", len(styleNodes))
	}

	firstContent := htmlquery.InnerText(styleNodes[0])
	secondContent := htmlquery.InnerText(styleNodes[1])
	if !strings.Contains(firstContent, `@charset "UTF-8";`) {
		t.Fatalf("expected merged scoped style to keep charset directive, got:\n%s", firstContent)
	}
	if strings.Count(firstContent, `@use "../base/theme.scss" as *;`) != 1 {
		t.Fatalf("expected merged scoped style to dedupe rewritten @use directive, got:\n%s", firstContent)
	}
	if strings.Count(firstContent, `@forward "../base/tokens.scss";`) != 1 {
		t.Fatalf("expected merged scoped style to dedupe rewritten @forward directive, got:\n%s", firstContent)
	}
	for _, want := range []string{
		`@use "./local.scss" as *;`,
		`.base { color: red; }`,
		`.child { color: blue; }`,
		`.child-2 { color: green; }`,
	} {
		if !strings.Contains(firstContent, want) {
			t.Fatalf("expected merged scoped style to contain %q, got:\n%s", want, firstContent)
		}
	}
	charsetIdx := strings.Index(firstContent, `@charset "UTF-8";`)
	useIdx := strings.Index(firstContent, `@use "../base/theme.scss" as *;`)
	forwardIdx := strings.Index(firstContent, `@forward "../base/tokens.scss";`)
	localUseIdx := strings.Index(firstContent, `@use "./local.scss" as *;`)
	bodyIdx := strings.Index(firstContent, `.base { color: red; }`)
	if !(charsetIdx != -1 && useIdx > charsetIdx && forwardIdx > useIdx && localUseIdx > forwardIdx && bodyIdx > localUseIdx) {
		t.Fatalf("expected hoisted directives to appear before merged body content, got:\n%s", firstContent)
	}
	if !strings.Contains(secondContent, `../base/img/bg.png`) {
		t.Fatalf("expected shared style url to be rewritten with stable ordering, got:\n%s", secondContent)
	}
}

func TestGetTemplateNode_UsesRenderedParentAndWrapsErrors(t *testing.T) {
	testRuntimeScope := newTestScope()
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	t.Run("prefers rendered parent template node when available", func(t *testing.T) {
		parentRaw := parseVueResult(t, testRuntimeScope, "/virtual/modules/test/web/base/BaseView.vue", `<template><section id="raw-target"><p>raw</p></section></template>
<script lang="ts" _name="BaseView">
import { defineComponent } from 'vue';
import LegacyView from './LegacyView.vue';

export default defineComponent({
  name: 'BaseView',
  extends: LegacyView,
});
</script>`)
		parentRendered := parseVueResult(t, testRuntimeScope, "/virtual/modules/test/web/base/BaseViewRendered.vue", `<template><section id="target"><p>rendered</p></section></template>
<script lang="ts" _name="BaseViewRendered">
import { defineComponent } from 'vue';
import LegacyView from './LegacyView.vue';

export default defineComponent({
  name: 'BaseViewRendered',
  extends: LegacyView,
});
</script>`)
		childParsed := parseVueResult(t, testRuntimeScope, "/virtual/modules/test/web/views/ChildView.vue", `<template><xpath expr="//*[@id='target']" position="replace"><strong id="new">child</strong></xpath></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from '../base/BaseView.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
});
</script>`)

		parentRaw.TemplateNode = parentRendered.RawTemplateNode

		templateNode, err := b.getTemplateNode(childParsed, parentRaw)
		if err != nil {
			t.Fatalf("getTemplateNode failed: %v", err)
		}
		if htmlquery.FindOne(templateNode, "//*[@id='new']") == nil {
			t.Fatalf("expected xpath replacement to use rendered parent template")
		}
		if htmlquery.FindOne(templateNode, "//*[@id='raw-target']") != nil {
			t.Fatalf("expected raw parent template to be ignored when TemplateNode is set")
		}
	})

	t.Run("wraps xpath application errors", func(t *testing.T) {
		parentParsed := parseVueResult(t, testRuntimeScope, "/virtual/modules/test/web/base/BaseView.vue", `<template><div id="target">base</div></template>
<script lang="ts" _name="BaseView">
import { defineComponent } from 'vue';
import LegacyView from './LegacyView.vue';

export default defineComponent({
  name: 'BaseView',
  extends: LegacyView,
});
</script>`)
		childParsed := parseVueResult(t, testRuntimeScope, "/virtual/modules/test/web/views/ChildView.vue", `<template><xpath expr="//*[" position="replace"><strong id="new">child</strong></xpath></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from '../base/BaseView.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
});
</script>`)

		if _, err := b.getTemplateNode(childParsed, parentParsed); err == nil || !strings.Contains(err.Error(), "Error applying XPath to template") {
			t.Fatalf("expected wrapped xpath error, got %v", err)
		}
	})
}

func TestGetNewExtendsPrefersConnectedInMemoryCandidate(t *testing.T) {
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	if err := testRuntimeScope.db.AutoMigrate(&meta.IrComponent{}); err != nil {
		t.Fatalf("auto migrate components failed: %v", err)
	}

	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}
	if got, err := b.getNewExtends(nil, &meta.IrComponent{Name: "DemoView", Path: "/child.vue"}); err != nil || got != nil {
		t.Fatalf("getNewExtends(no extends) = %#v, %v, want nil, nil", got, err)
	}

	base := &meta.IrComponent{Name: "DemoView", Path: "/base.vue"}
	legacy := &meta.IrComponent{Name: "DemoView", Path: "/legacy.vue", Extends: "/elsewhere.vue"}
	for _, comp := range []*meta.IrComponent{base, legacy} {
		if err := testRuntimeScope.db.Create(comp).Error; err != nil {
			t.Fatalf("create component %s: %v", comp.Path, err)
		}
	}

	inMemory := &meta.IrComponent{Name: "DemoView", Path: "/mid.vue", Extends: "/base.vue"}
	buildResult := withParserResults(&module.BuildResult{}, &parser.ParserResult{
		Path:         inMemory.Path,
		Content:      "rendered",
		VueComponent: inMemory,
	})

	next, err := b.getNewExtends(buildResult, &meta.IrComponent{Name: "DemoView", Path: "/child.vue", Extends: "/base.vue"})
	if err != nil {
		t.Fatalf("getNewExtends returned error: %v", err)
	}
	if next == nil || next.Path != "/mid.vue" {
		t.Fatalf("expected connected in-memory component to be selected, got %#v", next)
	}

	next, err = b.getNewExtends(nil, &meta.IrComponent{Name: "DemoView", Path: "/child.vue", Extends: "/unknown.vue"})
	if err != nil {
		t.Fatalf("getNewExtends(disconnected) returned error: %v", err)
	}
	if next != nil {
		t.Fatalf("expected no candidate for disconnected extends chain, got %#v", next)
	}
}

func TestValidateUiResourceDependencies(t *testing.T) {
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	if err := testRuntimeScope.db.AutoMigrate(&meta.IrModel{}, &meta.IrService{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	t.Run("invalid requires entry fails fast", func(t *testing.T) {
		err := b.validateUiResourceDependencies([]*meta.IrUiResource{{
			Name:     "menu.invalid",
			Requires: []byte(`["oops"]`),
		}})
		if err == nil || !strings.Contains(err.Error(), `resource menu.invalid has invalid requires entry "oops"`) {
			t.Fatalf("expected invalid requires error, got %v", err)
		}
	})

	t.Run("missing model fails", func(t *testing.T) {
		err := b.validateUiResourceDependencies([]*meta.IrUiResource{{
			Name:     "route.missing-model",
			Requires: []byte(`["rpc:/crm.Lead/read"]`),
		}})
		if err == nil || !strings.Contains(err.Error(), `referenced model "crm.Lead" not found`) {
			t.Fatalf("expected missing model error, got %v", err)
		}
	})

	model := &meta.IrModel{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: "model_lead", Valid: true}},
		Application: "crm",
		Name:        "Lead",
		Path:        "/virtual/modules/crm/models/lead.ts",
	}
	if err := testRuntimeScope.db.Create(model).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	if err := testRuntimeScope.db.Create(&meta.IrService{
		BaseModel: meta.BaseModel{Id: sql.NullString{String: "svc_read", Valid: true}},
		ModelId:   sql.NullString{String: "model_lead", Valid: true},
		Name:      "Read",
	}).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}

	t.Run("missing method fails", func(t *testing.T) {
		err := b.validateUiResourceDependencies([]*meta.IrUiResource{{
			Name:     "action.missing-method",
			Requires: []byte(`["rpc:/crm.Lead/write"]`),
		}})
		if err == nil || !strings.Contains(err.Error(), `resource action.missing-method requires "rpc:/crm.Lead/write" but service method not found`) {
			t.Fatalf("expected missing service method error, got %v", err)
		}
	})

	t.Run("known method and wildcard pass", func(t *testing.T) {
		resources := []*meta.IrUiResource{
			{Name: "route.read", Requires: []byte(`["rpc:/crm.Lead/read"]`)},
			{Name: "route.any", Requires: []byte(`["rpc:/crm.Lead/*"]`)},
		}
		if err := b.validateUiResourceDependencies(resources); err != nil {
			t.Fatalf("expected valid dependencies, got %v", err)
		}
	})
}

func TestReplaceUiResourceRelationsReplacesAndDedupesRows(t *testing.T) {
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	if err := testRuntimeScope.db.AutoMigrate(&meta.IrUiResource{}, &meta.IrUiResourceMenuRoute{}, &meta.IrUiResourceRouteAction{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	resources := []*meta.IrUiResource{
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "menu_1", Valid: true}}, Name: "menu.root", Type: meta.UiResourceTypeMenu},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "route_1", Valid: true}}, Name: "route.home", Type: meta.UiResourceTypeRoute},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "action_1", Valid: true}}, Name: "action.save", Type: meta.UiResourceTypeAction},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "other_1", Valid: true}}, Name: "other.misc", Type: meta.UiResourceTypeAction},
	}
	for _, resource := range resources {
		if err := testRuntimeScope.db.Create(resource).Error; err != nil {
			t.Fatalf("create resource %s: %v", resource.Name, err)
		}
	}
	if err := testRuntimeScope.db.Create(&meta.IrUiResourceMenuRoute{
		MenuUiResourceId:  sql.NullString{String: "menu_1", Valid: true},
		RouteUiResourceId: sql.NullString{String: "other_1", Valid: true},
	}).Error; err != nil {
		t.Fatalf("seed menu route relation: %v", err)
	}
	if err := testRuntimeScope.db.Create(&meta.IrUiResourceRouteAction{
		RouteUiResourceId:  sql.NullString{String: "route_1", Valid: true},
		ActionUiResourceId: sql.NullString{String: "other_1", Valid: true},
	}).Error; err != nil {
		t.Fatalf("seed route action relation: %v", err)
	}

	err := b.replaceUiResourceRelations(
		[]string{"menu_1", "route_1", "action_1"},
		[]uiResourceMenuRouteRef{{MenuName: "menu.root", RouteName: "route.home"}, {MenuName: "menu.root", RouteName: "route.home"}, {MenuName: "route.home", RouteName: "menu.root"}},
		[]uiResourceRouteActionRef{{RouteName: "route.home", ActionName: "action.save"}, {RouteName: "route.home", ActionName: "action.save"}, {RouteName: "menu.root", ActionName: "action.save"}},
	)
	if err != nil {
		t.Fatalf("replaceUiResourceRelations returned error: %v", err)
	}

	var menuRoutes []meta.IrUiResourceMenuRoute
	if err := testRuntimeScope.db.Order("menu_ui_resource_id, route_ui_resource_id").Find(&menuRoutes).Error; err != nil {
		t.Fatalf("query menu route rows: %v", err)
	}
	if len(menuRoutes) != 1 || menuRoutes[0].MenuUiResourceId.String != "menu_1" || menuRoutes[0].RouteUiResourceId.String != "route_1" {
		t.Fatalf("unexpected menu route rows: %#v", menuRoutes)
	}

	var routeActions []meta.IrUiResourceRouteAction
	if err := testRuntimeScope.db.Order("route_ui_resource_id, action_ui_resource_id").Find(&routeActions).Error; err != nil {
		t.Fatalf("query route action rows: %v", err)
	}
	if len(routeActions) != 1 || routeActions[0].RouteUiResourceId.String != "route_1" || routeActions[0].ActionUiResourceId.String != "action_1" {
		t.Fatalf("unexpected route action rows: %#v", routeActions)
	}

	if err := b.deleteUiResourceRelationsByIDs(nil); err != nil {
		t.Fatalf("deleteUiResourceRelationsByIDs(nil) error = %v", err)
	}
}

func TestPersistUiResourceDefaultRolesInsertsAndDedupes(t *testing.T) {
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	mustExec(t, testRuntimeScope.db, `CREATE TABLE auth_role (id TEXT PRIMARY KEY, code TEXT)`)
	mustExec(t, testRuntimeScope.db, `CREATE TABLE auth_role_ui_resource (id TEXT, role_id TEXT, mode TEXT, ir_application_id TEXT, ir_ui_resource_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`)
	mustExec(t, testRuntimeScope.db, `CREATE UNIQUE INDEX idx_auth_role_ui_resource_pair ON auth_role_ui_resource(role_id, ir_ui_resource_id)`)
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	if err := testRuntimeScope.db.Exec(`INSERT INTO auth_role (id, code) VALUES (?, ?), (?, ?)`, "role_user", "base.user", "role_admin", "sys.admin").Error; err != nil {
		t.Fatalf("seed auth_role failed: %v", err)
	}
	resources := []*meta.IrUiResource{
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "ui_1", Valid: true}}, Name: "menu.root", DefaultRoles: []byte(`["base.user","base.user","sys.admin"]`)},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "ui_2", Valid: true}}, Name: "route.home", DefaultRoles: []byte(`["base.user"]`)},
	}

	if err := b.persistUiResourceDefaultRoles(resources); err != nil {
		t.Fatalf("persistUiResourceDefaultRoles returned error: %v", err)
	}
	if err := b.persistUiResourceDefaultRoles(resources); err != nil {
		t.Fatalf("persistUiResourceDefaultRoles second call returned error: %v", err)
	}

	var rows []roleUiResourceGrantRow
	if err := testRuntimeScope.db.Table("auth_role_ui_resource").Order("role_id, ir_ui_resource_id").Find(&rows).Error; err != nil {
		t.Fatalf("query auth_role_ui_resource failed: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 distinct persisted default role rows, got %d (%#v)", len(rows), rows)
	}
	seen := map[string]bool{}
	for _, row := range rows {
		seen[row.RoleId.String+"/"+row.IrUiResourceId.String] = true
		if row.Mode != "allow" {
			t.Fatalf("expected allow mode, got %#v", row)
		}
	}
	for _, key := range []string{"role_admin/ui_1", "role_user/ui_1", "role_user/ui_2"} {
		if !seen[key] {
			t.Fatalf("missing persisted default role mapping %s in %#v", key, rows)
		}
	}

	if err := b.persistUiResourceDefaultRoles([]*meta.IrUiResource{{Name: "menu.empty"}}); err != nil {
		t.Fatalf("expected no-defaultRoles fast path to succeed, got %v", err)
	}
}

func TestPersistUiResourceDefaultRolesSkipsWhenAuthTablesMissing(t *testing.T) {
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}
	resources := []*meta.IrUiResource{
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "ui_1", Valid: true}}, Name: "menu.root", DefaultRoles: []byte(`["base.user"]`)},
	}

	if err := b.persistUiResourceDefaultRoles(resources); err != nil {
		t.Fatalf("expected missing auth tables to be skipped, got %v", err)
	}
}

func TestPersistModuleComponentsReplacesAndDedupesByPath(t *testing.T) {
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	if err := testRuntimeScope.db.AutoMigrate(&meta.IrComponent{}); err != nil {
		t.Fatalf("auto migrate components failed: %v", err)
	}
	b := &WebModuleBuilder{runtimeScope: testRuntimeScope}

	seedRows := []*meta.IrComponent{
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "old_mod1", Valid: true}}, Name: "OldComp", Path: "/old.vue", ModuleId: sql.NullString{String: "mod1", Valid: true}},
		{BaseModel: meta.BaseModel{Id: sql.NullString{String: "keep_mod2", Valid: true}}, Name: "KeepComp", Path: "/keep.vue", ModuleId: sql.NullString{String: "mod2", Valid: true}},
	}
	for _, row := range seedRows {
		if err := testRuntimeScope.db.Create(row).Error; err != nil {
			t.Fatalf("seed component %s: %v", row.Path, err)
		}
	}

	if err := b.persistModuleComponents("", []*meta.IrComponent{{Name: "Ignored", Path: "/ignored.vue"}}); err != nil {
		t.Fatalf("persistModuleComponents(blank moduleID) error = %v", err)
	}

	rows := []*meta.IrComponent{
		{Name: "First", Path: "/dup.vue"},
		{Name: "LastWins", Path: "/dup.vue", Extends: "/base.vue"},
		nil,
		{Name: "Other", Path: "/other.vue"},
		{Name: "BlankPath", Path: "   "},
	}
	if err := b.persistModuleComponents("mod1", rows); err != nil {
		t.Fatalf("persistModuleComponents(mod1) error = %v", err)
	}

	var mod1Rows []*meta.IrComponent
	if err := testRuntimeScope.db.Where("module_id = ?", "mod1").Order("path").Find(&mod1Rows).Error; err != nil {
		t.Fatalf("query mod1 components failed: %v", err)
	}
	if len(mod1Rows) != 2 {
		t.Fatalf("expected 2 deduped mod1 rows, got %#v", mod1Rows)
	}
	if mod1Rows[0].Path != "/dup.vue" || mod1Rows[0].Name != "LastWins" || mod1Rows[0].Extends != "/base.vue" {
		t.Fatalf("expected duplicate path to keep last row, got %#v", mod1Rows[0])
	}
	for _, row := range mod1Rows {
		if row.ModuleId.String != "mod1" {
			t.Fatalf("expected module id mod1, got %#v", row)
		}
	}

	var mod2Rows []*meta.IrComponent
	if err := testRuntimeScope.db.Where("module_id = ?", "mod2").Find(&mod2Rows).Error; err != nil {
		t.Fatalf("query mod2 components failed: %v", err)
	}
	if len(mod2Rows) != 1 || mod2Rows[0].Path != "/keep.vue" {
		t.Fatalf("expected unrelated module rows to remain, got %#v", mod2Rows)
	}

	if err := b.persistModuleComponents("mod1", nil); err != nil {
		t.Fatalf("persistModuleComponents(delete only) error = %v", err)
	}
	var afterDelete []*meta.IrComponent
	if err := testRuntimeScope.db.Where("module_id = ?", "mod1").Find(&afterDelete).Error; err != nil {
		t.Fatalf("query mod1 after delete failed: %v", err)
	}
	if len(afterDelete) != 0 {
		t.Fatalf("expected mod1 rows to be deleted on empty input, got %#v", afterDelete)
	}
}

func TestUpdateComponentBranches(t *testing.T) {
	root := t.TempDir()
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	testRuntimeScope.cfg.ModulesPath = root
	if err := testRuntimeScope.db.AutoMigrate(&meta.IrComponent{}); err != nil {
		t.Fatalf("auto migrate components failed: %v", err)
	}
	b := &WebModuleBuilder{
		runtimeScope: testRuntimeScope,
		module:       &meta.IrModule{Name: "test", Path: root},
		parser:       defaultparser.NewVueParser(testRuntimeScope, &meta.IrModule{Name: "test", Path: root}),
	}

	t.Run("missing file returns read error", func(t *testing.T) {
		buildResult := &module.BuildResult{}
		err := b.updateComponent(buildResult, map[string]string{}, filepath.Join(root, "missing.vue"))
		if err == nil || !strings.Contains(err.Error(), "Error reading file") {
			t.Fatalf("expected read error, got %v", err)
		}
	})

	t.Run("existing parsed result without extends copies raw content", func(t *testing.T) {
		buildResult := withParserResults(&module.BuildResult{}, &parser.ParserResult{
			Path:       filepath.Join(root, "plain.vue"),
			RawContent: "<template><div>plain</div></template>",
			VueComponent: &meta.IrComponent{
				Name: "PlainView",
				Path: filepath.Join(root, "plain.vue"),
			},
		})
		if err := b.updateComponent(buildResult, map[string]string{}, filepath.Join(root, "plain.vue")); err != nil {
			t.Fatalf("updateComponent(no extends) error = %v", err)
		}
		results := parserResultsOf(buildResult)
		if got := results[0].Content; got != results[0].RawContent {
			t.Fatalf("expected raw content to be copied, got %q", got)
		}
	})

	t.Run("already updated result is a no-op", func(t *testing.T) {
		buildResult := withParserResults(&module.BuildResult{}, &parser.ParserResult{
			Path:    filepath.Join(root, "done.vue"),
			Content: "ready",
			VueComponent: &meta.IrComponent{
				Name: "DoneView",
				Path: filepath.Join(root, "done.vue"),
			},
		})
		if err := b.updateComponent(buildResult, map[string]string{}, filepath.Join(root, "done.vue")); err != nil {
			t.Fatalf("updateComponent(already updated) error = %v", err)
		}
		results := parserResultsOf(buildResult)
		if results[0].Content != "ready" {
			t.Fatalf("expected existing content to remain untouched, got %q", results[0].Content)
		}
	})

	t.Run("loads missing parse result from file and renders extends chain", func(t *testing.T) {
		basePath := filepath.Join(root, "BaseView.vue")
		childPath := filepath.Join(root, "ChildView.vue")
		baseSFC := `<template><div><p id="target">base</p></div></template>
<script lang="ts" _name="BaseView">
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'BaseView',
});
</script>
<style>.base { color: red; }</style>`
		childSFC := `<template><xpath expr="//*[@id='target']" position="replace"><span id="child">child</span></xpath></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import BaseView from './BaseView.vue';

export default defineComponent({
  name: 'ChildView',
  extends: BaseView,
});
</script>
<style>.child { color: blue; }</style>`
		for path, content := range map[string]string{basePath: baseSFC, childPath: childSFC} {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatalf("write %s: %v", path, err)
			}
		}

		childParsed, err := b.parser.Parse(map[string]string{}, childPath, childSFC)
		if err != nil {
			t.Fatalf("parse child: %v", err)
		}
		buildResult := withParserResults(&module.BuildResult{}, childParsed)

		if err := b.updateComponent(buildResult, map[string]string{}, childPath); err != nil {
			t.Fatalf("updateComponent(recursive) error = %v", err)
		}
		results := parserResultsOf(buildResult)
		if len(results) < 2 {
			t.Fatalf("expected parent parse result to be loaded, got %#v", results)
		}
		if childParsed.Content == "" {
			t.Fatal("expected child content to be rendered")
		}
		for _, want := range []string{"id=\"child\"", ".base { color: red; }", ".child { color: blue; }"} {
			if !strings.Contains(childParsed.Content, want) {
				t.Fatalf("expected rendered child content to contain %q, got:\n%s", want, childParsed.Content)
			}
		}
		if childParsed.VueComponent == nil || childParsed.RawTemplateNode == nil || childParsed.RawScriptNode == nil {
			t.Fatalf("expected reparsed child result to refresh parsed nodes, got %#v", childParsed)
		}
	})
}

func TestWebBuilderHelperFunctions(t *testing.T) {
	if got := (roleUiResourceGrantRow{}).TableName(); got != "auth_role_ui_resource" {
		t.Fatalf("TableName() = %q, want %q", got, "auth_role_ui_resource")
	}

	parentCases := []struct {
		name       string
		path       string
		selfID     string
		wantParent string
	}{
		{name: "empty path", path: "", selfID: "child", wantParent: ""},
		{name: "top level self", path: "child/", selfID: "child", wantParent: ""},
		{name: "nested self", path: "root/child/", selfID: "child", wantParent: "root"},
		{name: "path without self suffix", path: "root/child", selfID: "leaf", wantParent: "child"},
	}
	for _, tc := range parentCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parentIDFromPath(tc.path, tc.selfID); got != tc.wantParent {
				t.Fatalf("parentIDFromPath(%q, %q) = %q, want %q", tc.path, tc.selfID, got, tc.wantParent)
			}
		})
	}

	if got, ok := normalizeModelKey(" auth.User "); !ok || got != "auth.User" {
		t.Fatalf("normalizeModelKey(valid) = %q, %v", got, ok)
	}
	for _, raw := range []string{"auth", "auth.", ".User"} {
		if _, ok := normalizeModelKey(raw); ok {
			t.Fatalf("normalizeModelKey(%q) unexpectedly succeeded", raw)
		}
	}

	modelKey, method, ok := parseRpcRequire("service:/auth.User/read")
	if !ok || modelKey != "auth.User" || method != "read" {
		t.Fatalf("parseRpcRequire(service alias) = %q, %q, %v", modelKey, method, ok)
	}
	modelKey, method, ok = parseRpcRequire("rpc:/auth.User/*")
	if !ok || modelKey != "auth.User" || method != "*" {
		t.Fatalf("parseRpcRequire(wildcard) = %q, %q, %v", modelKey, method, ok)
	}
	for _, raw := range []string{"auth.User/read", "rpc:/auth/read", "rpc:/auth.User/"} {
		if _, _, ok := parseRpcRequire(raw); ok {
			t.Fatalf("parseRpcRequire(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestBuildCtxAndBuildToDirCtxRespectCanceledContext(t *testing.T) {
	b := &WebModuleBuilder{
		runtimeScope:       newTestScope(),
		module:             &meta.IrModule{Name: "web", Path: "/virtual/modules/web"},
		publishDist:        false,
		distWebDirOverride: "previous",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := b.BuildCtx(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("BuildCtx(canceled) error = %v, want context.Canceled", err)
	}
	if b.distWebDirOverride != "previous" {
		t.Fatalf("BuildCtx should not mutate override, got %q", b.distWebDirOverride)
	}

	if _, err := b.BuildToDirCtx(ctx, "/tmp/dist/web"); !errors.Is(err, context.Canceled) {
		t.Fatalf("BuildToDirCtx(canceled) error = %v, want context.Canceled", err)
	}
	if b.distWebDirOverride != "previous" {
		t.Fatalf("BuildToDirCtx should restore override, got %q", b.distWebDirOverride)
	}
}

func TestBuildPipelineHelpers(t *testing.T) {
	t.Run("prebuild update build and BuildCtx succeed with fake plugins", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const answer = 42\n")
		if err := testRuntimeScope.db.Create(&meta.IrModule{
			BaseModel:     meta.BaseModel{Id: sql.NullString{String: "installed_auth", Valid: true}},
			Name:          "auth",
			Status:        meta.Installed,
			WebEntryPoint: entryPoint,
		}).Error; err != nil {
			t.Fatalf("seed installed module failed: %v", err)
		}

		preResults := []*parser.ParserResult{{Path: entryPoint, RawContent: "export const answer = 42", Content: "export const answer = 42"}}
		prePlugin := &buildTestPlugin{parserResults: preResults}
		buildPlugin := &buildTestPlugin{}
		builder := &WebModuleBuilder{
			runtimeScope:   testRuntimeScope,
			module:         moduleRef,
			entryPoint:     entryPoint,
			parser:         defaultparser.NewVueParser(testRuntimeScope, moduleRef),
			prebuildPlugin: prePlugin,
			buildPlugin:    buildPlugin,
			publishDist:    false,
		}

		prebuildResult, err := builder.prebuild()
		if err != nil {
			t.Fatalf("prebuild failed: %v", err)
		}
		prebuildParserResults := parserResultsOf(prebuildResult)
		if prebuildResult.Module != moduleRef || len(prebuildParserResults) != 1 {
			t.Fatalf("unexpected prebuild result: %#v", prebuildResult)
		}
		if prePlugin.defineCalls == 0 {
			t.Fatalf("expected prebuild plugin DefinePlugins to be called")
		}

		if err := builder.updatePrebuildResult(prebuildResult); err != nil {
			t.Fatalf("updatePrebuildResult failed: %v", err)
		}
		prebuildParserResults = parserResultsOf(prebuildResult)
		if len(prebuildParserResults[0].VueAppImportTree) != 1 || prebuildParserResults[0].VueAppImportTree[0] != entryPoint {
			t.Fatalf("expected VueAppImportTree to be populated, got %#v", prebuildParserResults[0].VueAppImportTree)
		}

		buildResult, err := builder.build(context.Background(), prebuildResult)
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}
		if buildPlugin.defineCalls == 0 {
			t.Fatalf("expected build plugin DefinePlugins to be called")
		}
		if len(buildPlugin.storedResults) != 1 || buildPlugin.storedResults[0].Path != entryPoint {
			t.Fatalf("expected build plugin to receive prebuild parser results, got %#v", buildPlugin.storedResults)
		}
		buildParserResults := parserResultsOf(buildResult)
		if len(buildParserResults) != 1 || buildParserResults[0].Path != entryPoint {
			t.Fatalf("unexpected build result parser results: %#v", buildParserResults)
		}

		buildCtxPlugin := &buildTestPlugin{parserResults: []*parser.ParserResult{{Path: entryPoint, RawContent: "export const answer = 42", Content: "export const answer = 42"}}}
		buildCtxBuildPlugin := &buildTestPlugin{}
		buildCtxBuilder := &WebModuleBuilder{
			runtimeScope:   testRuntimeScope,
			module:         moduleRef,
			entryPoint:     entryPoint,
			parser:         defaultparser.NewVueParser(testRuntimeScope, moduleRef),
			prebuildPlugin: buildCtxPlugin,
			buildPlugin:    buildCtxBuildPlugin,
			publishDist:    false,
		}

		fullResult, err := buildCtxBuilder.BuildCtx(context.Background())
		if err != nil {
			t.Fatalf("BuildCtx failed: %v", err)
		}
		if fullResult == nil || fullResult.Module != moduleRef || len(parserResultsOf(fullResult)) != 1 {
			t.Fatalf("unexpected BuildCtx result: %#v", fullResult)
		}
		if buildCtxBuildPlugin.defineCalls == 0 {
			t.Fatalf("expected BuildCtx build plugin DefinePlugins to be called")
		}
	})

	t.Run("BuildCtx uses context session for runtime state", func(t *testing.T) {
		baseScope := newTestScopeWithDB(t).(*testScope)
		if err := baseScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("migrate base db: %v", err)
		}

		runtimeDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "webbuilder_runtime.db")), &gorm.Config{})
		if err != nil {
			t.Fatalf("open runtime sqlite: %v", err)
		}
		if err := runtimeDB.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("migrate runtime db: %v", err)
		}

		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, baseScope, "export const answer = 42\n")
		if err := runtimeDB.Create(&meta.IrModule{
			BaseModel:     meta.BaseModel{Id: sql.NullString{String: "installed_base", Valid: true}},
			Name:          "base",
			Status:        meta.Installed,
			WebEntryPoint: "./web/index.ts",
		}).Error; err != nil {
			t.Fatalf("seed runtime installed module: %v", err)
		}

		prebuildPlugin := &buildTestPlugin{parserResults: []*parser.ParserResult{{Path: entryPoint, RawContent: "export const answer = 42", Content: "export const answer = 42"}}}
		buildPlugin := &buildTestPlugin{}
		var parserRuntimeScope scope.Scope
		builder := &WebModuleBuilder{
			runtimeScope:   baseScope,
			module:         moduleRef,
			entryPoint:     entryPoint,
			prebuildPlugin: prebuildPlugin,
			buildPlugin:    buildPlugin,
			publishDist:    false,
			parserFactory: func(runtimeScope scope.Scope, module *meta.IrModule) parser.Parser {
				parserRuntimeScope = runtimeScope
				return fixedParser{}
			},
		}

		runtimeScope := &testScope{ctx: context.Background(), cfg: baseScope.cfg, db: runtimeDB, log: baseScope.log}
		ctx := scope.ContextWithScope(context.Background(), runtimeScope)
		if _, err := builder.BuildCtx(ctx); err != nil {
			t.Fatalf("BuildCtx(runtime scope) error = %v", err)
		}
		if len(prebuildPlugin.entryPointImports) != 1 || prebuildPlugin.entryPointImports[0] != "@/base/web/index" {
			t.Fatalf("prebuild entry imports = %#v, want [@/base/web/index]", prebuildPlugin.entryPointImports)
		}
		if len(buildPlugin.entryPointImports) != 1 || buildPlugin.entryPointImports[0] != "@/base/web/index" {
			t.Fatalf("build entry imports = %#v, want [@/base/web/index]", buildPlugin.entryPointImports)
		}
		if parserRuntimeScope == nil || parserRuntimeScope.Session() == nil || parserRuntimeScope.Session().DB != runtimeDB {
			t.Fatalf("expected runtime parser env to use runtime DB, got %#v", parserRuntimeScope)
		}
	})

	t.Run("BuildToDirCtx preserves runtime transaction when caller context has no transaction", func(t *testing.T) {
		baseScope := newTestScopeWithDB(t).(*testScope)
		if err := baseScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("migrate base db: %v", err)
		}

		runtimeDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "webbuilder_runtime_fallback.db")), &gorm.Config{})
		if err != nil {
			t.Fatalf("open runtime sqlite: %v", err)
		}
		if err := runtimeDB.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("migrate runtime db: %v", err)
		}

		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, baseScope, "export const answer = 42\n")
		if err := runtimeDB.Create(&meta.IrModule{
			BaseModel:     meta.BaseModel{Id: sql.NullString{String: "installed_base_tx", Valid: true}},
			Name:          "base",
			Status:        meta.Installed,
			WebEntryPoint: "./web/index.ts",
		}).Error; err != nil {
			t.Fatalf("seed runtime installed module: %v", err)
		}

		prebuildPlugin := &buildTestPlugin{parserResults: []*parser.ParserResult{{Path: entryPoint, RawContent: "export const answer = 42", Content: "export const answer = 42"}}}
		buildPlugin := &buildTestPlugin{}
		var parserRuntimeScope scope.Scope
		builder := &WebModuleBuilder{
			runtimeScope:   baseScope,
			module:         moduleRef,
			entryPoint:     entryPoint,
			prebuildPlugin: prebuildPlugin,
			buildPlugin:    buildPlugin,
			publishDist:    false,
			parserFactory: func(runtimeScope scope.Scope, module *meta.IrModule) parser.Parser {
				parserRuntimeScope = runtimeScope
				return fixedParser{}
			},
		}

		txSession := &scope.Session{DB: runtimeDB}
		tx := &webBuilderTestTransaction{session: txSession}
		tx.ctx = scope.ContextWithTransaction(context.Background(), tx)
		builder.runtimeScope = &testScope{ctx: tx.ctx, cfg: baseScope.cfg, db: baseScope.db, log: baseScope.log}

		if _, err := builder.BuildToDirCtx(context.Background(), filepath.Join(t.TempDir(), "dist", "web")); err != nil {
			t.Fatalf("BuildToDirCtx(background) error = %v", err)
		}

		if parserRuntimeScope == nil || parserRuntimeScope.Session() == nil || parserRuntimeScope.Session().DB != runtimeDB {
			t.Fatalf("expected runtime parser env to preserve transaction DB, got %#v", parserRuntimeScope)
		}
		if len(prebuildPlugin.entryPointImports) != 1 || prebuildPlugin.entryPointImports[0] != "@/base/web/index" {
			t.Fatalf("prebuild entry imports = %#v, want [@/base/web/index]", prebuildPlugin.entryPointImports)
		}
		if len(buildPlugin.entryPointImports) != 1 || buildPlugin.entryPointImports[0] != "@/base/web/index" {
			t.Fatalf("build entry imports = %#v, want [@/base/web/index]", buildPlugin.entryPointImports)
		}
	})

	t.Run("prebuild wraps esbuild errors", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const = 42\n")
		builder := &WebModuleBuilder{
			runtimeScope:   testRuntimeScope,
			module:         moduleRef,
			entryPoint:     entryPoint,
			prebuildPlugin: &buildTestPlugin{},
			buildPlugin:    &buildTestPlugin{},
			publishDist:    false,
		}

		if _, err := builder.prebuild(); err == nil || !strings.Contains(err.Error(), "Expected identifier") {
			t.Fatalf("expected esbuild syntax error, got %v", err)
		}
	})

	t.Run("prebuild aggregates plugin errors without locations", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const answer = 42\n")
		builder := &WebModuleBuilder{
			runtimeScope: testRuntimeScope,
			module:       moduleRef,
			entryPoint:   entryPoint,
			prebuildPlugin: &buildTestPlugin{definePlugins: []api.Plugin{{
				Name: "prebuild-multi-error",
				Setup: func(build api.PluginBuild) {
					build.OnStart(func() (api.OnStartResult, error) {
						return api.OnStartResult{Errors: []api.Message{{Text: ""}, {Text: "second prebuild error"}}}, nil
					})
				},
			}}},
			buildPlugin: &buildTestPlugin{},
		}

		if _, err := builder.prebuild(); err == nil || !strings.Contains(err.Error(), "second prebuild error : Unknown error") {
			t.Fatalf("expected aggregated prebuild plugin errors, got %v", err)
		}
	})

	t.Run("prebuild wraps parser result fetch errors", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const answer = 42\n")
		builder := &WebModuleBuilder{
			runtimeScope:   testRuntimeScope,
			module:         moduleRef,
			entryPoint:     entryPoint,
			prebuildPlugin: &buildTestPlugin{getErr: errors.New("prebuild parser boom")},
			buildPlugin:    &buildTestPlugin{},
			publishDist:    false,
		}

		if _, err := builder.prebuild(); err == nil || !strings.Contains(err.Error(), "Error getting parser results") {
			t.Fatalf("expected parser result fetch error, got %v", err)
		}
	})

	t.Run("build wraps esbuild errors and parser result fetch errors", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const = 42\n")
		builder := &WebModuleBuilder{
			runtimeScope: testRuntimeScope,
			module:       moduleRef,
			entryPoint:   entryPoint,
			buildPlugin:  &buildTestPlugin{},
			publishDist:  false,
		}

		if _, err := builder.build(context.Background(), &module.BuildResult{Module: moduleRef}); err == nil || !strings.Contains(err.Error(), "Expected identifier") {
			t.Fatalf("expected build syntax error, got %v", err)
		}

		moduleRef, entryPoint = setupBuildPipelineTestFiles(t, testRuntimeScope, "export const answer = 42\n")
		builder = &WebModuleBuilder{
			runtimeScope: testRuntimeScope,
			module:       moduleRef,
			entryPoint:   entryPoint,
			buildPlugin:  &buildTestPlugin{getErr: errors.New("build parser boom")},
			publishDist:  false,
		}

		if _, err := builder.build(context.Background(), &module.BuildResult{Module: moduleRef}); err == nil || !strings.Contains(err.Error(), "Error getting parser results") {
			t.Fatalf("expected build parser result error, got %v", err)
		}
	})

	t.Run("build aggregates plugin errors without locations", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const answer = 42\n")
		builder := &WebModuleBuilder{
			runtimeScope: testRuntimeScope,
			module:       moduleRef,
			entryPoint:   entryPoint,
			buildPlugin: &buildTestPlugin{definePlugins: []api.Plugin{{
				Name: "build-multi-error",
				Setup: func(build api.PluginBuild) {
					build.OnStart(func() (api.OnStartResult, error) {
						return api.OnStartResult{Errors: []api.Message{{Text: ""}, {Text: "second build error"}}}, nil
					})
				},
			}}},
			publishDist: false,
		}

		if _, err := builder.build(context.Background(), &module.BuildResult{Module: moduleRef}); err == nil || !strings.Contains(err.Error(), "second build error : Unknown error") {
			t.Fatalf("expected aggregated build plugin errors, got %v", err)
		}
	})

	t.Run("build publish mode requires generated index html", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const answer = 42\n")
		publishDir := filepath.Join(t.TempDir(), "publish-web")

		builder := &WebModuleBuilder{
			runtimeScope:       testRuntimeScope,
			module:             moduleRef,
			entryPoint:         entryPoint,
			buildPlugin:        &buildTestPlugin{},
			publishDist:        true,
			distWebDirOverride: publishDir,
		}
		if _, err := builder.build(context.Background(), &module.BuildResult{Module: moduleRef}); err == nil || !strings.Contains(err.Error(), "index.html not generated") {
			t.Fatalf("expected missing index.html error, got %v", err)
		}

		builder.buildPlugin = &buildTestPlugin{writeIndexHTML: true}
		buildResult, err := builder.build(context.Background(), &module.BuildResult{Module: moduleRef})
		if err != nil {
			t.Fatalf("expected publish build success, got %v", err)
		}
		if buildResult == nil {
			t.Fatalf("expected build result in publish mode")
		}
		if _, err := os.Stat(filepath.Join(publishDir, "index.html")); err != nil {
			t.Fatalf("expected generated index.html, got %v", err)
		}
	})

	t.Run("build publish mode stages into dist web when override is empty", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const answer = 42\n")
		targetDir := filepath.Join(testRuntimeScope.cfg.DistPath, "web")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			t.Fatalf("mkdir target dir failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, "old.txt"), []byte("old"), 0o644); err != nil {
			t.Fatalf("seed old target file failed: %v", err)
		}

		builder := &WebModuleBuilder{
			runtimeScope: testRuntimeScope,
			module:       moduleRef,
			entryPoint:   entryPoint,
			buildPlugin:  &buildTestPlugin{writeIndexHTML: true},
			publishDist:  true,
		}

		tmpRoot := filepath.Join(t.TempDir(), "custom-tmp")
		ctx := staging.WithOpID(staging.WithTmpRoot(context.Background(), tmpRoot), "webmodulebuilder-build-stage")
		buildResult, err := builder.build(ctx, &module.BuildResult{Module: moduleRef})
		if err != nil {
			t.Fatalf("expected staging publish build success, got %v", err)
		}
		if buildResult == nil {
			t.Fatalf("expected build result for staging publish")
		}
		if _, err := os.Stat(filepath.Join(targetDir, "index.html")); err != nil {
			t.Fatalf("expected staged index.html in final dist/web, got %v", err)
		}
		if _, err := os.Stat(filepath.Join(targetDir, "old.txt")); err == nil {
			t.Fatalf("expected staging commit to replace previous dist contents")
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(testRuntimeScope.cfg.DistPath), ".choysum", "tmp", "staging", "webmodulebuilder-build-stage")); err == nil {
			t.Fatalf("expected staging op directory to be cleaned up after commit")
		}
	})

	t.Run("updatePrebuildResult wraps db errors", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const answer = 42\n")
		builder := &WebModuleBuilder{
			runtimeScope: testRuntimeScope,
			module:       moduleRef,
			entryPoint:   entryPoint,
			buildPlugin:  &buildTestPlugin{},
			parser:       defaultparser.NewVueParser(testRuntimeScope, moduleRef),
		}

		err := builder.updatePrebuildResult(withParserResults(&module.BuildResult{Module: moduleRef}, &parser.ParserResult{Path: entryPoint, Content: "export const answer = 42"}))
		if err == nil || !strings.Contains(err.Error(), "Error finding module web entry points") {
			t.Fatalf("expected db error, got %v", err)
		}
	})
}

func TestBuildCtx_WrapsStageErrors(t *testing.T) {
	t.Run("wraps prebuild errors", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const = 42\n")
		builder := &WebModuleBuilder{
			runtimeScope:   testRuntimeScope,
			module:         moduleRef,
			entryPoint:     entryPoint,
			parser:         defaultparser.NewVueParser(testRuntimeScope, moduleRef),
			prebuildPlugin: &buildTestPlugin{},
			buildPlugin:    &buildTestPlugin{},
		}

		if _, err := builder.BuildCtx(context.Background()); err == nil || !strings.Contains(err.Error(), "Error prebuilding") {
			t.Fatalf("expected wrapped prebuild error, got %v", err)
		}
	})

	t.Run("wraps validate errors", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const answer = 42\n")
		builder := &WebModuleBuilder{
			runtimeScope:   testRuntimeScope,
			module:         moduleRef,
			entryPoint:     entryPoint,
			parser:         defaultparser.NewVueParser(testRuntimeScope, moduleRef),
			prebuildPlugin: &buildTestPlugin{parserResults: []*parser.ParserResult{{Path: entryPoint, RawContent: "export const answer = 42", Content: "export const answer = 42"}}},
			buildPlugin: &buildTestPlugin{parserResults: []*parser.ParserResult{{
				Path:         entryPoint,
				VueComponent: &meta.IrComponent{Name: "CycleView", Path: "/a.vue", Extends: "/a.vue"},
			}}},
		}

		if _, err := builder.BuildCtx(context.Background()); err == nil || !strings.Contains(err.Error(), "Error validating") || !strings.Contains(err.Error(), "circular dependency detected") {
			t.Fatalf("expected wrapped validate error, got %v", err)
		}
	})

	t.Run("wraps persist errors", func(t *testing.T) {
		testRuntimeScope := newTestScopeWithDB(t).(*testScope)
		if err := testRuntimeScope.db.AutoMigrate(&meta.IrModule{}, &meta.IrApplication{}); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
		moduleRef, entryPoint := setupBuildPipelineTestFiles(t, testRuntimeScope, "export const answer = 42\n")
		moduleRef.Id = sql.NullString{String: "mod_build_ctx", Valid: true}
		builder := &WebModuleBuilder{
			runtimeScope:   testRuntimeScope,
			module:         moduleRef,
			entryPoint:     entryPoint,
			parser:         defaultparser.NewVueParser(testRuntimeScope, moduleRef),
			prebuildPlugin: &buildTestPlugin{parserResults: []*parser.ParserResult{{Path: entryPoint, RawContent: "export const answer = 42", Content: "export const answer = 42"}}},
			buildPlugin: &buildTestPlugin{parserResults: []*parser.ParserResult{{
				Path:       entryPoint,
				RawContent: "export const answer = 42",
				Content:    "export const answer = 42",
			}}},
		}

		if _, err := builder.BuildCtx(context.Background()); err == nil || !strings.Contains(err.Error(), "Error persisting build result") || !strings.Contains(err.Error(), "error persisting module components") {
			t.Fatalf("expected wrapped persist error, got %v", err)
		}
	})
}

func TestValidateWrapsBuildResultErrors(t *testing.T) {
	b := &WebModuleBuilder{}

	t.Run("wraps invalid inheritance chain", func(t *testing.T) {
		buildResult := withParserResults(&module.BuildResult{},
			&parser.ParserResult{VueComponent: &meta.IrComponent{Name: "DemoView", Path: "/child.vue", Extends: "/base.vue"}},
			&parser.ParserResult{VueComponent: &meta.IrComponent{Name: "DemoView", Path: "/sibling.vue", Extends: "/other.vue"}},
		)
		err := b.validate(buildResult)
		if err == nil || !strings.Contains(err.Error(), "invalid inheritance chain") {
			t.Fatalf("expected wrapped inheritance error, got %v", err)
		}
	})

	t.Run("wraps circular dependency", func(t *testing.T) {
		buildResult := withParserResults(&module.BuildResult{},
			&parser.ParserResult{VueComponent: &meta.IrComponent{Name: "CycleView", Path: "/a.vue", Extends: "/a.vue"}},
		)
		err := b.validate(buildResult)
		if err == nil || !strings.Contains(err.Error(), "circular dependency detected") {
			t.Fatalf("expected wrapped circular dependency error, got %v", err)
		}
	})

	t.Run("ignores nil parser results and unrelated names", func(t *testing.T) {
		buildResult := withParserResults(&module.BuildResult{},
			&parser.ParserResult{},
			&parser.ParserResult{VueComponent: &meta.IrComponent{Name: "FirstView", Path: "/first.vue"}},
			&parser.ParserResult{VueComponent: &meta.IrComponent{Name: "SecondView", Path: "/second.vue", Extends: "/missing.vue"}},
		)
		if err := b.validate(buildResult); err != nil {
			t.Fatalf("expected validate to ignore unrelated components, got %v", err)
		}
	})

	t.Run("deduplicates symlink alias component paths", func(t *testing.T) {
		realRoot := filepath.Join(t.TempDir(), "real")
		realComponentPath := filepath.Join(realRoot, "views", "CompanyListView.vue")
		if err := os.MkdirAll(filepath.Dir(realComponentPath), 0o755); err != nil {
			t.Fatalf("mkdir component directory: %v", err)
		}
		if err := os.WriteFile(realComponentPath, []byte("<template><div/></template>\n"), 0o644); err != nil {
			t.Fatalf("write component file: %v", err)
		}

		aliasRoot := filepath.Join(t.TempDir(), "alias")
		if err := os.Symlink(realRoot, aliasRoot); err != nil {
			t.Skipf("symlink not supported in this environment: %v", err)
		}
		aliasComponentPath := filepath.Join(aliasRoot, "views", "CompanyListView.vue")

		buildResult := withParserResults(&module.BuildResult{},
			&parser.ParserResult{VueComponent: &meta.IrComponent{Name: "CompanyListView", Path: realComponentPath}},
			&parser.ParserResult{VueComponent: &meta.IrComponent{Name: "CompanyListView", Path: aliasComponentPath}},
		)
		if err := b.validate(buildResult); err != nil {
			t.Fatalf("expected validate to deduplicate symlink alias paths, got %v", err)
		}
	})
}

func TestWebBuilderPathWithinRoot_ResolvesSymlinkAliases(t *testing.T) {
	realRoot := filepath.Join(t.TempDir(), "real")
	moduleRealRoot := filepath.Join(realRoot, "modules", "base")
	insideRealPath := filepath.Join(moduleRealRoot, "web", "views", "CompanyListView.vue")
	if err := os.MkdirAll(filepath.Dir(insideRealPath), 0o755); err != nil {
		t.Fatalf("mkdir inside real path: %v", err)
	}
	if err := os.WriteFile(insideRealPath, []byte("<template><div/></template>\n"), 0o644); err != nil {
		t.Fatalf("write inside real file: %v", err)
	}

	aliasRoot := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realRoot, aliasRoot); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	moduleAliasRoot := filepath.Join(aliasRoot, "modules", "base")
	insideAliasPath := filepath.Join(moduleAliasRoot, "web", "views", "CompanyListView.vue")
	if !webBuilderPathWithinRoot(insideRealPath, moduleAliasRoot) {
		t.Fatalf("expected real path %q to be within alias module root %q", insideRealPath, moduleAliasRoot)
	}
	if !webBuilderPathWithinRoot(insideAliasPath, moduleAliasRoot) {
		t.Fatalf("expected alias path %q to be within alias module root %q", insideAliasPath, moduleAliasRoot)
	}

	outsideRealPath := filepath.Join(realRoot, "modules", "auth", "web", "views", "LoginView.vue")
	if err := os.MkdirAll(filepath.Dir(outsideRealPath), 0o755); err != nil {
		t.Fatalf("mkdir outside real path: %v", err)
	}
	if err := os.WriteFile(outsideRealPath, []byte("<template><div/></template>\n"), 0o644); err != nil {
		t.Fatalf("write outside real file: %v", err)
	}
	if webBuilderPathWithinRoot(outsideRealPath, moduleAliasRoot) {
		t.Fatalf("expected outside path %q not to be within alias module root %q", outsideRealPath, moduleAliasRoot)
	}
}

func TestReparseXPathComponentsPropertyNode(t *testing.T) {
	t.Run("finds component property after reparsing", func(t *testing.T) {
		testRuntimeScope := newTestScope().(*testScope)
		root := t.TempDir()
		testRuntimeScope.cfg.ModulesPath = root
		if err := os.WriteFile(filepath.Join(root, "tsconfig.json"), []byte(`{"compilerOptions":{}}`), 0o644); err != nil {
			t.Fatalf("write tsconfig failed: %v", err)
		}
		builder := &WebModuleBuilder{runtimeScope: testRuntimeScope, module: &meta.IrModule{Name: "test", Path: root}}
		vueParser := defaultparser.NewVueParser(testRuntimeScope, builder.module)
		parsed, err := vueParser.Parse(map[string]string{}, filepath.Join(root, "ChildView.vue"), `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import Xpath from '/virtual/modules/core/web/component/xpath.vue';
import KeepMe from './KeepMe.vue';

export default defineComponent({
  name: 'ChildView',
  components: { Xpath: Xpath, KeepMe },
});
</script>`)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if parsed == nil || parsed.RawScriptNode == nil {
			t.Fatalf("expected RawScriptNode to be parsed")
		}

		node, err := builder.reparseXPathComponentsPropertyNode(htmlquery.InnerText(parsed.RawScriptNode), "Xpath")
		if err != nil {
			t.Fatalf("reparseXPathComponentsPropertyNode failed: %v", err)
		}
		if node == nil || !(strings.Contains(strings.TrimSpace(node.Text), "Xpath") || strings.Contains(strings.TrimSpace(node.ValueText), "Xpath")) {
			t.Fatalf("expected reparsed xpath property node, got %#v", node)
		}
	})

	t.Run("returns clear errors for invalid inputs", func(t *testing.T) {
		builder := &WebModuleBuilder{}
		if _, err := builder.reparseXPathComponentsPropertyNode("", "Xpath"); err == nil || !strings.Contains(err.Error(), "script content is empty") {
			t.Fatalf("expected empty content error, got %v", err)
		}
		if _, err := builder.reparseXPathComponentsPropertyNode("export default {}", "Xpath"); err == nil || !strings.Contains(err.Error(), "web builder context is not available") {
			t.Fatalf("expected missing builder context error, got %v", err)
		}
	})

	t.Run("returns symbol not found when reparsed script has no xpath component", func(t *testing.T) {
		testRuntimeScope := newTestScope().(*testScope)
		root := t.TempDir()
		testRuntimeScope.cfg.ModulesPath = root
		if err := os.WriteFile(filepath.Join(root, "tsconfig.json"), []byte(`{"compilerOptions":{}}`), 0o644); err != nil {
			t.Fatalf("write tsconfig failed: %v", err)
		}
		builder := &WebModuleBuilder{runtimeScope: testRuntimeScope, module: &meta.IrModule{Name: "test", Path: root}}
		vueParser := defaultparser.NewVueParser(testRuntimeScope, builder.module)
		parsed, err := vueParser.Parse(map[string]string{}, filepath.Join(root, "ChildView.vue"), `<template><div/></template>
<script lang="ts" _name="ChildView">
import { defineComponent } from 'vue';
import KeepMe from './KeepMe.vue';

export default defineComponent({
  name: 'ChildView',
  components: { KeepMe },
});
</script>`)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if parsed == nil || parsed.RawScriptNode == nil {
			t.Fatalf("expected RawScriptNode to be parsed")
		}

		if _, err := builder.reparseXPathComponentsPropertyNode(htmlquery.InnerText(parsed.RawScriptNode), "Xpath"); err == nil || !strings.Contains(err.Error(), "not found after reparse") {
			t.Fatalf("expected symbol not found error, got %v", err)
		}
	})
}

func TestInheritanceValidationHelpers(t *testing.T) {
	b := &WebModuleBuilder{}
	base := &meta.IrComponent{Name: "DemoView", Path: "/base.vue"}
	child := &meta.IrComponent{Name: "DemoView", Path: "/child.vue", Extends: "/base.vue"}
	grandchild := &meta.IrComponent{Name: "DemoView", Path: "/grandchild.vue", Extends: "/child.vue"}
	pathMap := map[string]*meta.IrComponent{
		base.Path:       base,
		child.Path:      child,
		grandchild.Path: grandchild,
	}

	if err := b.checkInheritanceChain([]*meta.IrComponent{base, child, grandchild}, pathMap); err != nil {
		t.Fatalf("checkInheritanceChain(valid) error = %v", err)
	}

	sibling := &meta.IrComponent{Name: "DemoView", Path: "/sibling.vue", Extends: "/other.vue"}
	if err := b.checkInheritanceChain([]*meta.IrComponent{child, sibling}, map[string]*meta.IrComponent{
		child.Path:   child,
		sibling.Path: sibling,
	}); err == nil || !strings.Contains(err.Error(), "same name but not in inheritance chain") {
		t.Fatalf("expected invalid inheritance chain error, got %v", err)
	}

	if err := b.checkCircularDependency(nil, pathMap, map[string]bool{}); err != nil {
		t.Fatalf("checkCircularDependency(nil) error = %v", err)
	}
	if err := b.checkCircularDependency(base, pathMap, map[string]bool{}); err != nil {
		t.Fatalf("checkCircularDependency(no extends) error = %v", err)
	}

	cycleA := &meta.IrComponent{Name: "CycleView", Path: "/cycle-a.vue", Extends: "/cycle-b.vue"}
	cycleB := &meta.IrComponent{Name: "CycleView", Path: "/cycle-b.vue", Extends: "/cycle-a.vue"}
	cycleMap := map[string]*meta.IrComponent{
		cycleA.Path: cycleA,
		cycleB.Path: cycleB,
	}
	if err := b.checkCircularDependency(cycleA, cycleMap, map[string]bool{}); err == nil || !strings.Contains(err.Error(), "circular dependency detected") {
		t.Fatalf("expected circular dependency error, got %v", err)
	}
}

func TestEntryPointImportsCollectsModulesAndAppStores(t *testing.T) {
	testRuntimeScope := newTestScopeWithDB(t).(*testScope)
	modulesDir := t.TempDir()
	testRuntimeScope.cfg.ModulesPath = modulesDir
	db := testRuntimeScope.db

	if err := db.AutoMigrate(&meta.IrApplication{}, &meta.IrModule{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	absEntry := filepath.Join(modulesDir, "partner", "web", "main.ts")
	if err := os.MkdirAll(filepath.Dir(absEntry), 0o755); err != nil {
		t.Fatalf("mkdir abs entry dir: %v", err)
	}
	if err := os.WriteFile(absEntry, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("write abs entry: %v", err)
	}
	_, crmWebDir, _, err := modulegenerator.WorkspaceGeneratedAPITargets(modulesDir, "crm", runtimeOptionsFromScope(testRuntimeScope).defaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets(crm) error = %v", err)
	}
	appStore := filepath.Join(crmWebDir, "stores", "index.ts")
	crmStoreImportPath := normalizeAbsImportPath(appStore)
	if err := os.MkdirAll(filepath.Dir(appStore), 0o755); err != nil {
		t.Fatalf("mkdir app store dir: %v", err)
	}
	if err := os.WriteFile(appStore, []byte("export {}"), 0o644); err != nil {
		t.Fatalf("write app store: %v", err)
	}
	_, hrWebDir, _, err := modulegenerator.WorkspaceGeneratedAPITargets(modulesDir, "hr", runtimeOptionsFromScope(testRuntimeScope).defaultChoysumPath)
	if err != nil {
		t.Fatalf("WorkspaceGeneratedAPITargets(hr) error = %v", err)
	}
	hrStoreImportPath := normalizeAbsImportPath(filepath.Join(hrWebDir, "stores", "index.ts"))

	modules := []*meta.IrModule{
		{Name: "auth", Status: meta.Installed, WebEntryPoint: "./web/index.ts"},
		{Name: "base", Status: meta.Installed, WebEntryPoint: "./web/entry.ts"},
		{Name: "partner", Status: meta.Installed, WebEntryPoint: absEntry},
		{Name: "draft", Status: meta.Uninstalled, WebEntryPoint: "./web/draft.ts"},
	}
	for _, m := range modules {
		if err := db.Create(m).Error; err != nil {
			t.Fatalf("create module %s: %v", m.Name, err)
		}
	}
	for _, app := range []*meta.IrApplication{{Name: "crm"}, {Name: "hr"}} {
		if err := db.Create(app).Error; err != nil {
			t.Fatalf("create app %s: %v", app.Name, err)
		}
	}

	b := &WebModuleBuilder{
		runtimeScope: testRuntimeScope,
		module:       &meta.IrModule{Name: "auth", Path: filepath.Join(modulesDir, "auth")},
	}

	imports := b.entryPointImports()
	importSet := make(map[string]bool, len(imports))
	for _, item := range imports {
		importSet[item] = true
	}

	if importSet["@/auth/web/index"] {
		t.Fatalf("expected builder to skip importing its own module entrypoint, got %#v", imports)
	}
	if !importSet["@/base/web/entry"] {
		t.Fatalf("expected relative installed module entrypoint to be imported, got %#v", imports)
	}
	if !importSet["@/partner/web/main"] {
		t.Fatalf("expected absolute installed module entrypoint to be normalized and imported, got %#v", imports)
	}
	if !importSet[crmStoreImportPath] {
		t.Fatalf("expected existing application store import, got %#v", imports)
	}
	if importSet["@/draft/web/draft"] {
		t.Fatalf("expected uninstalled module entrypoint to be skipped, got %#v", imports)
	}
	if importSet[hrStoreImportPath] {
		t.Fatalf("expected missing application store file to be skipped, got %#v", imports)
	}
}
