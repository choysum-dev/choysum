// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// TermStore is the in-process Lookup cache for one Application (`_t` hot path).
// Packaged writes go through i18nimport.UpsertPackagedTerms; overrides via
// TranslationTerm ORM. This type is not a Search/Update service backend.
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

const warmLanguageMaxAttempts = 5

// WarmLanguage loads all terms for lang from {app}_translation_term into cache.
// Retries when a concurrent mutation bumps warmEpoch so a stale snapshot is not
// discarded while leaving an incomplete cache.
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

	for attempt := 0; attempt < warmLanguageMaxAttempts; attempt++ {
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
			// Concurrent upsert/invalidation advanced the cache; reload.
			s.mu.Unlock()
			continue
		}
		s.cache[lang] = next
		s.termHash[lang] = hash
		s.mu.Unlock()
		return nil
	}
	return fmt.Errorf("warm language %s failed after %d attempts due to concurrent updates", lang, warmLanguageMaxAttempts)
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

// BumpTermHash updates the hash for lang without full recompute.
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
