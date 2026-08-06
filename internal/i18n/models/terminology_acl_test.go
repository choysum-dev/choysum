// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package models

import "testing"

func TestEnsureTerminologyEditorAllowsEarlyReturns(t *testing.T) {
	if err := EnsureTerminologyEditorAllows(nil, "auth"); err != nil {
		t.Fatalf("nil scope: %v", err)
	}
	if err := EnsureTerminologyEditorAllows(newTestScope(t), "core"); err != nil {
		t.Fatalf("core: %v", err)
	}
	if err := EnsureTerminologyEditorAllows(newTestScope(t), ""); err != nil {
		t.Fatalf("empty app: %v", err)
	}
	if err := EnsureTerminologyEditorAllows(newTestScope(t), "  "); err != nil {
		t.Fatalf("whitespace app: %v", err)
	}
	// No effective catalog / role tables: should no-op rather than error.
	if err := EnsureTerminologyEditorAllows(newTestScope(t), "auth"); err != nil {
		t.Fatalf("auth without catalog: %v", err)
	}
}
