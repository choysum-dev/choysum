// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jsexecutor

import (
	"context"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

// JsExecutor is the full facade contract exposed to runtime owners.
// It extends ScriptExecutor with lifecycle controls and script append support.
type JsExecutor interface {
	ScriptExecutor
	AppendJsScripts(scripts ...*jsengine.JsScript)
	Start() error
	Stop() error
}

// RuntimeInfo is a minimal read-only executor view exposed for runtime
// diagnostics without widening the main lifecycle contract.
type RuntimeInfo struct {
	MinPoolSize uint32
	MaxPoolSize uint32
}

// RuntimeInfoReader exposes optional read-only executor state for owners that
// want to enrich diagnostics without depending on a concrete implementation.
type RuntimeInfoReader interface {
	RuntimeInfo() RuntimeInfo
}

// ScriptExecutor is the script lifecycle contract shared by module hooks,
// migrations, and plugin/builder call sites.
type ScriptExecutor interface {
	Execute(ctx context.Context, request *jsengine.JsRequest) (*jsengine.JsResponse, error)
	GetJsScripts() []*jsengine.JsScript
	SetJsScripts(scripts []*jsengine.JsScript)
	Reload(scripts ...*jsengine.JsScript) error
}
