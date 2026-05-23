// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jsexecutortest

import (
	"context"
	"errors"

	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
)

var errExecutorNotInitialized = errors.New("executor-not-initialized: thread pool is not initialized")

type uninitializedExecutor struct{}

func NewUninitializedExecutor() jsexecutor.JsExecutor {
	return &uninitializedExecutor{}
}

func (*uninitializedExecutor) GetJsScripts() []*jsengine.JsScript {
	return nil
}

func (*uninitializedExecutor) SetJsScripts(_ []*jsengine.JsScript) {}

func (*uninitializedExecutor) AppendJsScripts(_ ...*jsengine.JsScript) {}

func (*uninitializedExecutor) Start() error {
	return errExecutorNotInitialized
}

func (*uninitializedExecutor) Execute(_ context.Context, _ *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	return nil, errExecutorNotInitialized
}

func (*uninitializedExecutor) Stop() error {
	return errExecutorNotInitialized
}

func (*uninitializedExecutor) Reload(_ ...*jsengine.JsScript) error {
	return errExecutorNotInitialized
}
