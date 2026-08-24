// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/plan"
	planstub "github.com/choysum-dev/choysum/internal/import/plan/stub"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

func TestWriter_WriteUnexpectedUnitType(t *testing.T) {
	err := (Writer{}).Write(context.Background(), nil, []plan.Unit{planstub.Unit{Index: 1}})
	var impErr *importpkg.Error
	if !errors.As(err, &impErr) || impErr.Code != importpkg.CodeInvalidFormat {
		t.Fatalf("Write() error = %v, want CodeInvalidFormat", err)
	}
}

func TestWriter_WriteNilUnits(t *testing.T) {
	if err := (Writer{}).Write(context.Background(), nil, nil); err != nil {
		t.Fatalf("Write(nil): %v", err)
	}
}
