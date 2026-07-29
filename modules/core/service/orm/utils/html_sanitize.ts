// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Authoritative HTML sanitize for ORM write paths via `$choysum.html.sanitize`
 * (Go bluemonday). Fail-closed when the bridge is unavailable.
 */

/** Strip tags / entities and decide whether the HTML has meaningful text (D10). */
export function isBlankHtml(html: string): boolean {
  const text = String(html || '')
    .replace(/<[^>]*>/g, ' ')
    .replace(/&nbsp;/gi, ' ')
    .replace(/&#160;/gi, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  return text === '';
}

/**
 * Sanitize an html field value for persistence.
 * - null / undefined / '' / blank-tag-only → null
 * - non-string → throw
 * - missing bridge → throw (fail-closed)
 */
export function sanitizeHtmlForWrite(value: unknown): string | null {
  if (value == null) return null;
  if (typeof value !== 'string') {
    throw new Error('html field value must be a string or null');
  }
  if (value === '') return null;

  const sanitize =
    typeof $choysum !== 'undefined' && $choysum.html && typeof $choysum.html.sanitize === 'function'
      ? $choysum.html.sanitize.bind($choysum.html)
      : undefined;
  if (typeof sanitize !== 'function') {
    throw new Error('html sanitize bridge unavailable ($choysum.html.sanitize)');
  }

  const cleaned = sanitize(value);
  if (cleaned == null) return null;
  const asString = String(cleaned);
  if (asString === '' || isBlankHtml(asString)) return null;
  return asString;
}
