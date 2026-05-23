// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/datatypes"
)

func TestTypeHelpers(t *testing.T) {
	if value := getDefaultValue("jsonobject"); !reflect.DeepEqual(value, datatypes.JSON([]byte("{}"))) {
		t.Fatalf("unexpected default jsonobject value: %#v", value)
	}
	if value := getDefaultValue("date"); value != "" {
		t.Fatalf("unexpected default date value: %#v", value)
	}
	if value := getDefaultValue("blob"); !reflect.DeepEqual(value, []byte{}) {
		t.Fatalf("unexpected default blob value: %#v", value)
	}
	if value := getDefaultValue("missing"); value != nil {
		t.Fatalf("expected nil default value, got %#v", value)
	}

	if tag := buildColumnTypeTag("postgres", "varchar", map[string]interface{}{"size": 64}); tag != "type:varchar(64)" {
		t.Fatalf("unexpected varchar tag: %s", tag)
	}
	if tag := buildColumnTypeTag("mysql", "varchar", map[string]interface{}{}); tag != "type:varchar(255)" {
		t.Fatalf("unexpected default varchar tag: %s", tag)
	}
	if tag := buildColumnTypeTag("postgres", "bool", map[string]interface{}{}); tag != "type:boolean" {
		t.Fatalf("unexpected bool tag: %s", tag)
	}
	if tag := buildColumnTypeTag("sqlite", "date", map[string]interface{}{}); tag != "type:text" {
		t.Fatalf("unexpected sqlite date tag: %s", tag)
	}
	if tag := buildColumnTypeTag("mysql", "decimal", map[string]interface{}{"precision": 10, "scale": 2}); tag != "type:decimal(38,18)" {
		t.Fatalf("unexpected decimal tag: %s", tag)
	}
	if tag := buildColumnTypeTag("sqlite", "char", map[string]interface{}{"size": "20"}); tag != "type:char(20)" {
		t.Fatalf("unexpected char tag: %s", tag)
	}
	if tag := buildColumnTypeTag("postgres", "blob", map[string]interface{}{}); tag != "type:bytea" {
		t.Fatalf("unexpected postgres blob tag: %s", tag)
	}
	if tag := buildColumnTypeTag("mysql", "blob", map[string]interface{}{}); tag != "type:longblob" {
		t.Fatalf("unexpected mysql blob tag: %s", tag)
	}

	tags := []string{}
	addStandardTags(&tags, map[string]interface{}{
		"primaryKey":      true,
		"notNull":         true,
		"unique":          true,
		"index":           "idx_status",
		"uniqueIndex":     "uniq_a uniq_b",
		"checkConstraint": " `status in ('draft','done')` ",
	})
	joined := strings.Join(tags, ";")
	for _, want := range []string{"primaryKey", "not null", "unique", "index:idx_status", "uniqueIndex:uniq_a", "uniqueIndex:uniq_b", "check:,(status in ('draft','done'))"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected tag %q in %q", want, joined)
		}
	}
}
