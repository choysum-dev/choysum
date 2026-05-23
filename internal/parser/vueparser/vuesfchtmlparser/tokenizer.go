// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vuesfchtmlparser

import (
	"bytes"
	"io"
	"reflect"

	"golang.org/x/net/html"
)

type span struct {
	start, end int
}

type casePreservingTokenizer struct {
	*html.Tokenizer
	raw           span           // Retrieved via reflection.
	data          span           // Retrieved via reflection.
	buf           []byte         // Retrieved via reflection.
	tt            html.TokenType // Retrieved via reflection.
	attr          [][2]span
	nAttrReturned int // Retrieved via reflection.
}

func newCasePreservingTokenizer(r io.Reader) *casePreservingTokenizer {
	content, _ := io.ReadAll(r)
	return &casePreservingTokenizer{
		Tokenizer: html.NewTokenizer(bytes.NewReader(content)),
	}
}

func (t *casePreservingTokenizer) getFields() {
	v := reflect.ValueOf(t.Tokenizer).Elem()

	// Read the raw span.
	rawField := v.FieldByName("raw")
	t.raw.start = int(rawField.FieldByName("start").Int())
	t.raw.end = int(rawField.FieldByName("end").Int())

	// Read the data span.
	dataField := v.FieldByName("data")
	t.data.start = int(dataField.FieldByName("start").Int())
	t.data.end = int(dataField.FieldByName("end").Int())

	t.nAttrReturned = int(v.FieldByName("nAttrReturned").Int())

	// Read the remaining fields.
	t.buf = v.FieldByName("buf").Bytes()
	t.tt = html.TokenType(v.FieldByName("tt").Uint())

	// Read the attribute spans.
	attrField := v.FieldByName("attr")
	t.attr = make([][2]span, attrField.Len())
	for i := 0; i < attrField.Len(); i++ {
		attr := attrField.Index(i)
		keySpan := attr.Index(0)
		valSpan := attr.Index(1)
		t.attr[i] = [2]span{
			{
				start: int(keySpan.FieldByName("start").Int()),
				end:   int(keySpan.FieldByName("end").Int()),
			},
			{
				start: int(valSpan.FieldByName("start").Int()),
				end:   int(valSpan.FieldByName("end").Int()),
			},
		}
	}
}

func (t *casePreservingTokenizer) TagName() (name []byte, hasAttr bool) {
	t.getFields()

	if t.data.start < t.data.end {
		switch t.tt {
		case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken:
			s := t.buf[t.data.start:t.data.end]
			t.data.start = t.raw.end
			t.data.end = t.raw.end
			return s, t.nAttrReturned < len(t.attr)
		}
	}

	return nil, false
}

func (t *casePreservingTokenizer) TagAttr() (key, val []byte, more bool) {
	if t.nAttrReturned < len(t.attr) {
		switch t.tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			x := t.attr[t.nAttrReturned]
			t.nAttrReturned++
			key = t.buf[x[0].start:x[0].end]
			val = t.buf[x[1].start:x[1].end]
			return key, val, t.nAttrReturned < len(t.attr)
		}
	}
	return nil, nil, false
}
