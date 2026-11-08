// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/module/plan"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

// Plan builds a draft meta_module_operation from plan.BuildPlan (or sync stub).
func (o *Ops) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	if o == nil {
		return PlanResult{}, errors.New("ops: nil host")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	action := strings.TrimSpace(strings.ToLower(req.Action))
	moduleName := strings.TrimSpace(req.ModuleName)
	origin := normalizeOrigin(req.Origin)

	switch action {
	case ActionInstall, ActionUpgrade, ActionUninstall:
		if moduleName == "" {
			return PlanResult{}, errors.New("ops: moduleName is required")
		}
		if o.Resolver == nil {
			return PlanResult{}, errors.New("ops: resolver is required")
		}
		return o.planModuleOp(ctx, action, moduleName, origin, req)
	case ActionSync:
		return o.planSyncOp(ctx, strings.TrimSpace(req.OperatorUserId))
	case ActionLoadDemo:
		return o.planLoadDemo(ctx, strings.TrimSpace(req.OperatorUserId))
	default:
		return PlanResult{}, fmt.Errorf("ops: unsupported action %q", req.Action)
	}
}

func normalizeOrigin(origin string) string {
	switch strings.TrimSpace(strings.ToLower(origin)) {
	case originRegistry:
		return originRegistry
	default:
		return originLocal
	}
}

func (o *Ops) planSyncOp(ctx context.Context, operatorUserId string) (PlanResult, error) {
	doc := PlanDocument{
		ModuleOrder:  nil,
		BaseRevision: "sync",
	}
	raw, err := marshalPlanDocument(doc)
	if err != nil {
		return PlanResult{}, err
	}
	row := &modmeta.ModuleOperation{
		Action:         ActionSync,
		ModuleName:     "",
		PlanJson:       raw,
		Status:         StatusDraft,
		OperatorUserId: operatorUserId,
	}
	if err := o.createOperation(ctx, row); err != nil {
		return PlanResult{}, err
	}
	return PlanResult{
		Id:           row.Id.String,
		Action:       ActionSync,
		BaseRevision: doc.BaseRevision,
	}, nil
}

func (o *Ops) planModuleOp(ctx context.Context, action, moduleName, origin string, req PlanRequest) (PlanResult, error) {
	root, blockers, err := o.resolveRootModule(ctx, moduleName, origin)
	if err != nil {
		return PlanResult{}, err
	}

	profile := strings.TrimSpace(req.WebShellProfile)
	if profile != "" {
		parsedProfile, profileErr := plan.ParseWebShellProfile(profile)
		if profileErr != nil {
			return PlanResult{}, profileErr
		}
		profile = string(parsedProfile)
	}
	var built plan.Plan
	if root != nil {
		opType, ok := toPlanOpType(action)
		if !ok {
			return PlanResult{}, fmt.Errorf("ops: unsupported action %q", action)
		}
		buildOpts := []plan.BuildOption{plan.WithSkipWebShell(req.SkipWebShell)}
		if profile != "" {
			buildOpts = append(buildOpts, plan.WithWebShellProfile(plan.WebShellProfile(profile)))
		}
		built, err = plan.BuildPlan(ctx, opType, root, o.Resolver, buildOpts...)
		if err != nil {
			return PlanResult{}, err
		}
	}

	doc := o.enrichPlanDocument(ctx, action, moduleName, origin, built, blockers)
	doc.SkipWebShell = req.SkipWebShell
	doc.WebShellProfile = profile
	doc.LifecycleInput = strings.TrimSpace(req.LifecycleInput)
	raw, err := marshalPlanDocument(doc)
	if err != nil {
		return PlanResult{}, err
	}
	row := &modmeta.ModuleOperation{
		Action:         action,
		ModuleName:     moduleName,
		PlanJson:       raw,
		Status:         StatusDraft,
		OperatorUserId: strings.TrimSpace(req.OperatorUserId),
	}
	if err := o.createOperation(ctx, row); err != nil {
		return PlanResult{}, err
	}
	return PlanResult{
		Id:           row.Id.String,
		Action:       action,
		ModuleName:   moduleName,
		ModuleOrder:  doc.ModuleOrder,
		AffectedApps: doc.AffectedApps,
		Risks:        doc.Risks,
		Blockers:     doc.Blockers,
		BaseRevision: doc.BaseRevision,
	}, nil
}

func toPlanOpType(action string) (plan.OpType, bool) {
	switch action {
	case ActionInstall:
		return plan.OpInstall, true
	case ActionUpgrade:
		return plan.OpUpgrade, true
	case ActionUninstall:
		return plan.OpUninstall, true
	default:
		return "", false
	}
}

func (o *Ops) resolveRootModule(ctx context.Context, moduleName, origin string) (*pkgmeta.Module, []PlanIssue, error) {
	_ = origin // origin pins PlanJson; resolution uses Peek then Load for installed modules.
	var blockers []PlanIssue
	mod, err := o.Resolver.Peek(ctx, moduleName)
	if err != nil || mod == nil {
		loaded, loadErr := o.Resolver.Load(moduleName)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		if loaded == nil {
			blockers = append(blockers, PlanIssue{
				Code:    "MODULE_NOT_FOUND",
				Level:   "error",
				Message: fmt.Sprintf("module %q is not available", moduleName),
			})
			return nil, blockers, nil
		}
		return loaded, blockers, nil
	}
	return mod, blockers, nil
}

func (o *Ops) enrichPlanDocument(ctx context.Context, action, rootName, origin string, built plan.Plan, blockers []PlanIssue) PlanDocument {
	orderNames := append([]string{}, built.ModuleOrder...)
	if len(orderNames) == 0 && rootName != "" && len(blockers) == 0 {
		orderNames = []string{rootName}
	}

	entries := make([]ModuleOrderEntry, 0, len(orderNames))
	for _, name := range orderNames {
		current, target := o.versionsForModule(ctx, name, origin)
		reason := action
		if name == rootName {
			reason = "root"
		} else if action == ActionInstall {
			reason = "dependency"
		}
		entries = append(entries, ModuleOrderEntry{
			ModuleName:     name,
			Reason:         reason,
			CurrentVersion: current,
			TargetVersion:  target,
			Origin:         origin,
		})
	}

	if action == ActionUninstall {
		blockers = append(blockers, o.uninstallBlockers(ctx, rootName, orderNames)...)
	}

	doc := PlanDocument{
		ModuleOrder:   entries,
		AffectedApps:  append([]string{}, built.AffectedApps...),
		Blockers:      blockers,
		EnsureOrder:   append([]string{}, built.EnsureOrder...),
		NeedsWebBuild: built.NeedsGlobalWebBuild,
	}
	doc.BaseRevision = computeBaseRevision(ctx, o, entries)
	return doc
}

func (o *Ops) versionsForModule(ctx context.Context, name, origin string) (current, target string) {
	if installed, err := o.Resolver.Load(name); err == nil && installed != nil {
		current = strings.TrimSpace(installed.Version)
	}
	localV, registryV := o.snapshotVersions(ctx, name)
	switch origin {
	case originRegistry:
		target = firstNonEmpty(registryV, localV, current)
	default:
		target = firstNonEmpty(localV, registryV, current)
	}
	if target == "" {
		if peeked, err := o.Resolver.Peek(ctx, name); err == nil && peeked != nil {
			target = strings.TrimSpace(peeked.Version)
		}
	}
	return current, target
}

func (o *Ops) snapshotVersions(ctx context.Context, name string) (localV, registryV string) {
	sess, err := o.session(ctx)
	if err != nil {
		return "", ""
	}
	var rows []modmeta.ModuleSourceSnapshot
	_ = sess.WithContext(ctx).
		Where("module_name = ? AND available = ?", name, true).
		Order("source_type ASC, source_ref ASC").
		Find(&rows).Error
	for _, row := range rows {
		v := ""
		if row.Version.Valid {
			v = strings.TrimSpace(row.Version.String)
		}
		switch row.SourceType {
		case originLocal:
			if localV == "" {
				localV = v
			}
		case originRegistry:
			if registryV == "" {
				registryV = v
			}
		}
	}
	return localV, registryV
}

func (o *Ops) uninstallBlockers(ctx context.Context, rootName string, planned []string) []PlanIssue {
	plannedSet := make(map[string]struct{}, len(planned))
	for _, n := range planned {
		plannedSet[n] = struct{}{}
	}
	root, err := o.Resolver.Load(rootName)
	if err != nil {
		return []PlanIssue{{
			Code:    "DEPENDENCY_LOOKUP_FAILED",
			Level:   "error",
			Message: fmt.Sprintf("could not check dependents of module %q: %v", rootName, err),
		}}
	}
	if root == nil {
		return nil
	}
	var out []PlanIssue
	for _, dep := range root.Dependents {
		if dep == nil {
			continue
		}
		name := strings.TrimSpace(dep.Name)
		if name == "" {
			continue
		}
		if _, ok := plannedSet[name]; ok {
			continue
		}
		if dep.Status != pkgmeta.Installed {
			continue
		}
		out = append(out, PlanIssue{
			Code:    "DEPENDENT_INSTALLED",
			Level:   "error",
			Message: fmt.Sprintf("module %q is still depended on by installed module %q", rootName, name),
		})
	}
	return out
}

func computeBaseRevision(ctx context.Context, o *Ops, entries []ModuleOrderEntry) string {
	// Hash mixes frozen PlanJson target/origin with live installed + snapshot
	// versions so Apply rejects when the world moved since Plan.
	type revPart struct {
		Name     string `json:"n"`
		Current  string `json:"c"`
		Target   string `json:"t"`
		Origin   string `json:"o"`
		Local    string `json:"l,omitempty"`
		Registry string `json:"r,omitempty"`
	}
	parts := make([]revPart, 0, len(entries))
	for _, e := range entries {
		localV, registryV := "", ""
		current := e.CurrentVersion
		if o != nil {
			localV, registryV = o.snapshotVersions(ctx, e.ModuleName)
			if o.Resolver != nil {
				if installed, err := o.Resolver.Load(e.ModuleName); err == nil {
					if installed != nil {
						current = strings.TrimSpace(installed.Version)
					} else {
						current = ""
					}
				}
			}
		}
		parts = append(parts, revPart{
			Name:     e.ModuleName,
			Current:  current,
			Target:   e.TargetVersion,
			Origin:   e.Origin,
			Local:    localV,
			Registry: registryV,
		})
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].Name < parts[j].Name })
	raw, _ := json.Marshal(parts)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
