// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package choysummount

import (
	"strings"
	"testing"
)

func TestEmbeddedScripts(t *testing.T) {
	if MinimalDOMScript == "" || !strings.Contains(MinimalDOMScript, "__choysumMinimalDOM") {
		t.Fatal("MinimalDOMScript missing")
	}
	if ChoysumMountScript == "" || !strings.Contains(ChoysumMountScript, "export function mount") {
		t.Fatal("ChoysumMountScript missing mount")
	}
	if VuePackageVersion == "" {
		t.Fatal("empty VuePackageVersion")
	}
}
