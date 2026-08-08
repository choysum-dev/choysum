// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import "strings"

// Session holds per-build inject plans and paths for Specs in a Registry.
type Session struct {
	ctx            BuildCtx
	reg            *Registry
	plans          map[string]Plan
	injectPaths    map[string][]string
	lastInjectPath map[string]string

	// ensuredServiceEntry tracks an in-memory Module.ServiceEntryPoint mutation
	// from Ensure so failed inject can restore the prior value.
	ensuredServiceEntry bool
	priorServiceEntry   string
	// ensuredVirtual is true when Ensure synthesized a virtual service/index.ts
	// because package.json had no entryPoints.service. Sibling Specs without
	// EnsureServiceEntry must keep treating the declared entry as empty.
	ensuredVirtual bool
}

// NewSession creates a Session bound to ctx and reg.
// If reg is nil, DefaultRegistry() is used.
func NewSession(ctx BuildCtx, reg *Registry) *Session {
	if reg == nil {
		reg = DefaultRegistry()
	}
	return &Session{
		ctx:            ctx,
		reg:            reg,
		plans:          make(map[string]Plan),
		injectPaths:    make(map[string][]string),
		lastInjectPath: make(map[string]string),
	}
}

// Context returns a pointer to the session BuildCtx (tests may mutate fields).
func (s *Session) Context() *BuildCtx {
	if s == nil {
		return nil
	}
	return &s.ctx
}

// Registry returns the Spec/claim registry for this session.
func (s *Session) Registry() *Registry {
	if s == nil {
		return nil
	}
	if s.reg == nil {
		return DefaultRegistry()
	}
	return s.reg
}

// ReleaseSchedules clears all scheduled apps claimed by this session.
func (s *Session) ReleaseSchedules() {
	if s == nil {
		return
	}
	reg := s.Registry()
	for modelName, plan := range s.plans {
		app := strings.TrimSpace(plan.ScheduledApp)
		if app == "" {
			continue
		}
		reg.ReleaseClaim(modelName, app)
		plan.ScheduledApp = ""
		s.SetPlan(modelName, plan)
	}
}

// SetPlan stores a plan for modelName (tests / legacy adapters).
func (s *Session) SetPlan(modelName string, plan Plan) {
	if s == nil {
		return
	}
	if s.plans == nil {
		s.plans = make(map[string]Plan)
	}
	s.plans[modelName] = plan
}

// Plan returns the stored plan for modelName.
func (s *Session) Plan(modelName string) Plan {
	if s == nil {
		return Plan{}
	}
	return s.plans[modelName]
}

// InjectPaths returns accumulated inject paths for modelName.
func (s *Session) InjectPaths(modelName string) []string {
	if s == nil {
		return nil
	}
	paths := s.injectPaths[modelName]
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, len(paths))
	copy(out, paths)
	return out
}

// LastInjectPath returns the most recent inject path for modelName (tests).
func (s *Session) LastInjectPath(modelName string) string {
	if s == nil {
		return ""
	}
	return s.lastInjectPath[modelName]
}

func (s *Session) rememberInjectPath(modelName, path string) {
	path = strings.TrimSpace(path)
	if s == nil || path == "" {
		return
	}
	if s.lastInjectPath == nil {
		s.lastInjectPath = make(map[string]string)
	}
	if s.injectPaths == nil {
		s.injectPaths = make(map[string][]string)
	}
	s.lastInjectPath[modelName] = path
	for _, existing := range s.injectPaths[modelName] {
		if existing == path {
			return
		}
	}
	s.injectPaths[modelName] = append(s.injectPaths[modelName], path)
}

// ClearInjectPaths resets remembered inject paths for modelName.
func (s *Session) ClearInjectPaths(modelName string) {
	if s == nil {
		return
	}
	delete(s.injectPaths, modelName)
	delete(s.lastInjectPath, modelName)
}

// ClearAllInjectPaths drops every remembered inject path (failed inject/bundle
// before Effects are applied, so buildOptions cannot import stale generated paths).
// Also reverts an in-memory Ensure of Module.ServiceEntryPoint.
func (s *Session) ClearAllInjectPaths() {
	if s == nil {
		return
	}
	s.injectPaths = nil
	s.lastInjectPath = nil
	s.revertEnsuredServiceEntry()
}

// ensureServiceEntryPath sets Module.ServiceEntryPoint for this build round.
// Remembers the prior value once so ClearAllInjectPaths can restore on failure.
// virtual marks a synthesized entry (no package.json entryPoints.service / no
// on-disk service entry); disk-adopted Ensure leaves virtual=false so sibling
// Specs may treat the entry as declared.
func (s *Session) ensureServiceEntryPath(path string, virtual bool) {
	path = strings.TrimSpace(path)
	if s == nil || path == "" || s.ctx.Module == nil {
		return
	}
	if !s.ensuredServiceEntry {
		s.priorServiceEntry = s.ctx.Module.ServiceEntryPoint
		s.ensuredServiceEntry = true
		s.ensuredVirtual = virtual
	} else if virtual {
		// Once virtual, stay virtual for this build even if a later Ensure
		// rewrites the path (should not unlock FieldDefault / AppSetting).
		s.ensuredVirtual = true
	}
	s.ctx.Module.ServiceEntryPoint = path
}

// declaredServiceEntry returns the package.json / Module service entry that
// Specs without EnsureServiceEntry should honor. A virtual TranslationTerm
// Ensure must not unlock FieldDefault / AppSetting for modules that only have
// entryPoints.web (e.g. modules/web today).
func (s *Session) declaredServiceEntry(spec *Spec) string {
	if s == nil || s.ctx.Module == nil {
		return ""
	}
	entry := strings.TrimSpace(s.ctx.Module.ServiceEntryPoint)
	if spec != nil && !spec.EnsureServiceEntry && s.ensuredServiceEntry && s.ensuredVirtual {
		return strings.TrimSpace(s.priorServiceEntry)
	}
	return entry
}

// revertEnsuredServiceEntry restores Module.ServiceEntryPoint when Ensure mutated
// it for this build. Exported for Persist so DB stays aligned with package.json
// (cold builds always re-Ensure from an empty disk entry).
func (s *Session) RevertEnsuredServiceEntry() {
	s.revertEnsuredServiceEntry()
}

func (s *Session) revertEnsuredServiceEntry() {
	if s == nil || !s.ensuredServiceEntry {
		return
	}
	if s.ctx.Module != nil {
		s.ctx.Module.ServiceEntryPoint = s.priorServiceEntry
	}
	s.ensuredServiceEntry = false
	s.priorServiceEntry = ""
	s.ensuredVirtual = false
}

func (s *Session) allInjectPaths() []string {
	return s.AllInjectPaths()
}

// AllInjectPaths returns inject paths for every Spec in the session registry.
func (s *Session) AllInjectPaths() []string {
	if s == nil {
		return nil
	}
	var out []string
	for _, spec := range s.Registry().specsList() {
		out = append(out, s.injectPaths[spec.ModelName]...)
	}
	return out
}

func (s *Session) releaseScheduleFor(spec *Spec) {
	if s == nil || spec == nil {
		return
	}
	plan := s.plans[spec.ModelName]
	app := strings.TrimSpace(plan.ScheduledApp)
	if app == "" {
		return
	}
	s.Registry().ReleaseClaim(spec.ModelName, app)
	plan.ScheduledApp = ""
	s.SetPlan(spec.ModelName, plan)
}
