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

func writeUnit(ctx context.Context, txScope scope.Scope, spec importpkg.Spec, writer registry.Writer, unit plan.Unit) error {
	if err := writer.Write(ctx, txScope, []plan.Unit{unit}); err != nil {
		return err
	}
	if spec.DryRun {
		return errDryRunRollback
	}
	return nil
}

func isDryRunRollback(err error) bool {
	return errors.Is(err, errDryRunRollback)
}

func runAtomic(ctx context.Context, runtimeScope scope.Scope, spec importpkg.Spec, p plan.Plan, writer registry.Writer) (importpkg.Report, error) {
	collector := newMessageCollector(p.Len())

	atomicFn := func(txScope scope.Scope, _ scope.Transaction) error {
		for _, unit := range p.Units {
			unitErr := writer.Write(txScope.Context(), txScope, []plan.Unit{unit})
			if unitErr != nil {
				collector.addError(unitErr, unit)
				continue
			}
			collector.addOK(1)
		}
		if collector.hasHardError() {
			return collector.firstErr
		}
		if spec.DryRun {
			return errDryRunRollback
		}
		return nil
	}

	// Prefer Nested so joined outer txs (BE unit harness) roll back via savepoint
	// without markRollback on the outer Required (dry-run / hard errors).
	// Lifecycle scopes often use runSessionTransactor, which returns ErrNestedUnsupported —
	// fall back to Required (fresh or joined) for those callers.
	err := runtimeScope.Transactor().Nested(ctx, atomicFn)
	if errors.Is(err, scope.ErrNestedUnsupported) {
		err = runtimeScope.Transactor().Required(ctx, atomicFn)
	}

	report := collector.buildReport(spec, p.Len())

	if spec.DryRun && errors.Is(err, errDryRunRollback) {
		return report, nil
	}
	if err != nil && !errors.Is(err, errDryRunRollback) {
		return report, err
	}
	return report, nil
}

func runStopKeep(ctx context.Context, runtimeScope scope.Scope, spec importpkg.Spec, p plan.Plan, writer registry.Writer) (importpkg.Report, error) {
	collector := newMessageCollector(p.Len())
	for i, unit := range p.Units {
		unitErr := runtimeScope.Transactor().RequiresNew(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
			return writeUnit(txScope.Context(), txScope, spec, writer, unit)
		})
		if unitErr != nil {
			if isDryRunRollback(unitErr) {
				collector.addOK(1)
				continue
			}
			collector.addError(unitErr, unit)
			collector.addSkip(len(p.Units) - i - 1)
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
			return writeUnit(txScope.Context(), txScope, spec, writer, unit)
		})
		if unitErr != nil {
			if isDryRunRollback(unitErr) {
				collector.addOK(1)
				continue
			}
			collector.addError(unitErr, unit)
			continue
		}
		collector.addOK(1)
	}
	report := collector.buildReport(spec, p.Len())
	return report, nil
}
