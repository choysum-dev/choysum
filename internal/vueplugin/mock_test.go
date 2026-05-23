// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vueplugin

import (
	"context"
	"fmt"
	"log/slog"

	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type vuePluginTestScope struct {
	ctx context.Context
	cfg *config.Config
}

func (e *vuePluginTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *vuePluginTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *vuePluginTestScope) Session() *scope.Session { return nil }
func (e *vuePluginTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *vuePluginTestScope) Context() context.Context { return e.ctx }
func (e *vuePluginTestScope) Logger() *slog.Logger     { return slog.Default() }
func (e *vuePluginTestScope) Config() *config.Config   { return e.cfg }
func (e *vuePluginTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newVuePluginTestScope() *vuePluginTestScope {
	return &vuePluginTestScope{
		ctx: context.Background(),
		cfg: &config.Config{Server: config.NewDefaultServerConfig()},
	}
}

func newTestExecutor(config *MockEngineConfig) (jsexecutor.JsExecutor, error) {
	return jsexecutor.NewCompilerExecutor(newVuePluginTestScope(), jsexecutor.WithJsEngine(NewMockEngineFactory(config)))
}

// MockEngineConfig defines the configuration for a mock engine
type MockEngineConfig struct {
	// Response to return, if nil then generate default response
	Response *jsengine.JsResponse
	// Error to return from Execute method
	ExecuteError error
	// Whether to return empty result
	EmptyResult bool
	// Whether to return invalid result type
	InvalidResult bool
	// Script configuration (for Vue SFC)
	Script *MockScriptConfig
	// Template configuration (for Vue SFC)
	Template *MockTemplateConfig
	// Styles configuration (for Vue SFC)
	Styles []*MockStyleConfig
	// Sass configuration (for Sass compilation)
	Sass *MockSassConfig
	// Service-specific responses for different services
	ServiceResponses map[string]interface{}
}

// MockScriptConfig defines script-specific configuration
type MockScriptConfig struct {
	// Script content, if nil then script will be nil
	Content interface{}
	// Script language
	Lang string
	// Script warnings, can be []interface{} or invalid type
	Warnings interface{}
	// Script map
	Map interface{}
	// Script setup flag
	Setup interface{}
}

// MockTemplateConfig defines template-specific configuration
type MockTemplateConfig struct {
	// Template code
	Code string
	// Template tips
	Tips []interface{}
	// Template errors
	Errors []interface{}
	// Template scoped flag
	Scoped bool
}

// MockStyleConfig defines style-specific configuration
type MockStyleConfig struct {
	// Style code
	Code string
	// Scoped flag, can be bool or invalid type to trigger generateEntryContents error
	Scoped interface{}
	// Style errors
	Errors []interface{}
}

// MockSassConfig defines Sass compilation configuration
type MockSassConfig struct {
	// Compiled CSS output
	CSS string
	// Source map output
	Map string
	// Stats information
	Stats *MockSassStatsConfig
	// Whether to return compilation error
	CompileError bool
}

// MockSassStatsConfig defines Sass stats configuration
type MockSassStatsConfig struct {
	Entry         string
	Start         int
	End           int
	Duration      int
	IncludedFiles []string
}

// MockEngine is a configurable mock engine
type MockEngine struct {
	config *MockEngineConfig
}

func (e *MockEngine) Load(scripts []*jsengine.JsScript) error { return nil }
func (e *MockEngine) Close() error                            { return nil }
func (e *MockEngine) Execute(ctx context.Context, req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	_ = ctx
	// Return execute error if configured
	if e.config.ExecuteError != nil {
		return nil, e.config.ExecuteError
	}

	// Return custom response if configured
	if e.config.Response != nil {
		e.config.Response.Id = req.Id
		return e.config.Response, nil
	}

	// Check for service-specific responses
	if e.config.ServiceResponses != nil {
		if serviceResponse, exists := e.config.ServiceResponses[req.Service]; exists {
			return &jsengine.JsResponse{
				Id:     req.Id,
				Result: serviceResponse,
			}, nil
		}
	}

	// Return empty result if configured
	if e.config.EmptyResult {
		return &jsengine.JsResponse{
			Id:     req.Id,
			Result: map[string]interface{}{},
		}, nil
	}

	// Return invalid result type if configured
	if e.config.InvalidResult {
		return &jsengine.JsResponse{
			Id:     req.Id,
			Result: "This is not a map[string]interface{}",
		}, nil
	}

	// Handle different services
	switch req.Service {
	case "sfc.vue.compileSFC":
		return e.handleVueCompileSFC(req)
	case "sfc.sass.renderSync":
		return e.handleSassRenderSync(req)
	default:
		return e.handleDefault(req)
	}
}

// handleVueCompileSFC handles the Vue SFC compilation service
func (e *MockEngine) handleVueCompileSFC(req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	result := map[string]interface{}{}

	// Add script
	if e.config.Script != nil {
		if e.config.Script.Content == nil {
			result["script"] = nil
		} else {
			scriptMap := map[string]interface{}{
				"content": e.config.Script.Content,
				"lang":    e.config.Script.Lang,
			}
			if e.config.Script.Warnings != nil {
				scriptMap["warnings"] = e.config.Script.Warnings
			}
			if e.config.Script.Map != nil {
				scriptMap["map"] = e.config.Script.Map
			}
			if e.config.Script.Setup != nil {
				scriptMap["setup"] = e.config.Script.Setup
			}
			result["script"] = scriptMap
		}
	} else {
		// Default script
		result["script"] = map[string]interface{}{
			"content": "export default { name: 'MockComponent' }",
			"lang":    "js",
		}
	}

	// Add template
	if e.config.Template != nil {
		templateMap := map[string]interface{}{
			"code":   e.config.Template.Code,
			"scoped": e.config.Template.Scoped,
		}
		if e.config.Template.Tips != nil {
			templateMap["tips"] = e.config.Template.Tips
		}
		if e.config.Template.Errors != nil {
			templateMap["errors"] = e.config.Template.Errors
		}
		result["template"] = templateMap
	} else {
		// Default template
		result["template"] = map[string]interface{}{
			"code":   "function render() { return h('div', 'Mock Component'); }",
			"scoped": false,
		}
	}

	// Add styles
	if e.config.Styles != nil {
		styles := make([]interface{}, len(e.config.Styles))
		for i, style := range e.config.Styles {
			styleMap := map[string]interface{}{
				"code":   style.Code,
				"scoped": style.Scoped, // This can be invalid types to trigger generateEntryContents error
			}
			if style.Errors != nil {
				styleMap["errors"] = style.Errors
			}
			styles[i] = styleMap
		}
		result["styles"] = styles
	} else {
		// Default empty styles
		result["styles"] = []interface{}{}
	}

	return &jsengine.JsResponse{
		Id:     req.Id,
		Result: result,
	}, nil
}

// handleSassRenderSync handles the Sass compilation service
func (e *MockEngine) handleSassRenderSync(req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	// Check if configured to return compilation error
	if e.config.Sass != nil && e.config.Sass.CompileError {
		return nil, fmt.Errorf("Sass compilation failed: syntax error")
	}

	// Build result based on configuration
	result := map[string]interface{}{}

	if e.config.Sass != nil {
		// Use configured Sass settings
		result["css"] = e.config.Sass.CSS
		result["map"] = e.config.Sass.Map

		// Add stats
		if e.config.Sass.Stats != nil {
			result["stats"] = map[string]interface{}{
				"entry":         e.config.Sass.Stats.Entry,
				"start":         e.config.Sass.Stats.Start,
				"end":           e.config.Sass.Stats.End,
				"duration":      e.config.Sass.Stats.Duration,
				"includedFiles": e.config.Sass.Stats.IncludedFiles,
			}
		} else {
			// Default stats
			result["stats"] = map[string]interface{}{
				"entry":         "",
				"start":         0,
				"end":           0,
				"duration":      0,
				"includedFiles": []string{},
			}
		}
	} else {
		// Default Sass compilation result
		result["css"] = ".mock-sass { color: red; }"
		result["map"] = ""
		result["stats"] = map[string]interface{}{
			"entry":         "",
			"start":         0,
			"end":           0,
			"duration":      0,
			"includedFiles": []string{},
		}
	}

	return &jsengine.JsResponse{
		Id:     req.Id,
		Result: result,
	}, nil
}

// handleDefault handles default service requests
func (e *MockEngine) handleDefault(req *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	result := map[string]interface{}{}

	// Add script
	if e.config.Script != nil {
		if e.config.Script.Content == nil {
			result["script"] = nil
		} else {
			scriptMap := map[string]interface{}{
				"content": e.config.Script.Content,
				"lang":    e.config.Script.Lang,
			}
			if e.config.Script.Warnings != nil {
				scriptMap["warnings"] = e.config.Script.Warnings
			}
			result["script"] = scriptMap
		}
	} else {
		// Default script
		result["script"] = map[string]interface{}{
			"content": "export default { name: 'MockComponent' }",
			"lang":    "js",
		}
	}

	// Add template
	if e.config.Template != nil {
		result["template"] = map[string]interface{}{
			"code": e.config.Template.Code,
		}
	} else {
		// Default template
		result["template"] = map[string]interface{}{
			"code": "function render() { return h('div', 'Mock Component'); }",
		}
	}

	// Add styles
	if e.config.Styles != nil {
		styles := make([]interface{}, len(e.config.Styles))
		for i, style := range e.config.Styles {
			styleMap := map[string]interface{}{
				"code":   style.Code,
				"scoped": style.Scoped,
			}
			styles[i] = styleMap
		}
		result["styles"] = styles
	} else {
		// Default empty styles
		result["styles"] = []interface{}{}
	}

	return &jsengine.JsResponse{
		Id:     req.Id,
		Result: result,
	}, nil
}

// NewMockEngineFactory creates a factory that returns MockEngine with given config
func NewMockEngineFactory(config *MockEngineConfig) jsengine.JsEngineFactory {
	return func() (jsengine.JsEngine, error) {
		return &MockEngine{config: config}, nil
	}
}

// NewMockEngineFactoryWithError creates a factory that returns error during engine creation
func NewMockEngineFactoryWithError(errorMessage string) jsengine.JsEngineFactory {
	return func() (jsengine.JsEngine, error) {
		return nil, fmt.Errorf("%s", errorMessage)
	}
}

// Helper functions for creating common mock responses

// CreateMockVueCompileSFCResponse creates a mock response for Vue SFC compilation
func CreateMockVueCompileSFCResponse(scriptContent string, templateCode string, styles []map[string]interface{}) *jsengine.JsResponse {
	return &jsengine.JsResponse{
		Result: map[string]interface{}{
			"script": map[string]interface{}{
				"content": scriptContent,
				"lang":    "js",
			},
			"template": map[string]interface{}{
				"code":   templateCode,
				"scoped": false,
			},
			"styles": styles,
		},
	}
}

// CreateMockVueCompileSFCResponseWithWarnings creates a mock response with script warnings
func CreateMockVueCompileSFCResponseWithWarnings(scriptContent string, templateCode string, warnings []interface{}) *jsengine.JsResponse {
	return &jsengine.JsResponse{
		Result: map[string]interface{}{
			"script": map[string]interface{}{
				"content":  scriptContent,
				"lang":     "js",
				"warnings": warnings,
			},
			"template": map[string]interface{}{
				"code":   templateCode,
				"scoped": false,
			},
			"styles": []interface{}{},
		},
	}
}

// CreateMockSassRenderSyncResponse creates a mock response for Sass compilation
func CreateMockSassRenderSyncResponse(css string, sourceMap string, stats map[string]interface{}) *jsengine.JsResponse {
	if stats == nil {
		stats = map[string]interface{}{
			"entry":         "",
			"start":         0,
			"end":           0,
			"duration":      0,
			"includedFiles": []string{},
		}
	}

	return &jsengine.JsResponse{
		Result: map[string]interface{}{
			"css":   css,
			"map":   sourceMap,
			"stats": stats,
		},
	}
}

// CreateMockResponse creates a standard mock response with given parameters (backward compatibility)
func CreateMockResponse(scriptContent string, templateCode string, styles []map[string]interface{}) *jsengine.JsResponse {
	return CreateMockVueCompileSFCResponse(scriptContent, templateCode, styles)
}

// CreateMockResponseWithWarnings creates a mock response with script warnings (backward compatibility)
func CreateMockResponseWithWarnings(scriptContent string, templateCode string, warnings []interface{}) *jsengine.JsResponse {
	return CreateMockVueCompileSFCResponseWithWarnings(scriptContent, templateCode, warnings)
}

// CreateMockResponseWithError creates a mock response for error scenarios
// Note: This function is for testing purposes only. In practice, errors should be
// returned through the error return value of the Execute method, not in the response.
func CreateMockResponseWithError(errorMessage string) *jsengine.JsResponse {
	return &jsengine.JsResponse{
		Result: map[string]interface{}{
			"error": errorMessage,
		},
	}
}

// Helper functions for creating mock configs that trigger generateEntryContents errors

// CreateMockEngineConfigWithInvalidStyleScoped creates a mock config with invalid scoped values
// This will trigger type assertion errors in generateEntryContents
func CreateMockEngineConfigWithInvalidStyleScoped(invalidScopedValue interface{}) *MockEngineConfig {
	return &MockEngineConfig{
		Script: &MockScriptConfig{
			Content: "export default { name: 'TestComponent' }",
			Lang:    "js",
		},
		Template: &MockTemplateConfig{
			Code: "function render() { return h('div', 'test'); }",
		},
		Styles: []*MockStyleConfig{
			{
				Code:   ".test { color: red; }",
				Scoped: invalidScopedValue, // This will cause type assertion error
			},
		},
	}
}

// CreateMockEngineConfigWithMultipleInvalidStyles creates a mock config with multiple invalid scoped values
func CreateMockEngineConfigWithMultipleInvalidStyles() *MockEngineConfig {
	return &MockEngineConfig{
		Script: &MockScriptConfig{
			Content: "export default { name: 'TestComponent' }",
			Lang:    "js",
		},
		Template: &MockTemplateConfig{
			Code: "function render() { return h('div', 'test'); }",
		},
		Styles: []*MockStyleConfig{
			{
				Code:   ".test1 { color: red; }",
				Scoped: "not-a-boolean", // Invalid type
			},
			{
				Code:   ".test2 { color: blue; }",
				Scoped: 42, // Invalid type
			},
			{
				Code:   ".test3 { color: green; }",
				Scoped: []string{"invalid"}, // Invalid type
			},
		},
	}
}

// CreateMockEngineConfigWithMixedValidInvalidStyles creates a mock config with mixed valid/invalid scoped values
func CreateMockEngineConfigWithMixedValidInvalidStyles() *MockEngineConfig {
	return &MockEngineConfig{
		Script: &MockScriptConfig{
			Content: "export default { name: 'TestComponent' }",
			Lang:    "js",
		},
		Template: &MockTemplateConfig{
			Code: "function render() { return h('div', 'test'); }",
		},
		Styles: []*MockStyleConfig{
			{
				Code:   ".test1 { color: red; }",
				Scoped: true, // Valid - causes early return in someScoped function
			},
			{
				Code:   ".test2 { color: blue; }",
				Scoped: "invalid-boolean", // Invalid - but won't be reached due to early return
			},
		},
	}
}

// CreateMockEngineConfigWithNilStyleScoped creates a mock config with nil scoped values
func CreateMockEngineConfigWithNilStyleScoped() *MockEngineConfig {
	return &MockEngineConfig{
		Script: &MockScriptConfig{
			Content: "export default { name: 'TestComponent' }",
			Lang:    "js",
		},
		Template: &MockTemplateConfig{
			Code: "function render() { return h('div', 'test'); }",
		},
		Styles: []*MockStyleConfig{
			{
				Code:   ".test { color: red; }",
				Scoped: nil, // This will cause type assertion error
			},
		},
	}
}
