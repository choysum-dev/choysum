// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term_test

import (
	"context"
	"testing"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/reader/term"
	"github.com/choysum-dev/choysum/internal/i18n/terms"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"google.golang.org/grpc/metadata"
)

func TestReaderAcceptsCLIInternalKey(t *testing.T) {
	search := func(_ context.Context, _, _, lang string, _ []string, _ string, _, offset int) (*terms.SearchResult, error) {
		if offset > 0 {
			return &terms.SearchResult{Lang: lang, Items: nil, Total: 1}, nil
		}
		return &terms.SearchResult{
			Lang:  lang,
			Total: 1,
			Items: []terms.Item{{Scope: "a@b", Src: "Hi", Value: "你好", Kind: "literal"}},
		}, nil
	}
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(terms.InternalKeyHeader, "cli-test-key"))
	ctx = terms.ContextWithCollectHooks(ctx, search, nil)

	result, err := term.Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile:     exportpkg.ProfileTerminology,
		Caller:      exportpkg.CallerCLI,
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if result.Outcomes.Total != 1 {
		t.Fatalf("total = %d", result.Outcomes.Total)
	}
}
