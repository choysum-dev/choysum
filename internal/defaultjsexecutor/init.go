// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package defaultjsexecutor

import (
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
)

const defaultFactoryName = "default"

func init() {
	jsexecutor.RegisterRuntimeFactory(defaultFactoryName, defaultRuntimeFactory)
	jsexecutor.RegisterCompilerFactory(defaultFactoryName, defaultCompilerFactory)
}

func defaultRuntimeFactory(runtimeScope scope.Scope, authenticator auth.Authenticator, opts ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
	internalOpts := buildInternalOptions(opts...)
	impl, err := NewRuntimeExecutor(runtimeScope, authenticator, internalOpts...)
	if err != nil {
		return nil, err
	}
	return impl, nil
}

func defaultCompilerFactory(runtimeScope scope.Scope, opts ...jsexecutor.Option) (jsexecutor.JsExecutor, error) {
	internalOpts := buildInternalOptions(opts...)
	impl, err := NewCompilerExecutor(runtimeScope, internalOpts...)
	if err != nil {
		return nil, err
	}
	return impl, nil
}

func buildInternalOptions(opts ...jsexecutor.Option) []func(*JsExecutor) {
	factoryOptions := jsexecutor.ResolveFactoryOptions(opts...)
	internalOpts := make([]func(*JsExecutor), 0, 10)
	if factoryOptions.JsEngineFactory != nil {
		internalOpts = append(internalOpts, WithJsEngine(factoryOptions.JsEngineFactory))
	}
	if factoryOptions.Logger != nil {
		internalOpts = append(internalOpts, WithLogger(factoryOptions.Logger))
	}
	if len(factoryOptions.JsScripts) > 0 {
		internalOpts = append(internalOpts, WithJsScripts(factoryOptions.JsScripts...))
	}
	if factoryOptions.MinPoolSize > 0 {
		internalOpts = append(internalOpts, WithMinPoolSize(factoryOptions.MinPoolSize))
	}
	if factoryOptions.MaxPoolSize > 0 {
		internalOpts = append(internalOpts, WithMaxPoolSize(factoryOptions.MaxPoolSize))
	}
	if factoryOptions.QueueSize > 0 {
		internalOpts = append(internalOpts, WithQueueSize(factoryOptions.QueueSize))
	}
	if factoryOptions.ThreadTTL > 0 {
		internalOpts = append(internalOpts, WithThreadTTL(factoryOptions.ThreadTTL))
	}
	if factoryOptions.MaxExecutions > 0 {
		internalOpts = append(internalOpts, WithMaxExecutions(factoryOptions.MaxExecutions))
	}
	if factoryOptions.ExecuteTimeout > 0 {
		internalOpts = append(internalOpts, WithExecuteTimeout(factoryOptions.ExecuteTimeout))
	}
	if factoryOptions.CreateThreshold > 0 {
		internalOpts = append(internalOpts, WithCreateThreshold(factoryOptions.CreateThreshold))
	}
	if factoryOptions.SelectThreshold > 0 {
		internalOpts = append(internalOpts, WithSelectThreshold(factoryOptions.SelectThreshold))
	}
	return internalOpts
}
