// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initialsetup

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestGeneratePasswordShape(t *testing.T) {
	shape := regexp.MustCompile(`^[` + passwordAlphabet + `]{4}(-[` + passwordAlphabet + `]{4}){4}$`)
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		pw, err := GeneratePassword(nil)
		if err != nil {
			t.Fatalf("GeneratePassword: %v", err)
		}
		if len(pw) != 24 || !shape.MatchString(pw) {
			t.Fatalf("unexpected password shape %q", pw)
		}
		seen[pw] = true
	}
	if len(seen) < 50 {
		t.Fatalf("expected unique passwords, got %d distinct of 50", len(seen))
	}
}

func TestGeneratePasswordAlphabetAvoidsLookAlikes(t *testing.T) {
	for _, c := range "0O1lI" {
		if strings.ContainsRune(passwordAlphabet, c) {
			t.Fatalf("alphabet must not contain %q", c)
		}
	}
}

func TestGeneratePasswordSourceError(t *testing.T) {
	if _, err := GeneratePassword(bytes.NewReader(nil)); err == nil {
		t.Fatal("expected error from exhausted random source")
	}
	if _, err := GeneratePassword(failingReader{}); err == nil {
		t.Fatal("expected error from failing random source")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestClientHashMatchesWebClient(t *testing.T) {
	// Pinned against hashPasswordClient in modules/auth/web/stores/auth/utils.test.ts.
	cases := []struct{ password, username, want string }{
		{"secret", "admin", "$CH$3a0f774e9e3410187269e9d493fe067af71fb547aedfa8a8d798cd29f340393b"},
		{"密码", "管理员", "$CH$80158961a09b8473f752dbfb39352e9891e6ccc08f3295f5840578c6f1760bd3"},
	}
	for _, tc := range cases {
		if got := ClientHash(tc.password, tc.username); got != tc.want {
			t.Fatalf("ClientHash(%q, %q) = %q, want %q", tc.password, tc.username, got, tc.want)
		}
	}
}
