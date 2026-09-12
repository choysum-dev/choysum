// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

var (
	applyTableTranslatedTrigramIndexesFn = func(m *modelMigrator, table string, model *meta.Model) error {
		return m.applyTableTranslatedTrigramIndexes(table, model)
	}
	applyTableTranslatedL2IndexesFn = func(m *modelMigrator, table string, model *meta.Model) error {
		return m.applyTableTranslatedL2Indexes(table, model)
	}
)

type modelMigrator struct {
	runtimeScope scope.Scope
	module       *meta.Module
	models       []*meta.Model
}

func newModelMigrator(runtimeScope scope.Scope, module *meta.Module, models []*meta.Model) *modelMigrator {
	return &modelMigrator{
		runtimeScope: runtimeScope,
		module:       module,
		models:       models,
	}
}

func normalizeModelIdentitySegment(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isStorageBlobCarrierModel(model *meta.Model) bool {
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

func (m *modelMigrator) getDialect() string {
	if m.runtimeScope == nil || m.runtimeScope.Session() == nil || m.runtimeScope.Session().Dialector == nil {
		return "unknown"
	}
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

func (m *modelMigrator) buildSchemaPlan() (DesiredSchema, SchemaPlan, error) {
	desired, err := buildDesired(m.models)
	if err != nil {
		return DesiredSchema{}, SchemaPlan{}, err
	}
	tables := desiredTableNames(desired)
	live, err := inspectTables(m.runtimeScope.Session().DB, tables)
	if err != nil {
		return DesiredSchema{}, SchemaPlan{}, fmt.Errorf("inspect tables: %w", err)
	}
	dialect := m.getDialect()
	moduleName := ""
	if m.module != nil {
		moduleName = m.module.Name
	}
	plan := buildPlan(moduleName, desired, live, dialect)
	return desired, plan, nil
}

func (m *modelMigrator) PlanSchema() (SchemaPlan, error) {
	_, plan, err := m.buildSchemaPlan()
	if err != nil {
		return SchemaPlan{}, err
	}
	if err := ValidatePlan(plan); err != nil {
		return plan, err
	}
	return plan, nil
}

func (m *modelMigrator) MigrateSchema() error {
	desired, plan, err := m.buildSchemaPlan()
	if err != nil {
		return err
	}
	m.logPlan(plan)
	if err := ValidatePlan(plan); err != nil {
		return err
	}
	dialect := m.getDialect()
	if err := applyPlan(m.runtimeScope, dialect, plan); err != nil {
		return err
	}

	// Trigram / L2 stay outside ordinary index plan (dialect-specific ensure).
	for _, model := range m.models {
		if model == nil || model.Readonly {
			continue
		}
		if model.AutoMigrate != nil && !*model.AutoMigrate {
			continue
		}
		tableName := strings.TrimSpace(model.ModelTable)
		if tableName == "" {
			continue
		}
		if err := applyTableTranslatedTrigramIndexesFn(m, tableName, model); err != nil {
			return fmt.Errorf("migrate table %s translated trigram indexes: %w", tableName, err)
		}
		if err := applyTableTranslatedL2IndexesFn(m, tableName, model); err != nil {
			return fmt.Errorf("migrate table %s translated L2 indexes: %w", tableName, err)
		}
	}

	if err := ensureTaskJobExecutionTable(m.runtimeScope); err != nil {
		return err
	}
	if err := SaveSnapshots(m.runtimeScope.Session().DB, desired, m.models, m.module); err != nil {
		return fmt.Errorf("save schema snapshots: %w", err)
	}
	return nil
}

func (m *modelMigrator) logPlan(plan SchemaPlan) {
	if m.runtimeScope == nil || m.runtimeScope.Logger() == nil {
		return
	}
	auto, guarded, manual := 0, 0, 0
	for _, op := range plan.Ops {
		switch op.Safety {
		case SafetyAuto:
			auto++
		case SafetyGuarded:
			guarded++
		case SafetyManual:
			manual++
		}
	}
	m.runtimeScope.Logger().Info("schema plan",
		"module", plan.Module,
		"ops", len(plan.Ops),
		"auto", auto,
		"guarded", guarded,
		"manual", manual,
		"leftover", len(plan.Leftover),
	)
}

func (m *modelMigrator) applyTableCheckConstraints(tableName string, model *meta.Model) error {
	dialect := m.getDialect()
	if dialect == "unknown" {
		return nil
	}
	for _, field := range model.Fields {
		col, err := columnSpecFromField(field, model)
		if err != nil {
			return err
		}
		if col == nil || strings.TrimSpace(col.CheckExpr) == "" {
			continue
		}
		columnName := col.Name
		if columnName == "" {
			columnName = strings.ToLower(field.Name)
		}
		constraintName := fmt.Sprintf("chk_%s_%s", tableName, columnName)
		legacyConstraintName := fmt.Sprintf("ck_%s_%s", tableName, columnName)
		if legacyConstraintName != constraintName {
			_ = dropCheckConstraintBestEffort(m.runtimeScope.Session().DB, dialect, tableName, legacyConstraintName)
		}
		if err := ensureCheckConstraint(m.runtimeScope.Session().DB, dialect, tableName, constraintName, col.CheckExpr); err != nil {
			return err
		}
	}
	return nil
}
