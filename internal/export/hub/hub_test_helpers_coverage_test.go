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
}

func TestAuthCtxWithServer(t *testing.T) {
	ctx := authCtxWithServer(t, denyAuthServer{})
	if ctx == nil {
		t.Fatal("expected auth context")
	}
}

func TestStubJSExecutorMethods(t *testing.T) {
	var ex jsexecutor.JsExecutor = stubJSExecutor{}
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
