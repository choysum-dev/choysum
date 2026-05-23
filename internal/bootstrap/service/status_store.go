// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"sync"
	"time"

	bootstrappb "github.com/choysum-dev/choysum/internal/bootstrap/proto/bootstrappb"
)

type beginOutcome int

const (
	beginCreated beginOutcome = iota
	beginReused
	beginConflict
)

type operationSnapshot struct {
	OperationID     string
	IdempotencyKey  string
	State           bootstrappb.InitializationState
	Stage           bootstrappb.InitializationStage
	ProgressPercent int32
	ReadyForLogin   bool
	RedirectURL     string
	ErrorCode       string
	ErrorMessage    string
	NextPollAfterMs int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     *time.Time
}

type bootstrapStatusStore interface {
	beginOperation(operationID, idempotencyKey string, nextPollAfterMs int64) (operationSnapshot, beginOutcome, string)
	getOperation(operationID string) (operationSnapshot, bool)
	markRunning(operationID string, stage bootstrappb.InitializationStage, progressPercent int32)
	markStage(operationID string, stage bootstrappb.InitializationStage, progressPercent int32)
	markFailed(operationID string, stage bootstrappb.InitializationStage, code, message string)
	markSucceeded(operationID string, redirectURL string)
}

type memoryStatusStore struct {
	mu               sync.RWMutex
	now              func() time.Time
	operations       map[string]operationSnapshot
	idempotencyIndex map[string]string
	activeOperation  string
}

var _ bootstrapStatusStore = (*memoryStatusStore)(nil)

func newInMemoryStatusStore(nowFn func() time.Time) bootstrapStatusStore {
	return newMemoryStatusStore(nowFn)
}

func newMemoryStatusStore(nowFn func() time.Time) *memoryStatusStore {
	if nowFn == nil {
		nowFn = time.Now
	}
	return &memoryStatusStore{
		now:              nowFn,
		operations:       map[string]operationSnapshot{},
		idempotencyIndex: map[string]string{},
	}
}

func (s *memoryStatusStore) beginOperation(operationID, idempotencyKey string, nextPollAfterMs int64) (operationSnapshot, beginOutcome, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idempotencyKey != "" {
		if existingID, ok := s.idempotencyIndex[idempotencyKey]; ok {
			existing, ok := s.operations[existingID]
			if ok {
				return existing, beginReused, ""
			}
		}
	}

	if s.activeOperation != "" {
		if active, ok := s.operations[s.activeOperation]; ok {
			return active, beginConflict, s.activeOperation
		}
	}

	now := s.now().UTC()
	op := operationSnapshot{
		OperationID:     operationID,
		IdempotencyKey:  idempotencyKey,
		State:           bootstrappb.InitializationState_INITIALIZATION_STATE_PENDING,
		Stage:           bootstrappb.InitializationStage_INITIALIZATION_STAGE_UNSPECIFIED,
		ProgressPercent: 0,
		NextPollAfterMs: nextPollAfterMs,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	s.operations[operationID] = op
	if idempotencyKey != "" {
		s.idempotencyIndex[idempotencyKey] = operationID
	}
	s.activeOperation = operationID

	return op, beginCreated, ""
}

func (s *memoryStatusStore) getOperation(operationID string) (operationSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	op, ok := s.operations[operationID]
	return op, ok
}

func (s *memoryStatusStore) markRunning(operationID string, stage bootstrappb.InitializationStage, progressPercent int32) {
	s.update(operationID, func(op *operationSnapshot) {
		op.State = bootstrappb.InitializationState_INITIALIZATION_STATE_RUNNING
		op.Stage = stage
		op.ProgressPercent = clampProgress(progressPercent)
		op.ErrorCode = ""
		op.ErrorMessage = ""
		op.CompletedAt = nil
	})
}

func (s *memoryStatusStore) markStage(operationID string, stage bootstrappb.InitializationStage, progressPercent int32) {
	s.update(operationID, func(op *operationSnapshot) {
		op.Stage = stage
		op.ProgressPercent = clampProgress(progressPercent)
		if op.State == bootstrappb.InitializationState_INITIALIZATION_STATE_PENDING {
			op.State = bootstrappb.InitializationState_INITIALIZATION_STATE_RUNNING
		}
	})
}

func (s *memoryStatusStore) markFailed(operationID string, stage bootstrappb.InitializationStage, code, message string) {
	s.update(operationID, func(op *operationSnapshot) {
		now := s.now().UTC()
		op.State = bootstrappb.InitializationState_INITIALIZATION_STATE_FAILED
		op.Stage = stage
		op.ErrorCode = code
		op.ErrorMessage = message
		op.ReadyForLogin = false
		op.RedirectURL = ""
		op.CompletedAt = &now
		if s.activeOperation == operationID {
			s.activeOperation = ""
		}
	})
}

func (s *memoryStatusStore) markSucceeded(operationID string, redirectURL string) {
	s.update(operationID, func(op *operationSnapshot) {
		now := s.now().UTC()
		op.State = bootstrappb.InitializationState_INITIALIZATION_STATE_SUCCEEDED
		op.Stage = bootstrappb.InitializationStage_INITIALIZATION_STAGE_DONE
		op.ProgressPercent = 100
		op.ReadyForLogin = true
		op.RedirectURL = redirectURL
		op.ErrorCode = ""
		op.ErrorMessage = ""
		op.CompletedAt = &now
		if s.activeOperation == operationID {
			s.activeOperation = ""
		}
	})
}

func (s *memoryStatusStore) update(operationID string, mutator func(op *operationSnapshot)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	op, ok := s.operations[operationID]
	if !ok {
		return
	}

	mutator(&op)
	op.UpdatedAt = s.now().UTC()
	s.operations[operationID] = op
}

func clampProgress(progress int32) int32 {
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}
