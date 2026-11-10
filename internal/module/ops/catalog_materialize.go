// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type catalogDemoState struct {
	DemoLoaded       bool
	InstalledVersion string
}

type catalogAggregate struct {
	local     *modmeta.ModuleSourceSnapshot
	registry  *modmeta.ModuleSourceSnapshot
	installed *pkgmeta.Module
}

// catalogLifecycleTimes holds the latest install/upgrade FinishedAt for Overview.
type catalogLifecycleTimes struct {
	installedAt    *time.Time
	lastUpgradedAt *time.Time
}

func (t catalogLifecycleTimes) lastInstalledOrUpgradedAt(fallback *time.Time) *time.Time {
	var best *time.Time
	pick := func(candidate *time.Time) {
		if candidate == nil {
			return
		}
		if best == nil || candidate.After(*best) {
			best = candidate
		}
	}
	pick(t.installedAt)
	pick(t.lastUpgradedAt)
	if best != nil {
		return best
	}
	return fallback
}

// SyncSourceSnapshotsThenMaterialize syncs source snapshots then rewrites catalog rows.
// originType: local | registry | all (empty => all).
// Origin sync errors are returned even when materialize succeeds so callers can
// decide soft vs hard failure; ActionSync treats materialize success as terminal success.
func (o *Ops) SyncSourceSnapshotsThenMaterialize(ctx context.Context, originType string) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
	stats, originErr := o.syncSourceSnapshots(ctx, originType)
	if originType == "" {
		originType = "all"
	}
	// Always materialize so a failed origin (e.g. offline registry) still
	// reflects snapshots that did sync successfully.
	if matErr := o.MaterializeCatalog(ctx, nil); matErr != nil {
		return stats, errors.Join(originErr, matErr)
	}
	return stats, originErr
}

// syncSourceSnapshots updates local and/or registry source snapshots without materializing.
func (o *Ops) syncSourceSnapshots(ctx context.Context, originType string) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
	stats := lifecycle.ModuleSourceSnapshotSyncStats{}
	if o == nil {
		return stats, errors.New("ops: nil host")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if o.LockerFactory == nil {
		return stats, errors.New("ops: locker factory is required")
	}
	originType = strings.TrimSpace(strings.ToLower(originType))
	if originType == "" {
		originType = "all"
	}

	syncLocal := o.SyncLocalSnapshot
	if syncLocal == nil {
		syncLocal = defaultSyncLocal
	}
	syncRegistry := o.SyncRegistrySnapshot
	if syncRegistry == nil {
		syncRegistry = defaultSyncRegistry
	}

	switch originType {
	case originLocal:
		return syncLocal(ctx, o.Scope, o.LockerFactory)
	case originRegistry:
		return syncRegistry(ctx, o.Scope, o.LockerFactory)
	case "all":
		regStats, regErr := syncRegistry(ctx, o.Scope, o.LockerFactory)
		stats.Total += regStats.Total
		stats.Success += regStats.Success
		stats.Failed += regStats.Failed
		localStats, localErr := syncLocal(ctx, o.Scope, o.LockerFactory)
		stats.Total += localStats.Total
		stats.Success += localStats.Success
		stats.Failed += localStats.Failed
		return stats, errors.Join(regErr, localErr)
	default:
		return stats, errors.New("ops: originType must be one of: local, registry, all")
	}
}

func (o *Ops) countCatalogRows(ctx context.Context) (int, error) {
	sess, err := o.session(ctx)
	if err != nil {
		return 0, err
	}
	var n int64
	if err := sess.WithContext(ctx).Model(&modmeta.ModuleCatalog{}).Count(&n).Error; err != nil {
		return 0, err
	}
	return int(n), nil
}

func (o *Ops) persistSyncResult(ctx context.Context, row *modmeta.ModuleOperation, summary SyncResultSummary) error {
	if row == nil {
		return errors.New("ops: nil operation")
	}
	doc, err := unmarshalPlanDocument(row.PlanJson)
	if err != nil {
		doc = PlanDocument{BaseRevision: "sync"}
	}
	if strings.TrimSpace(doc.BaseRevision) == "" {
		doc.BaseRevision = "sync"
	}
	copied := summary
	doc.SyncResult = &copied
	raw, err := marshalPlanDocument(doc)
	if err != nil {
		return err
	}
	row.PlanJson = raw
	return o.casSaveOperation(ctx, row, StatusRunning)
}

// MaterializeCatalog rewrites meta_module_catalog rows from source snapshots + installed modules.
// Empty moduleNames means full rewrite of all known names.
func (o *Ops) MaterializeCatalog(ctx context.Context, moduleNames []string) error {
	if o == nil {
		return errors.New("ops: nil host")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}

	filter := make(map[string]struct{})
	names := make([]string, 0, len(moduleNames))
	for _, name := range moduleNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := filter[name]; ok {
			continue
		}
		filter[name] = struct{}{}
		names = append(names, name)
	}
	filtered := len(filter) > 0

	var snapshots []modmeta.ModuleSourceSnapshot
	q := sess.WithContext(ctx).Model(&modmeta.ModuleSourceSnapshot{})
	if filtered {
		q = q.Where("module_name IN ?", names)
	}
	if err := q.Find(&snapshots).Error; err != nil {
		return err
	}

	var installed []pkgmeta.Module
	iq := sess.WithContext(ctx).Model(&pkgmeta.Module{}).Where("status = ?", pkgmeta.Installed)
	if filtered {
		iq = iq.Where("name IN ?", names)
	}
	if err := iq.Find(&installed).Error; err != nil {
		return err
	}

	byName := map[string]*catalogAggregate{}
	ensure := func(name string) *catalogAggregate {
		if a, ok := byName[name]; ok {
			return a
		}
		a := &catalogAggregate{}
		byName[name] = a
		return a
	}
	for i := range snapshots {
		row := snapshots[i]
		name := strings.TrimSpace(row.ModuleName)
		if name == "" {
			continue
		}
		a := ensure(name)
		switch row.SourceType {
		case originLocal:
			a.local = &snapshots[i]
		case originRegistry:
			a.registry = &snapshots[i]
		}
	}
	for i := range installed {
		mod := installed[i]
		name := strings.TrimSpace(mod.Name)
		if name == "" {
			continue
		}
		ensure(name).installed = &installed[i]
	}

	lifecycleTimes, err := loadCatalogLifecycleTimes(sess.WithContext(ctx), names, filtered)
	if err != nil {
		return err
	}

	existingDemo := map[string]catalogDemoState{}
	{
		var prior []modmeta.ModuleCatalog
		pq := sess.WithContext(ctx).Model(&modmeta.ModuleCatalog{}).
			Select("module_name", "demo_loaded", "installed_version")
		if filtered {
			pq = pq.Where("module_name IN ?", names)
		}
		if err := pq.Find(&prior).Error; err != nil {
			return err
		}
		for _, p := range prior {
			existingDemo[p.ModuleName] = catalogDemoState{
				DemoLoaded:       p.DemoLoaded,
				InstalledVersion: strings.TrimSpace(p.InstalledVersion.String),
			}
		}
	}

	now := time.Now().UTC()
	rows := make([]modmeta.ModuleCatalog, 0, len(byName))
	for name, a := range byName {
		rows = append(rows, materializeOne(name, a, now, existingDemo[name], lifecycleTimes[name]))
	}
	if len(rows) == 0 {
		return nil
	}
	fillCatalogDependents(rows)
	dialect := ""
	if sess.Dialector != nil {
		dialect = sess.Dialector.Name()
	}
	if err := sess.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "module_name"}},
		DoUpdates: catalogMaterializeDoUpdates(dialect),
	}).CreateInBatches(&rows, 200).Error; err != nil {
		return err
	}
	// Keep reverse-dep edges coherent for modules outside this batch
	// (and repair dependents_json after filtered materialize, which must not
	// upsert a partial dependents graph into the conflict update).
	if err := o.refreshCatalogDependents(ctx); err != nil {
		return err
	}
	return o.replaceCatalogDependsEdges(ctx)
}

// catalogMaterializeDoUpdates builds dialect-safe ON CONFLICT assignments.
// MySQL rewrites clause.Column{Table:"excluded"} to VALUES(col); raw "excluded.x"
// expressions are not rewritten and break nonempty materialize on MySQL.
// dependents_json is intentionally omitted — refreshCatalogDependents recomputes it.
//
// demo_loaded must be assigned before installed_version: MySQL ON DUPLICATE KEY
// UPDATE is left-to-right, so a later CASE that reads meta_module_catalog.installed_version
// would otherwise see the already-updated value and skip the upgrade clear.
func catalogMaterializeDoUpdates(dialect string) clause.Set {
	beforeInstalledVersion := []string{
		"summary", "description", "category", "application", "author", "license",
		"local_version", "registry_version", "installed",
		"last_installed_or_upgraded_at",
		"update_available", "registry_newer_than_local", "available", "unavailable_reason",
		"local_path", "registry_path", "registry_tarball", "depends_json", "has_demo", "last_sync_at", "updated_at",
	}
	updates := clause.AssignmentColumns(beforeInstalledVersion)
	updates = append(updates, clause.Assignment{
		Column: clause.Column{Name: "demo_loaded"},
		Value:  gorm.Expr(catalogDemoLoadedUpsertExpr(dialect)),
	})
	updates = append(updates, clause.Assignment{
		Column: clause.Column{Name: "installed_version"},
		Value:  clause.Column{Table: "excluded", Name: "installed_version"},
	})
	return updates
}

func catalogDemoLoadedUpsertExpr(dialect string) string {
	in := func(col string) string {
		if dialect == "mysql" {
			return "VALUES(`" + col + "`)"
		}
		return "excluded." + col
	}
	curDemo := "meta_module_catalog.demo_loaded"
	curInstalled := "meta_module_catalog.installed_version"
	if dialect == "mysql" {
		curDemo = "`meta_module_catalog`.`demo_loaded`"
		curInstalled = "`meta_module_catalog`.`installed_version`"
	}
	return `CASE
		WHEN ` + in("has_demo") + ` = false THEN false
		WHEN ` + in("installed") + ` = false THEN false
		WHEN ` + in("demo_loaded") + ` = false AND COALESCE(` + curInstalled + `, '') != COALESCE(` + in("installed_version") + `, '') THEN false
		WHEN ` + curDemo + ` = true THEN true
		ELSE ` + in("demo_loaded") + `
	END`
}

func materializeOne(name string, a *catalogAggregate, now time.Time, prior catalogDemoState, life catalogLifecycleTimes) modmeta.ModuleCatalog {
	row := modmeta.ModuleCatalog{
		ModuleName: name,
		Available:  true,
	}
	localV, registryV := "", ""
	if a.local != nil {
		if a.local.Version.Valid {
			localV = strings.TrimSpace(a.local.Version.String)
		}
		row.LocalVersion = nullString(displayVersion(localV))
		row.LocalPath = a.local.LocalPath
		if a.local.LastSyncAt != nil {
			row.LastSyncAt = a.local.LastSyncAt
		}
		fillCatalogFromManifest(&row, a.local.ManifestJson)
		if !a.local.Available {
			row.Available = false
			row.UnavailableReason = a.local.LastErrorMessage
		}
	}
	if a.registry != nil {
		if a.registry.Version.Valid {
			registryV = strings.TrimSpace(a.registry.Version.String)
		}
		row.RegistryVersion = nullString(displayVersion(registryV))
		pkg, tarball := registryOriginFromSnapshot(a.registry)
		row.RegistryPath = nullString(pkg)
		row.RegistryTarball = nullString(tarball)
		if row.LastSyncAt == nil && a.registry.LastSyncAt != nil {
			row.LastSyncAt = a.registry.LastSyncAt
		}
		if row.Summary == "" {
			fillCatalogFromManifest(&row, a.registry.ManifestJson)
		}
		if !a.registry.Available && a.local == nil {
			row.Available = false
			row.UnavailableReason = a.registry.LastErrorMessage
		}
	}
	installedV := ""
	if a.installed != nil {
		row.Installed = true
		installedV = strings.TrimSpace(a.installed.Version)
		row.InstalledVersion = nullString(displayVersion(installedV))
		var createdAt *time.Time
		if !a.installed.CreatedAt.IsZero() {
			t := a.installed.CreatedAt.UTC()
			createdAt = &t
		}
		row.LastInstalledOrUpgradedAt = life.lastInstalledOrUpgradedAt(createdAt)
		if row.Summary == "" {
			row.Summary = a.installed.Summary
		}
		if row.Description == "" {
			row.Description = a.installed.Description
		}
		if row.Category == "" {
			row.Category = a.installed.Category
		}
		if row.Application == "" {
			row.Application = a.installed.ApplicationStr
		}
		if row.Author == "" {
			row.Author = a.installed.Author
		}
		if row.License == "" {
			row.License = a.installed.License
		}
		row.DependsJson = a.installed.DependsStr
		row.HasDemo = hasDemoPayload(a.installed.DemoStr)
		if row.LocalPath.String == "" && a.installed.Path != "" {
			row.LocalPath = nullString(a.installed.Path)
		}
	}
	defaultFetch := firstNonEmpty(localV, registryV)
	row.UpdateAvailable = row.Installed && versionNewer(defaultFetch, installedV)
	row.RegistryNewerThanLocal = localV != "" && registryV != "" && versionNewer(registryV, localV)
	if row.LastSyncAt == nil && (a.local != nil || a.registry != nil) {
		row.LastSyncAt = &now
	}
	switch {
	case !row.Installed:
		// Uninstall (or never installed): demo rows are gone; clear the CTA gate.
		row.DemoLoaded = false
	case !row.HasDemo:
		row.DemoLoaded = false
	case prior.DemoLoaded && prior.InstalledVersion != "" && installedV != "" &&
		normalizeVersion(prior.InstalledVersion) != normalizeVersion(installedV):
		// Upgrade (installed version changed): allow load-demo to reappear.
		row.DemoLoaded = false
	default:
		row.DemoLoaded = prior.DemoLoaded
	}
	return row
}

// loadCatalogLifecycleTimes reads the latest succeeded install/upgrade FinishedAt
// per module for catalog Overview timestamps.
func loadCatalogLifecycleTimes(db *gorm.DB, names []string, filtered bool) (map[string]catalogLifecycleTimes, error) {
	out := map[string]catalogLifecycleTimes{}
	if db == nil {
		return out, nil
	}
	var rows []modmeta.ModuleOperation
	q := db.Model(&modmeta.ModuleOperation{}).
		Select("module_name", "action", "finished_at").
		Where("status = ? AND action IN ? AND finished_at IS NOT NULL", StatusSucceeded, []string{ActionInstall, ActionUpgrade})
	if filtered {
		if len(names) == 0 {
			return out, nil
		}
		q = q.Where("module_name IN ?", names)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		row := rows[i]
		name := strings.TrimSpace(row.ModuleName)
		if name == "" || row.FinishedAt == nil || row.FinishedAt.IsZero() {
			continue
		}
		t := row.FinishedAt.UTC()
		cur := out[name]
		switch row.Action {
		case ActionInstall:
			if cur.installedAt == nil || t.After(*cur.installedAt) {
				cur.installedAt = &t
			}
		case ActionUpgrade:
			if cur.lastUpgradedAt == nil || t.After(*cur.lastUpgradedAt) {
				cur.lastUpgradedAt = &t
			}
		}
		out[name] = cur
	}
	return out, nil
}

func decodeDependsNames(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return nil
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil
	}
	return uniqueTrimmedNames(names)
}

// dependsNamesForEdges preserves duplicate depends entries until replaceCatalogDependsEdges dedupes edges.
func dependsNamesForEdges(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return nil
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	return out
}

func fillCatalogDependents(rows []modmeta.ModuleCatalog) {
	depends := make(map[string][]string, len(rows))
	for _, row := range rows {
		depends[row.ModuleName] = decodeDependsNames(row.DependsJson)
	}
	reverse := map[string][]string{}
	for name, deps := range depends {
		for _, dep := range deps {
			reverse[dep] = append(reverse[dep], name)
		}
	}
	for i := range rows {
		deps := uniqueTrimmedNames(reverse[rows[i].ModuleName])
		sort.Strings(deps)
		if len(deps) == 0 {
			rows[i].DependentsJson = datatypes.JSON("[]")
			continue
		}
		b, _ := json.Marshal(deps) // []string always marshals
		rows[i].DependentsJson = datatypes.JSON(b)
	}
}

func (o *Ops) refreshCatalogDependents(ctx context.Context) error {
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	var rows []modmeta.ModuleCatalog
	if err := sess.WithContext(ctx).
		Model(&modmeta.ModuleCatalog{}).
		Select("module_name", "depends_json", "dependents_json").
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	prior := make(map[string]string, len(rows))
	for _, row := range rows {
		prior[row.ModuleName] = string(row.DependentsJson)
	}
	fillCatalogDependents(rows)
	grouped := map[string][]string{}
	for _, row := range rows {
		next := string(row.DependentsJson)
		if next == prior[row.ModuleName] {
			continue
		}
		grouped[next] = append(grouped[next], row.ModuleName)
	}
	if len(grouped) == 0 {
		return nil
	}
	return sess.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for val, names := range grouped {
			if err := tx.Model(&modmeta.ModuleCatalog{}).
				Where("module_name IN ?", names).
				Update("dependents_json", datatypes.JSON(val)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ensureModuleCatalogDependTable creates meta_module_catalog_depend when absent
// (e.g. load-demo / materialize before meta schema evolve has run in tests).
// Never AutoMigrate: schema authority remains the meta TS evolve path.
func ensureModuleCatalogDependTable(db *gorm.DB) error {
	if db == nil {
		return errors.New("ops: nil db")
	}
	m := db.Migrator()
	if m.HasTable(&modmeta.ModuleCatalogDepend{}) {
		return nil
	}
	return m.CreateTable(&modmeta.ModuleCatalogDepend{})
}

// replaceCatalogDependsEdges rewrites meta_module_catalog_depend from DependsJson names.
// Full replace keeps self-ManyToMany Depends/Dependents coherent after every materialize.
func (o *Ops) replaceCatalogDependsEdges(ctx context.Context) error {
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	if err := ensureModuleCatalogDependTable(sess.WithContext(ctx)); err != nil {
		return err
	}
	var rows []modmeta.ModuleCatalog
	if err := sess.WithContext(ctx).
		Model(&modmeta.ModuleCatalog{}).
		Select("id", "module_name", "depends_json").
		Find(&rows).Error; err != nil {
		return err
	}
	idByName := make(map[string]string, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.ModuleName)
		id := strings.TrimSpace(row.Id.String)
		if name == "" || id == "" {
			continue
		}
		idByName[name] = id
	}
	edges := make([]modmeta.ModuleCatalogDepend, 0)
	seen := make(map[string]struct{})
	for _, row := range rows {
		moduleID := strings.TrimSpace(row.Id.String)
		if moduleID == "" {
			continue
		}
		for _, depName := range dependsNamesForEdges(row.DependsJson) {
			dependID := idByName[depName]
			if dependID == "" || dependID == moduleID {
				continue
			}
			key := moduleID + "\x00" + dependID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			edges = append(edges, modmeta.ModuleCatalogDepend{
				ModuleID: moduleID,
				DependID: dependID,
			})
		}
	}
	return sess.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).
			Where("1 = 1").
			Delete(&modmeta.ModuleCatalogDepend{}).Error; err != nil {
			return err
		}
		if len(edges) == 0 {
			return nil
		}
		return tx.CreateInBatches(&edges, 200).Error
	})
}

type manifestLiteFields struct {
	Summary     string          `json:"summary"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Application string          `json:"application"`
	Author      string          `json:"author"`
	License     string          `json:"license"`
	Depends     json.RawMessage `json:"depends"`
	Demo        json.RawMessage `json:"demo"`
}

type manifestLite struct {
	manifestLiteFields
	Choysum *manifestLiteFields `json:"choysum"`
}

// registryOriginFromSnapshot reads index.choysum.dev package id + latest tarball
// from a registry ModuleSourceSnapshot (SourceRef and/or ManifestJson).
func registryOriginFromSnapshot(snap *modmeta.ModuleSourceSnapshot) (packageID, tarball string) {
	if snap == nil {
		return "", ""
	}
	packageID = strings.TrimSpace(snap.SourceRef)
	versionKey := ""
	if snap.Version.Valid {
		versionKey = displayVersion(strings.TrimSpace(snap.Version.String))
	}
	if len(snap.ManifestJson) == 0 {
		return packageID, ""
	}
	var doc struct {
		Package  string `json:"package"`
		Versions map[string]struct {
			Package string `json:"package"`
			Tarball string `json:"tarball"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(snap.ManifestJson, &doc); err != nil {
		return packageID, ""
	}
	if packageID == "" {
		packageID = strings.TrimSpace(doc.Package)
	}
	if versionKey != "" {
		if entry, ok := doc.Versions[versionKey]; ok {
			if packageID == "" {
				packageID = strings.TrimSpace(entry.Package)
			}
			return packageID, strings.TrimSpace(entry.Tarball)
		}
		// Snapshot Version may still carry a leading v; try the raw key too.
		if snap.Version.Valid {
			raw := strings.TrimSpace(snap.Version.String)
			if entry, ok := doc.Versions[raw]; ok {
				if packageID == "" {
					packageID = strings.TrimSpace(entry.Package)
				}
				return packageID, strings.TrimSpace(entry.Tarball)
			}
		}
	}
	return packageID, ""
}

func fillCatalogFromManifest(row *modmeta.ModuleCatalog, raw datatypes.JSON) {
	if len(raw) == 0 || row == nil {
		return
	}
	var m manifestLite
	if err := json.Unmarshal(raw, &m); err != nil {
		return
	}
	var nested manifestLiteFields
	if m.Choysum != nil {
		nested = *m.Choysum
	}
	if row.Summary == "" {
		row.Summary = firstNonEmpty(m.Summary, nested.Summary)
	}
	if row.Description == "" {
		row.Description = firstNonEmpty(m.Description, nested.Description)
	}
	if row.Category == "" {
		row.Category = firstNonEmpty(m.Category, nested.Category)
	}
	if row.Application == "" {
		row.Application = firstNonEmpty(m.Application, nested.Application)
	}
	if row.Author == "" {
		row.Author = firstNonEmpty(m.Author, nested.Author)
	}
	if row.License == "" {
		row.License = firstNonEmpty(m.License, nested.License)
	}
	depends := m.Depends
	if len(depends) == 0 {
		depends = nested.Depends
	}
	if len(row.DependsJson) == 0 && len(depends) > 0 {
		row.DependsJson = datatypes.JSON(depends)
	}
	if !row.HasDemo {
		demo := m.Demo
		if len(demo) == 0 || string(demo) == "null" {
			demo = nested.Demo
		}
		if len(demo) > 0 && string(demo) != "null" && string(demo) != "[]" && string(demo) != "{}" {
			row.HasDemo = true
		}
	}
}

func hasDemoPayload(raw datatypes.JSON) bool {
	if len(raw) == 0 {
		return false
	}
	s := strings.TrimSpace(string(raw))
	return s != "" && s != "null" && s != "[]" && s != "{}"
}
