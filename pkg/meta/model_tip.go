// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package meta

import (
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

// ResolveMetaModelTip returns the newest live meta_model row for (application, name).
//
// IMD may register multiple rows with the same Name (base + extension modules).
// Callers that need a single ModelId must pick a deterministic tip:
// created_at DESC, id DESC. Soft-deleted rows are excluded by GORM's default scope.
//
// excludeIDs skips candidates (e.g. rows about to be soft-deleted during uninstall)
// so the tip reflects the post-uninstall surviving set.
func ResolveMetaModelTip(tx *gorm.DB, application, name string, excludeIDs ...string) (*Model, error) {
	if tx == nil {
		return nil, xfmt.Errorf("resolve meta model tip: nil db")
	}
	app := strings.TrimSpace(application)
	modelName := strings.TrimSpace(name)
	if app == "" || modelName == "" {
		return nil, xfmt.Errorf("resolve meta model tip: empty application or name")
	}

	q := tx.Where("application = ? AND name = ?", app, modelName)
	if cleaned := nonEmptyIDs(excludeIDs); len(cleaned) > 0 {
		q = q.Where("id NOT IN ?", cleaned)
	}

	model := &Model{}
	if err := q.Order("created_at DESC, id DESC").First(model).Error; err != nil {
		return nil, err
	}
	return model, nil
}

func nonEmptyIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if s := strings.TrimSpace(id); s != "" {
			out = append(out, s)
		}
	}
	return out
}
