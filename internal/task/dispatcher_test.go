// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/jobtoken"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testScope struct {
	ctx    context.Context
	db     *gorm.DB
	logger *slog.Logger
	cfg    *config.Config
}

func (e *testScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *testScope) Transactor() scope.Transactor                      { return scopetest.NewPassthroughTransactor(e) }
func (e *testScope) Session() *scope.Session                           { return &scope.Session{DB: e.db} }
func (e *testScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *testScope) Context() context.Context { return e.ctx }
func (e *testScope) Logger() *slog.Logger     { return e.logger }
func (e *testScope) Config() *config.Config   { return e.cfg }
func (e *testScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

type fakeAuthenticator struct{}

func (f *fakeAuthenticator) ValidateToken(ctx context.Context, token string, tokenType auth.TokenType, checkRevoked bool) (auth.Identity, error) {
	return nil, status.Error(codes.Unauthenticated, "not implemented")
}
func (f *fakeAuthenticator) CreateTokens(ctx context.Context, userID string, metadata map[string]interface{}) (*auth.TokenPair, error) {
	return &auth.TokenPair{AccessToken: "token"}, nil
}
func (f *fakeAuthenticator) RefreshTokens(ctx context.Context, refreshToken string, metadata map[string]interface{}) (*auth.TokenPair, error) {
	return &auth.TokenPair{AccessToken: "token"}, nil
}
func (f *fakeAuthenticator) RevokeToken(ctx context.Context, token string, reason string) error {
	return nil
}
func (f *fakeAuthenticator) RevokeAllUserTokens(ctx context.Context, userID string, exceptTokenID string, reason string) (int, error) {
	return 0, nil
}
func (f *fakeAuthenticator) Close() error { return nil }

func (f *fakeAuthenticator) CreateAccessTokenWithTTL(ctx context.Context, userID string, metadata map[string]interface{}, ttl time.Duration) (string, int64, error) {
	return "token", time.Now().Add(ttl).Unix(), nil
}

func TestDispatcherUnauthenticatedSelfHeal(t *testing.T) {
	ctx := context.Background()

	dsn := "file:task_dispatcher_self_heal?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{
		Auth:   &config.AuthConfig{InternalKey: "test"},
		Server: &config.ServerConfig{Environment: "development"},
		Task:   config.NewDefaultTaskConfig(),
	}
	cfg.Task.Dispatch.JobTokenTTLms = 1000

	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	var executeCalls atomic.Int32
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()

	jobtokenSvc := jobtoken.NewService(runtimeScope, &fakeAuthenticator{})
	svcDesc, err := jobtokenSvc.ServiceDesc()
	if err != nil {
		t.Fatalf("jobtoken service desc: %v", err)
	}
	server.RegisterService(svcDesc, jobtokenSvc)

	methodDesc, err := taskWorkerMethod("auth")
	if err != nil {
		t.Fatalf("task worker method: %v", err)
	}
	serviceDesc := &grpc.ServiceDesc{
		ServiceName: "auth.TaskWorker",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "ExecuteJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					if interceptor != nil {
						info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/auth.TaskWorker/ExecuteJob"}
						return interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
							return nil, status.Error(codes.Internal, "unexpected interceptor path")
						})
					}
					reqMsg := dynamicpb.NewMessage(methodDesc.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					if executeCalls.Add(1) == 1 {
						return nil, status.Error(codes.Unauthenticated, "unauth")
					}
					respMsg := dynamicpb.NewMessage(methodDesc.Output())
					fields := respMsg.Descriptor().Fields()
					if f := fields.ByName("status"); f != nil {
						respMsg.Set(f, protoreflect.ValueOfEnum(1))
					}
					return respMsg, nil
				},
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "test",
	}
	server.RegisterService(serviceDesc, nil)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	var conn *grpc.ClientConn
	dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		conn = c
		return conn, nil
	}

	job := Job{
		Id:                "job1",
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		Status:            "queued",
		RunAfter:          time.Now().UTC().Add(-time.Minute),
		Attempt:           0,
		MaxAttempts:       3,
		TimeoutMs:         0,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	d := NewDispatcher(runtimeScope, client.ServiceDialer(dialer))
	d.handleJob(job.Id)

	var got Job
	if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if got.Status != "succeeded" {
		t.Fatalf("status: want succeeded, got %s", got.Status)
	}
	if got.Attempt != 2 {
		t.Fatalf("attempt: want 2, got %d", got.Attempt)
	}
	if executeCalls.Load() != 2 {
		t.Fatalf("execute calls: want 2, got %d", executeCalls.Load())
	}
}

func TestDispatcherUnauthenticatedFailFast(t *testing.T) {
	ctx := context.Background()

	dsn := "file:task_dispatcher_fail_fast?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{
		Auth:   &config.AuthConfig{InternalKey: "test"},
		Server: &config.ServerConfig{Environment: "development"},
		Task:   config.NewDefaultTaskConfig(),
	}

	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()

	jobtokenSvc := jobtoken.NewService(runtimeScope, &fakeAuthenticator{})
	svcDesc, err := jobtokenSvc.ServiceDesc()
	if err != nil {
		t.Fatalf("jobtoken service desc: %v", err)
	}
	server.RegisterService(svcDesc, jobtokenSvc)

	methodDesc, err := taskWorkerMethod("auth")
	if err != nil {
		t.Fatalf("task worker method: %v", err)
	}
	serviceDesc := &grpc.ServiceDesc{
		ServiceName: "auth.TaskWorker",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "ExecuteJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					reqMsg := dynamicpb.NewMessage(methodDesc.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					return nil, status.Error(codes.Unauthenticated, "unauth")
				},
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "test",
	}
	server.RegisterService(serviceDesc, nil)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	var conn *grpc.ClientConn
	dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		conn = c
		return conn, nil
	}

	job := Job{
		Id:                "job2",
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		Status:            "queued",
		RunAfter:          time.Now().UTC().Add(-time.Minute),
		Attempt:           0,
		MaxAttempts:       2,
		TimeoutMs:         0,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	d := NewDispatcher(runtimeScope, client.ServiceDialer(dialer))
	d.handleJob(job.Id)

	var got Job
	if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if got.Status != "failed" {
		t.Fatalf("status: want failed, got %s", got.Status)
	}
}

func TestDocumentAttachmentGCScheduleDispatchFlow(t *testing.T) {
	ctx := context.Background()
	fixedNow := time.Date(2026, 1, 20, 12, 0, 30, 0, time.UTC)

	dsn := "file:task_storage_attachment_gc_flow?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}, &Schedule{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{
		Auth:   &config.AuthConfig{InternalKey: "test"},
		Server: &config.ServerConfig{Environment: "development"},
		Db:     &config.DbConfig{Dialect: "sqlite"},
		Task:   config.NewDefaultTaskConfig(),
	}
	cfg.Task.Dispatch.JobTokenTTLms = 1000

	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	due := fixedNow.Add(-2 * time.Minute)
	schedule := Schedule{
		Id:                "sch-document-gc",
		Active:            true,
		Name:              "document.attachment.gc",
		TargetApp:         "document",
		FullMethod:        "document.AttachmentContent/RunGarbageCollection",
		PayloadTemplate:   []byte(`{}`),
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		CronExpr:          "*/5 * * * *",
		Timezone:          "UTC",
		NextRunAt:         &due,
		CreatedAt:         fixedNow,
		UpdatedAt:         fixedNow,
	}
	if err := db.Create(&schedule).Error; err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	s := NewScheduler(runtimeScope)
	s.processSchedule(db, &schedule, fixedNow)

	var queued []Job
	if err := db.Where("target_app = ?", "document").Order("created_at asc").Find(&queued).Error; err != nil {
		t.Fatalf("load queued jobs: %v", err)
	}
	if len(queued) != 1 {
		t.Fatalf("expected one queued job, got %d", len(queued))
	}
	if queued[0].FullMethod != "document.AttachmentContent/RunGarbageCollection" {
		t.Fatalf("queued job full method = %q", queued[0].FullMethod)
	}

	var executeCalls atomic.Int32
	var calledMethod atomic.Value
	calledMethod.Store("")

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()

	jobtokenSvc := jobtoken.NewService(runtimeScope, &fakeAuthenticator{})
	svcDesc, err := jobtokenSvc.ServiceDesc()
	if err != nil {
		t.Fatalf("jobtoken service desc: %v", err)
	}
	server.RegisterService(svcDesc, jobtokenSvc)

	methodDesc, err := taskWorkerMethod("document")
	if err != nil {
		t.Fatalf("task worker method: %v", err)
	}
	serviceDesc := &grpc.ServiceDesc{
		ServiceName: "document.TaskWorker",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "ExecuteJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					reqMsg := dynamicpb.NewMessage(methodDesc.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					if field := reqMsg.Descriptor().Fields().ByName("full_method"); field != nil {
						calledMethod.Store(reqMsg.Get(field).String())
					}
					executeCalls.Add(1)
					respMsg := dynamicpb.NewMessage(methodDesc.Output())
					if field := respMsg.Descriptor().Fields().ByName("status"); field != nil {
						respMsg.Set(field, protoreflect.ValueOfEnum(1))
					}
					return respMsg, nil
				},
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "test",
	}
	server.RegisterService(serviceDesc, nil)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	var conn *grpc.ClientConn
	dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
		if conn != nil {
			return conn, nil
		}
		c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	d := NewDispatcher(runtimeScope, client.ServiceDialer(dialer))
	d.handleJob(queued[0].Id)

	var done Job
	if err := db.Where("id = ?", queued[0].Id).First(&done).Error; err != nil {
		t.Fatalf("load dispatched job: %v", err)
	}
	if done.Status != "succeeded" {
		t.Fatalf("job status = %s, want succeeded", done.Status)
	}
	if done.Attempt != 1 {
		t.Fatalf("job attempt = %d, want 1", done.Attempt)
	}
	if executeCalls.Load() != 1 {
		t.Fatalf("execute calls = %d, want 1", executeCalls.Load())
	}
	if got := calledMethod.Load().(string); got != "document.AttachmentContent/RunGarbageCollection" {
		t.Fatalf("worker called full method = %q", got)
	}
}

func TestDispatcherJobMetadataPersistence(t *testing.T) {
	ctx := context.Background()

	dsn := "file:task_dispatcher_metadata?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    &config.Config{Task: config.NewDefaultTaskConfig()},
	}

	job := Job{
		Id:                "job-metadata",
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		Status:            "queued",
		RunAfter:          time.Now().UTC(),
		Attempt:           0,
		MaxAttempts:       1,
		TimeoutMs:         0,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	large := strings.Repeat("x", 70*1024)

	d := NewDispatcher(runtimeScope, nil)
	d.succeedJob(db, &job, map[string]any{"payload": large})

	var got Job
	if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if got.ResultHash == "" {
		t.Fatalf("result hash should be set")
	}
	if !got.ResultTruncated {
		t.Fatalf("result should be marked truncated")
	}

	d.failJob(db, &job, map[string]any{"error": large}, map[string]any{"result": large})
	if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if got.LastErrorHash == "" {
		t.Fatalf("error hash should be set")
	}
	if !got.LastErrorTruncated {
		t.Fatalf("error should be marked truncated")
	}
	if got.ResultHash == "" {
		t.Fatalf("result hash should be set after failure")
	}
	if !got.ResultTruncated {
		t.Fatalf("result should be marked truncated after failure")
	}
}

func TestDispatcherRetryAfterNonPositiveUsesMinimum(t *testing.T) {
	ctx := context.Background()

	dsn := "file:task_dispatcher_retry_after_min?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	job := Job{
		Id:                "job-retry-min",
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		Status:            "dispatching",
		RunAfter:          time.Now().UTC(),
		Attempt:           1,
		MaxAttempts:       3,
		TimeoutMs:         0,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	d := NewDispatcher(runtimeScope, nil)
	start := time.Now().UTC()
	d.retryJob(db, &job, -10, map[string]any{"message": "bad retry"}, "retry_after")

	var got Job
	if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	delta := got.RunAfter.Sub(start)
	if delta < 900*time.Millisecond || delta > 2*time.Second {
		t.Fatalf("run_after delta: want ~1s, got %v", delta)
	}
}

func TestDispatcherRetryAfterCapped(t *testing.T) {
	ctx := context.Background()

	dsn := "file:task_dispatcher_retry_after_cap?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
	cfg.Task.Dispatch.RetryAfterMsCap = 1500
	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	job := Job{
		Id:                "job-retry-cap",
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		Status:            "dispatching",
		RunAfter:          time.Now().UTC(),
		Attempt:           1,
		MaxAttempts:       3,
		TimeoutMs:         0,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	d := NewDispatcher(runtimeScope, nil)
	start := time.Now().UTC()
	d.retryJob(db, &job, 5000, map[string]any{"message": "too long"}, "retry_after")

	var got Job
	if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	delta := got.RunAfter.Sub(start)
	if delta < 1300*time.Millisecond || delta > 2*time.Second {
		t.Fatalf("run_after delta: want ~1.5s cap, got %v", delta)
	}
}

func TestDispatcherSelectDispatchJobsFairByApp(t *testing.T) {
	cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
	cfg.Task.Dispatch.MaxConcurrencyPerApp = 1

	d := NewDispatcher(&testScope{cfg: cfg}, nil)

	jobs := []Job{
		{Id: "a1", TargetApp: "appA"},
		{Id: "a2", TargetApp: "appA"},
		{Id: "b1", TargetApp: "appB"},
		{Id: "b2", TargetApp: "appB"},
	}
	selected := d.selectDispatchJobs(jobs, 3)
	if len(selected) != 3 {
		t.Fatalf("selected: want 3 jobs, got %d", len(selected))
	}
	if selected[0].Id != "a1" || selected[1].Id != "b1" || selected[2].Id != "a2" {
		t.Fatalf("unexpected round-robin order: %+v", []string{selected[0].Id, selected[1].Id, selected[2].Id})
	}

	d.appInFly["appA"] = 1
	limited := d.selectDispatchJobs(jobs, 2)
	if len(limited) != 2 {
		t.Fatalf("selected with cap: want 2 jobs, got %d", len(limited))
	}
	if limited[0].TargetApp != "appB" || limited[1].TargetApp != "appB" {
		t.Fatalf("expected appB jobs when appA at cap, got %+v", []string{limited[0].TargetApp, limited[1].TargetApp})
	}

}

func TestDispatcherWakeupTriggersPoll(t *testing.T) {
	ctx := context.Background()

	dsn := fmt.Sprintf("file:task_dispatcher_wakeup_%d?mode=memory&cache=shared&_busy_timeout=5000", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
	cfg.Task.Dispatch.PollIntervalMs = int64(time.Hour / time.Millisecond)

	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	job := Job{
		Id:                "job-wakeup",
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		Status:            "queued",
		RunAfter:          time.Now().UTC().Add(-time.Second),
		Attempt:           0,
		MaxAttempts:       1,
		TimeoutMs:         0,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	d := NewDispatcher(runtimeScope, nil)
	d.Start()
	t.Cleanup(d.Stop)

	WakeDispatch("enqueue")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status == "dispatching" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	var got Job
	if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	t.Fatalf("status: want dispatching after wakeup, got %s", got.Status)
}

func TestDispatcherReadyQueueDedupAndLimit(t *testing.T) {
	cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
	cfg.Task.Dispatch.ReadyQueueMax = 2

	d := NewDispatcher(&testScope{cfg: cfg}, nil)

	jobs := []Job{
		{Id: "job-a", TargetApp: "appA"},
		{Id: "job-b", TargetApp: "appB"},
		{Id: "job-a", TargetApp: "appA"},
	}
	d.enqueueReady(jobs)
	if len(d.readyQ) != 2 {
		t.Fatalf("ready queue size: want 2, got %d", len(d.readyQ))
	}
	if _, ok := d.readySet["job-a"]; !ok {
		t.Fatalf("ready set should include job-a")
	}
	if _, ok := d.readySet["job-b"]; !ok {
		t.Fatalf("ready set should include job-b")
	}

	d.enqueueReady([]Job{{Id: "job-c", TargetApp: "appC"}})
	if len(d.readyQ) != 2 {
		t.Fatalf("ready queue size after max: want 2, got %d", len(d.readyQ))
	}
	if _, ok := d.readySet["job-c"]; ok {
		t.Fatalf("ready set should not include job-c when at max")
	}

	selected := d.popReadyForDispatch(1)
	if len(selected) != 1 {
		t.Fatalf("selected: want 1 job, got %d", len(selected))
	}
	if len(d.readyQ) != 1 {
		t.Fatalf("ready queue size after pop: want 1, got %d", len(d.readyQ))
	}
	if _, ok := d.readySet[selected[0].Id]; ok {
		t.Fatalf("ready set should remove selected job %s", selected[0].Id)
	}
}

func TestDispatcherRetryAfterWakeupOnShortDelay(t *testing.T) {
	ctx := context.Background()

	dsn := "file:task_dispatcher_retry_wakeup?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
	cfg.Task.Dispatch.PollIntervalMs = 1000
	runtimeScope := &testScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	job := Job{
		Id:                "job-retry-wakeup",
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		Status:            "dispatching",
		RunAfter:          time.Now().UTC(),
		Attempt:           1,
		MaxAttempts:       3,
		TimeoutMs:         0,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	var wakeups atomic.Int32
	registerDispatchWakeup(func(source string) {
		if source == "run_after" {
			wakeups.Add(1)
		}
	})
	defer clearDispatchWakeup()

	d := NewDispatcher(runtimeScope, nil)
	d.retryJob(db, &job, 500, map[string]any{"message": "retry soon"}, "retry_after")

	if wakeups.Load() != 1 {
		t.Fatalf("run_after wakeups: want 1, got %d", wakeups.Load())
	}
}

func TestDispatcherHelpers(t *testing.T) {
	t.Run("unauthenticated error wrappers", func(t *testing.T) {
		base := status.Error(codes.Unauthenticated, "unauth")
		err := unauthenticatedError{err: base}
		if !strings.Contains(err.Error(), "unauth") {
			t.Fatalf("Error() = %q, want to contain unauth", err.Error())
		}
		if !errors.Is(err.Unwrap(), base) {
			t.Fatalf("expected Unwrap() to expose base error")
		}

		empty := unauthenticatedError{}
		if empty.Error() != "unauthenticated" {
			t.Fatalf("empty Error() = %q, want unauthenticated", empty.Error())
		}
	})

	t.Run("mark cancelled updates job state", func(t *testing.T) {
		dsn := "file:task_dispatcher_mark_cancelled?mode=memory&cache=shared"
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		if err := db.AutoMigrate(&Job{}); err != nil {
			t.Fatalf("migrate: %v", err)
		}

		now := time.Now().UTC()
		job := Job{
			Id:                "job-cancelled",
			TargetApp:         "auth",
			FullMethod:        "auth.User/Login",
			SchedulerUserId:   "admin",
			TriggeredByUserId: "admin",
			Status:            "dispatching",
			RunAfter:          now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d := NewDispatcher(&testScope{ctx: context.Background(), db: db, logger: slog.New(slog.NewTextHandler(io.Discard, nil)), cfg: &config.Config{Task: config.NewDefaultTaskConfig()}}, nil)
		d.markCancelled(db, &job, "requested")

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "cancelled" || got.CancelledAt == nil || got.FinishedAt == nil {
			t.Fatalf("unexpected cancelled job state: %#v", got)
		}
		if !strings.Contains(string(got.LastErrorJson), "requested") {
			t.Fatalf("expected cancellation reason in last_error_json, got %s", string(got.LastErrorJson))
		}
	})

	t.Run("backoff and status helpers", func(t *testing.T) {
		d := NewDispatcher(&testScope{cfg: &config.Config{Task: config.NewDefaultTaskConfig()}}, nil)
		if got := d.backoffMs(0); got != 1000 {
			t.Fatalf("backoffMs(0) = %d, want 1000", got)
		}

		d.runtimeOpts.dispatchBackoffMaxMs = 1500
		for i := 1; i <= 3; i++ {
			got := d.backoffMs(i)
			if got < 1000 || got > 1500 {
				t.Fatalf("backoffMs(%d) = %d, want within [1000,1500]", i, got)
			}
		}

		for input, want := range map[any]string{
			"EXECUTE_JOB_STATUS_SUCCEEDED": "SUCCEEDED",
			" FAILED_RETRYABLE ":           "FAILED_RETRYABLE",
			int32(6):                       "CANCELLED",
			int64(5):                       "RESOURCE_BUSY",
			int(4):                         "ALREADY_RUNNING",
			float64(2):                     "FAILED_NON_RETRYABLE",
			struct{}{}:                     "{}",
		} {
			if got := normalizeExecuteJobStatus(input); got != want {
				t.Fatalf("normalizeExecuteJobStatus(%#v) = %q, want %q", input, got, want)
			}
		}

		for code, want := range map[int32]string{
			0: "EXECUTE_JOB_STATUS_UNSPECIFIED",
			1: "SUCCEEDED",
			2: "FAILED_NON_RETRYABLE",
			3: "FAILED_RETRYABLE",
			4: "ALREADY_RUNNING",
			5: "RESOURCE_BUSY",
			6: "CANCELLED",
			9: "EXECUTE_JOB_STATUS_UNSPECIFIED",
		} {
			if got := statusFromCode(code); got != want {
				t.Fatalf("statusFromCode(%d) = %q, want %q", code, got, want)
			}
		}

		if code, ok := extractGrpcCode(map[string]any{"grpc_code": float64(codes.Unauthenticated)}); !ok || code != int32(codes.Unauthenticated) {
			t.Fatalf("unexpected grpc code extraction: code=%d ok=%v", code, ok)
		}
		if _, ok := extractGrpcCode(map[string]any{"grpc_code": "bad"}); ok {
			t.Fatalf("expected invalid grpc_code type to be ignored")
		}
		if !isUnauthenticatedResponse(&ExecuteJobResponse{Error: map[string]any{"grpc_code": int32(codes.Unauthenticated)}}) {
			t.Fatalf("expected unauthenticated response to be detected")
		}
		if isUnauthenticatedResponse(&ExecuteJobResponse{Error: map[string]any{"grpc_code": int32(codes.Internal)}}) {
			t.Fatalf("did not expect non-unauthenticated response to be detected")
		}
		if isUnauthenticatedResponse(nil) {
			t.Fatalf("nil response should not be unauthenticated")
		}
	})

	t.Run("app slot accounting", func(t *testing.T) {
		cfg := &config.Config{Task: config.NewDefaultTaskConfig()}
		cfg.Task.Dispatch.MaxConcurrencyPerApp = 1
		d := NewDispatcher(&testScope{cfg: cfg}, nil)

		if !d.reserveAppSlot("auth") {
			t.Fatal("expected first app slot reservation to succeed")
		}
		if d.reserveAppSlot("auth") {
			t.Fatal("expected second app slot reservation to fail at cap")
		}
		d.releaseAppSlot("auth")
		if !d.reserveAppSlot("auth") {
			t.Fatal("expected slot reservation to succeed after release")
		}
		d.releaseAppSlot("auth")
		d.releaseAppSlot("auth")
	})

	t.Run("config helpers and no-op guards", func(t *testing.T) {
		var nilDispatcher *Dispatcher
		nilDispatcher.Stop()
		nilDispatcher.Wakeup("ignored")

		defaultDispatcher := NewDispatcher(&testScope{cfg: nil}, nil)
		if defaultDispatcher.fetchBatchSize() != 0 {
			t.Fatalf("fetchBatchSize() = %d, want 0", defaultDispatcher.fetchBatchSize())
		}
		if defaultDispatcher.readyQueueMax() != 0 {
			t.Fatalf("readyQueueMax() = %d, want 0", defaultDispatcher.readyQueueMax())
		}
		if defaultDispatcher.maxConcurrencyPerApp() != 0 {
			t.Fatalf("maxConcurrencyPerApp() = %d, want 0", defaultDispatcher.maxConcurrencyPerApp())
		}
		if defaultDispatcher.isSQLite() {
			t.Fatal("expected dispatcher without db config not to be sqlite")
		}

		cfg := &config.Config{Db: &config.DbConfig{Dialect: "SQLite"}, Task: config.NewDefaultTaskConfig()}
		cfg.Task.Dispatch.FetchBatchSize = 7
		cfg.Task.Dispatch.ReadyQueueMax = 9
		cfg.Task.Dispatch.MaxConcurrencyPerApp = 2
		d := NewDispatcher(&testScope{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)), cfg: cfg}, nil)
		if d.fetchBatchSize() != 7 {
			t.Fatalf("fetchBatchSize() = %d, want 7", d.fetchBatchSize())
		}
		if d.readyQueueMax() != 9 {
			t.Fatalf("readyQueueMax() = %d, want 9", d.readyQueueMax())
		}
		if d.maxConcurrencyPerApp() != 2 {
			t.Fatalf("maxConcurrencyPerApp() = %d, want 2", d.maxConcurrencyPerApp())
		}
		if !d.isSQLite() {
			t.Fatal("expected sqlite dialect check to ignore case")
		}

		(&Dispatcher{}).pollOnce("poll")
		d.Stop()
		d.Stop()
	})

	t.Run("handleUnauthenticated nil inputs return handled", func(t *testing.T) {
		d := NewDispatcher(&testScope{cfg: &config.Config{Task: config.NewDefaultTaskConfig()}}, nil)
		resp, handled := d.handleUnauthenticated(context.Background(), nil, nil, 1)
		if resp != nil || !handled {
			t.Fatalf("handleUnauthenticated(nil,nil) = (%#v, %v), want (nil, true)", resp, handled)
		}
	})
}

func TestDispatcherHandleUnauthenticatedPaths(t *testing.T) {
	newDispatcherTestRuntimeScope := func(t *testing.T, dsn string) (*testScope, *gorm.DB) {
		t.Helper()
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		if err := db.AutoMigrate(&Job{}); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		cfg := &config.Config{
			Auth:   &config.AuthConfig{InternalKey: "test"},
			Server: &config.ServerConfig{Environment: "development"},
			Task:   config.NewDefaultTaskConfig(),
		}
		cfg.Task.Dispatch.JobTokenTTLms = 1000
		cfg.Task.Dispatch.BackoffMaxMs = 1000
		runtimeScope := &testScope{
			ctx:    context.Background(),
			db:     db,
			logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			cfg:    cfg,
		}
		return runtimeScope, db
	}

	newQueuedJob := func(id string, maxAttempts int) Job {
		now := time.Now().UTC()
		return Job{
			Id:                id,
			TargetApp:         "auth",
			FullMethod:        "auth.User/Login",
			SchedulerUserId:   "admin",
			TriggeredByUserId: "admin",
			Status:            "dispatching",
			RunAfter:          now.Add(-time.Minute),
			Attempt:           1,
			MaxAttempts:       maxAttempts,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
	}

	t.Run("max attempts reached fails immediately", func(t *testing.T) {
		runtimeScope, db := newDispatcherTestRuntimeScope(t, "file:task_dispatcher_handle_unauth_max?mode=memory&cache=shared")
		job := newQueuedJob("job-handle-unauth-max", 1)
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d := NewDispatcher(runtimeScope, nil)
		resp, handled := d.handleUnauthenticated(context.Background(), db, &job, 1)
		if resp != nil || !handled {
			t.Fatalf("handleUnauthenticated() = (%#v, %v), want (nil, true)", resp, handled)
		}

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "failed" || got.FinishedAt == nil {
			t.Fatalf("unexpected failed job state: %#v", got)
		}
		if !strings.Contains(string(got.LastErrorJson), "unauthenticated and max attempts reached") {
			t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
		}
	})

	t.Run("token refresh issue failure retries job", func(t *testing.T) {
		runtimeScope, db := newDispatcherTestRuntimeScope(t, "file:task_dispatcher_handle_unauth_issue_fail?mode=memory&cache=shared")
		job := newQueuedJob("job-handle-unauth-issue-fail", 3)
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d := NewDispatcher(runtimeScope, client.ServiceDialer(func(context.Context, string) (*grpc.ClientConn, error) {
			return nil, errors.New("dial failed")
		}))
		d.interval = time.Second

		resp, handled := d.handleUnauthenticated(context.Background(), db, &job, 1)
		if resp != nil || !handled {
			t.Fatalf("handleUnauthenticated() = (%#v, %v), want (nil, true)", resp, handled)
		}

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "queued" {
			t.Fatalf("status = %s, want queued", got.Status)
		}
		if !got.RunAfter.After(time.Now().UTC().Add(-2 * time.Second)) {
			t.Fatalf("expected run_after to move forward, got %v", got.RunAfter)
		}
		if !strings.Contains(string(got.LastErrorJson), "issue job token failed: dial failed") {
			t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
		}
	})

	t.Run("refresh then unauthenticated fails job", func(t *testing.T) {
		runtimeScope, db := newDispatcherTestRuntimeScope(t, "file:task_dispatcher_handle_unauth_refresh_unauth?mode=memory&cache=shared")
		job := newQueuedJob("job-handle-unauth-refresh-unauth", 4)
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		lis := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()

		jobtokenSvc := jobtoken.NewService(runtimeScope, &fakeAuthenticator{})
		svcDesc, err := jobtokenSvc.ServiceDesc()
		if err != nil {
			t.Fatalf("jobtoken service desc: %v", err)
		}
		server.RegisterService(svcDesc, jobtokenSvc)

		methodDesc, err := taskWorkerMethod("auth")
		if err != nil {
			t.Fatalf("task worker method: %v", err)
		}
		server.RegisterService(&grpc.ServiceDesc{
			ServiceName: "auth.TaskWorker",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "ExecuteJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					reqMsg := dynamicpb.NewMessage(methodDesc.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					return nil, status.Error(codes.Unauthenticated, "still unauth")
				},
			}},
		}, nil)
		go func() { _ = server.Serve(lis) }()
		t.Cleanup(server.Stop)

		var conn *grpc.ClientConn
		dialer := client.ServiceDialer(func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
			if conn != nil {
				return conn, nil
			}
			c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return nil, err
			}
			conn = c
			return conn, nil
		})
		t.Cleanup(func() {
			if conn != nil {
				_ = conn.Close()
			}
		})

		d := NewDispatcher(runtimeScope, dialer)
		resp, handled := d.handleUnauthenticated(context.Background(), db, &job, 1)
		if resp != nil || !handled {
			t.Fatalf("handleUnauthenticated() = (%#v, %v), want (nil, true)", resp, handled)
		}
		if job.Attempt != 2 {
			t.Fatalf("job attempt = %d, want 2", job.Attempt)
		}

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "failed" {
			t.Fatalf("status = %s, want failed", got.Status)
		}
		if got.Attempt != 2 {
			t.Fatalf("stored attempt = %d, want 2", got.Attempt)
		}
		if !strings.Contains(string(got.LastErrorJson), "unauthenticated after token refresh") {
			t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
		}
	})

	t.Run("refresh then retryable error requeues job", func(t *testing.T) {
		runtimeScope, db := newDispatcherTestRuntimeScope(t, "file:task_dispatcher_handle_unauth_refresh_retry?mode=memory&cache=shared")
		job := newQueuedJob("job-handle-unauth-refresh-retry", 5)
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		lis := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()

		jobtokenSvc := jobtoken.NewService(runtimeScope, &fakeAuthenticator{})
		svcDesc, err := jobtokenSvc.ServiceDesc()
		if err != nil {
			t.Fatalf("jobtoken service desc: %v", err)
		}
		server.RegisterService(svcDesc, jobtokenSvc)

		methodDesc, err := taskWorkerMethod("auth")
		if err != nil {
			t.Fatalf("task worker method: %v", err)
		}
		server.RegisterService(&grpc.ServiceDesc{
			ServiceName: "auth.TaskWorker",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "ExecuteJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					reqMsg := dynamicpb.NewMessage(methodDesc.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					return nil, status.Error(codes.Unavailable, "worker unavailable")
				},
			}},
		}, nil)
		go func() { _ = server.Serve(lis) }()
		t.Cleanup(server.Stop)

		var conn *grpc.ClientConn
		dialer := client.ServiceDialer(func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
			if conn != nil {
				return conn, nil
			}
			c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return nil, err
			}
			conn = c
			return conn, nil
		})
		t.Cleanup(func() {
			if conn != nil {
				_ = conn.Close()
			}
		})

		d := NewDispatcher(runtimeScope, dialer)
		d.interval = time.Second
		resp, handled := d.handleUnauthenticated(context.Background(), db, &job, 1)
		if resp != nil || !handled {
			t.Fatalf("handleUnauthenticated() = (%#v, %v), want (nil, true)", resp, handled)
		}
		if job.Attempt != 2 {
			t.Fatalf("job attempt = %d, want 2", job.Attempt)
		}

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "queued" {
			t.Fatalf("status = %s, want queued", got.Status)
		}
		if got.Attempt != 2 {
			t.Fatalf("stored attempt = %d, want 2", got.Attempt)
		}
		if !strings.Contains(string(got.LastErrorJson), "worker unavailable") {
			t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
		}
	})
}

func TestDispatcherHandleJobPaths(t *testing.T) {
	newHandleJobRuntimeScope := func(t *testing.T, dsn string) (*testScope, *gorm.DB) {
		t.Helper()
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		if err := db.AutoMigrate(&Job{}); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		cfg := &config.Config{
			Auth:   &config.AuthConfig{InternalKey: "test"},
			Server: &config.ServerConfig{Environment: "development"},
			Task:   config.NewDefaultTaskConfig(),
		}
		cfg.Task.Dispatch.JobTokenTTLms = 1000
		cfg.Task.Dispatch.BackoffMaxMs = 1000
		return &testScope{
			ctx:    context.Background(),
			db:     db,
			logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			cfg:    cfg,
		}, db
	}

	newHandleJob := func(id string) Job {
		now := time.Now().UTC()
		return Job{
			Id:                id,
			TargetApp:         "auth",
			FullMethod:        "auth.User/Login",
			SchedulerUserId:   "admin",
			TriggeredByUserId: "admin",
			Status:            "queued",
			RunAfter:          now.Add(-time.Minute),
			Attempt:           0,
			MaxAttempts:       3,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
	}

	newRPCDispatcher := func(t *testing.T, runtimeScope *testScope, worker func(ctx context.Context, reqMsg *dynamicpb.Message, respMsg *dynamicpb.Message) error) (*Dispatcher, func()) {
		t.Helper()
		lis := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()

		jobtokenSvc := jobtoken.NewService(runtimeScope, &fakeAuthenticator{})
		svcDesc, err := jobtokenSvc.ServiceDesc()
		if err != nil {
			t.Fatalf("jobtoken service desc: %v", err)
		}
		server.RegisterService(svcDesc, jobtokenSvc)

		methodDesc, err := taskWorkerMethod("auth")
		if err != nil {
			t.Fatalf("task worker method: %v", err)
		}
		server.RegisterService(&grpc.ServiceDesc{
			ServiceName: "auth.TaskWorker",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "ExecuteJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					reqMsg := dynamicpb.NewMessage(methodDesc.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					respMsg := dynamicpb.NewMessage(methodDesc.Output())
					if err := worker(ctx, reqMsg, respMsg); err != nil {
						return nil, err
					}
					return respMsg, nil
				},
			}},
		}, nil)

		go func() { _ = server.Serve(lis) }()

		var conn *grpc.ClientConn
		dialer := client.ServiceDialer(func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
			if conn != nil {
				return conn, nil
			}
			c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return nil, err
			}
			conn = c
			return conn, nil
		})

		cleanup := func() {
			if conn != nil {
				_ = conn.Close()
			}
			server.Stop()
		}
		return NewDispatcher(runtimeScope, dialer), cleanup
	}

	t.Run("cancel requested marks job cancelled without calling worker", func(t *testing.T) {
		runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_cancel_requested?mode=memory&cache=shared")
		job := newHandleJob("job-handle-cancel-requested")
		now := time.Now().UTC()
		job.CancelRequestedAt = &now
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		called := false
		d := NewDispatcher(runtimeScope, client.ServiceDialer(func(context.Context, string) (*grpc.ClientConn, error) {
			called = true
			return nil, errors.New("should not dial")
		}))
		d.handleJob(job.Id)

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "cancelled" || got.CancelledAt == nil || got.FinishedAt == nil {
			t.Fatalf("unexpected cancelled job state: %#v", got)
		}
		if called {
			t.Fatal("did not expect worker dial when cancel_requested_at is set")
		}
	})

	t.Run("attempt over max fails before token issuance", func(t *testing.T) {
		runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_max_attempts?mode=memory&cache=shared")
		job := newHandleJob("job-handle-max-attempts")
		job.Attempt = 2
		job.MaxAttempts = 2
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		called := false
		d := NewDispatcher(runtimeScope, client.ServiceDialer(func(context.Context, string) (*grpc.ClientConn, error) {
			called = true
			return nil, errors.New("should not dial")
		}))
		d.handleJob(job.Id)

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "failed" || got.FinishedAt == nil {
			t.Fatalf("unexpected failed job state: %#v", got)
		}
		if !strings.Contains(string(got.LastErrorJson), "max attempts reached") {
			t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
		}
		if called {
			t.Fatal("did not expect token issuance when max attempts already reached")
		}
	})

	t.Run("token issuance failure retries job", func(t *testing.T) {
		runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_token_issue?mode=memory&cache=shared")
		job := newHandleJob("job-handle-token-issue")
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d := NewDispatcher(runtimeScope, client.ServiceDialer(func(context.Context, string) (*grpc.ClientConn, error) {
			return nil, errors.New("dial failed")
		}))
		d.interval = time.Second
		d.handleJob(job.Id)

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "queued" {
			t.Fatalf("status = %s, want queued", got.Status)
		}
		if got.Attempt != 0 {
			t.Fatalf("attempt = %d, want 0 because token issuance should not consume it", got.Attempt)
		}
		if !strings.Contains(string(got.LastErrorJson), "issue job token failed: dial failed") {
			t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
		}
	})

	t.Run("already running response requeues using default retry_after", func(t *testing.T) {
		runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_already_running?mode=memory&cache=shared")
		job := newHandleJob("job-handle-already-running")
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d, cleanup := newRPCDispatcher(t, runtimeScope, func(ctx context.Context, reqMsg *dynamicpb.Message, respMsg *dynamicpb.Message) error {
			fields := respMsg.Descriptor().Fields()
			respMsg.Set(fields.ByName("status"), protoreflect.ValueOfEnum(4))
			return nil
		})
		defer cleanup()
		d.interval = time.Second
		start := time.Now().UTC()
		d.handleJob(job.Id)

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "queued" {
			t.Fatalf("status = %s, want queued", got.Status)
		}
		if got.Attempt != 1 {
			t.Fatalf("attempt = %d, want 1 after successful token issuance", got.Attempt)
		}
		delta := got.RunAfter.Sub(start)
		if delta < 900*time.Millisecond || delta > 2*time.Second {
			t.Fatalf("run_after delta: want ~1s default retry, got %v", delta)
		}
	})

	t.Run("rpc error retries while attempts remain", func(t *testing.T) {
		runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_rpc_retry?mode=memory&cache=shared")
		job := newHandleJob("job-handle-rpc-retry")
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d, cleanup := newRPCDispatcher(t, runtimeScope, func(ctx context.Context, reqMsg *dynamicpb.Message, respMsg *dynamicpb.Message) error {
			return status.Error(codes.Internal, "worker boom")
		})
		defer cleanup()
		d.interval = time.Second
		start := time.Now().UTC()
		d.handleJob(job.Id)

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "queued" {
			t.Fatalf("status = %s, want queued", got.Status)
		}
		if got.Attempt != 1 {
			t.Fatalf("attempt = %d, want 1", got.Attempt)
		}
		if !strings.Contains(string(got.LastErrorJson), "worker boom") {
			t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
		}
		delta := got.RunAfter.Sub(start)
		if delta < 900*time.Millisecond || delta > 2*time.Second {
			t.Fatalf("run_after delta: want ~1s backoff, got %v", delta)
		}
	})

	t.Run("rpc error fails when max attempts reached", func(t *testing.T) {
		runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_rpc_fail?mode=memory&cache=shared")
		job := newHandleJob("job-handle-rpc-fail")
		job.MaxAttempts = 1
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d, cleanup := newRPCDispatcher(t, runtimeScope, func(ctx context.Context, reqMsg *dynamicpb.Message, respMsg *dynamicpb.Message) error {
			return status.Error(codes.Internal, "worker boom")
		})
		defer cleanup()
		d.handleJob(job.Id)

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "failed" || got.FinishedAt == nil {
			t.Fatalf("unexpected failed job state: %#v", got)
		}
		if !strings.Contains(string(got.LastErrorJson), "worker boom") {
			t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
		}
	})

	t.Run("failed non retryable response fails job", func(t *testing.T) {
		runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_failed_non_retryable?mode=memory&cache=shared")
		job := newHandleJob("job-handle-failed-non-retryable")
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d, cleanup := newRPCDispatcher(t, runtimeScope, func(ctx context.Context, reqMsg *dynamicpb.Message, respMsg *dynamicpb.Message) error {
			fields := respMsg.Descriptor().Fields()
			respMsg.Set(fields.ByName("status"), protoreflect.ValueOfEnum(2))
			errField := fields.ByName("error")
			errMsg := dynamicpb.NewMessage(errField.Message())
			errFields := errMsg.Descriptor().Fields()
			errMsg.Set(errFields.ByName("message"), protoreflect.ValueOfString("bad request"))
			respMsg.Set(errField, protoreflect.ValueOfMessage(errMsg))
			return nil
		})
		defer cleanup()
		d.handleJob(job.Id)

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "failed" || got.FinishedAt == nil {
			t.Fatalf("unexpected failed job state: %#v", got)
		}
		if !strings.Contains(string(got.LastErrorJson), "bad request") {
			t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
		}
	})

	t.Run("failed retryable response retries or fails at limit", func(t *testing.T) {
		makeRetryableResponse := func(ctx context.Context, reqMsg *dynamicpb.Message, respMsg *dynamicpb.Message) error {
			fields := respMsg.Descriptor().Fields()
			respMsg.Set(fields.ByName("status"), protoreflect.ValueOfEnum(3))
			errField := fields.ByName("error")
			errMsg := dynamicpb.NewMessage(errField.Message())
			errFields := errMsg.Descriptor().Fields()
			errMsg.Set(errFields.ByName("message"), protoreflect.ValueOfString("retry me"))
			respMsg.Set(errField, protoreflect.ValueOfMessage(errMsg))
			return nil
		}

		t.Run("requeues when attempts remain", func(t *testing.T) {
			runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_failed_retryable_requeue?mode=memory&cache=shared")
			job := newHandleJob("job-handle-failed-retryable-requeue")
			if err := db.Create(&job).Error; err != nil {
				t.Fatalf("create job: %v", err)
			}

			d, cleanup := newRPCDispatcher(t, runtimeScope, makeRetryableResponse)
			defer cleanup()
			d.interval = time.Second
			d.handleJob(job.Id)

			var got Job
			if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
				t.Fatalf("load job: %v", err)
			}
			if got.Status != "queued" {
				t.Fatalf("status = %s, want queued", got.Status)
			}
			if !strings.Contains(string(got.LastErrorJson), "retry me") {
				t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
			}
		})

		t.Run("fails when max attempts reached", func(t *testing.T) {
			runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_failed_retryable_fail?mode=memory&cache=shared")
			job := newHandleJob("job-handle-failed-retryable-fail")
			job.MaxAttempts = 1
			if err := db.Create(&job).Error; err != nil {
				t.Fatalf("create job: %v", err)
			}

			d, cleanup := newRPCDispatcher(t, runtimeScope, makeRetryableResponse)
			defer cleanup()
			d.handleJob(job.Id)

			var got Job
			if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
				t.Fatalf("load job: %v", err)
			}
			if got.Status != "failed" || got.FinishedAt == nil {
				t.Fatalf("unexpected failed job state: %#v", got)
			}
			if !strings.Contains(string(got.LastErrorJson), "retry me") {
				t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
			}
		})
	})

	t.Run("resource busy response uses explicit retry_after", func(t *testing.T) {
		runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_resource_busy?mode=memory&cache=shared")
		job := newHandleJob("job-handle-resource-busy")
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d, cleanup := newRPCDispatcher(t, runtimeScope, func(ctx context.Context, reqMsg *dynamicpb.Message, respMsg *dynamicpb.Message) error {
			fields := respMsg.Descriptor().Fields()
			respMsg.Set(fields.ByName("status"), protoreflect.ValueOfEnum(5))
			respMsg.Set(fields.ByName("retry_after_ms"), protoreflect.ValueOfInt64(2500))
			return nil
		})
		defer cleanup()
		start := time.Now().UTC()
		d.handleJob(job.Id)

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "queued" {
			t.Fatalf("status = %s, want queued", got.Status)
		}
		delta := got.RunAfter.Sub(start)
		if delta < 2300*time.Millisecond || delta > 4*time.Second {
			t.Fatalf("run_after delta: want ~2.5s retry_after, got %v", delta)
		}
	})

	t.Run("cancelled response marks job cancelled", func(t *testing.T) {
		runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_cancelled?mode=memory&cache=shared")
		job := newHandleJob("job-handle-cancelled")
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d, cleanup := newRPCDispatcher(t, runtimeScope, func(ctx context.Context, reqMsg *dynamicpb.Message, respMsg *dynamicpb.Message) error {
			fields := respMsg.Descriptor().Fields()
			respMsg.Set(fields.ByName("status"), protoreflect.ValueOfEnum(6))
			return nil
		})
		defer cleanup()
		d.handleJob(job.Id)

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "cancelled" || got.CancelledAt == nil || got.FinishedAt == nil {
			t.Fatalf("unexpected cancelled job state: %#v", got)
		}
	})

	t.Run("unknown worker status retries with unknown status error", func(t *testing.T) {
		runtimeScope, db := newHandleJobRuntimeScope(t, "file:task_dispatcher_handle_job_unknown_status?mode=memory&cache=shared")
		job := newHandleJob("job-handle-unknown-status")
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d, cleanup := newRPCDispatcher(t, runtimeScope, func(ctx context.Context, reqMsg *dynamicpb.Message, respMsg *dynamicpb.Message) error {
			return nil
		})
		defer cleanup()
		d.interval = time.Second
		d.handleJob(job.Id)

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "queued" {
			t.Fatalf("status = %s, want queued", got.Status)
		}
		if !strings.Contains(string(got.LastErrorJson), "unknown status") {
			t.Fatalf("unexpected last_error_json: %s", string(got.LastErrorJson))
		}
	})
}

func TestDispatcherPollOncePaths(t *testing.T) {
	newPollRuntimeScope := func(t *testing.T, dsn string) (*testScope, *gorm.DB) {
		t.Helper()
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		if err := db.AutoMigrate(&Job{}); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		cfg := &config.Config{Db: &config.DbConfig{Dialect: "sqlite"}, Task: config.NewDefaultTaskConfig()}
		cfg.Task.Dispatch.MaxConcurrency = 1
		cfg.Task.Dispatch.MaxConcurrencyPerApp = 1
		return &testScope{
			ctx:    context.Background(),
			db:     db,
			logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			cfg:    cfg,
		}, db
	}

	newQueuedPollJob := func(id string) Job {
		now := time.Now().UTC()
		return Job{
			Id:                id,
			TargetApp:         "auth",
			FullMethod:        "auth.User/Login",
			SchedulerUserId:   "admin",
			TriggeredByUserId: "admin",
			Status:            "queued",
			RunAfter:          now.Add(-time.Minute),
			CreatedAt:         now,
			UpdatedAt:         now,
		}
	}

	t.Run("poll enqueues jobs when no worker slots are available", func(t *testing.T) {
		runtimeScope, db := newPollRuntimeScope(t, "file:task_dispatcher_poll_once_no_slots?mode=memory&cache=shared")
		job := newQueuedPollJob("job-poll-no-slots")
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d := NewDispatcher(runtimeScope, nil)
		d.sema <- struct{}{}
		d.pollOnce("poll")

		if len(d.readyQ) != 1 || d.readyQ[0].Id != job.Id {
			t.Fatalf("expected ready queue to retain queued job, got %#v", d.readyQ)
		}
		if _, ok := d.readySet[job.Id]; !ok {
			t.Fatalf("expected readySet to contain %s", job.Id)
		}

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "queued" {
			t.Fatalf("status = %s, want queued", got.Status)
		}
	})

	t.Run("poll keeps ready job queued when app concurrency is saturated", func(t *testing.T) {
		runtimeScope, db := newPollRuntimeScope(t, "file:task_dispatcher_poll_once_app_cap?mode=memory&cache=shared")
		job := newQueuedPollJob("job-poll-app-cap")
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d := NewDispatcher(runtimeScope, nil)
		d.appInFly["auth"] = 1
		d.pollOnce("wake")

		if len(d.readyQ) != 1 || d.readyQ[0].Id != job.Id {
			t.Fatalf("expected ready queue to keep blocked app job, got %#v", d.readyQ)
		}
		if len(d.sema) != 0 {
			t.Fatalf("expected no worker slot to be consumed, got %d", len(d.sema))
		}

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "queued" {
			t.Fatalf("status = %s, want queued", got.Status)
		}
	})

	t.Run("tryClaim failure releases app slot and drops popped ready item", func(t *testing.T) {
		runtimeScope, db := newPollRuntimeScope(t, "file:task_dispatcher_poll_once_tryclaim_fail?mode=memory&cache=shared")
		now := time.Now().UTC()
		job := Job{
			Id:                "job-poll-tryclaim-fail",
			TargetApp:         "auth",
			FullMethod:        "auth.User/Login",
			SchedulerUserId:   "admin",
			TriggeredByUserId: "admin",
			Status:            "dispatching",
			RunAfter:          now.Add(-time.Minute),
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		d := NewDispatcher(runtimeScope, nil)
		d.readyQ = append(d.readyQ, job)
		d.readySet[job.Id] = struct{}{}
		d.pollOnce("wake")

		if len(d.sema) != 0 {
			t.Fatalf("expected no worker slot to be consumed, got %d", len(d.sema))
		}
		if len(d.readyQ) != 0 {
			t.Fatalf("expected popped ready job to be removed after failed claim, got %#v", d.readyQ)
		}
		if _, ok := d.readySet[job.Id]; ok {
			t.Fatalf("expected readySet to drop %s after pop", job.Id)
		}
		if got := d.appInFly["auth"]; got != 0 {
			t.Fatalf("expected app slot release after failed claim, got %d", got)
		}

		var got Job
		if err := db.Where("id = ?", job.Id).First(&got).Error; err != nil {
			t.Fatalf("load job: %v", err)
		}
		if got.Status != "dispatching" {
			t.Fatalf("status = %s, want dispatching", got.Status)
		}
	})
}

func TestDispatcherCallExecuteJobPaths(t *testing.T) {
	type seenHeaders struct {
		authorization []string
		internal      []string
	}

	t.Run("successful response maps status retry_after and metadata", func(t *testing.T) {
		methodDesc, err := taskWorkerMethod("auth")
		if err != nil {
			t.Fatalf("taskWorkerMethod() error = %v", err)
		}

		lis := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()
		seenHeadersCh := make(chan seenHeaders, 1)
		serviceDesc := &grpc.ServiceDesc{
			ServiceName: "auth.TaskWorker",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "ExecuteJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					if md, ok := metadata.FromIncomingContext(ctx); ok {
						select {
						case seenHeadersCh <- seenHeaders{authorization: append([]string(nil), md.Get("authorization")...), internal: append([]string(nil), md.Get(internalKeyHeader)...)}:
						default:
						}
					}
					reqMsg := dynamicpb.NewMessage(methodDesc.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					respMsg := dynamicpb.NewMessage(methodDesc.Output())
					fields := respMsg.Descriptor().Fields()
					respMsg.Set(fields.ByName("status"), protoreflect.ValueOfEnum(3))
					respMsg.Set(fields.ByName("retry_after_ms"), protoreflect.ValueOfInt64(321))
					return respMsg, nil
				},
			}},
		}
		server.RegisterService(serviceDesc, nil)
		go func() { _ = server.Serve(lis) }()
		t.Cleanup(server.Stop)

		var conn *grpc.ClientConn
		dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
			if serviceName != "auth.TaskWorker" {
				t.Fatalf("serviceName = %q, want auth.TaskWorker", serviceName)
			}
			if conn != nil {
				return conn, nil
			}
			c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

		cfg := &config.Config{
			Auth:   &config.AuthConfig{InternalKey: "secret"},
			Server: &config.ServerConfig{Environment: "development"},
			Task:   config.NewDefaultTaskConfig(),
		}
		d := NewDispatcher(&testScope{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)), cfg: cfg}, client.ServiceDialer(dialer))
		job := &Job{
			Id:                "job-call-success",
			TargetApp:         "auth",
			FullMethod:        "auth.User/Login",
			PayloadJson:       []byte(`{"ok":true}`),
			SchedulerUserId:   "admin",
			TriggeredByUserId: "admin",
			Attempt:           2,
			TimeoutMs:         1000,
		}

		resp, err := d.callExecuteJob(context.Background(), job, "token-1")
		if err != nil {
			t.Fatalf("callExecuteJob() error = %v", err)
		}
		if resp.Status != "FAILED_RETRYABLE" {
			t.Fatalf("unexpected response mapping: %#v", resp)
		}
		seen := <-seenHeadersCh
		if got := seen.authorization; len(got) != 1 || got[0] != "Bearer token-1" {
			t.Fatalf("authorization metadata = %#v", got)
		}
		if got := seen.internal; len(got) != 1 || got[0] != "secret" {
			t.Fatalf("internal key metadata = %#v", got)
		}
	})

	t.Run("production omits internal header and default timeout enforces deadline", func(t *testing.T) {
		methodDesc, err := taskWorkerMethod("auth")
		if err != nil {
			t.Fatalf("taskWorkerMethod() error = %v", err)
		}

		lis := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()
		seenHeadersCh := make(chan seenHeaders, 1)
		serviceDesc := &grpc.ServiceDesc{
			ServiceName: "auth.TaskWorker",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "ExecuteJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					if md, ok := metadata.FromIncomingContext(ctx); ok {
						select {
						case seenHeadersCh <- seenHeaders{authorization: append([]string(nil), md.Get("authorization")...), internal: append([]string(nil), md.Get(internalKeyHeader)...)}:
						default:
						}
					}
					reqMsg := dynamicpb.NewMessage(methodDesc.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					<-ctx.Done()
					return nil, ctx.Err()
				},
			}},
		}
		server.RegisterService(serviceDesc, nil)
		go func() { _ = server.Serve(lis) }()
		t.Cleanup(server.Stop)

		var conn *grpc.ClientConn
		dialer := func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
			if serviceName != "auth.TaskWorker" {
				t.Fatalf("serviceName = %q, want auth.TaskWorker", serviceName)
			}
			if conn != nil {
				return conn, nil
			}
			c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

		cfg := &config.Config{
			Auth:   &config.AuthConfig{InternalKey: "secret"},
			Server: &config.ServerConfig{Environment: "production"},
			Task:   config.NewDefaultTaskConfig(),
		}
		cfg.Task.Dispatch.DefaultJobTimeoutMs = 20
		d := NewDispatcher(&testScope{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)), cfg: cfg}, client.ServiceDialer(dialer))
		job := &Job{
			Id:                "job-call-timeout",
			TargetApp:         "auth",
			FullMethod:        "auth.User/Login",
			PayloadJson:       []byte(`not-json`),
			SchedulerUserId:   "admin",
			TriggeredByUserId: "admin",
			Attempt:           1,
			TimeoutMs:         0,
		}

		if _, err := d.callExecuteJob(context.Background(), job, "token-2"); status.Code(err) != codes.DeadlineExceeded {
			t.Fatalf("expected deadline exceeded, got %v", err)
		}
		seen := <-seenHeadersCh
		if got := seen.authorization; len(got) != 1 || got[0] != "Bearer token-2" {
			t.Fatalf("authorization metadata = %#v", got)
		}
		if got := seen.internal; len(got) != 0 {
			t.Fatalf("did not expect internal key metadata in production, got %#v", got)
		}
	})

	t.Run("dialer and unauthenticated errors are surfaced", func(t *testing.T) {
		cfg := &config.Config{Task: config.NewDefaultTaskConfig(), Auth: &config.AuthConfig{}, Server: &config.ServerConfig{Environment: "development"}}
		job := &Job{Id: "job-call-errors", TargetApp: "auth", FullMethod: "auth.User/Login"}

		d := NewDispatcher(&testScope{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)), cfg: cfg}, client.ServiceDialer(func(context.Context, string) (*grpc.ClientConn, error) {
			return nil, errors.New("dial failed")
		}))
		if _, err := d.callExecuteJob(context.Background(), job, "token"); err == nil || err.Error() != "dial failed" {
			t.Fatalf("expected dial failure, got %v", err)
		}

		methodDesc, err := taskWorkerMethod("auth")
		if err != nil {
			t.Fatalf("taskWorkerMethod() error = %v", err)
		}
		lis := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()
		serviceDesc := &grpc.ServiceDesc{
			ServiceName: "auth.TaskWorker",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "ExecuteJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					reqMsg := dynamicpb.NewMessage(methodDesc.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					return nil, status.Error(codes.Unauthenticated, "unauth")
				},
			}},
		}
		server.RegisterService(serviceDesc, nil)
		go func() { _ = server.Serve(lis) }()
		t.Cleanup(server.Stop)

		var conn *grpc.ClientConn
		unauthDispatcher := NewDispatcher(&testScope{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)), cfg: cfg}, client.ServiceDialer(func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
			if serviceName != "auth.TaskWorker" {
				t.Fatalf("serviceName = %q, want auth.TaskWorker", serviceName)
			}
			if conn != nil {
				return conn, nil
			}
			c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return nil, err
			}
			conn = c
			return conn, nil
		}))
		t.Cleanup(func() {
			if conn != nil {
				_ = conn.Close()
			}
		})

		if _, err := unauthDispatcher.callExecuteJob(context.Background(), job, "token"); !isUnauthenticated(err) {
			t.Fatalf("expected unauthenticated wrapper, got %v", err)
		}
	})

	t.Run("unavailable errors are returned directly", func(t *testing.T) {
		cfg := &config.Config{Task: config.NewDefaultTaskConfig(), Auth: &config.AuthConfig{}, Server: &config.ServerConfig{Environment: "development"}}
		job := &Job{Id: "job-call-unavailable", TargetApp: "auth", FullMethod: "auth.User/Login"}

		methodDesc, err := taskWorkerMethod("auth")
		if err != nil {
			t.Fatalf("taskWorkerMethod() error = %v", err)
		}
		lis := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()
		serviceDesc := &grpc.ServiceDesc{
			ServiceName: "auth.TaskWorker",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "ExecuteJob",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					reqMsg := dynamicpb.NewMessage(methodDesc.Input())
					if err := dec(reqMsg); err != nil {
						return nil, err
					}
					return nil, status.Error(codes.Unavailable, "worker unavailable")
				},
			}},
		}
		server.RegisterService(serviceDesc, nil)
		go func() { _ = server.Serve(lis) }()
		t.Cleanup(server.Stop)

		var conn *grpc.ClientConn
		d := NewDispatcher(&testScope{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)), cfg: cfg}, client.ServiceDialer(func(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
			if serviceName != "auth.TaskWorker" {
				t.Fatalf("serviceName = %q, want auth.TaskWorker", serviceName)
			}
			if conn != nil {
				return conn, nil
			}
			c, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return lis.Dial()
			}), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return nil, err
			}
			conn = c
			return conn, nil
		}))
		t.Cleanup(func() {
			if conn != nil {
				_ = conn.Close()
			}
		})

		if _, err := d.callExecuteJob(context.Background(), job, "token"); status.Code(err) != codes.Unavailable {
			t.Fatalf("expected unavailable error, got %v", err)
		}
	})
}
