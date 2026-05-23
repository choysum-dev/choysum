// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	dynamicstruct "github.com/Chise1/dynamic-struct"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
)

type modelMigrator struct {
	runtimeScope scope.Scope
	module       *meta.IrModule
	models       []*meta.IrModel
}

func newModelMigrator(runtimeScope scope.Scope, module *meta.IrModule, models []*meta.IrModel) *modelMigrator {
	return &modelMigrator{
		runtimeScope: runtimeScope,
		module:       module,
		models:       models,
	}
}

func normalizeModelIdentitySegment(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isStorageBlobCarrierModel(model *meta.IrModel) bool {
	if model == nil {
		return false
	}

	application := normalizeModelIdentitySegment(model.Application)
	name := normalizeModelIdentitySegment(model.Name)
	modelTable := normalizeModelIdentitySegment(model.ModelTable)

	if application == "document" &&
		(name == "attachmentobject" ||
			name == "uploadsession" ||
			name == "attachmentcontent" ||
			name == "attachmentuploadsession" ||
			name == "storedcontent") {
		return true
	}

	if modelTable == "document_attachment_object" ||
		modelTable == "document_upload_session" ||
		modelTable == "document_attachment_content" ||
		modelTable == "document_attachment_upload_session" ||
		modelTable == "document_stored_content" {
		return true
	}

	return false
}

// getFieldColumnMeta adapts the newer @Field decorator options.
func (m *modelMigrator) getFieldColumnMeta(field *meta.IrField, modelCtx ...*meta.IrModel) (map[string]interface{}, error) {
	var model *meta.IrModel
	if len(modelCtx) > 0 {
		model = modelCtx[0]
	}

	for _, deco := range field.Decorators {
		if deco.Name != "Field" || len(deco.Arguments) == 0 {
			continue
		}
		arg := deco.Arguments[0]
		if arg.Type != "ObjectLiteral" {
			continue
		}

		var opts map[string]interface{}
		if err := json.Unmarshal([]byte(arg.Value), &opts); err != nil {
			return nil, fmt.Errorf("error unmarshal @Field options: %w", err)
		}

		typeStr, _ := opts["type"].(string)
		if typeStr == "" {
			return nil, nil
		}

		// Binary/Image are attachment-backed in owner models and should not materialize
		// physical columns there. Document internal blob carrier models still keep columns.
		if (typeStr == "binary" || typeStr == "image") && model != nil && !isStorageBlobCarrierModel(model) {
			return nil, nil
		}

		// Virtual field: skip physical column creation when select is present.
		if _, hasSelect := opts["select"]; hasSelect {
			return nil, nil
		}

		// Non-column relations: skip OneToMany/ManyToMany.
		if typeStr == "OneToMany" || typeStr == "ManyToMany" {
			return nil, nil
		}

		meta := map[string]interface{}{
			"type": typeStr,
		}

		// Merge column options.
		if col, ok := opts["column"].(map[string]interface{}); ok {
			for k, v := range col {
				meta[k] = v
			}
		}

		// Ensure ManyToOne always maps to a concrete physical column definition.
		if typeStr == "ManyToOne" {
			// Force physical type to char.
			meta["type"] = "char"
			// Default length.
			if _, ok := meta["size"]; !ok {
				meta["size"] = 20
			}
			// notNull: honor explicit column setting, otherwise inherit field.NotNull.
			if _, ok := meta["notNull"]; !ok && field.NotNull {
				meta["notNull"] = true
			}
			// Index: add default index when index/uniqueIndex are both absent.
			if _, ok := meta["index"]; !ok {
				if _, ok2 := meta["uniqueIndex"]; !ok2 {
					meta["index"] = true
				}
			}
		}

		// Selection maps to varchar with default size 255 when omitted.
		if typeStr == "selection" {
			meta["type"] = "varchar"
			if _, ok := meta["size"]; !ok {
				meta["size"] = 255
			}
		}

		// ManyToOneRef: cross-service single Id reference, stored as char(20) with default index.
		if typeStr == "ManyToOneRef" {
			meta["type"] = "char"
			if _, ok := meta["size"]; !ok {
				meta["size"] = 20
			}
			if _, ok := meta["index"]; !ok {
				if _, ok2 := meta["uniqueIndex"]; !ok2 {
					meta["index"] = true
				}
			}
		}

		// ManyToManyRef: cross-service Id list reference, mapped to jsonobject.
		if typeStr == "ManyToManyRef" {
			meta["type"] = "jsonobject"
		}

		// Binary/Image are unified as blob physical columns in migration layer.
		if typeStr == "binary" || typeStr == "image" {
			meta["type"] = "blob"
		}

		// ManyToOne without explicit column still follows the same mapping path above.
		return meta, nil
	}
	return nil, nil
}

func (m *modelMigrator) getDialect() string {
	dialector := m.runtimeScope.Session().Dialector.Name()
	switch dialector {
	case "postgres", "postgresql":
		return "postgres"
	case "mysql", "mariadb":
		return "mysql"
	case "sqlite":
		return "sqlite"
	case "sqlserver":
		return "sqlserver"
	default:
		return "unknown"
	}
}

// addFieldToStruct adds a field to the dynamic struct.
func (m *modelMigrator) addFieldToStruct(builder *dynamicstruct.Builder, field *meta.IrField, meta map[string]interface{}) error {
	if meta == nil {
		// Skipped relation fields (e.g., OneToMany / ManyToMany).
		return nil
	}

	// Build gorm tags.
	gormTags := make([]string, 0)

	// Resolve column type.
	columnType, ok := meta["type"].(string)
	if !ok || columnType == "" {
		return nil
	}

	defaultValue := getDefaultValue(columnType)
	if defaultValue == nil {
		// Safety fallback: skip unknown/unsupported column types.
		return nil
	}

	dialectName := m.getDialect()
	typeTag := buildColumnTypeTag(dialectName, columnType, meta)
	if typeTag != "" {
		gormTags = append(gormTags, typeTag)
	}

	// Append standard tags.
	addStandardTags(&gormTags, meta)

	// Build final struct tags.
	structTag := fmt.Sprintf(`gorm:"%s" json:"%s"`, strings.Join(gormTags, ";"), strcase.ToSnake(field.Name))

	(*builder).AddField(field.Name, defaultValue, structTag)
	return nil
}

func (m *modelMigrator) migrateTableSchema(models []*meta.IrModel) error {
	for _, model := range models {
		if model.Readonly {
			continue
		}
		if model.AutoMigrate != nil && !*model.AutoMigrate {
			continue
		}
		tableName := model.ModelTable
		tableStruct := dynamicstruct.NewStruct()

		// Add fields
		for _, field := range model.Fields {
			meta, err := m.getFieldColumnMeta(field, model)
			if err != nil {
				return err
			}
			if err := m.addFieldToStruct(&tableStruct, field, meta); err != nil {
				return err
			}
		}

		// Migrate table
		instance := tableStruct.Build().New()
		if err := m.runtimeScope.Session().Table(tableName).AutoMigrate(instance); err != nil {
			return fmt.Errorf("migrate table %s: %w", tableName, err)
		}

		// Apply CHECK constraints (GORM won't create them from tags).
		if err := m.applyTableCheckConstraints(tableName, model); err != nil {
			return fmt.Errorf("migrate table %s check constraints: %w", tableName, err)
		}
	}
	return nil
}

func (m *modelMigrator) applyTableCheckConstraints(tableName string, model *meta.IrModel) error {
	dialect := m.getDialect()
	if dialect == "unknown" {
		return nil
	}

	for _, field := range model.Fields {
		colMeta, err := m.getFieldColumnMeta(field, model)
		if err != nil {
			return err
		}
		if colMeta == nil {
			continue
		}

		expr, ok := colMeta["checkConstraint"].(string)
		if !ok || strings.TrimSpace(expr) == "" {
			continue
		}

		// Use deterministic constraint names. Align prefix with GORM's default checker name (chk_).
		columnName := strcase.ToSnake(field.Name)
		constraintName := fmt.Sprintf("chk_%s_%s", tableName, columnName)
		legacyConstraintName := fmt.Sprintf("ck_%s_%s", tableName, columnName)
		if legacyConstraintName != constraintName {
			// Best-effort cleanup for earlier prefix.
			_ = dropCheckConstraintBestEffort(m.runtimeScope.Session().DB, dialect, tableName, legacyConstraintName)
		}

		if err := ensureCheckConstraint(m.runtimeScope.Session().DB, dialect, tableName, constraintName, expr); err != nil {
			return err
		}
	}

	return nil
}

func (m *modelMigrator) MigrateSchema() error {
	if err := m.migrateTableSchema(m.models); err != nil {
		return err
	}
	if err := ensureTaskJobExecutionTable(m.runtimeScope); err != nil {
		return err
	}
	return nil
}
