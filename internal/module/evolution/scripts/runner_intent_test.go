// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/evolution/schema"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestWithIntentBag_AndExecuteAttachesBag(t *testing.T) {
	WithIntentBag(schema.NewMemoryIntentBag())(nil)

	bag := schema.NewMemoryIntentBag()
	mod := &meta.Module{Name: "sales", ApplicationStr: "sales", Version: "1.0.0"}
	var sawBag bool
	exec := &intentCaptureExecutor{onExecute: func(ctx context.Context) {
		if schema.IntentBagFromContext(ctx) == bag {
			sawBag = true
		}
	}}
	runner := NewRunner(newScriptsTestScope(t), exec, mod, WithIntentBag(bag), nil)
	if runner == nil || runner.intents != bag {
		t.Fatal("runner intents")
	}
	_, err := runner.executeWithScripts(context.Background(), nil, &jsengine.JsRequest{Id: "1"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !sawBag {
		t.Fatal("expected IntentBag on exec context")
	}
}

type intentCaptureExecutor struct {
	onExecute func(context.Context)
	scripts   []*jsengine.JsScript
}

func (e *intentCaptureExecutor) Execute(ctx context.Context, _ *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if e.onExecute != nil {
		e.onExecute(ctx)
	}
	return &jsengine.JsResponse{}, nil
}
func (e *intentCaptureExecutor) GetJsScripts() []*jsengine.JsScript { return e.scripts }
func (e *intentCaptureExecutor) SetJsScripts(scripts []*jsengine.JsScript) {
	e.scripts = scripts
}
func (e *intentCaptureExecutor) Reload(...*jsengine.JsScript) error { return nil }
