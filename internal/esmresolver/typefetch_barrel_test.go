// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"fmt"
	"strings"
	"testing"
)

func TestCollapseLargeDefaultAsReexportBarrel(t *testing.T) {
	var b strings.Builder
	for i := 0; i < largeDefaultAsReexportThreshold+5; i++ {
		fmt.Fprintf(&b, "export { default as Icon%d } from\"./leaf_%d.d.ts.d.ts\";\n", i, i)
	}
	got := collapseLargeDefaultAsReexportBarrel(b.String())
	if strings.Contains(got, `from"./leaf_`) || strings.Contains(got, `from "./leaf_`) {
		t.Fatalf("expected relative re-exports to be collapsed, got prefix %q", got[:min(200, len(got))])
	}
	if !strings.Contains(got, "export declare const Icon0:") {
		t.Fatalf("expected named export Icon0, got %q", got[:min(300, len(got))])
	}
	if !strings.Contains(got, "export declare const Icon104:") {
		t.Fatalf("expected named export Icon104, got suffix around end")
	}
	if !strings.Contains(got, "__choysumCollapsedIcon") {
		t.Fatalf("expected shared icon type, got %q", got[:min(300, len(got))])
	}
}

func TestCollapseLargeDefaultAsReexportBarrel_BelowThreshold(t *testing.T) {
	var b strings.Builder
	for i := 0; i < largeDefaultAsReexportThreshold-1; i++ {
		fmt.Fprintf(&b, "export { default as Icon%d } from\"./leaf_%d.d.ts.d.ts\";\n", i, i)
	}
	in := b.String()
	if got := collapseLargeDefaultAsReexportBarrel(in); got != in {
		t.Fatalf("expected unchanged below threshold")
	}
}

func TestCollapseLargeDefaultAsReexportBarrel_MixedContent(t *testing.T) {
	var b strings.Builder
	b.WriteString("export const keepMe = 1;\n")
	for i := 0; i < largeDefaultAsReexportThreshold+5; i++ {
		fmt.Fprintf(&b, "export { default as Icon%d } from\"./leaf_%d.d.ts.d.ts\";\n", i, i)
	}
	in := b.String()
	if got := collapseLargeDefaultAsReexportBarrel(in); got != in {
		t.Fatalf("expected mixed files to stay unchanged")
	}
}
