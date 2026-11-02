// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package setupgate

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type gateTestScope struct {
	ctx context.Context
	db  *gorm.DB
}

func (e *gateTestScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *gateTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *gateTestScope) Session() *scope.Session {
	if e == nil || e.db == nil {
		return nil
	}
	return &scope.Session{DB: e.db}
}
func (e *gateTestScope) WithContext(ctx context.Context) scope.Scope {
	return &gateTestScope{ctx: ctx, db: e.db}
}
func (e *gateTestScope) Context() context.Context {
	if e != nil && e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}
func (e *gateTestScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (e *gateTestScope) Config() *config.Config { return &config.Config{} }
func (e *gateTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func newGateScope(t *testing.T) (*gateTestScope, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "setupgate.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return &gateTestScope{ctx: context.Background(), db: db}, db
}

func mustExec(t *testing.T, db *gorm.DB, sql string, values ...any) {
	t.Helper()
	if err := db.Exec(sql, values...).Error; err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func seedSetupPending(t *testing.T, db *gorm.DB) {
	t.Helper()
	mustExec(t, db, `CREATE TABLE meta_module (
		id TEXT PRIMARY KEY, name TEXT, status TEXT, deleted_at DATETIME,
		created_at DATETIME, updated_at DATETIME
	)`)
	mustExec(t, db, `INSERT INTO meta_module (id, name, status, deleted_at, created_at, updated_at)
		VALUES (?, ?, ?, NULL, ?, ?)`, "mod-auth", "auth", string(meta.Installed), time.Now(), time.Now())
	mustExec(t, db, `CREATE TABLE auth_app_setting (
		id TEXT PRIMARY KEY, key TEXT NOT NULL UNIQUE, value TEXT,
		created_at DATETIME, updated_at DATETIME
	)`)
}

func seedReady(t *testing.T, db *gorm.DB) {
	t.Helper()
	seedSetupPending(t, db)
	mustExec(t, db, `INSERT INTO auth_app_setting (id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"s1", "setup.completed_at", time.Now().UTC().Format(time.RFC3339), time.Now(), time.Now())
}

func TestUnaryAllowsCompleteInitialSetupWhenSetupPending(t *testing.T) {
	runtimeScope, db := newGateScope(t)
	seedSetupPending(t, db)

	called := false
	interceptor := UnaryInterceptor(runtimeScope)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/auth.User/CompleteInitialSetup"},
		func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "ok", nil
		})
	if err != nil {
		t.Fatalf("CompleteInitialSetup: %v", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestUnaryRejectsLoginWhenSetupPending(t *testing.T) {
	runtimeScope, db := newGateScope(t)
	seedSetupPending(t, db)

	interceptor := UnaryInterceptor(runtimeScope)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/auth.User/Login"},
		func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler must not run")
			return nil, nil
		})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition; err=%v", status.Code(err), err)
	}
}

func TestUnaryRejectsRegisterAndRefreshWhenSetupPending(t *testing.T) {
	runtimeScope, db := newGateScope(t)
	seedSetupPending(t, db)
	interceptor := UnaryInterceptor(runtimeScope)

	for _, method := range []string{"/auth.User/Register", "/auth.User/RefreshTokens"} {
		_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: method},
			func(ctx context.Context, req interface{}) (interface{}, error) {
				t.Fatalf("%s handler must not run", method)
				return nil, nil
			})
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("%s: code = %v, want FailedPrecondition; err=%v", method, status.Code(err), err)
		}
	}
}

func TestUnaryAllowsLoginWhenReady(t *testing.T) {
	runtimeScope, db := newGateScope(t)
	seedReady(t, db)

	called := false
	interceptor := UnaryInterceptor(runtimeScope)
	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/auth.User/Login"},
		func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "token", nil
		})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if !called || resp != "token" {
		t.Fatalf("called=%v resp=%v", called, resp)
	}
}

func TestUnarySkipsGateWhenDepthNonZero(t *testing.T) {
	runtimeScope, db := newGateScope(t)
	seedSetupPending(t, db)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-choysum-depth", "1"))
	called := false
	interceptor := UnaryInterceptor(runtimeScope)
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/auth.User/Login"},
		func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			return "ok", nil
		})
	if err != nil {
		t.Fatalf("nested Login: %v", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestUnaryRejectsLoginWhenEmpty(t *testing.T) {
	runtimeScope, _ := newGateScope(t)
	interceptor := UnaryInterceptor(runtimeScope)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/auth.User/Login"},
		func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler must not run")
			return nil, nil
		})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition; err=%v", status.Code(err), err)
	}
}

type testServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *testServerStream) Context() context.Context { return s.ctx }

func TestStreamRejectsLoginWhenSetupPending(t *testing.T) {
	runtimeScope, db := newGateScope(t)
	seedSetupPending(t, db)

	err := StreamInterceptor(runtimeScope)(nil, &testServerStream{ctx: context.Background()},
		&grpc.StreamServerInfo{FullMethod: "/auth.User/Login"},
		func(srv interface{}, ss grpc.ServerStream) error {
			t.Fatal("handler must not run")
			return nil
		})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition; err=%v", status.Code(err), err)
	}
}
