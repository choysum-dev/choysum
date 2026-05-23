// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"sync"
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
	"gorm.io/gorm"
)

var _ taskcontract.GarbageCollector = (*GarbageCollector)(nil)

type GarbageCollector struct {
	runtimeScope scope.Scope
	runtimeOpts  runtimeOptions
	stopCh       chan struct{}
	wg           sync.WaitGroup
	interval     time.Duration
	batch        int
}

func NewGarbageCollector(runtimeScope scope.Scope) *GarbageCollector {
	opts := runtimeOptionsFromScope(runtimeScope)
	return &GarbageCollector{
		runtimeScope: runtimeScope,
		runtimeOpts:  opts,
		stopCh:       make(chan struct{}),
		interval:     opts.retentionGCInterval,
		batch:        1000,
	}
}

func (g *GarbageCollector) resolvedRuntimeOptions() runtimeOptions {
	if g != nil && g.runtimeOpts.initialized {
		return g.runtimeOpts
	}
	if g != nil {
		return runtimeOptionsFromScope(g.runtimeScope)
	}
	return newRuntimeOptions(scope.DatabaseRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false)
}

func (g *GarbageCollector) Start() {
	if g == nil || g.interval <= 0 {
		return
	}
	g.wg.Add(1)
	go g.loop()
}

func (g *GarbageCollector) Stop() {
	if g == nil {
		return
	}
	close(g.stopCh)
	g.wg.Wait()
}

func (g *GarbageCollector) loop() {
	defer g.wg.Done()
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()
	for {
		select {
		case <-g.stopCh:
			return
		case <-ticker.C:
			g.collectOnce()
		}
	}
}

func (g *GarbageCollector) collectOnce() {
	if g.runtimeScope == nil {
		return
	}
	cfg := g.retentionConfig()
	if cfg == nil {
		return
	}
	txRoot := g.runtimeScope.WithContext(g.runtimeScope.Context())
	_ = txRoot.Transactor().Required(txRoot.Context(), func(txScope scope.Scope, _ scope.Transaction) error {
		if txScope.Session() == nil || txScope.Session().DB == nil {
			return nil
		}
		now := time.Now().UTC()
		db := txScope.Session().DB
		g.purgeTaskJob(db, now, cfg.TaskJob)
		g.purgeTaskExecution(db, now, cfg.TaskExecution)
		return nil
	})
}

func (g *GarbageCollector) retentionConfig() *config.TaskRetentionConfig {
	opts := g.resolvedRuntimeOptions()
	if opts.retention != nil {
		return cloneTaskRetentionConfig(opts.retention, nil)
	}
	return cloneTaskRetentionConfig(nil, nil)
}

func (g *GarbageCollector) purgeTaskJob(db *gorm.DB, now time.Time, entry *config.TaskRetentionEntry) {
	if entry == nil {
		return
	}
	g.purgeOverrides(db, "task_job", "id", "full_method", entry.Overrides)
	g.purgeByStatus(db, "task_job", "id", "succeeded", daysAgo(now, entry.SucceededDays), g.batch)
	g.purgeByStatus(db, "task_job", "id", "failed", daysAgo(now, entry.FailedDays), g.batch)
	g.purgeByStatus(db, "task_job", "id", "cancelled", daysAgo(now, entry.CancelledDays), g.batch)
}

func (g *GarbageCollector) purgeTaskExecution(db *gorm.DB, now time.Time, entry *config.TaskRetentionEntry) {
	if entry == nil {
		return
	}
	g.purgeOverrides(db, "task_job_execution", "job_id", "full_method", entry.Overrides)
	g.purgeByStatus(db, "task_job_execution", "job_id", "succeeded", daysAgo(now, entry.SucceededDays), g.batch)
	g.purgeByStatus(db, "task_job_execution", "job_id", "failed", daysAgo(now, entry.FailedDays), g.batch)
	g.purgeByStatus(db, "task_job_execution", "job_id", "cancelled", daysAgo(now, entry.CancelledDays), g.batch)
}

func daysAgo(now time.Time, days int) *time.Time {
	if days <= 0 {
		return nil
	}
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	return &cutoff
}

func (g *GarbageCollector) purgeByStatus(db *gorm.DB, table string, idColumn string, status string, cutoff *time.Time, batch int) {
	if db == nil || cutoff == nil || batch <= 0 {
		return
	}
	for {
		var ids []string
		err := db.Table(table).
			Select(idColumn).
			Where("status = ? AND finished_at IS NOT NULL AND finished_at < ?", status, *cutoff).
			Limit(batch).
			Pluck(idColumn, &ids).Error
		if err != nil || len(ids) == 0 {
			return
		}
		res := db.Table(table).Where(idColumn+" IN ?", ids).Delete(nil)
		if res.Error != nil {
			return
		}
		if res.RowsAffected > 0 {
			g.emitMetric("task_gc_deleted", map[string]any{
				"table":  table,
				"status": status,
				"count":  res.RowsAffected,
			})
		}
	}
}

func (g *GarbageCollector) purgeOverrides(db *gorm.DB, table string, idColumn string, methodColumn string, overrides map[string]*config.TaskRetentionPolicy) {
	if db == nil || len(overrides) == 0 {
		return
	}
	now := time.Now().UTC()
	for method, policy := range overrides {
		if method == "" || policy == nil {
			continue
		}
		g.purgeByStatusWithMethod(db, table, idColumn, methodColumn, method, "succeeded", daysAgo(now, policy.SucceededDays), g.batch)
		g.purgeByStatusWithMethod(db, table, idColumn, methodColumn, method, "failed", daysAgo(now, policy.FailedDays), g.batch)
		g.purgeByStatusWithMethod(db, table, idColumn, methodColumn, method, "cancelled", daysAgo(now, policy.CancelledDays), g.batch)
	}
}

func (g *GarbageCollector) purgeByStatusWithMethod(db *gorm.DB, table string, idColumn string, methodColumn string, method string, status string, cutoff *time.Time, batch int) {
	if db == nil || cutoff == nil || batch <= 0 {
		return
	}
	for {
		var ids []string
		err := db.Table(table).
			Select(idColumn).
			Where(methodColumn+" = ? AND status = ? AND finished_at IS NOT NULL AND finished_at < ?", method, status, *cutoff).
			Limit(batch).
			Pluck(idColumn, &ids).Error
		if err != nil || len(ids) == 0 {
			return
		}
		res := db.Table(table).Where(idColumn+" IN ?", ids).Delete(nil)
		if res.Error != nil {
			return
		}
		if res.RowsAffected > 0 {
			g.emitMetric("task_gc_deleted", map[string]any{
				"table":       table,
				"status":      status,
				"full_method": method,
				"count":       res.RowsAffected,
			})
		}
	}
}

func (g *GarbageCollector) emitMetric(name string, fields map[string]any) {
	if g.runtimeScope == nil {
		return
	}
	fields["metric"] = name
	RecordMetric(name, fields)
}
