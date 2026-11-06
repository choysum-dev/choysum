// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

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
		syncLocal = lifecycle.SyncLocalModuleSourceSnapshot
	}
	syncRegistry := o.SyncRegistrySnapshot
	if syncRegistry == nil {
		syncRegistry = lifecycle.SyncRegistryModuleSourceSnapshot
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
	if err != nil {
		return stats, err
	}
	if matErr := o.MaterializeCatalog(ctx, nil); matErr != nil {
		return stats, matErr
	}
	return stats, nil
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

	now := time.Now().UTC()
	rows := make([]modmeta.ModuleCatalog, 0, len(byName))
	for name, a := range byName {
		rows = append(rows, materializeOne(name, a, now))
	}
	if len(rows) == 0 {
		return nil
	}
	return sess.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "module_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"summary", "description", "category", "application", "author", "license",
			"local_version", "registry_version", "installed", "installed_version",
			"update_available", "registry_newer_than_local", "available", "unavailable_reason",
			"local_path", "depends_json", "dependents_json", "has_demo", "demo_loaded",
			"last_sync_at", "updated_at",
		}),
	}).Create(&rows).Error
}

func materializeOne(name string, a *catalogAggregate, now time.Time) modmeta.ModuleCatalog {
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
	return row
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
