// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import (
	"os"
	"path/filepath"
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
	cache := filepath.Join(dir, "cache")
	res, err := BuildFrontendVueHostBundle(VueHostBundleOptions{
		RepoRoot:              repoRoot,
		EntryPath:             entry,
		Outfile:               out,
		CacheDir:              cache,
		DisableDefaultFEStubs: true,
	})
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}
	if !strings.Contains(res.JS, choysummount.VuePackageVersion) {
		t.Fatalf("expected pinned vue %s in bundle", choysummount.VuePackageVersion)
	}
	if strings.Contains(res.JS, "3.5.43") {
		t.Fatal("bundle pulled floating vue@3.5.43 despite WithBareImportPins")
	}
}
