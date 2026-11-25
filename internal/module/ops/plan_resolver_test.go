// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package ops

import (
	"context"
	"testing"

	pkgmeta "github.com/choysum-dev/choysum/pkg/meta"
)

type countingResolver struct {
	loads int
	peeks int
	load  map[string]*pkgmeta.Module
	peek  map[string]*pkgmeta.Module
}

func (r *countingResolver) Load(name string) (*pkgmeta.Module, error) {
	r.loads++
	if r.load == nil {
		return nil, nil
	}
	return r.load[name], nil
}

func (r *countingResolver) Peek(ctx context.Context, name string) (*pkgmeta.Module, error) {
	r.peeks++
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
