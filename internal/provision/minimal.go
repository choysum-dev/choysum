// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package provision

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/plan"
	configpkg "github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/scope"
	statepkg "github.com/choysum-dev/choysum/pkg/state"
	xfmt "golang.org/x/exp/errors/fmt"
)

// MinimalRootModule is the InstallModule input for host first-run provision.
// The web shell pulls core/auth/meta via depends; profile=minimal skips document.
const MinimalRootModule = "web"

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
}

// Result describes the outcome of ProvisionMinimal.
type Result struct {
	// Skipped is true when the workspace was already past Empty (SetupPending/Ready).
	Skipped bool
	// Phase is the workspace phase after the call.
	Phase Phase
}

// MinimalInstallRequest is the InstallModule payload for host provision and
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
	if log != nil {
		log.Info("provision started", "root", MinimalRootModule, "web_shell_profile", string(plan.WebShellProfileMinimal))
	}

	installTimeout := opts.InstallTimeout
	if installTimeout <= 0 {
		installTimeout = moduleInstallTimeout(runtimeScope)
	}
	installCtx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()

	handle, err := acquireProvisionLease(installCtx, runtimeScope, opts.LockerFactory, cancel)
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

	progress := logger.NewProgressLine(os.Stderr)
	installCtx = logger.WithProgressLine(installCtx, progress)
	spinnerTicker := logger.NewProgressTicker(progress, logger.ProgressTickerOptions{Interval: 120 * time.Millisecond})
	defer spinnerTicker.Clear()
	defer spinnerTicker.Stop()
	installCtx = logger.WithProgressTicker(installCtx, spinnerTicker)
	spinnerTicker.SetMessage("provision: installing minimal web shell...")

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
		if progress != nil {
			progress.Done("✗", "provision module installation failed")
		}
		if renewErr := checkProvisionLeaseRenew(handle); renewErr != nil {
			return Result{}, xfmt.Errorf("provision lease lost during install: %w", renewErr)
		}
		return Result{}, xfmt.Errorf("provision install %s: %w", req.Input, err)
	}
	if progress != nil {
		progress.Done("✓", "provision module installation completed")
	}

	if err := checkProvisionLeaseRenew(handle); err != nil {
		return Result{}, xfmt.Errorf("provision lease lost: %w", err)
	}

	phase, err = Resolve(runtimeScope)
	if err != nil {
		return Result{}, err
	}
	if phase == PhaseEmpty {
		return Result{}, errors.New("provision completed but workspace is still empty (auth not installed)")
	}
	if phase == PhaseReady {
		// Host provision never writes setup.completed_at. Ready here means a
		// legacy system.init.done backfill kicked in once auth appeared.
		if log != nil {
			log.Info("provision completed with legacy Ready marker", "root", MinimalRootModule)
		}
		return Result{Skipped: false, Phase: phase}, nil
	}

	if log != nil {
		log.Info("provision completed", "phase", string(phase), "root", MinimalRootModule)
	}
	return Result{Skipped: false, Phase: phase}, nil
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
