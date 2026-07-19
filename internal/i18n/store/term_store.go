// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

// TermStore is the in-process terminology cache for one Application.
// Hot path reads memory only; callers must WarmLanguage before Lookup hits.
type TermStore struct {
	mu           sync.RWMutex
	runtimeScope scope.Scope
	application  string
	// lang → module → scope → kind → src → value
	cache map[string]map[string]map[string]map[string]map[string]string
	// lang → termHash
	termHash map[string]string
	// lang → epoch bumped by mutations so a concurrent WarmLanguage cannot
	// overwrite a newer in-memory update with a stale DB snapshot.
	warmEpoch map[string]uint64
}

// NewTermStore creates an empty store for the given application.
func NewTermStore(runtimeScope scope.Scope, application string) *TermStore {
	return &TermStore{
		runtimeScope: runtimeScope,
		application:  strings.TrimSpace(application),
		cache:        make(map[string]map[string]map[string]map[string]map[string]string),
		termHash:     make(map[string]string),
		warmEpoch:    make(map[string]uint64),
	}
}

// Application returns the application this store belongs to.
func (s *TermStore) Application() string {
	return s.application
}

// warmAfterLoadHook is optional test plumbing invoked after DB rows are loaded
// and before the cache install decision.
var warmAfterLoadHook func(lang string)

// WarmLanguage loads all terms for lang from {app}_translation_term into cache.
func (s *TermStore) WarmLanguage(lang string) error {
	lang = strings.TrimSpace(lang)
	if lang == "" || s.application == "" || s.application == "core" {
		return nil
	}
	if s.runtimeScope == nil || s.runtimeScope.Session() == nil {
		return nil
	}

	tableName := i18nmodels.TranslationTermTableName(s.application)
	if !s.runtimeScope.Session().Migrator().HasTable(tableName) {
		s.mu.Lock()
		s.cache[lang] = make(map[string]map[string]map[string]map[string]string)
		s.termHash[lang] = emptyTermHash()
		s.mu.Unlock()
		return nil
	}

	s.mu.RLock()
	epoch := s.warmEpoch[lang]
	s.mu.RUnlock()

	var rows []i18nmodels.TranslationTerm
	if err := s.runtimeScope.Session().Table(tableName).
		Where("lang = ?", lang).
		Find(&rows).Error; err != nil {
		return fmt.Errorf("warm %s lang=%s: %w", tableName, lang, err)
	}

	next := make(map[string]map[string]map[string]map[string]string)
	for _, row := range rows {
		kind := i18nmodels.NormalizeKind(row.Kind)
		mod := row.Module
		scp := row.Scope
		src := row.Src
		if next[mod] == nil {
			next[mod] = make(map[string]map[string]map[string]string)
		}
		if next[mod][scp] == nil {
			next[mod][scp] = make(map[string]map[string]string)
		}
		if next[mod][scp][kind] == nil {
			next[mod][scp][kind] = make(map[string]string)
		}
		next[mod][scp][kind][src] = row.Value
	}

	hash := computeTermHash(rows)
	if warmAfterLoadHook != nil {
		warmAfterLoadHook(lang)
	}

	s.mu.Lock()
	if s.warmEpoch[lang] != epoch {
		// A concurrent upsert/invalidation landed newer cache state; keep it.
		s.mu.Unlock()
		return nil
	}
	s.cache[lang] = next
	s.termHash[lang] = hash
	s.mu.Unlock()
	return nil
}

// EvictLanguage removes a language from the hot cache.
func (s *TermStore) EvictLanguage(lang string) {
	lang = strings.TrimSpace(lang)
	s.mu.Lock()
	s.warmEpoch[lang]++
	delete(s.cache, lang)
	delete(s.termHash, lang)
	s.mu.Unlock()
}

// Lookup returns a cached translation. ok=false on miss (including unwarmed lang).
func (s *TermStore) Lookup(module, lang, scopeKey, src, kind string) (string, bool) {
	module = strings.TrimSpace(module)
	lang = strings.TrimSpace(lang)
	scopeKey = strings.TrimSpace(scopeKey)
	src = strings.TrimSpace(src)
	kind = i18nmodels.NormalizeKind(kind)

	s.mu.RLock()
	defer s.mu.RUnlock()
	byMod, ok := s.cache[lang]
	if !ok {
		return "", false
	}
	byScope, ok := byMod[module]
	if !ok {
		return "", false
	}
	byKind, ok := byScope[scopeKey]
	if !ok {
		return "", false
	}
	bySrc, ok := byKind[kind]
	if !ok {
		return "", false
	}
	val, ok := bySrc[src]
	return val, ok
}

// TermsByModules returns a deep copy of warmed **literal** terms for the given modules.
// Explicit non-literal kinds are omitted so Gateway vue-i18n messages stay kind-free.
// Modules absent from cache are omitted. Call WarmLanguage first for a complete view.
// Shape: module → scope → src → value.
func (s *TermStore) TermsByModules(lang string, modules []string) map[string]map[string]map[string]string {
	lang = strings.TrimSpace(lang)
	if lang == "" || len(modules) == 0 {
		return map[string]map[string]map[string]string{}
	}
	wanted := make(map[string]struct{}, len(modules))
	for _, m := range modules {
		m = strings.TrimSpace(m)
		if m != "" {
			wanted[m] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return map[string]map[string]map[string]string{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	byMod, ok := s.cache[lang]
	if !ok {
		return map[string]map[string]map[string]string{}
	}

	out := make(map[string]map[string]map[string]string)
	for mod := range wanted {
		byScope, ok := byMod[mod]
		if !ok {
			continue
		}
		modCopy := make(map[string]map[string]string)
		for scopeKey, byKind := range byScope {
			bySrc, ok := byKind[i18nmodels.KindLiteral]
			if !ok || len(bySrc) == 0 {
				continue
			}
			srcCopy := make(map[string]string, len(bySrc))
			for src, val := range bySrc {
				srcCopy[src] = val
			}
			modCopy[scopeKey] = srcCopy
		}
		if len(modCopy) > 0 {
			out[mod] = modCopy
		}
	}
	return out
}

// InvalidateModule drops module entries from all warmed languages and bumps hashes.
func (s *TermStore) InvalidateModule(module string) {
	module = strings.TrimSpace(module)
	s.mu.Lock()
	defer s.mu.Unlock()
	for lang, byMod := range s.cache {
		s.warmEpoch[lang]++
		delete(byMod, module)
		s.termHash[lang] = bumpHash(s.termHash[lang])
	}
}

// TermListItem is one terminology catalog row (SearchTerms / UpdateTerm).
type TermListItem struct {
	Application string
	Module      string
	Scope       string
	Src         string
	Value       string
	Kind        string
	Source      string
	Status      string
}

const (
	statusTranslated = "translated"
	statusMissing    = "missing"
	statusFuzzy      = "fuzzy"
)

// SearchTerms lists DB rows for lang with optional module filter and free-text q.
// When modules is empty, all modules in this application table are included.
// q matches module, scope, src, or value (case-insensitive substring).
func (s *TermStore) SearchTerms(lang string, modules []string, q string, limit, offset int) ([]TermListItem, int64, error) {
	lang = strings.TrimSpace(lang)
	q = strings.TrimSpace(q)
	if lang == "" || s.application == "" || s.application == "core" {
		return nil, 0, nil
	}
	if s.runtimeScope == nil || s.runtimeScope.Session() == nil {
		return nil, 0, fmt.Errorf("search terms: missing runtime session")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	tableName := i18nmodels.TranslationTermTableName(s.application)
	if !s.runtimeScope.Session().Migrator().HasTable(tableName) {
		return nil, 0, nil
	}

	base := s.runtimeScope.Session().Table(tableName).Where("lang = ?", lang)
	if len(modules) > 0 {
		wanted := make([]string, 0, len(modules))
		for _, m := range modules {
			m = strings.TrimSpace(m)
			if m != "" {
				wanted = append(wanted, m)
			}
		}
		if len(wanted) == 0 {
			return nil, 0, nil
		}
		base = base.Where("module IN ?", wanted)
	}
	if q != "" {
		like := "%" + q + "%"
		base = base.Where("(module LIKE ? OR scope LIKE ? OR src LIKE ? OR value LIKE ?)", like, like, like, like)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count terms: %w", err)
	}

	var rows []i18nmodels.TranslationTerm
	if err := base.Order("module ASC, scope ASC, src ASC, kind ASC").
		Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list terms: %w", err)
	}

	items := make([]TermListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, termListItemFromRow(s.application, row))
	}
	return items, total, nil
}

// UpsertOverride writes Source=override, refreshes the hot cache, and bumps termHash.
func (s *TermStore) UpsertOverride(module, lang, scopeKey, src, kind, value string) (TermListItem, error) {
	module = strings.TrimSpace(module)
	lang = strings.TrimSpace(lang)
	scopeKey = strings.TrimSpace(scopeKey)
	src = strings.TrimSpace(src)
	kind = i18nmodels.NormalizeKind(kind)
	if module == "" || lang == "" || scopeKey == "" || src == "" {
		return TermListItem{}, fmt.Errorf("upsert override: module, lang, scope, and src are required")
	}
	if s.application == "" || s.application == "core" {
		return TermListItem{}, fmt.Errorf("upsert override: invalid application")
	}
	if s.runtimeScope == nil || s.runtimeScope.Session() == nil {
		return TermListItem{}, fmt.Errorf("upsert override: missing runtime session")
	}

	if err := i18nmodels.EnsureTranslationTermTable(s.runtimeScope, s.application); err != nil {
		return TermListItem{}, err
	}
	tableName := i18nmodels.TranslationTermTableName(s.application)
	session := s.runtimeScope.Session()

	var existing i18nmodels.TranslationTerm
	err := session.Table(tableName).
		Where("module = ? AND lang = ? AND scope = ? AND src = ? AND kind = ?",
			module, lang, scopeKey, src, kind).
		Take(&existing).Error
	if err == nil {
		existing.Value = value
		existing.Source = i18nmodels.SourceOverride
		existing.Application = s.application
		if err := session.Table(tableName).Save(&existing).Error; err != nil {
			return TermListItem{}, fmt.Errorf("update override: %w", err)
		}
		s.applyOverrideToCache(module, lang, scopeKey, src, kind, value)
		s.BumpTermHash(lang)
		return termListItemFromRow(s.application, existing), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return TermListItem{}, fmt.Errorf("lookup term: %w", err)
	}

	row := i18nmodels.TranslationTerm{
		Application: s.application,
		Module:      module,
		Lang:        lang,
		Scope:       scopeKey,
		Src:         src,
		Value:       value,
		Kind:        kind,
		Source:      i18nmodels.SourceOverride,
	}
	if err := session.Table(tableName).Create(&row).Error; err != nil {
		return TermListItem{}, fmt.Errorf("create override: %w", err)
	}
	s.applyOverrideToCache(module, lang, scopeKey, src, kind, value)
	s.BumpTermHash(lang)
	return termListItemFromRow(s.application, row), nil
}

func (s *TermStore) applyOverrideToCache(module, lang, scopeKey, src, kind, value string) {
	kind = i18nmodels.NormalizeKind(kind)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.warmEpoch[lang]++
	if s.cache[lang] == nil {
		s.cache[lang] = make(map[string]map[string]map[string]map[string]string)
	}
	if s.cache[lang][module] == nil {
		s.cache[lang][module] = make(map[string]map[string]map[string]string)
	}
	if s.cache[lang][module][scopeKey] == nil {
		s.cache[lang][module][scopeKey] = make(map[string]map[string]string)
	}
	if s.cache[lang][module][scopeKey][kind] == nil {
		s.cache[lang][module][scopeKey][kind] = make(map[string]string)
	}
	s.cache[lang][module][scopeKey][kind][src] = value
}

func termListItemFromRow(application string, row i18nmodels.TranslationTerm) TermListItem {
	kind := i18nmodels.NormalizeKind(row.Kind)
	source := row.Source
	if source == "" {
		source = i18nmodels.SourcePackaged
	}
	status := statusTranslated
	if strings.TrimSpace(row.Value) == "" {
		status = statusMissing
	} else if strings.Contains(strings.ToLower(row.Comments), "fuzzy") {
		status = statusFuzzy
	}
	app := row.Application
	if app == "" {
		app = application
	}
	return TermListItem{
		Application: app,
		Module:      row.Module,
		Scope:       row.Scope,
		Src:         row.Src,
		Value:       row.Value,
		Kind:        kind,
		Source:      source,
		Status:      status,
	}
}

// BumpTermHash updates the hash for lang (e.g. after UpdateTerm) without full recompute.
func (s *TermStore) BumpTermHash(lang string) string {
	lang = strings.TrimSpace(lang)
	s.mu.Lock()
	defer s.mu.Unlock()
	next := bumpHash(s.termHash[lang])
	s.termHash[lang] = next
	return next
}

// TermHash returns the current hash for lang (empty string if unknown).
func (s *TermStore) TermHash(lang string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.termHash[strings.TrimSpace(lang)]
}

func emptyTermHash() string {
	sum := sha256.Sum256(nil)
	return hex.EncodeToString(sum[:8])
}

// EmptyTermHash returns the stable hash for an empty terminology set (D5).
func EmptyTermHash() string {
	return emptyTermHash()
}

func bumpHash(prev string) string {
	sum := sha256.Sum256([]byte(prev + "|bump"))
	return hex.EncodeToString(sum[:8])
}

func computeTermHash(rows []i18nmodels.TranslationTerm) string {
	type key struct {
		module, scope, src, kind, value, source string
	}
	keys := make([]key, 0, len(rows))
	for _, row := range rows {
		kind := i18nmodels.NormalizeKind(row.Kind)
		source := row.Source
		if source == "" {
			source = i18nmodels.SourcePackaged
		}
		keys = append(keys, key{row.Module, row.Scope, row.Src, kind, row.Value, source})
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.module != b.module {
			return a.module < b.module
		}
		if a.scope != b.scope {
			return a.scope < b.scope
		}
		if a.src != b.src {
			return a.src < b.src
		}
		if a.kind != b.kind {
			return a.kind < b.kind
		}
		if a.value != b.value {
			return a.value < b.value
		}
		return a.source < b.source
	})
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\n", k.module, k.scope, k.src, k.kind, k.value, k.source)
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:8])
}
