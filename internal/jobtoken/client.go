// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jobtoken

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/grpc/client"
	"github.com/choysum-dev/choysum/pkg/grpc/converter"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/dynamicpb"
)

// IssueRequest describes the payload used to request a job token.
type IssueRequest struct {
	JobId             string
	TargetApp         string
	FullMethod        string
	SchedulerUserId   string
	TriggeredByUserId string
	Attempt           int64
	TTL               time.Duration
}

// IssueResponse contains the issued access token and its expiration time.
type IssueResponse struct {
	AccessToken string
	ExpiresAt   int64
}

// IssueTaskJobToken requests a task job token from the internal job token service.
func IssueTaskJobToken(ctx context.Context, authCfg *config.AuthConfig, serverEnvironment string, dialer client.ServiceDialer, req IssueRequest) (*IssueResponse, error) {
	reqDesc, err := ReqDesc()
	if err != nil {
		return nil, err
	}
	respDesc, err := RespDesc()
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"job_id":               req.JobId,
		"target_app":           req.TargetApp,
		"full_method":          req.FullMethod,
		"scheduler_user_id":    req.SchedulerUserId,
		"triggered_by_user_id": req.TriggeredByUserId,
		"attempt":              req.Attempt,
	}
	if req.TTL > 0 {
		payload["ttl_ms"] = int64(req.TTL / time.Millisecond)
	}

	msg := dynamicpb.NewMessage(reqDesc)
	if err := converter.MapToMessage(payload, msg); err != nil {
		return nil, err
	}

	// Attach internal auth header if configured and not in production.
	if authCfg != nil {
		key := strings.TrimSpace(authCfg.InternalKey)
		if key != "" && !strings.EqualFold(strings.TrimSpace(serverEnvironment), "production") {
			md := metadata.Pairs(internalKeyHeader, key)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}
	}

	conn, err := dialer(ctx, ServiceFullName())
	if err != nil {
		return nil, err
	}

	resp := dynamicpb.NewMessage(respDesc)
	if err := conn.Invoke(ctx, FullMethod(), msg, resp); err != nil {
		return nil, err
	}

	outMap, err := converter.MessageToMap(resp)
	if err != nil {
		return nil, err
	}

	accessToken := ""
	if v, ok := outMap["access_token"]; ok {
		if s, ok := v.(string); ok {
			accessToken = strings.TrimSpace(s)
		} else {
			accessToken = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	expiresAt := int64(0)
	if v, ok := outMap["expires_at"]; ok {
		switch tv := v.(type) {
		case int64:
			expiresAt = tv
		case float64:
			expiresAt = int64(tv)
		case int:
			expiresAt = int64(tv)
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(tv), 10, 64); err == nil {
				expiresAt = parsed
			}
		}
	}

	return &IssueResponse{AccessToken: accessToken, ExpiresAt: expiresAt}, nil
}
