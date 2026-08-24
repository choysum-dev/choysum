// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"context"

	"github.com/choysum-dev/choysum/internal/import/plan"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	"github.com/choysum-dev/choysum/internal/import/registry"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func init() {
	registry.RegisterWriter(importpkg.ProfileRecord, Writer{})
}

// Writer persists record import units through the TS ORM write path (internal/import/caller).
// Target model comes from each unit's Model (Spec.Model); field/M2O handling uses meta.
type Writer struct{}

// Write implements registry.Writer.
func (Writer) Write(ctx context.Context, txScope scope.Scope, units []plan.Unit) error {
	if txScope == nil || txScope.Session() == nil {
		return importpkg.Errorf(importpkg.CodeInvalidFormat, "scope is required for record writer")
	}
	for _, unit := range units {
		u, ok := unit.(recordplan.Unit)
		if !ok {
			return importpkg.Errorf(importpkg.CodeInvalidFormat, "unexpected unit type for record writer")
		}
		if err := UpsertRecord(ctx, txScope, u); err != nil {
			return err
		}
	}
	return nil
}
