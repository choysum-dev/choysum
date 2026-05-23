// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/task"
	grpcclient "github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const internalKeyHeader = "x-choysum-internal-key"

type executeJobRequest struct {
	JobId             string
	Attempt           int
	FullMethod        string
	Payload           map[string]interface{}
	SchedulerUserId   string
	TriggeredByUserId string
	TimeoutMs         int64
}

type executeJobResponse struct {
	Status       string
	Result       any
	Error        map[string]any
	RetryAfterMs int64
}

func (s *ApplicationService) taskWorkerServiceDesc() (*grpc.ServiceDesc, error) {
	return s.taskWorkerAdapter().serviceDesc()
}

func (s *ApplicationService) taskWorkerMethodHandler() func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	return s.taskWorkerAdapter().methodHandler()
}

func (s *ApplicationService) taskWorkerUnaryHandler() grpc.UnaryHandler {
	return s.taskWorkerAdapter().unaryHandler()
}

func parseExecuteJobReq(reqMsg *dynamicpb.Message) (*executeJobRequest, error) {
	return taskWorkerAdapter{}.parseExecuteJobReq(reqMsg)
}

func (s *ApplicationService) validateJobToken(ctx context.Context, req *executeJobRequest) error {
	return s.guard().validateJobToken(ctx, req)
}

func (s *ApplicationService) invokeTargetMethod(ctx context.Context, runtimeScope scope.Scope, jsCtx map[string]interface{}, routing *jsengine.JsExecutionRouting, req *executeJobRequest) (any, error) {
	return s.runtime().invokeTargetMethod(ctx, runtimeScope, jsCtx, routing, req)
}

func (s *ApplicationService) buildExecuteJobResp(resp executeJobResponse) (any, error) {
	return s.taskWorkerAdapter().buildExecuteJobResp(resp)
}

func setTaskError(msg *dynamicpb.Message, errMap map[string]any) {
	taskWorkerAdapter{}.setTaskError(msg, errMap)
}

func statusToEnum(status string) protoreflect.EnumNumber {
	return taskWorkerAdapter{}.statusToEnum(status)
}

func toAnyMap(v any) (map[string]any, bool) {
	switch tv := v.(type) {
	case map[string]any:
		return tv, true
	case map[string]string:
		m := make(map[string]any, len(tv))
		for k, v := range tv {
			m[k] = v
		}
		return m, true
	default:
		return nil, false
	}
}

type taskExecutionStore struct {
	runtimeScope scope.Scope
	runtimeOpts  runtimeOptions
	ctx          context.Context
}

func (s taskExecutionStore) resolvedRuntimeOptions() runtimeOptions {
	if hasRuntimeOptions(s.runtimeOpts) {
		return s.runtimeOpts
	}
	if s.runtimeScope != nil {
		return runtimeOptionsFromScope(s.runtimeScope)
	}
	return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false)
}

func (s taskExecutionStore) db(ctx context.Context) *gorm.DB {
	if db := taskExecutionDBFromContext(ctx); db != nil {
		return db
	}
	if db := taskExecutionDBFromContext(s.ctx); db != nil {
		return db
	}
	if s.runtimeScope == nil || s.runtimeScope.Session() == nil {
		return nil
	}
	return s.runtimeScope.Session().DB
}

func taskExecutionDBFromContext(ctx context.Context) *gorm.DB {
	db, _ := scope.DBForScope(ctx, nil)
	return db
}

func (s taskExecutionStore) tryStart(ctx context.Context, req *executeJobRequest, leaseOwner string, leaseUntil time.Time, startedAt time.Time) (string, executeJobResponse, error) {
	db := s.db(ctx)
	payloadSan, _ := task.SanitizePayloadWithMaxBytes(s.resolvedRuntimeOptions().taskSanitizePayloadMaxBytes, req.Payload)
	payloadJSON, _ := json.Marshal(payloadSan.Value)

	rec := task.Execution{
		JobId:             req.JobId,
		Status:            "running",
		LeaseOwner:        leaseOwner,
		LeaseUntil:        &leaseUntil,
		Attempt:           req.Attempt,
		SchedulerUserId:   req.SchedulerUserId,
		TriggeredByUserId: req.TriggeredByUserId,
		FullMethod:        req.FullMethod,
		PayloadJson:       datatypes.JSON(payloadJSON),
		StartedAt:         &startedAt,
		CreatedAt:         startedAt,
		UpdatedAt:         startedAt,
	}

	if err := db.Create(&rec).Error; err != nil {
		if errorsIsDuplicate(err) {
			return s.handleExisting(ctx, req, leaseOwner)
		}
		return "FAILED_RETRYABLE", executeJobResponse{}, err
	}
	return "RUNNING", executeJobResponse{}, nil
}

func (s taskExecutionStore) handleExisting(ctx context.Context, req *executeJobRequest, leaseOwner string) (string, executeJobResponse, error) {
	db := s.db(ctx)
	var existing task.Execution
	if err := db.Where("job_id = ?", req.JobId).First(&existing).Error; err != nil {
		return "FAILED_RETRYABLE", executeJobResponse{}, err
	}
	now := time.Now().UTC()
	switch existing.Status {
	case "succeeded":
		return "SUCCEEDED", executeJobResponse{Status: "SUCCEEDED", Result: jsonToAny(existing.ResultJson)}, nil
	case "cancelled":
		return "CANCELLED", executeJobResponse{Status: "CANCELLED"}, nil
	case "running":
		if existing.LeaseUntil != nil && existing.LeaseUntil.After(now) {
			retryAfter := existing.LeaseUntil.Sub(now)
			maxRetry := s.alreadyRunningRetryAfterMax()
			if maxRetry > 0 && retryAfter > maxRetry {
				retryAfter = maxRetry
			}
			return "ALREADY_RUNNING", executeJobResponse{Status: "ALREADY_RUNNING", RetryAfterMs: retryAfter.Milliseconds()}, nil
		}
	case "failed":
		// allow retry
	}

	updated := db.Model(&task.Execution{}).
		Where("job_id = ? AND (status = ? OR (status = ? AND lease_until <= ?))", req.JobId, "failed", "running", now).
		Updates(map[string]any{
			"lease_owner": leaseOwner,
			"lease_until": now.Add(s.leaseDuration()),
			"attempt":     req.Attempt,
			"status":      "running",
			"updated_at":  now,
		})
	if updated.Error != nil {
		return "FAILED_RETRYABLE", executeJobResponse{}, updated.Error
	}
	if updated.RowsAffected == 0 {
		return s.handleExisting(ctx, req, leaseOwner)
	}
	return "RUNNING", executeJobResponse{}, nil
}

func (s taskExecutionStore) heartbeat(ctx context.Context, jobId string, leaseOwner string, stop <-chan struct{}, cancelled chan<- struct{}, cancel context.CancelFunc, leaseLost chan<- struct{}) {
	db := s.db(ctx)
	interval := s.heartbeatInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().UTC()
			res := db.Model(&task.Execution{}).
				Where("job_id = ? AND lease_owner = ? AND status = ?", jobId, leaseOwner, "running").
				Updates(map[string]any{"lease_until": now.Add(s.leaseDuration()), "updated_at": now})
			if res.Error != nil || res.RowsAffected == 0 {
				if leaseLost != nil {
					select {
					case leaseLost <- struct{}{}:
					default:
					}
				}
				if cancel != nil {
					cancel()
				}
				return
			}
			if cancelled != nil {
				if isCancelled, ok := s.isCancelRequestedViaTask(ctx, jobId); ok && isCancelled {
					select {
					case cancelled <- struct{}{}:
					default:
					}
					if cancel != nil {
						cancel()
					}
					return
				}
			}
		}
	}
}

func (s taskExecutionStore) heartbeatInterval() time.Duration {
	ms := s.resolvedRuntimeOptions().taskWorkerHeartbeatIntervalMs
	if ms <= 0 {
		return 20 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}

func (s taskExecutionStore) cancelWatchInterval() time.Duration {
	ms := s.resolvedRuntimeOptions().taskWorkerCancelPollIntervalMs
	if ms <= 0 {
		return 2 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}

func (s taskExecutionStore) leaseDuration() time.Duration {
	ms := s.resolvedRuntimeOptions().taskWorkerLeaseDurationMs
	if ms <= 0 {
		return 60 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}

func (s taskExecutionStore) alreadyRunningRetryAfterMax() time.Duration {
	ms := s.resolvedRuntimeOptions().taskWorkerAlreadyRunningRetryAfterMaxMs
	if ms <= 0 {
		return 60 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}

func (s taskExecutionStore) watchCancel(ctx context.Context, jobId string, cancelled chan<- struct{}, cancel context.CancelFunc, checker func(context.Context, string) (bool, bool)) {
	interval := s.cancelWatchInterval()
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if checker == nil {
				return
			}
			if isCancelled, ok := checker(ctx, jobId); ok && isCancelled {
				select {
				case cancelled <- struct{}{}:
				default:
				}
				if cancel != nil {
					cancel()
				}
				return
			}
		}
	}
}

func (s taskExecutionStore) isCancelRequestedViaTask(ctx context.Context, jobId string) (bool, bool) {
	dialer, ok := grpcclient.ServiceDialerFromContext(ctx)
	if !ok {
		return false, false
	}
	md, err := serviceCodec.methodDescriptor("task.Job.GetJob")
	if err != nil {
		return false, false
	}

	reqMsg := serviceCodec.newMessage(md.Input())
	request := map[string]any{
		"job_id": jobId,
		"fields": []string{"CancelRequestedAt", "Status"},
	}
	if err := serviceCodec.mapToMessage(request, reqMsg); err != nil {
		return false, false
	}

	// Attach internal auth header if configured and not in production.
	opts := s.resolvedRuntimeOptions()
	key := strings.TrimSpace(opts.authInternalKey)
	if key != "" && !strings.EqualFold(strings.TrimSpace(opts.serverEnv), "production") {
		md := metadata.Pairs(internalKeyHeader, key)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	conn, err := dialer(ctx, "task.Job")
	if err != nil {
		return false, false
	}

	resp := serviceCodec.newMessage(md.Output())
	if err := conn.Invoke(ctx, "/task.Job/GetJob", reqMsg, resp); err != nil {
		return false, false
	}

	out, err := serviceCodec.messageToMap(resp)
	if err != nil {
		return false, false
	}
	jobMap := out
	if inner, ok := out["job"].(map[string]any); ok {
		jobMap = inner
	}
	if v, ok := jobMap["cancel_requested_at"]; ok {
		if hasTimestampValue(v) {
			return true, true
		}
	}
	if v, ok := jobMap["cancelRequestedAt"]; ok {
		if hasTimestampValue(v) {
			return true, true
		}
	}
	if statusVal, ok := jobMap["status"]; ok {
		if statusStr := strings.TrimSpace(fmt.Sprintf("%v", statusVal)); strings.EqualFold(statusStr, "cancelled") {
			return true, true
		}
	}
	return false, true
}

func hasTimestampValue(v any) bool {
	switch tv := v.(type) {
	case string:
		return strings.TrimSpace(tv) != ""
	case map[string]any:
		if sec, ok := tv["seconds"]; ok {
			if n, ok := sec.(float64); ok && n > 0 {
				return true
			}
			if n, ok := sec.(int64); ok && n > 0 {
				return true
			}
		}
	}
	return false
}

func (s taskExecutionStore) finalizeSuccess(ctx context.Context, jobId string, leaseOwner string, result any) {
	db := s.db(ctx)
	san, _ := task.SanitizeResultWithMaxBytes(s.resolvedRuntimeOptions().taskSanitizeResultMaxBytes, result)
	resultJSON, _ := json.Marshal(san.Value)
	now := time.Now().UTC()
	_ = db.Model(&task.Execution{}).
		Where("job_id = ? AND lease_owner = ?", jobId, leaseOwner).
		Updates(map[string]any{
			"status":           "succeeded",
			"finished_at":      now,
			"result_json":      datatypes.JSON(resultJSON),
			"result_hash":      san.Hash,
			"result_truncated": san.Truncated,
			"updated_at":       now,
			"lease_until":      nil,
		}).Error
}

func (s taskExecutionStore) finalizeFailure(ctx context.Context, jobId string, leaseOwner string, errObj any) {
	db := s.db(ctx)
	errSan, _ := task.SanitizeErrorWithMaxBytes(s.resolvedRuntimeOptions().taskSanitizeErrorMaxBytes, errObj)
	errJSON, _ := json.Marshal(errSan.Value)
	now := time.Now().UTC()
	_ = db.Model(&task.Execution{}).
		Where("job_id = ? AND lease_owner = ?", jobId, leaseOwner).
		Updates(map[string]any{
			"status":          "failed",
			"finished_at":     now,
			"error_json":      datatypes.JSON(errJSON),
			"error_hash":      errSan.Hash,
			"error_truncated": errSan.Truncated,
			"updated_at":      now,
			"lease_until":     nil,
		}).Error
}

func (s taskExecutionStore) finalizeCancelled(ctx context.Context, jobId string, leaseOwner string) {
	db := s.db(ctx)
	now := time.Now().UTC()
	_ = db.Model(&task.Execution{}).
		Where("job_id = ? AND lease_owner = ?", jobId, leaseOwner).
		Updates(map[string]any{
			"status":       "cancelled",
			"cancelled_at": now,
			"finished_at":  now,
			"updated_at":   now,
			"lease_until":  nil,
		}).Error
}

func jsonToAny(raw datatypes.JSON) any {
	if len(raw) == 0 {
		return nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func errorsIsDuplicate(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}

func leaseOwner(jobId string) string {
	_ = jobId
	return xid.New().String()
}

func (s *ApplicationService) emitTaskMetric(name string, fields map[string]any) {
	if s == nil || s.runtimeScope == nil {
		return
	}
	fields["metric"] = name
	task.RecordMetric(name, fields)
}
