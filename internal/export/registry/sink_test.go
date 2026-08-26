// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry_test

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type stubSink struct{}

func (stubSink) Write(context.Context, scope.Scope, plan.Plan, *registry.Result) error {
	return nil
}

func TestSinkFor(t *testing.T) {
	t.Cleanup(registry.ResetSinksForTest)

	_, err := registry.SinkFor("csv")
	if err == nil {
		t.Fatal("expected unregistered sink error")
	}

	registry.RegisterSink("csv", stubSink{})
	s, err := registry.SinkFor("csv")
	if err != nil || s == nil {
		t.Fatalf("SinkFor() = %v, %v", s, err)
	}

	registry.RegisterSink("csv", nil)
	_, err = registry.SinkFor("csv")
	if err == nil {
		t.Fatal("expected unregistered sink error")
	}
}

var _ registry.Sink = stubSink{}
