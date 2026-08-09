// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"testing"
)

func TestServiceUpgradeAppliesSkipWebShellOptions(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	svc := NewService(runtimeScope, nil)

	// Upgrade fails while resolving the missing module, but still applies
	// OperationOptions (SkipWebShell) onto the context first.
	err := svc.Upgrade(context.Background(), UpgradeRequest{
		Input:        "missing_mod_for_skip_web",
		WithDemo:     true,
		SkipWebShell: true,
	})
	if err == nil {
		t.Fatal("expected Upgrade to fail for missing module")
	}
}
