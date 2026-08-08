// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import "sync"

// Registry holds the Spec catalog and NeedInject claims shared across builders
// that participate in the same process (DefaultRegistry) or install scope.
type Registry struct {
	mu        sync.RWMutex
	order     []string
	byName    map[string]*Spec
	scheduled map[string]*sync.Map // modelName → app → owning module name
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		byName:    make(map[string]*Spec),
		scheduled: make(map[string]*sync.Map),
	}
}

// NewRegistryWithDefaults returns a registry preloaded with DefaultSpecs().
func NewRegistryWithDefaults() *Registry {
	r := NewRegistry()
	for _, s := range DefaultSpecs() {
		r.Register(s)
	}
	return r
}

// Register adds a Spec. ModelName must be unique within r.
func (r *Registry) Register(spec Spec) {
	if r == nil {
		panic("injectappmodel: Register on nil Registry")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byName[spec.ModelName]; exists {
		panic("injectappmodel: duplicate Register for " + spec.ModelName)
	}
	s := spec
	r.byName[spec.ModelName] = &s
	r.order = append(r.order, spec.ModelName)
	if r.scheduled[spec.ModelName] == nil {
		r.scheduled[spec.ModelName] = &sync.Map{}
	}
}

// Specs returns a snapshot in registration order.
func (r *Registry) Specs() []Spec {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Spec, 0, len(r.order))
	for _, name := range r.order {
		if s, ok := r.byName[name]; ok {
			out = append(out, *s)
		}
	}
	return out
}

// Lookup returns the registered Spec by ModelName.
func (r *Registry) Lookup(modelName string) (Spec, bool) {
	if r == nil {
		return Spec{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.byName[modelName]
	if !ok {
		return Spec{}, false
	}
	return *s, true
}

func (r *Registry) lookupPtr(modelName string) (*Spec, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.byName[modelName]
	return s, ok
}

func (r *Registry) specsList() []*Spec {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Spec, 0, len(r.order))
	for _, name := range r.order {
		if s, ok := r.byName[name]; ok {
			out = append(out, s)
		}
	}
	return out
}

func (r *Registry) claimMapLocked(modelName string) *sync.Map {
	if r.scheduled == nil {
		r.scheduled = make(map[string]*sync.Map)
	}
	m := r.scheduled[modelName]
	if m == nil {
		m = &sync.Map{}
		r.scheduled[modelName] = m
	}
	return m
}

func (r *Registry) claimMap(modelName string) *sync.Map {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.claimMapLocked(modelName)
}

// TryClaim records that modName owns NeedInject for (modelName, app).
// loaded is true when another (or the same) owner already held the slot.
func (r *Registry) TryClaim(modelName, app, modName string) (owner string, loaded bool) {
	if r == nil {
		return "", false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	v, loaded := r.claimMapLocked(modelName).LoadOrStore(app, modName)
	owner, _ = v.(string)
	return owner, loaded
}

// ReleaseClaim drops the claim for (modelName, app) if present.
func (r *Registry) ReleaseClaim(modelName, app string) {
	if r == nil || app == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	m := r.scheduled[modelName]
	if m != nil {
		m.Delete(app)
	}
}

// ClaimOwner returns the module that currently owns NeedInject for (modelName, app).
func (r *Registry) ClaimOwner(modelName, app string) (modName string, ok bool) {
	if r == nil {
		return "", false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := r.scheduled[modelName]
	if m == nil {
		return "", false
	}
	v, ok := m.Load(app)
	if !ok {
		return "", false
	}
	modName, _ = v.(string)
	return modName, true
}

// ResetClaims clears all NeedInject claims (tests / install boundary).
func (r *Registry) ResetClaims() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for name := range r.scheduled {
		r.scheduled[name] = &sync.Map{}
	}
}

var (
	defaultRegistry     *Registry
	defaultRegistryOnce sync.Once
)

// DefaultSpecs returns builtin C2 Spec definitions without registering them.
// TranslationTerm is first so EnsureServiceEntry can synthesize a virtual
// service entry for i18n-only apps. FieldDefault / AppSetting still require a
// declared entryPoints.service (virtual Ensure must not unlock them).
func DefaultSpecs() []Spec {
	return []Spec{
		{
			ModelName:          "TranslationTerm",
			GeneratedRelPath:   "service/models/__generated__/translation_term.ts",
			DuplicateCode:      "TRANSLATION_TERM_DUPLICATE",
			BaseModelFile:      "core/service/orm/model/translation_term_base_model.ts",
			SoftDeleteFalse:    true,
			EnsureServiceEntry: true,
		},
		{
			ModelName:        "FieldDefault",
			GeneratedRelPath: "service/models/__generated__/field_default.ts",
			DuplicateCode:    "FIELD_DEFAULT_DUPLICATE",
			BaseModelFile:    "core/service/orm/model/field_default_base_model.ts",
			// EnsureServiceEntry: false — skip when package.json has no service entry.
		},
		{
			ModelName:        "AppSetting",
			GeneratedRelPath: "service/models/__generated__/app_setting.ts",
			DuplicateCode:    "APP_SETTING_DUPLICATE",
			BaseModelFile:    "core/service/orm/model/app_setting_base_model.ts",
			SoftDeleteFalse:  true,
			// EnsureServiceEntry: false — skip when package.json has no service entry.
		},
	}
}

// DefaultRegistry returns the process-wide shared registry, lazily initialized
// once with DefaultSpecs().
func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistryWithDefaults()
	})
	return defaultRegistry
}
