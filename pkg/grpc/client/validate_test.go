// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"strings"
	"testing"
)

func TestValidateServiceName(t *testing.T) {
	validNames := []string{"auth.User", "A1.B2", "Service"}
	for _, name := range validNames {
		if err := ValidateServiceName(name); err != nil {
			t.Fatalf("ValidateServiceName(%q) unexpected error: %v", name, err)
		}
	}

	invalidNames := []string{"", ".bad", "bad-Name", strings.Repeat("a", MaxServiceNameLen+1)}
	for _, name := range invalidNames {
		if err := ValidateServiceName(name); err == nil {
			t.Fatalf("ValidateServiceName(%q) expected error", name)
		}
	}
}
