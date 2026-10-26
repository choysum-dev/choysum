// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"errors"
	"strconv"
	"strings"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/gorm"
)

// seedRowSnapshot is one MetaModelData row with its physical table resolved
// before declaration flush, so uninstall can hard-delete the business seed.
type seedRowSnapshot struct {
	Module    string
	Name      string
	ResID     string
	TableName string
}

// snapshotModuleSeedRows loads MetaModelData for module and resolves each row's
// model_table while effective meta_model rows are still available.
func snapshotModuleSeedRows(db *gorm.DB, moduleName string) ([]seedRowSnapshot, error) {
	if db == nil || strings.TrimSpace(moduleName) == "" {
		return nil, nil
	}
	if !db.Migrator().HasTable((&modmeta.ModelData{}).TableName()) {
		return nil, nil
	}

	var mappings []modmeta.ModelData
	if err := db.Unscoped().Where("module = ?", moduleName).Find(&mappings).Error; err != nil {
		return nil, xfmt.Errorf("error listing meta model data for seed cascade: %w", err)
	}
	if len(mappings) == 0 {
		return nil, nil
	}

	out := make([]seedRowSnapshot, 0, len(mappings))
	for i := range mappings {
		m := &mappings[i]
		resID := strings.TrimSpace(m.ResID)
		if resID == "" {
			continue
		}
		table, err := resolveSeedTableName(db, m)
		if err != nil {
			return nil, err
		}
		if !isSafeSQLIdent(table) {
			return nil, xfmt.Errorf(
				"unsafe model_table %q for xml_id %s.%s (res_id=%s)",
				table, m.Module, m.Name, resID,
			)
		}
		out = append(out, seedRowSnapshot{
			Module:    m.Module,
			Name:      m.Name,
			ResID:     resID,
			TableName: table,
		})
	}
	return out, nil
}

// resolveSeedTableName prefers ModelId → meta_model.model_table, then
// (application, model_name). Empty or unresolved tables fail closed.
func resolveSeedTableName(db *gorm.DB, m *modmeta.ModelData) (string, error) {
	if m == nil {
		return "", xfmt.Errorf("nil model data mapping")
	}
	xmlID := m.Module + "." + m.Name

	if id := strings.TrimSpace(m.ModelId); id != "" {
		var model meta.Model
		err := db.Unscoped().Select("model_table").Where("id = ?", id).Take(&model).Error
		if err == nil {
			if table := strings.TrimSpace(model.ModelTable); table != "" {
				return table, nil
			}
			return "", xfmt.Errorf("meta_model %s has empty model_table for xml_id %s", id, xmlID)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", xfmt.Errorf("error loading meta_model %s for xml_id %s: %w", id, xmlID, err)
		}
	}

	app := strings.TrimSpace(m.Application)
	name := strings.TrimSpace(m.ModelName)
	if app != "" && name != "" {
		var model meta.Model
		err := db.Unscoped().Select("model_table").
			Where("application = ? AND name = ?", app, name).
			Order("created_at DESC").
			Take(&model).Error
		if err == nil {
			if table := strings.TrimSpace(model.ModelTable); table != "" {
				return table, nil
			}
			return "", xfmt.Errorf("meta_model %s.%s has empty model_table for xml_id %s", app, name, xmlID)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", xfmt.Errorf("error looking up meta_model %s.%s for xml_id %s: %w", app, name, xmlID, err)
		}
	}

	return "", xfmt.Errorf("cannot resolve model_table for xml_id %s (model_id=%q application=%q model_name=%q)",
		xmlID, m.ModelId, m.Application, m.ModelName)
}

// deleteSeedRowsForSnapshots hard-deletes business seed rows in multi-pass order
// so FK dependencies between seeds can clear. Missing rows are OK; missing tables
// are skipped. Unresolvable FK after no progress aborts uninstall.
//
// On Postgres/MySQL, a failed DELETE aborts the whole transaction (SQLSTATE 25P02),
// so each attempt runs inside a SAVEPOINT that is rolled back on failure.
func deleteSeedRowsForSnapshots(db *gorm.DB, snapshots []seedRowSnapshot) error {
	if db == nil || len(snapshots) == 0 {
		return nil
	}

	useSP := dialectNeedsDeleteSavepoint(db)
	remaining := append([]seedRowSnapshot(nil), snapshots...)
	for len(remaining) > 0 {
		var next []seedRowSnapshot
		progressed := false
		var lastErr error

		for i, snap := range remaining {
			err := hardDeleteSeedRow(db, snap, useSP, i)
			if err == nil {
				progressed = true
				continue
			}
			if isMissingSQLTableError(err) {
				progressed = true
				continue
			}
			lastErr = err
			next = append(next, snap)
		}

		if len(next) == 0 {
			return nil
		}
		if !progressed {
			sample := next[0]
			if lastErr == nil {
				lastErr = xfmt.Errorf("unknown delete failure")
			}
			return xfmt.Errorf(
				"error cascade-deleting seed rows for module %s (stuck on %s.%s table=%s res_id=%s): %w",
				sample.Module, sample.Module, sample.Name, sample.TableName, sample.ResID, lastErr,
			)
		}
		remaining = next
	}
	return nil
}

func dialectNeedsDeleteSavepoint(db *gorm.DB) bool {
	if db == nil || db.Dialector == nil {
		return false
	}
	switch db.Dialector.Name() {
	case "postgres", "mysql":
		return true
	default:
		return false
	}
}

func hardDeleteSeedRow(db *gorm.DB, snap seedRowSnapshot, useSavepoint bool, attempt int) error {
	if !isSafeSQLIdent(snap.TableName) {
		return xfmt.Errorf("unsafe model_table %q", snap.TableName)
	}

	spName := ""
	if useSavepoint {
		// Unique per attempt: ResID alone can collide across xml_ids / multi-pass retries.
		spName = "sp_seed_" + strconv.Itoa(attempt)
		if err := db.SavePoint(spName).Error; err != nil {
			return xfmt.Errorf("savepoint %s: %w", spName, err)
		}
	}

	// Identifier already validated; res_id is bound as a parameter.
	res := db.Exec("DELETE FROM "+snap.TableName+" WHERE id = ?", snap.ResID)
	if res.Error != nil {
		if useSavepoint {
			if rbErr := db.RollbackTo(spName).Error; rbErr != nil {
				return xfmt.Errorf("delete %s id=%s: %w (rollback savepoint: %v)", snap.TableName, snap.ResID, res.Error, rbErr)
			}
			// Postgres keeps the savepoint after ROLLBACK TO; release so a later
			// pass can recreate the same name for this res_id.
			_ = releaseSeedDeleteSavepoint(db, spName)
		}
		return res.Error
	}
	if useSavepoint {
		if err := releaseSeedDeleteSavepoint(db, spName); err != nil {
			return xfmt.Errorf("release savepoint %s after delete %s id=%s: %w", spName, snap.TableName, snap.ResID, err)
		}
	}
	return nil
}

func releaseSeedDeleteSavepoint(db *gorm.DB, name string) error {
	if db == nil || !isSafeSQLIdent(name) {
		return xfmt.Errorf("invalid savepoint name")
	}
	switch db.Dialector.Name() {
	case "mysql", "postgres":
		return db.Exec("RELEASE SAVEPOINT " + name).Error
	case "sqlite":
		return db.Exec("RELEASE " + name).Error
	default:
		return nil
	}
}

// isSafeSQLIdent reports whether s is a plain unquoted SQL identifier:
// [A-Za-z_][A-Za-z0-9_]*.
func isSafeSQLIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
			continue
		}
		if r == '_' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			continue
		}
		return false
	}
	return true
}
