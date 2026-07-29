// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	dynamicstruct "github.com/Chise1/dynamic-struct"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type schemaTestScope struct {
	ctx     context.Context
	cfg     *config.Config
	logger  *slog.Logger
	session *scope.Session
}

func (e *schemaTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }

func (e *schemaTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}

func (e *schemaTestScope) Session() *scope.Session { return e.session }

func (e *schemaTestScope) WithContext(ctx context.Context) scope.Scope {
	if ctx == nil {
		ctx = e.ctx
	}
	return &schemaTestScope{ctx: ctx, cfg: e.cfg, logger: e.logger, session: e.session}
}

func (e *schemaTestScope) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}

func (e *schemaTestScope) Logger() *slog.Logger { return e.logger }

func (e *schemaTestScope) Config() *config.Config { return e.cfg }

func (e *schemaTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newSchemaTestScope(t *testing.T) *schemaTestScope {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "schema.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &schemaTestScope{
		ctx:     context.Background(),
		cfg:     &config.Config{Db: &config.DbConfig{Dialect: "sqlite"}, Server: config.NewDefaultServerConfig(), Log: config.NewDefaultLogConfig()},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		session: &scope.Session{DB: db},
	}
}

func migrateSchemaMetaTables(t *testing.T, session *scope.Session) {
	t.Helper()
	if err := session.AutoMigrate(&meta.IrModel{}, &meta.IrField{}, &meta.IrDecorator{}, &meta.IrArgument{}, &meta.IrModule{}); err != nil {
		t.Fatalf("migrate schema meta tables: %v", err)
	}
}

func newFieldWithOptions(t *testing.T, name string, options string) *meta.IrField {
	t.Helper()
	field := &meta.IrField{
		Name: name,
		Decorators: []*meta.IrDecorator{{
			Name:      "Field",
			Arguments: []*meta.IrArgument{{Type: "ObjectLiteral", Value: options}},
		}},
	}
	attachResolvedSpecForTestField(field, options)
	return field
}

func newRelationField(name string, moduleSpecPath string, options string) *meta.IrField {
	field := &meta.IrField{
		Name:           name,
		ModuleSpecPath: moduleSpecPath,
		Decorators: []*meta.IrDecorator{{
			Name:      "Field",
			Arguments: []*meta.IrArgument{{Type: "ObjectLiteral", Value: options}},
		}},
	}
	field.ModuleSpecPath = moduleSpecPath
	attachResolvedSpecForTestField(field, options)
	return field
}

func attachResolvedSpecForTestField(field *meta.IrField, options string) {
	if field == nil {
		return
	}

	var opts map[string]any
	if err := json.Unmarshal([]byte(options), &opts); err != nil {
		// Keep malformed payloads to exercise parser error paths in tests.
		field.ResolvedSpec = options
		return
	}

	typeStr := strings.TrimSpace(asString(opts["type"]))
	if typeStr == "" {
		return
	}

	spec := &meta.IrFieldResolvedSpec{
		FieldName: field.Name,
		Structural: meta.IrFieldStructuralSpec{
			Name:      field.Name,
			FieldType: typeStr,
		},
		Migration: meta.IrFieldMigrationDecision{
			StorageKind:        "physical",
			ShouldCreateColumn: true,
			ResolvedColumnType: typeStr,
			ReasonCode:         "FIELD_DEFAULT",
		},
	}

	if v, ok := opts["required"].(bool); ok {
		if spec.Structural.StorageHints == nil {
			spec.Structural.StorageHints = &meta.IrFieldStructuralStorageHints{}
		}
		spec.Structural.StorageHints.Required = boolPtrValue(v)
	}
	if v, ok := opts["indexed"].(bool); ok {
		if spec.Structural.StorageHints == nil {
			spec.Structural.StorageHints = &meta.IrFieldStructuralStorageHints{}
		}
		spec.Structural.StorageHints.Indexed = boolPtrValue(v)
	}
	if v, ok := asIntValue(opts["size"]); ok {
		if spec.Structural.StorageHints == nil {
			spec.Structural.StorageHints = &meta.IrFieldStructuralStorageHints{}
		}
		spec.Structural.StorageHints.Size = intPtrValue(v)
	}
	if v, ok := asIntValue(opts["precision"]); ok {
		if spec.Structural.StorageHints == nil {
			spec.Structural.StorageHints = &meta.IrFieldStructuralStorageHints{}
		}
		spec.Structural.StorageHints.Precision = intPtrValue(v)
	}
	if v, ok := asIntValue(opts["scale"]); ok {
		if spec.Structural.StorageHints == nil {
			spec.Structural.StorageHints = &meta.IrFieldStructuralStorageHints{}
		}
		spec.Structural.StorageHints.Scale = intPtrValue(v)
	}
	if v, ok := opts["unique"].(bool); ok {
		if spec.Structural.StorageHints == nil {
			spec.Structural.StorageHints = &meta.IrFieldStructuralStorageHints{}
		}
		spec.Structural.StorageHints.Unique = boolPtrValue(v)
	}
	if v, ok := opts["uniqueIndex"]; ok {
		if spec.Structural.StorageHints == nil {
			spec.Structural.StorageHints = &meta.IrFieldStructuralStorageHints{}
		}
		switch val := v.(type) {
		case bool:
			spec.Structural.StorageHints.UniqueIndexEnabled = boolPtrValue(val)
		case string:
			trimmed := strings.TrimSpace(val)
			spec.Structural.StorageHints.UniqueIndex = &trimmed
		}
	}

	if col, ok := opts["column"].(map[string]any); ok {
		hints := spec.Structural.StorageHints
		if hints == nil {
			hints = &meta.IrFieldStructuralStorageHints{}
			spec.Structural.StorageHints = hints
		}
		if v, ok := col["notNull"].(bool); ok {
			hints.Required = boolPtrValue(v)
		}
		if v, ok := col["size"]; ok {
			if iv, ok := asIntValue(v); ok {
				hints.Size = intPtrValue(iv)
			}
		}
		if v, ok := col["precision"]; ok {
			if iv, ok := asIntValue(v); ok {
				hints.Precision = intPtrValue(iv)
			}
		}
		if v, ok := col["scale"]; ok {
			if iv, ok := asIntValue(v); ok {
				hints.Scale = intPtrValue(iv)
			}
		}
		if idx, ok := col["index"]; ok {
			switch v := idx.(type) {
			case bool:
				hints.Indexed = boolPtrValue(v)
			case string:
				hints.Indexed = boolPtrValue(strings.TrimSpace(v) != "")
			}
		}
		if uniq, ok := col["unique"].(bool); ok && uniq {
			hints.Indexed = boolPtrValue(true)
		}
		if uniqIdx, ok := col["uniqueIndex"]; ok {
			switch v := uniqIdx.(type) {
			case bool:
				if v {
					hints.Indexed = boolPtrValue(true)
				}
			case string:
				if strings.TrimSpace(v) != "" {
					hints.Indexed = boolPtrValue(true)
				}
			}
		}
		if hints.Required != nil || hints.Indexed != nil || hints.Size != nil || hints.Precision != nil || hints.Scale != nil {
			spec.Structural.StorageHints = hints
		}
		if cc := strings.TrimSpace(asString(col["checkConstraint"])); cc != "" {
			spec.Structural.CheckConstraint = cc
		}
	}

	if relation, ok := opts["relation"].(map[string]any); ok {
		spec.Structural.Relation = relation
	}
	if related, ok := opts["related"].(map[string]any); ok {
		relatedSpec := &meta.IrFieldRelatedSpec{Path: strings.TrimSpace(asString(related["path"]))}
		if v, ok := related["store"].(bool); ok {
			relatedSpec.Store = v
		}
		if deps := related["deps"]; deps != nil {
			if arr, ok := deps.([]any); ok {
				for _, dep := range arr {
					s := strings.TrimSpace(asString(dep))
					if s != "" {
						relatedSpec.Deps = append(relatedSpec.Deps, s)
					}
				}
			}
		}
		if relatedSpec.Path != "" {
			spec.Structural.Related = relatedSpec
		}
	}

	if selection, ok := opts["selection"].([]any); ok {
		for _, item := range selection {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			value := strings.TrimSpace(asString(entry["value"]))
			label := strings.TrimSpace(asString(entry["label"]))
			if value == "" || label == "" {
				continue
			}
			spec.Structural.Selection = append(spec.Structural.Selection, meta.IrFieldSelectionItem{Value: value, Label: label})
		}
	}

	if _, hasSelect := opts["select"]; hasSelect {
		spec.Migration.StorageKind = "virtualSql"
		spec.Migration.ShouldCreateColumn = false
		spec.Migration.ResolvedColumnType = ""
		spec.Migration.ReasonCode = "SQL_COMPUTE"
	}

	if typeStr == "OneToMany" || typeStr == "ManyToMany" {
		spec.Migration.StorageKind = "relationOnly"
		spec.Migration.ShouldCreateColumn = false
		spec.Migration.ResolvedColumnType = ""
		spec.Migration.ReasonCode = "RELATION_NON_COLUMN"
	}

	switch typeStr {
	case "selection":
		spec.Migration.ResolvedColumnType = "varchar"
	case "ManyToOne", "ManyToOneRef":
		spec.Migration.ResolvedColumnType = "char"
	case "ManyToManyRef":
		spec.Migration.ResolvedColumnType = "jsonobject"
	case "binary", "image":
		spec.Migration.ResolvedColumnType = "blob"
	}

	if v, ok := opts["companyDependent"].(bool); ok && v {
		trueVal := true
		spec.Structural.CompanyDependent = &trueVal
		spec.Structural.ColumnType = "jsonobject"
		spec.Migration.ResolvedColumnType = "jsonobject"
		spec.Migration.ReasonCode = "COMPANY_DEPENDENT_MAP"
	}

	_ = field.SetResolvedSpec(spec)
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

func asIntValue(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}

func boolPtrValue(value bool) *bool {
	v := value
	return &v
}

func intPtrValue(value int) *int {
	v := value
	return &v
}

func dynamicStructBuilder() dynamicstruct.Builder {
	return dynamicstruct.NewStruct()
}
