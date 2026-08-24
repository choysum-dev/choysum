// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package po

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/adapter"
	"github.com/choysum-dev/choysum/internal/import/plan"
	termplan "github.com/choysum-dev/choysum/internal/import/plan/term"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

const Format = "po"

func init() {
	adapter.RegisterPlanBuilder(Format, Builder{})
}

// Builder builds terminology plans from modules/<m>/i18n/*.po.
type Builder struct{}

// Build implements adapter.PlanBuilder.
func (Builder) Build(ctx context.Context, spec importpkg.Spec) (plan.Plan, error) {
	_ = ctx
	if spec.Profile != importpkg.ProfileTerminology {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "po adapter requires terminology profile")
	}
	modulePath := strings.TrimSpace(spec.Source.Path)
	if modulePath == "" {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "source path is required for terminology")
	}
	moduleName := strings.TrimSpace(spec.Module)
	if moduleName == "" {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "module is required for terminology")
	}
	application := strings.TrimSpace(spec.Application)
	if application == "" {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "application is required for terminology")
	}
	return BuildTermPlan(modulePath, "", application, moduleName)
}

// BuildTermPlan enumerates i18n/*.po under moduleRoot into terminology units.
// When lang is non-empty, only that language file is included.
func BuildTermPlan(moduleRoot, lang, application, moduleName string) (plan.Plan, error) {
	moduleRoot = strings.TrimSpace(moduleRoot)
	application = strings.TrimSpace(application)
	moduleName = strings.TrimSpace(moduleName)
	lang = strings.TrimSpace(lang)
	if moduleRoot == "" {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "module root is required for terminology")
	}

	i18nDir := filepath.Join(moduleRoot, "i18n")
	entries, err := os.ReadDir(i18nDir)
	if err != nil {
		if os.IsNotExist(err) {
			return plan.Plan{}, nil
		}
		return plan.Plan{}, importpkg.ErrorfWrap(importpkg.CodeInvalidFormat, "read i18n dir", err)
	}

	type poFile struct {
		lang string
		path string
	}
	files := make([]poFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".po") {
			continue
		}
		fileLang := strings.TrimSuffix(name, filepath.Ext(name))
		if fileLang == "" {
			continue
		}
		if lang != "" && fileLang != lang {
			continue
		}
		files = append(files, poFile{lang: fileLang, path: filepath.Join(i18nDir, name)})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].lang < files[j].lang })

	units := make([]plan.Unit, 0, len(files))
	for i, f := range files {
		units = append(units, termplan.Unit{
			Index:       i + 1,
			Application: application,
			ModuleName:  moduleName,
			ModulePath:  moduleRoot,
			Lang:        f.lang,
			PoPath:      f.path,
		})
	}
	return plan.Plan{Units: units}, nil
}
