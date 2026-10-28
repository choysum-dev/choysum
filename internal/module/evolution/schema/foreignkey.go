// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"encoding/json"
	"fmt"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/ettle/strcase"
)

// ForeignKeyInfo describes a foreign key relationship.
type ForeignKeyInfo struct {
	TableName       string // Source table name.
	ColumnName      string // Source column name.
	ReferTableName  string // Target table name.
	ReferColumnName string // Target column name.
	OnDelete        string // Delete rule.
	OnUpdate        string // Update rule.
}

// Relation type constants.
const (
	ManyToOne  = "ManyToOne"
	OneToMany  = "OneToMany"
	ManyToMany = "ManyToMany"
)

type foreignKeyMigrator struct {
	runtimeScope scope.Scope
	module       *meta.Module
	models       []*meta.Model
}

func newForeignKeyMigrator(runtimeScope scope.Scope, module *meta.Module, models []*meta.Model) ForeignKeyMigrator {
	return &foreignKeyMigrator{
		runtimeScope: runtimeScope,
		module:       module,
		models:       models,
	}
}

func (m *foreignKeyMigrator) MigrateForeignKeys() error {
	// 1. Collect foreign key relationships.
	fks, err := m.getForeignKeys()
	if err != nil {
		return err
	}

	// 2. Create the foreign keys.
	return m.createForeignKeys(fks)
}

func (m *foreignKeyMigrator) getForeignKeys() ([]ForeignKeyInfo, error) {
	var foreignKeys []ForeignKeyInfo

	manyToOneKeys, err := m.getManyToOneKeys()
	if err != nil {
		return nil, err
	}
	foreignKeys = append(foreignKeys, manyToOneKeys...)

	return foreignKeys, nil
}

// getManyToOneKeys reads relationships from the newer @Field decorator.
func (m *foreignKeyMigrator) getManyToOneKeys() ([]ForeignKeyInfo, error) {
	var foreignKeys []ForeignKeyInfo

	for _, model := range m.models {
		if model.Readonly {
			continue
		}
		if model.AutoMigrate != nil && !*model.AutoMigrate {
			continue
		}
		for _, field := range model.Fields {
			for _, decorator := range field.Decorators {
				if decorator.Name != "Field" || len(decorator.Arguments) == 0 {
					continue
				}
				arg := decorator.Arguments[0]
				if arg.Type != "ObjectLiteral" {
					continue
				}

				var opts map[string]interface{}
				if err := json.Unmarshal([]byte(arg.Value), &opts); err != nil {
					return nil, fmt.Errorf("parse @Field options failed: %v", err)
				}

				ftype, _ := opts["type"].(string)
				if ftype != ManyToOne {
					continue
				}

				skip, skipErr := shouldSkipManyToOneForeignKey(field, opts)
				if skipErr != nil {
					return nil, skipErr
				}
				if skip {
					continue
				}

				// Resolve target models from current module models first, then metadata storage.
				targetModelPath := field.ModuleSpecPath + ".ts"
				targetModel, err := m.resolveTargetModelByPath(targetModelPath)
				if err != nil {
					return nil, err
				}
				if targetModel == nil {
					return nil, fmt.Errorf("field %s target model %s not found", field.Name, targetModelPath)
				}

				onDelete := "NO ACTION"
				onUpdate := "NO ACTION"
				if rel, ok := opts["relation"].(map[string]interface{}); ok {
					if v, ok := rel["onDelete"].(string); ok && v != "" {
						onDelete = v
					}
					if v, ok := rel["onUpdate"].(string); ok && v != "" {
						onUpdate = v
					}
				}

				fk := ForeignKeyInfo{
					TableName:       model.ModelTable,
					ColumnName:      strcase.ToSnake(field.Name),
					ReferTableName:  targetModel.ModelTable,
					ReferColumnName: "id",
					OnDelete:        onDelete,
					OnUpdate:        onUpdate,
				}
				foreignKeys = append(foreignKeys, fk)
			}
		}
	}
	return foreignKeys, nil
}

// shouldSkipManyToOneForeignKey skips virtual / non-column ManyToOne fields.
// Prefer ResolvedSpec.Migration.ShouldCreateColumn (COMPUTE_STORE_FALSE, related,
// sqlCompute, …); also skip @Field({ select }) and bare @Compute({ store: false })
// when ResolvedSpec is absent.
func shouldSkipManyToOneForeignKey(field *meta.Field, opts map[string]interface{}) (bool, error) {
	if _, hasSelect := opts["select"]; hasSelect {
		return true, nil
	}
	if field == nil {
		return false, nil
	}
	resolved, err := field.GetResolvedSpec()
	if err != nil {
		return false, fmt.Errorf("error unmarshal field resolved spec for %s: %w", field.Name, err)
	}
	if resolved != nil {
		return !resolved.Migration.ShouldCreateColumn, nil
	}
	return fieldHasNonStoredComputeDecorator(field), nil
}

func fieldHasNonStoredComputeDecorator(field *meta.Field) bool {
	if field == nil {
		return false
	}
	for _, decorator := range field.Decorators {
		if decorator == nil || decorator.Name != "Compute" {
			continue
		}
		for _, arg := range decorator.Arguments {
			if arg == nil || arg.Type != "ObjectLiteral" || arg.Value == "" {
				continue
			}
			var computeOpts map[string]interface{}
			if err := json.Unmarshal([]byte(arg.Value), &computeOpts); err != nil {
				continue
			}
			if store, ok := computeOpts["store"].(bool); ok && !store {
				return true
			}
		}
	}
	return false
}

func (m *foreignKeyMigrator) resolveTargetModelByPath(targetModelPath string) (*meta.Model, error) {
	for _, model := range m.models {
		if model != nil && model.Path == targetModelPath {
			return model, nil
		}
	}

	session := m.runtimeScope.Session()

	// Prefer effective projection: tip path is canonical after dual-store recompute.
	var effective []meta.Model
	if err := session.Where("path = ?", targetModelPath).Order("id DESC").Find(&effective).Error; err != nil {
		return nil, err
	}
	if picked, err := pickModelWithUniqueTable(targetModelPath, effective); err != nil {
		return nil, err
	} else if picked != nil {
		return picked, nil
	}

	// Declaration fallback: Path is unique per ModuleId, so cross-module duplicates are possible.
	// Do not filter by the current module — FK targets often live in dependencies.
	decls, err := modmeta.ListDeclarations(session.DB, modmeta.DeclarationQuery{
		Path: targetModelPath,
	})
	if err != nil {
		return nil, err
	}
	if len(decls) == 0 {
		return nil, nil
	}
	models := make([]meta.Model, 0, len(decls))
	for _, c := range decls {
		if c != nil {
			models = append(models, *c)
		}
	}
	return pickModelWithUniqueTable(targetModelPath, models)
}

// pickModelWithUniqueTable returns the newest row when all candidates share ModelTable.
// Conflicting tables for one path would create an FK against the wrong table.
func pickModelWithUniqueTable(path string, models []meta.Model) (*meta.Model, error) {
	if len(models) == 0 {
		return nil, nil
	}
	table := models[0].ModelTable
	for i := 1; i < len(models); i++ {
		if models[i].ModelTable != table {
			return nil, fmt.Errorf(
				"ambiguous model path %q: ModelTable %q vs %q",
				path, table, models[i].ModelTable,
			)
		}
	}
	return &models[0], nil
}

// ForeignKeyBuilder builds SQL for foreign key creation.
type ForeignKeyBuilder interface {
	BuildForeignKeySQL(fk ForeignKeyInfo) string
	BuildDropForeignKeySQL(fk ForeignKeyInfo) string
}

// getForeignKeyBuilder returns the builder for the active database dialect.
func getForeignKeyBuilder(db *scope.Session) ForeignKeyBuilder {
	switch db.Dialector.Name() {
	case "postgres":
		return &PostgresForeignKeyBuilder{}
	case "mysql":
		return &MySQLForeignKeyBuilder{}
	default:
		return &PostgresForeignKeyBuilder{} // Default to PostgreSQL.
	}
}

func (m *foreignKeyMigrator) createForeignKeys(foreignKeys []ForeignKeyInfo) error {
	db := m.runtimeScope.Session()

	// SQLite does not support this foreign key creation path.
	if db.Dialector.Name() == "sqlite" {
		return nil
	}

	builder := getForeignKeyBuilder(db)
	for _, fk := range foreignKeys {
		// Try dropping any existing foreign key first.
		dropSQL := builder.BuildDropForeignKeySQL(fk)
		if err := db.Exec(dropSQL).Error; err != nil {
			// Ignore drop failures because the foreign key may not exist yet.
			// log.Printf("Warning: Failed to drop foreign key: %v", err)
		}

		// Create the new foreign key.
		createSQL := builder.BuildForeignKeySQL(fk)
		if err := db.Exec(createSQL).Error; err != nil {
			return fmt.Errorf(
				"create foreign key failed: table=%s column=%s refer=%s.%s: %w",
				fk.TableName,
				fk.ColumnName,
				fk.ReferTableName,
				fk.ReferColumnName,
				err,
			)
		}
	}

	return nil
}
