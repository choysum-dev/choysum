// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/quickjsengine"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysummount"
	xfmt "golang.org/x/exp/errors/fmt"
)

// InstallMinimalDOM injects the FE unit minimal DOM polyfill into a QuickJS engine.
// Scope is frozen for PR-unit-vue-host: document/window/Element enough for Vue mount,
// querySelector, and dispatchEvent — not a full happy-dom/jsdom.
func InstallMinimalDOM(engine jsengine.JsEngine) error {
	if engine == nil {
		return xfmt.Errorf("frontend host: nil engine")
	}
	if err := engine.Load([]*jsengine.JsScript{
		{FileName: "scripts/choysummount/dom.js", Content: choysummount.MinimalDOMScript},
	}); err != nil {
		return xfmt.Errorf("frontend host: InstallMinimalDOM: %w", err)
	}
	return nil
}

// PrepareVueHostEngine installs minimal DOM and bootstraps QuickJS timers (setTimeout).
// Call before loading a Vue host bundle that uses flushPromises / async updates.
func PrepareVueHostEngine(engine jsengine.JsEngine) error {
	if err := InstallMinimalDOM(engine); err != nil {
		return err
	}
	qjs, ok := engine.(*quickjsengine.QuickjsEngine)
	if !ok {
		return nil
	}
	if !qjs.Ctx.BootstrapTimers() {
		return xfmt.Errorf("frontend host: BootstrapTimers failed")
	}
	return nil
}
