// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"context"
	"errors"

	"github.com/choysum-dev/choysum/internal/import/plan"
	"github.com/choysum-dev/choysum/internal/import/registry"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

var errDryRunRollback = errors.New("import: dry run rollback")

func executePlan(ctx context.Context, runtimeScope scope.Scope, spec importpkg.Spec, p plan.Plan, writer registry.Writer) (importpkg.Report, error) {
	if runtimeScope == nil {
		return importpkg.Report{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "scope is required")
	}
	policy := importpkg.EffectivePolicy(spec)
	switch policy {
	case importpkg.PolicyAtomic:
		return runAtomic(ctx, runtimeScope, spec, p, writer)
	case importpkg.PolicyStopKeep:
		return runStopKeep(ctx, runtimeScope, spec, p, writer)
	case importpkg.PolicyBestEffort:
		return runBestEffort(ctx, runtimeScope, spec, p, writer)
	default:
		return importpkg.Report{}, importpkg.ErrPolicyDenied
	}
}

func runAtomic(ctx context.Context, runtimeScope scope.Scope, spec importpkg.Spec, p plan.Plan, writer registry.Writer) (importpkg.Report, error) {
	collector := newMessageCollector(p.Len())
	var runErr error

	err := runtimeScope.Transactor().Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
		for _, unit := range p.Units {
			unitErr := txScope.Transactor().Nested(txScope.Context(), func(nestedScope scope.Scope, _ scope.Transaction) error {
				return writer.Write(nestedScope.Context(), nestedScope, []plan.Unit{unit})
			})
			if unitErr != nil {
				collector.addError(unitErr, unit)
				continue
			}
			collector.addOK(1)
		}
		if spec.DryRun {
			return errDryRunRollback
		}
		if collector.hasHardError() {
			return collector.firstErr
		}
		return nil
	})

	report := collector.buildReport(spec, p.Len())

	if spec.DryRun && errors.Is(err, errDryRunRollback) {
		return report, nil
	}
	if err != nil && !errors.Is(err, errDryRunRollback) {
		runErr = err
	}
	if collector.hasHardError() && runErr == nil {
		runErr = collector.firstErr
	}
	return report, runErr
}

func runStopKeep(ctx context.Context, runtimeScope scope.Scope, spec importpkg.Spec, p plan.Plan, writer registry.Writer) (importpkg.Report, error) {
	collector := newMessageCollector(p.Len())
	for _, unit := range p.Units {
		unitErr := runtimeScope.Transactor().RequiresNew(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
			return writer.Write(txScope.Context(), txScope, []plan.Unit{unit})
		})
		if unitErr != nil {
			collector.addError(unitErr, unit)
			break
		}
		collector.addOK(1)
	}
	report := collector.buildReport(spec, p.Len())
	if collector.hasHardError() {
		return report, collector.firstErr
	}
	return report, nil
}

func runBestEffort(ctx context.Context, runtimeScope scope.Scope, spec importpkg.Spec, p plan.Plan, writer registry.Writer) (importpkg.Report, error) {
	collector := newMessageCollector(p.Len())
	for _, unit := range p.Units {
		unitErr := runtimeScope.Transactor().RequiresNew(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
			return writer.Write(txScope.Context(), txScope, []plan.Unit{unit})
		})
		if unitErr != nil {
			collector.addError(unitErr, unit)
			continue
		}
		collector.addOK(1)
	}
	report := collector.buildReport(spec, p.Len())
	return report, nil
}
