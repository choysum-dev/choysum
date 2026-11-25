// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	"github.com/choysum-dev/choysum/internal/module/plan"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

// planResolver is a Plan-scoped Resolver: read-only (no warm AutoMigrate) plus
// per-request Load/Peek memoization for enrichment / BuildPlan repeats.
type planResolver struct {
	inner plan.Resolver

	mu         sync.Mutex
	loadByName map[string]loadCacheEntry
	peekByName map[string]peekCacheEntry
}

type loadCacheEntry struct {
	mod *pkgmeta.Module
	err error
}

type peekCacheEntry struct {
	mod *pkgmeta.Module
	err error
}

func newPlanResolver(inner plan.Resolver) plan.Resolver {
	inner = lifecycle.AsReadOnlyResolver(inner)
	if inner == nil {
		return nil
	}
	return &planResolver{
		inner:      inner,
		loadByName: make(map[string]loadCacheEntry),
		peekByName: make(map[string]peekCacheEntry),
	}
}

func (r *planResolver) Load(name string) (*pkgmeta.Module, error) {
	if r == nil || r.inner == nil {
		return nil, nil
	}
	key := strings.TrimSpace(name)
	r.mu.Lock()
	if ent, ok := r.loadByName[key]; ok {
		r.mu.Unlock()
		return ent.mod, ent.err
	}
	r.mu.Unlock()

	mod, err := r.inner.Load(key)
	r.mu.Lock()
	r.loadByName[key] = loadCacheEntry{mod: mod, err: err}
	r.mu.Unlock()
	return mod, err
}

func (r *planResolver) Peek(ctx context.Context, name string) (*pkgmeta.Module, error) {
	if r == nil || r.inner == nil {
		return nil, nil
	}
	key := strings.TrimSpace(name)
	r.mu.Lock()
	if ent, ok := r.peekByName[key]; ok {
		r.mu.Unlock()
		return ent.mod, ent.err
	}
	r.mu.Unlock()

	mod, err := r.inner.Peek(ctx, key)
	r.mu.Lock()
	r.peekByName[key] = peekCacheEntry{mod: mod, err: err}
	r.mu.Unlock()
	return mod, err
}
