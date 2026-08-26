// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package term_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	exportplan "github.com/choysum-dev/choysum/internal/export/plan"
	"github.com/choysum-dev/choysum/internal/export/reader/term"
	"github.com/choysum-dev/choysum/internal/i18n/terms"
	"github.com/choysum-dev/choysum/pkg/auth"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"google.golang.org/grpc/metadata"
)

type testIdentity struct {
	userID string
	valid  bool
}

func (i testIdentity) GetUserID() string                   { return i.userID }
func (i testIdentity) GetTokenID() string                  { return "" }
func (i testIdentity) GetMetadata() map[string]interface{} { return nil }
func (i testIdentity) IsValid() bool                       { return i.valid }

func TestReaderRequiresAuthentication(t *testing.T) {
	_, err := term.Reader{}.Read(context.Background(), nil, exportplan.Plan{
		Profile:     exportpkg.ProfileTerminology,
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
	})
	if err == nil || !strings.Contains(err.Error(), "authentication is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestReaderRejectsIdentityWithoutTransferableCredential(t *testing.T) {
	ctx := auth.ContextWithIdentity(context.Background(), testIdentity{userID: "user-1", valid: true})
	_, err := term.Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile:     exportpkg.ProfileTerminology,
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
	})
	if err == nil || !strings.Contains(err.Error(), "authentication is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestReaderAcceptsIncomingAuthorizationMetadata(t *testing.T) {
	search := func(_ context.Context, _, _, lang string, modules []string, _ string, _, offset int) (*terms.SearchResult, error) {
		if offset > 0 {
			return &terms.SearchResult{Lang: lang, Items: nil, Total: 1}, nil
		}
		return &terms.SearchResult{
			Lang:  lang,
			Total: 1,
			Items: []terms.Item{{Scope: "a@b", Src: "Hi", Value: "你好", Kind: "literal"}},
		}, nil
	}
	base := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer meta-tok"))
	ctx := terms.ContextWithCollectHooks(base, search, nil)

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
}

func TestReaderRequiresApplicationModuleLang(t *testing.T) {
	ctx := auth.ContextWithAccessToken(context.Background(), "tok")
	_, err := term.Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile:     exportpkg.ProfileTerminology,
		Application: " ",
		Module:      "auth",
		Lang:        "zh_CN",
	})
	if err == nil || !strings.Contains(err.Error(), "application, module, and lang are required") {
		t.Fatalf("err = %v", err)
	}
}

func TestReaderCollectError(t *testing.T) {
	search := func(context.Context, string, string, string, []string, string, int, int) (*terms.SearchResult, error) {
		return nil, errors.New("search boom")
	}
	ctx := auth.ContextWithAccessToken(context.Background(), "tok")
	ctx = terms.ContextWithCollectHooks(ctx, search, nil)
	_, err := term.Reader{}.Read(ctx, nil, exportplan.Plan{
		Profile:     exportpkg.ProfileTerminology,
		Application: "auth",
		Module:      "auth",
		Lang:        "zh_CN",
	})
	if err == nil || !strings.Contains(err.Error(), "terminology search failed") {
		t.Fatalf("err = %v", err)
	}
}

func TestReaderTruncatedWarning(t *testing.T) {
	old := terms.ExportMaxItems
	terms.ExportMaxItems = 1
	t.Cleanup(func() { terms.ExportMaxItems = old })

	search := func(_ context.Context, _, _, lang string, _ []string, _ string, limit, offset int) (*terms.SearchResult, error) {
		if offset > 0 {
			return &terms.SearchResult{Lang: lang, Total: 2}, nil
		}
		items := []terms.Item{{Scope: "a@1", Src: "One", Value: "1"}}
		if limit > 1 {
			items = append(items, terms.Item{Scope: "a@2", Src: "Two", Value: "2"})
		}
		return &terms.SearchResult{Lang: lang, Total: 2, Items: items}, nil
	}
	ctx := auth.ContextWithAccessToken(context.Background(), "tok")
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
	if !result.Truncated || result.Outcomes.Warning != 1 {
		t.Fatalf("result = %+v", result)
	}
}
