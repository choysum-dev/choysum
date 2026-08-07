// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"context"
	"errors"
	"time"

	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"

)

type HistoryStore struct {
	runtimeScope scope.Scope
}

func NewHistoryStore(runtimeScope scope.Scope) *HistoryStore {
	if runtimeScope == nil {
		return nil
	}
	return &HistoryStore{runtimeScope: runtimeScope}
}

func (s *HistoryStore) session(ctx context.Context) *scope.Session {
	if s == nil || s.runtimeScope == nil {
		return nil
	}
	sess, _ := scope.SessionForScope(ctx, s.runtimeScope)
	return sess
}

func (s *HistoryStore) Prepare(ctx context.Context, script Script) (*modmeta.ModuleMigrationHistory, bool, error) {
	if s == nil || s.runtimeScope == nil {
		return nil, false, nil
	}
	db := s.session(ctx)
	if db == nil {
		return nil, false, gorm.ErrInvalidDB
	}

	var existing modmeta.ModuleMigrationHistory
	q := db.WithContext(ctx).Where(
		"module_name = ? AND version = ? AND phase = ? AND script = ?",
		script.ModuleName, script.Version, string(script.Phase), script.Name,
	)
	err := q.Take(&existing).Error
	if err == nil {
		if existing.Status == "success" && existing.Checksum == script.Checksum {
			return &existing, true, nil
		}
		return s.markRunning(ctx, &existing, script)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	entry := &modmeta.ModuleMigrationHistory{
		ModuleName: script.ModuleName,
		Version:    script.Version,
		Phase:      string(script.Phase),
		Script:     script.Name,
		Checksum:   script.Checksum,
		Status:     "running",
		StartedAt:  time.Now(),
		TraceId:    traceIdFromContext(ctx),
	}
	if err := db.WithContext(ctx).Create(entry).Error; err != nil {
		return nil, false, err
	}
	return entry, false, nil
}

func (s *HistoryStore) markRunning(ctx context.Context, entry *modmeta.ModuleMigrationHistory, script Script) (*modmeta.ModuleMigrationHistory, bool, error) {
	db := s.session(ctx)
	if db == nil {
		return nil, false, gorm.ErrInvalidDB
	}
	entry.Status = "running"
	entry.Checksum = script.Checksum
	entry.Error = ""
	entry.StartedAt = time.Now()
	entry.FinishedAt = time.Time{}
	entry.TraceId = traceIdFromContext(ctx)
	if err := db.WithContext(ctx).Save(entry).Error; err != nil {
		return nil, false, err
	}
	return entry, false, nil
}

func (s *HistoryStore) MarkSuccess(ctx context.Context, entry *modmeta.ModuleMigrationHistory) error {
	if s == nil || s.runtimeScope == nil || entry == nil {
		return nil
	}
	db := s.session(ctx)
	if db == nil {
		return gorm.ErrInvalidDB
	}
	entry.Status = "success"
	entry.FinishedAt = time.Now()
	entry.Error = ""
	return db.WithContext(ctx).Save(entry).Error
}

func (s *HistoryStore) MarkFailed(ctx context.Context, entry *modmeta.ModuleMigrationHistory, msg string) error {
	if s == nil || s.runtimeScope == nil || entry == nil {
		return nil
	}
	db := s.session(ctx)
	if db == nil {
		return gorm.ErrInvalidDB
	}
	entry.Status = "failed"
	entry.FinishedAt = time.Now()
	entry.Error = msg
	return db.WithContext(ctx).Save(entry).Error
}

func traceIdFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	sc := span.SpanContext()
	if sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}
