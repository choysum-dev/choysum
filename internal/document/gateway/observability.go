// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"context"
	"strings"
	"sync"

	"github.com/choysum-dev/choysum/pkg/scope"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type permissionDeniedObservationInput struct {
	stage      string
	reason     string
	ownerModel string
	fieldName  string
}

var (
	permissionDeniedCounterOnce sync.Once
	permissionDeniedCounter     metric.Int64Counter
)

func observePermissionDenied(ctx context.Context, runtimeScope scope.Scope, input permissionDeniedObservationInput) {
	stage := normalizeObservationLabel(input.stage, "unknown")
	reason := normalizeObservationLabel(input.reason, "unknown")
	ownerModel := strings.TrimSpace(input.ownerModel)
	fieldName := strings.TrimSpace(input.fieldName)

	if runtimeScope != nil && runtimeScope.Logger() != nil {
		runtimeScope.Logger().Warn(
			"document permission denied",
			"stage", stage,
			"owner_model", ownerModel,
			"field_name", fieldName,
			"reason", reason,
		)
	}

	counter := getPermissionDeniedCounter()
	if counter != nil {
		counter.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("stage", stage),
				attribute.String("reason", reason),
			),
		)
	}
}

func getPermissionDeniedCounter() metric.Int64Counter {
	permissionDeniedCounterOnce.Do(func() {
		counter, err := otel.Meter("choysum.document.gateway").Int64Counter("document.permission_denied_total")
		if err == nil {
			permissionDeniedCounter = counter
		}
	})
	return permissionDeniedCounter
}

func normalizeObservationLabel(value string, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = strings.ToLower(strings.TrimSpace(fallback))
	}
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	if normalized == "" {
		return "unknown"
	}
	return normalized
}
