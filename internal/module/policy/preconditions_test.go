// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package policy

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestRequireInstalledForUpgrade(t *testing.T) {
	if err := RequireInstalledForUpgrade(nil); err == nil || !strings.Contains(err.Error(), "module is nil") {
		t.Fatalf("RequireInstalledForUpgrade(nil) error = %v", err)
	}

	for _, status := range []meta.Status{meta.Installed, meta.ToUpgrade} {
		mod := &meta.Module{Name: "auth", Status: status}
		if err := RequireInstalledForUpgrade(mod); err != nil {
			t.Fatalf("RequireInstalledForUpgrade(%s) error = %v", status, err)
		}
	}

	err := RequireInstalledForUpgrade(&meta.Module{Name: "auth", Status: meta.Uninstalled})
	if err == nil || !strings.Contains(err.Error(), "module auth is not installed") {
		t.Fatalf("RequireInstalledForUpgrade(uninstalled) error = %v", err)
	}
}
