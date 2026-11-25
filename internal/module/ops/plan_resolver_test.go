// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/module/plan"
	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

type countingResolver struct {
	loads int
	peeks int
	load  map[string]*pkgmeta.Module
	peek  map[string]*pkgmeta.Module
	err   error
}

func (r *countingResolver) Load(name string) (*pkgmeta.Module, error) {
	r.loads++
	if r.err != nil {
		return nil, r.err
	}
	if r.load == nil {
		return nil, nil
	}
	return r.load[name], nil
}

func (r *countingResolver) Peek(ctx context.Context, name string) (*pkgmeta.Module, error) {
	r.peeks++
	if r.err != nil {
		return nil, r.err
	}
	if r.peek == nil {
		return nil, nil
	}
	return r.peek[name], nil
}

func TestPlanResolver_CachesLoadAndPeek(t *testing.T) {
	t.Parallel()
	inner := &countingResolver{
		load: map[string]*pkgmeta.Module{
			"core": {Name: "core", Version: "1.0.0", Status: pkgmeta.Installed},
		},
		peek: map[string]*pkgmeta.Module{
			"sandbox": {Name: "sandbox", Version: "1.0.0"},
		},
	}
	r := newPlanResolver(inner)
	if _, err := r.Load("core"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := r.Load("core"); err != nil {
		t.Fatalf("Load again: %v", err)
	}
	if inner.loads != 1 {
		t.Fatalf("loads=%d, want 1", inner.loads)
	}
	if _, err := r.Peek(context.Background(), "sandbox"); err != nil {
		t.Fatalf("Peek: %v", err)
	}
	if _, err := r.Peek(context.Background(), "sandbox"); err != nil {
		t.Fatalf("Peek again: %v", err)
	}
	if inner.peeks != 1 {
		t.Fatalf("peeks=%d, want 1", inner.peeks)
	}
}

func TestPlanResolver_DoesNotCacheErrors(t *testing.T) {
	t.Parallel()
	inner := &countingResolver{err: errors.New("transient")}
	r := newPlanResolver(inner)
	if _, err := r.Load("core"); err == nil {
		t.Fatal("expected Load error")
	}
	inner.err = nil
	inner.load = map[string]*pkgmeta.Module{
		"core": {Name: "core", Version: "1.0.0", Status: pkgmeta.Installed},
	}
	if _, err := r.Load("core"); err != nil {
		t.Fatalf("Load after recovery: %v", err)
	}
	if inner.loads != 2 {
		t.Fatalf("loads=%d, want 2 (error not cached)", inner.loads)
	}
}

func TestPlanResolver_NilReceiverAndNilResolverFallbacks(t *testing.T) {
	t.Parallel()
	var nilPR *planResolver
	if mod, err := nilPR.Load("x"); mod != nil || err != nil {
		t.Fatalf("nil planResolver.Load => (%v,%v)", mod, err)
	}
	if mod, err := nilPR.Peek(context.Background(), "x"); mod != nil || err != nil {
		t.Fatalf("nil planResolver.Peek => (%v,%v)", mod, err)
	}

	o := &Ops{}
	if _, _, err := o.resolveRootModule(context.Background(), nil, "core", originLocal); err == nil {
		t.Fatal("resolveRootModule with nil resolvers want error")
	}
	inner := &countingResolver{
		load: map[string]*pkgmeta.Module{
			"core": {Name: "core", Version: "1.0.0", Status: pkgmeta.Installed},
		},
	}
	o.Resolver = inner
	if _, _, err := o.resolveRootModule(context.Background(), nil, "core", originLocal); err != nil {
		t.Fatalf("resolveRootModule fallback: %v", err)
	}
	_ = o.enrichPlanDocument(context.Background(), nil, nil, ActionInstall, "core", originLocal, plan.Plan{}, nil)
	_, _ = o.versionsForModule(context.Background(), nil, nil, "core", originLocal)
	_ = computeBaseRevision(context.Background(), o, nil, nil, []ModuleOrderEntry{{ModuleName: "core"}})
	o.Resolver = nil
	if issues := o.uninstallBlockers(context.Background(), nil, "core", nil); len(issues) != 1 || issues[0].Code != "DEPENDENCY_LOOKUP_FAILED" {
		t.Fatalf("uninstallBlockers nil resolver => %#v", issues)
	}
}
