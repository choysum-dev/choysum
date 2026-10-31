// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hooks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	internalbackendbuilder "github.com/choysum-dev/choysum/internal/module/artifact/build/backend"
	module "github.com/choysum-dev/choysum/internal/module/artifact/result"
	"github.com/choysum-dev/choysum/internal/module/evolution/systemidentity"
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
	PhaseReady         Phase = "ready"
)

type RunOptions struct {
	Scripts     []*jsengine.JsScript
	FromVersion string
	// ReuseExecutorScripts keeps loaded scripts on the shared executor between
	// phase runs so callers can amortize reload costs across module loops.
	ReuseExecutorScripts bool
	// PreserveExecutorScripts appends this run's hook scripts onto the executor's
	// current set (instead of replacing it), then restores the previous set after
	// Execute. Fallback only; prefer InvokeOnly when application init already
	// registered hooks under CHOYSUM_* seeds.
	PreserveExecutorScripts bool
	// InvokeOnly runs __choysum_hook__ against the executor's current scripts
	// without Bundle / SetJsScripts / Reload. Requires Start-time seeded entry
	// registration and a resident dispatcher in application init.
	InvokeOnly bool
}

type Runner struct {
	runtimeScope scope.Scope
	jsExecutor   jsexecutor.ScriptExecutor
	module       *meta.Module
	// entryScript caches the last Bundle for this runner so multi-phase calls
	// (e.g. validate→pre→post) do not rebuild the same module entry.
	entryScript    *jsengine.JsScript
	entryScriptKey string
}

// newEntryModuleBuilder constructs the ModuleBuilder used to Bundle hook entry scripts.
// Tests may override to inject non-Bundler builders.
var newEntryModuleBuilder = func(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.Module, entryPoint string) module.Builder {
	return internalbackendbuilder.NewModuleBuilder(runtimeScope, jsExecutor, module, entryPoint, internalbackendbuilder.WithPublishDist(false))
}

func NewRunner(runtimeScope scope.Scope, jsExecutor jsexecutor.ScriptExecutor, module *meta.Module) (*Runner, error) {
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

	// Empty phase: skip Bundle / Reload when sources declare no @Hook* for phase.
	// Fail-open scan (missing Path / I/O) still loads so real hooks are not skipped.
	if !moduleSourceDeclaresHookPhase(r.module, phase) {
		return nil
	}

	resolved := normalizeConfig(phase)
	appName, moduleName := r.moduleSelector()
	phaseName := strings.TrimSpace(string(phase))

	attempts := 1
	if resolved.Retry > 0 {
		attempts = resolved.Retry + 1
	}

	var scripts []*jsengine.JsScript
	if !opts.InvokeOnly {
		var moduleScoped bool
		var err error
		scripts, moduleScoped, err = r.resolveScripts(ctx, phase, opts, resolved.Required)
		if err != nil {
			if !resolved.Required {
				if r.runtimeScope != nil {
					r.runtimeScope.Logger().Warn("hook script unavailable", "module", r.module.Name, "phase", string(phase), "reason", "ignored", "error", err)
				}
				return nil
			}
			return err
		}
		// Module-scoped entry: seed CHOYSUM_* before eval so @Hook* decorators register.
		// Shared dist fallback: do not seed before eval (would mis-attribute other modules'
		// decorators); identity is still set when __choysum_hook__ runs.
		if wrapper := r.buildHookWrapperScript(appName, moduleName, moduleScoped); wrapper != nil {
			if moduleScoped {
				scripts = append([]*jsengine.JsScript{wrapper}, scripts...)
			} else {
				scripts = append(scripts, wrapper)
			}
		}
	}

	jsCtx, jsCtxErr := r.buildJsContext(ctx, opts)
	if jsCtxErr != nil {
		if !resolved.Required {
			if r.runtimeScope != nil {
				r.runtimeScope.Logger().Warn("hook identity unavailable", "module", r.module.Name, "phase", phaseName, "reason", "ignored", "error", jsCtxErr)
			}
			return nil
		}
		return jsCtxErr
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		execCtx := r.buildExecContext(ctx)
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

		var execErr error
		if opts.InvokeOnly {
			execErr = r.executeInvokeOnly(ctxWithTimeout, req, &lastErr)
		} else {
			execErr = r.executeWithScripts(ctxWithTimeout, scripts, req, &lastErr, opts.ReuseExecutorScripts, opts.PreserveExecutorScripts)
		}
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

func (r *Runner) resolveScripts(ctx context.Context, phase Phase, opts RunOptions, required bool) (scripts []*jsengine.JsScript, moduleScoped bool, err error) {
	if len(opts.Scripts) > 0 {
		return opts.Scripts, true, nil
	}
	script, buildErr := r.buildModuleEntryScript(ctx)
	if buildErr != nil {
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
			return nil, false, fmt.Errorf("required hook phase %s: failed to build module entry script (module=%q, entry=%q): %w", phase, moduleName, entry, buildErr)
		}
	} else if script != nil {
		return []*jsengine.JsScript{script}, true, nil
	}
	loaded, loadErr := LoadDistScripts(deriveRuntimeScope(ctx, r.runtimeScope), r.module)
	return loaded, false, loadErr
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
	cacheKey := entryScriptCacheKey(entry, runtimeScope)
	if r.entryScript != nil && r.entryScriptKey == cacheKey {
		return r.entryScript, nil
	}
	builder := newEntryModuleBuilder(runtimeScope, r.jsExecutor, r.module, entry)
	bundler, ok := builder.(module.Bundler)
	if !ok {
		return nil, fmt.Errorf("module builder does not support Bundle")
	}
	result, err := bundler.Bundle()
	if err != nil {
		return nil, err
	}
	script, _ := ScriptFromBuildResult(result)
	r.entryScript = script
	r.entryScriptKey = cacheKey
	return script, nil
}

// entryScriptCacheKey ties Bundle reuse to the entry path and DB session identity so
// a context that injects a different Session (installed-module imports) still rebuilds.
func entryScriptCacheKey(entry string, runtimeScope scope.Scope) string {
	token := ""
	if runtimeScope != nil {
		if sess := runtimeScope.Session(); sess != nil {
			// Session identity (not *gorm.DB): nested wrappers around the same DB
			// still rebuild when the Session object differs.
			token = fmt.Sprintf("%p", sess)
		}
	}
	return entry + "|" + token
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

func (r *Runner) buildHookWrapperScript(appName, moduleName string, seedIdentityBeforeEval bool) *jsengine.JsScript {
	if r == nil || r.module == nil {
		return nil
	}
	appLit := strconv.Quote(strings.TrimSpace(appName))
	modLit := strconv.Quote(strings.TrimSpace(moduleName))
	seed := ""
	if seedIdentityBeforeEval {
		seed = `globalThis.CHOYSUM_APP_NAME = ` + appLit + `;
	globalThis.CHOYSUM_MODULE_NAME = ` + modLit + `;
`
	}
	// Hook handlers are module-level callbacks (static, no this). Invoke with bare
	// `await fn()` only — do not switch to fn.call/fn.apply unless the lifecycle
	// this-boundary plan is explicitly upgraded past the near-term contract.
	// See .dev/docs/core/service/lifecycle_hook_this_boundary_plan20260715.md.
	// Resolve via __choysum_hook_module_roots__ first so ApplicationStr IIFE
	// reassignment of globalThis[app] cannot drop the registered registry.
	content := `(() => {
	` + seed + `const resolveModuleRoot = (app, moduleName) => {
		const stable = globalThis.__choysum_hook_module_roots__;
		const fromStable = stable && stable[app] && stable[app][moduleName];
		if (fromStable) return fromStable;
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

func (r *Runner) buildJsContext(ctx context.Context, opts RunOptions) (execCtxPayload, error) {
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

	sessionUserID := ""
	if id := auth.IdentityFromContext(ctx); id != nil && id.IsValid() {
		sessionUserID = strings.TrimSpace(id.GetUserID())
	}
	userId, companyID, err := systemidentity.ResolvePrincipalUserID(ctx, r.runtimeScope, sessionUserID)
	if err != nil {
		return execCtxPayload{}, err
	}

	payload := map[string]any{
		"ctx": systemidentity.CompanyContextPayload(companyID),
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
	return execCtxPayload{requestId: spanId, payload: payload}, nil
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

// executeInvokeOnly calls __choysum_hook__ on the shared runtime without touching
// the executor script set (no Bundle / SetJsScripts / Reload).
func (r *Runner) executeInvokeOnly(ctx context.Context, req *jsengine.JsRequest, lastErr *error) error {
	resp, err := r.jsExecutor.Execute(ctx, req)
	return r.finishHookExecute(resp, err, lastErr)
}

func (r *Runner) executeWithScripts(ctx context.Context, scripts []*jsengine.JsScript, req *jsengine.JsRequest, lastErr *error, reuseExecutorScripts bool, preserveExecutorScripts bool) error {
	prevScripts := r.jsExecutor.GetJsScripts()
	scriptsToLoad := scripts
	restoreAfter := false
	if preserveExecutorScripts && len(scripts) > 0 {
		if len(prevScripts) > 0 {
			scriptsToLoad = append(append([]*jsengine.JsScript{}, prevScripts...), scripts...)
		}
		// Always restore when preserve was requested, including empty prevScripts,
		// so hook scripts do not leak into later shared-executor runs.
		restoreAfter = true
	}
	changedScripts := len(scriptsToLoad) > 0 && !equivalentScripts(prevScripts, scriptsToLoad)
	if changedScripts {
		r.jsExecutor.SetJsScripts(scriptsToLoad)
		if err := r.jsExecutor.Reload(scriptsToLoad...); err != nil {
			r.jsExecutor.SetJsScripts(prevScripts)
			_ = r.jsExecutor.Reload(prevScripts...)
			return err
		}
	}
	if changedScripts && (restoreAfter || !reuseExecutorScripts) {
		defer func() {
			r.jsExecutor.SetJsScripts(prevScripts)
			_ = r.jsExecutor.Reload(prevScripts...)
		}()
	}

	resp, err := r.jsExecutor.Execute(ctx, req)
	return r.finishHookExecute(resp, err, lastErr)
}

func (r *Runner) finishHookExecute(resp *jsengine.JsResponse, err error, lastErr *error) error {
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
	case PhasePreInit, PhasePreUpgrade, PhasePreUninstall, PhaseReady:
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

func LoadDistScripts(runtimeScope scope.Scope, module *meta.Module) ([]*jsengine.JsScript, error) {
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
