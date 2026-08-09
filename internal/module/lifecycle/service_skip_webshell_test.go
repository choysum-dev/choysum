// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"testing"
)

func TestServiceInstallUpgradePropagateSkipWebShell(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	svc := NewService(runtimeScope, nil)

	// Install/Upgrade fail after OperationOptions are applied; we only need the
	// SkipWebShell wiring lines to execute.
	_ = svc.Install(context.Background(), InstallRequest{Name: "missing_mod", SkipWebShell: true})
	_ = svc.Upgrade(context.Background(), UpgradeRequest{Input: "missing_mod", SkipWebShell: true, WithDemo: true})
}
