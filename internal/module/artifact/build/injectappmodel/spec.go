// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import "sync"

// Spec describes one injectable app-scoped model (FieldDefault, AppSetting,
// PropertyDefinition, …).
type Spec struct {
	ModelName        string
	GeneratedRelPath string
	DuplicateCode    string
	BaseModelFile    string // relative under modules, e.g. core/service/orm/model/field_default_base_model.ts
	SoftDeleteFalse  bool   // AppSetting: emit softDelete: false in @Model options
	// EnsureServiceEntry: when true, Decide may proceed without ServiceEntryPoint
	// and Materialize emits a virtual service/index.ts (no package.json / disk
	// pollution). FieldDefault / AppSetting / PropertyDefinition leave this false
	// so a virtual Ensure for TranslationTerm does not unlock them — they require
	// a declared entryPoints.service on the module.
	EnsureServiceEntry bool
}

// Register adds a Spec to DefaultRegistry(). ModelName must be unique.
func Register(spec Spec) {
	DefaultRegistry().Register(spec)
}

// Specs returns specs from DefaultRegistry() in registration order.
func Specs() []Spec {
	return DefaultRegistry().Specs()
}

func specByName(name string) (*Spec, bool) {
	return DefaultRegistry().lookupPtr(name)
}

func specsList() []*Spec {
	return DefaultRegistry().specsList()
}

// ResetScheduledForTest clears DefaultRegistry claims (tests only).
// Deprecated: prefer Registry.ResetClaims on a test-owned registry, or
// DefaultRegistry().ResetClaims().
func ResetScheduledForTest() {
	DefaultRegistry().ResetClaims()
}

// ScheduledApps returns the NeedInject dedup map for modelName on DefaultRegistry (tests).
// Deprecated: prefer Registry.ClaimOwner / TryClaim / ResetClaims.
func ScheduledApps(modelName string) *sync.Map {
	r := DefaultRegistry()
	if _, ok := r.Lookup(modelName); !ok {
		panic("injectappmodel: ScheduledApps unknown model " + modelName)
	}
	return r.claimMap(modelName)
}
