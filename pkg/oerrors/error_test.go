// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package oerrors

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	xerrors "golang.org/x/exp/errors"
	"google.golang.org/grpc/codes"
)

type testErrorPrinter struct {
	detail bool
	parts  []string
}

func (p *testErrorPrinter) Print(args ...interface{}) {
	p.parts = append(p.parts, fmt.Sprint(args...))
}

func (p *testErrorPrinter) Printf(format string, args ...interface{}) {
	p.parts = append(p.parts, fmt.Sprintf(format, args...))
}

func (p *testErrorPrinter) Detail() bool {
	return p.detail
}

var _ xerrors.Printer = (*testErrorPrinter)(nil)

func TestChoysumErrorErrorAndFluentHelpers(t *testing.T) {
	err := New("auth", "E001", "bad token").WithMetadata("service", "auth").WithMetadata("stack", "hidden").WithGrpcCode(codes.Unauthenticated)
	if !strings.Contains(err.Error(), "[auth] E001: bad token") {
		t.Fatalf("unexpected Error() output: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "service=auth") || strings.Contains(err.Error(), "stack=hidden") {
		t.Fatalf("unexpected metadata rendering: %q", err.Error())
	}
	if err.Unwrap() != nil {
		t.Fatalf("expected base error unwrap to be nil, got %v", err.Unwrap())
	}
}

func TestChoysumErrorFormatting(t *testing.T) {
	choysumErr := New("orders", "E777", "broken").WithGrpcCode(codes.InvalidArgument)
	st, err := choysumErr.ToGrpcStatus()
	if err != nil {
		t.Fatalf("ToGrpcStatus error: %v", err)
	}
	if st.Code() != codes.InvalidArgument || st.Message() != "broken" {
		t.Fatalf("unexpected grpc status: code=%v msg=%q", st.Code(), st.Message())
	}
	formattedBase := fmt.Sprintf("%v", choysumErr)
	if !strings.Contains(formattedBase, "[orders] E777: broken") {
		t.Fatalf("unexpected base formatted error: %q", formattedBase)
	}

	wrapped := Wrap(choysumErr, "orders", "E778", "stream failed")
	formatted := fmt.Sprintf("%v", wrapped)
	if !strings.Contains(formatted, "stream failed") || !strings.Contains(formatted, "[orders] E777: broken") {
		t.Fatalf("unexpected formatted wrapped error: %q", formatted)
	}
	detailed := fmt.Sprintf("%+v", wrapped)
	if !strings.Contains(detailed, "stream failed") || !strings.Contains(detailed, "[orders] E777: broken") {
		t.Fatalf("unexpected detailed formatting: %q", detailed)
	}
}

func TestChoysumErrorAdditionalBranches(t *testing.T) {
	t.Run("WithMetadata initializes nil map", func(t *testing.T) {
		err := &ChoysumError{ErrorInfo: &ErrorInfo{Domain: "auth", Code: "E002", Message: "missing metadata"}}
		err.WithMetadata("tenant", "acme")
		if err.Metadata["tenant"] != "acme" {
			t.Fatalf("expected metadata map to be initialized, got %#v", err.Metadata)
		}
	})

	t.Run("ToGrpcStatus defaults grpc code to internal", func(t *testing.T) {
		err := &ChoysumError{ErrorInfo: &ErrorInfo{Domain: "orders", Code: "E999", Message: "fallback", GrpcCode: 0, Metadata: map[string]string{}}}
		st, statusErr := err.ToGrpcStatus()
		if statusErr != nil {
			t.Fatalf("ToGrpcStatus() error = %v", statusErr)
		}
		if st.Code() != codes.Internal || st.Message() != "fallback" {
			t.Fatalf("unexpected grpc status: code=%v message=%q", st.Code(), st.Message())
		}
	})

	t.Run("Format handles wrapped error without metadata", func(t *testing.T) {
		wrapped := &ChoysumError{
			ErrorInfo: &ErrorInfo{Domain: "billing", Code: "E123", Message: "write failed"},
			cause:     errors.New("disk full"),
		}
		formatted := fmt.Sprintf("%v", wrapped)
		if !strings.Contains(formatted, "[billing] E123: write failed") || !strings.Contains(formatted, "disk full") {
			t.Fatalf("unexpected wrapped formatting without metadata: %q", formatted)
		}
	})

	t.Run("FormatError prints base and wrapped variants", func(t *testing.T) {
		base := &ChoysumError{ErrorInfo: &ErrorInfo{Domain: "billing", Code: "E124", Message: "plain"}}
		basePrinter := &testErrorPrinter{}
		if next := base.FormatError(basePrinter); next != nil {
			t.Fatalf("expected base FormatError to terminate chain, got %v", next)
		}
		if got := strings.Join(basePrinter.parts, ""); !strings.Contains(got, "[billing] E124: plain") {
			t.Fatalf("unexpected base FormatError output: %q", got)
		}

		wrapped := Wrap(errors.New("disk full"), "billing", "E125", "write failed").(*ChoysumError)
		wrappedPrinter := &testErrorPrinter{detail: true}
		next := wrapped.FormatError(wrappedPrinter)
		if next == nil || next.Error() != "disk full" {
			t.Fatalf("expected wrapped FormatError to return cause, got %v", next)
		}
		if got := strings.Join(wrappedPrinter.parts, ""); !strings.Contains(got, "[billing] E125: write failed") {
			t.Fatalf("unexpected wrapped FormatError output: %q", got)
		}
	})
}
