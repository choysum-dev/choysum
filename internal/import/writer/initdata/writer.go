// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initdata

import (
	"context"
	"errors"
	"fmt"

	"github.com/choysum-dev/choysum/internal/import/plan"
	initdataplan "github.com/choysum-dev/choysum/internal/import/plan/initdata"
	"github.com/choysum-dev/choysum/internal/import/registry"
	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func init() {
	registry.RegisterWriter(importpkg.ProfileInitdata, Writer{})
}

// Writer delegates initdata units to the data loader.
type Writer struct{}

// Write implements registry.Writer.
func (Writer) Write(ctx context.Context, txScope scope.Scope, units []plan.Unit) error {
	loader := dataloader.New(txScope)
	for _, unit := range units {
		u, ok := unit.(initdataplan.Unit)
		if !ok {
			return importpkg.Errorf(importpkg.CodeInvalidFormat, fmt.Sprintf("unexpected unit type %T for initdata writer", unit))
		}
		mod := &meta.Module{
			Name:           u.ModuleName,
			Path:           u.ModulePath,
			ApplicationStr: u.Application,
		}
		if err := loader.ApplyFiles(ctx, mod, u.Files); err != nil {
			return mapLoaderError(err)
		}
	}
	return nil
}

func mapLoaderError(err error) error {
	var loadErr *dataloader.LoadError
	if errors.As(err, &loadErr) {
		msg := LoadErrorToMessage(loadErr)
		return importpkg.ErrorfWrap(msg.Code, msg.Text, err)
	}
	return err
}
