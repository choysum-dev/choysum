// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	internalbackendbuilder "github.com/choysum-dev/choysum/internal/module/artifact/build/backend"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

type Phase string

const (
	PhasePreInit       Phase = "pre_init"
	PhasePostInit      Phase = "post_init"
	PhasePreUpgrade    Phase = "pre_upgrade"
	PhasePostUpgrade   Phase = "post_upgrade"
	PhasePreUninstall  Phase = "pre_uninstall"
	PhasePostUninstall Phase = "post_uninstall"
)

type RunOptions struct {
	Scripts     []*jsengine.JsScript
	FromVersion string
	// ReuseExecutorScripts keeps loaded scripts on the shared executor between
	// phase runs so callers can amortize reload costs across module loops.
	ReuseExecutorScripts bool
}

type Runner struct {
	runtimeScope scope.Scope
	jsExecutor   jsexecutor.ScriptExecutor
	module       *meta.IrModule
}

func NewRunner(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.IrModule) (*Runner, error) {
	if runtimeScope == nil || module == nil {
		return nil, nil
	}
	return &Runner{runtimeScope: runtimeScope, jsExecutor: jsExecutor, module: module}, nil
}

func (r *Runner) RunPhase(ctx context.Context, phase Phase, opts RunOptions) error {
	if r == nil {
		return nil
	}
	if r.jsExecutor == nil {
		return fmt.Errorf("js executor is nil")
	}
	if phase == "" {
		return nil
	}

	resolved := normalizeConfig(phase)

	scripts, err := r.resolveScripts(ctx, phase, opts, resolved.Required)
	if err != nil {
		if !resolved.Required {
			if r.runtimeScope != nil {
				r.runtimeScope.Logger().Warn("hook script unavailable", "module", r.module.Name, "phase", string(phase), "reason", "ignored", "error", err)
			}
			return nil
		}
		return err
	}
	if wrapper := r.buildHookWrapperScript(); wrapper != nil {
		scripts = append(scripts, wrapper)
	}
	appName, moduleName := r.moduleSelector()
	phaseName := strings.TrimSpace(string(phase))

	attempts := 1
	if resolved.Retry > 0 {
		attempts = resolved.Retry + 1
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		execCtx := r.buildExecContext(ctx)
		jsCtx := r.buildJsContext(ctx, opts)
		req := &jsengine.JsRequest{
			Id:      jsCtx.requestId,
			Service: "__choysum_hook__",
			Args:    []interface{}{appName, moduleName, phaseName},
			Context: jsCtx.payload,
		}

		ctxWithTimeout := execCtx
		var cancel context.CancelFunc
		if resolved.Timeout > 0 {
			ctxWithTimeout, cancel = context.WithTimeout(execCtx, resolved.Timeout)
		}

		execErr := r.executeWithScripts(ctxWithTimeout, scripts, req, &lastErr, opts.ReuseExecutorScripts)
		if cancel != nil {
			cancel()
		}
		if execErr == nil {
			return nil
		}

		if !errors.Is(execErr, errHookRetriable) {
			break
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("hook failed")
	}
	if !resolved.Required {
		if r.runtimeScope != nil {
			r.runtimeScope.Logger().Warn("hook execution failed", "module", r.module.Name, "phase", string(phase), "reason", "ignored", "error", lastErr)
		}
		return nil
	}
	return lastErr
}

func deriveRuntimeScope(ctx context.Context, baseScope scope.Scope) scope.Scope {
	if baseScope == nil || ctx == nil {
		return baseScope
	}
	if runtimeScope := baseScope.WithContext(ctx); runtimeScope != nil {
		return runtimeScope
	}
	return baseScope
}

func (r *Runner) resolveScripts(ctx context.Context, phase Phase, opts RunOptions, required bool) ([]*jsengine.JsScript, error) {
	if len(opts.Scripts) > 0 {
		return opts.Scripts, nil
	}
	script, err := r.buildModuleEntryScript(ctx)
	if err != nil {
		if required {
			moduleName := ""
			entry := ""
			if r != nil && r.module != nil {
				moduleName = strings.TrimSpace(r.module.Name)
				if moduleName == "" {
					moduleName = strings.TrimSpace(r.module.ApplicationStr)
				}
				entry = strings.TrimSpace(r.module.ServiceEntryPoint)
			}
			return nil, fmt.Errorf("required hook phase %s: failed to build module entry script (module=%q, entry=%q): %w", phase, moduleName, entry, err)
		}
	} else if script != nil {
		return []*jsengine.JsScript{script}, nil
	}
	return LoadDistScripts(deriveRuntimeScope(ctx, r.runtimeScope), r.module)
}

func (r *Runner) buildModuleEntryScript(ctx context.Context) (*jsengine.JsScript, error) {
	entry := strings.TrimSpace(r.module.ServiceEntryPoint)
	if entry == "" {
		return nil, nil
	}
	runtimeScope := deriveRuntimeScope(ctx, r.runtimeScope)
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	if !filepath.IsAbs(entry) {
		entry = filepath.Join(runtimeOpts.modulesPath, r.module.Name, entry)
	}
	builder := internalbackendbuilder.NewModuleBuilder(runtimeScope, r.jsExecutor, r.module, entry, internalbackendbuilder.WithPublishDist(false))
	if bundler, ok := builder.(module.Bundler); ok {
		result, err := bundler.Bundle()
		if err != nil {
			return nil, err
		}
		return ScriptFromBuildResult(result)
	}
	result, err := builder.Build()
	if err != nil {
		return nil, err
	}
	return ScriptFromBuildResult(result)
}

func (r *Runner) moduleSelector() (string, string) {
	if r == nil || r.module == nil {
		return "", ""
	}
	moduleName := strings.TrimSpace(r.module.Name)
	appName := strings.TrimSpace(r.module.ApplicationStr)
	if appName == "" {
		appName = moduleName
	}
	return appName, moduleName
}

func (r *Runner) buildHookWrapperScript() *jsengine.JsScript {
	if r == nil || r.module == nil {
		return nil
	}
	content := `(() => {
	const resolveModuleRoot = (app, moduleName) => {
		const root = globalThis[app];
		return root && root[moduleName];
	};

	globalThis.__choysum_hook__ = async function (app, moduleName, phase) {
		if (!app || !moduleName || !phase) {
			return;
		}
		globalThis.CHOYSUM_APP_NAME = app;
		globalThis.CHOYSUM_MODULE_NAME = moduleName;
		const moduleRoot = resolveModuleRoot(app, moduleName);
		const registryRoot = moduleRoot && moduleRoot.__hookRegistry__;
		const registry = registryRoot && registryRoot[phase];
		const hook = moduleRoot && moduleRoot.hook;
		if (!registry || registry.length === 0) {
      return;
    }
    for (const name of registry) {
      const fn = hook && hook[name];
      if (typeof fn !== 'function') {
		throw new Error('HOOK_UNSUPPORTED: hook not found ' + app + '.' + moduleName + '.hook.' + name);
      }
      await fn();
    }
  };
})();`
	return &jsengine.JsScript{FileName: "hook_wrapper.js", Content: content}
}

type execCtxPayload struct {
	requestId string
	payload   map[string]any
}

func (r *Runner) buildJsContext(ctx context.Context, opts RunOptions) execCtxPayload {
	traceId := ""
	spanId := ""
	span := trace.SpanFromContext(ctx)
	if span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			traceId = sc.TraceID().String()
			spanId = sc.SpanID().String()
		}
	}
	if traceId == "" {
		traceId = xid.New().String()
	}
	if spanId == "" {
		spanId = xid.New().String()
	}

	userId := "admin"
	if id := auth.IdentityFromContext(ctx); id != nil && id.IsValid() {
		if v := strings.TrimSpace(id.GetUserID()); v != "" {
			userId = v
		}
	}

	payload := map[string]any{
		"ctx": map[string]any{},
		"identity": map[string]any{
			"userId": userId,
		},
		"req": map[string]any{
			"requestId": spanId,
			"traceId":   traceId,
			"depth":     0,
			"kind":      "grpc",
		},
		"module": map[string]any{
			"name":        r.module.Name,
			"version":     r.module.Version,
			"fromVersion": opts.FromVersion,
		},
	}
	return execCtxPayload{requestId: spanId, payload: payload}
}

func contextWithEffectiveScope(ctx context.Context, runtimeScope scope.Scope) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := scope.SessionFromContext(ctx); ok {
		return ctx
	}
	if _, ok := scope.ScopeFromContext(ctx); ok {
		return ctx
	}
	if runtimeScope == nil {
		return ctx
	}
	return scope.ContextWithScope(ctx, runtimeScope)
}

func (r *Runner) buildExecContext(ctx context.Context) context.Context {
	execCtx := contextWithEffectiveScope(ctx, r.runtimeScope)
	if _, ok := auth.AccessTokenFromContext(execCtx); !ok {
		if key := strings.TrimSpace(runtimeOptionsFromScope(r.runtimeScope).authInternalKey); key != "" {
			execCtx = auth.ContextWithInternalKey(execCtx, key)
		}
	}

	md := metadata.New(nil)
	md.Set("x-choysum-depth", "0")
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			traceparent := fmt.Sprintf("00-%s-%s-01", sc.TraceID().String(), sc.SpanID().String())
			md.Set("traceparent", traceparent)
		}
	}
	execCtx = metadata.NewIncomingContext(execCtx, md)
	return execCtx
}

var errHookRetriable = errors.New("hook retriable")

func equivalentScripts(a []*jsengine.JsScript, b []*jsengine.JsScript) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] == nil && b[i] == nil {
			continue
		}
		if a[i] == nil || b[i] == nil {
			return false
		}
		if a[i].FileName != b[i].FileName || a[i].Content != b[i].Content {
			return false
		}
	}
	return true
}

func (r *Runner) executeWithScripts(ctx context.Context, scripts []*jsengine.JsScript, req *jsengine.JsRequest, lastErr *error, reuseExecutorScripts bool) error {
	prevScripts := r.jsExecutor.GetJsScripts()
	changedScripts := len(scripts) > 0 && !equivalentScripts(prevScripts, scripts)
	if changedScripts {
		r.jsExecutor.SetJsScripts(scripts)
		if err := r.jsExecutor.Reload(scripts...); err != nil {
			r.jsExecutor.SetJsScripts(prevScripts)
			_ = r.jsExecutor.Reload(prevScripts...)
			return err
		}
	}
	if changedScripts && !reuseExecutorScripts {
		defer func() {
			r.jsExecutor.SetJsScripts(prevScripts)
			_ = r.jsExecutor.Reload(prevScripts...)
		}()
	}

	resp, err := r.jsExecutor.Execute(ctx, req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			*lastErr = fmt.Errorf("HOOK_TIMEOUT: %w", err)
			return errHookRetriable
		}
		code, msg := parseHookError(err)
		if code == "" {
			code = "HOOK_EXCEPTION"
			msg = err.Error()
		}
		*lastErr = fmt.Errorf("%s: %s", code, msg)
		if code == "HOOK_UNSUPPORTED" {
			return *lastErr
		}
		return errHookRetriable
	}
	if resp == nil {
		*lastErr = fmt.Errorf("HOOK_EXCEPTION: empty response")
		return errHookRetriable
	}
	if resp.Result == nil {
		return nil
	}
	return nil
}

type resolvedConfig struct {
	Timeout  time.Duration
	Retry    int
	Required bool
}

func normalizeConfig(phase Phase) resolvedConfig {
	return resolvedConfig{
		Timeout:  defaultTimeout(phase),
		Retry:    0,
		Required: defaultRequired(phase),
	}
}

func defaultRequired(phase Phase) bool {
	switch phase {
	case PhasePreInit, PhasePreUpgrade, PhasePreUninstall:
		return true
	default:
		return false
	}
}

func defaultTimeout(phase Phase) time.Duration {
	switch phase {
	case PhasePreInit, PhasePreUpgrade, PhasePreUninstall:
		return 30 * time.Second
	default:
		return 60 * time.Second
	}
}

func parseHookError(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	msg := err.Error()
	idx := strings.Index(msg, "HOOK_")
	if idx < 0 {
		return "", msg
	}
	trimmed := strings.TrimSpace(msg[idx:])
	parts := strings.SplitN(trimmed, ":", 2)
	code := strings.TrimSpace(parts[0])
	detail := ""
	if len(parts) > 1 {
		detail = strings.TrimSpace(parts[1])
	}
	return code, detail
}

func LoadDistScripts(runtimeScope scope.Scope, module *meta.IrModule) ([]*jsengine.JsScript, error) {
	if runtimeScope == nil || module == nil {
		return nil, fmt.Errorf("missing env or module")
	}
	if strings.TrimSpace(module.ServiceEntryPoint) == "" {
		return nil, nil
	}
	runtimeOpts := runtimeOptionsFromScope(runtimeScope)
	mode := strings.ToLower(strings.TrimSpace(runtimeOpts.compileBundleMode))
	if mode == "" {
		mode = "bundle"
	}
	var scriptPath string
	if mode == "bundle" {
		scriptPath = config.BundlesIndexJS(runtimeOpts.distPath)
	} else {
		app := strings.TrimSpace(module.ApplicationStr)
		if app == "" {
			app = strings.TrimSpace(module.Name)
		}
		if app == "" {
			return nil, fmt.Errorf("module application is empty")
		}
		scriptPath = config.AppIndexJS(runtimeOpts.distPath, app)
	}
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			if strings.TrimSpace(module.ServiceEntryPoint) == "" {
				return nil, nil
			}
		}
		return nil, err
	}
	return []*jsengine.JsScript{{FileName: scriptPath, Content: string(content)}}, nil
}

func ScriptFromBuildResult(result *module.BuildResult) (*jsengine.JsScript, error) {
	if result == nil || result.EsbuildResult == nil {
		return nil, nil
	}
	var chosen *jsengine.JsScript
	for _, out := range result.EsbuildResult.OutputFiles {
		if len(out.Contents) == 0 {
			continue
		}
		if strings.HasSuffix(out.Path, "index.js") {
			chosen = &jsengine.JsScript{FileName: out.Path, Content: string(out.Contents)}
			break
		}
		if chosen == nil && strings.HasSuffix(out.Path, ".js") {
			chosen = &jsengine.JsScript{FileName: out.Path, Content: string(out.Contents)}
		}
	}
	return chosen, nil
}
