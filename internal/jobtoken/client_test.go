// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jobtoken

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestIssueTaskJobToken(t *testing.T) {
	runtimeScope := newJobTokenScope("development")
	runtimeScope.cfg.Auth.InternalKey = "secret"
	authenticator := &fakeAuthenticator{issueToken: "grpc-token", issueExp: 777}
	service := NewService(runtimeScope, authenticator)
	desc, err := service.ServiceDesc()
	if err != nil {
		t.Fatalf("ServiceDesc error: %v", err)
	}

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	server.RegisterService(desc, service)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	var conn *grpc.ClientConn
	dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if serviceName != ServiceFullName() {
			return nil, errors.New("unexpected service")
		}
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(ctx, "bufnet",
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, err
		}
		conn = c
		return conn, nil
	}
	t.Cleanup(func() {
		if conn != nil {
			_ = conn.Close()
		}
	})

	resp, err := IssueTaskJobToken(context.Background(), runtimeScope.cfg.Auth, runtimeScope.cfg.Server.Environment, dialer, IssueRequest{
		JobId:             "job-3",
		TargetApp:         "auth",
		FullMethod:        "/auth.Task/Run",
		SchedulerUserId:   "u-5",
		TriggeredByUserId: "u-6",
		Attempt:           3,
		TTL:               3 * time.Second,
	})
	if err != nil {
		t.Fatalf("IssueTaskJobToken error: %v", err)
	}
	if resp.AccessToken != "grpc-token" || resp.ExpiresAt != 777 {
		t.Fatalf("unexpected issue response: %#v", resp)
	}
	if authenticator.lastUserID != "u-5" || authenticator.lastTTL != 3*time.Second {
		t.Fatalf("unexpected authenticator call: user=%q ttl=%s meta=%#v", authenticator.lastUserID, authenticator.lastTTL, authenticator.lastMeta)
	}

	if _, err := IssueTaskJobToken(context.Background(), runtimeScope.cfg.Auth, runtimeScope.cfg.Server.Environment, func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		return nil, errors.New("dial failed")
	}, IssueRequest{}); err == nil {
		t.Fatal("expected dial failure to be returned")
	}
}
