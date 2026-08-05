// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import "sync"

// Spec describes one injectable app-scoped model (FieldDefault, AppSetting, …).
type Spec struct {
	ModelName        string
	GeneratedRelPath string
	DuplicateCode    string
	BaseModelFile    string // relative under modules, e.g. core/service/orm/model/field_default_base_model.ts
	SoftDeleteFalse  bool   // AppSetting: emit softDelete: false in @Model options
	// ForeignClaimOnOwnerReinject: when DB virtual rows belong to this module but another
	// in-process builder holds the schedule claim, still NeedInject without adopting release.
	ForeignClaimOnOwnerReinject bool

	scheduled sync.Map // process-wide NeedInject dedup keyed by application
}

var (
	specsMu   sync.RWMutex
	specOrder []string
	specsBy   = map[string]*Spec{}
)

// Register adds a Spec to the process-wide registry. ModelName must be unique.
func Register(spec Spec) {
	specsMu.Lock()
	defer specsMu.Unlock()
	if _, exists := specsBy[spec.ModelName]; exists {
		panic("injectappmodel: duplicate Register for " + spec.ModelName)
	}
	s := spec
	specsBy[spec.ModelName] = &s
	specOrder = append(specOrder, spec.ModelName)
}

// Specs returns registered specs in registration order.
func Specs() []Spec {
	specsMu.RLock()
	defer specsMu.RUnlock()
	out := make([]Spec, 0, len(specOrder))
	for _, name := range specOrder {
		if s, ok := specsBy[name]; ok {
			out = append(out, *s)
		}
	}
	return out
}

func specByName(name string) (*Spec, bool) {
	specsMu.RLock()
	defer specsMu.RUnlock()
	s, ok := specsBy[name]
	return s, ok
}

func specsList() []*Spec {
	specsMu.RLock()
	defer specsMu.RUnlock()
	out := make([]*Spec, 0, len(specOrder))
	for _, name := range specOrder {
		if s, ok := specsBy[name]; ok {
			out = append(out, s)
		}
	}
	return out
}

// ResetScheduledForTest clears process-wide inject dedup maps (tests only).
func ResetScheduledForTest() {
	for _, spec := range specsList() {
		spec.scheduled.Range(func(key, _ any) bool {
			spec.scheduled.Delete(key)
			return true
		})
	}
}

// ScheduledApps returns the process-wide NeedInject dedup map for modelName (tests).
func ScheduledApps(modelName string) *sync.Map {
	if spec, ok := specByName(modelName); ok {
		return &spec.scheduled
	}
	return &sync.Map{}
}
