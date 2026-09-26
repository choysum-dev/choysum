// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysummount"
)

func TestBuildFrontendVueHostBundlePinsVueVersion(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	entry := filepath.Join(dir, "entry.js")
	out := filepath.Join(dir, "out.js")
	// Import pinia so its peer /vue@^… path is rewritten to the host pin.
	if err := os.WriteFile(entry, []byte(`
import { ref } from 'vue';
import { defineStore, createPinia } from 'pinia';
import { createI18n } from 'vue-i18n';
console.log(ref, defineStore, createPinia, createI18n);
`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Prefer CHOYSUM_HOME when set; otherwise the repo-local .choysum cache
	// (populated by go generate / install) so CI stays offline-friendly.
	cache := strings.TrimSpace(os.Getenv("CHOYSUM_HOME"))
	if cache == "" {
		cache = filepath.Join(repoRoot, ".choysum")
	}
	res, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:              repoRoot,
		EntryPath:             entry,
		Outfile:               out,
		CacheDir:              cache,
		DisableDefaultFEStubs: true,
	})
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "download failed") ||
			strings.Contains(msg, "cache miss (offline)") ||
			strings.Contains(msg, "no such host") ||
			strings.Contains(msg, "connection refused") {
			t.Skipf("skipping vue pin bundle test without esm cache/network: %v", err)
		}
		t.Fatalf("bundle: %v", err)
	}
	refs := regexp.MustCompile(`vue@[0-9A-Za-z.^~*<>=-]+`).FindAllString(res.JS, -1)
	if len(refs) == 0 {
		t.Fatal("expected pinned vue version markers in bundle")
	}
	want := "vue@" + choysummount.VuePackageVersion
	for _, ref := range refs {
		if ref != want {
			t.Fatalf("bundle references %q, want only %s", ref, want)
		}
	}
}

func TestVueHostBareImportPinsIncludesWebExactPeers(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	pins, err := vueHostBareImportPins(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if pins["vue"] != choysummount.VuePackageVersion {
		t.Fatalf("vue pin = %q want %q", pins["vue"], choysummount.VuePackageVersion)
	}
	if pins["@tanstack/vue-table"] != "8.21.3" {
		t.Fatalf("@tanstack/vue-table pin = %q want 8.21.3", pins["@tanstack/vue-table"])
	}
	if pins["@tanstack/vue-virtual"] != "3.13.39" {
		t.Fatalf("@tanstack/vue-virtual pin = %q want 3.13.39", pins["@tanstack/vue-virtual"])
	}
}

func TestVueHostBareImportPinsFallbackAndHostVueWins(t *testing.T) {
	empty, err := vueHostBareImportPins(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 1 || empty["vue"] != choysummount.VuePackageVersion {
		t.Fatalf("missing web package.json => host vue only, got %#v", empty)
	}

	root := t.TempDir()
	webRoot := filepath.Join(root, "modules", "web")
	if err := os.MkdirAll(webRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := `{
  "peerDependencies": {
    "vue": "9.9.9",
    "@tanstack/vue-table": "8.21.3"
  }
}`
	if err := os.WriteFile(filepath.Join(webRoot, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	pins, err := vueHostBareImportPins(root)
	if err != nil {
		t.Fatal(err)
	}
	if pins["vue"] != choysummount.VuePackageVersion {
		t.Fatalf("host vue must win over package.json, got %q", pins["vue"])
	}
	if pins["@tanstack/vue-table"] != "8.21.3" {
		t.Fatalf("expected peer pin, got %#v", pins)
	}

	if err := os.WriteFile(filepath.Join(webRoot, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := vueHostBareImportPins(root); err == nil || !strings.Contains(err.Error(), "exact pins") {
		t.Fatalf("expected exact pins error, got %v", err)
	}
}
