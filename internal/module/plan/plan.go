// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

type OpType string

const (
	OpInstall   OpType = "install"
	OpUninstall OpType = "uninstall"
	OpUpgrade   OpType = "upgrade"
)

type Plan struct {
	Op OpType

	// ModuleOrder is the planned module execution order.
	// For install: topo order; for uninstall: reverse topo.
	ModuleOrder []string

	// EnsureOrder lists modules that must be installed (if missing) before the
	// primary ModuleOrder runs. Used by upgrade to pull in the web shell without
	// upgrading web itself.
	EnsureOrder []string

	// AffectedApps contains application names impacted by the operation.
	AffectedApps []string

	// NeedsGlobalWebBuild indicates whether global web assets build should be triggered.
	NeedsGlobalWebBuild bool
}
