// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"testing"

	"github.com/choysum-dev/choysum/pkg/auth"
	"github.com/choysum-dev/choysum/pkg/jsengine"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
)

func TestHubTestHelperIdentities(t *testing.T) {
	t.Run("invalidIdentity", func(t *testing.T) {
		var id auth.Identity = invalidIdentity{}
		if id.GetUserID() != "" || id.GetTokenID() != "" || id.GetMetadata() != nil || id.IsValid() {
			t.Fatal("invalidIdentity should be empty and invalid")
		}
	})
	t.Run("nonStringMetaIdentity", func(t *testing.T) {
		id := nonStringMetaIdentity{}
		if id.GetUserID() != "u1" || id.GetTokenID() != "tok" || !id.IsValid() {
			t.Fatal("unexpected nonStringMetaIdentity identity fields")
		}
		if activeCompanyID(auth.ContextWithIdentity(context.Background(), id)) != "" {
			t.Fatal("expected non-string metadata to be ignored")
		}
	})
	t.Run("nilMetaIdentity", func(t *testing.T) {
		id := nilMetaIdentity{}
		if id.GetUserID() != "u1" || id.GetTokenID() != "tok" || id.GetMetadata() != nil || !id.IsValid() {
			t.Fatal("unexpected nilMetaIdentity identity fields")
		}
	})
	t.Run("stubIdentity", func(t *testing.T) {
		id := stubIdentity{}
		if id.GetUserID() != "test-user" || id.GetTokenID() != "test-token" || !id.IsValid() {
			t.Fatal("unexpected stubIdentity fields")
		}
	})
	t.Run("companyMetaIdentity", func(t *testing.T) {
		id := companyMetaIdentity{}
		if id.GetUserID() != "u1" || id.GetTokenID() != "tok" || !id.IsValid() {
			t.Fatal("unexpected companyMetaIdentity fields")
		}
		if id.GetMetadata()["companyId"] != "cmp-fallback" {
			t.Fatal("expected companyId metadata")
		}
	})
	t.Run("spacedActiveCompanyIdentity", func(t *testing.T) {
		id := spacedActiveCompanyIdentity{}
		if id.GetUserID() != "u1" || id.GetTokenID() != "tok" || !id.IsValid() {
			t.Fatal("unexpected spacedActiveCompanyIdentity fields")
		}
		if activeCompanyID(auth.ContextWithIdentity(context.Background(), id)) != "cmp_trim" {
			t.Fatal("expected trimmed active company id")
		}
	})
}

func TestAuthCtxWithServer(t *testing.T) {
	ctx := authCtxWithServer(t, denyAuthServer{})
	if ctx == nil {
		t.Fatal("expected auth context")
	}
}

func TestAuthCtxAllowsExportAccess(t *testing.T) {
	ctx := authCtx(t)
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	if err := checkModelExportAccess(ctx, runtimeScope, "base.Country", ""); err != nil {
		t.Fatalf("checkModelExportAccess: %v", err)
	}
	if got := activeCompanyID(ctx); got != "cmp_test" {
		t.Fatalf("activeCompanyID = %q", got)
	}
	meta := stubIdentity{}.GetMetadata()["activeCompanyId"]
	if meta != "cmp_test" {
		t.Fatalf("metadata = %#v", meta)
	}
}

func TestAuthCtxWithAllowServer(t *testing.T) {
	ctx := authCtxWithServer(t, allowAuthServer{})
	runtimeScope := newHubTestScope(t)
	seedCountryModelMeta(t, runtimeScope.Session().DB)
	if err := checkModelExportAccess(ctx, runtimeScope, "base.Country", ""); err != nil {
		t.Fatalf("checkModelExportAccess: %v", err)
	}
}

func TestSeedHelpersDB(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := seedCountryModelMetaDB(db); err != nil {
		t.Fatalf("seedCountryModelMetaDB: %v", err)
	}
	if err := seedPartnerModelMetaDB(db); err != nil {
		t.Fatalf("seedPartnerModelMetaDB: %v", err)
	}
	if err := seedPartnerModelFieldsDB(db); err != nil {
		t.Fatalf("seedPartnerModelFieldsDB: %v", err)
	}
}

func TestSeedHelperWrappers(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	seedCountryModelMeta(t, db)
	seedPartnerModelMeta(t, db)
	seedPartnerModelFields(t, db)
}

func TestSeedHelperDBErrors(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	if err := seedCountryModelMetaDB(db); err == nil {
		t.Fatal("expected seedCountryModelMetaDB to fail on closed db")
	}
	if err := seedPartnerModelMetaDB(db); err == nil {
		t.Fatal("expected seedPartnerModelMetaDB to fail on closed db")
	}
}

func TestSeedPartnerModelFieldsDBDuplicate(t *testing.T) {
	runtimeScope := newHubTestScope(t)
	db := runtimeScope.Session().DB
	if err := seedPartnerModelFieldsDB(db); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := seedPartnerModelFieldsDB(db); err == nil {
		t.Fatal("expected duplicate seedPartnerModelFieldsDB to fail")
	}
}

func TestStubJSExecutorMethods(t *testing.T) {
	var ex jsexecutor.JsExecutor = stubJSExecutor{}
	stub := stubJSExecutor{}
	stub.AppendJsScripts(&jsengine.JsScript{})
	stub.SetJsScripts([]*jsengine.JsScript{})
	ex.AppendJsScripts(&jsengine.JsScript{})
	if err := ex.Start(); err != nil {
		t.Fatal(err)
	}
	if err := ex.Stop(); err != nil {
		t.Fatal(err)
	}
	resp, err := ex.Execute(context.Background(), &jsengine.JsRequest{})
	if err != nil || resp == nil {
		t.Fatalf("Execute = %v err=%v", resp, err)
	}
	if ex.GetJsScripts() != nil {
		t.Fatal("expected nil scripts")
	}
	ex.SetJsScripts(nil)
	if err := ex.Reload(); err != nil {
		t.Fatal(err)
	}
}
