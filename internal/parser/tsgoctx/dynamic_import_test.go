// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package tsgoctx

import (
	"testing"

	"github.com/choysum-dev/choysum/internal/parser"
)

func TestParseCtxCollectsDynamicImports(t *testing.T) {
	path := "/virtual/modules/auth/service/tests/dynamic.ts"
	content := `
const mod = await import('@/meta/service/models/ui_resource');
type Company = typeof import('@/base/service/models/company');
const local = import('./local_probe');
`
	ctx, err := Parse(map[string]string{"@/*": "/virtual/modules/*"}, path, content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(ctx.DynamicImports) != 2 {
		t.Fatalf("DynamicImports len = %d, want 2", len(ctx.DynamicImports))
	}
}

func TestParseCtxCollectDynamicImports_NilReceiverAndSource(t *testing.T) {
	var ctx *ParseCtx
	ctx.collectDynamicImports()

	c := &ParseCtx{}
	c.collectDynamicImports()
}

func TestMergeDynamicImports(t *testing.T) {
	first := []*parser.Import{{ReferenceIdent: "a"}}
	second := []*parser.Import{{ReferenceIdent: "b"}}
	merged := MergeDynamicImports(first, second)
	if len(merged) != 2 {
		t.Fatalf("merged len = %d, want 2", len(merged))
	}
	if len(MergeDynamicImports(first, nil)) != len(first) {
		t.Fatal("nil src should return dst unchanged")
	}
}
