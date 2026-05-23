// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatusMapping(t *testing.T) {
	if ToStatusError(nil) != nil {
		t.Fatal("expected nil status error for nil input")
	}

	already := status.Error(codes.Canceled, "stop")
	if got := ToStatusError(already); got != already {
		t.Fatal("expected existing status error to be returned as-is")
	}

	assertCode := func(err error, code codes.Code) {
		t.Helper()
		if status.Code(ToStatusError(err)) != code {
			t.Fatalf("status code for %T = %v, want %v", err, status.Code(ToStatusError(err)), code)
		}
	}
	assertCode(&InvalidServiceNameError{ServiceName: "bad"}, codes.InvalidArgument)
	assertCode(&MissingServiceDialerError{}, codes.FailedPrecondition)
	assertCode(&ConnCacheFullError{ServiceName: "svc", Max: 1, Current: 2}, codes.ResourceExhausted)
	assertCode(errors.New("plain"), codes.Unknown)
}
