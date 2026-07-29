// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isBlankHtml, sanitizeHtmlForWrite } from './html_sanitize';

test('isBlankHtml treats empty tags and nbsp as blank', () => {
  expect(isBlankHtml('')).toBe(true);
  expect(isBlankHtml('<p></p>')).toBe(true);
  expect(isBlankHtml('<p><br></p>')).toBe(true);
  expect(isBlankHtml('<p>&nbsp;</p>')).toBe(true);
  expect(isBlankHtml('<p>hi</p>')).toBe(false);
});

test('sanitizeHtmlForWrite nulls empty and blank values', () => {
  const original = (globalThis as any).$choysum;
  try {
    (globalThis as any).$choysum = {
      html: { sanitize: (s: string) => s },
    };
    expect(sanitizeHtmlForWrite(null)).toBeNull();
    expect(sanitizeHtmlForWrite(undefined)).toBeNull();
    expect(sanitizeHtmlForWrite('')).toBeNull();
    expect(sanitizeHtmlForWrite('<p></p>')).toBeNull();
    expect(sanitizeHtmlForWrite('<p>ok</p>')).toBe('<p>ok</p>');
  } finally {
    (globalThis as any).$choysum = original;
  }
});

test('sanitizeHtmlForWrite fails closed without bridge and rejects non-strings', () => {
  const original = (globalThis as any).$choysum;
  try {
    delete (globalThis as any).$choysum;
    expect(() => sanitizeHtmlForWrite('<p>x</p>')).toThrow('html sanitize bridge unavailable');
    (globalThis as any).$choysum = { html: { sanitize: (s: string) => s } };
    expect(() => sanitizeHtmlForWrite(1 as any)).toThrow('html field value must be a string or null');
  } finally {
    (globalThis as any).$choysum = original;
  }
});

test('sanitizeHtmlForWrite uses bridge output', () => {
  const original = (globalThis as any).$choysum;
  try {
    (globalThis as any).$choysum = {
      html: {
        // Fixture map only — not a general HTML sanitizer (avoids CodeQL fake-sanitizer alerts).
        sanitize: (s: string) => {
          if (s === '<script>bad</script><p>safe</p>') return '<p>safe</p>';
          return s;
        },
      },
    };
    expect(sanitizeHtmlForWrite('<script>bad</script><p>safe</p>')).toBe('<p>safe</p>');
  } finally {
    (globalThis as any).$choysum = original;
  }
});
