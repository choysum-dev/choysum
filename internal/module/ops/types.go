// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

// PlanIssue is a blocker or risk recorded in PlanJson.
type PlanIssue struct {
	Code    string `json:"code"`
	Level   string `json:"level"`
	Message string `json:"message,omitempty"`
}

// ModuleOrderEntry is one frozen step in PlanJson.moduleOrder.
type ModuleOrderEntry struct {
	ModuleName     string `json:"moduleName"`
	Reason         string `json:"reason,omitempty"`
	CurrentVersion string `json:"currentVersion,omitempty"`
	TargetVersion  string `json:"targetVersion,omitempty"`
	Origin         string `json:"origin,omitempty"`
}

// PlanDocument is the frozen plan body stored in meta_module_operation.plan_json.
type PlanDocument struct {
	ModuleOrder   []ModuleOrderEntry `json:"moduleOrder"`
	AffectedApps  []string           `json:"affectedApps,omitempty"`
	Risks         []PlanIssue        `json:"risks,omitempty"`
	Blockers      []PlanIssue        `json:"blockers,omitempty"`
	BaseRevision  string             `json:"baseRevision"`
	EnsureOrder   []string           `json:"ensureOrder,omitempty"`
	NeedsWebBuild bool               `json:"needsGlobalWebBuild,omitempty"`
	// CLI / bridge options frozen at Plan time for deterministic Run.
	SkipWebShell    bool   `json:"skipWebShell,omitempty"`
	WebShellProfile string `json:"webShellProfile,omitempty"`
	LifecycleInput  string `json:"lifecycleInput,omitempty"` // upgrade/install input; empty => ModuleName
}

// PlanRequest creates a draft operation.
type PlanRequest struct {
	ModuleName      string
	Action          string
	Origin          string // optional local|registry; empty means local-first
	OperatorUserId  string
	SkipWebShell    bool
	WebShellProfile string
	LifecycleInput  string // optional; empty => ModuleName (upgrade may pass name@version)
}

// PlanResult is returned to callers after a draft row is written.
type PlanResult struct {
	Id           string
	Action       string
	ModuleName   string
	ModuleOrder  []ModuleOrderEntry
	AffectedApps []string
	Risks        []PlanIssue
	Blockers     []PlanIssue
	BaseRevision string
}

// ApplyRequest moves a draft operation to queued after revision checks.
type ApplyRequest struct {
	Id string
}

// RunRequest executes a queued (or draft for CLI sync path) operation.
type RunRequest struct {
	Id       string
	WithDemo bool // when true, forces WithDemo for this run and sets modules.with_demo
}
