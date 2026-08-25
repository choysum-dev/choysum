// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package bridge

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/internal/import/adapter/csv"
	"github.com/choysum-dev/choysum/pkg/auth"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type attachLoaderIdentity struct {
	valid  bool
	userID string
}

func (i attachLoaderIdentity) GetUserID() string                   { return i.userID }
func (i attachLoaderIdentity) GetTokenID() string                  { return "tok" }
func (i attachLoaderIdentity) GetMetadata() map[string]interface{} { return nil }
func (i attachLoaderIdentity) IsValid() bool                       { return i.valid }

func TestAttachImportSourceLoader(t *testing.T) {
	ctx := context.Background()
	spec := importpkg.Spec{Source: importpkg.Source{DocumentRef: "doc-1"}}

	if got := attachImportSourceLoader(ctx, nil, spec); got != ctx {
		t.Fatal("nil scope should preserve ctx")
	}
	if got := attachImportSourceLoader(ctx, &noPathsScope{}, importpkg.Spec{}); got != ctx {
		t.Fatal("empty document ref should preserve ctx")
	}

	existing := csv.ContextWithSourceBytes(ctx, func(context.Context, string) ([]byte, error) {
		return []byte("existing"), nil
	})
	got := attachImportSourceLoader(existing, &noPathsScope{}, spec)
	if !csv.HasSourceBytesLoader(got) {
		t.Fatal("expected existing loader to remain")
	}
	loader, ok := csv.SourceBytesLoaderFromContext(got)
	if !ok {
		t.Fatal("expected loader")
	}
	raw, err := loader(got, "doc-1")
	if err != nil || string(raw) != "existing" {
		t.Fatalf("loader replaced unexpectedly: %q err=%v", raw, err)
	}

	attached := attachImportSourceLoader(ctx, &noPathsScope{}, spec)
	if !csv.HasSourceBytesLoader(attached) {
		t.Fatal("expected loader attached")
	}
	loader, ok = csv.SourceBytesLoaderFromContext(attached)
	if !ok {
		t.Fatal("expected attached loader")
	}
	if _, err := loader(context.Background(), "doc-1"); err == nil {
		t.Fatal("expected auth required")
	}
	authed := auth.ContextWithIdentity(context.Background(), attachLoaderIdentity{valid: false, userID: "u"})
	if _, err := loader(authed, "doc-1"); err == nil {
		t.Fatal("expected invalid identity error")
	}

	orig := readImportSourceBytes
	t.Cleanup(func() { readImportSourceBytes = orig })
	readImportSourceBytes = func(context.Context, scope.Scope, string, auth.Identity) ([]byte, error) {
		return []byte("from-gateway"), nil
	}
	validCtx := auth.ContextWithIdentity(context.Background(), attachLoaderIdentity{valid: true, userID: "u"})
	loaderAttached := attachImportSourceLoader(context.Background(), &noPathsScope{}, spec)
	loader, ok = csv.SourceBytesLoaderFromContext(loaderAttached)
	if !ok {
		t.Fatal("expected loader")
	}
	raw, err = loader(validCtx, "doc-1")
	if err != nil || string(raw) != "from-gateway" {
		t.Fatalf("gateway load = %q err=%v", raw, err)
	}
}
