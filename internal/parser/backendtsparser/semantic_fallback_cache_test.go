// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"testing"
)

func TestCachedTextFallback_CachesByAnnotation(t *testing.T) {
	r := newSemanticTypeResolver(nil)
	first := r.cachedTextFallback("string")
	second := r.cachedTextFallback("string")
	if first != second || first != "string" {
		t.Fatalf("fallback string=%q/%q", first, second)
	}
	r.mu.Lock()
	misses := r.fallbackMisses
	hits := r.fallbackCacheHits
	r.mu.Unlock()
	if misses != 1 {
		t.Fatalf("fallbackMisses=%d want 1", misses)
	}
	if hits != 1 {
		t.Fatalf("fallbackCacheHits=%d want 1", hits)
	}

	_ = r.cachedTextFallback("number")
	r.mu.Lock()
	misses = r.fallbackMisses
	r.mu.Unlock()
	if misses != 2 {
		t.Fatalf("fallbackMisses=%d want 2 after second annotation", misses)
	}
}

func TestCachedTextFallback_NilReceiver(t *testing.T) {
	if got := (*semanticTypeResolver)(nil).cachedTextFallback("boolean"); got != "bool" {
		t.Fatalf("nil receiver fallback=%q", got)
	}
}

func TestSharedSemanticTypeResolver_Reused(t *testing.T) {
	ResetSharedSemanticTypeResolverForTest()
	t.Cleanup(ResetSharedSemanticTypeResolverForTest)

	a := sharedSemanticTypeResolver(nil)
	b := sharedSemanticTypeResolver(nil)
	if a != b {
		t.Fatal("expected process-wide shared resolver")
	}
	ResetSharedSemanticTypeResolverForTest()
	c := sharedSemanticTypeResolver(nil)
	if c == a {
		t.Fatal("expected new resolver after reset")
	}
}
