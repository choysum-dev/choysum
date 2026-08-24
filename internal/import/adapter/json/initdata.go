// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package json

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/adapter"
	"github.com/choysum-dev/choysum/internal/import/plan"
	initdataplan "github.com/choysum-dev/choysum/internal/import/plan/initdata"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

const Format = "json"

func init() {
	adapter.RegisterPlanBuilder(Format, Builder{})
}

// Builder builds initdata plans from manifest file lists in Spec.Options.
type Builder struct{}

// Build implements adapter.PlanBuilder.
func (Builder) Build(ctx context.Context, spec importpkg.Spec) (plan.Plan, error) {
	_ = ctx
	if spec.Profile != importpkg.ProfileInitdata {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "json adapter requires initdata profile")
	}
	modulePath := strings.TrimSpace(spec.Source.Path)
	if modulePath == "" {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "source path is required for initdata")
	}
	moduleName := strings.TrimSpace(spec.Module)
	if moduleName == "" {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "module is required for initdata")
	}

	units := make([]plan.Unit, 0, 2)
	nextIndex := 1
	if files := cleanPaths(spec.Options.InitdataFiles); len(files) > 0 {
		units = append(units, initdataplan.Unit{
			Index:       nextIndex,
			ModuleName:  moduleName,
			ModulePath:  modulePath,
			Application: strings.TrimSpace(spec.Application),
			Files:       files,
		})
		nextIndex++
	}
	if spec.Options.WithDemo {
		if files := cleanPaths(spec.Options.DemoFiles); len(files) > 0 {
			units = append(units, initdataplan.Unit{
				Index:       nextIndex,
				ModuleName:  moduleName,
				ModulePath:  modulePath,
				Application: strings.TrimSpace(spec.Application),
				Files:       files,
			})
		}
	}
	return plan.Plan{Units: units}, nil
}

func cleanPaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(paths))
	for _, rel := range paths {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		out = append(out, rel)
	}
	return out
}
