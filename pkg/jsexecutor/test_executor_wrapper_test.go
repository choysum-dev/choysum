// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jsexecutor

import (
	"context"
	"errors"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

var errThreadPoolNotInitializedForTest = errors.New("executor-not-initialized: thread pool is not initialized")

type testExecutorWrapper struct {
	impl JsExecutor
}

func newTestExecutorWrapper(backend JsExecutor) JsExecutor {
	if backend == nil {
		return &testExecutorWrapper{}
	}
	if wrapped, ok := backend.(*testExecutorWrapper); ok {
		return wrapped
	}
	return &testExecutorWrapper{impl: backend}
}

func (e *testExecutorWrapper) delegate() JsExecutor {
	if e == nil {
		return nil
	}
	return e.impl
}

func (e *testExecutorWrapper) requireDelegate() (JsExecutor, error) {
	impl := e.delegate()
	if impl == nil {
		return nil, errThreadPoolNotInitializedForTest
	}
	return impl, nil
}

func (e *testExecutorWrapper) GetJsScripts() []*jsengine.JsScript {
	impl := e.delegate()
	if impl == nil {
		return nil
	}
	return impl.GetJsScripts()
}

func (e *testExecutorWrapper) SetJsScripts(scripts []*jsengine.JsScript) {
	impl := e.delegate()
	if impl == nil {
		return
	}
	impl.SetJsScripts(scripts)
}

func (e *testExecutorWrapper) AppendJsScripts(scripts ...*jsengine.JsScript) {
	impl := e.delegate()
	if impl == nil {
		return
	}
	impl.AppendJsScripts(scripts...)
}

func (e *testExecutorWrapper) Start() error {
	impl, err := e.requireDelegate()
	if err != nil {
		return err
	}
	return impl.Start()
}

func (e *testExecutorWrapper) Execute(ctx context.Context, request *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	impl, err := e.requireDelegate()
	if err != nil {
		return nil, err
	}
	return impl.Execute(ctx, request)
}

func (e *testExecutorWrapper) Stop() error {
	impl, err := e.requireDelegate()
	if err != nil {
		return err
	}
	return impl.Stop()
}

func (e *testExecutorWrapper) Reload(scripts ...*jsengine.JsScript) error {
	impl, err := e.requireDelegate()
	if err != nil {
		return err
	}
	return impl.Reload(scripts...)
}
