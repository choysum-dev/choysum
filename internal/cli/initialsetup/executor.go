// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initialsetup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/evolution/scripts"
	"github.com/choysum-dev/choysum/internal/module/evolution/systemidentity"
	"github.com/choysum-dev/choysum/internal/provision"
	"github.com/choysum-dev/choysum/internal/server/runplan"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
)

var (
	newRuntimeExecutor = jsexecutor.NewRuntimeExecutor
	planServe          = runplan.Plan
)

// ScopeDeps reads workspace state from runtimeScope and opens a runtime
// executor over the installed modules only when a service call is needed.
func ScopeDeps(runtimeScope scope.Scope) Deps {
	return Deps{
		Phase:      func() (provision.Phase, error) { return provision.Resolve(runtimeScope) },
		OpenCaller: func() (Caller, func(), error) { return openExecutorCaller(runtimeScope) },
	}
}

// openExecutorCaller starts a runtime executor loaded with the serve scripts
// and returns a Caller over it plus the func that stops it.
func openExecutorCaller(runtimeScope scope.Scope) (Caller, func(), error) {
	serveScripts, err := loadServeScripts(runtimeScope)
	if err != nil {
		return nil, nil, err
	}
	executor, err := newRuntimeExecutor(runtimeScope, nil)
	if err != nil {
		return nil, nil, err
	}
	executor.SetJsScripts(serveScripts)
	if err := executor.Start(); err != nil {
		_ = executor.Stop()
		return nil, nil, err
	}
	closeCaller := func() {
		if stopErr := executor.Stop(); stopErr != nil {
			runtimeScope.Logger().Warn("stop setup runtime executor", "error", stopErr)
		}
	}
	return executorCaller{executor: executor, scope: runtimeScope}, closeCaller, nil
}

// loadServeScripts returns the service init scripts `choysum run` would load
// for the installed apps (bundles/index.js, or apps/<name>/index.js per app).
func loadServeScripts(runtimeScope scope.Scope) ([]*jsengine.JsScript, error) {
	pathOpts, _ := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	distRoot := pathOpts.DistPath
	if strings.TrimSpace(distRoot) == "" {
		return nil, errors.New("dist path is not configured")
	}
	bundleMode := config.NewDefaultCompileConfig().BundleMode
	if compileOpts, ok := scope.CompileRuntimeOptionsFromScope(runtimeScope); ok && strings.TrimSpace(compileOpts.BundleMode) != "" {
		bundleMode = compileOpts.BundleMode
	}
	_, decision, err := planServe(distRoot, bundleMode, runtimeScope.Logger(), nil)
	if err != nil {
		return nil, err
	}
	if decision.RunMode != runplan.RunModeApplication {
		return nil, fmt.Errorf("installed apps are not ready to serve: %s", decision.Reason)
	}

	paths := []string{config.BundlesIndexJS(distRoot)}
	if strings.ToLower(strings.TrimSpace(decision.CompileBundleMode)) != "bundle" {
		paths = paths[:0]
		for _, app := range decision.ServeTargets {
			paths = append(paths, config.AppIndexJS(distRoot, app))
		}
	}
	serveScripts := make([]*jsengine.JsScript, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		serveScripts = append(serveScripts, &jsengine.JsScript{FileName: path, Content: string(content)})
	}
	if len(serveScripts) == 0 {
		return nil, fmt.Errorf("no service scripts found under %s; reinstall the modules to rebuild dist", distRoot)
	}
	return jsengine.DedupeInitScripts(serveScripts)
}

// executorCaller runs each call in its own transaction as the seeded
// bootstrap admin, the same principal lifecycle hooks and migrations use.
type executorCaller struct {
	executor jsexecutor.JsExecutor
	scope    scope.Scope
}

// Call executes service with args and returns the JS result; errors are wrapped
// with the service name and keep the underlying error chain.
func (c executorCaller) Call(ctx context.Context, service string, lang string, args ...any) (any, error) {
	userID, companyID, err := systemidentity.ResolvePrincipalUserID(ctx, c.scope, "")
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", service, err)
	}
	reqCtx := systemidentity.CompanyContextPayload(companyID)
	reqCtx["tz"] = "UTC"
	if lang != "" {
		reqCtx["lang"] = lang
	}
	requestID := "cli-setup-" + xid.New().String()
	req := &jsengine.JsRequest{
		Id:      requestID,
		Service: service,
		Args:    args,
		Context: map[string]any{
			"ctx":      reqCtx,
			"identity": map[string]any{"userId": userID},
			"req": map[string]any{
				"requestId": requestID,
				"traceId":   requestID,
				"depth":     0,
				"kind":      "grpc",
			},
		},
	}

	var result any
	err = c.scope.WithContext(ctx).Transactor().Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
		resp, execErr := c.executor.Execute(scripts.BuildExecContext(scope.ContextWithScope(ctx, txScope), txScope), req)
		if execErr != nil {
			return execErr
		}
		if resp != nil {
			result = resp.Result
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", service, err)
	}
	return result, nil
}
