// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/esbplugins"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/gorm"
)

func pathWithinModuleRoot(path, root string) bool {
	return esbplugins.PathWithinRoot(path, root)
}

func isGeneratedPath(spec *Spec, path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || spec == nil {
		return false
	}
	normalized := filepath.ToSlash(filepath.Clean(trimmed))
	rel := spec.GeneratedRelPath
	return strings.HasSuffix(normalized, "/"+rel) || normalized == rel
}

func generatedPath(spec *Spec, modulePath string) string {
	return filepath.ToSlash(filepath.Clean(filepath.Join(strings.TrimSpace(modulePath), spec.GeneratedRelPath)))
}

// canEnsureServiceEntry reports whether Ensure may proceed for spec.
// Virtual unit-test harnesses use a non-existent ModulesPath and are allowed.
// Real workspaces (ModulesPath exists) require Spec.BaseModelFile on disk so
// generated sources do not import missing core paths.
func canEnsureServiceEntry(sess *Session, spec *Spec, modulePath string) bool {
	if sess == nil || spec == nil {
		return false
	}
	rel := strings.TrimSpace(spec.BaseModelFile)
	if rel == "" {
		return true
	}
	modulesPath := strings.TrimSpace(sess.ctx.ModulesPath)
	if modulesPath == "" {
		modulesPath = filepath.Dir(strings.TrimSpace(modulePath))
	}
	if modulesPath == "" || modulesPath == "." {
		return false
	}
	if _, err := os.Stat(modulesPath); err != nil {
		return true
	}
	_, err := os.Stat(filepath.Join(modulesPath, filepath.FromSlash(rel)))
	return err == nil
}

func modelsIn(spec *Spec, results []*parser.ParserResult, modulePath string) []*meta.Model {
	out := make([]*meta.Model, 0)
	for _, result := range results {
		if result == nil || result.Model == nil || spec == nil {
			continue
		}
		m := result.Model
		if m.Name != spec.ModelName || m.Abstract {
			continue
		}
		if modulePath != "" && !pathWithinModuleRoot(result.Path, modulePath) && !pathWithinModuleRoot(m.Path, modulePath) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func handwrittenModels(spec *Spec, models []*meta.Model) []*meta.Model {
	out := make([]*meta.Model, 0, len(models))
	for _, m := range models {
		if m == nil || isGeneratedPath(spec, m.Path) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func generatedModels(spec *Spec, models []*meta.Model) []*meta.Model {
	out := make([]*meta.Model, 0, len(models))
	for _, m := range models {
		if m == nil || !isGeneratedPath(spec, m.Path) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func sameModule(existing []*meta.Model, mod *meta.Module) bool {
	if mod == nil {
		return false
	}
	modPath := strings.TrimSpace(mod.Path)
	modID := ""
	if mod.Id.Valid {
		modID = strings.TrimSpace(mod.Id.String)
	}
	for _, m := range existing {
		if m == nil {
			continue
		}
		if modID != "" && m.ModuleId.Valid && strings.TrimSpace(m.ModuleId.String) == modID {
			return true
		}
		if modPath != "" && pathWithinModuleRoot(m.Path, modPath) {
			return true
		}
	}
	return false
}

func dbLoadModels(spec *Spec, db *gorm.DB, app string) ([]*meta.Model, error) {
	app = strings.TrimSpace(app)
	if app == "" || db == nil || spec == nil {
		return nil, nil
	}
	absFalse := false
	return meta.ListDeclarations(db, meta.DeclarationQuery{
		Application: app,
		Name:        spec.ModelName,
		Abstract:    &absFalse,
	})
}

// mergeUniqueStrings copies base and extras into a fresh slice, dropping empties and duplicates.
func mergeUniqueStrings(base []string, extras ...[]string) []string {
	seen := make(map[string]struct{}, len(base))
	out := make([]string, 0, len(base))
	appendOne := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, s := range base {
		appendOne(s)
	}
	for _, chunk := range extras {
		for _, s := range chunk {
			appendOne(s)
		}
	}
	return out
}
