// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package exportpkg_test

import (
	"testing"

	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestMode_Valid(t *testing.T) {
	if !exportpkg.ModeData.Valid() || !exportpkg.ModeTemplate.Valid() {
		t.Fatal("data and template modes should be valid")
	}
	if exportpkg.Mode("bogus").Valid() {
		t.Fatal("unknown mode should be invalid")
	}
}

func TestEffectiveMode(t *testing.T) {
	if got := exportpkg.EffectiveMode(exportpkg.ModeUnspecified); got != exportpkg.ModeData {
		t.Fatalf("EffectiveMode(unspecified) = %q, want data", got)
	}
	if got := exportpkg.EffectiveMode(exportpkg.ModeTemplate); got != exportpkg.ModeTemplate {
		t.Fatalf("EffectiveMode(template) = %q, want template", got)
	}
}
