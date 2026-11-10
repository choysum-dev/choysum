// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package provision

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/ops"
	"github.com/choysum-dev/choysum/internal/module/plan"
	configpkg "github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	xfmt "golang.org/x/exp/errors/fmt"
)

// MinimalRootModule is the InstallModule input for ProvisionMinimal.
// The web shell pulls core/auth/meta via depends; profile=minimal skips document.
const MinimalRootModule = "web"

// MinimalClosureModules is the local ModulesPath set required before automatic
// ProvisionMinimal. Includes web plus the transitive depends closure of the
// minimal web shell (core/auth/meta/base/task) so install does not fall back
// to the registry for missing local packages.
func MinimalClosureModules() []string {
	return []string{MinimalRootModule, "core", "auth", "meta", "base", "task"}
}

// InstallFunc installs one module plan (overridable in tests).
type InstallFunc func(
	ctx context.Context,
	runtimeScope scope.Scope,
	jsExecutor jsexecutor.ScriptExecutor,
	req lifecycle.InstallModuleRequest,
	opts ...lifecycle.Option,
) error

// ExecutorFactory builds the compiler executor used during provision install.
type ExecutorFactory func(scope.Scope) (jsexecutor.JsExecutor, error)

// CatalogSeedFunc syncs source snapshots and materializes meta_module_catalog
// after a successful minimal install (overridable in tests).
type CatalogSeedFunc func(ctx context.Context, runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory) error

// Options configures ProvisionMinimal. Zero values use production defaults.
type Options struct {
	// InstallTimeout bounds the install (and executor) work. Zero uses
	// BootstrapModuleInstallTimeoutSeconds / the package default.
	InstallTimeout time.Duration
	// InstallModule overrides lifecycle.InstallModule (tests).
	InstallModule InstallFunc
	// NewExecutor overrides jsexecutor.NewCompilerExecutor (tests).
	NewExecutor ExecutorFactory
	// LockerFactory overrides the system:init lease locker (tests).
	LockerFactory statepkg.LockerFactory
	// SeedCatalog overrides the post-install catalog materialize (tests).
	SeedCatalog CatalogSeedFunc
}

// Result describes the outcome of ProvisionMinimal.
type Result struct {
	// Skipped is true when the workspace was already past Empty (SetupPending/Ready).
	Skipped bool
	// Phase is the workspace phase after the call.
	Phase Phase
}

// MinimalInstallRequest is the InstallModule payload for ProvisionMinimal and
// shared bootstrap minimal install.
func MinimalInstallRequest() lifecycle.InstallModuleRequest {
	return lifecycle.InstallModuleRequest{
		Input:           MinimalRootModule,
		WithDemo:        false,
		WebShellProfile: plan.WebShellProfileMinimal,
	}
}

// ProvisionMinimal installs the minimal web-shell closure when the workspace is
// Empty. It acquires the shared init lease, never sets an admin password, and
// never writes setup.completed_at — callers must leave the workspace in
// SetupPending for /web/setup.
func ProvisionMinimal(ctx context.Context, runtimeScope scope.Scope, opts Options) (Result, error) {
	if runtimeScope == nil {
		return Result{}, errors.New("scope is not available")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	phase, err := Resolve(runtimeScope)
	if err != nil {
		return Result{}, err
	}
	if phase != PhaseEmpty {
		return Result{Skipped: true, Phase: phase}, nil
	}

	log := runtimeScope.Logger()

	installTimeout := opts.InstallTimeout
	if installTimeout <= 0 {
		installTimeout = moduleInstallTimeout(runtimeScope)
	}
	// Lease renew spans install + catalog seed. Renew failure cancels this
	// context so seed cannot keep writing after the lease is lost. Caller
	// cancellation on ctx still propagates.
	provisionCtx, provisionCancel := context.WithCancel(ctx)
	defer provisionCancel()

	installCtx, installCancel := context.WithTimeout(provisionCtx, installTimeout)
	defer installCancel()

	handle, err := acquireProvisionLease(provisionCtx, runtimeScope, opts.LockerFactory, provisionCancel)
	if err != nil {
		if errors.Is(err, ErrLeaseBusy) {
			return Result{}, xfmt.Errorf("provision lease busy: %w", err)
		}
		return Result{}, xfmt.Errorf("acquire provision lease: %w", err)
	}
	defer releaseProvisionLease(runtimeScope, opts.LockerFactory, handle)

	// Re-check under the lease so a concurrent Initialize that finished between
	// the first Resolve and Acquire does not trigger a redundant install.
	phase, err = Resolve(runtimeScope)
	if err != nil {
		return Result{}, err
	}
	if phase != PhaseEmpty {
		return Result{Skipped: true, Phase: phase}, nil
	}
	if log != nil {
		log.Info("installing minimal modules for first-run setup", "root", MinimalRootModule, "web_shell_profile", string(plan.WebShellProfileMinimal))
	}

	progress := logger.NewProgressLine(os.Stderr)
	installCtx = logger.WithProgressLine(installCtx, progress)
	spinnerTicker := logger.NewProgressTicker(progress, logger.ProgressTickerOptions{Interval: 120 * time.Millisecond})
	defer spinnerTicker.Clear()
	defer spinnerTicker.Stop()
	installCtx = logger.WithProgressTicker(installCtx, spinnerTicker)
	spinnerTicker.SetMessage("installing minimal modules for first-run setup...")

	installScope := runtimeScope.WithContext(installCtx)
	if installScope == nil {
		installScope = runtimeScope
	}

	newExecutor := opts.NewExecutor
	if newExecutor == nil {
		newExecutor = func(s scope.Scope) (jsexecutor.JsExecutor, error) {
			return jsexecutor.NewCompilerExecutor(s)
		}
	}
	executor, err := newExecutor(installScope)
	if err != nil {
		return Result{}, xfmt.Errorf("create provision executor: %w", err)
	}
	if err := executor.Start(); err != nil {
		return Result{}, xfmt.Errorf("start provision executor: %w", err)
	}
	defer executor.Stop()

	installFn := opts.InstallModule
	if installFn == nil {
		installFn = lifecycle.InstallModule
	}
	req := MinimalInstallRequest()
	if err := installFn(installCtx, installScope, executor, req); err != nil {
		if renewErr := checkProvisionLeaseRenew(handle); renewErr != nil {
			// Keep ErrLeaseLost as the primary signal, but retain the install
			// error (often context.Canceled, sometimes an unrelated root cause).
			return Result{}, errors.Join(
				ErrLeaseLost,
				xfmt.Errorf("during install renew: %v", renewErr),
				xfmt.Errorf("provision install %s: %w", req.Input, err),
			)
		}
		return Result{}, xfmt.Errorf("provision install %s: %w", req.Input, err)
	}

	// First-run install goes through lifecycle.InstallModule, not ops.Run, so the
	// post-meta catalog rewrite in ops never runs. Seed local+registry snapshots
	// and materialize here so /web/meta/modules is not empty until a manual sync.
	// Soft-fail: install already succeeded (same policy as ops after meta install).
	// Fresh timeout under provisionCtx: install deadline must not starve seed, and
	// renew/caller cancel still abort seed via provisionCtx.
	seedCtx, seedCancel := context.WithTimeout(provisionCtx, moduleInstallTimeout(runtimeScope))
	defer seedCancel()
	seedScope := runtimeScope.WithContext(seedCtx)
	if seedScope == nil {
		seedScope = runtimeScope
	}
	seedCatalogAfterMinimalInstall(seedCtx, seedScope, opts, log)

	if err := checkProvisionLeaseRenew(handle); err != nil {
		return Result{}, errors.Join(ErrLeaseLost, err)
	}

	phase, err = Resolve(runtimeScope)
	if err != nil {
		return Result{}, err
	}
	if phase == PhaseEmpty {
		return Result{}, errors.New("provision completed but workspace is still empty (auth not installed)")
	}
	if phase == PhaseReady {
		// First-run provision never writes setup.completed_at. Ready here means the
		// workspace already had setup.completed_at once auth appeared (e.g. peer
		// completed setup under the lease, or a prior Ready marker).
		if log != nil {
			log.Info("minimal install finished; workspace already marked ready", "root", MinimalRootModule)
		}
		return Result{Skipped: false, Phase: phase}, nil
	}

	if log != nil {
		log.Info("minimal install finished; waiting for initial setup", "phase", string(phase), "root", MinimalRootModule)
	}
	return Result{Skipped: false, Phase: phase}, nil
}

func seedCatalogAfterMinimalInstall(ctx context.Context, runtimeScope scope.Scope, opts Options, log *slog.Logger) {
	lockerFactory := opts.LockerFactory
	if lockerFactory == nil {
		lockerFactory = defaultLockerFactory
	}
	seed := opts.SeedCatalog
	if seed == nil {
		seed = defaultSeedCatalogAfterMinimalInstall
	}
	if err := seed(ctx, runtimeScope, lockerFactory); err != nil && log != nil {
		log.Warn("provision: catalog seed after minimal install failed", "error", err)
	}
}

func defaultSeedCatalogAfterMinimalInstall(ctx context.Context, runtimeScope scope.Scope, lockerFactory statepkg.LockerFactory) error {
	host := &ops.Ops{
		Scope:         runtimeScope,
		LockerFactory: lockerFactory,
	}
	_, err := host.SyncSourceSnapshotsThenMaterialize(ctx, "all")
	return err
}

type bootstrapModuleInstallTimeoutInput interface {
	BootstrapModuleInstallTimeoutSeconds() int
}

func moduleInstallTimeout(runtimeScope scope.Scope) time.Duration {
	seconds := configpkg.DefaultBootstrapModuleInstallTimeoutSeconds
	if runtimeScope != nil {
		if input := scope.FactoryInputFromScope(runtimeScope); input != nil {
			if timeoutInput, ok := input.(bootstrapModuleInstallTimeoutInput); ok && timeoutInput.BootstrapModuleInstallTimeoutSeconds() > 0 {
				seconds = timeoutInput.BootstrapModuleInstallTimeoutSeconds()
			}
		}
	}
	return time.Duration(seconds) * time.Second
}
