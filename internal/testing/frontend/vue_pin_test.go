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
