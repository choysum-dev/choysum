// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package parser

import "testing"

func TestParseImportIsTypeOnly_TsParserPath(t *testing.T) {
	path := "/virtual/modules/partner/service/models/partner.ts"
	content := `
import type TypeOnlyDefault from '@/base/service/models/company'
import type { TypeA, TypeB } from '@/base/service/models/currency'
import { type MixedType, MixedValue } from '@/meta/service/models/model'
import ValueDefault from '@/auth/service/models/user/user'
import * as valueNS from '@/task/service/models/job'
`
	_, ctx := mustParseTSGoCtx(t, path, content)

	assertTypeOnly := func(local string, want bool) {
		imp := ctx.imports[local]
		if imp == nil {
			t.Fatalf("missing import binding %q", local)
		}
		if imp.IsTypeOnly != want {
			t.Fatalf("imports[%q].IsTypeOnly=%v want %v", local, imp.IsTypeOnly, want)
		}
	}

	assertTypeOnly("TypeOnlyDefault", true)
	assertTypeOnly("TypeA", true)
	assertTypeOnly("TypeB", true)
	assertTypeOnly("MixedType", true)
	assertTypeOnly("MixedValue", false)
	assertTypeOnly("ValueDefault", false)
	assertTypeOnly("valueNS", false)
}
