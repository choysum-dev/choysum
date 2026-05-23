// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import "testing"

func TestResolveHostRuntimeReturnsZeroRuntimeForNilProvider(t *testing.T) {
	runtime := ResolveHostRuntime(nil, nil)
	if runtime.Queue != nil || runtime.Store != nil || runtime.Events != nil || runtime.Collector != nil {
		t.Fatal("expected zero runtime bundle when no host runtime provider is configured")
	}
}

func TestStaticHostRuntimeProviderReturnsBundledRuntime(t *testing.T) {
	collector := &runtimeTestCollector{}
	bundled := Runtime{Collector: collector}

	resolved := ResolveHostRuntime(StaticHostRuntimeProvider(bundled), nil)
	if resolved.Collector != collector {
		t.Fatal("expected static host runtime provider to return the bundled runtime")
	}
}

type runtimeTestCollector struct{}

func (*runtimeTestCollector) Start() {}

func (*runtimeTestCollector) Stop() {}
