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
	"github.com/choysum-dev/choysum/pkg/oerrors"
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
	if row.Action == ActionLoadDemo {
		msg := "load-demo is not implemented until W4"
		runErr := oerrors.New("ops", "LOAD_DEMO_NOT_IMPLEMENTED", msg)
		if saveErr := o.failOperation(context.WithoutCancel(ctx), row, "ops", "LOAD_DEMO_NOT_IMPLEMENTED", msg); saveErr != nil {
			return errors.Join(runErr, saveErr)
		}
		return runErr
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
	o.publishOperationChanged(ctx, row)

	runErr := o.executeOperation(ctx, row, req.WithDemo)
	finishCtx := context.WithoutCancel(ctx)
	if runErr != nil {
		if saveErr := o.failOperation(finishCtx, row, "ops", "RUN_FAILED", runErr.Error()); saveErr != nil {
			return errors.Join(runErr, saveErr)
		}
		return runErr
	}
	return o.succeedOperation(finishCtx, row)
}

func (o *Ops) executeOperation(ctx context.Context, row *modmeta.ModuleOperation, requestWithDemo bool) error {
	switch row.Action {
	case ActionSync:
		_, err := o.SyncSourceSnapshotsThenMaterialize(ctx, "all")
		return err
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
		doc, err := unmarshalPlanDocument(row.PlanJson)
		if err != nil {
			return err
		}
		names := affectedModuleNames(doc, row.ModuleName)
		if err := o.runLifecycle(ctx, row.Action, row.ModuleName, withDemo); err != nil {
			return err
		}
		if err := o.MaterializeCatalog(ctx, names); err != nil {
			return fmt.Errorf("ops: materialize catalog after %s: %w", row.Action, err)
		}
		return nil
	default:
		return fmt.Errorf("ops: unsupported action %q", row.Action)
	}
}

func (o *Ops) runLifecycle(ctx context.Context, action, moduleName string, withDemo bool) error {
	name := strings.TrimSpace(moduleName)
	switch action {
	case ActionInstall:
		return o.Lifecycle.Install(ctx, lifecycle.InstallRequest{Name: name, WithDemo: withDemo})
	case ActionUpgrade:
		return o.Lifecycle.Upgrade(ctx, lifecycle.UpgradeRequest{Input: name, WithDemo: withDemo})
	case ActionUninstall:
		return o.Lifecycle.Uninstall(ctx, lifecycle.UninstallRequest{Name: name})
	default:
		return fmt.Errorf("ops: unsupported lifecycle action %q", action)
	}
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
	o.publishOperationChanged(ctx, row)
	return nil
}

func (o *Ops) succeedOperation(ctx context.Context, row *modmeta.ModuleOperation) error {
	expect := row.Status
	now := time.Now().UTC()
	row.Status = StatusSucceeded
	row.Phase = PhaseDone
	row.FinishedAt = &now
	row.ErrorDomain = ""
	row.ErrorCode = ""
	row.ErrorMessage = ""
	if err := o.casSaveOperation(ctx, row, expect); err != nil {
		return err
	}
	o.publishOperationChanged(ctx, row)
	return nil
}
