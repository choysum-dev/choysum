// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package accesslog

import (
	"context"
	"strings"

	"github.com/choysum-dev/choysum/pkg/auth"
)

type identityCaptureKey struct{}

type identityCapture struct {
	userID string
}

// ContextWithIdentityCapture attaches a mutable slot so auth middleware can
// report the authenticated user to an outer access-log interceptor.
func ContextWithIdentityCapture(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if identityCaptureFrom(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, identityCaptureKey{}, &identityCapture{})
}

// ObserveIdentity records the user id for a later access-log line.
func ObserveIdentity(ctx context.Context, identity auth.Identity) {
	cap := identityCaptureFrom(ctx)
	if cap == nil || identity == nil || !identity.IsValid() {
		return
	}
	cap.userID = strings.TrimSpace(identity.GetUserID())
}

func observedUserID(ctx context.Context) string {
	if cap := identityCaptureFrom(ctx); cap != nil {
		return cap.userID
	}
	if id := auth.IdentityFromContext(ctx); id != nil && id.IsValid() {
		return strings.TrimSpace(id.GetUserID())
	}
	return ""
}

func identityCaptureFrom(ctx context.Context) *identityCapture {
	if ctx == nil {
		return nil
	}
	cap, _ := ctx.Value(identityCaptureKey{}).(*identityCapture)
	return cap
}
