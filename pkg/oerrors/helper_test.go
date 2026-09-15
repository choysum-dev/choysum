// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package oerrors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGetErrorInfoAttrsAndFormatBrief(t *testing.T) {
	choysumErr := New("billing", "E100", "payment failed").WithMetadata("tenant", "acme").WithMetadata("stack", "hidden")
	info := GetErrorInfo(choysumErr)
	if info == nil || info.Domain != "billing" || info.Code != "E100" {
		t.Fatalf("unexpected error info: %#v", info)
	}

	attrs := GetErrorAttrs(choysumErr)
	if len(attrs) == 0 {
		t.Fatal("expected attrs for ChoysumError")
	}
	attrsStr := fmt.Sprint(attrs)
	if !strings.Contains(attrsStr, "tenant") || strings.Contains(attrsStr, "stack") {
		t.Fatalf("unexpected attrs: %v", attrs)
	}

	brief := FormatErrorBrief(choysumErr)
	if !strings.Contains(brief, "[billing] E100: payment failed") || !strings.Contains(brief, "id=") {
		t.Fatalf("unexpected brief: %q", brief)
	}
	if FormatErrorBrief(errors.New("plain")) != "plain" {
		t.Fatal("expected plain error brief to return original message")
	}
	if FormatErrorBrief(nil) != "<nil>" {
		t.Fatal("expected nil brief to be <nil>")
	}
	if GetErrorInfo(errors.New("plain")) != nil || GetErrorAttrs(errors.New("plain")) != nil || GetErrorInfo(nil) != nil {
		t.Fatal("expected non-Choysum errors to have no extracted info")
	}
}

func TestHelperNegatives(t *testing.T) {
	choysumErr := New("billing", "E100", "payment failed")
	if !Is(choysumErr, "billing", "") {
		t.Fatal("expected empty code to match on direct ChoysumError")
	}
	if Is(choysumErr, "orders", "") {
		t.Fatal("expected domain mismatch to return false")
	}
	if Is(choysumErr, "billing", "OTHER") {
		t.Fatal("expected code mismatch to return false")
	}
	if As(errors.New("plain")) != nil {
		t.Fatal("expected As to reject plain errors")
	}
	if !Has(fmt.Errorf("wrap: %w", choysumErr), choysumErr) {
		t.Fatal("expected Has to find wrapped ChoysumError")
	}
}

func TestFromInfo(t *testing.T) {
	cause := errors.New("root")
	src := &ErrorInfo{
		Domain:   "web",
		Code:     "EJS",
		Message:  "boom",
		Metadata: map[string]string{"k": "v"},
	}
	ce := FromInfo(src, cause)
	if ce == nil {
		t.Fatal("expected FromInfo to return ChoysumError")
	}
	if ce.ErrorId == "" {
		t.Fatal("expected missing ErrorId to be filled")
	}
	if src.ErrorId != "" {
		t.Fatal("expected FromInfo not to mutate caller's ErrorId")
	}
	if ce.Metadata["k"] != "v" {
		t.Fatalf("expected metadata copied, got %#v", ce.Metadata)
	}
	ce.Metadata["k"] = "mutated"
	if src.Metadata["k"] != "v" {
		t.Fatal("expected FromInfo to clone Metadata")
	}
	if !errors.Is(ce, cause) {
		t.Fatal("expected cause to be unwrap-able")
	}
	if FromInfo(nil, nil) != nil {
		t.Fatal("expected nil info to return nil")
	}
	if FromInfo(&ErrorInfo{ErrorId: "x"}, nil) != nil {
		t.Fatal("expected info without domain/code to return nil")
	}
	withID := FromInfo(&ErrorInfo{ErrorId: "keep", Domain: "web", Code: "E1"}, nil)
	if withID.ErrorId != "keep" {
		t.Fatalf("expected existing ErrorId preserved, got %q", withID.ErrorId)
	}
	if noMeta := FromInfo(&ErrorInfo{Domain: "web", Code: "E1"}, nil); noMeta.Metadata == nil {
		t.Fatal("expected nil Metadata to become an empty map")
	}
}
