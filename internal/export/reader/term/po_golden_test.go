// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term_test

import (
	"context"
	"strings"
	"testing"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/reader/term"
	"github.com/choysum-dev/choysum/internal/i18n/po"
	"github.com/choysum-dev/choysum/internal/i18n/terms"
	"github.com/choysum-dev/choysum/pkg/auth"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
)

func TestTermExport_POGolden(t *testing.T) {
	search := func(_ context.Context, _, _, lang string, modules []string, _ string, limit, offset int) (*terms.SearchResult, error) {
		if offset > 0 {
			return &terms.SearchResult{Lang: lang, Items: nil, Total: 1}, nil
		}
		return &terms.SearchResult{
			Lang:  lang,
			Total: 1,
			Items: []terms.Item{{
				Application: "auth",
				Module:      modules[0],
				Scope:       "web/pages/Login@title",
				Src:         "Sign in",
				Value:       "登录",
				Kind:        "literal",
				Source:      "po",
				Status:      "translated",
			}},
		}, nil
	}
	ctx := auth.ContextWithAccessToken(context.Background(), "test-token")
	ctx = terms.ContextWithCollectHooks(ctx, search, nil)

	result, err := term.Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile:     exportpkg.ProfileTerminology,
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

	var buf strings.Builder
	if err := po.Write(&buf, result.POEntries); err != nil {
		t.Fatalf("Write: %v", err)
	}
	body := buf.String()
	if !strings.Contains(body, `msgctxt "web/pages/Login@title"`) {
		t.Fatalf("missing msgctxt: %s", body)
	}
	if !strings.Contains(body, `msgid "Sign in"`) || !strings.Contains(body, `msgstr "登录"`) {
		t.Fatalf("missing msgid/msgstr: %s", body)
	}
}
