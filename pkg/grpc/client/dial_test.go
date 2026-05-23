// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"
)

func TestDial(t *testing.T) {
	if _, err := Dial(context.Background(), "bad-name"); err == nil {
		t.Fatal("expected invalid service name error")
	}

	if _, err := Dial(context.Background(), "auth.User"); err == nil {
		t.Fatal("expected missing dialer error")
	}

	expected := errors.New("network down")
	ctx := ContextWithServiceDialer(context.Background(), func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if serviceName != "auth.User" {
			t.Fatalf("service name = %q, want auth.User", serviceName)
		}
		return nil, expected
	})
	if _, err := Dial(ctx, "auth.User"); err == nil || !strings.Contains(err.Error(), "dial auth.User") || !strings.Contains(err.Error(), "network down") {
		t.Fatalf("expected wrapped dial error, got %v", err)
	}

	ctx = ContextWithServiceDialer(context.Background(), func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		return nil, nil
	})
	conn, err := Dial(ctx, "auth.User")
	if err != nil {
		t.Fatalf("Dial unexpected error: %v", err)
	}
	if conn != nil {
		t.Fatalf("expected nil connection from stub dialer, got %#v", conn)
	}
}
