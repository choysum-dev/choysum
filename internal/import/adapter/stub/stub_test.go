// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package stub_test

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/adapter/stub"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestBuilder_Build(t *testing.T) {
	p, err := stub.Builder{}.Build(context.Background(), importpkg.Spec{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(p.Units) != 1 {
		t.Fatalf("default unit count = %d, want 1", len(p.Units))
	}

	p, err = stub.Builder{}.Build(context.Background(), importpkg.Spec{
		Options: importpkg.Options{StubUnitCount: 2, StubFailUnitIndex: 2},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(p.Units) != 2 {
		t.Fatalf("unit count = %d, want 2", len(p.Units))
	}
}
