// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"strings"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"

)

// SupersedeInjectAppModels deletes generated-path declaration trees when Decide
// set SupersedeInject. Does not call modmeta.FlushEffective (EDS-opt-2).
func SupersedeInjectAppModels(sess *Session) error {
	if sess == nil {
		return nil
	}
	for _, spec := range sess.Registry().specsList() {
		if err := SupersedeOne(sess, spec.ModelName); err != nil {
			return err
		}
	}
	return nil
}

// SupersedeOne supersedes generated declarations for one Spec when its plan says so.
func SupersedeOne(sess *Session, modelName string) error {
	if sess == nil {
		return nil
	}
	spec, ok := sess.Registry().lookupPtr(modelName)
	if !ok {
		return nil
	}
	plan := sess.plans[spec.ModelName]
	if !plan.SupersedeInject {
		return nil
	}
	db := sess.ctx.DB
	mod := sess.ctx.Module
	if db == nil || mod == nil {
		return nil
	}
	app := strings.TrimSpace(mod.ApplicationStr)
	if app == "" {
		return nil
	}
	return supersedeGenerated(spec, db, app)
}

func supersedeGenerated(spec *Spec, db *gorm.DB, app string) error {
	absFalse := false
	existing, err := modmeta.ListDeclarations(db, modmeta.DeclarationQuery{
		Application: app,
		Name:        spec.ModelName,
		Abstract:    &absFalse,
	})
	if err != nil {
		return xfmt.Errorf("load %s rows for supersede: %w", spec.ModelName, err)
	}

	ids := make([]string, 0)
	for _, m := range existing {
		if m == nil || !isGeneratedPath(spec, m.Path) {
			continue
		}
		if !m.Id.Valid || strings.TrimSpace(m.Id.String) == "" {
			continue
		}
		ids = append(ids, m.Id.String)
	}
	if len(ids) == 0 {
		return nil
	}
	if err := modmeta.DeleteDeclarationTrees(db, ids); err != nil {
		return xfmt.Errorf("delete superseded generated %s rows: %w", spec.ModelName, err)
	}
	return nil
}
