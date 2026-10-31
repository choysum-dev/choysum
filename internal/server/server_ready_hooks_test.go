// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/server/transport"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type readyHooksDBScope struct {
	*serverTestScope
	session *scope.Session
}

func (s *readyHooksDBScope) Session() *scope.Session { return s.session }

type nilContextReadyHooksScope struct {
	*readyHooksDBScope
}

func (s *nilContextReadyHooksScope) Context() context.Context { return nil }

func seedReadyHooksBootstrapAdmin(t *testing.T, db *gorm.DB) {
	t.Helper()
	userID := "ready-admin-" + xid.New().String()
	companyID := "ready-co-" + xid.New().String()
	if err := db.AutoMigrate(&meta.Module{}, &modmeta.ModelData{}, &meta.Model{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if err := db.Create(&meta.Module{Name: "auth", Status: meta.Installed, Version: "1.0.0"}).Error; err != nil {
		t.Fatalf("auth module: %v", err)
	}
	if err := db.Create(&modmeta.ModelData{
		Module: "auth", Name: "user_admin", Application: "auth",
		ModelName: "User", ModelId: xid.New().String(), ResID: userID,
	}).Error; err != nil {
		t.Fatalf("user_admin mapping: %v", err)
	}
	table := "auth_user_ready_hooks"
	if err := db.Create(&meta.Model{
		BaseModel:   meta.BaseModel{Id: sql.NullString{String: xid.New().String(), Valid: true}},
		Name:        "User",
		Application: "auth",
		ModelTable:  table,
		Path:        "@/auth/service/models/user.ts",
	}).Error; err != nil {
		t.Fatalf("auth.User model: %v", err)
	}
	if err := db.Exec(`CREATE TABLE "` + table + `" (id TEXT PRIMARY KEY, company_id TEXT)`).Error; err != nil {
		t.Fatalf("user table: %v", err)
	}
	if err := db.Exec(`INSERT INTO "`+table+`" (id, company_id) VALUES (?, ?)`, userID, companyID).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

func newReadyHooksDBScope(t *testing.T) *readyHooksDBScope {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ready_hooks.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	seedReadyHooksBootstrapAdmin(t, db)
	base := newRichServerTestScope(t)
	return &readyHooksDBScope{
		serverTestScope: base,
		session:         &scope.Session{DB: db},
	}
}

type readyHooksMockExec struct {
	err error
}

func (e *readyHooksMockExec) Execute(context.Context, *jsengine.JsRequest) (*jsengine.JsResponse, error) {
	if e.err != nil {
		return nil, e.err
	}
	return &jsengine.JsResponse{Result: nil}, nil
}
func (e *readyHooksMockExec) GetJsScripts() []*jsengine.JsScript         { return nil }
func (e *readyHooksMockExec) SetJsScripts(scripts []*jsengine.JsScript)  {}
func (e *readyHooksMockExec) Reload(scripts ...*jsengine.JsScript) error { return nil }
func (e *readyHooksMockExec) AppendJsScripts(scripts ...*jsengine.JsScript) {}
func (e *readyHooksMockExec) Start() error                                   { return nil }
func (e *readyHooksMockExec) Stop() error                                    { return nil }

func TestRunApplicationReadyHooksEarlyReturns(t *testing.T) {
	t.Parallel()
	if err := (*GRPCWebServer)(nil).runApplicationReadyHooks(context.Background()); err != nil {
		t.Fatalf("nil server: %v", err)
	}
	srv := &GRPCWebServer{jsExecutor: &readyHooksMockExec{}}
	if err := srv.runApplicationReadyHooks(context.Background()); err != nil {
		t.Fatalf("nil scope: %v", err)
	}
	srv = &GRPCWebServer{runtimeScope: newRichServerTestScope(t)}
	if err := srv.runApplicationReadyHooks(context.Background()); err != nil {
		t.Fatalf("nil executor: %v", err)
	}
	srv = &GRPCWebServer{runtimeScope: &noSessionServerScope{serverTestScope: newRichServerTestScope(t)}, jsExecutor: &readyHooksMockExec{}}
	if err := srv.runApplicationReadyHooks(context.Background()); err != nil {
		t.Fatalf("nil session: %v", err)
	}
}

func TestRunApplicationReadyHooksNilSessionDB(t *testing.T) {
	t.Parallel()
	srv := &GRPCWebServer{
		runtimeScope: newRichServerTestScope(t),
		jsExecutor:   &readyHooksMockExec{},
	}
	if err := srv.runApplicationReadyHooks(context.Background()); err != nil {
		t.Fatalf("empty session db: %v", err)
	}
}

func TestRunApplicationReadyHooksFindError(t *testing.T) {
	t.Parallel()
	runtimeScope := newReadyHooksDBScope(t)
	sqlDB, err := runtimeScope.session.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		jsExecutor:   &readyHooksMockExec{},
		runState:     runState{applicationNames: []string{"document"}},
	}
	err = srv.runApplicationReadyHooks(context.Background())
	if err == nil || !strings.Contains(err.Error(), "load installed modules for ready hooks") {
		t.Fatalf("err=%v", err)
	}
}

func writeReadyHookModule(t *testing.T, db *gorm.DB, root string) {
	t.Helper()
	hookPath := filepath.Join(root, "service", "hook", "ready.ts")
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hookPath, []byte("@HookReady()\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&meta.Module{
		Name: "doc", Status: meta.Installed, Version: "1.0.0",
		ApplicationStr: "document", Path: root,
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestRunApplicationReadyHooksInjectsGRPCDialer(t *testing.T) {
	t.Parallel()
	runtimeScope := newReadyHooksDBScope(t)
	root := t.TempDir()
	writeReadyHookModule(t, runtimeScope.session.DB, root)
	srv := &GRPCWebServer{
		runtimeScope:   runtimeScope,
		jsExecutor:     &readyHooksMockExec{},
		grpcClientPool: &transport.GRPCClientPool{},
		runState:       runState{applicationNames: []string{"document"}},
	}
	if err := srv.runApplicationReadyHooks(context.Background()); err != nil {
		t.Fatalf("run with grpc pool: %v", err)
	}
}

func TestRunApplicationReadyHooksNilContextFallback(t *testing.T) {
	t.Parallel()
	runtimeScope := &nilContextReadyHooksScope{readyHooksDBScope: newReadyHooksDBScope(t)}
	root := t.TempDir()
	writeReadyHookModule(t, runtimeScope.session.DB, root)
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		jsExecutor:   &readyHooksMockExec{},
		runState:     runState{applicationNames: []string{"document"}},
	}
	if err := srv.runApplicationReadyHooks(nil); err != nil {
		t.Fatalf("nil ctx fallback: %v", err)
	}
}

func TestRunApplicationReadyHooksSuccess(t *testing.T) {
	t.Parallel()
	runtimeScope := newReadyHooksDBScope(t)
	root := t.TempDir()
	writeReadyHookModule(t, runtimeScope.session.DB, root)
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		jsExecutor:   &readyHooksMockExec{},
		runState:     runState{applicationNames: []string{"document"}},
	}
	if err := srv.runApplicationReadyHooks(nil); err != nil {
		t.Fatalf("success path: %v", err)
	}
}

func TestRunApplicationReadyHooksRunReadyError(t *testing.T) {
	t.Parallel()
	runtimeScope := newReadyHooksDBScope(t)
	root := t.TempDir()
	writeReadyHookModule(t, runtimeScope.session.DB, root)
	srv := &GRPCWebServer{
		runtimeScope: runtimeScope,
		jsExecutor:   &readyHooksMockExec{err: errors.New("HOOK_UNSUPPORTED: ready failed")},
		runState:     runState{applicationNames: []string{"document"}},
	}
	err := srv.runApplicationReadyHooks(context.Background())
	if err == nil || !strings.Contains(err.Error(), "application ready hooks") {
		t.Fatalf("err=%v", err)
	}
}
