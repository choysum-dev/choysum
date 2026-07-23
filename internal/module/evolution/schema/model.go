// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
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

// getResolvedFieldColumnMeta builds migration column metadata from parser-resolved field specs.
func (m *modelMigrator) getResolvedFieldColumnMeta(field *meta.IrField, modelCtx ...*meta.IrModel) (map[string]interface{}, error) {
	var model *meta.IrModel
	if len(modelCtx) > 0 {
		model = modelCtx[0]
	}
	if field == nil {
		return nil, nil
	}

	resolved, err := field.GetResolvedSpec()
	if err != nil {
		return nil, fmt.Errorf("error unmarshal field resolved spec: %w", err)
	}
	if resolved == nil {
		return nil, nil
	}

	if !resolved.Migration.ShouldCreateColumn {
		return nil, nil
	}

	typeStr := strings.TrimSpace(resolved.Structural.FieldType)
	if typeStr == "" {
		return nil, nil
	}

	// Binary/Image are attachment-backed in owner models and should not materialize
	// physical columns there. Document internal blob carrier models still keep columns.
	if (typeStr == "binary" || typeStr == "image") && model != nil && !isStorageBlobCarrierModel(model) {
		return nil, nil
	}

	columnType := strings.TrimSpace(resolved.Migration.ResolvedColumnType)
	if columnType == "" {
		columnType = typeStr
	}

	metaMap := map[string]interface{}{
		"type": columnType,
	}

	if hints := resolved.Structural.StorageHints; hints != nil {
		if hints.Required != nil {
			metaMap["notNull"] = *hints.Required
		}
		if hints.Index != nil && strings.TrimSpace(*hints.Index) != "" {
			metaMap["index"] = strings.TrimSpace(*hints.Index)
		} else if hints.Indexed != nil {
			metaMap["index"] = *hints.Indexed
		}
		if hints.Size != nil {
			metaMap["size"] = *hints.Size
		}
		if hints.Precision != nil {
			metaMap["precision"] = *hints.Precision
		}
		if hints.Scale != nil {
			metaMap["scale"] = *hints.Scale
		}
		if hints.PrimaryKey != nil {
			metaMap["primaryKey"] = *hints.PrimaryKey
		}
		if hints.Unique != nil {
			metaMap["unique"] = *hints.Unique
		}
		if hints.UniqueIndexEnabled != nil {
			metaMap["uniqueIndex"] = *hints.UniqueIndexEnabled
		}
		if hints.UniqueIndex != nil && strings.TrimSpace(*hints.UniqueIndex) != "" {
			metaMap["uniqueIndex"] = strings.TrimSpace(*hints.UniqueIndex)
		}
	}

	// Data i18n: JSON/JSONB lang map; size is per-lang limit only; never unique btree (D14·D15).
	// Trigram GIN is applied separately (ensureTranslatedTrigramIndex); never emit GORM btree tags.
	if resolved.Structural.Translate != nil && *resolved.Structural.Translate {
		metaMap["type"] = "jsonobject"
		delete(metaMap, "size")
		delete(metaMap, "unique")
		delete(metaMap, "uniqueIndex")
		delete(metaMap, "index")
		if hints := resolved.Structural.StorageHints; hints != nil && hints.Index != nil {
			if strings.EqualFold(strings.TrimSpace(*hints.Index), translatedTrigramIndexKind) {
				metaMap["trigram"] = true
			}
		}
	}

	// Keep compatibility defaults for reference and relation-like scalar carriers.
	if typeStr == "ManyToOne" || typeStr == "ManyToOneRef" {
		metaMap["type"] = "char"
		if _, ok := metaMap["size"]; !ok {
			metaMap["size"] = 20
		}
		if _, ok := metaMap["notNull"]; !ok && field.NotNull {
			metaMap["notNull"] = true
		}
		if _, ok := metaMap["index"]; !ok {
			hasUniqueIndex := false
			if val, ok := metaMap["uniqueIndex"]; ok {
				if b, ok := val.(bool); ok {
					hasUniqueIndex = b
				} else if s, ok := val.(string); ok {
					hasUniqueIndex = strings.TrimSpace(s) != ""
				}
			}
			hasUnique := false
			if val, ok := metaMap["unique"]; ok {
				if b, ok := val.(bool); ok {
					hasUnique = b
				}
			}
			if !hasUniqueIndex && !hasUnique {
				metaMap["index"] = true
			}
		}
	}
	if typeStr == "ManyToManyRef" {
		metaMap["type"] = "jsonobject"
	}
	if typeStr == "selection" {
		metaMap["type"] = "varchar"
		if _, ok := metaMap["size"]; !ok {
			metaMap["size"] = 255
		}
	}
	if typeStr == "binary" || typeStr == "image" {
		metaMap["type"] = "blob"
	}

	if cc := strings.TrimSpace(resolved.Structural.CheckConstraint); cc != "" {
		metaMap["checkConstraint"] = cc
	}

	return metaMap, nil
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
			meta, err := m.getResolvedFieldColumnMeta(field, model)
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

		// Optional PG full-language trigram GIN for translate fields (skipped without pg_trgm).
		if err := m.applyTableTranslatedTrigramIndexes(tableName, model); err != nil {
			return fmt.Errorf("migrate table %s translated trigram indexes: %w", tableName, err)
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
		colMeta, err := m.getResolvedFieldColumnMeta(field, model)
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
