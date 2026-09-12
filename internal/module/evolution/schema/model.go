// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"encoding/json"
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
	loadSnapshotsFn = LoadSnapshots
)

type modelMigrator struct {
	runtimeScope scope.Scope
	module       *meta.Module
	models       []*meta.Model
	intents      IntentBag
	toVersion    string
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
	plan, err := buildPlan(moduleName, desired, live, dialect)
	if err != nil {
		return DesiredSchema{}, SchemaPlan{}, err
	}
	plan = filterIntentCoveredLeftovers(plan, m.intents)
	if err := markLeftoverOwnership(&plan, m.runtimeScope); err != nil {
		return DesiredSchema{}, SchemaPlan{}, err
	}
	return desired, plan, nil
}

// filterIntentCoveredLeftovers drops leftover columns already covered by a drop Intent
// so Validate/logging do not imply a second drop after script helpers ran.
func filterIntentCoveredLeftovers(plan SchemaPlan, intents IntentBag) SchemaPlan {
	if len(plan.Leftover) == 0 || intents == nil {
		return plan
	}
	filtered := make([]Leftover, 0, len(plan.Leftover))
	for _, left := range plan.Leftover {
		if left.Kind == LeftoverColumn && IntentSatisfies(PlanOp{
			Kind:   OpKind(IntentDropColumn),
			Safety: SafetyManual,
			Table:  left.Table,
			Detail: "drop column " + left.Name,
			Column: &ColumnSpec{Name: left.Name},
		}, intents) {
			continue
		}
		filtered = append(filtered, left)
	}
	plan.Leftover = filtered
	return plan
}

// markLeftoverOwnership sets ChoysumOwned on leftovers (idx_ indexes; columns in schema snapshots).
func markLeftoverOwnership(plan *SchemaPlan, runtimeScope scope.Scope) error {
	if plan == nil || len(plan.Leftover) == 0 {
		return nil
	}
	for i := range plan.Leftover {
		left := &plan.Leftover[i]
		switch left.Kind {
		case LeftoverIndex:
			left.ChoysumOwned = strings.HasPrefix(strings.ToLower(strings.TrimSpace(left.Name)), "idx_")
		}
	}
	tables := make([]string, 0, len(plan.Leftover))
	seen := map[string]struct{}{}
	for _, left := range plan.Leftover {
		if left.Kind != LeftoverColumn {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(left.Table))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tables = append(tables, strings.TrimSpace(left.Table))
	}
	if len(tables) == 0 || runtimeScope == nil || runtimeScope.Session() == nil {
		return nil
	}
	snaps, err := loadSnapshotsFn(runtimeScope.Session().DB, tables)
	if err != nil {
		return fmt.Errorf("mark leftover ownership: %w", err)
	}
	ownedCols := map[string]map[string]struct{}{} // table → column names from snapshot
	for table, snap := range snaps {
		if len(snap.DesiredJSON) == 0 {
			continue
		}
		var cols []ColumnSpec
		if err := json.Unmarshal(snap.DesiredJSON, &cols); err != nil {
			return fmt.Errorf("mark leftover ownership decode %s: %w", table, err)
		}
		names := map[string]struct{}{}
		for _, col := range cols {
			if n := strings.TrimSpace(col.Name); n != "" {
				names[strings.ToLower(n)] = struct{}{}
			}
			if rf := strings.TrimSpace(col.RenameFrom); rf != "" {
				names[strings.ToLower(rf)] = struct{}{}
			}
		}
		ownedCols[strings.ToLower(strings.TrimSpace(table))] = names
	}
	for i := range plan.Leftover {
		left := &plan.Leftover[i]
		if left.Kind != LeftoverColumn {
			continue
		}
		names := ownedCols[strings.ToLower(strings.TrimSpace(left.Table))]
		if names == nil {
			continue
		}
		_, left.ChoysumOwned = names[strings.ToLower(strings.TrimSpace(left.Name))]
	}
	return nil
}

func (m *modelMigrator) PlanSchema() (SchemaPlan, error) {
	_, plan, err := m.buildSchemaPlan()
	if err != nil {
		return SchemaPlan{}, err
	}
	if err := ValidatePlan(plan, m.intents); err != nil {
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
	m.warnDropAfterLeftovers(plan)
	if err := ValidatePlan(plan, m.intents); err != nil {
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

// warnDropAfterLeftovers logs when leftover columns were marked dropAfter for this toVersion
// but no matching drop Intent was registered (does not fail the upgrade).
func (m *modelMigrator) warnDropAfterLeftovers(plan SchemaPlan) {
	toVersion := strings.TrimSpace(m.toVersion)
	if toVersion == "" || m.runtimeScope == nil || m.runtimeScope.Session() == nil || m.runtimeScope.Logger() == nil {
		return
	}
	tables := make([]string, 0, len(plan.Leftover))
	for _, left := range plan.Leftover {
		if left.Kind != LeftoverColumn {
			continue
		}
		tables = append(tables, left.Table)
	}
	if len(tables) == 0 {
		return
	}
	snaps, err := loadSnapshotsFn(m.runtimeScope.Session().DB, tables)
	if err != nil {
		m.runtimeScope.Logger().Warn("dropAfter leftover warning skipped: snapshot load failed", "error", err)
		return
	}
	if len(snaps) == 0 {
		return
	}
	for _, left := range plan.Leftover {
		if left.Kind != LeftoverColumn {
			continue
		}
		snap, ok := snaps[left.Table]
		if !ok || len(snap.DesiredJSON) == 0 {
			continue
		}
		var cols []ColumnSpec
		if err := json.Unmarshal(snap.DesiredJSON, &cols); err != nil {
			m.runtimeScope.Logger().Warn("dropAfter leftover warning skipped: snapshot decode failed",
				"table", left.Table, "error", err)
			continue
		}
		for _, col := range cols {
			if !columnMatchesLeftoverName(col, left.Name) {
				continue
			}
			if !versionHintEqual(col.DropAfter, toVersion) {
				continue
			}
			if IntentSatisfies(PlanOp{
				Kind:   OpKind(IntentDropColumn),
				Safety: SafetyManual,
				Table:  left.Table,
				Detail: "drop column " + left.Name,
				Column: &ColumnSpec{Name: left.Name},
			}, m.intents) {
				continue
			}
			m.runtimeScope.Logger().Warn("leftover column marked dropAfter for this version has no drop Intent",
				"table", left.Table,
				"column", left.Name,
				"dropAfter", col.DropAfter,
			)
		}
	}
}

// versionHintEqual compares dropAfter / module version hints, ignoring a leading v/V.
func versionHintEqual(a, b string) bool {
	return normalizeVersionHint(a) == normalizeVersionHint(b)
}

func normalizeVersionHint(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && (v[0] == 'v' || v[0] == 'V') && v[1] >= '0' && v[1] <= '9' {
		return v[1:]
	}
	return v
}

// columnMatchesLeftoverName reports whether a desired column describes leftover physical name
// (current name, or renameFrom when the leftover is the pre-rename column).
func columnMatchesLeftoverName(col ColumnSpec, leftoverName string) bool {
	if strings.EqualFold(col.Name, leftoverName) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(col.RenameFrom), leftoverName)
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
		if col == nil || strings.TrimSpace(col.CheckExpr) == "" || strings.TrimSpace(col.Name) == "" {
			continue
		}
		constraintName := fmt.Sprintf("chk_%s_%s", tableName, col.Name)
		legacyConstraintName := fmt.Sprintf("ck_%s_%s", tableName, col.Name)
		if legacyConstraintName != constraintName {
			_ = dropCheckConstraintBestEffort(m.runtimeScope.Session().DB, dialect, tableName, legacyConstraintName)
		}
		if err := ensureCheckConstraint(m.runtimeScope.Session().DB, dialect, tableName, constraintName, col.CheckExpr); err != nil {
			return err
		}
	}
	return nil
}
