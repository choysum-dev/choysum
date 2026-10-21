// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"strings"
	"testing"
)

func TestBuildContentDispositionKeepsASCIIFilenameParam(t *testing.T) {
	got := buildContentDisposition("inline", "截屏2026-09-30 13.45.31 (2).png")
	if !strings.HasPrefix(got, "inline; filename=") {
		t.Fatalf("disposition = %q, want inline prefix", got)
	}
	if !strings.Contains(got, "filename*=UTF-8''") {
		t.Fatalf("disposition = %q, want UTF-8 filename*", got)
	}
	// Quoted filename= must remain ASCII so net/http accepts the header.
	start := strings.Index(got, `filename="`)
	if start < 0 {
		t.Fatalf("disposition = %q, missing filename=", got)
	}
	rest := got[start+len(`filename="`):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatalf("disposition = %q, unclosed filename=", got)
	}
	quoted := rest[:end]
	for _, r := range quoted {
		if r < 0x20 || r > 0x7e {
			t.Fatalf("filename=%q contains non-ASCII rune %U", quoted, r)
		}
	}
	if !strings.Contains(quoted, ".png") {
		t.Fatalf("filename=%q, want retained .png extension", quoted)
	}
}
