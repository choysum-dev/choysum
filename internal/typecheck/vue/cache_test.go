// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue_test

import (
	"sync/atomic"
	"testing"

	"github.com/choysum-dev/choysum/internal/typecheck/vue"
)

type countingCoder struct {
	calls atomic.Int32
	inner vue.Coder
}

func (c *countingCoder) CreateServiceScript(path, source string, opts vue.CodegenOptions) (vue.ServiceScript, error) {
	c.calls.Add(1)
	return c.inner.CreateServiceScript(path, source, opts)
}

func TestCachedCoder_SameContentSkipsInner(t *testing.T) {
	fixture := "<script setup lang=\"ts\">const n = 1\n</script>\n<template>{{ n }}</template>\n"
	inner := vue.NewQuickJSCoder()
	t.Cleanup(func() { _ = inner.Close() })
	counter := &countingCoder{inner: inner}
	cached := vue.NewCachedCoder(counter)

	a, err := cached.CreateServiceScript("/a.vue", fixture, vue.CodegenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := cached.CreateServiceScript("/a.vue", fixture, vue.CodegenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if counter.calls.Load() != 1 {
		t.Fatalf("calls=%d want 1", counter.calls.Load())
	}
	if a.Content != b.Content || a.Content == "" {
		t.Fatal("cache miss or empty")
	}
	if _, err := cached.CreateServiceScript("/a.vue", fixture+" ", vue.CodegenOptions{}); err != nil {
		t.Fatal(err)
	}
	if counter.calls.Load() != 2 {
		t.Fatalf("calls=%d want 2 after source change", counter.calls.Load())
	}
}

func TestCachedCoder_Nil(t *testing.T) {
	var c *vue.CachedCoder
	if _, err := c.CreateServiceScript("a.vue", "", vue.CodegenOptions{}); err == nil {
		t.Fatal("expected nil error")
	}
	if _, err := vue.NewCachedCoder(nil).CreateServiceScript("a.vue", "", vue.CodegenOptions{}); err == nil {
		t.Fatal("expected nil inner error")
	}
}
