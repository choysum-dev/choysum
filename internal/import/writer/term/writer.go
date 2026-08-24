// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term

import (
	"context"
	"fmt"
	"os"

	i18nimport "github.com/choysum-dev/choysum/internal/i18n/import"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/internal/import/plan"
	termplan "github.com/choysum-dev/choysum/internal/import/plan/term"
	"github.com/choysum-dev/choysum/internal/import/registry"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func init() {
	registry.RegisterWriter(importpkg.ProfileTerminology, Writer{})
}

// Writer delegates terminology units to UpsertPackagedTerms.
type Writer struct{}

// Write implements registry.Writer.
func (Writer) Write(ctx context.Context, txScope scope.Scope, units []plan.Unit) error {
	_ = ctx
	reg := store.RegistryFor(txScope)
	for _, unit := range units {
		u, ok := unit.(termplan.Unit)
		if !ok {
			return importpkg.Errorf(importpkg.CodeInvalidFormat, fmt.Sprintf("unexpected unit type %T for terminology writer", unit))
		}
		raw, err := os.ReadFile(u.PoPath)
		if err != nil {
			return importpkg.ErrorfWrap(importpkg.CodeInvalidFormat, fmt.Sprintf("read PO %s", u.PoPath), err)
		}
		if _, err := i18nimport.UpsertPackagedTerms(txScope, reg, u.Application, u.ModuleName, u.Lang, raw); err != nil {
			return err
		}
	}
	return nil
}
