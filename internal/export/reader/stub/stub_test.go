// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package stub_test

import (
	"context"
	"errors"
	"testing"

	"github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/reader/stub"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestReader_Read_ok(t *testing.T) {
	result, err := stub.Reader{}.Read(context.Background(), nil, plan.Plan{StubUnitCount: 3})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if result.UnitCount != 3 || len(result.Messages) != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestReader_Read_negativeCount(t *testing.T) {
	result, err := stub.Reader{}.Read(context.Background(), nil, plan.Plan{StubUnitCount: -1})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if result.UnitCount != 0 {
		t.Fatalf("UnitCount = %d, want 0", result.UnitCount)
	}
}

func TestReader_Read_failUnit(t *testing.T) {
	result, err := stub.Reader{}.Read(context.Background(), nil, plan.Plan{
		StubUnitCount:     2,
		StubFailUnitIndex: 2,
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	var expErr *exportpkg.Error
	if !errors.As(err, &expErr) {
		t.Fatalf("error = %v, want *exportpkg.Error", err)
	}
	if len(result.Messages) != 1 || result.Messages[0].Row != 2 {
		t.Fatalf("messages = %+v", result.Messages)
	}
}

func TestReader_Read_failBeyondCount(t *testing.T) {
	result, err := stub.Reader{}.Read(context.Background(), nil, plan.Plan{
		StubUnitCount:     1,
		StubFailUnitIndex: 5,
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	if result.Messages[0].Row != 1 {
		t.Fatalf("fail row = %d, want clamped to count", result.Messages[0].Row)
	}
}

func TestReader_Read_failWithZeroCount(t *testing.T) {
	result, err := stub.Reader{}.Read(context.Background(), nil, plan.Plan{
		StubUnitCount:     0,
		StubFailUnitIndex: 1,
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	if result.Messages[0].Row != 1 {
		t.Fatalf("fail row = %d, want 1 when count is zero", result.Messages[0].Row)
	}
}
