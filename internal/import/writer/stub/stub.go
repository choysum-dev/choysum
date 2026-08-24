// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package stub

import (
	"context"
	"fmt"

	"github.com/choysum-dev/choysum/internal/import/plan"
	planstub "github.com/choysum-dev/choysum/internal/import/plan/stub"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// Writer is a no-op persistence stub that can inject per-unit failures.
type Writer struct{}

// Write implements registry.Writer.
func (Writer) Write(ctx context.Context, txScope scope.Scope, units []plan.Unit) error {
	_ = ctx
	for _, unit := range units {
		su, ok := unit.(planstub.Unit)
		if !ok {
			continue
		}
		if su.Fail {
			return &importpkg.Error{
				Code: "constraint",
				Text: fmt.Sprintf("stub unit %d failed", su.Index),
				Row:  su.Index,
			}
		}
		if txScope != nil && txScope.Session() != nil {
			if err := txScope.Session().Create(&Row{UnitIndex: su.Index}).Error; err != nil {
				return importpkg.ErrorfWrap(importpkg.CodeInvalidFormat, "stub write failed", err)
			}
		}
	}
	return nil
}
