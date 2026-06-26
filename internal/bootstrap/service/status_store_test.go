// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"fmt"
	"sync"
	"testing"
	"time"

	bootstrappb "github.com/choysum-dev/choysum/internal/bootstrap/proto/bootstrappb"
)

func TestStatusStoreBeginOperationIdempotentReuse(t *testing.T) {
	store := newMemoryStatusStore(func() time.Time { return time.Unix(100, 0) })

	op1, outcome1, _ := store.beginOperation("op-1", "idem-a", 1000)
	if outcome1 != beginCreated {
		t.Fatalf("first begin outcome = %v, want %v", outcome1, beginCreated)
	}

	op2, outcome2, _ := store.beginOperation("op-2", "idem-a", 1000)
	if outcome2 != beginReused {
		t.Fatalf("second begin outcome = %v, want %v", outcome2, beginReused)
	}
	if op1.OperationID != op2.OperationID {
		t.Fatalf("reused operation id = %q, want %q", op2.OperationID, op1.OperationID)
	}
}

func TestStatusStoreBeginOperationConflictWhenAnotherRunning(t *testing.T) {
	store := newMemoryStatusStore(func() time.Time { return time.Unix(200, 0) })

	_, outcome1, _ := store.beginOperation("op-1", "idem-a", 1000)
	if outcome1 != beginCreated {
		t.Fatalf("first begin outcome = %v, want %v", outcome1, beginCreated)
	}

	_, outcome2, conflictID := store.beginOperation("op-2", "idem-b", 1000)
	if outcome2 != beginConflict {
		t.Fatalf("second begin outcome = %v, want %v", outcome2, beginConflict)
	}
	if conflictID != "op-1" {
		t.Fatalf("conflict operation id = %q, want op-1", conflictID)
	}
}

func TestStatusStoreMarkSucceededClearsActiveOperation(t *testing.T) {
	store := newMemoryStatusStore(func() time.Time { return time.Unix(300, 0) })

	_, outcome1, _ := store.beginOperation("op-1", "idem-a", 1000)
	if outcome1 != beginCreated {
		t.Fatalf("first begin outcome = %v, want %v", outcome1, beginCreated)
	}

	store.markRunning("op-1", bootstrappb.InitializationStage_INITIALIZATION_STAGE_ENSURE_MINIMAL_RUNTIME, 30)
	store.markSucceeded("op-1", "/web/login")

	op1, ok := store.getOperation("op-1")
	if !ok {
		t.Fatal("expected operation op-1 to exist")
	}
	if op1.State != bootstrappb.InitializationState_INITIALIZATION_STATE_SUCCEEDED {
		t.Fatalf("op-1 state = %v, want SUCCEEDED", op1.State)
	}
	if op1.Stage != bootstrappb.InitializationStage_INITIALIZATION_STAGE_DONE {
		t.Fatalf("op-1 stage = %v, want DONE", op1.Stage)
	}

	_, outcome2, _ := store.beginOperation("op-2", "idem-b", 1000)
	if outcome2 != beginCreated {
		t.Fatalf("new begin outcome = %v, want %v", outcome2, beginCreated)
	}
}

func TestStatusStoreStageTransitionsClearStageDetail(t *testing.T) {
	store := newMemoryStatusStore(func() time.Time { return time.Unix(500, 0) })

	_, outcome, _ := store.beginOperation("op-1", "idem-a", 1000)
	if outcome != beginCreated {
		t.Fatalf("begin outcome = %v, want %v", outcome, beginCreated)
	}

	store.markStageDetail("op-1", "collecting module graph")
	store.markStage("op-1", bootstrappb.InitializationStage_INITIALIZATION_STAGE_UPDATE_ADMIN, 55)
	op, ok := store.getOperation("op-1")
	if !ok {
		t.Fatal("expected operation op-1 to exist")
	}
	if op.StageDetail != "" {
		t.Fatalf("stage detail after markStage = %q, want empty", op.StageDetail)
	}

	store.markStageDetail("op-1", "building runtime assets")
	store.markRunning("op-1", bootstrappb.InitializationStage_INITIALIZATION_STAGE_SWITCH_MODE, 75)
	op, _ = store.getOperation("op-1")
	if op.StageDetail != "" {
		t.Fatalf("stage detail after markRunning = %q, want empty", op.StageDetail)
	}

	store.markStageDetail("op-1", "finalizing")
	store.markFailed("op-1", bootstrappb.InitializationStage_INITIALIZATION_STAGE_UPDATE_ADMIN, "BOOTSTRAP_FAILED", "failed")
	op, _ = store.getOperation("op-1")
	if op.StageDetail != "" {
		t.Fatalf("stage detail after markFailed = %q, want empty", op.StageDetail)
	}

	_, outcome, _ = store.beginOperation("op-2", "idem-b", 1000)
	if outcome != beginCreated {
		t.Fatalf("second begin outcome = %v, want %v", outcome, beginCreated)
	}
	store.markStageDetail("op-2", "ready")
	store.markSucceeded("op-2", "/web/login")
	op, _ = store.getOperation("op-2")
	if op.StageDetail != "" {
		t.Fatalf("stage detail after markSucceeded = %q, want empty", op.StageDetail)
	}
}

func TestStatusStoreContractBeginAndMarkSucceededConcurrentRelease(t *testing.T) {
	store := newInMemoryStatusStore(func() time.Time { return time.Unix(400, 0) })

	_, outcome, _ := store.beginOperation("op-1", "idem-1", 1000)
	if outcome != beginCreated {
		t.Fatalf("initial begin outcome = %v, want %v", outcome, beginCreated)
	}

	const contenders = 8
	type beginResult struct {
		op            operationSnapshot
		outcome       beginOutcome
		conflictingID string
	}

	start := make(chan struct{})
	markDone := make(chan struct{})
	results := make(chan beginResult, contenders)

	var wg sync.WaitGroup
	for i := 0; i < contenders; i++ {
		opID := fmt.Sprintf("op-race-%d", i+2)
		idemKey := fmt.Sprintf("idem-race-%d", i+2)
		wg.Add(1)
		go func(operationID, idempotencyKey string) {
			defer wg.Done()
			<-start
			op, outcome, conflictingID := store.beginOperation(operationID, idempotencyKey, 1000)
			results <- beginResult{op: op, outcome: outcome, conflictingID: conflictingID}
		}(opID, idemKey)
	}

	go func() {
		<-start
		store.markSucceeded("op-1", "/web/login")
		close(markDone)
	}()

	close(start)
	wg.Wait()
	<-markDone
	close(results)

	createdCount := 0
	conflictCount := 0
	createdOperationID := ""
	for r := range results {
		switch r.outcome {
		case beginCreated:
			createdCount++
			if createdOperationID == "" {
				createdOperationID = r.op.OperationID
			}
		case beginConflict:
			conflictCount++
			if r.conflictingID == "" {
				t.Fatal("conflict result must include conflicting operation id")
			}
		case beginReused:
			t.Fatal("unexpected beginReused for unique idempotency keys")
		default:
			t.Fatalf("unexpected begin outcome: %v", r.outcome)
		}
	}

	if createdCount+conflictCount != contenders {
		t.Fatalf("race outcomes mismatch: created=%d conflict=%d contenders=%d", createdCount, conflictCount, contenders)
	}

	postOp, postOutcome, postConflictID := store.beginOperation("op-post", "idem-post", 1000)
	switch postOutcome {
	case beginCreated:
		createdCount++
		createdOperationID = postOp.OperationID
	case beginConflict:
		if postConflictID == "" {
			t.Fatal("post-release conflict must include conflicting operation id")
		}
	case beginReused:
		t.Fatal("unexpected beginReused for post operation")
	default:
		t.Fatalf("unexpected post begin outcome: %v", postOutcome)
	}

	if createdCount != 1 {
		t.Fatalf("exactly one operation must be created after release, got %d", createdCount)
	}

	op1, ok := store.getOperation("op-1")
	if !ok {
		t.Fatal("expected operation op-1 to exist")
	}
	if op1.State != bootstrappb.InitializationState_INITIALIZATION_STATE_SUCCEEDED {
		t.Fatalf("op-1 state = %v, want SUCCEEDED", op1.State)
	}

	createdOp, ok := store.getOperation(createdOperationID)
	if !ok {
		t.Fatalf("expected created operation %q to exist", createdOperationID)
	}
	if createdOp.State != bootstrappb.InitializationState_INITIALIZATION_STATE_PENDING {
		t.Fatalf("created operation state = %v, want PENDING", createdOp.State)
	}
}
