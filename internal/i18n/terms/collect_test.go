// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package terms

import (
	"context"
	"errors"
	"testing"
)

func TestCollectAllPaginatesUntilTotal(t *testing.T) {
	oldPage := ExportPageSize
	ExportPageSize = 1
	t.Cleanup(func() { ExportPageSize = oldPage })

	calls := 0
	search := func(_ context.Context, _, _, _ string, _ []string, _ string, limit, offset int) (*SearchResult, error) {
		calls++
		switch offset {
		case 0:
			return &SearchResult{
				Total: 2,
				Items: []Item{{Scope: "a@1", Src: "One", Value: "1"}},
			}, nil
		case 1:
			return &SearchResult{
				Total: 2,
				Items: []Item{{Scope: "a@2", Src: "Two", Value: "2"}},
			}, nil
		default:
			return &SearchResult{Total: 2, Items: nil}, nil
		}
	}
	ctx := ContextWithCollectHooks(context.Background(), search, nil)
	items, truncated, err := CollectAll(ctx, "tok", "auth", "zh_CN", []string{"auth"})
	if err != nil || truncated || len(items) != 2 || calls < 2 {
		t.Fatalf("items=%d truncated=%v calls=%d err=%v", len(items), truncated, calls, err)
	}
}

func TestCollectAllSearchHookProbeError(t *testing.T) {
	search := func(context.Context, string, string, string, []string, string, int, int) (*SearchResult, error) {
		return nil, errors.New("probe boom")
	}
	ctx := ContextWithCollectHooks(context.Background(), search, nil)
	if _, _, err := CollectAll(ctx, "tok", "auth", "zh_CN", []string{"auth"}); err == nil {
		t.Fatal("expected probe error")
	}
}

func TestCollectAllStopsOnEmptyPage(t *testing.T) {
	search := func(_ context.Context, _, _, _ string, _ []string, _ string, _, offset int) (*SearchResult, error) {
		if offset > 0 {
			return &SearchResult{Total: 1, Items: nil}, nil
		}
		return &SearchResult{
			Total: 1,
			Items: []Item{{Scope: "a@1", Src: "One", Value: "1"}},
		}, nil
	}
	ctx := ContextWithCollectHooks(context.Background(), search, nil)
	items, truncated, err := CollectAll(ctx, "tok", "auth", "zh_CN", []string{"auth"})
	if err != nil || truncated || len(items) != 1 {
		t.Fatalf("items=%d truncated=%v err=%v", len(items), truncated, err)
	}
}

func TestCollectAllUsesResultTotalWhenCountMissing(t *testing.T) {
	search := func(_ context.Context, _, _, _ string, _ []string, _ string, _, offset int) (*SearchResult, error) {
		if offset > 0 {
			return &SearchResult{Total: 1, Items: nil}, nil
		}
		return &SearchResult{
			Total: 1,
			Items: []Item{{Scope: "a@1", Src: "One", Value: "1"}},
		}, nil
	}
	ctx := ContextWithCollectHooks(context.Background(), search, func(context.Context, string, string, string, []string, string) (int64, error) {
		return 0, nil
	})
	items, truncated, err := CollectAll(ctx, "tok", "auth", "zh_CN", []string{"auth"})
	if err != nil || truncated || len(items) != 1 {
		t.Fatalf("items=%d truncated=%v err=%v", len(items), truncated, err)
	}
}

func TestCollectAllExportMaxItemsZero(t *testing.T) {
	old := ExportMaxItems
	ExportMaxItems = 0
	t.Cleanup(func() { ExportMaxItems = old })

	items, truncated, err := CollectAll(context.Background(), "tok", "auth", "zh_CN", []string{"auth"})
	if err != nil || len(items) != 0 || !truncated {
		t.Fatalf("items=%d truncated=%v err=%v", len(items), truncated, err)
	}
}

func TestCollectAllUsesCountHook(t *testing.T) {
	count := func(context.Context, string, string, string, []string, string) (int64, error) {
		return 0, nil
	}
	search := func(context.Context, string, string, string, []string, string, int, int) (*SearchResult, error) {
		return &SearchResult{Total: 0, Items: nil}, nil
	}
	ctx := ContextWithCollectHooks(context.Background(), search, count)
	items, truncated, err := CollectAll(ctx, "tok", "auth", "zh_CN", []string{"auth"})
	if err != nil || len(items) != 0 || truncated {
		t.Fatalf("items=%d truncated=%v err=%v", len(items), truncated, err)
	}
}

func TestCollectAllCountHookError(t *testing.T) {
	count := func(context.Context, string, string, string, []string, string) (int64, error) {
		return 0, errors.New("count boom")
	}
	ctx := ContextWithCollectHooks(context.Background(), nil, count)
	if _, _, err := CollectAll(ctx, "tok", "auth", "zh_CN", []string{"auth"}); err == nil {
		t.Fatal("expected count hook error")
	}
}

func TestCollectAllSearchHookPagesAndTruncates(t *testing.T) {
	oldMax := ExportMaxItems
	oldPage := ExportPageSize
	ExportMaxItems = 2
	ExportPageSize = 2
	t.Cleanup(func() {
		ExportMaxItems = oldMax
		ExportPageSize = oldPage
	})

	calls := 0
	search := func(_ context.Context, _, _, _ string, _ []string, _ string, limit, offset int) (*SearchResult, error) {
		calls++
		if offset == 0 {
			return &SearchResult{
				Total: 5,
				Items: []Item{
					{Scope: "a@1", Src: "One", Value: "1"},
					{Scope: "a@2", Src: "Two", Value: "2"},
				},
			}, nil
		}
		return &SearchResult{Total: 5, Items: nil}, nil
	}
	ctx := ContextWithCollectHooks(context.Background(), search, nil)
	items, truncated, err := CollectAll(ctx, "tok", "auth", "zh_CN", []string{"auth"})
	if err != nil || !truncated || len(items) != 2 || calls < 1 {
		t.Fatalf("items=%d truncated=%v calls=%d err=%v", len(items), truncated, calls, err)
	}
}

func TestCollectAllDoesNotTruncateWhenLastPageShort(t *testing.T) {
	oldMax := ExportMaxItems
	oldPage := ExportPageSize
	ExportMaxItems = 2
	ExportPageSize = 2
	t.Cleanup(func() {
		ExportMaxItems = oldMax
		ExportPageSize = oldPage
	})

	search := func(_ context.Context, _, _, _ string, _ []string, _ string, limit, offset int) (*SearchResult, error) {
		if offset > 0 {
			return &SearchResult{Total: 0, Items: nil}, nil
		}
		return &SearchResult{
			Total: 0,
			Items: []Item{{Scope: "a@1", Src: "One", Value: "1"}},
		}, nil
	}
	ctx := ContextWithCollectHooks(context.Background(), search, nil)
	items, truncated, err := CollectAll(ctx, "tok", "auth", "zh_CN", []string{"auth"})
	if err != nil || truncated || len(items) != 1 {
		t.Fatalf("items=%d truncated=%v err=%v", len(items), truncated, err)
	}
}

func TestCollectAllSearchHookError(t *testing.T) {
	search := func(context.Context, string, string, string, []string, string, int, int) (*SearchResult, error) {
		return nil, errors.New("search boom")
	}
	ctx := ContextWithCollectHooks(context.Background(), search, func(context.Context, string, string, string, []string, string) (int64, error) {
		return 1, nil
	})
	if _, _, err := CollectAll(ctx, "tok", "auth", "zh_CN", []string{"auth"}); err == nil {
		t.Fatal("expected search hook error")
	}
}
