// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"strings"

	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	"github.com/choysum-dev/choysum/pkg/meta"
	xfmt "golang.org/x/exp/errors/fmt"
)

// OriginCoordinator defines the origin resolution contract used by ModuleManager.
// It is exposed so tests and callers can inject deterministic origin behaviors via options.
type OriginCoordinator = internalorigin.Service

func (m *ModuleManager) newOriginCoordinator() OriginCoordinator {
	if m.originCoordinatorFactory != nil {
		if coordinator := m.originCoordinatorFactory(m.runtimeScope); coordinator != nil {
			return coordinator
		}
	}
	return internalorigin.NewCoordinator(m.runtimeScope)
}

func (m *ModuleManager) resolveUpgradeModuleFromOrigin(ctx context.Context, name string) (*meta.Module, error) {
	if err := m.ensureMetaTables(); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, xfmt.Errorf("module name is empty")
	}

	coordinator := m.newOriginCoordinator()
	module, err := coordinator.Peek(ctx, name)
	if err != nil {
		return nil, xfmt.Errorf("error resolving module %s from origin: %w", name, err)
	}
	if module == nil {
		return nil, xfmt.Errorf("module %s not found in origin", name)
	}

	if meta.IsCoreModule(module.Name) {
		if err := m.migrateBaseModule(); err != nil {
			return nil, err
		}
	}

	return module, nil
}

func (m *ModuleManager) prepareUpgradeOriginSwitch(ctx context.Context, parsed internalorigin.ParsedInput, moduleName string, opid string) (*internalorigin.UpgradeSwitchSnapshot, error) {
	runtimeOpts := m.resolvedRuntimeOptions()
	return internalorigin.PrepareUpgradeSwitch(
		ctx,
		m.newOriginCoordinator(),
		internalorigin.WorkspaceRoot(m.runtimeScope),
		runtimeOpts.modulesPath,
		runtimeOpts.tmpPath,
		runtimeOpts.defaultChoysumPath,
		parsed,
		moduleName,
		opid,
	)
}
