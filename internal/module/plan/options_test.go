// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package plan

import "testing"

func TestApplyBuildOptionsNilAndSkipWebShell(t *testing.T) {
	WithSkipWebShell(true)(nil)
	WithWebShellProfile(WebShellProfileMinimal)(nil)

	out := applyBuildOptions(nil)
	if out.SkipWebShell {
		t.Fatal("expected zero options")
	}
	if out.WebShellProfile != "" {
		t.Fatalf("expected empty profile, got %q", out.WebShellProfile)
	}
	out = applyBuildOptions([]BuildOption{nil, WithSkipWebShell(true), WithWebShellProfile(WebShellProfileFull)})
	if !out.SkipWebShell {
		t.Fatal("expected SkipWebShell")
	}
	if out.WebShellProfile != WebShellProfileFull {
		t.Fatalf("profile=%q, want full", out.WebShellProfile)
	}
}

func TestWithWebShellProfile(t *testing.T) {
	out := applyBuildOptions([]BuildOption{WithWebShellProfile(WebShellProfileMinimal)})
	if out.WebShellProfile != WebShellProfileMinimal {
		t.Fatalf("got %q", out.WebShellProfile)
	}
}

func TestParseWebShellProfile(t *testing.T) {
	cases := []struct {
		in   string
		want WebShellProfile
		ok   bool
	}{
		{"", WebShellProfileStandard, true},
		{"standard", WebShellProfileStandard, true},
		{"STANDARD", WebShellProfileStandard, true},
		{"minimal", WebShellProfileMinimal, true},
		{" full ", WebShellProfileFull, true},
		{"tiny", "", false},
	}
	for _, tc := range cases {
		got, err := ParseWebShellProfile(tc.in)
		if tc.ok {
			if err != nil {
				t.Fatalf("ParseWebShellProfile(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParseWebShellProfile(%q)=%q, want %q", tc.in, got, tc.want)
			}
			continue
		}
		if err == nil {
			t.Fatalf("ParseWebShellProfile(%q) expected error", tc.in)
		}
	}
}
