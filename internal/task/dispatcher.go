// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/choysum-dev/choysum/internal/jobtoken"
	"github.com/choysum-dev/choysum/pkg/bus"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"github.com/choysum-dev/choysum/pkg/scope"
	taskcontract "github.com/choysum-dev/choysum/pkg/task"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/dynamicpb"
	"gorm.io/gorm"
)

type Dispatcher struct {
	runtimeScope scope.Scope
	runtimeOpts  runtimeOptions
	dialer       client.ServiceDialer
	queue        taskcontract.TaskQueue
	events       bus.EventBus
	stopCh       chan struct{}
	wakeCh       chan struct{}
	wg           sync.WaitGroup
	stopOnce     sync.Once
	wakeSub      bus.Subscription
	sema         chan struct{}
	interval     time.Duration
	appMu        sync.Mutex
	appInFly     map[string]int
	readyMu      sync.Mutex
	readySet     map[string]struct{}
	readyQ       []Job
}

const internalKeyHeader = "x-choysum-internal-key"

type unauthenticatedError struct {
	err error
}

func (e unauthenticatedError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return "unauthenticated"
}

func (e unauthenticatedError) Unwrap() error {
	return e.err
}

func NewDispatcher(runtimeScope scope.Scope, dialer client.ServiceDialer) *Dispatcher {
	return NewDispatcherWithRuntime(runtimeScope, dialer, taskcontract.Runtime{})
}

func NewDispatcherWithRuntime(runtimeScope scope.Scope, dialer client.ServiceDialer, runtime taskcontract.Runtime) *Dispatcher {
	opts := runtimeOptionsFromScope(runtimeScope)
	runtime = runtimeWithDefaultTaskRuntimeDeps(runtimeScope, runtime)
	return &Dispatcher{
		runtimeScope: runtimeScope,
		runtimeOpts:  opts,
		dialer:       dialer,
		queue:        runtime.Queue,
		events:       runtime.Events,
		stopCh:       make(chan struct{}),
		wakeCh:       make(chan struct{}, 1),
		sema:         make(chan struct{}, opts.dispatchMaxConcurrency),
		interval:     opts.dispatchPollInterval,
		appInFly:     map[string]int{},
		readySet:     map[string]struct{}{},
	}
}

func (d *Dispatcher) resolvedRuntimeOptions() runtimeOptions {
	if d != nil && d.runtimeOpts.initialized {
		return d.runtimeOpts
	}
	if d != nil {
		return runtimeOptionsFromScope(d.runtimeScope)
	}
	return newRuntimeOptions(scope.DatabaseRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false)
}

func (d *Dispatcher) Start() {
	if d.events != nil {
		if sub, err := d.events.Subscribe(bus.TopicDispatchWakeup, func(ctx context.Context, event bus.Event) {
			d.Wakeup(event.Source)
		}); err == nil {
			d.wakeSub = sub
		}
	}
	d.wg.Add(1)
	go d.loop()
	go d.pollOnce("startup")
}

func (d *Dispatcher) publishWakeup(source string) {
	if d == nil || d.events == nil {
		return
	}
	_ = d.events.Publish(context.Background(), bus.Event{
		Topic:  bus.TopicDispatchWakeup,
		Source: source,
		At:     time.Now().UTC(),
	})
}

func (d *Dispatcher) Stop() {
	if d == nil {
		return
	}
	d.stopOnce.Do(func() {
		close(d.stopCh)
	})
	if d.wakeSub != nil {
		_ = d.wakeSub.Close()
		d.wakeSub = nil
	}
	d.wg.Wait()
}

func (d *Dispatcher) loop() {
	defer d.wg.Done()
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-d.wakeCh:
			d.pollOnce("wake")
		case <-ticker.C:
			d.pollOnce("poll")
		}
	}
}

func (d *Dispatcher) Wakeup(source string) {
	if d == nil {
		return
	}
	select {
	case d.wakeCh <- struct{}{}:
	default:
	}
	if source == "" {
		source = "unknown"
	}
	d.emitMetric("task_wakeup_events", map[string]any{"source": source})
}

func (d *Dispatcher) pollOnce(source string) {
	if d.runtimeScope == nil || d.queue == nil {
		return
	}
	ctx := d.runtimeScope.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC()

	available := cap(d.sema) - len(d.sema)
	if available < 0 {
		available = 0
	}
	batchSize := available
	pollLimit := d.fetchBatchSize()
	if pollLimit <= 0 {
		pollLimit = cap(d.sema) * 4
		if pollLimit < 1 {
			pollLimit = 1
		}
	}
	queueJobs, err := d.queue.ListReady(ctx, now, pollLimit)
	if err != nil {
		d.runtimeScope.Logger().Warn("task dispatch poll failed", "error", err)
		return
	}
	jobs := make([]Job, 0, len(queueJobs))
	for _, queueJob := range queueJobs {
		jobs = append(jobs, *queueJobToModel(queueJob))
	}
	if source == "poll" && len(jobs) > 0 {
		d.emitMetric("task_poll_fallback_hits", map[string]any{"count": len(jobs)})
	}

	if len(jobs) > 0 {
		d.enqueueReady(jobs)
	}
	if batchSize <= 0 {
		return
	}

	selected := d.popReadyForDispatch(batchSize)
	if len(selected) == 0 {
		return
	}
	for _, job := range selected {
		if !d.reserveAppSlot(job.TargetApp) {
			continue
		}
		claimed, err := d.queue.TryClaim(ctx, job.Id, now)
		if err != nil || !claimed {
			d.releaseAppSlot(job.TargetApp)
			continue
		}
		d.sema <- struct{}{}
		d.wg.Add(1)
		go func(jobId string, targetApp string) {
			defer func() {
				<-d.sema
				d.releaseAppSlot(targetApp)
				d.wg.Done()
			}()
			d.handleJob(jobId)
		}(job.Id, job.TargetApp)
	}
}

func (d *Dispatcher) fetchBatchSize() int {
	return d.resolvedRuntimeOptions().dispatchFetchBatchSize
}

func (d *Dispatcher) readyQueueMax() int {
	return d.resolvedRuntimeOptions().dispatchReadyQueueMax
}

func (d *Dispatcher) enqueueReady(jobs []Job) {
	if len(jobs) == 0 {
		return
	}
	max := d.readyQueueMax()
	d.readyMu.Lock()
	defer d.readyMu.Unlock()
	for _, job := range jobs {
		if _, ok := d.readySet[job.Id]; ok {
			continue
		}
		if max > 0 && len(d.readyQ) >= max {
			break
		}
		d.readyQ = append(d.readyQ, job)
		d.readySet[job.Id] = struct{}{}
	}
}

func (d *Dispatcher) popReadyForDispatch(limit int) []Job {
	if limit <= 0 {
		return nil
	}
	d.readyMu.Lock()
	defer d.readyMu.Unlock()
	if len(d.readyQ) == 0 {
		return nil
	}
	selected := d.selectDispatchJobs(d.readyQ, limit)
	if len(selected) == 0 {
		return nil
	}
	selectedSet := map[string]struct{}{}
	for _, job := range selected {
		selectedSet[job.Id] = struct{}{}
		delete(d.readySet, job.Id)
	}
	filtered := d.readyQ[:0]
	for _, job := range d.readyQ {
		if _, ok := selectedSet[job.Id]; ok {
			continue
		}
		filtered = append(filtered, job)
	}
	d.readyQ = filtered
	return selected
}

func (d *Dispatcher) isSQLite() bool {
	return strings.EqualFold(strings.TrimSpace(d.resolvedRuntimeOptions().dbDialect), "sqlite")
}

func (d *Dispatcher) selectDispatchJobs(jobs []Job, limit int) []Job {
	if limit <= 0 || len(jobs) == 0 {
		return nil
	}
	groups := map[string][]Job{}
	order := make([]string, 0, len(jobs))
	for _, job := range jobs {
		if _, ok := groups[job.TargetApp]; !ok {
			order = append(order, job.TargetApp)
		}
		groups[job.TargetApp] = append(groups[job.TargetApp], job)
	}
	selected := make([]Job, 0, limit)
	for len(selected) < limit {
		progressed := false
		for _, app := range order {
			if len(selected) >= limit {
				break
			}
			if len(groups[app]) == 0 {
				continue
			}
			if !d.canDispatchForApp(app) {
				continue
			}
			selected = append(selected, groups[app][0])
			groups[app] = groups[app][1:]
			progressed = true
		}
		if !progressed {
			break
		}
	}
	return selected
}

func (d *Dispatcher) maxConcurrencyPerApp() int {
	return d.resolvedRuntimeOptions().dispatchMaxConcurrencyPerApp
}

func (d *Dispatcher) canDispatchForApp(app string) bool {
	cap := d.maxConcurrencyPerApp()
	if cap <= 0 {
		return true
	}
	d.appMu.Lock()
	defer d.appMu.Unlock()
	return d.appInFly[app] < cap
}

func (d *Dispatcher) reserveAppSlot(app string) bool {
	cap := d.maxConcurrencyPerApp()
	if cap <= 0 {
		return true
	}
	d.appMu.Lock()
	defer d.appMu.Unlock()
	if d.appInFly[app] >= cap {
		return false
	}
	d.appInFly[app]++
	return true
}

func (d *Dispatcher) releaseAppSlot(app string) {
	cap := d.maxConcurrencyPerApp()
	if cap <= 0 {
		return
	}
	d.appMu.Lock()
	defer d.appMu.Unlock()
	if d.appInFly[app] > 1 {
		d.appInFly[app]--
		return
	}
	delete(d.appInFly, app)
}

func (d *Dispatcher) tryClaim(db *gorm.DB, jobId string, now time.Time) bool {
	if d.queue != nil {
		ctx := context.Background()
		if d.runtimeScope != nil && d.runtimeScope.Context() != nil {
			ctx = d.runtimeScope.Context()
		}
		claimed, err := d.queue.TryClaim(ctx, jobId, now)
		return err == nil && claimed
	}
	if db == nil {
		return false
	}
	res := db.Model(&Job{}).
		Where("id = ? AND status = ?", jobId, "queued").
		Updates(map[string]any{"status": "dispatching", "updated_at": now})
	return res.Error == nil && res.RowsAffected > 0
}

func (d *Dispatcher) handleJob(jobId string) {
	if d.runtimeScope == nil || d.queue == nil {
		return
	}
	if d.dialer == nil {
		return
	}
	defer d.emitMetric("task_inflight", map[string]any{"count": len(d.sema)})
	ctx := d.runtimeScope.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	queuedJob, err := d.queue.Get(ctx, jobId)
	if err != nil || queuedJob == nil {
		return
	}
	job := queueJobToModel(*queuedJob)

	if job.CancelRequestedAt != nil {
		d.markCancelled(nil, job, "cancel requested")
		return
	}

	queuedAge := time.Since(job.RunAfter)
	if queuedAge < 0 {
		queuedAge = 0
	}
	d.emitMetric("task_jobs_queued_age", map[string]any{
		"target_app":  job.TargetApp,
		"full_method": job.FullMethod,
		"age_ms":      queuedAge.Milliseconds(),
	})

	attempt := job.Attempt + 1
	if job.MaxAttempts > 0 && attempt > job.MaxAttempts {
		d.failJob(nil, job, map[string]any{"message": "max attempts reached"}, nil)
		return
	}

	// Issue JobToken (do not consume attempt on failure).
	jobTokenTTL := time.Duration(d.resolvedRuntimeOptions().dispatchJobTokenTTLms) * time.Millisecond
	tokenResp, err := d.issueTaskJobToken(ctx, jobtoken.IssueRequest{
		JobId:             job.Id,
		TargetApp:         job.TargetApp,
		FullMethod:        job.FullMethod,
		SchedulerUserId:   job.SchedulerUserId,
		TriggeredByUserId: job.TriggeredByUserId,
		Attempt:           int64(attempt),
		TTL:               jobTokenTTL,
	})
	if err != nil {
		d.retryJob(nil, job, 0, map[string]any{"message": fmt.Sprintf("issue job token failed: %v", err)}, "token_issue_failed")
		return
	}

	// Update attempt now that we have a token.
	if err := d.queue.UpdateAttempt(ctx, job.Id, attempt, time.Now().UTC()); err != nil {
		d.retryJob(nil, job, 0, map[string]any{"message": fmt.Sprintf("update attempt failed: %v", err)}, "token_attempt_update_failed")
		return
	}
	job.Attempt = attempt

	execStart := time.Now().UTC()
	resp, callErr := d.callExecuteJob(ctx, job, tokenResp.AccessToken)
	if callErr != nil {
		if isUnauthenticated(callErr) {
			resp2, handled := d.handleUnauthenticated(ctx, nil, job, attempt)
			if handled {
				if resp2 == nil {
					return
				}
				resp = resp2
				attempt = job.Attempt
				callErr = nil
			}
		}
		if callErr != nil {
			if job.MaxAttempts > 0 && attempt >= job.MaxAttempts {
				d.failJob(nil, job, map[string]any{"message": callErr.Error()}, nil)
				return
			}
			d.retryJob(nil, job, d.backoffMs(attempt), map[string]any{"message": callErr.Error()}, "rpc_error")
			return
		}
	}
	execDuration := time.Since(execStart)
	d.emitMetric("task_execute_duration", map[string]any{
		"target_app":  job.TargetApp,
		"full_method": job.FullMethod,
		"duration_ms": execDuration.Milliseconds(),
	})

	if resp != nil && resp.Status == "FAILED_NON_RETRYABLE" && isUnauthenticatedResponse(resp) {
		resp2, handled := d.handleUnauthenticated(ctx, nil, job, attempt)
		if handled {
			if resp2 == nil {
				return
			}
			resp = resp2
			attempt = job.Attempt
		}
	}
	switch resp.Status {
	case "SUCCEEDED":
		d.emitMetric("task_dispatch_result", map[string]any{"status": "succeeded", "target_app": job.TargetApp, "full_method": job.FullMethod})
		d.succeedJob(nil, job, resp.Result)
	case "FAILED_NON_RETRYABLE":
		d.emitMetric("task_dispatch_result", map[string]any{"status": "failed_non_retryable", "target_app": job.TargetApp, "full_method": job.FullMethod})
		d.failJob(nil, job, resp.Error, resp.Result)
	case "FAILED_RETRYABLE":
		d.emitMetric("task_dispatch_result", map[string]any{"status": "failed_retryable", "target_app": job.TargetApp, "full_method": job.FullMethod})
		if job.MaxAttempts > 0 && attempt >= job.MaxAttempts {
			d.failJob(nil, job, resp.Error, resp.Result)
			return
		}
		d.retryJob(nil, job, d.backoffMs(attempt), resp.Error, "failed_retryable")
	case "ALREADY_RUNNING", "RESOURCE_BUSY":
		d.emitMetric("task_dispatch_result", map[string]any{"status": strings.ToLower(resp.Status), "target_app": job.TargetApp, "full_method": job.FullMethod})
		retryAfter := resp.RetryAfterMs
		if retryAfter <= 0 {
			retryAfter = 1000
		}
		d.retryJob(nil, job, retryAfter, resp.Error, "retry_after")
	case "CANCELLED":
		d.markCancelled(nil, job, "cancelled by worker")
	default:
		d.retryJob(nil, job, d.backoffMs(attempt), map[string]any{"message": "unknown status"}, "unknown_status")
	}
}

func (d *Dispatcher) handleUnauthenticated(ctx context.Context, db *gorm.DB, job *Job, attempt int) (*ExecuteJobResponse, bool) {
	if job == nil || d.queue == nil {
		return nil, true
	}
	refreshAttempt := attempt + 1
	if job.MaxAttempts > 0 && refreshAttempt > job.MaxAttempts {
		d.failJob(db, job, map[string]any{"message": "unauthenticated and max attempts reached"}, nil)
		return nil, true
	}

	jobTokenTTL := time.Duration(d.resolvedRuntimeOptions().dispatchJobTokenTTLms) * time.Millisecond
	refreshed, err := d.issueTaskJobToken(ctx, jobtoken.IssueRequest{
		JobId:             job.Id,
		TargetApp:         job.TargetApp,
		FullMethod:        job.FullMethod,
		SchedulerUserId:   job.SchedulerUserId,
		TriggeredByUserId: job.TriggeredByUserId,
		Attempt:           int64(refreshAttempt),
		TTL:               jobTokenTTL,
	})
	if err != nil {
		d.retryJob(db, job, 0, map[string]any{"message": fmt.Sprintf("issue job token failed: %v", err)}, "token_issue_failed")
		return nil, true
	}

	if err := d.queue.UpdateAttempt(ctx, job.Id, refreshAttempt, time.Now().UTC()); err != nil {
		d.retryJob(db, job, 0, map[string]any{"message": fmt.Sprintf("update attempt failed: %v", err)}, "token_attempt_update_failed")
		return nil, true
	}
	job.Attempt = refreshAttempt

	resp, callErr := d.callExecuteJob(ctx, job, refreshed.AccessToken)
	if callErr != nil {
		if isUnauthenticated(callErr) {
			d.failJob(db, job, map[string]any{"message": "unauthenticated after token refresh"}, nil)
			return nil, true
		}
		if job.MaxAttempts > 0 && refreshAttempt >= job.MaxAttempts {
			d.failJob(db, job, map[string]any{"message": callErr.Error()}, nil)
			return nil, true
		}
		d.retryJob(db, job, d.backoffMs(refreshAttempt), map[string]any{"message": callErr.Error()}, "refresh_call_error")
		return nil, true
	}
	return resp, true
}

func (d *Dispatcher) issueTaskJobToken(ctx context.Context, req jobtoken.IssueRequest) (*jobtoken.IssueResponse, error) {
	opts := d.resolvedRuntimeOptions()
	return jobtoken.IssueTaskJobToken(
		ctx,
		&config.AuthConfig{InternalKey: opts.authInternalKey},
		opts.serverEnvironment,
		d.dialer,
		req,
	)
}

func (d *Dispatcher) succeedJob(db *gorm.DB, job *Job, result any) {
	if job == nil || d.queue == nil {
		return
	}
	sanitized, _ := SanitizeResultWithMaxBytes(d.resolvedRuntimeOptions().sanitizeResultMaxBytes, result)
	resultJSON, _ := json.Marshal(sanitized.Value)
	now := time.Now().UTC()
	ctx := context.Background()
	if d.runtimeScope != nil && d.runtimeScope.Context() != nil {
		ctx = d.runtimeScope.Context()
	}
	_ = d.queue.MarkSucceeded(ctx, job.Id, taskcontract.QueueSuccess{
		FinishedAt:      now,
		UpdatedAt:       now,
		ResultJSON:      resultJSON,
		ResultHash:      sanitized.Hash,
		ResultTruncated: sanitized.Truncated,
	})
}

func (d *Dispatcher) failJob(db *gorm.DB, job *Job, errObj any, result any) {
	if job == nil || d.queue == nil {
		return
	}
	errSan, _ := SanitizeErrorWithMaxBytes(d.resolvedRuntimeOptions().sanitizeErrorMaxBytes, errObj)
	errJSON, _ := json.Marshal(errSan.Value)
	resultSan, _ := SanitizeResultWithMaxBytes(d.resolvedRuntimeOptions().sanitizeResultMaxBytes, result)
	resultJSON, _ := json.Marshal(resultSan.Value)
	now := time.Now().UTC()
	ctx := context.Background()
	if d.runtimeScope != nil && d.runtimeScope.Context() != nil {
		ctx = d.runtimeScope.Context()
	}
	_ = d.queue.MarkFailed(ctx, job.Id, taskcontract.QueueFailure{
		FinishedAt:      now,
		UpdatedAt:       now,
		ErrorJSON:       errJSON,
		ErrorHash:       errSan.Hash,
		ErrorTruncated:  errSan.Truncated,
		ResultJSON:      resultJSON,
		ResultHash:      resultSan.Hash,
		ResultTruncated: resultSan.Truncated,
	})
}

func (d *Dispatcher) retryJob(db *gorm.DB, job *Job, retryAfterMs int64, errObj any, source string) {
	if job == nil || d.queue == nil {
		return
	}
	original := retryAfterMs
	if retryAfterMs <= 0 {
		if d.runtimeScope != nil {
			d.runtimeScope.Logger().Warn("task retry delay adjusted", "reason", "non_positive", "job_id", job.Id, "target_app", job.TargetApp, "full_method", job.FullMethod, "retry_after_ms", retryAfterMs, "adjusted_retry_after_ms", int64(1000), "source", source)
		}
		d.emitMetric("task_retry_after_adjusted", map[string]any{
			"reason":      "non_positive",
			"source":      source,
			"target_app":  job.TargetApp,
			"full_method": job.FullMethod,
			"value_ms":    original,
		})
		retryAfterMs = 1000
	}
	capMs := int64(300000)
	if runtimeCap := d.resolvedRuntimeOptions().dispatchRetryAfterMsCap; runtimeCap > 0 {
		capMs = runtimeCap
	}
	if retryAfterMs > capMs {
		if d.runtimeScope != nil {
			d.runtimeScope.Logger().Warn("task retry delay adjusted", "reason", "capped", "job_id", job.Id, "target_app", job.TargetApp, "full_method", job.FullMethod, "retry_after_ms", retryAfterMs, "adjusted_retry_after_ms", capMs, "cap_ms", capMs, "source", source)
		}
		d.emitMetric("task_retry_after_adjusted", map[string]any{
			"reason":      "capped",
			"source":      source,
			"target_app":  job.TargetApp,
			"full_method": job.FullMethod,
			"value_ms":    retryAfterMs,
			"cap_ms":      capMs,
		})
		retryAfterMs = capMs
	}
	d.emitMetric("task_retry_after", map[string]any{
		"source":         source,
		"target_app":     job.TargetApp,
		"full_method":    job.FullMethod,
		"retry_after_ms": retryAfterMs,
	})
	errSan, _ := SanitizeErrorWithMaxBytes(d.resolvedRuntimeOptions().sanitizeErrorMaxBytes, errObj)
	errJSON, _ := json.Marshal(errSan.Value)
	now := time.Now().UTC()
	runAfter := now.Add(time.Duration(retryAfterMs) * time.Millisecond)
	ctx := context.Background()
	if d.runtimeScope != nil && d.runtimeScope.Context() != nil {
		ctx = d.runtimeScope.Context()
	}
	_ = d.queue.Retry(ctx, job.Id, taskcontract.QueueRetry{
		RunAfter:       runAfter,
		UpdatedAt:      now,
		ErrorJSON:      errJSON,
		ErrorHash:      errSan.Hash,
		ErrorTruncated: errSan.Truncated,
	})
	if d.interval > 0 && retryAfterMs <= d.interval.Milliseconds() {
		d.publishWakeup("run_after")
	}
}

func (d *Dispatcher) markCancelled(db *gorm.DB, job *Job, reason string) {
	if job == nil || d.queue == nil {
		return
	}
	now := time.Now().UTC()
	errJSON, _ := json.Marshal(map[string]any{"message": reason})
	ctx := context.Background()
	if d.runtimeScope != nil && d.runtimeScope.Context() != nil {
		ctx = d.runtimeScope.Context()
	}
	_ = d.queue.MarkCancelled(ctx, job.Id, taskcontract.QueueCancellation{
		CancelledAt: now,
		FinishedAt:  now,
		UpdatedAt:   now,
		ErrorJSON:   errJSON,
	})
}

func (d *Dispatcher) backoffMs(attempt int) int64 {
	if attempt <= 0 {
		return 1000
	}
	base := float64(1000)
	maxMs := int64(60000)
	if runtimeMax := d.resolvedRuntimeOptions().dispatchBackoffMaxMs; runtimeMax > 0 {
		maxMs = runtimeMax
	}
	exp := math.Min(float64(maxMs), base*math.Pow(2, float64(attempt-1)))
	jitter := rand.Float64()*0.2 + 0.9
	val := int64(exp * jitter)
	if val < 1000 {
		val = 1000
	}
	if val > maxMs {
		val = maxMs
	}
	return val
}

type ExecuteJobResponse struct {
	Status       string
	Result       any
	Error        any
	RetryAfterMs int64
}

func (d *Dispatcher) callExecuteJob(ctx context.Context, job *Job, token string) (*ExecuteJobResponse, error) {
	md, err := taskWorkerMethod(job.TargetApp)
	if err != nil {
		return nil, err
	}
	reqDesc := md.Input()
	respDesc := md.Output()

	payload := map[string]interface{}{}
	if len(job.PayloadJson) > 0 {
		_ = json.Unmarshal(job.PayloadJson, &payload)
	}

	request := map[string]interface{}{
		"job_id":               job.Id,
		"attempt":              job.Attempt,
		"full_method":          job.FullMethod,
		"payload":              payload,
		"scheduler_user_id":    job.SchedulerUserId,
		"triggered_by_user_id": job.TriggeredByUserId,
		"timeout_ms":           job.TimeoutMs,
	}

	reqMsg := dynamicpb.NewMessage(reqDesc)
	if err := converter.MapToMessage(request, reqMsg); err != nil {
		return nil, err
	}

	jobTimeoutMs := job.TimeoutMs
	if jobTimeoutMs <= 0 {
		jobTimeoutMs = d.resolvedRuntimeOptions().dispatchDefaultJobTimeoutMs
	}
	if jobTimeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(jobTimeoutMs)*time.Millisecond)
		defer cancel()
	}

	mdOut := metadata.Pairs("authorization", "Bearer "+token)
	opts := d.resolvedRuntimeOptions()
	key := strings.TrimSpace(opts.authInternalKey)
	if key != "" && !strings.EqualFold(strings.TrimSpace(opts.serverEnvironment), "production") {
		mdOut.Set(internalKeyHeader, key)
	}
	ctx = metadata.NewOutgoingContext(ctx, mdOut)
	conn, err := d.dialer(ctx, job.TargetApp+".TaskWorker")
	if err != nil {
		return nil, err
	}

	fullMethod := "/" + job.TargetApp + ".TaskWorker/ExecuteJob"
	respMsg := dynamicpb.NewMessage(respDesc)
	start := time.Now()
	if err := conn.Invoke(ctx, fullMethod, reqMsg, respMsg); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.Unauthenticated {
				return nil, unauthenticatedError{err: err}
			}
			if st.Code() == codes.Unavailable {
				return nil, err
			}
		}
		return nil, err
	}
	latency := time.Since(start)
	d.emitMetric("task_dispatch_latency", map[string]any{
		"target_app":  job.TargetApp,
		"full_method": job.FullMethod,
		"latency_ms":  latency.Milliseconds(),
	})

	outMap, err := converter.MessageToMap(respMsg)
	if err != nil {
		return nil, err
	}

	resp := &ExecuteJobResponse{Status: normalizeExecuteJobStatus(outMap["status"])}
	if v, ok := outMap["result"]; ok {
		resp.Result = v
	}
	if v, ok := outMap["error"]; ok {
		resp.Error = v
	}
	if v, ok := outMap["retry_after_ms"]; ok {
		switch tv := v.(type) {
		case int64:
			resp.RetryAfterMs = tv
		case float64:
			resp.RetryAfterMs = int64(tv)
		case int:
			resp.RetryAfterMs = int64(tv)
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(tv), 10, 64); err == nil {
				resp.RetryAfterMs = parsed
			}
		}
	}
	return resp, nil
}

func isUnauthenticated(err error) bool {
	var unauth unauthenticatedError
	return errors.As(err, &unauth)
}

func isUnauthenticatedResponse(resp *ExecuteJobResponse) bool {
	if resp == nil || resp.Error == nil {
		return false
	}
	if code, ok := extractGrpcCode(resp.Error); ok {
		return code == int32(codes.Unauthenticated)
	}
	return false
}

func extractGrpcCode(errObj any) (int32, bool) {
	if errObj == nil {
		return 0, false
	}
	if m, ok := errObj.(map[string]any); ok {
		if v, ok := m["grpc_code"]; ok {
			switch tv := v.(type) {
			case int32:
				return tv, true
			case int64:
				return int32(tv), true
			case int:
				return int32(tv), true
			case float64:
				return int32(tv), true
			}
		}
	}
	return 0, false
}

func normalizeExecuteJobStatus(v any) string {
	switch tv := v.(type) {
	case string:
		trimmed := strings.TrimSpace(tv)
		if strings.HasPrefix(trimmed, "EXECUTE_JOB_STATUS_") {
			return strings.TrimPrefix(trimmed, "EXECUTE_JOB_STATUS_")
		}
		return trimmed
	case int32:
		return statusFromCode(int32(tv))
	case int64:
		return statusFromCode(int32(tv))
	case int:
		return statusFromCode(int32(tv))
	case float64:
		return statusFromCode(int32(tv))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func statusFromCode(code int32) string {
	switch code {
	case 1:
		return "SUCCEEDED"
	case 2:
		return "FAILED_NON_RETRYABLE"
	case 3:
		return "FAILED_RETRYABLE"
	case 4:
		return "ALREADY_RUNNING"
	case 5:
		return "RESOURCE_BUSY"
	case 6:
		return "CANCELLED"
	default:
		return "EXECUTE_JOB_STATUS_UNSPECIFIED"
	}
}

func (d *Dispatcher) emitMetric(name string, fields map[string]any) {
	if d.runtimeScope == nil {
		return
	}
	fields["metric"] = name
	RecordMetric(name, fields)
}
