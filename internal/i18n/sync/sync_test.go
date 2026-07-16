// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package sync

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/i18n/po"
)

func TestSyncModulePoPreservesMsgstrAndObsoletes(t *testing.T) {
	root := t.TempDir()
	i18nDir := filepath.Join(root, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		t.Fatal(err)
	}

	pot := `#
msgid ""
msgstr ""

#: web/a.ts:1
msgctxt "web/a@title"
msgid "Hello"
msgstr ""

#: web/a.ts:2
msgctxt "web/a@ok"
msgid "OK"
msgstr ""
`
	if err := os.WriteFile(filepath.Join(i18nDir, "demo.pot"), []byte(pot), 0o644); err != nil {
		t.Fatal(err)
	}

	oldPo := `#
msgid ""
msgstr "Language: zh_CN\n"

msgctxt "web/a@title"
msgid "Hello"
msgstr "你好"

msgctxt "web/a@gone"
msgid "Gone"
msgstr "已删除"
`
	if err := os.WriteFile(filepath.Join(i18nDir, "zh_CN.po"), []byte(oldPo), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := SyncModulePo(root, "demo", "zh_CN")
	if err != nil {
		t.Fatalf("SyncModulePo: %v", err)
	}
	if result.Kept != 1 || result.Added != 1 || result.Obsoleted != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}

	raw, err := os.ReadFile(result.PoPath)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := po.Parse(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}

	byKey := map[string]po.Entry{}
	for _, e := range entries {
		if po.IsHeader(e) {
			continue
		}
		byKey[e.Key()] = e
	}

	hello := byKey["web/a@title\x00Hello"]
	if hello.Msgstr != "你好" || hello.Obsolete {
		t.Fatalf("Hello not preserved: %+v", hello)
	}
	ok := byKey["web/a@ok\x00OK"]
	if ok.Msgstr != "" || ok.Obsolete {
		t.Fatalf("OK should be new empty: %+v", ok)
	}
	gone := byKey["web/a@gone\x00Gone"]
	if !gone.Obsolete || gone.Msgstr != "已删除" {
		t.Fatalf("Gone should be obsolete with msgstr: %+v", gone)
	}
	if !strings.Contains(string(raw), "#~") {
		t.Fatalf("expected obsolete markers in:\n%s", raw)
	}
}
