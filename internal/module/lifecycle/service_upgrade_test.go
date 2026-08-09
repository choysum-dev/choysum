// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type skipWebShellPeekOrigin struct {
	skipWebShell bool
	peeked       bool
}

func (o *skipWebShellPeekOrigin) Peek(ctx context.Context, _ string) (*meta.Module, error) {
	o.peeked = true
	o.skipWebShell = OperationOptionsFromContext(ctx).SkipWebShell
	return nil, errors.New("stop after peek")
}

func (*skipWebShellPeekOrigin) ResolveInstallModule(context.Context, string) (*meta.Module, error) {
	return nil, errors.New("not implemented")
}

func (*skipWebShellPeekOrigin) Fetch(context.Context, string) (*meta.Module, error) {
	return nil, errors.New("not implemented")
}

func (*skipWebShellPeekOrigin) Purge(context.Context, string) error {
	return errors.New("not implemented")
}

func TestServiceUpgradeAppliesSkipWebShellOptions(t *testing.T) {
	runtimeScope := newLifecycleCommitTestScope(t)
	origin := &skipWebShellPeekOrigin{}
	svc := NewService(runtimeScope, nil, WithOriginCoordinatorFactory(func(scope.Scope) OriginCoordinator {
		return origin
	}))

	// Registry Peek runs before lease and receives the operation context.
	err := svc.Upgrade(context.Background(), UpgradeRequest{
		Input:        "probe@1.0.0",
		WithDemo:     true,
		SkipWebShell: true,
	})
	if err == nil {
		t.Fatal("expected Upgrade to fail after Peek")
	}
	if !origin.peeked {
		t.Fatal("expected OriginCoordinator.Peek to observe the upgrade context")
	}
	if !origin.skipWebShell {
		t.Fatal("expected OperationOptions.SkipWebShell=true on Peek context")
	}
}
