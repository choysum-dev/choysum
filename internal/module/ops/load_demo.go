// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// applyDemoFilesFn imports only choysum.demo files for one installed module (no Build).
type applyDemoFilesFn func(ctx context.Context, runtimeScope scope.Scope, mod *pkgmeta.Module) error

// Overridable for tests; production uses importpkg demo-only initdata.
var applyDemoFiles applyDemoFilesFn = defaultApplyDemoFiles

func defaultApplyDemoFiles(ctx context.Context, runtimeScope scope.Scope, mod *pkgmeta.Module) error {
	if mod == nil {
		return nil
	}
	demoFiles, err := dataloader.ManifestDemoFiles(mod)
	if err != nil {
		return err
	}
	if len(demoFiles) == 0 {
		return nil
	}
	path := strings.TrimSpace(mod.Path)
	if path == "" {
		return fmt.Errorf("ops: module %q has empty path", mod.Name)
	}
	spec := importpkg.Spec{
		Profile:     importpkg.ProfileInitdata,
		Caller:      importpkg.CallerLifecycle,
		Policy:      importpkg.PolicyAtomic,
		Module:      mod.Name,
		Application: mod.ApplicationStr,
		Source: importpkg.Source{
			Format: "json",
			Path:   path,
		},
		Options: importpkg.Options{
			WithDemo:  true,
			DemoFiles: demoFiles,
		},
	}
	_, err = importpkg.Run(ctx, runtimeScope, spec)
	return err
}

func (o *Ops) planLoadDemo(ctx context.Context, operatorUserId string) (PlanResult, error) {
	pending, err := o.listPendingDemoCatalog(ctx)
	if err != nil {
		return PlanResult{}, err
	}
	order := make([]ModuleOrderEntry, 0, len(pending))
	for _, row := range pending {
		order = append(order, ModuleOrderEntry{
			ModuleName:     row.ModuleName,
			Reason:         "pending-demo",
			CurrentVersion: strings.TrimSpace(row.InstalledVersion.String),
			TargetVersion:  strings.TrimSpace(row.InstalledVersion.String),
		})
	}
	doc := PlanDocument{
		ModuleOrder:  order,
		BaseRevision: "load-demo",
	}
	if len(order) == 0 {
		doc.Blockers = []PlanIssue{{
			Code:    "NO_DEMO_PENDING",
			Level:   "error",
			Message: "no installed modules need demo data",
		}}
	}
	raw, err := marshalPlanDocument(doc)
	if err != nil {
		return PlanResult{}, err
	}
	row := &modmeta.ModuleOperation{
		Action:         ActionLoadDemo,
		PlanJson:       raw,
		Status:         StatusDraft,
		OperatorUserId: operatorUserId,
	}
	if err := o.createOperation(ctx, row); err != nil {
		return PlanResult{}, err
	}
	return PlanResult{
		Id:           row.Id.String,
		Action:       ActionLoadDemo,
		ModuleOrder:  order,
		Blockers:     doc.Blockers,
		BaseRevision: doc.BaseRevision,
	}, nil
}

func (o *Ops) listPendingDemoCatalog(ctx context.Context) ([]modmeta.ModuleCatalog, error) {
	sess, err := o.session(ctx)
	if err != nil {
		return nil, err
	}
	var rows []modmeta.ModuleCatalog
	if err := sess.WithContext(ctx).
		Where("installed = ? AND has_demo = ? AND demo_loaded = ?", true, true, false).
		Order("module_name asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (o *Ops) executeLoadDemo(ctx context.Context, row *modmeta.ModuleOperation) error {
	doc, err := unmarshalPlanDocument(row.PlanJson)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(doc.ModuleOrder))
	for _, entry := range doc.ModuleOrder {
		name := strings.TrimSpace(entry.ModuleName)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil
	}

	if o.Scope == nil {
		return errors.New("ops: scope is required")
	}
	return o.withModuleManagementLease(ctx, func(leaseCtx context.Context) error {
		latchSet := false
		touched := make([]string, 0, len(names))
		for _, name := range names {
			pending, pendingErr := o.isPendingDemoModule(leaseCtx, name)
			if pendingErr != nil {
				return pendingErr
			}
			if !pending {
				// Catalog moved on since Plan (already loaded / uninstalled / no demo).
				continue
			}
			mod, loadErr := o.loadInstalledModule(leaseCtx, name)
			if loadErr != nil {
				return loadErr
			}
			if err := applyDemoFiles(leaseCtx, o.Scope, mod); err != nil {
				return fmt.Errorf("ops: load demo for %q: %w", name, err)
			}
			// Latch before DemoLoaded so a latch write failure does not leave modules
			// marked (retry would skip them and never set modules.with_demo).
			if !latchSet {
				if err := o.SetWithDemo(leaseCtx, true); err != nil {
					return err
				}
				latchSet = true
			}
			// Mark per module after a successful import so a later failure does not
			// leave earlier modules unmarked (retry would re-import them).
			if err := o.markDemoLoaded(leaseCtx, []string{name}); err != nil {
				return err
			}
			touched = append(touched, name)
		}
		if len(touched) == 0 {
			return nil
		}
		if err := o.MaterializeCatalog(leaseCtx, touched); err != nil {
			return fmt.Errorf("ops: materialize catalog after load-demo: %w", err)
		}
		return nil
	})
}

func (o *Ops) isPendingDemoModule(ctx context.Context, name string) (bool, error) {
	sess, err := o.session(ctx)
	if err != nil {
		return false, err
	}
	var count int64
	if err := sess.WithContext(ctx).
		Model(&modmeta.ModuleCatalog{}).
		Where("module_name = ? AND installed = ? AND has_demo = ? AND demo_loaded = ?", name, true, true, false).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (o *Ops) loadInstalledModule(ctx context.Context, name string) (*pkgmeta.Module, error) {
	sess, err := o.session(ctx)
	if err != nil {
		return nil, err
	}
	var mod pkgmeta.Module
	if err := sess.WithContext(ctx).
		Where("name = ? AND status = ?", name, pkgmeta.Installed).
		Take(&mod).Error; err != nil {
		return nil, fmt.Errorf("ops: installed module %q: %w", name, err)
	}
	return &mod, nil
}

func (o *Ops) markDemoLoaded(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	sess, err := o.session(ctx)
	if err != nil {
		return err
	}
	return sess.WithContext(ctx).
		Model(&modmeta.ModuleCatalog{}).
		Where("module_name IN ? AND installed = ? AND has_demo = ? AND demo_loaded = ?", names, true, true, false).
		Update("demo_loaded", true).Error
}
