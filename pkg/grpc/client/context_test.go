// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"context"
	"testing"

	"google.golang.org/grpc"
)

func TestContextDialerHelpers(t *testing.T) {
	base := context.Background()
	if _, ok := ServiceDialerFromContext(base); ok {
		t.Fatal("expected no dialer in base context")
	}
	ctx := ContextWithServiceDialer(nil, nil)
	if ctx == nil {
		t.Fatal("expected nil context helper to default to background")
	}
	dialer := func(context.Context, string) (*grpc.ClientConn, error) { return nil, nil }
	ctx = ContextWithServiceDialer(base, dialer)
	got, ok := ServiceDialerFromContext(ctx)
	if !ok || got == nil {
		t.Fatal("expected dialer from context")
	}
	if ContextWithServiceDialer(base, nil) != base {
		t.Fatal("expected nil dialer to leave context unchanged")
	}
	if _, ok := ServiceDialerFromContext(nil); ok {
		t.Fatal("expected nil context to have no dialer")
	}
}
