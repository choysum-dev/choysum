// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LoadSnapshots returns snapshot rows keyed by model_table.
func LoadSnapshots(db *gorm.DB, tables []string) (map[string]modmeta.SchemaSnapshot, error) {
	out := make(map[string]modmeta.SchemaSnapshot)
	if db == nil || len(tables) == 0 {
		return out, nil
	}
	names := make([]string, 0, len(tables))
	for _, t := range tables {
		t = strings.TrimSpace(t)
		if t != "" {
			names = append(names, t)
		}
	}
	if len(names) == 0 {
		return out, nil
	}
	var rows []modmeta.SchemaSnapshot
	if err := db.Where("model_table IN ?", names).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load schema snapshots: %w", err)
	}
	for _, row := range rows {
		out[row.ModelTable] = row
	}
	return out, nil
}

// SaveSnapshots upserts desired column specs per model_table after a successful apply.
func SaveSnapshots(db *gorm.DB, desired DesiredSchema, models []*meta.Model, module *meta.Module) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := db.AutoMigrate(&modmeta.SchemaSnapshot{}); err != nil {
		return fmt.Errorf("ensure meta_schema_snapshot: %w", err)
	}
	metaByTable := map[string]*meta.Model{}
	for _, m := range models {
		if m == nil {
			continue
		}
		table := strings.TrimSpace(m.ModelTable)
		if table == "" {
			continue
		}
		metaByTable[table] = m
	}
	moduleName := ""
	moduleVersion := ""
	if module != nil {
		moduleName = strings.TrimSpace(module.Name)
		moduleVersion = strings.TrimSpace(module.Version)
	}
	now := time.Now().UTC()
	for table, cols := range desired.Tables {
		payload, err := json.Marshal(cols)
		if err != nil {
			return fmt.Errorf("marshal desired for %s: %w", table, err)
		}
		row := modmeta.SchemaSnapshot{
			ModelTable:      table,
			DesiredJSON:     datatypes.JSON(payload),
			UpdatedByModule: moduleName,
			ModuleVersion:   moduleVersion,
			UpdatedAt:       now,
		}
		if m := metaByTable[table]; m != nil {
			row.Application = strings.TrimSpace(m.Application)
			row.ModelName = strings.TrimSpace(m.Name)
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "model_table"}},
			DoUpdates: clause.AssignmentColumns([]string{"application", "model_name", "desired_json", "updated_by_module", "module_version", "updated_at"}),
		}).Create(&row).Error; err != nil {
			return fmt.Errorf("save schema snapshot %s: %w", table, err)
		}
	}
	return nil
}
