// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jobtoken

import "testing"

func TestProtoDescriptors(t *testing.T) {
	md, err := MethodDesc()
	if err != nil {
		t.Fatalf("MethodDesc error: %v", err)
	}
	if md == nil {
		t.Fatalf("MethodDesc is nil")
	}

	if _, err := ReqDesc(); err != nil {
		t.Fatalf("ReqDesc error: %v", err)
	}
	if _, err := RespDesc(); err != nil {
		t.Fatalf("RespDesc error: %v", err)
	}
}

func TestServiceNames(t *testing.T) {
	if ServiceFullName() != "auth.JobTokenService" {
		t.Fatalf("unexpected service name: %s", ServiceFullName())
	}
	if FullMethod() != "/auth.JobTokenService/IssueTaskJobToken" {
		t.Fatalf("unexpected full method: %s", FullMethod())
	}
}
