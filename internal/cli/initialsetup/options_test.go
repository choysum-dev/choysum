// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initialsetup

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseMode(t *testing.T) {
	cases := map[string]Mode{
		"":           ModeUnset,
		"  ":         ModeUnset,
		"complete":   ModeComplete,
		" Complete ": ModeComplete,
		"wizard":     ModeWizard,
		"WIZARD":     ModeWizard,
	}
	for raw, want := range cases {
		got, err := ParseMode(raw)
		if err != nil || got != want {
			t.Fatalf("ParseMode(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
	for _, raw := range []string{"auto", "skip", "true"} {
		if _, err := ParseMode(raw); err == nil {
			t.Fatalf("ParseMode(%q) expected error", raw)
		}
	}
}

func TestResolveOptionsPasswordPrecedence(t *testing.T) {
	var warn bytes.Buffer
	opts, err := ResolveOptions("", "from-flag", true, "from-env", &warn)
	if err != nil {
		t.Fatalf("ResolveOptions: %v", err)
	}
	if opts.Password != "from-flag" || opts.PasswordSource != PasswordFlag {
		t.Fatalf("flag should win: %+v", opts)
	}
	if !strings.Contains(warn.String(), "--admin-password is visible") {
		t.Fatalf("flag password must warn, got %q", warn.String())
	}

	warn.Reset()
	opts, err = ResolveOptions("complete", "", false, "from-env", &warn)
	if err != nil {
		t.Fatalf("ResolveOptions: %v", err)
	}
	if opts.Mode != ModeComplete || opts.Password != "from-env" || opts.PasswordSource != PasswordEnvVar {
		t.Fatalf("env password: %+v", opts)
	}
	if warn.Len() != 0 {
		t.Fatalf("env password must not warn, got %q", warn.String())
	}

	opts, err = ResolveOptions("", "", false, "", nil)
	if err != nil {
		t.Fatalf("ResolveOptions: %v", err)
	}
	opts, err = ResolveOptions("", "", false, " \t", nil)
	if err != nil || opts.PasswordSource != PasswordNone {
		t.Fatalf("whitespace-only env password must count as unset: %#v, %v", opts, err)
	}
	if opts.PasswordSource != PasswordNone || opts.Password != "" {
		t.Fatalf("no password: %+v", opts)
	}
}

func TestResolveOptionsRejects(t *testing.T) {
	cases := []struct {
		name     string
		setup    string
		flagPw   string
		flagSet  bool
		envPw    string
		contains string
	}{
		{name: "invalid mode", setup: "auto", contains: "invalid --setup"},
		{name: "empty flag password", flagSet: true, contains: "must not be empty"},
		{name: "whitespace flag password", flagSet: true, flagPw: " \t", contains: "must not be empty"},
		{name: "wizard with flag password", setup: "wizard", flagPw: "x", flagSet: true, contains: "--admin-password"},
		{name: "wizard with env password", setup: "wizard", envPw: "x", contains: PasswordEnv},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var warn bytes.Buffer
			_, err := ResolveOptions(tc.setup, tc.flagPw, tc.flagSet, tc.envPw, &warn)
			if err == nil || !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("err = %v, want containing %q", err, tc.contains)
			}
			if warn.Len() != 0 {
				t.Fatalf("rejected options must not warn, got %q", warn.String())
			}
		})
	}
}
