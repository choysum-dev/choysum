// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package artifact_test

import (
	"context"
	"testing"

	exportartifact "github.com/choysum-dev/choysum/internal/export/artifact"
	"github.com/choysum-dev/choysum/pkg/auth"
)

type testIdentity struct {
	valid    bool
	metadata map[string]any
}

func (i testIdentity) GetUserID() string                   { return "user-1" }
func (i testIdentity) GetTokenID() string                  { return "token-1" }
func (i testIdentity) GetMetadata() map[string]interface{} { return i.metadata }
func (i testIdentity) IsValid() bool                       { return i.valid }

func TestResolveArtifactCompanyID_unauthenticatedUsesRequested(t *testing.T) {
	got := exportartifact.ResolveArtifactCompanyID(context.Background(), "co-1")
	if got != "co-1" {
		t.Fatalf("company id = %q", got)
	}
}

func TestResolveArtifactCompanyID_authenticatedUsesActiveCompany(t *testing.T) {
	ctx := auth.ContextWithIdentity(context.Background(), testIdentity{
		valid: true,
		metadata: map[string]any{
			"activeCompanyId": "co-active",
		},
	})
	got := exportartifact.ResolveArtifactCompanyID(ctx, "co-other")
	if got != "co-active" {
		t.Fatalf("company id = %q, want co-active", got)
	}
}

func TestResolveArtifactCompanyID_authenticatedWithoutCompanySkips(t *testing.T) {
	ctx := auth.ContextWithIdentity(context.Background(), testIdentity{valid: true})
	got := exportartifact.ResolveArtifactCompanyID(ctx, "co-other")
	if got != "" {
		t.Fatalf("company id = %q, want empty", got)
	}
}

func TestResolveArtifactCompanyID_invalidIdentityUsesRequested(t *testing.T) {
	ctx := auth.ContextWithIdentity(context.Background(), testIdentity{valid: false})
	got := exportartifact.ResolveArtifactCompanyID(ctx, " co-1 ")
	if got != "co-1" {
		t.Fatalf("company id = %q, want co-1", got)
	}
}
