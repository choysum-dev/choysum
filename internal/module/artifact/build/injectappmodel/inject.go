// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
	xfmt "golang.org/x/exp/errors/fmt"
)

// InjectAppModels runs Decide + Materialize for every Spec and returns Effects
// for the caller to apply (virtual sources / entry imports).
func InjectAppModels(sess *Session, prebuildResults []*parser.ParserResult) (Effects, error) {
	var out Effects
	if sess == nil {
		return out, nil
	}
	for _, spec := range sess.Registry().specsList() {
		fx, err := DecideAndInjectOne(sess, spec.ModelName, prebuildResults)
		if err != nil {
			return out, err
		}
		out.Files = append(out.Files, fx.Files...)
		out.Imports = mergeUniqueStrings(out.Imports, fx.Imports)
	}
	return out, nil
}

// DecideAndInjectOne runs Decide + Materialize for a single Spec.
func DecideAndInjectOne(sess *Session, modelName string, prebuildResults []*parser.ParserResult) (Effects, error) {
	var out Effects
	if sess == nil {
		return out, nil
	}
	spec, ok := sess.Registry().lookupPtr(modelName)
	if !ok {
		return out, xfmt.Errorf("injectappmodel: unknown Spec %q", modelName)
	}
	plan, err := decidePlan(spec, sess, prebuildResults)
	if err != nil {
		return out, err
	}
	sess.SetPlan(spec.ModelName, plan)
	fx, err := materializeInject(sess, spec, plan)
	if err != nil {
		sess.releaseScheduleFor(spec)
		return out, xfmt.Errorf("error injecting %s: %w", spec.ModelName, err)
	}
	return fx, nil
}

// DecideOne runs Decide for one Spec and stores the plan.
func DecideOne(sess *Session, modelName string, prebuildResults []*parser.ParserResult) (Plan, error) {
	plan := Plan{}
	if sess == nil {
		return plan, nil
	}
	spec, ok := sess.Registry().lookupPtr(modelName)
	if !ok {
		return plan, xfmt.Errorf("injectappmodel: unknown Spec %q", modelName)
	}
	plan, err := decidePlan(spec, sess, prebuildResults)
	if err != nil {
		return plan, err
	}
	sess.SetPlan(spec.ModelName, plan)
	return plan, nil
}

// ApplyInjectOne materializes NeedInject for one Spec using the stored plan.
func ApplyInjectOne(sess *Session, modelName string) (Effects, error) {
	var out Effects
	if sess == nil {
		return out, nil
	}
	spec, ok := sess.Registry().lookupPtr(modelName)
	if !ok {
		return out, xfmt.Errorf("injectappmodel: unknown Spec %q", modelName)
	}
	return materializeInject(sess, spec, sess.Plan(modelName))
}

func decidePlan(spec *Spec, sess *Session, prebuildResults []*parser.ParserResult) (Plan, error) {
	plan := Plan{}
	if spec == nil || sess == nil {
		return plan, nil
	}
	mod := sess.ctx.Module
	if mod == nil {
		return plan, nil
	}
	if strings.TrimSpace(mod.ServiceEntryPoint) == "" ||
		strings.TrimSpace(mod.ApplicationStr) == "" ||
		strings.TrimSpace(mod.ApplicationStr) == "core" {
		return plan, nil
	}

	app := strings.TrimSpace(mod.ApplicationStr)
	local := modelsIn(spec, prebuildResults, mod.Path)
	localHand := handwrittenModels(spec, local)
	if len(localHand) > 1 {
		return plan, xfmt.Errorf("%s: application %q has multiple handwritten %s models in module %q", spec.DuplicateCode, app, spec.ModelName, mod.Name)
	}

	existing, err := dbLoadModels(spec, sess.ctx.DB, app)
	if err != nil {
		return plan, xfmt.Errorf("load %s models for application %q: %w", spec.ModelName, app, err)
	}
	existingVirt := generatedModels(spec, existing)
	existingHand := handwrittenModels(spec, existing)

	if len(localHand) > 0 {
		if len(existingHand) > 0 && !sameModule(existingHand, mod) {
			return plan, xfmt.Errorf("%s: application %q already has a handwritten %s outside module %q", spec.DuplicateCode, app, spec.ModelName, mod.Name)
		}
		if len(existingVirt) > 0 {
			return Plan{SupersedeInject: true}, nil
		}
		return plan, nil
	}

	// No local handwritten model — consider C2 inject.
	if len(existingHand) > 0 {
		return plan, nil
	}
	if len(local) > 0 {
		return plan, nil
	}
	if len(existingVirt) > 0 {
		if sameModule(existingVirt, mod) {
			return claimNeedInject(sess.Registry(), spec, app, mod.Name), nil
		}
		return plan, nil
	}
	return claimFirstNeedInject(sess.Registry(), spec, app, mod.Name), nil
}

func claimNeedInject(reg *Registry, spec *Spec, app, modName string) Plan {
	if spec.ForeignClaimOnOwnerReinject {
		owner, loaded := reg.TryClaim(spec.ModelName, app, modName)
		if loaded && owner != "" && owner != modName {
			return Plan{NeedInject: true}
		}
		return Plan{NeedInject: true, ScheduledApp: app}
	}
	return Plan{NeedInject: true, ScheduledApp: app}
}

func claimFirstNeedInject(reg *Registry, spec *Spec, app, modName string) Plan {
	owner, loaded := reg.TryClaim(spec.ModelName, app, modName)
	if loaded {
		if owner == modName {
			return Plan{NeedInject: true, ScheduledApp: app}
		}
		return Plan{}
	}
	return Plan{NeedInject: true, ScheduledApp: app}
}

// materializeInject builds Effects for a NeedInject plan (no build side effects).
func materializeInject(sess *Session, spec *Spec, plan Plan) (Effects, error) {
	var out Effects
	if sess == nil || spec == nil || !plan.NeedInject {
		return out, nil
	}
	mod := sess.ctx.Module
	if mod == nil {
		return out, nil
	}
	if strings.TrimSpace(mod.Path) == "" {
		return out, xfmt.Errorf("%s inject requires a non-empty module path", spec.ModelName)
	}
	path := generatedPath(spec, mod.Path)
	sess.rememberInjectPath(spec.ModelName, path)

	modulesPath := strings.TrimSpace(sess.ctx.ModulesPath)
	if modulesPath == "" {
		modulesPath = filepath.Dir(mod.Path)
	}
	source := generatedSource(spec, modulesPath, mod.ApplicationStr)
	out.Files = []VirtualFile{{Path: path, Contents: source}}
	out.Imports = []string{path}
	return out, nil
}

// ValidateInjectAppModels checks build output for duplicate inject models per application.
func ValidateInjectAppModels(sess *Session, buildResults []*parser.ParserResult) error {
	if sess == nil {
		return nil
	}
	mod := sess.ctx.Module
	if mod == nil {
		return nil
	}
	app := strings.TrimSpace(mod.ApplicationStr)
	if app == "" || app == "core" {
		return nil
	}
	for _, spec := range sess.Registry().specsList() {
		models := modelsIn(spec, buildResults, mod.Path)
		if len(models) <= 1 {
			continue
		}
		return xfmt.Errorf("%s: application %q build produced multiple %s models", spec.DuplicateCode, app, spec.ModelName)
	}
	return nil
}
