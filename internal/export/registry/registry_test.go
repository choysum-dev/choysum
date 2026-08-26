// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry_test

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/registry"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type stubReader struct{}

func (stubReader) Read(context.Context, scope.Scope, plan.Plan) (registry.Result, error) {
	return registry.Result{}, nil
}

func TestReaderFor(t *testing.T) {
	t.Cleanup(func() {
		registry.ResetForTest()
	})

	_, err := registry.ReaderFor(exportpkg.ProfileRecord)
	if !errors.Is(err, exportpkg.ErrReaderNotRegistered) {
		t.Fatalf("ReaderFor() error = %v, want ErrReaderNotRegistered", err)
	}

	registry.Register(exportpkg.ProfileRecord, stubReader{})
	r, err := registry.ReaderFor(exportpkg.ProfileRecord)
	if err != nil || r == nil {
		t.Fatalf("ReaderFor() = %v, %v", r, err)
	}

	registry.Register(exportpkg.ProfileRecord, nil)
	_, err = registry.ReaderFor(exportpkg.ProfileRecord)
	if !errors.Is(err, exportpkg.ErrReaderNotRegistered) {
		t.Fatalf("ReaderFor(nil) error = %v, want ErrReaderNotRegistered", err)
	}
}
