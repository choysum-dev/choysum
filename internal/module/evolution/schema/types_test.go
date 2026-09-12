// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package schema

import (
	"reflect"
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
	if tag := buildColumnTypeTag("postgres", "monetary", map[string]interface{}{"precision": 10, "scale": 2}); tag != "type:decimal(38,18)" {
		t.Fatalf("unexpected monetary tag: %s", tag)
	}
	if tag := buildColumnTypeTag("postgres", "html", map[string]interface{}{}); tag != "type:text" {
		t.Fatalf("unexpected html tag: %s", tag)
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

	if got := normalizeDefaultStringLiteral("active"); got != "'active'" {
		t.Fatalf("normalizeDefaultStringLiteral(active) = %q", got)
	}
	if got := normalizeDefaultStringLiteral("NULL"); got != "NULL" {
		t.Fatalf("normalizeDefaultStringLiteral(NULL) = %q", got)
	}
	if got := normalizeDefaultStringLiteral("uuid_generate_v4()"); got != "uuid_generate_v4()" {
		t.Fatalf("normalizeDefaultStringLiteral(fn) = %q", got)
	}
	if got := normalizeDefaultStringLiteral("O'Reilly"); got != "'O''Reilly'" {
		t.Fatalf("normalizeDefaultStringLiteral(quote) = %q", got)
	}
	if got := normalizeDefaultStringLiteral("'A => B'"); got != "'A => B'" {
		t.Fatalf("normalizeDefaultStringLiteral(quoted) = %q", got)
	}
	if got := normalizeDefaultStringLiteral("`A => B`"); got != "`A => B`" {
		t.Fatalf("normalizeDefaultStringLiteral(backtick) = %q", got)
	}

	if !isJSFunctionDefaultLiteral("() => true") {
		t.Fatal("expected arrow function default")
	}
	if !isJSFunctionDefaultLiteral("function () { return 'active'; }") {
		t.Fatal("expected function() default")
	}
	if !isJSFunctionDefaultLiteral("A => B") {
		t.Fatal("expected arrow-like default")
	}
	if !isJSFunctionDefaultLiteral("x => 'active'") {
		t.Fatal("expected single-param arrow default")
	}
	if isJSFunctionDefaultLiteral("text with function keyword") {
		t.Fatal("did not expect function keyword alone to be JS default")
	}
	if isJSFunctionDefaultLiteral("'A => B'") {
		t.Fatal("quoted arrow string is a scalar literal")
	}
}
