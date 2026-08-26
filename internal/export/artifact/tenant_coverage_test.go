// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package artifact

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth"
)

type tenantTestIdentity struct {
	valid    bool
	metadata map[string]any
}

func (i tenantTestIdentity) GetUserID() string                   { return "user-1" }
func (i tenantTestIdentity) GetTokenID() string                  { return "token-1" }
func (i tenantTestIdentity) GetMetadata() map[string]interface{} { return i.metadata }
func (i tenantTestIdentity) IsValid() bool                       { return i.valid }

func TestActiveCompanyID_nilMetadata(t *testing.T) {
	if got := activeCompanyID(tenantTestIdentity{metadata: nil}); got != "" {
		t.Fatalf("activeCompanyID = %q", got)
	}
}

func TestActiveCompanyID_skipsWhitespaceOnlyValues(t *testing.T) {
	identity := tenantTestIdentity{
		metadata: map[string]any{
			"activeCompanyId": "   ",
			"companyId":       "co-fallback",
		},
	}
	if got := activeCompanyID(identity); got != "co-fallback" {
		t.Fatalf("activeCompanyID = %q", got)
	}
}

func TestActiveCompanyID_emptyMetadata(t *testing.T) {
	if got := activeCompanyID(tenantTestIdentity{metadata: map[string]any{}}); got != "" {
		t.Fatalf("activeCompanyID = %q", got)
	}
}

func TestActiveCompanyID_allInvalidTypes(t *testing.T) {
	identity := tenantTestIdentity{
		metadata: map[string]any{
			"activeCompanyId": 1,
			"companyId":       2,
		},
	}
	if got := activeCompanyID(identity); got != "" {
		t.Fatalf("activeCompanyID = %q", got)
	}
}

func TestActiveCompanyID_companyIdFallback(t *testing.T) {
	identity := tenantTestIdentity{
		metadata: map[string]any{"companyId": " co-fallback "},
	}
	if got := activeCompanyID(identity); got != "co-fallback" {
		t.Fatalf("activeCompanyID = %q", got)
	}
}

func TestActiveCompanyID_ignoresInvalidMetadata(t *testing.T) {
	identity := tenantTestIdentity{
		metadata: map[string]any{
			"activeCompanyId": 42,
			"companyId":       "co-ok",
		},
	}
	if got := activeCompanyID(identity); got != "co-ok" {
		t.Fatalf("activeCompanyID = %q", got)
	}
}

func TestResolveArtifactCompanyID_prefersActiveCompanyId(t *testing.T) {
	ctx := auth.ContextWithIdentity(context.Background(), tenantTestIdentity{
		valid: true,
		metadata: map[string]any{
			"activeCompanyId": "co-active",
			"companyId":       "co-fallback",
		},
	})
	if got := ResolveArtifactCompanyID(ctx, "co-requested"); got != "co-active" {
		t.Fatalf("ResolveArtifactCompanyID = %q", got)
	}
}
