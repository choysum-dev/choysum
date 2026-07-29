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

	defaultLiteralTags := []string{}
	addStandardTags(&defaultLiteralTags, map[string]interface{}{
		"default": "active",
	})
	if !strings.Contains(strings.Join(defaultLiteralTags, ";"), "default:'active'") {
		t.Fatalf("expected scalar default tag, got %q", strings.Join(defaultLiteralTags, ";"))
	}

	arrowDefaultTags := []string{}
	addStandardTags(&arrowDefaultTags, map[string]interface{}{
		"default": "() => true",
	})
	if strings.Contains(strings.Join(arrowDefaultTags, ";"), "default:") {
		t.Fatalf("did not expect function-like default tag, got %q", strings.Join(arrowDefaultTags, ";"))
	}

	functionDefaultTags := []string{}
	addStandardTags(&functionDefaultTags, map[string]interface{}{
		"default": "function () { return 'active'; }",
	})
	if strings.Contains(strings.Join(functionDefaultTags, ";"), "default:") {
		t.Fatalf("did not expect function-like default tag, got %q", strings.Join(functionDefaultTags, ";"))
	}

	nonFunctionWordTags := []string{}
	addStandardTags(&nonFunctionWordTags, map[string]interface{}{
		"default": "text with function keyword",
	})
	if !strings.Contains(strings.Join(nonFunctionWordTags, ";"), "default:'text with function keyword'") {
		t.Fatalf("expected scalar default tag, got %q", strings.Join(nonFunctionWordTags, ";"))
	}

	sqlKeywordDefaultTags := []string{}
	addStandardTags(&sqlKeywordDefaultTags, map[string]interface{}{
		"default": "NULL",
	})
	if !strings.Contains(strings.Join(sqlKeywordDefaultTags, ";"), "default:NULL") {
		t.Fatalf("expected SQL keyword default tag, got %q", strings.Join(sqlKeywordDefaultTags, ";"))
	}

	sqlFunctionDefaultTags := []string{}
	addStandardTags(&sqlFunctionDefaultTags, map[string]interface{}{
		"default": "uuid_generate_v4()",
	})
	if !strings.Contains(strings.Join(sqlFunctionDefaultTags, ";"), "default:uuid_generate_v4()") {
		t.Fatalf("expected SQL function default tag, got %q", strings.Join(sqlFunctionDefaultTags, ";"))
	}

	escapedQuoteDefaultTags := []string{}
	addStandardTags(&escapedQuoteDefaultTags, map[string]interface{}{
		"default": "O'Reilly",
	})
	if !strings.Contains(strings.Join(escapedQuoteDefaultTags, ";"), "default:'O''Reilly'") {
		t.Fatalf("expected escaped scalar default tag, got %q", strings.Join(escapedQuoteDefaultTags, ";"))
	}

	// A => B (unquoted, contains =>): treated as arrow function, no default: tag.
	arrowStringLiteralTags := []string{}
	addStandardTags(&arrowStringLiteralTags, map[string]interface{}{
		"default": "A => B",
	})
	if strings.Contains(strings.Join(arrowStringLiteralTags, ";"), "default:") {
		t.Fatalf("did not expect default: tag for arrow-like value, got %q", strings.Join(arrowStringLiteralTags, ";"))
	}

	// Single-param arrow with space: x => 'active'
	singleParamArrowTags := []string{}
	addStandardTags(&singleParamArrowTags, map[string]interface{}{
		"default": "x => 'active'",
	})
	if strings.Contains(strings.Join(singleParamArrowTags, ";"), "default:") {
		t.Fatalf("did not expect default: tag for single-param arrow, got %q", strings.Join(singleParamArrowTags, ";"))
	}

	// Quoted string containing => is still a scalar literal.
	quotedArrowLiteralTags := []string{}
	addStandardTags(&quotedArrowLiteralTags, map[string]interface{}{
		"default": "'A => B'",
	})
	if !strings.Contains(strings.Join(quotedArrowLiteralTags, ";"), "default:'A => B'") {
		t.Fatalf("expected scalar default tag for quoted arrow string, got %q", strings.Join(quotedArrowLiteralTags, ";"))
	}

	// Backtick-quoted template literal containing => is still a scalar literal.
	backtickArrowLiteralTags := []string{}
	addStandardTags(&backtickArrowLiteralTags, map[string]interface{}{
		"default": "`A => B`",
	})
	if !strings.Contains(strings.Join(backtickArrowLiteralTags, ";"), "default:`A => B`") {
		t.Fatalf("expected scalar default tag for backtick-quoted arrow string, got %q", strings.Join(backtickArrowLiteralTags, ";"))
	}
}
