// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"

	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func applyInitdata(ctx context.Context, runtimeScope scope.Scope, mod *meta.Module, caller importpkg.Caller, withDemo bool) error {
	if mod == nil {
		return nil
	}
	dataFiles, err := dataloader.ManifestDataFiles(mod)
	if err != nil {
		return err
	}
	demoFiles, err := dataloader.ManifestDemoFiles(mod)
	if err != nil {
		return err
	}
	spec := importpkg.Spec{
		Profile:     importpkg.ProfileInitdata,
		Caller:      caller,
		Policy:      importpkg.PolicyAtomic,
		Module:      mod.Name,
		Application: mod.ApplicationStr,
		Source: importpkg.Source{
			Format: "json",
			Path:   mod.Path,
		},
		Options: importpkg.Options{
			WithDemo:      withDemo,
			InitdataFiles: dataFiles,
			DemoFiles:     demoFiles,
		},
	}
	_, err = importpkg.Run(ctx, runtimeScope, spec)
	return err
}
