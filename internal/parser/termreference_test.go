// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import (
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestParseResourceTitleExprTermReferenceBinding(t *testing.T) {
	source := `
const { _t: _tRef } = createTranslate('base', { output: 'reference', scope: 'web/views/CountryListView' });
`
	bindings := ParseTranslateBindings(source)
	if len(bindings) != 1 || !bindings["_tRef"].ReferenceOutput {
		t.Fatalf("unexpected bindings: %#v", bindings)
	}

	title, titleText, ok := ParseResourceTitleExpr("_tRef('Country')", "base", bindings)
	if !ok || title != "Country" || titleText == nil || titleText.Src != "Country" {
		t.Fatalf("ParseResourceTitleExpr() = (%q, %#v, %v)", title, titleText, ok)
	}
}

func TestCloneTermReferenceWithSrc(t *testing.T) {
	base := NewTermReferenceFromParts("base", "web/views/CountryListView", "Country", "literal")
	clone := CloneTermReferenceWithSrc(&base, "Create Country")
	if clone == nil || clone.Src != "Create Country" || clone.Scope != base.Scope {
		t.Fatalf("CloneTermReferenceWithSrc() = %#v", clone)
	}
}

func NewTermReferenceFromParts(module, scope, src, kind string) meta.TermReference {
	return meta.NewTermReference(module, scope, src, kind)
}
