// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/plan"
	recordplan "github.com/choysum-dev/choysum/internal/import/plan/record"
	"github.com/choysum-dev/choysum/internal/import/registry"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func init() {
	registry.RegisterWriter(importpkg.ProfileRecord, Writer{})
}

// Writer persists record import units through the ORM write path.
type Writer struct{}

// Write implements registry.Writer.
func (Writer) Write(ctx context.Context, txScope scope.Scope, units []plan.Unit) error {
	_ = ctx
	if txScope == nil || txScope.Session() == nil {
		return importpkg.Errorf(importpkg.CodeInvalidFormat, "scope is required for record writer")
	}
	for _, unit := range units {
		u, ok := unit.(recordplan.Unit)
		if !ok {
			return importpkg.Errorf(importpkg.CodeInvalidFormat, "unexpected unit type for record writer")
		}
		if err := writeUnit(txScope, u); err != nil {
			return err
		}
	}
	return nil
}

func writeUnit(txScope scope.Scope, unit recordplan.Unit) error {
	switch strings.TrimSpace(unit.Model) {
	case countryModelFull:
		return UpsertCountry(txScope, unit)
	default:
		return rowError(unit, "", importpkg.CodeModelNotFound, "unsupported record import model")
	}
}
