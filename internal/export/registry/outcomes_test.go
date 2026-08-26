// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry_test

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/export/registry"
)

func TestResult_HasOutcomes(t *testing.T) {
	if (registry.Result{}).HasOutcomes() {
		t.Fatal("zero result should not have outcomes")
	}
	if !(registry.Result{Outcomes: registry.Outcomes{Total: 1}}).HasOutcomes() {
		t.Fatal("non-zero total should have outcomes")
	}
}
