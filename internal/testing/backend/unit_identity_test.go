// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backend

import (
	"context"
	"testing"
)

func TestResolveUnitTestDefaultIdentityNilScope(t *testing.T) {
	_, ok := resolveUnitTestDefaultIdentity(context.Background(), nil)
	if ok {
		t.Fatal("expected ok=false for nil scope")
	}
}

func TestUnitTestJsRequestContextShape(t *testing.T) {
	ctx := unitTestJsRequestContext(unitTestDefaultIdentity{UserID: "u1", CompanyID: "c1"})
	identity, _ := ctx["identity"].(map[string]interface{})
	if identity["userId"] != "u1" {
		t.Fatalf("userId=%v", identity["userId"])
	}
	biz, _ := ctx["ctx"].(map[string]interface{})
	if biz["activeCompanyId"] != "c1" {
		t.Fatalf("activeCompanyId=%v", biz["activeCompanyId"])
	}
	ids, _ := biz["enabledCompanyIds"].([]string)
	if len(ids) != 1 || ids[0] != "c1" {
		t.Fatalf("enabledCompanyIds=%v", biz["enabledCompanyIds"])
	}
}
