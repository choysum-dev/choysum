// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import "strings"

// Session holds per-build inject plans and paths for all registered Specs.
type Session struct {
	host           Host
	plans          map[string]Plan
	injectPaths    map[string][]string
	lastInjectPath map[string]string
}

// NewSession creates a Session bound to host.
func NewSession(host Host) *Session {
	return &Session{
		host:           host,
		plans:          make(map[string]Plan),
		injectPaths:    make(map[string][]string),
		lastInjectPath: make(map[string]string),
	}
}

// ReleaseSchedules clears all scheduled apps claimed by this session.
func (s *Session) ReleaseSchedules() {
	if s == nil {
		return
	}
	for modelName, plan := range s.plans {
		app := strings.TrimSpace(plan.ScheduledApp)
		if app == "" {
			continue
		}
		if spec, ok := specByName(modelName); ok && spec.scheduled != nil {
			spec.scheduled.Delete(app)
		}
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

func (s *Session) allInjectPaths() []string {
	return s.AllInjectPaths()
}

// AllInjectPaths returns inject paths for every registered Spec (registration order).
func (s *Session) AllInjectPaths() []string {
	if s == nil {
		return nil
	}
	var out []string
	for _, spec := range specsList() {
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
	if spec.scheduled != nil {
		spec.scheduled.Delete(app)
	}
	plan.ScheduledApp = ""
	s.SetPlan(spec.ModelName, plan)
}
