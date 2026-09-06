// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/vuevirtual"
)

// QuickJSCoder runs embedded language-core createServiceScript inside QuickJS.
// It does not invoke Node.
type QuickJSCoder struct {
	mu     sync.Mutex
	engine *quickjsengine.QuickjsEngine
}

type jsServiceScript struct {
	EmbeddedID string        `json:"embeddedId"`
	ScriptKind string        `json:"scriptKind"`
	Content    string        `json:"content"`
	Mappings   []SpanMapping `json:"mappings"`
}

// Test hooks (overridden in *_test.go).
var (
	vueVirtualScriptContent = func() string { return vuevirtual.VueVirtualScript }
	newVueVirtualEngine     = defaultNewVueVirtualEngine
	marshalJSValue          = defaultMarshalJSValue
)

func defaultNewVueVirtualEngine(script string) (jsengine.JsEngine, error) {
	return quickjsengine.NewFactory(
		quickjsengine.WithScript(&jsengine.JsScript{
			FileName: "scripts/vuevirtual/dist/index.js",
			Content:  script,
		}),
	)()
}

func defaultMarshalJSValue(eng *quickjsengine.QuickjsEngine, v any) (*quickjs.Value, error) {
	return eng.Ctx.Marshal(v)
}

// NewQuickJSCoder returns a Coder backed by the embedded vuevirtual IIFE.
func NewQuickJSCoder() *QuickJSCoder {
	return &QuickJSCoder{}
}

func (c *QuickJSCoder) ensureEngineLocked() error {
	if c.engine != nil {
		return nil
	}
	eng, err := newVueVirtualEngine(vueVirtualScriptContent())
	if err != nil {
		return fmt.Errorf("vue: create QuickJS engine: %w", err)
	}
	qje, ok := eng.(*quickjsengine.QuickjsEngine)
	if !ok {
		_ = eng.Close()
		return fmt.Errorf("vue: unexpected engine type %T", eng)
	}
	c.engine = qje
	return nil
}

// Close releases the underlying QuickJS engine.
func (c *QuickJSCoder) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.engine == nil {
		return nil
	}
	err := c.engine.Close()
	c.engine = nil
	return err
}

// CreateServiceScript evaluates vuevirtual.createServiceScript in QuickJS.
func (c *QuickJSCoder) CreateServiceScript(path, source string, opts CodegenOptions) (ServiceScript, error) {
	if c == nil {
		return ServiceScript{}, fmt.Errorf("vue: QuickJSCoder is nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureEngineLocked(); err != nil {
		return ServiceScript{}, err
	}
	eng := c.engine

	fn := eng.Ctx.Eval("vuevirtual.createServiceScript")
	defer fn.Free()
	if fn.IsException() {
		return ServiceScript{}, fmt.Errorf("vue: eval createServiceScript: %w", eng.Ctx.Exception())
	}
	if !fn.IsFunction() {
		return ServiceScript{}, fmt.Errorf("vue: vuevirtual.createServiceScript is not a function")
	}

	pathVal, err := marshalJSValue(eng, path)
	if err != nil {
		return ServiceScript{}, fmt.Errorf("vue: marshal path: %w", err)
	}
	defer pathVal.Free()
	sourceVal, err := marshalJSValue(eng, source)
	if err != nil {
		return ServiceScript{}, fmt.Errorf("vue: marshal source: %w", err)
	}
	defer sourceVal.Free()
	optsMap := map[string]any{}
	if opts.CurrentDirectory != "" {
		optsMap["currentDirectory"] = opts.CurrentDirectory
	}
	optsVal, err := marshalJSValue(eng, optsMap)
	if err != nil {
		return ServiceScript{}, fmt.Errorf("vue: marshal opts: %w", err)
	}
	defer optsVal.Free()

	ret := fn.Execute(eng.Ctx.Null(), pathVal, sourceVal, optsVal)
	defer ret.Free()
	if ret.IsException() {
		return ServiceScript{}, fmt.Errorf("vue: createServiceScript: %w", eng.Ctx.Exception())
	}

	// Round-trip via JSON so mapping.verification (bool or object) decodes into any.
	stringify := eng.Ctx.Eval("JSON.stringify")
	defer stringify.Free()
	if stringify.IsException() || !stringify.IsFunction() {
		return ServiceScript{}, fmt.Errorf("vue: JSON.stringify unavailable")
	}
	jsonVal := stringify.Execute(eng.Ctx.Null(), ret)
	defer jsonVal.Free()
	if jsonVal.IsException() {
		return ServiceScript{}, fmt.Errorf("vue: stringify result: %w", eng.Ctx.Exception())
	}
	var parsed jsServiceScript
	if err := json.Unmarshal([]byte(jsonVal.String()), &parsed); err != nil {
		return ServiceScript{}, fmt.Errorf("vue: decode service script: %w", err)
	}
	if parsed.Content == "" {
		return ServiceScript{}, fmt.Errorf("vue: empty service script content for %s", path)
	}
	return ServiceScript{
		EmbeddedID:    parsed.EmbeddedID,
		ScriptKind:    parsed.ScriptKind,
		Content:       parsed.Content,
		SourceContent: source,
		Mappings:      parsed.Mappings,
	}, nil
}
