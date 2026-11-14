// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/plan"
)

// Run executes a queued (or draft sync/load-demo for CLI convenience) operation.
func (o *Ops) Run(ctx context.Context, req RunRequest) error {
	if o == nil {
		return errors.New("ops: nil host")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	id := strings.TrimSpace(req.Id)
	if id == "" {
		return errors.New("ops: operation id is required")
	}
	row, err := o.loadOperation(ctx, id)
	if err != nil {
		return err
	}
	expectStatus := row.Status
	switch row.Status {
	case StatusQueued:
		// ok
	case StatusDraft:
		// draft is allowed only for CLI sync / load-demo stubs that skip Apply
		if row.Action != ActionSync && row.Action != ActionLoadDemo {
			return fmt.Errorf("ops: operation %q must be applied before run", id)
		}
	default:
		return fmt.Errorf("ops: operation %q status is %q, want queued or draft", id, row.Status)
	}
	now := time.Now().UTC()
	row.Status = StatusRunning
	row.Phase = PhaseExecuting
	row.StartedAt = &now
	row.ErrorDomain = ""
	row.ErrorCode = ""
	row.ErrorMessage = ""
	if err := o.casSaveOperation(ctx, row, expectStatus); err != nil {
		return err
	}
	var planDoc PlanDocument
	if row.Action != ActionSync {
		doc, docErr := unmarshalPlanDocument(row.PlanJson)
		if docErr != nil {
			finishCtx := context.WithoutCancel(ctx)
			if failErr := o.failOperation(finishCtx, row, "ops", "PLAN_DECODE_FAILED", docErr.Error()); failErr != nil {
				return errors.Join(docErr, failErr)
			}
			return docErr
		}
		planDoc = doc
		if stampErr := o.setCatalogActiveOp(ctx, operationModuleNames(row, doc), row.Id.String); stampErr != nil {
			finishCtx := context.WithoutCancel(ctx)
			// Drop any partial stamps if the terminal write also fails.
			o.clearCatalogActiveOpBestEffort(finishCtx, row.Id.String)
			if failErr := o.failOperation(finishCtx, row, "ops", "ACTIVE_OP_STAMP_FAILED", stampErr.Error()); failErr != nil {
				return errors.Join(stampErr, failErr)
			}
			return stampErr
		}
	}
	o.publishOperationChanged(ctx, row)

	runErr := o.executeOperation(ctx, row, req.WithDemo, planDoc)
	finishCtx := context.WithoutCancel(ctx)
	if runErr != nil {
		var saveErr error
		var committed *errAfterLifecycleCommit
		if errors.As(runErr, &committed) {
			// Lifecycle already committed; keep catalog failure visible but still
			// request process reload so the running binary picks up mutations.
			saveErr = o.failOperationAfterLifecycleCommit(finishCtx, row, "ops", "RUN_FAILED", runErr.Error())
		} else {
			saveErr = o.failOperation(finishCtx, row, "ops", "RUN_FAILED", runErr.Error())
		}
		if saveErr != nil {
			return errors.Join(runErr, saveErr)
		}
		return runErr
	}
	return o.succeedOperation(finishCtx, row)
}

// errAfterLifecycleCommit marks a failure after install/upgrade/uninstall already
// committed module mutations. Run fails the operation but still requests reload.
type errAfterLifecycleCommit struct {
	err error
}

func (e *errAfterLifecycleCommit) Error() string {
	if e == nil || e.err == nil {
		return "ops: catalog update failed after lifecycle commit"
	}
	return e.err.Error()
}

func (e *errAfterLifecycleCommit) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func afterLifecycleCommit(err error) error {
	if err == nil {
		return nil
	}
	return &errAfterLifecycleCommit{err: err}
}

func (o *Ops) executeOperation(ctx context.Context, row *modmeta.ModuleOperation, requestWithDemo bool, doc PlanDocument) error {
	switch row.Action {
	case ActionSync:
		return o.executeSync(ctx, row)
	case ActionLoadDemo:
		return o.executeLoadDemo(ctx, row)
	case ActionInstall, ActionUpgrade, ActionUninstall:
		if o.Lifecycle == nil {
			return errors.New("ops: lifecycle service is required")
		}
		withDemo := requestWithDemo || o.ResolveWithDemo(ctx)
		if requestWithDemo {
			if err := o.SetWithDemo(ctx, true); err != nil {
				return err
			}
			withDemo = true
		}
		names := affectedModuleNames(doc, row.ModuleName)
		stampNames := mutatedInstallUpgradeNames(doc)
		if err := o.runLifecycle(ctx, row.Action, row.ModuleName, withDemo, doc); err != nil {
			return err
		}
		// Soft-fail only origin snapshot sync (network). Local installed projection and
		// MaterializeCatalog read the DB only and must hard-fail after lifecycle success.
		// Do not re-acquire module_management here: Materialize was never leased on this
		// path, and test/runtime scopes often lack RequiresNew for lease.New.
		projectMaterializeStamp := func(matNames []string) error {
			if err := o.ProjectCatalogInstalled(ctx); err != nil {
				return fmt.Errorf("ops: project catalog installed after %s: %w", row.Action, err)
			}
			if err := o.MaterializeCatalog(ctx, matNames); err != nil {
				return fmt.Errorf("ops: materialize catalog after %s: %w", row.Action, err)
			}
			if err := o.stampCatalogLifecycleAt(ctx, stampNames, row.Action); err != nil {
				return fmt.Errorf("ops: stamp catalog lifecycle after %s: %w", row.Action, err)
			}
			return nil
		}
		if row.Action == ActionInstall && moduleNamesInclude(names, "meta") {
			// First install that lands meta: full local+registry snapshot sync so Catalog UI
			// is populated without a page-open auto sync. Offline registry must not fail the op.
			if _, originErr := o.syncSourceSnapshots(ctx, "all"); originErr != nil {
				if o.Scope != nil && o.Scope.Logger() != nil {
					o.Scope.Logger().Warn("ops: catalog sync after meta install failed", "error", originErr)
				}
			}
			if err := projectMaterializeStamp(nil); err != nil {
				return afterLifecycleCommit(err)
			}
			return nil
		}
		if err := projectMaterializeStamp(names); err != nil {
			return afterLifecycleCommit(err)
		}
		return nil
	default:
		return fmt.Errorf("ops: unsupported action %q", row.Action)
	}
}

func (o *Ops) runLifecycle(ctx context.Context, action, moduleName string, withDemo bool, doc PlanDocument) error {
	name := strings.TrimSpace(moduleName)
	input := strings.TrimSpace(doc.LifecycleInput)
	if input == "" {
		input = name
	}
	profile := plan.WebShellProfile(strings.TrimSpace(doc.WebShellProfile))
	switch action {
	case ActionInstall:
		return o.Lifecycle.Install(ctx, lifecycle.InstallRequest{
			Name:            name,
			WithDemo:        withDemo,
			SkipWebShell:    doc.SkipWebShell,
			WebShellProfile: profile,
		})
	case ActionUpgrade:
		return o.Lifecycle.Upgrade(ctx, lifecycle.UpgradeRequest{
			Input:           input,
			WithDemo:        withDemo,
			SkipWebShell:    doc.SkipWebShell,
			WebShellProfile: profile,
		})
	case ActionUninstall:
		return o.Lifecycle.Uninstall(ctx, lifecycle.UninstallRequest{Name: name})
	default:
		return fmt.Errorf("ops: unsupported lifecycle action %q", action)
	}
}

// executeSync syncs all origins, materializes catalog, and persists SyncResult on the op.
// Origin sync failures with a successful materialize are soft (return nil) so the UI can
// show a partial-failure summary while still refreshing the catalog.
func (o *Ops) executeSync(ctx context.Context, row *modmeta.ModuleOperation) error {
	const originType = "all"
	stats, originErr := o.syncSourceSnapshots(ctx, originType)
	matErr := o.MaterializeCatalog(ctx, nil)
	catalogRows, countErr := o.countCatalogRows(ctx)
	if countErr != nil {
		catalogRows = 0
	}
	summary := SyncResultSummary{
		Total:       stats.Total,
		Success:     stats.Success,
		Failed:      stats.Failed,
		CatalogRows: catalogRows,
		Origin:      originType,
	}
	if persistErr := o.persistSyncResult(ctx, row, summary); persistErr != nil {
		return errors.Join(matErr, originErr, persistErr)
	}
	if matErr != nil {
		return matErr
	}
	// Soft-fail origin errors: stats.Failed / originErr surface via SyncResult in the UI.
	if originErr != nil && o.Scope != nil && o.Scope.Logger() != nil {
		o.Scope.Logger().Warn("ops: catalog sync origin partially failed", "error", originErr)
	}
	return nil
}

func affectedModuleNames(doc PlanDocument, root string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(doc.ModuleOrder)+1)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, e := range doc.ModuleOrder {
		add(e.ModuleName)
	}
	for _, name := range doc.EnsureOrder {
		add(name)
	}
	add(root)
	return out
}

// mutatedInstallUpgradeNames lists modules whose plan effect is install or upgrade.
// Used for last_installed_or_upgraded_at stamps; already_satisfied deps are omitted.
func mutatedInstallUpgradeNames(doc PlanDocument) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(doc.ModuleOrder)+len(doc.EnsureModules))
	add := func(e ModuleOrderEntry) {
		switch strings.TrimSpace(e.Effect) {
		case EffectInstall, EffectUpgrade:
		default:
			return
		}
		name := strings.TrimSpace(e.ModuleName)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, e := range doc.ModuleOrder {
		add(e)
	}
	for _, e := range doc.EnsureModules {
		add(e)
	}
	return out
}

func moduleNamesInclude(names []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, name := range names {
		if strings.EqualFold(strings.TrimSpace(name), want) {
			return true
		}
	}
	return false
}

func (o *Ops) failOperation(ctx context.Context, row *modmeta.ModuleOperation, domain, code, message string) error {
	expect := row.Status
	now := time.Now().UTC()
	row.Status = StatusFailed
	row.Phase = PhaseDone
	row.ErrorDomain = domain
	row.ErrorCode = code
	row.ErrorMessage = message
	row.FinishedAt = &now
	if err := o.casSaveOperation(ctx, row, expect); err != nil {
		return err
	}
	// Terminal status is already persisted; do not fail the caller on a clear glitch.
	o.clearCatalogActiveOpBestEffort(ctx, row.Id.String)
	o.publishOperationChanged(ctx, row)
	return nil
}

// failOperationAfterLifecycleCommit marks the op failed while still requesting a
// process reload when the plan already committed mutating install/upgrade/uninstall.
func (o *Ops) failOperationAfterLifecycleCommit(ctx context.Context, row *modmeta.ModuleOperation, domain, code, message string) error {
	expect := row.Status
	now := time.Now().UTC()
	row.Status = StatusFailed
	row.ErrorDomain = domain
	row.ErrorCode = code
	row.ErrorMessage = message
	row.FinishedAt = &now
	if operationRequiresProcessReload(row) {
		row.Phase = PhaseReloading
		row.ReloadTriggered = true
		row.ReloadFailed = false
	} else {
		row.Phase = PhaseDone
		row.ReloadTriggered = false
		row.ReloadFailed = false
	}
	if err := o.casSaveOperation(ctx, row, expect); err != nil {
		return err
	}
	o.clearCatalogActiveOpBestEffort(ctx, row.Id.String)
	o.publishOperationChanged(ctx, row)
	return nil
}

// operationRequiresProcessReload is true when the plan actually changes installed
// modules (or decode fails and we fail safe). Already-satisfied install plans skip reload.
func operationRequiresProcessReload(row *modmeta.ModuleOperation) bool {
	if row == nil {
		return false
	}
	switch row.Action {
	case ActionUpgrade, ActionUninstall:
		return true
	case ActionInstall:
		doc, err := unmarshalPlanDocument(row.PlanJson)
		if err != nil || len(doc.ModuleOrder) == 0 {
			return true
		}
		for _, e := range doc.ModuleOrder {
			switch strings.TrimSpace(e.Effect) {
			case EffectInstall, EffectUpgrade, EffectUninstall:
				return true
			case EffectAlreadySatisfied, "":
				continue
			default:
				return true
			}
		}
		for _, e := range doc.EnsureModules {
			switch strings.TrimSpace(e.Effect) {
			case EffectInstall, EffectUpgrade, EffectUninstall:
				return true
			case EffectAlreadySatisfied, "":
				continue
			default:
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (o *Ops) succeedOperation(ctx context.Context, row *modmeta.ModuleOperation) error {
	expect := row.Status
	now := time.Now().UTC()
	row.Status = StatusSucceeded
	row.FinishedAt = &now
	row.ErrorDomain = ""
	row.ErrorCode = ""
	row.ErrorMessage = ""
	switch row.Action {
	case ActionInstall, ActionUpgrade, ActionUninstall:
		// Data is committed. When the plan mutates installed modules, leave
		// Phase=reloading so MetaModuleOperation.Execute schedules one process
		// reload (C1 ReloadWeb). No-op already_satisfied installs skip reload.
		if operationRequiresProcessReload(row) {
			row.Phase = PhaseReloading
			row.ReloadTriggered = true
			row.ReloadFailed = false
		} else {
			row.Phase = PhaseDone
			row.ReloadTriggered = false
			row.ReloadFailed = false
		}
	default:
		row.Phase = PhaseDone
	}
	if err := o.casSaveOperation(ctx, row, expect); err != nil {
		return err
	}
	// Terminal status is already persisted; do not fail the caller on a clear glitch.
	o.clearCatalogActiveOpBestEffort(ctx, row.Id.String)
	o.publishOperationChanged(ctx, row)
	return nil
}

// ForceFailTerminal flips an operation to failed with a CAS on the loaded status.
// Used by e2e bridge seams when a post-Run hook must override a successful Run write.
func (o *Ops) ForceFailTerminal(ctx context.Context, id, domain, code, message string) error {
	if o == nil {
		return errors.New("ops: nil host")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("ops: operation id is required")
	}
	row, err := o.loadOperation(ctx, id)
	if err != nil {
		return err
	}
	// Keep an earlier failure (e.g. Apply ACTIVE_OP_STAMP_FAILED) instead of overwriting.
	if row.Status == StatusFailed {
		return nil
	}
	expect := row.Status
	now := time.Now().UTC()
	row.Status = StatusFailed
	row.Phase = PhaseDone
	row.ErrorDomain = strings.TrimSpace(domain)
	row.ErrorCode = strings.TrimSpace(code)
	row.ErrorMessage = strings.TrimSpace(message)
	row.FinishedAt = &now
	if err := o.casSaveOperation(ctx, row, expect); err != nil {
		return err
	}
	o.clearCatalogActiveOpBestEffort(ctx, row.Id.String)
	o.publishOperationChanged(ctx, row)
	return nil
}

// MarkReloadFailed sets reload_triggered/reload_failed with a CAS on the loaded status.
// Used by the e2e FORCE_RELOAD_FAILED bridge seam.
func (o *Ops) MarkReloadFailed(ctx context.Context, id string) error {
	if o == nil {
		return errors.New("ops: nil host")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("ops: operation id is required")
	}
	row, err := o.loadOperation(ctx, id)
	if err != nil {
		return err
	}
	expect := row.Status
	row.ReloadTriggered = true
	row.ReloadFailed = true
	if err := o.casSaveOperation(ctx, row, expect); err != nil {
		return err
	}
	o.publishOperationChanged(ctx, row)
	return nil
}

// AbandonDraft marks a draft operation cancelled so CLI Plan→Apply failures
// do not leave persistent draft rows. No-op when the row is already non-draft.
func (o *Ops) AbandonDraft(ctx context.Context, id, reason string) error {
	if o == nil {
		return errors.New("ops: nil host")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("ops: operation id is required")
	}
	row, err := o.loadOperation(ctx, id)
	if err != nil {
		return err
	}
	if row.Status != StatusDraft {
		return nil
	}
	now := time.Now().UTC()
	row.Status = StatusCancelled
	row.Phase = PhaseDone
	row.ErrorDomain = "ops"
	row.ErrorCode = "ABANDONED"
	row.ErrorMessage = strings.TrimSpace(reason)
	row.FinishedAt = &now
	if err := o.casSaveOperation(ctx, row, StatusDraft); err != nil {
		return err
	}
	o.clearCatalogActiveOpBestEffort(ctx, row.Id.String)
	o.publishOperationChanged(ctx, row)
	return nil
}
