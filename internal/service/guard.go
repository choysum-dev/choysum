// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/jwtauth"
	"github.com/choysum-dev/choysum/pkg/auth"
	authgrpcclient "github.com/choysum-dev/choysum/pkg/auth/grpcclient"
	"github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/oerrors"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/rs/xid"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type serviceGuard struct {
	runtimeScope  scope.Scope
	runtimeOpts   runtimeOptions
	appName       string
	jsExecutor    jsexecutor.ScriptExecutor
	hasGrpcMethod func(fullMethod string) bool
}

func (s *ApplicationService) guard() serviceGuard {
	return serviceGuard{
		runtimeScope:  s.runtimeScope,
		runtimeOpts:   s.resolvedRuntimeOptions(),
		appName:       s.name,
		jsExecutor:    s.jsExecutor,
		hasGrpcMethod: s.hasGrpcMethod,
	}
}

func (g serviceGuard) authorizeUnary(ctx context.Context, runtimeScope scope.Scope, jsCtx map[string]interface{}, fullMethod string) (*jsengine.JsExecutionRouting, error) {
	g.injectMethodMetaAndEntryPolicy(jsCtx, fullMethod)
	return g.runMethodAccess(ctx, runtimeScope, jsCtx, fullMethod, false)
}

func (g serviceGuard) authorizeExecuteJobCaller(ctx context.Context, req *executeJobRequest) (context.Context, bool, error) {
	internalCaller := authorizeInternalCallerWithOptions(ctx, g.runtimeOpts)
	if internalCaller {
		ctx = ensureInternalIdentity(ctx, g.appName, req)
	} else if err := g.validateJobToken(ctx, req); err != nil {
		return ctx, false, err
	}

	if req == nil || !strings.HasPrefix(req.FullMethod, g.appName+".") {
		return ctx, false, status.Error(codes.InvalidArgument, "fullMethod does not match target app")
	}

	return ctx, internalCaller, nil
}

func (g serviceGuard) authorizeExecuteJobMethod(ctx context.Context, runtimeScope scope.Scope, jsCtx map[string]interface{}, req *executeJobRequest, internalCaller bool) (*jsengine.JsExecutionRouting, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "fullMethod does not match target app")
	}

	fullMethod := "/" + req.FullMethod
	g.injectMethodMetaAndEntryPolicy(jsCtx, fullMethod)
	if internalCaller {
		return nil, nil
	}

	return g.runMethodAccess(ctx, runtimeScope, jsCtx, fullMethod, true)
}

func (g serviceGuard) enforceMethodAccess(ctx context.Context, runtimeScope scope.Scope, jsCtx map[string]interface{}, fullMethod string) error {
	_, err := g.runMethodAccess(ctx, runtimeScope, jsCtx, fullMethod, false)
	return err
}

func (g serviceGuard) enforceMethodAccessStrict(ctx context.Context, runtimeScope scope.Scope, jsCtx map[string]interface{}, fullMethod string) error {
	_, err := g.runMethodAccess(ctx, runtimeScope, jsCtx, fullMethod, true)
	return err
}

func (g serviceGuard) runMethodAccess(ctx context.Context, runtimeScope scope.Scope, jsCtx map[string]interface{}, fullMethod string, strict bool) (*jsengine.JsExecutionRouting, error) {
	opts := g.runtimeOpts
	if !hasRuntimeOptions(opts) {
		opts = runtimeOptionsFromScope(g.runtimeScope)
	}
	if !opts.authEnabled || !opts.authGrpcMethodAccess {
		return nil, nil
	}

	emit := func(string, string, map[string]any) {}
	if !strict {
		logMode := strings.TrimSpace(opts.authzDecisionLog)
		logDenied := logMode == "deny" || logMode == "all"
		logAllowed := logMode == "all"
		auditEnabled := opts.authzDecisionAudit

		emit = func(decision string, basis string, extra map[string]any) {
			if decision == "deny" {
				if !logDenied {
					return
				}
			} else if !logAllowed {
				return
			}

			userID := ""
			if id := auth.IdentityFromContext(ctx); id != nil {
				userID = strings.TrimSpace(id.GetUserID())
			}
			activeCompanyID := ""
			enabledCompanyIDs := []string{}
			if cm, ok := jsCtx["ctx"].(map[string]any); ok {
				if v, ok := cm["activeCompanyId"].(string); ok {
					activeCompanyID = strings.TrimSpace(v)
				} else if v, ok := cm["companyId"].(string); ok {
					activeCompanyID = strings.TrimSpace(v)
				}
				if vv, ok := cm["enabledCompanyIds"].([]string); ok {
					enabledCompanyIDs = append([]string{}, vv...)
				} else if vv, ok := cm["enabledCompanyIds"].([]any); ok {
					for _, item := range vv {
						value := strings.TrimSpace(fmt.Sprintf("%v", item))
						if value != "" {
							enabledCompanyIDs = append(enabledCompanyIDs, value)
						}
					}
				}
			}

			payload := map[string]any{
				"event":             "authz.decision_summary",
				"layer":             "method_access",
				"decision":          decision,
				"basis":             basis,
				"reason":            basis,
				"fullMethod":        fullMethod,
				"userId":            userID,
				"activeCompanyId":   activeCompanyID,
				"enabledCompanyIds": enabledCompanyIDs,
			}
			for key, value := range extra {
				payload[key] = value
			}

			g.runtimeScope.Logger().Info("authz decision", "event", payload["event"], "layer", payload["layer"], "decision", payload["decision"], "basis", payload["basis"], "reason", payload["reason"], "fullMethod", payload["fullMethod"], "userId", payload["userId"], "activeCompanyId", payload["activeCompanyId"], "enabledCompanyIds", payload["enabledCompanyIds"], "extra", extra)
			if auditEnabled {
				g.runtimeScope.Logger().Info("authz decision", "audit", true, "event", payload["event"], "layer", payload["layer"], "decision", payload["decision"], "basis", payload["basis"], "reason", payload["reason"], "fullMethod", payload["fullMethod"], "userId", payload["userId"], "activeCompanyId", payload["activeCompanyId"], "enabledCompanyIds", payload["enabledCompanyIds"], "extra", extra)
			}
		}
	}

	entryPolicy := opts.authGrpcEntryPolicy
	excluded := []string{
		"grpc.health.v1.Health/*",
		"Health/*",
		"grpc.reflection.v1alpha.ServerReflection/*",
		"grpc.reflection.v1.ServerReflection/*",
		"grpc.channelz.v1.Channelz/*",
		"auth.User/CheckMethodAccess",
	}
	if !strict && entryPolicy != nil {
		for methodKey, methodCfg := range entryPolicy {
			if methodCfg == nil || !methodCfg.SkipMethodAccess {
				continue
			}
			excluded = append(excluded, strings.TrimSpace(methodKey))
		}
	}

	var reqMeta map[string]any
	if meta, ok := getJsReqMeta(jsCtx); ok {
		reqMeta = meta
	}
	if getJsReqDepth(reqMeta) != 0 {
		return nil, nil
	}

	methodKey := strings.TrimPrefix(fullMethod, "/")
	if !strict && entryPolicy != nil {
		if methodCfg, ok := entryPolicy[methodKey]; ok && methodCfg != nil {
			if methodCfg.SkipAuthentication || methodCfg.SkipMethodAccess {
				return nil, nil
			}
		}
	}

	for _, method := range excluded {
		if method == fullMethod || strings.HasSuffix(fullMethod, "/"+method) {
			return nil, nil
		}
	}

	identity := auth.IdentityFromContext(ctx)
	if identity == nil || strings.TrimSpace(identity.GetUserID()) == "" {
		return nil, status.Error(codes.Unauthenticated, "missing identity")
	}

	companyID := ""
	if cm, ok := jsCtx["ctx"].(map[string]any); ok {
		if v, ok := cm["activeCompanyId"].(string); ok {
			companyID = v
		} else if v, ok := cm["companyId"].(string); ok {
			companyID = v
		}
	}

	allowed, routing, aclErr := g.checkMethodAccess(ctx, runtimeScope, jsCtx, fullMethod, companyID)
	if aclErr != nil {
		emit("deny", "acl_check_failed", map[string]any{"error": aclErr.Error()})
		if g.hasGrpcMethod == nil || !g.hasGrpcMethod("/auth.User/CheckMethodAccess") {
			return nil, client.ToStatusError(aclErr)
		}
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("permission check failed: %v", aclErr))
	}
	if !allowed {
		emit("deny", "acl_denied", map[string]any{})
		return nil, status.Error(codes.PermissionDenied, "access denied")
	}

	emit("allow", "acl_allowed", map[string]any{})
	return routing, nil
}

func (g serviceGuard) checkMethodAccess(ctx context.Context, runtimeScope scope.Scope, jsCtx map[string]interface{}, fullMethod string, companyID string) (bool, *jsengine.JsExecutionRouting, error) {
	if g.hasGrpcMethod != nil && g.hasGrpcMethod("/auth.User/CheckMethodAccess") {
		execCtx := scope.ContextWithScope(ctx, runtimeScope)
		aclReq := &jsengine.JsRequest{
			Id:      fmt.Sprintf("%s:acl", trace.SpanFromContext(ctx).SpanContext().SpanID().String()),
			Service: "auth.User.CheckMethodAccess",
			Context: jsCtx,
			Args:    []any{companyID, fullMethod},
		}
		aclResp, err := g.jsExecutor.Execute(execCtx, aclReq)
		if err != nil {
			return false, nil, err
		}

		allowed, ok := aclResp.Result.(bool)
		if !ok {
			return false, nil, fmt.Errorf("invalid CheckMethodAccess result type: %T", aclResp.Result)
		}
		if aclResp.Routing != nil {
			routing := *aclResp.Routing
			return allowed, &routing, nil
		}
		return allowed, nil, nil
	}

	allowed, err := authgrpcclient.CheckMethodAccess(ctx, companyID, fullMethod)
	return allowed, nil, err
}

func (g serviceGuard) injectMethodMetaAndEntryPolicy(jsCtx map[string]interface{}, fullMethod string) {
	reqMeta, ok := getJsReqMeta(jsCtx)
	if !ok {
		return
	}

	methodKey := strings.TrimPrefix(fullMethod, "/")
	reqMeta["fullMethod"] = fullMethod
	reqMeta["method"] = methodKey

	if getJsReqDepth(reqMeta) != 0 {
		return
	}

	opts := g.runtimeOpts
	if !hasRuntimeOptions(opts) {
		opts = runtimeOptionsFromScope(g.runtimeScope)
	}
	if !opts.authEnabled {
		reqMeta["companyMode"] = "skip"
		reqMeta["recordRuleMode"] = "skip"
		reqMeta["fieldRuleMode"] = "skip"
		return
	}

	companyFilterEnabled := opts.authGrpcCompanyFilter
	recordRuleEnabled := opts.authGrpcRecordRule
	fieldRuleEnabled := opts.authGrpcFieldRule

	if !companyFilterEnabled {
		reqMeta["companyMode"] = "skip"
	}
	if !recordRuleEnabled {
		reqMeta["recordRuleMode"] = "skip"
	}
	if !fieldRuleEnabled {
		reqMeta["fieldRuleMode"] = "skip"
	}

	if opts.authGrpcEntryPolicy == nil {
		return
	}

	methodCfg, ok := opts.authGrpcEntryPolicy[methodKey]
	if !ok || methodCfg == nil {
		return
	}

	if companyFilterEnabled && methodCfg.SkipCompanyFilter {
		reqMeta["companyMode"] = "skip"
	}
	if recordRuleEnabled {
		if methodCfg.SkipRecordRule {
			reqMeta["recordRuleMode"] = "skip"
		} else if len(methodCfg.RecordRuleAllow) > 0 {
			reqMeta["recordRuleMode"] = "allowlist"
			capHint := 0
			for _, item := range methodCfg.RecordRuleAllow {
				model := strings.TrimSpace(item.Model)
				if model == "" {
					continue
				}
				for _, op := range item.Ops {
					if strings.TrimSpace(op) == "" {
						continue
					}
					capHint++
				}
			}
			allow := make([]string, 0, capHint)
			for _, item := range methodCfg.RecordRuleAllow {
				model := strings.TrimSpace(item.Model)
				if model == "" {
					continue
				}
				for _, op := range item.Ops {
					op = strings.TrimSpace(op)
					if op == "" {
						continue
					}
					allow = append(allow, fmt.Sprintf("%s:%s", model, op))
				}
			}
			if len(allow) > 0 {
				reqMeta["recordRuleAllow"] = allow
			}
		}
	}
	if fieldRuleEnabled && methodCfg.SkipFieldRule {
		reqMeta["fieldRuleMode"] = "skip"
	}
}

func (g serviceGuard) validateJobToken(ctx context.Context, req *executeJobRequest) error {
	identity := auth.IdentityFromContext(ctx)
	if identity == nil || !identity.IsValid() {
		return status.Error(codes.Unauthenticated, "missing identity")
	}
	if strings.TrimSpace(identity.GetUserID()) != req.SchedulerUserId {
		return status.Error(codes.Unauthenticated, "identity mismatch")
	}
	meta := identity.GetMetadata()
	if meta == nil {
		return status.Error(codes.Unauthenticated, "missing task token metadata")
	}
	if fmt.Sprintf("%v", meta["purpose"]) != "task_job" {
		return status.Error(codes.Unauthenticated, "task token purpose mismatch")
	}
	if fmt.Sprintf("%v", meta["jobId"]) != req.JobId {
		return status.Error(codes.Unauthenticated, "jobId mismatch")
	}
	if fmt.Sprintf("%v", meta["fullMethod"]) != req.FullMethod {
		return status.Error(codes.Unauthenticated, "fullMethod mismatch")
	}
	if fmt.Sprintf("%v", meta["targetApp"]) != g.appName {
		return status.Error(codes.Unauthenticated, "targetApp mismatch")
	}
	if fmt.Sprintf("%v", meta["schedulerUserId"]) != req.SchedulerUserId {
		return status.Error(codes.Unauthenticated, "schedulerUserId mismatch")
	}
	if fmt.Sprintf("%v", meta["triggeredByUserId"]) != req.TriggeredByUserId {
		return status.Error(codes.Unauthenticated, "triggeredByUserId mismatch")
	}
	if value, ok := meta["attempt"]; ok {
		attempt := 0
		switch typed := value.(type) {
		case int:
			attempt = typed
		case int64:
			attempt = int(typed)
		case float64:
			attempt = int(typed)
		}
		if attempt != req.Attempt {
			return status.Error(codes.Unauthenticated, "attempt mismatch")
		}
	}
	return nil
}

func (g serviceGuard) mapExecutionError(err error) executeJobResponse {
	return mapExecutionError(err)
}

func authorizeInternalCallerFromContext(ctx context.Context, runtimeScope scope.Scope) bool {
	return authorizeInternalCallerWithOptions(ctx, runtimeOptionsFromScope(runtimeScope))
}

func authorizeInternalCallerWithOptions(ctx context.Context, opts runtimeOptions) bool {
	if strings.EqualFold(strings.TrimSpace(opts.serverEnv), "production") {
		return false
	}
	key := strings.TrimSpace(opts.authInternalKey)
	if key == "" {
		return false
	}
	md, _ := metadata.FromIncomingContext(ctx)
	if md == nil {
		return false
	}
	values := md.Get(internalKeyHeader)
	if len(values) == 0 {
		return false
	}
	provided := strings.TrimSpace(values[0])
	return subtle.ConstantTimeCompare([]byte(provided), []byte(key)) == 1
}

func ensureInternalIdentity(ctx context.Context, appName string, req *executeJobRequest) context.Context {
	if req == nil {
		return ctx
	}
	if identity := auth.IdentityFromContext(ctx); identity != nil && identity.IsValid() {
		return ctx
	}
	userID := strings.TrimSpace(req.SchedulerUserId)
	if userID == "" {
		return ctx
	}
	jobID := strings.TrimSpace(req.JobId)
	if jobID == "" {
		jobID = xid.New().String()
	}
	meta := map[string]any{
		"purpose":           "task_job",
		"jobId":             jobID,
		"targetApp":         appName,
		"fullMethod":        req.FullMethod,
		"schedulerUserId":   req.SchedulerUserId,
		"triggeredByUserId": req.TriggeredByUserId,
		"attempt":           req.Attempt,
	}
	expiresAt := time.Now().Add(5 * time.Minute)
	identity := jwtauth.NewIdentity(userID, jobID, meta, expiresAt, time.Now(), auth.AccessToken, "choysum-internal")
	return auth.ContextWithIdentity(ctx, identity)
}

func errToTaskError(err error) map[string]any {
	if err == nil {
		return nil
	}
	st, _ := status.FromError(err)
	info := oerrors.GetErrorInfo(err)
	out := map[string]any{
		"grpc_code": int32(st.Code()),
		"message":   st.Message(),
	}
	if info != nil {
		if info.Domain != "" {
			out["domain"] = info.Domain
		}
		if info.Code != "" {
			out["code"] = info.Code
		}
		if len(info.Metadata) > 0 {
			details := make(map[string]any, len(info.Metadata))
			for key, value := range info.Metadata {
				details[key] = value
			}
			out["details"] = details
		}
	}
	return out
}

func mapExecutionError(err error) executeJobResponse {
	mapped := executeJobResponse{Status: "FAILED_NON_RETRYABLE"}
	st, _ := status.FromError(err)
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled, codes.ResourceExhausted, codes.Aborted, codes.Internal, codes.Unknown:
		mapped.Status = "FAILED_RETRYABLE"
	}
	info := oerrors.GetErrorInfo(err)
	if info != nil && info.Domain == "meta.lock" && info.Code == "LEASE_CONFLICT" {
		mapped.Status = "RESOURCE_BUSY"
		if info.Metadata != nil {
			if value, ok := info.Metadata["retry_after_ms"]; ok {
				if parsed, parseErr := time.ParseDuration(value + "ms"); parseErr == nil {
					mapped.RetryAfterMs = int64(parsed / time.Millisecond)
				}
			}
		}
	}
	mapped.Error = errToTaskError(err)
	return mapped
}
