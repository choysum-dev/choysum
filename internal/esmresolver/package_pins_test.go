// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExactPinsFromPackageJSON(t *testing.T) {
	t.Parallel()
	pins, err := ExactPinsFromPackageJSON("")
	if err != nil || pins != nil {
		t.Fatalf("empty root => nil,nil got %#v %v", pins, err)
	}
	pins, err = ExactPinsFromPackageJSON(t.TempDir())
	if err != nil || pins != nil {
		t.Fatalf("missing package.json => nil,nil got %#v %v", pins, err)
	}

	dir := t.TempDir()
	pkg := `{
  "dependencies": {"left-pad": "1.3.0", "lodash": "^4.17.21"},
  "peerDependencies": {
    "@tanstack/vue-table": "8.21.3",
    "vue": "^3.5.35",
    "reka-ui": "v2.10.4"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	pins, err = ExactPinsFromPackageJSON(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"left-pad":             "1.3.0",
		"@tanstack/vue-table":  "8.21.3",
		"reka-ui":              "2.10.4",
	}
	if len(pins) != len(want) {
		t.Fatalf("pins=%v want %v", pins, want)
	}
	for k, v := range want {
		if pins[k] != v {
			t.Fatalf("pins[%q]=%q want %q", k, pins[k], v)
		}
	}
	if _, ok := pins["vue"]; ok {
		t.Fatal("range peer vue must not be pinned")
	}
	if _, ok := pins["lodash"]; ok {
		t.Fatal("range dependency lodash must not be pinned")
	}
}

func TestExactPinsFromPackageJSONDependencyWinsOverPeer(t *testing.T) {
	dir := t.TempDir()
	pkg := `{
  "dependencies": {"left-pad": "1.3.0"},
  "peerDependencies": {"left-pad": "1.2.0"}
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	pins, err := ExactPinsFromPackageJSON(dir)
	if err != nil {
		t.Fatal(err)
	}
	if pins["left-pad"] != "1.3.0" {
		t.Fatalf("exact dependency must win over peer, got %q", pins["left-pad"])
	}
}

func TestExactPinsFromPackageJSONInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ExactPinsFromPackageJSON(dir)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestExactPinsFromPackageJSONReadErrorAndEmptyExact(t *testing.T) {
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgPath, []byte(`{"dependencies":{"left-pad":"^1.0.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	pins, err := ExactPinsFromPackageJSON(dir)
	if err != nil || pins != nil {
		t.Fatalf("only ranges => nil,nil got %#v %v", pins, err)
	}

	// package.json as a directory makes ReadFile fail with a non-NotExist error.
	badRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(badRoot, "package.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = ExactPinsFromPackageJSON(badRoot)
	if err == nil {
		t.Fatal("expected read error when package.json is a directory")
	}
}
