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

// SyncSourceSnapshotsThenMaterialize syncs source snapshots then rewrites catalog rows.
// originType: local | registry | all (empty => all).
func (o *Ops) SyncSourceSnapshotsThenMaterialize(ctx context.Context, originType string) (lifecycle.ModuleSourceSnapshotSyncStats, error) {
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

	var err error
	switch originType {
	case originLocal:
		stats, err = syncLocal(ctx, o.Scope, o.LockerFactory)
	case originRegistry:
		stats, err = syncRegistry(ctx, o.Scope, o.LockerFactory)
	case "all":
		regStats, regErr := syncRegistry(ctx, o.Scope, o.LockerFactory)
		stats.Total += regStats.Total
		stats.Success += regStats.Success
		stats.Failed += regStats.Failed
		localStats, localErr := syncLocal(ctx, o.Scope, o.LockerFactory)
		stats.Total += localStats.Total
		stats.Success += localStats.Success
		stats.Failed += localStats.Failed
		err = errors.Join(regErr, localErr)
	default:
		return stats, errors.New("ops: originType must be one of: local, registry, all")
	}
	// Always materialize so a failed origin (e.g. offline registry) still
	// reflects snapshots that did sync successfully.
	if matErr := o.MaterializeCatalog(ctx, nil); matErr != nil {
		return stats, errors.Join(err, matErr)
	}
	return stats, err
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
		rows = append(rows, materializeOne(name, a, now, existingDemo[name]))
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
	return o.refreshCatalogDependents(ctx)
}

// catalogMaterializeDoUpdates builds dialect-safe ON CONFLICT assignments.
// MySQL rewrites clause.Column{Table:"excluded"} to VALUES(col); raw "excluded.x"
// expressions are not rewritten and break nonempty materialize on MySQL.
// dependents_json is intentionally omitted — refreshCatalogDependents recomputes it.
func catalogMaterializeDoUpdates(dialect string) clause.Set {
	cols := []string{
		"summary", "description", "category", "application", "author", "license",
		"local_version", "registry_version", "installed", "installed_version",
		"update_available", "registry_newer_than_local", "available", "unavailable_reason",
		"local_path", "depends_json", "has_demo", "last_sync_at", "updated_at",
	}
	updates := clause.AssignmentColumns(cols)
	return append(updates, clause.Assignment{
		Column: clause.Column{Name: "demo_loaded"},
		Value:  gorm.Expr(catalogDemoLoadedUpsertExpr(dialect)),
	})
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

func materializeOne(name string, a *catalogAggregate, now time.Time, prior catalogDemoState) modmeta.ModuleCatalog {
	row := modmeta.ModuleCatalog{
		ModuleName: name,
		Available:  true,
	}
	localV, registryV := "", ""
	if a.local != nil {
		if a.local.Version.Valid {
			localV = strings.TrimSpace(a.local.Version.String)
		}
		row.LocalVersion = a.local.Version
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
		row.RegistryVersion = a.registry.Version
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
		row.InstalledVersion = nullString(installedV)
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

type manifestLite struct {
	Summary     string          `json:"summary"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Application string          `json:"application"`
	Author      string          `json:"author"`
	License     string          `json:"license"`
	Depends     json.RawMessage `json:"depends"`
	Demo        json.RawMessage `json:"demo"`
	Choysum     *struct {
		Demo json.RawMessage `json:"demo"`
	} `json:"choysum"`
}

func fillCatalogFromManifest(row *modmeta.ModuleCatalog, raw datatypes.JSON) {
	if len(raw) == 0 || row == nil {
		return
	}
	var m manifestLite
	if err := json.Unmarshal(raw, &m); err != nil {
		return
	}
	if row.Summary == "" {
		row.Summary = m.Summary
	}
	if row.Description == "" {
		row.Description = m.Description
	}
	if row.Category == "" {
		row.Category = m.Category
	}
	if row.Application == "" {
		row.Application = m.Application
	}
	if row.Author == "" {
		row.Author = m.Author
	}
	if row.License == "" {
		row.License = m.License
	}
	if len(row.DependsJson) == 0 && len(m.Depends) > 0 {
		row.DependsJson = datatypes.JSON(m.Depends)
	}
	if !row.HasDemo {
		if len(m.Demo) > 0 && string(m.Demo) != "null" && string(m.Demo) != "[]" && string(m.Demo) != "{}" {
			row.HasDemo = true
		} else if m.Choysum != nil && len(m.Choysum.Demo) > 0 && string(m.Choysum.Demo) != "null" {
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
