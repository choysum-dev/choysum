// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry_test

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/plan"
	"github.com/choysum-dev/choysum/internal/import/registry"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type stubWriter struct{}

func (stubWriter) Write(context.Context, scope.Scope, []plan.Unit) error { return nil }

func TestWriterFor(t *testing.T) {
	t.Cleanup(registry.ResetWritersForTest)

	_, err := registry.WriterFor(importpkg.ProfileRecord)
	if !errors.Is(err, importpkg.ErrWriterNotRegistered) {
		t.Fatalf("WriterFor() error = %v, want ErrWriterNotRegistered", err)
	}

	registry.RegisterWriter(importpkg.ProfileRecord, stubWriter{})
	w, err := registry.WriterFor(importpkg.ProfileRecord)
	if err != nil || w == nil {
		t.Fatalf("WriterFor() = %v, %v", w, err)
	}

	registry.RegisterWriter(importpkg.ProfileRecord, nil)
	_, err = registry.WriterFor(importpkg.ProfileRecord)
	if !errors.Is(err, importpkg.ErrWriterNotRegistered) {
		t.Fatalf("WriterFor(nil) error = %v, want ErrWriterNotRegistered", err)
	}
}
