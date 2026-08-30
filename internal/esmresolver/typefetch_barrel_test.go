// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestCollapseLargeDefaultAsReexportBarrel_EmptyOrWhitespace(t *testing.T) {
	if got := collapseLargeDefaultAsReexportBarrel(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
	if got := collapseLargeDefaultAsReexportBarrel("   \n\t  "); got != "   \n\t  " {
		t.Fatalf("whitespace: got %q", got)
	}
}

func TestCollapseLargeDefaultAsReexportBarrel_AllowsComments(t *testing.T) {
	var b strings.Builder
	b.WriteString("// header\n")
	b.WriteString("/* block */\n")
	b.WriteString(" * continued\n")
	b.WriteString(" */\n")
	for i := 0; i < largeDefaultAsReexportThreshold; i++ {
		fmt.Fprintf(&b, "export { default as Icon%d } from\"./leaf_%d.d.ts.d.ts\";\n", i, i)
	}
	got := collapseLargeDefaultAsReexportBarrel(b.String())
	if strings.Contains(got, `from"./leaf_`) {
		t.Fatalf("expected collapse with comments present, got %q", got[:min(200, len(got))])
	}
	if !strings.Contains(got, "export declare const Icon0:") {
		t.Fatalf("missing Icon0: %q", got[:min(200, len(got))])
	}
}

func TestCollapseLargeDefaultAsReexportBarrel_DuplicateNamesBelowThreshold(t *testing.T) {
	var b strings.Builder
	for i := 0; i < largeDefaultAsReexportThreshold+20; i++ {
		fmt.Fprintf(&b, "export { default as Icon%d } from\"./leaf_%d.d.ts.d.ts\";\n", i%10, i)
	}
	in := b.String()
	if got := collapseLargeDefaultAsReexportBarrel(in); got != in {
		t.Fatalf("expected unchanged when unique names stay below threshold")
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

func TestFetchTypeDefinition_CollapsesLargeDefaultAsBarrel(t *testing.T) {
	var body strings.Builder
	for i := 0; i < largeDefaultAsReexportThreshold+3; i++ {
		fmt.Fprintf(&body, "export { default as Icon%d } from \"./leaf_%d.d.ts\";\n", i, i)
	}
	typesURLPath := "/@vicons/material@0.13.0/es/index.d.ts"
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead && strings.HasPrefix(r.URL.Path, "/@vicons/material@0.13.0") && r.URL.RawQuery == "dts":
			w.Header().Set("x-typescript-types", srv.URL+typesURLPath)
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == typesURLPath:
			w.Header().Set("Content-Type", "application/typescript")
			_, _ = w.Write([]byte(body.String()))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	typesDir := t.TempDir()
	result, _, err := FetchTypeDefinition(srv.Client(), srv.URL, typesDir, "@vicons/material", "0.13.0")
	if err != nil {
		t.Fatalf("FetchTypeDefinition: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	data, err := os.ReadFile(result.CachedPath)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	content := string(data)
	if strings.Contains(content, `from "./leaf_`) || strings.Contains(content, `from"./leaf_`) {
		t.Fatalf("expected collapsed barrel on disk, got %q", content[:min(300, len(content))])
	}
	if !strings.Contains(content, "export declare const Icon0:") {
		t.Fatalf("expected collapsed named export, got %q", content[:min(300, len(content))])
	}
}
