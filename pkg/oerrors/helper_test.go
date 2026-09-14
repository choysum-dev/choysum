// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package oerrors

import (
	"encoding/json"
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

type foreignJSONError struct {
	Message    string
	JSONString string
}

func (e *foreignJSONError) Error() string { return e.Message }

func TestGetErrorInfoFromForeignExtractorAndHelperNegatives(t *testing.T) {
	prev := foreignErrorInfoExtractor.Load()
	t.Cleanup(func() { foreignErrorInfoExtractor.Store(prev) })

	SetForeignErrorInfoExtractor(func(err error) *ErrorInfo {
		var fe *foreignJSONError
		if !errors.As(err, &fe) {
			return nil
		}
		var payload struct {
			ErrorId  string            `json:"errorId"`
			Domain   string            `json:"domain"`
			Code     string            `json:"code"`
			GrpcCode int32             `json:"grpcCode"`
			Metadata map[string]string `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(fe.JSONString), &payload); err != nil {
			return nil
		}
		return &ErrorInfo{
			ErrorId:  payload.ErrorId,
			Domain:   payload.Domain,
			Code:     payload.Code,
			Message:  fe.Message,
			GrpcCode: payload.GrpcCode,
			Metadata: payload.Metadata,
		}
	})

	t.Run("extracts foreign error info from JSON", func(t *testing.T) {
		fe := &foreignJSONError{
			Message:    "js exploded",
			JSONString: `{"errorId":"err-1","domain":"web","code":"EJS","grpcCode":7,"metadata":{"tenant":"acme"}}`,
		}

		info := GetErrorInfo(fe)
		if info == nil {
			t.Fatal("expected foreign error info to be extracted")
		}
		if info.ErrorId != "err-1" || info.Domain != "web" || info.Code != "EJS" || info.Message != "js exploded" || info.GrpcCode != 7 {
			t.Fatalf("unexpected foreign error info: %#v", info)
		}
		if info.Metadata["tenant"] != "acme" {
			t.Fatalf("expected foreign metadata to be preserved, got %#v", info.Metadata)
		}
	})

	t.Run("invalid foreign JSON returns nil", func(t *testing.T) {
		fe := &foreignJSONError{Message: "bad json", JSONString: "{"}
		if info := GetErrorInfo(fe); info != nil {
			t.Fatalf("expected invalid foreign JSON to return nil, got %#v", info)
		}
	})

	t.Run("Is and As reject non matching inputs", func(t *testing.T) {
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
	})
}

func TestSetForeignErrorInfoExtractorClear(t *testing.T) {
	prev := foreignErrorInfoExtractor.Load()
	t.Cleanup(func() { foreignErrorInfoExtractor.Store(prev) })

	SetForeignErrorInfoExtractor(func(error) *ErrorInfo {
		return &ErrorInfo{Domain: "x", Code: "y", Message: "z"}
	})
	if GetErrorInfo(errors.New("plain")) == nil {
		t.Fatal("expected extractor to map plain error")
	}
	SetForeignErrorInfoExtractor(nil)
	if GetErrorInfo(errors.New("plain")) != nil {
		t.Fatal("expected cleared extractor to leave plain errors unrecognized")
	}
}
