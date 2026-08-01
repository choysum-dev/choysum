// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type gcTestScope struct {
	ctx    context.Context
	db     *gorm.DB
	logger *slog.Logger
	cfg    *config.Config
	props  *[]scope.Propagation
}

type gcTestTransaction struct {
	ctx     context.Context
	session *scope.Session
}

type gcTestTransactor struct {
	runtimeScope *gcTestScope
}

func (e *gcTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *gcTestScope) Session() *scope.Session                           { return &scope.Session{DB: e.db} }
func (e *gcTestScope) Transactor() scope.Transactor                      { return &gcTestTransactor{runtimeScope: e} }
func (e *gcTestScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *gcTestScope) Context() context.Context { return e.ctx }
func (e *gcTestScope) Logger() *slog.Logger     { return e.logger }
func (e *gcTestScope) Config() *config.Config   { return e.cfg }
func (e *gcTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

func (u *gcTestTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	if opts.Propagation == "" {
		opts.Propagation = scope.PropagationRequired
	}
	if u.runtimeScope.props != nil {
		*u.runtimeScope.props = append(*u.runtimeScope.props, opts.Propagation)
	}
	txCtx := ctx
	if txCtx == nil {
		txCtx = u.runtimeScope.Context()
	}
	txScope, _ := u.runtimeScope.WithContext(txCtx).(*gcTestScope)
	tx := &gcTestTransaction{session: txScope.Session()}
	tx.ctx = scope.ContextWithTransaction(txCtx, tx)
	txScope.ctx = tx.ctx
	return fn(txScope, tx)
}

func (u *gcTestTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequired}, fn)
}

func (u *gcTestTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequiresNew}, fn)
}

func (u *gcTestTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return u.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationNested}, fn)
}

func (tx *gcTestTransaction) Context() context.Context {
	if tx == nil {
		return nil
	}
	return tx.ctx
}

func (tx *gcTestTransaction) Session() *scope.Session {
	if tx == nil {
		return nil
	}
	return tx.session
}

func (tx *gcTestTransaction) Savepoint(string) error           { return nil }
func (tx *gcTestTransaction) RollbackToSavepoint(string) error { return nil }
func (tx *gcTestTransaction) ReleaseSavepoint(string) error    { return nil }

func TestGarbageCollectorRetention(t *testing.T) {
	ctx := context.Background()

	dsn := "file:task_gc_retention?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}, &Execution{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC()
	old := now.Add(-48 * time.Hour)
	recent := now.Add(-2 * time.Hour)

	jobs := []Job{
		{Id: "j1", Status: "succeeded", FinishedAt: &old},
		{Id: "j2", Status: "succeeded", FinishedAt: &recent},
		{Id: "j3", Status: "failed", FinishedAt: &old},
		{Id: "j4", Status: "cancelled", FinishedAt: &old},
		{Id: "j5", Status: "failed"},
	}
	for i := range jobs {
		jobs[i].RunAfter = now
		jobs[i].TargetApp = "auth"
		jobs[i].FullMethod = "auth.User/Login"
		jobs[i].SchedulerUserId = "admin"
		jobs[i].TriggeredByUserId = "admin"
		jobs[i].CreatedAt = now
		jobs[i].UpdatedAt = now
		if err := db.Create(&jobs[i]).Error; err != nil {
			t.Fatalf("create job %s: %v", jobs[i].Id, err)
		}
	}

	execs := []Execution{
		{JobId: "e1", Status: "succeeded", FinishedAt: &old},
		{JobId: "e2", Status: "succeeded", FinishedAt: &recent},
		{JobId: "e3", Status: "failed", FinishedAt: &old},
		{JobId: "e4", Status: "cancelled", FinishedAt: &old},
	}
	for i := range execs {
		execs[i].CreatedAt = now
		execs[i].UpdatedAt = now
		execs[i].FullMethod = "auth.User/Login"
		if err := db.Create(&execs[i]).Error; err != nil {
			t.Fatalf("create exec %s: %v", execs[i].JobId, err)
		}
	}

	cfg := &config.Config{
		Task: &config.TaskConfig{
			Retention: &config.TaskRetentionConfig{
				GCIntervalMs: 1000,
				TaskJob: &config.TaskRetentionEntry{
					TaskRetentionPolicy: config.TaskRetentionPolicy{SucceededDays: 1, FailedDays: 1, CancelledDays: 1},
					Overrides:           map[string]*config.TaskRetentionPolicy{},
				},
				TaskExecution: &config.TaskRetentionEntry{
					TaskRetentionPolicy: config.TaskRetentionPolicy{SucceededDays: 1, FailedDays: 1, CancelledDays: 1},
					Overrides:           map[string]*config.TaskRetentionPolicy{},
				},
			},
		},
	}

	runtimeScope := &gcTestScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	gc := NewGarbageCollector(runtimeScope)
	gc.collectOnce()

	var jobCount int64
	if err := db.Model(&Job{}).Count(&jobCount).Error; err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if jobCount != 2 {
		t.Fatalf("job count: want 2, got %d", jobCount)
	}

	var execCount int64
	if err := db.Model(&Execution{}).Count(&execCount).Error; err != nil {
		t.Fatalf("count execs: %v", err)
	}
	if execCount != 1 {
		t.Fatalf("exec count: want 1, got %d", execCount)
	}
}

func TestGarbageCollectorRetentionOverrides(t *testing.T) {
	ctx := context.Background()

	dsn := "file:task_gc_overrides?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}, &Execution{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC()
	old := now.Add(-72 * time.Hour)

	jobs := []Job{
		{Id: "job-default", Status: "succeeded", FullMethod: "auth.User/Login", FinishedAt: &old},
		{Id: "job-short", Status: "succeeded", FullMethod: "meta.MetaModule/Install", FinishedAt: &old},
	}
	for i := range jobs {
		jobs[i].RunAfter = now
		jobs[i].TargetApp = "auth"
		jobs[i].SchedulerUserId = "admin"
		jobs[i].TriggeredByUserId = "admin"
		jobs[i].CreatedAt = now
		jobs[i].UpdatedAt = now
		if err := db.Create(&jobs[i]).Error; err != nil {
			t.Fatalf("create job %s: %v", jobs[i].Id, err)
		}
	}

	execs := []Execution{
		{JobId: "exec-default", Status: "succeeded", FullMethod: "auth.User/Login", FinishedAt: &old},
		{JobId: "exec-short", Status: "succeeded", FullMethod: "meta.MetaModule/Install", FinishedAt: &old},
	}
	for i := range execs {
		execs[i].CreatedAt = now
		execs[i].UpdatedAt = now
		if err := db.Create(&execs[i]).Error; err != nil {
			t.Fatalf("create exec %s: %v", execs[i].JobId, err)
		}
	}

	cfg := &config.Config{
		Task: &config.TaskConfig{
			Retention: &config.TaskRetentionConfig{
				GCIntervalMs: 1000,
				TaskJob: &config.TaskRetentionEntry{
					TaskRetentionPolicy: config.TaskRetentionPolicy{SucceededDays: 10, FailedDays: 10, CancelledDays: 10},
					Overrides: map[string]*config.TaskRetentionPolicy{
						"meta.MetaModule/Install": {SucceededDays: 1, FailedDays: 1, CancelledDays: 1},
					},
				},
				TaskExecution: &config.TaskRetentionEntry{
					TaskRetentionPolicy: config.TaskRetentionPolicy{SucceededDays: 10, FailedDays: 10, CancelledDays: 10},
					Overrides: map[string]*config.TaskRetentionPolicy{
						"meta.MetaModule/Install": {SucceededDays: 1, FailedDays: 1, CancelledDays: 1},
					},
				},
			},
		},
	}

	runtimeScope := &gcTestScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:    cfg,
	}

	gc := NewGarbageCollector(runtimeScope)
	gc.collectOnce()

	var jobCount int64
	if err := db.Model(&Job{}).Count(&jobCount).Error; err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if jobCount != 1 {
		t.Fatalf("job count: want 1, got %d", jobCount)
	}

	var execCount int64
	if err := db.Model(&Execution{}).Count(&execCount).Error; err != nil {
		t.Fatalf("count execs: %v", err)
	}
	if execCount != 1 {
		t.Fatalf("exec count: want 1, got %d", execCount)
	}
}

func TestGarbageCollectorCollectOnceUsesRequiredTransactionBoundary(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:task_gc_required_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Job{}, &Execution{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC()
	old := now.Add(-72 * time.Hour)
	job := Job{
		Id:                "job-required",
		TargetApp:         "auth",
		FullMethod:        "auth.User/Login",
		SchedulerUserId:   "admin",
		TriggeredByUserId: "admin",
		Status:            "succeeded",
		RunAfter:          now,
		FinishedAt:        &old,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	propagations := make([]scope.Propagation, 0, 2)
	runtimeScope := &gcTestScope{
		ctx:    ctx,
		db:     db,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg: &config.Config{Task: &config.TaskConfig{Retention: &config.TaskRetentionConfig{
			GCIntervalMs: 1000,
			TaskJob: &config.TaskRetentionEntry{
				TaskRetentionPolicy: config.TaskRetentionPolicy{SucceededDays: 1, FailedDays: 1, CancelledDays: 1},
				Overrides:           map[string]*config.TaskRetentionPolicy{},
			},
			TaskExecution: &config.TaskRetentionEntry{
				TaskRetentionPolicy: config.TaskRetentionPolicy{SucceededDays: 1, FailedDays: 1, CancelledDays: 1},
				Overrides:           map[string]*config.TaskRetentionPolicy{},
			},
		}}},
		props: &propagations,
	}

	gc := NewGarbageCollector(runtimeScope)
	gc.collectOnce()

	if len(propagations) != 1 || propagations[0] != scope.PropagationRequired {
		t.Fatalf("collectOnce propagation = %#v, want [required]", propagations)
	}

	var count int64
	if err := db.Model(&Job{}).Count(&count).Error; err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if count != 0 {
		t.Fatalf("job count: want 0, got %d", count)
	}
}

func TestGarbageCollectorLifecycle(t *testing.T) {
	t.Run("nil and zero interval start stop are safe", func(t *testing.T) {
		var nilGC *GarbageCollector
		nilGC.Start()
		nilGC.Stop()

		gc := &GarbageCollector{stopCh: make(chan struct{}), interval: 0}
		gc.Start()
		gc.Stop()
	})

	t.Run("start loop collects and stop waits for exit", func(t *testing.T) {
		dsn := fmt.Sprintf("file:task_gc_lifecycle_%d?mode=memory&cache=shared", time.Now().UnixNano())
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		if err := db.AutoMigrate(&Job{}, &Execution{}); err != nil {
			t.Fatalf("migrate: %v", err)
		}

		now := time.Now().UTC()
		old := now.Add(-72 * time.Hour)
		job := Job{
			Id:                "job-loop",
			TargetApp:         "auth",
			FullMethod:        "auth.User/Login",
			SchedulerUserId:   "admin",
			TriggeredByUserId: "admin",
			Status:            "succeeded",
			RunAfter:          now,
			FinishedAt:        &old,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job: %v", err)
		}

		runtimeScope := &gcTestScope{
			ctx:    context.Background(),
			db:     db,
			logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			cfg: &config.Config{Task: &config.TaskConfig{Retention: &config.TaskRetentionConfig{
				GCIntervalMs: 5,
				TaskJob: &config.TaskRetentionEntry{
					TaskRetentionPolicy: config.TaskRetentionPolicy{SucceededDays: 1, FailedDays: 1, CancelledDays: 1},
					Overrides:           map[string]*config.TaskRetentionPolicy{},
				},
				TaskExecution: &config.TaskRetentionEntry{
					TaskRetentionPolicy: config.TaskRetentionPolicy{SucceededDays: 1, FailedDays: 1, CancelledDays: 1},
					Overrides:           map[string]*config.TaskRetentionPolicy{},
				},
			}}},
		}

		gc := NewGarbageCollector(runtimeScope)
		gc.Start()
		stopped := false
		defer func() {
			if !stopped {
				gc.Stop()
			}
		}()

		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			var count int64
			if err := db.Model(&Job{}).Count(&count).Error; err != nil {
				t.Fatalf("count jobs: %v", err)
			}
			if count == 0 {
				gc.Stop()
				stopped = true
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("expected garbage collector loop to delete expired job")
	})
}

func TestGarbageCollectorHelperDefaults(t *testing.T) {
	t.Run("retention config falls back to defaults", func(t *testing.T) {
		gc := &GarbageCollector{}
		retention := gc.retentionConfig()
		if retention == nil || retention.TaskJob == nil || retention.TaskExecution == nil {
			t.Fatalf("expected default retention config, got %#v", retention)
		}
		if retention.TaskJob.Overrides == nil || retention.TaskExecution.Overrides == nil {
			t.Fatalf("expected default override maps, got %#v", retention)
		}
	})

	t.Run("retention config fills missing nested defaults", func(t *testing.T) {
		cfg := &config.Config{
			Task: &config.TaskConfig{
				Retention: &config.TaskRetentionConfig{
					TaskJob:       &config.TaskRetentionEntry{},
					TaskExecution: &config.TaskRetentionEntry{},
				},
			},
		}
		runtimeScope := &gcTestScope{ctx: context.Background(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)), cfg: cfg}
		gc := NewGarbageCollector(runtimeScope)
		retention := gc.retentionConfig()
		if retention.TaskJob == nil || retention.TaskExecution == nil {
			t.Fatalf("expected nested retention entries, got %#v", retention)
		}
		if retention.TaskJob.Overrides == nil || retention.TaskExecution.Overrides == nil {
			t.Fatalf("expected override maps to be filled, got %#v", retention)
		}
	})

	t.Run("daysAgo handles disabled and positive cutoffs", func(t *testing.T) {
		now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
		if daysAgo(now, 0) != nil {
			t.Fatal("expected daysAgo with non-positive days to return nil")
		}
		cutoff := daysAgo(now, 2)
		if cutoff == nil || !cutoff.Equal(now.Add(-48*time.Hour)) {
			t.Fatalf("unexpected cutoff: %#v", cutoff)
		}
	})
}
