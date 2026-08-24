// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ormbridge

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/xid"
)

// CallRequest is the stable ORM invoke shape used by RecordWriter and $choysum.orm.call.
type CallRequest struct {
	Model   string         `json:"model"`
	Method  string         `json:"method"`
	Args    []any          `json:"args"`
	Context map[string]any `json:"context,omitempty"`
}

// Caller invokes TS Model methods (Create / UpdateById / Search / …) through the ORM path.
type Caller interface {
	Call(ctx context.Context, req CallRequest) (any, error)
}

type callerContextKey struct{}

// ContextWithCaller stores an ORM caller for RecordWriter (and tests).
func ContextWithCaller(ctx context.Context, caller Caller) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, callerContextKey{}, caller)
}

// CallerFromContext loads the ORM caller injected for this import run.
func CallerFromContext(ctx context.Context) (Caller, bool) {
	if ctx == nil {
		return nil, false
	}
	caller, ok := ctx.Value(callerContextKey{}).(Caller)
	return caller, ok && caller != nil
}

// ServiceName builds the __rpc__ service path "app.Model.Method".
func ServiceName(model, method string) (string, error) {
	model = strings.TrimSpace(model)
	method = strings.TrimSpace(method)
	if model == "" || method == "" {
		return "", fmt.Errorf("model and method are required")
	}
	if !strings.Contains(model, ".") {
		return "", fmt.Errorf("model must be app.Model, got %q", model)
	}
	return model + "." + method, nil
}

// MergeImportContext ensures import_file=true and merges caller-provided context keys.
// import_file is reserved and always forced true after copying extras.
func MergeImportContext(extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range extra {
		if strings.TrimSpace(k) == "" {
			continue
		}
		out[k] = v
	}
	out["import_file"] = true
	return out
}

// NewRequestID returns a unique JsRequest id for ORM calls.
func NewRequestID() string {
	return "import-orm-" + xid.New().String()
}
